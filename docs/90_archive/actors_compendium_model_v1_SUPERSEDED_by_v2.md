# DreamChat Actors Compendium Model

## Purpose

This document defines the **Actors** section inside the DreamChat **Compendium**.

The goal is to give the user a readable, perception-bound view of the people, beings, groups, institutions, and organized forces that matter in the world.

Actors are not shown as omniscient database records. They are shown as the current perspective understands them.

---

## 1. Naming

### User-facing name

**Actors**

### Internal model name

`actor`

### Parent container

**Compendium**

Recommended Compendium structure:

```text
Compendium
  ├── Timeline
  ├── Actors
  ├── Relationships
  ├── Locations
  ├── Possessions
  └── Report / Corrections
```

---

## 2. Actor Definition

An **Actor** is any world participant or force that can act, know, remember, influence, relate, or change over time.

Actors may include:

- people
- NPCs
- companions
- rivals
- user-controlled characters
- creatures or animals with agency
- spirits, gods, intelligences, AIs, monsters, or non-human agents
- groups, families, crews, factions, institutions, companies, governments, courts, cults, criminal networks, schools, labs, or other organized forces

The practical test:

> Can this thing behave like a world participant or force with continuity?

If yes, it is probably an **Actor**.

---

## 3. What Is Not an Actor

The following should not be modeled as Actors by default:

```text
A sword          → Possession / Object
A letter         → Possession / Document
A market         → Location
A rumor          → Perception / Information
A murder         → Canon Event
A friendship     → Relationship
A memory         → Perception / Timeline Record
```

Objects, places, events, memories, and relationships can link to Actors, but they are not Actors themselves.

---

## 4. Core Product Rule

> Actors are canonical world participants internally, but player-facing Actor pages are perception-driven.

This means:

- Internally, an Actor is a canonical world record.
- Externally, the user sees what their current perspective knows, believes, suspects, remembers, heard, observed, inferred, or misunderstood about that Actor.
- The Actor page must not expose hidden truth during normal play.
- Different users, NPCs, groups, or public knowledge layers may hold different perceptions of the same Actor.

---

## 5. Canon vs Perception

### Actor Canon Record

The canonical Actor record represents the objective actor in the world model.

It may include hidden truth, stable identity, internal continuity, and objective world-state links.

### Actor Perception

The Actor Perception represents what a given perspective currently knows or understands about that Actor.

A perspective may belong to:

- the user-controlled character
- another player/user
- an NPC
- a group or institution
- the public
- a rumor network or information source

The Actor page shown to a user is built from that user perspective's Actor Perception and collected perception records.

---

## 6. Recommended Internal Schema

This is a product-level schema, not final implementation detail.

```text
Actor Canon
  actor_id
  world_id
  canonical_type
  canonical_identity
  existence_state
  hidden_truth_fields
  linked_canon_records
  created_at_in_world_time
  updated_at_in_world_time

Actor Perception
  actor_perception_id
  actor_id
  holder_id
  holder_type: user | npc | group | public | system-perspective
  perceived_name
  perceived_role
  current_synthesis
  first_perceived_at
  last_updated_at
  visibility_scope
  linked_perception_records

Perception Record
  perception_record_id
  actor_id
  actor_perception_id
  linked_canon_record_id optional
  topic
  source_type: direct | told_by_actor | told_by_other | rumor | public_record | document | broadcast | inference | observation | institutional_record | unknown
  source_actor_id optional
  source_text / source_reference
  in_world_time
  content
  linked_actors
  linked_locations
  linked_possessions
  linked_timeline_records
  confidence_metadata optional
  uncertainty_metadata optional

Actor Page Projection
  perceived_name
  perceived_role
  current_synthesis
  relationship_to_you
  last_known_status
  known_possessions_or_associated_objects
  collected_knowledge_groups
  inline_links
```

---

## 7. Important Schema Rule

The Actor page should not flatten all perception records into one truth.

It should preserve:

- source
- timing
- contradiction
- uncertainty
- partial knowledge
- indirect knowledge
- rumor existence
- evolving understanding

The system should avoid hard status labels such as:

```text
confirmed
false
disproven
suspected
```

Those labels can bias the user too much.

Instead, the UI should show the collected information and its source context.

Example:

```text
Market rumor
A spice seller claimed Seren killed her last contractor.

Kael's account
Kael said the contractor was seen alive after Seren left the job.

Your direct knowledge
You have not personally seen evidence that Seren killed the contractor.
```

This lets contradiction exist without the UI prematurely deciding what the user should believe.

---

## 8. Actor Page UX Structure

The Actor page should not feel like a database table or dashboard.

It should feel like a readable dossier / living memory surface.

Recommended order:

```text
1. Actor visual / portrait / atmospheric image
2. Perceived name
3. Perceived role
4. Short synthesis paragraph
5. Relationship to you
6. Last known status
7. Known possessions / associated objects
8. Collected Knowledge
```

### 8.1 Perceived Name

The name shown should be what this perspective knows.

The user may not know the Actor's canonical name.

Examples:

```text
Seren
The Fox-Eyed Woman
The Harbor Captain
Unknown Red Court Courier
```

### 8.2 Perceived Role

A short role description based on current perception.

Examples:

```text
Market informant, as you currently understand her.
Former courier, according to Kael.
Unknown faction agent.
Local police liaison.
```

### 8.3 Short Synthesis

A concise natural-language summary of current understanding.

This is not final truth. It is the system's best current synthesis from this perspective's collected knowledge.

