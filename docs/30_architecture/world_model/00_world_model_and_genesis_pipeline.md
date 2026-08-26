# The World Model and the Genesis Pipeline

**Status:** design record, 2026-08-25. Nothing here is implemented.
**What this is:** the contract by which any world is described to a system that will build it, and the
pipeline that produces and consumes that contract.
**Working record:** `docs/superpowers/debates/2026-08-25-world-model-clean-sheet/`

---

> ## ⚠️ ON THE EXAMPLES IN THIS DOCUMENT
>
> **Every example here is illustrative ONLY.**
>
> - **No example is a requirement.** Nothing in any example must appear in any world.
> - **No example is a template.** Do not produce worlds shaped like these.
> - **No example may be hardcoded** — not as a default, not as a fallback, not as a fixture, not as a
>   prompt exemplar that becomes the shape of every output.
> - Examples exist to show **what a construct means**, never what a world contains.
>
> Every construct below is illustrated with **several unrelated worlds on purpose**, so the *shape* is
> visible and no single instance is copyable. If you find yourself reaching for one of these worlds
> because it is the one you have seen, that is the failure this warning exists to prevent.
>
> The system must never learn what a world is "usually" like. That is the deepest constraint in this
> design and every example is a threat to it.

---

## 1. The pipeline

```
WORLD GENESIS ─────────────────────────────────────────────────────┐
                                                                    │
  1. Input          the user's prose; optionally a guided           │
                    questionnaire that only asks what the           │
                    brief left genuinely open                       │
       │                                                            │
       │  ◄── ask-loop: defining reveals gaps; gaps become          │
       │      questions; answers re-enter definition                │
       ▼                                                            │
  2. Definition     understand it AS A WORLD. Establish this        │
     & sorting      world's own vocabulary, its rules, what         │
                    exists, who is there, what is forbidden.        │
                    Everything STATED or REFERENCED is realised.    │
       │                                                            │
       ▼                                                            │
  3. Filling        INVENT what the stated content entails but      │
                    does not say — until the world is OPEN.         │
                    Every invention downstream of something         │
                    stated. Nothing downstream of genre.            │
       │                                                            │
       ▼                                                            │
  4. Confirmation   show the user the LOAD-BEARING inferences.      │
                    The only correction window that exists.         │
       │                                                            │
       ▼                                                            │
  5. Validation     valid (obligations + refusals) AND              │
                    sufficient (openness). Fails ⇒ refused whole.   │
       │                                                            │
       ▼                                                            │
  6. WORLD DEFINED  one document. The contract. ─────────────────────┤
                                                                    │
WORLD CREATION ─────────────────────────────────────────────────────┤
                                                                    │
  7. Transposition  every element of the defined world becomes      │
                    engine state — entities, events, knowledge,     │
                    physics vocabulary, schedules.                  │
       │                                                            │
       ▼                                                            │
  8. WORLD RUNNING                                                  │
                                                                    │
WORLD GROWTH (during play) ─────────────────────────────────────────┤
                                                                    │
  9. Lazy filling   the world keeps inventing where players reach   │
                    past what exists, bounded by the document's     │
                    own rules and its `excluded` list.              │
                                                                    ┘
```

### Why the split is the point

**The document is the interface.** Genesis produces it; Creation consumes it; neither knows the other.
Consequences that make this worth doing:

- how worlds are elicited can change completely — prose, questionnaire, import, an external tool —
  without the engine knowing
- the engine can change completely without re-authoring a single world
- a defined world can be **re-transposed** into a later engine
- the contract is testable on its own, before either side exists

### Three things I would add to the split as stated

1. **The ask-loop is not linear.** Defining reveals gaps; some gaps are worth asking about rather than
   inventing. Definition and input iterate.
2. **Validation is an explicit gate**, not an implicit hope. A document that is valid but *thin* passes
   the first bar and fails the second; both are checked, and failure refuses the whole document.
