package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// beatTickCap is the chain budget backstop: a beat may advance the clock at most this many ticks (parity with the retired apply_beat parameter).
const beatTickCap = 1000

// Orchestrator runs the five-stage per-attempt loop for a beat.
// Stage 1: World-first hook (CognitionBatch) — NPC decisions.
// Stage 2: Premise re-check (deterministic, generalized) — premiseHolds over all six types.
// Stage 3: Route — passthrough (apply_event) or adjudicate (Resolve driver).
// Stage 4: Advance clock — ActorMoved adds fn_move_duration ticks; others add seq.
// Stage 5: UNRESOLVED sentinel — halt immediately with candidate list.
type Orchestrator struct {
	DB                *pgxpool.Pool
	Resolve           Driver
	CognitionBatch    Driver
	CognitionIsolated Driver
	WorldActor        Driver
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

// ActorAttempt pairs an attempt with the actor who makes it. A single beat can carry
// attempts from several actors at once (multi-actor collisions), so adjudicate takes a
// set of these rather than one actorID + a flat attempt list. Each ActorID is folded into
// the gathered slice so its entity — and every bystander co-located with it — is whitelisted.
type ActorAttempt struct {
	ActorID string
	Attempt Attempt
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

		// ── Stage 1: World-first (cognition seats) ───────────────────────────────
		// Every present NPC's decision runs through the SAME pipeline as the player's — no bypass.
		// worldFirst does the §5 mechanical split (batch = public moment only; secret-holders
		// isolated) and commits each NPC decision (passthrough via apply_event, adjudicated via
		// o.adjudicate). It returns how many (tick,seq) slots the NPC commits consumed.
		wf, err := o.worldFirst(ctx, worldID, actorID, attempt, curTick, curSeq)
		if err != nil {
			return outcome, fmt.Errorf("worldFirst stage1: %w", err)
		}
		outcome.Committed = append(outcome.Committed, wf.Committed...)
		outcome.Telegraphs = append(outcome.Telegraphs, wf.TelegraphedStated...)
		curSeq += wf.SeqAdvance
		if wf.Halt != "" {
			outcome.HaltReason = wf.Halt
			outcome.TicksAdvanced = curTick - startTick
			return outcome, nil
		}

		// ── Beat-end on telegraph (RULINGS-2026-07-24 §1, §3) ────────────────────
		// A world-first seat telegraphed a disruptive act this action: its wind-up committed as
		// canon and a held_outcome row was written (possibly several — simultaneous telegraphs all
		// commit + hold). The beat ENDS here. The player's triggering attempt never resolves — the
		// world seized the moment before it landed — and every un-run chain step is DISCARDED. The
		// narration (post-beat perceptions) delivers the moment; the player's next input meets the
		// held act(s) as a reaction (Task 6).
		if wf.HeldWritten {
			outcome.HaltReason = "telegraph"
			outcome.TicksAdvanced = curTick - startTick
			return outcome, nil
		}

		// ── Stage 2: Premise re-check (deterministic, generalized — all six types) ──
		// After the world-first NPC commits, the player's next step may no longer stand: its
		// listener walked off, its target was destroyed, its destination vanished. premiseHolds
		// runs floor-shaped reads over whichever type this is; a false return stops the chain
		// (prefix stands) with premise_broken.
		holds, premErr := o.premiseHolds(ctx, worldID, actorID, attempt)
		if premErr != nil {
			return outcome, fmt.Errorf("premise re-check stage2: %w", premErr)
		}
		if !holds {
			outcome.HaltReason = "premise_broken"
			outcome.TicksAdvanced = curTick - startTick
			return outcome, nil
		}

		// ── Stage 3: Route ────────────────────────────────────────────────────────
		switch attempt.Type {
		case "ActorMoved", "Communicated", "ObjectRelocated":
			// Passthrough: commit directly via apply_event (gate enforces structural floor).
			// Capture "from" location before commit (for move duration calculation).
			fromLoc, err := o.actorLocation(ctx, worldID, actorID)
			if err != nil {
				return outcome, fmt.Errorf("actor location (pre-move): %w", err)
			}

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
			ar, adjErr := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: actorID, Attempt: attempt}}, nil, curTick, curSeq)
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

