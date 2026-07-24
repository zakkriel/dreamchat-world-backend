# POST-COMPACTION RULINGS — 2026-07-03 → 2026-07-22

**Why this file exists.** The transcript set ends at the 2026-07-01 compaction. Every ruling below was
made by the founder in live sessions AFTER that point and has no transcript backing. When grilling the
FINAL docs against the transcripts, treat this file as the ground truth for the gap — a FINAL-doc claim
tracing here is sourced, not invented.

## Core principles (founder-locked)
- **Blocker-only:** gates and deterministic resolution exist to BLOCK impossibilities, never to award
  success. An action "happens" only because nothing stopped it. Commit = recording the survivor (clerk,
  not judge).
- **Measurements, not verdicts:** state stores measurements (`max_room`, `occupied_room`, `max_load`,
  `carried_weight`); verdicts ("has room", "fits time") are computed at ask-time, never stored. Nouns for
  state, verbs for checks. Elevated to core rule at founder request ("keep it close to your heart").
- **No LLM-free path exists.** Decomposer + world-first cognition are always in the flow; door
  interactions are always-adjudicated. (Claude claimed an LLM-free player-move path twice — the chase
  error and the §5.5 error — corrected both times.)

## Resolution / router
- Physics arithmetic (fits-time, room-check) runs at the **GATE**; resolve = passthrough for
  contract-covered actions. The router exists on A11 grounds (computed once, deterministic), NOT on
  cost-saving grounds (cost framing was Claude's invention, retracted).
- Everything contract-less → the resolution LLM. Damage has no contract → adjudicated.

## Tension (the beat budget)
- Scene attribute. **LLM sets it at scene creation; reviewed at the start of resolving each new act** by
  whichever LLM seat is already open (resolution LLM on adjudicated acts; cognition seats on world/NPC
  steps). Applies to the act being resolved — zero lag. No dedicated review call. The Decomposer NEVER
  touches it.
- Enum: frantic 5s · tense 30s · normal 60s · calm 600s · **none = ∞** (deliberate time-skip; atomic —
  known limitation). Engine validates enum membership + provenance only.
- **Budget consumed cumulatively across the chain**; each resolved action eats its duration; next runs
  only if budget remains. Fitting the budget is JUST another blocker — never awards the action.
- Long adjudicated actions ("six-hour ritual"): **Way A** — the budget is an input to the ruling; the
  ruling fits itself to the beat; progress accumulates as AttributeChanged until a threshold. The engine
  owns the clock; the LLM never invents time.

## move
- Seeded: `walk` 1.4 m/s, one per world. All other movement types LLM-minted `{movementTypeId, baseSpeed}`.
- Modifiers are percentages on specific movement types; stack multiplicatively; **floor −100%**
  (derived: speed 0 = prevented; below is meaningless) ; **no cap** (founder corrected himself from "cap"
  to "no cap"). Units fixed: meters, seconds, kilograms.
- Over-budget move = **REJECT** (v1); journey-split later when between-places topology exists; never
  clock-jump (except `none` tension).

## Space
- **Nested coordinates:** every location has a coordinate within its parent; things inside a location
  (door, bar, ACTORS) have coordinates within it. Actor position-within-scene is NEW state, ruled in.
- Distance = coordinates at the **nearest common parent**. Cross-level imprecision ACCEPTED (fixing it
  needs inferred exit points = reasoning = LLM; hierarchy makes the error marginal). Do not "improve".
- Coordinates are **LLM-minted, never hand-authored** ("bullshit with the authored now" — the seed world
  is a test artifact, not the pattern).

## ObjectRelocated
- Two dimensions exactly. **Volume BLOCKS** (geometric impossibility: `occupied_room + 4^(size-1) ≤
  max_room`). **Weight never blocks — it CONSEQUENCES**: `within-load` as a blocker is DEAD (founder:
  "it does not really support the status system, just works around it — get rid of it").
- **Eager encumbrance:** on any commit changing a carry chain, the engine recomputes carried_weight
  recursively and writes/clears the seeded `encumbered` status (movement −100%) in the SAME commit.
  Engine-written physics statuses are legitimate (provenance's "world rule" clause); judgment-shaped
  statuses stay LLM-side. Gradient (strained-but-moving) deferred.
- Container formula (founder-specified): `effective_weight = (empty_weight + Σ contents) × modifier`,
  recursive. Mundane `max_room ≤ 4^(size-1)`; bag-of-holding = a volume modifier, not an exemption.
  Container `max_load` dormant. Content-modifier IN for v1.
- Relocation has **no time cost of its own** — time lives in the preceding move.
- Throw: no contract → adjudication.

## AttributeChanged (actor-driven)
- **Always adjudicated, v1.** No trivial-sorting exists anywhere: the engine can't read meaning, the
  Decomposer doesn't sort ("the decomposer doesn't do shit besides decompose and assign IDs").
  Cohesion rationale: a locked door is locked for everyone, five beats or five days later.

## Artifact types (the Portal discussion)
- One TYPE, many instances: "door" is not a type; **Portal** is (`connects[2]`, `open`, `locked`).
  **Container** was the first typed artifact (§4 already built it).
- **Engine-bound types are seeded, never minted** (a check is code; the LLM mints vocabulary, never
  grammar). Seed set = Portal + Container, exactly.
- **Two-tier attribute wall:** Tier-1 engine-known (closed set, feeds checks, strict validation) vs
  Tier-2 LLM-invented (open, infinite, engine NEVER reads them, used only by LLM seats — "they are
  created and used by the llm only, and that is ok").
- Discipline rule: a fact meant to physically stop people lands in Tier-1 AND Tier-2 carries the meaning
  — writing only prose-state is a corrupted write.
- Portal = **accessibility, not geometry** (no exit points; distance model unchanged). `reachable` as a
  stored column dies — computed over portal measurements. Single-type per artifact v1 (Narnia wardrobe
  deferred).

## Decompose
- **Adds NOTHING. Ever.** No implied legs, no plan-building, no journeys. Proof case: "get the potions
  from my wardrobe" (two months away) — any helpful-completion rule can't tell it from the three-step
  storeroom walk without judging scale. "Nope, you're not in the room" is CORRECT behavior; failure is
  an answer; the player decides. UNRESOLVED on genuine reference ties.

## Process/meta
- FINAL doc set staged for the repo at `docs/superpowers/specs/chunk-5.5-final/` (branch
  `docs/chunk-5.5-final-set`, delivered as patch + zip; no push credentials in the chat environment).
- Checkpoint confidence at 2026-07-22: expectations ~87 · gate+contracts ~97 · decompose ~90 ·
  commit/perception ~70 · cognition ~50 · resolution mechanism ~35.
- Known Claude failure pattern (twice): zooming into one stage's resolve path and deleting the
  ever-present LLM seats from the accounting. Grill for it.
