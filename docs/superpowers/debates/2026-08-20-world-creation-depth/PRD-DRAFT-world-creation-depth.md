# PRD Amendment (DRAFT): World Creation Depth — the brief's implied system becomes operative structure

> **Status:** Draft v2 (2026-08-20), staged in the debate directory pending promotion to
> `docs/10_prds/`. Produced from the full four-seat adversarial debate in this directory
> (`SYNTHESIS-ux.md` is the consolidated record; v2 incorporates gamedesign's late filing).
> **Amends:** `docs/10_prds/prd_world_creation.md` — extends, never re-litigates. Where this
> draft and the baseline PRD disagree, the baseline wins unless a criterion below explicitly
> amends it. The engine set stays read-only input; gaps become SPEC entries, never workarounds.
> **Depends on:** everything the baseline depends on, plus the shipped split flow
> (`world_genesis_frame/3`, durable-worlds design 2026-08-21).

## 1. Problem Statement

**The problem:** a brief that states a *system* is committed as *flavor*. "The world is heavily
ruled by a social caste called Alphas" implies a caste taxonomy, power asymmetry, enforcement,
player-facing consequence and per-caste epistemics — and today's pipeline flattens all of it into
prose fields nothing downstream can join on. The proof is mechanical, and it is not one field:

- `cast[].standing` — "where they sit in this world's order"
  (`core/api/schema/world_genesis.v1.schema.json:130-134`) — is schema-required (`:117`), refused
  when blank (`core/api/worldgenesis.go:310-311`), and written by no commit path: the only
  consumers of a `genesisActor` are `registerEntities` (descriptor + name,
  `worldgenesiscommit.go:292-321`) and `insertMind` (traits, speech_manner, hiding, `:516-557`).
- `arrival.why` — the player's premise — labels a choice button
  (`worldgenesishandler.go:534,661`) and is never persisted; `world_character` stores descriptor
  and canonical name only (`worldgenesiscommit.go:201`).
- `places[].kind` is dropped; only objects' `kind` reaches state (`worldgenesiscommit.go:677`).

The pipeline pays the model to author the world's social order and discards it before canon.

**A second, coupled defect:** when the world *does* refuse an act, it refuses silently. A
`gate_reject` with an empty delta renders one line — "NOTHING RESOLVED: the attempted action did
not happen" (`narrateprompt.go:227-228`) — with no cause; `halt_reason` reaches the client as a
bare machine string (`beat_frame.v5.schema.json:204-206`). Depth delivered through hard refusal
without stated cause reads as a broken game, not a caste system.

**Impact hypothesis:** the difference between a world that *mentions* Alphas and a world *ruled
by* them is the set of facts that reach a surface the player can feel: what a mind decides, what
the world refuses (and says why), what the compendium accumulates, what arrives unprompted. The
existing proof this shape works is `tension`: authored class → engine number → beat-budget
mechanic → visible tone (`world_genesis.v1.schema.json:327-331` → `tension.go:28-45` →
`scenehandler.go:213-216`). Class in, mechanic out, visible surface — or it is a brochure.

**Why the mechanism exists already:** the engine's substrate is deployed and unused:
`entity_registry` accepts `entity_kind='group'` with no CHECK (`schema.sql:3699-3710`, kinds at
`:3722`); `fn_visible_perceptions` makes group-held perceptions visible to every viewer
(`schema.sql:3080-3086`); `pending_event` is read every clock crossing (`ledger.go:122-220`) and
written today by nothing but tests; the compendium index is kind-parameterised
(`schema.sql:1296-1304`). Zero DDL is required.

## 2. Goals & Success Signals

**Goal:** a one-line brief implying a social system yields a world in which that system is
committed structure — groups that exist, laws that are common knowledge with per-holder variance,
minds that act on them, refusals that state their cause, and one demonstration the player watches
— which the user saw and corrected before spend.

| Signal | Target | Measurement |
|---|---|---|
| Implied structure extracted, not invented | 0% invented groups; <10% missed structure | N=20 plant-and-measure briefs (fake + sampled live audit, I-6 methodology) |
| Law is common knowledge without costing the player's ignorance | all norms visible; player perception count exactly 1 | at arrival: `fn_visible_perceptions(world, player)` returns every norm; player's own `perception_record` rows = 1 (baseline AC-4 preserved) |
| Beliefs vary per holder | different beliefs, different epistemics | in a caste world, the two lowest-standing cast members hold different beliefs about the same rule with different `epistemic_type`s; neither `direct` for a rule only heard (SQL assertion) |
| Depth changes play, or it is cut | NPC decisions differ in first 5 beats | A/B one brief with and without `orders[]`; no difference ⇒ mechanism removed |
| The rule is demonstrated, not explained | 1 unprompted enforcement within 5 beats | the player perceives one application of the rule they did not trigger (scheduled via `pending_event`) |
| The user owns the inferred law | 100% of playback amendments honored | amended statements reach the genesis ANSWERS block verbatim; struck content appears in no order, norm or scheduled demonstration |
| Fast lane friction unchanged | one tap with defaults accepted | brief → playback (defaults) → build adds one interaction, zero when no statements derive |
| Cost ceiling holds | p50 ≤ $0.25/world, p95 ≤ 180s (baseline §2) | genesis stays one seat call (`worldgenesis.go:172-175`); playback adds one interview-sized turn |
| No authored field the player cannot feel | 0 | CI: every non-numeric `world_genesis/1` leaf reaches a table read by a prompt or payload |

## 3. Scope / Non-Goals

**In scope:** `orders[]` (with legibility) and `cast[].belongs_to`; norms as group-held `public`
perceptions plus per-holder variance in `history[].knowledge`; cognition-prompt rendering; the
pre-genesis playback; stated-cause refusals; one scheduled demonstration per world via
`pending_event`; the group compendium page; belt refusals; the interview honesty state; systemic
kickstart candidates; the two-layer eval.

**Non-goals — decided in debate, with the deciding argument:**
- **No relation-kind or group-kind enum.** A closed list of kinds is the service learning what
  worlds usually contain — the GA-2 trap. "Caste" appears only in prose the user typed.
- **No numeric authority tier, access level, or Tier-1 key.** Baseline AC-7 forbids seat numbers
  (`prd_world_creation.md:70`); `tier1.go:3` grows only for code checks; no consumer exists.
- **No `relationship_state` writes at launch.** Read by zero lines of `core/api`; the context
  spec's `[RELATIONSHIPS]` block (`06_context_assembly_spec.md:76,88`) is unrendered. Re-entry
  condition: wire the block first.
- **No engine-side norm enforcement.** No sanction pass, no rule engine, no viewer-aware
  `fn_portal_permits`. Caste access is enforced by bodies and locks — a `locked` way whose key an
  authored person carries makes entry a social problem, which is the game. Laws enforced by
  people are stories; laws enforced by the gate are 403s.
- **No per-world rules table, no DDL.** Group rows, perceptions and the pending ledger exist and
  are read; new tables make generated worlds distinguishable from hand-written ones (baseline §9).
- **No post-commit correction.** Canon is append-only; the world commits when authoring ends
  (`world_genesis_frame.v3.schema.json:4`); the correction surface is pre-genesis or nothing.
- **No dedicated "social structure" interview question type.** "Ask what changes the world most"
  (`prompts/world_interview.txt:3`) selects the enforcement question when open; a fixed type is
  slot-filling (`world_interview.txt:4`).
- **No rules/codex panel.** The compendium is the discovery surface; a rules screen tells the
  player what they should have discovered and kills the discovery it documents.
- **No mechanisms depending on threshold accumulation.** Designed on paper
  (`07_test_and_invariant_spec.md:78-79`) but no `threshold_ledger` exists in `schema.sql`; do not
  design play against undeployed machinery.
- **No second authoring seat.** One genesis call, no repair loop, is a deliberate cost decision
  (`worldgenesis.go:172-174`). Depth arrives as output tokens; the playback turn is
  interview-sized, not an authoring pass.

## 4. Acceptance Criteria

1. **Orders are entities.** `world_genesis/1` gains optional `orders[]` —
   `{descriptor, canonical_name, standing_over[], legibility, norms[]}` — and `cast[].belongs_to`;
   `minItems: 0`. Each order commits as an `entity_registry` row with `entity_kind='group'`
   exactly as places register (`worldgenesiscommit.go:305-313`), so `fn_display_name`'s wall
   covers it. A brief implying no order emits none; every existing path is byte-identical.
2. **Norms are common knowledge with provenance; beliefs vary per holder.** Each norm
   `{stated, bearing, sanction}` commits as a `perception_record` held by the order's group
   entity, `epistemic_type='public'`, citing a backstory event (the no-mutation
   `AttributeChanged` shape, `worldgenesiscommit.go:404-407`) — never the `world_genesis` naming
   event, which `fn_perceived_name` reads as a name (`schema.sql:2584-2599`). Per-holder variant
   beliefs about the same rule are authored in `history[].knowledge` with differing
   `epistemic_type`s. At arrival, `fn_visible_perceptions(world, player)` returns every norm and
   the player's own perception count is exactly 1 (baseline AC-4 unchanged — this is why the
   group-holder seam is the right one).
