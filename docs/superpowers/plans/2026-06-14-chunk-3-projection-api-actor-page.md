# Chunk 3 — Read-only Projection API + Perception-bound Actor Page — Implementation Plan

> **⚠️ SUPERSEDED (frontend location only) — 2026-06-14.** The in-repo `frontend/` shell
> described below was extracted to its own repo, **github.com/zakkriel/dreamchat-frontend**,
> per design §6 / D-7 / D-10. The backend repo is now backend-only. Treat every `frontend/…`
> path and `cd frontend` step in this plan as historical record, not live instructions — do
> **not** recreate `frontend/` here. The API contract source of truth
> (`core/api/schema/actor_page.v1.schema.json`) stays in this repo; the frontend repo generates
> its types from it. All backend/SQL/Go steps below remain accurate.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a read-only, perception-bound projection API serving the Actor-page payload for `(world, viewer, actor)`, plus a minimal frontend that renders one Actor page from seeded data — proving "can the user inspect a world and trust it?" (Validation Ladder Q2).

**Architecture:** The epistemic safety wall (B-1/I-3) lives in SQL functions, pgTAP-tested. Go is a thin reader that calls `fn_actor_page` and never reimplements the filter (ADR-P017 binding). About-ness is an explicit `perception_subject` junction (ADR-035). The FE is presentation-only (D-7), types generated from a single JSON-Schema source of truth. Live SQL joins only — no materialized projections/snapshots/sharding (SPEC-009 tripwires unfired).

**Tech Stack:** PostgreSQL + pgTAP + dbmate (existing) · Go 1.22 + pgx/v5 + net/http (new, first Go surface) · JSON Schema → TypeScript codegen · Vite + React + TypeScript (FE).

**Design source:** `docs/superpowers/specs/2026-06-14-chunk-3-projection-api-actor-page-design.md` (approved 2026-06-14).

**Process:** one worktree · one plan · one PR · TDD iron law (failing test first, always) · gate red → stop. Branch the worktree before Task 1 (see Phase 0). ~~FE lives under `frontend/` in this repo for Chunk 3 so the whole chunk is one PR (founder-confirmed); migration to a dedicated repo per D-10/Bridge is deferred to when it grows.~~ **Superseded 2026-06-14 (see banner above): the FE was extracted to github.com/zakkriel/dreamchat-frontend per design §6 / D-10; this repo is backend-only.**

**Fixed UUIDs (from the seed + new fixtures):**
- world `11111111-1111-1111-1111-111111111111`
- Player `aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa` · Mara `bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb` · Jonas `cccccccc-cccc-cccc-cccc-cccccccccccc`
- Tavern `dddddddd-dddd-dddd-dddd-dddddddddddd` · Common Knowledge (faction) `eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee` · Square `000000a0-0000-0000-0000-0000000000a1`
- O1 `00000000-0000-0000-0000-000000000001`
- E1 `e0000000-0000-0000-0000-000000000001` · E102 `e0000000-0000-0000-0000-000000000102`
- **NEW** genesis event `e0000000-0000-0000-0000-0000000000e0`
- **NEW** Player-private-about-Mara perception `dca70000-0000-0000-0000-000000000a01`

---

## Phase 0 — Worktree & branch

### Task 0: Create the chunk worktree

**Files:** none (workspace setup)

- [ ] **Step 1: Create an isolated worktree/branch** (superpowers:using-git-worktrees if available; else native)

```bash
cd /Users/pelao/REPOS/dreamchat/dreamchat-world-backend
git worktree add ../dreamchat-chunk-3 -b chunk-3-projection-api
cd ../dreamchat-chunk-3
```

- [ ] **Step 2: Verify clean baseline (all current tests green before touching anything)**

Run:
```bash
make reset && make test
```
Expected: pgTAP suite all green (this is the 0A/0B baseline the gate must preserve).

- [ ] **Step 3: Carry the approved spec + plan into the branch** (they live uncommitted in the original checkout, intentionally never committed to `main`, so the fresh worktree does not contain them — copy them in, then commit)

```bash
SRC=/Users/pelao/REPOS/dreamchat/dreamchat-world-backend/docs/superpowers
mkdir -p docs/superpowers/specs docs/superpowers/plans
cp "$SRC/specs/2026-06-14-chunk-3-projection-api-actor-page-design.md" docs/superpowers/specs/
cp "$SRC/plans/2026-06-14-chunk-3-projection-api-actor-page.md"        docs/superpowers/plans/
git add docs/superpowers/specs/2026-06-14-chunk-3-projection-api-actor-page-design.md docs/superpowers/plans/2026-06-14-chunk-3-projection-api-actor-page.md
git commit -m "docs: chunk-3 design spec + implementation plan"
```
> If the original checkout already has these committed on some branch, `git checkout <branch> -- <paths>` is equivalent. The point: they must exist in the chunk worktree before Task 1.

---

## Phase 1 — Schema: `perception_subject` junction (ADR-035, SPEC-008)

### Task 1: `perception_subject` table (TDD — failing schema test first)

**Files:**
- Test: `core/db/tests/12_perception_subject_schema_test.sql` (create)
- Create: `core/db/migrations/20260614090001_perception_subject.sql`
- Modify (regenerated): `core/db/schema.sql`

- [ ] **Step 1: Write the failing schema test**

Create `core/db/tests/12_perception_subject_schema_test.sql`:
```sql
BEGIN;
SELECT plan(7);
SELECT has_table('perception_subject', 'perception_subject exists (ADR-035)');
SELECT col_is_pk('perception_subject', ARRAY['perception_id','entity_id'],
  'perception_subject PK is (perception_id, entity_id)');
SELECT has_column('perception_subject', 'world_id', 'carries world_id from birth (SPEC-009 tenant key)');
SELECT col_type_is('perception_subject', 'world_id', 'uuid', 'world_id is UUID');
SELECT col_not_null('perception_subject', 'world_id', 'world_id NOT NULL');
SELECT fk_ok('perception_subject', 'perception_id', 'perception_record', 'perception_id',
  'perception_id FK → perception_record');
SELECT has_index('perception_subject', 'idx_ps_entity', 'index on entity_id exists');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to confirm it fails**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/12_perception_subject_schema_test.sql
```
Expected: FAIL — `relation "perception_subject" does not exist`.

- [ ] **Step 3: Write the migration**

