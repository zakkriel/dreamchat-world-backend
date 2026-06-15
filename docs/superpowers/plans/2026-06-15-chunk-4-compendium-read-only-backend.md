# Chunk 4 — Read-only Compendium (Backend Leg) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the read-only, perception-bound Location / Artifact / Timeline pages and the per-type Compendium index to the DreamChat backend, all over the proven Chunk-3 two-filter spine, with an existence wall that hides unperceived entities on both the browse path (index) and the direct-id path (404).

**Architecture:** SQL is the single source of the perception/existence filter (ADR-P017): FILTER 1 (`fn_visible_perceptions`) and the name gate (`fn_perceived_name`) are reused **untouched**; one new `fn_collected_knowledge` core carries the shared about-ness lens; new `fn_entity_visible` is the existence predicate used by both the per-id pages (NULL → 404) and `fn_compendium_index`. Go stays a thin reader that calls a function, stamps `schema_version`, and maps a NULL result to 404. Four versioned JSON schemas are published for the frontend repo to codegen from.

**Tech Stack:** PostgreSQL + pgTAP (tests via `pg_prove`), dbmate migrations, Go 1.26 (`pgx/v5`, stdlib `net/http`/`httptest`), Docker Compose.

**Spec:** `docs/superpowers/specs/2026-06-14-chunk-4-compendium-read-only-design.md`. Cited rule IDs: B-1, B-2, B-5, I-2, I-3, I-9, ADR-005, ADR-034, ADR-035, ADR-P017.

**Execution context:** Run in the Chunk-4 backend worktree (create via `superpowers:using-git-worktrees` at execution start). One worktree / one plan / one PR to backend `main`. The frontend leg (`dreamchat-frontend`) is a **separate** plan and PR — nothing here touches a `frontend/` directory (D-7/D-10).

**Conventions used throughout:**
- Fixed UUIDs (match `core/db/tests/helpers.sql`): world `11111111-1111-1111-1111-111111111111`, Player `aaaaaaaa-…`, Mara `bbbbbbbb-…`, Jonas `cccccccc-…`, Tavern `dddddddd-…`, Common Knowledge `eeeeeeee-…`, Square `000000a0-0000-0000-0000-0000000000a1`, O1 `00000000-0000-0000-0000-000000000001`.
- **NEW Chunk-4 ids:** Sealed Note artifact `a4000000-0000-0000-0000-0000000000a1`; discovery event `e0000000-0000-0000-0000-0000000000d1`; note perception `dca70000-0000-0000-0000-000000000b01`; tavern perception `dca70000-0000-0000-0000-000000000c01`.
- **Run a single pgTAP file:** `docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/<FILE>'` (requires `make reset` first so the migrated+seeded DB is current).
- **Run the whole suite:** `make reset && make test`.
- **Fail-first demonstration:** write the test, run it against the current DB *before* adding the migration/seed change — it errors (function/row missing) = RED. Then implement and re-run = GREEN.

---

## Task 1: Seed the Sealed Note fixture + bump the cast manifest

Adds the non-CK artifact, its discovery `observation` event (no state_mutation), and the two Player-private perceptions (subjects = Note alone, Tavern alone). One existing assertion (`40`'s cast count) legitimately moves 11→12.

**Files:**
- Test: `core/db/tests/16_chunk4_seed_test.sql` (create)
- Modify: `core/db/seeds/seed_mara_0A.sql` (insert Chunk-4 block before the `(B)` backfill)
- Modify: `core/db/tests/40_perception_test.sql:4` (11 → 12)

- [ ] **Step 1: Write the failing seed-data test**

Create `core/db/tests/16_chunk4_seed_test.sql`:

```sql
BEGIN;
SELECT plan(6);
-- the Sealed Note artifact exists, is an artifact, created mid-timeline by the discovery event
SELECT is( (SELECT entity_kind FROM entity_registry
            WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'artifact', 'Sealed Note is an artifact');
SELECT is( (SELECT created_by_event FROM entity_registry
            WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'e0000000-0000-0000-0000-0000000000d1'::uuid,
           'note created_by_event = discovery event (introduced mid-timeline)');
-- discovery event accepted at (100,1): unique vs E1 (100,0) under uq_ce_accepted_order
SELECT is( (SELECT in_world_tick::text||','||beat_seq::text FROM canon_event
            WHERE event_id='e0000000-0000-0000-0000-0000000000d1' AND status='accepted'),
           '100,1', 'discovery event accepted at (100,1)');
-- two Player 'direct' perceptions sourced to the discovery event
SELECT is( (SELECT count(*) FROM perception_record
            WHERE source_event_id='e0000000-0000-0000-0000-0000000000d1'
              AND holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
              AND epistemic_type='direct')::int,
           2, 'two Player direct perceptions from the discovery event');
-- note perception subject = the Note ALONE (subject ≠ participants {Player,Note,Tavern}; ADR-035)
SELECT is( (SELECT count(*) FROM perception_subject
            WHERE perception_id='dca70000-0000-0000-0000-000000000b01')::int,
           1, 'note perception has exactly one subject');
SELECT is( (SELECT entity_id FROM perception_subject
            WHERE perception_id='dca70000-0000-0000-0000-000000000b01'),
           'a4000000-0000-0000-0000-0000000000a1'::uuid,
           'note perception subject is the Note alone');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/16_chunk4_seed_test.sql'`
Expected: FAIL (the note/event/perceptions are not seeded yet).

- [ ] **Step 3: Add the Chunk-4 seed block**

In `core/db/seeds/seed_mara_0A.sql`, find the comment line `-- (B) Generic about-ness backfill for the ORIGINAL event-derived perceptions only` and insert this block **immediately before it** (after the `(A)` about-Mara block):

```sql
-- =====================================================================================
-- (C4) Chunk-4 additions (design 2026-06-14): the Sealed Note artifact fixture. It powers the
-- Artifact page AND the index existence-leak asymmetry (a non-CK entity perceived ONLY by Player,
-- ABSENT for Jonas). The Tavern observation gives the Location page real about-ness. An observation
-- changes PERCEPTION, not canon (ADR-005), so there is NO state_mutation and NO artifact_state row
-- (golden/replay untouched). Deterministic; chosen to miss every existing scoped 0A assertion.
-- =====================================================================================

-- discovery event @ tick 100, beat_seq 1 — distinct slot from E1 (100,0) under uq_ce_accepted_order.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111',
        'observation','Player, alone in the tavern, finds a sealed note and notes the room''s tension',
        100,1,'Day 1','accepted', now(), 'private','fast_path');

-- the Sealed Note artifact — NON-CK (no genesis name perception), created by the discovery event.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name, created_by_event) VALUES
 ('a4000000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111','artifact','Sealed Note',
  'e0000000-0000-0000-0000-0000000000d1');

-- participants: observer + the two subjects. subject ≠ participants — the explicit perception_subject
-- rows below carry the PRECISE about-ness (ADR-035), not the participant set.
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-0000000000d1','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','observer'),
 ('e0000000-0000-0000-0000-0000000000d1','a4000000-0000-0000-0000-0000000000a1','artifact','discovered'),
 ('e0000000-0000-0000-0000-0000000000d1','dddddddd-dddd-dddd-dddd-dddddddddddd','location','setting');

-- two Player-private 'direct' perceptions, fixed ids, each with an EXPLICIT single subject.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 ('dca70000-0000-0000-0000-000000000b01','11111111-1111-1111-1111-111111111111',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-0000000000d1',
  'A small folded note, sealed with dark wax. No markings, no sender.','direct',100,100),
 ('dca70000-0000-0000-0000-000000000c01','11111111-1111-1111-1111-111111111111',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-0000000000d1',
  'The tavern was tense and quieter than usual.','direct',100,100);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('dca70000-0000-0000-0000-000000000b01','a4000000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111'),
 ('dca70000-0000-0000-0000-000000000c01','dddddddd-dddd-dddd-dddd-dddddddddddd','11111111-1111-1111-1111-111111111111');
```

- [ ] **Step 4: Bump the cast manifest assertion**

In `core/db/tests/40_perception_test.sql`, change the registry-count assertion (lines 3-5):

```sql
SELECT is( (SELECT count(*) FROM entity_registry
            WHERE world_id='11111111-1111-1111-1111-111111111111')::int, 12,
       'registry seeded with cast: P,M,J,Tavern,PUB,O1..O5,Square,SealedNote (chunk-4 +1)');
```

- [ ] **Step 5: Run the suite to verify green**

Run: `make reset && make test`
Expected: `16_chunk4_seed_test.sql` PASSES; `40_perception_test.sql` PASSES (now 12); all other 0A/0B files stay green.

- [ ] **Step 6: Commit**

```bash
git add core/db/seeds/seed_mara_0A.sql core/db/tests/16_chunk4_seed_test.sql core/db/tests/40_perception_test.sql
git commit -m "test(seed): add Sealed Note artifact fixture (chunk-4 existence vehicle); bump cast manifest 11->12"
```

---

## Task 2: `fn_entity_visible` — the existence predicate (boolean)

The boolean form of FILTER 1 ∘ `perception_subject`. Used by every per-id page (NULL → 404) and mirrored by the set-form index.

**Files:**
- Create: `core/db/migrations/20260615090001_compendium_read_functions.sql`
- Test: `core/db/tests/17_entity_visible_test.sql` (create)

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/17_entity_visible_test.sql`:

```sql
BEGIN;
SELECT plan(5);
SELECT ok(    fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1'),
              'Player sees the note exists (holds a perception about it)');
SELECT ok(NOT fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'cccccccc-cccc-cccc-cccc-cccccccccccc','a4000000-0000-0000-0000-0000000000a1'),
              'Jonas does NOT see the note (existence withheld — the new sharp condition)');
