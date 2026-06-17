# Chunk 5 — Play Loop: architecture capture (pre-brainstorm)

**Status:** design notes, NOT a chunk spec. The Chunk-5 brainstorm (topics 2–8) is not finished;
this records the event-model + cognition-layering discussion so it isn't lost (D-6). Nothing here
amends the frozen engine — engine changes are filed as ADRs **with running-code evidence** at
implementation time (D-9). Intended location on commit: `docs/superpowers/specs/`.

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
**substrate NPCs think with.** An NPC perceives the world through the same perception bundle the
Compendium renders and the narrator is fed (ADR-020, no omniscient pass), holds beliefs on top, and
*acts on them*. Misperception → misaligned behaviour is a **feature** (an NPC can act cold because it
misread a glance).

**Belief/appraisal updates** live in the per-actor perception layer (NOT a separate spine), each
**linked to its trigger via provenance** (B-10 linked_*): canon event → actor perceives it → belief
updates, linked back. **Event-driven cognition (new design rule):** NPCs update beliefs *when they
perceive something*, never on a free-running idle loop — this guarantees every belief-update has a
trigger and bounds cost/drift.

**Inference** is already a valid knowledge path (B-2). The model's inferred conclusion is **logged**
as the mental event, so replay replays the conclusion rather than re-running the model (same
determinism trick as narration, consistent with ADR-009 + B-5 append-only). **Residual frontier:**
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
2. **Narration** — Seren, output side, perception-bound (ADR-020).
3. **NPC cognition** — perceive → believe → act.

Chunk 5 takes only the **safe** one to build first (narration). Adjudication (deciding *uncertain*
outcomes — the LLM deciding what's *true*) and NPC cognition are deferred — those are where the model
enters the trust path.

**Leg plan (one hard thing at a time, mirroring Chunk 4's two-leg split):**

- **Leg 1 — engine harness (NOT a playable product slice).** Prove the deterministic perception
  engine — events → state → perception-generation → replay — with **fixture events** fed directly,
  independent of the model. This isolates the highest-correctness-risk piece (who-perceives-what on
  *generated* events) and is validated *through the existing Compendium* (Chunk 4) + replay.
- **Leg 2 — the LLM bridge + narration.** prose→events (schema-constrained) + perception-bound
  narration. New risks: mapping correctness and the canon-authority boundary (the model must not emit
  events the player didn't take).

**Thin-slice action set:** the perception primitives only — **presence (move) + communication (say)**,
maybe possession. These are deterministic *and* cover the axes knowledge propagates through. Adjudicated
actions (persuade/attack/search) are a later slice.

---

## 8. Registration routing

**Already law — cite, do not re-file:** B-1, B-2 (incl. inference + propagation + common knowledge),
B-3, B-4, B-5, B-6, B-7, B-10, C-5, C-10, ADR-009 / D-1, ADR-017, ADR-020.

**Write now (this doc + ledger):** this design-capture doc; the deferred-item ledger entries below.

**File later as engine ADR(s) WITH running-code evidence (D-9; frozen engine):** the canon event
spine (§3, reconciled against the Master DDL), the perceivability flag on AttributeChanged, the two
perception-generation triggers if they extend the engine. Candidate register rule (B/D), not engine
ADR: "NPC cognition is event-driven, never free-running."

**Open (decide in the Chunk-5 brainstorm, topics 2–8):** loop skeleton / turn cycle; exact thin-slice
action set; the canon-authority boundary for decomposition; the Chunk-5 operator gate definition.

---

## 9. Proposed ledger entries (`docs/open-spec-items.md`, SPEC-012+)

> Numbers assigned on commit (2026-06-16): the live ledger's last entry was SPEC-011 (the
> payload-vs-schema CI test), so these six take the next free numbers SPEC-012…017 as listed below.
> Format mirrors SPEC-009/010.

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
