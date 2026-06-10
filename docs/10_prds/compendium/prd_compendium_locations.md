# PRD: Compendium — Locations

> **Status:** Draft (formalized 2026-06-10) | **Owner:** TBD | **In MVP scope**
> **Depends on:** Canon Engine, Glossary, Rules Register (`00_strategy/06`), `00_time_and_mutability_rules.md`, `01_epistemic_type_canonical_enum.md`, Actors PRD (Key Actors links).

## 1. Problem Statement
**The problem:** Places accumulate meaning faster than users can hold it — who was seen where, what changed, what lies inside. Without a perception-bound place reference, users either lose spatial continuity or get a map editor / omniscient world inspector, both of which the product explicitly is not.
**Who is affected:** the returning player (re-orient: where does this place sit, why does it matter now); the investigating player (what do I *know* happens here, from which sources).

## 2. Goals
**Impact hypothesis:** if Location pages make remembered places browsable and atmospheric without leaking hidden geography, users will trust that *places change and persist*, reinforcing the world-continuity promise. Signals: Location page opens per session; navigation via inline links (page→actor→location chains); zero hidden-area leaks.

## 3. Scope / Non-Goals
**In scope (MVP):** Location list + page: visual, perceived name, synthesis, single "Part of" hierarchy line, Known areas inside (known sublocations only), Key Actors with context, collected knowledge by topic, recent-state note with decay language, inline links, report entry.
**Non-goals:** map editor; world map rendering; hierarchy shown more than once (breadcrumb+tree+panel); revealing hidden rooms/secret routes/undiscovered areas; occupancy dashboards; relationship graphs.

## 4. Acceptance Criteria
1. **Perception-bound (B-1):** page payload contains only known/perceived data; hidden sublocations and secret routes are absent from the response, not hidden by UI.
2. **One hierarchy expression (C-12):** exactly one "Part of" line; the same connection never repeats as tree + breadcrumb + panel.
3. **Known areas inside** lists only sublocations with qualifying perception records (valid paths incl. common knowledge — B-2); "+N more" style affordances never enumerate unknown areas.
4. **Key Actors** entries carry context lines and inline links; groups/institutions appear here (no separate "Linked to" panel); hidden associations never surface without qualifying records.
5. **Decay language** for stale state ("Last known…", per-record ticks rendered as display labels) — never hiding known info (Decay mechanic, Glossary).
6. **Time model (B-5)** and **canonical epistemic enum (G-4)** compliance.
7. **Vocabulary (GA-2/F-1):** UI says Artifacts, Actors, Known World — never possessions/entities.
8. **Mock parity, with exceptions:** `mock_compendium_location_dawnfall_market.png` is the reference **except** nav label "Possessions"→Artifacts and no Relationships nav item.

## 5. Open Questions
- Sublocation card visuals: generated per known area, or location-level art only? (Image Platform / Bridge doc.)
- "View all" threshold for known areas.

---

# Spec Body (the validated model — register-compliant, unchanged otherwise)

# DreamChat Locations Compendium Model

## Purpose

This document defines the Locations section of the DreamChat Compendium.

Locations are part of the optional world workspace layer. They help the user understand remembered and known places without turning the product into a map editor, dashboard, or omniscient world-state inspector.

The user-facing Location page should answer:

> What do I know about this place, where does it sit in the known world, what known sub-places does it contain, who or what is associated with it, and why might it matter now?

## Core Definition

A Location is a place, region, room, structure, territory, route, district, settlement, world area, or spatial container that can hold events, actors, artifacts, memories, tensions, and world changes.

A Location can be physical, virtual, symbolic, institutional, or genre-specific, as long as the world treats it as a place where things can happen or be situated.

Examples:

```text
Dawnfall Market
Dawnfall City
Morning Light Empire
Conference Room 4B
Helix Station Docking Ring C
The Old Family House
The Ash Archive
A hidden chat server
A dream corridor
A government district
```

## Core Product Rule

Locations are canonical world places internally, but player-facing Location pages are perception-bound.

