# Round 3 — seat: simarch (final, held by the extraction agent)

## 1. Verdicts on the unresolved

**(a) Collectives: conditionally legal, narrower than either round-1 claim.** Not "no engine
change" (mine, round 1) and not flatly illegal (simarch, round 1) — both papers argued the entity
as a whole; the entity is two separable claims. `canonical_name` + `descriptor` only, minted as
`entity_registry entity_kind='group'`, is legal: it has a real reader today, `fn_display_name`
(falls through straight to `canonical_name` when no `actor_state`/`artifact_state` row exists,
`schema.sql:1445-1453` — verified directly, not inherited), and it is the join-key vocabulary a
norm's `binds` needs to address a set of people without enumerating names. `legibility` is
**illegal** until a naming-wall gate for group entities exists — none does, so `concealed` asserts
a mechanism that isn't there. `description` is **illegal** — no compendium page exists to read it
(open question, not built). Ship the two-field form; the rest waits on its reader.

**(b) The authoring floor holds as a principle, not as precedented.** `trg_validate_tension`
(`20260723100002_six_type_spine.sql:43-49`) fires `IF NEW.attrs ? 'tension'` — it is a **value
guard**, refusing an invalid tension when one is present; it does **not** refuse a location written
with no `tension` key at all. Presence is actually enforced one layer up, in the genesis JSON
schema's `required` array (`world_genesis.v1.schema.json:53`) and Go's `validate()`
(`worldgenesis.go:284-287`), not the DB trigger. So the floor is right — every entity a landing
mints should carry every engine-read key its kind has — but its enforcement mechanism is
schema-`required` + `validate()`, the same layer tension already uses, with a DB trigger as a
*second* guard against writes from outside genesis. Extending `objects[]`'s `required` list to
include a size class the same way `places[]` already requires `tension` is what actually refuses
an under-authored object; the trigger alone never would.

**(c) One class table per unit family, not one generic table and not one per column.**
`extent_class_metres`/`duration_class_seconds` are separate physical tables with separate closed
`CHECK`-enumerated vocabularies (metres, seconds) — a generic table would either collapse two
unrelated scales into one enum (ambiguous) or need a discriminator column, which is "one per
quantity" wearing one table. Grain should follow **unit**, not raw column: `size`,
`empty_weight`, `max_load`, `carried_weight` are all kilograms and share one mass-class table;
`max_room` is a slot count, its own table; pace is m/s, its own table — mirroring how extent
(metres) and duration (seconds) are already kept apart.

**(d) The commit-time norm router does not work as proposed — it is the keyword table §1.2
forbids.** r2's attack lands: inferring a destination from `stated` prose at commit is either a
second seat call (breaks the one-call ceiling) or pattern-matching words for "speech" vs
"passage" — an ad-hoc classifier over open vocabulary, exactly the disguised ontology GA-2
forbids. The fix is not classification, it's **structure**: give the norm the same closed,
`oneOf`-shaped discrimination `objects[].where` already uses (`in_place` XOR `carried_by`,
`world_genesis.v1.schema.json:190-197`) — the seat states, by which reference field is present,
whether this norm binds a speech-act or a place/way, never inferred from prose. No new seat call,
no exemption list: a structural choice the model makes explicitly, read by the engine as grammar.

## 2. The answer — definitive genesis topics

| Topic | Destination | Engine work | Felt at |
|---|---|---|---|
| Place graph + ways | `entity_registry`/`location_state`/`artifact_state` | none, live | beat 1 — what you can name |
| Tension per place | `attrs.tension` → beat budget | none, live | first act too big for the beat |
| One private thing per person | `perception_record`, gates isolated cognition | none, live | first deflection |
| Per-holder history | `canon_event`/`perception_record` | none, live | beats 3-5, accounts diverge |
| Present intention + opposition | cognition prompt, new per-mind section | **yes — cheapest real lift**, one section + one query, no column | beat 2-4, unaddressed act |
| Norms (structural, §1(d)) | speech-constraint reader when bound to speech; else knowledge/history | **yes**, decomposer-adjacent, no new table | the moment a rule is tested |
| Collectives, narrow form | `entity_registry entity_kind='group'`, name only | none — genesis stops omitting the call | beat 1, first-sight descriptor |
| Scheduled change | `pending_event`, genesis first writer, magnitude class | small — needs an attributable actor (`pendingPayload{actor_id,attempt}`, `ledger.go:16-19`) | beats 3-5, happens whether you came or not |
| Object bulk / carry | mass-class table (§1c) | small, one table + resolution | beat 2, leverage |
| Statuses + motion | `tier1.go` registration; pace-class table; un-hardcode `'walk'` | larger, several call sites | first obstacle that differs per person |
| Passage by movement type | portal `impedes` | larger + founder ruling (§5.1) | same stair, different outcome |
| Recurrence | its own program | out of scope here | long-session believability, not 5 beats |

Never author into: `sensory_mode`, `distortion_level`, `relationship_state`,
`collectives[].legibility|description` — dead readers, class C.

## 3. Build order, cheapest real gain first

1. Present intention + opposition (no schema/DB change, largest play delta).
2. Norm structural discrimination, riding the same reader.
3. Collectives, narrow form (now has a consumer via norm `binds`).
4. Mass-class table → object bulk/carry, restores leverage.
5. `pending_event` writer + magnitude, with an attributable actor.
6. Statuses into Tier-1; pace-class table; motion binding.
7. Passage/`impedes`, gated on the founder ruling.
8. Recurrence — scoped, gated, built against its own design, not here.

## 4. What changed across the three hats

I no longer believe referential pull is an elicitation force. Round 1 me proposed it as the
answer to "coverage without a spine"; round 2 was right that every edge in this document points
downward toward what the schema already requires, so a minimal world — two places, one person, one
secret — is a global optimum under that pressure, not a floor with room to grow. I no longer
believe schema shape is the starting question at all. The real method, learned from wearing
simarch and gamedesign, is: read the commit path and the cognition prompt for what already has a
consumer and gets nothing, then design the smallest authored surface that feeds exactly that. A
schema drafted before that reading — mine in round 1, `regard`/`role` in the reference file — is
indistinguishable from decoration until proven otherwise, no matter how genre-agnostic its field
names are.
