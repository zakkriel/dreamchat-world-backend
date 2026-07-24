package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// beatTickCap is the chain budget backstop: a beat may advance the clock at most this many ticks (parity with the retired apply_beat parameter).
const beatTickCap = 1000

// Orchestrator runs the five-stage per-attempt loop for a beat.
// Stage 1: World-first hook (CognitionBatch) — NPC decisions.
// Stage 2: Premise re-check (deterministic) — co-location for Communicated/ObjectRelocated.
// Stage 3: Route — passthrough (apply_event) or adjudicate (Resolve driver).
// Stage 4: Advance clock — ActorMoved adds fn_move_duration ticks; others add seq.
// Stage 5: UNRESOLVED sentinel — halt immediately with candidate list.
type Orchestrator struct {
	DB             *pgxpool.Pool
	Resolve        Driver
	CognitionBatch Driver
	WorldActor     Driver
}

// BeatOutcome is the result of RunBeat.
type BeatOutcome struct {
	Committed            []string `json:"committed"`
	HaltReason           string   `json:"halt_reason"`
	TicksAdvanced        int64    `json:"ticks_advanced"`
	UnresolvedCandidates []string `json:"unresolved_candidates"`
	Telegraphs           []string `json:"telegraphs"`
}

// adjResult is the result of a single adjudicate call.
type adjResult struct {
	Committed  []string
	Halt       string
	SeqAdvance int
}

