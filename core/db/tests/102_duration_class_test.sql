BEGIN;
SELECT plan(3);

-- Living World Task 2: per-world duration_class → seconds config +
-- fn_duration_class_seconds. Self-contained fixture (fixed world uuid, no seed
-- dependency beyond calling seed_world_defaults itself).
-- world: 22222222-2222-2222-2222-222222222222

-- (a) seed_world_defaults(w) populates all five duration_class rows.
SELECT seed_world_defaults('22222222-2222-2222-2222-222222222222');

SELECT is(
  (SELECT count(*)::int FROM duration_class_seconds
   WHERE world_id = '22222222-2222-2222-2222-222222222222'
     AND class IN ('instant','short','medium','long','extremely_long')),
  5,
  '(a) seed_world_defaults(w) seeds all five duration_class rows');

-- (b) fn_duration_class_seconds(w,'long') returns the seeded seconds and is
-- strictly greater than 'short'.
SELECT ok(
  fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','long') = 300::bigint
  AND
  fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','long')
    > fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','short'),
  '(b) fn_duration_class_seconds(w,long)=300 (seeded) and long > short');

-- (c) fn_duration_class_seconds(w,'instant') is > 0 (the floor is non-zero).
SELECT ok(
  fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','instant') > 0,
  '(c) fn_duration_class_seconds(w,instant) > 0 (non-zero floor)');

SELECT * FROM finish();
ROLLBACK;
