-- =====================================================================================
-- seed_drowned_lantern.sql — the Drowned Lantern scene (Station E, chunk-5.5).
-- Loaded AFTER seed_mara_0A.sql by `make seed`, but writes a DISJOINT, DEDICATED world:
--   22222222-2222-2222-2222-222222222222 (the PLAY world).
--
-- Founder Option B (2026-07-26): the play scene gets its OWN world so nothing bleeds in from
-- the legacy test-fixture world 11111111-… (noise actors O2/O5, the 0A noise-loop scatter of
-- Mara/Jonas, a location literally named 'Tavern', ledger lore in memory). The fixture world
-- 1111 reverts to pristine 0A; play lives here, in a world this seed owns end to end.
--
-- Content canon (founder-corrected; transcribe faithfully, do NOT re-paraphrase):
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
-- Determinism / replay: fixed uuids, no random(). ALL four actors are PLACED in the Drowned
-- Lantern by replay-safe state — absolute attrs.location_id state_mutation writes under accepted
-- canon events (the sm_project trigger projects them on insert; replay_0a rebuilds them identically).
-- Each (entity, attribute_path) is written EXACTLY ONCE, so live-insert order and replay order agree
-- (last-writer-wins is a no-op). We own this world's tick space; backstory sits at 30–37, the scene
-- genesis at 40, Kade's arrival at 50 — all well under the Go tests' 50000 floor.
--
-- Tick-space note: uq_ce_accepted_order is (world_id,in_world_tick,beat_seq) WHERE status='accepted'
-- (schema.sql), i.e. WORLD-SCOPED — so this world's tick space is ours. The ONE caveat: pgTAP
-- test 85 (85_causal_acyclicity_test.sql) uses world 2222… as a transient scratch world inside a
-- BEGIN/ROLLBACK at ticks 10–22 (beat_seq 0); demo_cycle_0B.sql (not in the *_test.sql suite) uses
-- ticks 1–2. We deliberately avoid 1–2 and 10–22 so a test-85 run never collides with this seed's
-- standing rows.
-- =====================================================================================
BEGIN;

-- Own idempotence guard: refuse a double-load. `make reset` is the clean re-run path.
-- Guard on the PLAY world's tavern (the Drowned Lantern) — it exists iff this seed already ran.
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM entity_registry
             WHERE entity_id='210c0000-0000-0000-0000-0000000000d1') THEN
    RAISE EXCEPTION 'seed_drowned_lantern already applied (the Drowned Lantern exists) — run `make reset` for a clean load';
  END IF;
END $$;

-- Physics defaults for the play world (contracts §2: exactly walk 1.4 + encumbered -100 on walk).
SELECT seed_world_defaults('22222222-2222-2222-2222-222222222222');

-- ── Registry: 4 actors + 4 locations (REAL names) + 5 artifacts ───────────────────────
-- All-new fixed uuids (entity_registry PK is global). Kade is 'Kade' now — a real name, not the
-- fixture world's 'Player'. The tavern is 'The Drowned Lantern', not 'Tavern'.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('2ac70000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222','actor',   'Kade'),
 ('2ac70000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222','actor',   'Mara'),
 ('2ac70000-0000-0000-0000-0000000000a3','22222222-2222-2222-2222-222222222222','actor',   'Jonas'),
 ('2ac70000-0000-0000-0000-0000000000a4','22222222-2222-2222-2222-222222222222','actor',   'Hooded Woman'),
 ('210c0000-0000-0000-0000-0000000000d1','22222222-2222-2222-2222-222222222222','location','The Drowned Lantern'),
 ('210c0000-0000-0000-0000-0000000000d2','22222222-2222-2222-2222-222222222222','location','Dock Street'),
 ('210c0000-0000-0000-0000-0000000000d3','22222222-2222-2222-2222-222222222222','location','Alley'),
 ('210c0000-0000-0000-0000-0000000000d4','22222222-2222-2222-2222-222222222222','location','Cellar'),
 ('2a7f0000-0000-0000-0000-0000000000b1','22222222-2222-2222-2222-222222222222','artifact','Sealed Note (gray wax)'),
 ('2a7f0000-0000-0000-0000-0000000000c1','22222222-2222-2222-2222-222222222222','artifact','Front Door'),
 ('2a7f0000-0000-0000-0000-0000000000c2','22222222-2222-2222-2222-222222222222','artifact','Back Door'),
 ('2a7f0000-0000-0000-0000-0000000000c3','22222222-2222-2222-2222-222222222222','artifact','Cellar Hatch'),
 ('2a7f0000-0000-0000-0000-0000000000d1','22222222-2222-2222-2222-222222222222','artifact','Cellar Key');

