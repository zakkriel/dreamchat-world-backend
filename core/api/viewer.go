package main

import (
	"context"
	"errors"

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
// A world with no player yet (created through POST /worlds, before anything is seeded into it) is a
// real state, not a broken row — it returns a viewer-less error the handler renders as 404 rather
// than 500, because "nobody plays this world yet" is an answer.
func ResolveViewer(ctx context.Context, pool *pgxpool.Pool, worldID, debugOverride string, debug bool) (string, error) {
	if debug && debugOverride != "" {
		return debugOverride, nil
	}
	var id *string
	if err := pool.QueryRow(ctx,
		`SELECT player_entity_id::text FROM world WHERE world_id = $1`, worldID).Scan(&id); err != nil {
		return "", err
	}
	if id == nil {
		return "", errNoPlayerInWorld
	}
	return *id, nil
}

// errNoPlayerInWorld distinguishes "this world has nobody to play as" from "the lookup failed". The
// first is an ordinary answer about a world that exists but is not yet inhabited; the second is a
// broken database. Conflating them is how a normal state becomes a 500.
var errNoPlayerInWorld = errors.New("world has no player_entity_id")