Create `core/db/migrations/20260614090001_perception_subject.sql`:
```sql
-- migrate:up
-- ADR-035 (Proposed → Accepted under chunk-3 gate): perception about-ness is an explicit junction,
-- not a derivation from the source event's participants (SPEC-008). DELTA vs ADR-035's two-column
-- sketch: world_id carried from birth (SPEC-009 tenant-key posture). New table → carrying the
-- tenant key costs zero migration; reopening frozen tables would require a firing trigger (§7 of the
-- design / SPEC-009). Additive only: no existing engine ADR/invariant/DDL column is modified.
CREATE TABLE perception_subject (
  perception_id UUID NOT NULL REFERENCES perception_record(perception_id),
  entity_id     UUID NOT NULL,
  world_id      UUID NOT NULL,
  PRIMARY KEY (perception_id, entity_id)
);
CREATE INDEX idx_ps_entity ON perception_subject (entity_id);
CREATE INDEX idx_ps_world  ON perception_subject (world_id);

-- DELETE guard: about-ness is append-only like its parent perception (ADR-006).
-- forbid_delete() was created in migration 0002 (canon spine) and reused across the schema.
CREATE TRIGGER trg_perception_subject_no_delete
  BEFORE DELETE ON perception_subject FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TRIGGER IF EXISTS trg_perception_subject_no_delete ON perception_subject;
DROP TABLE IF EXISTS perception_subject;
```

- [ ] **Step 4: Apply and re-run the test**

Run:
```bash
make migrate && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/12_perception_subject_schema_test.sql
```
Expected: PASS (7/7).

- [ ] **Step 5: Confirm schema.sql regenerated and no unexpected drift**

Run:
```bash
make schema-check || true   # dbmate dumps schema.sql during `make migrate`; this confirms it's committed-clean after we add it
git add core/db/schema.sql
```
Expected: `core/db/schema.sql` now contains `perception_subject`.

- [ ] **Step 6: Commit**

```bash
git add core/db/migrations/20260614090001_perception_subject.sql core/db/tests/12_perception_subject_schema_test.sql core/db/schema.sql
git commit -m "feat(db): add perception_subject junction (ADR-035) with world_id tenant key"
```

---

## Phase 2 — Seed additions (deterministic; designed to miss every existing scoped assertion)

### Task 2: Genesis event + CK name perceptions + Player-private-about-Mara row + backfill

**Files:**
- Modify: `core/db/seeds/seed_mara_0A.sql` (insert before final `COMMIT;`)
- Modify: `core/db/tests/helpers.sql` (add fixture vars)
- Test: `core/db/tests/14_perception_subject_data_test.sql` (create)

- [ ] **Step 1: Write the failing data test (subjects + backfill + agreement)**

Create `core/db/tests/14_perception_subject_data_test.sql`:
```sql
BEGIN;
SELECT plan(6);
-- positive: every ORIGINAL event-derived perception got subjects from its source event's participants
SELECT is(
  (SELECT count(*) FROM perception_record pr
   WHERE NOT EXISTS (SELECT 1 FROM perception_subject ps WHERE ps.perception_id = pr.perception_id))::int,
  0, 'every perception has at least one subject (backfill + explicit)');
-- positive: principal-cast name perceptions exist, CK-held, genesis-sourced, subject = the named entity
SELECT is(
  (SELECT count(*) FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.event_type='world_genesis'
   JOIN perception_subject ps ON ps.perception_id = pr.perception_id
   WHERE pr.holder_id='eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee' AND pr.epistemic_type='public')::int,
  5, '5 common-knowledge name perceptions (Player,Mara,Jonas,Tavern,Square)');
-- negative: noise actor O1 has NO common-knowledge name perception (withhold path is real)
SELECT is(
  (SELECT count(*) FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.event_type='world_genesis'
   JOIN perception_subject ps ON ps.perception_id = pr.perception_id
   WHERE ps.entity_id='00000000-0000-0000-0000-000000000001')::int,
  0, 'O1 deliberately unnamed (no genesis name perception)');
-- the gate fixture: Player-private-about-Mara perception exists with fixed id, direct, subject = Mara only
SELECT is(
  (SELECT epistemic_type FROM perception_record WHERE perception_id='dca70000-0000-0000-0000-000000000a01'),
  'direct', 'about-Mara fixture is a direct perception');
SELECT is(
  (SELECT count(*) FROM perception_subject WHERE perception_id='dca70000-0000-0000-0000-000000000a01')::int,
  1, 'about-Mara fixture has exactly one subject');
SELECT is(
  (SELECT entity_id FROM perception_subject WHERE perception_id='dca70000-0000-0000-0000-000000000a01'),
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid, 'about-Mara fixture subject is Mara only (not Player)');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to confirm it fails**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/14_perception_subject_data_test.sql
```
Expected: FAIL (no genesis event, no subjects, fixture missing).

- [ ] **Step 3: Add the seed block** — insert this immediately **before** the final `COMMIT;` in `core/db/seeds/seed_mara_0A.sql`:

```sql
-- =====================================================================================
-- Chunk-3 additions (design 2026-06-14). Deterministic. Chosen to miss every existing
-- scoped 0A assertion: name perceptions are 'public' sourced to world_genesis (≠ E102);
-- the about-Mara fixture is 'direct' sourced to E1 (≠ Player's 'shared'); no state_mutation
-- is added (replay/golden projections untouched).
-- =====================================================================================

-- (G) world_genesis @ tick 0 — sources common-knowledge identity (names). No state mutation.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000e0','11111111-1111-1111-1111-111111111111',
        'world_genesis','the world is established; its principal figures are publicly known',0,0,
        'Genesis','accepted', now(), 'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-0000000000e0','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','faction','source');

-- (N) Common-knowledge NAME perceptions — principal cast only; held by Common Knowledge (PUB),
-- public, sourced to genesis. content = the canonical name (read at projection time via the
-- perception layer, NEVER a raw entity_registry read — going-in 5). O1..O5 deliberately omitted
-- so fn_perceived_name's WITHHOLD path is exercised on real seed rows. Fixed perception_ids so
-- the explicit subject links are unambiguous.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick, visibility_scope) VALUES
 ('ace00000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Player','public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000b1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Mara',  'public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000c1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Jonas', 'public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Tavern','public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000f1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Square','public',0,0,'public');
-- explicit subjects for the name perceptions (one entity each)
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('ace00000-0000-0000-0000-0000000000a1','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000b1','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000c1','cccccccc-cccc-cccc-cccc-cccccccccccc','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000d1','dddddddd-dddd-dddd-dddd-dddddddddddd','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000f1','000000a0-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111');

-- (A) Player-private-about-Mara — the gate fixture. Genuinely about Mara (content-subject = Mara,
-- who is also an E1 participant → junction ⊆ derivation, future-proof). 'direct' (≠ Player's
-- 'shared' of E1, so the existing scoped assertion holds). Private → invisible to Jonas. Fixed id.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 ('dca70000-0000-0000-0000-000000000a01','11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-000000000001','Mara listened intently and seemed unsettled','direct',100,100);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('dca70000-0000-0000-0000-000000000a01','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111');

-- (B) Generic about-ness backfill for the ORIGINAL event-derived perceptions only (ADR-035:
-- subjects = source event participants). NOT EXISTS skips rows that already carry explicit subjects
-- (the names and the about-Mara fixture), so those stay precise and are never over-attributed.
INSERT INTO perception_subject (perception_id, entity_id, world_id)
SELECT pr.perception_id, ep.entity_id, pr.world_id
FROM perception_record pr
JOIN event_participant ep ON ep.event_id = pr.source_event_id
WHERE NOT EXISTS (SELECT 1 FROM perception_subject ps WHERE ps.perception_id = pr.perception_id)
ON CONFLICT (perception_id, entity_id) DO NOTHING;
```

