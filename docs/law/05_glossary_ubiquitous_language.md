# DreamChat — Ubiquitous Language & Bounded Contexts

**Status:** Draft for review — once accepted, this doc governs terminology in all active docs and code.
**Date:** 2026-06-10
**Rule:** One concept, one name. Code, PRDs, UI copy, and architecture docs use these terms. Deviations are bugs.

---

## 1. Bounded Contexts

| Context | Owns | Language register |
|---|---|---|
| **World Engine** (canon_engine set) | Canon events, projections, validation gate, entity registry, world clock, claim ledger | Engine terms below; `entity` as supertype is legal *only here* |
| **Compendium / Play UX** | Actor/Location/Artifact pages, Timeline, Aux sidebar, Carrying overlay, Corrections UX | Domain terms only — never `entity`, never `projection`, never DB language |
| **Content Governance** | Private/public/monetizable classification, media eligibility | ADR-P016 terms |
| **Image Platform** (separate service) | Image jobs, asset packs, sprite sheets | Never owns world truth; receives authorized requests only |

Translation between contexts happens at explicit mapping points (see §4), never implicitly.

## 2. Core Domain Terms (ubiquitous language)

| Term | Definition | Is NOT |
|---|---|---|
| **World** | One persistent fictional world instance with its own canon, time, and cast. | A chat session. |
| **Actor** | Any world participant or force with continuity: person, NPC, creature, AI, group, institution, government. Test: *can it act, know, remember, relate, change?* | An object, a place, an event, a relationship. |
| **Artifact** | Any meaningful object or asset with continuity: item, document, vehicle, property, evidence, key, record. Test: *can it matter across time, be known differently, be owned/moved/hidden/used?* | Limited to what the user owns or carries. |
| **Location** | A place/region/structure/route that can hold events, actors, artifacts, memories. Supports hierarchy (Part of / Known areas inside). | An Actor (it doesn't act), an Artifact, an event. |
| **Canon Event** | An immutable, validated fact of world truth: what actually happened/changed. Only the validation gate writes canon. (Engine ADR-001, -009.) | Narration prose; a belief; retrievable memory. |
| **Perception** | What one holder (user-perspective, NPC, group, public) currently understands about a canon element. Perspective-bound, versioned, sourced. First-class domain concept — *every user-facing page is built from Perceptions, never raw canon.* | Internal plumbing; a UI cache; the truth. |
| **Perception Version** | A historical layer of a Perception, created when understanding *materially* changes. | A new version per dialogue line. |
| **Perception Record** | One sourced piece of collected knowledge (source_type: direct, told_by_actor, rumor, public_record, document, inference, observation, …). | A canon event. |
| **Timeline Record** | A chronological display entry pointing to a Perception Version — never directly to canon. | A log of canon events. |
| **Carry State** | The *state* of an Artifact relative to an Actor: carried, worn, held, packed, stored_elsewhere, lost, unknown. Powers the **Carrying overlay** (fast play-facing view: "what does my character have on them now?"). | A Compendium category; a sibling concept of Artifact; an inventory system. |
| **Relationship** | A meaningful connection between two Actors (canon + perception sides). Modeled internally; **no relationship UI in MVP** — relationship information reaches the user only as ordinary sourced knowledge records. 🅿️ Parked as a surfaced feature. | A panel, a stat meter, a graph dashboard, a narrator clue. |
| **Decay** | A mechanic, not a rule: a known state that has gone uncorroborated long enough that its current validity is uncertain. Decay lowers confidence and produces "last known…" language; it never hides previously known information. | A visibility filter; forgetting. |
| **Common Knowledge** | World facts the current perspective is presumed to know without an explicit acquisition event (cultural, public, ambient knowledge of their world). A valid knowledge path alongside observation, told, record, broadcast, inference. | Omniscience; hidden canon. |
| **Compendium** | The user's perception-bound knowledge workspace: Timeline, Actors, Locations, Artifacts, Report/Corrections. | An omniscient world database browser; the user's inventory. |
| **Scene** | The current diegetic moment: place, participants, atmosphere, viewpoint. | A chat thread. |
| **Thread** | An unresolved tension, commitment, or open question the world may return to. | A quest log entry (genre-specific). |
| **Backstage Update** | World evolution computed for entities/groups/places that were away from the current scene while in-world time passed. | Real-time simulation of everything. |
| **Correction** | A user-initiated, present-forward fix to canon/perception via the correction window. Deep retroactive rewrite is parked (engine ADR-016). | Editing chat history. |
| **In-World Time** | Logical fictional time: tick + label, owned by the World Clock (engine ADR-021, -030). | Wall-clock time; a TIMESTAMPTZ. |
| **Module** | An optional, per-world pluggable system (Stats, Battle, …) that *proposes* changes; the Core validates and commits. | Something that writes canon directly. |

## 3. Engine-Internal Terms (legal only inside World Engine context)

| Term | Meaning | Domain mapping |
|---|---|---|
| `entity` | Supertype: any canonical world object with identity (actors, locations, artifacts are entity types). | Actor / Location / Artifact are entity *types*. UI and PRDs never say "entity". |
| `projection` | Materialized read model derived from canon events. | What a Compendium page is built *from* (via Perception). |
| `canonization pipeline / validation gate` | The only writer of canon. *(2026-07-23 precision: this names the whole PIPELINE. The gate STAGE inside it reads canon, is structural-only, and writes nothing; the canonization point is COMMIT. See `chunk-5.5-final/` FINAL loop PRD.)* | Invisible to users except via the correction window. |
| `narrative claim ledger` | Tracked lifecycle of durable assertions in prose. | Invisible to users. |

## 4. Mapping points (where contexts translate)

1. **Compendium page ⇄ engine:** every page = projection of (entity-type, holder's Perception). PRD field → DDL mapping appendix required per Compendium PRD (open task, MASTER_INDEX conflict #2).
2. **Carrying overlay ⇄ engine:** overlay = filter over Carry State for the user-controlled Actor. Carry State derives from canon events + perception records, with "last confirmed at" uncertainty language ("decay is not visibility").
3. **UI copy rule:** users see *Actors, Locations, Artifacts, Timeline, Known World, Carrying* — never entity/canon/projection/perception-record. Uncertainty is expressed in story language ("Last known…", "You have not confirmed this recently.").
4. **Player interiority rule:** the system narrates the world; it **never authors the player character's inner state** — no system-written statements of what the user feels, wants, or trusts, and no relationship/trust meters. (User-authored notes as `source_type: user_note` perception records are the future home for the player's own stance — 🅿️ parked, post-MVP.)
5. **Relationship rule (rev 2):** no relationship UI exists in the MVP. Relationships are modeled internally; relationship-flavored information surfaces only as ordinary sourced knowledge records that entered the perspective through a valid path. The system never synthesizes, labels, or scores a relationship for the user.
6. **Time rule:** all domain time is logical tick + display label; canon and perception are append-only. See `10_prds/compendium/00_time_and_mutability_rules.md` (normative).

## 5. Term hygiene (principle, not list)

> **System terms must be genre-agnostic and register-correct.** Genre-specific concepts (quests, mana, factions, relics, spells, …) may exist as *world or module content*, never as core system vocabulary. Domain terms stay in their bounded context (no "entity" in PRDs/UI; no DB vocabulary in UI copy).

Illustrative corrections (not exhaustive): Possession→Artifact · Faction→Group/Actor · Quest→Thread · Inventory→Carrying overlay · "memory log"→Collected knowledge. The test for any new system term: *would it still make sense in a sci-fi thriller, a workplace drama, and a horror story?*
