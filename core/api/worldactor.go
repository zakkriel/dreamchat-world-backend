package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// errIntrusionRejected marks an authored intrusion the GATE refused — a bad proposal, not a broken
// machine. D-1: the model proposes, the deterministic gate decides, and "no" is an ordinary answer.
//
// It exists because the difference used to be invisible, and that cost a wedged world. During a
// journey the world's turn runs in whatever scene the leg is passing through; the dev World Actor
// authored an intrusion by an actor standing somewhere else; the scope check below correctly refused
// it — and the refusal came back as a plain error, which failed the whole beat. Nothing committed, so
// the tick never advanced, and because the pressure roll is a PURE function of
// (world, tick, lastEruption, tier) the next attempt drew the identical fire and refused it
// identically. The journey could never advance again: a deterministic livelock, reachable by any
// invalid proposal, not just that one.
//
// So refusal is now separable from failure. runWorldTurn treats a wrapped rejection as "the world does
// not erupt this turn" — logged, traced, nothing committed, beat continues — while an infrastructure
// error (a failed DB read) still propagates and fails loudly. The mis-scoped intrusion stays out of
// canon either way, which was always the point of refusing it.
var errIntrusionRejected = errors.New("world actor: intrusion rejected")

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
	//
	// Everything from here down is the GATE judging a PROPOSAL, so every rejection is wrapped in
	// errIntrusionRejected. D-1: the model proposes, the deterministic gate decides, and "no" is a
	// normal answer — the world simply does not erupt this turn. It is NOT an infrastructure failure,
	// and it must not fail the player's beat: see errIntrusionRejected's own note for what that cost.
	// Errors that are genuinely infrastructure (a DB read below, the Generate call above) stay bare
	// and still fail loudly.
	var authored pendingPayload
	if unmarshalErr := json.Unmarshal([]byte(raw), &authored); unmarshalErr != nil {
		return "", 0, fmt.Errorf("%w: decode authored intrusion: %v", errIntrusionRejected, unmarshalErr)
	}
	if authored.ActorID == "" {
		return "", 0, fmt.Errorf("%w: authored intrusion missing actor_id", errIntrusionRejected)
	}
	// Belt-and-suspenders behind the schema leash (SPEC-015/D-1 pattern): a correctly-bound structured
	// driver never trips this, a rogue/misbound one does. The World Actor always ACTS — UNRESOLVED/QUERY
	// are player-decompose-only parse shapes, never a valid authored intrusion (mirrors
	// DecodeAndValidateNPCDecisions' own belt for the cognition seats).
	if authored.Attempt.Stated == "" || !allowedBeatTypesV2[authored.Attempt.Type] ||
		authored.Attempt.Type == "UNRESOLVED" || authored.Attempt.Type == "QUERY" {
		return "", 0, fmt.Errorf("%w: authored attempt type %q invalid", errIntrusionRejected, authored.Attempt.Type)
	}
	if fieldErr := validateAttemptFields(0, authored.Attempt); fieldErr != nil {
		return "", 0, fmt.Errorf("%w: %v", errIntrusionRejected, fieldErr)
	}
	// v1 SCOPE, ENFORCED (design doc Unit 5; Living World deferral B): the intrusion manifests
	// perceivably AT `scene`. Two lawful shapes, and nothing else:
	//   * an ActorMoved whose target resolves to `scene` — the presence-boundary move, this seat's
	//     unique power to pull a non-present NPC INTO the scene;
	//   * any other act by an entity ALREADY standing in `scene`.
	// Prompt-only until now (world_actor.txt). REFUSING rather than committing keeps a mis-scoped
	// intrusion out of canon entirely — an event the player cannot be positioned to perceive is worse
	// than no eruption at all. Refusing is not the same as failing the beat, which is what this used
	// to do; see errIntrusionRejected.
	if authored.Attempt.Type == "ActorMoved" {
		dest, destErr := o.fnTargetScene(ctx, worldID, authored.Attempt.ToTargetID)
		if destErr != nil {
			return "", 0, fmt.Errorf("runWorldActor: resolve move target scene: %w", destErr)
		}
		if dest != scene {
			return "", 0, fmt.Errorf("%w: authored move lands in %s, not the scene %s", errIntrusionRejected, dest, scene)
		}
	} else {
		here, locErr := o.actorLocation(ctx, worldID, authored.ActorID)
		if locErr != nil {
			return "", 0, fmt.Errorf("runWorldActor: actor %s location: %w", authored.ActorID, locErr)
		}
		if here != scene {
			return "", 0, fmt.Errorf("%w: authored actor %s is in %s, not the scene %s", errIntrusionRejected, authored.ActorID, here, scene)
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
		//
		// This is the MOST canonical rejection of the lot: the deterministic gate looked at the world
		// actor's proposal and said no (gate_reject, premise_broken, …), exactly as it does for a
		// player's attempt. An NPC cannot walk to a place no portal reaches, and the world does not get
		// to cheat because it is the world (D-1, no trusted fast path). So the eruption simply does not
		// happen — it is not a malfunction, and it must not fail the beat.
		return "", 0, fmt.Errorf("%w: authored intrusion did not commit (%s)", errIntrusionRejected, halt)
	}
	return eventIDs[0], seqAdvance, nil
}