- [ ] **Step 4: Add fixture vars to `core/db/tests/helpers.sql`** (append):

```sql
\set square_id  '000000a0-0000-0000-0000-0000000000a1'
\set o1_id      '00000000-0000-0000-0000-000000000001'
\set genesis_id 'e0000000-0000-0000-0000-0000000000e0'
\set about_mara_pid 'dca70000-0000-0000-0000-000000000a01'
\set ck_id      'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee'
```

- [ ] **Step 5: Re-run the data test**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/14_perception_subject_data_test.sql
```
Expected: PASS (6/6).

- [ ] **Step 6: Confirm NO existing 0A/0B assertion regressed (the whole point of the seed design)**

Run:
```bash
make test
```
Expected: entire suite PASS — including `40_perception_test.sql`, `50_provenance_test.sql`, `70_determinism_guards_test.sql`, `80_golden_projection_test.sql`, `90_replay_test.sql`. If any total-count assertion in `80`/`90` trips (not expected — no state_mutation added), STOP and reconcile it as a seed-shape assertion update in this same task; do not weaken an invariant.

- [ ] **Step 7: Confirm replay invariance still holds (I-1)**

Run:
```bash
make replay
```
Expected: returns `t` (true).

- [ ] **Step 8: Commit**

```bash
git add core/db/seeds/seed_mara_0A.sql core/db/tests/helpers.sql core/db/tests/14_perception_subject_data_test.sql
git commit -m "feat(seed): genesis identity + CK names + about-Mara gate fixture; backfill perception_subject"
```

---

## Phase 3 — SQL functions: the two filters + name gate (the epistemic wall)

### Task 3: `fn_visible_perceptions` — FILTER 1 safety wall (TDD — leak test first)

**Files:**
- Test: `core/db/tests/42_visible_perceptions_test.sql` (create)
- Create: `core/db/migrations/20260614090002_projection_functions.sql` (functions added incrementally across Tasks 3–5; create file here with `fn_visible_perceptions`)
- Modify (regenerated): `core/db/schema.sql`

- [ ] **Step 1: Write the failing safety-wall test (I-3 made executable)**

Create `core/db/tests/42_visible_perceptions_test.sql`:
```sql
BEGIN;
SELECT plan(4);
-- Player sees his own + common-knowledge perceptions
SELECT ok(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')) > 0,
  'Player has visible perceptions');
-- GATE-CRITICAL NEGATIVE: a perception held only by Mara is ABSENT for viewer=Jonas
SELECT is(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc')
   WHERE holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')::int,
  0, 'I-3: no Mara-held perception is visible to Jonas');
-- common knowledge IS visible to everyone (the public ledger record, held by PUB)
SELECT ok(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc')
   WHERE holder_id='eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee') > 0,
  'common-knowledge holder perceptions are visible to Jonas');
-- closed perceptions (invalid_tick / expired_at) never returned (none in seed → boundary holds at 0)
SELECT is(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')
   WHERE invalid_tick IS NOT NULL OR expired_at IS NOT NULL)::int,
  0, 'closed perceptions are never returned');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to confirm it fails**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/42_visible_perceptions_test.sql
```
Expected: FAIL — `function fn_visible_perceptions(...) does not exist`.

- [ ] **Step 3: Create the functions migration with `fn_visible_perceptions`**

Create `core/db/migrations/20260614090002_projection_functions.sql`:
```sql
-- migrate:up
-- Chunk-3 read-only projection functions. The perception/safety filter (B-1, I-3) lives HERE,
-- in SQL, pgTAP-tested; the Go app layer is a thin reader and never reimplements it (ADR-P017).
-- Live joins only — no materialized projection (SPEC-009 tripwire unfired). SECURITY: these read
-- authoritative perception rows; the safety wall is the WHERE clause, not the caller.

-- FILTER 1 — the safety wall. holder ∈ {viewer} ∪ {world's universal common-knowledge holders},
-- AND the perception is still held (invalid_tick IS NULL AND expired_at IS NULL). 0A: common-knowledge
-- holders are the world's faction/group entities (ambient membership; the one seeded such holder is
-- 'Common Knowledge'). A per-actor group-membership table is a deferred STORAGE optimization
-- (SPEC-006 scale trigger), never a new knowledge path.
CREATE FUNCTION fn_visible_perceptions(p_world_id uuid, p_viewer_id uuid)
RETURNS SETOF perception_record
LANGUAGE sql STABLE AS $$
  SELECT pr.*
  FROM perception_record pr
  WHERE pr.world_id = p_world_id
    AND pr.invalid_tick IS NULL
    AND pr.expired_at  IS NULL
    AND ( pr.holder_id = p_viewer_id
          OR pr.holder_id IN (
            SELECT er.entity_id FROM entity_registry er
            WHERE er.world_id = p_world_id
              AND er.entity_kind IN ('faction','group')
          )
        );
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_visible_perceptions(uuid, uuid);
```

- [ ] **Step 4: Apply and re-run**

Run:
```bash
make migrate && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/42_visible_perceptions_test.sql
```
Expected: PASS (4/4).

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260614090002_projection_functions.sql core/db/tests/42_visible_perceptions_test.sql core/db/schema.sql
git commit -m "feat(db): fn_visible_perceptions safety wall (FILTER 1, I-3)"
```

### Task 4: `fn_perceived_name` — genuine common-knowledge name gate (TDD)

