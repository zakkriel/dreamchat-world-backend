package main

// Grounded Reasoning / Unit 3 — THE REASONING TRACE (BeatTrace → reasoning_log).
//
// The trace is a PURE CAPTURE of what the beat pipeline already computed (no new LLM): the decoded
// chain, per-attempt physics (the fact sheet + the move gate), the world-first decisions, the
// adjudicated ruling's reasoning→therefore→outcome, and the halt. It is TRUTH-REVEALING (it can carry a
// secret's truth-side reasoning), so it is emitted under `reasoning_log` ONLY when the handler is in
// debug mode; a non-debug response carries NO `reasoning_log` key at all (the security-relevant
// invariant — a real player never gets debug, RULINGS-2026-07-23 §9, design Unit 3).
//
// These tests drive the WHOLE path through the REAL HTTP beatHandler (the station_f/query exit-gate
// pattern), toggling the handler's debug flag, and assert on the shipped JSON. Each test mints a fresh,
// random world (setupQueryWorld) so counts see only its rows.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// reasoningLog mirrors the BeatTrace JSON the handler emits under `reasoning_log` — the developer view.
type reasoningLog struct {
	Decompose []struct {
		Type   string   `json:"type"`
		Stated string   `json:"stated"`
		IDs    []string `json:"ids"`
	} `json:"decompose"`
	WorldFirst []struct {
		ActorID  string `json:"actor_id"`
		Decision string `json:"decision"`
		Stated   string `json:"stated"`
	} `json:"world_first"`
	Attempts []struct {
		Type          string          `json:"type"`
		Stated        string          `json:"stated"`
		FactSheet     json.RawMessage `json:"fact_sheet"`
		MoveDurationS int64           `json:"move_duration_s"`
	} `json:"attempts"`
	Rulings []struct {
		ActorIDs  []string `json:"actor_ids"`
		Reasoning string   `json:"reasoning"`
		Therefore string   `json:"therefore"`
		Outcome   string   `json:"outcome"`
	} `json:"rulings"`
	HaltReason string   `json:"halt_reason"`
	Committed  []string `json:"committed"`
}

type traceBeatResp struct {
	Narration string          `json:"narration"`
	Messages  json.RawMessage `json:"messages"`
	Result    struct {
		Committed     []string `json:"committed"`
		HaltReason    string   `json:"halt_reason"`
		TicksAdvanced int64    `json:"ticks_advanced"`
	} `json:"result"`
	ReasoningLog *reasoningLog `json:"reasoning_log"`
}

// traceHandler builds the real HTTP handler with a scripted decompose table + a fixed-ruling resolve
// driver (a sentinel when the beat should never reach the referee) + a deterministic text narrator, so a
// test can toggle debug and read back the shipped reasoning_log. Cognition/world-actor stay quiet fakes.
func traceHandler(t *testing.T, pool *pgxpool.Pool, debug bool, decomposeText, chainJSON, resolveRuling string) (http.Handler, *capturingResolveDriver) {
	t.Helper()
	dec := NewFakeStructuredDriver("fake-structured:trace", map[string]string{decomposeText: chainJSON})
	resolve := &capturingResolveDriver{name: "capture-resolve:trace", ruling: resolveRuling}
	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:         dec,
		SeatNarrate.Name:           NewFakeTextDriver("fake-text:trace"),
		SeatResolve.Name:           resolve,
		SeatCognitionBatch.Name:    NewFakeCognitionDriver(),
		SeatCognitionIsolated.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:        NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return NewBeatHandler(pool, debug, bridge), resolve
}

