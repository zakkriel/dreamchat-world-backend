# T2 — Sueño Común in `world_model/2`

## 1. Encoding — detailed tier

```jsonc
{
  "world_model": "2",
  "world": { "name": "Orbe", "premise": "Everyone who sleeps in the same district enters the same dream, every night, without exception. You appear as you are. What you want manifests around you, visible to everyone. None of it is a crime; all of it is known by morning.",
             "mood": "cold, watched, intimate by obligation; all the damage is exposure" },

  "excluded": [
    "The dream does not predict. The user's anomaly is unique and no other scene may validate prediction as a rule.",
    "Nobody chooses what they dream. No technique, discipline, substance or teacher.",
    "There is no magic and there are no entities inside the dream. Only people from the district.",
    "Nobody dies or is injured inside. The damage is always social.",
    "Identity cannot be hidden inside. No trick works.",
    "Transcriptions are text. There is no image, recording, or way to revisit a past dream.",
    "Outside Orbe the phenomenon does not occur and no scene may suggest it spreads.",
    "Children under ten do not connect. No exceptions."
  ],

  "layers": [
    { "name": "waking", "default": true },
    { "name": "the dream", "default": false,
      "descriptor": "one shared dream per district, open while enough of the district sleeps" }
  ],

  "vocabulary": {
    "media": [ { "name": "street air", "resists": [] },
               { "name": "dream air", "layer": "the dream",
                 "resists": [ { "to": "concealment", "degree": "total" },
                              { "to": "injury", "degree": "total" } ] } ],
    "movements": [ { "name": "walk", "pace_class": "steady" },
                   { "name": "appear where the people are", "pace_class": "immediate",
                     "layer": "the dream", "note": "a late sleeper cannot choose where they arrive" } ],
    "channels": [
      { "name": "sight" }, { "name": "speech" },
      { "name": "the dream", "layer": "the dream",
        "descriptor": "everyone present sees everyone present, and what each of them wants",
        "emitted_by": "any sleeper of the district", "received_by": "any sleeper of the district, and any solitary",
        "latency_class": "immediate", "reach": "one district's dream", "decay": "three nights",
        "conceals": "none" },
      { "name": "the market look", "descriptor": "what the neighbourhood does with what it saw",
        "emitted_by": "anyone who was present", "received_by": "the district",
        "latency_class": "immediate", "reach": "one district", "decay": "never", "conceals": "identity" },
      { "name": "the volume", "descriptor": "the transcriber's daily book",
        "emitted_by": "the office of Transcriber", "received_by": "anyone",
        "latency_class": "slow", "reach": "the whole city", "decay": "never", "conceals": "none" } ],
    "conditions": [
      { "name": "asleep", "alters": [ { "channel": "the dream", "effect": "broadcast" },
                                      { "act": "concealing identity", "effect": "hinder", "class": "total" } ] },
      { "name": "solitary", "alters": [ { "channel": "the dream", "effect": "grant" },
                                        { "act": "being perceived in the dream", "effect": "immune" } ] },
      { "name": "projecting", "alters": [ { "channel": "the dream", "effect": "broadcast" } ],
        "note": "the waking form of the same broadcast: the district sees what the person is thinking" },
      { "name": "unconnected", "alters": [ { "channel": "the dream", "effect": "immune" } ],
        "note": "under ten years old" } ],
    "substances": [ { "name": "sleep" } ]
  },

  "law": [
    { "name": "one district, one dream", "enforced_by": "physics",
      "stated": "Everyone who sleeps in a district enters the same dream, every night, and the only way out is not sleeping." },
    { "name": "nobody hides inside", "enforced_by": "physics",
      "stated": "Face, body, age, the day's clothes. No mask, no absence, no visual lie.",
      "forbids": { "subject": "any entity with agency", "act": "concealing identity in the dream" } },
    { "name": "own memory lasts three nights", "enforced_by": "physics",
      "stated": "On the fourth day it is gone. After that only the volume remains, and someone else wrote it." },
    { "name": "the dream does not predict", "enforced_by": "physics",
      "stated": "No recorded exception in three hundred years.",
      "forbids": { "subject": "the dream", "act": "showing what has not happened" } },
    { "name": "another's dream", "enforced_by": "office", "binds": [],
      "stated": "Sleeping outside your assigned district is an offence: fine, reassignment, and publication of the file.",
      "precedent": null },
    { "name": "four nights unslept", "enforced_by": "physics",
      "stated": "From the fourth night you dream awake, in public, visible to the district. It accumulates and does not reverse on its own." },
    { "name": "you do not discuss another's dream in the street", "enforced_by": "persons", "binds": [],
      "stated": "Behind doors, yes. In the street, no." },
    { "name": "marriages are arranged between districts", "enforced_by": "persons", "binds": [],
      "stated": "Nobody explains why. Everybody knows why." },
    { "name": "you do not ask a solitary what they saw", "enforced_by": "persons", "binds": [],
      "stated": "Universally kept." },
    { "name": "district assignment at ten", "enforced_by": "office", "binds": [],
      "stated": "Before ten, children sleep disconnected." }
  ],

  "entities": [
    { "name": "Orbe", "facets": ["extent"], "layer": "waking", "extent_class": "vast",
      "medium": "street air",
      "seen_as": "four hundred thousand people, twenty-eight registered sleeping districts, and one that is not" },

    { "name": "Barrio Doce", "facets": ["extent"], "within": "Orbe", "layer": "waking",
      "extent_class": "large", "medium": "street air", "tension": "tense",
      "seen_as": "forty thousand people, a middling district, nothing notable until last night" },
    { "name": "Barrio Siete", "facets": ["extent"], "within": "Orbe", "layer": "waking",
      "extent_class": "large", "seen_as": "the district on Bald's papers, where he does not sleep" },
    { "name": "Barrio Veintinueve", "facets": ["extent"], "within": "Orbe", "layer": "waking",
      "extent_class": "large", "tension": "calm",
      "seen_as": "unregistered, no transcriber, no volume; an estimated six thousand people nobody counts" },

    { "name": "the dream of the Doce", "facets": ["extent", "demand"], "layer": "the dream",
      "extent_class": "large", "medium": "dream air", "tension": "tense",
      "seen_as": "a deformed version of the district, generated by whoever is in it, controlled by nobody",
      "demands": [ { "of": "sleep", "from": "the people of the Doce", "rate_class": "continuous",
                     "unmet": { "effect": "the dream closes", "onset_class": "immediate" } } ] },
    { "name": "the broken dream of the Veintinueve", "facets": ["extent"], "layer": "the dream",
      "extent_class": "large", "medium": "dream air",
      "seen_as": "it never sets; it comes apart into disconnected fragments, and the fragments are from other nights" },

    { "name": "the Office of the Doce", "facets": ["extent", "holding"], "within": "Barrio Doce",
      "layer": "waking", "extent_class": "small", "capacity_class": "small",
      "seen_as": "on the central square; the archive is open nine to sixteen" },
    { "name": "the reading room", "facets": ["extent"], "within": "the Office of the Doce",
      "layer": "waking", "extent_class": "intimate", "tension": "tense",
      "seen_as": "where you read what your neighbour dreamt; silence obligatory; always full" },
    { "name": "the Tarma boarding house", "facets": ["extent", "holding"], "within": "Barrio Doce",
      "layer": "waking", "extent_class": "small", "capacity_class": "moderate",
      "seen_as": "beds by the night for people from other districts; illegal; notorious" },
    { "name": "Onel's house", "facets": ["extent"], "within": "Barrio Doce", "layer": "waking",
      "extent_class": "intimate", "seen_as": "set apart, by rule of the Registry" },
    { "name": "the passage to the Twenty-Nine", "facets": ["passage"], "layer": "waking",
      "connects": ["Barrio Doce", "Barrio Veintinueve"],
      "admits": [ { "movement": "walk" } ], "hazard_class": "none",
      "seen_as": "an unwatched alley at the end of Ruma street that everybody can point to" },
    { "name": "the bell tower", "facets": ["extent", "matter"], "within": "Barrio Doce",
      "layer": "waking", "bulk_class": "immense", "integrity": "sound" },
    { "name": "the vigil bell", "facets": ["matter"], "within": "the bell tower", "layer": "waking",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "rung at five; it cuts the dream off at once" },

    { "name": "the volume of last night", "facets": ["record", "matter"], "layer": "waking",
      "within": "the Office of the Doce", "bulk_class": "slight",
      "authority": "the office of Transcriber of the Doce",
      "access": { "who": "anyone", "when": "nine, the following day" },
      "asserts": [ { "claim": "what the district dreamt, as one person chose to write it", "accurate": "unaudited" } ] },
    { "name": "the central Archive", "facets": ["extent", "holding", "record"], "within": "Orbe",
      "layer": "waking", "capacity_class": "vast", "authority": "the Office of Vigil",
      "access": { "who": "anyone", "when": "nine to sixteen" },
      "asserts": [ { "claim": "roughly twelve thousand volumes, one per night per district", "accurate": true } ] },
    { "name": "Rem's list", "facets": ["record", "matter"], "within": "Barrio Doce", "layer": "waking",
      "bulk_class": "slight", "authority": "Rem Salas", "access": { "who": "Rem Salas" },
      "asserts": [ { "claim": "eleven omissions, eleven names", "accurate": true } ] },
    { "name": "the incomplete volumes", "facets": ["record"], "within": "the central Archive",
      "layer": "waking", "authority": "the Office of Vigil", "access": { "who": "anyone" },
      "asserts": [ { "claim": "forty-one volumes with missing nights, across six districts, all within the last nine years", "accurate": true } ] },
    { "name": "the Registry of Solitaries", "facets": ["record"], "within": "the central Archive",
      "layer": "waking", "authority": "the Office of Vigil",
      "access": { "who": "the Office of Vigil", "note": "reserved, and everybody knows the names anyway" },
      "asserts": [ { "claim": "eleven names; fourteen applications to leave, none granted", "accurate": true } ] },
    { "name": "a district card", "facets": ["record", "matter"], "layer": "waking", "bulk_class": "slight",
      "authority": "the Office of Vigil", "access": { "who": "anyone who asks" },
      "asserts": [ { "claim": "where you are required to sleep", "accurate": true } ] },

    { "name": "the Office of Vigil", "facets": ["agency", "collective"], "layer": "waking",
      "legibility": "marked", "interest": "that nobody audits the transcriptions",
      "vulnerability": "every volume rests on one person, and that person is replaceable but not auditable" },
    { "name": "the Registry", "facets": ["agency", "collective"], "layer": "waking",
      "legibility": "marked", "interest": "keep the number at eleven — one solitary fewer is one trial fewer",
      "vulnerability": "there are eleven, they are named, and none of them wants to be there" },
    { "name": "the Council of Orbe", "facets": ["agency", "collective"], "layer": "waking",
      "legibility": "marked", "interest": "to govern",
      "vulnerability": "it cannot legislate on the dream because any debate would be transcribed" },
    { "name": "the Insomniacs", "facets": ["magnitude", "agency"], "within": "Orbe", "layer": "waking",
      "magnitude_class": "many", "legibility": "concealed",
      "seen_as": "official figure three hundred and forty; the real one is thought to be three thousand or more",
      "interest": "not to be seen", "vulnerability": "the method destroys them" },
    { "name": "the people of the Doce", "facets": ["magnitude", "agency"], "within": "Barrio Doce",
      "layer": "waking", "magnitude_class": "many",
      "seen_as": "forty thousand people who all saw the same thing last night" },

    { "name": "Rem Salas", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "waking",
      "seen_as": "the Doce's transcriber, forty-four, twelve years in the chair",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "conditions": ["asleep"],
      "disposition": [ { "trait": "scrupulous", "strength": "defining", "manner": "checks a line three times before it goes in" } ],
      "doing": "deciding whether last night's scene goes in the volume as she saw it",
      "pursuing": [ { "horizon": "long_standing", "toward": "never to be wrong" } ],
      "hiding": "She has omitted material from eleven volumes, always to protect someone, and she keeps the list at home." },
    { "name": "Onel", "facets": ["matter", "agency"], "within": "Onel's house", "layer": "waking",
      "seen_as": "solitary number four, thirty-eight, nineteen years registered",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "conditions": ["solitary"],
      "doing": "preparing a fifteenth application to be struck from the register",
      "pursuing": [ { "horizon": "long_standing", "toward": "the discharge" } ],
      "hiding": "He sees the dream in far more detail than he declares, and can follow one named person all night. The Office does not know." },
    { "name": "Vira Cor", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "waking",
      "seen_as": "twenty-six, eleven nights without sleep, and people are starting to stand further away",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "conditions": ["projecting"],
      "doing": "staying awake a twelfth night",
      "pursuing": [ { "horizon": "imminent", "toward": "never to sleep again" } ],
      "hiding": "What she does not want the district to see — and the district has begun to see it anyway." },
    { "name": "Inspector Bald", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "waking",
      "seen_as": "fifty-one, Office of Vigil, another-dream section",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" }, "conditions": ["asleep"],
      "doing": "building the case that closes the Tarma",
      "pursuing": [ { "horizon": "imminent", "toward": "close the Tarma boarding house" } ],
      "hiding": "He sleeps in the Doce. His card says Barrio Siete. Four years in breach of the law he enforces." },
    { "name": "Nea Salas", "facets": ["matter", "agency"], "within": "Barrio Doce", "layer": "waking",
      "seen_as": "seventeen, Rem's daughter, always halfway out the door",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" }, "conditions": ["asleep"],
      "pursuing": [ { "horizon": "long_standing", "toward": "leave Orbe" } ],
      "hiding": "She has slept three times in the Twenty-Nine, and knows the dream there does not fully break — it fragments, and the fragments are from other nights." },
    { "name": "Archivista Mayor Ossen", "facets": ["matter", "agency"], "within": "the central Archive",
      "layer": "waking", "seen_as": "sixty-three, counting the months",
      "pursuing": [ { "horizon": "imminent", "toward": "retire" } ],
      "hiding": "He knows exactly how many volumes have missing nights: forty-one, six districts, nine years." },
    { "name": "Solitary number one", "facets": ["matter", "agency"], "within": "Orbe", "layer": "waking",
      "seen_as": "seventy-one, no public name, registered since she was twelve",
      "conditions": ["solitary"],
      "hiding": "She is the only person left who was in the register before the reform, and knows how solitaries were selected then." }
  ],

  "offices": [
    { "name": "Transcriber of the Doce", "held_by": "Rem Salas", "of": "the Office of Vigil",
      "confers": [ { "act": "deciding what enters the volume" }, { "act": "closing the daily volume at noon" } ],
      "succeeds_by": "appointment by the Office of Vigil" },
    { "name": "Solitary number four", "held_by": "Onel", "of": "the Registry",
      "confers": [ { "act": "testifying at trial as full proof" } ],
      "succeeds_by": "never; no application to leave has been granted" },
    { "name": "Inspector of another-dream", "held_by": "Inspector Bald", "of": "the Office of Vigil",
      "confers": [ { "act": "opening an another-dream file" }, { "act": "demanding a district card" } ],
      "succeeds_by": "appointment" }
  ],

  "standing": [
    { "from": "the people of the Doce", "toward": "the arriving stranger",
      "stance": "everyone saw you standing next to the body and nobody will say so",
      "carried_by": "the market look", "persistence": "permanent" },
    { "from": "the Registry", "toward": "Onel", "stance": "useful and not releasable",
      "carried_by": null, "persistence": "until changed" },
    { "from": "the people of the Doce", "toward": "Vira Cor",
      "stance": "standing further away each day", "carried_by": "the market look", "persistence": "until changed" } ],

  "opposition": [
    { "between": ["Rem Salas", "the people of the Doce"],
      "incompatible": "She cannot both protect whoever she has been protecting and write a volume forty thousand witnesses can check against their own memory.",
      "stakes": "Her list of eleven omissions becomes the second story." },
    { "between": ["Inspector Bald", "the Tarma boarding house"],
      "incompatible": "He cannot close the house without the guest book being read.",
      "stakes": "His own name is in it." },
    { "between": ["Onel", "the Registry"],
      "incompatible": "He cannot be discharged while his testimony is full proof.",
      "stakes": "Fourteen refusals and a fifteenth being written." } ],

  "processes": [
    { "name": "memory going", "acts_on": "any holder's memory of a dream", "direction": "erase",
      "rate_class": "short", "terminus": "the fourth day" },
    { "name": "the district works it out", "acts_on": "standing toward the arriving stranger",
      "direction": "harden", "rate_class": "short", "terminus": "the volume opens at nine" } ],

  "cycles": [
    { "name": "the night", "period_class": "short", "starts_in_phase": "the reading room",
      "phases": [
        { "name": "the dream", "changes": [ { "entity": "the dream of the Doce", "becomes": "open" } ] },
        { "name": "the bell at five", "changes": [ { "entity": "the dream of the Doce", "becomes": "closed" } ] },
        { "name": "the market", "changes": [ { "entity": "Barrio Doce", "becomes": "tense" } ] },
        { "name": "the reading room", "changes": [ { "entity": "the volume of last night", "becomes": "public" } ] } ] } ],

  "accumulators": [
    { "name": "the district asleep", "per": "each entity with extent in layer the dream",
      "starts_at": "none", "stated": "How much of the district is asleep right now.",
      "raised_by": [ { "aggregate_of": "sleepers", "over": "the people of the district" } ],
      "thresholds": [ { "at": "low", "then": "the dream opens" },
                      { "at": "very low", "then": "the dream closes" } ] },
    { "name": "nights unslept", "per": "each entity with agency", "starts_at": "none",
      "stated": "How long since this person last slept.",
      "raised_by": [ { "event": "a night passed awake" } ],
      "thresholds": [ { "at": "low", "then": "condition:projecting — brief daytime projections" },
                      { "at": "moderate", "then": "frequent projections, visible at twenty metres" },
                      { "at": "high", "then": "continuous projection: the district sees what the person thinks" },
                      { "at": "extreme", "then": "collapse", "irreversible": true } ] } ],

  "indicators": [
    { "of": "nights unslept", "shows_as": ["a shape at the edge of sight that is not there", "people standing further off in the market"],
      "read_by": { "channel": "sight", "requires": null }, "reliability_class": "good" },
    { "of": "what a transcriber left out",
      "shows_as": ["a night everyone remembers and the volume does not", "a name that stops appearing"],
      "read_by": { "channel": "the volume", "requires": null }, "reliability_class": "poor" } ],

  "traces": [
    { "of": "a night dreamt", "leaves": "a volume in the Archive", "ages": "never" },
    { "of": "a night dreamt", "leaves": "the holder's own memory of it", "ages": "three nights" },
    { "of": "sleeping in the wrong district", "leaves": "a line in a boarding-house guest book", "ages": "never" },
    { "of": "a night in the Twenty-Nine", "leaves": "nothing in any register", "ages": "never" } ],

  "epochs": [
    { "name": "before the reform", "differed": [
        { "topic": "institution", "subject": "the Registry", "then": "solitaries were selected by another method" } ],
      "surviving_traces": ["Solitary number one, who is still alive"] },
    { "name": "the eighty years of study", "differed": [
        { "topic": "institution", "subject": "the phenomenon", "then": "under investigation" } ],
      "surviving_traces": ["nothing; the study was abandoned and nobody knows why it is urban"] } ],

  "history": [
    { "name": "the-murder-in-the-dream", "standing": "disputed",
      "what_happened": "The district dreamt a killing in a place that is not Orbe, with a dead man nobody recognised, and the arriving stranger standing beside the body.",
      "where": "the dream of the Doce", "who": ["the people of the Doce", "Onel", "Rem Salas"],
      "knowledge": [
        { "holder": "the people of the Doce", "channel": "the dream", "path": "direct",
          "believes": "They watched him stand there and look at it." },
        { "holder": "Onel", "channel": "the dream", "path": "direct",
          "believes": "More than he will say, because he was following.", "accurate": true },
        { "holder": "Rem Salas", "channel": "the dream", "path": "direct",
          "believes": "What she wrote, which is not everything she saw." } ] },
    { "name": "the-eleven-omissions", "standing": "occurred",
      "what_happened": "Over twelve years a transcriber left material out of eleven volumes, each time to protect someone.",
      "where": "the Office of the Doce", "who": ["Rem Salas"],
      "knowledge": [
        { "holder": "Rem Salas", "channel": "sight", "path": "direct", "believes": "Every one of them, and why." },
        { "holder": "Archivista Mayor Ossen", "channel": "the volume", "path": "inference",
          "believes": "Forty-one volumes have missing nights and he does not know whose fault each one is.",
          "accurate": true } ] } ],

  "arrivals": [
    { "premise": "Thirty-one, Barrio Doce, no record, ordinary work, never once relevant in a volume. Last night the district dreamt a murder and forty thousand people saw you standing beside the body. You remember being there. You do not remember doing anything. The volume opens at nine.",
      "seen_as": "someone the market is being careful not to look at",
      "within": "Barrio Doce", "layer": "waking",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "the dream": "normal" } } ]
}
```

