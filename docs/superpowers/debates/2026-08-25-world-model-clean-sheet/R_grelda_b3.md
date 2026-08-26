# R — reading 3: stateless one-shot, slow model — model diversity for the round

> Blind reading of `R_input_grelda.md` against `R_answersheet_grelda.md` (22 questions), under
> `SCHEMA-v3.md` §1/§4 as corrected by `SCHEMA-v5.md` §2, plus `SCHEMA-v4.md` §2 and §4 F1.
> Stateless one-shot: no history, no tools, no knowledge that another builder exists, no knowledge that
> an earlier pass ran. Scored in `R_score_grelda.md`. Verbatim, unedited.

---

# GRELDA — FILLED ANSWER SHEET

---

**Q1:** Footprint of `la subida` = **60 m long × 8 m wide = 480 m²** (a 60 m stone stair, doorways both sides). Distance from the arrival position (the bottom of the climb) to `la puerta de la Ochenta y Tres` = **55 m**.
      basis: `la subida.extent_class: "medium"`; `la Ochenta y Tres.seen_as: "the last house before the top"`, `within: "la subida"`; `arrivals[0].within: "la subida"` + `movements: "climb the Cuesta" — "the bottom of the climb"`.
      [CHOSEN] (the 55 m placement of the door within the 60 m footprint)

**Q2:** **1 minute 32 seconds** (55 m ÷ 0.6 m/s = 91.7 s).
      basis: `movements[].name: "climb the Cuesta", pace_class: "slow"`; Q1 distance.

**Q4:** **PERMITTED.** Deciding predicate: `admits[0] = { standing: "opened for", held_by: "la Ochenta y Tres" }` — satisfied by `standing[0]`: from `la Ochenta y Tres` toward `el aprendiz`, stance `"opened for, once, unasked"`, `persistence: "until changed"`, which is unchanged. `obstructs[0] = { act: "entering without a pact" }` is evaluated only against those who fail `admits`; a holder of the house's own `opened for` standing is not entering *without* the house — he is entering *without a pact*, which the house's granted standing supersedes at the threshold. Note the standing does not create a pact: law `two nights` still caps him at one night inside.
      basis: `la puerta de la Ochenta y Tres.admits/obstructs`; `standing[0]`; `law: "two nights"`.
      [CHOSEN] (the precedence ruling: `admits` resolves before `obstructs`)

**Q5:** **Nil — 0.** Zero beats spent, zero conditions applied, zero accumulator ticks, no predicate to satisfy. Crossing `la bajada al centro` costs only the pace-and-distance duration of the move itself.
      basis: `la bajada al centro.hazard_class: "none"`.

**Q6:** Volume **30 litres**, mass **20 kg**. Carrying it out: **REFUSED.** Bulk permits — 20 kg is under the apprentice's `moderate` carry anchor of 35 kg. The failing predicate is `access.who: "anyone who asks at the counter"`: that grants consultation *at the counter*, not removal from `el granero de la Junta`. Second failing predicate, if he takes it anyway: `law: "the allotment is the Junta's"`, `enforced_by: "office"`, held by Perla Anís.
      basis: `el registro de la Junta.bulk_class: "moderate"`, `access.who`; `el aprendiz.capability.carry_class: "moderate"`; `law: "the allotment is the Junta's"`.
      [CHOSEN] (that counter-access excludes removal)

**Q7:** The granary holds **12,000 kg of grain** (`large` capacity = 20,000 kg × `adequate` = 60%). `las casas de la Cuesta` = **220 houses** × `slight` = 0.5 kg/day = **110 kg/day**. That feeds them for **109 days**.
      basis: `el granero.capacity_class: "large"`, `holds[].abundance: "adequate"`; `las casas de la Cuesta.magnitude_class: "many"` calibrated to `seen_as: "two hundred-odd doorways"`; `demands[].rate_class: "slight"`.

**Q8:** `worn` = **55 of 100 degradation points**; **45 points remain** before terminus. Terminus for a house is `emptiness` at `extreme`: **"it goes out and cannot be woken"**, `irreversible: true` — a frontage with no opening of any kind, permanent, non-reopenable. At the running-down rate (Q11) the Eighty-Three is **1,500 days — 4 years 1 month — from terminus**.
      basis: `la Ochenta y Tres.integrity: "worn"`; `accumulators[emptiness].thresholds[extreme].irreversible`; `excluded[1]`; `traces: "a house closing"`.

