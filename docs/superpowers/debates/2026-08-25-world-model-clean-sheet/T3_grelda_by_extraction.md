# v3 re-encoding — Las Casas de Grelda

## 1. Encoding

```json
{
  "world_model": "3",
  "world": { "name": "Grelda", "premise": "The houses are alive. They grow from cuttings, eat grain, fire-warmth and the noise of people, grow and shrink rooms unasked, and close forever after forty days unfed." },

  "excluded": [
    "Houses do not speak in words — only knock, open, close, grow, shrink. No dialogue or inner voice for a house.",
    "No magic, spells, wizards, or schools of it.",
    "No way to force a deal. Every attempt fails, and the failure is always the scene.",
    "No house acts out of cruelty — harm a house causes is indifference, never malice.",
    "A closed house never reopens, by any means, ever. One known anomaly exists and must stay unique.",
    "No technical fix for the unhoused — the only path is convincing house by house."
  ],

  "vocabulary": {
    "movements": [ { "name": "walk", "pace_class": "steady" } ],
    "channels": [ { "name": "underground speech", "descriptor": "knock-patterns carried through the ground",
      "emitted_by": "any entity with matter, struck correctly", "received_by": "houses; a person with the ear or the staff",
      "latency_class": "very slow", "reach": "district to district over weeks", "decay": "never", "conceals": "none" } ],
    "conditions": [ { "name": "lost the ear", "alters": [ { "channel": "underground speech", "effect": "hinder", "class": "severe" } ] } ],
    "substances": [ { "name": "grain" } ]
  },

  "law": [
    { "name": "a house cannot be forced", "enforced_by": "physics",
      "stated": "No money, law, tool or violence has ever produced a deal a house did not choose.",
      "forbids": { "subject": "any entity with agency", "act": "compelling a house to accept an occupant" } },
    { "name": "forty days unfed closes a house forever", "enforced_by": "physics",
      "stated": "A house unfed forty days seals every door, window and flue, with whatever is inside, and never reopens." },
    { "name": "one courtesy night, never two", "enforced_by": "physics",
      "stated": "A second night in a house with no deal is felt as intrusion; the house shuts its inner doors." },
    { "name": "no planting without the Ration Board's leave", "enforced_by": "office",
      "stated": "Planting requires the Board's authorisation.", "binds": [ "anyone" ],
      "precedent": "half the upper districts broke this eighty years ago and it is no longer discussed" }
  ],

  "entities": [
    { "name": "Grelda", "facets": ["extent"], "extent_class": "vast", "medium": "waking air", "tension": "normal" },
    { "name": "Cuesta Menor", "facets": ["extent"], "within": "Grelda", "extent_class": "large", "medium": "waking air", "tension": "calm" },

    { "name": "La Ochenta y Tres", "facets": ["extent", "matter", "holding", "demand", "agency"],
      "within": "Cuesta Menor", "seen_as": "a warm three-storey house against the upper wall, with a fourth floor that comes and goes",
      "bulk_class": "immense", "integrity": "sound", "extent_class": "medium", "medium": "waking air", "tension": "tense",
      "capacity_class": "large", "holds": [],
      "demands": [ { "substance": "grain", "rate_class": "minimal", "unmet": { "effect": "law:forty days unfed closes a house forever", "onset_class": "long" } } ],
      "doing": "knocking the ground every night, earlier each time",
      "pursuing": [ { "horizon": "long_standing", "toward": "the knocking pattern reaching someone who can answer it", "progress": "advanced" } ],
      "hiding": "why it opened for someone who already had a roof, after a hundred and forty refusals" },

    { "name": "the door of the Eighty-Three", "facets": ["matter", "passage"],
      "bulk_class": "small", "connects": [ "Cuesta Menor", "La Ochenta y Tres" ],
      "obstructs": [ { "standing": "no deal made" } ], "admits": [ { "movement": "walk" } ] },

    { "name": "Ordo Bes", "facets": ["matter", "agency"], "within": "Cuesta Menor", "bulk_class": "moderate",
      "seen_as": "a sour, exacting man with thirty years at the trade",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" }, "conditions": ["lost the ear"],
      "disposition": [ { "trait": "exacting", "strength": "strong", "manner": "explains everything twice, except when he doesn't" } ],
      "doing": "refusing to explain why you must not go in",
      "pursuing": [ { "horizon": "long_standing", "toward": "retire an apprentice who actually hears", "progress": "advanced" } ],
      "hiding": "he lost the ear four years ago in a flood; his results never dropped, which frightens him about everything before" },

    { "name": "Perla Anís", "facets": ["matter", "agency"], "within": "Cuesta Menor", "bulk_class": "moderate",
      "seen_as": "a cordial, quick official with a folder",
      "pursuing": [ { "horizon": "ongoing", "toward": "a transfer to the inner ring", "progress": "advanced" } ],
      "doing": "asking cordial questions about the Eighty-Three",
      "hiding": "she wrote the lower-ring ration cut herself and privately tallies the houses it has closed" },

    { "name": "Tomás", "facets": ["matter", "agency"], "within": "Grelda", "bulk_class": "moderate",
      "seen_as": "a calm man, eleven years in the plaza, no house has ever taken him",
      "pursuing": [ { "horizon": "long_standing", "toward": "an explanation, more than a roof" } ],
      "hiding": "he got out of a house that closed on him and four others, and has never told anyone how" },

    { "name": "the Guild of Brokers", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "sole rights to negotiate with houses", "pursuing": [ { "horizon": "ongoing", "toward": "keep negotiation unteachable outside the Guild" } ],
      "vulnerability": "much of the craft is theatre; provable in a year, the Guild ends" },

    { "name": "the Ration Board", "facets": ["agency", "collective", "holding"], "legibility": "marked",
      "interest": "budget stability and no inquiry", "pursuing": [ { "horizon": "ongoing", "toward": "keep the ration debate settled the same way every year" } ],
      "vulnerability": "its own register proves the twelve-year cut closed houses",
      "capacity_class": "vast", "holds": [ { "substance": "grain", "abundance": "adequate" } ] },

    { "name": "the Unhoused", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "recognition that the shortage is houses' judgement, not supply",
      "pursuing": [ { "horizon": "right_now", "toward": "be heard on why so many rooms sit empty" } ],
      "vulnerability": "their only strategy is occupation, and a house that never accepted you will not let you leave either" },

    { "name": "the crowd in the plaza", "facets": ["magnitude"], "within": "Grelda",
      "magnitude_class": "many", "seen_as": "tents in the square, hundreds of them" },

    { "name": "Halma Ruiz", "facets": ["matter", "agency"], "within": "Grelda", "bulk_class": "moderate",
      "seen_as": "the only person in Grelda raised never having slept under a roof",
      "pursuing": [ { "horizon": "right_now", "toward": "occupy one of the Old Houses, knowing it will fail" } ] },

    { "name": "the Ration Register", "facets": ["record"], "within": "Cuesta Menor",
      "asserts": [ "what every house eats, how much, since when" ], "access": { "scope": "public" }, "authority": "the Ration Board" },
    { "name": "the Guild Register", "facets": ["record"], "within": "Cuesta Menor",
      "asserts": [ "every deal request and its result", "an eighteen-month gap sixty years ago with no entries" ],
      "access": { "scope": "public" }, "authority": "the Guild of Brokers" },

    { "name": "the broker's staff", "facets": ["matter"], "within": "Cuesta Menor", "bulk_class": "small", "integrity": "sound",
      "confers": [ { "channel": "underground speech" } ] }
  ],

  "offices": [ { "name": "Voice of the Unhoused", "held_by": "Halma Ruiz", "of": "the Unhoused", "succeeds_by": "rotation among three" } ],

  "standing": [ { "from": "Ordo Bes", "toward": "you", "stance": "will not explain, which from him is itself an explanation", "persistence": "until changed" } ],

  "opposition": [ { "between": ["the Ration Board", "the Unhoused"],
    "incompatible": "the Board needs the ration debate to keep going nowhere; the Unhoused need it settled",
    "stakes": "who eats, and which districts empty out" } ],

  "accumulators": [ { "name": "houses closed since the cut", "per": "world", "starts_at": "none",
    "raised_by": [ { "event": "a district-twelve house going forty days unfed" } ],
    "thresholds": [ { "at": "moderate", "then": "Perla Anís's private tally becomes hard to explain administratively" } ] } ],

  "indicators": [ { "of": "La Ochenta y Tres's hiding", "shows_as": [ "the knocking rhythm, earlier every night", "warmer than its diet explains" ],
    "read_by": { "channel": "underground speech" }, "reliability_class": "poor" } ],

  "history": [
    { "name": "the-one-who-came-out", "standing": "occurred",
      "what_happened": "A house closed with five people inside. One of them, Tomás, was later found outside it.",
      "where": "Grelda", "who": [ "Tomás" ],
      "knowledge": [ { "holder": "Tomás", "channel": "sight", "path": "direct", "believes": "I got out. I don't fully know how." } ] },
    { "name": "why-the-door-opened", "standing": "disputed",
      "what_happened": "La Ochenta y Tres opened, unforced, for you.", "where": "Cuesta Menor", "who": [],
      "knowledge": [
        { "holder": "Ordo Bes", "channel": "underground speech", "path": "inference", "believes": "it heard something in you it hasn't heard in sixty years" },
        { "holder": "Perla Anís", "channel": "hearsay", "path": "told", "believes": "it is a malfunction the Board should investigate", "accurate": false, "plausible_because": "that is the only explanation the Board's own registers admit" } ] }
  ],

  "arrivals": [ { "premise": "Broker's apprentice, three years with Ordo Bes. Two days ago the Eighty-Three's door opened for you — no knock, no deal — after sixty years and a hundred and forty refusals.",
    "seen_as": "a young apprentice with a broker's staff you haven't earned the right to carve", "place": "Cuesta Menor",
    "capability": { "moves_by": ["walk"], "carry_class": "moderate" } } ]
}
```

