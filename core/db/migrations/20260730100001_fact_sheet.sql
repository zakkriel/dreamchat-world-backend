-- migrate:up

-- Grounded Reasoning / Unit 1 — THE FACT SHEET (the deterministic spine). "The reasoning is detached
-- from the math": the engine computes the physics facts and feeds them to the LLM seats, so every seat
-- reasons FROM computed truth instead of inventing it. This migration builds ONLY that spine — a pure
-- SQL function that, given an action's involved entities (the small set gather_slice already pulls —
-- never the whole world), computes the deterministic facts AMONG them and returns one action-scoped
-- jsonb sheet. NO LLM, no seat reference: it composes the Station F contract functions and nothing more.
--
-- Measurements-not-verdicts (§0, D-1): every fact is COMPUTED at ask-time from stored state; nothing is
-- cached or stored. O(involved), never O(world²) — the aggregate walks unnest(p_involved) once.
--
-- THE WALL (the one flavor difference in v1): p_truth_side selects the referee (truth) flavor vs the
-- character-mind (perceived) flavor. Spatial facts (distance, duration, reachability) are perceivable by
-- anyone in the scene, so they are identical across flavors. The ONLY gated state is a CLOSED container's
-- `contents`: truth-side always shows the id array; perceived-side WITHHOLDS it (json null) unless the
-- container is open (RULINGS-2026-07-23 §9 — the perception walls protect the character-mind seats,
-- never the referee).

-- fn_fact_sheet: the action-scoped fact sheet over p_involved, relative to p_viewer.
--   Returns { "budget_remaining": null, "targets": [ {…per target…} ] } — one entry per involved id
--   EXCEPT the viewer itself. budget_remaining is ALWAYS json null here: it is the beat's tension budget
--   (beat state), filled by the orchestrator later, NOT an entity fact this function can know.
--
--   Per target, relative to the viewer (all via the Station F contract functions):
--     distance_m      = fn_distance(viewer, target)                    — meters, nearest common parent (§3)
--     move_duration_s = fn_move_duration_actor(viewer, target)         — CEIL(distance ÷ effective_speed) (§2)
--     reachable       = same-scene OR fn_portal_permits(viewer_scene, target_scene)   (§5.3)
--     open / locked   = a PORTAL target's Tier-1 booleans (target has `connects`); else json null.
--     weight_kg / volume / would_encumber = an OBJECT target's facts (target has `size`); else json null.
--                       would_encumber = effective_weight > the viewer's max_load.
--     contents        = a CONTAINER target's ids (target has `max_room`); else json null. THE WALL: on
--                       the perceived side a CLOSED container's contents are json null (withheld).
--   Scene resolution reuses fn_target_position (§3: location=itself; actor=attrs.location_id;
--   artifact=COALESCE(attrs.location_id, current_scene_id)) — the SAME resolution the move gate uses, so
--   the fact sheet and the commit never disagree about where a thing is. LANGUAGE sql STABLE (reads
--   current state; no writes) — no LLM, no seat.
CREATE FUNCTION public.fn_fact_sheet(p_world_id uuid, p_viewer uuid, p_involved uuid[], p_truth_side boolean)
RETURNS jsonb
LANGUAGE sql STABLE AS $$
  SELECT jsonb_build_object(
    -- ALWAYS json null: beat state the orchestrator fills, never an entity fact (see header).
    'budget_remaining', NULL::jsonb,
    'targets', COALESCE(
      (SELECT jsonb_agg(s.obj ORDER BY s.ord)
       FROM (
         SELECT
           t.ord,
           jsonb_build_object(
             'id',              t.target_id,
             'kind',            er.entity_kind,
             'name',            er.canonical_name,
             'distance_m',      fn_distance(p_world_id, p_viewer, t.target_id),
             'move_duration_s', fn_move_duration_actor(p_world_id, p_viewer, t.target_id),
             -- same-scene is trivially reachable; else a Portal must permit viewer_scene → target_scene.
             -- COALESCE folds a NULL scene comparison to false so an unresolved scene never yields a
             -- null `reachable` (json false, not json null).
             'reachable',       COALESCE(vs.scene = ts.scene, false)
                                OR fn_portal_permits(p_world_id, vs.scene, ts.scene),
             -- PORTAL facts (target has `connects`): the Tier-1 open/locked booleans; else json null.
             'open',            CASE WHEN art.attrs ? 'connects' THEN art.attrs->'open'   ELSE NULL::jsonb END,
             'locked',          CASE WHEN art.attrs ? 'connects' THEN art.attrs->'locked' ELSE NULL::jsonb END,
             -- OBJECT facts (target has `size`): recursive effective weight, derived volume, and whether
             -- grabbing it would encumber THIS viewer (weight > the viewer's max_load); else json null.
             'weight_kg',       CASE WHEN art.attrs ? 'size'
                                     THEN fn_effective_weight(p_world_id, t.target_id)
                                     ELSE NULL::numeric END,
             'volume',          CASE WHEN art.attrs ? 'size'
                                     THEN fn_volume((art.attrs->>'size')::int)
                                     ELSE NULL::numeric END,
             'would_encumber',  CASE WHEN art.attrs ? 'size'
                                     THEN fn_effective_weight(p_world_id, t.target_id)
                                          > (SELECT (a.attrs->>'max_load')::numeric FROM actor_state a
                                             WHERE a.world_id = p_world_id AND a.entity_id = p_viewer)
                                     ELSE NULL::boolean END,
             -- CONTAINER facts (target has `max_room`): the ids it holds (contents = artifacts whose
             -- contained_by = the container). THE WALL: truth-side always shows them; perceived-side
             -- withholds a CLOSED container's contents (json null). Non-container → json null.
             'contents',        CASE
                                  WHEN NOT (art.attrs ? 'max_room') THEN NULL::jsonb
                                  WHEN p_truth_side OR art.attrs->>'open' = 'true' THEN
                                    COALESCE(
                                      (SELECT jsonb_agg(c.entity_id::text ORDER BY c.entity_id)
                                       FROM artifact_state c
                                       WHERE c.world_id = p_world_id
                                         AND (c.attrs->>'contained_by')::uuid = t.target_id),
                                      '[]'::jsonb)
                                  ELSE NULL::jsonb
                                END
           ) AS obj
         FROM unnest(p_involved) WITH ORDINALITY AS t(target_id, ord)
         JOIN entity_registry er
           ON er.world_id = p_world_id AND er.entity_id = t.target_id
         LEFT JOIN artifact_state art
           ON art.world_id = p_world_id AND art.entity_id = t.target_id
         -- scene resolution (§3), reusing the move gate's resolver so facts never disagree with commits.
         LEFT JOIN LATERAL fn_target_position(p_world_id, t.target_id) AS ts ON true
         LEFT JOIN LATERAL fn_target_position(p_world_id, p_viewer)    AS vs ON true
         WHERE t.target_id <> p_viewer            -- one entry per involved id EXCEPT the viewer itself
       ) s),
      '[]'::jsonb)                                -- no targets (only the viewer, or empty) → empty array
  );
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_fact_sheet(uuid, uuid, uuid[], boolean);
