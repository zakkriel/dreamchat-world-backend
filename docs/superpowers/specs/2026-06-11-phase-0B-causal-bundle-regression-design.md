# Phase 0B — Manual Causal-Bundle Regression (Chunk 2) — Design

**Date:** 2026-06-11
**Chunk:** 2 (Phase 0B) · one worktree = one plan = one PR
**Status:** Approved design (brainstorm). Next step: writing-plans.
**Gate:** I-4 green; cyclic rejection demonstrated by hand; ALL 0A tests still green (zero
regression); tag `chunk-2-0B-gate` operator-cut.

---

## 1. Purpose & scope

Implement and prove the **causal acyclicity invariant (I-4)** on the bundle layer that has
existed empty since 0A (migration `…0004`, deployed per doc 03 §1.4, unused per ADR-008).
0B is the deliberate, contract-sanctioned moment to exercise bundles by hand — ADR-029:
*"bundle tables may exist in the Phase 0 schema and manual bundle test data is allowed; no
automated runtime path writes bundles before Phase 4."*

**In scope (exactly 0B):**
- Insert-time acyclicity enforcement for `causal_bundle` / `causal_bundle_input` (ADR-007, G3, I-4).
- Bundle topology immutability so the insert-time walk is sound (ADR-006).
- A hand-inserted **Seren mini-scenario**: a few accepted events + manual bundles expressing
  conjunctive ("A AND B caused C") and disjunctive ("D OR E caused F") shapes.
