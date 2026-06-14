BEGIN;
SELECT plan(7);
SELECT has_table('perception_subject', 'perception_subject exists (ADR-035)');
SELECT col_is_pk('perception_subject', ARRAY['perception_id','entity_id'],
  'perception_subject PK is (perception_id, entity_id)');
SELECT has_column('perception_subject', 'world_id', 'carries world_id from birth (SPEC-009 tenant key)');
SELECT col_type_is('perception_subject', 'world_id', 'uuid', 'world_id is UUID');
SELECT col_not_null('perception_subject', 'world_id', 'world_id NOT NULL');
SELECT fk_ok('perception_subject', 'perception_id', 'perception_record', 'perception_id',
  'perception_id FK → perception_record');
SELECT has_index('perception_subject', 'idx_ps_entity', 'index on entity_id exists');
SELECT * FROM finish();
ROLLBACK;
