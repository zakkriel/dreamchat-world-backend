BEGIN;
SELECT plan(3);

-- Three twins held by Player, about Mara, sourced to E1 (non-genesis). Identical except:
--  c1 = active (control), c2 = invalidated (invalid_tick set), c3 = expired (expired_at set).
INSERT INTO perception_record
  (perception_id, world_id, holder_id, source_event_id, content, epistemic_type,
   acquired_tick, valid_tick, invalid_tick, expired_at) VALUES
 ('c0000000-0000-0000-0000-0000000000c1','11111111-1111-1111-1111-111111111111',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-000000000001',
  'control: still believed','direct',100,100,NULL,NULL),
 ('c0000000-0000-0000-0000-0000000000c2','11111111-1111-1111-1111-111111111111',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-000000000001',
  'retracted: falsified in-world','direct',100,100,150,NULL),
 ('c0000000-0000-0000-0000-0000000000c3','11111111-1111-1111-1111-111111111111',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-000000000001',
  'expired: system-superseded','direct',100,100,NULL,now());
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('c0000000-0000-0000-0000-0000000000c1','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111'),
 ('c0000000-0000-0000-0000-0000000000c2','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111'),
 ('c0000000-0000-0000-0000-0000000000c3','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111');

-- NON-VACUITY CONTROL: the active twin IS present on Player's Mara page.
SELECT ok(
  EXISTS (
    SELECT 1
    FROM json_array_elements(
           fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                         'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
         json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'c0000000-0000-0000-0000-0000000000c1'),
  'control (active) perception PRESENT on Player''s Mara page — twins are otherwise eligible');

-- invalid_tick clause exercised: the retracted twin is ABSENT (would resurface if the clause were dropped).
SELECT ok(
  NOT EXISTS (
    SELECT 1
    FROM json_array_elements(
           fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                         'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
         json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'c0000000-0000-0000-0000-0000000000c2'),
  'invalidated perception ABSENT (invalid_tick IS NULL clause exercised)');

-- expired_at clause exercised: the expired twin is ABSENT.
SELECT ok(
  NOT EXISTS (
    SELECT 1
    FROM json_array_elements(
           fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                         'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
         json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'c0000000-0000-0000-0000-0000000000c3'),
  'expired perception ABSENT (expired_at IS NULL clause exercised)');

SELECT * FROM finish();
ROLLBACK;
