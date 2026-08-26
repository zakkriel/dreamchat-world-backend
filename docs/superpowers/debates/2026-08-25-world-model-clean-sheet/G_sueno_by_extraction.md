# v4 genesis — Sueño Común (from the 400-word tier only)

Provenance is attached per authored object (entity, law, history entry), not per leaf field, for
space; every object's `source` is stated or bottoms out in a quoted line below.

## 1. Emission

```json
{
  "world_model": "4",
  "world": { "source": "stated", "name": "Orbe",
    "premise": "Everyone asleep in the same district enters one shared dream, appearing as themselves, wanting visibly. A public office transcribes it every morning." },

  "excluded": [
    { "source": "stated", "text": "No disguise, no mask — nobody's appearance in the dream differs from themselves." },
    { "source": "stated", "text": "Outside Orbe, none of this happens; a sleeper outside the city dreams alone." }
  ],

  "vocabulary": {
    "channels": [ { "name": "the shared dream", "source": "stated",
      "descriptor": "the one dream a whole district enters together, every night",
      "emitted_by": "any agency entity asleep and registered to the district",
      "received_by": "every other entity asleep in the same dream",
      "latency_class": "immediate", "reach": "the whole registered district, at once",
      "decay": { "class": "brief", "exemplar": "three nights" },
      "conceals": "none" } ],
    "conditions": [ { "name": "solitary", "source": "stated",
      "alters": [ { "channel": "the shared dream", "effect": "immune", "class": "total" },
                  { "channel": "the shared dream", "effect": "grant", "class": "total" } ] } ],
    "substances": []
  },

  "law": [
    { "name": "one district, one dream, every night", "source": "stated", "enforced_by": "physics",
      "stated": "Everyone asleep in the same district enters the identical dream, every night, without exception." },
    { "name": "nobody hides inside the dream", "source": "stated", "enforced_by": "physics",
      "stated": "No disguise, no mask, no concealment inside the shared dream — and what a sleeper wants renders visibly too." },
    { "name": "foreign sleep", "source": "stated", "enforced_by": "office",
      "stated": "Sleeping registered to a district other than your own is a crime.",
      "forbids": { "subject": "any entity with agency", "act": "sleeping in a district other than the one their card names" },
      "binds": [ "anyone" ] },
    { "name": "no asking a solitary what they saw", "source": "stated", "enforced_by": "persons",
      "stated": "Nobody asks a registered solitary what they witnessed." },
    { "name": "an unshareable marriage", "source": { "inferred_from": [ "marriages are made between different districts, by custom", "sleeping outside your registered district is a crime" ] },
      "enforced_by": "physics",
      "stated": "A married couple from two districts can never lawfully share a single night's dream — each sleeps registered to their own." }
  ],

  "entities": [
    { "name": "Orbe", "source": "stated", "facets": ["extent"], "extent_class": "vast", "medium": "waking air", "tension": "normal" },
    { "name": "Barrio Doce", "source": "stated", "facets": ["extent"], "within": "Orbe", "extent_class": "large", "medium": "waking air", "tension": "tense" },
    { "name": "the plaza of the Twelve", "source": "stated", "facets": ["extent"], "within": "Barrio Doce", "extent_class": "small", "medium": "waking air", "tension": "normal" },

    { "name": "the shared dream of the Twelve", "source": { "inferred_from": [ "everyone who sleeps in the same district enters the same dream", "forty thousand people saw you standing beside the body" ] },
      "facets": ["extent"], "extent_class": "vast", "medium": "the shared dream", "tension": "tense" },
    { "name": "the unrecognised place", "source": "stated", "facets": ["extent"], "within": "the shared dream of the Twelve",
      "seen_as": "a scene nobody in the Twelve recognises as anywhere in the district", "extent_class": "medium", "medium": "the shared dream", "tension": "frantic" },
    { "name": "the body", "source": "stated", "facets": ["matter"], "within": "the unrecognised place",
      "seen_as": "someone nobody in the district can name", "bulk_class": "moderate", "integrity": "sound" },

    { "name": "falling asleep in Barrio Doce", "source": { "inferred_from": [ "everyone who sleeps in the same district enters the same dream" ] },
      "facets": ["passage"], "connects": [ "Barrio Doce", "the shared dream of the Twelve" ],
      "admits": [ { "act": "falling asleep" } ] },

    { "name": "Rem Salas", "source": "stated", "facets": ["matter", "agency"], "within": "the plaza of the Twelve",
      "bulk_class": "moderate", "seen_as": "the Twelve's transcriber, twelve years in the post",
      "pursuing": [ { "horizon": "right_now", "toward": "stand behind last night's volume, published since nine" } ] },

    { "name": "Onel", "source": "stated", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "seen_as": "registered solitary, number four of the Twelve", "conditions": [ "solitary" ],
      "pursuing": [ { "horizon": "long_standing", "toward": "not be asked, and not have to refuse" } ],
      "hiding": "what he saw in last night's dream, which nobody by custom will ask him" },

    { "name": "Vira Cor", "source": "stated", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "seen_as": "an insomniac eleven nights in, and it shows",
      "pursuing": [ { "horizon": "right_now", "toward": "make it through one more day without projecting in public" } ] },

    { "name": "Inspector Bald", "source": "stated", "facets": ["matter", "agency"], "within": "Barrio Doce",
      "bulk_class": "moderate", "seen_as": "the Office of Vigil's investigator for foreign sleep",
      "pursuing": [ { "horizon": "ongoing", "toward": "find who slept where they shouldn't have, this time" } ] },

    { "name": "the district of Barrio Doce", "source": "stated", "facets": ["magnitude", "agency"], "within": "Barrio Doce",
      "magnitude_class": { "class": "vast", "exemplar": "forty thousand" }, "seen_as": "everyone who dreamed the same thing you did last night",
      "pursuing": [ { "horizon": "right_now", "toward": "find out who you are and what you did" } ] },

    { "name": "the Registry of Solitaries", "source": "stated", "facets": ["record"], "within": "Orbe",
      "asserts": [ "eleven names, citywide" ], "access": { "scope": "reserved" }, "authority": "the Office of Vigil" },

    { "name": "last night's volume, the Twelve", "source": "stated", "facets": ["record"], "within": "the plaza of the Twelve",
      "asserts": [ "the district dreamed a killing, in a place nobody recognises, and you were standing beside the body" ],
      "access": { "scope": "public", "cadence": "since nine this morning" }, "authority": "Rem Salas" },

    { "name": "your district card", "source": "stated", "facets": ["matter"], "within": "Barrio Doce",
      "bulk_class": "small", "integrity": "sound" }
  ],

  "offices": [
    { "name": "Transcriber of the Twelve", "source": "stated", "held_by": "Rem Salas", "of": "the Office of Vigil",
      "confers": [ { "act": "writing what the district dreamed, into the public record" } ], "succeeds_by": "appointment" } ,
    { "name": "the Office of Vigil (Twelve)", "source": "stated", "held_by": null, "confers": [ { "act": "investigating foreign sleep" } ], "succeeds_by": "appointment" }
  ],

  "opposition": [
    { "source": { "inferred_from": [ "Onel is a professional witness", "nobody asks a solitary what they saw", "Inspector Bald investigates" ] },
      "between": [ "Inspector Bald", "the custom of not asking a solitary" ],
      "incompatible": "an investigation that needs the one total witness cannot lawfully ask him anything",
      "stakes": "whether the killing can be investigated by custom at all" } ],

  "indicators": [
    { "source": { "inferred_from": [ "it hasn't happened yet", "nobody recognises the place" ] },
      "of": "why-the-Twelve-dreamed-it", "shows_as": [ "no name in the district matches the body", "the place matches no address in the Twelve", "your own memory of it, still holding, has not yet begun to fray" ],
      "read_by": { "channel": "the shared dream" }, "reliability_class": "poor" } ],

  "history": [
    { "name": "why-the-Twelve-dreamed-it", "source": "stated", "standing": "disputed",
      "what_happened": "Last night the district of Barrio Doce dreamed a killing. It has not happened. Nobody recognises the place or the body.",
      "where": "the shared dream of the Twelve", "who": [ "you", "Rem Salas" ],
      "knowledge": [
        { "holder": "you", "channel": "the shared dream", "path": "direct", "believes": "I don't remember doing anything. I remember standing there." },
        { "holder": "Rem Salas", "channel": "the shared dream", "path": "direct", "believes": "It happened exactly as I wrote it — I stand behind every volume I've published in twelve years." } ] } ],

  "arrivals": [
    { "source": "stated",
      "premise": "Barrio Doce, thirty-one, no record. Last night the district dreamed a killing that hasn't happened, in a place nobody recognises. Forty thousand people saw you standing beside the body. Rem Salas's transcription has been public since nine, and you haven't read it yet.",
      "seen_as": "yourself, exactly as you are", "place": "Barrio Doce", "capability": { "moves_by": ["walk"], "carry_class": "moderate" } } ]
}
```

