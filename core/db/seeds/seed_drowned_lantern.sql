-- =====================================================================================
-- seed_drowned_lantern.sql — the Drowned Lantern scene (Station E, chunk-5.5).
-- Loaded AFTER seed_mara_0A.sql by `make seed`. APPEND-ONLY: adds rows to the existing
-- world 11111111-… ; never modifies or deletes seed_mara_0A's rows.
--
-- Content canon (founder-corrected 2026-07-24):
--   docs/superpowers/specs/chunk-5.5-final/FINAL-drowned-lantern-souls.md
--
-- The shape that matters:
--   * approved souls become rows: personality cores grounded in backstory canon events;
--   * SECRETS are private perception_records WITH subject links — NEVER core traits (cores
--     ride SHARED cognition prompts; a secret in a core would leak by construction);
--   * knows_kade_as lives INSIDE Mara's private record, not her core;
--   * every private record references a grounding event and its subject links;
--   * the first playable room holds a real Tier-1 locked Portal (the cellar hatch).
--
-- Determinism: fixed uuids, no random(). State that must survive replay (tavern tension,
-- the portals, the note, the key) is written as state_mutation rows under one scene event
-- (the sm_project trigger projects them; replay_0A rebuilds them identically). NO actor_state
-- is written for world 1111 (the golden projection, test 80, freezes the 8-actor set; the
-- hooded woman is deliberately un-placed — presence is supplied to the lookups at play time).
--
-- Ticks: backstory events at 50–57 and the scene-set event at 58 — all before E1@100, and clear
-- of every world-1111 slot the fresh-world tests reserve (smoke tests at ticks 5/7/10/100; scenario
-- setups at 210+) under uq_ce_accepted_order(world_id,in_world_tick,beat_seq). All below the seed's
-- max tick (201), so no max()-based minted tick shifts.
-- =====================================================================================
BEGIN;

-- Own idempotence guard: refuse a double-load. `make reset` is the clean re-run path.
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM entity_registry
             WHERE entity_id='ffffffff-ffff-ffff-ffff-ffffffffffff') THEN
    RAISE EXCEPTION 'seed_drowned_lantern already applied (hooded woman exists) — run `make reset` for a clean load';
  END IF;
END $$;

-- ── Registry (+9): the hooded woman + the room''s places and objects ──────────────────
-- The named cast (Kade aaaa / Mara bbbb / Jonas cccc / Tavern dddd) already exists.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('ffffffff-ffff-ffff-ffff-ffffffffffff','11111111-1111-1111-1111-111111111111','actor',   'Hooded Woman'),
 ('d1000000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111','location','Dock Street'),
 ('d1000000-0000-0000-0000-0000000000a2','11111111-1111-1111-1111-111111111111','location','Alley'),
 ('d1000000-0000-0000-0000-0000000000a3','11111111-1111-1111-1111-111111111111','location','Cellar'),
 ('d1000000-0000-0000-0000-0000000000b1','11111111-1111-1111-1111-111111111111','artifact','Sealed Note (gray wax)'),
 ('d1000000-0000-0000-0000-0000000000c1','11111111-1111-1111-1111-111111111111','artifact','Front Door'),
 ('d1000000-0000-0000-0000-0000000000c2','11111111-1111-1111-1111-111111111111','artifact','Back Door'),
 ('d1000000-0000-0000-0000-0000000000c3','11111111-1111-1111-1111-111111111111','artifact','Cellar Hatch'),
 ('d1000000-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111','artifact','Cellar Key');

