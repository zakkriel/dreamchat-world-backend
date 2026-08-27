-- migrate:up

ALTER TABLE world
  ADD COLUMN template_key text,
  ADD COLUMN archived_at timestamptz;

COMMENT ON COLUMN world.template_key IS
  'Template lineage key so worlds can be re-instantiated without deleting canon (append-only, ADR-001/006).';
COMMENT ON COLUMN world.archived_at IS
  'Superseded marker so retired worlds leave canon intact but drop out of active directory listings (append-only, ADR-001/006).';

CREATE OR REPLACE FUNCTION public.fn_world_directory() RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
    'schema_version', 'world_directory/2',
    'worlds', COALESCE((
      SELECT json_agg(json_build_object(
               'id',            w.world_id,
               'display_name',  w.display_name,
               'tagline',       w.tagline,
               'theme',         w.theme,
               'playable',      w.player_entity_id IS NOT NULL,
               'cover_image',   fn_image_ref(w.world_id, 'world', w.world_id),
               'last_place_label', (
                  SELECT fn_display_name(w.world_id, w.player_entity_id,
                                         (a.attrs->>'location_id')::uuid)
                    FROM actor_state a
                   WHERE a.world_id = w.world_id
                     AND a.entity_id = w.player_entity_id
                     AND a.attrs->>'location_id' IS NOT NULL
               )
             ) ORDER BY w.display_name, w.world_id)
        FROM world w
       WHERE w.archived_at IS NULL), '[]'::json)
  );
$$;

CREATE OR REPLACE FUNCTION public.fn_instantiate_drowned_lantern(p_world_id uuid, p_pin jsonb DEFAULT NULL)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
  v_kade uuid := COALESCE((p_pin->>'kade')::uuid, gen_random_uuid());
  v_mara uuid := COALESCE((p_pin->>'mara')::uuid, gen_random_uuid());
  v_jonas uuid := COALESCE((p_pin->>'jonas')::uuid, gen_random_uuid());
  v_hooded_woman uuid := COALESCE((p_pin->>'hooded_woman')::uuid, gen_random_uuid());
  v_hooded_companion uuid := COALESCE((p_pin->>'hooded_companion')::uuid, gen_random_uuid());
  v_harbor_quarter uuid := COALESCE((p_pin->>'harbor_quarter')::uuid, gen_random_uuid());
  v_drowned_lantern uuid := COALESCE((p_pin->>'drowned_lantern')::uuid, gen_random_uuid());
  v_dock_street uuid := COALESCE((p_pin->>'dock_street')::uuid, gen_random_uuid());
  v_alley uuid := COALESCE((p_pin->>'alley')::uuid, gen_random_uuid());
  v_cellar uuid := COALESCE((p_pin->>'cellar')::uuid, gen_random_uuid());
  v_harbormaster_office uuid := COALESCE((p_pin->>'harbormaster_office')::uuid, gen_random_uuid());
  v_sealed_note uuid := COALESCE((p_pin->>'sealed_note')::uuid, gen_random_uuid());
  v_front_door uuid := COALESCE((p_pin->>'front_door')::uuid, gen_random_uuid());
  v_back_door uuid := COALESCE((p_pin->>'back_door')::uuid, gen_random_uuid());
  v_cellar_hatch uuid := COALESCE((p_pin->>'cellar_hatch')::uuid, gen_random_uuid());
  v_office_door uuid := COALESCE((p_pin->>'office_door')::uuid, gen_random_uuid());
  v_cellar_key uuid := COALESCE((p_pin->>'cellar_key')::uuid, gen_random_uuid());
  v_bar_fixture uuid := COALESCE((p_pin->>'bar_fixture')::uuid, gen_random_uuid());
  v_ballast_crate uuid := COALESCE((p_pin->>'ballast_crate')::uuid, gen_random_uuid());
  v_ballast_stone uuid := COALESCE((p_pin->>'ballast_stone')::uuid, gen_random_uuid());
  v_event_m_e1 uuid := COALESCE((p_pin->>'event_m_e1')::uuid, gen_random_uuid());
  v_event_m_e2 uuid := COALESCE((p_pin->>'event_m_e2')::uuid, gen_random_uuid());
  v_event_m_e3 uuid := COALESCE((p_pin->>'event_m_e3')::uuid, gen_random_uuid());
  v_event_m_e4_private uuid := COALESCE((p_pin->>'event_m_e4_private')::uuid, gen_random_uuid());
  v_event_j_e1 uuid := COALESCE((p_pin->>'event_j_e1')::uuid, gen_random_uuid());
  v_event_j_e2 uuid := COALESCE((p_pin->>'event_j_e2')::uuid, gen_random_uuid());
  v_event_j_e3_private uuid := COALESCE((p_pin->>'event_j_e3_private')::uuid, gen_random_uuid());
  v_event_h_e1_private uuid := COALESCE((p_pin->>'event_h_e1_private')::uuid, gen_random_uuid());
  v_event_scene_genesis uuid := COALESCE((p_pin->>'event_scene_genesis')::uuid, gen_random_uuid());
  v_event_kade_arrival uuid := COALESCE((p_pin->>'event_kade_arrival')::uuid, gen_random_uuid());
  v_event_world_genesis uuid := COALESCE((p_pin->>'event_world_genesis')::uuid, gen_random_uuid());
  v_perception_mara_secret uuid := COALESCE((p_pin->>'perception_mara_secret')::uuid, gen_random_uuid());
  v_perception_jonas_secret uuid := COALESCE((p_pin->>'perception_jonas_secret')::uuid, gen_random_uuid());
  v_perception_hooded_contract uuid := COALESCE((p_pin->>'perception_hooded_contract')::uuid, gen_random_uuid());
  v_perception_kade_arrival uuid := COALESCE((p_pin->>'perception_kade_arrival')::uuid, gen_random_uuid());
  v_name_perception_kade_knows_mara uuid := COALESCE((p_pin->>'name_perception_kade_knows_mara')::uuid, gen_random_uuid());
  v_name_perception_mara_knows_jonas uuid := COALESCE((p_pin->>'name_perception_mara_knows_jonas')::uuid, gen_random_uuid());
  v_name_perception_jonas_knows_mara uuid := COALESCE((p_pin->>'name_perception_jonas_knows_mara')::uuid, gen_random_uuid());
  v_name_perception_mara_knows_kade uuid := COALESCE((p_pin->>'name_perception_mara_knows_kade')::uuid, gen_random_uuid());
