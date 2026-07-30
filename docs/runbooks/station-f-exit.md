# Station F exit runbook — the founder walks the room

The Station F gate is not green until the founder walks the Drowned Lantern himself, in a real browser,
against real drivers. Station F gave the room **space and consequence**: movement now costs real
beat-time (§2), the locked cellar hatch actually blocks (§5.3 Portal), the front door lets him out, and
a heavy grab **pins him where he stands** (§4 encumbrance). This runbook is that exit.

The deterministic CI proof of the same deliverables is `core/api/station_f_exit_test.go`
(`TestStationF_FakeE2E`): the in-scene move to the bar, the locked-hatch block, the open-door crossing,
and the heavy-grab pin — all through the real HTTP handler, zero network. This runbook is the human
counterpart: the same mechanics, but with a person at the keyboard and models in the seats.

---

## What you're proving

- **Movement is not free.** "Approach the bar" is a real ~6-second walk (`CEIL(8 m / 1.4 m/s)`), and it
  **draws down the beat's tension budget** (§6) instead of teleporting. The bar move repositions Kade's
  coordinate *within* the tavern — his `location_id` does not change, because he never crossed a threshold.
- **The locked hatch is real for everyone, forever.** "Go down to the cellar" is refused: the cellar
  hatch is a closed, **locked** Portal, so the §5.3 accessibility gate denies the traversal and nothing
  is written. Mara holds the key; until it's used, the cellar stays shut.
- **The open door lets him out.** "Go out the front door" crosses Kade to Dock Street — the front door
  is an open, unlocked Portal, so the traversal is permitted and his `location_id` changes.
- **A heavy grab pins him.** Grabbing the ballast crate (100 kg) with an 80 kg carry capacity flips Kade
  to `encumbered` in the same commit (the eager rule, §4). Encumbered is a −100 % move modifier → speed 0
  → **no move fits any budget**. The strain is visible the instant it happens, never a stale reading.

---

## Environment variables

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres DSN. Defaults to `postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable` if unset. |
| `DREAMCHAT_MODE=debug` | Enables the `?viewer=<uuid>` override so you can play as (and inspect) any actor's wall. Required for the founder walk (the play page pins the viewer to Kade). |
| `DREAMCHAT_BRIDGE=fake` | Keyless **dry run**: all seats bound to deterministic fakes. Proves the pipeline boots and a beat round-trips with no API keys — but the fakes emit empty chains, so this mode does NOT produce the founder beat. Use it only to confirm plumbing. |
| `DREAMCHAT_MODEL` | Global model for the real (Anthropic) bridge. Defaults to `claude-opus-4-8`. The live call needs `ANTHROPIC_API_KEY` at request time (bind succeeds without it). |
| `DREAMCHAT_RESOLVE_PROVIDER` / `DREAMCHAT_RESOLVE_BASE_URL` / `DREAMCHAT_RESOLVE_MODEL` / `DREAMCHAT_RESOLVE_API_KEY` | Re-point ONLY the **resolve** seat at an alternate provider (e.g. `openai-compat` → DeepInfra/OpenRouter). See `resolver-live-smoke.md` for the resolve-only recipe. |
| `DREAMCHAT_COGNITION_PROVIDER` / `DREAMCHAT_COGNITION_BASE_URL` / `DREAMCHAT_COGNITION_MODEL` / `DREAMCHAT_COGNITION_API_KEY` | Re-point BOTH **cognition** seats — batch AND isolated — at one alternate provider. |

All other seats stay on the global default (`DREAMCHAT_MODEL` via Anthropic) unless overridden.

---

## 1. Reset the database (seed the world with geometry)

```bash
make reset      # db-down + db-up + migrate + seed
```