-- ── Backstory canon events (ticks 50–57, before E1@100) + one scene-set event (tick 58) ──
-- event_type='AttributeChanged' (backstory grounds who they are); origin='fast_path' (matches
-- seed_mara_0A''s authored history and the recognized gated origin set). M-E4 / J-E3 / H-E1 are
-- PRIVATE — each is the grounding source of exactly one NPC''s private perception below.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 ('e0000000-0000-0000-0000-0000000000f1','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'M-E1: grew up behind this bar; her father taught her a keeper who reacts has already lost',50,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f2','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'M-E2: the harbormaster''s predecessor shook the tavern for protection money; the watch shrugged; her father died that winter',51,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f3','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'M-E3: a dock brawl left Jonas half-dead outside her door; she stitched him up and gave him work',52,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f4','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'M-E4 (private): she hid Reyna''s family in the cellar nine days; Reyna''s teenage brother ran the messages that got them out',53,0,
  'Backstory','accepted',now(),'private','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f5','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'J-E1: beaten near to death over a fixed fight and left in the alley; Mara took him in',54,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f6','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'J-E2: a prizefighter until he killed a man in the ring with one unlucky blow; never fought clean for money again',55,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f7','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'J-E3 (private): twice he watched Mara go pale at a harbor face and learned to stand closer instead of asking',56,0,
  'Backstory','accepted',now(),'private','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f8','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'H-E1 (private): took the paymaster''s contract in a counting-house above the silk quay, three days ago',57,0,
  'Backstory','accepted',now(),'private','fast_path'),
 ('e0000000-0000-0000-0000-0000000000f9','11111111-1111-1111-1111-111111111111','AttributeChanged',
  'the Drowned Lantern is set: tension, the doors, the hatch, the note, the key',58,0,
  'Scene','accepted',now(),'public','fast_path');

-- Participants (brief: the NPC + any named co-subject). subject ≠ about-ness (perception_subject
-- carries the precise about-ness, ADR-035) — these are the event''s people.
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-0000000000f1','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f2','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f3','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f4','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f4','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','co_subject'),
 ('e0000000-0000-0000-0000-0000000000f5','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f5','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','co_subject'),
 ('e0000000-0000-0000-0000-0000000000f6','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f7','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f7','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','co_subject'),
 ('e0000000-0000-0000-0000-0000000000f8','ffffffff-ffff-ffff-ffff-ffffffffffff','actor','subject'),
 ('e0000000-0000-0000-0000-0000000000f9','dddddddd-dddd-dddd-dddd-dddddddddddd','location','setting');

-- ── personality_core — WHO THEY ARE IN THE ROOM. No secret ever lives here. ───────────
-- traits jsonb: real traits are objects {value, manner}; schema_version + speech_manner are
-- strings. Kade gets NO core (premise, not a mind). Malleability per GATE-A.
INSERT INTO personality_core (world_id, actor_id, traits, malleability) VALUES
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  '{"schema_version":"traits/1",
    "guarded":{"value":0.8,"manner":"answers questions with questions; volunteers nothing"},
    "dry_witted":{"value":0.7,"manner":"deflects with humor before she deflects with silence"},
    "loyal_to_jonas":{"value":0.9,"manner":"treats Jonas as family; will not see him harmed"},
    "distrusts_authority":{"value":0.85,"manner":"the harbormaster''s men drink free and learn nothing"},
    "steady_under_pressure":{"value":0.8,"manner":"the last in the room to raise her voice"},
    "speech_manner":"short sentences; harbor slang; calls strangers sailor regardless of trade; never says a name she was not given"}'::jsonb,
  0.25),
 ('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc',
  '{"schema_version":"traits/1",
    "protective_of_mara":{"value":0.9,"manner":"reads every stranger as a threat to her first, himself second"},
    "slow_to_speak":{"value":0.7,"manner":"acts before he explains; three words where others use ten"},
    "brawler_not_killer":{"value":0.8,"manner":"ends fights; does not start them; hates blades"},
    "debt_of_gratitude":{"value":0.85,"manner":"the tavern is the only place that ever took him back"},
    "speech_manner":"monosyllables; states facts not opinions; uses names only for Mara"}'::jsonb,
  0.45),
 ('11111111-1111-1111-1111-111111111111','ffffffff-ffff-ffff-ffff-ffffffffffff',
  '{"schema_version":"traits/1",
    "watchful":{"value":0.8,"manner":"tracks the door and every hand near a purse"},
    "unhurried":{"value":0.7,"manner":"never the first to move, never the last to leave"},
    "clean_coin":{"value":0.6,"manner":"pays in coin too clean for this district"},
    "speech_manner":"says little; asks less; watches much"}'::jsonb,
  0.6);

