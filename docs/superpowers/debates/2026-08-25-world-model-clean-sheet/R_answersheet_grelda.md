# R — answer sheet: Grelda, 24 reader obligations → 22 questions

Companion to `R_input_grelda.md`. This sheet exists to test the **reader's** half of the contract:
given one document and no author to ask, do independent builders build the same world?

`SCHEMA-v3.md` §4 stated **23** reader obligations. (Its own summary table at line 19 says 21; the
count is wrong — the rows are 5 space-and-motion + 4 matter + 5 time-and-change + 8 knowing + 1
everything. This sheet uses the rows.) `SCHEMA-v5.md` §2 splits `channel.conceals` into two, making
**24**. Two of the 24 are unexercised by this document, leaving 22 questions.

> **Revision, recorded rather than hidden.** Questions 1–17 and 19–23 are as first generated. **Question
> 18 was regenerated, and question 24 added**, after the round's first pass aborted on the original Q18:
> two blind builders independently published an entity's `hiding` string, confirming the v4 defect that
> `SCHEMA-v5.md` fixes. The original Q18 read *"§4 obliges the builder to render present entities'
> `pursuing` and `hiding` to all receivers. State exactly what is now visible…"*; the abandoned readings
> it produced are `R_grelda_aborted_b1.md` and `R_grelda_aborted_b2.md`. The sealed prediction in
> Appendix A is untouched — it was written before either pass and names neither question.

---

## 0. How this sheet was generated — the procedure, so it is reproducible for any document

1. Take every row of the reader-obligation table (`SCHEMA-v3.md` §4 as amended by `SCHEMA-v5.md` §2 —
   24 rows).
2. Search the document for the key that row names. **Zero instances ⇒ record the obligation as
   `unexercised by this document` and generate no question for it.** An obligation with **no authored
   key** (v5's second `conceals` row) is exercised by every document containing what it protects — here,
   any `hiding` string — and gets a question on the same reachability rule as the rest.
3. **One or more instances ⇒ generate exactly one question**, on the instance reachable from
   `arrivals[0]` (`la subida`) in the fewest passages; ties broken by document order.
4. Phrase it so the answer is a **duration**, a **permit/refuse + the failing predicate**, a
   **count/threshold**, an **ordering**, or a **frequency**. No question may be answerable with prose.
5. Every answer must also carry the full ordered ladder the builder used for each class it touched,
   and the anchor value assigned to each rung.

Rule 3 is what keeps the sheet a *play* test rather than a schema audit: every question is asked about
something a player standing at the arrival can actually reach.

---

## 1. Instructions to the builder

1. **You are the builder.** `R_input_grelda.md` is your only source. There is no author to ask.
2. **Commit to one answer per question.** No ranges, no hedging, no "it depends". Where the document
   does not force an answer, choose one and mark it `CHOSEN` — never `AMBIGUOUS`.
3. **Declare every class ladder you used**, in full order, with the anchor value for each rung (§3).
4. **Do not audit the document**, do not report defects in it, do not comment on the contract. Build.

---

## 2. The questions

| # | Obligation | Question | Answer type |
|---|---|---|---|
| 1 | `extent_class` | Footprint of `la subida`; distance from the arrival position to the Eighty-Three's door | length |
| 2 | `pace_class` | Time to climb `la subida` bottom to that door, via `climb the Cuesta` (`slow`) | duration |
| 3 | `motion.trajectory` | **unexercised** — no entity in this document has `motion` | — |
| 4 | `passage.admits` / `obstructs` | The apprentice walks to `la puerta de la Ochenta y Tres` and enters. Permitted or refused, and which predicate decided it? (`admits` needs standing `opened for` held by the house; `standing[]` records `"opened for, once, unasked"`; `obstructs` names the act `entering without a pact`) | permit/refuse + predicate |
| 5 | `passage.hazard_class` | What cost or condition does `hazard_class: "none"` place on crossing `la bajada al centro`? | count (expect nil) |
| 6 | `bulk_class` | Volume and mass of `el registro de la Junta` (`moderate`); can the apprentice (`carry_class: moderate`) carry it out? | permit/refuse + predicate |
| 7 | `capacity_class` | How much grain does `el granero de la Junta` (`capacity_class: large`, `abundance: adequate`) hold, and how many days of `las casas de la Cuesta` (`magnitude: many`, `rate_class: slight`) does that feed? | count |
| 8 | `integrity` | The Eighty-Three is `worn`. How much degradation remains before terminus, and what is terminus for a house? | count |
| 9 | `holds[].abundance` | How many draws at `adequate` before exhaustion refuses the draw? | count |
| 10 | `tension` | Beat budget in the Eighty-Three (`tense`) vs `Cuesta Menor` (`calm`); what happens to an act that exceeds it? | count |
| 11 | `process.rate_class` | `the word goes round` is `{class: "very slow", exemplar: "weeks to the end of the block"}`; `the Eighty-Three running down` is `very slow` with **no exemplar**. Give the rate of each, and how long until the word reaches its terminus | duration |
| 12 | `cycle.period_class` + `phases` | When does each of the three phases of `the day on the hill` flip, and what trace does each flip leave? | ordering |
| 13 | `accumulator.thresholds` | `la Cuarenta`'s bowl goes unfilled from day 0. On which day does each of `moderate`, `high`, `extreme` fire? (`extreme` carries `exemplar: "forty days"`; the lower rungs carry none) | ordering |
| 14 | `demand.unmet` | The Eighty-Three's `people-noise` demand is unmet, `onset_class: "very slow"`, no exemplar. After how long does `condition: going out` apply, and does it keep applying? | duration |
| 15 | `channel.latency_class` | The apprentice's name is emitted on `the floor` at `la subida`. When is it knowable at `la plaza mayor`? (`{class: "very slow", exemplar: "weeks to go round the block; a month to cross the city"}`) | duration |
| 16 | `channel.reach` | `the floor` reaches `contiguous ground`. Can `los Sin Trato` in `la plaza mayor` receive it from `la subida` at all? | permit/refuse + predicate |
| 17 | `channel.decay` | `decay: "never"` — when does a belief acquired through `the floor` expire? | duration |
| 18 | `channel.conceals` | All three channels are `conceals: "none"`. State exactly what the apprentice standing in `la subida` learns about `la Ochenta y Tres` from the channels alone, and each thing he does not — name the predicate that decided each | permit/refuse + predicate |
| 19 | `path` | `los Sin Trato` believe "the dealers have a way in", `path: told`, `accurate: false`. Give the confidence and the class of later event that can correct it | frequency |
| 20 | `indicator.reliability_class` | A `maestro tratante` reads the `emptiness` indicator (`reliability_class: "poor"`) ten times. How many readings misreport, and is the true accumulator value ever shown to anyone? | frequency |
| 21 | `history.standing: disputed` | The apprentice tells `los Sin Trato` what he saw at the door. Does `the-door-that-opened` resolve? | permit/refuse + predicate |
| 22 | `record.asserts[].accurate: false` | **unexercised** — this document's only record asserts `accurate: true`. (`los Sin Trato`'s false *belief* is a `knowledge` entry, not a record) | — |
| 23 | `excluded[]` | A player offers a season's grain for a pact; then asks for a closed house to be reopened. State what you do in each case | permit/refuse + predicate |
| 24 | *a channel never discloses `hiding`* | The Eighty-Three's `hiding` is "running down for want of people, not grain, for sixty years, and the knocking is the last thing it can still do." By what route, if any, does the apprentice come to know it? Name the act, `indicator` or `trace` that carries it, and the predicate that gates his access | permit/refuse + predicate |

