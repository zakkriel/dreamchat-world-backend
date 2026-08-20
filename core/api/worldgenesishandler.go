package main

// worldgenesishandler.go — the three routes the create-world journey talks to.
//
//	POST /worlds/interview          →  one JSON turn: the next question, or nothing left to ask
//	POST /worlds/genesis            →  an SSE stream of world_genesis_frame/2, ending in a `choice`
//	                                    frame the player must answer before anything commits
//	POST /worlds/genesis/kickstart  →  one JSON turn per answer (world_kickstart_turn/1); the LAST
//	                                    answer is the one transaction that makes the world playable
//
// None of these hang off /worlds/{id}: there is no world yet, which is the whole point. They sit
// beside GET/POST /worlds as collection-level acts.
//
// WHY GENESIS STREAMS. A build is a long authored act with intermediate results — the same shape as a
// beat, and the same transport for the same reason. The alternative was a two-minute blank screen ending
// in a redirect, which tells the user nothing while it works and nothing about why if it fails. Every
// frame here names something that was actually authored, in the world's own language; there is no
// percentage, no ETA and no stage checklist rendered from a timer (law 2: never invent a displayed value).
//
// THREE PHASES, ONE TRANSACTION AT THE END. build() authors the whole world and stops — it narrates what
// it wrote, then ends the stream in a `choice` frame instead of committing; nothing is written yet.
// kickstart() takes it from there, one HTTP call per answer: the character turn (who the player is)
// authors the scenario options and asks again; the scenario turn (how it starts) is the one that
// commits. All three phases share the same authored genesisDoc, handed between requests as an in-memory
// draft (genesisdrafts.go) instead of round-tripped over the wire — sending it back to the client would
// leak every secret and knowledge path the world holds (AC-7). Only the LAST turn opens a transaction,
// and it is the SAME commitWorldGenesis the old single-shot build used: authored first, committed once
// (AC-2). A failure anywhere in the whole journey — a refused character, a refused opening, a commit
// that fails — leaves no directory row; a player who answered two questions and then hit a fault has no
// half-world waiting for them, which is what `playable` exists to promise.

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
const worldGenesisFrameSchemaVersion = "world_genesis_frame/2"

// worldKickstartTurnSchemaVersion stamps the kickstart response. Same contract discipline.
const worldKickstartTurnSchemaVersion = "world_kickstart_turn/1"

// worldInterviewTurnSchemaVersion stamps the interview response. Same contract discipline.
const worldInterviewTurnSchemaVersion = "world_interview_turn/1"

// genesisCostCeilingEnv bounds what one build may spend. Env-configured with a default rather than a
// constant, because the honest ceiling depends on the seat map and nobody here knows what model a
// deployment routes world_genesis to (the same reasoning DREAMCHAT_BEAT_COST_WARN_USD already follows).
const genesisCostCeilingEnv = "DREAMCHAT_GENESIS_COST_WARN_USD"

const defaultGenesisCostCeilingUSD = 0.50

var (
	worldGenesisRoute     = regexp.MustCompile(`^/worlds/genesis$`)
	worldInterviewRoute   = regexp.MustCompile(`^/worlds/interview$`)
	genesisKickstartRoute = regexp.MustCompile(`^/worlds/genesis/kickstart$`)
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
	// drafts holds an authored-but-uncommitted world between the choice frame that ends build() and
	// the kickstart route that turns the user's answer into a commit (genesisdrafts.go).
	drafts *draftStore
}

func NewWorldGenesisHandler(pool *pgxpool.Pool, debug bool, bridge *Bridge, images *imageClient) http.Handler {
	return &worldGenesisHandler{pool: pool, dbg: debug, bridge: bridge, images: images, drafts: newDraftStore(genesisDraftTTL)}
}

