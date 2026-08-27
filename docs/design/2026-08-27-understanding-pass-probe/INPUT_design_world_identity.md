<!-- EVIDENCE INPUT — recovered verbatim from git 70840b9:docs/30_architecture/world_model/03_world_identity_and_the_understanding_pass.md.
Deleted by 88486c1 (workspace:ADR-W006 consolidation); no successor file exists.
Not law. Input to the 2026-08-27 understanding-pass probe; cited by its REPORT.md.
-->
# World identity, and the understanding pass

**Date:** 2026-08-26 · **Status:** design, settled by grilling session. Not yet built.
**Scope: WORLD GENESIS ONLY.** Nothing here describes gameplay, the engine, or world creation during
play. Where a decision has consequences there, it is marked and deferred — see `SPEC-036`.

**What this closes.** Genesis is five steps: take an input · understand the intention · transcribe what
was given · **fill, always governed by that understanding** · emit the completed document. Step 2 was the
weakest and least specified. This document specifies it, and specifies how step 4 is governed by it.

---

## 1. The problem, stated as a failure mechanism

A filler produces generic content for one specific reason: **when it does not know what makes this world
different from the nearest familiar one, it fills from the familiar one.** Living houses become a
haunted-house story. A rising sea becomes a post-apocalypse. Not laziness — nothing else to reach for.

Second failure, downstream of the first: the filler cannot tell whether what it invented is **entailed**
by the world or merely **compatible** with it. Compatible content is infinite and worthless — a tavern is
compatible with almost any world and entailed by almost none.

So the understanding pass exists to answer one question well enough that filling can be governed by it:
**does this thing have to exist here, and what does it cost?**

## 2. Identity is invention rules, not description

**Rejected: a description.** Boxes filled in the same way for every world — mood, tone, setting,
technology, magic level. Three reasons it fails:

1. It is a claim about what worlds are usually like, which **GA-2** and **GA-3** forbid outright, and
   `prd_world_creation.md:177` bans by name: *"no genre taxonomy anywhere in code, prompt or schema."*
2. It reads the same for two different worlds often enough to be worthless.
3. It cannot be applied. Given *"whimsical, industrial"*, a filler inventing a post office learns nothing
   about whether its post office belongs.

**Adopted: rules the world makes about itself, each carrying its consequence.** A rule can be applied to a
candidate and answer *does this belong* — the only property that makes filling governable rather than
hopeful.

**The `therefore` is not decoration.** *"The world is indifferent"* is an assertion a filler will
acknowledge and then ignore. *"The world is indifferent, therefore nobody invented may be motivated by
malice toward the player"* is actionable. Every rule carries one.

**What this adds that does not exist today.** The document carries `premise` and `mood` — two prose
strings — and `excluded[]`. So a world can be stopped from being *wrong* and has nothing that makes it
*itself*. `excluded[]` is the **never** half. This supplies the **always** half.

## 3. What the understanding pass emits

Every item below is **inferred from the brief**, never asked as a form. Slots are minted because a world
needed them, not because a template had them.

### 3.1 Condition and bargain

**The condition** — what is simply true here. *The houses are alive. The water never stops. Everyone in a
district shares one dream.*

**The bargain** — what living with that condition costs you. One sentence.

> *You reach a place only if something that doesn't need you agrees to carry you.*

These are two things, not one, and conflating them loses the bargain. A condition can be stated by the
author; the bargain usually has to be derived.

**One bargain, not several.** Five bargains of equal weight produce mush — the filler splits the
difference or picks one at random per entity, and the world loses its centre.

**Faces, instead of multiple bargains.** The same bargain as it meets different lives. In a world where
shelter requires a house's consent: the dealer's face is *I sell patience*; the excluded's face is *I am
not refused, I am simply never accepted*; the rationing office's face is *I ration the thing that isn't
the scarce thing*. One bargain, three lives, no dilution.

**Test for one world or two fused:** try to derive the second pressure from the first. If you can, it is a
face. If you genuinely cannot, the bargain is stated at the wrong altitude or two worlds were stapled
together — a quality problem to catch, not accommodate.

### 3.2 The departure

