# Round 2 — simarch seat (rotated)

**Checks run first (§1.2).** Brief-traceability: every name below is a capability, no value from any
brief. No exemption lists: each is a comparison. Vocabulary/grammar: authored side minted and open,
class→number ours. Engine gap ⇒ program: labelled. Leaf-reaches-a-reader: the organising principle.

**Confirmed, not re-derived:** legality = absence leaves a mechanism dark; coverage = starved inputs.
The five-field cut stands. The diorama finding I verified independently in the authoritative migration
(handover §3.1): `weight_kg`/`volume`/`would_encumber` gate on `art.attrs ? 'size'`
(`20260730100001_fact_sheet.sql:76,79,82`), `contents` on `? 'max_room'` (`:91`), and genesis writes
descriptor, kind, and location-or-`contained_by` only (`worldgenesiscommit.go:672-695`), with
`max_load` a single constant reused for every body (`:62,666,786,804`). It holds. It is the most
important finding in round 1.

## 1. Attack on simarch's round 1

**His table asks the seat to emit five raw numbers, which his own §2 forbids.** Cluster 1 says write
`size`, `empty_weight`, `max_room`, `weight_modifier`, `max_load`, engine work **"none — genesis just
stops omitting them"**. Every one of those is a number. Baseline AC-7 forbids the seat emitting any,
and his §2 says class→number "stays ours". There is no `size_class` table, no carry-class table —
`extent_class_metres` and `duration_class_seconds` are the only precedents. So cluster 1 is not
zero-work; it needs one new per-world class family plus resolution in the commit path. He caught this
exact error in the reference file's motion row (his §5 item 3: `movement_type` takes a raw number, so
"⚠️ understates it to ❌") and missed it in his own top-ranked cluster. **The extraction angle is what
exposes it:** he inventoried destinations and never asked what the authoring path emits.

**Ranking by arithmetic makes decision inputs invisible.** He scores clusters by mechanical
consequence, defined as engine computation — so items 8 and 9 get "live, none" and the paper moves on.
But the cognition prompt is a starved input with a real consumer: it renders location, tension,
roster, own traits, private records, public moment, fact sheet, imminent act
(`cognitionprompt.go:119-178`) and nothing about what any mind is *trying to get*. Nothing in his nine
clusters changes an NPC's decision. Arithmetic is measurable so it ranks; decision inputs aren't so
they vanish — and NPC decisions are the only thing that produces beats 2-5. **The gamedesign angle
exposes it**, and both peers put it first.

**"World temperament" is a difficulty slider wearing a cluster's clothes.** `intensity` multiplies
inside a pressure roll that is a pure function of ticks since the last eruption
(`schema.sql:2661-2665`); the resulting intrusion is unsituated — attributed to any world entity the
seat likes (`world_actor.txt:3`). Authoring it per world makes a world *noisier*, not more alive.
That is decoration dressed as mechanism, which is the charge his own paper exists to level.

## 2. Starved engine inputs — the enumerated list

A reader exists and receives nothing. That is the whole test.

1. **Object bulk** → `fn_volume`, `fn_effective_weight`, and three fact-sheet fields
   (`20260730100001_fact_sheet.sql:76-82`). Author: one bulk class per object. **Engine work:**
   class→number table + resolution.
2. **Container capacity and self-weight** → `fn_effective_weight`'s recursion and `apply_event`'s
   volume floor (`schema.sql:190-200`), fact-sheet `contents`. Author: a capacity class on things that
   hold things. **Engine work:** same table. (1+2 are one authored family, not five leaves — smaller
   than his shape.)
3. **Carry capacity per body** → `would_encumber`, the eager encumbrance rule. Author: a carry class
   per body. **Engine work:** same table; today one constant for everyone.
4. **Actor statuses** → `fn_effective_speed` (`20260729100002_move_duration.sql:46-48`). Author: named
   conditions per body. **Engine work:** yes, and `statuses` must join `tier1.go` — engine-read today,
   unregistered (`tier1.go:4-22`).
5. **Movement types + an actor↔type binding** → `fn_move_duration_actor`, which hardcodes `'walk'`
   (`20260729100006_move_target.sql:63,66`). Author: named ways of moving, each a pace class.
   **Engine work:** yes, blocking. Do not mint before the binding.
6. **The scheduled-future ledger** → `fn_due_pending`/`fireDuePending` (`ledger.go:122-220`), written
   by tests only. Author: one imminent change, a magnitude class, attributed to an entity that can
   act. **Engine work:** none for one-shot; genesis becomes first writer.
7. **The cognition prompt's absent per-mind section** → the only channel that changes behaviour.
   Author: one line of present intention per mind, plus obligations as non-trait text (traits drift,
   handover §3.4). **Engine work:** yes — one prompt section, and it is the destination all five cut
   fields were competing for (`REFERENCE:152-154`).
8. **Portal passage by movement type** → `fn_portal_permits`, which takes no mover. Author: what a
   barrier impedes. **Engine work:** yes; unruled signature change (handover §5.1) — a program, not a
   creation-time patch.

Not starved, and never author into them: `relationship_state`, and
`sensory_mode`/`distortion_level` (an abandoned program — handover §3.3, §5.2). All have zero readers,
and a reader that does not exist cannot be starved; it is dead.

## 3. The two omissions are one defect, seen from opposite ends

Not equivalent, and not independent. The world already has autonomous motion — the pressure roll fires
without the player — so "the world stops after one event" is imprecise. The precise statement:
**every autonomous mover in this engine is either recurring but unsituated (the pressure roll) or
situated but one-shot (`pending_event`, which has `fire_at_tick`, `magnitude`, `status ∈
pending|fired|cancelled` and no interval or re-arm).** Nothing is both.

That reframes the adjudication. Recurrence alone is a metronome: the water rises on schedule, no one's
interests are crossed, no beat is generated. Opposition alone is one confrontation, then stillness.
**Opposition is more load-bearing**, structurally rather than by taste: it rides the prompt section
item 7 already requires, while recurrence changes the ledger's shape and re-entry path — his own §4
rightly calls that a program. So: opposition first on the back of item 7; recurrence scoped and gated
separately, with the cluster designed against it. His secondary point is right and cheap — the
reference authors no magnitude class, and that column is NOT NULL with a three-value CHECK, so as
shaped it cannot commit at all.

## 4. Round-1 factual corrections

1. **"Bulk & capacity: engine work none" is wrong** — §1. It is the smallest engine program here, not
   zero.
2. **Extraction's collectives row is wrong.** It states "**none** — genesis-side only, no engine
   change" while citing the reference. Handover §3.4: group-held perceptions reach no mind and leak to
   the player, and nothing walls a group's name (`schema.sql:1445-1453`, `:2937`). Minting a group with
   no reader is the defect this effort exists to stop — simarch's own §2 corollary says so, and the
   two papers contradict each other on the same page of evidence.
3. **Gamedesign's "objects as leverage — live" is wrong**, and the diorama finding is what breaks it.
   With no `size` and no `max_room`, the fact sheet returns null for weight, volume, encumbrance and
   contents, so an object is a name that can be moved, not leverage. That tier-A row should be ⚠️,
   dependent on item 1.
4. **Handover §3.1's example is inaccurate.** It says `fn_fact_sheet` and `fn_portal_permits` are "not
   in" `schema.sql`; both are (`schema.sql:1732`, `:2625`). The gating is identical to the
   authoritative migration, so no round-1 conclusion moves — but "absent from the dump" is not a safe
   staleness heuristic, and simarch's `schema.sql` citation for the fact sheet was legitimate.
