BEGIN;
SELECT plan(4);
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
           'Mara', 'Mara name renders for Player (common knowledge)');
-- common knowledge ⇒ renders even for a viewer who knows nothing else about Mara
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'cccccccc-cccc-cccc-cccc-cccccccccccc','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
           'Mara', 'Mara name renders for Jonas (common knowledge, not viewer-specific)');
-- THE REAL GATE: a noise actor with no CK name perception is WITHHELD (NULL), not leaked
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','00000000-0000-0000-0000-000000000001'),
           NULL, 'O1 name withheld — gate is real, not a raw entity_registry read');
-- name comes from the perception layer, never entity_registry (Player resolves Player)
SELECT is( fn_perceived_name('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
           'Player', 'Player name renders for self');
SELECT * FROM finish();
ROLLBACK;
