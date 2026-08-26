BEGIN;
SELECT plan(14);

-- SPEC-035 — an ObjectRelocated event must RECORD who saw it, and those witnesses must perceive it.
--
-- ── WRITTEN AGAINST THE REPRODUCTION, NOT THE REPORT ────────────────────────────────────────────
-- Measured on the seeded world (The Drowned Lantern) before the fix existed. Kade hands the Sealed
-- Note to Mara with Jonas standing in the same room, passing witnesses: [jonas]:
--
--     WHO SAW IT, per the event      → 'instigator'   ← Jonas is nowhere
--     Jonas's perceptions of it      → 0
--     witnesses[] kept in payload?   → {}             ← SILENTLY DISCARDED
--     Mara's perceptions (SPEC-034)  → 1              ← the holder arm already worked
--
-- The third line is the defect this suite defends against regressing. The caller could ALREADY name
-- witnesses and apply_event dropped the field without a word or a halt_reason, so "who watched this
-- handover" had no answer anywhere in the database — and per ADR-026 could never be given one after
-- the fact, because replay reproduces projection state and perceptions are not projections.
--
-- Self-contained fixture with fixed uuids, following 113/122. UNLIKE 122 this fixture needs
-- actor_state rows: co-presence is read through fn_actors_at, which reads actor_state.attrs, so
-- actors with no state row are nowhere and every named witness would be refused. Discovering that
-- is what this suite is for.

\set W  'f6000000-ffff-0000-0000-000000000000'
\set A1 'f6000000-0000-0000-0000-0000000000a1'
\set A2 'f6000000-0000-0000-0000-0000000000a2'
\set A3 'f6000000-0000-0000-0000-0000000000a3'
\set A4 'f6000000-0000-0000-0000-0000000000a4'
\set OB 'f6000000-0000-0000-0000-0000000000e1'
\set LC 'f6000000-0000-0000-0000-0000000000c1'
\set LF 'f6000000-0000-0000-0000-0000000000c2'

INSERT INTO world (world_id, display_name) VALUES (:'W'::uuid, 'SPEC-035 fixture')
  ON CONFLICT DO NOTHING;
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  (:'A1'::uuid, :'W'::uuid, 'actor',    'Giver'),
  (:'A2'::uuid, :'W'::uuid, 'actor',    'Taker'),
  (:'A3'::uuid, :'W'::uuid, 'actor',    'Watcher'),
  (:'A4'::uuid, :'W'::uuid, 'actor',    'Absent Friend'),
  (:'OB'::uuid, :'W'::uuid, 'artifact', 'Sealed Packet'),
  (:'LC'::uuid, :'W'::uuid, 'location', 'Tap Room'),
  (:'LF'::uuid, :'W'::uuid, 'location', 'Far Cellar');

-- Three in the Tap Room; the fourth is a room away. That distance is the whole point of the gate.
INSERT INTO actor_state (entity_id, world_id, attrs, dirty, updated_at) VALUES
  (:'A1'::uuid, :'W'::uuid, jsonb_build_object('location_id', :'LC'), false, now()),
  (:'A2'::uuid, :'W'::uuid, jsonb_build_object('location_id', :'LC'), false, now()),
  (:'A3'::uuid, :'W'::uuid, jsonb_build_object('location_id', :'LC'), false, now()),
  (:'A4'::uuid, :'W'::uuid, jsonb_build_object('location_id', :'LF'), false, now());

-- ── preconditions ──────────────────────────────────────────────────────────────────────────────
SELECT ok( NOT fn_entity_visible(:'W'::uuid, :'A3'::uuid, :'OB'::uuid),
  'precondition: the watcher cannot see the packet before anything happens' );
SELECT is( (SELECT COUNT(*)::int FROM fn_actors_at(:'W'::uuid, :'LC'::uuid)), 3,
  'precondition: three actors are co-present in the tap room, the fourth is not' );

