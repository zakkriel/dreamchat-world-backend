package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Living World / Task 3 — every beat costs world-time. Before this task runChain advanced curTick
// only for ActorMoved (fn_move_duration_actor); every non-move step advanced curSeq only, costing
// ZERO world-time. These tests exercise the real seeded Drowned Lantern play world (make reset
// precedes go test in the battery — matches beathandler_test.go's dlWorldID/dlKadeID pattern) so the
// duration comes from a REAL fn_duration_class_seconds lookup, not a fake.

// Fixed play-seed uuids (seed_drowned_lantern.sql). dlWorldID/dlKadeID already exist in
// beathandler_test.go; wtMaraID/wtTavernID are new here (the `wt` prefix keeps them clear of the
// world-1111 fixture constants and the `dl` prefix already in use).
const (
	wtMaraID   = "2ac70000-0000-0000-0000-0000000000a2" // Mara — co-present listener in the tavern
	wtTavernID = "210c0000-0000-0000-0000-0000000000d1" // The Drowned Lantern — Kade's and Mara's location
)

// wtBaseTick returns a tick above any existing max for the play world, so the test is re-runnable
// without `make reset` between invocations (mirrors nextBaseTick/reactionBaseTick/telegraphBaseTick's
// established per-file pattern).
func wtBaseTick(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int64 {
	t.Helper()
	var tick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		dlWorldID).Scan(&tick); err != nil {
		t.Fatalf("wtBaseTick: %v", err)
	}
	return tick
}

// wtSetTavernTension overrides the Drowned Lantern's CURRENT tension (seeded 'tense', a 30 s beat
// budget — see seed_drowned_lantern.sql's f9 state_mutation) directly on location_state, the same
// direct-projection-table approach orchestrator_test.go's seedOrchestratorEntities already uses for
// test fixtures. No other Go test in this package depends on the play world's real tension value
// (TestPayload_PerceivedCandidates_DrownedLantern only reads candidates), so this is safe within the
// package's test run; each test here sets its OWN required value rather than relying on order.
func wtSetTavernTension(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tension string) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`UPDATE location_state SET attrs = jsonb_set(attrs, '{tension}', to_jsonb($1::text)) WHERE entity_id=$2 AND world_id=$3`,
		tension, wtTavernID, dlWorldID); err != nil {
		t.Fatalf("wtSetTavernTension(%s): %v", tension, err)
	}
}

// wtOrchestrator builds an Orchestrator with fakes wired for every seat (world-first cognition may
// fire — Kade always sits in his own actionIDs, and the hooded woman's private record is about Kade,
// so she is isolated on every beat he makes in this scene; wiring both batch and isolated fakes keeps
// the test independent of exactly who lands in which bucket).
func wtOrchestrator(pool *pgxpool.Pool) *Orchestrator {
	return &Orchestrator{
		DB:                pool,
		Resolve:           NewFakeResolveDriver(),
		CognitionBatch:    NewFakeCognitionDriver(),
		CognitionIsolated: NewFakeCognitionDriver(),
		WorldActor:        NewFakeWorldActorDriver(),
	}
}

