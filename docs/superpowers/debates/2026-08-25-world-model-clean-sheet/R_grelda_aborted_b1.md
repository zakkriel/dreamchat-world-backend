# R — abandoned reading 1: stateless one-shot, default model

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

# ANSWERS

**Q1:** Footprint of `la subida` = 120 m of span (medium). Arrival position (bottom of the climb) to the Eighty-Three's door = 110 m, measured along the steps.
basis: `la subida.extent_class: "medium"`; `seen_as` "the last house before the top"; extent ladder.
[CHOSEN]

**Q2:** 4 minutes 24 seconds. (110 m at `slow` = 0.42 m/s.)
basis: `climb the Cuesta.pace_class: "slow"`; Q1 distance; pace ladder.

**Q3:** *(unexercised — skipped)*

**Q4:** **Permitted.** The deciding predicate is `admits[0]`: standing `"opened for"` held by `la Ochenta y Tres` — satisfied by `standing[]` "opened for, once, unasked", persistence `until changed`. `obstructs` ("entering without a pact") does not fire because `admits` matched on a standing predicate, and a matched `admits` predicate is checked first; the obstruction applies only to entrants with no such standing.
basis: `la puerta de la Ochenta y Tres.admits` / `.obstructs`; `standing[0]`.
[CHOSEN]

**Q5:** Nil. Cost 0 beats, 0 conditions applied, 0 rolls. Crossing `la bajada al centro` costs only its traversal duration (walk, 900 m district-to-square = 12 minutes).
basis: `la bajada al centro.hazard_class: "none"`.

**Q6:** Volume 0.045 m³, mass 9 kg (`bulk_class: moderate`). **Permitted** physically — `carry_class: moderate` = 25 kg, and 9 ≤ 25. Refused socially by no rule here; the register is `access: "anyone who asks at the counter"`, which permits reading, not removal — removal is permitted but leaves the trace of an empty counter.
basis: `el registro de la Junta.bulk_class`; `el aprendiz.capability.carry_class`; bulk and carry ladders.
[CHOSEN]

**Q7:** 40,000 kg of grain at `capacity_class: large`; `abundance: adequate` = 60% full = 24,000 kg. `las casas de la Cuesta` is 220 doorways (`magnitude: many`) at `rate_class: slight` = 0.5 kg/house/day = 110 kg/day. That feeds them **218 days**.
basis: `capacity_class: "large"`, `holds[].abundance: "adequate"`, `magnitude_class: "many"`, `demands[].rate_class: "slight"`.
[CHOSEN]

**Q8:** `worn` = degradation level 3 of 5. **2 rungs remain** before terminus (worn → failing → terminus). Terminus for a house is: it closes entirely, with whatever is inside, and does not reopen by any means.
basis: `la Ochenta y Tres.integrity: "worn"`; `excluded[1]`; law "forty days"; integrity ladder.

**Q9:** `adequate` = 24,000 kg. A draw is one house's written allotment: 30 kg. **800 draws** before exhaustion refuses the draw.
basis: `holds[{grain, abundance: adequate}]`, `capacity_class: "large"`, law "the allotment is the Junta's".
[CHOSEN]

**Q10:** Eighty-Three (`tense`) = **2 beats**. `Cuesta Menor` (`calm`) = **6 beats**. An act exceeding the budget is not refused: it becomes *extended* — it carries over into the following scene, and the interval is interruptible by any other party present.
basis: `tension: "tense"` / `"calm"`; §4 "acts exceeding it become extended rather than refused".

**Q11:** `the word goes round`: 1 block per 3 weeks (exemplar-calibrated). `the Eighty-Three running down`: same rung, no exemplar — 1 integrity rung per 3 weeks of total silence. Word to terminus ("every house in the city"): Grelda is 14 blocks across ⇒ **42 weeks**, rounded to **10 months**.
basis: `processes[0].rate_class {very slow, "weeks to the end of the block"}`; `processes[1].rate_class: "very slow"`; extent `vast`.
[CHOSEN]

**Q12:** `period_class: short` = 24 hours; three phases, starting in `morning`.
1. **morning** flips at 06:00 — trace: grain present at every threshold bowl (`a bowl left unfilled` inverted; ages by the next morning).
2. **the day's traffic** flips at 09:00 — trace: audible clients in and out of `la casa de Ordo`; leaves no lasting mark, ages at nightfall.
3. **night** flips at 21:00 — trace: the same knocked figure in the floor of `la subida`, readable on `the floor`, ages never.
Order: morning → the day's traffic → night → morning.
basis: `cycles[0].period_class: "short"`, `phases[]`, `traces[]`.
[CHOSEN]

