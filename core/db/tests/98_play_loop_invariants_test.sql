BEGIN;
SELECT plan(5);

-- Position the cast and run a happy beat that produces both a move (mutation → projection) and a say.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000060','11111111-1111-1111-1111-111111111111','move','P→tavern',1000,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000061','11111111-1111-1111-1111-111111111111','move','M→tavern',1001,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000060','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000061','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000060','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),1000,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000061','bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb','actor','attrs.location_id', to_jsonb('tavern'::text),1001,0);

SELECT apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  '[{"type":"say","listener":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","content":"the note"},
    {"type":"move","to":"square"}]'::jsonb, 1100, 100, 'fast_path');

-- I-1: replay rebuilds PROJECTIONS to the same domain state (the beat's move is replayed). Perceptions
-- are durable canon (doc 13 §6), not rebuilt here — their reproducibility is test 97's determinism.
SELECT ok(replay_0A(), 'I-1: replay is domain-equivalent after a generated beat (ADR-026)');
-- I-2: every generated perception references an accepted event (zero orphans).
SELECT is((SELECT count(*) FROM perception_record pr
           LEFT JOIN canon_event ce ON ce.event_id=pr.source_event_id AND ce.status='accepted'
           WHERE pr.source_event_id IN (SELECT event_id FROM canon_event WHERE in_world_tick>=1100)
             AND ce.event_id IS NULL)::int,
          0, 'I-2: no orphan generated perceptions');
-- I-3: Jonas (uninvolved) perceives nothing from the beat (the wall).
SELECT is((SELECT count(*) FROM fn_visible_perceptions('11111111-1111-1111-1111-111111111111',
                                                       'cccccccc-cccc-cccc-cccc-cccccccccccc') v
           JOIN canon_event ce ON ce.event_id=v.source_event_id WHERE ce.in_world_tick>=1100)::int,
          0, 'I-3: no hidden-canon leakage to Jonas');
-- I-9: acquired_tick >= source event in_world_tick for every generated perception.
SELECT is((SELECT count(*) FROM perception_record pr JOIN canon_event ce ON ce.event_id=pr.source_event_id
           WHERE ce.in_world_tick>=1100 AND pr.acquired_tick < ce.in_world_tick)::int,
          0, 'I-9: no perception acquired before its event happened');
-- the move projection is present (Player at square after the beat)
SELECT is((SELECT attrs->>'location_id' FROM actor_state WHERE entity_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
          'square', 'projection reflects the committed move');

SELECT * FROM finish();
ROLLBACK;
