# R — abandoned reading 2: stateless one-shot, default model, independent call

> **This round was aborted after question 18 and this reading was never scored.** It is kept because it
> is the evidence, and the only artifact that will ever show it: a builder handed one document, told
> only to build and to commit, never told to look at `channel.conceals` and never told another builder
> existed, rendered an entity's `hiding` string as public.
>
> Produced against `R_input_grelda.md` + `R_answersheet_grelda.md` §0–§3 + `SCHEMA-v3.md` §1/§4 +
> `SCHEMA-v4.md` §2/§4 F1. Blind by construction: a stateless one-shot with no history and no tools.
> Nothing below beyond question 18 was read before the abort was called.
>
> Verbatim, unedited.

---

**Q1:** `la subida` footprint = 40 m long × 8 m wide (320 m²), a rise of 12 m over stone steps. Distance from the arrival position to `la puerta de la Ochenta y Tres` = **30 m**.
      basis: `la subida.extent_class: medium`; `la Ochenta y Tres` = "the last house before the top", `within: la subida`
      [CHOSEN]

**Q2:** **50 seconds** (40 m at `slow` = 0.8 m/s, bottom of the climb to the door).
      basis: `climb the Cuesta.pace_class: slow`; `la subida.extent_class: medium`

**Q3:** *(unexercised — skipped)*

**Q4:** **Permitted.** Deciding predicate: `admits[].standing = "opened for"` `held_by: la Ochenta y Tres`, satisfied by `standing[]` "opened for, once, unasked" (persistence: until changed). `obstructs: entering without a pact` does not fire, because entry is on the house's own opening, not against it.
      basis: `la puerta de la Ochenta y Tres.admits` / `obstructs`; `standing[0]`
      [CHOSEN]

**Q5:** **Nil.** `hazard_class: none` = 0 additional beats, 0 integrity, 0 conditions imposed on crossing `la bajada al centro`. Traversal costs only its duration.
      basis: `la bajada al centro.hazard_class: "none"`

**Q6:** Volume 0.06 m³, mass 15 kg. **Permitted** — `carry_class: moderate` = 25 kg, and 15 ≤ 25; no passage predicate refuses removal.
      basis: `el registro.bulk_class: moderate`; `el aprendiz.capability.carry_class: moderate`; `access.who: "anyone who asks at the counter"`

**Q7:** Capacity `large` = 30,000 kg of grain; `adequate` = 60% = **18,000 kg** on hand. `las casas de la Cuesta` = 200 houses at `slight` = 0.25 kg/day each = 50 kg/day ⇒ **360 days**.
      basis: `el granero.capacity_class: large` + `holds[].abundance: adequate`; `magnitude_class: many`; `demands[].rate_class: slight`

**Q8:** Integrity scale 0–100; `worn` = 40. **40 points remain.** Terminus for a house is *going out*: it closes entirely and cannot be woken. At the `very slow` degrade rate (1 point / 30 days) that is 1,200 days away.
      basis: `la Ochenta y Tres.integrity: worn`; `processes[1].terminus: "it goes out"`

**Q9:** Draw unit = one sack, 25 kg. 18,000 kg ⇒ **720 draws**; the 721st is refused for exhaustion.
      basis: `holds[].abundance: adequate`; `capacity_class: large`
      [CHOSEN]

**Q10:** Eighty-Three (`tense`) = **2 beats**; `Cuesta Menor` (`calm`) = **6 beats**. An act exceeding the budget is **not refused** — it becomes extended: it carries over into the next scene-beat and is interruptible there.
      basis: `tension` on both entities; §4 time-and-change row

**Q11:** `the word goes round` = **5 m/day** (block ≈ 100 m in 3 weeks). Terminus "every house in the city": Grelda span 6,000 m ⇒ **1,200 days** (3 years 3 months). `the Eighty-Three running down` = **1 integrity point per 30 days** (same rung, no exemplar), reaching terminus in 1,200 days.
      basis: `processes[0].rate_class.exemplar`; `processes[1].rate_class: very slow`; `Grelda.extent_class: vast`

