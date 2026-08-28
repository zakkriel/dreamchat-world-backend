# time-and-clock · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-6 · Time and the world clock ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the write and read paths, validation, traps.
`time-and-clock.product.md` holds what it means; `time-and-clock.seams.md` holds what crosses its
boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate by
grep before relying on one.

---

## Storage

- **`canon_event`** carries the clock's columns: `in_world_tick bigint NOT NULL`,
  `in_world_label text`, `beat_seq integer DEFAULT 0 NOT NULL`, `temporal_uncertainty boolean`
  (`schema.sql:3797-3800`; grep `CREATE TABLE public.canon_event`). The table itself is the canon
  spine's; the columns' semantics are this domain's.
- **`uq_ce_accepted_order`** — partial unique index on `(world_id, in_world_tick, beat_seq)
  WHERE status='accepted'` (`schema.sql:4768`; migration `20260610090007`). This is what makes
  `(tick, beat_seq)` a total order without a wall-clock tiebreaker (`SPEC-002`, resolved → `ADR-034`).
- **`trg_canon_event_carry_in_world_label`** — BEFORE INSERT (`schema.sql:4796`, function `:690`;
  migration `20260808100002`): a NULL label inherits the last authored label at or before
  `(tick, beat_seq)` in the same world. Continuity, never invention; an explicit label always wins.
- **`fn_world_now(world)`** (`schema.sql:3212`) — `GREATEST(max(canon_event.in_world_tick),
  max(journey.current_tick))`. Journeys are **not** filtered by status: an ended journey still holds
  its tick, *"because time must never rewind when one stops (B-5)"* (migration `20260807100003`).
- **`duration_class_seconds`** + `fn_duration_class_seconds(world, class)` — per-world retunable
  mapping of the closed class set `instant|short|medium|long|extremely_long` to seconds, with a
  built-in fallback (migration `20260805100001`; built under Living World Task 2).
- **Tension** lives on `location_state.attrs->>'tension'`, enum-validated by `trg_validate_tension`
  (`schema.sql:4782`; migration `20260723100002`). The mapping to a budget is Go:
  `core/api/tension.go` (frantic 5 · tense 30 · normal 60 · calm 600 · none ∞ via `math.MaxInt64`;
  unknown and missing both default to `none`).

## The write path — how the clock advances

Two doors advance the clock, and they compute durations differently (see Traps):

- **`apply_beat`** (migration `20260618090001`, since rewritten): per committed step,
  `IF dur > 0 THEN cur_tick := cur_tick + dur; cur_seq := 0; ELSE cur_seq := cur_seq + 1;`
  (grep that line in `schema.sql`) — a zero-duration event shares the tick and increments `beat_seq`,
  which is exactly what `uq_ce_accepted_order` needs. Duration comes from `fn_move_duration(world,
  from, to)` — the location-pair form (`schema.sql:2521`), walk speed hard-coded at 1.4 m/s.
