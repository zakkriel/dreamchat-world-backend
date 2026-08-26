# What a full world must contain — simarch (mechanical consequence vs decoration)

Checks run first (§1.2). Brief-traceability: every identifier below is a capability name, never a
value from an example. No exemption lists: each is a comparison the engine already performs.
Vocabulary/grammar: the authored column is minted and open, class→number stays ours. Engine gaps are
labelled as programs. Leaf-reaches-a-reader: stated per cluster.

**Verified first, because it reframes everything.** `seed_world_defaults`
(`20260808100001_interruption_tuning.sql:40-58`) gives **every world the identical physics**: one
movement type (`walk` 1.4), one status modifier (`encumbered` −100%), the same five duration
classes, the same three pressure tiers, the same five extent classes, the same journey bands. And
genesis writes **no** `size`, `weight`, `empty_weight`, `max_room` or `weight_modifier` on any
object (`worldgenesiscommit.go:672-695`), and one hardcoded `max_load` = 80 for every actor, from a
constant named for the player (`:62,666`).

So in every generated world: `fn_fact_sheet`'s `weight_kg`, `volume`, `would_encumber` and
`contents` are gated on `size`/`max_room` and return **null** (`schema.sql:1763-1778`); nothing can
ever become `encumbered`; the one seeded modifier is unreachable; and the engine's only worked
state-depends-on-state chain — `contained_by` → effective weight → encumbrance → speed 0 →
over-budget → Journey — is dead. It runs only in the hand-authored benchmark. A generated world
today is a diorama with a conversation in it.

## 1. Clusters, ranked by mechanical consequence

| # | Cluster | What it must write to matter | Destination | Engine work |
|---|---|---|---|---|
| 1 | **Bulk & capacity** — a size class per object, a carry class per body | `size`, `empty_weight`, `max_room`, `weight_modifier`, `max_load` | all Tier-1 today (`tier1.go:5-21`), read by `fn_effective_weight` | **none** — genesis just stops omitting them |
| 2 | **Motion** — named ways of moving, each a pace class | rows in `movement_type` + an actor↔type binding | `movement_type` (seeded `walk` only) | un-hardcode `'walk'` (`20260729100006_move_target.sql:63,66`); pace-class→m/s table (AC-7 forbids the seat emitting 1.4) |
| 3 | **Impediment** — named conditions and what each hinders | `statuses` on the actor + rows in `status_modifier` | live; `fn_effective_speed` reads `statuses` (`20260729100002_move_duration.sql:46-48`) | add `statuses` to `tier1.go` — engine-read but **not** in the registry; `status_modifier.action_type CHECK ('move')` means motion is the only meterable resistance |
| 4 | **Recurrence & schedule** — what is about to happen, and what happens again | `pending_event` rows | read every clock crossing (`ledger.go:122-220`), written by tests only | see §4 |
| 5 | **World temperament** — how restless this world is, as one class | `world_actor_setting.intensity`, `world_actor_config.climb_rate` | live; multiplied inside `fn_pressure_chance` (`schema.sql:2661-2665`) | **none** — pure omission |
| 6 | **Passage** — which ways of moving a barrier stops | portal Tier-1 `impedes[]` | `fn_portal_permits` | founder ruling (handover §5.1); this is §4.2's accepted shape |
| 7 | **Pressure** — a tension class per place | `attrs.tension` | live → beat budget | none |
| 8 | **Knowledge & history** — events, per-holder beliefs, secrets | `canon_event`, `perception_record` | live | none |
| 9 | **Standing & mind** — traits, malleability, speech manner | `personality_core` | live | none |

1–5 are the whole of "alive": 1 and 3 are what *resists*, 2 is what resistance is measured against,
4 and 5 are what *changes without the player*. 6–9 exist and work. Items 1 and 5 need no engine
work at all — the two cheapest wins here.

## 2. Where the GA-2 line sits

**A cluster is legal when it names an engine capability, and illegal when it names a world feature.**
The test is invertible: state the justification with zero reference to any fiction. If you cannot,
it is an ontology.

That is exactly why the existing universals are legal. Two places because one room makes travel
unreachable and the Journey unbuildable (SPEC-030). At least one way because `fn_portal_permits`
needs an edge to read. `hiding` because `fn_private_records` and the I-3 planted-secret test need a
private perception to exist. Each traces to a surface that is otherwise dead code. "Worlds have
collectives" traces to nothing the engine computes — which is the whole difference.

