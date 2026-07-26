# Station E exit runbook — the founder plays the beat in a browser

The Station E gate is not green until the founder walks the loop himself, in a real browser, against
real drivers. This runbook is that exit: walk in as Kade, lean on a **Mara who has her secret** about
the harbormaster, and watch **Jonas react** — the telegraph → reaction beat, played end to end.

The deterministic CI proof of this same loop is `core/api/station_e_exit_test.go`
(`TestStationE_FakeE2E`): telegraph beat → reaction beat through the real HTTP handler, zero network.
This runbook is the human counterpart — the same loop, but with a person at the keyboard and models in
the seats.

---

## What you're proving

- **Mara is guarded, not omniscient.** She holds a PRIVATE record about the player (the harbormaster
  secret). When the player leans on her, the mechanical §5 split flags her → she rides an **isolated**
  cognition call that carries her secret *alone*. Her FACE to a stranger stays a stranger's: the secret
  never reaches the shared/decompose/narrate seats, so it cannot leak into what the player sees. The
  wall holds by construction, not by prompt discipline.
- **Jonas reacts.** He holds nothing private → he rides the shared **batch** call. On a disruptive
  press he **telegraphs** his cut-in: the wind-up commits as canon and the beat ends.
- **The reaction beat is playable.** The player's next input meets Jonas's held act in ONE combined
  ruling; the world carries the reaction state in the `held_outcome` table (no server session), so it
  survives even a server restart between the two inputs.

---

## Environment variables

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres DSN. Defaults to `postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable` if unset. |
| `DREAMCHAT_MODE=debug` | Enables the `?viewer=<uuid>` override so you can play as (and inspect) any actor's wall. Required for the founder walk. |
| `DREAMCHAT_BRIDGE=fake` | Keyless **dry run**: all six seats bound to deterministic fakes. Proves the pipeline boots and a beat round-trips with no API keys — but the fakes are *quiet* (no telegraph, no secret cognition), so this mode does NOT produce the founder beat. Use it only to confirm plumbing. |
| `DREAMCHAT_MODEL` | Global model for the real (Anthropic) bridge. Defaults to `claude-opus-4-8`. The live call needs `ANTHROPIC_API_KEY` at request time (bind succeeds without it). |
| `DREAMCHAT_RESOLVE_PROVIDER` / `DREAMCHAT_RESOLVE_BASE_URL` / `DREAMCHAT_RESOLVE_MODEL` / `DREAMCHAT_RESOLVE_API_KEY` | Re-point ONLY the **resolve** seat at an alternate provider (e.g. `openai-compat` → DeepInfra/OpenRouter). See `resolver-live-smoke.md` for the resolve-only recipe. |
| `DREAMCHAT_COGNITION_PROVIDER` / `DREAMCHAT_COGNITION_BASE_URL` / `DREAMCHAT_COGNITION_MODEL` / `DREAMCHAT_COGNITION_API_KEY` | Re-point BOTH **cognition** seats — batch AND isolated — at one alternate provider. They are one env family: the same NPC-decision workload split only by the wall. |

All other seats stay on the global default (`DREAMCHAT_MODEL` via Anthropic) unless overridden. Repointing
one seat's entry changes only that seat (D-13).

---

## 1. Reset the database (seed the world)

```bash
make reset      # db-down + db-up + migrate + seed
```

`make reset` applies all migrations, then chains the **seed** step: `seed_mara_0A.sql` (the
deterministic Mara **fixture** world `1111…`, for the test suites) followed by
`seed_drowned_lantern.sql` (the Drowned Lantern scene in its own **play** world `2222…` — the two
seeds write DISJOINT worlds; see the `Makefile` `seed` target). Content canon for the scene lives in
`docs/superpowers/specs/chunk-5.5-final/FINAL-drowned-lantern-souls.md`.

The cast the founder plays against — what the seed actually places:

- **Kade — the Player.** No personality core: he is a premise, not a mind. He walks in **carrying the
  sealed note** (gray wax; near-weightless; contents deliberately **unauthored** — Tier-2 flavor only,
  no canon text behind the seal).
- **Mara — with her secret.** She knows Kade is *Reyna's brother* — the boy who ran the messages while
  she hid Reyna's family in the cellar nine days, five winters back — a **life-debt she has never said
  aloud**. That recognition is a **private `perception_record`** subject-linked to Kade *and* Mara —
  never a core trait (cores ride the shared cognition prompt, so a secret in a core would leak by
  construction). Her core is only the guarded-barkeep face a stranger gets. She holds the **cellar key**.
- **Jonas — knows OF the secret without knowing it.** His private record is only that twice he watched
  Mara go pale at a harbor face and learned to stand closer instead of asking — no "Reyna", no ledger.
- **The hooded woman — at the corner table.** She has a thin core and her own private record (she took
  the paymaster's contract to confirm a young, dark-haired courier — "The one by the door could be him.
  I am not sure. Yet."). In the dedicated play world she is **placed in the room** with the others (the
  fixture-world compromise that kept her un-placed no longer applies — there is no frozen golden
  projection to protect here).

The room itself:

