package main

import (
	"context"
	"fmt"
)

// Living World / Task 9 (Unit 6) — the world's-turn composer. Everything the earlier tasks built now
// composes into ONE reusable per-slot unit, called after each committed attempt's clock advance
// (orchestrator.go's runChain, both the passthrough and adjudicated Stage-4 blocks): it fires due
// scheduled events (fireDuePending, Task 4/ledger.go — deterministic, pre-caused world truth) FIRST,
// then — only if nothing medium/large already fired — rolls each pressure tier large→medium→small
// (rollTier, Task 6/pressure.go) against the derived chance (fn_pressure_chance, Task 5), and on the
// FIRST tier that fires in that scan order — which is therefore the BIGGEST magnitude that fired,
// per the founder-approved design (Unit 6: "returns the biggest magnitude that fired") — calls the
// World Actor (runWorldActor, Task 8/worldactor.go) to author ONE intrusion sized to that tier and
// records the fire in the append-only fire-log (world_eruption). The caller (runChain) reads the
// returned magnitude to apply the §5 cut: small (or nothing) lets the chain run on; medium/large ends
// the beat right there, discarding the rest of the chain — mechanical, not a judgment call, keyed only
// to the fired magnitude.
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
// seqUsed reports how many (tick,seq) slots this call ACTUALLY consumed at tickAfter — the ledger's own
// seqUsed (fireDuePending, Task 4/task-9-review) plus, if the roll also fires, the eruption commit's own
// seqUsed (commitWorldPayload's seqAdvance, surfaced through runWorldActor). This matters because a
// small-magnitude ledger fire does NOT skip the pressure roll (only medium/large do) — so a due pending
// row firing AND a tier rolling a fire in the SAME turn is an ordinary, reachable combination, not a
// hypothetical (task-9 review, Important #1). The roll's own commit is offset past whatever the ledger
// already wrote (seq+ledgerSeq, not the raw seq) so both commits land on distinct (tick,seq) pairs
// instead of colliding on the identical slot.
func (o *Orchestrator) runWorldTurn(ctx context.Context, worldID, scene string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, seqUsed int, err error) {
	// Task 10 (U7) — pure capture, guarded by trace != nil at every step (mirrors the GR trace
	// discipline: a non-debug beat pays ~zero extra cost, no extra queries, no extra allocations). This
	// snapshot lets us read back EXACTLY the ids fireDuePending itself commits below (outcome.Committed
	// already accumulates them — ledger.go's commitWorldPayload appends on every successful fire), with
	// no change to fireDuePending's own signature.
	var committedBeforeLedger int
	if trace != nil && outcome != nil {
		committedBeforeLedger = len(outcome.Committed)
	}

	// (a) Scheduled/deterministic ledger fires FIRST — pre-caused world truth due in this crossing.
	ledgerMag, ledgerSeq, err := o.fireDuePending(ctx, worldID, tickBefore, tickAfter, seq, outcome, trace)
	if err != nil {
		return "", 0, fmt.Errorf("runWorldTurn: fireDuePending: %w", err)
	}

	var firedScheduled []string
	if trace != nil && outcome != nil && len(outcome.Committed) > committedBeforeLedger {
		firedScheduled = append([]string(nil), outcome.Committed[committedBeforeLedger:]...)
	}

	if ledgerMag == "medium" || ledgerMag == "large" {
		// The beat is already ending on ledger-fired scheduled truth — SKIP the pressure roll entirely
		// (ambiguity resolution #2a): rolling a fresh eruption on top of a beat that's already over
		// would never be seen (the caller discards the rest of the chain on this magnitude). The trace
		// still records the clock delta + what the ledger fired; Rolls stays empty (nothing was rolled —
		// reporting zeros there would misrepresent a roll that never ran) and Eruption stays nil (no
		// pressure-roll eruption occurred).
		if trace != nil {
			trace.appendWorldTurn(TraceWorldTurn{ClockDeltaS: tickAfter - tickBefore, Fired: firedScheduled})
		}
		return ledgerMag, ledgerSeq, nil
	}

	// (b) Roll each tier large→medium→small; the FIRST tier that fires in this fixed scan order wins —
	// a single turn fires AT MOST ONE eruption (ambiguity resolution #2b; this is a fixed-order scan,
	// not a cross-tier "most significant chance" comparison). Scanning BIGGEST-first means the first
	// fire IS the biggest magnitude that fired (design Unit 6) — scanning small-first would let small's
	// much higher chance mask a rarer medium/large fire, silently suppressing the §5 beat-cut that is
	// the feature's whole point (whole-branch review, Fix 1). Its commit starts PAST every (tick,seq)
	// slot the ledger already used above, so it can never collide with a pending row fired this turn.
	//
	// Task 10 (U7) / ambiguity resolution #2: behavior is UNCHANGED — the first-fired tier in scan order
	// still acts, at most one eruption per turn. Non-debug (trace == nil) keeps the EXACT original
	// short-circuit: stop scanning the instant a tier fires, so zero extra queries run past the winner.
	// Debug (trace != nil) keeps scanning through all three tiers so every roll is captured for tuning
	// (Fork 6) — rollTier/lastEruptionTick are both read-only (no world_eruption write happens until the
	// winner is acted on below), so rolling the remaining tiers has no side effect on what fires.
	nextSeq := seq + ledgerSeq
	var rolls []TraceRoll
	firedTier := ""
	for _, tier := range livingWorldTierOrder {
		lastEruption, lastErr := o.lastEruptionTick(ctx, worldID, tier)
		if lastErr != nil {
			return "", 0, fmt.Errorf("runWorldTurn: lastEruptionTick(%s): %w", tier, lastErr)
		}
		fired, chance, roll, rollErr := o.rollTier(ctx, worldID, tier, tickAfter, lastEruption)
		if rollErr != nil {
			return "", 0, fmt.Errorf("runWorldTurn: rollTier(%s): %w", tier, rollErr)
		}
		if trace != nil {
			rolls = append(rolls, TraceRoll{Tier: tier, Chance: chance, Roll: roll, Fired: fired})
		}
		if !fired {
			continue
		}
		if firedTier == "" {
			firedTier = tier
		}
		if trace == nil {
			// Non-debug: stop at the first fire — identical to the pre-Task-10 code, zero extra cost.
			break
		}
		// Debug: keep scanning so every remaining tier's roll is captured too; firedTier above already
		// pinned to the FIRST tier that fired, so which tier acts below is unaffected.
	}

	if firedTier == "" {
		// Nothing in the roll fired — whatever the ledger itself consumed (possibly 0, possibly nonzero
		// even on a "" magnitude — e.g. a due row that gate-rejected still consumes its slot) is the
		// total.
		if trace != nil {
			trace.appendWorldTurn(TraceWorldTurn{ClockDeltaS: tickAfter - tickBefore, Fired: firedScheduled, Rolls: rolls})
		}
		return ledgerMag, ledgerSeq, nil
	}

	eventID, actorSeq, actorErr := o.runWorldActor(ctx, worldID, scene, firedTier, tickAfter, nextSeq, outcome, trace)
	if actorErr != nil {
		return "", 0, fmt.Errorf("runWorldTurn: runWorldActor(%s): %w", firedTier, actorErr)
	}

	// TODO(atomicity follow-up): eruption commit + this fire-log write are not yet in one tx;
	// deferred to a dedicated atomicity task (see ledger progress).
	if _, execErr := o.DB.Exec(ctx,
		`INSERT INTO world_eruption (world_id, tier, fired_tick, event_id) VALUES ($1, $2, $3, $4)`,
		worldID, firedTier, tickAfter, eventID); execErr != nil {
		return "", 0, fmt.Errorf("runWorldTurn: insert world_eruption(%s): %w", firedTier, execErr)
	}

	if trace != nil {
		trace.appendWorldTurn(TraceWorldTurn{
			ClockDeltaS: tickAfter - tickBefore,
			Fired:       firedScheduled,
			Rolls:       rolls,
			Eruption:    &TraceElement{Type: firedTier, IDs: []string{eventID}},
		})
	}
	return firedTier, ledgerSeq + actorSeq, nil
}

// livingWorldTierOrder is the fixed order the composer rolls: large→medium→small — BIGGEST first, so
// the first tier that fires in this scan is always the biggest magnitude that fired (design Unit 6:
// "returns the biggest magnitude that fired"). Scanning small-first would let small's much higher
// climb rate mask a rarer medium/large fire (whole-branch review, Fix 1).
var livingWorldTierOrder = []string{"large", "medium", "small"}

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
