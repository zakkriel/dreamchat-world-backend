-- migrate:up

-- Station F / FINAL-action-contracts.md §8 (Minting: validation and the three nets). Makes minting REAL:
-- until now the `mints` ruling slot was decoded but INERT — no committed path inserted a movement_type or
-- status_modifier, so the move/relocate arithmetic ran only against the two SEEDED rows. apply_mint is the
-- persistence half (the Go validateMints is the shape+bounds half). It runs INSIDE the ruling's single tx,
-- AFTER verdict passes, audit-trailed to the ruling event — so a mint's blast radius is ONE logged row
-- with provenance (net 2), and every mint traces to the ruling that produced it (net 3).
--
-- Units are fixed system-wide (meters, seconds, kilograms). apply_mint enforces NO plausibility bound
-- ("is 400 m/s too fast?"): plausibility was already judged (the mint only exists because an adjudicated
-- ruling passed the reality check, net 1). SHAPE + DERIVABLE BOUNDS were checked in Go before commit.

-- ── (1) Provenance columns. movement_type and status_modifier had no audit-trail column; net 3 requires
--    every mint trace to its ruling. Add created_by_event (nullable — seeded rows have no event) mirroring
--    entity_registry.created_by_event. A minted row's provenance is the ruling event; the compensating
--    path (net 2: "repaired by compensating event/merge, never deleted") can always find the origin.
ALTER TABLE movement_type
  ADD COLUMN created_by_event uuid REFERENCES canon_event(event_id);
ALTER TABLE status_modifier
  ADD COLUMN created_by_event uuid REFERENCES canon_event(event_id);

-- ── (2) apply_mint: persist ONE mint. Discriminates the mint kind by its JSON shape/fields — the SAME
--    discriminator validateMints uses (core/api/mint.go), documented there:
--      has 'baseSpeed'                         → MOVEMENT-TYPE mint  → movement_type
--      has 'statusTypeId' | 'movementModifiers'→ MODIFIER mint       → status_modifier (one row per
--                                                                       movementModifiers entry; the FK
--                                                                       (world, movement_type) is satisfied
--                                                                       by mint-ordering — the referenced
--                                                                       movement type was minted by an
--                                                                       earlier apply_mint IN THIS SAME tx)
--      artifact/place shape                    → RAISE (see below)
--
--    ESCALATED (task-6 report §escalation): persisting a typed ARTIFACT mint (entity_registry + which
--    state table + descriptor/canonical_name/entity_kind + attrs mapping for size/coordinate/parent/
--    max_room/connects) is NOT specified in §8, is covered by no test, and overlaps the still-unimplemented
--    EntityCreated event path. Rather than GUESS that schema, apply_mint RAISEs on an artifact-shaped mint.
--    validateMints still accepts a well-formed artifact mint's SHAPE, so the gate can already reject a
--    malformed one; the RAISE here rolls back the whole ruling (fail-safe — a well-formed-but-unpersistable
--    mint corrupts nothing) and is the marker for the future task that lands the artifact-persistence schema.
--
--    Returns the minted id (movement_type_id or status_type_id) as text — useful to callers/tests; the
--    orchestrator ignores it. ON CONFLICT DO NOTHING makes a re-mint of an existing row idempotent
--    (reuse-before-create: minting an id that already exists is a no-op, not an error).
CREATE FUNCTION public.apply_mint(p_world_id uuid, p_ruling_event uuid, p_mint jsonb)
RETURNS text
LANGUAGE plpgsql AS $$
DECLARE
  v_action text;
  v_row    jsonb;
  v_status text;
BEGIN
  -- MOVEMENT-TYPE mint: { movementTypeId, baseSpeed }.
  IF p_mint ? 'baseSpeed' THEN
    INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps, created_by_event)
    VALUES (p_world_id, p_mint->>'movementTypeId', (p_mint->>'baseSpeed')::numeric, p_ruling_event)
    ON CONFLICT (world_id, movement_type_id) DO NOTHING;
    RETURN p_mint->>'movementTypeId';

  -- MODIFIER mint: { statusTypeId, actionType, movementModifiers:[{ movementTypeId, modifierPercent }] }.
  ELSIF p_mint ? 'statusTypeId' OR p_mint ? 'movementModifiers' THEN
    v_action := COALESCE(p_mint->>'actionType', 'move');   -- 'move' is the only metered action (§6)
    v_status := p_mint->>'statusTypeId';
    FOR v_row IN SELECT jsonb_array_elements(COALESCE(p_mint->'movementModifiers', '[]'::jsonb))
    LOOP
      -- FK (world_id, movement_type_id) → movement_type: satisfied by ordering (validated in Go; the
      -- referenced type was minted by an earlier apply_mint in this tx, or is seeded/committed).
      INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id,
                                   modifier_percent, created_by_event)
      VALUES (p_world_id, v_status, v_action, v_row->>'movementTypeId',
              (v_row->>'modifierPercent')::numeric, p_ruling_event)
      ON CONFLICT (world_id, status_type_id, action_type, movement_type_id) DO NOTHING;
    END LOOP;
    RETURN v_status;

  -- ARTIFACT/PLACE mint: persistence escalated (see header) — refuse to guess the schema.
  ELSIF p_mint ? 'size' OR p_mint ? 'maxRoom' OR p_mint ? 'coordinate'
     OR p_mint ? 'parentLocationId' OR p_mint ? 'locationId' THEN
    RAISE EXCEPTION
      'apply_mint: artifact/place mint persistence is not yet specified (escalated: the entity_registry+state schema is undefined in FINAL §8) — refusing to guess. mint=%',
      p_mint;

  ELSE
    RAISE EXCEPTION 'apply_mint: unrecognized mint shape (no baseSpeed / statusTypeId / artifact fields). mint=%', p_mint;
  END IF;
