BEGIN;
SELECT plan(5);

-- (a) an UNSEEDED world must fall back, never return NULL — the lookup never fails closed
-- (the fn_duration_class_seconds lesson: an unconfigured world returned NULL and broke the caller).
SELECT ok(
  fn_extent_class_metres('fd000000-ffff-0000-0000-000000000000','small') > 0,
  '(a) an unseeded world falls back to a positive radius, not NULL');

-- (b) a table row WINS over the fallback — proven with a radius no fallback would ever produce.
INSERT INTO extent_class_metres (world_id, class, radius_m)
VALUES ('fd000000-ffff-0000-0000-000000000000','small',12345);
SELECT is(
  fn_extent_class_metres('fd000000-ffff-0000-0000-000000000000','small'),
  12345::numeric,
  '(b) the per-world row overrides the built-in fallback exactly');

-- (c) the classes are strictly increasing — otherwise "bigger place" is meaningless.
SELECT ok(
  fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','intimate')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','small')
  AND fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','small')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','medium')
  AND fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','medium')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','large')
  AND fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','large')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','vast'),
  '(c) intimate < small < medium < large < vast');

-- (d) the engine draws an 8-point outline around the centre.
SELECT is(
  jsonb_array_length(fn_area_around('{"x":100,"y":100}'::jsonb, 50)->'points'),
  8,
  '(d) fn_area_around draws an 8-point outline');

-- (e) THE ROUND TRIP: the drawn outline is a polygon fn_area_polygon accepts, and it contains the
-- centre it was drawn around. This is what rung 2 depends on — a created place must contain the
-- traveller standing at the point it was created for.
SELECT ok(
  fn_area_polygon(jsonb_build_object('area', fn_area_around('{"x":100,"y":100}'::jsonb, 50)))
    @> point(100,100),
  '(e) the engine-drawn footprint contains its own centre');

SELECT * FROM finish();
ROLLBACK;
