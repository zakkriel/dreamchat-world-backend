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

-- fn_compendium_index — set-form of the existence predicate, bucketed by kind AFTER the existence
-- join (kind is a post-filter from entity_registry, never a parallel path). perceived_name is the
-- perception-layer name (fn_perceived_name), NULL when withheld — never entity_registry.canonical_name.
CREATE FUNCTION fn_compendium_index(p_world_id uuid, p_viewer_id uuid, p_kind text)
RETURNS TABLE (entity_id uuid, perceived_name text)
LANGUAGE sql STABLE AS $$
  SELECT DISTINCT er.entity_id,
         fn_perceived_name(p_world_id, p_viewer_id, er.entity_id)
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp       -- FILTER 1, unchanged
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id
  JOIN entity_registry er ON er.entity_id = ps.entity_id AND er.world_id = p_world_id
  WHERE er.entity_kind = p_kind;
$$;

-- thin JSON envelope for the endpoint (compendium_index/1). Flat per-kind list; entries may carry
-- a NULL perceived_name (existence perceived, name withheld).
CREATE FUNCTION fn_compendium_index_json(p_world_id uuid, p_viewer_id uuid, p_kind text)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT json_build_object(
    'schema_version', 'compendium_index/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'kind',      p_kind,
    'entries', coalesce(
      (SELECT json_agg(json_build_object('id', entity_id, 'perceived_name', perceived_name)
                       ORDER BY entity_id)
       FROM fn_compendium_index(p_world_id, p_viewer_id, p_kind)), '[]'::json)
  );
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_compendium_index_json(uuid, uuid, text);
DROP FUNCTION IF EXISTS fn_compendium_index(uuid, uuid, text);
DROP FUNCTION IF EXISTS fn_entity_visible(uuid, uuid, uuid);
