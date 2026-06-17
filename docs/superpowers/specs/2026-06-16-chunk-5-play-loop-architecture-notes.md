# Chunk 5 — Play Loop: architecture capture (pre-brainstorm)

**Status:** design notes, NOT a chunk spec. The Chunk-5 brainstorm (topics 2–8) is not finished;
this records the event-model + cognition-layering discussion so it isn't lost (D-6). Nothing here
amends the frozen engine — engine changes are filed as ADRs **with running-code evidence** at
implementation time (D-9). Intended location on commit: `docs/superpowers/specs/`.

**Updated 2026-06-17:** §§1–6 unchanged. §7 (leg plan) updated for incremental-apply. New §§8–12
capture the **turn loop**, **world-pushback structure**, the **action-driven time model**,
**duration-as-world-data**, and the **spatial-engine split** decided in the 2026-06-17 session.
Registration + ledger (now §§17–18) extended.

**Updated 2026-06-17 (cont.):** §7 narrator naming corrected — it is **the narrator**, not "Seren"
(Seren is a *seeded Actor*). §8 loop expanded to a **four-stage per-event pipeline**
(gate → **resolve** → apply → interrupt). §9 gains the **perception-vs-expectation continuation
rule**. New **§§13–16**: canon-write authority (the gate), perception payload (not "bundle"),
deception-lives-in-the-world, and corrections (off-beat, leaf-only on the provenance spine).

**Updated 2026-06-17 (cont. 2):** §9 refined — the interrupt is **one rule (perception breaks
expectation), two sources** (the world acting is not itself a stop); adds the **world-time turn budget**
(third pushback face) and the **telegraph + reaction-window** UX principle (held outcome; SPEC-012/013).
New **§17 — the Chunk-5 operator gate (Q3)**; Registration + ledger now §§18–19. Companion flow diagram:
`chunk5-play-loop-flow.mermaid` (same directory).

**Validation Ladder:** this chunk carries **Q3**. Stated precisely, the *gateable* claim is:
> the trust guarantees already proven — deterministic replay (Q1 / I-1) and no perception leakage
> (Q2 / B-1 / I-3) — survive a **live, mutating, LLM-mediated** loop.

A soft quality bar (the world evolves coherently, the prose reads well) sits alongside but is not
the formal gate.

---

## 1. Scope decision

Chunk 5 is the **full play loop, with the LLM in it** — not a deterministic loop with the model
deferred. Rationale captured in the brainstorm: for free-text input, the LLM is the *decomposer*
(prose → events) and cannot be deferred from the product; there is no "deterministic playable first
leg." What *can* be sequenced is engine-proof-before-model-trust (see §7).

---

## 2. The event-vocabulary architecture (the core realization)

The engine does **not** enumerate verbs/actions (that would be an unbounded "eternal enum"). It
enumerates a **closed, finite set of canon EVENT TYPES**, because the *state* has only a few
dimensions. Open-ended player richness rides on top as:

- **content** — descriptive text attached to a perception (most of a sentiment-laden sentence is
  this; it changes no state), and
- **a generic AttributeChanged** — one event type that sets `entity.X = Y` for any `X`, so novelty
  lives in the attribute *name/value* (data), not in new event types.

The LLM's job is **prose → events**, constrained to emit only schema-valid events. This is exactly
**ADR-009 / D-1** ("modules and LLMs propose; the gate commits"): the closed schema both avoids the
enum problem *and* is the leash on the model — it cannot invent event types or write free-form canon;
malformed proposals are rejected.

> Key line to hold: **infinite English collapses onto a small closed set of state-deltas; the
> openness is on the language side (LLM), the closure is on the engine side (events).**

---

## 3. Canon event spine — PROPOSED (reconcile against Master DDL; file as ADR with evidence)

Status: **proposed vocabulary**, to be reconciled against the existing engine event model
(ADR-001, Master DDL) and filed as engine ADR(s) *with running-code evidence* per D-9. The seed
already contains `move` and genesis-style creation, so some of this exists; the rest is additive.

| Event | Changes | Perceived by (generation rule) |
|---|---|---|
| **ActorMoved** | an actor's location | co-present at origin (saw them leave) + destination (saw them arrive) + the actor |
| **ObjectRelocated** | an object's physical position — relation+target: `at-place` / `in-container` / `carried-by` (covers move, drop, abandon, file-in-drawer, give, pickup) | co-present at the location(s) involved — possibly **nobody** |
| **Ownership/AccessChanged** | who owns / may access an object — *distinct from physical position* | co-present / the parties involved |
| **EntityCreated / EntityDestroyed** | an entity's existence (multi-agent creation allowed) | co-present witnesses — possibly none |
| **AttributeChanged** | a generic entity property `X = Y`; **carries a perceivability flag** (outwardly-visible vs hidden) | co-present witnesses at the moment; later discoverable on inspection iff outwardly visible |
| **Communicated** | an actor conveys information to others (say/tell/show) | addressed + co-present recipients + speaker. Generates a *told* perception (B-7: told ≠ witnessed) |

