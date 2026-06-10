# DreamChat Modular Architecture — World Engine Modules

## 1. Purpose

This document defines the proposed technical architecture for DreamChat’s modular world engine.

The goal is to make the product extensible without turning the backend into a fragile collection of disconnected services.

The product should support optional gameplay/world systems such as:

- Stats
- JRPG Battle
- Inventory
- Crafting
- Relationships
- Factions
- Economy
- Reputation
- Skills
- Rulesets
- Mature-content controls
- Creator tools

These systems should be easy to add, disable, replace, or extend without breaking the core world runtime.

The target architecture is:

> Modular monolith + module registry + dependency graph + domain events + capability ports.

This is not a microservice-first architecture.

The product should remain world-state-first. Modules must attach to the same authoritative world substrate instead of creating isolated feature islands.

---

## 2. Product Architecture Context

The high-level app architecture remains:

```text
frontend
world-backend
image-platform
```

## 2.1 Frontend

The frontend owns presentation.

It should handle:

- scene layout
- participant strip
- conversation/narration panel
- aux context sidebar
- timeline views
- correction UX
- theme system
- local UI state
- streaming response rendering
- module-provided UI slots

The frontend should not own world truth.

It should not decide:

- what becomes canon
- who knows what
- whether an entity can act
- whether a module action succeeds
- whether a correction is valid
- whether backstage updates happen

## 2.2 World Backend

The world backend owns meaning.

It should own:

- worlds
- scenes
- entities/NPCs
- locations
- relationships
- timeline
- event log
- canon state
- knowledge boundaries
- memory/canon correction
- module runtime
- module dependency validation
- scene orchestration
- backend-facing module contracts

This is where the module system belongs.

## 2.3 Image Platform

The image platform already exists as its own service.

It should be treated as an external capability used by the world backend.

The image platform owns:

- image jobs
- image generation lifecycle
- image provider adapters
- generated image metadata
- image storage
- image variants
- regeneration history
- cost tracking

It should not own world truth.

The world backend may request images and then attach returned asset references to scenes, entities, locations, or artifacts.

---

## 3. Core Architecture Principle

The core rule is:

> The world engine is stable. Modules extend it without owning it.

The core backend should provide the substrate that every module uses:

- canonical world state
- event log
- timeline
- scene lifecycle
- entity lifecycle
- canon write pipeline
- correction pipeline
- knowledge boundary rules
- module registry
- dependency graph
- domain event bus
- permissions
- background job scheduling

Modules should not bypass this substrate.

A module may create events, state, UI surfaces, and capabilities, but it must do so through core contracts.

---

## 4. What Is a Module?

A module is a pluggable world system that adds optional behavior, state, actions, UI surfaces, or rules.

Examples:

```text
Stats Module
Battle JRPG Module
Inventory Module
Crafting Module
Faction Module
Economy Module
Relationship Module
Reputation Module
```

A module may contribute:

- domain logic
- entity attributes
- world attributes
- scene actions
- validation rules
- canon event types
- timeline renderers
- memory/canon write policies
- prompt fragments
- LLM tools
- background jobs
- UI contracts
- database migrations
- admin/creator controls

A module should be installable per world, not necessarily globally active for every world.

Example:

```text
World A: Stats + Battle + Inventory
World B: Relationships + Reputation + Factions
World C: No battle, no stats, only narrative continuity
```

---

## 5. Module Lifecycle

Modules should support at least three lifecycle states.

## 5.1 Available

The module exists in the backend codebase and can be enabled for a world.

It is not necessarily active in every world.

## 5.2 Enabled

The module is active for a specific world.

It can:

- expose actions
- create module state
- contribute UI slots
- write events through the canon pipeline
- run background jobs
- provide capabilities to other modules

## 5.3 Disabled

The module no longer runs for the world, but historical data remains readable.

This is important.

If a world once used Battle and later disables it, old battle events should not disappear from the timeline.

The module should become inactive, not erased.

## 5.4 Uninstalled / Migrated Out