-- ── the handover, naming its witness, through the gate (D-1) ────────────────────────────────────
DO $$
BEGIN
  PERFORM apply_event(
    'f6000000-ffff-0000-0000-000000000000'::uuid,
    'f6000000-0000-0000-0000-0000000000a1'::uuid,
    jsonb_build_object('type','ObjectRelocated','stated','The giver slides the sealed packet across.',
      'object_id','f6000000-0000-0000-0000-0000000000e1',
      'dest_kind','actor',
      'dest_id',  'f6000000-0000-0000-0000-0000000000a2',
      'witnesses', jsonb_build_array('f6000000-0000-0000-0000-0000000000a3')),
    6100, 0, 'fast_path');
END $$;

-- ── 1. THE EVENT RECORDS WHO SAW IT. This is the assertion the whole SPEC exists for. ───────────
SELECT is( (SELECT string_agg(role_qualifier, ',' ORDER BY role_qualifier)
              FROM event_participant
             WHERE event_id = (SELECT event_id FROM canon_event
                                WHERE world_id = :'W'::uuid AND in_world_tick = 6100)),
           'instigator,witness',
  'SPEC-035: the event names its witness, not just its instigator' );

-- ── 2. and the witness perceives it ────────────────────────────────────────────────────────────
SELECT ok( fn_entity_visible(:'W'::uuid, :'A3'::uuid, :'OB'::uuid),
  'SPEC-035: the named witness can now see the object that changed hands' );

-- ── 3. the holders are unaffected — SPEC-034 must not regress ─────────────────────────────────
SELECT ok( fn_entity_visible(:'W'::uuid, :'A2'::uuid, :'OB'::uuid)
       AND fn_entity_visible(:'W'::uuid, :'A1'::uuid, :'OB'::uuid),
  'SPEC-034 holds: both holders still perceive the handover' );

-- ── 4. one event, one perception each — a witness is not minted twice ──────────────────────────
SELECT is( (SELECT COUNT(*)::int FROM perception_record
             WHERE world_id = :'W'::uuid AND holder_id = :'A3'::uuid
               AND source_event_id = (SELECT event_id FROM canon_event
                                       WHERE world_id = :'W'::uuid AND in_world_tick = 6100)), 1,
  'SPEC-035: the witness holds exactly one perception of the one event' );

-- ── 5. A NAMED WITNESS WHO WAS NOT THERE IS REFUSED ───────────────────────────────────────────
-- The founder's ruling is that co-presence is necessary but not sufficient: the caller supplies
-- sufficiency by naming them, the engine supplies necessity by refusing the impossible. Same shape as
-- the Communicated listener gate. A world that lets you claim a witness in another room is a world
-- where perception is awarded rather than blocked (FINAL-action-contracts.md).
SELECT is( (SELECT apply_event(:'W'::uuid, :'A1'::uuid,
             jsonb_build_object('type','ObjectRelocated','stated','x',
               'object_id', :'OB', 'dest_kind','actor', 'dest_id', :'A2',
               'witnesses', jsonb_build_array(:'A4')),
             6200, 0, 'fast_path')->>'halt_reason'), 'gate_reject',
  'SPEC-035: naming a witness who was a room away is refused at the gate' );

-- ── 6. a holder named as their own witness is not double-recorded ──────────────────────────────
DO $$
BEGIN
  PERFORM apply_event(
    'f6000000-ffff-0000-0000-000000000000'::uuid,
    'f6000000-0000-0000-0000-0000000000a2'::uuid,
    jsonb_build_object('type','ObjectRelocated','stated','The taker hands it back.',
      'object_id','f6000000-0000-0000-0000-0000000000e1',
      'dest_kind','actor',
      'dest_id',  'f6000000-0000-0000-0000-0000000000a1',
      'witnesses', jsonb_build_array('f6000000-0000-0000-0000-0000000000a1',
                                     'f6000000-0000-0000-0000-0000000000a2')),
    6300, 0, 'fast_path');
END $$;

SELECT is( (SELECT COUNT(*)::int FROM event_participant
             WHERE event_id = (SELECT event_id FROM canon_event
                                WHERE world_id = :'W'::uuid AND in_world_tick = 6300)
               AND role_qualifier = 'witness'), 0,
  'SPEC-035: a party to the handover is not also recorded as its witness' );

SELECT is( (SELECT COUNT(*)::int FROM perception_record
             WHERE world_id = :'W'::uuid AND holder_id = :'A1'::uuid
               AND source_event_id = (SELECT event_id FROM canon_event
                                       WHERE world_id = :'W'::uuid AND in_world_tick = 6300)), 1,
  'SPEC-035: and so they hold one perception of it, not two' );

