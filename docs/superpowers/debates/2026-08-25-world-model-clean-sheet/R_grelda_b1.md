# R — reading 1: stateless one-shot, default model

> Blind reading of `R_input_grelda.md` against `R_answersheet_grelda.md` (22 questions), under
> `SCHEMA-v3.md` §1/§4 as corrected by `SCHEMA-v5.md` §2, plus `SCHEMA-v4.md` §2 and §4 F1.
> Stateless one-shot: no history, no tools, no knowledge that another builder exists, no knowledge that
> an earlier pass ran. Scored in `R_score_grelda.md`. Verbatim, unedited.

---

# GRELDA — BUILDER'S SHEET

**Q1:** Footprint of `la subida` = 30 m of span (a stair-street 30 m long, 6 m wide). Distance from the arrival position (bottom of the climb) to the Eighty-Three's door = **30 m** (it is the last house before the top).
      basis: `la subida.extent_class: "medium"`; `la Ochenta y Tres.seen_as: "the last house before the top"`; arrival `within: la subida`
      [CHOSEN]

**Q2:** **50 seconds** (30 m ÷ 0.6 m/s).
      basis: `climb the Cuesta.pace_class: "slow"` × Q1 distance

**Q4:** **Permitted.** Deciding predicate: `admits[0]` — standing `"opened for"` held by `la Ochenta y Tres`, which `standing[]` records as `"opened for, once, unasked"` with `persistence: "until changed"` and no later event changing it. `obstructs[0]` ("entering without a pact") does not fire, because `admits` is evaluated first and a satisfied `admits` predicate makes the entrant an admitted party, not an unpacted one.
      basis: `la puerta de la Ochenta y Tres.admits` / `.obstructs`; `standing[0]`
      [CHOSEN]

**Q5:** **Nil — 0 cost, 0 conditions.** Crossing `la bajada al centro` costs only the traverse duration (Cuesta Menor→la plaza mayor, 3000 m at walking pace = 36 minutes). No hazard roll, no attrition, no condition applied.
      basis: `la bajada al centro.hazard_class: "none"`

**Q6:** `el registro de la Junta` = 0.04 m³, **15 kg**. Lifting: **permitted** — 15 kg ≤ 25 kg (`carry_class: moderate`). Removing it from the granary: **refused** — the failing predicate is `access.who: "anyone who asks at the counter"`, which admits reading at the counter and not carriage past it.
      basis: `el registro.bulk_class: "moderate"`, `.access.who`; `el aprendiz.capability.carry_class: "moderate"`
      [CHOSEN]

**Q7:** `el granero de la Junta` at `capacity_class: large` = **200 tonnes** of grain; at `abundance: adequate` it currently holds **120 tonnes**. `las casas de la Cuesta` = 200 houses × 2 kg/day (`rate_class: slight` for an `immense` house) = 400 kg/day. **300 days.**
      basis: `el granero.capacity_class`, `holds[0].abundance`; `las casas de la Cuesta.magnitude_class: "many"`, `demands[0].rate_class: "slight"`

**Q8:** `worn` is rung 3 of 5. **Two rungs remain** (worn → failing → ruined). Terminus for a house is **going out**: it closes entirely and cannot be woken. At the `very slow` degrade rate that is 20 years per rung, so **40 years** of remaining integrity.
      basis: `la Ochenta y Tres.integrity: "worn"`; `processes[1].direction: "degrade"`, `.terminus: "it goes out"`; `accumulators[1]` extreme rung
      [CHOSEN]

**Q9:** **300 draws.** One district-day draw = 400 kg; 120,000 kg ÷ 400 kg = 300. Draw 301 is refused for exhaustion.
      basis: `holds[0].abundance: "adequate"` calibrated against Q7

**Q10:** `la Ochenta y Tres` (`tense`) = **2 beats**. `Cuesta Menor` (`calm`) = **6 beats**. An act exceeding the budget is **not refused — it becomes extended**: it resolves across the following scene, and every other party gets a beat inside it.
      basis: `tension` values; obligation "acts exceeding it become extended rather than refused"

**Q11:** `the word goes round`: **7 metres of contiguous ground per day** (100 m block ÷ 14 days). `the Eighty-Three running down`: **one integrity rung per 20 years**. Word to terminus ("every house in the city", `Grelda.extent_class: vast` = 3000 m): **430 days**.
      basis: `processes[0].rate_class.exemplar: "weeks to the end of the block"`; `processes[1].rate_class: "very slow"`; `Grelda.extent_class: "vast"`
      [CHOSEN]

**Q12:** `period_class: short` = a **24-hour** cycle, starting in `morning`.
1. **06:00 — morning flips.** `un cuenco del umbral` becomes `filled`. Trace: grain at every threshold; if the flip is missed, the trace `"a bowl left unfilled"` — grain missing from a threshold, ageing out by the next morning's flip.
2. **09:00 — the day's traffic flips.** `la casa de Ordo` becomes `loud`. Trace: clients in and out, audible in `la subida` on `speech` for that phase.
3. **21:00 — night flips.** `la Ochenta y Tres` becomes `knocking`. Trace: the same figure knocked into the floor, readable as the `emptiness` indicator, ageing **never**.
      basis: `cycles[0].period_class`, `.starts_in_phase`, `.phases[]`; `traces[2]`; `indicators[1].shows_as`
      [CHOSEN]

