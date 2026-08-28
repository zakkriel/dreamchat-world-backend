package main

// Governed-by: D-1 — the AI proposes and the engine decides. This records what the proposal step
// actually produced, including when it produced nothing, so "which action types are missing?" is a
// query instead of an argument. Also B-1 — it is not a player surface and holds nothing the viewer
// could not already perceive.
// Derived 2026-08-28 from SPEC-037 and the founder ruling that a "cannot do that" vocabulary shape is
// "at best a try/catch, at worse a plaster that hides a need" — count first, then decide.

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// derivationRetention is how long a parse record lives. Founder ruling, 2026-08-28: 15 days. This is
// operational telemetry, so it is measured in wall-clock, not in-world ticks (B-5).
const derivationRetention = "15 days"

// derivationSweepInterval is how often expired rows are removed. Six hours is arbitrary and cheap: the
// window is fifteen days, so nothing depends on the sweep being prompt.
const derivationSweepInterval = 6 * time.Hour

// recordBeatDerivation writes one row per beat: the player's sentence and what the decompose stage
// made of it. It is BEST-EFFORT — every failure is logged and swallowed, because a telemetry write
// must never be able to cost a player their turn.
//
// It is called for every beat, and deliberately writes a row when the chain is EMPTY. That row is the
// entire point: an unparsed sentence leaves no other trace anywhere in the system. transcript_entry
// does not hold it (that table writes nothing when a beat produced no prose), and BeatTrace is
// debug-only and never persisted — and builds its element list FROM the chain, so an empty chain
// leaves an empty trace even in debug.
//
// Only the truth-free part of the trace shape is stored: the element type, the player's own words,
// and the ids the parse bound from the CANDIDATES list, which is already perception-bound to this
// viewer (B-1). The referee's reasoning and the truth-side fact sheets — the reason BeatTrace as a
// whole is debug-gated — are not here and must never be added.
func recordBeatDerivation(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string,
	tick int64, stated string, chain []Attempt) {

	if pool == nil {
		return
	}

	// The same element shape the debug trace already builds — one home for "what a decoded chain
	// element looks like" (D-6), minus every truth-side field.
	elements := make([]TraceElement, 0, len(chain))
	for _, a := range chain {
		elements = append(elements, TraceElement{Type: a.Type, Stated: a.Stated, IDs: attemptBoundIDs(a)})
	}
	payload, err := json.Marshal(elements)
	if err != nil {
		log.Printf("derivation: marshal elements: %v", err)
		return
	}

	// A NULL `stated` keeps "the player sent no text at all" distinguishable from "the player sent an
	// empty string", exactly as transcript_entry does. Every beat carries a sentence since the
	// continue press was deleted (2026-08-28), so this is now a malformed-client case rather than a
	// routine one — which is precisely why it should still be visible in the data rather than
	// flattened into "".
	var statedArg any
	if stated != "" {
		statedArg = stated
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO beat_derivation (world_id, viewer_id, in_world_tick, stated, elements)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5::jsonb)`,
		worldID, viewerID, tick, statedArg, string(payload)); err != nil {
		log.Printf("derivation: write failed for world %s viewer %s: %v", worldID, viewerID, err)
	}
}

// runDerivationRetention deletes parse records past the retention window, forever, in the background.
// It sweeps once at boot and then on an interval — sweeping first rather than sleeping first, so a
// service that restarts more often than the interval still expires its rows.
func runDerivationRetention(pool *pgxpool.Pool) {
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			tag, err := pool.Exec(ctx,
				`DELETE FROM beat_derivation WHERE recorded_at < now() - $1::interval`, derivationRetention)
			cancel()
			if err != nil {
				log.Printf("derivation: retention sweep: %v", err)
			} else if n := tag.RowsAffected(); n > 0 {
				log.Printf("derivation: retention sweep removed %d row(s) older than %s", n, derivationRetention)
			}
			time.Sleep(derivationSweepInterval)
		}
	}()
}