// worldFirstResult is what the world-first stage produced for one attempt: the NPC events it
// committed, the wind-up strings it telegraphed (Task 5 turns these into held outcomes), whether
// a held-outcome row was written, how many (tick,seq) slots it consumed, and a halt (unused in
// this task — the telegraph→beat-end flow lands in Task 5).
type worldFirstResult struct {
	Committed         []string
	TelegraphedStated []string
	HeldWritten       bool
	SeqAdvance        int
	Halt              string
}

// worldFirst is stage 1: the present NPCs get their word before the player's attempt resolves.
// It performs the §5 mechanical split — one cognition call per NPC, each in EXACTLY one seat —
// and commits each decision through the same pipeline the player uses (no trusted-NPC bypass).
//
//	present = fn_actors_at(player's location); npcs = present − player.
//	action ids = the attempt's bound entity ids + the player.
//	isolated = fn_isolated_npcs (private about-ness intersects the action ids); batch = npcs − isolated.
//	≤1 batch call (skipped when the batch set is empty), validated against the batch ids only;
//	one isolated call per flagged NPC, validated against exactly her own id.
//
// Deterministic processing order: batch response order first, then isolated NPCs by uuid asc.
func (o *Orchestrator) worldFirst(ctx context.Context, worldID, playerID string, attempt Attempt, tick int64, seq int) (worldFirstResult, error) {
	var res worldFirstResult

	// present roster on the player's location; npcs = present − player.
	loc, err := o.actorLocation(ctx, worldID, playerID)
	if err != nil {
		return res, fmt.Errorf("player location: %w", err)
	}
	present, err := o.fnActorsAt(ctx, worldID, loc)
	if err != nil {
		return res, fmt.Errorf("fn_actors_at: %w", err)
	}
	npcs := make([]string, 0, len(present))
	for _, id := range present {
		if id != playerID {
			npcs = append(npcs, id)
		}
	}
	// Skip cognition entirely when no NPC is present — the quiet room has nothing to think.
	if len(npcs) == 0 {
		return res, nil
	}

	// action ids = the attempt's bound ids + the player. The isolation lookup intersects these
	// with each NPC's private about-ness links, one hop (RULINGS-2026-07-23 §5).
	actionIDs := append(o.collectParticipantIDs(attempt), playerID)

	isolated, err := o.isolatedNPCs(ctx, worldID, actionIDs, present, npcs)
	if err != nil {
		return res, fmt.Errorf("fn_isolated_npcs: %w", err)
	}
	isoSet := make(map[string]bool, len(isolated))
	for _, id := range isolated {
		isoSet[id] = true
	}
	batchIDs := make([]string, 0, len(npcs))
	for _, id := range npcs {
		if !isoSet[id] {
			batchIDs = append(batchIDs, id)
		}
	}
	sort.Strings(batchIDs) // stable, cache-native DECIDE FOR list
	sort.Strings(isolated) // isolated NPCs processed by uuid asc (deterministic intra-tick order)

	// The public moment (modal face of every event shared by ALL present holders) and the scene
	// frame are shared by both seats. The isolated seat adds the flagged NPC's private records.
	moment, err := o.publicMoment(ctx, worldID, present)
	if err != nil {
		return res, fmt.Errorf("fn_public_moment: %w", err)
	}
	scene, err := o.loadScene(ctx, worldID, loc, present)
	if err != nil {
		return res, fmt.Errorf("load scene: %w", err)
	}
	imminentActor := playerID
	for _, r := range scene.Present {
		if r.ID == playerID {
			imminentActor = r.Name
			break
		}
	}

	localSeq := seq

	// ── Shared batch: ONE call for the NPCs whose read needs nothing beyond the public moment.
	// WALL INVARIANT (RULINGS-2026-07-23 §5): buildBatchPrompt is fed ONLY the batch minds' cores
	// and the PUBLIC moment — never a private line, never an isolated NPC's core beyond the public
	// roster name line. Every secret rides its own isolated call below. The wall holds by
	// construction: a secret that never enters a shared prompt cannot bleed into another mind.
	if len(batchIDs) > 0 && o.CognitionBatch != nil {
		minds, err := o.loadMinds(ctx, worldID, batchIDs)
		if err != nil {
			return res, fmt.Errorf("load batch minds: %w", err)
		}
		prompt := buildBatchPrompt(scene, minds, moment, imminentActor, attempt)
		raw, genErr := o.CognitionBatch.Generate(ctx, GenRequest{Schema: json.RawMessage(npcAttemptsSchemaJSON), Prompt: prompt})
		if genErr == nil {
			// Validated against the BATCH ids only — a decision for an isolated NPC (or the player)
			// is rejected by DecodeAndValidateNPCDecisions' allowlist (non-present-for-this-call).
			if decisions, decErr := DecodeAndValidateNPCDecisions(raw, batchIDs); decErr == nil {
				advance, applyErr := o.applyNPCDecisions(ctx, worldID, decisions, tick, localSeq, &res)
				if applyErr != nil {
					return res, applyErr
				}
				localSeq += advance
			}
		}
	}

	// ── Isolated calls: one per flagged NPC, her secret riding alone. Processed by uuid asc.
	if o.CognitionIsolated != nil {
		for _, npcID := range isolated {
			minds, err := o.loadMinds(ctx, worldID, []string{npcID})
			if err != nil {
				return res, fmt.Errorf("load isolated mind %s: %w", npcID, err)
			}
			if len(minds) == 0 {
				continue
			}
			private, err := o.privateRecords(ctx, worldID, npcID, actionIDs, present)
			if err != nil {
				return res, fmt.Errorf("fn_private_records %s: %w", npcID, err)
			}
			prompt := buildIsolatedPrompt(scene, minds[0], private, moment, imminentActor, attempt)
			raw, genErr := o.CognitionIsolated.Generate(ctx, GenRequest{Schema: json.RawMessage(npcAttemptsSchemaJSON), Prompt: prompt})
			if genErr != nil {
				continue
			}
			// Validated against EXACTLY her own id — her call may speak only for her.
			decisions, decErr := DecodeAndValidateNPCDecisions(raw, []string{npcID})
			if decErr != nil {
				continue
			}
			advance, applyErr := o.applyNPCDecisions(ctx, worldID, decisions, tick, localSeq, &res)
			if applyErr != nil {
				return res, applyErr
			}
			localSeq += advance
		}
	}

	res.SeqAdvance = localSeq - seq
	return res, nil
}

