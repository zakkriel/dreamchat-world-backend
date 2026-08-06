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

// commitWorldPayload commits ONE world-sourced payload — an acting entity's Attempt, not yet canon —
// through the SAME routing runChain's Stage 3 uses for a live attempt: the three passthrough types
// (ActorMoved, Communicated, ObjectRelocated) commit via applyEvent; every other type adjudicates as a
// single-actor set via o.adjudicate. This is the ONE place that routing lives (Task 8's DRY extraction,
// task-8-brief ambiguity resolution #3, folding the founder's modular mandate into a station that
// otherwise would have grown a second copy): fireDuePending (pre-caused world truth firing off the
// ledger, below) and runWorldActor (worldactor.go — freshly authored world truth) BOTH call this
// instead of each carrying its own copy of the switch.
//
// seqAdvance mirrors fireDuePending's original per-branch bookkeeping exactly: a passthrough commit
// always consumes exactly one (tick,seq) slot (1), committed or not; an adjudicated commit consumes
// ar.SeqAdvance when the ruling reports one, else falls back to 1 — the SAME fallback fireDuePending's
// prior inline switch used (`if ar.SeqAdvance > 0 { curSeq += ar.SeqAdvance } else { curSeq++ }`). The
// caller adds seqAdvance to its own running curSeq unconditionally, exactly as before.
//
// On a successful commit, every committed event id is appended to outcome.Committed (nil-safe: a nil
// outcome is a no-op append) and returned in eventIDs; halt is "" and err is nil. On gate_reject /
// bounce / ruled_event_rejected / an empty committed set — the attempt reached the pipeline but did NOT
// land in canon — eventIDs is nil, halt carries the reason (never "" in this branch), and err is nil:
// this is an ordinary, expected outcome the caller decides how to handle (fireDuePending cancels the
// row; runWorldActor fails loud — an authored intrusion that doesn't land is not silently swallowed).
// err is reserved for a genuine infrastructure failure (a DB error, a malformed adjudicate call).
func (o *Orchestrator) commitWorldPayload(ctx context.Context, worldID, actorID string, attempt Attempt, tick int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (eventIDs []string, seqAdvance int, halt string, err error) {
	switch attempt.Type {
	case "ActorMoved", "Communicated", "ObjectRelocated":
		// Passthrough — the same routing runChain's Stage 3 uses for these three types.
		attemptJSON, marshalErr := json.Marshal(attempt)
		if marshalErr != nil {
			return nil, 1, "", fmt.Errorf("commitWorldPayload: marshal attempt: %w", marshalErr)
		}
		result, applyErr := o.applyEvent(ctx, worldID, actorID, attemptJSON, tick, seq)
		if applyErr != nil {
			return nil, 1, "", fmt.Errorf("commitWorldPayload: apply_event: %w", applyErr)
		}
		// Mirror runChain's own passthrough check (orchestrator.go Stage 3): halt_reason is the
		// authoritative signal from apply_event's structural floor, not just an empty event_id.
		if hr, _ := result["halt_reason"].(string); hr == "gate_reject" {
			return nil, 1, "gate_reject", nil
		}
		evID, _ := result["event_id"].(string)
		if evID == "" {
			return nil, 1, "no_event_id", nil
		}
		if outcome != nil {
			outcome.Committed = append(outcome.Committed, evID)
		}
		return []string{evID}, 1, "", nil

	default:
		// Adjudicated — a single-actor set, mirroring runChain's Stage 3 default branch.
		ar, adjErr := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: actorID, Attempt: attempt}}, nil, tick, seq, "", trace)
		if adjErr != nil {
			return nil, 1, "", fmt.Errorf("commitWorldPayload: adjudicate: %w", adjErr)
		}
		adv := ar.SeqAdvance
		if adv <= 0 {
			adv = 1
		}
		if ar.Halt != "" {
			return nil, adv, ar.Halt, nil
		}
		if len(ar.Committed) == 0 {
			return nil, adv, "no_committed_ids", nil
		}
		if outcome != nil {
			outcome.Committed = append(outcome.Committed, ar.Committed...)
		}
		return ar.Committed, adv, "", nil
	}
}

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
//
// seqUsed reports how many (tick,seq) slots this call actually consumed — curSeq's total advance past
// the `seq` it started at — so a caller that ALSO commits something else at tickAfter in the same turn
// (Task 9's composer, when the roll also fires) can start that commit past every slot this call already
// used, instead of colliding on the SAME (tick,seq) a fired row already wrote (task-9 review, Important
// #1: a single small-magnitude pending fire does NOT skip the composer's pressure roll, so both firing
// in the same turn is an ordinary, reachable combination — not a hypothetical).
func (o *Orchestrator) fireDuePending(ctx context.Context, worldID string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, seqUsed int, err error) {
	rows, err := o.DB.Query(ctx,
		`SELECT pending_id, magnitude, payload FROM pending_event
		 WHERE world_id=$1 AND status='pending' AND fire_at_tick > $2 AND fire_at_tick <= $3
		 ORDER BY fire_at_tick`,
		worldID, tickBefore, tickAfter)
	if err != nil {
		return "", 0, fmt.Errorf("fireDuePending: query due rows: %w", err)
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
			return "", 0, fmt.Errorf("fireDuePending: scan due row: %w", scanErr)
		}
		due = append(due, d)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		rows.Close()
		return "", 0, fmt.Errorf("fireDuePending: iterate due rows: %w", rowsErr)
	}
	rows.Close()

	curSeq := seq
	for _, d := range due {
		var pp pendingPayload
		if unmarshalErr := json.Unmarshal(d.payload, &pp); unmarshalErr != nil {
			return "", curSeq - seq, fmt.Errorf("fireDuePending: pending_event %s payload: %w", d.id, unmarshalErr)
		}

		// commitWorldPayload IS Stage 3's routing (Task 8's DRY extraction) — the passthrough/adjudicated
		// switch that used to live inline here now lives in ONE place, shared with runWorldActor
		// (worldactor.go). committedIDs + halt together decide whether this row actually landed in canon,
		// exactly as before: halt != "" OR committedIDs empty ⇒ nothing committed ⇒ this row cancels, not
		// fires. commitWorldPayload already appended any committed ids to outcome.Committed on success.
		committedIDs, seqAdvance, halt, commitErr := o.commitWorldPayload(ctx, worldID, pp.ActorID, pp.Attempt, tickAfter, curSeq, outcome, trace)
		if commitErr != nil {
			return "", curSeq - seq, fmt.Errorf("fireDuePending: pending_event %s: %w", d.id, commitErr)
		}
		curSeq += seqAdvance

		if halt != "" || len(committedIDs) == 0 {
			// commitWorldPayload never returns an empty committedIDs set with an empty halt reason (see
			// its own doc comment above) — halt is always non-empty here, so there is no "" fallback to
			// cover (whole-branch review, Fix 3: the dead-code fallback that used to live here is gone).
			log.Printf("fireDuePending: pending_event %s did not commit (%s) — marking cancelled", d.id, halt)
			if _, execErr := o.DB.Exec(ctx, `UPDATE pending_event SET status='cancelled' WHERE pending_id=$1`, d.id); execErr != nil {
				return "", curSeq - seq, fmt.Errorf("fireDuePending: pending_event %s cancel status: %w", d.id, execErr)
			}
			continue
		}

		// TODO(Task 9): the payload commit and this status flip are not yet in one transaction — the
		// world's-turn composer owns the beat's tx boundary and retry semantics and will make
		// commit+flip atomic (cf. commitRulingTx/resolveHeldIDs).
		if _, execErr := o.DB.Exec(ctx, `UPDATE pending_event SET status='fired' WHERE pending_id=$1`, d.id); execErr != nil {
			return "", curSeq - seq, fmt.Errorf("fireDuePending: pending_event %s flip status: %w", d.id, execErr)
		}

		if magnitudeRank[d.magnitude] > magnitudeRank[firedMag] {
			firedMag = d.magnitude
		}
	}

	return firedMag, curSeq - seq, nil
}
