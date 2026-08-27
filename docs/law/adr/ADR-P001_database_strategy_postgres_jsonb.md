> ⚠️ **STATUS: SUPERSEDED IN PART (2026-06-10).** The core schema proposed here (`entity`, `entity_profile`, `relationship_edge`, …) is **replaced by the Canon Engine Master DDL** — see `30_architecture/canon_engine/03_world_state_technical_reference.md` and engine ADR-001/-003. **The engine DDL is the only core schema for implementation.**
> **What survives from this ADR:** (1) Postgres-first, no MongoDB — reaffirmed by engine ADR-003; (2) **JSONB for module-owned / evolving / experimental state** (`module_state`, `module_config`, payloads) with mandatory `schema_version` and runtime validation — still the standing rule for the Module Runtime, which the engine set does not cover; (3) the strict-vs-flexible principle: world truth relational and auditable, module state flexible and namespaced.

---

# ADR-001: Database Strategy — Postgres Core with JSONB Flexibility

## Status

Proposed

## Date

2026-06-09

## Context

DreamChat is a persistent AI RPG world platform. The product is not only storing chat messages or generated content. It is maintaining a living world with entities, relationships, timelines, knowledge boundaries, canon events, module-driven mechanics, backstage updates, and correction UX.

During early product development, schemas will evolve quickly. New systems such as battle, stats, investigation, factions, crafting, economy, romance, or reputation may require different state shapes. A document database such as MongoDB is attractive because it allows flexible data structures while the product is still being discovered.

However, the hardest parts of DreamChat are not flexible content storage. The hardest parts are continuity, causality, auditability, consistency, and controlled world mutation. The app needs to know what happened, when it happened, who knows it, whether it is canon, what changed as a result, and how a correction should affect future behavior.

The architecture also assumes a future plug-and-play module system. Modules should be able to introduce their own state and mechanics without forcing the core database schema to know every possible future feature. At the same time, modules must not bypass the authoritative world core or write directly into canon.

## Decision

Use **Postgres as the authoritative core database from day one**, while using **JSONB-based flexible tables** for evolving, module-owned, and experimental state.

Do **not** use MongoDB as the primary builder database with the intention of migrating to Postgres later.

The chosen model is:

```text
Postgres authoritative core
  - strict relational schema for world-critical data
  - JSONB flexibility for module-owned and evolving data
  - event log as durable source of world change history
  - future compatibility with pgvector for memory/retrieval
```

## Architectural Principle

> Core world truth should be relational, auditable, and transactional.  
> Module-specific state can be flexible, namespaced, and schema-versioned.

This keeps the world stable while allowing gameplay systems to evolve.

## What Must Be Strict / Relational

The following concepts should live in normalized relational tables because they are shared across the whole product and must remain queryable, auditable, and consistent:

```text
world
world_version
user_controlled_entity
entity
location
scene
scene_participant
relationship_edge
canon_entry
knowledge_state
event_log
timeline_entry
module_installation
audit_log
correction_record
```

These tables represent the substrate of the world. They should not be hidden inside opaque documents.

## What Can Be Flexible / JSONB

The following data can use JSONB because it is evolving, module-specific, experimental, or not yet stable enough for strict modeling:

```text
entity_profile.profile_jsonb
module_state.state_jsonb
module_config.config_jsonb
event_log.payload_jsonb
canon_entry.payload_jsonb
memory_episode.payload_jsonb
generated_blueprint.payload_jsonb
draft_world_content.payload_jsonb
scene_working_state.state_jsonb
uncommitted_proposal.payload_jsonb
```

JSONB should not mean “no structure.” JSONB payloads should still have schema versions and, where possible, runtime validation.

## Proposed Core Tables

### `entity`

