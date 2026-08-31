# The filling stage — putting skeleton and meat on the soul

**Status:** design, founder-ruled 2026-08-28. Governing document for stage 2 of world genesis.
**Supersedes:** the flat batch schedule shipped 2026-08-28 (places → history → lives → objects → revise
→ sufficiency), and with it design §7's "rules are the work plan".
**Depends on:** `SPEC-040` (unperceived canon), `SPEC-041` (perception lineage), `SPEC-042` (persona
linkage), `SPEC-043` (concept entities). Written on the assumption those will be built.

---

## 0. Where this sits, and why that matters

World creation is **stages**, and confusing them is the single most expensive mistake available here.

| stage | what it does | state |
|---|---|---|
| 1 · Intention & understanding | finds the **soul** of the world — condition, bargain, departure, exclusions, the twenty ordinary functions | **done, and good** |
| 2 · **Filling** | puts **skeleton and meat** on that soul | this document |
| 3 · Transcription | turns the authored document into engine rows — canon events, perceptions, entities | later |
| 4 · Play | the world runs | later |

**At fill time nothing is canon and nothing is a perception.** Fill authors *fiction*. It takes no
database handle: `authorWorld(...)` returns a `*genesisDoc` and writes nothing. The draft becomes canon
in exactly one place — `commitWorldContent` — which is stage 3.

That does **not** mean fill may ignore the canon model. It means the opposite: fill authors the fiction
*that transcription will turn into canon and perceptions*, so it must author it in a shape transcription
can accept. Founder, 2026-08-28:

> it is ok to discuss how canon and perception will work cause the next steps will have to transcribe it
> to the rules of the engine. so while you fill it you can define each canon with at least 1 perception
> (is it common knowledge? who knows it? what is the perception?) so the lore makes sense. **the world
> existed before we started to play in it.**

---

## 1. Product view

### 1.1 What filling is for

The world existed before the player arrived. Filling is what makes that true rather than claimed.

- **Lore is knowledge.** It is not decoration and it is not a summary. A piece of lore has **holders** —
  people who know it, people who half-know it, people who are wrong about it.
- **Depth is specificity, not volume.** Twelve rooms that could be any world are worse than four that
  could only be this one.
- **The author's words outrank inference**, always. What the brief and the answers state is stated; the
  rest is ours to invent under the identity.

### 1.2 The layered loop

Each layer is authored **from** the layers above it. Founder's ordering:

| # | layer | authors | lore |
|---|---|---|---|
| 1 | **Foundation** | nothing — map what the author already gave us | — |
| 2 | **Places** | rooms and the ways between them | public |
| 3 | **Factions & groups** | organised bodies and collectives, each with a goal | public |
| 5 | **People** | individuals: public surface, personality, goal, example speech | public **and** private, including private lore *about* the places, factions and concepts above |
| 4 | **Concepts** | schools of thought, bodies of knowledge — `SPEC-043` | knowledge held about them |
| 6 | **Artifacts & elements** | things a body can touch | lore, plus who knows where they are and who is wrong |
| 7 | **Arrival** | the world header, the region, and the way in | — |

Concepts sit **above people and below factions**: a school of thought exists before the apprentice who is
wrong about it, and the College that teaches it exists before the school is a thing anyone argues over.
They are a full layer, on the same footing as every other — nothing about them is conditional on the
engine, because the engine is not in this stage.

**The loop always uses the previous data.** A person's private lore is about *these* rooms and *these*
guilds, named. An artifact's lore says which of *these* people knows where it is.

### 1.2.1 Descent, then ascent — create going down, connect coming back up

Founder, 2026-08-28:

> we go for the first layer down, making content on each layer. and then we bounce up
> fixing/correcting and making content on each layer ... the second round is more to fill gaps and
> connect with clicks rather than creation (creating is not forbidden) and the result can be hundreds of
> new stuff or not. there is no minimum or maximum

So the loop runs **twice, in opposite directions**, and the two passes have different jobs.

**Descending** — foundation → places → factions & groups → concepts → people → artifacts — each layer
authors from what is already above it. This is where the world is *created*, and it can only ever look
upward, so nothing is named before it exists.

**Ascending** — artifacts → people → concepts → factions & groups → places — each layer is revisited
knowing **everything below it**, which on the way down it could not see. A room authored before anyone
lived in it can now be told who sleeps there and what happened in it. A guild authored before its members
existed can now be connected to the people who joined, left, or were struck from its roll.

