BEGIN;
SELECT plan(6);

-- Isolated fixture world: root frame (…a0); inside it BIG (…b1, (0,0)-(1000,1000)) and SMALL
-- (…c1, (100,100)-(200,200)), SMALL entirely inside BIG. DOT (…d1) has no area at all. Mirrors
-- 110_fn_distance_test.sql's standalone style; suffixes are hex-only (uuid columns reject r/s/t).
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('fc000000-0000-0000-0000-0000000000a0','fc000000-ffff-0000-0000-000000000000','location','Root'),
  ('fc000000-0000-0000-0000-0000000000b1','fc000000-ffff-0000-0000-000000000000','location','Big'),
  ('fc000000-0000-0000-0000-0000000000c1','fc000000-ffff-0000-0000-000000000000','location','Small'),
  ('fc000000-0000-0000-0000-0000000000d1','fc000000-ffff-0000-0000-000000000000','location','Dot');

INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('fc000000-0000-0000-0000-0000000000a0','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0}}'::jsonb),
  ('fc000000-0000-0000-0000-0000000000b1','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0},"parent_location_id":"fc000000-0000-0000-0000-0000000000a0",
     "area":{"points":[{"x":0,"y":0},{"x":1000,"y":0},{"x":1000,"y":1000},{"x":0,"y":1000}]}}'::jsonb),
  ('fc000000-0000-0000-0000-0000000000c1','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":150,"y":150},"parent_location_id":"fc000000-0000-0000-0000-0000000000a0",
     "area":{"points":[{"x":100,"y":100},{"x":200,"y":100},{"x":200,"y":200},{"x":100,"y":200}]}}'::jsonb),
  ('fc000000-0000-0000-0000-0000000000d1','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":500,"y":500},"parent_location_id":"fc000000-0000-0000-0000-0000000000a0"}'::jsonb);

-- (a) a point inside BIG only resolves to BIG
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000a0','{"x":800,"y":800}'::jsonb),
  'fc000000-0000-0000-0000-0000000000b1'::uuid,
  '(a) point inside only BIG resolves to BIG');

-- (b) a point inside BOTH resolves to the SMALLER area — the whole point of "smallest wins"
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000a0','{"x":150,"y":150}'::jsonb),
  'fc000000-0000-0000-0000-0000000000c1'::uuid,
  '(b) point inside BIG and SMALL resolves to SMALL (smallest area wins)');

-- (c) a point inside nothing is the open road
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000a0','{"x":5000,"y":5000}'::jsonb),
  NULL,
  '(c) point outside every area returns NULL — the open road');

-- (d) an arealess place never contains anybody, even standing on its exact coordinate
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000a0','{"x":500,"y":500}'::jsonb),
  'fc000000-0000-0000-0000-0000000000b1'::uuid,
  '(d) DOT has no area, so its own coordinate still resolves to BIG, never to DOT');

-- (e) a frame with no children yields NULL rather than erroring
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000d1','{"x":0,"y":0}'::jsonb),
  NULL,
  '(e) a frame with no children returns NULL');

-- (f) fewer than 3 points is not a polygon and must not be treated as one. ok()/IS NULL, not is(): the
-- polygon type has no equality operator in Postgres (only ~= "same as"), so is()'s IS DISTINCT FROM
-- cannot be resolved against a polygon-typed result.
SELECT ok(
  fn_area_polygon('{"area":{"points":[{"x":0,"y":0},{"x":1,"y":1}]}}'::jsonb) IS NULL,
  '(f) a 2-point area is NULL, not a degenerate polygon');

SELECT * FROM finish();
ROLLBACK;