**Files:**
- Test: `core/db/tests/43_perceived_name_test.sql` (create)
- Modify: `core/db/migrations/20260614090002_projection_functions.sql` (add function + down drop)

- [ ] **Step 1: Write the failing test (renders for cast, withholds for noise)**

Create `core/db/tests/43_perceived_name_test.sql`:
```sql
BEGIN;
SELECT plan(4);
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
           'Mara', 'Mara name renders for Player (common knowledge)');
-- common knowledge ⇒ renders even for a viewer who knows nothing else about Mara
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'cccccccc-cccc-cccc-cccc-cccccccccccc','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
           'Mara', 'Mara name renders for Jonas (common knowledge, not viewer-specific)');
-- THE REAL GATE: a noise actor with no CK name perception is WITHHELD (NULL), not leaked
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','00000000-0000-0000-0000-000000000001'),
           NULL, 'O1 name withheld — gate is real, not a raw entity_registry read');
-- name comes from the perception layer, never entity_registry (Player resolves Player)
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
           'Player', 'Player name renders for self');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to confirm it fails**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/43_perceived_name_test.sql
```
Expected: FAIL — `function fn_perceived_name(...) does not exist`.

- [ ] **Step 3: Add `fn_perceived_name`** — insert into `20260614090002_projection_functions.sql` immediately **after** `fn_visible_perceptions` (before the `-- migrate:down` line), and add its drop to the down section:

```sql
-- Name resolution — a GENUINE knowability gate (not a raw entity_registry read; going-in 5).
-- Returns the perception-layer name content IFF the entity is knowable to the viewer:
--  priority 1: a viewer-held divergent perceived-name perception (DEFERRED — none in 0A; the
--              branch is intentionally absent, the seam is this function boundary);
--  priority 2: a common-knowledge name perception (CK-held, world_genesis-sourced, subject=entity)
--              that the viewer is permitted to see (routed through FILTER 1);
--  else NULL (WITHHELD). A noise actor with no CK name perception returns NULL.
CREATE FUNCTION fn_perceived_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS text
LANGUAGE sql STABLE AS $$
  SELECT vp.content
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id AND ps.entity_id = p_entity_id
  JOIN canon_event ce ON ce.event_id = vp.source_event_id
  WHERE ce.event_type = 'world_genesis'
  ORDER BY vp.acquired_tick
  LIMIT 1;
$$;
```
And add to `-- migrate:down` (before the existing `DROP FUNCTION ... fn_visible_perceptions` line so dependents drop first):
```sql
DROP FUNCTION IF EXISTS fn_perceived_name(uuid, uuid, uuid);
```

- [ ] **Step 4: Recreate the function (migrations are immutable once applied; re-run from clean)**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/43_perceived_name_test.sql
```
Expected: PASS (4/4). (`make reset` re-runs all migrations on a clean DB, picking up the edited file.)

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260614090002_projection_functions.sql core/db/tests/43_perceived_name_test.sql core/db/schema.sql
git commit -m "feat(db): fn_perceived_name common-knowledge name gate (real withhold path)"
```

### Task 5: `fn_actor_page` — FILTER 2 + assembly + the non-vacuous leak test (TDD)

**Files:**
- Test: `core/db/tests/45_actor_page_test.sql` (create)
- Modify: `core/db/migrations/20260614090002_projection_functions.sql` (add function + down drop)

- [ ] **Step 1: Write the failing assembly + paired leak test**

Create `core/db/tests/45_actor_page_test.sql` (uses literal UUIDs — pg_prove runs each `*_test.sql` standalone and does NOT load `helpers.sql`):
```sql
BEGIN;
SELECT plan(7);

-- (1) coherent view: Mara's perceived name resolves for Player
SELECT is(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->>'perceived_name',
  'Mara', 'perceived_name = Mara on Mara page');
-- (2) schema_version present
SELECT is(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->>'schema_version',
  'actor_page/1', 'payload carries schema_version actor_page/1');
-- (3) current_synthesis is null in 0A (no LLM; honest emptiness — Change 2).
--     ->> on a JSON null yields SQL NULL (json has no '=' operator, so test via ->> IS NULL).
SELECT ok(
  (fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->>'current_synthesis') IS NULL,
  'current_synthesis is null in 0A');
-- (4) NO relationship fields anywhere in the payload (B-3/B-4, AC#7/#8)
SELECT ok(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')::text NOT ILIKE '%relationship%',
  'no relationship field in payload (B-3)');

-- (5) NON-VACUOUS LEAK TEST — same row (about-Mara, dca7…a01), same page (Mara), both halves:
--     PRESENT for viewer=Player ...
SELECT ok(
  EXISTS (
    SELECT 1
    FROM json_array_elements(
           fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                         'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
         json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000a01'),
  'about-Mara private perception PRESENT for viewer=Player');
--     ... and ABSENT for viewer=Jonas. If (5a) ever went vacuous (empty page), (5b) alone could
--     pass on nothing — so BOTH are asserted on the SAME perception_id. This pair is the gate.
SELECT ok(
  NOT EXISTS (
    SELECT 1
    FROM json_array_elements(
           fn_actor_page('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc',
                         'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
         json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000a01'),
  'about-Mara private perception ABSENT for viewer=Jonas (I-3, fails loud on leak)');

-- (6) the page never surfaces a canon row — the source-event summary text never leaks (B-1)
SELECT ok(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')::text NOT LIKE '%P tells M%',
  'canon_event.summary never appears in the payload (B-1)');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to confirm it fails**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/45_actor_page_test.sql
```
Expected: FAIL — `function fn_actor_page(...) does not exist`.

- [ ] **Step 3: Add `fn_actor_page`** — insert into `20260614090002_projection_functions.sql` after `fn_perceived_name` (before `-- migrate:down`), and add its drop FIRST in the down section:

