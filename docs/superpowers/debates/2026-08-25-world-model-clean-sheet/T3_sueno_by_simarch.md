# T3 — Sueño Común under `world_model/3`

## 1. Encoding

```jsonc
{
  "world_model": "3",
  "world": { "name": "Orbe", "premise": "Everyone who sleeps in the same district enters the same dream, every night. You appear as you are. What you want manifests around you, visible to all. None of it is a crime; all of it is known by morning.",
             "mood": "cold, watched, intimate by obligation; all the damage is exposure" },

  "excluded": [
    "The dream does not predict. The arriving stranger's anomaly is unique and no other scene may validate prediction as a rule.",
    "Nobody chooses what they dream. No technique, discipline, substance or teacher.",
    "There is no magic and there are no entities inside the dream. Only people of the district.",
    "Nobody dies or is injured inside. The damage is always social.",
    "Identity cannot be hidden inside. No trick works.",
    "Transcriptions are text. No image, no recording, no way to revisit a past dream.",
    "Outside Orbe the phenomenon does not occur, and no scene may suggest it spreads.",
    "Children under ten do not connect."
  ],

  "vocabulary": {
    "media": [
      { "name": "street air", "resists": [] },
      { "name": "dream air", "resists": [ { "to": "concealing identity", "degree": "total" },
                                          { "to": "injury", "degree": "total" } ] } ],
    "movements": [
      { "name": "walk", "pace_class": "steady" },
      { "name": "falling asleep", "pace_class": "immediate" } ],
    "channels": [
      { "name": "sight", "emitted_by": "anything present", "received_by": "anyone present",
        "latency_class": "immediate", "reach": "one extent", "decay": "never", "conceals": "none" },
      { "name": "the dream", "descriptor": "everyone present sees everyone present, and what each of them wants",
        "emitted_by": "anyone within the dream of a district", "received_by": "anyone within it, and any solitary of that district",
        "latency_class": "immediate", "reach": "one district's dream", "decay": "brief", "conceals": "none" },
      { "name": "the market look", "descriptor": "what a district does with what it saw",
        "emitted_by": "anyone who was present", "received_by": "the district",
        "latency_class": "immediate", "reach": "one district", "decay": "never", "conceals": "identity" },
      { "name": "the volume", "descriptor": "the transcriber's daily book",
        "emitted_by": "a holder of a Transcriber office", "received_by": "anyone",
        "latency_class": "short", "reach": "the whole city", "decay": "never", "conceals": "none" } ],
    "conditions": [
      { "name": "solitary", "alters": [ { "channel": "the dream", "effect": "grant" },
                                        { "act": "being perceived in the dream", "effect": "immune" } ] },
      { "name": "projecting", "alters": [ { "channel": "the dream", "effect": "broadcast" } ] },
      { "name": "unconnected", "alters": [ { "channel": "the dream", "effect": "immune" } ] } ],
    "substances": [ { "name": "sleep" } ]
  },

  "law": [
    { "name": "one district, one dream", "enforced_by": "physics", "within": "Orbe",
      "stated": "Everyone who sleeps in a district enters that district's dream, every night. The only way out is not sleeping." },
    { "name": "nobody hides inside", "enforced_by": "physics", "within": "the dream of the Doce",
      "stated": "Face, body, age, the day's clothes. No mask, no absence, no visual lie.",
      "forbids": { "subject": "any entity with agency", "act": "concealing identity" } },
    { "name": "nobody is hurt inside", "enforced_by": "physics", "within": "the dream of the Doce",
      "stated": "Nobody bleeds, nobody dies, nobody is injured.",
      "forbids": { "subject": "any entity with agency", "act": "injuring another" } },
    { "name": "own memory lasts three nights", "enforced_by": "physics", "within": "Orbe",
      "stated": "On the fourth day it is gone. After that only the volume remains, and someone else wrote it." },
    { "name": "the dream does not predict", "enforced_by": "physics", "within": "Orbe",
      "stated": "No recorded exception in three hundred years.",
      "forbids": { "subject": "the dream of the Doce", "act": "showing what has not happened" } },
    { "name": "another's dream", "enforced_by": "office", "within": "Orbe", "binds": [],
      "stated": "Sleeping outside your assigned district is an offence: fine, reassignment, and publication of the file." },
    { "name": "four nights unslept", "enforced_by": "physics", "within": "Orbe",
      "stated": "From the fourth night you dream awake, in public, visible to the district. It accumulates and does not reverse on its own." },
    { "name": "you do not discuss another's dream in the street", "enforced_by": "persons", "within": "Orbe", "binds": [],
      "stated": "Behind doors, yes. In the street, no." },
    { "name": "you do not ask a solitary what they saw", "enforced_by": "persons", "within": "Orbe", "binds": [],
      "stated": "Universally kept." },
    { "name": "assignment at ten", "enforced_by": "office", "within": "Orbe", "binds": [],
      "stated": "Before ten, children sleep unconnected." }
  ],

  "entities": [
    { "name": "Orbe", "facets": ["extent"], "extent_class": "vast", "medium": "street air",
      "seen_as": "four hundred thousand people; twenty-eight registered sleeping districts, and one that is not" },

    { "name": "Barrio Doce", "facets": ["extent"], "within": "Orbe", "extent_class": "large",
      "medium": "street air", "tension": "tense",
      "seen_as": "forty thousand people, a middling district, nothing notable until last night" },
    { "name": "Barrio Veintinueve", "facets": ["extent"], "within": "Orbe", "extent_class": "large",
      "medium": "street air", "tension": "calm",
      "seen_as": "unregistered: no transcriber, no volume; an estimated six thousand people nobody counts" },

    { "name": "the dream of the Doce", "facets": ["extent", "demand"], "within": "Barrio Doce",
      "extent_class": "large", "medium": "dream air", "tension": "tense",
      "seen_as": "a deformed version of the district, generated by whoever is inside it and controlled by nobody",
      "demands": [ { "substance": "sleep", "from": "the sleepers of the Doce", "rate_class": "continuous",
                     "unmet": { "effect": "the dream ends and everyone within it leaves", "onset_class": "immediate" } } ] },
    { "name": "the dream of the Veintinueve", "facets": ["extent"], "within": "Barrio Veintinueve",
      "extent_class": "large", "medium": "dream air", "tension": "calm",
      "seen_as": "it never sets; it comes apart into disconnected fragments, and the fragments are from other nights" },

    { "name": "falling asleep in the Doce", "facets": ["passage"],
      "connects": ["Barrio Doce", "the dream of the Doce"],
      "admits": [ { "movement": "falling asleep" } ],
      "obstructs": [ { "movement": "walk" }, { "condition": "unconnected" } ],
      "hazard_class": "none" },
    { "name": "the passage to the Twenty-Nine", "facets": ["passage"],
      "connects": ["Barrio Doce", "Barrio Veintinueve"],
      "admits": [ { "movement": "walk" } ], "hazard_class": "none",
      "seen_as": "an unwatched alley at the end of Ruma street that everybody can point to" },

    { "name": "the Office of the Doce", "facets": ["extent"], "within": "Barrio Doce",
      "extent_class": "small", "seen_as": "on the central square; the archive is open nine to sixteen" },
    { "name": "the reading room", "facets": ["extent"], "within": "the Office of the Doce",
      "extent_class": "intimate", "tension": "tense",
      "seen_as": "where you read what your neighbour dreamt; silence obligatory; always full" },
    { "name": "the Tarma boarding house", "facets": ["extent"], "within": "Barrio Doce",
      "extent_class": "small", "seen_as": "beds by the night for people from other districts; illegal; notorious" },
    { "name": "the central Archive", "facets": ["extent"], "within": "Orbe", "extent_class": "medium",
      "seen_as": "roughly twelve thousand volumes, one per night per district" },
    { "name": "the vigil bell", "facets": ["matter"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "rung at five; it wakes the district at once" },

    { "name": "the volume of last night", "facets": ["record", "matter"], "within": "the Office of the Doce",
      "bulk_class": "slight", "integrity": "sound",
      "authority": "Transcriber of the Doce", "access": { "who": "anyone", "when": "nine, the following day" },
      "asserts": [ { "claim": "what the district dreamt, as one person chose to write it", "accurate": false } ] },
    { "name": "Rem's list", "facets": ["record", "matter"], "within": "Barrio Doce", "bulk_class": "slight",
      "integrity": "sound", "authority": "Rem Salas", "access": { "who": "Rem Salas" },
      "asserts": [ { "claim": "eleven omissions, eleven names", "accurate": true } ] },
    { "name": "the incomplete volumes", "facets": ["record", "matter"], "within": "the central Archive",
      "bulk_class": "moderate", "integrity": "sound", "authority": "the Office of Vigil",
      "access": { "who": "anyone" },
      "asserts": [ { "claim": "forty-one volumes with missing nights, across six districts, all within the last nine years", "accurate": true } ] },
    { "name": "the Registry of Solitaries", "facets": ["record", "matter"], "within": "the central Archive",
      "bulk_class": "slight", "integrity": "sound", "authority": "the Office of Vigil",
      "access": { "who": "the Office of Vigil" },
      "asserts": [ { "claim": "eleven names; fourteen applications to leave, none granted", "accurate": true } ] },

    { "name": "the Office of Vigil", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "that nobody audits the transcriptions",
      "vulnerability": "every volume rests on one person, replaceable but not auditable",
      "pursuing": [ { "horizon": "long_standing", "toward": "keep the volumes unaudited" } ] },
    { "name": "the Registry", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "keep the number at eleven",
      "vulnerability": "there are eleven, they are named, and none wants to be there",
      "pursuing": [ { "horizon": "long_standing", "toward": "grant no discharge" } ] },
    { "name": "the Council of Orbe", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "to govern", "vulnerability": "any debate on the dream would itself be transcribed",
      "pursuing": [ { "horizon": "long_standing", "toward": "legislate on anything but the dream" } ] },

    { "name": "the sleepers of the Doce", "facets": ["magnitude", "agency", "holding"], "within": "Barrio Doce",
      "magnitude_class": "many", "capacity_class": "vast",
      "holds": [ { "substance": "sleep", "abundance": "adequate" } ],
      "seen_as": "forty thousand people who all saw the same thing last night",
      "pursuing": [ { "horizon": "imminent", "toward": "read the volume at nine and say nothing" } ] },
    { "name": "the Insomniacs", "facets": ["magnitude", "agency"], "within": "Orbe",
      "magnitude_class": "many",
      "seen_as": "official figure three hundred and forty; the real one is thought to be three thousand or more",
      "pursuing": [ { "horizon": "long_standing", "toward": "not to be seen" } ] },

    { "name": "Rem Salas", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "the Doce's transcriber, forty-four, twelve years in the chair",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "moderate" },
      "disposition": [ { "trait": "scrupulous", "strength": "defining", "manner": "checks a line three times before it goes in" } ],
      "doing": "deciding whether last night's scene goes in the volume as she saw it",
      "pursuing": [ { "horizon": "long_standing", "toward": "never to be wrong" } ],
      "hiding": "She has omitted material from eleven volumes, always to protect someone, and keeps the list at home." },
    { "name": "Onel", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "solitary number four, thirty-eight, nineteen years registered",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "moderate" },
      "conditions": ["solitary"],
      "disposition": [ { "trait": "withheld", "strength": "defining", "manner": "answers a narrower question than the one asked" } ],
      "doing": "writing a fifteenth application to be struck from the register",
      "pursuing": [ { "horizon": "long_standing", "toward": "the discharge" } ],
      "hiding": "He sees far more than he declares and can follow one named person all night. The Office does not know." },
    { "name": "Vira Cor", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "worn",
      "seen_as": "twenty-six, eleven nights without sleep, and people are standing further away",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "conditions": ["projecting"],
      "doing": "staying awake a twelfth night",
      "pursuing": [ { "horizon": "imminent", "toward": "never to sleep again" } ],
      "hiding": "What she does not want the district to see — and the district has begun to see it anyway." },
    { "name": "Inspector Bald", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "fifty-one, Office of Vigil, another-dream section",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "moderate" },
      "doing": "building the case that closes the Tarma",
      "pursuing": [ { "horizon": "imminent", "toward": "close the Tarma boarding house" } ],
      "hiding": "He sleeps in the Doce. His card says another district. Four years in breach of the law he enforces." },
    { "name": "Nea Salas", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "seventeen, Rem's daughter, always halfway out the door",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "moderate" },
      "pursuing": [ { "horizon": "long_standing", "toward": "leave Orbe" } ],
      "hiding": "She has slept three times in the Twenty-Nine and knows its dream does not fully break: it fragments, and the fragments are from other nights." },
    { "name": "Archivista Mayor Ossen", "facets": ["matter", "agency"], "within": "the central Archive",
      "bulk_class": "moderate", "integrity": "worn",
      "seen_as": "sixty-three, counting the months",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "moderate" },
      "pursuing": [ { "horizon": "imminent", "toward": "retire before anyone asks him a question" } ],
      "hiding": "He knows exactly how many volumes have missing nights: forty-one, six districts, nine years." },
    { "name": "Solitary number one", "facets": ["matter", "agency"], "within": "Orbe",
      "bulk_class": "moderate", "integrity": "worn",
      "seen_as": "seventy-one, no public name, registered since she was twelve",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "slight" },
      "conditions": ["solitary"],
      "pursuing": [ { "horizon": "long_standing", "toward": "outlive the reform that replaced her method" } ],
      "hiding": "She is the only person left who knows how solitaries were selected before the reform." },
    { "name": "the arriving stranger", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "someone the market is being careful not to look at",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "moderate" },
      "pursuing": [ { "horizon": "imminent", "toward": "read the volume before the district does" } ] }
  ],

  "offices": [
    { "name": "Transcriber of the Doce", "held_by": "Rem Salas", "of": "the Office of Vigil",
      "confers": [ { "act": "deciding what enters the volume" }, { "act": "closing the volume at noon" } ],
      "succeeds_by": "appointment by the Office of Vigil" },
    { "name": "Solitary number four", "held_by": "Onel", "of": "the Registry",
      "confers": [ { "act": "testifying at trial as full proof" } ],
      "succeeds_by": "no application to leave has ever been granted" },
    { "name": "Inspector of another-dream", "held_by": "Inspector Bald", "of": "the Office of Vigil",
      "confers": [ { "act": "opening an another-dream file" }, { "act": "demanding a district card" } ],
      "succeeds_by": "appointment" }
  ],

  "standing": [
    { "from": "the sleepers of the Doce", "toward": "the arriving stranger",
      "stance": "everyone saw him beside the body and nobody will say so",
      "carried_by": "the market look", "persistence": "permanent" },
    { "from": "the Registry", "toward": "Onel", "stance": "useful and not releasable",
      "carried_by": null, "persistence": "until changed" },
    { "from": "the sleepers of the Doce", "toward": "Vira Cor",
      "stance": "standing further away each day", "carried_by": "the market look", "persistence": "until changed" }
  ],

  "opposition": [
    { "between": ["Rem Salas", "the arriving stranger"],
      "incompatible": "She cannot both protect whoever she protected last night and hand him a volume he can check against his own memory.",
      "stakes": "Her eleven omissions become the second story." },
    { "between": ["Inspector Bald", "the Tarma boarding house"],
      "incompatible": "He cannot close the house without the guest book being read.",
      "stakes": "His own name is in it." },
    { "between": ["Onel", "the Registry"],
      "incompatible": "He cannot be discharged while his testimony is full proof.",
      "stakes": "Fourteen refusals and a fifteenth being written." }
  ],

  "processes": [
    { "name": "the district works it out", "acts_on": "standing toward the arriving stranger",
      "direction": "harden", "rate_class": "short", "terminus": "the volume opens" }
  ],

  "cycles": [
    { "name": "the night of the Doce", "period_class": "short", "starts_in_phase": "the reading room",
      "phases": [
        { "name": "the dreaming", "changes": [ { "entity": "the dream of the Doce", "becomes": "open" } ] },
        { "name": "the bell", "changes": [ { "entity": "the dream of the Doce", "becomes": "ended" } ] },
        { "name": "the market", "changes": [ { "entity": "Barrio Doce", "becomes": "tense" } ] },
        { "name": "the reading room", "changes": [ { "entity": "the volume of last night", "becomes": "public" } ] } ] }
  ],

  "accumulators": [
    { "name": "the district asleep", "per": "each entity with extent",
      "starts_at": "none", "stated": "How much of the district has fallen asleep tonight.",
      "raised_by": [ { "event": "a person falls asleep in the district" } ],
      "thresholds": [ { "at": "low", "then": "the dream of the district opens" },
                      { "at": "high", "then": "late sleepers arrive where the people already are" } ] },
    { "name": "nights unslept", "per": "each entity with agency", "starts_at": "none",
      "stated": "How long since this person last slept.",
      "raised_by": [ { "event": "a night passed awake" } ],
      "thresholds": [ { "at": "low", "then": "condition:projecting — brief daytime projections" },
                      { "at": "moderate", "then": "frequent projections, visible across a street" },
                      { "at": "high", "then": "continuous projection: the district sees what the person thinks" },
                      { "at": "extreme", "then": "collapse", "irreversible": true } ] }
  ],

  "indicators": [
    { "of": "nights unslept",
      "shows_as": ["a shape at the edge of sight that is not there", "people standing further off in the market"],
      "read_by": { "channel": "sight", "requires": null }, "reliability_class": "good" },
    { "of": "the district asleep",
      "shows_as": ["lamps going out along a street", "the reading room emptying"],
      "read_by": { "channel": "sight", "requires": null }, "reliability_class": "moderate" }
  ],

  "traces": [
    { "of": "a night dreamt", "leaves": "a volume in the Archive", "ages": "never" },
    { "of": "sleeping in the wrong district", "leaves": "a line in a boarding-house guest book", "ages": "never" },
    { "of": "a night in the Twenty-Nine", "leaves": "nothing in any register", "ages": "never" }
  ],

  "epochs": [
    { "name": "before the reform", "differed": [
        { "topic": "institution", "subject": "the Registry", "then": "solitaries were selected by another method" } ],
      "surviving_traces": ["Solitary number one, who is still alive"] }
  ],

  "history": [
    { "name": "the-killing-in-the-dream", "standing": "disputed",
      "what_happened": "The district dreamt a killing in a place that is not Orbe, with a dead man nobody recognised, and the arriving stranger standing beside the body.",
      "where": "the dream of the Doce", "who": ["the sleepers of the Doce", "Onel", "Rem Salas", "the arriving stranger"],
      "knowledge": [
        { "holder": "the sleepers of the Doce", "channel": "the dream", "path": "direct",
          "believes": "He stood over the body and looked at it, so he had something to do with it.",
          "accurate": false, "plausible_because": "Forty thousand people watched him not move." },
        { "holder": "the arriving stranger", "channel": "the dream", "path": "direct",
          "believes": "He was there and did nothing, and cannot account for the place or the man." },
        { "holder": "Onel", "channel": "the dream", "path": "direct",
          "believes": "Something he has not declared, because he was following one person all night.",
          "accurate": true },
        { "holder": "Rem Salas", "channel": "the dream", "path": "direct",
          "believes": "What she wrote, which is not everything she saw." } ] },
    { "name": "the-eleven-omissions", "standing": "occurred",
      "what_happened": "Over twelve years a transcriber left material out of eleven volumes, each time to protect someone.",
      "where": "the Office of the Doce", "who": ["Rem Salas"],
      "knowledge": [
        { "holder": "Rem Salas", "channel": "sight", "path": "direct", "believes": "Every one of them, and why." },
        { "holder": "Archivista Mayor Ossen", "channel": "the volume", "path": "inference",
          "believes": "Forty-one volumes have missing nights and he cannot say whose fault each is.", "accurate": true } ] }
  ],

  "arrivals": [
    { "premise": "Thirty-one, Barrio Doce, no record, ordinary work, never once relevant in a volume. Last night the district dreamt a killing and forty thousand people saw you standing beside the body. You remember being there. You do not remember doing anything. The volume opens at nine.",
      "seen_as": "someone the market is being careful not to look at",
      "within": "Barrio Doce",
      "capability": { "moves_by": ["walk", "falling asleep"], "carry_class": "moderate" } }
  ]
}
```