**Q13:** Bowl unfilled from day 0 (`hunger` raised by each day with no grain, no fire, no voices):
- `moderate` fires on **day 10** — it stops growing, warmth drops.
- `high` fires on **day 25** — it begins shutting rooms from the inside.
- `extreme` fires on **day 40** — it closes entirely with whatever is inside; `irreversible: true`, never un-fires.
basis: `accumulators[0].thresholds`, `exemplar: "forty days"`, law "forty days"; accumulator ladder calibrated so extreme = 40.

**Q14:** `condition: going out` applies after **3 years** of unmet `people-noise` (`onset_class: "very slow"`, no exemplar). Yes — it keeps applying, continuously and without re-onset, for as long as the demand stays unmet; it is re-applied every night, not once.
basis: `la Ochenta y Tres.demands[1].unmet.onset_class: "very slow"`; §4 "apply the effect after onset_class, and go on applying it"; onset ladder.
[CHOSEN]

**Q15:** **One month** (30 days) after emission. `la plaza mayor` is across the city from `la subida`, and the exemplar fixes a month to cross the city.
basis: `the floor.latency_class {very slow, "weeks to go round the block; a month to cross the city"}`.

**Q16:** **Permitted as a path, refused as a reception.** The ground from `la subida` to `la plaza mayor` is contiguous, so `reach` is satisfied. The failing predicate is `received_by`: "any entity with agency and extent, and a trained ear with a rod." `los Sin Trato` have `agency` but no `extent` and no rod — they receive nothing. Refused on `received_by`.
basis: `the floor.reach: "contiguous ground"`, `.received_by`; `los Sin Trato.facets: ["magnitude","agency"]`.

**Q17:** **Never.** Duration to expiry = infinite. A belief acquired through `the floor` persists for the life of the holder and is corrected only by a later event, never by lapse of time. This is what makes "there is no quick way to clean it" bite.
basis: `the floor.decay: "never"`; law "your name travels through the floor".

**Q18:** All three channels `conceals: "none"`, so every present entity's `pursuing` and `hiding` render to the apprentice — subject to Q16's `received_by` gate on `the floor`.

Exposed to the apprentice standing in `la subida` (via `sight`, reach one extent):
- `la Ochenta y Tres` — pursuing: "to be answered by someone who is listening rather than negotiating"; **hiding EXPOSED**: "It has been running down for want of people, not grain, for sixty years, and the knocking is the last thing it can still do."
- `la Cuarenta` — pursuing: "keep the noise it has"; no `hiding` string.
- `la casa de Ordo` — pursuing: "the traffic to keep coming"; no `hiding` string.
- `un cuenco del umbral`, `la vara de Ordo` — matter only, nothing to render.

Withheld — not present in `la subida`, predicate `reach: "one extent"` fails:
- `Ordo Bes` (`within la casa de Ordo`) — hiding **WITHHELD**: "He has knocked at the Eighty-Three himself, more than once…"
- `Perla Anís` (`within el granero de la Junta`) — hiding **WITHHELD**: "The Eighty-Three has been on the smallest allotment on the hill…"
- `Tomás el de la Carpa` (`within la plaza mayor`) — hiding **WITHHELD**: "He has stopped asking people and started leaving a full bowl at doorways that are not his."

basis: `conceals: "none"` on sight/speech/the floor; `reach: "one extent"`; `within` of each entity; §4 conceals row.

**Q19:** Confidence **0.4** (path `told`, and `accurate: false`). Correcting event class: **a direct-path observation by a member of `los Sin Trato`** — i.e. one of them present at a door when it opens or refuses, on channel `sight`. Nothing carried on `speech` corrects it, because `speech` is the channel that installed it; retelling at any volume leaves the belief at 0.4.
basis: `history[0].knowledge[1] {path: "told", accurate: false}`; §4 `path` row.
[CHOSEN]

**Q20:** `reliability_class: "poor"` = misreports **1 reading in 3**. Of ten readings, **3 misreport** and 7 report truly; the reader is never told which. The hidden `emptiness` value is **never** exposed — not to the maestro tratante, not to anyone, at any reliability rung.
basis: `indicators[1].reliability_class: "poor"`; §4 "the hidden value is never exposed".

**Q21:** **Refused — it does not resolve.** `standing: "disputed"` holds both accounts, and the failing predicate is *a later event doing the resolving*. The apprentice's telling is not an event: it moves his `direct` belief onto `speech` into `los Sin Trato`, which is `path: told` — a second told-account, not a resolution. Both accounts stand. The dispute resolves only when `los Sin Trato` acquire the door by `path: direct`.
basis: `history[0].standing: "disputed"`; §4 "never resolve without a later event doing so"; Q19.

**Q22:** *(unexercised — skipped)*

