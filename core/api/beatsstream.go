package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// beatsRoute is POST /worlds/{w}/beats (plural). beatsContinueRoute is POST /worlds/{w}/beats/continue
// — same frame protocol, no body: "continue" IS an empty chain (RunBeat's own docstring,
// orchestrator.go) — it advances the moment by exactly one beat and never fast-forwards (C-6). Both
// are the ONLY beat write paths left after rung3 Task 5 deleted the singular /beat endpoint
// (founder-approved clean cutover, no alias, no deprecation shim).
var beatsRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beats$`)
var beatsContinueRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beats/continue$`)

// beatsStreamHandler serves both routes above, delivering the beat as a stream of validated frames
// instead of one buffered JSON response (design §4.8, plan rung3 Task 3). It reuses beatHandler's
// SURVIVING pipeline pieces wholesale — payload, speechTexts, narrateRoster, narrateMessages,
// buildDecomposePrompt (beathandler.go; its own HTTP entry point is gone, Task 5) — plus
// Orchestrator.RunBeat/RunReactionBeat, buildScene (scenehandler.go), and
// projectJourneyBlock/journeyBlock (journey.go) for the scene/journey frames. /beats/continue skips
// decompose entirely: an empty chain against an active journey IS the continue press (rung 2 commit
// 9ec9d7e) — the same beat, one fewer stage.
type beatsStreamHandler struct {
	pool   *pgxpool.Pool
	dbg    bool
	bridge *Bridge
}

// NewBeatsStreamHandler injects the bridge exactly like the old NewBeatHandler did (D-13) — CI uses
// fakes, the operator gate/production uses the live per-seat drivers, both behind the same interface.
func NewBeatsStreamHandler(pool *pgxpool.Pool, debug bool, bridge *Bridge) http.Handler {
	return &beatsStreamHandler{pool: pool, dbg: debug, bridge: bridge}
}

func (h *beatsStreamHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost &&
		(beatsRoute.MatchString(r.URL.Path) || beatsContinueRoute.MatchString(r.URL.Path))
}

// interpretationFrame carries the decoded intent chain — "how the input was understood", for the
// Intent lens later (plan Task 3's frame table). Chain reuses the Attempt shape wholesale
// (beatseats.go) — the SAME decoded/validated value RunBeat itself consumes, not a re-derived summary.
type interpretationFrame struct {
	Chain []Attempt `json:"chain"`
}

// narrationFrame carries one validated beatMessage (beathandler.go:291) — the existing founder-envelope
// shape, nested under "message" so its own "kind" (narration|speech|action) never collides with the
// frame envelope's "kind" (always "narration" for this frame type).
type narrationFrame struct {
	Message beatMessage `json:"message"`
}

// sceneFrame carries the Task 1 scene projection wholesale (scenehandler.go's sceneView, produced by
// buildScene) — nested under "scene" so its own nested "schema_version" (scene_current/1) is preserved
// rather than clobbered by the envelope's beat_frame/1.
type sceneFrame struct {
	Scene sceneView `json:"scene"`
}

// journeyFrame carries the Task 2 journey block (journey.go's journeyBlock, or nil when the viewer
// holds no active journey) — nested under "journey" so its own "kind" (travel|wait|watch) never
// collides with the frame envelope's "kind" (always "journey" for this frame type).
type journeyFrame struct {
	Journey *journeyBlock `json:"journey"`
}

// resultBlock mirrors the singular /beat endpoint's own "result" object exactly (beathandler.go:268-274)
// — committed, halt_reason, ticks_advanced, unresolved_candidates, telegraphs — so the two endpoints
// report the SAME beat outcome shape, one buffered, one streamed.
type resultBlock struct {
	Committed            []string `json:"committed"`
	HaltReason           string   `json:"halt_reason"`
	TicksAdvanced        int64    `json:"ticks_advanced"`
	UnresolvedCandidates []string `json:"unresolved_candidates"`
	Telegraphs           []string `json:"telegraphs"`
}