Cross-cutting:

- **Non-actor agency** — events may have no actor cause (world/rule/time-driven). Nullable
  "cause-agent", not a new type.
- **Folds in — NOT separate types:**
  - *Environmental* (volcano erupts, storm) = AttributeChanged / Destroy on an entity.
  - *Topology / access* (bridge collapses, door locks) = AttributeChanged on the connecting entity;
    connectivity is *derived*. (Its real residual is move-validity — see §6.)
  - *Time* = infra clock + **derived** decay, never per-tick log events (would bloat for zero info).

---

## 4. Two perception-generation triggers + unwitnessed truth

Perception is **generated** by the engine; it is not a player-issued event. Two triggers (both are
"observation" paths under **B-2**):

1. **Witnessing an event** live (you were co-present when it happened).
2. **Inspecting standing state on arrival** — you reach a place and perceive what's there now,
   without having seen it arrive (**discovery**).

- **Active examination** ("I study the seal") is a perception *mode* — attention buys finer
  resolution — not a canon event.
- **Unwitnessed truth (consequence, not a bug):** an event can have **zero perceivers** — true in
  canon (replays, I-1), absent from every timeline (B-1 / I-3). The abandoned-car-in-the-mountains
  case. It enters someone's knowledge only via trigger (2), on arrival.

---

## 5. Canon vs Mental layering; relationships; NPC cognition

Two layers, both logged for replay, kept distinct:

- **Canon (objective):** world facts — the §3 spine.
- **Mental / perception (per-actor, subjective):** what each actor *knows and believes*. Generated
  from canon (witnessing + arrival) + communication + inference.

**Relationships are NOT canon** — there is no objective relationship, only each actor's *belief*
about it, which can be asymmetric and wrong. This is already law: **B-3** (relationship is internal
model only, surfaces solely as ordinary sourced knowledge records — no relationship UI), **B-4**
(system never authors the player's inner state), **B-6** (contradiction lives in perception, never
canon). Sharpened here: a relationship is a **belief-perception** whose *subject* is the relationship,
held per-actor, in the existing perception layer.

**Realization (new framing):** the perception system isn't only for the inspector — it is the
**substrate NPCs think with.** An NPC perceives the world through the same **perception payload** (§14)
the Compendium renders and the narrator is fed (ADR-020, no omniscient pass), holds beliefs on top, and
*acts on them*. Misperception → misaligned behaviour is a **feature** (an NPC can act cold because it
misread a glance).

**Belief/appraisal updates** live in the per-actor perception layer (NOT a separate spine), each
**linked to its trigger via provenance** (B-10 linked_*): canon event → actor perceives it → belief
updates, linked back. **Event-driven cognition (new design rule):** NPCs update beliefs *when they
perceive something*, never on a free-running idle loop — this guarantees every belief-update has a
trigger and bounds cost/drift.

**Inference** is already a valid knowledge path (B-2). The model's inferred conclusion is **logged**
as the mental event, so replay replays the conclusion rather than re-running the model (consistent
with ADR-009 + B-5 append-only). Note: the logged item here is the **perception-layer conclusion**, a
canon-side record — distinct from narration prose, which is presentation and is *not* logged for
determinism (see §10). **Residual frontier:**
*cascading* inference (infer → act → others perceive → infer …) can run hot; bounding its depth
relates to ADR-017 (propagation bounded, Traversal Matrix). → ledger.

---

## 6. Move-validity = an engine concern (new)

Topology/access isn't an event type — it surfaces as a **move-validity / physical-possibility gate**:
the engine checks whether an actor or NPC *can* physically do an action against current world state
(locked door, collapsed bridge, not co-located with the target). This is adjudication-flavoured but
mechanical for the thin slice; uncertain-outcome adjudication is deferred (§7).

---

## 7. The three LLM roles, and the Chunk-5 leg plan

The LLM could sit in **three** places — all "propose," all gated (ADR-009 / D-1):

1. **Intent decomposition** — prose → events (input side).
2. **Narration** — the narrator, output side, perception-bound (ADR-020).
3. **NPC cognition** — perceive → believe → act.

