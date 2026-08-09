package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResolveViewer returns the perception viewer for a request. The viewer is the epistemic
// boundary and is therefore resolved SERVER-SIDE (D-7/B-1): the client never picks whose
// truth is rendered in play mode. A debug override is honored ONLY when debug==true; it
// swaps the resolved identity and is still run through the identical safety filter downstream
// (fn_actor_page) — it never bypasses the wall.
//
// The world now states its own player (world.player_entity_id, SPEC-028). This replaces the 0A stub
// that resolved "the actor whose canonical_name is 'Player'" — a naming convention that could not
// survive a second world and had already broken in the first one that mattered: the seeded play
// world's player is named Kade, so every non-debug request against it resolved nothing and 500'd at
// the door. Identity is a fact the world records, not a name the engine pattern-matches.
//
// STILL NOT AUTH. This answers "who does a caller play as in this world", never "who is calling" —
// the session model (B1) remains absent, so any caller is still every player. What changed is that
// the answer is now a lookup with somewhere for auth to attach: when B1 lands it decides WHICH world
// a caller may enter, and this function keeps working unchanged.
//
// TWO ANSWERS AND ONE BREAKAGE. A world that does not exist and a world nobody plays yet are both
// real states of the question "who do I play as here" — the answer is "nobody" — and both are 404.
// Only a failed lookup is a 500. Before this the first case fell through pgx.ErrNoRows into the
// generic error branch, so every world-scoped endpoint answered a typo'd or deleted world id with
// `500 viewer resolution failed`: a broken-server signal for an ordinary client mistake, which is
// exactly the conflation the errNoPlayerInWorld comment was already written to prevent for the
// other half. Found by hand-driving /carrying; it was never specific to /carrying.
func ResolveViewer(ctx context.Context, pool *pgxpool.Pool, worldID, debugOverride string, debug bool) (string, error) {
	if debug && debugOverride != "" {
		return debugOverride, nil
	}
	var id *string
	if err := pool.QueryRow(ctx,
		`SELECT player_entity_id::text FROM world WHERE world_id = $1`, worldID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errNoSuchWorld
		}
		return "", err
	}
	if id == nil {
		return "", errNoPlayerInWorld
	}
	return *id, nil
}

// The two "there is nobody to play as" answers. Both are 404 — an answer about a world, not a
// broken server — and they are deliberately distinct so the body can say which.
var (
	errNoSuchWorld     = errors.New("no such world")
	errNoPlayerInWorld = errors.New("this world has no player yet")
)

// writeNoViewer answers a ResolveViewer failure and reports whether it handled it. Every
// world-scoped handler asks the same question — is this an answer or a breakage? — so it lives
// here once instead of as the same eight-line branch copied into six files, which is how the
// unknown-world case came to be handled in none of them.
func writeNoViewer(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, errNoSuchWorld), errors.Is(err, errNoPlayerInWorld):
		http.Error(w, err.Error(), http.StatusNotFound)
	case err != nil:
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
	default:
		return false
	}
	return true
}