END $$;

-- ── (3) Carried-forward guard (Task-1 review, item 1b): the parent-walk CTEs in 20260729100001 recurse on
--    parent_location_id with NO cycle/depth guard. THIS migration is where parent_location_id first gets
--    validated on a mint write (validateMints rejects a coordinate/parent cycle at mint time, item 1a); the
--    DEFENSIVE backstop below guarantees that even a bad row that slips through can never infinite-loop the
--    engine. Mirroring causal_bundle_assert_acyclic's depth cap (I-4): cap the recursion at 64 so each
--    function TERMINATES (returns bounded) on a parent_location_id cycle instead of looping forever. These
--    stay LANGUAGE sql (the primary guard is the Go mint validation; this is termination insurance, so a
--    bounded return is the right shape — RAISEing from three hot STABLE read functions is heavier than the
--    backstop warrants). CREATE OR REPLACE here (never edit the shipped migration) — schema.sql regenerates.

CREATE OR REPLACE FUNCTION public.fn_location_depth(p_world_id uuid, p_location uuid)
RETURNS int
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE chain AS (
    SELECT p_location AS loc, 0 AS depth
    UNION ALL
    SELECT (ls.attrs->>'parent_location_id')::uuid, c.depth + 1
    FROM chain c
    JOIN location_state ls
      ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
      AND c.depth < 64   -- defensive depth cap (I-4 mirror): a parent_location_id cycle can't infinite-loop
  )
  SELECT max(depth) FROM chain;
$$;

CREATE OR REPLACE FUNCTION public.fn_nearest_common_parent(p_world_id uuid, p_a uuid, p_b uuid)
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
      AND a.up < 64   -- defensive depth cap (I-4 mirror)
  )
  SELECT aa.loc
  FROM anc aa
  JOIN anc bb ON bb.which = 'b' AND bb.loc = aa.loc
  WHERE aa.which = 'a'
  ORDER BY aa.up ASC
  LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION public.fn_distance(p_world_id uuid, p_a uuid, p_b uuid)
RETURNS numeric
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  ncp AS (
    SELECT fn_nearest_common_parent(p_world_id, p_a, p_b) AS loc
  ),
  ent AS (
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
    SELECT e.which, e.scene AS loc, 0 AS up
    FROM ent e
    UNION ALL
    SELECT c.which, (ls.attrs->>'parent_location_id')::uuid, c.up + 1
    FROM climb c
    JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
      AND c.loc IS DISTINCT FROM (SELECT loc FROM ncp)
      AND c.up < 64   -- defensive depth cap (I-4 mirror)
  ),
  frame_coord AS (
    SELECT e.which,
      CASE
        WHEN e.scene = (SELECT loc FROM ncp)
          THEN e.own_coord
        ELSE (
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

DROP FUNCTION IF EXISTS public.apply_mint(uuid, uuid, jsonb);
ALTER TABLE status_modifier DROP COLUMN IF EXISTS created_by_event;
ALTER TABLE movement_type   DROP COLUMN IF EXISTS created_by_event;

-- Restore the UNCAPPED parent-walk CTEs (verbatim from 20260729100001_coordinates_distance.sql).
CREATE OR REPLACE FUNCTION public.fn_location_depth(p_world_id uuid, p_location uuid)
RETURNS int
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE chain AS (
    SELECT p_location AS loc, 0 AS depth
    UNION ALL
    SELECT (ls.attrs->>'parent_location_id')::uuid, c.depth + 1
    FROM chain c
    JOIN location_state ls
      ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
  )
  SELECT max(depth) FROM chain;
$$;

CREATE OR REPLACE FUNCTION public.fn_nearest_common_parent(p_world_id uuid, p_a uuid, p_b uuid)
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
  JOIN anc bb ON bb.which = 'b' AND bb.loc = aa.loc
  WHERE aa.which = 'a'
  ORDER BY aa.up ASC
  LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION public.fn_distance(p_world_id uuid, p_a uuid, p_b uuid)
RETURNS numeric
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  ncp AS (
    SELECT fn_nearest_common_parent(p_world_id, p_a, p_b) AS loc
  ),
  ent AS (
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
    SELECT e.which, e.scene AS loc
    FROM ent e
    UNION ALL
    SELECT c.which, (ls.attrs->>'parent_location_id')::uuid
    FROM climb c
    JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
      AND c.loc IS DISTINCT FROM (SELECT loc FROM ncp)
  ),
  frame_coord AS (
    SELECT e.which,
      CASE
        WHEN e.scene = (SELECT loc FROM ncp)
          THEN e.own_coord
        ELSE (
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