3. **Growth during play is a third phase.** A world is a strong seed, not a finished object — so the
   contract must also say what may be invented *later*, under what constraint, and what can never be
   added after genesis. Genesis is not the only author; it is the only *initial* one.

---

## 2. Principles

These are why the design is the shape it is. They are not negotiable within it.

| # | Principle |
|---|---|
| P1 | **Grammar is closed and ours; vocabulary is open and the world's.** We own the kinds of thing and the class ladders. The world invents every name, every medium, every way of moving, every sense channel. |
| P2 | **The author picks a class; the engine owns the number.** No document ever states a quantity the engine computes on. `slow`, `vast`, `severe` — never metres or seconds. |
| P3 | **Prevention emerges from comparison, never from an exemption list.** A barrier states what it resists; a mover states what it is. Passage is the comparison. There is never a list of who is exempt. |
| P4 | **Knowledge is per-holder and earned**, carries the path by which it arrived, and may be false. |
| P5 | **The system never learns what a world is "usually" like.** No genre taxonomy, no template library, no identifier traceable to a single example world. |
| P6 | **Invention is downstream of statement.** Something exists because something stated required it — never because worlds usually have one. |
| P7 | **Every boundary a player meets must be fictional, never an authoring boundary.** |

---

## 3. Version history — what each version was and why it died

Recorded because each death is a constraint on what comes next.

### v1 — sections
Nineteen top-level sections; an entity's kind was **which array it lived in** — `places`, `things`,
`people`, `collectives`.

**Died on premises.** Four externally-authored world briefs broke it in one round, all the same way: a
house that is alive is a place *and* an agent; a creature the size of a country carrying cities is a
thing *and* a place *and* in motion; a shared dream exists only while enough people sleep. Each had to be
authored twice under one name.

> *"The schema encodes what a thing is by which array it is in. That is a closed ontology in the clothes
> of an open one."*

### v2 — facets
One recursive `entities[]`; kind expressed as **composable facets**. Four fixed kinds → one. `ways` and
`stocks` absorbed.

**Survived every world — no new top-level section needed for any of them.** Died on **ambiguity**: two
independent encoders, one brief, produced structurally different documents. Containment had three
readings. `layers[]` was a stub that looked solved.

> *"An empty section that looks solved is worse than a missing one: I could not tell whether my encoding
> was wrong or merely unguided, and neither will the other encoder."*

### v3 — the contract
Added what a shape lacks: **author obligations, refusals, and reader obligations.** Disambiguated
containment. Deleted `layers[]` — a concurrent reality is an ordinary entity. Froze the facet list.

**The controlled re-test worked**: the pair that diverged hardest now produced identical structure.
Died on **four named fields**, one of which three encoders found independently in three different
worlds — quantities that are *both* canon-fact and engine-computed, forced by the rules into being
stated twice and meaning once.

### v4 — the generative contract
The realisation that reframed everything: **genesis invents.** v1–v3 were tested for *transcription*;
the actual act is *generation*. Added provenance, the inference discipline, and **sufficiency** as a
second bar distinct from validity.

**Passed its first generative test**: from 400-word briefs alone, with the authors' own 3,000-word canon
documents held back, generation landed on the load-bearing structure of those canons — including
independently reproducing two of the authors' own *negative canon* entries — and refused genre defaults
under explicit pressure.

**Died on the reader half.** The first round that actually ran it — two builders, one v4 document, a
fixed sheet of play questions, neither builder told what was being tested — stopped on its first
question. v4's reader half **conflated channel-level identity with mind-level interiority**: the
`channel.conceals` obligation was named for what identity a channel withholds, and then also obliged the
builder to disclose every present entity's `hiding`. Since the ordinary senses conceal nothing in the
normal case, both builders independently published a secret the author had written to be withheld, and
both were correct to. **O4 obliged the author to withhold; §4 obliged the builder to publish.** No
author-side round could have found it, because no author-side round reads.

