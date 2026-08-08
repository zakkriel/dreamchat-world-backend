package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func worldsGet(t *testing.T, h http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds", nil))
	return rec
}

func worldsPost(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/worlds", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	return rec
}

// SPEC-028. Nothing could answer "which worlds are there" — world_id was a bare uuid on
// twenty-seven tables, so the frontend could route on a world but never let anyone choose one.
func TestWorlds_DirectoryListsTheSeededWorldsWithTheirLook(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()

	rec := worldsGet(t, NewWorldsHandler(pool, true))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		SchemaVersion string `json:"schema_version"`
		Worlds        []struct {
			ID          string     `json:"id"`
			DisplayName string     `json:"display_name"`
			Theme       worldTheme `json:"theme"`
			Playable    bool       `json:"playable"`
		} `json:"worlds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, rec.Body.String())
	}
	if got.SchemaVersion != "world_directory/1" {
		t.Fatalf("schema_version = %q, want world_directory/1", got.SchemaVersion)
	}

	var lantern *struct {
		ID          string     `json:"id"`
		DisplayName string     `json:"display_name"`
		Theme       worldTheme `json:"theme"`
		Playable    bool       `json:"playable"`
	}
	for i := range got.Worlds {
		if got.Worlds[i].ID == "22222222-2222-2222-2222-222222222222" {
			lantern = &got.Worlds[i]
		}
	}
	if lantern == nil {
		t.Fatalf("the seeded play world is missing from the directory: %s", rec.Body.String())
	}
	if lantern.DisplayName != "The Drowned Lantern" {
		t.Fatalf("display_name = %q", lantern.DisplayName)
	}
	if lantern.Theme.SchemaVersion != "world_theme/1" || lantern.Theme.Accent != "#c9a227" {
		t.Fatalf("theme = %+v, want world_theme/1 with the tavern's accent", lantern.Theme)
	}
	// It has a player (Kade), so it is enterable.
	if !lantern.Playable {
		t.Fatal("the seeded play world reports playable=false — the viewer seam is not wired")
	}
}

// A directory is not canon (SPEC-028: "a world list is a directory, not canon"). Nothing about what
// is INSIDE a world may ride this surface — no scene, no tick, no entities, no counts.
func TestWorlds_DirectoryCarriesNoWorldState(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()

	// Checked STRUCTURALLY, on the key set, not by scanning values for suspicious words: a world may
	// legitimately be NAMED after a character ("Mara 0A Fixture"), and a substring scan would call
	// that a leak. What must never appear is a FIELD carrying world state.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(worldsGet(t, NewWorldsHandler(pool, true)).Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	topLevel := map[string]bool{"schema_version": true, "worlds": true}
	for k := range raw {
		if !topLevel[k] {
			t.Fatalf("unexpected top-level field %q on the world directory", k)
		}
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw["worlds"], &entries); err != nil {
		t.Fatalf("decode worlds: %v", err)
	}
	allowed := map[string]bool{"id": true, "display_name": true, "theme": true, "playable": true}
	for _, e := range entries {
		for k := range e {
			if !allowed[k] {
				t.Fatalf("world entry carries %q — a directory is not canon; no world state may ride this surface", k)
			}
		}
	}
	if len(entries) == 0 {
		t.Fatal("no worlds in the directory — the assertion above would pass vacuously")
	}
}

// Creation is a real endpoint, per the founder's ruling, and it is honest about what it produces: a
// world that exists, is listed, and is NOT yet playable, because nothing has been authored into it.
func TestWorlds_CreateMakesARealButUninhabitedWorld(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	h := NewWorldsHandler(pool, true)

	rec := worldsPost(t, h, `{"display_name":"Test World — Create","theme":{"schema_version":"world_theme/1","accent":"#3b6ea5","mood":"bleak","ornament":"rivet"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		SchemaVersion string     `json:"schema_version"`
		ID            string     `json:"id"`
		DisplayName   string     `json:"display_name"`
		Theme         worldTheme `json:"theme"`
		Playable      bool       `json:"playable"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, got.ID) })

	if got.SchemaVersion != "world_created/1" || got.ID == "" {
		t.Fatalf("payload = %+v, want world_created/1 with an id", got)
	}
	if got.Playable {
		t.Fatal("a world with nothing authored into it reported playable=true")
	}
	if got.Theme.Mood != "bleak" || got.Theme.Ornament != "rivet" {
		t.Fatalf("theme = %+v, want the caller's own tokens", got.Theme)
	}

	// The world's OPERATING DEFAULTS landed with it. A directory row without movement speeds or
	// duration classes is a world every later call fails against, so the pair must be atomic.
	var speeds int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM movement_type WHERE world_id=$1`, got.ID).Scan(&speeds); err != nil {
		t.Fatalf("movement_type: %v", err)
	}
	if speeds == 0 {
		t.Fatal("created world has no movement types — seed_world_defaults did not run with it")
	}

	// And it is immediately in the directory.
	if !strings.Contains(worldsGet(t, h).Body.String(), got.ID) {
		t.Fatal("the created world is not in the directory")
	}
}

// Unknown theme vocabulary must DEGRADE, never fail. A world authored against a newer mood word has
// to keep working; a backend that 400s on a word it has not heard of is the thing that breaks.
func TestWorlds_CreateAcceptsUnknownThemeVocabulary(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()

	rec := worldsPost(t, NewWorldsHandler(pool, true),
		`{"display_name":"Test World — Future Mood","theme":{"schema_version":"world_theme/1","accent":"#112233","mood":"thunderhead","ornament":"lattice"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 — unknown mood/ornament must degrade in the client, not fail here: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		ID    string     `json:"id"`
		Theme worldTheme `json:"theme"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, got.ID) })
	if got.Theme.Mood != "thunderhead" {
		t.Fatalf("mood = %q, want it stored verbatim", got.Theme.Mood)
	}
}

