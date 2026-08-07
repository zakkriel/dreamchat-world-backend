# Rung 0 — Close the three Living-World gates (Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the three deferrals the Living World recorded as gated **before the real (non-fake) World Actor driver runs at play**, so the Journey's road-creation can be authored by a live driver without a known lost-write, a known truth-leak, or a known silent skip.

**Architecture:** Three independent fixes to shipped units, in dependency order. (1) A single `postCommitFn` hook threaded through the existing commit paths makes each world-sourced commit atomic with its own bookkeeping row — the pattern `commitRulingTx`/`resolveHeldIDs` already established for held outcomes, generalised. (2) A runtime scene check in `runWorldActor` enforces at execution what `world_actor.txt` currently only asks for in the prompt. (3) The floor-window clock crossing gets its world's turn, closing the one path where world-time advances with no world's turn.

**Tech Stack:** Go (`core/api`, package `main`), pgx v5, plpgsql + dbmate migrations, pgTAP. Test commands: `make reset` (clean+migrate+seed), `make test` (pgTAP), `cd core/api && go test ./...`, `make schema-check`.

## Global Constraints

Every task's requirements implicitly include these:

- **Branch:** `rung0/living-world-gates`, off `feat/living-world` (`97d3dc4`). One PR.
- **D-1: canon flows only through the SQL apply paths.** No task adds a second writer, and no task lets anything bypass the gate.
- **No behaviour change to the happy path.** All three fixes are closures of known holes; the existing 362 pgTAP assertions and the whole Go suite must pass unchanged. A test that needs editing is a signal the fix is wrong — stop and report, do not "fix" the test.
- **No Journey logic.** No progress, thresholds, legs, extents, or place creation. Those are rungs 1–2.
- **No migration is required by this plan.** If you believe you need one, stop and report — `world_eruption` and `pending_event` already exist with the needed columns.
- **The fake driver stays the CI driver.** Nothing here wires a live model.
- **Modular mandate:** the hook is ONE mechanism used by two callers, not two bespoke transaction blocks.
- **Design source:** `docs/superpowers/specs/2026-08-07-journey-design.md` §5 rung 0. The deferrals themselves are recorded verbatim in `.git/sdd/progress.md` under the whole-branch review.

**Confirmed interfaces from the real code (consume these by exact name):**

- `func (o *Orchestrator) commitWorldPayload(ctx context.Context, worldID, actorID string, attempt Attempt, tick int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (eventIDs []string, seqAdvance int, halt string, err error)` — `ledger.go:45`
- `func (o *Orchestrator) fireDuePending(ctx context.Context, worldID string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, seqUsed int, err error)` — `ledger.go:131`
- `func (o *Orchestrator) runWorldTurn(ctx context.Context, worldID, scene string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, seqUsed int, err error)` — `worldturn.go:39`
- `func (o *Orchestrator) runWorldActor(ctx context.Context, worldID, scene, size string, now int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (eventID string, seqUsed int, err error)` — `worldactor.go:46`
- `func (o *Orchestrator) adjudicate(ctx context.Context, worldID string, set []ActorAttempt, resolveHeldIDs []string, tick int64, curSeq int, playerAnswer string, trace *BeatTrace) (adjResult, error)` — `orchestrator.go:1181`
- `func (o *Orchestrator) commitRulingTx(ctx context.Context, tx pgx.Tx, worldID string, ruling RulingV2, resolveHeldIDs []string, tick int64, curSeq int) ([]string, int, error)` — `orchestrator.go:1380`
- `func (o *Orchestrator) applyEvent(ctx context.Context, worldID, actorID string, attemptJSON []byte, tick int64, seq int) (map[string]any, error)` — `orchestrator.go:1094`
- `func applyRuledEventOnQuerier(ctx context.Context, q dbQuerier, worldID string, evt RuledEventV2, tick int64, seq int, origin string) (map[string]any, error)` — `orchestrator.go:1462` (**the pattern to copy for `applyEvent`**)
- `type dbQuerier interface { QueryRow(ctx context.Context, sql string, args ...any) pgx.Row }` — `orchestrator.go:1455`
- `func (o *Orchestrator) actorLocation(ctx context.Context, worldID, actorID string) (string, error)` — `orchestrator.go:992`
- `func (o *Orchestrator) fnTargetScene(ctx context.Context, worldID, target string) (string, error)` — `orchestrator.go:1060`
- `func (o *Orchestrator) advanceWorldTurn(ctx context.Context, worldID, sceneID string, attemptTickBefore, curTick int64, curSeq int, outcome *BeatOutcome, trace *BeatTrace) (newSeq int, cutBeat bool, err error)` — `orchestrator.go:433`
- `ensureScene := func() (string, error)` — `orchestrator.go:129`, a closure defined **before** `runChain`'s loop and therefore in scope at the tail
- Test helpers: `testPool(t)` (`viewer_test.go:15`), `wtOrchestrator(pool)` (`orchestrator_worldtime_test.go:57`), `wtBaseTick(t, ctx, pool)` (`:27`), `wtForceTierFires` / `wtDisableWorldActor` / `wtEruptionRowForEvent` / `wtDeleteEruptionRows` (`worldturn_test.go:66/115/155/187`)
- Test ids: `dlWorldID = "22222222-…-2222"` (`beathandler_test.go:321`), `dlKadeID = "2ac70000-…-a1"`, `wtMaraID = "2ac70000-…-a2"`, `wtTavernID = "210c0000-…-d1"`, Dock Street = `"210c0000-0000-0000-0000-0000000000d2"` (`seed_drowned_lantern.sql:73`)

