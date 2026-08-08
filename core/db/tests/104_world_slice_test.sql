BEGIN;
SELECT plan(6);

-- Living World / Task 7 (Unit 5): fn_world_slice(p_world_id, p_scene) — the bounded WORLD-scope payload
-- handed to the World Actor (Task 8 calls this). It is the world-granularity sibling of fn_fact_sheet,
-- but TRUTH-side / world-omniscient: the World Actor is the ONE seat that sees the whole world, not a
-- scene, so this uses CANONICAL truth throughout (no fn_display_name, no perception gate — that wall
-- only applies to character-mind seats; RULINGS-2026-07-23 §9 exempts the referee).
--
-- Returns jsonb_build_object('ledger', …, 'presence', …, 'locations', …, 'recent', …, 'scene', …):
--   ledger    = pending_event rows for this world (status='pending'): pending_id/fire_at_tick/magnitude/payload.
--   presence  = {actor, location} for every actor in the world (actor_state.attrs.location_id path —
--               the same resolution fn_target_position / fn_actors_at use for an actor's scene).
--   locations = the world's locations (entity_registry entity_kind='location' + location_state.attrs).
--   recent    = a BOUNDED tail of recent world canon (ORDER BY in_world_tick DESC, beat_seq DESC LIMIT 20;
--               status='accepted' — proposed/rejected/superseded rows are not canon) — never the whole
--               history (bounded, never O(world²)).
--   scene     = the current location object for p_scene, nested, so the seat CAN aim at the player's scene.
--
-- Tested on the seeded PLAY world (22222222-…), scene = the Drowned Lantern tavern (210c…d1). The seed
-- places ALL FOUR actors (Kade/Mara/Jonas/Hooded Woman) in the tavern — nobody is elsewhere — so the
-- world-scope (not scene-scope) property is NOT free; this test INSERTs one actor at a DIFFERENT
-- location (Dock Street, 210c…d2) before asserting presence, so "presence includes an actor outside the
-- current scene" is a genuine world-scope proof and not a coincidence of the fixture.

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('2ac70000-0000-0000-0000-0000000000a5','22222222-2222-2222-2222-222222222222','actor','Dockhand');
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('2ac70000-0000-0000-0000-0000000000a5','22222222-2222-2222-2222-222222222222',
   '{"location_id":"210c0000-0000-0000-0000-0000000000d2"}'::jsonb);

-- ──────────────────────────────────────────────────────────────────────────────
-- (a) The result is jsonb, and carries exactly the five documented keys — no more, no less.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  jsonb_typeof(fn_world_slice('22222222-2222-2222-2222-222222222222',
                              '210c0000-0000-0000-0000-0000000000d1')),
  'object',
  '(a) fn_world_slice returns a jsonb object');

WITH slice AS (
  SELECT fn_world_slice('22222222-2222-2222-2222-222222222222',
                        '210c0000-0000-0000-0000-0000000000d1') AS j
)
SELECT is(
  (SELECT array_agg(k ORDER BY k) FROM slice, jsonb_object_keys(slice.j) AS k),
  ARRAY['ledger','locations','presence','recent','scene'],
  '(a2) the five documented keys, no more no less: ledger/locations/presence/recent/scene');

-- ──────────────────────────────────────────────────────────────────────────────
-- (b) ledger, presence, locations, recent, scene are all non-null (may be empty arrays, but never
--     json null) — the shape the World Actor's prompt assembly can always rely on.
-- ──────────────────────────────────────────────────────────────────────────────
WITH slice AS (
  SELECT fn_world_slice('22222222-2222-2222-2222-222222222222',
                        '210c0000-0000-0000-0000-0000000000d1') AS j
)
SELECT ok(
  (SELECT j->'ledger'    FROM slice) IS NOT NULL AND jsonb_typeof((SELECT j->'ledger'    FROM slice)) <> 'null'
  AND (SELECT j->'presence'  FROM slice) IS NOT NULL AND jsonb_typeof((SELECT j->'presence'  FROM slice)) <> 'null'
  AND (SELECT j->'locations' FROM slice) IS NOT NULL AND jsonb_typeof((SELECT j->'locations' FROM slice)) <> 'null'
  AND (SELECT j->'recent'    FROM slice) IS NOT NULL AND jsonb_typeof((SELECT j->'recent'    FROM slice)) <> 'null'
  AND (SELECT j->'scene'     FROM slice) IS NOT NULL AND jsonb_typeof((SELECT j->'scene'     FROM slice)) <> 'null',
  '(b) ledger/presence/locations/recent/scene are all non-null');