// The floors that ARE enforced return a reason, not a constraint violation surfacing as a 500.
func TestWorlds_CreateRejectsWhatStorageWouldReject(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewWorldsHandler(pool, true)

	for _, tc := range []struct{ name, body string }{
		{"no name", `{"display_name":"   "}`},
		{"blank name", `{}`},
		{"bad accent", `{"display_name":"X","theme":{"schema_version":"world_theme/1","accent":"gold","mood":"ember","ornament":"vine"}}`},
		{"wrong theme version", `{"display_name":"X","theme":{"schema_version":"world_theme/2","accent":"#112233","mood":"ember","ornament":"vine"}}`},
	} {
		if rec := worldsPost(t, h, tc.body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400: %s", tc.name, rec.Code, rec.Body.String())
		}
	}
}

// TestGenWorldPayloads writes the REAL responses of both /worlds surfaces as schema-contract
// payloads. Gated on WORLD_PAYLOAD_DIR so it never runs in the ordinary suite (it creates a world);
// ci/gen_payloads.sh drives it, mirroring TestGenSceneCurrentPayloads.
func TestGenWorldPayloads(t *testing.T) {
	dir := os.Getenv("WORLD_PAYLOAD_DIR")
	if dir == "" {
		t.Skip("WORLD_PAYLOAD_DIR unset — this generator only runs from ci/gen_payloads.sh")
	}
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	h := NewWorldsHandler(pool, true)

	rec := worldsGet(t, h)
	if rec.Code != http.StatusOK {
		t.Fatalf("directory: status = %d: %s", rec.Code, rec.Body.String())
	}
	if err := os.WriteFile(filepath.Join(dir, "world_directory_1.json"), rec.Body.Bytes(), 0o644); err != nil {
		t.Fatalf("write directory payload: %v", err)
	}

	created := worldsPost(t, h, `{"display_name":"Payload Fixture World","theme":{"schema_version":"world_theme/1","accent":"#3b6ea5","mood":"bleak","ornament":"rivet"}}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: status = %d: %s", created.Code, created.Body.String())
	}
	if err := os.WriteFile(filepath.Join(dir, "world_created_1.json"), created.Body.Bytes(), 0o644); err != nil {
		t.Fatalf("write created payload: %v", err)
	}
	// Leave the seeded db as we found it: the directory payload above must keep listing exactly the
	// seeded worlds on the next run, not accumulate one fixture per CI invocation.
	var got struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(created.Body.Bytes(), &got)
	if got.ID != "" {
		if _, err := pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, got.ID); err != nil {
			t.Fatalf("cleanup created world: %v", err)
		}
	}
}