The ascent's job is **connection and gap-filling**, not bulk. It may create — creation is never forbidden
— but a new person invented on the way up should exist because a *connection* demanded them, not because
the pass felt thin. **There is no minimum and no maximum.** An ascent that adds one sentence to one room
and a hundred links is a good ascent; so is one that adds forty people because the world genuinely had
holes.

**This replaces the flat `revise` batch**, which sat in the middle of a one-way schedule and had no way to
know what came after it.

**A layer may prove redundant, and that is a finding, not a failure.** The founder's expectation is that
the ascent may show some layer had nothing left to do. Measure before deleting: a layer that adds nothing
on a dense brief may be carrying a sparse one.

### 1.3 Why layered rather than flat

The shipped design had one `history` batch running **before** people existed, so canon named people who
did not exist yet. That forward reference cost eleven failed production builds and three separate
mechanisms to survive it — a debt ledger, an in-context nudge, and two closing passes.

In the layered loop lore is authored **with the layer it belongs to**, after its foundations exist. The
failure class does not need managing because it cannot arise.

### 1.4 Public and private lore

- **Public** — common knowledge. Everyone here knows the tower floods; nobody is impressed by it.
- **Private** — one holder, or a few. The thing that makes a scene possible.

Both are authored per layer. A layer with only public lore is a brochure; a layer with only private lore
is a conspiracy with no world around it.

### 1.5 People are where the depth lives

Founder, 2026-08-28:

> people also need their whole upbringing behavior. each person is unique and that uniqueness is derived
> from their life circumstances. with one or two exceptions you could have had the worst shitty life ever
> but still be optimistic (really fucking hard but possible and interesting). so people has personality,
> believes, mantras, traumas, etc.

Two things follow, and the second is the hard one:

1. A person carries **personality**: traits, beliefs, mantras, traumas — plus a **goal** and what they
   would sacrifice for it, and **example phrases** in their own voice.
2. **Circumstance and disposition must be allowed to disagree.** The worst life and an optimistic
   disposition is the interesting case, so life events and temperament are authored **separately**. A
   single flat trait list cannot express it, which is why the persona is not a summary of the biography.

Example phrases matter beyond flavour: they are the material stage 4 uses to *be* this person. A line
they would actually say is worth more to the narrator than three adjectives.

### 1.6 Goals

A goal is a driver, for **people and for factions**. One goal each, plus **what they would sacrifice for
it** — the sacrifice is what makes it dramatic instead of a label.

### 1.7 Question templates, not inventories

The understanding pass works because it **asks**: what is the condition, what is the bargain, what is the
departure, what does this forbid. The answers *are* the artifact.

The shipped fill prompt gave imperative inventories — *"author the rooms and the ways between them"* — and
got inventories back. Each layer gets its own questions instead, and **the perception questions are part
of the layer**:

- **Places** — what rooms does this condition force into existence? what joins them? what does a stranger
  see first? what does everyone here know about this room, and what do they have wrong?
- **Factions** — who organises here? what do they control? what do they **publish**, and what do they
  **bury**? who outside them believes the published version?
- **People** — who holds this room? what do they carry that they will not say? what do they believe about
  the guild that is not true? what do they know that nobody else does? what would they say, in their own
  words?
- **Artifacts** — what does a body touch here? who knows where it is? who thinks it is somewhere else?

The last question in each set is where lore stops being a summary and becomes knowledge with holders —
including holders who are **wrong**, which is what `epistemic_type: rumor` and `inference` exist for and
which the shipped prompt never asked for.

---

## 2. Tech view

### 2.1 Three axes, and why conflating them hollows out the world

| axis | what it is | mutable? | carries weight? |
|---|---|---|---|
| **Canon** | what happened | never — append-only (`D-1`, `I-1`, `I-2`) | n/a |
| **Perception** | who knows it, how sure, how garbled | replaced, never edited (`SPEC-041`) | **no** |
| **Personality** | who they are — traits, beliefs, mantras, traumas | evolves | **yes** |

**Perception carries no weight, and this is load-bearing.** `perception_record` holds `content`,
`epistemic_type`, `sensory_mode`, `confidence`, `distortion_level`. Every weight there is *epistemic* —
how sure, how garbled. There is no salience, intensity or valence. Founder:

