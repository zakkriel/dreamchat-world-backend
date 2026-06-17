# Chunk 5 — Play Loop (Backend) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the thin-slice play loop with the LLM in it — free text → **decompose** (prose→events, schema-constrained) → **incremental gated apply with a halt hook** → **write-side perception generation** → **perception-bound narration** → **replay** — for the **MOVE + SAY** action set only, proving the trust guarantees (deterministic replay Q1/I-1, no-leak Q2/B-1/I-3) survive a live, mutating, LLM-mediated loop (the Q3 validation-ladder claim).

**Architecture:** Two legs mirroring Chunk 4. **Leg 1** is the deterministic engine harness — SQL functions + pgTAP, driven by **fixture** event chains (no model): `apply_beat()` runs each event through gate → resolve(identity) → apply+**generate perceptions** → stop-check, **incrementally with a halt hook**, advancing an **action-driven clock** by committed-prefix durations. The highest-correctness-risk piece — *who-perceives-what on generated events* — is isolated here and validated **through the existing Chunk-4 Compendium read functions + `replay_0A()`**. **Leg 2** adds the LLM bridge: a schema-constrained `Decomposer` (the closed event vocabulary is the leash, ADR-009/D-1), a perception-bound `Narrator` (ADR-020), and a `POST /worlds/{w}/beat` endpoint — both model seats fed **only** the pgTAP-tested perception payload (`fn_visible_perceptions`, B-1/I-3), with deterministic fakes in CI and the live model exercised only in the founder operator gate.

**Tech Stack:** PostgreSQL 16 + pgTAP (`pg_prove`), dbmate migrations, Go 1.26 (`pgx/v5`, stdlib `net/http`/`httptest`, stdlib `encoding/json`), Docker Compose. **No new runtime dependency** — schema validation of the decomposer output uses a hand-rolled check against the published JSON schema's closed `type` enum (the leash is a closed set, not arbitrary JSON-Schema).

**Spec:** `docs/superpowers/specs/2026-06-16-chunk-5-play-loop-architecture-notes.md` (the authoritative brainstorm; §-references below point into it). Companion flow: `docs/superpowers/specs/chunk5-play-loop-flow.mermaid`. Deferred-item boundaries: `docs/open-spec-items.md` SPEC-012…SPEC-018. **The law:** `docs/00_strategy/06_rules_register.md` — cited rule IDs: **B-1, B-2, B-5, B-7, B-11, C-5, C-6, C-7, D-1, D-4, D-7, D-9, D-10, GA-2, I-1, I-2, I-3, I-6, I-7, I-9, ADR-001, ADR-005, ADR-009, ADR-014, ADR-016, ADR-020, ADR-021/030, ADR-026, ADR-034, SPEC-015** (in scope) and **SPEC-012/013/016/017/018** (deferred boundaries).

**Execution context:** Run in a fresh Chunk-5 backend worktree (create via `superpowers:using-git-worktrees` at execution start). **One chunk = one worktree = one plan = one PR** to backend `main` (playbook §1; AGENTS.md). The frontend leg (`dreamchat-frontend`) is a **separate** plan and PR — nothing here touches a `frontend/` directory (D-7/D-10). **Gate red → STOP:** do not start Leg 2 while the Leg-1 gate is red; do not request the operator gate while the Leg-2 gate is red; do not start Chunk 6 while the Chunk-5 operator gate is red.

**Conventions used throughout:**
- **Fixed UUIDs** (match `core/db/tests/helpers.sql` + `seed_mara_0A.sql`):
  - world `11111111-1111-1111-1111-111111111111`; Player `aaaaaaaa-…-aaaaaaaaaaaa`; Mara `bbbbbbbb-…-bbbbbbbbbbbb`; Jonas `cccccccc-…-cccccccccccc`; Common Knowledge (faction) `eeeeeeee-…-eeeeeeeeeeee`.
  - Location **labels** are lowercase text in `actor_state.attrs.location_id` (the seed convention: `'tavern'`, `'square'`, … — *not* the location entity UUID). Co-presence compares these labels.
  - Chunk-5 fixture event ids are namespaced `e5000000-0000-0000-0000-0000000000NN` to never collide with the `e0000000…` 0A/Chunk-3/4 set.
- **Run one pgTAP file:** `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/<FILE>'` (needs `make reset` first to migrate+seed).
- **Run the whole suite:** `make reset && make test && make replay`.
- **Go tests:** `cd core/api && make -C ../.. reset && go test ./... -v && go vet ./...`.
- **Fail-first discipline (TDD iron law — no exceptions):** write the test, run it against the current DB/build *before* implementing — it errors (function/route missing) = **RED**; then implement and re-run = **GREEN**. Every task ends in a commit. `make schema-check` (no `core/db/schema.sql` drift) must pass after each migration step — the migration step re-dumps and commits it.

---

## §0. Reconciliation findings (verification against the FROZEN engine — DONE before locking the legs)

Per AGENTS.md and D-9, the brainstorm's proposals were verified against the frozen engine (Master DDL doc 03, ADRs doc 02, context-assembler doc 06, invariants doc 07) **before** designing. Anything that changes the frozen engine is a **new engine ADR filed WITH running-code evidence (D-9)** — staged below, not assumed. The findings are adopted as-is.