### v5 — the reader-half correction
`channel.conceals` split into two obligations: a channel discloses **identity** only, and **never** an
entity's `hiding` — an interior becomes knowable only through an act, an `indicator`, or a `trace`.
`pursuing` stays visible, disclosed by the entity's `doing` rather than by the channel. One change, no
other. The evidence is `R_grelda_aborted_b1.md` and `R_grelda_aborted_b2.md`; the delta is `SCHEMA-v5.md`.

**The fix is confirmed.** The round re-ran with three fresh blind builders against the corrected
obligation: all three disclosed only emitter identity and refused both `hiding` and `pursuing`.

**Not yet dead, and now measured.** The re-run scored 22 questions across three builders
(`R_score_grelda.md`). Its results, in order of weight:

- **The reader half transmits rulings and not quantities.** 9 of 22 identical — every one a
  permit/refuse question. 9 of 22 contradictory — every one a magnitude question.
- **The ladder gap is now evidence, not assertion.** On one accumulator whose top rung carries
  `exemplar: "forty days"`, all three builders fired `extreme` on day 40 exactly, and returned
  10/14/18 and 25/27/28 for the two rungs below it, which carry no exemplar.
- **v4's F1 is narrower than it claims.** An `exemplar` pins one point on one scale. It fixes neither
  the dimension being measured nor the interior rungs — but it does **bound** a ladder (1.8× spread
  against 5–10× elsewhere). That argues for exemplars on interval endpoints.
- **Two calibration mechanisms work and are not in the contract:** a terminal rung name (`never`
  converged exactly) and **prose in `seen_as`** ("two hundred-odd doorways" → `many` ≈ 220, in two
  builders on two different models).
- **Grammar divergence is worse than calibration divergence and hides better.** Builders minted
  different rung *sets* and *orders* for `path` confidence — deciding whether a false belief can ever be
  corrected — and inverted the polarity of `reliability` while their answers still passed a
  factor-of-2 agreement test.
- **The reader half cannot see refusals.** No builder noticed the `within`/R7 contradiction sitting in
  the fixture's path. Only a validator can; it is scoped in `R_score_grelda.md` §7 and is now the
  blocking prerequisite for further reader rounds, since none has yet run against a validated fixture.
- **The largest remaining hole is composition, and it is a candidate v6.** Every reader obligation is a
  function of a *single* authored key, and play happens where keys meet. `medium`, `affords`, `resists`,
  `alters`, `senses` and `confer` — six keys deciding whether a receiver receives anything — have no
  reader obligation at all. A blind seat proposed one row: a `receipt` obligation over the
  *(emitter, receiver)* pair.

**Method finding, kept because it will recur:** a round whose runner writes the questions and scores
them mis-attributes toward the contract. The adversarial seat moved 5 of 22 rows, found 2 builder errors
under a claimed zero, and withdrew a defect the score file had asserted against v5
(`R_score_grelda_review.md`).

### v6 — containment is one authored key, and the engine derives the tree
`within` removed from the `extent` facet's key list and made **ungated**. D4 and R7 continue to hold for
every other facet key. In its place, a new reader obligation: **which containment tree an edge belongs to
is derived from the container's facets** — a container with `extent` *places* what is inside it and the
edge carries geometry; a container with `matter` and no `extent` *bears* it and the edge carries mass and
volume. One change. The delta is `SCHEMA-v6.md`.

**Why it had to go.** `SCHEMA-v2.md:28` gated `within` to `extent`; D4 makes a facet key without its
facet an R7 refusal; and D1's own worked example is *"a person in a room."* **The sentence establishing
the rule broke the rule.** Counted rather than estimated: the gate is violated **28 times across the
three v4 documents** — 9, 10 and 9 — and every instance is ordinary containment: a person in a house, a
rod in a house, a ledger in a granary, a crowd in a square. Not one is an error. Three blind readers used
`within` that way throughout and none noticed, which is the same finding as "the reader half cannot see
refusals."