The nearest familiar thing this world resembles, and the specific way it is not that.

> *Not whimsical talking vehicles — the trains are indifferent, not friendly.*

Cheapest item on this list and among the highest-value, because it blocks the exact failure of §1: it
names the neighbour the filler would otherwise default to.

### 3.3 What is scarce, and what is wrongly abundant

Scarcity gives conflict for free. **Inverted abundance is what makes a world feel invented rather than
borrowed** — a city full of empty rooms with people sleeping in tents says more than any adjective.

### 3.4 How consequence arrives

What happens when someone oversteps, and what does it. Every person invented afterwards then has
something specific to fear, and fear is what makes them act rather than wait.

### 3.5 Exclusions, each with its reason

**Reasons generalise; lists do not.** *"No money buys a pact"* stops one thing. *"No money buys a pact,
because consent is the scarce thing here"* stops everything of that shape, including cases nobody listed.

Each exclusion is additionally marked as **exist-kind** or **happen-kind**, and happen-kind rules carry an
**enforcement marker** — see §7.

### 3.6 Register

The size of a normal problem here. Missing a funeral, or losing a war. Getting it wrong makes everything
tonally wrong even when it is factually right.

### 3.7 Content demand — a requirement, not a permission

Not a rating. A world about bodily exposure, or debt-bondage, or addiction, **does not exist in a
sanitised form** — remove the content and the pressure goes with it, leaving a world wearing its own
nouns. So it is a rule with a consequence:

> *This world's central pressure is involuntary exposure of what you want; therefore any invention that
> lets a person keep their wanting private is decoration, and any scene resolving without someone being
> seen has not happened.*

*"NSFW: allowed"* governs nothing. The inverse is equally a demand: a clean world is not a restricted
world, it is a world whose pressure comes from elsewhere — and stating that stops a filler reaching for
transgression as a cheap source of stakes.

### 3.8 Voice — imitable, never described

**Real prose, not adjectives.** *"Grim and terse"* produces generic grimness. Three sentences of actual
world narration can be imitated for the life of the world.

The slot holds: narration in the world's register — rhythm, how much is left unsaid, how close it sits to
the body — how raw it goes, shown rather than rated, and what feeling the world reaches for.

**Voice composes with worked examples.** A worked example teaches the world's *logic*; voice teaches its
*sound*. Write the worked examples **in** the world's voice and one artifact does both. Do this
deliberately rather than letting the two slots drift apart.

### 3.9 Worked examples

Two or three short resolutions of *someone tries X, here's what actually happens.*

The highest-value item for a filler and the thing a schema is worst at holding. One worked resolution
teaches more than five adjectives, because it can be pattern-matched against forever.

### 3.10 The twenty universal functions

The ordinary life the pressure does not demand. **Phrased as human functions, never professions** —
*"carpenter, baker, blacksmith, innkeeper"* is a fantasy template that fails GA-2's own test (*does the
term survive a sci-fi thriller, a workplace drama and a horror story?*).

Who feeds people · who repairs and makes things · who moves goods · what happens to the sick and the dead ·
who raises and teaches the young · where people with nothing sleep · what exchange is · what people do for
pleasure · how two ordinary people settle a dispute · what happens to the old · what the world does with
its waste · how ordinary people learn ordinary news · what a normal day's work is · what people fear that
has nothing to do with the bargain · what people find funny · what marks status · what children are warned
about · what a stranger is · what privacy means · what counts as clean and dirty.

**These do three jobs at once**, which is why they are worth twenty answers:

1. **They test the identity.** A thin identity produces a generic answer on a topic it never mentioned,
   and you see it immediately.
2. **They build the ordinary life** the pressure does not demand, which is what stops a world being a
   theme park inhabited only by the bargain's cast.
3. **They become the reference for everything minted later.** When play needs a carpenter in beat 200,
   *"who repairs and makes things"* has already established how making and mending works here — so the
   unauthored carpenter arrives consistent instead of invented from nothing. This is what makes lazy
   filling coherent rather than a slow drift to genre.