// TestRunBeat_NonMoveCostsWorldTime is the brief's happy path: a single non-move Communicated with
// duration_class "long" must advance TicksAdvanced by the seeded long seconds (300), not 0. The
// Drowned Lantern's seeded tension is 'tense' (30 s budget), which would turn_budget-halt a 300 s
// non-move before it ever committed — so this test first widens the scene to 'none' (unbounded)
// tension, the only way the 300 s assertion can hold (task-3-brief ambiguity resolution #3).
func TestRunBeat_NonMoveCostsWorldTime(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with the world_eruption cleanup below, so the delete runs BEFORE the
	// pool closes (ledger_test.go's documented t.Cleanup-ordering pattern).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	wtSetTavernTension(t, ctx, pool, "none")

	// Living World / Task 9 review (Important #2) + whole-branch review (Fix 1/Fix 4): this is the ONLY
	// dlWorldID RunBeat call in this file that actually reaches a committed, clock-advancing Stage-4 (the
	// other two either halt turn_budget before Stage 4 or run an empty chain) — so it is the one place
	// the world's-turn composer runs against dlWorldID's DEFAULT (unforced) pressure config. Once Fix 1
	// reversed the roll's scan order to large→medium→small (so the biggest fired tier wins instead of
	// small silently masking it), a medium/large roll is no longer masked by small's much higher climb
	// rate — at a high enough tick (a function of every canon_event already committed by earlier tests
	// across `go test` invocations without an intervening `make reset`) this call could incidentally roll
	// a REAL medium/large eruption, halting the beat and breaking the "completed"/300-tick assertions
	// below. wtDisableWorldActor removes the world's turn from this test entirely (it only means to
	// exercise the beat's own non-move clock cost, not pressure), so no roll — of any tier, in any scan
	// order — can ever fire here.
	wtDisableWorldActor(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	chain := []Attempt{{Type: "Communicated", Stated: "I tell Mara my whole life story", ListenerID: wtMaraID, Content: "...", DurationClass: "long"}}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	// Belt-and-suspenders: the world actor is disabled above, so this should be a no-op — kept in case
	// anything ever committed to world_eruption before the disable took effect (mirrors worldturn_test.go's
	// own wtDeleteEruptionRows, same package, Task 9's forced-fire tests) — world_eruption is append-only
	// in production, and a leftover row here would break pressure_test.go's
	// TestRollTier_FiredMatchesRollLessThanChance (hardcodes lastEruption=0 for dlWorldID/small).
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, out.Committed) })
	if out.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed", out.HaltReason)
	}
	if out.TicksAdvanced != 300 {
		t.Fatalf("non-move beat advanced %d, want 300 (the seeded 'long' duration_class seconds)", out.TicksAdvanced)
	}
}

// TestRunBeat_NonMoveOverBudgetHaltsTurnBudget is the interim over-budget behavior this task adds
// for non-moves: in a 'tense' scene (30 s budget), a 'long' (300 s) Communicated cannot fit, so the
// chain halts turn_budget WITHOUT committing (mirrors the move branch's existing "prefix stands"
// shape — no Journey/accumulation logic is built here, task-3-brief ambiguity resolution #1).
// TicksAdvanced must be 0: the halt fires before curTick ever advances.
func TestRunBeat_NonMoveOverBudgetHaltsTurnBudget(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	wtSetTavernTension(t, ctx, pool, "tense")

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	chain := []Attempt{{Type: "Communicated", Stated: "I tell Mara my whole life story", ListenerID: wtMaraID, Content: "...", DurationClass: "long"}}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "turn_budget" {
		t.Fatalf("HaltReason = %q, want turn_budget (a 300s non-move cannot fit a tense scene's 30s budget)", out.HaltReason)
	}
	if out.TicksAdvanced != 0 {
		t.Fatalf("TicksAdvanced = %d, want 0 (the over-budget non-move does not commit, so the clock never advances)", out.TicksAdvanced)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("Committed = %v, want empty — the over-budget non-move must NOT commit (mirrors the move branch)", out.Committed)
	}
}

// TestRunBeat_EmptyBeatAdvancesByInstantFloor is the Step-5 floor: a beat with no clock-advancing
// attempt at all (an empty "I watch" beat — nothing in the chain, so the loop body never runs) still
// costs the instant floor (2 s, the seeded fallback), not 0 — stillness ticks too. This only applies
// on the completed path (asserted here); a halted beat keeps its own halt semantics untouched.
func TestRunBeat_EmptyBeatAdvancesByInstantFloor(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, []Attempt{}, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed", out.HaltReason)
	}
	if out.TicksAdvanced != 2 {
		t.Fatalf("TicksAdvanced = %d, want 2 (the instant floor — an empty beat still costs stillness)", out.TicksAdvanced)
	}
}

// ── Review fix: the adjudicated default: branch must charge duration_class too ──────────────────────
//
// The passthrough branch above (Communicated/ObjectRelocated) was the only place attempt.DurationClass
// was ever read; the adjudicated default: branch (AttributeChanged/OwnershipAccessChanged/
// EntityCreated/EntityDestroyed) only did curSeq += ar.SeqAdvance, so a chain of adjudicated non-moves
// cost ZERO world-time regardless of duration_class — breaking "beat world-time = sum of per-attempt
// durations" for a whole attempt category. These two tests use the world-1111 fixture + the
// inlineRulingDriver/validRulingJSON adjudicated-path setup from orchestrator_ruled_test.go (a fake
// resolver that returns a committing AttributeChanged ruling), and tension_test.go's seedTensionGeometry
// (tenA stamped 'tense' → 30s budget; tenD unstamped → 'none' → unbounded) so no new fixtures are needed.

