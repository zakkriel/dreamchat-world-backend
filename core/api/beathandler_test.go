package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// rogueStructuredDriver REPORTS structured output but emits out-of-vocab. It binds to decompose
// (capability floor passes) yet proves the handler's DEFENSE-IN-DEPTH belt (DecodeAndValidateChainV2)
// still rejects out-of-vocab → 422, even if a bound driver misbehaves.
type rogueStructuredDriver struct{}

func (rogueStructuredDriver) Name() string                { return "rogue-structured" }
func (rogueStructuredDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }
func (rogueStructuredDriver) Generate(context.Context, GenRequest) (string, error) {
	return `[{"type":"attack","to":"x"}]`, nil
}

func mustBridge(t *testing.T, decompose, narrate Driver) *Bridge {
	t.Helper()
	b, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:      decompose,
		SeatNarrate.Name:        narrate,
		SeatResolve.Name:        NewFakeResolveDriver(),
		SeatCognitionBatch.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:     NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return b
}

// Happy path: position Player + Mara together at seedTavernID (existing seed entity, no registry row
// added) so the Communicated-gate passes and the beat commits end-to-end through the bridge. Event IDs
// are random per run and ticks start at ≥50000 so re-runs never hit canon_event_pkey or
// uq_ce_accepted_order collisions, and SQL-suite tick-range assertions (all ≤1400) are unaffected.
// After the beat, perception_subject rows are backfilled (mirrors seed_mara_0A) so SQL test 14's
// every-perception-has-a-subject invariant holds even when this test runs before the SQL suite.
// Writes persist (additive, legal origin) — make reset is run before go test per the Makefile.
func TestBeat_HappyPath_CommitsAndNarrates(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Compute base tick: ≥50000 and above any existing max to avoid uq_ce_accepted_order collision.
	var baseTick int
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// Position Player and Mara at seedTavernID using the uuid location model (migration 20260723100001).
	// Using the existing seed entity avoids entity_registry row additions (SQL test 40 asserts count=12).
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→tavern',$2,0,'accepted',now(),'public','fast_path')
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
		  VALUES (gen_random_uuid(),$1,'move','M→tavern',$4,0,'accepted',now(),'public','fast_path')
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
		worldID, baseTick, playerID, baseTick+1, seedTavernID, maraID)
	if err != nil {
		t.Fatalf("setup positions: %v", err)
	}

	// v2 format: Communicated (listener_id + content), not legacy "say".
	bridge := mustBridge(t,
		NewFakeStructuredDriver("fake-structured:test", map[string]string{
			"tell mara about the note": `[{"type":"Communicated","stated":"tell mara about the note","listener_id":"` + maraID + `","content":"the note"}]`,
		}),
		NewFakeTextDriver("fake-text:test"))
	h := NewBeatHandler(pool, true, bridge) // debug → honor ?viewer=

	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+playerID, strings.NewReader(`{"text":"tell mara about the note"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "narration") {
		t.Fatalf("response missing narration: %s", body)
	}
	if !strings.Contains(body, `"halt_reason":"completed"`) {
		t.Fatalf("beat did not commit end-to-end (halt_reason != completed): %s", body)
	}
	if strings.Contains(body, `"status":"accepted"`) {
		t.Fatalf("response leaked a raw canon row (B-1): %s", body)
	}

	// Backfill perception_subject for runtime-generated perceptions (mirrors seed_mara_0A backfill)
	// so SQL test 14's every-perception-has-a-subject invariant holds after this test runs.
	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// Defense-in-depth: a misbehaving structured driver that emits out-of-vocab is rejected by the
// handler's belt → 422 (the primary leash is the capability floor + constrained decoding; this is
// the backstop, SPEC-015/D-1).
func TestBeat_OutOfVocabularyRejectedByBelt(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewBeatHandler(pool, true, mustBridge(t, rogueStructuredDriver{}, NewFakeTextDriver("fake-text:test")))
	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beat?viewer="+playerID,
		strings.NewReader(`{"text":"anything"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422 (belt rejects out-of-vocab; SPEC-015)", rec.Code)
	}
}
