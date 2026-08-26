BEGIN;
SELECT plan(8);

-- SPEC-034 — a handover must make the object perceptible to the holders the event NAMES.
--
-- ── WRITTEN AGAINST THE REPRODUCTION, NOT THE REPORT ────────────────────────────────────────────
-- Measured on the seeded world before the fix existed:
--     fn_carrying(DL, Kade)             → lists the Sealed Note
--     fn_entity_visible(DL, Kade, note) → false
--     fn_artifact_page(DL, Kade, note)  → NULL  ⇒ 404
--     perceptions Kade holds re note    → 0
--     ObjectRelocated events in world   → 0
--
-- That last line is why this suite COMMITS A REAL EVENT instead of asserting on the seed: the seeded
-- carry edges were authored as STATE, so no perception rule can reach them. A suite that asserted on
-- the seeded row would pass with this arm deleted, which is the definition of a vacuous test.
--
-- Self-contained fixture with fixed uuids, no seed dependency — the pattern
-- 113_object_relocated_test.sql establishes for this event type. Relocations are committed in a DO
-- block before being asserted, because a read in the same statement as the write uses the calling
-- statement's snapshot and would not see it (113's own note).

\set W  'f5000000-ffff-0000-0000-000000000000'
\set A1 'f5000000-0000-0000-0000-0000000000a1'
\set A2 'f5000000-0000-0000-0000-0000000000a2'
\set A3 'f5000000-0000-0000-0000-0000000000a3'
\set OB 'f5000000-0000-0000-0000-0000000000e1'
\set LC 'f5000000-0000-0000-0000-0000000000c1'

INSERT INTO world (world_id, display_name) VALUES (:'W'::uuid, 'SPEC-034 fixture')
  ON CONFLICT DO NOTHING;
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  (:'A1'::uuid, :'W'::uuid, 'actor',    'Giver'),
  (:'A2'::uuid, :'W'::uuid, 'actor',    'Taker'),
  (:'A3'::uuid, :'W'::uuid, 'actor',    'Bystander'),
  (:'OB'::uuid, :'W'::uuid, 'artifact', 'Sealed Packet'),
  (:'LC'::uuid, :'W'::uuid, 'location', 'Back Room');

-- ── the defect's shape, before we act ───────────────────────────────────────────────────────────
SELECT ok( NOT fn_entity_visible(:'W'::uuid, :'A2'::uuid, :'OB'::uuid),
  'precondition: the receiver cannot see the object before the handover' );
SELECT ok( NOT fn_entity_visible(:'W'::uuid, :'A1'::uuid, :'OB'::uuid),
  'precondition: the giver cannot see it either' );

-- ── the handover, through the gate (D-1: never hand-written into canon) ─────────────────────────
DO $$
BEGIN
  PERFORM apply_event(
    'f5000000-ffff-0000-0000-000000000000'::uuid,
    'f5000000-0000-0000-0000-0000000000a1'::uuid,
    jsonb_build_object('type','ObjectRelocated','stated','The giver passes the sealed packet over.',
      'object_id','f5000000-0000-0000-0000-0000000000e1',
      'dest_kind','actor',
      'dest_id',  'f5000000-0000-0000-0000-0000000000a2'),
    5100, 0, 'fast_path');
END $$;

SELECT ok( EXISTS(SELECT 1 FROM canon_event
                  WHERE world_id = :'W'::uuid AND in_world_tick = 5100
                    AND event_type = 'ObjectRelocated' AND status = 'accepted'),
  'the gate accepted the handover' );

-- ── the fix ────────────────────────────────────────────────────────────────────────────────────
SELECT ok( fn_entity_visible(:'W'::uuid, :'A2'::uuid, :'OB'::uuid),
  'SPEC-034: the NEW holder can now see the object in his hands' );
SELECT ok( fn_entity_visible(:'W'::uuid, :'A1'::uuid, :'OB'::uuid),
  'SPEC-034: the actor who handed it over perceives it too' );
SELECT ok( fn_artifact_page(:'W'::uuid, :'A2'::uuid, :'OB'::uuid) IS NOT NULL,
  'SPEC-034: the Artifact page no longer 404s for the new holder' );

-- ── the negative case, and it is what keeps the arm honest ──────────────────────────────────────
-- The founder ruled that co-presence is not perception — "just because they were there doesn't mean
-- they saw it" — and ObjectRelocated carries no witness field, so this arm names nobody the event
-- does not name. If a later round widens this to witnesses (SPEC-035), THIS assertion is the one that
-- must change deliberately, with a ruling, rather than drifting.
SELECT ok( NOT fn_entity_visible(:'W'::uuid, :'A3'::uuid, :'OB'::uuid),
  'SPEC-034: a third actor the event does not name perceives nothing (witnesses are SPEC-035)' );

-- ── a destination that is not a person names no perceiver ───────────────────────────────────────
-- "The packet is now in the back room" is not knowledge anybody acquired. The receiver arm must key
-- on the destination's entity_kind from the registry, not on the payload's unenforced dest_kind.
DO $$
BEGIN
  PERFORM apply_event(
    'f5000000-ffff-0000-0000-000000000000'::uuid,
    'f5000000-0000-0000-0000-0000000000a2'::uuid,
    jsonb_build_object('type','ObjectRelocated','stated','The taker leaves the packet in the back room.',
      'object_id','f5000000-0000-0000-0000-0000000000e1',
      'dest_id',  'f5000000-0000-0000-0000-0000000000c1'),
    5200, 0, 'fast_path');
END $$;

SELECT is( (SELECT COUNT(DISTINCT pr.holder_id)::int
              FROM perception_record pr
             WHERE pr.world_id = :'W'::uuid
               AND pr.source_event_id = (SELECT event_id FROM canon_event
                                          WHERE world_id = :'W'::uuid AND in_world_tick = 5200
                                            AND event_type = 'ObjectRelocated')), 1,
  'SPEC-034: a drop into a location perceives for the actor only, not for the location' );

SELECT * FROM finish();
ROLLBACK;