3. **Wire or delete — no authored field survives that no surface renders.** Every non-numeric
   `world_genesis/1` leaf reaches a table read by a prompt or a payload, or leaves the schema:
   `cast[].standing` becomes `belongs_to` or is deleted; `arrival.why` lands on `world_character`
   and the narrator's YOU ARE block or is deleted; `places[].kind` likewise. CI asserts the
   property for every leaf.
4. **The law reaches the minds, with a kill-switch.** `buildCognitionPrompt` renders each
   decided-for mind's order and bearing norms, riding the stable cache prefix
   (`cognitionprompt.go:12-16`). Roster lines carry membership only as the viewer perceives it —
   an unguarded roster field is a naming-wall breach by another door. Acceptance is empirical:
   one brief built with and without `orders[]` produces differing NPC decisions within five
   beats, or the mechanism is cut.
5. **Membership legibility is authored and load-bearing twice.** Each order states whether
   membership is visible on sight or concealed. It gates (a) roster rendering and (b) whether the
   group's descriptor may render as a first sight at all — a concealed order surfacing a
   first-sight descriptor leaks what the fiction says strangers cannot see. Two briefs differing
   only in visible/hidden rank produce differing roster lines, differing playback statements, and
   no first-sight group descriptor in the hidden case.
6. **The playback: see and correct the inferred law before spend.** After the brief (both lanes),
   before genesis, the brief's entailed orders, norms and the scheduled demonstration (AC-8)
   render as strikeable world-language statements — the constitution, never the plot; no cast
   secrets, no history (AC-7 secrecy boundary intact). Amendments travel as `InterviewAnswer`
   rows into the ANSWERS block, where the user's words outrank the seat (`world_genesis.txt:2`).
   "Build now" always live; all statements pre-accepted; a statement renders only if it traces to
   a committed, consumed field (shown ⇒ committed ⇒ consumed). Fast lane with defaults is one tap.
