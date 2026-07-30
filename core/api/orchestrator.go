package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The flat beatTickCap=1000 backstop is retired: the REAL beat budget is now the tension-derived
// seconds (FINAL-action-contracts.md §6), read at beat start and consumed CUMULATIVELY across the
// chain (see runChain's ActorMoved clock stage and tension.go). A move whose duration exceeds the
// remaining budget REJECTs and does not commit — the prefix stands. Only a belt-and-suspenders
// runaway/overflow guard survives (dur > MaxInt64-curTick), which also catches the speed-0
// "blocked by arithmetic" sentinel (§2) even under `none` (∞), where the budget alone never fires.

// Orchestrator runs the five-stage per-attempt loop for a beat.
// Stage 1: World-first hook (CognitionBatch) — NPC decisions.
// Stage 2: Premise re-check (deterministic, generalized) — premiseHolds over all six types.
// Stage 3: Route — passthrough (apply_event) or adjudicate (Resolve driver).
// Stage 4: Advance clock — ActorMoved adds fn_move_duration_actor ticks; others add seq.
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
//
// The per-attempt loop itself lives in runChain, which RunReactionBeat's post-collision remainder
// also uses — a normal chain is a normal chain wherever it runs (RULINGS-2026-07-24 §2).
func (o *Orchestrator) RunBeat(ctx context.Context, worldID, actorID string, chain []Attempt, startTick int64) (BeatOutcome, error) {
	outcome := BeatOutcome{
		Committed:            []string{},
		UnresolvedCandidates: []string{},
		Telegraphs:           []string{},
	}
	// §6: the beat's cumulative time budget is derived from the scene's CURRENT tension at beat
	// start (measured at ask-time, never stored). An unset scene reads 'none' → ∞ and never halts.
	budgetRemaining, err := o.beatBudgetSeconds(ctx, worldID, actorID)
	if err != nil {
		return outcome, fmt.Errorf("beat budget: %w", err)
	}
	if err := o.runChain(ctx, worldID, actorID, chain, startTick, 0, budgetRemaining, &outcome); err != nil {
		return outcome, err
	}
	return outcome, nil
}

