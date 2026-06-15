BEGIN;
SELECT plan(2);
SELECT ok( fn_actor_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb') IS NOT NULL,
           'actor page returns for a visible actor (Mara)');
SELECT ok( fn_actor_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','00000000-0000-0000-0000-000000000001') IS NULL,
           'actor page is NULL for an unperceived actor (O1) → Go maps to 404');
SELECT * FROM finish();
ROLLBACK;