## 2. Self-check

**S1** every reachable extent has content — Barrio Doce (people/office), the shared dream (the
unrecognised place, the body), the plaza (record, Rem). Pass.
**S2** every reachable `agency` entity wants something actionable — all five do. Pass.
**S3** every name resolves — checked; `your district card` and `the Office of Vigil` added
specifically to close references the brief makes in passing. Pass.
**S4** every `passage` leads somewhere authored — one passage, one destination. Pass.
**S5** opposition and no accumulators exist — the one opposition has a real outcome (investigate or
don't); no accumulator was authored (see §4, item 9). Pass.
**S6** no implied-but-missing place/office/practice — "Oficina de Vigilia," "Archivo central,"
"Registro de solitarios," "carné de barrio" are all mentioned in the brief and all now exist. Pass.

**O1–O11:** O1 three extents, one passage — pass. O2–O4 five `agency` entities, all with
`pursuing`, `hiding` on Onel — pass. O5 one opposition — pass. O6 every `matter`-only entity has
`bulk_class` (F4: not required on `agency` entities that already carry it by convention here, so I
added it anyway for consistency) — pass. O7 no `demand` was authored — vacuously pass. O8 no
`accumulator` was authored — vacuously pass. O9 the one indicator points at the disputed history
entry's `standing` — pass. O10 one `arrivals` — pass. O11 `excluded[]` present, two entries —
pass.

**R1–R13:** all clear. R12 exercised on purpose (§why-the-Twelve-dreamed-it, two holders who
genuinely differ). R13: every `inferred_from` chain bottoms in a quoted line from the brief — none
references another inference.

## 3. Provenance in full

Every entity/law/history object above carries `source`; five are `inferred_from` chains
(the shared dream itself, the passage into it, the unshareable marriage, the opposition, the
indicator) — each cites the exact stated line(s) it follows from, none references another
inference, satisfying R13 by construction.

## 4. The ten most consequential inventions

1. **The shared dream as its own `extent` entity**, not a state flag on the district — forced by
   "entran en el mismo sueño" (you *enter* it) and "parado al lado del cuerpo" (standing beside a
   body — a position implies a place). Generic rejected: treating the dream as a narration effect
   with no addressable geography.
2. **`falling asleep` as the passage's `admits` predicate**, not a movement — forced by there being
   no stated gait into the dream at all. Generic rejected: a door, a threshold, a ritual.
3. **`conceals: none` on the shared-dream channel**, world-default — direct transcription of "no
   hay disfraz, no hay máscara" plus "lo que querés se ve," not an invention so much as a
   translation, but the *placement* (on the channel, not on each person) was a choice against
   repeating a privacy flag on every entity.
4. **Solitary as a `condition`, not a facet or a species** — forced by "existen los solitarios,"
   phrased as a state some people are in, registered, not a kind of being. Generic rejected: a
   separate entity-kind for "seers."
5. **The unshareable marriage** (§law) — forced by combining two stated soft/hard rules that were
   never stated together. Generic rejected: leaving the custom unexplained, or inventing a
   sentimental reason the brief doesn't give.
6. **Onel's `hiding`** — forced by rule 10 existing at all: a rule against asking implies there is
   an answer worth withholding. Generic rejected: no hiding at all, since the brief never states
   what he knows.
7. **The Bald/custom `opposition`** — forced by "testigo profesional" plus rule 10 sitting in overt
   tension. Generic rejected: a generic "the law and the truth conflict" opposition with no named
   mechanism.
8. **Rem Salas's stated confidence as the second `disputed`-history holder**, not a second witness
   with different facts — the brief gives me her role and tenure, nothing about what she saw, so I
   grounded her belief in her *institutional* stance rather than inventing content she has no
   textual basis for.
9. **No accumulator authored** for Vira's insomnia — I considered one (nights-without-sleep →
   threshold) and cut it: rule 6's effect is fully stated as a *consequence*, and nothing in the
   brief needs the exact count tracked as a rising number for this document to be sufficient at 400
   words. Sufficiency, not completeness.
