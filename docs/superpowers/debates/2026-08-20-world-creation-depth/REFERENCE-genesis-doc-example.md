# Reference — the genesis document as compiler IR (worked example + reader status)

Context for the "what must a full world contain" review round. This is the current working shape,
**not** an approved design. Attack it.

## The model

```
brief + answers  →  genesis seat  →  world_genesis JSON  →  commit  →  canon
   (source)         (front end)         (IR)               (codegen)   (binary)
```

Consequences already agreed:
- the seat emits **no uuid, tick, coordinate, or number of any kind** — quantities are **classes**, and a
  per-world table owns what each class means (`duration_class_seconds`, `extent_class_metres`, …).
  Founder principle: *"the author picks the class, the engine owns what it means in metres."*
- the belt validates the IR before codegen; there is no runtime and no repair loop, so validation is the
  only defence — a wrongly compiled world is permanently wrong (canon is append-only).
- the IR is discarded once the pipeline ends. Nothing from genesis is consulted during play.
- **an authored field with no reader is the core defect** this whole effort exists to fix
  (`cast[].standing` is authored, validated, and written nowhere).

## Worked example

Brief: *"A city half-swallowed by a tide that comes twice a day. People live in the upper floors and move
by water. Some can read when the water will turn, and everyone else depends on them."*

Deliberately **not** a hierarchy/caste world — anything that only works for ruling-class worlds is a
GA-2 violation.

```json
{
  "schema_version": "world_genesis/2",
  "world": { "display_name": "Vaunt", "tagline": "The water keeps the hours here.",
             "mood": ["patient","salt-stung","watchful"], "ornament": ["rope","brass","tide-mark"] },

  "movement": [
    { "canonical_name": "wade",  "descriptor": "pushing through water at chest height", "pace_class": "slow" },
    { "canonical_name": "swim",  "descriptor": "swimming the flooded runs",             "pace_class": "steady" },
    { "canonical_name": "climb", "descriptor": "going up the outside by rope and sill", "pace_class": "crawling" }
  ],

  "conditions": [
    { "canonical_name": "lame", "descriptor": "a leg that never set right",
      "hinders": [ { "movement": "wade", "hindrance_class": "severe" },
                   { "movement": "climb", "hindrance_class": "total" } ] }
  ],

  "places": [
    { "canonical_name": "Vaunt Shallows", "parent": null, "kind": "drowned city",
      "descriptor": "a city standing waist-deep in its own harbour", "extent_class": "large" },
    { "canonical_name": "The Stair Hall", "parent": "Vaunt Shallows", "kind": "flooded atrium",
      "descriptor": "a wide hall whose staircase disappears into green water",
      "description": "Brass rails, a tide-mark chest-high on every column, rope strung for handholds.",
      "tension": "normal", "extent_class": "medium" },
    { "canonical_name": "The Sunk Chapel", "parent": "Vaunt Shallows", "kind": "submerged shrine",
      "descriptor": "a chapel entirely under water",
      "description": "Pews in rows beneath the surface; light comes down in shafts and moves.",
      "tension": "frantic", "extent_class": "intimate" }
  ],

  "ways": [
    { "canonical_name": "the chapel mouth", "connects": ["The Stair Hall","The Sunk Chapel"],
      "descriptor": "an archway with no air behind it",
      "open": true, "locked": false, "impedes": ["walk","wade","climb"] }
  ],

  "collectives": [
    { "canonical_name": "The Water-Readers", "legibility": "marked",
      "descriptor": "people with a slate tied at the hip",
      "description": "They call when the tide turns; the city arranges its day around whether they are right." },
    { "canonical_name": "The Quiet Debt", "legibility": "concealed",
      "descriptor": "no outward sign at all",
      "description": "Those pulled from the water by someone else and not yet square. They know each other." }
  ],

  "rules": [
    { "canonical_name": "the drowned go unnamed",
      "stated": "No one says the name of a drowned person above water.", "binds": [] },
    { "canonical_name": "a false call ends a reader",
      "stated": "A reader who calls a tide wrong never calls again, and hands over the slate.",
      "binds": ["The Water-Readers"] }
  ],

  "cast": [
    { "canonical_name": "Perrin Vasque", "descriptor": "a woman with a wet slate at her hip",
      "role": "calls the turn of the tide for the whole lower city",
      "belongs_to": ["The Water-Readers"], "starts_in": "The Stair Hall",
      "doing": "scratching a new mark on the slate and not liking what it says",
      "wants": "to be believed through one more morning without having to be right",
      "speech_manner": "clipped, gives numbers before greetings",
      "moves_by": ["wade","swim"],
      "regard": [
        { "toward": "Old Hesk", "stance": "protective, and it costs her — she covers for how slow he is" },
        { "toward": "Wren Sil", "stance": "wary; the child watches her hands, not the water, when she reads" } ],
      "traits": [ { "key": "exacting", "strength": "strong", "manner": "corrects other people's timings aloud" } ],
      "malleability": "faint",
      "hiding": "She has called the last two tides from memory of the pattern, not from the water — and it is drifting." }
  ],

  "objects": [
    { "canonical_name": "the tide slate", "kind": "instrument", "size_class": "small",
      "descriptor": "a wet slate scratched with marks", "where": { "carried_by": "Perrin Vasque" } }
  ],

  "history": [
    { "what_happened": "A rising tide came early and a man cutting loose a snagged boat did not come back up.",
      "where": "The Stair Hall", "who": ["Old Hesk","Wren Sil"],
      "knowledge": [
        { "holder": "Old Hesk", "epistemic_type": "direct",
          "content": "He cut the rope himself to save the boat, and the man went under with it." },
        { "holder": "Wren Sil", "epistemic_type": "direct",
          "content": "She was under the surface and saw the cut end come down past her." },
        { "holder": "Perrin Vasque", "epistemic_type": "told",
          "content": "The tide came early and took someone; she believes she mistimed the call." } ] }
  ],

  "near_future": [
    { "canonical_name": "the turn of the tide", "stated": "The water starts to climb the stair again.",
      "when_class": "short", "involves": ["The Stair Hall","Perrin Vasque"],
      "sets": { "subject": "Vaunt Shallows", "attribute": "water", "value": "rising" } }
  ],

  "arrival": {
    "place": "The Stair Hall",
    "candidates": [
      { "canonical_name": "Sabe Orrin", "descriptor": "someone soaked to the shoulder",
        "why": "You came down the drowned stair looking for the chapel and took a wrong turn.",
        "moves_by": ["wade","swim"], "belongs_to": [] },
      { "canonical_name": "Sabe Orrin", "descriptor": "someone with a rope burn across both palms",
        "why": "You were pulled out of the water here last season and have not repaid it.",
        "moves_by": ["wade"], "belongs_to": ["The Quiet Debt"] } ] }
}
```

