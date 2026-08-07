package main

import (
	"context"
	"fmt"
	"hash/fnv"
)

// Living World / Task 6 — the deterministic pressure roll. Task 5 built fn_pressure_chance (SQL): a
// per-tier fire chance in [0,1], derived from world-time elapsed since that tier's last eruption,
// never NULL. This file adds the Go side that decides whether a tier actually fires THIS slot.
// "Deterministic" is a hard requirement (replay must reproduce the identical result byte-for-byte),
// so the draw is a pure hash of committed state — worldID/tick/lastEruption/tier — NEVER math/rand
// or wall-clock. One responsibility per function: deterministicRoll (the pure draw), pressureChance
// (the SQL read), rollTier (compose the two into a fire/no-fire decision).

// deterministicRoll hashes worldID|tick|lastEruption|tier with fnv64a and maps the result into
// [0,1) by taking the top 53 bits of the 64-bit sum (float64's mantissa is 53 bits) and dividing by
// 2^53. Pure: no math/rand, no wall-clock — identical inputs ALWAYS produce the identical output
// (replay-safe), and changing any one of the four inputs (tick, lastEruption, tier, or worldID)
// changes the output. The '|' separators between fields keep e.g. worldID="a",tick=12 distinct from
// worldID="a1",tick=2 — an unseparated concatenation could collide across such boundaries.
func deterministicRoll(worldID string, tick, lastEruption int64, tier string) float64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%d|%d|%s", worldID, tick, lastEruption, tier)
	return float64(h.Sum64()>>11) / float64(uint64(1)<<53)
}

// pressureChance calls fn_pressure_chance (Task 5) — the derived, capped, per-tier fire chance in
// [0,1] for worldID/tier at now. fn_pressure_chance never returns NULL (an unconfigured world or a
// disabled world_actor_setting both COALESCE to 0), so scanning straight into a float64 is safe —
// no sql.NullFloat64 needed. Mirrors factSheetJSON/durationClassSeconds's established Go→SQL
// helper pattern (orchestrator.go): one QueryRow, one Scan, wrap the error with the fn name.
func (o *Orchestrator) pressureChance(ctx context.Context, worldID, tier string, now int64) (float64, error) {
	var chance float64
	if err := o.DB.QueryRow(ctx, `SELECT fn_pressure_chance($1,$2,$3)`, worldID, tier, now).Scan(&chance); err != nil {
		return 0, fmt.Errorf("fn_pressure_chance: %w", err)
	}
	return chance, nil
}

// rollTier decides whether tier fires THIS slot: chance is read from fn_pressure_chance (Task 5),
// roll is the deterministic draw for (worldID, now, lastEruption, tier), and fired = roll < chance.
// Both chance and roll are returned alongside fired (not just the boolean) so Task 10's trace can
// show the actual numbers behind the decision, not just the outcome.
//
// IMPORTANT: lastEruption is supplied BY THE CALLER (Task 9 queries max(fired_tick) FROM
// world_eruption WHERE world_id=worldID AND tier=tier before calling this) and MUST be the SAME
// last-eruption tick fn_pressure_chance computes internally for that same (worldID, tier, now).
// fn_pressure_chance derives its own "ticks since last eruption" from world_eruption directly, so if
// the caller's lastEruption disagrees with what's actually in world_eruption, the chance and the
// draw-seed fall out of sync: replay would still be byte-identical (both are still pure functions of
// their inputs), but the pairing would no longer reflect the same "time since this tier last fired"
// story that made the chance climb in the first place.
func (o *Orchestrator) rollTier(ctx context.Context, worldID, tier string, now, lastEruption int64) (fired bool, chance, roll float64, err error) {
	chance, err = o.pressureChance(ctx, worldID, tier, now)
	if err != nil {
		return false, 0, 0, err
	}
	roll = deterministicRoll(worldID, now, lastEruption, tier)
	fired = roll < chance
	return fired, chance, roll, nil
}