Internally, the product may know the true geography, hidden rooms, secret routes, current occupants, private changes, and backstage state.

Externally, normal play should only show what the current perspective knows, remembers, has observed, has been told, can infer, or has access to.

This follows the product principle that play mode shows the known/perceived world, not omniscient world state.

## What Locations Are Not

A Location is not an Actor. It does not act by itself unless the setting explicitly treats it as an agent.

A Location is not a Artifact. Objects, documents, tools, weapons, letters, keys, or inventory-like things belong in Artifact records.

A Location is not a Timeline Event. Events can happen at locations, but the event itself is not the place.

A Location is not a Relationship. Relationships can be associated with a place, but should not be represented as locations.

## Location Hierarchy

Locations support hierarchy and containment.

A place can be part of a larger place, and it can contain smaller known places.

Example:

```text
Morning Light Empire
  -> Dawnfall City
      -> Lower District
          -> Dawnfall Market
              -> East Stalls
              -> Merchant Row
              -> Old Fountain
              -> Fox Alley
```

For MVP, the user-facing Location page should not duplicate this hierarchy in multiple places.

Use only one clear hierarchy expression:

> **Part of:** one level up, shown as a subtitle or small contextual line near the title.

Example:

```text
Dawnfall Market
Part of Dawnfall City
```

Do not show the same connection again as a hierarchy tree, breadcrumb block, and linked panel at the same time.

The hierarchy can still exist internally, but the UI should avoid repeating it.

## Known Areas Inside

The strongest part of the Location UX is the "Known areas inside" section.

This section shows sublocations the user perspective knows about.

It should feel visual, browsable, and atmospheric, not like a raw folder tree.

Example:

```text
Known areas inside
- East Stalls: spice and exotic goods.
- Merchant Row: shops, appraisers, and money changers.
- Old Fountain: center of the square, popular meeting spot.
- Fox Alley: known service route behind the eastern stalls.
```

MVP behavior:

- show only known/perceived sublocations
- do not reveal hidden rooms, secret routes, or undiscovered areas
- allow sublocation cards or inline links
- support "view all" only if there are many known areas
- keep this section visually richer than a plain list

## Key Actors

The Location page should include **Key Actors**.

This replaces a separate "Linked to" panel.

Key Actors can include people, NPCs, organizations, groups, institutions, families, factions, companies, cults, gangs, governments, or any Actor associated with the place.

Examples:

```text
Key Actors
- Seren: seen near the east gate.
- Kael: kept watch near the market edge.
- Liora: merchant contact in the western stalls.
- Dark Foxes: rumored to use the market as a message route.
```

Rules:

- include organizations and groups here, not in a separate links section
- actor names should be inline links to Actor pages
- include small context, not just names
- avoid relationship graphs or network panels in MVP
- do not expose hidden associations unless the user perspective knows or suspects them through valid information

## Remove Key Links / Linked To

The Location page should not include a generic "Linked to" section in MVP.

It duplicates information already expressed through:

- Part of
- Known areas inside
- Key Actors
- Collected Knowledge
- Inline links

A generic link panel makes the page feel more like a database inspector and less like a living-world compendium.

Use inline links instead.

Example:

```text
Market sellers claim the Dark Foxes pass messages through the east side of Dawnfall Market.
```

Here **Dark Foxes** links to an Actor page and **Dawnfall Market** links to the current Location page if referenced elsewhere.

## Collected Knowledge

Locations should have Collected Knowledge, grouped by topic.

This is source-based and perception-bound, similar to Actors, but the topics are about place meaning, history, use, public reputation, access, danger, changes, and remembered events.

Example:

```text
Collected Knowledge

The market's role
  Market gossip -- Day 3, Morning
  Market sellers say important news and messages flow through here faster than the city watch.

  Your observation -- Day 3, Morning
  You noticed several couriers using the eastern exits and passing sealed notes.

Dark Foxes activity
  Rumor -- Day 3, Late Morning
  A fixer claimed the Dark Foxes use the market to move information and small valuables.

  Your observation -- Day 3, Afternoon
  You saw Seren speak with a cloaked figure near the Old Fountain. They exchanged a pouch.

Prices and trade
  Market record -- Day 3
  Spice prices have risen sharply in the last two days.
```

