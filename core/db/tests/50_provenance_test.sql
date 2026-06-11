BEGIN;
SELECT plan(3);
SELECT is( (SELECT count(*) FROM state_mutation sm
            LEFT JOIN canon_event ce ON ce.event_id=sm.event_id AND ce.status='accepted'
            WHERE ce.event_id IS NULL)::int, 0, 'I-2: zero orphan state_mutations');
SELECT is( (SELECT count(*) FROM perception_record pr
            LEFT JOIN canon_event ce ON ce.event_id=pr.source_event_id AND ce.status='accepted'
            WHERE ce.event_id IS NULL)::int, 0, 'I-2: zero orphan perceptions');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick BETWEEN 101 AND 200)::int,
       100, '100 noise events present');
SELECT * FROM finish();
ROLLBACK;