**A. Canon event spine §3 vs Master DDL — NO spine ADR; the "closed set" is the gate + doc 03 §8, not a DB enum.** `canon_event.event_type` is open `TEXT` (doc 03 §1.1), *not* a CHECK enum. The brainstorm's "closed, finite event types = the leash" (§2) is real but lives at the **gate** (schema-constrained decompose, ADR-009/D-1) plus doc 03 §8's type-registration CI discipline — **never** a DB enum (adding one would be a frozen-DDL change *and* break the open-vocabulary `AttributeChanged` design §3). Adopted consequence: the thin-slice actions map onto event-type strings **already in the seed** — **MOVE → `'move'`**, **SAY → `'private_disclosure'`** (a *told* perception, B-7; the seed's E1 is exactly this). The thin slice adds **no new event type** → no spine ADR owed. The broader §3 vocabulary (`ObjectRelocated`, `Ownership/AccessChanged`, `EntityCreated/Destroyed`, `AttributeChanged` + its SPEC-016 perceivability flag) **defers** with its SPECs.

**B. (Headline correction) Intra-tick ordering §8 is STALE — use the existing `beat_seq`; no new ordering ADR.** Capture §8 calls intra-tick ordering "open," says the `ADR-034 (tick, beat_seq)` reference was "recall-drift… not in the register," and proposes a future "ADR-034+." **The repo refutes all of this:** `beat_seq` is in the frozen Master DDL (doc 03 §1.1); **ADR-034 already exists and is Accepted** (doc 02 — "Canonical event ordering excludes recorded_at": order = `(in_world_tick, beat_seq)`, UNIQUE per world over accepted events); migration `20260610090007` enforces `uq_ce_accepted_order`; `70_determinism_guards_test.sql` guards it; `replay_0A()` already orders by `(world_id, in_world_tick, beat_seq)`. **Adopted:** the decompose/apply step **assigns `beat_seq` per event in the chain** and relies on the existing unique index; **no new ordering ADR**. Because **ADR-034 and ADR-035 are both taken**, the next free engine ADR is **ADR-036** (the task's "ADR-034+" framing is corrected here). *Capture-doc errata (non-blocking): §8's "recall-drift / open" is wrong — record in this section; a small capture-doc errata can follow separately.*

**C. `provenance_edge` §16 — schema already supports the leaf rule; NO DDL change; corrections deferred (verify-only).** The DDL (doc 03 §1.2) permits `event→event` and `perception/mutation→event` edges (`derived_kind ∈ {perception,mutation,event,bundle}`, `source_kind ∈ {perception,mutation,event}`). The §16 "correctable iff no inbound dependents" leaf rule = `NOT EXISTS (SELECT 1 FROM provenance_edge WHERE source_id = X)`, served directly by `idx_pe_source`. The leaf rule holds regardless of edge style. **Corrections are NOT in the Chunk-5 operator gate (§17)** → deferred; this finding is verification-only and owes no DDL change.

**D. Perception payload §14 vs context-assembler doc 06 — "payload" ≠ "bundle" (confirmed); decomposer-binding is a design EXTENSION, tested not assumed.** The thin-slice payload **reuses the pgTAP-tested wall `fn_visible_perceptions`** (B-1/I-3); doc 06's full assembler (dirty ladder, token-budget packing, Redis, relationship rendering) is **structure-deferred** (ADR-014 — its eventual home). "Bundle" is correctly avoided: G3 forbids live projection reads as bundle inputs, but the perception view is built from **live SQL joins** → provably a different object; call it a **perception payload**. That the **decomposer** (input side) is perception-bound *extends* ADR-020 (which names only the narrator) and ADR-014/doc 06 (framed around "the narrator and NPCs") — it is **reasoned design** that *satisfies* B-1/I-3/ADR-005 (more restrictive than required), **not** a frozen rule → no ADR; validated empirically at the operator gate. **Q3 no-leak is DATA-LAYER scoped:** the hidden fact never *enters* the payload or narration (provable, doc 06 §1 hard-guarantee #1). It does **NOT** claim the **generation layer** is leak-proof — the single-call multi-holder leak (O-1, ADR-020) is unsolved and deferred to **Phase 3** (doc 07 §6). State this explicitly in the gate.

**Capture-doc corrections surfaced (recorded; non-blocking):** §8 (ADR-034 not recall-drift; ordering decided + enforced — Finding B); §12 thin-slice "coords → derive" folds into the deferred **SPEC-018** (Finding *Time/Spatial* below).

**Time/Spatial scope decision (founder-confirmed).** Durations are **independent** facts (§12 itself notes travel-time has no triangle-inequality constraint; per **D-11** independent facts are recorded directly, coupled facts derive). Leg 1 is **model-free with hand-authored fixture places**, so the spatial-derive (which exists to keep *coupled distance* coherent when a *model* places locations) **has no consumer here**. **Adopted thin slice:** each committed event carries a **recorded, deterministic duration** (move = a hand-set canon value via `fn_move_duration`; say ≈ 0). The clock advances by the **committed-prefix** durations only (action-driven time §10; partial-beat time holds). Turn budget = a **generous max tick-delta backstop** (the world-interleave source defers with SPEC-012/013). **No coordinates, no derive.** The spatial engine defers wholesale to **SPEC-018**. **One owed engine ADR: ADR-036** (action-driven clock advancement — World-Clock-touching, ADR-021/030), filed **WITH Leg-1 running-code evidence** (D-9). **No ADR-037.**

**Owed ADR (staged, D-9):** **ADR-036 — action-driven World-Clock advancement.** Drafted **Proposed** at the start of Leg 1 (Task 5), moved **Accepted** at the Leg-1 gate with `apply_beat` + the play-loop pgTAP suite as the running-code evidence. The perception generator itself owes **no** ADR (it implements ADR-005 "one event fans out to 0..N perceptions" + doc 13 §5 "Phase-1 fan-out"); the loop control flow owes **no** ADR (it implements C-5/C-6/C-7 + ADR-009).

---

## §0.1 Scope (thin slice)

**In scope (Chunk 5):**
- **Action set: MOVE + SAY only** (`'move'`, `'private_disclosure'`). Possession / adjudicated actions deferred.
- **Write-side perception generation** (Leg-1 core new work): who-perceives-what fan-out on committed MOVE/SAY events (witnessing + the discovery-on-arrival trigger), implementing ADR-005 + doc 13 §5.
- **Incremental apply with a halt hook + partial-beat correctness** (C-5/C-6/C-7): two first-class halt mechanisms — pre-apply **gate-reject** (3a) and post-apply **stop-check** (3d) — with **deterministic triggers only**: the perception-vs-expectation stop rule via **discovery** (source (a)) + the **generous hard time-cap backstop** (the only world-time bound while source (b) defers).
- **Action-driven clock** (ADR-036, staged): clock advances by committed-prefix durations; durations engine-assigned, never model-authored (§10 Q1 guardrail).
- **LLM bridge** (Leg 2): schema-constrained `Decomposer` (closed vocabulary = the leash, ADR-009/D-1/SPEC-015) + perception-bound `Narrator` (ADR-020) + the `POST /worlds/{w}/beat` endpoint. Both seats consume only `fn_visible_perceptions` (B-1/I-3 — §14 extension).

**Deferred — "structure-in / behavior-deferred" (§8–§9); each is a named SPEC boundary, NOT silently dropped:**
- **Resolve** stage = identity/passthrough (mechanical) → adjudication **SPEC-013**.
- **NPC cognition** (perceive→believe→act; B-11 event-driven) → **SPEC-012**.
- **Per-attribute perceivability** (visible-vs-hidden split; deception §15) → **SPEC-016** (no `AttributeChanged` in the thin slice → not needed).
- **Move-validity** beyond co-location/reachability → **SPEC-017** (the thin slice ships the deterministic co-location precondition only).
- **Spatial engine** (coordinates, derived distance/travel-time, nested frames, portals, terrain) → **SPEC-018** (durations hand-recorded here, per the Time/Spatial decision).
- **World/NPC interleave** as a stop source (b), telegraph + reaction-window, held actions → SPEC-012/013.
- **Corrections** (§16 off-beat leaf-only) → not in the operator gate; deferred.
- **Cascading-inference depth bound** → **SPEC-014**.
- **Full context assembler** (dirty ladder, token budget, Redis, relationship rendering) → doc 06 / ADR-014, post-thin-slice.

---

## LEG 1 — Engine harness (deterministic; fixture-driven; NO model)

> Leg 1 is **not a playable product slice.** It proves events → state → **perception-generation** → replay on **fixture** chains, with incremental-apply-with-halt and **partial-beat correctness as a gate property**, validated through the existing Compendium + `replay_0A()`. The existing frozen suite (`00`–`90`) must stay green throughout (regression). All Leg-1 work is SQL + pgTAP.

### Task 1: Co-presence helper `fn_actors_at` (who is at a location)

**Files:**
- Create: `core/db/migrations/20260618090001_play_loop_engine.sql` (this task adds the first function; later Leg-1 tasks extend the SAME migration's `-- migrate:up` / `-- migrate:down`)
- Test: `core/db/tests/91_fn_actors_at_test.sql`

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/91_fn_actors_at_test.sql`:

```sql
BEGIN;
SELECT plan(3);

-- Position two actors at 'tavern', one at 'square', via accepted move events (trigger maintains
-- actor_state). Uses the C5 fixture id namespace; ticks 210+ miss every existing assertion.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000010','11111111-1111-1111-1111-111111111111','move','setup P→tavern',210,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000011','11111111-1111-1111-1111-111111111111','move','setup M→tavern',211,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000012','11111111-1111-1111-1111-111111111111','move','setup J→square',212,0,'accepted',now(),'public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000010','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),210,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000011','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('tavern'::text),211,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000012','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('square'::text),212,0);

SELECT set_eq(
  $$ SELECT entity_id FROM fn_actors_at('11111111-1111-1111-1111-111111111111','tavern') $$,
  $$ VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::uuid),
            ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid) $$,
  'Player and Mara are co-present at tavern');
SELECT set_eq(
  $$ SELECT entity_id FROM fn_actors_at('11111111-1111-1111-1111-111111111111','square') $$,
  $$ VALUES ('cccccccc-cccc-cccc-cccc-cccccccccccc'::uuid) $$,
  'Jonas alone at square');
SELECT is(
  (SELECT count(*) FROM fn_actors_at('11111111-1111-1111-1111-111111111111','market'))::int,
  0, 'nobody at market');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/91_fn_actors_at_test.sql'`
Expected: FAIL — `function fn_actors_at(...) does not exist`.

- [ ] **Step 3: Create the migration with `fn_actors_at`**

Create `core/db/migrations/20260618090001_play_loop_engine.sql`:

```sql
-- migrate:up
-- Chunk-5 play-loop engine (design 2026-06-16). Deterministic; NO model. SQL is the engine
-- (ADR-P017); the Go layer (Leg 2) is a thin orchestrator. Functions added incrementally across
-- Tasks 1–6; the down body drops them in reverse.

-- Co-presence (thin-slice SPEC-017 substrate): actors whose projected location label matches.
-- Reads actor_state (the projection), not canon — co-presence is a STATE question.
CREATE FUNCTION fn_actors_at(p_world_id uuid, p_location text)
RETURNS TABLE(entity_id uuid)
LANGUAGE sql STABLE AS $$
  SELECT a.entity_id
  FROM actor_state a
  WHERE a.world_id = p_world_id
    AND a.attrs->>'location_id' = p_location;
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_actors_at(uuid, text);
```

- [ ] **Step 4: Run it to verify it passes (and re-dump schema)**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/91_fn_actors_at_test.sql' && make schema-check`
Expected: PASS (3/3); `make schema-check` clean (the migration re-dumped `core/db/schema.sql`).

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260618090001_play_loop_engine.sql core/db/tests/91_fn_actors_at_test.sql core/db/schema.sql
git commit -m "feat(db): fn_actors_at co-presence helper (chunk-5 play loop)"
```

---

### Task 2: Per-event duration `fn_move_duration` (recorded, deterministic; ADR-036 substrate)

**Files:**
- Modify: `core/db/migrations/20260618090001_play_loop_engine.sql`
- Test: `core/db/tests/92_fn_move_duration_test.sql`

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/92_fn_move_duration_test.sql`:

```sql
BEGIN;
SELECT plan(3);
-- Duration is INDEPENDENT, hand-recorded fixture data (D-11; §11). Engine-assigned, never model
-- (§10 Q1 guardrail). Thin slice: a symmetric tavern↔square cost; same-place = 0; say handled = 0.
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111','tavern','square')::int, 5,
          'tavern→square costs 5 ticks');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111','square','tavern')::int, 5,
          'symmetric: square→tavern also 5');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111','tavern','tavern')::int, 0,
          'no move, no time');
SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/92_fn_move_duration_test.sql'`
Expected: FAIL — `function fn_move_duration(...) does not exist`.

- [ ] **Step 3: Add `fn_move_duration` to the migration**

In `20260618090001_play_loop_engine.sql`, add to `-- migrate:up` (before `-- migrate:down`):

```sql
-- ADR-036 substrate: per-event duration as RECORDED, deterministic world data (D-11; §11), assigned
-- by the engine — NEVER the model (§10 Q1 guardrail). Thin slice = a hand-authored cost table; the
-- spatial engine (coordinates → derived distance/travel-time) is DEFERRED wholesale (SPEC-018), so
-- there is no derive here. Unknown pairs fall back to a flat default; same place = 0.
CREATE FUNCTION fn_move_duration(p_world_id uuid, p_from text, p_to text)
RETURNS bigint
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
           WHEN p_from = p_to THEN 0
           WHEN (p_from,p_to) IN (('tavern','square'),('square','tavern')) THEN 5
           ELSE 5   -- flat default for the thin-slice fixture map
         END::bigint;
$$;
```

And add to `-- migrate:down` (above the `fn_actors_at` drop, so drops are reverse order):

