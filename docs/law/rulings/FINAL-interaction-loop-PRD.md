# FINAL — Interaction Loop PRD

**What this is.** One player turn, end to end. This document describes the loop in plain terms and is the
reference for implementation. Each stage gets its own spec (with JSON schemas) as a separate FINAL doc.

**Two things to hold above everything else, because earlier drafts lost them:**
1. **The world acts first on every step.** Before the player's step resolves, the World Actor and present
   NPCs get the first word. There is **no "sacred first action"** — the player's action is never
   auto-resolved as typed; it proceeds or is pre-empted by what the world just did.
2. **A contested outcome is held, not committed.** A disruptive action's **wind-up** (the perceivable
   "about to") commits and telegraphs; its **outcome is held — not committed — until the player has had a
   reaction beat**, then resolves *with the reaction in it*. "The enemy moves to strike," never "the enemy
   killed you." No fait accompli.

---

## 1. Two layers

The loop is built as two layers, not one flat pipeline.

**The orchestrator** sequences one beat:
1. **Input** — the player sends text.
2. **Decompose (once)** — turn the text into a **chain of intended steps**. Each step is an **attempt**
   (with real ids), not a committed event — an attempt's outcome is unknown until resolve.
3. **For each step in the chain, in order:**
   - **World acts first** — the World Actor and present NPCs *decide* what to do and run the action process;
     their acts commit or telegraph a wind-up. **The decision is an LLM call** (this is where the world
     pushes back / chooses to intervene) — DEFERRED, SPEC-012.
   - **Premise re-check** — after the world acts, re-run the *next* step's gate against the new state: is its
     premise still true? A **deterministic boolean** — no interpretation. If it flipped false, stop the chain
     (commit only what happened so far). This only checks *whether* the world's committed act broke your
     step; it does **not** decide to interrupt — that decision was the LLM world-actor above.
   - **The player's step** — run the action process. If the outcome is contested/reactable, hold it (see
     point 2 above) rather than committing it outright.
   - **Advance the clock** by the step's duration.
4. **Narrate** what the player perceived.
5. **Return** the response.

**The action process** is the reusable, **actor-agnostic** unit — the same machine whether the actor is the
player, an NPC, or the World Actor:

> **perceive → gate → resolve → commit**

- **perceive** — the actor's *own* view of current state (NPCs and the World Actor see through the same
  perception wall the player does; no omniscient actors).
- **gate** — is it *structurally* possible? (exists / reach / access / fit). Deterministic, no LLM, writes
  nothing.
