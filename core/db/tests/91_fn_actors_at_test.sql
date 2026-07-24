BEGIN;
SELECT plan(3);

-- Position two actors at vault-uuid, one at cellar-uuid, via accepted move events (trigger maintains
-- actor_state). Fresh location uuids registered in entity_registry; these names never appear in the
-- seed, so co-presence is unambiguous and seed-independent. Ticks 210+ miss every existing assertion.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('e5ffffff-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','location','test-vault-91'),
 ('e5ffffff-0000-0000-0000-000000000002','11111111-1111-1111-1111-111111111111','location','test-cellar-91'),
 ('e5ffffff-0000-0000-0000-000000000003','11111111-1111-1111-1111-111111111111','location','test-crypt-91');

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000010','11111111-1111-1111-1111-111111111111','move','setup P→vault',210,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000011','11111111-1111-1111-1111-111111111111','move','setup M→vault',211,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000012','11111111-1111-1111-1111-111111111111','move','setup J→cellar',212,0,'accepted',now(),'public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000010','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000001'::text),210,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000011','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000001'::text),211,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000012','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000002'::text),212,0);

SELECT set_eq(
  $$ SELECT entity_id FROM fn_actors_at('11111111-1111-1111-1111-111111111111','e5ffffff-0000-0000-0000-000000000001'::uuid) $$,
  $$ VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::uuid),
            ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid) $$,
  'Player and Mara are co-present at vault-uuid');
SELECT set_eq(
  $$ SELECT entity_id FROM fn_actors_at('11111111-1111-1111-1111-111111111111','e5ffffff-0000-0000-0000-000000000002'::uuid) $$,
  $$ VALUES ('cccccccc-cccc-cccc-cccc-cccccccccccc'::uuid) $$,
  'Jonas alone at cellar-uuid');
SELECT is(
  (SELECT count(*) FROM fn_actors_at('11111111-1111-1111-1111-111111111111','e5ffffff-0000-0000-0000-000000000003'::uuid))::int,
  0, 'nobody at crypt-uuid (a fresh, unoccupied location)');

SELECT * FROM finish();
ROLLBACK;