BEGIN
-- Own idempotence guard: refuse a double-load. `make reset` is the clean re-run path.
-- Guard on the target world itself — any existing registry row means this template already ran there.
IF EXISTS (SELECT 1 FROM entity_registry WHERE world_id = p_world_id) THEN
  RAISE EXCEPTION 'fn_instantiate_drowned_lantern already applied for world % — run `make reset` for a clean load', p_world_id;
END IF;

-- Physics defaults for the play world (contracts §2: exactly walk 1.4 + encumbered -100 on walk).
PERFORM seed_world_defaults(p_world_id);

-- ── Registry: 4 actors + 5 locations (REAL names) + 8 artifacts ───────────────────────
-- All-new fixed uuids (entity_registry PK is global). Kade is 'Kade' now — a real name, not the
-- fixture world's 'Player'. The tavern is 'The Drowned Lantern', not 'Tavern'.
--
-- Task 7 (Station F) adds the SPATIAL layer (§3 nested coordinates): a parent location 'Harbor Quarter
-- of Vael' (…-d0) over the four rooms, a fixed room feature 'the bar' (…-f1, the anchor Kade walks to),
-- and a Container instance 'ballast crate' (…-f2) holding a 'ballast stone' (…-f3) so the §4 ObjectRelocated
-- physics has a heavy thing to bite on in play. Coordinates are a SANCTIONED hand-authored test artifact
-- (spec §3 — the hand-placed seed world is a test artifact; production mints coordinates via Task 6).
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 (v_kade,p_world_id,'actor',   'Kade'),
 (v_mara,p_world_id,'actor',   'Mara'),
 (v_jonas,p_world_id,'actor',   'Jonas'),
 (v_hooded_woman,p_world_id,'actor',   'Hooded Woman'),
 -- A SECOND hooded figure at the same table. Kade has no name for either, so fn_display_name renders
 -- both as the identical descriptor 'a hooded figure' — which is the point: "ask the hooded figure
 -- about the note" now names two people equally well, and decompose must refuse to guess (UNRESOLVED)
 -- instead of silently picking one. Until this, every candidate in the room resolved uniquely and the
 -- UNRESOLVED path could not be reached in play at all. Id is …aa, not the next free …a5: pgTAP's
 -- 104_world_slice_test mints …a5 itself, and entity_registry's PK is global, so the seed and the
 -- tests share one id space. Anything added here needs a suffix no test already claims.
 (v_hooded_companion,p_world_id,'actor',   'Hooded Companion'),
 (v_harbor_quarter,p_world_id,'location','Harbor Quarter of Vael'),
 (v_drowned_lantern,p_world_id,'location','The Drowned Lantern'),
 (v_dock_street,p_world_id,'location','Dock Street'),
 (v_alley,p_world_id,'location','Alley'),
 (v_cellar,p_world_id,'location','Cellar'),
 -- SPEC-030 (founder-named, 2026-08-08): the Harbormaster's Office, off Dock Street and far enough up
 -- the quarter that walking there cannot fit in one beat — the first destination in this world that
 -- starts a JOURNEY rather than an instant arrival. See the geometry note in the spatial block.
 (v_harbormaster_office,p_world_id,'location','Harbormaster''s Office'),
 (v_sealed_note,p_world_id,'artifact','Sealed Note (gray wax)'),
 (v_front_door,p_world_id,'artifact','Front Door'),
 (v_back_door,p_world_id,'artifact','Back Door'),
 (v_cellar_hatch,p_world_id,'artifact','Cellar Hatch'),
 (v_office_door,p_world_id,'artifact','Office Door'),
 (v_cellar_key,p_world_id,'artifact','Cellar Key'),
 (v_bar_fixture,p_world_id,'artifact','the bar'),
 (v_ballast_crate,p_world_id,'artifact','Ballast Crate'),
 (v_ballast_stone,p_world_id,'artifact','Ballast Stone');

