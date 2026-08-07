package main

// GET /worlds/{w}/scene/current (design §4.8, mvp_slice_and_bridge.md §4.1, rung3 Task 1) — the first
// read side of the BE⇄FE contract. Each correctness test mints its own fresh, random world (the
// station-F/query_test.go convention: hermetic, re-runnable with no DB reset, and immune to whatever
// the seed's noise loop happens to leave actors standing on).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sceneTestIDs holds the freshly-minted ids for one scene-world invocation.
type sceneTestIDs struct{ World, Viewer, Companion, Stranger, Place, ElsewherePlace, Prop string }

// setupSceneWorld mints a fresh world: Viewer and Companion co-located at Place (a tavern); a Prop
// (artifact) also co-located at Place — present, but never a participant (UX doctrine §2.2, "never
// objects, locations, or factions as participants"); a Stranger at a wholly different, unconnected
// ElsewherePlace, so the Viewer cannot perceive them at all (the naming-reach wall, RULINGS-2026-07-23
// §3, doubling as this endpoint's own wall test, B-1/I-3).
//
// Place and Companion each carry a CANONICAL name distinct from what the Viewer actually knows them
// by (a world_genesis name perception for Place; an actor_state descriptor for Companion) — proving
// scene/current ships the viewer's OWN naming, never the canonical registry name (§3, D-7), which a
// test that only checked "a label is present" could not.
func setupSceneWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool) sceneTestIDs {
	t.Helper()
	var id sceneTestIDs
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.Viewer, &id.Companion, &id.Stranger, &id.Place, &id.ElsewherePlace, &id.Prop); err != nil {
		t.Fatalf("mint scene ids: %v", err)
	}

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("setupSceneWorld: %v\nsql: %s", err, sql)
		}
	}

	mustExec(`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		($2,$1,'actor',   'Player'),
		($3,$1,'actor',   'COMPANION-CANONICAL-SECRET'),
		($4,$1,'actor',   'Stranger'),
		($5,$1,'location','PLACE-CANONICAL-SECRET'),
		($6,$1,'location','Elsewhere'),
		($7,$1,'artifact','a weathered lantern')`,
		id.World, id.Viewer, id.Companion, id.Stranger, id.Place, id.ElsewherePlace, id.Prop)

	mustExec(`INSERT INTO location_state (entity_id, world_id, attrs) VALUES
		($2,$1, jsonb_build_object('description','A weathered tavern by the docks.','tone','tense')),
		($3,$1, '{}'::jsonb)`,
		id.World, id.Place, id.ElsewherePlace)

	mustExec(`INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		($2,$1, jsonb_build_object('location_id',$4::text)),
		($3,$1, jsonb_build_object('location_id',$4::text,'descriptor','a weary road companion')),
		($5,$1, jsonb_build_object('location_id',$6::text,'descriptor','a hooded stranger'))`,
		id.World, id.Viewer, id.Companion, id.Place, id.Stranger, id.ElsewherePlace)

	mustExec(`INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
		($2,$1, jsonb_build_object('location_id',$3::text))`,
		id.World, id.Prop, id.Place)

	// world_genesis name perception: the Viewer's OWN name-knowledge of Place, distinct from the
	// canonical registry name (fn_perceived_name/fn_display_name, schema.sql:1598/1067).
	mustExec(`WITH ev AS (
		  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
		                           status, accepted_at, visibility_scope, origin)
		  VALUES (gen_random_uuid(),$1,'world_genesis','name grant',1,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		), pr AS (
		  INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
		                                 acquired_tick, valid_tick)
		  SELECT $1, $2, event_id, 'The Rusty Anchor', 'public', 1, 1 FROM ev
		  RETURNING perception_id
		)
		INSERT INTO perception_subject (perception_id, entity_id, world_id)
		SELECT perception_id, $3, $1 FROM pr`,
		id.World, id.Viewer, id.Place)

	// The Viewer's own arrival perception (tick 5, in_world_label "the third bell") — the newest line,
	// so scene/current's `now.display_label` resolves to it (never wall-clock, B-5).
	mustExec(`WITH ev AS (
		  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
		                           in_world_label, status, accepted_at, visibility_scope, origin)
		  VALUES (gen_random_uuid(),$1,'move','V arrives',5,0,'the third bell','accepted',now(),'public','fast_path')
		  RETURNING event_id
		), pr AS (
		  INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
		                                 acquired_tick, valid_tick)
		  SELECT $1, $2, event_id, 'You arrive at the tavern.', 'direct', 5, 5 FROM ev
		  RETURNING perception_id
		)
		INSERT INTO perception_subject (perception_id, entity_id, world_id)
		SELECT perception_id, $2, $1 FROM pr`,
		id.World, id.Viewer)

	// A perception scoped to the Stranger alone — never held by the Viewer — proving the wall holds
	// for perception CONTENT too, not just co-location.
	mustExec(`WITH ev AS (
		  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
		                           status, accepted_at, visibility_scope, origin)
		  VALUES (gen_random_uuid(),$1,'move','stranger arrives',3,0,'accepted',now(),'private','fast_path')
		  RETURNING event_id
		), pr AS (
		  INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
		                                 acquired_tick, valid_tick, visibility_scope)
		  SELECT $1, $2, event_id, 'the hooded stranger slips into the fog, unseen', 'direct', 3, 3, 'private' FROM ev
		  RETURNING perception_id
		)
		INSERT INTO perception_subject (perception_id, entity_id, world_id)
		SELECT perception_id, $2, $1 FROM pr`,
		id.World, id.Stranger)

	return id
}

