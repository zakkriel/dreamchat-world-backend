# R — adversarial review of the scoring

**Seat:** attribution adversary. Written criteria: the handover's §1.2 design tests plus §1.1's
mechanism-first and shorter rules. Framing in the style of `harness/roles/`.

**Brief:** attack the **attribution**, not the answers. Every divergence attributed to the contract that
is really builder error inflates the case for changing the contract, and that is this round's failure
mode. The runner derived the questions *and* scored them, so the bias runs one way.

**Given:** `R_score_grelda.md` §1–§7 and the three readings verbatim.
**Withheld:** §8 (the blind-spot candidates), §9 (conclusions), §10.

**Seat form:** a fresh stateless seat, not a persistent Herdr pane — the subagent transport was down for
the whole round (`R_score_grelda.md` §0) and no clean pane was used. Recorded as the plan requires.

**Outcome: 11 findings. 9 accepted, 1 accepted in part, 1 rejected on evidence.** Two of them corrected
factual errors in the scoring, one corrected its arithmetic, and one withdrew a claimed defect in
`SCHEMA-v5.md`. The disposition of each, and what changed in the score file, is in
`R_score_grelda.md` §0.1. The review is reproduced verbatim below, including the finding that was
rejected.

---

# Findings — attribution adversary

Verbs: **block** = the finding cannot stand as written; **gate** = stands only after the named check; **accept-with-reason** = stands, reason given.

---

**F1 — Q18. BLOCK. The divergence does not exist in the readings.**
§1 records "`pursuing` disclosed 0/3 vs 2/3" and §5 attributes it to *obligation ambiguous*, then derives a v5 defect ("one row of text to fix"). Read the sheets: B2 lists pursuing under **Refused** ("Failing predicate: interior; and `excluded[2]`"). B3 lists it under **DOES NOT LEARN (refused)** ("predicate: interior; reachable only through an act, an `indicator` or a `trace`"). Three of three refuse, and B3 cites the carrier rule verbatim. Q18 is **identical**, attribution **none**. Ambiguity count falls 3 → 2 and the v5 bug is withdrawn.

**F2 — Q24. BLOCK on the classification, and one builder error inside it.**
Two separate defects.
(a) B1 and B2 rule the same thing in opposite headline words. B1: "Refused by every route currently open to him; one route exists and is gated." B2: "Permitted, by exactly one route… Therefore today he can learn that it is not hunger… only after predicates 1 and 2 both clear." Same gates, same present-tense outcome. Scoring the headline word produced a contradiction that the bodies do not contain.
(b) B3's permit is refuted by B3's own minted arithmetic: acuity `faint` 0.25 × affords `full` ×3 × alteration `severe` ×0.15 = **0.1125**, against B3's own stated content threshold **≥0.5**. B3 asserts content is heard. That is **builder error**, internally checkable, no obligation required.
Consequence: Q24 is not 1/1/1, it is 2-agree + 1 error. "The largest single finding of the re-run" and §4.3's "a ladder no other builder minted flipped a permit/refuse" both fall. A residual gap survives — the new row names three carriers and does not order them — **accept-with-reason**, demoted to an ordinary silence.

**F3 — Q6. BLOCK the attribution. Correct attribution: builder error (B2).**
§5 absolves B2 on the ground that §4 has no row for `record.access`. But B2 did not treat the field as absent; B2 cited it and read it as licensing removal: "Access is separately open: `access.who` = 'anyone who asks at the counter'; no passage obstructs removal." B1 and B3 both read "at the counter" as locative and reached the refusal with no obligation row at all. This is a divergence over the plain sense of an authored string, which a competent builder gets right from the document alone. Silence count falls 2 → 1.
Secondary, and it must not be lost in the absolution: if §4 truly has no reader for `access.who`, that is an **inert authored leaf**. Per the fifth design test that is a defect *unless* the engine work is named, scoped and scheduled. **Gate**: either the leaf is readable (B2 erred) or the reader program is named. Both branches move the attribution off "obligation silent."

**F4 — Q8, Q19, Q20. GATE: three attributions are on the wrong ladder rung of the taxonomy.**
The runner's own discriminator (used to move Q9 and Q11 off "ladder unwritten") is: *if the rung order agreed and the divergence is in the dimension, it is ambiguity, not calibration.* Apply it consistently.
- **Q8**: order agreed 3/3 (pristine<sound<worn<failing<ruined); the divergence is remaining-rungs vs accumulated-steps vs points-of-100 — three different **quantities**, as §4.2 itself says. Numbers in a table do not fix a direction. → *obligation ambiguous.*
- **Q20**: §4.2 and §6 both establish that B1's `poor`=60% means *correct* and B2/B3's means *misreporting* — the same rung with opposite polarity. Polarity is not calibration. → *obligation ambiguous*, and the classification moves from **same magnitude** to **contradictory** (see F5).
- **Q19**: builders minted different *rung sets* (B1: 4 rungs, no `inferred`; B2: 5, `inferred`>`told`; B3: 4, `inferred`<`told`). Per the vocabulary/grammar test, a closed enumeration's membership and order is **grammar, ours, in code** — its absence is a silence, not a missing anchor table. → *obligation silent (grammar).*
Effect: ladder-unwritten falls from 10 to 7.

