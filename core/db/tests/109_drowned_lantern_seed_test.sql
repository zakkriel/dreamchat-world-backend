-- =====================================================================================
-- 109_drowned_lantern_seed_test.sql — asserts the Drowned Lantern seed
-- (seed_drowned_lantern.sql, loaded by `make seed` into its OWN dedicated PLAY world
-- 22222222-2222-2222-2222-222222222222 — founder Option B: play/fixture separation).
-- Runs against the SEEDED, COMMITTED db (like the other seed checks) — reads the seed's
-- fixed uuids inside a BEGIN/ROLLBACK envelope. Content canon:
-- docs/law/rulings/FINAL-drowned-lantern-souls.md.
--
-- The gate this file guards: approved souls became rows, secrets are subject-linked
-- PRIVATE records (never core traits), the wall (fn_isolated_npcs) trips on those secrets,
-- the first playable room holds a real Tier-1 locked hatch, and ALL FOUR souls stand IN the
-- Drowned Lantern (the play world is set up ready to open in the room).
-- =====================================================================================
BEGIN;
SELECT plan(17);

-- Fixed seed uuids (must match seed_drowned_lantern.sql):
--   world  22222222-…   Kade 2ac70000-…-a1   Mara …-a2   Jonas …-a3   hooded …-a4
--   the Drowned Lantern (tavern) 210c0000-0000-0000-0000-0000000000d1
--   cellar hatch 2a7f0000-0000-0000-0000-0000000000c3
--   Mara's secret perception 2ce50000-0000-0000-0000-0000000000a1

-- (a) three cores exist (Mara, Jonas, hooded); Kade — the player, a premise not a mind — has none.
SELECT is(
  (SELECT count(*) FROM personality_core
   WHERE world_id='22222222-2222-2222-2222-222222222222'
     AND actor_id IN ('2ac70000-0000-0000-0000-0000000000a2',
                      '2ac70000-0000-0000-0000-0000000000a3',
                      '2ac70000-0000-0000-0000-0000000000a4'))::int,
  3, '(a) three personality cores exist: Mara, Jonas, hooded woman');
SELECT is(
  (SELECT count(*) FROM personality_core
   WHERE actor_id='2ac70000-0000-0000-0000-0000000000a1')::int,
  0, '(a) Kade (the player) has NO personality core');

