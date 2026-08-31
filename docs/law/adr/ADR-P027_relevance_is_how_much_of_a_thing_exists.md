# ADR-P027: Relevance is how much of a thing exists, and it only ever ratchets up

**Status:** Accepted (2026-08-31)
**Date:** 2026-08-31
**Series:** Platform / Engine (`ADR-P###`, per `D-5`)
**Governing rules:** `D-8` (the synchronous path stays small; slow work runs async), `B-1`/`I-3`
(hidden truth is *absent* from a payload, not hidden by the UI), `GA-2` (system vocabulary is
genre-agnostic), `SPEC-040` (a thing may exist before anyone knows it fully), `ADR-P021` (art is
reconciled, never commissioned inline), `ADR-P023` (a style's look lives in `artstyle.go`).
**Owner of decision:** founder rulings in conversation, 2026-08-30 and 2026-08-31.
**Related:** `SPEC-045` (the loading window), `SPEC-047` (asset reuse), `ADR-P026` (the domain).

---

## Context

A world of interest is far larger than a world anyone will ever look at. The 2026-08-30 measurements
made the arithmetic unavoidable:

| shape | entities | output tokens | wall clock (20-wide) |
|---|---|---|---|
| what genesis actually built | 33 | 76,288 | 14.6 min |
| founder's target depth | ~3,600 | ~8.3M | ~110 min |

The cost driver is **2,312 output tokens per entity** — because every entity was authored to the same
fullness whether or not anybody would ever meet it. Authoring 600 location-keepers to the same depth as
the city's ruler is the whole waste, and no amount of parallelism fixes it: parallelism divides wall
clock, never volume.

The founder's proposal was a name for the missing dimension:

> we should call it relevance and not obscurity, and it should be from 1 to 3 … 1 is exist with a
> description and tag like personality for people, mantra or oath or anything similar for factions,
> slogan for business, etc. 2 is it got referenced and it gets proper interactive depth. 3 is it got
> interacted and it gets much more (images) and stuff

and then, the following day, the top of the ladder:

> relevance 4 might work out as a "VIP" concept … NPC in the party, NPC that is your wife, a location
> that is your house … you don't want the whole Pub wearing the same face as your wife just because
> their tags kinda look the same

I argued against level 4 on the grounds that asset exclusivity is not depth. **That objection was
wrong.** Level 4 is not defined by exclusivity; it is defined by *being bound to the player*, which is
the natural top of the same progression — and exclusivity follows from it.

## Decision

### 1. Relevance is a field on every entity, 1 to 4, and it means how much of that thing exists

| level | earned by | what exists |
|---|---|---|
| **1** | being named | name, one-line descriptor, kind, and **one tag** — the single characteristic thing the narrator can play with |
| **2** | being referenced | enough to hold a scene: standing, manner, what it controls or publishes, what it withholds |
| **3** | being interacted with | the full interior — beliefs, mantras, traumas, a goal with its sacrifice, example phrases, canon located here, **and an image** |
| **4** | being bound to the player | everything at 3, plus an asset that is **theirs alone** |

Per kind, in full:

| | 1 | 2 (adds) | 3 (adds) |
|---|---|---|---|
| **person** | name, descriptor, tag | standing, speech manner, hiding, ≥1 trait | upbringing, beliefs, mantras, traumas, goal + sacrifice, example phrases, image |
| **location** | name, descriptor, kind, extent, `within` | a description the narrator can work from | canon located here, objects, its people reassessed |
| **faction** | name, descriptor, kind, tag (mantra / oath / slogan) | what it controls, publishes, buries | goal + sacrifice, seat, its canon |
| **object** | name, minimal descriptor, kind | whose hand, or which location | real description, canon, image |
| **concept** | name, what it is, in a line | what is contested, who teaches it | who holds which position, and who is wrong |

**Placement is structural, not depth.** A person's `starts_in`, an object's `where` and a location's
`within` are required at *every* level, including 1: "exists" means exists somewhere in the world, and an
entity the engine cannot place cannot be stored. Only *description* scales with relevance.

**Relevance 4 adds no content class of its own.** Its whole content is exclusivity: a relevance-4 asset
is never reused for another entity, and never borrowed as another entity's anchor, in any world
(`SPEC-047`).

### 2. There are two mechanics, and confusing them is the error this ADR exists to prevent

**Genesis assigns.** The scaffold pass tags every name it creates with the relevance that name deserves,
and the fill authors *to that level*. Relevance is therefore an **instruction to the author**, not a
filter applied afterwards — which is precisely what makes the fill cheap, rather than making an
expensive fill look cheap.

**Play promotes.** Reaching an entity raises its relevance, and the promotion mints the content the new
level demands.

- referencing an entity at 1 → promote to **2**
- interacting with an entity at 1 or 2 → promote to **3**
- **promotion is a ratchet: relevance never falls.**

### 3. Promotion is authored reachability, not inferred from a click

Entering a location fires **one** call that promotes the location *and* names which of its people rise
with it — the one who serves you, the one on the door, the one demanding attention. Everyone else stays
at 1 with their description.

This is a judgement about who a scene makes speakable, and the model states it; code never infers it
from proximity. The accepted degradation is explicit: **speak to someone the promotion call did not
raise, and they answer from their one-line tag.** Thin, not broken.

### 4. New entities always enter at relevance 1

This is the law that makes the system terminate. Without it, promoting A authors B in full, which
authors C, forever.

A relevance-1 entity is a **complete** entity at its level, not a defective one. The belt validates
against the level (§5), so "thin" is a legal state and needs no placeholder, no `TODO`, and no
sentinel.

### 5. The belt validates against the level

`genesisDoc.validate()` stops demanding the same fullness from everyone. Every check becomes: *does this
entity carry what its relevance requires?* A relevance-1 person missing `standing` is correct; a
relevance-3 person missing it is refused.

`SPEC-040` already established the same principle for perception — a thing may exist before anyone
knows it fully — so this is that rule applied to authorship.

### 6. An image is a precondition of speech, and that is why 3 is expensive

Measured 2026-08-31: **6.04 s and $0.04** per image; a subject nobody has drawn needs anchor *then*
portrait (~12 s, $0.08), because generation is reference-conditioned (`image:ADR-017`).

There is no thin conversation. Either an entity is at 3 with an image, or interacting with it makes it 3
first. That single fact is what forces promotion onto a **foreground** art path inside the loading window
(`SPEC-045`) — the 2-minute reconciler sweep (`ADR-P021`) remains correct for everything nobody is
looking at, and wrong for the person standing in front of the player.

**Evidence in code (`D-9`):**
- `core/api/worldidentity.go` — the two-stage schedule, what each level owes (`personOwing`, `placeOwing`,
  `factionsOwing`), the compiled mandate, and the deepening that lets a thin thing be filled in later
  without being renamed or demoted.
- `core/api/worldgenesis.go` — the belt, validating against the level rather than demanding one fullness of
  everyone; `validRelevance` is the ladder.
- `core/api/worldkickstart.go` — the §4 law where a creation path outside fill obeys it: kin the player names
  enter at relevance 1, decided by code and not by a seat.

## Consequences

**A relevance-1 entity costs ~20 output tokens instead of ~2,300.** That is the entire scale-up: hundreds
of named, playable-enough figures become affordable, and depth is spent where the player is.

**`relevance` becomes a field the engine reads,** so it needs a home in `entity_registry` /
`*_state` — a migration, and per `ADR-P020` applied to production *before* the merge that needs it.

**The tier race is real and measured.** 2026-08-30, on eleven production worlds: entities were altered by
gameplay inside a 19-minute window on **three of five worlds that were played at all**, blast radius 1–3
entities, and **nothing at all inside the first 60 seconds**. So the danger is not committing early — it
is how long authoring continues afterwards. Background promotion must re-read live state at merge rather
than trusting a snapshot, and `entity_registry.status`/`actor_state.last_event_id` are the existing
hooks. Authoring a dead person's *past* is legitimate; authoring their *present* is the bug.

**Two mechanics, one field.** Genesis assigns and play promotes. Any future check must be able to say
which one it is implementing, or it will be re-deriving this decision by accident.

## Alternatives rejected

**One depth for everything, and parallelism to pay for it.** Rejected on measurement: 20-wide brings
~8.3M output tokens to ~110 minutes, not to acceptable. Volume is the problem; concurrency only divides
wall clock.

**"Obscurity", counting downward.** The founder's own correction. The scale reads 1 thin → 3 rich, and the
triggers promote *upward*; naming it obscurity inverted every rule written on top of it.

**Exclusivity as a flag on relevance 3 rather than a level.** My proposal, and wrong. Being bound to the
player is a genuine rung — it changes what content is owed *and* who may share an asset — and splitting it
into "level 3 plus a boolean" would have hidden the strongest thing about the model behind a flag name.

**Filtering a full authoring pass down to the visible parts.** Would have cost the full 8.3M tokens and
thrown most of them away. Relevance is an instruction given *before* authoring, never a filter applied
after.
