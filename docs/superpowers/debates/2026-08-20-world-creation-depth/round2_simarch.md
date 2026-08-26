# Round 2 — simarch (Systems/Simulation Architect)

## 0. Retraction, first, because it is mine

My R1 **M1 proposed `orders[]` with `standing_over[]` — "a DAG the belt validates acyclic".** That is
a hierarchy ontology. An alien tidal ecology, a shared dream, a two-person hospice, a world whose only
organising fact is a rota, or a world with no collectives at all cannot fill that field, and a schema
key nobody can decline bends every world toward the one that can. Same crime as a genre enum, one
abstraction up. **Retracted whole**, along with `belongs_to` and the `stated/bearing/sanction` triple
(`bearing` presumes rules bind *groups*).

gamedesign wrote the rule before I broke it: *"an acceptance criterion only a hierarchical world can
satisfy is itself a GA-2/GA-3 violation"* (`round1_gamedesign.md:9-10`), and applied it to their own
draft (`:166-169`). I did not apply it to mine.

What survives is the **channel, not the shape**: `fn_visible_perceptions` (`core/db/schema.sql:3080-3086`)
makes any perception held by a `faction|group` entity visible to every viewer. That mechanism knows
nothing about hierarchy, collectives or genre — it is B-2's *"or common knowledge (presumed ambient/
cultural knowledge of the world)"* (`docs/00_strategy/06_rules_register.md:27`) with a live
implementation. One untyped array of world-language sentences, committed there. Nothing else.

## 1. Attacks

### extraction — writes to a table with no reader, then forbids exactly that

*"Commits into `relationship_state.attrs` (schema.sql:3911-3918)"* (`round1_extraction.md:50`), against
its own cut: *"No new engine table. `relationship_state` and `group`/`faction` entity kinds already
exist, deployed, unused. Building anything new before wiring the existing substrate is waste I'd fight
in round 2"* (`:138-140`). **Verified:** `grep -rn relationship_state core/api | wc -l` → **0**.
Deployed-and-unused is not substrate, it is a table. The handoff at `:92-99` — *"I'm handing simarch
the substrate, not claiming to be the gate"* — hands it to a gate I refused to build
(`round1_simarch.md` M4), so `relations[]` lands nowhere on either side. Cut it.

**Citation error.** `:35-36` claims `entity_kind='group'` is *"already legal (schema.sql:3722)"*.
Verified: `:3722` is `event_participant_entity_kind_check`, a CHECK on **event_participant**.
`entity_registry` (`schema.sql:3699-3710`) has **no** `entity_kind` CHECK at all. Right conclusion,
wrong table — and it matters, because the actual reason `group` means anything is
`fn_visible_perceptions:3084`, which extraction never cites.

**Substantive danger.** M2 commits `member_of` *"as `event_participant` rows off the `world_genesis`
naming event ... with `role_qualifier='member'`"* (`:40-43`). Membership is not participation in an
event, and that event is booby-trapped: `fn_perceived_name` reads every `world_genesis`-sourced,
subject-linked perception as that entity's NAME (`schema.sql:2584-2599`) — the bug that rendered an
archivist's forgery scheme where her name belonged (`core/api/worldgenesiscommit.go:550-555`).
`event_participant` also drives `perception_subject` about-ness in the ruled path
(`20260724100002_apply_ruled_event.sql:213-221`). Same class of mistake, longer fuse.

And after the founder's correction: `groups[]` with a required prose `standing` is my retracted
`orders[]` with the DAG filed off. `minItems: 0` does not save it — a named slot is an instruction to
fill it, which is why extraction needs M7's prompt guardrail *and* M8's invention harness to police a
field it invented.

### gamedesign — one verified-false fact, one wrong economics

**False, and it is their own "consequence the other seats will miss":** *"`fn_unearned_names`'s
`unearned` CTE has **no kind filter** (`:2924-2939`), so a collective's name falls behind the naming
wall — the player hears 'the ones in the grey sash' until told the word"* (`round1_gamedesign.md:119-122`).
The `unearned` CTE requires `fn_display_name(...) IS DISTINCT FROM er.canonical_name`
(`schema.sql:2937`), and `fn_display_name` is `COALESCE(fn_perceived_name, actor_state.descriptor,
artifact_state.descriptor, canonical_name)` (`schema.sql:1445-1453`) — **no branch for a group**, and a
collective carries no `actor_state`/`artifact_state` row. It falls through to `canonical_name`, the
predicate fails, the collective is never returned. Its name is speakable at tick 0. (The token half is
filtered to `entity_kind='actor'` at `:2963` too.) The product decision gamedesign asks for does not
exist — which helps G5, and warns everyone: a collective inherits **no** wall, so if you ever want one
you must author it.