## 2. Validity self-check

**O1–O11:** all pass. O1: Grelda/Cuesta Menor/La Ochenta y Tres, three extents, one `passage`.
O2–O4: multiple `agency` entities, each with `pursuing`; `hiding` on four of them. O5: Board vs.
Unhoused. O6: every `matter` entity now carries `bulk_class` — v2 omitted this on every person. O7:
one demand (`grain`), one named supplier (the Board `holds` it) — I dropped the "noise of people"
demand rather than fake a supplier; see §4. O8: the accumulator has both a `raised_by` and a
threshold. O9: the indicator points at `hiding`, an actual held field. O10: one `arrivals` entry.
O11: `excluded[]` present, six entries.

**R1–R12:** all clear except one deliberately exercised, not violated. R1: every name used resolves
to a declared entity (added `Grelda` and `Tomás`, both missing in v2). R5: `the crowd in the plaza`
(magnitude) is never referenced individually anywhere else — Halma is her own entity with no
formal link back to it, by design. R7: audited every key against its entity's facet list. R10: every
`agency` entity, including both collectives, now has `pursuing`. R12 is the one I exercised on
purpose: `why-the-door-opened` is `disputed` with two holders who genuinely disagree (inference vs.
told-and-wrong) — satisfying, not violating, the refusal.

