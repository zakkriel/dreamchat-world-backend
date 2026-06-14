package main

import (
	"context"
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
