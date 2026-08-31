package main

// Governed-by: ADR-P021 — a built world commissions its own art, AFTER the transaction commits and outside it.
// Change what this file decides, and that ADR changes with it (D-9).

// worldgenesishandler.go — the three routes the create-world journey talks to.
//
//	POST /worlds/interview          →  one JSON turn: the next question, or nothing left to ask
//	POST /worlds/genesis            →  an SSE stream of world_genesis_frame/3, ending in a `choice`
//	                                    frame carrying the id of the world it already committed
//	POST /worlds/genesis/kickstart  →  one JSON turn per answer (world_kickstart_turn/2); the LAST
//	                                    answer is the arrival transaction that makes the world playable
//
// None of these hang off /worlds/{id}: there is no world yet, which is the whole point. They sit
// beside GET/POST /worlds as collection-level acts.
//
// WHY GENESIS STREAMS. A build is a long authored act with intermediate results — the same shape as a
// beat, and the same transport for the same reason. The alternative was a two-minute blank screen ending
// in a redirect, which tells the user nothing while it works and nothing about why if it fails. Every
// frame here names something real, in the world's own language; there is no percentage, no ETA and no
// stage checklist rendered from nothing (law 2: never invent a displayed value). While authoring (identity, then fill)
// is in flight the stream also says it is still alive — each liveness line states only measured
// fact (the call is running, and for how long), because a silent minute is indistinguishable from a
// hang and was reported as one.
//
// THREE PHASES, TWO TRANSACTIONS (durable-worlds spec, 2026-08-21). build() authors the whole world,
// narrates what it wrote, then COMMITS everything the world IS — entities, naming, history, minds,
// opening state, and the authored document itself — before ending the stream in a `choice` frame
// carrying the real world id. player_entity_id stays NULL, so the directory lists a real world that
// is not yet enterable. kickstart() takes it from there, one HTTP call per answer, keyed by the
// world id and resumable forever: the character turn (who the player is) authors the scenario
// options — and any people the identity references into existence — and asks again; the scenario
// turn (how it starts) runs the arrival transaction that makes the world playable. An empty answer
// is the resume path: it re-serves the pending question, so a process restart or an abandoned tab
// costs at most one re-asked question, never the world. A refused turn is a 422 with the stated
// reason and the world untouched — the expensive part can no longer be destroyed by the cheap part.

import (
	"context"
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
const worldGenesisFrameSchemaVersion = "world_genesis_frame/3"

// worldKickstartTurnSchemaVersion stamps the kickstart response. Same contract discipline.
const worldKickstartTurnSchemaVersion = "world_kickstart_turn/2"

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
	worldIdentityRoute    = regexp.MustCompile(`^/worlds/identity$`)
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
}

func NewWorldGenesisHandler(pool *pgxpool.Pool, debug bool, bridge *Bridge, images *imageClient) http.Handler {
	return &worldGenesisHandler{pool: pool, dbg: debug, bridge: bridge, images: images}
}

func (h *worldGenesisHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost &&
		(worldGenesisRoute.MatchString(r.URL.Path) || worldInterviewRoute.MatchString(r.URL.Path) ||
			worldIdentityRoute.MatchString(r.URL.Path) || genesisKickstartRoute.MatchString(r.URL.Path))
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
	// Identity is the Custom confirmation round-trip: the world_identity/1 the user just saw.
	// Fast omits it; genesis infers. Voice, when three sentences, is the author rewrite (§8).
	Identity json.RawMessage `json:"identity,omitempty"`
	Voice    []string        `json:"voice,omitempty"`
	// Depth is how much WORLD to author, 1-5, and it buys breadth rather than richness: how deep a
	// single entity goes is relevance's job (ADR-P027). 1 is a locality, 5 is continents. Omitted means
	// 1, which is the only setting anything has measured, and nothing raises it for the user yet.
	Depth int `json:"depth,omitempty"`
}

