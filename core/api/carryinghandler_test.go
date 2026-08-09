package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mara in the seeded play world. beathandler_test.go already names her cellar key (dlMaraKeyID) but
// not her; the whole point of the tests below is that those two are hers and stay hers.
const dlMaraID = "2ac70000-0000-0000-0000-0000000000a2"

type carryingPayload struct {
	SchemaVersion string `json:"schema_version"`
	WorldID       string `json:"world_id"`
	ViewerID      string `json:"viewer_id"`
	Carried       []struct {
		ID                  string  `json:"id"`
		Label               string  `json:"label"`
		State               string  `json:"state"`
		LastConfirmedTick   int64   `json:"last_confirmed_tick"`
		QuickInspectPreview *string `json:"quick_inspect_preview"`
		Container           *struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"container"`
		Decay struct {
			Stale              bool    `json:"stale"`
			LastConfirmedLabel *string `json:"last_confirmed_label"`
		} `json:"decay"`
	} `json:"carried"`
}

func getCarrying(t *testing.T, debug bool, world, query string) (*httptest.ResponseRecorder, carryingPayload) {
	t.Helper()
	pool := testPool(t)
	defer pool.Close()
	h := NewCarryingHandler(pool, debug)
	req := httptest.NewRequest(http.MethodGet, "/worlds/"+world+"/carrying"+query, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var p carryingPayload
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatalf("decode carrying payload: %v (body %s)", err, rec.Body.String())
		}
	}
	return rec, p
}

// The world says who you play as (world.player_entity_id, SPEC-028); the overlay is that actor's.
// Requires the play seed.
func TestCarrying_ResolvesTheWorldsOwnPlayer(t *testing.T) {
	rec, p := getCarrying(t, false, dlWorldID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if p.SchemaVersion != "carrying/1" {
		t.Fatalf("schema_version = %q, want carrying/1", p.SchemaVersion)
	}
	if p.ViewerID != dlKadeID {
		t.Fatalf("viewer_id = %q, want the world's player %q", p.ViewerID, dlKadeID)
	}
	if len(p.Carried) != 1 || p.Carried[0].ID != dlNoteID {
		t.Fatalf("carried = %+v, want exactly the sealed note %s", p.Carried, dlNoteID)
	}
	it := p.Carried[0]
	if it.State != "carried" {
		t.Fatalf("state = %q, want carried", it.State)
	}
	if it.Container != nil {
		t.Fatalf("container = %+v, want null for a thing directly on you", it.Container)
	}
	if it.LastConfirmedTick <= 0 {
		t.Fatalf("last_confirmed_tick = %d, want the tick of the confirming event (AC#3)", it.LastConfirmedTick)
	}
	if it.Label == "" {
		t.Fatalf("label is empty; the viewer's own naming must always render")
	}
}

// THE POINT OF THE SURFACE. Mara's cellar key is on Mara. A play-mode caller cannot reach it, and
// cannot reach it by ASKING either: outside debug the ?viewer= override is ignored, so "show me what
// that other actor is carrying" is unanswerable through this endpoint (PRD non-goal, and B-1 — the
// viewer is resolved server-side, D-7).
func TestCarrying_PlayModeIgnoresAViewerOverride(t *testing.T) {
	rec, p := getCarrying(t, false, dlWorldID, "?viewer="+dlMaraID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if p.ViewerID != dlKadeID {
		t.Fatalf("viewer_id = %q — a play-mode caller spoofed the carrier", p.ViewerID)
	}
	if strings.Contains(rec.Body.String(), dlMaraKeyID) {
		t.Fatalf("Mara's cellar key crossed the boundary into Kade's overlay: %s", rec.Body.String())
	}
}

// The debug override swaps the identity you play as — it does not widen anyone's overlay. As Mara
// you see Mara's key and not Kade's note, which is the same wall from the other side.
func TestCarrying_DebugOverrideSwapsTheCarrierAndNothingElse(t *testing.T) {
	rec, p := getCarrying(t, true, dlWorldID, "?viewer="+dlMaraID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if p.ViewerID != dlMaraID {
		t.Fatalf("viewer_id = %q, want %q", p.ViewerID, dlMaraID)
	}
	if len(p.Carried) != 1 || p.Carried[0].ID != dlMaraKeyID {
		t.Fatalf("carried = %+v, want exactly Mara's cellar key %s", p.Carried, dlMaraKeyID)
	}
	if strings.Contains(rec.Body.String(), dlNoteID) {
		t.Fatalf("Kade's note appeared in Mara's overlay: %s", rec.Body.String())
	}
}

// Carrying nothing is an ANSWER: 200 with an empty list, never a 404. The 0A fixture's Player holds
// nothing, so an overlay that 404'd on empty would tell a new world's player their pockets are
// broken rather than empty.
func TestCarrying_EmptyIsTwoHundredNotFourOhFour(t *testing.T) {
	rec, p := getCarrying(t, false, worldID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a viewer carrying nothing", rec.Code)
	}
	if p.Carried == nil {
		t.Fatalf("carried is null; it must always be an array")
	}
	if len(p.Carried) != 0 {
		t.Fatalf("carried = %+v, want empty for the fixture Player", p.Carried)
	}
}

// The route owns GET only. A write verb on a read surface must not fall through to the projection.
func TestCarrying_MatchesGetOnly(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewCarryingHandler(pool, false).(matcher)
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(m, "/worlds/"+dlWorldID+"/carrying", nil)
		if h.Match(req) {
			t.Fatalf("%s /carrying matched a read-only handler", m)
		}
	}
	if !h.Match(httptest.NewRequest(http.MethodGet, "/worlds/"+dlWorldID+"/carrying", nil)) {
		t.Fatalf("GET /carrying did not match")
	}
}

// A handler that works perfectly and was never added to newRouter is a 404 in production and a
// green suite. This drives the COMPOSED router — the same slice main() serves — so registration is
// covered rather than assumed. Nothing else in this package did that; the /carrying route was the
// occasion to fix it, and every future endpoint inherits the check.
func TestRouter_ServesCarrying(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	rt := newRouter(pool, false,
		mustBridge(t, NewFakeStructuredDriver("fake-structured:test", map[string]string{}),
			NewFakeTextDriver("fake-text:test")),
		nil)

	req := httptest.NewRequest(http.MethodGet, "/worlds/"+dlWorldID+"/carrying", nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /worlds/{w}/carrying through the real router = %d, want 200 (route not registered?)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"carrying/1"`) {
		t.Fatalf("router served something else on /carrying: %s", rec.Body.String())
	}
}