```sql
-- Assembly: FILTER 1 ∘ FILTER 2 + name + perception-bound fields. Returns the actor_page/1 payload.
-- FILTER 2 = perception_subject (primary about-ness). Identity/name substrate (world_genesis-sourced)
-- is EXCLUDED from collected knowledge so a name never masquerades as a knowledge item.
-- TRIPWIRE (design §3): the `event_type <> 'world_genesis'` exclusion is correct ONLY while genesis
-- sources names exclusively. If genesis ever sources a non-name perception, switch this to a real
-- name/identity discriminator instead of keying on the source event.
-- HARD RULE: never reads actor_state/location_state (authoritative canon) — last_known_status is
-- perception-bound or null (B-1/I-3).
CREATE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json
LANGUAGE sql STABLE AS $$
  WITH about_actor AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence,
           ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_actor_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id',   aa.perception_id,
             'content',         aa.content,
             'epistemic_type',  aa.epistemic_type,
             'occurred_at_tick',aa.valid_tick,
             'display_label',   aa.in_world_label,
             'confidence',      aa.confidence,
             'decay',           json_build_object('stale', false, 'last_confirmed_label', aa.in_world_label),
             'source',          json_build_object('epistemic_type', aa.epistemic_type,
                                                  'source_event_label', aa.in_world_label)
           ) AS item,
           aa.valid_tick AS sort_tick
    FROM about_actor aa
  ),
  groups AS (
    SELECT CASE WHEN count(*) = 0 THEN '[]'::json
                ELSE json_build_array(json_build_object(
                       'group_key',   p_actor_id::text,
                       'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
                       'items',       coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                     ))
           END AS arr
    FROM items i
  )
  SELECT json_build_object(
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
      'collected_knowledge_groups', (SELECT arr FROM groups),
      'inline_links',               '[]'::json
    )
  );
$$;
```
Add to `-- migrate:down`, as the FIRST drop (dependents before dependencies):
```sql
DROP FUNCTION IF EXISTS fn_actor_page(uuid, uuid, uuid);
```

- [ ] **Step 4: Recreate from clean and re-run**

Run:
```bash
make reset && docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests/45_actor_page_test.sql
```
Expected: PASS (7/7). The two leak halves (5) must both pass.

- [ ] **Step 5: Full suite green (nothing regressed)**

Run:
```bash
make test && make replay
```
Expected: all pgTAP green; replay returns `t`.

- [ ] **Step 6: Commit**

```bash
git add core/db/migrations/20260614090002_projection_functions.sql core/db/tests/45_actor_page_test.sql core/db/schema.sql
git commit -m "feat(db): fn_actor_page assembly (FILTER 2) + non-vacuous I-3 leak test"
```

---

## Phase 4 — Go reader (first Go surface, ADR-P017)

### Task 6: JSON-Schema source of truth for `actor_page/1`

**Files:**
- Create: `core/api/schema/actor_page.v1.schema.json`

- [ ] **Step 1: Write the schema (single source of truth for FE/BE types)**

