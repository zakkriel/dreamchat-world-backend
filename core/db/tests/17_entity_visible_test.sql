BEGIN;
SELECT plan(5);
SELECT ok(    fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1'),
              'Player sees the note exists (holds a perception about it)');
SELECT ok(NOT fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'cccccccc-cccc-cccc-cccc-cccccccccccc','a4000000-0000-0000-0000-0000000000a1'),
              'Jonas does NOT see the note (existence withheld — the new sharp condition)');
SELECT ok(    fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
              'Player sees Mara (common knowledge)');
SELECT ok(    fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'cccccccc-cccc-cccc-cccc-cccccccccccc','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
              'Jonas sees Mara (common knowledge — symmetric)');
SELECT ok(NOT fn_entity_visible('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','00000000-0000-0000-0000-000000000001'),
              'Player does NOT see O1 (unperceived, non-CK)');
SELECT * FROM finish();
ROLLBACK;
