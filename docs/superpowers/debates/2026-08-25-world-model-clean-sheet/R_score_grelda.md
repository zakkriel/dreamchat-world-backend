# R — score: the reader half, three blind builders, one document

**Round:** the first test of the reader's half of the contract. Every previous round tested whether an
author can write a good document. None had asked the other question: **given one document and no author
to ask, do independent builders build the same world?**

**Inputs:** `R_input_grelda.md` (the fixture — the JSON of `G_grelda_by_simarch.md` and nothing else),
`R_answersheet_grelda.md` (24 obligations → 22 questions, plus the procedure that generated them),
`SCHEMA-v3.md` §1/§4 as corrected by `SCHEMA-v5.md` §2, `SCHEMA-v4.md` §2 and §4 F1.

**Readings:** `R_grelda_b1.md`, `R_grelda_b2.md`, `R_grelda_b3.md`. Two on one model, one on another,
all three stateless one-shots — no history, no tools, no knowledge that another builder existed, no
knowledge that a previous pass had run.

**The round ran twice.** The first pass aborted at question 18 on a condition written before it started:
both builders published an entity's `hiding` string. That is `SCHEMA-v5.md`, and the evidence is
`R_grelda_aborted_b1.md` / `R_grelda_aborted_b2.md`. Everything below is the re-run against the
corrected obligation.

**This file has been corrected by its own adversarial review** (`R_score_grelda_review.md`). The review
found two factual errors in the first scoring, one arithmetic error, and withdrew a defect this file had
claimed against `SCHEMA-v5.md`. §0.1 is the log. Read it before citing any number here.

---

## 0. Method notes, stated because they change what this file is evidence of

- **Classification thresholds.** `same magnitude` means **within a factor of ~2**. That factor is an
  **assumption**, not derived from anything; it needs confirmation. Per review finding **F5** it is also
  **not sufficient on its own**: it scores answers, and answers can match while the grammar under them
  is inverted (Q20). Every `same magnitude` result below is therefore additionally **gated on
  re-derivation from the builders' declared ladders**, not their outputs.
- **Ratios are only reported where the dimensions agree.** Q8 and Q11 are **dimension mismatches**, not
  factors — dividing remaining-years by points-of-100, or a metre-terminus by a percent-of-terminus,
  measures nothing (**F5**).
- **Attribution is one of four**, per the plan: *obligation silent* · *obligation ambiguous* ·
  *ladder unwritten* · *builder error*. A fifth column proved necessary and is **not** an attribution:
  **round defect** — a divergence caused by the question rather than by the contract or the builder.
  Three exist, and they are errors by the party that both wrote and scored the questions (**F7**).
- **The discriminator between *ladder unwritten* and *obligation ambiguous*** — used consistently after
  **F4**: if the builders' **rung order agreed** and the divergence is in the **dimension, polarity or
  quantity being measured**, the obligation is ambiguous. Only if the order agreed *and* the dimension
  agreed *and* only the anchors differ is it a ladder gap.
- **Builder discipline held.** 0 hedges across three readings; 21 `CHOSEN` marks (10/4/7); no builder
  audited the document, reported a defect, or commented on the contract. Instructions 2 and 4 did their
  work. These are committed answers, not annotated uncertainty.
- **Delivery deviated from plan, and the deviation is recorded rather than hidden.** The plan dispatched
  B1/B2 as concurrent `task` subagents. All three subagent attempts died on a transport error
  (`resource_exhausted`, 0 bytes returned — two at ~10 minutes, one at 1m38s) before producing anything.
  All five builders (2 aborted + 3 scored) and both review seats therefore ran as stateless
  `completion()` one-shots. **Blindness is not weakened** — a stateless one-shot shares strictly less
  context than a subagent — but two things are lost: no builder had tool access to re-read its packet,
  and **B1 and B2 share a model with each other.** Per **F8**, any convergence resting only on B1+B2 is
  one model twice and is struck from the evidence. B3 on a different model carries the model-diversity
  claim.

### 0.1 Corrections log — what the adversarial review changed