SELECT ok(    fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
              'Player sees Mara (common knowledge)');
SELECT ok(    fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'cccccccc-cccc-cccc-cccc-cccccccccccc','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
              'Jonas sees Mara (common knowledge — symmetric)');
SELECT ok(NOT fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','00000000-0000-0000-0000-000000000001'),
              'Player does NOT see O1 (unperceived, non-CK)');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/17_entity_visible_test.sql'`
Expected: FAIL — `function fn_entity_visible(...) does not exist`.

- [ ] **Step 3: Create the migration with `fn_entity_visible`**

Create `core/db/migrations/20260615090001_compendium_read_functions.sql`:

```sql
-- migrate:up
-- Chunk-4 read-only projection functions (design 2026-06-14). The perception/existence filter lives
-- HERE in SQL, pgTAP-tested; Go is a thin reader (ADR-P017). Live joins only (SPEC-009 unfired).
-- FILTER 1 (fn_visible_perceptions) and the name gate (fn_perceived_name) from migration 0002 are
-- REUSED UNCHANGED — only FILTER 2 / page envelopes are added here.

-- fn_entity_visible — existence predicate (B-1/I-3/B-2, read-side; no new engine ADR). Boolean form of
-- FILTER 1 ∘ perception_subject. CK entities pass for every viewer via universal-holder genesis rows.
CREATE FUNCTION fn_entity_visible(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT EXISTS (
    SELECT 1
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp     -- FILTER 1, unchanged
    JOIN perception_subject ps ON ps.perception_id = vp.perception_id
    WHERE ps.entity_id = p_entity_id);
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_entity_visible(uuid, uuid, uuid);
```

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/17_entity_visible_test.sql'`
Expected: PASS (5/5).

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260615090001_compendium_read_functions.sql core/db/tests/17_entity_visible_test.sql core/db/schema.sql
git commit -m "feat(db): fn_entity_visible existence predicate (FILTER 1 ∘ perception_subject)"
```

> Note: `make reset` runs `dbmate up`, which re-dumps `core/db/schema.sql`. Always stage `schema.sql` with each migration so `make schema-check` stays green.

---

## Task 3: `fn_compendium_index` (+ JSON wrapper) — the browse-path existence wall + the gate

The set-form of the same predicate, bucketed by `entity_kind` *after* the existence join. This task carries the gate-critical paired present/absent assertion.

**Files:**
- Modify: `core/db/migrations/20260615090001_compendium_read_functions.sql`
- Test: `core/db/tests/18_compendium_index_test.sql` (create)

- [ ] **Step 1: Write the failing test (the index existence gate)**

Create `core/db/tests/18_compendium_index_test.sql`:

```sql
BEGIN;
SELECT plan(7);
-- PAIRED existence asymmetry on the SAME note id (an absence-only assertion is forbidden):
SELECT ok( EXISTS (SELECT 1 FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','artifact')
             WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'note PRESENT in Player artifact index');
SELECT ok( NOT EXISTS (SELECT 1 FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','artifact')
             WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'note ABSENT from Jonas artifact index (existence not leaked — fails loud on breach)');
-- withheld name in the index is perception-layer NULL, NEVER entity_registry.canonical_name
SELECT is( (SELECT perceived_name FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','artifact')
             WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           NULL, 'note name withheld in index (NULL, not the canon name)');
-- actor index: O1..O5 absent for both viewers (symmetric negative)
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor')
             WHERE entity_id IN ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002',
               '00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004',
               '00000000-0000-0000-0000-000000000005'))::int,
           0, 'O1..O5 absent from Player actor index');
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','actor')
             WHERE entity_id IN ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002',
               '00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004',
               '00000000-0000-0000-0000-000000000005'))::int,
           0, 'O1..O5 absent from Jonas actor index');
-- CK cast present in the actor index for both
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','actor')
             WHERE entity_id IN ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
               'cccccccc-cccc-cccc-cccc-cccccccccccc'))::int,
           3, 'CK cast (Player,Mara,Jonas) present in Jonas actor index');
-- CK locations present
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','location')
             WHERE entity_id IN ('dddddddd-dddd-dddd-dddd-dddddddddddd','000000a0-0000-0000-0000-0000000000a1'))::int,
           2, 'CK locations (Tavern,Square) present in Jonas location index');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/18_compendium_index_test.sql'`
Expected: FAIL — `function fn_compendium_index(...) does not exist`.

- [ ] **Step 3: Add `fn_compendium_index` + `fn_compendium_index_json` to the migration**