-- ── Backstory canon events (ticks 30–37) + one scene-genesis event (tick 40) ──────────
-- event_type='AttributeChanged' (backstory grounds who they are); origin='fast_path'. M-E4 / J-E3 /
-- H-E1 are PRIVATE — each grounds exactly one NPC's private perception below. The scene-genesis event
-- (f9) is public and carries the room state AND places the three residents (Mara behind the bar,
-- Jonas by the bar, the hooded woman at the corner table) via absolute location writes.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 (v_event_m_e1,p_world_id,'AttributeChanged',
  'M-E1: grew up behind this bar; her father taught her a keeper who reacts has already lost',30,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_m_e2,p_world_id,'AttributeChanged',
  'M-E2: the harbormaster''s predecessor shook the tavern for protection money; the watch shrugged; her father died that winter',31,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_m_e3,p_world_id,'AttributeChanged',
  'M-E3: a dock brawl left Jonas half-dead outside her door; she stitched him up and gave him work',32,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_m_e4_private,p_world_id,'AttributeChanged',
  'M-E4 (private): she hid Reyna''s family in the cellar nine days; Reyna''s teenage brother ran the messages that got them out',33,0,
  'Backstory','accepted',now(),'private','fast_path'),
 (v_event_j_e1,p_world_id,'AttributeChanged',
  'J-E1: beaten near to death over a fixed fight and left in the alley; Mara took him in',34,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_j_e2,p_world_id,'AttributeChanged',
  'J-E2: a prizefighter until he killed a man in the ring with one unlucky blow; never fought clean for money again',35,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_j_e3_private,p_world_id,'AttributeChanged',
  'J-E3 (private): twice he watched Mara go pale at a harbor face and learned to stand closer instead of asking',36,0,
  'Backstory','accepted',now(),'private','fast_path'),
 (v_event_h_e1_private,p_world_id,'AttributeChanged',
  'H-E1 (private): took the paymaster''s contract in a counting-house above the silk quay, three days ago',37,0,
  'Backstory','accepted',now(),'private','fast_path'),
 (v_event_scene_genesis,p_world_id,'AttributeChanged',
  'the Drowned Lantern is set: Mara behind the bar, Jonas by it, a hooded woman at the corner table; tension, the doors, the hatch, the note, the key',40,0,
  'Scene','accepted',now(),'public','fast_path');

-- Participants (brief: the NPC + any named co-subject). subject ≠ about-ness (perception_subject
-- carries the precise about-ness, ADR-035) — these are the event''s people. The scene-genesis event
-- names the room (setting) and the three residents it places.
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 (v_event_m_e1,v_mara,'actor','subject'),
 (v_event_m_e2,v_mara,'actor','subject'),
 (v_event_m_e3,v_mara,'actor','subject'),
 (v_event_m_e4_private,v_mara,'actor','subject'),
 (v_event_m_e4_private,v_kade,'actor','co_subject'),
 (v_event_j_e1,v_jonas,'actor','subject'),
 (v_event_j_e1,v_mara,'actor','co_subject'),
 (v_event_j_e2,v_jonas,'actor','subject'),
 (v_event_j_e3_private,v_jonas,'actor','subject'),
 (v_event_j_e3_private,v_mara,'actor','co_subject'),
 (v_event_h_e1_private,v_hooded_woman,'actor','subject'),
 (v_event_scene_genesis,v_drowned_lantern,'location','setting'),
 (v_event_scene_genesis,v_mara,'actor','subject'),
 (v_event_scene_genesis,v_jonas,'actor','subject'),
 (v_event_scene_genesis,v_hooded_woman,'actor','subject');

-- ── personality_core — WHO THEY ARE IN THE ROOM. No secret ever lives here. ───────────
-- traits jsonb: real traits are objects {value, manner}; schema_version + speech_manner are
-- strings. Kade gets NO core (premise, not a mind). Malleability per FINAL (Mara 0.25 / Jonas 0.45 /
-- hooded 0.6).
INSERT INTO personality_core (world_id, actor_id, traits, malleability) VALUES
 (p_world_id,v_mara,
  '{"schema_version":"traits/1",
    "guarded":{"value":0.8,"manner":"answers questions with questions; volunteers nothing"},
    "dry_witted":{"value":0.7,"manner":"deflects with humor before she deflects with silence"},
    "loyal_to_jonas":{"value":0.9,"manner":"treats Jonas as family; will not see him harmed"},
    "distrusts_authority":{"value":0.85,"manner":"the harbormaster''s men drink free and learn nothing"},
    "steady_under_pressure":{"value":0.8,"manner":"the last in the room to raise her voice"},
    "speech_manner":"short sentences; harbor slang; calls strangers sailor regardless of trade; never says a name she was not given"}'::jsonb,
  0.25),
 (p_world_id,v_jonas,
  '{"schema_version":"traits/1",
    "protective_of_mara":{"value":0.9,"manner":"reads every stranger as a threat to her first, himself second"},
    "slow_to_speak":{"value":0.7,"manner":"acts before he explains; three words where others use ten"},
    "brawler_not_killer":{"value":0.8,"manner":"ends fights; does not start them; hates blades"},
    "debt_of_gratitude":{"value":0.85,"manner":"the tavern is the only place that ever took him back"},
    "speech_manner":"monosyllables; states facts not opinions; uses names only for Mara"}'::jsonb,
  0.45),
 (p_world_id,v_hooded_woman,
  '{"schema_version":"traits/1",
    "watchful":{"value":0.8,"manner":"tracks the door and every hand near a purse"},
    "unhurried":{"value":0.7,"manner":"never the first to move, never the last to leave"},
    "clean_coin":{"value":0.6,"manner":"pays in coin too clean for this district"},
    "speech_manner":"says little; asks less; watches much"}'::jsonb,
  0.6);

