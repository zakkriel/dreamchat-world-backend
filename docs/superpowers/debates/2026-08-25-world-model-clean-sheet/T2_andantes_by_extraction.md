# v2 encoding — Los Andantes (detailed tier)

## 1. Encoding

```json
{
  "world_model": "2",
  "world": { "name": "Los Andantes", "premise": "The only habitable ground is the backs of nine living creatures, each walking a fixed closed route. Where their routes cross is the only safe crossing between their cities, and it is known centuries ahead. Geography here is a calendar." },

  "excluded": [
    "No treatment, cure or intervention reverses an Andante's illness. None, ever.",
    "Andantes do not communicate, have no intention toward inhabitants, and do not answer requests, rituals, or offerings.",
    "No magic, gods, or inexplicable phenomena. Andantes are large animals.",
    "No migratory route can be altered by any means.",
    "No hidden dry land, lost continent, tenth Andante, or young growing Andante.",
    "A dead Andante sinks in 5–7 days, firmly, with no exception on record.",
    "Auscultation does not predict with exactness — its timelines derive from a single documented case; no scene should treat them as certain."
  ],

  "layers": [ { "name": "the walked world", "default": true } ],

  "vocabulary": {
    "movements": [ { "name": "walk", "pace_class": "steady" } ],
    "channels": [ { "name": "auscultation", "descriptor": "direct listening and palpation of an Andante's body",
      "emitted_by": "an Andante's own condition", "received_by": "a licensed auscultator with the horn",
      "latency_class": "immediate", "reach": "the point of contact", "decay": "never", "conceals": "none" } ],
    "conditions": [],
    "substances": []
  },

  "law": [
    { "name": "indirigibility", "enforced_by": "physics", "stated": "No method exists to alter an Andante's route or speed." },
    { "name": "the weight limit", "enforced_by": "physics",
      "stated": "Overload runs, in order: slower marching, then a limp, then joint injury, then systemic illness.",
      "forbids": { "subject": "any entity with matter", "act": "loading a sector past its tabulated limit without consequence" } },
    { "name": "transit outside Convergence", "enforced_by": "physics",
      "stated": "Crossing between Andantes off-Convergence needs a barge; survival is under 30%." },
    { "name": "no reversal of illness", "enforced_by": "physics", "stated": "No treatment, cure or intervention has ever reversed an Andante's decline." },
    { "name": "auscultators keep the Council off the record", "enforced_by": "office",
      "stated": "Auscultators do not sit on the city Council, by custom — in practice they run it.", "precedent": null },
    { "name": "no Andante meat", "enforced_by": "persons", "stated": "Unwritten, universally kept, origin undocumented.", "precedent": null }
  ],

  "entities": [
    { "name": "Tercera Hembra", "facets": ["extent", "matter", "motion", "demand"],
      "seen_as": "a walked continent, 74 km long, two cities on its back",
      "extent_class": "vast", "bulk_class": "immense", "integrity": "worsening",
      "trajectory": { "period_class": "long", "phase_at_start": "unknown" },
      "demands": [ { "substance": "?", "rate_class": "?" } ] },

    { "name": "Ossa", "facets": ["extent"], "within": "Tercera Hembra", "seen_as": "a city of 31,000, founded in 402",
      "extent_class": "large", "medium": "open air", "tension": "tense" },
    { "name": "Belna", "facets": ["extent"], "within": "Tercera Hembra", "seen_as": "Ossa's sister city, forty kilometres of dorsal road off, cold relations", "extent_class": "large" },

    { "name": "Cuenca district", "facets": ["extent", "matter"], "within": "Ossa", "seen_as": "the lumbar depression, 14,000 people",
      "extent_class": "medium", "bulk_class": "large", "integrity": "sound" },
    { "name": "Los Espiráculos", "facets": ["extent"], "within": "Ossa", "seen_as": "an uninhabited 400-metre exclusion ring around the blowholes", "extent_class": "medium" },

    { "name": "Del Vas", "facets": ["matter", "agency"], "within": "Ossa", "seen_as": "the Chief Auscultator of Ossa, fifty-eight",
      "capability": { "moves_by": ["walk"], "carry_class": "light" },
      "pursuing": [ { "horizon": "ongoing", "toward": "retire in 643 with no collapse on her record", "progress": "advanced" } ],
      "doing": "signing another monthly report", "hiding": "she found the arrhythmia nine months ago and never recorded it" },

    { "name": "Registrador Onn", "facets": ["matter", "agency"], "within": "Ossa", "seen_as": "Weight Guild registrar, forty-four",
      "pursuing": [ { "horizon": "ongoing", "toward": "a transfer to Quinta" } ],
      "hiding": "he licensed fourteen over-limit works in Cuenca between 637 and 640" },

    { "name": "Illa", "facets": ["matter", "agency"], "within": "Ossa", "seen_as": "a first-year apprentice, nineteen",
      "pursuing": [ { "horizon": "right_now", "toward": "graduate" } ],
      "hiding": "she repeated your reading independently, got the same result, and did not report it either" },

    { "name": "Auscultador Mayor Renn", "facets": ["matter", "agency"], "within": "Belna", "seen_as": "Belna's Chief Auscultator, forty-nine",
      "pursuing": [ { "horizon": "ongoing", "toward": "diagnostic independence for Belna" } ],
      "doing": "publishing a monthly report that has not matched Ossa's for four months" },

    { "name": "the College of Auscultators", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "sole authority over an Andante's declared state",
      "vulnerability": "a falsified report is not verifiable from outside the College" },
    { "name": "the Weight Guild", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "licence revenue and advancement", "vulnerability": "licences over the limit stand on at least three Andantes" },
    { "name": "the Dead Fleet", "facets": ["agency", "collective"], "legibility": "marked",
      "interest": "monopoly on emergency crossing", "vulnerability": "seventy percent losses; fourteen barges operable" },

    { "name": "the monthly report", "facets": ["record"], "within": "Ossa",
      "asserts": [ "\"en observación, sin hallazgos\" — unchanged for eight months running" ],
      "access": { "scope": "public", "cadence": "the first of each month" }, "authority": "Del Vas" },
    { "name": "the College archive", "facets": ["record"], "within": "Ossa",
      "asserts": [ "the true readings of the last nine months" ], "access": { "scope": "restricted", "held_by": "Del Vas" }, "authority": "the College of Auscultators" },
    { "name": "Illa's parallel register", "facets": ["record"], "within": "Ossa",
      "asserts": [ "an independent confirmation of your own readings" ], "access": { "scope": "private", "held_by": "Illa" }, "authority": null },
    { "name": "the Convergence Tables", "facets": ["record"], "within": "Ossa",
      "asserts": [ "the crossing calendar to the year 900", "Tercera Hembra's next window: 645, then 659" ],
      "access": { "scope": "public" }, "authority": "the Convergents" },

    { "name": "the auscultation horn", "facets": ["matter"], "within": "Ossa", "bulk_class": "small", "integrity": "sound",
      "confers": [ { "channel": "auscultation" } ] }
  ],

  "offices": [ { "name": "Chief Auscultator of Ossa", "held_by": "Del Vas", "of": "the College of Auscultators",
      "confers": [ { "act": "signing the monthly report" }, { "channel": "auscultation" } ], "succeeds_by": "designation" } ],

  "standing": [
    { "from": "Del Vas", "toward": "the Council of Ossa", "stance": "withholds what she knows, deliberately", "persistence": "until changed" },
    { "from": "Illa", "toward": "you", "stance": "waiting to see whether you report first", "persistence": "until changed" }
  ],

  "opposition": [
    { "between": ["the population of Cuenca (14,000)", "the population at risk on Tercera Hembra (48,000)"],
      "incompatible": "no available option — evict Cuenca, run the Fleet, wait, publish, or beg Primera for room — saves everyone",
      "stakes": "the Council's forced recommendation decides who is offered a route out" }
  ],

  "accumulators": [
    { "name": "Tercera Hembra's decline", "per": "world", "starts_at": "alert",
      "raised_by": [ { "event": "sustained arrhythmia" }, { "event": "Cuenca standing over its load limit since 639" } ],
      "thresholds": [
        { "at": "alert", "then": "4 to 10 years remain" },
        { "at": "critical", "then": "14 to 30 months remain" },
        { "at": "terminal", "then": "under 90 days remain" },
        { "at": "march-stopped", "then": "under 20 days remain", "irreversible": true } ] }
  ],

  "indicators": [
    { "of": "Tercera Hembra's decline", "shows_as": [ "deep pulse turning irregular", "daily march falling from 3.4 to 2.1 km", "clear discharge at the blowholes" ],
      "read_by": { "channel": "auscultation" }, "reliability_class": "poor",
      "note": "derived from a single prior fatality; the College knows the statistical base is thin" }
  ],

  "history": [
    { "name": "the-death-of-Octavo", "standing": "occurred",
      "what_happened": "An Andante sank in six days in the year 611; the city of Ruma, sixty thousand strong, evacuated three thousand one hundred.",
      "who": [ "Vieja Marda" ],
      "knowledge": [ { "holder": "Vieja Marda", "channel": "sight", "path": "direct", "believes": "I was there. I was fifty-one. I say little about it." } ] }
  ],

  "arrivals": [
    { "premise": "Fourth-year auscultation apprentice, twenty-seven, stationed in Ossa. Your routine reading came back critical. Illa's independent repeat matched it. Del Vas's report, published four days later, says the Andante is fine — the same thing it said for eight months before this one.",
      "seen_as": "an apprentice with a horn you have not yet earned the right to sign with", "place": "Ossa",
      "capability": { "moves_by": ["walk"], "carry_class": "moderate" } }
  ]
}
```

