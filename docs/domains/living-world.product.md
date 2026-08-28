# living-world · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-12 · The living world ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `living-world.tech.md` holds how it is built; `living-world.seams.md` holds
what crosses its boundary.

---

## What this domain is for

**One job: the world moves without the player causing it.**

Two mechanisms are built. The **pending ledger** carries known futures — consequences already
caused, fired by the advancing clock, zero model calls until their moment. The **World Actor**
supplies spontaneity — intrusions nobody scheduled, drawn by rising pressure on world-time. A third
subsystem, **Backstage Updates**, is fully designed and has no code (see "deliberately not built").

The product reason: *"you're the clock, not the cause."* A world whose every event traces back to
the player is a chat scene with scenery. Pressure rides clock ticks, not player steps — a chatty
player never accelerates the world; real-world absence never moves it (Time's law).

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Pending event** | Pre-caused world truth with a `fire_at_tick`. Fires on the **clock**. Distinct from a held outcome (play-loop's), which fires on the player's next input — different trigger, different row. |
| **Eruption** | An unscheduled intrusion the pressure roll fired. Recorded in the append-only fire-log. **Not contestable** — no held outcome, no reaction beat. |
| **Pressure** | World-time elapsed since a tier last erupted. **Derived, never a stored counter.** There is no forced eruption: the cap means nothing is ever guaranteed, and no hard maximum quiet stretch exists. |
| **Tier / magnitude** | `small · medium · large` — a closed set. Frequency and magnitude are different dials; pools are independent, and a pool drains when it fires. |
| **The world's turn** | The per-slot unit run after each committed clock advance: ledger first, then the roll. |
| **Intrusion** | What the World Actor authors: ONE truth event with a location, sized to the drawn tier — never who perceives it. |
| **Backstage Update** | *"Not memory retrieval… a controlled world-state review process."* Designed only. |
| **Node Decay** | Accumulated **review pressure**, not forgetting. High decay triggers review; a review may legitimately conclude *no meaningful change*. Designed only. |
| **Structural depth** | Active / background pressure / lore-only — a full-product scaling principle, not a PoC shortcut. Designed only. |

The three designed terms live in `digest/S01_the_law_and_the_language.md` Topics 6–7; their source
strategy files were deleted (see `tech.md` §Open questions).

## What this domain is not

- **Not the clock.** Time & the world clock (WE-6) owns ticks, `fn_world_now`, and duration classes.
  This domain consumes crossings; it never re-derives time.
- **Not the beat loop.** The play loop (WE-7) calls the world's turn and owns the halt. This domain
  reports a magnitude; it never ends a beat itself.
- **Not a mind.** NPC cognition (WE-8) fires when an actor perceives something (`B-11`). The World
  Actor is world-scope and fires on the clock — the one seat that is deliberately not a mind.
- **Not genesis.** World genesis (WE-10) makes worlds; this domain makes them move. Today it has
  nothing to move at birth (`seams.md`, the genesis seam).

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `D-8` | Backstage (with summarization, reflection, images, evals) runs **async**; the synchronous path stays small. | Building world-state review inline in the beat re-decides platform shape. |
| `B-11` | Cognition is event-driven: belief updates only on perception, never a free-running idle loop. | Letting the pressure clock drive a *mind* erases the boundary that keeps the World Actor the only clock-driven seat. |
| `B-2` | Knowledge moves through valid in-world paths — the design's own guardrail: *"social propagation must not become omniscience."* | An eruption everyone "just knows about" is assigned knowledge. |
| `SPEC-031` | Eruption frequency is a **felt-experience dial owned by the founder**, tuned by playing (`medium` climb chunk 3600→300, landed). | Retuning the numbers yourself re-takes a founder judgement. |

## What is deliberately not built here

- **Backstage Updates / Node Decay.** Fully specified, zero code. The only trace in the schema is the
  reserved `origin='backstage'` value in `canon_event`'s CHECK — no production writer exists
  (`tech.md` §Designed, not driven). Deferred by PoC sequencing and `D-8`'s async placement; building
  it is starting a subsystem, not filling a gap.
- **Radius 2+ update cascades — never.** Bounded radius is the design: radius 0 the node, radius 1
  direct connections, and *"wider consequences become background pressure, a future hook, or
  unresolved review pressure"* instead of a cascade (S01 T6). A cascade is the named failure, not a
  missing feature.
- **Recurrence.** `pending_event` has no interval and no re-arm; *"a tide that comes twice a day"* is
  inexpressible. Ruled a separate engine program: *"every autonomous mover in this engine is either
  recurring but unsituated (the pressure roll) or situated but one-shot (`pending_event`). Nothing is
  both"* (S07a T13).
- **No appropriateness/mood filter, no tension-damping.** *"Disruption at the worst moment is a
  feature"* — a tense scene never lowers the odds. Adding either dial reverses a founder-approved
  design sentence, not a bug.
- **Off-scene eruptions ("B").** v1 manifests at the current scene. The author-truth-with-a-location
  invariant exists so B can grow in **without a seat or schema change** — do not "prepare" for it.
- **`world_relevance_score`.** Parked: *"the user is not automatically the center of the world"*
  binds as a rule today; the score itself is unbuilt (S01 T9).
