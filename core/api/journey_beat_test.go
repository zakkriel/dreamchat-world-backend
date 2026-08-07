package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Task 6 — over-budget becomes a journey, the dead end dies. RULINGS-2026-07-30 §2: "'You can't even
// try to leave' is dramatically dead — the exact player-centric wall this system refuses." An
// over-budget attempt must BEGIN rather than bounce; the one thing that must still halt is the
// impossible move (speed 0 → fn_move_duration_actor's max-bigint sentinel — arithmetic impossibility,
// not a budget failure, orchestrator.go:420-435). These two tests exercise both gates' shared shape
// directly through runChain's ActorMoved branch, reusing tension_test.go's seedTensionGeometry (tenA
// carries 'tense' → 30 s budget, tenA↔tenB↔tenC↔tenD are 16 m apart, D is 48 m from A — 35 s of walk,
// over a tense budget but not impossible).

// jbEncumberedID is a fresh actor, self-contained to this file, so setting it 'encumbered' cannot
// leak into any other test's use of the shared playerID fixture.
const jbEncumberedID = "f2000000-0000-0000-0000-0000000000e1"

// jbOrchestrator builds a quiet Orchestrator against the tension-geometry world: no adjudication is
// exercised by either test (both stay on the ActorMoved passthrough branch), so the resolve driver is
// never called.
func jbOrchestrator(pool *pgxpool.Pool) *Orchestrator {
	return &Orchestrator{
		DB:                pool,
		Resolve:           NewFakeResolveDriver(),
		CognitionBatch:    &scriptedCognitionDriver{name: "jb-quiet-batch", body: `[]`},
		CognitionIsolated: &scriptedCognitionDriver{name: "jb-quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver(),
	}
}

// TestRunBeat_OverBudgetMoveBecomesAJourney is the founding ruling, end to end: a tense scene (30 s
// budget) and a target far enough that the walk cannot fit (tenA→tenD, 35 s of real physics —
// seedTensionGeometry's own worked derivation). The beat must NOT bounce: HaltReason is "journey_leg"
// (not "turn_budget"), an active journey row exists for the actor, and the actor has NOT teleported to
// the goal — starting a journey commits nothing to canon; only arrival would, and this beat's first
// leg does not reach the threshold (span 35 s, leg 1 of a 5-leg journey covers ceil(35/5) = 7 s).
func TestRunBeat_OverBudgetMoveBecomesAJourney(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with the journey delete below, so it runs BEFORE the pool closes.
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	seedTensionGeometry(t, ctx)
	orc := jbOrchestrator(pool)

	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,95000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	placeActorAt(t, ctx, playerID, tenA, baseTick) // tenA: stamped 'tense' → 30 s budget

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM journey WHERE world_id=$1 AND actor_id=$2 AND status='active'`, worldID, playerID); err != nil {
			t.Errorf("cleanup active journey: %v", err)
		}
	})

	chain := []Attempt{{Type: "ActorMoved", Stated: "I walk the long way to D", ToTargetID: tenD}}
	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+1, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if outcome.HaltReason != "journey_leg" {
		t.Fatalf("halt_reason = %q, want journey_leg — a move that overflows the beat budget must begin a journey, not bounce (RULINGS-2026-07-30 §2)", outcome.HaltReason)
	}
	if len(outcome.Committed) != 0 {
		t.Fatalf("committed = %v, want empty — starting a journey and running one leg commits nothing to canon", outcome.Committed)
	}

	var status string
	var legsTotal, legsDone int
	if err := pool.QueryRow(ctx,
		`SELECT status, legs_total, legs_done FROM journey WHERE world_id=$1 AND actor_id=$2`,
		worldID, playerID).Scan(&status, &legsTotal, &legsDone); err != nil {
		t.Fatalf("active journey row: %v", err)
	}
	if status != "active" {
		t.Fatalf("journey status = %q, want active — the founding ruling gives the world MULTIPLE chances across legs, not one", status)
	}
	if legsDone != 1 {
		t.Fatalf("legs_done = %d, want 1 (exactly one leg ran this beat)", legsDone)
	}
	if legsTotal < 5 || legsTotal > 10 {
		t.Fatalf("legs_total = %d, want in [5,10] (R7's bounded-press band)", legsTotal)
	}

	// The actor has NOT teleported to the goal: only the journey's own arrival commit may move them,
	// and this leg did not reach the threshold.
	if loc, err := orc.actorLocation(ctx, worldID, playerID); err != nil {
		t.Fatalf("actorLocation: %v", err)
	} else if loc != tenA {
		t.Fatalf("player location = %s, want tenA (%s) — starting a journey must not teleport the actor to the goal", loc, tenA)
	}
}

// TestRunBeat_ImpossibleMoveStillHaltsTurnBudget is the one case the Journey does not swallow: an
// encumbered actor (movement_type walk x -100% status_modifier, §2) has effective speed 0, so
// fn_move_duration_actor returns the max-bigint sentinel — "cannot be done at all", not "too long for
// this beat". That must still halt turn_budget, commit nothing, and start no journey — even against a
// nearby, perfectly reachable target (tenA→tenB, only 16 m/12 s under a normal budget) so the halt is
// provably about the actor's speed, not the distance.
func TestRunBeat_ImpossibleMoveStillHaltsTurnBudget(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	seedTensionGeometry(t, ctx)

	if _, err := pool.Exec(ctx,
		`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		 VALUES ($1, $2, 'actor', 'jb-encumbered') ON CONFLICT (entity_id) DO NOTHING`,
		jbEncumberedID, worldID); err != nil {
		t.Fatalf("seed encumbered actor: %v", err)
	}
	// seedTensionGeometry seeds the walk movement_type but not the encumbered status_modifier (that
	// geometry is about distance, not encumbrance) — this test adds the -100% row directly.
	if _, err := pool.Exec(ctx,
		`INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
		 VALUES ($1, 'encumbered', 'move', 'walk', -100) ON CONFLICT DO NOTHING`,
		worldID); err != nil {
		t.Fatalf("seed status_modifier: %v", err)
	}

	orc := jbOrchestrator(pool)

	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,96000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	placeActorAt(t, ctx, jbEncumberedID, tenA, baseTick)
	if _, err := pool.Exec(ctx,
		`UPDATE actor_state SET attrs = jsonb_set(COALESCE(attrs,'{}'::jsonb), '{statuses}', '["encumbered"]'::jsonb)
		 WHERE world_id=$1 AND entity_id=$2`, worldID, jbEncumberedID); err != nil {
		t.Fatalf("set encumbered status: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM journey WHERE world_id=$1 AND actor_id=$2`, worldID, jbEncumberedID); err != nil {
			t.Errorf("cleanup journey: %v", err)
		}
	})

	chain := []Attempt{{Type: "ActorMoved", Stated: "I try to walk to B", ToTargetID: tenB}}
	outcome, err := orc.RunBeat(ctx, worldID, jbEncumberedID, chain, baseTick+1, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if outcome.HaltReason != "turn_budget" {
		t.Fatalf("halt_reason = %q, want turn_budget — speed 0 is an arithmetic impossibility, not an over-budget duration the Journey can carry", outcome.HaltReason)
	}
	if outcome.TicksAdvanced != 0 {
		t.Fatalf("ticks_advanced = %d, want 0 (the halt fires before curTick ever advances)", outcome.TicksAdvanced)
	}
	if len(outcome.Committed) != 0 {
		t.Fatalf("committed = %v, want empty — the impossible move must not commit", outcome.Committed)
	}

	var journeyCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM journey WHERE world_id=$1 AND actor_id=$2`, worldID, jbEncumberedID).Scan(&journeyCount); err != nil {
		t.Fatalf("journey count: %v", err)
	}
	if journeyCount != 0 {
		t.Fatalf("journey rows = %d, want 0 — the impossible move must not start a journey", journeyCount)
	}
}
