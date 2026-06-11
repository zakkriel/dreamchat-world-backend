BEGIN;
SELECT plan(3);

-- (1) happy path: rebuild from the event log == live domain state
SELECT ok( replay_0A(), 'I-1: replay reproduces domain-equivalent projection state (ADR-026)' );

-- (2) detection: corrupt one live projection value; the snapshot captures the corruption, so the
--     event-log rebuild differs from the snapshot and the diff MUST bite -> replay_0A() returns FALSE.
UPDATE actor_state SET attrs = jsonb_set(attrs,'{location_id}','"WRONG"',true)
 WHERE entity_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb';
SELECT ok( NOT replay_0A(),
       'negative control: corrupted snapshot != event-log rebuild -> replay_0A() returns FALSE' );

-- (3) repair: the prior replay_0A() rebuilt the table from the event log, repairing the corruption,
--     so the snapshot now equals the rebuild -> replay_0A() returns TRUE.
SELECT ok( replay_0A(), 'repair: after rebuild the projection matches the event log again' );

SELECT * FROM finish();
ROLLBACK;
