BEGIN;
SELECT plan(4);
-- Player sees his own + common-knowledge perceptions
SELECT ok(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')) > 0,
  'Player has visible perceptions');
-- GATE-CRITICAL NEGATIVE: a perception held only by Mara is ABSENT for viewer=Jonas
SELECT is(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc')
   WHERE holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')::int,
  0, 'I-3: no Mara-held perception is visible to Jonas');
-- common knowledge IS visible to everyone (the public ledger record, held by PUB)
SELECT ok(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc')
   WHERE holder_id='eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee') > 0,
  'common-knowledge holder perceptions are visible to Jonas');
-- closed perceptions (invalid_tick / expired_at) never returned (none in seed → boundary holds at 0)
SELECT is(
  (SELECT count(*) FROM fn_visible_perceptions(
     '11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')
   WHERE invalid_tick IS NOT NULL OR expired_at IS NOT NULL)::int,
  0, 'closed perceptions are never returned');
SELECT * FROM finish();
ROLLBACK;
