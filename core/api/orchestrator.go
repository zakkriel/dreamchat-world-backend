package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// beatTickCap is the generous hard time-cap backstop (§9; ADR-025 provisional — tune at the gate).
// NOTE: Also defined in beathandler.go; orchestrator uses this shared constant.
// Using a local const to avoid redeclaration — see beathandler.go for the canonical value.
const orchBeatTickCap = 1000

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
				if curTick-startTick > orchBeatTickCap {
					outcome.HaltReason = "turn_budget"
					outcome.TicksAdvanced = curTick - startTick
					return outcome, nil
				}
			} else {
				curSeq++
			}

		default:
			// Adjudicated: AttributeChanged, OwnershipAccessChanged, EntityCreated, EntityDestroyed.
			// Gather truth slice for the resolve driver.
			participantIDs := o.collectParticipantIDs(attempt)
			truthJSON, truthErr := o.gatherTruth(ctx, worldID, participantIDs)
			if truthErr != nil {
				return outcome, fmt.Errorf("gather truth: %w", truthErr)
			}

			attemptJSONBytes, _ := json.Marshal(attempt)
			resolvePrompt := fmt.Sprintf("truth:%s attempt:%s", truthJSON, string(attemptJSONBytes))

			// Call Resolve driver with retry on validation failure.
			var ruling Ruling
			var resolveErr error
			for retry := 0; retry < 2; retry++ {
				p := resolvePrompt
				if retry > 0 && resolveErr != nil {
					p = resolvePrompt + " error:" + resolveErr.Error()
				}
				rawRuling, genErr := o.Resolve.Generate(ctx, GenRequest{
					Schema: json.RawMessage(rulingV1SchemaJSON),
					Prompt: p,
				})
				if genErr != nil {
					resolveErr = genErr
					continue
				}
				ruling, resolveErr = DecodeAndValidateRuling(rawRuling)
				if resolveErr == nil {
					break
				}
			}
			if resolveErr != nil {
				outcome.HaltReason = "bounce"
				outcome.TicksAdvanced = curTick - startTick
				return outcome, nil
			}
			if ruling.Therefore == "impossible" {
				outcome.HaltReason = "bounce"
				outcome.TicksAdvanced = curTick - startTick
				return outcome, nil
			}

			// Commit each ruled event via apply_event.
			for _, evt := range ruling.Outcome.Events {
				ruledAttempt := Attempt{
					Type:   evt.Type,
					Stated: evt.Summary,
				}
				if len(evt.ParticipantIDs) > 0 {
					ruledAttempt.TargetID = evt.ParticipantIDs[0]
				}
				ruledJSON, _ := json.Marshal(ruledAttempt)
				result, applyErr := o.applyEvent(ctx, worldID, actorID, ruledJSON, curTick, curSeq)
				if applyErr != nil {
					return outcome, fmt.Errorf("apply_event ruled: %w", applyErr)
				}
				if result["halt_reason"] == "committed" {
					if evID, ok := result["event_id"].(string); ok && evID != "" {
						outcome.Committed = append(outcome.Committed, evID)
					}
				}
				curSeq++
			}
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

// collectParticipantIDs gathers all UUID fields from an Attempt for the truth slice query.
func (o *Orchestrator) collectParticipantIDs(a Attempt) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	add(a.TargetID)
	add(a.GranteeID)
	add(a.ObjectID)
	add(a.DestID)
	add(a.ListenerID)
	for _, id := range a.ComponentIDs {
		add(id)
	}
	return ids
}

// gatherTruth runs the truth-slice SQL and returns a JSON string.
func (o *Orchestrator) gatherTruth(ctx context.Context, worldID string, participantIDs []string) (string, error) {
	if len(participantIDs) == 0 {
		return "[]", nil
	}
	// Build a UUID array for the query.
	// pgx accepts []string but the SQL needs uuid[]; cast in the query.
	var truthJSON []byte
	err := o.DB.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(row), '[]'::jsonb) FROM (
			SELECT er.entity_id, er.entity_kind, er.canonical_name,
				(SELECT attrs FROM actor_state    WHERE entity_id=er.entity_id AND world_id=$1 LIMIT 1) as actor_attrs,
				(SELECT attrs FROM artifact_state WHERE entity_id=er.entity_id AND world_id=$1 LIMIT 1) as artifact_attrs,
				(SELECT location_attrs FROM location_state WHERE entity_id=er.entity_id AND world_id=$1 LIMIT 1) as location_attrs,
				(SELECT jsonb_agg(ce.event_type||':'||ce.summary ORDER BY ce.in_world_tick DESC)
				 FROM canon_event ce
				 JOIN event_participant ep ON ep.event_id=ce.event_id
				 WHERE ep.entity_id=er.entity_id AND ce.world_id=$1
				 LIMIT 10) as last_10_events
			FROM entity_registry er
			WHERE er.world_id=$1
			  AND er.entity_id = ANY($2::uuid[])
		) rows`,
		worldID, participantIDs).Scan(&truthJSON)
	if err != nil {
		return "[]", nil // non-fatal: proceed with empty truth
	}
	return string(truthJSON), nil
}