// applyNPCDecisions commits a validated decision set (in response order) into canon and returns
// how many (tick,seq) slots it consumed. NO BYPASS: a `commit` of a passthrough type
// (ActorMoved|Communicated|ObjectRelocated) goes through apply_event; every other (adjudicated)
// type goes through o.adjudicate with the NPC as the ActorAttempt actor — the same resolve
// pipeline the player's attempts use. `none` is a skip; `telegraph` is recorded (Task 5 replaces
// this stub with the held-outcome write + beat-end).
func (o *Orchestrator) applyNPCDecisions(ctx context.Context, worldID string, decisions []NPCDecision, tick int64, startSeq int, res *worldFirstResult) (int, error) {
	localSeq := startSeq
	for _, dec := range decisions {
		if dec.Reaction == nil {
			continue // "none" — the mind does nothing this moment
		}
		switch dec.Reaction.CommitKind {
		case "commit":
			switch dec.Reaction.Attempt.Type {
			case "ActorMoved", "Communicated", "ObjectRelocated":
				// Passthrough: the structural floor is the gate; no ruling needed.
				attemptJSON, _ := json.Marshal(dec.Reaction.Attempt)
				result, err := o.applyEvent(ctx, worldID, dec.ActorID, attemptJSON, tick, localSeq)
				if err != nil {
					return localSeq - startSeq, fmt.Errorf("npc apply_event: %w", err)
				}
				if result["halt_reason"] == "committed" {
					if evID, ok := result["event_id"].(string); ok && evID != "" {
						res.Committed = append(res.Committed, evID)
					}
					localSeq++ // advance so the next event never collides on (tick,seq)
				}
			default:
				// Adjudicated types (OwnershipAccessChanged|EntityDestroyed|AttributeChanged, …):
				// the SAME resolve pipeline the player uses — a "trusted NPC" fast path is a named
				// consistency hole (FINAL-world-npc-cognition "No bypass").
				ar, err := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: dec.ActorID, Attempt: dec.Reaction.Attempt}}, nil, tick, localSeq)
				if err != nil {
					return localSeq - startSeq, fmt.Errorf("npc adjudicate: %w", err)
				}
				res.Committed = append(res.Committed, ar.Committed...)
				localSeq += ar.SeqAdvance
			}
		case "telegraph":
			// A disruptive act: commit the wind-up as perceivable canon and write the paired
			// held_outcome row (one tx — wind-up + hold land together or not at all). RunBeat ends
			// the beat once worldFirst returns with HeldWritten set (RULINGS-2026-07-24 §1, §3).
			evID, err := o.commitTelegraph(ctx, worldID, dec.ActorID, dec.Reaction.Attempt, tick, localSeq)
			if err != nil {
				return localSeq - startSeq, fmt.Errorf("commit telegraph: %w", err)
			}
			res.Committed = append(res.Committed, evID)
			res.TelegraphedStated = append(res.TelegraphedStated, dec.Reaction.Attempt.Stated)
			res.HeldWritten = true
			// The wind-up consumed a (tick,seq) slot; advance so simultaneous telegraphs
			// (multiple seats — §3) each commit at a distinct seq.
			localSeq++
		}
	}
	return localSeq - startSeq, nil
}

