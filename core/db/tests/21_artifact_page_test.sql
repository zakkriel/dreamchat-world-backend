BEGIN;
SELECT plan(4);
SELECT is( fn_artifact_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1')->>'schema_version',
           'artifact_page/1', 'schema_version is artifact_page/1');
-- name withheld (NULL) — existence perceived, canon name never substituted
SELECT ok( (fn_artifact_page('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1')
             ->'artifact'->>'perceived_name') IS NULL,
           'note name withheld on the artifact page (NULL, not the canon name)');
-- discovery observation present for Player
SELECT ok( EXISTS (
    SELECT 1 FROM json_array_elements(
      fn_artifact_page('11111111-1111-1111-1111-111111111111',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','a4000000-0000-0000-0000-0000000000a1')
        ->'artifact'->'collected_knowledge_groups') g,
      json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000b01'),
  'Player artifact page contains the discovery observation');
-- Jonas → NULL (existence withheld → 404)
SELECT ok( fn_artifact_page('11111111-1111-1111-1111-111111111111',
             'cccccccc-cccc-cccc-cccc-cccccccccccc','a4000000-0000-0000-0000-0000000000a1') IS NULL,
           'Jonas artifact page for the note is NULL → 404 (existence not leaked)');
SELECT * FROM finish();
ROLLBACK;
