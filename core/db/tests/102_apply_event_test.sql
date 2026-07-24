BEGIN;
SELECT plan(18);

-- Fixed UUIDs for test 102 (apply_event).
-- world:    e6000000-ffff-0000-0000-000000000000
-- actor A (speaker/instigator): e6000000-0000-0000-0000-000000000001
-- actor B (listener, co-located): e6000000-0000-0000-0000-000000000002
-- actor C (elsewhere):           e6000000-0000-0000-0000-000000000003
-- loc 1 (shared):                e6000000-0000-0000-0000-000000000010
-- loc 2 (actor C is here):       e6000000-0000-0000-0000-000000000011
-- target entity for AttributeChanged: e6000000-0000-0000-0000-000000000020
-- destination location for ActorMoved: e6000000-0000-0000-0000-000000000012

-- ── Seed: entities and actor positions ───────────────────────────────────────
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('e6000000-0000-0000-0000-000000000001','e6000000-ffff-0000-0000-000000000000','actor','test-actor-A-102'),
 ('e6000000-0000-0000-0000-000000000002','e6000000-ffff-0000-0000-000000000000','actor','test-actor-B-102'),
 ('e6000000-0000-0000-0000-000000000003','e6000000-ffff-0000-0000-000000000000','actor','test-actor-C-102'),
 ('e6000000-0000-0000-0000-000000000010','e6000000-ffff-0000-0000-000000000000','location','test-loc1-102'),
 ('e6000000-0000-0000-0000-000000000011','e6000000-ffff-0000-0000-000000000000','location','test-loc2-102'),
 ('e6000000-0000-0000-0000-000000000012','e6000000-ffff-0000-0000-000000000000','location','test-dest-102'),
 ('e6000000-0000-0000-0000-000000000020','e6000000-ffff-0000-0000-000000000000','actor','test-target-102');

