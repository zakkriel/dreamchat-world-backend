BEGIN;
SELECT plan(5);

-- (1) accepted event + actor mutation projects via the live trigger
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111',
        'move','t',5,0,'accepted',now(),'public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000a1',
        '00000000-0000-0000-0000-0000000000f1','actor','attrs.location_id','"tavern"'::jsonb,5,0);
SELECT is( (SELECT attrs->>'location_id' FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1'),
       'tavern', 'live trigger applied the actor mutation (absolute set)');
SELECT is( (SELECT last_event_id FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1'),
       'e0000000-0000-0000-0000-0000000000a1'::uuid, 'last_event_id provenance set');

-- (2) idempotency (Rider B): standalone re-apply changes no domain value (excl. volatile updated_at)
CREATE TEMP TABLE _before AS
  SELECT attrs,last_event_id,dirty FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1';
SELECT apply_mutation(m.*) FROM state_mutation m WHERE m.entity_id='00000000-0000-0000-0000-0000000000f1';
SELECT is(
  (SELECT row(attrs,last_event_id,dirty) FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f1'),
  (SELECT row(attrs,last_event_id,dirty) FROM _before),
  'apply_mutation idempotent on domain columns (absolute set)');

-- (3) relationship mutation = no-op stub (SPEC-001)
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000a1',
        '00000000-0000-0000-0000-0000000000f2','relationship','attrs.trust','0.5'::jsonb,5,1);
SELECT is( (SELECT count(*) FROM relationship_state
            WHERE world_id='11111111-1111-1111-1111-111111111111')::int, 0,
       'relationship mutation is a documented no-op (SPEC-001): zero rows');

-- (4) mutation on a non-accepted parent does not project
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000a2','11111111-1111-1111-1111-111111111111',
        'move','p',6,0,'proposed','public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000a2',
        '00000000-0000-0000-0000-0000000000f3','actor','attrs.location_id','"road"'::jsonb,6,0);
SELECT is( (SELECT count(*) FROM actor_state WHERE entity_id='00000000-0000-0000-0000-0000000000f3')::int,
       0, 'mutation on a non-accepted event does not project (doc 03 §3.1)');

SELECT * FROM finish();
ROLLBACK;
