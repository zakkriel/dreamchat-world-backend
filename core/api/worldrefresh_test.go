package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func worldRefreshPost(t *testing.T, h http.Handler, worldID string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/refresh", nil))
	return rec
}

func TestWorldRefresh_TemplatedWorldCreatesSuccessorAndArchivesSource(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	theme := worldTheme{SchemaVersion: "world_theme/1", Accent: "#5a7ca2", Mood: "ember", Ornament: "rivet"}
	themeJSON, err := json.Marshal(theme)
	if err != nil {
		t.Fatalf("marshal theme: %v", err)
	}
	sourceWorldID, err := createWorld(ctx, pool, "Refresh Source World", themeJSON)
	if err != nil {
		t.Fatalf("create source world: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, sourceWorldID) })
	if _, err := pool.Exec(ctx,
		`UPDATE world SET template_key='drowned_lantern' WHERE world_id=$1`, sourceWorldID,
	); err != nil {
		t.Fatalf("set source template_key: %v", err)
	}
	if _, err := pool.Exec(ctx, `SELECT fn_instantiate_drowned_lantern($1::uuid, NULL)`, sourceWorldID); err != nil {
		t.Fatalf("instantiate source template: %v", err)
	}

	var sourceCanonRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM location_state WHERE world_id=$1`, sourceWorldID).Scan(&sourceCanonRows); err != nil {
		t.Fatalf("source canon row count: %v", err)
	}
	if sourceCanonRows == 0 {
		t.Fatal("source world has zero canon rows in location_state; refresh test would be vacuous")
	}

	h := NewWorldRefreshHandler(pool, true)
	rec := worldRefreshPost(t, h, sourceWorldID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var got struct {
		SchemaVersion string     `json:"schema_version"`
		SourceWorldID string     `json:"source_world_id"`
		ID            string     `json:"id"`
		DisplayName   string     `json:"display_name"`
		Theme         worldTheme `json:"theme"`
		Playable      bool       `json:"playable"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, rec.Body.String())
	}
	if got.SchemaVersion != "world_refreshed/1" {
		t.Fatalf("schema_version = %q, want world_refreshed/1", got.SchemaVersion)
	}
	if got.SourceWorldID != sourceWorldID {
		t.Fatalf("source_world_id = %q, want %q", got.SourceWorldID, sourceWorldID)
	}
	if got.ID == "" || got.ID == sourceWorldID {
		t.Fatalf("new id = %q, want a non-empty id different from source", got.ID)
	}
	if !got.Playable {
		t.Fatal("refreshed world reported playable=false, want true")
	}

	newWorldID := got.ID
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, newWorldID)
	})

	var archivedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT archived_at FROM world WHERE world_id=$1`, sourceWorldID).Scan(&archivedAt); err != nil {
		t.Fatalf("source archive status: %v", err)
	}
	if archivedAt == nil {
		t.Fatal("source world archived_at is NULL after refresh")
	}

	var afterCanonRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM location_state WHERE world_id=$1`, sourceWorldID).Scan(&afterCanonRows); err != nil {
		t.Fatalf("source canon row count after refresh: %v", err)
	}
	if afterCanonRows != sourceCanonRows {
		t.Fatalf("source canon rows changed: before=%d after=%d", sourceCanonRows, afterCanonRows)
	}

	dirRec := worldsGet(t, NewWorldsHandler(pool, true))
	if dirRec.Code != http.StatusOK {
		t.Fatalf("directory status = %d, want 200: %s", dirRec.Code, dirRec.Body.String())
	}
	var directory struct {
		Worlds []struct {
			ID       string `json:"id"`
			Playable bool   `json:"playable"`
		} `json:"worlds"`
	}
	if err := json.Unmarshal(dirRec.Body.Bytes(), &directory); err != nil {
		t.Fatalf("decode directory: %v", err)
	}
	sourcePresent := false
	newPresent := false
	newPlayable := false
	for _, w := range directory.Worlds {
		switch w.ID {
		case sourceWorldID:
			sourcePresent = true
		case newWorldID:
			newPresent = true
			newPlayable = w.Playable
		}
	}
	if sourcePresent {
		t.Fatalf("source world %s is still present in directory after refresh", sourceWorldID)
	}
	if !newPresent {
		t.Fatalf("refreshed world %s missing from directory", newWorldID)
	}
	if !newPlayable {
		t.Fatalf("refreshed world %s is present but playable=false in directory", newWorldID)
	}
}

func TestWorldRefresh_NullTemplateKeyReturns404(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })

	theme := worldTheme{SchemaVersion: "world_theme/1", Accent: "#224466", Mood: "mist", Ornament: "vine"}
	themeJSON, err := json.Marshal(theme)
	if err != nil {
		t.Fatalf("marshal theme: %v", err)
	}
	sourceWorldID, err := createWorld(context.Background(), pool, "Refresh Null Template", themeJSON)
	if err != nil {
		t.Fatalf("create world: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, sourceWorldID) })

	rec := worldRefreshPost(t, NewWorldRefreshHandler(pool, true), sourceWorldID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) == "" {
		t.Fatal("empty error body")
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if got["error"] != "world has no template" {
		t.Fatalf("error = %q, want %q", got["error"], "world has no template")
	}
}
