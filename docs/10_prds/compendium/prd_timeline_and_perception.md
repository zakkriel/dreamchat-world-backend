# PRD: Timeline & Perception Layer

> **Status:** Draft (formalized 2026-06-10) | **Owner:** TBD | **In MVP scope**
> **Depends on:** Canon Engine (perception_record, world clock, projections), Glossary, Rules Register, `00_time_and_mutability_rules.md`, `01_epistemic_type_canonical_enum.md`.
> **Note:** this PRD is the *product-side* contract of what the engine already freezes architecturally (ADR-005/006/021/030); on any conflict, the engine set wins.

## 1. Problem Statement
**The problem:** without a chronological record of *what the user's perspective experienced and learned*, long worlds become unnavigable and unverifiable — users can't check what happened, when, or how their understanding evolved, which erodes trust in continuity (the product's core promise).

## 2. Goals
**Impact hypothesis:** a perception-bound timeline that shows understanding *evolving* (not just events listing) is the most visible proof that the world remembers — driving return sessions and correction engagement. Signals: timeline opens per session; jumps from timeline records into Actor/Location/Artifact pages; perception-version views.

## 3. Scope / Non-Goals
**In scope (MVP):** one timeline registry per world/session bound to the user perspective; chronological records grouped by in-world time labels (Day 3, Morning); record detail (what happened / why it matters / linked actors-locations-artifacts); evolution visibility when a perception was versioned.
**Non-goals:** canon event browser; NPC timelines as UI; multi-perspective comparison views; editing the past from the timeline (corrections flow through the correction UX); auto-regenerated summaries on read.

## 4. Acceptance Criteria
1. **Timeline points to Perception Versions, never canon (B-10 + core rule):** every timeline record references a perception version — even when perception and canon are identical.
2. **Versioning on material change only:** new versions for reframing/contradiction/confirmation/visibility change — never per dialogue line or wording tweak.
3. **Deterministic from store (B-9):** records and syntheses derive from stored versions; reload never regenerates differing text. AI may help at write-time; the registry is store behavior.
4. **Evolution is visible:** when a record's underlying perception was versioned, the user can see that understanding changed over time (v1→v2→v3), with sources per the canonical epistemic enum.
5. **In-world time only (B-5):** ordering by tick, display by label; wall-clock never rendered; "no records yet" periods render as unknown, not empty fabrication.
6. **Perception-bound (B-1):** records the perspective doesn't hold are absent from the payload.
7. **Mock parity, with exceptions:** `mock_compendium_timeline.png` is the reference **except** nav corrections (no Relationships item; Possessions→Artifacts; "Entities"→Actors).

## 5. Open Questions
- How is perception-version history surfaced — inline expansion vs. a "how my understanding changed" view?
- Timeline record importance filtering (everything vs. curated by importance score, doc 06 scoring)?

---

# Spec Body (the validated model — register-compliant, unchanged otherwise)

# DreamChat Perception Layer and Timeline Model

## 1. Core Idea

Timeline should not display raw canon directly. Timeline should display a perspective-bound perception of canon.

Canon is the source of truth: what actually happened, exists, changed, or was established in the world. Perception is how a specific observer currently understands that canon element. Timeline is the user-facing chronological projection of those perception records.

**Product rule:** Canon is objective world truth. Perception is perspective-bound understanding. Timeline is the chronological UX projection of perception versions.

## 2. Conceptual Model

| Concept | Definition |
|---|---|
| Canon Element | The objective world truth. This can be an event, fact, state change, relationship change, artifact/ownership change, public situation, location change, or other accepted world-canon element. |
| Perception | A perspective-specific interpretation of a canon element, held by a user, NPC, group, institution, public audience, or other observer. |
| Perception Version | A historical version of that perception. New versions are created when understanding materially changes. |
| Timeline Record | A display entry that points to a specific perception version. The timeline never links directly to canon, even when perception and canon are identical. |

## 3. Relationship Structure

```text
Canon Element
  -> Perception Record(s)
      -> Perception Version(s)
          -> Timeline Record(s)
```

A timeline registry is created when the world/session starts and is linked to the user-controlled perspective. Each timeline record belongs to that perspective and points to a perception version. This allows different users, NPCs, or groups to hold different histories of the same underlying canon element.

## 4. Why Timeline Links to Perception, Not Canon

- The same canon event can be understood differently by different observers.
- A user may have partial knowledge, mistaken knowledge, indirect knowledge, or updated knowledge.
- NPCs can maintain their own timeline-like understanding without becoming omniscient.
- Future multiplayer worlds can support different player histories and secrets.
- Public knowledge can diverge from private truth through gossip, propaganda, records, lies, misinterpretation, or missing information.

## 5. Perception Versioning

Perception should be versioned when an observer’s understanding of a canon element materially changes. This does not rewrite the canon event. It creates a new historical layer of understanding.

This allows the product to show not only what someone currently believes, but how their understanding evolved over time.

### When to create a new perception version

- A new source contradicts or reframes the previous understanding.
- Uncertainty meaningfully increases or decreases.
- Private information becomes public or shared.
- A rumor becomes confirmed, disproven, distorted, or weaponized.
- Direct evidence replaces indirect information.
- An observer changes interpretation because of new context.
- A lie, omission, propaganda, or unreliable witness changes the perceived story.

### When not to create a new perception version

- Tiny wording changes that do not affect meaning.
- Repeated retrieval of the same known fact.
- Minor UI display changes.
- Every line of dialogue or every beat. Perception should be updated only when understanding changes materially.

## 6. Example: One Canon Event, Multiple Perceptions

| Layer | Holder / Scope | Content |
|---|---|---|
| Canon Element | World truth | Kael killed the courier in the alley. |
| Perception v1 | User | I saw Kael standing over the body with a bloody knife. He may be involved. |
| Perception v2 | User | Seren says Kael was defending himself, but she may be protecting him. |
| Perception v3 | User | The courier’s note suggests Kael arrived after the first attack. His role is unclear. |
| Perception v1 | Public | A thief killed the courier. |
| Perception v2 | Public | Rumor says Kael was seen fleeing the alley. |
| Perception v1 | Kael | The courier was already dying when I arrived. |

## 7. Timeline Registry Mechanics

- Create one timeline registry at world/session creation and link it to the user-controlled perspective.
- Timeline records point to perception versions, never directly to canon.
- Each record carries in-world time, display title, display summary, visibility, and links to relevant entities, locations, relationships, and artifacts.
- If the perception changes materially, a new perception version is created and the timeline can show that understanding evolved.
- The timeline itself is deterministic registry/store behavior. AI may help summarize or propose perception content at write-time, but the timeline should not be regenerated from scratch by an LLM during display.

## 8. Suggested Data Shape

```text
timeline_registry
  id
  world_id
  holder_type: user | npc | group | public
  holder_id
  perspective_label

canon_element
  id
  world_id
  type: event | state | relationship | artifact | location | knowledge | other
  objective_summary
  occurred_at_tick + display_label
  canon_status

perception
  id
  canon_element_id
  holder_type
  holder_id
  current_version_id

perception_version
  id
  perception_id
  version_number
  perceived_summary
  interpretation
  source_type: direct | told | inferred | public | record | rumor | propaganda | unknown
  reliability
  confidence
  uncertainty_notes
  created_at_tick
  supersedes_version_id

timeline_record
  id
  timeline_registry_id
  perception_version_id
  display_time_marker
  display_title
  display_summary
  visibility_scope
  linked_entities
  linked_locations
  linked_artifacts
```

## 9. UX Implications

- The user timeline is not “what objectively happened.” It is what that perspective currently understands as its history.
- Timeline can show perception evolution: “At first you believed X; later you learned Y.”
- Known World can expose uncertainty without spoiling hidden canon.
- NPC-specific timelines become possible later because NPCs can hold their own perception versions.
- Multiplayer becomes cleaner because each user can have a different timeline over the same canon world.

## 10. Guardrails

- Do not create perception versions for every tiny update.
- Do not let timeline reveal canon that the holder cannot know.
- Do not overwrite old perceptions; supersede them with new versions.
- Do not treat indirect knowledge as direct memory.
- Do not collapse public knowledge, private knowledge, and objective truth into one record.

## 11. Final Product Rule

**Canon is what happened. Perception is what a perspective believes or understands about what happened. Timeline is the readable history of perception over in-world time.**
