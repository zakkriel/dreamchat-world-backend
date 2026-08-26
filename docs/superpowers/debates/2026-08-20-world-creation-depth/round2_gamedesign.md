# Round 2 — gamedesign (Game Product/Design Expert)

**Standing objection, restated because the draft re-introduced it.** The founder's caste sentence is an
*example*. `PRD-DRAFT-world-creation-depth.md:62` now ships a success signal reading *"in a caste world,
the two lowest-standing cast members hold different beliefs"*, and `:174` requires candidates to differ
*"under it, above it, outside it"*. That is my Round 1 G1 re-specialised into a rank ontology — a
criterion no non-hierarchical world can satisfy, in the one artifact implementers will treat as law.
A world whose norm is a tide table, a rota, a taboo, or an alien protocol with no ranking fails those
signals while being exactly what this feature promises. **This is veto #2 below.**

## 1. Attacks

### On simarch — the seam is inverted: it hides the law from minds and hands it to the player

**S-1 (verified, fatal as specified).** You wrote: *"Commit each as a `perception_record` **held by the
order's group entity**, so `fn_visible_perceptions` (`schema.sql:3080-3086`) hands it to every viewer
with no new read path"* (`round1_simarch.md:47-51`), and made it an invariant: *"at arrival tick,
`fn_visible_perceptions(world, player)` returns every authored norm"* (`:94-96`). I checked both ends.

The channel is real — `fn_visible_perceptions` has **no viewer filter** on group holders
(`core/db/schema.sql:3080-3086`). But its consumers are the wrong ones in both directions:

1. **It never reaches a mind.** Cognition reads exactly two perception sources: `fn_public_moment` and
   `fn_private_records`, and both filter `holder_id = ANY(p_present)` (`schema.sql:2679-2684`,
   `:2734-2740`), where `present = fn_actors_at(player's location)` (`orchestrator.go:752`). A group
   entity has no `actor_state` row and no location, so it is never present, so its perceptions enter
   **no cognition prompt** — which is precisely why your own M3 has to hand-render the norms into the
   prompt (`round1_simarch.md:65-70`). The storage seam does none of the work you assign it; M3 does
   all of it. `fn_public_moment` also requires `count(DISTINCT holder_id) = cardinality(p_present)`
   (`schema.sql:2740`), so a group-held row can never become the public moment either.
2. **It goes straight to the player.** The player's beat payload is built from
   `fn_visible_perceptions` (`beathandler.go:213-228`) with a window of
   `acquired_tick >= max(acquired_tick) - 50` (`:209,224`). Your norms ground in backstory events —
   ticks 30-37 — and arrival is tick 50 (`worldgenesiscommit.go:52-57`). 50 − 50 = 0: **every authored
   norm falls inside the player's very first payload**, capped only by `recencyMaxRows = 20`. They also
   surface in the player's compendium (`fn_collected_knowledge`, `schema.sql:1154`) and timeline
   (`:2834`) at tick 0.

So the design ships a newcomer who knows the constitution and NPCs who do not. Your defence —
*"AC-4 (`prd_world_creation.md:65`) holds unchanged because the norms are group-held"* (`:96-97`) —
satisfies AC-4's *count* and destroys the thing AC-4 protects: *"a user … ends up standing in a world
… in which they know nothing yet"* (`prd_world_creation.md:18`). It is a codex screen implemented in
SQL, which all four of us claim to have cut.

**S-2 — `standing_over` has no reader, by your own test.** You wrote *"Anything else is decoration
with a receipt"* (`round1_simarch.md:25`) and cut a `caste` attr because *"nothing reads"* it (`:26`).
`standing_over[]` (`:41`) is read by nothing: you explicitly refuse viewer-aware `fn_portal_permits`
(`:76-81`), no Tier-1 key computes with it (`tier1.go:3`), and the cognition prompt renders text. Yet
M5 spends a belt refusal on its **acyclicity** (`:86`) — validating the topology of an edge no consumer
traverses. It is also the only genre-shaped assumption in your proposal: your GA-2 defence tests it
against *"a sci-fi thriller, a workplace drama and a horror story"* (`:103-104`), three human social
settings. It fails the moment the norm is a tide, a rota, or a protocol with no dominance at all.

