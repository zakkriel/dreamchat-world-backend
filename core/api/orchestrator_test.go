package main

import (
	"context"
	"fmt"
	"testing"
)

// Stable test UUIDs for orchestrator tests — not in entity_registry (we insert them).
const (
	locA     = "f1000000-0000-0000-0000-000000000001"
	locB     = "f1000000-0000-0000-0000-000000000002"
	doorID   = "f1000000-0000-0000-0000-000000000003"
	district = "f1000000-0000-0000-0000-000000000004"
)

// seedOrchestratorEntities inserts entity_registry rows for the orchestrator tests, plus the §3 space
// geometry and the seeded walk movement type that the real move-duration arithmetic needs.
// ON CONFLICT DO NOTHING so re-runs are safe.
//
// Station F / §2: fn_move_duration_actor = CEIL(fn_distance(from,to) / effective_speed(actor,'walk')).
// locA and locB are sibling children of a district, 7 m apart → CEIL(7/1.4) = 5 — the same 5 ticks the
// orchestrator has always charged a locA→locB move, now honestly derived from distance/speed. walk=1.4
// must be seeded for this world (fn_effective_speed reads movement_type; the pre-Station-F Mara seed
// never planted it).
func seedOrchestratorEntities(t *testing.T, ctx context.Context) {
	t.Helper()
	pool := testPool(t)
	_, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		VALUES
		  ($1, $2, 'location', 'orch-loc-A'),
		  ($3, $2, 'location', 'orch-loc-B'),
		  ($4, $2, 'artifact', 'orch-door'),
		  ($5, $2, 'location', 'orch-district')
		ON CONFLICT (entity_id) DO NOTHING`,
		locA, worldID, locB, doorID, district)
	if err != nil {
		t.Fatalf("seed orchestrator entities: %v", err)
	}
	// §3 geometry: district (root) with locA {0,0} and locB {7,0} as children → distance 7 m.
	_, err = pool.Exec(ctx, `
		INSERT INTO location_state (entity_id, world_id, attrs)
		VALUES
		  ($4, $1, '{"coordinates":{"x":0,"y":0},"extent":{"w":2000,"h":2000}}'::jsonb),
		  ($2, $1, jsonb_build_object('coordinates', jsonb_build_object('x',0,'y',0), 'parent_location_id', $4::text)),
		  ($3, $1, jsonb_build_object('coordinates', jsonb_build_object('x',7,'y',0), 'parent_location_id', $4::text))
		ON CONFLICT (entity_id) DO NOTHING`,
		worldID, locA, locB, district)
	if err != nil {
		t.Fatalf("seed orchestrator location geometry: %v", err)
	}
	// Seeded default movement type for this world (§2): walk, 1.4 m/s.
	_, err = pool.Exec(ctx, `
		INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps)
		VALUES ($1, 'walk', 1.4)
		ON CONFLICT (world_id, movement_type_id) DO NOTHING`,
		worldID)
	if err != nil {
		t.Fatalf("seed orchestrator walk movement_type: %v", err)
	}
	// Station F / §5.3: the door is a PERMITTING Portal (open ∧ ¬locked) connecting locA↔locB, so a
	// cross-location ActorMoved locA→locB passes the new accessibility floor. Portal is accessibility,
	// NOT geometry — it carries NO coordinates and never feeds fn_distance/fn_move_duration.
	_, err = pool.Exec(ctx, `
		INSERT INTO artifact_state (entity_id, world_id, attrs)
		VALUES ($1, $2, jsonb_build_object('open', true, 'locked', false,
		         'connects', jsonb_build_array($3::text, $4::text)))
		ON CONFLICT (entity_id) DO NOTHING`,
		doorID, worldID, locA, locB)
	if err != nil {
		t.Fatalf("seed orchestrator door portal: %v", err)
	}
}

// TestRunBeatPassthroughAndAdjudicated verifies the full five-stage orchestrator loop:
//   - ActorMoved (passthrough via apply_event): committed + ticks_advanced=5 (CEIL(7m/1.4) derived)
//   - AttributeChanged x2 (adjudicated via FakeResolveDriver): committed
//   - halt_reason="completed", len(Committed)>=3
func TestRunBeatPassthroughAndAdjudicated(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Seed entity_registry rows for locA, locB, doorID.
	seedOrchestratorEntities(t, ctx)

	// Compute base tick: ≥50000 and above any existing max to avoid order collision.
	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// Position player at locA, Mara at locB (player moves to locB in the chain).
	// Both need actor_state rows so the orchestrator can read their locations.
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→locA',$2,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep1 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$3,'actor','instigator' FROM ev1
		),
		sm1 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($5::text),$2,0 FROM ev1
		),
		ev2 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','M→locB',$4,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep2 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$6,'actor','instigator' FROM ev2
		),
		sm2 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$6,'actor','attrs.location_id',to_jsonb($7::text),$4,0 FROM ev2
		)
		SELECT 1`,
		worldID, baseTick, playerID, baseTick+1, locA, maraID, locB)
	if err != nil {
		t.Fatalf("setup positions: %v", err)
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        NewFakeResolveDriver(),
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	// Chain: player at locA moves to locB (ActorMoved → passthrough, ticks+CEIL(7/1.4)=5),
	// then two AttributeChanged on doorID (adjudicated).
	chain := []Attempt{
		{
			Type:       "ActorMoved",
			Stated:     "I move to locB",
			ToTargetID: locB,
		},
		{
			Type:     "AttributeChanged",
			Stated:   "I try the door handle",
			TargetID: doorID,
		},
		{
			Type:     "AttributeChanged",
			Stated:   "I push harder on the door",
			TargetID: doorID,
		},
	}

	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+2)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}

	if len(outcome.Committed) < 3 {
		t.Fatalf("expected >=3 committed events, got %d: %v", len(outcome.Committed), outcome.Committed)
	}
	if outcome.HaltReason != "completed" {
		t.Fatalf("halt_reason = %q, want %q", outcome.HaltReason, "completed")
	}
	if outcome.TicksAdvanced != 5 {
		t.Fatalf("ticks_advanced = %d, want 5 (one locA→locB move: CEIL(7m / 1.4 m/s) = 5)", outcome.TicksAdvanced)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(baseTick))
}

