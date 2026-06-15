package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	noteID       = "a4000000-0000-0000-0000-0000000000a1"
	notePID      = "dca70000-0000-0000-0000-000000000b01"
	tavernID     = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	o1ID         = "00000000-0000-0000-0000-000000000001"
	fabricatedID = "deadbeef-0000-0000-0000-000000000000"
)

func TestArtifactPage_PlayerSeesNote(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewPageHandler(pool, false, "artifacts", "fn_artifact_page")
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/artifacts/"+noteID+"/page", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), notePID) {
		t.Fatalf("discovery observation absent for Player: %s", rec.Body.String())
	}
}

func TestArtifactPage_JonasGets404_IndistinguishableFromFabricated(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewPageHandler(pool, true, "artifacts", "fn_artifact_page") // debug → honor ?viewer=
	get := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/compendium/artifacts/"+id+"/page?viewer="+jonasID, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	note := get(noteID)
	fake := get(fabricatedID)
	if note.Code != 404 {
		t.Fatalf("Jonas note page status = %d, want 404 (existence leak via 200)", note.Code)
	}
	if fake.Code != 404 {
		t.Fatalf("fabricated id status = %d, want 404", fake.Code)
	}
	if note.Body.String() != fake.Body.String() {
		t.Fatalf("note 404 distinguishable from fabricated 404: %q vs %q",
			note.Body.String(), fake.Body.String())
	}
}

func TestActorPage_UnperceivedActor404(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewActorPageHandler(pool, false) // viewer = Player
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/actors/"+o1ID+"/page", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("unperceived actor O1 status = %d, want 404 (latent leak closed)", rec.Code)
	}
}

func TestLocationPage_JonasTavern200Empty(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewPageHandler(pool, true, "locations", "fn_location_page")
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/locations/"+tavernID+"/page?viewer="+jonasID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 { // Tavern is common knowledge → exists for Jonas
		t.Fatalf("Jonas Tavern status = %d, want 200 (CK)", rec.Code)
	}
	if strings.Contains(rec.Body.String(), notePID) {
		t.Fatalf("Jonas Tavern leaked a Player perception")
	}
}
