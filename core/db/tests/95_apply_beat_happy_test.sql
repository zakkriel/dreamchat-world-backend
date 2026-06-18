BEGIN;
SELECT plan(8);

-- Player and Mara co-present at tavern (setup moves). Then a 2-step beat: [say to Mara, move to square].
-- The say-gate is an EXISTS(Mara ∈ actors_at(tavern)) check (tolerant of seed actors also at tavern);
-- the move→square discovery is empty (Player left square via the setup), so no move-discovery rows.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000040','11111111-1111-1111-1111-111111111111','move','setup P→tavern',400,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000041','11111111-1111-1111-1111-111111111111','move','setup M→tavern',401,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000040','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000041','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000040','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),400,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000041','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('tavern'::text),401,0);

SELECT lives_ok($$
  SELECT apply_beat(
    '11111111-1111-1111-1111-111111111111',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    '[{"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"I saw the note"},
      {"type":"move","to":"square"}]'::jsonb,
    500, 100, 'fast_path')
$$, 'apply_beat runs a 2-step happy beat');

-- both steps committed (the beat is the only content at tick >= 500)
SELECT is((SELECT count(*) FROM canon_event
           WHERE world_id='11111111-1111-1111-1111-111111111111' AND in_world_tick >= 500)::int,
          2, 'both events committed');
-- SAY generated Mara 'told'; MOVE generated Player 'direct'
SELECT ok(EXISTS(SELECT 1 FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
           WHERE ce.in_world_tick>=500 AND pr.holder_id='bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
             AND pr.epistemic_type='told'),
          'Mara TOLD by the SAY step');
SELECT ok(EXISTS(SELECT 1 FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
           WHERE ce.in_world_tick>=500 AND pr.holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
             AND pr.epistemic_type='direct'),
          'Player DIRECT from the MOVE step');
-- POSITIVE round-trip (Check #1): the intended holder ACTUALLY sees her generated perception via the
-- read path — not just present in perception_record. Catches a generate/wall scope/column mismatch.
SELECT ok(EXISTS(SELECT 1 FROM fn_visible_perceptions('11111111-1111-1111-1111-111111111111',
                   'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb') v
           JOIN canon_event ce ON ce.event_id=v.source_event_id
           WHERE ce.in_world_tick>=500 AND v.content='I saw the note'),
          'Mara SEES her TOLD through fn_visible_perceptions (positive round-trip, not table-only)');
SELECT ok(fn_timeline('11111111-1111-1111-1111-111111111111',
                      'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')::text LIKE '%I saw the note%',
          'the TOLD perception surfaces in Mara''s Compendium timeline (fn_timeline read path)');
-- I-3 no-leak through the existing Compendium read function: Jonas sees none of it.
SELECT is((SELECT count(*) FROM fn_visible_perceptions('11111111-1111-1111-1111-111111111111',
                                                       'cccccccc-cccc-cccc-cccc-cccccccccccc') v
           JOIN canon_event ce ON ce.event_id=v.source_event_id WHERE ce.in_world_tick>=500)::int,
          0, 'I-3: Jonas perceives nothing from the beat (validated through fn_visible_perceptions)');
-- action-driven clock: say(0) + move tavern→square(5) → 2 events in distinct (tick,beat_seq) slots.
SELECT is((SELECT count(DISTINCT (in_world_tick, beat_seq)) FROM canon_event
           WHERE world_id='11111111-1111-1111-1111-111111111111' AND in_world_tick>=500)::int,
          2, 'both committed events occupy distinct (tick, beat_seq) slots (ADR-034)');

SELECT * FROM finish();
ROLLBACK;
