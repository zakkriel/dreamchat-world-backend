# T2 — Los Andantes in `world_model/2`

## 1. Encoding — detailed tier

```jsonc
{
  "world_model": "2",
  "world": { "name": "Los Andantes", "premise": "There are no continents. The habitable surface of the world is the backs of nine living creatures that walk a shallow ocean on fixed migratory routes. A sick Andante is a city with an expiry date.",
             "mood": "clinical, administrative; the catastrophe is slow and predictable, which is what makes it political" },

  "excluded": [
    "No treatment, cure or intervention reverses an Andante's illness.",
    "The Andantes do not communicate, hold no intention toward their inhabitants, and do not respond to petition, ritual or offering.",
    "No route can be altered by any means.",
    "There is no hidden land, lost continent, tenth Andante, or young Andante growing.",
    "A dead Andante sinks in five to seven days. No case extends it.",
    "Auscultation does not predict exactly; the table 5.3 intervals rest on one documented case."
  ],

  "layers": [ { "name": "the world", "default": true } ],

  "vocabulary": {
    "media": [ { "name": "open sea", "resists": [ { "to": "walk", "degree": "total" } ] },
               { "name": "dorsal air", "resists": [] } ],
    "movements": [ { "name": "walk", "pace_class": "steady" },
                   { "name": "barge crossing", "pace_class": "slow" } ],
    "channels": [
      { "name": "sight" }, { "name": "speech" },
      { "name": "the trumpet", "descriptor": "direct listening on the dorsum through a bronze horn",
        "emitted_by": "the Andante's body", "received_by": "an entity holding a trumpet",
        "latency_class": "immediate", "reach": "the point of contact", "decay": "never", "conceals": "none" },
      { "name": "the monthly bulletin", "descriptor": "the published state of the Andante",
        "emitted_by": "the office of Auscultador Mayor", "received_by": "anyone in the city",
        "latency_class": "slow", "reach": "one city", "decay": "never", "conceals": "none" } ],
    "conditions": [
      { "name": "overloaded", "alters": [ { "movement": "walk", "effect": "hinder", "class": "moderate" } ] },
      { "name": "lamed", "alters": [ { "movement": "walk", "effect": "hinder", "class": "severe" } ] },
      { "name": "systemically ill", "alters": [ { "movement": "walk", "effect": "hinder", "class": "total" } ] } ],
    "substances": [ { "name": "load" }, { "name": "passage rights" } ]
  },

  "law": [
    { "name": "undirigibility", "enforced_by": "physics",
      "stated": "No method alters an Andante's route or pace.",
      "forbids": { "subject": "any entity", "act": "changing a trajectory" } },
    { "name": "the weight limit", "enforced_by": "physics",
      "stated": "Each Andante bears a tabulated maximum. Over it: slowing, then lameness, then joint injury, then systemic illness." },
    { "name": "transit only at Convergence", "enforced_by": "physics",
      "stated": "Passage between Andantes is safe only during a Convergence. Outside it, by barge, survival is under thirty per cent." },
    { "name": "sinking", "enforced_by": "physics",
      "stated": "A dead Andante sinks in five to seven days.",
      "forbids": { "subject": "any entity", "act": "keeping a dead Andante afloat" } },
    { "name": "the forbidden zones", "enforced_by": "office", "binds": [],
      "stated": "No construction on the head, within four hundred metres of a blowhole, or on a major joint.",
      "precedent": null },
    { "name": "no work without licence", "enforced_by": "office", "binds": [],
      "stated": "Every work requires prior licence from the Weight Guild." },
    { "name": "no Andante meat", "enforced_by": "persons", "binds": [],
      "stated": "Nobody eats Andante. Unwritten, universally kept, origin undocumented." },
    { "name": "auscultators do not sit on the Council", "enforced_by": "persons", "binds": [],
      "stated": "Custom. In practice they direct it." }
  ],

  "entities": [
    { "name": "the ocean", "facets": ["extent"], "extent_class": "vast", "medium": "open sea",
      "seen_as": "thirty to sixty metres of water over firm bottom, no coast in any direction" },

    { "name": "Tercera Hembra", "facets": ["extent", "matter", "motion", "demand"],
      "within": "the ocean", "extent_class": "vast", "medium": "dorsal air", "tension": "calm",
      "seen_as": "seventy-four kilometres of living back, sixty to two hundred metres above the water",
      "bulk_class": "immense", "integrity": "worn",
      "trajectory": { "period_class": "generational", "phase_at_start": "outbound" },
      "demands": [ { "of": "load below the limit", "rate_class": "continuous",
                     "unmet": { "effect": "condition:overloaded", "onset_class": "very slow" } } ] },

    { "name": "Primera", "facets": ["extent", "matter", "motion"], "within": "the ocean",
      "extent_class": "vast", "medium": "dorsal air", "bulk_class": "immense", "integrity": "sound",
      "trajectory": { "period_class": "generational", "phase_at_start": "outbound" },
      "seen_as": "eighty-eight kilometres, three cities, one hundred and forty thousand people" },

    { "name": "Quinta", "facets": ["extent", "matter", "motion"], "within": "the ocean",
      "extent_class": "vast", "bulk_class": "immense", "integrity": "worn",
      "seen_as": "seventy-nine kilometres, four cities, two hundred and ten thousand people, at ninety-four per cent of its weight limit",
      "trajectory": { "period_class": "generational", "phase_at_start": "outbound" } },

    { "name": "Sexto", "facets": ["extent", "matter", "motion"], "within": "the ocean",
      "extent_class": "large", "bulk_class": "immense", "integrity": "worn",
      "seen_as": "forty-four kilometres, convalescent since 638, bulletin unchanged for three years",
      "trajectory": { "period_class": "generational", "phase_at_start": "outbound" } },

    { "name": "Octavo", "facets": ["matter"], "bulk_class": "immense", "integrity": "destroyed",
      "seen_as": "nothing; it sank in 611 in six days with the city of Ruma on it" },
      // the other four Andantes (Segundo, Cuarto, Séptima, Novena) take the Primera shape exactly.

    { "name": "Ossa", "facets": ["extent"], "within": "Tercera Hembra", "extent_class": "large",
      "medium": "dorsal air", "tension": "normal",
      "seen_as": "thirty-one thousand people on the forward third, founded 402" },
    { "name": "Belna", "facets": ["extent"], "within": "Tercera Hembra", "extent_class": "medium",
      "seen_as": "seventeen thousand on the rear third, forty kilometres of dorsal road away" },

    { "name": "Alto Omóplato", "facets": ["extent"], "within": "Ossa", "extent_class": "medium",
      "seen_as": "the left shoulder blade: College, Guild, Council" },
    { "name": "Cuenca", "facets": ["extent"], "within": "Ossa", "extent_class": "medium",
      "tension": "tense", "seen_as": "the lumbar depression, fourteen thousand people, forty-one thousand three hundred tonnes, over its sector limit since 639" },
    { "name": "Los Espiráculos", "facets": ["extent"], "within": "Ossa", "extent_class": "medium",
      "seen_as": "mid-back, a four-hundred-metre exclusion around the blowholes, nobody lives here" },
    { "name": "Cola Baja", "facets": ["extent"], "within": "Ossa", "extent_class": "medium",
      "seen_as": "the rear third, poor, the ground never still" },

    { "name": "the Convergence road", "facets": ["passage"],
      "connects": ["Tercera Hembra", "Primera"],
      "admits": [ { "condition": "a Convergence in progress" } ],
      "obstructs": [ { "movement": "walk" } ], "hazard_class": "none" },
    { "name": "the barge crossing", "facets": ["passage"],
      "connects": ["Tercera Hembra", "Primera"],
      "admits": [ { "movement": "barge crossing" } ], "hazard_class": "extreme" },

    { "name": "the Auscultation Room", "facets": ["extent", "holding"], "within": "Alto Omóplato",
      "extent_class": "small", "capacity_class": "small",
      "seen_as": "four stations and a locked annexe" },
    { "name": "the Bell of March", "facets": ["matter"], "within": "Alto Omóplato",
      "bulk_class": "moderate", "integrity": "sound",
      "seen_as": "rung at dawn; it measures the day's displacement and the figure is public" },
    { "name": "the Barge Dock", "facets": ["extent", "holding"], "within": "Cola Baja",
      "capacity_class": "moderate", "seen_as": "fourteen double-hulled units on the right flank" },

    { "name": "a bronze trumpet", "facets": ["matter"], "bulk_class": "slight", "integrity": "sound",
      "confer": [ { "channel": "the trumpet" } ],
      "seen_as": "sixty centimetres of bronze, handed over at qualification" },

    { "name": "the monthly bulletin", "facets": ["record"],
      "authority": "the office of Auscultador Mayor of Ossa",
      "access": { "who": "anyone", "when": "the first of the month", "where": "Ossa" },
      "asserts": [ { "claim": "Tercera Hembra: under observation, no findings.", "accurate": false,
                     "since": "eight consecutive months" } ] },
    { "name": "the College archive", "facets": ["record", "holding"],
      "authority": "the College of Auscultators",
      "access": { "who": "holder of the office of Auscultador Mayor", "requires": { "office": "Auscultador Mayor of Ossa" } },
      "asserts": [ { "claim": "the raw readings of the last nine months, including the arrhythmia", "accurate": true } ] },
    { "name": "the Weight Register", "facets": ["record"], "within": "Alto Omóplato",
      "authority": "the Weight Guild",
      "access": { "who": "anyone on written request", "when": "three days later" },
      "asserts": [ { "claim": "load by sector; Cuenca over limit since 639; fourteen works licensed above it between 637 and 640", "accurate": true } ] },
    { "name": "the Convergence Tables", "facets": ["record"],
      "authority": "the College of Auscultators", "access": { "who": "anyone" },
      "asserts": [ { "claim": "Quinta–Séptima 641, 40 days, in progress; Primera–Novena 643, 22 days; Tercera Hembra–Primera 645, 31 days; Segundo–Cuarto 646, 18 days; Tercera Hembra–Séptima 659, 12 days. Calendar runs to the year 900.", "accurate": true } ] },
    { "name": "the Octavo file", "facets": ["record"],
      "authority": "the College of Auscultators",
      "access": { "who": "nobody", "stated_reason": "prevention of alarm" },
      "asserts": [ { "claim": "the cause of the Octavo's death", "accurate": true } ] },
    { "name": "Illa's parallel register", "facets": ["record", "matter"], "bulk_class": "slight",
      "authority": "Illa", "access": { "who": "Illa" },
      "asserts": [ { "claim": "an independent second reading matching the apprentice's", "accurate": true } ] },

    { "name": "the College of Auscultators", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "exclusive authority over the Andante's stated condition",
      "vulnerability": "a falsified bulletin is not verifiable from outside the College" },
    { "name": "the Weight Guild", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "licence revenue and advancement",
      "vulnerability": "licences above the limit on at least three Andantes" },
    { "name": "the Convergents", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "long and lucrative Convergences",
      "vulnerability": "their power exists only in meeting years" },
    { "name": "the Dead Fleet", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "monopoly on emergency transit",
      "vulnerability": "seventy per cent losses and fourteen hulls" },
    { "name": "the Council of Ossa", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "continuity and order", "vulnerability": "de facto subordinate to the College" },
    { "name": "the College of Old Weight", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "none; a residual institution", "vulnerability": "it still holds the Octavo's register" },

    { "name": "Del Vas", "facets": ["matter", "agency"], "within": "Alto Omóplato",
      "seen_as": "the Chief Auscultator of Ossa, fifty-eight, unhurried",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "the trumpet": "acute" },
      "disposition": [ { "trait": "guarded", "strength": "defining", "manner": "answers the question that was asked and no other" } ],
      "doing": "signing the ninth consecutive bulletin that says no findings",
      "pursuing": [ { "horizon": "long_standing", "toward": "retire in 643 with no collapse on her record" } ],
      "hiding": "She detected the arrhythmia nine months ago and did not record it." },
    { "name": "Registrador Onn", "facets": ["matter", "agency"], "within": "Alto Omóplato",
      "seen_as": "a Weight Guild registrar of forty-four with a good coat",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "disposition": [ { "trait": "accommodating", "strength": "strong", "manner": "finds the reading that suits the applicant" } ],
      "doing": "preparing a transfer request to Quinta",
      "pursuing": [ { "horizon": "long_standing", "toward": "revenue, then transfer to Quinta" } ],
      "hiding": "He licensed fourteen works above the limit in Cuenca between 637 and 640." },
    { "name": "Bara Quel", "facets": ["matter", "agency"], "within": "Alto Omóplato",
      "seen_as": "a Convergent of thirty-six who has been waiting eighteen years for a working year",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "doing": "selling passage rights for 645",
      "pursuing": [ { "horizon": "imminent", "toward": "that the 645 Convergence happens and is his" } ],
      "hiding": "He has sold three thousand passage rights. Primera admits twelve thousand in total, Belna included." },
    { "name": "Sento", "facets": ["matter", "agency"], "within": "Cola Baja",
      "seen_as": "a bargeman of forty who does not talk about the crossings",
      "capability": { "moves_by": ["walk", "barge crossing"], "carry_class": "moderate" },
      "doing": "offering a crossing to Primera outside Convergence",
      "hiding": "On two of his three crossings he was the only survivor of his barge." },
    { "name": "Consejera Uma Ret", "facets": ["matter", "agency"], "within": "Alto Omóplato",
      "seen_as": "a councillor of sixty-one who has stopped sleeping well",
      "doing": "not tabling the paper in her drawer",
      "pursuing": [ { "horizon": "imminent", "toward": "avoid panic" } ],
      "hiding": "She has a finished triage plan she has shown to nobody." },
    { "name": "Illa", "facets": ["matter", "agency"], "within": "Alto Omóplato",
      "seen_as": "a first-year apprentice of nineteen who writes everything down",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "the trumpet": "normal" },
      "doing": "keeping a register nobody asked her to keep",
      "pursuing": [ { "horizon": "long_standing", "toward": "qualify" } ],
      "hiding": "She repeated the auscultation, got the same result, and did not declare it." },
    { "name": "Auscultador Mayor Renn", "facets": ["matter", "agency"], "within": "Belna",
      "seen_as": "Belna's chief auscultator, forty-nine",
      "doing": "publishing a bulletin that has not matched Ossa's for four months",
      "pursuing": [ { "horizon": "long_standing", "toward": "diagnostic independence for Belna" } ] },
    { "name": "Vieja Marda", "facets": ["matter", "agency"], "within": "Cola Baja",
      "seen_as": "a woman of eighty-one who says very little",
      "disposition": [ { "trait": "withheld", "strength": "defining", "manner": "lets the question sit" } ],
      "hiding": "She was fifty-one in Ruma and is the only living source on 611." },

    { "name": "the people of Ossa", "facets": ["magnitude", "agency"], "within": "Ossa",
      "magnitude_class": "many", "seen_as": "thirty-one thousand people who read the bulletin on the first" },
    { "name": "the people of Cuenca", "facets": ["magnitude", "agency"], "within": "Cuenca",
      "magnitude_class": "many", "seen_as": "fourteen thousand living over the sector limit" }
  ],

  "offices": [
    { "name": "Auscultador Mayor of Ossa", "held_by": "Del Vas", "of": "the College of Auscultators",
      "confers": [ { "act": "signing the monthly bulletin" },
                   { "act": "opening the College archive" } ],
      "succeeds_by": "designation by the College" },
    { "name": "Auscultador Mayor of Belna", "held_by": "Auscultador Mayor Renn",
      "of": "the College of Auscultators", "confers": [ { "act": "signing Belna's bulletin" } ],
      "succeeds_by": "designation by the College" },
    { "name": "Registrar of Weight", "held_by": "Registrador Onn", "of": "the Weight Guild",
      "confers": [ { "act": "granting a works licence" } ], "succeeds_by": "appointment" },
    { "name": "Auscultator", "held_by": null, "of": "the College of Auscultators",
      "confers": [ { "act": "recording an official reading" }, { "channel": "the trumpet" } ],
      "succeeds_by": "four years apprentice, three years second, then examination" }
  ],

  "standing": [
    { "from": "the Council of Ossa", "toward": "the Council of Belna",
      "stance": "institutionally cold", "carried_by": null, "persistence": "until changed" },
    { "from": "Del Vas", "toward": "Illa", "stance": "watching her more closely than she knows",
      "carried_by": "sight", "persistence": "until changed" },
    { "from": "the College of Auscultators", "toward": "the College of Auscultators",
      "stance": "no College audits another College", "persistence": "permanent" }
  ],

  "opposition": [
    { "between": ["Del Vas", "Illa"],
      "incompatible": "The arrhythmia cannot both stay off the record and be independently registered.",
      "stakes": "Whoever declares first owns the recommendation that decides the triage." },
    { "between": ["Bara Quel", "Primera"],
      "incompatible": "Three thousand sold rights cannot all be honoured inside a total capacity of twelve thousand that already includes Belna.",
      "stakes": "The Convergence he has waited eighteen years for is also the evacuation window." },
    { "between": ["Consejera Uma Ret", "the people of Ossa"],
      "incompatible": "Order for four years and preparation time for four years are the same four years.",
      "stakes": "Forty-eight thousand people and one window." }
  ],

  "processes": [
    { "name": "the march slows", "acts_on": "Tercera Hembra", "direction": "degrade",
      "rate_class": "very slow", "terminus": "the march stops" },
    { "name": "Cuenca keeps building", "acts_on": "the load on Tercera Hembra", "direction": "raise",
      "rate_class": "slow", "terminus": "the Guild refuses a licence" }
  ],

  "cycles": [
    { "name": "the migratory route of Tercera Hembra", "period_class": "generational",
      "starts_in_phase": "outbound",
      "phases": [ { "name": "outbound", "changes": [] },
                  { "name": "at the crossing of Primera", "changes": [
                      { "entity": "the Convergence road", "becomes": "passable" } ] },
                  { "name": "return", "changes": [] } ] },
    { "name": "the bulletin month", "period_class": "short", "starts_in_phase": "between",
      "phases": [ { "name": "the first", "changes": [
                      { "entity": "the monthly bulletin", "becomes": "reissued" } ] },
                  { "name": "between", "changes": [] } ] }
  ],

  "accumulators": [
    { "name": "load on Tercera Hembra", "per": "each entity with motion", "starts_at": "high",
      "stated": "How much is built and carried on the back against what the animal bears.",
      "raised_by": [ { "aggregate_of": "bulk_class", "over": "everything within it" },
                     { "event": "a works licence granted" } ],
      "thresholds": [ { "at": "high", "then": "condition:overloaded — the march slows" },
                      { "at": "very high", "then": "condition:lamed" },
                      { "at": "extreme", "then": "joint injury" },
                      { "at": "terminal", "then": "condition:systemically ill", "irreversible": true } ] },
    { "name": "the illness of Tercera Hembra", "per": "each entity with motion", "starts_at": "moderate",
      "stated": "How far the animal has gone toward dying.",
      "raised_by": [ { "event": "time under condition:overloaded" } ],
      "thresholds": [ { "at": "moderate", "then": "alert: four to ten years remain" },
                      { "at": "high", "then": "critical: fourteen to thirty months remain" },
                      { "at": "extreme", "then": "terminal: under ninety days", "irreversible": true },
                      { "at": "terminal", "then": "the march stops: under twenty days", "irreversible": true } ] }
  ],

  "indicators": [
    { "of": "the illness of Tercera Hembra",
      "shows_as": ["deep pulse irregular, then arrhythmic, then inaudible",
                   "march falling from three to five kilometres a day toward none",
                   "blowhole discharge clear, then dense, then obstructed",
                   "dorsal temperature up three degrees, then six, then falling sharply",
                   "response to stimulus from four hours out past forty-eight, then none",
                   "flank vibration intermittent, then continuous"],
      "read_by": { "channel": "the trumpet", "requires": { "office": "Auscultator" } },
      "reliability_class": "poor" },
    { "of": "load on Tercera Hembra",
      "shows_as": ["the Bell of March reading a shorter day than last year"],
      "read_by": { "channel": "sight", "requires": null },
      "reliability_class": "moderate" }
  ],

  "traces": [
    { "of": "a works licence granted", "leaves": "an entry in the Weight Register", "ages": "never" },
    { "of": "an auscultation", "leaves": "a raw sheet in the College archive", "ages": "never" },
    { "of": "an Andante sinking", "leaves": "nothing at all", "ages": "never" }
  ],

  "epochs": [
    { "name": "before the Octavo", "differed": [
        { "topic": "institution", "subject": "the College of Auscultators", "then": "advisory, not authoritative" } ],
      "surviving_traces": ["the whole institutional architecture built after 611", "the classified file"] } ],

  "history": [
    { "name": "the-Octavo", "standing": "disputed",
      "what_happened": "An Andante died of an unestablished cause and sank in six days. Ruma held sixty thousand people; three thousand one hundred were evacuated.",
      "where": "the ocean", "who": ["Vieja Marda"],
      "knowledge": [
        { "holder": "Vieja Marda", "channel": "sight", "path": "direct",
          "believes": "What she saw at fifty-one, which she does not repeat." },
        { "holder": "the people of Ossa", "channel": "speech", "path": "told",
          "believes": "It was sudden and nobody could have known.", "accurate": false,
          "plausible_because": "The file has been classified since 611 for the prevention of alarm." } ] },
    { "name": "the-Cuenca-licences", "standing": "occurred",
      "what_happened": "Fourteen works were licensed above the sector limit in Cuenca between 637 and 640; the arrhythmia appears in 640.",
      "where": "Cuenca", "who": ["Registrador Onn"],
      "knowledge": [
        { "holder": "Registrador Onn", "channel": "sight", "path": "direct", "believes": "He signed them and the register says so." },
        { "holder": "Del Vas", "channel": "sight", "path": "inference",
          "believes": "The dates line up and she has not written that down either." } ] },
    { "name": "the-reading-of-the-fourth", "standing": "occurred",
      "what_happened": "A routine auscultation returned a sustained arrhythmia — critical — with alert readings on temperature, march and blowholes.",
      "where": "the Auscultation Room", "who": ["Illa"],
      "knowledge": [
        { "holder": "Illa", "channel": "the trumpet", "path": "direct",
          "believes": "She repeated it separately and got the same thing." } ] }
  ],

  "arrivals": [
    { "premise": "Fourth-year apprentice auscultator in Ossa, qualifying in 643. On the fourth of this month you took a routine reading and it came back critical. The bulletin of the first, signed by Del Vas, says no findings, and so do the eight before it.",
      "seen_as": "an apprentice of twenty-seven carrying a trumpet they have not been given yet",
      "within": "Alto Omóplato",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "the trumpet": "normal" } } ]
}
```

