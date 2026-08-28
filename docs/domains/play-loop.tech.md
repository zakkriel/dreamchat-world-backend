# play-loop · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-7 · The play loop ·
**Parent bounded context:** World Engine (output crosses into Compendium & Play UX)

This file holds how the domain is built — the turn, validation, traps. `play-loop.product.md` holds
what it means; `play-loop.seams.md` holds what crosses its boundary.

Line numbers are as of 2026-08-27; re-locate by grep before relying on one.

---

## The turn, file by file

`POST /worlds/{w}/beats` → `core/api/beatsstream.go` (SSE handler; body field is `text`) →
decompose seat → `beatseats.go` `DecodeAndValidateChain` (the belt) → `orchestrator.go` `RunBeat`
(`:135`): world-first cognition → premise → route → adjudicate via the resolve seat → commit through
the doors (`apply_beat` for the fast path — `grep -n 'FUNCTION public.apply_beat' core/db/schema.sql`
— `apply_ruled_event` for ruled outcomes; both doors are canon spine's, see `seams.md`) →
`narration.go` renders behind the wall → frames stream. `D-8` binds the shape: the synchronous path
stays parse → read → route → validate → stream; summarisation, reflection, backstage, images run
async.

Supporting parts: `tension.go` (the beat budget: `tensionBudgetSeconds` maps the tension enum to
seconds, `beatBudgetSeconds` is the orchestrator's once-per-beat read; 1 tick = 1 second),
`pressure.go` (deterministic pressure roll — replay must reproduce it byte-for-byte),
`tier1.go` (`tier1Registry`, the engine-known closed attribute set; grows only by adding a check in
code), `verdict.go` (`jsonTypeOf` shape helpers), `beathandler.go` (the turn; keeps the deliberate
legacy joined `narration` string — documented at the code, not precedent).

## Leash then belt

The schema constrains generation (structured output); Go re-validates anyway, because *"D-13 forbids
assuming the driver honoured its schema"* (`core/api/ruling.go:6`). `beatvocab.go` keeps the closed
vocabulary twice — extracted from the schema at runtime (`vocabularyTypesV2`, `:39`) and as a Go map
(`allowedBeatTypesV2`, `:63`) — with a test asserting they match. `validateAttemptFields` is shared
by three seats (decompose, cognition, world actor): a constraint change binds all three.

**The ruling is mechanical where it can be:** `therefore=impossible ⇔ bounce; succeeds|fails ⇔
resolved(≥1 event)` (`ruling.go:232`). A failure writes canon; an impossibility writes nothing.

## Loop state, not canon

`held_outcome` (`schema.sql`, grep `CREATE TABLE public.held_outcome`) carries a telegraphing NPC's
intended act verbatim until the player's next input; rows may be deleted — no append-only guard, by
design. The wind-up itself IS canon (`orchestrator.go` `commitTelegraph`, `:1048`), written in the
same transaction. On the reaction beat, no cognition fires first: the collision is the world's move,
and the world's turn runs **after** the combined ruling (ordering is doctrine, recorded at the code —
grep `AFTER the ruling` in `orchestrator.go`).

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `D-1` | Proposals only; the doors commit. The leash (`ruling.v*`, `beat_chain.v*`) is this domain's; the door is not. | A seat or module reaching `apply_event` directly is the leak this rule names. |
| `D-8` | The synchronous path stays small. | A "story so far" seat in a handler breaks `D-8` and `B-9` at once. |
| `D-13` | Model-agnostic per-seat routing; never trust the driver's schema compliance. | Deleting the belt because "the schema already checks" re-opens the LLM-to-canon gate. |
| `D-11` | Verdicts are computed at the moment of asking, never stored. | A `has_`/`can_` column is a cached judgment, and cached judgments rot. |
| `ADR-P018` | Seat semantics live at the call site; only provider shaping in the driver. Fakes report a `CapabilitySet` at least as strict as the real driver. | A lax fake shipped three bugs green (failure-log #37). |
| `ADR-036` | The clock advances by committed durations; zero-duration steps increment `beat_seq`. | Hand-advancing ticks breaks the accepted total order. |
| `SPEC-015` | The out-of-vocab belt is continuously enforced (`go-tests.yml`). | An event type added in one place is silently unreachable — the vocabulary lives in four files across three domains (`seams.md`). |
| `ADR-P022` | The latitude block is byte-identical in every prompt. | `go test -run Latitude` goes red; a founder caught the last partial edit (failure-log #5). |

### What you may not decide alone

1. **Adding an event type.** `beat_chain.v2` `items.oneOf` + `beatvocab.go` + `apply_event` (canon
   spine) + a `generate_perceptions` arm (perception). Four files, three domains.
2. **Any new engine computation.** Not in `FINAL-action-contracts.md`'s table → it routes to the
   resolution LLM. Adding it to the table is a founder ruling.
3. **Moving `beat_frame.v5` or `scene_current.v4`.** Cross-repo pin; same-round frontend change
   (`seams.md`).
4. **Reusing `UNRESOLVED` for anything but ≥2-candidate ambiguity.** A reviewer recommended exactly
   that and it was retracted — the schema's `minItems: 2` and `beatseats.go:176` both reject it
   (failure-log #36).
5. **Growing the synchronous path.** `D-8`; say what moved out of async and why.

## Validation for this domain

- `cd core/api && go test ./... -count=1` — `-count=1` always: the suite is seed-dependent and the
  cache shows stale passes.
- `cd core/api && go test -run Latitude` — every prompt file and embedded prompt (`ADR-P022`).
- pgTAP: `95_/96_apply_beat_*`, `98_play_loop_invariants`, `103_gather_slice`, `103_world_pressure`,
  `104_world_slice`, `105_gather_slice_multiactor`, `116_watch_horizon`.
- `../harness/check.sh contract-drift` after any schema move. Never `make reset` (it drops the dev
  volume); the Go suite writes and does not roll back, and it poisons pgTAP that follows.

**What counts as evidence here:** this area has the worst verification record in the workspace
(`docs/areas/play-loop.md` §2). Reproduce first; and revert the fix and watch a test fail before
claiming coverage — 40 of 70 mutation probes once survived a green run (failure-log #15).

**What counts as ceremony here:** the legacy `beat_chain/1` belt, which nothing in production uses,
is tested — while all four live resolve-seat canon guards survived mutation for fifteen days
(failure-log #14; now gated by `ruling_v2_guards_test.go`, 7/7). And `./stack.sh smoke` cannot tell a
real beat from `{"text":""}` (failure-log #22) — never cite the smoke as behavioural evidence.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **Routes are not covered until you delete one and see red.** 7 of 8 registrations were deletable green, including the only write path. | failure-log #13; now `router_coverage_test.go`, bidirectional. |
| **The tested guard and the load-bearing guard can be different guards.** | failure-log #14. |
| **A test can pass 36 times against one stale row.** | failure-log #17 (`orchestrator_test.go`, `TestRunBeatEntityCreatedRoutesAndCommits`). |
| **A fake laxer than the real driver ships bugs green.** | failure-log #37; `bridge_fakes.go` header. |
| **The empty chain is overloaded**: "continue" and "could not understand you" are indistinguishable by construction at `RunBeat` — no press-origin parameter (verified 2026-08-27: `orchestrator.go:135`). | `QA-SPAN-2026-08-11` §1; failure-log #22. |
| **The prompts and the validator disagree on the vocabulary count.** `resolve.txt:4` says *"the only six"* and omits `EntityCreated`; `ruling.go`'s `allowedRuledEventTypes` accepts it (verified 2026-08-27). Recorded, not resolved. | `core/api/prompts/resolve.txt:4`; `core/api/ruling.go:97-110`. |
| **Check the schema and the code before writing a recommendation down.** | failure-log #36 — the retraction is the receipt. |

## Open questions

1. **The empty-chain overload.** The named fix — carry the press kind into `RunBeat` — is a design
   call, not a QA call, and has not landed. Whose ruling?
2. **Does `SPEC-013` still say "deferred"?** `docs/open-spec-items.md` §SPEC-013 defers the
   adjudication engine ("identity/passthrough in the thin slice"), but the resolve seat adjudicates
   with full rulings today (`ruling.go`, `resolveprompt.go`). Both sides recorded; a status update is
   the founder's. Same question for `SPEC-015`'s firing trigger — the Leg-2 bridge it waits on is
   built, and the belt it asks for is enforced, yet the item carries no LANDED mark (`SPEC-012`
   remains genuinely deferred).
3. **The six-vs-seven prompt inconsistency** (trap above) — fix the prompts, or is the omission
   deliberate for those seats?
4. **`participant_ids` v1-compat residue in `ruling.v2.schema.json`** — retired when the v1 decode
   path retires; who owns pulling that thread? (Also carried in
   `perception-and-knowledge.tech.md` §Open questions.)
