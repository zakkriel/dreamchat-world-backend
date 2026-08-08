-- migrate:up

-- Tunable staleness horizon for compendium/timeline decay labels.
CREATE OR REPLACE FUNCTION fn_compendium_decay_horizon_ticks()
RETURNS bigint
LANGUAGE sql IMMUTABLE AS $$
  SELECT 72::bigint;
$$;

-- DESIGN CALL #2 (decay.stale): a perception is stale when the world's current tick has advanced
-- more than fn_compendium_decay_horizon_ticks() beyond the perception's valid_tick. This keeps the
-- threshold in one named place and ties "stale" to elapsed in-world time, not wall clock.
CREATE OR REPLACE FUNCTION fn_compendium_decay(
  p_world_id uuid,
  p_valid_tick bigint,
  p_last_confirmed_label text
)
RETURNS json
LANGUAGE sql STABLE AS $$
  SELECT json_build_object(
    'stale', (fn_world_now(p_world_id) - p_valid_tick) > fn_compendium_decay_horizon_ticks(),
    'last_confirmed_label', p_last_confirmed_label
  );
$$;

-- DESIGN CALL #1 (current_synthesis): a deterministic composition of the viewer's OWN held
-- perceptions about the target — newest first by (valid_tick, acquired_tick, perception_id), capped
-- to the 3 most recent, newline-joined. No LLM (SQL cannot call one, and a synthesis that invented
-- connective prose would be asserting more than the viewer holds), no hidden truth: every line is a
-- perception this viewer already has, so the paragraph can never say more than he knows (B-1, I-3).
-- Actors AC#3 asks exactly this — "the synthesis paragraph reflects only held perception records".
--
-- What it deliberately does NOT do:
--   • no ordinals ("1." / "2."). That is presentation, and presentation belongs to the frontend.
--   • no time label. An unlabelled event used to render as "[Tick 51]", which manufactures a display
--     label out of the logical tick — precisely what B-5 forbids (the tick is operational; the
--     display label is authored, and when none is authored the honest answer is silence). Each
--     knowledge item already carries its own `display_label`, so the label belongs there, not here.
CREATE OR REPLACE FUNCTION fn_compendium_current_synthesis(
  p_world_id uuid,
  p_viewer_id uuid,
  p_target_id uuid
)
RETURNS text
LANGUAGE sql STABLE AS $$
  WITH ranked AS (
    SELECT vp.content,
           row_number() OVER (
             ORDER BY vp.valid_tick DESC, vp.acquired_tick DESC, vp.perception_id DESC
           ) AS rn
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
    JOIN perception_subject ps
      ON ps.perception_id = vp.perception_id
     AND ps.entity_id = p_target_id
    JOIN canon_event ce
      ON ce.event_id = vp.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  )
  SELECT CASE WHEN count(*) = 0 THEN NULL
              ELSE string_agg(r.content, E'\n' ORDER BY r.rn)
         END
  FROM ranked r
  WHERE r.rn <= 3;
$$;