func (h *worldGenesisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case worldInterviewRoute.MatchString(r.URL.Path):
		h.interview(w, r)
	case worldIdentityRoute.MatchString(r.URL.Path):
		h.identity(w, r)
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

const worldIdentityConfirmSchemaVersion = "world_identity_confirm/1"

// identity infers world_identity/1 and returns the Custom confirmation view (design §8). Stateless:
// the client sends brief + answers, same as interview. Fast never calls this.
func (h *worldGenesisHandler) identity(w http.ResponseWriter, r *http.Request) {
	req, ok := readBrief(w, r)
	if !ok {
		return
	}
	ctx, costs := withCostSink(r.Context())
	start := time.Now()
	ident, err := inferIdentity(ctx, h.bridge.Driver(SeatWorldUnderstanding.Name), req.Brief, req.Answers)
	usd, tokIn, tokOut, cached, calls := costs.snapshot()
	log.Printf("world identity timing: ms=%d answers=%d calls=%d tok_in=%d cached=%d tok_out=%d cost_usd=%.6f",
		time.Since(start).Milliseconds(), len(req.Answers), calls, tokIn, cached, tokOut, usd)
	if err != nil {
		if IsGenesisRefusal(err) {
			writeJSONError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		log.Printf("world identity: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "the identity could not be inferred")
		return
	}
	raw, err := json.Marshal(ident)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "the identity could not be encoded")
		return
	}
	var identObj any
	_ = json.Unmarshal(raw, &identObj)
	cond := map[string]any{"text": ident.Condition.Text}
	if ident.Condition.Origin == "axiomatic" || ident.Condition.Origin == "contingent" {
		cond["origin"] = ident.Condition.Origin
	}
	body := map[string]any{
		"schema_version": worldIdentityConfirmSchemaVersion,
		"condition":      cond,
		"bargain":        map[string]any{"text": ident.Bargain.Text, "therefore": ident.Bargain.Therefore},
		"departure":      map[string]any{"neighbour": ident.Departure.Neighbour, "how_not": ident.Departure.HowNot},
		"content_demand": map[string]any{"text": ident.ContentDemand.Text, "therefore": ident.ContentDemand.Therefore},
		"register":       ident.Register,
		"voice":          ident.Voice,
		"identity":       identObj,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
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
	//
	// THE BUILD DOES NOT DIE WITH THE CONNECTION. Authoring used to run on r.Context(), so when the client
	// went away the whole build was cancelled mid-flight: measured 2026-08-28, an edge timeout at exactly
	// 899,997 ms killed two builds that had already spent minutes of model time, logging "context
	// canceled" from a seat call. A streamed response has a ceiling; a build has no business inheriting
	// it. WithoutCancel keeps the deadline-free lineage and the cost sink while dropping the
	// disconnect, so the work finishes and commits whether or not anyone is still listening — and the
	// existing resume path is how a client picks the world back up.
	//
	// Frames are still written to the live response: newFrameWriter is a no-op once the socket is gone,
	// so a disconnected build simply narrates to nobody.
	ctx, costs := withCostSink(context.WithoutCancel(r.Context()))
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
		h.fail(frames, refuse("%s", err.Error()), "")
		return
	}

	// Authoring first, and it is the slow part: identity, then one fill call per scheduled rule. Nothing is written yet, so
	// a refusal here costs nothing but the call. While the author writes, the stream keeps saying so —
	// every heartbeat carries only measured fact (the call is in flight, and for how long), never a
	// percentage or a stage. The heartbeats also keep an idle-sensitive proxy from closing the stream.
	_ = frames.emit("working", map[string]any{"stated": "Reading what you asked for."})
	stopHeartbeat := stillWriting(frames, start)
	doc, ident, err := authorWorld(ctx, h.bridge.Driver(SeatWorldUnderstanding.Name), h.bridge.Driver(SeatWorldFill.Name), h.bridge.Driver(SeatWorldFillReview.Name), req.Brief, req.Answers, req.Identity, req.Voice, req.Depth)
	stopHeartbeat()
	if err != nil {
		h.fail(frames, err, "")
		return
	}
	if ident != nil {
		_ = frames.emit("working", map[string]any{"stated": ident.Bargain.Text})
	}

	// The reply is back and already validated (authorWorld runs every belt check before returning), so
	// these counts are facts about a world that now exists on paper — said first because the user has
	// been staring at heartbeats for the whole authoring call.
	_ = frames.emit("working", map[string]any{"stated": fmt.Sprintf(
		"The author answered — %d places, %d people, %d objects, %d moments of history.",
		len(doc.Places), len(doc.Cast), len(doc.Objects), len(doc.History))})

	// Narration first — every line names authored content (law 2), commit or not.
	//
	// A DEAD SOCKET IS NOT A REASON TO THROW THE WORLD AWAY. This loop used to `return` when emit failed,
	// three lines above the commit. Measured 2026-08-28: a 23-call build spent 1,256 seconds and $0.046,
	// authored 11 places, 4 factions, 4 concepts, 6 people and 14 objects — and was discarded in silence
	// because the client had gone at the edge's 900-second cut and a narration frame could not be written.
	// No log, no commit, no world.
	//
	// Every other emit in this function already ignores its error deliberately; this was the one that did
	// not. Detaching the build's context (above) only got it as far as here. The user's connection is a
	// convenience for watching; it is never a condition of the world existing.
	for _, line := range genesisNarration(doc) {
		_ = frames.emit("working", map[string]any{"stated": line})
	}

	// The commit that makes the world durable: everything it IS, before anyone is anyone in it.
	// From here on, no failure in this stream or any later turn can cost the user the world.
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		h.fail(frames, fmt.Errorf("genesis: begin: %w", err), "")
		return
	}
	worldID, err = func() (string, error) {
		defer func() { _ = tx.Rollback(ctx) }()
		id, err := commitWorldContent(ctx, tx, doc, ident, req.Brief, req.ArtStyle)
		if err != nil {
			return "", err
		}
		return id, tx.Commit(ctx)
	}()
	if err != nil {
		worldID = "(none)"
		h.fail(frames, err, "")
		return
	}
	_ = frames.emit("working", map[string]any{"stated": "The world is written down — it keeps, even if you stop here.", "world_id": worldID})

	state := &kickstartState{}
	usd, in, out, cached, calls := costs.snapshot()
	state.Tally.add(usd, in, out, cached, calls)

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
			// The world is already committed and resumable; only this question failed. The frame
			// carries the world id so the surface can offer to continue rather than to rebuild.
			h.fail(frames, err, worldID)
			return
		}
		state.Identity = &k.Identity
		state.Scenarios = k.Scenarios
		state.NewCast = k.NewCast
		usd, in, out, cached, calls = costs.snapshot()
		state.Tally = kickstartTally{}
		state.Tally.add(usd, in, out, cached, calls) // snapshot is cumulative per sink; reset-then-add keeps the tally honest
		question = "How does it start?"
		options = scenarioTurnOptions(k.Scenarios)
	}
	if err := saveKickstartState(ctx, h.pool, worldID, state); err != nil {
		h.fail(frames, err, worldID)
		return
	}
	_ = frames.emit("choice", map[string]any{"world_id": worldID, "question": question, "options": options})
}

