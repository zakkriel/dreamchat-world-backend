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

-- Assembly: FILTER 1 ∘ FILTER 2 + name + perception-bound fields. Returns the actor_page/1 payload.
-- FILTER 2 = perception_subject (primary about-ness). Identity/name substrate (world_genesis-sourced)
-- is EXCLUDED from collected knowledge so a name never masquerades as a knowledge item.
-- TRIPWIRE (design §3): the `event_type <> 'world_genesis'` exclusion is correct ONLY while genesis
-- sources names exclusively. If genesis ever sources a non-name perception, switch this to a real
-- name/identity discriminator instead of keying on the source event.
-- HARD RULE: never reads actor_state/location_state (authoritative canon) — last_known_status is
-- perception-bound or null (B-1/I-3).
CREATE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json
LANGUAGE sql STABLE AS $$
  WITH about_actor AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence,
           ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_actor_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id',   aa.perception_id,
             'content',         aa.content,
             'epistemic_type',  aa.epistemic_type,
             'occurred_at_tick',aa.valid_tick,
             'display_label',   aa.in_world_label,
             'confidence',      aa.confidence,
             'decay',           json_build_object('stale', false, 'last_confirmed_label', aa.in_world_label),
             'source',          json_build_object('epistemic_type', aa.epistemic_type,
                                                  'source_event_label', aa.in_world_label)
           ) AS item,
           aa.valid_tick AS sort_tick
    FROM about_actor aa
  ),
  groups AS (
    SELECT CASE WHEN count(*) = 0 THEN '[]'::json
                ELSE json_build_array(json_build_object(
                       'group_key',   p_actor_id::text,
                       'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
                       'items',       coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                     ))
           END AS arr
    FROM items i
  )
  SELECT json_build_object(
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
      'collected_knowledge_groups', (SELECT arr FROM groups),
      'inline_links',               '[]'::json
    )
  );
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_actor_page(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS fn_perceived_name(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS fn_visible_perceptions(uuid, uuid);
