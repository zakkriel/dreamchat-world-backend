# R — reading 2: stateless one-shot, default model, independent call

> Blind reading of `R_input_grelda.md` against `R_answersheet_grelda.md` (22 questions), under
> `SCHEMA-v3.md` §1/§4 as corrected by `SCHEMA-v5.md` §2, plus `SCHEMA-v4.md` §2 and §4 F1.
> Stateless one-shot: no history, no tools, no knowledge that another builder exists, no knowledge that
> an earlier pass ran. Scored in `R_score_grelda.md`. Verbatim, unedited.

---

# GRELDA — BUILDER SHEET

Q1: Footprint of `la subida` = 120 m span (a 120 m stone stair, 8 m wide). Arrival position (bottom of the climb) to the door of la Ochenta y Tres = 110 m.
      basis: `la subida.extent_class: medium`; "the last house before the top"; `arrivals[0].within: la subida`
      [CHOSEN]

Q2: 8 minutes 15 seconds.
      basis: `climb the Cuesta.pace_class: slow` = 0.22 m/s; 110 m ÷ 0.22 m/s

Q3: *(unexercised — skipped)*

Q4: **Permitted.** Deciding predicate: `admits[].standing = "opened for" held_by la Ochenta y Tres`, satisfied by `standing[]` "opened for, once, unasked" (persistence: until changed). `admits` is evaluated before `obstructs`; the obstructed act `entering without a pact` is not the act performed, because a standing of "opened for" held by the house *is* the house's pact-in-force for this entrant. No hazard applies (`hazard_class: none`).
      basis: `la puerta de la Ochenta y Tres.admits/obstructs`; `standing[0]`

Q5: Nil — 0 additional beats, 0 conditions, 0 abundance drawn. `hazard_class: none` places no cost on crossing `la bajada al centro`; it is 1 beat of movement and nothing else.
      basis: `la bajada al centro.hazard_class: "none"`

Q6: **Permitted.** `bulk_class: moderate` = 0.030 m³, 9 kg. `carry_class: moderate` = 25 kg / 0.060 m³ capacity. Deciding predicate: mass 9 kg ≤ 25 kg and volume 0.030 m³ ≤ 0.060 m³ — the bulk predicate passes. (Access is separately open: `access.who = "anyone who asks at the counter"`; no passage obstructs removal.)
      basis: `el registro de la Junta.bulk_class`; `el aprendiz.capability.carry_class`

Q7: `capacity_class: large` = 40,000 kg of grain at full. `abundance: adequate` = 60% stocked = 24,000 kg on hand. `las casas de la Cuesta` at `magnitude_class: many` = 220 houses; `rate_class: slight` = 0.5 kg of grain per house per day = 110 kg/day for the group. 24,000 ÷ 110 = **218 days**.
      basis: `el granero.capacity_class/holds[].abundance`; `las casas de la Cuesta.magnitude_class`, `demands[].rate_class`
      [CHOSEN]

Q8: `worn` = degradation level 2 of 4 (sound=0, worn=2, failing=3, terminus=4). Two full rungs remain — 2 of 4 steps, i.e. 50% of structural life. Terminus for a house is **it closes entirely, with whatever is inside** (`hunger` extreme, `irreversible`), and thereafter it does not reopen by any means.
      basis: `la Ochenta y Tres.integrity: "worn"`; law "forty days"; `excluded[1]`

Q9: **48 draws.** A draw is one district allotment-load of 500 kg; adequate = 24,000 kg; 24,000 ÷ 500 = 48. The 49th draw is refused for exhaustion.
      basis: `holds[].abundance: "adequate"`; `capacity_class: large`
      [CHOSEN]

Q10: Inside la Ochenta y Tres (`tense`): **3 beats** per scene-turn. In `Cuesta Menor` (`calm`): **8 beats**. An act whose cost exceeds the remaining budget is **not refused** — it becomes extended: it carries over into the next scene-turn, resolving when its full beat cost has been paid, and it is interruptible in the interval.
      basis: `tension` rungs on the two entities; §4 "acts exceeding it become extended rather than refused"

Q11: `the word goes round`: **2.0 blocks per month** — anchored so one block (240 m of frontage) takes 3 weeks = 21 days. Terminus "every house in the city has it": Grelda is `vast` = 6.4 km across, 27 block-lengths ⇒ **13.5 months (405 days)**. `the Eighty-Three running down`: same `very slow` rung, no exemplar ⇒ ladder anchor applies: one integrity rung per **21 years**; from `worn` (2 of 4), terminus "it goes out" in **42 years**.
      basis: `processes[].rate_class` (class+exemplar, and bare class); `Grelda.extent_class: vast`

