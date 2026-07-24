package main

// Station D exit gate — Task 7.
//
// TestStationD_FakeE2E (always runs): full HTTP path through the real beatHandler.
//   Fixture: Player + Mara co-located at seedTavernID, high ticks.
//   Decompose fake yields exactly: [{"type":"AttributeChanged","stated":"I lean on her to talk about the harbormaster","target_id":"<mara-uuid>"}]
//   Resolve fake (shared): therefore "fails", one AttributeChanged, truth + appearance texts.
//   Asserts: HTTP 200; halt_reason "completed"; a canon_event with event_type AttributeChanged
//   whose summary = the fake truth text; the PLAYER's perception_record for that event carries
//   the appearance text (canon truth ≠ player-perceived face, end-to-end); narration non-empty.
//
// TestStationD_LiveSmoke (gated): t.Skip unless DREAMCHAT_RESOLVE_API_KEY set.
//   Binds ONLY the resolve seat to newOpenAICompatDriver (decompose/narrate stay fakes).
//   Asserts STRUCTURE only — never model prose content.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// leanOnMaraText is the beat text used in both sub-tests.
const leanOnMaraText = "I lean on her to talk about the harbormaster"

// leanOnMaraDecomposeOut is the exact chain the fake decompose driver returns for leanOnMaraText.
// The target_id is maraID (from actorpage_test.go const).
const leanOnMaraDecomposeOut = `[{"type":"AttributeChanged","stated":"I lean on her to talk about the harbormaster","target_id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}]`

// The fake resolve driver emits these exact strings (see bridge_fakes.go fakeResolveDriver).
const fakeResolveTruth = "The attempt does not land; the target hardens and deflects."
const fakeResolveAppearance = "The target seems unmoved."

// seedPlayerAndMaraAtTavern co-locates Player + Mara at seedTavernID with ticks ≥ baseTick.
// Uses the existing seed entity so no entity_registry rows are added (SQL test 40 asserts count=12).
func seedPlayerAndMaraAtTavern(t *testing.T, ctx context.Context, pool interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (interface{ RowsAffected() int64 }, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) interface{ Scan(dest ...interface{}) error }
}, baseTick int) {
	t.Helper()
	// (unused shim — real seeding below uses pool directly)
}

// TestStationD_FakeE2E exercises the full HTTP beatHandler pipeline with fake drivers,
// proving the deception split end-to-end: canon carries TRUTH, player perceives APPEARANCE.
func TestStationD_FakeE2E(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// --- Fixture: compute high-water tick to avoid collision with SQL suite (all ≤1400) ---
	var baseTick int
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// --- Co-locate Player and Mara at seedTavernID ---
	// Two move events (gen_random_uuid avoids pkey collision on repeated runs).
	// state_mutation trigger updates actor_state; fn_actors_at reads actor_state.
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→tavern-stationD',$2,0,'accepted',now(),'public','fast_path')
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
		  VALUES (gen_random_uuid(),$1,'move','M→tavern-stationD',$4,0,'accepted',now(),'public','fast_path')
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
		t.Fatalf("seed positions: %v", err)
	}

	// --- Build bridge with inline structured-fake decompose driver ---
	// The fake decompose driver table maps the raw beat text → the AttributeChanged chain.
	// This is an inline variant of NewFakeStructuredDriver, identical to the beathandler_test.go pattern.
	decomposeDriver := NewFakeStructuredDriver("fake-structured:lean-on-mara", map[string]string{
		leanOnMaraText: leanOnMaraDecomposeOut,
	})

	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:      decomposeDriver,
		SeatNarrate.Name:        NewFakeTextDriver("fake-text:stationD"),
		SeatResolve.Name:        NewFakeResolveDriver(),
		SeatCognitionBatch.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:     NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}

	// debug=true so ?viewer= override is honored.
	h := NewBeatHandler(pool, true, bridge)

	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+playerID,
		strings.NewReader(`{"text":"`+leanOnMaraText+`"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// --- Assert HTTP 200 ---
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	// --- Assert halt_reason = "completed" ---
	if !strings.Contains(body, `"halt_reason":"completed"`) {
		t.Fatalf("halt_reason != completed\nbody: %s", body)
	}

	// --- Assert narration non-empty ---
	if !strings.Contains(body, `"narration"`) || strings.Contains(body, `"narration":""`) {
		t.Fatalf("narration missing or empty\nbody: %s", body)
	}

	// --- Parse the committed event IDs from the response ---
	var resp struct {
		Result struct {
			Committed  []string `json:"committed"`
			HaltReason string   `json:"halt_reason"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("parse response: %v\nbody: %s", err, body)
	}
	if len(resp.Result.Committed) == 0 {
		t.Fatalf("no committed events in response\nbody: %s", body)
	}

	// --- Assert: a canon_event with event_type AttributeChanged and summary = truth exists ---
	// The fake resolve driver emits one AttributeChanged event. Its truth becomes the canon summary.
	var eventType, canonSummary string
	evID := resp.Result.Committed[0]
	if err := pool.QueryRow(ctx,
		`SELECT event_type, summary FROM canon_event WHERE event_id=$1::uuid`,
		evID).Scan(&eventType, &canonSummary); err != nil {
		t.Fatalf("canon_event lookup for %s: %v", evID, err)
	}
	if eventType != "AttributeChanged" {
		t.Fatalf("canon event_type = %q, want AttributeChanged", eventType)
	}
	if canonSummary != fakeResolveTruth {
		t.Fatalf("canon summary = %q, want truth %q", canonSummary, fakeResolveTruth)
	}

	// --- Assert: PLAYER's perception_record carries appearance text (NOT truth) ---
	// This is the deception split: canon carries truth, the observer's perception carries appearance.
	var playerPerception string
	if err := pool.QueryRow(ctx,
		`SELECT content FROM perception_record WHERE source_event_id=$1::uuid AND holder_id=$2::uuid`,
		evID, playerID).Scan(&playerPerception); err != nil {
		t.Fatalf("player perception lookup (event %s, player %s): %v", evID, playerID, err)
	}
	if playerPerception != fakeResolveAppearance {
		t.Fatalf("player perception = %q, want appearance %q (deception split broken)", playerPerception, fakeResolveAppearance)
	}
	// Confirm the deception split: perception ≠ canon truth.
	if playerPerception == canonSummary {
		t.Fatalf("player perception equals canon truth — deception split is not working: %q", playerPerception)
	}

	// --- Backfill perception_subject (mirrors beathandler_test.go pattern) ---
	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// TestStationD_LiveSmoke is a gated integration test: it binds ONLY the resolve seat to a live
