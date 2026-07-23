BEGIN;
SELECT plan(3);

-- location ids: attrs.location_id stores a registry uuid; fn_actors_at takes uuid.
-- Fresh world + location + actor (no seed dependency).
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
VALUES ('f9000000-0000-0000-0000-000000000001', 'f9000000-ffff-0000-0000-000000000000', 'location', 'test-vault-99');
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
VALUES ('f9000000-0000-0000-0000-000000000002', 'f9000000-ffff-0000-0000-000000000000', 'actor', 'test-kade-99');
INSERT INTO actor_state (entity_id, world_id, attrs)
VALUES ('f9000000-0000-0000-0000-000000000002', 'f9000000-ffff-0000-0000-000000000000',
        jsonb_build_object('location_id', 'f9000000-0000-0000-0000-000000000001'));

SELECT is(
  (SELECT count(*) FROM fn_actors_at('f9000000-ffff-0000-0000-000000000000', 'f9000000-0000-0000-0000-000000000001'::uuid))::int,
  1, 'fn_actors_at(world, loc_uuid) finds exactly the one actor at that location');

SELECT set_eq(
  $$ SELECT entity_id FROM fn_actors_at('f9000000-ffff-0000-0000-000000000000', 'f9000000-0000-0000-0000-000000000001'::uuid) $$,
  $$ VALUES ('f9000000-0000-0000-0000-000000000002'::uuid) $$,
  'fn_actors_at returns the correct actor uuid');

SELECT is(
  (SELECT count(*) FROM fn_actors_at('f9000000-ffff-0000-0000-000000000000', 'f9000000-0000-0000-0000-000000000003'::uuid))::int,
  0, 'fn_actors_at returns empty for an unoccupied location uuid');

SELECT * FROM finish();
ROLLBACK;