Example:

```text
Seren is a quick-witted informant who moves easily through Dawnfall Market's many circles. She trades in rumors, half-truths, and hard-to-find details. Her loyalties are unclear, but she has shown you glimpses of useful, carefully measured truth.
```

### 8.4 Relationship to You

This should remain in the Actor page.

It is a core part of persistent continuity and helps the user understand the Actor's stance toward the user-controlled character.

It should be written naturally, not as a meter.

Example:

```text
Cautious but engaged. Seren shares information selectively and watches your reactions closely. Trust is provisional -- earned, not given.
```

Avoid numeric trust bars or rigid relationship stats in MVP.

### 8.5 Last Known Status

This should combine location and status when possible.

Examples:

```text
Last seen at Dawnfall Market, near the eastern stalls. Day 3, Morning.

Last known to be traveling with the Harbor Police convoy. Two weeks ago, according to Kael.

Current location unknown. Last public mention placed her near the southern docks.
```

Known locations should not become a separate mini-panel in MVP unless the Actor has many meaningful location records.

### 8.6 Known Possessions / Associated Objects

This should be a light section.

Only show objects that matter to how the user understands the Actor.

Examples:

```text
- Worn coin purse with concealed seam
- Folded scrap of charcoal paper
- Silver hairpin etched with a fox motif
```

Objects should be inline-linkable when they have their own Possession / Object record.

---

## 9. Collected Knowledge

The Actor page should use **Collected Knowledge**, not **Collected Perceptions**, as the main UX label.

Reason:

- "Collected Perceptions" is too mechanical.
- The user is not trying to inspect a perception database.
- They are trying to understand what they know about the Actor.

Collected Knowledge is still powered by perception records internally.

### Grouping Rule

Collected Knowledge should be grouped by topic, not by raw timeline/log order.

Example:

```text
Collected Knowledge

The informant
  Direct interaction -- Day 3, Morning
  Seren approached you in Dawnfall Market and offered information for a price.

  Observation -- Day 3, Morning
  She listens more than she speaks and rarely volunteers anything without testing the listener first.

Dark Foxes connection
  Market rumor -- Day 3, Late Morning
  A fixer in the docks called her "Fox-ears" and said she knows too much about their routes.

  Your observation -- Day 3, Afternoon
  You saw her pass a sealed note to a cloaked figure. The seal matched the one found on Dark Foxes orders.
```

### Why Grouping Matters

The user should not read an Actor page as a log.

They should see:

- what topics matter
- what information exists under each topic
- where each piece came from
- how the information conflicts or evolves

The raw event order belongs more naturally in Timeline.

---

## 10. Inline Links

MVP should use inline links only.

No "related actors" or "also check" section yet.

Example:

```text
Seren is said by some market sellers to have worked with the Dark Foxes. Kael claims she only worked with them once.
```

In this example:

- `Dark Foxes` links to the Dark Foxes Actor page.
- `Kael` links to Kael's Actor page.

This keeps the UI clean and avoids turning Actor pages into graph dashboards.

---

## 11. Actor Page MVP Requirements

MVP Actor pages should support:

```text
- Perceived name
- Perceived role
- Short synthesis
- Relationship to you
- Last known status
- Known possessions / associated objects
- Collected Knowledge grouped by topic
- Source/timing for perception records
- Inline links to Actors, Locations, Possessions, and Timeline records
- Hidden canon exclusion during normal play
```

MVP Actor pages should not include:

```text
- Omniscient actor truth
- Hidden motivations unless known/perceived
- Relationship meters
- Stats
- Character sheet editing
- "Also check" or related-actor panels
- Heavy dashboards
- Raw event logs
- Direct database correction buttons
```

---

## 12. Example Actor Page Content

```text
Compendium > Actors > Seren

Seren
Market informant, as you currently understand her.

Seren is a quick-witted informant who moves easily through Dawnfall Market's many circles. She trades in rumors, half-truths, and hard-to-find details. Her loyalties are unclear, but she has shown you glimpses of useful, carefully measured truth.

Relationship to you
Cautious but engaged. Seren shares information selectively and watches your reactions closely. Trust is provisional -- earned, not given.

Last known
Dawnfall Market, near the eastern stalls. Day 3, Morning.

Known possessions / associated objects
- Worn coin purse with concealed seam
- Folded scrap of charcoal paper
- Silver hairpin etched with a fox motif

Collected Knowledge

The informant
Direct interaction -- Day 3, Morning
Seren approached you in Dawnfall Market and offered information for a price.

Observation -- Day 3, Morning
She listens more than she speaks and rarely volunteers anything without testing the listener first.

Dark Foxes connection
Market rumor -- Day 3, Late Morning
A fixer in the docks called Seren "Fox-ears" and said she knows too much about the Dark Foxes routes.

Kael's account -- Day 3, Noon
Kael believes Seren once passed information to someone aligned with the Dark Foxes.

Your observation -- Day 3, Afternoon
You saw Seren pass a sealed note to a cloaked figure. The seal matched the one found on Dark Foxes orders.
```

---

## 13. Product Rule

> Actors are the Compendium's perception-bound records of world participants and forces. Internally they link to canonical Actor records, Canon Events, Perception Records, Relationships, Locations, Possessions, and Timeline Records. Externally they show what this perspective has collected, understood, questioned, misheard, or inferred about that Actor.

The Actor page should help the user understand who or what matters in the world without exposing omniscient truth or turning the experience into a database tool.
