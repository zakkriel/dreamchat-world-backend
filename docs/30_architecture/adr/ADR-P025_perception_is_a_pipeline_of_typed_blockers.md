# ADR-P025: Perception is a pipeline of typed blockers, and overrides are named permissions

**Status:** Accepted (2026-08-26)
**Date:** 2026-08-26
**Series:** Platform / Engine (`ADR-P###`, per `D-5`)
**Governing rules:** `D-1` (nothing mutates canon directly), `D-2` (the Core owns the substrate, modules
own mechanics), `B-1`, `B-2`, `B-11`, and the founder-locked doctrine in
`docs/superpowers/specs/chunk-5.5-final/FINAL-action-contracts.md` §1.
**Owner of decision:** founder ruling in conversation, 2026-08-26.

---

## Context

Three things forced this, and they are the same problem wearing three hats.

**1. Who perceives an event is hardcoded per event type.** `generate_perceptions` has one arm per event
type, and each arm decides its own perceiver set in SQL. That is why widening perception to *witnesses*
(SPEC-035) required a founder ruling and a migration rather than a configuration change. Every future
question of the form *"should this person have noticed?"* costs the same.

**2. Concealment cannot be expressed at all.** The flagship perception example in the PRDs is
*"You saw Seren pass a sealed note to a cloaked figure."* A **witnessed** handover works as of
SPEC-035. A **concealed** one has nowhere to live: there is no way to say an act was done discreetly,
and nothing to judge whether the discretion worked. Perception today is binary and decided by hand at
the moment the event is written.

**3. Growth is meant to happen by modules (`D-2`).** A future attribute system — stealth versus
perception checks — must be able to join the decision without the engine knowing what it is or how
many others there are. Nothing in the current shape admits a new participant.

## Decision

**Perception is decided by a pipeline. An actor perceives an event unless something stopped them.**

This is the founder-locked action doctrine applied to perception. `FINAL-action-contracts.md` §1:

> *"Deterministic machinery — the gate and the contract arithmetic — exists to **BLOCK
> impossibilities**, never to award success. Nothing gets a free pass. An action 'happens' only because
> nothing stopped it."*

Read *"perceives"* for *"happens"* and the rule carries over unchanged.

### 1. The arm establishes candidates; the chain decides

An event-type arm's only job becomes **who could conceivably have perceived this** — the candidate set.
Everything after is a chain of links that can remove candidates. This splits one hardcoded decision
into a cheap authored part and an extensible part.

### 2. A link may block. A link may not grant

Per candidate, a link returns **`blocked(kind, reason)`** or **`abstain`**. There is no `allow`.

This is what makes the chain safe to extend: since links only ever narrow, the **outcome does not
depend on execution order**, so a link never needs to know whether it is second of two or second of
two hundred. Order is then free to be chosen for **cost** alone.

### 3. Blocks are typed, and the type is what an override addresses

| Kind | Means | Example |
|---|---|---|
| **`physical`** | It was *impossible* to perceive. | Titus was behind Erik. |
| **`attentional`** | It was possible; they were not attending. | Mid-conversation; the act was quick. |

`physical` is deterministic and belongs to **Physics**. `attentional` is a judgement and belongs to the
**resolve seat** (the LLM), which is the best judge of attention in the system.

### 4. Overrides are named permissions, declared at module setup — never numeric ranks

A link declares, at install time, **which block kinds it may override**. Not a priority number.

| Link | May override | May never override |
|---|---|---|
| a "keen senses" attribute module | `attentional` | `physical` |
| the resolve seat (LLM) | `attentional` | `physical` |
| a scrying/divine-sight module | `physical` — **and it must say so out loud** | — |

An alert character therefore catches what a distracted one misses, and still cannot see through Erik.
A module that claims it can beat physical occlusion is making a statement a human can rule on in one
sentence, which is the entire point of naming the permission instead of numbering it.

### 5. A link may be marked *must-run*, for audit

By default the chain may short-circuit. A link configured **must-run** executes even after an earlier
block, so all reasons are collected rather than only the first. This changes no outcome — it exists so
*"why didn't he see it?"* has a complete answer.

Short-circuiting is a **cost** optimisation only, and in practice it has exactly one job: do not spend
an LLM call once a deterministic link has already blocked. Deterministic links are cheap enough to
always run.

### 6. Every block is recorded with its kind, reason and origin

Which link blocked, of what kind, and why. Two reasons this is not optional:

- **A world becomes debuggable.** Today *"why didn't he see it?"* has no answer anywhere.
- **The player can be told something true.** *"You were turned away"* is a story beat; silence is a bug
  that looks like content.

It also makes a module's behaviour auditable — you can see which link is doing work, or none.

## Alternatives rejected

**Numeric decision weights (a low-rank NO loses to a high-rank YES).** Proposed and rejected in the
same conversation, for three reasons:

1. **Ranks are execution order wearing a costume.** The ordering problem moves from *when links run* to
   *who assigns the numbers*, which is a global decision with no owner. A module that declares its own
   rank starts an arms race — every author believes their check is the important one, so every number
   drifts to the maximum. This is how CSS `z-index`, firewall rule ordering and check-priority systems
   all rot.
2. **A high-rank YES is a grant, which `FINAL-action-contracts.md` forbids outright.** Concretely: keen
   senses would let you see through a wall, because a geometric block was overridden by a number. The
   value of block-only is not purity — it is that impossible things stay impossible *for free*. Once
   anything may beat a `physical` block, every future module needs auditing for "does this let someone
   see through a wall?", forever.
3. **Weights make outcomes unexplainable.** *"3 said no, 7 said yes, 5 said no, net no"* cannot be told
   to a player or debugged at 2am. A typed block always yields one sentence.

**Block-only with no overrides at all.** Also rejected, and this was the harness author's initial
position. Too rigid for the genre: exceptional senses and magic are the point of an RPG, and a system
that cannot express them is a worse product. Overrides must exist — they just must be typed.

## Consequences

- **Concealment is unblocked in design and blocked in practice** until Physics exists as a domain.
  Perception must **not** grow its own occlusion logic in the meantime; *"just add a `concealed` flag to
  the event"* is the obvious local fix and it is wrong.
- **There is no sub-place geometry today.** Co-presence is place-level and binary — no position within a
  room, no facing, no furniture. `physical` blocks therefore have almost nothing to reason with yet.
  The two-step design still holds: the LLM's `attentional` judgement carries the weight until Physics
  has something to say.
- **`generate_perceptions` needs restructuring**, not extending: the arms shrink to candidate
  selection, and the blocker chain becomes the extensible surface. This is the migration path, not a
  rewrite done in one round.
- **The perception record gains a companion**: why a candidate did *not* perceive. That is new storage
  and a new read surface.
- **Perception stays permanently core** (`D-2`, `workspace:ADR-W005`). The *links* are pluggable; the
  chain, the block kinds and the override permissions are Core mechanism.
- **`SPEC-016`** (per-attribute perceivability) becomes the same shape as everything else: whether an
  attribute is visible is a blocker question, not a new subsystem.

## Open

**Is `social` a third block kind, or is it candidate selection?** If nobody told Jonas and he was not
present, he was arguably never a candidate — in which case there are only two kinds, which is better.
Unresolved; do not add a third kind without settling it.
