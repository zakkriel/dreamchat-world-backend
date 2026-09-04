-- migrate:up
--
-- Closes the faction knowledge leak: fn_visible_perceptions returned any perception whose
-- holder was a 'faction'/'group' entity to EVERY viewer, with no membership condition --
-- broadcast, not shared knowledge. It was inert only because genesis never registers a
-- faction. Removed here; restored when membership exists to check against. See
-- docs/design/2026-09-02-concepts-as-knowledge.md §7 and SPEC-051 item 8.
--
-- Function body carried forward verbatim per this repo's migration convention, less the
-- unconditional faction/group branch.

CREATE OR REPLACE FUNCTION public.fn_visible_perceptions(p_world_id uuid, p_viewer_id uuid) RETURNS SETOF public.perception_record
    LANGUAGE sql STABLE
    AS $$
  -- A perception is visible to its holder. The previous version also returned every
  -- perception held by ANY 'faction'/'group' entity to EVERY viewer, with no membership
  -- condition -- broadcast, not shared knowledge. It was inert only because genesis never
  -- registers a faction. Restored when membership exists; see
  -- docs/design/2026-09-02-concepts-as-knowledge.md §7 and SPEC-051 item 8.
  SELECT pr.*
  FROM perception_record pr
  WHERE pr.world_id = p_world_id
    AND pr.invalid_tick IS NULL
    AND pr.expired_at  IS NULL
    AND pr.holder_id = p_viewer_id;
$$;

-- migrate:down

CREATE OR REPLACE FUNCTION public.fn_visible_perceptions(p_world_id uuid, p_viewer_id uuid) RETURNS SETOF public.perception_record
    LANGUAGE sql STABLE
    AS $$
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