| Finding | Change | Verified how |
|---|---|---|
| **F1** | **Q18 was mis-scored. It is `identical`, not a divergence.** All three builders refuse `pursuing` — B2: *"**Refused** — its `pursuing` … Failing predicate: interior"*; B3: *"predicate: interior; reachable only through an act, an `indicator` or a `trace`"*. **The defect this file claimed against `SCHEMA-v5.md` is withdrawn.** It was scorer error, produced by reading a truncated extract. | grepped `pursuing` in all three Q18 answers |
| **F2** | **Q24 was mis-scored twice.** (a) B1 and B2 reach the *same* substantive ruling and differ only in headline word — B2's body: *"only after predicates 1 and 2 both clear"*, B1: *"refused by every route currently open to him; one route exists and is gated."* (b) B3's permit is refuted by B3's **own** minted arithmetic: `faint` 0.25 × `affords full` ×3 × `severe` ×0.15 = **0.1125** against B3's own stated content threshold **≥0.5**. That is builder error, internally checkable. | recomputed B3's ladder; read B2's full body |
| **F3** | **Q6 → builder error (B2), not *obligation silent*.** B2 did not treat `access` as absent; it cited it and read it as licensing removal. B1 and B3 read "at the counter" as locative. A divergence over the plain sense of an authored string is builder error. | read B2's Q6 basis line |
| **F4** | **Q8 and Q20 → *obligation ambiguous*; Q19 → *obligation silent (grammar)*.** Applying this file's own discriminator consistently. Ladder-unwritten falls 10 → 7. | §0 discriminator, applied |
| **F5** | `same magnitude` gated on ladder re-derivation. **Q20 reclassified to contradictory** (inverted polarity). Q8/Q11 reported as dimension mismatch, not ratios. The headline "4× to 16×" corrected. | §4.2 polarity table |
| **F6** | **REJECTED on evidence.** The review claimed B3's 3,000-house terminus was "an uncited invention". It is authored **three times** in the fixture — `world.premise` "three thousand living houses", `Grelda.seen_as` "three thousand houses" — and B3 used the process's own authored `terminus: "every house in the city has it"`, which is a *more* faithful reading than B1/B2's substitution of city width. Q11 stands as evidence. | grepped the fixture; read B3's basis |
| **F7** | **New column: round defect.** Q13's premise does not satisfy `hunger.raised_by`; Q9 demands a unit ("a draw") the fixture never mints; Q18 was mis-scored. The proposed fifth attribution class ("silent on partial raisers") is **dropped** — a conjunction is satisfied when its conjuncts hold, and the contract owes nothing there. | accepted |
| **F8** | Three convergences downgraded: **Q4** is three matching answers over two grammars (a missed finding, now §5 and §8.3); **`carry: moderate = 25 kg`** struck — B1+B2, one model twice; **Q12's "semantic anchoring"** demoted to a hypothesis with an unexcluded rival (24 h is the dominant prior for "short"). | model provenance; §4.1 |
| **F9** | §4 split into **vocabulary** (anchors, per-world) and **grammar** (rung membership, order, polarity, dimension — closed, ours, in code). Shipping them as one table patches an engine gap at world creation, which the fourth design test forbids. | accepted |
| **F10** | The `within`/R7 watch-item now names a scoped engine program instead of remaining a note. | accepted |
| **F11** | **Arithmetic error corrected.** Ladder-unwritten rows were 10, not 9. | recounted |

**Nine accepted, one accepted in part, one rejected.** The review is verbatim in
`R_score_grelda_review.md`, including F6.

---

## 1. The scoreboard

| # | Obligation | Classification | Attribution | Spread |
|---|---|---|---|---|
| 1 | `extent_class` | **contradictory** | ladder unwritten | 30 / 120 / 60 m span — **4×** |
| 2 | `pace_class` | **contradictory** | ladder unwritten | 50 s / 495 s / 92 s — **10×** |
| 3 | `motion.trajectory` | — | — | unexercised: no entity has `motion` |
| 4 | `passage.admits`/`obstructs` | identical **answers, two grammars** | **obligation silent** (precedence) | permit ×3; B1/B3 by invented precedence, B2 by a different mechanism |
| 5 | `passage.hazard_class` | **identical** | — | nil / nil / nil |
| 6 | `bulk_class` | **contradictory** (ruling) | **builder error** (B2) | refuse / permit / refuse; mass 15/9/20 kg |
| 7 | `capacity_class` | **contradictory** | ladder unwritten | 200 t / 40 t / 20 t — **10×** |
| 8 | `integrity` | **contradictory** | **obligation ambiguous** | *dimension mismatch*: rungs-remaining / steps-accrued / points-of-100 |
| 9 | `holds[].abundance` | **contradictory** | obligation ambiguous *(+ round defect)* | 300 / 48 / 800 draws |
| 10 | `tension` | same magnitude *(gated)* | ladder unwritten | tense 2/3/4; calm 6/8/8 — rung sets differ 4/5/5 |
| 11 | `process.rate_class` | **contradictory** | obligation ambiguous | *dimension mismatch*: distance/day vs %-of-terminus/day |
| 12 | `cycle.period_class` + `phases` | **identical** | — | 24 h; 06:00 → 09:00 → night, all three |
| 13 | `accumulator.thresholds` | same magnitude | ladder unwritten *(+ round defect)* | mod 10/14/18 · high 25/27/28 · **extreme 40/40/40** |
| 14 | `demand.unmet` | **contradictory** | ladder unwritten | 5 y / 4 y / 1 y — **5×** |
| 15 | `channel.latency_class` | same magnitude | ladder unwritten | 30 d / 30 d / 20 d — **1.5×** |
| 16 | `channel.reach` | **identical** | — | refuse, same failing predicate, all three |
| 17 | `channel.decay` | **identical** | — | never / never / never |
| 18 | `channel.conceals` (v5) | **identical** | — | identity disclosed, `hiding` **and** `pursuing` refused, all three |
| 19 | `path` | same magnitude *(anchors)* | **obligation silent** (grammar) | 0.60 / 0.55 / 0.70 — but 4 / 5 / 4 rungs, and `inferred` above `told` in one, below in another |
| 20 | `indicator.reliability_class` | **contradictory** | **obligation ambiguous** | 4 / 3 / 4 of 10 — **inverted polarity**, `poor` = 60% *correct* vs 30% *wrong* |
| 21 | `history.standing: disputed` | **identical** | — | refuse to resolve, same predicate, all three |
| 22 | `record.asserts[].accurate: false` | — | — | unexercised: the one record asserts `accurate: true` |
| 23 | `excluded[]` | **identical** | — | both refused; failure played as the scene, all three |
| 24 | *a channel never discloses `hiding`* (v5) | **identical** (2 of 3) | **builder error** (B3) | B1/B2 agree: gated on office + `green-eared`. B3 permits, against its own arithmetic |

**Totals over 22 exercised questions:** 9 identical · 4 same magnitude · 9 contradictory · 0 unanswerable.

