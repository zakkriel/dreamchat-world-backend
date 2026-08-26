# Round 1 — extraction (LLM Extraction/Schema Expert)

## 1. Thesis

The founder's caste example does not require a new engine capability. It requires the genesis
seat to stop being forced to flatten relational and collective facts into free-text prose fields
that nothing downstream can join on. The engine's own frozen design already has the substrate —
`relationship_state` (pairwise, free-JSONB `attrs`, schema.sql:3911-3918) and `event_participant`'s
`faction`/`group` entity kinds (schema.sql:3722) — deployed, invariant-compliant, and **completely
unused**: `registerEntities` (worldgenesiscommit.go:292-338) only ever inserts `entity_kind`
`location`/`actor`/`artifact`, and a repo-wide grep for `relationship_state` inside `core/api`
returns zero hits. The gap is not "the model doesn't understand caste systems." The gap is that
`world_genesis/1` (`core/api/schema/world_genesis.v1.schema.json`) has no join key for "a named
collective" and no join key for "an asymmetric fact between two named entities," so even a model
that infers the whole Alphas system correctly has nowhere to put it except inside `cast[].standing`
prose (`world_genesis.v1.schema.json:130-134`), which nothing can reference, validate, or feed to
an NPC cognition seat.

The fix is **closed shape, open vocabulary** — the exact discipline `traits/1` already proves
(`world_genesis.v1.schema.json:140-162`: `key` is free string, `strength` is a closed enum) — applied
one level up, to groups and relations instead of only to individual actors. This is extraction
work, not simulation work: get the implied structure into append-only canon with a resolvable name
and a source event. Whether an engine-side gate *enforces* "a Beta may not enter the temple" is
simarch's problem, and I say so in §3.

## 2. Concrete mechanism proposals

