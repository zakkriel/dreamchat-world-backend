# R — input fixture: a `world_model/4` world document (Grelda)

```jsonc
{
  "world_model": "4",
  "world": { "name": "Grelda", "source": "stated",
    "premise": "Twenty thousand people and three thousand living houses. Nobody builds; you plant, you wait two generations, and then the house decides whether you sleep inside. Getting a roof is a negotiation, and some people do it for a living.",
    "mood": "domestic, patient, neighbourly — and the arithmetic underneath is that a house can simply say no forever" },

  "excluded": [
    { "stated": "No money, law, tool or violence obtains a pact. Every attempt fails and the failure is the scene.", "source": "stated" },
    { "stated": "A house that has closed does not reopen by any means.", "source": { "inferred_from": ["a house that does not eat for forty days closes with whatever is inside"] } },
    { "stated": "Houses do not speak in words. They knock, open, close, grow and shrink; anything known about a house is known from what it did.", "source": { "inferred_from": ["they talk to each other through the floor", "you do not yet know enough to understand what it says"] } },
    { "stated": "There is no technique that makes a house accept you. The trade is listening and patience, not a formula.", "source": { "inferred_from": ["no money or law will convince it", "some people do this professionally"] } },
    { "stated": "No house can be planted and used in one lifetime. Nothing shortens the forty to fifty years.", "source": "stated" }
  ],

  "vocabulary": {
    "media": [
      { "name": "house-warm", "source": "stated", "resists": [],
        "affords": [ { "to": "the floor", "degree": "full" } ] },
      { "name": "street air", "source": "stated", "resists": [ { "to": "the floor", "degree": "severe" } ] } ],
    "movements": [
      { "name": "walk", "pace_class": "steady", "source": "stated" },
      { "name": "climb the Cuesta", "pace_class": "slow", "source": { "inferred_from": ["the bottom of the climb"] } } ],
    "channels": [
      { "name": "sight", "source": "stated", "emitted_by": "anything present", "received_by": "anyone present",
        "latency_class": "immediate", "reach": "one extent", "decay": "never", "conceals": "none" },
      { "name": "speech", "source": "stated", "emitted_by": "anyone with agency", "received_by": "anyone present",
        "latency_class": "immediate", "reach": "one extent", "decay": "never", "conceals": "none" },
      { "name": "the floor", "source": "stated",
        "descriptor": "houses talking to each other through the subsoil, slowly",
        "emitted_by": "any entity with agency and extent", "received_by": "the same, and a trained ear with a rod",
        "latency_class": { "class": "very slow", "exemplar": "weeks to go round the block; a month to cross the city" },
        "reach": "contiguous ground", "decay": "never", "conceals": "none" } ],
    "conditions": [
      { "name": "green-eared", "source": "stated",
        "descriptor": "can knock and wait, cannot yet hear an answer",
        "alters": [ { "channel": "the floor", "effect": "hinder", "class": "severe" } ] },
      { "name": "going out", "source": { "inferred_from": ["a house alone and in silence goes out"] },
        "descriptor": "a house running down for want of people rather than grain",
        "alters": [ { "act": "growing a room", "effect": "hinder", "class": "total" } ] } ],
    "substances": [
      { "name": "grain", "source": "stated" },
      { "name": "fire-heat", "source": "stated" },
      { "name": "people-noise", "source": "stated" } ]
  },

  "law": [
    { "name": "a house is not forced", "source": "stated", "enforced_by": "physics", "within": "Grelda",
      "stated": "You can go in through a window and you can move in, but if it did not accept you it will simply outlast you — and it knows how to shut the door when you want to leave.",
      "forbids": { "subject": "any entity with agency", "act": "compelling a pact" } },
    { "name": "forty days", "source": "stated", "enforced_by": "physics", "within": "Grelda",
      "stated": "A house that does not eat closes itself entirely, with whatever is inside. It is the commonest way to die in Grelda.",
      "exemplar": "forty days" },
    { "name": "they grow and shrink unasked", "source": "stated", "enforced_by": "physics", "within": "Grelda",
      "stated": "A room may appear. A room may go. You put up with it." },
    { "name": "your name travels through the floor", "source": "stated", "enforced_by": "physics", "within": "Grelda",
      "stated": "How you behaved as a tenant is public among the houses, and there is no quick way to clean it." },
    { "name": "two nights", "source": "stated", "enforced_by": "physics", "within": "Grelda",
      "stated": "Nobody sleeps two nights running in a house that made them no pact. The first is courtesy; the second the house feels as an intrusion.",
      "exemplar": "the second night" },
    { "name": "no name before three winters", "source": "stated", "enforced_by": "persons", "within": "Grelda", "binds": [],
      "stated": "You do not name a house until it has put up with you.", "exemplar": "three winters" },
    { "name": "you do not praise another house at home", "source": "stated", "enforced_by": "persons", "within": "Grelda", "binds": [],
      "stated": "Not in front of your own." },
    { "name": "the bowl is filled", "source": "stated", "enforced_by": "persons", "within": "Grelda", "binds": [],
      "stated": "A handful of grain at the threshold daily, even when the house is fully rationed. Courtesy, not food." },
    { "name": "the allotment is the Junta's", "source": { "inferred_from": ["Perla Anís decides how much grain each house in the district gets", "the Junta's register"] },
      "enforced_by": "office", "within": "Grelda", "binds": [],
      "stated": "No house draws from the granary except on the allotment the Junta has written for it." }
  ],

  "entities": [
    { "name": "Grelda", "facets": ["extent"], "source": "stated", "extent_class": "vast", "medium": "street air",
      "seen_as": "twenty thousand people, three thousand houses, and tents in the middle of it" },

    { "name": "la plaza mayor", "facets": ["extent"], "source": "stated", "within": "Grelda",
      "extent_class": "large", "medium": "street air", "tension": "tense",
      "seen_as": "canvas rows in the central square, in full view of a city with empty rooms in it" },

    { "name": "Cuesta Menor", "facets": ["extent"], "source": "stated", "within": "Grelda",
      "extent_class": "large", "medium": "street air", "tension": "calm",
      "seen_as": "a hillside district of middling young houses with good tempers" },

    { "name": "la subida", "facets": ["extent"], "source": { "inferred_from": ["at the bottom of the climb"] },
      "within": "Cuesta Menor", "extent_class": "medium", "medium": "street air", "tension": "normal",
      "seen_as": "the climb itself: stone steps, doorways either side, a bowl on every threshold" },

    { "name": "la Ochenta y Tres", "facets": ["extent", "matter", "agency", "demand"], "source": "stated",
      "within": "la subida", "extent_class": "large", "medium": "house-warm", "tension": "tense",
      "bulk_class": "immense", "integrity": "worn",
      "seen_as": "the last house before the top, old and far too big, warm to the hand and shut for as long as anyone remembers",
      "conditions": ["going out"],
      "demands": [
        { "substance": "grain", "rate_class": "slight", "source": "stated",
          "unmet": { "effect": "it closes itself with whatever is inside", "onset_class": { "class": "long", "exemplar": "forty days" } } },
        { "substance": "people-noise", "rate_class": "continuous", "source": "stated",
          "unmet": { "effect": "condition:going out", "onset_class": "very slow" } } ],
      "disposition": [ { "trait": "patient", "strength": "defining", "manner": "waits longer than the person asking" } ],
      "doing": "knocking the same figure into the floor every night",
      "pursuing": [ { "horizon": "long_standing", "toward": "to be answered by someone who is listening rather than negotiating",
                      "source": { "inferred_from": ["the house is asking for something", "it knocks the same rhythm every night", "the door opened for the apprentice and nobody else in sixty years"] } } ],
      "hiding": "It has been running down for want of people, not grain, for sixty years, and the knocking is the last thing it can still do." },

    { "name": "la Cuarenta", "facets": ["extent", "matter", "agency", "demand"],
      "source": { "inferred_from": ["a district of middling young houses with good tempers"] },
      "within": "la subida", "extent_class": "medium", "medium": "house-warm", "tension": "calm",
      "bulk_class": "immense", "integrity": "sound",
      "seen_as": "a young house halfway up with a family in it and a room that was not there last spring",
      "demands": [ { "substance": "grain", "rate_class": "slight", "source": "stated",
                     "unmet": { "effect": "it closes itself with whatever is inside", "onset_class": { "class": "long", "exemplar": "forty days" } } } ],
      "disposition": [ { "trait": "easy", "strength": "moderate", "manner": "opens before you have finished knocking" } ],
      "doing": "growing, slowly, because the children never stop",
      "pursuing": [ { "horizon": "long_standing", "toward": "keep the noise it has",
                      "source": { "inferred_from": ["houses eat people-noise", "a loved house grows"] } } ] },

    { "name": "la casa de Ordo", "facets": ["extent", "matter", "agency", "demand"],
      "source": { "inferred_from": ["Ordo Bes is the best dealer in Cuesta Menor", "he is your master"] },
      "within": "la subida", "extent_class": "medium", "medium": "house-warm", "tension": "normal",
      "bulk_class": "immense", "integrity": "sound",
      "seen_as": "the one with clients in and out all day and no name on it",
      "demands": [ { "substance": "grain", "rate_class": "slight", "source": "stated",
                     "unmet": { "effect": "it closes itself with whatever is inside", "onset_class": { "class": "long", "exemplar": "forty days" } } } ],
      "disposition": [ { "trait": "tolerant", "strength": "strong", "manner": "puts up with strangers all day and shuts early" } ],
      "pursuing": [ { "horizon": "long_standing", "toward": "the traffic to keep coming",
                      "source": { "inferred_from": ["houses eat people-noise"] } } ] },

    { "name": "el granero de la Junta", "facets": ["extent", "holding"],
      "source": { "inferred_from": ["the Junta decides how much grain each house gets", "the Junta's register records what each house eats"] },
      "within": "Cuesta Menor", "extent_class": "small", "capacity_class": "large",
      "holds": [ { "substance": "grain", "abundance": "adequate" } ],
      "seen_as": "a cold stone room with sacks and a counter, the only building on the hill that is not alive" },

    { "name": "las casas de la Cuesta", "facets": ["magnitude", "agency", "demand"],
      "source": { "inferred_from": ["three thousand houses", "a district of middling young houses"] },
      "within": "Cuesta Menor", "magnitude_class": "many",
      "seen_as": "two hundred-odd doorways, each with a bowl, each with an opinion",
      "demands": [ { "substance": "grain", "rate_class": "slight", "source": "stated",
                     "unmet": { "effect": "it closes itself with whatever is inside", "onset_class": { "class": "long", "exemplar": "forty days" } } } ],
      "pursuing": [ { "horizon": "long_standing", "toward": "be fed and be lived in",
                      "source": { "inferred_from": ["houses eat grain, fire-heat and people-noise"] } } ] },

    { "name": "los Sin Trato", "facets": ["magnitude", "agency"], "source": "stated",
      "within": "la plaza mayor", "magnitude_class": "many",
      "seen_as": "people under canvas in the middle of the city, some of them for years",
      "pursuing": [ { "horizon": "long_standing", "toward": "an answer to why the houses will not have them",
                      "source": { "inferred_from": ["nobody knows what Tomás did", "some of them took it as a personal insult"] } } ] },

    { "name": "la Junta de Alimento", "facets": ["agency", "collective"], "source": "stated",
      "legibility": "marked", "descriptor": "a counter, a ledger and a measure",
      "interest": "that the allotment is never questioned house by house",
      "vulnerability": "it writes the register itself, and the register is the only record of what any house was fed" },

    { "name": "los tratantes", "facets": ["agency", "collective"],
      "source": { "inferred_from": ["some people do this professionally", "Ordo is the best in Cuesta Menor", "you are his apprentice"] },
      "legibility": "marked", "descriptor": "a rod of dead-house wood carried at the hip",
      "interest": "that the trade stays a trade — learned slowly, from one person, over years",
      "vulnerability": "a house opened for an apprentice who cannot hear, in front of the whole district" },

    { "name": "Ordo Bes", "facets": ["matter", "agency"], "source": "stated", "within": "la casa de Ordo",
      "seen_as": "fifty, short-tempered, the best on this hill and unpleasant about it",
      "capability": { "moves_by": ["walk", "climb the Cuesta"], "carry_class": "moderate" },
      "senses": { "the floor": "acute" },
      "disposition": [ { "trait": "impatient", "strength": "defining", "manner": "answers the part of the question that was stupid" } ],
      "doing": "refusing, for the second day, to say why the apprentice must not go in",
      "pursuing": [ { "horizon": "long_standing", "toward": "hand the trade to one apprentice who is actually any good",
                      "source": { "inferred_from": ["he is your master", "he is the best in the district"] } } ],
      "hiding": "He has knocked at the Eighty-Three himself, more than once, and it never once answered him." },

    { "name": "Perla Anís", "facets": ["matter", "agency"], "source": "stated", "within": "el granero de la Junta",
      "seen_as": "cordial, quick with a measure, carrying the district's ledger",
      "capability": { "moves_by": ["walk", "climb the Cuesta"], "carry_class": "moderate" },
      "disposition": [ { "trait": "orderly", "strength": "strong", "manner": "writes it down before she answers" } ],
      "doing": "asking pleasant questions about a door that opened",
      "pursuing": [ { "horizon": "imminent", "toward": "get the Eighty-Three's allotment right before anyone notices it was wrong",
                      "source": { "inferred_from": ["she decides how much grain each house gets", "a house that starts accepting people changes what it needs"] } } ],
      "hiding": "The Eighty-Three has been on the smallest allotment on the hill for as long as the ledger goes back, and she has never been asked why." },

    { "name": "Tomás el de la Carpa", "facets": ["matter", "agency"], "source": "stated", "within": "la plaza mayor",
      "seen_as": "a quiet man in tent rows who people defer to without being able to say why",
      "capability": { "moves_by": ["walk", "climb the Cuesta"], "carry_class": "moderate" },
      "disposition": [ { "trait": "patient", "strength": "defining", "manner": "waits out the question" } ],
      "doing": "walking to the foot of the Cuesta and not going up",
      "pursuing": [ { "horizon": "long_standing", "toward": "to be told what he did, by anything that knows",
                      "source": { "inferred_from": ["eleven years and no house has taken him", "nobody knows what he did and he says he does not either"] } } ],
      "hiding": "He has stopped asking people and started leaving a full bowl at doorways that are not his." },

    { "name": "el aprendiz", "facets": ["matter", "agency"], "source": "stated", "within": "la subida",
      "seen_as": "twenty-four, carrying a rod they have not earned",
      "capability": { "moves_by": ["walk", "climb the Cuesta"], "carry_class": "moderate" },
      "conditions": ["green-eared"], "senses": { "the floor": "faint" },
      "pursuing": [ { "horizon": "imminent", "toward": "find out what the Eighty-Three is knocking", "source": "stated" } ] },

    { "name": "la vara de Ordo", "facets": ["matter"], "source": "stated", "within": "la casa de Ordo",
      "bulk_class": "slight", "integrity": "sound",
      "confer": [ { "channel": "the floor" } ],
      "seen_as": "dead-house wood, the only thing the houses hear clearly" },

    { "name": "un cuenco del umbral", "facets": ["matter"], "source": "stated", "within": "la subida",
      "bulk_class": "slight", "integrity": "sound",
      "seen_as": "a bowl at every door in the city, refilled daily whether or not it is needed" },

    { "name": "el registro de la Junta", "facets": ["record", "matter"], "source": "stated",
      "within": "el granero de la Junta", "bulk_class": "moderate", "integrity": "sound",
      "authority": "la Junta de Alimento", "access": { "who": "anyone who asks at the counter" },
      "asserts": [ { "claim": "what each house on the hill eats, how much, and since when", "accurate": true } ] },

    { "name": "la puerta de la Ochenta y Tres", "facets": ["passage"], "source": "stated",
      "connects": ["la subida", "la Ochenta y Tres"],
      "admits": [ { "standing": "opened for", "held_by": "la Ochenta y Tres" } ],
      "obstructs": [ { "act": "entering without a pact" } ], "hazard_class": "none" },

    { "name": "la bajada al centro", "facets": ["passage"],
      "source": { "inferred_from": ["the district is on a hillside", "the square is in the middle of the city"] },
      "connects": ["Cuesta Menor", "la plaza mayor"],
      "admits": [ { "movement": "walk" } ], "hazard_class": "none" }
  ],

  "offices": [
    { "name": "Alimentadora de Cuesta Menor", "source": { "inferred_from": ["Perla decides how much grain each house in the district gets"] },
      "held_by": "Perla Anís", "of": "la Junta de Alimento",
      "confers": [ { "act": "setting a house's allotment" }, { "act": "writing the register" } ],
      "succeeds_by": "appointment by the Junta" },
    { "name": "maestro tratante", "source": { "inferred_from": ["you are his apprentice", "he is the best in the district"] },
      "held_by": "Ordo Bes", "of": "los tratantes",
      "confers": [ { "act": "presenting an apprentice as qualified" } ],
      "succeeds_by": "the trade agreeing that someone is better" }
  ],

  "standing": [
    { "from": "la Ochenta y Tres", "toward": "el aprendiz", "source": "stated",
      "stance": "opened for, once, unasked", "carried_by": "the floor", "persistence": "until changed" },
    { "from": "los Sin Trato", "toward": "el aprendiz", "source": "stated",
      "stance": "taken as a personal insult by some of them", "carried_by": "speech", "persistence": "until changed" },
    { "from": "las casas de la Cuesta", "toward": "el aprendiz",
      "source": { "inferred_from": ["your reputation as a tenant is public among the houses", "the Eighty-Three opened for you"] },
      "stance": "a name going round the block that nobody has finished hearing yet",
      "carried_by": "the floor", "persistence": "permanent" }
  ],

  "opposition": [
    { "between": ["Ordo Bes", "el aprendiz"], "source": "stated",
      "incompatible": "The apprentice cannot both obey the one instruction his master has given him and answer the only house that ever opened for him.",
      "stakes": "Whichever he does, he stops being an apprentice." },
    { "between": ["los Sin Trato", "el aprendiz"], "source": "stated",
      "incompatible": "A door opened for someone who already had a bed, in front of people who have had none for years.",
      "stakes": "If he takes it, he is what they think he is; if he refuses it, the door may not open again." },
    { "between": ["Perla Anís", "la Ochenta y Tres"],
      "source": { "inferred_from": ["she decides each house's allotment", "the Eighty-Three has begun to accept someone"] },
      "incompatible": "A house that accepts a tenant cannot stay on the smallest allotment on the hill, and raising it means explaining the sixty years.",
      "stakes": "The register is the only record, and she writes it." }
  ],

  "processes": [
    { "name": "the word goes round", "source": "stated",
      "acts_on": "standing toward el aprendiz", "direction": "spread",
      "rate_class": { "class": "very slow", "exemplar": "weeks to the end of the block" },
      "terminus": "every house in the city has it" },
    { "name": "the Eighty-Three running down", "source": { "inferred_from": ["a house alone and in silence goes out"] },
      "acts_on": "la Ochenta y Tres", "direction": "degrade", "rate_class": "very slow", "terminus": "it goes out" }
  ],

  "cycles": [
    { "name": "the day on the hill", "source": { "inferred_from": ["the bowl is filled daily", "it knocks at night"] },
      "period_class": "short", "starts_in_phase": "morning",
      "phases": [
        { "name": "morning", "changes": [ { "entity": "un cuenco del umbral", "becomes": "filled" } ] },
        { "name": "the day's traffic", "changes": [ { "entity": "la casa de Ordo", "becomes": "loud" } ] },
        { "name": "night", "changes": [ { "entity": "la Ochenta y Tres", "becomes": "knocking" } ] } ] }
  ],

  "accumulators": [
    { "name": "hunger", "source": "stated", "per": "each entity with demand", "starts_at": "none",
      "stated": "How long since this house last ate.",
      "raised_by": [ { "event": "a day with no grain, no fire and no voices" } ],
      "thresholds": [
        { "at": "moderate", "then": "it stops growing and the warmth drops" },
        { "at": "high", "then": "it begins shutting rooms from the inside" },
        { "at": "extreme", "then": "it closes entirely, with whatever is inside", "irreversible": true,
          "exemplar": "forty days" } ] },
    { "name": "emptiness", "source": { "inferred_from": ["a house alone and in silence goes out", "sixty years without accepting anyone"] },
      "per": "each entity with demand", "starts_at": "none",
      "stated": "How long this house has gone without anyone living in it.",
      "raised_by": [ { "event": "a night with nobody sleeping inside" } ],
      "thresholds": [
        { "at": "moderate", "then": "it stops growing" },
        { "at": "high", "then": "condition:going out — it knocks at night and does not stop" },
        { "at": "extreme", "then": "it goes out and cannot be woken", "irreversible": true } ] }
  ],

  "indicators": [
    { "of": "hunger", "source": { "inferred_from": ["houses grow when loved and shrink when unhappy"] },
      "shows_as": ["a doorway that has narrowed since last month", "a corridor that ends where it did not", "walls cool to the hand"],
      "read_by": { "channel": "sight", "requires": null }, "reliability_class": "moderate" },
    { "of": "emptiness", "source": "stated",
      "shows_as": ["knocking at night, the same figure, earlier each time", "a house too warm for what it is fed"],
      "read_by": { "channel": "the floor", "requires": { "office": "maestro tratante" } },
      "reliability_class": "poor" }
  ],

  "traces": [
    { "of": "a house closing", "source": "stated", "leaves": "a frontage with no opening of any kind", "ages": "never" },
    { "of": "a room reabsorbed", "source": "stated", "leaves": "wall where a doorway was", "ages": "never" },
    { "of": "a bowl left unfilled", "source": "stated", "leaves": "grain missing from a threshold", "ages": "by the next morning" },
    { "of": "a pact made", "source": { "inferred_from": ["you do not name a house until three winters"] },
      "leaves": "a name spoken aloud at a door, or the conspicuous absence of one", "ages": "never" }
  ],

  "epochs": [
    { "name": "when the Eighty-Three still took people", "source": "stated",
      "differed": [ { "topic": "entity", "subject": "la Ochenta y Tres", "then": "it accepted tenants like any other house" } ],
      "surviving_traces": ["a house far too big for a hill of young ones", "the smallest allotment in the ledger, going back further than anyone checks"] }
  ],

  "history": [
    { "name": "the-door-that-opened", "standing": "disputed", "source": "stated",
      "what_happened": "The apprentice walked past the Eighty-Three on an errand and the door opened. He did not knock.",
      "where": "la subida", "who": ["el aprendiz", "Ordo Bes", "los Sin Trato", "la Ochenta y Tres"],
      "knowledge": [
        { "holder": "el aprendiz", "channel": "sight", "path": "direct", "believes": "It opened and he did nothing to make it." },
        { "holder": "los Sin Trato", "channel": "speech", "path": "told",
          "believes": "The dealers have a way in and have been keeping it.", "accurate": false,
          "plausible_because": "Sixty years of refusals, and the one person it opens for is a dealer's apprentice." },
        { "holder": "Ordo Bes", "channel": "speech", "path": "told",
          "believes": "Something he will not say, and he has told the boy to stay out." },
        { "holder": "la Ochenta y Tres", "channel": "the floor", "path": "direct", "believes": "It opened." } ] },
    { "name": "the-eleven-years", "standing": "occurred", "source": "stated",
      "what_happened": "No house in Grelda has accepted Tomás in eleven years, and no reason has ever been given.",
      "where": "la plaza mayor", "who": ["Tomás el de la Carpa", "las casas de la Cuesta"],
      "knowledge": [
        { "holder": "Tomás el de la Carpa", "channel": "sight", "path": "direct", "believes": "It has happened every time and he cannot say why." },
        { "holder": "las casas de la Cuesta", "channel": "the floor", "path": "told",
          "believes": "Whatever went round the floor about him, which has not been revised since." } ] }
  ],

  "arrivals": [
    { "premise": "You are twenty-four and three years apprenticed to Ordo Bes. You can knock and you can wait; you cannot hear an answer yet, and he tells you so daily. The day before yesterday you walked past the Eighty-Three on an errand and its door opened. Ordo says do not go in and will not say why. The Junta has questions. In the square they have heard.",
      "seen_as": "an apprentice carrying a rod they have not earned",
      "within": "la subida",
      "capability": { "moves_by": ["walk", "climb the Cuesta"], "carry_class": "moderate" },
      "source": "stated" }
  ]
}
```