Chunk 5 takes only the **safe** one to build first (narration). Adjudication (deciding *uncertain*
outcomes — the LLM deciding what's *true*) and NPC cognition are deferred — those are where the model
enters the trust path.

**Leg plan (one hard thing at a time, mirroring Chunk 4's two-leg split):**

- **Leg 1 — engine harness (NOT a playable product slice).** Prove the deterministic perception
  engine — events → state → perception-generation → replay — with **fixture events** fed directly,
  independent of the model. This isolates the highest-correctness-risk piece (who-perceives-what on
  *generated* events) and is validated *through the existing Compendium* (Chunk 4) + replay.
  Events are applied **incrementally with a halt hook**, not batch (see §8–§9): the interruptible
  control flow and its **deterministic triggers** are in Leg 1 scope; rich triggers defer
  (SPEC-012/013). **Partial-beat correctness** becomes a gate property (§9).
- **Leg 2 — the LLM bridge + narration.** prose→events (schema-constrained) + perception-bound
  narration. New risks: mapping correctness and the canon-authority boundary (the model must not emit
  events the player didn't take).

**Thin-slice action set:** the perception primitives only — **presence (move) + communication (say)**,
maybe possession. These are deterministic *and* cover the axes knowledge propagates through. Adjudicated
actions (persuade/attack/search) are a later slice.

---

## 8. The turn loop (beat cycle) — DECIDED (shape)

**Governing rules (checked against the register before writing):** C-5, C-6, C-7, ADR-009/D-1,
ADR-020, ADR-001, ADR-021/030, B-5, D-7, I-1, SPEC-015, SPEC-017.

One **beat** runs:

1. **Player input** (free text).
2. **Decompose** *(LLM — proposes)* — prose → an **ordered chain of canon events** (ADR-009/D-1). The
   model proposes; it never commits.
3. **Execute the chain incrementally** — for each event, in order, a **four-stage pipeline**:
   - **(a) Gate** *(can it happen?)* — impossibility / canon-authority / move-validity against current
     world state (SPEC-015, SPEC-017). On failure the step does **not** apply and the chain **halts** here.
   - **(b) Resolve** *(what actually happens?)* — the world decides the **real outcome**, which **may
     differ from the proposed action** (adjudication, **SPEC-013** — deferred; **passthrough / identity
     in the thin slice**, where outcome = intent). The resolved outcome carries its **visible-vs-hidden
     split** (perceivability, SPEC-016).
   - **(c) Apply + generate** — commit the **resolved outcome** (*not* the raw proposal) to canon — the
     gate is the canonization point (D-1, §13) — generate the perceptions it produces **from canon's
     *visible* aspect** (witnessing trigger, §4; the visible aspect may differ from the hidden truth →
     deception, §15), advance the clock by the event's duration (§10).
   - **(d) Interrupt** *(does the chain continue?)* — world pushback (C-7), per the **continuation rule**
     (§9). If it fires, the chain **halts** here, *after* this step's effects are committed.

**Four loci where truth and experience diverge** (per event): **gate** (can it) → **resolve** (what
actually happened — outcome B may ≠ intent A) → **perceive** (what the user sees — appearance may ≠
hidden truth) → **interrupt** (does the chain continue). In the thin slice **resolve / perceive /
interrupt are identity / mechanical**; each gets its real engine later (SPEC-013 / SPEC-016 / SPEC-012).
**Structure-in / behavior-deferred**, mirroring §9.
4. **Narrate** *(LLM)* — render prose from the **player's perceptions only** (ADR-020, no omniscient
   pass), up to the halt point.
5. **Return control** — the player acts again, or **Continue** advances the moment (C-6) without
   fast-forwarding the world.

**A beat resolves a *prefix* of intent, not the whole intent.** Understanding the full chain ≠
guaranteeing it runs (C-7); a beat yields zero / one / many committed changes (C-5); Continue advances
the moment, never the world (C-6).

**Two halt mechanisms, both first-class (DECIDED — "we do both"):**
- **Pre-apply impossibility gate** (3a) — *rejects* a step before it applies; nothing is committed.
- **Post-apply interruption** (3d) — *halts* the chain after a step's effects land; the prefix stands,
  the remainder never runs.

**Deterministic core vs model.** gate + resolve (where mechanical) + apply + generate + replay are the
**deterministic engine**.
The LLM is only **decompose** (2) and **narrate** (4). The trust guarantees — Q1/I-1 replay, Q2/B-1
no-leak — are properties of the deterministic core; the model sits on either side of it, gated.

**Narration is presentation, not canon.** Only the committed events (3c) are canon and are replayed;
narration prose is **not** logged for determinism. (It *may* be logged for session fidelity only — a
separate concern that does not touch replay.) This is why a non-deterministic narrator does not
threaten Q1: replay reconstructs the world from the event log, not from prose. (ADR-001 canon = events;
I-6 chatter ≠ canon; ADR-020 narrator-bound; D-1 only the gate commits.)

**Open — intra-tick ordering.** When several events share a tick they need a deterministic tiebreaker.
**The register has no established ordering field for this.** (An earlier reference to "ADR-034
(tick, beat_seq)" was recall-drift — it is *not* in the register; do not treat it as law.) If the
engine needs an explicit intra-tick sequence, it is a **new engine ADR (ADR-034+) with running-code
evidence** (D-9), decided at implementation. Captured as open, not settled.

---

## 9. World pushback: structure vs triggers — DECIDED

The pattern that matters is **interruptibility**, and it splits:

- **Structure** = the interruptible control flow — incremental apply, a halt hook after each step,
  partial-beat resolution, return-of-control, Continue.
- **Triggers** = what *decides* to interrupt.

**DECISION: the structure is in the thin slice; triggers grow.** Rationale: the loop's control flow is
load-bearing, and retrofitting it later is the worst kind of change (re-threading how every turn
resolves). It is **not** speculative — YAGNI does not apply — because the deterministic triggers
exercise the same control flow from day one. And it is cheap: **incremental apply is needed anyway**
for intra-beat ordering (move-then-speak, where each event's perception generation runs against the
state the prior event left); the halt hook is a small addition on top.

**The unified interrupt trigger — perception vs. expectation (DECIDED).** The chain advances on
**perception matching expectation**, *not* canon matching intent. Precisely: it advances from action
*n* to *n+1* **iff *n*'s perceived outcome still satisfies *n+1*'s premise.** **"Surprise" = the next
link's premise is broken** by what the player perceives. Forward, link-by-link:
- resolve B / perceive B (≠ intent A): the user sees the surprise (it breaks the next link) → **STOP**.
- resolve B / perceive A (hidden divergence): premise holds → **CONTINUE**, deception rides (§15).
- resolve A / perceive A: normal success → **CONTINUE**.

Outcome-vs-intent is **not independently a stop:** a divergence in *n* matters only if it breaks *n+1*
(queue *[kill, leave]* — the kill fails but leaving doesn't depend on it → continue; the **last** link
has no next premise → the beat just ends). This **subsumes** the old co-presence trigger (arriving to
find someone is one kind of broken premise).

**Hard constraint (Q1 / I-1):** the stop/continue decision is **canon-affecting** (it changes which
events commit) → it **must be deterministic** — a **state-computable predicate** (next-link precondition
broken), **never an LLM judgment**, or replay breaks. Scope "surprise" to **plan-relevant**
(precondition-break), *not any* novel perception (else every described room halts the chain).

**One stop rule, two sources of the triggering perception (REFINED — corrects "two doors").** The chain
stops on exactly one condition: **a perceived expectation-break.** The *world acting is not itself a
stop* — the world interacts continuously as the clock advances, and the chain rides **through** every
world interaction the player either doesn't witness or perceives without surprise (unwitnessed /
non-disruptive; B-7, B-1). A world interaction halts only when it yields a perception that breaks the
player's expectation. One rule, fed by **two sources of perception**:
- **(a) the player's own resolved action** — fires now in the thin slice via *discovery* (move into a
  room, perceive an actor, the queued action no longer holds); deterministic, no model.
- **(b) the world's / an NPC's interactions** — events that interleave; most unwitnessed or
  non-disruptive; only a perceived, expectation-breaking one stops the chain. (SPEC-012 / SPEC-013, deferred.)

**Deceptions self-unravel under (a):** the fake-death rides "loot the body" until looting makes the
"corpse" react and breaks the next premise — no scripting needed (§15).

**World-time turn budget (third pushback face — DECIDED shape).** Bounding a beat is **not** about action
*count*; it is about **world-time.** Committing actions moves the world (each event advances the clock by
its duration, §10–11), and a turn can move the world only so far before the world must react —
*"that's more than your turn"* (C-6: a beat advances the moment, not weeks of world). Mostly **not new
machinery** — it is source (b): as the chain advances the clock, the moment it crosses a pending
world/NPC event, that event fires and (if perceived-disruptive) stops the chain. Context-sensitivity is
therefore **emergent** — a fight is dense with imminent events → short beats; downtime is sparse → long
beats / time-skips — with **no per-context table, no "combat mode" flag.** The **one new piece** is a
**generous hard time-cap (backstop)** for the *sparse* case (a 10-week skip through a dead world, where
nothing interleaves): the world refuses to advance more than ~a turn regardless. Generous, so calm-time
creativity flows; it bites only at the absurd. In the thin slice (source (b) deferred) the cap is the
*only* bound. OPEN: the cap's budget; whether long actions *halt-before* or *begin as multi-turn
commitments* the world moves through (build halt-before first).

**Telegraph + reaction window (UX principle — banked for SPEC-012/013).** A disruptive NPC action should
interrupt the player at its **telegraph** (perceivable wind-up / declared intent), *not* at its committed
outcome — *"the enemy moves to smite your friend"* (room to react), not *"the enemy smited your friend"*
(fait accompli). The action's lifecycle splits: the **wind-up** is committed & perceivable (it fires the
stop-check — the one rule, keyed on "about to"); the **outcome** is **held — not committed** — until the
player has had a reaction beat, then resolves *with the reaction in it* (contested resolution). Introduces
one new concept: a **declared-but-unresolved (held) action.** Replay holds (Q1): telegraph → reaction →
resolution are committed events in order. Only for perceived + reactable actions; unwitnessed ones just
commit. B-4: author the *world*-perception, never inner state. Deferred (SPEC-012 / SPEC-013).