In `core/db/migrations/20260615090001_compendium_read_functions.sql`, append to the `-- migrate:up` section (after `fn_entity_visible`):

```sql
-- fn_compendium_index — set-form of the existence predicate, bucketed by kind AFTER the existence
-- join (kind is a post-filter from entity_registry, never a parallel path). perceived_name is the
-- perception-layer name (fn_perceived_name), NULL when withheld — never entity_registry.canonical_name.
CREATE FUNCTION fn_compendium_index(p_world_id uuid, p_viewer_id uuid, p_kind text)
RETURNS TABLE (entity_id uuid, perceived_name text)
LANGUAGE sql STABLE AS $$
  SELECT DISTINCT er.entity_id,
         fn_perceived_name(p_world_id, p_viewer_id, er.entity_id)
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp       -- FILTER 1, unchanged
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id
  JOIN entity_registry er ON er.entity_id = ps.entity_id AND er.world_id = p_world_id
  WHERE er.entity_kind = p_kind;
$$;

-- thin JSON envelope for the endpoint (compendium_index/1). Flat per-kind list; entries may carry
-- a NULL perceived_name (existence perceived, name withheld).
CREATE FUNCTION fn_compendium_index_json(p_world_id uuid, p_viewer_id uuid, p_kind text)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT json_build_object(
    'schema_version', 'compendium_index/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'kind',      p_kind,
    'entries', coalesce(
      (SELECT json_agg(json_build_object('id', entity_id, 'perceived_name', perceived_name)
                       ORDER BY entity_id)
       FROM fn_compendium_index(p_world_id, p_viewer_id, p_kind)), '[]'::json)
  );
$$;
```

And add their drops to the `-- migrate:down` section (above the existing `fn_entity_visible` drop):

```sql
DROP FUNCTION IF EXISTS fn_compendium_index_json(uuid, uuid, text);
DROP FUNCTION IF EXISTS fn_compendium_index(uuid, uuid, text);
```

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/18_compendium_index_test.sql'`
Expected: PASS (7/7).

- [ ] **Step 5: Teeth-prove the gate (manual, then revert)**

Temporarily replace the `FROM fn_visible_perceptions(...) vp JOIN perception_subject ps ...` body of `fn_compendium_index` with an unfiltered `FROM perception_subject ps JOIN entity_registry er ...` (drop the FILTER-1 composition). Run Step 4.
Expected: assertion **"note ABSENT from Jonas artifact index"** goes **RED** while "note PRESENT in Player" stays green — proving the pair is non-vacuous. **Revert the edit** and re-run Step 4 → 7/7 green again. Do not commit the breach.

- [ ] **Step 6: Commit**

```bash
git add core/db/migrations/20260615090001_compendium_read_functions.sql core/db/tests/18_compendium_index_test.sql core/db/schema.sql
git commit -m "feat(db): fn_compendium_index existence wall + gate (note present for Player, absent for Jonas)"
```

---

## Task 4: `fn_collected_knowledge` shared core + refactor `fn_actor_page` onto it (45 stays green) + existence gate

Extract the FILTER-2/collected-knowledge core, re-point `fn_actor_page` to it, and gate it with `fn_entity_visible` (closing the latent Chunk-3 direct-id leak). `45_actor_page_test.sql` must pass **unchanged**.

**Files:**
- Modify: `core/db/migrations/20260615090001_compendium_read_functions.sql`
- Test: `core/db/tests/19_actor_page_gate_test.sql` (create)
- Unchanged regression guard: `core/db/tests/45_actor_page_test.sql` (do NOT edit)

- [ ] **Step 1: Write the failing actor-page existence-gate test**

Create `core/db/tests/19_actor_page_gate_test.sql`:

```sql
BEGIN;
SELECT plan(2);
SELECT ok( fn_actor_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb') IS NOT NULL,
           'actor page returns for a visible actor (Mara)');
SELECT ok( fn_actor_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','00000000-0000-0000-0000-000000000001') IS NULL,
           'actor page is NULL for an unperceived actor (O1) → Go maps to 404');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/19_actor_page_gate_test.sql'`
Expected: FAIL — current `fn_actor_page` returns a (withheld) page for O1, so the second assertion fails (`IS NULL` is false).

- [ ] **Step 3: Add `fn_collected_knowledge` and replace `fn_actor_page`**

In `core/db/migrations/20260615090001_compendium_read_functions.sql`, append to `-- migrate:up` (after the index functions):

```sql
-- fn_collected_knowledge — the SHARED about-ness core (FILTER 1 ∘ perception_subject ∘ genesis
-- exclusion ∘ grouping). Returns the collected_knowledge_groups JSON array (or '[]'). Identical lens
-- for actor/location/artifact pages — never reimplemented per page.
-- TRIPWIRE (design §3): the world_genesis exclusion is correct ONLY while genesis sources names
-- exclusively. If genesis ever sources a non-name perception, switch to a real name/identity marker.
CREATE FUNCTION fn_collected_knowledge(p_world_id uuid, p_viewer_id uuid, p_target_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH about AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_target_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',  json_build_object('stale', false, 'last_confirmed_label', a.in_world_label),
             'source', json_build_object('epistemic_type', a.epistemic_type,
                                         'source_event_label', a.in_world_label)
           ) AS item,
           a.valid_tick AS sort_tick
    FROM about a
  )
  SELECT CASE WHEN count(*) = 0 THEN '[]'::json
              ELSE json_build_array(json_build_object(
                     'group_key',   p_target_id::text,
                     'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_target_id),
                     'items',       coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                   ))
         END
  FROM items i;
$$;

-- fn_actor_page — REFACTORED onto the shared core + existence gate. Returns NULL when the actor is
-- not in the viewer's existence set (Go → 404), closing the latent Chunk-3 direct-id leak. The JSON
-- shape is byte-identical to Chunk-3 for any VISIBLE actor → 45_actor_page_test passes UNCHANGED.
CREATE OR REPLACE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_actor_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'actor_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id',                         p_actor_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      'perceived_role',             NULL,
      'current_synthesis',          NULL,
      'last_known_status',          NULL,
      'known_artifacts',            '[]'::json,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;
```

In the `-- migrate:down` section, add (above the index drops) a restore of the **original** `fn_actor_page` body (from migration 0002) and a drop of the core, so `down` is correct on a non-volume-reset rollback:

```sql
DROP FUNCTION IF EXISTS fn_collected_knowledge(uuid, uuid, uuid);
-- restore the Chunk-3 fn_actor_page (verbatim from migration 0002) on rollback
CREATE OR REPLACE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH about_actor AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_actor_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id', aa.perception_id, 'content', aa.content,
             'epistemic_type', aa.epistemic_type, 'occurred_at_tick', aa.valid_tick,
             'display_label', aa.in_world_label, 'confidence', aa.confidence,
             'decay', json_build_object('stale', false, 'last_confirmed_label', aa.in_world_label),
             'source', json_build_object('epistemic_type', aa.epistemic_type,
                                         'source_event_label', aa.in_world_label)
           ) AS item, aa.valid_tick AS sort_tick
    FROM about_actor aa
  ),
  groups AS (
    SELECT CASE WHEN count(*) = 0 THEN '[]'::json
                ELSE json_build_array(json_build_object(
                       'group_key', p_actor_id::text,
                       'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
                       'items', coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                     )) END AS arr
    FROM items i
  )
  SELECT json_build_object(
    'schema_version', 'actor_page/1', 'world_id', p_world_id, 'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id', p_actor_id, 'perceived_name', fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      'perceived_role', NULL, 'current_synthesis', NULL, 'last_known_status', NULL,
      'known_artifacts', '[]'::json, 'collected_knowledge_groups', (SELECT arr FROM groups),
      'inline_links', '[]'::json));
$$;
```

