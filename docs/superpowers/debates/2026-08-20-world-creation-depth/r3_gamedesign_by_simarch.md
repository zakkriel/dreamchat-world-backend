# R3 — gamedesign seat (final synthesis)

Checks first (§1.2): no identifier below traces to one brief; the floor is a comparison against a
function that already exists; class→number stays grammar; three items are named as separate programs.

## 1. Verdicts

**(a) `collectives` is illegal in v1. Cut it.** No mechanism goes dark without it. The
`faction|group` branch of `fn_visible_perceptions` (`schema.sql:3080-3086`) is a trap, not a reader —
group-held rows reach no mind and leak to the player (handover §3.4); nothing walls a group's name
(`schema.sql:1445-1453`, `:2937`). Extraction's "no engine change" (`world_topics_extraction.md:33`)
was true only because the cluster does nothing. The *play* it served survives without it: both felt
moments gamedesign named — an NPC refusing something physically possible, and misreading who someone
is — come from an obligation reaching a mind and from descriptor + naming wall, both live.
**Re-entry condition:** when a group *acts*. `event_participant.entity_kind` already permits
`faction|group` (`schema.sql:3722`), so the day a collective participates in a canon event it has
about-ness, a compendium page (`:1296-1304`), and a reason to exist. Belonging is not that day.

**(b) The floor holds, but "every engine-read key its kind has" is too blunt** — it would demand
`max_room` of every object and make every object a container, which is a shape mandate. Restate it
against the function that already enumerates the keys: **fact-sheet completeness.** `fn_fact_sheet`
branches on exactly three discriminators — `? 'connects'` → `open`/`locked`, `? 'size'` →
`weight_kg`/`volume`/`would_encumber`, `? 'max_room'` → `contents`
(`20260730100001_fact_sheet.sql:76-91`). The floor: **every artifact must satisfy exactly one branch**,
and capacity is required only where something's `contained_by` points at it — closure downward from an
existing reference, which is the one direction referential pull legitimately works.

What it refuses: an artifact that answers no branch — a named nothing the referee is told about and
cannot measure. That is every object in every generated world today
(`worldgenesiscommit.go:672-695`). What it does **not** refuse: a world with no containers, no
conditions, no obligations, no groups. Enforce it as a trigger on `artifact_state`, the
`trg_validate_tension` pattern (`20260723100002_six_type_spine.sql:43-49`), because a trigger cannot be
forgotten by the next landing.

**(c) One generic class table, and fold the two existing ones into it in the same change.**
`extent_class_metres` and `duration_class_seconds` are the same table twice with a renamed value
column; bulk, capacity, carry and pace would make it six — six migrations, six inserts in
`seed_world_defaults` (already nine statements, `20260808100001_interruption_tuning.sql:40-58`), six
resolvers, against a founder constraint that quantity N+1 be one declaration. Fold them:
`fn_extent_class_metres` and `fn_duration_class_seconds` keep their signatures and read the new table,
so no caller changes. Two conventions side by side is the accretion this effort exists to end. Class
*ladders* stay grammar; per-world values stay seeded, since AC-7 forbids the seat emitting a number.

**(d) The commit-time norm router is an exemption list. Reject it.** Routing a free sentence to one of
three destinations needs either a second seat call against a one-call ceiling or string matching, and
handover §3.5 explicitly refused a law API. But the decisive objection is a play objection: **a rule
the engine routes is a rule the engine must understand, and a rule the player feels is a rule a
*person* enforces.** The felt moment is an NPC refusing something physically possible. So an obligation
needs one destination, not three — the mind that will refuse. That is smaller than the router and it
is the same destination presence and opposition already need.

## 2. The answer — what a genesis document must contain