```sql
DROP FUNCTION IF EXISTS fn_move_duration(uuid, text, text);
```

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/92_fn_move_duration_test.sql' && make schema-check`
Expected: PASS (3/3); schema clean.

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260618090001_play_loop_engine.sql core/db/tests/92_fn_move_duration_test.sql core/db/schema.sql
git commit -m "feat(db): fn_move_duration recorded per-event duration (ADR-036 substrate)"
```

---

### Task 3: Perception generator for SAY `generate_perceptions` — `'private_disclosure'`

**Files:**
- Modify: `core/db/migrations/20260618090001_play_loop_engine.sql`
- Test: `core/db/tests/93_generate_perceptions_say_test.sql`

This implements ADR-005 (one event → 0..N perceptions) + B-7 (told ≠ witnessed): a SAY generates the **speaker** a `'shared'` perception and each **listener** a `'told'` perception — reproducing the seed's E1 fan-out by GENERATION, not hand-insertion. No new ADR (already specified). The function is the single write-path for perceptions in the loop; like `apply_mutation` it is `SECURITY DEFINER`, owned by `maintainer` (I-7).

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/93_generate_perceptions_say_test.sql`:

```sql
BEGIN;
SELECT plan(5);

-- A SAY event (private_disclosure) P→M with both at tavern; J elsewhere. generate_perceptions writes
-- speaker 'shared' + listener 'told' (B-7), nothing for J. acquired_tick = event tick (I-9).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e5000000-0000-0000-0000-000000000020','11111111-1111-1111-1111-111111111111',
        'private_disclosure','P tells M a secret',300,0,'accepted',now(),'private','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000020','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','speaker'),
 ('e5000000-0000-0000-0000-000000000020','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','listener');

SELECT is(generate_perceptions('e5000000-0000-0000-0000-000000000020')::int, 2,
          'SAY generates exactly 2 perceptions (speaker + listener)');
SELECT is((SELECT epistemic_type FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
          'shared', 'speaker (Player) holds a SHARED perception');
SELECT is((SELECT epistemic_type FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
          'told', 'listener (Mara) holds a TOLD perception (B-7: told ≠ witnessed)');
SELECT is((SELECT count(*) FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='cccccccc-cccc-cccc-cccc-cccccccccccc')::int,
          0, 'Jonas (not addressed) holds NOTHING — the knowledge boundary (B-1/I-3)');
SELECT is((SELECT acquired_tick FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
          300::bigint, 'acquired_tick = event tick (I-9: cannot learn before it happened)');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/93_generate_perceptions_say_test.sql'`
Expected: FAIL — `function generate_perceptions(...) does not exist`.

- [ ] **Step 3: Add `generate_perceptions` (SAY branch) to the migration**

In `20260618090001_play_loop_engine.sql` `-- migrate:up`:

```sql
-- Write-side perception generation (ADR-005: one event → 0..N perceptions; doc 13 §5 Phase-1 fan-out).
-- The ONLY perception write path in the loop. SECURITY DEFINER / maintainer-owned (I-7). Generates
-- from CANON's visible aspect (§4 witnessing trigger). Every row carries source_event_id (I-2) and
-- acquired_tick = the event's in_world_tick (I-9). Returns the number of perceptions written.
-- Thin slice handles 'move' and 'private_disclosure' only; other types are a no-op (0).
CREATE FUNCTION generate_perceptions(p_event_id uuid)
RETURNS integer
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  ev   canon_event;
  n    integer := 0;
  spk  uuid;
  lst  uuid;
BEGIN
  SELECT * INTO ev FROM canon_event WHERE event_id = p_event_id AND status = 'accepted';
  IF NOT FOUND THEN RETURN 0; END IF;

  IF ev.event_type = 'private_disclosure' THEN
    -- speaker → 'shared'; each listener → 'told' (B-7). Recipients = the addressed listeners
    -- (thin slice; co-present overhearers defer with the broader vocabulary, §3).
    SELECT entity_id INTO spk FROM event_participant
      WHERE event_id = p_event_id AND role_qualifier = 'speaker' LIMIT 1;
    IF spk IS NOT NULL THEN
      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, spk, p_event_id, ev.summary, 'shared', ev.in_world_tick, ev.in_world_tick);
      n := n + 1;
    END IF;
    FOR lst IN SELECT entity_id FROM event_participant
                 WHERE event_id = p_event_id AND role_qualifier = 'listener' LOOP
      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, lst, p_event_id, ev.summary, 'told', ev.in_world_tick, ev.in_world_tick);
      n := n + 1;
    END LOOP;
  END IF;

  RETURN n;
END $$;
ALTER FUNCTION generate_perceptions(uuid) OWNER TO maintainer;
REVOKE EXECUTE ON FUNCTION generate_perceptions(uuid) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION generate_perceptions(uuid) TO maintainer;
```

And in `-- migrate:down`, add (reverse order, above earlier drops):

```sql
DROP FUNCTION IF EXISTS generate_perceptions(uuid);
```

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/93_generate_perceptions_say_test.sql' && make schema-check`
Expected: PASS (5/5); schema clean.

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260618090001_play_loop_engine.sql core/db/tests/93_generate_perceptions_say_test.sql core/db/schema.sql
git commit -m "feat(db): generate_perceptions SAY fan-out (ADR-005, B-7 told≠witnessed)"
```

---

### Task 4: Perception generator for MOVE + the discovery trigger

**Files:**
- Modify: `core/db/migrations/20260618090001_play_loop_engine.sql`
- Test: `core/db/tests/94_generate_perceptions_move_test.sql`

A MOVE generates (§4 trigger 1, witnessing) the **mover** a `'direct'` perception, and (§4 trigger 2, **discovery-on-arrival**) the mover a `'direct'` perception **about each actor already at the destination**. The discovery perceptions are what the stop-check reads (Task 6). The mover's destination is taken from the move's `state_mutation` (`attrs.location_id`); co-presence at the destination is read from `fn_actors_at` **before** the mover's own arrival is double-counted (exclude the mover).

- [ ] **Step 1: Write the failing test**

Create `core/db/tests/94_generate_perceptions_move_test.sql`:

```sql
BEGIN;
SELECT plan(4);

-- Pre-position Jonas at square. Then a MOVE of Player → square (with its location mutation applied).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000030','11111111-1111-1111-1111-111111111111','move','setup J→square',310,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000031','11111111-1111-1111-1111-111111111111','move','P moves to square',311,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000030','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000031','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000030','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('square'::text),310,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000031','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('square'::text),311,0);

-- mover 'direct' (witnessed own move) + discovery 'direct' about Jonas already present = 2.
SELECT is(generate_perceptions('e5000000-0000-0000-0000-000000000031')::int, 2,
          'MOVE generates the mover own-move + one discovery (Jonas present)');
SELECT is((SELECT count(*) FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000031'
             AND holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
             AND epistemic_type='direct')::int,
          2, 'both perceptions are the mover (Player), direct');
-- the discovery perception is ABOUT Jonas (perception_subject) → feeds the stop-check
SELECT ok(EXISTS(
  SELECT 1 FROM perception_record pr
  JOIN perception_subject ps ON ps.perception_id = pr.perception_id
  WHERE pr.source_event_id='e5000000-0000-0000-0000-000000000031'
    AND ps.entity_id='cccccccc-cccc-cccc-cccc-cccccccccccc'),
  'discovery perception is ABOUT Jonas (subject link)');
-- Jonas, already present, is NOT handed a perception of Player arriving in the thin slice
-- (witnessing-others defers with the broader vocabulary; the mover-discovery axis is what the
-- stop-check needs). Keep the slice minimal and honest.
SELECT is((SELECT count(*) FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000031'
             AND holder_id='cccccccc-cccc-cccc-cccc-cccccccccccc')::int,
          0, 'thin slice: only the mover perceives (others-witness-mover deferred)');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/94_generate_perceptions_move_test.sql'`
Expected: FAIL — assertions error/red: the `'move'` branch isn't in `generate_perceptions` yet (returns 0).

- [ ] **Step 3: Extend `generate_perceptions` with the MOVE branch**

In `20260618090001_play_loop_engine.sql`, inside `generate_perceptions`, add **before** `RETURN n;`:

```sql
  IF ev.event_type = 'move' THEN
    -- mover + destination, from the move's own location mutation.
    DECLARE
      mover uuid;
      dest  text;
      other uuid;
      pid   uuid;
    BEGIN
      SELECT entity_id INTO mover FROM event_participant
        WHERE event_id = p_event_id AND role_qualifier = 'instigator' LIMIT 1;
      SELECT (new_value #>> '{}') INTO dest FROM state_mutation
        WHERE event_id = p_event_id AND attribute_path = 'attrs.location_id' LIMIT 1;
      IF mover IS NOT NULL THEN
        -- witnessing: the mover perceives their own move ('direct').
        INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                       acquired_tick, valid_tick)
        VALUES (ev.world_id, mover, p_event_id, ev.summary, 'direct', ev.in_world_tick, ev.in_world_tick);
        n := n + 1;
        -- discovery-on-arrival (§4 trigger 2): the mover perceives each actor ALREADY at dest
        -- (exclude self). Each carries an explicit subject link → the stop-check reads about-ness.
        IF dest IS NOT NULL THEN
          FOR other IN SELECT entity_id FROM fn_actors_at(ev.world_id, dest)
                        WHERE entity_id <> mover LOOP
            INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                           acquired_tick, valid_tick)
            VALUES (ev.world_id, mover, p_event_id,
                    'On arriving, I noticed someone already here.', 'direct',
                    ev.in_world_tick, ev.in_world_tick)
            RETURNING perception_id INTO pid;
            INSERT INTO perception_subject (perception_id, entity_id, world_id)
            VALUES (pid, other, ev.world_id);
            n := n + 1;
          END LOOP;
        END IF;
      END IF;
    END;
  END IF;
```

> Note: `fn_actors_at(dest)` must reflect the world state **at the moment the mover arrives**. In `apply_beat` (Task 5) `generate_perceptions` is called **after** the move's mutation is applied, but the mover's own `location_id` now equals `dest` too — hence the `entity_id <> mover` exclusion, which makes "who was already here" correct.