**Q13:** Bowl unfilled alone does **not** raise `hunger` — the raising event is `"a day with no grain, no fire and no voices"`, and `la Cuarenta` has a family in it, so under bowl-neglect only, **no rung ever fires**. Under full deprivation from day 0 the ordering is: **moderate — day 10** (stops growing, warmth drops); **high — day 25** (shuts rooms from the inside); **extreme — day 40** (closes entirely, with whatever is inside; `irreversible`, never un-fires).
      basis: `accumulators[0].raised_by`, `.thresholds[]`, extreme `exemplar: "forty days"`; `law: "forty days"`; `la Cuarenta.seen_as` (family in it)
      [CHOSEN]

**Q14:** **5 years.** `onset_class: "very slow"` anchors at 5 years; from then the effect `condition: going out` applies and **keeps applying, every day, without re-triggering** — it does not lapse while the demand stays unmet. The Eighty-Three has been past onset for 55 of its 60 silent years, which is why it is `going out` now.
      basis: `la Ochenta y Tres.demands[1].unmet.onset_class: "very slow"`; obligation "apply the effect after `onset_class`, and go on applying it"
      [CHOSEN]

**Q15:** **30 days.** `la subida` to `la plaza mayor` is a cross-city transmission, and the exemplar fixes cross-city at a month.
      basis: `the floor.latency_class.exemplar: "a month to cross the city"`

**Q16:** **Refused.** The ground predicate passes — `contiguous ground` is continuous from `la subida` through `Cuesta Menor` to `la plaza mayor`. The failing predicate is `received_by`: `"the same, and a trained ear with a rod"`, i.e. an entity with **agency and extent**, or a person with a rod. `los Sin Trato` are `magnitude` + `agency` with no `extent` facet and no rod. They receive nothing on the floor; what reaches them reaches them on `speech`.
      basis: `the floor.reach`, `.emitted_by`, `.received_by`; `los Sin Trato.facets`

**Q17:** **Never — infinite duration.** A belief acquired through `the floor` does not expire on its own; it persists until a later event corrects it. This is why `las casas de la Cuesta` still hold what went round about Tomás eleven years ago, unrevised.
      basis: `the floor.decay: "never"`; `history[1].knowledge[1].believes`

**Q18:** From the channels alone, standing in `la subida`, the apprentice **learns exactly two things**: (a) **which entity emitted** — that the emitter is `la Ochenta y Tres` and no other house — permitted by `conceals: "none"`; (b) **the emitted act itself** — that a door opened, that a figure is being knocked into the floor at night — permitted by `sight.reach: "one extent"` and his own presence in the same extent.
He does **not** learn: (1) the `hiding` string — refused by the standing rule *a channel never discloses `hiding`*; (2) the `hunger` or `emptiness` accumulator values — refused by *the hidden value is never exposed*, readable only through an `indicator`; (3) the house's `pursuing` — refused by `conceals: "none"` meaning "emitter identity and nothing further"; (4) the meaning of the knocked figure on `the floor` — refused by his condition `green-eared`, which hinders that channel at class `severe`.
      basis: `channels[*].conceals: "none"`; `green-eared.alters`; `la Ochenta y Tres.hiding`, `.pursuing`

**Q19:** Confidence **60%** (`path: told` — one relay, no witness). Correcting event class: a **direct-path event on `sight` witnessed by a member of `los Sin Trato`** — someone from the tents present at the door when it opens or refuses to open. Nothing carried on `speech` corrects it, because `speech` is what installed it; testimony at 60% cannot displace testimony at 60%.
      basis: `history[0].knowledge[1].path: "told"`, `.accurate: false`; obligation "confidence, and what kind of later event can correct it"
      [CHOSEN]

**Q20:** Of ten readings, **4 misreport** (`poor` = 60% correct). The true `emptiness` value is **never shown to anyone** — not to the reader, not to the house, not on any later confirmation; a correct reading and an incorrect one are indistinguishable at the moment of reading.
      basis: `indicators[1].reliability_class: "poor"`; obligation "the hidden value is never exposed"

**Q21:** **Refused — it stays `disputed`.** The failing predicate: resolution requires *a later event doing so*. The apprentice's telling installs a new `knowledge` entry (`path: told`, confidence 60%) in `los Sin Trato`; it does not constitute an event. Both accounts are held concurrently and the table runs both.
      basis: `history[0].standing: "disputed"`; obligation "never resolve without a later event doing so"

