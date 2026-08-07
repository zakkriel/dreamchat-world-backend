package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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
// ATOMICITY (rung 0, deferral A — the deferral task-9-brief ambiguity resolution #5 recorded is closed):
// the world_eruption insert below is no longer a standalone o.DB.Exec. It is handed down as a
// postCommitFn (ledger.go) so it runs INSIDE the eruption commit's own transaction — the pair lands
// together or not at all, as it already does for fireDuePending's pending-row flip.
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

	if eruptionCutsBeat(ledgerMag) {
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
	for i := len(livingWorldTiers) - 1; i >= 0; i-- { // reverse = large→medium→small (biggest first)
		tier := livingWorldTiers[i]
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

	// The fire-log row IS this eruption's bookkeeping: it rides the eruption's own tx, so the pair can
	// never split. A committed eruption with no fire-log row would leave the tier's derived pressure
	// (fn_pressure_chance reads max(fired_tick)) permanently undrained — the whole-branch review's
	// "lost drain". Ownership is unchanged: the composer still decides what goes in the row.
	logEruption := func(ctx context.Context, tx pgx.Tx, eventIDs []string) error {
		if len(eventIDs) == 0 {
			return fmt.Errorf("eruption committed no event id")
		}
		_, execErr := tx.Exec(ctx,
			`INSERT INTO world_eruption (world_id, tier, fired_tick, event_id) VALUES ($1, $2, $3, $4)`,
			worldID, firedTier, tickAfter, eventIDs[0])
		return execErr
	}
	eventID, actorSeq, actorErr := o.runWorldActor(ctx, worldID, scene, firedTier, tickAfter, nextSeq, logEruption, outcome, trace)
	if actorErr != nil {
		return "", 0, fmt.Errorf("runWorldTurn: runWorldActor(%s): %w", firedTier, actorErr)
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

// livingWorldTiers is the SINGLE SOURCE OF TRUTH for the world's three magnitude tiers, ascending
// (small < medium < large). Everything tier-ordered derives from it: magnitudeRank (rank by index),
// eruptionCutsBeat (§5: medium/large cut the beat), and the composer's roll scan (BIGGEST-first = this
// slice in REVERSE — design Unit 6 "returns the biggest magnitude that fired"; scanning small-first
// would let small's much higher climb rate mask a rarer medium/large fire and suppress the §5 beat-cut,
// whole-branch review Fix 1). A future tier is added HERE (and to the SQL CHECK constraints) — nothing
// downstream hand-encodes the order or the cut threshold.
var livingWorldTiers = []string{"small", "medium", "large"}

// magnitudeRank maps a magnitude to its 1-based rank (small=1 … large=3), derived from livingWorldTiers,
// so fireDuePending can track the LARGEST magnitude fired across a crossing. Unranked/empty ("") is 0,
// below every real magnitude.
var magnitudeRank = func() map[string]int {
	m := make(map[string]int, len(livingWorldTiers))
	for i, t := range livingWorldTiers {
		m[t] = i + 1
	}
	return m
}()

// eruptionCutsBeat reports whether an eruption of this magnitude ends the beat (§5): medium/large cut,
// small runs on. Keyed off the rank of "medium", so a new tier inserted in livingWorldTiers needs no
// edit here.
func eruptionCutsBeat(mag string) bool {
	return magnitudeRank[mag] >= magnitudeRank["medium"]
}

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
