-- migrate:up

-- §3 naming reach (RULINGS-2026-07-23): "the candidate whitelist for identification = what the acting
-- actor perceives or knows, one hop… Nobody can bind an entity they have no perception/knowledge path
-- to." The live founder-gate leak: canonical names reached the character-mind seats past knowledge
-- paths (the narration named "Jonas" to Kade, who knows him only as "the muscle"). fn_display_name is
-- the per-viewer LOOKUP that closes it: known-name (the viewer's own name-knowledge) else descriptor
-- else canonical. It does NOT touch fn_perceived_name's pinned contract (43_perceived_name_test.sql) —
-- it composes it.

CREATE FUNCTION public.fn_display_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS text
LANGUAGE sql STABLE AS $$
  -- The label an entity wears in ONE viewer's mind (§3). Known name (the viewer's own name-knowledge,
  -- via fn_perceived_name — a world_genesis-sourced perception subject-linked to the entity that the
  -- viewer holds) if any; else the entity's DESCRIPTOR (Tier-2 'descriptor' attr — what a stranger
  -- sees, e.g. "the muscle by the bar"); else the canonical registry name (engine fallback; a seed lag,
  -- never shown once descriptors are seeded). ids stay real at every call site — the model binds the
  -- id, this is only the label the viewer's knowledge puts on it.
  SELECT COALESCE(
    fn_perceived_name(p_world_id, p_viewer_id, p_entity_id),
    (SELECT ast.attrs->>'descriptor' FROM actor_state ast
      WHERE ast.world_id = p_world_id AND ast.entity_id = p_entity_id),
    (SELECT er.canonical_name FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_entity_id)
  );
$$;

-- §3 + §5 naming reach for the BATCH seat: one shared prompt speaks for several minds, so a name may
-- appear only if EVERY batch mind resolves the SAME known name for the entity (shared-by-all
-- intersection — the same mechanical philosophy as fn_public_moment's modal face, no judgment).
-- Otherwise the descriptor, else canonical. An empty mind set has nothing to intersect → descriptor/
-- canonical. Failure asymmetry matches the wall's: an over-strict intersection shows a descriptor
-- where a shared name existed (dull, never a leak); it can never surface a name a mind does not know.
CREATE FUNCTION public.fn_batch_display_name(p_world_id uuid, p_minds uuid[], p_entity_id uuid)
RETURNS text
LANGUAGE sql STABLE AS $$
  WITH names AS (
    SELECT fn_perceived_name(p_world_id, m, p_entity_id) AS nm
    FROM unnest(p_minds) AS m
  ), agreed AS (
    SELECT max(nm) AS nm
    FROM names
    HAVING cardinality(p_minds) > 0
       AND count(*) = count(nm)          -- every mind resolved a name (no NULLs among the minds)
       AND count(DISTINCT nm) = 1        -- and it is the SAME name for all of them
  )
  SELECT COALESCE(
    (SELECT nm FROM agreed),
    (SELECT ast.attrs->>'descriptor' FROM actor_state ast
      WHERE ast.world_id = p_world_id AND ast.entity_id = p_entity_id),
    (SELECT er.canonical_name FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_entity_id)
  );
$$;

-- Name-knowledge is NOT a secret. fn_perceived_name identifies name perceptions mechanically by their
-- world_genesis source (43_perceived_name_test.sql; the fn_actor_page tripwire already excludes them
-- from "collected knowledge" so a name never masquerades as a knowledge item). Per-viewer name
-- knowledge (Kade holds "Mara", Mara privately holds Kade's name) is stored the same way — a
-- world_genesis-sourced perception held by that viewer — so WITHOUT this exclusion those name records
-- would (a) flag their holder ISOLATED whenever the named entity is in the action (splitting a regular
-- like Jonas out of the batch for merely knowing Mara's name), and (b) ride into her private-records
-- block as if they were secrets. Exclude world_genesis-sourced rows from BOTH private-record lookups,
-- mirroring the established fn_actor_page discipline: a name is an identity substrate, never a secret.

CREATE OR REPLACE FUNCTION public.fn_isolated_npcs(
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
    -- name-knowledge is an identity substrate, not a secret: a world_genesis-sourced record must
    -- never pull its holder isolated (§3; mirrors the fn_actor_page tripwire).
    AND NOT EXISTS (
      SELECT 1 FROM canon_event ce
      WHERE ce.event_id = pr.source_event_id AND ce.event_type = 'world_genesis'
    )
    AND EXISTS (
      SELECT 1 FROM perception_subject ps
      WHERE ps.perception_id = pr.perception_id
        AND ps.entity_id = ANY(p_action_ids)  -- one-hop id intersection (ADR-035)
    )
$$;

CREATE OR REPLACE FUNCTION public.fn_private_records(
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
      -- name-knowledge is an identity substrate, not a secret: a world_genesis-sourced record must
      -- never ride the private block (§3; mirrors the fn_actor_page tripwire).
      AND NOT EXISTS (
        SELECT 1 FROM canon_event ce
        WHERE ce.event_id = pr.source_event_id AND ce.event_type = 'world_genesis'
      )
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

DROP FUNCTION IF EXISTS public.fn_batch_display_name(uuid, uuid[], uuid);
DROP FUNCTION IF EXISTS public.fn_display_name(uuid, uuid, uuid);

-- Restore fn_isolated_npcs / fn_private_records to their pre-exclusion (cognition_lookups) bodies.
CREATE OR REPLACE FUNCTION public.fn_isolated_npcs(
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
  SELECT DISTINCT pr.holder_id AS actor_id
  FROM perception_record pr
  LEFT JOIN shared s
    ON s.source_event_id = pr.source_event_id
   AND s.modal_content   = pr.content
  WHERE pr.world_id = p_world_id
    AND pr.holder_id = ANY(p_npcs)
    AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    AND s.source_event_id IS NULL
    AND EXISTS (
      SELECT 1 FROM perception_subject ps
      WHERE ps.perception_id = pr.perception_id
        AND ps.entity_id = ANY(p_action_ids)
    )
$$;

CREATE OR REPLACE FUNCTION public.fn_private_records(
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
    SELECT pr.content, pr.acquired_tick, pr.perception_id
    FROM perception_record pr
    LEFT JOIN shared s
      ON s.source_event_id = pr.source_event_id
     AND s.modal_content   = pr.content
    WHERE pr.world_id = p_world_id
      AND pr.holder_id = p_npc
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
      AND s.source_event_id IS NULL
      AND EXISTS (
        SELECT 1 FROM perception_subject ps
        WHERE ps.perception_id = pr.perception_id
          AND ps.entity_id = ANY(p_action_ids)
      )
    ORDER BY pr.acquired_tick DESC, pr.content DESC, pr.perception_id DESC
    LIMIT 20
  )
  SELECT f.content, f.acquired_tick
  FROM freshest f
  ORDER BY f.acquired_tick ASC, f.content ASC, f.perception_id ASC
$$;