**"None, and here's why" is a valid and often superior answer.** A world of immortals has no answer for
*what happens to the old*. The absence is identity-revealing and belongs in the exclusions with its
reason, which then stops a filler inventing a marketplace in a world without exchange.

**One or two sentences each, never paragraphs.** Twenty topics answered at length is the *"generated
filler"* the contract warns about — four hundred authored words drowning under thirty generated pages.
These are anchors, not chapters, and they must be short enough that all twenty stay in context on every
later call, because that is how they do their third job.

### 3.11 The pass criterion for §3.10 — not a separate test

**There is one test, and §3.10 is it.** This section is the criterion you apply to those twenty answers,
not a second mechanism. An earlier draft presented it as its own test with a worked carpenter, which read
as two things and misled the first agent to inherit the document. Corrected here.

The criterion, applied to each of the twenty answers:

> **Could this answer exist in any other world?**

If it could, the identity is too thin, and every element minted for the life of that world will be
generic. The twenty functions are chosen precisely because **the identity never mentions any of them** —
which is what makes them a real test rather than a restatement of what the identity already said.

Worked, on *"who repairs and makes things"* in the trains world: he cannot guarantee materials arrive, so
he keeps years of stock or builds only from what is already in the yard. His timber comes from wherever a
train felt like going, so he works in opportunistic materials and has a reputation for the wrong wood. He
has never worked the far side of the city — not because he won't, because he cannot reliably get back.

That answer could not be lifted into another city. An answer that comes back as *a carpenter* is the
failure signal, and it costs ten seconds to see.

## 4 · Rules have kinds, and the kind sets the work order

Not every rule generates content. Assuming so was an error worth recording, because it produced a check
that validated nothing.

| Kind | Example | What it demands |
|---|---|---|
| **Constraining** | *the water turns lighter at night and everything on it sinks* | **No entity.** It changes what is possible; everything else must be invented inside it |
| **Generative** | *you reach a place only if something that doesn't need you agrees* | Entities: people who negotiate, places reachable-but-not, a trade in persuasion |
| **Prohibiting** | *no timetable is enforceable* | Removes possibilities |
| **Voicing** | the register itself | Language, not content |

**Constraining rules are established before generative rules run**, because they change the space the
generative rules invent into. A harbour invented before the water rule is known is a harbour that makes no
sense after dark.

**Ordering by importance was dropped.** It was carried from an assumption — that the artifact would be
truncated and the filler would need to triage — which the arithmetic disproves: the whole artifact is
about 1,600 words and fits in context on every call. Ordering by *kind* is real; ranking by weight solved
nothing. And if two rules genuinely conflict, ranking would be the wrong fix — that is a defect in the
identity, to be surfaced rather than papered over by precedence.

## 5 · Origin decides what play can change

**A rule that names an origin is contingent. A rule with no origin is axiomatic.**

A drought caused by a curse: kill the cause, the rule goes. *"After the war, a law was passed"*: repeal is
possible, and the world says so. The same statement offered as bare fact — *"the houses are alive"*, *"the
water never stops"* — offers no cause, so there is nothing to undo.

Three reasons this is the right mechanism:

- **It is inferable.** Genesis can see whether the brief offers a cause. No new question needed.
- **It uses machinery that exists.** An origin is an event or an epoch — both are already sections of the
  document. A contingent rule points at what made it so; an axiomatic rule points at nothing.
- **It gives the world its own theory of what can be changed**, which is what a player needs in order to
  have hope. And the ratio is itself an identity decision: a world where nothing has an origin cannot be
  changed by anyone; a world where everything does has no stable ground. **The mix is the game.**

**Risk to watch:** a filler will hand out origins freely, because a contingent world is easier to write
drama for. If everything turns out to have a cause, nothing is load-bearing. *How much of this world is
contingent* should be an explicit identity output, not an accident of generation.

**Identity is emitted versioned**, and is immutable for the duration of genesis.

## 6 · Asking the author, when inference fails

**Never ask a question in the system's vocabulary. Ask a question answerable as fiction, then mine the
answer for rules the person never mentioned.**

Not *"how does consequence arrive?"* but:

> *Someone breaks the biggest unwritten rule in your world. Who notices first, and what do they do about
> it?*

