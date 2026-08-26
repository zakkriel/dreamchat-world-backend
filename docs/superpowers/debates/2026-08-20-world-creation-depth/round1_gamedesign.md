# Round 1 — gamedesign (Game Product/Design Expert)

## 0. The example is not the requirement

The founder's ruling-caste sentence is *one instance* of the general case: **a brief implies a norm —
any rule binding who may do what to whom, in whatever fiction.** A ship's rota, a debt falling due, a
river nobody crosses after dark, a hive's chemical protocol, a hospice's night-shift authority are the
same object to this engine and must be the same object to this PRD. So a disqualifier applies to every
proposal below, mine included: **an acceptance criterion only a hierarchical world can satisfy is
itself a GA-2/GA-3 violation** (`prd_world_creation.md:175-180`) — it teaches the service what a world
usually contains through the test suite instead of the schema, the same crime through a side door.
Every criterion in §2 is phrased over "the authored norm, whatever it is"; where I write "caste" it is
an illustration of a shape, never a field, enum, or requirement. The word belongs in no identifier,
prompt line, schema key or fixture name — only inside user prose, where it is content.

## 1. Thesis

Depth is exactly the set of facts reaching one of the **four surfaces the player can feel**. Four, all
deployed, all countable:

1. **What a mind decides** — the cognition prompt renders precisely location, tension, the present
   roster, the decided-for minds' `personality_core.traits` + malleability, that mind's private
   perceptions, the public moment, the computed fact sheet, the imminent wind-up
   (`core/api/cognitionprompt.go:119-178`). Nothing else exists to an NPC.
2. **What the world refuses** — the gate's floors are registry existence, portal passage, speaker/
   listener co-location, container volume, a mandatory descriptor (`core/db/schema.sql:161-218`,
   mirrored `:444-497`). The referee reasons from a fact sheet whose whole vocabulary is `distance_m,
   move_duration_s, reachable, open, locked, weight_kg, would_encumber, contents` (`:1735-1800`),
   declared engine truth (`prompts/resolve.txt:8`).
3. **What the player can review** — compendium pages built from perceptions filed by
   `perception_subject`, each carrying `epistemic_type`, `confidence`, decay (`:1143-1206`), for the
   three kinds routed at `core/api/main.go:45-50`.
4. **What arrives unasked** — the pending ledger (`:1557-1566`, `fn_due_pending:1573-1579`) and the
   World Actor's pressure roll (`fn_pressure_chance:2643-2670`).

The one existing proof of a rule that changes play is `tension`: an authored class
(`world_genesis.v1.schema.json:327-331`) → an engine number the beat budget enforces
(`core/api/tension.go:28-45`, which turns leaving a room into a Journey,
`20260813142100_world_templates.sql:470-494`) → a word the player sees as `tone`
(`core/api/scenehandler.go:213-216,243`). Class in, mechanic out, visible surface, zero knowledge of
genre. Every depth mechanism here must take that shape or it is a brochure.

Measured against those surfaces, the pipeline **already discards the depth it authors**:

- `cast[].standing` — the schema's deliberately structure-free "what they do and where they sit in
  this world's order" (`world_genesis.v1.schema.json:130-134`) — is parsed (`worldgenesis.go:119`),
  validated non-empty (`:310`), and **written nowhere**: `insertMind` commits only `speech_manner`,
  traits, the secret (`worldgenesiscommit.go:516-557`). The one field already carrying a person's place
  in *any* order is dropped. Adding `groups[]` beside it changes nothing about that.
- `arrival.why` — the player's premise — labels a choice button (`worldgenesishandler.go:534,661`) and
  is never persisted; `world_character` stores descriptor and canonical name only (`:201`).
  `places[].kind` is dropped too; only objects' `kind` reaches state (`:677`).
- **Refusal is silent.** `gate_reject`/`premise_broken` with an empty delta renders one line —
  *"NOTHING RESOLVED: the attempted action did not happen"* (`narrateprompt.go:227-228`) — with **no
  cause**, and `narrate.txt:29` then forbids depicting change. Walk at a locked hatch today and the
  player is told the world held its breath. `halt_reason` reaches the client as a bare machine string
  (`schema/beat_frame.v5.schema.json:204-206`).

The ordering is therefore forced, and it is not the one the other seats will propose: **behaviour
first, legibility second, discovery third, gate enforcement last.** Any rule pushed into the gate
before `narrateprompt.go:228` is fixed makes the product worse — it converts depth into an
unexplained refusal, which players read as a broken game, not a world with rules.