Full removal should be treated as an advanced operation.

It may require:

- data export
- historical event preservation
- migration scripts
- timeline renderer fallbacks
- warning prompts
- creator confirmation

For the first version, prefer **disable** over **delete**.

---

## 6. Module Manifest

Every module should define a manifest.

The manifest allows the backend to understand:

- module identity
- version
- dependencies
- capabilities provided
- capabilities consumed
- event types emitted
- actions exposed
- UI slots contributed
- DB migrations required
- compatibility rules

Example:

```ts
export const BattleJrpgModule = {
  id: "battle.jrpg",
  name: "JRPG Battle",
  version: "0.1.0",

  dependsOn: [
    "stats.core"
  ],

  optionalDependencies: [
    "inventory.core",
    "relationships.core"
  ],

  provides: [
    "scene_action:battle.start",
    "scene_action:battle.attack",
    "scene_action:battle.defend",
    "scene_action:battle.use_item",
    "canon_event:battle.started",
    "canon_event:battle.turn_resolved",
    "canon_event:battle.ended",
    "ui_slot:scene.bottom_panel_addon",
    "timeline_renderer:battle.event"
  ],

  consumes: [
    "entity.stats",
    "scene.participants",
    "timeline.write",
    "canon.write"
  ],

  migrations: [
    "001_create_battle_encounters",
    "002_create_battle_turns"
  ],

  canDisable: true,
  disableMode: "read_only_history"
}
```

---

## 7. Dependency Graph

The module runtime should maintain a dependency graph.

This graph answers questions such as:

```text
Can this world enable Battle?
No, because Stats is missing.

Can this world disable Stats?
No, because Battle depends on it.

Can this world disable Battle?
Yes, but existing battle events remain readable.
```

## 7.1 Dependency Types

### Required Dependency

The module cannot run without it.

Example:

```text
Battle requires Stats.
```

### Optional Dependency

The module can run without it, but becomes richer if present.

Example:

```text
Battle optionally uses Inventory for equipment bonuses and consumables.
Battle optionally uses Relationships for morale or loyalty effects.
```

### Soft Integration

A module listens to events from another module but does not require it.

Example:

```text
Reputation listens to battle.ended if Battle exists.
If Battle does not exist, Reputation still works.
```

---

## 8. Capability-Based Communication

Modules should not call each other directly.

Instead, they should communicate through capabilities exposed by the core runtime.

Bad:

```ts
battleModule.statsService.getStrength(entityId)
```

Better:

```ts
capabilities.get("entity.stats").getStat(entityId, "strength")
```

Or:

```ts
world.queryCapability("entity.stats", {
  entityId,
  stat: "strength"
})
```

This keeps modules replaceable.

The Battle module should not care whether Stats are implemented by:

- a simple numeric table
- a D&D-style ruleset
- a JRPG-style stat curve
- a custom creator-defined formula
- a future external rules engine

It should only depend on the capability contract.

---

## 9. Domain Events

Modules should communicate primarily through domain events.

Example events:

```text
battle.started
battle.turn.resolved
battle.ended
entity.health.changed
entity.defeated
entity.status_effect.added
item.consumed
relationship.trust.changed
faction.reputation.changed
canon.event.created
scene.segment.closed
memory.episode.created
```

The core event log remains the source of truth.

Modules may emit events, but the core decides how those events become canon, timeline history, memory candidates, or UI updates.

## 9.1 Event Requirements

Every domain event should include:

```json
{
  "id": "evt_123",
  "world_id": "world_001",
  "scene_id": "scene_456",
  "module_id": "battle.jrpg",
  "type": "battle.turn.resolved",
  "created_at": "2026-06-05T12:00:00Z",
  "in_world_time": "Day 12, Evening",
  "actor_entity_ids": ["ent_player", "ent_kael"],
  "target_entity_ids": ["ent_bandit_01"],
  "payload": {},
  "canon_status": "candidate",
  "visibility": "known_to_participants"
}
```

