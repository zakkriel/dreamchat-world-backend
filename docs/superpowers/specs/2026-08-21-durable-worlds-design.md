# Durable worlds: a built world is never lost (2026-08-21)

**Status:** Approved design (product decision 2026-08-21), pre-implementation
**Amends:** `2026-08-20-kickstart-arrival-choice-design.md` (reverses its "commit or nothing across the whole journey" posture and its rejection of "commit world first, arrival later"); `10_prds/prd_world_creation.md` AC-2 (transaction scope) and AC-6-adjacent draft expiry.
**Depends on:** the shipped split flow (PR #119): `worldgenesishandler.go`, `genesisdrafts.go`, `worldkickstart.go`, `worldgenesiscommit.go`, contracts `world_genesis_frame/2`, `world_kickstart_turn/1`.

## The problem, observed in production

Two builds on 2026-08-20/21 died the same way: `world_genesis` succeeded (33s, then 63s
of paid authoring), and the `world_kickstart` seat then authored a scenario opening in a
real but unpopulated place. The belt refused — correctly — but the flow's posture turned a
refused *answer* into a lost *world*: the draft lives only in memory with a 15-minute TTL,
so a redeploy, a timeout, or a user who walks away discards an authored world outright.
The 2026-08-20 spec chose that posture deliberately ("expiry or abandonment loses the
build and its spend"). Production experience overrules it: the world is the expensive,
correct part; the kickstart is the cheap, flaky part; the cheap part must not be able to
destroy the expensive part.

Second observed defect, same builds: the kickstart seat must derive "places someone
starts in" by joining `cast[].starts_in` against `places[]` inside a serialized document.
The routed flash model failed that join twice in a row. The legal set is computable
server-side; making a model re-derive it is authoring the failure.

Third, a gap rather than a regression: an identity that references people — "Joe, son of
Dalma and Harry" — silently drops them. The kickstart prompt declares the world immutable,
so kin named at the character question can never exist in the world that premise walks into.

## Decisions (all made 2026-08-21)

1. **The world commits when authoring ends.** Phase 1 (`POST /worlds/genesis`) ends in one
   transaction that writes the whole world EXCEPT the player and the arrival:
   entities, naming, history, minds, opening state, theme, tagline, brief, art style.
   `player_entity_id` stays NULL, so the directory lists it as `playable:false` — a state
   the directory contract already names ("real, listed, and not yet enterable"). The
   stream still ends in the choice frame; the frame now carries the world's real id.
2. **The arrival is its own transaction, retryable forever.** The final kickstart answer
   registers the player entity, writes the arrival event/state/perception, records the
   character, and stamps `player_entity_id` — the same rungs as today's ladder tail, in
   one transaction. A refused scenario, a seat fault, or a crashed process costs exactly
   one answer, never the world.
3. **No TTL, no handle, no in-memory draft.** The kickstart is keyed by `world_id`. The
   authored genesis document persists server-side (a `world` column, never served — the
   AC-7 secrecy boundary moves into the database, it does not weaken). Turn state
   (chosen identity, authored scenarios) persists beside it. `genesisdrafts.go` is
   deleted, not deprecated.
4. **Multi-character ready, single-character now.** A new `world_character` table records
   every character ever committed into a world; `world.player_entity_id` remains the one
   active pointer every existing consumer reads (SPEC-028 unchanged). Allowing a second
   row per world — and moving the pointer, or scoping it per session — is explicitly
   deferred; nothing in this design has to move to allow it later.
5. **Referenced people become real.** The character turn's seat may emit `new_cast`:
   people the chosen identity explicitly references who do not exist in the cast, in the
   genesis cast shape (descriptor, canonical name, standing, speech manner, traits,
   hiding, starts_in). They are committed in the arrival transaction: registered, given
   minds (B-4 still holds — the PLAYER gets none), placed, and made mutually known to the
   player at the arrival tick — you know your own parents' names, and they know yours;
   that knowledge is premise, earned at arrival, satisfying I-9 by construction.
6. **The scenario prompt states the legal places outright.** The kickstart prompt gains a
   section enumerating exactly the places a scenario may open in (cast placements plus
   accepted `new_cast` placements). The belt check stays; it should now never fire.

## What moves in the contracts

- `world_genesis_frame/2` → `world_genesis_frame/3`: the `choice` frame's `handle` is
  replaced by `world_id`. Working/refused/error frames unchanged.
- `world_kickstart_turn/1` → `world_kickstart_turn/2`: the request becomes
  `{world_id, answer}`; `answer` may be empty, which means "show me the pending question"
  — the resume path. Responses keep the turn grammar; `done:true` still carries the world.
- `world_directory/2`: unchanged. `playable:false` plus a non-null brief is what the
  frontend reads as "creation unfinished — offer to continue".
- Vendored frontend contracts, generated types and PINs move in the same commit as each
  version bump, per standing discipline.

## Failure semantics, stated

| Failure | Before | Now |
|---|---|---|
| Kickstart seat refusal | 422, draft re-put, lost on TTL | 422 with the stated reason; world untouched, retry forever |
| Process restart between phases | build lost | world listed `playable:false`; kickstart resumes from the pending question (identity re-asked if it was never committed to turn state) |
| Genesis commit fails | n/a (no commit until the end) | `error` frame — the world honestly could not be built |
| Kickstart on a world that already has its player | 410 via spent handle | 409 — the world is finished; nothing to resume |
| Unknown world id | 410 | 404 |

## The amendment

PRD AC-2 promised "playable or nothing" for the whole journey. What that rule protects is
the half-world: a directory row that 500s when entered. That protection is kept — a world
is `playable:false` until the arrival transaction lands whole, and `playable:true` still
means every rung exists. What the rule cost — a finished, paid authoring destroyed by the
flakiest call in the chain — is what this design removes. The 2026-08-20 spec's rejection
of "commit world first" cited AC-2 and "debris on abandonment"; an unfinished world is
now a *resumable good*, listed honestly as not yet enterable, not debris.

## Acceptance criteria

1. A build whose stream reaches the choice frame has a committed world row with
   `player_entity_id IS NULL`, listed by the directory as `playable:false`, carrying the
   brief and (when chosen) the art style.
2. Killing the process (or losing all memory) after phase 1 loses nothing: a kickstart
   POST with that `world_id` and an empty answer returns the pending question.
3. A kickstart refusal (e.g. unpopulated scenario place) returns 422 with the seat's or
   belt's stated reason; the world row and its content are byte-identical before/after.
4. The final answer commits, in one transaction: player entity, `world_character` row,
   arrival event + state + one direct perception, `new_cast` entities with minds and
   placements, `player_entity_id` stamped. `playable` flips true only then.
5. An identity referencing people not in the cast produces `new_cast` entries for exactly
   those people; after commit the player and each referenced person hold each other's
   names at the arrival tick, and every earlier tick holds no such perception (I-9).
6. The kickstart prompt contains an explicit populated-places list; every scenario the
   fake and live seats emit opens in a listed place.
7. The genesis document appears in no HTTP response body at any phase (AC-7 held with
   the doc now in the database).
8. A second kickstart against a finished world answers 409; an unknown world id answers
   404; both with stated reasons.
9. The full journey — build, resume after a simulated restart, refuse, retry, commit —
   runs green under `DREAMCHAT_BRIDGE=fake` with captured payloads for the new contract
   versions validated in CI.
10. Invariants I-1…I-10 hold on the committed world; the player's epistemic state at
    arrival is exactly one direct perception plus the kin name perceptions of AC-5 when
    kin exist, and nothing else.