## 2. Mechanisms

**G0 — Wire or delete; no authored field survives that no surface renders.** `standing` lands in
`personality_core` (the only channel cognition reads, `cognitionprompt.go:143-146`) or leaves the
schema; `arrival.why` lands on `world_character` and in the narrator's context or leaves. *Acceptance:*
a test asserts every non-numeric `world_genesis/1` leaf reaches a table some prompt or payload reads.
We are debating new schema while the current schema leaks.

**G1 — A norm is authored per-mind and per-holder, never as world prose.** Whatever the brief implies
becomes play only as (a) a trait on whoever upholds it whose `manner` states it behaviourally, and (b)
a perception held by whoever is bound by it stating what *they* believe breaking it costs. Both
channels are proven by hand and neither knows what kind of rule it carries: `distrusts_authority —
"the harbormaster's men drink free and learn nothing"` (`20260813142100_world_templates.sql:199`) is a
norm wearing a trait; Jonas *knows of* a secret without knowing it (`:248-251`) is asymmetric
epistemics per holder. Genesis already demands knowledge carry its path (`prompts/world_genesis.txt:12`,
`world_genesis.v1.schema.json:240-266`). *Acceptance, structure-free:* where a norm is authored, **two
holders' beliefs about it differ in content and in `epistemic_type`**, and no holder holds `direct` for
what they were only told. No rank, tier or membership required; identical for a caste, a debt or a
taboo; vacuous — not failed — when the brief implies no norm.

**G2 — Refusal states its cause, in world.** Extend the NOTHING RESOLVED render (`:224-228`) with the
obstacle already computed: the portal's `locked` (`fn_portal_permits`, `schema.sql:2625-2636`), the
absent listener, the encumbrance. The narrator already answers player questions from that exact
vocabulary (`narrate.txt:33`); it is simply never told why the act failed. *Acceptance:* a move at a
locked way narrates the obstruction. Zero new seats, zero new calls. Without G2, nothing else here is
worth building.

**G3 — Norms are enforced by bodies and locks, not permissions.** A mind can already *physically
block* you: `cognition.txt:8` licenses the cut-in `ActorMoved`; `resolve.txt:6` gives the worked
example ("Torrek steps between them, blocking the way"). A place a norm keeps you out of is a `locked`
way (`world_genesis.v1.schema.json:332-336`) whose key an authored person carries
(`objects[].where.carried_by`, `:204-208`) — entry becomes a social problem, which is the game. The
design law is already written: *"Offering the target is what lets the world say no; hiding it is what
made the refusal impossible to reach"* (`beathandler.go:373-377`). *Acceptance:* the transgression is
always expressible and always answered — by a person, a lock, or G2's stated reason. The engine never
learns what the rule *is*, only that a door is shut and someone is standing in front of it, which is
why it works identically in a fantasy court and an alien hive.

**G4 — The world has a near future, authored from the brief.** `pending_event` is deployed, read every
clock crossing (`ledger.go:122-220`), and written by **nothing but tests**
(`core/db/tests/101_personality_world_test.sql:58` plus two `_test.go` files). Genesis should be its
first production writer: **one authored thing about to happen, in every world** — a shift change, a
tide, a delivery, a debt due — as a `when` **class** (the engine assigns `fire_at_tick`, exactly as
`extent_class` becomes metres, `prd_world_creation.md:70`) with a `{canonical_name, attempt}` payload
the commit resolves to ids (`ledger.go:12`). It is not a norm mechanism and must not be typed as one;
it makes a world *ongoing*. Where a norm exists, this is the cheapest place it becomes visible, because
the player sees it applied **to someone else** — no codex, no exposition, no transgression needed.
*Acceptance:* in every world, inside the 5-beat window the PRD already measures
(`prd_world_creation.md:22`), the player perceives one authored future event they did not cause.

**G5 — A collective, when implied, is a page you fill in, and its name is earned.**
`fn_compendium_index` is already kind-parameterised (`schema.sql:1296-1304`): one handler registration
beside `main.go:45-50` plus a page function yields a surface accumulating *contradictory* accounts with
epistemic framing for free (`:1194-1206`). **Zero collectives is the normal case** and must cost
nothing — a two-person world authors none, and a validator expecting one is the taxonomy re-entering
through the back door. Consequence the other seats will miss: `fn_unearned_names`'s `unearned` CTE has
**no kind filter** (`:2924-2939`), so a collective's name falls behind the naming wall — the player
hears "the ones in the grey sash" until told the word. Correct and better, but a product decision, and
it makes the collective's `descriptor` load-bearing as a first sight.

