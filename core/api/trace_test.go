package main

// Grounded Reasoning / Unit 3 — THE REASONING TRACE (BeatTrace → reasoning_log).
//
// The trace is a PURE CAPTURE of what the beat pipeline already computed (no new LLM): the decoded
// chain, per-attempt physics (the fact sheet + the move gate), the world-first decisions, the
// adjudicated ruling's reasoning→therefore→outcome, and the halt. It is TRUTH-REVEALING (it can carry a
// secret's truth-side reasoning) — RULINGS-2026-07-23 §9, design Unit 3.
//
// rung3 Task 5 deleted the singular POST /worlds/{w}/beat endpoint that once shipped BeatTrace under a
// debug-only `reasoning_log` JSON key (founder-approved clean cutover, no alias). AT THE TIME this
// comment was first written, the streaming replacement (/beats, /beats/continue) surfaced no trace on
// the wire at all — Task 3's frame protocol had no trace frame, debug or not — so there was no HTTP
// surface left to drive the two deleted tests through. rung3 Task 4 (commit adding the "trace" frame,
// beatsstream.go) restored that surface: a debug beat now emits a `trace` frame LAST carrying the full
// BeatTrace under `reasoning_log` — the exact key name the deleted endpoint used — and a non-debug beat
// emits no frame of that kind at all. TestBeats_DebugEmitsTraceFrame and
// TestBeats_NonDebugEmitsNoTraceFrame (beatsstream_test.go) are that surface's new home: they repoint
// the deleted TestTrace_NonDebugBeat_NoReasoningLogKey's and TestTrace_NonDebugBeat_ResponseShapeUnchanged's
// INTENT (a real player's stream must carry no reasoning key, present or null) at the frame protocol,
// rather than the deleted JSON envelope's exact key set — so the wall those two tests once stood for
// is enforced again, under new names, at the HTTP boundary.
//
// The trace MECHANISM itself is unchanged (BeatTrace/NewBeatTrace/Finish are pure Go, independent of
// any handler), so the two tests below that pin its CONTENT — move physics captured, an adjudicated
// ruling's reasoning→therefore→outcome captured — stay repointed to drive Orchestrator.RunBeat
// DIRECTLY with a real trace and inspect the Go struct, exactly the pattern this file's own WorldTurn
// tests below already use for the same reason (a real pressure roll needs the real seeded Drowned
// Lantern world, not the synthetic setupQueryWorld HTTP harness). traceOutcomeDirect is that harness's
// non-HTTP twin — narrower and cheaper than driving the same assertions through /beats' full frame
// protocol, and does not need to: the HTTP-level presence/absence of the trace frame is now
// beatsstream_test.go's job, not this file's.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// traceOutcomeDirect drives the SAME pipeline stages the deleted beatHandler.ServeHTTP once did for a
// single normal beat — decompose (scripted) → DecodeAndValidateChainV2 → Orchestrator.RunBeat with a
// real trace — without HTTP. debug=false threads a nil trace (RunBeat/Finish are nil-safe end to end,
// the same non-debug discipline the deleted handler followed). narrate is deliberately never reached:
// every trace assertion below is computed inside RunBeat, before narration, so leaving it out is a
// tighter test, not a smaller one.
func traceOutcomeDirect(t *testing.T, pool *pgxpool.Pool, debug bool, worldID, actorID, decomposeText, chainJSON, resolveRuling string) (BeatOutcome, *BeatTrace, *capturingResolveDriver) {
	t.Helper()
	ctx := context.Background()

	dec := NewFakeStructuredDriver("fake-structured:trace-direct", map[string]string{decomposeText: chainJSON})
	resolve := &capturingResolveDriver{name: "capture-resolve:trace-direct", ruling: resolveRuling}
	orc := &Orchestrator{
		DB:                pool,
		Resolve:           resolve,
		CognitionBatch:    NewFakeCognitionDriver(),
		CognitionIsolated: NewFakeCognitionDriver(),
		WorldActor:        NewFakeWorldActorDriver(),
	}

	bh := &beatHandler{pool: pool}
	pre, err := bh.payload(ctx, worldID, actorID)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	raw, err := dec.Generate(ctx, GenRequest{Payload: pre, Prompt: buildDecomposePrompt(pre, decomposeText), Schema: json.RawMessage(beatChainV2SchemaJSON)})
	if err != nil {
		t.Fatalf("decompose: %v", err)
	}
	chain, err := DecodeAndValidateChainV2(raw)
	if err != nil {
		t.Fatalf("decode chain: %v", err)
	}

	var trace *BeatTrace
	if debug {
		trace = NewBeatTrace(chain)
	}

	var startTick int64
	if err := pool.QueryRow(ctx, `SELECT fn_world_now($1::uuid)+1`, worldID).Scan(&startTick); err != nil {
		t.Fatalf("start tick: %v", err)
	}

	outcome, err := orc.RunBeat(ctx, worldID, actorID, chain, startTick, trace)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	trace.Finish(outcome) // nil-safe
	return outcome, trace, resolve
}

