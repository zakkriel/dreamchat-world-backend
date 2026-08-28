# time-and-clock · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-6 · Time and the world clock ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `time-and-clock.tech.md` holds how it is built; `time-and-clock.seams.md`
holds what crosses its boundary.

---

## What this domain is for

**One job: what time it is inside the fiction, and how it moves.**

*"Time does not pass because the user was offline. It passes when it passes inside the fiction"*
(`digest/S01_the_law_and_the_language.md` §Topic 4 — one of the seven product pillars). This domain
owns the world clock: the logical tick, the authored label the player actually sees, the ordering of
events within a moment, and the rule for when any of it advances. The product reason it is its own
domain: DreamChat is genre-agnostic — voyages, eras, dream-time, time loops — and a real-world clock
smuggled anywhere into the model silently imposes a Gregorian calendar on every world (`ADR-030`'s
rationale, verbatim in `docs/law/02_world_state_adrs.md`).

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **In-world tick** | `in_world_tick` — logical, monotonic, *"the ONLY thing compared for sequence"* (`ADR-030`). 1 tick = 1 second (binding; stated in `core/api/tension.go:10` and migration `20260729100002`). Never rendered to a player. |
| **In-world label** | `in_world_label` — the fiction-facing time ("Day 1"). **Authored, never derived**: the tick is truth, the label is presentation, and manufacturing a label from the tick is a `B-5` violation with a receipt (see `tech.md` §Traps). Distinct from entity display names — those are WE-4's wall, a different label entirely. |
| **beat_seq** | Intra-tick order. Events in one moment share the tick and are ordered by `beat_seq`; the pair `(in_world_tick, beat_seq)` is the domain's total order (`SPEC-002` → `ADR-034`). |
| **Three time axes** | Valid time (tick+label) · transaction time (`recorded_at`) · epistemic time (`acquired_tick`, WE-3's). Never conflated (`ADR-006`, `B-5`). Wall-clock is operational telemetry — never world logic, never rendered. |
| **Duration** | Seconds an event costs, assigned by the **engine** from recorded world data, never by the LLM (`ADR-036`). |
| **Tension** | A scene attribute mapped to the beat's time budget (frantic 5 s … calm 600 s, `none` = ∞). *"Values are data, retunable. The shape is the rule"* (`digest/S06_specs_and_final_doctrine.md` §Topic 10). The budget is a blocker, never an award. |

## What this domain is not

- **Not the physics of how long a move takes.** Distance and speed are Space & Journey's (WE-5);
  durations derive from geometry (`D-11`). This domain consumes the number.
- **Not the beat pipeline.** Decompose/resolve/commit is the play loop (WE-7). This domain owns what
  the committed steps cost and the rule for advancing the clock by them.
- **Not pressure or eruptions.** Living World (WE-12) consumes the clock; it never re-derives time.
- **Not epistemic time.** `acquired_tick`/`valid_tick` semantics belong to perception (WE-3).

## Product rules — decisions already made

Ids only; the law lives where the id resolves.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `B-5` | Append-only time: tick + label only, no mutable `updated_at` domain fields, three axes never conflated, wall-clock never rendered or used for world logic. | A mutable time field silently loses history — replay, audit, correction, and the product promise all break. |
| `ADR-030` | Fictional time is logical (tick + label), never `TIMESTAMPTZ`. | A timestamp clock imposes Gregorian linear time on every world; genre-agnosticism dies in the DDL. |
| `ADR-006` | Three time axes; invalidation, never deletion. | "What did this character believe at that moment" becomes unanswerable. |
| `ADR-036` | The clock advances by the durations of events that actually commit — not per beat, not by wall-clock. Interrupted chain → committed prefix only. | Wall-clock breaks replay (I-1); per-beat advancement breaks `C-6`. |
| `C-6` | Continue advances the current moment; it does not fast-forward the world. | Time-skips become a free action; see also the open question on the stillness floor (`tech.md` §Open questions). |

## What is deliberately not built here

- **No LLM time parsing, no `time_advance` event.** `ADR-021`/doc 10 designed extraction-proposed
  temporal interpretation; Phase 0 scope said *"No LLM time parsing in Phase 0"* and none exists —
  `grep -rn time_advance` across `core/` returns nothing (verified 2026-08-27). `ADR-036` supplies
  advancement instead: committed durations, engine-assigned. Building a time-parser is reopening
  `ADR-036`, not filling a gap.
- **No tick→timestamp mapping.** `ADR-030` permits worlds on real-world time to map a tick to a
  timestamp *in config*; *"the core never depends on it."* Nothing does; keep it that way.
- **Mid-skip interruption under `none` does not exist.** Founder ruling: `none` is the deliberate
  atomic time-skip — scheduled events inside the skipped span fire *after the fact* as the clock
  crosses them; *"nothing interrupts mid-journey until journey-split exists"* (`digest/S06` §Topic 10,
  stated so nobody "fixes" it).
- **No in-game label authoring.** Only the seed authors a label ("Backstory", "Day 1"); every commit
  path leaves it NULL and the carry trigger propagates continuity, never invention (migration
  `20260808100002`). `[INFER]` the absence of any "Day 2" author looks like a gap, not a decision —
  no document states a reason. Raised in `tech.md` §Open questions; do not improvise an author.
