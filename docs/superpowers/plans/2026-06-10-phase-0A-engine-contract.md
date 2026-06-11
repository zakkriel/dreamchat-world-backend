# Phase 0A Engine Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deploy the deterministic world-state spine (Phase 0A) so that hand-seeded `accepted` events produce correct projections + perceptions, and projections rebuild from the event log to domain-equivalent state — proven by I-1/I-2/I-7 green on the Mara scenario.

**Architecture:** Pure PostgreSQL 16 + pgTAP. Schema/triggers/seed are plain SQL applied by `dbmate`; every invariant is a pgTAP assertion run by `pg_prove`. One shared `apply_mutation()` function is the *only* projection write path — both the live `AFTER INSERT` trigger and the replay procedure call it, so I-1 exercises production code, never a reimplementation. No application language, no LLM, no API, no frontend.

**Tech Stack:** PostgreSQL 16 + pgTAP + pg_prove (one pinned Docker image, shared by local / CI / Railway), dbmate (plain-SQL migrations + `schema.sql` dump), Docker Compose, GNU Make, GitHub Actions. **Docker is the only sanctioned environment** — local dev, CI, and the deployed image family all run the same pinned `postgres:16 + pgTAP` image, so the DoD determinism check is validated against what actually ships.

**Governing docs (frozen, read-only):** spec `docs/superpowers/specs/2026-06-10-phase-0A-engine-contract-design.md`; engine `canon_engine/13`, `03 §1/§3/§5`, `07 §6`; ADR-004/006/008/026/029/030; register B-5/D-4/D-5/I-1/I-2/I-7.

**Frozen-contract rule:** if a task reveals a genuine engine-spec problem, STOP and draft a *proposed new ADR* in `canon_engine/02` (number assigned at proposal time) — never a code workaround.

---

## File Structure (decomposition locked here)

```
core/db/
  Dockerfile                     -- postgres:16 (pinned family) + pgTAP extension + pg_prove
  migrations/                    -- dbmate plain-SQL migrations (timestamped)
    20260610090001_extensions_roles.sql
    20260610090002_canon_spine.sql
    20260610090003_registry.sql
    20260610090004_deltas_epistemic_causal.sql
    20260610090005_projections.sql
    20260610090006_apply_mutation_and_triggers.sql
  seeds/
    seed_mara_0A.sql             -- deterministic Mara scenario (one re-runnable script)
  tests/
    00_smoke_test.sql            -- pgTAP wiring smoke
    10_schema_test.sql           -- tables/columns/roles exist; canon_event columns_are (doc 03 §1.1)
    20_append_only_test.sql      -- append-only UPDATE + canon_event/event_participant DELETE guard
    25_delete_guard_test.sql     -- DELETE guard on state_mutation/perception_record/provenance_edge
    30_apply_mutation_test.sql   -- projection write path + idempotency + relationship no-op
    40_perception_test.sql       -- doc 13 §7 epistemic checks (Mara/Jonas/public)
    50_provenance_test.sql       -- I-2 universal provenance
    60_permissions_test.sql      -- I-7 maintainer-only projection writes + function EXECUTE hardening
    70_determinism_guards_test.sql -- uniqueness guards (Rider C) + zero-relationship (SPEC-001)
    80_golden_projection_test.sql  -- hand-computed expected rows (spot-check)
    90_replay_test.sql           -- I-1 replay invariance + negative control
    checks_0A.sql                -- doc 13 §7 boolean suite, verbatim, runnable by hand
    replay_0A.sql                -- by-hand I-1 runner (calls replay_0A())
    expected_projections_0A.csv  -- hand-computed golden rows
    helpers.sql                  -- fixed-UUID psql \set vars for by-hand checks
  schema.sql                     -- dbmate-generated dump (audit artifact, committed)
docs/
  open-spec-items.md             -- SPEC-001, SPEC-002
docker-compose.yml               -- db service (+ dbmate tools profile)
.env                             -- DATABASE_URL (local values)
Makefile                         -- doctor / db-up / migrate / seed / test / replay / reset / schema-check
.github/workflows/invariants.yml -- CI: migrate → seed → pg_prove → schema-check
.gitignore
```

**Note:** `apply_mutation()`, `sm_project()`, and `replay_0A()` are created by migration `0006` (production code shared with the trigger path). `tests/replay_0A.sql` is a thin by-hand runner.

---

## Conventions used by every task

- **Fixed UUIDs** (deterministic; defined once in `core/db/tests/helpers.sql`):
  - World `W` = `11111111-1111-1111-1111-111111111111`
  - Player `P` = `aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`; Mara `M` = `bbbbbbbb-…-bbbb`; Jonas `J` = `cccccccc-…-cccc`
  - Tavern = `dddddddd-…-dddd`; Common-Knowledge holder `PUB` = `eeeeeeee-…-eeee`
  - Noise NPCs `O1..O5` = `00000000-0000-0000-0000-000000000001` … `…0005`; `Square` = `000000a0-0000-0000-0000-0000000000a1`
  - `E1` = `e0000000-0000-0000-0000-000000000001`; `E102` = `e0000000-0000-0000-0000-000000000102`
  - Noise events (Task 11) = `e0000000-0000-0000-0000-9` + zero-padded index (collision-free vs E1/E102)
  - Test-only entities (rolled-back) use the `…0000f1`/`f2`/`f3`/`de`/`0a` suffixes — never seeded
- **TDD loop:** failing pgTAP test → run → watch FAIL for the stated reason → add SQL → watch PASS → commit.
- **Absolute state sets (Rider B):** every seed `state_mutation.new_value` is the *resulting* absolute JSONB value at a single-key path under `attrs.` (e.g. `attrs.location_id`). No deltas, no nested-path creation in 0A.
- **Run command by phase:**
  - **Tasks 3–8** (no seed-dependent tests): `make migrate && make test`.
  - **Tasks 9–17** (seed-dependent): `make reset && make test` — `reset` wipes to a clean DB, applies
    migrations, and seeds once; `test` runs `pg_prove` against it. Never run `seed` twice without a
    `reset` (the clean-DB guard, Task 9, will RAISE).
- **Commit prefix:** `feat(0A):` schema/seed, `test(0A):` test-only, `chore(0A):` tooling.

---

## Task 1: Verify Docker (sanctioned), then Postgres 16 + pgTAP + dbmate up

**Docker is the only sanctioned environment.** The operator installs a Docker runtime manually:
`brew install colima docker docker-compose && colima start` (or Docker Desktop). The plan never
provisions host-native Postgres as a sanctioned path (see Appendix A — emergency-only). This task's first
step only *verifies* Docker and fails with the install instruction if absent.

**Files:** Create `core/db/Dockerfile`, `docker-compose.yml`, `.env`, `.gitignore`, `Makefile`, `core/db/tests/00_smoke_test.sql`

- [ ] **Step 1: Preflight — verify Docker (fail fast with the fix)**

After writing the Makefile (Step 4), run `make doctor`. `doctor` body:
```makefile
doctor:          ## verify the sanctioned Docker runtime is available
	@command -v docker >/dev/null 2>&1 || { \
	  echo "ERROR: Docker is required (the only sanctioned environment)."; \
	  echo "Install a runtime manually, then re-run:"; \
	  echo "  brew install colima docker docker-compose && colima start   (or Docker Desktop)"; \
	  exit 1; }
	@docker compose version >/dev/null 2>&1 || { echo "ERROR: 'docker compose' v2 not available"; exit 1; }
	@echo "docker OK: $$(docker --version)"
```
Expected with Docker present: `docker OK: …`. Absent: non-zero exit with the `brew install colima …`
instruction. **Resolve Docker before continuing.**

- [ ] **Step 2: pgTAP smoke test**

