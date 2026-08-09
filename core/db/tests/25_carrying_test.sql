-- fn_carrying — the Carrying overlay (mvp_slice_and_bridge §4.1; PRD Artifacts & Carrying).
--
-- Runs against BOTH seeded worlds on purpose: the Mara 0A fixture, whose Player carries nothing (the
-- empty case is a real answer and has its own shape), and The Drowned Lantern, where Kade and Mara
-- each hold exactly one thing — which is what makes the per-carrier negative below meaningful.
--
--   fixture   11111111-…  Player aaaaaaaa-…
--   lantern   22222222-…  Kade 2ac7…a1  Mara 2ac7…a2
--             note 2a7f…b1  cellar key 2a7f…d1  ballast crate 2a7f…f2  ballast stone 2a7f…f3
BEGIN;
SELECT plan(14);

-- ── envelope ────────────────────────────────────────────────────────────────────────────────────
SELECT is(
  fn_carrying('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->>'schema_version',
  'carrying/1',
  'the envelope declares carrying/1');

-- Carrying nothing is an ANSWER, not a missing page: an envelope with an empty array, never NULL.
SELECT ok(
  fn_carrying('11111111-1111-1111-1111-111111111111',
              'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa') IS NOT NULL
  AND json_array_length(fn_carrying('11111111-1111-1111-1111-111111111111',
                                    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')->'carried') = 0,
  'a viewer who carries nothing gets an envelope with an empty carried list, not NULL');

-- ── what the carrier holds ──────────────────────────────────────────────────────────────────────
SELECT is(
  (SELECT c->>'id'
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a1')->'carried') c),
  '2a7f0000-0000-0000-0000-0000000000b1',
  'Kade carries the sealed note');

SELECT is(
  (SELECT c->>'label'
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a1')->'carried') c),
  fn_display_name('22222222-2222-2222-2222-222222222222',
                  '2ac70000-0000-0000-0000-0000000000a1',
                  '2a7f0000-0000-0000-0000-0000000000b1'),
  'the label is the viewer''s own naming (fn_display_name), not the canonical identity');

SELECT is(
  (SELECT c->>'state'
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a1')->'carried') c),
  'carried',
  'state is the one value contained_by can honestly produce');

SELECT ok(
  (SELECT json_typeof(c->'container') = 'null'
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a1')->'carried') c),
  'a thing directly on you reports a null container');

-- Artifacts AC#3: last_confirmed_tick comes from the append-only ledger, so there is an accepted
-- event behind it. Reading artifact_state instead would have had to invent this number.
SELECT is(
  (SELECT (c->>'last_confirmed_tick')::bigint
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a1')->'carried') c),
  (SELECT sm.valid_from_tick FROM state_mutation sm
    WHERE sm.world_id = '22222222-2222-2222-2222-222222222222'
      AND sm.entity_id = '2a7f0000-0000-0000-0000-0000000000b1'
      AND sm.attribute_path = 'attrs.contained_by'
    ORDER BY sm.valid_from_tick DESC, sm.valid_from_seq DESC LIMIT 1),
  'last_confirmed_tick is the tick of the newest applied contained_by mutation');

SELECT ok(
  (SELECT (c->'decay'->>'stale')::boolean IS NOT NULL
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a1')->'carried') c),
  'decay.stale is computed, so a stale carry state can render decay language (AC#3)');

-- ── the per-carrier negative: this is the whole point of the surface ────────────────────────────
-- Mara holds the Cellar Key. It must never appear in Kade's overlay, and the reason it cannot is
-- structural: fn_carrying has no carrier argument, so the carrier is always the viewer.
SELECT ok(
  NOT EXISTS (
    SELECT 1 FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                                  '2ac70000-0000-0000-0000-0000000000a1')->'carried') c
     WHERE c->>'id' = '2a7f0000-0000-0000-0000-0000000000d1'),
  'Kade''s overlay never shows what Mara is carrying');

SELECT is(
  (SELECT c->>'id'
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a2')->'carried') c),
  '2a7f0000-0000-0000-0000-0000000000d1',
  'Mara''s own overlay does show her cellar key');

-- ── nesting: the engine already charges the whole subtree to the root carrier ───────────────────
-- The Ballast Stone sits inside the Ballast Crate. Hand the crate to Kade and both must appear —
-- an overlay that showed only the top layer would contradict the carried_weight the same world
-- computes in fn_apply_carry_change, which climbs contained_by to the root carrier.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
SELECT '22222222-2222-2222-2222-222222222222', sm.event_id,
       '2a7f0000-0000-0000-0000-0000000000f2', 'artifact', 'attrs.contained_by',
       to_jsonb('2ac70000-0000-0000-0000-0000000000a1'::text), 9000, 0
  FROM state_mutation sm
 WHERE sm.world_id = '22222222-2222-2222-2222-222222222222'
   AND sm.entity_id = '2a7f0000-0000-0000-0000-0000000000f3'
   AND sm.attribute_path = 'attrs.contained_by'
 LIMIT 1;

SELECT ok(
  EXISTS (
    SELECT 1 FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                                  '2ac70000-0000-0000-0000-0000000000a1')->'carried') c
     WHERE c->>'id' = '2a7f0000-0000-0000-0000-0000000000f3'
       AND c->'container'->>'id' = '2a7f0000-0000-0000-0000-0000000000f2'),
  'a thing inside a container you carry appears, naming the container it is in');

-- ── the ledger decides, and the newest applied edge wins ────────────────────────────────────────
-- Give the note to Mara at a later tick. It must leave Kade's overlay and enter hers, with no
-- special case: "who has it now" is just the newest contained_by mutation.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
SELECT '22222222-2222-2222-2222-222222222222', sm.event_id,
       '2a7f0000-0000-0000-0000-0000000000b1', 'artifact', 'attrs.contained_by',
       to_jsonb('2ac70000-0000-0000-0000-0000000000a2'::text), 9001, 0
  FROM state_mutation sm
 WHERE sm.world_id = '22222222-2222-2222-2222-222222222222'
   AND sm.entity_id = '2a7f0000-0000-0000-0000-0000000000b1'
   AND sm.attribute_path = 'attrs.contained_by'
 LIMIT 1;

SELECT ok(
  NOT EXISTS (
    SELECT 1 FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                                  '2ac70000-0000-0000-0000-0000000000a1')->'carried') c
     WHERE c->>'id' = '2a7f0000-0000-0000-0000-0000000000b1')
  AND EXISTS (
    SELECT 1 FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                                  '2ac70000-0000-0000-0000-0000000000a2')->'carried') c
     WHERE c->>'id' = '2a7f0000-0000-0000-0000-0000000000b1'),
  'handing a thing over moves it between overlays — the newest applied edge wins');

-- Putting a thing down is an ObjectRelocated whose destination is a place, not a person: apply_event
-- always names a dest_eid (state_mutation.new_value is NOT NULL, so there is no "contained by
-- nothing" edge to write). A destination that is not the viewer simply stops rooting the chain at
-- them — the thing leaves the overlay by the same mechanism that put it there, with no tombstone and
-- no special case.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                            new_value, valid_from_tick, valid_from_seq)
SELECT '22222222-2222-2222-2222-222222222222', sm.event_id,
       '2a7f0000-0000-0000-0000-0000000000b1', 'artifact', 'attrs.contained_by',
       to_jsonb('210c0000-0000-0000-0000-0000000000d1'::text), 9002, 0
  FROM state_mutation sm
 WHERE sm.world_id = '22222222-2222-2222-2222-222222222222'
   AND sm.entity_id = '2a7f0000-0000-0000-0000-0000000000b1'
   AND sm.attribute_path = 'attrs.contained_by'
 LIMIT 1;

SELECT ok(
  NOT EXISTS (
    SELECT 1 FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                                  '2ac70000-0000-0000-0000-0000000000a2')->'carried') c
     WHERE c->>'id' = '2a7f0000-0000-0000-0000-0000000000b1'),
  'putting a thing down into a room removes it from the overlay');

