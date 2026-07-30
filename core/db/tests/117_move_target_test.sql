-- =====================================================================================
-- 117_move_target_test.sql — Station F Task 8: unify the move — TARGET any positioned entity.
-- The move EVENT is no longer location-keyed: ActorMoved carries to_target_id (the id of ANY
-- positioned entity — a location, an object like the bar, an actor), and the commit sets the mover's
-- attrs.coordinates to the target's position AND attrs.location_id to the target's containing scene.
-- A same-scene move (the target's scene == the actor's current location) changes only coordinates —
-- that falls out naturally (location_id written unchanged-in-value); a cross-location move changes
-- location_id, so the §5.3 portal gate applies exactly then. This is ONE operation at every layer:
-- "approach the bar" (in-scene) and "walk to the dock street" (cross-location) are the SAME move.
--
-- Runs against the SEEDED, COMMITTED db (like 116) inside a BEGIN/ROLLBACK envelope, reading the
-- seed's fixed uuids. Ticks 6000+ are well above the seed's max (Kade arrives at tick 50) and clear of
-- test-85's transient world-2222 use (ticks 10–22). Content canon + coordinates (a SANCTIONED
-- hand-authored test artifact, §3):
--   docs/superpowers/specs/chunk-5.5-final/FINAL-action-contracts.md §2/§3
--   core/db/seeds/seed_drowned_lantern.sql
--
-- Fixed seed uuids:
--   world 22222222-…   Kade a1
--   The Drowned Lantern (tavern) …-d1   Dock Street …-d2   Cellar …-d4
--   the bar (fixed room feature) 2a7f0000-…-f1
--   Kade {6,1} at the door; the bar {6,9} along the back wall ⇒ 8 m ⇒ CEIL(8/1.4)=6 s.
--   Front door (open ∧ ¬locked) tavern↔dock-street; cellar hatch (LOCKED) tavern↔cellar.
--
-- Each committing move is run as its OWN statement (captured into a temp table) BEFORE the assertion
-- reads state — a subquery in the SAME statement as apply_event would read the pre-commit snapshot and
-- never see the mutation. Execution order: (e) reads the duration BEFORE any move (Kade still at the
-- door {6,1}); the in-scene and rejected cases keep Kade in the tavern; the one cross-location commit
-- that moves Kade to dock street runs LAST so it never disturbs the earlier tavern-anchored cases.
-- =====================================================================================
BEGIN;
SELECT plan(5);

-- (e) Duration is real geometry over the TARGET: fn_move_duration_actor(world, Kade, bar) — the 3-arg
-- form (the actor's position is implicit; the target supplies the destination). Kade {6,1} → bar {6,9}
-- = 8 m, no movement statuses ⇒ speed = seeded walk 1.4 ⇒ CEIL(8/1.4) = 6. Read FIRST, before (a) moves
-- Kade's coordinate onto the bar (which would make the Kade↔bar distance 0).
SELECT is(
  fn_move_duration_actor('22222222-2222-2222-2222-222222222222',
                         '2ac70000-0000-0000-0000-0000000000a1',   -- Kade (supplies the speed + origin)
                         '2a7f0000-0000-0000-0000-0000000000f1'),   -- the bar (the target)
  6::bigint,
  '(e) fn_move_duration_actor(world, Kade, bar) = CEIL(8/1.4) = 6 (3-arg target form; actor origin implicit)');

-- (a) IN-SCENE move to an OBJECT: Kade → the bar (same tavern). Commits; Kade's coordinates become the
-- bar's {6,9}; his location_id is UNCHANGED (still the tavern) — a same-scene move repositions without
-- changing scene. Run the move first (its own statement), then assert the committed state.
CREATE TEMP TABLE r_a AS SELECT (apply_event('22222222-2222-2222-2222-222222222222',
    '2ac70000-0000-0000-0000-0000000000a1',
    '{"type":"ActorMoved","stated":"Kade crosses to the bar","to_target_id":"2a7f0000-0000-0000-0000-0000000000f1"}'::jsonb,
    6000, 0, 'freeform'))->>'halt_reason' AS halt;
SELECT ok(
  (SELECT halt FROM r_a) = 'committed'
  AND (SELECT attrs->'coordinates' FROM actor_state
         WHERE world_id='22222222-2222-2222-2222-222222222222'
           AND entity_id='2ac70000-0000-0000-0000-0000000000a1') = '{"x":6,"y":9}'::jsonb
  AND (SELECT attrs->>'location_id' FROM actor_state
         WHERE world_id='22222222-2222-2222-2222-222222222222'
           AND entity_id='2ac70000-0000-0000-0000-0000000000a1') = '210c0000-0000-0000-0000-0000000000d1',
  '(a) in-scene Kade→bar commits; coordinates now the bar''s {6,9}; location_id unchanged (still the tavern)');

-- (d) IN-SCENE move needs NO portal: there is no Portal between Kade and the bar (the bar is a fixed
-- feature, not a portal endpoint), yet the reposition commits — a same-scene move is not a traversal.
SELECT is(
  (apply_event('22222222-2222-2222-2222-222222222222',
      '2ac70000-0000-0000-0000-0000000000a1',
      '{"type":"ActorMoved","stated":"Kade leans back to the bar","to_target_id":"2a7f0000-0000-0000-0000-0000000000f1"}'::jsonb,
      6001, 0, 'freeform'))->>'halt_reason',
  'committed',
  '(d) in-scene reposition to the bar needs NO portal (none connects Kade and the bar) — commits anyway');

-- (c) CROSS-LOCATION move to the LOCKED cellar → gate_reject, nothing written. Kade is in the tavern
-- (the earlier moves were in-scene); the cellar hatch connects tavern↔cellar but is LOCKED, so the
-- §5.3 portal gate blocks passage. Blocker-only proof: zero canon rows at this tick.
SELECT ok(
  ((apply_event('22222222-2222-2222-2222-222222222222',
      '2ac70000-0000-0000-0000-0000000000a1',
      '{"type":"ActorMoved","stated":"Kade tries the cellar hatch","to_target_id":"210c0000-0000-0000-0000-0000000000d4"}'::jsonb,
      6002, 0, 'freeform'))->>'halt_reason' = 'gate_reject')
  AND (SELECT count(*)::int FROM canon_event
         WHERE world_id='22222222-2222-2222-2222-222222222222' AND in_world_tick=6002) = 0,
  '(c) cross-location Kade→locked cellar → gate_reject (hatch locked), nothing written');

-- (b) CROSS-LOCATION move to a LOCATION: Kade → dock street. Commits (the front door is open ∧
-- unlocked); location_id → dock street. Moving to a LOCATION lands the actor at the location's local
-- origin/authored entry (§3 accepted imprecision — no exit-point inference). Runs LAST (changes scene).
CREATE TEMP TABLE r_b AS SELECT (apply_event('22222222-2222-2222-2222-222222222222',
    '2ac70000-0000-0000-0000-0000000000a1',
    '{"type":"ActorMoved","stated":"Kade steps out the front door","to_target_id":"210c0000-0000-0000-0000-0000000000d2"}'::jsonb,
    6003, 0, 'freeform'))->>'halt_reason' AS halt;
SELECT ok(
  (SELECT halt FROM r_b) = 'committed'
  AND (SELECT attrs->>'location_id' FROM actor_state
         WHERE world_id='22222222-2222-2222-2222-222222222222'
           AND entity_id='2ac70000-0000-0000-0000-0000000000a1') = '210c0000-0000-0000-0000-0000000000d2',
  '(b) cross-location Kade→dock street commits (front door permits); location_id → dock street');

SELECT * FROM finish();
ROLLBACK;
