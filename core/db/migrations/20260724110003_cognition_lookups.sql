-- migrate:up

-- The mechanical cognition lookups (RULINGS-2026-07-23 §5-§6). These are the WALL's semantic
-- foundation: Task 4's cognition seats sit exactly one NPC per action in either the shared batch
-- (public moment only) or an isolated call (her secret rides alone). Which seat is a MECHANICAL
-- LOOKUP, not a judgment (§5): intersect the action's bound ids with the subject links of the NPC's
-- private records. No seat judges "relevance"; these functions are pure set operations.
--
-- Definition (plan flagged-decision 3, binding):
--   PUBLIC  = a source event EVERY id in p_present holds a perception of, rendered with the modal
--             (most common) content among present holders — deterministic tie-break via
--             mode() WITHIN GROUP (ORDER BY content) (lexicographically smallest of the tied-most).
--   PRIVATE = any perception record that FAILS that test: source event not perceived by all present,
--             OR the record's content differs from the modal content, OR source_event_id IS NULL.
--             The split is PER-RECORD keyed on (source_event_id, content): E_variant's modal face is
--             public while a holder's divergent read of that SAME event is his private record.
--
-- All three share one core (the held/shared CTE): held = the (source, holder, content) triples any
-- present id perceived; shared = the sources every present id perceived, tagged with their modal
-- content. A held record is private iff its (source_event_id, content) has no matching shared row
-- (NULL source never joins, so a null-source record is always private — "isolate MORE when in doubt").
--
-- Temporal validity (mirrors fn_visible_perceptions, schema.sql): every scan of perception_record
-- below — in the held CTE and in the outer per-record scans — excludes invalidated/expired rows.
-- Cognition reads CURRENT knowledge only; a stale copy must never flip the modal face. Without this
-- filter a stale invalidated copy of a private reading could inflate the modal vote and flip a
-- genuinely-private record to "public", putting its content in a shared prompt — the exact leak the
-- wall forbids.

-- Failure asymmetry (RULINGS-2026-07-23 §5): a missed flag makes the NPC dull for one action; an over-flag costs one extra call. The dangerous failure must stay structurally impossible.
-- Caller contract: p_present must be DISTINCT present ids and p_k >= 0. Duplicate ids or an empty
-- p_present degrade to everything-private (nobody meets the count(DISTINCT holder_id) = cardinality
-- bar, so nothing renders as public — the safe direction); a negative p_k errors (LIMIT rejects a
-- negative bound) rather than silently returning the wrong slice.
CREATE FUNCTION public.fn_public_moment(
    p_world_id uuid,
    p_present  uuid[],
    p_k        int
)
RETURNS TABLE(source_event_id uuid, acquired_tick bigint, content text)
LANGUAGE sql
STABLE
AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick
    FROM perception_record pr
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content,
           min(h.tick) AS tick
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
  ), recent AS (
    -- the LAST p_k shared source events by tick (most recent), deterministic tie-break by id
    SELECT s.source_event_id, s.tick, s.modal_content
    FROM shared s
    ORDER BY s.tick DESC, s.source_event_id DESC
    LIMIT p_k
  )
  SELECT r.source_event_id, r.tick AS acquired_tick, r.modal_content AS content
  FROM recent r
  ORDER BY r.tick ASC, r.source_event_id ASC   -- append-only, cache-native
$$;

-- Failure asymmetry (RULINGS-2026-07-23 §5): a missed flag makes the NPC dull for one action; an over-flag costs one extra call. The dangerous failure must stay structurally impossible.
CREATE FUNCTION public.fn_isolated_npcs(
    p_world_id   uuid,
    p_action_ids uuid[],
    p_present    uuid[],
    p_npcs       uuid[]
)
RETURNS TABLE(actor_id uuid)
LANGUAGE sql
STABLE
AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick
    FROM perception_record pr
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
  )
  -- an NPC is isolated iff she holds >=1 PRIVATE record whose about-ness intersects the action ids
  SELECT DISTINCT pr.holder_id AS actor_id
  FROM perception_record pr
  LEFT JOIN shared s
    ON s.source_event_id = pr.source_event_id
   AND s.modal_content   = pr.content        -- matches only the public (modal) face
  WHERE pr.world_id = p_world_id
    AND pr.holder_id = ANY(p_npcs)
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    AND s.source_event_id IS NULL             -- private: no matching public face
    AND EXISTS (
      SELECT 1 FROM perception_subject ps
      WHERE ps.perception_id = pr.perception_id
        AND ps.entity_id = ANY(p_action_ids)  -- one-hop id intersection (ADR-035)
    )
$$;

-- Failure asymmetry (RULINGS-2026-07-23 §5): a missed flag makes the NPC dull for one action; an over-flag costs one extra call. The dangerous failure must stay structurally impossible.
CREATE FUNCTION public.fn_private_records(
    p_world_id   uuid,
    p_npc        uuid,
    p_action_ids uuid[],
    p_present    uuid[]
)
RETURNS TABLE(content text, acquired_tick bigint)
LANGUAGE sql
STABLE
AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick
    FROM perception_record pr
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
  ), freshest AS (
    -- cap is a v1 dial; §10's retrieval assembly refines it in Station I. Keep the FRESHEST 20, not
    -- the oldest: a private cap must drop old records first, never the ones most likely to matter
    -- now. Full tie-break (content, perception_id) keeps the 20-row cut deterministic.
    SELECT pr.content, pr.acquired_tick, pr.perception_id
    FROM perception_record pr
    LEFT JOIN shared s
      ON s.source_event_id = pr.source_event_id
     AND s.modal_content   = pr.content
    WHERE pr.world_id = p_world_id
      AND pr.holder_id = p_npc
      -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
      AND s.source_event_id IS NULL             -- private records only
      AND EXISTS (
        SELECT 1 FROM perception_subject ps
        WHERE ps.perception_id = pr.perception_id
          AND ps.entity_id = ANY(p_action_ids)  -- subjects intersect the action's bound ids
      )
    ORDER BY pr.acquired_tick DESC, pr.content DESC, pr.perception_id DESC
    LIMIT 20
  )
  SELECT f.content, f.acquired_tick
  FROM freshest f
  ORDER BY f.acquired_tick ASC, f.content ASC, f.perception_id ASC   -- present oldest-of-the-freshest first
$$;

-- migrate:down
DROP FUNCTION IF EXISTS public.fn_private_records(uuid, uuid, uuid[], uuid[]);
DROP FUNCTION IF EXISTS public.fn_isolated_npcs(uuid, uuid[], uuid[], uuid[]);
DROP FUNCTION IF EXISTS public.fn_public_moment(uuid, uuid[], int);