**Q9:** **800 draws.** One draw = one house's written monthly allotment = 15 kg (30 days × `slight` 0.5 kg/day). 12,000 kg ÷ 15 kg = 800. The **801st draw is refused** with `abundance: exhausted`.
      basis: `holds[].abundance: "adequate"`; `capacity_class: "large"`; `law: "the allotment is the Junta's"` (a draw is an allotment as written).
      [CHOSEN] (fixing the unit of "a draw" at one written monthly allotment)

**Q10:** Inside `la Ochenta y Tres` (`tense`): **4 beats per scene.** In `Cuesta Menor` (`calm`): **8 beats per scene.** An act that exceeds the budget is **not refused — it becomes extended**: it carries past the scene boundary unfinished, resolves at one beat per subsequent scene, and remains interruptible for the whole of that carry. The apprentice cannot complete an 8-beat listening inside the Eighty-Three in one night; he gets 4 beats and the rest hangs.
      basis: `tension` on `la Ochenta y Tres` and `Cuesta Menor`; §4 obligation "acts exceeding it become extended rather than refused".

**Q11:** Both run at the `very slow` anchor: **0.03% of the distance to terminus per day.**
 - `the word goes round`: 0.03%/day → end of the block (20 of 3,000 houses = 0.67% of terminus) in **22 days ≈ three weeks**, matching the exemplar. Terminus ("every house in the city has it") at **3,333 days = 9 years 1 month.**
 - `the Eighty-Three running down`: 0.03%/day, 45 points remaining from `worn` → terminus in **1,500 days = 4 years 1 month.**
      basis: `processes[].rate_class` (`{class:"very slow", exemplar:"weeks to the end of the block"}` and bare `"very slow"`); `integrity: "worn"`.

**Q12:** `period_class: "short"` = a **24-hour** cycle, three phases, `starts_in_phase: "morning"`. Flips, in order:
 1. **morning — 06:00** (runs 06:00–09:00, 3 h). Change: `un cuenco del umbral` becomes `filled`. Trace left: grain standing in every threshold bowl; the *failed* flip leaves `"grain missing from a threshold"`, which `ages: "by the next morning"`.
 2. **the day's traffic — 09:00** (runs 09:00–20:00, 11 h). Change: `la casa de Ordo` becomes `loud`. Trace left: the door of `la casa de Ordo` standing open with clients in and out, and traffic legible on `the floor`; ages by nightfall.
 3. **night — 20:00** (runs 20:00–06:00, 10 h). Change: `la Ochenta y Tres` becomes `knocking`. Trace left: the same figure knocked into the floor, **earlier each time** — the `emptiness` indicator's first sign; `ages: never`.
      basis: `cycles[0].period_class/starts_in_phase/phases`; `traces[]`; `indicators[of: emptiness].shows_as`.
      [CHOSEN] (the clock times 06:00 / 09:00 / 20:00 within the 24-hour period)

**Q13:** Bowl unfilled from day 0 on `la Cuarenta`:
 - `moderate` — **day 18** ("it stops growing and the warmth drops")
 - `high` — **day 28** ("it begins shutting rooms from the inside")
 - `extreme` — **day 40** ("it closes entirely, with whatever is inside", `irreversible: true`)
 Each fires **once**, in that order, at the crossing. Day 40 never un-fires, and `excluded[1]` forbids reopening. Refilling on day 30 un-fires nothing already fired; it stops the climb only.
      basis: `accumulators[hunger].thresholds`, `exemplar: "forty days"`; `law: "forty days"`; `excluded[1]`.

**Q14:** `condition: going out` applies **365 days — one year — after the last night anyone slept inside**, and **yes, it keeps applying**, every day thereafter, without limit. On the Eighty-Three it has been applying for **59 of its 60 empty years**, which is what put it at `integrity: worn` and gave it the condition it now carries.
      basis: `la Ochenta y Tres.demands[people-noise].unmet.onset_class: "very slow"`; `conditions: "going out"`; §4 "apply the effect after `onset_class`, and go on applying it".

**Q15:** **20 days.** `the floor` propagates at 200 m/day of contiguous ground (`very slow`, calibrated so a 6,000 m city span = 30 days = "a month to cross the city"). Route length `la subida` → `la plaza mayor` = 4,000 m → 4,000 ÷ 200 = **20 days** before the apprentice's name is knowable there.
      basis: `channels["the floor"].latency_class {class:"very slow", exemplar:"weeks to go round the block; a month to cross the city"}`; `Grelda.extent_class: "vast"`.