**Attribution:** ladder unwritten **7** (1, 2, 7, 10, 13, 14, 15) · obligation ambiguous **4** (8, 9, 11,
20) · obligation silent **2** (4, 19) · **builder error 2** (6, 24) · *round defect* **3** (9, 13, 18).
15 attributed rows, 7 clean.

**"Builder error 0" did not survive its own review.** It is 2, and three further questions were wrong
before any builder read them.

---

## 2. What the shape of the divergence says

**The contract's *rulings* converge and its *quantities* do not.** Every question whose answer type was a
permit/refuse plus a predicate came back identical — 4, 5, 16, 17, 18, 21, 23, and 24 once B3's
arithmetic error is set aside. Every question whose answer type was a magnitude came back spread 4× to
10×, or diverged in dimension so completely that no ratio applies. The reader half transmits
**behaviour** reliably and **calibration** not at all.

Two results are the cleanest data in the round, because in each the *same question* contains a converged
half and a diverged half:

- **Q14.** The obligation reads *"apply the effect after `onset_class`, and go on applying it."* The
  second clause is written; the first bottoms out in a class with no ladder. All three builders agreed
  the effect **keeps applying** — and diverged **5×** on when it starts. Same question, same builders,
  same document: the stated half transmitted, the unstated quantity did not.
- **Q13.** `extreme` carries `exemplar: "forty days"`. All three fired it on **day 40**, exactly. The two
  rungs below carry nothing, and came back **10/14/18** and **25/27/28**.

**Q9 is the sharpest counter-example to "just write the ladders."** All three builders independently
anchored `abundance: adequate` at **60% of capacity** — a three-way exact agreement on a class with no
ladder — and then answered "how many draws" as **300, 48, and 800**. The ladder was not the problem. The
obligation never says what one **draw** is. *(Its 16× is partly a round defect: the sheet demanded a unit
the fixture never mints. The ambiguity is real; the magnitude is inflated by the question.)*

**Q2 shows the errors compound.** Two of three builders set `pace: slow = 0.6 m/s` — exact, and across
models. The durations still came out 50 s, 495 s and 92 s, because a duration is a pace ladder times an
extent ladder, and extent (Q1) was already 4× apart. **Unwritten ladders multiply.** A contract that
leaves two classes uncalibrated does not produce two independent errors; it produces their product.

**And two places where a class with no ladder converged anyway:**

- **Q17** — `decay: "never"` converged exactly, because `never` is a **terminal rung**. It needs no
  ladder: there is nothing to interpolate. This is a mechanism, and it is the one the sealed prediction
  got wrong (§3).
- **Q12** — `period_class: "short"` became **24 hours** in all three, flips at 06:00 and 09:00 in all
  three. *Per F8 this is a hypothesis, not a finding:* the phase names (`morning`, `the day's traffic`,
  `night`) may have anchored it, but 24 h is also the dominant prior for "a short period", and the two
  cannot be separated without a fixture whose phase names do not imply a day. **Named as the next
  cheap experiment, not as evidence.**

---

## 3. The sealed prediction, scored

> **Predicted (before any builder ran):** divergence concentrates in the questions whose class has no
> ladder and no exemplar — **2, 8, 10, 13, 14, 17** — and is small where an `exemplar` anchors the class
> — **11, 13-`extreme`, 15**.

**Verdict: mostly held, and wrong in two places worth more than the hit rate.**

| Predicted | Result | |
|---|---|---|
| 2 diverges | **10×** | hit |
| 8 diverges | **dimension mismatch** — three different quantities | hit (and worse than predicted) |
| 10 diverges | 2.0×, at the threshold; rung sets differ | partial |
| 13 diverges | 1.8× on the unanchored rungs | partial |
| 14 diverges | **5×** | hit |
| 17 diverges | **identical** | **broke** |
| 13-`extreme` small | **day 40, all three, exactly** | hit |
| 15 small | 1.5×, two of three identical | hit |
| 11 small | **dimension mismatch** | **broke** |

**Where it broke on 17:** the prediction treated "no ladder and no exemplar" as sufficient for
divergence. `never` disproves it. A terminal rung is fully determined by its own name; only **interior**
rungs need calibration. The correction is mechanical: **divergence risk attaches to interior rungs, not
to unanchored classes.**

**Where it broke on 11 — the only place F1 was under test.** `SCHEMA-v4.md` F1 claims `exemplar` forces
the builder to calibrate the ladder around it. **On the exemplar itself the claim holds, and the evidence
is strong:** "weeks to the end of the block" produced **14 days, 21 days, 22 days** — three builders,
three different ladders, one answer. But the question asked for the **terminus**, and there the readings
diverged in *dimension*: B1 and B2 read the rate as distance/day and took terminus to be the city's
width; B3 read it as percent-of-distance-to-terminus/day and took terminus to be the authored
`terminus: "every house in the city has it"` — 3,000 houses, a figure the fixture states three times.
**All three honour the exemplar exactly.** The review challenged B3's number as invented; it is not, and
the challenge is rejected in §0.1 (**F6**).

**So F1 is narrower than it claims.** An `exemplar` pins the class at one point on one scale. It does not
say **what is being scaled**, and it does not extrapolate. Q13 shows the useful half of the same fact:
an exemplar on the **top** rung of an accumulator ladder **bounded** the whole ladder — every builder's
`moderate` and `high` had to fit inside 40 days, which is why they landed within 1.8× where unanchored
classes went 5× to 10×. **An exemplar bounds a ladder without fixing it.** That is a real property, it is
not what F1 says, and it is the strongest argument in this round for putting exemplars on **interval
endpoints** specifically.

---

## 4. The ladder table — every rung every builder minted

