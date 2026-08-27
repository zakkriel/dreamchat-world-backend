# Handover — World Genesis (stage 1)

**Date:** 2026-08-27 · **From:** the session that ran the reader-half test, wrote v5–v7, audited the
engine, and grilled out the world-identity design. **Read this whole file before doing anything.**

---

## 0. MANDATORY — the four things that will get you stopped

These are not style notes. Every one of them is a scar from this session, and the founder caught all of
them. He will catch you too.

### 0.1 Never turn an observation into a gate

I proposed three checks and all three died in session:

- *"Every identity rule must produce at least one entity."* Killed by one counter-example: *"the water at
  night turns lighter and everything on it sinks."* That rule demands **no entity**. The check would
  either pass vacuously or refuse a good world.
- *"Rank rules by weight."* Solved a truncation problem I had already disproved with arithmetic.
- *"Exclusions get a hard gate."* They can't. Checking *"no institution may promise a schedule"* is a
  model judgement. It is a **review**, and calling it a gate was theatre.

The founder's words, and they are permanent: *"is once again the validation of something that is
validating nothing"* and *"created 300000 tests that test nothing but it gives the impression of
safety."*

**Before you propose any check, ask: what specific bad thing does this catch, and could it pass while
that bad thing is present?** If yes, drop it. A report a human reads beats a gate that goes green on
nothing.

### 0.2 Verify every count. Never assert one from a filename or from memory

I got this wrong **three times in one session**:

- Claimed 14 containment violations because a document said so. **It was 9.** I had repeated it into two
  other docs.
- Claimed the engine had 3 class→number conversions. **It has 4**, and two of them fail open — which was
  the actual finding and I'd missed it.
- Claimed the archived frontend's mockups existed nowhere else, and built a hard constraint on it. **All
  16 were byte-identical by sha256** to files in the backend, and two live documents already forbade
  vendoring them.

The pattern: I compared **filenames** and asserted **uniqueness**. Checksum, count, and grep. Then state
the number.

### 0.3 Report in product terms, not in validation terms

I once told the founder *"all three gates pass"* while his product was returning **502**. I filed the
outage as a footnote.

Count what shipped, not what passed. If a session produced documents and no product, say so plainly.

### 0.4 Stay in the stage you are in

The founder has said this many times and I drifted twice: *"Im tired of saying we are doing this in
stages."*

**We are in WORLD GENESIS.** Input → understand → transcribe → fill → emit the completed document. That
is all. World creation, the engine, the play loop, gameplay tuning, backstage updates, dead columns,
pressure numbers — **all of it is a later stage.** If you find something there, file it in
`docs/open-spec-items.md` and move on. `SPEC-036` is the precedent.

---

## 1. The goal, in the founder's terms

A person writes a **short prose description** — a few hundred words. The system understands it, invents
the rest, and emits one document. That document goes to world creation, and eventually a person plays a
character inside a world worth being in.

**Genesis is stage 1 and its output is a document. Nothing else.**

The product's own words, worth holding: *"A world that lives with you, not for you."* The player is part
of the world and **not its centre by default**. Success is emotional — *"I remember this world. This
world remembers me. Time changed things. I can return to it and still believe in it."*

The single best test in the corpus, from `A_world_by_simarch.md`:

> **"If every character froze, what still happens, and how does a player find out?"**

And the bar for a world being finished, from `SCHEMA-v4.md:62-63`:

> **"Every boundary a player reaches must be a fictional boundary, never an authoring boundary."**

That one is load-bearing because **the player cannot tell the two apart.** A locked door and an
unauthored edge both read as *"I can't go there"* — so one authoring boundary retroactively makes every
real refusal suspect.

---

## 2. Read these, in this order