## 2. Facet assignments and where I hesitated

| Entity | Facets | Hesitation |
|---|---|---|
| Orbe, districts, rooms | `extent` | No. |
| **the dream of the Doce** | `extent + demand` | **Severe.** Three defensible readings: (a) an entity in a non-default layer, which I took; (b) not an entity at all — the layer *is* the dream, twenty-eight of them, so `layers[]` would be per-district; (c) `extent + demand + magnitude`, since it is constituted by whoever is in it. I chose (a) because `demand` gives me the sleeper requirement and a layer cannot hold one. Another encoder taking (b) produces a structurally different document. |
| the broken dream of the Twenty-Nine | `extent` | Dropped `demand` because it never opens properly. Arbitrary. |
| A person | `matter + agency` | **Severe.** Awake they are matter; asleep *"nadie se lastima, nadie muere, nadie sangra"* — inside they have no matter at all. Facets are one flat list per entity, so I asserted the waking body and lost the fact that the dream body is not a body. |
| Onel, Solitary one | `matter + agency` + condition `solitary` | Considered making solitary an office instead — the Registry numbers them and refuses discharge. I did **both**, which is a smell. |
| the volume, Rem's list, a card | `record` (+ `matter` when takeable) | Same inconsistency I hit in Andantes. No rule. |
| the central Archive | `extent + holding + record` | Is a building full of records itself a record? I said yes, because it asserts its own catalogue. Thin. |
| institutions | `agency + collective` | No. |
| the Insomniacs | `magnitude + agency` | Also `legibility: concealed`, a `collective` key. I used a collective key on a non-collective; either `magnitude` should carry legibility or they need `collective` too. |
| the vigil bell | `matter` | It ends the dream for the whole district — arguably `passage` between layers. I did not, because `passage.connects` takes two extents and I could not decide whether a layer is one. |