**The distinction the gate reached for is real, and is now the engine's job.** `core/api/tier1.go:16-20`
keeps three separate containment edges because each feeds different arithmetic — `parent_location_id` for
distance, `contained_by` for mass and volume, `location_id` for placement. An author writing "the rod is
in the house" should not have to know which one the engine files it under.

**Deliberately left open:** whether a container declaring **both** `extent` and `matter` — a living
house, a train, a floating island — aggregates its contents into its own mass. Placement is settled
(`extent` wins: you are *in* the house, never carried by it). Aggregation needs increment 2's resolved
quantities and increment 4's composition rules, and is filed as increment 2's first design question.

**Reader-obligation count: 24 → 25.** Unblocks the validator, which was the reason to write it now.

### v7 — the stranded capabilities, reclaimed
Seven behaviours `world_genesis/1` already enforced were silently dropped across the `world_model`
rewrite — never removed on purpose, just never carried forward. `tension` went from required to optional
and its absence now gives an infinite beat budget, the SPEC-030 regression again; nothing obligates a
single root `extent`, though R8 already makes containment a tree; `arrivals[]` obligates presence but not
the three-candidate, one-`chosen` shape the old schema enforced; the machine-shaped-name guard with a
logged production incident behind it (the Ironmoor breach) had no `world_model` refusal at all; two
arrival-floor checks the engine runs today — nothing leads out of where the player starts, and someone is
there to meet them — existed only as unchecked sufficiency prose (S1, S4); and no array in any
`world_model` version bounds its length, so the tick-ladder assertion and per-build token cost both lost
their ceiling. `world.tagline` and `world.ornament`, which structurally gate cover-art commissioning, had
no `world_model` source at all.

**Found by reading the engine the contract is meant to replace**, not by design review:
`docs/superpowers/debates/2026-08-26-landing-contract-retarget/FINDINGS_contracts.md` (C11–C18), during
the round that retargeted the landing contract at `world_model/6`. Five became new author obligations and
refusals; two — `tagline`, `ornament` — are derived by the runner instead, with the cost of that choice
(a line the founder never approved, gating a purchase the founder used to structurally control) stated
rather than hidden, and made reviewable at increment 8's amendment surface. The delta is `SCHEMA-v7.md`.

**Counts.** Author obligations: 11 → 14. Refusals: 13 → 17. Reader-obligation count: 25 → 27. Facets and
top-level sections are unchanged, still frozen at eleven and sixteen.

---

## 4. The schema

### 4.1 Facets — the only fixed kinds, frozen at eleven

> **FROZEN.** A twelfth facet may be added only by deleting an existing one. If a world needs a new facet
> that deletes nothing, this approach has failed and we say so rather than widening.

| Facet | Means | Illustrated by, in unrelated worlds |
|---|---|---|
| `extent` | has an interior; things are `within` it | a flooded atrium · a district · a shared dream · the inside of a living house |
| `matter` | is physical | a rope · a rib · a brass ring · a dead body |
| `agency` | decides and acts | a diver · a bureaucrat · a house that refuses tenants · a guild |
| `holding` | **stores** substances | a cistern · a granary · a container |
| `demand` | requires something to keep going | a house that eats grain · a dream that needs sleepers · a person who needs breathable air |
| `passage` | joins two extents | a stair · a hatch · falling asleep · a convergence of two migration routes |
| `motion` | moves on its own | a walking creature · a drifting barge-town · a glacier |
| `collective` | constituted of members | a guild · a ration board · people who owe a debt |
| `magnitude` | stands for many; individuals promotable | two hundred houses · a crowd under canvas · the unregistered |
| `record` | its content is a **claim**, which may be false | a register · a signed report · a transcription |
| `office` | authority that outlives the holder | a measurer · a rotating speakership · a criminal position held by whoever holds it |

*Combinations are the point.* A living house is `extent + matter + agency + demand`. A walking creature
carrying cities is `extent + matter + motion + demand`. A guild is `agency + collective`.
A door is `matter + passage`.

### 4.2 Sections