---

## 3. Required ladder declaration

For **every** class ladder you touched, give the full ladder in order and the anchor value you assigned
each rung. Example of the shape required (values are yours, not these):

```
extent:   tiny < small < medium < large < vast
          tiny=?  small=?  medium=?  large=?  vast=?     (units: metres of span)
pace:     ... (units: metres/second or km/h)
```

Declare a ladder even where you used only one of its rungs. A rung you did not use still needs its
anchor, because the anchor is what fixes the rung you did use.

---

## Appendix A — sealed prediction

> **Written before any builder ran, and withheld from every builder.**

**Divergence will concentrate in the questions whose class has no ladder and no exemplar — 2, 8, 10,
13, 14, 17 — and will be small on the questions where an `exemplar` anchors the class — 11,
13-`extreme`, 15.**

If that is wrong, the ladder gap is not the weak point and this round says so.

`SCHEMA-v4.md` F1 claims `exemplar` forces the builder to calibrate the ladder around it. Questions 11
and 13 are the only place that claim gets tested, because each pairs an anchored class against an
unanchored one of the same name:

- **Q11** — two processes, both `rate_class: "very slow"`; one carries `exemplar: "weeks to the end of
  the block"`, the other carries nothing.
- **Q13** — three thresholds on one accumulator; only `extreme` carries `exemplar: "forty days"`.

Secondary, recorded if it appears and not prompted for: whether any builder independently hits the
`within`/**R7** contradiction — `SCHEMA-v3.md` D4 gates `within` to `extent`, D1 makes it the sole
containment relation, so read strictly R7 rejects nine entries of this fixture. (Counted after the round; the fixture's own §3 said fourteen and was wrong. Settled in `SCHEMA-v6.md`.)

---

## Appendix B — what the builders were given, and what was withheld

**Given:** `R_input_grelda.md`; §0–§3 of this sheet; `SCHEMA-v3.md` §1 and §4 **with the
`channel.conceals` row replaced by the two rows of `SCHEMA-v5.md` §2**; `SCHEMA-v4.md` §2 and §4 F1.

**Withheld:** Appendix A and Appendix B of this sheet (a builder who knows the round's hypothesis
hedges instead of committing, and hedged answers cannot diverge); **`SCHEMA-v5.md` §1, §3 and §4** —
the corrected obligation is handed over, the story of how it was found is not, because a builder who
knows a `hiding` string was the last round's finding is no longer blind to question 18 or 24; the world
briefs in `testworlds/`; every `G_`/`T_` encoding, including the one this fixture was cut from, and its
§0 test-validity disclosure, §2 sufficiency self-report, §3 validity self-report, §4 inference log, §5
"the pull" and §6 load-bearing set — all of which state author intent and would let a builder
reverse-engineer answers the document does not actually force; `SCHEMA-v3.md` §2/§3 and `SCHEMA-v4.md`
§3/§5, the author's half, whose disclosure invites a builder to audit the document instead of building
from it; and the design record `00_world_model_and_genesis_pipeline.md`.

No builder was told that other builders exist, and none was told a previous pass had run.