- pgTAP: valid bundles accepted; **cyclic bundle insert REJECTED** (the chunk's negative control).
- An operator by-hand demo: insert a causality loop, watch the database refuse.
- Hygiene: world-scope the latent one-world-fragile 0A assertions (0B is the first chunk where
  a second world's data can exist).

**Out of scope (reaffirmed — PRD non-goals + register ARE the out-of-scope list):**
templates (ADR-012), thresholds (ADR-015), automated bundle creation, dirty-ladder /
bounded invalidation (Phase 4), the **nightly full acyclicity check** (→ SPEC-005), any LLM,
any API.

**Rules in play:** ADR-006 (invalidation never deletion), ADR-007/008 (validated bundles;
provenance always; bundles selective, Phase 4+), ADR-029 (0A/0B split), G3 (inputs reference
durable records only), I-4 (acyclicity is CI law), B-5 (logical tick, no `TIMESTAMPTZ` domain
time), D-1 (no runtime path writes bundles here), TDD iron law (failing test first).

---

## 2. Open-edge resolutions

### Edge 1 — Where enforcement lives: the contract is **not** silent
- doc 03 §1.4 (G3): *"Acyclicity is checked at bundle insert (bounded ancestor walk; reject on cycle)."*
- doc 07 I-4 method: *"insert-time bounded ancestor walk (doc 03 §1.4) + nightly full check
  (recursive CTE with depth cap; cap hit = investigation)."*

→ **Insert-time, reject-on-cycle, bounded ancestor walk.** 0B implements the **insert-time half**
(the I-4 gate). The nightly full-check is the operational second half, tracked as **SPEC-005**
(deferred, not built here). No new SPEC is needed for the enforcement itself — already specified,
same posture as SPEC-003/004. (Note: the prompt's "SPEC-003" reference was stale — SPEC-003 is
already the acceptance-transition projection item.)

### Edge 2 & 3 — Where Seren data lives: **test-transaction-only fixtures**
The mechanism (trigger) is **permanent** (migration); the scenario data is **ephemeral**
(inside the 0B pgTAP `BEGIN…ROLLBACK`). Bundle rows are **not** in CI's standing seed
(Edge 3 = ephemeral). This respects the clean-DB guard untouched, matches every existing test
file's pattern, and is immune to the unscoped golden test. Decisive fact: `80_golden_projection_test`
reads `actor_state` **unscoped by world**, so any standing Seren rows touching projections would
break a green 0A test even under a different `world_id`. The Seren mini-scenario therefore uses
**accepted events only — no `state_mutation`, no perceptions** — so it writes zero projection rows.

---

## 3. Components

### Component 1 — New migration: acyclicity + topology immutability
`core/db/migrations/20260611NNNNNN_causal_acyclicity.sql` — kept **separate** from the verbatim
doc-03 migrations 0002–0006, following the precedent set by SPEC-002's `…0007`.

**(a) Insert-time acyclicity walk.** `BEFORE INSERT ON causal_bundle_input FOR EACH ROW`.
- Rationale for the table choice: inputs are inserted *after* their parent `causal_bundle` row
  (FK direction), so the bundle's effect already exists when each input edge lands; any cycle
  through a new bundle must cross at least one new input edge, so checking each input edge at
  insert is complete.
- Algorithm: look up `(effect_ref, effect_kind)` for `NEW.bundle_id`.
  - Base case — `NEW.input_ref = effect_ref AND NEW.input_kind = effect_kind` → self-loop, reject.
  - Recursive case — bounded ancestor walk from `(NEW.input_ref, NEW.input_kind)`:
    a node `(ref, kind)` has predecessors = inputs of all bundles whose
    `(effect_ref, effect_kind) = (ref, kind)`. **Walk ALL edges regardless of `status`**
    (see Edge B resolution §4). Depth-capped at **64**; cap-hit raises a distinct
    *"depth cap exceeded — investigate (I-4)"* fail-safe. If `(effect_ref, effect_kind)` appears
    in the ancestor set → reject.
  - `effect_kind ∈ {event, mutation}` only; perceptions are never effects, so perception inputs
    are natural leaves that bound the walk.
- Error: plain `RAISE EXCEPTION` (default SQLSTATE `P0001`), message citing I-4 — consistent with
  `forbid_delete()` / `canon_event_append_only()`. `throws_ok` matches on the message.

**(b) Topology immutability (Rider A — the walk is sound only if edges cannot mutate).**
- `causal_bundle_append_only()` — `BEFORE UPDATE ON causal_bundle FOR EACH ROW`, mirrors
  `canon_event_append_only()`: `ROW(all columns except status) IS DISTINCT FROM ROW(OLD …)` →
  reject. Only `{status}` may change; `effect_ref`/`effect_kind`/`world_id`/`semantics`/
  `template_id`/`bundle_id`/`created_at` are immutable. **No status-transition guard is added**
  (transitions are unspecified in the contract — §4 — and we invent no rule).
- `causal_bundle_input` is **fully immutable**: a `BEFORE UPDATE` trigger raises unconditionally
  (`causal_bundle_input_immutable()`), and a `BEFORE DELETE` uses the existing `forbid_delete()`.
- `forbid_delete()` `BEFORE DELETE` triggers on **both** `causal_bundle` and `causal_bundle_input`
  (ADR-006 — invalidation never deletion; closes the deployed gap where 0004 added delete guards
  to the lineage tables but not the bundle tables).
- `migrate:down` drops the four new triggers + the two new functions (DROP TRIGGER / DROP FUNCTION;
  the existing `forbid_delete()` is owned by migration 0002 and is **not** dropped here).

### Component 2 — New pgTAP: Seren mini-scenario + I-4 (ephemeral)
`core/db/tests/85_causal_acyclicity_test.sql` — `BEGIN … ROLLBACK`, picked up automatically by
`make test`'s `*_test.sql` glob (runs in CI, transaction-isolated). Seren world
`22222222-2222-2222-2222-222222222222`; `entity_registry` rows + a handful of **accepted
`canon_event` rows only** (no mutations, no perceptions). Plan ≈ 9:
- `lives_ok` — **conjunctive** "A AND B caused C": one `conjunctive` bundle, effect `EV_C`,
  inputs `{EV_A, EV_B}` (`necessity=true`).
- `lives_ok` — **disjunctive** "D OR E caused F": two `disjunctive_member` bundles, same effect
  `EV_F`, one input each (ADR-007: disjunction = multiple bundles for one effect).
- `ok` — all bundle inputs reference durable record ids (hard rule G3).
- `throws_ok` — **self-loop** (effect `X`, input `X`) → rejected (I-4).
- `throws_ok` — **2-cycle**: with `EV_A,EV_B→EV_C` present, insert bundle effect `EV_A` input
  `EV_C` → rejected. **The chunk's negative control and reason to exist.**
- `lives_ok` — a longer valid chain accepted; then `throws_ok` — its closing edge rejected.
- `throws_ok` — `UPDATE` of `effect_ref` on `causal_bundle` rejected (Rider A).
- `lives_ok` — `UPDATE` of `status` on `causal_bundle` accepted (proves `{status}` is the one
  permitted exception).
- `throws_ok` — `DELETE` on `causal_bundle` and on `causal_bundle_input` rejected (Rider A).

### Component 3 — Operator by-hand demo (excluded from CI glob)
`core/db/tests/demo_cycle_0B.sql` — `BEGIN … ROLLBACK`, excluded from the `*_test.sql` glob (same
convention as `checks_0A.sql` / `replay_0A.sql`; the `test` target runs only `*_test.sql`). Creates
two throwaway bundles closing a loop and surfaces the rejection error. Optional `helpers_0B.sql`
with Seren `\set` UUIDs (mirrors `helpers.sql`). Operator also runs the loop raw in `psql`.
Satisfies the personal gate: *"insert a paradox, watch it refuse."*

### Component 4 — Ledger: SPEC-005
Append to `docs/open-spec-items.md`:
> **SPEC-005 — nightly full acyclicity check (deferred half of I-4).** 0B implements the
> insert-time bounded ancestor walk (doc 03 §1.4). doc 07 I-4 also specifies a *nightly full
> check (recursive CTE with depth cap; cap hit = investigation)*. Not built in 0B (no scheduled
> jobs exist yet). **Owner:** the first chunk introducing scheduled/operational jobs.
> **Expected outcome:** a nightly per-world recursive-CTE sweep + alert. **No ADR** — already
> specified in doc 07.

### Component 5 — Hygiene: world-scope the 0A assertions
0B is the first moment a second world's data can exist, so the latent one-world-fragile queries
are fixed now — **identical pass/fail for world W**, just no longer global. Targets from the audit:
- `80_golden_projection_test.sql:14` — `set_eq` over `actor_state` (unscoped) → add
  `WHERE world_id = '11111111-…-1111'`.
- `30_apply_mutation_test.sql:32` and `70_determinism_guards_test.sql:11` — global
  `count(*) FROM relationship_state` → scope to the Mara world.
- `50_provenance_test.sql:9` — `count(*) FROM canon_event WHERE in_world_tick BETWEEN 101 AND 200`
  (unscoped by world) → add world predicate.
- `40_perception_test.sql` — sweep global counts over `perception_record` / `entity_registry`
  → scope to the Mara world.
- **Exempt:** `60_permissions_test.sql:12` asserts a *grant* (`lives_ok` SELECT), not a value —
  global is correct there.

---

## 4. Bundle status transitions (Rider B — transcribed from the contract)

doc 03 gives `causal_bundle.status` only a `CHECK` enum (`valid` / `invalidated` /
`pending_review`) — **no transition state machine** anywhere in the frozen set. The causal layer
is described as *"Append + invalidate … DAG only"* (doc 01 §layer table) and *"invalidation
operates on bundles: an effect is dirty when no surviving bundle suffices"* (ADR-007), but
`invalidated → valid` **re-validation is unspecified**.

Per the approved rule — *if re-validation is legal **or unspecified**, remove the filter and walk
all edges regardless of status; invent no transition rules* — the insert-time walk **walks all
edges regardless of `status`**. This closes the resurrection loophole (an invalidated cycle that
could become real on re-validation with no insert) without inventing a transition rule, and is
free in 0B (no invalidation path exists until Phase 4). Consistent with Rider A omitting a
status-transition guard on `causal_bundle_append_only()`.

---

## 5. Data flow / error handling

`causal_bundle` insert (effect) → per-input `causal_bundle_input` insert → `BEFORE INSERT` walk →
row lands, or `RAISE` aborts the transaction. **No cycle ever persists.** Topology is frozen after
insert: `causal_bundle` accepts only `{status}` updates; `causal_bundle_input` accepts none;
neither table accepts deletes. Every rejection raises `P0001` with an I-4 / ADR-006 message; the
depth-cap path raises a distinct fail-safe message.

---

## 6. File manifest

| Action | Path |
|---|---|
| **new** | `core/db/migrations/20260611NNNNNN_causal_acyclicity.sql` |
| **new** | `core/db/tests/85_causal_acyclicity_test.sql` |
| **new** | `core/db/tests/demo_cycle_0B.sql` (+ optional `helpers_0B.sql`) |
| **edit** | `docs/open-spec-items.md` (append SPEC-005) |
| **edit** | `core/db/tests/80_golden_projection_test.sql` (world-scope) |
| **edit** | `core/db/tests/30_apply_mutation_test.sql` (world-scope `relationship_state`) |
| **edit** | `core/db/tests/70_determinism_guards_test.sql` (world-scope `relationship_state`) |
| **edit** | `core/db/tests/50_provenance_test.sql` (world-scope tick-range count) |
| **edit** | `core/db/tests/40_perception_test.sql` (world-scope global counts) |
| **regenerated** | `core/db/schema.sql` (by `make migrate`; covered by `schema-check`) |

No CI workflow change: the new `*_test.sql` is auto-globbed; the new migration is auto-applied;
`schema-check` and the determinism double-deploy already run.

---

## 7. Gate / DoD

- **I-4 green** — `85`'s `lives_ok` (valid conjunctive + disjunctive) and `throws_ok` (self-loop,
  2-cycle, chain-closing edge) pass in CI.
- **Cyclic rejection by hand** — `demo_cycle_0B.sql` + raw `psql` show the refusal.
- **Zero 0A regression** — test-transaction isolation (no standing rows added) + identical-assertion
  hygiene fixes; CI's determinism double-deploy and `schema-check` cover the new migration.
- **Topology immutability** — `throws_ok` on `UPDATE effect_ref`, `DELETE` (both tables);
  `lives_ok` on `UPDATE status`.
- **Tag `chunk-2-0B-gate`** — operator-cut (not automated).

---

## 8. Build order (for writing-plans)

1. Migration: acyclicity walk + topology immutability (Components 1a, 1b) — TDD: write `85` first.
2. `85_causal_acyclicity_test.sql` (Component 2) — failing → green against the migration.
3. `demo_cycle_0B.sql` (+ `helpers_0B.sql`) (Component 3).
4. Hygiene world-scoping of 0A tests (Component 5) — assertions unchanged, still green.
5. SPEC-005 ledger note (Component 4).
6. `make reset && make test && make schema-check` + determinism double-deploy locally; operator
   by-hand demo; operator cuts `chunk-2-0B-gate`.
