BEGIN;
SELECT plan(1);
-- Hand-derived from seed_mara_0A.sql loop (f0056c7): for i in 1..100,
--   actor   = actors[(i % 8) + 1]  (8 actors, 0-based Python mod)
--   loc_id  = loc_ids[(i % 5) + 1] (5 locs: Tavern,Square,Market,Road,Dock)
-- Final location = last i where actor was chosen (highest i wins):
--   P  (actor slot 0): last noise at i=96 → Square, BUT seed_drowned_lantern's Kade arrival
--       (ActorMoved @ tick 300 > all noise ticks) re-places the Player → Tavern
--       → dddddddd-dddd-dddd-dddd-dddddddddddd  (founder-gate placement fix)
--   M  (actor slot 1): last at i=97  → i%5=2 → Market  → 000000b0-0000-0000-0000-0000000000b1
--   J  (actor slot 2): last at i=98  → i%5=3 → Road    → 000000c0-0000-0000-0000-0000000000c1
--   O1 (actor slot 3): last at i=99  → i%5=4 → Dock    → 000000d0-0000-0000-0000-0000000000d1
--   O2 (actor slot 4): last at i=100 → i%5=0 → Tavern  → dddddddd-dddd-dddd-dddd-dddddddddddd
--   O3 (actor slot 5): last at i=93  → i%5=3 → Road    → 000000c0-0000-0000-0000-0000000000c1
--   O4 (actor slot 6): last at i=94  → i%5=4 → Dock    → 000000d0-0000-0000-0000-0000000000d1
--   O5 (actor slot 7): last at i=95  → i%5=0 → Tavern  → dddddddd-dddd-dddd-dddd-dddddddddddd
-- state_mutations write to_jsonb(loc_id::text) → attrs->>location_id is the UUID string.
CREATE TEMP TABLE expected (entity_id uuid, location_id text);
INSERT INTO expected VALUES
 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','dddddddd-dddd-dddd-dddd-dddddddddddd'),  -- P  → Tavern (Kade arrival @ tick 300)
 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','000000b0-0000-0000-0000-0000000000b1'),  -- M  → Market
 ('cccccccc-cccc-cccc-cccc-cccccccccccc','000000c0-0000-0000-0000-0000000000c1'),  -- J  → Road
 ('00000000-0000-0000-0000-000000000001','000000d0-0000-0000-0000-0000000000d1'),  -- O1 → Dock
 ('00000000-0000-0000-0000-000000000002','dddddddd-dddd-dddd-dddd-dddddddddddd'),  -- O2 → Tavern
 ('00000000-0000-0000-0000-000000000003','000000c0-0000-0000-0000-0000000000c1'),  -- O3 → Road
 ('00000000-0000-0000-0000-000000000004','000000d0-0000-0000-0000-0000000000d1'),  -- O4 → Dock
 ('00000000-0000-0000-0000-000000000005','dddddddd-dddd-dddd-dddd-dddddddddddd'); -- O5 → Tavern
SELECT set_eq(
  'SELECT entity_id, attrs->>''location_id'' FROM actor_state WHERE world_id = ''11111111-1111-1111-1111-111111111111''',
  'SELECT entity_id, location_id FROM expected',
  'actor_state final locations match the hand-computed golden (doc 13 §5 spot-check) — uuid refs post-f0056c7');
SELECT * FROM finish();
ROLLBACK;