**Gate consequence (a stronger Q3 property): partial-beat correctness.** A chain that halts at step 2
must leave *exactly* the step-1 perceptions and *zero* step-3 perceptions. This is the core safety
property of an interruptible loop (an interrupted chain must not over-apply) — a better thing to gate
than "the full intent resolved."

---

## 10. Time model: action-driven time — DECIDED

**Governing rules (checked):** ADR-021/030 (World Clock owns time; logical tick + label, never
TIMESTAMPTZ), B-5 (append-only; tick + label), ADR-006 (three time axes), C-5/C-6, D-1, I-1.

Three candidate models; two are ruled out by existing law:

- **Wall-clock** (time passes on its own) — dies on **Q1/I-1**: replaying the log at a different real
  time yields a different clock. Non-deterministic. (Also violates ADR-021/030: fictional time is a
  logical tick, not a timestamp.)
- **Per-beat** (each beat ticks +1) — dies on **C-6**: every Continue would advance world-time, the
  exact fast-forward C-6 forbids.
- **Action-driven** (the clock advances by the **durations of committed events**) — survives both. ✓

**DECISION: action-driven time.**
- Pure **Continue** commits no new events → **no world-time passes** (C-6 ✓).
- **Replay** reproduces the clock because it reproduces the events and their durations (Q1/I-1 ✓).
- **Beat ≠ tick** — decoupled, the same way C-5 decouples beats from messages.

