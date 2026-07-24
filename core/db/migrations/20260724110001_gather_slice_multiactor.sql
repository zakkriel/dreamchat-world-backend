-- migrate:up

-- Multi-actor slice: co_present becomes the DISTINCT union of actors at EVERY location
-- occupied by any actor in p_ids, excluding all of p_ids (they are already in 'entities').
-- The pre-fix body anchored co_present on p_ids[1] alone, so a multi-actor collision
-- (different actors at different locations) dropped every bystander outside the first
-- actor's room — legitimate rulings then bounced on whitelist gaps (the PR #26 blocker).
-- Only the co_present block changes; entities / relationships / recent_events are verbatim.

CREATE OR REPLACE FUNCTION public.gather_slice(
    p_world_id uuid,
    p_ids      uuid[]
)
RETURNS jsonb
LANGUAGE sql
STABLE
AS $$
  SELECT jsonb_build_object(
    'entities',      COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'id',   er.entity_id,
          'kind', er.entity_kind,
          'name', er.canonical_name,
          'attrs', COALESCE(
            CASE er.entity_kind
              WHEN 'actor'    THEN (SELECT ast.attrs FROM actor_state    ast WHERE ast.entity_id = er.entity_id)
              WHEN 'artifact' THEN (SELECT aft.attrs FROM artifact_state aft WHERE aft.entity_id = er.entity_id)
              WHEN 'location' THEN (SELECT lst.attrs FROM location_state lst WHERE lst.entity_id = er.entity_id)
              ELSE NULL
            END,
            '{}'::jsonb
          )
        )
      )
      FROM entity_registry er
      WHERE er.world_id  = p_world_id
        AND er.entity_id = ANY(p_ids)
    ), '[]'::jsonb),

    'relationships', COALESCE((
      SELECT jsonb_agg(to_jsonb(rs))
      FROM relationship_state rs
      WHERE rs.world_id = p_world_id
        AND (rs.a_id = ANY(p_ids) OR rs.b_id = ANY(p_ids))
    ), '[]'::jsonb),

    'recent_events', COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'event_id', sub.event_id,
          'type',     sub.event_type,
          'tick',     sub.in_world_tick,
          'seq',      sub.beat_seq,
          'summary',  sub.summary
        )
        ORDER BY sub.in_world_tick DESC, sub.beat_seq DESC
      )
      FROM (
        SELECT DISTINCT ON (ranked.event_id)
               ranked.event_id,
               ranked.event_type,
               ranked.in_world_tick,
               ranked.beat_seq,
               ranked.summary
        FROM (
          SELECT
            ce.event_id,
            ce.event_type,
            ce.in_world_tick,
            ce.beat_seq,
            ce.summary,
            ep.entity_id,
            ROW_NUMBER() OVER (
              PARTITION BY ep.entity_id
              ORDER BY ce.in_world_tick DESC, ce.beat_seq DESC
            ) AS rn
          FROM event_participant ep
          JOIN canon_event ce ON ce.event_id = ep.event_id
          WHERE ep.entity_id = ANY(p_ids)
            AND ce.world_id  = p_world_id
        ) ranked
        WHERE ranked.rn <= 10
        ORDER BY ranked.event_id, ranked.in_world_tick DESC, ranked.beat_seq DESC
      ) sub
    ), '[]'::jsonb),

    -- co_present: DISTINCT union of actors at EVERY location any actor in p_ids occupies,
    -- minus p_ids themselves (they already appear in 'entities').
    'co_present', COALESCE((
      SELECT jsonb_agg(DISTINCT jsonb_build_object('id', er.entity_id, 'name', er.canonical_name))
      FROM entity_registry er
      WHERE er.world_id = p_world_id
        AND er.entity_id IN (
          SELECT fa.entity_id
          FROM ( SELECT DISTINCT (ast.attrs->>'location_id')::uuid AS loc
                 FROM actor_state ast
                 WHERE ast.world_id = p_world_id
                   AND ast.entity_id = ANY(p_ids)
                   AND ast.attrs ? 'location_id' ) locs,
          LATERAL fn_actors_at(p_world_id, locs.loc) fa )
        AND NOT (er.entity_id = ANY(p_ids))
    ), '[]'::jsonb)
  )
$$;

-- migrate:down

-- Restore the previous single-anchor body verbatim (co_present keyed on p_ids[1]).
-- Copied from 20260724100001_gather_slice.sql.

CREATE OR REPLACE FUNCTION public.gather_slice(
    p_world_id uuid,
    p_ids      uuid[]
)
RETURNS jsonb
LANGUAGE sql
STABLE
AS $$
  SELECT jsonb_build_object(
    'entities',      COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'id',   er.entity_id,
          'kind', er.entity_kind,
          'name', er.canonical_name,
          'attrs', COALESCE(
            CASE er.entity_kind
              WHEN 'actor'    THEN (SELECT ast.attrs FROM actor_state    ast WHERE ast.entity_id = er.entity_id)
              WHEN 'artifact' THEN (SELECT aft.attrs FROM artifact_state aft WHERE aft.entity_id = er.entity_id)
              WHEN 'location' THEN (SELECT lst.attrs FROM location_state lst WHERE lst.entity_id = er.entity_id)
              ELSE NULL
            END,
            '{}'::jsonb
          )
        )
      )
      FROM entity_registry er
      WHERE er.world_id  = p_world_id
        AND er.entity_id = ANY(p_ids)
    ), '[]'::jsonb),

    'relationships', COALESCE((
      SELECT jsonb_agg(to_jsonb(rs))
      FROM relationship_state rs
      WHERE rs.world_id = p_world_id
        AND (rs.a_id = ANY(p_ids) OR rs.b_id = ANY(p_ids))
    ), '[]'::jsonb),

    'recent_events', COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'event_id', sub.event_id,
          'type',     sub.event_type,
          'tick',     sub.in_world_tick,
          'seq',      sub.beat_seq,
          'summary',  sub.summary
        )
        ORDER BY sub.in_world_tick DESC, sub.beat_seq DESC
      )
      FROM (
        SELECT DISTINCT ON (ranked.event_id)
               ranked.event_id,
               ranked.event_type,
               ranked.in_world_tick,
               ranked.beat_seq,
               ranked.summary
        FROM (
          SELECT
            ce.event_id,
            ce.event_type,
            ce.in_world_tick,
            ce.beat_seq,
            ce.summary,
            ep.entity_id,
            ROW_NUMBER() OVER (
              PARTITION BY ep.entity_id
              ORDER BY ce.in_world_tick DESC, ce.beat_seq DESC
            ) AS rn
          FROM event_participant ep
          JOIN canon_event ce ON ce.event_id = ep.event_id
          WHERE ep.entity_id = ANY(p_ids)
            AND ce.world_id  = p_world_id
        ) ranked
        WHERE ranked.rn <= 10
        ORDER BY ranked.event_id, ranked.in_world_tick DESC, ranked.beat_seq DESC
      ) sub
    ), '[]'::jsonb),

    'co_present',    COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'id',   er.entity_id,
          'name', er.canonical_name
        )
      )
      FROM entity_registry er
      WHERE er.world_id  = p_world_id
        AND er.entity_id IN (
          SELECT fa.entity_id
          FROM fn_actors_at(
            p_world_id,
            (
              SELECT (ast.attrs->>'location_id')::uuid
              FROM actor_state ast
              WHERE ast.entity_id = p_ids[1]
                AND ast.world_id  = p_world_id
            )
          ) fa
        )
        AND er.entity_id <> p_ids[1]
    ), '[]'::jsonb)
  )
$$;
