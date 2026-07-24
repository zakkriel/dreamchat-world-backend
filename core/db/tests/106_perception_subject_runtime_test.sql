BEGIN;
SELECT plan(4);

-- generate_perceptions must write perception_subject rows on EVERY branch, not just the
-- discovery-on-arrival loop (ledger flag; RULINGS-2026-07-23 §6). Fresh, self-contained
-- fixture (f1060000-prefixed UUIDs), ROLLBACK at end.
--   world:       f1060000-ffff-0000-0000-000000000000
--   actor S (speaker, at room):    f1060000-0000-0000-0000-000000000001
--   actor L (listener, at room):   f1060000-0000-0000-0000-000000000002
--   actor M (mover, elsewhere then arrives at room): f1060000-0000-0000-0000-000000000003
--   loc room:     f1060000-0000-0000-0000-000000000010
--   loc elsewhere:f1060000-0000-0000-0000-000000000011

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('f1060000-0000-0000-0000-000000000001','f1060000-ffff-0000-0000-000000000000','actor','test-S-106'),
 ('f1060000-0000-0000-0000-000000000002','f1060000-ffff-0000-0000-000000000000','actor','test-L-106'),
 ('f1060000-0000-0000-0000-000000000003','f1060000-ffff-0000-0000-000000000000','actor','test-M-106'),
 ('f1060000-0000-0000-0000-000000000010','f1060000-ffff-0000-0000-000000000000','location','test-room-106'),
 ('f1060000-0000-0000-0000-000000000011','f1060000-ffff-0000-0000-000000000000','location','test-elsewhere-106');

-- Position S, L at the room; M elsewhere (setup via canonical move events + mutations).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('f1060000-0000-0000-0000-000000000050','f1060000-ffff-0000-0000-000000000000','move','setup S->room',1000,0,'accepted',now(),'public','fast_path'),
 ('f1060000-0000-0000-0000-000000000051','f1060000-ffff-0000-0000-000000000000','move','setup L->room',1001,0,'accepted',now(),'public','fast_path'),
 ('f1060000-0000-0000-0000-000000000052','f1060000-ffff-0000-0000-000000000000','move','setup M->elsewhere',1002,0,'accepted',now(),'public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('f1060000-ffff-0000-0000-000000000000','f1060000-0000-0000-0000-000000000050','f1060000-0000-0000-0000-000000000001','actor','attrs.location_id',to_jsonb('f1060000-0000-0000-0000-000000000010'::text),1000,0),
 ('f1060000-ffff-0000-0000-000000000000','f1060000-0000-0000-0000-000000000051','f1060000-0000-0000-0000-000000000002','actor','attrs.location_id',to_jsonb('f1060000-0000-0000-0000-000000000010'::text),1001,0),
 ('f1060000-ffff-0000-0000-000000000000','f1060000-0000-0000-0000-000000000052','f1060000-0000-0000-0000-000000000003','actor','attrs.location_id',to_jsonb('f1060000-0000-0000-0000-000000000011'::text),1002,0);

-- (a) Communicated S→L: BOTH resulting perceptions (speaker 'shared', listener 'told') must carry
-- subject rows for S and L (about-ness = the source event's participants). 2 perceptions x 2
-- subjects = 4 rows.
SELECT apply_event(
  'f1060000-ffff-0000-0000-000000000000',
  'f1060000-0000-0000-0000-000000000001',
  '{"type":"Communicated","stated":"S tells L something","listener_id":"f1060000-0000-0000-0000-000000000002","content":"S tells L something"}'::jsonb,
  2000, 0, 'freeform'
);
SELECT is(
  (SELECT count(*)::int FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id
   JOIN perception_subject ps ON ps.perception_id = pr.perception_id
   WHERE ce.world_id = 'f1060000-ffff-0000-0000-000000000000'
     AND ce.event_type = 'Communicated' AND ce.in_world_tick = 2000
     AND ps.entity_id IN ('f1060000-0000-0000-0000-000000000001','f1060000-0000-0000-0000-000000000002')),
  4,
  '(a) both Communicated perceptions (speaker shared + listener told) carry subject rows for S and L'
);

-- (b) ActorMoved M into the room (S, L already present): the mover-direct perception (M's own-move,
-- content = the stated summary, NOT the discovery boilerplate) must carry a subject row for M.
SELECT apply_event(
  'f1060000-ffff-0000-0000-000000000000',
  'f1060000-0000-0000-0000-000000000003',
  '{"type":"ActorMoved","stated":"M moves into the room","to_location_id":"f1060000-0000-0000-0000-000000000010"}'::jsonb,
  2001, 0, 'freeform'
);
SELECT ok(
  EXISTS(
    SELECT 1 FROM perception_record pr
    JOIN canon_event ce ON ce.event_id = pr.source_event_id
    JOIN perception_subject ps ON ps.perception_id = pr.perception_id
    WHERE ce.world_id = 'f1060000-ffff-0000-0000-000000000000'
      AND ce.event_type = 'ActorMoved' AND ce.in_world_tick = 2001
      AND pr.content = 'M moves into the room'
      AND ps.entity_id = 'f1060000-0000-0000-0000-000000000003'
  ),
  '(b) mover-direct perception (M''s own-move) carries a subject row for M'
);

-- (c) discovery perceptions (untouched loop) keep their existing noticed-actor subject: M's arrival
-- discovers BOTH S and L already present → 2 discovery rows, subjects S and L respectively.
SELECT is(
  (SELECT count(*)::int FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id
   JOIN perception_subject ps ON ps.perception_id = pr.perception_id
   WHERE ce.world_id = 'f1060000-ffff-0000-0000-000000000000'
     AND ce.event_type = 'ActorMoved' AND ce.in_world_tick = 2001
     AND pr.content = 'On arriving, I noticed someone already here.'
     AND ps.entity_id IN ('f1060000-0000-0000-0000-000000000001','f1060000-0000-0000-0000-000000000002')),
  2,
  '(c) discovery perceptions keep their noticed-actor subject (S and L both discovered)'
);

-- (d) orphan-free proof: zero perception_records in this world lack a subject row.
SELECT is(
  (SELECT count(*) FROM perception_record pr
   WHERE pr.world_id = 'f1060000-ffff-0000-0000-000000000000'
     AND NOT EXISTS (SELECT 1 FROM perception_subject ps WHERE ps.perception_id = pr.perception_id))::int,
  0,
  '(d) zero perception_records in this world lack a subject row (orphan-free)'
);

SELECT * FROM finish();
ROLLBACK;
