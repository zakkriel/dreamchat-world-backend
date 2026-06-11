BEGIN;
SELECT plan(3);
SELECT is( (SELECT count(*) FROM (
              SELECT world_id,in_world_tick,beat_seq FROM canon_event WHERE status='accepted'
              GROUP BY world_id,in_world_tick,beat_seq HAVING count(*)>1) dup)::int,
       0, 'Rider C: (world_id,in_world_tick,beat_seq) unique across accepted events');
SELECT is( (SELECT count(*) FROM (
              SELECT entity_id,attribute_path,valid_from_tick,valid_from_seq FROM state_mutation
              GROUP BY entity_id,attribute_path,valid_from_tick,valid_from_seq HAVING count(*)>1) dup)::int,
       0, 'Rider C: mutation order key unique per (entity,attribute)');
SELECT is( (SELECT count(*) FROM relationship_state)::int, 0,
       'SPEC-001: relationship_state is empty in 0A (intentional vacuous satisfaction)');
SELECT * FROM finish();
ROLLBACK;