> perception has no "weight" cause is more oriented to knowledge (what you know) not so much on how you
> feel about it ... so having personality traits like trauma or beliefs are much more meaningful at
> defining each NPCs behaviour and speech

So at engine level, *learning to bake bread* and *witnessing your family's murder* are the same kind of
row. **A trauma modelled as a perception is a fact someone knows, with no expression of what it did to
them.** Trauma, belief and mantra are **personality**. This was got wrong once during design and the
correction is the reason §1.5 exists.

`SPEC-042` carries the consequence: nothing currently links a trait to the knowledge that formed it, and
`personality_core`'s own header is *"WHO THEY ARE IN THE ROOM. No secret ever lives here"* — so a private
trauma has no home in the persona table at all.

### 2.2 One canon event, many perceptions, arriving whenever

`perception_subject (perception_id, entity_id, world_id)` — a perception is *about* an entity. One canon
event may have any number of perceptions, and they need not exist when the event does. Founder:

> something happen. 2 months later someone learns about it by gossip! BOOM that is a perception of a
> canonical event

Two consequences for fill:

- **`who` is not `knowledge`.** `who` is who was *there*; `knowledge` is who *knows*. A knower need never
  have been present — that is what `told`, `overheard`, `public`, `rumor` and `inference` are for. An event
  with two people in the room and five who have heard some version of it is a world with lore. An event
  known only to its participants is a world where nothing travels.
- **Fill authors the perceptions that exist now** and does not strain to give every event a full audience.
  Play adds more.

### 2.3 Canon is never edited to make a later choice fit

Canon is the record. If a later layer wants a name canon already used, **the later layer yields**. This
is why the arrival takes an alternative rather than history being rewritten: the arrival is authored last
and is attached to nothing, while the cast are embedded in canon, in hands and in rooms.

An earlier version of this pipeline stripped a participant out of history to resolve a name collision.
That was wrong twice over — it deleted authored content, and it treated the record as negotiable.

### 2.4 Fill authors a document. The engine is not in this stage.

**This section used to say "fill may author only what transcription can commit". That was wrong, and it
was wrong in the way this whole design exists to prevent.** It imported a later stage's constraints into
this one, and then used them to shrink what the world is allowed to contain.

At fill time there is no `entity_kind`, no CHECK constraint, no migration and no engine. There is a JSON
document. A concept in that document is a name, a description and the knowledge people hold about it —
exactly like a place or a person, and no more privileged. **How any of it becomes rows is transcription's
problem, in transcription's stage.**

So fill authors the full world the identity demands: places, factions, groups, **concepts**, people,
artifacts, canon, perceptions, personality, goals, example speech. Nothing here is gated on engine
support.

What *is* worth writing down, as a note for whoever builds stage 3 rather than a limit on stage 2:

| the document will contain | stage 3 will need |
|---|---|
| places, people, artifacts, canon, perceptions | already handled |
| factions, groups | `entity_kind IN ('faction','group')` already exists — likely trivial |
| concepts | no `entity_kind` for them yet (`SPEC-043`) — a decision for stage 3 |
| goals, example phrases, private personality | no home yet (`SPEC-042`) — a decision for stage 3 |
| unperceived canon | refused and unreachable today (`SPEC-040`) — a decision for stage 3 |

**Today's code happens to chain fill straight into commit in one request.** That is a property of the
current implementation, not of the design, and it is the reason a document richer than the engine will
currently fail late. Fixing that is stage 3 work. It is not a reason to author a shallower world.

### 2.5 Faction, group, and concept are three different things

A world will want all three, and collapsing any two of them recreates the failure above in a new costume.

- **Faction** — an organised body with interests and a command structure. The College of Abjurers.
- **Group** — a collective without one. The eleven families. The survivors of the outbreak.
- **Concept** — a body of knowledge people hold beliefs about and are wrong about. Abjuration itself.

The College is a faction; abjuration is a concept; they coexist. Proposed discipline for concepts, so they
do not swallow every noun (`SPEC-043`): **a concept is an entity only if someone can be wrong about it in
a way that changes what they do.**

### 2.6 What changes in the fill contract

`world_fill/1` gains:

- **`factions`** — canonical name, kind (`faction` | `group`), what it controls, what it publishes, what
  it buries, its goal and what it would sacrifice.
