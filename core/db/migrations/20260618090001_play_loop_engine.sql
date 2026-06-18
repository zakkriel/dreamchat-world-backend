-- migrate:up
-- Chunk-5 play-loop engine (design 2026-06-16). Deterministic; NO model. SQL is the engine
-- (ADR-P017); the Go layer (Leg 2) is a thin orchestrator. Functions added incrementally across
-- Tasks 1–5; the down body drops them in reverse.

-- Co-presence (thin-slice SPEC-017 substrate): actors whose projected location label matches.
-- Reads actor_state (the projection), not canon — co-presence is a STATE question.
CREATE FUNCTION fn_actors_at(p_world_id uuid, p_location text)
RETURNS TABLE(entity_id uuid)
LANGUAGE sql STABLE AS $$
  SELECT a.entity_id
  FROM actor_state a
  WHERE a.world_id = p_world_id
    AND a.attrs->>'location_id' = p_location;
$$;

-- ADR-036 substrate: per-event duration as RECORDED, deterministic world data (D-11; §11), assigned
-- by the engine — NEVER the model (§10 Q1 guardrail). Thin slice = a hand-authored cost table; the
-- spatial engine (coordinates → derived distance/travel-time) is DEFERRED wholesale (SPEC-018), so
-- there is no derive here. Unknown pairs fall back to a flat default; same place = 0.
CREATE FUNCTION fn_move_duration(p_world_id uuid, p_from text, p_to text)
RETURNS bigint
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
           WHEN p_from = p_to THEN 0
           WHEN (p_from,p_to) IN (('tavern','square'),('square','tavern')) THEN 5
           ELSE 5   -- flat default for the thin-slice fixture map
         END::bigint;
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_move_duration(uuid, text, text);
DROP FUNCTION IF EXISTS fn_actors_at(uuid, text);