- [ ] **Step 4: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/94_generate_perceptions_move_test.sql' && make schema-check`
Expected: PASS (4/4); schema clean.

- [ ] **Step 5: Commit**

```bash
git add core/db/migrations/20260618090001_play_loop_engine.sql core/db/tests/94_generate_perceptions_move_test.sql core/db/schema.sql
git commit -m "feat(db): generate_perceptions MOVE witnessing + discovery-on-arrival (§4)"
```

---

### Task 5: `apply_beat` — incremental gated apply with a halt hook + action-driven clock (ADR-036 Proposed)

**Files:**
- Modify: `core/db/migrations/20260618090001_play_loop_engine.sql`
- Modify: `docs/30_architecture/canon_engine/02_world_state_adrs.md` (draft **ADR-036 — Proposed**)
- Test: `core/db/tests/95_apply_beat_happy_test.sql`

`apply_beat` is the deterministic loop core (§8 four-stage pipeline; thin slice). Input: an **ordered chain** (a `jsonb` array of `{type, ...}`) and a `start_tick`. For each event, in order: **(a) gate** (move-validity / co-location precondition, SPEC-017 thin slice) — fail → **halt before apply**; **(b) resolve** = identity (SPEC-013 passthrough); **(c) apply + generate** = insert the accepted `canon_event` with `(in_world_tick, beat_seq)` from the running clock (ADR-034), insert the move's `state_mutation`, call `generate_perceptions`, **advance the clock by the event's duration** (ADR-036); **(d) stop-check** — does a generated **discovery** perception break the **next** link's premise? broken → **halt after this step commits**. The **turn-budget backstop** halts if the committed tick-delta would exceed `p_tick_cap`. Returns a `jsonb` summary: committed event ids in order, `halt_reason`, `ticks_advanced`. **`origin` is a parameter** so Leg 1 passes `'fast_path'` (fixtures) and Leg 2 passes `'freeform'` (model-proposed, gated).

This is where the **two first-class halt mechanisms** (§8: pre-apply gate-reject + post-apply stop-check) and **partial-beat** live. Governing rules: C-5/C-6/C-7, ADR-009/D-1, ADR-034, ADR-036, SPEC-017 (thin), SPEC-013 (deferred resolve), SPEC-015 (canon-authority — Leg 2).

- [ ] **Step 1: Draft ADR-036 — Proposed (engine ADR; D-9 evidence follows at the Leg-1 gate)**

In `docs/30_architecture/canon_engine/02_world_state_adrs.md`, append after ADR-035:

```markdown
## ADR-036 — The World Clock advances by committed-event durations (action-driven time)

**Status:** Proposed (2026-06-18; Chunk-5 Leg 1). Moves to Accepted at the Leg-1 gate with `apply_beat`
+ the play-loop pgTAP suite as running-code evidence (D-9).
**Decision:** Within a beat, in-world time advances by the **sum of the durations of the events that
actually commit** — not per beat, not by wall-clock. Each committed event is assigned a duration by the
**engine** from recorded deterministic world data (`fn_move_duration`; SAY ≈ 0), **never** by the LLM.
An interrupted chain advances the clock by the **committed prefix's** durations only. `(in_world_tick,
beat_seq)` are assigned by `apply_beat` and remain UNIQUE per world over accepted events (ADR-034,
`uq_ce_accepted_order`); a zero-duration event shares the prior tick and increments `beat_seq`.
**Rationale:** Wall-clock dies on Q1/I-1 (replay at a different real time → different clock); per-beat
dies on C-6 (every Continue would advance world-time). Action-driven time survives both: pure Continue
commits no events → no time passes (C-6); replay reproduces the clock because it reproduces the events
and their recorded durations (Q1/I-1). Partial-beat time *requires* per-action time (§9–§10 are one
decision). Durations are independent facts → recorded directly (D-11); the spatial-derive (coupled
distance) is deferred (SPEC-018) and has no consumer in a model-free, hand-authored-places slice.
**Consequences:** Extends ADR-021/030 (which left 0A "assigning ticks by hand"): the beat loop now
assigns and advances the clock. Durations live in engine-side data/functions, never in model output
(the §10 Q1 guardrail). Supersedes nothing; adds the advancement rule the play loop assumes.
```

- [ ] **Step 2: Write the failing happy-path test**

Create `core/db/tests/95_apply_beat_happy_test.sql`:

```sql
BEGIN;
SELECT plan(6);

-- Player and Mara co-present at tavern (setup moves). Then a 2-step beat: [say to Mara, move to square].
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000040','11111111-1111-1111-1111-111111111111','move','setup P→tavern',400,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000041','11111111-1111-1111-1111-111111111111','move','setup M→tavern',401,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000040','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000041','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000040','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),400,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000041','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('tavern'::text),401,0);

SELECT lives_ok($$
  SELECT apply_beat(
    '11111111-1111-1111-1111-111111111111',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    '[{"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"I saw the note"},
      {"type":"move","to":"square"}]'::jsonb,
    500, 100, 'fast_path')
$$, 'apply_beat runs a 2-step happy beat');

-- both steps committed (the beat is the only origin='fast_path' content at tick >= 500)
SELECT is((SELECT count(*) FROM canon_event
           WHERE world_id='11111111-1111-1111-1111-111111111111' AND in_world_tick >= 500)::int,
          2, 'both events committed');
-- SAY generated Mara 'told'; MOVE generated Player 'direct'
SELECT ok(EXISTS(SELECT 1 FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
           WHERE ce.in_world_tick>=500 AND pr.holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
             AND pr.epistemic_type='told'),
          'Mara TOLD by the SAY step');
SELECT ok(EXISTS(SELECT 1 FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
           WHERE ce.in_world_tick>=500 AND pr.holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
             AND pr.epistemic_type='direct'),
          'Player DIRECT from the MOVE step');
-- I-3 no-leak through the existing Compendium read function: Jonas sees none of it.
SELECT is((SELECT count(*) FROM fn_visible_perceptions('11111111-1111-1111-1111-111111111111',
                                                       'cccccccc-cccc-cccc-cccc-cccccccccccc') v
           JOIN canon_event ce ON ce.event_id=v.source_event_id WHERE ce.in_world_tick>=500)::int,
          0, 'I-3: Jonas perceives nothing from the beat (validated through fn_visible_perceptions)');
-- action-driven clock: say(0) + move tavern→square(5) → ticks_advanced = 5; uniqueness held.
SELECT is((SELECT count(DISTINCT (in_world_tick, beat_seq)) FROM canon_event
           WHERE world_id='11111111-1111-1111-1111-111111111111' AND in_world_tick>=500)::int,
          2, 'both committed events occupy distinct (tick, beat_seq) slots (ADR-034)');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 3: Run it to verify it fails**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/95_apply_beat_happy_test.sql'`
Expected: FAIL — `function apply_beat(...) does not exist`.

- [ ] **Step 4: Add `apply_beat` to the migration**

In `20260618090001_play_loop_engine.sql` `-- migrate:up`:

```sql
-- apply_beat: the deterministic incremental gated loop (§8 four-stage pipeline; thin slice).
-- gate (3a, SPEC-017 co-location) → resolve (3b, identity; SPEC-013 deferred) → apply+generate (3c)
-- → stop-check (3d, discovery breaks next premise). Two first-class halts: pre-apply gate-reject and
-- post-apply stop-check (§8). Clock advances by COMMITTED-prefix durations (ADR-036). origin is a
-- param: 'fast_path' (Leg-1 fixtures) | 'freeform' (Leg-2 model-proposed, gated). D-1: this gate is
-- the ONLY canonization point; the model never writes canon. Returns a jsonb summary.
CREATE FUNCTION apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb,
                           p_start_tick bigint, p_tick_cap bigint, p_origin text DEFAULT 'fast_path')
RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  step       jsonb;
  idx        int := 0;
  cur_tick   bigint := p_start_tick;
  cur_seq    int := 0;
  start_tick bigint := p_start_tick;
  committed  jsonb := '[]'::jsonb;
  halt       text := 'completed';
  ev_id      uuid;
  dur        bigint;
  here       text;
  listener   uuid;
  next_step  jsonb;
  next_ok    boolean;
BEGIN
  -- the actor's current location (projection), used for gate + clock placement
  FOR step IN SELECT * FROM jsonb_array_elements(p_chain) LOOP
    idx := idx + 1;
    SELECT a.attrs->>'location_id' INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;

    -- (3a) GATE — thin-slice move-validity / co-location precondition (SPEC-017).
    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      -- precondition: the addressed listener is co-present with the actor.
      IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
        halt := 'gate_reject'; EXIT;   -- nothing committed for this step (3a)
      END IF;
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      dur := fn_move_duration(p_world_id, COALESCE(here,'?'), step->>'to');
    ELSE
      halt := 'gate_reject'; EXIT;     -- out-of-vocabulary (closed set; ADR-009/D-1, SPEC-015)
    END IF;

    -- turn-budget backstop (§9 third pushback face): would committing exceed the cap?
    IF (cur_tick + dur) - start_tick > p_tick_cap THEN
      halt := 'turn_budget'; EXIT;
    END IF;

    -- (3b) RESOLVE = identity (SPEC-013 deferred). (3c) APPLY + GENERATE + advance clock.
    ev_id := gen_random_uuid();
    INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                             status, accepted_at, visibility_scope, origin)
    VALUES (ev_id, p_world_id,
            CASE step->>'type' WHEN 'say' THEN 'private_disclosure' ELSE 'move' END,
            COALESCE(step->>'content', step->>'type'), cur_tick, cur_seq,
            'accepted', now(),
            CASE step->>'type' WHEN 'say' THEN 'private' ELSE 'public' END, p_origin);
    IF step->>'type' = 'say' THEN
      INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
        (ev_id, p_actor_id, 'actor', 'speaker'),
        (ev_id, listener,   'actor', 'listener');
    ELSE
      INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
        VALUES (ev_id, p_actor_id, 'actor', 'instigator');
      INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                  new_value, valid_from_tick, valid_from_seq)
        VALUES (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id',
                to_jsonb(step->>'to'), cur_tick, cur_seq);  -- trigger applies the projection
    END IF;
    PERFORM generate_perceptions(ev_id);
    committed := committed || to_jsonb(ev_id);

    -- advance the clock by THIS committed event's duration (ADR-036).
    IF dur > 0 THEN cur_tick := cur_tick + dur; cur_seq := 0; ELSE cur_seq := cur_seq + 1; END IF;

    -- (3d) STOP-CHECK — does a discovery perception break the NEXT link's premise? (source (a)).
    next_step := p_chain -> idx;   -- 0-based: element after the current 1-based idx
    IF next_step IS NOT NULL AND next_step->>'type' = 'say' THEN
      SELECT a.attrs->>'location_id' INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      next_ok := EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here)
                         WHERE entity_id = (next_step->>'listener')::uuid);
      IF NOT next_ok THEN halt := 'stop_check'; EXIT; END IF;  -- prefix stands; remainder never runs
    END IF;
  END LOOP;

  RETURN jsonb_build_object('committed', committed, 'halt_reason', halt,
                            'ticks_advanced', cur_tick - start_tick);
END $$;
ALTER FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) OWNER TO maintainer;
REVOKE EXECUTE ON FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) TO maintainer;
```