Collected Knowledge should not become a raw event log. It should group information by meaning.

## Last Known

Locations should show a last-known state, but this must not pretend to be live omniscient truth.

Example:

```text
Last known
Day 3, Morning
Busy and loud. Merchants shouted prices while city guards moved in routine patrols.
```

If time has passed, the page may show uncertainty:

```text
Last known
Three weeks ago
The east gate was under repair. This may no longer be current.
```

This aligns with decay as review pressure, not forgetting. The location is remembered, but the current state may deserve review before use.

## UX Layout Recommendation

The Location page should use the same elegant, image-led Compendium style as the Actor and Timeline references.

Recommended layout:

```text
Compendium > Locations > Dawnfall Market

Dawnfall Market
Part of Dawnfall City

Short synthesis paragraph
A crowded trade hub in the lower district. Information moves as freely as coin.

Last known
Day 3, Morning. Busy, loud, bright, unpredictable.

Known areas inside
[East Stalls] [Merchant Row] [Old Fountain] [Fox Alley]

Key Actors
Seren, Kael, Liora, Dark Foxes

Collected Knowledge
Grouped by topic with source and timing.
```

Avoid:

```text
- Separate location hierarchy tree and Part of and Linked To repeating the same connection
- Generic Key Links panel
- Heavy dashboard boxes
- Omniscient map state
- Revealing hidden sublocations too early
- Treating locations as live GPS maps
```

## Recommended Schema

```text
Location Canon
  location_id
  world_id
  canonical_name
  canonical_type
  parent_location_id optional
  canonical_description
  hidden_truth_fields
  contained_location_ids
  linked_actor_ids
  linked_artifact_ids
  linked_canon_records
  created_at_tick
  as_of_tick (derived from latest committed event/version — never stored as a mutable field)

Location Perception
  location_perception_id
  location_id
  holder_id
  holder_type: user | npc | group | public | system-perspective
  perceived_name
  perceived_summary
  perceived_parent_location_id optional
  known_sublocation_ids
  current_synthesis
  first_perceived_tick
  latest_version_tick (derived)
  visibility_scope
  linked_perception_records

Location Knowledge Record
  knowledge_record_id
  location_id
  location_perception_id
  linked_canon_record_id optional
  topic
  epistemic\_type (canonical enum — see 01\_epistemic\_type\_canonical\_enum.md) + source metadata
  source_actor_id optional
  source_text / source_reference
  occurred_at_tick + display_label
  content
  linked_actors
  linked_locations
  linked_artifacts
  linked_timeline_records
  confidence_metadata optional
  uncertainty_metadata optional

Location Page Projection
  perceived_name
  part_of_one_level_up
  current_synthesis
  last_known_status
  known_areas_inside
  key_actors
  collected_knowledge_groups
  inline_links
```

## MVP Location Page Includes

```text
- Perceived location name
- One-level-up Part of subtitle
- Short synthesis
- Last known state
- Known areas inside
- Key Actors, including organizations and groups
- Collected Knowledge grouped by topic
- Source/timing for knowledge records
- Inline links to Actors, Locations, Artifacts, and Timeline records
- Hidden canon exclusion during normal play
```

## MVP Location Page Excludes

```text
- Full omniscient map
- Hidden sublocations unless discovered
- Generic Linked To panel
- Separate repeated hierarchy tree
- Coordinate systems
- Live current occupants unless known/perceived
- Relationship graphs
- Heavy dashboard state
- Direct database correction buttons
```

## Final Product Rule

Locations are the Compendium's perception-bound records of places.

Internally they link to canonical Location records, Canon Events, Perception Records, Actors, Artifacts, and Timeline Records.

Externally they show where the place sits in the known world, what known sub-places it contains, which Actors are meaningfully associated with it, and what this perspective has collected, remembered, heard, observed, or inferred about it.

The Location page should help the user return to a place with context, not manage a world database.
