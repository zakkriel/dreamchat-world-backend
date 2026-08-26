# T — Las Casas de Grelda vs `world_model/1`

Encoded the detailed tier. The schema survives the furniture and fails the premise. `// BREAK` marks
where I had to lie.

## 1. The encoding

```jsonc
{
  "world_model": "1",
  "world": { "name": "Grelda", "premise": "A city of three thousand living houses. Nobody builds; you plant, you wait fifty years, and then the house decides whether you sleep inside.",
             "mood": "domestic, warm, neighbourhood humour, with arithmetic underneath" },

  "vocabulary": {
    "media": [ { "name": "house-warm", "descriptor": "air that smells of cooked grain, walls slightly elastic",
                 "resists": [], "affords": [ { "to": "floor-knock", "degree": "full" } ] },
               { "name": "outside air", "resists": [ { "to": "floor-knock", "degree": "total" } ] } ],
    "movements": [ { "name": "walk", "pace_class": "steady" },
                   { "name": "climb the stair", "pace_class": "slow" } ],
    "channels": [
      { "name": "sight" }, { "name": "speech" },
      { "name": "the floor", "descriptor": "knocks carried through the subsoil",
        "note": "only houses emit and receive it; a trained dealer receives it partially. Two to six weeks to cross a district, months across the city. It never stops and never erases."
        // BREAK: emitter set, receiver set, latency and reach are all prose in `note`.
      } ],
    "conditions": [ { "name": "deaf to the floor", "hinders": [ { "channel": "the floor", "class": "total" } ] } ],
    "substances": [ { "name": "grain" }, { "name": "fire-heat" }, { "name": "people-noise" } ]
    // BREAK: fire-heat is free and unrationed; people-noise is emitted by presence, not stored.
    //        A substance that is produced by activity rather than held has no expression.
  },

  "law": [
    { "name": "a house cannot be forced", "governs": "acceptance",
      "stated": "No money, law, tool or violence has ever obtained a pact. Every attempt fails, and the failure is the scene."
      // BREAK: an impossibility stated as prose. Nothing compares, so nothing refuses.
    },
    { "name": "a closed house never reopens", "governs": "integrity",
      "stated": "Forty days unfed and a house seals itself — doors, windows, chimneys — with whatever is inside. It has never been reopened by any means." },
    { "name": "a cutting only takes near its stock", "governs": "growth",
      "stated": "Transplanting between cities kills the cutting and sickens the mother." }
  ],

  "places": [
    { "name": "Grelda", "parent": null, "extent_class": "vast", "sort": "valley city", "medium": "outside air" },
    { "name": "Cuesta Menor", "parent": "Grelda", "extent_class": "large", "sort": "hillside district",
      "medium": "outside air", "tension": "calm" },
    { "name": "Plazoleta de los Cuencos", "parent": "Cuesta Menor", "extent_class": "medium",
      "sort": "grain market", "medium": "outside air", "tension": "normal" },
    { "name": "the Board office", "parent": "Cuesta Menor", "extent_class": "small",
      "sort": "the only dead building in the district", "medium": "outside air", "tension": "calm" },
    { "name": "the workshop under the Forty-One", "parent": "Cuesta Menor", "extent_class": "small",
      "sort": "dealer's workshop inside a tolerant old house", "medium": "house-warm", "tension": "normal" },

    { "name": "the Eighty-Three", "parent": "Cuesta Menor", "extent_class": "medium",
      "sort": "two-hundred-year-old house, three floors and a fourth that comes and goes",
      "medium": "house-warm", "tension": "tense",
      "ambient_demand": [ { "requires": "a pact with this house", "absent_effect": "the doors close from inside",
                            "onset": "the second night" } ]
      // BREAK 1: this is also a person. See people[] below — the SAME entity, authored twice,
      //          under one canonical name, which the schema says is the only join key.
      // BREAK: "a fourth floor that comes and goes" — extent is a static authored class.
      // BREAK: ambient_demand is the wrong shape. The demand is not the place's air; it is a
      //        standing relation between this entity and that person.
    },
    { "name": "the main square", "parent": "Grelda", "extent_class": "large",
      "sort": "tent field at the exact centre of the city", "medium": "outside air", "tension": "tense" }
  ],

  "ways": [
    { "name": "the door of the Eighty-Three", "connects": ["Cuesta Menor", "the Eighty-Three"],
      "state": "open", "obstructs": [], "affords": ["walk"] }
      // BREAK: it obstructs everyone in Grelda except one apprentice, on the basis of a relation,
      //        not a movement. `obstructs` can only name movements, so the gate is unstatable.
  ],

  "things": [
    { "name": "the threshold bowl", "bulk_class": "slight", "integrity": "sound",
      "where": { "in_place": "Cuesta Menor" } },
    { "name": "a dealer's rod", "bulk_class": "slight", "integrity": "sound",
      "where": { "carried_by": "Ordo Bes" } },
    { "name": "the Board register", "bulk_class": "moderate", "integrity": "sound",
      "where": { "in_place": "the Board office" } },
    { "name": "the Guild register", "bulk_class": "moderate", "integrity": "sound",
      "where": { "in_place": "Cuesta Menor" } },
    { "name": "a mis-born cutting", "bulk_class": "slight", "integrity": "sound",
      "where": { "in_place": "Grelda" } }
  ],

  "stocks": [
    { "name": "grain", "held_in": "Grelda", "abundance": "adequate",
      "drawn_by": "the Board, allotting to every house", "replenished_by": "the harvest" }
    // BREAK: the houses EAT this. A stock that an entity requires, at a rate, on pain of death,
    //        has no expression: `drawn_by` is prose and there is no consumer side.
  ],

  "processes": [
    { "name": "a contented house grows", "acts_on": "a well-fed, well-noised house",
      "direction": "grow", "rate_class": "very slow", "terminus": null },
    { "name": "an unhappy house reabsorbs a room", "acts_on": "a house kept silent",
      "direction": "shrink", "rate_class": "very slow", "terminus": null },
    { "name": "the knocking comes earlier", "acts_on": "the Eighty-Three's nightly rhythm",
      "direction": "advance", "rate_class": "slow", "terminus": null }
    // BREAK: all three targets are prose. Nothing states that the effect is a change of extent,
    //        of containment (a room ceases to exist), or of a cycle's own period.
  ],

  "cycles": [
    { "name": "the night knocking", "period_class": "short", "starts_in_phase": "silent",
      "phases": [ { "name": "silent", "changes": [] },
                  { "name": "knocking", "changes": [] } ] }
    // BREAK: no phase change expressible — the knock is a channel emission, not a state flip,
    //        and the period is itself drifting (see processes).
  ],

  "accumulators": [
    { "name": "hunger", "stated": "How long a house has gone without eating.",
      "starts_at": "low", "raised_by": ["a day with no grain, heat or noise"],
      "threshold": { "class": "extreme", "then": "The house seals itself with whatever is inside, permanently." } }
    // BREAK: three thousand houses need three thousand of these. An accumulator has no scope,
    //        so this reads as one city-wide hunger. And `then` cannot say "irreversible",
    //        nor "traps the contents".
  ],

  "traces": [
    { "of": "a house sealing", "leaves": "a blank frontage with no opening of any kind", "ages": "never" },
    { "of": "a room being reabsorbed", "leaves": "a wall where a doorway was", "ages": "never" },
    { "of": "a bowl left unfilled", "leaves": "grain missing from the threshold", "ages": "by the next day" }
  ],

  "propagation": [
    { "of": "how a tenant behaved", "spreads": "house to house through the floor, weeks across a district, months across the city, and it never erases" },
    { "of": "an Old House striking the floor", "spreads": "the whole district goes still for a day" }
    // BREAK: `spreads` is prose. The window during which nobody yet knows — the load-bearing
    //        mechanic of the whole brief — is not a quantity the engine can hold.
  ],

  "collectives": [
    { "name": "the Dealers' Guild", "legibility": "marked", "descriptor": "a carved rod at the hip",
      "interest": "that nobody proves the trade can be learned in a year",
      "speaks_through": "Ordo Bes" },
    { "name": "the Food Board", "legibility": "marked", "descriptor": "a folder and a cordial manner",
      "interest": "budget stability and no inquiry", "speaks_through": "Perla Anís" },
    { "name": "the Pactless", "legibility": "marked", "descriptor": "tent canvas and no address",
      "interest": "to have it admitted that the shortage is criterion, not supply",
      "speaks_through": "Halma Ruiz" },
      // BREAK: "tres voceros rotativos" — the office rotates. `speaks_through` is a person, so
      //        the role and its current holder are the same field.
    { "name": "the Old Houses", "legibility": "marked", "descriptor": "eleven frontages nobody approaches",
      "interest": "unknown", "speaks_through": null }
      // BREAK: a collective of non-people. Its members are entities in places[].
  ],

  "people": [
    { "name": "Ordo Bes", "seen_as": "a spare, short-tempered man with a carved rod",
      "role": "the district's best dealer", "belongs_to": ["the Dealers' Guild"],
      "starts_in": "the workshop under the Forty-One",
      "capability": { "moves_by": ["walk", "climb the stair"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the floor": "absent" },
      "conditions": ["deaf to the floor"],
      "disposition": [ { "trait": "impatient", "strength": "defining",
                         "manner": "answers a question with the part of it that was stupid" } ],
      "doing": "refusing to explain why the apprentice must not go in",
      "pursuing": [ { "horizon": "long_standing", "toward": "retire leaving an apprentice worth the name",
                      "progress": "late", "step": "withholding the one thing he cannot teach" } ],
      "obligation": [ { "owed_to": "the Dealers' Guild", "stated": "does not teach outside the apprenticeship" } ],
      "regard": [ { "toward": "the Eighty-Three", "stance": "will not go near it and will not say why" } ],
      "hiding": "He lost the floor-hearing four years ago and his results did not fall, which terrifies him for what it says about the thirty years before." },

    { "name": "Perla Anís", "seen_as": "a cordial woman, quick with figures, carrying a folder",
      "role": "allots the district's grain", "belongs_to": ["the Food Board"],
      "starts_in": "the Board office",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the floor": "absent" },
      "disposition": [ { "trait": "orderly", "strength": "strong", "manner": "writes it down before answering" } ],
      "doing": "asking pleasant questions about a door that opened",
      "pursuing": [ { "horizon": "long_standing", "toward": "promotion to the inner ring",
                      "progress": "advanced", "step": "no inquiry into the low-ring cut" } ],
      "hiding": "She keeps a private count of the houses sealed since her ration cut, in a notebook with no administrative reason to exist." },

    { "name": "Tomás", "seen_as": "a calm man from tent forty-seven whom people defer to",
      "role": "Pactless, eleven years in the square", "belongs_to": ["the Pactless"],
      "starts_in": "the main square",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the floor": "absent" },
      "disposition": [ { "trait": "patient", "strength": "defining", "manner": "waits out the question" } ],
      "doing": "walking to the foot of the Cuesta and not going up",
      "pursuing": [ { "horizon": "long_standing", "toward": "an explanation rather than a roof",
                      "progress": "early", "step": "never asking anyone directly" } ],
      "hiding": "At twenty-three he was inside a house when it sealed, with four others, and he came out. He has never said how. The houses know." },

    { "name": "Halma Ruiz", "seen_as": "a young woman who has never slept under a roof",
      "role": "current speaker of the Pactless", "belongs_to": ["the Pactless"],
      "starts_in": "the main square",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the floor": "absent" },
      "disposition": [ { "trait": "direct", "strength": "strong", "manner": "puts the worst option on the table first" } ],
      "doing": "counting votes for a motion she knows is suicide",
      "pursuing": [ { "horizon": "imminent", "toward": "occupy one of the Old Houses",
                      "progress": "advanced", "step": "the square assembly" } ],
      "hiding": "She has no second card and knows it." },

    { "name": "Vela Roncal", "seen_as": "an old woman with soil to the elbow",
      "role": "planter, legal for the Board and otherwise for anyone", "belongs_to": [],
      "starts_in": "Grelda",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the floor": "faint" },
      "disposition": [ { "trait": "secretive", "strength": "strong", "manner": "answers about the weather" } ],
      "pursuing": [ { "horizon": "long_standing", "toward": "a stock that survives northern cold",
                      "progress": "early", "step": "cuttings she will not live to see planted" } ],
      "hiding": "Her last three cuttings were born wrong — fast-growing, over-eating, deaf to the floor — and one of them is no longer in her nursery." },

    { "name": "the Eighty-Three", "seen_as": "the last house before the wall, always warm",
      "role": "two hundred years old; one hundred and forty documented refusals; sixty years accepting nobody",
      "belongs_to": [], "starts_in": "Cuesta Menor",
      "capability": { "moves_by": [], "carry_class": "immense" },
      "senses": { "sight": "absent", "the floor": "acute" },
      "disposition": [ { "trait": "withdrawn", "strength": "defining", "manner": "does not open" } ],
      "doing": "knocking the floor every night, earlier each time, in a rhythm no dealer can translate",
      "pursuing": [ { "horizon": "long_standing", "toward": "unknown", "progress": "unknown", "step": "the knocking" } ],
      "regard": [ { "toward": "the apprentice", "stance": "opened the door, unasked" } ],
      "hiding": "Why it stopped, sixty years ago, in the same eighteen months the Guild register skips."
      // BREAK 1, the whole point: this entity now exists TWICE — once in places[] and once here —
      //   under one canonical name. It needs a body people stand inside AND a mind that decides.
      //   `capability.moves_by: []` is a lie of omission; `senses.sight: absent` is the only way
      //   to say it perceives by the floor alone; `carry_class: immense` is me abusing a field to
      //   mean "you are inside it".
    }
  ],

  "opposition": [
    { "between": ["Halma Ruiz", "Perla Anís"],
      "incompatible": "The square cannot be housed without admitting the Board's allotment decides who gets refused.",
      "stakes": "Twelve years of ration cuts and thirty-one sealed houses become a matter of record." },
    { "between": ["Ordo Bes", "the Eighty-Three"],
      "incompatible": "He cannot both keep the apprentice out and keep the trade's competence unquestioned.",
      "stakes": "If a boy who cannot hear the floor gets a door opened, the Guild's examination means nothing." }
  ],

  "norms": [
    { "name": "no name before three winters", "binds": [],
      "stated": "You do not name a house until it has put up with you for three winters." },
    { "name": "the bowl is left filled", "binds": [],
      "stated": "A handful of grain on the threshold daily, even when the house is fully rationed. It is courtesy, not food, and a house notices when it is missing." },
    { "name": "you do not praise another house at home", "binds": [], "stated": "..." },
    { "name": "you do not ask a Pactless why", "binds": [], "stated": "..." },
    { "name": "not aloud near the inner ring", "binds": [],
      "stated": "Nobody discusses the Old Houses within earshot of them, and nobody can say why." }
  ],

  "epochs": [
    { "name": "before the cut", "differed": [ { "topic": "stock", "subject": "grain", "then": "adequate to the low ring" } ],
      "surviving_traces": ["thirty-one blank frontages in the low ring"] },
    { "name": "the eighteen months", "differed": [ { "topic": "record", "subject": "the Guild register", "then": "kept" } ],
      "surviving_traces": ["a gap with no entries, sixty years back"] }
  ],

  "history": [
    { "name": "the-sealing-Tomás-walked-out-of", "what_happened": "A house sealed with five people inside. One came out.",
      "where": "Grelda", "who": ["Tomás"],
      "knowledge": [
        { "holder": "Tomás", "channel": "sight", "path": "direct", "believes": "He got out and has never said how." },
        { "holder": "Ordo Bes", "channel": "speech", "path": "rumor",
          "believes": "The boy was never inside; the count was wrong.", "accurate": false,
          "plausible_because": "It is the only version that does not break a hard rule." } ] },
    { "name": "the-ration-cut", "what_happened": "The low ring's allotment was reduced and thirty-one houses sealed over twelve years.",
      "where": "Grelda", "who": ["Perla Anís"],
      "knowledge": [
        { "holder": "Perla Anís", "channel": "sight", "path": "direct", "believes": "She did it and she has the count." },
        { "holder": "Halma Ruiz", "channel": "speech", "path": "inference",
          "believes": "The Board is emptying the low ring on purpose.", "accurate": false,
          "plausible_because": "The outcome is indistinguishable from intent." } ] }
  ],

  "arrivals": [
    { "premise": "You are three years apprenticed to Ordo Bes. The day before yesterday you walked past the Eighty-Three on an errand and its door opened. You did not knock.",
      "seen_as": "a young apprentice carrying a rod they have not earned",
      "place": "Cuesta Menor",
      "capability": { "moves_by": ["walk", "climb the stair"], "carry_class": "moderate" },
      "senses": { "sight": "normal", "the floor": "faint" } } ]
}
```

