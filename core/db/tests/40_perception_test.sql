BEGIN;
SELECT plan(4);
SELECT is( (SELECT count(*) FROM entity_registry
            WHERE world_id='11111111-1111-1111-1111-111111111111')::int, 11,
       'registry seeded with cast: P,M,J,Tavern,PUB,O1..O5,Square');
SELECT is( (SELECT count(*) FROM perception_record
            WHERE holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
              AND source_event_id='e0000000-0000-0000-0000-000000000001'
              AND epistemic_type='told' AND invalid_tick IS NULL AND expired_at IS NULL)::int,
       1, 'Mara has exactly one active told perception of E1 (mara_knows_ok)');
SELECT is( (SELECT count(*) FROM perception_record
            WHERE holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
              AND source_event_id='e0000000-0000-0000-0000-000000000001'
              AND epistemic_type='shared')::int, 1, 'Player has one shared perception of E1');
SELECT is( (SELECT count(*) FROM perception_record
            WHERE holder_id='cccccccc-cccc-cccc-cccc-cccccccccccc'
              AND source_event_id='e0000000-0000-0000-0000-000000000001')::int,
       0, 'Jonas has ZERO perceptions of E1 (knowledge boundary, j_ignorant_ok)');
SELECT * FROM finish();
ROLLBACK;