## 2. Facet assignments and where I hesitated

| Entity class | Facets I took | Hesitation |
|---|---|---|
| An Andante | `extent + matter + motion + demand` | **Yes, sharp.** `borne` was the alternative for the cities. I took `within` because the back has sub-locations belonging to no city — but `borne` is what propagates position when the carrier walks, and `within` nowhere says it does. |
| Andantes other than Tercera Hembra | dropped `demand` | Arbitrary: every Andante has a weight limit. I kept it on the focus animal to avoid nine identical blocks — authoring convenience, not a reading. |
| Octavo | `matter`, `integrity: "destroyed"` | **Yes.** It does not exist. No lifecycle state, so I abused `integrity`. Dropping it loses the census row the institutions were built on. |
| A city / district | `extent` | No. |
| Convergence road, barge crossing | `passage` | **Yes.** Is the crossing the fourteen barges, or a route they serve? I split them; another reader would fuse them. |
| A record | `record` (+ `matter` when takeable) | **Yes, everywhere.** No rule says whether a physical book is `record` or `record + matter`. I was inconsistent inside one document. |
| Institutions | `agency + collective` | No. |
| Named people | `matter + agency` | No. |
| Populations | `magnitude + agency` | Mild — followed the schema's own Pactless example. |
| The bronze trumpet | `matter` + `confer` | `confer` is described in prose but is on no facet's key list. I hung it on the entity. |

