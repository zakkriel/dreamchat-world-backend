# Open Spec Items

ADR numbers are assigned at proposal time in `canon_engine/02` (D-5). Do not pre-assign.

## SPEC-001 — mutation → relationship_state addressing
doc 03 does not specify how a single-`entity_id` `state_mutation` addresses a
`(world_id, a_id, b_id)` `relationship_state` row. Phase 0A ships a **no-op stub** branch in
`apply_mutation()` and keeps `relationship_state` empty (pgTAP-asserted zero rows).
- **Owner:** Chunk 5 (Phase-1 fan-out) brainstorm.
- **Expected outcome:** a proposed new ADR in `canon_engine/02`, informed by Phase-1 evidence (D-9).

## SPEC-002 — recorded_at in the canonical replay ordering key
doc 13 §6 and doc 03 §3.4 write the replay order as `(in_world_tick, beat_seq, recorded_at)`.
`recorded_at` is volatile wall-clock (B-5 transaction-time; ADR-026 volatile) and is a latent
determinism foot-gun as an ordering tiebreaker. Phase 0A removes it and guarantees a domain-only
total order via per-world `(in_world_tick, beat_seq)` uniqueness.
- **Owner:** Chunk 1 retro.
- **Expected outcome:** a proposed new ADR in `canon_engine/02` (number assigned at proposal time).
- **Proposal text:** "Canonical event ordering is `(in_world_tick, beat_seq)`, required UNIQUE per
  world; `recorded_at` is transaction-time (B-5) and excluded from domain ordering (ADR-026)."
- **Status:** non-blocking for 0A. The "required UNIQUE" half is now enforced by schema, not by
  seed-data shape: migration `20260610090007` adds partial unique index `uq_ce_accepted_order`
  on `(world_id, in_world_tick, beat_seq) WHERE status='accepted'` (kept out of the verbatim
  doc 03 migrations 0002–0006), with positive/negative pgTAP guards in
  `70_determinism_guards_test.sql`. The ADR proposal itself is still owed.

## SPEC-003 — projection on the proposed→accepted transition (doc 03 §3.1, second half)
doc 03 §3 rule 1: projection triggers fire "on insert with `status='accepted'` **or transition to
it**". Phase 0A implements only the insert-under-already-accepted half (`sm_project()` on
`state_mutation` insert); there is no trigger that projects an event's mutations when a
`canon_event` flips `proposed→accepted`. Correct and untestable in 0A (doc 13 §3: no proposed
lifecycle), but it is the half Phase 1's validation gate hits first.
`30_apply_mutation_test.sql` case (4) pins the current behaviour (non-accepted parent does not
project).
- **Owner:** the first Phase-1 chunk that introduces the proposed lifecycle (validation gate).
- **Expected outcome:** an `AFTER UPDATE` acceptance-transition trigger on `canon_event` calling
  the same `apply_mutation()`, with pgTAP coverage. No ADR needed — already specified in doc 03;
  this item only tracks the unimplemented half.

## SPEC-004 — idempotency mechanism: absolute-set semantics vs. the §3.2 mutation ledger
doc 03 §3 rule 2 prescribes idempotent projection writes via an applied-mutations ledger or
deterministic upserts keyed by `mutation_id`. Phase 0A instead relies on Rider B absolute-set
semantics (re-applying the same absolute value is a no-op), which is sound **only while deltas
are forbidden**. The moment any delta semantics appear (e.g. `attrs.inventory.gold` arithmetic),
absolute-set idempotency breaks and the §3.2 ledger becomes mandatory. Related:
`apply_mutation()` currently ignores `state_mutation.status`, so a future `reversed` row would
still apply — same revisit.
- **Owner:** the first Phase-1 chunk that introduces non-absolute mutations.
- **Expected outcome:** mutation-id-keyed idempotency per doc 03 §3.2 + a `status` guard in
  `apply_mutation()`. No ADR needed — already specified in doc 03.

## SPEC-005 — nightly full acyclicity check (deferred half of I-4)
Phase 0B implements the **insert-time** half of I-4 (doc 03 §1.4: bounded ancestor walk on
`causal_bundle_input` insert; migration `20260611090001`). doc 07 I-4 also specifies a **nightly
full check** (recursive CTE with depth cap; cap hit = investigation). Not built in 0B — no
scheduled/operational jobs exist yet, and 0B's gate is satisfied by insert-time rejection.
- **Owner:** the first chunk that introduces scheduled/operational jobs (per-world nightly sweeps).
- **Expected outcome:** a nightly per-world recursive-CTE acyclicity sweep + alert on a positive
  hit or a depth-cap hit. **No ADR needed** — already specified in doc 07 I-4.
- **Note (depth cap):** the insert-time walk's depth cap of 64 is a Phase-0B fail-safe ceiling
  (it raises a distinct "investigate" error on cap-hit), **not** a domain limit on causal-chain
  length. The future full-graph check must raise or remove this cap deliberately rather than
  inherit 64 as if it were a modeled bound.

## SPEC-007 — CI invariant workflow never executed (Actions $0 stop-budget)
**Status:** Resolved — root cause was a **$0 Actions stop-budget**, fixed from PR #4 onward
(GitHub Free, $0 owed, 168/2000 free minutes; a $0 spending limit with stop-usage-on, **not** a
real billing problem). Every `invariants.yml` run had 0s-failed at startup with **zero jobs** from
chunk-1 onward, so CI had never actually executed here. A payment method / non-zero Actions budget
now allows Actions to run.
- **Consequence for the gate map:** `chunk-1-0A-gate` was cut on **local evidence only** (it
  predated any CI execution). **chunk-2 gates on CI green + local** (PR #4 is the first real CI
  execution in this repo).
- **Owner:** chunk-2 (this chunk). No code change — operational/account fix.
