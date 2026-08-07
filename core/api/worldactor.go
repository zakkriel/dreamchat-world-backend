package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// Living World / Task 8 (Unit 5) — the World Actor seat. The ONLY LLM boundary in the Living World
// station: when a pressure tier fires (Task 9's composer decides that), this authors ONE intrusion
// within the drawn size. It sees the WHOLE WORLD (fn_world_slice — unlike every other seat, which
// reasons over a gather_slice-bounded action or a fn_fact_sheet-bounded set of targets), authors a truth
// event constrained to the drawn size, and commits it through the SAME pipeline everyone else uses
// (commitWorldPayload — ledger.go's Task 8 DRY extraction, shared with fireDuePending). It is LIVE in
// every beat: runWorldTurn (worldturn.go) calls it when a tier fires, and runChain invokes the world's
// turn after each committed clock-advancing attempt (orchestrator.go advanceWorldTurn); the real driver
// is bound in beathandler.go. It also stays directly callable (and directly tested) on its own.

// runWorldActor builds the world-scope payload (fn_world_slice(worldID, scene)), hands it to the bound
// WorldActor driver with `size` as an input constraint (the drawn tier: "small"|"medium"|"large") and
// the world_actor.v1 schema (the leash), decodes the ONE authored {actor_id, attempt} it returns,
// validates the attempt against the SAME closed-vocabulary field rules every other attempt obeys
// (validateAttemptFields — no bypass: the World Actor is not a trusted fast path), and commits it via
// commitWorldPayload — the SAME routing fireDuePending uses for pre-caused world truth. Returns the
// committed event's id.
//
// B-GROWTH INVARIANT (do not violate — design doc Unit 5): the World Actor authors a TRUTH event that
// carries a LOCATION; it NEVER encodes who perceives it. This function does NO perception-edge
// computation of its own — it commits the truth event and the EXISTING commit-path fan-out
// (generate_perceptions for a passthrough commit; apply_ruled_event's own receiver loop for an
// adjudicated one) delivers it to witnesses, exactly as it does for every other committed event. v1
// scope: the authored event manifests perceivably AT `scene` (world_actor.txt instructs the seat to
// attribute the intrusion to a world entity already at the scene, or to bring a non-present NPC INTO it
// via an ActorMoved — the presence-boundary mover, a power unique to this seat) — the location is
// whatever the commit path's own accessibility/co-location floor already enforces (apply_event's
// Communicated co-presence check, an ActorMoved's resolved destination scene, …), never a location this
// function assigns itself.
//
// Does NOT write world_eruption itself — Task 9's composer still OWNS the fire-log row; it now supplies
// that write as a postCommitFn (ledger.go) which this seat hands straight to commitWorldPayload, so the
// row lands inside the SAME transaction as the truth event it records. This seat only authors and
// commits the truth event, and never inspects or invents the bookkeeping it carries.
//
// seqUsed reports how many (tick,seq) slots the commit consumed (commitWorldPayload's own seqAdvance —
// 1 for a passthrough commit, ar.SeqAdvance/fallback-1 for an adjudicated one) — 0 on every error path
// (the composer treats any error here as fatal and never threads seqUsed past it; task-9 review,
// Important #1).
func (o *Orchestrator) runWorldActor(ctx context.Context, worldID, scene, size string, now int64, seq int, postCommit postCommitFn, outcome *BeatOutcome, trace *BeatTrace) (eventID string, seqUsed int, err error) {
	var sliceRaw []byte
	if err := o.DB.QueryRow(ctx, `SELECT fn_world_slice($1::uuid, $2::uuid)`, worldID, scene).Scan(&sliceRaw); err != nil {
		return "", 0, fmt.Errorf("runWorldActor: fn_world_slice: %w", err)
	}

	prompt := buildWorldActorPrompt(string(sliceRaw), size)
	raw, genErr := o.WorldActor.Generate(ctx, GenRequest{
		Schema: json.RawMessage(worldActorSchemaJSON),
		Prompt: prompt,
	})
	if genErr != nil {
		return "", 0, fmt.Errorf("runWorldActor: Generate: %w", genErr)
	}

	// The authored shape is EXACTLY pendingPayload's {"actor_id":..., "attempt":{...}} (ledger.go) — Task
	// 4 established it so the World Actor could reuse it verbatim (task-8-brief ambiguity resolution #2).
	var authored pendingPayload
	if unmarshalErr := json.Unmarshal([]byte(raw), &authored); unmarshalErr != nil {
		return "", 0, fmt.Errorf("runWorldActor: decode authored intrusion: %w", unmarshalErr)
	}
	if authored.ActorID == "" {
		return "", 0, fmt.Errorf("runWorldActor: authored intrusion missing actor_id")
	}
	// Belt-and-suspenders behind the schema leash (SPEC-015/D-1 pattern): a correctly-bound structured
	// driver never trips this, a rogue/misbound one does. The World Actor always ACTS — UNRESOLVED/QUERY
	// are player-decompose-only parse shapes, never a valid authored intrusion (mirrors
	// DecodeAndValidateNPCDecisions' own belt for the cognition seats).
	if authored.Attempt.Stated == "" || !allowedBeatTypesV2[authored.Attempt.Type] ||
		authored.Attempt.Type == "UNRESOLVED" || authored.Attempt.Type == "QUERY" {
		return "", 0, fmt.Errorf("runWorldActor: authored attempt type %q invalid", authored.Attempt.Type)
	}
	if fieldErr := validateAttemptFields(0, authored.Attempt); fieldErr != nil {
		return "", 0, fmt.Errorf("runWorldActor: %w", fieldErr)
	}
	// v1 SCOPE, ENFORCED (design doc Unit 5; Living World deferral B): the intrusion manifests
	// perceivably AT `scene`. Two lawful shapes, and nothing else:
	//   * an ActorMoved whose target resolves to `scene` — the presence-boundary move, this seat's
	//     unique power to pull a non-present NPC INTO the scene;
	//   * any other act by an entity ALREADY standing in `scene`.
	// Prompt-only until now (world_actor.txt). Failing loud rather than committing keeps a
	// mis-scoped intrusion out of canon entirely — an event the player cannot be positioned to
	// perceive is worse than no eruption at all.
	if authored.Attempt.Type == "ActorMoved" {
		dest, destErr := o.fnTargetScene(ctx, worldID, authored.Attempt.ToTargetID)
		if destErr != nil {
			return "", 0, fmt.Errorf("runWorldActor: resolve move target scene: %w", destErr)
		}
		if dest != scene {
			return "", 0, fmt.Errorf("runWorldActor: authored move lands in %s, not the scene %s", dest, scene)
		}
	} else {
		here, locErr := o.actorLocation(ctx, worldID, authored.ActorID)
		if locErr != nil {
			return "", 0, fmt.Errorf("runWorldActor: actor %s location: %w", authored.ActorID, locErr)
		}
		if here != scene {
			return "", 0, fmt.Errorf("runWorldActor: authored actor %s is in %s, not the scene %s", authored.ActorID, here, scene)
		}
	}

	eventIDs, seqAdvance, halt, commitErr := o.commitWorldPayload(ctx, worldID, authored.ActorID, authored.Attempt, now, seq, postCommit, outcome, trace)
	if commitErr != nil {
		return "", 0, fmt.Errorf("runWorldActor: commit: %w", commitErr)
	}
	if halt != "" || len(eventIDs) == 0 {
		// commitWorldPayload never returns an empty eventIDs set with an empty halt reason (see its own
		// doc comment, ledger.go) — halt is always non-empty here, so there is no "" fallback to cover
		// (whole-branch review, Fix 3: the dead-code fallback that used to live here is gone).
		return "", 0, fmt.Errorf("runWorldActor: authored intrusion did not commit (%s)", halt)
	}
	return eventIDs[0], seqAdvance, nil
}
