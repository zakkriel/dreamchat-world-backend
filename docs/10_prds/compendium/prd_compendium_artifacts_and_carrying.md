# PRD: Compendium — Artifacts & Carrying Overlay

> **Status:** Draft (formalized 2026-06-10) | **Owner:** TBD | **In MVP scope**
> **Depends on:** Canon Engine, Glossary (Artifact, Carry State, Decay, Common Knowledge), Rules Register, `00_time_and_mutability_rules.md`, `01_epistemic_type_canonical_enum.md`.

## 1. Problem Statement
**The problem:** meaningful objects (the sealed note, the stolen car, the warrant) carry plot weight but no continuity surface — users can't recall what they know about an object, who holds it, or what they have on them right now. Generic inventory UIs solve the wrong problem (slots/stats) and break perception boundaries (showing true state instead of known state).
**Who is affected:** the player mid-scene ("what do I have on me that helps *now*?" — Carrying overlay) and the player investigating ("what do I know about this object across time?" — Artifact page).

## 2. Goals
**Impact hypothesis:** if objects have perception-bound dossiers and a trustworthy carried-now overlay, object-driven stories (evidence, gifts, stolen goods, documents) become playable across long gaps — a continuity proof no competitor's inventory system delivers. Signals: Artifact page opens; Carrying overlay opens per scene; corrections on carry state (healthy accuracy signal).

