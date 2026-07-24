# Gate & State Model — v2 (DESIGN)

> **⚠ SUPERSEDED IN PART (2026-07-22/23) by the FINAL set**
> (`docs/superpowers/specs/chunk-5.5-final/`, esp. `FINAL-action-contracts.md` + the RULINGS files).
> Founder rulings after this doc was written:
> 1. **`within-load` is DEAD as a gate blocker** (§2 list, §3 ObjectRelocated row, §7). Weight never
>    blocks — it CONSEQUENCES: the engine eagerly recomputes `carried_weight` on any carry-chain commit
>    and writes/clears the seeded `encumbered` status (movement −100%). Volume (`has-room` over
>    `max_room`/`occupied_room`, computed at ask-time) remains the only blocking dimension.
> 2. **The §5 predefined status-effect catalog is superseded by A11-final:** no enumerable catalogs —
>    the engine hardcodes the physics grammar; the LLM mints the status/modifier vocabulary as typed
>    rows inside that grammar. `not-blocked(axis)`-via-catalog moves out of the gate.
> 3. **"The gate is the canonization point (D-1)" is imprecise** (§1, header): ruled — the gate READS
>    canon and is structural-only; it writes nothing. The canonization point is COMMIT. D-1's "only the
>    engine writes canon" refers to the whole pipeline, not the gate stage.
> Where this doc and the FINAL set conflict, the FINAL set wins. Full doc cleanup deferred (founder:
> "we will clean the whole documentation later").

**Status:** design proposal. Grounded in §2 (closed sets / compose from primitives), §3 (event spine), §10
(action-driven time), **SPEC-017** (move-validity), **SPEC-018** (spatial: distance/speed/time),
**SPEC-013** (adjudication of uncertain outcomes), **D-1** (the gate is the canonization point). **NOT
canon.** The **status-effect catalog** (§5), **object-physics** model (§7), and **comprehension/language**
model (§8) are new → file as new SPECs.

> **v2 correction.** v1 forced every event type through a uniform six-column matrix, producing
> forced/meaningless cells. **v2: a library of check primitives; each event type composes only the ones it
> needs.** And — see §3/§6 — not every event type is even a *gated action*; some are **state-writes**
> emitted by resolution, gated only by provenance.

> Thin-slice reality: the shipped gate is **reachability only** (move + say). Everything below is the
> richer model the world-acts / adjudication tiers need.

---

## 1. Principle

The gate is the **physical-possibility / authority** check against canon (SPEC-017). It **reads canon
directly** (omniscient, §13) and is the **canonization point** (D-1). Each gated action has a predefined
rule = a composition of check primitives; **the engine derives and runs it, the LLM never selects it**
(§9, *state-computable predicate*).

**The gate is structural and artifact-agnostic — its defining constraint.** It knows the *event* (type,
target, value) and *canon* (the registry + catalogs). It does **not** know what *kind* of thing a target
is, or what states it "should" have — that would require a catalogue of every artifact and its
properties, which we refuse to build. So the gate may only ask questions answerable from the event +
canon alone: `exists(target)` (registry), `in-reach`/`reachable` (spatial), `can-act`/`not-blocked`
(actor + status catalog), and the contracted access/authority of the dimension being touched. It asks
**nothing** about what the thing *is*. The moment a check needs "a door has an `open` state" or "shoes
don't make fireballs," that check is **not** a gate check — it is **adjudication**.

