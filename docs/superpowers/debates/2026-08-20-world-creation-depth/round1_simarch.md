# Round 1 — simarch (Systems/Simulation Architect)

## 1. Thesis

**Systemic depth is not new structure. It is world-authored rows in the three places the engine
already reads — and everywhere else is prose wearing a schema.**

The engine has exactly three surfaces where a world-authored fact changes what happens without any
new read path:

1. **`entity_registry.entity_kind IN ('faction','group')`** → `fn_visible_perceptions`
   (`core/db/schema.sql:3080-3086`) makes every perception held by such an entity visible to *every*
   viewer in the world. This is the engine's common-knowledge channel. It is live, it is the safety
   wall itself (`core/db/migrations/20260614090002_projection_functions.sql:7-11`), and it is used by
   exactly one seed row (`core/db/seeds/seed_mara_0A.sql:24`) — never by genesis
   (`core/api/worldgenesiscommit.go:305-338` registers only `location|actor|artifact`) and never by
   the benchmark world (`core/db/migrations/20260813142100_world_templates.sql:102-132`).
   `entity_registry` has no `entity_kind` CHECK (`schema.sql:3699-3710`), so this costs zero DDL.
2. **`personality_core.traits`** (`schema.sql:3882-3888`) → the only per-mind text every cognition
   call renders (`core/api/cognitionprompt.go:143-146`, loaded at `:215-220`).
3. **`artifact_state.attrs.{open,locked,connects}`** → `fn_portal_permits`
   (`schema.sql:2625-2636`), the engine's only hard access gate, mirrored in Go at
   `core/api/orchestrator.go:1241-1256` and read by `fn_fact_sheet.reachable` (`schema.sql:1756-1757`).

Anything else is decoration with a receipt. A `caste` attribute on `actor_state` is a legal Tier-2
write (`core/api/verdict.go:148-152`) that **nothing reads**: `fn_fact_sheet` emits a closed key set
(`schema.sql:1743-1787`), and the cognition prompt carries location, tension, roster, traits and
nothing else per-world (`cognitionprompt.go:119-151`). Prose in jsonb is still prose.

**And the founder's complaint is already provable in one grep.** `cast[].standing` — *"What they do
and where they sit in this world's order"* (`core/api/schema/world_genesis.v1.schema.json:130-134`)
— is required (`:117`), refused when blank (`core/api/worldgenesis.go:310-311`), and then **written
nowhere**. The only commit-path consumers of a `genesisActor` are `registerEntities` (descriptor +
canonical name, `worldgenesiscommit.go:292-321`) and `insertMind` (traits, speech_manner, hiding,
`:516-557`). `grep -rn Standing core/api/*.go` returns two hits: the struct field and the validator.
Today the pipeline pays the model to author the world's social order and throws it in the bin.

## 2. Mechanisms

**M1 — the order is entities, not a field.** Add optional `orders[]` to `world_genesis/1`:
`{descriptor, canonical_name, standing_over[]}` where `standing_over` names other authored orders.
Cast gain optional `belongs_to`. Commit each order as an `entity_registry` row with
`entity_kind='group'`, descriptor and canonical name exactly as places are registered
(`worldgenesiscommit.go:305-313`) so `fn_display_name`'s wall covers it. `standing` as it exists is
deleted or becomes `belongs_to`.

**M2 — the law is common knowledge, not prose.** Each order carries `norms[]`:
`{stated, bearing, sanction}` — the rule as this world says it, which order it binds, what happens
when it is broken. Commit each as a `perception_record` **held by the order's group entity**, so
`fn_visible_perceptions` (`schema.sql:3080-3086`) hands it to every viewer with no new read path and
no D-1 bypass. `epistemic_type: public`, already in the closed enum
(`world_genesis.v1.schema.json:262`) and explicitly licensed by B-2's common-knowledge amendment
(`docs/00_strategy/06_rules_register.md:27`).

Ground them in a backstory event — an `AttributeChanged` with no mutations, the shape `writeHistory`
already uses (`worldgenesiscommit.go:404-407`) — and **never** in `world_genesis`.
`fn_perceived_name` reads every `world_genesis`-sourced, subject-linked perception as that entity's
NAME (`schema.sql:2584-2599`); the archivist whose forgery scheme rendered where her name belonged
(`worldgenesiscommit.go:550-555`) is the live proof.

Cost: **zero extra seat calls.** Same single authoring call, no repair loop
(`worldgenesis.go:172-175`); the delta is output tokens. p50 ≤ $0.25 / p95 ≤ 180 s
(`docs/10_prds/prd_world_creation.md:26-27`) survive.

**M3 — the law reaches the minds or it does not exist.** `buildCognitionPrompt` renders SCENE and
THE MINDS YOU SPEAK FOR and nothing else world-specific (`cognitionprompt.go:119-151`). Two additions:
(a) each decided-for mind's line carries its order and the norms bearing on it; (b) roster lines
carry each present actor's order **as that viewer perceives it** — membership is knowledge, and an
unguarded roster field is a naming-wall breach by another door. Both ride the stable cache prefix
(`cognitionprompt.go:12-16`), so they cost cached tokens.

Skip (a) and you reproduce the exact live failure the ADDRESSED line was written for
(`cognitionprompt.go:185-195`): the id was in the payload, nothing said it meant anything, and the
loudest character won.