// genesisHeartbeatEvery is how often the build stream says the author is still writing. A var rather
// than a const only so the handler test can shrink it; nothing else writes it.
var genesisHeartbeatEvery = 10 * time.Second

// stillWriting emits a liveness frame at a steady interval until stopped. Each line carries only
// measured fact: the seat call is still in flight, and for how long — never a percentage, an ETA or a
// stage (law 2). The returned stop blocks until the ticker goroutine has exited, so the caller can go
// back to emitting on its own without racing it.
func stillWriting(frames *frameWriter, start time.Time) (stop func()) {
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		tick := time.NewTicker(genesisHeartbeatEvery)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				_ = frames.emit("working", map[string]any{"stated": fmt.Sprintf(
					"Still writing — %d seconds in.", int(time.Since(start).Seconds()))})
			}
		}
	}()
	return func() {
		close(done)
		<-exited
	}
}

// fail ends the stream honestly. A refusal carries the seat's own stated reason, because the user asked for
// something that could not become a world and deserves to know what; a fault carries a generic line,
// because "connection reset by peer" is not a sentence a player can act on. Both reach the log in full.
// worldID is non-empty once the content commit landed: the frame then names the world that survives
// the failure, so the surface can offer to continue it instead of rebuilding.
func (h *worldGenesisHandler) fail(frames *frameWriter, err error, worldID string) {
	payload := func(stated string) map[string]any {
		p := map[string]any{"stated": stated}
		if worldID != "" {
			p["world_id"] = worldID
		}
		return p
	}
	var refusal *genesisRefusal
	if errors.As(err, &refusal) {
		log.Printf("world genesis refused: %v", err)
		_ = frames.emit("refused", payload(refusal.why))
		return
	}
	log.Printf("world genesis failed: %v", err)
	_ = frames.emit("error", payload("the world could not be built"))
}