// runChain is the shared per-attempt loop: it advances curTick/curSeq from (startTick, startSeq),
// appending every commit/telegraph into outcome and setting outcome.HaltReason + TicksAdvanced when
// it stops (halt) or reaches the end (completed). TicksAdvanced is measured from the startTick
// passed in, so a reaction beat's remainder — which starts at the reaction's own startTick (the
// combined ruling advances seq only) — reports the whole beat's tick span.
//
// This is the identical five-stage loop RunBeat always ran; extracting it is a pure refactor for
// the existing path so RunReactionBeat can run "a normal chain against the post-collision world"
// (RULINGS-2026-07-24 §2) through the very same code.
//
// budgetRemaining is the tension-derived beat budget in seconds (§6), consumed CUMULATIVELY: each
// ActorMoved subtracts its real duration, and a move whose duration exceeds what remains halts the
// chain (turn_budget) WITHOUT committing — the prefix stands. `none` passes ∞ (math.MaxInt64), so it
// never blocks. Non-move steps have no computable duration in v1 and consume 0 (Way-A long-action
// accumulation is a later station, §6).
func (o *Orchestrator) runChain(ctx context.Context, worldID, actorID string, chain []Attempt, startTick int64, startSeq int, budgetRemaining int64, outcome *BeatOutcome) error {
	curTick := startTick
	curSeq := startSeq

	for _, attempt := range chain {
		// ── Stage 5: UNRESOLVED sentinel (check first — no commit, just halt) ────
		if attempt.Type == "UNRESOLVED" {
			outcome.HaltReason = "unresolved"
			outcome.UnresolvedCandidates = attempt.CandidateIDs
			outcome.TicksAdvanced = curTick - startTick
			return nil
		}

		// ── Stage 1: World-first (cognition seats) ───────────────────────────────
		// Every present NPC's decision runs through the SAME pipeline as the player's — no bypass.
		// worldFirst does the §5 mechanical split (batch = public moment only; secret-holders
		// isolated) and commits each NPC decision (passthrough via apply_event, adjudicated via
		// o.adjudicate). It returns how many (tick,seq) slots the NPC commits consumed.
		wf, err := o.worldFirst(ctx, worldID, actorID, attempt, curTick, curSeq)
		if err != nil {
			return fmt.Errorf("worldFirst stage1: %w", err)
		}
		outcome.Committed = append(outcome.Committed, wf.Committed...)
		outcome.Telegraphs = append(outcome.Telegraphs, wf.TelegraphedStated...)
		curSeq += wf.SeqAdvance

		// ── Beat-end on telegraph (RULINGS-2026-07-24 §1, §3) ────────────────────
		// A world-first seat telegraphed a disruptive act this action: its wind-up committed as
		// canon and a held_outcome row was written (possibly several — simultaneous telegraphs all
		// commit + hold). The beat ENDS here. The player's triggering attempt never resolves — the
		// world seized the moment before it landed — and every un-run chain step is DISCARDED. The
		// narration (post-beat perceptions) delivers the moment; the player's next input meets the
		// held act(s) as a reaction (Task 6). A remainder chain (RunReactionBeat) reaching this same
		// branch legally ends the reaction beat on its own fresh telegraph.
		if wf.HeldWritten {
			outcome.HaltReason = "telegraph"
			outcome.TicksAdvanced = curTick - startTick
			return nil
		}

		// ── Stage 2: Premise re-check (deterministic, generalized — all six types) ──
		// After the world-first NPC commits, the player's next step may no longer stand: its
		// listener walked off, its target was destroyed, its destination vanished. premiseHolds
		// runs floor-shaped reads over whichever type this is; a false return stops the chain
		// (prefix stands) with premise_broken.
		holds, premErr := o.premiseHolds(ctx, worldID, actorID, attempt)
		if premErr != nil {
			return fmt.Errorf("premise re-check stage2: %w", premErr)
		}
		if !holds {
			outcome.HaltReason = "premise_broken"
			outcome.TicksAdvanced = curTick - startTick
			return nil
		}

		// ── Stage 3: Route ────────────────────────────────────────────────────────
		switch attempt.Type {
		case "ActorMoved", "Communicated", "ObjectRelocated":
			// Passthrough: commit directly via apply_event (gate enforces structural floor).
			// Capture "from" location before commit (for move duration calculation).
			fromLoc, err := o.actorLocation(ctx, worldID, actorID)
			if err != nil {
				return fmt.Errorf("actor location (pre-move): %w", err)
			}

			// ── §6 tension budget: a move must FIT the remaining beat budget BEFORE it commits ──────
			// Compute the move's real, status-aware duration up front (§2: distance(from,to) /
			// effective_speed(actor,'walk'); a -100% modifier → speed 0 → max-bigint sentinel). If it
			// exceeds what remains, the chain halts turn_budget and this move does NOT commit — the
			// prefix stands (§6 example: three 12 s moves in a 30 s beat → 12,24 commit, 36 rejects).
			// The runaway/overflow guard (dur > MaxInt64-curTick) is belt-and-suspenders and, under
			// `none` (∞ budget) where the first clause never fires, still catches the speed-0 sentinel.
			var dur int64
			if attempt.Type == "ActorMoved" {
				dur, err = o.fnMoveDurationActor(ctx, worldID, actorID, fromLoc, attempt.ToLocationID)
				if err != nil {
					return fmt.Errorf("fn_move_duration_actor: %w", err)
				}
				if dur > budgetRemaining || dur > math.MaxInt64-curTick {
					outcome.HaltReason = "turn_budget"
					outcome.TicksAdvanced = curTick - startTick
					return nil
				}
			}

			attemptJSONBytes, _ := json.Marshal(attempt)
			result, err := o.applyEvent(ctx, worldID, actorID, attemptJSONBytes, curTick, curSeq)
			if err != nil {
				return fmt.Errorf("apply_event passthrough: %w", err)
			}
			if result["halt_reason"] == "gate_reject" {
				outcome.HaltReason = "gate_reject"
				outcome.TicksAdvanced = curTick - startTick
				return nil
			}
			evID, _ := result["event_id"].(string)
			if evID != "" {
				outcome.Committed = append(outcome.Committed, evID)
			}

			// Stage 4: advance the clock + consume the budget after a committed passthrough.
			if attempt.Type == "ActorMoved" {
				budgetRemaining -= dur // cumulative consumption (§6); under `none` it stays effectively unbounded
				curTick += dur
				if dur > 0 {
					// The clock advanced to a fresh tick — seq restarts at 0.
					curSeq = 0
				} else {
					// Zero-duration (same-location) move: the tick did NOT advance, so seq must keep
					// incrementing or the next event collides with the move on (tick,seq).
					curSeq++
				}
			} else {
				// Non-move passthrough: no computable duration in v1 → consumes 0 budget.
				curSeq++
			}

		default:
			// Adjudicated: AttributeChanged, OwnershipAccessChanged, EntityCreated,
			// EntityDestroyed, and now ALL six types via the v2 adjudicated path.
			ar, adjErr := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: actorID, Attempt: attempt}}, nil, curTick, curSeq, "")
			if adjErr != nil {
				return adjErr
			}
			outcome.Committed = append(outcome.Committed, ar.Committed...)
			if ar.Halt != "" {
				outcome.HaltReason = ar.Halt
				outcome.TicksAdvanced = curTick - startTick
				return nil
			}
			curSeq += ar.SeqAdvance
		}
	}

	outcome.HaltReason = "completed"
	outcome.TicksAdvanced = curTick - startTick
	return nil
}

