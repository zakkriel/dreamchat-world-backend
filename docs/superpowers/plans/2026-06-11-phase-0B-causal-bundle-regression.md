# Phase 0B — Causal-Bundle Regression Implementation Plan

> **RENAMED 2026-08-27 — founder ruling.** This plan's riders were lettered `A`/`B`, colliding with
> the Phase 0A plan written **one day earlier**. Both were cited bare in shipped code. The letters are
> gone; the rules are unchanged.
>
> | Was | Now | The rule |
> |---|---|---|
> | 0B Rider A | **`IMMUTABLE-BUNDLE-TOPOLOGY`** | `effect_ref` is append-only; bundle topology never mutates |
> | 0B Rider B | **`ACYCLICITY-WALK`** | block a cycle that could resurrect on re-validation; depth-capped at 64 |
>
> Citation site updated: `20260611090001_causal_acyclicity.sql:13`.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce causal acyclicity (invariant I-4) on the bundle layer at insert time, freeze bundle topology after insert, and prove both with a hand-inserted Seren mini-scenario — with zero regression to the Phase 0A suite.

**Architecture:** One new dbmate migration adds (a) a `BEFORE INSERT` trigger on `causal_bundle_input` running a bounded ancestor walk that rejects any edge closing a cycle, and (b) append-only/no-delete triggers freezing bundle topology (ADR-006). A new pgTAP file proves valid conjunctive/disjunctive bundles are accepted and cyclic/topology-violating writes are rejected, entirely inside a `BEGIN…ROLLBACK` transaction (no standing data). Existing 0A assertions that read projections/global counts unscoped get a `world_id` predicate so the suite stops being one-world-fragile.

**Tech Stack:** PostgreSQL 16, plpgsql triggers, pgTAP (`pg_prove`), dbmate migrations, Docker Compose, Make.

**Spec:** `docs/superpowers/specs/2026-06-11-phase-0B-causal-bundle-regression-design.md` (branch `chunk-2-phase-0b`).

**Conventions (from the repo):**
- Migrations are hand-numbered timestamps `YYYYMMDDHHMMSS_name.sql` with `-- migrate:up` / `-- migrate:down` sections. The new one is `20260611090001_causal_acyclicity.sql`, kept **separate** from the verbatim doc-03 migrations 0002–0006 (precedent: `…0007`).
- Tests run via `make test` (globs `core/db/tests/*_test.sql`). By-hand scripts are **not** suffixed `_test.sql` so the glob skips them (precedent: `checks_0A.sql`, `replay_0A.sql`).
- Every test file is wrapped `BEGIN; … ROLLBACK;`.
- Custom `RAISE EXCEPTION` (no explicit ERRCODE) surfaces as SQLSTATE `P0001`; `throws_ok` matches on that.
- The Mara world id is `11111111-1111-1111-1111-111111111111`. The 0B Seren world id is `22222222-2222-2222-2222-222222222222`.

**Reset/run commands (used throughout):**
- `make reset` — clean DB from scratch (db-down, db-up, migrate, seed).
- `make test` — run the pgTAP suite.
- `make schema-check` — fail if `core/db/schema.sql` drifted from a clean migrated DB.
- `make fingerprint` — dump domain-only projection state (determinism diff).

---

## File Structure

| Action | Path | Responsibility |
|---|---|---|
| **Create** | `core/db/migrations/20260611090001_causal_acyclicity.sql` | I-4 insert-time walk + bundle topology immutability triggers. |
| **Create** | `core/db/tests/85_causal_acyclicity_test.sql` | Seren mini-scenario; asserts valid accept + cyclic/topology reject (ephemeral). |
| **Create** | `core/db/tests/demo_cycle_0B.sql` | Operator by-hand "insert a loop, watch it refuse" (excluded from CI glob). |
| **Modify** | `core/db/tests/80_golden_projection_test.sql` | World-scope the `actor_state` `set_eq`. |
| **Modify** | `core/db/tests/30_apply_mutation_test.sql` | World-scope the `relationship_state` count. |
| **Modify** | `core/db/tests/70_determinism_guards_test.sql` | World-scope the `relationship_state` count. |
| **Modify** | `core/db/tests/50_provenance_test.sql` | World-scope the noise-event tick-range count. |
| **Modify** | `docs/open-spec-items.md` | Append SPEC-005 (deferred nightly check). |
| **Regenerated** | `core/db/schema.sql` | Produced by `make migrate`; verified by `make schema-check`. |