## 3. Breaks

**B1 — an irregular published schedule. (iii) wrong-shaped.** *"Los cruces de ruta … son predecibles con siglos de anticipación"*; the table lists 641, 643, 645, 646, 659. `cycles[]` carries one `period_class`, so it can say "a Convergence recurs on a generational rhythm" but not *when*. I put the calendar in a `record`'s `asserts` — player-readable and mechanically inert. The one date the whole story turns on (645) is therefore fiction the engine cannot schedule against.

**B2 — racing clocks. (i) inexpressible.** The premise is an arithmetic collision: fourteen-to-thirty months of animal against four years to the window against forty-eight thousand people against twelve thousand berths. `opposition[]` holds only `between` **entities**. There is no way to state an incompatibility between an accumulator's ladder and a scheduled event. This is v1's deadline gap unmoved.

**B3 — capacity as an engine quantity. (iii).** *"Capacidad de acogida de Primera: 12.000 … Ossa y Belna suman 48.000."* v2 permits numbers "in fields a player reads". These two figures are simultaneously player-read (§9 is a table people cite) and engine-computed (does everyone fit?). The new rule gives me no way to decide, and the answer changes whether the triage is adjudicable or narrated.

**B4 — an entity that no longer exists. (iii).** Octavo. No lifecycle; I wrote `integrity: "destroyed"`, which is a matter reading of a fact about existence.