**M4 — access is a portal, and I am refusing the obvious version.** `fn_portal_permits` takes no
viewer (`schema.sql:2625-2636`). Making "Deltas may not enter the gallery" a portal fact means a new
overload plus `premiseHolds` (`orchestrator.go:1241-1256`), `fn_fact_sheet.reachable`
(`schema.sql:1756-1757`) and the journey's `journey_barred` path. **Not in v1.** A way is authored
`locked` and the caste gate is enforced by the person standing at it, via M3. The door is physics;
the guard is a mind.

**M5 — three new refusals in the belt, no more.** `validate()` already refuses dangling references,
duplicate names, an arrival room with no exit and a secret nobody holds
(`worldgenesis.go:249-495`). Add:
- `standing_over` resolves to authored orders and is acyclic;
- every `norm.bearing` names an authored order with at least one cast member who starts in a place
  reachable from `arrival.place` — a law binding nobody the player can reach is decoration;
- at least one `history[]` entry's `knowledge` records that norm being enforced or broken. The law
  must have already happened once. This is the sibling of the existing "somebody is in the arrival
  room" refusal (`worldgenesis.go:386-395`), the rule that made openings be *in motion*.

**M6 — the invariant, or none of this survives the gate.** The PRD already requires I-1…I-10 in CI
against a *generated* world (`prd_world_creation.md:23`). Add: at arrival tick,
`fn_visible_perceptions(world, player)` returns every authored norm **and the player's own
`perception_record` count is still exactly 1**. AC-4 (`prd_world_creation.md:65`) holds unchanged
*because* the norms are group-held — that is why this seam is the right one. Assert it, because the
naive implementation writes norms as player-held rows and ships a player who arrives knowing the law.

## 3. The three hardest attacks, pre-answered

**extraction: "`orders[]` is an ontology. GA-2/GA-3 forbid teaching the service what worlds contain."**
GA-2's own test is that the *term* survives a sci-fi thriller, a workplace drama and a horror story
(`06_rules_register.md:59`). "A group; a group that outranks another; a rule binding a group; what
happens when it is broken" survives all three. What GA-2 forbids is genre nouns in core vocabulary —
quests, mana, relics. "Alphas" and "caste" are strings the user typed. And the schema already
asserts universals about every world: at least two places
(`world_genesis.v1.schema.json:48`), at least one person (`:112`), everyone hides exactly one thing
(`:163-167`), everyone has traits (`:140-143`). The line was never "no structure"; it was "no *genre*
structure". `orders` is optional and empty by default: a brief implying no order emits none and every
existing path is byte-identical. Conceded: the interview must never ask for one as a slot
(`prompts/world_interview.txt:4` already forbids exactly that), and there must be no fallback that
invents one.

**gamedesign: "Three tables and no play. Depth the player never feels is waste."**
Agreed — which is why M3 and M5 are load-bearing and M1/M2 are only their storage. The player-facing
surface of an order is one thing: an NPC who acts differently because of it, in the same beat loop,
through the same six types. M5's third refusal guarantees the law was already enforced once in
authored history, so it enters the public moment inside the player's first beats. And it is a
falsifiable claim, not a promise: run one brief with and without `orders` and diff the NPC decisions.
If they do not differ, cut the mechanism.

**ux: "You are inflating the document; slower build, and the user can neither see nor correct it."**
One seat call, unchanged (`worldgenesis.go:172-175`) — output tokens, not a second pass. Visibility
is already specified: `working` frames render world-authored text verbatim
(`world_genesis_frame.v3.schema.json:24-27`); "the Alphas, and what the rest owe them" is a frame the
way "four people" is a frame. Correction is genuinely hard and I will not pretend otherwise: canon is
append-only and a built world is not editable (`prd_world_creation.md:38`). So the choice happens
*before* commit, in the existing `choice` frame shape (`world_genesis_frame.v3.schema.json:79-89`),
or it does not happen. That is ux's problem to solve; I will veto any post-commit mutation path.

## 4. What I would cut

- **Any per-world rules table.** No `world_rule`, no DDL for social structure. The group-holder
  channel and `personality_core` exist and are already read; a new table is a new read path, a new
  gate surface, and a new way for a generated world to become distinguishable from a hand-written one
  (`prd_world_creation.md:184`).
- **Any Tier-1 growth.** `tier1Registry` grows "only when we add a check in code"
  (`core/api/tier1.go:3`). No `caste`, no `rank`, no `standing`. Nothing in the engine computes with
  them.
- **Viewer-aware `fn_portal_permits`.** M4's rejected half. Too much blast radius for v1.
- **`cast[].standing` as it exists.** Either it becomes `belongs_to` and is committed, or it is
  deleted from the schema. A required field the validator enforces and the commit silently discards
  (`worldgenesis.go:310-311` versus the whole of `worldgenesiscommit.go`) is worse than absent.
- **Engine-side norm enforcement.** No sanction firing, no rule pass in the beat loop. The gate's
  floor is structural — entity existence, co-location
  (`core/db/migrations/20260724100002_apply_ruled_event.sql:20-27`) — and it stays that way. Laws are
  enforced by people, which is what makes breaking one a story instead of a 403.
