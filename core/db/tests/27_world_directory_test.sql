-- fn_world_directory (world_directory/2) — the world picker's card fields.
--
-- The three additions are a tagline (authored fiction, never composed here), a cover image (the same
-- image_ref/1 shape a portrait uses), and last_place_label (a LABEL and nothing else). Everything
-- below is asserted against worlds this test creates, so it does not depend on what the seed happens
-- to contain.
BEGIN;
SELECT plan(10);

-- w1: authored, playable, player standing somewhere
INSERT INTO world (world_id, display_name, tagline, player_entity_id) VALUES
  ('d1000000-0000-0000-0000-000000000001', 'ZZ Directory Fixture',
   'A line the world wrote about itself.', 'd1ac0000-0000-0000-0000-0000000000a1');
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('d1ac0000-0000-0000-0000-0000000000a1','d1000000-0000-0000-0000-000000000001','actor','Wanderer'),
  ('d110c000-0000-0000-0000-0000000000c1','d1000000-0000-0000-0000-000000000001','location','Lamp Row');
INSERT INTO actor_state (world_id, entity_id, attrs) VALUES
  ('d1000000-0000-0000-0000-000000000001','d1ac0000-0000-0000-0000-0000000000a1',
   '{"location_id":"d110c000-0000-0000-0000-0000000000c1"}'::jsonb);
INSERT INTO location_state (world_id, entity_id, attrs) VALUES
  ('d1000000-0000-0000-0000-000000000001','d110c000-0000-0000-0000-0000000000c1',
   '{"descriptor":"a row of drowned lamps"}'::jsonb);

-- w2: playable, but the player has never been placed anywhere
INSERT INTO world (world_id, display_name, player_entity_id) VALUES
  ('d2000000-0000-0000-0000-000000000002', 'ZZ Unentered Fixture', 'd2ac0000-0000-0000-0000-0000000000a1');
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('d2ac0000-0000-0000-0000-0000000000a1','d2000000-0000-0000-0000-000000000002','actor','Nobody');

-- w3: created but uninhabited — no player at all
INSERT INTO world (world_id, display_name) VALUES
  ('d3000000-0000-0000-0000-000000000003', 'ZZ Uninhabited Fixture');

CREATE TEMP VIEW dir AS
SELECT w->>'id' AS id, w FROM json_array_elements(fn_world_directory()->'worlds') w;

SELECT is(fn_world_directory()->>'schema_version', 'world_directory/2',
  'the directory declares world_directory/2');

-- ── tagline: authored fiction, verbatim, never composed ─────────────────────────────────────────
SELECT is((SELECT w->>'tagline' FROM dir WHERE id='d1000000-0000-0000-0000-000000000001'),
  'A line the world wrote about itself.',
  'an authored tagline reaches the card verbatim');

SELECT ok((SELECT json_typeof(w->'tagline') = 'null' FROM dir WHERE id='d2000000-0000-0000-0000-000000000002'),
  'a world whose author wrote no tagline ships null — this service never composes one (GA-2)');

-- A blank is not a second spelling of absent.
SELECT throws_ok($$
  UPDATE world SET tagline = '   ' WHERE world_id='d1000000-0000-0000-0000-000000000001' $$,
  '23514', NULL, 'a blank tagline is rejected at the column, so "" can never mean absent');

-- ── cover_image: image_ref/1, null until generated ──────────────────────────────────────────────
SELECT ok((SELECT json_typeof(w->'cover_image') = 'null' FROM dir WHERE id='d1000000-0000-0000-0000-000000000001'),
  'cover_image is null until a cover exists — the ordinary state');

INSERT INTO image_slot (world_id, owner_kind, owner_id, asset_id) VALUES
  ('d1000000-0000-0000-0000-000000000001','world','d1000000-0000-0000-0000-000000000001','asset_cover_zz');

SELECT is((SELECT w->'cover_image'->>'schema_version' FROM dir WHERE id='d1000000-0000-0000-0000-000000000001'),
  'image_ref/1', 'a filled cover ships the same image_ref/1 shape as a portrait');

-- A PATH back to this service, never a presigned URL: those expire, so an embedded one would rot in
-- any cache or log.
SELECT is((SELECT w->'cover_image'->>'path' FROM dir WHERE id='d1000000-0000-0000-0000-000000000001'),
  '/worlds/d1000000-0000-0000-0000-000000000001/images/asset_cover_zz',
  'the cover carries a path back to this service, not a URL');

-- ── last_place_label: a label, and nothing else ─────────────────────────────────────────────────
SELECT is((SELECT w->>'last_place_label' FROM dir WHERE id='d1000000-0000-0000-0000-000000000001'),
  fn_display_name('d1000000-0000-0000-0000-000000000001',
                  'd1ac0000-0000-0000-0000-0000000000a1',
                  'd110c000-0000-0000-0000-0000000000c1'),
  'last_place_label is the world player''s OWN naming of where they stand');

SELECT ok((SELECT json_typeof(w->'last_place_label') = 'null' FROM dir WHERE id='d2000000-0000-0000-0000-000000000002'),
  'a player who has never been placed anywhere yields null — never entered, nothing to say');

SELECT ok((SELECT json_typeof(w->'last_place_label') = 'null' FROM dir WHERE id='d3000000-0000-0000-0000-000000000003'),
  'a world with no player yet yields null rather than guessing at somebody');

SELECT * FROM finish();
ROLLBACK;