CREATE OR REPLACE FUNCTION fn_compendium_latest_fact(
  p_world_id uuid,
  p_viewer_id uuid,
  p_target_id uuid
)
RETURNS text
LANGUAGE sql STABLE AS $$
  SELECT vp.content
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
  JOIN perception_subject ps
    ON ps.perception_id = vp.perception_id
   AND ps.entity_id = p_target_id
  JOIN canon_event ce
    ON ce.event_id = vp.source_event_id
  WHERE ce.event_type <> 'world_genesis'
  ORDER BY vp.valid_tick DESC, vp.acquired_tick DESC, vp.perception_id DESC
  LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION fn_compendium_related_entities(
  p_world_id uuid,
  p_viewer_id uuid,
  p_target_id uuid,
  p_related_kinds text[]
)
RETURNS json
LANGUAGE sql STABLE AS $$
  WITH about_target AS (
    SELECT vp.perception_id,
           vp.valid_tick
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
    JOIN perception_subject pst
      ON pst.perception_id = vp.perception_id
     AND pst.entity_id = p_target_id
    JOIN canon_event ce
      ON ce.event_id = vp.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  related AS (
    SELECT er.entity_id,
           er.entity_kind,
           at.valid_tick,
           at.perception_id
    FROM about_target at
    JOIN perception_subject ps2
      ON ps2.perception_id = at.perception_id
     AND ps2.entity_id <> p_target_id
    JOIN entity_registry er
      ON er.world_id = p_world_id
     AND er.entity_id = ps2.entity_id
    WHERE er.entity_kind = ANY (p_related_kinds)
  ),
  collapsed AS (
    SELECT r.entity_id,
           r.entity_kind,
           max(r.valid_tick) AS last_seen_tick,
           count(DISTINCT r.perception_id) AS evidence_count
    FROM related r
    GROUP BY r.entity_id, r.entity_kind
  )
  SELECT coalesce(
           json_agg(
             json_build_object(
               'id', c.entity_id,
               'kind', c.entity_kind,
               'perceived_name', fn_perceived_name(p_world_id, p_viewer_id, c.entity_id),
               'last_seen_tick', c.last_seen_tick,
               'evidence_count', c.evidence_count
             )
             ORDER BY c.last_seen_tick DESC, c.entity_id
           ),
           '[]'::json
         )
  FROM collapsed c;
$$;

CREATE OR REPLACE FUNCTION fn_compendium_last_known_location(
  p_world_id uuid,
  p_viewer_id uuid,
  p_artifact_id uuid
)
RETURNS text
LANGUAGE sql STABLE AS $$
  WITH about_artifact AS (
    SELECT vp.perception_id,
           vp.valid_tick,
           vp.acquired_tick
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
    JOIN perception_subject psa
      ON psa.perception_id = vp.perception_id
     AND psa.entity_id = p_artifact_id
    JOIN canon_event ce
      ON ce.event_id = vp.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  )
  SELECT fn_perceived_name(p_world_id, p_viewer_id, er.entity_id)
  FROM about_artifact aa
  JOIN perception_subject psl
    ON psl.perception_id = aa.perception_id
   AND psl.entity_id <> p_artifact_id
  JOIN entity_registry er
    ON er.world_id = p_world_id
   AND er.entity_id = psl.entity_id
   AND er.entity_kind = 'location'
  ORDER BY aa.valid_tick DESC, aa.acquired_tick DESC, aa.perception_id DESC, er.entity_id
  LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION fn_collected_knowledge(p_world_id uuid, p_viewer_id uuid, p_target_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH about AS (
    SELECT v.perception_id,
           v.source_event_id,
           v.content,
           v.epistemic_type,
           v.valid_tick,
           v.confidence,
           ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps
      ON ps.perception_id = v.perception_id
     AND ps.entity_id = p_target_id
    JOIN canon_event ce
      ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT a.source_event_id,
           a.in_world_label,
           a.valid_tick AS sort_tick,
           a.perception_id,
           json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',            fn_compendium_decay(p_world_id, a.valid_tick, a.in_world_label),
             'source',           json_build_object(
                                   'epistemic_type', a.epistemic_type,
                                   'source_event_label', a.in_world_label
                                 )
           ) AS item
    FROM about a
  ),
  grouped AS (
    SELECT i.source_event_id,
           max(i.in_world_label) AS group_label,
           max(i.sort_tick) AS group_sort_tick,
           coalesce(
             json_agg(i.item ORDER BY i.sort_tick, i.perception_id),
             '[]'::json
           ) AS group_items
    FROM items i
    GROUP BY i.source_event_id
  )
  SELECT coalesce(
           json_agg(
             json_build_object(
               'group_key',   'event:' || g.source_event_id::text,
               'group_label', g.group_label,
               'items',       g.group_items
             )
             ORDER BY g.group_sort_tick DESC, g.source_event_id
           ),
           '[]'::json
         )
  FROM grouped g;
$$;

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
      -- Perception rows do not carry a structured role taxonomy for actors.
      'perceived_role',             NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_actor_id),
      'last_known_status',          fn_compendium_latest_fact(p_world_id, p_viewer_id, p_actor_id),
      'known_artifacts',            fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['artifact']),
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['actor','location','artifact'])
    )
  ) END;
$$;

CREATE OR REPLACE FUNCTION fn_location_page(p_world_id uuid, p_viewer_id uuid, p_location_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_location_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'location_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'location', json_build_object(
      'id',                         p_location_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_location_id),
      -- No perception-level containment relation exists yet; "part_of" stays a stub.
      'part_of',                    NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_location_id),
      'last_known_status',          fn_compendium_latest_fact(p_world_id, p_viewer_id, p_location_id),
      -- Co-mentioned locations exist, but "inside" requires a containment edge not present in perception rows.
      'known_areas_inside',         '[]'::json,
      'key_actors',                 fn_compendium_related_entities(p_world_id, p_viewer_id, p_location_id, ARRAY['actor']),
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_location_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_location_id, ARRAY['actor','location','artifact'])
    )
  ) END;
