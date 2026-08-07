-- migrate:up

-- Rung 1 (the Journey ladder) / Task 3 — SIZE CLASSES: the engine draws the footprint.
--
-- Design §4.5, founder ruling R3: "a coordinate is a single point in space… if your coordinates are
-- within an area, you are actually somewhere." No language model ever draws geometry — the author picks
-- a size class, the engine turns it into metres and draws the footprint. Identical split to time
-- (fn_duration_class_seconds, migration 20260805100001): the author owns intent, the engine owns units.
--
-- Class names are genre-agnostic (rule GA-2): intimate|small|medium|large|vast read sensibly in a
-- sci-fi thriller, a workplace drama, or a horror story alike — never hamlet/town/city.

CREATE TABLE extent_class_metres (
  world_id uuid NOT NULL,
  class    text NOT NULL CHECK (class IN ('intimate','small','medium','large','vast')),
  radius_m numeric NOT NULL CHECK (radius_m > 0),
  PRIMARY KEY (world_id, class)
);

CREATE FUNCTION fn_extent_class_metres(p_world_id uuid, p_class text) RETURNS numeric
  LANGUAGE sql STABLE AS $$
  SELECT COALESCE(
    (SELECT radius_m FROM extent_class_metres WHERE world_id = p_world_id AND class = p_class),
    CASE p_class  -- built-in fallback (retune per-world via the table) — never fails closed
      WHEN 'intimate' THEN 5 WHEN 'small' THEN 50 WHEN 'medium' THEN 200
      WHEN 'large' THEN 1000 WHEN 'vast' THEN 5000 ELSE 50 END
  );
$$;

-- fn_area_around: the engine draws a regular 8-point outline of p_radius metres around p_centre, ready
-- to write straight into attrs.area. Deterministic, no randomness — the same centre + radius always
-- draws the same footprint. The round trip (fn_area_polygon(fn_area_around(...)) containing the centre)
-- is what rung 2 depends on: a created place must contain the traveller standing at the point it was
-- created for. cosd/sind (degrees-native trig) are built into Postgres — verified against the running
-- server, no extension needed.
CREATE FUNCTION public.fn_area_around(p_centre jsonb, p_radius numeric) RETURNS jsonb
LANGUAGE sql IMMUTABLE AS $$
  SELECT jsonb_build_object('points', jsonb_agg(
           jsonb_build_object(
             'x', round(((p_centre->>'x')::numeric + p_radius * cosd(45 * g))::numeric, 3),
             'y', round(((p_centre->>'y')::numeric + p_radius * sind(45 * g))::numeric, 3))
           ORDER BY g))
  FROM generate_series(0, 7) AS g;
$$;

-- Extend seed_world_defaults (existing function, currently seeds movement_type + status_modifier +
-- duration_class_seconds + world_actor_config + world_actor_setting — copied verbatim below) with the
-- five extent_class_metres rows, so every newly-seeded world gets a default size-class config.
CREATE OR REPLACE FUNCTION public.seed_world_defaults(p_world_id uuid) RETURNS void
    LANGUAGE sql
    AS $$
  INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps)
  VALUES (p_world_id, 'walk', 1.4) ON CONFLICT DO NOTHING;
  INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
  VALUES (p_world_id, 'encumbered', 'move', 'walk', -100) ON CONFLICT DO NOTHING;
  INSERT INTO duration_class_seconds (world_id, class, seconds)
  VALUES (p_world_id, 'instant', 2), (p_world_id, 'short', 5), (p_world_id, 'medium', 60),
         (p_world_id, 'long', 300), (p_world_id, 'extremely_long', 7200) ON CONFLICT DO NOTHING;
  INSERT INTO world_actor_config (world_id, tier, climb_rate, climb_chunk_ticks, cap)
  VALUES (p_world_id, 'small', 0.01, 60, 0.70),
         (p_world_id, 'medium', 0.01, 3600, 0.70),
         (p_world_id, 'large', 0.01, 86400, 0.70) ON CONFLICT DO NOTHING;
  INSERT INTO world_actor_setting (world_id) VALUES (p_world_id) ON CONFLICT DO NOTHING;
  INSERT INTO extent_class_metres (world_id, class, radius_m)
  VALUES (p_world_id, 'intimate', 5), (p_world_id, 'small', 50), (p_world_id, 'medium', 200),
         (p_world_id, 'large', 1000), (p_world_id, 'vast', 5000) ON CONFLICT DO NOTHING;
$$;

-- migrate:down

-- Restore the pre-Task-3 seed_world_defaults (movement_type + status_modifier + duration_class_seconds
-- + world_actor_config + world_actor_setting only) FIRST — seed_world_defaults (LANGUAGE sql) is parsed
-- against the tables it references, so it must stop referencing extent_class_metres before the table is
-- dropped, or the DROP TABLE below would error (the Living World migrations' dependency-order lesson).
CREATE OR REPLACE FUNCTION public.seed_world_defaults(p_world_id uuid) RETURNS void
    LANGUAGE sql
    AS $$
  INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps)
  VALUES (p_world_id, 'walk', 1.4) ON CONFLICT DO NOTHING;
  INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
  VALUES (p_world_id, 'encumbered', 'move', 'walk', -100) ON CONFLICT DO NOTHING;
  INSERT INTO duration_class_seconds (world_id, class, seconds)
  VALUES (p_world_id, 'instant', 2), (p_world_id, 'short', 5), (p_world_id, 'medium', 60),
         (p_world_id, 'long', 300), (p_world_id, 'extremely_long', 7200) ON CONFLICT DO NOTHING;
  INSERT INTO world_actor_config (world_id, tier, climb_rate, climb_chunk_ticks, cap)
  VALUES (p_world_id, 'small', 0.01, 60, 0.70),
         (p_world_id, 'medium', 0.01, 3600, 0.70),
         (p_world_id, 'large', 0.01, 86400, 0.70) ON CONFLICT DO NOTHING;
  INSERT INTO world_actor_setting (world_id) VALUES (p_world_id) ON CONFLICT DO NOTHING;
$$;

DROP FUNCTION IF EXISTS fn_area_around(jsonb, numeric);
DROP FUNCTION IF EXISTS fn_extent_class_metres(uuid, text);
DROP TABLE IF EXISTS extent_class_metres;