func (h *worldGenesisHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost &&
		(worldGenesisRoute.MatchString(r.URL.Path) || worldInterviewRoute.MatchString(r.URL.Path) ||
			genesisKickstartRoute.MatchString(r.URL.Path))
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
	case genesisKickstartRoute.MatchString(r.URL.Path):
		h.kickstart(w, r)
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
	// This log covers only THIS request's own spend — authoring, and (when the brief already states
	// who the player is) the scenario call made in the same pass. It is not the whole build's bill: the
	// AGGREGATE line, covering every seat call across both kickstart turns too, is logged once, at
	// commit, in kickstart() — from draft.tally, which starts as this request's own tally and gets
	// added to on every subsequent turn.
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

	// Narration first — every line names authored content (law 2), commit or not.
	for _, line := range genesisNarration(doc) {
		if err := frames.emit("working", map[string]any{"stated": line}); err != nil {
			return
		}
	}

	draft := &genesisDraft{doc: doc, brief: req.Brief, artStyle: req.ArtStyle}
	usd, in, out, cached, calls := costs.snapshot()
	draft.tally.add(usd, in, out, cached, calls)

	var question string
	var options []map[string]any
	if len(doc.ArrivalCandidates) > 0 {
		question = "Who are you here?"
		options = characterTurnOptions(doc)
	} else {
		// Identity stated in the brief: author the scenario options now, so the stream
		// ends in the scenario question with no extra round-trip (spec, phase 1).
		k, err := authorKickstart(ctx, h.bridge.Driver(SeatWorldKickstart.Name), doc, req.Brief, doc.Arrival.CanonicalName, "")
		if err != nil {
			h.fail(frames, err)
			return
		}
		draft.identity = &k.Identity
		draft.scenarios = k.Scenarios
		usd, in, out, cached, calls = costs.snapshot()
		draft.tally = draftTally{}
		draft.tally.add(usd, in, out, cached, calls) // snapshot is cumulative per sink; reset-then-add keeps the tally honest
		question = "How does it start?"
		options = scenarioTurnOptions(k.Scenarios)
	}

	handle := h.drafts.mint()
	h.drafts.put(handle, draft)
	_ = frames.emit("choice", map[string]any{"handle": handle, "question": question, "options": options})
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

// kickstartRequest is the input to every kickstart turn: the handle a prior turn minted, and the
// player's answer — a chosen option's label, or their own words entirely.
type kickstartRequest struct {
	Handle string `json:"handle"`
	Answer string `json:"answer"`
}

// kickstart turns one answer into the next question, or — on the scenario answer — into the one
// transaction that makes the world playable (AC-2). draft.identity == nil is the whole state machine:
// nil means the character question is still open, set means only the opening remains.
//
// Every early return below a successful claim that is NOT the terminal commit re-puts the draft under
// the SAME handle first: claim removes on read, so a refusal, a seat fault or a commit failure that
// forgot to re-put would silently delete an otherwise-retryable build out from under a player about to
// hit retry.
func (h *worldGenesisHandler) kickstart(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, briefMaxBytes)
	var req kickstartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "that request did not parse")
		return
	}

	draft, ok := h.drafts.claim(req.Handle)
	if !ok {
		// Expired, unknown, or already spent — the commit below removes the draft too, so a repeat
		// answer against a finished build lands here as well. Either way nothing was ever half-written:
		// an absent draft never touched the database (AC-2 holds by construction, not by cleanup).
		writeJSONError(w, http.StatusGone, errDraftExpired.Error())
		return
	}

	ctx, costs := withCostSink(r.Context())
	start := time.Now()
	answer := strings.TrimSpace(req.Answer)
	bail := func(err error) {
		h.drafts.put(req.Handle, draft)
		var refusal *genesisRefusal
		if errors.As(err, &refusal) {
			log.Printf("world kickstart refused: %v", err)
			writeJSONError(w, http.StatusUnprocessableEntity, refusal.why)
			return
		}
		log.Printf("world kickstart failed: %v", err)
		writeJSONError(w, http.StatusBadGateway, "the opening could not be authored")
	}

	if draft.identity == nil {
		// Character turn: a match against the offered candidates names who the player picked; no
		// match means the answer is their own words, and authorKickstart takes it exactly as who they
		// are — free text is a first-class answer here, not a rejection.
		who := answer
		if c, ok := matchCandidateAnswer(draft.doc.ArrivalCandidates, answer); ok {
			who = c.CanonicalName
		}
		k, err := authorKickstart(ctx, h.bridge.Driver(SeatWorldKickstart.Name), draft.doc, draft.brief, who, "")
		if err != nil {
			bail(err)
			return
		}
		usd, in, out, cached, calls := costs.snapshot()
		draft.tally.add(usd, in, out, cached, calls)
		draft.identity = &k.Identity
		draft.scenarios = k.Scenarios
		h.drafts.put(req.Handle, draft)

		body := map[string]any{
			"schema_version": worldKickstartTurnSchemaVersion,
			"done":           false,
			"question":       "How does it start?",
			"options":        scenarioTurnOptions(k.Scenarios),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
		return
	}

	// Scenario turn — the last one. A match against the authored options names the scenario chosen; no
	// match grounds the player's own opening as the single scenario (authorKickstart's second mode).
	chosen, matched := matchScenarioAnswer(draft.scenarios, answer)
	if !matched {
		k, err := authorKickstart(ctx, h.bridge.Driver(SeatWorldKickstart.Name), draft.doc, draft.brief,
			draft.identity.CanonicalName, answer)
		if err != nil {
			bail(err)
			return
		}
		usd, in, out, cached, calls := costs.snapshot()
		draft.tally.add(usd, in, out, cached, calls)
		chosen = k.Scenarios[0]
	}

	doc := draft.doc
	doc.Arrival = genesisArrival{
		Descriptor:    draft.identity.Descriptor,
		CanonicalName: draft.identity.CanonicalName,
		Place:         chosen.Place,
		Stated:        chosen.Stated,
		Why:           chosen.Why,
	}
	doc.ArrivalCandidates = nil
	if err := doc.validate(); err != nil {
		// Belt: the new arrival must still hit a populated place with an exit. The candidates were
		// already vetted at build time, so this is not expected to fire — but when it does, it is a
		// refusal like any other, and the draft is retryable exactly the same way.
		bail(err)
		return
	}

	// One transaction for the whole world (AC-2): commit or nothing, and every failure from here re-puts
	// the draft too — a rolled-back commit is a retryable draft, not a lost one.
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		bail(fmt.Errorf("kickstart: begin: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	newID, err := commitWorldGenesis(ctx, tx, doc, draft.brief, draft.artStyle)
	if err != nil {
		bail(err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		bail(fmt.Errorf("kickstart: commit: %w", err))
		return
	}
	// Committed. The claim at the top of kickstart() already removed the draft, and nothing puts it
	// back from here on: the handle is spent. A repeat answer lands on the expired-handle 410 above,
	// which is the truth — there is nothing left to retry.

	usd, in, out, cached, calls := costs.snapshot()
	draft.tally.add(usd, in, out, cached, calls)
	// The aggregate line: draft.tally is the WHOLE build's spend — every seat call across every turn,
	// genesis authoring through both kickstart answers — not just this request's. Mirrors build()'s own
	// "world genesis timing" line exactly, but logged once, here, because this is the only moment a
	// build-wide total exists to report.
	log.Printf("world genesis timing: total_ms=%d world=%s calls=%d tok_in=%d cached=%d tok_out=%d "+
		"cost_usd=%.6f session_usd=%.4f",
		time.Since(start).Milliseconds(), newID, draft.tally.calls, draft.tally.tokIn, draft.tally.cached,
		draft.tally.tokOut, draft.tally.usd, sessionTotalUSD())
	if ceiling := genesisCostCeilingUSD(); ceiling > 0 && draft.tally.usd > ceiling {
		log.Printf("COST WARNING: building a world spent $%.4f (>$%.4f) across %d call(s) — "+
			"check the seat map for world_genesis", draft.tally.usd, ceiling, draft.tally.calls)
	}

	// The world is committed; its pictures are commissioned after that, not before (build()'s own note
	// on this ordering applies unchanged — kickArt is detached and outlives this request).
	kickArt(h.pool, h.images, newID)

	body := map[string]any{
		"schema_version": worldKickstartTurnSchemaVersion,
		"done":           true,
		"world": map[string]any{
			"id":           newID,
			"display_name": strings.TrimSpace(doc.World.DisplayName),
			"tagline":      strings.TrimSpace(doc.World.Tagline),
			"playable":     true,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

// matchCandidateAnswer resolves the character turn's answer against the offered arrival candidates by
// canonical name, trimmed: case-sensitive first, then case-insensitive. No match is not an error — it
// means the answer is the player's own words, a first-class way to answer this turn.
func matchCandidateAnswer(candidates []genesisCandidate, answer string) (genesisCandidate, bool) {
	for _, c := range candidates {
		if strings.TrimSpace(c.CanonicalName) == answer {
			return c, true
		}
	}
	for _, c := range candidates {
		if strings.EqualFold(strings.TrimSpace(c.CanonicalName), answer) {
			return c, true
		}
	}
	return genesisCandidate{}, false
}

// matchScenarioAnswer resolves the scenario turn's answer the same way, against the authored options'
// labels. No match means the player wrote their own opening.
func matchScenarioAnswer(scenarios []kickstartScenario, answer string) (kickstartScenario, bool) {
	for _, s := range scenarios {
		if strings.TrimSpace(s.Label) == answer {
			return s, true
		}
	}
	for _, s := range scenarios {
		if strings.EqualFold(strings.TrimSpace(s.Label), answer) {
			return s, true
		}
	}
	return kickstartScenario{}, false
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
	lines := []string{
		fmt.Sprintf("%s — %s", strings.TrimSpace(doc.World.DisplayName), strings.TrimSpace(doc.World.Tagline)),
		fmt.Sprintf("The place: %s.", strings.TrimSpace(doc.Region.Descriptor)),
		"Rooms: " + strings.Join(rooms, "; ") + ".",
		"Already here: " + strings.Join(people, "; ") + ".",
		fmt.Sprintf("%d thing(s) that matter, and %d thing(s) somebody is not saying.", len(doc.Objects), len(doc.Cast)),
		fmt.Sprintf("Before you: %d moment(s), and everyone remembers them differently.", len(doc.History)),
	}
	// The arrival line names what has been decided. When candidates are offered, nothing has been
	// decided yet — the arrival is a guess pending the user's choice — so law 2 forbids stating it.
	if len(doc.ArrivalCandidates) == 0 {
		lines = append(lines, strings.TrimSpace(doc.Arrival.Stated))
	}
	return lines
}

// characterTurnOptions renders candidates as choice options; recommended = the one matching the arrival.
func characterTurnOptions(doc *genesisDoc) []map[string]any {
	opts := make([]map[string]any, 0, len(doc.ArrivalCandidates))
	rec := strings.TrimSpace(doc.Arrival.CanonicalName)
	for _, c := range doc.ArrivalCandidates {
		o := map[string]any{"label": c.CanonicalName, "implication": c.Descriptor + " — " + c.Why}
		if strings.TrimSpace(c.CanonicalName) == rec {
			o["recommended"] = true
		}
		opts = append(opts, o)
	}
	return opts
}

// scenarioTurnOptions renders authored scenarios as choice options.
func scenarioTurnOptions(scenarios []kickstartScenario) []map[string]any {
	opts := make([]map[string]any, 0, len(scenarios))
	for _, s := range scenarios {
		o := map[string]any{"label": s.Label, "implication": s.Why}
		if s.Recommended {
			o["recommended"] = true
		}
		opts = append(opts, o)
	}
	return opts
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
