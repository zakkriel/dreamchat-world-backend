package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

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
// buildScene) — nested under "scene" so its own nested "schema_version" (scene_current/3) is preserved
// rather than clobbered by the envelope's beat_frame/3.
type sceneFrame struct {
	Scene sceneView `json:"scene"`
}

// journeyFrame carries the Task 2 journey block (journey.go's journeyBlock, or nil when the viewer
// holds no active journey) — nested under "journey" so its own "kind" (travel|wait|watch) never
// collides with the frame envelope's "kind" (always "journey" for this frame type).
type journeyFrame struct {
	Journey *journeyBlock `json:"journey"`
}

// unresolvedCandidate is ONE thing the player's words could have meant: the real id, plus the label
// that viewer's own knowledge puts on it (fn_display_name — the same path scene/current and the
// candidate whitelist already use, so this can never surface a name the viewer does not hold).
//
// v1 shipped bare ids. The frontend cannot name an id and will not invent one (B-1/D-7), so the
// "which did you mean?" affordance could never render and it correctly fell back to generic copy —
// the ask was unanswerable because it was unsayable.
//
// NOTE: two candidates may legitimately carry the SAME label. That is not a defect to paper over,
// it IS the ambiguity — the seeded world has two hooded figures a viewer cannot tell apart, which is
// precisely why UNRESOLVED fires. Callers must key on `id` (or list position) and must never assume
// labels are distinct.
type unresolvedCandidate struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// resultBlock is the beat outcome the stream reports — committed, halt_reason, ticks_advanced,
// unresolved_candidates, telegraphs.
type resultBlock struct {
	Committed            []string              `json:"committed"`
	HaltReason           string                `json:"halt_reason"`
	TicksAdvanced        int64                 `json:"ticks_advanced"`
	UnresolvedCandidates []unresolvedCandidate `json:"unresolved_candidates"`
	Telegraphs           []string              `json:"telegraphs"`
}