-- ── trait_provenance — every trait traces to a backstory event (D-11 for character) ───
INSERT INTO trait_provenance (world_id, actor_id, trait_key, event_id) VALUES
 -- Mara
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','dry_witted',           'e0000000-0000-0000-0000-0000000000f1'),
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','steady_under_pressure','e0000000-0000-0000-0000-0000000000f1'),
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','guarded',              'e0000000-0000-0000-0000-0000000000f2'),
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','distrusts_authority',  'e0000000-0000-0000-0000-0000000000f2'),
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','loyal_to_jonas',       'e0000000-0000-0000-0000-0000000000f3'),
 -- Jonas
 ('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','protective_of_mara',   'e0000000-0000-0000-0000-0000000000f5'),
 ('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','debt_of_gratitude',    'e0000000-0000-0000-0000-0000000000f5'),
 ('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','slow_to_speak',        'e0000000-0000-0000-0000-0000000000f6'),
 ('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc','brawler_not_killer',   'e0000000-0000-0000-0000-0000000000f6'),
 -- Hooded woman (one event grounds her thin core)
 ('11111111-1111-1111-1111-111111111111','ffffffff-ffff-ffff-ffff-ffffffffffff','watchful',   'e0000000-0000-0000-0000-0000000000f8'),
 ('11111111-1111-1111-1111-111111111111','ffffffff-ffff-ffff-ffff-ffffffffffff','unhurried',  'e0000000-0000-0000-0000-0000000000f8'),
 ('11111111-1111-1111-1111-111111111111','ffffffff-ffff-ffff-ffff-ffffffffffff','clean_coin', 'e0000000-0000-0000-0000-0000000000f8');

-- ── Private knowledge — perception_record WITH subject links (the whole point) ────────
-- Only the holder holds a perception of each private event → private to the lookups. Fixed
-- perception_ids so the subject links are unambiguous. source_event_id is NOT NULL (grounded).
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 -- Mara''s secret: recognition + the life-debt + how she knows him ("Reyna''s brother").
 ('d15ec000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','e0000000-0000-0000-0000-0000000000f4',
  'Kade is Reyna''s brother — the boy who ran messages while I hid Reyna''s family in the cellar nine days, five winters back. I owe that family a life-debt I have never said aloud. To him, and to this room, I am a stranger; if the wrong people learn I know him, the debt gets us both killed.',
  'direct',53,53),
 -- Jonas knows OF a secret without knowing IT: a debt she never explains. (No "Reyna", no "ledger".)
 ('d15ec000-0000-0000-0000-0000000000b1','11111111-1111-1111-1111-111111111111',
  'cccccccc-cccc-cccc-cccc-cccccccccccc','e0000000-0000-0000-0000-0000000000f7',
  'Mara keeps a knife under the till and a debt she never explains. Twice in four years I have watched her go pale at a face off the harbor. I have learned to stand closer and not to ask. I do not know what it is.',
  'inference',56,56),
 -- The hooded woman''s contract: a description and a purse, and one word of doubt — "Yet."
 ('d15ec000-0000-0000-0000-0000000000c1','11111111-1111-1111-1111-111111111111',
  'ffffffff-ffff-ffff-ffff-ffffffffffff','e0000000-0000-0000-0000-0000000000f8',
  'The paymaster''s coin bought a description: a courier, young, dark-haired, moves like a dock rat — and a purse for whoever confirms him. The one by the door could be him. I am not sure. Yet.',
  'told',57,57);