// TestTrace_DebugBeat_ReasoningLogHasMovePhysicsAndHalt is the brief's (a): a DEBUG move beat's trace
// carries the committed move's distance/duration (from the fact sheet) and the halt reason. K→bar is
// 8 m / 6 s (setupQueryWorld's geometry).
func TestTrace_DebugBeat_ReasoningLogHasMovePhysicsAndHalt(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	chain := `[{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"}]`
	outcome, trace, resolve := traceOutcomeDirect(t, pool, true, id.World, id.K, move, chain, `SHOULD NOT BE CALLED`)

	// A passthrough move never reaches the referee.
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (a passthrough move never reaches the referee)", resolve.calls)
	}
	// The move committed and completed.
	if outcome.HaltReason != "completed" || len(outcome.Committed) != 1 {
		t.Fatalf("outcome halt=%q committed=%v, want completed/1", outcome.HaltReason, outcome.Committed)
	}

	// (a) the trace is built under debug.
	if trace == nil {
		t.Fatalf("debug beat built no trace")
	}

	// The halt reason rides the trace.
	if trace.HaltReason != "completed" {
		t.Fatalf("trace.HaltReason = %q, want completed", trace.HaltReason)
	}
	// The decoded chain is captured (the move, bound to the bar id).
	if len(trace.Decompose) != 1 || trace.Decompose[0].Type != "ActorMoved" {
		t.Fatalf("trace.Decompose = %+v, want one ActorMoved", trace.Decompose)
	}
	// The per-attempt physics: the move's computed duration (6 s) + the fact sheet's distance/duration.
	if len(trace.Attempts) != 1 || trace.Attempts[0].Type != "ActorMoved" {
		t.Fatalf("trace.Attempts = %+v, want one ActorMoved attempt", trace.Attempts)
	}
	if trace.Attempts[0].MoveDurationS != 6 {
		t.Fatalf("trace.Attempts[0].MoveDurationS = %d, want 6 (CEIL(8/1.4))", trace.Attempts[0].MoveDurationS)
	}
	var fs struct {
		Targets []struct {
			DistanceM     json.Number `json:"distance_m"`
			MoveDurationS json.Number `json:"move_duration_s"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(trace.Attempts[0].FactSheet, &fs); err != nil {
		t.Fatalf("trace.Attempts[0].FactSheet not the fn_fact_sheet shape: %v\n%s", err, trace.Attempts[0].FactSheet)
	}
	if len(fs.Targets) != 1 {
		t.Fatalf("fact sheet targets = %+v, want the bar only", fs.Targets)
	}
	if !strings.HasPrefix(fs.Targets[0].DistanceM.String(), "8") {
		t.Fatalf("fact sheet distance_m = %q, want ~8 m to the bar", fs.Targets[0].DistanceM.String())
	}
	if fs.Targets[0].MoveDurationS.String() != "6" {
		t.Fatalf("fact sheet move_duration_s = %q, want 6 s", fs.Targets[0].MoveDurationS.String())
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestTrace_AdjudicatedRuling_ThereforeCaptured is the brief's (c): an adjudicated attempt drives the
// referee (a scripted ruling), and the trace records that ruling's reasoning→therefore→outcome. The
// distinctive reasoning string proves it is THIS ruling captured, not a coincidence.
func TestTrace_AdjudicatedRuling_ThereforeCaptured(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const marker = "Kade's grip tests the bar; it holds firm."
	ruling := `{"reasoning":"` + marker + `","therefore":"succeeds","outcome":{"kind":"resolved","events":[` +
		`{"type":"AttributeChanged","actor_id":"` + id.K + `","target_id":"` + id.Bar + `","truth":"The bar holds.","appearance":"The bar holds.","visible":true}` +
		`]}}`

	const grip = "I grip the bar to steady myself"
	chain := `[{"type":"AttributeChanged","stated":"` + grip + `","target_id":"` + id.Bar + `"}]`
	outcome, trace, resolve := traceOutcomeDirect(t, pool, true, id.World, id.K, grip, chain, ruling)

	// The attempt actually reached the referee (proving the ruling in the trace is a real capture).
	if resolve.calls != 1 {
		t.Fatalf("resolve calls = %d, want 1 (an AttributeChanged is adjudicated)", resolve.calls)
	}
	if outcome.HaltReason != "completed" {
		t.Fatalf("outcome.HaltReason = %q, want completed", outcome.HaltReason)
	}
	if trace == nil {
		t.Fatalf("debug beat built no trace")
	}
	if len(trace.Rulings) != 1 {
		t.Fatalf("trace.Rulings = %+v, want exactly one adjudicated ruling", trace.Rulings)
	}
	got := trace.Rulings[0]
	if got.Therefore != "succeeds" {
		t.Fatalf("trace.Rulings[0].Therefore = %q, want succeeds", got.Therefore)
	}
	if !strings.Contains(got.Reasoning, "grip tests the bar") {
		t.Fatalf("trace.Rulings[0].Reasoning = %q, want the scripted ruling's reasoning", got.Reasoning)
	}
	if got.Outcome != "resolved" {
		t.Fatalf("trace.Rulings[0].Outcome = %q, want resolved", got.Outcome)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// Living World / Task 10 (U7) — the world's-turn trace block. These two tests drive RunBeat DIRECTLY
// (not through the HTTP beatHandler) against the real seeded Drowned Lantern world, mirroring
// worldturn_test.go's own wtOrchestrator/wtForceTierFires/dlWorldID pattern (Task 9's forced-fire
// config), because the world's turn only fires against real pressure config + a real fire-log — the
// synthetic setupQueryWorld the rest of trace_test.go uses has neither. A real (non-nil) *BeatTrace is
// built via NewBeatTrace and threaded straight into RunBeat, then inspected as a Go struct (the same
// value beathandler.go would later serialize under reasoning_log — Step 4 below confirms the
// serialization + non-debug absence separately, reusing trace_test.go's existing non-debug assertions).

// TestTrace_WorldTurn_AllThreeTiersCapturedOnForcedFire is the brief's core test: with the small
// pressure tier forced to fire (medium/large pinned off) and a real trace threaded through RunBeat, the
// trace's world_turn block records ONE TraceWorldTurn for the single committed attempt whose Rolls
// slice carries ALL THREE tiers — small/medium/large, including the two that did NOT fire (Fork 6,
// "you can't tune what you can't see") — whose ClockDeltaS matches the attempt's own instant duration
// (2s, the seeded duration_class_seconds), and whose Eruption names the tier + event id that actually
// acted, cross-checked against the real world_eruption fire-log row.
func TestTrace_WorldTurn_AllThreeTiersCapturedOnForcedFire(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with wtForceTierFires' own config-restore t.Cleanup, so the restore
	// runs BEFORE the pool closes (ledger_test.go's/worldturn_test.go's documented ordering pattern).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	wtForceTierFires(t, ctx, pool, dlWorldID, "small")

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	chain := []Attempt{
		{Type: "Communicated", Stated: "Kade nods to Mara across the bar", ListenerID: wtMaraID, Content: "evening"},
	}
	trace := NewBeatTrace(chain)
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, trace)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, out.Committed) })
	if out.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed (a forced SMALL fire must not halt the beat — §5 lets small run on)", out.HaltReason)
	}

	if len(trace.WorldTurn) != 1 {
		t.Fatalf("trace.WorldTurn = %+v, want exactly 1 entry (one committed clock-advancing attempt)", trace.WorldTurn)
	}
	wt := trace.WorldTurn[0]

	if wt.ClockDeltaS != 2 {
		t.Fatalf("world_turn.clock_delta_s = %d, want 2 (the attempt's own instant duration)", wt.ClockDeltaS)
	}

	if len(wt.Rolls) != 3 {
		t.Fatalf("world_turn.rolls = %+v, want exactly 3 entries (small, medium, large — including non-firing)", wt.Rolls)
	}
	byTier := map[string]TraceRoll{}
	for _, r := range wt.Rolls {
		byTier[r.Tier] = r
	}
	for _, tier := range []string{"small", "medium", "large"} {
		if _, ok := byTier[tier]; !ok {
			t.Fatalf("world_turn.rolls missing tier %q: %+v", tier, wt.Rolls)
		}
	}
	if !byTier["small"].Fired {
		t.Fatalf("world_turn.rolls[small].fired = false, want true (climb_rate/cap forced to saturate)")
	}
	if byTier["small"].Chance != 1.0 {
		t.Fatalf("world_turn.rolls[small].chance = %v, want 1.0 (forced saturation)", byTier["small"].Chance)
	}
	if byTier["medium"].Fired || byTier["large"].Fired {
		t.Fatalf("world_turn.rolls = %+v, want medium AND large NOT fired (pinned to climb_rate=0/cap=0)", wt.Rolls)
	}

	if wt.Eruption == nil {
		t.Fatalf("world_turn.eruption = nil, want the small tier's committed intrusion")
	}
	if wt.Eruption.Type != "small" {
		t.Fatalf("world_turn.eruption.type = %q, want small", wt.Eruption.Type)
	}
	if len(wt.Eruption.IDs) != 1 || wt.Eruption.IDs[0] == "" {
		t.Fatalf("world_turn.eruption.ids = %v, want exactly one non-empty event id", wt.Eruption.IDs)
	}
	tier, _, eventID, ok := wtEruptionRowForEvent(t, ctx, pool, dlWorldID, out.Committed)
	if !ok {
		t.Fatalf("no world_eruption row references any committed event id %v — the fire-log write is missing", out.Committed)
	}
	if tier != "small" || eventID != wt.Eruption.IDs[0] {
		t.Fatalf("world_turn.eruption.ids[0] = %q, world_eruption row = (tier=%q, event_id=%q) — mismatch", wt.Eruption.IDs[0], tier, eventID)
	}
}

// TestTrace_WorldTurn_LedgerFireRecordsFiredAndSkipsRolls covers the ledger side of the world_turn
// block: a pending_event due at medium magnitude fires FIRST (ambiguity resolution #2a) and the pressure
// roll is skipped ENTIRELY (worldturn_test.go's TestRunWorldTurn_DueMediumPendingSkipsRoll already pins
// that world_eruption gains no new row here) — so the captured TraceWorldTurn must show the ledger's own
// committed event id under Fired, and NO rolls/eruption (the roll body never ran, so there is nothing
// honest to report there).
func TestTrace_WorldTurn_LedgerFireRecordsFiredAndSkipsRolls(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	// Every tier forced to certain-fire — if the roll body ran at all, it would fire too; the absence
	// of any Rolls/Eruption in the captured trace proves the roll was genuinely skipped, not merely that
	// pressure happened to be off (mirrors wtForceAllTiersFire's own rationale in worldturn_test.go).
	wtForceAllTiersFire(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	pendingSummary := "a commotion erupts from the cellar hatch"
	pendingAttempt := `{"type":"Communicated","stated":"` + pendingSummary + `","listener_id":"` + dlKadeID + `","content":"something below just broke loose"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick+2, "medium", wtMaraID, pendingAttempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	chain := []Attempt{
		{Type: "Communicated", Stated: "Kade orders a round before the crossing", ListenerID: wtMaraID, Content: "one more round"},
	}
	trace := NewBeatTrace(chain)
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, trace)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, out.Committed) })
	if out.HaltReason != "world_eruption" {
		t.Fatalf("HaltReason = %q, want world_eruption (the due medium pending_event applies the §5 cut)", out.HaltReason)
	}

	if len(trace.WorldTurn) != 1 {
		t.Fatalf("trace.WorldTurn = %+v, want exactly 1 entry", trace.WorldTurn)
	}
	wt := trace.WorldTurn[0]

	if wt.ClockDeltaS != 2 {
		t.Fatalf("world_turn.clock_delta_s = %d, want 2 (the attempt's own instant duration)", wt.ClockDeltaS)
	}
	if len(wt.Fired) != 1 {
		t.Fatalf("world_turn.fired_scheduled = %v, want exactly 1 (the ledger-committed pending event)", wt.Fired)
	}
	found := false
	for _, id := range out.Committed {
		if id == wt.Fired[0] {
			found = true
		}
	}
	if !found {
		t.Fatalf("world_turn.fired_scheduled[0] = %q, not among out.Committed %v", wt.Fired[0], out.Committed)
	}
	if len(wt.Rolls) != 0 {
		t.Fatalf("world_turn.rolls = %+v, want EMPTY — the roll body must never run when the ledger already fired medium/large", wt.Rolls)
	}
	if wt.Eruption != nil {
		t.Fatalf("world_turn.eruption = %+v, want nil — no pressure-roll eruption occurred", wt.Eruption)
	}
}