## 3. Auditing my v2 document against v3's refusals

**R10 fires, correctly, on my own v2 encoding.** `La Ochenta y Tres` had `agency` with
`disposition`/`pursuing` deliberately left empty, to respect the brief's "no explained intention"
(§11). v3 refuses that outright. I do not think the refusal is wrong — I think my v2 reading of
`agency` was too narrow. The fix above states `pursuing.toward` as an observable trajectory ("the
knocking reaching someone who can answer it") rather than a felt want, which stays inside the
brief's letter while satisfying O3. Same fire, same fix, on both collectives (v2 gave them
`interest` only, no `pursuing`).

**R6 nearly fires and I think it's a real, defensible near-miss, not a bug.** `excluded[]` forbids
a house acting from "malice"; giving the house `agency`+`pursuing` risks reading as exactly the
"casa con plan" the brief bans. I judge it doesn't cross the line because the pursuing entry
narrates pattern, not intent — but a stricter reader could reasonably disagree, and nothing in the
refusal list can tell the two apart mechanically. That is a genuine gap, not a false positive: R6
can only catch a *stated* contradiction, never a *tonal* one.

**No refusal I audited rejects a document that should be legal.** Everything that fired in v2 fired
on a real gap I had actually left.

## 4. Residual ambiguity

1. **`agency.pursuing` vs. `collective.interest` on the same entity.** Both Guild-of-Brokers-shaped
   collectives now carry near-duplicate content in two keys. Nothing says whether `pursuing` should
   restate the standing `interest` or must differ from it (a current tactic vs. a permanent goal).
2. **What "suppliable" means for an activity, not a stock.** `grain` supplies cleanly (the Board
   `holds` it). "The noise of people" has no natural holder — O7 assumes every demand is drawable
   from a stock; a demand satisfied by *presence* rather than *consumption* has nowhere to point its
   supplier, so I cut it rather than guess.
3. **Whether a promoted individual needs a formal link back to the `magnitude` entity it came from.**
   Halma and the plaza crowd share no field. If two independent builds are meant to agree on *who*
   the crowd could plausibly promote, this needs one; if not, it's fine as-is — the contract doesn't
   say.

## 5. Under-specified reader obligations

**`channel.conceals: none`** — "renders present entities' `pursuing` and `hiding` to all
receivers" doesn't say *who counts as present*: same room, same city, same `within` tree at any
depth? Two builders could legally scope this very differently.

**`demand.unmet`** — "apply the effect after `onset_class`, and go on applying it" doesn't state
whether the onset clock resets on a partial feeding (one good week after twenty starved days) or
only on full satisfaction. Grelda's forty-day rule turns entirely on this.

**`indicator.reliability_class`** — "how often the sign misreports" gives no ladder and no
direction of error (false sign of danger vs. false sign of safety). For `La Ochenta y Tres`'s
knocking, a builder who assumes false-safe and one who assumes false-danger produce visibly
different suspense.