## 2. Validity self-check

**O1** pass — nine extents; two passages. **O2** pass. **O3** pass — every `agency` entity carries `pursuing`, including the three collectives and both `magnitude` entities. **O4** pass. **O5** pass — three. **O6** pass — every `matter` entity has `bulk_class`, including all nine people. **O7** pass — `sleep` is demanded by the dream and held by `the sleepers of the Doce`. **O8** pass. **O9** pass — both indicators name an accumulator I declared; I deleted my v2 "what a transcriber left out" indicator, which named no held state. **O10** pass. **O11** pass.

**R1** pass — every reference resolves; I promoted the player to `the arriving stranger` because `standing`, `opposition` and `history.who` all name it. **R2** pass. **R3** pass — no number in an engine field; `decay: "brief"` replaces v2's `"three nights"`. **R4** pass. **R5** — *pass under my reading, and this is contestable*: `the sleepers of the Doce` appears in `standing.from`, `opposition` and `history.who` as a body, never as one person. **R6** pass. **R7** pass — I dropped `legibility` from the Insomniacs (a `collective` key on a non-collective) and `holds` from every non-`holding` entity. **R8** pass — `within` is a tree rooted at Orbe. **R9** pass. **R10** pass. **R11** pass — the hysteresis ladder is gone; opening is a threshold, closing is the dream's unmet `demand`. **R12** pass — the sleepers and the stranger disagree on the same event, one flagged inaccurate.