// labelCandidates renders each unresolved id the way the VIEWER would name it, through
// fn_display_names_distinct — the same set-aware path the candidate whitelist uses, so the words the
// player is offered here are exactly the words they can say back.
//
// Set-aware matters most on THIS surface: an unresolved list is by definition a group of things one
// phrase named equally well, so it is the likeliest place for two labels to collide. Where perceived
// detail can separate them it is added ("… by the bar"); where it genuinely cannot, both entries read
// the same, which is the honest answer and the founder's explicit ruling — see
// fn_display_names_distinct's own note.
//
// It never reaches for a canonical registry name the viewer may not know (§3 naming reach), so the
// "which did you mean?" list cannot leak a name the ask itself proves the player does not have.
// Returns a non-nil empty slice for no candidates, so the payload carries `[]` and never `null`.
func labelCandidates(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string, ids []string) ([]unresolvedCandidate, error) {
	out := make([]unresolvedCandidate, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	labels, err := distinctLabels(ctx, pool, worldID, viewerID, ids)
	if err != nil {
		return nil, fmt.Errorf("labelCandidates: %w", err)
	}
	for _, id := range ids {
		out = append(out, unresolvedCandidate{ID: id, Label: labels[id]})
	}
	return out, nil
}

// distinctLabels is the one call site for fn_display_names_distinct: ids in, viewer-facing labels out,
// collisions already broken by perceived detail. Keyed by id rather than returned as a slice so both
// callers (the unresolved list, and the candidate whitelist's in-place relabel) can use it without
// either depending on the other's ordering.
func distinctLabels(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string, ids []string) (map[string]string, error) {
	labels := make(map[string]string, len(ids))
	rows, err := pool.Query(ctx,
		`SELECT entity_id, label FROM fn_display_names_distinct($1::uuid, $2::uuid, $3::uuid[])`,
		worldID, viewerID, ids)
	if err != nil {
		return nil, fmt.Errorf("fn_display_names_distinct: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, fmt.Errorf("fn_display_names_distinct: scan: %w", err)
		}
		labels[id] = label
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fn_display_names_distinct: %w", err)
	}
	return labels, nil
}

// relabelDistinct rewrites a candidate slice's labels in place. Only labels that COLLIDE change; a
// candidate whose name was already unique comes back byte-identical.
func relabelDistinct(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string, cands []Candidate) error {
	if len(cands) == 0 {
		return nil
	}
	ids := make([]string, 0, len(cands))
	for _, c := range cands {
		ids = append(ids, c.ID)
	}
	labels, err := distinctLabels(ctx, pool, worldID, viewerID, ids)
	if err != nil {
		return fmt.Errorf("relabelDistinct: %w", err)
	}
	for i := range cands {
		if l, ok := labels[cands[i].ID]; ok && l != "" {
			cands[i].Name = l
		}
	}
	return nil
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
	if writeNoViewer(w, err) {
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
	var decomposeText string
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

		decomposeText = in.Text
	}

	// The total the founder actually asked for: "how long did one reply take". The per-seat lines
	// (timedDriver) say where it went; this says what he waited. Deferred so it is logged on EVERY
	// exit including the error frames, because a beat that died slowly is the one worth knowing about.
	beatStart := time.Now()
	orc := &Orchestrator{
		DB:                h.pool,
		Resolve:           h.bridge.Driver(SeatResolve.Name),
		CognitionBatch:    h.bridge.Driver(SeatCognitionBatch.Name),
		CognitionIsolated: h.bridge.Driver(SeatCognitionIsolated.Name),
		WorldActor:        h.bridge.Driver(SeatWorldActor.Name),
		PlaceAuthor:       h.bridge.Driver(SeatPlaceAuthor.Name),
	}
	defer func() {
		adj, fanout := orc.BeatCounters()
		log.Printf("beat timing: total_ms=%d world=%s continue=%v adjudications=%d npc_fanout=%d",
			time.Since(beatStart).Milliseconds(), worldID, continuePress, adj, fanout)
	}()

	frames, ok := newFrameWriter(w)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// DECOMPOSE RUNS INSIDE THE STREAM, and that is the point of it being here rather than above.
	//
	// It used to run before the stream existed and answer a failure with http.Error — a 502 with a
	// text/plain body to a client that asked for text/event-stream. To a browser mid-play that is not
	// an error, it is the connection dying: no frame, no message, and an edge proxy free to turn it
	// into its own 502 page. The founder saw exactly this when the model spent its whole budget on
	// reasoning tokens. Every other failure in this handler has emitted an honest `error` frame since
	// the streaming design landed ("it just stops" is the bug this design exists to prevent); these
	// two were simply on the wrong side of the line.
	if !continuePress {
		raw, err := h.bridge.Driver(SeatDecompose.Name).Generate(ctx,
			GenRequest{Payload: pre, Prompt: buildDecomposePrompt(pre, decomposeText), Schema: json.RawMessage(beatChainV2SchemaJSON)})
		if err != nil {
			log.Printf("beats stream: decompose error: %v", err)
			_ = frames.emit("error", errorFrame{Message: "the world could not read that"})
			return
		}
		chain, err = DecodeAndValidateChainV2(raw)
		if err != nil {
			// The leash refusing a chain is a REAL answer about this input, not a broken server, and
			// the player is owed a sentence rather than a status code.
			log.Printf("beats stream: decompose produced an invalid chain: %v", err)
			_ = frames.emit("error", errorFrame{Message: "the world could not make sense of that — try saying it another way"})
			return
		}
	}

	// From here on, status 200 is already on the wire — every failure path chooses an `error` frame
	// and returns; "it just stops" is the bug this design exists to prevent (plan Task 3).
	if err := frames.emit("interpretation", interpretationFrame{Chain: chain}); err != nil {
		log.Printf("beats stream: interpretation frame: %v", err)
		return
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
	// THE NAMING WALL (B-1, I-3, naming reach §3). The source seam renders perception content per
	// holder, so a clean world hands the narrator nothing to leak; this is the belt for what the seat
	// invents on its own. Loaded once per beat — the set only changes when the viewer learns a name,
	// which is itself a canon event.
	wall, err := loadNamingWall(ctx, bh.pool, worldID, viewerID)
	if err != nil {
		// Fail CLOSED is not an option (it would kill the beat over a projection read) and fail-silent
		// is what got us here, so: fail LOUD and keep the source-seam fix, which is the real guarantee.
		log.Printf("beats stream: NAMING WALL could not be loaded (%v) — narration belt is DOWN this beat", err)
		wall = nil
	}
	speechTexts, err := bh.speechTexts(ctx, worldID, viewerID, startTick)
	if err != nil {
		log.Printf("beats stream: speechTexts lookup failed (belt runs with no evidence): %v", err)
		speechTexts = map[string][]string{} // fail toward repair, never crash the beat — mirrors beathandler.go
	}

	// Structured narration with ONE repair, then a plain prose fallback — byte-identical sequence to
	// beathandler.go:225-259. The belts (ghost speaker, verbatim speech) run INSIDE
	// DecodeAndValidateNarration, on the WHOLE segment array, before any of it becomes a frame — an
	// unvalidated segment never reaches narrateMessages, so it never reaches the wire (plan Task 3).
	//
	// rung3 Task 4: when the bound narrate driver ALSO implements StreamingDriver, narrateStream runs
	// this same belt per LINE and emits each validated narration frame the instant it completes,
	// rather than waiting for the whole reply — the plan's "real line-by-line, where the driver can".
	// A streaming attempt that validates at least one line is treated as this beat's narration in
	// full: those frames are ALREADY on the wire, so there is no repair/fallback for a streaming
	// attempt — only the ordinary generate-then-validate-then-emit loop below retries. A streaming
	// attempt that validates NOTHING has put nothing on the wire yet, so it falls straight through to
	// that identical loop, unchanged.
	// One set of belts for every attempt this beat: the same segment must be judged identically by the
	// streaming path, the repair loop and the fallback, or a rejection becomes a matter of which code
	// path happened to run.
	belts := NarrationBelts{
		PresentIDs:  presentIDs,
		SpeechTexts: speechTexts,
		Wall:        wall,
		Player:      newPlayerVoice(playerText),
	}
	nd := h.bridge.Driver(SeatNarrate.Name)
	var segments []NarrationSegment
	var lastErr error
	streamed := false
	if sd, ok := nd.(StreamingDriver); ok {
		prompt := buildNarratePrompt(post, viewerID, preIDs, outcome.HaltReason, outcome.QueryAnswers...)
		req := GenRequest{Payload: post, Prompt: prompt, Schema: json.RawMessage(narrationV1SchemaJSON)}
		segs, err := narrateStream(ctx, sd, req, belts, labelFor, frames)
		switch {
		case err != nil && len(segs) == 0:
			log.Printf("beats stream: streaming narrate produced no valid line (%v), falling back to the ordinary attempt loop", err)
		case err != nil:
			log.Printf("beats stream: streaming narrate failed after %d line(s) were already emitted: %v", len(segs), err)
			return
		case len(segs) > 0:
			segments, streamed = segs, true
		default:
			log.Printf("beats stream: streaming narrate produced no valid line, falling back to the ordinary attempt loop")
		}
	}

	if !streamed {
		for attempt := range 2 {
			prompt := buildNarratePrompt(post, viewerID, preIDs, outcome.HaltReason, outcome.QueryAnswers...)
			if attempt > 0 && lastErr != nil {
				prompt = buildNarrateRepairPrompt(post, viewerID, preIDs, outcome.HaltReason, lastErr.Error(), outcome.QueryAnswers...)
			}
			raw, genErr := nd.Generate(ctx, GenRequest{Payload: post, Prompt: prompt, Schema: json.RawMessage(narrationV1SchemaJSON), Repair: attempt > 0})
			if genErr != nil {
				log.Printf("beats stream: narrate structured Generate failed (attempt %d/2): %v", attempt+1, genErr)
				lastErr = genErr
				continue
			}
			segs, decErr := DecodeAndValidateNarration(raw, belts)
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
			// The plain fallback bypasses every belt by design (it exists because the structured path
			// failed), so the wall is applied here as a SCRUB rather than a rejection: there is no third
			// attempt to spend, and a breach must not reach the player just because the narrator is
			// already having a bad beat.
			if v := wall.Violations(raw); len(v) > 0 {
				log.Printf("NAMING WALL: plain fallback narration leaked %v for viewer %s — scrubbed before emit", v, viewerID)
				raw = wall.Scrub(raw)
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

	// Labels are attached HERE, at the API boundary, not in the orchestrator: the engine reasons in
	// ids (it must — ids are what bind), and whose knowledge renders them is a projection question.
	unresolved, err := labelCandidates(ctx, h.pool, worldID, viewerID, outcome.UnresolvedCandidates)
	if err != nil {
		log.Printf("beats stream: labelCandidates: %v", err)
		_ = frames.emit("error", errorFrame{Message: "the world could not name what you meant"})
		return
	}
	// NPC telegraph wind-ups are cognition-seat text (`Attempt.Stated`) travelling straight to the
	// player with no validation loop of their own — the second path the founder's leak could have
	// taken. No model to re-ask here, so the wall scrubs and reports.
	if breached := wall.scrubAll(outcome.Telegraphs); len(breached) > 0 {
		log.Printf("NAMING WALL: telegraph text leaked %v for viewer %s — scrubbed before emit", breached, viewerID)
	}
	result := resultBlock{
		Committed:            outcome.Committed,
		HaltReason:           outcome.HaltReason,
		TicksAdvanced:        outcome.TicksAdvanced,
		UnresolvedCandidates: unresolved,
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

// narrationLineSplitter incrementally extracts complete top-level elements of a JSON array of objects
// (narration/1's shape: `[{"speaker_id":…,"kind":…,"text":…}, …]`) from a growing text buffer fed one
// delta at a time — the mechanism that lets narrateStream validate and emit each narration LINE (one
// array element) the moment its closing brace arrives, rather than waiting for the whole array to
// close. It tracks object/array nesting depth and string/escape state so a brace or comma inside a
// quoted text field is never mistaken for structure.
type narrationLineSplitter struct {
	buf       []byte
	depth     int // nesting depth of [ and { seen so far (the outer array itself is depth 1)
	inString  bool
	escaped   bool
	elemStart int // byte offset in buf where the in-flight element began, or -1 when not in one
}

func newNarrationLineSplitter() *narrationLineSplitter {
	return &narrationLineSplitter{elemStart: -1}
}

// feed appends delta to the buffer and returns every element (raw JSON object text) that became
// complete as a result, in order.
func (s *narrationLineSplitter) feed(delta string) []string {
	var elems []string
	for _, c := range []byte(delta) {
		pos := len(s.buf)
		s.buf = append(s.buf, c)
		if s.inString {
			switch {
			case s.escaped:
				s.escaped = false
			case c == '\\':
				s.escaped = true
			case c == '"':
				s.inString = false
			}
			continue
		}
		switch c {
		case '"':
			s.inString = true
		case '{', '[':
			if s.depth == 1 && s.elemStart == -1 && c == '{' {
				s.elemStart = pos
			}
			s.depth++
		case '}', ']':
			s.depth--
			if s.depth == 1 && s.elemStart != -1 && c == '}' {
				elems = append(elems, string(s.buf[s.elemStart:pos+1]))
				s.elemStart = -1
			}
		}
	}
	return elems
}

// narrateStream drives a StreamingDriver's GenerateStream call, splitting the growing narration/1
// array text into complete elements as they arrive and running EACH one through the SAME belt
// (DecodeAndValidateNarration, narration.go) an ordinary batch call would — one line, wrapped as its
// own single-element array, so the ghost-speaker/verbatim-speech checks judge it exactly as they
// always have. A line that passes is turned into its beatMessage (narrateMessages' own per-segment
// rule, repeated here rather than imported so this file never widens narration.go's surface) and
// emitted as a real narration frame BEFORE GenerateStream returns — the entire point of Task 4 (plan:
// "the first narration frame is written before the driver returns"; "that ordering IS the feature").
//
// Once a line is emitted here it is ALREADY on the wire, so this function never retries: an invalid
// line is simply never emitted (the belt still bites, mid-stream, exactly as
// TestBeats_GhostSpeakerNeverReachesTheWire pins for the non-streaming path) and the stream is left to
// run to completion. Returns the segments it emitted; the caller (ServeHTTP) treats a zero-segment,
// no-error-yet-nothing-emitted result as "nothing reached the wire, fall back to the ordinary
// generate → validate → emit loop", and a non-nil error the SAME way once nothing was emitted.
func narrateStream(ctx context.Context, sd StreamingDriver, req GenRequest, belts NarrationBelts, labelFor map[string]string, frames *frameWriter) ([]NarrationSegment, error) {
	splitter := newNarrationLineSplitter()
	var segments []NarrationSegment
	var emitErr error
	_, genErr := sd.GenerateStream(ctx, req, func(delta string) {
		if emitErr != nil {
			return
		}
		for _, raw := range splitter.feed(delta) {
			segs, err := DecodeAndValidateNarration("["+raw+"]", belts)
			if err != nil {
				log.Printf("beats stream: streamed narration line rejected (never emitted): %v", err)
				continue
			}
			seg := segs[0]
			msg := beatMessage{SpeakerID: seg.SpeakerID, Kind: seg.Kind, Text: seg.Text}
			if seg.SpeakerID != nil {
				msg.SpeakerLabel = labelFor[*seg.SpeakerID]
			}
			if err := frames.emit("narration", narrationFrame{Message: msg}); err != nil {
				emitErr = err
				return
			}
			segments = append(segments, seg)
		}
	})
	if genErr != nil {
		return segments, genErr
	}
	return segments, emitErr
}