- [ ] **Step 4: Run the new test AND the unchanged regression guard**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/19_actor_page_gate_test.sql /work/tests/45_actor_page_test.sql'`
Expected: BOTH PASS. `45_actor_page_test.sql` is **unedited** — its green proves the refactor is behaviour-preserving (HARD CONDITION). If 45 goes red, STOP: behaviour moved.

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260615090001_compendium_read_functions.sql core/db/tests/19_actor_page_gate_test.sql core/db/schema.sql
git commit -m "refactor(db): extract fn_collected_knowledge; fn_actor_page onto shared core + existence gate (45 unchanged)"
```

---

## Task 5: `fn_location_page`

Same lens as the actor page (subject = location), location envelope; deferred lenses (`known_areas_inside`, `key_actors`) ship `[]`.

**Files:**
- Modify: `core/db/migrations/20260615090001_compendium_read_functions.sql`
- Test: `core/db/tests/20_location_page_test.sql` (create)

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/20_location_page_test.sql`:

```sql
BEGIN;
SELECT plan(5);
-- Player/Tavern: page present, schema, name via CK, contains the tavern observation
SELECT is( fn_location_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','dddddddd-dddd-dddd-dddd-dddddddddddd')->>'schema_version',
           'location_page/1', 'schema_version is location_page/1');
SELECT is( fn_location_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','dddddddd-dddd-dddd-dddd-dddddddddddd')
             ->'location'->>'perceived_name', 'Tavern', 'Tavern name via common knowledge');
SELECT ok( EXISTS (
    SELECT 1 FROM json_array_elements(
      fn_location_page('11111111-1111-1111-1111-111111111111',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','dddddddd-dddd-dddd-dddd-dddddddddddd')
        ->'location'->'collected_knowledge_groups') g,
      json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000c01'),
  'Player Tavern page contains the tavern observation');
-- Jonas/Tavern: present (CK) but empty collected knowledge
SELECT ok( fn_location_page('11111111-1111-1111-1111-111111111111',
             'cccccccc-cccc-cccc-cccc-cccccccccccc','dddddddd-dddd-dddd-dddd-dddddddddddd') IS NOT NULL,
           'Jonas Tavern page returns (Tavern is common knowledge)');
SELECT is( (SELECT count(*) FROM json_array_elements(
             fn_location_page('11111111-1111-1111-1111-111111111111',
               'cccccccc-cccc-cccc-cccc-cccccccccccc','dddddddd-dddd-dddd-dddd-dddddddddddd')
               ->'location'->'collected_knowledge_groups'))::int,
           0, 'Jonas Tavern page has empty collected knowledge (perception-bound)');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/20_location_page_test.sql'`
Expected: FAIL — `function fn_location_page(...) does not exist`.

- [ ] **Step 3: Add `fn_location_page`**

In the migration `-- migrate:up`, append:

```sql
-- fn_location_page — subject = location lens via the shared core + existence gate (NULL → 404).
-- Deferred lenses ship []: known_areas_inside (containment), key_actors (co-location). Never reads
-- location_state (B-1/I-3).
CREATE FUNCTION fn_location_page(p_world_id uuid, p_viewer_id uuid, p_location_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_location_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'location_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'location', json_build_object(
      'id',                         p_location_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_location_id),
      'part_of',                    NULL,
      'current_synthesis',          NULL,
      'last_known_status',          NULL,
      'known_areas_inside',         '[]'::json,
      'key_actors',                 '[]'::json,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_location_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;
```

In `-- migrate:down`, add: `DROP FUNCTION IF EXISTS fn_location_page(uuid, uuid, uuid);`

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/20_location_page_test.sql'`
Expected: PASS (5/5).

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260615090001_compendium_read_functions.sql core/db/tests/20_location_page_test.sql core/db/schema.sql
git commit -m "feat(db): fn_location_page (subject=location lens, CK name, deferred sub-lenses)"
```

---

## Task 6: `fn_artifact_page`

Subject = artifact lens; perceived_name is withheld (NULL) for the non-CK note; Jonas → NULL (404).

**Files:**
- Modify: `core/db/migrations/20260615090001_compendium_read_functions.sql`
- Test: `core/db/tests/21_artifact_page_test.sql` (create)

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/21_artifact_page_test.sql`:

```sql
BEGIN;
SELECT plan(4);
SELECT is( fn_artifact_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1')->>'schema_version',
           'artifact_page/1', 'schema_version is artifact_page/1');
-- name withheld (NULL) — existence perceived, canon name never substituted
SELECT ok( (fn_artifact_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1')
             ->'artifact'->>'perceived_name') IS NULL,
           'note name withheld on the artifact page (NULL, not the canon name)');
-- discovery observation present for Player
SELECT ok( EXISTS (
    SELECT 1 FROM json_array_elements(
      fn_artifact_page('11111111-1111-1111-1111-111111111111',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1')
        ->'artifact'->'collected_knowledge_groups') g,
      json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000b01'),
  'Player artifact page contains the discovery observation');
-- Jonas → NULL (existence withheld → 404)
SELECT ok( fn_artifact_page('11111111-1111-1111-1111-111111111111',
             'cccccccc-cccc-cccc-cccc-cccccccccccc','a4000000-0000-0000-0000-0000000000a1') IS NULL,
           'Jonas artifact page for the note is NULL → 404 (existence not leaked)');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/21_artifact_page_test.sql'`
Expected: FAIL — `function fn_artifact_page(...) does not exist`.

- [ ] **Step 3: Add `fn_artifact_page`**

In the migration `-- migrate:up`, append:

```sql
-- fn_artifact_page — subject = artifact lens via the shared core + existence gate (NULL → 404).
-- Carry-state (holder/owner/access, last_known_location) deferred → NULL. Never reads artifact_state.
CREATE FUNCTION fn_artifact_page(p_world_id uuid, p_viewer_id uuid, p_artifact_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_artifact_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'artifact_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'artifact', json_build_object(
      'id',                         p_artifact_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_artifact_id),
      'perceived_type',             NULL,
      'current_synthesis',          NULL,
      'last_known_location',        NULL,
      'current_holder_owner_access',NULL,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_artifact_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;
```