Three sentences from anyone, no vocabulary required, and the answer yields four things: who enforces, how
fast, how harshly, and whether it is social or physical.

**Multiple choice with an open option.** The options *teach* what a useful answer looks like without
demanding that anyone learn the model; the open option means it never railroads someone with a better
answer.

**Three attempts, then stop.** If the answers are still flat after three, the author wants a flat world
and gets to play one.

**Asking well converts inference into fact.** An answer is `stated`, not `inferred` — which outranks
inference on every later contradiction, and is worth more than the content of the answer itself.

### 6.1 A flat world is a legitimate outcome, and must be recorded as one

> *This world has no central pressure, by the author's choice; therefore invent no hidden stakes, no
> secret cruelty, and no buried threat.*

Not a consolation prize. This is what stops a narrator putting a dark secret in the cosy village by the
third scene. **A flat world that stays flat is a better product than a flat world that grows a serial
killer.**

## 7 · How identity governs filling

**Rules are the work plan.** The filler never invents freely; it works from one rule at a time.

> *Here is the world's identity. Your current work item is this rule. What must exist because of it?*

**The code schedules; the model interprets.** Code reads the list, loops, and tags each returned element
with the rule that produced it. Minting rules per world costs nothing, because the loop never needs to
know what is in them.

**Why this beats putting the rules in a prompt and hoping.** With rules as context only, the early output
is decent and by the fortieth entity the model is pattern-matching on its own earlier output — *city* plus
*authority* pulls hard toward police, and a Transit Police gets invented in a world where no authority can
strand or rescue anyone. Working *from* rule five, that entity is never requested, because nothing asked
for it. Content becomes **entailed** rather than merely permitted.

**Both, in fact:** rules drive the order *and* stay in context throughout, for the judgements no plan can
pre-specify.

### 7.1 Volume is not bounded by the number of rules

A rule does not produce an entity. It produces a **space of positions**, and every position is a life.
From one rule — *you reach a place only if something that doesn't need you agrees* — with nothing else:
someone good at asking, and within that the old master, the apprentice with the words but not the timing,
the one who cheats, the one losing the knack and hiding it; someone who never asks; someone a train
inexplicably favours and is hated for; someone who sells access to askers; someone whose living depends on
others arriving late; someone who tried to force a train once and is marked; an academic despised by
practitioners; someone who must be somewhere tomorrow and cannot get there; someone who arranged their
life never to travel; a child certain the trains are her friends.

**Depth = a specific angle on the pressure × what it privately costs them.** Not *a merchant* but *a man
selling access to a skill he no longer has*. The private cost is the `hiding` the contract already
carries, and it is the difference between a position and a person.

**So the claim is not** *"nothing is invented without a rule asking for it"* — that was too tight and made
worlds small. It is: **nothing is invented that cannot answer to the identity.** Volume is unbounded; what
is bounded is arbitrariness.

### 7.2 Three tiers of content

1. **Demanded by identity** — the pressure's cast. Authored at genesis.
2. **Demanded by sufficiency, not by the pressure** — the arrival neighbourhood must be inhabited whether
   or not the bargain cares, because every reachable place needs something to see, take or speak to.
   Authored, shaped by identity, not demanded by it.
3. **Available on request** — carpenters, until someone asks. Minted during play against the same
   identity. Already stage nine of the pipeline.

Tier 3 is why not authoring carpenters is not a gap. If a player asks and gets one who belongs, there was
never a hole. If they ask and get nothing, that is an **authoring boundary** — and by
`SCHEMA-v4.md:62-63` that is a product failure, because the player cannot tell it from a fictional one.

### 7.3 Reviews, not gates

**The coverage check was wrong and is dropped.** *"Every rule must produce at least one entity"* fails on
the first constraining rule it meets — *the water turns lighter at night* demands no entity at all — and
would either pass vacuously or refuse a good world.

What survives is the **tagging**, for one reason: when the author corrects a rule, the elements that came
from it are known, so correction is a scoped retraction instead of a re-run. That was the real prize, and
it closes the biggest gap found across all three existing encodings — they emit a flat load-bearing list
with no dependency edges, so correcting one inference leaves orphans and the world argues with itself.