## 3. Breaks

**B1 — facets cannot vary by layer. (i) inexpressible.** *"Nadie se lastima, nadie muere, nadie sangra"* (III). A person is `matter` awake and not-matter asleep. One flat facet list per entity means the schema cannot hold a thing whose *kind* differs between layers — which is what a two-layer world is for.

**B2 — position per layer. (i).** `within` is a single value. Vira Cor is in her room and in the dream simultaneously. I wrote her waking position and dropped the other. Every sleeper in the document has the same hole.

**B3 — hysteresis. (iii) wrong-shaped.** *"Apertura … 30% del barrio. Cierre … bajo el 10%."* Thresholds are an ordered ladder crossed in one direction. Opening at one level and closing at a lower one is two ladders, rising and falling. I wrote `low` then `very low` in the same list, which reads as a monotone descent and is simply wrong.

**B4 — knowledge that expires. (i).** *"Memoria propia: tres noches. Al cuarto día se borra."* Rule 3, and the engine of the whole plot — after three nights only the volume remains and someone else wrote it. `history[].knowledge` has no expiry. I faked it with a `process` acting on "any holder's memory", which no reader consumes, and a `trace` with `ages: three nights`, which is about residue in the world, not in a head.

**B5 — involuntary broadcast of interior state. (iii).** *"Lo que uno desea se manifiesta alrededor suyo, visible para todos"* — the stated central mechanic. `conditions[].alters` offers `broadcast`, but it names a **channel**, not a **content**. There is no way to say the channel discloses the holder's `pursuing` and `hiding`. So the one thing this world is about is a condition with a verb and no object.