---

### Task 1: Atomic commit + bookkeeping (deferral A)

**The bug.** Two world-sourced commits write a bookkeeping row in a *separate* statement from the canon commit it describes:

- `ledger.go:193` — `UPDATE pending_event SET status='fired'` after `commitWorldPayload` returned. Crash between the two and the row stays `pending`, so the same event fires again on the next crossing.
- `worldturn.go:135` — `INSERT INTO world_eruption …` after `runWorldActor` returned. Crash between the two and the eruption is in canon with **no fire-log row** — and because `fn_pressure_chance` derives pressure from `max(fired_tick)`, the tier never drains. That is the "lost drain" the whole-branch review called the real exposure.

Both carry a `TODO` at the site. This task removes both TODOs by making each pair one transaction.

**Files:**
- Modify: `core/api/orchestrator.go:1094` (split `applyEvent` into a querier-shared form), `:1181` (`adjudicate` gains the hook param), `:1380` (`commitRulingTx` runs the hook in-tx)
- Modify: `core/api/ledger.go:45` (`commitWorldPayload` gains the hook; passthrough branch becomes a tx), `:131` (`fireDuePending` passes the flip as the hook)
- Modify: `core/api/worldactor.go:46` (`runWorldActor` threads the hook)
- Modify: `core/api/worldturn.go:128-139` (build the fire-log hook; delete the standalone INSERT)
- Modify: every other `adjudicate(` call site — `orchestrator.go` (`runChain`'s Stage-3 default branch, `RunReactionBeat`'s combined ruling) — passing `nil`
- Test: `core/api/ledger_test.go`

**Interfaces:**
- Consumes: the confirmed signatures above.
- **No nested transactions.** `commitWorldPayload` is only ever reached from `fireDuePending` and
  `runWorldActor`, both called by `runWorldTurn` from `runChain` — none of which holds an open
  transaction (`adjudicate` opens and closes its own). The `o.DB.Begin` added below therefore takes a
  fresh pooled connection. If a future caller ever holds a tx, this needs a querier parameter instead —
  out of scope here, but do not "helpfully" add one.
- Produces:
  - `type postCommitFn func(ctx context.Context, tx pgx.Tx, eventIDs []string) error` (declared in `ledger.go`)
  - `func applyEventOnQuerier(ctx context.Context, q dbQuerier, worldID, actorID string, attemptJSON []byte, tick int64, seq int) (map[string]any, error)`
  - `commitWorldPayload(…, seq int, postCommit postCommitFn, outcome *BeatOutcome, trace *BeatTrace)` — hook inserted **before** `outcome`
  - `adjudicate(…, playerAnswer string, postCommit postCommitFn, trace *BeatTrace)` — hook inserted **before** `trace`
  - `commitRulingTx(…, tick int64, curSeq int, postCommit postCommitFn)`
  - `runWorldActor(…, seq int, postCommit postCommitFn, outcome *BeatOutcome, trace *BeatTrace)`

