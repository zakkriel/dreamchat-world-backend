# v2 encoding — Las Casas de Grelda (detailed tier)

## 1. Encoding

```json
{
  "world_model": "2",
  "world": { "name": "Grelda", "premise": "The houses are alive. They grow from cuttings, eat grain, fire-warmth and the noise of people, grow and shrink rooms unasked, and close forever after forty days unfed." },

  "excluded": [
    "Houses do not speak in words — only knock, open, close, grow, shrink. No dialogue, inner voice, or explained intention for a house, ever.",
    "No magic, spells, wizards, or schools of it. Craft, ear, custom, strange biology only.",
    "No way to force a deal. Every attempt fails, and the failure is always the scene.",
    "No malicious or scheming houses — they are slow with preferences; almost all harm they cause is indifference, not cruelty.",
    "A closed house never reopens, by any means, ever. One known anomaly exists and must stay unique.",
    "No technical fix for the unhoused — the only path is convincing house by house."
  ],

  "layers": [ { "name": "waking", "default": true } ],

  "vocabulary": {
    "movements": [ { "name": "walk", "pace_class": "steady" } ],
    "channels": [
      { "name": "underground speech", "descriptor": "knock-patterns carried through the ground between houses and toward a listener",
        "emitted_by": "any entity with matter, struck correctly", "received_by": "houses; a person with the ear or the staff",
        "latency_class": "very slow", "reach": "spreads district to district over weeks", "decay": "never", "conceals": "none" }
    ],
    "conditions": [
      { "name": "lost the ear", "alters": [ { "channel": "underground speech", "effect": "hinder", "class": "severe" } ] }
    ],
    "substances": [ { "name": "grain" }, { "name": "fire-warmth" }, { "name": "the noise of people" } ]
  },

  "law": [
    { "name": "a house cannot be forced", "enforced_by": "physics",
      "stated": "No money, law, tool or violence has ever produced a deal a house did not choose.",
      "forbids": { "subject": "any entity with agency", "act": "compelling a house to accept an occupant" } },
    { "name": "forty days unfed closes a house forever", "enforced_by": "physics",
      "stated": "A house that goes forty days without grain, warmth or noise seals every door, window and flue, with whatever is inside, and never reopens.", "precedent": null },
    { "name": "one courtesy night, never two", "enforced_by": "physics",
      "stated": "A first night in a house with no deal is tolerated as courtesy. A second is felt as intrusion, and the house answers by shutting its inner doors.", "precedent": null },
    { "name": "a cutting roots only near its stock", "enforced_by": "physics",
      "stated": "Transplant a cutting between cities and it dies, and sickens the mother it was cut from.", "precedent": null },
    { "name": "no planting without the Ration Board's leave", "enforced_by": "office",
      "stated": "Planting requires the Board's authorisation.", "binds": [ "anyone" ],
      "precedent": "half the upper districts broke this eighty years ago and it is no longer discussed" }
  ],

  "entities": [
    { "name": "Cuesta Menor", "facets": ["extent"], "within": "Grelda", "extent_class": "large", "medium": "waking air", "tension": "calm" },

    { "name": "La Ochenta y Tres", "facets": ["extent", "matter", "holding", "demand", "agency"],
      "within": "Cuesta Menor", "seen_as": "a warm three-storey house against the upper wall, with a fourth floor that comes and goes",
      "bulk_class": "immense", "integrity": "sound", "extent_class": "medium", "medium": "waking air", "tension": "tense",
      "capacity_class": "large", "holds": [],
      "demands": [ { "substance": "grain", "rate_class": "minimal" }, { "substance": "the noise of people", "rate_class": "none-received" },
        "unmet": { "effect": "law:forty days unfed closes a house forever", "onset_class": "long" } ],
      "doing": "knocking the ground every night, earlier each time",
      "hiding": "why it opened for someone who already had a roof, after a hundred and forty refusals" },

    { "name": "Ordo Bes", "facets": ["matter", "agency"], "within": "Cuesta Menor", "seen_as": "a sour, exacting man with thirty years at the trade",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" }, "conditions": ["lost the ear"],
      "disposition": [ { "trait": "exacting", "strength": "strong", "manner": "explains everything twice, except when he doesn't" } ],
      "doing": "refusing to explain why you must not go in",
      "pursuing": [ { "horizon": "long_standing", "toward": "retire an apprentice who actually hears", "progress": "advanced" } ],
      "hiding": "he lost the ear four years ago in a winter flood, works on craft and memory since, and his results never dropped — which frightens him about everything he did before" },

    { "name": "Perla Anís", "facets": ["matter", "agency"], "within": "Cuesta Menor", "seen_as": "a cordial, quick official with a folder",
      "capability": { "moves_by": ["walk"], "carry_class": "light" },
      "pursuing": [ { "horizon": "ongoing", "toward": "a transfer to the inner ring", "progress": "advanced" } ],
      "doing": "asking cordial questions about the Eighty-Three",
      "hiding": "she wrote the lower-ring ration cut herself and keeps a private tally of the thirty-one houses it has closed" },

    { "name": "the Guild of Brokers", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "keep sole rights to negotiate with houses",
      "vulnerability": "much of the craft is theatre; if learning it in a year were ever proven, the Guild ends" },

    { "name": "the Ration Board", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "budget stability and no inquiry", "vulnerability": "its own register proves the twelve-year cut closed thirty-one houses" },

    { "name": "the Unhoused", "facets": ["magnitude", "agency", "collective"], "magnitude_class": "many",
      "legibility": "marked", "interest": "recognition that the shortage is a matter of houses' judgement, not of supply",
      "vulnerability": "their only possible strategy is occupation, and a house that never accepted you will not let you leave either" },

    { "name": "Halma Ruiz", "facets": ["matter", "agency"], "within": "Grelda", "seen_as": "the only person in Grelda raised never having slept under a roof",
      "capability": { "moves_by": ["walk"], "carry_class": "light" },
      "pursuing": [ { "horizon": "right_now", "toward": "occupy one of the Old Houses, knowing it will fail" } ] },

    { "name": "the Ration Register", "facets": ["record"], "within": "Cuesta Menor",
      "asserts": [ "what every house eats, how much, since when" ], "access": { "scope": "public" }, "authority": "the Ration Board" },

    { "name": "the Guild Register", "facets": ["record"], "within": "Cuesta Menor",
      "asserts": [ "every deal request and its result", "an eighteen-month gap sixty years ago with no entries at all" ],
      "access": { "scope": "public" }, "authority": "the Guild of Brokers" },

    { "name": "the broker's staff", "facets": ["matter"], "within": "Cuesta Menor",
      "bulk_class": "small", "integrity": "sound",
      "confers": [ { "channel": "underground speech" } ] }
  ],

  "offices": [
    { "name": "Voice of the Unhoused", "held_by": "Halma Ruiz", "of": "the Unhoused",
      "succeeds_by": "rotation among three", "confers": [] }
  ],

  "standing": [
    { "from": "Ordo Bes", "toward": "you", "stance": "will not explain, which from him is itself an explanation", "since": null, "persistence": "until changed" }
  ],

  "opposition": [
    { "between": ["the Ration Board", "the Unhoused"],
      "incompatible": "the Board needs the ration debate to keep going nowhere; the Unhoused need it settled", "stakes": "who eats, and which districts empty out" }
  ],

  "accumulators": [
    { "name": "houses closed since the cut", "per": "world", "starts_at": "none",
      "raised_by": [ { "event": "a district-twelve-ring house going forty days unfed" } ],
      "thresholds": [ { "at": "moderate", "then": "Perla Anís's private tally becomes hard to explain administratively" } ] }
  ],

  "indicators": [
    { "of": "what the Eighty-Three wants", "shows_as": [ "the knocking rhythm, earlier every night", "warmer than its diet explains" ],
      "read_by": { "channel": "underground speech" }, "reliability_class": "poor" }
  ],

  "epochs": [
    { "name": "the eighteen-month gap", "differed": [ { "subject": "the Eighty-Three", "topic": "ration", "then": "cut, at exactly this span" } ],
      "surviving_traces": [ "the gap in the Guild Register", "Tomás's habit of stopping at the foot of the Cuesta and never climbing" ] }
  ],

  "history": [
    { "name": "the-one-who-came-out", "standing": "occurred",
      "what_happened": "A house closed with five people inside. One of them, Tomás, was later found outside it.",
      "where": "Grelda", "who": [ "Tomás" ],
      "knowledge": [ { "holder": "Tomás", "channel": "sight", "path": "direct",
        "believes": "I got out. I have never told anyone how, because I don't fully know." } ] }
  ],

  "arrivals": [
    { "premise": "Broker's apprentice, three years with Ordo Bes. Two days ago the Eighty-Three's door opened for you — no knock, no deal — after sixty years and a hundred and forty refusals.",
      "seen_as": "a young apprentice with a broker's staff you haven't earned the right to carve", "place": "Cuesta Menor",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" } }
  ]
}
```

