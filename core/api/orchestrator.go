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
// Stage 4: Advance clock — ActorMoved adds fn_move_duration_actor ticks; every other non-move (passthrough or adjudicated) adds its duration_class seconds (fn_duration_class_seconds).
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
	Committed            []string      `json:"committed"`
	HaltReason           string        `json:"halt_reason"`
	TicksAdvanced        int64         `json:"ticks_advanced"`
	UnresolvedCandidates []string      `json:"unresolved_candidates"`
	Telegraphs           []string      `json:"telegraphs"`
	QueryAnswers         []QueryAnswer `json:"query_answers,omitempty"`
}

// QueryAnswer is one QUERY element's read-only answer (Grounded Reasoning / Unit 2): the player's
// question (Stated) paired with the PERCEIVED fact sheet computed for its targets. A QUERY is not an
// action — asking is not acting (RULINGS-2026-07-23 §3) — so this carries NO canon: it is accumulated
// on the outcome and handed to the narrator, who phrases the answer in-world from the facts. FactSheet
// is the raw fn_fact_sheet jsonb (perception-scoped, p_truth_side=false), embedded verbatim in the
// narrate prompt so a withheld/absent fact stays withheld ("you can't tell from here").
type QueryAnswer struct {
	Stated    string          `json:"stated"`
	FactSheet json.RawMessage `json:"fact_sheet"`
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
//
// trace is the Unit 3 reasoning log (nil except in debug): it is threaded through the whole per-beat
// call tree so each stage appends what it computed, and is nil-safe end-to-end (a nil trace's append
// methods are no-ops, so a non-debug beat pays ~zero and behaves byte-identically).
func (o *Orchestrator) RunBeat(ctx context.Context, worldID, actorID string, chain []Attempt, startTick int64, trace *BeatTrace) (BeatOutcome, error) {
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
	if err := o.runChain(ctx, worldID, actorID, chain, startTick, 0, budgetRemaining, &outcome, trace); err != nil {
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
// never blocks. Non-move steps (passthrough OR adjudicated) subtract their decomposer-tagged
// duration_class → seconds the SAME way (fn_duration_class_seconds; an empty class floors to
// "instant" — Task 3, Living World); Way-A long-action accumulation across multiple beats is a later
// station, §6.
func (o *Orchestrator) runChain(ctx context.Context, worldID, actorID string, chain []Attempt, startTick int64, startSeq int, budgetRemaining int64, outcome *BeatOutcome, trace *BeatTrace) error {
	curTick := startTick
	curSeq := startSeq

	// Living World / Task 9: scene = the acting actor's current location, read ONCE per beat
	// (task-9-brief ambiguity resolution #3 explicitly allows this), LAZILY on first use — a chain that
	// never reaches a committed clock-advancing attempt (an empty "I watch" beat, or an all-QUERY chain)
	// never issues the extra read. Every committed attempt's world's-turn call below (runWorldTurn) hands
	// the World Actor this same cached scene.
	var scene string
	var sceneLoaded bool
	ensureScene := func() (string, error) {
		if !sceneLoaded {
			s, locErr := o.actorLocation(ctx, worldID, actorID)
			if locErr != nil {
				return "", fmt.Errorf("world's turn: actor location: %w", locErr)
			}
			scene = s
			sceneLoaded = true
		}
		return scene, nil
	}

	for _, attempt := range chain {
		// ── Stage 5: UNRESOLVED sentinel (check first — no commit, just halt) ────
		if attempt.Type == "UNRESOLVED" {
			outcome.HaltReason = "unresolved"
			outcome.UnresolvedCandidates = attempt.CandidateIDs
			outcome.TicksAdvanced = curTick - startTick
			return nil
		}

		// ── QUERY: a read-only question (Grounded Reasoning / Unit 2) ────────────
		// A QUERY is NOT an action — asking is not acting (RULINGS-2026-07-23 §3). Build the PERCEIVED
		// fact sheet for its targets (viewer = the acting player, perception-scoped truth_side=false —
		// the narrator is a character-mind seat) and accumulate a QueryAnswer for the narrator. It does
		// NOT commit, does NOT advance the clock, and NEVER runs Stage 1–4 (world-first / premise /
		// route / adjudicate) — a QUERY never reaches the referee. The chain CONTINUES so a mixed
		// [action, QUERY] beat commits its action AND answers the question in one narration.
		if attempt.Type == "QUERY" {
			fs, fsErr := o.factSheetJSON(ctx, worldID, actorID, attempt.QueryTargetIDs, false)
			if fsErr != nil {
				return fmt.Errorf("query fact sheet: %w", fsErr)
			}
			outcome.QueryAnswers = append(outcome.QueryAnswers, QueryAnswer{Stated: attempt.Stated, FactSheet: json.RawMessage(fs)})
			continue
		}

		// ── Stage 1: World-first (cognition seats) ───────────────────────────────
		// Every present NPC's decision runs through the SAME pipeline as the player's — no bypass.
		// worldFirst does the §5 mechanical split (batch = public moment only; secret-holders
		// isolated) and commits each NPC decision (passthrough via apply_event, adjudicated via
		// o.adjudicate). It returns how many (tick,seq) slots the NPC commits consumed.
		wf, err := o.worldFirst(ctx, worldID, actorID, attempt, curTick, curSeq, trace)
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
		// Living World / Task 9: captured before Stage 4 (below, in each branch) mutates curTick — the
		// world's-turn call needs the tick BEFORE this attempt advanced it.
		attemptTickBefore := curTick
		switch attempt.Type {
		case "ActorMoved", "Communicated", "ObjectRelocated":
			// Passthrough: commit directly via apply_event (gate enforces structural floor).

			// ── §6 tension budget: EVERY passthrough — move OR non-move — must FIT the remaining beat
			// budget BEFORE it commits (Task 3/Living World: non-move acts now cost world-time too, so
			// the world can move even during talk). A move's duration is real physics (§2:
			// distance(actor,target) / effective_speed(actor,'walk'); a -100% modifier → speed 0 →
			// max-bigint sentinel; the actor's origin is IMPLICIT, the target supplies the destination,
			// Task 8). A non-move's duration is its decomposer-tagged duration_class → seconds
			// (fn_duration_class_seconds, per-world retunable, Task 1/Task 2); an EMPTY class (legacy
			// input, or simply no estimate) floors to "instant" so even stillness ticks. If the duration
			// exceeds what remains, the chain halts turn_budget and this step does NOT commit — the
			// prefix stands (§6 example: three 12 s moves in a 30 s beat → 12,24 commit, 36 rejects; the
			// same shape now applies to a non-move that can't fit — this is the honest INTERIM
			// over-budget behavior, no Journey/accumulation logic yet). The runaway/overflow guard
			// (dur > MaxInt64-curTick) is belt-and-suspenders and, under `none` (∞ budget) where the
			// first clause never fires, still catches the speed-0 sentinel.
			var dur int64
			if attempt.Type == "ActorMoved" {
				d, derr := o.fnMoveDurationActor(ctx, worldID, actorID, attempt.ToTargetID)
				if derr != nil {
					return fmt.Errorf("fn_move_duration_actor: %w", derr)
				}
				dur = d
			} else {
				d, derr := o.nonMoveDurationSeconds(ctx, worldID, attempt.DurationClass)
				if derr != nil {
					return fmt.Errorf("fn_duration_class_seconds: %w", derr)
				}
				dur = d
			}
			over := overBudget(dur, budgetRemaining, curTick)
			// Trace (Unit 3): capture the MOVE's physics + the §6 gate result. The perceived fact sheet
			// is a DEBUG-ONLY extra read, so it is gated on trace != nil — a non-debug beat (nil trace)
			// issues NO extra query and pays zero. A capture-read failure degrades (log + record without
			// the sheet) rather than failing the beat: the trace is a debug affordance, never
			// load-bearing for the commit. Non-move trace capture is not part of this task.
			if trace != nil && attempt.Type == "ActorMoved" {
				fs, fsErr := o.factSheetJSON(ctx, worldID, actorID, []string{attempt.ToTargetID}, false)
				if fsErr != nil {
					log.Printf("trace: move fact sheet failed (debug-only capture, degrading): %v", fsErr)
					fs = ""
				}
				trace.appendMove(attempt, fs, dur, budgetRemaining, !over)
			}
			if over {
				outcome.HaltReason = "turn_budget"
				outcome.TicksAdvanced = curTick - startTick
				return nil
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

			// Stage 4: advance the clock + consume the budget after a committed passthrough (move or
			// non-move alike — every beat costs world-time now, Task 3).
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

			// Living World / Task 9: the world's turn — fires due scheduled events, then (if nothing
			// medium/large already did) rolls the pressure tiers, after EVERY committed clock-advancing
			// attempt (task-9-brief ambiguity resolution #3). small/"" lets the chain run on; medium/large
			// applies the §5 cut — the beat ends HERE, discarding the rest of the chain, same shape as the
			// telegraph/turn_budget halts above.
			sceneID, sceneErr := ensureScene()
			if sceneErr != nil {
				return sceneErr
			}
			newSeq, cutBeat, wtErr := o.advanceWorldTurn(ctx, worldID, sceneID, attemptTickBefore, curTick, curSeq, outcome, trace)
			if wtErr != nil {
				return wtErr
			}
			curSeq = newSeq
			if cutBeat {
				outcome.HaltReason = "world_eruption"
				outcome.TicksAdvanced = curTick - startTick
				return nil
			}

		default:
			// Adjudicated: AttributeChanged, OwnershipAccessChanged, EntityCreated,
			// EntityDestroyed, and now ALL six types via the v2 adjudicated path.

			// ── §6 tension budget: an adjudicated non-move must ALSO fit the remaining beat budget
			// BEFORE it is adjudicated/committed — symmetric with the passthrough non-move branch above
			// (every non-move attempt now charges its class duration, whether routed via apply_event or
			// the referee; Task 3 review fix). Its duration is its decomposer-tagged duration_class →
			// seconds (fn_duration_class_seconds; an EMPTY class floors to "instant", same as the
			// passthrough branch). If the duration exceeds what remains, the chain halts turn_budget
			// WITHOUT calling adjudicate — the prefix stands, mirroring the pre-commit gate every other
			// branch in this loop already uses. Still no Journey/accumulation logic.
			dur, durErr := o.nonMoveDurationSeconds(ctx, worldID, attempt.DurationClass)
			if durErr != nil {
				return fmt.Errorf("fn_duration_class_seconds: %w", durErr)
			}
			if overBudget(dur, budgetRemaining, curTick) {
				outcome.HaltReason = "turn_budget"
				outcome.TicksAdvanced = curTick - startTick
				return nil
			}

			ar, adjErr := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: actorID, Attempt: attempt}}, nil, curTick, curSeq, "", nil, trace)
			if adjErr != nil {
				return adjErr
			}
			outcome.Committed = append(outcome.Committed, ar.Committed...)
			if ar.Halt != "" {
				outcome.HaltReason = ar.Halt
				outcome.TicksAdvanced = curTick - startTick
				return nil
			}

			// Stage 4: advance the clock + consume the budget after a committed adjudication (Task 3
			// review fix — symmetric with the passthrough branch).
			budgetRemaining -= dur
			curTick += dur
			if dur > 0 {
				// The clock advanced to a fresh tick — seq restarts at 0.
				curSeq = 0
			} else {
				curSeq += ar.SeqAdvance
			}

			// Living World / Task 9: the world's turn — symmetric with the passthrough branch above.
			sceneID, sceneErr := ensureScene()
			if sceneErr != nil {
				return sceneErr
			}
			newSeq, cutBeat, wtErr := o.advanceWorldTurn(ctx, worldID, sceneID, attemptTickBefore, curTick, curSeq, outcome, trace)
			if wtErr != nil {
				return wtErr
			}
			curSeq = newSeq
			if cutBeat {
				outcome.HaltReason = "world_eruption"
				outcome.TicksAdvanced = curTick - startTick
				return nil
			}
		}
	}

	// §6 / Task 3 floor: a beat where NOTHING advanced world-time — an empty "I watch" beat (no
	// attempts at all) or a chain of QUERY-only elements (QUERY is not an action and never reaches
	// Stage 3/4 — RULINGS-2026-07-23 §3 — so it cannot tick the clock itself) — still costs the
	// instant floor: stillness ticks too. This applies ONLY on the completed path, here at the tail
	// after every attempt has run without halting; every halt branch above (turn_budget, gate_reject,
	// premise_broken, telegraph, unresolved) returns before reaching this line, so a halted beat keeps
	// its own halt semantics untouched — the floor is never a consolation prize for a beat that stopped.
	if curTick == startTick {
		floor, floorErr := o.durationClassSeconds(ctx, worldID, "instant")
		if floorErr != nil {
			return fmt.Errorf("fn_duration_class_seconds (floor): %w", floorErr)
		}
		// Belt-and-suspenders overflow guard, matching the loop's dur > MaxInt64-curTick check: at
		// startTick == math.MaxInt64 (or within floor of it) curTick+floor would wrap. Clamp instead of
		// overflowing — this is defensive only, never expected to fire in practice.
		if floor > math.MaxInt64-curTick {
			curTick = math.MaxInt64
		} else {
			curTick += floor
		}
	}
	outcome.HaltReason = "completed"
	outcome.TicksAdvanced = curTick - startTick
	return nil
}

// nonMoveDurationSeconds resolves a non-move attempt's decomposer-tagged duration_class (Task 1's
// parse-shape enum instant|short|medium|long|extremely_long) to seconds via fn_duration_class_seconds
// (Task 1/Task 2), owning the ""→"instant" fallback (legacy input, or simply no estimate — Task 3) so
// runChain's passthrough and adjudicated Stage-3 branches share ONE lookup instead of two byte-identical
// copies. The move branch (ActorMoved) never calls this — its duration is real physics
// (fnMoveDurationActor), not a duration_class.
func (o *Orchestrator) nonMoveDurationSeconds(ctx context.Context, worldID, durationClass string) (int64, error) {
	class := durationClass
	if class == "" {
		class = "instant"
	}
	return o.durationClassSeconds(ctx, worldID, class)
}

// overBudget reports whether dur (seconds) does not fit the beat's remaining §6 budget — either it
// exceeds budgetRemaining outright, or curTick+dur would overflow int64 (the runaway/overflow guard,
// belt-and-suspenders under `none`'s math.MaxInt64 budget where the first clause alone never fires, and
// which also catches the speed-0 "blocked by arithmetic" sentinel, §2). Pure — no I/O, no receiver — so
// both the move and non-move gates in runChain's Stage 3 (and the passthrough/adjudicated non-move
// branches specifically) share the identical formula instead of each inlining its own copy.
func overBudget(dur, budgetRemaining, curTick int64) bool {
	return dur > budgetRemaining || dur > math.MaxInt64-curTick
}

// advanceWorldTurn runs the world's turn (runWorldTurn) after a committed clock-advancing attempt —
// fires due scheduled events, then (if nothing medium/large already did) rolls the pressure tiers — and
// reports whether the §5 cut applies (cutBeat, magnitude medium/large). It does NOT set
// outcome.HaltReason/TicksAdvanced or return early: the caller (runChain's passthrough and adjudicated
// Stage-4 blocks) owns the halt itself, exactly as before this extraction — this is a pure DRY
// extraction of the two blocks that used to be byte-identical in both branches (task-9-brief ambiguity
// resolution #3), including the "world's turn: %w" error-wrap text. sceneID is resolved by the caller's
// own ensureScene() call BEFORE invoking this helper, unchanged from before. newSeq is curSeq +
// runWorldTurn's own seqUsed — a direct drop-in replacement for the caller's prior `curSeq += seqUsed`.
func (o *Orchestrator) advanceWorldTurn(ctx context.Context, worldID, sceneID string, attemptTickBefore, curTick int64, curSeq int, outcome *BeatOutcome, trace *BeatTrace) (newSeq int, cutBeat bool, err error) {
	firedMag, seqUsed, wtErr := o.runWorldTurn(ctx, worldID, sceneID, attemptTickBefore, curTick, curSeq, outcome, trace)
	if wtErr != nil {
		return curSeq, false, fmt.Errorf("world's turn: %w", wtErr)
	}
	return curSeq + seqUsed, eruptionCutsBeat(firedMag), nil
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
func (o *Orchestrator) RunReactionBeat(ctx context.Context, worldID, playerID string, chain []Attempt, held []HeldOutcome, startTick int64, playerAnswer string, trace *BeatTrace) (BeatOutcome, error) {
	outcome := BeatOutcome{
		Committed:            []string{},
		UnresolvedCandidates: []string{},
		Telegraphs:           []string{},
	}

	// (0) Peel any LEADING QUERY elements — read-only, never adjudicated (Grounded Reasoning / Unit 2).
	// The reaction's first action is folded into the combined ruling below; a QUERY is not an action, so
	// a leading question must be answered here (PERCEIVED fact sheet, viewer = the player) and popped —
	// otherwise it would enter the ruling and reach the referee, breaking the invariant (asking is not
	// acting, RULINGS-2026-07-23 §3). Queries in the REMAINDER (chain[1:]) are handled by runChain below.
	for len(chain) > 0 && chain[0].Type == "QUERY" {
		fs, fsErr := o.factSheetJSON(ctx, worldID, playerID, chain[0].QueryTargetIDs, false)
		if fsErr != nil {
			return outcome, fmt.Errorf("reaction query fact sheet: %w", fsErr)
		}
		outcome.QueryAnswers = append(outcome.QueryAnswers, QueryAnswer{Stated: chain[0].Stated, FactSheet: json.RawMessage(fs)})
		chain = chain[1:]
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

	ar, err := o.adjudicate(ctx, worldID, set, heldIDs, startTick, 0, answer, nil, trace)
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
		if err := o.runChain(ctx, worldID, playerID, chain[1:], startTick, ar.SeqAdvance, budgetRemaining, &outcome, trace); err != nil {
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
func (o *Orchestrator) worldFirst(ctx context.Context, worldID, playerID string, attempt Attempt, tick int64, seq int, trace *BeatTrace) (worldFirstResult, error) {
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
		// PERCEIVED fact sheet (Grounded Reasoning / Unit 1 → cognition), computed ONCE for the acting
		// PLAYER as viewer: spatial facts are public — everyone in the room shares the distances — so the
		// batch (every non-flagged mind) reads the same player-viewer sheet. truth_side=FALSE: the minds
		// are perception-walled (§5/§9); the ONLY gate in v1 is a closed container's withheld contents
		// (physics facts, never perception records — so the sheet cannot carry a secret into the batch).
		batchSheet, fsErr := o.factSheetJSON(ctx, worldID, playerID, actionIDs, false)
		if fsErr != nil {
			return res, fmt.Errorf("batch fact sheet: %w", fsErr)
		}
		prompt := buildBatchPrompt(relabelScene(scene, bLabels), minds, moment, bImminent, attempt, batchSheet)
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
			// Trace (Unit 3): record each batch NPC's world-first decision (none/commit/telegraph).
			trace.appendDecisions(decisions)
			advance, applyErr := o.applyNPCDecisions(ctx, worldID, decisions, tick, localSeq, &res, trace)
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
			// This flagged NPC's OWN perceived fact sheet: viewer = HER, so the spatial facts read as SHE
			// sees the moment (her own distances/reachability), truth_side=FALSE (walled). Computed per
			// isolated NPC, as needed — the batch's player-viewer sheet is not hers to reuse.
			isoSheet, fsErr := o.factSheetJSON(ctx, worldID, npcID, actionIDs, false)
			if fsErr != nil {
				return res, fmt.Errorf("isolated fact sheet %s: %w", npcID, fsErr)
			}
			prompt := buildIsolatedPrompt(relabelScene(scene, iLabels), minds[0], private, moment, iImminent, attempt, isoSheet)
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
			// Trace (Unit 3): record this isolated NPC's world-first decision (none/commit/telegraph).
			trace.appendDecisions(decisions)
			advance, applyErr := o.applyNPCDecisions(ctx, worldID, decisions, tick, localSeq, &res, trace)
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
func (o *Orchestrator) applyNPCDecisions(ctx context.Context, worldID string, decisions []NPCDecision, tick int64, startSeq int, res *worldFirstResult, trace *BeatTrace) (int, error) {
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
				ar, err := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: dec.ActorID, Attempt: dec.Reaction.Attempt}}, nil, tick, localSeq, "", nil, trace)
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
		// (§5.3, via fn_actor_move_permitted): the TARGET (any positioned entity — a location, an object
		// like the bar, an actor) must still exist AND, for a CROSS-LOCATION move, a Portal must permit
		// here→(the target's scene) (open ∧ ¬locked) — UNLESS it is a SAME-SCENE move (the target's
		// scene == here, or the actor has no origin yet), which is not a traversal and needs no portal.
		// The scene is resolved via fn_target_position (the SAME resolution the SQL twins commit). Portal
		// is accessibility, NOT geometry: this mirror never consults fn_distance/coordinates.
		if ok, err := exists(a.ToTargetID, ""); err != nil || !ok {
			return ok, err
		}
		here, err := o.actorLocation(ctx, worldID, actorID)
		if err != nil {
			return false, err
		}
		scene, err := o.fnTargetScene(ctx, worldID, a.ToTargetID)
		if err != nil {
			return false, err
		}
		if here == "" || here == scene {
			return true, nil // same-scene / no origin — not a traversal, no portal required
		}
		return o.fnPortalPermits(ctx, worldID, here, scene)
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

// fnMoveDurationActor calls fn_move_duration_actor(world, actor, target) and returns the tick count.
// Task 8: the 3-arg TARGET form — the actor's origin is implicit (its own coordinate/scene), the target
// (a location, an object, an actor) supplies the destination; distance = fn_distance(actor, target). The
// actor supplies the effective speed (movement type + active statuses). Speed 0 (e.g. encumbered, -100%)
// → max bigint, so an impossible move never fits the budget.
func (o *Orchestrator) fnMoveDurationActor(ctx context.Context, worldID, actorID, target string) (int64, error) {
	if target == "" {
		return 0, nil
	}
	var dur int64
	err := o.DB.QueryRow(ctx,
		`SELECT fn_move_duration_actor($1, $2::uuid, $3::uuid)`,
		worldID, actorID, target).Scan(&dur)
	return dur, err
}

// fnTargetScene resolves ANY positioned target to its containing SCENE via fn_target_position — a
// location resolves to itself, an actor/artifact to its attrs.location_id. It backs the premiseHolds
// ActorMoved mirror so the Go premise re-check gates the portal on the SAME scene the SQL commit twins
// resolve. Empty target → "". A NULL scene (target with no location) coalesces to "".
func (o *Orchestrator) fnTargetScene(ctx context.Context, worldID, target string) (string, error) {
	if target == "" {
		return "", nil
	}
	var scene string
	err := o.DB.QueryRow(ctx,
		`SELECT COALESCE(tp.scene::text, '') FROM fn_target_position($1, $2::uuid) tp`,
		worldID, target).Scan(&scene)
	return scene, err
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

// applyEvent calls apply_event on the pool and returns the result as a map.
func (o *Orchestrator) applyEvent(ctx context.Context, worldID, actorID string, attemptJSON []byte, tick int64, seq int) (map[string]any, error) {
	return applyEventOnQuerier(ctx, o.DB, worldID, actorID, attemptJSON, tick, seq)
}

// applyEventOnQuerier is the shared implementation used by both the pool and tx paths — the twin of
// applyRuledEventOnQuerier, so a world-sourced passthrough commit can share a transaction with the
// bookkeeping row that records it.
func applyEventOnQuerier(ctx context.Context, q dbQuerier, worldID, actorID string, attemptJSON []byte, tick int64, seq int) (map[string]any, error) {
	var resultJSON []byte
	err := q.QueryRow(ctx,
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

// factSheetJSON returns the action-scoped fact sheet computed by fn_fact_sheet (Grounded Reasoning /
// Unit 1) for the involved entities relative to viewer, as the raw jsonb rendered to a string — ready
// to embed verbatim in a seat prompt. The engine computes the deterministic physics facts (distance,
// move duration, reachability, weight, lock/open state) AMONG the involved entities so the seat reasons
// FROM computed truth instead of inventing it ("the reasoning was detached from the math"). truthSide
// selects the referee flavor (true — never walled, RULINGS-2026-07-23 §9) vs the perception-scoped
// character-mind flavor (false, for cognition/narrate — Units 2/3).
func (o *Orchestrator) factSheetJSON(ctx context.Context, worldID, viewer string, involved []string, truthSide bool) (string, error) {
	var raw []byte
	if err := o.DB.QueryRow(ctx,
		`SELECT fn_fact_sheet($1::uuid, $2::uuid, $3::uuid[], $4)`,
		worldID, viewer, involved, truthSide).Scan(&raw); err != nil {
		return "", fmt.Errorf("fn_fact_sheet: %w", err)
	}
	return string(raw), nil
}

// durationClassSeconds reads fn_duration_class_seconds: this world's seconds for a non-move
// duration_class (Task 1's parse-shape enum instant|short|medium|long|extremely_long), retunable
// per-world via the duration_class_seconds table with a built-in fallback so the lookup never fails
// closed on an unseeded world (Task 2). Mirrors factSheetJSON's Go→SQL pattern — the small helper
// runChain's Stage-4 clock advance and the Step-5 stillness floor both call (Task 3).
func (o *Orchestrator) durationClassSeconds(ctx context.Context, worldID, class string) (int64, error) {
	var s int64
	err := o.DB.QueryRow(ctx, `SELECT fn_duration_class_seconds($1,$2)`, worldID, class).Scan(&s)
	return s, err
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
func (o *Orchestrator) adjudicate(ctx context.Context, worldID string, set []ActorAttempt, resolveHeldIDs []string, tick int64, curSeq int, playerAnswer string, postCommit postCommitFn, trace *BeatTrace) (adjResult, error) {
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

	// Grounded Reasoning / Unit 1 → adjudication: hand the referee the engine's COMPUTED physics facts
	// so it reasons FROM measured truth (distance/duration/reachability) instead of guessing. TRUTH-SIDE
	// (p_truth_side=true): the referee is never walled (RULINGS-2026-07-23 §9). Viewer = the primary
	// acting actor (set[0]); involved = the SAME ids gathered for the slice, so the facts cover exactly
	// the entities in the ruling. budget_remaining stays null in v1 — it is beat-state (runChain's
	// tension logic), not an entity fact, and the entity physics facts are the adjudication value; wiring
	// the live budget in is a clean follow-up. Skipped only for the degenerate empty-set/empty-ids call
	// (nothing to compute); a fact-sheet read failure is a hard error like the gather above.
	factSheet := ""
	if len(set) > 0 && set[0].ActorID != "" && len(ids) > 0 {
		fs, fsErr := o.factSheetJSON(ctx, worldID, set[0].ActorID, ids, true)
		if fsErr != nil {
			return adjResult{}, fmt.Errorf("fact sheet: %w", fsErr)
		}
		factSheet = fs
	}

	prompt := buildResolvePrompt(sliceJSON, factSheet, set, nil, playerAnswer)
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
			p = buildResolvePrompt(sliceJSON, factSheet, set, repairErrs, playerAnswer)
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

	// After two attempts, if still broken → bounce (no valid ruling to trace).
	if decodeErr != nil || len(violations) > 0 {
		log.Printf("adjudicate: bounce after repair (seat=%s, actors=%v)", o.Resolve.Name(), actorIDs)
		return adjResult{Halt: "bounce"}, nil
	}

	// Trace (Unit 3): capture the referee's own reasoning → therefore → outcome + the TRUTH-SIDE fact
	// sheet it reasoned from (never walled — this is why the trace is debug-only). Recorded here, once a
	// valid ruling is in hand, so an `impossible` verdict is captured too (before its bounce below).
	trace.appendRuling(actorIDs, factSheet, ruling)

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
	committed, seqAdvance, commitErr := o.commitRulingTx(ctx, tx, worldID, ruling, resolveHeldIDs, tick, curSeq, postCommit)
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
//
// postCommit is the caller's own bookkeeping for this commit (ledger.go's postCommitFn — a world-sourced
// commit's pending-row flip or fire-log row). It runs last, inside this same tx, on the SAME terms as
// resolveHeldIDs: an error rolls the whole ruling back. Nil on every non-world caller.
func (o *Orchestrator) commitRulingTx(ctx context.Context, tx pgx.Tx, worldID string, ruling RulingV2, resolveHeldIDs []string, tick int64, curSeq int, postCommit postCommitFn) ([]string, int, error) {
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

	if postCommit != nil {
		if hookErr := postCommit(ctx, tx, committed); hookErr != nil {
			return nil, 0, fmt.Errorf("post-commit hook: %w", hookErr)
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
