-- fn_speech_perception_candidates / fn_speech_perception_facts — structural correctness of the
-- read-side facts assembly, independent of any model judgment. Fixed UUIDs, f8000000-prefixed,
-- self-contained.
--
--   world:                         f8000000-ffff-0000-0000-000000000000 (player_entity_id = B)
--   A (speaker):                   f8000000-0000-0000-0000-000000000001
--   B (co-present NPC, player):    f8000000-0000-0000-0000-000000000002
--   C (co-present, intended addr): f8000000-0000-0000-0000-000000000003
--   D (elsewhere, valid actor):    f8000000-0000-0000-0000-000000000004
--   room:                          f8000000-0000-0000-0000-000000000010
--   elsewhere:                     f8000000-0000-0000-0000-000000000011
--   note (artifact, C's perception subject — must NOT leak into references):
--                                  f8000000-0000-0000-0000-000000000020
BEGIN;
SELECT plan(17);

\set w    'f8000000-ffff-0000-0000-000000000000'
\set a    'f8000000-0000-0000-0000-000000000001'
\set b    'f8000000-0000-0000-0000-000000000002'
\set c    'f8000000-0000-0000-0000-000000000003'
\set d    'f8000000-0000-0000-0000-000000000004'
\set room 'f8000000-0000-0000-0000-000000000010'
\set away 'f8000000-0000-0000-0000-000000000011'
\set note 'f8000000-0000-0000-0000-000000000020'

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 (:'a','f8000000-ffff-0000-0000-000000000000','actor','test-A-125'),
 (:'b','f8000000-ffff-0000-0000-000000000000','actor','test-B-125'),
 (:'c','f8000000-ffff-0000-0000-000000000000','actor','test-C-125'),
 (:'d','f8000000-ffff-0000-0000-000000000000','actor','test-D-125'),
 (:'room','f8000000-ffff-0000-0000-000000000000','location','test-room-125'),
 (:'away','f8000000-ffff-0000-0000-000000000000','location','test-away-125'),
 (:'note','f8000000-ffff-0000-0000-000000000000','artifact','test-note-125');

INSERT INTO world (world_id, display_name, tagline, player_entity_id)
VALUES (:'w'::uuid, 'test-125', 'candidates and facts', :'b'::uuid);

-- Positions relative to the room: A=(0,0), B=(3,4), C=(6,8).
-- Their distances from A are independently 5 and 10 metres.
INSERT INTO location_state (entity_id, world_id, attrs)
VALUES (:'room', :'w', '{"coordinates":{"x":0,"y":0}}'::jsonb);
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
 (:'a', :'w', '{"coordinates":{"x":0,"y":0}}'::jsonb),
 (:'b', :'w', '{"descriptor":"a face at the table","coordinates":{"x":3,"y":4}}'::jsonb),
 (:'c', :'w', '{"descriptor":"a stranger by the door","coordinates":{"x":6,"y":8}}'::jsonb);

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('f8500000-0000-0000-0000-000000000001',:'w','move','setup A->room',1,0,'accepted',now(),'public','fast_path'),
 ('f8500000-0000-0000-0000-000000000002',:'w','move','setup B->room',2,0,'accepted',now(),'public','fast_path'),
 ('f8500000-0000-0000-0000-000000000003',:'w','move','setup C->room',3,0,'accepted',now(),'public','fast_path'),
 ('f8500000-0000-0000-0000-000000000004',:'w','move','setup D->away',4,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('f8500000-0000-0000-0000-000000000001',:'a','actor','instigator'),
 ('f8500000-0000-0000-0000-000000000002',:'b','actor','instigator'),
 ('f8500000-0000-0000-0000-000000000003',:'c','actor','instigator'),
 ('f8500000-0000-0000-0000-000000000004',:'d','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 (:'w','f8500000-0000-0000-0000-000000000001',:'a','actor','attrs.location_id', to_jsonb(:'room'::text), 1, 0),
 (:'w','f8500000-0000-0000-0000-000000000002',:'b','actor','attrs.location_id', to_jsonb(:'room'::text), 2, 0),
 (:'w','f8500000-0000-0000-0000-000000000003',:'c','actor','attrs.location_id', to_jsonb(:'room'::text), 3, 0),
 (:'w','f8500000-0000-0000-0000-000000000004',:'d','actor','attrs.location_id', to_jsonb(:'away'::text), 4, 0);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- fn_speech_perception_candidates: co-present except the speaker, plus the addressed recipient
-- when valid — independent of co-presence.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT is((SELECT count(*)::int FROM fn_speech_perception_candidates(:'w'::uuid,:'a'::uuid,:'c'::uuid)),
          2, '(1) candidates(A, intended=C): exactly B and C — A excludes herself');
SELECT ok(EXISTS(SELECT 1 FROM fn_speech_perception_candidates(:'w'::uuid,:'a'::uuid,:'c'::uuid) WHERE actor_id=:'b'::uuid),
          '(2) B (co-present, not addressed) is a candidate');
SELECT ok(NOT EXISTS(SELECT 1 FROM fn_speech_perception_candidates(:'w'::uuid,:'a'::uuid,:'c'::uuid) WHERE actor_id=:'d'::uuid),
          '(3) D (elsewhere, not addressed) is NOT a candidate');

SELECT is((SELECT count(*)::int FROM fn_speech_perception_candidates(:'w'::uuid,:'a'::uuid,:'d'::uuid)),
          3, '(4) candidates(A, intended=D): B, C (co-present) UNION D (addressed though absent)');
SELECT ok(EXISTS(SELECT 1 FROM fn_speech_perception_candidates(:'w'::uuid,:'a'::uuid,:'d'::uuid) WHERE actor_id=:'d'::uuid),
          '(5) the addressed recipient is a candidate even when not co-present, "when valid"');

SELECT ok(NOT EXISTS(SELECT 1 FROM fn_speech_perception_candidates(:'w'::uuid,:'a'::uuid,:'a'::uuid) WHERE actor_id=:'a'::uuid),
          '(6) self-addressed speech cannot teach the speaker as though she were a listener');

SELECT is((SELECT count(*)::int FROM fn_speech_perception_candidates(:'w'::uuid,:'a'::uuid,:'room'::uuid)),
          2, '(7) an intended_listener that is not a real actor (a location id) is not "valid" — excluded'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- fn_speech_perception_facts: one object per candidate, correctly shaped.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

-- B's recent knowledge: one row inside the window, one far outside it (window=50, limit=20 —
-- beatHandler.go's own recencyTickWindow/recencyMaxRows values, passed through unchanged).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('f8500000-0000-0000-0000-000000000005',:'w','AttributeChanged','old news',10,0,'accepted',now(),'public','fast_path'),
 ('f8500000-0000-0000-0000-000000000006',:'w','AttributeChanged','recent news',995,0,'accepted',now(),'public','fast_path');
INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick) VALUES
 (:'w',:'b','f8500000-0000-0000-0000-000000000005','old news outside the window','direct',10,10),
 (:'w',:'b','f8500000-0000-0000-0000-000000000006','recent news inside the window','direct',995,995);

-- Only C has sourced knowledge of absent D; the note is not an actor reference.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
VALUES ('f8510000-0000-0000-0000-000000000001', :'w', :'c', 'f8500000-0000-0000-0000-000000000003',
        'C reads a note about D', 'direct', 3, 3);
INSERT INTO perception_subject (perception_id, entity_id, world_id)
VALUES ('f8510000-0000-0000-0000-000000000001', :'note'::uuid, :'w'::uuid),
       ('f8510000-0000-0000-0000-000000000001', :'d'::uuid, :'w'::uuid);

-- C has a pending held reaction; B has none.
INSERT INTO held_outcome (world_id, actor_id, attempt, telegraph_event_id, status, created_tick)
VALUES (:'w'::uuid, :'c'::uuid, '{"type":"AttributeChanged","target_id":"deadbeef-dead-dead-dead-deaddeadbeef"}'::jsonb,
        'f8500000-0000-0000-0000-000000000003', 'pending', 3);


SELECT is(
  (fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid, 50, 20)->>'schema_version'),
  'speech_perception_facts/1', '(8) schema_version is speech_perception_facts/1'
);
SELECT is(
  jsonb_array_length(fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid, 50, 20)->'listeners'),
  2, '(9) exactly one listener object per candidate'
);

SELECT ok(
  (SELECT (l->>'is_player')::boolean FROM jsonb_array_elements(
     fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid, 50, 20)->'listeners') l
   WHERE l->>'actor_id' = :'b'),
  '(10) is_player is true for the world''s player_entity_id'
);
SELECT ok(
  NOT (SELECT (l->>'is_player')::boolean FROM jsonb_array_elements(
     fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid, 50, 20)->'listeners') l
   WHERE l->>'actor_id' = :'c'),
  '(11) is_player is false for an NPC candidate'
);

