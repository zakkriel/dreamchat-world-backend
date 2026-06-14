package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const maraID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
const aboutMaraPID = "dca70000-0000-0000-0000-000000000a01"

func TestActorPage_DefaultViewerSeesAboutMara(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewActorPageHandler(pool, false) // debug=false: viewer resolves to Player
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/actors/"+maraID+"/page", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	var probe map[string]any
	if err := json.Unmarshal([]byte(body), &probe); err != nil {
		t.Fatalf("payload not valid JSON: %v — %s", err, body)
	}
	if probe["schema_version"] != "actor_page/1" {
		t.Fatalf("missing/wrong schema_version: %s", body)
	}
	if !strings.Contains(body, aboutMaraPID) {
		t.Fatalf("about-Mara perception ABSENT for Player (should be present): %s", body)
	}
}

func TestActorPage_DebugViewerJonas_NoLeak(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewActorPageHandler(pool, true) // debug=true: honor ?viewer=
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/actors/"+maraID+"/page?viewer="+jonasID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	// GATE-CRITICAL: the private about-Mara perception must NOT appear for Jonas, and the
	// canon summary must never appear for anyone.
	if strings.Contains(body, aboutMaraPID) {
		t.Fatalf("LEAK: about-Mara perception present for Jonas: %s", body)
	}
	if strings.Contains(body, "P tells M") {
		t.Fatalf("LEAK: canon summary in payload: %s", body)
	}
	// sanity: it IS a well-formed payload (not an empty error body the test could pass on vacuously)
	var probe map[string]any
	if err := json.Unmarshal([]byte(body), &probe); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if probe["schema_version"] != "actor_page/1" {
		t.Fatalf("not a real actor page payload: %s", body)
	}
	_ = context.Background()
}
