# Engine capability audit — what the contract demands vs what the engine has

**Date:** 2026-08-26 · **Method:** static read of `core/api` and `core/db/schema.sql`, every row cited to
file:line. **Not** observed running — "the code path exists and is called" is a weaker claim than "it was
seen working."

**Companion to** `00_world_model_and_genesis_pipeline.md`. Reference material for the eight increments in
`docs/superpowers/plans/2026-08-26-world-model-eight-increments.md`. **Do not re-derive these numbers —
cite this file.**

**Rule set audited:** the 24 reader obligations = `SCHEMA-v3.md` §4 (23 rows, lines 89-129) with the
`channel.conceals` row replaced by the two rows of `SCHEMA-v5.md` §2.

**Rubric, applied strictly.** WORKING = code computes it *and* the value changes what happens in play.
PARTIAL = a table, column, enum value or function exists but is hardcoded, unwritten, unread, or not
connected to the authored key the rule names. ABSENT = no code.

## Headline

**3 WORKING · 12 PARTIAL · 9 ABSENT.**

The split is not random. Every WORKING row and most PARTIAL rows are **space, movement, containment,
weight and the epistemic wall** — the pre-existing play engine, which is real and live. The contract's
entire **time-and-change** block is one WORKING and four ABSENT-or-dead. Its **knowing** block is one
WORKING and eight PARTIAL-or-absent.

---

## The 24 rows

