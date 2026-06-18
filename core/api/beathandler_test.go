package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// rogueStructuredDriver REPORTS structured output but emits out-of-vocab. It binds to decompose
// (capability floor passes) yet proves the handler's DEFENSE-IN-DEPTH belt (DecodeAndValidateChain)
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
		SeatDecompose.Name: decompose,
		SeatNarrate.Name:   narrate,
	}, SeatDecompose, SeatNarrate)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return b
}

// Happy path: position Player + Mara together at a seed-clean label ('hall', off the noise map) so
// the say-gate passes and the beat commits end-to-end through the bridge. Writes persist (additive,
// high ticks, legal origin) — harmless to sibling tests; the gate runs `make reset` before `go test`.
func TestBeat_HappyPath_CommitsAndNarrates(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin) VALUES
		 ('e5000000-0000-0000-0000-00000000bc01','`+worldID+`','move','P→hall',300,0,'accepted',now(),'public','fast_path'),
		 ('e5000000-0000-0000-0000-00000000bc02','`+worldID+`','move','M→hall',301,0,'accepted',now(),'public','fast_path');
		INSERT INTO event_participant VALUES
		 ('e5000000-0000-0000-0000-00000000bc01','`+playerID+`','actor','instigator'),
		 ('e5000000-0000-0000-0000-00000000bc02','`+maraID+`','actor','instigator');
		INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq) VALUES
		 ('`+worldID+`','e5000000-0000-0000-0000-00000000bc01','`+playerID+`','actor','attrs.location_id',to_jsonb('hall'::text),300,0),
		 ('`+worldID+`','e5000000-0000-0000-0000-00000000bc02','`+maraID+`','actor','attrs.location_id',to_jsonb('hall'::text),301,0);`)
	if err != nil {
		t.Fatalf("setup positions: %v", err)
	}

	bridge := mustBridge(t,
		NewFakeStructuredDriver("fake-structured:test", map[string]string{
			"tell mara about the note": `[{"type":"say","listener":"` + maraID + `","content":"the note"}]`,
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
	// NOTE: Go's json.Marshal COMPACTS json.RawMessage, so postgres's ": " spacing becomes ":".
	if !strings.Contains(body, `"halt_reason":"completed"`) {
		t.Fatalf("beat did not commit end-to-end (halt_reason != completed): %s", body)
	}
	if strings.Contains(body, `"status":"accepted"`) {
		t.Fatalf("response leaked a raw canon row (B-1): %s", body)
	}
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