10. **`your district card` and the central `Office of Vigil`** authored as bare entities with almost
    no properties — needed only so that "carné de barrio" and "Oficina de Vigilia" (mentioned,
    load-bearing to the crime of foreign sleep) resolve under S3/S6, not because they earn more.

## 5. The pull

**A population threshold for the dream's existence.** The schema's own worked illustration for this
world gestures at a demand-for-sleepers mechanic. The brief gives "todas las noches, sin
excepción" — unconditional, not threshold-gated — so I left it out. This was the strongest pull,
because a conditional trigger is the generic shape and the brief's actual answer is starker: it
always happens, to everyone asleep, no exceptions, no gate.

**A "no magic / no supernatural" exclusion.** Nearly every world in this test series gets one; I
have zero textual basis for it in 400 words, so `excluded[]` stays at two entries, both stated, not
padded to look thorough.

**A reason for the cross-district marriage custom.** "Todos entienden por qué" begs for an
invented rationale — I left it unexplained, exactly as unexplained as the brief leaves it.

## 6. Load-bearing set

Per §2.1: referenced elsewhere, or living in `law`/`opposition`/`offices`/`excluded`/`history`.
Load-bearing: the shared-dream extent (referenced by the passage, the history entry, the
indicator); the passage itself (`law`-adjacent, load-bearing by construction); the unshareable
marriage law; the Bald/custom opposition; both offices; both `excluded` entries; the disputed
history entry and its two holders; the "solitary" condition (referenced by Onel and by the
channel's `alters`). Texture, correctly hidden from a confirmation screen: Rem's twelve years, Vira
Cor's specific descriptor, the plaza's `extent_class`, the body's `bulk_class`.