| # | Topic | Destination | Engine work | Beat felt |
|---|---|---|---|---|
| 1 | Place graph, ways, tension class | `location_state`, portal artifacts | none | 1 — the room, its exits, and the first act that doesn't fit the beat |
| 2 | First-sight descriptors | `entity_registry`, `attrs.descriptor` | none | 1 — everyone is a stranger |
| 3 | **What each present mind is doing, what it is trying to get, and at least two of those pulling against each other** | `actor_state.attrs` (Tier-2, free per `verdict.go:148-152`) → **new cognition-prompt section**, scrubbed through `fn_viewer_text` (`schema.sql:3001`) | **yes — one query, one prompt section, no DDL** | 1 you interrupt something; 2–4 someone acts without being addressed |
| 4 | Object bulk — a bulk class per artifact; capacity only where something is contained | `attrs.size`, `empty_weight`, `max_room` → `fn_effective_weight`, fact sheet | class family (c) | 2 — who won't put it down, and what it costs to take it |
| 5 | Carry class per body | `attrs.max_load` — one constant for everyone today (`worldgenesiscommit.go:62,666`) | same family | 2 — the same crate is nothing to one person and pins another |
| 6 | One private thing per person | private `perception_record` | none | 2 — the first deflection; it is also what earns an NPC her own cognition call |
| 7 | One shared past event, remembered differently per holder | `canon_event` + `perception_record` | none | 3–5 — two accounts disagree and the player works the gap; the deepest thing already live |
| 8 | Obligations — one sentence, who it binds | same reader as 3, **never `personality_core.traits`** | shares 3's reader | 4–5 — a refusal of something physically possible |
| 9 | One scheduled change, attributed to an entity that can act, with a magnitude class | `pending_event` (`magnitude` NOT NULL, three-value CHECK) | writer only | 3–5 — something happens that was not about you |
| 10 | Motion vocabulary + actor↔type binding, each a pace class | `movement_type`, `status_modifier` | **blocking program** — `'walk'` hardcoded (`20260729100006_move_target.sql:63,66`) | 3–4 — distance stops being one number |
| 11 | Conditions and what they hinder | `attrs.statuses` → `fn_effective_speed` (`20260729100002_move_duration.sql:46-48`) | needs 10; `statuses` must join `tier1.go:4-22` | 3–4 — a body that cannot do what another can |

**Cut, final:** `collectives` and `collectives[].description` (a); `regard`; `role`/`wants`/`doing` as
three fields — one situation line, not three strings competing for one unbuilt reader;
`places[].kind`; `near_future.sets`; `mood`/`ornament` as arrays. **Separate programs:** passage by
movement type (unruled signature, §5.1), recurrence (`pendingPayload` is `{actor_id, attempt}`,
`ledger.go:16-19`, so an actorless tide is inexpressible before repetition is discussed), senses
(§5.2).

## 3. Build sequence

1. **Item 3 — the per-mind situation section.** No DDL, no migration, no new table: Tier-2 attrs plus
   one query plus one prompt section, scrubbed like every other viewer-facing string. It is the only
   change that alters an NPC decision, and NPC decisions are the entire content of beats 2–5.
2. **Items 4+5 — the class family and the three writes.** Turns every generated world from a diorama
   into a place with weight, and makes the `contained_by` → weight → encumbrance → speed-0 → Journey
   chain reachable outside the hand-authored benchmark for the first time.
3. **Item 9 — genesis becomes `pending_event`'s first production writer.** One-shot only.
4. **Item 8 — obligations onto item 3's reader.** Free once 1 ships.
5. **Item 10/11 — the motion program.** Largest, and it is the founder's own transposition insight.
6. Passage, recurrence, senses — each with its own design and gate.

Steps 1–4 need no founder ruling and no signature change.

## 4. What I believe now that I did not in round 1

**I ranked by arithmetic, and arithmetic hid the thing that matters.** My round-1 list scored clusters
by engine computation, so bulk, motion and "world temperament" came top and the cognition prompt
appeared nowhere. Nothing in those nine clusters changes what an NPC decides. Weight makes a world
solid; it does not make it *move*. Opposition is now item 3 and everything mechanical sits under it.

**"World temperament" was a difficulty slider.** `intensity` multiplies a pressure roll that is a pure
function of ticks since the last eruption (`schema.sql:2661-2665`) and produces an unsituated
intrusion. Authoring it per world makes a world noisier, not more alive — decoration dressed as
mechanism, the exact charge my own paper existed to level. Withdrawn.

**"Zero engine work" was wrong twice.** Bulk and temperament both require the seat to emit raw numbers,
which AC-7 forbids; I applied that rule to motion and waived it for my own top-ranked cluster. I
inventoried destinations and never asked what the authoring path can emit — which is what the
extraction hat is for.

**And reader-existence is orientation-dependent.** Author→reader catches dead leaves; it cannot see a
starved input, because a starved input has no leaf to trace from. The coverage question lives entirely
in the direction that test cannot look.