$$;

CREATE OR REPLACE FUNCTION fn_artifact_page(p_world_id uuid, p_viewer_id uuid, p_artifact_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_artifact_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'artifact_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'artifact', json_build_object(
      'id',                         p_artifact_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_artifact_id),
      -- Perception rows do not encode a typed artifact classification.
      'perceived_type',             NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_artifact_id),
      'last_known_location',        fn_compendium_last_known_location(p_world_id, p_viewer_id, p_artifact_id),
      -- Holder/owner/access requires carry-state lenses that are not modeled in perception rows.
      'current_holder_owner_access',NULL,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_artifact_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_artifact_id, ARRAY['actor','location','artifact'])
    )
  ) END;
$$;

CREATE OR REPLACE FUNCTION fn_timeline(p_world_id uuid, p_viewer_id uuid, p_before_tick bigint DEFAULT NULL)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH mine AS (
    SELECT v.perception_id,
           v.content,
           v.epistemic_type,
           v.valid_tick,
           v.confidence,
           ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN canon_event ce
      ON ce.event_id = v.source_event_id
    WHERE v.holder_id = p_viewer_id
      AND (p_before_tick IS NULL OR v.valid_tick < p_before_tick)
  )
  SELECT json_build_object(
    'schema_version', 'timeline/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'records', coalesce(
      (SELECT json_agg(
                json_build_object(
                  'perception_id',    m.perception_id,
                  'content',          m.content,
                  'epistemic_type',   m.epistemic_type,
                  'occurred_at_tick', m.valid_tick,
                  'display_label',    m.in_world_label,
                  'confidence',       m.confidence,
                  'decay',            fn_compendium_decay(p_world_id, m.valid_tick, m.in_world_label)
                )
                ORDER BY m.valid_tick, m.perception_id
              )
       FROM mine m),
      '[]'::json
    )
  );
$$;

-- migrate:down

CREATE OR REPLACE FUNCTION fn_collected_knowledge(p_world_id uuid, p_viewer_id uuid, p_target_id uuid)
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

CREATE OR REPLACE FUNCTION fn_location_page(p_world_id uuid, p_viewer_id uuid, p_location_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_location_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'location_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'location', json_build_object(
      'id',                         p_location_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_location_id),
      'part_of',                    NULL,
      'current_synthesis',          NULL,
      'last_known_status',          NULL,
      'known_areas_inside',         '[]'::json,
      'key_actors',                 '[]'::json,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_location_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;

CREATE OR REPLACE FUNCTION fn_artifact_page(p_world_id uuid, p_viewer_id uuid, p_artifact_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_artifact_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'artifact_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'artifact', json_build_object(
      'id',                         p_artifact_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_artifact_id),
      'perceived_type',             NULL,
      'current_synthesis',          NULL,
      'last_known_location',        NULL,
      'current_holder_owner_access',NULL,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_artifact_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;

CREATE OR REPLACE FUNCTION fn_timeline(p_world_id uuid, p_viewer_id uuid, p_before_tick bigint DEFAULT NULL)
RETURNS json LANGUAGE sql STABLE AS $$
  WITH mine AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE v.holder_id = p_viewer_id
      AND (p_before_tick IS NULL OR v.valid_tick < p_before_tick)
  )
  SELECT json_build_object(
    'schema_version', 'timeline/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'records', coalesce(
      (SELECT json_agg(json_build_object(
                'perception_id',    m.perception_id,
                'content',          m.content,
                'epistemic_type',   m.epistemic_type,
                'occurred_at_tick', m.valid_tick,
                'display_label',    m.in_world_label,
                'confidence',       m.confidence,
                'decay', json_build_object('stale', false, 'last_confirmed_label', m.in_world_label))
              ORDER BY m.valid_tick, m.perception_id)
       FROM mine m), '[]'::json)
  );
$$;

DROP FUNCTION IF EXISTS fn_compendium_last_known_location(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS fn_compendium_related_entities(uuid, uuid, uuid, text[]);
DROP FUNCTION IF EXISTS fn_compendium_latest_fact(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS fn_compendium_current_synthesis(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS fn_compendium_decay(uuid, bigint, text);
DROP FUNCTION IF EXISTS fn_compendium_decay_horizon_ticks();