-- ── trait_provenance — every trait traces to a backstory event (D-11 for character) ───
INSERT INTO trait_provenance (world_id, actor_id, trait_key, event_id) VALUES
 -- Mara
 (p_world_id,v_mara,'dry_witted',           v_event_m_e1),
 (p_world_id,v_mara,'steady_under_pressure',v_event_m_e1),
 (p_world_id,v_mara,'guarded',              v_event_m_e2),
 (p_world_id,v_mara,'distrusts_authority',  v_event_m_e2),
 (p_world_id,v_mara,'loyal_to_jonas',       v_event_m_e3),
 -- Jonas
 (p_world_id,v_jonas,'protective_of_mara',   v_event_j_e1),
 (p_world_id,v_jonas,'debt_of_gratitude',    v_event_j_e1),
 (p_world_id,v_jonas,'slow_to_speak',        v_event_j_e2),
 (p_world_id,v_jonas,'brawler_not_killer',   v_event_j_e2),
 -- Hooded woman (one event grounds her thin core)
 (p_world_id,v_hooded_woman,'watchful',   v_event_h_e1_private),
 (p_world_id,v_hooded_woman,'unhurried',  v_event_h_e1_private),
 (p_world_id,v_hooded_woman,'clean_coin', v_event_h_e1_private);

-- ── Private knowledge — perception_record WITH subject links (the whole point) ────────
-- Only the holder holds a perception of each private event → private to the lookups. Fixed
-- perception_ids so the subject links are unambiguous. source_event_id is NOT NULL (grounded).
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 -- Mara''s secret: recognition + the life-debt + how she knows him ("Reyna''s brother").
 (v_perception_mara_secret,p_world_id,
  v_mara,v_event_m_e4_private,
  'Kade is Reyna''s brother — the boy who ran messages while I hid Reyna''s family in the cellar nine days, five winters back. I owe that family a life-debt I have never said aloud. To him, and to this room, I am a stranger; if the wrong people learn I know him, the debt gets us both killed.',
  'direct',33,33),
 -- Jonas knows OF a secret without knowing IT: a debt she never explains. (No "Reyna", no "ledger".)
 (v_perception_jonas_secret,p_world_id,
  v_jonas,v_event_j_e3_private,
  'Mara keeps a knife under the till and a debt she never explains. Twice in four years I have watched her go pale at a face off the harbor. I have learned to stand closer and not to ask. I do not know what it is.',
  'inference',36,36),
 -- The hooded woman''s contract: a description and a purse, and one word of doubt — "Yet." (founder-trimmed;
 -- NO characterization of the note''s contents.)
 (v_perception_hooded_contract,p_world_id,
  v_hooded_woman,v_event_h_e1_private,
  'The paymaster''s coin bought a description: a courier, young, dark-haired, moves like a dock rat — and a purse for whoever confirms him. The one by the door could be him. I am not sure. Yet.',
  'told',37,37);

-- about-ness (RULINGS §6): Mara''s secret → Kade AND Mara; Jonas → Mara; hooded → Kade.
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 (v_perception_mara_secret,v_kade,p_world_id),
 (v_perception_mara_secret,v_mara,p_world_id),
 (v_perception_jonas_secret,v_mara,p_world_id),
 (v_perception_hooded_contract,v_kade,p_world_id);