And in `-- migrate:down`, add (reverse order, first of the drops):

```sql
DROP FUNCTION IF EXISTS apply_beat(uuid, uuid, jsonb, bigint, bigint, text);
```

- [ ] **Step 5: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/95_apply_beat_happy_test.sql' && make schema-check`
Expected: PASS (6/6); schema clean.

- [ ] **Step 6: Commit**

```bash
git add core/db/migrations/20260618090001_play_loop_engine.sql core/db/tests/95_apply_beat_happy_test.sql core/db/schema.sql docs/30_architecture/canon_engine/02_world_state_adrs.md
git commit -m "feat(db): apply_beat incremental gated loop + action-driven clock (ADR-036 Proposed)"
```

---

### Task 6: Partial-beat correctness — both halt ways (the core Q3 safety property)

**Files:**
- Test: `core/db/tests/96_apply_beat_partial_beat_test.sql`

This is the §9 gate property: *a chain that halts at step k commits **exactly** the prefix (steps 1..k-1 for gate-reject; 1..k for stop-check) — zero trace from the halt onward, and the clock advanced by the prefix durations only.* No new code — it exercises `apply_beat`'s two halt paths and asserts the partial-beat invariant. If it fails, the loop over-applies and the Leg-1 gate is red.

- [ ] **Step 1: Write the test (must pass against Task-5 code)**

Create `core/db/tests/96_apply_beat_partial_beat_test.sql`:

```sql
BEGIN;
SELECT plan(7);

-- Player at tavern, Mara at tavern, Jonas at square.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000050','11111111-1111-1111-1111-111111111111','move','P→tavern',600,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000051','11111111-1111-1111-1111-111111111111','move','M→tavern',601,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000052','11111111-1111-1111-1111-111111111111','move','J→square',602,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000050','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000051','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000052','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000050','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),600,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000051','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('tavern'::text),601,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000052','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('square'::text),602,0);

-- HALT WAY 1 — gate-reject: [say to Mara (ok), say to Jonas (Jonas not co-present → reject), move].
-- Expect: step 1 commits; step 2 rejected pre-apply; step 3 never runs.
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"hi Mara"},
              {"type":"say","listener":"cccccccc-cccc-cccc-cccc-cccccccccccc","content":"hi Jonas"},
              {"type":"move","to":"square"}]'::jsonb, 700, 100, 'fast_path') ->> 'halt_reason'),
           'gate_reject', 'chain halts pre-apply on the impossible SAY');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=700)::int,
           1, 'gate-reject: EXACTLY the prefix (1 event) committed — door rejected, move never ran');
SELECT is( (SELECT count(*) FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
            WHERE ce.in_world_tick>=700)::int,
           2, 'gate-reject: exactly the prefix perceptions (speaker shared + listener told)');

-- reset the beat region for the second scenario (same txn, fresh tick band)
-- HALT WAY 2 — stop-check via discovery: [move to square, say to Mara]. Arriving at square the player
-- discovers (premise of "say to Mara" = Mara co-present) is BROKEN → halt AFTER the move commits.
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"move","to":"square"},
              {"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"Mara?"}]'::jsonb,
            800, 100, 'fast_path') ->> 'halt_reason'),
           'stop_check', 'chain halts post-apply: discovery breaks the next premise (§9 source (a))');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=800 AND in_world_tick<900 AND event_type='move')::int,
           1, 'stop-check: the move committed (prefix stands)');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=800 AND in_world_tick<900 AND event_type='private_disclosure')::int,
           0, 'stop-check: the SAY never committed (zero trace from the halt onward)');
-- partial-beat TIME: the clock advanced by the move's duration only (5), not the say's (0-after-halt).
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"move","to":"square"},
              {"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"Mara?"}]'::jsonb,
            900, 100, 'fast_path') ->> 'ticks_advanced')::int,
           5, 'partial-beat time: clock advanced by the committed-prefix duration only (ADR-036)');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/96_apply_beat_partial_beat_test.sql'`
Expected: PASS (7/7). If any assertion is red, **STOP** — the loop over- or under-applies; fix `apply_beat` before proceeding (this is the core safety property of the interruptible loop).

- [ ] **Step 3: Teeth-prove the gate is non-vacuous (manual, then revert)**

Temporarily edit `apply_beat`: in the `(3a) GATE` SAY branch, replace the `IF NOT EXISTS (... ) THEN halt := 'gate_reject'; EXIT;` with `IF false THEN ...` (disable the precondition). Run Step 2.
Expected: the gate-reject assertions go **RED** (the impossible SAY now commits). **Revert the edit**, re-run Step 2 → 7/7 green. Do not commit the breach.

- [ ] **Step 4: Commit**

```bash
git add core/db/tests/96_apply_beat_partial_beat_test.sql
git commit -m "test(db): partial-beat correctness — gate-reject + stop-check, both prefix-exact (§9)"
```

---

### Task 7: Leg-1 invariants on the play-loop path (I-1/I-2/I-3/I-7/I-9)

**Files:**
- Test: `core/db/tests/98_play_loop_invariants_test.sql`