**Consequence: the gate is permissive by design.** It rejects only the **structurally broken** — no
target (`exists`), out of reach (`in-reach`), can't act (`can-act`) — i.e. *broken references*. It lets
through everything structurally fine but semantically absurd ("I shoot a fireball from my shoes" passes
the gate; **adjudication** fails it, reading the shoes' real properties — same shape as "a golem from a
stick"). Catching absurdity requires world-knowledge: exactly what the adjudicator has and the gate must
not. **Structure at the gate; meaning at the adjudicator.**

---

## 2. Check-primitive library

A gate rule is a **subset** of these, parameterised.

**Existence & validity**
- `exists(target)` — the referenced entity/place exists and is current. (Schema-validity of the event
  itself is the decompose leash — §2 of the architecture notes — not a separate gate primitive. The gate
  never checks "does this entity legitimately *have* this attribute"; that would need an artifact
  catalogue.)
- `destructible(target)` — a property the *causing action* reads, see §6.

**Capacity** (object physics, §7)
- `has-room(container, object)` — object's volume fits the container's remaining volume budget.
- `within-load(carrier, object)` — object's weight fits the carrier/container's remaining max-load.

**Actor readiness**
- `can-act(actor)` — conscious / not incapacitated. *(universal to actor-driven actions)*
- `not-blocked(actor, axis)` — no active status **prevents** this axis (status catalog, §5). Modifiers
  don't block here — they feed `fits-time`.

**Spatial** (SPEC-018)
- `reachable(actor, place)` — a path exists (movement destination).
- `in-reach(actor, object)` — close enough to physically manipulate / hand over.
- `fits-time(duration, budget)` — duration (= distance ÷ speed, modifiers applied) fits the beat/scene.

*(Earshot / line-of-sight is a **reception** condition — who perceives a `Communicated` (§8) — not a gate
check.)*

**Permission & power**
- `accessible(target)` — not locked / physically barred (a door, a container).
- `has-authority(actor, resource)` — owns it / controls its access (a **rights** change).
- `has-capability(actor, action)` — whether the actor has *any* power for this class (magic/skill). For
  create/destroy this is a **fast-fail input to adjudication**, not a hard gate (§6).

**Causal integrity**
- `valid-provenance(change)` — the state-write traces to a **licensed cause** (a resolved action / world
  rule), never a free assertion. The single guard on every state-write (§3, §6).

---

## 3. Per-event-type rules

| event type | rule |
|---|---|
| **ActorMoved** | `can-act` · `not-blocked(movement)` · `exists(dest)` · `reachable(dest)` · `accessible(dest)` · `fits-time(dist÷speed)` |
| **Communicated** | `can-act` · `not-blocked(channel)` — **production only** (gagged blocks `say`, not `show`). Recipient-present is a *precondition* (interrupt); who-hears / with-what-fidelity is *perception* (§8). **Neither is a gate check** — you can shout into an empty room and the saying still happened. |
| **ObjectRelocated** | `can-act` · `not-blocked(manipulation)` · `exists(object, dest)` · `in-reach(object)` · *(take-from)* `accessible(source)` · *(give)* `in-reach(recipient)` · *(container/actor dest)* `accessible(dest)` · `has-room(dest)` *[vol]* · `within-load(dest)` *[wt]* · `fits-time` |
| **Ownership / AccessChanged** | `can-act` · `exists(resource, grantee)` · `has-authority(resource)` |
| **EntityCreated** | *not a gated action* — an **adjudicated intent** (SPEC-013), then a state-write gated by `valid-provenance`. See §6. |
| **EntityDestroyed** | *not a gated action* — outcome of an adjudicated **causing action**; damage = `AttributeChanged` until a threshold; terminal removal is a state-write gated by `valid-provenance`. See §6. |
| **AttributeChanged** *(actor-driven — player **or** NPC — "open the door", "activate the wand")* | `can-act` · `not-blocked` · `exists(target)` · `in-reach(target)` — **structural only**. Whether the change actually *takes effect* (does the wand fire? the antidote cure?) is **adjudication**, not the gate. |
| **AttributeChanged** *(emitted as an outcome — a resolved intimidation writing `composure ↓`)* | `valid-provenance` — the resolved cause is the guard; resolution already happened. |

**Two origins — the real organizing line (it is *attempt vs consequence*, not player vs NPC).** A state
change is either an **actor-driven attempt** — *any* actor, player **or** NPC, intends to change the world
(move, communicate, relocate, open/activate/drink, lock) → a **structural gate** (`can-act`, `exists`,
reach/range, contracted access), with any *semantic* outcome handed to **adjudication**; or an **emitted
consequence** — resolution / adjudication / a world rule writes a resulting state that nobody attempted
directly → gated by **`valid-provenance`** (it traces to a licensed cause). `AttributeChanged` appears in
**both**: Mara locking the door is an actor-driven attempt (structural gate); a resolved intimidation
writing `composure ↓` is a consequence (provenance). Same event type, different origin, different gate.

**No NPC bypass.** A cognition-proposed action (Mara decides to lock the door) hits the **identical**
structural gate as a decompose-proposed one — cognition proposes, the gate commits (ADR-009 / D-1). The
gate doesn't know or care whether the proposer was decompose or cognition; a "trusted NPC" path would be a
consistency/leak hole. `EntityCreated/Destroyed` are emitted consequences of adjudication (§6);
`ActorMoved`/`Communicated`/`ObjectRelocated` are actor-driven physical attempts; `Ownership/AccessChanged`
is an actor-driven non-physical attempt.

---

## 4. The non-obvious primitives (where the bugs hide)

- **`has-capability`.** NOT a deny-by-default gate — create/destroy are **adjudicated** (§6). Capability is
  an **input** the adjudicator reads, plus an optional fast-fail ("no creative power"). The plausibility
  ruling is the real check; there is **no recipe registry**.
- **`has-authority` vs `accessible`.** Different barriers: *authority* is ownership/rights ("you can't give
  what you don't own"); *accessible* is a physical lock/bar. A room can be `reachable` yet not `accessible`.
- **Ownership / AccessChanged is the *writer* for `accessible`.** Locking a door is an AccessChanged that
  flips the very state the gate's `accessible` check reads — a write/read pair, no new machinery. Two
  sub-modes (transfer-ownership · grant/revoke-access) share one `has-authority` check; **no reach or
  presence**; a *forced* transfer is plain, a *refusable* one adjudicates (hybrid, like `give`).
- **`valid-provenance`.** The guard on every state-write — the change must trace to a licensed cause. This
  is what stops `AttributeChanged` / `EntityCreated` from being a free-assertion escape hatch.

---

## 5. Status-effect catalog (feeds `not-blocked` + `fits-time`)

A status (`tied`, `limping`, `gagged`, `blinded`) is set via **AttributeChanged**. **Alone it means
nothing.** A **predefined catalog (the contract)** maps it:

```
status → { impacts: [action axes], effect: <effect> }
```

Two effect kinds: **`prevent`** (hard-gates `not-blocked` on that axis) · **`modify(param, factor)`**
(scales a parameter — feeds `fits-time`, never gates). The engine applies it **generically**; **new
statuses are new catalog rows, never new gate logic.** Effects compose from a small closed primitive set
(`prevent`, `modify-speed`, `modify-perception`, `modify-reach`, …).

```
tied     → { impacts: movement,                effect: prevent }
limping  → { impacts: movement.{walk,run,swim}, effect: modify-speed ×0.5 }   # not fly
gagged   → { impacts: communicate.say,          effect: prevent }
blinded  → { impacts: [perceive.see, communicate.show.receive], effect: prevent }
```

---

## 6. Creation & destruction are adjudicated, not gated (REVISED)

There is **no item/recipe registry** and **no per-thing simulation** — pre-enumerating what can be made
would mean simulating a fantasy world's entire catalogue just to answer a gate. Instead, creation and
destruction are **adjudicated intents** (SPEC-013), the *same machinery* that rules whether Mara cracks:

- "I craft a golem from a stick" → an **intent**, judged for plausibility against the actor's profile + the
  materials + canon. A stick → a golem is absurd → **fails** ("you whittle the stick; it stays a stick"). A
  golem-artificer with a proper core and a ritual might get a **yes**. Same call, different inputs.
- On success, `EntityCreated` commits, gated only by **`valid-provenance`** (it traces to the ruling).
- Destruction is the same: the **causing action** (attack / burn / break) is adjudicated, reading the
  target's `destructible` property and the actor's means; **damage rides as `AttributeChanged`** until a
  threshold; the terminal **`EntityDestroyed`** commits with provenance.

So an actor's **capabilities** (magical power, skills, the means at hand) are **inputs the adjudicator
reads** — plus an optional **cheap fast-fail** ("no creative power of any kind" → instant no, skip the
adjudication call). They are **not** a deny-by-default gate, and there is no registry to maintain.

**Destroy cascade (a commit obligation, not a gate):** when an entity is removed, its contents **spill**
(`ObjectRelocated`), ownership **lapses**, and perceptions/relationships pointing at it **persist as
stale** — never hard-deleted (append-only, replay-safe).

---

## 7. Object physics — size, weight, capacity (NEW)

Two **independent** dimensions on every object, both pure arithmetic (no LLM):

- **Volume.** A `size` 1–10 (`10` = 1 m³); each tier holds **4× the previous tier's volume**, so size *n* =
  `4^(n-1)` base units. A container has a `volume_budget`. `has-room` ⟺ `used_volume + 4^(size-1) ≤
  volume_budget` (e.g. four size-2 objects fill a size-3 budget of 16).
- **Weight.** A `weight` on every object. Every carrier (actor) and container has a `max_load`.
  `within-load` ⟺ `used_weight + weight ≤ max_load`. A carrier's `max_load` **is** its strength dimension.

The two are **orthogonal** — a small black hole has tiny `size` (fits the pouch) but enormous `weight` (the
pouch's `max_load` refuses it).

**`movable` dissolves.** "Fixed/anchored" is the degenerate weight case. Keep an optional `fixed: true`
boolean **only** because a wall is *destroyable but not relocatable*; both gate identically.

**Encumbrance (flag, not built):** an actor's `used_load / max_load` ratio → feed it to `modify-speed`
(status catalog), tying weight back into movement's `fits-time`. Later.

---

## 8. Reception, comprehension, and the three consumers of state

The same state model is read by **three** stages, not one — the gate is only the first.

| consumer | side | uses state for |
|---|---|---|
| **Gate** | canon (truth) | **possibility** — can the action happen at all |
| **Resolution** (SPEC-013) | canon (truth) | **outcome** — what actually happens: reception-gated effects + physical modifiers |
| **Perception / fan-out** | filter | **reception** — who perceives it, with what fidelity |

**Reception has two axes:**
- **Sensory** (status catalog): `deaf → prevent perceive.hear + say.receive`; `blind → prevent
  perceive.see + show.receive`.
- **Comprehension** (NEW): a `Communicated` event carries a **language**; an actor holds **known
  languages**; fan-out compares → **full content** (match) or **content stripped, act-only** (mismatch).
  The non-comprehending listener perceives *that Kade spoke, at whom, the tone* — not the meaning (B-7).
  Binary understand/not for v1; partial fluency later.

**Tie to resolution:** a **meaning-dependent outcome** (verbal intimidate / persuade) lands only if the
target **received** *and* **understood**. Resolution computes **reception first**; a deaf or
non-comprehending target can't be talked into anything. Physical states (`limping`, …) feed resolution as
**modifiers** on contests.

**No-leak holds.** Resolution is **truth-side** — it sees the hidden state (Mara's deafness), produces the
true outcome (the threat didn't land), and the actor learns only from the **perceivable result** and
**infers**.

**Implementation note:** because resolution may be an LLM, involved actors' relevant states **must be in its
input**, or it rules outcomes that ignore deafness, language, and physics.

---

## 9. Axis taxonomies (closed sets to define)

- **Movement modes:** `walk · run · swim · fly · climb` (SPEC-018).
- **Action axes a status may impact:** `movement(.mode) · communicate(.say/.show) · perceive(.see/.hear)
  · manipulate · …`

---

## 10. Scope & grounding

- **Thin slice:** gate = reachability only (move + say).
- **This design:** **SPEC-017** (`can-act`, `not-blocked`, `exists`, spatial-reach, `accessible`,
  `has-authority`) + **SPEC-018** (`reachable`, `fits-time`, movement modes) + **SPEC-013**
  (creation/destruction & contested outcomes via adjudication) + **two new contracts**: status-effect
  catalog (§5) and object-physics size/weight (§7).
- All **DESIGN**; file as specs with running-code evidence (D-9) when built.

## 11. Open questions

1. The exact axis taxonomy (which movement modes / action axes ship first).
2. Where the **catalogs** live (DB tables vs config) and how they version.
3. `valid-provenance` enforcement — how the engine verifies a "licensed cause" (tie to the causal-bundle /
   provenance machinery).
4. Remote `Communicated` (non-co-located channels) — in scope, or co-located only for now?
5. The longer-than-a-beat move — reject, split into multi-beat travel, or jump the clock? (SPEC-018 × §10).
6. **Comprehension model** — language representation/compare at fan-out (binary v1); and the **wiring** that
   puts involved actors' states into the resolution LLM's input.
7. **Ownership / AccessChanged** — `has-authority` ownership-only, or include **delegated control**? v1?
8. **Destroy cascade** edge cases — spill vs consume (fire) depends on the *means*; confirm rule + where
   enforced (resolution/commit, not the gate).
