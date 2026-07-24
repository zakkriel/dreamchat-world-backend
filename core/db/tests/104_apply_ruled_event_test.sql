BEGIN;
SELECT plan(13);

-- Fixed UUIDs for test 104 (apply_ruled_event + apply_attribute_writes).
-- world:    e4000000-ffff-0000-0000-000000000000
-- actor A (ruled actor / instigator): e4000000-0000-0000-0000-000000000001
-- actor B (co-located observer):      e4000000-0000-0000-0000-000000000002
-- actor C (named receiver_variant):   e4000000-0000-0000-0000-000000000003
-- actor D (remote, not co-located):   e4000000-0000-0000-0000-000000000004
-- loc 1 (A, B, C are here):           e4000000-0000-0000-0000-000000000010
-- loc 2 (D is here, Communicated dst): e4000000-0000-0000-0000-000000000011
-- dest loc for ActorMoved:            e4000000-0000-0000-0000-000000000012
-- target entity for AttributeChanged: e4000000-0000-0000-0000-000000000020

-- ── Seed: entities and actor positions ───────────────────────────────────────
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('e4000000-0000-0000-0000-000000000001','e4000000-ffff-0000-0000-000000000000','actor','test-actor-A-104'),
 ('e4000000-0000-0000-0000-000000000002','e4000000-ffff-0000-0000-000000000000','actor','test-actor-B-104'),
 ('e4000000-0000-0000-0000-000000000003','e4000000-ffff-0000-0000-000000000000','actor','test-actor-C-104'),
 ('e4000000-0000-0000-0000-000000000004','e4000000-ffff-0000-0000-000000000000','actor','test-actor-D-104'),
 ('e4000000-0000-0000-0000-000000000010','e4000000-ffff-0000-0000-000000000000','location','test-loc1-104'),
 ('e4000000-0000-0000-0000-000000000011','e4000000-ffff-0000-0000-000000000000','location','test-loc2-104'),
 ('e4000000-0000-0000-0000-000000000012','e4000000-ffff-0000-0000-000000000000','location','test-dest-104'),
 ('e4000000-0000-0000-0000-000000000020','e4000000-ffff-0000-0000-000000000000','actor','test-target-104');

-- Position actors via setup mutations through canonical events.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e4000000-0000-0000-0000-000000000050','e4000000-ffff-0000-0000-000000000000','move','setup A→loc1',1000,0,'accepted',now(),'public','fast_path'),
 ('e4000000-0000-0000-0000-000000000051','e4000000-ffff-0000-0000-000000000000','move','setup B→loc1',1001,0,'accepted',now(),'public','fast_path'),
 ('e4000000-0000-0000-0000-000000000052','e4000000-ffff-0000-0000-000000000000','move','setup C→loc1',1002,0,'accepted',now(),'public','fast_path'),
 ('e4000000-0000-0000-0000-000000000053','e4000000-ffff-0000-0000-000000000000','move','setup D→loc2',1003,0,'accepted',now(),'public','fast_path');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('e4000000-ffff-0000-0000-000000000000','e4000000-0000-0000-0000-000000000050','e4000000-0000-0000-0000-000000000001','actor','attrs.location_id',to_jsonb('e4000000-0000-0000-0000-000000000010'::text),1000,0),
 ('e4000000-ffff-0000-0000-000000000000','e4000000-0000-0000-0000-000000000051','e4000000-0000-0000-0000-000000000002','actor','attrs.location_id',to_jsonb('e4000000-0000-0000-0000-000000000010'::text),1001,0),
 ('e4000000-ffff-0000-0000-000000000000','e4000000-0000-0000-0000-000000000052','e4000000-0000-0000-0000-000000000003','actor','attrs.location_id',to_jsonb('e4000000-0000-0000-0000-000000000010'::text),1002,0),
 ('e4000000-ffff-0000-0000-000000000000','e4000000-0000-0000-0000-000000000053','e4000000-0000-0000-0000-000000000004','actor','attrs.location_id',to_jsonb('e4000000-0000-0000-0000-000000000011'::text),1003,0);

