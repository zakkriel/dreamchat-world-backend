-- migrate:up

-- Living World / Task 7 (Unit 5) — the world-scope payload: fn_world_slice. The World Actor is the ONE
-- seat that sees the whole world, not a scene (unlike every other seat, which reasons over a
-- gather_slice-bounded action or a fn_fact_sheet-bounded set of targets). This migration builds ONLY
-- that bounded payload — Task 8 wires it into the seat's prompt assembly. Pure SQL, no LLM, no seat
-- reference. LANGUAGE sql STABLE (reads current state; no writes) — mirrors fn_fact_sheet's
-- jsonb_build_object + jsonb_agg construction style (migration 20260730100001_fact_sheet.sql).
--
-- TRUTH-side / world-omniscient — do NOT perception-scope. Unlike fn_fact_sheet's p_truth_side flavor
-- split, this function has only one flavor: truth. It shows CANONICAL names and real state; it never
-- calls fn_display_name and never gates on a viewer's knowledge (RULINGS-2026-07-23 §9 — the perception
-- walls protect the character-mind seats, never the referee, and the World Actor is world-omniscient by
-- role). Perception happens later, as a separate engine step, when the authored intrusion fans out
-- through the normal commit path (design doc Unit 5) — never inside this read.
--
-- Bounded, never O(world²): ledger/presence/locations are each one row per entity (no cross-join), and
-- `recent` is a hard LIMIT-ed tail — never the whole history.
--
-- Returns jsonb_build_object('ledger', …, 'presence', …, 'locations', …, 'recent', …, 'scene', …):
--   ledger    = pending_event rows for this world with status='pending' — the scheduled-events ledger
--               (Unit 3) the World Actor's authored intrusion composes against. One entry per row:
--               pending_id/fire_at_tick/magnitude/payload.
--   presence  = {actor, location} for EVERY actor in the world (not just the current scene) — read via
--               actor_state.attrs.location_id, the SAME path fn_target_position (kind='actor' branch)
--               and fn_actors_at read, so this never disagrees with the move gate about where an actor
--               is. This is the presence-boundary-mover's reach: the World Actor may pull a non-present
--               NPC into the scene (design doc Unit 5), which requires seeing where every NPC actually is.
--   locations = the world's locations: one entry per entity_registry row with entity_kind='location',
--               LEFT JOINed to location_state for its attrs (coordinates, parent_location_id, tension,
--               description, …) — the same {id,name,attrs} shape `scene` below uses, so the seat can
--               treat any locations[] entry and `scene` interchangeably.
--   recent    = a BOUNDED tail of recent world canon: accepted canon_event rows, ORDER BY in_world_tick
--               DESC, beat_seq DESC, LIMIT 20. status='accepted' — a proposed/rejected/superseded row
--               is not canon yet/anymore, so it is never "recent world canon." Bounded — never the
--               world's whole history.
--   scene     = the current location object for p_scene, nested (NOT an id) — so the seat CAN aim its
--               authored intrusion at the player's own scene (v1 scope, design doc Unit 5) without a
--               second lookup. Same {id,name,attrs} shape as a `locations` entry; json null if p_scene
--               does not resolve to a live location row (caller error — never silently fabricated).
CREATE FUNCTION public.fn_world_slice(p_world_id uuid, p_scene uuid)
RETURNS jsonb
LANGUAGE sql STABLE AS $$
  WITH loc AS (
    -- The world's location rows ({id,name,attrs}) computed ONCE; both `locations`
    -- (all of them) and `scene` (the p_scene one) select from this, instead of
    -- running the same entity_registry⋈location_state join twice.
    SELECT er.entity_id AS id, er.canonical_name AS name, COALESCE(ls.attrs, '{}'::jsonb) AS attrs
    FROM entity_registry er
    LEFT JOIN location_state ls
      ON ls.world_id = er.world_id AND ls.entity_id = er.entity_id
    WHERE er.world_id = p_world_id AND er.entity_kind = 'location'
  )
  SELECT jsonb_build_object(
    'ledger',
    COALESCE(
      (SELECT jsonb_agg(jsonb_build_object(
                 'pending_id',   pe.pending_id,
                 'fire_at_tick', pe.fire_at_tick,
                 'magnitude',    pe.magnitude,
                 'payload',      pe.payload
               ) ORDER BY pe.fire_at_tick)
       FROM pending_event pe
       WHERE pe.world_id = p_world_id AND pe.status = 'pending'),
      '[]'::jsonb),

    'presence',
    COALESCE(
      (SELECT jsonb_agg(jsonb_build_object(
                 'actor',    a.entity_id,
                 'location', (a.attrs->>'location_id')::uuid
               ) ORDER BY a.entity_id)
       FROM actor_state a
       WHERE a.world_id = p_world_id),
      '[]'::jsonb),

    'locations',
    COALESCE(
      (SELECT jsonb_agg(jsonb_build_object(
                 'id',    loc.id,
                 'name',  loc.name,
                 'attrs', loc.attrs
               ) ORDER BY loc.id)
       FROM loc),
      '[]'::jsonb),

    'recent',
    COALESCE(
      (SELECT jsonb_agg(r.obj ORDER BY r.in_world_tick DESC, r.beat_seq DESC)
       FROM (
         SELECT ce.in_world_tick, ce.beat_seq,
                jsonb_build_object(
                  'event_id',      ce.event_id,
                  'event_type',    ce.event_type,
                  'summary',       ce.summary,
                  'in_world_tick', ce.in_world_tick,
                  'beat_seq',      ce.beat_seq
                ) AS obj
         FROM canon_event ce
         WHERE ce.world_id = p_world_id AND ce.status = 'accepted'
         ORDER BY ce.in_world_tick DESC, ce.beat_seq DESC
         LIMIT 20
       ) r),
      '[]'::jsonb),

    'scene',
    (SELECT jsonb_build_object(
               'id',    loc.id,
               'name',  loc.name,
               'attrs', loc.attrs
             )
     FROM loc WHERE loc.id = p_scene)
  );
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_world_slice(uuid, uuid);