## 2. The breaks

**B1 — an entity that is a place and an agent at once. (i) inexpressible.** *"las casas están vivas …
toman decisiones sobre quién duerme adentro"* (det. §1); *"Es un personaje, no un escenario"* (medio
§7). Sections `places[]`/`people[]`. The top-level arrays **are the type system** — an entity's kind
is which array it lives in. Grelda's central object needs body, mind, matter and appetite at once, so
I authored it twice under one canonical name, which decision 3 declares the only join key. Every
break below is downstream of this one.

**B2 — an entity that consumes. (i).** *"Comen tres cosas. Grano … Calor de fuego … ruido de gente"*
(§2); *"Cuarenta días sin comer y una casa se cierra"* (§3). `stocks[]` are place-held with a prose
`drawn_by`: no consumer, no rate, no consequence of non-supply. And *people-noise* is emitted by
activity and cannot be stored, so it is not a stock in any sense the section admits.

**B3 — passage gated on a relation, not a capability. (iii) wrong-shaped.** *"Una casa no se puede
forzar"*; *"nadie duerme dos noches seguidas en una casa que no le hizo trato"* (§3).
`ways[].obstructs/affords` can only name **movements**. The Eighty-Three's door obstructs a whole city
and affords one apprentice on the basis of a standing relation. Expressing it needs a movement type
called "having a pact" — a lie about what a movement is.

