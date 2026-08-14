-- =====================================================================================
-- 120_world_template_test.sql — template-instantiation guarantees for the Drowned Lantern.
-- Ensures template worlds instantiate with fresh IDs, remain playable, respect archival in
-- the world directory lens, and keep canonical fixture IDs pinned in world 2222....
-- =====================================================================================
BEGIN;
SELECT plan(8);

-- Canonical play world and one fresh world instantiated from the template.
SELECT lives_ok(
  $$
    INSERT INTO world (world_id, display_name)
    VALUES ('33333333-3333-3333-3333-333333333333', 'Drowned Lantern Copy');

    SELECT fn_instantiate_drowned_lantern('33333333-3333-3333-3333-333333333333'::uuid);
  $$,
  'instantiating drowned_lantern into a fresh world succeeds'
);

-- (a) The instantiated world reuses NO canonical entity IDs.
SELECT is(
  (
    SELECT count(*)::int
    FROM entity_registry src
    JOIN entity_registry canon
      ON canon.entity_id = src.entity_id
     AND canon.world_id = '22222222-2222-2222-2222-222222222222'
    WHERE src.world_id = '33333333-3333-3333-3333-333333333333'
  ),
  0,
  '(a) instantiated world entity IDs are disjoint from canonical world IDs'
);

-- (b) The instantiated world is playable and marked with the template lineage key.
SELECT ok(
  (
    SELECT player_entity_id IS NOT NULL
       AND template_key = 'drowned_lantern'
    FROM world
    WHERE world_id = '33333333-3333-3333-3333-333333333333'
  ),
  '(b) instantiated world is playable and template_key is drowned_lantern'
);

-- (c) Archived worlds are hidden from fn_world_directory(), non-archived worlds remain visible.
INSERT INTO world (world_id, display_name, archived_at)
VALUES ('44444444-4444-4444-4444-444444444444', 'Archived Listing Probe', now());
INSERT INTO world (world_id, display_name)
VALUES ('55555555-5555-5555-5555-555555555555', 'Active Listing Probe');

SELECT ok(
  EXISTS (
    SELECT 1
    FROM json_array_elements((fn_world_directory()->'worlds')) w
    WHERE w->>'id' = '55555555-5555-5555-5555-555555555555'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM json_array_elements((fn_world_directory()->'worlds')) w
    WHERE w->>'id' = '44444444-4444-4444-4444-444444444444'
  ),
  '(c) fn_world_directory excludes archived worlds and includes non-archived worlds'
);

-- (d) Canonical world still carries the pinned fixture IDs after make seed.
SELECT set_eq(
  $$
    SELECT entity_id FROM entity_registry
    WHERE world_id = '22222222-2222-2222-2222-222222222222'
  $$,
  $$
    VALUES
      ('2ac70000-0000-0000-0000-0000000000a1'::uuid),
      ('2ac70000-0000-0000-0000-0000000000a2'::uuid),
      ('2ac70000-0000-0000-0000-0000000000a3'::uuid),
      ('2ac70000-0000-0000-0000-0000000000a4'::uuid),
      ('2ac70000-0000-0000-0000-0000000000aa'::uuid),
      ('210c0000-0000-0000-0000-0000000000d0'::uuid),
      ('210c0000-0000-0000-0000-0000000000d1'::uuid),
      ('210c0000-0000-0000-0000-0000000000d2'::uuid),
      ('210c0000-0000-0000-0000-0000000000d3'::uuid),
      ('210c0000-0000-0000-0000-0000000000d4'::uuid),
      ('210c0000-0000-0000-0000-0000000000d5'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000b1'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000c1'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000c2'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000c3'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000c4'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000d1'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000f1'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000f2'::uuid),
      ('2a7f0000-0000-0000-0000-0000000000f3'::uuid)
  $$,
  '(d) canonical entity_registry IDs remain pinned'
);

SELECT set_eq(
  $$
    SELECT event_id FROM canon_event
    WHERE world_id = '22222222-2222-2222-2222-222222222222'
      AND event_id IN (
        '2e000000-0000-0000-0000-0000000000f1'::uuid,
        '2e000000-0000-0000-0000-0000000000f2'::uuid,
        '2e000000-0000-0000-0000-0000000000f3'::uuid,
        '2e000000-0000-0000-0000-0000000000f4'::uuid,
        '2e000000-0000-0000-0000-0000000000f5'::uuid,
        '2e000000-0000-0000-0000-0000000000f6'::uuid,
        '2e000000-0000-0000-0000-0000000000f7'::uuid,
        '2e000000-0000-0000-0000-0000000000f8'::uuid,
        '2e000000-0000-0000-0000-0000000000f9'::uuid,
        '2e000000-0000-0000-0000-0000000000fa'::uuid,
        '2e000000-0000-0000-0000-0000000000e0'::uuid
      )
  $$,
  $$
    VALUES
      ('2e000000-0000-0000-0000-0000000000f1'::uuid),
      ('2e000000-0000-0000-0000-0000000000f2'::uuid),
      ('2e000000-0000-0000-0000-0000000000f3'::uuid),
      ('2e000000-0000-0000-0000-0000000000f4'::uuid),
      ('2e000000-0000-0000-0000-0000000000f5'::uuid),
      ('2e000000-0000-0000-0000-0000000000f6'::uuid),
      ('2e000000-0000-0000-0000-0000000000f7'::uuid),
      ('2e000000-0000-0000-0000-0000000000f8'::uuid),
      ('2e000000-0000-0000-0000-0000000000f9'::uuid),
      ('2e000000-0000-0000-0000-0000000000fa'::uuid),
      ('2e000000-0000-0000-0000-0000000000e0'::uuid)
  $$,
  '(d) canonical canon_event IDs remain pinned'
);

SELECT set_eq(
  $$
    SELECT perception_id FROM perception_record
    WHERE world_id = '22222222-2222-2222-2222-222222222222'
  $$,
  $$
    VALUES
      ('2ce50000-0000-0000-0000-0000000000a1'::uuid),
      ('2ce50000-0000-0000-0000-0000000000b1'::uuid),
      ('2ce50000-0000-0000-0000-0000000000c1'::uuid),
      ('2ca40000-0000-0000-0000-0000000000a1'::uuid),
      ('2a4e0000-0000-0000-0000-0000000000a1'::uuid),
      ('2a4e0000-0000-0000-0000-0000000000a2'::uuid),
      ('2a4e0000-0000-0000-0000-0000000000a3'::uuid),
      ('2a4e0000-0000-0000-0000-0000000000a4'::uuid)
  $$,
  '(d) canonical perception_record IDs remain pinned'
);

SELECT set_eq(
  $$
    SELECT player_entity_id FROM world
    WHERE world_id = '22222222-2222-2222-2222-222222222222'
  $$,
  $$ VALUES ('2ac70000-0000-0000-0000-0000000000a1'::uuid) $$,
  '(d) canonical world player_entity_id remains pinned to Kade'
);

SELECT * FROM finish();
ROLLBACK;