-- ── Scene state via state_mutation (event f9) — projects through sm_project; replay-safe ──
-- Single-key absolute sets under attrs (ABSOLUTE-STATE-SETS, was 0A Rider B). Tier-1 keys: open, locked, connects, size,
-- weight, tension (see core/api/tier1.go). carry is the single Tier-1 key `contained_by` (§4 eager
-- encumbrance requires carry to be engine-readable state; the former Tier-2 carried_by/held_by are
-- unified into it — contents of X = entities whose contained_by = X, actors are root carriers). connects is the
-- Portal''s [room, room] pair. The cellar hatch is closed and LOCKED — the first Tier-1 lock in play.
-- The three residents are PLACED here (absolute attrs.location_id → the Drowned Lantern); Kade arrives
-- separately below. Each (entity, attribute_path) is written exactly once → replay-order-independent.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- residents in the room (Mara behind the bar, Jonas by it, the hooded woman at the corner table)
 (p_world_id,v_event_scene_genesis,v_mara,'actor',   'attrs.location_id', to_jsonb(v_drowned_lantern::text), 40,0),
 (p_world_id,v_event_scene_genesis,v_jonas,'actor',   'attrs.location_id', to_jsonb(v_drowned_lantern::text), 40,1),
 (p_world_id,v_event_scene_genesis,v_hooded_woman,'actor',   'attrs.location_id', to_jsonb(v_drowned_lantern::text), 40,2),
 -- the room reads calm; two of the four people in it are pretending → tension 'tense'
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.tension',     to_jsonb('tense'::text),                                                                            40,3),
 -- art_note: sealed, near-weightless, carried by Kade. Contents deliberately UNAUTHORED (Tier-2 flavor only).
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.size',                 to_jsonb(1),                                                                                    40,4),
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.weight',               to_jsonb(0),                                                                                    40,5),
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.sealed_with_gray_wax', to_jsonb(true),                                                                                 40,6),
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.contained_by',         to_jsonb(v_kade::text),                                          40,7),
 -- front door: OPEN, unlocked, tavern↔dock street
 (p_world_id,v_event_scene_genesis,v_front_door,'artifact','attrs.open',                 to_jsonb(true),                                                                                 40,8),
 (p_world_id,v_event_scene_genesis,v_front_door,'artifact','attrs.locked',               to_jsonb(false),                                                                                40,9),
 (p_world_id,v_event_scene_genesis,v_front_door,'artifact','attrs.connects',             jsonb_build_array(v_drowned_lantern,v_dock_street), 40,10),
 -- back door: closed, unlocked, tavern↔alley
 (p_world_id,v_event_scene_genesis,v_back_door,'artifact','attrs.open',                 to_jsonb(false),                                                                                40,11),
 (p_world_id,v_event_scene_genesis,v_back_door,'artifact','attrs.locked',               to_jsonb(false),                                                                                40,12),
 (p_world_id,v_event_scene_genesis,v_back_door,'artifact','attrs.connects',             jsonb_build_array(v_drowned_lantern,v_alley), 40,13),
 -- cellar hatch: closed and LOCKED (Tier-1), tavern↔cellar. Mara holds the key; the cellar is where M-E4 happened.
 (p_world_id,v_event_scene_genesis,v_cellar_hatch,'artifact','attrs.open',                 to_jsonb(false),                                                                                40,14),
 (p_world_id,v_event_scene_genesis,v_cellar_hatch,'artifact','attrs.locked',               to_jsonb(true),                                                                                 40,15),
 (p_world_id,v_event_scene_genesis,v_cellar_hatch,'artifact','attrs.connects',             jsonb_build_array(v_drowned_lantern,v_cellar), 40,16),
 -- the cellar key, held by Mara
 (p_world_id,v_event_scene_genesis,v_cellar_key,'artifact','attrs.size',                 to_jsonb(1),                                                                                    40,17),
 (p_world_id,v_event_scene_genesis,v_cellar_key,'artifact','attrs.contained_by',         to_jsonb(v_mara::text),                                          40,18),
 -- Tier-2 scene DESCRIPTION per location (Defect B): the narrate PLACE line renders it, so the room's
 -- fixed character is DATA the narrator draws on, never something it invents. The Drowned Lantern text
 -- is verbatim from FINAL-drowned-lantern-souls.md's scene section; the three stubs are brief, honest
 -- one-liners so movement has somewhere with a described face to go.
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.description', to_jsonb('Low beams, salt-rot, one hearth, a bar with a hatch, a back door to the alley.'::text), 40,19),
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.description', to_jsonb('A rain-slick harbor road; gulls, tar, and black water past the pilings.'::text),        40,20),
 (p_world_id,v_event_scene_genesis,v_alley,'location','attrs.description', to_jsonb('A narrow dead-end behind the tavern; stacked crates and standing water.'::text),         40,21),
 (p_world_id,v_event_scene_genesis,v_cellar,'location','attrs.description', to_jsonb('A cold stone undercroft beneath the tavern; barrels, damp, one shuttered lantern.'::text),   40,22);