`core/db/tests/00_smoke_test.sql`:
```sql
BEGIN;
SELECT plan(1);
SELECT ok( true, 'pgTAP harness is wired and pg_prove can run a test' );
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 3: Pinned Postgres image with pgTAP + pg_prove**

`core/db/Dockerfile`:
```dockerfile
# Pinned postgres:16 family — the single image shared by local dev, CI, and the deployed (Railway) family.
# The DoD determinism check is only authoritative against THIS environment.
FROM postgres:16
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      postgresql-16-pgtap \
      libtap-parser-sourcehandler-pgtap-perl \
 && rm -rf /var/lib/apt/lists/*
# postgresql-16-pgtap = pgTAP extension SQL; libtap-parser-sourcehandler-pgtap-perl = /usr/bin/pg_prove.
```

- [ ] **Step 4: Compose + full Makefile**

`docker-compose.yml`:
```yaml
services:
  db:
    build: ./core/db
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: dreamchat
    ports: ["5432:5432"]
    volumes:
      - ./core/db:/work
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d dreamchat"]
      interval: 2s
      timeout: 3s
      retries: 30

  dbmate:
    image: ghcr.io/amacneil/dbmate:2.21.0
    environment:
      DATABASE_URL: postgres://postgres:postgres@db:5432/dreamchat?sslmode=disable
    volumes:
      - ./core/db:/db
    working_dir: /
    depends_on:
      db:
        condition: service_healthy
    profiles: ["tools"]
```

`.env`:
```dotenv
DATABASE_URL=postgres://postgres:postgres@db:5432/dreamchat?sslmode=disable
```

`.gitignore`:
```gitignore
.env.local
*.log
```

`Makefile`:
```makefile
.PHONY: doctor db-up db-down migrate seed test replay reset schema-check pgtap

doctor:          ## verify the sanctioned Docker runtime is available
	@command -v docker >/dev/null 2>&1 || { \
	  echo "ERROR: Docker is required (the only sanctioned environment)."; \
	  echo "Install a runtime manually, then re-run:"; \
	  echo "  brew install colima docker docker-compose && colima start   (or Docker Desktop)"; \
	  exit 1; }
	@docker compose version >/dev/null 2>&1 || { echo "ERROR: 'docker compose' v2 not available"; exit 1; }
	@echo "docker OK: $$(docker --version)"

db-up: doctor    ## start Postgres, wait for healthy
	docker compose up -d --build db
	docker compose exec -T db bash -c 'until pg_isready -U postgres -d dreamchat; do sleep 1; done'

db-down:
	docker compose down -v

pgtap:           ## install pgTAP extension into the db
	docker compose exec -T db psql -U postgres -d dreamchat -c 'CREATE EXTENSION IF NOT EXISTS pgtap;'

migrate:         ## apply dbmate migrations + dump schema.sql
	docker compose run --rm dbmate up

seed:            ## load the deterministic Mara seed
	docker compose exec -T db psql -U postgres -d dreamchat -v ON_ERROR_STOP=1 -f /work/seeds/seed_mara_0A.sql

test: pgtap      ## run the pgTAP suite (run `make reset` first for seed-dependent tests)
	docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests

replay:          ## run I-1 replay by hand (boolean)
	docker compose exec -T db psql -U postgres -d dreamchat -c 'SELECT replay_0A();'

reset: db-down db-up migrate seed ## clean DB from scratch (determinism check helper)

schema-check:    ## fail if dbmate schema.sql has uncommitted drift
	docker compose run --rm dbmate dump
	git diff --exit-code core/db/schema.sql
```

- [ ] **Step 5: Up + smoke (expect PASS)**
```bash
make db-up && make pgtap
docker compose exec -T db pg_prove -U postgres -d dreamchat --ext .sql /work/tests
```
Expected: `00_smoke_test.sql .. ok`, `Result: PASS`.

- [ ] **Step 6: Commit**
```bash
git add core/db/Dockerfile docker-compose.yml .env .gitignore Makefile core/db/tests/00_smoke_test.sql
git commit -m "chore(0A): scaffold pinned Docker Postgres 16 + pgTAP + dbmate toolchain"
```

---

## Task 2: Open-spec items ledger

**Files:** Create `docs/open-spec-items.md`

- [ ] **Step 1: Write the ledger**

`docs/open-spec-items.md`:
```markdown
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
- **Status:** non-blocking for 0A.
```

- [ ] **Step 2: Commit**
```bash
git add docs/open-spec-items.md
git commit -m "docs(0A): open-spec ledger (SPEC-001 relationship addressing, SPEC-002 ordering key)"
```

---

## Task 3: Migration — extensions + roles

**Files:** Create `core/db/migrations/20260610090001_extensions_roles.sql`, `core/db/tests/10_schema_test.sql`

- [ ] **Step 1: Failing roles test**

`core/db/tests/10_schema_test.sql`:
```sql
BEGIN;
SELECT plan(2);
SELECT has_role('maintainer', 'maintainer role exists');
SELECT has_role('app_reader', 'app_reader role exists');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run — expect FAIL** (`maintainer role exists` not ok).

- [ ] **Step 3: Migration**

`core/db/migrations/20260610090001_extensions_roles.sql`:
```sql
-- migrate:up
-- Source: canon_engine/03 §1 (frozen v4.1) — roles support I-7. PG16: gen_random_uuid() is core.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'maintainer') THEN CREATE ROLE maintainer NOLOGIN; END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_reader') THEN CREATE ROLE app_reader NOLOGIN; END IF;
END $$;

-- migrate:down
DROP ROLE IF EXISTS app_reader;
DROP ROLE IF EXISTS maintainer;
```

- [ ] **Step 4: Run — expect PASS** (2/2).

- [ ] **Step 5: Commit**
```bash
git add core/db/migrations/20260610090001_extensions_roles.sql core/db/tests/10_schema_test.sql core/db/schema.sql
git commit -m "feat(0A): maintainer/app_reader roles (I-7 substrate)"
```

## Task 4: Migration — canon spine + append-only + DELETE guard (canon_event, event_participant)

**Files:** Create `core/db/migrations/20260610090002_canon_spine.sql`, `core/db/tests/20_append_only_test.sql`; Modify `core/db/tests/10_schema_test.sql`

- [ ] **Step 1: Extend schema test (failing).** Set `plan(10)`; add before `finish()`:
```sql
SELECT has_table('canon_event',       'canon_event exists');
SELECT has_table('event_participant', 'event_participant exists');
SELECT col_is_pk('canon_event', 'event_id', 'canon_event PK is event_id');
SELECT has_column('canon_event', 'in_world_tick', 'canon_event has in_world_tick (logical time, ADR-030)');
SELECT col_type_is('canon_event', 'in_world_tick', 'bigint', 'in_world_tick is BIGINT');
SELECT hasnt_column('canon_event', 'in_world_time', 'no TIMESTAMPTZ fictional clock (ADR-030)');
SELECT col_type_is('canon_event', 'recorded_at', 'timestamp with time zone', 'recorded_at is TIMESTAMPTZ (B-5)');
-- column-by-column verification vs doc 03 §1.1 (catches payload/schema_version drift)
SELECT columns_are('canon_event', ARRAY[
  'event_id','world_id','scene_id','beat_id','event_type','summary','payload','schema_version',
  'in_world_tick','in_world_label','beat_seq','temporal_uncertainty','recorded_at','accepted_at',
  'status','visibility_scope','confidence','origin','template_id','source_refs','superseded_by'],
  'canon_event columns match doc 03 §1.1 exactly (column-by-column)');
```

- [ ] **Step 2: Write the append-only + DELETE-guard test (failing)**

`core/db/tests/20_append_only_test.sql`:
```sql
BEGIN;
SELECT plan(5);

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000ff', '11111111-1111-1111-1111-111111111111',
        'move', 'test event', 10, 0, 'accepted', now(), 'private', 'fast_path');

SELECT throws_ok(
  $$ UPDATE canon_event SET summary='tampered' WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'append-only: editing summary raises');

SELECT throws_ok(
  $$ UPDATE canon_event SET status='proposed' WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'illegal status transition accepted->proposed raises');

SELECT lives_ok(
  $$ UPDATE canon_event SET status='superseded' WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  'legal transition accepted->superseded is allowed');

SELECT throws_ok(
  $$ DELETE FROM canon_event WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'DELETE on canon_event raises (append-only store)');

-- event_participant is a canon table: DELETE forbidden too
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('e0000000-0000-0000-0000-0000000000ff','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','speaker');
SELECT throws_ok(
  $$ DELETE FROM event_participant WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'DELETE on event_participant raises (canon table, ADR-006)');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 3: Run — expect FAIL** (`relation "canon_event" does not exist`).

- [ ] **Step 4: Write the migration**

`core/db/migrations/20260610090002_canon_spine.sql`:
```sql
-- migrate:up
-- Source: canon_engine/03 §1.1 (frozen v4.1)

CREATE TABLE canon_event (
  event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  scene_id         UUID,
  beat_id          UUID,
  event_type       TEXT NOT NULL,
  summary          TEXT NOT NULL,
  payload          JSONB NOT NULL DEFAULT '{}',
  schema_version   INT  NOT NULL DEFAULT 1,
  in_world_tick    BIGINT NOT NULL,
  in_world_label   TEXT,
  beat_seq         INT NOT NULL DEFAULT 0,
  temporal_uncertainty BOOLEAN NOT NULL DEFAULT false,
  recorded_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  accepted_at      TIMESTAMPTZ,
  status           TEXT NOT NULL DEFAULT 'proposed'
                   CHECK (status IN ('proposed','accepted','rejected','retconned','superseded')),
  visibility_scope TEXT NOT NULL DEFAULT 'private',
  confidence       REAL,
  origin           TEXT NOT NULL DEFAULT 'fast_path'
                   CHECK (origin IN ('fast_path','template','freeform','threshold','backstage','compensation')),
  template_id      TEXT,
  source_refs      JSONB,
  superseded_by    UUID REFERENCES canon_event(event_id)
);
CREATE INDEX idx_ce_world_time   ON canon_event (world_id, in_world_tick, beat_seq);
CREATE INDEX idx_ce_status       ON canon_event (world_id, status) WHERE status = 'accepted';
CREATE INDEX idx_ce_beat         ON canon_event (beat_id);
CREATE INDEX idx_ce_scene        ON canon_event (scene_id);
CREATE INDEX idx_ce_payload_gin  ON canon_event USING GIN (payload);

CREATE TABLE event_participant (
  event_id       UUID NOT NULL REFERENCES canon_event(event_id),
  entity_id      UUID NOT NULL,
  entity_kind    TEXT NOT NULL CHECK (entity_kind IN ('actor','location','artifact','faction','group')),
  role_qualifier TEXT NOT NULL,
  PRIMARY KEY (event_id, entity_id, role_qualifier)
);
CREATE INDEX idx_ep_entity ON event_participant (entity_id);

-- Append-only (doc 03 §1.1): only {status, accepted_at, superseded_by} may change. The ROW() list below
-- enumerates ALL 18 immutable columns verbatim from doc 03 §1.1 (payload + schema_version INCLUDED);
-- it must equal the CREATE TABLE column set minus the 3 mutable columns. The columns_are test (10_schema)
-- guards the CREATE TABLE side; keep both in sync if doc 03 ever changes (via ADR).
CREATE FUNCTION canon_event_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF ROW(NEW.event_id, NEW.world_id, NEW.scene_id, NEW.beat_id, NEW.event_type, NEW.summary,
         NEW.payload, NEW.schema_version, NEW.in_world_tick, NEW.in_world_label, NEW.beat_seq,
         NEW.temporal_uncertainty, NEW.recorded_at, NEW.visibility_scope, NEW.confidence,
         NEW.origin, NEW.template_id, NEW.source_refs)
     IS DISTINCT FROM
     ROW(OLD.event_id, OLD.world_id, OLD.scene_id, OLD.beat_id, OLD.event_type, OLD.summary,
         OLD.payload, OLD.schema_version, OLD.in_world_tick, OLD.in_world_label, OLD.beat_seq,
         OLD.temporal_uncertainty, OLD.recorded_at, OLD.visibility_scope, OLD.confidence,
         OLD.origin, OLD.template_id, OLD.source_refs)
  THEN
    RAISE EXCEPTION 'canon_event is append-only: only {status, accepted_at, superseded_by} may change (event %)', OLD.event_id;
  END IF;

  IF OLD.status IS DISTINCT FROM NEW.status
     AND NOT ( (OLD.status='proposed' AND NEW.status IN ('accepted','rejected'))
            OR (OLD.status='accepted' AND NEW.status IN ('retconned','superseded')) ) THEN
    RAISE EXCEPTION 'illegal canon_event status transition % -> %', OLD.status, NEW.status;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_canon_event_append_only
  BEFORE UPDATE ON canon_event FOR EACH ROW EXECUTE FUNCTION canon_event_append_only();

-- Generic DELETE guard for canon tables (doc 03 §1.1 "DELETE revoked"; ADR-006 invalidation-never-deletion).
CREATE FUNCTION forbid_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'DELETE forbidden on % (append-only canon, ADR-001/006)', TG_TABLE_NAME;
END $$;
CREATE TRIGGER trg_canon_event_no_delete
  BEFORE DELETE ON canon_event FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_event_participant_no_delete
  BEFORE DELETE ON event_participant FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TABLE IF EXISTS event_participant;   -- cascades its triggers
DROP TABLE IF EXISTS canon_event;         -- cascades its triggers
DROP FUNCTION IF EXISTS canon_event_append_only();
DROP FUNCTION IF EXISTS forbid_delete();
```

- [ ] **Step 5: Run — expect PASS** (`10_schema_test` 10/10, `20_append_only_test` 5/5).

- [ ] **Step 6: Commit**
```bash
git add core/db/migrations/20260610090002_canon_spine.sql core/db/tests/10_schema_test.sql core/db/tests/20_append_only_test.sql core/db/schema.sql
git commit -m "feat(0A): canon spine + ROW()-verified append-only + DELETE guard (doc 03 §1.1)"
```

---

## Task 5: Migration — `entity_registry`

**Files:** Create `core/db/migrations/20260610090003_registry.sql`; Modify `core/db/tests/10_schema_test.sql`

- [ ] **Step 1: Extend schema test (failing).** Set `plan(12)`; add:
```sql
SELECT has_table('entity_registry', 'entity_registry exists');
SELECT col_is_pk('entity_registry', 'entity_id', 'entity_registry PK is entity_id');
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Migration (doc 03 §1.5 verbatim)**

`core/db/migrations/20260610090003_registry.sql`:
```sql
-- migrate:up
-- Source: canon_engine/03 §1.5 (frozen v4.1)
CREATE TABLE entity_registry (
  entity_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  entity_kind      TEXT NOT NULL,
  canonical_name   TEXT NOT NULL,
  aliases          TEXT[] NOT NULL DEFAULT '{}',
  descriptor       TEXT,
  current_scene_id UUID,
  created_by_event UUID REFERENCES canon_event(event_id),
  status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','merged'))
);
CREATE INDEX idx_er_scene ON entity_registry (world_id, current_scene_id);
CREATE INDEX idx_er_name  ON entity_registry (world_id, canonical_name);

-- migrate:down
DROP TABLE IF EXISTS entity_registry;
```

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**
```bash
git add core/db/migrations/20260610090003_registry.sql core/db/tests/10_schema_test.sql core/db/schema.sql
git commit -m "feat(0A): entity_registry (doc 03 §1.5)"
```

---

## Task 6: Migration — deltas, lineage, epistemic, causal + DELETE guards

**Files:** Create `core/db/migrations/20260610090004_deltas_epistemic_causal.sql`, `core/db/tests/25_delete_guard_test.sql`; Modify `core/db/tests/10_schema_test.sql`

- [ ] **Step 1: Extend schema test (failing).** Set `plan(18)`; add:
```sql
SELECT has_table('state_mutation',      'state_mutation exists');
SELECT has_table('provenance_edge',     'provenance_edge exists (deployed, unused in 0A — ADR-008)');
SELECT has_table('perception_record',   'perception_record exists');
SELECT has_table('causal_bundle',       'causal_bundle exists (schema-ready, unused — ADR-008)');
SELECT has_table('causal_bundle_input', 'causal_bundle_input exists (schema-ready, unused — ADR-008)');
SELECT col_type_is('perception_record', 'acquired_tick', 'bigint', 'perception acquired_tick is logical (ADR-030)');
```

- [ ] **Step 2: Write the DELETE-guard test for the new canon tables (failing)**

`core/db/tests/25_delete_guard_test.sql`:
```sql
BEGIN;
SELECT plan(3);

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000de','11111111-1111-1111-1111-111111111111',
        'move','dg',7,0,'accepted',now(),'public','fast_path');

INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000de',
        '00000000-0000-0000-0000-00000000000a','actor','attrs.location_id','"x"'::jsonb,7,0);
SELECT throws_ok($$ DELETE FROM state_mutation WHERE event_id='e0000000-0000-0000-0000-0000000000de' $$,
  NULL,NULL,'DELETE on state_mutation raises (canon table, ADR-006)');

INSERT INTO perception_record (world_id,holder_id,source_event_id,content,epistemic_type,acquired_tick,valid_tick)
VALUES ('11111111-1111-1111-1111-111111111111','00000000-0000-0000-0000-00000000000a',
        'e0000000-0000-0000-0000-0000000000de','p','direct',7,7);
SELECT throws_ok($$ DELETE FROM perception_record WHERE source_event_id='e0000000-0000-0000-0000-0000000000de' $$,
  NULL,NULL,'DELETE on perception_record raises (ADR-006)');

INSERT INTO provenance_edge (derived_id,derived_kind,source_id,source_kind,how_type)
VALUES (gen_random_uuid(),'perception',gen_random_uuid(),'event','derived_from');
SELECT throws_ok($$ DELETE FROM provenance_edge $$, NULL,NULL,'DELETE on provenance_edge raises (canon lineage)');

SELECT * FROM finish();
ROLLBACK;
```
*(In the full suite this runs after migration 0006, so the `state_mutation` insert harmlessly projects to
`actor_state` inside the rolled-back transaction.)*

- [ ] **Step 3: Run — expect FAIL** (tables missing).

- [ ] **Step 4: Migration (doc 03 §1.2/§1.3/§1.4 verbatim + DELETE guards on the canon tables)**

`core/db/migrations/20260610090004_deltas_epistemic_causal.sql`:
```sql
-- migrate:up
-- Source: canon_engine/03 §1.2, §1.3, §1.4 (frozen v4.1)

CREATE TABLE state_mutation (
  mutation_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id       UUID NOT NULL,
  event_id       UUID NOT NULL REFERENCES canon_event(event_id),
  entity_id      UUID NOT NULL,
  entity_kind    TEXT NOT NULL,
  attribute_path TEXT NOT NULL,
  old_value      JSONB,
  new_value      JSONB NOT NULL,
  valid_from_tick BIGINT NOT NULL,
  valid_from_seq  INT NOT NULL DEFAULT 0,
  status         TEXT NOT NULL DEFAULT 'applied'
                 CHECK (status IN ('applied','reversed','dirty'))
);
CREATE INDEX idx_sm_entity ON state_mutation (entity_id, valid_from_tick, valid_from_seq);
CREATE INDEX idx_sm_event  ON state_mutation (event_id);

CREATE TABLE provenance_edge (
  derived_id   UUID NOT NULL,
  derived_kind TEXT NOT NULL CHECK (derived_kind IN ('perception','mutation','event','bundle')),
  source_id    UUID NOT NULL,
  source_kind  TEXT NOT NULL CHECK (source_kind  IN ('perception','mutation','event')),
  how_type     TEXT NOT NULL CHECK (how_type IN
               ('derived_from','inferred_from','reported_by','witnessed_by','compensates','supersedes')),
  PRIMARY KEY (derived_id, source_id, how_type)
);
CREATE INDEX idx_pe_source ON provenance_edge (source_id);

CREATE TABLE perception_record (
  perception_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  holder_id        UUID NOT NULL,
  source_event_id  UUID NOT NULL REFERENCES canon_event(event_id),
  content          TEXT NOT NULL,
  epistemic_type   TEXT NOT NULL CHECK (epistemic_type IN
                   ('direct','shared','told','overheard','public','rumor',
                    'inference','mistaken','confirmed','disputed')),
  sensory_mode     TEXT,
  confidence       REAL NOT NULL DEFAULT 1.0,
  distortion_level REAL NOT NULL DEFAULT 0,
  acquired_tick    BIGINT NOT NULL,
  valid_tick       BIGINT NOT NULL,
  invalid_tick     BIGINT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  expired_at       TIMESTAMPTZ,
  visibility_scope TEXT NOT NULL DEFAULT 'private',
  dirty            BOOLEAN NOT NULL DEFAULT false,
  importance       REAL NOT NULL DEFAULT 5.0
);
CREATE INDEX idx_pr_holder  ON perception_record (holder_id, acquired_tick);
CREATE INDEX idx_pr_source  ON perception_record (source_event_id);
CREATE INDEX idx_pr_active  ON perception_record (holder_id) WHERE invalid_tick IS NULL AND expired_at IS NULL;

CREATE TABLE causal_bundle (
  bundle_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id    UUID NOT NULL,
  effect_ref  UUID NOT NULL,
  effect_kind TEXT NOT NULL CHECK (effect_kind IN ('event','mutation')),
  semantics   TEXT NOT NULL CHECK (semantics IN ('conjunctive','disjunctive_member','probabilistic')),
  template_id TEXT,
  status      TEXT NOT NULL DEFAULT 'valid'
              CHECK (status IN ('valid','invalidated','pending_review')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cb_effect ON causal_bundle (effect_ref);

CREATE TABLE causal_bundle_input (
  bundle_id  UUID NOT NULL REFERENCES causal_bundle(bundle_id),
  input_ref  UUID NOT NULL,
  input_kind TEXT NOT NULL CHECK (input_kind IN ('event','mutation','perception')),
  role       TEXT NOT NULL CHECK (role IN ('trigger','enabler','blocker','influence')),
  polarity   SMALLINT NOT NULL DEFAULT 1 CHECK (polarity IN (1,-1)),
  weight     REAL NOT NULL DEFAULT 1.0,
  necessity  BOOLEAN NOT NULL DEFAULT true,
  PRIMARY KEY (bundle_id, input_ref, role)
);

-- DELETE guards on the canon/lineage tables (forbid_delete() defined in migration 0002).
CREATE TRIGGER trg_state_mutation_no_delete
  BEFORE DELETE ON state_mutation FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_perception_record_no_delete
  BEFORE DELETE ON perception_record FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_provenance_edge_no_delete
  BEFORE DELETE ON provenance_edge FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TABLE IF EXISTS causal_bundle_input;
DROP TABLE IF EXISTS causal_bundle;
DROP TABLE IF EXISTS perception_record;   -- cascades its DELETE-guard trigger
DROP TABLE IF EXISTS provenance_edge;     -- cascades its DELETE-guard trigger
DROP TABLE IF EXISTS state_mutation;      -- cascades its DELETE-guard trigger
```

- [ ] **Step 5: Run — expect PASS** (`10_schema_test` 18/18, `25_delete_guard_test` 3/3).

- [ ] **Step 6: Commit**
```bash
git add core/db/migrations/20260610090004_deltas_epistemic_causal.sql core/db/tests/10_schema_test.sql core/db/tests/25_delete_guard_test.sql core/db/schema.sql
git commit -m "feat(0A): deltas/lineage/epistemic + causal tables + DELETE guards (doc 03 §1.2-1.4; ADR-008)"
```

---

## Task 7: Migration — projection tables + I-7 grants

**Files:** Create `core/db/migrations/20260610090005_projections.sql`, `core/db/tests/60_permissions_test.sql`; Modify `core/db/tests/10_schema_test.sql`

- [ ] **Step 1: Extend schema test (failing).** Set `plan(23)`; add:
```sql
SELECT has_table('actor_state',        'actor_state exists');
SELECT has_table('location_state',     'location_state exists');
SELECT has_table('artifact_state',     'artifact_state exists');
SELECT has_table('relationship_state', 'relationship_state exists');
SELECT hasnt_column('relationship_state', 'updated_at',
       'relationship_state has no updated_at (doc 03 §1.5) — no volatile col to exclude in I-1');
```

- [ ] **Step 2: Write the I-7 projection-write test (failing)**

`core/db/tests/60_permissions_test.sql`:
```sql
BEGIN;
SELECT plan(2);

SET ROLE app_reader;
SELECT throws_ok(
  $$ INSERT INTO actor_state (entity_id, world_id)
     VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111') $$,
  '42501', NULL, 'app_reader INSERT into actor_state is denied (I-7)');
RESET ROLE;

SET ROLE app_reader;
SELECT lives_ok( $$ SELECT count(*) FROM actor_state $$, 'app_reader may SELECT projections');
RESET ROLE;

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 3: Run — expect FAIL** (`actor_state` missing).

- [ ] **Step 4: Migration (doc 03 §1.5 verbatim + grants)**

`core/db/migrations/20260610090005_projections.sql`:
```sql
-- migrate:up
-- Source: canon_engine/03 §1.5 (frozen v4.1)

CREATE TABLE actor_state (
  entity_id     UUID PRIMARY KEY,
  world_id      UUID NOT NULL,
  attrs         JSONB NOT NULL DEFAULT '{}',
  dirty         BOOLEAN NOT NULL DEFAULT false,
  last_event_id UUID,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE location_state (LIKE actor_state INCLUDING ALL);
CREATE TABLE artifact_state (LIKE actor_state INCLUDING ALL);

CREATE TABLE relationship_state (
  world_id      UUID NOT NULL,
  a_id          UUID NOT NULL,
  b_id          UUID NOT NULL,
  attrs         JSONB NOT NULL DEFAULT '{}',
  dirty         BOOLEAN NOT NULL DEFAULT false,
  last_event_id UUID,
  PRIMARY KEY (world_id, a_id, b_id)
);

-- I-7: projections writable only by the maintainer role; app_reader reads only.
REVOKE ALL ON actor_state, location_state, artifact_state, relationship_state FROM PUBLIC;
GRANT  ALL    ON actor_state, location_state, artifact_state, relationship_state TO maintainer;
GRANT  SELECT ON actor_state, location_state, artifact_state, relationship_state TO app_reader;

-- migrate:down
DROP TABLE IF EXISTS relationship_state;
DROP TABLE IF EXISTS artifact_state;
DROP TABLE IF EXISTS location_state;
DROP TABLE IF EXISTS actor_state;
```

- [ ] **Step 5: Run — expect PASS** (`10_schema_test` 23/23, `60_permissions_test` 2/2).

- [ ] **Step 6: Commit**
```bash
git add core/db/migrations/20260610090005_projections.sql core/db/tests/10_schema_test.sql core/db/tests/60_permissions_test.sql core/db/schema.sql
git commit -m "feat(0A): projection tables + I-7 maintainer-only write grants (doc 03 §1.5)"
```

---

## Task 8: Migration — `apply_mutation()` + projection trigger + `replay_0A()` (shared write path, hardened)

The correctness heart (Rider A). Adds I-7 function hardening (#2) and the `maintainer` SELECT grants the
`SECURITY DEFINER` functions need to read `canon_event`/`state_mutation`.

**Files:** Create `core/db/migrations/20260610090006_apply_mutation_and_triggers.sql`, `core/db/tests/30_apply_mutation_test.sql`; Modify `core/db/tests/60_permissions_test.sql`

- [ ] **Step 1: Write the behavior test (failing; fresh test-only entities — never seeded)**

`core/db/tests/30_apply_mutation_test.sql`:
```sql
BEGIN;
SELECT plan(5);

-- (1) accepted event + actor mutation projects via the live trigger
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111',
        'move','t',5,0,'accepted',now(),'public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000a1',
        '00000000-0000-0000-0000-0000000000f1','actor','attrs.location_id','"tavern"'::jsonb,5,0);
SELECT is( (SELECT attrs->>'location_id' FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1'),
       'tavern', 'live trigger applied the actor mutation (absolute set)');
SELECT is( (SELECT last_event_id FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1'),
       'e0000000-0000-0000-0000-0000000000a1'::uuid, 'last_event_id provenance set');

-- (2) idempotency (Rider B): standalone re-apply changes no domain value (excl. volatile updated_at)
CREATE TEMP TABLE _before AS
  SELECT attrs,last_event_id,dirty FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1';
SELECT apply_mutation(m.*) FROM state_mutation m WHERE m.entity_id='00000000-0000-0000-0000-0000000000f1';
SELECT is(
  (SELECT row(attrs,last_event_id,dirty) FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1'),
  (SELECT row(attrs,last_event_id,dirty) FROM _before),
  'apply_mutation idempotent on domain columns (absolute set)');

-- (3) relationship mutation = no-op stub (SPEC-001)
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000a1',
        '00000000-0000-0000-0000-0000000000f2','relationship','attrs.trust','0.5'::jsonb,5,1);
SELECT is( (SELECT count(*) FROM relationship_state)::int, 0,
       'relationship mutation is a documented no-op (SPEC-001): zero rows');

-- (4) mutation on a non-accepted parent does not project
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000a2','11111111-1111-1111-1111-111111111111',
        'move','p',6,0,'proposed','public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000a2',
        '00000000-0000-0000-0000-0000000000f3','actor','attrs.location_id','"road"'::jsonb,6,0);
SELECT is( (SELECT count(*) FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f3')::int,
       0, 'mutation on a non-accepted event does not project (doc 03 §3.1)');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run — expect FAIL** (`function apply_mutation does not exist`).

- [ ] **Step 3: Migration**

`core/db/migrations/20260610090006_apply_mutation_and_triggers.sql`:
```sql
-- migrate:up
-- Source: canon_engine/03 §3 (projection rules), §6 (replay); design §4.3/§6.5.
-- Rider A: apply_mutation() is the ONLY projection write path; live trigger AND replay both call it.

CREATE FUNCTION apply_mutation(m state_mutation) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  -- strip leading 'attrs.' (6 chars) -> single-key JSON path under attrs (0A convention, Rider B)
  jpath text[] := string_to_array(substring(m.attribute_path from 7), '.');
BEGIN
  IF m.entity_kind = 'actor' THEN
    INSERT INTO actor_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(actor_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'location' THEN
    INSERT INTO location_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(location_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'artifact' THEN
    INSERT INTO artifact_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(artifact_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'relationship' THEN
    -- SPEC-001: doc 03 does not define mutation->(a_id,b_id) addressing. NO-OP stub in 0A.
    NULL;
  END IF;
END $$;
ALTER FUNCTION apply_mutation(state_mutation) OWNER TO maintainer;

-- Live projection trigger: fire on accepted parent only (doc 03 §3.1).
CREATE FUNCTION sm_project() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
  IF (SELECT status FROM canon_event WHERE event_id = NEW.event_id) = 'accepted' THEN
    PERFORM apply_mutation(NEW);
  END IF;
  RETURN NEW;
END $$;
ALTER FUNCTION sm_project() OWNER TO maintainer;
CREATE TRIGGER trg_sm_project
  AFTER INSERT ON state_mutation FOR EACH ROW EXECUTE FUNCTION sm_project();

-- I-1 replay (design §6.5): snapshot -> truncate -> rebuild via the SAME apply_mutation -> domain diff.
-- DROP TABLE IF EXISTS makes it re-entrant within one transaction (the negative-control test calls it 3x).
CREATE FUNCTION replay_0A() RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE ev RECORD; m RECORD; diff_count int;
BEGIN
  DROP TABLE IF EXISTS snap_actor, snap_location, snap_artifact, snap_rel;
  CREATE TEMP TABLE snap_actor    ON COMMIT DROP AS SELECT * FROM actor_state;
  CREATE TEMP TABLE snap_location ON COMMIT DROP AS SELECT * FROM location_state;
  CREATE TEMP TABLE snap_artifact ON COMMIT DROP AS SELECT * FROM artifact_state;
  CREATE TEMP TABLE snap_rel      ON COMMIT DROP AS SELECT * FROM relationship_state;

  TRUNCATE actor_state, location_state, artifact_state, relationship_state;

  -- Rider C: domain-only deterministic order. recorded_at (volatile) excluded.
  FOR ev IN SELECT event_id FROM canon_event WHERE status='accepted'
            ORDER BY world_id, in_world_tick, beat_seq LOOP
    FOR m IN SELECT * FROM state_mutation WHERE event_id = ev.event_id
             ORDER BY valid_from_tick, valid_from_seq LOOP
      PERFORM apply_mutation(m);
    END LOOP;
  END LOOP;

  -- §6.5.1 per-table domain diff (exclude volatile updated_at; identity = PK).
  SELECT
      (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state)) d)
    + (SELECT count(*) FROM (
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel)
        UNION ALL
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state)) d)
  INTO diff_count;
  RETURN diff_count = 0;
END $$;
ALTER FUNCTION replay_0A() OWNER TO maintainer;

-- The SECURITY DEFINER functions run as maintainer; grant the reads they perform.
GRANT SELECT ON canon_event   TO maintainer;
GRANT SELECT ON state_mutation TO maintainer;

-- I-7 function hardening (#2): SECURITY DEFINER functions are doors through the grant wall.
REVOKE EXECUTE ON FUNCTION apply_mutation(state_mutation) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION sm_project()                   FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION replay_0A()                    FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION apply_mutation(state_mutation) TO maintainer;
GRANT  EXECUTE ON FUNCTION replay_0A()                    TO maintainer;

-- migrate:down
DROP FUNCTION IF EXISTS replay_0A();
DROP TRIGGER IF EXISTS trg_sm_project ON state_mutation;
DROP FUNCTION IF EXISTS sm_project();
DROP FUNCTION IF EXISTS apply_mutation(state_mutation);
```

- [ ] **Step 4: Extend `60_permissions_test.sql` for function hardening.** Set `plan(4)`; add before `finish()`:
```sql
SET ROLE app_reader;
SELECT throws_ok( $$ SELECT apply_mutation(NULL::state_mutation) $$, '42501', NULL,
  'app_reader cannot EXECUTE apply_mutation (I-7 function hardening)');
SELECT throws_ok( $$ SELECT replay_0A() $$, '42501', NULL,
  'app_reader cannot EXECUTE replay_0A (I-7 function hardening)');
RESET ROLE;
```

- [ ] **Step 5: Run — expect PASS** (`30_apply_mutation_test` 5/5, `60_permissions_test` 4/4).

- [ ] **Step 6: Commit**
```bash
git add core/db/migrations/20260610090006_apply_mutation_and_triggers.sql core/db/tests/30_apply_mutation_test.sql core/db/tests/60_permissions_test.sql core/db/schema.sql
git commit -m "feat(0A): shared apply_mutation() + projection trigger + replay_0A() + I-7 fn hardening (Rider A)"
```

---

## Task 9: Seed part 1 — clean-DB guard + `entity_registry` cast + helpers

From here run **`make reset && make test`** (seed targets a clean DB; `reset` provides it).

**Files:** Create `core/db/seeds/seed_mara_0A.sql`, `core/db/tests/helpers.sql`, `core/db/tests/40_perception_test.sql`

- [ ] **Step 1: Write the by-hand var helper**

`core/db/tests/helpers.sql`:
```sql
-- Fixed UUIDs for by-hand checks_0A.sql. Usage: psql -f helpers.sql -f checks_0A.sql
\set world_id   '11111111-1111-1111-1111-111111111111'
\set player_id  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
\set mara_id    'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
\set jonas_id   'cccccccc-cccc-cccc-cccc-cccccccccccc'
\set e1_id      'e0000000-0000-0000-0000-000000000001'
\set e102_id    'e0000000-0000-0000-0000-000000000102'
```

- [ ] **Step 2: Write the registry-presence test (failing)**

`core/db/tests/40_perception_test.sql`:
```sql
BEGIN;
SELECT plan(1);
SELECT is( (SELECT count(*) FROM entity_registry
            WHERE world_id='11111111-1111-1111-1111-111111111111')::int, 11,
       'registry seeded with cast: P,M,J,Tavern,PUB,O1..O5,Square');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 3: Start the seed with the clean-DB guard + registry (NO truncate/delete — #5)**

`core/db/seeds/seed_mara_0A.sql`:
```sql
-- =====================================================================================
-- seed_mara_0A.sql — deterministic Mara scenario (doc 13 §4). Targets a CLEAN DB ONLY.
-- Re-run path is `make reset` (NOT a self-clearing script). A seed that DELETEs canon or
-- disables append-only enforcement would be the silent-workaround pattern made executable.
-- CONVENTION (Rider B): every state_mutation.new_value is an ABSOLUTE state set at a single-key
-- path under attrs. (e.g. attrs.location_id). No deltas.
-- =====================================================================================
BEGIN;

-- Clean-DB guard: refuse to seed a non-empty world (canon is append-only; no DELETE here).
DO $$ BEGIN
  IF (SELECT count(*) FROM canon_event) > 0 THEN
    RAISE EXCEPTION 'seed_mara_0A targets a CLEAN DB only — run `make reset` first';
  END IF;
END $$;

-- Cast (doc 13 §4) + noise NPCs ("...other moves", §4) + one named noise location + a
-- common-knowledge holder for the public record (§5 "a public knowledge record exists").
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111','actor',   'Player'),
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111','actor',   'Mara'),
 ('cccccccc-cccc-cccc-cccc-cccccccccccc','11111111-1111-1111-1111-111111111111','actor',   'Jonas'),
 ('dddddddd-dddd-dddd-dddd-dddddddddddd','11111111-1111-1111-1111-111111111111','location','Tavern'),
 ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','11111111-1111-1111-1111-111111111111','faction', 'Common Knowledge'),
 ('00000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','actor',   'O1'),
 ('00000000-0000-0000-0000-000000000002','11111111-1111-1111-1111-111111111111','actor',   'O2'),
 ('00000000-0000-0000-0000-000000000003','11111111-1111-1111-1111-111111111111','actor',   'O3'),
 ('00000000-0000-0000-0000-000000000004','11111111-1111-1111-1111-111111111111','actor',   'O4'),
 ('00000000-0000-0000-0000-000000000005','11111111-1111-1111-1111-111111111111','actor',   'O5'),
 ('000000a0-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111','location','Square');

-- (E1, noise, E102 appended by later tasks, before COMMIT)
COMMIT;
```
*(Noise moves actors to `market`/`road`/`dock` too; those appear only as JSON values in
`attrs.location_id`, not registry rows.)*

- [ ] **Step 4: Run — expect PASS** (registry count 11). Run: `make reset && make test`.

- [ ] **Step 5: Commit**
```bash
git add core/db/seeds/seed_mara_0A.sql core/db/tests/helpers.sql core/db/tests/40_perception_test.sql
git commit -m "feat(0A): seed registry cast + clean-DB guard (no canon-clearing; reset is the re-run path)"
```

---

## Task 10: Seed part 2 — E1 disclosure + perceptions

**Files:** Modify `core/db/seeds/seed_mara_0A.sql`, `core/db/tests/40_perception_test.sql`

- [ ] **Step 1: Extend the perception test (failing).** Set `plan(4)`; add:
```sql
SELECT is( (SELECT count(*) FROM perception_record
            WHERE holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
              AND source_event_id='e0000000-0000-0000-0000-000000000001'
              AND epistemic_type='told' AND invalid_tick IS NULL AND expired_at IS NULL)::int,
       1, 'Mara has exactly one active told perception of E1 (mara_knows_ok)');
SELECT is( (SELECT count(*) FROM perception_record
            WHERE holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
              AND source_event_id='e0000000-0000-0000-0000-000000000001'
              AND epistemic_type='shared')::int, 1, 'Player has one shared perception of E1');
SELECT is( (SELECT count(*) FROM perception_record
            WHERE holder_id='cccccccc-cccc-cccc-cccc-cccccccccccc'
              AND source_event_id='e0000000-0000-0000-0000-000000000001')::int,
       0, 'Jonas has ZERO perceptions of E1 (knowledge boundary, j_ignorant_ok)');
```

- [ ] **Step 2: Run — expect FAIL** (`make reset && make test`).

- [ ] **Step 3: Add E1 to the seed (immediately before `COMMIT;`)**
```sql
-- E1 @ tick 100: P privately discloses the secret to M (doc 13 §4). No state mutation.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111',
        'private_disclosure','P tells M the mayor keeps a hidden ledger',100,0,
        'Day 1', 'accepted', now(), 'private', 'fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-000000000001','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','speaker'),
 ('e0000000-0000-0000-0000-000000000001','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','listener');
INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                               acquired_tick, valid_tick) VALUES
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  'e0000000-0000-0000-0000-000000000001','P told me the mayor keeps a hidden ledger','told',100,100),
 ('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  'e0000000-0000-0000-0000-000000000001','I told Mara the mayor keeps a hidden ledger','shared',100,100);
```

- [ ] **Step 4: Run — expect PASS** (`40_perception_test` 4/4).

- [ ] **Step 5: Commit**
```bash
git add core/db/seeds/seed_mara_0A.sql core/db/tests/40_perception_test.sql
git commit -m "feat(0A): seed E1 private_disclosure + told/shared perceptions (knowledge boundary)"
```

---

## Task 11: Seed part 3 — 100 deterministic noise events (fixed UUIDs) + provenance + guards

**Files:** Modify `core/db/seeds/seed_mara_0A.sql`; Create `core/db/tests/50_provenance_test.sql`, `core/db/tests/70_determinism_guards_test.sql`

- [ ] **Step 1: Write the provenance + count test (failing)**

`core/db/tests/50_provenance_test.sql`:
```sql
BEGIN;
SELECT plan(3);
SELECT is( (SELECT count(*) FROM state_mutation sm
            LEFT JOIN canon_event ce ON ce.event_id=sm.event_id AND ce.status='accepted'
            WHERE ce.event_id IS NULL)::int, 0, 'I-2: zero orphan state_mutations');
SELECT is( (SELECT count(*) FROM perception_record pr
            LEFT JOIN canon_event ce ON ce.event_id=pr.source_event_id AND ce.status='accepted'
            WHERE ce.event_id IS NULL)::int, 0, 'I-2: zero orphan perceptions');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick BETWEEN 101 AND 200)::int,
       100, '100 noise events present');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Write the determinism-guard test (failing)**

`core/db/tests/70_determinism_guards_test.sql`:
```sql
BEGIN;
SELECT plan(3);
SELECT is( (SELECT count(*) FROM (
              SELECT world_id,in_world_tick,beat_seq FROM canon_event WHERE status='accepted'
              GROUP BY world_id,in_world_tick,beat_seq HAVING count(*)>1) dup)::int,
       0, 'Rider C: (world_id,in_world_tick,beat_seq) unique across accepted events');
SELECT is( (SELECT count(*) FROM (
              SELECT entity_id,attribute_path,valid_from_tick,valid_from_seq FROM state_mutation
              GROUP BY entity_id,attribute_path,valid_from_tick,valid_from_seq HAVING count(*)>1) dup)::int,
       0, 'Rider C: mutation order key unique per (entity,attribute)');
SELECT is( (SELECT count(*) FROM relationship_state)::int, 0,
       'SPEC-001: relationship_state is empty in 0A (intentional vacuous satisfaction)');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 3: Run — expect FAIL** (0 noise events).

- [ ] **Step 4: Append the deterministic noise generator to the seed (before `COMMIT;`)**
```sql
-- 100 noise events, ticks 101..200, beat_seq 0 (each tick unique => total order). FULLY DETERMINISTIC:
-- event_id = 'e0000000-0000-0000-0000-9' + lpad(i,11,'0')  (collision-free vs E1 ...001 / E102 ...102).
-- Rule (hand-verifiable): for i in 1..100, tick=100+i,
--   actor    = (P,M,J,O1,O2,O3,O4,O5)[(i % 8)+1]   (1-based SQL arrays)
--   location = ('tavern','square','market','road','dock')[(i % 5)+1]
-- Each event: 'move', one ABSOLUTE attrs.location_id set, one 'direct' perception for the mover.
DO $$
DECLARE
  i int; tick bigint; ev uuid; actor uuid; loc text;
  actors uuid[] := ARRAY[
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
    'cccccccc-cccc-cccc-cccc-cccccccccccc','00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000005']::uuid[];
  locs text[] := ARRAY['tavern','square','market','road','dock'];
BEGIN
  FOR i IN 1..100 LOOP
    tick  := 100 + i;
    ev    := ('e0000000-0000-0000-0000-9' || lpad(i::text, 11, '0'))::uuid;
    actor := actors[(i % 8) + 1];
    loc   := locs[(i % 5) + 1];
    INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                             status, accepted_at, visibility_scope, origin)
    VALUES (ev,'11111111-1111-1111-1111-111111111111','move',
            'noise move '||i, tick, 0, 'accepted', now(), 'public', 'fast_path');
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
    VALUES (ev, actor, 'actor', 'instigator');
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES ('11111111-1111-1111-1111-111111111111', ev, actor, 'actor', 'attrs.location_id',
            to_jsonb(loc), tick, 0);
    INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                   acquired_tick, valid_tick)
    VALUES ('11111111-1111-1111-1111-111111111111', actor, ev, 'I moved to '||loc, 'direct', tick, tick);
  END LOOP;
END $$;
```

- [ ] **Step 5: Run — expect PASS** (`50_provenance` 3/3, `70_determinism_guards` 3/3).

- [ ] **Step 6: Commit**
```bash
git add core/db/seeds/seed_mara_0A.sql core/db/tests/50_provenance_test.sql core/db/tests/70_determinism_guards_test.sql
git commit -m "feat(0A): seed 100 deterministic (fixed-UUID) noise events + I-2 provenance + Rider C guards"
```

---

## Task 12: Seed part 4 — E102 publicize + public knowledge; Mara survives

**Files:** Modify `core/db/seeds/seed_mara_0A.sql`, `core/db/tests/40_perception_test.sql`

- [ ] **Step 1: Extend the perception test (failing).** Set `plan(6)`; add:
```sql
SELECT is( (SELECT count(*) FROM perception_record
            WHERE source_event_id='e0000000-0000-0000-0000-000000000102'
              AND epistemic_type='public')::int, 1, 'public knowledge record exists (public_knowledge_ok)');
SELECT is( (SELECT count(*) FROM perception_record
            WHERE holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
              AND source_event_id='e0000000-0000-0000-0000-000000000001')::int, 1,
       'Mara original perception SURVIVES publication (ADR-006, mara_perception_survives_ok)');
```

- [ ] **Step 2: Run — expect FAIL** (public record absent).

- [ ] **Step 3: Append E102 to the seed (before `COMMIT;`)**
```sql
-- E102 @ tick 201: P publicizes the ledger. No state mutation. Present-forward (ADR-016):
-- M's E1 perception untouched; public-knowledge record created (held by Common Knowledge);
-- J is *eligible* but acquires nothing in 0A (Phase-1 fan-out, doc 13 §5).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-000000000102','11111111-1111-1111-1111-111111111111',
        'publicize','the hidden ledger becomes common knowledge',201,0,
        'Day 2', 'accepted', now(), 'public', 'fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-000000000102','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                               acquired_tick, valid_tick, visibility_scope) VALUES
 ('11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
  'e0000000-0000-0000-0000-000000000102','It is now common knowledge that the mayor keeps a hidden ledger',
  'public',201,201,'public');
```

- [ ] **Step 4: Run — expect PASS** (`40_perception_test` 6/6).

- [ ] **Step 5: Commit**
```bash
git add core/db/seeds/seed_mara_0A.sql core/db/tests/40_perception_test.sql
git commit -m "feat(0A): seed E102 publicize + public knowledge; Mara perception survives (ADR-006)"
```

---

## Task 13: `checks_0A.sql` — doc 13 §7 boolean suite (operator-runnable)

**Files:** Create `core/db/tests/checks_0A.sql`

- [ ] **Step 1: Write the verbatim §7 suite (replay wrapped in BEGIN/ROLLBACK — #6)**

`core/db/tests/checks_0A.sql`:
```sql
-- doc 13 §7 pass/fail checks, verbatim. Run by hand:
--   docker compose exec -T db psql -U postgres -d dreamchat -f /work/tests/helpers.sql -f /work/tests/checks_0A.sql
-- Every column must return TRUE.
SELECT count(*) = 0 AS i2_mutations_ok
FROM state_mutation sm
LEFT JOIN canon_event ce ON ce.event_id = sm.event_id AND ce.status='accepted'
WHERE ce.event_id IS NULL;

SELECT count(*) = 0 AS i2_perceptions_ok
FROM perception_record pr
LEFT JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.status='accepted'
WHERE ce.event_id IS NULL;

SELECT count(*) = 0 AS j_ignorant_ok
FROM perception_record WHERE holder_id = :'jonas_id' AND source_event_id = :'e1_id';

SELECT count(*) = 1 AS mara_knows_ok
FROM perception_record
WHERE holder_id = :'mara_id' AND source_event_id = :'e1_id'
  AND epistemic_type='told' AND invalid_tick IS NULL AND expired_at IS NULL;

SELECT count(*) = 1 AS mara_perception_survives_ok
FROM perception_record WHERE holder_id = :'mara_id' AND source_event_id = :'e1_id';

SELECT count(*) >= 1 AS public_knowledge_ok
FROM perception_record WHERE source_event_id = :'e102_id' AND epistemic_type='public';

-- I-1 replay (single boolean). Wrapped so the truncate/rebuild does not persist.
BEGIN;
SELECT replay_0A() AS i1_replay_ok;
ROLLBACK;
```

- [ ] **Step 2: Run by hand — expect all TRUE**
```bash
make reset
docker compose exec -T db psql -U postgres -d dreamchat -f /work/tests/helpers.sql -f /work/tests/checks_0A.sql
```
Expected: `i2_mutations_ok`, `i2_perceptions_ok`, `j_ignorant_ok`, `mara_knows_ok`,
`mara_perception_survives_ok`, `public_knowledge_ok`, `i1_replay_ok` all `t`.

- [ ] **Step 3: Commit**
```bash
git add core/db/tests/checks_0A.sql
git commit -m "test(0A): checks_0A.sql — doc 13 §7 boolean suite (operator-runnable exit gate)"
```

---

## Task 14: Golden projection spot-check (hand-computed)

**Files:** Create `core/db/tests/expected_projections_0A.csv`, `core/db/tests/80_golden_projection_test.sql`

- [ ] **Step 1: Write the hand-computed golden CSV**

`core/db/tests/expected_projections_0A.csv`:
```csv
entity_id,location_id
aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa,square
bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb,market
cccccccc-cccc-cccc-cccc-cccccccccccc,road
00000000-0000-0000-0000-000000000001,dock
00000000-0000-0000-0000-000000000002,tavern
00000000-0000-0000-0000-000000000003,road
00000000-0000-0000-0000-000000000004,dock
00000000-0000-0000-0000-000000000005,tavern
```
*Derivation (hand): last `i` with `i%8==k` over 1..100 → P:i96 loc(96%5→idx1)=square, M:i97 market,
J:i98 road, O1:i99 dock, O2:i100 tavern, O3:i93 road, O4:i94 dock, O5:i95 tavern.*

- [ ] **Step 2: Write the golden spot-check test (failing if seed/rule drifts)**

`core/db/tests/80_golden_projection_test.sql`:
```sql
BEGIN;
SELECT plan(1);
CREATE TEMP TABLE expected (entity_id uuid, location_id text);
INSERT INTO expected VALUES
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','square'),
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','market'),
 ('cccccccc-cccc-cccc-cccc-cccccccccccc','road'),
 ('00000000-0000-0000-0000-000000000001','dock'),
 ('00000000-0000-0000-0000-000000000002','tavern'),
 ('00000000-0000-0000-0000-000000000003','road'),
 ('00000000-0000-0000-0000-000000000004','dock'),
 ('00000000-0000-0000-0000-000000000005','tavern');
SELECT set_eq(
  'SELECT entity_id, attrs->>''location_id'' FROM actor_state',
  'SELECT entity_id, location_id FROM expected',
  'actor_state final locations match the hand-computed golden (doc 13 §5 spot-check)');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 3: Run — expect PASS** (`make reset && make test`).

- [ ] **Step 4: Commit**
```bash
git add core/db/tests/expected_projections_0A.csv core/db/tests/80_golden_projection_test.sql
git commit -m "test(0A): hand-computed golden projection spot-check (doc 13 §5)"
```

---

## Task 15: I-1 replay invariance + negative control (detect → repair)

**Files:** Create `core/db/tests/90_replay_test.sql`, `core/db/tests/replay_0A.sql`

- [ ] **Step 1: Write the I-1 test — happy path + detection + repair (failing until full seed)**

`core/db/tests/90_replay_test.sql`:
```sql
BEGIN;
SELECT plan(3);

-- (1) happy path: rebuild from the event log == live domain state
SELECT ok( replay_0A(), 'I-1: replay reproduces domain-equivalent projection state (ADR-026)' );

-- (2) detection: corrupt one live projection value; the snapshot captures the corruption, so the
--     event-log rebuild differs from the snapshot and the diff MUST bite -> replay_0A() returns FALSE.
UPDATE actor_state SET attrs = jsonb_set(attrs,'{location_id}','"WRONG"',true)
 WHERE entity_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb';
SELECT ok( NOT replay_0A(),
       'negative control: corrupted snapshot != event-log rebuild -> replay_0A() returns FALSE' );

-- (3) repair: the prior replay_0A() rebuilt the table from the event log, repairing the corruption,
--     so the snapshot now equals the rebuild -> replay_0A() returns TRUE.
SELECT ok( replay_0A(), 'repair: after rebuild the projection matches the event log again' );

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Write the by-hand replay runner**

`core/db/tests/replay_0A.sql`:
```sql
-- By-hand I-1 runner (exit gate). Wrapped so the truncate/rebuild does not persist.
BEGIN;
SELECT replay_0A() AS i1_replay_ok;
ROLLBACK;
```

- [ ] **Step 3: Run — expect PASS** (`90_replay_test` 3/3). Run: `make reset && make test`.

- [ ] **Step 4: Commit**
```bash
git add core/db/tests/90_replay_test.sql core/db/tests/replay_0A.sql
git commit -m "test(0A): I-1 replay invariance + negative control (detect corruption, repair on rebuild)"
```

---

## Task 16: CI workflow + schema.sql audit artifact

**Files:** Create `.github/workflows/invariants.yml`

- [ ] **Step 1: Verify the schema audit artifact is clean locally**

Run: `make schema-check`
Expected: `dbmate dump` regenerates `core/db/schema.sql`; `git diff --exit-code` reports no drift. If it
drifts, commit the regenerated `schema.sql`.

- [ ] **Step 2: Write the CI workflow (Docker-based, seed on a clean DB then test)**

`.github/workflows/invariants.yml`:
```yaml
name: invariants
on:
  push: { branches: ["**"] }
  pull_request:
jobs:
  phase-0a:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build db image (pinned Postgres 16 + pgTAP + pg_prove)
        run: docker compose build db
      - name: Reset to a clean, migrated, seeded DB
        run: make reset
      - name: Run invariant suite (pgTAP: I-1/I-2/I-7 + guards + golden)
        run: make test
      - name: Schema audit (schema.sql matches committed)
        run: make schema-check
```

- [ ] **Step 3: Full local run — expect PASS**

Run: `make reset && make test && make schema-check`
Expected: every test file `ok`, `Result: PASS`, no schema drift.

- [ ] **Step 4: Commit**
```bash
git add .github/workflows/invariants.yml core/db/schema.sql
git commit -m "chore(0A): CI invariants workflow + schema.sql audit artifact"
```

---

## Task 17: GATE — doc 13 §8 Definition of Done (operator-run; Validation-Ladder Q1)

Verification, not new code. **Do not proceed to Chunk 2 until every box is checked.**

- [ ] **Step 1: Clean-deploy determinism (doc 13 §8.1, §8.5) — now "same data", not just "same checks"**
```bash
make reset
docker compose exec -T db psql -U postgres -d dreamchat -At -c \
  "SELECT entity_id, attrs->>'location_id' FROM actor_state ORDER BY entity_id" > /tmp/run1.txt
make reset
docker compose exec -T db psql -U postgres -d dreamchat -At -c \
  "SELECT entity_id, attrs->>'location_id' FROM actor_state ORDER BY entity_id" > /tmp/run2.txt
diff /tmp/run1.txt /tmp/run2.txt && echo "DETERMINISTIC: identical projection DATA across fresh DBs"
```
Expected: no diff. *(Deterministic event UUIDs + absolute sets + domain-only ordering make the actual rows
identical, not merely the boolean check outputs.)*

- [ ] **Step 2: §7 checks all TRUE (doc 13 §8.2) — the human exit gate.** Run `checks_0A.sql` (Task 13 Step 2); confirm `i1_replay_ok=t` and all six §7 booleans `t`.

- [ ] **Step 3: Should-fail tests raise (doc 13 §8.3).** Confirm green in `make test`: `20_append_only_test` (UPDATE/DELETE on canon_event + event_participant), `25_delete_guard_test` (state_mutation/perception_record/provenance_edge), `60_permissions_test` (I-7 projection write + function EXECUTE).

- [ ] **Step 4: Replay empty diff (doc 13 §8.4).** Run `make replay` → `t`. (And `90_replay_test` green.)

- [ ] **Step 5: Invariant gate (doc 07 §6).** Three Phase-0A invariants green on the Mara spine:
  - **I-1** → `replay_0A()` TRUE + negative control detects/repairs (`90_replay_test`).
  - **I-2** → `i2_mutations_ok` + `i2_perceptions_ok` (`50_provenance_test`).
  - **I-7** → projection-write denial + function-EXECUTE denial (`60_permissions_test`).

- [ ] **Step 6: Answer Validation-Ladder Q1 honestly (playbook §0.5)**

> "Can world state replay deterministically?"

Green CI is necessary, not sufficient. Confirm determinism did not hold *only* by excluding things that
matter — the only domain-diff exclusion is `updated_at`; the only ordering change is Rider C dropping
volatile `recorded_at` (SPEC-002). If replay diverges or determinism is hollow → **STOP, debug, do not
climb to Chunk 2.**

- [ ] **Step 7: Tag the gate**
```bash
git tag -a chunk-1-0A-gate -m "Phase 0A gate green: I-1/I-2/I-7 on the Mara spine; Validation-Ladder Q1 = yes"
```

---

## Appendix A — Non-sanctioned no-Docker exception (gate NOT authoritative)

> **Not a sanctioned path.** Docker (Task 1: `colima + docker`, or Docker Desktop) is the only environment
> the gate trusts (local == CI == deployed image). Use this ONLY in a genuine emergency where no Docker
> runtime can be installed, and treat any green result as provisional: **the Phase 0A gate is authoritative
> only when the Docker CI run is green.** Never run this in parallel with Docker as an alternative.

Pinned to Postgres 16 + pgTAP (must match the Docker image family):
```bash
# EMERGENCY ONLY — results are NOT gate-authoritative; re-validate on Docker CI before claiming the gate.
brew install postgresql@16 pgtap dbmate        # postgresql@16 ONLY; never a newer major
brew services start postgresql@16
createdb dreamchat
psql dreamchat -c 'CREATE EXTENSION IF NOT EXISTS pgtap;'
export DATABASE_URL='postgres:///dreamchat?sslmode=disable'
export DBMATE_MIGRATIONS_DIR=./core/db/migrations
export DBMATE_SCHEMA_FILE=./core/db/schema.sql
dbmate up
psql dreamchat -v ON_ERROR_STOP=1 -f core/db/seeds/seed_mara_0A.sql   # clean DB only (guarded)
pg_prove -d dreamchat --ext .sql core/db/tests
```

---

## Self-review notes (author)

- **Spec coverage:** schema scope §4.1→T4-7; apply_mutation/Rider A §4.3→T8; relationship no-op/SPEC-001
  §4.4→T8/T11; seed §5→T9-12; Rider B absolute sets→T8/T11; §6.1 §7 checks→T10/T12/T13; §6.2 should-fail→
  T4/T6/T7/T8; §6.3 golden→T14; §6.4 guards→T11; §6.5 replay/Rider C→T8/T15; §7 layout/dbmate/schema.sql→
  T1/T16; §8 CI→T16; §9 DoD/gate→T17; §11 open-spec→T2. Review riders: negative control→T15; I-7 fn
  hardening→T8; canon-table DELETE guards→T4/T6; ROW() column verification→T4; clean-DB seed guard→T9;
  deterministic noise UUIDs→T11; Docker-only→T1/Appendix A.
- **No placeholders:** all SQL/tests concrete; no TBD/TODO.
- **Type/name consistency:** `apply_mutation(state_mutation)`, `sm_project()`, `replay_0A()`,
  `canon_event_append_only()`, `forbid_delete()` used identically across tasks; fixed UUIDs consistent;
  `10_schema_test` plan grows 2→10→12→18→23; `40_perception_test` 1→4→6; `60_permissions_test` 2→4.
- **Determinism chain:** fixed event UUIDs + absolute new_value + `(world_id,in_world_tick,beat_seq)` total
  order (no `recorded_at`) ⇒ §8.5 compares identical DATA, and I-1 rebuilds via the production
  `apply_mutation()` path (Rider A), so replay can't pass by reimplementation.