**Q16:** **REFUSED.** `reach: "contiguous ground"` is *satisfied* — `la subida` and `la plaza mayor` sit on one contiguous ground, joined by `la bajada al centro`. The failing predicate is `received_by: "any entity with agency and extent, and a trained ear with a rod"`. `los Sin Trato` have `facets: ["magnitude","agency"]` — **agency but no `extent`** — and hold no rod (`la vara de Ordo` is `within la casa de Ordo`). They cannot receive on `the floor` at any distance. Their knowledge of the door reaches them on `speech`, `path: told`, which is why it arrives wrong.
      basis: `channels["the floor"].reach/received_by`; `los Sin Trato.facets`; `la vara de Ordo.within`; `history["the-door-that-opened"].knowledge[los Sin Trato].channel: "speech"`.

**Q17:** **Never — unbounded duration, ∞.** A belief acquired through `the floor` has no expiry timer and is not re-checked. It ends only when a later event of strictly higher confidence corrects it (Q19), or when the holder ends. This is the engine of `law: "your name travels through the floor"` — "there is no quick way to clean it" — and of `standing[2]`, `persistence: "permanent"`, and of `las casas de la Cuesta` still holding what went round about Tomás eleven years on.
      basis: `channels["the floor"].decay: "never"`; `law: "your name travels through the floor"`; `history["the-eleven-years"]`.

**Q18:** Standing in `la subida`, from the channels alone:
 **LEARNS (permitted):**
 - `sight` — that `la Ochenta y Tres` exists, **which entity it is** (predicate: `conceals: "none"` ⇒ every receiver learns which entity emitted, *and nothing further*); its `seen_as` surface: last house before the top, old, far too big, warm to the hand, shut; `reach: "one extent"` satisfied, both are in `la subida`.
 - `sight` — the traces present at the frontage: `"a frontage with no opening of any kind"`, `"wall where a doorway was"`, whether the threshold bowl is filled or `"grain missing from a threshold"`.
 - `sight`/`the floor` — the `doing` it performs publicly: that it knocks at night; **that** it knocked, and that it was the Eighty-Three knocking.
 **DOES NOT LEARN (refused), with the predicate that decided each:**
 - The *content* of the knocking — predicate: `conditions: "green-eared"` alters `the floor` with `effect: hinder, class: "severe"`, against `senses: {the floor: "faint"}`; net legibility 0. He can knock and wait, he cannot hear an answer.
 - The `hiding` — predicate: **a channel never discloses an entity's `hiding`** (see Q24).
 - The `pursuing` ("to be answered by someone who is listening rather than negotiating") — predicate: interior; reachable only through an act, an `indicator` or a `trace`.
 - The `hunger` and `emptiness` accumulator values — predicate: `indicator.reliability_class`; the hidden value is **never** exposed to anyone.
 - The `disposition` strength (`patient`/`defining`) — predicate: interior, inferable from acts only.
      basis: `channels[].conceals: "none"`, `reach: "one extent"`; `conditions["green-eared"].alters`; `el aprendiz.senses`; `traces[]`; §4 knowing rows.

**Q19:** Confidence **0.70** — `path: "told"`. Frequency reading: in 7 of 10 occasions they act on it and restate it as fact; in 3 of 10 they hedge it. The class of later event that can correct it: **only an event of strictly higher path-confidence than `told`** — i.e. a `direct` observation on `sight` by a member of `los Sin Trato` (the door refusing a dealer in their presence, or opening for a non-dealer), or a `trace` read first-hand: `"a name spoken aloud at a door, or the conspicuous absence of one"`. Another `told` — including the apprentice's own testimony on `speech` — cannot correct it (see Q21). `decay: "never"`, so until such an event it stands forever.
      basis: `history["the-door-that-opened"].knowledge[los Sin Trato]`: `path: "told"`, `accurate: false`, `plausible_because`; `traces["a pact made"]`; `channels[speech].decay: "never"`.
      [CHOSEN] (the numeric confidence anchors on the path ladder)

**Q20:** **4 of the 10 readings misreport** (`poor` = 40% misreport rate). And: **no — the true `emptiness` value is never shown to anyone, at any time, by any route.** Not to the maestro tratante, not to the house, not to the Junta. The reader receives only a reading; a misreport is indistinguishable from a true report at the moment of reading, and only a later event can separate them.
      basis: `indicators[of: "emptiness"].reliability_class: "poor"`, `read_by: {channel:"the floor", requires:{office:"maestro tratante"}}`; §4 "the hidden value is **never** exposed".

**Q21:** **REFUSED — it does not resolve.** `the-door-that-opened` remains `standing: "disputed"`; both accounts continue to be held simultaneously. Failing predicate: `history.standing: "disputed"` may be resolved **only by a later event**, and the apprentice's telling is not one — it enters `los Sin Trato`'s knowledge on `channel: "speech"` at `path: "told"`, which is the same confidence rung the false belief already occupies (0.70) and therefore cannot displace it. It adds a third account to hold, it does not collapse the dispute. What would resolve it: the door opening or refusing in front of them (`sight`, `direct`), or the trace of a pact.
      basis: `history["the-door-that-opened"].standing: "disputed"`; `knowledge[].path` values; §4 "never resolve without a later event doing so".