**Wrong economics — the real disagreement.** G1: a norm *"becomes play only as (a) a trait on whoever
upholds it ... and (b) a perception held by whoever is bound by it"* (`:72-74`). Per-holder authoring
of an *ambient* fact is O(cast) tokens for one sentence and re-authors the same law N times with N
chances to drift — the copy-the-record shortcut B-7 exists to forbid (`06_rules_register.md:32`). B-2
names the correct path in the same register (`:27`) and `fn_visible_perceptions:3080-3086` implements
it. Write it once. Their *divergence* criterion (`:79-81`) is right and I adopt it — but it governs
what people believe about a **breach**, not the law. A law everyone knows is not two beliefs.

Everything else gamedesign wrote I verified and accept: `pending_event` written only by tests
(`ledger_test.go`, `orchestrator_worldtime_test.go`, `core/db/tests/101_personality_world_test.sql`);
NOTHING RESOLVED carries no cause (`narrateprompt.go:227-228`); `arrival.why` reaches only an option
label (`worldgenesishandler.go:661`) while `world_character` stores descriptor + canonical name
(`worldgenesiscommit.go:201`); `[RELATIONSHIPS]` specced (`06_context_assembly_spec.md:88`) and absent
from `cognitionprompt.go`.

### ux — M2's contract is unenforceable in M1's ordering

M1: *"After the brief (both lanes), **before the expensive genesis call**, one cheap seat turn renders
the brief's entailments"* (`:13`). M2: *"A playback statement may only be rendered if it traces to a
field that survives commit into a location a play-time seat reads"* (`:28`). These contradict.
Pre-genesis no document exists, so nothing can trace to a commit that has not happened; genesis is one
call with no repair loop by explicit cost decision (`worldgenesis.go:171-175`), so nothing re-checks
the built world against the confirmed statements. The user ticks *"Rank is visible on sight"* (`:17`)
and the seat may author a world where it is not, unnoticed. ux's own best sentence indicts it:
*"Confirming prose that evaporates at commit is worse than no confirmation"* (`:28`).

The fix costs nothing: **the document already exists at the `choice` frame.** Durable-worlds commits
world content the moment authoring succeeds, `player_entity_id` NULL, arrival still open
(`worldgenesiscommit.go:13-19`), and the frame already carries `question` + `options` + per-option
`implication` (`world_genesis_frame.v3.schema.json:79-89`; precedent `world_interview.v1.schema.json:48-52`,
which ux cites at `:34`). Show the law there, verbatim — authored content, so law 2 holds
(`world_genesis_frame.v3.schema.json:24-27`). What you cannot have is *correction* there: canon is
append-only (`prd_world_creation.md:38`). Correction belongs to the interview, where the user's words
already *"outrank your judgement completely"* (`prompts/world_genesis.txt:2`).

Where ux is right and **under-cited**: M4. `askNextQuestion` returns `{Done:true}` beside the error
(`worldinterview.go:72,77`) and the handler logs and drops it — *"the fault goes to the log"*
(`worldgenesishandler.go:155-159`) — so the wire body is byte-identical to a real "nothing worth asking"
(`:164-167`). Silent demotion to Fast lane. One boolean.

## 2. Concessions

1. `orders[]`/`standing_over` retracted (§0). gamedesign named the rule; I broke it.
2. My R1 said register a collective *"with descriptor and canonical name exactly as places are
   registered ... so `fn_display_name`'s wall covers it"*. Wrong: `fn_display_name` never reads
   `entity_registry.descriptor` (`schema.sql:1445-1453`). The comment asserting otherwise
   (`worldgenesiscommit.go:281-284`) is itself wrong, for locations too. Nothing walls a group.
3. gamedesign's G0 outranks my belt work. Validating fields we already drop is theatre.
4. gamedesign's G2 outranks everything I proposed. A rule met as an uncaused "NOTHING RESOLVED" is
   worse than no rule. I had it nowhere; it is now #2.
5. extraction is right that `traits/1` — closed shape, open vocabulary
   (`world_genesis.v1.schema.json:149-154`) — is the discipline. My triple violated it. Reduced to one
   field.
6. ux is right that AC-4 needs *restating*, not asserting: the hazard is any check written over
   `fn_visible_perceptions` rather than `holder_id`.