**Guardrail (Q1):** durations are **deterministic engine/world data, never LLM-authored.** The LLM
proposes events (D-1); the engine assigns each event its duration from recorded state (§11). If the
model emitted durations, replay could drift.

**Composes with the interruptible loop — one decision seen twice.** An interrupted chain advances the
clock by the **prefix's** durations only: move-then-speak, halted after the move, advanced time by the
move's duration, not the speak's. A per-beat clock cannot represent that; **partial-beat time requires
per-action time.** So §9 and §10 are the same decision.

**The world moving on its own** (decay, scheduled world events) is itself **engine-committed events**
with a null cause-agent (§3) → still event-driven. This is the mechanism behind §3's "time = infra
clock + derived decay."

---

## 11. Duration is world data, not a type table — DECIDED

A duration is **not** a property of the event *type*. The same type (a move) takes wildly different
times by instance — next room vs. next city; A→B ≠ A→C. A static per-type table cannot express this,
because what varies is the **world**, not the verb. *(This corrects an earlier proposal of a
per-event-type duration table — rejected.)*

**Duration is read from world state** — a canon attribute of the specific configuration the event
operates on (for a move, the traversal cost of the specific geometry; consistent with §3, where
topology lives as attributes on the connecting entity). The engine reads it and sums it into the clock.
**Q1 holds** because the value is recorded canon, read on replay — not regenerated.

**Where the first value comes from** splits by whether the facts are coupled:

- **Independent facts (e.g. process / craft durations):** a 100-year craft and a 1-week craft do not
  constrain each other. **LLM establishes the duration on first occurrence → engine records it →
  reuse** on repeat and on replay. No coherence problem exists, because nothing couples them. *(The
  "trigger then record" model — correct here.)*
- **Coupled facts (e.g. spatial distance):** A→B, B→C, A→C must live in one consistent geometry.
  Recording them independently is exactly what *creates* contradictions → handled by the spatial
  engine (§12).

**The principle (general):** **record independent facts directly; for coupled facts, record the
*generating structure* and derive the facts.** Independent recording is safe only when the facts are
independent.

---

## 12. The spatial engine (deferred subsystem) — DECIDED

**DECISION: spatial gets its own engine** — a bounded subsystem beside canon, perception, cognition
(SPEC-012), and adjudication (SPEC-013). It is §11's principle applied to geometry.

**Model:**
- **Record coordinates** per place — the only recorded *geometric* fact.
- **Distance = derived** — the magnitude of the vector between two coordinates.
- **Travel time = distance / speed.** **Speed** is the other recorded input — a property of the
  **mover / mode** (walk, mount, …), not of the place.
- Everything else (distance, travel time) **derives**. Two recorded inputs (coordinate-per-place,
  speed-per-mover); the rest is computed.

**This dissolves the coherence problem — it does not merely contain it:**
- Coordinate-derived **distances are coherent by construction** — three of them cannot violate the
  triangle inequality, because they read off one geometry.
- **Travel time has no triangle-inequality constraint.** So "A→C in two days" is either impossible
  (equal speed → geometry forces consistency) or *legitimate* (a road/portal → higher speed → genuinely
  faster). The contradiction either can't be authored or has a real cause.
- **Irregularity (terrain, roads) lives in the speed layer**, keeping the geometry pristine.