Create `core/api/schema/actor_page.v1.schema.json`:
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "actor_page/1",
  "title": "ActorPage",
  "type": "object",
  "required": ["schema_version", "world_id", "viewer_id", "actor"],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "const": "actor_page/1" },
    "world_id":  { "type": "string", "format": "uuid" },
    "viewer_id": { "type": "string", "format": "uuid" },
    "actor": {
      "type": "object",
      "required": ["id","perceived_name","perceived_role","current_synthesis","last_known_status","known_artifacts","collected_knowledge_groups","inline_links"],
      "additionalProperties": false,
      "properties": {
        "id": { "type": "string", "format": "uuid" },
        "perceived_name":   { "type": ["string","null"] },
        "perceived_role":   { "type": ["string","null"] },
        "current_synthesis":{ "type": ["string","null"] },
        "last_known_status":{ "type": ["string","null"] },
        "known_artifacts":  { "type": "array", "items": { "type": "object" } },
        "inline_links":     { "type": "array", "items": { "type": "object" } },
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
                    "confidence":      { "type": "number" },
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

- [ ] **Step 2: Commit**

```bash
git add core/api/schema/actor_page.v1.schema.json
git commit -m "feat(api): actor_page/1 JSON Schema (FE/BE type source of truth)"
```

### Task 7: Go module + viewer resolution (TDD)

**Files:**
- Create: `core/api/go.mod`, `core/api/viewer.go`, `core/api/viewer_test.go`

- [ ] **Step 1: Initialize the Go module and add pgx**

Run:
```bash
cd core/api && go mod init dreamchat/core/api && go get github.com/jackc/pgx/v5@v5.6.0 && cd ../..
```
Expected: `core/api/go.mod` + `go.sum` created.

- [ ] **Step 2: Write the failing viewer-resolution test**

Create `core/api/viewer_test.go`:
```go
package main

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const worldID = "11111111-1111-1111-1111-111111111111"
const playerID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
const jonasID = "cccccccc-cccc-cccc-cccc-cccccccccccc"

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return pool
}

func TestResolveViewer_DefaultIsPlayer(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	got, err := ResolveViewer(context.Background(), pool, worldID, "", false)
	if err != nil {
		t.Fatalf("ResolveViewer: %v", err)
	}
	if got != playerID {
		t.Fatalf("default viewer = %s, want player %s", got, playerID)
	}
}

func TestResolveViewer_DebugOverrideHonoredOnlyInDebug(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	// override ignored when debug=false
	got, _ := ResolveViewer(context.Background(), pool, worldID, jonasID, false)
	if got != playerID {
		t.Fatalf("override leaked outside debug mode: got %s", got)
	}
	// override honored when debug=true
	got, _ = ResolveViewer(context.Background(), pool, worldID, jonasID, true)
	if got != jonasID {
		t.Fatalf("debug override not honored: got %s", got)
	}
}
```

- [ ] **Step 3: Run it to confirm it fails**

Run:
```bash
make reset
cd core/api && go test ./... -run TestResolveViewer ; cd ../..
```
Expected: FAIL — `undefined: ResolveViewer`.

- [ ] **Step 4: Write `viewer.go`**

Create `core/api/viewer.go`:
```go
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ResolveViewer returns the perception viewer for a request. The viewer is the epistemic
// boundary and is therefore resolved SERVER-SIDE (D-7/B-1): the client never picks whose
// truth is rendered in play mode. A debug override is honored ONLY when debug==true; it
// swaps the resolved identity and is still run through the identical safety filter downstream
// (fn_actor_page) — it never bypasses the wall.
//
// 0A stub: the player-controlled actor is resolved as the world's actor named 'Player'.
// Auth/session is out of scope this chunk (Bridge §6 item 4); this is the documented stand-in.
func ResolveViewer(ctx context.Context, pool *pgxpool.Pool, worldID, debugOverride string, debug bool) (string, error) {
	if debug && debugOverride != "" {
		return debugOverride, nil
	}
	var id string
	err := pool.QueryRow(ctx,
		`SELECT entity_id::text FROM entity_registry
		 WHERE world_id = $1 AND entity_kind = 'actor' AND canonical_name = 'Player' LIMIT 1`,
		worldID).Scan(&id)
	return id, err
}
```

- [ ] **Step 5: Run to verify pass**

Run:
```bash
cd core/api && go test ./... -run TestResolveViewer -v ; cd ../..
```
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
git add core/api/go.mod core/api/go.sum core/api/viewer.go core/api/viewer_test.go
git commit -m "feat(api): Go module + server-side viewer resolution (debug override gated)"
```

### Task 8: Actor-page HTTP handler — thin reader (TDD)

**Files:**
- Create: `core/api/actorpage.go`, `core/api/actorpage_test.go`, `core/api/main.go`

- [ ] **Step 1: Write the failing handler test (present-for-Player / absent-for-Jonas over HTTP)**

Create `core/api/actorpage_test.go`:
```go
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const maraID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
const aboutMaraPID = "dca70000-0000-0000-0000-000000000a01"

func TestActorPage_DefaultViewerSeesAboutMara(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewActorPageHandler(pool, false) // debug=false: viewer resolves to Player
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/actors/"+maraID+"/page", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	var probe map[string]any
	if err := json.Unmarshal([]byte(body), &probe); err != nil {
		t.Fatalf("payload not valid JSON: %v — %s", err, body)
	}
	if probe["schema_version"] != "actor_page/1" {
		t.Fatalf("missing/wrong schema_version: %s", body)
	}
	if !strings.Contains(body, aboutMaraPID) {
		t.Fatalf("about-Mara perception ABSENT for Player (should be present): %s", body)
	}
}

func TestActorPage_DebugViewerJonas_NoLeak(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewActorPageHandler(pool, true) // debug=true: honor ?viewer=
	req := httptest.NewRequest(http.MethodGet,
		"/worlds/"+worldID+"/compendium/actors/"+maraID+"/page?viewer="+jonasID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	// GATE-CRITICAL: the private about-Mara perception must NOT appear for Jonas, and the
	// canon summary must never appear for anyone.
	if strings.Contains(body, aboutMaraPID) {
		t.Fatalf("LEAK: about-Mara perception present for Jonas: %s", body)
	}
	if strings.Contains(body, "P tells M") {
		t.Fatalf("LEAK: canon summary in payload: %s", body)
	}
	// sanity: it IS a well-formed payload (not an empty error body the test could pass on vacuously)
	var probe map[string]any
	if err := json.Unmarshal([]byte(body), &probe); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if probe["schema_version"] != "actor_page/1" {
		t.Fatalf("not a real actor page payload: %s", body)
	}
	_ = context.Background()
}
```

- [ ] **Step 2: Run it to confirm it fails**

Run:
```bash
make reset
cd core/api && go test ./... -run TestActorPage ; cd ../..
```
Expected: FAIL — `undefined: NewActorPageHandler`.

- [ ] **Step 3: Write the handler (thin reader — calls fn_actor_page, returns its JSON verbatim)**

Create `core/api/actorpage.go`:
```go
package main

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// route: GET /worlds/{w}/compendium/actors/{id}/page
var routeRE = regexp.MustCompile(
	`^/worlds/([0-9a-fA-F-]{36})/compendium/actors/([0-9a-fA-F-]{36})/page$`)

type actorPageHandler struct {
	pool  *pgxpool.Pool
	debug bool
}

// NewActorPageHandler returns the read-only Actor-page handler. debug enables the creator/debug
// `?viewer=` override (C-4). The handler is a THIN READER: the entire perception/safety filter
// lives in fn_actor_page (SQL), never here (ADR-P017 binding).
func NewActorPageHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &actorPageHandler{pool: pool, debug: debug}
}

func (h *actorPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := routeRE.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID, actorID := m[1], m[2]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.debug)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var payload []byte
	err = h.pool.QueryRow(ctx,
		`SELECT fn_actor_page($1, $2, $3)::text`, worldID, viewerID, actorID).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
```

- [ ] **Step 4: Write `main.go` (server entrypoint)**

Create `core/api/main.go`:
```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
	mux := http.NewServeMux()
	mux.Handle("/worlds/", NewActorPageHandler(pool, debug))

	addr := ":8080"
	log.Printf("dreamchat world backend (read-only projection API) on %s (debug=%v)", addr, debug)
	log.Fatal(http.ListenAndServe(addr, mux))
}
```

- [ ] **Step 5: Run to verify pass**

Run:
```bash
cd core/api && go test ./... -v ; cd ../..
```
Expected: PASS (viewer + actorpage tests).

- [ ] **Step 6: Verify build + vet (Go's strict-compiler discipline, ADR-P017 rationale)**

Run:
```bash
cd core/api && go vet ./... && go build ./... ; cd ../..
```
Expected: no output, exit 0.

- [ ] **Step 7: Commit**

```bash
git add core/api/actorpage.go core/api/actorpage_test.go core/api/main.go core/api/go.sum
git commit -m "feat(api): read-only Actor-page endpoint (thin reader over fn_actor_page)"
```

---

## Phase 5 — Frontend minimal shell (presentation only, D-7)

### Task 9: Scaffold FE + generated types from the JSON Schema

**Files:**
- Create: `frontend/package.json`, `frontend/index.html`, `frontend/vite.config.ts`, `frontend/tsconfig.json`, `frontend/src/main.tsx`

- [ ] **Step 1: Create `frontend/package.json`**

```json
{
  "name": "dreamchat-world-frontend",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "gen:types": "json2ts -i ../core/api/schema/actor_page.v1.schema.json -o src/types/actor_page.ts"
  },
  "dependencies": { "react": "^18.3.1", "react-dom": "^18.3.1" },
  "devDependencies": {
    "@types/react": "^18.3.3", "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.1", "typescript": "^5.5.4", "vite": "^5.4.0",
    "json-schema-to-typescript": "^15.0.0"
  }
}
```

- [ ] **Step 2: Create `frontend/vite.config.ts` (proxy API to the Go server)**

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: { proxy: { "/worlds": "http://localhost:8080" } },
});
```

- [ ] **Step 3: Create `frontend/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2020", "useDefineForClassFields": true, "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext", "skipLibCheck": true, "moduleResolution": "bundler",
    "jsx": "react-jsx", "strict": true, "noUnusedLocals": true, "noUnusedParameters": true
  },
  "include": ["src"]
}
```

- [ ] **Step 4: Create `frontend/index.html`**

```html
<!doctype html>
<html lang="en">
  <head><meta charset="UTF-8" /><title>DreamChat — Compendium</title></head>
  <body><div id="root"></div><script type="module" src="/src/main.tsx"></script></body>
</html>
```

- [ ] **Step 5: Install + generate types from the schema source of truth**