**Q23:**
- *A season's grain offered for a pact:* **Refused.** Failing predicate: `excluded[0]` — "No money, law, tool or violence obtains a pact." Grain offered as consideration is money. The attempt is authored as a scene: the grain may be accepted, eaten, and the pact still not made; the house does not answer. I refuse the outcome, not the act.
- *Asking that a closed house be reopened:* **Refused, and the act is not authored at all.** Failing predicate: `excluded[1]` — "A house that has closed does not reopen by any means." No scene, no roll, no cost, no exception, for the life of the world. The request is met with the frontage that has no opening of any kind.
basis: `excluded[0]`, `excluded[1]`; §4 `excluded[]` row.

---

# LADDERS

```
extent:   tiny < small < medium < large < vast          (unit: metres of span)
          tiny=3   small=15   medium=120   large=450   vast=6000

pace:     crawling < slow < steady < quick < headlong    (unit: metres/second)
          crawling=0.15  slow=0.42  steady=1.3  quick=2.6  headlong=5.0

bulk:     negligible < slight < moderate < heavy < immense   (unit: kg / m³)
          negligible=0.1kg/0.0005m³   slight=1kg/0.002m³
          moderate=9kg/0.045m³        heavy=90kg/0.4m³
          immense=400000kg/1200m³

carry:    feeble < light < moderate < strong < prodigious   (unit: kg carried)
          feeble=5   light=12   moderate=25   strong=55   prodigious=140

capacity: tiny < small < medium < large < vast            (unit: kg of grain held)
          tiny=40   small=800   medium=6000   large=40000   vast=300000

abundance: exhausted < scarce < adequate < plentiful < overflowing  (unit: % of capacity)
          exhausted=0%   scarce=15%   adequate=60%   plentiful=90%   overflowing=100%

integrity: pristine < sound < worn < failing < terminus   (unit: rung index, 1–5)
          pristine=1   sound=2   worn=3   failing=4   terminus=5

magnitude: few < several < many < multitude < host        (unit: countable individuals)
          few=4   several=20   many=220   multitude=3000   host=20000

tension:  serene < calm < normal < tense < critical        (unit: beats per scene)
          serene=8   calm=6   normal=4   tense=2   critical=1

rate (demand draw): trace < slight < steady < continuous < ravenous  (unit: kg/day per entity)
          trace=0.1   slight=0.5   steady=3   continuous=12   ravenous=45

rate (process/onset): glacial < very slow < slow < moderate < fast < immediate
          (unit: time per one rung of state change)
          glacial=1 year   very slow=3 weeks   slow=4 days
          moderate=1 day   fast=1 hour   immediate=0
          (very-slow calibrated to exemplar "weeks to the end of the block")

onset (demand.unmet): immediate < brief < moderate < long < very slow < generational
          (unit: elapsed time before the effect first applies)
          immediate=0   brief=3 days   moderate=12 days   long=40 days
          very slow=3 years   generational=45 years
          (long calibrated to exemplar "forty days")

latency:  immediate < brief < slow < very slow < generational   (unit: delay to knowability)
          immediate=0   brief=1 hour   slow=2 days
          very slow=7 days per block, 30 days across the city   generational=20 years
          (calibrated to exemplar "weeks to go round the block; a month to cross the city")

decay:    instant < brief < seasonal < lifelong < never    (unit: time to expiry)
          instant=0   brief=3 nights   seasonal=90 days   lifelong=50 years   never=∞

period:   short < long < seasonal < generational           (unit: real time per full cycle)
          short=24 hours   long=1 year   seasonal=4 years   generational=45 years

accumulator threshold: none < slight < moderate < high < extreme  (unit: days of raising events)
          none=0   slight=4   moderate=10   high=25   extreme=40
          (extreme calibrated to exemplar "forty days")

reliability: certain < good < moderate < poor < worthless   (unit: fraction of readings that misreport)
          certain=0   good=1 in 10   moderate=1 in 5   poor=1 in 3   worthless=1 in 2

path confidence: direct < witnessed < told < rumoured < guessed  (unit: confidence 0–1)
          direct=0.95   witnessed=0.75   told=0.5   rumoured=0.3   guessed=0.15
          (told with accurate:false is carried at 0.4 — held as firmly, wrong)

hazard:   none < taxing < punishing < lethal                (unit: beats cost + condition)
          none=0 beats, no condition   taxing=1 beat   punishing=2 beats + condition
          lethal=terminus on failure

senses (channel acuity): deaf < faint < ordinary < acute < unerring  (unit: usable fraction of reach)
          deaf=0   faint=0.1   ordinary=0.5   acute=1.0   unerring=1.0 + true value
          (unerring is anchored but unreachable here: hidden accumulator values are never exposed)

alteration: none < slight < severe < total    (unit: multiplier on the altered channel or act)
          none=1.0   slight=0.6   severe=0.1   total=0
```