- **`cast` gains** — beliefs, mantras, traumas, a goal with its sacrifice, and example phrases in the
  person's own voice.

`scheduleWork()` becomes the layered loop of §1.2 instead of six flat batches. Each layer's prompt becomes
a question template.

**Ceilings are not depth budgets.** They bound one reply against the seat's token budget so it cannot
truncate. A world needing more places than one reply allows gets them across layers.

### 2.7 Latency, measured

`ms = 1349 + 15.84 × tokens_out` — **1.35 s fixed per call, then ~63 tokens/second** (two calibration
points, and it predicted a 234 s build to within 1%).

So **wall clock is bought with output tokens, not with call count.** Fewer, larger calls save only the
fixed cost; parallelising saves only overlap, and the layers genuinely depend on each other. Cost is not
the constraint: a deep build measured **$0.02** against a **$0.25** budget. Time is. And there is a hard
**900-second ceiling** on the streaming request that no tuning removes — a genuinely deep world will
eventually need the build detached from the request, the way the image platform already is
(`image:ADR-006`, pull not push).

---

### 2.8 Places are nested, and at whatever scale the brief describes

`extent_class` runs `intimate | small | medium | large | vast`, `kind` is free text, and `within` names
the place that contains this one. The engine has always nested locations —
`location_state.attrs->>'parent_location_id'` with a recursive walk (`core/db/schema.sql:1582-1599`) and a
documented precedent of a parent quarter — and the fill contract simply had no way to say it.

**This was a real defect and it was a vocabulary defect.** The fill prompt said "room" thirteen times and
carried one world's shape as a worked example. That framing narrowed every possible world to interior
spaces, and the day's builds duly returned small interiors for a brief describing something vast. No
amount of belt tuning would have fixed a prompt that asked the wrong question.

So the places layer reads the scale off the brief and authors the whole ladder, from the largest thing it
describes down to the smallest place a body can stand in. And **`GA-2` is now the discipline stated in the
prompt**: before naming a kind, a role or an institution, check the word would still make sense in a
sci-fi thriller, a workplace drama and a horror story. Any world's shape is an example, never a template.

### 2.9 The 900-second ceiling is on the response, not on the build

Authoring used to run on `r.Context()`, so when the edge cut the connection the request context was
cancelled and minutes of paid model time died with it — the `context canceled` in the logs of 2026-08-28,
twice, at exactly 899,997 ms. **The ceiling is on the streamed response. The build had no business
inheriting it.**

The fix is one line: author under `context.WithoutCancel(r.Context())`. The build keeps its cost sink,
loses the disconnect, and finishes and commits whether or not anyone is listening. Frames still go to the
live socket while there is one. A client picks the world back up through the resume path the handler
already has.

**There is no architectural fork here.** Committing after the descent so a player can start sooner may
still be desirable one day, but that is a product choice about when a world becomes visible — not a
workaround for a timeout.

What remains true: wall clock is bought with output tokens, not call count. A deep world takes as long as
it takes, and costs about **$0.02** against a **$0.25** budget.

## 2.9 BUILT 2026-08-31 — relevance, and the two-stage loop that pays for it

The layered loop in §1.2 shipped as a **namespace stage and a content stage**, and the thing that makes
it affordable is `relevance` (**ADR-P027**), not the layering.

**Measured 2026-08-30, and the reason this section exists:** 33 entities cost 76,288 output tokens —
**2,312 per entity** — because a location-keeper was authored as richly as a city's ruler. Parallelism
divides wall clock and never volume, so the only way to a bigger world is to stop writing everything at
the same depth.

**The stages, as implemented:**

|#|call|shape|what it does|
|---|---|---|---|
|1|`concepts`|one, sequential|what this world argues over. Nothing below may contradict it.|
|2|`scaffold-1`|one, sequential|the top of the world: largest locations, spanning factions, structural objects. Names, one line, a level.|
|3|`scaffold-2`|**parallel**, per top location AND per faction|what sits inside each: nested locations, the people in them, local factions. **Assigns relevance.**|
|4|`geography`|**parallel**, per top location|authors the tree to the level each node was given.|
|5|`faction`|**parallel**, per owing faction|what it controls, publishes, buries.|
|6|`people`|**parallel**, packs of ~10 by faction then location|authors each person to their level and what they know about the others in the pack.|
|7|`arrival`|one|world header, region, the way in.|