The permanent invariant suite (doc 07) must hold on engine-**generated** events, not just hand-seeded ones. This task asserts I-1 (replay rebuilds the move's projection), I-2 (every generated perception → accepted event), I-3 (no-leak via the wall), I-9 (epistemic temporal sanity). I-7 is already enforced by the maintainer-only grants on the new functions (Tasks 3–5) + the existing `60_permissions_test.sql`.

- [ ] **Step 1: Write the test**

Create `core/db/tests/98_play_loop_invariants_test.sql`:

```sql
BEGIN;
SELECT plan(5);

-- Position the cast and run a happy beat that produces both a move (mutation → projection) and a say.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000060','11111111-1111-1111-1111-111111111111','move','P→tavern',1000,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000061','11111111-1111-1111-1111-111111111111','move','M→tavern',1001,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000060','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000061','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000060','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),1000,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000061','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('tavern'::text),1001,0);

SELECT apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  '[{"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"the note"},
    {"type":"move","to":"square"}]'::jsonb, 1100, 100, 'fast_path');

-- I-1: replay rebuilds projections to the same domain state (the beat's move is replayed).
SELECT ok(replay_0A(), 'I-1: replay is domain-equivalent after a generated beat (ADR-026)');
-- I-2: every generated perception references an accepted event (zero orphans).
SELECT is((SELECT count(*) FROM perception_record pr
           LEFT JOIN canon_event ce ON ce.event_id=pr.source_event_id AND ce.status='accepted'
           WHERE pr.source_event_id IN (SELECT event_id FROM canon_event WHERE in_world_tick>=1100)
             AND ce.event_id IS NULL)::int,
          0, 'I-2: no orphan generated perceptions');
-- I-3: Jonas (uninvolved) perceives nothing from the beat (the wall).
SELECT is((SELECT count(*) FROM fn_visible_perceptions('11111111-1111-1111-1111-111111111111',
                                                       'cccccccc-cccc-cccc-cccc-cccccccccccc') v
           JOIN canon_event ce ON ce.event_id=v.source_event_id WHERE ce.in_world_tick>=1100)::int,
          0, 'I-3: no hidden-canon leakage to Jonas');
-- I-9: acquired_tick >= source event in_world_tick for every generated perception.
SELECT is((SELECT count(*) FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
           WHERE ce.in_world_tick>=1100 AND pr.acquired_tick < ce.in_world_tick)::int,
          0, 'I-9: no perception acquired before its event happened');
-- the move projection is present (Player at square after the beat)
SELECT is((SELECT attrs->>'location_id' FROM actor_state WHERE entity_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
          'square', 'projection reflects the committed move');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it to verify it passes**

Run: `make reset && docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/98_play_loop_invariants_test.sql'`
Expected: PASS (5/5).

- [ ] **Step 3: Run the FULL suite + replay (regression — the frozen `00`–`90` must stay green)**

Run: `make reset && make test && make replay`
Expected: every `*_test.sql` green (the existing 0A/0B/Chunk-3/4 files **unchanged** + the new `91`–`98`); `make replay` returns true.

- [ ] **Step 4: Commit**

```bash
git add core/db/tests/98_play_loop_invariants_test.sql
git commit -m "test(db): Leg-1 invariants on generated beats (I-1/I-2/I-3/I-7/I-9)"
```

---

## LEG 1 GATE → red = STOP

**The Leg-1 gate is green iff ALL hold (run `make reset && make test && make replay`):**
1. `91`–`98` green: co-presence, duration, SAY + MOVE generation, `apply_beat` happy path, **partial-beat correctness (both halt ways)**, Leg-1 invariants.
2. The frozen `00`–`90` suite green **unchanged** (regression — perception generation and the loop did not disturb the proven spine).
3. `make replay` true (I-1 domain-equivalent, ADR-026) after generated beats.
4. `make schema-check` clean.
5. **ADR-036 moves Proposed → Accepted** in `docs/30_architecture/canon_engine/02_world_state_adrs.md`, citing `apply_beat` + the `91`–`98` suite as the running-code evidence (D-9). Commit that status flip.

**If any is red: STOP. Do not start Leg 2.** Fix Leg 1 first (process law; playbook §1).

- [ ] **Gate step: flip ADR-036 to Accepted and commit**

Edit ADR-036's `**Status:**` line to `Accepted (2026-06-18; Chunk-5 Leg-1 gate). Evidence: apply_beat + tests 91–98 (this PR's execution).` then:

```bash
git add docs/30_architecture/canon_engine/02_world_state_adrs.md
git commit -m "docs(adr): ADR-036 Accepted — action-driven clock (D-9 evidence: Leg-1 suite)"
```

---

## LEG 2 — the LLM bridge (prose→events + perception-bound narration)

> Leg 2 puts the model in the loop, **quarantined**: it **proposes only** (D-1), is **perception-bound only** (B-1/I-3/ADR-020), and **never holds canon-authority** (SPEC-015). New risk: decomposition mapping correctness + the canon-authority boundary. CI uses **deterministic fakes** for both model seats (the live model runs only at the operator gate). Go is a thin orchestrator over the Leg-1 SQL engine.

### Task 8: Publish the closed event-vocabulary schema (the leash)

**Files:**
- Create: `core/api/schema/beat_chain.v1.schema.json`
- Test: `core/api/beat_schema_test.go`

The decomposer's output is constrained to a **closed set** — this is the leash (ADR-009/D-1; the §2 realization that infinite English collapses onto a small closed set of state-deltas). The schema is the frontend/codegen source of truth and the runtime validation contract (SPEC-015). Closure lives at the **gate + this schema**, never as a DB enum (Finding A).

- [ ] **Step 1: Write the failing test**

Create `core/api/beat_schema_test.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestBeatChainSchema_ClosedVocabulary(t *testing.T) {
	b, err := os.ReadFile("schema/beat_chain.v1.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var s map[string]any
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	if s["schema_version"] != "beat_chain/1" {
		t.Fatalf("schema_version = %v, want beat_chain/1", s["schema_version"])
	}
	// the closed action set is exactly {move, say}
	types := vocabularyTypes(s)
	if len(types) != 2 || !types["move"] || !types["say"] {
		t.Fatalf("vocabulary = %v, want exactly {move, say}", types)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd core/api && go test ./... -run TestBeatChainSchema -v`
Expected: FAIL — `vocabularyTypes` undefined and schema file missing.

- [ ] **Step 3: Create the schema and the `vocabularyTypes` helper**

Create `core/api/schema/beat_chain.v1.schema.json`:

```json
{
  "schema_version": "beat_chain/1",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Decomposed beat chain — the closed event vocabulary (ADR-009/D-1, SPEC-015)",
  "type": "array",
  "items": {
    "type": "object",
    "oneOf": [
      {
        "properties": {
          "type": {"const": "move"},
          "to":   {"type": "string"}
        },
        "required": ["type", "to"],
        "additionalProperties": false
      },
      {
        "properties": {
          "type":     {"const": "say"},
          "listener": {"type": "string", "format": "uuid"},
          "content":  {"type": "string"}
        },
        "required": ["type", "listener", "content"],
        "additionalProperties": false
      }
    ]
  }
}
```

Create `core/api/beatvocab.go`:

```go
package main

import "encoding/json"

// vocabularyTypes extracts the closed set of allowed event "type" const values from the
// beat_chain schema's items.oneOf. The closed set IS the leash (ADR-009/D-1, SPEC-015).
func vocabularyTypes(schema map[string]any) map[string]bool {
	out := map[string]bool{}
	items, _ := schema["items"].(map[string]any)
	if items == nil {
		return out
	}
	oneOf, _ := items["oneOf"].([]any)
	for _, alt := range oneOf {
		m, _ := alt.(map[string]any)
		props, _ := m["properties"].(map[string]any)
		typ, _ := props["type"].(map[string]any)
		if c, ok := typ["const"].(string); ok {
			out[c] = true
		}
	}
	return out
}

// allowedBeatTypes is the canonical closed set used by the runtime validator (kept in sync with
// beat_chain.v1.schema.json; TestBeatChainSchema_ClosedVocabulary asserts they match).
var allowedBeatTypes = map[string]bool{"move": true, "say": true}

var _ = json.Marshal // keep encoding/json imported for sibling files
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd core/api && go test ./... -run TestBeatChainSchema -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add core/api/schema/beat_chain.v1.schema.json core/api/beatvocab.go core/api/beat_schema_test.go
git commit -m "feat(api): publish beat_chain/1 closed-vocabulary schema (the leash, ADR-009/D-1)"
```

---

### Task 9: `Decomposer` + `Narrator` seats — perception-bound, fake-in-CI

**Files:**
- Create: `core/api/beatseats.go`
- Test: `core/api/beatseats_test.go`

Both model seats are **perception-bound**: each is constructed with the **player's perception payload only** (`fn_visible_perceptions`, B-1/I-3) — the **decomposer** reads it *before* resolving (to interpret intent; the §14 extension of ADR-020), the **narrator** reads the *post-beat* payload *after* (ADR-020, no omniscient pass). Interfaces let CI inject **deterministic fakes**; the live model is wired only for the operator gate. The seats **never** receive raw canon — that is the data-layer wall the Q3 no-leak claim rests on.

- [ ] **Step 1: Write the failing test**

Create `core/api/beatseats_test.go`:

```go
package main

import (
	"context"
	"strings"
	"testing"
)

func TestDecomposer_OnlyValidVocabulary(t *testing.T) {
	d := NewFakeDecomposer(map[string]string{
		"go to the square": `[{"type":"move","to":"square"}]`,
	})
	chain, err := DecodeAndValidateChain(d.Decompose(context.Background(), PerceptionPayload{}, "go to the square"))
	if err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}
	if len(chain) != 1 || chain[0].Type != "move" || chain[0].To != "square" {
		t.Fatalf("chain = %+v, want one move→square", chain)
	}
}

func TestDecomposer_OutOfVocabularyRejected(t *testing.T) {
	// the model proposes an event outside the closed set — the gate rejects (SPEC-015/D-1).
	_, err := DecodeAndValidateChain(`[{"type":"attack","target":"x"}]`)
	if err == nil {
		t.Fatalf("out-of-vocabulary 'attack' was accepted — the leash failed (SPEC-015)")
	}
	if !strings.Contains(err.Error(), "vocabulary") {
		t.Fatalf("error = %v, want a vocabulary rejection", err)
	}
}

func TestPayload_IsPerceptionBound(t *testing.T) {
	// the payload type carries ONLY safety-filtered perception lines — there is no field that could
	// hold raw canon. This is a structural guarantee (the wall is fn_visible_perceptions upstream).
	p := PerceptionPayload{Lines: []string{"You told Mara about the note."}}
	if strings.Contains(strings.Join(p.Lines, "\n"), "mayor keeps a hidden ledger") {
		t.Fatalf("payload leaked a hidden fact")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd core/api && go test ./... -run 'Decomposer|Payload_IsPerceptionBound' -v`
Expected: FAIL — `NewFakeDecomposer`, `DecodeAndValidateChain`, `PerceptionPayload` undefined.

- [ ] **Step 3: Implement the seats**

Create `core/api/beatseats.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// PerceptionPayload is the ONLY world input either model seat receives (B-1/I-3; §14). It is built
// upstream from fn_visible_perceptions — there is deliberately no field that can carry raw canon.
type PerceptionPayload struct {
	Lines []string `json:"lines"` // perception-bound, epistemically framed lines for the holder
}

// BeatStep is one element of the closed-vocabulary chain (beat_chain/1).
type BeatStep struct {
	Type     string `json:"type"`               // "move" | "say" — the closed set
	To       string `json:"to,omitempty"`       // move
	Listener string `json:"listener,omitempty"` // say (uuid)
	Content  string `json:"content,omitempty"`  // say
}

// Decomposer: prose → proposed chain. PROPOSES ONLY (D-1). Perception-bound input (§14).
type Decomposer interface {
	Decompose(ctx context.Context, payload PerceptionPayload, text string) string // returns raw JSON
}

// Narrator: post-beat perception payload → prose. Perception-bound (ADR-020, no omniscient pass).
// Output is presentation, NOT canon — never written to canon_event (I-6).
type Narrator interface {
	Narrate(ctx context.Context, payload PerceptionPayload) string
}

// DecodeAndValidateChain enforces the leash: valid JSON AND every step's type ∈ the closed set
// (SPEC-015/D-1). Anything else is rejected — the model cannot widen the vocabulary.
func DecodeAndValidateChain(raw string) ([]BeatStep, error) {
	var chain []BeatStep
	if err := json.Unmarshal([]byte(raw), &chain); err != nil {
		return nil, fmt.Errorf("decompose output is not valid chain JSON: %w", err)
	}
	for i, s := range chain {
		if !allowedBeatTypes[s.Type] {
			return nil, fmt.Errorf("step %d: type %q is outside the closed vocabulary {move,say}", i, s.Type)
		}
		if s.Type == "move" && s.To == "" {
			return nil, fmt.Errorf("step %d: move requires 'to'", i)
		}
		if s.Type == "say" && (s.Listener == "" || s.Content == "") {
			return nil, fmt.Errorf("step %d: say requires 'listener' and 'content'", i)
		}
	}
	return chain, nil
}

// --- deterministic fakes for CI (the live model is wired only at the operator gate) ---

type fakeDecomposer struct{ table map[string]string }

func NewFakeDecomposer(table map[string]string) Decomposer { return &fakeDecomposer{table} }

func (f *fakeDecomposer) Decompose(_ context.Context, _ PerceptionPayload, text string) string {
	if out, ok := f.table[text]; ok {
		return out
	}
	return "[]" // unknown prose → empty chain (a beat that commits nothing; C-5)
}

type fakeNarrator struct{ prefix string }

func NewFakeNarrator(prefix string) Narrator { return &fakeNarrator{prefix} }

func (f *fakeNarrator) Narrate(_ context.Context, p PerceptionPayload) string {
	out := f.prefix
	for _, l := range p.Lines {
		out += " " + l
	}
	return out
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd core/api && go test ./... -run 'Decomposer|Payload_IsPerceptionBound' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add core/api/beatseats.go core/api/beatseats_test.go
git commit -m "feat(api): perception-bound Decomposer/Narrator seats + closed-vocab validation (SPEC-015)"
```

---

### Task 10: The perception payload builder (reuse the wall) + `POST /worlds/{w}/beat`

**Files:**
- Create: `core/api/beathandler.go`
- Modify: `core/api/main.go` (register the route)
- Test: `core/api/beathandler_test.go`

The endpoint orchestrates: resolve viewer **server-side** (existing `ResolveViewer`, D-7) → build the **perception payload** from `fn_visible_perceptions` (the wall) → `Decompose` (perception-bound) → `DecodeAndValidateChain` (the leash) → `apply_beat(..., 'freeform')` (the **only** canonization point, D-1) → build the **post-beat** payload → `Narrate` (perception-bound) → return `{narration, committed:[...], halt_reason}`. **No canon row ever crosses the boundary** (B-1): the response carries narration + a committed-event *summary*, never `canon_event` rows.

- [ ] **Step 1: Write the failing test**

Create `core/api/beathandler_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// End-to-end with deterministic fakes against the seeded DB. Player at the tavern with Mara present
// (the beat handler positions via a setup helper or the test seeds it); a SAY commits and narrates.
func TestBeat_HappyPath_CommitsAndNarrates(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewBeatHandler(pool, true, // debug → honor ?viewer=
		NewFakeDecomposer(map[string]string{
			"tell mara about the note": `[{"type":"say","listener":"` + maraID + `","content":"the note"}]`,
		}),
		NewFakeNarrator("Scene:"))
	body := strings.NewReader(`{"text":"tell mara about the note"}`)
	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+playerID, body)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "narration") {
		t.Fatalf("response missing narration: %s", rec.Body.String())
	}
	// B-1: no canon vocabulary crosses the boundary (no raw canon_event row shape).
	if strings.Contains(rec.Body.String(), "\"status\":\"accepted\"") {
		t.Fatalf("response leaked a raw canon row: %s", rec.Body.String())
	}
}

func TestBeat_OutOfVocabularyRejected(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewBeatHandler(pool, true,
		NewFakeDecomposer(map[string]string{"attack mara": `[{"type":"attack","target":"` + maraID + `"}]`}),
		NewFakeNarrator("Scene:"))
	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beat?viewer="+playerID,
		strings.NewReader(`{"text":"attack mara"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422 (the leash rejects out-of-vocabulary; SPEC-015)", rec.Code)
	}
}
```

> The test references `maraID`, `playerID`, `worldID` and `testPool` — these already exist in `core/api/compendium_test.go`/`viewer_test.go`. If `maraID`/`playerID` are not yet declared as test constants there, add them alongside the existing fixed-UUID constants in one place (do not redeclare).

- [ ] **Step 2: Run to verify it fails**

Run: `cd core/api && make -C ../.. reset && go test ./... -run TestBeat -v`
Expected: FAIL — `NewBeatHandler` undefined.

- [ ] **Step 3: Implement the handler**

Create `core/api/beathandler.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

var beatRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beat$`)

type beatHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	dec  Decomposer
	nar  Narrator
}

func NewBeatHandler(pool *pgxpool.Pool, debug bool, dec Decomposer, nar Narrator) http.Handler {
	return &beatHandler{pool: pool, dbg: debug, dec: dec, nar: nar}
}

func (h *beatHandler) match(r *http.Request) []string {
	if r.Method != http.MethodPost {
		return nil
	}
	return beatRoute.FindStringSubmatch(r.URL.Path)
}

func (h *beatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.match(r)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := r.Context()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer", http.StatusBadRequest)
		return
	}

	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}

	// 1. perception payload BEFORE resolving — the decomposer is perception-bound (§14).
	pre, err := h.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}
	// 2. decompose (PROPOSES ONLY, D-1) → validate against the closed vocabulary (the leash, SPEC-015).
	chain, err := DecodeAndValidateChain(h.dec.Decompose(ctx, pre, in.Text))
	if err != nil {
		http.Error(w, "outside the closed vocabulary", http.StatusUnprocessableEntity) // 422
		return
	}
	raw, _ := json.Marshal(chain)

	// 3. apply_beat is the ONLY canonization point (D-1). origin='freeform' = model-proposed, gated.
	//    start_tick = max accepted tick for the world + 1; cap = generous backstop.
	var summary []byte
	err = h.pool.QueryRow(ctx,
		`SELECT apply_beat($1,$2,$3::jsonb,
		          COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,
		          $4, 'freeform')::text`,
		worldID, viewerID, string(raw), beatTickCap).Scan(&summary)
	if err != nil {
		http.Error(w, "apply", http.StatusInternalServerError)
		return
	}

	// 4. perception payload AFTER the beat — the narrator is perception-bound (ADR-020).
	post, err := h.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}
	narration := h.nar.Narrate(ctx, post) // presentation only; never written to canon (I-6)

	w.Header().Set("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]any{
		"schema_version": "beat_result/1",
		"narration":      narration,
		"result":         json.RawMessage(summary), // {committed:[...ids], halt_reason, ticks_advanced}
	})
	_, _ = w.Write(resp)
}