### On extraction — your own paper concedes the package changes nothing the player can feel

You wrote: *"Whether it changes play depends on the NPC cognition seat consuming
`relationship_state`/`member_of` — and today it doesn't"* (`round1_extraction.md:104-106`). Held.
That is not a boundary note, it is the product verdict on M1-M8, and two of your commits are inert on
inspection:

- **M2's membership rows have no reader.** *"Commits as `event_participant` rows off the `world_genesis`
  naming event … with `role_qualifier='member'`"* (`:40-43`). Name knowledge at genesis is derived from
  `history[].who` co-participation in Go (`worldgenesiscommit.go:366-396`), not from
  `event_participant`; and every `event_participant` reader in SQL sits inside `gather_slice`
  (`schema.sql:3320`) and `generate_perceptions` (`:3383-3444`), neither of which runs on genesis's
  direct writes. The rows land and nothing ever selects them.
- **M6 mistakes a past event for a felt one.** *"'What happens when a rule is broken' is an ordinary
  `history[]` beat … No enforcement schema needed"* (`:69-72`). A history beat's perceptions are held by
  cast members; the player holds none by AC-4, and their payload window carries only their own visible
  rows (`beathandler.go:223-228`). So an authored precedent is invisible to the player until somebody
  tells them — real depth for NPC context, zero player-facing surface. It is a necessary input to my
  rec 6, not a substitute for it.

Your M4/M8 discipline is right and I adopt it below; your storage targets are not where play happens.

### On ux — the playback manufactures consent for statements it never binds

Your seam survives S-1 and I concede the screen. But the contract is broken in a way your own AC hides.
Amendments travel: *"amended statements travel as `InterviewAnswer` rows, which `buildWorldGenesisPrompt`
already threads into genesis"* (`round1_ux.md:20`), and the draft's signal is *"100% of playback
amendments honored"* (`PRD-DRAFT:65`). But an **accepted** statement travels as nothing:
`buildWorldGenesisPrompt` renders only non-empty question/answer pairs and skips the rest
(`worldgenesis.go:213-228`). So the user ticks "Rank is visible on sight ✓", genesis never receives that
confirmation, re-infers independently, and — because the document is never served (AC-7,
`durable-worlds-design.md:111-112`) — no surface can ever show the divergence. Two inferences, one
signature, no binding: that is a consent screen over a document the user is structurally forbidden to
read. One-line fix, which I put in the list: **every statement the user is shown travels as an
`InterviewAnswer` row — accepted ones too** ("Rank is visible on sight" → "yes"), so the seat is bound
by the whole constitution it displayed, not only by the edits.

Second, smaller: your G-A2 claim that G4 *"decides what the player watches happen in their first
beats"* is right, and I take the amendment — but note the demonstration is the one mechanism that
*needs* no playback to be honest, because it names a practice the brief already implies. Ranked
accordingly, not vetoed.

## 2. Concessions

1. **To ux (G-A1): I overclaimed.** *"Without G2, nothing else here is worth building"* was wrong.
   Depth delivered through cognition produces narratable acts, not refusals; the silent-refusal defect
   binds the *access* surface only. G2 drops from precondition to rank 4.
2. **To ux (G-A3): legibility is required and I missed it.** My G5 assumed membership visible on sight;
   a concealed order rendering a first-sight group descriptor leaks what the fiction hides. Adopted as a
   gate on rec 7.
3. **To ux (G-A2): the scheduled demonstration must be playback-visible** — the practice, never the
   scene.
4. **To simarch: your grounding rule is right and my G1 omitted it.** Norms must ground in a backstory
   event, never `world_genesis`, because `fn_perceived_name` reads every `world_genesis`-sourced,
   subject-linked perception as that entity's **name** (`schema.sql:2584-2599`). With a group holder the
   blast radius is worse than the archivist bug you cite (`worldgenesiscommit.go:550-555`): a
   world-visible row would rename an entity for *every* viewer at once.
5. **To simarch: M5's third refusal beats my G6 for authoring-time guarantees.** *"At least one
   `history[]` entry's `knowledge` records that norm being enforced or broken. The law must have already
   happened once"* (`round1_simarch.md:89-91`) is a belt check, not a sampled audit. Adopted as rec 9.