// TestRunBeatPremiseBreakStopsChain verifies that when the player moves away from Mara,
// a subsequent Communicated to Mara fails premise check → halt_reason="premise_broken",
// only the move is committed.
func TestRunBeatPremiseBreakStopsChain(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Seed entity_registry rows (ON CONFLICT DO NOTHING).
	seedOrchestratorEntities(t, ctx)

	// Compute base tick.
	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// Position player at locA, Mara at locA (same location initially).
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→locA-premise',$2,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep1 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$3,'actor','instigator' FROM ev1
		),
		sm1 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($5::text),$2,0 FROM ev1
		),
		ev2 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','M→locA-premise',$4,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep2 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$6,'actor','instigator' FROM ev2
		),
		sm2 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$6,'actor','attrs.location_id',to_jsonb($5::text),$4,0 FROM ev2
		)
		SELECT 1`,
		worldID, baseTick, playerID, baseTick+1, locA, maraID)
	if err != nil {
		t.Fatalf("setup positions: %v", err)
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        NewFakeResolveDriver(),
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	// Chain: player moves from locA to locB (committed), then tries to talk to Mara
	// (still at locA) — premise fails because they're no longer co-located.
	chain := []Attempt{
		{
			Type:       "ActorMoved",
			Stated:     "I move to locB to get away",
			ToTargetID: locB,
		},
		{
			Type:       "Communicated",
			Stated:     "I call out to Mara",
			ListenerID: maraID,
			Content:    "hello from afar",
		},
	}

	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+2)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}

	if len(outcome.Committed) != 1 {
		t.Fatalf("expected exactly 1 committed event (the move), got %d: %v", len(outcome.Committed), outcome.Committed)
	}
	if outcome.HaltReason != "premise_broken" {
		t.Fatalf("halt_reason = %q, want %q", outcome.HaltReason, "premise_broken")
	}

	perceptionSubjectBackfill(t, ctx, pool, int(baseTick))
}

// TestRunBeatNPCCommitDistinctSeq asserts that an NPC commit in Stage 1 and the
// player's first event in Stage 3 receive DISTINCT (in_world_tick, beat_seq) pairs,
// even when they share the same tick. Before Fix 1 both landed at seq=0, causing
// a uq_ce_accepted_order collision.
func TestRunBeatNPCCommitDistinctSeq(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Seed entity_registry rows (ON CONFLICT DO NOTHING).
	seedOrchestratorEntities(t, ctx)

	// Compute base tick.
	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,60000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// Position player at locA, Mara at locA (co-located so NPC Communicated is valid).
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→locA-seq',$2,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep1 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$3,'actor','instigator' FROM ev1
		),
		sm1 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($5::text),$2,0 FROM ev1
		),
		ev2 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','M→locA-seq',$4,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep2 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$6,'actor','instigator' FROM ev2
		),
		sm2 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$6,'actor','attrs.location_id',to_jsonb($5::text),$4,0 FROM ev2
		)
		SELECT 1`,
		worldID, baseTick, playerID, baseTick+1, locA, maraID)
	if err != nil {
		t.Fatalf("setup positions: %v", err)
	}

	// Inline fake cognition driver: Mara says something to the player.
	npcJSON := `[{"actor_id":"` + maraID + `","decision":{"commit_kind":"commit","attempt":{"type":"Communicated","stated":"Mara greets the player","listener_id":"` + playerID + `","content":"greetings traveller"}}}]`
	fakeCog := &inlineCommitCognitionDriver{json: npcJSON}

	// Mara holds a private record about the player (seed E1), so the §5 split flags her into the
	// ISOLATED seat for this player-directed action. Bind the same inline driver to both seats so
	// her one decision is produced by whichever seat she lands in (here: isolated).
	orc := &Orchestrator{
		DB:                pool,
		Resolve:           NewFakeResolveDriver(),
		CognitionBatch:    fakeCog,
		CognitionIsolated: fakeCog,
		WorldActor:        NewFakeWorldActorDriver(),
	}

	// Player's chain: Communicated to Mara (passthrough).
	chain := []Attempt{
		{
			Type:       "Communicated",
			Stated:     "I greet Mara back",
			ListenerID: maraID,
			Content:    "hello back",
		},
	}

	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+2)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if outcome.HaltReason != "completed" {
		t.Fatalf("halt_reason = %q, want %q", outcome.HaltReason, "completed")
	}
	if len(outcome.Committed) != 2 {
		t.Fatalf("expected 2 committed events (NPC + player), got %d: %v", len(outcome.Committed), outcome.Committed)
	}

	// Assert (in_world_tick, beat_seq) pairs are DISTINCT for both committed events.
	type tickSeq struct {
		tick int64
		seq  int
	}
	pairs := make([]tickSeq, 0, 2)
	for _, evID := range outcome.Committed {
		var ts tickSeq
		if err := pool.QueryRow(ctx,
			`SELECT in_world_tick, beat_seq FROM canon_event WHERE event_id=$1::uuid`,
			evID).Scan(&ts.tick, &ts.seq); err != nil {
			t.Fatalf("query canon_event %s: %v", evID, err)
		}
		pairs = append(pairs, ts)
	}
	if pairs[0] == pairs[1] {
		t.Fatalf("seq collision: both events at tick=%d seq=%d — NPC commit must advance curSeq",
			pairs[0].tick, pairs[0].seq)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(baseTick))
}