| File | What it gives you |
|---|---|
| `docs/00_strategy/01_product_vision_and_promise.md` | The intention. Seven pillars. **Read it first — I designed against the mechanics for a full day before reading this, and it reframed everything.** |
| `docs/00_strategy/02_poc_scope_and_success_criteria.md` | The PoC question and the 11-row validation matrix |
| `docs/30_architecture/world_model/03_world_identity_and_the_understanding_pass.md` | **Genesis step 2, settled.** The design you are continuing. Its appendix holds the 13 open questions |
| `docs/30_architecture/world_model/02_what_makes_a_world_alive.md` | What alive decomposes into, and the distance to it |
| `docs/30_architecture/world_model/00_world_model_and_genesis_pipeline.md` | The nine-stage pipeline, the eleven frozen facets, and §3 — why v1–v7 each died |
| `.../debates/2026-08-25-world-model-clean-sheet/SCHEMA-v3.md` §1,§4 · `v4` · `v5` · `v6` · `v7` | The contract. v3 §4 is the reader half; v5/v6/v7 are the deltas this session earned |
| `docs/10_prds/prd_world_creation.md` | The shipped genesis PRD — **and it contradicts the contract, see §5** |
| `docs/30_architecture/world_model/01_engine_capability_audit.md` | 3 working / 12 partial / 9 absent, cited to file:line. Reference it; do not re-derive it |
| `.../testworlds/` (12 files, several in Spanish) | The real product input. Four worlds at three densities each |

---

## 3. Closed decisions — do not reopen

Each was settled by the founder in a grilling session. Reopening one loses a day.

1. **Identity is inferred invention rules, not a description.** A description is a claim about what worlds
   are usually like, which `GA-2`/`GA-3` forbid and `prd_world_creation.md:177` bans by name.
2. **One condition, one bargain derived from it, many faces.** Not multiple bargains — five of equal
   weight produces mush.
3. **Every rule carries a `therefore`.** An assertion without one gets acknowledged and ignored.
4. **Rules have kinds. Constraining rules run before generative ones.**
5. **Origin decides what play can change.** A named cause makes a rule contingent; no cause makes it
   axiomatic. Identity is emitted versioned, immutable during genesis.
6. **Never ask the author a question in the system's vocabulary.** Ask in fiction, mine the answer.
   Multiple choice with an open option. Three attempts, then a flat world is a legitimate outcome and gets
   recorded as a rule so nothing invents a dark secret in the cosy village.
7. **Twenty universal human functions, phrased as functions, never professions.** They test the identity,
   build the ordinary life the pressure doesn't demand, and become the reference for content minted later.
   *"None, and here's why"* is a valid answer.
   **There is ONE test and this is it.** Doc 03 §3.11 is the *criterion* applied to these twenty answers —
   *could this answer exist in any other world?* — **not a second mechanism.** The carpenter is one
   illustration of one of the twenty (*"who repairs and makes things"*), and there is no separate
   "carpenter test". An earlier draft of doc 03 read as two things and misled the first agent to inherit
   it; §3.11 is corrected. If you find yourself planning a carpenter test alongside the twenty functions,
   you have read a stale draft.
8. **Voice is imitable prose, not adjectives.** Shown to the author before any content exists, and
   **rewritable** — a rewritten sample is authored voice, not inferred.
9. **Content register is a demand with a therefore, not a permission level.**
10. **Filling works rule by rule: code schedules, the model interprets, every element tagged with its
    cause.** The tagging is the prize — it yields the correction graph all three existing encodings lack.
11. **Volume is unbounded.** A rule produces a *space of positions*, not an entity. The claim is *nothing
    invented that cannot answer to the identity* — **not** *nothing invented without a rule asking for
    it*, which made worlds narrow and was corrected.
12. **Reviews, not gates.** And a reviewer must never be the generator, nor see the generation context.

---

## 4. Start here — the immediate next question

**Q1 from the appendix: does rule-by-rule filling fit the call budget?**

`prd_world_creation.md:26-27` sets the Fast lane at **p50 ≤ 90 s, p95 ≤ 180 s, p50 ≤ $0.25 per world**,
and `:32` says *"one LLM-authored world document per build."*

The mechanism in doc 03 §7 wants one call per rule — roughly nine identity rules plus twenty universal
functions is **up to 29 calls**. About 3 seconds and $0.008 each.

Either the calls batch, or they parallelise, or the budget moves, or §7's mechanism does. **Everything
else in genesis waits behind this**, because if the budget wins, §7 needs rebuilding and everything
downstream moves with it.

