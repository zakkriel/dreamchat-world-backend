# perception-and-knowledge · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-3 · The epistemic layer ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `perception-and-knowledge.tech.md` holds how it is built;
`perception-and-knowledge.seams.md` holds what crosses its boundary.

---

## What this domain is for

**One job: who knows what, and how they came to know it.**

Perception is the layer between what is true and every surface a person sees. Canon records that a
thing happened. This domain decides *whether anyone noticed*, *what they would call it*, and *what
they are allowed to be told about it*. Nothing renders from canon; everything renders from here.

The product reason it exists as its own domain: **a world where everyone knows everything is not a
world, it is a database.** Secrets, mistaken beliefs, rumour, and the slow arrival of information are
the substance of the fiction. This domain is where that substance lives, and it is the reason a
DreamChat world cannot be replaced by a chatbot with a good memory.

## Ubiquitous language

Use these words with these meanings. They are not interchangeable, and the codebase enforces some of
the distinctions mechanically.

| Term | Means, precisely |
|---|---|
| **Perception** | One holder's knowledge of one thing. Distinct from its records and its versions — the three are routinely elided and must not be (`digest/01_TOPIC_MAP.md` §WE-3). |
| **Perception record** | One sourced piece of that knowledge: acquired at a tick, from a source event, with an epistemic type. Never edited — invalidated or superseded (`ADR-006`). |
| **Perception version** | Created when a holder's understanding *materially* changes — not per dialogue line. The append-only history stays queryable: "how my understanding changed" is a product feature. |
| **Holder** | The entity whose knowledge it is. A perception always belongs to someone. |
| **Subject** | The entity the perception is *about*. **Load-bearing:** visibility reads subjects, so a perception that names the right holder and the wrong subjects is invisible. This caused SPEC-034. |
| **Epistemic type** | *How* the knowledge was acquired. A closed set, enforced by CHECK (`core/db/migrations/20260610090004_deltas_epistemic_causal.sql:38-40`). Adding one is not a local decision (see `tech.md`). |
| **Synthesis** | A summary derived **deterministically** from stored perception versions. Never regenerated on read (`B-9`) — regeneration drift is a bug, not a refresh. |
| **Projection** | Any derived read of perception: a page, an index, the timeline. Recomputable. |

**Earned name** and **the naming wall** are not this domain's vocabulary — they belong to WE-4 · The
naming wall, and appear only in `perception-and-knowledge.seams.md`.

## What this domain is not

- **Not whether something happened.** That is Canon & Time (WE-1). If you find yourself deciding what
  is true, you are in the wrong domain.
- **Not what an NPC thinks or feels about what it knows.** That is NPC Cognition (WE-8), which
  consumes perception and must never be confused with it. A perception is an acquisition; a belief is
  an appraisal.
- **Not the rendering.** The frontend owns presentation only (`D-7`).
- **Not the naming wall.** WE-4 owns `name_knowledge` and the substitution (see `seams.md`).
- **Not relationship state.** That is Social & Relationships (WE-9). `[INFER]` — a relationship that
  changed because of something an actor never perceived would mean the world reacting to information
  the character does not have, which contradicts `B-2`. So the direction looks forced: perception is
  upstream of relationship. But nothing states it anywhere, and the surface is deliberately absent, so
  this is a domain with a suppressed surface and an unstated contract. It stays marked.

## Product rules — decisions already made

Ids only; the law lives where the id resolves. Cite it, never restate it.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `B-1` | Every user-facing surface renders from the holder's perception; hidden truth is absent from the payload. | A UI-level filter is a `B-1` violation even when it looks correct. |
| `B-2` | Knowledge enters only through valid in-world paths. | Assigning knowledge directly is not a shortcut, it is a different product. |
| `B-4` | Player interiority: the system never authors the player character's inner state. | A surface that tells the player what they believe overwrites the one reading that is theirs. |
| `B-6` | Contradiction lives in perception, never in canon. | "Resolving" a contradiction in canon destroys the fiction's memory. |
| `B-7` | Knowledge transfer never copies memory — a propagated perception is a new record with its own epistemic type. | Copying the row makes hearsay indistinguishable from witness. |
| `B-9` | Syntheses derive deterministically from stored versions. | Regeneration on reload is drift, and it is a bug. |
| `C-4` | Play mode shows the perceived world; only creator/debug may show authoritative state. | A debug surface leaking into play is a `B-1` breach with extra steps. |

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No relationship surface.** There *is* a relationship system; there is *no* relationship UI, for
  two distinct reasons: surfacing it would hand the player knowledge no valid path delivered (`B-1`),
  and it would overwrite the player's own reading with the engine's (`B-4`). *"The divergence between
  the logged relationship and the player's belief about it **is the product**"* (`digest/HOW.md`
  §2.4). What is parked is the surfacing, never the modelling.
- **No per-attribute perceivability.** `SPEC-016`, open. Every new actor attribute needs *"can
  another actor see this?"* and there is no answer yet.
- **No concealment.** `ADR-P025`'s own Consequences: concealment is *"unblocked in design and blocked
  in practice"* until Physics exists as a domain. This domain must not grow its own occlusion logic
  in the meantime.