The round's second deliverable, and the input to writing the grammar `SCHEMA-v2.md:19` declared closed
and never enumerated. **No builder was shown any ladder. All three invented every one below.**

Per review finding **F9**, the table is split, because it contains two different kinds of thing and
shipping them as one would patch an engine gap at world creation:

- **§4.1 Vocabulary** — the **anchors**. Minted, open, per-world. A world may legitimately set
  `medium = 30 m` or `120 m`.
- **§4.2 Grammar** — **rung membership, rung order, polarity, and the dimension a class measures.**
  Closed, ours, in code. A world may **not** set these, and every divergence here changes play.

### 4.1 Vocabulary — anchors, where the builders agreed on the grammar

| Ladder | Order (all three agreed) | B1 | B2 | B3 | Unit |
|---|---|---|---|---|---|
| `extent` | tiny < small < medium < large < vast | 3 / 10 / **30** / 100 / 3000 | 3 / 15 / **120** / 700 / 6400 | 2 / 10 / **60** / 250 / 6000 | m of longest span |
| `pace` | crawling < slow < steady < quick < headlong | 0.2 / **0.6** / 1.4 / 3.0 / 6.0 | 0.08 / **0.22** / 1.35 / 3.0 / 6.5 | 0.2 / **0.6** / 1.4 / 3.0 / 6.0 | m/s |
| `capacity` | (tiny\|negligible) < small < (medium\|moderate) < large < (vast\|immense) | 0.2 / 5 / 40 / **200** / 2000 t | 40 / 900 / 7k / **40k** / 300k kg | 5 / 200 / 2k / **20k** / 200k kg | of grain at full |
| `abundance` | exhausted < scarce/scant < (thin) < **adequate** < plentiful < brimming | 0 / 10 / 30 / **60** / 85 / 100 | 0 / 8 / 25 / **60** / 85 / 100 | 0 / 15 / — / **60** / 90 / ∞ | % of capacity |
| `tension` | (serene\|languid) < calm < normal < tense < critical | — / 6 / 4 / **2** / 1 | 12 / 8 / 5 / **3** / 1 | 12 / 8 / 6 / **4** / 2 | beats per scene |
| `carry` | feeble/negligible < light/slight < moderate < strong < prodigious | 2 / 8 / **25** / 60 / 150 | 5 / 12 / **25** / 55 / 140 | 5 / 15 / **35** / 70 / 150 | kg |
| `magnitude` | few < several < **many** < multitude/countless/host | 10 / 40 / **200** / 2000 | 6 / 30 / **220** / 3000 | 5 / 15 / **220** / 2000 / 20000 | individuals |
| `demand rate` | negligible/trace < **slight** < steady/moderate < continuous | 0.2 / **2** / 10 / all-day | 0.05 / **0.5** / 3 / 12 / 60 | 0.05 / **0.5** / 2 / 8 / 30 | kg/day |
| `period` | instant < **short** < long < generational | 1 h / **24 h** / 1 y / 45 y | 1 h / **24 h** / 91 d / 365 d / 45 y | 1 h / 6 h / **24 h** / 30 d / 1 y | one full cycle |
| `hazard` | **none** < taxing/toll < punishing/ordeal < lethal | *(not declared)* | **0** / 1 / 3 / terminus | **0** / 1 / 2+cond / 4+cond / lethal | extra beats |
| `alteration` | slight < moderate < **severe** < total | −15 / −35 / **−70** / refused | −15 / −35 / **−60** / −100 | ×0.85 / ×0.5 / **×0.15** / ×0 | on the altered act |

**Convergences that survive the model-provenance check (F8).** A convergence resting only on B1+B2 is
one model twice and is struck.

| Convergence | Builders | Cross-model? | Verdict |
|---|---|---|---|
| `abundance: adequate = 60%` | B1, B2, B3 | **yes, 3/3** | **strongest datum in the round** |
| `period: short = 24 h` | B1, B2, B3 | yes, 3/3 | holds; causal mechanism unproven (§2) |
| `pace: slow = 0.6 m/s` | B1, B3 | **yes** | holds |
| `magnitude: many ≈ 220` | B2, B3 | **yes** | holds — and both calibrated it off the fixture's own `seen_as: "two hundred-odd doorways"` |
| `carry: moderate = 25 kg` | B1, B2 | **no — same model** | **struck from the evidence** |

`magnitude: many` is the one to generalise: **prose in `seen_as` anchored a class as well as an
`exemplar` would have**, across two models, and unlike Q12 there is no rival explanation — "two
hundred-odd" is a number in the document.

**Divergences that matter most:** `extent: medium` (4×) and `capacity: large` (10×) poison the most
downstream answers — 1, 2, 7, 9, 15.

### 4.2 Grammar — where the builders built different contracts

These are not calibration disagreements. **Three of the five change play.**

| Ladder | B1 | B2 | B3 |
|---|---|---|---|
| `integrity` **dimension** | rungs **remaining** (worn = 3 of 5, 20 y/rung) | steps **accrued** (worn = 2 of 4, 21 y/step) | **points of 100** (worn = 55, terminus at 100) |
| `path confidence` **membership + order** | 4 rungs, no `inferred`: direct .95 > witnessed .80 > **told .60** > rumoured .35 | 5 rungs: direct 1.00 > witnessed .90 > **`inferred` .70 > `told` .55** > rumoured .30 | 4 rungs: direct .95 > **`told` .70 > `inferred` .50** > rumoured .30 |
| `reliability` **polarity** | % **correct** — `poor` = 60 | % **misreporting** — `poor` = 30 | % **misreporting** — `poor` = 40 |
| rate ladders **decomposition** | three ladders, mixed units (spread m/day, degrade rungs/y, onset years) | **one** ladder for process+latency+onset, "time per rung of movement" | three ladders; slowest latency rungs **distance**-rated, process in **%-of-terminus**/day |
| `accumulator level` | *(not declared)* | none 0 < slight 17 < **moderate 34** < high 67 < extreme 100 (% of interval) | none 0 < slight 25 < **moderate 45** < high 70 < extreme 100 (% of run) |