-- (b) every real trait (object-valued key; schema_version / speech_manner are strings and skip)
-- has a trait_provenance row that points at a real canon_event. Zero traits float unexplained.
SELECT is(
  (SELECT count(*) FROM (
     SELECT pc.actor_id, t.key AS trait_key
     FROM personality_core pc, jsonb_each(pc.traits) t
     WHERE pc.world_id='22222222-2222-2222-2222-222222222222'
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
          WHERE pr.perception_id='2ce50000-0000-0000-0000-0000000000a1'
            AND pr.holder_id='2ac70000-0000-0000-0000-0000000000a2'
            AND ps.entity_id='2ac70000-0000-0000-0000-0000000000a1'),
  '(c) Mara''s secret (held by Mara) is subject-linked to Kade');

-- (d) the wall trips: for an action bound to Kade with all four present, exactly Mara and the hooded
-- woman are isolated (their private records are about Kade). Jonas — who knows OF a debt but nothing
-- about Kade — stays in the shared batch.
SELECT set_eq(
  $$ SELECT actor_id FROM fn_isolated_npcs(
       '22222222-2222-2222-2222-222222222222',
       ARRAY['2ac70000-0000-0000-0000-0000000000a1']::uuid[],
       ARRAY['2ac70000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a2',
             '2ac70000-0000-0000-0000-0000000000a3','2ac70000-0000-0000-0000-0000000000a4']::uuid[],
       ARRAY['2ac70000-0000-0000-0000-0000000000a2','2ac70000-0000-0000-0000-0000000000a3',
             '2ac70000-0000-0000-0000-0000000000a4']::uuid[]) $$,
  $$ VALUES ('2ac70000-0000-0000-0000-0000000000a2'::uuid),
            ('2ac70000-0000-0000-0000-0000000000a4'::uuid) $$,
  '(d) fn_isolated_npcs(action=Kade) = {Mara, hooded woman}; Jonas stays batched');

-- (e) the cellar hatch carries a real Tier-1 lock (locked=true) — the first locked Portal in play.
SELECT is(
  (SELECT attrs->>'locked' FROM artifact_state
   WHERE entity_id='2a7f0000-0000-0000-0000-0000000000c3'),
  'true', '(e) cellar hatch has Tier-1 locked=true in artifact_state');

-- (f) the wall-in-the-seed: no core's traits jsonb carries the secret strings. Recognition, the
-- life-debt, and how Mara knows Kade live ONLY in her private record — never in shared cognition.
SELECT is(
  (SELECT count(*) FROM personality_core
   WHERE world_id='22222222-2222-2222-2222-222222222222'
     AND (traits::text ILIKE '%Reyna%'
          OR traits::text ILIKE '%life-debt%'
          OR traits::text ILIKE '%knows_kade%'))::int,
  0, '(f) no personality_core traits leak Reyna / life-debt / knows_kade');

-- (g) about-ness is total: every perception_record in the play world has >=1 subject row.
SELECT is(
  (SELECT count(*) FROM perception_record pr
   WHERE pr.world_id='22222222-2222-2222-2222-222222222222'
     AND NOT EXISTS (SELECT 1 FROM perception_subject ps
                     WHERE ps.perception_id=pr.perception_id))::int,
  0, '(g) zero perception_records in the play world lack subject rows');

-- (h) all four souls stand IN the Drowned Lantern — replay-safe placement (residents at the scene
-- genesis @ tick 40, Kade's arrival @ tick 50) puts every one of them at the tavern uuid. This is the
-- founder-gate: the play world opens with everyone in the room, no fixture scatter.
SELECT is(
  (SELECT count(*) FROM actor_state
   WHERE world_id='22222222-2222-2222-2222-222222222222'
     AND entity_id IN ('2ac70000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a2',
                       '2ac70000-0000-0000-0000-0000000000a3','2ac70000-0000-0000-0000-0000000000a4')
     AND attrs->>'location_id'='210c0000-0000-0000-0000-0000000000d1')::int,
  4, '(h) all four actors'' live location_id is the Drowned Lantern');

-- (i) the room has its REAL name — no fixture naming constraint anymore.
SELECT is(
  (SELECT canonical_name FROM entity_registry
   WHERE entity_id='210c0000-0000-0000-0000-0000000000d1'
     AND world_id='22222222-2222-2222-2222-222222222222'),
  'The Drowned Lantern',
  '(i) the tavern''s canonical_name is ''The Drowned Lantern''');

-- (j) the Drowned Lantern carries its Tier-2 scene DESCRIPTION verbatim from FINAL (Defect B): the
-- narrate PLACE line renders it, so the room's fixed character is DATA, not the narrator's invention.
SELECT is(
  (SELECT attrs->>'description' FROM location_state
   WHERE entity_id='210c0000-0000-0000-0000-0000000000d1'
     AND world_id='22222222-2222-2222-2222-222222222222'),
  'Low beams, salt-rot, one hearth, a bar with a hatch, a back door to the alley.',
  '(j) the Drowned Lantern has its Tier-2 scene description in location_state');

-- ── §3 naming reach in the play world (Defect C) — the founder-gate leak, closed ─────────────
-- ids: Kade a1, Mara a2, Jonas a3, hooded a4.
-- (k) Kade knows Mara's name (five winters back) — his display name for her is 'Mara'.
SELECT is( fn_display_name('22222222-2222-2222-2222-222222222222',
             '2ac70000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a2'),
           'Mara', '(k) Kade''s view of Mara = her name (he knew her)');
-- (l) Kade does NOT know Jonas — his display name for Jonas is the DESCRIPTOR, never the canonical name.
SELECT is( fn_display_name('22222222-2222-2222-2222-222222222222',
             '2ac70000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a3'),
           'the muscle by the bar', '(l) Kade''s view of Jonas = descriptor (he knows him only as the muscle)');
-- (m) The founder-gate leak's other half: Jonas does NOT know Kade's name — never 'Kade'.
SELECT isnt( fn_display_name('22222222-2222-2222-2222-222222222222',
               '2ac70000-0000-0000-0000-0000000000a3','2ac70000-0000-0000-0000-0000000000a1'),
             'Kade', '(m) Jonas''s view of Kade is NOT the canonical name ''Kade''');
SELECT is( fn_display_name('22222222-2222-2222-2222-222222222222',
             '2ac70000-0000-0000-0000-0000000000a3','2ac70000-0000-0000-0000-0000000000a1'),
           'a young stranger, dark-haired', '(m) …it is Kade''s descriptor');
-- (n) Mara PRIVATELY knows Kade — as "Reyna's brother", her knowledge alone; NO other viewer resolves it.
SELECT is( fn_display_name('22222222-2222-2222-2222-222222222222',
             '2ac70000-0000-0000-0000-0000000000a2','2ac70000-0000-0000-0000-0000000000a1'),
           'Reyna''s brother', '(n) Mara''s view of Kade = her private name for him');
-- (o) BATCH intersection over {Jonas, hooded}: neither knows Kade's name → the descriptor, never 'Kade'.
SELECT is( fn_batch_display_name('22222222-2222-2222-2222-222222222222',
             ARRAY['2ac70000-0000-0000-0000-0000000000a3','2ac70000-0000-0000-0000-0000000000a4']::uuid[],
             '2ac70000-0000-0000-0000-0000000000a1'),
           'a young stranger, dark-haired', '(o) batch {Jonas,hooded} labels Kade by descriptor (no batch mind knows his name)');

SELECT * FROM finish();
ROLLBACK;
