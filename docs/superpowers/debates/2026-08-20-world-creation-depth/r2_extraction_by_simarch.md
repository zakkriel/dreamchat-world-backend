# R2 — extraction seat (rotated)

Checks first (§1.2): every identifier names an engine column or capability, never a value; the floor
below is a comparison against what the engine reads, not an exemption list; class→number stays
grammar; engine gaps are named as programs.

## 1. Attack on extraction's round-1 paper

**Referential pull is a closure property, not an elicitation force, and its fixed point is the floor.**

Extraction's mechanism: *"A collective is authored only when something else already needed to point at
one"* (`world_topics_extraction.md:11-12`), reusing `validate()`'s dangling-reference refusals *"as the
positive elicitation force"* (`:10-11`). Look at the direction of the edges. Every reference points
**downward** to clusters that exist for other reasons: `starts_in`→`places`, `who`→`cast`,
`binds`→`cast|collectives`. Nothing points **at** an optional cluster unless the seat already chose to
author the pointing thing. So the pressure is zero: two places, one way, one person, one secret, one
history beat, one object and an arrival resolves perfectly and is a global optimum. Extraction lists
nine clusters; referential pull reaches **none** of the optional ones. A closure check prunes; it never
solicits, and calling it positive does not reverse the arrow.

**The floor is cardinality wearing the word adequacy.** *"A small, closed, mechanism-adequacy floor…
≥2 `places`, ≥1 `cast`, exactly one `hiding` per person"* (`:14-16`). Those are array lengths, and they
do not guarantee the mechanisms named: a document with eight places and eight objects passes and still
leaves `fn_fact_sheet` returning `null` for `weight_kg`, `volume`, `would_encumber` and `contents` on
every object, because each is gated on `attrs ? 'size'` / `? 'max_room'` and genesis writes neither
(`worldgenesiscommit.go:672-695`; `fn_volume`, `20260729100003_object_relocated.sql:30-33`). Adequacy
is a property of **fields**, not array lengths.

**What the angle missed.** Extraction runs one test — reader-existence, oriented author→reader. There
are three defects and it sees two:

- **A — authored leaf, no commit path:** `cast[].standing`, `arrival.why`, `places[].kind`.
- **C — engine column, no reader:** `sensory_mode`, `distortion_level`, `relationship_state`.
- **B — engine input, no author:** `size`, `empty_weight`, `max_room`, `max_load`, `statuses`,
  `pending_event`, pressure intensity.

Extraction catches A and C and is structurally blind to B, which has no authored leaf to trace from.
**B is the entire coverage question** — "what must a full world contain" *is* "which engine inputs are
starved". simarch found B by reading the commit path, gamedesign by asking what happens in beat 1. A
schema-first orientation cannot see it, which is why the paper's answer to comprehensiveness stops at
*"does everything you did author resolve"* (`:22-23`) — a test no starved input can ever fail.

**Two inconsistencies.** (a) Criterion (b) — *"maps onto engine machinery that… already computes with
something there"* (`:46-47`) — kills `role`/`wants`/`doing`, then exempts `collectives` as *"none —
genesis-side only, no engine change"* (`:33`). Nothing computes with a group; group-held perceptions
reach no mind and leak to the player (handover §3.4). By its own test, collectives fail. (b) The §4
norm router — *"classified at commit time"* (`:75-78`) — moves handover §4.1's classification off the
decomposer onto free prose at commit: either a second seat call against a one-call ceiling, or a
keyword table, which is the exemption list §1.2 forbids.

## 2. Starved engine inputs — the coverage list