func sceneGet(t *testing.T, h http.Handler, world, viewer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/worlds/"+world+"/scene/current?viewer="+viewer, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestSceneCurrent_ShowsWhereYouAreAndWhoIsPresent pins the two hardest things (plan Task 1): the
// Viewer is NEVER their own participant, and participants are CHARACTERS ONLY (the co-located Prop,
// an artifact, must be absent) — plus the place/label/now/schema_version shape the plan specifies.
func TestSceneCurrent_ShowsWhereYouAreAndWhoIsPresent(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)

	h := NewSceneHandler(pool, true) // debug → honor ?viewer=
	rec := sceneGet(t, h, id.World, id.Viewer)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var scene sceneView
	if err := json.Unmarshal(rec.Body.Bytes(), &scene); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, rec.Body.String())
	}

	if scene.SchemaVersion != "scene_current/1" {
		t.Fatalf("schema_version = %q, want scene_current/1", scene.SchemaVersion)
	}

	// place: the Viewer's OWN name for it (world_genesis perception) — never the canonical secret.
	if scene.Place.ID != id.Place {
		t.Fatalf("place.id = %q, want %q", scene.Place.ID, id.Place)
	}
	if scene.Place.Label != "The Rusty Anchor" {
		t.Fatalf("place.label = %q, want the viewer's own name %q", scene.Place.Label, "The Rusty Anchor")
	}
	if scene.Place.Description == nil || *scene.Place.Description != "A weathered tavern by the docks." {
		t.Fatalf("place.description = %v, want the authored description", scene.Place.Description)
	}
	if scene.Place.Tone == nil || *scene.Place.Tone != "tense" {
		t.Fatalf("place.tone = %v, want %q", scene.Place.Tone, "tense")
	}

	// participants: exactly the Companion, labeled by descriptor (never the canonical secret name).
	if len(scene.Participants) != 1 {
		t.Fatalf("participants = %+v, want exactly the Companion", scene.Participants)
	}
	p := scene.Participants[0]
	if p.ID != id.Companion {
		t.Fatalf("participant.id = %q, want the Companion %q", p.ID, id.Companion)
	}
	if p.Label != "a weary road companion" {
		t.Fatalf("participant.label = %q, want the viewer's own name %q", p.Label, "a weary road companion")
	}
	if p.Kind != "actor" {
		t.Fatalf("participant.kind = %q, want actor", p.Kind)
	}
	// THE HARD ONE (1/2): the viewer is not a participant in their own scene.
	for _, pp := range scene.Participants {
		if pp.ID == id.Viewer {
			t.Fatalf("the viewer appears in their own participants list: %+v", scene.Participants)
		}
	}
	// THE HARD ONE (2/2): participants are characters only — the co-located Prop (an artifact) is absent.
	for _, pp := range scene.Participants {
		if pp.ID == id.Prop {
			t.Fatalf("a co-located ARTIFACT appears as a participant (UX doctrine §2.2): %+v", scene.Participants)
		}
	}

	// now: tick + display_label, never wall-clock (B-5).
	if scene.Now.Tick != 5 {
		t.Fatalf("now.tick = %d, want 5 (fn_world_now = max in_world_tick)", scene.Now.Tick)
	}
	if scene.Now.DisplayLabel == nil || *scene.Now.DisplayLabel != "the third bell" {
		t.Fatalf("now.display_label = %v, want %q", scene.Now.DisplayLabel, "the third bell")
	}

	// journey: null this rung (Task 2 fills it in).
	if scene.Journey != nil {
		t.Fatalf("journey = %v, want null (Task 2 not yet built)", scene.Journey)
	}

	// current: the viewer's own recent perceptions, prose the FE renders verbatim (D-7).
	found := false
	for _, line := range scene.Current {
		if line == "You arrive at the tavern." {
			found = true
		}
	}
	if !found {
		t.Fatalf("current = %v, missing the viewer's own arrival line", scene.Current)
	}

	// Naming wall, restated on the raw body: the canonical secrets never cross the boundary.
	body := rec.Body.String()
	if strings.Contains(body, "PLACE-CANONICAL-SECRET") {
		t.Fatalf("response leaked the place's canonical registry name: %s", body)
	}
	if strings.Contains(body, "COMPANION-CANONICAL-SECRET") {
		t.Fatalf("response leaked the companion's canonical registry name: %s", body)
	}
}

