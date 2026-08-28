# play-loop · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-7 · The play loop ·
**Parent bounded context:** World Engine (output crosses into Compendium & Play UX)

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `play-loop.tech.md` holds how it is built; `play-loop.seams.md` holds what
crosses its boundary.

---

## What this domain is for

**One job: one player turn** — decompose → world-first → resolve → commit → narrate → stream.

The player types; the decomposer turns the words into a chain of **attempts, never outcomes**
(`beat_chain/2`'s own title: *"chain of ATTEMPTS (never outcomes), ids from the candidate
whitelist"* — `core/api/schema/beat_chain.v2.schema.json:4`). The world gets the first word on every
step; the resolve seat rules what actually happens; the door records it; the narrator renders what
the player perceived. Deterministic machinery exists to **block impossibilities, never to award
success** — founder-locked in `docs/law/rulings/FINAL-action-contracts.md`: *"An action 'happens'
only because nothing stopped it."* Success is the absence of a blocker.

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Beat** | One player input and everything until the next point of agency. Never maps 1:1 to canon (`C-5`). |
| **Attempt** | One element of the chain: what the player *tries*, with real ids. What happens is resolve's job. |
| **Gate reject** | Structurally impossible — broken reference, out of vocabulary. **No canon.** |
| **Reality bounce** | Reachable but incoherent, ruled `impossible` by resolve. **No canon.** |
| **Outcome failure** | Possible and coherent, but it didn't land. **Writes canon** — *"the keeper hardens and lies"* (`core/api/ruling.go:233`). The three "no"s are not interchangeable. |
| **Telegraph** | A disruptive NPC's wind-up, committed as canon; it ends the beat and the un-run chain is discarded. |
| **Held outcome** | The NPC's intended act, carried as loop state (not canon) until the player's next input. |
| **Reaction beat** | The next input: all held acts + the player's answer resolve in **one combined ruling**, depth 1. |
| **Tension** | The beat's time budget in ticks (`core/api/tension.go`). Over budget is a Journey (WE-5), not a reject. |
| **`UNRESOLVED`** | A genuine tie: the player's own phrase plus **≥2** candidates. Definitionally ambiguity, never "could not bind" (failure-log #36). |
| **`QUERY`** | A question put to the world. Not an action: never commits, never advances the clock, never reaches the referee. A question put to a *person* is speech (`Communicated`). |
| **Continue** | Advances the current moment; never fast-forwards the world (`C-6`). |

## What this domain is not

- **Not recording.** We propose and decide; the commit doors record (`D-1`). Canon spine owns them.
- **Not who learns it.** Perception (WE-3); and narration passes the naming wall (WE-4) before a player sees it.
- **Not what an NPC decides.** Cognition (WE-8) proposes attempts; they run through *our* pipeline, no bypass.
- **Not physical possibility.** Space (WE-5) owns distance, duration, reachability. We ask; it answers.
- **Not the clock.** Time (WE-6). We spend the budget; we do not define it.
- **Not the surface.** `dream-weaver-visuals` pins the beat-frame contract by exact string equality (`seams.md`; versions: the frontend `const PIN` block).
- **Not genre mechanics.** HP, spells, sanity are a module (`GA-4`, `D-2`) — and modules do not exist yet.

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `C-5` | A beat may produce zero, one, or several canonical changes. | One-in-one-out code asserts what the design forbids. |
| `C-6` | Continue advances the current moment only. | Fast-forwarding steals the world's chances to act. |
| `C-7` | Longer chains give the world more chances to interrupt. | Executing a whole chain blind forces outcomes past the world. |
| `C-9` | Scene transitions are internal mechanics; the UX is seamless. | A player managing a state machine is the failure this rule names. |
| `B-11` | Cognition fires on a perceptual trigger, never a free-running loop. | A belief ticker is the banned architecture. |

Two more product laws live as founder-locked text, cited not restated: **block-never-award**
(`FINAL-action-contracts.md`, quoted above) and **the decomposer adds nothing** — *"a failure
message beats a guessed plan, every time"* (`FINAL-decompose.md`; the two-month-horse-trek proof
case lives there).

## What is deliberately not built here

- **No action-type taxonomy.** The UX law forbids it: *"The PRD should not define a long taxonomy of
  action types such as combat, romance, investigation…"* — LLM abstraction is the product decision,
  not unfinished work (live UX doc §4.2, via `digest/S09` topic 19).
- **No helpful completion in decompose.** No missing steps, no plan-building, no judgment calls of any
  kind — the ban is absolute, not a default (`FINAL-decompose.md`, founder-locked).
- **No engine math beyond the contract table.** If a computation is not in
  `FINAL-action-contracts.md`, the engine cannot do it; it routes to the resolution LLM. Opposition
  by another actor is always that actor's own act, never arithmetic.
- **No automatic world rollback.** The correction window is local to the current moment; deep
  rewrites need explicit advanced handling (UX doc §4.6). Do not grow an undo system.
- **No genre mechanics in core.** `GA-4`/`D-2`; module architecture is B3 and unbuilt. A module may
  propose; it may never write (`D-1`).