## 9.2 Canon Status

Events should not all immediately become full canon.

Possible statuses:

```text
candidate
accepted
corrected
rejected
present_forward_corrected
archived
```

This supports the correction window and future correction logic.

---

## 10. Canon Write Pipeline

Modules must write through the canon pipeline.

They should not directly mutate authoritative world truth without going through core validation.

Example:

```text
Battle Module resolves attack
  -> emits battle.turn.resolved
  -> requests entity.health.changed
  -> core validates
  -> canon pipeline accepts
  -> event log writes
  -> memory candidate created
  -> timeline updated
  -> frontend receives state patch
```

This protects:

- memory continuity
- correction UX
- knowledge boundaries
- timeline accuracy
- backstage updates
- auditability

## 10.1 Why This Matters

If each module writes its own truth separately, the world fragments.

Examples of bad fragmentation:

```text
Battle says Entity A is dead.
Relationship module still treats Entity A as alive.
Narrator retrieves old memory saying Entity A is traveling.
Timeline has no death event.
Correction UI cannot explain what happened.
```

The canon pipeline prevents this.

---

## 11. Module Data Ownership

Modules may own their own tables, but not their own isolated truth.

Example:

```text
stats_entity_stats
battle_encounters
battle_turns
inventory_items
inventory_ownership
```

However, module tables should reference core IDs:

```text
world_id
scene_id
entity_id
event_id
canon_event_id
```

This allows module data to remain connected to the core world substrate.

## 11.1 Module Tables

Example Battle tables:

```sql
battle_encounter (
  id,
  world_id,
  scene_id,
  status,
  started_event_id,
  ended_event_id,
  created_at
)

battle_turn (
  id,
  encounter_id,
  actor_entity_id,
  action_type,
  target_entity_id,
  result_payload,
  event_id,
  created_at
)
```

Example Stats table:

```sql
entity_stats (
  entity_id,
  world_id,
  stat_key,
  base_value,
  current_value,
  source_module_id,
  updated_event_id
)
```

---

## 12. Frontend Module Slots

The frontend should support module slots.

These are UI extension points where enabled modules can render additional UI.

Possible slots:

```text
scene.top_overlay
scene.participant_badge
scene.bottom_panel_addon
scene.right_aux_card
entity.sheet_tab
timeline.event_renderer
known_world.card_renderer
correction.editor
creator.module_settings
```

Example Battle module UI contributions:

```text
scene.bottom_panel_addon -> Battle action controls
scene.right_aux_card -> Active encounter summary
entity.sheet_tab -> Combat stats
timeline.event_renderer -> Battle result summary
correction.editor -> Fix battle result
```

The frontend renders module-provided state, but it does not calculate authoritative outcomes.

---

## 13. Module Action Flow

Example: user clicks Attack.

```text
frontend
  POST /worlds/{worldId}/modules/battle.jrpg/actions/attack

world-backend
  1. Verify module is enabled for this world.
  2. Verify dependencies are enabled.
  3. Verify action is valid in the current scene.
  4. Load scene, participants, and relevant entity state.
  5. Query Stats capability.
  6. Query optional Inventory capability if enabled.
  7. Resolve battle action.
  8. Emit domain events.
  9. Pass events through canon pipeline.
  10. Update timeline and memory candidates.
  11. Build narration context.
  12. Return UI state patch + narration payload.
```

The narrator may dramatize the result.

The narrator should not invent the mechanical result.

---

## 14. Example: Stats Module

## 14.1 Purpose

The Stats module provides structured attributes for entities.

It can support:

- battle
- skill checks
- crafting
- training
- exhaustion
- social influence
- physical limitations
- magical ability
- difficulty systems

## 14.2 Provides

```text
entity.stats
stat.check
stat.modify
stat.derive
```

## 14.3 Example Stats

```json
{
  "entity_id": "ent_kael",
  "stats": {
    "strength": 14,
    "agility": 11,
    "endurance": 15,
    "focus": 9,
    "presence": 12
  }
}
```

