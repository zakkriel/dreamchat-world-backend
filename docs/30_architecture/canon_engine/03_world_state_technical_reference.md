# 03 — World-State Technical Reference

**Status:** Engineering reference. Contains the Master DDL, the event lifecycle, projection update rules, the Traversal Matrix, propagation protocol, dirty-flag semantics, snapshotting, indexing/caching, and schema evolution. Behavior of the canonization service, entity resolution, and context assembly live in docs 04–06.

---

## 1. Master DDL (PostgreSQL 16+)

Conventions: every table carries `world_id` as the partition/tenant key; **system/transaction timestamps** (`recorded_at`, `accepted_at`, `created_at`) are `TIMESTAMPTZ`; **fictional/in-world time is logical, never `TIMESTAMPTZ`** (see below); JSONB payloads carry a `schema_version`.

**Logical world time (ADR-021/030).** DreamChat is genre-agnostic — worlds may use non-Earth calendars, voyages ("Day 17"), eras, dream-time, or time loops. A real-world timestamp would silently impose a Gregorian linear-time model on every fiction. Fictional time is therefore stored as a monotonic logical tick plus a human label, never as a calendar timestamp:

```
world_time_tick   BIGINT       -- monotonic ordering primitive; the ONLY thing compared for sequence
world_time_label  TEXT         -- human/fiction-facing: 'Day 17 of the voyage', 'the third Age', '21:30'
calendar_system   TEXT         -- per-world: 'gregorian','voyage_days','custom:...'; interpretation only
temporal_uncertainty BOOLEAN   -- set when the tick is approximate (doc 10 §3)
```

`world_time_tick` is the universal sortable in-world clock; ordering and "what was true at tick T" comparisons use it exclusively. `world_time_label` is for display and prompt rendering. Worlds that genuinely run on real-world time may additionally map a tick to a `TIMESTAMPTZ` in world config, but the core never depends on that mapping. The World Clock service (doc 10) owns tick assignment. Intra-beat ordering uses `beat_seq` *within* a tick (doc 10 §4); on the Narrative Claim Ledger, ordering lives on the claim (doc 12).

### 1.1 Canon spine

```sql
CREATE TABLE canon_event (
  event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  scene_id         UUID,
  beat_id          UUID,                       -- correction-window block this event belongs to
  event_type       TEXT NOT NULL,              -- 'move','trade','private_disclosure','theft','utterance',...
  summary          TEXT NOT NULL,              -- one-line human-readable canon statement
  payload          JSONB NOT NULL DEFAULT '{}',
  schema_version   INT  NOT NULL DEFAULT 1,
  in_world_tick    BIGINT NOT NULL,           -- VALID TIME (logical fictional clock; sortable)
  in_world_label   TEXT,                       -- human/fiction-facing time label
  beat_seq         INT NOT NULL DEFAULT 0,     -- intra-tick ordering within a beat (doc 10 §4)
  temporal_uncertainty BOOLEAN NOT NULL DEFAULT false,
  recorded_at      TIMESTAMPTZ NOT NULL DEFAULT now(),  -- TRANSACTION TIME (system clock)
  accepted_at      TIMESTAMPTZ,                -- set on acceptance; NULL while proposed
  status           TEXT NOT NULL DEFAULT 'proposed'
                   CHECK (status IN ('proposed','accepted','rejected','retconned','superseded')),
  visibility_scope TEXT NOT NULL DEFAULT 'private',     -- 'public' | 'faction:<uuid>' | 'private' | 'secret'
  confidence       REAL,
  origin           TEXT NOT NULL DEFAULT 'fast_path'
                   CHECK (origin IN ('fast_path','template','freeform','threshold','backstage','compensation')),
  template_id      TEXT,                       -- which template produced it, if any
  source_refs      JSONB,                      -- transcript message ids / extraction inputs (AUDIT ONLY, never truth)
  superseded_by    UUID REFERENCES canon_event(event_id)
);

CREATE INDEX idx_ce_world_time   ON canon_event (world_id, in_world_tick, beat_seq);
CREATE INDEX idx_ce_status       ON canon_event (world_id, status) WHERE status = 'accepted';
CREATE INDEX idx_ce_beat         ON canon_event (beat_id);
CREATE INDEX idx_ce_scene        ON canon_event (scene_id);
CREATE INDEX idx_ce_payload_gin  ON canon_event USING GIN (payload);
```