**Q23:** Two different rulings, both refusals, from two different `excluded[]` entries:
 - **A season's grain for a pact — REFUSED at the table, played as a scene.** Failing predicate: `excluded[0]` — "No money, law, tool or violence obtains a pact. Every attempt fails and **the failure is the scene**." The offer is *permitted to be made and must be played*: the grain is carried up, set down, and the house does not answer. The pact does not occur. Cross-check: `law: "a house is not forced"` forbids `{subject: "any entity with agency", act: "compelling a pact"}`, `enforced_by: "physics"` — there is no roll, no margin, no partial success.
 - **Reopening a closed house — REFUSED at authoring.** Failing predicate: `excluded[1]` — "A house that has closed does not reopen by any means", reinforced by `accumulators[hunger].thresholds[extreme].irreversible: true` and `traces["a house closing"].ages: "never"`. I refuse to author the reopening in any seat, by any agent, in any epoch, for the life of the world. The player may spend a lifetime trying; the frontage stays blank.
      basis: `excluded[0]`, `excluded[1]`; `law: "a house is not forced"`, `law: "forty days"`; `accumulators[hunger]`; `traces`.

**Q24:** **PERMITTED, by exactly one route: an act the house itself performs, received from inside.**
 - **The route:** `la Ochenta y Tres`'s `doing` — "knocking the same figure into the floor every night" — heard while he is *inside* the house. Inside, the medium is `house-warm`, which `affords the floor` at `degree: "full"`; outside, `street air` `resists the floor` at `degree: "severe"`. The full affordance lifts his `faint` floor-sense above the `green-eared` severe hindrance for one night's worth of the figure: he hears that the knock is a call for people and not for grain, and that it is the last act the house has left.
 - **Gating predicates he must pass, in order:** (1) `la puerta de la Ochenta y Tres.admits[0]` — the standing `"opened for"` held by the house, which `standing[0]` gives him (Q4); (2) `medium: "house-warm"` affording `the floor` at `full`; (3) `law: "two nights"` — he has **one night only**; the second night the house feels as intrusion and the route closes.
 - **Routes REFUSED, with the predicate that closed each:**
   - Any channel — predicate: **a channel never discloses an entity's `hiding`**. `conceals: "none"` gets him the emitter's identity and nothing further.
   - The `emptiness` indicator — predicate: `read_by.requires.office: "maestro tratante"`. That office is `held_by: "Ordo Bes"`; the apprentice is not the holder and `succeeds_by: "the trade agreeing that someone is better"`. Refused on office. And even passed, it is `reliability_class: "poor"` and would never show the value.
   - Asking Ordo — predicate: `excluded[2]`, houses do not speak in words, and Ordo's own `hiding` plus `doing: "refusing, for the second day, to say why"`.
   - The register at the counter — predicate: `asserts[].claim` covers "what each house eats, how much, and since when" and is `accurate: true`; it carries the smallest allotment on the hill, which is evidence *toward* the hiding but is not the hiding, and reading it does not disclose it.
      basis: `la Ochenta y Tres.hiding/doing/medium`; `vocabulary.media` affordances; `conditions["green-eared"]`; `el aprendiz.senses`; `standing[0]`; `la puerta.admits`; `law: "two nights"`; `indicators[of:"emptiness"].read_by.requires`; `offices["maestro tratante"]`; §4 "an interior becomes knowable only through an act the entity performs, an `indicator` that reads it, or a `trace` it leaves".
      [CHOSEN] (that `house-warm`'s full affordance is what defeats the `green-eared` hindrance)

---

# LADDERS

```
extent          tiny  <  small  <  medium  <  large  <  vast
                tiny=2 m   small=10 m   medium=60 m   large=250 m   vast=6,000 m
                (units: metres of longest span; footprint = span × span/7.5 for corridors
                 and streets, span × span for open ground)

pace            crawling < slow < steady < quick < headlong
                crawling=0.2   slow=0.6   steady=1.4   quick=3.0   headlong=6.0
                (units: metres per second)

bulk            negligible < slight < moderate < heavy < immense
                negligible = 0.2 L / 0.1 kg
                slight     = 3 L / 1.5 kg
                moderate   = 30 L / 20 kg
                heavy      = 500 L / 400 kg
                immense    = 2,000 m³ / 1,500 tonnes
                (units: litres of displacement / kilogrammes)

carry           feeble < light < moderate < strong < prodigious
                feeble=5   light=15   moderate=35   strong=70   prodigious=150
                (units: kilogrammes carried over a full move without penalty)

capacity        negligible < small < moderate < large < immense
                negligible=5   small=200   moderate=2,000   large=20,000   immense=200,000
                (units: kilogrammes of grain held at full)

abundance       exhausted < scant < adequate < plentiful < inexhaustible
                exhausted=0%   scant=15%   adequate=60%   plentiful=90%   inexhaustible=unbounded
                (units: percent of the holder's capacity_class anchor)

magnitude       few < several < many < multitude < host
                few=5   several=15   many=220   multitude=2,000   host=20,000
                (units: individuals subsumed; "many"=220 calibrated to "two hundred-odd doorways")

demand rate     negligible < slight < steady < continuous < voracious
                negligible=0.05   slight=0.5   steady=2   continuous=8   voracious=30
                (units: kilogrammes of substance per day per entity; for non-weighable
                 substances — people-noise, fire-heat — the same rungs read as
                 person-hours per day: slight=0.5, continuous=8)

integrity       pristine < sound < worn < failing < ruined(terminus)
                pristine=0   sound=20   worn=55   failing=80   ruined=100
                (units: degradation points of 100; terminus fires at 100 and is irreversible)

tension         languid < calm < normal < tense < critical
                languid=12   calm=8   normal=6   tense=4   critical=2
                (units: beats available in one scene; the beat over budget extends the act
                 into following scenes at 1 beat per scene, never refuses it)

process rate    imperceptible < very slow < slow < steady < swift < headlong
                imperceptible=0.005   very slow=0.03   slow=0.3   steady=2   swift=10   headlong=50
                (units: percent of the distance to terminus per day.
                 very slow=0.03 calibrated so 0.67% of terminus — one block of 20 houses
                 in 3,000 — takes 22 days: "weeks to the end of the block")

cycle period    fleeting < brief < short < long < slow < ages
                fleeting=1 h   brief=6 h   short=24 h   long=30 d   slow=1 y   ages=100 y
                (units: hours/days/years of one full turn of the cycle)

latency         immediate < brief < moderate < slow < very slow < geological
                immediate=0 s   brief=10 s   moderate=1 h   slow=1 d
                very slow=200 m/day of propagation   geological=20 m/day
                (units: delay; the two slowest rungs are distance-rated.
                 very slow=200 m/day calibrated so a 6,000 m city span = 30 days:
                 "a month to cross the city")

onset           immediate < quick < short < long < very slow < generational
                immediate=0   quick=1 d   short=7 d   long=40 d   very slow=365 d
                generational=25 y
                (units: elapsed time from the demand first going unmet to the effect
                 first applying; long=40 d calibrated to exemplar "forty days")

accumulator     none < slight < moderate < high < extreme
level           none=0%   slight=25%   moderate=45%   high=70%   extreme=100%
                (units: percent of the run to the irreversible threshold. On hunger,
                 whose extreme carries exemplar "forty days", the rungs land on
                 day 0 / 10 / 18 / 28 / 40)

reliability     unfailing < high < moderate < poor < worthless
                unfailing=0%   high=10%   moderate=25%   poor=40%   worthless=70%
                (units: percent of readings that misreport; the true value is never
                 exposed at any rung, including unfailing)

path confidence direct < told < inferred < rumoured
                direct=0.95   told=0.70   inferred=0.50   rumoured=0.30
                (units: probability the holder acts on the belief as fact.
                 Correction requires an event on a strictly higher rung)

hazard          none < toll < ordeal < peril < lethal
                none=0 beats/0 conditions   toll=1 beat   ordeal=2 beats + 1 condition
                peril=4 beats + 1 condition   lethal=terminus on failure
                (units: beats spent and conditions applied on crossing)

affordance      total < severe < partial < slight < none  (resists)
                none < slight < partial < full            (affords)
                resists: total=×0, severe=×0.15, partial=×0.5, slight=×0.85, none=×1
                affords: none=×1, slight=×1.3, partial=×1.8, full=×3
                (units: multiplier on the receiver's sense acuity for that channel)

alteration      slight < moderate < severe < total
                slight=×0.85   moderate=×0.5   severe=×0.15   total=×0
                (units: multiplier a condition applies to the altered channel or act)

sense acuity    deaf < faint < fair < acute < unerring
                deaf=0   faint=0.25   fair=0.6   acute=1.0   unerring=1.5
                (units: legibility score; a channel yields content at ≥0.5 after all
                 affordance and alteration multipliers, identity-of-emitter only below)
```
