BEGIN;
SELECT plan(4);
-- I-2 (universal provenance): both new perceptions reference an ACCEPTED event
SELECT is( (SELECT count(*) FROM perception_record pr
            JOIN canon_event ce ON ce.event_id = pr.source_event_id
            WHERE pr.source_event_id='e0000000-0000-0000-0000-0000000000d1' AND ce.status='accepted')::int,
           2, 'I-2: the two new perceptions reference an accepted event (no orphans)');
-- I-1 (replay): observation ≠ canon mutation (ADR-005) → no state_mutation introduced
SELECT is( (SELECT count(*) FROM state_mutation
            WHERE event_id='e0000000-0000-0000-0000-0000000000d1')::int,
           0, 'I-1: discovery event carries no state_mutation');
-- I-1: the note is perceived, not state-bearing → no artifact_state row (like Tavern/Square locations)
SELECT is( (SELECT count(*) FROM artifact_state
            WHERE entity_id='a4000000-0000-0000-0000-0000000000a1')::int,
           0, 'I-1: the note has no artifact_state row');
-- I-1: replay invariance still holds on the EXPANDED seed (truncate+rebuild; rolled back at ROLLBACK)
SELECT ok( replay_0A(), 'I-1: replay invariance holds on the expanded seed');
SELECT * FROM finish();
ROLLBACK;
