BEGIN;
SELECT plan(5);
-- Player/Tavern: page present, schema, name via CK, contains the tavern observation
SELECT is( fn_location_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','dddddddd-dddd-dddd-dddd-dddddddddddd')->>'schema_version',
           'location_page/1', 'schema_version is location_page/1');
SELECT is( fn_location_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','dddddddd-dddd-dddd-dddd-dddddddddddd')
             ->'location'->>'perceived_name', 'Tavern', 'Tavern name via common knowledge');
SELECT ok( EXISTS (
    SELECT 1 FROM json_array_elements(
      fn_location_page('11111111-1111-1111-1111-111111111111',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','dddddddd-dddd-dddd-dddd-dddddddddddd')
        ->'location'->'collected_knowledge_groups') g,
      json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000c01'),
  'Player Tavern page contains the tavern observation');
-- Jonas/Tavern: present (CK) but empty collected knowledge
SELECT ok( fn_location_page('11111111-1111-1111-1111-111111111111',
             'cccccccc-cccc-cccc-cccc-cccccccccccc','dddddddd-dddd-dddd-dddd-dddddddddddd') IS NOT NULL,
           'Jonas Tavern page returns (Tavern is common knowledge)');
SELECT is( (SELECT count(*) FROM json_array_elements(
             fn_location_page('11111111-1111-1111-1111-111111111111',
               'cccccccc-cccc-cccc-cccc-cccccccccccc','dddddddd-dddd-dddd-dddd-dddddddddddd')
               ->'location'->'collected_knowledge_groups'))::int,
           0, 'Jonas Tavern page has empty collected knowledge (perception-bound)');
SELECT * FROM finish();
ROLLBACK;