## 3. Final converged recommendations — ranked

1. **[HARD VETO]** No schema key, enum, prompt line, fixture or test may name a kind of social
   structure, or presume one exists — including my retracted `orders`/`standing_over` and extraction's
   `groups[]`/`member_of`/`relations[]`. *Acceptance:* `grep -rniE 'caste|rank|faction|guild|hierarchy|member_of|standing_over'` over
   `core/api/schema`, `core/api/prompts`, `core/api/*.go` returns only pre-existing hits, and every new
   criterion is satisfiable by a world with two rooms, two people and no collective.
2. **Refusal states its cause, in world** (gamedesign G2). *Acceptance:* a move at a `locked` way
   narrates the obstruction; the NOTHING RESOLVED branch (`narrateprompt.go:227-228`) carries the
   computed obstacle from the vocabulary the fact sheet already emits (`schema.sql:1743-1787`).
3. **[HARD VETO]** **Wire or delete** — no authored field survives that no table stores and no prompt
   reads (gamedesign G0). *Acceptance:* a test enumerates every `world_genesis/1` leaf and asserts each
   reaches a column some prompt or payload reads; `cast[].standing`, `places[].kind` and `arrival.why`
   land or leave in the same change.
4. **One untyped `common_knowledge[]` of world-language sentences — the entire depth schema change.**
   `{stated}`, `minItems: 0`, no `bearing`, no `applies_to`, no kind. *Acceptance:* the field is
   describable in one sentence whose example is not a hierarchy, and a brief implying no shared law
   authors zero entries at zero extra token cost.
5. **Commit it once, to the holder the engine already honours.** One `entity_registry` row per world,
   `entity_kind='group'`, one `public` perception per sentence, grounded in a backstory
   `AttributeChanged` (`worldgenesiscommit.go:404-407`) — **never** `world_genesis`
   (`schema.sql:2592`; `worldgenesiscommit.go:550-555`). *Acceptance:* `fn_visible_perceptions(world,
   any viewer)` returns every sentence at tick 0, and AC-4's check is rewritten to count `holder_id =
   player` rows (still exactly 1), not visible rows (`prd_world_creation.md:24,65`).
6. **The minds read it or none of this happened.** A `WHAT EVERYONE HERE KNOWS` section in the
   cognition prompt's **stable prefix** (`cognitionprompt.go:12-16,119-151`). *Acceptance:* a scripted
   beat over a brief with a shared law yields ≥1 NPC decision consistent with it; the identical brief
   with the sentences stripped does not — diffed, not asserted.
7. **Per-holder divergence governs the breach, not the law** (gamedesign G1, narrowed). *Acceptance:*
   where authored history records a law broken or enforced, two holders' `perception_record` rows
   differ in `content` and `epistemic_type` and no holder holds `direct` for what they were told —
   vacuous where no breach was authored.
8. **The world has one authored near-future** (gamedesign G4). Genesis becomes `pending_event`'s first
   production writer via a `when` **class** the engine resolves to `fire_at_tick`. *Acceptance:*
   inside the 5-beat window the PRD already measures (`prd_world_creation.md:22`), the player perceives
   one authored event they did not cause — in every world, law or no law.
9. **Show the law at the `choice` frame; correct it in the interview, never after.** No pre-genesis
   seat call — Fast lane is "the brief alone" (`prd_world_creation.md:109`). *Acceptance:* `choice`
   renders the committed sentences verbatim (`world_genesis_frame.v3.schema.json:24-27,79-89`); no
   response body carries the genesis document; no post-commit edit path exists
   (`prd_world_creation.md:38`).
10. **The interview stops lying about why it stopped** (ux M4). *Acceptance:* the error path is
    distinguishable on the wire from a genuine "nothing worth asking" (`worldinterview.go:72,77`;
    swallow at `worldgenesishandler.go:155-159`), so the surface can offer "try again" instead of
    silently demoting a Custom-lane user.

**Also vetoed, against live proposals:** `relations[]` → `relationship_state` (zero readers);
`groups[]` + `member_of` off the `world_genesis` event; viewer-aware `fn_portal_permits`; any Tier-1
registry growth (`core/api/tier1.go:3`); a second LLM seat anywhere in the lane; engine-side sanction
firing — the gate's floor stays structural (`20260724100002_apply_ruled_event.sql:20-27`). Laws are
enforced by people; that is what makes breaking one a story instead of a 403.