// TestRunBeatZeroDurationMoveKeepsSeqMonotonic pins the (tick,seq) fix for same-location moves: a
// zero-duration ActorMoved does NOT advance the clock, so curSeq must keep incrementing rather than
// reset to 0 — otherwise the following Communicated collides with the move on (tick,seq) and
// apply_event trips uq_ce_accepted_order. Both events must commit at the SAME tick with DISTINCT seqs.
func TestRunBeatZeroDurationMoveKeepsSeqMonotonic(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,62000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// Position player AND Mara at locA: the Communicated listener must be co-located, and the player
	// must already be at locA so the move to locA is same-location → fn_move_duration_actor(...,locA,locA)=CEIL(0/1.4)=0.
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→locA-zerodur',$2,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep1 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$3,'actor','instigator' FROM ev1
		),
		sm1 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($5::text),$2,0 FROM ev1
		),
		ev2 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','M→locA-zerodur',$4,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep2 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$6,'actor','instigator' FROM ev2
		),
		sm2 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$6,'actor','attrs.location_id',to_jsonb($5::text),$4,0 FROM ev2
		)
		SELECT 1`,
		worldID, baseTick, playerID, baseTick+1, locA, maraID)
	if err != nil {
		t.Fatalf("setup positions: %v", err)
	}

	// Cognition stays quiet so no NPC act interposes a seq between the move and the say.
	orc := &Orchestrator{
		DB:                pool,
		Resolve:           NewFakeResolveDriver(),
		CognitionBatch:    &scriptedCognitionDriver{name: "quiet-batch", body: `[]`},
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver(),
	}

	// [ same-location ActorMoved (dur=0), then Communicated to co-located Mara ].
	chain := []Attempt{
		{Type: "ActorMoved", Stated: "I shift my weight but stay put", ToTargetID: locA},
		{Type: "Communicated", Stated: "I lean in and speak", ListenerID: maraID, Content: "a word"},
	}

	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+2)
	if err != nil {
		t.Fatalf("RunBeat (zero-duration move + say must not collide): %v", err)
	}
	if outcome.HaltReason != "completed" {
		t.Fatalf("halt_reason = %q, want completed", outcome.HaltReason)
	}
	if len(outcome.Committed) != 2 {
		t.Fatalf("expected 2 committed events (move + say), got %d: %v", len(outcome.Committed), outcome.Committed)
	}

	// Both at the SAME tick (dur=0 → clock never advanced), DISTINCT seqs (the fix).
	type tickSeq struct {
		tick int64
		seq  int
	}
	pairs := make([]tickSeq, 0, 2)
	for _, evID := range outcome.Committed {
		var ts tickSeq
		if err := pool.QueryRow(ctx,
			`SELECT in_world_tick, beat_seq FROM canon_event WHERE event_id=$1::uuid`,
			evID).Scan(&ts.tick, &ts.seq); err != nil {
			t.Fatalf("query canon_event %s: %v", evID, err)
		}
		pairs = append(pairs, ts)
	}
	if pairs[0].tick != pairs[1].tick {
		t.Fatalf("zero-duration move must not advance the tick: got ticks %d and %d", pairs[0].tick, pairs[1].tick)
	}
	if pairs[0].seq == pairs[1].seq {
		t.Fatalf("seq collision at tick=%d seq=%d — a zero-duration move must keep curSeq monotonic", pairs[0].tick, pairs[0].seq)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(baseTick))
}

// inlineRulingResolveDriver returns a fixed ruling/2 JSON for CI — used to script a specific ruled
// outcome (here, an EntityCreated) through the real adjudicate → verdict → commit path.
type inlineRulingResolveDriver struct{ json string }

func (f *inlineRulingResolveDriver) Name() string { return "inline-ruling-resolve" }
func (f *inlineRulingResolveDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (f *inlineRulingResolveDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("inline-ruling-resolve: used without schema")
	}
	return f.json, nil
}

// TestRunBeatEntityCreatedRoutesAndCommits — Station F Task 9: a ruling emitting EntityCreated routes
// through adjudicate (EntityCreated is adjudicated, not passthrough — §7), its LLM-minted new id is NOT
// a whitelist violation (the one true-introduction opening, §8), and the commit persists a real
// entity_registry row with provenance (created_by_event). "When it resolves it updates it all" (founder).
func TestRunBeatEntityCreatedRoutesAndCommits(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,64000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// Position the player at locA so the create has a scene to anchor perceptions.
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→locA-create',$2,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep1 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$3,'actor','instigator' FROM ev1
		),
		sm1 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($4::text),$2,0 FROM ev1
		)
		SELECT 1`,
		worldID, baseTick, playerID, locA)
	if err != nil {
		t.Fatalf("setup player position: %v", err)
	}

	// The LLM mints a fresh id for the forged dagger — NOT in entity_registry, NOT in the slice. The
	// verdict must admit it (true introduction + descriptor); the commit must persist it with provenance.
	const newID = "f3000000-0000-0000-0000-000000000abc"
	ruling := fmt.Sprintf(`{"reasoning":"Kade forges a plain iron dagger from the shard — a novice can make a plain blade.","therefore":"succeeds","outcome":{"kind":"resolved","events":[{"type":"EntityCreated","actor_id":%q,"truth":"a plain dagger takes shape under his hammer","target_id":%q,"new_entity_kind":"artifact","canonical_name":"iron dagger","descriptor":"a plain iron dagger (task9 route)","new_attrs":{"location_id":%q,"coordinates":{"x":0,"y":0},"size":2,"weight":1}}]}}`,
		playerID, newID, locA)

	orc := &Orchestrator{
		DB:                pool,
		Resolve:           &inlineRulingResolveDriver{json: ruling},
		CognitionBatch:    &scriptedCognitionDriver{name: "quiet-batch", body: `[]`},
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver(),
	}

	// The player's attempt is an EntityCreated intent → routes to adjudicate (not passthrough).
	chain := []Attempt{{Type: "EntityCreated", Stated: "I forge a dagger from the starmetal shard"}}

	outcome, err := orc.RunBeat(ctx, worldID, playerID, chain, baseTick+2)
	if err != nil {
		t.Fatalf("RunBeat (EntityCreated route): %v", err)
	}
	if outcome.HaltReason != "completed" {
		t.Fatalf("halt_reason = %q, want completed (the create must commit, not bounce/reject)", outcome.HaltReason)
	}
	if len(outcome.Committed) < 1 {
		t.Fatalf("expected >=1 committed event (the EntityCreated), got %d: %v", len(outcome.Committed), outcome.Committed)
	}

	// The new entity is REAL: a provenance-stamped entity_registry row that did not exist before.
	var createdBy string
	if err := pool.QueryRow(ctx,
		`SELECT created_by_event::text FROM entity_registry WHERE world_id=$1 AND entity_id=$2`,
		worldID, newID).Scan(&createdBy); err != nil {
		t.Fatalf("created entity_registry row for %s not found (routing/commit failed): %v", newID, err)
	}
	if createdBy == "" {
		t.Fatalf("created entity has no created_by_event provenance (§8 net 3)")
	}

	perceptionSubjectBackfill(t, ctx, pool, int(baseTick))
}

// inlineCommitCognitionDriver returns a fixed NPC decision JSON for CI.
type inlineCommitCognitionDriver struct{ json string }

func (f *inlineCommitCognitionDriver) Name() string { return "inline-commit-cognition" }
func (f *inlineCommitCognitionDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (f *inlineCommitCognitionDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("inline-commit-cognition: used without schema")
	}
	return f.json, nil
}