-- ──────────────────────────────────────────────────────────────────────────────
-- (a) Ruled AttributeChanged commits; canon summary = truth text (not appearance).
-- Actor A ruled: truth="Mara hardens inside", appearance="Mara seems unmoved".
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  (SELECT (apply_ruled_event(
    'e4000000-ffff-0000-0000-000000000000',
    jsonb_build_object(
      'type',       'AttributeChanged',
      'actor_id',   'e4000000-0000-0000-0000-000000000001',
      'truth',      'Mara hardens inside',
      'appearance', 'Mara seems unmoved',
      'target_id',  'e4000000-0000-0000-0000-000000000020'
    ),
    2000, 0, 'ruling'
  ))->>'halt_reason'),
  'committed',
  '(a) ruled AttributeChanged returns halt_reason=committed'
);

SELECT is(
  (SELECT summary FROM canon_event
   WHERE world_id='e4000000-ffff-0000-0000-000000000000'
     AND in_world_tick=2000
     AND origin='ruling'),
  'Mara hardens inside',
  '(a) canon summary = truth (CANON NEVER LIES — appearance is NOT stored in canon)'
);

-- ──────────────────────────────────────────────────────────────────────────────
-- (b) Deception: co-located observer B gets appearance text, not truth.
-- We reuse the event committed in (a) at tick 2000.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  (SELECT pr.content FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id
   WHERE ce.world_id = 'e4000000-ffff-0000-0000-000000000000'
     AND ce.in_world_tick = 2000
     AND pr.holder_id = 'e4000000-0000-0000-0000-000000000002'),
  'Mara seems unmoved',
  '(b) co-located observer B perceives appearance text, not truth'
);

-- ──────────────────────────────────────────────────────────────────────────────
-- (c) Named receiver_variant: actor C gets variant text instead of appearance.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  (SELECT (apply_ruled_event(
    'e4000000-ffff-0000-0000-000000000000',
    jsonb_build_object(
      'type',       'AttributeChanged',
      'actor_id',   'e4000000-0000-0000-0000-000000000001',
      'truth',      'The door is locked',
      'appearance', 'The door looks stuck',
      'target_id',  'e4000000-0000-0000-0000-000000000020',
      'receiver_variants', jsonb_build_array(
        jsonb_build_object(
          'receiver_id', 'e4000000-0000-0000-0000-000000000003',
          'text',        'You see the key-hole is blocked from inside'
        )
      )
    ),
    2001, 0, 'ruling'
  ))->>'halt_reason'),
  'committed',
  '(c) ruled event with receiver_variant commits'
);

SELECT is(
  (SELECT pr.content FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id
   WHERE ce.world_id = 'e4000000-ffff-0000-0000-000000000000'
     AND ce.in_world_tick = 2001
     AND pr.holder_id = 'e4000000-0000-0000-0000-000000000003'),
  'You see the key-hole is blocked from inside',
  '(c) named receiver C gets variant text, not appearance'
);

-- ──────────────────────────────────────────────────────────────────────────────
-- (d) visible:false → zero perception rows generated.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  (WITH result AS (
    SELECT apply_ruled_event(
      'e4000000-ffff-0000-0000-000000000000',
      jsonb_build_object(
        'type',     'AttributeChanged',
        'actor_id', 'e4000000-0000-0000-0000-000000000001',
        'truth',    'Hidden change happens',
        'visible',  false,
        'target_id','e4000000-0000-0000-0000-000000000020'
      ),
      2002, 0, 'ruling'
    ) AS r
  )
  SELECT count(*)::int FROM perception_record pr
  JOIN canon_event ce ON ce.event_id = pr.source_event_id
  WHERE ce.world_id = 'e4000000-ffff-0000-0000-000000000000'
    AND ce.in_world_tick = 2002),
  0,
  '(d) visible:false → zero perception rows'
);