**B4 — reputation that propagates slowly and never erases. (ii) inert prose.** *"la reputación de un
inquilino es municipal … no se detiene y no se borra"* (§3); *"hay una ventana durante la cual todavía
nadie sabe lo que hiciste"* (§2). `channels[]`/`propagation[]` carry emitter set, receiver set,
latency, reach and decay entirely in free text. That window is the brief's load-bearing device and
nothing can hold it.

**B5 — one accumulator per entity. (iii).** Forty days × three thousand houses. `accumulators[]` has
no scope, so `hunger` reads as one city-wide quantity. `threshold.then` also cannot say *irreversible*
or *traps the contents*, both stated absolutely: *"No se vuelve a abrir nunca, por ningún medio"*.

**B6 — structural change. (ii).** *"Crecen y se encogen … suma un cuarto … se lo reabsorbe"* (§2);
*"un cuarto piso que aparece y desaparece"* (§6). `processes[].acts_on` is prose and `extent_class` is
authored once; no process can target a property and no place can gain or lose a contained place.
Relatedly the knocking *"cada vez más temprano"* is a cycle whose own period drifts, and
`period_class` is static.

**B7 — a rotating office. (iii).** *"tres voceros rotativos"*, *"vocera actual"* (§5, §7).
`collectives[].speaks_through` names a person, collapsing the office into its holder.