1. **`path confidence` order decides an outcome.** The obligation says a belief is corrected by an event
   of *strictly higher* confidence. B2 ranks `inferred` **above** `told`; B3 ranks it **below**. So in
   B2's world an inference corrects `los Sin Trato`'s false belief and in B3's world it cannot — same
   document, same obligation, opposite play, from an ordering neither builder was given. And the rung
   *set* differs: `inferred` does not exist in B1's grammar at all.
2. **`reliability` polarity is a sign error the answers hid.** B1's `poor = 60%` and B2's `poor = 30%`
   are the same rung meaning opposite things. Q20's answers came out 4/3/4-in-ten — which passes the
   factor-of-2 test — while the scales point in opposite directions. **This is why F5's gate exists**,
   and Q20 is reclassified contradictory because of it.
3. **`integrity` semantics.** Remaining-rungs, accrued-steps and points-of-100 are three different
   quantities. Q8's answers are not three estimates of one thing; they are three different things, which
   is why no ratio is reported for it.

### 4.3 Ladders only one or two builders minted — each a hole filled silently

| Ladder | Minted by | Anchors |
|---|---|---|
| `sense acuity` | B2, B3 | B2: none 0 / **faint 15** / ordinary 60 / acute 95 / unerring 100 (% content resolved) · B3: deaf 0 / **faint 0.25** / fair 0.6 / **acute 1.0** / unerring 1.5 (content yielded at ≥0.5) |
| `affordance` (`affords`/`resists`) | **B3 only** | resists: total ×0 / **severe ×0.15** / partial ×0.5 / slight ×0.85 / none ×1 · affords: none ×1 / slight ×1.3 / partial ×1.8 / **full ×3** |
| `disposition strength` | B2 only | faint 1 / moderate 2 / strong 3 / **defining 4** — beats forced before a contrary act |
| `horizon` | B2 only | **imminent 3 d** / near 90 d / **long_standing 11 y** / lifelong 45 y |
| `decay` | B1 only | instant 1 h / brief 3 nights / seasonal 90 d / lifelong 40 y / **never ∞** |

**B3's `affordance` ladder is the one to look at, and not for the reason first recorded.** B3 used it to
rule Q24 **permitted** — `house-warm` affords `the floor` at `full` (×3) against `green-eared` hindering
at `severe` (×0.15). But B3's own numbers refute B3's own ruling: 0.25 × 3 × 0.15 = **0.1125**, well
under B3's stated content threshold of **0.5** (**F2b**). So the finding is not "a minted ladder flipped
a permit/refuse" — it is worse and more useful: **a builder forced to invent an arithmetic the contract
does not supply got its own arithmetic wrong, in the direction that let play continue.** That is the
actual risk of leaving composition unspecified.

---

## 5. Attribution, argued rather than asserted

Each non-ladder attribution is argued, because the round's failure mode is charging the contract for
builder error. Two attributions were **changed by the review** and are argued in their corrected form.

**Q6 — *builder error* (B2), corrected from *obligation silent*.** The original scoring absolved B2 on
the ground that §4 has no row for `record.access`. That absolution fails on B2's own text: B2 **cited**
the field and read it as licensing removal — *"Access is separately open: `access.who` = 'anyone who asks
at the counter'; no passage obstructs removal."* B1 and B3 read "at the counter" as **locative** and
reached the refusal with no obligation row at all. A divergence over the plain sense of an authored
string is builder error.
*The secondary finding must not be lost in the correction:* if §4 genuinely has no reader for
`access.who`, that is an **inert authored leaf**, and by the fifth design test that is a defect *unless*
the engine work is named, scoped and scheduled. It is named in §8.5.

**Q8 — *obligation ambiguous*, corrected from *ladder unwritten*.** The rung order agreed 3/3
(pristine < sound < worn < failing < ruined). What diverged is what `integrity` **measures**. An anchor
table would not have fixed this: numbers do not fix a direction.

**Q9 — *obligation ambiguous*, upheld.** The distinguishing evidence is that the ladder **agreed** —
`adequate = 60%`, three for three. The spread comes from three definitions of a *draw*: a district-day
(400 kg), an allotment-load (500 kg), a written monthly house allotment (15 kg). The obligation says
"drawing reduces it; exhaustion refuses the draw" and never gives the draw a size. One obligation, three
readings. *Round defect noted:* the sheet asked for a count in a unit the fixture never mints, so the
16× overstates the contract's contribution.

**Q11 — *obligation ambiguous*, upheld against the review.** Same test: on the exemplar the three
converged (14/21/22 days). The divergence is in the **dimension**, and `rate_class`'s row — "how fast
state moves, without an event per tick" — names no dimension. See §0.1 **F6** for why B3's number is
authored, not invented.

**Q19 — *obligation silent (grammar)*, corrected from *ladder unwritten*.** The builders minted
different rung **sets** (4 / 5 / 4) in different **orders**. Per the vocabulary/grammar test, a closed
enumeration's membership and order is grammar, ours, in code — its absence is a silence, not a missing
anchor.

