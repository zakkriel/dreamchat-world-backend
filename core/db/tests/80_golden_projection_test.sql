BEGIN;
SELECT plan(1);
CREATE TEMP TABLE expected (entity_id uuid, location_id text);
INSERT INTO expected VALUES
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','square'),
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','market'),
 ('cccccccc-cccc-cccc-cccc-cccccccccccc','road'),
 ('00000000-0000-0000-0000-000000000001','dock'),
 ('00000000-0000-0000-0000-000000000002','tavern'),
 ('00000000-0000-0000-0000-000000000003','road'),
 ('00000000-0000-0000-0000-000000000004','dock'),
 ('00000000-0000-0000-0000-000000000005','tavern');
SELECT set_eq(
  'SELECT entity_id, attrs->>''location_id'' FROM actor_state',
  'SELECT entity_id, location_id FROM expected',
  'actor_state final locations match the hand-computed golden (doc 13 §5 spot-check)');
SELECT * FROM finish();
ROLLBACK;