- **Tavern tension = `tense`.** The room reads calm, but two of the four people in it are pretending.
- **Three portals off the tavern** — the **front door** (open, unlocked → Dock Street), the **back
  door** (closed, unlocked → Alley), and the **cellar hatch** (closed and **LOCKED** — the first
  Tier-1 lock in play → Cellar; Mara holds the key, and the cellar is where the life-debt happened).

The play world/actor ids from the seed (founder Option B: the play scene has its OWN dedicated world
`22222222-…`; the legacy fixture world `11111111-…` is for the pgTAP/Go test suites only and no longer
carries any play content):

| Entity | UUID |
|---|---|
| World (play) | `22222222-2222-2222-2222-222222222222` |
| Kade (the player) | `2ac70000-0000-0000-0000-0000000000a1` |
| Mara | `2ac70000-0000-0000-0000-0000000000a2` |
| Jonas | `2ac70000-0000-0000-0000-0000000000a3` |
| Hooded Woman (at the corner table) | `2ac70000-0000-0000-0000-0000000000a4` |
| The Drowned Lantern (tavern) | `210c0000-0000-0000-0000-0000000000d1` |

---

## 2. Run the API on :8080

Real drivers (Anthropic default). Set `ANTHROPIC_API_KEY` (or the per-seat overrides above) first:

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

The play page lives in the frontend worktree. Point it at the backend and start the dev server:

```bash
cd /Users/pelao/REPOS/dreamchat/dreamchat-frontend-play
npm install                                   # first run only
BACKEND_URL=http://localhost:8080 npm run dev
```

Then open the play surface:

```
http://localhost:5173/#/play
```

> The `#/play` surface ships with the Station-E **frontend** PR (from its own worktree branch). If the
> route isn't present yet on the checked-out frontend, pull that branch first; the Compendium surfaces
> (`#/actors`, `#/timeline`, …) are always available for inspecting the wall in the meantime.

---

## 4. The founder's script

Play as Kade. The play page's `postBeat` (`src/api.ts`) forwards Kade's uuid as `?viewer=` on the beat
POST — the play world has no name-based `Player` actor to fall back to, so the viewer is passed
explicitly (`PLAY_VIEWER` in `src/pages/PlayPage.tsx`). The backend honors that override only in debug
mode (`ResolveViewer` in `core/api/viewer.go`, gated on `DREAMCHAT_MODE=debug`), so keep
`DREAMCHAT_MODE=debug` set. The Compendium reads (`#/actors`, `#/timeline`, …) still forward the
page-URL `?viewer=` for inspecting any actor's wall. Switching which actor you play from the play page
is not supported today — the viewer is pinned to Kade.

1. **Walk in** to the Drowned Lantern. Mara and Jonas are present.
2. **Lean on Mara about the harbormaster** — e.g. type: *"I lean on Mara and press her about the
   harbormaster."*

What to expect:

- **Mara's isolated call fires.** Because Mara holds a private record *about the player*, the split
  routes her to the isolated seat — her secret rides that call alone. She answers guarded: she knows
  more than she says, but she is **not omniscient**, and **her FACE to you stays a stranger's**. The
  harbormaster secret does not appear in your narration. (That is the deception split + the perception
  wall, both holding at once.)
- **Jonas telegraphs.** If your press is disruptive, Jonas (batch seat) telegraphs his cut-in: the
  narration delivers his **wind-up**, and the beat ENDS on it — your lean never resolves, because the
  world seized the moment first.
3. **React.** Your next input (e.g. *"I shove Jonas back, then whisper to Mara"*) is the **reaction
   beat**: Jonas's held cut-in and your shove collide in one combined ruling, then the whisper runs as
   a normal follow-on. Play it end to end.

---

## 5. When it feels wrong — which seat's prompt to inspect

Turn on `DREAMCHAT_MODE=debug` and, when a beat reads wrong, dump the offending seat's prompt (log it
from the driver, or reproduce the seat call in isolation). The symptom points at the seat:

| Symptom | Suspect seat | Why |
|---|---|---|
| The parse of your text is wrong (wrong target, wrong action) | **decompose** | It turned your prose into the wrong closed-vocabulary chain. |
| Mara leaks the secret into narration, OR she acts blankly ignorant of it | **cognition_isolated** / **narrate** | Her secret must ride the isolated call only; narration is perception-bound. A leak means the wall broke; blankness means her private record didn't reach her isolated prompt. |
| Jonas never reacts (or reacts when he shouldn't) | **cognition_batch** | His decision (`none` / `commit` / `telegraph`) comes from the batch call. |
| The combined ruling is wrong (misses an actor, wrong outcome) | **resolve** | The referee is truth-side and sees all involved parties (§9); inspect the one combined-ruling prompt. |

The **wall is proven mechanically** in `core/api/wall_test.go`: it asserts on the EXACT prompts every
seat is handed — secrets never enter the shared/decompose/narrate prompts, the isolated call carries
them, and the referee stays sighted. If a live model *seems* to leak, re-run those tests first: a green
wall test means the construction is intact and the fault is model prose, not a broken wall.

For the whole loop deterministically (no keys, no models), run the CI proof:

```bash
cd core/api
DATABASE_URL=postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable \
  go test -count=1 -run TestStationE_FakeE2E -v .
```