-- ── §3 SPATIAL LAYER (Station F Task 7) — the scene gets space, under the same scene-genesis event f9 ──
-- Nested coordinates (FINAL-action-contracts.md §3): every location has a coordinate WITHIN its parent
-- (attrs.coordinates) + a parent edge (attrs.parent_location_id, Tier-1 string); a parent carries an
-- attrs.area outlining its children (an ordered ring of ≥3 points, founder ruling R12 — no {w,h} box).
-- Things inside a scene (actors + fixed features) carry a coordinate in that scene's LOCAL frame.
-- fn_distance measures any pair at their nearest common parent's frame; fn_place_at measures which
-- child's area contains a point.
-- Coordinates are a SANCTIONED hand-authored test artifact (§3); production mints them (Task 6). Each
-- (entity, attribute_path) is written EXACTLY ONCE → replay-order-independent (ABSOLUTE-STATE-SETS, was 0A Rider B; D-1). Tier-1 keys
-- only for engine-read attrs (coordinates, parent_location_id, max_room, empty_weight, weight, size,
-- contained_by, area — fn_place_at reads it, so it is engine-read, not descriptive). seq 26+ continues
-- f9's single monotonic seq space.
--
-- Harbor Quarter frame (meters): tavern {200,200}; dock street {207,200} → 7 m ⇒ CEIL(7/1.4)=5 s, a SHORT
-- STEP out the front door onto the harbor road (Task 11 seed tune, RULINGS-2026-07-30 §1: the playable
-- moves must fit the beat budget — 5 s fits the tense 30 s budget so "step out the front" plays; the
-- earlier {280,200} put it 80 m ⇒ 58 s away, an over-budget dead end. Dock Street stays a DISTINCT
-- location behind the front-door portal — just a short step, not merged into the tavern); alley {200,240}
-- → 40 m (out the back); cellar {205,205} → beneath the tavern (portal-locked anyway). The quarter's
-- 2000×2000 area (a four-corner outline) bounds them.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- Harbor Quarter of Vael: the root parent (no parent edge), its own origin + the area outline that bounds the rooms.
 (p_world_id,v_event_scene_genesis,v_harbor_quarter,'location','attrs.coordinates',        '{"x":0,"y":0}'::jsonb,       40,26),
 (p_world_id,v_event_scene_genesis,v_harbor_quarter,'location','attrs.area',               '{"points":[{"x":0,"y":0},{"x":2000,"y":0},{"x":2000,"y":2000},{"x":0,"y":2000}]}'::jsonb, 40,27),
 -- the four rooms: each a child of Harbor Quarter with a coordinate in the quarter frame.
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,28),
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.coordinates',        '{"x":200,"y":200}'::jsonb,   40,29),
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,30),
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.coordinates',        '{"x":207,"y":200}'::jsonb,   40,31),
 (p_world_id,v_event_scene_genesis,v_alley,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,32),
 (p_world_id,v_event_scene_genesis,v_alley,'location','attrs.coordinates',        '{"x":200,"y":240}'::jsonb,   40,33),
 (p_world_id,v_event_scene_genesis,v_cellar,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,34),
 (p_world_id,v_event_scene_genesis,v_cellar,'location','attrs.coordinates',        '{"x":205,"y":205}'::jsonb,   40,35),
 -- Tavern local frame (meters): the three residents where the scene-genesis places them, and the bar
 -- feature along the back wall. Kade's own coordinate rides his arrival event (fa) below.
 --   Mara behind the bar {6,10}; Jonas by it {5,8}; the hooded woman at the corner table {1,1}; the bar {6,9}.
 (p_world_id,v_event_scene_genesis,v_mara,'actor',   'attrs.coordinates',        '{"x":6,"y":10}'::jsonb,      40,36),
 (p_world_id,v_event_scene_genesis,v_jonas,'actor',   'attrs.coordinates',        '{"x":5,"y":8}'::jsonb,       40,37),
 (p_world_id,v_event_scene_genesis,v_hooded_woman,'actor',   'attrs.coordinates',        '{"x":1,"y":1}'::jsonb,       40,38),
 -- the bar: a fixed room feature (FINAL "contains: the bar…"). location_id = the tavern (its scene) so
 -- fn_distance resolves it to the tavern frame; coordinates {6,9} along the back wall — the anchor Kade
 -- walks to (Task 8). size-2, weightless fixture (never relocated); Tier-2 descriptor for the narrator.
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.location_id',        to_jsonb(v_drowned_lantern::text), 40,39),
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.coordinates',        '{"x":6,"y":9}'::jsonb,       40,40),
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.size',              to_jsonb(2),                  40,41),
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.descriptor',        to_jsonb('the bar'::text),    40,42),
 -- §4 ObjectRelocated physics has something to grab: a Container instance (ballast crate) + a heavy
 -- ballast stone inside it. crate = (empty_weight 8 + effective_weight(stone 92)) × 1 = 100 kg; Kade's
 -- max_load is 80, so "grab the crate → encumbered" is REACHABLE (the eager rule flips it on that commit).
 -- The crate RESTS on the tavern floor: like the bar (f1) above, that is attrs.location_id = the tavern
 -- (a location is not a carrier -- `contained_by` is the carry-chain key for "inside a container / held
 -- by an actor", which this crate is not, yet). fn_distance's artifact-scene COALESCE has no other
 -- source of scene (current_scene_id is never written), so an omitted location_id here would silently
 -- resolve to NULL and fn_distance(Kade, crate) would silently read 0 instead of the true ~8.94 m.
 -- The crate is a mundane container (weight_modifier absent → 1), by the hatch {2,9}; the stone lives
 -- inside the crate (attrs.contained_by = the crate). size-2 stone (vol 4) fits max_room 16.
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.max_room',          to_jsonb(16),                 40,43),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.empty_weight',      to_jsonb(8),                  40,44),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.size',              to_jsonb(4),                  40,45),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.location_id',       to_jsonb(v_drowned_lantern::text), 40,46),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.coordinates',        '{"x":2,"y":9}'::jsonb,       40,47),
 (p_world_id,v_event_scene_genesis,v_ballast_stone,'artifact','attrs.weight',            to_jsonb(92),                 40,48),
 (p_world_id,v_event_scene_genesis,v_ballast_stone,'artifact','attrs.size',              to_jsonb(2),                  40,49),
 (p_world_id,v_event_scene_genesis,v_ballast_stone,'artifact','attrs.contained_by',      to_jsonb(v_ballast_crate::text), 40,50);

-- ── Kade's arrival (tick 50) — he steps into the room the scene is set in ─────────────
-- Replay-safe & append-only: one accepted ActorMoved with an ABSOLUTE attrs.location_id set (the
-- sm_project trigger projects it; replay_0a rebuilds it). Tick 50 is this world's max; the live
-- handler mints the next beat after it.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 (v_event_kade_arrival,p_world_id,'ActorMoved',
  'Kade steps into the Drowned Lantern.',50,0,
  'Arrival','accepted',now(),'public','fast_path');
-- Participant: the mover (instigator).
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 (v_event_kade_arrival,v_kade,'actor','instigator');
-- Absolute location set → the Drowned Lantern. The projection trigger places Kade in the room.
-- §3/§4 (Task 7): Kade also arrives WITH a position and a carrying capacity. coordinates {6,1} put him
-- just inside the front door — 8 m from the bar {6,9} ⇒ fn_distance(Kade,bar)=8, CEIL(8/1.4)=6 s (fits
-- tense's 30 s beat: the Task-8 "walk to the bar"). max_load 80 is his static capacity: the ballast crate
-- weighs 100 kg, so grabbing it exceeds max_load → the eager encumbrance rule (§4) can fire in play.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.location_id',
  to_jsonb(v_drowned_lantern::text),50,0),
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.max_load',    to_jsonb(80),           50,2),
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.coordinates', '{"x":6,"y":1}'::jsonb, 50,3);
-- Kade's own honest, minimal perception of stepping in. NOT an authored roster of who is present (that
-- would fake fan-out he never received) — just the move itself, subject-linked to the mover + the room.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 (v_perception_kade_arrival,p_world_id,
  v_kade,v_event_kade_arrival,
  'I stepped into the Drowned Lantern.','direct',50,50);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 (v_perception_kade_arrival,v_kade,p_world_id),
 (v_perception_kade_arrival,v_drowned_lantern,p_world_id);