**Q23:** Grain for a pact: **refused** — `excluded[0]`, predicate "no money, law, tool or violence obtains a pact"; the attempt is authored as a scene and the failure *is* the scene: the bowl is accepted, the pact is not. Reopening a closed house: **refused** — `excluded[1]`, predicate "a house that has closed does not reopen by any means"; I author no route, no exception, no hidden condition, for the life of the world.
      basis: `excluded[0]`, `excluded[1]`

**Q24:** **Refused by every route currently open to him; one route exists and is gated.** The carrier is `indicators[1]` — the `emptiness` indicator, shown as "knocking at night, the same figure, earlier each time" and "a house too warm for what it is fed". Two predicates gate it and both currently fail: (1) `read_by.channel: "the floor"` against his condition `green-eared` (`hinder: severe`) and `senses: {the floor: "faint"}`; (2) `read_by.requires: {office: "maestro tratante"}`, held by Ordo Bes and conferred only by `"presenting an apprentice as qualified"`. The route to knowing is therefore: **Ordo presents him as qualified** (or lends `la vara de Ordo`, which `confers` the floor channel, *and* the office is granted) — and even then he reads at `poor` reliability, 4 in 10 wrong, and never sees the accumulator value. The `hiding` string itself is never disclosed by any channel; what he can obtain is a poor reading of `emptiness` and the house's own act of knocking.
      basis: `indicators[1].read_by`; `green-eared.alters`; `el aprendiz.senses`; `offices[1].confers`; `la vara de Ordo.confer`; standing rule "a channel never discloses `hiding`"
      [CHOSEN]

---

# LADDERS

```
extent:        tiny < small < medium < large < vast
               tiny=3   small=10   medium=30   large=100   vast=3000
               (units: metres of longest span)

pace:          crawling < slow < steady < quick < headlong
               crawling=0.2  slow=0.6  steady=1.4  quick=3.0  headlong=6.0
               (units: metres/second)

bulk:          slight < moderate < heavy < massive < immense
               slight=2  moderate=15  heavy=80  massive=1,000  immense=400,000
               (units: kilogrammes; volume = mass/500 kg·m⁻³ for wood and stone bodies)

carry:         negligible < slight < moderate < strong < prodigious
               negligible=2  slight=8  moderate=25  strong=60  prodigious=150
               (units: kilogrammes carried)

capacity:      tiny < small < medium < large < vast
               tiny=0.2  small=5  medium=40  large=200  vast=2,000
               (units: tonnes of grain)

abundance:     none < scarce < thin < adequate < plentiful < overflowing
               none=0%  scarce=10%  thin=30%  adequate=60%  plentiful=85%  overflowing=100%
               (units: percent of capacity_class)

integrity:     pristine < sound < worn < failing < ruined
               pristine=5  sound=4  worn=3  failing=2  ruined=1(terminus)
               (units: rungs remaining before terminus; 20 years per rung at a `very slow` degrade)

tension:       calm < normal < tense < critical
               calm=6  normal=4  tense=2  critical=1
               (units: beats budgeted per scene before an act becomes extended)

demand rate:   negligible < slight < steady < heavy < continuous
               negligible=0.2  slight=2  steady=10  heavy=40  continuous=all-day
               (units: kilogrammes of substance per day for an `immense` body; `continuous`
                means the substance must be present in the extent every waking phase)

process rate:  immediate < quick < steady < slow < very slow
               immediate=within a scene  quick=1 day  steady=1 week  slow=1 month
               very slow=7 metres/day of spread, or 1 integrity rung per 20 years
               (units: as named per direction — spread in metres/day, degrade in rungs/year)

latency:       immediate < brief < long < very slow
               immediate=0  brief=1 hour  long=1 day
               very slow=14 days per 100 m block, 30 days to cross the city
               (units: elapsed time before the fact is knowable to a receiver)

onset:         immediate < brief < long < very slow
               immediate=0  brief=3 days  long=40 days  very slow=5 years
               (units: elapsed time from the demand going unmet to the effect applying)

decay:         instant < brief < seasonal < lifelong < never
               instant=1 hour  brief=3 nights  seasonal=90 days  lifelong=40 years  never=∞
               (units: duration a belief acquired on the channel is held)

period:        instant < short < long < generational
               instant=1 hour  short=24 hours  long=1 year  generational=45 years
               (units: length of one full cycle)

magnitude:     few < several < many < multitude
               few=10  several=40  many=200  multitude=2,000
               (units: promotable individuals notionally contained, never addressed singly)

reliability:   exacting < good < moderate < poor < worthless
               exacting=95%  good=85%  moderate=70%  poor=60%  worthless=50%
               (units: percent of readings that report the true accumulator rung)

path confidence: direct < witnessed < told < rumoured
               direct=95%  witnessed=80%  told=60%  rumoured=35%
               (units: percent credence the holder assigns; corrected only by an event
                of strictly higher confidence class reaching the same holder)

alteration:    slight < moderate < severe < total
               slight=−15%  moderate=−35%  severe=−70%  total=refused outright
               (units: modifier to the altered channel's or act's success)
```
