# Round 2 — extraction (LLM Extraction/Schema Expert)

## 1. Sharpest attack on each peer

### On simarch: your common-knowledge mechanism ships the player the law before arrival

simarch's M2: *"Commit each [norm] as a `perception_record` **held by the order's group entity**, so
`fn_visible_perceptions` (`schema.sql:3080-3086`) hands it to every viewer with no new read path and
no D-1 bypass."* M6's guard: *"at arrival tick, `fn_visible_perceptions(world, player)` returns every
authored norm **and the player's own `perception_record` count is still exactly 1**."*

I read `fn_visible_perceptions` myself (`core/db/schema.sql:3072-3087`, verified against the live
schema, not the design doc):

```sql
WHERE pr.world_id = p_world_id AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
  AND ( pr.holder_id = p_viewer_id
        OR pr.holder_id IN (SELECT er.entity_id FROM entity_registry er
                             WHERE er.world_id = p_world_id AND er.entity_kind IN ('faction','group')) )
```

The group branch carries **no viewer filter, no co-location check, no distance, no prior-encounter
gate** — it returns for *any* `p_viewer_id`, including one who has never met the group. This is not
a side function: `beatseats.go:8-9` states it is *"the ONLY world input either model seat receives"*
— confirmed by reading it — and `beathandler.go:213-227`/`scenehandler.go:199-201` build the player's
own narration payload from exactly this call. So the moment M2 writes a group-held `public` norm at
content-commit, `fn_visible_perceptions(world, player_id)` returns it — before the arrival
transaction even runs, let alone before the player meets an Alpha. That is structurally the same
failure the PRD names and forbids by name: *"No authored roster of who is present: that would fake
fan-out the player never received"* (`prd_world_creation.md:65`). A law-roster is a roster.

M6's own acceptance check does not catch this: it counts rows where `holder_id = player`, which stays
1 regardless — the leak lives entirely in the group branch, on a row the player never holds, read
through a function M6 never queries directly. **simarch verified the wrong invariant.** I also note
this is a *new* failure mode M2 introduces, not a latent one: today's pipeline never creates a
`group`/`faction` `entity_registry` row (`registerEntities`, `worldgenesiscommit.go:292-338`, confirmed
by both of us independently — simarch's own round-1 line 15-17 says the same), so the dormant branch
has never fired from genesis. M2 is the first thing that would fire it, and it fires it wrong. Fix in
§3 item 4.

### On gamedesign: G1 and G5 contradict each other on whether a collective needs to exist

G1's own acceptance text: *"No rank, tier or **membership** required; identical for a caste, a debt or
a taboo; vacuous — not failed — when the brief implies no norm."* (round1_gamedesign.md, G1). G5,
three paragraphs later: *"A collective, when implied, is a page you fill in, and its name is earned...
`fn_compendium_index` is already kind-parameterised... one handler registration... plus a page
function yields a surface accumulating contradictory accounts."*

G5 cannot exist without exactly what G1 just declared unnecessary: a compendium *page* for "the
Alphas" requires an entity to *be* the Alphas — something `fn_compendium_index` can key on, something
the naming wall can hide behind a descriptor (G5's own point about `fn_unearned_names`'s missing kind
filter). G1 gives you a trait on Mara and a belief in Jonas's head; neither is a thing named "the
Alphas" that a page can be *about*. gamedesign built the discovery surface for an entity their own
behavior mechanism refuses to mint. This is exactly why I'm not dropping `groups[]`/`member_of` (§3
item 3) despite conceding the rest of my round-1 relational proposal (§2) — G5 needs it and gamedesign
didn't notice they needed it.

### On ux: M1's playback screen breaks the fast lane it claims to leave alone

ux's M1: *"After the brief (**both lanes**), before the expensive genesis call, one cheap seat turn
renders the brief's entailments as world-language statements."* This is a third seat call, always run
— including fast lane, which the PRD's own mermaid diagram routes straight from brief to build with no
intermediate node (`prd_world_creation.md:96-107`: `L -->|"Fast lane"| B["build now"]`), and which the
spec states in as many words: *"**Fast lane** is the brief alone... Anything else would mean two
things to keep correct"* (`prd_world_creation.md:109`, verified — my own round-1 read of this file).
M1 makes fast lane "the brief, plus a screen you must look at and a call you always pay for," even
when `groups`/`relations` come back empty and there is nothing to confirm. ux never prices this against
p50 ≤ $0.25 (`prd_world_creation.md:26-27`) or reconciles it with gamedesign's explicit cut of *"a
second LLM seat for 'systems extraction' — one call, no repair loop, is a deliberate cost decision"*
(round1_gamedesign.md §4) — a cut I share. I don't reject the screen (see concessions); I reject it as
an unconditional third call on every brief regardless of whether there is anything to show.

