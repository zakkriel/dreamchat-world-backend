package main

import (
	"context"
	"fmt"
)

// Living World / Task 9 (Unit 6) — the world's-turn composer. Everything the earlier tasks built now
// composes into ONE reusable per-slot unit, called after each committed attempt's clock advance
// (orchestrator.go's runChain, both the passthrough and adjudicated Stage-4 blocks): it fires due
// scheduled events (fireDuePending, Task 4/ledger.go — deterministic, pre-caused world truth) FIRST,
// then — only if nothing medium/large already fired — rolls each pressure tier small→medium→large
// (rollTier, Task 6/pressure.go) against the derived chance (fn_pressure_chance, Task 5), and on the
// most-significant tier that fires, calls the World Actor (runWorldActor, Task 8/worldactor.go) to
// author ONE intrusion sized to that tier and records the fire in the append-only fire-log
// (world_eruption). The caller (runChain) reads the returned magnitude to apply the §5 cut: small (or
// nothing) lets the chain run on; medium/large ends the beat right there, discarding the rest of the
// chain — mechanical, not a judgment call, keyed only to the fired magnitude.
//
// This SAME function is the reusable unit the Journey will later call per leg (design doc, Unit 6) — it
// carries NO progress/threshold/"until" logic of its own; that boundary (Station-G/Journey) is not
// crossed here.
//
// ATOMICITY IS DEFERRED (task-9-brief ambiguity resolution #5): the world_eruption insert below is a
// plain o.DB.Exec, not yet in the same tx as the eruption commit it records — see the matching TODO at
// the insert site below, and ledger.go's own TODO(Task 9) on fireDuePending's row commit + status flip.
// Both are flagged for a dedicated whole-branch atomicity follow-up; no tx refactor happens in this task.
//
// seqUsed reports how many (tick,seq) slots this call consumed — 1 if anything fired this turn (either
// the ledger or the roll; a turn fires at most one thing that matters to the caller), 0 otherwise — so
// runChain can thread it into curSeq and the next chain step never collides with what this turn wrote
// at tickAfter. NOTE: fireDuePending can itself fire SEVERAL due rows inside one crossing window, each
// consuming its own seq slot internally, but its signature (fixed by Task 4) reports back only the
// largest magnitude fired, not a count — so a turn where the ledger fires more than one row AND the
// roll also fires in the very same turn is a known, untested edge (seqUsed would undercount). The
// ambiguity resolution this task follows passes runWorldActor the plain `seq` this function received
// (not a ledger-adjusted one), matching that same simplification — see task-9-brief ambiguity
// resolution #2. Not exercised by this task's tests; flagged here for the atomicity follow-up above.
func (o *Orchestrator) runWorldTurn(ctx context.Context, worldID, scene string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, seqUsed int, err error) {
	// (a) Scheduled/deterministic ledger fires FIRST — pre-caused world truth due in this crossing.
	ledgerMag, err := o.fireDuePending(ctx, worldID, tickBefore, tickAfter, seq, outcome, trace)
	if err != nil {
		return "", 0, fmt.Errorf("runWorldTurn: fireDuePending: %w", err)
	}
	if ledgerMag == "medium" || ledgerMag == "large" {
		// The beat is already ending on ledger-fired scheduled truth — SKIP the pressure roll entirely
		// (ambiguity resolution #2a): rolling a fresh eruption on top of a beat that's already over
		// would never be seen (the caller discards the rest of the chain on this magnitude).
		return ledgerMag, 1, nil
	}

	// (b) Roll each tier small→medium→large; the MOST-significant tier that fires wins — a single turn
	// fires AT MOST ONE eruption (ambiguity resolution #2b).
	for _, tier := range livingWorldTierOrder {
		lastEruption, lastErr := o.lastEruptionTick(ctx, worldID, tier)
		if lastErr != nil {
			return "", 0, fmt.Errorf("runWorldTurn: lastEruptionTick(%s): %w", tier, lastErr)
		}
		fired, _, _, rollErr := o.rollTier(ctx, worldID, tier, tickAfter, lastEruption)
		if rollErr != nil {
			return "", 0, fmt.Errorf("runWorldTurn: rollTier(%s): %w", tier, rollErr)
		}
		if !fired {
			continue
		}

		eventID, actorErr := o.runWorldActor(ctx, worldID, scene, tier, tickAfter, seq, outcome, trace)
		if actorErr != nil {
			return "", 0, fmt.Errorf("runWorldTurn: runWorldActor(%s): %w", tier, actorErr)
		}

		// TODO(atomicity follow-up): eruption commit + this fire-log write are not yet in one tx;
		// deferred to a dedicated atomicity task (see ledger progress).
		if _, execErr := o.DB.Exec(ctx,
			`INSERT INTO world_eruption (world_id, tier, fired_tick, event_id) VALUES ($1, $2, $3, $4)`,
			worldID, tier, tickAfter, eventID); execErr != nil {
			return "", 0, fmt.Errorf("runWorldTurn: insert world_eruption(%s): %w", tier, execErr)
		}
		return tier, 1, nil
	}

	// Nothing in the roll fired. A "small" ledger fire (the only non-empty magnitude that can reach
	// here — medium/large already returned above) still consumed one (tick,seq) slot.
	if ledgerMag != "" {
		return ledgerMag, 1, nil
	}
	return "", 0, nil
}

// livingWorldTierOrder is the fixed rank order the composer rolls: small→medium→large — mirrors
// ledger.go's magnitudeRank ordering.
var livingWorldTierOrder = []string{"small", "medium", "large"}

// lastEruptionTick returns the last tick this tier fired for worldID — COALESCE(max(fired_tick), 0)
// from the append-only world_eruption fire-log — 0 if the tier has never fired. This MUST be the same
// value fn_pressure_chance derives internally for (worldID, tier, now) (Task 6's rollTier docstring),
// so rollTier's chance and its deterministic draw-seed stay paired to the same "time since this tier
// last fired" story fn_pressure_chance itself tells.
func (o *Orchestrator) lastEruptionTick(ctx context.Context, worldID, tier string) (int64, error) {
	var tick int64
	if err := o.DB.QueryRow(ctx,
		`SELECT COALESCE(max(fired_tick), 0) FROM world_eruption WHERE world_id=$1 AND tier=$2`,
		worldID, tier).Scan(&tick); err != nil {
		return 0, fmt.Errorf("lastEruptionTick: %w", err)
	}
	return tick, nil
}
