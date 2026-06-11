# Design — Chunk 1: Phase 0A Engine Contract (Deterministic Spine)

**Date:** 2026-06-10
**Chunk:** 1 of the build ladder (playbook §2) · 🪜 carries Validation-Ladder Q1
**Status:** Approved design (open edges resolved); ready for `writing-plans`.
**Governing contract (frozen, read-only input):**
- `docs/30_architecture/canon_engine/13_phase_0A_engine_contract.md` — the buildable artifact.
- `docs/30_architecture/canon_engine/03_world_state_technical_reference.md` §1 (Master DDL), §3 (projection rules), §5 (propagation — R1 only).
- `docs/30_architecture/canon_engine/07_test_and_invariant_spec.md` — I-1/I-2/I-7; Phase 0A gate (§6).
- `docs/30_architecture/canon_engine/02_world_state_adrs.md` — ADR-004, 006, 008, 026, 029, 030.
- `docs/00_strategy/06_rules_register.md` — the law (B-5, D-4, I-1/2/7).

> **Frozen-contract discipline.** The engine set is read-only input to this plan. If implementation
> reveals a genuine problem, the output is a *proposed new ADR* (superseding by number in doc 02; the
> number is assigned at proposal time), never a code workaround (CLAUDE.md; playbook §3; ADR-029). This
> design surfaces one such latent issue as an open-spec item (SPEC-002, §11) carrying a concrete proposal
> for the Chunk 1 retro, rather than silently absorbing it.

---

## 1. Goal & scope

**What 0A proves (doc 13 §1, verbatim intent):** manually inserted `accepted` events produce correct
materialized projections and correct perception records, and the projections can be dropped and rebuilt
from the event log to **domain-equivalent** state (ADR-026 — *not* byte-identity; volatile columns
excluded).

**Exit gate (doc 07 §6):** I-1 (replay invariance, domain-equivalent), I-2 (universal provenance),
I-7 (projections written only by the maintainer) green on the **Mara** deterministic spine. No bundles
in use, no Seren (ADR-029: "Phase 0 protected from cleverness").

**Validation-Ladder Q1 (playbook §0.5):** "Can world state replay deterministically?" The human exit
gate is: the operator runs the doc 13 §7 SQL by hand and I-1 is green. Divergence ⇒ **stop**, do not
proceed to Chunk 2.

**Hard out of scope:** any LLM call, the canonization pipeline, entity resolution, context assembly,
Seren/bundle *data*, any API endpoint, anything frontend. `causal_bundle*` and `provenance_edge` tables
are deployed but **never written** in 0A (ADR-008: schema-ready, used from Phase 4).

---

## 2. Open-edge decisions (the only things this brainstorm resolved)

The spec governs everything else. Four forks were decided:

