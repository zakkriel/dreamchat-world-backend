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

-- fn_collected_knowledge — the SHARED about-ness core (FILTER 1 ∘ perception_subject ∘ genesis
-- exclusion ∘ grouping). Returns the collected_knowledge_groups JSON array (or '[]'). Identical lens
-- for actor/location/artifact pages — never reimplemented per page.
-- TRIPWIRE (design §3): the world_genesis exclusion is correct ONLY while genesis sources names
-- exclusively. If genesis ever sources a non-name perception, switch to a real name/identity marker.
CREATE FUNCTION fn_collected_knowledge(p_world_id uuid, p_viewer_id uuid, p_target_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH about AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_target_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',  json_build_object('stale', false, 'last_confirmed_label', a.in_world_label),
             'source', json_build_object('epistemic_type', a.epistemic_type,
                                         'source_event_label', a.in_world_label)
           ) AS item,
           a.valid_tick AS sort_tick
    FROM about a
  )
  SELECT CASE WHEN count(*) = 0 THEN '[]'::json
              ELSE json_build_array(json_build_object(
                     'group_key',   p_target_id::text,
                     'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_target_id),
                     'items',       coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                   ))
         END
  FROM items i;
$$;

-- fn_actor_page — REFACTORED onto the shared core + existence gate. Returns NULL when the actor is
-- not in the viewer's existence set (Go → 404), closing the latent Chunk-3 direct-id leak. The JSON
-- shape is byte-identical to Chunk-3 for any VISIBLE actor → 45_actor_page_test passes UNCHANGED.
CREATE OR REPLACE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_actor_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'actor_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id',                         p_actor_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      'perceived_role',             NULL,
      'current_synthesis',          NULL,
      'last_known_status',          NULL,
      'known_artifacts',            '[]'::json,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_collected_knowledge(uuid, uuid, uuid);
-- restore the Chunk-3 fn_actor_page (verbatim from migration 0002) on rollback
CREATE OR REPLACE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH about_actor AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_actor_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id', aa.perception_id, 'content', aa.content,
             'epistemic_type', aa.epistemic_type, 'occurred_at_tick', aa.valid_tick,
             'display_label', aa.in_world_label, 'confidence', aa.confidence,
             'decay', json_build_object('stale', false, 'last_confirmed_label', aa.in_world_label),
             'source', json_build_object('epistemic_type', aa.epistemic_type,
                                         'source_event_label', aa.in_world_label)
           ) AS item, aa.valid_tick AS sort_tick
    FROM about_actor aa
  ),
  groups AS (
    SELECT CASE WHEN count(*) = 0 THEN '[]'::json
                ELSE json_build_array(json_build_object(
                       'group_key', p_actor_id::text,
                       'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
                       'items', coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                     )) END AS arr
    FROM items i
  )
  SELECT json_build_object(
    'schema_version', 'actor_page/1', 'world_id', p_world_id, 'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id', p_actor_id, 'perceived_name', fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      'perceived_role', NULL, 'current_synthesis', NULL, 'last_known_status', NULL,
      'known_artifacts', '[]'::json, 'collected_knowledge_groups', (SELECT arr FROM groups),
      'inline_links', '[]'::json));
$$;
DROP FUNCTION IF EXISTS fn_compendium_index_json(uuid, uuid, text);
DROP FUNCTION IF EXISTS fn_compendium_index(uuid, uuid, text);
DROP FUNCTION IF EXISTS fn_entity_visible(uuid, uuid, uuid);
