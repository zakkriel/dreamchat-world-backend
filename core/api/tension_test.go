package main

import (
	"context"
	"math"
	"testing"
)

// TestTensionBudgetSeconds is the §6 enum table, pinned as data: the five tension values map to their
// budgets in seconds (1 tick = 1 second), and any out-of-enum value falls back to `none` (∞) — the
// defensive default that mirrors loadScene's "missing location_state → none".
func TestTensionBudgetSeconds(t *testing.T) {
	cases := []struct {
		tension string
		want    int64
	}{
		{"frantic", 5},
		{"tense", 30},
		{"normal", 60},
		{"calm", 600},
		{"none", math.MaxInt64},
		{"", math.MaxInt64},            // unknown (empty) → none
		{"apocalyptic", math.MaxInt64}, // unknown (non-enum) → none
	}
	for _, c := range cases {
		if got := tensionBudgetSeconds(c.tension); got != c.want {
			t.Errorf("tensionBudgetSeconds(%q) = %d, want %d", c.tension, got, c.want)
		}
	}
}

// Stable UUIDs for the tension-budget geometry — distinct from the orch locA/locB set so the two
// suites never share a scene. A district with four children in a line, 16 m apart.
const (
	tenDistrict = "f2000000-0000-0000-0000-000000000000"
	tenA        = "f2000000-0000-0000-0000-000000000001"
	tenB        = "f2000000-0000-0000-0000-000000000002"
	tenC        = "f2000000-0000-0000-0000-000000000003"
	tenD        = "f2000000-0000-0000-0000-000000000004"
)

