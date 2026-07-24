BEGIN;
SELECT plan(6);

-- Fixed UUIDs for reproducibility
-- World: a5000000-ffff-0000-0000-000000000000
-- Location entity for tension test: a5000000-0000-0000-0000-000000000001
-- Canon event: a5000000-0000-0000-0000-000000000010

-- (a) Inserting a canon_event with event_type='AttributeChanged' succeeds.
-- canon_event columns: event_id, world_id, event_type, summary, in_world_tick, beat_seq,
--                      status, accepted_at, visibility_scope, origin
SELECT lives_ok($$
  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES ('a5000000-0000-0000-0000-000000000010',
          'a5000000-ffff-0000-0000-000000000000',
          'AttributeChanged',
          'test row',
          1, 0,
          'accepted', now(), 'public', 'freeform')
$$, 'canon_event accepts AttributeChanged as event_type');

-- (b) seed_world_defaults plants walk=1.4 and encumbered -100 on walk.
SELECT lives_ok($$
  SELECT seed_world_defaults('a5000000-ffff-0000-0000-000000000000')
$$, 'seed_world_defaults runs without error');

SELECT is(
  (SELECT base_speed_mps FROM movement_type
   WHERE world_id = 'a5000000-ffff-0000-0000-000000000000'
     AND movement_type_id = 'walk'),
  1.4::numeric,
  'seed_world_defaults plants walk base_speed_mps = 1.4'
);

SELECT is(
  (SELECT modifier_percent FROM status_modifier
   WHERE world_id        = 'a5000000-ffff-0000-0000-000000000000'
     AND status_type_id  = 'encumbered'
     AND movement_type_id = 'walk'),
  (-100)::numeric,
  'seed_world_defaults plants encumbered modifier_percent = -100 on walk'
);

-- (c) A bad tension value on location_state raises (trigger rejection).
-- location_state PK is entity_id (uuid); also needs world_id.
SELECT throws_ok($$
  INSERT INTO location_state (entity_id, world_id, attrs)
  VALUES ('a5000000-0000-0000-0000-000000000001',
          'a5000000-ffff-0000-0000-000000000000',
          '{"tension":"apocalyptic"}'::jsonb)
$$, NULL, NULL,
  'location_state rejects bad tension value via trigger');

-- (d) A good tension value inserts fine.
SELECT lives_ok($$
  INSERT INTO location_state (entity_id, world_id, attrs)
  VALUES ('a5000000-0000-0000-0000-000000000001',
          'a5000000-ffff-0000-0000-000000000000',
          '{"tension":"tense"}'::jsonb)
$$, 'location_state accepts valid tension value tense');

SELECT * FROM finish();
ROLLBACK;
