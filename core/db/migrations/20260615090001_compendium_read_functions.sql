-- migrate:up
-- Chunk-4 read-only projection functions (design 2026-06-14). The perception/existence filter lives
-- HERE in SQL, pgTAP-tested; Go is a thin reader (ADR-P017). Live joins only (SPEC-009 unfired).
-- FILTER 1 (fn_visible_perceptions) and the name gate (fn_perceived_name) from migration 0002 are
-- REUSED UNCHANGED — only FILTER 2 / page envelopes are added here.

-- fn_entity_visible — existence predicate (B-1/I-3/B-2, read-side; no new engine ADR). Boolean form of
-- FILTER 1 ∘ perception_subject. CK entities pass for every viewer via universal-holder genesis rows.
CREATE FUNCTION fn_entity_visible(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT EXISTS (
    SELECT 1
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp     -- FILTER 1, unchanged
    JOIN perception_subject ps ON ps.perception_id = vp.perception_id
    WHERE ps.entity_id = p_entity_id);
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_entity_visible(uuid, uuid, uuid);