**Why the waves are safe to run at once:** after stage 3 every entity in the world has a name, so a
content call cannot invent a dangling reference — it can only refer to something that exists. That is a
property of the schedule, not a hope about the model.

**The compiled mandate.** A content call is shown *its own slice* of the namespace and nothing else,
assembled in Go. Before this, every call carried the whole growing document — ~13,300 input tokens per
call, the world re-sent nineteen times, and the output was worse for it because restating what exists
crowds out authoring what does not.

**`depth` (1–5, default 1) buys BREADTH, never richness.** How much world: a locality, a town, a city and
its region, a province, continents. How deep any single thing goes is relevance's job. Nothing raises it
for the user yet, and the projection of ~400 tokens per entity is **a projection, not a measurement** —
the first thing to measure on the next live run.

### Two bugs this stage found, both of which would have cost a live build

1. **The merge SKIPPED a second mention** of an existing location, faction, concept or object — only
   actors and events deepened. Under a design where the scaffold names things thin and a later wave fills
   them in, every description would have been silently discarded and the belt would then have refused the
   world for content that was in fact authored. Merging now deepens, never overwrites, and **relevance
   ratchets**: a later answer may raise a level, never lower one.
2. **The commit wrote an empty private perception** for anyone with no `hiding`. Under the old belt
   everyone carried one; at relevance 1 nobody does. That put junk in the perception ledger which read as
   one secret shared by everybody who had none. A person who hides nothing now writes no secret.

### FIRST LIVE RUN, 2026-08-31 — refused at 777 s, and what it taught

Andantes brief, `deepseek-v4-flash` on `coreweave`, `json_schema` strict. **16 calls, 32,018 output
tokens, $0.036, refused at 12 min 57 s** on *"nothing joins the places, so no one can leave the room you
start in"*. Three defects, none of them the model's:

**1. `unknown field "relevance"` killed 3 of 16 calls.** Told that every entity carries a level, the
model applied it to `ways` and `arrival_candidates` too — whose Go shapes did not accept it. Refusing a
four-minute call over a field the belt never reads is the expensive kind of pedantry, so both shapes now
accept it, unread.

**2. Connectivity and canon had no owner.** The scaffold returned every one of five locations at
relevance 1, so nothing owed a description, so `anyPlaceOwing` skipped the geography wave entirely — and
geography is the only thing that authors `ways` and `history`. **Neither of those scales with relevance:
a relevance-1 location still has doors, and canon needs one event whoever is standing there.** The gate
is gone; geography runs per top location, always.

The residual risk this exposed is worse than the refusal that revealed it: the belt demands only *one*
way and an exit from the arrival, so a partly-connected world **passes** while most of it is unreachable.
The test now asserts **no orphan locations** — every location a wave named is joined to something.

**3. A world can be coherently unplayable.** Five locations and ten people, all at relevance 1, is a
correct reading of "most of what you name is relevance 1" and a world with no scene in it: nowhere
described, nobody with a standing or anything they will not say. `ensurePlayableFloor` now guarantees the
minimum — **one** location described, **one** person who can be dealt with — and runs *between* the
namespace and the content waves so whatever it promotes actually gets authored. The prompt asks for the
floor as well, because code stepping in should be the exception.

**And a fake laxer than its seat hid one of these.** The fill fake emitted `ways` from the *scaffold*
while the prompt says scaffold is names-only and geography joins things up. With the fake honest, deleting
the geography gate fails loudly; before, it passed.

### SECOND LIVE RUN, 2026-08-31 — the design works; a late pass broke it

Same brief and seat. **41 calls, 108,241 output tokens, $0.094, refused at 806 s** — but the shape is
now proven:

|measured|value|
|---|---|
|entities named|**104** — 26 locations (4 top), 16 factions, 14 concepts, 48 people|
|complete at relevance 1|**91 of 104**|
|owed at 2+|7 locations, 6 factions, 13 people|
|calls|41 (4 geography ‖, 6 faction ‖, 11 people ‖)|
|input cache hits|**484,352 of 700,622 — 69%**|
|output tokens per entity|**~1,040**, down from 2,312|

Connectivity was authored, every wave answered, and the world is **three times the size of the flat
build for 40% more money.** The ~400 tokens/entity figure in §2.9 remains a projection — the measured
number is ~1,040, and the gap is the compiled mandate still carrying more than it needs.

