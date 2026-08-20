package main

// worldgenesishandler.go — the two routes the create-world journey talks to.
//
//	POST /worlds/interview  →  one JSON turn: the next question, or nothing left to ask
//	POST /worlds/genesis    →  an SSE stream of world_genesis_frame/1, ending in a world you can enter
//
// Neither hangs off /worlds/{id}: there is no world yet, which is the whole point. They sit beside
// GET/POST /worlds as collection-level acts.
//
// WHY GENESIS STREAMS. A build is a long authored act with intermediate results — the same shape as a
// beat, and the same transport for the same reason. The alternative was a two-minute blank screen ending
// in a redirect, which tells the user nothing while it works and nothing about why if it fails. Every
// frame here names something that was actually authored, in the world's own language; there is no
// percentage, no ETA and no stage checklist rendered from a timer (law 2: never invent a displayed value).
//
// WHY THE TRANSACTION SPANS THE WHOLE STREAM. The frames narrate work already done: the world is authored,
// then committed inside one transaction, and the stream reports each part as it lands. A failure anywhere
// rolls the whole thing back, so a user who watched five frames go by and then saw a refusal has no
// half-world in their directory. Committing per frame would leave exactly the listed-but-unenterable world
// that `playable` exists to prevent.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// worldGenesisFrameSchemaVersion stamps every frame of the build stream. The frontend pins it exactly and
// fails the load on a mismatch, so reshaping a frame means moving this string — the version moving IS the
// notification.
const worldGenesisFrameSchemaVersion = "world_genesis_frame/1"

// worldInterviewTurnSchemaVersion stamps the interview response. Same contract discipline.
const worldInterviewTurnSchemaVersion = "world_interview_turn/1"

// genesisCostCeilingEnv bounds what one build may spend. Env-configured with a default rather than a
// constant, because the honest ceiling depends on the seat map and nobody here knows what model a
// deployment routes world_genesis to (the same reasoning DREAMCHAT_BEAT_COST_WARN_USD already follows).
const genesisCostCeilingEnv = "DREAMCHAT_GENESIS_COST_WARN_USD"

const defaultGenesisCostCeilingUSD = 0.50

var (
	worldGenesisRoute   = regexp.MustCompile(`^/worlds/genesis$`)
	worldInterviewRoute = regexp.MustCompile(`^/worlds/interview$`)
)

// briefMaxBytes caps the brief. Generous — three long paragraphs sit well inside it — but finite, because
// the brief goes into a prompt and an unbounded body is an unbounded bill.
const briefMaxBytes = 8 << 10

type worldGenesisHandler struct {
	pool   *pgxpool.Pool
	dbg    bool
	bridge *Bridge
	// images commissions the new world's art once it exists. Nil is an ordinary state — the world
	// is built and playable either way; it simply has no pictures yet.
	images *imageClient
}

func NewWorldGenesisHandler(pool *pgxpool.Pool, debug bool, bridge *Bridge, images *imageClient) http.Handler {
	return &worldGenesisHandler{pool: pool, dbg: debug, bridge: bridge, images: images}
}

func (h *worldGenesisHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost &&
		(worldGenesisRoute.MatchString(r.URL.Path) || worldInterviewRoute.MatchString(r.URL.Path))
}

// genesisRequest is the whole input surface of both routes: the brief, and whatever has been asked and
// answered so far. The Fast lane omits answers and is otherwise identical — one pipeline, two lanes.
type genesisRequest struct {
	Brief string `json:"brief"`
	// ArtStyle is the look every picture this world ever renders is drawn in: a preset key from the
	// catalogue, or "custom:" and the user's own description. Omitted is a real answer — it means the
	// house look, which is what every world made before the picker existed still uses.
	ArtStyle string            `json:"art_style,omitempty"`
	Answers  []InterviewAnswer `json:"answers,omitempty"`
}

func (h *worldGenesisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case worldInterviewRoute.MatchString(r.URL.Path):
		h.interview(w, r)
	case worldGenesisRoute.MatchString(r.URL.Path):
		h.build(w, r)
	default:
		http.NotFound(w, r)
	}
}

