BEGIN;
SELECT plan(3);

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000de','11111111-1111-1111-1111-111111111111',
        'move','dg',7,0,'accepted',now(),'public','fast_path');

INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000de',
        '00000000-0000-0000-0000-00000000000a','actor','attrs.location_id','"x"'::jsonb,7,0);
SELECT throws_ok($$ DELETE FROM state_mutation WHERE event_id='e0000000-0000-0000-0000-0000000000de' $$,
  NULL,NULL,'DELETE on state_mutation raises (canon table, ADR-006)');

INSERT INTO perception_record (world_id,holder_id,source_event_id,content,epistemic_type,acquired_tick,valid_tick)
VALUES ('11111111-1111-1111-1111-111111111111','00000000-0000-0000-0000-00000000000a',
        'e0000000-0000-0000-0000-0000000000de','p','direct',7,7);
SELECT throws_ok($$ DELETE FROM perception_record WHERE source_event_id='e0000000-0000-0000-0000-0000000000de' $$,
  NULL,NULL,'DELETE on perception_record raises (ADR-006)');

INSERT INTO provenance_edge (derived_id,derived_kind,source_id,source_kind,how_type)
VALUES (gen_random_uuid(),'perception',gen_random_uuid(),'event','derived_from');
SELECT throws_ok($$ DELETE FROM provenance_edge $$, NULL,NULL,'DELETE on provenance_edge raises (canon lineage)');

SELECT * FROM finish();
ROLLBACK;