// TestTrace_DebugBeat_ReasoningLogHasMovePhysicsAndHalt is the brief's (a): a DEBUG move beat ships a
// `reasoning_log` object carrying the committed move's distance/duration (from the fact sheet) and the
// halt reason. K→bar is 8 m / 6 s (setupQueryWorld's geometry).
func TestTrace_DebugBeat_ReasoningLogHasMovePhysicsAndHalt(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	chain := `[{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"}]`
	h, resolve := traceHandler(t, pool, true, move, chain, `SHOULD NOT BE CALLED`)

	code, body := postBeat(h, id.World, id.K, move)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var r traceBeatResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}

	// A passthrough move never reaches the referee.
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (a passthrough move never reaches the referee)", resolve.calls)
	}
	// The move committed and completed.
	if r.Result.HaltReason != "completed" || len(r.Result.Committed) != 1 {
		t.Fatalf("result halt=%q committed=%v, want completed/1", r.Result.HaltReason, r.Result.Committed)
	}

	// (a) the reasoning_log is present under debug.
	if r.ReasoningLog == nil {
		t.Fatalf("debug beat shipped NO reasoning_log\nbody: %s", body)
	}
	rl := r.ReasoningLog

	// The halt reason rides the trace.
	if rl.HaltReason != "completed" {
		t.Fatalf("reasoning_log.halt_reason = %q, want completed", rl.HaltReason)
	}
	// The decoded chain is captured (the move, bound to the bar id).
	if len(rl.Decompose) != 1 || rl.Decompose[0].Type != "ActorMoved" {
		t.Fatalf("reasoning_log.decompose = %+v, want one ActorMoved", rl.Decompose)
	}
	// The per-attempt physics: the move's computed duration (6 s) + the fact sheet's distance/duration.
	if len(rl.Attempts) != 1 || rl.Attempts[0].Type != "ActorMoved" {
		t.Fatalf("reasoning_log.attempts = %+v, want one ActorMoved attempt", rl.Attempts)
	}
	if rl.Attempts[0].MoveDurationS != 6 {
		t.Fatalf("reasoning_log.attempts[0].move_duration_s = %d, want 6 (CEIL(8/1.4))", rl.Attempts[0].MoveDurationS)
	}
	var fs struct {
		Targets []struct {
			DistanceM     json.Number `json:"distance_m"`
			MoveDurationS json.Number `json:"move_duration_s"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(rl.Attempts[0].FactSheet, &fs); err != nil {
		t.Fatalf("reasoning_log.attempts[0].fact_sheet not the fn_fact_sheet shape: %v\n%s", err, rl.Attempts[0].FactSheet)
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

// TestTrace_NonDebugBeat_NoReasoningLogKey is the brief's (b) — the SECURITY-relevant assertion: a
// NON-debug beat's response has NO `reasoning_log` key AT ALL (not null — absent). The trace is
// truth-revealing, so a real player (never in debug) must never receive it.
func TestTrace_NonDebugBeat_NoReasoningLogKey(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	chain := `[{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"}]`
	// debug=false. ResolveViewer ignores ?viewer= without debug and resolves the actor named 'Player'
	// (= K in setupQueryWorld), so the same move runs.
	h, _ := traceHandler(t, pool, false, move, chain, `SHOULD NOT BE CALLED`)

	code, body := postBeat(h, id.World, id.K, move)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}

	// THE WALL: the raw response text must not carry the key at all.
	if strings.Contains(body, "reasoning_log") {
		t.Fatalf("non-debug response leaked a reasoning_log key (truth-revealing → debug-only):\n%s", body)
	}
	// Structural proof of absence (not merely null): the key is not present in the top-level object.
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}
	if _, present := top["reasoning_log"]; present {
		t.Fatalf("non-debug response has a reasoning_log key (must be ABSENT):\n%s", body)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestTrace_NonDebugBeat_ResponseShapeUnchanged is the brief's (d): the nil trace changes NOTHING. A
// non-debug beat's response carries exactly the pre-existing key set {schema_version, narration,
// messages, result} — the trace threading added no field and altered no other.
func TestTrace_NonDebugBeat_ResponseShapeUnchanged(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	chain := `[{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"}]`
	h, _ := traceHandler(t, pool, false, move, chain, `SHOULD NOT BE CALLED`)

	code, body := postBeat(h, id.World, id.K, move)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}
	want := map[string]bool{"schema_version": true, "narration": true, "messages": true, "result": true}
	for k := range top {
		if !want[k] {
			t.Fatalf("non-debug response gained an unexpected key %q (nil trace must change nothing):\n%s", k, body)
		}
	}
	for k := range want {
		if _, ok := top[k]; !ok {
			t.Fatalf("non-debug response dropped the pre-existing key %q:\n%s", k, body)
		}
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
	h, resolve := traceHandler(t, pool, true, grip, chain, ruling)

	code, body := postBeat(h, id.World, id.K, grip)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var r traceBeatResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}

	// The attempt actually reached the referee (proving the ruling in the trace is a real capture).
	if resolve.calls != 1 {
		t.Fatalf("resolve calls = %d, want 1 (an AttributeChanged is adjudicated)", resolve.calls)
	}
	if r.ReasoningLog == nil {
		t.Fatalf("debug beat shipped NO reasoning_log\nbody: %s", body)
	}
	if len(r.ReasoningLog.Rulings) != 1 {
		t.Fatalf("reasoning_log.rulings = %+v, want exactly one adjudicated ruling", r.ReasoningLog.Rulings)
	}
	got := r.ReasoningLog.Rulings[0]
	if got.Therefore != "succeeds" {
		t.Fatalf("reasoning_log.rulings[0].therefore = %q, want succeeds", got.Therefore)
	}
	if !strings.Contains(got.Reasoning, "grip tests the bar") {
		t.Fatalf("reasoning_log.rulings[0].reasoning = %q, want the scripted ruling's reasoning", got.Reasoning)
	}
	if got.Outcome != "resolved" {
		t.Fatalf("reasoning_log.rulings[0].outcome = %q, want resolved", got.Outcome)
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
