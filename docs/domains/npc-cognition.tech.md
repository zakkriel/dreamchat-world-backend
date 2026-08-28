# npc-cognition · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-8 · NPC cognition and minds ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the paths, validation, traps.
`npc-cognition.product.md` holds what it means; `npc-cognition.seams.md` holds what crosses its
boundary.

Line numbers are as of 2026-08-27; `core/api/orchestrator.go` and `core/db/schema.sql` move, so
re-locate by grep before relying on one.

---

## Storage

Created by `core/db/migrations/20260723100003_personality_world.sql` (which also creates WE-12's
ledger shapes — see the .map overlap note):

- **`personality_core`** — `(world_id, actor_id PK, traits jsonb, malleability CHECK (>0 AND <=1))`.
  The header rule: **"WHO THEY ARE IN THE ROOM. No secret ever lives here"** (`schema.sql`, grep
  it) — cores ride shared prompts. The player has no row (`B-4`).
- **`trait_provenance`** — every trait traces to a backstory canon event (`D-11` for character).
  Written only by genesis (`core/api/worldgenesiscommit.go`, grep `writeMinds`).
- **`trait_pool`** — sub-threshold trait accrual. **A socket: zero readers, zero writers** —
  verified 2026-08-27, `grep -rn trait_pool core/api core/db/schema.sql` returns only the DDL.

## The lookups — which seat, mechanically

`core/db/migrations/20260724110003_cognition_lookups.sql` builds three read-only set functions;
*"which seat is a MECHANICAL LOOKUP, not a judgment … pure set operations"* (its header):

- **`fn_isolated_npcs`** — flags the NPCs whose *private* about-ness intersects the action's bound
  ids, one hop through `perception_subject` (`ADR-035`).
- **`fn_public_moment`** — the modal face of events every present holder perceives; deterministic
  tie-break. PRIVATE = fails that test, per `(source_event_id, content)`; a NULL-source record is
  always private (*"isolate MORE when in doubt"*).
- **`fn_private_records`** — one NPC's own records whose subjects intersect the action ids,
  freshest 20 (a labelled v1 dial), presented oldest-first.

The failure asymmetry is stated verbatim above all three: *"a missed flag makes the NPC dull for
one action; an over-flag costs one extra call. The dangerous failure must stay structurally
impossible."*