The rule-to-content mapping is something a person or a model **looks at**, not something a build refuses
over. A rule with nothing under it is a question worth asking, and sometimes the honest answer is *it is a
constraint, and constraints do not produce rows.*

**Exclusions get one review pass**, not a gate: hand a model the exclusions and the finished element list
and ask what breaches. One call over the set, not a check per invention. **The reviewer must not be the
generator and must not see the generation context** — a model reviewing its own output is lenient.

## 8 · Two correction points, and the first is nearly free

**Identity confirmation, before any content exists.** Shown to the author: **voice**, the **condition**,
the **bargain**, the **departure**, the **content demand**, the **register**.

Nothing there spoils anything — **the identity's shape is showable; its content is not.** Voice reveals no
entity, no secret, no plot. The worked examples and the twenty answers are *not* shown, because those
contain invented specifics and the twenty in particular are where a world's surprises live.

**Let the author rewrite the voice sample.** Someone who cannot articulate a rule can almost always show
you the sound they want — and a rewritten sample is **authored** voice rather than inferred, in the one
slot where the author's own hand matters most.

**Document confirmation** remains where it is, after filling.

### 8.1 Fast lane gets no pre-build step at all — founder ruling, 2026-08-27

> *"If the user wants the fast lane then they get the fast lane. We should not turn the goal of a fast
> lane into a 'fast lane' that asks 35 questions to feel we are doing a good job."*

So in the Fast lane: **no interview, and no identity confirmation screen.** The identity is inferred, the
world is built, and **the finished world is the confirmation.** If it is wrong, the author regenerates or
switches to Custom — which is what Custom is for.

§8's confirmation point therefore belongs to **Custom only**, alongside §6's questions.

**The cost, stated so it is chosen rather than discovered:** a bargain inferred at the wrong altitude
produces a world that is coherently and confidently wrong, and in the Fast lane the author's first sight
of that is the finished world. This is a deliberate trade of correctness-before-build for speed, and the
mitigation is that the world is cheap to regenerate — not that a screen catches it.

**The general principle, which outranks this instance:** a step that exists so the builder feels
thorough is not a safeguard. If a confirmation cannot be justified by what it catches, it is friction
wearing a safety costume, and it does not ship.

The first makes the second far more likely to pass, because the expensive regeneration is usually caused
by a misread that was visible at the cheap moment. The pipeline currently calls confirmation *"the only
correction window that exists"*; this adds an earlier one that costs a screen.

## 9 · Deferred, and why

**Enforcement of the world's own rules — `SPEC-036`.** Rules about what may **exist** have no engine check
possible and are enforced by a model at every content-creation moment. Rules about what may **happen** map
to an act and can be checked inside the checks the play loop already runs — but only for acts the engine
recognises. A rule naming an unknown act is **narrator guidance, not enforcement**, and genesis must mark
it as such rather than presenting both as rules the world will hold.

No code is generated per world, ever. Checks are instances of shapes already coded, with the world's
values filled in — the discipline `core/api/tier1.go` already states.

**This is the item to remember when world creation during gameplay is designed**, because the exist-kind
cost scales with hours played rather than with world size.

## 10 · What is deliberately not decided here

- Whether the twenty functions are exactly twenty, and whether the list is fixed forever or itself
  evolves as worlds are made.
- How a *contradiction between elements produced by different rules* is caught. Code cannot see that an
  apology-publishing body and a compelling police force cannot coexist. A review over the finished set,
  and no pretence that a count could catch it.
- The shape of the emitted artifact as a machine format. That waits on the world-model schema becoming a
  real artifact, which is the blocker for all of step 5.

---

# Appendix — remaining questions for phase 1 (genesis)

**Thirteen open. Five could change a design decision already made; five are real but local; three are
small or already noted in §10.** Genesis-wide, not only the understanding pass. Nothing below is a
gameplay or engine question — those live in `SPEC-036` and the roadmap.

## Could change the design

