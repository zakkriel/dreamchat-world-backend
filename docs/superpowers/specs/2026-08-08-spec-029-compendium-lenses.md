# SPEC-029 — Compendium lenses (2026-08-08)

## Scope

This spec upgrades the read-side compendium/timeline SQL lenses from hardcoded placeholders to perception-derived values only.

Non-goals held: no Go changes, no JSON schema changes, no `/carrying` endpoint.

## Inputs read before implementation

- `docs/open-spec-items.md` (`SPEC-029`)
- `docs/superpowers/handovers/2026-08-07-frontend-needs-from-backend.md` (§2.5)
- `core/db/migrations/20260615090001_compendium_read_functions.sql` (full; superseded contract)
- Perception plumbing under `core/db/migrations/`:
  - `fn_visible_perceptions` (`20260614090002_projection_functions.sql`)
  - `fn_perceived_name` (`20260614090002_projection_functions.sql`)
  - `fn_display_name` (`20260726100001_display_name_naming_reach.sql`)
  - `fn_entity_visible` (`20260615090001_compendium_read_functions.sql`)
  - `fn_world_now` (`20260807100003_journey.sql`)
  - `perception_record` (`20260610090004_deltas_epistemic_causal.sql`)
  - `perception_subject` (`20260614090001_perception_subject.sql`)

## Design call 1 — `current_synthesis`

`current_synthesis` is now deterministic text assembled from held perceptions only:

1. Pull only `fn_visible_perceptions(world, viewer)` rows that are `perception_subject`-linked to the target entity.
2. Exclude `world_genesis` rows (identity substrate, not knowledge lens).
3. Sort by `(valid_tick DESC, acquired_tick DESC, perception_id DESC)`.
4. Keep top 3 rows.
5. Render one line per row: `N. [<in_world_label or Tick X>] <content>`.
6. Join lines with `\n`; return `NULL` when no qualifying rows.

Rationale:
- Uses only viewer-held perception records.
- No SQL-side LLM generation.
- Fully deterministic and testable.
- Gives a readable lens without inventing hidden state.

## Design call 2 — `decay.stale`

`decay.stale` is now based on world-time age:

- `stale = (fn_world_now(world_id) - valid_tick) > fn_compendium_decay_horizon_ticks()`
- Horizon lives in one named function: `fn_compendium_decay_horizon_ticks()` (currently `72` ticks).

Rationale for 72:
- It is long enough that very recent perceptions stay fresh.
- It is short enough to age day-old fixture perceptions in the seeded world.
- Single-function constant makes tuning cheap without query surgery.

## Field-by-field mapping

### Actor page (`actor_page/1`)

- `id`: request target id passthrough.
- `perceived_name`: `fn_perceived_name(world, viewer, actor_id)`.
- `perceived_role`: **stubbed `NULL`** (no structured role taxonomy in perception rows).
- `current_synthesis`: `fn_compendium_current_synthesis(world, viewer, actor_id)`.
- `last_known_status`: `fn_compendium_latest_fact(world, viewer, actor_id)` (latest held, non-genesis perception content about actor).
- `known_artifacts`: `fn_compendium_related_entities(..., ARRAY['artifact'])` from co-subjects on the same held perception rows.
- `collected_knowledge_groups`: `fn_collected_knowledge(...)`, now grouped by source event (`group_key = event:<event_id>`), not target id.
- `inline_links`: `fn_compendium_related_entities(..., ARRAY['actor','location','artifact'])` from same-row co-subject evidence.

### Location page (`location_page/1`)

- `id`: request target id passthrough.
- `perceived_name`: `fn_perceived_name(world, viewer, location_id)`.
- `part_of`: **stubbed `NULL`** (no containment edge in perception tables).
- `current_synthesis`: `fn_compendium_current_synthesis(world, viewer, location_id)`.
- `last_known_status`: `fn_compendium_latest_fact(world, viewer, location_id)`.
- `known_areas_inside`: **stubbed `[]`** (co-mention != containment; no reliable inside relation exists in perception rows).
- `key_actors`: `fn_compendium_related_entities(..., ARRAY['actor'])` from co-subjects.
- `collected_knowledge_groups`: `fn_collected_knowledge(...)` event-keyed groups.
- `inline_links`: `fn_compendium_related_entities(..., ARRAY['actor','location','artifact'])`.

### Artifact page (`artifact_page/1`)

- `id`: request target id passthrough.
- `perceived_name`: `fn_perceived_name(world, viewer, artifact_id)`.
- `perceived_type`: **stubbed `NULL`** (no typed artifact classification signal in perception rows).
- `current_synthesis`: `fn_compendium_current_synthesis(world, viewer, artifact_id)`.
- `last_known_location`: `fn_compendium_last_known_location(world, viewer, artifact_id)` from latest held artifact perception that co-subject-links a location.
- `current_holder_owner_access`: **stubbed `NULL`** (carry/holder/owner/access state is not represented in perception schema alone).
- `collected_knowledge_groups`: `fn_collected_knowledge(...)` event-keyed groups.
- `inline_links`: `fn_compendium_related_entities(..., ARRAY['actor','location','artifact'])`.

### Shared item payload (`collected_knowledge_groups[*].items[*]`)

- `perception_id`: `perception_record.perception_id`.
- `content`: `perception_record.content`.
- `epistemic_type`: `perception_record.epistemic_type`.
- `occurred_at_tick`: `perception_record.valid_tick`.
- `display_label`: `canon_event.in_world_label` from `source_event_id`.
- `confidence`: `perception_record.confidence`.
- `decay`: `fn_compendium_decay(world_id, valid_tick, in_world_label)`.
- `source`: `{ epistemic_type, source_event_label }` from the same row/event.

### Timeline (`timeline/1`)

- `records[*].perception_id`: `perception_record.perception_id`.
- `records[*].content`: `perception_record.content`.
- `records[*].epistemic_type`: `perception_record.epistemic_type`.
- `records[*].occurred_at_tick`: `perception_record.valid_tick`.
- `records[*].display_label`: `canon_event.in_world_label`.
- `records[*].confidence`: `perception_record.confidence`.
- `records[*].decay`: `fn_compendium_decay(world_id, valid_tick, in_world_label)`.

## Test coverage added

`core/db/tests/24_compendium_lenses_test.sql` covers:

- Actor/location/artifact `current_synthesis` population.
- Actor/location `last_known_status` / artifact `last_known_location` population.
- Actor `known_artifacts`, location `key_actors`, and all `inline_links` lenses.
- Event-keyed `collected_knowledge_groups`.
- `decay.stale` for both stale and fresh records (page + timeline).
- Perception wall negative case: visible location page for Jonas keeps `current_synthesis` null and `key_actors` empty when he has no held detail perceptions.