-- Neither listener knows the speaker's private name for B or either registry identity.
INSERT INTO name_knowledge(world_id,holder_id,entity_id,name,learned_tick,source_event_id)
VALUES (:'w',:'a',:'b','speaker-private-name',995,'f8500000-0000-0000-0000-000000000006');
SELECT ok(
  fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid,50,20)::text
    !~ '(speaker-private-name|test-B-125|test-C-125)',
  '(12) listener facts do not reveal speaker-private names or registry identities as metadata'
);

SELECT is(
  (SELECT jsonb_object_agg(l->>'actor_id',
            (SELECT jsonb_agg(r->>'actor_id' ORDER BY r->>'actor_id')
             FROM jsonb_array_elements(l->'references') r))
   FROM jsonb_array_elements(
     fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid,50,20)->'listeners') l),
  jsonb_build_object(:'b', jsonb_build_array(:'a', :'b', :'c'),
                    :'c', jsonb_build_array(:'a', :'b', :'c', :'d')),
  '(13) present actors are referenceable to both listeners; absent D only to C; neither receives the note as an actor'
);

SELECT is(
  (SELECT l->'knowledge' FROM jsonb_array_elements(
     fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid, 50, 20)->'listeners') l
   WHERE l->>'actor_id' = :'b'),
  jsonb_build_array('recent news inside the window'),
  '(14) knowledge is windowed exactly like beatHandler.payload(): recent kept, far-outside dropped'
);