Run:
```bash
cd frontend && npm install && npm run gen:types && cd ..
```
Expected: `frontend/src/types/actor_page.ts` generated (the `ActorPage` interface), derived from `core/api/schema/actor_page.v1.schema.json`.

- [ ] **Step 6: Create `frontend/src/main.tsx`** (mounts the page for the seeded Mara, viewed as Player by default)

```tsx
import React from "react";
import { createRoot } from "react-dom/client";
import { ActorPage } from "./ActorPage";

const WORLD = "11111111-1111-1111-1111-111111111111";
const MARA = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb";

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ActorPage world={WORLD} actorId={MARA} />
  </React.StrictMode>
);
```

- [ ] **Step 7: Commit**

```bash
git add frontend/package.json frontend/vite.config.ts frontend/tsconfig.json frontend/index.html frontend/src/main.tsx frontend/src/types/actor_page.ts frontend/package-lock.json
git commit -m "feat(fe): scaffold minimal shell + generate actor_page types from schema"
```

### Task 10: The Actor page component (presentation only, Glossary vocabulary)

**Files:**
- Create: `frontend/src/api.ts`, `frontend/src/ActorPage.tsx`

- [ ] **Step 1: Create `frontend/src/api.ts`** (typed fetch against the generated type)

```ts
import type { ActorPage } from "./types/actor_page";

export async function fetchActorPage(world: string, actorId: string): Promise<ActorPage> {
  const res = await fetch(`/worlds/${world}/compendium/actors/${actorId}/page`);
  if (!res.ok) throw new Error(`actor page ${res.status}`);
  return (await res.json()) as ActorPage;
}
```

- [ ] **Step 2: Create `frontend/src/ActorPage.tsx`**