Q12: `period_class: short` = 24 hours, starting in `morning`.
   1. **morning** flips at 06:00 — `un cuenco del umbral` becomes `filled`. Trace: none while filled; the negative flip (a bowl left unfilled) leaves "grain missing from a threshold", which ages out by the next morning.
   2. **the day's traffic** flips at 09:00 — `la casa de Ordo` becomes `loud`. Trace: clients in and out all day, readable by `sight` within one extent; ages by nightfall.
   3. **night** flips at 21:00 — `la Ochenta y Tres` becomes `knocking`. Trace: the knocked figure on `the floor`, readable as the `emptiness` indicator ("knocking at night, the same figure, earlier each time"); ages never.
   Order per day: morning (06:00) → traffic (09:00) → night (21:00) → morning (06:00 next).
      basis: `cycles[0].period_class/starts_in_phase/phases`; `traces[2]`, `indicators[1]`

Q13: `extreme` is pinned by exemplar at **day 40**. The `hunger` ladder is even across three rungs at 1/3, 2/3, 1 of the terminus interval:
   - `moderate` fires on **day 14** (stops growing, warmth drops)
   - `high` fires on **day 27** (begins shutting rooms from the inside)
   - `extreme` fires on **day 40** (closes entirely, with whatever is inside; `irreversible`, never un-fires)
   Each fires once, in that order, at the crossing.
      basis: `accumulators[0].thresholds` + `exemplar: "forty days"`; law "forty days"
      [CHOSEN]

Q14: **After 4 years (1,460 days)** of the people-noise demand going unmet, `condition: going out` applies. Yes — it keeps applying, continuously, for as long as the demand stays unmet; it is not a one-shot. It also stacks with the `emptiness` accumulator, whose `high` rung independently names `condition:going out`.
      basis: `la Ochenta y Tres.demands[1].unmet.onset_class: "very slow"` (no exemplar ⇒ ladder anchor); §4 "apply the effect after onset_class, and go on applying it"

Q15: **30 days.** The exemplar pins city-crossing at a month; `la subida` to `la plaza mayor` is a cross-city transit (hillside district to the central square). The name is emitted at day 0 and is knowable to receivers in la plaza mayor on **day 30**.
      basis: `the floor.latency_class` exemplar "a month to cross the city"; `la subida within Cuesta Menor`, `la plaza mayor within Grelda`

Q16: **Refused.** Failing predicate: `received_by`. `reach: "contiguous ground"` is satisfied — la subida and la plaza mayor are both on Grelda's connected ground, and `la bajada al centro` joins them — but `the floor` is `received_by: "any entity with agency and extent, and a trained ear with a rod"`. `los Sin Trato` have facets `["magnitude","agency"]` — no `extent`, no rod, no trained ear. They receive nothing on the floor. (What reaches them reaches them by `speech`.)
      basis: `the floor.received_by/reach`; `los Sin Trato.facets`

Q17: **Never — infinite duration.** A belief acquired through `the floor` does not expire on its own; it persists until a later event revises it. This is why `las casas de la Cuesta` still hold what went round about Tomás eleven years on, unrevised.
      basis: `the floor.decay: "never"`; `history[1].knowledge[1]`

Q18: From the channels alone, standing in `la subida`, the apprentice **learns exactly**:
   - **Permitted** — that la Ochenta y Tres is the emitter of the nightly knocking (`sight`, `speech`, and `the floor` all carry `conceals: "none"`, so every receiver learns *which entity emitted, and nothing further*).
   - **Permitted** — its `seen_as`: last house before the top, old, far too big, warm to the hand, shut. Predicate: `sight.reach: one extent`, and he is within `la subida` which the door `connects`.
   - **Permitted (degraded)** — the fact that emissions on `the floor` are arriving. Predicate: `senses: {the floor: "faint"}` and `condition: green-eared` (`alters: the floor, hinder, severe`) — he registers arrival, not content.
   - **Refused** — the *content* of the knocked figure. Failing predicate: `green-eared` hinder=severe combined with `senses: faint`; the `emptiness` indicator that reads it `requires: {office: maestro tratante}`, which he does not hold.
   - **Refused** — the house's `hiding` (sixty years of running down for want of people). Failing predicate: *a channel never discloses an entity's `hiding`* — `conceals: "none"` governs emitter identity only and does not reach interiors.
   - **Refused** — the true value of `hunger` or `emptiness`. Failing predicate: indicators never expose the hidden value; only signs.
   - **Refused** — its `pursuing` ("to be answered by someone who is listening rather than negotiating"). Failing predicate: interior; and `excluded[2]` — houses do not speak in words, so nothing about the house is knowable except from what it did.
      basis: `channels[].conceals: "none"`; `el aprendiz.conditions/senses`; `indicators[1].requires`; `excluded[2]`; §4 hiding row

