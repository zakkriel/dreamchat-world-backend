BEGIN;
SELECT plan(7);

-- Player at tavern-uuid, Mara at tavern-uuid, Jonas at square-uuid (square-uuid has no seed occupant
-- besides Player, who is moved to tavern-uuid here → square-uuid = {Jonas}). say-gates are EXISTS
-- checks (tolerant of seed actors also at tavern-uuid).
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('e5ffffff-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111','location','test-district-96'),
 ('e5ffffff-0000-0000-0000-000000000010','11111111-1111-1111-1111-111111111111','location','test-tavern-96'),
 ('e5ffffff-0000-0000-0000-000000000011','11111111-1111-1111-1111-111111111111','location','test-square-96');
-- Station F / §2: fn_move_duration is now CEIL(fn_distance / 1.4). Give the two locations a real §3
-- geometry (sibling children of a district). 7 m apart → CEIL(7/1.4) = 5 ticks — the move duration this
-- partial-beat test asserts below, now honestly derived from distance/speed (and < the 100 cap, so the
-- stop-check, not the turn-budget, is what halts).
INSERT INTO location_state (entity_id, world_id, attrs) VALUES
 ('e5ffffff-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111',
  '{"coordinates":{"x":0,"y":0},"area":{"points":[{"x":0,"y":0},{"x":2000,"y":0},{"x":2000,"y":2000},{"x":0,"y":2000}]}}'::jsonb),
 ('e5ffffff-0000-0000-0000-000000000010','11111111-1111-1111-1111-111111111111',
  '{"coordinates":{"x":0,"y":0},"parent_location_id":"e5ffffff-0000-0000-0000-0000000000d1"}'::jsonb),
 ('e5ffffff-0000-0000-0000-000000000011','11111111-1111-1111-1111-111111111111',
  '{"coordinates":{"x":7,"y":0},"parent_location_id":"e5ffffff-0000-0000-0000-0000000000d1"}'::jsonb);

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
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000050','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000010'::text),600,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000051','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000010'::text),601,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000052','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000011'::text),602,0);

-- Station F / §5.3: a PERMITTING portal (open ∧ ¬locked) connecting tavern↔square so every
-- tavern→square move in this test passes the accessibility floor. Accessibility, NOT geometry (no coords).
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('e5ffffff-0000-0000-0000-0000000000c1','11111111-1111-1111-1111-111111111111','artifact','portal-tavern-square-96');
INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
 ('e5ffffff-0000-0000-0000-0000000000c1','11111111-1111-1111-1111-111111111111',
  jsonb_build_object('open', true, 'locked', false,
    'connects', jsonb_build_array('e5ffffff-0000-0000-0000-000000000010','e5ffffff-0000-0000-0000-000000000011')));

-- HALT WAY 1 — gate-reject: [say to Mara (ok), say to Jonas (Jonas not co-present → reject), move].
-- Expect: step 1 commits; step 2 rejected pre-apply; step 3 never runs.
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"hi Mara"},
              {"type":"say","listener":"cccccccc-cccc-cccc-cccc-cccccccccccc","content":"hi Jonas"},
              {"type":"move","to":"e5ffffff-0000-0000-0000-000000000011"}]'::jsonb, 700, 100, 'fast_path') ->> 'halt_reason'),
           'gate_reject', 'chain halts pre-apply on the impossible SAY');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=700 AND in_world_tick<800)::int,
           1, 'gate-reject: EXACTLY the prefix (1 event) committed — door rejected, move never ran');
SELECT is( (SELECT count(*) FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
            WHERE ce.in_world_tick>=700 AND ce.in_world_tick<800)::int,
           2, 'gate-reject: exactly the prefix perceptions (speaker shared + listener told)');

-- HALT WAY 2 — stop-check via discovery: [move to square-uuid, say to Mara]. Arriving at square-uuid
-- the player discovers (premise of "say to Mara" = Mara co-present) is BROKEN → halt AFTER the move commits.
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"move","to":"e5ffffff-0000-0000-0000-000000000011"},
              {"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"Mara?"}]'::jsonb,
            800, 100, 'fast_path') ->> 'halt_reason'),
           'stop_check', 'chain halts post-apply: discovery breaks the next premise (§9 source (a))');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=800 AND in_world_tick<900 AND event_type='move')::int,
           1, 'stop-check: the move committed (prefix stands)');
SELECT is( (SELECT count(*) FROM canon_event WHERE in_world_tick>=800 AND in_world_tick<900 AND event_type='private_disclosure')::int,
           0, 'stop-check: the SAY never committed (zero trace from the halt onward)');

-- Reposition Player → tavern-uuid: the 800-run left Player AT square-uuid, so without this the next
-- move→square-uuid would be zero-distance (fn_move_duration square→square = CEIL(0/1.4) = 0) and
-- ticks_advanced = 0.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e5000000-0000-0000-0000-000000000053','11111111-1111-1111-1111-111111111111','move','P→tavern',895,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('e5000000-0000-0000-0000-000000000053','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000053','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000010'::text),895,0);
-- partial-beat TIME: the clock advanced by the move's duration only (CEIL(7/1.4)=5), not the say's
-- (0-after-halt).
SELECT is( (apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
            '[{"type":"move","to":"e5ffffff-0000-0000-0000-000000000011"},
              {"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"Mara?"}]'::jsonb,
            900, 100, 'fast_path') ->> 'ticks_advanced')::int,
           5, 'partial-beat time: clock advanced by the committed-prefix duration only (ADR-036)');

SELECT * FROM finish();
ROLLBACK;