// commitTelegraph commits a disruptive NPC's wind-up as perceivable canon and writes the paired
// held_outcome row in ONE pgx tx (RULINGS-2026-07-24 §1, §3). The wind-up is ALWAYS an
// AttributeChanged on the acting NPC (self-targeted, target_id=actor_id) via apply_ruled_event with
// p_origin='telegraph': the six-type spine carries no dedicated "wind-up" type, and self-targeting
// yields the perception fan-out + perception_subject rows for free (Station D's machinery — the
// function derives participant_ids from actor_id for AttributeChanged, so no separate
// participant_ids field is passed). The NPC's FULL typed act (her real intended move/grab/…, not
// this stand-in AttributeChanged) is preserved verbatim in held_outcome.attempt so the reaction
// beat can resolve it on the player's next input. Both writes commit together or roll back together.
func (o *Orchestrator) commitTelegraph(ctx context.Context, worldID, npcID string, attempt Attempt, tick int64, seq int) (string, error) {
	tx, err := o.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin telegraph tx: %w", err)
	}
	// Roll back on any early return; a successful Commit below flips committed so this is a no-op.
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	visible := true
	windUp := RuledEventV2{
		Type:     "AttributeChanged",
		ActorID:  npcID,
		TargetID: npcID, // self-targeted; participant_ids derive from actor_id in apply_ruled_event
		Truth:    attempt.Stated,
		Visible:  &visible,
	}
	result, err := applyRuledEventOnQuerier(ctx, tx, worldID, windUp, tick, seq, "telegraph")
	if err != nil {
		return "", fmt.Errorf("apply_ruled_event (telegraph): %w", err)
	}
	if result["halt_reason"] == "gate_reject" {
		// A present NPC's self-targeted AttributeChanged always clears the structural floor
		// (her actor row exists as kind='actor', and it is its own target). A reject here is a real
		// invariant breach, not a routine outcome — fail loud and write nothing.
		return "", fmt.Errorf("telegraph wind-up gate_reject for npc %s (self-target must never reject)", npcID)
	}
	evID, _ := result["event_id"].(string)
	if evID == "" {
		return "", fmt.Errorf("telegraph wind-up returned no event_id for npc %s", npcID)
	}

	attemptJSON, err := json.Marshal(attempt)
	if err != nil {
		return "", fmt.Errorf("marshal held attempt: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO held_outcome (world_id, actor_id, attempt, telegraph_event_id, created_tick)
		 VALUES ($1::uuid, $2::uuid, $3::jsonb, $4::uuid, $5)`,
		worldID, npcID, string(attemptJSON), evID, tick); err != nil {
		return "", fmt.Errorf("insert held_outcome: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit telegraph tx: %w", err)
	}
	committed = true
	return evID, nil
}

// premiseHolds is the generalized, deterministic premise re-check (Stage 2). It verifies the
// acting actor's next chain attempt still stands after the world-first NPC commits, using only
// floor-shaped reads — entity existence in entity_registry and co-location via fn_actors_at,
// mirroring the structural floor apply_event/apply_ruled_event will enforce at commit. It replaces
// the former two-case switch (Communicated listener + ObjectRelocated→actor dest) with coverage of
// all six types; a false return drives RunBeat's premise_broken halt.
//
// actorID is the acting actor whose current location anchors the co-location reads (Communicated
// listener, ObjectRelocated→actor dest) — the same location the retired switch read via
// actorLocation. Existence checks are location-independent so they need only the ids.
func (o *Orchestrator) premiseHolds(ctx context.Context, worldID, actorID string, a Attempt) (bool, error) {
	// exists: an entity_registry row for id in this world, optionally constrained to a kind.
	exists := func(id, kind string) (bool, error) {
		if id == "" {
			return false, nil
		}
		var ok bool
		q := `SELECT EXISTS(SELECT 1 FROM entity_registry WHERE entity_id=$1::uuid AND world_id=$2::uuid`
		args := []any{id, worldID}
		if kind != "" {
			q += ` AND entity_kind=$3`
			args = append(args, kind)
		}
		q += `)`
		if err := o.DB.QueryRow(ctx, q, args...).Scan(&ok); err != nil {
			return false, err
		}
		return ok, nil
	}
	// coLocated: id is co-located with the acting actor (both at the actor's current location).
	coLocated := func(id string) (bool, error) {
		loc, err := o.actorLocation(ctx, worldID, actorID)
		if err != nil {
			return false, err
		}
		present, err := o.fnActorsAt(ctx, worldID, loc)
		if err != nil {
			return false, err
		}
		for _, p := range present {
			if p == id {
				return true, nil
			}
		}
		return false, nil
	}

	switch a.Type {
	case "ActorMoved":
		// Destination must still exist as a location.
		return exists(a.ToLocationID, "location")
	case "Communicated":
		// Listener must still be co-located with the speaker.
		return coLocated(a.ListenerID)
	case "ObjectRelocated":
		// Object and destination must exist; an actor destination must still be co-located.
		if ok, err := exists(a.ObjectID, ""); err != nil || !ok {
			return ok, err
		}
		if ok, err := exists(a.DestID, ""); err != nil || !ok {
			return ok, err
		}
		if a.DestKind == "actor" {
			return coLocated(a.DestID)
		}
		return true, nil
	case "OwnershipAccessChanged", "EntityDestroyed", "AttributeChanged":
		// Target must still exist.
		return exists(a.TargetID, "")
	case "EntityCreated":
		// Every referenced component must exist.
		for _, cid := range a.ComponentIDs {
			if ok, err := exists(cid, ""); err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	}
	// UNRESOLVED is intercepted before Stage 2; any other type carries no premise gate.
	return true, nil
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

// adjudicate resolves an actor-attributed set of attempts using the Resolve driver +
// apply_ruled_event. It gathers the relevant slice, calls Generate (with one repair retry),
// verifies the ruling against sliceIDs, commits all events, and applies attribute writes.
//
// Every attempt carries its own actor (set[i].ActorID), so a single beat may resolve a
// multi-actor collision at once. Each actor is folded into the slice — anchoring the
// co_present union on every actor's location, not just the first — which is what keeps
// legitimately-witnessed bystanders inside the whitelist (the PR #26 blocker).
//
// resolveHeldIDs marks those held_outcome rows 'resolved' inside commitRulingTx's
// transaction. It is nil for every caller today (the held_outcome table arrives in a later
// task) and is a no-op when empty.
func (o *Orchestrator) adjudicate(ctx context.Context, worldID string, set []ActorAttempt, resolveHeldIDs []string, tick int64, curSeq int) (adjResult, error) {
	// Flatten the raw attempts for participant-ID collection and slice gathering.
	attempts := make([]Attempt, 0, len(set))
	for _, aa := range set {
		attempts = append(attempts, aa.Attempt)
	}

	// Participant ids = every actor first (deterministic — actor[0] anchors p_ids[1]),
	// then the attempt targets. verdictRuling checks that each actor_id is in sliceIDs.
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, aa := range set {
		add(aa.ActorID)
	}
	for _, id := range collectParticipantIDsFromAttempts(attempts) {
		add(id)
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
		Entities []struct {
			ID string `json:"id"`
		} `json:"entities"`
		CoPresent []struct {
			ID string `json:"id"`
		} `json:"co_present"`
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
	prompt := buildResolvePrompt(sliceJSON, set, nil)
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
			p = buildResolvePrompt(sliceJSON, set, repairErrs)
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
	committed, seqAdvance, commitErr := o.commitRulingTx(ctx, tx, worldID, ruling, resolveHeldIDs, tick, curSeq)
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
// the provided transaction, then marks any resolveHeldIDs held_outcome rows 'resolved' in
// that same transaction. Returns (nil, 0, nil) on gate_reject (caller rolls back),
// (nil, 0, err) on hard error, or (committedIDs, seqAdvance, nil) on success.
//
// resolveHeldIDs is nil for every caller today (the held_outcome table is created by a later
// task); the held-resolution UPDATE is skipped entirely when it is empty so nothing references
// the not-yet-existent table.
func (o *Orchestrator) commitRulingTx(ctx context.Context, tx pgx.Tx, worldID string, ruling RulingV2, resolveHeldIDs []string, tick int64, curSeq int) ([]string, int, error) {
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

	// Held-outcome resolution: mark the named held_outcome rows 'resolved' inside this tx so
	// the resolution lands atomically with the ruling. Skipped when empty — the held_outcome
	// table does not exist yet, so the statement must never run with no ids to resolve.
	if len(resolveHeldIDs) > 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE held_outcome SET status = 'resolved' WHERE held_id = ANY($1)`,
			resolveHeldIDs); err != nil {
			return nil, 0, fmt.Errorf("resolve held_outcome: %w", err)
		}
	}

	return committed, localSeq - curSeq, nil
}