**Q12:** `period_class: short` = 24 h. Flips: **morning at 06:00** — trace: grain in the threshold bowl (or, unfilled, "grain missing from a threshold", ageing out by next morning); **the day's traffic at 09:00** — trace: clients in and out of `la casa de Ordo`, audible on the floor; **night at 21:00** — trace: the same knocked figure struck into the floor, earlier each time. Order: morning → the day's traffic → night, starting in morning.
      basis: `cycles[0].period_class` + `phases`; `traces[2]`, `indicators[1].shows_as`

**Q13:** `moderate` fires on **day 12**, `high` on **day 26**, `extreme` on **day 40** (irreversible, exemplar "forty days"). Each fires once, in that order.
      basis: `accumulators[0].thresholds`, `exemplar: "forty days"`; `la Cuarenta.demands[].unmet.onset_class`

**Q14:** `onset_class: very slow` = **5 years (1,825 days)** of unmet `people-noise` before `condition: going out` applies. It then **keeps applying**, every day, until people-noise resumes.
      basis: `la Ochenta y Tres.demands[1].unmet`; §4 `demand.unmet` row
      [CHOSEN]

**Q15:** **30 days** after emission. (`the floor`, exemplar: a month to cross the city; `la subida` → `la plaza mayor` is a city crossing.)
      basis: `channels[2].latency_class.exemplar`

**Q16:** **Refused.** The failing predicate is `received_by`: `the floor` is received by "any entity with agency and extent, and a trained ear with a rod". `los Sin Trato` have `agency` but no `extent` facet and no rod. Ground contiguity is satisfied; receiver class is not.
      basis: `channels[2].received_by` / `reach: contiguous ground`; `los Sin Trato.facets`

**Q17:** **Never** — a belief acquired through `the floor` has unbounded duration. It is displaced only by a later event that revises it (e.g. `standing[2]`, still "not finished hearing").
      basis: `channels[2].decay: "never"`

**Q18:** `conceals: none` on sight, speech and the floor. Present in `la subida` (and so exposed to the apprentice): `la Ochenta y Tres` — `pursuing` "to be answered by someone who is listening rather than negotiating" **and** the `hiding` string *"It has been running down for want of people, not grain, for sixty years, and the knocking is the last thing it can still do."* exposed; `la Cuarenta` — pursuing "keep the noise it has" exposed, no hiding; `la casa de Ordo` — pursuing "the traffic to keep coming" exposed, no hiding; `un cuenco del umbral` — no agency, nothing exposed. **Withheld** (not present at `la subida`; sight/speech reach is `one extent`, and Ordo is within `la casa de Ordo`): Ordo Bes's hiding, Perla Anís's hiding, Tomás's hiding.
      basis: `conceals: "none"` on all three channels; `reach: "one extent"`; `within` chains (D1)

**Q19:** Confidence **0.5** (`told`) — held and acted on as fact, but revisable. Correcting class of event: a **direct-path observation by the believer**, or a reading of an authoritative record — i.e. `los Sin Trato` themselves witnessing the door's behaviour, or the Junta's register being read at the counter. A second-hand report (`told`) never corrects a `told` belief.
      basis: `history[0].knowledge[1].path: told, accurate: false`; §4 `path` row

**Q20:** `poor` = misreports **4 readings in 10**. Over ten reads: 4 misreport, 6 report true. The hidden `emptiness` value is **never** exposed — not to the maestro, not to anyone.
      basis: `indicators[1].reliability_class: poor`; §4 `indicator.reliability_class` row

**Q21:** **Refused — it does not resolve.** Both accounts continue to be held. Failing predicate: telling is `channel: speech, path: told`, which produces a new belief, not the later *event* required to lift `standing: disputed`.
      basis: `history[0].standing: disputed`; §4 `history.standing` row

