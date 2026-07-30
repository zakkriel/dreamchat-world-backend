# Design — Grounded Reasoning (computed facts feed the LLM) + Query path + Trace

**Status: DESIGN, founder-approved 2026-07-30 (brainstorming session).** Awaiting founder review of
this written spec, then an implementation plan (writing-plans).

## The problem

**The reasoning is detached from the math.** The engine computes deterministic facts (distance,
duration, reachability, weight, lock state) in SQL; the LLM seats reason in prose; the two never share
a substrate. So the referee decides "can Kade reach Mara" without being handed the actual distance or
reachability — it *guesses*, and a guess can contradict the world. Two visible symptoms during Station
F play:

- A player question — "how much time to get from the door to the bar?" — gets atmospheric prose, never
  the answer (6 s), because a question is not an action and falls through to the narrator.
- The player (and the developer) cannot see the math the engine just did — there is no window into the
  computation.

## The principle (the reframe)

The fix is not a log bolted on the side. It is: **the engine computes the facts and feeds them to the
LLM, so every seat reasons FROM computed truth instead of inventing it.** This is the founder's
"engine does the math, the LLM reasons" split made literal — the math becomes an *input* to the
reasoning. It is a deepening of the existing gather step (RULINGS-2026-07-23 §2: "the facts handed to a
ruling = the entities bound in the action … + their current state and recent history + one hop"), now
also carrying the *computed* physics facts, not only raw state.

Three units follow from it, each with one job.

---

## Unit 1 — The Fact Sheet (the spine)

**What it does.** Given an action's involved entities — the same small set `gather_slice` already
pulls (the actor + the bound targets + one hop of links), never the whole world — it computes the
deterministic physics facts *among those entities* and hands them over with the slice. It is a tiny,
action-scoped fact sheet, not a map of everything.

**Contents (the F contract functions, over the involved entities only):**
- `distance(actor, target)` — via `fn_distance` (nearest common parent, any layer).
- `move_duration(actor, target)` — via `fn_move_duration_actor` (distance ÷ effective_speed).
- `reachable(actor → target's location)` — via `fn_portal_permits` (open ∧ ¬locked when crossing;
  same-scene is trivially reachable).
- `weight / volume` — via `fn_effective_weight` / `fn_volume` for a relocation target (would grabbing
  it encumber; does it fit).
- `lock / open state` — a portal target's Tier-1 `open`/`locked`.
- `budget_remaining` — the beat's tension budget left.

Only the facts relevant to the involved entities are computed (O(involved), never O(world²)). Example
fact sheets:
- "approach the bar" → `Kade → bar: 8 m, ~6 s, same room, no portal needed`.
- "lean on Mara" → `Kade → Mara: same room, reachable, ~2 m`.
- "go down to the cellar" → `Kade → cellar: portal cellar-hatch, locked → not reachable`.

**Computed once, fed to every seat that reasons about that action** (founder-approved symmetric fix —
"reasoning detached from math" is every seat's problem, not just the referee's): the world-first
**cognition** prompt (so an NPC deciding to cut in knows Kade is 2 m from Mara, interceptable), the
**referee's** slice (adjudication rules from truth), and — for queries — the **narrator**. One
computation, reused; no extra cost per seat.

**Perception-scoping (the wall, honored):**
- **Referee = truth-side.** Per RULINGS-2026-07-23 §9's wall note ("the perception walls protect the
  character-mind seats … never the referee"), the referee gets the true fact sheet.
- **Cognition + narrator = perception-scoped.** State a character could not perceive is withheld: a
  *closed* crate's contents, a lock state you cannot see. Spatial facts (distance, duration) are
  perceivable by anyone in the scene — you can see how far the bar is — so they are exposed to all;
  only hidden *state* is gated.

**Interface:** `fn_fact_sheet(p_world_id uuid, p_viewer uuid, p_involved uuid[], p_truth_side bool)
RETURNS jsonb` — assembled beside `gather_slice`. `p_truth_side` selects the referee flavor vs the
perceived flavor. Depends on: the F contract functions (Station F), `gather_slice`.

---

## Unit 2 — The Query path (questions)

**What it does.** Lets the player *ask* about the world and get a real, computed, in-world answer
without acting.

**Flow:**
1. **Decompose emits a `QUERY` element** — a sibling of `UNRESOLVED` in the `beat_chain` output. It
   recognizes the interrogative form ("how long to the bar?", "is the door locked?", "what's in the
   crate?") and binds the referenced entity's id from the candidate whitelist. This is *parsing*
   (syntactic mood: interrogative vs imperative), not the semantic judgment the single-job rule bans
   (RULINGS-2026-07-23 §4: "the decomposer doesn't do shit besides decompose and assign IDs" — it may
   not judge tension/intent/outcome; recognizing a question's *form* and binding its id is the same
   parse job, one more shape). `QUERY` carries the bound target id(s) and the raw question text.
2. **No commit, no adjudication.** A query writes no canon — asking is not acting (RULINGS-2026-07-23
   §3: "the player gets a reaction, never a record"). It is not routed to the referee.
3. **The engine builds the target's fact sheet** (Unit 1, perception-scoped — you learn only what you
   would perceive; a closed crate answers "you can't tell from here").
4. **The narrator phrases it in-world** (founder-approved: the narrator is already the perception-scoped
   player-facing voice; no new seat, no extra call). Given the query + its fact sheet, it answers:
   "It's only a few strides to the bar — a couple of seconds." "The hatch is bolted — locked, and you
   don't hold the key."
5. **Mixed input just works.** "I walk over — how long to slip her the note?" decomposes to
   `[ActorMoved(bar), QUERY(note→Mara)]`; the walk resolves, the query's fact sheet rides to the
   narrator, one answer covers both.

**Interface:** a `QUERY` variant in `beat_chain.v2` (target ids + question text) + `DecodeAndValidate…`
handling; the orchestrator routes `QUERY` elements to a read-only fact-sheet build; the narrate payload
gains the query fact sheets. Depends on: Unit 1, the decompose seat, the narrate seat.

**Out of scope (v1):** a "preview the cost before committing an action" affordance (distinct from a
question) — a natural follow-on that also rides Unit 1, deferred.

---

## Unit 3 — The Trace (the reasoning log) — built LAST, over the new code

**What it does.** Surfaces the whole beat pipeline — the math and the reasoning — as a structured
"behind the curtain" log, so the player/developer can see exactly what happened and why.

**What it captures** (appended stage-by-stage as the beat runs — pure capture, no new LLM):
- **Decompose:** raw text → the chain (attempts / queries / UNRESOLVED) with bound ids and labels.
- **Per attempt:** the fact sheet (distance → speed → duration; budget total/spent/remaining; portal
  permit; volume/weight) and the gate result.
- **World-first:** which present NPCs decided none / commit / telegraph.
- **Referee (adjudicated only):** the ruling's own `reasoning → therefore → outcome` text.
- **Result:** the halt reason and what committed.

**Surfacing:** emitted as a `reasoning_log` field in the beat response **only when
`DREAMCHAT_MODE=debug`**; the play page renders it as a collapsible per-beat panel. Raw numbers + the
LLM's own words (a developer view, not phrased prose).

**Truth-revealing by design → debug-gated.** The trace shows truth-side reasoning (it may contain a
secret's truth — the referee's actual read of Mara). That is correct for a debug tool — the founder
looking behind the curtain of his own world — and it is exactly why it is `debug`-only and never
emitted to a real player. The perception wall is not violated: the trace is a developer affordance, not
a character perception.

**Interface:** a `BeatTrace` struct threaded through the orchestrator (each stage appends), serialized
into the response under `reasoning_log` when debug. Depends on: Units 1–2 and the existing pipeline
(this is why it is built last — it captures the new code).

---

## Data flow (one beat, with the new steps)

```
input text
  → decompose  → chain of [attempt | QUERY | UNRESOLVED]         (Unit 2: QUERY is new)
  → per element:
      QUERY    → fn_fact_sheet(perceived) → held for the narrator (Unit 1,2; no commit)
      attempt  → world-first cognition   [+ fact_sheet]           (Unit 1 feeds cognition)
               → premise re-check
               → route: passthrough → commit
                        adjudicated → gather_slice [+ fact_sheet truth-side] → referee → verdict → commit
               → advance clock
  → narrate    [+ query fact sheets]                              (Unit 2: narrator answers queries)
  → response { narration, result, reasoning_log? }               (Unit 3: trace when debug)
     ↑ BeatTrace appended at every stage above                    (Unit 3)
```

## Corpus grounding (every decision traces to a locked ruling)

- Facts-feed-the-LLM = a deepening of the gather step (RULINGS-2026-07-23 §2).
- Fact sheet computed by the F contract functions (FINAL-action-contracts §2–§6; Station F).
- Referee is truth-side; character-mind seats are walled (RULINGS-2026-07-23 §9).
- A query writes no canon — asking is not acting (RULINGS-2026-07-23 §3).
- Decomposer stays single-job; `QUERY` is a parse shape, not a judgment (RULINGS-2026-07-23 §4;
  FINAL-decompose).
- Candidate binding for `QUERY` reuses the perceived-entity whitelist (RULINGS-2026-07-30 §1).

## Decisions resolved (the forks)

1. **Fact delivery: eager (A)** — an action-scoped fact sheet pre-computed with the slice, not
   on-demand LLM tool-use (a model that forgets to ask keeps guessing).
2. **Question entry: decomposer emits `QUERY` (A)** — one seat, one pass; syntactic, not judgment.
3. **Query answered by: the narrator (A)** — the existing perception-scoped player voice; read-only.
4. **Fed to: all reasoning seats (symmetric)** — cognition, referee, narrator; computed once, reused.
5. **Trace: debug-gated, raw, truth-revealing** — a developer affordance, built last over the new code.

## Testing approach

- Unit 1: SQL/Go tests that `fn_fact_sheet` computes the right numbers over the involved entities
  (distance/duration/reachability/weight/lock) and that the truth-side vs perceived flavors differ
  where a character cannot perceive (closed crate contents withheld in the perceived flavor, present in
  truth).
- Unit 2: a `QUERY` decodes and binds ids; a query commits nothing (canon count unchanged); the narrator
  prompt carries the query fact sheet; mixed input resolves the action AND carries the query.
- Unit 3: the `reasoning_log` appears only under debug; it captures each stage's real values (assert on
  a beat's distance/duration/halt in the trace); non-debug responses carry no trace.

## Build order & scope

- **Build order:** Unit 1 (fact sheet) → Unit 2 (query path) → Unit 3 (trace, last, over the new code),
  exactly as the founder set it.
- **Its own plan / body of work**, separate from Station F and the Journey. Cross-cuts decompose,
  gather, cognition, resolve, narrate.
- **Sequencing vs F/Journey is a separate founder decision** (this design does not set it): it can land
  before the F playthrough (so the math is visible while playing) or after.
- **Out of scope:** the cost-preview affordance (v1 defer); any new physics (Station F owns those);
  the Journey (its own plan).