**F5 — The ~2× threshold. GATE: it is applied to answers and cannot see grammar.**
Q20 is the disproof the runner half-noticed and then left in §1 as "same magnitude": 4/3/4-in-ten passes the ratio test while the underlying scale is inverted. A test that scores an inverted grammar as convergence carries no attributional weight. **Gate every "same magnitude" result on re-derivation from the builders' ladders, not their outputs.** Q20 fails and reclassifies as contradictory; Q10 (tense 2/3/4 = exactly 2.0, and 4-rung vs 5-rung ladders) is gated pending rung-set comparison.
Separately: the headline "spread by 4× to 16×" is inflated. Q8 divides remaining-years by points-of-100; Q11 divides a metre-terminus by a percent-of-terminus. A ratio across incommensurable dimensions is not a measurement of disagreement. Report those two as *dimension mismatch*, not as a factor.

**F6 — §3's "the important one" (Q11 / F1's narrowness). GATE.**
The 8× terminus spread is not clean evidence about `exemplar`. B3's 3,333 days rests on "20 of 3,000 houses" — a city-wide house count B3 invents and cites nothing for, while B1 and B2 both used the authored `Grelda.extent_class`. The *dimension* choice is defensible; the *magnitude* is driven by an uncited invention where an authored leaf was available. Strip it and the F1 claim weakens to what Q13 already shows. The conclusion "an exemplar bounds a ladder without fixing it" survives on Q13 alone — **accept-with-reason** — but not on Q11's number.

**F7 — §6's proposed fifth attribution class ("obligation silent on partial raisers"). BLOCK.**
`raised_by: "a day with no grain, no fire and no voices"` is a conjunction. A conjunction is satisfied when its conjuncts hold; that is grammar the contract does not owe. B1 read it correctly and said so. There is no silence — there is a **defective question premise**, authored by the same party that scored it. That is scorer error, and it belongs in a defects-of-the-round column, not as a new gap in the contract. Same column for Q9: the sheet asked for a count in a unit ("a draw") the fixture never mints, then scored the 16× spread against the contract. The obligation's use of the word does owe a definition — **accept-with-reason** that Q9 stays *ambiguous* — but its 16× is a question artefact and must not be cited as a magnitude of contract failure.

**F8 — Unchecked convergences. GATE on three.**
- **Q4** is recorded as "identical, same predicate, all three." It is not. B1 and B3 rule by an invented **precedence** rule (`admits` resolves before `obstructs` — B3 marks it `[CHOSEN]`); B2 rules by a different mechanism entirely (the standing *is* the pact-in-force, so the obstructed act is not the act performed). Three matching answers, two grammars, and a precedence rule nobody was given. Under a fixture where a standing coexists with a genuinely obstructed act these diverge. This is a **missed finding**, not a pass.
- **`carry: moderate = 25 kg`**, listed in §4.1 as "two for three exactly," is B1 and B2 — the two builders §0 says share a model. That is one model twice. **Strike it from the evidence.** (`pace: slow`=B1/B3 and `magnitude: many`=B2/B3 cross models and survive.)
- **Q12 / "semantic anchoring."** The claim that phase names anchored `short` at 24 h is causal inference from a single observation, with an unexcluded rival: 24 h is the dominant prior for "a short period" in any corpus. Untestable without a fixture whose phase names do not imply a day. Until then it is a hypothesis, not the finding that "stops 'write the ladders' from being the whole answer."

**F9 — §4 as a deliverable. GATE on the vocabulary/grammar split.**
§4 proposes one table. It contains two kinds of thing. Anchors (`medium` = 30 m; `many` = 220) are minted, open, per-world — **vocabulary**. Rung membership, rung order, polarity and the dimension a class measures (F4's Q8/Q19/Q20) are closed and ours — **grammar, in code**. Shipping them as one authored table patches an engine gap at world creation, which the fourth design test forbids. Split before anything is written.

**F10 — §7's watch-items. GATE.**
The `within`/R7 conclusion is sound — R-class violations are unreachable from the reader side, only a validator catches them. That is an engine gap and therefore owes an engine program. It is currently emitted as a "watch-item," which is documentation. Name, scope and schedule the validator or the item is inert.

**F11 — §1 totals. GATE (arithmetic).**
Ladder-unwritten rows in the scoreboard are 1, 2, 7, 8, 10, 13, 14, 15, 19, 20 = **10**, not 9; 10+3+2+0 = 15 attributions against 15 attributed rows. The stated total of 14 is wrong, and the error understates the ladder count — but the brief's opening claim ("Nine of 22") inherits it. Correct the count before correcting the attributions.

---

## Verdict

**"Builder error 0" does not survive.** Two questions move: **Q6** (B2 — misread an authored locative as a licence to remove) and **Q24** (B3 — ruling contradicts its own minted multipliers). Both were reachable from the document and the obligations alone; neither needed a contract change.

Three further questions move off the contract without becoming builder error — **Q18** (scorer misread the readings), **Q13** (defective question premise), **Q9** (question demanded an unminted unit) — and these are the more damaging class, because they are errors by the party that both wrote and scored the questions.

Revised counts: ladder unwritten **10 → 7**; ambiguous **3 → 4** (Q18 out, Q8 and Q20 in); silent **2 → 1**, plus Q19 in as a grammar silence; **builder error 2**; **round defects 3**.

The file's own closing sentence — *"the contract's silences are wide enough that almost nothing a committed builder does can be called wrong"* — is the claim I am attacking, and it is false as stated. Two things a builder did are wrong on the document's face, and three more of the twenty-two were wrong before any builder read them.