-- Position A and B at loc1, C at loc2 (setup via canonical move events + mutations).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e6000000-0000-0000-0000-000000000050','e6000000-ffff-0000-0000-000000000000','move','setup A→loc1',1000,0,'accepted',now(),'public','fast_path'),
 ('e6000000-0000-0000-0000-000000000051','e6000000-ffff-0000-0000-000000000000','move','setup B→loc1',1001,0,'accepted',now(),'public','fast_path'),
 ('e6000000-0000-0000-0000-000000000052','e6000000-ffff-0000-0000-000000000000','move','setup C→loc2',1002,0,'accepted',now(),'public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('e6000000-ffff-0000-0000-000000000000','e6000000-0000-0000-0000-000000000050','e6000000-0000-0000-0000-000000000001','actor','attrs.location_id',to_jsonb('e6000000-0000-0000-0000-000000000010'::text),1000,0),
 ('e6000000-ffff-0000-0000-000000000000','e6000000-0000-0000-0000-000000000051','e6000000-0000-0000-0000-000000000002','actor','attrs.location_id',to_jsonb('e6000000-0000-0000-0000-000000000010'::text),1001,0),
 ('e6000000-ffff-0000-0000-000000000000','e6000000-0000-0000-0000-000000000052','e6000000-0000-0000-0000-000000000003','actor','attrs.location_id',to_jsonb('e6000000-0000-0000-0000-000000000011'::text),1002,0);

-- ─── (a) Valid Communicated between two co-located actors ─────────────────────
-- A and B are both at loc1. Should commit: event_type='Communicated', speaker+listener, 2 perceptions.
SELECT is(
  (SELECT (apply_event(
      'e6000000-ffff-0000-0000-000000000000',
      'e6000000-0000-0000-0000-000000000001',
      '{"type":"Communicated","stated":"hello","listener_id":"e6000000-0000-0000-0000-000000000002","content":"hello"}'::jsonb,
      2000, 0, 'freeform'
  ))->>'halt_reason'),
  'committed',
  '(a) co-located Communicated returns halt_reason=committed'
);

SELECT is(
  (SELECT event_type FROM canon_event
   WHERE world_id='e6000000-ffff-0000-0000-000000000000'
     AND event_type='Communicated'
     AND in_world_tick=2000),
  'Communicated',
  '(a) canon_event row has event_type=Communicated'
);

SELECT is(
  (SELECT count(*)::int FROM event_participant ep
   JOIN canon_event ce ON ce.event_id=ep.event_id
   WHERE ce.world_id='e6000000-ffff-0000-0000-000000000000'
     AND ce.event_type='Communicated'
     AND ce.in_world_tick=2000
     AND ep.role_qualifier IN ('speaker','listener')),
  2,
  '(a) two participants: speaker + listener'
);

SELECT is(
  (SELECT ep.role_qualifier FROM event_participant ep
   JOIN canon_event ce ON ce.event_id=ep.event_id
   WHERE ce.world_id='e6000000-ffff-0000-0000-000000000000'
     AND ce.event_type='Communicated'
     AND ce.in_world_tick=2000
     AND ep.entity_id='e6000000-0000-0000-0000-000000000001'),
  'speaker',
  '(a) actor A is the speaker'
);

SELECT is(
  (SELECT ep.role_qualifier FROM event_participant ep
   JOIN canon_event ce ON ce.event_id=ep.event_id
   WHERE ce.world_id='e6000000-ffff-0000-0000-000000000000'
     AND ce.event_type='Communicated'
     AND ce.in_world_tick=2000
     AND ep.entity_id='e6000000-0000-0000-0000-000000000002'),
  'listener',
  '(a) actor B is the listener'
);

SELECT is(
  (SELECT count(*)::int FROM perception_record pr
   JOIN canon_event ce ON ce.event_id=pr.source_event_id
   WHERE ce.world_id='e6000000-ffff-0000-0000-000000000000'
     AND ce.event_type='Communicated'
     AND ce.in_world_tick=2000),
  2,
  '(a) Communicated generates exactly 2 perception rows'
);

-- ─── (b) Communicated to an actor NOT co-located → gate_reject, nothing written ─
-- A is at loc1, C is at loc2. Floor blocks.
SELECT is(
  (SELECT (apply_event(
      'e6000000-ffff-0000-0000-000000000000',
      'e6000000-0000-0000-0000-000000000001',
      '{"type":"Communicated","stated":"hi","listener_id":"e6000000-0000-0000-0000-000000000003","content":"hi"}'::jsonb,
      2001, 0, 'freeform'
  ))->>'halt_reason'),
  'gate_reject',
  '(b) Communicated to non-co-located actor returns gate_reject'
);

SELECT is(
  (SELECT count(*)::int FROM canon_event
   WHERE world_id='e6000000-ffff-0000-0000-000000000000'
     AND in_world_tick=2001),
  0,
  '(b) gate_reject: nothing written to canon_event (blocker-only proof)'
);

-- ─── (c) AttributeChanged on an existing target → committed, instigator participant ─
SELECT is(
  (SELECT (apply_event(
      'e6000000-ffff-0000-0000-000000000000',
      'e6000000-0000-0000-0000-000000000001',
      '{"type":"AttributeChanged","stated":"courage updated","target_id":"e6000000-0000-0000-0000-000000000020"}'::jsonb,
      2002, 0, 'freeform'
  ))->>'halt_reason'),
  'committed',
  '(c) AttributeChanged on existing target returns committed'
);

SELECT is(
  (SELECT event_type FROM canon_event
   WHERE world_id='e6000000-ffff-0000-0000-000000000000'
     AND in_world_tick=2002),
  'AttributeChanged',
  '(c) canon_event has event_type=AttributeChanged'
);

SELECT is(
  (SELECT ep.role_qualifier FROM event_participant ep
   JOIN canon_event ce ON ce.event_id=ep.event_id
   WHERE ce.world_id='e6000000-ffff-0000-0000-000000000000'
     AND ce.in_world_tick=2002
     AND ep.entity_id='e6000000-0000-0000-0000-000000000001'),
  'instigator',
  '(c) actor A is the instigator for AttributeChanged'
);

-- ─── (d) ActorMoved to a registry location → committed + actor_state updated ────
SELECT is(
  (SELECT (apply_event(
      'e6000000-ffff-0000-0000-000000000000',
      'e6000000-0000-0000-0000-000000000001',
      '{"type":"ActorMoved","stated":"moved to dest","to_location_id":"e6000000-0000-0000-0000-000000000012"}'::jsonb,
      2003, 0, 'freeform'
  ))->>'halt_reason'),
  'committed',
  '(d) ActorMoved to registry location returns committed'
);

SELECT is(
  (SELECT event_type FROM canon_event
   WHERE world_id='e6000000-ffff-0000-0000-000000000000'
     AND in_world_tick=2003),
  'ActorMoved',
  '(d) canon_event has event_type=ActorMoved'
);

SELECT is(
  (SELECT (attrs->>'location_id')::text FROM actor_state
   WHERE world_id='e6000000-ffff-0000-0000-000000000000'
     AND entity_id='e6000000-0000-0000-0000-000000000001'),
  'e6000000-0000-0000-0000-000000000012',
  '(d) actor_state.attrs.location_id updated via state_mutation trigger after ActorMoved'
);

-- ─── (e) Unknown actor uuid → gate_reject ───────────────────────────────────
SELECT is(
  (SELECT (apply_event(
      'e6000000-ffff-0000-0000-000000000000',
      'deadbeef-dead-dead-dead-deaddeadbeef',
      '{"type":"Communicated","stated":"ghost","listener_id":"e6000000-0000-0000-0000-000000000002","content":"boo"}'::jsonb,
      2004, 0, 'freeform'
  ))->>'halt_reason'),
  'gate_reject',
  '(e) unknown actor uuid returns gate_reject'
);

SELECT is(
  (SELECT count(*)::int FROM canon_event
   WHERE world_id='e6000000-ffff-0000-0000-000000000000'
     AND in_world_tick=2004),
  0,
  '(e) gate_reject for unknown actor: nothing written'
);

-- ─── Verify event_id is non-null on committed, null on rejected ───────────────
SELECT ok(
  (SELECT r->'event_id' != 'null'::jsonb FROM apply_event(
      'e6000000-ffff-0000-0000-000000000000',
      'e6000000-0000-0000-0000-000000000001',
      '{"type":"AttributeChanged","stated":"bonus","target_id":"e6000000-0000-0000-0000-000000000020"}'::jsonb,
      2005, 0, 'freeform'
  ) r),
  'committed result carries a non-null event_id'
);

SELECT is(
  (SELECT r->'event_id' FROM apply_event(
      'e6000000-ffff-0000-0000-000000000000',
      'deadbeef-dead-dead-dead-deaddeadbeef',
      '{"type":"AttributeChanged","stated":"nope","target_id":"e6000000-0000-0000-0000-000000000020"}'::jsonb,
      2006, 0, 'freeform'
  ) r),
  'null'::jsonb,
  'gate_reject result carries null event_id'
);

SELECT * FROM finish();
ROLLBACK;