## Reader status — what lands today vs what is currently inert

| Field / cluster | Destination | Status |
|---|---|---|
| places, ways, objects, `tension`, `extent_class` | `entity_registry`, `location_state`, `artifact_state` | ✅ live |
| `traits`, `malleability`, `speech_manner` | `personality_core` | ✅ live |
| `hiding` | private `perception_record` | ✅ live |
| `history` + `knowledge` | `canon_event` + `perception_record` | ✅ live |
| `collectives` | `entity_registry` `entity_kind='group'` | ⚠️ legal, unused by genesis; note nothing walls a group's name |
| `rules` | prompt-rendered rows (soft) | ⚠️ needs the cognition/resolve/narrate prompts to carry them |
| `near_future` | `pending_event` | ⚠️ table live, read every clock crossing, written by tests only |
| `movement`, `moves_by`, `conditions.hinders` | `movement_type`, `status_modifier` | ⚠️ tables live + mintable, but `fn_move_duration_actor` **hardcodes `'walk'`** and no actor↔type binding exists |
| `ways.impedes` | portal gate | ❌ gate takes no mover — needs a ruling |
| `role`, `wants`, `doing` | — | ❌ **no reader anywhere** |
| `regard` | `relationship_state` | ❌ table exists, **zero readers**; `[RELATIONSHIPS]` block specced, never rendered |

Note the pattern: `role`, `wants`, `doing`, `regard` all want the **same** destination — the cognition
prompt, which today carries only location, tension, roster, own traits, private records, public moment,
fact sheet, imminent act.

## Standing constraints (law)

- **GA-2/GA-3:** the service must never learn what a world is "usually" like. No genre taxonomy, no
  template library, no fixed ontology of what worlds contain.
- No schema identifier traceable to a single brief. `movement_type_id` good (`swim` is a value);
  `caste`, `granter`, `standing_over` bad.
- Vocabulary is minted, open, per-world. Grammar is closed, ours, in code.
- Prevention emerges from comparison, never from an exemption list.
- Player arrives knowing nothing: exactly one perception at arrival.
- Cost: one genesis seat call, p50 ≤ $0.25, p95 ≤ 180 s.