- [ ] **Step 1: Write the failing test**

Add to `core/api/ledger_test.go`:

```go
// Deferral A: a world-sourced commit and its bookkeeping row are ONE transaction. Proven from the
// bookkeeping side: when the post-commit hook fails, the canon commit must roll back with it — no
// half-pair can survive. Before the fix there was no hook at all and the two writes were separate
// statements, so nothing could roll the commit back.
func TestCommitWorldPayload_PostCommitFailureRollsBackTheCommit(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	orc := wtOrchestrator(pool)
	tick := wtBaseTick(t, ctx, pool)

	stated := fmt.Sprintf("post-commit rollback probe @%d", tick)
	attempt := Attempt{
		Type:       "Communicated",
		Stated:     stated,
		ListenerID: wtMaraID, // co-located with Kade in the tavern, so the structural floor passes
		Content:    "probe",
	}
	boom := func(ctx context.Context, tx pgx.Tx, eventIDs []string) error {
		if len(eventIDs) == 0 {
			return fmt.Errorf("hook received no event ids")
		}
		return fmt.Errorf("hook failed on purpose")
	}

	var out BeatOutcome
	ids, _, halt, err := orc.commitWorldPayload(ctx, dlWorldID, dlKadeID, attempt, tick, 0, boom, &out, nil)
	if err == nil {
		t.Fatalf("post-commit failure did not surface: ids=%v halt=%q", ids, halt)
	}

	var n int
	if qErr := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND summary=$2`,
		dlWorldID, stated).Scan(&n); qErr != nil {
		t.Fatalf("count canon_event: %v", qErr)
	}
	if n != 0 {
		t.Fatalf("canon_event rows for the probe = %d, want 0 — the commit did not roll back with the hook", n)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("outcome.Committed = %v, want empty — a rolled-back commit must never be reported", out.Committed)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd core/api && go test ./... -run TestCommitWorldPayload_PostCommitFailure -v`
Expected: FAIL — compile error, `too many arguments in call to orc.commitWorldPayload`. That is the correct first failure: the hook does not exist yet.

- [ ] **Step 3: Add the hook type and the querier-shared `applyEvent`**

In `core/api/ledger.go`, above `commitWorldPayload`:

```go
// postCommitFn is bookkeeping that MUST land in the same transaction as the canon commit it describes:
// the ledger's pending-row flip, the world's fire-log row. It runs inside the commit's tx with the
// committed event ids in hand; returning an error rolls the whole commit back. This is the same
// discipline commitRulingTx already applies to held_outcome resolution (resolveHeldIDs) — generalised so
// the two world-sourced callers share ONE mechanism instead of each bolting on a second statement.
type postCommitFn func(ctx context.Context, tx pgx.Tx, eventIDs []string) error
```

Add `"github.com/jackc/pgx/v5"` to `ledger.go`'s imports.

In `core/api/orchestrator.go`, replace `applyEvent` (`:1094`) with the querier-shared pair, mirroring `applyRuledEventOnQuerier`:

```go
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
```

- [ ] **Step 4: Thread the hook through `commitWorldPayload`**

Replace the body of `commitWorldPayload` (`ledger.go:45-92`) with:

```go
func (o *Orchestrator) commitWorldPayload(ctx context.Context, worldID, actorID string, attempt Attempt, tick int64, seq int, postCommit postCommitFn, outcome *BeatOutcome, trace *BeatTrace) (eventIDs []string, seqAdvance int, halt string, err error) {
	switch attempt.Type {
	case "ActorMoved", "Communicated", "ObjectRelocated":
		// Passthrough — the same routing runChain's Stage 3 uses for these three types, now inside a tx
		// so postCommit's bookkeeping row lands with the commit or not at all.
		attemptJSON, marshalErr := json.Marshal(attempt)
		if marshalErr != nil {
			return nil, 1, "", fmt.Errorf("commitWorldPayload: marshal attempt: %w", marshalErr)
		}
		tx, beginErr := o.DB.Begin(ctx)
		if beginErr != nil {
			return nil, 1, "", fmt.Errorf("commitWorldPayload: begin tx: %w", beginErr)
		}
		result, applyErr := applyEventOnQuerier(ctx, tx, worldID, actorID, attemptJSON, tick, seq)
		if applyErr != nil {
			_ = tx.Rollback(ctx)
			return nil, 1, "", fmt.Errorf("commitWorldPayload: apply_event: %w", applyErr)
		}
		// Mirror runChain's own passthrough check: halt_reason is the authoritative signal from
		// apply_event's structural floor, not just an empty event_id.
		if hr, _ := result["halt_reason"].(string); hr == "gate_reject" {
			_ = tx.Rollback(ctx)
			return nil, 1, "gate_reject", nil
		}
		evID, _ := result["event_id"].(string)
		if evID == "" {
			_ = tx.Rollback(ctx)
			return nil, 1, "no_event_id", nil
		}
		if postCommit != nil {
			if hookErr := postCommit(ctx, tx, []string{evID}); hookErr != nil {
				_ = tx.Rollback(ctx)
				return nil, 1, "", fmt.Errorf("commitWorldPayload: post-commit: %w", hookErr)
			}
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return nil, 1, "", fmt.Errorf("commitWorldPayload: commit tx: %w", commitErr)
		}
		// Reported only AFTER the tx committed — a rolled-back commit is not a commit.
		if outcome != nil {
			outcome.Committed = append(outcome.Committed, evID)
		}
		return []string{evID}, 1, "", nil

	default:
		// Adjudicated — a single-actor set, mirroring runChain's Stage 3 default branch. adjudicate owns
		// the ruling's tx, so postCommit rides into it (commitRulingTx runs it beside resolveHeldIDs).
		ar, adjErr := o.adjudicate(ctx, worldID, []ActorAttempt{{ActorID: actorID, Attempt: attempt}}, nil, tick, seq, "", postCommit, trace)
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
```

- [ ] **Step 5: Thread the hook through `adjudicate` and `commitRulingTx`**

In `orchestrator.go:1181`, add `postCommit postCommitFn` to `adjudicate`'s parameters immediately before `trace *BeatTrace`, and pass it on at `:1356`:

```go
	committed, seqAdvance, commitErr := o.commitRulingTx(ctx, tx, worldID, ruling, resolveHeldIDs, tick, curSeq, postCommit)
```

In `commitRulingTx` (`orchestrator.go:1380`), add `postCommit postCommitFn` as the final parameter, and run it as the last thing before returning success — after the events, the attribute writes, and the `resolveHeldIDs` update:

```go
	if postCommit != nil {
		if hookErr := postCommit(ctx, tx, committedIDs); hookErr != nil {
			return nil, 0, fmt.Errorf("post-commit hook: %w", hookErr)
		}
	}
```

Use whatever the local variable holding the committed ids is actually called at that point in the function; do not rename it.

Update the two remaining `adjudicate(` call sites in `orchestrator.go` (`runChain`'s Stage-3 default branch and `RunReactionBeat`'s combined ruling) to pass `nil` for `postCommit`. Find them with: `cd core/api && grep -n 'o.adjudicate(' *.go`

- [ ] **Step 6: Run the test and confirm it passes**

Run: `make reset && cd core/api && go test ./... -run TestCommitWorldPayload_PostCommitFailure -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add core/api/ledger.go core/api/orchestrator.go core/api/ledger_test.go
git commit -m "fix(livingworld): world-sourced commits carry their bookkeeping in-tx (deferral A, part 1)"
```

- [ ] **Step 8: Move the ledger flip onto the hook**

In `fireDuePending` (`ledger.go:162-200`), build the flip as the hook and delete the separate statement. Replace the `commitWorldPayload` call and the `TODO(Task 9)` block with:

```go
		// The pending row's status flip IS this commit's bookkeeping — it rides the same tx, so a row can
		// never be committed-but-still-pending (which would re-fire it on the next crossing).
		pendingID := d.id
		flipFired := func(ctx context.Context, tx pgx.Tx, _ []string) error {
			_, execErr := tx.Exec(ctx, `UPDATE pending_event SET status='fired' WHERE pending_id=$1`, pendingID)
			return execErr
		}
		committedIDs, seqAdvance, halt, commitErr := o.commitWorldPayload(ctx, worldID, pp.ActorID, pp.Attempt, tickAfter, curSeq, flipFired, outcome, trace)
```

Then delete lines `190-195` entirely (the `TODO(Task 9)` comment and the standalone `UPDATE … status='fired'` Exec). Leave the `cancelled` branch exactly as it is — nothing committed there, so there is no transaction to join.

- [ ] **Step 9: Move the fire-log insert onto the hook**

In `runWorldTurn` (`worldturn.go:128-139`), replace the `runWorldActor` call and the standalone insert with:

```go
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
```

Add `"github.com/jackc/pgx/v5"` to `worldturn.go`'s imports.

In `runWorldActor` (`worldactor.go:46`), add `postCommit postCommitFn` immediately before `outcome *BeatOutcome`, pass it straight through to `commitWorldPayload` at `:82`, and update the docstring line that currently reads *"Does NOT write world_eruption — Task 9's composer owns the fire-log row"* to say the composer still owns the row and now supplies it as a post-commit hook so it lands in the same transaction.

- [ ] **Step 10: Run the full suite**

Run: `make reset && cd core/api && go test ./... && go vet ./...`
Expected: PASS, with **no test edits**. The existing `TestRunWorldTurn_Standalone_CallableWithoutBeatLoop` already asserts a `world_eruption` row referencing the committed event — it now proves the atomic path.

- [ ] **Step 11: Commit**

```bash
git add core/api/ledger.go core/api/worldturn.go core/api/worldactor.go
git commit -m "fix(livingworld): ledger flip + fire-log write ride their commit's tx (deferral A)"
```


**Plan corrections found during execution (commits `660a81d`, `e1d0d17`) — recorded so the plan matches
what the code actually required:**

1. **Three `o.adjudicate(` call sites in `orchestrator.go`, not two.** Step 5 named `runChain` and
   `RunReactionBeat`; the third is `applyNPCDecisions` (an NPC's own adjudicated decision). All three
   pass `nil` — an NPC decision has no world bookkeeping row, and nothing may be wired into that hook.
2. **Pre-existing tests call `adjudicate` and `runWorldActor` directly**, so a Go arity change cannot
   coexist with a literal "zero edits to any pre-existing test". Ten call sites:
   `orchestrator_nary_test.go:97`; `orchestrator_ruled_test.go:142,234,299,377,462,530`;
   `resolve_factsheet_test.go:79`; `worldactor_test.go:69,115`. Each received exactly one literal `nil`.
   The rule this plan actually means: **no assertion, expectation, fixture, or test semantics may
   change.** A compiler-forced arity edit is not a test edit.
3. **Step 7's `git add` list produced a non-compiling tree** — `worldactor.go:82` still called the
   8-argument `commitWorldPayload`. Commit 1 must also include `core/api/worldactor.go` (passing `nil`,
   no signature change yet) and the four arity-only test files, so both commits build green.
4. **`runWorldTurn`'s docstring repeated the deleted TODO's claim** ("ATOMICITY IS DEFERRED … not yet in
   the same tx"). Step 9 only named the TODO at the insert site; deleting just that leaves a docstring
   that lies. Both that docstring and `worldactor.go`'s fire-log line were rewritten.

---

### Task 2: The World Actor acts where it says it acts (deferral B)

**The bug.** `runWorldActor`'s v1 scope is that the intrusion manifests perceivably **at `scene`** — the seat may attribute it to an entity already there, or bring a non-present NPC in via an `ActorMoved`. Today that is **prompt-only** (`prompts/world_actor.txt`); nothing checks it at runtime. A live model that authors an intrusion by someone standing somewhere else commits a real event the player is not positioned to perceive — and once the Journey has the world erupting at freshly-created roadside places, "acted in the wrong scene" stops being theoretical.

**Files:**
- Modify: `core/api/worldactor.go:46` (add the check after `validateAttemptFields`)
- Test: `core/api/worldactor_test.go`

**Interfaces:**
- Consumes: `actorLocation` and `fnTargetScene` (signatures above); `runWorldActor` as modified by Task 1.
- Produces: no signature change — the check is internal and fails loud (`err != nil`, nothing committed).

- [ ] **Step 1: Write the failing test**

Add to `core/api/worldactor_test.go`:

```go
// Deferral B: v1 scope is that the intrusion manifests AT the scene the composer passed. The fake
// authors an intrusion by a tavern resident; pointing the composer at Dock Street must be refused at
// runtime, not merely discouraged by the prompt.
func TestRunWorldActor_RefusesToActOutsideTheScene(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	const dockStreetID = "210c0000-0000-0000-0000-0000000000d2"

	orc := wtOrchestrator(pool)
	tick := wtBaseTick(t, ctx, pool)

	var out BeatOutcome
	eventID, seqUsed, err := orc.runWorldActor(ctx, dlWorldID, dockStreetID, "small", tick, 0, nil, &out, nil)
	if err == nil {
		t.Fatalf("authored intrusion from outside the scene was accepted: eventID=%q seqUsed=%d", eventID, seqUsed)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("outcome.Committed = %v, want empty — a refused intrusion must commit nothing", out.Committed)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `make reset && cd core/api && go test ./... -run TestRunWorldActor_RefusesToActOutsideTheScene -v`
Expected: FAIL — `err == nil`; the intrusion commits today because nothing checks the scene.

- [ ] **Step 3: Add the runtime check**

In `core/api/worldactor.go`, immediately after the `validateAttemptFields` block (`:78-80`) and before the `commitWorldPayload` call:

```go
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
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `make reset && cd core/api && go test ./... -run TestRunWorldActor -v`
Expected: PASS — the new test and every existing `runWorldActor` test, which all pass `wtTavernID` as the scene and therefore satisfy the check.

- [ ] **Step 5: Update the prompt rulebook to match**

In `core/api/prompts/world_actor.txt`, append this exact rule to the section that already describes the
scene constraint (do not restructure the file):

```text
WHERE IT HAPPENS. The entity you attribute the intrusion to must ALREADY be at the scene, or your
authored act must be an ActorMoved that brings it INTO the scene. An intrusion authored by an entity
standing anywhere else is rejected by the engine and never reaches the world.
```

- [ ] **Step 6: Run the full suite and commit**

```bash
make reset && cd core/api && go test ./... && go vet ./... && cd ../..
git add core/api/worldactor.go core/api/worldactor_test.go core/api/prompts/world_actor.txt
git commit -m "fix(livingworld): runtime scene check in runWorldActor (deferral B)"
```

---

### Task 3: The floor window gets its world's turn (deferral C)

**The bug.** A beat where nothing advanced the clock — an empty "I just watch", or a QUERY-only chain — still costs the instant floor at `runChain`'s tail (`orchestrator.go:381-394`), advancing world-time by ~2 seconds. But the world's turn only runs inside the loop, after a committed attempt. So the crossing `(startTick, startTick+floor]` is the one clock advance in the system with **no world's turn**: a pending event due in that window is silently skipped and never fires, and the tier rolls never happen. Narrow, but it is a lost fire, and the Journey makes quiet beats common.

**Files:**
- Modify: `core/api/orchestrator.go:381-394` (the floor block at `runChain`'s tail)
- Test: `core/api/orchestrator_worldtime_test.go`

**Interfaces:**
- Consumes: `ensureScene` (in scope, `orchestrator.go:129`), `advanceWorldTurn` (`:433`).
- Produces: no signature change.

- [ ] **Step 1: Write the failing test**

Add to `core/api/orchestrator_worldtime_test.go`:

```go
// Deferral C: the instant floor is a real clock crossing, so a pending event due inside it must fire.
// Before the fix, an empty beat advanced world-time by the floor with no world's turn at all, and the
// row sat pending forever.
func TestRunBeat_FloorWindowStillRunsTheWorldsTurn(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	// Pressure off: this test is about the LEDGER firing inside the floor window, not about rolls.
	wtDisableWorldActor(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	// A pending event due at baseTick+1 — inside the instant floor window (startTick, startTick+2].
	stated := fmt.Sprintf("floor-window ledger probe @%d", baseTick)
	payload := fmt.Sprintf(
		`{"actor_id":%q,"attempt":{"type":"Communicated","stated":%q,"listener_id":%q,"content":"probe"}}`,
		wtMaraID, stated, dlKadeID)
	var pendingID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO pending_event (world_id, fire_at_tick, magnitude, payload, status)
		 VALUES ($1, $2, 'small', $3::jsonb, 'pending') RETURNING pending_id`,
		dlWorldID, baseTick+1, payload).Scan(&pendingID); err != nil {
		t.Fatalf("insert pending_event: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM pending_event WHERE pending_id=$1`, pendingID); err != nil {
			t.Errorf("cleanup pending_event: %v", err)
		}
	})

	// An EMPTY chain: nothing advances the clock except the floor.
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, nil, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed", out.HaltReason)
	}

	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM pending_event WHERE pending_id=$1`, pendingID).Scan(&status); err != nil {
		t.Fatalf("read pending status: %v", err)
	}
	if status != "fired" {
		t.Fatalf("pending_event status = %q, want fired — the floor window skipped its world's turn", status)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `make reset && cd core/api && go test ./... -run TestRunBeat_FloorWindowStillRunsTheWorldsTurn -v`
Expected: FAIL with `pending_event status = "pending", want fired`.

- [ ] **Step 3: Run the world's turn for the floor crossing**

In `core/api/orchestrator.go`, inside the `if curTick == startTick {` block, after the floor has been added to `curTick` (i.e. after the overflow-guarded `curTick += floor`), append:

```go
		// Living World deferral C: the floor is a REAL clock crossing, so it gets its world's turn like
		// every other one — otherwise a pending event due inside (startTick, curTick] is silently
		// skipped and the tier rolls never happen. cutBeat is deliberately ignored: the chain is already
		// exhausted, so there is nothing left to discard, and the beat still completed. Any eruption
		// here commits and narrates normally.
		sceneID, sceneErr := ensureScene()
		if sceneErr != nil {
			return sceneErr
		}
		if _, _, wtErr := o.advanceWorldTurn(ctx, worldID, sceneID, startTick, curTick, curSeq, outcome, trace); wtErr != nil {
			return wtErr
		}
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `make reset && cd core/api && go test ./... -run TestRunBeat_FloorWindowStillRunsTheWorldsTurn -v`
Expected: PASS.

- [ ] **Step 5: Run the full suite**

Run: `make reset && cd core/api && go test ./... && go vet ./...`
Expected: PASS. Watch specifically for empty/QUERY-only beat tests that now see a world's turn where they previously saw none. If one fails **because a tier fired**, the correct fix is `wtDisableWorldActor` in that test (the established helper), not weakening the assertion. If one fails for any other reason, stop and report.

- [ ] **Step 6: Commit**

```bash
git add core/api/orchestrator.go core/api/orchestrator_worldtime_test.go
git commit -m "fix(livingworld): the instant-floor crossing runs its world's turn (deferral C)"
```

---

### Task 4: Rung gate

**Files:** none — verification only.

- [ ] **Step 1: Full battery from a clean database**

```bash
make reset && make test && cd core/api && go test ./... && go vet ./... && cd ../.. && make schema-check
```
Expected: 362 pgTAP assertions pass, the Go suite passes, no schema drift (this rung adds no migration, so `schema.sql` must be untouched).

- [ ] **Step 2: Re-run the Go suite without a reset to prove no cross-test pollution**

```bash
cd core/api && go test ./... && go test ./... && cd ../..
```
Expected: PASS both times. The new tests all register cleanups; a second run failing means one of them leaked state.

- [ ] **Step 3: Confirm both TODOs are gone**

```bash
grep -rn "TODO(Task 9)\|TODO(atomicity" core/api/ || echo "both atomicity TODOs cleared"
```
Expected: `both atomicity TODOs cleared`.

- [ ] **Step 4: Record the rung in the ledger**

Append a `# RUNG 0 COMPLETE` entry to `.git/sdd/progress.md` naming the three deferrals, the commits, and the gate results, in the style of the existing entries.

- [ ] **Step 5: Open the PR**

```bash
git push -u origin rung0/living-world-gates
```
PR body: the three deferrals, quoting the ledger's original gating condition ("gated BEFORE enabling the live/non-fake World Actor driver at play"), the commits that close each, and the gate output. Cite rule D-1 (nothing bypasses the gate) and the design doc §5.
