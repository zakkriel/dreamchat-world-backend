package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Happy path: position Player + Mara together at a seed-clean label ('hall', off the noise map) so
// the say-gate passes and the beat commits end-to-end. Writes persist (additive, high ticks, legal
// origin) — harmless to sibling tests; the gate runs `make reset` before `go test`.
func TestBeat_HappyPath_CommitsAndNarrates(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// setup: Player + Mara co-present at 'hall' via accepted move events (trigger maintains state).
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

	h := NewBeatHandler(pool, true, // debug → honor ?viewer=
		NewFakeDecomposer(map[string]string{
			"tell mara about the note": `[{"type":"say","listener":"` + maraID + `","content":"the note"}]`,
		}),
		NewFakeNarrator("Scene:"))
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
	// B-1: no canon vocabulary crosses the boundary (no raw canon_event row shape).
	if strings.Contains(body, `"status":"accepted"`) {
		t.Fatalf("response leaked a raw canon row: %s", body)
	}
}

func TestBeat_OutOfVocabularyRejected(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewBeatHandler(pool, true,
		NewFakeDecomposer(map[string]string{"attack mara": `[{"type":"attack","target":"` + maraID + `"}]`}),
		NewFakeNarrator("Scene:"))
	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beat?viewer="+playerID,
		strings.NewReader(`{"text":"attack mara"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422 (the leash rejects out-of-vocabulary; SPEC-015)", rec.Code)
	}
}
