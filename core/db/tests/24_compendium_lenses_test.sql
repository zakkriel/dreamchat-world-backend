BEGIN;
SELECT plan(16);

-- Synthetic multi-subject perception for Player: links Mara + Sealed Note + Tavern in one held record.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-000000000290','11111111-1111-1111-1111-111111111111',
        'observation','Mara concealed the note in the Tavern',250,0,
        'Day 3', 'accepted', now(), 'private', 'fast_path');

INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick)
VALUES ('dca70000-0000-0000-0000-000000000290','11111111-1111-1111-1111-111111111111',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-000000000290',
        'Mara tucked the sealed note under her cloak in the Tavern.','direct',250,250);

INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
('dca70000-0000-0000-0000-000000000290','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111'),
('dca70000-0000-0000-0000-000000000290','a4000000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111'),
('dca70000-0000-0000-0000-000000000290','dddddddd-dddd-dddd-dddd-dddddddddddd','11111111-1111-1111-1111-111111111111');

-- Actor page lens population
SELECT ok(
  (fn_actor_page('11111111-1111-1111-1111-111111111111',
                 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->>'current_synthesis') IS NOT NULL,
  'actor.current_synthesis is populated from held perceptions');

SELECT is(
  fn_actor_page('11111111-1111-1111-1111-111111111111',
                'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->>'last_known_status',
  'Mara tucked the sealed note under her cloak in the Tavern.',
  'actor.last_known_status is latest held perception content');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_actor_page('11111111-1111-1111-1111-111111111111',
                       'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                       'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'known_artifacts') a
  WHERE a->>'id' = 'a4000000-0000-0000-0000-0000000000a1'
), 'actor.known_artifacts links co-perceived artifacts');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_actor_page('11111111-1111-1111-1111-111111111111',
                       'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                       'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'inline_links') l
  WHERE l->>'id' = 'dddddddd-dddd-dddd-dddd-dddddddddddd'
), 'actor.inline_links includes related location links');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_actor_page('11111111-1111-1111-1111-111111111111',
                       'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                       'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g
  WHERE g->>'group_key' = 'event:e0000000-0000-0000-0000-000000000290'
), 'collected_knowledge_groups are event-keyed, not target-id keyed');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_actor_page('11111111-1111-1111-1111-111111111111',
                       'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                       'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
       json_array_elements(g->'items') it
  WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000a01'
    AND (it->'decay'->>'stale')::boolean = true
), 'page decay marks old perceptions stale based on world time');

-- Location page lens population
SELECT ok(
  fn_location_page('11111111-1111-1111-1111-111111111111',
                   'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                   'dddddddd-dddd-dddd-dddd-dddddddddddd')->'location'->>'current_synthesis'
  LIKE '%Mara tucked the sealed note%',
  'location.current_synthesis composes held perceptions about the location');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_location_page('11111111-1111-1111-1111-111111111111',
                          'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                          'dddddddd-dddd-dddd-dddd-dddddddddddd')->'location'->'key_actors') a
  WHERE a->>'id' = 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
), 'location.key_actors links co-perceived actors');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_location_page('11111111-1111-1111-1111-111111111111',
                          'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                          'dddddddd-dddd-dddd-dddd-dddddddddddd')->'location'->'inline_links') l
  WHERE l->>'id' = 'a4000000-0000-0000-0000-0000000000a1'
), 'location.inline_links includes related artifact links');

-- Artifact page lens population
SELECT ok(
  fn_artifact_page('11111111-1111-1111-1111-111111111111',
                   'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                   'a4000000-0000-0000-0000-0000000000a1')->'artifact'->>'current_synthesis'
  LIKE '%sealed note%',
  'artifact.current_synthesis composes held perceptions about the artifact');

SELECT is(
  fn_artifact_page('11111111-1111-1111-1111-111111111111',
                   'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                   'a4000000-0000-0000-0000-0000000000a1')->'artifact'->>'last_known_location',
  'Tavern',
  'artifact.last_known_location resolves from co-subject location perception');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_artifact_page('11111111-1111-1111-1111-111111111111',
                          'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                          'a4000000-0000-0000-0000-0000000000a1')->'artifact'->'inline_links') l
  WHERE l->>'id' = 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
), 'artifact.inline_links includes related actor links');

-- Timeline decay population
SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_timeline('11111111-1111-1111-1111-111111111111',
                     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->'records') r
  WHERE r->>'perception_id' = 'dca70000-0000-0000-0000-000000000a01'
    AND (r->'decay'->>'stale')::boolean = true
), 'timeline decay marks old records stale');

SELECT ok( EXISTS (
  SELECT 1
  FROM json_array_elements(
         fn_timeline('11111111-1111-1111-1111-111111111111',
                     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->'records') r
  WHERE r->>'perception_id' = 'dca70000-0000-0000-0000-000000000290'
    AND (r->'decay'->>'stale')::boolean = false
), 'timeline decay keeps fresh records non-stale');

-- Perception wall: visible via common knowledge, but no held perception details for Jonas.
SELECT ok(
  (fn_location_page('11111111-1111-1111-1111-111111111111',
                    'cccccccc-cccc-cccc-cccc-cccccccccccc',
                    'dddddddd-dddd-dddd-dddd-dddddddddddd')->'location'->>'current_synthesis') IS NULL,
  'perception wall: synthesis stays NULL when viewer has not perceived the location details');

SELECT is(
  (SELECT count(*)
   FROM json_array_elements(
          fn_location_page('11111111-1111-1111-1111-111111111111',
                           'cccccccc-cccc-cccc-cccc-cccccccccccc',
                           'dddddddd-dddd-dddd-dddd-dddddddddddd')->'location'->'key_actors'))::int,
  0,
  'perception wall: key_actors stays empty when viewer has no such perception');

SELECT * FROM finish();
ROLLBACK;