In `-- migrate:down`, add: `DROP FUNCTION IF EXISTS fn_artifact_page(uuid, uuid, uuid);`

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/21_artifact_page_test.sql'`
Expected: PASS (4/4).

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260615090001_compendium_read_functions.sql core/db/tests/21_artifact_page_test.sql core/db/schema.sql
git commit -m "feat(db): fn_artifact_page (subject=artifact lens, withheld name, NULL→404 gate)"
```

---

## Task 7: `fn_timeline`

Relevance lens = `holder = viewer` on the unchanged CK-inclusive FILTER 1 (ambient CK rows drop because `holder ≠ viewer`). Ordered by `valid_tick`; optional `before_tick` cursor.

**Files:**
- Modify: `core/db/migrations/20260615090001_compendium_read_functions.sql`
- Test: `core/db/tests/22_timeline_test.sql` (create)

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/22_timeline_test.sql`:

```sql
BEGIN;
SELECT plan(5);
SELECT is( fn_timeline('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->>'schema_version',
           'timeline/1', 'schema_version is timeline/1');
-- note observation is on Player's own timeline (PRESENT) ...
SELECT ok( EXISTS (SELECT 1 FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->'records') r
             WHERE r->>'perception_id'='dca70000-0000-0000-0000-000000000b01'),
           'note observation PRESENT on Player timeline');
-- ... and ABSENT from Jonas's (same perception_id — paired)
SELECT ok( NOT EXISTS (SELECT 1 FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'cccccccc-cccc-cccc-cccc-cccccccccccc')->'records') r
             WHERE r->>'perception_id'='dca70000-0000-0000-0000-000000000b01'),
           'note observation ABSENT from Jonas timeline');
-- Jonas holds nothing → empty history (honest emptiness, not omniscient canon)
SELECT is( (SELECT count(*) FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'cccccccc-cccc-cccc-cccc-cccccccccccc')->'records'))::int,
           0, 'Jonas timeline is empty');
-- before_tick=101 keeps only rows with valid_tick < 101 (tick-100 rows in, tick>=101 noise out)
SELECT ok( (SELECT coalesce(max((r->>'occurred_at_tick')::int), 0) FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 101)->'records') r) < 101,
           'before_tick=101 excludes all rows at tick >= 101');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/22_timeline_test.sql'`
Expected: FAIL — `function fn_timeline(...) does not exist`.

- [ ] **Step 3: Add `fn_timeline`**

In the migration `-- migrate:up`, append:

```sql
-- fn_timeline — the viewer's OWN perception history. FILTER 1 (unchanged, CK-inclusive) ∘ relevance
-- (holder = viewer): ambient CK rows drop because holder ≠ viewer (NOT a narrower safety wall).
-- Ordered by valid_tick (I-9: acquired_tick ≥ valid_tick). before_tick is an optional cursor.
-- Each record points to a perception_record (perception_id), never a canon row (Timeline AC#1).
CREATE FUNCTION fn_timeline(p_world_id uuid, p_viewer_id uuid, p_before_tick bigint DEFAULT NULL)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH mine AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE v.holder_id = p_viewer_id                              -- FILTER 2: relevance = own holdings
      AND (p_before_tick IS NULL OR v.valid_tick < p_before_tick)
  )
  SELECT json_build_object(
    'schema_version', 'timeline/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'records', coalesce(
      (SELECT json_agg(json_build_object(
                'perception_id',    m.perception_id,
                'content',          m.content,
                'epistemic_type',   m.epistemic_type,
                'occurred_at_tick', m.valid_tick,
                'display_label',    m.in_world_label,
                'confidence',       m.confidence,
                'decay', json_build_object('stale', false, 'last_confirmed_label', m.in_world_label))
              ORDER BY m.valid_tick, m.perception_id)
       FROM mine m), '[]'::json)
  );
$$;
```

In `-- migrate:down`, add: `DROP FUNCTION IF EXISTS fn_timeline(uuid, uuid, bigint);`

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/22_timeline_test.sql'`
Expected: PASS (5/5).

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260615090001_compendium_read_functions.sql core/db/tests/22_timeline_test.sql core/db/schema.sql
git commit -m "feat(db): fn_timeline (holder=viewer relevance on unchanged FILTER 1; before_tick cursor)"
```

---

## Task 8: Expanded-seed invariant re-run (I-1 / I-2 / I-7)

Prove the new rows are replay-clean and provenance-complete — not merely that existing assertions still pass.

**Files:**
- Test: `core/db/tests/23_chunk4_invariants_test.sql` (create)

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/23_chunk4_invariants_test.sql`:

```sql
BEGIN;
SELECT plan(4);
-- I-2 (universal provenance): both new perceptions reference an ACCEPTED event
SELECT is( (SELECT count(*) FROM perception_record pr
            JOIN canon_event ce ON ce.event_id = pr.source_event_id
            WHERE pr.source_event_id='e0000000-0000-0000-0000-0000000000d1' AND ce.status='accepted')::int,
           2, 'I-2: the two new perceptions reference an accepted event (no orphans)');
-- I-1 (replay): observation ≠ canon mutation (ADR-005) → no state_mutation introduced
SELECT is( (SELECT count(*) FROM state_mutation
            WHERE event_id='e0000000-0000-0000-0000-0000000000d1')::int,
           0, 'I-1: discovery event carries no state_mutation');
-- I-1: the note is perceived, not state-bearing → no artifact_state row (like Tavern/Square locations)
SELECT is( (SELECT count(*) FROM artifact_state
            WHERE entity_id='a4000000-0000-0000-0000-0000000000a1')::int,
           0, 'I-1: the note has no artifact_state row');
-- I-1: replay invariance still holds on the EXPANDED seed (truncate+rebuild; rolled back at ROLLBACK)
SELECT ok( replay_0A(), 'I-1: replay invariance holds on the expanded seed');
SELECT * FROM finish();
ROLLBACK;
```

> I-7 (projections writable only by the maintainer) is unaffected by seed data and remains covered by the unchanged `60_permissions_test.sql`; no new projection writes are introduced.

- [ ] **Step 2: Run it to verify it passes (data already seeded in Task 1)**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/23_chunk4_invariants_test.sql'`
Expected: PASS (4/4). (To see it as a guard, temporarily add a dummy `state_mutation` for the discovery event → assertion 2 goes RED → remove it.)

- [ ] **Step 3: Run the full suite + the by-hand replay**

Run: `make reset && make test && make replay`
Expected: all `*_test.sql` green; `make replay` prints `i1_replay_ok = t` / `replay_0a = t`.

- [ ] **Step 4: Commit**

```bash
git add core/db/tests/23_chunk4_invariants_test.sql
git commit -m "test(db): re-run I-1/I-2/I-7 on the expanded seed (replay-clean, provenance-complete)"
```

---

## Task 9: Generic Go page handler + NULL→404 + actor delegation

Replace the bespoke actor handler with a generic per-id page handler; map a NULL SQL result to 404 (the existence gate, §5.1). Add a `Match` method so multiple handlers can share the `/worlds/` prefix.

**Files:**
- Create: `core/api/pagehandler.go`
- Rewrite: `core/api/actorpage.go` (delegate to the generic handler)
- Test: `core/api/compendium_test.go` (create — shared constants + page tests)

- [ ] **Step 1: Write the failing Go tests**