**What refused it:** `"Kar" is relevance 2 but has no standing`. Canon referenced 13 people who did not
exist, the closing pass authored them to pay that debt — and it runs *after* the wave that gives a person
a standing. So a late pass could name someone at a level **no later pass can author**. Same failure class
as the geography gate: a creation path with no owner.

Two fixes, and the second removes the class rather than the instance:

1. **The closing and repair passes author at relevance 1, and are told why:** canon named these people,
   nobody has met them, and anything deeper written there is content nobody will finish.
2. **`settleUnauthored` runs before the belt.** Anything still owing content is recorded at the level it
   actually *reached*, which is 1. It loses nothing — a description already written stays, it is simply no
   longer owed — and the two alternatives are both worse: throw away a thirteen-minute build over a person
   who could have been thin, or leave a relevance-2 person with no standing for the engine to meet.

This is **not** the merge ratchet. The ratchet stops one answer from lowering another's level; settling
reconciles the finished document with what is actually written in it.

### THIRD LIVE RUN, 2026-08-31 — settling worked; a reference I added broke it

**21 calls, 42,726 output tokens, $0.049, refused at 750 s.**

`settleUnauthored` did its job and is now proven live: six entities were recorded at the level they
actually reached — one person, one location, four factions — instead of the build being thrown away.

**What refused it:** `the faction "Colegio de Auscultadores" is seated in "Alto Omóplato, en el edificio de
contraventanas de hueso.", which is not a place in this world`.

A **real** location, with a description appended. This is the em-dash failure in its **third** costume —
`starts_in` cost a 234 s build on 2026-08-28 and was fixed in the prompt by quoting names on their own
line. And it is squarely my fault: I added `faction.seat` and `concept.taught_by` to the belt in the same
round that introduced them, and gave **neither a repair path**, so the only outcome available was to
refuse a finished world over a comma.

`snapCrossReferences` now runs with the other bookkeeping. Exact match first, then the longest authored
name the value *begins* with **where the name ends at a boundary** — which is how
`"Alto Omóplato, en el edificio…"` resolves to `Alto Omóplato` while `"Altozano"` does not resolve to
`Alto`. Unresolvable clears the field: both are optional, and an empty seat costs a detail where a
dangling one costs the world.

**Mutation testing corrected my own claim here.** I wrote that the longest-match protects against
`"Alto"` swallowing `"Alto Omóplato"`. Deleting the longest-match left the test green; deleting the
**boundary check** turned it red. The boundary check is the guard, and the test now says so.

**Also observed, not fixed (out of scope, worth recording):** the scaffold emitted person-shaped names in
`factions` — *"Archivera Mota"*, *"Bara Quel"*, *"Celia Firme"*, *"El Curador"* are people, not
institutions. The belt permits it because a faction named like a person is legal. The prompt forbids a
collective being a person; it does not forbid the reverse.

### FOURTH LIVE RUN, 2026-08-31 — and the correction that should have come three runs earlier

**21 calls, 99,591 output tokens, $0.063, refused at 1,201 s** on
`a way leads to "Segundo", which is not a place in this world`.

Reconciling and settling both worked — the log shows seven seats and six `taught_by` values repaired or
cleared, and three entities settled. Then one edge killed it.

**The real finding is the pattern, not the field.** Four live runs, four refusals, each on a *different*
name-shaped field:

|run|field|cost|
|---|---|---|
|(2026-08-28)|`starts_in`|234 s|
|2|`ways` missing entirely|806 s|
|3|`faction.seat`|750 s|
|4|`way.to_place`|1,201 s|

Every reference in this document is a name-shaped field, and **a model writing prose fills name-shaped
fields with prose.** Three times I repaired the field that had just failed, and the next run failed on the
next field. That is a method error, and it is mine: the disease is the class.

`reconcileReferences` is now **one pass over every cross-reference**, and what cannot resolve degrades in
the cheapest honest way — the choice differing per kind, because what each costs to lose differs:

|reference|unresolvable →|why|
|---|---|---|
|`way.from_place` / `to_place`|drop the edge|one connection; the belt needs one way and an exit|
|`object.where`|drop the object|nothing references an object by name|
|`history.where`|drop the event||
|`history.who` / `knowledge.holder`|drop that entry; drop the event only if nobody is left holding it|an event nobody knows cannot be perceived|
|`actor.starts_in`|**drop the person**, then reconcile canon against the survivors|placement is structural; and clearing it instead would just move the refusal to the belt|
|`faction.seat`, `concept.taught_by`|clear the field|both optional|

