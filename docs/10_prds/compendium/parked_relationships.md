> ⚠️ **AMENDMENT (2026-06-10, rev 2):** The "Relationship to you" Actor-page surfacing described below is **removed from the MVP UI entirely** — no panel, no field, no synthesis. Relationships remain modeled internally (canon + perception); relationship-flavored information reaches the user only as ordinary sourced Collected Knowledge records via valid in-world paths. The system never authors the player character's own stance (player interiority rule — see Glossary). Meters/sliders remain banned. Time fields below were patched per `00_time_and_mutability_rules.md`.

---

# DreamChat Relationships — Parked Topic

## Status

Parked for later Compendium / Memory / Social Context PRDs.

Relationships should be modeled internally now, but should **not** become a top-level Compendium section in the MVP.

In MVP, relationship information appears inside Actor pages as part of the user's perception-bound understanding of an Actor.

---

## Core Idea

A **Relationship** is a meaningful connection between two Actors.

It may represent trust, suspicion, loyalty, obligation, debt, rivalry, intimacy, fear, dependency, alliance, hierarchy, shared history, family, employment, political alignment, membership, betrayal, protection, hostility, secrecy, or any other persistent social/world connection.

Relationships are part of what makes the world feel continuous. The PoC already requires that entities preserve relationship state and knowledge boundaries over long gaps.

---

## MVP UX Decision

Relationships are **not** a top-level Compendium section in MVP.

MVP Compendium structure:

```text
Compendium
  ├── Timeline
  ├── Actors
  ├── Locations
  ├── Artifacts
  └── Report / Corrections
```

Relationship information appears inside Actor pages as:

```text
Relationship to you
Known relationships
Relationship-related collected knowledge
Inline links to other Actors
```

Reason:

- A top-level Relationship section risks becoming a graph dashboard too early.
- Most relationship understanding is meaningful only in relation to a specific Actor.
- The Actor page is already the user's main surface for understanding who or what matters.
- MVP should stay readable, not managerial.

---

## Internal Model

Even if Relationships are not exposed as a main UX section, the system should still model them internally.

```text
Relationship Canon
  relationship_id
  world_id
  actor_a_id
  actor_b_id
  canonical_relationship_state
  linked_canon_records
  created_at_tick
  as_of_tick (derived from latest committed event/version — never stored as a mutable field)

Relationship Perception
  relationship_perception_id
  relationship_id
  holder_id
  holder_type: user | npc | group | public | system-perspective
  perceived_relationship_summary
  linked_perception_records
  first_perceived_tick
  latest_version_tick (derived)
  visibility_scope

Relationship Perception Record
  perception_record_id
  relationship_id
  linked_canon_record_id optional
  holder_id
  topic
  epistemic\_type (canonical enum — see 01\_epistemic\_type\_canonical\_enum.md) + source metadata
  source_actor_id optional
  occurred_at_tick + display_label
  content
  linked_actors
  linked_locations
  linked_artifacts
  linked_timeline_records
```

---

## Canon vs Perception

Relationship Canon is the objective relationship state in the world model.

Relationship Perception is what a given perspective believes, knows, suspects, heard, inferred, or misunderstands about that relationship.

Example:

```text
Relationship Canon
Seren secretly reported to the Dark Foxes for three months, then broke contact.

User perception
Seren may have some connection to the Dark Foxes, but the nature of it is unclear.

Public perception
Seren is independent.

Dark Foxes perception
Seren is a former courier asset and possible liability.
```

Normal play should show the perception-bound version, not the hidden canon.

---

## Actor Page Surfacing

In MVP, relationship information should be surfaced inside Actor pages.

Example:

```text
Seren

Relationship to you
Cautious but engaged. Seren shares information selectively and watches your reactions closely. Trust is provisional -- earned, not given.

Known relationships
Seren is said by some market sellers to have worked with the Dark Foxes. Kael claims she only worked with them once. The exact nature of the connection is unclear from your current knowledge.

Collected Knowledge

Dark Foxes connection
  Market rumor -- Day 3, Late Morning
  A fixer in the docks called Seren "Fox-ears" and said she knows too much about the Dark Foxes routes.

  Kael's account -- Day 3, Noon
  Kael believes Seren once passed information to someone aligned with the Dark Foxes.

  Your observation -- Day 3, Afternoon
  You saw Seren pass a sealed note to a cloaked figure. The seal matched the one found on Dark Foxes orders.
```

Use inline links only in MVP.

Do not add:

```text
- relationship graph
- related actors panel
- relationship meter
- trust/fear numeric dashboard
- social network visualization
- also check section
```

---

## Why This Is Parked

A full Relationship section may become useful later if the world supports:

- large casts
- multiple users in the same world
- faction politics
- family trees
- institutional hierarchies
- relationship history visualizations
- social network inspection
- creator/debug tools
- advanced correction workflows

But for MVP, this would add too much structural complexity and risk making the Compendium feel like a database tool.

---

## Later Relationship Feature Direction

A later dedicated Relationships section could show:

```text
- relationship map
- relationship history
- conflicting perceptions of a relationship
- who knows what about a relationship
- public vs private relationship knowledge
- relationship changes over in-world time
- source trails for each relationship belief
- relationship consequences in timeline
```

It should remain perception-bound, not omniscient, unless the user is in creator/debug mode.

---

## Product Rule

> Relationships should be modeled internally from the start, but not promoted to a top-level user-facing Compendium feature in MVP. In normal play, relationship information appears inside Actor pages as perception-bound collected knowledge.

This preserves relationship continuity without turning the MVP into a graph-management interface.