Subject-triggered retrieval is these same functions: nothing forgotten means nothing *unreachable*,
not everything always present — presence in a mind is triggered by whose ids are in play. An
unlinked record is unreachable by every lookup here, which is why about-ness links are validated at
write (WE-3's side of the seam).

## The call path

`worldFirst` (`core/api/orchestrator.go:797`; grep `func (o *Orchestrator) worldFirst`) runs per
attempt — **one cognition round per action, never per text and never per NPC**:

1. roster = `fn_actors_at(player's location)` minus the player; empty ⇒ skip entirely.
2. split: isolated = `fn_isolated_npcs`; batch = the rest. **≤1 batch call**, one isolated call per
   flagged NPC, uuid-ascending — deterministic order.
3. prompts built by `core/api/cognitionprompt.go` — cache-native layout, section order pinned by
   unit tests: header (`prompts/cognition.txt`) → SCENE → MINDS → *(isolated only)* WHAT ONLY YOU
   KNOW → PUBLIC MOMENT → mutable tail (COMPUTED FACTS → IMMINENT/ATTEMPT → ADDRESSED → DECIDE FOR).
4. output decoded by `DecodeAndValidateNPCDecisions` (`core/api/cognition.go`): each decision
   validated against exactly that call's allowed ids; NPCs act, never ask — `QUERY`/`UNRESOLVED`
   rejected.
5. `applyNPCDecisions` (`orchestrator.go:982`) commits — passthrough types via `apply_event`,
   everything else through `o.adjudicate`, the player's own resolve pipeline. **No bypass.**

The wall holds by construction at step 3: `buildBatchPrompt` is *fed* only the batch minds' cores
and the public moment (`cognitionprompt.go`, grep `WALL INVARIANT`). There is no filter to get
wrong, and *"'was this reaction influenced?' is undetectable after the fact"* — hence no filter.

Invocation sites are exactly two, both event-driven (`B-11`): the beat loop's Stage 1
(`orchestrator.go:251`) and the reaction beat (`orchestrator.go:722`). No ticker, no timer.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-035` | About-ness is an explicit `perception_subject` junction, never derived from event participants. | Both the batch/isolate split and retrieval intersect on it; a derivation drifts them apart. |
| `D-1` | Proposals in, deterministic gate decides; refusal is an ordinary answer. | See the livelock trap below. |
| `D-13` | Per-seat model routing; the quarantine holds per seat regardless of bound model. | The seats are `cognition_batch` / `cognition_isolated` (`core/api/bridge.go`, grep `SeatCognition`); the Go validator re-checks everything the schema already constrained. |

### What you may not decide alone

1. **Widening what a shared prompt carries.** Any new batch-prompt content must be provably public;
   the burden of proof is construction, not review.
2. **Adding a decision kind.** `none | commit | telegraph` is a closed set enforced in
   `cognition.go` and the seat rulebook.
3. **Letting an NPC emit a QUERY** — asking is the player's decompose-only element; rejected by
   design, not omission.
4. **Changing the roster denominator** of the public/private test (see the trap below) — it *is*
   the wall's definition.
5. **Filling the `trait_pool` socket.** Personality evolution is a designed, unbuilt station;
   building it ad hoc re-decides RULINGS-2026-07-23 §8.

## Validation for this domain

- Go: `go test -run 'Cognition|NPCDecision|WorldFirst' -count=1 .` in `core/api` — **`-count=1`
  always**; the suite is seed-dependent and a cached pass is stale.
- pgTAP: `core/db/tests/107_cognition_lookups_test.sql` (the wall functions).
- **`make reset` destroys the dev volume and must never be run** (WE-3's warning; it binds here too).

**What counts as evidence here:** wall failures are invisible in output — a leak reads as a
slightly-too-knowing reaction, undetectable after the fact. Evidence is *construction inspection*
(what was the prompt fed?), never play transcripts. A *dull* NPC is diagnosable from the
`worldFirst:` log lines; a *leaky* one is not — keep failures on the dull side.

**What counts as ceremony here:** asserting the prompt *layout* while feeding it hand-built
structs. `TestBuildBatchPrompt_SectionOrder` passes with `worldFirst`'s data plumbing deleted; the
tests that defend the wall are the ones exercising `fn_isolated_npcs`/`fn_public_moment` against
real rows (pgTAP 107) and `TestCognitionFlow`.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The roster is the NPCs, not the room.** Putting the player in the public-test denominator made the whole seeded history "private", isolated every NPC, and batch never fired — measured live: four isolated calls per beat, ~4–6s the founder felt as slowness. | `orchestrator.go`, grep `THE ROSTER IS THE NPCs` — the comment carries the incident. |
| **An id in the payload is not a fact.** listener_id bound correctly and the wrong NPC answered ("Mara, I want to rest here" — Jonas replied): one batch call decides for every mind, so the loudest wins unless being-addressed is stated in words. | `cognitionprompt.go`, grep `ADDRESSED` (the comment quotes the live symptom); the rule lives in `prompts/cognition.txt`. |
| **Stale invalidated records can flip private to public.** Every lookup excludes `invalid_tick`/`expired_at`; drop that filter and a stale copy inflates the modal vote — the exact leak the wall forbids. | `20260724110003_cognition_lookups.sql`, grep `stale invalidated`. |
| **A name-only mind must never fail the beat.** Seed lags engine; a missing `personality_core` row degrades to name-only, not to error. | `cognitionprompt.go`, grep `NAME-ONLY`. |
| **Seat failure degrades DULL, and used to be silent.** Generate/decode errors skip the minds for one action; they now log — a mute room was undiagnosable before. | `orchestrator.go`, grep `no longer silent`. |
| **Determinism plus a fatal path equals a permanent trap.** A refused proposal must not fail the beat: same inputs re-draw the same decision forever (the World Actor's livelock is the recorded incident). | `core/api/bridge.go`, grep `permanent trap`; `digest/S13a…` §Topic 24. |

## Open questions

1. **The prompt is starved.** The cognition prompt is the one channel that changes an NPC's
   decision, and nothing in it says what any mind is *trying to get* — no pursuit, no opposition,
   no stance. Three review seats enumerated its contents identically and reached the same build
   order: the per-mind situation section first (`digest/S07a_debates_world_creation_and_landing.md`
   §Topic 15, with file:line receipts). Verified against `cognitionprompt.go` 2026-08-27: no such
   section exists. Whoever fills it: scrub through `fn_viewer_text` like every viewer-facing string.
2. **`SPEC-012` text vs the build.** The spec still reads "deferred subsystem … NOT in Chunk 5";
   the decide→act half is live in `worldFirst`. The appraise→believe half (belief updates,
   personality evolution) matches "deferred". Both sides recorded; reconciling the spec text is a
   ruling.
3. **Cognition output shape** — `FINAL-world-npc-cognition.md` records as open *"does it pass
   through decompose or arrive typed?"*; the code today decodes typed attempts directly
   (`cognition.go`). Contradiction recorded, not resolved: the doc's open item may simply postdate
   its answer.
4. **`SPEC-014`** — cascade depth. Reaction depth 1 bounds *acts*; nothing yet bounds inference
   chains (NPC infers → acts → others perceive → infer). Open by its own firing trigger.
5. **`addressed_test.go` / `batchpath_test.go` file ownership.** Both sit on the WE-7/WE-8 seam
   (invocation vs mind behaviour). Not settled here; a re-cut is a founder call.