`40_perception_test.sql` was audited and needs **no change** — its queries are already keyed by `world_id` (line 3), `holder_id`, or `source_event_id`, so they are world-robust.

---

## Task 1: Migration — I-4 acyclicity + bundle topology immutability (with failing test first)

**Files:**
- Test: `core/db/tests/85_causal_acyclicity_test.sql` (create)
- Create: `core/db/migrations/20260611090001_causal_acyclicity.sql`
- Regenerated: `core/db/schema.sql`

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/85_causal_acyclicity_test.sql` with exactly this content:

```sql
-- Phase 0B (chunk-2): I-4 causal acyclicity + bundle topology immutability.
-- Seren mini-scenario, TEST-TRANSACTION-ONLY (BEGIN/ROLLBACK) — no standing rows,
-- distinct world 2222… so 0A (world 1111…) is untouched. Events only; no state_mutation,
-- no perception_record => zero projection writes. Spec: docs/superpowers/specs/2026-06-11-*.
BEGIN;
SELECT plan(11);

-- Seren accepted events (durable refs for bundle inputs/effects; G3). Distinct ticks satisfy
-- uq_ce_accepted_order (unique world,tick,beat among accepted). No registry/mutations needed.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('20000000-0000-0000-0000-00000000000a','22222222-2222-2222-2222-222222222222','move','A',10,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000b','22222222-2222-2222-2222-222222222222','move','B',11,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000c','22222222-2222-2222-2222-222222222222','move','C',12,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000d','22222222-2222-2222-2222-222222222222','move','D',13,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000e','22222222-2222-2222-2222-222222222222','move','E',14,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000f','22222222-2222-2222-2222-222222222222','move','F',15,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-000000000020','22222222-2222-2222-2222-222222222222','move','G',20,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-000000000021','22222222-2222-2222-2222-222222222222','move','H',21,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-000000000022','22222222-2222-2222-2222-222222222222','move','I',22,0,'accepted',now(),'public','fast_path');

-- (1) Conjunctive: A AND B caused C — one bundle, two necessary inputs (ADR-007).
SELECT lives_ok($$
  DO $do$ BEGIN
    INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
    VALUES ('b0000000-0000-0000-0000-0000000000c0','22222222-2222-2222-2222-222222222222',
            '20000000-0000-0000-0000-00000000000c','event','conjunctive','MANUAL_0B','valid');
    INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role, necessity) VALUES
     ('b0000000-0000-0000-0000-0000000000c0','20000000-0000-0000-0000-00000000000a','event','enabler',true),
     ('b0000000-0000-0000-0000-0000000000c0','20000000-0000-0000-0000-00000000000b','event','trigger',true);
  END $do$;
$$, 'conjunctive A AND B -> C accepted (acyclic)');

-- (2) Disjunctive: D OR E caused F — two bundles, same effect, one input each (ADR-007).
SELECT lives_ok($$
  DO $do$ BEGIN
    INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status) VALUES
     ('b0000000-0000-0000-0000-0000000000d1','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-00000000000f','event','disjunctive_member','MANUAL_0B','valid'),
     ('b0000000-0000-0000-0000-0000000000d2','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-00000000000f','event','disjunctive_member','MANUAL_0B','valid');
    INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role) VALUES
     ('b0000000-0000-0000-0000-0000000000d1','20000000-0000-0000-0000-00000000000d','event','trigger'),
     ('b0000000-0000-0000-0000-0000000000d2','20000000-0000-0000-0000-00000000000e','event','trigger');
  END $do$;
$$, 'disjunctive D OR E -> F accepted (two bundles, one effect)');