6. **To extraction: my per-holder criterion needed a join key I refused to give it.** "Two holders differ
   about the same rule" is unassertable unless the rule has an identity; your *"closed shape, open
   vocabulary"* (`round1_extraction.md:19-22`) is how it gets one. The norm carries a stable
   `canonical_name` as a join key and no kind enum.

## 3. Final converged recommendations

Ranked. **[VETO]** = I consider shipping without this a defect, not a tradeoff.

1. **[VETO] The player does not arrive knowing the law.** Delete the group-visible-norms invariant
   (`PRD-DRAFT:61,119-121`; `round2_ux.md:34` rec 2). Norms commit as perceptions held by the **NPCs the
   norm binds** — the holders cognition actually reads (`schema.sql:2679-2684,2734-2740`) — with a
   group-held copy permitted only if `fn_visible_perceptions` gains a viewer clause first.
   *AC: at arrival tick, `fn_visible_perceptions(world, player)` returns exactly one row, and no norm
   text appears in the player's first beat payload (`beathandler.go:213-228`).*
2. **[VETO] No hierarchy, and no caste, anywhere in the contract.** Cut `standing_over[]`, "two
   lowest-standing" (`PRD-DRAFT:62`), and "under it, above it, outside it" (`:174`). A norm is
   `{canonical_name, stated, bearing}`; ordering, when it exists, lives in `stated` prose.
   *AC: no schema key, prompt line, belt check, or eval fixture encodes rank or an ordering between
   collectives, and a world whose norm binds everyone equally (a taboo, a tide) passes every criterion
   unchanged.*
3. **[VETO] Wire or delete every authored field** (my G0, adopted at `round2_ux.md:86-88`): `standing`,
   `arrival.why`, `places[].kind`.
   *AC: CI asserts every non-numeric `world_genesis/1` leaf reaches a table read by a prompt or payload;
   `grep Standing core/api` resolves to a commit-path write or to nothing.*
4. **The law reaches minds or the mechanism is cut** (simarch M3 + the kill-switch he offered).
   *AC: one brief built with and without norms produces differing NPC decisions inside five beats; no
   difference ⇒ removed.*
5. **Per-holder variance, joinable** (my G1 + extraction's join key).
   *AC: where a norm is authored, ≥2 holders' beliefs referencing that norm's `canonical_name` differ in
   content and `epistemic_type`, and no holder holds `direct` for what they were only told — vacuous,
   not failed, when no norm is implied.*
6. **One authored near future per world, playback-visible** (my G4, ux's amendment). Genesis becomes
   `pending_event`'s first production writer (today tests only: `101_personality_world_test.sql:58`),
   `when` as a class, `{canonical_name, attempt}` resolved at commit (`ledger.go:12`).
   *AC: within the baseline 5-beat window (`prd_world_creation.md:22`), the player perceives one authored
   future event they did not cause, in every world — norm or no norm.*
7. **Refusal states its cause** (my G2, rescoped to the access surface per ux G-A1).
   *AC: a move at a locked way narrates the obstruction from the fact vocabulary the narrator already
   answers questions from (`narrate.txt:33`); zero new seats or calls.*
8. **The collective is an earned-name compendium page, gated by legibility** (my G5 + ux S-2).
   *AC: the group's canonical name is unreachable until earned (`fn_unearned_names` has no kind filter,
   `schema.sql:2924-2939`); a concealed order renders no first-sight descriptor; page items carry
   epistemic framing (`schema.sql:1194-1206`).*
9. **The belt refuses law that never happened, and law parked on the naming event** (simarch M5 +
   concession 4).
   *AC: a document whose norm has no `history[]` enforcement/breach beat refuses with a stated reason; a
   norm perception sourced from `world_genesis` refuses; each class has a captured fake payload in CI.*
10. **Playback binds everything it shows, and the eval is non-hierarchical** (ux M1 fixed + extraction
    M8 + my G6).
    *AC: every displayed statement — accepted or amended — appears in the genesis ANSWERS block
    (`worldgenesis.go:213-228`); the eval's N briefs share no norm shape and include a negative control
    that authors no collective and costs no extra tokens; behaviour targets are sampled audits, never CI
    equalities.*