-- ── the wall on the knowledge field ─────────────────────────────────────────────────────────────
-- quick_inspect_preview reads fn_visible_perceptions only. A perception MARA holds about the crate
-- must not surface in KADE's preview, even though Kade is the one carrying it.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('e0000000-0000-0000-0000-000000000925', '22222222-2222-2222-2222-222222222222',
        'observation', 'Mara notices the crate has a false bottom', 9100, 0,
        'Scene', 'accepted', now(), 'private', 'fast_path');

INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick)
VALUES ('dca70000-0000-0000-0000-000000000925', '22222222-2222-2222-2222-222222222222',
        '2ac70000-0000-0000-0000-0000000000a2', 'e0000000-0000-0000-0000-000000000925',
        'The crate has a false bottom.', 'direct', 9100, 9100);

INSERT INTO perception_subject (perception_id, entity_id, world_id)
VALUES ('dca70000-0000-0000-0000-000000000925',
        '2a7f0000-0000-0000-0000-0000000000f2', '22222222-2222-2222-2222-222222222222');

SELECT ok(
  (SELECT json_typeof(c->'quick_inspect_preview') = 'null'
     FROM json_array_elements(fn_carrying('22222222-2222-2222-2222-222222222222',
                                          '2ac70000-0000-0000-0000-0000000000a1')->'carried') c
    WHERE c->>'id' = '2a7f0000-0000-0000-0000-0000000000f2'),
  'quick_inspect_preview stays null for the carrier who does not hold the perception (B-1)');

SELECT * FROM finish();
ROLLBACK;
