-- migrate:up

-- Station F / FINAL-action-contracts.md §3 (Space): nested coordinates on the existing location
-- hierarchy, and the ONE distance formula that spans every scale (barroom to continent).
--
-- Model: every location has a coordinate WITHIN its parent (location_state.attrs.coordinates), and a
-- parent edge (location_state.attrs->>'parent_location_id', a Tier-1 string). Things inside a location
-- -- actors and artifacts -- carry a coordinate within their current scene (actor_state /
-- artifact_state attrs.coordinates). Distance between two things = distance between their coordinates
-- at the NEAREST COMMON PARENT's local frame (§3). No special in-scene case: same-scene is just the
-- case where the nearest common parent IS the shared scene.
--
-- This path is deliberately LLM-FREE (D-1): it is pure arithmetic over stored measurements, so the
-- move contract (§2, duration = distance / speed) can compute a blocker without ever asking a model.

-- fn_location_depth: breadcrumb depth of a location, root = 0, walking the parent_location_id edge up
-- the hierarchy (recursive CTE). Consumed by nearest-common-parent to climb both sides to equal depth.
CREATE FUNCTION public.fn_location_depth(p_world_id uuid, p_location uuid)
RETURNS int
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE chain AS (
    SELECT p_location AS loc, 0 AS depth
    UNION ALL
    SELECT (ls.attrs->>'parent_location_id')::uuid, c.depth + 1
    FROM chain c
    JOIN location_state ls
      ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL   -- stop at the root (no parent edge)
  )
  SELECT max(depth) FROM chain;   -- deepest step reached climbing to root = the location's depth
$$;

-- fn_nearest_common_parent: the deepest location that is an ancestor-or-self of BOTH entities' scenes.
-- Entity -> scene resolution (§3): an actor's scene is actor_state.attrs->>'location_id'; a location
-- IS its own scene; an artifact's scene is its own attrs->>'location_id' if present, else its registry
-- current_scene_id. We enumerate the ancestor-or-self chain of each scene, intersect them, and take
-- the common ancestor closest to A (fewest steps up) -- equivalently the deepest -- which is exactly
-- "climb to equal depth, then together until equal" expressed set-wise.
CREATE FUNCTION public.fn_nearest_common_parent(p_world_id uuid, p_a uuid, p_b uuid)
RETURNS uuid
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  scenes AS (
    SELECT x.which,
      CASE er.entity_kind
        WHEN 'location' THEN x.eid
        WHEN 'actor'    THEN (ast.attrs->>'location_id')::uuid
        WHEN 'artifact' THEN COALESCE((art.attrs->>'location_id')::uuid, er.current_scene_id)
      END AS scene
    FROM (VALUES ('a', p_a), ('b', p_b)) AS x(which, eid)
    JOIN entity_registry er ON er.world_id = p_world_id AND er.entity_id = x.eid
    LEFT JOIN actor_state    ast ON ast.world_id = p_world_id AND ast.entity_id = x.eid
    LEFT JOIN artifact_state art ON art.world_id = p_world_id AND art.entity_id = x.eid
  ),
  anc AS (
    SELECT s.which, s.scene AS loc, 0 AS up
    FROM scenes s
    UNION ALL
    SELECT a.which, (ls.attrs->>'parent_location_id')::uuid, a.up + 1
    FROM anc a
    JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = a.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
  )
  SELECT aa.loc
  FROM anc aa
  JOIN anc bb ON bb.which = 'b' AND bb.loc = aa.loc   -- common ancestor of both scenes
  WHERE aa.which = 'a'
  ORDER BY aa.up ASC   -- fewest steps up from A = closest to A = the DEEPEST common ancestor
  LIMIT 1;
$$;