`make reset` applies all migrations, then seeds `seed_drowned_lantern.sql` — the Drowned Lantern in its
own **play** world `2222…`. Station F's Task 7 gave that seed a **spatial layer**: nested §3 coordinates
under a `Harbor Quarter of Vael` parent, in-tavern positions for the four souls **and the bar**, the two
portals (front door / cellar hatch) already wired, and a ballast crate holding a heavy stone. That is
exactly what `fn_distance` / `fn_move_duration_actor` / `fn_portal_permits` / `fn_effective_weight` read
at play time — the hand-authored coordinates are a sanctioned §3 test artifact; production mints them.

The play-world ids the founder walk uses (from the seed):

| Entity | UUID |
|---|---|
| World (play) | `22222222-2222-2222-2222-222222222222` |
| Kade (the player) | `2ac70000-0000-0000-0000-0000000000a1` |
| The Drowned Lantern (tavern) | `210c0000-0000-0000-0000-0000000000d1` |
| Dock Street | `210c0000-0000-0000-0000-0000000000d2` |
| Cellar | `210c0000-0000-0000-0000-0000000000d4` |
| the bar (fixture Kade walks to) | `2a7f0000-0000-0000-0000-0000000000f1` |
| Front Door (open, unlocked → Dock Street) | `2a7f0000-0000-0000-0000-0000000000c1` |
| Cellar Hatch (closed, **LOCKED** → Cellar) | `2a7f0000-0000-0000-0000-0000000000c3` |
| Ballast Crate (100 kg with the stone) | `2a7f0000-0000-0000-0000-0000000000f2` |

The geometry that matters (Harbor-Quarter frame, meters): tavern `{200,200}`, dock street `{280,200}`
(80 m out the front), alley `{200,240}` (40 m out the back), cellar `{205,205}` (beneath). In the tavern
frame: Kade `{6,1}` just inside the door, the bar `{6,9}` along the back wall — 8 m, a ~6 s walk.

---

## 2. Run the API on :8080

```bash
export DREAMCHAT_MODE=debug
export ANTHROPIC_API_KEY=sk-ant-...        # for the default Anthropic bridge
# (optional) re-point resolve and/or cognition seats — see the table above

cd core/api
go run .
```

The server logs `... on :8080 (debug=true)`. Keyless dry run instead (plumbing only, no founder beat):

```bash
DREAMCHAT_BRIDGE=fake DREAMCHAT_MODE=debug go run .
```

---

## 3. Run the play frontend

```bash
cd /Users/pelao/REPOS/dreamchat/dreamchat-frontend-play
npm install                                   # first run only
BACKEND_URL=http://localhost:8080 npm run dev
```

Then open the play surface:

```
http://localhost:5173/#/play
```

The play page forwards Kade's uuid as `?viewer=` on the beat POST; the backend honors that override only
in debug mode (`ResolveViewer`, gated on `DREAMCHAT_MODE=debug`), so keep `DREAMCHAT_MODE=debug` set.

---

## 4. The founder's script

Play as Kade. Walk in to the Drowned Lantern; Mara, Jonas and the hooded woman are in the room.

1. **Approach the bar** — e.g. *"I walk over to the bar."* Watch the beat take **real time** (a few
   seconds of the tense budget), not a free jump. Kade ends up **at** the bar, still inside the tavern.
2. **Order a beer** — e.g. *"I order a beer from the barkeep."* A normal Communicated beat.
3. **Try the cellar** — e.g. *"I head down to the cellar."* The hatch is **locked** → the move is
   refused. The narration should not put Kade in the cellar; canon gains no move.
4. **Go out the front door** — e.g. *"I step out the front door onto the street."* The door is open →
   Kade crosses to Dock Street. **See the caveat in §5** — in a `tense` scene the 80 m walk may not fit
   the 30 s budget; drop the scene tension (or expect a `turn_budget` halt) if the door "won't open."