-- (3) G3: every event-kind bundle input references a durable canon_event.
SELECT ok( NOT EXISTS (
    SELECT 1 FROM causal_bundle_input cbi
    WHERE cbi.input_kind='event'
      AND NOT EXISTS (SELECT 1 FROM canon_event ce WHERE ce.event_id=cbi.input_ref)
  ), 'G3: all event-kind bundle inputs reference durable canon_event rows');

-- (4) Self-loop: effect == input is rejected (I-4).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-00000000005e','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-00000000000c','event','conjunctive','MANUAL_0B','valid');
SELECT throws_ok($$
  INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
  VALUES ('b0000000-0000-0000-0000-00000000005e','20000000-0000-0000-0000-00000000000c','event','trigger')
$$, 'P0001', NULL, 'I-4: self-loop (input == effect) rejected');

-- (5) 2-cycle: C already causes-> nothing, but A AND B -> C exists, so C has ancestor A.
--     Adding a bundle "C caused A" closes A -> C -> A. Rejected (I-4). NEGATIVE CONTROL.
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-00000000002c','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-00000000000a','event','conjunctive','MANUAL_0B','valid');
SELECT throws_ok($$
  INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
  VALUES ('b0000000-0000-0000-0000-00000000002c','20000000-0000-0000-0000-00000000000c','event','trigger')
$$, 'P0001', NULL, 'I-4: edge C->A closing a 2-cycle (A->C exists) is rejected');

-- (6) Valid chain G -> H -> I accepted.
SELECT lives_ok($$
  DO $do$ BEGIN
    INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status) VALUES
     ('b0000000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-000000000021','event','conjunctive','MANUAL_0B','valid'),
     ('b0000000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-000000000022','event','conjunctive','MANUAL_0B','valid');
    INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role) VALUES
     ('b0000000-0000-0000-0000-0000000000a1','20000000-0000-0000-0000-000000000020','event','trigger'),
     ('b0000000-0000-0000-0000-0000000000a2','20000000-0000-0000-0000-000000000021','event','trigger');
  END $do$;
$$, 'chain G -> H -> I accepted (acyclic)');

-- (7) Closing the chain: "I caused G" (G ->* I exists) is rejected (I-4).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-0000000000a3','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-000000000020','event','conjunctive','MANUAL_0B','valid');
SELECT throws_ok($$
  INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
  VALUES ('b0000000-0000-0000-0000-0000000000a3','20000000-0000-0000-0000-000000000022','event','trigger')
$$, 'P0001', NULL, 'I-4: edge I->G closing the G->H->I chain is rejected');

-- (8) Rider A: effect_ref is immutable (append-only; only {status} may change).
SELECT throws_ok($$
  UPDATE causal_bundle SET effect_ref='20000000-0000-0000-0000-00000000000b'
  WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'P0001', NULL, 'Rider A: UPDATE of effect_ref on causal_bundle rejected (append-only)');

-- (9) Rider A: status alone may change.
SELECT lives_ok($$
  UPDATE causal_bundle SET status='invalidated'
  WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'Rider A: UPDATE of status alone is permitted');

-- (10) ADR-006: bundle inputs cannot be deleted.
SELECT throws_ok($$
  DELETE FROM causal_bundle_input WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'P0001', NULL, 'ADR-006: DELETE on causal_bundle_input rejected');

-- (11) ADR-006: bundles cannot be deleted.
SELECT throws_ok($$
  DELETE FROM causal_bundle WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'P0001', NULL, 'ADR-006: DELETE on causal_bundle rejected');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
