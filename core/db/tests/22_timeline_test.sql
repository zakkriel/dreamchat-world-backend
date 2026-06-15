BEGIN;
SELECT plan(7);
SELECT is( fn_timeline('11111111-1111-1111-1111-111111111111',
             'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->>'schema_version',
           'timeline/1', 'schema_version is timeline/1');
-- note observation is on Player's own timeline (PRESENT) ...
SELECT ok( EXISTS (SELECT 1 FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->'records') r
             WHERE r->>'perception_id'='dca70000-0000-0000-0000-000000000b01'),
           'note observation PRESENT on Player timeline');
-- ... and ABSENT from Jonas's (same perception_id — paired)
SELECT ok( NOT EXISTS (SELECT 1 FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'cccccccc-cccc-cccc-cccc-cccccccccccc')->'records') r
             WHERE r->>'perception_id'='dca70000-0000-0000-0000-000000000b01'),
           'note observation ABSENT from Jonas timeline');
-- THE LEAK GATE (corrected): the planted secret ("hidden ledger", E1) is on Player's timeline ...
SELECT ok( EXISTS (SELECT 1 FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->'records') r
             WHERE r->>'content' ILIKE '%ledger%'),
           'planted secret PRESENT on Player timeline (he holds the shared-of-E1 record)');
-- ... and NEVER on Jonas's, even though Jonas's timeline is NOT empty (he holds his own moves).
-- (The Chunk-3 "Jonas ignorant" claim is E1-scoped; the noise loop still gives Jonas 13 self-moves.)
SELECT ok( NOT EXISTS (SELECT 1 FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'cccccccc-cccc-cccc-cccc-cccccccccccc')->'records') r
             WHERE r->>'content' ILIKE '%ledger%'),
           'planted secret ABSENT from Jonas timeline (perception-bound; secret never leaks)');
-- non-vacuity: Jonas's timeline IS populated (his own noise self-observations) so the absence
-- assertions above are not passing on an empty page.
SELECT ok( (SELECT count(*) FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'cccccccc-cccc-cccc-cccc-cccccccccccc')->'records'))::int > 0,
           'Jonas timeline is non-empty (his own movements) — absence gate is non-vacuous');
-- before_tick=101 keeps only rows with valid_tick < 101 (tick-100 rows in, tick>=101 noise out)
SELECT ok( (SELECT coalesce(max((r->>'occurred_at_tick')::int), 0) FROM json_array_elements(
             fn_timeline('11111111-1111-1111-1111-111111111111',
               'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 101)->'records') r) < 101,
           'before_tick=101 excludes all rows at tick >= 101');
SELECT * FROM finish();
ROLLBACK;
