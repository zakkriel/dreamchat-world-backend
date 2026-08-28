# play-loop · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-7 · The play loop ·
**Parent bounded context:** World Engine (output crosses into Compendium & Play UX)

Each row declares an expectation — one side owns a fact, the other consumes it and must not
re-derive or re-decide it. `play-loop.product.md` holds what the domain means; `play-loop.tech.md`
holds how it is built.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Canon spine** (the commit doors) | `apply_beat` / `apply_ruled_event` | We propose and decide; the door records (`D-1`). *"A leash belongs to the seat it leashes, not to the gate that consumes its output"* (`docs/areas/play-loop.md`) — `ruling.v*`/`beat_chain.v*` are ours, the doors are not. Play loop never inserts into `canon_event`. |
| consumes | **The naming wall** (WE-4) | the wall belt — `loadNamingWall` (`beatsstream.go:435`), `namingWallError` (`narration.go:156`) | Seat output is walled before it reaches a player; a leak that reaches a player is a wall failure, not a model failure. Play loop never performs substitution itself. **Ownership wording is contested:** `perception-and-knowledge.seams.md` carries `fn_unearned_names` as WE-3's own provides-to-WE-7 row, while WE-4's assignment names the wall as owner of the predicate. Recorded, not resolved. |
| consumes | **Perception** (WE-3) | viewer-scoped payloads — the narrator and cognition read perception, never raw canon (`B-1`); the referee alone is truth-sided | Fact sheets are viewer-scoped per seat; do not hand the narrator a truth-side sheet. |
| consumes | **Space & Journey** (WE-5) | distance, reachability, move duration (`fn_move_duration*`, `fn_actor_move_permitted`); over-budget → a Journey | We ask, Space answers; the decomposer never carries `duration_class` on `ActorMoved` — *"its length is physics."* `premiseHolds` (`orchestrator.go:278`) is a deliberate mirror of `fn_actor_move_permitted` — keep the mirror, never fork it (wording agreed with WE-5's writer 2026-08-27). The `journey` table sits on this seam; `116_watch_horizon` is ruled to space-and-movement (`harness2/DOMAINS.map` header, 2026-08-27). |
| consumes | **Time & clock** (WE-6) | the tick, `ADR-036` advancement, tension→seconds mapping | 1 tick = 1 second is binding (`tension.go:9-11`). `tension.go`/`pressure.go` are contested files with WE-6/WE-12 — the budget is spent here, defined there. |
| consumes | **NPC cognition** (WE-8) | `npc_attempts/1` decisions: none, commit, or telegraph | Cognition proposes; every NPC decision runs through the same pipeline as the player's — no bypass (`applyNPCDecisions`, `orchestrator.go:976`). Cognition never touches the doors. Wording agreed with WE-8's writer 2026-08-27. |
| consumes | **Seats & bridge** (WE-13) | per-seat routing, capability floor, drivers, fakes | Capability is the driver's *report*, never a config label; a fake is at least as strict as the real driver (`ADR-P018`). Play loop never binds a provider itself. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Play surface** (`dream-weaver-visuals`) | the beat-frame and scene-current payloads over SSE (versions: the frontend `const PIN` block, `dream-weaver-visuals/src/api/index.ts`) | Both are vendored and pinned by exact string equality — a change is a cross-repo round, same round (`../harness/check.sh contract-drift`). A new frame kind needs its consumer in the same round: 3 of 7 kinds were once silently dropped downstream (`docs/00_workspace/evidence/QA-2026-08-11-visuals.md` F-10). |
| provides | **Compendium surfaces** (UX-1) | the transcript write — `beatsstream.go:633` calls `persistTranscript` post-belt, `context.WithoutCancel`, never fails the beat | Play loop decides WHEN a row is written and hands post-belt messages; UX-1 owns the record's shape, the read route, and the never-recomputed rule. Wording agreed with UX-1's writer 2026-08-27. |
| provides | **Living world** (WE-12) | the world's-turn hook after the ruling; the deterministic pressure roll (`pressure.go`) | The world's turn runs AFTER the combined ruling on a reaction beat — the ordering is doctrine, not convenience (`tech.md` §Loop state). `worldturn.go` is living-world's (`harness2/DOMAINS.map` header, 2026-08-27); not claimed here. |
| provides | **Canon spine** | validated ruled events (`DecodeAndValidateRulingV2`) | Only what survives the belt reaches a door. The belt is load-bearing: its guards were dead for fifteen days once (failure-log #14). |

## The seams that do not exist

- **No press-origin seam.** `RunBeat` cannot distinguish "continue" from "could not understand you" —
  the empty chain is overloaded and the fix is a signature change nobody has ruled on
  (`tech.md` §Open questions 1). An agent needing the distinction is deciding something new.
- **No module seam.** A module may propose, never write (`D-1`, `workspace:ADR-W005`) — but module
  architecture is B3 and does not exist. Do not improvise a plugin API here.
- **No autonomous cognition engine.** `SPEC-012` is deferred; NPC minds act only inside the beat's
  one-call-per-action flow (`B-11`). A scheduler feeding NPC turns outside a beat has no seam to
  plug into, on purpose.
