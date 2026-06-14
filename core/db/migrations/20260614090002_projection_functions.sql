-- migrate:up
-- Chunk-3 read-only projection functions. The perception/safety filter (B-1, I-3) lives HERE,
-- in SQL, pgTAP-tested; the Go app layer is a thin reader and never reimplements it (ADR-P017).
-- Live joins only — no materialized projection (SPEC-009 tripwire unfired). SECURITY: these read
-- authoritative perception rows; the safety wall is the WHERE clause, not the caller.

-- FILTER 1 — the safety wall. holder ∈ {viewer} ∪ {world's universal common-knowledge holders},
-- AND the perception is still held (invalid_tick IS NULL AND expired_at IS NULL). 0A: common-knowledge
-- holders are the world's faction/group entities (ambient membership; the one seeded such holder is
-- 'Common Knowledge'). A per-actor group-membership table is a deferred STORAGE optimization
-- (SPEC-006 scale trigger), never a new knowledge path.
CREATE FUNCTION fn_visible_perceptions(p_world_id uuid, p_viewer_id uuid)
RETURNS SETOF perception_record
LANGUAGE sql STABLE AS $$
  SELECT pr.*
  FROM perception_record pr
  WHERE pr.world_id = p_world_id
    AND pr.invalid_tick IS NULL
    AND pr.expired_at  IS NULL
    AND ( pr.holder_id = p_viewer_id
          OR pr.holder_id IN (
            SELECT er.entity_id FROM entity_registry er
            WHERE er.world_id = p_world_id
              AND er.entity_kind IN ('faction','group')
          )
        );
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_visible_perceptions(uuid, uuid);