Presentation only (D-7): renders the perception-bound payload verbatim. Glossary vocabulary (F-1/F-2): "Collected Knowledge", "Actor"; no "entity/perception/confirmed/false" labels; no relationship UI (B-3). Decay/source language (AC#2/#4). Honest emptiness when `current_synthesis`/items are null.

```tsx
import { useEffect, useState } from "react";
import type { ActorPage as ActorPageT } from "./types/actor_page";
import { fetchActorPage } from "./api";

export function ActorPage({ world, actorId }: { world: string; actorId: string }) {
  const [page, setPage] = useState<ActorPageT | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    fetchActorPage(world, actorId).then(setPage).catch((e) => setErr(String(e)));
  }, [world, actorId]);

  if (err) return <main>Could not load this page.</main>;
  if (!page) return <main>Loading…</main>;

  const a = page.actor;
  const groups = a.collected_knowledge_groups ?? [];
  const hasKnowledge = groups.some((g) => (g.items ?? []).length > 0);

  return (
    <main style={{ maxWidth: 680, margin: "2rem auto", fontFamily: "system-ui" }}>
      <h1>{a.perceived_name ?? "Unknown"}</h1>
      {a.perceived_role && <p style={{ color: "#666" }}>{a.perceived_role}</p>}

      <section>
        <h2>Synthesis</h2>
        <p>{a.current_synthesis ?? <em>Nothing synthesized yet.</em>}</p>
      </section>

      {a.last_known_status && (
        <section><h2>Last known</h2><p>{a.last_known_status}</p></section>
      )}

      <section>
        <h2>Collected Knowledge</h2>
        {!hasKnowledge && <p><em>You know nothing about them yet.</em></p>}
        {groups.map((g) => (
          <div key={g.group_key}>
            {g.group_label && <h3>{g.group_label}</h3>}
            <ul>
              {(g.items ?? []).map((it) => (
                <li key={it.perception_id} style={{ marginBottom: "0.5rem" }}>
                  <div>{it.content}</div>
                  <small style={{ color: "#888" }}>
                    {it.epistemic_type}
                    {it.display_label ? ` · ${it.display_label}` : ""}
                    {it.decay && (it.decay as { stale?: boolean }).stale ? " · last known…" : ""}
                  </small>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </section>
    </main>
  );
}
```

- [ ] **Step 3: Verify the FE type-checks and builds**

Run:
```bash
cd frontend && npm run build ; cd ..
```
Expected: `tsc` passes (strict), Vite build succeeds.

- [ ] **Step 4: Manual smoke (by-eye, the human gate)**

Run (two terminals):
```bash
# terminal 1 — backend (DB must be seeded: make reset)
cd core/api && DREAMCHAT_MODE=debug go run . ; 
# terminal 2 — frontend
cd frontend && npm run dev
```
Open the printed Vite URL. Expected: Mara's page renders perceived name "Mara", honest empty synthesis, and the about-Mara knowledge item with `direct · Day 1`. **DevTools → Network**: the `/page` payload contains NO `relationship`, NO canon summary text, and (as Player) DOES contain the about-Mara perception_id. Append `?viewer=cccccccc-cccc-cccc-cccc-cccccccccccc` mentally via the debug path (or temporarily point `main.tsx` viewer) → the about-Mara item disappears from the payload.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api.ts frontend/src/ActorPage.tsx
git commit -m "feat(fe): perception-bound Actor page (presentation only, Glossary vocabulary)"
```

---

## Phase 6 — Docs: ADR acceptances, DDL, SPEC ledger, playbook reconciliation

### Task 11: Promote ADRs, update frozen DDL doc, record SPEC entries

**Files:**
- Modify: `docs/30_architecture/adr/ADR-P017_backend_application_language_go.md`
- Modify: `docs/30_architecture/canon_engine/02_world_state_adrs.md` (ADR-035 status)
- Modify: `docs/30_architecture/canon_engine/03_world_state_technical_reference.md` (§1.3 gains `perception_subject` DDL)
- Modify: `docs/open-spec-items.md` (SPEC-009 wording fix + new deferrals; SPEC-008 marked implemented)
- Modify: `docs/30_architecture/implementation_playbook_superpowers.md` (lines 28 & 66 → Mara)

- [ ] **Step 1: ADR-P017 Proposed → Accepted**

In `ADR-P017_...go.md`, change `**Status:** Proposed` to:
```
**Status:** Accepted (chunk-3 gate, 2026-06-14)
```

- [ ] **Step 2: ADR-035 Proposed → Accepted + record the world_id delta**

In `02_world_state_adrs.md`, ADR-035 heading: change `**Status:** Proposed (...)` to `**Status:** Accepted (chunk-3 gate, 2026-06-14)`. Append to its **Consequences** paragraph:
```
Accepted under chunk-3: the migration adds `perception_subject(perception_id, entity_id, world_id)` — a third column `world_id` carried from birth (delta vs this ADR's two-column sketch) matching the SPEC-009 tenant-key posture; justified because a new table carries the tenant key at zero migration cost, whereas the three bare junction/edge tables stay bare pending the SPEC-009 sharding trigger (see open-spec-items). doc 03 §1.3 now carries the DDL.
```

- [ ] **Step 3: Add `perception_subject` DDL to doc 03 §1.3**

In `03_world_state_technical_reference.md`, in §1.3 (after the `perception_record` block + invalidation note), insert:
```sql
-- about-ness (ADR-035, accepted chunk-3): the entities a perception is *about*, populated at
-- write time. Derivation source_event_id → event_participant is retained as fallback/validation.
CREATE TABLE perception_subject (
  perception_id UUID NOT NULL REFERENCES perception_record(perception_id),
  entity_id     UUID NOT NULL,
  world_id      UUID NOT NULL,          -- tenant key from birth (SPEC-009)
  PRIMARY KEY (perception_id, entity_id)
);
CREATE INDEX idx_ps_entity ON perception_subject (entity_id);
```

- [ ] **Step 4: Fix the SPEC-009 false sentence + record the §7 deferral**

In `docs/open-spec-items.md`, under SPEC-009's Citus bullet, replace the parenthetical
`(every core table already carries it — see SPEC-009 verification / Task 5b)` with:
```
(core tables carry world_id; EXCEPTION: the junction/edge tables event_participant,
provenance_edge, causal_bundle_input do NOT — see the deferral below).
```
Then append a new subsection:
```
### SPEC-009b — world_id absent on three junction/edge tables (deferred, chunk-3 audit)
event_participant, provenance_edge, causal_bundle_input carry no world_id. Chunk-3 (read-only
Actor page) does not need it: event_participant is reached only through its world-scoped parent
canon_event; the other two are not read in chunk-3. The new perception_subject carries world_id
from birth. **Firing trigger:** when SPEC-009 row-based sharding is implemented, these three must
either gain world_id as the distribution key OR be co-located by their world-scoped parent —
decided then. Until then, unchanged. No frozen-DDL change in chunk-3.
```

- [ ] **Step 5: Mark SPEC-008 implemented**

In SPEC-008's `Expected outcome`, append:
```
DONE (chunk-3, 2026-06-14): perception_subject shipped (migration 20260614090001), write-time
population in the seed, pgTAP positive/negative + derivation usage; ADR-035 Accepted.
```

- [ ] **Step 6: Reconcile the playbook to Mara**

In `implementation_playbook_superpowers.md`: line ~28 (Q "open Seren's page") and line ~66 ("Mara/Seren seed", "Open Seren's page in a browser") — change "Seren" → "Mara" in the chunk-3 rows, and append:
```
(Seren does not exist in 0A/0B — ADR-029; she is the Phase-4 golden. Chunk-3 gate runs on Mara:
viewer=Player sees a coherent Mara page; viewer=Jonas's payload omits Mara's private belief.
"Seren's page" is satisfied later at S4.)
```

- [ ] **Step 7: Commit**

```bash
git add docs/30_architecture/adr/ADR-P017_backend_application_language_go.md \
        docs/30_architecture/canon_engine/02_world_state_adrs.md \
        docs/30_architecture/canon_engine/03_world_state_technical_reference.md \
        docs/open-spec-items.md \
        docs/30_architecture/implementation_playbook_superpowers.md
git commit -m "docs: accept ADR-P017 & ADR-035; doc03 perception_subject DDL; SPEC-009 fix; playbook→Mara"
```

---

## Phase 7 — Gate

### Task 12: Full green + by-eye gate + tag

**Files:** none (verification + tag)

- [ ] **Step 1: Full pgTAP suite + replay from clean**

Run:
```bash
make reset && make test && make replay
```
Expected: every pgTAP test green (existing 0A/0B + new `12`,`14`,`42`,`43`,`45`); replay returns `t`.

- [ ] **Step 2: Go tests + vet + build**

Run:
```bash
make reset && cd core/api && go vet ./... && go build ./... && go test ./... -v ; cd ../..
```
Expected: all Go tests pass (including the two leak/no-leak HTTP tests).

- [ ] **Step 3: FE build**

Run:
```bash
cd frontend && npm run build ; cd ..
```
Expected: type-check + build succeed.

- [ ] **Step 4: Schema committed-clean**

Run:
```bash
make schema-check
```
Expected: no diff in `core/db/schema.sql`.

- [ ] **Step 5: By-eye founder gate (the Q2 product check)**

Run the backend (`DREAMCHAT_MODE=debug go run .` in `core/api`, DB seeded) + FE (`npm run dev`). Confirm, in the browser + DevTools Network:
- Mara's page renders a coherent perception-bound view (perceived name, sourced knowledge, tick label "Day 1").
- No relationship field/panel anywhere (B-3).
- As **Player**: payload contains the about-Mara perception (`dca70000-…a01`).
- Switch the debug viewer to **Jonas** (`?viewer=cccccccc-…cccc`): the about-Mara perception and Mara's secret are **absent** from the payload (I-3), and no canon `summary` text appears for anyone.

- [ ] **Step 6: Open the PR**

```bash
git push -u origin chunk-3-projection-api
gh pr create --title "Chunk 3: read-only projection API + perception-bound Actor page" \
  --body "Implements docs/superpowers/specs/2026-06-14-chunk-3-projection-api-actor-page-design.md. Gate: Mara page coherent for Player; I-3 leak test (about-Mara present for Player, absent for Jonas) green at SQL + HTTP layers. ADR-P017 & ADR-035 accepted. Cites: B-1, B-3, D-7, I-3, ADR-029, ADR-035, ADR-P017."
```

- [ ] **Step 7: After merge to main — tag the gate on the verified merge commit**

```bash
git checkout main && git pull
git tag chunk-3-actor-page-gate
git push origin chunk-3-actor-page-gate
```

---

## Out of scope (do not implement — PRD non-goals + register ARE the out-of-scope list)

beat loop / any LLM / event creation / any write path · images & portraits · relationship UI (B-3) · Timeline/Location/Artifact pages · actors-**list** endpoint (next S0 increment) · materialized projections / snapshots / sharding (SPEC-009 unfired) · true `perceived_name` divergence (deferred, SPEC ledger) · per-actor group-membership table (SPEC-006 trigger) · world_id on event_participant/provenance_edge/causal_bundle_input (SPEC-009b trigger) · auth/session (Bridge §6 item 4; 0A viewer is the documented 'Player' stub).
