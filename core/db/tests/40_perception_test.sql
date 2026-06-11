BEGIN;
SELECT plan(1);
SELECT is( (SELECT count(*) FROM entity_registry
            WHERE world_id='11111111-1111-1111-1111-111111111111')::int, 11,
       'registry seeded with cast: P,M,J,Tavern,PUB,O1..O5,Square');
SELECT * FROM finish();
ROLLBACK;
