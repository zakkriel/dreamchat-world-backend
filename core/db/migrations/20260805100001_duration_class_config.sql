-- migrate:up

-- Living World / Task 2 — per-world duration_class → seconds config, and the
-- fn_duration_class_seconds lookup the Go orchestrator (Task 3) will call to turn
-- a non-move attempt's duration_class (Task 1's parse-shape enum, instant|short|
-- medium|long|extremely_long) into a concrete number of seconds. Per-world
-- retunable via this table; a built-in fallback covers unseeded worlds so the
-- function never fails closed.

CREATE TABLE duration_class_seconds (
  world_id uuid NOT NULL,
  class    text NOT NULL CHECK (class IN ('instant','short','medium','long','extremely_long')),
  seconds  bigint NOT NULL CHECK (seconds > 0),
  PRIMARY KEY (world_id, class)
);

CREATE FUNCTION fn_duration_class_seconds(p_world_id uuid, p_class text) RETURNS bigint
  LANGUAGE sql STABLE AS $$
  SELECT COALESCE(
    (SELECT seconds FROM duration_class_seconds WHERE world_id = p_world_id AND class = p_class),
    CASE p_class  -- built-in fallback (retune per-world via the table)
      WHEN 'instant' THEN 2 WHEN 'short' THEN 5 WHEN 'medium' THEN 60
      WHEN 'long' THEN 300 WHEN 'extremely_long' THEN 7200 ELSE 2 END
  );
$$;

-- Extend seed_world_defaults (existing function, currently seeds movement_type +
-- status_modifier — copied verbatim below) with the five duration_class rows, so
-- every newly-seeded world gets a default duration_class_seconds config.
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
$$;

-- migrate:down

-- Restore the pre-Task-2 seed_world_defaults (movement_type + status_modifier only).
CREATE OR REPLACE FUNCTION public.seed_world_defaults(p_world_id uuid) RETURNS void
    LANGUAGE sql
    AS $$
  INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps)
  VALUES (p_world_id, 'walk', 1.4) ON CONFLICT DO NOTHING;
  INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
  VALUES (p_world_id, 'encumbered', 'move', 'walk', -100) ON CONFLICT DO NOTHING;
$$;

DROP FUNCTION IF EXISTS fn_duration_class_seconds(uuid, text);
DROP TABLE IF EXISTS duration_class_seconds;