Do not guess. The founder decides. Bring him the options with a recommendation and the arithmetic.

Then the other four that could change a decision: the Fast lane has **no interview** at all (so §6's
question protocol only exists in Custom); genre-reference input (*"like Dune but underwater"*) collides
head-on with GA-2 and will arrive on day one; nothing says **when filling stops**; and whether identity
travels **inside** the emitted document or beside it.

---

## 5. Two blockers you will hit immediately

**The contract has no machine representation.** No schema file, no Go type. `SCHEMA-v2` through `v7` are
prose in markdown. **You cannot emit a completed schema that is not an artifact**, so step 5 of genesis is
blocked until this exists. The plan for it is
`docs/superpowers/plans/2026-08-27-increment-1-landing-contract.md`, which chose Go-type-as-source.

**The genesis PRD describes a dead schema.** `prd_world_creation.md` commits to
`places / cast / objects / ways` — structurally **v1 of the world model, the version that died**.
`world_model/2-7` replaced it with one recursive `entities[]` and eleven composable facets, and nothing in
the PRD acknowledges the replacement. Two live documents disagree about what a world *is*. Settle it
before building against either.

Three more contradictions, all live: `tagline` is *authored fiction the service never composes* (PRD) vs
*derived from premise* (`SCHEMA-v7` §7); the seat emits *"no number of any kind"* (PRD AC-7) vs `exemplar`
*"is fiction and may contain a number"* (`SCHEMA-v4` F1); and confirmation as *"the only correction
window"* against *"canon is append-only"*.

---

## 6. How the founder works

- **He will out-design you on shape.** Several of his corrections this session were better *and* smaller
  than what I proposed. When he pushes back, he is usually right — check before defending.
- **He wants product, not process.** *"we are building a product not a validation that sounds super safe
  but it does nothing."*
- **He will not accept hedging or padding.** *"stop with the apologies that gives me nothing."* Lead with
  the fact, the decision, or the risk.
- **He asks you to think, not to ask.** Twice he redirected me: *"the proper step is not to ask that from
  the User, but to infer the question and create an answer."* Bring an answer with your reasoning; put
  only genuine forks to him.
- **Plain language.** He stopped me for jargon: *"what the flying fuck does that mean?!"* No `Q4`, no
  `B3`, no *convergence*, no *attribution*. Say "the door question", "the third AI", "the document".
- **He will tell you when you drift.** Believe him immediately.
- **Do not chop your own creativity with defensive design.** *"do not chop off your own creativity by
  adding dumb validations that are not needed, or legal bullshit, or assumptions that are just laziness
  covered."*

---

## 7. What is already done, so you don't redo it

**Merged to main:** the reader-half test round (`R_*` files — three blind builders, one document, 22
questions); `SCHEMA-v5` (the `conceals` fix — v4 died on it); `SCHEMA-v6` (containment — `within` ungated,
28 violations across three documents, none an error); `SCHEMA-v7` (eight capabilities reclaimed);
the engine audit; the nine-increment roadmap; `AMENDMENT.md` (increment 1's scope after a three-seat round
returned 23 blocking findings); increment 1's 9-task plan; and the narration change that finally tells the
narrator which world it is in.

**Open PR:** #124 — this handover's two design documents plus `SPEC-036`.

**Unmerged branch in `dream-weaver-visuals`:** doc consolidation and three Opus reviews (architecture, UX
with 16 screenshots, and a file-by-file consolidation that corrected four of my claims).

**Known and filed, not to be re-found:** `SPEC-036` (rule enforcement — the exist/happen split);
`Los Andantes` has never been encoded at any tier, and the contract carries two primitives designed for it
that nothing has ever exercised — the cheapest real test of whether the contract works.

---

## 8. The one thing I would do differently

I read the mechanics for a full day before reading the product vision. Everything I built in that day was
technically defensible and pointed slightly wrong, and the founder felt it long before I could see it.

**Read §2's first two files before you touch anything else.** The contract is a means. The world is the
product.