-- ──────────────────────────────────────────────────────────────────────────────
-- (e) Every perception has ≥1 perception_subject row (about-ness, engine-written).
-- Reuse the event at tick 2000: actor A is participant → every perception has a ps row.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT ok(
  (SELECT min(sub_count) >= 1
   FROM (
     SELECT count(ps.entity_id)::int AS sub_count
     FROM perception_record pr
     JOIN canon_event ce ON ce.event_id = pr.source_event_id
     LEFT JOIN perception_subject ps ON ps.perception_id = pr.perception_id
     WHERE ce.world_id = 'e4000000-ffff-0000-0000-000000000000'
       AND ce.in_world_tick = 2000
     GROUP BY pr.perception_id
   ) t),
  '(e) every perception from ruled event has ≥1 perception_subject row'
);

-- ──────────────────────────────────────────────────────────────────────────────
-- (f) Ruled ActorMoved commits + actor_state.attrs.location_id updated via trigger.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  (SELECT (apply_ruled_event(
    'e4000000-ffff-0000-0000-000000000000',
    jsonb_build_object(
      'type',           'ActorMoved',
      'actor_id',       'e4000000-0000-0000-0000-000000000001',
      'truth',          'Mara steps through the door',
      'to_location_id', 'e4000000-0000-0000-0000-000000000012'
    ),
    2003, 0, 'ruling'
  ))->>'halt_reason'),
  'committed',
  '(f) ruled ActorMoved returns committed'
);

SELECT is(
  (SELECT attrs->>'location_id' FROM actor_state
   WHERE world_id='e4000000-ffff-0000-0000-000000000000'
     AND entity_id='e4000000-0000-0000-0000-000000000001'),
  'e4000000-0000-0000-0000-000000000012',
  '(f) actor_state.attrs.location_id updated via state_mutation trigger after ruled ActorMoved'
);

-- ──────────────────────────────────────────────────────────────────────────────
-- (g) Floor reject: Communicated to non-co-located listener → gate_reject, nothing written.
-- D is at loc2; A is at loc1 (after move in (f) A is at dest, but D is still at loc2).
-- Use a fresh actor at loc1 as speaker to ensure clean floor test.
-- We'll use actor B (still at loc1) as speaker, D (at loc2) as listener.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  (SELECT (apply_ruled_event(
    'e4000000-ffff-0000-0000-000000000000',
    jsonb_build_object(
      'type',        'Communicated',
      'actor_id',    'e4000000-0000-0000-0000-000000000002',
      'truth',       'secret message',
      'listener_id', 'e4000000-0000-0000-0000-000000000004'
    ),
    2004, 0, 'ruling'
  ))->>'halt_reason'),
  'gate_reject',
  '(g) Communicated to non-co-located listener returns gate_reject'
);

SELECT is(
  (SELECT count(*)::int FROM canon_event
   WHERE world_id='e4000000-ffff-0000-0000-000000000000'
     AND in_world_tick=2004),
  0,
  '(g) gate_reject: nothing written to canon_event'
);

-- ──────────────────────────────────────────────────────────────────────────────
-- (h) apply_attribute_writes: writes state_mutation; projection trigger updates actor attrs.
-- Write a custom attribute on actor B at this tick.
-- We need a provenance event — use the AttributeChanged event committed at tick 2000.
-- ──────────────────────────────────────────────────────────────────────────────
SELECT is(
  (WITH prov_event AS (
    SELECT event_id FROM canon_event
    WHERE world_id='e4000000-ffff-0000-0000-000000000000'
      AND in_world_tick=2000
    LIMIT 1
  )
  SELECT apply_attribute_writes(
    'e4000000-ffff-0000-0000-000000000000',
    jsonb_build_array(
      jsonb_build_object(
        'target_id', 'e4000000-0000-0000-0000-000000000002',
        'attribute', 'courage',
        'value',     to_jsonb(7),
        'tier',      'Tier2'
      )
    ),
    (SELECT event_id FROM prov_event),
    2000, 1
  )),
  1,
  '(h) apply_attribute_writes returns 1 row written'
);

SELECT is(
  (SELECT attrs->'courage' FROM actor_state
   WHERE world_id='e4000000-ffff-0000-0000-000000000000'
     AND entity_id='e4000000-0000-0000-0000-000000000002'),
  to_jsonb(7),
  '(h) projection trigger applied attribute write: attrs.courage = 7'
);

SELECT * FROM finish();
ROLLBACK;