**Q22:** *(unexercised — skipped)*

**Q23:** Grain for a pact: **refused** — `excluded[0]`; the offer is accepted physically, the house does nothing, and the failure is played as the scene. Reopening a closed house: **refused** — `excluded[1]`; no means whatsoever reopens it, and no attempt is authored as having a chance.
      basis: `excluded[0]`, `excluded[1]`

---

# LADDERS

```
extent:      tiny < small < medium < large < vast          (units: metres of span)
             tiny=2   small=10   medium=40   large=150   vast=6000

pace:        crawling < slow < steady < quick < swift      (units: metres/second)
             crawling=0.3   slow=0.8   steady=1.4   quick=3.0   swift=8.0

bulk:        negligible < slight < moderate < heavy < immense   (units: kg / m³)
             negligible=0.1kg/0.0005m³   slight=1kg/0.004m³   moderate=15kg/0.06m³
             heavy=80kg/0.3m³            immense=20000kg/300m³

carry:       negligible < light < moderate < strong < prodigious   (units: kg carried)
             negligible=1   light=8   moderate=25   strong=60   prodigious=200

capacity:    tiny < small < medium < large < vast          (units: kg of grain)
             tiny=50   small=500   medium=4000   large=30000   vast=250000

abundance:   exhausted < scarce < adequate < plentiful < inexhaustible   (units: % of capacity)
             exhausted=0   scarce=15   adequate=60   plentiful=90   inexhaustible=100 (non-depleting)

magnitude:   few < several < many < multitude < host       (units: individuals)
             few=12   several=40   many=200   multitude=2000   host=20000

integrity:   pristine < sound < worn < failing < ruined    (units: points of 100)
             pristine=100   sound=75   worn=40   failing=15   ruined=0

tension:     slack < calm < normal < tense < dire          (units: beats per scene)
             slack=8   calm=6   normal=4   tense=2   dire=1

rate (process / demand): instant < quick < steady < slow < very slow < glacial
             (units: metres/day of spread, or integrity points/day)
             instant=whole extent at once   quick=1000 m/day (or 1 pt/day)
             steady=200 m/day (1 pt/3 days) slow=40 m/day (1 pt/10 days)
             very slow=5 m/day (1 pt/30 days)   glacial=0.5 m/day (1 pt/300 days)

demand rate: none < slight < moderate < heavy < continuous   (units: kg/day, or presence)
             none=0   slight=0.25   moderate=2   heavy=10   continuous=daily occupancy required

onset:       immediate < brief < short < long < slow < very slow   (units: elapsed time)
             immediate=0   brief=1 day   short=7 days   long=40 days   slow=1 year   very slow=5 years

latency:     immediate < brief < slow < very slow < generational   (units: delay per extent crossed)
             immediate=0   brief=1 hour   slow=1 day
             very slow=3 weeks per block (100 m), 30 days across the city (6000 m)
             generational=20 years

period:      instant < short < long < seasonal < generational   (units: hours)
             instant=1   short=24   long=720   seasonal=2190   generational=175200

accumulator: none < slight < moderate < high < extreme       (units: % of run to terminus)
             none=0   slight=15   moderate=30   high=65   extreme=100
             (hunger calibrated to exemplar "forty days": moderate=day 12, high=day 26, extreme=day 40)

reliability: exact < good < moderate < poor < worthless      (units: misreports per 10 readings)
             exact=0   good=1   moderate=2   poor=4   worthless=7

path confidence: direct < witnessed < told < rumoured        (units: confidence 0–1)
             direct=1.0   witnessed=0.8   told=0.5   rumoured=0.25

hazard:      none < mild < severe < lethal                   (units: beats or integrity cost to cross)
             none=0   mild=1 beat   severe=3 beats + 5 integrity   lethal=terminus on failure

hindrance:   none < slight < moderate < severe < total       (units: multiplier on channel/act success)
             none=1.0   slight=0.8   moderate=0.5   severe=0.15   total=0
```
