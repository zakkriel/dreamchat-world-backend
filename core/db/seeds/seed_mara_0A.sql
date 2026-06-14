-- =====================================================================================
-- seed_mara_0A.sql — deterministic Mara scenario (doc 13 §4). Targets a CLEAN DB ONLY.
-- Re-run path is `make reset` (NOT a self-clearing script). A seed that DELETEs canon or
-- disables append-only enforcement would be the silent-workaround pattern made executable.
-- CONVENTION (Rider B): every state_mutation.new_value is an ABSOLUTE state set at a single-key
-- path under attrs. (e.g. attrs.location_id). No deltas.
-- =====================================================================================
BEGIN;

-- Clean-DB guard: refuse to seed a non-empty world (canon is append-only; no DELETE here).
DO $$ BEGIN
  IF (SELECT count(*) FROM entity_registry) > 0 THEN
    RAISE EXCEPTION 'seed_mara_0A targets a CLEAN DB only — run `make reset` first';
  END IF;
END $$;

-- Cast (doc 13 §4) + noise NPCs ("...other moves", §4) + one named noise location + a
-- common-knowledge holder for the public record (§5 "a public knowledge record exists").
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111','actor',   'Player'),
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111','actor',   'Mara'),
 ('cccccccc-cccc-cccc-cccc-cccccccccccc','11111111-1111-1111-1111-111111111111','actor',   'Jonas'),
 ('dddddddd-dddd-dddd-dddd-dddddddddddd','11111111-1111-1111-1111-111111111111','location','Tavern'),
 ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','11111111-1111-1111-1111-111111111111','faction', 'Common Knowledge'),
 ('00000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','actor',   'O1'),
 ('00000000-0000-0000-0000-000000000002','11111111-1111-1111-1111-111111111111','actor',   'O2'),
 ('00000000-0000-0000-0000-000000000003','11111111-1111-1111-1111-111111111111','actor',   'O3'),
 ('00000000-0000-0000-0000-000000000004','11111111-1111-1111-1111-111111111111','actor',   'O4'),
 ('00000000-0000-0000-0000-000000000005','11111111-1111-1111-1111-111111111111','actor',   'O5'),
 ('000000a0-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111','location','Square');

-- E1 @ tick 100: P privately discloses the secret to M (doc 13 §4). No state mutation.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111',
        'private_disclosure','P tells M the mayor keeps a hidden ledger',100,0,
        'Day 1', 'accepted', now(), 'private', 'fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-000000000001','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','speaker'),
 ('e0000000-0000-0000-0000-000000000001','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','listener');
INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                               acquired_tick, valid_tick) VALUES
 ('11111111-1111-1111-1111-111111111111','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  'e0000000-0000-0000-0000-000000000001','P told me the mayor keeps a hidden ledger','told',100,100),
 ('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  'e0000000-0000-0000-0000-000000000001','I told Mara the mayor keeps a hidden ledger','shared',100,100);
-- 100 noise events, ticks 101..200, beat_seq 0 (each tick unique => total order). FULLY DETERMINISTIC:
-- event_id = 'e0000000-0000-0000-0000-9' + lpad(i,11,'0')  (collision-free vs E1 ...001 / E102 ...102).
-- Rule (hand-verifiable): for i in 1..100, tick=100+i,
--   actor    = (P,M,J,O1,O2,O3,O4,O5)[(i % 8)+1]   (1-based SQL arrays)
--   location = ('tavern','square','market','road','dock')[(i % 5)+1]
-- Each event: 'move', one ABSOLUTE attrs.location_id set, one 'direct' perception for the mover.
DO $$
DECLARE
  i int; tick bigint; ev uuid; actor uuid; loc text;
  actors uuid[] := ARRAY[
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
    'cccccccc-cccc-cccc-cccc-cccccccccccc','00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000005']::uuid[];
  locs text[] := ARRAY['tavern','square','market','road','dock'];
BEGIN
  FOR i IN 1..100 LOOP
    tick  := 100 + i;
    ev    := ('e0000000-0000-0000-0000-9' || lpad(i::text, 11, '0'))::uuid;
    actor := actors[(i % 8) + 1];
    loc   := locs[(i % 5) + 1];
    INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                             status, accepted_at, visibility_scope, origin)
    VALUES (ev,'11111111-1111-1111-1111-111111111111','move',
            'noise move '||i, tick, 0, 'accepted', now(), 'public', 'fast_path');
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
    VALUES (ev, actor, 'actor', 'instigator');
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES ('11111111-1111-1111-1111-111111111111', ev, actor, 'actor', 'attrs.location_id',
            to_jsonb(loc), tick, 0);
    INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                   acquired_tick, valid_tick)
    VALUES ('11111111-1111-1111-1111-111111111111', actor, ev, 'I moved to '||loc, 'direct', tick, tick);
  END LOOP;