-- about-ness (RULINGS §6): Mara''s secret → Kade AND Mara; Jonas → Mara; hooded → Kade.
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('d15ec000-0000-0000-0000-0000000000a1','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111'),
 ('d15ec000-0000-0000-0000-0000000000a1','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111'),
 ('d15ec000-0000-0000-0000-0000000000b1','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111'),
 ('d15ec000-0000-0000-0000-0000000000c1','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111');

-- ── Scene state via state_mutation (event f9) — projects through sm_project; replay-safe ──
-- Single-key absolute sets under attrs (Rider B). Tier-1 keys: open, locked, connects, size,
-- weight, tension (see core/api/tier1.go). carried_by / held_by are Tier-2 (carry-state is
-- deferred — no actor_state is touched, so the golden 8-actor set is unchanged). connects is the
-- Portal''s [room, room] pair. The cellar hatch is closed and LOCKED — the first Tier-1 lock in play.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- the room reads calm; two of the four people in it are pretending → tension 'tense'
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','dddddddd-dddd-dddd-dddd-dddddddddddd','location','attrs.tension',              to_jsonb('tense'::text),                                                                         58,0),
 -- art_note: sealed, near-weightless, carried by Kade. Contents deliberately UNAUTHORED (Tier-2 flavor only).
 -- NOTE: chunk-4's seed_mara_0A.sql already registers an older, separate "Sealed Note" entity
 -- (a4000000-0000-0000-0000-0000000000a1) that co-exists in this world. This art_note
 -- (d1000000-0000-0000-0000-0000000000b1) is the Drowned Lantern's canonical note for play —
 -- do not conflate the two when reading state or writing new tests.
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000b1','artifact','attrs.size',                 to_jsonb(1),                                                                                    58,1),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000b1','artifact','attrs.weight',               to_jsonb(0),                                                                                    58,2),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000b1','artifact','attrs.sealed_with_gray_wax', to_jsonb(true),                                                                                 58,3),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000b1','artifact','attrs.carried_by',           to_jsonb('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::text),                                          58,4),
 -- front door: OPEN, unlocked, tavern↔dock street
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c1','artifact','attrs.open',                 to_jsonb(true),                                                                                 58,5),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c1','artifact','attrs.locked',               to_jsonb(false),                                                                                58,6),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c1','artifact','attrs.connects',             jsonb_build_array('dddddddd-dddd-dddd-dddd-dddddddddddd','d1000000-0000-0000-0000-0000000000a1'), 58,7),
 -- back door: closed, unlocked, tavern↔alley
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c2','artifact','attrs.open',                 to_jsonb(false),                                                                                58,8),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c2','artifact','attrs.locked',               to_jsonb(false),                                                                                58,9),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c2','artifact','attrs.connects',             jsonb_build_array('dddddddd-dddd-dddd-dddd-dddddddddddd','d1000000-0000-0000-0000-0000000000a2'), 58,10),
 -- cellar hatch: closed and LOCKED (Tier-1), tavern↔cellar. Mara holds the key; the cellar is where M-E4 happened.
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c3','artifact','attrs.open',                 to_jsonb(false),                                                                                58,11),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c3','artifact','attrs.locked',               to_jsonb(true),                                                                                 58,12),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000c3','artifact','attrs.connects',             jsonb_build_array('dddddddd-dddd-dddd-dddd-dddddddddddd','d1000000-0000-0000-0000-0000000000a3'), 58,13),
 -- the cellar key, held by Mara
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000d1','artifact','attrs.size',                 to_jsonb(1),                                                                                    58,14),
 ('11111111-1111-1111-1111-111111111111','e0000000-0000-0000-0000-0000000000f9','d1000000-0000-0000-0000-0000000000d1','artifact','attrs.held_by',              to_jsonb('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::text),                                          58,15);

COMMIT;