**B5 — a record that lies. (ii) inert prose.** The monthly bulletin says *"en observación, sin hallazgos"* for nine months and this is the central deceit. `record.asserts` has no falsity key; `accurate` exists only under `history[].knowledge`. I invented `asserts[].accurate` because the alternative was losing the plot.

**B6 — an absence of oversight. (ii).** *"Ningún Colegio audita a otro Colegio."* I encoded it as a `standing` from the College to itself — a hack: standing is a directed relation, and this is the *non-existence* of one.

**Fixed by v2, honestly:** `indicators[]` holds the entire auscultation table including *"la base estadística es insuficiente"* as `reliability_class: poor` — v1 had nowhere for a diagnostic system. `accumulators` with ordered ladders hold the overload cascade exactly. `excluded[]` holds Libro X almost verbatim. `offices[]` holds the qualification ladder with `held_by: null` for the unfilled grade. Three real v1 breaks closed.

## 4. Ambiguity report

1. **`within` vs `borne_by` for a city on a walking animal.** Took `within`. Disambiguate by stating whether `within` propagates position when the container has `motion`; if it does, `borne` is only for objects, and that should be said.
2. **Does an entity with `motion` carry its contents?** Unstated anywhere. If not, every city needs `borne` *and* `within`, which the schema does not allow.
3. **Route as `motion.trajectory` vs `cycles[]`.** I wrote both — a trajectory on the animal and a cycle for the phases. They are the same fact twice. One should be illegal.
4. **Accumulator scope wording.** `per: "each entity with motion"` — I invented that phrasing from the facet list. The example only shows `per: "world"`. Whether scope names a facet, a kind, or an entity list is undefined.
5. **Conditions on a non-person.** `overloaded` and `lamed` are conditions of an Andante. `conditions[].alters` names movements, which the Andante has via `motion` not via `capability.moves_by`. Whether a condition can bite on `trajectory` is undefined.
6. **Where does the weight *limit* live?** I made it the accumulator's threshold ladder. It could equally be a `demand` (`of: "load below the limit"`), which is what I also wrote. I encoded it twice and cannot tell which is intended.
7. **`record` + `matter`, or `record` alone?** No rule. I was inconsistent within one document.
8. **`confer` is described in prose but appears in no facet's key list.** I attached it to a `matter` entity; it could belong only to `offices[]`.
9. **Nine near-identical Andantes.** Enumerate all nine, or one plus `magnitude`? I enumerated five and elided four in a comment. A convergence test between two encoders will diverge here for certain.