## 2. Facets, and where I hesitated

`Tercera Hembra`: `extent+matter+motion+demand`. This is the schema's own headline example
("a walking creature carrying cities") almost verbatim, so the fit is clean by construction — but I
still could not fill `demands[]` honestly: the brief never states what an Andante consumes to keep
walking or living, only *that* it is dying. I left it `"?"`.

`Cuenca district`: `extent+matter`, not `agency`. A district isn't itself a decider, but it *is*
physical enough to be over a load limit. Hesitated on whether an over-limit district needs `demand`
too (a load ceiling read as "requires staying under X or Y happens") — I left it off because
`demand` is written for *consumption*, and a weight ceiling is a *capacity comparison*, not
something the district draws down. No facet fits a bare ceiling well.

`Records vs. offices`: I gave `Del Vas` the office and the report a separate `record` entity rather
than folding the report into her `agency`. Defensible either way — a report an office-holder signs
could plausibly be a key *on* the office (`offices[].confers`) rather than its own entity. I kept
them separate because the report outlives any one Chief Auscultator and is read independently of
her.

## 3. Breaks

**A field that is simultaneously plot-critical public knowledge and an engine trigger.** *"Cuenca...
41.300 t... SOBRE el límite del sector desde 639."* This is the sharpest break in this world, and
it's in the new numbers rule itself. The rule sorts fields into "player-reads-it" (write the number)
and "engine-computes-on-it" (keep it a class) — but Cuenca's tonnage is *both at once*: a
character can look it up in the Registro de Peso (numbers may be written), and it is exactly the
figure the overload law (§`law`, "the weight limit") has to compare against a threshold to fire its
consequence (numbers forbidden). I wrote `41,300` in `seen_as` prose as a compromise and left the
entity with no structured `carga`/`limit` pair at all — there is nowhere to put a number that is
legitimately both.

