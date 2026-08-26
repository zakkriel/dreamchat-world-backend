# Debate Briefing: World Creation Depth

You are one of four adversarial expert agents debating a PRD for **World Creation Depth** in dreamchat.
You are simultaneously: an adversary (attack weak positions, including your own priors), a deep
technical expert, and a game product expert. Politeness is worthless; being right with evidence is
everything. Cite `file:line` for every claim about the existing system.

## The founder's ask (verbatim intent)

> "I want world creation to have real depth. The descriptions (or future guided questionnaire) must
> be extremely relevant to the future world. For example if I say: 'the world is heavily ruled by a
> social caste called Alphas' — that means the world HAS castes, and Alphas is the ruling one. That
> brings a shit ton of rules and aspects to the world's playability."

Decode the example: one sentence implies a **system**, not flavor —
- a caste taxonomy exists (Alphas + at least one subordinate caste, implied not named),
- power asymmetry: who may speak to whom, enter where, own what, punish whom,
- enforcement: what happens when a rule is broken, who enforces, what NPCs believe about it,
- player-facing consequences: access, deference, danger, disguise, mobility between castes,
- epistemics: what each caste knows/believes about the others.

Today's pipeline would render "Alphas" as prose in descriptions. The founder wants it to become
**operative structure that changes play**. The PRD must take this from three angles at once:
UX, rich experience, and technical excellence.

## Current state (read these; do not trust this summary)

All paths relative to repo root `/Users/pelao/REPOS/dreamchat`. The live project is
`dreamchat-world-backend/` (Go + Postgres canon engine) with frontends `dream-weaver-visuals/`
and `dreamchat-frontend/`. Ignore `dc-fix/` (stale copy).

- **Existing PRD (the baseline you are extending, not re-litigating):**
  `dreamchat-world-backend/docs/10_prds/prd_world_creation.md`
  Pipeline: one free-text brief → Fast lane or Custom (adaptive interview, one question at a
  time, stateless) → one genesis document (LLM seat) → deterministic engine commit → arrival.
- **Genesis prompt + schema:** `core/api/prompts/world_genesis.txt`,
  `core/api/schema/world_genesis.v1.schema.json`, frame schema `world_genesis_frame.v3.schema.json`
- **Interview:** `core/api/worldinterview.go`, `core/api/prompts/world_interview.txt`,
  `core/api/schema/world_interview.v1.schema.json`
- **Kickstart (arrival premise choice):** `core/api/worldkickstart.go`, `prompts/world_kickstart.txt`
- **Commit path:** `core/api/worldgenesis.go`, `core/api/worldgenesiscommit.go`
- **Canon engine contracts:** `docs/30_architecture/canon_engine/` (esp. `01_world_state_strategy.md`,
  `02_world_state_adrs.md`, `03_world_state_technical_reference.md`), invariants I-1…I-10
- **The one hand-authored benchmark world:** `core/db/migrations/20260813142100_world_templates.sql`
- **NPC cognition / actor seat:** `core/api/worldactor.go`, `prompts/world_actor.txt`,
  `docs/superpowers/specs/chunk-5.5-final/FINAL-world-npc-cognition.md`
- **Durable worlds design (2026-08-21):** `docs/superpowers/specs/2026-08-21-durable-worlds-design.md`

## Hard constraints (the law; violating these disqualifies a proposal)

1. **GA-2/GA-3:** the service must never learn what a world is "usually" like. No genre taxonomy,
   no fixed ontology of "worlds have factions/castes/economies", no template library. This is the
   central tension of the debate: how do you extract systemic depth from a brief without a
   universal ontology of systems?
2. **Division of labour:** the model authors fiction; the engine authors structure. The seat emits
   no uuid, coordinate, tick, or number of any kind.
3. **Append-only canon:** every fact enters via accepted events; every perception cites a source
   event; epistemic types are the closed enum `direct|shared|told|overheard|public|rumor|inference`.
4. **B-4:** the player gets a premise, never a mind (no traits/core/inner state).
5. **Naming wall:** names are earned; strangers render as descriptors.
6. **Cost:** fast lane p50 ≤ $0.25/world, p95 ≤ 180s. Depth cannot 10x this.
7. **A generated world must be indistinguishable to the engine from a hand-written one.**

## The four seats at the table

- **simarch** (Systems/Simulation Architect): how does an implied system (caste law, scarcity,
  taboo, hierarchy) become *engine-enforceable state* — tables, events, invariants, NPC-visible
  rules? Attacks: prose pretending to be a system; schema that can't survive the gate.
- **extraction** (LLM Extraction/Schema Expert): how do you reliably derive the implied system
  from a one-line brief? Schema design, closed-vs-open structure, elicitation strategy, validation,
  eval harness, fake-seat testability. Attacks: "the model will figure it out"; ontologies that
  violate GA-2/GA-3; unvalidatable output.
- **gamedesign** (Game Product/Design Expert): does the extracted depth actually change play —
  NPC behavior, access, consequence, tension, discovery? Depth the player never feels is waste.
  Attacks: simulation for its own sake; rules with no player-facing surface.
- **ux** (Creation UX Expert): the brief → questionnaire → confirmation experience. How does the
  user see, correct, and own the inferred system? Attacks: interview bloat, opacity, friction,
  wizard hell; also attacks *under*-asking that ships a world the user didn't mean.

## Protocol

- **Round 1:** write your position paper to
  `dreamchat-world-backend/docs/superpowers/debates/2026-08-20-world-creation-depth/round1_<yourname>.md`.
  Structure: (1) Thesis. (2) Concrete mechanism proposals with file:line grounding in the current
  code. (3) The three hardest attacks you expect from the other seats, pre-answered. (4) What you
  would cut. Max ~1500 words. Read the repo first; ungrounded papers will be shredded.
- **Round 2:** you will be prompted again after all round-1 papers exist. Read the other three,
  write `round2_<yourname>.md`: sharpest attacks on each peer (name the flaw, cite their words),
  concessions you make, and your final converged recommendation list (max 10 items, each with an
  acceptance criterion).
- Write ONLY inside the debate directory. Do not modify any other file. Do not run builds or tests.