**M1 — `groups[]` in `world_genesis/1`.** New optional array, same shape discipline as `places[]`/
`objects[]`: `{descriptor, canonical_name, standing}` — `standing` is free prose ("the ruling caste;
only they may enter the temple, own land, or strike a Beta without answer"), not a typed field. No
`kind` enum of "caste|faction|guild" — the word "caste" only ever appears in prose, exactly the way
`places[].kind` is already a free string, never a genre-locked noun (`world_genesis.v1.schema.json:
66-70`). `minItems` is 0: a coffee-shop brief authors zero groups and pays nothing extra, mirroring
`arrival_candidates`' present-only-when-implied pattern (`world_genesis.v1.schema.json:304-308`).
Commits via `registerEntities`-style insert with `entity_kind='group'` — already legal
(schema.sql:3722), zero new tables.

**M2 — `member_of` on cast.** `cast[].member_of: string[]` — canonical_names from `groups`, same
join-key mechanism `cast[].starts_in` already uses against `places` (`world_genesis.v1.schema.json:
169-173`, validated at `worldgenesis.go:318-319`). Commits as `event_participant` rows off the
`world_genesis` naming event (`worldgenesiscommit.go:349-398`, `writeNamingEvent`) with
`role_qualifier='member'` — `event_participant.role_qualifier` is free text already
(schema.sql:3721), no schema change there either.

**M3 — `relations[]` in `world_genesis/1`.** `{between: [canonical_name, canonical_name],
symmetric: bool, stance}` where `stance` is free prose authored per direction when asymmetric
("Alphas may enter the temple, own land, strike a Beta unanswered; a Beta who strikes an Alpha is
put to death"). `between` may name two cast members, or a cast member and a group (M1) — one
mechanism covers "who may speak to whom" and "what a caste as a whole may do." Commits into
`relationship_state.attrs` (schema.sql:3911-3918) exactly as `world_theme`'s `mood`/`ornament` are
free vocabulary by design (`world_genesis.v1.schema.json:24-32`) — the engine assigns nothing about
*what* the stance says, only *whom* it's attached to, same split as everywhere else in this file
(`10_prds/prd_world_creation.md:115-124`, "The model authors... the engine authors...").

**M4 — validate() cross-references.** Extend `genesisDoc.validate()` (`worldgenesis.go:249-495`)
with the same style of check already there for `ways`/`objects`/`history`: every `relations[].between`
entry and `member_of` entry must resolve against `cast`/`groups` union or refuse
(mirrors `worldgenesis.go:367-370`'s "a way leads from X, which is not a place"); a `groups` name
colliding with an existing `cast`/`places` name refuses (mirrors the `places[name]`/`cast[name]`
collision checks at `worldgenesis.go:276-277,304-305`).

**M5 — zero interview prompt changes.** `world_interview.txt` already instructs "ASK WHAT CHANGES THE
WORLD MOST... the pressure everyone is under, who wants something from whom" (`core/api/prompts/
world_interview.txt:3`) and forbids slot-filling (`world_interview.txt:4`). That's already the right
instinct for surfacing "is there a caste system, and who enforces it" *if the brief leaves it open*.
The interview doesn't need a new question type; it needs somewhere to put the answer once genesis
runs, which is M1-M3. Don't touch this seat — that's my answer to ux's likely "under-asking" attack.

**M6 — enforcement is already expressible, no new section.** "What happens when a rule is broken" is
an ordinary `history[]` beat — `{what_happened: "a Beta was caught past the temple steps and beaten
in the square", where, who, knowledge}` — using the exact mechanism the template already proves
(`20260813142100_world_templates.sql:141-167`, M-E1..M-E4). No enforcement schema needed.

**M7 — genesis prompt guardrail (one paragraph, not a new seat).** Add to `world_genesis.txt`,
adjacent to its existing "speak the world's language, never a genre's" instruction
(`core/api/prompts/world_genesis.txt:13`): groups/relations are authored **only** when the brief's
own words imply a named collective or an asymmetric access/permission fact — never invented to
look thorough. Same discipline as `prd_world_creation.md:178`'s "no fixed cast/place/object counts
as a rule."

**M8 — fake-seat + eval harness.** Extend `NewFakeWorldGenesisDriver` (`bridge_fakes.go:667-800`) to
emit ≥1 group and ≥1 relation so `schema_payloads_test.go`-style CI captures the new shape from day
one (AC-13, `prd_world_creation.md:76`). For the *quality* question no CI check can answer — "did
this trace to the brief, and was nothing implied missed" — reuse I-6's own methodology exactly
(`07_test_and_invariant_spec.md:26-27`, sampled N=50 human audit): sample N=20 briefs with a
plantable implied-system phrase ("ruled by a caste called Alphas"), assert every authored group/
relation traces to brief language (0% invention) and the Alphas/subordinate structure was actually
extracted (target: <10% missed-structure, mirroring I-6's own miss-rate ceiling).

## 3. Three hardest attacks, pre-answered

**simarch: "`relationship_state.attrs` is free JSONB nothing reads — this is prose wearing a
column."** Correct, and I'm conceding the boundary rather than pretending otherwise: extraction gets
the fact into append-only canon with a resolvable join key and a source event
(`worldgenesiscommit.go:349-398`'s naming-event pattern). It does **not** build the gate that makes
`ActorMoved` into the temple check `relationship_state` before committing — that's a new rule in the
apply-event path, squarely simarch's turf. But the two are strictly ordered: you cannot enforce a
rule that was never extracted into a referenceable row, and today it structurally cannot be (M1-M3
close that). I'm handing simarch the substrate, not claiming to be the gate.

**gamedesign: "this is schema bloat nobody plays against — depth the player never feels."** M1-M3 are
`minItems: 0` and cost nothing when unimplied (M7's guardrail is the enforcement mechanism against
padding — same posture the interview already takes at `world_interview.txt:7`, "be willing to be
done early"). Whether it *changes play* depends on the NPC cognition seat consuming
`relationship_state`/`member_of` — and today it doesn't, because that seat's output shape is still
explicitly OPEN (`FINAL-world-npc-cognition.md:49`, "Cognition OUTPUT: the structured attempt
shape... OPEN"). That's a legitimate scope question for gamedesign to press on simarch/gamedesign's
side of the table, not evidence the extraction shape is wrong — a fact that can't be joined on can
never be played against; a fact that can, can be wired in later without a genesis schema version
bump.

**Self-attack — "the model will just invent an Alphas caste for a coffee-shop brief to seem
thorough."** This is the real risk in my own proposal. `validate()` (M4) can check referential
integrity, never "was this implied" — that's not mechanically checkable, same reason `I-6`
(canonization threshold) is a sampled human audit, not a CI assertion
(`07_test_and_invariant_spec.md:26-27`). I answer it at the prompt layer (M7) and the eval layer
(M8's plant-and-measure harness), explicitly modeled on I-6 rather than invented from scratch — this
is a known, already-accepted class of risk in this codebase, not a new one.

## 4. What I would cut

- **No `relation_type`/`group_kind` enum** (`kinship|hierarchy|rivalry|caste|faction`). That is
  exactly the GA-2 trap: the moment there's a closed list of relation *kinds*, the service has
  learned what a world "usually" has. `stance`/`standing` stay 100% prose, same as `traits[].key`
  (`world_genesis.v1.schema.json:149-153`) — free vocabulary, no taxonomy, ever.
- **No numeric "authority tier" or "access level" field.** PRD AC-7 already forbids any number
  anywhere (`prd_world_creation.md:70`); unlike `strength`/`malleability`, which route through
  classes because the engine needs them *numerically* (`worldgenesis.go:497-526`), nothing
  downstream today consumes an access-tier number, so there is no engine need to justify even a
  class enum. Keep it prose until a numeric consumer exists and can name what it needs.
- **No dedicated interview question type for "social structure."** Would violate
  `world_interview.txt:4`'s own ban on slot-filling questions ("no question exists because a
  genre usually has one" — `prd_world_creation.md:71`). The existing "what changes the world most"
  instinct already covers it when relevant.
- **No enforcement-specific schema section (M6 supersedes it).** Enforcement is a `history[]` beat
  like any other; adding a typed "consequences" field would be the same taxonomy mistake one layer
  down.
- **No new engine table.** `relationship_state` and `group`/`faction` entity kinds already exist,
  deployed, unused. Building anything new before wiring the existing substrate is waste I'd fight
  in round 2 if simarch proposes it.
