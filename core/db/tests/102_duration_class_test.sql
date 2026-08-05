BEGIN;
SELECT plan(6);

-- Living World Task 2: per-world duration_class → seconds config +
-- fn_duration_class_seconds. Self-contained fixture (fixed world uuid, no seed
-- dependency beyond calling seed_world_defaults itself).
-- seeded world:   22222222-2222-2222-2222-222222222222
-- unseeded world: 33333333-3333-3333-3333-333333333333 (NEVER passed to
--   seed_world_defaults — exercises the fn_duration_class_seconds built-in
--   CASE fallback, not the table lookup).

-- (a) seed_world_defaults(w) populates all five duration_class rows.
SELECT seed_world_defaults('22222222-2222-2222-2222-222222222222');

SELECT is(
  (SELECT count(*)::int FROM duration_class_seconds
   WHERE world_id = '22222222-2222-2222-2222-222222222222'
     AND class IN ('instant','short','medium','long','extremely_long')),
  5,
  '(a) seed_world_defaults(w) seeds all five duration_class rows');

-- (b) fn_duration_class_seconds(w,'long') returns the seeded seconds, and is
-- strictly greater than 'short' — split into two assertions so a failure
-- pinpoints which half broke.
SELECT is(
  fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','long'),
  300::bigint,
  '(b1) fn_duration_class_seconds(w,long) = 300 (seeded value, table lookup)');

SELECT ok(
  fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','long')
    > fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','short'),
  '(b2) fn_duration_class_seconds(w,long) > fn_duration_class_seconds(w,short) (seeded)');

-- (c) The built-in fallback: an UNSEEDED world (never passed to
-- seed_world_defaults) has zero duration_class_seconds rows, so
-- fn_duration_class_seconds must fall through the COALESCE to the CASE
-- branch rather than the table lookup. First prove the world really is
-- unseeded, then exercise the fallback on it.
SELECT is(
  (SELECT count(*)::int FROM duration_class_seconds
   WHERE world_id = '33333333-3333-3333-3333-333333333333'),
  0,
  '(c0) unseeded world has no duration_class_seconds rows (sanity: fallback path is live)');

SELECT is(
  fn_duration_class_seconds('33333333-3333-3333-3333-333333333333','instant'),
  2::bigint,
  '(c1) fn_duration_class_seconds(unseeded,instant) = 2 via the built-in CASE fallback (non-zero floor)');

SELECT ok(
  fn_duration_class_seconds('33333333-3333-3333-3333-333333333333','long')
    > fn_duration_class_seconds('33333333-3333-3333-3333-333333333333','short'),
  '(c2) fallback CASE also orders long > short on the unseeded world (300 > 5)');

SELECT * FROM finish();
ROLLBACK;