7. **Refusal states its cause.** The NOTHING RESOLVED render (`narrateprompt.go:224-228`) carries
   the deterministic obstacle already computed — the portal's `locked`, the absent listener, the
   encumbrance — in the fact vocabulary the narrator already answers questions from
   (`narrate.txt:33`). A move at a locked way produces narration naming the obstruction; zero new
   seats, zero new calls. This is a dependency of the *access* surface (locked ways as caste
   boundaries), not of the creation-side criteria above.
8. **The rule gets a scheduled demonstration.** Genesis becomes `pending_event`'s first
   production writer (today: tests only): one authored near future per world — the tithe
   collected, the inspection walked — expressed as a `when` class (the engine assigns
   `fire_at_tick`, the `extent_class` pattern) with a `{canonical_name, attempt}` payload the
   commit resolves to ids (`ledger.go:12`). The demonstration appears in the playback as a
   strikeable statement naming the practice, never the scene. Within the baseline's 5-beat window
   (`prd_world_creation.md:22`), the player perceives one enforcement they did not trigger.
9. **The belt refuses malformed law.** `validate()` gains: every `belongs_to`/`standing_over`
   reference resolves; `standing_over` is acyclic; every norm's `bearing` names an order with at
   least one member starting in a place reachable from `arrival.place`; at least one `history[]`
   entry records a norm enforced or broken. Each refusal class has a stated reason and a captured
   fake-driver payload in CI (baseline AC-13 discipline).