END $$;
-- E102 @ tick 201: P publicizes the ledger. No state mutation. Present-forward (ADR-016):
-- M's E1 perception untouched; public-knowledge record created (held by Common Knowledge);
-- J is *eligible* but acquires nothing in 0A (Phase-1 fan-out, doc 13 §5).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-000000000102','11111111-1111-1111-1111-111111111111',
        'publicize','the hidden ledger becomes common knowledge',201,0,
        'Day 2', 'accepted', now(), 'public', 'fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-000000000102','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                               acquired_tick, valid_tick, visibility_scope) VALUES
 ('11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
  'e0000000-0000-0000-0000-000000000102','It is now common knowledge that the mayor keeps a hidden ledger',
  'public',201,201,'public');
-- =====================================================================================
-- Chunk-3 additions (design 2026-06-14). Deterministic. Chosen to miss every existing
-- scoped 0A assertion: name perceptions are 'public' sourced to world_genesis (≠ E102);
-- the about-Mara fixture is 'direct' sourced to E1 (≠ Player's 'shared'); no state_mutation
-- is added (replay/golden projections untouched).
-- =====================================================================================

-- (G) world_genesis @ tick 0 — sources common-knowledge identity (names). No state mutation.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-0000000000e0','11111111-1111-1111-1111-111111111111',
        'world_genesis','the world is established; its principal figures are publicly known',0,0,
        'Genesis','accepted', now(), 'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e0000000-0000-0000-0000-0000000000e0','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','faction','source');

-- (N) Common-knowledge NAME perceptions — principal cast only; held by Common Knowledge (PUB),
-- public, sourced to genesis. content = the canonical name (read at projection time via the
-- perception layer, NEVER a raw entity_registry read — going-in 5). O1..O5 deliberately omitted
-- so fn_perceived_name's WITHHOLD path is exercised on real seed rows. Fixed perception_ids so
-- the explicit subject links are unambiguous.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick, visibility_scope) VALUES
 ('ace00000-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Player','public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000b1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Mara',  'public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000c1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Jonas', 'public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000d1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Tavern','public',0,0,'public'),
 ('ace00000-0000-0000-0000-0000000000f1','11111111-1111-1111-1111-111111111111','eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee','e0000000-0000-0000-0000-0000000000e0','Square','public',0,0,'public');
-- explicit subjects for the name perceptions (one entity each)
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('ace00000-0000-0000-0000-0000000000a1','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000b1','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000c1','cccccccc-cccc-cccc-cccc-cccccccccccc','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000d1','dddddddd-dddd-dddd-dddd-dddddddddddd','11111111-1111-1111-1111-111111111111'),
 ('ace00000-0000-0000-0000-0000000000f1','000000a0-0000-0000-0000-0000000000a1','11111111-1111-1111-1111-111111111111');

-- (A) Player-private-about-Mara — the gate fixture. Genuinely about Mara (content-subject = Mara,
-- who is also an E1 participant → junction ⊆ derivation, future-proof). 'direct' (≠ Player's
-- 'shared' of E1, so the existing scoped assertion holds). Private → invisible to Jonas. Fixed id.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 ('dca70000-0000-0000-0000-000000000a01','11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','e0000000-0000-0000-0000-000000000001','Mara listened intently and seemed unsettled','direct',100,100);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('dca70000-0000-0000-0000-000000000a01','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','11111111-1111-1111-1111-111111111111');

-- (B) Generic about-ness backfill for the ORIGINAL event-derived perceptions only (ADR-035:
-- subjects = source event participants). NOT EXISTS skips rows that already carry explicit subjects
-- (the names and the about-Mara fixture), so those stay precise and are never over-attributed.
INSERT INTO perception_subject (perception_id, entity_id, world_id)
SELECT pr.perception_id, ep.entity_id, pr.world_id
FROM perception_record pr
JOIN event_participant ep ON ep.event_id = pr.source_event_id
WHERE NOT EXISTS (SELECT 1 FROM perception_subject ps WHERE ps.perception_id = pr.perception_id)
ON CONFLICT (perception_id, entity_id) DO NOTHING;

COMMIT;