**B6 — a record that omits. (ii) inert prose.** *"El transcriptor decide qué entra. No hay auditoría."* `record.asserts` has no incompleteness or falsity key; I wrote `accurate: "unaudited"`, inventing both the key and a third value. **This is the same break I hit in Andantes with the falsified bulletin** — two unrelated worlds, same missing primitive. That convergence is the strongest signal in this test.

**B7 — one office, many concurrent holders. (iii).** Twenty-eight transcribers, eleven numbered solitaries. `offices[].held_by` is singular, so I authored one office per holder and the *class* of office — the thing the law and the norms actually refer to — exists nowhere.

**B8 — law scoped to a place. (ii).** The Twenty-Nine's dream fragments and *"los fragmentos son de otras noches"*. `law[]` has `binds[]`, which reads as binding persons. I put the fact in the entity's `seen_as`, i.e. prose.

**B9 — exact thresholds forced into classes. (iii).** 30% and 10% are engine-computed, so v2 forbids them; I wrote `low` / `very low` and lost the precision the brief states outright. The new number rule is right in direction and gives no ladder for proportion.

**B10 — content crossing between instances. (i).** *"los fragmentos son de otras noches"*, and open line 1: the same scene appearing in Barrio Siete three nights later. Nothing relates one night's dream to another's.