// openai-compat provider. Decompose and narrate stay fakes so this test exercises ONLY the resolver.
// Skip unless DREAMCHAT_RESOLVE_API_KEY is set.
func TestStationD_LiveSmoke(t *testing.T) {
	apiKey := os.Getenv("DREAMCHAT_RESOLVE_API_KEY")
	if apiKey == "" {
		t.Skip("DREAMCHAT_RESOLVE_API_KEY not set — live smoke skipped")
	}
	baseURL := os.Getenv("DREAMCHAT_RESOLVE_BASE_URL")
	model := os.Getenv("DREAMCHAT_RESOLVE_MODEL")
	if baseURL == "" || model == "" {
		t.Fatalf("DREAMCHAT_RESOLVE_API_KEY is set but DREAMCHAT_RESOLVE_BASE_URL and/or DREAMCHAT_RESOLVE_MODEL are missing — cannot build live resolver")
	}

	// Build the live resolve driver.
	resolveDriver, err := newOpenAICompatDriver(DriverConfig{
		Provider: "openai-compat",
		Model:    model,
		Params: map[string]string{
			"base_url": baseURL,
			"api_key":  apiKey,
		},
	})
	if err != nil {
		t.Fatalf("build live resolve driver: %v", err)
	}

	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// --- Fixture: high-water tick ---
	var baseTick int
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// --- Co-locate Player and Mara at seedTavernID ---
	_, err = pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→tavern-live',$2,0,'accepted',now(),'public','fast_path')
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
		  VALUES (gen_random_uuid(),$1,'move','M→tavern-live',$4,0,'accepted',now(),'public','fast_path')
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
		t.Fatalf("seed positions (live): %v", err)
	}

	// --- Build bridge: only resolve seat is live; decompose + narrate stay fakes ---
	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:      NewFakeStructuredDriver("fake-structured:lean-on-mara-live", map[string]string{leanOnMaraText: leanOnMaraDecomposeOut}),
		SeatNarrate.Name:        NewFakeTextDriver("fake-text:stationD-live"),
		SeatResolve.Name:        resolveDriver,
		SeatCognitionBatch.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:     NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge (live): %v", err)
	}

	h := NewBeatHandler(pool, true, bridge)

	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+playerID,
		strings.NewReader(`{"text":"`+leanOnMaraText+`"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Live models may return 200 with any valid halt_reason.
	if rec.Code != 200 {
		t.Fatalf("live smoke: status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	// STRUCTURAL assertions only — never assert model prose content.
	var resp struct {
		Result struct {
			Committed  []string `json:"committed"`
			HaltReason string   `json:"halt_reason"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("live smoke: parse response: %v\nbody: %s", err, body)
	}

	// halt_reason must be completed or bounce — bounce is legitimate ("impossible" ruling).
	haltReason := resp.Result.HaltReason
	if haltReason != "completed" && haltReason != "bounce" {
		t.Fatalf("live smoke: halt_reason = %q, want completed|bounce\nbody: %s", haltReason, body)
	}

	// If completed, exactly one adjudicated canon event must have been committed.
	if haltReason == "completed" {
		if len(resp.Result.Committed) == 0 {
			t.Fatalf("live smoke: halt_reason=completed but no committed events\nbody: %s", body)
		}
		// The committed event must exist in canon with a non-empty summary.
		evID := resp.Result.Committed[0]
		var canonSummary string
		if err := pool.QueryRow(ctx,
			`SELECT summary FROM canon_event WHERE event_id=$1::uuid`, evID).Scan(&canonSummary); err != nil {
			t.Fatalf("live smoke: canon_event lookup for %s: %v", evID, err)
		}
		if canonSummary == "" {
			t.Fatalf("live smoke: canon event %s has empty summary", evID)
		}

		// Backfill perception_subject so invariant tests stay green.
		perceptionSubjectBackfill(t, ctx, pool, baseTick)
	}
}