// kickstartRequest is the input to every kickstart turn: the world a build committed, and the
// player's answer — a chosen option's label, their own words entirely, or EMPTY, which means "show
// me the pending question" (the resume path).
type kickstartRequest struct {
	WorldID string `json:"world_id"`
	Answer  string `json:"answer"`
}

// kickstart turns one answer into the next question, or — on the scenario answer — into the arrival
// transaction that makes the world playable. state.Identity == nil is the whole state machine: nil
// means the character question is still open, set means only the opening remains.
//
// Nothing here can lose the world. State saves only after a turn succeeds; a refusal or a fault
// leaves the row exactly as it was, and the same request can simply be sent again. The final commit
// guards the player stamp with IS NULL, so a duplicate final answer answers 409 instead of racing.
func (h *worldGenesisHandler) kickstart(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, briefMaxBytes)
	var req kickstartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "that request did not parse")
		return
	}

	ctx, costs := withCostSink(r.Context())
	start := time.Now()

	c, err := loadCreation(ctx, h.pool, req.WorldID)
	switch {
	case errors.Is(err, errNoSuchWorld):
		writeJSONError(w, http.StatusNotFound, "no such world")
		return
	case errors.Is(err, errWorldAlreadyPlayable):
		writeJSONError(w, http.StatusConflict, errWorldAlreadyPlayable.Error())
		return
	case errors.Is(err, errNotResumable):
		writeJSONError(w, http.StatusConflict, errNotResumable.Error())
		return
	case err != nil:
		log.Printf("world kickstart: load: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "the world could not be read")
		return
	}

	answer := strings.TrimSpace(req.Answer)
	bail := func(err error) {
		var refusal *genesisRefusal
		if errors.As(err, &refusal) {
			log.Printf("world kickstart refused: %v", err)
			writeJSONError(w, http.StatusUnprocessableEntity, refusal.why)
			return
		}
		log.Printf("world kickstart failed: %v", err)
		writeJSONError(w, http.StatusBadGateway, "the opening could not be authored")
	}
	turn := func(question string, options []map[string]any) {
		body := map[string]any{
			"schema_version": worldKickstartTurnSchemaVersion,
			"done":           false,
			"question":       question,
			"options":        options,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	}

	// The identity the arrival will wear: chosen at the character turn, or stated in the brief and
	// carried by the authored document itself (that is what an absent candidates list means).
	identity := c.state.Identity
	if identity == nil && len(c.doc.ArrivalCandidates) == 0 {
		identity = &kickstartIdentity{
			Descriptor:    c.doc.Arrival.Descriptor,
			CanonicalName: c.doc.Arrival.CanonicalName,
		}
	}

	// mergedDoc is the world as the scenario turn must see it: the committed cast plus every person
	// the identity referenced into existence. The kickstart seat grounds custom openings against it,
	// the populated-places list grows with it, and validate() belts the whole of it.
	mergedDoc := func() *genesisDoc {
		if len(c.state.NewCast) == 0 {
			return c.doc
		}
		d := *c.doc
		d.Cast = append(append([]genesisActor{}, c.doc.Cast...), c.state.NewCast...)
		return &d
	}

	// Character turn — pending whenever no identity is settled yet.
	if identity == nil {
		if answer == "" {
			// Resume: re-serve the question exactly as the build stream first asked it.
			turn("Who are you here?", characterTurnOptions(c.doc))
			return
		}
		// A match against the offered candidates names who the player picked; no match means the
		// answer is their own words, and authorKickstart takes it exactly as who they are — free
		// text is a first-class answer here, not a rejection.
		who := answer
		if cand, ok := matchCandidateAnswer(c.doc.ArrivalCandidates, answer); ok {
			who = cand.CanonicalName
		}
		k, err := authorKickstart(ctx, h.bridge.Driver(SeatWorldKickstart.Name), c.doc, c.brief, who, "")
		if err != nil {
			bail(err)
			return
		}
		usd, in, out, cached, calls := costs.snapshot()
		c.state.Tally.add(usd, in, out, cached, calls)
		c.state.Identity = &k.Identity
		c.state.Scenarios = k.Scenarios
		c.state.NewCast = mergeNewCast(c.state.NewCast, k.NewCast)
		if err := saveKickstartState(ctx, h.pool, c.worldID, &c.state); err != nil {
			bail(err)
			return
		}
		turn("How does it start?", scenarioTurnOptions(k.Scenarios))
		return
	}

	// Scenario turn. A resume with no authored options yet (identity was stated in the brief and the
	// process restarted before the scenario call, or that call refused) authors them now.
	if answer == "" {
		if len(c.state.Scenarios) == 0 {
			k, err := authorKickstart(ctx, h.bridge.Driver(SeatWorldKickstart.Name), mergedDoc(), c.brief, identity.CanonicalName, "")
			if err != nil {
				bail(err)
				return
			}
			usd, in, out, cached, calls := costs.snapshot()
			c.state.Tally.add(usd, in, out, cached, calls)
			c.state.Identity = identity
			c.state.Scenarios = k.Scenarios
			c.state.NewCast = mergeNewCast(c.state.NewCast, k.NewCast)
			if err := saveKickstartState(ctx, h.pool, c.worldID, &c.state); err != nil {
				bail(err)
				return
			}
		}
		turn("How does it start?", scenarioTurnOptions(c.state.Scenarios))
		return
	}

	// The last answer. A match names the scenario chosen; no match grounds the player's own opening
	// as the single scenario (authorKickstart's second mode), against the merged world.
	chosen, matched := matchScenarioAnswer(c.state.Scenarios, answer)
	if !matched {
		k, err := authorKickstart(ctx, h.bridge.Driver(SeatWorldKickstart.Name), mergedDoc(), c.brief,
			identity.CanonicalName, answer)
		if err != nil {
			bail(err)
			return
		}
		// No tally add here: the sink is cumulative per request, and the aggregate below snapshots
		// it once — adding now double-counted the custom-opening call before (fix 7191477 relearned).
		c.state.NewCast = mergeNewCast(c.state.NewCast, k.NewCast)
		chosen = k.Scenarios[0]
	}

	fdoc := *mergedDoc()
	fdoc.Arrival = genesisArrival{
		Descriptor:    identity.Descriptor,
		CanonicalName: identity.CanonicalName,
		Place:         chosen.Place,
		Stated:        chosen.Stated,
		Why:           chosen.Why,
	}
	fdoc.ArrivalCandidates = nil
	if err := fdoc.validate(); err != nil {
		// Belt: the chosen arrival must still hit a populated place with an exit, and every person
		// the identity referenced must stand as a whole cast entry. A refusal here costs one answer.
		bail(err)
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		bail(fmt.Errorf("kickstart: begin: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := commitArrival(ctx, tx, c.worldID, &fdoc, c.state.NewCast); err != nil {
		if errors.Is(err, errWorldAlreadyPlayable) {
			writeJSONError(w, http.StatusConflict, errWorldAlreadyPlayable.Error())
			return
		}
		bail(err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		bail(fmt.Errorf("kickstart: commit: %w", err))
		return
	}

	// The aggregate line: the WHOLE build's spend — every seat call across authoring and every turn —
	// logged once, here, because this is the only moment a build-wide total exists to report.
	usd, in, out, cached, calls := costs.snapshot()
	c.state.Tally.add(usd, in, out, cached, calls)
	log.Printf("world genesis timing: total_ms=%d world=%s calls=%d tok_in=%d cached=%d tok_out=%d "+
		"cost_usd=%.6f session_usd=%.4f",
		time.Since(start).Milliseconds(), c.worldID, c.state.Tally.Calls, c.state.Tally.TokIn,
		c.state.Tally.Cached, c.state.Tally.TokOut, c.state.Tally.USD, sessionTotalUSD())
	if ceiling := genesisCostCeilingUSD(); ceiling > 0 && c.state.Tally.USD > ceiling {
		log.Printf("COST WARNING: building a world spent $%.4f (>$%.4f) across %d call(s) — "+
			"check the seat map for world_genesis", c.state.Tally.USD, ceiling, c.state.Tally.Calls)
	}

	// The world is playable; its pictures are commissioned after that, not before (build()'s own note
	// on this ordering applies unchanged — kickArt is detached and outlives this request).
	kickArt(h.pool, h.images, c.worldID)

	body := map[string]any{
		"schema_version": worldKickstartTurnSchemaVersion,
		"done":           true,
		"world": map[string]any{
			"id":           c.worldID,
			"display_name": strings.TrimSpace(fdoc.World.DisplayName),
			"tagline":      strings.TrimSpace(fdoc.World.Tagline),
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