// applyRuledEvent calls apply_ruled_event (p_origin='ruling') and returns the result as a map.
func (o *Orchestrator) applyRuledEvent(ctx context.Context, worldID string, evt RuledEventV2, tick int64, seq int) (map[string]any, error) {
	return applyRuledEventOnQuerier(ctx, o.DB, worldID, evt, tick, seq, "ruling")
}

// applyRuledEventTx calls apply_ruled_event (p_origin='ruling') inside a transaction.
func (o *Orchestrator) applyRuledEventTx(ctx context.Context, tx pgx.Tx, worldID string, evt RuledEventV2, tick int64, seq int) (map[string]any, error) {
	return applyRuledEventOnQuerier(ctx, tx, worldID, evt, tick, seq, "ruling")
}

// dbQuerier is the subset of pgxpool.Pool / pgx.Tx used by applyRuledEventOnQuerier.
type dbQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// applyRuledEventOnQuerier is the shared implementation used by both pool and tx paths. origin is
// the p_origin the commit lands under — 'ruling' for adjudicated outcomes, 'telegraph' for a
// held act's wind-up (Task 5).
func applyRuledEventOnQuerier(ctx context.Context, q dbQuerier, worldID string, evt RuledEventV2, tick int64, seq int, origin string) (map[string]any, error) {
	// Marshal evt to JSON (snake_case due to the json tags on RuledEventV2).
	evtJSON, _ := json.Marshal(evt)
	var resultJSON []byte
	err := q.QueryRow(ctx,
		`SELECT apply_ruled_event($1::uuid, $2::jsonb, $3, $4, $5)`,
		worldID, string(evtJSON), tick, seq, origin).Scan(&resultJSON)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("apply_ruled_event result parse: %w", err)
	}
	return result, nil
}