type resultFrame struct {
	Result resultBlock `json:"result"`
}

// errorFrame carries a PLAYER-SAFE message only — never a stack trace, never engine internals (plan
// Task 3). Every internal error is logged server-side (log.Printf, mirroring beathandler.go's own
// pattern) before this frame is emitted with a fixed, generic message.
type errorFrame struct {
	Message string `json:"message"`
}

// traceFrame carries the full BeatTrace (trace.go) — the developer's behind-the-curtain reasoning log
// (the decoded chain, per-attempt physics, world-first decisions, adjudicated rulings, the halt) —
// nested under "reasoning_log", the SAME JSON key the deleted singular /beat endpoint once used
// (beathandler.go, pre rung3 Task 5). TRUTH-REVEALING → DEBUG-ONLY (trace.go's own security
// invariant, RULINGS-2026-07-23 §9): this frame is emitted LAST, and ONLY when the handler runs in
// debug mode. A non-debug stream carries no frame of this KIND at all — not an empty one, not a null
// one — so ServeHTTP below gates the emit call itself on h.dbg rather than threading a maybe-nil
// Trace through an always-emitted frame.
type traceFrame struct {
	Trace *BeatTrace `json:"reasoning_log"`
}

// ServeHTTP runs the SAME beat pipeline the deleted beatHandler.ServeHTTP once did, stage for stage,
// and streams it as frames instead of buffering one JSON response.
//
// Everything BEFORE the chain decodes uses ordinary HTTP status codes (400/422/500/502), exactly like
// the old singular endpoint did — no SSE headers are sent and no frame is written, so a client that
// never gets a valid beat still gets an honest status code, not a 200 wrapping an error frame. The
// INSTANT the chain decodes, the interpretation frame is emitted: the response is now streaming,
// status 200 is already on the wire, and every failure from here on is a defined state (plan Task 3)
// — an `error` frame carrying a player-safe message, never a 5xx, because the status line has already
// gone out.
func (h *beatsStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := beatsRoute.FindStringSubmatch(r.URL.Path)
	continuePress := false
	if m == nil {
		m = beatsContinueRoute.FindStringSubmatch(r.URL.Path)
		continuePress = true
	}
	if m == nil || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	// bh is a minimal *beatHandler carrying only pool — the same construction buildScene uses
	// (scenehandler.go:115) to reach beatHandler.payload/speechTexts, the only entry points to the
	// perception-bound Candidates/Lines assembly and the verbatim-speech evidence.
	bh := &beatHandler{pool: h.pool}

	pre, err := bh.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}

	// The chain: decoded from the request body, or — on /beats/continue — the empty chain outright.
	// "continue" carries no body and decodes nothing (rung3 Task 5): an empty chain against an active
	// journey IS the continue press (RunBeat's own docstring, orchestrator.go), so there is no
	// decompose call, no driver round trip, and no §7 injection surface on this path.
	var chain []Attempt
	var playerText string
	if continuePress {
		chain = []Attempt{}
	} else {
		// §7 injection bound — identical cap to the deleted beatHandler.ServeHTTP (RULINGS-2026-07-24 §7).
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		var in struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		playerText = in.Text

		raw, err := h.bridge.Driver(SeatDecompose.Name).Generate(ctx,
			GenRequest{Payload: pre, Prompt: buildDecomposePrompt(pre, in.Text), Schema: json.RawMessage(beatChainV2SchemaJSON)})
		if err != nil {
			log.Printf("beats stream: decompose error: %v", err)
			http.Error(w, "decompose failed", http.StatusBadGateway)
			return
		}
		chain, err = DecodeAndValidateChainV2(raw)
		if err != nil {
			http.Error(w, "outside the closed vocabulary", http.StatusUnprocessableEntity)
			return
		}
	}

	frames, ok := newFrameWriter(w)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// From here on, status 200 is already on the wire — every failure path chooses an `error` frame
	// and returns; "it just stops" is the bug this design exists to prevent (plan Task 3).
	if err := frames.emit("interpretation", interpretationFrame{Chain: chain}); err != nil {
		log.Printf("beats stream: interpretation frame: %v", err)
		return
	}

	orc := &Orchestrator{
		DB:                h.pool,
		Resolve:           h.bridge.Driver(SeatResolve.Name),
		CognitionBatch:    h.bridge.Driver(SeatCognitionBatch.Name),
		CognitionIsolated: h.bridge.Driver(SeatCognitionIsolated.Name),
		WorldActor:        h.bridge.Driver(SeatWorldActor.Name),
		PlaceAuthor:       h.bridge.Driver(SeatPlaceAuthor.Name),
	}

	var startTick int64
	if err := h.pool.QueryRow(ctx, `SELECT fn_world_now($1::uuid)+1`, worldID).Scan(&startTick); err != nil {
		log.Printf("beats stream: start tick: %v", err)
		_ = frames.emit("error", errorFrame{Message: "the world's clock could not be read"})
		return
	}

	held, err := pendingHeldOutcomes(ctx, h.pool, worldID)
	if err != nil {
		log.Printf("beats stream: pendingHeldOutcomes error: %v", err)
		_ = frames.emit("error", errorFrame{Message: "held outcomes could not be read"})
		return
	}

	var trace *BeatTrace
	if h.dbg {
		trace = NewBeatTrace(chain)
	}

	var outcome BeatOutcome
	if len(held) > 0 {
		outcome, err = orc.RunReactionBeat(ctx, worldID, viewerID, chain, held, startTick, playerText, trace)
	} else {
		outcome, err = orc.RunBeat(ctx, worldID, viewerID, chain, startTick, trace)
	}
	if err != nil {
		log.Printf("beats stream: beat error: %v", err)
		_ = frames.emit("error", errorFrame{Message: "the world could not resolve that beat"})
		return
	}
	trace.Finish(outcome)

	post, err := bh.payload(ctx, worldID, viewerID)
	if err != nil {
		log.Printf("beats stream: post payload: %v", err)
		_ = frames.emit("error", errorFrame{Message: "the aftermath could not be perceived"})
		return
	}
	preIDs := make(map[string]bool, len(pre.LineIDs))
	for _, id := range pre.LineIDs {
		preIDs[id] = true
	}

	presentIDs, labelFor := narrateRoster(post, viewerID)
	speechTexts, err := bh.speechTexts(ctx, worldID, viewerID, startTick)
	if err != nil {
		log.Printf("beats stream: speechTexts lookup failed (belt runs with no evidence): %v", err)
		speechTexts = map[string][]string{} // fail toward repair, never crash the beat — mirrors beathandler.go
	}

	// Structured narration with ONE repair, then a plain prose fallback — byte-identical sequence to
	// beathandler.go:225-259. The belts (ghost speaker, verbatim speech) run INSIDE
	// DecodeAndValidateNarration, on the WHOLE segment array, before any of it becomes a frame — an
	// unvalidated segment never reaches narrateMessages, so it never reaches the wire (plan Task 3).
	nd := h.bridge.Driver(SeatNarrate.Name)
	var segments []NarrationSegment
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		prompt := buildNarratePrompt(post, viewerID, preIDs, outcome.HaltReason, outcome.QueryAnswers...)
		if attempt > 0 && lastErr != nil {
			prompt = buildNarrateRepairPrompt(post, viewerID, preIDs, outcome.HaltReason, lastErr.Error(), outcome.QueryAnswers...)
		}
		raw, genErr := nd.Generate(ctx, GenRequest{Payload: post, Prompt: prompt, Schema: json.RawMessage(narrationV1SchemaJSON)})
		if genErr != nil {
			log.Printf("beats stream: narrate structured Generate failed (attempt %d/2): %v", attempt+1, genErr)
			lastErr = genErr
			continue
		}
		segs, decErr := DecodeAndValidateNarration(raw, presentIDs, speechTexts)
		if decErr != nil {
			log.Printf("beats stream: narrate segment decode/validate failed (attempt %d/2): %v", attempt+1, decErr)
			lastErr = decErr
			continue
		}
		segments = segs
		break
	}
	if segments == nil {
		raw, genErr := nd.Generate(ctx, GenRequest{Payload: post, Prompt: buildNarratePlainPrompt(post, viewerID, preIDs, outcome.HaltReason, outcome.QueryAnswers...)})
		if genErr != nil {
			log.Printf("beats stream: narrate plain fallback Generate failed: %v", genErr)
			_ = frames.emit("error", errorFrame{Message: "the narrator could not find words"})
			return
		}
		segments = []NarrationSegment{{Kind: "narration", Text: raw}}
	}

	messages, _ := narrateMessages(segments, labelFor)
	for _, msg := range messages {
		if err := frames.emit("narration", narrationFrame{Message: msg}); err != nil {
			log.Printf("beats stream: narration frame: %v", err)
			return
		}
	}

	scene, err := buildScene(ctx, h.pool, worldID, viewerID, h.dbg)
	if err != nil {
		log.Printf("beats stream: buildScene: %v", err)
		_ = frames.emit("error", errorFrame{Message: "the scene could not be assembled"})
		return
	}
	if err := frames.emit("scene", sceneFrame{Scene: scene}); err != nil {
		log.Printf("beats stream: scene frame: %v", err)
		return
	}

	// journey prefers the journey THIS BEAT touched (outcome.Journey, set by runJourneyLeg the
	// instant a leg runs — journey.go) over a fresh activeJourney lookup: a journey that ARRIVES or
	// ENDS this very beat is no longer 'active', so activeJourney's status='active' WHERE clause would
	// return nil and this frame would go blank at exactly the beat that should report the arrival
	// (rung3 Task 5 correction — journey.go's own journeyBlock docstring flagged this). Every OTHER
	// beat (no journey touched at all) falls back to journeyBlock()'s fresh lookup, unchanged.
	journeyOrc := &Orchestrator{DB: h.pool}
	var journey *journeyBlock
	if outcome.Journey != nil {
		journey, err = journeyOrc.projectJourneyBlock(ctx, worldID, viewerID, outcome.Journey)
	} else {
		journey, err = journeyOrc.journeyBlock(ctx, worldID, viewerID)
	}
	if err != nil {
		log.Printf("beats stream: journeyBlock: %v", err)
		_ = frames.emit("error", errorFrame{Message: "the journey could not be assembled"})
		return
	}
	if err := frames.emit("journey", journeyFrame{Journey: journey}); err != nil {
		log.Printf("beats stream: journey frame: %v", err)
		return
	}

	result := resultBlock{
		Committed:            outcome.Committed,
		HaltReason:           outcome.HaltReason,
		TicksAdvanced:        outcome.TicksAdvanced,
		UnresolvedCandidates: outcome.UnresolvedCandidates,
		Telegraphs:           outcome.Telegraphs,
	}
	if err := frames.emit("result", resultFrame{Result: result}); err != nil {
		log.Printf("beats stream: result frame: %v", err)
		return
	}

	// The trace frame is LAST, and gated on h.dbg here rather than on trace itself: trace is only
	// ever non-nil when h.dbg (line ~223 above), so the two conditions already agree, but gating on
	// h.dbg directly keeps the debug-only discipline visible at the one call site that puts
	// truth-revealing reasoning on the wire (trace.go's security invariant) rather than resting on
	// trace's nilness as an implicit proxy.
	if h.dbg {
		if err := frames.emit("trace", traceFrame{Trace: trace}); err != nil {
			log.Printf("beats stream: trace frame: %v", err)
			return
		}
	}
}