5. **(Optional) Grab something heavy** — e.g. *"I heave the ballast crate onto my shoulder."* Kade goes
   `encumbered`; a follow-up *"I carry it outside"* is **pinned** (he can't move under 100 kg).

### ⚠️ Live-binding caveat — the bar is not a candidate yet

The decompose seat is handed a candidate whitelist of **present actors + the current location only**;
**artifacts are deliberately excluded** (`beatHandler.payload`, `core/api/beathandler.go:457-462` —
"artifacts are not nameable-by-id yet, fail closed beats leaking world contents"). So a live *"approach
the bar"* has **no bar id to bind** — the decompose model can only bind Mara/Jonas/the hooded woman or a
location. The in-scene move **machinery** is real and is proven end-to-end in CI (the E2E scripts the bar
as `to_target_id`, which the passthrough path accepts by existence, never by candidacy) — but the founder
cannot reach it by typing until the bar joins the candidate set. **Two ways to still walk the move:**
approach a **present actor** (Mara/Jonas — an actor target repositions you to them, same machinery), or
move to a **location** (out the front door / down to the cellar — the location cases below). Adding
artifacts (or at least fixed features like the bar) to the candidate whitelist is the follow-up that makes
"approach the bar" playable by hand.

---

## 5. What to watch — and when it feels wrong

| Symptom | What it means | Probe |
|---|---|---|
| "Approach the bar" feels instant / free | A move must cost `distance ÷ speed` ticks and draw down the beat budget. | `SELECT fn_distance(w,kade,bar);` then `SELECT fn_move_duration_actor(w,kade,bar);` — expect 8 and 6. |
| Wrong distance / duration | Coordinates or the walk speed are off, or an entity has no coordinate (treated as origin `{0,0}`). | `fn_distance` (geometry) then `fn_move_duration_actor` (geometry ÷ `fn_effective_speed`). Distance is **LLM-free** — a wrong number is data, not prose. |
| The cellar move "works" when it shouldn't (or the front door is refused) | The Portal gate read the wrong open/locked/connects. | `SELECT fn_portal_permits(w, tavern, cellar);` (expect **false** — locked) and `…(w, tavern, dock_street);` (expect **true**). Portal is accessibility, NOT geometry. |
| "Go out the front door" halts `turn_budget` | The 80 m front-door walk is **58 s** (`CEIL(80/1.4)`), which **exceeds the 30 s `tense` budget** — so in a tense scene Kade is budget-bound indoors (the back door is closed, the hatch locked, and the front door too far). This is correct §6 behavior, not a bug. | Check the scene tension (`SELECT attrs->>'tension' FROM location_state WHERE entity_id = tavern`) and `fn_move_duration_actor`. Loosen the tension (`normal`/`calm`) for the walk to fit, or accept the pin. |
| The heavy grab does NOT encumber | `carried_weight` or `max_load` is wrong, or the crate's contents aren't summing. | `SELECT fn_effective_weight(w, crate);` (expect 100 = empty 8 + stone 92) and `SELECT attrs->>'max_load' FROM actor_state WHERE entity_id = kade;` (80). Encumbered = `carried_weight > max_load`. |
| A move that *should* commit `bounce`s / reads wrong | An adjudicated step (not a passthrough move) went to the resolve seat. | Dump the **resolve-seat** prompt: a bounce is the referee ruling the attempt impossible — read its `reasoning`/`therefore` in the resolve log to see *why* it refused. Passthrough moves (`ActorMoved`/`ObjectRelocated`) never hit resolve; a portal/budget refusal is `gate_reject`/`premise_broken`/`turn_budget`, not `bounce`. |

### Reading a blocked move

A cross-location move that fails the Portal gate surfaces as **`premise_broken`** (the Go premise
re-check, `premiseHolds → fn_portal_permits`, mirrors the SQL twin's `gate_reject` and fires one stage
earlier). Either way the move commits **nothing**. A move that fits no budget (encumbered, or simply too
far for the tension) surfaces as **`turn_budget`**, prefix intact. Neither is a model failure — both are
the engine's arithmetic, computed without asking an LLM.

For the whole set of deliverables deterministically (no keys, no models), run the CI proof:

```bash
cd core/api
DATABASE_URL=postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable \
  go test -count=1 -run TestStationF_FakeE2E -v .
```
