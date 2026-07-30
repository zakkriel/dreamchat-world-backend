-- =====================================================================================
-- 116_drowned_lantern_space_test.sql — the Drowned Lantern gets SPACE (Station F, Task 7).
-- Asserts the play world's hand-authored §3 spatial layer + the §2/§4/§5.3 engine reads that
-- make the scene PLAYABLE (the founder's exit, Task 8, is Kade walking to the bar in a browser).
--
-- Runs against the SEEDED, COMMITTED db (like 109), reading the seed's fixed uuids inside a
-- BEGIN/ROLLBACK envelope. Content canon (coordinates are a SANCTIONED hand-authored test artifact,
-- spec §3 — "the hand-placed seed world is a test artifact, not the pattern"; production mints them):
--   docs/superpowers/specs/chunk-5.5-final/FINAL-drowned-lantern-souls.md
--   core/db/seeds/seed_drowned_lantern.sql
--
-- Fixed seed uuids (must match seed_drowned_lantern.sql):
--   world  22222222-…       Kade a1  Mara a2  Jonas a3  hooded a4
--   Harbor Quarter of Vael (parent location)  210c0000-…-d0
--   The Drowned Lantern (tavern) …-d1   Dock Street …-d2   Alley …-d3   Cellar …-d4
--   the bar (fixed room feature) 2a7f0000-…-f1   ballast crate (container) …-f2   ballast stone …-f3
--
-- The COORDINATE LAYOUT this file pins (see the seed for the authored values):
--   Harbor Quarter frame (meters): tavern {200,200}, dock street {280,200} (80 m ⇒ CEIL(80/1.4)=58 s),
--     alley {200,240} (40 m), cellar {205,205} (beneath). tavern↔dock-street is the longer, sensible walk.
--   Tavern local frame (meters): Kade {6,1} at the door, the bar {6,9} along the back wall
--     ⇒ fn_distance(Kade, bar) = 8 m ⇒ CEIL(8/1.4) = 6 s (the in-scene approach; fits tense's 30 s).
--     Mara {6,10} behind the bar, Jonas {5,8} by it, the hooded woman {1,1} at the corner table.
-- =====================================================================================
BEGIN;
SELECT plan(5);

-- (a) THE in-scene distance is real geometry: Kade → the bar is a few meters, not 0/NULL. The nearest
-- common parent is the tavern (both share the scene), so fn_distance reads their own in-scene
-- coordinates: sqrt((6-6)^2 + (9-1)^2) = 8 m. Assert a few-meter band (robust to numeric form) so the
-- test states the SHAPE (a small positive number), not a brittle float.
SELECT ok(
  fn_distance('22222222-2222-2222-2222-222222222222',
              '2ac70000-0000-0000-0000-0000000000a1',   -- Kade
              '2a7f0000-0000-0000-0000-0000000000f1')    -- the bar
    > 0
  AND fn_distance('22222222-2222-2222-2222-222222222222',
                  '2ac70000-0000-0000-0000-0000000000a1',
                  '2a7f0000-0000-0000-0000-0000000000f1') < 15,
  '(a) fn_distance(Kade, bar) is a few meters (0 < d < 15), not 0/NULL — computed 8 m');

-- (b) The in-scene APPROACH fits the tense beat budget. fn_move_duration_actor's p_from/p_to are
-- ENTITIES (fn_distance is entity-agnostic): the meaningful "in-tavern approach" per the task Context is
-- Kade → the bar (the literal same-LOCATION pair (kade, tavern, tavern) is distance 0 → 0 s, a degenerate
-- pair that exercises no geometry). Kade carries no movement statuses ⇒ speed = seeded walk 1.4 m/s, so
-- duration = CEIL(8 / 1.4) = 6 s. tense = 30 s (core/api/tension.go tensionBudgetSeconds) ⇒ 6 s fits with
-- room to spare, which is exactly what Task 8's "walk to the bar" needs.
SELECT ok(
  fn_move_duration_actor('22222222-2222-2222-2222-222222222222',
                         '2ac70000-0000-0000-0000-0000000000a1',   -- Kade (supplies the speed)
                         '2ac70000-0000-0000-0000-0000000000a1',   -- from: Kade's position
                         '2a7f0000-0000-0000-0000-0000000000f1')    -- to:   the bar
    BETWEEN 1 AND 30
  AND fn_move_duration_actor('22222222-2222-2222-2222-222222222222',
                             '2ac70000-0000-0000-0000-0000000000a1',
                             '2ac70000-0000-0000-0000-0000000000a1',
                             '2a7f0000-0000-0000-0000-0000000000f1') = 6,
  '(b) in-scene approach Kade→bar = CEIL(8/1.4) = 6 s: a few seconds, fits the tense 30 s beat budget');

-- (c) Portals gate traversal (§5.3): the front door (open ∧ unlocked) permits tavern→dock street so
-- Kade can leave; the cellar hatch (locked) blocks tavern→cellar so he can't descend. Both are wired by
-- the base seed — this asserts Task 7 did not break them.
SELECT ok(
  fn_portal_permits('22222222-2222-2222-2222-222222222222',
                    '210c0000-0000-0000-0000-0000000000d1',   -- tavern
                    '210c0000-0000-0000-0000-0000000000d2') = true   -- dock street
  AND
  fn_portal_permits('22222222-2222-2222-2222-222222222222',
                    '210c0000-0000-0000-0000-0000000000d1',   -- tavern
                    '210c0000-0000-0000-0000-0000000000d4') = false, -- cellar (hatch locked)
  '(c) fn_portal_permits(tavern, dock_street)=true (front door); (tavern, cellar)=false (hatch locked)');

-- (d) The ObjectRelocated physics (§4) has something to bite on in play: a Container instance (the
-- ballast crate carries max_room + empty_weight) AND a heavy thing. The crate's recursive effective
-- weight = (empty_weight 8 + effective_weight(ballast stone 92)) × 1 = 100 > Kade's max_load 80, so
-- "grab the crate → encumbered" is REACHABLE (the eager rule flips encumbered on that commit).
SELECT ok(
  (SELECT attrs ? 'max_room' FROM artifact_state
     WHERE world_id='22222222-2222-2222-2222-222222222222'
       AND entity_id='2a7f0000-0000-0000-0000-0000000000f2')   -- ballast crate (container)
  AND fn_effective_weight('22222222-2222-2222-2222-222222222222',
                          '2a7f0000-0000-0000-0000-0000000000f2') > 80,
  '(d) ballast crate has max_room; fn_effective_weight = (8 + 92) = 100 > Kade''s max_load 80');

-- (e) The scene is inhabited in SPACE: all four actors + the bar sit in the tavern AND carry coordinates
-- in its local frame (so fn_distance can measure any pair, and the narrator/engine read real positions).
SELECT is(
  (SELECT count(*)::int FROM (
     SELECT entity_id FROM actor_state
       WHERE world_id='22222222-2222-2222-2222-222222222222'
         AND entity_id IN ('2ac70000-0000-0000-0000-0000000000a1',   -- Kade
                           '2ac70000-0000-0000-0000-0000000000a2',   -- Mara
                           '2ac70000-0000-0000-0000-0000000000a3',   -- Jonas
                           '2ac70000-0000-0000-0000-0000000000a4')   -- hooded woman
         AND attrs->>'location_id'='210c0000-0000-0000-0000-0000000000d1'
         AND attrs ? 'coordinates'
     UNION ALL
     SELECT entity_id FROM artifact_state
       WHERE world_id='22222222-2222-2222-2222-222222222222'
         AND entity_id='2a7f0000-0000-0000-0000-0000000000f1'        -- the bar
         AND attrs->>'location_id'='210c0000-0000-0000-0000-0000000000d1'
         AND attrs ? 'coordinates'
   ) s),
  5,
  '(e) Kade, Mara, Jonas, the hooded woman + the bar each carry coordinates and sit in the tavern');

SELECT * FROM finish();
ROLLBACK;