**Append-only enforcement.** A `BEFORE UPDATE` trigger rejects any update that touches columns other than `{status, accepted_at, superseded_by}` and rejects illegal status transitions. Legal transitions: `proposed→accepted`, `proposed→rejected`, `accepted→retconned`, `accepted→superseded`. `DELETE` is revoked from all roles.

```sql
CREATE TABLE event_participant (        -- qualified E2O hyperedge spokes
  event_id       UUID NOT NULL REFERENCES canon_event(event_id),
  entity_id      UUID NOT NULL,
  entity_kind    TEXT NOT NULL CHECK (entity_kind IN ('actor','location','artifact','faction','group')),
  role_qualifier TEXT NOT NULL,         -- 'instigator','target','tool','witness','scene','speaker','listener',
                                        -- 'owner','carrier','container','source','recipient','affected'
  PRIMARY KEY (event_id, entity_id, role_qualifier)
);
CREATE INDEX idx_ep_entity ON event_participant (entity_id);
```

One event, many qualified participant rows. **Never** inline participant arrays in the event payload.

### 1.2 Deltas and provenance

```sql
CREATE TABLE state_mutation (
  mutation_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id       UUID NOT NULL,
  event_id       UUID NOT NULL REFERENCES canon_event(event_id),   -- provenance, mandatory (ADR-008)
  entity_id      UUID NOT NULL,
  entity_kind    TEXT NOT NULL,
  attribute_path TEXT NOT NULL,          -- 'attrs.inventory.gold' | 'attrs.location_id' | 'attrs.trust'
  old_value      JSONB,
  new_value      JSONB NOT NULL,
  valid_from_tick BIGINT NOT NULL,       -- in-world tick the new value takes effect
  valid_from_seq  INT NOT NULL DEFAULT 0, -- intra-tick order
  status         TEXT NOT NULL DEFAULT 'applied'
                 CHECK (status IN ('applied','reversed','dirty'))
);
CREATE INDEX idx_sm_entity ON state_mutation (entity_id, valid_from_tick, valid_from_seq);
CREATE INDEX idx_sm_event  ON state_mutation (event_id);
```

```sql
CREATE TABLE provenance_edge (           -- PROV-style typed lineage
  derived_id   UUID NOT NULL,
  derived_kind TEXT NOT NULL CHECK (derived_kind IN ('perception','mutation','event','bundle')),
  source_id    UUID NOT NULL,
  source_kind  TEXT NOT NULL CHECK (source_kind  IN ('perception','mutation','event')),
  how_type     TEXT NOT NULL CHECK (how_type IN
               ('derived_from','inferred_from','reported_by','witnessed_by','compensates','supersedes')),
  PRIMARY KEY (derived_id, source_id, how_type)
);
CREATE INDEX idx_pe_source ON provenance_edge (source_id);
```

### 1.3 Epistemic layer