| # | Rule (author writes ⇒ engine derives) | Status | Evidence |
|---|---|---|---|
| 1 | `extent_class` ⇒ footprint + distances within it | **WORKING** | `fn_extent_class_metres` `schema.sql:1777` over table `:3901`, seeded per-world `:3716-3718`; footprint `fn_area_around:1016`; called at genesis `worldgenesiscommit.go:858-866`, at place minting `placeauthor.go:328-334`; distance `fn_distance:1557` (recursive climb to nearest common parent), consumed by the move clock `orchestrator.go:308-312` |
| 2 | `pace_class` ⇒ a speed; with distance ⇒ a duration | **PARTIAL** | Duration half live: `fn_move_duration_actor:2529-2543` = CEIL(distance/speed), charged as ticks `orchestrator.go:26-27,308-312`. **No `pace_class` and no class→speed ladder** — speed is a raw number `movement_type.base_speed_mps:4007`, seeded `walk = 1.4` `:3704-3706`, modulated `fn_effective_speed:1663-1706`. `fn_move_duration_actor` **hardcodes `'walk'`**; no actor↔movement-type binding exists |
| 3 | `motion.trajectory` ⇒ position over time; contents move with it | **PARTIAL** | Traveller position interpolated per leg `journey.go:468-487`, resolved into a frame `:489-513`, and it changes play (picks the scene the world's turn fires in, `journeyScene:534-568`). But **transient — never persisted** to `attrs.coordinates`; only `stage_id` written `:562-566`. **No moving container**: `within` transitivity is static only (`contained_by` in `fn_effective_weight_r:1719-1755`, `fn_carrying:1105`, `fn_occupied_room:2608`) |
| 4 | `passage.admits`/`obstructs` ⇒ permit/refuse per predicate; refusal names the failed predicate | **PARTIAL** | Gate live and blocks commits: `fn_portal_permits:2686-2697`, `fn_actor_move_permitted:792-816`, enforced in `apply_event`/`apply_ruled_event`, mirrored in Go `orchestrator.go:1156-1160,1241-1256`. But it **takes no mover properties** — the predicate is the portal's own `open ∧ ¬locked` booleans. And the **refusal names no predicate**: generic `premise_broken`/`journey_barred` (`station_f_exit_test.go:285-287`, `placeauthor_test.go:355`) |
| 5 | `passage.hazard_class` ⇒ a cost or condition on crossers, distinct from refusal | **ABSENT** | Zero occurrences of `hazard` in `schema.sql` or `core/api`. Traversal has exactly two outcomes, permitted or blocked (`fn_portal_permits:2686`). No cost, no toll, no condition |
| 6 | `bulk_class` ⇒ a volume and a mass | **PARTIAL** | Both exist and are read in play: `fn_volume:3155` = 4^(size−1), `fn_effective_weight_r:1719-1755` (recursive; container = (empty + Σ contents) × modifier), surfaced `fn_fact_sheet:1824-1834`. But **no `bulk_class`** — `size` is an authored integer, `weight`/`empty_weight` authored numbers, no ladder. The genesis seat is forbidden any number (`world_genesis.v1.schema.json:4`), so **an authored world reaches play with no size and no weight at all** |
| 7 | `capacity_class` ⇒ how much it holds; exceeding refuses | **PARTIAL** | Refusal real and fires at commit: `fn_occupied_room + fn_volume > attrs.max_room` rejects relocation `schema.sql:224-231` (twin `apply_ruled_event:531-538`); `attrs.max_load` sets/clears `encumbered` `fn_apply_carry_change:925-948`, and `encumbered` is a −100% walk modifier `:3707` ⇒ speed 0 ⇒ move impossible `:2529-2543`. But **no `capacity_class`** — `max_room`/`max_load` are authored numbers, and genesis emits neither |
| 8 | `integrity` ⇒ a degradation level, and how much remains before terminus | **ABSENT** | Zero occurrences of `integrity`, `degrad`, `durability`, `wear`, `condition` in `schema.sql`. No condition attribute in any projection table (`actor_state:3760`, `artifact_state:3774`, `location_state:3993`). Nothing has a terminus |
| 9 | `holds[].abundance` ⇒ a quantity; drawing reduces it; exhaustion refuses | **ABSENT** | Zero occurrences of `abundance`, `exhaust`, `quantity`, `stock`. Containment is by identity (`contained_by`), never amount — `fn_carrying:1105`, `fn_occupied_room:2608` count discrete objects. Nothing is drawn from anything |
| 10 | `tension` ⇒ a beat budget; acts exceeding it become extended, not refused | **WORKING** | Full ladder in code `tension.go:28-43` (frantic 5 / tense 30 / normal 60 / calm 600 / none ∞), read from the acting actor's scene `beatBudgetSeconds:49-63`, enum enforced `trg_validate_tension` `schema.sql:3745`. Consumption cumulative `tension.go:20-22`. **"Extended rather than refused" is live** — over-budget becomes a journey `journey.go:92-96`, `journey_beat_test.go:12-15` |
| 11 | `process.rate_class` ⇒ how fast state moves, without an event per tick | **ABSENT** | No `process` entity, no `rate_class`, no authored continuous state. Two *engine-owned* derive-from-elapsed-ticks instances exist, each **welded to one purpose**: `fn_pressure_chance:2704-2731`, `fn_compendium_decay:1332-1339` (fixed 72-tick horizon `:1346-1350`). Neither bindable to anything authored |
| 12 | `cycle.period_class` + `phases` ⇒ when each phase flips; the flip leaves a trace | **ABSENT** | Zero occurrences of `phase`, `day_night`, `weather`, `season` (the only `cycle` hits are acyclicity and containment-cycle guards `:737-763`, `:1731`, `:2479`). World time is a monotonic tick `fn_world_now:3212` with nothing periodic on it |
| 13 | `accumulator.thresholds` ⇒ fire once, in order, at each crossing; `irreversible` never un-fires | **PARTIAL** | `world_pressure(accrued numeric, threshold numeric)` `:4307-4313` is **exactly this shape and completely dead** — no Go and no SQL reads or writes either column (only its PK is exercised, `tests/101_personality_world_test.sql:77-85`). The one live threshold is `journey.threshold jsonb:3956` via `thresholdMet` `journey.go:246-290` — a *single* predicate per journey, not an ordered ladder, and it ends the journey rather than latching. The live pressure mechanism (`fn_pressure_chance:2704` + `rollTier` `pressure.go:56-64`) is a probability that **re-arms after each eruption** — the opposite of irreversible |
| 14 | `demand.unmet` ⇒ apply the effect after `onset_class`, and go on applying it | **ABSENT** | Zero occurrences of `demand`, `unmet`, `onset`. The `statuses` array on `actor_state` is the only standing-condition mechanism and has exactly one producer — `encumbered`, set synchronously by weight `fn_apply_carry_change:937-948`. Nothing accrues; nothing has an onset |
| 15 | `channel.latency_class` ⇒ the delay before a fact is knowable | **ABSENT** | Zero occurrences of `latency`. Every `perception_record` insert in `generate_perceptions` stamps `acquired_tick = ev.in_world_tick` — the event's own tick, no offset (`:3454-3457`, `:3475-3478`, `:3499-3502`, `:3611-3615`). **Knowledge is instantaneous by construction.** `pending_event:1618` schedules a future *event*, never a future *perception of a past event* |
| 16 | `channel.reach` ⇒ who can receive at all | **PARTIAL** | A receiver rule exists, is live, and gates everything downstream: `generate_perceptions:3415-3640` fans Communicated to speaker (`shared`) and listener (`told`), ActorMoved to mover and co-located, ObjectRelocated to witnesses; `fn_visible_perceptions:3133-3148` walls every read to holder ∪ faction/group holders. But **hardcoded to co-location and event participation** — no channel entity, no `reach` value, no way for an author to make one channel reach further than another |
| 17 | `channel.decay` ⇒ when a belief acquired through it expires | **PARTIAL** | Columns exist and are read on every knowledge path — `perception_record.invalid_tick:3119`, `expired_at:3121`, filtered `fn_visible_perceptions:3139-3140` and in all three cognition lookups (`:2410`, `:2428`, `:2747`, `:2802`), indexed `:4705`. **No production code writes either** — only pgTAP fixtures (`tests/46_wall_clause_coverage_test.sql:8-11`, `107_cognition_lookups_test.sql:159-162`). The one shipped decay `fn_compendium_decay:1332-1339` is a hardcoded 72-tick staleness label that removes nothing |
| 18 | `channel.conceals` ⇒ what identity is withheld; `none` ⇒ the emitter and nothing further | **PARTIAL** | Identity disclosure *is* governed and changes play: a name comes only via `name_knowledge:4020` earned through a knowledge path, resolved `fn_perceived_name:2645`/`fn_display_name:1503`, enforced by the naming-wall belt on narration `beatsstream.go:427-431,545-547,606-608`, split truth/perceived `fn_fact_sheet:1808-1811`. But **no channel and no `conceals`** — one global earned-name rule, not a per-channel setting an author can vary |
| 19 | (v5) a channel **never** discloses `hiding`; interiors reachable only by an act, an `indicator` or a `trace` | **WORKING** | An authored `hiding` is committed as a private perception **held by the actor about themselves**, grounded in the moment that caused it: `worldgenesiscommit.go:546-557` (`insertMind` → `writePerception(..., actorID, ..., "direct", ..., []string{actorID})`). `fn_visible_perceptions:3141-3147` returns it to that holder alone; no fan-out path in `generate_perceptions:3415-3640` can produce a copy; the naming wall is a second belt. **Holds by construction.** Note the engine has **no `indicator` and no `trace`**, so two of the three sanctioned carriers do not exist (rows 21, 12) |
| 20 | `path` ⇒ confidence, and what class of later event corrects it | **PARTIAL** | Path recorded and surfaced: `epistemic_type:3125` written per fan-out branch (`direct`/`shared`/`told` at `:3456`, `:3477`, `:3502`), rendered as `source.epistemic_type` `fn_collected_knowledge:1264`. But **`confidence real DEFAULT 1.0` `:3115` is never set by any insert** — every belief in the system is confidence 1.0, passed straight through `:1261`, `:2913`. **No correction mechanism**: `invalid_tick` is the retraction column and nothing writes it (row 17) |
| 21 | `indicator.reliability_class` ⇒ how often the sign misreports; the hidden value is **never** exposed | **ABSENT** | Zero occurrences of `indicator` or `reliab`. `distortion_level real DEFAULT 0 NOT NULL:3116` is the only column in the neighbourhood and is **written by nothing and read by nothing** — it appears in the DDL and its migration (`migrations/20260610090004_deltas_epistemic_causal.sql:43`) and nowhere else in the repository |
| 22 | `history.standing: disputed` ⇒ hold both accounts; never resolve without a later event | **PARTIAL** | `'disputed'` exists as an `epistemic_type` enum value `:3125` and is **never produced by any code path**. Divergent private readings *can* coexist as separate holders' records (`fn_private_records:2738`, exercised `tests/107_cognition_lookups_test.sql:141-162`), but that is an emergent side effect of per-holder storage, not a standing an author can set or the engine protects from resolution |
| 23 | `record.asserts[].accurate: false` ⇒ readable and wrong; reading it doesn't correct the reader | **PARTIAL** | `'mistaken'` exists as an enum value `:3125` and, like `'disputed'`, is **never written**. No record/document concept: `artifact_state.attrs:3774` carries `size`/`weight`/`open`/`locked`/`connects`/`max_room`/`contained_by`, nothing assertable. `fn_fact_sheet`'s `p_truth_side:1793` is a referee-vs-viewer split of the *same true* facts (`:1808-1811`, `:1835-1846`), not a false claim |
| 24 | `excluded[]` ⇒ refuse to author anything matching, in every seat, for the life of the world | **ABSENT** | No `excluded` key in any seat schema (`world_genesis.v1.schema.json`, `place_author.v1.schema.json`), no field on `genesisDoc` `worldgenesis.go:82-125`, no check in `validate:249-330`, no column on `world:4195-4213`. The genesis belt refuses only structural faults — dangling references, duplicate names, empty descriptors, out-of-enum `tension`/`extent_class` (`:250-320`). The only "never" in the system is a **global** content floor in prompt prose, identical in all nine prompt files (`prompts/place_author.txt:17,19` and siblings) — model instruction, not enforcement |

**WORKING (3):** 1, 10, 19.
**PARTIAL (12):** 2, 3, 4, 6, 7, 13, 16, 17, 18, 20, 22, 23.
**ABSENT (9):** 5, 8, 9, 11, 12, 14, 15, 21, 24.

---

## The most load-bearing absence: class-word → number

**Thirteen of the twenty-four rules dead-end in a word the engine cannot resolve:** 2 (`pace_class`),
5 (`hazard_class`), 6 (`bulk_class`), 7 (`capacity_class`), 8 (`integrity`), 9 (`abundance`),
11 (`rate_class`), 12 (`period_class`), 13 (threshold rungs), 14 (`onset_class`), 15 (`latency_class`),
17 (`decay`), 21 (`reliability_class`).

The class-to-number surface of the engine is **four** conversions, **two of which fail open.**

> **Corrected 2026-08-26.** This section originally said three. `FINDINGS_contracts.md` C21 put it at five;
> re-verified independently, **four** are confirmed and the fifth could not be. More important than the
> count: **two carry a silent numeric fallback**, which the original text missed entirely.

| Conversion | Where | On an unknown class |
|---|---|---|
| `extent_class` → metres | table `extent_class_metres:3901`, read by `fn_extent_class_metres:1777` | **`ELSE 50`** — silently 50 m. The comment says *"never fails closed"* |
| duration class → seconds | table `duration_class_seconds:3857`, read by `fn_duration_class_seconds:1647` | **`ELSE 2`** — silently 2 s |
| watch horizon → seconds | `fn_watch_horizon_seconds:3166` | `COALESCE(..., 86400)` — hardcoded fallback |
| `tension` → seconds | Go, `tension.go:28-43` | closed set, enum-enforced by `trg_validate_tension:3745` |

**The fail-open behaviour is the finding.** Under v6 a class ladder is per-world vocabulary, so a
hallucinated, mis-spelled or unseeded class word does not raise — it becomes 50 metres or 2 seconds and
play continues on a wrong number. That is the silent-drop class (SPEC-035) living inside the exact
function the generic resolver replaces. **The resolver must fail closed**, and a wrong-class fixture must
be refused with a named cause.

And the seat leash compounds it: `world_genesis.v1.schema.json:4` forbids the authoring model from
emitting **any** number, and `prd_world_creation.md:70` makes that a law. **So a genesis-authored world
is structurally incapable of arriving with a raw number — the class ladder is not one option for
supplying these quantities, it is the only one, and twelve of thirteen rungs do not exist.**

**Runner-up prerequisite:** state that changes with elapsed time and is derived at read time rather than
by an event per tick. Rows 8, 9, 11, 12, 13, 14 and 17 all need it. The engine has the pattern twice
(`fn_pressure_chance:2704`, `fn_compendium_decay:1332`), each welded to one purpose;
`state_mutation`/`apply_mutation:427` is strictly discrete-event.

---

## World identity during play: no global statement

> **Refined 2026-08-26 by `FINDINGS_playloop.md` F4.** This section originally read "there is none." That
> is **right about the brief and overstated about the world.** The narration path *does* receive
> world-authored prose every beat — the place description, the actor labels and the perception lines are
> all genesis-authored text (`narrateprompt.go:157-267`, blocks at `:169-192`, `:170-197`, `:211-249`).
> What is missing is the **global, per-world statement** — the world's premise, mood and minted
> vocabulary as a whole — not world content per beat. The corrected claim is below.

Stated with each near-miss, because each looks like it might be one.

- **`world.brief`** — the person's own words, the richest statement of a world's identity — is stored
  `schema.sql:4204` and carries an explicit `COMMENT` at `:4234`: *"Operational provenance, never
  rendered: no projection selects it."* Only pre-play readers: `kickstartstate.go:96-99`, and the
  genesis/interview handlers `worldgenesishandler.go:151,237,267,292`. **No beat-path code reads it.**
- **`world.theme`** `:4198` (mood/ornament/accent) is validated on write and echoed to the client
  `worldshandler.go:122-167`. Frontend chrome. No seat prompt, belt or SQL function consults it in a beat.
- **`world.art_style`** `:4205` governs generated images only — `artstyle.go:33-35`.
- **The belts that do run every beat are epistemic, not stylistic:** the naming wall
  `beatsstream.go:427-431,542-547,606-608`, the closed beat vocabulary `beatseats.go:40-44`, the portal
  floor, the perception wall `fn_visible_perceptions:3133`. They stop a seat leaking an unearned name or
  committing an unlawful act. **Not one of them can tell whether the prose belongs to this world.**
- **What the narrator does get, every beat:** the matched location's description, each present actor's
  label, and the perception lines — all world-authored at genesis. So a beat is not world-blind. What no
  prompt carries is the world's own global statement, and **the fix must source it from the committed
  document, never from `world.brief`** — handing the narrator the brief while the document is short of it
  is the founder-gate bug `NEVER CONTRADICT OR EXTEND THE STATE` exists to stop (`narrateprompt.go:14-17,28`).
- **The only refusal list is global prompt prose**, identical across nine files — a content floor for the
  service, not a per-world authored `excluded[]`, and nothing checks output against it.

---

## Which rules each world actually needs

Mechanical key scan of the three v4 documents in
`docs/superpowers/debates/2026-08-25-world-model-clean-sheet/`, cross-referenced against the briefs in
its `testworlds/`. `y` = the document uses the key; **(brief)** = the document omits it but its brief
plainly implies it.

| # | Rule | Grelda | Marea | Sueño | Engine |
|---|---|---|---|---|---|
| 1 | `extent_class` | y | y | y | WORKING |
| 2 | `pace_class` | y | y | · | partial |
| 3 | `motion` | · | · | · | partial |
| 4 | `admits`/`obstructs` | y | y | y | partial |
| 5 | `hazard_class` | y | y | · | **ABSENT** |
| 6 | `bulk_class` | y | y | y | partial |
| 7 | `capacity_class` | y | y | · | partial |
| 8 | `integrity` | y | y | y | **ABSENT** |
| 9 | `abundance` | y | y | · | **ABSENT** |
| 10 | `tension` | y | y | y | WORKING |
| 11 | `process.rate_class` | y | y | **(brief)** | **ABSENT** |
| 12 | `cycle` + `phases` | y | y | **(brief)** | **ABSENT** |
| 13 | `accumulator.thresholds` | y | y | **(brief)** | partial |
| 14 | `demand.unmet` | y | **(brief)** | **(brief)** | **ABSENT** |
| 15 | `latency_class` | y | y | y | **ABSENT** |
| 16 | `reach` | y | y | y | partial |
| 17 | `decay` | y | y | y | partial |
| 18 | `conceals` | y | y | y | partial |
| 19 | `hiding` | y | y | y | WORKING |
| 20 | `path` | y | y | y | partial |
| 21 | `reliability_class` | y | y | y | **ABSENT** |
| 22 | disputed history | y | y | y | partial |
| 23 | false record | **(brief)** | y | **(brief)** | partial |
| 24 | `excluded[]` | y | y | y | **ABSENT** |

**14 of 24 used by all three documents.** After brief evidence, **7 of the 9 ABSENT features are needed
by essentially every world** — only `abundance` (stocks) and `hazard_class` are genuinely optional, and
`hazard` is clear in one brief.

**Facets used by no world:** `motion` and `office`. `motion` is **not** a deletion candidate — the
product owner has confirmed moving locations are real (a train that is a location containing locations;
a floating island). `office` appears as a top-level section in all three worlds and as a facet in none,
which is a redundancy worth resolving.

### The genesis coverage bug this scan found

**The thin document is not a thin world.** `mundo-08-sueno-comun-1-basico.md` states a numbered threshold
(*"whoever doesn't sleep four nights running starts dreaming awake, in public, and the neighbourhood sees
it"*, rule 6), names a character eleven nights into it to demonstrate it (Vira Cor), states a daily cycle
with opening hours (*"every night, without exception"*; the archive *"from nine to four"*; the
transcription *"public from nine"*), and is built entirely on a record that is true about a dream and
false as an accusation. **`G_sueno_by_extraction.md` encoded none of it, and still passed its own validity
and sufficiency checks.**

That is one layer earlier than the "authored but never read" defect class of
`prd_world_creation_depth.md`: this is **stated in the brief and never authored at all.** Nothing
currently catches it. A brief-to-document coverage check is mechanical and cheap.

---

## Scope limits of this audit

1. **Static read, nothing executed.** Every WORKING row is supported by the call chain from derivation to
   the commit or clock advance that consumes it, but no test was run and no beat played.
2. **Dead-code claims are scoped to `core/`.** `world_pressure.accrued`/`threshold`, `distortion_level`,
   `confidence`, `invalid_tick`/`expired_at` and the `mistaken`/`disputed`/`confirmed` enum values were
   searched across Go, the SQL schema, migrations, pgTAP tests and seeds under `core/`. A consumer
   outside `core/` was not searched for.
3. **`fn_instantiate_drowned_lantern:1889-2400`** (a ~500-line seed for one hand-authored world) was read
   selectively. A seed-only attribute could have been missed; it would not change a row, since a
   seed-only attribute is by definition not a general engine capability.
4. **Three ABSENT rows rest on negative searches** — 5, 12, 14 — established by keyword search returning
   zero relevant hits, cross-checked against the complete function inventory (60 `CREATE FUNCTION`) and
   table inventory (34 `CREATE TABLE`), both read in full.