- **The Go orchestrator** (`core/api/orchestrator.go`, `runChain`): reads the beat budget once at
  beat start from the scene's *current* tension (`tension.go` `beatBudgetSeconds`); consumes it
  **cumulatively** across the chain; a move's duration is `fn_move_duration_actor` (status-aware,
  speed 0 → `MaxInt64` sentinel, `schema.sql:2529`), a non-move's is its decomposer-tagged
  `duration_class`. Over budget halts the chain before committing — the prefix stands; an over-budget
  *move* becomes a Journey (WE-5's noun), never a reject.
- **The stillness floor** (`orchestrator.go:484-517`): a completed beat where nothing advanced
  world-time still costs the `instant` floor — *"stillness ticks too"* — and the floor is a real
  clock crossing, so it gets its own world's turn, *"otherwise a pending event due inside
  (startTick, curTick] is silently skipped."* The floor never applies to a halted beat.

## The read path

Consumers of `fn_world_now`: `core/api/scenehandler.go:198` (scene "now"), `core/api/journey.go`,
`core/api/beatsstream.go`, and compendium staleness (`schema.sql:1336` — `stale` is
`fn_world_now - valid_tick > decay horizon`). The label a player sees rides their own newest
*perceived* line (`scenehandler.go:209` joins `canon_event.in_world_label` through
`perception_record`) — even the clock's display is perception-bound.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `SPEC-002` | **Resolved → `ADR-034`.** Replay order is `(in_world_tick, beat_seq)`, domain-only, unique per world by constraint. `recorded_at` is excluded — it is volatile telemetry. | A wall-clock tiebreaker silently breaks I-1 replay invariance on rebuild/restore. |
| `ADR-036` | Accepted, with shipped evidence: pgTAP 95/96/97 and `apply_beat`. Advancement = committed durations, engine-assigned, LLM never invents one. | An LLM-supplied duration is a number the engine must swallow — the Q1 guardrail dies. |
| `SPEC-030` | **Landed.** Movement is nameable; and its receipt for this domain: an unstamped room reads tension `none` ⇒ an infinite budget, *"the exact condition that made the Journey unreachable"* — so genesis stamps every location, parents included (`core/api/worldgenesiscommit.go:629-631`). | An unstamped location can never go over budget, so no journey ever starts from it. |
| `B-5` / `ADR-030` | Tick + authored label; the flat `beatTickCap=1000` backstop is retired for the tension budget (`orchestrator.go:20-24`). | Re-adding a flat cap makes the tension tiers a lie. |

### What you may not decide alone

1. **Adding a tension tier or a duration class.** Both are closed sets (CHECK on
   `duration_class_seconds`; `trg_validate_tension`). Retune the *values* per world; the shape is law.
2. **Changing the ordering key.** `(tick, beat_seq)` uniqueness is what replay stands on.
3. **Any new author of `in_world_label`.** Continuity-vs-invention was ruled once (migration
   `20260808100002`); a new authoring path re-decides it.
4. **Deriving a label from the tick.** That exact fix shipped once and was reverted as a `B-5`
   violation (Traps, row 1).

## Validation for this domain

pgTAP in `core/db/tests/`: `26_in_world_label_carry*` (the carry trigger, 4 assertions on a private
world) · `70_determinism_guards*` (ordering-key uniqueness by constraint, positive and negative) ·
`95_apply_beat_happy*` / `96_apply_beat_partial_beat*` (clock placement; interrupted chain advances
by the committed prefix only) · `114_fn_world_now*`. Go: `core/api/tension_test.go`,
`core/api/orchestrator_worldtime_test.go` — the Go suite is seed-dependent, so `-count=1` always,
and **never `make reset`** (it destroys the dev volume; the exemplar's warning applies verbatim).

**What counts as evidence here:** this domain fails silently — the label going NULL errored nowhere;
three surfaces just went blank (migration `20260808100002` header). Reproduce-first.

**What counts as ceremony here:** asserting a label on *seeded* rows. The seed authors labels
directly, so a seeded assertion passes with the carry trigger deleted. Test 26 exists because it
builds a private world and inserts unlabelled events itself.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **A label is authored, never derived.** Synthesis once rendered "[Tick 51]" — manufacturing a display label from the logical tick. | Migration `20260808100002` header, which names it *"exactly what B-5 forbids."* |
| **The two doors time the same move differently.** `apply_beat` uses location-pair `fn_move_duration` (hard-coded 1.4 m/s, no status modifiers, *"legacy apply_beat compat, decision 6"*); the orchestrator uses `fn_move_duration_actor`. An encumbered actor is full-speed through one door and arithmetic-blocked through the other. | `schema.sql:97` vs `orchestrator.go` Stage 4; migration `20260729100002` comments. Recorded, not resolved — see `digest/S13b_code_doctrine_migrations.md` §Topic 23. |
| **A location with no tension is an infinite budget.** | `core/api/worldgenesis_test.go:88-89`: *"the exact SPEC-030 blocker."* |
| **Time never rewinds when a journey ends.** `fn_world_now` deliberately reads ended journeys. | Migration `20260807100003`; `core/db/tests/114_fn_world_now_test.sql`. |
| **Do not patch commit functions; there are ~18 `INSERT INTO canon_event` sites re-copied verbatim across migrations.** Copying a stale body silently reverted three later extensions once. | Migration `20260808100002` header (the SPEC-031 tuning cost). The carry trigger exists *because* of this. |
| **A clock crossing without a world's turn silently skips due events.** | `orchestrator.go:505-509`. |

## Open questions

1. **`ADR-036`'s rationale vs the stillness floor.** The ADR: *"pure Continue commits no events → no
   time passes (C-6)."* Living World Task 3: *"every beat costs world-time"* — an empty completed
   beat costs the instant floor (`orchestrator_worldtime_test.go:11`, `orchestrator.go:484`). Both
   are in force; which sentence yields is a ruling.
2. **`calendar_system` is in `ADR-030`'s decision text and nowhere in the schema** (grep empty,
   2026-08-27). Amend the ADR, or build the column?
3. **`ADR-034` is still `Status: Proposed`** while its enforcement (index + test 70) shipped and
   `SPEC-002` says Resolved. Who accepts it?
4. **Who ever authors "Day 2"?** No in-game path writes a label (`product.md` §not built). Is the
   fiction's calendar meant to advance, and through which seat?
