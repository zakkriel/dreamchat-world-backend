# projections-and-replay · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-2 · Projections and the read model ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `projections-and-replay.tech.md` holds how it is built;
`projections-and-replay.seams.md` holds what crosses its boundary.

---

## What this domain is for

**One job: the current state of the world, derived from canon and provably re-derivable.**

Canon records what happened; this domain maintains what is *now true because of it* — where each
actor stands, what a door's `open`/`locked` is, what an artifact contains — as materialized
projection tables (`ADR-004`). The play loop needs O(1) state reads; replaying the event log at
read time is unacceptable (`ADR-003`'s rationale). So state is precomputed — and the price of
precomputation is a standing proof that the precomputed copy still equals the log. That proof is
replay (`I-1`), and it is this domain's second half: a projection is **ephemeral** — corrupt or
buggy, it is dropped and rebuilt from accepted events, never repaired by hand.

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Projection** | A read model derived from accepted canon events. Rebuildable, never authoritative, never a write target for anyone but the maintainer (`I-7`). |
| **State mutation** | One attribute change carried by an event: `(entity, attribute_path, new_value)` plus provenance (`event_id`, mandatory — `I-2`). The only input this domain accepts. |
| **The maintainer** | The `NOLOGIN` role that alone may write projections. Everything else gets `SELECT` (`I-7`). |
| **Replay** | Truncate the projections, re-derive them from accepted events in domain order, diff against what was live. Equality is **domain equivalence, not byte identity** (`ADR-026`) — volatile columns are excluded by name. |
| **Domain order** | `(world_id, in_world_tick, beat_seq)`, unique per world for accepted events by index. Transaction time never orders the domain (`ADR-034`). |
| **Ephemeral** | Disposable *because* re-derivable. Perceptions are the counterexample: never regenerated on replay (`ADR-026` boundary — see tech.md §What replay does not reproduce). |

## What this domain is not

- **Not whether something happened.** Canon spine (WE-1) owns events, acceptance, append-only.
  This domain begins after `status='accepted'`.
- **Not who knows it.** Perception (WE-3) is written once at commit and is *not* a projection —
  it does not rebuild on replay. Confusing the two is the costliest mistake available here.
- **Not the pages.** Compendium surfaces render from perception, never from `*_state`
  (`seams.md`). A page field populated from a projection is a `B-1` breach, not a feature.

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-004` | Projections are the read model, maintained by plain triggers first; ephemeral, rebuilt by replay. | Hand-repairing a projection hides the bug replay would have caught. |
| `I-1` | Replay invariance is the permanent correctness check. | A projection that cannot rebuild is corrupt *now*, whether or not anyone noticed. |
| `I-7` | Only the maintainer writes projections. | A second writer makes replay a lie: the rebuild can no longer reproduce what the writer added. Failure-log row 24 is the receipt. |
| `D-11` | Coupled quantities are derived from a recorded generating structure, never stored as independent facts. | An independently-stored copy of derivable state is the copy that drifts. |
| `B-9` | Derived read surfaces are deterministic from stored records — no regeneration drift on reload. | A surface that re-derives differently per read cannot be tested for `I-1`-style honesty. |
| `ADR-003` | PostgreSQL-first; a graph engine arrives post-PoC only as a secondary projection, never the source of truth. | A second transactional store re-opens a settled ruling. |

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No relationship projection.** `apply_mutation`'s relationship branch is a documented NO-OP
  (`SPEC-001`): doc 03 never defined how a single-`entity_id` mutation addresses a
  `(world_id, a_id, b_id)` row. `relationship_state` exists, is diffed by replay, and is
  pgTAP-asserted **empty**. Populating it needs the `SPEC-001` ruling first.
- **No pg_ivm / IVM engine.** `ADR-004`: plain triggers until trigger cost *measurably* hurts.
- **No Redis hot cache, no snapshots, no propagation queue.** `ADR-003`/`ADR-017` design them;
  none exists — no `world_snapshot`, `review_queue`, or `threshold_ledger` table is in the schema,
  and no scheduled job runs anywhere (`SPEC-005` states it plainly). Replay starts from zero,
  and that is currently fine.
- **No nightly replay.** Doc 03 said `I-1` "runs nightly per active world"; there is no scheduler.
  `I-1` actually runs in CI on every push and by hand (`make replay`). Do not describe the nightly
  run as existing.
