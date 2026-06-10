BEGIN;
SELECT plan(12);
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
SELECT * FROM finish();
ROLLBACK;