-- ── Backstory canon events (ticks 30–37) + one scene-genesis event (tick 40) ──────────
-- event_type='AttributeChanged' (backstory grounds who they are); origin='fast_path'. M-E4 / J-E3 /
-- H-E1 are PRIVATE — each grounds exactly one NPC's private perception below. The scene-genesis event
-- (f9) is public and carries the room state AND places the three residents (Mara behind the bar,
-- Jonas by the bar, the hooded woman at the corner table) via absolute location writes.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 ('2e000000-0000-0000-0000-0000000000f1','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'M-E1: grew up behind this bar; her father taught her a keeper who reacts has already lost',30,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f2','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'M-E2: the harbormaster''s predecessor shook the tavern for protection money; the watch shrugged; her father died that winter',31,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f3','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'M-E3: a dock brawl left Jonas half-dead outside her door; she stitched him up and gave him work',32,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f4','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'M-E4 (private): she hid Reyna''s family in the cellar nine days; Reyna''s teenage brother ran the messages that got them out',33,0,
  'Backstory','accepted',now(),'private','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f5','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'J-E1: beaten near to death over a fixed fight and left in the alley; Mara took him in',34,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f6','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'J-E2: a prizefighter until he killed a man in the ring with one unlucky blow; never fought clean for money again',35,0,
  'Backstory','accepted',now(),'public','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f7','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'J-E3 (private): twice he watched Mara go pale at a harbor face and learned to stand closer instead of asking',36,0,
  'Backstory','accepted',now(),'private','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f8','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'H-E1 (private): took the paymaster''s contract in a counting-house above the silk quay, three days ago',37,0,
  'Backstory','accepted',now(),'private','fast_path'),
 ('2e000000-0000-0000-0000-0000000000f9','22222222-2222-2222-2222-222222222222','AttributeChanged',
  'the Drowned Lantern is set: Mara behind the bar, Jonas by it, a hooded woman at the corner table; tension, the doors, the hatch, the note, the key',40,0,
  'Scene','accepted',now(),'public','fast_path');

-- Participants (brief: the NPC + any named co-subject). subject ≠ about-ness (perception_subject
-- carries the precise about-ness, ADR-035) — these are the event''s people. The scene-genesis event
-- names the room (setting) and the three residents it places.
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('2e000000-0000-0000-0000-0000000000f1','2ac70000-0000-0000-0000-0000000000a2','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f2','2ac70000-0000-0000-0000-0000000000a2','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f3','2ac70000-0000-0000-0000-0000000000a2','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f4','2ac70000-0000-0000-0000-0000000000a2','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f4','2ac70000-0000-0000-0000-0000000000a1','actor','co_subject'),
 ('2e000000-0000-0000-0000-0000000000f5','2ac70000-0000-0000-0000-0000000000a3','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f5','2ac70000-0000-0000-0000-0000000000a2','actor','co_subject'),
 ('2e000000-0000-0000-0000-0000000000f6','2ac70000-0000-0000-0000-0000000000a3','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f7','2ac70000-0000-0000-0000-0000000000a3','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f7','2ac70000-0000-0000-0000-0000000000a2','actor','co_subject'),
 ('2e000000-0000-0000-0000-0000000000f8','2ac70000-0000-0000-0000-0000000000a4','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d1','location','setting'),
 ('2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a2','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a3','actor','subject'),
 ('2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a4','actor','subject');

-- ── personality_core — WHO THEY ARE IN THE ROOM. No secret ever lives here. ───────────
-- traits jsonb: real traits are objects {value, manner}; schema_version + speech_manner are
-- strings. Kade gets NO core (premise, not a mind). Malleability per FINAL (Mara 0.25 / Jonas 0.45 /
-- hooded 0.6).
INSERT INTO personality_core (world_id, actor_id, traits, malleability) VALUES
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a2',
  '{"schema_version":"traits/1",
    "guarded":{"value":0.8,"manner":"answers questions with questions; volunteers nothing"},
    "dry_witted":{"value":0.7,"manner":"deflects with humor before she deflects with silence"},
    "loyal_to_jonas":{"value":0.9,"manner":"treats Jonas as family; will not see him harmed"},
    "distrusts_authority":{"value":0.85,"manner":"the harbormaster''s men drink free and learn nothing"},
    "steady_under_pressure":{"value":0.8,"manner":"the last in the room to raise her voice"},
    "speech_manner":"short sentences; harbor slang; calls strangers sailor regardless of trade; never says a name she was not given"}'::jsonb,
  0.25),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a3',
  '{"schema_version":"traits/1",
    "protective_of_mara":{"value":0.9,"manner":"reads every stranger as a threat to her first, himself second"},
    "slow_to_speak":{"value":0.7,"manner":"acts before he explains; three words where others use ten"},
    "brawler_not_killer":{"value":0.8,"manner":"ends fights; does not start them; hates blades"},
    "debt_of_gratitude":{"value":0.85,"manner":"the tavern is the only place that ever took him back"},
    "speech_manner":"monosyllables; states facts not opinions; uses names only for Mara"}'::jsonb,
  0.45),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a4',
  '{"schema_version":"traits/1",
    "watchful":{"value":0.8,"manner":"tracks the door and every hand near a purse"},
    "unhurried":{"value":0.7,"manner":"never the first to move, never the last to leave"},
    "clean_coin":{"value":0.6,"manner":"pays in coin too clean for this district"},
    "speech_manner":"says little; asks less; watches much"}'::jsonb,
  0.6);

