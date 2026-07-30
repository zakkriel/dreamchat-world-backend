BEGIN;
SELECT plan(6);

-- Station F / FINAL-action-contracts.md §4 (ObjectRelocated): two dimensions, exactly.
--   VOLUME blocks (geometric impossibility): occupied_room + 4^(size-1) <= max_room, computed at
--     ask-time, NEVER stored. A size-5 crate cannot enter a size-4 pouch.
--   WEIGHT never blocks; it CONSEQUENCES via the EAGER encumbrance rule (founder-locked): on any
--     commit that changes a carry chain, recompute carried_weight for every affected carrier up the
--     chain and write/clear the seeded `encumbered` status (movement -100%) in the SAME commit.
--   effective_weight(container) = (empty_weight + Σ effective_weight(contents)) × weight_modifier.
--
-- Self-contained fixture (fixed uuids, no seed dependency). Containment is the Tier-1 `contained_by`
-- edge: contents of X = artifacts whose attrs->>'contained_by' = X. Actors are root carriers.
--   world f4000000-ffff-...
--     actor A  (a1)  max_load = 80            — the grabber
--     room  d1 (d1)  a location               — the floor things sit on / get dropped to
--     pouch c1 (c1)  max_room = 4             — tiny container, for the VOLUME dimension
--     crate5   (e5)  size 5  (plain object)   — 4^4 = 256, cannot fit the pouch
--     pebble   (e1)  size 1  (plain object)   — 4^0 = 1, fits
--     pack  c2 (c2)  empty_weight 2, mod 1.0  — the waterlogged pack, for the WEIGHT dimension
--       └ 4 crates (41..44) empty_weight 25, weight_modifier 1.6 → (25+0)×1.6 = 40 each
--         → pack effective_weight = (2 + 4×40) × 1.0 = 162
--
-- TEST STRUCTURE NOTE: a mutating engine call (apply_event/apply_ruled_event) and the read that checks
-- its effect MUST be in SEPARATE top-level statements. STABLE readers (fn_occupied_room,
-- fn_effective_weight) and plain sub-SELECTs use the CALLING STATEMENT's snapshot, so a read in the
-- same statement as the write would not see the write. So each committing case does the relocation in a
-- DO block first, then asserts the persisted state next. A committed relocation ⟺ its ObjectRelocated
-- canon_event is present at that tick (a gate_reject writes nothing) — that is the committed proxy here.

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('f4000000-0000-0000-0000-0000000000a1','f4000000-ffff-0000-0000-000000000000','actor',   'A'),
  ('f4000000-0000-0000-0000-0000000000d1','f4000000-ffff-0000-0000-000000000000','location','room'),
  ('f4000000-0000-0000-0000-0000000000c1','f4000000-ffff-0000-0000-000000000000','artifact','pouch'),
  ('f4000000-0000-0000-0000-0000000000c2','f4000000-ffff-0000-0000-000000000000','artifact','pack'),
  ('f4000000-0000-0000-0000-0000000000e5','f4000000-ffff-0000-0000-000000000000','artifact','crate5'),
  ('f4000000-0000-0000-0000-0000000000e1','f4000000-ffff-0000-0000-000000000000','artifact','pebble'),
  ('f4000000-0000-0000-0000-000000000041','f4000000-ffff-0000-0000-000000000000','artifact','wcrate1'),
  ('f4000000-0000-0000-0000-000000000042','f4000000-ffff-0000-0000-000000000000','artifact','wcrate2'),
  ('f4000000-0000-0000-0000-000000000043','f4000000-ffff-0000-0000-000000000000','artifact','wcrate3'),
  ('f4000000-0000-0000-0000-000000000044','f4000000-ffff-0000-0000-000000000000','artifact','wcrate4');

-- Actor A, at the room, max_load 80, no statuses yet.
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('f4000000-0000-0000-0000-0000000000a1','f4000000-ffff-0000-0000-000000000000',
   '{"location_id":"f4000000-0000-0000-0000-0000000000d1","max_load":80}'::jsonb);

INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
  -- pouch: a container with room for exactly a size-4 thing (mundane max_room <= 4^(size-1)).
  ('f4000000-0000-0000-0000-0000000000c1','f4000000-ffff-0000-0000-000000000000',
   '{"max_room":4}'::jsonb),
  -- the two volume-test objects sit in the room (contained_by = the location, i.e. on the floor).
  ('f4000000-0000-0000-0000-0000000000e5','f4000000-ffff-0000-0000-000000000000',
   '{"size":5,"weight":9,"contained_by":"f4000000-0000-0000-0000-0000000000d1"}'::jsonb),
  ('f4000000-0000-0000-0000-0000000000e1','f4000000-ffff-0000-0000-000000000000',
   '{"size":1,"weight":1,"contained_by":"f4000000-0000-0000-0000-0000000000d1"}'::jsonb),
  -- the waterlogged pack: empty_weight 2, mundane modifier 1.0, sitting on the floor of the room.
  ('f4000000-0000-0000-0000-0000000000c2','f4000000-ffff-0000-0000-000000000000',
   '{"empty_weight":2,"weight_modifier":1.0,"size":6,"contained_by":"f4000000-0000-0000-0000-0000000000d1"}'::jsonb),
  -- 4 waterlogged crates inside the pack: each (25 + 0) × 1.6 = 40 (exercises weight_modifier + recursion).
  ('f4000000-0000-0000-0000-000000000041','f4000000-ffff-0000-0000-000000000000',
   '{"empty_weight":25,"weight_modifier":1.6,"contained_by":"f4000000-0000-0000-0000-0000000000c2"}'::jsonb),
  ('f4000000-0000-0000-0000-000000000042','f4000000-ffff-0000-0000-000000000000',
   '{"empty_weight":25,"weight_modifier":1.6,"contained_by":"f4000000-0000-0000-0000-0000000000c2"}'::jsonb),
  ('f4000000-0000-0000-0000-000000000043','f4000000-ffff-0000-0000-000000000000',
   '{"empty_weight":25,"weight_modifier":1.6,"contained_by":"f4000000-0000-0000-0000-0000000000c2"}'::jsonb),
  ('f4000000-0000-0000-0000-000000000044','f4000000-ffff-0000-0000-000000000000',
   '{"empty_weight":25,"weight_modifier":1.6,"contained_by":"f4000000-0000-0000-0000-0000000000c2"}'::jsonb);

-- ──────────────────────────────────────────────────────────────────────────────
-- (a) fn_volume is the derived geometry: volume(size) = 4^(size-1). NOT stored (§4, decision 3).
-- ──────────────────────────────────────────────────────────────────────────────
SELECT ok(
  fn_volume(1) = 1::numeric AND fn_volume(5) = 256::numeric,
  '(a) fn_volume(1)=1 and fn_volume(5)=256 (4^(size-1), derived)');

-- ──────────────────────────────────────────────────────────────────────────────
-- (b) VOLUME BLOCKS: relocate the size-5 crate into the max_room=4 pouch.
--     0 + 4^4 = 256 > 4 → gate_reject (the function returns it), and NOTHING is written.
--     (No mutation → same-statement read is safe here.)
-- ──────────────────────────────────────────────────────────────────────────────
SELECT ok(
  (apply_event(
    'f4000000-ffff-0000-0000-000000000000',
    'f4000000-0000-0000-0000-0000000000a1',
    jsonb_build_object('type','ObjectRelocated','stated','put crate5 in pouch',
      'object_id','f4000000-0000-0000-0000-0000000000e5',
      'dest_id',  'f4000000-0000-0000-0000-0000000000c1'),
    3000, 0, 'fast_path'
  ))->>'halt_reason' = 'gate_reject'
  AND NOT EXISTS (SELECT 1 FROM canon_event
                  WHERE world_id='f4000000-ffff-0000-0000-000000000000' AND in_world_tick=3000),
  '(b) size-5 into max_room=4 → gate_reject, nothing written (volume is a true blocker)');

-- ──────────────────────────────────────────────────────────────────────────────
-- (c) VOLUME PASSES: relocate the size-1 pebble into the pouch. 0 + 4^0 = 1 <= 4 → committed.
--     occupied_room is DERIVED at ask-time (fn_occupied_room = Σ volume of contents), never stored;
--     after the commit it reflects the pebble now inside → 1.
-- ──────────────────────────────────────────────────────────────────────────────
DO $$ BEGIN
  PERFORM apply_event(
    'f4000000-ffff-0000-0000-000000000000',
    'f4000000-0000-0000-0000-0000000000a1',
    jsonb_build_object('type','ObjectRelocated','stated','put pebble in pouch',
      'object_id','f4000000-0000-0000-0000-0000000000e1',
      'dest_id',  'f4000000-0000-0000-0000-0000000000c1'),
    3001, 0, 'fast_path');