**Order is load-bearing:** people are reconciled *first* and canon *second*, so a dropped person cannot
survive as a participant or a knowledge holder. Reconciling canon first would have turned a repaired
reference into a dangling one.

**And one place where the model was right and my rule was wrong.** `taught_by` accepted only a faction,
and a live build filled it with *"Auscultadora Mayor Del Vas"* six times — because **a craft is taught by
a person.** The belt now accepts a person or a faction, and clearing those was throwing away true content
to satisfy a rule nobody had thought about.

### FIFTH LIVE RUN, 2026-08-31 — and sorting the refusal surface once instead of six times

**17 calls, 39,584 output tokens, $0.037, refused at 1,460 s** on
`"Colegio de Auscultadores de Ossa" is both a person and a place`.

A **namespace collision** — a fourth distinct class, after three runs of repairing one field at a time.
So the surface is now sorted **once**, by whether a finished world can honestly be repaired into a valid
one. `reconcileDocument` runs the repairs in dependency order; nothing in it authors anything.

|refusal class|pass|why it is repairable|
|---|---|---|
|dangling reference|`reconcileReferences`|the name before the prose is real|
|level content missing|`settleUnauthored`|it reached the level it reached|
|closed-set violation|`normaliseClosedSets`|one word from a fixed list|
|malformed row|`dropMalformed`|drop the row, keep the world|
|**namespace collision**|`resolveNameCollisions`|one name belongs to one kind|
|**structurally empty**|**REFUSE**|no locations, nobody in it, no history, no way out — bookkeeping cannot rescue that, and the repair pass gets one try|

**`I-3` stays a refusal.** The player holding knowledge they did not earn is not repaired quietly, because
deleting the leak would hide a real defect.

**Collision precedence is by how much depends on the name, never by what is more interesting:**
`location > person > faction > concept > object`. A location is what people stand in, ways join, events
happen in and objects sit in; dropping one orphans everything above it. An object is referenced by
nothing.

**Duplicates within a kind MERGE** using the same deepening the waves use, so two half-answers about one
thing become one whole answer rather than one being thrown away.

**Closed-set defaults add nothing:** neutral tension, an open way, a moderate strength — and knowledge
held by having been **told**, never `direct`, because direct knowledge is a claim about presence and
defaulting to it would invent a witness at an event they missed.

### Corrections to this document's own earlier claims

- §1.2's "six flat batches" and the descent/ascent pair are **superseded** by the table above.
- The fragment belt may **not** check depth. The scaffold legitimately names a relevance-3 person before
  anybody authors them, so a fragment-level depth check refuses the scaffold for doing its job. Depth is
  checked at the document, after every wave, where the repair pass can still answer it.
- `tension` is **structural**, like placement: one word from a closed set, and the engine's
  `location_state` enum refuses an empty one. Only the *description* scales with relevance. Found the
  honest way — the belt passed and the commit failed on `tension  not in enum`.

## 3. Open questions

1. **Per layer or per item for people?** One call per layer is fast and averages over everyone. One call
   per person is the only way each person's private lore genuinely reasons about the specific rooms and
   factions that exist. Founder leaned per-item for people; unruled for the rest.
2. **Where do goals and example phrases land at transcription?** No home today.
3. **Do factions hold institutional perceptions** — "the College's official position" as distinct from
   what its members believe? The engine allows it (`perception_record` is keyed by holder). Deferred to
   avoid doubling the model.
4. **Does the second pass survive?** The layered loop may remove the need for a separate `revise` layer,
   or depth may still want one.
5. **Concepts layer position** — between factions and people is the natural reading, pending `SPEC-043`.

---

## 4. What this replaces

- The flat six-batch schedule, and every mechanism built to survive its ordering: the debt ledger, the
  in-context nudge, and the closing passes. Those were correct responses to a schedule that named things
  before they existed. Under the layered loop they should become unnecessary; they are removed only once
  a build proves it.
- Design §7's "rules are the work plan". The identity's rules remain **law the fill prompt must honour**
  and are no longer the schedule. `rule.kind` is descriptive; nothing dispatches on it.