## 3. Scope / Non-Goals
**In scope (MVP):** Artifact Compendium (known objects incl. not-owned/heard-about); Artifact page as inspectable dossier (visual, perceived name/type, synthesis, part-of/associated, last known location, holder/owner/access *as known*, collected knowledge by topic, links); Carrying overlay (user-controlled Actor's current carried/worn/held items, quick inspect, link to full page).
**Non-goals:** inventory slots/grids/encumbrance; stats cards; crafting; trade UI; canonical-owner display when the perspective doesn't know it; Carrying for NPCs.

## 4. Acceptance Criteria
1. **Compendium ≠ inventory:** the Artifact Compendium shows meaningful *known* objects regardless of ownership; the Carrying overlay shows only the user-controlled Actor's current Carry States. The two are never merged.
2. **Perception-bound (B-1):** Artifact pages render known/believed owner, holder, location, access — hidden truth absent from the payload. Conflicting accounts of an object's whereabouts are preserved with sources (B-6 — perception side).
3. **Carry State is derived (B-5):** carried/worn/held/packed/stored_elsewhere/lost/unknown derives from events + perception records with `last_confirmed_tick`; stale states render decay language ("you haven't confirmed this recently"), never disappear (Decay mechanic).
4. **Dossier, not database row:** page order per spec body (visual → name → type → synthesis → associations → last known → holder/access → knowledge → links).
5. **Links are provenance/reference only (B-10):** `linked_*` fields never imply causality; user-facing links point to perception-layer records.
6. **Time model (B-5), epistemic enum (G-4), vocabulary (GA-2)** compliance — UI never says "inventory" or "possessions".
7. **Mock parity:** `mock_aux_lens_inspect_artifact.png` (Red Gem Necklace) is the Inspect-lens reference for quick inspect; full page follows the dossier order.

## 5. Open Questions
- Container semantics (pouch inside bag) in MVP or post-MVP?
- Artifact imagery: generated per artifact at creation, on first inspection, or lazily? (Bridge doc.)

---

# Spec Body (the validated model — register-compliant, unchanged otherwise)

# DreamChat Artifacts Compendium and Carrying Overlay Model

## Core Definition

An Artifact is any meaningful object, asset, document, item, device, vehicle, property, evidence, tool, weapon, container, key, message, record, or artifact that has continuity in the world.

Artifacts are not limited to what the user owns or carries.

Artifacts may include a Ming dynasty vase in an embassy gallery, Seren's silver hairpin, a sealed note, a stolen car, a flat, a contract, a warrant, a phone with deleted messages, a spaceship access card, a cursed sword, a family photograph, or a company share certificate.

The practical test:

Can this object or asset matter across time, be known differently by different perspectives, be carried, owned, hidden, moved, used, lost, stolen, inspected, or linked to events?

If yes, it is probably an Artifact.

## Core Product Rule

Artifacts are canonical world objects internally, but user-facing Artifact pages are perception-bound.

This means:

- Internally, an Artifact has a canonical record.
- Externally, the user sees what their current perspective knows, believes, remembers, observed, inferred, heard, misunderstood, or has access to about that Artifact.
- The Artifact page must not expose hidden truth during normal play.
- Different users, NPCs, groups, or public knowledge layers may hold different perceptions of the same Artifact.
- Conflicting information should be preserved through source and context, not collapsed too quickly into one final answer.

Product rule:

The Artifact Compendium shows meaningful known objects. It is not the user's inventory.

## Artifacts vs Carrying

Artifacts and Carrying are related but not the same feature.

Artifacts is the deep Compendium category.
It answers:

What meaningful objects, assets, documents, tools, or properties do I know about in this world?

Carrying is a fast play-facing overlay.
It answers:

What does my user-controlled character currently have on them and can likely use right now?

The split is important because the Compendium may contain objects the user does not own, does not carry, cannot access, or has only heard about. The Carrying overlay should only show what is currently carried by the user-controlled character.

## Recommended Schema

Artifact Canon
  artifact_id
  world_id
  canonical_type
  canonical_identity
  canonical_description
  physical_or_digital_form
  canonical_owner_actor_id optional
  canonical_holder_actor_id optional
  canonical_location_id optional
  canonical_access_state
  hidden_truth_fields
  linked_canon_records
  created_at_tick
  as_of_tick (derived from latest committed event/version — never stored as a mutable field)

Artifact Perception
  artifact_perception_id
  artifact_id
  holder_id
  holder_type: user | npc | group | public | system-perspective
  perceived_name
  perceived_role_or_type
  current_synthesis
  perceived_owner
  perceived_holder
  perceived_location
  perceived_access
  first_perceived_tick
  latest_version_tick (derived)
  visibility_scope
  linked_perception_records

Perception Record
  perception_record_id
  artifact_id
  artifact_perception_id
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

Carry State
  carry_state_id
  artifact_id
  actor_id
  state: carried | worn | held | packed | stored_elsewhere | lost | unknown
  container_artifact_id optional
  last_confirmed_tick
  source_perception_record_id optional

Artifact Page Projection
  perceived_name
  perceived_role_or_type
  current_synthesis
  part_of_or_associated_with optional
  last_known_location
  current_holder_owner_access
  collected_knowledge_groups
  inline_links

Carrying Overlay Projection
  carried_artifacts
  quick_inspect_preview
  contextual_actions
  open_full_artifact_link

## Artifact Page UX Structure

The Artifact page should not feel like an inventory slot, stats card, or database row.

It should feel like an inspectable object dossier: visual, readable, source-aware, and perception-bound.

Recommended order:

1. Artifact visual / object image / atmospheric representation
2. Perceived name
3. Perceived type or role, if useful
4. Short synthesis paragraph
5. Part of / associated with, if relevant
6. Last known location
7. Current holder / owner / access, if known
8. Collected Knowledge grouped by topic
9. Inline links to Actors, Locations, Timeline records, and other Artifacts

The page should not duplicate information through multiple separate link panels. Use inline links and a few useful context rows.

## Collected Knowledge

Use Collected Knowledge as the main UX label for what is known about an Artifact.

Do not add a separate What is Known summary panel. That becomes redundant and has no room to explain the information. Collected Knowledge is the source-of-truth UX for what is known, remembered, inferred, heard, or questioned about an Artifact.

Collected Knowledge should be grouped by topic, not by raw timeline/log order.

Example:

Collected Knowledge

The vase itself
  Museum catalog -- Day 12
  Cataloged as Ming dynasty, mid-15th century porcelain.

  Your observation -- Day 23
  A faint hairline crack near the base. Repaired once.

Hidden compartment
  Handler briefing -- Day 31
  Handler mentioned a false rim and internal cavity.

  Guard conversation -- Day 32
  Guard joked it could hide more than letters.

Movement / security
  Gala observation -- Day 46
  Moved from study to East Gallery before the gala.

  Staff rumor -- Day 47
  Staff say it is never left the estate without escort.

## Current Holder, Owner, Access

Holder, owner, and access are different things.

Owner answers:
Who is believed to own it?

Holder answers:
Who is believed to physically or practically have it now?

Access answers:
Can the user-controlled character currently reach, use, read, unlock, open, carry, enter, or control it?

Example:

Ming Dynasty Vase

Owner:
Ambassador Renwick officially owns it.

Holder / Location:
Last known in the East Gallery, Embassy Residence.

Access:
You do not currently have access.

These fields should preserve uncertainty. If access, ownership, or location is old or secondhand, say so through source and timing rather than presenting it as final truth.

## Inline Links

MVP should use inline links, not a heavy related-items panel.

Example:

The vase is officially part of Ambassador Renwick's Collection and was last seen in the East Gallery of the Embassy Residence.

Here:

- Ambassador Renwick links to an Actor page.
- Ambassador Renwick's Collection may link to an Artifact or Actor/Structure record depending on how the world models it.
- East Gallery links to a Location page.
- Embassy Residence links to a Location page.

Do not add a separate Also check or Linked to section in MVP.

## Carrying Overlay Definition

The Carrying overlay is a compact right-sidebar gadget inside the play-first experience.

It is not a full inventory page.
It is not the Artifact Compendium.
It is not a nearby-object detector.

It shows only what the user-controlled character is currently carrying, wearing, holding, or has immediately on them.

Its purpose is fast usability during play:

What do I have on me right now?

The overlay should support quick inspect and basic contextual actions without taking the user out of the current scene.

## Carrying Overlay UX

Placement:
The Carrying overlay lives in the lower part of the right AUX sidebar, below the active lens content. It can be collapsed into a small row or expanded into a compact list.

Collapsed state:

Carrying now   Open
Coin pouch, sealed note, red gem necklace, small knife

Expanded state:

Carrying now   Close

Coin pouch        Inspect   Use
Sealed note       Inspect   Use
Red gem necklace  Inspect   Use
Small knife       Inspect   Use

Open full Artifacts

The overlay should stay visually light. It should not introduce a second full-page inventory system.

It should respect the same right-sidebar style already used by Current, Inspect, Intent, and Known: dark translucent surface, minimal icons, readable type, soft separators, low box density, and no dashboard grid.

## Carrying Overlay Interaction

Primary interaction:
Clicking an item opens a quick AUX Inspect preview in the same right sidebar.

Example:

Sealed note
A small folded note, sealed with dark wax. No markings, no sender.

Actions:
- Read
- Show to Seren
- Put away
- Hide
- Open full Artifact record

The action list must be contextual. It should not always show the same buttons for every item.

Examples:

Sealed note:
Read, show, hide, put away.

Small knife:
Draw, hide, offer, inspect.

Coin pouch:
Count, pay, offer, hide.

Red gem necklace:
Inspect, show, wear, hide, ask about.

The quick actions are proposed affordances, not hard command limits. The user can always type any natural language action.

## No Nearby Objects in MVP

The Carrying overlay should not show nearby objects in MVP.

Nearby objects require a separate scene-object affordance mechanic. The system would need to continuously decide which objects are present, visible, reachable, usable, owned by someone else, safe to touch, or important enough to surface.

That is a valuable later feature, but it should not be mixed into the Carrying overlay now.

Nearby objects can still appear in the Current or Inspect lenses when relevant:

A loose knife lies beside the stall.

A guard places a warrant on the table.

A cracked radio blinks from the counter.

But they should not appear in Carrying unless the user-controlled character actually takes, receives, wears, holds, or packs them.

## MVP Artifact Page Includes

- Artifact visual
- Perceived name
- Perceived type or role, if useful
- Short synthesis
- Part of / associated with, if relevant
- Last known location
- Current holder / owner / access, if known
- Collected Knowledge grouped by topic
- Source/timing for perception records
- Inline links to Actors, Locations, Artifacts, and Timeline records
- Hidden canon exclusion during normal play

## MVP Artifact Page Excludes

- Omniscient artifact truth
- Hidden properties unless known/perceived
- Generic item stats
- RPG inventory grid
- Durability/weight/equipment slots unless the world style requires them later
- Separate What is Known panel
- Heavy related-link panels
- Nearby object list
- Full object simulation controls

## MVP Carrying Overlay Includes

- Compact lower-right AUX placement
- Collapsed and expanded states
- Only carried / worn / held / packed items
- Quick inspect preview
- Contextual action suggestions
- Link to full Artifact record
- Link to full Artifacts Compendium

## MVP Carrying Overlay Excludes

- Full inventory page
- Nearby objects
- Owned elsewhere
- Stored elsewhere
- Weight management
- Equipment grid
- Loot system
- Crafting system
- Auto-surfaced environment objects

## Final Product Rule

Artifacts are the Compendium's perception-bound records of meaningful world objects, assets, documents, items, and properties.

Carrying is the play-facing quick overlay for objects currently on the user-controlled character.

Artifacts preserve world continuity and knowledge boundaries.
Carrying supports immediate play.

Do not collapse them into one feature.