Create `core/api/compendium_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	noteID       = "a4000000-0000-0000-0000-0000000000a1"
	notePID      = "dca70000-0000-0000-0000-000000000b01"
	tavernID     = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	o1ID         = "00000000-0000-0000-0000-000000000001"
	fabricatedID = "deadbeef-0000-0000-0000-000000000000"
)

func TestArtifactPage_PlayerSeesNote(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewPageHandler(pool, false, "artifacts", "fn_artifact_page")
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/artifacts/"+noteID+"/page", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), notePID) {
		t.Fatalf("discovery observation absent for Player: %s", rec.Body.String())
	}
}

func TestArtifactPage_JonasGets404_IndistinguishableFromFabricated(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewPageHandler(pool, true, "artifacts", "fn_artifact_page") // debug → honor ?viewer=
	get := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/compendium/artifacts/"+id+"/page?viewer="+jonasID, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	note := get(noteID)
	fake := get(fabricatedID)
	if note.Code != 404 {
		t.Fatalf("Jonas note page status = %d, want 404 (existence leak via 200)", note.Code)
	}
	if fake.Code != 404 {
		t.Fatalf("fabricated id status = %d, want 404", fake.Code)
	}
	if note.Body.String() != fake.Body.String() {
		t.Fatalf("note 404 distinguishable from fabricated 404: %q vs %q",
			note.Body.String(), fake.Body.String())
	}
}

func TestActorPage_UnperceivedActor404(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewActorPageHandler(pool, false) // viewer = Player
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/actors/"+o1ID+"/page", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("unperceived actor O1 status = %d, want 404 (latent leak closed)", rec.Code)
	}
}

func TestLocationPage_JonasTavern200Empty(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewPageHandler(pool, true, "locations", "fn_location_page")
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/locations/"+tavernID+"/page?viewer="+jonasID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 { // Tavern is common knowledge → exists for Jonas
		t.Fatalf("Jonas Tavern status = %d, want 200 (CK)", rec.Code)
	}
	if strings.Contains(rec.Body.String(), notePID) {
		t.Fatalf("Jonas Tavern leaked a Player perception")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd core/api && make -C ../.. reset && go test ./... -run 'ArtifactPage|ActorPage_Unperceived|LocationPage' -v`
Expected: FAIL — `NewPageHandler` undefined (and the DB-backed assertions can't run yet).

> The Go tests require the seeded DB reachable at `localhost:5432` (default `DATABASE_URL`). `make reset` (run from the repo root) brings up and seeds it.

- [ ] **Step 3: Create the generic page handler**

Create `core/api/pagehandler.go`:

```go
package main

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pageHandler serves GET /worlds/{w}/compendium/{kind}/{id}/page for any entity kind. It is a THIN
// READER: the entire perception/existence filter lives in the SQL function (ADR-P017). A NULL result
// means the entity is not in the viewer's existence set (§5.1) → 404, indistinguishable from a
// nonexistent id.
type pageHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	fn   string // SQL function name, e.g. "fn_artifact_page" (internal constant, never user input)
	re   *regexp.Regexp
}

// NewPageHandler builds a handler for the given URL kind segment ("actors"/"locations"/"artifacts")
// and SQL function. debug enables the creator/debug ?viewer= override (run through the same gate).
func NewPageHandler(pool *pgxpool.Pool, debug bool, kind, fn string) http.Handler {
	return &pageHandler{
		pool: pool, dbg: debug, fn: fn,
		re: regexp.MustCompile(
			`^/worlds/([0-9a-fA-F-]{36})/compendium/` + kind + `/([0-9a-fA-F-]{36})/page$`),
	}
}

// Match reports whether this handler owns the request (used by the router in main.go).
func (h *pageHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *pageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.re.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID, entityID := m[1], m[2]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var payload []byte
	// h.fn is an internal constant (never request-derived); ids are constrained by the route regex.
	err = h.pool.QueryRow(ctx,
		`SELECT `+h.fn+`($1, $2, $3)::text`, worldID, viewerID, entityID).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	if payload == nil { // SQL NULL → not in viewer's existence set (§5.1) → 404
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
```

- [ ] **Step 4: Rewrite `actorpage.go` to delegate**

Replace the entire contents of `core/api/actorpage.go` with:

```go
package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewActorPageHandler returns the read-only Actor-page handler, delegating to the generic page
// handler (route GET /worlds/{w}/compendium/actors/{id}/page → fn_actor_page). The filter and the
// existence 404 live in SQL (ADR-P017 / §5.1).
func NewActorPageHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return NewPageHandler(pool, debug, "actors", "fn_actor_page")
}
```

- [ ] **Step 5: Run to verify it passes (incl. the unchanged Chunk-3 Go tests)**

Run: `cd core/api && make -C ../.. reset && go test ./... -v`
Expected: PASS — new page tests green; `TestActorPage_DefaultViewerSeesAboutMara` and `TestActorPage_DebugViewerJonas_NoLeak` (unchanged, Mara is CK → still 200) green.

- [ ] **Step 6: Commit**

```bash
git add core/api/pagehandler.go core/api/actorpage.go core/api/compendium_test.go
git commit -m "feat(api): generic page handler with NULL->404 existence gate; actor handler delegates"
```

---

## Task 10: Index + Timeline handlers + router wiring

**Files:**
- Create: `core/api/indexhandler.go`
- Create: `core/api/timelinehandler.go`
- Modify: `core/api/main.go` (route all compendium handlers)
- Test: `core/api/compendium_index_test.go` (create)

- [ ] **Step 1: Write the failing tests**

Create `core/api/compendium_index_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArtifactIndex_PlayerHasNote_JonasDoesNot(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewIndexHandler(pool, true, "artifacts", "artifact") // debug → honor ?viewer=
	get := func(viewer string) string {
		req := httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/compendium/artifacts?viewer="+viewer, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("index status = %d", rec.Code)
		}
		return rec.Body.String()
	}
	if !strings.Contains(get(playerID), noteID) {
		t.Fatalf("note absent from Player artifact index")
	}
	if strings.Contains(get(jonasID), noteID) {
		t.Fatalf("LEAK: note present in Jonas artifact index")
	}
}

func TestTimeline_PlayerNonEmpty_JonasEmpty(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewTimelineHandler(pool, true)
	get := func(viewer string) string {
		req := httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/compendium/timeline?viewer="+viewer, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("timeline status = %d", rec.Code)
		}
		return rec.Body.String()
	}
	if !strings.Contains(get(playerID), notePID) {
		t.Fatalf("Player timeline missing the note observation")
	}
	jonas := get(jonasID)
	if strings.Contains(jonas, notePID) {
		t.Fatalf("LEAK: note observation in Jonas timeline")
	}
	if !strings.Contains(jonas, `"records":[]`) {
		t.Fatalf("Jonas timeline not empty: %s", jonas)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd core/api && make -C ../.. reset && go test ./... -run 'ArtifactIndex|Timeline_Player' -v`
Expected: FAIL — `NewIndexHandler` / `NewTimelineHandler` undefined.

- [ ] **Step 3: Create the index handler**

Create `core/api/indexhandler.go`:

```go
package main

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// indexHandler serves GET /worlds/{w}/compendium/{urlKind} → compendium_index/1 (flat per-kind list).
// The existence filter is fn_compendium_index in SQL (ADR-P017); an unperceived entity is simply
// absent from entries (never redacted/placeholdered).
type indexHandler struct {
	pool       *pgxpool.Pool
	dbg        bool
	entityKind string // entity_registry.entity_kind, e.g. "artifact"
	re         *regexp.Regexp
}

// NewIndexHandler builds an index handler for a URL segment ("artifacts") mapped to an entity_kind
// ("artifact").
func NewIndexHandler(pool *pgxpool.Pool, debug bool, urlKind, entityKind string) http.Handler {
	return &indexHandler{
		pool: pool, dbg: debug, entityKind: entityKind,
		re: regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/compendium/` + urlKind + `$`),
	}
}