SELECT is(
  (SELECT jsonb_object_agg(l->>'actor_id', l->'activities'->'held'->'attempt')
   FROM jsonb_array_elements(
     fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid,50,20)->'listeners') l),
  jsonb_build_object(:'b', NULL, :'c',
    '{"type":"AttributeChanged","target_id":"deadbeef-dead-dead-dead-deaddeadbeef"}'::jsonb),
  '(15) C''s pending intention reaches C''s facts, never B''s'
);

UPDATE held_outcome SET status='resolved' WHERE world_id=:'w'::uuid AND actor_id=:'c'::uuid;
SELECT ok(
  (SELECT bool_and(l->'activities'->'held' = 'null'::jsonb)
   FROM jsonb_array_elements(
     fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid,50,20)->'listeners') l),
  '(17) a resolved intention is no longer an active fact for any listener'
);

SELECT is(
  (SELECT jsonb_object_agg(l->>'actor_id',
            jsonb_build_array(l->'physics'->>'id', (l->'physics'->>'distance_m')::numeric))
   FROM jsonb_array_elements(
     fn_speech_perception_facts(:'w'::uuid,:'a'::uuid,:'c'::uuid,50,20)->'listeners') l),
  jsonb_build_object(:'b', jsonb_build_array(:'b', 5), :'c', jsonb_build_array(:'c', 10)),
  '(18) each listener receives their own physical target and independently derived distance'
);

SELECT * FROM finish();
ROLLBACK;
