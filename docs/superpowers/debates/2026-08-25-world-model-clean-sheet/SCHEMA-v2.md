# world_model/2 — candidate contract

Rebuilt from the four-world stress test. **Under test, not approved.**

## What changed and why

v1's `vocabulary` half was agnostic; its content half handed the world **a fixed cast of eight kinds**
(`places`, `things`, `people`, `collectives`…) — a closed ontology wearing open clothes. A living house,
a walking creature carrying cities, and a dream that exists only while enough people sleep all broke it
the same way: an entity's kind was *which array it was in*.

**The core change: one `entities[]`, kind expressed as composable facets.** Four fixed kinds → one.

## Grammar (closed, ours) vs vocabulary (open, the world's)

| Closed — we own it | Open — the world invents it |
|---|---|
| facet names | media, movements, channels, conditions, substances |
| class ladders (`slow`, `vast`, `severe`) | every entity, law, office, indicator |
| `path`, `standing`, `conceals`, `direction` enums | all names, all prose |

## Facets

An entity holds any combination. Facets are the only fixed kinds in the schema.

| Facet | Grants | Keys |
|---|---|---|
| `extent` | it has an interior; things can be within it | `extent_class`, `medium`, `tension`, `within` |
| `matter` | it is physical | `bulk_class`, `integrity`, `size_class` |
| `agency` | it decides and acts | `disposition[]`, `doing`, `pursuing[]`, `hiding` |
| `holding` | it contains substances or entities | `capacity_class`, `holds[]` |
| `demand` | it requires something to keep going | `demands[]` |
| `passage` | it joins two extents | `connects[2]`, `admits[]`, `obstructs[]`, `hazard_class` |
| `borne` | its position is another entity's | `borne_by` |
| `motion` | it moves on its own | `trajectory{period_class, phase_at_start}` |
| `collective` | it is constituted of members | `legibility`, `interest`, `vulnerability` |
| `magnitude` | it stands for many, individuals promotable on demand | `magnitude_class` |
| `record` | its content is a claim | `asserts[]`, `access{}`, `authority` |

A living house = `extent + matter + agency + demand`. A walking creature carrying cities =
`extent + matter + motion + demand`. A guild = `agency + collective`. A door = `matter + passage`.
A crowd = `magnitude + agency`. A tent = `matter`.

## Sections

```
world           name, premise, mood
excluded[]      what does not exist or cannot happen here — binding on every authoring seat
layers[]        concurrent rule-realities; default one. law/entities/channels declare membership
vocabulary      media[] · movements[] · channels[] · conditions[] · substances[]
law[]           what this world permits, physical or social
entities[]      recursive; facets above; the only container of nouns
offices[]       authority that outlives its holder
standing[]      directed relation between any two entities
opposition[]    stated incompatibility between two or more entities
processes[]     ongoing change: acts_on, direction, rate_class, terminus
cycles[]        ordered phases, period_class, what each changes
accumulators[]  scope + ORDERED threshold ladder
indicators[]    a hidden state, its signs, what reads them, how reliably
traces[]        residue a change leaves, and how it ages
epochs[]        a structurally different past
history[]       events; per-holder knowledge; an event may be `disputed`
arrivals[]      plural premises; no opening state
```

### Rule changes from v1

- **Numbers** are forbidden only in fields the engine computes on. A figure a player *reads* — a register's
  count, a dated chronology — is fiction and may be written.
- **`conditions[].alters`** replaces `hinders`: `hinder | grant | broadcast | immune`, each naming a
  movement, channel, or act-class.
- **`channels[]`** carry `emitted_by`, `received_by`, `latency_class`, `reach`, `decay`, and
  `conceals: all | identity | none`. Propagation is no longer its own section.
- **`law[]`** absorbs norms: every entry has `stated`, `binds[]`, `enforced_by` (`physics | persons |
  office`), optional `forbids{subject, act}` so an impossibility is a comparison, and `precedent`.
- **Passage compares predicates.** `admits[]`/`obstructs[]` entries may name a movement, a condition, a
  standing, an office, or a layer — not only a gait.
- **`demands[]`** generalises v1's `ambient_demand`: any subject may require any substance at a rate
  class, supplied by draw *or* by emission from activity, with an `unmet{effect, onset_class}`.
- **Capability is conferrable** — `things`/`offices` may `confer` an act or channel.
- **`ways` and `stocks` are gone**: a way is an entity with `passage`; a stock is a substance held by an
  entity with `holding`.

## Illustrative fragments