**B8 — impossibility as canon. (i).** §11: *"No hay forma de forzar un trato. Cualquier intento …
fracasa, y el fracaso es siempre la escena, nunca el éxito."* `law[]` holds this as prose with a
`governs` tag; nothing compares, so nothing refuses. In a world whose drama is a thing that cannot be
done, the negative is more load-bearing than most positives.

**B9 — a collective of non-people. (iii).** *"Las Casas Viejas … las casas jóvenes les hacen caso"*
(§5). `belongs_to` lives on `people[]`, so eleven buildings cannot be members of anything.

**B10 — counts that are fiction, not engineering. (iii).** *"ciento cuarenta rechazos"*, *"treinta y
una casas cerradas"* are the **contents of a register a player reads**. Decision 2's no-number rule is
absolute and forbids them; it should forbid numbers only in fields the engine computes on.

**B11 — magnitude. (i).** *"tres mil casas"*, *"ochocientas personas bajo lona"*. Entities are
enumerated one by one; the Pactless are a crowd that must exist, be counted roughly, and yield
individuals on demand.

## 3. Minimal fixes — the general primitive in each case

Second world throughout: **Cinder Reach**, a mining town on a slag field. No biology anywhere.

**F1. Facets, not sections.** *An entity holds a set of facets; each is a capacity, and any
combination is legal.* One `entities[]`; the current sections become facet definitions — `extent`
(things can be inside it), `agency` (it decides), `matter` (bulk, integrity), `holding`. B1, B9 and
half of B6 dissolve into it. *Grelda:* the Eighty-Three = `extent + agency + matter`; a tent =
`matter`; a district = `extent`. *Cinder Reach:* the Kiln = `extent + agency + matter` — crews work
inside it and it refuses to light for some of them; the cage = `matter`; the Guild = `agency`, no body.

