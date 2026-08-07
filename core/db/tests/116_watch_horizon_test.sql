BEGIN;
SELECT plan(3);

-- Rung 3 Task 0: watch horizon gets its own dial (correction to the rung 2 journey work,
-- which borrowed fn_duration_class_seconds('extremely_long') — a duration class is not a
-- horizon). watch_horizon(world_id PK, horizon_seconds) + fn_watch_horizon_seconds, same
-- per-world-table-with-fallback shape as fn_duration_class_seconds/fn_journey_legs.
-- unseeded world: 55555555-5555-5555-5555-555555555555 (never passed to
--   seed_world_defaults — exercises the built-in COALESCE fallback, not the table row).
-- seeded world:   44444444-4444-4444-4444-444444444444
-- override world: 66666666-6666-6666-6666-666666666666 (a per-world row, no seeding)

-- (a) an unseeded world falls back to a positive horizon — never NULL. ok() treats a NULL
-- condition as a failure, so "> 0" alone proves both "positive" and "not null".
SELECT ok(
  fn_watch_horizon_seconds('55555555-5555-5555-5555-555555555555'::uuid) > 0,
  '(a) an unseeded world falls back to a positive watch horizon (never NULL)');

-- (b) a per-world row overrides the fallback exactly.
INSERT INTO watch_horizon (world_id, horizon_seconds)
VALUES ('66666666-6666-6666-6666-666666666666', 43200);

SELECT is(
  fn_watch_horizon_seconds('66666666-6666-6666-6666-666666666666'::uuid),
  43200::bigint,
  '(b) a per-world watch_horizon row overrides the built-in fallback exactly');

-- (c) a freshly seeded world has the row.
SELECT seed_world_defaults('44444444-4444-4444-4444-444444444444');

SELECT is(
  (SELECT count(*)::int FROM watch_horizon
   WHERE world_id = '44444444-4444-4444-4444-444444444444'),
  1,
  '(c) seed_world_defaults(w) seeds the watch_horizon row');

SELECT * FROM finish();
ROLLBACK;