## 3. Audit of my v2 document

Six refusals fire. Four are right; two reject a document that should be legal.

**Correct catches.** *R1* — I wrote `standing` toward "the arriving stranger" with no such entity; a real dangling reference. *R7* — `legibility` on the Insomniacs, who had `magnitude + agency` and no `collective`. *R11* — my hysteresis ladder (`low` → open, `very low` → close) was descending inside an ascending list; genuinely malformed. *R10* — three collectives and two magnitudes had no `pursuing`.

**R10 is half-wrong.** All five entities *did* state a goal — in `interest`, the `collective` key. The contract has two keys for the same thing and an obligation that names only one, so a document that says what an institution wants is rejected for saying it in the other field. Either `interest` satisfies O3 for collectives, or delete `interest`.

**O6/R7 on people is wrong.** Requiring `bulk_class` on every named human made me write `"moderate"` nine times, which is noise carrying no information and inviting a builder to treat a body as cargo. O6's own justification — "nothing can be lifted, blocked, or contained" — is an argument about objects. Recommend: `bulk_class` required on `matter` entities that are not also `agency`, defaulted otherwise.

**R3 is wrong here, and it is the sharpest finding.** v2 had `channel.decay: "three nights"`. That is a number in an engine-computed field, so R3 rejects it — but *"Memoria propia: tres noches. Al cuarto día se borra"* is hard rule 3, stated exactly, and the whole plot turns on it: after three nights only the volume remains and someone else wrote it. v3 forces `"brief"`, and a builder resolving `brief` to two nights or five has broken the world. The class-not-number rule is right for quantities the world invents and wrong for a quantity the *author states as law*. Recommend a narrow exception: a figure named inside a `law[].stated` may be cited verbatim by a field that implements that law.