**Distance ≠ reachability — do not conflate.** Coordinates give *how far*; **connectivity /
move-validity** (SPEC-017, §6) gives *whether you can go at all* (a wall, a locked door, no path). Two
places can be coordinate-close yet unreachable.

**Why the split is the win:** it collapses the LLM's spatial error surface. The model no longer judges
travel times per action (which it cannot keep coherent) — it **places a new location once** (assigns a
coordinate), and everything derives. A single internally-coherent fact per place is something the model
can produce without contradicting itself. (Q1 guardrail rides along: speed is recorded/deterministic,
not judged per traversal.)

**Owned by the engine, deferred (not now):** coordinate frame — single global vs **nested frames**
(containment — the cup in the tavern in the town — likely forces nesting); dimensionality (2D / 3D);
non-geometric moves (portals/teleport as explicit overrides, containment changes as ~instant);
record-on-first-use for emergent geography; rich speed/terrain modifiers.

**Thin slice:** give the two or three places **coordinates** + one **flat default speed** → derived
distance → derived travel time → the clock advances on real derived durations. `say` ≈ 0. This exercises
the whole action-driven-time + spatial-derive path on minimal data; nesting, terrain, portals, and
record-on-first-use all defer. **The model is adopted; the machinery is minimal.**

---

## 13. Canon-write authority — the gate writes canon (DECIDED)

**Who writes canon? The gate — and only the gate.** It is the **canonization point** for everything:
the LLM (decompose), modules, and world-rules all **propose**; the gate **commits** (ADR-009 / D-1).
The decomposer is only the *source* of candidate player-events; the narrator writes nothing.

**The gate is the one *non*-perception-bound component.** The LLM seats read the player's partial view
(§14); the gate reads the **objective world**. That asymmetry is the whole point of D-1 — a partial,
fallible model proposes, and the component holding full objective state decides. So canon stays
**"more than perception"**:
- the gate validates proposals against **real** state the decomposer never saw (move-validity SPEC-017,
  impossibility) — believed-unlocked-but-objectively-locked is rejected, and that mismatch *is* the game;
- what's committed is an **objective event**, not a perception; perceptions are then *generated from*
  it (§4), each actor a partial view, the event true even where unwitnessed;
- **non-player writers** add canon never sourced from the player's view — world / rule events (null
  cause-agent, §3) now; NPC cognition (SPEC-012) and adjudication (SPEC-013) later.

**The LLM must not invent canon** (SPEC-015): it proposes events within the player's actual intent; the
gate enforces and commits.

---

## 14. Perception payload — what the two LLM seats consume (NOT "bundle") (DECIDED)

**Both LLM seats are perception-bound.** The **decomposer** reads the player's perception *before*
resolving (to interpret intent); the **narrator** reads the resolved beat's perceptions *after* (to
render — ADR-020, no omniscient pass). Both consume a perception-bound payload built by the **same
safety filter the Compendium uses** (the pgTAP-tested safety wall; B-1 / I-3). **The model never
touches raw canon** → Q3's no-leak claim for the live loop collapses to: *the model only ever sees
safety-filtered perception.*

