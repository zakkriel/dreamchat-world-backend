# projections-and-replay · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-2 · Projections and the read model ·
**Parent bounded context:** World Engine

A seam belongs to two domains, so it gets its own file. Each row declares an expectation — one
side owns a fact, the other consumes it and must not re-derive or re-decide it.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Canon spine** (WE-1) | `state_mutation` rows, written by the doors inside the commit | Provenance is mandatory (`I-2`: `event_id` NOT NULL FK) and events are inserted `accepted` before their mutations, which is the only reason the trigger's acceptance gap (`SPEC-003`, tech.md §The write path) is unreachable. This domain never decides what happened; it applies what the spine committed, in domain order. The spine owns the order key and its uniqueness index (`ADR-034`, `uq_ce_accepted_order`). |
| consumes | **World genesis** (WE-10) | seeded worlds, as mutation rows | Templates write `state_mutation`, never `*_state` (`fn_instantiate_drowned_lantern`, each `(entity, attribute_path)` written once → replay-order-independent). A seed that writes a projection directly is unreplayable from birth. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Play loop** (WE-7) / **Space & journey** (WE-5) / **Living world** / **NPC cognition** (WE-8) | current state, by SELECT (`app_reader`) | O(1) reads of `attrs` (`location_id`, `open`/`locked`/`connects`, tension, statuses). Consumers read, never write, and never cache-and-mutate: the projection is the cache. Tier-1 key vocabulary (`core/api/tier1.go`) is the doors' decision, not this domain's. |
| provides | **Art & image seam** | authoritative `*_state` for generation prompts | Deliberate exception to perception-bound rendering, documented at `core/api/imagehandler.go` (WE-3's seams file owns the full row — one home; this side only guarantees the read is a plain SELECT). |
| provides | **Compendium surfaces** (UX-1) | **nothing, and that is the seam** | Pages render from perception only; the live bodies of `fn_actor_page`/`fn_location_page`/`fn_artifact_page` read no `*_state` table (verifiable by grep of each definition), and the written prohibition is SPEC-029's: *"do not populate them from `*_state` — that is the wall"* (`B-1`). One exception-shaped fact: `fn_carrying` (`schema.sql:1105`) reads the `state_mutation` **log** (newest `attrs.contained_by` per entity, domain-ordered) — the domain's input stream, not its tables, and still never canon. |
| provides | **Canon spine** (WE-1) | the replay verdict | `replay_0A()` is the proof the spine's log is sufficient to reconstruct state. When replay fails, the defect is a second writer or a non-domain ordering — not a reason to patch the projection (`ADR-004`: drop and rebuild). |

Nothing outside the engine reads the canon table (`ADR-005`) — surfaces reach world state only
through perception (WE-3) or, engine-internally, through these projections.

## The seams that do not exist

- **A perception rebuild path.** Replay reproduces `*_state` and nothing else (tech.md §Replay).
  There is no mechanism that regenerates `perception_record` from the log, on purpose
  (`ADR-026`); perception repair is explicit migration work, WE-3's to decide. An agent wiring
  `generate_perceptions` into `replay_0A` is re-deciding a settled boundary.
- **R2+ propagation.** `ADR-017` designs a queue, budgets, and a depth cap; no `review_queue`
  table, no backstage worker, and no writer exists. A downstream-consequence feature has no seam
  to plug into yet — building one starts at the Traversal Matrix question (tech.md §Open
  questions 1), not at code.
- **The outbound change stream.** `ADR-003`'s consequences say to design it now for the future
  graph projection; nothing is designed. `[INFER]` — any external consumer of projection changes
  would need it, so the first such feature must file the design rather than tail the trigger.
- **A relationship consumer.** `relationship_state` is read by `gather_slice` and diffed by
  replay, written by nothing (`SPEC-001`). Social & relationships (WE-9) has no projection to
  consume until that ruling lands.
