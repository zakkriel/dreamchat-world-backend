# Synthesis — World Creation Depth debate

> **Authorship and standing:** written by seat **ux**, not a neutral moderator. All four Round 1
> papers now exist (`gamedesign` filed late, after Round 2 and the first synthesis draft); no peer
> filed Round 2, so peer positions are engaged through `round2_ux.md` (including its gamedesign
> addendum). Resolutions one seat argued unanswered are marked **[unrebutted]** — reopenable with
> evidence, burden on the reopener.

## 1. The one finding all four seats hit independently

`cast[].standing` — the only field in `world_genesis/1` encoding "where they sit in this world's
order" (`core/api/schema/world_genesis.v1.schema.json:130-134`) — is schema-required (`:117`),
refused when blank (`core/api/worldgenesis.go:310-311`), and **written nowhere**: no commit path in
`worldgenesiscommit.go` references it. All four papers found this without coordination
(`round1_simarch.md:30-36`, `round1_extraction.md:12-17`, `round1_ux.md` §1,
`round1_gamedesign.md:34-38`). Gamedesign extended the indictment: `arrival.why` labels a choice
button and is never persisted (`worldgenesishandler.go:534,661`; `world_character` stores
descriptor + name only, `worldgenesiscommit.go:201`), and `places[].kind` is dropped while
objects' `kind` reaches state (`:677`).