-- ── trait_provenance — every trait traces to a backstory event (D-11 for character) ───
INSERT INTO trait_provenance (world_id, actor_id, trait_key, event_id) VALUES
 -- Mara
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a2','dry_witted',           '2e000000-0000-0000-0000-0000000000f1'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a2','steady_under_pressure','2e000000-0000-0000-0000-0000000000f1'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a2','guarded',              '2e000000-0000-0000-0000-0000000000f2'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a2','distrusts_authority',  '2e000000-0000-0000-0000-0000000000f2'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a2','loyal_to_jonas',       '2e000000-0000-0000-0000-0000000000f3'),
 -- Jonas
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a3','protective_of_mara',   '2e000000-0000-0000-0000-0000000000f5'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a3','debt_of_gratitude',    '2e000000-0000-0000-0000-0000000000f5'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a3','slow_to_speak',        '2e000000-0000-0000-0000-0000000000f6'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a3','brawler_not_killer',   '2e000000-0000-0000-0000-0000000000f6'),
 -- Hooded woman (one event grounds her thin core)
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a4','watchful',   '2e000000-0000-0000-0000-0000000000f8'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a4','unhurried',  '2e000000-0000-0000-0000-0000000000f8'),
 ('22222222-2222-2222-2222-222222222222','2ac70000-0000-0000-0000-0000000000a4','clean_coin', '2e000000-0000-0000-0000-0000000000f8');

-- ── Private knowledge — perception_record WITH subject links (the whole point) ────────
-- Only the holder holds a perception of each private event → private to the lookups. Fixed
-- perception_ids so the subject links are unambiguous. source_event_id is NOT NULL (grounded).
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 -- Mara''s secret: recognition + the life-debt + how she knows him ("Reyna''s brother").
 ('2ce50000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a2','2e000000-0000-0000-0000-0000000000f4',
  'Kade is Reyna''s brother — the boy who ran messages while I hid Reyna''s family in the cellar nine days, five winters back. I owe that family a life-debt I have never said aloud. To him, and to this room, I am a stranger; if the wrong people learn I know him, the debt gets us both killed.',
  'direct',33,33),
 -- Jonas knows OF a secret without knowing IT: a debt she never explains. (No "Reyna", no "ledger".)
 ('2ce50000-0000-0000-0000-0000000000b1','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a3','2e000000-0000-0000-0000-0000000000f7',
  'Mara keeps a knife under the till and a debt she never explains. Twice in four years I have watched her go pale at a face off the harbor. I have learned to stand closer and not to ask. I do not know what it is.',
  'inference',36,36),
 -- The hooded woman''s contract: a description and a purse, and one word of doubt — "Yet." (founder-trimmed;
 -- NO characterization of the note''s contents.)
 ('2ce50000-0000-0000-0000-0000000000c1','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a4','2e000000-0000-0000-0000-0000000000f8',
  'The paymaster''s coin bought a description: a courier, young, dark-haired, moves like a dock rat — and a purse for whoever confirms him. The one by the door could be him. I am not sure. Yet.',
  'told',37,37);

-- about-ness (RULINGS §6): Mara''s secret → Kade AND Mara; Jonas → Mara; hooded → Kade.
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('2ce50000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222'),
 ('2ce50000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222'),
 ('2ce50000-0000-0000-0000-0000000000b1','2ac70000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222'),
 ('2ce50000-0000-0000-0000-0000000000c1','2ac70000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222');

