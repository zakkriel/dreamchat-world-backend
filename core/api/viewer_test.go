package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const worldID = "11111111-1111-1111-1111-111111111111"
const playerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
const jonasID = "cccccccc-cccc-cccc-cccc-cccccccccccc"

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return pool
}

func TestResolveViewer_DefaultIsPlayer(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	got, err := ResolveViewer(context.Background(), pool, worldID, "", false)
	if err != nil {
		t.Fatalf("ResolveViewer: %v", err)
	}
	if got != playerID {
		t.Fatalf("default viewer = %s, want player %s", got, playerID)
	}
}

func TestResolveViewer_DebugOverrideHonoredOnlyInDebug(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	// override ignored when debug=false
	got, _ := ResolveViewer(context.Background(), pool, worldID, jonasID, false)
	if got != playerID {
		t.Fatalf("override leaked outside debug mode: got %s", got)
	}
	// override honored when debug=true
	got, _ = ResolveViewer(context.Background(), pool, worldID, jonasID, true)
	if got != jonasID {
		t.Fatalf("debug override not honored: got %s", got)
	}
}

// An id nobody ever minted is a CLIENT mistake, and "who do I play as in that world" has an
// answer: nobody. It used to fall through pgx.ErrNoRows into the generic branch, so every
// world-scoped endpoint reported a typo as `500 viewer resolution failed`. Found by hand-driving
// /carrying; asserted here at the shared source, and below at two unrelated endpoints so a
// regression cannot hide behind the one route that noticed it.
func TestResolveViewer_UnknownWorldIsAnAnswerNotABreakage(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	_, err := ResolveViewer(context.Background(), pool, fabricatedID, "", false)
	if !errors.Is(err, errNoSuchWorld) {
		t.Fatalf("unknown world err = %v, want errNoSuchWorld", err)
	}
}

func TestUnknownWorld404sOnEveryWorldScopedEndpoint(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	for name, h := range map[string]http.Handler{
		"carrying": NewCarryingHandler(pool, false),
		"timeline": NewTimelineHandler(pool, false),
		"index":    NewIndexHandler(pool, false, "actors", "actor"),
	} {
		path := map[string]string{
			"carrying": "/worlds/" + fabricatedID + "/carrying",
			"timeline": "/worlds/" + fabricatedID + "/compendium/timeline",
			"index":    "/worlds/" + fabricatedID + "/compendium/actors",
		}[name]
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s on an unknown world = %d, want 404 (a typo is not a broken server)", name, rec.Code)
		}
	}
}