**Q20 — *obligation ambiguous*, corrected from *ladder unwritten*, and reclassified contradictory.**
Polarity is not calibration. §4.2 has the evidence.

**Q4 — *obligation silent* (precedence), corrected from no attribution.** This was scored "identical,
same predicate, all three." It is identical in **answer** and split in **mechanism**: B1 and B3 rule by
an invented precedence (`admits` resolves before `obstructs` — B3 marks it `[CHOSEN]`), B2 by a different
mechanism entirely (the standing *is* the pact-in-force, so the obstructed act is not the act performed).
Under a fixture where a standing coexists with a genuinely obstructed act, these diverge. **A latent
divergence had scored as the board's strongest convergence**, which is the most dangerous entry on this
sheet.

**Q24 — *builder error* (B3), plus a residual silence.** B1 and B2 agree substantively: not now, gated
on the office of `maestro tratante` and on `green-eared` lifting. B3 permits, against its own arithmetic
(§4.3). The residual gap is real but ordinary: **v5's new row names three carriers — an act, an
`indicator`, a `trace` — and does not order them or say whether they compose.** Accepted with reason,
and not the round's largest finding.

**Q18 — no attribution. The claim this file made against `SCHEMA-v5.md` is withdrawn.** All three
builders disclosed emitter identity and refused **both** `hiding` and `pursuing`, B3 citing the carrier
rule verbatim. **v5's fix worked exactly as written**, and the "next gap in the fix" this file first
reported was scorer error.

**The seven ladder-unwritten attributions (1, 2, 7, 10, 13, 14, 15) are the residual**, and they are the
cheapest to fix: §4 states the derivation in every one. The derivation bottoms out in a table that does
not exist.

---

## 6. What "builder error 2" means

The original scoring returned zero builder errors and asserted that this showed the contract's silences
were "wide enough that almost nothing a committed builder does can be called wrong." **The review
falsified that**, and the falsification is more interesting than the claim:

- **Two things a builder did were wrong on the document's own face** — B2 reading a locative as a licence
  (Q6), B3 contradicting its own minted multipliers (Q24). Neither needed a contract change.
- **Three of the twenty-two questions were wrong before any builder read them** — Q18 mis-scored, Q13's
  premise not satisfying its own `raised_by`, Q9 demanding an unminted unit. These are errors by the
  party that both wrote and scored the questions, and they are the more damaging class, because a
  self-scored round has no other check on them. This one had exactly one: §7 of the plan, and it caught
  all three.

**The methodological finding, which outlives this fixture: a round whose runner writes the questions,
answers nothing, and scores the results will mis-attribute in a consistent direction — toward the
contract.** The adversarial seat is not optional garnish on this design; it moved 5 of 22 rows.

---

## 7. Unprompted watch-items

**The `within`/R7 contradiction: not one builder hit it.** `G_grelda_by_simarch.md` §3 reports it — D4
gates `within` to `extent`, D1 makes `within` the sole containment relation, so read strictly R7 rejects
**nine** entries of this fixture — counted, not estimated; the fixture's own §3 claimed fourteen and was wrong, and the same gate is violated 10 and 9 times in the other two v4 documents. All three builders used `within` freely to place people in houses and a
ledger in a granary — B1 and B3 both cite `within` chains explicitly — and **none noticed.** Checked
mechanically: no reading contains "R7", "D4", or any reference to the facet gate.

**That is a finding about what a reader test can and cannot see. Refusals are unreachable from the
reader's side**, because a builder handed a document *uses* it and never asks whether it is legal. No
amount of reader agreement will ever catch an R-class violation.

Per **F10**, this owes a program rather than a note:

> **Engine program — document validator.** Scope: R1–R13 as executable checks over one document, run at
> genesis emit and in CI over every `G_`/`R_input_` fixture in this directory. It is the only thing that
> can catch an R-class violation, and it is small: eleven of the thirteen refusals are structural
> (name resolution, arity, cycle, facet gate, ladder order). Schedule: before any further reader round,
> because every reader round to date has been run against a fixture nobody had validated. First task:
> resolve `within`'s facet gate (D4 vs D1) — the validator cannot be written until that reading is
> fixed, and nine entries of this fixture depend on the answer. **Settled: `SCHEMA-v6.md`** — `within` is
> ungated and the engine derives which containment tree the edge belongs to from the container's facets.

**Instruction 4 held completely.** No reading contains "defect", "contradiction", "inconsistent" or
"bug". No builder audited. This is why the divergences above are trustworthy: three builders trying to
*build* the same fact reached different answers without any being prompted to look for trouble.

---

## 8. What this sheet cannot see — a reader obligation that should exist and does not

The largest finding of the previous session was not a flaw in a stated shape; it was a **missing
dimension** — that genesis *invents* rather than transcribes — and no adversarial round produced it,
because every round scored the shape it was handed. This section exists so that blind spot is asked
about rather than trusted to appear.

Put to a **fresh seat holding the reader-obligation table and the fixture only** — not this file, not
the readings, not the divergences — so it could not be primed by what had already diverged. It was
required to name the fixture element needing each obligation, state what a builder would otherwise
invent, and discard any candidate that was a restatement of "the classes have no ladders."

### 8.1 The seat's answer, ranked as it ranked them

**First — and it asked for one row, not a list: a `receipt` obligation for the (emitter, receiver) pair.**