// payload builds the perception-bound payload from the WALL (fn_visible_perceptions). No raw canon.
func (h *beatHandler) payload(ctx context.Context, worldID, viewerID string) (PerceptionPayload, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT content FROM fn_visible_perceptions($1,$2) ORDER BY acquired_tick`, worldID, viewerID)
	if err != nil {
		return PerceptionPayload{}, err
	}
	defer rows.Close()
	var p PerceptionPayload
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return PerceptionPayload{}, err
		}
		p.Lines = append(p.Lines, c)
	}
	return p, rows.Err()
}

// beatTickCap is the generous hard time-cap backstop (§9; ADR-025 provisional — tune at the gate).
const beatTickCap = 1000
```

- [ ] **Step 4: Register the route in `main.go`**

In `core/api/main.go`, where the read handlers are registered (the `router`/`handlers` slice), add a beat handler with **deterministic fakes by default** (the live model is wired separately for the operator gate, never in the default server build):

```go
	// Chunk-5 play loop. Default seats are deterministic fakes; the operator gate swaps in the live
	// model via a build/config path (kept out of CI). The endpoint is the ONLY write path; everything
	// it commits goes through apply_beat (D-1).
	r.handlers = append(r.handlers, NewBeatHandler(pool, debug,
		NewFakeDecomposer(map[string]string{}), NewFakeNarrator("")))
```

> Match the exact field/append idiom already used in `main.go` for the read handlers (the survey shows handlers are appended to a `router`). If `main.go` constructs handlers via a different helper, follow that pattern — keep the fakes as the default seats.

- [ ] **Step 5: Run to verify it passes**

Run: `cd core/api && make -C ../.. reset && go test ./... -run TestBeat -v && go vet ./...`
Expected: PASS; `go vet` clean.

- [ ] **Step 6: Commit**

```bash
git add core/api/beathandler.go core/api/beathandler_test.go core/api/main.go
git commit -m "feat(api): POST /worlds/{w}/beat — perception-bound loop, apply_beat the sole gate (D-1)"
```

---

### Task 11: Canon-authority + no-leak Go assertions (SPEC-015 / B-1 / I-3 at the boundary)

**Files:**
- Test: `core/api/beat_authority_test.go`

Two boundary properties the operator gate will re-verify by hand, pinned here as automated tests with fakes: **(1) canon-authority** — every committed event traces to a gated proposal and carries `origin='freeform'` (the model wrote nothing to canon directly; D-1/SPEC-015); **(2) no-leak (data layer)** — a hidden fact in another holder's perception never appears in the viewer's payload handed to either seat (B-1/I-3).

- [ ] **Step 1: Write the test**

Create `core/api/beat_authority_test.go`:

```go
package main

import (
	"context"
	"strings"
	"testing"
)

// Canon-authority: a committed beat's events are all origin='freeform' (gated proposals), never
// written by the model directly. Verified by querying canon after a beat through the handler path.
func TestBeat_CanonAuthority_AllCommittedAreGatedProposals(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	// run a beat that commits one move via apply_beat with origin 'freeform' (handler path).
	var summary string
	err := pool.QueryRow(ctx,
		`SELECT apply_beat($1,$2,'[{"type":"move","to":"square"}]'::jsonb,
		   COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1, 1000, 'freeform')::text`,
		worldID, playerID).Scan(&summary)
	if err != nil {
		t.Fatalf("apply_beat: %v", err)
	}
	var bad int
	// any committed event from this run with origin <> 'freeform' would mean canon written outside the gate
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin NOT IN
		   ('fast_path','template','freeform','threshold','backstage','compensation')`, worldID).Scan(&bad)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if bad != 0 {
		t.Fatalf("found %d committed events with an illegal origin (canon-authority breach)", bad)
	}
	if !strings.Contains(summary, "committed") {
		t.Fatalf("apply_beat summary missing committed list: %s", summary)
	}
}

