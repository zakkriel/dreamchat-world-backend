BEGIN;
SELECT plan(6);

-- A SAY event (private_disclosure) P→M with both at tavern; J elsewhere. generate_perceptions writes
-- speaker 'shared' + listener 'told' (B-7), nothing for J. acquired_tick = event tick (I-9).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e5000000-0000-0000-0000-000000000020','11111111-1111-1111-1111-111111111111',
        'private_disclosure','P tells M a secret',300,0,'accepted',now(),'private','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000020','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','speaker'),
 ('e5000000-0000-0000-0000-000000000020','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','listener');

SELECT is(generate_perceptions('e5000000-0000-0000-0000-000000000020')::int, 2,
          'SAY generates exactly 2 perceptions (speaker + listener)');
SELECT is((SELECT epistemic_type FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
          'shared', 'speaker (Player) holds a SHARED perception');
SELECT is((SELECT epistemic_type FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
          'told', 'listener (Mara) holds a TOLD perception (B-7: told ≠ witnessed)');
SELECT is((SELECT count(*) FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='cccccccc-cccc-cccc-cccc-cccccccccccc')::int,
          0, 'Jonas (not addressed) holds NOTHING — the knowledge boundary (B-1/I-3)');
SELECT is((SELECT acquired_tick FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000020'
             AND holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
          300::bigint, 'acquired_tick = event tick (I-9: cannot learn before it happened)');
-- about-ness (RULINGS-2026-07-23 §6, engine-written): both perceptions (speaker 'shared' +
-- listener 'told') carry subject rows for the source event's participants (P and M).
-- 2 perceptions x 2 subjects = 4 rows.
SELECT is((SELECT count(*)::int FROM perception_record pr
           JOIN perception_subject ps ON ps.perception_id = pr.perception_id
           WHERE pr.source_event_id='e5000000-0000-0000-0000-000000000020'
             AND ps.entity_id IN ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')),
          4, 'both SAY perceptions (shared+told) carry subject rows for speaker P and listener M');

SELECT * FROM finish();
ROLLBACK;
