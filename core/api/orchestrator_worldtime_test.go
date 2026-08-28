package main

import (
	"context"
	"fmt"
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

// TestRunBeat_NonMoveOverBudgetHaltsTurnBudget is Task 6's fallout: a 'long' (300 s) Communicated
// no longer fits a 'tense' (30 s budget) scene, but over-budget is no longer a dead end (design
// §4.7) — it starts a journey and runs its first leg. There is no explicit `sustain` here, so
// startJourney's class-only case fires: kind="wait", span = the SAME 300 s duration_class lookup
// that decided the thing didn't fit. dlWorldID's fn_journey_legs(300) is 5 (the built-in ≤1h
// fallback — no per-world override seeded), so the first leg is ceil(300/5) = 60 s; the world's
// turn is disabled so the leg's own outcome is deterministic (no incidental eruption competing
// with the assertion this test exists to make — mirrors wtDisableWorldActor's documented use).
// TicksAdvanced must be 60, not 0: the beat no longer bounces before the clock moves.
func TestRunBeat_NonMoveOverBudgetHaltsTurnBudget(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with wtDisableWorldActor's own restore and the journey delete
	// below, so both run BEFORE the pool closes (mirrors TestRunBeat_EmptyBeatAdvancesByInstantFloor's
	// documented t.Cleanup-ordering pattern).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	wtSetTavernTension(t, ctx, pool, "tense")
	wtDisableWorldActor(t, ctx, pool, dlWorldID)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM journey WHERE world_id=$1 AND actor_id=$2 AND status='active'`, dlWorldID, dlKadeID); err != nil {
			t.Errorf("cleanup active journey: %v", err)
		}
	})

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	chain := []Attempt{{Type: "Communicated", Stated: "I tell Mara my whole life story", ListenerID: wtMaraID, Content: "...", DurationClass: "long"}}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason == "turn_budget" {
		t.Fatalf("HaltReason = turn_budget — a 300s non-move that does not fit a tense scene's 30s budget must become a vigil, never bounce")
	}
	if out.HaltReason != "journey_arrived" {
		t.Fatalf("HaltReason = %q, want journey_arrived — the vigil runs its legs back to back inside the beat (2026-08-28) and completes on a quiet scene", out.HaltReason)
	}
	// The WHOLE vigil now runs inside the beat, so the clock advances by its full span rather than by
	// one leg's slice (2026-08-28, runJourneyToCompletion). The leg structure is unchanged underneath
	// — five legs, five world's turns, five interruption rolls — only the clicking is gone.
	if out.TicksAdvanced != 300 {
		t.Fatalf("TicksAdvanced = %d, want 300 (the vigil's whole span, run leg by leg inside the beat)", out.TicksAdvanced)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("Committed = %v, want empty — starting a journey and running a leg commits nothing to canon (only arrival does, and this leg didn't reach the threshold)", out.Committed)
	}
	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM journey WHERE world_id=$1 AND actor_id=$2`, dlWorldID, dlKadeID).Scan(&status); err != nil {
		t.Fatalf("journey row: %v", err)
	}
	if status == "active" {
		t.Fatalf("journey status = active — the vigil must not be left waiting to be clicked through")
	}
}