## 4. Residual ambiguity

1. **The dream is `within` Barrio Doce, or `within` Orbe as its sibling?** D2 says a concurrent reality is an entity with `extent`; it does not say where it hangs. I chose the district. Both are trees, and the choice changes distance, reach and what "one extent" means for `sight`.
2. **`within` is a single mutable position — so a sleeper's body is nowhere.** Deleting `layers[]` moved the problem into D1 rather than solving it. Under v3 falling asleep *moves you* into the dream, so while you are there your body has no location. Bald's entire offence is that his body sleeps in the Doce while his card says elsewhere, and that fact is unstatable during the only time it is true. This is the one place where v3 is strictly less able than a layered model.
3. **Is `magnitude` in `opposition.between` or `standing.from` "referencing it individually"?** R5's wording does not say. I read collective reference as legal; a stricter encoder would forbid it and be unable to say the district holds an opinion.
4. **Does `accumulator.per: "each entity with extent"` include the dream itself?** It has `extent`. Then the dream has a sleep-count, which is nonsense. Scope needs to name facets *and* exclusions.
5. **`falling asleep` as a `movement` with `pace_class`.** A pace resolves to a speed, so entering a dream acquires a duration. Legal, and meaningless.
6. **Collective `interest` vs agency `pursuing`** — see §3; two keys, one meaning.
7. **Is an arrival also an entity?** `standing`, `opposition` and `history.who` all needed to name the
   player, and R1 refuses dangling references — so I authored `the arriving stranger` as an entity
   *and* as an `arrivals` premise. Nothing says whether that is duplication or the required pattern.

## 5. Three under-specified reader obligations

**`channel.conceals: none` → "renders present entities' `pursuing` and `hiding` to all receivers".** This single line carries the world's central mechanic — *"lo que uno desea se manifiesta alrededor suyo"*. But `pursuing` is a goal list and `hiding` is one sentence. One builder prints them as text to every receiver; another stages them as a visible tableau around the person. Same document, two different games.

**`indicator.reliability_class` → "how often the sign misreports".** No ladder maps a class to a rate. Whether `good` means the district can usually tell an insomniac at eleven nights, or usually cannot, decides whether Vira Cor's arc happens at all.

**`demand.unmet` → "apply the effect after `onset_class`, and go on applying it".** The dream's unmet effect is that it ends. A terminating effect cannot "go on applying". One builder ends it once and reopens on the next threshold crossing; another re-applies termination every tick and the dream can never reopen. The obligation assumes every unmet demand is a continuous harm.