**G6 — Measure play, not payloads, across unlike worlds.** `DREAMCHAT_BRIDGE=fake` proves shape, never
behaviour. The eval is a scripted 5-beat run over **N briefs whose implied norms share no shape** — an
order, a debt, a forbidden place, a duty rota, a non-human protocol — **plus a negative control
implying no norm**, asserting the same four observables: one NPC act consistent with the authored norm,
one refusal with a stated cause, one compendium item, one authored future the player did not cause. The
control must author no collective and cost no extra tokens. Sampled and human-audited exactly as
`I-3`/`I-6` already are (`07_test_and_invariant_spec.md:18,26-27`), never a CI equality. **If the
fixtures are all hierarchies, the harness is the ontology.**

## 3. Three hardest attacks, pre-answered

**simarch: "G1 is prose pretending to be a system — a `manner` string enforces nothing."** Correct; I
claim felt consequence, not enforcement, and that is the cheaper product. The only enforcement this
engine can do is physical (`schema.sql:161-218`), and `tier1.go:3` states the law for extending it: a
Tier-1 key grows only when code checks it. A permission predicate cannot even be written generically —
it must know *what* confers permission, the one thing the service is forbidden to learn
(`prd_world_creation.md:175-180`). Meanwhile the NPC channel is where players attribute meaning: a mind
that refuses to be addressed and steps between you and the person you came for *is* the rule, committed
through the same no-bypass pipeline (`FINAL-world-npc-cognition.md:12`).

**extraction: "you rejected `relations[]` while G1 needs that structure, and 'per holder' is as
unvalidatable as what you attack."** (1) I reject it *as targeted*: `relationship_state`
(`schema.sql:3911-3918`) is read by **zero** lines of `core/api`, and the `[RELATIONSHIPS]` block the
context spec promises (`06_context_assembly_spec.md:76,88`) does not exist in `cognitionprompt.go`. A
field whose only consumer is a doc is depth the player cannot feel — render the block and I vote for
it. (2) My criteria are checkable where "did the model infer the social system" is not: "two holders
differ in content and `epistemic_type`" is a SQL assertion over any world, and G4's authored future is
an observed perception. Neither names a structure.

**ux: "G4 and G5 add asking, and G5 hides the word the user typed."** G4 asks nothing — the near future
is inferred in the same call, and the interview's own law forbids slot-filling
(`world_interview.txt:4,7`). On G5, the naming wall already does this to every person the founder names,
deliberately (`prd_world_creation.md:25`, `schema.sql:2994`). Concession: the confirmation surface
should show the inferred rule and who upholds it in the world's own words — a user who cannot see the
inference cannot correct it. A review screen, not another question.

## 4. What I would cut

- **Every identifier naming a social structure.** No `caste`, `rank`, `tier`, `faction_kind`,
  `authority`, or `member_of` enum in code, prompt, schema key or fixture name. The founder's example
  is content; the moment it becomes vocabulary, `prd_world_creation.md:177-180` is broken and the
  sandbox is a genre engine.
- **Any acceptance criterion only a hierarchical world can pass** — including two I wrote in my own
  first draft ("the two lowest-standing cast members…", "in a caste world…"). Fixtures that are all
  hierarchies teach the service what a world usually contains as effectively as an enum, and hide it in
  CI rather than in the contract.
- **`relationship_state` stance matrices** until `cognitionprompt.go` renders a relationships block.
  Unread state is not depth.
- **Any Tier-1 permission/access key or numeric "authority".** `tier1.go:3` and GA-2 refuse it; no
  consumer exists.
- **Anything depending on threshold accumulation** ("suspicion rises, the watch turns"): designed at
  `07_test_and_invariant_spec.md:78-79`, and there is no `threshold_ledger` table in `schema.sql`.
- **A second LLM seat for "systems extraction".** One call, no repair loop, is a deliberate cost
  decision (`worldgenesis.go:172-174`) against p50 ≤ $0.25 / p95 ≤ 180 s
  (`prd_world_creation.md:26-27`).
- **A rules/codex panel.** The compendium is the surface (G5); a rules screen tells the player what
  they should have discovered.
- **`places[].kind` and `cast[].standing` as they stand.** Wire (G0) or delete: a field only the schema
  believes in is worse than none, because the seat spends tokens authoring depth we drop.
