package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// ─── inline drivers for ruled tests ──────────────────────────────────────────

// inlineRulingDriver returns a fixed ruling JSON string regardless of the prompt.
type inlineRulingDriver struct {
	name    string
	ruling  string
	callCount int
}

func (d *inlineRulingDriver) Name() string                { return d.name }
func (d *inlineRulingDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }
func (d *inlineRulingDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: used without schema", d.name)
	}
	d.callCount++
	return d.ruling, nil
}

// countingRulingDriver returns different rulings on successive calls.
type countingRulingDriver struct {
	name     string
	rulings  []string
	callCount int
}

func (d *countingRulingDriver) Name() string                { return d.name }
func (d *countingRulingDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }
func (d *countingRulingDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: used without schema", d.name)
	}
	idx := d.callCount
	d.callCount++
	if idx < len(d.rulings) {
		return d.rulings[idx], nil
	}
	// Return invalid JSON if we run out of rulings
	return `{"reasoning":"","therefore":"impossible","outcome":{"kind":"bounce"}}`, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// seedActorAtLoc positions an actor at a location by inserting a move event + state_mutation.
func seedActorAtLoc(t *testing.T, ctx context.Context, actorID, locID string, tick int64, seq int) {
	t.Helper()
	pool := testPool(t)
	_, err := pool.Exec(ctx, `
		WITH ev AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','seed-pos',$2,$3,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$4,'actor','instigator' FROM ev
		)
		INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		SELECT $1,event_id,$4,'actor','attrs.location_id',to_jsonb($5::text),$2,$3 FROM ev`,
		worldID, tick, seq, actorID, locID)
	if err != nil {
		t.Fatalf("seedActorAtLoc(%s, %s): %v", actorID, locID, err)
	}
}

