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
--   * the first playable room holds a real Tier-1 locked Portal (the cellar hatch);
--   * (Station F Task 7) the scene has SPACE: nested §3 coordinates over a 'Harbor Quarter of Vael'
--     parent, in-tavern positions for the four souls + the bar, and a Container (ballast crate) with a
--     heavy stone so §2 move-duration, §4 encumbrance and §5.3 portals all compute real results in play.
--     This is what fn_distance/fn_move_duration_actor/fn_effective_weight/fn_portal_permits read — the
--     hand-authored coordinates are a SANCTIONED test artifact (§3); production mints them (Task 6). This
--     makes the coordinates_distance migration's "the hand-authored seed world supplies all coordinates"
--     note true: it is THIS seed that supplies them.
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
 ('2ac70000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222','actor',   'Kade'),
 ('2ac70000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222','actor',   'Mara'),
 ('2ac70000-0000-0000-0000-0000000000a3','22222222-2222-2222-2222-222222222222','actor',   'Jonas'),
 ('2ac70000-0000-0000-0000-0000000000a4','22222222-2222-2222-2222-222222222222','actor',   'Hooded Woman'),
 ('210c0000-0000-0000-0000-0000000000d0','22222222-2222-2222-2222-222222222222','location','Harbor Quarter of Vael'),
 ('210c0000-0000-0000-0000-0000000000d1','22222222-2222-2222-2222-222222222222','location','The Drowned Lantern'),
 ('210c0000-0000-0000-0000-0000000000d2','22222222-2222-2222-2222-222222222222','location','Dock Street'),
 ('210c0000-0000-0000-0000-0000000000d3','22222222-2222-2222-2222-222222222222','location','Alley'),
 ('210c0000-0000-0000-0000-0000000000d4','22222222-2222-2222-2222-222222222222','location','Cellar'),
 ('2a7f0000-0000-0000-0000-0000000000b1','22222222-2222-2222-2222-222222222222','artifact','Sealed Note (gray wax)'),
 ('2a7f0000-0000-0000-0000-0000000000c1','22222222-2222-2222-2222-222222222222','artifact','Front Door'),
 ('2a7f0000-0000-0000-0000-0000000000c2','22222222-2222-2222-2222-222222222222','artifact','Back Door'),
 ('2a7f0000-0000-0000-0000-0000000000c3','22222222-2222-2222-2222-222222222222','artifact','Cellar Hatch'),
 ('2a7f0000-0000-0000-0000-0000000000d1','22222222-2222-2222-2222-222222222222','artifact','Cellar Key'),
 ('2a7f0000-0000-0000-0000-0000000000f1','22222222-2222-2222-2222-222222222222','artifact','the bar'),
 ('2a7f0000-0000-0000-0000-0000000000f2','22222222-2222-2222-2222-222222222222','artifact','Ballast Crate'),
 ('2a7f0000-0000-0000-0000-0000000000f3','22222222-2222-2222-2222-222222222222','artifact','Ballast Stone');

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
-- weight, tension (see core/api/tier1.go). carry is the single Tier-1 key `contained_by` (§4 eager
-- encumbrance requires carry to be engine-readable state; the former Tier-2 carried_by/held_by are
-- unified into it — contents of X = entities whose contained_by = X, actors are root carriers). connects is the
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
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000b1','artifact','attrs.contained_by',         to_jsonb('2ac70000-0000-0000-0000-0000000000a1'::text),                                          40,7),
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
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000d1','artifact','attrs.contained_by',         to_jsonb('2ac70000-0000-0000-0000-0000000000a2'::text),                                          40,18),
 -- Tier-2 scene DESCRIPTION per location (Defect B): the narrate PLACE line renders it, so the room's
 -- fixed character is DATA the narrator draws on, never something it invents. The Drowned Lantern text
 -- is verbatim from FINAL-drowned-lantern-souls.md's scene section; the three stubs are brief, honest
 -- one-liners so movement has somewhere with a described face to go.
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d1','location','attrs.description', to_jsonb('Low beams, salt-rot, one hearth, a bar with a hatch, a back door to the alley.'::text), 40,19),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d2','location','attrs.description', to_jsonb('A rain-slick harbor road; gulls, tar, and black water past the pilings.'::text),        40,20),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d3','location','attrs.description', to_jsonb('A narrow dead-end behind the tavern; stacked crates and standing water.'::text),         40,21),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d4','location','attrs.description', to_jsonb('A cold stone undercroft beneath the tavern; barrels, damp, one shuttered lantern.'::text),   40,22);