**Terminology (DECIDED — do not call it a "bundle"):** in the register, **"bundle" = the causal /
validated bundle** (G3, ADR-007/008, Phase 4+). This is a **different object** — call it a
**perception payload**. *Proof they differ:* Chunk 3 assembles the perception view with **live SQL
joins**, which **G3 forbids** for causal-bundle inputs ("durable records only, never live projection
reads") — so they cannot be the same thing.

**Grounded vs reasoned:**
- *Grounded:* narrator perception-binding (ADR-020); perception-bound surfaces / no leak (B-1, I-3);
  perception isolated from canon (ADR-005).
- *Reasoned (confirm against engine **doc 06 — context assembler** + the glossary, not readable from
  chat):* that the **decomposer** (not only the narrator) is perception-bound is an extension of the
  B-1/I-3 wall (ADR-020 names only the narrator); reusing the Chunk-3/4 safety filter as the assembler
  is a design choice that *satisfies* B-1/I-3/ADR-005, not a stated rule.

---

## 15. Deception lives in the world (DECIDED — rationale)

A perception can be **flatly false** relative to canon — and that is the architecture working, not
failing. This is **B-6** ("contradiction lives in perception, never canon").

**Example (fake-death):** canon = *alive-and-faking* (hidden); perception = *dead* (the visible
appearance). Realized through the **perceivability flag** (SPEC-016, §3): the event carries a
**visible** aspect (collapses, lifeless) and a **hidden** one (still alive); perception-generation
surfaces the visible appearance, the hidden truth stays in canon, unperceived.

**No component lies — the deception is *in the world*.** The gate honestly records both truth and
appearance; perception honestly surfaces the appearance; the narrator honestly renders what the player
perceives. The **injector** is the **gate + resolve / outcome layer** (the side holding objective
truth), **never** the perception-bound LLM seats — which preserves D-1.

**Boundary (B-4):** author **world-perceptions** (the enemy appears dead) — never the **player's inner
state** ("you're certain you've won"). Keep the lie external to the world.

**Deferred:** the *capability* exists the moment canon ≠ perception (now); the thing that **decides** a
specific divergence is **adjudication (SPEC-013)** — the textbook adjudicated-attack outcome — possibly
with **NPC cognition (SPEC-012)**. Thin slice: outcomes mechanical, canon = perception, no divergence yet.

---

## 16. Corrections — off-beat, leaf-only on the provenance spine (DECIDED)

**Corrections are OFF-BEAT** — applied after a beat fully resolves and stores; they do **not** run
inside the chain and do **not** tick the clock.

**A record is correctable iff it is a LEAF in the provenance spine** — nothing depends on it (no inbound
provenance dependents on `provenance_edge`). *"Remove the going-to-the-mountain → the fall never
happens":* the going has a dependent (the fall), so it is **locked**; the fall, if a leaf, is
**correctable**. The no-dependent rule is **cascade-free by construction** — a stricter implementation
of **ADR-016** (present-forward only).

**Mechanism:** correction = **append-supersession** (B-5 / ADR-006), fully **replayable** (Q1). The
window **self-bounds** — a record stays correctable until something links to it; **no timer**
(cf. ADR-011 / C-11).

**Grounding (provenance ≠ causality):** the gate reads the **provenance graph** (`provenance_edge`;
universal provenance, I-2) — **separate** from **causality** (validated bundles, ADR-007/008,
Phase 4+). The correction gate needs only **dependency** (provenance, present today), **not** the
Phase-4 causal layer.

**OPEN (verify before planning):** the exact `provenance_edge` **edge semantics** — canon-event →
canon-event direct, or routed via perception / participant records — against the frozen Master DDL
(grep). The leaf-rule holds regardless of which; only the implementation differs.

---

## 17. Chunk-5 operator gate (Q3) — DECIDED (shape)

**Ladder claim (Q3):** the trust guarantees survive even though a **non-deterministic LLM is now in the
loop** — because the model is **quarantined**: it proposes only (D-1), is perception-bound only
(B-1 / I-3 / ADR-020), and never holds canon-authority. Q1 proved replay with *fixed* events; Q3 proves
replay + no-leak still hold when the events come from a *live, variable* model.

**The sharpest idea the gate rests on:** *narration may vary run-to-run; canon must replay.* Different
prose across runs is fine; each session's committed events must replay to that session's exact world, and
the model must never have seen or surfaced a hidden fact. (Process law: a 🪜 chunk needs an honest
Validation-Ladder answer — green CI + product "no" = not done.)

**Form** — a scripted scenario (seed world with at least one *hidden* fact the player cannot perceive),
played end-to-end through the live loop, then four inspectable checks, each producing reviewable evidence
(verify the artifact, not a green check):

1. **Replay invariance (I-1 / ADR-026).** Replay the session's committed event log; assert the rebuilt
   world is **domain-equivalent** (ADR-026 — *domain* equivalence, **not** byte-identity; volatile
   transaction-time fields like `recorded_at` are excluded, SPEC-002). Evidence: an empty *domain* diff.
   *Q1 surviving the loop.*
2. **No-leak (B-1 / I-3).** The seeded hidden fact never appears in *any* perception payload the model was
   handed, nor in any narration. Evidence: payloads + narration, grepped for the hidden fact — absent.
3. **Partial-beat correctness.** A deliberately-halting chain commits *exactly* its prefix — both halt
   ways: a gate-reject (open chest → try locked door → sit: chest commits, door rejects, sit never runs)
   and a stop-check (walk in, discover an actor that breaks the next queued action). Zero trace from the
   halt onward. Evidence: that beat's event log + perceptions.
4. **Canon-authority (D-1 / SPEC-015).** Every committed event traces to the gate committing a gated
   proposal; the model wrote nothing to canon directly. Evidence: authorship / provenance of committed events.

Founder-run, by hand: play it, review the four pieces of evidence, sign off, tag — same shape as the
Compendium gate (Q2), but the object under inspection is the live loop, not a static page. **Gate red →
stop** (process law).

---

## 18. Registration routing

**Already law — cite, do not re-file:** B-1, B-2 (incl. inference + propagation + common knowledge),
B-3, B-4, B-5, B-6, B-7, B-10, C-5, C-6, C-7, C-10, ADR-001, ADR-005, ADR-006, ADR-007/008,
ADR-009 / D-1, ADR-011 / C-11, ADR-016, ADR-017, ADR-020, ADR-021/030, ADR-026, D-7, G3, I-1, I-2, I-3, I-6.

**Write now (this doc + ledger):** this design-capture doc; the deferred-item ledger entries below.

**File later as engine ADR(s) WITH running-code evidence (D-9; frozen engine):** the canon event
spine (§3, reconciled against the Master DDL), the perceivability flag on AttributeChanged, the two
perception-generation triggers if they extend the engine; **action-driven clock advancement + per-event
duration read from world state (§10–§11), where they touch the World Clock**; the **spatial engine**
(§12); an **intra-tick ordering field, if needed — new ADR-034+** (§8).

**Promoted to the register (2026-06-17) — design-intent amendments, NOT engine-behavior claims, so
not gated by D-9 (precedent: D-3 declares the Image Platform boundary ahead of that build):**
- **D-11** (record/derive discipline) — §11's general principle: coupled facts derive from a recorded
  generating structure; only independent facts are recorded directly.
- **D-12** (spatial is a bounded subsystem — its own engine) — §12; owns geometry only, distance ≠
  reachability.
- **B-11** (event-driven cognition) — §5: NPC beliefs update only on perception, never on a
  free-running idle loop; every update carries a trigger and provenance. (Subsumes the earlier
  "candidate register rule" framing of event-driven NPC cognition — promoted, no longer a candidate.)

Still **awaiting ADRs with running-code evidence at implementation** (D-9), NOT register rules: the
action-driven clock, the canon event spine, and the duration mechanism — engine-behavior claims all.

**Open (remaining):** exact thin-slice action set (move + say; possession in or out?); **intra-tick event
ordering** (ADR-034+ if the engine needs it, §8); the **turn-budget cap value** + **long-action behavior**
(halt-before vs multi-turn commitment, §9). Resolved this session: ~~loop skeleton~~ (§8); ~~canon-authority
boundary~~ (§13); ~~the gate's own (non-perception) view~~ (§13/§17 — the gate is the omniscient truth-side
reader, opposite end of the same filter from the perception payload); ~~the Chunk-5 operator gate~~ (§17).

