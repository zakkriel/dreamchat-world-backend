# time-and-clock · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-6 · Time and the world clock ·
**Parent bounded context:** World Engine

A seam belongs to two domains, so each row declares an expectation: one side owns a fact, the other
consumes it and must not re-derive or re-decide it. `time-and-clock.product.md` holds the language;
`time-and-clock.tech.md` holds the code.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Space & Journey** (WE-5) | move durations — `fn_move_duration_actor` (and `apply_beat`'s legacy location-pair form) | Durations derive from geometry and speed, which are Space's (`D-11`, `D-12`). The clock never computes a distance; Space never advances a tick. The journey's `current_tick` is loop state Space/WE-7 write; this domain only *reads* it inside `fn_world_now`. |
| consumes | **The play loop** (WE-7) | committed events, through `apply_beat` and `runChain` Stage 4 | Only committed events (and journey legs) move time (`ADR-036`). The loop must not advance time anywhere else, must not let a seat invent a duration, and must not re-add a flat tick cap — the tension budget is the only beat limit (`tech.md` §decisions). |
| consumes | **Living World** (WE-12) | `duration_class_seconds` config (built under its Task 2); tension writes | The LLM stamps tension at scene mint and already-open seats review it (founder-locked flow, `digest/S06` §Topic 10 — *"no seat watches mood freely"*). This domain validates enum membership only — *"mood is not arithmetic"* — and owns the tension→seconds mapping (`core/api/tension.go`). |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **The play loop** (WE-7) | the beat budget (`beatBudgetSeconds`) | Read once at beat start from *current* tension, consumed cumulatively across the chain. The budget is a **blocker, never an award** — *"do not read 'fits' as 'happens'"*. Over-budget moves become Journeys, not rejects. |
| provides | **Living World** (WE-12) | `fn_world_now` and each beat's clock crossings `(tickBefore, tickAfter]` | Pressure is a pure function of world-time, never beat count — *"a chatty player never accelerates the world."* WE-12 consumes the clock and never re-derives it. The stillness floor exists for this seam: a crossing with no world's turn silently skips due events (`tech.md` §Traps). |
| provides | **Canon spine** | `(in_world_tick, beat_seq)` semantics, `uq_ce_accepted_order`, the label-carry trigger | The spine owns the `canon_event` table and the commit doors; this domain owns what the time columns *mean*, the total-order guarantee, and the fill-on-insert label rule. The doors must not set labels ad hoc or reuse an accepted `(tick, seq)` slot. |
| provides | **Projections & replay** | the replay ordering key | Replay orders by `(in_world_tick, beat_seq)` only — `recorded_at` is telemetry and never a tiebreaker (`SPEC-002` → `ADR-034`). A projection re-deriving order any other way is re-deciding a settled question. |
| provides | **Compendium & play surfaces** (UX) | `in_world_label` and `fn_world_now` | Surfaces render the **label**, never the tick (`B-5`; the "[Tick 51]" incident, `tech.md` §Traps). The label reaches a viewer through their own perceived lines (`core/api/scenehandler.go:209`) — WE-3's wall applies even to the clock's display. Staleness ("last known…") derives from `fn_world_now` (`schema.sql:1336`); decay drives language, never visibility. |

## The seams that do not exist

- **Timer-driven anything.** Cognition fires when an actor perceives something, *never on a timer or
  a tick* (`B-11`). There is no cron, no idle loop; a scheduled `pending_event` fires only when a
  committed crossing reaches it. An agent adding a background ticker is building a second clock.
- **Tick→wall-clock mapping.** Permitted by `ADR-030` as world config; unbuilt, and the core may
  never depend on it.
- **Journey-split / mid-skip interruption.** Under `none` the time-skip is atomic (founder ruling,
  `digest/S06` §Topic 10). Events inside the span fire after the fact. Deliberate; do not fix.
- **A label-authoring seam.** Nothing after the seed writes `in_world_label` (`product.md` §not
  built). An agent hitting this is deciding something new and must say so.