// TestSceneCurrent_LeaksNoCanon is the wall test (B-1/I-3): no event_id, no canon row, and nobody the
// viewer cannot perceive — the Stranger, standing at an unconnected place, must be wholly absent:
// not their id, not their descriptor, not the private perception scoped to them alone.
func TestSceneCurrent_LeaksNoCanon(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)

	h := NewSceneHandler(pool, true)
	rec := sceneGet(t, h, id.World, id.Viewer)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	if strings.Contains(body, "event_id") {
		t.Fatalf("response carries a raw event_id field (B-1): %s", body)
	}
	if strings.Contains(body, "canon_event") || strings.Contains(body, `"canon"`) {
		t.Fatalf("response carries a raw canon row (B-1): %s", body)
	}
	if strings.Contains(body, id.Stranger) {
		t.Fatalf("response leaked the Stranger's id — unperceivable (different, unconnected place): %s", body)
	}
	if strings.Contains(body, "hooded stranger") {
		t.Fatalf("response leaked the Stranger's descriptor: %s", body)
	}
	if strings.Contains(body, "slips into the fog") {
		t.Fatalf("response leaked a perception scoped to the Stranger alone: %s", body)
	}
}

// TestGenSceneCurrentPayloads is SPEC-011's Go-side payload generator: scene/current is assembled in
// Go (buildScene), not a SQL function, so ci/gen_payloads.sh cannot call it directly the way it calls
// fn_actor_page etc. Skipped in the normal suite; ci/gen_payloads.sh runs it explicitly (with
// SCENE_PAYLOAD_DIR set) against the seeded FIXTURE world (worldID/playerID/jonasID, viewer_test.go)
// so `make schema-contract` has a REAL payload — produced by the real ServeHTTP, the same call
// net/http itself would make — to validate against schema/scene_current.v1.schema.json.
func TestGenSceneCurrentPayloads(t *testing.T) {
	dir := os.Getenv("SCENE_PAYLOAD_DIR")
	if dir == "" {
		t.Skip("SCENE_PAYLOAD_DIR unset — this generator only runs from ci/gen_payloads.sh")
	}
	pool := testPool(t)
	defer pool.Close()
	h := NewSceneHandler(pool, true) // debug → honor ?viewer=

	write := func(name, viewer string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/worlds/"+worldID+"/scene/current?viewer="+viewer, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200: %s", name, rec.Code, rec.Body.String())
		}
		if err := os.WriteFile(filepath.Join(dir, name), rec.Body.Bytes(), 0o644); err != nil {
			t.Fatalf("%s: write: %v", name, err)
		}
	}
	write("scene_current_P.json", playerID)
	write("scene_current_J.json", jonasID)
}
