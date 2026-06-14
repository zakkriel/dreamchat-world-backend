package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ResolveViewer returns the perception viewer for a request. The viewer is the epistemic
// boundary and is therefore resolved SERVER-SIDE (D-7/B-1): the client never picks whose
// truth is rendered in play mode. A debug override is honored ONLY when debug==true; it
// swaps the resolved identity and is still run through the identical safety filter downstream
// (fn_actor_page) — it never bypasses the wall.
//
// 0A stub: the player-controlled actor is resolved as the world's actor named 'Player'.
// Auth/session is out of scope this chunk (Bridge §6 item 4); this is the documented stand-in.
func ResolveViewer(ctx context.Context, pool *pgxpool.Pool, worldID, debugOverride string, debug bool) (string, error) {
	if debug && debugOverride != "" {
		return debugOverride, nil
	}
	var id string
	err := pool.QueryRow(ctx,
		`SELECT entity_id::text FROM entity_registry
		 WHERE world_id = $1 AND entity_kind = 'actor' AND canonical_name = 'Player' LIMIT 1`,
		worldID).Scan(&id)
	return id, err
}