| Section | Means |
|---|---|
| `world` | name, premise, mood |
| `excluded[]` | **what does not exist or cannot happen here.** Binding on every authoring seat, forever. The strongest defence against genre drift. |
| `vocabulary` | the world's own words: `media`, `movements`, `channels`, `conditions`, `substances` |
| `law[]` | what this world permits — physical or social. Carries `enforced_by`, `binds`, optional `forbids` |
| `entities[]` | every noun. Facets above. The only container of things |
| `standing[]` | directed relation between any two entities |
| `opposition[]` | stated incompatibility — what cannot both be satisfied |
| `processes[]` | ongoing change: direction, rate class, terminus |
| `cycles[]` | ordered phases, period class, what each changes |
| `accumulators[]` | scope + **ordered threshold ladder** |
| `indicators[]` | a hidden state, its signs, what reads them, how reliably |
| `traces[]` | residue a change leaves, and how it ages |
| `epochs[]` | a structurally different past |
| `history[]` | events; per-holder knowledge; an event may be **`disputed`** |
| `arrivals[]` | plural premises; there is no opening state |

### 4.3 The three constructs worth understanding deeply

**`medium` — how worlds differ physically without a genre taxonomy.** What you are immersed in *here*,
which modifies every act attempted in it, for everyone, with no actor involved. A medium is an **invented
word with stated resistances**, never a type from a list.

*Illustrative only, four unrelated worlds:* still air that smells of wet chalk, resisting sight and open
flame · abrasive air that resists speech totally and unshielded skin severely · green water below a
visible boundary, in which objects speak · dream-matter that resists concealment and injury totally.

**`trace` — the load-bearing multiplayer primitive.** Every change leaves physical residue that is itself
perceivable and that ages. Without it, the past is only testimony and a player who was not there can only
be told. Most multiplayer is archaeology: one player acts, another reads the residue much later.

*Illustrative only:* a bright unweathered face where bone was cut, ageing slowly · wet prints on dry
steps, ageing quickly · a fallen mark on a catch-wall that never ages.

**`indicator` — a hidden state read through unreliable signs.** The state is never exposed; the signs
are. This is what lets a world be *about* measuring something badly.

*Illustrative only:* doors that stick and dust falling in still rooms, poorly indicating structural
lean · six diagnostic readings of an animal's health whose mapping to outcome rests on one documented
case.

### 4.4 The three contracts

**Author obligations** — what must be present, each justified by *what a builder needs to make a world
live*, never by what worlds contain. Two places and a way (movement needs a target). At least one agent
(nothing acts otherwise). Every agent wants something (a mind with no goal only reacts). Something
withheld somewhere (otherwise every conversation is exhaustive). At least one authored incompatibility
(otherwise the only motion in a room is the player). An `excluded[]` list, possibly empty, but explicit.

**Refusals** — what makes a document invalid, rejected whole with the reason named. Dangling references.
A number in an engine-computed field. A facet key without its facet. An inference chain that bottoms out
in nothing stated. Containment cycles. A demand with no supplier.

**Reader obligations** — *the half that is usually forgotten.* For every class, what the builder **must**
derive. `tension` ⇒ a beat budget. `pace_class` ⇒ a speed, and with distance a duration.
`reliability_class` ⇒ how often the sign misreports, and the hidden value is never exposed.
`disputed` ⇒ hold both accounts and never resolve without a later event. `excluded[]` ⇒ refuse to author
anything matching, in every seat, for the life of the world.

Two builders honouring these produce the same world. Without them, *"same document, different world"* is
as fatal as *"same brief, different document."*

### 4.5 Provenance, and the class–exemplar pair

Every element carries `"source": "stated"` or `{"inferred_from": [...]}`. This is what makes the
confirmation surface possible, what makes stated content outrank inferred on every contradiction, and
what makes genre bleed **mechanically detectable** — an inference chain terminating in nothing stated is
a refusal.

Some quantities are *both* fiction and computation. Forcing a choice made documents that said a number
twice and meant it once. So a class may carry an exemplar:

