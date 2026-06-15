BEGIN;
SELECT plan(7);
-- PAIRED existence asymmetry on the SAME note id (an absence-only assertion is forbidden):
SELECT ok( EXISTS (SELECT 1 FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','artifact')
             WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'note PRESENT in Player artifact index');
SELECT ok( NOT EXISTS (SELECT 1 FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','artifact')
             WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'note ABSENT from Jonas artifact index (existence not leaked — fails loud on breach)');
-- withheld name in the index is perception-layer NULL, NEVER entity_registry.canonical_name
SELECT is( (SELECT perceived_name FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','artifact')
             WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           NULL, 'note name withheld in index (NULL, not the canon name)');
-- actor index: O1..O5 absent for both viewers (symmetric negative)
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor')
             WHERE entity_id IN ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002',
               '00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004',
               '00000000-0000-0000-0000-000000000005'))::int,
           0, 'O1..O5 absent from Player actor index');
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','actor')
             WHERE entity_id IN ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002',
               '00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004',
               '00000000-0000-0000-0000-000000000005'))::int,
           0, 'O1..O5 absent from Jonas actor index');
-- CK cast present in the actor index for both
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','actor')
             WHERE entity_id IN ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
               'cccccccc-cccc-cccc-cccc-cccccccccccc'))::int,
           3, 'CK cast (Player,Mara,Jonas) present in Jonas actor index');
-- CK locations present
SELECT is( (SELECT count(*) FROM fn_compendium_index(
             '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','location')
             WHERE entity_id IN ('dddddddd-dddd-dddd-dddd-dddddddddddd','000000a0-0000-0000-0000-0000000000a1'))::int,
           2, 'CK locations (Tavern,Square) present in Jonas location index');
SELECT * FROM finish();
ROLLBACK;