// readBrief decodes and sanity-checks the body both routes share.
func readBrief(w http.ResponseWriter, r *http.Request) (genesisRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, briefMaxBytes)
	var req genesisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "that request did not parse")
		return req, false
	}
	req.Brief = strings.TrimSpace(req.Brief)
	if req.Brief == "" {
		writeJSONError(w, http.StatusBadRequest, "describe the world you want, and I will build it")
		return req, false
	}
	return req, true
}

// interview answers with the next question or with "nothing more to ask". Stateless: the client sends the
// brief and every prior answer and gets back one turn. Nothing is stored, so an abandoned interview leaves
// nothing behind.
func (h *worldGenesisHandler) interview(w http.ResponseWriter, r *http.Request) {
	req, ok := readBrief(w, r)
	if !ok {
		return
	}

	ctx, costs := withCostSink(r.Context())
	start := time.Now()
	turn, err := askNextQuestion(ctx, h.bridge.Driver(SeatWorldInterview.Name), req.Brief, req.Answers)
	usd, tokIn, tokOut, cached, calls := costs.snapshot()
	log.Printf("world interview timing: ms=%d asked=%d done=%v calls=%d tok_in=%d cached=%d tok_out=%d cost_usd=%.6f",
		time.Since(start).Milliseconds(), len(req.Answers), turn.Done, calls, tokIn, cached, tokOut, usd)
	if err != nil {
		// A question that could not be authored is not a dead end: the user can still build with what they
		// have already said, so the honest answer is "nothing more to ask" and the fault goes to the log.
		log.Printf("world interview: falling through to done: %v", err)
	}

	// Shape follows the answer rather than padding it: a finished interview sends {done:true} and nothing
	// else. Emitting question:"" and options:null would make the client sniff empty values to find out
	// whether it had a question, and would publish a contract whose fields lie about being present.
	body := map[string]any{
		"schema_version": worldInterviewTurnSchemaVersion,
		"done":           turn.Done,
	}
	if !turn.Done {
		body["question"] = turn.Question
		body["options"] = turn.Options
		if strings.TrimSpace(turn.Why) != "" {
			body["why"] = turn.Why
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

// build authors the world and streams what lands. The response is 200 + text/event-stream from the moment
// the request is accepted, because a client that asked for a stream must be answered with a stream — a
// mid-flight http.Error is not an error to a browser, it is the connection dying (the lesson beatsstream
// records at length).
func (h *worldGenesisHandler) build(w http.ResponseWriter, r *http.Request) {
	req, ok := readBrief(w, r)
	if !ok {
		return
	}

	frames, ok := newFrameWriter(w, worldGenesisFrameSchemaVersion)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Every provider call this build makes bills into this sink. Installed before the seat is touched: the
	// existing sink is scoped to the beat handler, so a genesis path without its own would report
	// cost_usd=0.000000 and roll up nowhere.
	ctx, costs := withCostSink(r.Context())
	start := time.Now()
	worldID := "(none)"
	defer func() {
		usd, tokIn, tokOut, cached, calls := costs.snapshot()
		log.Printf("world genesis timing: total_ms=%d world=%s calls=%d tok_in=%d cached=%d tok_out=%d "+
			"cost_usd=%.6f session_usd=%.4f",
			time.Since(start).Milliseconds(), worldID, calls, tokIn, cached, tokOut, usd, sessionTotalUSD())
		if ceiling := genesisCostCeilingUSD(); ceiling > 0 && usd > ceiling {
			log.Printf("COST WARNING: building a world spent $%.4f (>$%.4f) across %d call(s) — "+
				"check the seat map for world_genesis", usd, ceiling, calls)
		}
	}()

	// The style is validated BEFORE the seat call, because an unusable style key is the one refusal
	// that can cost nothing: authoring a whole world and then discovering we cannot draw it would
	// spend a build to report a typo. The module's own message is what the user reads — it names the
	// styles that do exist.
	if _, err := ResolveArtStyle(req.ArtStyle); err != nil {
		h.fail(frames, refuse("%s", err.Error()))
		return
	}

	// Authoring first, and it is the slow part: one seat call for a whole world. Nothing is written yet, so
	// a refusal here costs nothing but the call.
	_ = frames.emit("working", map[string]any{"stated": "Reading what you asked for."})
	doc, err := authorWorld(ctx, h.bridge.Driver(SeatWorldGenesis.Name), req.Brief, req.Answers)
	if err != nil {
		h.fail(frames, err)
		return
	}

	// One transaction for the whole world (AC-2), rolled back on every failure path below.
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		h.fail(frames, fmt.Errorf("build: begin: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	newID, err := commitWorldGenesis(ctx, tx, doc, req.Brief, req.ArtStyle)
	if err != nil {
		h.fail(frames, err)
		return
	}

	// The frames narrate work that is now done and about to be made permanent. Emitted from the AUTHORED
	// document rather than re-read from the database on purpose: what the user is told was authored is
	// exactly what was authored, with no second projection to disagree with the first.
	for _, line := range genesisNarration(doc) {
		_ = frames.emit("working", map[string]any{"stated": line})
	}

	if err := tx.Commit(ctx); err != nil {
		h.fail(frames, fmt.Errorf("build: commit: %w", err))
		return
	}
	worldID = newID

	_ = frames.emit("world", map[string]any{
		"id":           newID,
		"display_name": strings.TrimSpace(doc.World.DisplayName),
		"tagline":      strings.TrimSpace(doc.World.Tagline),
		"playable":     true,
	})

	// The world is committed and the user has been told it is ready; its pictures are commissioned
	// after that line, not before it. A dozen images is several minutes of another service, and
	// nothing in the world is waiting on them: `image` is null until it is not, and swaps in on a
	// later read (image_ref/1, D-8). Making the user watch a spinner for art they have not asked to
	// see yet — or worse, losing an authored world because an image provider was down — is the
	// trade this ordering refuses.
	//
	// It is detached from this request deliberately: the stream ends here, and the sweep outlives it.
	commissionArtInBackground(h.pool, h.images, newID)
}

// fail ends the stream honestly. A refusal carries the seat's own stated reason, because the user asked for
// something that could not become a world and deserves to know what; a fault carries a generic line,
// because "connection reset by peer" is not a sentence a player can act on. Both reach the log in full.
func (h *worldGenesisHandler) fail(frames *frameWriter, err error) {
	var refusal *genesisRefusal
	if errors.As(err, &refusal) {
		log.Printf("world genesis refused: %v", err)
		_ = frames.emit("refused", map[string]any{"stated": refusal.why})
		return
	}
	log.Printf("world genesis failed: %v", err)
	_ = frames.emit("error", map[string]any{"stated": "the world could not be built"})
}

// genesisNarration turns the authored document into the lines the user watches land — the world's own
// language, never a stage list. Every line names real authored content, so a frame on screen always has
// something behind it.
func genesisNarration(doc *genesisDoc) []string {
	rooms := make([]string, 0, len(doc.Places))
	for _, p := range doc.Places {
		rooms = append(rooms, strings.TrimSpace(p.Descriptor))
	}
	people := make([]string, 0, len(doc.Cast))
	for _, a := range doc.Cast {
		people = append(people, strings.TrimSpace(a.Descriptor))
	}
	return []string{
		fmt.Sprintf("%s — %s", strings.TrimSpace(doc.World.DisplayName), strings.TrimSpace(doc.World.Tagline)),
		fmt.Sprintf("The place: %s.", strings.TrimSpace(doc.Region.Descriptor)),
		"Rooms: " + strings.Join(rooms, "; ") + ".",
		"Already here: " + strings.Join(people, "; ") + ".",
		fmt.Sprintf("%d thing(s) that matter, and %d thing(s) somebody is not saying.", len(doc.Objects), len(doc.Cast)),
		fmt.Sprintf("Before you: %d moment(s), and everyone remembers them differently.", len(doc.History)),
		strings.TrimSpace(doc.Arrival.Stated),
	}
}

// genesisCostCeilingUSD reads the per-build warning ceiling. 0 disables it, mirroring
// DREAMCHAT_BEAT_COST_WARN_USD exactly so an operator learns one convention, not two.
func genesisCostCeilingUSD() float64 {
	raw := strings.TrimSpace(os.Getenv(genesisCostCeilingEnv))
	if raw == "" {
		return defaultGenesisCostCeilingUSD
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		log.Printf("%s=%q is not a usable dollar amount — using the default $%.2f",
			genesisCostCeilingEnv, raw, defaultGenesisCostCeilingUSD)
		return defaultGenesisCostCeilingUSD
	}
	return v
}
