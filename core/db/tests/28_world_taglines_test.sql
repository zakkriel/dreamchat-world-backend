-- The seeded worlds carry their founder-approved taglines, verbatim, and the directory ships them.
--
-- This test exists because of the failure mode SPEC-031 recorded: the seeds insert
-- `ON CONFLICT (world_id) DO NOTHING`, so adding a column to the INSERT changes nothing at all for a
-- world that already exists — the change lands green and the only world anyone plays never sees it.
-- The seeds now DO UPDATE the tagline specifically; this asserts the result rather than the
-- intention, against the real seeded rows.
BEGIN;
SELECT plan(5);

SELECT is(
  (SELECT tagline FROM world WHERE world_id = '22222222-2222-2222-2222-222222222222'),
  'A harbor town where everyone is owed something, and the tide keeps the ledger.',
  'The Drowned Lantern carries its approved tagline verbatim');

SELECT is(
  (SELECT tagline FROM world WHERE world_id = '11111111-1111-1111-1111-111111111111'),
  'A test world. Two people, one room, and every rule watching.',
  'the Mara 0A fixture carries its approved tagline verbatim');

-- Authored fiction has to survive the trip to the card unedited: no truncation, no title-casing, no
-- "…" — the frontend decides how much of it to show, this service decides nothing.
SELECT is(
  (SELECT w->>'tagline' FROM json_array_elements(fn_world_directory()->'worlds') w
    WHERE w->>'id' = '22222222-2222-2222-2222-222222222222'),
  (SELECT tagline FROM world WHERE world_id = '22222222-2222-2222-2222-222222222222'),
  'the directory ships the authored line unchanged');

SELECT is(
  (SELECT count(*)::int FROM world WHERE tagline IS NOT NULL),
  2,
  'both seeded worlds are authored — a world with no line would ship null, not a composed one');

-- The gate that makes a cover possible at all: fillScenes renders a world only when it has a
-- tagline, so an approved line is exactly what unblocks its cover.
SELECT ok(
  EXISTS (SELECT 1 FROM world
           WHERE world_id = '22222222-2222-2222-2222-222222222222'
             AND tagline IS NOT NULL AND length(btrim(tagline)) > 0),
  'the play world now has the authored target its cover is generated from');

SELECT * FROM finish();
ROLLBACK;
