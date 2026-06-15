package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArtifactIndex_PlayerHasNote_JonasDoesNot(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewIndexHandler(pool, true, "artifacts", "artifact") // debug → honor ?viewer=
	get := func(viewer string) string {
		req := httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/compendium/artifacts?viewer="+viewer, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("index status = %d", rec.Code)
		}
		return rec.Body.String()
	}
	if !strings.Contains(get(playerID), noteID) {
		t.Fatalf("note absent from Player artifact index")
	}
	if strings.Contains(get(jonasID), noteID) {
		t.Fatalf("LEAK: note present in Jonas artifact index")
	}
}

func TestTimeline_PlayerHasNote_JonasNoSecret(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewTimelineHandler(pool, true)
	get := func(viewer string) string {
		req := httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/compendium/timeline?viewer="+viewer, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("timeline status = %d", rec.Code)
		}
		return rec.Body.String()
	}
	if !strings.Contains(get(playerID), notePID) {
		t.Fatalf("Player timeline missing the note observation")
	}
	jonas := get(jonasID)
	if strings.Contains(jonas, notePID) {
		t.Fatalf("LEAK: note observation in Jonas timeline")
	}
	// Jonas holds his own self-move perceptions so his timeline is non-empty,
	// but the planted secret ("ledger" content) must never appear.
	if strings.Contains(strings.ToLower(jonas), "ledger") {
		t.Fatalf("LEAK: planted secret (ledger) found in Jonas timeline: %s", jonas)
	}
}