## 2. Concessions

**To simarch and gamedesign, fully: `relationship_state.attrs` is dead on arrival — my round-1 M3 is
withdrawn.** Both independently found what I found myself in round 1 and didn't act on hard enough:
zero lines in `core/api` read it, and the `[RELATIONSHIPS]` block `06_context_assembly_spec.md:88`
promises does not exist in `cognitionprompt.go` (I re-grepped it myself for round 2: zero matches for
`RELATIONSHIP` or `relationship_state` in that file). gamedesign's rule is the right one and I adopt
it verbatim: *"render the block and I vote for it"* — until then, unread state is not depth, full
stop.

**To gamedesign: G0 should have led my round-1 paper, not trailed it.** I proposed new schema
(`groups[]`, `relations[]`) while the one field that *already* carries exactly what the founder asked
for — `cast[].standing`, *"What they do and where they sit in this world's order"*
(`world_genesis.v1.schema.json:130-134`) — is authored, required, refused-when-empty
(`worldgenesis.go:310-311`), and dropped at commit. I re-verified this myself: `grep -n Standing
core/api/*.go` returns exactly two hits, the struct field and the validator — nothing in
`worldgenesiscommit.go` ever reads it. Proposing new fields before fixing a field that already leaks
is the wrong order of operations. I also verified gamedesign's `arrival.why` claim independently:
`worldgenesishandler.go:534` assigns it into the committed doc, but `worldgenesiscommit.go:201`'s
`world_character` insert is `(world_id, entity_id, descriptor, canonical_name)` — no `why` column, and
a repo-wide grep for `.Why` outside the handler/tests confirms it is never read again after that
assignment. Both citations check out exactly as gamedesign stated them.

**To gamedesign: G2 (refusal states its cause) outranks everything I proposed in round 1.** I didn't
touch the gate side at all. I re-read `narrateprompt.go:227-228` myself and confirmed the quote is
exact: *"NOTHING RESOLVED: the attempted action did not happen; the situation is exactly as it was."*
No cause, ever, for any refusal. A caste system the player can violate but never be told *why* it
failed is a worse product than no caste system — I adopt gamedesign's ordering (*"behaviour first,
legibility second, discovery third, gate enforcement last"*) over my own round-1 ordering, which had
no ordering at all.

**To ux: "shown ⇒ committed ⇒ consumed" (M2) is the correct meta-rule and I accept it as a hard
dependency on my own proposal.** ux named it as a requirement on me directly (*"I place this as a
requirement on simarch and extraction"*), and it's correct: nothing in `groups[]`/`member_of` (§3
item 3) should ship without a confirmation surface reading from the exact fields that survive commit
— otherwise I've reproduced the `standing` bug one field over. I also independently verified ux's
`worldinterview.go` claim and it is *stronger* than stated: the seat error at
`worldgenesishandler.go:151` is logged (`:158`, *"the fault goes to the log"*) but the HTTP response
at `:164-177` renders `{"done": true}` with **200 OK**, indistinguishable from a genuine "nothing left
to ask." ux's "silently collapses" is not an approximation; I read the handler and it is exactly true.

## 3. Final converged recommendation list

Ranked; items 1, 4, and 10 are marked **HARD VETO** conditions — ship past them and the rest of the
list is unsound regardless of how well-built it is.

1. **HARD VETO gate — wire or delete every existing leaked field first.** `cast[].standing` and
   `arrival.why` are authored, validated, and discarded today (verified above). *AC:* a test asserts
   every non-numeric `world_genesis/1` leaf reaches a table some prompt or payload reads
   (gamedesign G0); `standing` becomes item 3's `member_of` or is deleted from the schema; `why`
   lands on `world_character` or is deleted. No item below ships while this fails.

2. **Refusal states its cause, in world.** Extend the `NOTHING RESOLVED` render
   (`narrateprompt.go:224-228`) with the already-computed obstacle (locked portal, absent listener,
   encumbrance) (gamedesign G2). *AC:* a move at a locked way narrates the obstruction; zero new
   seats, zero new calls.

3. **Collectives are entities, not prose.** Optional `groups[]` (`{descriptor, canonical_name,
   standing}`, `standing` free prose, no `kind`/`caste`/`faction`/`rank` field ever), `minItems: 0`.
   `cast[].member_of[]` replaces the dead `standing`, join-keyed exactly like the proven
   `starts_in`→`places` pattern (`worldgenesis.go:318-319`). Commits via `entity_registry` with
   `entity_kind='group'` — legal today (`schema.sql:3722`), unused today
   (`worldgenesiscommit.go:292-338`), zero new DDL. Independently converged: this is simarch's
   `orders[]` and my round-1 M1 arrived at the same mechanism from different directions.
   *AC:* a brief implying a named collective produces one queryable `entity_registry` row and
   resolvable membership; a brief implying none produces zero rows, zero added tokens.

