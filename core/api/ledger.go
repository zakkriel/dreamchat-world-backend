package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// pendingPayload is the {"actor_id":..., "attempt":{...}} shape a pending_event.payload carries — the
// world entity acting, paired with its Attempt JSON. Task 4 establishes this shape cleanly (the
// brief's ambiguity resolution #1) so Task 8 (World Actor) can reuse it verbatim when it writes rows
// for fireDuePending to fire later.
type pendingPayload struct {
	ActorID string  `json:"actor_id"`
	Attempt Attempt `json:"attempt"`
}

// magnitudeRank orders pending_event.magnitude small < medium < large so fireDuePending can track the
// LARGEST magnitude fired across a crossing. Unranked/empty ("") sorts below every real magnitude.
var magnitudeRank = map[string]int{"small": 1, "medium": 2, "large": 3}

// fireDuePending fires every pending_event row for worldID whose fire_at_tick falls inside the
// clock-crossing window (tickBefore, tickAfter] — strict lower bound, inclusive upper (brief ambiguity
// resolution #3): a row exactly AT tickBefore already fired in a prior slot, a row exactly AT
// tickAfter fires now. Rows are processed in fire_at_tick order.
//
// A pending row is PRE-CAUSED WORLD TRUTH (ambiguity resolution #2) — it is not a fresh player/NPC
// proposal, so it runs NEITHER the world-first hook NOR the premise re-check runChain uses for a live
// attempt; it commits directly. Each payload unmarshals to {actor_id, attempt} and is routed by
// attempt.Type EXACTLY as runChain's Stage 3 routes a live attempt: the three passthrough types
// (ActorMoved, Communicated, ObjectRelocated) commit via applyEvent; everything else adjudicates as a
// single-actor set (mirroring the Stage-3 default branch). The commit path's own perception fan-out is
// what delivers the event to witnesses — nothing is synthesized here.
//
// Every row that fires this call commits at tickAfter (the tick the beat crossed into firing it — the
// row's own fire_at_tick is only ever used as the WHERE-clause cutoff, never as the commit tick), with
// curSeq starting at seq and advancing per attempted row so two rows processed in the same call never
// collide on (tick, seq).
//
// A row only counts as FIRED if its payload actually landed in canon — a gate-rejected passthrough
// (e.g. Communicated's listener walked off co-presence) or a bounced/gate-rejected adjudication (the
// referee couldn't produce a valid ruling, or one of its events failed the structural floor) never
// flips to 'fired' and never folds into the returned magnitude: Task 9's composer reads that magnitude
// to decide the §5 beat-cut, and cutting a beat for an event that isn't actually in canon would be a
// lie. A row that didn't land instead flips to the terminal 'cancelled' state (never retried, never
// silently re-attempted) with an observable log line — see review fix (Critical #1).
//
// Each fired row's committed id(s) are appended to outcome.Committed. The return value is the LARGEST
// magnitude ACTUALLY fired, ranked small<medium<large ("" if nothing fired this call) — the caller
// (Task 9's composer) decides the §5 beat-cut from that magnitude; this helper never applies the cut
// itself.
func (o *Orchestrator) fireDuePending(ctx context.Context, worldID string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, err error) {
	rows, err := o.DB.Query(ctx,
		`SELECT pending_id, magnitude, payload FROM pending_event
		 WHERE world_id=$1 AND status='pending' AND fire_at_tick > $2 AND fire_at_tick <= $3
		 ORDER BY fire_at_tick`,
		worldID, tickBefore, tickAfter)
	if err != nil {
		return "", fmt.Errorf("fireDuePending: query due rows: %w", err)
	}

	type dueRow struct {
		id        string
		magnitude string
		payload   []byte
	}
	var due []dueRow
	for rows.Next() {
		var d dueRow
		if scanErr := rows.Scan(&d.id, &d.magnitude, &d.payload); scanErr != nil {
			rows.Close()
			return "", fmt.Errorf("fireDuePending: scan due row: %w", scanErr)
		}
		due = append(due, d)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		rows.Close()
		return "", fmt.Errorf("fireDuePending: iterate due rows: %w", rowsErr)
	}
	rows.Close()

	curSeq := seq
	for _, d := range due {
		var pp pendingPayload
		if unmarshalErr := json.Unmarshal(d.payload, &pp); unmarshalErr != nil {
			return "", fmt.Errorf("fireDuePending: pending_event %s payload: %w", d.id, unmarshalErr)
		}

		// committedIDs + haltReason together decide whether this row actually landed in canon.
		// haltReason != "" OR committedIDs empty ⇒ nothing committed ⇒ this row cancels, not fires.
		var committedIDs []string
		var haltReason string

		switch pp.Attempt.Type {
		case "ActorMoved", "Communicated", "ObjectRelocated":
			// Passthrough — the same routing runChain's Stage 3 uses for these three types.
			attemptJSON, marshalErr := json.Marshal(pp.Attempt)
			if marshalErr != nil {
				return "", fmt.Errorf("fireDuePending: pending_event %s attempt marshal: %w", d.id, marshalErr)
			}
			result, applyErr := o.applyEvent(ctx, worldID, pp.ActorID, attemptJSON, tickAfter, curSeq)
			if applyErr != nil {
				return "", fmt.Errorf("fireDuePending: pending_event %s apply_event: %w", d.id, applyErr)
			}
			// Mirror runChain's own passthrough check (orchestrator.go Stage 3, ~line 251): halt_reason
			// is the authoritative signal from apply_event's structural floor, not just an empty
			// event_id. If it isn't gate_reject, fall through to the event_id check as a
			// belt-and-suspenders guard against a hypothetically malformed result.
			if hr, _ := result["halt_reason"].(string); hr == "gate_reject" {
				haltReason = "gate_reject"
			} else if evID, _ := result["event_id"].(string); evID != "" {
				committedIDs = []string{evID}
			}
			curSeq++

		default:
			// Adjudicated — a single-actor set, mirroring runChain's Stage 3 default branch.
			ar, adjErr := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: pp.ActorID, Attempt: pp.Attempt}}, nil, tickAfter, curSeq, "", trace)
			if adjErr != nil {
				return "", fmt.Errorf("fireDuePending: pending_event %s adjudicate: %w", d.id, adjErr)
			}
			if ar.Halt != "" {
				haltReason = ar.Halt
			} else {
				committedIDs = ar.Committed
			}
			if ar.SeqAdvance > 0 {
				curSeq += ar.SeqAdvance
			} else {
				curSeq++
			}
		}

		if haltReason != "" || len(committedIDs) == 0 {
			reason := haltReason
			if reason == "" {
				reason = "no committed ids"
			}
			log.Printf("fireDuePending: pending_event %s did not commit (%s) — marking cancelled", d.id, reason)
			if _, execErr := o.DB.Exec(ctx, `UPDATE pending_event SET status='cancelled' WHERE pending_id=$1`, d.id); execErr != nil {
				return "", fmt.Errorf("fireDuePending: pending_event %s cancel status: %w", d.id, execErr)
			}
			continue
		}

		outcome.Committed = append(outcome.Committed, committedIDs...)

		// TODO(Task 9): the payload commit and this status flip are not yet in one transaction — the
		// world's-turn composer owns the beat's tx boundary and retry semantics and will make
		// commit+flip atomic (cf. commitRulingTx/resolveHeldIDs).
		if _, execErr := o.DB.Exec(ctx, `UPDATE pending_event SET status='fired' WHERE pending_id=$1`, d.id); execErr != nil {
			return "", fmt.Errorf("fireDuePending: pending_event %s flip status: %w", d.id, execErr)
		}

		if magnitudeRank[d.magnitude] > magnitudeRank[firedMag] {
			firedMag = d.magnitude
		}
	}

	return firedMag, nil
}
