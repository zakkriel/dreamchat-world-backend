BEGIN;
SELECT plan(7);

-- Fixed UUIDs for this test
-- world: fa000000-ffff-0000-0000-000000000000
-- actor: fa000000-0000-0000-0000-000000000001
-- event: fa000000-0000-0000-0000-000000000002
-- pending: fa000000-0000-0000-0000-000000000003

-- ── Block (a): personality_core + trait_provenance ──────────────────────────

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('fa000000-0000-0000-0000-000000000002',
        'fa000000-ffff-0000-0000-000000000000',
        'AttributeChanged',
        'backstory: gained courage',
        0, 0,
        'accepted', now(), 'public', 'backstage');

INSERT INTO personality_core (world_id, actor_id, traits, malleability)
VALUES ('fa000000-ffff-0000-0000-000000000000',
        'fa000000-0000-0000-0000-000000000001',
        '{"schema_version":"traits/1","courage":0.8}',
        0.3);

SELECT is(
  (SELECT malleability FROM personality_core
   WHERE actor_id = 'fa000000-0000-0000-0000-000000000001'),
  0.3::numeric,
  'personality_core: malleability 0.3 round-trips correctly');

INSERT INTO trait_provenance (world_id, actor_id, trait_key, event_id)
VALUES ('fa000000-ffff-0000-0000-000000000000',
        'fa000000-0000-0000-0000-000000000001',
        'courage',
        'fa000000-0000-0000-0000-000000000002');

SELECT is(
  (SELECT trait_key FROM trait_provenance
   WHERE actor_id = 'fa000000-0000-0000-0000-000000000001'
     AND event_id = 'fa000000-0000-0000-0000-000000000002'),
  'courage',
  'trait_provenance: row links actor trait to canon_event');

SELECT throws_ok(
  $$ INSERT INTO personality_core (world_id, actor_id, traits, malleability)
     VALUES ('fa000000-ffff-0000-0000-000000000000',
             'fa000000-0000-0000-0000-000000000099',
             '{"schema_version":"traits/1"}',
             1.5) $$,
  '23514',
  NULL,
  'personality_core: malleability CHECK rejects 1.5 (above 1)');

-- ── Block (b): pending_event + fn_due_pending ────────────────────────────────

INSERT INTO pending_event (pending_id, world_id, fire_at_tick, magnitude, payload)
VALUES ('fa000000-0000-0000-0000-000000000003',
        'fa000000-ffff-0000-0000-000000000000',
        10,
        'medium',
        '{"desc":"storm arrives"}');

SELECT is(
  (SELECT count(*)::int FROM fn_due_pending('fa000000-ffff-0000-0000-000000000000', 9)),
  0,
  'fn_due_pending(world, 9) returns 0 rows before fire_at_tick 10');

SELECT is(
  (SELECT count(*)::int FROM fn_due_pending('fa000000-ffff-0000-0000-000000000000', 10)),
  1,
  'fn_due_pending(world, 10) returns 1 row at fire_at_tick 10');

-- ── Block (c): world_pressure PK uniqueness ───────────────────────────────────

INSERT INTO world_pressure (world_id, tier)
VALUES ('fa000000-ffff-0000-0000-000000000000', 'small');

SELECT throws_ok(
  $$ INSERT INTO world_pressure (world_id, tier)
     VALUES ('fa000000-ffff-0000-0000-000000000000', 'small') $$,
  '23505',
  NULL,
  'world_pressure: duplicate (world_id, tier) raises PK violation');

-- ── trait_pool smoke ──────────────────────────────────────────────────────────

INSERT INTO trait_pool (world_id, actor_id, trait_key, threshold)
VALUES ('fa000000-ffff-0000-0000-000000000000',
        'fa000000-0000-0000-0000-000000000001',
        'courage',
        1.0);

SELECT is(
  (SELECT accrued FROM trait_pool
   WHERE actor_id = 'fa000000-0000-0000-0000-000000000001'
     AND trait_key = 'courage'),
  0::numeric,
  'trait_pool: accrued defaults to 0');

SELECT * FROM finish();
ROLLBACK;
