# PRD: Compendium — Actors

> **Status:** Draft (formalized 2026-06-10 from `dreamchat_actors_compendium_model_v2`) | **Owner:** TBD
> **Scope decision:** **In MVP scope** — product decision 2026-06-10: DreamChat ships an MVP (not engine-only PoC); the Compendium UI demonstrates system value.
> **Depends on:** Canon Engine (perceptions, projections, entity registry), Glossary (`00_strategy/05`), Timeline & Perception PRD, **`00_time_and_mutability_rules.md` (normative — schemas below were patched to tick-based append-only fields on 2026-06-10)**.

---

## 1. Problem Statement

**The problem:** In long-running persistent worlds, the user cannot hold the cast in their head. Without a trustworthy, perception-bound reference for "who matters and what do I know about them," users either lose the thread (and stop believing the world remembers) or are given an omniscient database view (which destroys secrets, rumor, and discovery — the core differentiators vs. companion apps and AI-GM tools, per `00_strategy/03_market_research`).

**Who is affected:**

| User | Job to be done |
|---|---|
| Player returning after a gap | Re-orient: who is this, what do I know, why do they matter now |
| Player mid-investigation | Compare sources: what was rumor vs. observed vs. told — without the system flattening it into one "truth" |
| Player spotting an error | Jump to corrections from the place the error is visible |

**Why now:** The Actor page is the single strongest user-visible proof of the product promise ("NPCs are not disposable; the world remembers"). It is the MVP's value showcase.

## 2. Goals & Success Signals

**Impact hypothesis:** If the Actor page shows perception-bound, source-preserving knowledge that visibly evolves, users will trust world memory, leading to longer-running worlds and return sessions — because trust in continuity is the retention driver this category lacks.

**Success signals (baselines TBD at MVP launch):** % of sessions opening ≥1 Actor page; return-session rate for users who used the Compendium vs. not; correction reports initiated from Actor pages (a *healthy* signal — it means users care about accuracy); zero hidden-truth leak incidents.

**Guardrails:** Page open must not stall play (perceived load < ~1s from cached projection); no regression of narration latency.

## 3. Scope / Non-Goals

**In scope (MVP):** Actor list + Actor page per the spec body below; perception-bound projection; Collected Knowledge grouped by topic with sources and in-world timestamps;  known artifacts/associated objects; inline links; report/correction entry point.

**Non-goals (MVP):** Relationships as a top-level section (🅿️ parked — see `parked_relationships.md`); relationship graphs, trust meters, network panels; omniscient/creator view in normal play; editing canon from the Actor page; numeric stats (module territory).

## 4. Acceptance Criteria

1. **Perception-bound, always.** The page renders exclusively from the user-perspective's Actor Perception + Perception Records. A planted secret (engine invariant I-3) is never present in the page payload — not hidden by CSS, **absent from the response**.
2. **Source preservation.** Every Collected Knowledge item displays source type and in-world time. Direct experience, rumor, third-party account, and observation remain distinguishable; contradictory records co-exist without auto-resolution.
3. **Synthesis honesty.** The synthesis paragraph reflects only held perception records. If understanding materially changed, the new synthesis derives from the latest Perception Version (no regeneration drift on reload — deterministic from stored state).
4. **Uncertainty language, not omission.** Stale knowledge renders with decay language ("Last known…", "remembered, not verified") — decay never hides previously known info ("decay is not visibility").
5. **Identity continuity.** After a long in-world gap + backstage update, the page shows updated state sourced to valid information paths (told, observed, public record…) — never silently mutated facts.
6. **Vocabulary compliance.** UI copy uses Glossary terms only; never "entity", "artifact", "inventory", "memory log".
7. **No relationship UI in MVP — at all.** (Product decision 2026-06-10, superseding the earlier "gated panel" version of this AC.) The Relationship Perception remains modeled internally, but the Actor page has **no relationship panel, field, or summary of any kind**. Relationship-flavored information reaches the user only as ordinary Collected Knowledge records when it entered their perspective through a valid path (e.g., an NPC spoke about the relationship → that statement is a sourced knowledge record like any other). The system never synthesizes or labels a relationship.
8. **The system never authors the player character's interiority.** No system-rendered statement of what *you* feel about an Actor, and **no trust/relationship meter or slider** (reaffirms `parked_relationships` non-goals — the slider in the v2 mock is rejected).
9. **Time model compliance.** All page data derives from tick-based, append-only canon/perception per `00_time_and_mutability_rules.md`; in-world time renders as display labels only.
10. **Mockup parity, with exceptions:** `20_design_ux/mockups/mock_compendium_actor_seren_v2.png` is the reference layout **except**: (a) the trust slider — removed per AC #8; (b) the "Relationship to you" panel — **removed entirely** per AC #7; (c) nav + rail label "Artifacts" — renders as **Artifacts** per Glossary; (d) the "Add note" button — **not in MVP** (parked, see §5). v1 mock is superseded (old top-level Relationships nav).