-- fn_distance: Euclidean meters between p_a and p_b in their nearest common parent's local frame (§3).
--   * Same-scene (an entity's scene == the common parent): use the entity's OWN in-scene coordinates
--     (actor/artifact position within the scene, or a location's own coordinate-in-parent).
--   * Cross-level (an entity's scene is deeper than the common parent): use the coordinate of that
--     entity's ancestor location which is the DIRECT CHILD of the common parent -- the child-location's
--     coordinate at the parent frame.
--
-- ACCEPTED IMPRECISION (§3 -- do NOT "fix" this): the cross-level distance (e.g. bar -> alley) ignores
-- the walk-to-the-tavern-door part. Correcting it would require inferring an exit point, which is
-- reasoning, which is an LLM -- and this path is deliberately LLM-free. The hierarchy makes the error
-- marginal by construction: a child location is small inside its parent -- "15 seconds of barroom
-- inside a 10-minute walk." An agent who "improves" this with exit-point inference is breaking the
-- design (see also §5.3: Portal is accessibility, NOT geometry).
--
-- NULL/missing coordinate -> treated as the origin {x:0,y:0} of its frame. Pre-spine rows (seeded
-- before this migration) converge on coordinates when the LLM mints them; the hand-authored seed world
-- (a sanctioned test artifact, §3) supplies all coordinates directly.
CREATE FUNCTION public.fn_distance(p_world_id uuid, p_a uuid, p_b uuid)
RETURNS numeric
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  ncp AS (
    SELECT fn_nearest_common_parent(p_world_id, p_a, p_b) AS loc
  ),
  ent AS (
    -- each entity's kind, its scene, and its OWN coordinates (from whichever state table it lives in)
    SELECT x.which, x.eid,
      CASE er.entity_kind
        WHEN 'location' THEN x.eid
        WHEN 'actor'    THEN (ast.attrs->>'location_id')::uuid
        WHEN 'artifact' THEN COALESCE((art.attrs->>'location_id')::uuid, er.current_scene_id)
      END AS scene,
      COALESCE(ast.attrs->'coordinates', art.attrs->'coordinates', ls.attrs->'coordinates') AS own_coord
    FROM (VALUES ('a', p_a), ('b', p_b)) AS x(which, eid)
    JOIN entity_registry er ON er.world_id = p_world_id AND er.entity_id = x.eid
    LEFT JOIN actor_state    ast ON ast.world_id = p_world_id AND ast.entity_id = x.eid
    LEFT JOIN artifact_state art ON art.world_id = p_world_id AND art.entity_id = x.eid
    LEFT JOIN location_state  ls ON  ls.world_id = p_world_id AND  ls.entity_id = x.eid
  ),
  climb AS (
    -- ancestor-or-self chain of each entity's scene, stopping once we reach the common parent
    SELECT e.which, e.scene AS loc
    FROM ent e
    UNION ALL
    SELECT c.which, (ls.attrs->>'parent_location_id')::uuid
    FROM climb c
    JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
      AND c.loc IS DISTINCT FROM (SELECT loc FROM ncp)   -- never climb above the common parent
  ),
  frame_coord AS (
    SELECT e.which,
      CASE
        WHEN e.scene = (SELECT loc FROM ncp)
          THEN e.own_coord   -- same frame: the entity's own coordinate is already parent-local
        ELSE (
          -- cross-level: the ancestor location that is the direct child of the common parent
          SELECT ls.attrs->'coordinates'
          FROM climb c
          JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = c.loc
          WHERE c.which = e.which
            AND (ls.attrs->>'parent_location_id')::uuid = (SELECT loc FROM ncp)
          LIMIT 1
        )
      END AS coord
    FROM ent e
  )
  SELECT sqrt(
      power(COALESCE((ca.coord->>'x')::numeric, 0) - COALESCE((cb.coord->>'x')::numeric, 0), 2)
    + power(COALESCE((ca.coord->>'y')::numeric, 0) - COALESCE((cb.coord->>'y')::numeric, 0), 2)
  )
  FROM (SELECT coord FROM frame_coord WHERE which = 'a') ca,
       (SELECT coord FROM frame_coord WHERE which = 'b') cb;
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_distance(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS public.fn_nearest_common_parent(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS public.fn_location_depth(uuid, uuid);