// TestRunBeat_AdjudicatedNonMoveCostsWorldTime is the adjudicated-path mirror of
// TestRunBeat_NonMoveCostsWorldTime: a single adjudicated AttributeChanged{duration_class:"long"}
// advances TicksAdvanced by 300 (the class duration), NOT the 2s instant floor. Starts at tenD
// (unstamped → unbounded budget) so the 300s duration always fits.
func TestRunBeat_AdjudicatedNonMoveCostsWorldTime(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedTensionGeometry(t, ctx)

	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,92000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	placeActorAt(t, ctx, playerID, tenD, baseTick) // tenD: unstamped → 'none' (unbounded budget)

	driver := &inlineRulingDriver{
		name:   "adj-worldtime-long",
		ruling: validRulingJSON(playerID, playerID, "Kade steels himself and begins the long telling", "he takes a slow breath and starts"),
	}
	orc := &Orchestrator{
		DB:                pool,
		Resolve:           driver,
		CognitionBatch:    &scriptedCognitionDriver{name: "quiet-batch", body: `[]`},
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver(),
	}

	chain := []Attempt{{Type: "AttributeChanged", Stated: "I brace myself and begin the long telling", TargetID: playerID, DurationClass: "long"}}
	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+1, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if outcome.HaltReason != "completed" {
		t.Fatalf("halt_reason = %q, want completed", outcome.HaltReason)
	}
	if len(outcome.Committed) != 1 {
		t.Fatalf("committed = %v, want exactly 1", outcome.Committed)
	}
	if outcome.TicksAdvanced != 300 {
		t.Fatalf("ticks_advanced = %d, want 300 (the adjudicated AttributeChanged's 'long' duration_class — NOT the 2s instant floor)", outcome.TicksAdvanced)
	}
}

// TestRunBeat_AdjudicatedNonMoveOverBudgetHaltsTurnBudget is the adjudicated-path mirror of
// TestRunBeat_NonMoveOverBudgetHaltsTurnBudget: starting at tenA ('tense' → 30s budget), the same
// 'long' (300s) AttributeChanged cannot fit, so the chain halts turn_budget WITHOUT calling adjudicate
// at all (driver.callCount stays 0 — the budget gate fires strictly before the referee is consulted,
// mirroring the pre-commit gate on every other branch of this loop) and commits nothing.
func TestRunBeat_AdjudicatedNonMoveOverBudgetHaltsTurnBudget(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedTensionGeometry(t, ctx)

	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,93000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	placeActorAt(t, ctx, playerID, tenA, baseTick) // tenA: stamped 'tense' → 30s budget

	driver := &inlineRulingDriver{
		name:   "adj-worldtime-overbudget",
		ruling: validRulingJSON(playerID, playerID, "Kade steels himself and begins the long telling", "he takes a slow breath and starts"),
	}
	orc := &Orchestrator{
		DB:                pool,
		Resolve:           driver,
		CognitionBatch:    &scriptedCognitionDriver{name: "quiet-batch", body: `[]`},
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver(),
	}

	chain := []Attempt{{Type: "AttributeChanged", Stated: "I brace myself and begin the long telling", TargetID: playerID, DurationClass: "long"}}
	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+1, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if outcome.HaltReason != "turn_budget" {
		t.Fatalf("halt_reason = %q, want turn_budget (a 300s adjudicated non-move cannot fit a tense scene's 30s budget)", outcome.HaltReason)
	}
	if len(outcome.Committed) != 0 {
		t.Fatalf("committed = %v, want empty — the over-budget adjudicated non-move must NOT commit", outcome.Committed)
	}
	if outcome.TicksAdvanced != 0 {
		t.Fatalf("ticks_advanced = %d, want 0 (the halt fires before curTick ever advances)", outcome.TicksAdvanced)
	}
	if driver.callCount != 0 {
		t.Fatalf("resolve driver called %d times, want 0 — the budget gate must fire BEFORE adjudicate() ever consults the referee", driver.callCount)
	}
}