4. **HARD VETO — norms are per-mind behavior and per-holder belief, never a group-held
   `perception_record` at genesis.** Corrects simarch's M2/M6 (§1): author norms as (a) a trait on
   whoever upholds them, `manner` stating it behaviourally, grounded via `trait_provenance` exactly as
   traits already are (`worldgenesiscommit.go:537-544`); (b) `history[].knowledge` per-holder beliefs
   — existing schema, zero change (`world_genesis.v1.schema.json:240-267`). *AC replaces simarch's
   M6:* `SELECT count(*) FROM fn_visible_perceptions(world_id, player_id) WHERE holder_id != player_id
   AND holder_id NOT IN (SELECT entity_id FROM entity_registry WHERE entity_kind IN
   ('actor'))` immediately after content-commit (before the arrival transaction ever runs) returns
   zero — no group/faction-held perception is visible to a viewer id who has never held one directly.
   Veto any `perception_record` write with `holder_id` resolving to a `group`/`faction` entity at
   genesis time until this assertion is in the belt.

5. **The norm has already been broken or enforced once, in authored history.** simarch's M5 third
   refusal: at least one `history[]` entry's `knowledge` records the norm being enforced or violated —
   "what happens when it's broken" is an ordinary backstory beat
   (`20260813142100_world_templates.sql:141-167` shape), no new schema section. *AC:* a norm bearing on
   an authored group/trait fails validation unless at least one `history[]` entry evidences it having
   already mattered once.

6. **Confirmation surface: shown ⇒ committed ⇒ consumed, and only when there is something to
   confirm.** ux's M1/M2, corrected per §1: the playback screen renders **only** when items 3-4
   authored something non-empty; a brief implying no structure skips the screen and the extra seat
   call entirely, preserving *"Fast lane is the brief alone"* (`prd_world_creation.md:109`) and its
   cost budget. *AC:* every rendered playback statement traces to a field item 1's test proves reaches
   a consumer; a structureless brief in fast lane makes the identical single genesis call it makes
   today, unconditionally.

7. **The interview needs zero new question types.** ux's M3 / my round-1 M5, converged: the existing
   *"ask what changes the world most... the pressure everyone is under, who wants something from
   whom"* instinct (`world_interview.txt:3`) already covers a norm-implying brief. No prompt change,
   no schema change to `world_interview/1`.

8. **Eval harness: one sampled-audit methodology, not two.** Merge gamedesign's G6 (N briefs whose
   implied norms "share no shape," plus a negative control, asserting behavior/refusal/discovery
   observables) with my round-1 M8 (plant-and-measure: 0% invented structure, <10% missed structure) —
   both independently modeled on `07_test_and_invariant_spec.md:26-27`'s I-6 audit without either of us
   citing the other in round 1, which is itself evidence this is the correct methodology rather than a
   novel one. *AC:* N ≥ 20 briefs across unlike shapes + 1 negative control, human-audited, sampled per
   release exactly as I-6 already is.

9. **`pending_event` becomes genesis's first production writer.** gamedesign's G4, lower-ranked
   because it is orthogonal to the norm/collective mechanism, not required by it: one authored
   near-future per world, `when` as a class the engine resolves via the same precedent
   `fn_duration_class_seconds` already sets (`schema.sql:1586-1594`, verified: an existing
   class→numeric mapper, same shape as `extent_class`→metres). *AC:* inside the PRD's own 5-beat
   window (`prd_world_creation.md:22`), the player perceives one authored future event they did not
   cause.

10. **HARD VETO — no taxonomy, ever, and no new gate.** Three-way independent convergence, unanimous
    across all four round-1 papers: no `caste|faction|guild|rank|tier|authority` identifier in code,
    prompt, schema key, or fixture name — content only (my round-1 cut, gamedesign's cut, simarch's own
    GA-2 test). No viewer-aware `fn_portal_permits`, no numeric authority/access field, no Tier-1
    growth (simarch's own M4 self-cut, gamedesign's cut, my round-1 cut — three-way agreement without
    coordination). No engine-side sanction firing: access stays a lock plus a person standing at it
    (simarch: *"the door is physics; the guard is a mind"*), enforced through item 4's behavior
    channel, never a permission predicate the service would have to understand in order to check.