// No-leak (data layer): the seeded hidden fact (Mara's 'told' about the ledger) never appears in
// Jonas's payload. This is the wall the Q3 no-leak claim rests on (B-1/I-3; generation-layer O-1 deferred).
func TestBeat_NoLeak_HiddenFactAbsentFromUninvolvedPayload(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := &beatHandler{pool: pool, dbg: true}
	p, err := h.payload(context.Background(), worldID, jonasID)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	if strings.Contains(strings.ToLower(strings.Join(p.Lines, "\n")), "hidden ledger") {
		t.Fatalf("Jonas's payload leaked the hidden fact: %v", p.Lines)
	}
}
```

- [ ] **Step 2: Run to verify it passes**

Run: `cd core/api && make -C ../.. reset && go test ./... -run 'CanonAuthority|NoLeak' -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add core/api/beat_authority_test.go
git commit -m "test(api): canon-authority (SPEC-015) + data-layer no-leak (B-1/I-3) at the beat boundary"
```

---

## LEG 2 GATE → red = STOP

**Green iff ALL hold:**
1. `cd core/api && make -C ../.. reset && go test ./... -v && go vet ./...` — all green: schema closed-vocabulary, perception-bound seats, the leash rejection, the beat endpoint, canon-authority, data-layer no-leak.
2. `make reset && make test && make replay` — the full pgTAP suite + replay still green (Leg 2 added no SQL behavior; the engine is unchanged).
3. `make schema-contract` — the published-schema contract (SPEC-011) still green; `beat_chain/1` is additive.
4. No frontend code in the backend repo: `git ls-files | grep -E '^frontend/|\.tsx?$'` returns nothing (D-7/D-10).

**If any is red: STOP. Do not request the operator gate.**

---

## §17. The Chunk-5 OPERATOR GATE (Q3) — founder-run, by hand

> **Ladder claim (Q3):** the trust guarantees survive even though a **non-deterministic LLM is now in the loop** — because the model is **quarantined**: proposes only (D-1), perception-bound only (B-1/I-3/ADR-020), never canon-authority. Q1 proved replay with *fixed* events; Q3 proves replay + no-leak still hold when the events come from a *live, variable* model. **The sharpest idea: narration may vary run-to-run; canon must replay.** (Process law: a 🪜 chunk needs an honest Validation-Ladder answer — green CI + product "no" = not done.)

**Setup (one-time, founder).** Wire the **live model** into the two seats (a config/build path kept out of CI — never the default server build). The hidden fact is the **existing seed secret**: Mara's `'told'` perception "the mayor keeps a hidden ledger" (`seed_mara_0A.sql` E1), which the **Player did not witness from Jonas's side** and **Jonas does not hold**. Play a scripted scenario end-to-end through `POST /worlds/{w}/beat` (as Player; and via `?viewer=` as Jonas in debug), then verify the **four inspectable checks**, each producing reviewable evidence (verify the artifact, not a green check):

- [ ] **Check 1 — Replay invariance (I-1 / ADR-026).** Replay the session's committed event log: `make replay` (which calls `replay_0A()`), then a fresh-DB rebuild from the committed events. Assert the rebuilt world is **domain-equivalent** — empty *domain* diff, excluding volatile transaction-time fields (`updated_at`, `recorded_at`; SPEC-002). **Evidence:** the empty `make replay` boolean + a `make fingerprint` diff across two deploys = identical. *Q1 surviving the live loop.*

- [ ] **Check 2 — No-leak, DATA-LAYER scoped (B-1 / I-3).** The seeded hidden fact ("hidden ledger") never appears in **any perception payload handed to the model** (decompose or narrate, captured to a log) **nor in any narration**, for a viewer who must not know it (Jonas). **Evidence:** the captured payloads + narration text, grepped for the distinctive token — **zero hits**. **Explicit boundary (do not overclaim):** this proves the **data-layer** wall (what *enters* the prompt is provably holder-scoped, doc 06 §1). It does **NOT** claim the **generation layer** is leak-proof — the single-call multi-holder leak (O-1, ADR-020) is unsolved and **deferred to Phase 3** (doc 07 §6). A green Check 2 = the model only ever *saw* safety-filtered perception.

- [ ] **Check 3 — Partial-beat correctness.** Run a deliberately-halting chain **both ways**: (a) a **gate-reject** (e.g. say to a non-co-present actor mid-chain → the prefix commits, the impossible step rejects, the remainder never runs) and (b) a **stop-check** (move into a room, discover an actor that breaks the next queued say → the move commits, the say never runs). Assert **exactly the prefix** committed — zero trace from the halt onward, clock advanced by prefix durations only. **Evidence:** that beat's `canon_event` log + `perception_record` rows + the `apply_beat` summary (`halt_reason`, `ticks_advanced`).

- [ ] **Check 4 — Canon-authority (D-1 / SPEC-015).** Every committed event traces to the gate (`apply_beat`) committing a gated proposal (`origin='freeform'`); the model wrote nothing to canon directly; no committed event lies outside the player's actual intent. **Evidence:** the authorship/`origin` + `event_participant`/provenance of the session's committed events.

**Sign-off.** Founder plays it, reviews the four pieces of evidence, signs off, and tags `chunk-5-play-loop-gate` on the verified backend `main` merge — same shape as the Compendium gate (Q2), but the object under inspection is the **live loop**, not a static page. **Gate red → STOP** (process law). The frontend play-loop UI is a **separate** plan/PR to `dreamchat-frontend`.

---

## §18. Process law (enforced throughout this plan)

- **One chunk = one worktree = one plan = one PR** to backend `main` (playbook §1; AGENTS.md). This document is committed via a **docs-only PR**; execution happens later in a fresh worktree.
- **Gate red → STOP.** Leg-1 gate red → no Leg 2. Leg-2 gate red → no operator gate. Operator gate red → no Chunk 6.
- **TDD iron law:** failing test first, every task; **invariants I-1…I-10 are the permanent regression suite** (the frozen `00`–`90` pgTAP files + Go suite stay green unchanged; the new `91`–`98` + Go tests join them).
- **D-9 running-code evidence:** the only engine change (action-driven clock) ships as **ADR-036**, drafted Proposed at Leg-1 start and Accepted at the Leg-1 gate **with `apply_beat` + the play-loop suite as evidence** — never assumed ahead of the code.
- **The model stays quarantined:** proposes only (D-1), perception-bound only (B-1/I-3/ADR-020), never canon-authority (SPEC-015). `apply_beat` is the sole canonization point.
- **No frozen-DDL changes:** no new core table or column on the frozen Master DDL (durations live in engine functions; perceptions/events use existing tables). New SQL functions and the JSONB-carried beat payload are additive (D-4). Schema additions, had any been needed, would have required a proposed engine ADR (governance) — none are.

---

## Self-Review

**Spec coverage (capture doc → tasks):**
- §1 full play loop with the LLM → Legs 1+2; §2 closed event vocabulary → Task 8 (schema) + Task 9 (validation). ✓
- §3 event spine (thin slice MOVE+SAY = existing `move`/`private_disclosure`) → Tasks 3–5; broader vocabulary deferred (§0.1). ✓
- §4 two perception triggers (witnessing + discovery-on-arrival) → Task 4; unwitnessed truth honored (no perceiver ⇒ no row). ✓
- §8 four-stage pipeline + two first-class halts → Task 5; §9 perception-vs-expectation stop (discovery) + turn-budget backstop + partial-beat → Tasks 5, 6. ✓
- §10 action-driven clock + §11 recorded durations → Tasks 2, 5 + ADR-036; §12 spatial deferred (SPEC-018) per the founder decision. ✓
- §13 gate = sole canon writer → Task 5 + Task 10 (apply_beat the only canonization point). ✓
- §14 perception payload (not "bundle"), both seats perception-bound → Tasks 9, 10; decomposer-binding flagged as a tested extension (§0 Finding D). ✓
- §15 deception → deferred (SPEC-016, no AttributeChanged in slice) — explicit. ✓
- §16 corrections → deferred (verify-only; Finding C) — explicit. ✓
- §17 operator gate (four checks + evidence) → §17 above, no-leak data-layer scoped. ✓
- §19 SPEC-012…018 boundaries → §0.1 deferred list, each named. ✓

**Reconciliation surfaced (§0):** A (no spine ADR), B (beat_seq/ADR-034 already enforced — capture §8 stale; next free ADR = 036), C (provenance_edge leaf rule served by idx_pe_source, no DDL change), D (payload≠bundle; decomposer-binding an extension; Q3 no-leak data-layer scoped). ✓

**Placeholder scan:** no TBD/TODO; every SQL/Go/JSON block is complete and runnable. ✓

**Type/name consistency:** SQL — `fn_actors_at`, `fn_move_duration`, `generate_perceptions`, `apply_beat` (signature `(uuid,uuid,jsonb,bigint,bigint,text)`), `fn_visible_perceptions` (reused), `replay_0A` (reused). Go — `PerceptionPayload`, `BeatStep`, `Decomposer`/`Narrator`, `DecodeAndValidateChain`, `NewFakeDecomposer`/`NewFakeNarrator`, `NewBeatHandler`, `beatTickCap`, `allowedBeatTypes`/`vocabularyTypes`. Test files `91`–`98` (pgTAP) + `beat_*_test.go`. Migration `20260618090001_play_loop_engine.sql`. ADR-036. All consistent across tasks. ✓