## 5. Open Questions

- Metrics instrumentation plan + baselines (none exist pre-MVP).
- 🅿️ **Parked — user notes:** "Add note" (user-authored annotations as Perception Records with `source_type: user_note`) is a strong post-MVP concept: it would give the player's own interpretation a first-class, sourced home without the system authoring it. Out of MVP scope by product decision 2026-06-10.
- Portrait generation policy: when (if ever) does an Actor portrait regenerate as perception materially changes? (Image Platform dependency — for the Bridge doc.)
- Multi-Actor disambiguation UX when entity resolution (engine doc 05) is uncertain — does ambiguity surface to the user, and how?
- How much of the page projection is precomputed vs. assembled on open (engine doc 06 budget rules)? → `technical_questions` for engineering.

---

# Spec Body (the validated model — unchanged below)

# DreamChat Actors Compendium Model — Updated

## Purpose

This document defines the **Actors** section inside the DreamChat **Compendium**.

The goal is to give the user a readable, perception-bound view of the people, beings, groups, institutions, and organized forces that matter in the world.

Actors are not shown as omniscient database records. They are shown as the current perspective understands them.

---

## 1. Naming

**User-facing name:** Actors  
**Internal model name:** `actor`  
**Parent container:** Compendium

Recommended MVP Compendium structure:

```text
Compendium
  ├── Timeline
  ├── Actors
  ├── Locations
  ├── Artifacts
  └── Report / Corrections
```

Important: **Relationships are not a top-level Compendium section in MVP.** They are modeled internally and surfaced inside Actor pages.

---

## 2. Actor Definition

An **Actor** is any world participant or force that can act, know, remember, influence, relate, or change over time.

Actors may include people, NPCs, companions, rivals, user-controlled characters, creatures or animals with agency, spirits, gods, intelligences, AIs, monsters, groups, families, crews, factions, institutions, companies, governments, courts, cults, criminal networks, schools, labs, or other organized forces.

The practical test:

> Can this thing behave like a world participant or force with continuity?

If yes, it is probably an **Actor**.

---

## 3. What Is Not an Actor

```text
A sword          → Artifact / Object
A letter         → Artifact / Document
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

The canonical Actor record represents the objective actor in the world model. It may include hidden truth, stable identity, internal continuity, and objective world-state links.

### Actor Perception

The Actor Perception represents what a given perspective currently knows or understands about that Actor.

A perspective may belong to the user-controlled character, another player/user, an NPC, a group or institution, the public, a rumor network, or another information source.

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
  created_at_tick
  as_of_tick (derived from latest committed event/version — never stored as a mutable field)

Actor Perception
  actor_perception_id
  actor_id
  holder_id
  holder_type: user | npc | group | public | system-perspective
  perceived_name
  perceived_role
  current_synthesis
  first_perceived_tick
  latest_version_tick (derived)
  visibility_scope
  linked_perception_records

Perception Record
  perception_record_id
  actor_id
  actor_perception_id
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
  linked_relationship_records optional

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

Actor Page Projection
  perceived_name
  perceived_role
  current_synthesis
  relationship_to_you
  known_relationships
  last_known_status
  known_artifacts_or_associated_objects
  collected_knowledge_groups
  inline_links
```

---

## 7. Important Schema Rule

The Actor page should not flatten all perception records into one truth.