```jsonc
{
  "world_model": "2",
  "world": { "name": "Ribsdown", "premise": "A town inside the ribcage of something that died here." },

  "excluded": [
    "There is no way to make cut bone sound again.",
    "Nothing living remains of the animal. It does not wake."
  ],

  "layers": [ { "name": "waking", "default": true } ],

  "vocabulary": {
    "media": [ { "name": "marrow-damp", "resists": [ { "to": "sight", "degree": "moderate" } ] } ],
    "movements": [ { "name": "walk", "pace_class": "steady" },
                   { "name": "squeeze", "pace_class": "slow" } ],
    "channels": [ { "name": "the bone", "descriptor": "knocks carried through the structure",
        "emitted_by": "any entity with matter touching bone", "received_by": "anyone in contact",
        "latency_class": "immediate", "reach": "contiguous bone", "decay": "never",
        "conceals": "identity" } ],
    "conditions": [ { "name": "chalk-lung",
        "alters": [ { "movement": "climb", "effect": "hinder", "class": "severe" } ] } ],
    "substances": [ { "name": "sound bone" } ]
  },

  "law": [
    { "name": "the bone remembers load", "enforced_by": "physics",
      "stated": "Cut bone never regains strength.",
      "forbids": { "subject": "any entity with matter", "act": "restoring integrity to cut bone" } },
    { "name": "no cut without a reading", "enforced_by": "persons", "binds": [],
      "stated": "Nobody opens a standing rib until the Measure has hung a line on it.",
      "precedent": "the-winter-shoring" }
  ],

  "entities": [
    { "name": "The Nave", "facets": ["extent"],
      "within": "Ribsdown", "extent_class": "medium", "medium": "marrow-damp", "tension": "normal" },

    { "name": "the third rib", "facets": ["matter"],
      "within": "The Nave", "bulk_class": "immense", "integrity": "worn",
      "supports": [ { "entity": "The Nave", "provides": "standing room" } ] },

    { "name": "the seep stair", "facets": ["matter", "passage"],
      "connects": ["The Nave", "The Low Seeps"],
      "obstructs": [ { "movement": "walk" } ],
      "admits": [ { "movement": "squeeze" }, { "standing": "pact", "held_by": "self" } ] },

    { "name": "Adren Kel", "facets": ["matter", "agency"],
      "within": "The Nave", "seen_as": "a spare woman with a plumb-line at her wrist",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "the bone": "acute" },
      "disposition": [ { "trait": "exacting", "strength": "defining",
                         "manner": "re-measures rather than answer" } ],
      "doing": "hanging the line against the third rib for the second time",
      "pursuing": [ { "horizon": "long_standing", "toward": "get the town to move outward" } ],
      "hiding": "The lean has already passed the mark she told them was the limit." },

    { "name": "the Measure", "facets": ["agency", "collective"],
      "legibility": "marked", "interest": "that the town never learns how far the lean has gone",
      "vulnerability": "its authority rests on a reading it has already shaded once" },

    { "name": "the Pactless", "facets": ["magnitude", "agency"],
      "within": "Ribsdown", "magnitude_class": "many",
      "seen_as": "people living under canvas in the square" }
  ],

  "offices": [
    { "name": "Line-holder", "held_by": "Adren Kel", "of": "the Measure",
      "confers": [ { "act": "declaring a rib cuttable" } ], "succeeds_by": "appointment by the Measure" }
  ],

  "standing": [
    { "from": "Adren Kel", "toward": "Bettin Roe",
      "stance": "needs him and cannot say so", "since": "the-winter-shoring",
      "carried_by": null, "persistence": "until changed" }
  ],

  "accumulators": [
    { "name": "the lean", "per": "world", "starts_at": "low",
      "stated": "How far the standing ribs have gone out of true.",
      "raised_by": [ { "event": "a cut into a supporting rib" },
                     { "aggregate_of": "bulk_class", "over": "everything within The Nave" } ],
      "thresholds": [
        { "at": "moderate", "then": "the Measure starts refusing cuts" },
        { "at": "high",     "then": "The Nave stops being safe to stand in", "irreversible": true } ] }
  ],

  "indicators": [
    { "of": "the lean", "shows_as": ["doors that stick", "dust falling in a still room", "a rib singing"],
      "read_by": { "channel": "sight", "requires": { "office": "Line-holder" } },
      "reliability_class": "poor" }
  ],

  "history": [
    { "name": "the-winter-shoring", "standing": "occurred",
      "what_happened": "A rib was cut without a reading and the chamber came down on two people.",
      "where": "The Nave", "who": ["Adren Kel", "Bettin Roe"],
      "knowledge": [
        { "holder": "Adren Kel", "channel": "sight", "path": "direct",
          "believes": "She approved the reading and got it wrong." },
        { "holder": "Bettin Roe", "channel": "sound", "path": "told",
          "believes": "The cutter went in without a reading at all.",
          "accurate": false, "plausible_because": "That is what the Measure said afterwards." } ] }
  ]
}
```

## Fixed-kind count — the convergence metric

| | v1 | v2 |
|---|---|---|
| fixed entity kinds | 4 (`places`,`things`,`people`,`collectives`) | **1**, with 11 open-composed facets |
| top-level sections | 19 | 17 |
| sections absorbed | — | `ways`, `stocks`, `norms`, `propagation` |
| sections added | — | `excluded`, `layers`, `offices`, `standing`, `indicators` |

The closed core shrank where it counts. **If v3 needs a new top-level section per new world, the approach
is failing.**