## 14.4 Important Rule

Stats should not replace characterization.

Stats are mechanical support, not full identity.

An entity's actions should still be constrained by:

- knowledge
- competence
- personality
- motivation
- relationships
- setting context
- risk tolerance

---

## 15. Example: JRPG Battle Module

## 15.1 Purpose

The JRPG Battle module adds structured battle scenes.

It may simulate:

- turn order
- attacks
- defense
- status effects
- abilities
- party members
- enemies
- victory/defeat
- rewards
- consequences

## 15.2 Required Dependencies

```text
stats.core
```

## 15.3 Optional Dependencies

```text
inventory.core
relationships.core
skills.core
reputation.core
```

## 15.4 Provides

```text
battle.encounter
battle.turn
battle.action.attack
battle.action.defend
battle.action.item
battle.action.ability
battle.result
```

## 15.5 Emits

```text
battle.started
battle.turn.resolved
entity.health.changed
entity.status_effect.added
entity.defeated
battle.ended
reward.granted
```

## 15.6 Product Guardrail

Battle should not hijack the entire product.

A world may use Battle heavily, lightly, or not at all.

The core app remains a persistent RPG world experience, not a mandatory combat simulator.

---

## 16. Module Compatibility with Memory and Backstage

Modules must integrate with memory and backstage carefully.

Example:

```text
Battle ended with Kael injured.
```

This may affect:

- Kael's physical state
- Kael's future availability
- relationship with the player
- faction reputation
- public rumor
- future backstage decay
- entity competence
- timeline recap

But the Battle module should not directly update all of those systems.

It should emit events.

Other modules or core processors decide what follows.

## 16.1 Bounded Propagation

Module events should respect bounded propagation.

A battle result may create immediate changes, but should not automatically trigger full-world recomputation.

Possible effects:

```text
Radius 0: update battle participant state
Radius 1: update directly connected relationships or location status
Radius 2+: convert to background pressure, future hook, or review pressure
```

This protects performance and coherence.

---

## 17. Module Compatibility with Corrections

Every module that writes canon must support correction.

Example correction:

```text
The user corrects: Kael was not present in this battle.
```

The system must be able to identify:

- which module created the event
- which entities were affected
- which canon records were written
- which UI/timeline records should be corrected
- whether the correction is inside the correction window
- whether it is present-forward only

Module events should therefore be traceable.

Bad:

```text
health = 0
```

Better:

```text
health changed from 20 to 0 because event battle.turn.resolved/evt_123 dealt 20 damage.
```

Traceability is required for correction UX.

---

## 18. Suggested Backend Folder Structure

```text
world-backend/
  src/
    core/
      world/
      scene/
      entity/
      canon/
      timeline/
      memory/
      backstage/
      correction/
      module-runtime/
      event-bus/
      permissions/

    ports/
      llm.port.ts
      image-platform.port.ts
      storage.port.ts
      queue.port.ts
      embedding.port.ts
      moderation.port.ts

    adapters/
      llm/
      image-platform/
      storage/
      queue/
      embedding/
      moderation/

    modules/
      stats/
        manifest.ts
        domain/
        application/
        db/
        prompts/
        ui-contracts/
        tests/

      battle-jrpg/
        manifest.ts
        domain/
        application/
        db/
        prompts/
        ui-contracts/
        tests/

      inventory/
      relationships/
      factions/
      reputation/
      crafting/

    shared/
      types/
      errors/
      utils/
```

---

## 19. Suggested Module Internal Structure

```text
modules/battle-jrpg/
  manifest.ts

  domain/
    battle-encounter.ts
    battle-turn.ts
    battle-action.ts
    battle-result.ts
    battle-rules.ts

  application/
    start-battle.usecase.ts
    resolve-attack.usecase.ts
    resolve-defense.usecase.ts
    end-battle.usecase.ts

  db/
    migrations/
    battle.repository.ts

  prompts/
    battle-narration.fragment.ts
    battle-summary.fragment.ts

  ui-contracts/
    scene-bottom-panel.schema.ts
    right-aux-card.schema.ts
    timeline-renderer.schema.ts

  tests/
    battle-module.spec.ts
    battle-dependencies.spec.ts
    battle-correction.spec.ts
```