func (h *indexHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.re.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var payload []byte
	err = h.pool.QueryRow(ctx,
		`SELECT fn_compendium_index_json($1, $2, $3)::text`, worldID, viewerID, h.entityKind).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
```

- [ ] **Step 4: Create the timeline handler**

Create `core/api/timelinehandler.go`:

```go
package main

import (
	"context"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// timelineHandler serves GET /worlds/{w}/compendium/timeline?before_tick=… → timeline/1.
// Relevance lens (holder = viewer) lives in fn_timeline (ADR-P017).
type timelineHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	re   *regexp.Regexp
}

func NewTimelineHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &timelineHandler{
		pool: pool, dbg: debug,
		re: regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/compendium/timeline$`),
	}
}

func (h *timelineHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *timelineHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.re.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var beforeTick *int64 // NULL unless a valid before_tick is supplied
	if s := r.URL.Query().Get("before_tick"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			http.Error(w, "before_tick must be an integer", http.StatusBadRequest)
			return
		}
		beforeTick = &v
	}

	var payload []byte
	err = h.pool.QueryRow(ctx,
		`SELECT fn_timeline($1, $2, $3)::text`, worldID, viewerID, beforeTick).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
```

- [ ] **Step 5: Wire the router in `main.go`**

Replace the entire contents of `core/api/main.go` with:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// matcher is any compendium handler that can claim a request (all share the /worlds/ prefix).
type matcher interface {
	http.Handler
	Match(*http.Request) bool
}

// router dispatches to the first handler whose Match returns true; otherwise 404.
type router struct{ handlers []matcher }

func (rt *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, h := range rt.handlers {
		if h.Match(r) {
			h.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	debug := os.Getenv("DREAMCHAT_MODE") == "debug"
	rt := &router{handlers: []matcher{
		NewPageHandler(pool, debug, "actors", "fn_actor_page").(matcher),
		NewPageHandler(pool, debug, "locations", "fn_location_page").(matcher),
		NewPageHandler(pool, debug, "artifacts", "fn_artifact_page").(matcher),
		NewIndexHandler(pool, debug, "actors", "actor").(matcher),
		NewIndexHandler(pool, debug, "locations", "location").(matcher),
		NewIndexHandler(pool, debug, "artifacts", "artifact").(matcher),
		NewTimelineHandler(pool, debug).(matcher),
	}}

	mux := http.NewServeMux()
	mux.Handle("/worlds/", rt)

	addr := ":8080"
	log.Printf("dreamchat world backend (read-only compendium API) on %s (debug=%v)", addr, debug)
	log.Fatal(http.ListenAndServe(addr, mux))
}
```

- [ ] **Step 6: Run to verify it passes**

Run: `cd core/api && make -C ../.. reset && go test ./... -v`
Expected: PASS — index and timeline tests green; all prior Go tests green.

- [ ] **Step 7: Commit**

```bash
git add core/api/indexhandler.go core/api/timelinehandler.go core/api/main.go core/api/compendium_index_test.go
git commit -m "feat(api): per-type index + timeline handlers; router dispatch over /worlds/ prefix"
```

---

## Task 11: Publish the four versioned JSON schemas

Source of truth the frontend repo codegens from. Mirror the structure of the existing `actor_page.v1.schema.json`.

**Files:**
- Create: `core/api/schema/location_page.v1.schema.json`
- Create: `core/api/schema/artifact_page.v1.schema.json`
- Create: `core/api/schema/timeline.v1.schema.json`
- Create: `core/api/schema/compendium_index.v1.schema.json`

- [ ] **Step 1: Write `location_page.v1.schema.json`**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "location_page/1",
  "title": "LocationPage",
  "type": "object",
  "required": ["schema_version", "world_id", "viewer_id", "location"],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "const": "location_page/1" },
    "world_id":  { "type": "string", "format": "uuid" },
    "viewer_id": { "type": "string", "format": "uuid" },
    "location": {
      "type": "object",
      "required": ["id","perceived_name","part_of","current_synthesis","last_known_status","known_areas_inside","key_actors","collected_knowledge_groups","inline_links"],
      "additionalProperties": false,
      "properties": {
        "id": { "type": "string", "format": "uuid" },
        "perceived_name":    { "type": ["string","null"] },
        "part_of":           { "type": ["string","null"] },
        "current_synthesis": { "type": ["string","null"] },
        "last_known_status": { "type": ["string","null"] },
        "known_areas_inside":{ "type": "array", "items": { "type": "object" } },
        "key_actors":        { "type": "array", "items": { "type": "object" } },
        "inline_links":      { "type": "array", "items": { "type": "object" } },
        "collected_knowledge_groups": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["group_key","group_label","items"],
            "additionalProperties": false,
            "properties": {
              "group_key":   { "type": "string" },
              "group_label": { "type": ["string","null"] },
              "items": {
                "type": "array",
                "items": {
                  "type": "object",
                  "required": ["perception_id","content","epistemic_type","occurred_at_tick","display_label","confidence","decay","source"],
                  "additionalProperties": false,
                  "properties": {
                    "perception_id":   { "type": "string", "format": "uuid" },
                    "content":         { "type": "string" },
                    "epistemic_type":  { "type": "string" },
                    "occurred_at_tick":{ "type": "integer" },
                    "display_label":   { "type": ["string","null"] },
                    "confidence":      { "type": ["number","null"] },
                    "decay":           { "type": "object" },
                    "source":          { "type": "object" }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
```

- [ ] **Step 2: Write `artifact_page.v1.schema.json`**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "artifact_page/1",
  "title": "ArtifactPage",
  "type": "object",
  "required": ["schema_version", "world_id", "viewer_id", "artifact"],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "const": "artifact_page/1" },
    "world_id":  { "type": "string", "format": "uuid" },
    "viewer_id": { "type": "string", "format": "uuid" },
    "artifact": {
      "type": "object",
      "required": ["id","perceived_name","perceived_type","current_synthesis","last_known_location","current_holder_owner_access","collected_knowledge_groups","inline_links"],
      "additionalProperties": false,
      "properties": {
        "id": { "type": "string", "format": "uuid" },
        "perceived_name":             { "type": ["string","null"] },
        "perceived_type":             { "type": ["string","null"] },
        "current_synthesis":          { "type": ["string","null"] },
        "last_known_location":        { "type": ["string","null"] },
        "current_holder_owner_access":{ "type": ["string","null"] },
        "inline_links":               { "type": "array", "items": { "type": "object" } },
        "collected_knowledge_groups": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["group_key","group_label","items"],
            "additionalProperties": false,
            "properties": {
              "group_key":   { "type": "string" },
              "group_label": { "type": ["string","null"] },
              "items": {
                "type": "array",
                "items": {
                  "type": "object",
                  "required": ["perception_id","content","epistemic_type","occurred_at_tick","display_label","confidence","decay","source"],
                  "additionalProperties": false,
                  "properties": {
                    "perception_id":   { "type": "string", "format": "uuid" },
                    "content":         { "type": "string" },
                    "epistemic_type":  { "type": "string" },
                    "occurred_at_tick":{ "type": "integer" },
                    "display_label":   { "type": ["string","null"] },
                    "confidence":      { "type": ["number","null"] },
                    "decay":           { "type": "object" },
                    "source":          { "type": "object" }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
```

- [ ] **Step 3: Write `timeline.v1.schema.json`**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "timeline/1",
  "title": "Timeline",
  "type": "object",
  "required": ["schema_version", "world_id", "viewer_id", "records"],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "const": "timeline/1" },
    "world_id":  { "type": "string", "format": "uuid" },
    "viewer_id": { "type": "string", "format": "uuid" },
    "records": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["perception_id","content","epistemic_type","occurred_at_tick","display_label","confidence","decay"],
        "additionalProperties": false,
        "properties": {
          "perception_id":   { "type": "string", "format": "uuid" },
          "content":         { "type": "string" },
          "epistemic_type":  { "type": "string" },
          "occurred_at_tick":{ "type": "integer" },
          "display_label":   { "type": ["string","null"] },
          "confidence":      { "type": ["number","null"] },
          "decay":           { "type": "object" }
        }
      }
    }
  }
}
```

- [ ] **Step 4: Write `compendium_index.v1.schema.json`**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "compendium_index/1",
  "title": "CompendiumIndex",
  "type": "object",
  "required": ["schema_version", "world_id", "viewer_id", "kind", "entries"],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "const": "compendium_index/1" },
    "world_id":  { "type": "string", "format": "uuid" },
    "viewer_id": { "type": "string", "format": "uuid" },
    "kind":      { "type": "string", "enum": ["actor","location","artifact"] },
    "entries": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id","perceived_name"],
        "additionalProperties": false,
        "properties": {
          "id":             { "type": "string", "format": "uuid" },
          "perceived_name": { "type": ["string","null"] }
        }
      }
    }
  }
}
```

- [ ] **Step 5: Sanity-check the schemas are valid JSON**

Run: `for f in core/api/schema/*.v1.schema.json; do python3 -c "import json,sys; json.load(open('$f'))" && echo "OK $f"; done`
Expected: `OK …` for all five files (the four new + the existing `actor_page`).

- [ ] **Step 6: Commit**

```bash
git add core/api/schema/location_page.v1.schema.json core/api/schema/artifact_page.v1.schema.json core/api/schema/timeline.v1.schema.json core/api/schema/compendium_index.v1.schema.json
git commit -m "docs(api): publish location_page/1, artifact_page/1, timeline/1, compendium_index/1 schemas"
```

---

## Task 12: Full green sweep + schema-drift check + ledger touchpoints

**Files:**
- Modify: `docs/superpowers/specs/2026-06-14-chunk-4-compendium-read-only-design.md` (mark Status: implemented) — optional
- Verify only: whole repo

- [ ] **Step 1: Full pgTAP suite + replay**

Run: `make reset && make test && make replay`
Expected: every `*_test.sql` green (including the unchanged 0A/0B files, `45_actor_page_test`, and the new `16`–`23`); `make replay` returns true.

- [ ] **Step 2: Schema-drift guard**

Run: `make schema-check`
Expected: no diff in `core/db/schema.sql` (it was re-dumped and committed with each migration step).

- [ ] **Step 3: Full Go test suite**

Run: `cd core/api && make -C ../.. reset && go test ./... -v && go vet ./...`
Expected: all tests PASS, `go vet` clean.

- [ ] **Step 4: Confirm no frontend code leaked into the backend repo**

Run: `git ls-files | grep -E '^frontend/|\.tsx?$' || echo "clean — no frontend in backend repo (D-7/D-10)"`
Expected: `clean — no frontend in backend repo`.

- [ ] **Step 5: Final commit (if any doc status touched)**

```bash
git add -A
git commit -m "chore(chunk-4): backend leg green — pages, index, timeline, existence wall, published schemas"
```

> **Gate (operator-run, after this plan):** the founder browses Mara's world as Player and (via `?viewer=`) Jonas across Location/Artifact/Timeline + the index, confirming by eye in DevTools: every page perception-bound; the Sealed Note present in Player's index/`200` page and absent (`404`) for Jonas; the planted secret never in Jonas's payloads. On green + by-eye pass, tag `chunk-4-compendium-gate` on the verified backend `main` merge. The frontend leg is a separate plan/PR to `dreamchat-frontend`.

---

## Self-Review

**Spec coverage:**
- Locations / Artifacts / Timeline / Index pages → Tasks 5, 6, 7, 3. ✓
- FILTER 1 + name gate reused unchanged → Tasks 2–7 only ADD functions; `fn_visible_perceptions`/`fn_perceived_name` never edited. ✓
- Shared core + `fn_actor_page` refactor with 45 unchanged → Task 4 (Step 4 runs 45 unedited). ✓
- Existence wall, both channels (index + per-id 404) → Task 3 (index gate, teeth) + Tasks 4/9 (NULL→404) + Task 9 Go test (404 indistinguishability). ✓
- Withheld-name in index/page (perception-layer NULL, never canonical_name) → Task 3 Step 1 assertion, Task 6 assertion. ✓
- Timeline holder=viewer relevance, valid_tick order, before_tick → Task 7. ✓
- Seed additions (artifact + observation event + 2 perceptions, subject=entity-alone), cast 11→12, created_by_event, (100,1) non-collision → Task 1. ✓
- Expanded-seed I-1/I-2/I-7 → Task 8. ✓
- Four published schemas → Task 11. ✓
- Go thin readers, ?viewer= through the same gate → Tasks 9, 10. ✓
- One PR per leg, no frontend in backend → header + Task 12 Step 4. ✓

**Placeholder scan:** no TBD/TODO; every code/SQL/JSON block is complete and concrete. ✓

**Type/name consistency:** SQL function names (`fn_entity_visible`, `fn_collected_knowledge`, `fn_location_page`, `fn_artifact_page`, `fn_timeline`, `fn_compendium_index`, `fn_compendium_index_json`) are used identically in migrations, Go (`h.fn` strings), and tests. Fixed UUIDs match across seed, pgTAP, and Go constants. Payload envelope keys match the published schemas (e.g. `location`→`part_of`/`known_areas_inside`/`key_actors`; `artifact`→`perceived_type`/`last_known_location`/`current_holder_owner_access`; `timeline`→`records`; `compendium_index`→`entries`/`kind`). ✓