```jsonc
"decay": { "class": "brief", "exemplar": "three nights" }
```

The class stays the interface and the builder still owns the ladder — but must calibrate it so the
exemplar holds. One fact, one place.

---

## 5. How it was tested

Four world briefs, **authored independently of this design**, each at three levels of detail
(~400 / ~1,500 / ~3,000 words). Deliberately unlike each other: a city inside a living ribcage of
institutions; houses that are alive; nine creatures of geographic scale carrying cities; a city where
each district shares one dream.

| Round | Method | Result |
|---|---|---|
| Expressiveness | encode the detailed tier; report what breaks | v1 died on premises |
| Two-fold | two blind encoders, same brief | v2 died on ambiguity |
| Controlled re-test | **same** pair, same world, new schema | v3 converged the worst divergence; died on four fields |
| Generative | **400-word tier only**, canon held back as answer key | v4 passed |

**The method that found the most:** two independent encoders on one brief. Ambiguity is invisible to a
single reader by definition, and their *disagreements* were worth more than their agreements.

**The finding that mattered most** came from none of it — the realisation that genesis *invents* rather
than transcribes, which invalidated the tier test's criterion and produced v4.

---

> **The work is planned.** What the engine actually has against these obligations is
> `01_engine_capability_audit.md` (3 working, 12 partial, 9 absent, cited to file:line). The eight product
> increments that close the gap are
> `docs/superpowers/plans/2026-08-26-world-model-eight-increments.md`, which carries nine closed decisions
> and the parallel-wave plan. The items below are what those increments deliberately do **not** cover.

## 6. Open

1. **The reader half is tested once, on one document.** One round ran it (§3, v4→v5, and
   `R_score_grelda.md`): 22 questions, three blind builders, one fixture, one language, and two of the
   three builders shared a model. It found the `conceals` defect, demonstrated the ladder gap with
   evidence, and narrowed F1's calibration claim. **Still unexercised entirely:**
   `motion.trajectory` and `record.asserts[].accurate: false` — no question in that fixture reaches
   them. The named next step is to regenerate the sheet by the same procedure against
   `G_marea_by_gamedesign`, which exercises `motion` and is written in Spanish, so its questions must be
   too.
2. **Composition is unspecified, and it is the largest hole the reader round found.** Every reader
   obligation is a function of one authored key. Where two keys bear on one derived value the contract
   states no precedence — `admits` vs `obstructs` on one act, and the six keys governing whether a
   receiver receives at all. Candidate v6; see `R_score_grelda.md` §8.
3. **The class ladders v2 declared closed grammar are still unwritten** — but there is now evidence of
   what they must fix, and of what they must **not** try to: the anchors are vocabulary and belong
   per-world, while rung membership, rung order, polarity and the dimension a class measures are grammar
   and belong in code. `R_score_grelda.md` §4.1 and §4.2 are split on exactly that line. Do not write
   the ladders by averaging the builders' anchors.
4. **No fixture has ever been validated.** R1–R13 have no executable checker, and a reader round cannot
   catch an R-class violation — three builders used `within` in the way the facet gate forbids and none
   noticed. The validator is scoped in `R_score_grelda.md` §7 and blocks further reader rounds.
   **`within`'s facet gate is now settled — `SCHEMA-v6.md`.** Counted rather than estimated: the gate is
   violated **28 times across the three v4 documents** (9 / 10 / 9), every instance ordinary containment
   and not one an error. `G_grelda_by_simarch.md` §3's figure of "fourteen" was wrong and had propagated;
   the real count in that document is 9.
5. **Growth during play is unspecified.** The document must say what may be invented later and what may
   never be added after genesis.
6. **Provenance is unvalidated at scale** — hand-authored documents never tripped it.
7. **Multiplayer consequences are designed but untested:** there is no opening state; knowledge is never
   globally queryable; an NPC's memory must not merge across players; absence is a state, not a
   disappearance; accumulation is world-scoped so one player moves the stakes for everyone.
