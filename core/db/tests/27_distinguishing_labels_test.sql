BEGIN;
SELECT plan(6);

-- Distinguishing detail in colliding labels (founder ruling, 2026-08-08). When two things a viewer
-- can see wear the SAME label, the label gains perceived detail so the player can name which one.
-- Detail is added ONLY when it is both needed and useful; otherwise the plain label stands.

-- ── On the seeded play world: two hooded figures, one at the bar, one by the ballast crate ────────
-- Kade has no name for either, so both resolve to the descriptor "a hooded figure".

-- (1) The collision is broken, and by something the viewer can actually see.
SELECT is(
  (SELECT label FROM fn_display_names_distinct(
     '22222222-2222-2222-2222-222222222222'::uuid,'2ac70000-0000-0000-0000-0000000000a1'::uuid,
     ARRAY['2ac70000-0000-0000-0000-0000000000a4','2ac70000-0000-0000-0000-0000000000aa']::uuid[])
    WHERE entity_id='2ac70000-0000-0000-0000-0000000000aa'),
  'a hooded figure by the bar',
  'the figure at the bar is named by what he is standing next to');

SELECT is(
  (SELECT label FROM fn_display_names_distinct(
     '22222222-2222-2222-2222-222222222222'::uuid,'2ac70000-0000-0000-0000-0000000000a1'::uuid,
     ARRAY['2ac70000-0000-0000-0000-0000000000a4','2ac70000-0000-0000-0000-0000000000aa']::uuid[])
    WHERE entity_id='2ac70000-0000-0000-0000-0000000000a4'),
  'a hooded figure by the ballast crate',
  'the other figure is named by a DIFFERENT anchor — the two are now distinguishable');

-- (2) Silence when nothing collides: a unique label is returned byte-identical. Detail is a response
--     to ambiguity, not decoration — nothing is renamed when nothing is ambiguous.
SELECT is(
  (SELECT label FROM fn_display_names_distinct(
     '22222222-2222-2222-2222-222222222222'::uuid,'2ac70000-0000-0000-0000-0000000000a1'::uuid,
     ARRAY['2ac70000-0000-0000-0000-0000000000a2','2ac70000-0000-0000-0000-0000000000a4']::uuid[])
    WHERE entity_id='2ac70000-0000-0000-0000-0000000000a2'),
  'Mara',
  'a label with no rival keeps its plain form');

-- (3) The SAME entity, asked about alone, stays plain: collision is a property of the group, not the
--     entity. Ask about one hooded figure and there is nothing to disambiguate from.
SELECT is(
  (SELECT label FROM fn_display_names_distinct(
     '22222222-2222-2222-2222-222222222222'::uuid,'2ac70000-0000-0000-0000-0000000000a1'::uuid,
     ARRAY['2ac70000-0000-0000-0000-0000000000aa']::uuid[])),
  'a hooded figure',
  'alone in the set, the same figure needs no detail — ambiguity is a property of the group');

-- ── THE HONEST EDGE, ruled explicitly: detail that cannot separate is not invented ────────────────
-- Two identical figures in a bare room with exactly ONE thing to stand near. Both anchor to it, so
-- the anchor distinguishes nothing and both labels stay plain. The player is told the truth: these
-- two look the same. Inventing "the first" or "the taller" would assert what the viewer cannot see.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('c0111000-0000-0000-0000-000000000001','c0111000-0000-0000-0000-0000000000ff','location','ROOM-SECRET'),
  ('c0111000-0000-0000-0000-000000000002','c0111000-0000-0000-0000-0000000000ff','actor','TWIN-A-SECRET'),
  ('c0111000-0000-0000-0000-000000000003','c0111000-0000-0000-0000-0000000000ff','actor','TWIN-B-SECRET'),
  ('c0111000-0000-0000-0000-000000000004','c0111000-0000-0000-0000-0000000000ff','actor','WATCHER'),
  ('c0111000-0000-0000-0000-000000000005','c0111000-0000-0000-0000-0000000000ff','artifact','THE-ONLY-THING');
INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('c0111000-0000-0000-0000-000000000001','c0111000-0000-0000-0000-0000000000ff',
   jsonb_build_object('coordinates', jsonb_build_object('x',0,'y',0)));
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('c0111000-0000-0000-0000-000000000002','c0111000-0000-0000-0000-0000000000ff',
   jsonb_build_object('location_id','c0111000-0000-0000-0000-000000000001','descriptor','a cloaked twin',
                      'coordinates', jsonb_build_object('x',1,'y',1))),
  ('c0111000-0000-0000-0000-000000000003','c0111000-0000-0000-0000-0000000000ff',
   jsonb_build_object('location_id','c0111000-0000-0000-0000-000000000001','descriptor','a cloaked twin',
                      'coordinates', jsonb_build_object('x',1,'y',2))),
  ('c0111000-0000-0000-0000-000000000004','c0111000-0000-0000-0000-0000000000ff',
   jsonb_build_object('location_id','c0111000-0000-0000-0000-000000000001','descriptor','a watcher',
                      'coordinates', jsonb_build_object('x',9,'y',9)));
INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
  ('c0111000-0000-0000-0000-000000000005','c0111000-0000-0000-0000-0000000000ff',
   jsonb_build_object('location_id','c0111000-0000-0000-0000-000000000001','descriptor','a lone stool',
                      'coordinates', jsonb_build_object('x',1,'y',0)));

SELECT is(
  (SELECT count(DISTINCT label)::int FROM fn_display_names_distinct(
     'c0111000-0000-0000-0000-0000000000ff'::uuid,'c0111000-0000-0000-0000-000000000004'::uuid,
     ARRAY['c0111000-0000-0000-0000-000000000002','c0111000-0000-0000-0000-000000000003']::uuid[])),
  1,
  'twins with the same nearest anchor stay identical — the world does not invent a difference');

SELECT is(
  (SELECT DISTINCT label FROM fn_display_names_distinct(
     'c0111000-0000-0000-0000-0000000000ff'::uuid,'c0111000-0000-0000-0000-000000000004'::uuid,
     ARRAY['c0111000-0000-0000-0000-000000000002','c0111000-0000-0000-0000-000000000003']::uuid[])),
  'a cloaked twin',
  'and the label they share is the plain one — never a canonical name, never a fabricated detail');

SELECT * FROM finish();
ROLLBACK;
