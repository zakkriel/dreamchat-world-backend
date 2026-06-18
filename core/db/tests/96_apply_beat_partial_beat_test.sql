BEGIN;
SELECT plan(7);

-- Player at tavern, Mara at tavern, Jonas at square (square has no seed occupant besides Player,
-- who is moved to tavern here → square = {Jonas}). say-gates are EXISTS checks (tolerant of seed
-- actors also at tavern).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000050','11111111-1111-1111-1111-111111111111','move','P→tavern',600,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000051','11111111-1111-1111-1111-111111111111','move','M→tavern',601,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000052','11111111-1111-1111-1111-111111111111','move','J→square',602,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000050','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000051','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000052','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000050','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),600,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000051','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('tavern'::text),601,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000052','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('square'::text),602,0);

-- HALT WAY 1 — gate-reject: [say to Mara (ok), say to Jonas (Jonas not co-present → reject), move].
-- Expect: step 1 commits; step 2 rejected pre-apply; step 3 never runs.
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"hi Mara"},
              {"type":"say","listener":"cccccccc-cccc-cccc-cccc-cccccccccccc","content":"hi Jonas"},
              {"type":"move","to":"square"}]'::jsonb, 700, 100, 'fast_path') ->> 'halt_reason'),
           'gate_reject', 'chain halts pre-apply on the impossible SAY');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=700 AND in_world_tick<800)::int,
           1, 'gate-reject: EXACTLY the prefix (1 event) committed — door rejected, move never ran');
SELECT is( (SELECT count(*) FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
            WHERE ce.in_world_tick>=700 AND ce.in_world_tick<800)::int,
           2, 'gate-reject: exactly the prefix perceptions (speaker shared + listener told)');

-- HALT WAY 2 — stop-check via discovery: [move to square, say to Mara]. Arriving at square the player
-- discovers (premise of "say to Mara" = Mara co-present) is BROKEN → halt AFTER the move commits.
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"move","to":"square"},
              {"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"Mara?"}]'::jsonb,
            800, 100, 'fast_path') ->> 'halt_reason'),
           'stop_check', 'chain halts post-apply: discovery breaks the next premise (§9 source (a))');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=800 AND in_world_tick<900 AND event_type='move')::int,
           1, 'stop-check: the move committed (prefix stands)');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=800 AND in_world_tick<900 AND event_type='private_disclosure')::int,
           0, 'stop-check: the SAY never committed (zero trace from the halt onward)');

-- Reposition Player → tavern: the 800-run left Player AT square, so without this the next
-- move→square would be zero-distance (fn_move_duration square→square = 0) and ticks_advanced = 0.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e5000000-0000-0000-0000-000000000053','11111111-1111-1111-1111-111111111111','move','P→tavern',895,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('e5000000-0000-0000-0000-000000000053','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000053','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),895,0);
-- partial-beat TIME: the clock advanced by the move's duration only (5), not the say's (0-after-halt).
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"move","to":"square"},
              {"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"Mara?"}]'::jsonb,
            900, 100, 'fast_path') ->> 'ticks_advanced')::int,
           5, 'partial-beat time: clock advanced by the committed-prefix duration only (ADR-036)');

SELECT * FROM finish();
ROLLBACK;