## 5. Convergence check

**No new top-level section.** `cycles[]` needs an irregular/calendared variant and `opposition[]` needs to accept non-entity operands, but both are field changes inside existing sections. The 17 sections held.

## 6. Basic tier vs detailed — the diff

I encoded tier 1 as well. **Entity-for-entity it is a clean subset**: the same facet sets, fewer entities (four people not eight, four records not seven, no Belna, no Convergence calendar), fewer keys per entity, no `hiding` on anyone but the player's problem statement. Nothing in tier 1 needed a facet tier 3 did not use, and nothing changed shape. On the `entities[]` axis the schema reads detail as depth, correctly.

**Section-for-section it is not a subset, and this is the defect.** Four sections are *empty* in tier 1 and populated in tier 3: `excluded`, `indicators`, `offices`, `epochs`. Three of those are merely thinner worlds. One is not:

**An empty `excluded[]` is not neutral — it is a positive statement.** Tier 1 never says there is no cure, so the tier-1 document permits one; tier 3's Libro X forbids it in six clauses. Two documents encoding the same world differ on whether the central tragedy is a tragedy. Every other section degrades by saying less; `excluded` degrades by saying *the opposite*.

Second-order: without `indicators`, tier 1's Andante has a state and no signs, so *"o tu lectura está mal, o el parte está falseado"* — the tier-1 detonator — has no mechanism behind it. The reading exists only as prose in the arrival premise. So the sparse brief does not author a shallower version of the same drama; it authors a world where the drama's instrument is missing.

Both are arguably the brief's silence rather than the schema's fault. But a contract in which *omitting a section flips a world rule* is a contract with a default it never declares, and that is the schema's fault. Fix: `excluded[]` should be required, even if the only honest entry is "nothing is excluded" — make the author say it.