## 2. Facets, and where I hesitated

`La Ochenta y Tres`: `extent+matter+holding+demand+agency`. Hesitated hardest here. `agency` grants
`disposition[]`/`pursuing[]` — authored *interior* — and the brief forbids a house any "intención
explicada" (§11). I gave it `agency` only for `doing`/`hiding` (observable behaviour, a real
secret) and left `disposition`/`pursuing` empty on purpose. The facet bundle doesn't let you take
"decides and acts" without also getting slots for stated inner motive; nothing marks that split.

`the Unhoused`: `magnitude+agency+collective`, all three at once. The schema's own worked example
splits an anonymous mass (`magnitude+agency`, "the Pactless") from an organised body
(`agency+collective`, "the Measure") as *separate* entities. Los Sin Trato are genuinely both — an
uncounted mass and a body with named rotating spokespeople and a demand — and I couldn't tell
whether that's meant to be one entity with three facets or two linked entities. I took one; I'm not
confident.

`Las Casas Viejas`: I dropped it as an authored entity entirely rather than fake a `collective` for
eleven buildings each of which is really its own `extent+matter+holding+demand+agency` house. There
is no `members[]` key on the `collective` facet — nothing says how a collective enumerates entities
that are themselves fully-faceted, only prose `interest`/`vulnerability`.

