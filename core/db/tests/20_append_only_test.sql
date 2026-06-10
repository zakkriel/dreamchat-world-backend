BEGIN;
SELECT plan(5);

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000ff', '11111111-1111-1111-1111-111111111111',
        'move', 'test event', 10, 0, 'accepted', now(), 'private', 'fast_path');

SELECT throws_ok(
  $$ UPDATE canon_event SET summary='tampered' WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'append-only: editing summary raises');

SELECT throws_ok(
  $$ UPDATE canon_event SET status='proposed' WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'illegal status transition accepted->proposed raises');

SELECT lives_ok(
  $$ UPDATE canon_event SET status='superseded' WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  'legal transition accepted->superseded is allowed');

SELECT throws_ok(
  $$ DELETE FROM canon_event WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'DELETE on canon_event raises (append-only store)');

INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('e0000000-0000-0000-0000-0000000000ff','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','speaker');
SELECT throws_ok(
  $$ DELETE FROM event_participant WHERE event_id='e0000000-0000-0000-0000-0000000000ff' $$,
  NULL, NULL, 'DELETE on event_participant raises (canon table, ADR-006)');

SELECT * FROM finish();
ROLLBACK;
