BEGIN;
SELECT plan(5);

-- Station F / FINAL-action-contracts.md §2 (move contract): real duration = distance / effective_speed,
-- effective_speed = base_speed(movement_type) * Pi(1 + modifier_percent/100) over the actor's ACTIVE
-- statuses. Floor -100% (speed 0 => infinite duration => never fits => prevention EMERGES from the
-- arithmetic, never a special case). NO upper cap. Self-contained fixture (fixed uuids, no seed
-- dependency), reusing the §3 nested-frame geometry from 110_fn_distance_test:
--   district (root, coord {0,0})
--     ├─ tavern (child, coord {100,0} in district)
--     └─ alley  (child, coord {900,0} in district)      → tavern↔alley distance = |900-100| = 800 m
-- actor A stands at the tavern. Its ACTIVE statuses live in actor_state.attrs->'statuses' (a JSON array
-- of status_type_id strings -- the read model this migration introduces). Seeded vocabulary: walk=1.4;
-- modifiers limping(-30) and encumbered(-100) on walk.

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('fb000000-0000-0000-0000-0000000000d1', 'fb000000-ffff-0000-0000-000000000000', 'location', 'district'),
  ('fb000000-0000-0000-0000-0000000000c1', 'fb000000-ffff-0000-0000-000000000000', 'location', 'tavern'),
  ('fb000000-0000-0000-0000-0000000000c2', 'fb000000-ffff-0000-0000-000000000000', 'location', 'alley'),
  ('fb000000-0000-0000-0000-0000000000a1', 'fb000000-ffff-0000-0000-000000000000', 'actor',    'A');

INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('fb000000-0000-0000-0000-0000000000d1', 'fb000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0},"extent":{"w":2000,"h":2000}}'::jsonb),
  ('fb000000-0000-0000-0000-0000000000c1', 'fb000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":100,"y":0},"parent_location_id":"fb000000-0000-0000-0000-0000000000d1"}'::jsonb),
  ('fb000000-0000-0000-0000-0000000000c2', 'fb000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":900,"y":0},"parent_location_id":"fb000000-0000-0000-0000-0000000000d1"}'::jsonb);

-- actor A at the tavern, no statuses yet (attrs.statuses absent → treated as the empty set).
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('fb000000-0000-0000-0000-0000000000a1', 'fb000000-ffff-0000-0000-000000000000',
   '{"location_id":"fb000000-0000-0000-0000-0000000000c1"}'::jsonb);

-- Seeded vocabulary for this world: the default walk type + the two modifiers we exercise.
INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps) VALUES
  ('fb000000-ffff-0000-0000-000000000000', 'walk', 1.4);

-- (a) No active statuses → effective speed is exactly the walk base, 1.4 m/s.
SELECT is(
  fn_effective_speed('fb000000-ffff-0000-0000-000000000000',
                     'fb000000-0000-0000-0000-0000000000a1', 'walk'),
  1.4::numeric,
  '(a) fn_effective_speed(A, walk) = 1.4 with no active statuses');

-- (c) Duration is distance/speed, engine-owned. Checked here in the UNMODIFIED walker state so the
-- value is the clean base case: CEIL(800 / 1.4) = CEIL(571.42...) = 572 ticks (1 tick = 1 s).
-- (Ordered before (b) deliberately: (c) is the no-status base; (b)/(d) then escalate A's statuses.)
SELECT is(
  fn_move_duration_actor('fb000000-ffff-0000-0000-000000000000',
                         'fb000000-0000-0000-0000-0000000000a1',
                         'fb000000-0000-0000-0000-0000000000c1',
                         'fb000000-0000-0000-0000-0000000000c2'),
  572::bigint,
  '(c) fn_move_duration_actor(A, tavern, alley) = CEIL(800/1.4) = 572');

-- (b) Mint a limping modifier (-30% on walk) and mark A limping. 1.4 * (1 - 0.30) = 1.4 * 0.70 = 0.98.
INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
VALUES ('fb000000-ffff-0000-0000-000000000000', 'limping', 'move', 'walk', -30);
UPDATE actor_state SET attrs = attrs || '{"statuses":["limping"]}'::jsonb
  WHERE world_id = 'fb000000-ffff-0000-0000-000000000000'
    AND entity_id = 'fb000000-0000-0000-0000-0000000000a1';
SELECT is(
  fn_effective_speed('fb000000-ffff-0000-0000-000000000000',
                     'fb000000-0000-0000-0000-0000000000a1', 'walk'),
  0.98::numeric,
  '(b) limping (-30%) → fn_effective_speed = 1.4 * 0.70 = 0.98');

-- (d) Escalate A to encumbered (-100% on walk). effective speed floors at 0 → duration is "infinite":
-- fn_move_duration_actor returns max bigint so "never fits any budget" EMERGES from the arithmetic
-- (§2), never a special "tied/blocked" branch.
INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
VALUES ('fb000000-ffff-0000-0000-000000000000', 'encumbered', 'move', 'walk', -100);
UPDATE actor_state SET attrs = attrs || '{"statuses":["encumbered"]}'::jsonb
  WHERE world_id = 'fb000000-ffff-0000-0000-000000000000'
    AND entity_id = 'fb000000-0000-0000-0000-0000000000a1';
SELECT ok(
  fn_effective_speed('fb000000-ffff-0000-0000-000000000000',
                     'fb000000-0000-0000-0000-0000000000a1', 'walk') = 0::numeric
  AND
  fn_move_duration_actor('fb000000-ffff-0000-0000-000000000000',
                         'fb000000-0000-0000-0000-0000000000a1',
                         'fb000000-0000-0000-0000-0000000000c1',
                         'fb000000-0000-0000-0000-0000000000c2') = 9223372036854775807::bigint,
  '(d) encumbered (-100%) → speed 0 AND duration = max bigint (blocked by arithmetic)');

-- (e) Legacy fn_move_duration (unchanged signature, no actor param) = walk 1.4, no statuses:
-- CEIL(800/1.4) = 572. Keeps apply_beat working with REAL distances (decision 6).
SELECT is(
  fn_move_duration('fb000000-ffff-0000-0000-000000000000',
                   'fb000000-0000-0000-0000-0000000000c1',
                   'fb000000-0000-0000-0000-0000000000c2'),
  572::bigint,
  '(e) legacy fn_move_duration(tavern, alley) = CEIL(800/1.4) = 572 (walk, no statuses)');

SELECT * FROM finish();
ROLLBACK;