**Two independently-authored timelines for the same fact.** `motion.trajectory.period_class` is
meant to be engine-resolved (the way `extent_class` becomes metres); the Convergence Tables record
authors *exact years* (645, 659) for specific crossings. If the engine derives the calendar from
the class, the authored table is either redundant or can silently contradict it; if the calendar is
authored independently, the class computes nothing at all for this quantity. The schema doesn't say
which; I encoded both, because the brief needs the exact dates as canon (numbers a player reads),
which no class resolves to precisely.

**A ceiling, as opposed to a demand or a threshold.** Reported in §2. **(iii) wrong-shaped.**
Weight capacity is neither consumption (`demand`) nor decline (`accumulators`); it's a static
comparison a law checks against, and no facet or section states "this entity has a limit another
number is measured against."

Against v1: this world is where `record` and the new numbers rule are the clear wins — v1 could not
have held the Registro de Peso, the monthly report, or Cuenca's tonnage at all (v1 forbade every
number, full stop), so this brief would have broken v1 far worse, on almost every table in it.

## 4. Ambiguity report

1. **A number that is both read and computed on** (§3) — the sharpest one. Disambiguator: a third
   category, a number that's canon-fact *and* feeds a named comparison, distinct from pure flavour.
2. **Does `motion.trajectory` generate a calendar, or does an authored `record` own it independently?**
   (§3). Disambiguator: state whether class-resolution ever produces dates, or only rates/spans.
3. **Ceilings vs. demands** (§2/§3). Disambiguator: a `limit` key, symmetric to `demand`'s draw,
   naming a static capacity a comparison reads rather than something consumed.
4. **Report-as-entity vs. report-as-office-key.** (§2). Both readings are legal under the section
   list as given; nothing forces one.

## 5. Convergence check

No new top-level section. Every break here is a missing *key* (a ceiling, a dual-purpose number) on
existing facets, not a missing kind of thing. That's a second, independent world converging on the
same verdict as Grelda — worth weighting.
