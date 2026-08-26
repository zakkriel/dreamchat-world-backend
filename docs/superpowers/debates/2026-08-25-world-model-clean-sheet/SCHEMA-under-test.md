# Schema under test — world_model/1 (candidate)

This is the candidate genesis-document schema produced from the clean-sheet round. **It is under test,
not approved.** Your job is to break it against a real world brief.

## Structural decisions

1. **Two halves: a `vocabulary` the world invents, then content that references it.** This is what makes
   agnosticism structural. A world declares its own media, movements, channels, conditions and
   substances; everything else points at those names. The schema has never heard of water, houses,
   dreams or castes — only that a place has a medium and a medium resists things.
2. **No number anywhere.** Classes and prose only; the engine resolves a class to a number per world.
3. **Canonical names are the only join keys.** The engine mints every id.
4. **Places recurse** — one array, each naming its parent. No region/places split.
5. **World-motion is peer to people** — processes, cycles, accumulators, traces, propagation sit at top
   level because they run whether or not anyone is present.
6. **Knowledge is per-holder, carries a channel and a path, and may be false.** Falsity is authored.
7. **Arrival is plural.** There is no "opening state".

## Section list

```
world            name, premise, mood
vocabulary       media[] · movements[] · channels[] · conditions[] · substances[]
law[]            what matter does HERE, stated as comparisons
places[]         name, parent, extent_class, sort, medium, tension, ambient_demand[]
ways[]           connects[2], state, obstructs[], affords[]
things[]         bulk_class, integrity, capacity_class, where{}, supports[], depends_on[]
stocks[]         held_in, abundance, drawn_by, replenished_by
processes[]      acts_on, direction, rate_class, terminus
cycles[]         phases[] in order, period_class, starts_in_phase
accumulators[]   stated, starts_at, raised_by[], threshold{class, then}
traces[]         of, leaves, ages
propagation[]    of, spreads
collectives[]    legibility, descriptor, interest, speaks_through
people[]         seen_as, role, belongs_to[], starts_in, capability{moves_by[], carry_class},
                 senses{}, conditions[], disposition[], doing, pursuing[], obligation[],
                 regard[], hiding
opposition[]     between[2], incompatible, stakes
norms[]          stated, binds[], precedent
epochs[]         differed[], surviving_traces[]
history[]        what_happened, where, who[], knowledge[{holder, channel, path, believes,
                 accurate, plausible_because}]
arrivals[]       premise, seen_as, place, capability{}, senses{}
```

## Worked example (abridged) — a town inside a ribcage