make reset && make test
```
Expected: `85_causal_acyclicity_test.sql` runs but **fails** — assertions (4),(5),(7) (cyclic inserts), (8) (effect_ref UPDATE), (10),(11) (DELETEs) report `ok` was expected to `throw` but the statement **lived** (no triggers exist yet). Assertions (1),(2),(3),(6),(9) pass. The 0A files remain green.

- [ ] **Step 3: Write the migration**

Create `core/db/migrations/20260611090001_causal_acyclicity.sql` with exactly this content:

```sql
-- migrate:up
-- Phase 0B (chunk-2): implements the insert-time half of invariant I-4 (doc 07) per the
-- contract in doc 03 §1.4 ("Acyclicity is checked at bundle insert; bounded ancestor walk;
-- reject on cycle"), plus bundle topology immutability (ADR-006). The deferred nightly full
-- check is tracked as SPEC-005. NOT part of frozen doc 03 §1 migrations (kept out of 0002-0006),
-- same precedent as SPEC-002's …0007. No automated runtime path writes bundles before Phase 4
-- (ADR-008/029) — this only constrains the manual/Phase-4 write path.

-- (a) Insert-time acyclicity. A new input edge asserts input -> effect (input causes effect).
-- It closes a cycle iff the effect is already a causal ancestor of the input. Walk ancestors of
-- the new input; reject if the effect is reachable. Walk ALL edges regardless of status: bundle
-- status transitions are unspecified in the frozen contract, so an invalidated edge must still
-- block a cycle that could resurrect on re-validation (spec Rider B). Depth-capped (64) fail-safe.
CREATE FUNCTION causal_bundle_assert_acyclic() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  v_effect_ref  uuid;
  v_effect_kind text;
  v_cycle       boolean;
  v_maxdepth    int;
BEGIN
  SELECT effect_ref, effect_kind INTO v_effect_ref, v_effect_kind
  FROM causal_bundle WHERE bundle_id = NEW.bundle_id;

  WITH RECURSIVE anc(ref, kind, depth) AS (
    SELECT NEW.input_ref, NEW.input_kind, 0
    UNION ALL
    SELECT cbi.input_ref, cbi.input_kind, anc.depth + 1
    FROM anc
    JOIN causal_bundle cb
      ON cb.effect_ref = anc.ref AND cb.effect_kind = anc.kind
    JOIN causal_bundle_input cbi
      ON cbi.bundle_id = cb.bundle_id
    WHERE anc.depth < 64
  )
  SELECT bool_or(ref = v_effect_ref AND kind = v_effect_kind), max(depth)
  INTO v_cycle, v_maxdepth
  FROM anc;

  IF v_cycle THEN
    RAISE EXCEPTION
      'causal cycle rejected (I-4): effect %/% is already a causal ancestor of input %/% (bundle %)',
      v_effect_kind, v_effect_ref, NEW.input_kind, NEW.input_ref, NEW.bundle_id;
  END IF;

  IF v_maxdepth >= 64 THEN
    RAISE EXCEPTION
      'causal acyclicity depth cap (64) exceeded walking ancestors of input %/% — investigate (I-4)',
      NEW.input_kind, NEW.input_ref;
  END IF;

  RETURN NEW;
END $$;
CREATE TRIGGER trg_cbi_assert_acyclic
  BEFORE INSERT ON causal_bundle_input FOR EACH ROW
  EXECUTE FUNCTION causal_bundle_assert_acyclic();

-- (b) Topology immutability (ADR-006: invalidation never deletion; the insert-time walk is only
-- sound if edges cannot mutate after insert).
-- causal_bundle: append-only, only {status} may change (effect_ref/effect_kind/etc. frozen).
CREATE FUNCTION causal_bundle_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF ROW(NEW.bundle_id, NEW.world_id, NEW.effect_ref, NEW.effect_kind,
         NEW.semantics, NEW.template_id, NEW.created_at)
     IS DISTINCT FROM
     ROW(OLD.bundle_id, OLD.world_id, OLD.effect_ref, OLD.effect_kind,
         OLD.semantics, OLD.template_id, OLD.created_at)
  THEN
    RAISE EXCEPTION 'causal_bundle is append-only: only {status} may change (bundle %)', OLD.bundle_id;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_causal_bundle_append_only
  BEFORE UPDATE ON causal_bundle FOR EACH ROW EXECUTE FUNCTION causal_bundle_append_only();

-- causal_bundle_input: fully immutable (no UPDATE).
CREATE FUNCTION causal_bundle_input_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'causal_bundle_input is immutable: UPDATE forbidden (bundle %, input %)',
    OLD.bundle_id, OLD.input_ref;
END $$;
CREATE TRIGGER trg_causal_bundle_input_immutable
  BEFORE UPDATE ON causal_bundle_input FOR EACH ROW EXECUTE FUNCTION causal_bundle_input_immutable();

-- DELETE guards on both bundle tables (reuse forbid_delete() from migration 0002; ADR-006).
CREATE TRIGGER trg_causal_bundle_no_delete
  BEFORE DELETE ON causal_bundle FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_causal_bundle_input_no_delete
  BEFORE DELETE ON causal_bundle_input FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TRIGGER IF EXISTS trg_causal_bundle_input_no_delete ON causal_bundle_input;
DROP TRIGGER IF EXISTS trg_causal_bundle_no_delete ON causal_bundle;
DROP TRIGGER IF EXISTS trg_causal_bundle_input_immutable ON causal_bundle_input;
DROP TRIGGER IF EXISTS trg_causal_bundle_append_only ON causal_bundle;
DROP TRIGGER IF EXISTS trg_cbi_assert_acyclic ON causal_bundle_input;
DROP FUNCTION IF EXISTS causal_bundle_input_immutable();
DROP FUNCTION IF EXISTS causal_bundle_append_only();
DROP FUNCTION IF EXISTS causal_bundle_assert_acyclic();
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
make reset && make test
```
Expected: `85_causal_acyclicity_test.sql` — all **11** assertions pass. All 0A files still green.

- [ ] **Step 5: Verify schema captured + commit**

Run:
```bash
make schema-check
```
Expected: exit 0 (the migration's objects are in `core/db/schema.sql`; `make reset` already ran `migrate` which dumps it — if `schema-check` reports drift, run `make migrate` then re-check, then stage `core/db/schema.sql`).

Then commit:
```bash
git add core/db/migrations/20260611090001_causal_acyclicity.sql \
        core/db/tests/85_causal_acyclicity_test.sql \
        core/db/schema.sql
git commit -m "feat(0B): I-4 insert-time acyclicity + bundle topology immutability (ADR-006/007)"
```

---

## Task 2: Operator by-hand cycle demo

**Files:**
- Create: `core/db/tests/demo_cycle_0B.sql`

- [ ] **Step 1: Write the demo script**

Create `core/db/tests/demo_cycle_0B.sql` with exactly this content (NOT suffixed `_test.sql`, so `make test` skips it):

```sql
-- Phase 0B operator demo (by hand): insert a causality loop and watch the database refuse.
-- Run:  docker compose exec -T db psql -U postgres -d dreamchat -f /work/tests/demo_cycle_0B.sql
-- ON_ERROR_STOP is OFF so psql prints the rejection ERROR and proceeds to ROLLBACK.
-- Nothing persists. Excluded from the CI glob (filename is not *_test.sql).
\set ON_ERROR_STOP off
BEGIN;

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('20000000-0000-0000-0000-0000000000f1','22222222-2222-2222-2222-222222222222','move','demo A',1,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-0000000000f2','22222222-2222-2222-2222-222222222222','move','demo B',2,0,'accepted',now(),'public','fast_path');

-- Bundle 1: A causes B  (valid).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-0000000000f1','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-0000000000f2','event','conjunctive','MANUAL_0B','valid');
INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
VALUES ('b0000000-0000-0000-0000-0000000000f1','20000000-0000-0000-0000-0000000000f1','event','trigger');

-- Bundle 2: B causes A  (CLOSES THE LOOP — the next input insert must be REFUSED).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-0000000000f2','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-0000000000f1','event','conjunctive','MANUAL_0B','valid');
\echo '>>> Attempting to close the loop (B causes A). Expect: ERROR ... causal cycle rejected (I-4)'
INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
VALUES ('b0000000-0000-0000-0000-0000000000f2','20000000-0000-0000-0000-0000000000f2','event','trigger');

ROLLBACK;
```

- [ ] **Step 2: Run the demo and confirm refusal**

Run:
```bash
make reset
docker compose exec -T db psql -U postgres -d dreamchat -f /work/tests/demo_cycle_0B.sql
```
Expected: the first three inserts + bundle 2 succeed; the final input insert prints
`ERROR:  causal cycle rejected (I-4): effect event/20000000-…-f1 is already a causal ancestor of input event/20000000-…-f2 …`, then `ROLLBACK`. (This is the operator's personal-gate demonstration.)

- [ ] **Step 3: Commit**

```bash
git add core/db/tests/demo_cycle_0B.sql
git commit -m "test(0B): operator by-hand cycle demo (excluded from CI glob)"
```

---

## Task 3: Hygiene — world-scope the one-world-fragile 0A assertions

**Files:**
- Modify: `core/db/tests/80_golden_projection_test.sql`
- Modify: `core/db/tests/30_apply_mutation_test.sql`
- Modify: `core/db/tests/70_determinism_guards_test.sql`
- Modify: `core/db/tests/50_provenance_test.sql`

These edits keep every assertion's pass/fail identical for the Mara world; they only stop the queries being global. `40_perception_test.sql` is intentionally untouched (already scoped).

- [ ] **Step 1: Scope the golden projection set_eq (`80`)**

In `core/db/tests/80_golden_projection_test.sql`, change the left query of the `set_eq` from:

```sql
  'SELECT entity_id, attrs->>''location_id'' FROM actor_state',
```
to:
```sql
  'SELECT entity_id, attrs->>''location_id'' FROM actor_state WHERE world_id = ''11111111-1111-1111-1111-111111111111''',
```

- [ ] **Step 2: Scope the `relationship_state` count in `30`**

In `core/db/tests/30_apply_mutation_test.sql`, change:

```sql
SELECT is( (SELECT count(*) FROM relationship_state)::int, 0,
       'relationship mutation is a documented no-op (SPEC-001): zero rows');
```
to:
```sql
SELECT is( (SELECT count(*) FROM relationship_state
            WHERE world_id='11111111-1111-1111-1111-111111111111')::int, 0,
       'relationship mutation is a documented no-op (SPEC-001): zero rows');
```

- [ ] **Step 3: Scope the `relationship_state` count in `70`**

In `core/db/tests/70_determinism_guards_test.sql`, change:

```sql
SELECT is( (SELECT count(*) FROM relationship_state)::int, 0,
       'SPEC-001: relationship_state is empty in 0A (intentional vacuous satisfaction)');
```
to:
```sql
SELECT is( (SELECT count(*) FROM relationship_state
            WHERE world_id='11111111-1111-1111-1111-111111111111')::int, 0,
       'SPEC-001: relationship_state is empty in 0A (intentional vacuous satisfaction)');
```

- [ ] **Step 4: Scope the noise-event count in `50`**

In `core/db/tests/50_provenance_test.sql`, change:

```sql
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick BETWEEN 101 AND 200)::int,
       100, '100 noise events present');