// seedTensionGeometry seeds a district with four children A,B,C,D in a line at x = 0,16,32,48
// (§3 nested coordinates), plus the seeded walk=1.4 m/s movement type. tenA carries tension 'tense'
// (30 s budget); the rest carry no tension (→ 'none', ∞). ON CONFLICT DO NOTHING so re-runs are safe.
//
// Derivation (§2 fn_move_duration_actor = CEIL(fn_distance / effective_speed(walk=1.4))):
//
//	A→B = B→C = C→D = 16 m → CEIL(16 / 1.4) = CEIL(11.43) = 12 s each   (the §6 example's 12 s move)
//	D→A                    = 48 m → CEIL(48 / 1.4) = CEIL(34.29) = 35 s  (one long move, > a tense beat)
func seedTensionGeometry(t *testing.T, ctx context.Context) {
	t.Helper()
	pool := testPool(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		VALUES
		  ($1, $6, 'location', 'ten-district'),
		  ($2, $6, 'location', 'ten-A'),
		  ($3, $6, 'location', 'ten-B'),
		  ($4, $6, 'location', 'ten-C'),
		  ($5, $6, 'location', 'ten-D')
		ON CONFLICT (entity_id) DO NOTHING`,
		tenDistrict, tenA, tenB, tenC, tenD, worldID); err != nil {
		t.Fatalf("seed tension entities: %v", err)
	}
	// Root district + four sibling children. tenA is stamped 'tense' (30 s); B/C/D unstamped → none.
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_state (entity_id, world_id, attrs)
		VALUES
		  ($1, $6, '{"coordinates":{"x":0,"y":0},"extent":{"w":2000,"h":2000}}'::jsonb),
		  ($2, $6, jsonb_build_object('coordinates', jsonb_build_object('x',0,'y',0),  'parent_location_id', $1::text, 'tension', 'tense')),
		  ($3, $6, jsonb_build_object('coordinates', jsonb_build_object('x',16,'y',0), 'parent_location_id', $1::text)),
		  ($4, $6, jsonb_build_object('coordinates', jsonb_build_object('x',32,'y',0), 'parent_location_id', $1::text)),
		  ($5, $6, jsonb_build_object('coordinates', jsonb_build_object('x',48,'y',0), 'parent_location_id', $1::text))
		ON CONFLICT (entity_id) DO NOTHING`,
		tenDistrict, tenA, tenB, tenC, tenD, worldID); err != nil {
		t.Fatalf("seed tension geometry: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps)
		VALUES ($1, 'walk', 1.4)
		ON CONFLICT (world_id, movement_type_id) DO NOTHING`,
		worldID); err != nil {
		t.Fatalf("seed tension walk movement_type: %v", err)
	}
}

// placeActorAt positions actorID at locationID by committing a 'move' canon_event + state_mutation
// at (tick,0); the state_mutation trigger projects it into actor_state so actorLocation reads it.
// Returns nothing — it just moves the projection. Used to set a beat's START location.
func placeActorAt(t *testing.T, ctx context.Context, actorID, locationID string, tick int64) {
	t.Helper()
	pool := testPool(t)
	if _, err := pool.Exec(ctx, `
		WITH ev AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','seed-place',$3,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$2,'actor','instigator' FROM ev
		)
		INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		SELECT $1,event_id,$2,'actor','attrs.location_id',to_jsonb($4::text),$3,0 FROM ev`,
		worldID, actorID, tick, locationID); err != nil {
		t.Fatalf("place actor at %s: %v", locationID, err)
	}
}

// TestRunBeat_TensionBudgetCumulative is the §6 cumulative-budget contract under the DB harness.
//
// A `tense` scene (30 s) and a chain of three 16 m moves (12 s each, derived above):
//
//	move 1  A→B  12 s   remaining 30−12 = 18   → commits
//	move 2  B→C  12 s   remaining 18−12 =  6   → commits
//	move 3  C→D  12 s   12 > 6                 → REJECT: chain halts turn_budget, this move does NOT commit
//
// The prefix stands — exactly two ActorMoved rows land, the player ends at C. Then a `none` scene
// (∞) runs one long move (D→A, 35 s > any tense beat) and it commits: `none` never budget-blocks (§5/§6).
func TestRunBeat_TensionBudgetCumulative(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedTensionGeometry(t, ctx)

	orc := &Orchestrator{
		DB:                pool,
		Resolve:           NewFakeResolveDriver(),
		CognitionBatch:    &scriptedCognitionDriver{name: "quiet-batch", body: `[]`},
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver(),
	}

	// ── Cumulative halt under `tense` ────────────────────────────────────────────────────────────
	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,70000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	placeActorAt(t, ctx, playerID, tenA, baseTick) // start the beat at tenA (tension 'tense' → 30 s)

	chain := []Attempt{
		{Type: "ActorMoved", Stated: "I cross to B", ToLocationID: tenB},
		{Type: "ActorMoved", Stated: "I cross to C", ToLocationID: tenC},
		{Type: "ActorMoved", Stated: "I cross to D", ToLocationID: tenD},
	}

	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+1)
	if err != nil {
		t.Fatalf("RunBeat (cumulative): %v", err)
	}
	if outcome.HaltReason != "turn_budget" {
		t.Fatalf("halt_reason = %q, want %q (third 12 s move overflows the 30 s tense budget)", outcome.HaltReason, "turn_budget")
	}
	if len(outcome.Committed) != 2 {
		t.Fatalf("committed = %d (%v), want exactly 2 (12+12=24 ≤ 30; the 36 s move rejects, prefix stands)",
			len(outcome.Committed), outcome.Committed)
	}
	if outcome.TicksAdvanced != 24 {
		t.Fatalf("ticks_advanced = %d, want 24 (two 12 s moves committed before the over-budget halt)", outcome.TicksAdvanced)
	}
	// The prefix stood: the player advanced A→B→C, and the rejected C→D never committed.
	if loc, err := orc.actorLocation(ctx, worldID, playerID); err != nil {
		t.Fatalf("actorLocation after cumulative beat: %v", err)
	} else if loc != tenC {
		t.Fatalf("player location = %s, want tenC (%s) — the over-budget move must not commit", loc, tenC)
	}
	// Exactly two committed ActorMoved canon rows for the player in this beat's tick span. The
	// orchestrator commits passthrough moves with p_legacy_types=false, so event_type is 'ActorMoved'
	// (the seed's placeActorAt writes the legacy 'move' label at baseTick, outside this window).
	var moveRows int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM canon_event ce
		JOIN event_participant ep ON ep.event_id = ce.event_id AND ep.entity_id = $2::uuid
		WHERE ce.world_id = $1 AND ce.event_type = 'ActorMoved' AND ce.in_world_tick >= $3 AND ce.in_world_tick < $3 + 30`,
		worldID, playerID, baseTick+1).Scan(&moveRows); err != nil {
		t.Fatalf("count player ActorMoved rows: %v", err)
	}
	if moveRows != 2 {
		t.Fatalf("player ActorMoved canon rows in beat = %d, want exactly 2", moveRows)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(baseTick))

	// ── `none` never budget-blocks ───────────────────────────────────────────────────────────────
	var noneTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,80000)`,
		worldID).Scan(&noneTick); err != nil {
		t.Fatalf("none base tick: %v", err)
	}
	placeActorAt(t, ctx, playerID, tenD, noneTick) // start the beat at tenD (no tension stamp → 'none' → ∞)

	noneChain := []Attempt{
		{Type: "ActorMoved", Stated: "I make the long crossing to A", ToLocationID: tenA},
	}
	noneOutcome, err := orc.RunBeat(ctx, worldID, playerID, noneChain, noneTick+1)
	if err != nil {
		t.Fatalf("RunBeat (none): %v", err)
	}
	if noneOutcome.HaltReason != "completed" {
		t.Fatalf("none halt_reason = %q, want %q — a 35 s move must commit under an unbounded budget", noneOutcome.HaltReason, "completed")
	}
	if len(noneOutcome.Committed) != 1 {
		t.Fatalf("none committed = %d (%v), want 1 (the long move commits — none never budget-blocks)",
			len(noneOutcome.Committed), noneOutcome.Committed)
	}
	if noneOutcome.TicksAdvanced != 35 {
		t.Fatalf("none ticks_advanced = %d, want 35 (CEIL(48 m / 1.4 m/s))", noneOutcome.TicksAdvanced)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(noneTick))
}