```jsonc
{
  "world_model": "1",
  "world": { "name": "Ribsdown",
    "premise": "A town built inside the ribcage of something enormous that died here. The bone is the only building material, and cutting it is how the town grows and how it falls in." },

  "vocabulary": {
    "media": [ { "name": "marrow-damp", "descriptor": "still air that smells of wet chalk",
        "resists": [ { "to": "sight", "degree": "moderate" }, { "to": "open flame", "degree": "total" } ],
        "affords": [ { "to": "sound", "degree": "full" } ] } ],
    "movements": [ { "name": "walk", "pace_class": "steady" },
                   { "name": "squeeze", "pace_class": "slow", "note": "only bodies of slight bulk" } ],
    "channels": [ { "name": "sight" }, { "name": "sound" },
                  { "name": "the bone", "descriptor": "knocks carried through the structure itself",
                    "note": "reaches anywhere sharing a rib, including sealed spaces" } ],
    "conditions": [ { "name": "chalk-lung", "hinders": [ { "movement": "climb", "class": "severe" } ] },
                    { "name": "bone-deaf",  "hinders": [ { "channel": "the bone", "class": "total" } ] } ],
    "substances": [ { "name": "sound bone" }, { "name": "seep water" } ]
  },

  "law": [ { "name": "the bone remembers load",
      "stated": "Cut bone never regains strength. What a rib carried before, it cannot carry again once opened.",
      "governs": "integrity" } ],

  "places": [
    { "name": "The Nave", "parent": "Ribsdown", "extent_class": "medium", "sort": "main cavity",
      "medium": "marrow-damp", "tension": "normal" },
    { "name": "The Low Seeps", "parent": "Ribsdown", "extent_class": "small", "medium": "marrow-damp",
      "ambient_demand": [ { "requires": "dry footing", "absent_effect": "condition:chalk-lung",
                            "onset": "exposure" } ] } ],

  "ways": [ { "name": "the seep stair", "connects": ["The Nave", "The Low Seeps"],
      "state": "open", "obstructs": ["walk"], "affords": ["climb", "squeeze"] } ],

  "things": [ { "name": "the third rib", "bulk_class": "immense", "integrity": "worn",
      "where": { "in_place": "The Nave" },
      "supports": [ { "place": "The Nave", "provides": "standing room" } ] } ],

  "stocks": [ { "name": "sound bone", "held_in": "Ribsdown", "abundance": "thin",
                "drawn_by": "anyone cutting", "replenished_by": null } ],

  "processes": [ { "name": "settling", "acts_on": "anything with integrity in marrow-damp",
                   "direction": "degrade", "rate_class": "very slow", "terminus": null } ],

  "cycles": [ { "name": "the drawing down", "period_class": "long", "starts_in_phase": "full",
      "phases": [ { "name": "full",    "changes": [ { "stock": "seep water", "becomes": "adequate" } ] },
                  { "name": "dry",     "changes": [ { "medium_of": "The Low Seeps", "becomes": "outside air" } ] } ] } ],

  "accumulators": [ { "name": "the lean", "stated": "How far the standing ribs have gone out of true.",
      "starts_at": "low", "raised_by": ["every cut into a supporting rib", "settling"],
      "threshold": { "class": "high", "then": "The Nave stops being safe to stand in." } } ],

  "traces": [ { "of": "cutting bone", "leaves": "a bright unweathered face and a drift of dust",
                "ages": "slowly" } ],

  "propagation": [ { "of": "a rib being struck", "spreads": "everywhere sharing the bone, at once" },
                   { "of": "a cut being made",   "spreads": "at travel speed" } ],

  "collectives": [ { "name": "the Measure", "legibility": "marked",
      "interest": "that the town never learns how far the lean has already gone",
      "speaks_through": "Adren Kel" } ],

  "people": [ { "name": "Adren Kel", "seen_as": "a spare woman with a plumb-line at her wrist",
      "role": "reads the lean and says what may be cut", "belongs_to": ["the Measure"],
      "starts_in": "The Nave",
      "capability": { "moves_by": ["walk", "climb"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the bone": "acute" },
      "disposition": [ { "trait": "exacting", "strength": "defining",
                         "manner": "re-measures rather than answer a question" } ],
      "doing": "hanging the line against the third rib for the second time this morning",
      "pursuing": [ { "horizon": "long_standing", "toward": "get the town to move outward",
                      "progress": "early", "step": "building a record nobody can argue with" } ],
      "obligation": [ { "owed_to": "the Measure", "stated": "does not publish a reading first" } ],
      "regard": [ { "toward": "Bettin Roe", "stance": "needs him and cannot say so", "since": "the-winter-shoring" } ],
      "hiding": "The lean has already passed the mark she told them was the limit." } ],

  "opposition": [ { "between": ["Adren Kel", "Bettin Roe"],
      "incompatible": "The third rib cannot both be measured intact and cut through this morning.",
      "stakes": "If it goes, the Nave's lean crosses and the town has to move." } ],

  "norms": [ { "name": "no cut without a reading", "binds": [],
      "stated": "Nobody opens a standing rib until the Measure has hung a line on it.",
      "precedent": "the-winter-shoring" } ],

  "epochs": [ { "name": "when the ribs were whole",
      "differed": [ { "topic": "stock", "subject": "sound bone", "then": "abundant" } ],
      "surviving_traces": ["the uncut ribs on the seaward side"] } ],

  "history": [ { "name": "the-winter-shoring",
      "what_happened": "A rib was cut without a reading and the chamber came down on two people.",
      "where": "The Nave", "who": ["Adren Kel", "Bettin Roe"],
      "knowledge": [
        { "holder": "Adren Kel", "channel": "sight", "path": "direct",
          "believes": "She approved the reading that allowed it and got it wrong." },
        { "holder": "Bettin Roe", "channel": "sound", "path": "told",
          "believes": "The cutter went in without a reading at all.",
          "accurate": false, "plausible_because": "That is what the Measure said afterwards." } ] } ],

  "arrivals": [
    { "premise": "You came in on the bone road with a commission to buy cut stock, and nobody will quote you a price.",
      "seen_as": "someone in road clothes holding a folded order", "place": "The Nave",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the bone": "absent" } } ]
}
```

## Known-weak points I already suspect

- `stocks` are **place-held**. A stock held by a *person*, depleting, possibly hidden from them, and
  spent by a specific kind of act, has no expression.
- There is no way to say **an act costs a resource** or **an act inflicts a condition**.
- `ambient_demand` is **binary per place** — no graduated effect along a continuum.
- `places` are **static containers**: they cannot move, act, want, sicken, die, or gain and lose rooms.
- No concept of **negative canon** — what does not exist in this world.
- No **office/role held by a rotating person**, distinct from the person holding it.
- No **immunity** — `conditions` only hinder.