**Fixed by v2:** `excluded[]` takes section X almost verbatim, including the uniqueness of the anomaly — and this world's negative canon is load-bearing. `history[].standing: "disputed"` holds the murder exactly. `indicators[]` holds insomnia's visible signs. `conditions[].alters` with `immune` holds *"no los ve nadie"* cleanly.

## 4. Ambiguity report

1. **Is the dream an entity or a layer?** Two clean readings, structurally different documents. Disambiguate by stating whether `layers[]` entries may be instantiated per place. This is the single largest divergence risk in this world.
2. **Per-layer facets and per-layer position.** Undeclared, and unavoidable here. Either entities gain a per-layer block or layers are not usable.
3. **`solitary` as a condition or an office.** The Registry numbers them, confers courtroom standing, and refuses discharge — all office behaviour. But it is also a property of the person's perception. I encoded both.
4. **Does `conditions[].alters.immune` mean immune to performing an act, or to having it performed on you?** *"No los ve nadie"* is the second. The examples only show the first.
5. **`accumulators.per` scoping to a layer.** I wrote `per: "each entity with extent in layer the dream"`, inventing the syntax outright.
6. **What is `aggregate_of` allowed to name?** The example aggregates `bulk_class`, a facet key. I aggregated "sleepers", a state. Undefined.
7. **`legibility` on a `magnitude` that is not a `collective`.** Used it anyway.
8. **Are the twenty-eight districts one entity with `magnitude` or twenty-eight entities?** I wrote three and implied the rest — the same divergence I hit with nine Andantes.
9. **Does `channel.decay` mean the record fades or the *memory of receiving it* fades?** I used it for memory (`decay: "three nights"`) because nothing else could hold rule 3. That is probably a misreading, and it is load-bearing.

## 5. Convergence check

**No new top-level section — but that is not good news.** `layers[]` exists and is specified by exactly two keys and one sentence. It is the section this world is entirely about, and it carries no rule for per-layer position, per-layer facets, per-layer law, or whether layers instantiate. An empty section that looks solved is worse than a missing one: I could not tell whether my encoding was wrong or merely unguided, and neither will the other encoder.