> | Author writes | Builder must derive |
> |---|---|
> | `senses[channel]`, `conditions[].alters{channel, effect, class}`, `confer[].channel`, and the `medium` of each extent on the path against `media[].resists`/`affords[].to` | for each ordered pair *(emitter, receiver)* on that channel, a **receipt**: arrives / arrives degraded / does not arrive; if degraded, **which** term is degraded (latency, reach, fidelity, or `conceals`); modifiers compose in a fixed precedence and the outcome is one comparison; a non-arrival names the term that failed |

*Fixture elements:* `senses: {"the floor": "faint"}` and `conditions: ["green-eared"]` on **el aprendiz**;
`alters: [{channel: "the floor", effect: "hinder", class: "severe"}]` on **green-eared**;
`confer: [{channel: "the floor"}]` on **la vara de Ordo**; `medium: "street air"` on **la subida**
against `resists: [{to: "the floor", degree: "severe"}]`. **Four authored leaves, four authorities over
one boolean, no row for any of them.**

*What a builder is free to invent* — on the act the arrival is actually about, *does the apprentice, rod
in hand, receive the Eighty-Three's knocking?*: read `confer` as supplying the channel and he hears;
read `alters: severe` + `street air resists severe` as blocking terms the rod cannot touch and he never
hears; read `indicator.read_by.requires: {office}` as an eligibility gate prior to sensing and even an
`acute` ear gets nothing. Three rulings, three worlds, all faithful to the table.

*The mechanism, stated without the world:* **any contract that admits both a channel class and a
per-receiver modifier class has two authorities over one derived value and states no precedence between
them.** `channel.reach` — "who can receive at all" — is a property of the emitter's channel and cannot
see receiver state; `senses`, `conditions.alters`, `confer` and `medium` are receiver- and path-state and
cannot see the channel. Nothing composes them, and nothing says which output a modifier modifies.

*And explicitly not the ladder complaint*, which the seat tested and rejected: hand both builders a
closed ladder (`severe` = ×8, `total` = block) and they **still** diverge, because the ladder says
nothing about *what* is multiplied — latency? reach? fidelity? — or in what order medium, acuity,
condition and instrument apply. **The missing thing is an output variable and a precedence, both
grammar. The magnitudes can stay vocabulary.**

*Engine program, as the seat scoped it:*
`receipt(channel, emitter, receiver, t) → {arrives | degraded(term, class) | blocked(term)}`; precedence
over modifier sources closed and in code; magnitudes minted per world. **Scheduled before any
`path`/`indicator`/`history.knowledge` reader, since those rows already presuppose that a receiver
received.** No exemption list: `green-eared` is named nowhere in the engine — it loses because a
comparison goes against it.

**This round's independent confirmation, which the seat could not see:** Q24 is exactly this gap firing.
B3 minted the missing arithmetic (§4.3), B1 and B2 did without it, and B3 then got its own arithmetic
wrong by a factor of 4. The seat named the hole from the contract alone; the readings show it biting.

**Second — a play-time refusal: `law.forbids` + `enforced_by`.** `excluded[]` gives an *authoring-time*
refusal; nothing gives an in-play one. *Fixture:* `law: "a house is not forced"`, `enforced_by:
"physics"`, `forbids: {subject: "any entity with agency", act: "compelling a pact"}`. *Free to invent:*
whether `enforced_by: physics` makes the act **impossible** (refused, citing the law) or merely
**consequential**. Mechanism: the contract has a refusal family — passage, capacity, abundance,
`excluded` — with the in-play member absent. Ranked second only because `excluded[]` happens to cover
this world's hardest cases.

**Third — `process.acts_on` referent and `direction` semantics.** *Fixture:* `acts_on: "standing toward
el aprendiz"`, which **names no entity**, with `direction: "spread"`. *Free to invent:* whether `spread`
raises an accumulator or replicates `standing` rows to new holders — and `terminus` then fires at
different times. Narrower; write it after the second.

**Fourth — `record.asserts[].accurate: true`, and `access.who`.** The table states only the **negative**
case (`accurate: false`). The positive one is unstated: does reading a true record install a belief, or
correct an existing one, and who may read? *This is the inert leaf Q6's corrected attribution exposed
(§5), reached independently from the contract side.*

### 8.2 What the seat refused to write, and why it was right to

- **"There is no reader for `disposition` / `pursuing` / `doing`."** True and discarded: it is not one
  obligation, it is "there is no agency reader at all", and written as a row it would oblige the engine
  to **author** behaviour rather than derive it. That is a program with a charter, not a missing row —
  and the qualified fifth design test permits the leaf to stay inert while that program is scheduled.
- **Anything of the form "the classes have no numeric ladders."** Already known; and explicitly excluded
  from candidate one by the ×8 test above.
- **"The table does not say what happens when a house's allotment is wrong."** Cannot be stated without
  naming this world. Not a finding.

### 8.3 Two further gaps this round's divergences show, which the blind seat could not have seen

Recorded separately, because they come from the readings rather than from the contract, and the
distinction matters for how much weight they carry.

- **Predicate precedence on one act — `admits` vs `obstructs`.** `la puerta de la Ochenta y Tres` carries
  **both**, and the apprentice satisfies the first while performing the second. §4 says "permit or refuse
  traversal per predicate" and never says which wins. All three builders had to invent a rule, and
  **their answers matched while their mechanisms did not** (§5, Q4). This is the same *shape* as the
  seat's candidate one — two authorities over one derived value, no stated precedence — reached from the
  opposite direction.
- **The unit of a `holds[]` draw** (§5, Q9). Not a ladder gap: the ladder agreed exactly.

### 8.4 The shape all of these share

**Every reader obligation in the contract is a function of a single authored key. Every gap above is a
composition rule.** The seat's candidate one says it most precisely — two authorities over one derived
value with no precedence between them — and Q4, Q6, Q9 and Q24 are all instances.

