# living-world · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-12 · The living world ·
**Parent bounded context:** World Engine

Each row declares an expectation — one side owns a fact, the other consumes it and must not
re-derive or re-decide it. `living-world.product.md` holds what the domain means;
`living-world.tech.md` holds how it is built.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Time & the world clock** (WE-6) | the crossing `(tickBefore, tickAfter]` and `fn_world_now` | Pressure is a pure function of world-time; `climb_chunk` is ticks, never beats — *"a chatty player never accelerates the world."* This domain never re-derives time; the clock never decides what erupts. |
| consumes | **Canon & the spine** (WE-1) | the commit doors | Everything the world authors — ledger fires and eruptions alike — commits through `applyEvent`/`adjudicate` via `commitWorldPayload`. **No third door.** `postCommitFn` bookkeeping rides the commit transaction (tech.md §The write path). `origin='backstage'` is a reserved value in the spine's CHECK with zero production writers. |
| consumes | **Seats & the LLM bridge** (WE-13) | the `world_actor` seat (`bridge.go:113`, `CapStructuredOutput`) | The bridge owns routing and capability; this domain owns the prompt, schema, payload, and the runtime gate. The dev binding is a fake driver — a live driver is the condition rung 0's deferrals were closed for. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **The play loop** (WE-7) | `firedMag` from `runWorldTurn` | The caller owns the halt: orchestrator applies the §5 cut (`HaltReason="world_eruption"`, `eruptionCutsBeat`); small runs on, medium/large end the beat, eruptions are never contestable. The composer is **called, never modified**; the loop must not re-derive tier order or the cut threshold. On a reaction beat the world's turn runs **AFTER** the combined ruling — doctrine, not convenience (`reaction_worldturn_test.go`; S13a T26). |
| provides | **Space & the journey** (WE-5) | the same composer, once per leg (`journey.go:375`) | Zero changes for the Journey; a medium/large fire ends the whole journey (R5 — the player restates). SPEC-032's lesson lives at this seam: what blocked road interruptions was the Journey's waystation guard, never this domain's gate. |
| provides | **Perception & knowledge** (WE-3) | a committed truth event with a **location** | *"Author truth + engine fans out."* The World Actor and the ledger never encode who perceives — perception is the shared fan-out, and that invariant is what lets off-scene "B" grow without a schema change. Do not synthesize perceptions in `ledger.go`. |
| provides | **NPC cognition** (WE-8) | the boundary, not a call | Minds fire on perception (`B-11`); the World Actor fires on the clock and is world-scope by role — the one omniscient seat, deliberately not a mind. An eruption reaches a mind only as a perceived event. `applyNPCDecisions` passes a nil `postCommitFn` on purpose. |

## The seams that do not exist

- **Genesis → the ledger.** A freshly generated world has an empty `pending_event` and no physics
  attributes — the diorama finding (tech.md §Designed, not driven). The proposal making genesis the
  first production writer is conceded-but-blocked (S07a T13). Until a ruling lands, an agent writing
  ledger rows from genesis is deciding something new.
- **Backstage Updates.** Designed end-to-end (triggers, queue priority, bounded radius, guardrails),
  built nowhere. No seam exists to consume decay or emit reviews; do not improvise one against the
  reserved `origin` value.
- **Off-scene eruptions ("B").** v1 manifests at the current scene; distance/sense fan-out is out of
  scope by name. The growth path is already paid for by author-truth-with-a-location — nothing to
  prepare.
- **Recurrence.** No re-arm, no interval, ruled a separate program. The pressure roll is recurring
  but unsituated; `pending_event` is situated but one-shot. Nothing is both, and closing that gap is
  a founder-scoped design, not a schema tweak.
