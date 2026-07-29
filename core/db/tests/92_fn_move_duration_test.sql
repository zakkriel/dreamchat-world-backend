BEGIN;
SELECT plan(3);
-- Station F / §2: fn_move_duration is no longer a flat 5 — it is the REAL legacy wrapper,
-- CEIL(fn_distance / 1.4) (walk 1.4 m/s, no statuses; decision 6, apply_beat compat). So this test now
-- needs a real §3 geometry: two sibling locations under a shared parent, each with a coordinate, so
-- fn_distance is defined (bare uuids with no location_state → fn_distance NULL → duration NULL). Fixture
-- mirrors 110/111:
--   district (root, {0,0})
--     ├─ tavern (child, {100,0})
--     └─ square (child, {900,0})   → distance = |900-100| = 800 m → CEIL(800/1.4) = 572 ticks.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('e5ffffff-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111','location','district-92'),
  ('e5ffffff-0000-0000-0000-000000000010','11111111-1111-1111-1111-111111111111','location','tavern-92'),
  ('e5ffffff-0000-0000-0000-000000000011','11111111-1111-1111-1111-111111111111','location','square-92');
INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('e5ffffff-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111',
   '{"coordinates":{"x":0,"y":0},"extent":{"w":2000,"h":2000}}'::jsonb),
  ('e5ffffff-0000-0000-0000-000000000010','11111111-1111-1111-1111-111111111111',
   '{"coordinates":{"x":100,"y":0},"parent_location_id":"e5ffffff-0000-0000-0000-0000000000d1"}'::jsonb),
  ('e5ffffff-0000-0000-0000-000000000011','11111111-1111-1111-1111-111111111111',
   '{"coordinates":{"x":900,"y":0},"parent_location_id":"e5ffffff-0000-0000-0000-0000000000d1"}'::jsonb);

-- RECOMPUTED (was flat 5): tavern→square is 800 m of walking → CEIL(800/1.4) = 572 ticks.
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111',
          'e5ffffff-0000-0000-0000-000000000010'::uuid,
          'e5ffffff-0000-0000-0000-000000000011'::uuid), 572::bigint,
          'tavern→square (800 m) costs CEIL(800/1.4) = 572 ticks');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111',
          'e5ffffff-0000-0000-0000-000000000011'::uuid,
          'e5ffffff-0000-0000-0000-000000000010'::uuid), 572::bigint,
          'symmetric: square→tavern also 572');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111',
          'e5ffffff-0000-0000-0000-000000000010'::uuid,
          'e5ffffff-0000-0000-0000-000000000010'::uuid), 0::bigint,
          'same place = 0 ticks (distance 0)');
SELECT * FROM finish();
ROLLBACK;