-- ── §3 SPATIAL LAYER (Station F Task 7) — the scene gets space, under the same scene-genesis event f9 ──
-- Nested coordinates (FINAL-action-contracts.md §3): every location has a coordinate WITHIN its parent
-- (attrs.coordinates) + a parent edge (attrs.parent_location_id, Tier-1 string); a parent carries an
-- attrs.extent bounding its children. Things inside a scene (actors + fixed features) carry a coordinate
-- in that scene's LOCAL frame. fn_distance measures any pair at their nearest common parent's frame.
-- Coordinates are a SANCTIONED hand-authored test artifact (§3); production mints them (Task 6). Each
-- (entity, attribute_path) is written EXACTLY ONCE → replay-order-independent (Rider B, D-1). Tier-1 keys
-- only for engine-read attrs (coordinates, parent_location_id, max_room, empty_weight, weight, size,
-- contained_by; extent is descriptive). seq 26+ continues f9's single monotonic seq space.
--
-- Harbor Quarter frame (meters): tavern {200,200}; dock street {207,200} → 7 m ⇒ CEIL(7/1.4)=5 s, a SHORT
-- STEP out the front door onto the harbor road (Task 11 seed tune, RULINGS-2026-07-30 §1: the playable
-- moves must fit the beat budget — 5 s fits the tense 30 s budget so "step out the front" plays; the
-- earlier {280,200} put it 80 m ⇒ 58 s away, an over-budget dead end. Dock Street stays a DISTINCT
-- location behind the front-door portal — just a short step, not merged into the tavern); alley {200,240}
-- → 40 m (out the back); cellar {205,205} → beneath the tavern (portal-locked anyway). The quarter's
-- 2000×2000 extent bounds them.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- Harbor Quarter of Vael: the root parent (no parent edge), its own origin + the extent that bounds the rooms.
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d0','location','attrs.coordinates',        '{"x":0,"y":0}'::jsonb,       40,26),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d0','location','attrs.extent',             '{"w":2000,"h":2000}'::jsonb, 40,27),
 -- the four rooms: each a child of Harbor Quarter with a coordinate in the quarter frame.
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d1','location','attrs.parent_location_id', to_jsonb('210c0000-0000-0000-0000-0000000000d0'::text), 40,28),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d1','location','attrs.coordinates',        '{"x":200,"y":200}'::jsonb,   40,29),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d2','location','attrs.parent_location_id', to_jsonb('210c0000-0000-0000-0000-0000000000d0'::text), 40,30),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d2','location','attrs.coordinates',        '{"x":207,"y":200}'::jsonb,   40,31),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d3','location','attrs.parent_location_id', to_jsonb('210c0000-0000-0000-0000-0000000000d0'::text), 40,32),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d3','location','attrs.coordinates',        '{"x":200,"y":240}'::jsonb,   40,33),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d4','location','attrs.parent_location_id', to_jsonb('210c0000-0000-0000-0000-0000000000d0'::text), 40,34),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d4','location','attrs.coordinates',        '{"x":205,"y":205}'::jsonb,   40,35),
 -- Tavern local frame (meters): the three residents where the scene-genesis places them, and the bar
 -- feature along the back wall. Kade's own coordinate rides his arrival event (fa) below.
 --   Mara behind the bar {6,10}; Jonas by it {5,8}; the hooded woman at the corner table {1,1}; the bar {6,9}.
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a2','actor',   'attrs.coordinates',        '{"x":6,"y":10}'::jsonb,      40,36),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a3','actor',   'attrs.coordinates',        '{"x":5,"y":8}'::jsonb,       40,37),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a4','actor',   'attrs.coordinates',        '{"x":1,"y":1}'::jsonb,       40,38),
 -- the bar: a fixed room feature (FINAL "contains: the bar…"). location_id = the tavern (its scene) so
 -- fn_distance resolves it to the tavern frame; coordinates {6,9} along the back wall — the anchor Kade
 -- walks to (Task 8). size-2, weightless fixture (never relocated); Tier-2 descriptor for the narrator.
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f1','artifact','attrs.location_id',        to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text), 40,39),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f1','artifact','attrs.coordinates',        '{"x":6,"y":9}'::jsonb,       40,40),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f1','artifact','attrs.size',              to_jsonb(2),                  40,41),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f1','artifact','attrs.descriptor',        to_jsonb('the bar'::text),    40,42),
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
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f2','artifact','attrs.max_room',          to_jsonb(16),                 40,43),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f2','artifact','attrs.empty_weight',      to_jsonb(8),                  40,44),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f2','artifact','attrs.size',              to_jsonb(4),                  40,45),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f2','artifact','attrs.location_id',       to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text), 40,46),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f2','artifact','attrs.coordinates',        '{"x":2,"y":9}'::jsonb,       40,47),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f3','artifact','attrs.weight',            to_jsonb(92),                 40,48),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f3','artifact','attrs.size',              to_jsonb(2),                  40,49),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2a7f0000-0000-0000-0000-0000000000f3','artifact','attrs.contained_by',      to_jsonb('2a7f0000-0000-0000-0000-0000000000f2'::text), 40,50);

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
-- §3/§4 (Task 7): Kade also arrives WITH a position and a carrying capacity. coordinates {6,1} put him
-- just inside the front door — 8 m from the bar {6,9} ⇒ fn_distance(Kade,bar)=8, CEIL(8/1.4)=6 s (fits
-- tense's 30 s beat: the Task-8 "walk to the bar"). max_load 80 is his static capacity: the ballast crate
-- weighs 100 kg, so grabbing it exceeds max_load → the eager encumbrance rule (§4) can fire in play.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000fa','2ac70000-0000-0000-0000-0000000000a1','actor','attrs.location_id',
  to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text),50,0),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000fa','2ac70000-0000-0000-0000-0000000000a1','actor','attrs.max_load',    to_jsonb(80),           50,2),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000fa','2ac70000-0000-0000-0000-0000000000a1','actor','attrs.coordinates', '{"x":6,"y":1}'::jsonb, 50,3);
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
 ('2e000000-0000-0000-0000-0000000000e0','22222222-2222-2222-2222-222222222222','world_genesis',
  'the harbor-quarter figures known to each other by name (per-viewer identity substrate)',25,0,
  'Genesis','accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('2e000000-0000-0000-0000-0000000000e0','2ac70000-0000-0000-0000-0000000000a1','actor','named'),
 ('2e000000-0000-0000-0000-0000000000e0','2ac70000-0000-0000-0000-0000000000a2','actor','named'),
 ('2e000000-0000-0000-0000-0000000000e0','2ac70000-0000-0000-0000-0000000000a3','actor','named'),
 ('2e000000-0000-0000-0000-0000000000e0','2ac70000-0000-0000-0000-0000000000a4','actor','named');

-- Name perceptions (content = the name the viewer knows; source = genesis; subject = the named entity).
-- Fixed perception_ids (prefix 2a4e = "name"). Held per-viewer ⇒ each is that viewer's knowledge alone.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 -- Kade knows Mara by name (five years ago).
 ('2a4e0000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a1','2e000000-0000-0000-0000-0000000000e0','Mara','told',25,25),
 -- Mara knows Jonas by name (regulars).
 ('2a4e0000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a2','2e000000-0000-0000-0000-0000000000e0','Jonas','told',25,25),
 -- Jonas knows Mara by name (regulars).
 ('2a4e0000-0000-0000-0000-0000000000a3','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a3','2e000000-0000-0000-0000-0000000000e0','Mara','told',25,25),
 -- Mara PRIVATELY knows Kade — as "Reyna's brother", the name he had then, not the one he carries now.
 -- Held by Mara ALONE ⇒ only her own calls resolve it; the wall holds (part of her secret cluster).
 ('2a4e0000-0000-0000-0000-0000000000a4','22222222-2222-2222-2222-222222222222',
  '2ac70000-0000-0000-0000-0000000000a2','2e000000-0000-0000-0000-0000000000e0','Reyna''s brother','told',25,25);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('2a4e0000-0000-0000-0000-0000000000a1','2ac70000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222'),
 ('2a4e0000-0000-0000-0000-0000000000a2','2ac70000-0000-0000-0000-0000000000a3','22222222-2222-2222-2222-222222222222'),
 ('2a4e0000-0000-0000-0000-0000000000a3','2ac70000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222'),
 ('2a4e0000-0000-0000-0000-0000000000a4','2ac70000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222');

-- DESCRIPTOR fallbacks (Tier-2 attrs.descriptor) — what a viewer with no name-knowledge sees. Via
-- state_mutation (sm_project → actor_state), replay-safe (each (entity,path) written once). The three
-- residents under the scene-genesis event f9 (tick 40); Kade under his arrival fa (tick 50).
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a2','actor','attrs.descriptor', to_jsonb('the keeper'::text),                    40,23),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a3','actor','attrs.descriptor', to_jsonb('the muscle by the bar'::text),         40,24),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','2ac70000-0000-0000-0000-0000000000a4','actor','attrs.descriptor', to_jsonb('a hooded figure'::text),               40,25),
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000fa','2ac70000-0000-0000-0000-0000000000a1','actor','attrs.descriptor', to_jsonb('a young stranger, dark-haired'::text), 50,1);

COMMIT;
