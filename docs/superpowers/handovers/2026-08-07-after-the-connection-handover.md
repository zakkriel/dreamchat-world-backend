# HANDOVER — after the Journey and the connection (2026-08-07)

**For the agent picking this up.** The five-step program the founder approved this session is **built, merged, and green on trunk**. This document says what that means, what is genuinely left, and the two specific things standing between today and the founder actually playing the world. Read the two "before you touch anything" items in §6 before running a command — the database and the branch ruleset both bite.

---

## 1. State in one line

`main` = `140e111`, CI green (go-tests, invariants, schema-contract), **zero open PRs, one branch in the whole repo**. The backend can be played against; nobody has played it.

## 2. What shipped this session

The founder's ask was *"start the journey… and make sure the BE ⇄ FE connection is relevant as we need to connect it to a proper FE and not the ugly 'test' FE."* Both halves are done, in five steps he approved as an ordered ladder:

| Step | What it did | Landed as |
|---|---|---|
| Close the gates | The three Living-World deferrals: bookkeeping atomicity, a runtime scene check on the World Actor, the floor-window world's turn | #29 |
| Ground | Places gained real **areas**; the `{w,h}` box retired outright; `fn_place_at` answers "which place contains this point" | #30 |
| The Journey | Over-budget stops being a dead end and becomes a trip the world can interrupt | #32 |
| The connection | `GET scene/current`, `POST /beats` (streamed frames), `POST /beats/continue`, the journey block; the singular `/beat` deleted | #33 |
| The play page | Built **in the frontend repo, by a parallel effort**, against the frozen contract | frontend #10 |

**The design and every ruling live in `docs/superpowers/specs/2026-08-07-journey-design.md`** — thirteen founder rulings (R1–R13) quoted in his own words, then the derived design. Read §3 before proposing anything that touches the Journey; several of those rulings were made after he was shown the alternatives and rejected them.

The per-step plans are in `docs/superpowers/plans/2026-08-07-rung{0,1,2,3}-*.md`, each carrying a "plan corrections found during execution" section — those are honest records of where the plan was wrong about the code, and they are worth reading before trusting any plan in this repo.

**The durable ledger is `.git/sdd/progress.md`** (git-ignored). It has a task-by-task record with commit hashes, every defect found, and every deferral. Trust it and `git log` over recollection.

## 3. What is actually left

### 3.1 The founder has never played it — and that is the real gate

Every station in this program has had the same exit gate: **he plays it himself**. That has not happened for the Living World, the Journey, or the connection. Green CI is not the gate; his verdict is. The Journey's acceptance test encodes his own worked example (walk out, get interrupted, restate, arrive) and passes — but a passing test is not a person finding the world dull, confusing, or alive.

### 3.2 Two things block a real playthrough

Both are recorded in the ledger and neither is a defect in this session's work:

1. **Nobody is `Player`.** `ResolveViewer` (`core/api/viewer.go:17`) resolves the player as *the world's actor named `Player`* — a documented 0A stub, and the seeded Drowned Lantern has Kade, Mara, Jonas and a hooded woman. Outside debug mode, every request against that world **500s at the door**. Today the only way in is the debug override `?viewer=<uuid>`, honoured only when the server runs in debug. Playing for real needs a session/identity model — Bridge open item #4, still open.

2. **Every seat defaults to Anthropic and needs a key.** `defaultSeatConfig` (`core/api/main.go:90`) binds all seven seats to `anthropic`; only `resolve` and `cognition` have per-seat overrides. Provider neutrality is a standing founder mandate and remains **owed** — and `place_author`, added this session, inherited the same default, so it is one seat worse than before. Without keys the live path fails at decompose; with them, the first real playthrough is also the first exercise of the live World Actor and place-author seats.

**There is a fake path that works today.** `DREAMCHAT_BRIDGE=fake` binds every seat to a deterministic stand-in. Combined with `DREAMCHAT_MODE=debug` and `?viewer=`, that gives a complete, playable loop with no API keys — narration is nonsense, but scene state, journeys, continue, frames and the trace are all real. Use it to exercise mechanics; do not mistake it for a playtest.

### 3.3 Deliberately not built, with reasons

Do not "finish" these without asking — each was scoped out on purpose and the reason matters:

- **The async channel** (`image.ready`, `projection.updated`, `backstage.applied`, `correction.window_closed`) and the **correction-window frame**. None of those subsystems exist. Standing up a channel that can only ever be silent is scaffolding pretending to be a feature.
- **Time-of-day waits** ("wait until 2am"). There is no world calendar — `in_world_label` is free text authored per perception, so no tick means "2am". Waits bind to a *stated span* or a *fact*, never a clock face. A calendar is its own piece of work.
- **Nested coordinate frames.** `fn_place_at` answers within ONE frame, because a place's coordinates live in its parent's frame. Resolving *through* frames needs coordinate transforms and stays with the deferred spatial engine (SPEC-018).
- **NPC and world journeys.** Only the player travels. The journey row is already actor-keyed, so nothing forbids it later.
- **Off-scene eruptions**, the four Aux lenses at full fidelity, World Workspace, images, corrections, multiplayer.