```
to:
```sql
SELECT is( (SELECT count(*) FROM canon_event
            WHERE world_id='11111111-1111-1111-1111-111111111111'
              AND in_world_tick BETWEEN 101 AND 200)::int,
       100, '100 noise events present');
```

- [ ] **Step 5: Run the full suite to confirm zero regression**

Run:
```bash
make reset && make test
```
Expected: every file green, including the four edited ones (identical assertion outcomes) and `85`.

- [ ] **Step 6: Commit**

```bash
git add core/db/tests/80_golden_projection_test.sql \
        core/db/tests/30_apply_mutation_test.sql \
        core/db/tests/70_determinism_guards_test.sql \
        core/db/tests/50_provenance_test.sql
git commit -m "test(0B): world-scope 0A projection/global-count assertions (multi-world hygiene)"
```

---

## Task 4: Ledger — record SPEC-005

**Files:**
- Modify: `docs/open-spec-items.md`

- [ ] **Step 1: Append SPEC-005**

Add this section to the end of `docs/open-spec-items.md`:

```markdown

## SPEC-005 — nightly full acyclicity check (deferred half of I-4)
Phase 0B implements the **insert-time** half of I-4 (doc 03 §1.4: bounded ancestor walk on
`causal_bundle_input` insert; migration `20260611090001`). doc 07 I-4 also specifies a **nightly
full check** (recursive CTE with depth cap; cap hit = investigation). Not built in 0B — no
scheduled/operational jobs exist yet, and 0B's gate is satisfied by insert-time rejection.
- **Owner:** the first chunk that introduces scheduled/operational jobs (per-world nightly sweeps).
- **Expected outcome:** a nightly per-world recursive-CTE acyclicity sweep + alert on a positive
  hit or a depth-cap hit. **No ADR needed** — already specified in doc 07 I-4.