// RunBeat executes the five-stage loop for each attempt in chain, starting at startTick.
// It commits passthrough types via apply_event and routes adjudicated types through the
// Resolve driver. Returns BeatOutcome describing what was committed and why halted.
func (o *Orchestrator) RunBeat(ctx context.Context, worldID, actorID string, chain []Attempt, startTick int64) (BeatOutcome, error) {
	curTick := startTick
	curSeq := 0
	outcome := BeatOutcome{
		Committed:            []string{},
		UnresolvedCandidates: []string{},
		Telegraphs:           []string{},
	}

	for _, attempt := range chain {
		// ── Stage 5: UNRESOLVED sentinel (check first — no commit, just halt) ────
		if attempt.Type == "UNRESOLVED" {
			outcome.HaltReason = "unresolved"
			outcome.UnresolvedCandidates = attempt.CandidateIDs
			outcome.TicksAdvanced = curTick - startTick
			return outcome, nil
		}

		// ── Stage 1: World-first hook (CognitionBatch) ───────────────────────────
		// Get actor's current location for fn_actors_at query.
		loc, err := o.actorLocation(ctx, worldID, actorID)
		if err != nil {
			return outcome, fmt.Errorf("actor location stage1: %w", err)
		}

		presentIDs, err := o.fnActorsAt(ctx, worldID, loc)
		if err != nil {
			return outcome, fmt.Errorf("fn_actors_at stage1: %w", err)
		}

		// Build prompt for cognition batch: present actor IDs + attempt JSON.
		attemptJSON, _ := json.Marshal(attempt)
		cogPrompt := fmt.Sprintf("present_actors:%v attempt:%s", presentIDs, string(attemptJSON))

		// Call CognitionBatch for NPC decisions.
		if o.CognitionBatch != nil {
			rawDecisions, err := o.CognitionBatch.Generate(ctx, GenRequest{
				Schema: json.RawMessage(npcAttemptsSchemaJSON),
				Prompt: cogPrompt,
			})
			if err == nil && rawDecisions != "" && rawDecisions != "[]" {
				decisions, decErr := DecodeAndValidateNPCDecisions(rawDecisions, presentIDs)
				if decErr == nil {
					for _, dec := range decisions {
						if dec.Reaction == nil {
							continue
						}
						switch dec.Reaction.CommitKind {
						case "commit":
							// Route NPC commit through apply_event directly (not recursive RunBeat).
							npcAttemptJSON, _ := json.Marshal(dec.Reaction.Attempt)
							result, applyErr := o.applyEvent(ctx, worldID, dec.ActorID, npcAttemptJSON, curTick, curSeq)
							if applyErr == nil && result["halt_reason"] == "committed" {
								if evID, ok := result["event_id"].(string); ok && evID != "" {
									outcome.Committed = append(outcome.Committed, evID)
								}
								curSeq++ // Fix 1: advance seq after each NPC commit to avoid (tick,seq) collision with player events.
							}
						case "telegraph":
							outcome.Telegraphs = append(outcome.Telegraphs, dec.Reaction.Attempt.Stated)
						}
					}
				}
			}
		}

		// ── Stage 2: Premise re-check (deterministic) ────────────────────────────
		// Re-read actor location (may have changed from NPC commits above).
		loc, err = o.actorLocation(ctx, worldID, actorID)
		if err != nil {
			return outcome, fmt.Errorf("actor location stage2: %w", err)
		}

		if attempt.Type == "Communicated" {
			// Listener must be co-located with actor.
			presentNow, premErr := o.fnActorsAt(ctx, worldID, loc)
			if premErr != nil {
				return outcome, fmt.Errorf("fn_actors_at premise: %w", premErr)
			}
			found := false
			for _, id := range presentNow {
				if id == attempt.ListenerID {
					found = true
					break
				}
			}
			if !found {
				outcome.HaltReason = "premise_broken"
				outcome.TicksAdvanced = curTick - startTick
				return outcome, nil
			}
		} else if attempt.Type == "ObjectRelocated" && attempt.DestKind == "actor" {
			// Destination actor must be co-located.
			presentNow, premErr := o.fnActorsAt(ctx, worldID, loc)
			if premErr != nil {
				return outcome, fmt.Errorf("fn_actors_at premise (ObjectRelocated): %w", premErr)
			}
			found := false
			for _, id := range presentNow {
				if id == attempt.DestID {
					found = true
					break
				}
			}
			if !found {
				outcome.HaltReason = "premise_broken"
				outcome.TicksAdvanced = curTick - startTick
				return outcome, nil
			}
		}

		// ── Stage 3: Route ────────────────────────────────────────────────────────
		switch attempt.Type {
		case "ActorMoved", "Communicated", "ObjectRelocated":
			// Passthrough: commit directly via apply_event (gate enforces structural floor).
			// Capture "from" location before commit (for move duration calculation).
			fromLoc := loc

			attemptJSONBytes, _ := json.Marshal(attempt)
			result, err := o.applyEvent(ctx, worldID, actorID, attemptJSONBytes, curTick, curSeq)
			if err != nil {
				return outcome, fmt.Errorf("apply_event passthrough: %w", err)
			}
			if result["halt_reason"] == "gate_reject" {
				outcome.HaltReason = "gate_reject"
				outcome.TicksAdvanced = curTick - startTick
				return outcome, nil
			}
			evID, _ := result["event_id"].(string)
			if evID != "" {
				outcome.Committed = append(outcome.Committed, evID)
			}

			// Stage 4: Advance clock after passthrough.
			if attempt.Type == "ActorMoved" {
				dur, durErr := o.fnMoveDuration(ctx, worldID, fromLoc, attempt.ToLocationID)
				if durErr != nil {
					return outcome, fmt.Errorf("fn_move_duration: %w", durErr)
				}
				curTick += dur
				curSeq = 0
				// Backstop: turn budget check.
				if curTick-startTick > beatTickCap {
					outcome.HaltReason = "turn_budget"
					outcome.TicksAdvanced = curTick - startTick
					return outcome, nil
				}
			} else {
				curSeq++
			}

		default:
			// Adjudicated: AttributeChanged, OwnershipAccessChanged, EntityCreated,
			// EntityDestroyed, and now ALL six types via the v2 adjudicated path.
			ar, adjErr := o.adjudicate(ctx, worldID, actorID, []Attempt{attempt}, curTick, curSeq)
			if adjErr != nil {
				return outcome, adjErr
			}
			outcome.Committed = append(outcome.Committed, ar.Committed...)
			if ar.Halt != "" {
				outcome.HaltReason = ar.Halt
				outcome.TicksAdvanced = curTick - startTick
				return outcome, nil
			}
			curSeq += ar.SeqAdvance
		}
	}

	outcome.HaltReason = "completed"
	outcome.TicksAdvanced = curTick - startTick
	return outcome, nil
}

