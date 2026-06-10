BEGIN;
SELECT plan(23);
SELECT has_role('maintainer', 'maintainer role exists');
SELECT has_role('app_reader', 'app_reader role exists');
SELECT has_table('canon_event',       'canon_event exists');
SELECT has_table('event_participant', 'event_participant exists');
SELECT col_is_pk('canon_event', 'event_id', 'canon_event PK is event_id');
SELECT has_column('canon_event', 'in_world_tick', 'canon_event has in_world_tick (logical time, ADR-030)');
SELECT col_type_is('canon_event', 'in_world_tick', 'bigint', 'in_world_tick is BIGINT');
SELECT hasnt_column('canon_event', 'in_world_time', 'no TIMESTAMPTZ fictional clock (ADR-030)');
SELECT col_type_is('canon_event', 'recorded_at', 'timestamp with time zone', 'recorded_at is TIMESTAMPTZ (B-5)');
SELECT columns_are('canon_event', ARRAY[
  'event_id','world_id','scene_id','beat_id','event_type','summary','payload','schema_version',
  'in_world_tick','in_world_label','beat_seq','temporal_uncertainty','recorded_at','accepted_at',
  'status','visibility_scope','confidence','origin','template_id','source_refs','superseded_by'],
  'canon_event columns match doc 03 §1.1 exactly (column-by-column)');
SELECT has_table('entity_registry', 'entity_registry exists');
SELECT col_is_pk('entity_registry', 'entity_id', 'entity_registry PK is entity_id');
SELECT has_table('state_mutation',      'state_mutation exists');
SELECT has_table('provenance_edge',     'provenance_edge exists (deployed, unused in 0A — ADR-008)');
SELECT has_table('perception_record',   'perception_record exists');
SELECT has_table('causal_bundle',       'causal_bundle exists (schema-ready, unused — ADR-008)');
SELECT has_table('causal_bundle_input', 'causal_bundle_input exists (schema-ready, unused — ADR-008)');
SELECT col_type_is('perception_record', 'acquired_tick', 'bigint', 'perception acquired_tick is logical (ADR-030)');
SELECT has_table('actor_state',        'actor_state exists');
SELECT has_table('location_state',     'location_state exists');
SELECT has_table('artifact_state',     'artifact_state exists');
SELECT has_table('relationship_state', 'relationship_state exists');
SELECT hasnt_column('relationship_state', 'updated_at',
       'relationship_state has no updated_at (doc 03 §1.5) — no volatile col to exclude in I-1');
SELECT * FROM finish();
ROLLBACK;
