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

-- (E1, noise, E102 appended by later tasks, before COMMIT)
COMMIT;
