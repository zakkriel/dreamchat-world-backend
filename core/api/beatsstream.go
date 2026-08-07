package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// beatsRoute is POST /worlds/{w}/beats (plural) — the streaming sibling of beatRoute
// (beathandler.go:20, POST /worlds/{w}/beat, singular). The plan's clean cutover deletes the
// singular endpoint in Task 5; this task leaves it alone and working.
var beatsRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beats$`)

// beatsStreamHandler serves POST /worlds/{w}/beats — the SAME beat as beatHandler
// (beathandler.go:81), delivered as a stream of validated frames instead of one buffered JSON
// response (design §4.8, plan rung3 Task 3). It reuses the existing pipeline WHOLESALE: viewer
// resolution, the perception payload, decompose, DecodeAndValidateChainV2, RunBeat/RunReactionBeat,
// the narrate loop with its one repair and plain-prose fallback (all beathandler.go:102-286), plus
// buildScene (scenehandler.go) and journeyBlock (journey.go) for the scene/journey frames — the new
// endpoint is the same beat, differently delivered, never a second implementation of it.
type beatsStreamHandler struct {
	pool   *pgxpool.Pool
	dbg    bool
	bridge *Bridge
}

// NewBeatsStreamHandler injects the bridge exactly like NewBeatHandler (D-13) — CI uses fakes, the
// operator gate/production uses the live per-seat drivers, both behind the same interface.
func NewBeatsStreamHandler(pool *pgxpool.Pool, debug bool, bridge *Bridge) http.Handler {
	return &beatsStreamHandler{pool: pool, dbg: debug, bridge: bridge}
}

func (h *beatsStreamHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost && beatsRoute.MatchString(r.URL.Path)
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

// ServeHTTP runs the SAME beat pipeline beatHandler.ServeHTTP does (beathandler.go:102-285), stage for
// stage, and streams it as frames instead of buffering one JSON response.
//
// Everything BEFORE the chain decodes uses ordinary HTTP status codes (400/422/500/502), exactly like
// the singular endpoint — no SSE headers are sent and no frame is written, so a client that never gets
// a valid beat still gets an honest status code, not a 200 wrapping an error frame. The INSTANT the
// chain decodes, the interpretation frame is emitted: the response is now streaming, status 200 is
// already on the wire, and every failure from here on is a defined state (plan Task 3) — an `error`
// frame carrying a player-safe message, never a 5xx, because the status line has already gone out.
func (h *beatsStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := beatsRoute.FindStringSubmatch(r.URL.Path)
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

	// §7 injection bound — identical cap to beathandler.go's ServeHTTP (RULINGS-2026-07-24 §7).
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}

	// bh is a minimal *beatHandler carrying only pool — the same construction buildScene uses
	// (scenehandler.go:115) to reach beatHandler.payload/speechTexts, the only entry points to the
	// perception-bound Candidates/Lines assembly and the verbatim-speech evidence. beathandler.go
	// stays untouched; this borrows its exported-within-package methods, not its file.
	bh := &beatHandler{pool: h.pool}

	pre, err := bh.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}

	raw, err := h.bridge.Driver(SeatDecompose.Name).Generate(ctx,
		GenRequest{Payload: pre, Prompt: buildDecomposePrompt(pre, in.Text), Schema: json.RawMessage(beatChainV2SchemaJSON)})
	if err != nil {
		log.Printf("beats stream: decompose error: %v", err)
		http.Error(w, "decompose failed", http.StatusBadGateway)
		return
	}
	chain, err := DecodeAndValidateChainV2(raw)
	if err != nil {
		http.Error(w, "outside the closed vocabulary", http.StatusUnprocessableEntity)
		return
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
		outcome, err = orc.RunReactionBeat(ctx, worldID, viewerID, chain, held, startTick, in.Text, trace)
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

	// journeyBlock is read FRESH, post-resolve, exactly like buildScene's own journey field does
	// (scenehandler.go:188) — the same function, the same discipline (no server memory). A journey
	// that ENDED this very beat (arrived/ended) therefore projects as null here, same as it does inside
	// the scene frame above: activeJourney's own WHERE clause only ever returns an 'active' row
	// (journey.go:591-594's own docstring flags this — see report).
	journey, err := (&Orchestrator{DB: h.pool}).journeyBlock(ctx, worldID, viewerID)
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
}
