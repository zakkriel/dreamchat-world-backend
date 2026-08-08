package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// addPortal wires a portal between two rooms exactly the way the real world stores one
// (seed_drowned_lantern.sql): an artifact whose attrs carry open/locked and a `connects` array of the
// two location ids — and, critically, NO `location_id`. That missing key is why the co-located
// artifact query never returned the doors, so a fixture that invented a location_id here would agree
// with the bug instead of the world. Returns the portal's id.
func addPortal(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, a, b, name string, open, locked bool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&id); err != nil {
		t.Fatalf("mint portal id: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES ($1,$2,'artifact',$3)`,
		id, worldID, name); err != nil {
		t.Fatalf("register portal: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO artifact_state (entity_id, world_id, attrs)
		 VALUES ($1,$2, jsonb_build_object('open',$3::boolean,'locked',$4::boolean,
		                                   'connects', jsonb_build_array($5::text,$6::text)))`,
		id, worldID, open, locked, a, b); err != nil {
		t.Fatalf("portal state: %v", err)
	}
	return id
}

func candidateByID(cands []Candidate, id string) *Candidate {
	for i := range cands {
		if cands[i].ID == id {
			return &cands[i]
		}
	}
	return nil
}

// SPEC-030. Before this, standing in a room offered the player no way out of it: the only location
// ever handed to decompose was the room itself, and portals carry `connects` with no `location_id`
// so the doors were invisible too. ActorMoved could only target where you already stood.
func TestPayload_OffersThePortalsOfThisRoomAndWhereTheyLead(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)
	door := addPortal(t, ctx, pool, id.World, id.Place, id.ElsewherePlace, "a plain door", true, false)

	payload, err := (&beatHandler{pool: pool}).payload(ctx, id.World, id.Viewer)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}

	if c := candidateByID(payload.Candidates, door); c == nil {
		t.Fatal("the door of the room the viewer is standing in is not a candidate — the room has no visible exits")
	} else if c.Kind != "artifact" {
		t.Fatalf("door kind = %q, want artifact", c.Kind)
	}

	if c := candidateByID(payload.Candidates, id.ElsewherePlace); c == nil {
		t.Fatal("the room on the other side of a visible door is not a candidate — ActorMoved has nowhere to go")
	} else if c.Kind != "location" {
		t.Fatalf("neighbour kind = %q, want location", c.Kind)
	}

	// The room you are in is still offered, and is still what Here names.
	if candidateByID(payload.Candidates, id.Place) == nil {
		t.Fatal("the current room stopped being a candidate")
	}
	if payload.Here != id.Place {
		t.Fatalf("Here = %q, want the room the viewer stands in %q", payload.Here, id.Place)
	}
}

// The wall. Offering the room next door must not offer the people in it: a door tells you there is
// somewhere on the other side, never who is standing there (B-1, I-3).
func TestPayload_NeighbouringRoomDoesNotExposeItsPeople(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)
	addPortal(t, ctx, pool, id.World, id.Place, id.ElsewherePlace, "a plain door", true, false)

	payload, err := (&beatHandler{pool: pool}).payload(ctx, id.World, id.Viewer)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}

	if candidateByID(payload.Candidates, id.ElsewherePlace) == nil {
		t.Fatal("precondition failed: the neighbouring room should be a candidate")
	}
	if c := candidateByID(payload.Candidates, id.Stranger); c != nil {
		t.Fatalf("the stranger in the next room leaked into candidates as %q — a door reveals the room, never its occupants", c.Name)
	}
}

// A shut door still names its room. Passage is decided by the accessibility floor at commit time
// (fn_actor_move_permitted), never by hiding the target: offering it is what lets the world refuse
// with a reason, and hiding it is what made the refusal unreachable.
func TestPayload_AShutDoorStillNamesTheRoomBehindIt(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)
	addPortal(t, ctx, pool, id.World, id.Place, id.ElsewherePlace, "a barred door", false, true)

	payload, err := (&beatHandler{pool: pool}).payload(ctx, id.World, id.Viewer)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}

	if candidateByID(payload.Candidates, id.ElsewherePlace) == nil {
		t.Fatal("a shut door hid the room behind it; the player can no longer even attempt the move")
	}
}

// REGRESSION for the defect SPEC-030's own fix introduced. buildScene used to resolve the place as
// "the last candidate of kind location", which was only ever right while exactly one location could
// be a candidate. The moment the neighbouring rooms became candidates, the scene endpoint started
// naming a room the player was not in — walk through the front door and it still reported the room
// behind you. Caught by hand-driving the endpoint, not by any existing test.
func TestBuildScene_NamesTheRoomYouAreInNotTheOneNextDoor(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)
	addPortal(t, ctx, pool, id.World, id.Place, id.ElsewherePlace, "a plain door", true, false)

	rec := sceneGet(t, NewSceneHandler(pool, true), id.World, id.Viewer)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var scene sceneView
	if err := json.Unmarshal(rec.Body.Bytes(), &scene); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if scene.Place.ID != id.Place {
		t.Fatalf("place.id = %q, want the room the viewer stands in %q — the scene named a neighbour", scene.Place.ID, id.Place)
	}
	if scene.Place.Label != "The Rusty Anchor" {
		t.Fatalf("place.label = %q, want the viewer's own name for the room he is in", scene.Place.Label)
	}
	// The neighbouring room being nameable must not make its occupant a participant.
	for _, p := range scene.Participants {
		if p.ID == id.Stranger {
			t.Fatal("the stranger next door appeared in the participants strip")
		}
	}
}

// beat_frame/2 — unresolved_candidates carries {id, label}, not bare ids. v1 shipped ids alone, so
// the frontend could not name what it was asking about and would not invent a name (B-1/D-7): the
// "which did you mean?" affordance could never render. The ask was unanswerable because it was
// unsayable.
func TestLabelCandidates_NamesEachOneInTheViewersOwnWords(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)

	got, err := labelCandidates(ctx, pool, id.World, id.Viewer, []string{id.Companion, id.Place})
	if err != nil {
		t.Fatalf("labelCandidates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want 2: %+v", len(got), got)
	}
	// Engine order is preserved — the frontend keys disambiguation on list position.
	if got[0].ID != id.Companion || got[1].ID != id.Place {
		t.Fatalf("order not preserved: %+v", got)
	}
	// The viewer knows the Companion only by descriptor, never the canonical registry secret.
	if got[0].Label != "a weary road companion" {
		t.Fatalf("label = %q, want the viewer's own descriptor", got[0].Label)
	}
	if got[1].Label != "The Rusty Anchor" {
		t.Fatalf("label = %q, want the viewer's own name for the place", got[1].Label)
	}
}

// The wall, on this surface too: the list may never surface a canonical name the viewer does not
// hold. Asking "which did you mean?" must not answer a question the player never got to ask.
func TestLabelCandidates_NeverLeaksACanonicalName(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)

	got, err := labelCandidates(ctx, pool, id.World, id.Viewer, []string{id.Companion, id.Place})
	if err != nil {
		t.Fatalf("labelCandidates: %v", err)
	}
	for _, c := range got {
		if strings.Contains(c.Label, "SECRET") {
			t.Fatalf("candidate %s leaked a canonical name: %q", c.ID, c.Label)
		}
	}
}

// No candidates must serialise as [] and never null — the frontend renders the list unconditionally.
func TestLabelCandidates_EmptyIsAnEmptyList(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	got, err := labelCandidates(ctx, pool, "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000", nil)
	if err != nil {
		t.Fatalf("labelCandidates: %v", err)
	}
	if got == nil {
		t.Fatal("labelCandidates returned nil; the payload would carry null instead of []")
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}