END $$;
SELECT ok(
  EXISTS (SELECT 1 FROM canon_event
          WHERE world_id='f4000000-ffff-0000-0000-000000000000'
            AND in_world_tick=3001 AND event_type='ObjectRelocated')
  AND fn_occupied_room('f4000000-ffff-0000-0000-000000000000',
                       'f4000000-0000-0000-0000-0000000000c1') = 1::numeric,
  '(c) size-1 into max_room=4,occupied_room=0 → committed; fn_occupied_room now reflects it (=1)');

-- ──────────────────────────────────────────────────────────────────────────────
-- (d) The container formula, recursive: pack = (2 + 4×40) × 1.0 = 162. Each crate = (25+0)×1.6 = 40.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  fn_effective_weight('f4000000-ffff-0000-0000-000000000000',
                      'f4000000-0000-0000-0000-0000000000c2'),
  162::numeric,
  '(d) fn_effective_weight(waterlogged pack) = (2 + 4×40) × 1.0 = 162');

-- ──────────────────────────────────────────────────────────────────────────────
-- (e) EAGER encumbrance: actor A (max_load 80) grabs the pack. Same commit writes carried_weight=162
--     and, since 162 > 80, sets `encumbered` in attrs.statuses. The grab HAPPENED (canon holds it).
-- ──────────────────────────────────────────────────────────────────────────────
DO $$ BEGIN
  PERFORM apply_ruled_event(
    'f4000000-ffff-0000-0000-000000000000',
    jsonb_build_object('type','ObjectRelocated','actor_id','f4000000-0000-0000-0000-0000000000a1',
      'truth','A grabs the waterlogged pack',
      'object_id','f4000000-0000-0000-0000-0000000000c2',
      'dest_id',  'f4000000-0000-0000-0000-0000000000a1'),
    3002, 0, 'ruling');
END $$;
SELECT ok(
  EXISTS (SELECT 1 FROM canon_event
          WHERE world_id='f4000000-ffff-0000-0000-000000000000'
            AND in_world_tick=3002 AND event_type='ObjectRelocated')
  AND (SELECT (attrs->>'carried_weight')::numeric FROM actor_state
       WHERE world_id='f4000000-ffff-0000-0000-000000000000'
         AND entity_id='f4000000-0000-0000-0000-0000000000a1') = 162::numeric
  AND (SELECT attrs->'statuses' FROM actor_state
       WHERE world_id='f4000000-ffff-0000-0000-000000000000'
         AND entity_id='f4000000-0000-0000-0000-0000000000a1') ? 'encumbered',
  '(e) A (max_load 80) grabs the pack → committed; carried_weight=162; encumbered ∈ attrs.statuses');

-- ──────────────────────────────────────────────────────────────────────────────
-- (f) Relocation needs no move (§4): A drops the pack to the floor. The same commit recomputes A up
--     the chain → carried_weight back to 0 → `encumbered` cleared. The strain releases the instant it ends.
-- ──────────────────────────────────────────────────────────────────────────────
DO $$ BEGIN
  PERFORM apply_ruled_event(
    'f4000000-ffff-0000-0000-000000000000',
    jsonb_build_object('type','ObjectRelocated','actor_id','f4000000-0000-0000-0000-0000000000a1',
      'truth','A drops the pack',
      'object_id','f4000000-0000-0000-0000-0000000000c2',
      'dest_id',  'f4000000-0000-0000-0000-0000000000d1'),
    3003, 0, 'ruling');
END $$;
SELECT ok(
  (SELECT (attrs->>'carried_weight')::numeric FROM actor_state
       WHERE world_id='f4000000-ffff-0000-0000-000000000000'
         AND entity_id='f4000000-0000-0000-0000-0000000000a1') = 0::numeric
  AND NOT ((SELECT COALESCE(attrs->'statuses','[]'::jsonb) FROM actor_state
       WHERE world_id='f4000000-ffff-0000-0000-000000000000'
         AND entity_id='f4000000-0000-0000-0000-0000000000a1') ? 'encumbered'),
  '(f) A drops the pack → committed; carried_weight back to 0; encumbered cleared');

SELECT * FROM finish();
ROLLBACK;