| # | Starved input | Read by | Genesis authors | Engine work |
|---|---|---|---|---|
| 1 | `artifact_state.attrs.size` | fact-sheet `weight_kg`/`volume` gate; `fn_volume`; `fn_occupied_room` (`20260729100003:44`) | a size class per object | class→int map, `extent_class_metres` shape |
| 2 | `…empty_weight` | `fn_effective_weight` | a weight class | class→kg map |
| 3 | `…max_room` | fact-sheet `contents` gate; `fn_occupied_room` | a capacity class, only on holders | as above |
| 4 | `…weight_modifier` | `fn_effective_weight` | optional; absent ⇒ 1 | none |
| 5 | `actor_state.attrs.max_load` | encumbrance comparison | a carry class per person — today one literal 80 for everyone, from a constant named for the player (`worldgenesiscommit.go:62,666`) | class→kg map |
| 6 | `actor_state.attrs.statuses` | `fn_effective_speed` (`20260729100002_move_duration.sql:46-48`) | named conditions per person | **yes** — engine-read, absent from `tier1.go:4-22` |
| 7 | `movement_type` past seeded `walk` | `fn_move_duration_actor` | ways of moving + pace class | **blocking** — `'walk'` hardcoded (`20260729100006_move_target.sql:63,66`); no actor↔type binding |
| 8 | `status_modifier` past seeded `encumbered` | `fn_effective_speed` | which condition hinders which movement | needs 6+7; `action_type CHECK ('move')` caps scope |
| 9 | `pending_event` | `fn_due_pending`, every clock crossing (`ledger.go:122-220`) | one scheduled thing + a magnitude class (NOT NULL, `small\|medium\|large`) | writer only |
| 10 | `world_actor_setting.intensity`, `world_actor_config.climb_rate` | `fn_pressure_chance` (`schema.sql:2661-2665`) | one class for this world's restlessness — identical for every world today (`20260808100001_interruption_tuning.sql:47-51`) | **none** |

1–5 and 10 are pure omission. 6–8 are one program. 9 is a writer plus §3.

**The floor that replaces extraction's:** *every entity a landing mints must carry every engine-read
key its kind has.* An artifact without `size` is under-authored exactly as a place without `tension`
is — which the engine already refuses at write (`trg_validate_tension`,
`20260723100002_six_type_spine.sql:43-49`). This is a comparison against the engine-read key registry,
not a shape mandate: it says nothing about how many objects a world has, only that an object the
referee is told about must be one the referee can measure. It also makes the fake driver a real
contract — the fake must emit a fully-keyed world or CI fails — which is the eval story the round-1
paper never gives.

## 3. Adjudicating the two omissions

Not one thing, and **gamedesign's is more load-bearing**, on two grounds a schema seat can weigh.

*Consumption frequency.* Opposed wants feed a reader that runs for every present mind on every beat
(`worldFirst`); recurrence feeds one that fires at most once per authored event.

*Cost of the fix.* Wants is one prompt section and one query — no column, no migration, no signature
change. Recurrence needs an interval column, re-arm logic and a re-entry decision, and it is **doubly
blocked**: `pendingPayload` is `{actor_id, attempt}` (`ledger.go:16-19`), so an event with no actor —
weather, a tide, a bell — is inexpressible whether or not it repeats. Under §1.2 "engine gap ⇒ engine
program" recurrence leaves this document; opposed wants stays in it.

Both claims were overstated. `fn_pressure_chance` is live and seeded, so autonomous motion already
exists without recurrence — it is *unsituated*, not absent; and cognition runs for every present NPC
each beat and may commit unaddressed, so the player is not the only mover. The real gap is identical
in both: authored motive versus random motive.

## 4. Round 1 got this wrong

**Traits do not drift.** gamedesign: *"traits drift under `malleability` (handover §3.4), so a law
stored there decays per mind"* (`world_topics_gamedesign.md:47-49`), inherited from the handover.
`trait_pool` has **zero** references outside its `CREATE TABLE` — no accrual code exists in `core/api`
or the migrations. `malleability` is written by genesis (`worldgenesiscommit.go:531-533`), read by
`loadMinds` (`cognitionprompt.go:216`), and printed as `" | malleability: %.2f"` (`:146`). It is
**prompt text, not a mechanism.** The conclusion may survive on real grounds — an unscrubbed norm
sentence in the trait blob reaches every bound mind's prompt — but the stated reason is unverified.

Smaller: gamedesign cites `fn_isolated_npcs` at `schema.sql:2359-2360`, the stale dump §3.1 warns
against; authoritative is `20260726100001_display_name_naming_reach.sql:67`. Extraction cites ref-doc
row 144 as evidence collectives need no engine change; that row actually flags that nothing walls a
group's name.