```

- [ ] **Step 2: Commit**

```bash
git add docs/open-spec-items.md
git commit -m "docs(0B): record SPEC-005 (deferred nightly acyclicity check, I-4 second half)"
```

---

## Task 5: Gate verification (operator-cut tag)

**Files:** none (verification + tag).

- [ ] **Step 1: Full clean gate run**

Run:
```bash
make reset && make test && make schema-check
```
Expected: all pgTAP green (0A + `85`); `schema-check` exit 0.

- [ ] **Step 2: Determinism double-deploy (DoD §8.5, mirrors CI)**

Run:
```bash
make fingerprint > /tmp/deploy1.txt
make reset
make fingerprint > /tmp/deploy2.txt
diff /tmp/deploy1.txt /tmp/deploy2.txt
```
Expected: empty diff (0B added no standing rows, so projections are byte-stable across deploys).

- [ ] **Step 3: Operator demonstrates the refusal by hand**

The operator runs Task 2 Step 2 (and/or a raw `psql` loop insert) and confirms the
`causal cycle rejected (I-4)` ERROR. This is the personal gate — not automatable away.

- [ ] **Step 4: Push branch and open the PR**

```bash
git push -u origin chunk-2-phase-0b
gh pr create --fill --title "chunk-2 (Phase 0B): causal-bundle regression — I-4 acyclicity"
```
Expected: CI (`invariants.yml`) green — invariant suite + determinism double-deploy + schema audit.

- [ ] **Step 5: Operator cuts the gate tag**

After CI is green and the by-hand refusal is confirmed, the **operator** (not the agent) cuts:
```bash
git tag chunk-2-0B-gate
git push origin chunk-2-0B-gate
```

---

## Self-Review (against the spec)

**Spec coverage:**
- §3 Component 1a (insert-time walk) → Task 1 (migration part a + assertions 4,5,7). ✔
- §3 Component 1b / Rider A (topology immutability) → Task 1 (migration part b + assertions 8,9,10,11). ✔
- §4 Rider B (walk all edges regardless of status) → Task 1 migration walk has no status filter; comment cites Rider B. ✔
- §3 Component 2 (Seren mini-scenario, conjunctive + disjunctive, ephemeral) → Task 1 Step 1 (events-only, BEGIN/ROLLBACK, world 2222…). ✔
- §3 Component 3 (operator demo, excluded from glob) → Task 2. ✔
- §3 Component 4 (SPEC-005) → Task 4. ✔
- §3 Component 5 (world-scope hygiene; 60 exempt; 40 already-robust noted) → Task 3. ✔
- §2 Edge 1 (insert-time half; nightly deferred) → Task 1 + SPEC-005. ✔
- §7 Gate / DoD (I-4 green; by-hand refusal; zero 0A regression; tag operator-cut) → Task 5. ✔
- Micro-decisions: test file `85` (Task 1), depth cap `64` (migration). ✔

**Placeholder scan:** migration filename resolved to `20260611090001_causal_acyclicity.sql`; no TBD/TODO; all code blocks complete. ✔

**Type/name consistency:** function names `causal_bundle_assert_acyclic` / `causal_bundle_append_only` / `causal_bundle_input_immutable` and trigger names are identical between the migration and its `migrate:down`; bundle/event UUIDs in `85` are internally consistent (effect/input refs match the inserted events; cycle tests reference bundles created in the same file). ✔