// TestTrace_WorldTurn_MultiTierHot_DebugAndNonDebugAgree closes the Task 10 review's Important gap:
// every existing test that reaches the roll loop (wtForceTierFires) pins the OTHER two tiers to
// climb_rate=0/cap=0 — structurally unable to fire — so no test ever put ALL THREE tiers simultaneously
// fire-eligible through the plain roll loop (not fireDuePending's medium/large ledger-skip branch, which
// is the only place wtForceAllTiersFire was previously exercised) and checked that only the
// FIRST-scanned tier acts, in BOTH debug and non-debug mode.
//
// wtForceAllTiersFire saturates every tier's chance to 1.0; there is NO pending ledger row, so the roll
// loop itself runs. The SAME fixture (same actor, same chain shape) runs twice — once with trace == nil
// (non-debug: runWorldTurn's `if trace == nil { break }` short-circuit), once with a real trace (debug:
// the loop keeps scanning all three tiers for capture) — asserting in BOTH runs that EXACTLY ONE
// world_eruption row was written, its tier is "large" (the first tier in the large→medium→small scan
// order — medium/small must NOT act even though they are ALSO hot; the biggest fired magnitude always
// wins, design Unit 6), and exactly one eruption event committed. This pins debug/non-debug equivalence
// directly and guards against a future "simplification" of the debug/non-debug guard silently letting a
// smaller tier act instead, or double-inserting.
func TestTrace_WorldTurn_MultiTierHot_DebugAndNonDebugAgree(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with wtForceAllTiersFire's own config-restore t.Cleanup, so the
	// restore runs BEFORE the pool closes (ledger_test.go's/worldturn_test.go's documented pattern).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	// One shared forced-hot config for BOTH runs below — every tier certain to fire, no pending row, so
	// the roll loop (not the ledger-skip branch) runs both times.
	wtForceAllTiersFire(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)

	run := func(t *testing.T, useTrace bool) {
		t.Helper()
		baseTick := wtBaseTick(t, ctx, pool)
		chain := []Attempt{
			{Type: "Communicated", Stated: "Kade murmurs something under his breath", ListenerID: wtMaraID, Content: "just thinking aloud"},
		}
		var trace *BeatTrace
		if useTrace {
			trace = NewBeatTrace(chain)
		}
		out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, trace)
		if err != nil {
			t.Fatalf("RunBeat: %v", err)
		}
		t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, out.Committed) })

		// Exactly ONE world_eruption row referencing this beat's committed ids — even though small,
		// medium, AND large were all certain to fire, at most one tier may ever act per turn.
		var totalCount int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM world_eruption WHERE world_id=$1 AND event_id = ANY($2)`,
			dlWorldID, out.Committed).Scan(&totalCount); err != nil {
			t.Fatalf("count world_eruption: %v", err)
		}
		if totalCount != 1 {
			t.Fatalf("world_eruption rows referencing this beat = %d, want exactly 1 (only ONE tier may act per turn, even with all three hot)", totalCount)
		}
		tier, _, eventID, ok := wtEruptionRowForEvent(t, ctx, pool, dlWorldID, out.Committed)
		if !ok {
			t.Fatalf("no world_eruption row references any committed event id %v", out.Committed)
		}
		if tier != "large" {
			t.Fatalf("world_eruption.tier = %q, want large (first in the large→medium→small scan order — medium/small must NOT act despite also being hot)", tier)
		}
		if eventID == "" {
			t.Fatalf("world_eruption.event_id is empty")
		}

		if useTrace {
			if len(trace.WorldTurn) != 1 {
				t.Fatalf("trace.WorldTurn = %+v, want exactly 1 entry", trace.WorldTurn)
			}
			wt := trace.WorldTurn[0]
			if len(wt.Rolls) != 3 {
				t.Fatalf("world_turn.rolls = %+v, want exactly 3 (small, medium, large — all captured even though only small acts)", wt.Rolls)
			}
			for _, r := range wt.Rolls {
				if !r.Fired {
					t.Fatalf("world_turn.rolls = %+v, want ALL THREE fired=true (wtForceAllTiersFire saturates every tier)", wt.Rolls)
				}
			}
			if wt.Eruption == nil || wt.Eruption.Type != "large" || len(wt.Eruption.IDs) != 1 || wt.Eruption.IDs[0] != eventID {
				t.Fatalf("world_turn.eruption = %+v, want {Type:large, IDs:[%s]}", wt.Eruption, eventID)
			}
		}
	}

	// Same fixture, both modes — the crux equivalence this test exists to pin.
	t.Run("non-debug", func(t *testing.T) { run(t, false) })
	t.Run("debug", func(t *testing.T) { run(t, true) })
}