It should preserve source, timing, contradiction, uncertainty, partial knowledge, indirect knowledge, rumor existence, and evolving understanding.

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
6. Known relationships
7. Last known status
8. Known artifacts / associated objects
9. Collected Knowledge
```

The visual direction should remain close to the stronger Actor mockup: atmospheric, spacious, scrollable, low-box, low-icon, and close to the existing DreamChat play UI.

---

## 9. Actor Page Sections

### 9.1 Perceived Name

The name shown should be what this perspective knows.

Examples:

```text
Seren
The Fox-Eyed Woman
The Harbor Captain
Unknown Red Court Courier
```

### 9.2 Perceived Role

A short role description based on current perception.

Examples:

```text
Market informant, as you currently understand her.
Former courier, according to Kael.
Unknown faction agent.
Local police liaison.
```

### 9.3 Short Synthesis

A concise natural-language summary of current understanding.

This is not final truth. It is the system's best current synthesis from this perspective's collected knowledge.

Example:

```text
Seren is a quick-witted informant who moves easily through Dawnfall Market's many circles. She trades in rumors, half-truths, and hard-to-find details. Her loyalties are unclear, but she has shown you glimpses of useful, carefully measured truth.
```

### 9.4 Relationship to You

This should remain in the Actor page.

It is a core part of persistent continuity and helps the user understand the Actor's stance toward the user-controlled character.

It should be written naturally, not as a meter.

Example:

```text
Cautious but engaged. Seren shares information selectively and watches your reactions closely. Trust is provisional -- earned, not given.
```

Avoid numeric trust bars or rigid relationship stats in MVP.

### 9.5 Known Relationships

Relationships are not a top-level Compendium section in MVP.

Known relationships should appear inside the relevant Actor page using source-based language and inline links.

Example:

```text
Known relationships
Seren is said by some market sellers to have worked with the Dark Foxes. Kael claims she only worked with them once. The exact nature of the connection is unclear from your current knowledge.
```

In this example:

- `Dark Foxes` links to the Dark Foxes Actor page.
- `Kael` links to Kael's Actor page.

This should remain a light prose section or be integrated into Collected Knowledge. MVP should avoid a separate relationship graph, related-actor panel, or relationship dashboard.

### 9.6 Last Known Status

This should combine location and status when possible.

Examples:

```text
Last seen at Dawnfall Market, near the eastern stalls. Day 3, Morning.

Last known to be traveling with the Harbor Police convoy. Two weeks ago, according to Kael.

Current location unknown. Last public mention placed her near the southern docks.
```

Known locations should not become a separate mini-panel in MVP unless the Actor has many meaningful location records.

### 9.7 Known Artifacts / Associated Objects

This should be a light section.

Only show objects that matter to how the user understands the Actor.

Examples:

```text
- Worn coin purse with concealed seam
- Folded scrap of charcoal paper
- Silver hairpin etched with a fox motif
```

Objects should be inline-linkable when they have their own Artifact / Object record.

---

## 10. Collected Knowledge

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

They should see what topics matter, what information exists under each topic, where each piece came from, and how the information conflicts or evolves.

The raw event order belongs more naturally in Timeline.

---

## 11. Inline Links

MVP should use inline links only.

No "related actors" or "also check" section yet.

Example:

```text
Seren is said by some market sellers to have worked with the Dark Foxes. Kael claims she only worked with them once.
```

This keeps the UI clean and avoids turning Actor pages into graph dashboards.

---

## 12. Actor Page MVP Requirements

MVP Actor pages should support:

```text
- Perceived name
- Perceived role
- Short synthesis
- Relationship to you
- Known relationships as embedded source-based knowledge
- Last known status
- Known artifacts / associated objects
- Collected Knowledge grouped by topic
- Source/timing for perception records
- Inline links to Actors, Locations, Artifacts, Timeline records, and relationship-relevant records
- Hidden canon exclusion during normal play
```

MVP Actor pages should not include:

```text
- Omniscient actor truth
- Hidden motivations unless known/perceived
- Relationship meters
- Stats
- Character sheet editing
- Top-level relationship graph
- "Also check" or related-actor panels
- Heavy dashboards
- Raw event logs
- Direct database correction buttons
```

---

## 13. Example Actor Page Content

```text
Compendium > Actors > Seren

Seren
Market informant, as you currently understand her.

Seren is a quick-witted informant who moves easily through Dawnfall Market's many circles. She trades in rumors, half-truths, and hard-to-find details. Her loyalties are unclear, but she has shown you glimpses of useful, carefully measured truth.

Relationship to you
Cautious but engaged. Seren shares information selectively and watches your reactions closely. Trust is provisional -- earned, not given.

Known relationships
Seren is said by some market sellers to have worked with the Dark Foxes. Kael claims she only worked with them once. The exact nature of the connection is unclear from your current knowledge.

Last known
Dawnfall Market, near the eastern stalls. Day 3, Morning.

Known artifacts / associated objects
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

## 14. Product Rule

> Actors are the Compendium's perception-bound records of world participants and forces. Internally they link to canonical Actor records, Canon Events, Perception Records, Relationship records, Locations, Artifacts, and Timeline Records. Externally they show what this perspective has collected, understood, questioned, misheard, or inferred about that Actor.

The Actor page should help the user understand who or what matters in the world without exposing omniscient truth or turning the experience into a database tool.