-- ── §3 naming reach: per-viewer NAME KNOWLEDGE + DESCRIPTOR fallbacks (Defect C) ──────────
-- The live founder-gate leak: canonical names reached the character-mind seats past knowledge paths
-- (the narration named "Jonas" to Kade, who knows him only as "the muscle"; Jonas's wind-up named
-- "Kade"). fn_display_name closes it — known-name (a viewer's own name-knowledge) else descriptor else
-- canonical. Name knowledge is stored as chunk-4's identity substrate: a world_genesis-sourced
-- perception, subject-linked to the named entity, HELD BY the viewer who knows the name (per-viewer, so
-- Kade knowing "Mara" never grants Jonas or the hooded woman that name). Held by ONE viewer ⇒ private to
-- that viewer's calls (fn_visible_perceptions is holder-keyed) — the wall holds by construction.
--
-- Who knows whose name (FINAL-drowned-lantern-souls.md): Kade knows Mara (five winters back). Mara and
-- Jonas know each other as regulars would. Mara knows Kade ONLY privately — as "Reyna's brother", the
-- name he had then, NOT the name he carries now (her secret cluster); NO public name record for Kade
-- exists, so nobody in the room publicly knows his name. The hooded woman knows no one's name (she has
-- a description of a courier, not a name), and no one knows hers — she stays "a hooded figure".
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 (v_event_world_genesis,p_world_id,'world_genesis',
  'the harbor-quarter figures known to each other by name (per-viewer identity substrate)',25,0,
  'Genesis','accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 (v_event_world_genesis,v_kade,'actor','named'),
 (v_event_world_genesis,v_mara,'actor','named'),
 (v_event_world_genesis,v_jonas,'actor','named'),
 (v_event_world_genesis,v_hooded_woman,'actor','named');

-- Name perceptions (content = the name the viewer knows; source = genesis; subject = the named entity).
-- Fixed perception_ids (prefix 2a4e = "name"). Held per-viewer ⇒ each is that viewer's knowledge alone.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 -- Kade knows Mara by name (five years ago).
 (v_name_perception_kade_knows_mara,p_world_id,
  v_kade,v_event_world_genesis,'Mara','told',25,25),
 -- Mara knows Jonas by name (regulars).
 (v_name_perception_mara_knows_jonas,p_world_id,
  v_mara,v_event_world_genesis,'Jonas','told',25,25),
 -- Jonas knows Mara by name (regulars).
 (v_name_perception_jonas_knows_mara,p_world_id,
  v_jonas,v_event_world_genesis,'Mara','told',25,25),
 -- Mara PRIVATELY knows Kade — as "Reyna's brother", the name he had then, not the one he carries now.
 -- Held by Mara ALONE ⇒ only her own calls resolve it; the wall holds (part of her secret cluster).
 (v_name_perception_mara_knows_kade,p_world_id,
  v_mara,v_event_world_genesis,'Reyna''s brother','told',25,25);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 (v_name_perception_kade_knows_mara,v_mara,p_world_id),
 (v_name_perception_mara_knows_jonas,v_jonas,p_world_id),
 (v_name_perception_jonas_knows_mara,v_mara,p_world_id),
 (v_name_perception_mara_knows_kade,v_kade,p_world_id);

-- DESCRIPTOR fallbacks (Tier-2 attrs.descriptor) — what a viewer with no name-knowledge sees. Via
-- state_mutation (sm_project → actor_state), replay-safe (each (entity,path) written once). The three
-- residents under the scene-genesis event f9 (tick 40); Kade under his arrival fa (tick 50).
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 (p_world_id,v_event_scene_genesis,v_mara,'actor','attrs.descriptor', to_jsonb('the keeper'::text),                    40,23),
 (p_world_id,v_event_scene_genesis,v_jonas,'actor','attrs.descriptor', to_jsonb('the muscle by the bar'::text),         40,24),
 (p_world_id,v_event_scene_genesis,v_hooded_woman,'actor','attrs.descriptor', to_jsonb('a hooded figure'::text),               40,25),
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.descriptor', to_jsonb('a young stranger, dark-haired'::text), 50,1);

-- ── THE WAY OUT OF TOWN (SPEC-030, founder-named 2026-08-08) — the first JOURNEY in this world ──
-- Everything in the quarter used to sit inside one beat: the tavern's tension is 'tense' → a 30 s
-- budget, and its farthest neighbour (the alley) is 40 m ⇒ 29 s. Every destination fit, so no move
-- could ever go over budget and the Journey shipped in #32 was unreachable by any client.
--
-- Two things were needed, and neither is engine work:
--
--  1. A DESTINATION FAR ENOUGH. The Harbormaster's Office sits off Dock Street at {627,200} — 420 m
--     from the road at {207,200} ⇒ CEIL(420/1.4) = 300 s of walking (the same 1.4 m/s the rest of
--     this seed is tuned against). That is five times the origin's budget, so the walk cannot be
--     swallowed by one beat: it becomes a journey with legs the world can interrupt.
--  2. A FINITE BUDGET TO EXCEED. Dock Street carried no tension at all, and a missing tension reads
--     as 'none' ⇒ an INFINITE budget (tensionBudgetSeconds), which means no move from the road could
--     ever be over budget however far it went. It is now 'normal' ⇒ 60 s: an open harbour road is
--     not tense, but it is not timeless either.
--
-- So the founder's worked example plays: step out the front door onto Dock Street (5 s, instant),
-- then walk for the office (300 s vs a 60 s budget) → journey → interruption → restate → arrival.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- Dock Street gains a finite beat budget (see note 2 above).
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.tension',            to_jsonb('normal'::text),      40,51),
 -- The Harbormaster's Office: a child of the quarter, up the road from the docks.
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,52),
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.coordinates',        '{"x":627,"y":200}'::jsonb,    40,53),
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.tension',            to_jsonb('normal'::text),      40,54),
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.description',        to_jsonb('A ledger-room above the wharf; tide charts, a brass scale, and the harbourmaster''s long window over the water.'::text), 40,55),
 -- Office Door: OPEN and unlocked, dock street↔office. The way is clear; the DISTANCE is the obstacle.
 (p_world_id,v_event_scene_genesis,v_office_door,'artifact','attrs.open',               to_jsonb(true),                40,56),
 (p_world_id,v_event_scene_genesis,v_office_door,'artifact','attrs.locked',             to_jsonb(false),               40,57),
 (p_world_id,v_event_scene_genesis,v_office_door,'artifact','attrs.connects',           jsonb_build_array(v_dock_street,v_harbormaster_office), 40,58),
 -- The second hooded figure: same table, same descriptor as the first, so Kade cannot tell them apart
 -- and "the hooded figure" names both. This is what makes UNRESOLVED reachable in play.
 (p_world_id,v_event_scene_genesis,v_hooded_companion,'actor',   'attrs.location_id',        to_jsonb(v_drowned_lantern::text), 40,59),
 -- At the BAR, not the corner table: the two hooded figures wear the same descriptor, so the only way
 -- a player can tell them apart is where each one is standing. Standing them beside DIFFERENT things
 -- is what gives fn_display_names_distinct something true to say ("by the bar" vs "by the ballast
 -- crate"); put them side by side and the honest answer becomes "you cannot tell", which is a real
 -- outcome but a poor one to ship as the only one the seeded world can produce.
 (p_world_id,v_event_scene_genesis,v_hooded_companion,'actor',   'attrs.coordinates',        '{"x":5,"y":9}'::jsonb,        40,60),
 (p_world_id,v_event_scene_genesis,v_hooded_companion,'actor',   'attrs.descriptor',         to_jsonb('a hooded figure'::text), 40,61);

-- A descriptor for the ballast crate. It had none, so fn_display_name fell through to the canonical
-- registry name and a disambiguated label read "a hooded figure by the Ballast Crate" — a database
-- row wearing a sentence. Every OTHER thing a viewer can be told about carries a descriptor (§ the
-- DESCRIPTOR fallbacks block above); this closes the gap now that anchors are player-visible text.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.descriptor', to_jsonb('the ballast crate'::text), 40,62);

-- SPEC-028 directory entry. THIS is the row that retires the 'Player' naming convention: the player
-- here is KADE, and because ResolveViewer could only look for an actor literally named 'Player',
-- every non-debug request against the one world anyone actually plays used to 500 at the door.
-- Theme is the tavern's own: lamplight gold, nocturne, filigree.
-- The tagline is AUTHORED FICTION and the seed is where it lives (GA-2): the service never composes
-- one, so the only way a world has a line is that somebody wrote it here. Founder-approved verbatim
-- 2026-08-09; do not reword it in passing.
--
-- DO UPDATE on the tagline specifically, and the rest still DO NOTHING. The seeds are re-run against
-- a live shared database, and `ON CONFLICT DO NOTHING` is exactly how the SPEC-031 tuning nearly
-- landed green while changing nothing in the only world anyone plays. The other columns keep DO
-- NOTHING because they are identity, not content: re-seeding must never reset a world's player or
-- rename it, but it MUST converge the fiction it owns.
INSERT INTO world (world_id, display_name, tagline, theme, player_entity_id) VALUES
 (p_world_id, 'The Drowned Lantern',
  'A harbor town where everyone is owed something, and the tide keeps the ledger.',
  '{"schema_version":"world_theme/1","accent":"#c9a227","mood":"nocturne","ornament":"filigree"}'::jsonb,
  v_kade)
ON CONFLICT (world_id) DO UPDATE SET tagline = EXCLUDED.tagline;

UPDATE world
   SET player_entity_id = v_kade,
       template_key = 'drowned_lantern'
 WHERE world_id = p_world_id;

END;
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_instantiate_drowned_lantern(uuid, jsonb);

CREATE OR REPLACE FUNCTION public.fn_world_directory() RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
    'schema_version', 'world_directory/2',
    'worlds', COALESCE((
      SELECT json_agg(json_build_object(
               'id',            w.world_id,
               'display_name',  w.display_name,
               'tagline',       w.tagline,
               'theme',         w.theme,
               'playable',      w.player_entity_id IS NOT NULL,
               'cover_image',   fn_image_ref(w.world_id, 'world', w.world_id),
               'last_place_label', (
                  SELECT fn_display_name(w.world_id, w.player_entity_id,
                                         (a.attrs->>'location_id')::uuid)
                    FROM actor_state a
                   WHERE a.world_id = w.world_id
                     AND a.entity_id = w.player_entity_id
                     AND a.attrs->>'location_id' IS NOT NULL
               )
             ) ORDER BY w.display_name, w.world_id)
        FROM world w), '[]'::json)
  );
$$;

ALTER TABLE world
  DROP COLUMN IF EXISTS archived_at,
  DROP COLUMN IF EXISTS template_key;

