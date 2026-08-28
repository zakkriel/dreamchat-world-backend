# space-and-movement · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-5 · Space, movement and the journey ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `space-and-movement.tech.md` holds how it is built;
`space-and-movement.seams.md` holds what crosses its boundary.

---

## What this domain is for

**One job: where things are, and how long it takes to get there.**

Geometry (positions, distance), movement cost (duration from speed), accessibility (portals), and
the Journey (a move too long for one beat). The product reason it is deterministic: a wrong number
is data, not prose. This path is deliberately LLM-free (`D-1`); the engine computes the facts and
the seats reason *from* them, never instead of them.

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Coordinate** | A thing's `{x,y}` position **within its parent's frame**. A location's own coordinate is its position among its siblings — never a landing spot inside it (`tech.md`, traps). |
| **Frame** | The parent whose local coordinates a measurement is expressed in. Distance is computed at the nearest common parent's frame — one formula, every scale. |
| **Area** | An ordered outline of ≥3 points making a place a containable region. Optional: a place with no area is a point and contains nobody. |
| **Portal** | An artifact whose `connects` names two places. **Accessibility, not geometry** — it gates *whether* you can cross; no distance, no positions (`FINAL-action-contracts.md` §5.3). |
| **Movement type** | LLM-minted vocabulary row (`walk`, or whatever a scene invents) carrying a base speed. Only `walk` is seeded. |
| **Modifier** | A percentage on specific movement types, attached to a status. Stacks multiplicatively; floor −100%, **no upper cap**. |
| **Journey** | An attempt whose span exceeds the beat's budget: not a rejection — a sequence of legs the world may interrupt. **Loop state, not canon** (`core/api/journey.go` header). Kinds: travel, wait, watch. |
| **Leg** | One journey slice, advanced by one continue press. Atomic — there is no cancel path. |
| **Stage** | The known place containing the traveller mid-journey; NULL is the open road. |
| **Encumbered** | The seeded status written eagerly when carried weight exceeds capacity: −100% on all movement, so no move fits any budget. The world responding, not the system refusing. |

## What this domain is not

- **Not whether a move is allowed narratively.** We answer *can*; contested outcomes are the play
  loop's (WE-7).
- **Not the clock.** Ticks, labels and budgets are Time's (WE-6).
- **Not who sees the movement.** Perception's (WE-3). We provide co-presence; they decide knowledge.
- **Not whether it is recorded.** The doors commit (canon spine); we compute what a move writes.
- **Not the drawing.** Frontend owns presentation (`D-7`); geometry never leaves the engine at the
  seat boundary either (`place_author/1`, `tech.md`).

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `D-12` | Spatial is a bounded subsystem owning geometry only; distance (how far) is distinct from reachability (whether you can go). | Conflating them is the classic error here — adjacent rooms can be unreachable, far ones one portal away. |
| `FINAL-action-contracts.md` §3 | **The accepted imprecision is law.** Cross-level distance ignores the walk-to-the-door part; fixing it needs exit-point inference, which is reasoning, which is an LLM. | *"An agent who 'improves' it with exit-point inference is breaking the design"* (§3; repeated at §5.3 for portals). |
| `FINAL-action-contracts.md` §4 | Volume blocks; weight never blocks — it consequences. The grab happens; the next move hits speed 0. | "REJECT: too heavy" is the system refusing where the world should respond. |
| `SPEC-030` | Movement is expressible through the beat API: portals and their far-side rooms are candidates. A candidate is a thing you may *name*, never a thing you may *do*. | Hiding the shut door makes the world's refusal unreachable. |
| `C-6` | Continue advances the current moment. One press, one leg — never a fast-forward to arrival. | Auto-resolving legs deletes the world's chances to interrupt, which is the Journey's whole point. |
| `SPEC-031` | Interruption frequency is a founder dial in data (`world_actor_config`), not code. | "Interruption looks broken" is usually tick arithmetic; check the dial before designing a mechanism. |

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No graded encumbrance.** *"Encumbrance (weight slowing your move) is explicitly NOT built…
  Known simplification; revisit only when ruled"* (`FINAL-action-contracts.md` §4). Below capacity,
  full speed; above it, none.
- **No throwing contract.** It breaks the in-reach precondition and *"routes to the resolution LLM
  until a contract is ruled"* (§4).
- **No sub-place co-presence.** `fn_actors_at` is place-level and binary; whether an actor can be
  positioned *within* a place for perception purposes is an open question (`tech.md` §Open), not a
  gap to fill.
- **No frame transforms.** Resolving *through* frames belongs to the deferred spatial engine
  (`SPEC-018`); callers pass the frame they travel in (stated in
  `core/db/migrations/20260807100001_place_area.sql`).
- **No time-of-day waits.** No world calendar — `in_world_label` is free text, so no tick means
  "2am" (journey ruling R9, carried in `digest/S06` §Topic 19; the design doc is consolidated out
  of this repo).
- **No NPC journeys.** Only the beat path starts one: `startJourney`'s callers are the player-input
  chain in `core/api/orchestrator.go`. The row is actor-keyed, so nothing forbids them later — but
  building them now is deciding something new.