**F2. Demand.** *A subject requires a substance at a rate class; unmet, an effect after an onset
class.* Generalises `ambient_demand`, which is this idea nailed to places and their occupants. Supply
may be a stock draw **or** emission by activity. *Grelda:* the house demands grain (slight),
fire-heat (free), people-noise (emitted by presence, unrationable). *Cinder Reach:* a miner demands
clean air from the bellows lines; unmet ⇒ a condition, onset by exposure.

**F3. Channels carry participants and latency.** *A channel declares who emits, who receives, a
latency class, a reach and a decay; propagation names a channel instead of describing itself.*
*Grelda:* the floor — emitted and received by entities with `agency + extent`, latency `very slow`,
reach `contiguous ground`, decay `never`. *Cinder Reach:* relay shout — posted relays only, latency
`slow`, reach `one shaft`, decay `by shift end`.

**F4. Standing is a top-level relation.** *held_by → toward, a stance, a propagating channel, a
persistence class.* Lifting `regard` off `people[]` lets a non-person hold one. *Grelda:* every house
holds standing toward every tenant, via the floor, persistence `permanent`. *Cinder Reach:* the Guild
holds standing toward each crew, via the shift log, persistence `until reviewed`.

**F5. Passage compares predicates.** *A barrier states requirements over the mover; a requirement may
name a capability, a condition, or a standing.* *Grelda:* the door `admits: [{standing:"pact",
held_by:"self"}]`. *Cinder Reach:* the cage `admits: [{standing:"in good order", held_by:"the Guild"}]`
and separately `obstructs: ["climb"]`.