This is the founder's complaint made mechanical: the pipeline already pays the model to author
depth and discards it before canon. Everything below is downstream of one rule — **a fact that is
authored must be committed, a fact that is committed must be consumed, and a fact that is
consumed must reach a surface the player can feel** (ux's "shown ⇒ committed ⇒ consumed" gate
composed with gamedesign's four felt surfaces, `round1_gamedesign.md:5-23`).

## 2. The agreed architecture

Four layers, each owned by the seat that proposed it, none contested after the full exchange:

1. **Representation (simarch M1–M2 ⊇ extraction M1–M2):** optional `orders[]`
   (`{descriptor, canonical_name, standing_over[], norms[], legibility}`) plus `cast[].belongs_to`,
   committed as `entity_registry` rows with `entity_kind='group'` — legal today with zero DDL
   (`schema.sql:3699-3710` has no kind CHECK; `event_participant` admits the kind at `:3722`).
   Norms commit as **group-held `public` perceptions** grounded in a backstory event
   (`AttributeChanged`, no mutations — `worldgenesiscommit.go:404-407`), never the `world_genesis`
   naming event (`fn_perceived_name` would read them as names, `schema.sql:2584-2599`). Common
   knowledge flows through `fn_visible_perceptions` (`schema.sql:3080-3086`) with no new read
   path. **Per-holder variance rides `history[].knowledge`** (gamedesign G1): low-standing holders
   carry *different* beliefs about the same rule with different `epistemic_type`s.
2. **Consumption (simarch M3 + gamedesign G1):** the cognition prompt renders each decided-for
   mind's order and bearing norms on the stable cache prefix (`cognitionprompt.go:12-16`); roster
   lines carry membership only as the viewer perceives it. Falsifiable: A/B one brief with and
   without `orders`; no behavioral difference ⇒ cut (`round1_simarch.md:120-121`).
3. **Confirmation (ux M1, conceded by all three peers):** the brief's inferred orders, norms —
   and gamedesign's scheduled demonstration — render pre-genesis as strikeable world-language
   statements; amendments travel as `InterviewAnswer` rows into the ANSWERS block
   (`worldgenesis.go:164-169`, `world_genesis.txt:2`). "Build now" always live; fast lane stays
   one tap. Gamedesign closed the loop in its own words: *"a user who cannot see the inferred
   system cannot correct it. That is a review screen, not another question"*
   (`round1_gamedesign.md:146-148`).
4. **Felt surface (gamedesign G2–G5):** refusals state their deterministic cause
   (`narrateprompt.go:227-228` is today's silent failure); genesis becomes `pending_event`'s first
   production writer so the player *watches the rule applied to someone else* within the first
   beats (`ledger.go:122-220`; today only tests write it); the collective gets a compendium page
   via the kind-parameterised index (`schema.sql:1296-1304`) and its name is behind the naming
   wall (`fn_unearned_names` has no kind filter, `schema.sql:2924-2939`) — "the ones in the grey
   sash" until someone speaks the word.

Constraint compliance (briefing's law): no genre nouns or relation-kind enums (GA-2/GA-3); no
numbers from any seat; norms are perceptions citing accepted events with closed-enum epistemics;
player perception count at arrival stays exactly 1 (B-4/AC-4, simarch M6); genesis remains one
seat call — depth is output tokens, the playback adds one interview-sized turn (cost constraint 6);
group rows and perception rows are tables a hand-written world could use (indistinguishability).

## 3. Disputes and resolutions

| # | Dispute | Positions | Resolution | Status |
|---|---|---|---|---|
| 1 | Where relational facts live | extraction M3: `relations[]` → `relationship_state.attrs`, "wired later" (`round1_extraction.md:109-110`) vs. simarch M2 + gamedesign | **Against extraction, twice over.** "Committed but unread" repeats the `standing` failure (`round2_ux.md` E-2); gamedesign adds that `relationship_state` is read by zero lines of `core/api` and the promised `[RELATIONSHIPS]` block (`06_context_assembly_spec.md:76,88`) is unrendered — *"Wire the block, then I will vote for it"* (`round1_gamedesign.md:136-137`). Deferred until a reader exists. | resolved, 3 seats aligned |
| 2 | Where correction happens | simarch: "before commit, in the `choice` frame… ux's problem" (`round1_simarch.md:128-130`) | **Timing error.** The `choice` frame renders after the world commits (`world_genesis_frame.v3.schema.json:4`; durable-worlds decisions 1–2). Post-commit mutation is vetoed by simarch himself; gamedesign independently conceded the review screen. Pre-genesis playback is the only seam. | resolved, 3 seats aligned |
| 3 | Interview changes | extraction M5: "zero prompt changes" (`round1_extraction.md:62-67`) vs. ux M3/M4 | **Split.** No new question *type* (extraction wins; ux conceded; gamedesign's G4 also asks nothing). But the silent seat-error → `Done:true` collapse (`worldinterview.go:71-84`) must become distinguishable, else M5 is unfalsifiable (`round2_ux.md` E-1). | [unrebutted] |
| 4 | Membership legibility | simarch M3(b) renders membership per-viewer with no authored input; gamedesign G5 calls the group descriptor "load-bearing as a first sight" (`round1_gamedesign.md:111`) | **Gap in both.** Visible-vs-concealed rank is an authored per-order fact the user confirms in playback; it gates roster rendering *and* whether a group descriptor may honestly render at first sight — a concealed order with a first-sight descriptor leaks what the fiction says strangers cannot see (`round2_ux.md` S-2, G-A3). | [unrebutted] |
| 5 | Norm shape: common knowledge vs. per-holder belief | simarch M2 (one group-held `public` row) vs. gamedesign G1 ("never as world prose"; different beliefs per holder, `round1_gamedesign.md:63-72`) | **Both, layered.** The group-held row is the common-knowledge baseline (what "everyone knows"); per-holder variance is authored in `history[].knowledge`, which already expresses it. Not a conflict — a baseline plus deviations, mirroring how the epistemic enum already splits `public` from `told`/`rumor`. | resolved by composition |
| 6 | Ordering | gamedesign: "behaviour first… Every rule pushed into the gate before `narrateprompt.go:228` is fixed makes the product worse… Without G2, nothing else is worth building" (`round1_gamedesign.md:49-52,80`) | **Partially rejected.** G2 is real and adopted, but its own G3 ("enforced by bodies and locks, not permissions") means cognition-delivered depth produces narratable NPC acts, not gate rejections — the unexplained-refusal failure only fires on the physical-access path. G2 gates the access surface, not the PRD (`round2_ux.md` G-A1). | [unrebutted] |
| 7 | Enforcement representation | all four | **Converged.** No enforcement schema, no engine sanction pass, no Tier-1 growth, no viewer-aware `fn_portal_permits`: enforcement is a `history[]` beat (extraction M6), required once by the belt (simarch M5), demonstrated once in the near future (gamedesign G4), elicited when open (ux M3). "Laws are enforced by people, which is what makes breaking one a story instead of a 403" (`round1_simarch.md:148`). | agreed, 4 seats |

## 4. The reconciled recommendation list

Supersedes `round2_ux.md` §3 (written before gamedesign filed; its addendum records the deltas).

1. **`orders[]` + `cast[].belongs_to`**, committed as `entity_kind='group'` registry rows; `minItems: 0`. *(AC: caste brief ≥1 group row; plain brief zero, byte-identical path.)*
2. **Norms as group-held `public` perceptions off a backstory event**, plus per-holder variant beliefs in `history[].knowledge`; no `relationship_state` writes until a reader exists. *(AC: all norms visible to the player at arrival with player perception count = 1; in a caste world the two lowest-standing members hold different beliefs about the same rule with different epistemic types, neither `direct` for a rule only heard — gamedesign G1's assertion, SQL-checkable.)*
3. **Wire-or-delete, generalized (G0):** every non-numeric `world_genesis/1` leaf reaches a table read by a prompt or payload, or leaves the schema — `standing`, `arrival.why`, `places[].kind` included. *(AC: CI test asserts the property for every leaf.)*
4. **Cognition renders orders + bearing norms**, with the A/B kill-switch. *(AC: NPC decisions differ within five beats or the mechanism is cut.)*
5. **Membership legibility is authored** and gates both roster rendering and first-sight group descriptors. *(AC: visible/hidden-rank briefs produce differing roster lines, differing playback statements, and a hidden-rank group's descriptor renders at first sight nowhere.)*
6. **Pre-genesis playback** of inferred orders, norms and the scheduled demonstration — strikeable statements, amendments into ANSWERS, Build-now always live, statements only over consumed fields. *(AC: amendments reach the genesis prompt verbatim; struck content absent from canon; fast lane stays one tap.)*
7. **Refusal states its cause (G2):** the NOTHING RESOLVED render carries the deterministic obstacle already computed, in the fact vocabulary the narrator answers questions from (`narrate.txt:33`). Dependency of the access surface. *(AC: a move at a locked way produces narration naming the obstruction; zero new seats.)*
8. **Genesis writes one scheduled demonstration (G4):** first production writer of `pending_event`, `when` as a class, payload by canonical name; visible in playback. *(AC: in a caste world the player perceives one enforcement they did not trigger within five beats; the demonstration appeared as a strikeable playback statement.)*
9. **Belt refusals:** references resolve, `standing_over` acyclic, every norm binds a reachable member, ≥1 history beat enforcing/breaking a norm. *(AC: each malformed class refuses with a stated reason and a captured fake payload in CI.)*
10. **Group compendium page (G5):** one handler registration beside `main.go:45-50` on the kind-parameterised index; the group's name is earned like any person's. *(AC: a caste world's group page accumulates contradictory epistemic-framed accounts; the canonical group name is unreachable until told.)*
11. **Interview distinguishes "nothing to ask" from "could not ask."** *(AC: seat error renders a retryable state, not silent done.)*
12. **Kickstart candidates are positions in the system** when orders exist. *(AC: three `why` fields reference different standings, audited.)*
13. **Two-layer eval:** extraction's plant-and-measure (N=20, 0% invention, <10% missed structure, playback traceability) plus gamedesign's scripted 5-beat transgression with behaviour targets (NPC act citing the norm, stated-cause refusal, compendium item, unprompted enforcement) — sampled and human-audited per I-6's methodology. *(AC: both run; behaviour targets are audit findings, never CI equalities.)*

**Sequencing** (dispute 6's resolution applied): (a) recs 1–4 + 9 (substrate: schema, commit,
cognition, belt); (b) recs 5–6 + 11–12 (authoring surfaces); (c) recs 7–8 + 10 (felt surfaces:
refusal cause, demonstration, discovery); (d) rec 13 alongside all. Gate enforcement: never, in v1
— by four-seat agreement.

## 5. What remains open

- **Round 2 rebuttals were never filed by peers.** Disputes 3, 4 and 6 stand [unrebutted];
  disputes 1, 2, 5, 7 are multi-seat aligned and effectively closed.
- **`relations[]` re-entry condition** is now concrete: render the `[RELATIONSHIPS]` block the
  context spec promises (`06_context_assembly_spec.md:76,88`), then extraction's shape earns its
  vote (gamedesign's own condition).
- **Playback statement ceiling** — bound by cost, set empirically from first builds
  (`prd_world_creation.md:83` posture).
- **SPEC ids** — assigned by the implementing chunk (D-5): at minimum the `world_genesis/2` bump,
  the playback turn contract, the pending-event genesis writer, and the group compendium route.