```sql
CREATE TABLE perception_record (
  perception_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  holder_id        UUID NOT NULL,        -- actor or faction
  source_event_id  UUID NOT NULL REFERENCES canon_event(event_id),  -- provenance, mandatory
  content          TEXT NOT NULL,        -- the narrative string injected into LLM context
  epistemic_type   TEXT NOT NULL CHECK (epistemic_type IN
                   ('direct','shared','told','overheard','public','rumor',
                    'inference','mistaken','confirmed','disputed')),
  sensory_mode     TEXT,                 -- 'visual','auditory','deduction', ...
  confidence       REAL NOT NULL DEFAULT 1.0,
  distortion_level REAL NOT NULL DEFAULT 0,
  acquired_tick    BIGINT NOT NULL,      -- EPISTEMIC TIME (in-world tick the holder learned it)
  valid_tick       BIGINT NOT NULL,      -- in-world tick the believed fact became true
  invalid_tick     BIGINT,               -- in-world tick belief falsified/superseded (NULL = still held)
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  expired_at       TIMESTAMPTZ,          -- system-level supersession
  visibility_scope TEXT NOT NULL DEFAULT 'private',
  dirty            BOOLEAN NOT NULL DEFAULT false,
  importance       REAL NOT NULL DEFAULT 5.0  -- 1..10, set at creation (doc 06 scoring)
);
CREATE INDEX idx_pr_holder  ON perception_record (holder_id, acquired_tick);
CREATE INDEX idx_pr_source  ON perception_record (source_event_id);
CREATE INDEX idx_pr_active  ON perception_record (holder_id) WHERE invalid_tick IS NULL AND expired_at IS NULL;
```

