# Architecture round — gamedesign

P2 (invariants in the runner) is genuinely structural and I support it. P1 is a claim about *concepts*
sold as a claim about *fields*, and it was traded for a working test. Ranked.

## 1. [HARD VETO] P1 does not cover the bug it retires, and the test that did was deleted

*":81 — `cast[].standing` stops being a bug a test catches and becomes a program that does not build.
This replaces v1's CI leaf-test entirely."*

P1 constrains concepts, not leaves. `Operate` non-empty says the *cast* landing writes engine-read
state — which it always did (`actor_state`, `personality_core`). `standing` was inert because no stage
**parsed** it, and `Parse` reads whatever subset of leaves the landing chooses. A cast landing that
ignores `standing` and returns `actor_state` writes passes P1 carrying the identical bug. So AC-4's
*"as a consequence, not as a task"* (:115-118) is false, and the assertion it replaced was the only
thing that ever caught this.

**Instead:** keep the leaf test as the declaration's fixture — every non-numeric `world_genesis/1` leaf
must be read by some `Parse` and reach some `stateWrite`. This PRD retired a low-level check by
promising a structural property one layer above where the defect lives.

## 2. [HARD VETO] `norms → personality_core.traits` puts law in a channel that erodes it

AC-6 (:136-139) fails twice, both as game design.

**Traits mutate.** The Personality Module composes event magnitude against `malleability` to license
core changes, with sub-threshold experience pooling per trait (`FINAL-world-npc-cognition.md:36-39`).
A law written as a trait entry **decays per mind during play** — the constitution drifting because
someone had a bad week. Nothing marks a trait immutable, and `malleability` cannot distinguish
"guarded" from "only Alphas may speak first".

**Law-as-identity destroys the surface it was buying.** A trait is who a mind *is* —
`{value, manner}`, value from a strength class (`worldgenesiscommit.go:516-525`). Render a norm as a
trait and the mind who breaks the law is out of character: no defiant NPC, no corrupt official, no one
who looks away — which is where a rule becomes a story instead of a rule. A norm also has no
`strength`, so the runner must manufacture the number the seat is forbidden to emit.

**Instead — one line, same channel, no new read path:** land norms as a top-level key in the same
`traits` jsonb, beside `speech_manner`, which is already a top-level *string*, not a trait object
(`worldgenesiscommit.go:514-519`; template `20260813142100_world_templates.sql:201`). Same blob, same
render (`cognitionprompt.go:143-146`), no `value`, and the personality module never touches it because
it is not a trait. Obligation ≠ disposition.

## 3. `arrival` is the escape hatch, and it is concept #8 of 8

AC-3 demands *"no escape hatches … if any of them requires a special case, the contract is wrong"*
(:108-110), with the runner *"inside the existing single transaction"* (:62). Arrival is not in that
transaction: durable-worlds split it into a separate, retryable one — player entity, `world_character`,
the guarded `player_entity_id` stamp, `errWorldAlreadyPlayable` (`worldgenesiscommit.go:134-146`,
`:39`, `:201`) — and it mints `newCast` late, grounding their minds in the arrival event (`:141-146`).
Cross-transaction execution, a non-projection `world`-row write, a conditional guarded update, and
injecting items into another concept's set: four special cases in the one concept you cannot skip.

## 4. `Ground(item) eventSpec` contradicts AC-3's byte-identical migration

Places, ways, objects and every cast placement share **one** scene-genesis event — summary *"the world
as the visitor found it"*, tick 40, `scene_id` = arrival place (`worldgenesiscommit.go:577-585`) — with
order carried by `beat_seq`. A per-item `eventSpec` mints one event per item unless the runner silently
coalesces equal specs; coalescing forbids per-concept text, not coalescing changes canon and fails the
byte-identical diff AC-3 makes the gate. Worse, `cast`'s grounding event is *selected from another
landing's output* — the first authored moment they took part in, else the opening moment (`:462-463`).
The signature cannot say "ground me on `history`'s event for item Y", so that dependency moves into the
runner as knowledge about named concepts: the accretion this PRD exists to end.

## 5. "Operate required non-empty" is validation in costume, as specified

AC-2 wants *"a startup-time failure with a named concept"* (:105-107). At startup there are no items;
`Operate(item) []stateWrite` can only be shown empty by calling it. So the runner either invokes it
with a synthetic item — a runtime check with a fixture, the thing being disowned — or checks a
*declared* target list, letting a landing declare a target and return nothing for real items.
**Instead:** make it type-level — `Operate(item) (stateWrite, []stateWrite)`. One mandatory write,
unrepresentable when absent, no startup ceremony.

## 6. What gets harder: cross-concept tuning, which is where the game lives

The bespoke stages made one thing easy that the contract has no home for: tuning a *felt* property
emerging from several concepts at once. `genesisPlaceCoords` rings rooms at 0.6 of the region radius
**specifically so leaving a room can exceed a beat budget and become a journey** — the SPEC-030 lesson
(`worldgenesiscommit.go:884-892`). That number is a joint function of geometry, `tension`→budget
(`tension.go:28-45`) and speed. Under P2 the runner owns every class→number resolution, so the knobs
deciding whether a world feels big, tense or slow are centralised behind "the runner resolves classes",
owned by no landing and reviewed by nobody with the game in view. AC-3 then freezes today's numbers as
correctness, so the migration cannot fix known-bad tuning and any rebalance surfaces as a diff failure
rather than a design decision.

Answer to (3): the runner is a fine invariant owner. The god object to watch is
`Refuse(item, worldView)`'s `worldView` — every cross-concept game rule ("somebody is already in the
arrival room", `worldgenesis.go:386-395`) migrates into it, and a `worldView` rich enough for eight
concepts is the old accretion under a new name.