-- ── 7. no witnesses named = no witness rows. The field is opt-in. ──────────────────────────────
-- 122's sixth assertion depends on this: "a third actor the event does not name perceives nothing."
-- SPEC-035 did not have to change that assertion, because witnesses are NAMED rather than inferred
-- from co-presence — which is precisely the ruling. If this ever fails, 122 fails with it.
DO $$
BEGIN
  PERFORM apply_event(
    'f6000000-ffff-0000-0000-000000000000'::uuid,
    'f6000000-0000-0000-0000-0000000000a1'::uuid,
    jsonb_build_object('type','ObjectRelocated','stated','Quietly, this time.',
      'object_id','f6000000-0000-0000-0000-0000000000e1',
      'dest_kind','actor',
      'dest_id',  'f6000000-0000-0000-0000-0000000000a2'),
    6400, 0, 'fast_path');
END $$;

SELECT is( (SELECT COUNT(*)::int FROM event_participant
             WHERE event_id = (SELECT event_id FROM canon_event
                                WHERE world_id = :'W'::uuid AND in_world_tick = 6400)
               AND role_qualifier = 'witness'), 0,
  'SPEC-035: a handover naming no witnesses records none — co-presence alone perceives nothing' );

-- ── 8. PRESENT-BUT-MALFORMED IS A REFUSAL, NOT A SHRUG ─────────────────────────────────────────
-- Found by reviewing SPEC-035 one commit after it shipped, asking the question none of its four
-- mutants asked: not "what if the code is wrong" but "what if the INPUT is wrong". The first cut
-- keyed every branch on `jsonb_typeof(...) = 'array'`, so a bare string fell through all of them and
-- committed silently with zero witnesses and no halt_reason — the exact defect SPEC-035 was filed to
-- remove, reintroduced by its own fix.
--
-- This is not an exotic shape. `Communicated`'s recipient field is `listener_id`, a BARE STRING in
-- the same payload, so a caller reasoning from the nearest sibling writes exactly this.
--
-- absent and null are legitimate ("nobody was named"). Anything else present is a caller bug.
SELECT is( (SELECT apply_event(:'W'::uuid, :'A1'::uuid,
             jsonb_build_object('type','ObjectRelocated','stated','bare string',
               'object_id', :'OB', 'dest_kind','actor', 'dest_id', :'A2',
               'witnesses', :'A3'),
             6500, 0, 'fast_path')->>'halt_reason'), 'gate_reject',
  'SPEC-035: witnesses as a BARE STRING is refused, not silently dropped' );

SELECT is( (SELECT apply_event(:'W'::uuid, :'A1'::uuid,
             jsonb_build_object('type','ObjectRelocated','stated','a number',
               'object_id', :'OB', 'dest_kind','actor', 'dest_id', :'A2',
               'witnesses', 42),
             6501, 0, 'fast_path')->>'halt_reason'), 'gate_reject',
  'SPEC-035: witnesses as a number is refused' );

SELECT is( (SELECT apply_event(:'W'::uuid, :'A1'::uuid,
             jsonb_build_object('type','ObjectRelocated','stated','an object',
               'object_id', :'OB', 'dest_kind','actor', 'dest_id', :'A2',
               'witnesses', jsonb_build_object('who', :'A3')),
             6502, 0, 'fast_path')->>'halt_reason'), 'gate_reject',
  'SPEC-035: witnesses as an object is refused' );

-- and the two legitimate absences still commit — a refusal that also refuses valid input is worse
-- than the silence it replaced.
SELECT is( (SELECT apply_event(:'W'::uuid, :'A1'::uuid,
             jsonb_build_object('type','ObjectRelocated','stated','explicit null',
               'object_id', :'OB', 'dest_kind','actor', 'dest_id', :'A2',
               'witnesses', NULL),
             6503, 0, 'fast_path')->>'halt_reason'), 'committed',
  'SPEC-035: an explicit null witnesses field still commits — that is "nobody was named"' );

SELECT * FROM finish();
ROLLBACK;
