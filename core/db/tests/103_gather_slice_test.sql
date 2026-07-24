BEGIN;
SELECT plan(5);

-- Fixed UUIDs for test 103
-- world:    a3000000-0000-0000-0000-000000000000
-- actor A:  a3000000-0000-0000-0000-000000000001  (first in p_ids; located at loc-1)
-- actor B:  a3000000-0000-0000-0000-000000000002  (co-located with A at loc-1)
-- artifact: a3000000-0000-0000-0000-000000000003
-- loc-1:    a3000000-0000-0000-0000-000000000010
-- rel:      A↔B in relationship_state
-- events:   a3000000-0000-0000-0000-000000000101/102/103  (ticks 300/301/302)
-- unknown:  a3000000-0000-0000-0000-000000009999

-- ── Fixture: entity_registry ─────────────────────────────────────────────────

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('a3000000-0000-0000-0000-000000000001', 'a3000000-0000-0000-0000-000000000000', 'actor',    'test-actor-A-103'),
  ('a3000000-0000-0000-0000-000000000002', 'a3000000-0000-0000-0000-000000000000', 'actor',    'test-actor-B-103'),
  ('a3000000-0000-0000-0000-000000000003', 'a3000000-0000-0000-0000-000000000000', 'artifact', 'test-artifact-103'),
  ('a3000000-0000-0000-0000-000000000010', 'a3000000-0000-0000-0000-000000000000', 'location', 'test-loc-1-103');

-- ── Fixture: actor_state (place A and B at loc-1) ────────────────────────────

INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('a3000000-0000-0000-0000-000000000001', 'a3000000-0000-0000-0000-000000000000',
   jsonb_build_object('location_id', 'a3000000-0000-0000-0000-000000000010')),
  ('a3000000-0000-0000-0000-000000000002', 'a3000000-0000-0000-0000-000000000000',
   jsonb_build_object('location_id', 'a3000000-0000-0000-0000-000000000010'));

-- ── Fixture: artifact_state ──────────────────────────────────────────────────

INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
  ('a3000000-0000-0000-0000-000000000003', 'a3000000-0000-0000-0000-000000000000',
   '{"material":"gold"}'::jsonb);

-- ── Fixture: relationship_state (A↔B) ────────────────────────────────────────

INSERT INTO relationship_state (world_id, a_id, b_id, attrs) VALUES
  ('a3000000-0000-0000-0000-000000000000',
   'a3000000-0000-0000-0000-000000000001',
   'a3000000-0000-0000-0000-000000000002',
   '{"bond":"ally"}'::jsonb);

-- ── Fixture: canon_events (ticks 300/301/302) ────────────────────────────────

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES
  ('a3000000-0000-0000-0000-000000000101', 'a3000000-0000-0000-0000-000000000000',
   'observation', 'event-oldest', 300, 0, 'accepted', now(), 'public', 'fast_path'),
  ('a3000000-0000-0000-0000-000000000102', 'a3000000-0000-0000-0000-000000000000',
   'observation', 'event-middle', 301, 0, 'accepted', now(), 'public', 'fast_path'),
  ('a3000000-0000-0000-0000-000000000103', 'a3000000-0000-0000-0000-000000000000',
   'observation', 'event-newest', 302, 0, 'accepted', now(), 'public', 'fast_path');

-- ── Fixture: event_participant (actor A participates in all three events) ─────

INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
  ('a3000000-0000-0000-0000-000000000101', 'a3000000-0000-0000-0000-000000000001', 'actor', 'instigator'),
  ('a3000000-0000-0000-0000-000000000102', 'a3000000-0000-0000-0000-000000000001', 'actor', 'instigator'),
  ('a3000000-0000-0000-0000-000000000103', 'a3000000-0000-0000-0000-000000000001', 'actor', 'instigator');

-- ── Capture result for reuse ─────────────────────────────────────────────────

DO $$
BEGIN
  -- just ensure it doesn't throw
  PERFORM gather_slice(
    'a3000000-0000-0000-0000-000000000000'::uuid,
    ARRAY[
      'a3000000-0000-0000-0000-000000000001'::uuid,
      'a3000000-0000-0000-0000-000000000003'::uuid
    ]
  );
END $$;

-- (a) entities array: actor A + artifact both present with attrs ──────────────

SELECT ok(
  (
    SELECT
      (result->'entities' @> jsonb_build_array(jsonb_build_object('id', 'a3000000-0000-0000-0000-000000000001')))
      AND
      (result->'entities' @> jsonb_build_array(jsonb_build_object('id', 'a3000000-0000-0000-0000-000000000003')))
    FROM (
      SELECT gather_slice(
        'a3000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a3000000-0000-0000-0000-000000000001'::uuid,
          'a3000000-0000-0000-0000-000000000003'::uuid
        ]
      ) AS result
    ) s
  ),
  'entities contains both actor A and the artifact'
);

-- (b) relationships non-empty ─────────────────────────────────────────────────

SELECT ok(
  (
    SELECT jsonb_array_length(result->'relationships') > 0
    FROM (
      SELECT gather_slice(
        'a3000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a3000000-0000-0000-0000-000000000001'::uuid,
          'a3000000-0000-0000-0000-000000000003'::uuid
        ]
      ) AS result
    ) s
  ),
  'relationships array is non-empty (A↔B row present)'
);

-- (c) recent_events: three events, newest first (tick 302 first) ──────────────

SELECT ok(
  (
    SELECT
      jsonb_array_length(result->'recent_events') = 3
      AND (result->'recent_events'->0->>'event_id') = 'a3000000-0000-0000-0000-000000000103'
      AND (result->'recent_events'->2->>'event_id') = 'a3000000-0000-0000-0000-000000000101'
    FROM (
      SELECT gather_slice(
        'a3000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a3000000-0000-0000-0000-000000000001'::uuid,
          'a3000000-0000-0000-0000-000000000003'::uuid
        ]
      ) AS result
    ) s
  ),
  'recent_events has 3 items newest-first (tick 302 → 300)'
);

-- (d) co_present includes actor B (A is first in p_ids, located at loc-1) ─────

SELECT ok(
  (
    SELECT result->'co_present' @> jsonb_build_array(jsonb_build_object('id', 'a3000000-0000-0000-0000-000000000002'))
    FROM (
      SELECT gather_slice(
        'a3000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a3000000-0000-0000-0000-000000000001'::uuid,
          'a3000000-0000-0000-0000-000000000003'::uuid
        ]
      ) AS result
    ) s
  ),
  'co_present includes actor B who shares location with A'
);

-- (e) unknown id yields no entity row and no error ────────────────────────────

SELECT ok(
  (
    SELECT
      NOT (result->'entities' @> jsonb_build_array(jsonb_build_object('id', 'a3000000-0000-0000-0000-000000009999')))
      AND jsonb_typeof(result->'entities') = 'array'
    FROM (
      SELECT gather_slice(
        'a3000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a3000000-0000-0000-0000-000000009999'::uuid
        ]
      ) AS result
    ) s
  ),
  'unknown id yields no entity row and no error'
);

SELECT * FROM finish();
ROLLBACK;
