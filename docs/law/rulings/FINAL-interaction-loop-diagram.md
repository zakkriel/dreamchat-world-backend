# FINAL — Interaction Loop Diagram

The loop has **two layers**, and the world gets the **first word on every step**.

- **Orchestrator** — sequences the beat: decompose once, then for each step the **world acts first** (an LLM
  decision), a **deterministic premise re-check** runs, the player's step runs, the clock advances; after all
  steps, narrate and return.
- **Action process** — the reusable, **actor-agnostic** unit: **perceive → gate → resolve → commit**, run
  identically whether the actor is the Player, an NPC, or the World Actor.

Two principles it encodes: **no "sacred first action"** (the world reacts before your step resolves), and a
**contested outcome is held, not committed** (wind-up telegraphs, you get a reaction beat, then it resolves
with your reaction in it — no fait accompli).

---

## Cast — who performs each stage (the names we use in discussion)

| Actor | What it is | Performs |
|---|---|---|
| **Player** | the human | Input; reaction beats |
| **Engine** | the deterministic Go core; the **only writer of canon** | Scene Payload, Gate, Premise re-check, Commit, clock, the Orchestrator |
| **Decomposer** | LLM seat | Decompose (text → attempts + ids) |
| **Resolver** | the outcome stage, truth-side — **routed sub-process (Diagram 3)**: R0 deterministic router (most actions = zero LLM) → R1 Engine gathers the truth slice → R2 LLM adjudicates (adjudicated routes only) → R3 Engine verdicts the ruling + repair ×1 → R4 commit | Resolve |
| **World Actor / NPC cognition** | LLM seat(s) | "World acts first" — decides what NPCs & the world do · **DEFERRED (SPEC-012)** |
| **Narrator** | LLM seat | Narrate (prose from the Player's perceptions) |

**The Engine has no model of the world.** It reads only structural, always-present fields (existence, reach,
capacity, authority, access, readiness) and does arithmetic over **LLM-minted** typed data; *what things
are* and *what's possible* live in the LLM (which reasons and mints) and in committed canon — never hard-coded
in Engine logic. Judging whether an actor *has the capability* is an input to adjudication (the LLM), not a
gate check.

**Named artifacts:** **Scene Payload** = the scoped context the Engine builds for the Decomposer.
**Perception Payload** = an actor's own view (feeds its `perceive`; the post-beat one feeds the Narrator).

**Named stages:** Decompose · Gate · Resolve (= reality check + outcome) · Commit · Premise re-check · Held
outcome · Narrate — plus the two structures they sit in, the **Orchestrator** (sequences the beat) and the
**Action Process** (perceive → Gate → Resolve → Commit).

**Colours below:** rose = Player · blue = Engine (deterministic) · amber = LLM seat · dashed = deferred.

---

## Diagram 1 — The Orchestrator (one beat)

```mermaid
flowchart TD
    A["INPUT<br/>Player — types free text"]:::player --> SP["SCENE PAYLOAD<br/>Engine builds the scoped context<br/>→ handed to the Decomposer"]:::engine
    SP --> B["DECOMPOSE<br/>Decomposer · LLM<br/>text → chain of ATTEMPTS with ids"]:::llm
    B -->|a reference is unclear| U["UNRESOLVED<br/>Engine asks the Player which one"]:::engine
    U --> A
    B -->|chain ready| L{{"ORCHESTRATOR loop<br/>Engine — for each step in the chain"}}:::engine

    L --> W["WORLD ACTS FIRST<br/>World Actor / NPC cognition · LLM<br/>receives the imminent step's TELEGRAPHED INTENT<br/>(they see 'he's turning to leave', not 'he left')<br/>decides + acts — commits or telegraphs a wind-up<br/>DEFERRED — SPEC-012"]:::llmDef
    W --> IC{"PREMISE RE-CHECK<br/>Engine · deterministic<br/>re-run the next step's GATE on new state:<br/>is its premise still true?"}:::engine
    IC -->|flipped false| STOP["STOP the chain<br/>Engine commits only what happened so far"]:::engine
    IC -->|holds| AP["ACTION PROCESS — the Player's step<br/>Engine + seats · see Diagram 2"]:::engine

    AP -->|contested / reactable| HOLD["HELD OUTCOME<br/>commit only the wind-up (telegraph)<br/>Player reaction beat → resolve WITH it in<br/>DEFERRED — our extension"]:::llmDef
    AP -->|ordinary| CLK
    HOLD --> CLK["ADVANCE CLOCK<br/>Engine"]:::engine
    CLK --> MORE{"more steps?"}:::engine
    MORE -->|yes| L
    MORE -->|no| N
    STOP --> N["NARRATE<br/>Narrator · LLM<br/>renders the Player's Perception Payload"]:::llm
    N --> R["RETURN<br/>Engine → Player"]:::engine

    classDef player fill:#fde5ec,stroke:#c0466b,color:#000
    classDef engine fill:#e8f0ff,stroke:#3b6fb6,color:#000
    classDef llm fill:#fff3d6,stroke:#c8912a,color:#000
    classDef llmDef fill:#f3efe6,stroke:#8a8a8a,stroke-dasharray:5 4,color:#000
```

## Diagram 2 — The Action Process (one machine, any actor)

```mermaid
flowchart TD
    P["PERCEIVE<br/>Engine builds the actor's PERCEPTION PAYLOAD<br/>its own view — same wall for Player, NPC, World Actor"]:::engine --> G{"GATE<br/>Engine · deterministic<br/>structurally possible? exists / reach / access / fit"}:::engine
    G -->|reject| GR["not possible — no commit"]:::engine
    G -->|pass| RS["RESOLVE — the Resolver · **a sub-process** (see Diagram 3)<br/>a hand-off: Engine → LLM → Engine<br/>impossible → bounces (no canon)"]:::engine
    RS --> CM["COMMIT<br/>Engine writes the canon event + generates perceptions<br/>(only for a resolved, non-bounced outcome)"]:::engine

    classDef engine fill:#e8f0ff,stroke:#3b6fb6,color:#000
    classDef llm fill:#fff3d6,stroke:#c8912a,color:#000
    classDef llmDef fill:#f3efe6,stroke:#8a8a8a,stroke-dasharray:5 4,color:#000
```

## Diagram 3 — Resolve (the sub-process: route first, LLM only where unavoidable)

Resolve receives an **attempt** (typed, ids bound). A **deterministic router** picks the resolution mode by
event type — most actions resolve with **zero LLM calls** (thin-slice canon: move/say = passthrough). Only
adjudicated routes (create/destroy, contested changes) pay the LLM path — and its ruling is an **untrusted
proposal** that faces the same verdict-and-repair wall the input side has (ADR-012 discipline, ported).

```mermaid
flowchart TD
    IN["RESOLVE receives: an ATTEMPT (typed, ids bound)"]:::engine --> R0{"R0 · ROUTE — Engine · deterministic, no LLM<br/>resolution mode keyed by event type"}:::engine

    R0 -->|"move / say / access-granted"| PA["PASSTHROUGH<br/>outcome = intent · zero LLM<br/>(thin-slice canon)"]:::engine
    R0 -->|"contract-covered<br/>(move; ObjectRelocated)"| CO["already checked at the GATE<br/>(fits-time / has-room / within-load — blockers only)<br/>nothing left to decide → passthrough<br/>(damage has NO contract → adjudicated)"]:::engine
    R0 -->|"AttributeChanged · unopposed + accessible"| PA
    R0 -->|"create / destroy · contested changes<br/>(AttributeChanged opposed-rule: OPEN)"| R1["R1 · GATHER — Engine · deterministic<br/>truth-side slice BY PARTICIPATION:<br/>bound participants+targets → attrs, links, held items, last-K events<br/>(doc-05 slice pattern, truth-side · widening = open §11.6)"]:::engine

    R1 --> R2["R2 · ADJUDICATE — LLM · DEFERRED SPEC-013<br/>fast-fail (has-capability, an input not a gate)<br/>→ reality check (golem rule) → ruling as TYPED EVENTS<br/>mints only inside contract shapes · whitelist = the R1 slice<br/>will-dependent parts → cognition (SPEC-012), not here"]:::llmDef
    R2 -->|impossible / incoherent| BO["BOUNCE — no canon"]:::engine
    R2 -->|ruling| R3{"R3 · VERDICT — Engine · deterministic<br/>ids ∈ slice ∪ minted (mint = doc-05 matcher first: REUSE or create)<br/>typed-event shape (D-1) · contract shape (A11)<br/>no write contradicting a slice value · provenance stamp"}:::engine

    R3 -->|fail| RP["REPAIR ×1 — errors attached<br/>(doc-04 §2.2 step 5, ported)"]:::llm
    RP --> R3
    R3 -->|"fail after repair"| BO
    R3 -->|pass| OUT
    PA --> OUT["R4 · OUTCOME → COMMIT (Diagram 2)<br/>carries visible/hidden split (SPEC-016)<br/>reactable → HELD for a reaction beat (Diagram 1)"]:::engine
    CO --> OUT

    classDef engine fill:#e8f0ff,stroke:#3b6fb6,color:#000
    classDef llm fill:#fff3d6,stroke:#c8912a,color:#000
    classDef llmDef fill:#f3efe6,stroke:#8a8a8a,stroke-dasharray:5 4,color:#000
```

## Reading it

- **The Action Process is one machine, run by anyone.** The Player's step, an NPC's action, and a World
  Actor event all flow through **perceive → Gate → Resolve → Commit**. Same pipeline for every actor — the
  anti-enum guarantee.
- **Gate ≠ Resolve, and Resolve routes before it reasons (Diagram 3).** Gate (Engine, deterministic) =
  *"can you attempt it?"* — structural only. Then the Resolver: a **deterministic router** (R0) sends most
  actions through passthrough or contract arithmetic — **zero LLM**. Only adjudicated routes pay: Engine
  **gathers** the truth-side slice by participation (R1), the **LLM** fast-fails / reality-checks / rules
  as typed events (R2; impossible → **bounce, no canon**), and the Engine **verdicts** the ruling as an
  untrusted proposal — id-whitelist, shapes, no-contradiction, provenance — with **repair ×1** then bounce
  (R3). Capability lives here, not in the Gate. The Engine brackets the LLM: routes first, verdicts last.
- **Three different "no"s — keep them apart:**
  - **Gate reject** — structurally impossible (not in reach). No canon.
  - **Reality bounce** — possible to reach, but incoherent (you can't fly). No canon.
  - **Outcome failure** — possible *and* coherent, but it didn't land (the keeper stays hard and lies).
    **Writes canon** — failure is a result, and the chain continues.
- **The world's "first word" is two steps, not one.** The *decision* to act — the World Actor / NPC
  cognition choosing to cut in — is an **LLM call** (deferred); that is the real pushback. The **Premise
  re-check** after is the only deterministic part: re-run your next step's Gate and see if its precondition
  flipped. The world *acting* is not a stop; a *broken premise* is.
- **Held outcome = the reaction window.** For a reactable action, only the wind-up commits; the outcome waits
  for the Player's reaction beat, then resolves contested. Replay still holds: telegraph → reaction →
  resolution are committed events, in order.
- **Only three things stop the chain:** Gate reject, a broken premise (Premise re-check), or the turn
  budget. An outcome *failure* does not stop it.

## What is built / deferred / ours / still open

- **Canon, buildable:** the Action Process spine (perceive → Gate → Resolve → Commit); the Premise re-check;
  structural deception (Gate reads canon, the re-check reads perception).
- **Deferred subsystems:** the World Actor + NPC cognition (SPEC-012); uncertain-outcome Resolve /
  adjudication (SPEC-013). Until these exist, "world acts first" has nothing to run and Resolve is
  passthrough.
- **Our extension (banked, not engine-canon):** world-first-per-step + symmetric telegraph + the
  held-outcome reaction window.
- **Still open (decide before building the world tier):**
  1. Does the world cascade run on **every step**, or **only when the clock crosses a pending event**?
  2. Intra-tick **ordering** when several NPCs act on the same tick.
  3. **Presence caps** (~6 present-and-acting; ~10 present-but-known-to-the-World-Actor).
  4. Reaction depth: only the **Player** gets a reaction beat; NPCs do **not** re-react (depth-1).