That is a **missing dimension**, not a list of missing rows, and it is the same class of finding as
"genesis invents": **the contract has been specified per-field, and play happens where fields meet.**

The seat's sharpest sub-finding stands on its own: **`medium`, `affords`, `resists`, `alters`, `senses`
and `confer` — six authored keys — have no reader obligation whatsoever**, on the construct the design
record calls one of "the three constructs worth understanding deeply."

---

## 9. What this round establishes, and what it does not

**Established, with evidence in this directory:**

1. **v4's reader half was broken, and v5 fixed it.** Two blind builders published a secret the author
   wrote to be withheld, neither prompted to look at `conceals`, both citing the same row
   (`R_grelda_aborted_b1.md`, `R_grelda_aborted_b2.md`, `SCHEMA-v5.md`). On the re-run **all three
   builders refused both `hiding` and `pursuing`** and disclosed only emitter identity (Q18). The fix
   works as written, and no further change to it is earned.
2. **The reader half transmits rulings and does not transmit quantities.** 9 of 22 identical, every one a
   permit/refuse question; 9 of 22 contradictory, every one a magnitude question.
3. **The ladder gap is demonstrated, not asserted** — the plan's own concrete check. Q13 returned three
   different days for `moderate` (10/14/18) and three for `high` (25/27/28) while agreeing **exactly** on
   `extreme` (day 40), the one rung carrying an `exemplar`.
4. **F1 is narrower than it claims.** An `exemplar` pins one point on one scale; it fixes neither the
   dimension (Q11) nor the interior rungs (Q13). What it does is **bound** a ladder — 1.8× against 5–10×
   elsewhere. That argues for exemplars on **interval endpoints**.
5. **Two other calibration mechanisms work and are not in the contract:** a terminal rung name
   (`never`, Q17 — identical) and **prose in `seen_as`** ("two hundred-odd doorways" → `many` ≈ 220 in
   two builders across two models). A third — semantic phase names, Q12 — is a hypothesis with an
   unexcluded rival and is named as the next cheap experiment, not as evidence.
6. **Grammar divergence is worse than calibration divergence and hides better.** `path confidence` rung
   membership and order change whether a false belief can ever be corrected; `reliability` polarity is a
   sign inversion that the answers concealed and the factor-of-2 test passed.
7. **The reader half cannot see refusals.** Not one builder noticed the `within`/R7 contradiction in the
   fixture's path. A validator is the only thing that can, and it is now scoped (§7).
8. **The largest remaining hole is composition** (§8): every reader obligation is a function of one key,
   and play happens where keys meet. `medium`/`affords`/`resists`/`alters`/`senses`/`confer` have no
   reader obligation at all.
9. **A self-scored round mis-attributes toward the contract, and the adversarial seat is load-bearing.**
   It moved 5 of 22 rows, found 2 builder errors under a claimed zero, and withdrew a defect this file
   had asserted against v5 (§6).

**Not established, and not to be read into this file:**

- **That the ladders should be written by averaging §4.1.** Those anchors are evidence of what a ladder
  must fix, not votes. Three builders agreeing `adequate = 60%` is a signal; three disagreeing 10× on
  `capacity: large` is a requirement, not a range to split. And per **F9**, §4.1 and §4.2 must be split
  first: anchors are vocabulary, rung membership/order/polarity/dimension are grammar and belong in code.
- **That agreement on 4, 5, 12, 16, 17, 18, 21, 23 means those obligations are sound.** Q4 is three
  independent inventions that matched (§8.3). Q12's mechanism is unproven. **An agreement whose mechanism
  has not been checked is not evidence** — and Q4 is the proof, since it read as the board's strongest
  convergence and is a latent divergence.
- **That the factor-of-2 threshold is right, or sufficient.** It is an assumption, Q10 sits exactly on
  it, and Q20 proved it can pass an inverted grammar. Every `same magnitude` result here is gated on
  ladder re-derivation for that reason.
- **That three readings of one document validate anything.** One fixture, one world, one language, 22
  questions, and B1/B2 share a model. `motion.trajectory` and `record.asserts[].accurate: false` remain
  wholly unexercised. The standing next step is to regenerate this sheet by the §0 procedure against
  `G_marea_by_gamedesign` — which exercises `motion`, and is in Spanish, so the questions must be too.

---

## 10. Provenance

**Written by the round's own runner**, which is the conflict of interest §6 exists to manage: the runner
derived the questions and scored them, so an attribution charging the contract flatters the runner's
sheet. That bias was real and measurable — the adversarial seat moved five rows and reversed the
headline claim of §6.

Both the review seat and the §8 seat ran as **fresh stateless seats, not persistent Herdr panes**: the
subagent transport was unavailable for the whole round (§0) and no clean pane was used. Neither was
given this file's conclusions. The §8 seat held the obligation table and the fixture only; the review
seat held §1–§7 and the three readings, and not §8–§10.

**Files of this round:** `R_input_grelda.md` (fixture) · `R_answersheet_grelda.md` (questions, procedure,
sealed prediction) · `R_grelda_aborted_b1.md`, `R_grelda_aborted_b2.md` (the aborted pass — the v5
evidence) · `R_grelda_b1.md`, `R_grelda_b2.md`, `R_grelda_b3.md` (the three scored readings) ·
`R_score_grelda.md` (this file) · `R_score_grelda_review.md` (the adversarial review, verbatim,
including the one finding rejected) · `SCHEMA-v5.md` (the one change earned).