- **resolve** — two parts: first a **reality check** (is it possible at all — capability / coherence; if not
  it **bounces, no canon**), then the **outcome** (what actually happens; may differ from intent — the
  world's "no").
- **commit** — write the canon event and generate perceptions.

---

## 2. The stages that get confused (read once, carefully)

**Gate — possibility, structural.** *"Can you attempt this?"* Pure state lookup — are you co-located, is it
in reach, is it accessible, does it fit. No LLM. Writes nothing. It does **not** ask whether the attempt
*succeeds*.

**Resolve — truth-side, in two parts.** Reads the real state of actor and target (the full model the gate
doesn't have).

*Part 1 — reality check.* *"Is it actually possible in reality?"* — capability / coherence, "can this actor
even do this". "Fly with no wings" is impossible → it **bounces, writes no canon** ("you can't do that").
Capability lives here, not in the gate — this judgement needs the full model.

*Part 2 — outcome.* *"What actually happens?"* A **deterministic router** picks the resolution mode by
event type — **most actions never touch an LLM**:
- `ActorMoved` / `Communicated` / access-granted changes → **passthrough**: the outcome just *is* the
  intent (thin-slice canon).
- Contract-covered actions (`move`; `ObjectRelocated`) → the arithmetic already ran **at the gate**, as
  blockers only (fits-time / has-room / within-load — see FINAL-action-contracts). Arithmetic never awards
  success; if nothing blocked and the world didn't stop it, the action survives and resolve is
  **passthrough**. (Damage has **no designed contract** — it routes to adjudication.)
- `EntityCreated/Destroyed` and **contested** changes → **adjudicated**: the bounded LLM path below — never
  a free verdict; the decided outcome is logged and replayed. *Deferred (SPEC-013).* A *failure* here still
  writes canon (the keeper hardens and lies) — failure is a result, not an exit.
- `AttributeChanged` splits: unopposed + accessible (open the unlocked door) → passthrough; opposed →
  adjudicated. **The exact opposed/unopposed rule is open.**
- Where the outcome hangs on another actor's **will** (does Mara comply) → that part is **cognition
  (SPEC-012)**, not adjudication: adjudication rules what reality permits; cognition rules what the NPC
  chooses.

**Three different "no"s, and they behave differently:** **gate reject** (structurally impossible → no
canon), **reality bounce** (reachable but incoherent → no canon), **outcome failure** (possible *and*
coherent but it didn't land → **writes canon**, the chain continues).

**How resolve runs — the routed sub-process (R0–R4).** Route first; the LLM only where judgment is
unavoidable; the LLM's ruling treated as an **untrusted proposal** (the same trust class the input side
already has). Walked on *"forge a dagger from a starmetal shard"* — an `EntityCreated`, so it takes the
adjudicated route:
1. **R0 — Route (Engine, deterministic).** `EntityCreated` → adjudicated (the gate table itself: *"not a
   gated action — an adjudicated intent"*). A `move` or `say` would exit here as passthrough — zero LLM.
2. **R1 — Gather (Engine, deterministic).** The truth-side slice **by participation**: the bound
   participants and targets → their attributes, links, held items, last-K events. The doc-05 slice pattern
   reused truth-side — scoped by participation and recency (computable), not "relevance" (not computable).
3. **R2 — Adjudicate (LLM, deferred SPEC-013).** Fast-fail (`has-capability` is an *input*, not a gate; no
   relevant capability → bounce) → reality check (a novice can forge a plain dagger; a legendary
   greatsword → bounce, no canon) → ruling as **typed events**, minting the new thing inside the contract
   shape (a dagger: size 2, weight 1). The whitelist is the R1 slice.
4. **R3 — Verdict + repair (Engine, deterministic).** Every id in the ruling ∈ slice ∪ properly minted —
   and a mint first runs the **doc-05 matcher** against slice + registry: match → **reuse the existing
   id**; no match + a true introduction → create (descriptor mandatory). Then: typed-event shape, contract
   shape, **no write contradicting a slice value**, provenance stamp. Fail → **repair ×1** with errors
   attached (the doc-04 pipeline's own move) → fail again → **bounce** — prefer a rejected ruling over a
   corrupted canon (ADR-013's stance, mirrored to the output side).
5. **R4 — Commit.** The outcome carries its visible/hidden split; reactable outcomes go through the held
   beat; the Engine writes canon with provenance, generates perceptions, advances the clock. Logged once,
   replayed forever.

**Open, named:** the `AttributeChanged` opposed/unopposed rule; slice widening (gate/state §11.6); semantic
coherence beyond the slice; deep provenance enforcement (§11.3); the R2 mechanism itself (SPEC-013).

**Interrupt — two different things that must not be welded together.** "The world interrupts you" is really
two steps:
- **The world *deciding* to act (and what it does)** — Jonas choosing to cut in, an NPC choosing to react.
  This is a *judgment*, so it is an **LLM call** (the World Actor / NPC cognition, SPEC-012, deferred). This
  is the actual pushback.
- **Whether that committed act broke your next step's premise** — re-run the next step's gate against the new
  state; is its precondition still true? This is a **deterministic boolean**, no interpretation, because gate
  is deterministic. `[walk to Mara, talk to Mara]` — after the world acts, is Mara still in reach? If not,
  stop.

So the *decision* to interrupt is never deterministic; the *premise re-check after* is the only
state-computable part. The world *acting* is not itself a stop — a *broken premise* is.

**Held outcome — the reaction window.** A contested/disruptive action does not commit its outcome outright.
Its wind-up commits and is perceivable; the player gets a reaction beat; then the outcome resolves *with the
reaction included* (contested resolution). Replay still holds: telegraph → reaction → resolution are
committed events, in order.

Short version: **gate** = can it be attempted (deterministic). **resolve** = first "possible in reality?"
(bounce, no canon, if not), then "what results" (the "no" — LLM for uncertain outcomes). **interrupt** = the
world *decides* to act (LLM), then a *boolean* re-check of whether that broke your next step. **held
outcome** = the world telegraphs, you react, then it resolves — nothing lands on you unanswered.

---

## 3. The world model the loop runs on

**Six event types — the whole closed set:** ActorMoved, Communicated, ObjectRelocated,
Ownership/AccessChanged, EntityCreated/Destroyed, AttributeChanged.

**No list of verbs.** New richness is **data**, not new types: a novel state change is an `AttributeChanged`
with a new attribute; a purely descriptive action that changes nothing is just perception text.

**`act` is not a type.** A generic physical action is an `AttributeChanged` (open the door →
`door.open = true`). No seventh "act" event.

**Decompose emits attempts, not outcomes.** A `move` maps straight to `ActorMoved`; an `attack` cannot — its
outcome is unknown until resolve. Decompose's job ends at *"here is what the actor is trying."*

**Canon vs perception.** Canon is the truth; perception is what a viewer got. They can differ on purpose:
the keeper can *be* afraid and lie while the player *perceives* confidence. No component lies — the world
does, and the narrator renders only what the player perceived. This is structural: the **gate reads canon,
the interrupt reads perception**, and the gap between them is where deception lives.

---

## 4. The physics contract (the one non-obvious mechanic)

For outcomes with a physical measure, we do **not** ask the LLM each time and do **not** keep a catalog of
every possible effect:

- **The engine hardcodes the grammar** of an action type (its fixed arithmetic and axes).
- **The LLM mints typed vocabulary** inside that grammar, once, when something new appears.
- **The engine computes** with that data forever after — no LLM in the loop.

**`move`:** speed = movement-type base rate × percentage modifiers; time = distance ÷ speed; does it fit.
The LLM may mint a movement type (`climb`, 0.4 m/s) or a modifier status (`limping`: walk −30%). Then:
`walk 1.4 × 0.70 = 0.98 m/s`. Modifiers stack multiplicatively.

**`ObjectRelocated`:** two fixed axes, **volume** and **weight**. Objects have size + weight; containers have
a volume budget + max load. The LLM mints object types and modifier statuses (`waterlogged`: weight ×1.6).
The engine computes fit and carry.

An action gets an open "type" axis **only** when its physics has distinct modes with different base rates
(movement does; `ObjectRelocated` stays two fixed scalars). A purely **social** attempt has no physics axis
— that is the only case that falls to the deferred bounded-LLM ruling.

---

## 5. Worked example — one turn at the Drowned Lantern

**Scene.** Kade (player) is a hunted courier in a dockside tavern. He's been working **Mara**, the keeper,
who secretly recognises him. A **hooded woman** just sat in the corner. **Jonas**, the muscle, is by the
bar. Kade types: **"I cross to the bar and slip the note to her, then ask about the harbormaster."**

**Decompose (once)** → a chain of three attempts, ids filled from the scene:
1. `move` Kade → the bar.
2. `ObjectRelocated` — give `art_note` from Kade to **her**. ("her" is contested: Mara vs the hooded woman.
   If decompose can't choose confidently it emits `UNRESOLVED` and Kade is asked which — say it resolves to
   Mara.)
3. `Communicated` — ask Mara about the harbormaster.

**Step 1 — move to the bar.**
- *World acts first:* Jonas, at the bar, *decides* to shift to block the stool and eye Kade (the World-Actor
  LLM decision; DEFERRED — SPEC-012). This commits and Kade perceives it.
- *Premise re-check (deterministic):* did Jonas blocking flip step 1's premise (bar reachable) to false? He
  didn't fully block — a path still exists — premise holds.
- *Player's step (action process):* perceive (Jonas is there) → gate (bar reachable ✓) → resolve (ordinary
  move, passthrough) → commit `ActorMoved`.
- *Advance clock.*

**Step 2 — slip the note to Mara.**
- *World acts first:* nothing new fires this tick.
- *Premise re-check (deterministic):* premise (Mara in reach, Kade holds the note) still true.
- *Player's step:* perceive → gate (note exists ✓, held ✓, Mara in reach ✓) → resolve (handing a small
  object is certain → passthrough) → commit `ObjectRelocated(art_note, Kade → Mara)`.
- *Advance clock.*

**Step 3 — ask Mara about the harbormaster (the contested one).**
- *World acts first:* Jonas *decides* to lean in, catching that Kade is pressing Mara (the World-Actor LLM
  decision). This is a **wind-up / telegraph** — Jonas is *about to* intervene, but the **outcome is held,
  not committed**.
- *Premise re-check (deterministic):* Jonas's telegraphed lean didn't flip step 3's premise (Kade can still
  speak) — so the chain isn't stopped. But because the world's act is a held, reactable event, Kade **gets a
  reaction beat**: intervene, back off, or press on.
- *Player reacts* (say, presses on). Now the held outcome **resolves with the reaction in it** (contested):
  Jonas's move and Kade's insistence resolve together — maybe Jonas cuts the conversation off, maybe Mara
  clams up. This is the **uncertain social** case → bounded LLM (DEFERRED). Suppose it fails: **canon**
  records *Mara stayed guarded and deflected, Jonas hovering*; Kade's **perception** may be *she looked
  rattled and almost said something*. The narrator renders the perceived version.
- *Advance clock.*

**Narrate** the whole committed sequence from Kade's perceptions → **Return**.

Note what this shows: the world moved **before** each of Kade's steps; the disruptive intervention was
**telegraphed and held**, giving Kade a reaction rather than a fait accompli; and a **failed** social
attempt still wrote canon, with truth and perception diverging on purpose.

---

## 6. What we build, what's deferred, what's still open

**Canon, buildable now:** the action-process spine (perceive → gate → resolve → commit); the interrupt/stop
rule (a perceived event breaking the next step's premise); structural deception (gate reads canon, interrupt
reads perception); the six event types this scene needs; the physics contract for `move` and
`ObjectRelocated`.

**Deferred subsystems:** the World Actor + NPC cognition (SPEC-012); uncertain-outcome resolve /
adjudication (SPEC-013). Until these exist, "world acts first" has nothing to run and resolve is passthrough.

**Our extension (designed, not yet engine-canon):** world-first-per-step + symmetric telegraph + the
held-outcome reaction window.

**Still open — decide before building the world tier:**
1. Does the world cascade run on **every step**, or **only when the clock crosses a pending event**?
   (reactive-per-step vs proactive-clock-driven — cost rides on this.)
2. **Ordering** when several NPCs act on the same tick (a deterministic order is required).
3. **Presence caps** (~6 present-and-acting; ~10 present-but-known-to-the-World-Actor).
4. Reaction depth: only the **player** gets a reaction beat; NPCs do **not** re-react to the reaction
   (depth-1, to keep the cascade from running hot).

---

## 7. Pointers to the specs

Each stage gets a FINAL spec with its data shapes and JSON schemas: `input`, `scene-payload`, `decompose`
(attempts + `UNRESOLVED`), `gate`, `resolve` (the three outcome paths + physics contracts), `apply/commit`,
the `orchestrator` (world-first + interrupt + held-outcome), `narrate`, `return`.