**Q1 · Does rule-by-rule filling fit the call budget?** `prd_world_creation.md:26-27` sets the Fast lane
at **p50 ≤ 90 s, p95 ≤ 180 s, p50 ≤ $0.25 per world**, and `:32` says *"one LLM-authored world document
per build."* §7's mechanism is one call per rule — roughly nine identity rules plus twenty universal
functions is up to **29 sequential calls**. That is ~3 s and ~$0.008 each. Either the calls batch, or they
parallelise, or the budget moves, or §7's mechanism does. **This is the largest risk to this document.**

**Q2 · Does the understanding pass work identically in the Fast lane, which has no interview?**
`:32` and `:62`: two lanes chosen *after* the brief; Custom *"only adds answers"*. So §6's in-fiction
question protocol exists only in Custom, and Fast must infer everything or accept a thinner identity.
Does Fast run the same passes with the questions skipped, or a different pass entirely? And is a
Fast-lane identity allowed to be weaker, or must it meet the same §3.11 test?

**Q3 · What happens when the input is a genre reference?** *"Like Dune but underwater."* *"Skyrim with
guns."* A common real input, and a direct collision with **GA-2/GA-3** and `:177` (*no genre taxonomy
anywhere*). The system may not know what Dune is in the sense of holding a template — but the author has
just told it the most informative thing they know. Refusing loses the signal; absorbing it imports exactly
what the rules forbid. Unresolved, and it will arrive on day one.

**Q4 · When does filling stop?** Sufficiency (S1–S6) is a *bar*, not a stopping rule — *"the world is
open"* describes a property, not a termination condition. Rule-by-rule gives a natural end for the
identity rules and the twenty functions, but nothing bounds how deep each answer goes, or how many
positions a generative rule should be worked into. Without an answer, filling is bounded only by the call
budget, which means the budget is silently making a product decision.

**Q5 · Does identity travel inside the emitted document, or beside it?** §3.9 argues worked examples
belong in the document because the narrator needs them. The same argument applies to the bargain, the
departure and the voice — those govern every later invention, including lazy fills during play. If
identity is a sibling artifact instead, it has to be loaded everywhere the document is, and can drift from
it. If it is inside, the document gains a section that is not world content but instructions about world
content — which changes what the schema is.

## Real but local

**Q6 · Is there a minimum viable input, and what happens below it?** *"A fantasy world."* Four words.
Refuse, ask, or generate? The Fast lane has no interview, so refusal and generation are the only options
there.

**Q7 · Where exactly is the line between transcription and filling?** Provenance depends on it: `stated`
outranks `inferred` on every contradiction, and R13 refuses a chain bottoming out in nothing stated. If
the author names a character with no detail, the name is `stated` — is the character's role? Their
presence in a place? The line needs drawing precisely, because everything downstream cites it.

**Q8 · What is the filling order past "constraints before generative"?** Places before people, or people
before places? Both are defensible and they produce different worlds — placing people first yields a world
shaped around lives; placing places first yields lives shaped around geography.

**Q9 · Three arrivals, or one?** The world-brief reading argues for three at different distances from the
detonante, with at least one being someone the detonante happens *to* rather than *through* — pillar 1 in
mechanical form. `SCHEMA-v7.md` O14 obligates three only when identity is left open. Needs a product
ruling, not more analysis.

**Q10 · What does genesis do with a brief that contradicts itself?** Stated content outranks inferred, but
nothing orders *stated against stated*. Refuse, pick one and say so, or carry the contradiction as an
authored tension?

## Small, or already recorded

**Q11 · Are the twenty functions exactly twenty, fixed forever, or do they evolve?** (§10)

**Q12 · How is a contradiction between elements produced by *different* rules caught?** Code cannot see
that an apology-publishing body and a compelling police force cannot coexist. (§10)

**Q13 · Can the author supply structured input** — a list of characters they want, a map — or only prose?
`:32` says *"one free-text brief"*, which reads as prose-only, but does not say whether that is a
constraint or a description of the MVP.

---

**Not questions, but blockers already known:** the world-model schema has no machine representation, which
gates step 5 entirely; and `prd_world_creation.md` still describes `places / cast / objects / ways` —
structurally the world-model version that died — which must be reconciled before either document can be
built against.