The line is therefore **not** "no structure". It is: *the schema may assert what the engine needs to
run, never what a world is like.* Bulk, motion, impediment and schedule pass because
`fn_effective_weight`, `fn_move_duration_actor`, `fn_effective_speed` and `fn_due_pending` are
already written and currently starved. A caste, a rota or a tide are all *values* those capabilities
carry.

Corollary I will hold against my own earlier position: **`collectives` currently fails this test.**
Group-held perceptions reach no mind and leak to the player (handover §3.4). Membership computes
nothing. It is a naming convenience until it is either a `status` the modifier tables can meter or
the join key for something rendered.

## 3. What I would cut from the reference shape

- **`regard`.** `relationship_state` has zero readers and `[RELATIONSHIPS]` is unrendered
  (`06_context_assembly_spec.md:76,88`). Re-entry condition: render the block first.
- **`role`, `wants`, `doing`.** Three leaves, one missing destination — the cognition prompt. Keep
  whichever survives *after* the prompt gains a section; do not author three inert fields to fill a
  section that does not exist.
- **`collectives[].description`.** A second prose field on a cluster whose first prose field already
  has no reader.
- **`near_future.sets`.** `{attribute:"water", value:"rising"}` is Tier-2 free state. Nothing reads
  it, and Tier-1 grows only when we add a check in code (`tier1.go:3`). The *fire* is real (the
  ruled path fans perceptions); the `sets` payload is decoration.
- **`world.mood`/`ornament` as arrays.** Live schema is single strings
  (`world_genesis.v1.schema.json:24-32`).

## 4. The single biggest omission

**Nothing in the document makes the world recur.** `pending_event` is
`{pending_id, world_id, fire_at_tick, magnitude, payload, status}` with `status ∈
pending|fired|cancelled` (`20260723100003_personality_world.sql`) and **no interval, no re-arm** —
`ledger.go` has no recurrence logic. So the reference file's own brief, *"a tide that comes twice a
day"* (`REFERENCE:25-26`), is inexpressible: the world can be wound up once and then it stops.

Everything else in the document is a noun. `near_future` is the only verb, and it fires exactly
once. A world whose only autonomous behaviour is a single scheduled event plus a uniform random
eruption roll is not alive; it is a set piece with a timer.

This is an **engine program**, not a creation-time patch: recurrence needs a class resolved per world
like every other quantity, a re-arm rule, and a decision about whether a recurring fire re-enters
through the same ruled path. It should be scoped and gated on its own, and the genesis cluster
should be designed against it rather than around it.

Secondary omission, same family: `pending_event.magnitude` is NOT NULL with `CHECK
('small','medium','large')` and drives the §5 beat cut, yet the reference's `near_future` authors no
magnitude class. As shaped it cannot be committed.

## 5. Reference-file failures against §1.2

1. **"Every authored leaf reaches a reader" — the ✅ rows are wrong.** `objects` is marked live, but
   `size_class` (`REFERENCE:102`) reaches nothing: `writeOpeningState` writes descriptor, kind, and
   location-or-`contained_by` only (`worldgenesiscommit.go:672-695`). Likewise `places[].kind`
   (`REFERENCE:50,52,56`) is dropped — handover §3.3 says so, and the table still shows `places` ✅.
   Two inert leaves are hiding inside green rows.
2. **`near_future.sets` is a new inert leaf** — §3 above. This is the `cast[].standing` defect
   repeated inside the newest cluster, which is the one thing this effort exists to stop.
3. **`movement` violates the class discipline it claims.** `pace_class` is right, but the reference
   marks `movement` ⚠️ "tables live + mintable" while handover §3.2 records that `movement_type`
   takes a **raw number** (`FINAL-action-contracts.md:73`). Until the pace-class table exists the
   cluster is not mintable by genesis at all — the ⚠️ understates it to ❌.
4. **`conditions.hinders` presumes only motion can be hindered.** True today by
   `status_modifier.action_type CHECK ('move')`, but the field name promises generality the grammar
   does not have. Either name it for motion or file the widening as its own program.
