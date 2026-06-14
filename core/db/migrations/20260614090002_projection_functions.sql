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

-- Name resolution — a GENUINE knowability gate (not a raw entity_registry read; going-in 5).
-- Returns the perception-layer name content IFF the entity is knowable to the viewer:
--  priority 1: a viewer-held divergent perceived-name perception (DEFERRED — none in 0A; the
--              branch is intentionally absent, the seam is this function boundary);
--  priority 2: a common-knowledge name perception (CK-held, world_genesis-sourced, subject=entity)
--              that the viewer is permitted to see (routed through FILTER 1);
--  else NULL (WITHHELD). A noise actor with no CK name perception returns NULL.
CREATE FUNCTION fn_perceived_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS text
LANGUAGE sql STABLE AS $$
  SELECT vp.content
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id AND ps.entity_id = p_entity_id
  JOIN canon_event ce ON ce.event_id = vp.source_event_id
  WHERE ce.event_type = 'world_genesis'
  ORDER BY vp.acquired_tick
  LIMIT 1;
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_perceived_name(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS fn_visible_perceptions(uuid, uuid);
