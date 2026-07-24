BEGIN;
SELECT plan(4);

-- Multi-actor gather_slice: co_present must be the DISTINCT union of actors at
-- EVERY location occupied by any actor in p_ids, excluding all of p_ids.
-- The pre-fix body anchored co_present on p_ids[1] only, so a second actor at a
-- different location contributed NO bystanders — the PR #26 whitelist gap.
--
-- Fixture (fresh a5-prefixed UUIDs, self-contained, ROLLBACK at end):
--   world:      a5000000-0000-0000-0000-000000000000
--   actor A:    a5000000-0000-0000-0000-000000000001  (in p_ids; at L1)
--   bystander B1:a5000000-0000-0000-0000-000000000002 (co-located with A at L1)
--   actor C:    a5000000-0000-0000-0000-000000000003  (in p_ids; at L2)
--   bystander B2:a5000000-0000-0000-0000-000000000004 (co-located with C at L2)
--   loc L1:     a5000000-0000-0000-0000-000000000010
--   loc L2:     a5000000-0000-0000-0000-000000000020

-- ── Fixture: entity_registry ─────────────────────────────────────────────────

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('a5000000-0000-0000-0000-000000000001', 'a5000000-0000-0000-0000-000000000000', 'actor',    'multiactor-A-105'),
  ('a5000000-0000-0000-0000-000000000002', 'a5000000-0000-0000-0000-000000000000', 'actor',    'multiactor-B1-105'),
  ('a5000000-0000-0000-0000-000000000003', 'a5000000-0000-0000-0000-000000000000', 'actor',    'multiactor-C-105'),
  ('a5000000-0000-0000-0000-000000000004', 'a5000000-0000-0000-0000-000000000000', 'actor',    'multiactor-B2-105'),
  ('a5000000-0000-0000-0000-000000000010', 'a5000000-0000-0000-0000-000000000000', 'location', 'multiactor-L1-105'),
  ('a5000000-0000-0000-0000-000000000020', 'a5000000-0000-0000-0000-000000000000', 'location', 'multiactor-L2-105');

-- ── Fixture: actor_state (A,B1 at L1; C,B2 at L2) ────────────────────────────

INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('a5000000-0000-0000-0000-000000000001', 'a5000000-0000-0000-0000-000000000000',
   jsonb_build_object('location_id', 'a5000000-0000-0000-0000-000000000010')),
  ('a5000000-0000-0000-0000-000000000002', 'a5000000-0000-0000-0000-000000000000',
   jsonb_build_object('location_id', 'a5000000-0000-0000-0000-000000000010')),
  ('a5000000-0000-0000-0000-000000000003', 'a5000000-0000-0000-0000-000000000000',
   jsonb_build_object('location_id', 'a5000000-0000-0000-0000-000000000020')),
  ('a5000000-0000-0000-0000-000000000004', 'a5000000-0000-0000-0000-000000000000',
   jsonb_build_object('location_id', 'a5000000-0000-0000-0000-000000000020'));

-- (a) co_present contains B1 (bystander at A's location) ──────────────────────

SELECT ok(
  (
    SELECT result->'co_present' @> jsonb_build_array(jsonb_build_object('id', 'a5000000-0000-0000-0000-000000000002'))
    FROM (
      SELECT gather_slice(
        'a5000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a5000000-0000-0000-0000-000000000001'::uuid,
          'a5000000-0000-0000-0000-000000000003'::uuid
        ]
      ) AS result
    ) s
  ),
  'two-actor co_present includes B1 (bystander at actor A''s location)'
);

-- (b) co_present contains B2 (bystander at C's location — the union old code missed)

SELECT ok(
  (
    SELECT result->'co_present' @> jsonb_build_array(jsonb_build_object('id', 'a5000000-0000-0000-0000-000000000004'))
    FROM (
      SELECT gather_slice(
        'a5000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a5000000-0000-0000-0000-000000000001'::uuid,
          'a5000000-0000-0000-0000-000000000003'::uuid
        ]
      ) AS result
    ) s
  ),
  'two-actor co_present includes B2 (bystander at actor C''s location — the union old code missed)'
);

-- (c) co_present contains NEITHER A nor C (they belong in entities, not co_present)

SELECT ok(
  (
    SELECT
      NOT (result->'co_present' @> jsonb_build_array(jsonb_build_object('id', 'a5000000-0000-0000-0000-000000000001')))
      AND
      NOT (result->'co_present' @> jsonb_build_array(jsonb_build_object('id', 'a5000000-0000-0000-0000-000000000003')))
    FROM (
      SELECT gather_slice(
        'a5000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a5000000-0000-0000-0000-000000000001'::uuid,
          'a5000000-0000-0000-0000-000000000003'::uuid
        ]
      ) AS result
    ) s
  ),
  'two-actor co_present excludes both actors A and C'
);

-- (d) single-actor gather_slice(w, ARRAY[A]) still returns B1 only (unchanged) ─

SELECT ok(
  (
    SELECT
      jsonb_array_length(result->'co_present') = 1
      AND (result->'co_present' @> jsonb_build_array(jsonb_build_object('id', 'a5000000-0000-0000-0000-000000000002')))
    FROM (
      SELECT gather_slice(
        'a5000000-0000-0000-0000-000000000000'::uuid,
        ARRAY[
          'a5000000-0000-0000-0000-000000000001'::uuid
        ]
      ) AS result
    ) s
  ),
  'single-actor co_present is exactly [B1] (single-actor behavior unchanged)'
);

SELECT * FROM finish();
ROLLBACK;