---

## 20. API Surface

The backend should expose generic module endpoints.

## 20.1 Module Management

```http
GET /worlds/{worldId}/modules
POST /worlds/{worldId}/modules/{moduleId}/enable
POST /worlds/{worldId}/modules/{moduleId}/disable
GET /worlds/{worldId}/modules/{moduleId}/status
```

## 20.2 Module Actions

```http
POST /worlds/{worldId}/modules/{moduleId}/actions/{actionId}
```

Example:

```http
POST /worlds/world_001/modules/battle.jrpg/actions/attack
```

Payload:

```json
{
  "scene_id": "scene_123",
  "actor_entity_id": "ent_player",
  "target_entity_id": "ent_bandit_01",
  "selected_ability_id": "basic_attack"
}
```

## 20.3 Module UI State

```http
GET /worlds/{worldId}/modules/{moduleId}/ui-state?scene_id=scene_123
```

This allows the frontend to request module-specific UI state without owning module logic.

---

## 21. Validation Rules

Before enabling a module, the backend should validate:

- required dependencies are enabled
- migrations can run
- world type allows this module
- user/creator has permission
- module version is compatible
- no incompatible modules are enabled
- existing world state can support the module

Example:

```text
Cannot enable battle.jrpg:
- Missing required dependency: stats.core
```

Example:

```text
Cannot disable stats.core:
- battle.jrpg depends on stats.core
```

---

## 22. Module Versioning

Modules should be versioned.

Example:

```text
battle.jrpg@0.1.0
battle.jrpg@0.2.0
stats.core@0.1.0
```

A world should record which module version created historical data.

This helps with:

- replay
- debugging
- migration
- timeline rendering
- creator support
- compatibility

Historical events should preserve:

```json
{
  "module_id": "battle.jrpg",
  "module_version": "0.1.0"
}
```

---

## 23. Plugin System Boundaries

Modules should be powerful, but not unrestricted.

A module should not be allowed to:

- directly overwrite unrelated canon
- bypass correction rules
- bypass permissions
- access another world's data
- inject arbitrary prompt instructions globally
- mutate memory without event traceability
- make hidden external calls without core approval
- create unbounded background jobs

All module effects should go through core ports and policies.

---

## 24. MVP Recommendation

For the first technical implementation, build the module system with only two sample modules:

```text
stats.core
battle.jrpg
```

This proves:

- dependencies
- module enable/disable
- capability contracts
- module actions
- module events
- module UI slots
- canon writes
- correction compatibility

Do not build ten modules first.

Build the runtime with two modules and make the contracts clean.

---

## 25. What Not To Do Yet

Avoid these early:

```text
One deployable service per module
Dynamic third-party plugin code execution
Marketplace module installation
Hot-loading arbitrary code
Full ruleset DSL
Uninstall with destructive cleanup
Deep historical rewrites across module data
```

These are later-stage capabilities.

For now, modules should be first-party code packages inside the backend repo.

---

## 26. Final Architecture Statement

DreamChat should use a modular world backend where optional gameplay and world systems plug into a shared authoritative substrate.

The backend should not become a pile of disconnected feature services.

The right first architecture is:

> One world backend, many internal modules, strict contracts.

Modules should be:

- discoverable
- dependency-aware
- enableable per world
- disableable without destroying history
- event-driven
- capability-based
- canon-safe
- correction-compatible
- frontend-extensible through UI slots

This lets DreamChat support different world styles over time:

```text
Pure narrative world
JRPG battle world
Political intrigue world
Romance-heavy world
Economy/crafting world
Detective investigation world
Horror survival world
```

without rebuilding the core product each time.

The product remains the same:

> A persistent world that remembers, evolves, and can support different systems without losing continuity.