### 3.4 Smaller things a fresh pass would want

- **`turn_budget` still exists** — correctly, as the impossible-move guard (speed 0 → max bigint). If you see it in a halt reason, that means "cannot be done at all", never "too long for this beat".
- **`journey_barred`** appeared during the build: a journey that meets an existing shut or locked way ends rather than routing around it (creation fills gaps only). It is honest, but no play surface explains it to a player yet.
- **Two trace tests were deleted and their intent re-homed** as frame tests. If you are auditing coverage, that is the one place tests left rather than moved.

## 4. The shape of the thing you inherit

**The engine boundary that matters:** the world's-turn composer (`core/api/worldturn.go:39`) is called once per journey leg and is **byte-for-byte unchanged** from the Living World. The Journey lives in `core/api/journey.go` and touches the beat loop in exactly two places — the over-budget gates in `runChain`. If you find yourself editing `worldturn.go` or growing `orchestrator.go`, the boundary is wrong.

**The contract:** every payload carries `schema_version` and every published shape lives in `core/api/schema/` — that directory is what the frontend generates its types from. `make schema-contract` validates real payloads against those schemas in both directions and currently exits 0 with 9/15 covered. Adding an endpoint without adding payload coverage is how a contract quietly rots.

**The wall:** no canon crosses the API boundary, and labels are always the *viewer's own* naming. You can see it working: hand-driving the scene endpoint as Kade returns "Mara" but "the muscle by the bar" and "a hooded figure" — he does not know their names, so neither does the API.

## 5. Two defects worth learning from

Both were found by curling the running server, neither by any test — and both are the same lesson:

- **Every scene reported `tone: null`.** The code read `attrs.tone`; places store `tension`. The unit test *passed*, because its own fixture also wrote `tone`. The test agreed with the bug instead of with the world.
- **The dev server died the moment the world stirred.** `DREAMCHAT_BRIDGE=fake` bound the world-actor and place-author seats to the generic chain-shaped fake, so any fired pressure tier crashed. Their real fakes existed but were reachable only by tests that bind drivers directly, bypassing the factory.

**Hand-drive anything you build.** A test suite that constructs its own fixtures cannot see either class of bug.

## 6. Before you touch anything

1. **The database is shared and stateful.** `make reset` (~4s) rebuilds and reseeds. pgTAP (`make test`) only passes on a **fresh seed**; Go tests write into the fixture world by design, so the order that works is `make reset && make test`, *then* the Go suite. `make schema-check` does **not** reseed — running it leaves the DB unseeded and breaks the next Go run. Go tests need `DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable'`, which is not set in the shell, and always `-count=1` (a cached `ok` proves nothing).
2. **`main` is protected by a repository ruleset** requiring three passing checks. You **cannot** push directly, even for a docs typo. Every change goes through a short-lived branch and PR, then delete the branch — the founder wants the repo branch-free at rest.

Full battery, in the order that works:

```bash
make reset && make test \
  && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... \
  && go vet ./... && cd ../.. && make schema-check && make reset && make schema-contract
```

## 7. Working with this founder

From the standing memory and confirmed repeatedly today: plain language, **no index codes or rule IDs in what you show him** (they belong in docs and PRs, not chat), quote-or-don't-assert — cite the code, never hand-wave — challenge rather than agree, and **one fork at a time, each with a recommendation**. He moves fast and dislikes walls of text.

He is a good editor of technical proposals when given real tradeoffs: this session he rejected a fixed leg count, rejected adding tension to the eruption odds ("leave it only time based… requires no LLM in the middle"), and killed the `{w,h}` box outright with *"that was a cheap solution from the coding agent… an area is the more real and even 'drawable if needed'"* — which was correct, and which no amount of agreeing would have produced.

## 8. The obvious next move

Get him into the world. The shortest honest path is the fake bridge — no keys, no session model, everything mechanical is real:

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
DREAMCHAT_MODE=debug DREAMCHAT_BRIDGE=fake go run ./core/api
# then the frontend repo, pointed at :8080, with ?viewer=2ac70000-0000-0000-0000-0000000000a1 (Kade)
```

That answers "does the loop hold together" in an afternoon. It does **not** answer "is this world alive", because the narration is a deterministic stub — for that he needs the live seats, which needs the keys and, for anything but a debug URL, the session model in §3.2.

Ask him which he wants first. That is a genuine fork with different costs, and it is his call, not yours.