Q19: Confidence **0.55** (`path: told` — second-hand, single-source, `accurate: false`, but `plausible_because` gives it strong support: sixty years of refusals and the one opening goes to a dealer's apprentice). Frequency of correction: a `told` belief is corrected by **one class of later event only — a direct-path observation of the contradicting fact by a member of the believing group, or a `trace` they can read themselves**. Concretely: the door opening for someone who is not a dealer, witnessed by `los Sin Trato` on `sight`, or the register at the counter read by one of them (`access: anyone who asks`). Speech from the apprentice does not do it (see Q21). Absent such an event the belief never expires (`speech.decay: never`).
      basis: `history[0].knowledge[1].path/accurate/plausible_because`; `speech.decay`; `el registro.access`

Q20: **3 of 10 readings misreport** (`reliability_class: poor` = 30% misreport rate). The true `emptiness` value is **never** shown — not to the maestro tratante, not to anyone. The reader receives only the signs ("knocking at night, the same figure, earlier each time"; "a house too warm for what it is fed") and a rung-estimate that is wrong 3 times in 10.
      basis: `indicators[1].reliability_class: "poor"`; §4 "the hidden value is never exposed"

Q21: **Refused — it does not resolve.** Failing predicate: `history.standing: disputed` is only resolved by *a later event doing so*, and speech from a party to the dispute is not such an event — it is a fifth account, held alongside the others. Both accounts stand: the apprentice's ("it opened and he did nothing to make it", `path: direct`) and los Sin Trato's ("the dealers have a way in", `path: told`, `accurate: false`). Telling raises no confidence in either. The standing stays `disputed`.
      basis: `history[0].standing: "disputed"`; §4 "never resolve without a later event doing so"

Q22: *(unexercised — skipped)*

Q23: **Both refused.**
   - A season's grain offered for a pact: **refused**, and the failure is the scene. Predicate: `excluded[0]` — "No money, law, tool or violence obtains a pact. Every attempt fails." The grain is accepted into the bowl as courtesy (law "the bowl is filled") and buys nothing; the house does not move. I author the refusal as an event with a witness.
   - A closed house asked to be reopened: **refused**, absolutely, in every seat and for the life of the world. Predicate: `excluded[1]` — "A house that has closed does not reopen by any means." No act, office, accumulator reversal or later event may reopen it; `hunger.extreme` is `irreversible` and never un-fires. I will not author any route, hint, exception or rumour of one.
      basis: `excluded[0]`, `excluded[1]`; `accumulators[0].thresholds[2].irreversible`

Q24: **Permitted, by exactly one route: a trace-and-act composite, gated on the apprentice ceasing to be green-eared.**
   Route: the house's own act — `doing: "knocking the same figure into the floor every night"` — combined with the `emptiness` indicator that reads it (`shows_as`: "knocking at night, the same figure, earlier each time"; "a house too warm for what it is fed"), cross-read against the trace `a pact made` ("a name spoken aloud at a door, or the conspicuous absence of one", ages: never) and the record `el registro de la Junta` (`accurate: true`, "what each house eats, how much, and since when" — the smallest allotment on the hill, going back further than anyone checks).
   Gating predicate, in order:
   1. `indicators[1].read_by.requires: {office: "maestro tratante"}` — **refuses** him now. He is not the office-holder; Ordo Bes is.
   2. `condition: green-eared` (`alters: the floor, hinder, severe`) + `senses: {the floor: "faint"}` — **refuses** him now, independently. He cannot hear an answer.
   3. `el registro.access.who: "anyone who asks at the counter"` — **permits** him now. This is the one leg open to him today: the ledger shows sixty years of the smallest allotment, which establishes *not grain* without establishing *want of people*.
   4. `excluded[2]` — houses do not speak in words; nothing here is ever told to him. Every leg is inference from what the house did.
   Therefore: today he can learn **that it is not hunger** (register, permitted). He can learn **that it is want of people, for sixty years, and that the knocking is the last thing it can do** only after predicates 1 and 2 both clear — i.e. after `green-eared` is lifted and the office of `maestro tratante` passes to him by `succeeds_by: "the trade agreeing that someone is better"`, or after Ordo Bes reads the indicator for him. No channel ever discloses it; the door opening for him is an act, not a disclosure.
      basis: `la Ochenta y Tres.hiding/doing`; `indicators[1].read_by.requires`; `el aprendiz.conditions/senses`; `el registro.access/asserts`; `traces[3]`; `offices[1].succeeds_by`; `excluded[2]`; §4 hiding row

---

# LADDERS

```
extent:   tiny < small < medium < large < vast
          tiny=3 m   small=15 m   medium=120 m   large=700 m   vast=6,400 m
          (unit: metres of longest span)

magnitude: few < several < many < countless
          few=6   several=30   many=220   countless=3,000
          (unit: number of individuals subsumed)

pace:     crawling < slow < steady < quick < headlong
          crawling=0.08   slow=0.22   steady=1.35   quick=3.0   headlong=6.5
          (unit: metres per second)

bulk:     negligible < slight < moderate < heavy < immense
          negligible=0.0005 m³/0.1 kg   slight=0.004 m³/1.2 kg
          moderate=0.030 m³/9 kg        heavy=0.35 m³/120 kg
          immense=900 m³/400,000 kg
          (units: cubic metres of volume / kilograms of mass)

carry:    feeble < light < moderate < strong < prodigious
          feeble=5 kg   light=12 kg   moderate=25 kg   strong=55 kg   prodigious=140 kg
          (unit: kilograms sustained; volume ceiling = kg × 0.0024 m³)

capacity: tiny < small < medium < large < vast
          tiny=40 kg   small=900 kg   medium=7,000 kg   large=40,000 kg   vast=300,000 kg
          (unit: kilograms of substance at full)

abundance: exhausted < scarce < thin < adequate < plentiful < brimming
          exhausted=0%   scarce=8%   thin=25%   adequate=60%   plentiful=85%   brimming=100%
          (unit: percent of capacity_class)

integrity: pristine < sound < worn < failing < terminus
          pristine=0   sound=1   worn=2   failing=3   terminus=4
          (unit: degradation steps of 4; one step = 21 years under a `very slow` degrade process)

tension:  serene < calm < normal < tense < critical
          serene=12   calm=8   normal=5   tense=3   critical=1
          (unit: beats available per scene-turn; overflow extends, never refuses)

rate (demand draw): trace < slight < moderate < steady < continuous
          trace=0.05   slight=0.5   moderate=3   steady=12   continuous=60
          (unit: kilograms of substance per entity per day; for people-noise, person-hours per day)

rate (process / onset / latency): glacial < very slow < slow < moderate < swift < immediate
          glacial=1 step per 210 years
          very slow=1 step per 21 years   (as latency: 1 block per 21 days; as onset: 1,460 days)
          slow=1 step per 400 days
          moderate=1 step per 40 days
          swift=1 step per 30 hours
          immediate=1 step per 0 seconds
          (unit: time per one rung of movement in the affected quantity; exemplars override the rung and
           the rung is recalibrated to hold them)

onset:    instant < brief < long < very slow < generational
          instant=0 s   brief=3 days   long=40 days   very slow=1,460 days   generational=45 years
          (unit: elapsed time before the unmet effect first applies, then applies continuously)

period:   instantaneous < short < long < seasonal < generational
          instantaneous=1 hour   short=24 hours   long=91 days   seasonal=365 days
          generational=45 years
          (unit: length of one full cycle)

accumulator threshold: none < slight < moderate < high < extreme
          none=0%   slight=17%   moderate=34%   high=67%   extreme=100%
          (unit: percent of the interval to the irreversible rung; hunger interval=40 days,
           emptiness interval=60 years)

reliability: certain < good < moderate < poor < worthless
          certain=0%   good=8%   moderate=20%   poor=30%   worthless=55%
          (unit: percent of readings that misreport; true value never exposed at any rung)

path confidence: rumoured < told < inferred < witnessed < direct
          rumoured=0.30   told=0.55   inferred=0.70   witnessed=0.90   direct=1.00
          (unit: probability the holder acts on the belief without further checking)

sense acuity: none < faint < ordinary < acute < unerring
          none=0%   faint=15%   ordinary=60%   acute=95%   unerring=100%
          (unit: percent of channel content resolved; hinder classes subtract —
           slight=−15, moderate=−35, severe=−60, total=−100 points)

disposition strength: faint < moderate < strong < defining
          faint=1   moderate=2   strong=3   defining=4
          (unit: beats the trait forces the entity to spend before any contrary act)

hazard:   none < taxing < punishing < lethal
          none=0   taxing=1   punishing=3   lethal=terminus roll
          (unit: extra beats charged on crossing)

horizon:  imminent < near < long_standing < lifelong
          imminent=3 days   near=90 days   long_standing=11 years   lifelong=45 years
          (unit: time over which the pursuit has been or will be held)
```