// actorLocation returns the actor's current location_id from actor_state.
func (o *Orchestrator) actorLocation(ctx context.Context, worldID, actorID string) (string, error) {
	var loc string
	err := o.DB.QueryRow(ctx,
		`SELECT (attrs->>'location_id')::text FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		worldID, actorID).Scan(&loc)
	return loc, err
}

// fnActorsAt returns the entity IDs of actors at the given location.
func (o *Orchestrator) fnActorsAt(ctx context.Context, worldID, locationID string) ([]string, error) {
	if locationID == "" {
		return nil, nil
	}
	rows, err := o.DB.Query(ctx,
		`SELECT entity_id::text FROM fn_actors_at($1, $2::uuid)`,
		worldID, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// fnMoveDuration calls fn_move_duration(world, from, to) and returns the tick count.
func (o *Orchestrator) fnMoveDuration(ctx context.Context, worldID, fromLoc, toLoc string) (int64, error) {
	if fromLoc == "" || toLoc == "" {
		return 0, nil
	}
	var dur int64
	err := o.DB.QueryRow(ctx,
		`SELECT fn_move_duration($1, $2::uuid, $3::uuid)`,
		worldID, fromLoc, toLoc).Scan(&dur)
	return dur, err
}

// applyEvent calls apply_event and returns the result as a map.
func (o *Orchestrator) applyEvent(ctx context.Context, worldID, actorID string, attemptJSON []byte, tick int64, seq int) (map[string]any, error) {
	var resultJSON []byte
	err := o.DB.QueryRow(ctx,
		`SELECT apply_event($1::uuid, $2::uuid, $3::jsonb, $4, $5, 'freeform', false)`,
		worldID, actorID, string(attemptJSON), tick, seq).Scan(&resultJSON)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("apply_event result parse: %w", err)
	}
	return result, nil
}

// collectParticipantIDs gathers all UUID fields from a single Attempt (for backward compat).
func (o *Orchestrator) collectParticipantIDs(a Attempt) []string {
	return collectParticipantIDsFromAttempts([]Attempt{a})
}

// collectParticipantIDsFromAttempts gathers all UUID fields from multiple Attempts.
func collectParticipantIDsFromAttempts(attempts []Attempt) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, a := range attempts {
		add(a.TargetID)
		add(a.GranteeID)
		add(a.ObjectID)
		add(a.DestID)
		add(a.ListenerID)
		for _, id := range a.ComponentIDs {
			add(id)
		}
	}
	return ids
}

// adjudicate resolves one or more attempts using the Resolve driver + apply_ruled_event.
// It gathers the relevant slice, calls Generate (with one repair retry), verifies the
// ruling against sliceIDs, commits all events, and applies attribute writes.
// actorID is the entity initiating the action (always included in the slice).
func (o *Orchestrator) adjudicate(ctx context.Context, worldID, actorID string, attempts []Attempt, tick int64, curSeq int) (adjResult, error) {
	// Collect participant IDs from all attempts, including the acting actor.
	ids := collectParticipantIDsFromAttempts(attempts)
	// Ensure the actor is in the slice (their entity appears in the facts, and
	// verdictRuling checks that actor_id is in sliceIDs).
	actorAlreadyIn := false
	for _, id := range ids {
		if id == actorID {
			actorAlreadyIn = true
			break
		}
	}
	if actorID != "" && !actorAlreadyIn {
		ids = append(ids, actorID)
	}

	// Gather slice from DB.
	var sliceRaw []byte
	if len(ids) > 0 {
		err := o.DB.QueryRow(ctx,
			`SELECT gather_slice($1::uuid, $2::uuid[])`,
			worldID, ids).Scan(&sliceRaw)
		if err != nil {
			return adjResult{}, fmt.Errorf("gather_slice: %w", err)
		}
	} else {
		sliceRaw = []byte(`{"entities":[],"relationships":[],"recent_events":[],"co_present":[]}`)
	}
	sliceJSON := string(sliceRaw)

	// Build sliceIDs: entities[].id ∪ co_present[].id (grounded facts shown to the LLM)
	// ∪ attempt participant IDs ∪ actorID. We unmarshal only the top-level id fields so
	// that relationship counterparties and UUIDs embedded in event summary text are NOT
	// whitelisted — those are fetched context, not grounded entities the LLM may claim.
	var sliceTop struct {
		Entities  []struct{ ID string `json:"id"` } `json:"entities"`
		CoPresent []struct{ ID string `json:"id"` } `json:"co_present"`
	}
	_ = json.Unmarshal(sliceRaw, &sliceTop) // best-effort; empty on failure is safe
	sliceIDs := map[string]bool{}
	for _, e := range sliceTop.Entities {
		if e.ID != "" {
			sliceIDs[e.ID] = true
		}
	}
	for _, e := range sliceTop.CoPresent {
		if e.ID != "" {
			sliceIDs[e.ID] = true
		}
	}
	// Also include all attempt participant IDs and actorID: these are the direct
	// inputs to the ruling and are always legitimate whitelist entries.
	for _, id := range ids {
		sliceIDs[id] = true
	}

	// Resolve: one combined judgment over the attempt set, with one repair retry on decode/verdict failure.
	prompt := buildResolvePrompt(sliceJSON, attempts, nil)
	var ruling RulingV2
	var violations []string
	var decodeErr error

	for retry := 0; retry < 2; retry++ {
		p := prompt
		if retry > 0 {
			var repairErrs []string
			if decodeErr != nil {
				repairErrs = []string{decodeErr.Error()}
			} else {
				repairErrs = violations
			}
			p = buildResolvePrompt(sliceJSON, attempts, repairErrs)
		}

		rawRuling, genErr := o.Resolve.Generate(ctx, GenRequest{
			Schema: json.RawMessage(rulingV2SchemaJSON),
			Prompt: p,
		})
		if genErr != nil {
			decodeErr = genErr
			violations = nil
			continue
		}

		ruling, decodeErr = DecodeAndValidateRulingV2(rawRuling)
		if decodeErr != nil {
			violations = nil
			continue
		}

		violations = verdictRuling(ruling, sliceIDs, ids)
		if len(violations) == 0 {
			break
		}
		// violations found — will retry once with repair prompt
		decodeErr = nil
	}

	// After two attempts, if still broken → bounce.
	if decodeErr != nil || len(violations) > 0 {
		return adjResult{Halt: "bounce"}, nil
	}

	// impossible → bounce (nothing written).
	if ruling.Therefore == "impossible" {
		return adjResult{Halt: "bounce"}, nil
	}

	// Commit phase — ONE transaction: all apply_ruled_event calls + apply_attribute_writes.
	// A gate_reject or error anywhere rolls back the entire ruling so zero events land
	// (preventing the half-committed combined-ruling class of corrupted canon).
	tx, err := o.DB.Begin(ctx)
	if err != nil {
		return adjResult{}, fmt.Errorf("begin ruling tx: %w", err)
	}
	committed, seqAdvance, commitErr := o.commitRulingTx(ctx, tx, worldID, ruling, tick, curSeq)
	if commitErr != nil {
		_ = tx.Rollback(ctx)
		return adjResult{}, commitErr
	}
	if committed == nil {
		// gate_reject during one of the events — rollback, halt with zero durable events.
		_ = tx.Rollback(ctx)
		return adjResult{Halt: "ruled_event_rejected"}, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return adjResult{}, fmt.Errorf("commit ruling tx: %w", err)
	}
	return adjResult{Committed: committed, SeqAdvance: seqAdvance}, nil
}

// commitRulingTx executes all apply_ruled_event + apply_attribute_writes calls inside
// the provided transaction. Returns (nil, 0, nil) on gate_reject (caller rolls back),
// (nil, 0, err) on hard error, or (committedIDs, seqAdvance, nil) on success.
func (o *Orchestrator) commitRulingTx(ctx context.Context, tx pgx.Tx, worldID string, ruling RulingV2, tick int64, curSeq int) ([]string, int, error) {
	localSeq := curSeq
	var committed []string

	for _, evt := range ruling.Outcome.Events {
		result, err := o.applyRuledEventTx(ctx, tx, worldID, evt, tick, localSeq)
		if err != nil {
			return nil, 0, fmt.Errorf("apply_ruled_event: %w", err)
		}
		if result["halt_reason"] == "gate_reject" {
			return nil, 0, nil // signal gate_reject without a hard error
		}
		if evID, ok := result["event_id"].(string); ok && evID != "" {
			committed = append(committed, evID)
		}
		localSeq++
	}

	// Apply attribute writes if any.
	if len(ruling.Outcome.AttributeWrites) > 0 && len(committed) > 0 {
		writesJSON, _ := json.Marshal(ruling.Outcome.AttributeWrites)
		var rowsWritten int
		err := tx.QueryRow(ctx,
			`SELECT apply_attribute_writes($1::uuid, $2::jsonb, $3::uuid, $4, $5)`,
			worldID, string(writesJSON), committed[0], tick, localSeq).Scan(&rowsWritten)
		if err != nil {
			return nil, 0, fmt.Errorf("apply_attribute_writes: %w", err)
		}
		if rowsWritten != len(ruling.Outcome.AttributeWrites) {
			return nil, 0, fmt.Errorf("engine inconsistency: apply_attribute_writes wrote %d rows, expected %d", rowsWritten, len(ruling.Outcome.AttributeWrites))
		}
	}

	return committed, localSeq - curSeq, nil
}

// applyRuledEvent calls apply_ruled_event and returns the result as a map.
func (o *Orchestrator) applyRuledEvent(ctx context.Context, worldID string, evt RuledEventV2, tick int64, seq int) (map[string]any, error) {
	return applyRuledEventOnQuerier(ctx, o.DB, worldID, evt, tick, seq)
}

// applyRuledEventTx calls apply_ruled_event inside a transaction.
func (o *Orchestrator) applyRuledEventTx(ctx context.Context, tx pgx.Tx, worldID string, evt RuledEventV2, tick int64, seq int) (map[string]any, error) {
	return applyRuledEventOnQuerier(ctx, tx, worldID, evt, tick, seq)
}

// dbQuerier is the subset of pgxpool.Pool / pgx.Tx used by applyRuledEventOnQuerier.
type dbQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// applyRuledEventOnQuerier is the shared implementation used by both pool and tx paths.
func applyRuledEventOnQuerier(ctx context.Context, q dbQuerier, worldID string, evt RuledEventV2, tick int64, seq int) (map[string]any, error) {
	// Marshal evt to JSON (snake_case due to the json tags on RuledEventV2).
	evtJSON, _ := json.Marshal(evt)
	var resultJSON []byte
	err := q.QueryRow(ctx,
		`SELECT apply_ruled_event($1::uuid, $2::jsonb, $3, $4, 'ruling')`,
		worldID, string(evtJSON), tick, seq).Scan(&resultJSON)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("apply_ruled_event result parse: %w", err)
	}
	return result, nil
}