## 3. Breaks

**No way to mark a demand-satisfier as scarce vs. free.** *"Grano... es lo que reparte la Junta y lo
que se mide... Calor de fuego, que es gratis y por eso no se discute."* Section: `demand`'s
`substances`. **(iii) wrong-shaped.** The brief's whole politics (*"la política municipal no gira
alrededor de impuestos sino alrededor del grano"*) hangs on grain being rationed and warmth not
being — the schema has one undifferentiated `substances` list with no scarcity flag.

**Growth as a reward, not a deficit.** *"Una casa contenta suma un cuarto cada tanto."* Section:
`demands[].unmet`. **(i) inexpressible.** `unmet` only names a *deficit* consequence; there is no
surplus-consequence symmetric to it, so I could not encode growth at all, only closure.

**A collective's members, when they are themselves complex entities.** Reported in §2. **(ii) inert
prose** at best — you can say "eleven Old Houses" in a `descriptor`, you cannot reference them.

This is a genuine improvement over v1 in three places: `record` (the Ration and Guild registers now
have a real facet, where v1 left evidentiary objects as bare `things[]`); `confers`
(the staff → channel link is exactly what v1 could only state as inert prose); `epochs` plus
`history[].standing` together hold the sixty-year gap and its one true anomaly cleanly, where v1 had
no dispute-status at all. The two breaks above are new, not relocated old ones.

## 4. Ambiguity report

1. **Does `agency` imply authored interiority is *available*, or *required*?** (§2). Disambiguator:
   split `agency` into "decides" (behaviour only) and a separate optional interiority grant.
2. **Can `magnitude` and `collective` coexist on one entity, or must an organised mass always
   split into two?** (§2). Disambiguator: a worked example showing both, or an explicit rule.
3. **`demands[]`'s array vs. object shape.** In my encoding, `demands` is a list but I also wrote a
   bare `"unmet": {...}` sibling key outside the list because I could not tell whether `unmet`
   belongs to *one* demand entry or to the demand facet as a whole covering all of them. The
   section list gives one `unmet{}` per facet description, singular, which reads as facet-level —
   but a house eating three different things plausibly fails at each independently.
4. **`record.asserts[]` — free prose claims, or references into `history[].knowledge[]`?** I took
   free prose. If it's meant to reference specific beliefs it could correct, that is a materially
   more powerful (and differently-shaped) mechanism than the one I encoded.

## 5. Convergence check

No new top-level section needed. Everything in this brief found a section; the failures were all
inside sections (missing keys — `members[]`, a scarcity flag, a surplus effect), never a missing
kind of section.

## 6. Basic-tier diff

Encoding the basic brief against the same shape: `world`, `excluded` (shorter — only the "no
forcing" and "closed forever" absolutes are stated, not all six), `law` (the same four hard rules,
same shape, fewer words), one `entities` slice (Cuesta Menor, La Ochenta y Tres with `doing`/`hiding`
but no `demands`/`indicators` detail because the brief gives none), `history` (same anomaly, same
`standing: "occurred"`), one `arrivals` entry, near-identical to the detailed one.

**It is a clean subset, not a different document.** Every section the basic tier fills is filled
with the *same shape* as the detailed tier, just fewer and thinner entries — no field the basic
brief implies goes somewhere the detailed brief's version of that field doesn't also go. What
disappears going from detailed to basic is exactly the material that revealed the breaks above
(`offices`, `standing`, `opposition`, `accumulators`, `epochs` all sit empty at the basic tier) —
which is the good outcome: depth is what's missing, not a different kind of document.