```sql
CREATE TABLE entity (
  id UUID PRIMARY KEY,
  world_id UUID NOT NULL,
  entity_type TEXT NOT NULL,
  display_name TEXT NOT NULL,
  lifecycle_status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### `entity_profile`

```sql
CREATE TABLE entity_profile (
  entity_id UUID PRIMARY KEY REFERENCES entity(id),
  schema_version TEXT NOT NULL,
  profile_jsonb JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### `relationship_edge`

```sql
CREATE TABLE relationship_edge (
  id UUID PRIMARY KEY,
  world_id UUID NOT NULL,
  source_entity_id UUID NOT NULL REFERENCES entity(id),
  target_entity_id UUID NOT NULL REFERENCES entity(id),
  relationship_type TEXT NOT NULL,
  weight NUMERIC,
  payload_jsonb JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### `module_state`

```sql
CREATE TABLE module_state (
  id UUID PRIMARY KEY,
  world_id UUID NOT NULL,
  module_id TEXT NOT NULL,
  scope_type TEXT NOT NULL,
  scope_id UUID,
  schema_version TEXT NOT NULL,
  state_jsonb JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(world_id, module_id, scope_type, scope_id)
);
```

### `event_log`

```sql
CREATE TABLE event_log (
  id UUID PRIMARY KEY,
  world_id UUID NOT NULL,
  source_type TEXT NOT NULL, -- core | module | user | system
  source_id TEXT,
  event_type TEXT NOT NULL,
  canon_status TEXT NOT NULL, -- proposed | committed | rejected | corrected
  visibility TEXT NOT NULL, -- private | shared | public | hidden | system
  payload_jsonb JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### `module_installation`

```sql
CREATE TABLE module_installation (
  id UUID PRIMARY KEY,
  world_id UUID NOT NULL,
  module_id TEXT NOT NULL,
  module_version TEXT NOT NULL,
  status TEXT NOT NULL, -- enabled | disabled | archived
  config_jsonb JSONB NOT NULL DEFAULT '{}',
  installed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(world_id, module_id)
);
```

## Example: Battle Module State

A JRPG battle module can store its internal state as JSONB without forcing the core schema to understand HP, turn order, stances, or battle resources.

```json
{
  "encounter_id": "enc_123",
  "turn_order": ["player_1", "bandit_01", "mara"],
  "active_turn": "player_1",
  "combatants": {
    "player_1": {
      "hp": 34,
      "max_hp": 40,
      "stance": "guarded"
    },
    "bandit_01": {
      "hp": 12,
      "max_hp": 25,
      "status": "bleeding"
    }
  }
}
```

The core does not need to know what HP means. The battle module owns that state.

However, if the battle result should affect canon, the module must produce a proposed event:

```json
{
  "event_type": "combat.action_resolved",
  "source_type": "module",
  "source_id": "battle.jrpg",
  "payload": {
    "actor_entity_id": "player_1",
    "target_entity_id": "bandit_01",
    "result": "hit",
    "damage": 12,
    "status_change": "wounded"
  }
}
```

The World Core then validates and commits/rejects the proposed event.

## Why Not MongoDB First?

MongoDB would make early iteration feel easier, but it creates risk around the exact parts of the product that need the most discipline.

A Mongo-first approach risks postponing hard decisions about:

- canonical event ordering
- correction history
- relationship queries
- knowledge ownership
- module dependency validation
- timeline consistency
- transactional canon commits
- audit trails
- replay/debug tooling
- future relational analytics

Migrating later from flexible documents to a relational, event-sourced world core would likely require a major rewrite of assumptions, not only a data migration.

## Why Postgres + JSONB Works Better

Postgres gives the product:

- relational integrity for core world entities
- transactions for canon commits
- strong joins across entities, scenes, relationships, timelines, and knowledge states
- JSONB flexibility for evolving data
- schema-versioned module state
- auditability
- future pgvector support for memory retrieval
- easier BI/debug/ops queries
- one DB technology for MVP instead of Mongo + later migration

This avoids premature rigidity without losing future discipline.

## Consequences

### Positive Consequences

- The authoritative world core starts on the long-term database foundation.
- The team avoids a painful Mongo-to-Postgres migration later.
- Modules can still evolve quickly using namespaced JSONB state.
- The event log remains queryable and auditable.
- Correction, replay, and debugging become easier.
- Relationship and knowledge-boundary queries remain practical.
- pgvector can be added without introducing a separate vector database too early.

### Negative Consequences

- Early development requires more schema thinking than MongoDB.
- JSONB validation discipline must be introduced early.
- Developers must avoid using JSONB as an uncontrolled dumping ground.
- Some module data may later need migration from JSONB into relational tables.
- Poor indexing choices on JSONB can create performance issues.

## Guardrails

1. **Core identifiers must be relational.**
   Entities, scenes, locations, worlds, modules, and events must have stable IDs in relational tables.

2. **All JSONB payloads need schema versions.**
   Example: `schema_version = "battle_state.v1"`.

3. **Modules own their state namespace.**
   A module should not write into another module's state directly.

4. **Modules cannot directly commit canon.**
   They propose events. The core validates and commits.

5. **Promote only stable JSONB shapes into relational tables.**
   Do not over-normalize too early.

6. **Index deliberately.**
   Frequently queried JSONB keys should either receive expression indexes or be promoted to columns.

7. **Audit all canon-relevant writes.**
   Every meaningful state mutation should link back to an event, module action, user action, or system process.

## Migration Strategy Inside Postgres

Instead of Mongo-to-Postgres migration, use progressive hardening:

```text
Phase 1: JSONB-heavy experimental state
Phase 2: identify repeated query patterns
Phase 3: add generated columns / indexes
Phase 4: promote stable structures to relational tables
Phase 5: preserve old JSONB payloads for audit/history
```

Example:

```text
stats module starts with JSONB
  ↓
combat queries stabilize
  ↓
extract hp/current_status into indexed generated columns
  ↓
if widely reused, create dedicated module-owned relational tables
```

## Alternatives Considered

### Option A: MongoDB first, migrate later

Rejected.

Good for early schema flexibility, but risky for canon, relationships, event sourcing, correction, and future migration.

### Option B: Strict Postgres only

Rejected.

Too rigid for an evolving product with unknown module mechanics and experimental gameplay systems.

### Option C: Postgres core + JSONB flexibility

Accepted.

Best balance between continuity infrastructure and early product discovery.

### Option D: Polyglot persistence from day one

Rejected for MVP.

Using Postgres, MongoDB, vector DB, Redis, and object storage all at once would create unnecessary operational complexity before the core loop is validated.

## Decision Summary

DreamChat should use **Postgres as the authoritative database from day one**, with **JSONB as the flexibility layer** for evolving module and world payloads.

This supports the product's need for persistent continuity, memory, canon correction, relationship tracking, knowledge boundaries, and module extensibility, without locking the team into a rigid schema too early.

## Confidence

**87%**

This is a strong recommendation because it preserves the long-term architecture while keeping V1 flexible. The remaining uncertainty is mostly about how quickly module schemas stabilize and how much experimental state should remain JSONB versus becoming relational.

## Follow-Up Recommendations

- Define the first version of the core relational schema.
- Define JSONB schema-versioning rules.
- Define module-state ownership rules.
- Define event-log payload validation rules.
- Create example schemas for:
  - relationship system
  - knowledge state
  - JRPG battle module
  - investigation/evidence module
- Revisit this ADR after the first playable PoC.
