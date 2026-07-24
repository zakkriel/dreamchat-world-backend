package main

import (
	"context"
	"fmt"
	"testing"
)

// Stable test UUIDs for orchestrator tests — not in entity_registry (we insert them).
const (
	locA   = "f1000000-0000-0000-0000-000000000001"
	locB   = "f1000000-0000-0000-0000-000000000002"
	doorID = "f1000000-0000-0000-0000-000000000003"
)

// seedOrchestratorEntities inserts entity_registry rows for the orchestrator tests.
// ON CONFLICT DO NOTHING so re-runs are safe.
func seedOrchestratorEntities(t *testing.T, ctx context.Context) {
	t.Helper()
	pool := testPool(t)
	_, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		VALUES
		  ($1, $2, 'location', 'orch-loc-A'),
		  ($3, $2, 'location', 'orch-loc-B'),
		  ($4, $2, 'artifact', 'orch-door')
		ON CONFLICT (entity_id) DO NOTHING`,
		locA, worldID, locB, doorID)
	if err != nil {
		t.Fatalf("seed orchestrator entities: %v", err)
	}
}

// TestRunBeatPassthroughAndAdjudicated verifies the full five-stage orchestrator loop:
//   - ActorMoved (passthrough via apply_event): committed + ticks_advanced=5
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

	// Chain: player at locA moves to locB (ActorMoved → passthrough, ticks+5),
	// then two AttributeChanged on doorID (adjudicated).
	chain := []Attempt{
		{
			Type:         "ActorMoved",
			Stated:       "I move to locB",
			ToLocationID: locB,
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
		t.Fatalf("ticks_advanced = %d, want 5 (one ActorMoved between different locations)", outcome.TicksAdvanced)
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
			Type:         "ActorMoved",
			Stated:       "I move to locB to get away",
			ToLocationID: locB,
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