**F6. Accumulators declare scope.** *`per: world | place | each entity with facet X`.* *Grelda:*
hunger, per entity with `agency + extent`. *Cinder Reach:* lung-load per person; slag depth per world.

Three smaller, same discipline: a **role** is an entity with `agency` that a person *occupies*,
occupancy being a standing (B7); `law[]` gains `forbids` with a typed subject, so an impossibility is a
comparison not a sentence (B8); an entity may carry a **magnitude class** and stand in for many, from
which individuals are promoted (B11).

I propose nothing for §11's *"que Tomás haya salido … debe seguir siendo única"*. A constraint that one
historical exception must never generalise is an authoring directive, not world state; inventing a
field for it is exactly the failure this test looks for.

## 4. Tier fidelity

It degrades **gracefully downward and badly upward.** The basic tier authors less, not worse: five
hard rules, three people, three objects, one district, one house — all have homes, nothing distorted.

But the failure is not a detail-tier luxury. Tier 1's own opening says *"las casas están vivas"* and
*"hablan entre ellas por el suelo"* — B1 and B4 are in the simplest brief. The schema fails tier 1
too; it just fails quietly, because with one named house the duplication reads as an eccentricity
rather than a collapse.

What the extra detail has nowhere to go: the eighteen-month gap in the Guild register (§8), an
*absence in an object*, expressible only as a descriptor string; the rotating speakership (§5); the
propagation latencies (§2); the thirty-one closures (§5); the §10 seed list, authored *potential* —
armed situations with no trigger — of which `accumulators` and `opposition` cover perhaps half; and
§11's negative canon, longer and more load-bearing here than several positive sections.

## 5. The one change to the shape

**Stop making the section list the type system.**

Every break above is one mistake in variation: the schema encodes *what a thing is* by *which array it
is in*. That is a closed ontology in the clothes of an open one. The vocabulary half is genuinely
agnostic; then the content half hands the world a fixed cast of eight kinds and says yours must be one
of these. Grelda's houses are not exotic — they are the ordinary case of an entity with more than one
capacity, and the same collision arrives for a sapient river, a haunted object, a ship with a mind, or
an institution that owns a building.

One `entities[]`, facets composing freely, sections demoted to facet definitions — `extent`, `agency`,
`matter`, `holding`, `demand`, `standing`. A seat can produce that from this brief in one pass,
because it matches how the brief reads: *"Es un personaje, no un escenario"* is the author telling you
in plain Spanish that your two boxes were always one box.
