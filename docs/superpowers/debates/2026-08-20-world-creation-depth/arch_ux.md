# Architecture round — ux

Seat: creation UX. Target: PRD v3, not killed v2. I will not re-litigate Operate-non-empty. Ranked.

## 1. HARD VETO — `traits.norms` cannot mean "everyone"

AC-6: the law appears in every bound mind's prompt; empty `binds[]` means everyone. The player is not a mind. `writeMinds`: **"THE PLAYER GETS NO CORE. Not an empty one… no row"** (`worldgenesiscommit.go:458-463`). Arrival writes location, descriptor, coordinates, max_load — never `personality_core` (`:725-788`). Cognition renders a name-only mind as **"no personality core yet"** (`cognitionprompt.go:147-149`).

A taboo that "binds everyone" is written onto every *cast* core and **not** onto the person the user plays. Playback (AC-12) will show a strikeable "this binds everyone"; the engine binds everyone-but-you.

**Instead:** empty `binds` = every *authored* mind (cast + late `newCast`). If a brief binds the visitor, that is an arrival perception, not a core key on a row that must not exist. Do not invent a player core — B-4 is the product.

The sibling-key move (not `{value, manner}`) is sound. Malleability will not eat it. Magnitude is not invented. The remaining defect is **who has a blob**, not the key's shape.

## 2. R2 collides with UI-only leaves — hatch: `arrival.why` (and `places[].kind`)

Q1 recast: v3 patches the eight-concept lie. `world` is the named non-landing (AC-4). `arrival` is a phase (R6), matching a second retryable transaction that rebuilds ids (`:134-256`) and stamps `player_entity_id` last (`:208-220`). `history` can declare `perception + referenced(cast)` with no dummy state. `shares("scene_genesis")` matches one event plus `beat_seq` (`:570-604`).

The remaining hatch is **leaves that exist so a human can choose, not so the engine can run**. `arrival.why` is copied onto the chosen identity (`worldgenesishandler.go:529-535`) and **never written** by `commitArrival`. It is button copy. `places[].kind` is refused when blank (`worldgenesis.go:282-283`) and dropped, while `objects[].kind` is stored (`:677`).

R2 diffs every non-numeric genesis leaf against `⋃ Consumes`. Unclaimed `why` is a registration failure. Three bad exits: delete the copy and the choice frame goes mute; persist a marketing sentence as canon; or a secret `ui-only` reader — the escape hatch AC-3 forbade, moved from concepts to leaves.

**This is structural.** The contract's unit is "authored concept." The product's unit is "what the user saw and tapped." Those sets are not equal. Add Reader `surface(path)`: satisfies R2, writes nothing, legal home for choice-frame copy. Without it, R2 re-creates `standing`: a required string whose only honest landing is delete.

## 3. R1 is structural; Operate-non-empty was a costume (Q2)

**Was** validation in a costume; v3's replacement is not. A non-empty `[]stateWrite` cannot be checked at registration (no items). A non-empty `Readers` sum type can. `history` passes honestly. Leaf coverage is where `standing` actually lived (schema-required, `validate` `:310-311`, zero commit writes).

Caveat, not a veto: `referenced(concept)` proves another landing *cites* you, not that a player felt you. History should pass. Depth must not hide behind R1. AC-9 (diffed NPC decisions) and AC-12 (strikeable statements before spend) remain the only victory conditions this seat signs.

## 4. Runner invariants: not a provenance SPOF; a *copy* SPOF (Q3)

R3 is right for ids, ticks, class→number, `source_event_id`, I-9. The archivist bug (`:550-555`) should be unrepresentable (R4). Hidden coupling is elsewhere:

- **`Refuse(item, resolver)`.** "Nobody is in the arrival room" is a *document* rule (`worldgenesis.go:384-395`) — the opening beat the surface promises. Park it in a fat resolver and the named refuse becomes a generic Apply toast. Bound the resolver or ~40 `refuse()` causes collapse.
- **Naming-wall as afterthought.** AC-6 walls because `sb.Write(m.Traits)` dumps JSON raw (`cognitionprompt.go:143-146`). That pass is not in `Declaration`. N+1 stuffing a sentence into jsonb will ship the archivist bug in a new blob. Put the wall on Reader `state(jsonb-that-renders-raw)`.
- **Confirmation is not a Phase.** AC-12 is the only correction surface. The contract has `content | arrival`. Playback is a third step, before genesis spend. Add required `Playback() []Statement` mapping `Consumes` → world-language, or admit N+1 is two files.

## 5. What this makes HARDER (Q5)

1. **A refuse the user can use.** Per-field `refuse("%q has no standing")` vs one Apply error. Keep `Refuse` on the same named-cause type `validate()` uses.
2. **Choice-frame copy that must die at the door.** `why` is good product, bad leaf. See §2.
3. **Cross-concept opening promises.** Occupied arrival room, way out (`:357-381`), player not in cast (`:337-344`). Keep these as arrival phase checks, not landing internals.
4. **Late `newCast`.** Arrival mints people in another process, grounds them on the arrival event, writes mutual **names** off the naming `world_genesis` event at the *arrival tick* (`:175-197`). R4 must say: name perceptions may cite `world_genesis`; secrets and norms may not. Hide that only in the runner and the next author grounds a law sentence there.
5. **Partial recovery.** Interview `{done:true}` collapse (`worldinterview.go:71-84`) is still expelled (§10). Composing landings does not compose seat honesty.

## Verdict

The abstraction is **not premature** for commit. Declaration + Apply, R1–R6, and `world` as the one named non-landing match the code. Ship the runner.

Do not ship AC-6's "everyone" until it names the visitor-shaped hole. Do not ship R2 until `surface(path)` (or deletion of `why` / `places[].kind`) exists. Add `Playback()` or stop claiming one file. Bound `Refuse` causes. Put the naming wall on the Reader.

Depth may follow those patches. Otherwise the contract will accrete a "just this once" for the one person the user actually is.
