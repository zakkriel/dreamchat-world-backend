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
// seedTensionGeometry's own worked derivation). **The beat must NOT bounce to turn_budget** — that is
// the ruling, and it is what this test has always guarded.
//
// The journey now runs its legs back to back inside this one beat (2026-08-28: the player does not
// click through the world's dice — runJourneyToCompletion), so a QUIET road ends "journey_arrived"
// here rather than "journey_leg" with an active row waiting for presses. What did not change: the
// world still gets one turn, and one interruption roll, per leg; over-budget is still never a reject.
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
	if outcome.HaltReason == "turn_budget" {
		t.Fatalf("halt_reason = turn_budget — a move that overflows the beat budget must become a journey, never bounce (RULINGS-2026-07-30 §2)")
	}
	if outcome.HaltReason != "journey_arrived" {
		t.Fatalf("halt_reason = %q, want journey_arrived — a quiet road runs its legs to the end in one beat", outcome.HaltReason)
	}

	var status string
	var legsTotal, legsDone int
	// Task 7 note: filtered to status='active' (not just world_id/actor_id) — this playerID/worldID
	// pair is reused across several files' fixtures, and an unfiltered read risks picking up an
	// unrelated leftover row (any prior test's own journey, ended by an unrelated later beat) instead
	// of the one THIS test just created.
	if err := pool.QueryRow(ctx,
		`SELECT status, legs_total, legs_done FROM journey
		   WHERE world_id=$1 AND actor_id=$2 ORDER BY started_tick DESC LIMIT 1`,
		worldID, playerID).Scan(&status, &legsTotal, &legsDone); err != nil {
		t.Fatalf("journey row: %v", err)
	}
	if status == "active" {
		t.Fatalf("journey status = active — the beat must not return with a journey still waiting to be clicked through")
	}
	if legsTotal < 5 || legsTotal > 10 {
		t.Fatalf("legs_total = %d, want in [5,10] (R7's bounded band)", legsTotal)
	}
	// THE WORLD KEPT ITS CHANCES. Every leg ran, which is every interruption roll the founding ruling
	// promises — the clicking went, the danger did not.
	if legsDone != legsTotal {
		t.Fatalf("legs_done = %d, want all %d — the world must get its roll on EVERY leg, not just the first", legsDone, legsTotal)
	}

	// Arrival is the ONLY thing that may move the actor, and it commits through the ordinary path.
	if loc, err := orc.actorLocation(ctx, worldID, playerID); err != nil {
		t.Fatalf("actorLocation: %v", err)
	} else if loc != tenD {
		t.Fatalf("player location = %s, want tenD (%s) — a completed quiet journey arrives", loc, tenD)
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

// SPEC-032. A journey leg that fires the world's turn while the traveller is nowhere known mints a
// waystation, wires it into the road, and STEPS THE TRAVELLER ONTO IT — which is what finally gives
// journey.where_label something to say.
//
// The from→place portal used to be built only when no direct origin→goal connection existed. That
// guard conflated "is there already a way to the goal?" (R4's bar check, answered earlier) with "can
// the traveller stand where they are?" (always yes). Seeding the Harbormaster's Office behind an open
// door from Dock Street made the guard true for the first time, and every eruption on that road began
// minting a waystation wired only to the GOAL, then failing to move the traveller onto it:
// gate_reject, a hard error that killed the beat. It is also why where_label was permanently null.
//
// Runs against the real seeded road, because the defect only appears when origin and goal are already
// connected — a fixture that forgot to connect them would pass against the broken code.
func TestJourney_LegMintsAWaystationTheTravellerCanStandOn(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	const dockStreetID = "210c0000-0000-0000-0000-0000000000d2"
	const officeID = "210c0000-0000-0000-0000-0000000000d5"

	// Precondition the bug depended on: origin and goal are ALREADY directly connected.
	orc := wtOrchestrator(pool)
	exists, permits, err := orc.connectionBetween(ctx, dlWorldID, dockStreetID, officeID)
	if err != nil {
		t.Fatalf("connectionBetween: %v", err)
	}
	if !exists || !permits {
		t.Fatalf("precondition: expected an open door Dock Street↔Office (exists=%v permits=%v)", exists, permits)
	}

	// Stand the traveller on Dock Street and force every turn to fire, so the leg takes the
	// place-authoring path deterministically instead of waiting on the odds.
	var wasAt string
	if err := pool.QueryRow(ctx,
		`SELECT attrs->>'location_id' FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		dlWorldID, dlKadeID).Scan(&wasAt); err != nil {
		t.Fatalf("read traveller's starting location: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE actor_state SET attrs = jsonb_set(attrs,'{location_id}', to_jsonb($1::text))
		  WHERE world_id=$2 AND entity_id=$3`, dockStreetID, dlWorldID, dlKadeID); err != nil {
		t.Fatalf("place the traveller: %v", err)
	}
	// Put him back: this suite shares one fixture world, and a test that walks the traveller onto a
	// waystation and leaves him there changes the scene every later test reads.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE actor_state SET attrs = jsonb_set(attrs,'{location_id}', to_jsonb($1::text))
			  WHERE world_id=$2 AND entity_id=$3`, wasAt, dlWorldID, dlKadeID)
	})
	// Clear this world's fire-log FIRST. Forcing a tier to fire is meaningless if a stale eruption
	// row sits at or beyond the test's tick: fn_pressure_chance derives the chance from elapsed time
	// SINCE the last fire, so a leftover row drives it to zero and the "forced" tier never fires. This
	// suite shares one database and ordinary beats write fire-log rows, so the residue is normal.
	if _, err := pool.Exec(ctx, `DELETE FROM world_eruption WHERE world_id=$1`, dlWorldID); err != nil {
		t.Fatalf("clear fire-log: %v", err)
	}
	// And clear waystations minted by earlier runs. The place author mints by DESCRIPTOR and the
	// engine's reuse-before-create floor matches on it, so a leftover stretch of road from a previous
	// run is silently reused — wired to wherever THAT run came from, not to where this traveller
	// stands. The move onto it then cannot be permitted and the leg proves nothing.
	clearMintedWaystations(t, ctx, pool, dlWorldID)
	wtForceTierFires(t, ctx, pool, dlWorldID, "small")
	// wtOrchestrator leaves PlaceAuthor nil — nothing else in this package reaches the minting path.
	orc.PlaceAuthor = NewFakePlaceAuthorDriver()

	tick := wtBaseTick(t, ctx, pool)
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, Attempt{
		Type: "ActorMoved", Stated: "I walk for the Harbormaster's Office", ToTargetID: officeID,
	}, tick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM journey WHERE journey_id=$1`, j.ID)
	})

	var out BeatOutcome
	if err := orc.runJourneyLeg(ctx, j, &out, nil); err != nil {
		t.Fatalf("runJourneyLeg failed: %v — a leg that mints a waystation must be able to put the traveller on it", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, out.Committed) })

	if j.StageID == "" {
		t.Fatal("no stage_id after a firing leg — nothing was minted, so where_label has nothing to report")
	}

	// The traveller is actually standing on it, not merely near it.
	var loc string
	if err := pool.QueryRow(ctx,
		`SELECT attrs->>'location_id' FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		dlWorldID, dlKadeID).Scan(&loc); err != nil {
		t.Fatalf("read traveller location: %v", err)
	}
	if loc != j.StageID {
		t.Fatalf("traveller is at %s but the minted waystation is %s — they were stranded off the road they walked onto", loc, j.StageID)
	}

	// And the way behind them exists, which is what the move required.
	back, backPermits, err := orc.connectionBetween(ctx, dlWorldID, dockStreetID, j.StageID)
	if err != nil {
		t.Fatalf("connectionBetween(from, waystation): %v", err)
	}
	if !back || !backPermits {
		t.Fatalf("no open way from Dock Street onto the new waystation (exists=%v permits=%v)", back, backPermits)
	}
}

// The interrupting beat still says where you were going. When a beat-cutting eruption stops a
// journey the row ends, so the block goes active:false / status:"ended" — but goal_label and
// where_label are still projected, and the frontend needs them: "restate" is the next step of the
// founder's worked example, and a player cannot restate a destination the screen has forgotten.
//
// Pinned because it is a contract the play surface now depends on and nothing else would catch its
// loss: an interrupted journey is a terminal row, and journeyBlock's own activeJourney lookup
// (status='active') returns nil for it — the labels survive only because the beat stream projects
// the in-memory journey instead. A refactor that dropped that preference would blank the line and
// break no other test.
func TestJourney_InterruptedBlockStillNamesTheDestination(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	const dockStreetID = "210c0000-0000-0000-0000-0000000000d2"
	const officeID = "210c0000-0000-0000-0000-0000000000d5"

	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = NewFakePlaceAuthorDriver()

	var wasAt string
	if err := pool.QueryRow(ctx,
		`SELECT attrs->>'location_id' FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		dlWorldID, dlKadeID).Scan(&wasAt); err != nil {
		t.Fatalf("read traveller location: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE actor_state SET attrs=jsonb_set(attrs,'{location_id}',to_jsonb($1::text))
		  WHERE world_id=$2 AND entity_id=$3`, dockStreetID, dlWorldID, dlKadeID); err != nil {
		t.Fatalf("place the traveller: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE actor_state SET attrs=jsonb_set(attrs,'{location_id}',to_jsonb($1::text))
			  WHERE world_id=$2 AND entity_id=$3`, wasAt, dlWorldID, dlKadeID)
	})

	// medium cuts the beat (eruptionCutsBeat); small never does.
	// Clear this world's fire-log FIRST. Forcing a tier to fire is meaningless if a stale eruption
	// row sits at or beyond the test's tick: fn_pressure_chance derives the chance from elapsed time
	// SINCE the last fire, so a leftover row drives it to zero and the "forced" tier never fires. This
	// suite shares one database and ordinary beats write fire-log rows, so the residue is normal.
	if _, err := pool.Exec(ctx, `DELETE FROM world_eruption WHERE world_id=$1`, dlWorldID); err != nil {
		t.Fatalf("clear fire-log: %v", err)
	}
	// And clear waystations minted by earlier runs. The place author mints by DESCRIPTOR and the
	// engine's reuse-before-create floor matches on it, so a leftover stretch of road from a previous
	// run is silently reused — wired to wherever THAT run came from, not to where this traveller
	// stands. The move onto it then cannot be permitted and the leg proves nothing.
	clearMintedWaystations(t, ctx, pool, dlWorldID)
	wtForceTierFires(t, ctx, pool, dlWorldID, "medium")

	tick := wtBaseTick(t, ctx, pool)
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, Attempt{
		Type: "ActorMoved", Stated: "I walk for the Harbormaster's Office", ToTargetID: officeID}, tick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM journey WHERE journey_id=$1`, j.ID) })

	var out BeatOutcome
	if err := orc.runJourneyLeg(ctx, j, &out, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, out.Committed) })

	if out.HaltReason != "journey_interrupted" {
		t.Fatalf("HaltReason = %q, want journey_interrupted — the leg was not cut, so this proves nothing", out.HaltReason)
	}
	blk, err := orc.projectJourneyBlock(ctx, dlWorldID, dlKadeID, out.Journey)
	if err != nil {
		t.Fatalf("projectJourneyBlock: %v", err)
	}
	if blk.Active || blk.Status != "ended" {
		t.Fatalf("block = %+v, want an ended, inactive journey", blk)
	}
	if blk.GoalLabel == nil || *blk.GoalLabel != "Harbormaster's Office" {
		t.Fatalf("goal_label = %v, want the destination the traveller can restate", blk.GoalLabel)
	}
}

// clearMintedWaystations removes roads authored by earlier journeys in this shared fixture world —
// the minted places and the ways connecting them. Matching on the fake place author's own descriptor
// prefix keeps it surgical: seeded locations and their portals are untouched.
func clearMintedWaystations(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string) {
	t.Helper()
	const minted = fakePlaceAuthorDescriptorPrefix + "%"
	const ways = "the way connecting%"
	for _, tbl := range []string{"location_state", "artifact_state"} {
		if _, err := pool.Exec(ctx,
			`DELETE FROM `+tbl+` WHERE world_id=$1 AND entity_id IN (
			   SELECT entity_id FROM entity_registry
			    WHERE world_id=$1 AND (canonical_name LIKE $2 OR canonical_name LIKE $3))`,
			worldID, minted, ways); err != nil {
			t.Fatalf("clear minted %s: %v", tbl, err)
		}
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM entity_registry WHERE world_id=$1 AND (canonical_name LIKE $2 OR canonical_name LIKE $3)`,
		worldID, minted, ways); err != nil {
		t.Fatalf("clear minted registry rows: %v", err)
	}
}