**Verify before planning (grep, no decision yet):** exact `provenance_edge` edge semantics against the
frozen Master DDL (§16) — canon-event→canon-event direct vs routed via perception/participant.

---

## 19. Proposed ledger entries (`docs/open-spec-items.md`, SPEC-012+)

> Numbers provisional — assign the next free SPEC-### on commit (SPEC-011 was the payload-vs-schema
> CI test). Format mirrors SPEC-009/010.

- **SPEC-012 — NPC cognition engine (deferred subsystem).** The perceive → appraise → believe →
  decide → act loop, LLM-run and event-driven, is its own subsystem beside the canon and perception
  engines. NOT in Chunk 5. **Firing trigger:** when the play loop + write-side perception generation
  are proven and the world needs autonomous NPCs (post-Chunk-5).

- **SPEC-013 — Outcome-resolution / adjudication engine (deferred).** Resolving *uncertain* actions
  (persuade, attack, search) requires a ruling (rules+dice or LLM) — the path that puts the model in
  the trust/canon-authority position. **Firing trigger:** when the action set extends beyond the
  mechanically-resolvable primitives.

- **SPEC-014 — Cascading-inference depth bound.** Inference → act → others perceive → infer can
  cascade unbounded; bound its depth/fan-out (cf. ADR-017 Traversal Matrix). **Firing trigger:** when
  NPC inference is implemented.

- **SPEC-015 — Decomposition reliability + canon-authority boundary.** prose → events mapping must be
  correct, and the LLM must not emit events the player did not take (inventing canon). **Firing
  trigger:** when the Chunk-5 Leg-2 LLM bridge is built.

- **SPEC-016 — Per-attribute perceivability model.** AttributeChanged needs an outwardly-visible vs
  hidden flag deciding discovery-on-inspection (Jimmy's missing arm shows; Sabin's secret PhD
  doesn't). **Firing trigger:** when AttributeChanged is implemented in Chunk 5.

- **SPEC-017 — Move-validity / physical-possibility gate.** Engine checks an action is physically
  possible against current state before applying it. **Firing trigger:** when ActorMoved / actions
  land in Chunk 5.

- **SPEC-018 — Spatial engine (deferred subsystem).** Coordinates recorded per place; **distance
  derived** from coordinate vectors; **travel time = distance / speed** (speed a mover/mode property).
  Subsumes the geometric case of coupled-fact coherence — distances are coherent by construction.
  Owns: nested coordinate frames, non-geometric move overrides (portals/teleport), record-on-first-use
  for emergent geography, rich speed/terrain modifiers. Distinct from reachability (that is SPEC-017).
  **Standing rule (general):** *record independent facts directly; for coupled facts record the
  generating structure and derive.* **Firing trigger:** when travel must span more than a hand-authored
  handful of places, or emergent geography appears (post thin-slice). The thin slice uses hand-set
  coordinates + a flat default speed and needs none of the deferred machinery.
