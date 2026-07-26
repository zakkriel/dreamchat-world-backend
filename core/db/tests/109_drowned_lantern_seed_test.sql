-- =====================================================================================
-- 109_drowned_lantern_seed_test.sql — asserts the Drowned Lantern seed
-- (seed_drowned_lantern.sql, loaded AFTER seed_mara_0A.sql by `make seed`).
-- Runs against the SEEDED, COMMITTED db (like the other 1111-world checks) — reads the
-- seed's fixed uuids inside a BEGIN/ROLLBACK envelope. Content canon:
-- docs/superpowers/specs/chunk-5.5-final/FINAL-drowned-lantern-souls.md.
--
-- The gate this file guards: approved souls became rows, secrets are subject-linked
-- PRIVATE records (never core traits), the wall (fn_isolated_npcs) trips on those secrets,
-- and the first playable room holds a real Tier-1 locked hatch.
-- =====================================================================================
BEGIN;
SELECT plan(10);

-- Fixed seed uuids (must match seed_drowned_lantern.sql):
--   world  11111111-…   kade/Player aaaaaaaa-…   Mara bbbbbbbb-…   Jonas cccccccc-…
--   hooded ffffffff-ffff-ffff-ffff-ffffffffffff
--   cellar hatch d1000000-0000-0000-0000-0000000000c3
--   Mara's secret perception d15ec000-0000-0000-0000-0000000000a1

-- (a) three cores exist (Mara, Jonas, hooded); Kade — the player, a premise not a mind — has none.
SELECT is(
  (SELECT count(*) FROM personality_core
   WHERE world_id='11111111-1111-1111-1111-111111111111'
     AND actor_id IN ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
                      'cccccccc-cccc-cccc-cccc-cccccccccccc',
                      'ffffffff-ffff-ffff-ffff-ffffffffffff'))::int,
  3, '(a) three personality cores exist: Mara, Jonas, hooded woman');
SELECT is(
  (SELECT count(*) FROM personality_core
   WHERE actor_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')::int,
  0, '(a) Kade (the player) has NO personality core');

-- (b) every real trait (object-valued key; schema_version / speech_manner are strings and skip)
-- has a trait_provenance row that points at a real canon_event. Zero traits float unexplained.
SELECT is(
  (SELECT count(*) FROM (
     SELECT pc.actor_id, t.key AS trait_key
     FROM personality_core pc, jsonb_each(pc.traits) t
     WHERE pc.world_id='11111111-1111-1111-1111-111111111111'
       AND jsonb_typeof(t.value)='object'
   ) traits
   LEFT JOIN trait_provenance tp
     ON tp.actor_id=traits.actor_id AND tp.trait_key=traits.trait_key
   LEFT JOIN canon_event ce ON ce.event_id=tp.event_id
   WHERE tp.event_id IS NULL OR ce.event_id IS NULL)::int,
  0, '(b) every core trait has a provenance row pointing at a real canon_event');

-- (c) Mara's secret is subject-linked to Kade (about-ness hard rule — the isolation lookup keys on it).
SELECT ok(
  EXISTS (SELECT 1 FROM perception_record pr
          JOIN perception_subject ps ON ps.perception_id=pr.perception_id
          WHERE pr.perception_id='d15ec000-0000-0000-0000-0000000000a1'
            AND pr.holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
            AND ps.entity_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
  '(c) Mara''s secret (held by Mara) is subject-linked to Kade');

-- (d) the wall trips: for an action bound to Kade with all four present, exactly Mara and the hooded
-- woman are isolated (their private records are about Kade). Jonas — who knows OF a debt but nothing
-- about Kade — stays in the shared batch.
SELECT set_eq(
  $$ SELECT actor_id FROM fn_isolated_npcs(
       '11111111-1111-1111-1111-111111111111',
       ARRAY['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa']::uuid[],
       ARRAY['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
             'cccccccc-cccc-cccc-cccc-cccccccccccc','ffffffff-ffff-ffff-ffff-ffffffffffff']::uuid[],
       ARRAY['bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','cccccccc-cccc-cccc-cccc-cccccccccccc',
             'ffffffff-ffff-ffff-ffff-ffffffffffff']::uuid[]) $$,
  $$ VALUES ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid),
            ('ffffffff-ffff-ffff-ffff-ffffffffffff'::uuid) $$,
  '(d) fn_isolated_npcs(action=Kade) = {Mara, hooded woman}; Jonas stays batched');

-- (e) the cellar hatch carries a real Tier-1 lock (locked=true) — the first locked Portal in play.
SELECT is(
  (SELECT attrs->>'locked' FROM artifact_state
   WHERE entity_id='d1000000-0000-0000-0000-0000000000c3'),
  'true', '(e) cellar hatch has Tier-1 locked=true in artifact_state');

-- (f) the wall-in-the-seed: no core's traits jsonb carries the secret strings. Recognition, the
-- life-debt, and how Mara knows Kade live ONLY in her private record — never in shared cognition.
SELECT is(
  (SELECT count(*) FROM personality_core
   WHERE world_id='11111111-1111-1111-1111-111111111111'
     AND (traits::text ILIKE '%Reyna%'
          OR traits::text ILIKE '%life-debt%'
          OR traits::text ILIKE '%knows_kade%'))::int,
  0, '(f) no personality_core traits leak Reyna / life-debt / knows_kade');

-- (g) about-ness is total: every perception_record in world 1111 has >=1 subject row.
SELECT is(
  (SELECT count(*) FROM perception_record pr
   WHERE pr.world_id='11111111-1111-1111-1111-111111111111'
     AND NOT EXISTS (SELECT 1 FROM perception_subject ps
                     WHERE ps.perception_id=pr.perception_id))::int,
  0, '(g) zero perception_records in world 1111 lack subject rows');

-- (h) founder-gate placement: Kade's LIVE actor_state puts him IN the Tavern — the arrival
-- ActorMoved @ tick 300 is the last mutation to touch him, so the projection reads the Tavern uuid
-- (he no longer starts in the Square). This is the fix that lets the founder open play inside the room.
SELECT is(
  (SELECT attrs->>'location_id' FROM actor_state
   WHERE world_id='11111111-1111-1111-1111-111111111111'
     AND entity_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
  'dddddddd-dddd-dddd-dddd-dddddddddddd',
  '(h) Kade''s live location_id is the Tavern (arrival @ tick 300 places him in the room)');

-- (i) his arrival perception is real, honest (sourced to the ActorMoved), and subject-linked to BOTH
-- the mover (Kade) and the room (the Tavern) — mirroring the move's about-ness, never faking who is present.
SELECT ok(
  EXISTS (SELECT 1 FROM perception_record
          WHERE perception_id='ca4e0000-0000-0000-0000-0000000000a1'
            AND holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
            AND source_event_id='e0000000-0000-0000-0000-0000000000fa')
  AND EXISTS (SELECT 1 FROM perception_subject
              WHERE perception_id='ca4e0000-0000-0000-0000-0000000000a1'
                AND entity_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')
  AND EXISTS (SELECT 1 FROM perception_subject
              WHERE perception_id='ca4e0000-0000-0000-0000-0000000000a1'
                AND entity_id='dddddddd-dddd-dddd-dddd-dddddddddddd'),
  '(i) Kade''s arrival perception exists (held by Kade, sourced to the ActorMoved) and is subject-linked to Kade + the Tavern');

SELECT * FROM finish();
ROLLBACK;