10. **The collective is a compendium page and its name is earned.** One handler registration
    beside `main.go:45-50` on the kind-parameterised index (`schema.sql:1296-1304`). Because
    `fn_unearned_names` has no kind filter (`schema.sql:2924-2939`), the group's canonical name is
    behind the naming wall: the player hears the descriptor until someone speaks the word, and
    the page accumulates contradictory epistemic-framed accounts (`schema.sql:1194-1206`).
11. **The interview cannot silently fail.** The turn shape distinguishes "nothing genuinely open"
    from "could not author the next question" (today both collapse to `Done: true`,
    `worldinterview.go:71-84`); the surface renders the latter as a retryable state. No new
    question type, no prompt spine change.
12. **Kickstart candidates are positions in the system.** When `orders[]` is non-empty, the three
    `arrival_candidates` differ by standing relative to the authored orders — under it, above it,
    outside it — audited in AC-13's harness. Existing candidate rules unchanged
    (`world_genesis.txt:27`).
13. **The eval measures payloads AND play.** Layer 1: N=20 planted briefs under
    `DREAMCHAT_BRIDGE=fake` plus sampled live audit — 0% invented groups, <10% missed structure,
    every playback statement traces to a brief phrase or amendment; the fake genesis driver emits
    ≥1 order and ≥1 norm so shapes are CI-captured from day one. Layer 2: a scripted 5-beat
    transgression on a planted brief asserting one NPC act citing the norm, one refusal with
    stated cause, one compendium item about the collective, one unprompted enforcement — sampled
    and human-audited per I-6's methodology, behaviour targets never CI equalities.

## 5. Open Questions

1. **When does `relations[]` (pairwise, asymmetric) earn its way in?** Concrete re-entry
   condition from the debate: render the `[RELATIONSHIPS]` block the context spec promises
   (`06_context_assembly_spec.md:76,88`); then extraction's shape gets its vote.
2. **Playback statement ceiling.** Bound by cost, not shape — set empirically from the first
   builds (`prd_world_creation.md:83` posture).
3. **Does the scheduled demonstration repeat?** One `pending_event` per world at genesis is the
   committed scope; whether the world's turn re-arms demonstrations in play belongs to the beat
   loop's owners, not this PRD.
4. **Which SPEC entries does this file?** At minimum the `world_genesis/2` bump, the playback
   turn contract, the pending-event genesis writer, and the group compendium route. Assigned by
   the implementing chunk, not pre-assigned (D-5).

## 6. Sequencing

Four waves, from the debate's resolved ordering dispute (`SYNTHESIS-ux.md` §3 #6):
**(a) substrate** — AC-1…4, 9 (schema, commit, cognition, belt);
**(b) authoring surfaces** — AC-5, 6, 11, 12 (legibility, playback, interview state, candidates);
**(c) felt surfaces** — AC-7, 8, 10 (stated-cause refusal, demonstration, discovery);
**(d) eval** — AC-13 alongside all.
Gate enforcement: never, in v1 — by four-seat agreement. The ordering principle stands:
the interview and the playback need somewhere to put the answer before they are worth asking with,
and every rule must reach a felt surface before any rule reaches the gate.