// HeldOutcome is one pending held act read back from the held_outcome table: the NPC whose
// disruptive move was telegraphed (ActorID), her FULL typed intended act (Attempt), the held row's
// id (HeldID — flipped to 'resolved' inside the reaction's ruling tx), and the committed wind-up it
// is paired to (TelegraphEventID). It is LOOP STATE the world carries, not canon (RULINGS-2026-07-24 §3).
type HeldOutcome struct {
	HeldID           string
	ActorID          string
	Attempt          Attempt
	TelegraphEventID string
}

// pendingHeldOutcomes reads this world's pending held acts fresh from the table — no server memory,
// no session machine: the world (the table) carries the reaction state (RULINGS-2026-07-24 §3). The
// order is deterministic (created_tick, held_id) so simultaneous telegraphs enter the combined
// ruling in a stable order; a beat that errored after committing a telegraph leaves a pending row
// that this same query surfaces, so the very next input still resolves it (§3 — the wind-up IS canon).
func pendingHeldOutcomes(ctx context.Context, db *pgxpool.Pool, worldID string) ([]HeldOutcome, error) {
	rows, err := db.Query(ctx,
		`SELECT held_id::text, actor_id::text, attempt::text, telegraph_event_id::text
		 FROM held_outcome
		 WHERE world_id = $1 AND status = 'pending'
		 ORDER BY created_tick, held_id`,
		worldID)
	if err != nil {
		return nil, fmt.Errorf("query pending held_outcome: %w", err)
	}
	defer rows.Close()

	var out []HeldOutcome
	for rows.Next() {
		var h HeldOutcome
		var attemptJSON string
		if err := rows.Scan(&h.HeldID, &h.ActorID, &attemptJSON, &h.TelegraphEventID); err != nil {
			return nil, fmt.Errorf("scan held_outcome: %w", err)
		}
		if err := json.Unmarshal([]byte(attemptJSON), &h.Attempt); err != nil {
			return nil, fmt.Errorf("held_outcome %s attempt not valid Attempt JSON: %w", h.HeldID, err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// RunReactionBeat resolves the player's next input against the pending held act(s) (RULINGS-2026-07-24
// §2 + §3, and §9's collision rule). It is the reaction path RunBeat is not: the check that chose it
// already read the held rows fresh from the world (§3), so this method is handed them.
//
//	(1) chain[0].Type == "UNRESOLVED" → clarify halt; the holds STAY pending. The clarify answer is
//	    the NEXT input and re-enters this same reaction path — nothing about the held rows changes.
//	(2) The collision: ALL held acts + the reaction's FIRST action → ONE combined ruling
//	    (§2; N-ary via Task 1's adjudicate). NO cognition fires first — the collision IS the world's
//	    move, and depth-1 forbids re-reactions. The reaction's first action is adjudicated whatever
//	    its type (a Communicated "ask for ale" is not a passthrough here — it is the player's answer
//	    to the moment; §2 no-special-case). The held rows flip to 'resolved' INSIDE the ruling's tx
//	    (resolveHeldIDs). A bounce (or gate_reject) returns before/rolls back that tx untouched, so
//	    the holds STAY pending and the same next-input door reopens.
//	(3) The remainder chain[1:] runs as a NORMAL chain against the post-collision world — the shared
//	    runChain (world-first, generalized premise re-check, route, clock all apply). curSeq carries
//	    the ruling's seq advance so the remainder never collides on (tick,seq); the tick is unchanged
//	    (the combined ruling advances seq only).
//
// playerAnswer is the player's raw input text (beathandler passes in.Text). It is forwarded into
// the combined ruling ONLY when chain is empty — decompose emitted ZERO attempts, e.g. "I just
// watch" (RULINGS-2026-07-24 §7). When a first attempt exists, its own `stated` field already
// carries the player's words per §2, so forwarding the raw text too would double-inject; it is
// dropped in that branch.
func (o *Orchestrator) RunReactionBeat(ctx context.Context, worldID, playerID string, chain []Attempt, held []HeldOutcome, startTick int64, playerAnswer string) (BeatOutcome, error) {
	outcome := BeatOutcome{
		Committed:            []string{},
		UnresolvedCandidates: []string{},
		Telegraphs:           []string{},
	}

	// (1) UNRESOLVED first action → clarify halt; holds stay pending.
	if len(chain) > 0 && chain[0].Type == "UNRESOLVED" {
		outcome.HaltReason = "unresolved"
		outcome.UnresolvedCandidates = chain[0].CandidateIDs
		outcome.TicksAdvanced = 0
		return outcome, nil
	}

	// (2) Combined ruling: held acts (deterministic order) + the reaction's first action.
	set := make([]ActorAttempt, 0, len(held)+1)
	heldIDs := make([]string, 0, len(held))
	for _, h := range held {
		set = append(set, ActorAttempt{ActorID: h.ActorID, Attempt: h.Attempt})
		heldIDs = append(heldIDs, h.HeldID)
	}
	// answer rides into the ruling ONLY on the empty-chain branch (§7); a non-empty chain's first
	// attempt already carries the player's words in its own `stated` field (§2 — no double-inject).
	answer := ""
	if len(chain) > 0 {
		set = append(set, ActorAttempt{ActorID: playerID, Attempt: chain[0]})
	} else {
		answer = playerAnswer
	}

	ar, err := o.adjudicate(ctx, worldID, set, heldIDs, startTick, 0, answer)
	if err != nil {
		return outcome, err
	}
	outcome.Committed = append(outcome.Committed, ar.Committed...)
	if ar.Halt != "" {
		// bounce | ruled_event_rejected — the ruling's tx (which also carries the resolveHeldIDs
		// UPDATE) never committed, so the holds STAY pending. Nothing landed.
		outcome.HaltReason = ar.Halt
		outcome.TicksAdvanced = 0
		return outcome, nil
	}

	// (3) The remainder runs as a normal chain against the post-collision world — under the same §6
	// beat budget, read from the scene's current tension at the reaction player's location. The
	// combined ruling advanced seq only (tick unchanged), so the remainder shares the beat's budget.
	if len(chain) > 1 {
		budgetRemaining, err := o.beatBudgetSeconds(ctx, worldID, playerID)
		if err != nil {
			return outcome, fmt.Errorf("reaction beat budget: %w", err)
		}
		if err := o.runChain(ctx, worldID, playerID, chain[1:], startTick, ar.SeqAdvance, budgetRemaining, &outcome); err != nil {
			return outcome, err
		}
		return outcome, nil
	}

	outcome.HaltReason = "completed"
	outcome.TicksAdvanced = 0
	return outcome, nil
}

// worldFirstResult is what the world-first stage produced for one attempt: the NPC events it
// committed, the wind-up strings it telegraphed (turned into held outcomes), whether a held-outcome
// row was written, and how many (tick,seq) slots it consumed.
type worldFirstResult struct {
	Committed         []string
	TelegraphedStated []string
	HeldWritten       bool
	SeqAdvance        int
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
	// The scene roster + the imminent actor's label are relabeled PER SEAT (§3 naming reach): the batch
	// shows a name only if every batch mind shares it (else a descriptor); each isolated NPC reads the
	// room as SHE knows it. ids never change — only the labels the seat is shown. Computed at each call
	// site below; `present` is the full roster to relabel.

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
		// Relabel the shared scene + imminent line by the batch intersection (§3/§5): a name appears
		// only if EVERY batch mind knows the same one, else a descriptor.
		bLabels, err := o.batchDisplayLabels(ctx, worldID, batchIDs, present)
		if err != nil {
			return res, fmt.Errorf("batch display labels: %w", err)
		}
		bImminent := imminentLabel(bLabels, playerID)
		prompt := buildBatchPrompt(relabelScene(scene, bLabels), minds, moment, bImminent, attempt)
		raw, genErr := o.CognitionBatch.Generate(ctx, GenRequest{Schema: json.RawMessage(npcAttemptsSchemaJSON), Prompt: prompt})
		// A Generate or decode failure degrades DULL (the batch minds do nothing this moment) but is
		// no longer silent: log it so a mute room is diagnosable. Behavior unchanged — still skipped.
		if genErr != nil {
			log.Printf("worldFirst: batch cognition Generate failed (seat=%s, actors=%v, imminent=%s): %v", o.CognitionBatch.Name(), batchIDs, bImminent, genErr)
		} else if decisions, decErr := DecodeAndValidateNPCDecisions(raw, batchIDs); decErr != nil {
			// Validated against the BATCH ids only — a decision for an isolated NPC (or the player)
			// is rejected by DecodeAndValidateNPCDecisions' allowlist (non-present-for-this-call).
			log.Printf("worldFirst: batch cognition decode/validate failed (seat=%s, actors=%v, imminent=%s): %v", o.CognitionBatch.Name(), batchIDs, bImminent, decErr)
		} else {
			advance, applyErr := o.applyNPCDecisions(ctx, worldID, decisions, tick, localSeq, &res)
			if applyErr != nil {
				return res, applyErr
			}
			localSeq += advance
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
			// Relabel the scene + imminent line as THIS NPC knows it (§3): her own display names.
			iLabels, err := o.displayLabels(ctx, worldID, npcID, present)
			if err != nil {
				return res, fmt.Errorf("isolated display labels %s: %w", npcID, err)
			}
			iImminent := imminentLabel(iLabels, playerID)
			prompt := buildIsolatedPrompt(relabelScene(scene, iLabels), minds[0], private, moment, iImminent, attempt)
			raw, genErr := o.CognitionIsolated.Generate(ctx, GenRequest{Schema: json.RawMessage(npcAttemptsSchemaJSON), Prompt: prompt})
			// Degrade DULL (this secret-holder does nothing this moment) but observable — log the
			// swallowed failure. Behavior unchanged: still a continue.
			if genErr != nil {
				log.Printf("worldFirst: isolated cognition Generate failed (seat=%s, actor=%s, imminent=%s): %v", o.CognitionIsolated.Name(), npcID, iImminent, genErr)
				continue
			}
			// Validated against EXACTLY her own id — her call may speak only for her.
			decisions, decErr := DecodeAndValidateNPCDecisions(raw, []string{npcID})
			if decErr != nil {
				log.Printf("worldFirst: isolated cognition decode/validate failed (seat=%s, actor=%s, imminent=%s): %v", o.CognitionIsolated.Name(), npcID, iImminent, decErr)
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
				ar, err := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: dec.ActorID, Attempt: dec.Reaction.Attempt}}, nil, tick, localSeq, "")
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
		// Floor/premise parity with apply_event/apply_ruled_event's ActorMoved accessibility floor
		// (§5.3, via fn_actor_move_permitted): the destination must still exist AND a Portal must
		// permit here→dest (open ∧ ¬locked) — UNLESS it is a SAME-SCENE move (here == dest, or the
		// actor has no origin yet), which is not a traversal and needs no portal. Portal is
		// accessibility, NOT geometry: this mirror never consults fn_distance/coordinates.
		if ok, err := exists(a.ToLocationID, "location"); err != nil || !ok {
			return ok, err
		}
		here, err := o.actorLocation(ctx, worldID, actorID)
		if err != nil {
			return false, err
		}
		if here == "" || here == a.ToLocationID {
			return true, nil // same-scene / no origin — not a traversal, no portal required
		}
		return o.fnPortalPermits(ctx, worldID, here, a.ToLocationID)
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

// fnPortalPermits calls fn_portal_permits(world, from, to): TRUE iff a Portal artifact connecting
// from↔to permits passage (open ∧ ¬locked, §5.3 v1). It backs the premiseHolds ActorMoved mirror so
// the Go premise re-check enforces the same accessibility floor as the SQL twins. Portal is
// accessibility, NOT geometry — this never touches fn_distance/coordinates.
func (o *Orchestrator) fnPortalPermits(ctx context.Context, worldID, fromLoc, toLoc string) (bool, error) {
	if fromLoc == "" || toLoc == "" {
		return false, nil
	}
	var ok bool
	if err := o.DB.QueryRow(ctx,
		`SELECT fn_portal_permits($1, $2::uuid, $3::uuid)`,
		worldID, fromLoc, toLoc).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

// fnMoveDurationActor calls fn_move_duration_actor(world, actor, from, to) and returns the tick count.
// The actor supplies the effective speed (movement type + active statuses); distance is between the two
// locations. Speed 0 (e.g. encumbered, -100%) → max bigint, so an impossible move never fits the budget.
func (o *Orchestrator) fnMoveDurationActor(ctx context.Context, worldID, actorID, fromLoc, toLoc string) (int64, error) {
	if fromLoc == "" || toLoc == "" {
		return 0, nil
	}
	var dur int64
	err := o.DB.QueryRow(ctx,
		`SELECT fn_move_duration_actor($1, $2::uuid, $3::uuid, $4::uuid)`,
		worldID, actorID, fromLoc, toLoc).Scan(&dur)
	return dur, err
}

// existingMovementTypes reads the world's committed movement vocabulary (movement_type_ids) as a set —
// the mint-ordering baseline for validateMints (§8): a modifier may reference a seeded/committed type, or
// one minted earlier in the SAME ruling (validateMints grows the set intra-slice). Read fresh (no cache)
// so a type minted by an earlier ruling this session is visible.
func (o *Orchestrator) existingMovementTypes(ctx context.Context, worldID string) (map[string]bool, error) {
	rows, err := o.DB.Query(ctx,
		`SELECT movement_type_id FROM movement_type WHERE world_id = $1`, worldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
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
// resolveHeldIDs are the held_outcome rows this ruling resolves: RunReactionBeat passes the pending
// holds' ids so commitRulingTx flips them 'resolved' inside the ruling's own tx (ruling + resolution
// land together or roll back together). It is empty on every other caller — a normal adjudicate
// resolves no holds — and the resolution UPDATE is skipped when empty.
//
// playerAnswer is empty on every call except RunReactionBeat's empty-chain branch
// (RULINGS-2026-07-24 §7) — see buildResolvePrompt for the rendering rule.
func (o *Orchestrator) adjudicate(ctx context.Context, worldID string, set []ActorAttempt, resolveHeldIDs []string, tick int64, curSeq int, playerAnswer string) (adjResult, error) {
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
	// The founder's combined ruling bounced with no trace (Defect D): each failed attempt and the final
	// bounce are now logged (seat + attempt actor ids + the error/violation text), matching worldFirst's
	// degrade-dull style. Observability only — no behavior change.
	actorIDs := make([]string, 0, len(set))
	for _, aa := range set {
		actorIDs = append(actorIDs, aa.ActorID)
	}
	prompt := buildResolvePrompt(sliceJSON, set, nil, playerAnswer)
	var ruling RulingV2
	var violations []string
	var decodeErr error
	var movementTypes map[string]bool // lazily fetched only when a ruling actually mints (§8)

	for retry := 0; retry < 2; retry++ {
		p := prompt
		if retry > 0 {
			var repairErrs []string
			if decodeErr != nil {
				repairErrs = []string{decodeErr.Error()}
			} else {
				repairErrs = violations
			}
			p = buildResolvePrompt(sliceJSON, set, repairErrs, playerAnswer)
		}

		rawRuling, genErr := o.Resolve.Generate(ctx, GenRequest{
			Schema: json.RawMessage(rulingV2SchemaJSON),
			Prompt: p,
		})
		if genErr != nil {
			log.Printf("adjudicate: resolve Generate failed (seat=%s, actors=%v, attempt=%d/2): %v", o.Resolve.Name(), actorIDs, retry+1, genErr)
			decodeErr = genErr
			violations = nil
			continue
		}

		ruling, decodeErr = DecodeAndValidateRulingV2(rawRuling)
		if decodeErr != nil {
			log.Printf("adjudicate: ruling decode/validate failed (seat=%s, actors=%v, attempt=%d/2): %v", o.Resolve.Name(), actorIDs, retry+1, decodeErr)
			violations = nil
			continue
		}

		violations = verdictRuling(ruling, sliceIDs, ids)
		// Mint validation shares the verdict stage's repair-once-then-bounce (§8): fold any mint SHAPE +
		// BOUNDS + ordering violations into the SAME slice. Fetch the world's movement vocabulary once,
		// and only when this ruling actually mints (the common ruling mints nothing → zero DB overhead).
		if len(ruling.Outcome.Mints) > 0 {
			if movementTypes == nil {
				mt, mtErr := o.existingMovementTypes(ctx, worldID)
				if mtErr != nil {
					return adjResult{}, fmt.Errorf("existing movement types: %w", mtErr)
				}
				movementTypes = mt
			}
			violations = append(violations, verdictMints(ruling, movementTypes)...)
		}
		if len(violations) == 0 {
			break
		}
		// violations found — will retry once with repair prompt
		log.Printf("adjudicate: ruling verdict violations (seat=%s, actors=%v, attempt=%d/2): %v", o.Resolve.Name(), actorIDs, retry+1, violations)
		decodeErr = nil
	}

	// After two attempts, if still broken → bounce.
	if decodeErr != nil || len(violations) > 0 {
		log.Printf("adjudicate: bounce after repair (seat=%s, actors=%v)", o.Resolve.Name(), actorIDs)
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
// resolveHeldIDs are the pending holds a reaction ruling resolves (empty for a normal ruling); the
// held-resolution UPDATE runs inside this same tx so it commits atomically with the ruling, and is
// skipped entirely when empty so a hold-free ruling issues no stray statement.
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

	// Mint persistence (§8): each accepted mint becomes ONE logged row with provenance = this ruling's
	// event (blast radius = one row, net 2), persisted in the SAME tx as the events/writes, AFTER verdict
	// passed. committed[0] is the ruling event id every mint is audit-trailed to (net 3) — the same anchor
	// apply_attribute_writes uses. Guarded by len(committed)>0: a resolved ruling always has ≥1 event, and
	// mints only ride resolved rulings. apply_mint RAISEs on an artifact-shaped mint (persistence of a
	// typed artifact into entity_registry+state is not yet specified — see task-6 report §escalation),
	// which rolls back the whole ruling — fail-safe: a well-formed-but-unpersistable mint corrupts nothing.
	if len(ruling.Outcome.Mints) > 0 && len(committed) > 0 {
		for i, mint := range ruling.Outcome.Mints {
			if _, err := tx.Exec(ctx,
				`SELECT apply_mint($1::uuid, $2::uuid, $3::jsonb)`,
				worldID, committed[0], string(mint)); err != nil {
				return nil, 0, fmt.Errorf("apply_mint %d: %w", i, err)
			}
		}
	}

	// Held-outcome resolution: mark the named held_outcome rows 'resolved' inside this tx so the
	// resolution lands atomically with the ruling (RunReactionBeat's combined ruling). Skipped when
	// empty — a normal ruling resolves no holds, so the UPDATE never runs with no ids.
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