Invalidation rule: contradiction or correction **closes** a perception (sets `invalid_tick` for in-world falsification and/or `expired_at` for system supersession) and writes a replacement; nothing is deleted (ADR-006). Note `expired_at` is a system timestamp (transaction-time supersession); `invalid_tick` is fictional time (when the belief became false in-world) — the two axes stay distinct.

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
CREATE INDEX idx_ps_world  ON perception_subject (world_id);
```

### 1.4 Causal layer (schema-ready from Phase 0; used from Phase 4 — ADR-008)

```sql
CREATE TABLE causal_bundle (
  bundle_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id    UUID NOT NULL,
  effect_ref  UUID NOT NULL,             -- the event or mutation this bundle explains
  effect_kind TEXT NOT NULL CHECK (effect_kind IN ('event','mutation')),
  semantics   TEXT NOT NULL CHECK (semantics IN ('conjunctive','disjunctive_member','probabilistic')),
  template_id TEXT,                      -- NULL = free-form escape hatch (must be pending_review)
  status      TEXT NOT NULL DEFAULT 'valid'
              CHECK (status IN ('valid','invalidated','pending_review')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cb_effect ON causal_bundle (effect_ref);

CREATE TABLE causal_bundle_input (
  bundle_id  UUID NOT NULL REFERENCES causal_bundle(bundle_id),
  input_ref  UUID NOT NULL,              -- durable records ONLY: event, mutation, or perception id
  input_kind TEXT NOT NULL CHECK (input_kind IN ('event','mutation','perception')),
  role       TEXT NOT NULL CHECK (role IN ('trigger','enabler','blocker','influence')),
  polarity   SMALLINT NOT NULL DEFAULT 1 CHECK (polarity IN (1,-1)),
  weight     REAL NOT NULL DEFAULT 1.0,
  necessity  BOOLEAN NOT NULL DEFAULT true,
  PRIMARY KEY (bundle_id, input_ref, role)
);
```

**Hard rule (G3):** bundle inputs reference durable records only — never free text, never a live projection read. If a condition has no record, the extractor must propose the missing event first or drop the input. **Acyclicity** is checked at bundle insert (bounded ancestor walk; reject on cycle).

### 1.5 Projections (read models — rebuildable, never authoritative)

```sql
CREATE TABLE actor_state (
  entity_id     UUID PRIMARY KEY,
  world_id      UUID NOT NULL,
  attrs         JSONB NOT NULL DEFAULT '{}',  -- hp, inventory, location_id, condition, mood vector...
  dirty         BOOLEAN NOT NULL DEFAULT false,
  last_event_id UUID,                         -- provenance of last applied delta
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- location_state and artifact_state: identical shape.

CREATE TABLE relationship_state (
  world_id      UUID NOT NULL,
  a_id          UUID NOT NULL,
  b_id          UUID NOT NULL,               -- store with a_id < b_id; direction inside attrs if needed
  attrs         JSONB NOT NULL DEFAULT '{}', -- affinity, trust, suspicion, debt...
  dirty         BOOLEAN NOT NULL DEFAULT false,
  last_event_id UUID,
  PRIMARY KEY (world_id, a_id, b_id)
);
```

```sql
CREATE TABLE entity_registry (               -- doc 05's substrate; also a Phase 1 dependency
  entity_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  entity_kind      TEXT NOT NULL,
  canonical_name   TEXT NOT NULL,
  aliases          TEXT[] NOT NULL DEFAULT '{}',
  descriptor       TEXT,                     -- short disambiguator: 'night-shift museum guard'
  current_scene_id UUID,
  created_by_event UUID REFERENCES canon_event(event_id),
  status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','merged'))
);
CREATE INDEX idx_er_scene ON entity_registry (world_id, current_scene_id);
CREATE INDEX idx_er_name  ON entity_registry (world_id, canonical_name);
```

### 1.6 Operational tables

```sql
CREATE TABLE review_queue (
  item_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  trigger_event_id UUID NOT NULL,
  target_ref       UUID NOT NULL,
  target_kind      TEXT NOT NULL,
  radius           INT  NOT NULL,
  priority         INT  NOT NULL DEFAULT 5,   -- promoted by player proximity (ADR-014 ladder)
  status           TEXT NOT NULL DEFAULT 'queued'
                   CHECK (status IN ('queued','in_progress','resolved','abandoned')),
  payload          JSONB NOT NULL DEFAULT '{}',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE threshold_ledger (                -- ADR-015 accumulator
  world_id        UUID NOT NULL,
  target_ref      UUID NOT NULL,
  attribute_path  TEXT NOT NULL,
  source_ref      UUID NOT NULL,               -- the influencing perception/event
  weight          REAL NOT NULL,
  valid           BOOLEAN NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (world_id, target_ref, attribute_path, source_ref)
);

CREATE TABLE world_snapshot (
  world_id        UUID NOT NULL,
  as_of_event_seq BIGINT NOT NULL,
  taken_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  storage_ref     TEXT NOT NULL,               -- object-storage key
  PRIMARY KEY (world_id, as_of_event_seq)
);

CREATE TABLE extraction_log (                  -- ADR-012: corpus for the future SLM; audit trail
  log_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id    UUID NOT NULL,
  beat_id     UUID,
  input_text  TEXT NOT NULL,
  prompt_meta JSONB,
  output_json JSONB,
  verdicts    JSONB,                           -- gate verdicts incl. errors and repair attempts
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## 2. Event lifecycle

```
            fast path ─────────────────────────────┐
user/system action ─► [proposed] ─► validation ─► [accepted] ─► triggers fire:
                          │            │              │           R1 mutations applied
                          │            ▼              │           R1 perceptions written
                          │        [rejected]         │           threshold_ledger updated
                          │      (repair retry ×1)    │           R2+ items queued
                          ▼                           ▼
                   discarded / parked        [retconned | superseded]
                                              (via compensating events only)
```

Fast-path events skip `proposed` (origin `fast_path`, validated by construction). Slow-path events are born `proposed` inside a beat's correction window (doc 04). Only `accepted` events are visible to projections, perceptions, thresholds, and replay.

## 3. Projection update rules

1. Triggers fire **on acceptance only** (insert with `status='accepted'` or transition to it).
2. Each `state_mutation` is applied to its projection by `attribute_path`; `last_event_id` is set; the write is **idempotent** (re-applying the same mutation_id is a no-op — keep an applied-mutations ledger or use deterministic upserts keyed by mutation_id).
3. Projections never receive writes from any other source. Code reviews enforce this; I-7 (doc 07) tests it.
4. Rebuild procedure: truncate projections → stream accepted events in `(in_world_tick, beat_seq, recorded_at)` order → re-apply mutations → compare. This is invariant I-1 and runs nightly per active world.
5. pg_ivm or an IVM engine replaces triggers only when measured trigger cost hurts (ADR-004).

## 4. The Traversal Matrix (enforced)

Propagation code filters relations through this matrix. Adding a relation type without a matrix row fails CI.

| Relation / mechanism | Stored as | Radius-1 sync | Radius-2+ queue | Forbidden for propagation |
|---|---|---|---|---|
| mutates_state_of | state_mutation row | ✅ | — | — |
| creates_perception | perception_record row | ✅ | — | — |
| causes (conjunctive bundle) | bundle | — | ✅ (strict dirty on break) | — |
| enables (bundle input) | bundle input | — | ✅ (plausibility review on break) | — |
| blocks / prevents | bundle input, polarity −1 | — | ✅ | — |
| affects_probability_of | threshold_ledger row | ✅ (ledger write) | threshold event on trip | ❌ never cascades as causality |
| inferred_from / reported_by | provenance_edge | — | ✅ (belief-tree invalidation only) | — |
| compensates / supersedes | provenance_edge + status | ✅ | — | — |
| contradicts | perception pairing | — | ✅ (review item) | — |
| temporally_before | derived from timestamps | — | — | ❌ **forbidden** |
| same_scene_as | scene_id grouping | — | — | ❌ **forbidden** |
| references_* / knows / located_in | reference layer | — | — | ❌ **forbidden** |

## 5. Propagation protocol & dirty flags

- **R0:** compensating event only (cosmetic fixes). No traversal.
- **R1 (synchronous):** the accepted event's own mutations and perceptions, plus threshold-ledger writes. Bounded by construction (the event's own rows).
- **R2+ (asynchronous):** queue items created from bundle reachability (Phase 4+) or explicit consequence rules, with `radius` recorded, hard depth cap (default 3), and a per-event propagation budget (default 25 queue items; overflow → single review item for a human/designer).
- **Dirty flags:** queue processing may mark projections/perceptions `dirty=true` instead of resolving immediately. Resolution happens (a) by the backstage worker in priority order, or (b) just-in-time at the context assembler (doc 06), whichever comes first. Clearing a flag requires writing the resolving compensation/update with provenance.
- **Belief-tree invalidation:** when an event or perception is falsified, walk `inferred_from`/`reported_by` descendants only — this invalidates exactly the beliefs descended from it and nothing else.

## 6. Visibility scopes

`public` — eligible for any holder in scope of the location/faction broadcast rules. `faction:<id>` — members only. `private` — named holders only. `secret` — no automatic perception fan-out at all; perceptions exist only if explicitly created (e.g., the perpetrator's own). Scope checks run in the gate (KNOWLEDGE/SCOPE violations) and again in context assembly (defense in depth).

## 7. Snapshots, caching, indexing

- **Snapshots:** every N accepted events per world (default 100) or at session end; compacted projection state + high-water mark to object storage. Replay for rebuilds starts from the last snapshot.
- **Hot cache (Redis):** active scene's projections, present entities' active perceptions, scene registry slice. Invalidated by the same triggers that update projections. Target: context-assembly data reads < 10 ms from cache.
- **Indexing:** as in the DDL — entity-keyed projections, time+status-keyed events (partial index on accepted), holder-keyed perceptions, GIN on JSONB payloads.

## 8. Schema evolution (G6)

Every JSONB payload carries `schema_version`. Projection builders and the replay engine route payloads through **upcasters** (pure functions `v(n) → v(n+1)`) before application. Events are never migrated in place — upcasting happens at read/replay time. New event types require: payload schema + upcaster registration + projection handler + Traversal Matrix row + (if applicable) template entry. CI rejects event types missing any of these.

## 9. Performance posture (summary)

Hot path = Redis-cached projections + holder perceptions + dirty checks + token packing. One synchronous event append + R1 deltas per accepted beat. Everything else queued. Lineage queries are depth-capped recursive CTEs, off the hot path. The system's read cost is independent of world age by construction.