// TestRunBeat_QueryOnlyBeatAdvancesByInstantFloor is the Step-5 floor: a beat with no
// clock-advancing attempt still costs the instant floor (2 s, the seeded fallback), not 0 — stillness
// ticks too. This only applies on the completed path (asserted here); a halted beat keeps its own halt
// semantics untouched.
//
// It is driven by a QUERY-only chain. It used to be driven by an EMPTY chain, which was possible when
// an empty chain and the continue press were the same value — SPEC-037, fixed 2026-08-28. An empty
// chain now means "the parse produced nothing" and deliberately costs no time, so it can no longer
// reach the floor. QUERY is the honest remaining route: a question is a real element that never ticks
// the clock itself (RULINGS-2026-07-23 §3).
//
// This test therefore does DOUBLE duty and must not be deleted for being redundant: it guards the
// floor, AND it proves the bounce branch is narrow — if bounce were ever widened from "nothing parsed"
// to "nothing committed", a questions-only beat would stop costing its moment and this goes red.
//
// Deferral C (Task 3): the floor crossing now runs its own world's turn, which would otherwise roll
// the pressure tiers for dlWorldID on every run of this test. This test is about the TICK COUNT, not
// about rolls, so pressure is disabled the same way TestRunBeat_FloorWindowStillRunsTheWorldsTurn
// disables it — an undisabled roll here could fire and leave a world_eruption row that breaks
// pressure_test.go's TestRollTier_FiredMatchesRollLessThanChance (hardcodes lastEruption=0).
func TestRunBeat_QueryOnlyBeatAdvancesByInstantFloor(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with wtDisableWorldActor's own restore Cleanup below, so the
	// restore runs BEFORE the pool closes (mirrors TestRunBeat_NonMoveCostsWorldTime's documented
	// t.Cleanup-ordering pattern).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	wtDisableWorldActor(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	queryOnly := []Attempt{{Type: "QUERY", Stated: "who is here?", QueryTargetIDs: []string{wtMaraID}}}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, queryOnly, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed", out.HaltReason)
	}
	if out.TicksAdvanced != 2 {
		t.Fatalf("TicksAdvanced = %d, want 2 (the instant floor — asking is not acting, but it still costs a moment)", out.TicksAdvanced)
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
// 'long' (300s) AttributeChanged still cannot fit — but Task 6 means that no longer bounces. The
// chain hands it to the journey WITHOUT calling adjudicate at all (driver.callCount stays 0 — the
// budget gate fires strictly before the referee is consulted, mirroring the pre-commit gate on
// every other branch of this loop) and still commits nothing to canon (only arrival ever would,
// and a class-only wait never arrives — it only ends). worldID's fn_journey_legs(300) is 5 (no
// per-world override seeded here either), so the first leg is ceil(300/5) = 60 s; this fixture
// world carries no pressure config, so the leg is quiet without needing wtDisableWorldActor.
func TestRunBeat_AdjudicatedNonMoveOverBudgetHaltsTurnBudget(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with the journey delete below, so it runs BEFORE the pool closes.
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	seedTensionGeometry(t, ctx)

	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,93000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	placeActorAt(t, ctx, playerID, tenA, baseTick) // tenA: stamped 'tense' → 30s budget

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM journey WHERE world_id=$1 AND actor_id=$2 AND status='active'`, worldID, playerID); err != nil {
			t.Errorf("cleanup active journey: %v", err)
		}
	})

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
	if outcome.HaltReason == "turn_budget" {
		t.Fatalf("halt_reason = turn_budget — a 300s adjudicated non-move must become a vigil, never bounce")
	}
	if outcome.HaltReason != "journey_arrived" {
		t.Fatalf("halt_reason = %q, want journey_arrived — the vigil runs to completion inside the beat (2026-08-28)", outcome.HaltReason)
	}
	if len(outcome.Committed) != 0 {
		t.Fatalf("committed = %v, want empty — starting a journey and running a leg commits nothing to canon", outcome.Committed)
	}
	if outcome.TicksAdvanced != 300 {
		t.Fatalf("ticks_advanced = %d, want 300 (the vigil's whole span — it runs to completion inside the beat)", outcome.TicksAdvanced)
	}
	if driver.callCount != 0 {
		t.Fatalf("resolve driver called %d times, want 0 — the budget gate must fire BEFORE adjudicate() ever consults the referee", driver.callCount)
	}
}

// Deferral C: the instant floor is a real clock crossing, so a pending event due inside it must fire.
// Before the fix, an empty beat advanced world-time by the floor with no world's turn at all, and the
// row sat pending forever.
func TestRunBeat_FloorWindowStillRunsTheWorldsTurn(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	// Pressure off: this test is about the LEDGER firing inside the floor window, not about rolls.
	wtDisableWorldActor(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	// A pending event due at baseTick+1 — inside the instant floor window (startTick, startTick+2].
	stated := fmt.Sprintf("floor-window ledger probe @%d", baseTick)
	payload := fmt.Sprintf(
		`{"actor_id":%q,"attempt":{"type":"Communicated","stated":%q,"listener_id":%q,"content":"probe"}}`,
		wtMaraID, stated, dlKadeID)
	var pendingID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO pending_event (pending_id, world_id, fire_at_tick, magnitude, payload, status)
		 VALUES (gen_random_uuid(), $1, $2, 'small', $3::jsonb, 'pending') RETURNING pending_id`,
		dlWorldID, baseTick+1, payload).Scan(&pendingID); err != nil {
		t.Fatalf("insert pending_event: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM pending_event WHERE pending_id=$1`, pendingID); err != nil {
			t.Errorf("cleanup pending_event: %v", err)
		}
	})

	// A QUERY-only chain: nothing advances the clock except the floor. (Was an empty chain until
	// SPEC-037 landed 2026-08-28 — an empty chain now costs no time at all and never reaches the floor.)
	queryOnly := []Attempt{{Type: "QUERY", Stated: "who is here?", QueryTargetIDs: []string{wtMaraID}}}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, queryOnly, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed", out.HaltReason)
	}

	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM pending_event WHERE pending_id=$1`, pendingID).Scan(&status); err != nil {
		t.Fatalf("read pending status: %v", err)
	}
	if status != "fired" {
		t.Fatalf("pending_event status = %q, want fired — the floor window skipped its world's turn", status)
	}
}