| Edge | Decision | Rationale |
|---|---|---|
| **Harness / language posture** | **Pure SQL + pgTAP**, run by `pg_prove` entirely inside Postgres 16. No backend-language commitment. | 0A has no application runtime; the invariant suite *is* SQL (doc 07/13). Language is chosen later when Chunk 5 needs it. |
| **Schema scope** | doc 13 §2 in-0A tables **+** `causal_bundle`, `causal_bundle_input`, `provenance_edge` (empty, unused). `review_queue`/`extraction_log`→Chunk 7, `threshold_ledger`→Chunk 10, `world_snapshot`→when snapshotting lands. | doc 13 governs Chunk 1 over the looser brief. ADR-008/029 carve-out covers exactly bundles + provenance (frozen shape, can't be retrofitted). Other operational tables arrive with their semantics + invariants (ADR-029; D-9). |
| **`relationship_state` in seed** | **Empty in 0A.** Table deployed; projection trigger is a documented **no-op stub** (SPEC-001); pgTAP asserts zero rows. | doc 13 §5 ("rows exist only where interaction events created them") is satisfied vacuously. doc 03 does not specify how a single-`entity_id` mutation addresses a `(a_id,b_id)` row — that mechanism belongs to Phase-1 fan-out, not 0A. No invented mechanism (silent-workaround ban). |
| **Repo layout** | **Minimal SQL-first** under `core/db/` + `Makefile` + `docker-compose.yml` + `.github/workflows/`. No empty scaffolding. | Platform §5 layer tree is 🟡-directional (D-9); folders arrive with the chunk whose code needs them. The Master DDL is one frozen artifact with one ordered migration sequence (D-4) — never split per-domain. |

**Migration tool:** `dbmate` — single Go binary, plain-SQL up/down, tracks `schema_migrations`, zero
language runtime. Its generated `db/schema.sql` dump is wired as the **schema-vs-doc-03 audit artifact**
checked at every gate (§7, §8).

---

## 3. Components

1. **Migrations** (`core/db/migrations/`) — schema (§4 table set), append-only trigger, `apply_mutation()`
   function, projection triggers, role grants. Each file header cites `-- Source: canon_engine/03 §1 (frozen v4.1)`.
2. **Seed** (`core/db/seeds/seed_mara_0A.sql`) — deterministic, re-runnable into a clean DB (§5).
3. **Assertion suite** (`core/db/tests/`) — pgTAP specs wrapping doc 13 §7 boolean SQL + golden-projection
   diff + should-fail tests + guards (§6).
4. **Replay harness** — a plpgsql procedure implementing doc 13 §6, wrapped as a pgTAP boolean (I-1),
   calling the **same** `apply_mutation()` as the live trigger (§6, Rider A).
5. **Orchestration** — `Makefile`, `docker-compose.yml` (Postgres 16 + pgTAP), `.github/workflows/invariants.yml`.

---

## 4. Schema & triggers (revised per Rider A)

### 4.1 Tables (doc 03 §1, verbatim columns/indexes, logical time per ADR-030)

Deployed in the first migration:

- **Canon spine:** `canon_event`, `event_participant`
- **Deltas & lineage:** `state_mutation`, `provenance_edge` *(deployed, never written in 0A)*
- **Epistemic:** `perception_record`
- **Causal (schema-ready, unused — ADR-008):** `causal_bundle`, `causal_bundle_input`
- **Projections:** `actor_state`, `location_state`, `artifact_state`, `relationship_state`
- **Registry:** `entity_registry`

**Deferred (not in this migration):** `review_queue`, `extraction_log` (Chunk 7), `threshold_ledger`
(Chunk 10), `world_snapshot` (snapshotting).

### 4.2 Append-only enforcement (doc 03 §1.1; I-2 substrate)

`BEFORE UPDATE` trigger on `canon_event`: reject any update touching columns other than
`{status, accepted_at, superseded_by}`; reject illegal status transitions (legal set:
`proposed→accepted`, `proposed→rejected`, `accepted→retconned`, `accepted→superseded`). `DELETE` revoked
from all roles. *(In 0A every row is inserted already `status='accepted'`; the trigger exists to be
**tested** by the should-fail case, not exercised by the seed.)*

### 4.3 `apply_mutation()` — single shared write path (Rider A)

> **Rider A (correctness-critical):** there is exactly **one** function that writes a projection from a
> `state_mutation`. The live projection trigger AND the replay procedure both call it. I-1 must exercise
> the **production write path**, never a parallel reimplementation — otherwise replay tests a copy, not
> the system.

```
apply_mutation(m state_mutation) -- plpgsql, SECURITY DEFINER owned by maintainer role
  routes by m.entity_kind:
    'actor'    -> upsert actor_state    at m.attribute_path := m.new_value, set last_event_id
    'location' -> upsert location_state at m.attribute_path := m.new_value, set last_event_id
    'artifact' -> upsert artifact_state at m.attribute_path := m.new_value, set last_event_id
    'relationship' -> NO-OP STUB (SPEC-001); never reached in 0A (seed emits none)
  semantics: absolute SET of new_value at attribute_path (Rider B) => idempotent by construction.
```

- **Live trigger:** `AFTER INSERT ON state_mutation` (and on transition-to-accepted), gated on the parent
  `canon_event.status='accepted'` (doc 03 §3.1), calls `apply_mutation(NEW)`.
- **Replay procedure:** iterates accepted events' mutations in deterministic order (§6) and calls the
  **same** `apply_mutation()`.

### 4.4 `relationship_state` trigger = documented no-op stub (SPEC-001)

doc 03 does not specify how a single-`entity_id` `state_mutation` addresses a `(world_id, a_id, b_id)`
relationship row. The actor/location/artifact branches are fully writable from doc 03 (route by
`entity_kind` → single-PK table); the relationship branch is **not**. Per the silent-workaround ban it
ships as an explicit no-op with a comment referencing **SPEC-001** (§11). The table is deployed; the
branch is never reached in 0A (the seed emits zero relationship mutations).

### 4.5 Perceptions

`perception_record` rows are written **directly by the seed** (append-only; ADR-006: invalidation never
deletion). They are **not** rebuilt from mutations and are **excluded from replay** — checked for
provenance only (I-2), per doc 13 §6.

### 4.6 Roles & grants (I-7)

- `maintainer` role: sole writer of projection tables; owns `apply_mutation()` (SECURITY DEFINER).
- `app_reader` (or equivalent low-privilege role): used by the I-7 should-fail test to attempt an illegal
  projection write and be rejected.
- `DELETE` revoked on canon tables from all roles.

---

## 5. The Mara seed (doc 13 §4)

Cast pre-seeded in `entity_registry`: player `P`, NPCs `Mara (M)`, `Jonas (J)`; location `tavern`;
world `W`. Ticks are integers.

| seq | tick | event | participants | mutations | perceptions |
|---|---|---|---|---|---|
| E1 | 100 | `private_disclosure` (scope=private) | P:speaker, M:listener | — | M:`told`, P:`shared` |
| — | 101–200 | 100 noise events (moves + small disclosures among others; **not** the secret) | various | location/inventory **absolute state sets** | direct perceptions for participants only |
| E102 | 201 | `publicize` (scope=public) | P:instigator | — | public-knowledge record; M's E1 perception untouched; J eligible, not auto-acquired |

All events inserted `status='accepted'`, `origin='fast_path'`, with `in_world_tick`, `beat_seq`,
`in_world_label`, `accepted_at` set. No `proposed` lifecycle in 0A.

### 5.1 Rider B — absolute state sets, not deltas

> **Rider B:** all 0A seed mutations use **absolute `new_value` semantics**. A mutation's `new_value` is
> the *resulting* absolute value at `attribute_path` (e.g. `attrs.inventory.gold = 40`), never a `+N`
> delta. This is what makes `apply_mutation()` idempotent and replay order-stable.

doc 13 §5's loose word "deltas" is read as **absolute state sets** for 0A. Enforced by:
- An explicit comment block at the top of `seed_mara_0A.sql` stating the convention.
- A pgTAP **idempotency guard** (§6.4): applying the full mutation stream twice yields identical
  projection state — a property only absolute-set semantics satisfy.

### 5.2 Determinism preconditions the seed must guarantee (Rider C support)

- `(world_id, in_world_tick, beat_seq)` is **unique** across all accepted events (a per-world total order
  from domain data — this is exactly the SPEC-002 canonical-ordering proposal).
- `(valid_from_tick, valid_from_seq)` is **unique per `(entity_id, attribute_path)`** (deterministic
  last-writer per attribute).

Both are pgTAP-asserted (§6.4) so a future seed edit can never silently reintroduce non-determinism.

---

## 6. Assertions & replay (revised per Rider C)

All assertions are pgTAP, run by `pg_prove`. The doc 13 §7 boolean SQL is transcribed **verbatim** into
`checks_0A.sql` and wrapped as pgTAP assertions (parameterized via `\set` / psql variables for the entity
and event ids).

### 6.1 doc 13 §7 checks (verbatim)
- I-2 mutations: no orphan `state_mutation` (left-join to accepted `canon_event` is empty).
- I-2 perceptions: no orphan `perception_record` (left-join to accepted `canon_event` is empty).
- Knowledge boundary: Jonas has **zero** perceptions referencing E1 before E102 (the negative assertion).
- Mara knows: exactly **one** active `told` perception referencing E1 (`invalid_tick`/`expired_at` NULL).
- Mara survives publication: her E1 perception still present after E102 (count = 1).
- Public knowledge exists after E102: ≥1 `public` perception referencing E102.

### 6.2 Should-fail tests (pgTAP `throws_ok`)
- Illegal `UPDATE canon_event SET summary=…` raises (append-only trigger).
- Non-maintainer write to a projection table raises (I-7).

### 6.3 Golden-projection spot-check (doc 13 §5, §9)
`expected_projections_0A.csv` (hand-computed) is loaded into a temp table and diffed against
`actor_state` / `location_state` via pgTAP `set_eq` / `results_eq` (prints the diff on failure).

### 6.4 Guards (riders B/C, made explicit)
- **Idempotency guard (Rider B):** apply the full seed mutation stream a second time via
  `apply_mutation()`; assert projection **domain state** is unchanged (compared over the §6.5.1 domain
  columns, excluding volatile `updated_at`, which the re-apply legitimately re-stamps). Absolute-set ⇒ pass.
- **Event total-order guard (Rider C):** assert `(world_id, in_world_tick, beat_seq)` is UNIQUE across
  accepted events.
- **Mutation order guard (Rider C):** assert `(valid_from_tick, valid_from_seq)` is unique per
  `(entity_id, attribute_path)`.
- **Zero-relationship guard (SPEC-001):** assert `relationship_state` has **0 rows** after the seed —
  the vacuous condition made an intentional, queryable assertion.

### 6.5 Replay harness (I-1) — domain-only deterministic ordering

> **Rider C:** replay ordering uses **only deterministic domain data**. The wall-clock `recorded_at`
> (B-5 transaction-time axis; volatile per ADR-026) is **removed** from the ordering key.

doc 03 provides **no** canonical event-sequence column beyond `(in_world_tick, beat_seq)`: `event_id` is a
random UUID (non-deterministic across re-seeds) and `recorded_at` is the wall-clock column being removed.
Therefore the deterministic total order is `(world_id, in_world_tick, beat_seq)`, made total by the §5.2
seed uniqueness guarantee. Under that guarantee `recorded_at` was a **never-consulted** tiebreaker, so
removing it changes no result — a precondition hardening, not a deviation from doc 13 §6's outcome
(see SPEC-002).

```
replay_0A():  -- plpgsql, returns boolean; wrapped in pgTAP ok()
  1. Snapshot live projections (actor/location/artifact/relationship_state) into shadow tables.
  2. TRUNCATE the projection tables.
  3. Stream accepted events ORDER BY (world_id, in_world_tick, beat_seq);   -- domain-only (Rider C)
     for each event, its state_mutations ORDER BY (valid_from_tick, valid_from_seq),
     call apply_mutation(m)  -- SAME production path as the live trigger (Rider A).
  4. Diff rebuilt projections vs snapshot on the per-table comparison key below (§6.5.1).
  5. Return TRUE iff every per-table diff is empty over its domain column set.
```

Perceptions are append-only and not rebuilt (checked separately for provenance, I-2).

#### 6.5.1 Comparison key per projection table (ADR-026 domain equivalence)

The diff is defined positively — match rows on **natural domain identity**, compare the enumerated
**domain columns**, and ignore the enumerated **volatile/surrogate** columns. Nothing is left implicit.

| Projection table | Domain identity (join key) | Domain columns compared | Volatile (excluded) | Surrogate (excluded) |
|---|---|---|---|---|
| `actor_state` | `(world_id, entity_id)` | `attrs`, `last_event_id`, `dirty` | `updated_at` | — (`entity_id` PK *is* the domain identity) |
| `location_state` | `(world_id, entity_id)` | `attrs`, `last_event_id`, `dirty` | `updated_at` | — |
| `artifact_state` | `(world_id, entity_id)` | `attrs`, `last_event_id`, `dirty` | `updated_at` | — |
| `relationship_state` | `(world_id, a_id, b_id)` | `attrs`, `last_event_id`, `dirty` | — (no `updated_at` column in doc 03 §1.5) | — (composite key *is* the domain identity) |

Method per table: full-outer-join live↔rebuilt on the domain identity; assert no key present on only one
side, and for shared keys assert every domain column is equal. `relationship_state` has 0 rows in 0A, so
its diff is trivially empty (and the §6.4 zero-row guard makes that explicit). `attrs` (JSONB) is compared
by value (`jsonb` equality), independent of key order.

---

## 7. Repo layout & tooling

```
core/db/
  migrations/   NNNN_*.sql   -- schema + apply_mutation() + triggers + grants; header cites doc 03 §1
  seeds/        seed_mara_0A.sql
  tests/        checks_0A.sql, replay_0A.sql, *_test.sql (pgTAP), expected_projections_0A.csv
  schema.sql    -- dbmate-generated dump; the schema-vs-doc-03 audit artifact (committed)
docs/
  open-spec-items.md          -- SPEC-001, SPEC-002
  superpowers/specs/2026-06-10-phase-0A-engine-contract-design.md   -- this file
Makefile                      -- db-up / seed / test / replay / reset / schema-check
docker-compose.yml            -- Postgres 16 + pgTAP extension
.github/workflows/invariants.yml
```

No `ai-orchestration/`, `memory/`, `workers/`, `adapters/`, `modules/`, or per-domain `core/{world,…}/`
folders yet — each arrives with the chunk whose code needs it (D-9; platform §5 is directional). When
Chunk 5 introduces the application runtime, the layer-tree layout is an explicit item in that chunk's
brainstorm, informed by the platform doc's post-0A editorial, not preempting it.

**dbmate `schema.sql` as audit artifact:** after every migration, `dbmate` regenerates `core/db/schema.sql`
(pg_dump of the live schema + `schema_migrations`). This committed dump is the canonical object diffed
against doc 03 §1 at every gate. `make schema-check` (and a CI step) fail if `schema.sql` is dirty
(uncommitted drift), so "schema matches the frozen contract" is a read, not a hunt.

---

## 8. CI / invariant suite

`.github/workflows/invariants.yml`:
1. Spin up Postgres 16 with pgTAP.
2. `dbmate up` (apply migrations).
3. Load `seed_mara_0A.sql`.
4. `pg_prove core/db/tests/` — runs §6 (doc 13 §7 checks + should-fail + golden diff + guards + I-1 replay).
5. `make schema-check` — fail on `schema.sql` drift.

A red invariant blocks merge, always (CLAUDE.md; I-1…I-10 are the permanent regression suite). This suite
is the spine every later chunk runs against.

---

## 9. Definition of done (doc 13 §8) + exit gate

All five hold on a **clean deploy** of the seed:
1. Seed runs without error; all triggers fire.
2. Every doc 13 §7 SQL check returns its asserted boolean.
3. The illegal-`UPDATE` and non-maintainer-write tests both raise.
4. The replay procedure (§6.5) returns an empty diff (domain-equivalent, ADR-026).
5. Re-running the entire seed into a fresh DB produces identical results (determinism) — provable now that
   ordering is domain-only (Rider C) and mutations are absolute sets (Rider B).

Plus: `schema.sql` matches doc 03 §1 (audit artifact, §7).

**Human exit gate (operator-run):** the operator runs the doc 13 §7 SQL by hand and confirms **I-1 green**.
This answers Validation-Ladder Q1. If replay diverges, **stop and fix — do not proceed to Chunk 2.**

---

## 10. Deliverables (doc 13 §9, mapped to this layout)

- `core/db/migrations/NNNN_*.sql` — schema + `apply_mutation()` + triggers + grants *(= `schema_0A.sql`)*.
- `core/db/seeds/seed_mara_0A.sql` — the deterministic seed.
- `core/db/tests/expected_projections_0A.csv` — hand-computed expected projection rows.
- `core/db/tests/checks_0A.sql` + `*_test.sql` — the §7 assertions + guards as a runnable pgTAP suite.
- `core/db/tests/replay_0A.sql` — the §6.5 replay procedure returning pass/fail.
- `core/db/schema.sql` — dbmate dump (audit artifact).
- `Makefile`, `docker-compose.yml`, `.github/workflows/invariants.yml`, `docs/open-spec-items.md`.

---

## 11. Open spec items (`docs/open-spec-items.md`)

ADR numbers are **not** pre-assigned — they are allocated at proposal time in doc 02 (D-5).

- **SPEC-001 — mutation→`relationship_state` addressing.** doc 03 does not specify how a single-`entity_id`
  `state_mutation` addresses a `(world_id, a_id, b_id)` relationship row. 0A ships a no-op stub trigger and
  keeps `relationship_state` empty. *Owner:* Chunk 5 (Phase-1 fan-out) brainstorm. *Expected outcome:* a
  proposed new ADR in doc 02 (number assigned at proposal time), informed by Phase-1 evidence (D-9).
- **SPEC-002 — `recorded_at` in the canonical replay ordering key.** doc 13 §6 and doc 03 §3.4 both write
  the replay order as `(in_world_tick, beat_seq, recorded_at)`; `recorded_at` is a volatile wall-clock
  column (B-5 transaction-time; ADR-026 volatile) and is a latent determinism foot-gun as an ordering
  tiebreaker. 0A removes it and guarantees a domain-only total order via per-world
  `(in_world_tick, beat_seq)` uniqueness. *Owner:* Chunk 1 retro. *Expected outcome:* a proposed new ADR
  in doc 02 (number assigned at proposal time) — the only legal mechanism for a frozen-contract change
  (this design's preamble; D-5). *Proposal text:* "Canonical event ordering is
  `(in_world_tick, beat_seq)`, required UNIQUE per world; `recorded_at` is transaction-time (B-5) and
  excluded from domain ordering (ADR-026)." Non-blocking for 0A.

---

## 12. Non-goals (explicit, from PRD non-goals + register)

No LLM anywhere; no canonization pipeline, entity resolution, or context assembly; no bundle *writes*
(tables exist empty); no `proposed` lifecycle; no visibility fan-out beyond direct/private/public; no
World Clock service (ticks assigned by hand); no Redis; no pg_ivm (plain triggers only — ADR-004); no API;
no frontend. "Wouldn't it be nice to…" already has an answer in the register (playbook §3).