-- ── Scene state via state_mutation (event f9) — projects through sm_project; replay-safe ──
-- Single-key absolute sets under attrs (Rider B). Tier-1 keys: open, locked, connects, size,
-- weight, tension (see core/api/tier1.go). carried_by / held_by are Tier-2. connects is the
-- Portal''s [room, room] pair. The cellar hatch is closed and LOCKED — the first Tier-1 lock in play.
-- The three residents are PLACED here (absolute attrs.location_id → the Drowned Lantern); Kade arrives
-- separately below. Each (entity, attribute_path) is written exactly once → replay-order-independent.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- residents in the room (Mara behind the bar, Jonas by it, the hooded woman at the corner table)
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a2','actor',   'attrs.location_id', to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text), 40,0),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a3','actor',   'attrs.location_id', to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text), 40,1),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a4','actor',   'attrs.location_id', to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text), 40,2),
 -- the room reads calm; two of the four people in it are pretending → tension 'tense'
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d1','location','attrs.tension',     to_jsonb('tense'::text),                                                                            40,3),
 -- art_note: sealed, near-weightless, carried by Kade. Contents deliberately UNAUTHORED (Tier-2 flavor only).
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000b1','artifact','attrs.size',                 to_jsonb(1),                                                                                    40,4),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000b1','artifact','attrs.weight',               to_jsonb(0),                                                                                    40,5),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000b1','artifact','attrs.sealed_with_gray_wax', to_jsonb(true),                                                                                 40,6),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000b1','artifact','attrs.carried_by',           to_jsonb('2ac70000-0000-0000-0000-0000000000a1'::text),                                          40,7),
 -- front door: OPEN, unlocked, tavern↔dock street
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c1','artifact','attrs.open',                 to_jsonb(true),                                                                                 40,8),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c1','artifact','attrs.locked',               to_jsonb(false),                                                                                40,9),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c1','artifact','attrs.connects',             jsonb_build_array('210c0000-0000-0000-0000-0000000000d1','210c0000-0000-0000-0000-0000000000d2'), 40,10),
 -- back door: closed, unlocked, tavern↔alley
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c2','artifact','attrs.open',                 to_jsonb(false),                                                                                40,11),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c2','artifact','attrs.locked',               to_jsonb(false),                                                                                40,12),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c2','artifact','attrs.connects',             jsonb_build_array('210c0000-0000-0000-0000-0000000000d1','210c0000-0000-0000-0000-0000000000d3'), 40,13),
 -- cellar hatch: closed and LOCKED (Tier-1), tavern↔cellar. Mara holds the key; the cellar is where M-E4 happened.
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c3','artifact','attrs.open',                 to_jsonb(false),                                                                                40,14),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c3','artifact','attrs.locked',               to_jsonb(true),                                                                                 40,15),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000c3','artifact','attrs.connects',             jsonb_build_array('210c0000-0000-0000-0000-0000000000d1','210c0000-0000-0000-0000-0000000000d4'), 40,16),
 -- the cellar key, held by Mara
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000d1','artifact','attrs.size',                 to_jsonb(1),                                                                                    40,17),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000d1','artifact','attrs.held_by',              to_jsonb('2ac70000-0000-0000-0000-0000000000a2'::text),                                          40,18);

-- ── Kade's arrival (tick 50) — he steps into the room the scene is set in ─────────────
-- Replay-safe & append-only: one accepted ActorMoved with an ABSOLUTE attrs.location_id set (the
-- sm_project trigger projects it; replay_0a rebuilds it). Tick 50 is this world's max; the live
-- handler mints the next beat after it.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 ('2e000000-0000-0000-0000-0000000000fa','22222222-2222-2222-2222-222222222222','ActorMoved',
  'Kade steps into the Drowned Lantern.',50,0,
  'Arrival','accepted',now(),'public','fast_path');
-- Participant: the mover (instigator).
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('2e000000-0000-0000-0000-0000000000fa','2ac70000-0000-0000-0000-0000000000a1','actor','instigator');
-- Absolute location set → the Drowned Lantern. The projection trigger places Kade in the room.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000fa','2ac70000-0000-0000-0000-0000000000a1','actor','attrs.location_id',
  to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text),50,0);
-- Kade's own honest, minimal perception of stepping in. NOT an authored roster of who is present (that
-- would fake fan-out he never received) — just the move itself, subject-linked to the mover + the room.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 ('2ca40000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a1','2e000000-0000-0000-0000-0000000000fa',
  'I stepped into the Drowned Lantern.','direct',50,50);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('2ca40000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222'),
 ('2ca40000-0000-0000-0000-0000000000a1','210c0000-0000-0000-0000-0000000000d1','22222222-2222-2222-2222-222222222222');

COMMIT;