-- ──────────────────────────────────────────────────────────────────────────────
-- (c) THE WORLD-SCOPE PROOF: presence includes the Dockhand (at Dock Street, 210c…d2) — an actor NOT
--     in the current scene (the tavern, 210c…d1). A scene-scoped payload (like gather_slice) could
--     never surface this; only a true world-scope read can.
-- ──────────────────────────────────────────────────────────────────────────────
WITH slice AS (
  SELECT fn_world_slice('22222222-2222-2222-2222-222222222222',
                        '210c0000-0000-0000-0000-0000000000d1') AS j
), elems AS (
  SELECT elem FROM slice CROSS JOIN LATERAL jsonb_array_elements(slice.j->'presence') AS t(elem)
)
SELECT ok(
  EXISTS (
    SELECT 1 FROM elems
    WHERE elem->>'actor' = '2ac70000-0000-0000-0000-0000000000a5'
      AND elem->>'location' = '210c0000-0000-0000-0000-0000000000d2'
      AND elem->>'location' <> '210c0000-0000-0000-0000-0000000000d1'
  ),
  '(c) presence includes the Dockhand at Dock Street — an actor outside p_scene (the tavern): world-scope, not scene-scope');

-- ──────────────────────────────────────────────────────────────────────────────
-- (d) presence ALSO still carries the in-scene actors (e.g. Mara, at the tavern) — the payload is the
--     WHOLE world's presence, not a replacement of the scene-scoped view.
-- ──────────────────────────────────────────────────────────────────────────────
WITH slice AS (
  SELECT fn_world_slice('22222222-2222-2222-2222-222222222222',
                        '210c0000-0000-0000-0000-0000000000d1') AS j
), elems AS (
  SELECT elem FROM slice CROSS JOIN LATERAL jsonb_array_elements(slice.j->'presence') AS t(elem)
)
SELECT ok(
  EXISTS (
    SELECT 1 FROM elems
    WHERE elem->>'actor' = '2ac70000-0000-0000-0000-0000000000a2'  -- Mara
      AND elem->>'location' = '210c0000-0000-0000-0000-0000000000d1'
  ),
  '(d) presence still carries in-scene actors too (Mara, at the tavern) — world coverage, not a swap');

-- ──────────────────────────────────────────────────────────────────────────────
-- (e) scene resolves to the tavern's OWN location object (id = p_scene, canonical name = 'The Drowned
--     Lantern' — TRUTH-side canonical, never fn_display_name/perception-gated), and locations includes
--     all five seeded locations (Harbor Quarter, tavern, Dock Street, Alley, Cellar).
-- ──────────────────────────────────────────────────────────────────────────────
WITH slice AS (
  SELECT fn_world_slice('22222222-2222-2222-2222-222222222222',
                        '210c0000-0000-0000-0000-0000000000d1') AS j
)
SELECT ok(
  (SELECT j->'scene'->>'id' FROM slice) = '210c0000-0000-0000-0000-0000000000d1'
  AND (SELECT j->'scene'->>'name' FROM slice) = 'The Drowned Lantern'
  AND (SELECT jsonb_array_length(j->'locations') FROM slice) = 6,
  '(e) scene = the tavern''s own canonical location object; locations carries all 6 seeded locations');

SELECT * FROM finish();
ROLLBACK;
