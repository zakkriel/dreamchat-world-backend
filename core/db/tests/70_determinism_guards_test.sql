BEGIN;
SELECT plan(5);
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

-- SPEC-002 enforcement: the replay ordering key must be unique BY CONSTRAINT for accepted
-- events, not merely true of the current seed data. Tick 100 / beat_seq 0 is E1's slot.
SELECT throws_ok($$
  INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES ('11111111-1111-1111-1111-111111111111','move','dup of E1 order slot',100,0,
          'accepted',now(),'public','fast_path') $$,
  '23505', NULL,
  'SPEC-002: duplicate (world_id,in_world_tick,beat_seq) among ACCEPTED events is rejected by schema');
SELECT lives_ok($$
  INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq,
                           status, visibility_scope, origin)
  VALUES ('11111111-1111-1111-1111-111111111111','move','proposed may share a slot',100,0,
          'proposed','public','fast_path') $$,
  'SPEC-002: proposed events may share an order slot (uniqueness is canonical-order-only)');
SELECT * FROM finish();
ROLLBACK;