// baseTick returns a tick well above any existing max.
func nextBaseTick(t *testing.T, ctx context.Context) int64 {
	t.Helper()
	pool := testPool(t)
	var tick int64
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,70000)`,
		worldID).Scan(&tick); err != nil {
		t.Fatalf("nextBaseTick: %v", err)
	}
	return tick
}

// validRulingJSON builds a v2 ruling JSON for an AttributeChanged event.
func validRulingJSON(actorID, targetID, truth, appearance string) string {
	r := map[string]any{
		"reasoning": "Test ruling for adjudication.",
		"therefore": "succeeds",
		"outcome": map[string]any{
			"kind": "resolved",
			"events": []map[string]any{
				{
					"type":       "AttributeChanged",
					"actor_id":   actorID,
					"target_id":  targetID,
					"truth":      truth,
					"appearance": appearance,
					"visible":    true,
				},
			},
		},
	}
	b, _ := json.Marshal(r)
	return string(b)
}

// ─── tests ────────────────────────────────────────────────────────────────────

// TestAdjudicated_DeceptionE2E verifies the truth/appearance split in ruled commits:
//   - canon summary = truth text (CANON NEVER LIES)
//   - a co-located observer's perception content = appearance text
func TestAdjudicated_DeceptionE2E(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	bt := nextBaseTick(t, ctx)
	seedActorAtLoc(t, ctx, playerID, locA, bt, 0)
	seedActorAtLoc(t, ctx, maraID, locA, bt, 1)

	const truth = "Mara quietly pockets the coin."
	const appearance = "Mara shrugs, hands empty."

	driver := &inlineRulingDriver{
		name:   "deception-driver",
		ruling: validRulingJSON(playerID, doorID, truth, appearance),
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        driver,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	ar, err := orc.adjudicate(ctx, worldID, []ActorAttempt{
		{ActorID: playerID, Attempt: Attempt{Type: "AttributeChanged", Stated: "I try to bribe the guard", TargetID: doorID}},
	}, nil, bt+2, 0)
	if err != nil {
		t.Fatalf("adjudicate: %v", err)
	}
	if ar.Halt != "" {
		t.Fatalf("expected no halt, got %q", ar.Halt)
	}
	if len(ar.Committed) != 1 {
		t.Fatalf("expected 1 committed event, got %d", len(ar.Committed))
	}

	evID := ar.Committed[0]

	// Canon summary = truth (CANON NEVER LIES).
	var canonSummary string
	if err := pool.QueryRow(ctx,
		`SELECT summary FROM canon_event WHERE event_id=$1::uuid`, evID).Scan(&canonSummary); err != nil {
		t.Fatalf("read canon summary: %v", err)
	}
	if canonSummary != truth {
		t.Fatalf("canon summary = %q, want truth %q", canonSummary, truth)
	}

	// Co-located observer (Mara) gets appearance text.
	var maraContent string
	if err := pool.QueryRow(ctx,
		`SELECT content FROM perception_record WHERE source_event_id=$1::uuid AND holder_id=$2::uuid`,
		evID, maraID).Scan(&maraContent); err != nil {
		t.Fatalf("read mara perception: %v", err)
	}
	if maraContent != appearance {
		t.Fatalf("mara perception = %q, want appearance %q", maraContent, appearance)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(bt))
}

// TestAdjudicated_CommunicatedCommits verifies that a Communicated event from a ruling commits.
func TestAdjudicated_CommunicatedCommits(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	bt := nextBaseTick(t, ctx)
	seedActorAtLoc(t, ctx, playerID, locA, bt, 0)
	seedActorAtLoc(t, ctx, maraID, locA, bt, 1)

	// Two-event ruling: AttributeChanged (targeting maraID, already in slice) then Communicated.
	// Both entities (playerID actor, maraID target/listener) are in the slice because they appear
	// in the attempt's fields and are gathered by gather_slice.
	r := map[string]any{
		"reasoning": "The player signals success and shouts to Mara.",
		"therefore": "succeeds",
		"outcome": map[string]any{
			"kind": "resolved",
			"events": []map[string]any{
				{
					"type":      "AttributeChanged",
					"actor_id":  playerID,
					"target_id": maraID,
					"truth":     "The player taps Mara's shoulder signalling success.",
					"visible":   true,
				},
				{
					"type":        "Communicated",
					"actor_id":    playerID,
					"listener_id": maraID,
					"content":     "I did it!",
					"truth":       "The player shouts to Mara: I did it!",
					"visible":     true,
				},
			},
		},
	}
	rJSON, _ := json.Marshal(r)

	driver := &inlineRulingDriver{
		name:   "communicated-driver",
		ruling: string(rJSON),
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        driver,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	ar, err := orc.adjudicate(ctx, worldID, []ActorAttempt{
		{ActorID: playerID, Attempt: Attempt{Type: "Communicated", Stated: "I shout my success to Mara", ListenerID: maraID, Content: "I did it!"}},
	}, nil, bt+2, 0)
	if err != nil {
		t.Fatalf("adjudicate: %v", err)
	}
	if ar.Halt != "" {
		t.Fatalf("expected no halt, got %q", ar.Halt)
	}
	if len(ar.Committed) != 2 {
		t.Fatalf("expected 2 committed events, got %d: %v", len(ar.Committed), ar.Committed)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(bt))
}

// TestAdjudicated_AttributeWriteLandsInAttrs verifies that attribute_writes are applied.
func TestAdjudicated_AttributeWriteLandsInAttrs(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	bt := nextBaseTick(t, ctx)
	seedActorAtLoc(t, ctx, playerID, locA, bt, 0)

	r := map[string]any{
		"reasoning": "The player opens the door.",
		"therefore": "succeeds",
		"outcome": map[string]any{
			"kind": "resolved",
			"events": []map[string]any{
				{
					"type":      "AttributeChanged",
					"actor_id":  playerID,
					"target_id": doorID,
					"truth":     "The door swings open.",
					"visible":   true,
				},
			},
			"attribute_writes": []map[string]any{
				{
					"target_id": doorID,
					"attribute": "open",
					"value":     true,
					"tier":      1,
				},
			},
		},
	}
	rJSON, _ := json.Marshal(r)

	driver := &inlineRulingDriver{
		name:   "attrwrite-driver",
		ruling: string(rJSON),
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        driver,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	ar, err := orc.adjudicate(ctx, worldID, []ActorAttempt{
		{ActorID: playerID, Attempt: Attempt{Type: "AttributeChanged", Stated: "I open the door", TargetID: doorID}},
	}, nil, bt+2, 0)
	if err != nil {
		t.Fatalf("adjudicate: %v", err)
	}
	if ar.Halt != "" {
		t.Fatalf("expected no halt, got %q", ar.Halt)
	}
	if len(ar.Committed) < 1 {
		t.Fatalf("expected >=1 committed event, got 0")
	}

	// Read artifact_state attrs for doorID → assert "open" = true.
	var attrs map[string]any
	if err := pool.QueryRow(ctx,
		`SELECT attrs FROM artifact_state WHERE world_id=$1::uuid AND entity_id=$2::uuid`,
		worldID, doorID).Scan(&attrs); err != nil {
		t.Fatalf("read artifact_state: %v", err)
	}
	openVal, ok := attrs["open"]
	if !ok {
		t.Fatalf("artifact_state.attrs missing 'open' key: %v", attrs)
	}
	if openBool, ok := openVal.(bool); !ok || !openBool {
		t.Fatalf("artifact_state.attrs['open'] = %v, want true", openVal)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(bt))
}

// TestAdjudicated_OutOfSliceUUIDRepairThenBounce verifies that two consecutive UUID violations
// result in Halt="bounce" and exactly 2 driver calls.
func TestAdjudicated_OutOfSliceUUIDRepairThenBounce(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	bt := nextBaseTick(t, ctx)
	seedActorAtLoc(t, ctx, playerID, locA, bt, 0)

	// Both calls emit a ruling with a random UUID not in the slice.
	const badUUID = "deadbeef-dead-dead-dead-deaddeadbeef"
	badRuling := func() string {
		r := map[string]any{
			"reasoning": "bad actor.",
			"therefore": "succeeds",
			"outcome": map[string]any{
				"kind": "resolved",
				"events": []map[string]any{
					{
						"type":      "AttributeChanged",
						"actor_id":  badUUID,
						"target_id": doorID,
						"truth":     "something happens.",
						"visible":   true,
					},
				},
			},
		}
		b, _ := json.Marshal(r)
		return string(b)
	}

	driver := &countingRulingDriver{
		name:    "bad-uuid-driver",
		rulings: []string{badRuling(), badRuling()},
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        driver,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	ar, err := orc.adjudicate(ctx, worldID, []ActorAttempt{
		{ActorID: playerID, Attempt: Attempt{Type: "AttributeChanged", Stated: "test", TargetID: doorID}},
	}, nil, bt+2, 0)
	if err != nil {
		t.Fatalf("adjudicate: %v", err)
	}
	if ar.Halt != "bounce" {
		t.Fatalf("halt = %q, want %q", ar.Halt, "bounce")
	}
	if len(ar.Committed) != 0 {
		t.Fatalf("expected 0 committed, got %d", len(ar.Committed))
	}
	if driver.callCount != 2 {
		t.Fatalf("driver called %d times, want 2", driver.callCount)
	}
}

// TestAdjudicated_PartialRulingRollback verifies that a two-event ruling where event[1]
// fails the structural floor (gate_reject: Communicated to a non-co-located listener)
// results in halt="ruled_event_rejected" AND zero events durable — event[0] is rolled back.
//
// This is the transaction test for Fix 2: the entire commit phase runs in one pgx transaction;
// a gate_reject mid-loop rolls back even the already-applied events from earlier in the loop.
func TestAdjudicated_PartialRulingRollback(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	bt := nextBaseTick(t, ctx)

	// Player at locA, Mara at locB → NOT co-located.
	// The two-event ruling will: event[0]=AttributeChanged(player→door) [would pass],
	// event[1]=Communicated(player→mara) [gate_reject: listener not co-located].
	seedActorAtLoc(t, ctx, playerID, locA, bt, 0)
	seedActorAtLoc(t, ctx, maraID, locB, bt, 1)

	// Record canon_event count before — must be unchanged after the rollback.
	var beforeCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1::uuid`, worldID).Scan(&beforeCount); err != nil {
		t.Fatalf("count before: %v", err)
	}

	// Two-event ruling: first event valid, second Communicated to non-co-located Mara.
	r := map[string]any{
		"reasoning": "Player acts then speaks to Mara.",
		"therefore": "succeeds",
		"outcome": map[string]any{
			"kind": "resolved",
			"events": []map[string]any{
				{
					"type":      "AttributeChanged",
					"actor_id":  playerID,
					"target_id": doorID,
					"truth":     "The player examines the door.",
					"visible":   true,
				},
				{
					"type":        "Communicated",
					"actor_id":    playerID,
					"listener_id": maraID,
					"content":     "Hey Mara!",
					"truth":       "The player shouts across the room to Mara.",
					"visible":     true,
				},
			},
		},
	}
	rJSON, _ := json.Marshal(r)

	driver := &inlineRulingDriver{
		name:   "partial-ruling-driver",
		ruling: string(rJSON),
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        driver,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	// Two attempts so both doorID and maraID pass verdictRuling (are in sliceIDs via participant IDs).
	ar, err := orc.adjudicate(ctx, worldID, []ActorAttempt{
		{ActorID: playerID, Attempt: Attempt{Type: "AttributeChanged", Stated: "I examine the door", TargetID: doorID}},
		{ActorID: playerID, Attempt: Attempt{Type: "Communicated", Stated: "I call out to Mara", ListenerID: maraID, Content: "Hey Mara!"}},
	}, nil, bt+2, 0)
	if err != nil {
		t.Fatalf("adjudicate: %v", err)
	}
	if ar.Halt != "ruled_event_rejected" {
		t.Fatalf("halt = %q, want %q", ar.Halt, "ruled_event_rejected")
	}
	if len(ar.Committed) != 0 {
		t.Fatalf("expected 0 committed (rollback), got %d: %v", len(ar.Committed), ar.Committed)
	}

	// Assert canon_event count is unchanged — event[0] must have been rolled back.
	var afterCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1::uuid`, worldID).Scan(&afterCount); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if afterCount != beforeCount {
		t.Fatalf("canon_event count changed: before=%d after=%d — event[0] was NOT rolled back", beforeCount, afterCount)
	}
}

// TestAdjudicated_NAry_TwoAttemptsOneGenCall verifies that adjudicate makes ONE Resolve
// call for multiple attempts (n-ary judgment).
func TestAdjudicated_NAry_TwoAttemptsOneGenCall(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	bt := nextBaseTick(t, ctx)
	seedActorAtLoc(t, ctx, playerID, locA, bt, 0)

	// Valid ruling for both attempts combined.
	r := map[string]any{
		"reasoning": "Both attempts are resolved together.",
		"therefore": "succeeds",
		"outcome": map[string]any{
			"kind": "resolved",
			"events": []map[string]any{
				{
					"type":      "AttributeChanged",
					"actor_id":  playerID,
					"target_id": doorID,
					"truth":     "Combined outcome.",
					"visible":   true,
				},
			},
		},
	}
	rJSON, _ := json.Marshal(r)

	driver := &countingRulingDriver{
		name:    "nary-driver",
		rulings: []string{string(rJSON), string(rJSON)},
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        driver,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	ar, err := orc.adjudicate(ctx, worldID, []ActorAttempt{
		{ActorID: playerID, Attempt: Attempt{Type: "AttributeChanged", Stated: "first attempt", TargetID: doorID}},
		{ActorID: playerID, Attempt: Attempt{Type: "AttributeChanged", Stated: "second attempt", TargetID: doorID}},
	}, nil, bt+2, 0)
	if err != nil {
		t.Fatalf("adjudicate: %v", err)
	}
	if ar.Halt != "" {
		t.Fatalf("expected no halt, got %q", ar.Halt)
	}
	// One generate call for both attempts.
	if driver.callCount != 1 {
		t.Fatalf("driver called %d times, want 1 (n-ary)", driver.callCount)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(bt))
}
