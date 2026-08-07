-- migrate:up

-- Rung 3 / Task 0 — correction to the rung 2 journey work: a watch ("wait until the ship is
-- in") needs a horizon so nothing waits forever. startJourney (20260807100003_journey.sql)
-- borrowed fn_duration_class_seconds(world, 'extremely_long') for this — two hours, and a
-- duration class for a spoken line's length, not a vigil's ceiling. Same per-world-table-
-- with-built-in-fallback shape as fn_duration_class_seconds/fn_journey_legs, its own dial:
-- watch_horizon(world_id PK, horizon_seconds), retunable per world; a built-in fallback
-- covers unseeded worlds so the function never fails closed. Default 86400 — a day, which
-- reads as a horizon rather than a speech length.

CREATE TABLE watch_horizon (
  world_id        uuid PRIMARY KEY,
  horizon_seconds bigint NOT NULL CHECK (horizon_seconds > 0)
);

CREATE FUNCTION fn_watch_horizon_seconds(p_world_id uuid) RETURNS bigint
  LANGUAGE sql STABLE AS $$
  SELECT COALESCE(
    (SELECT horizon_seconds FROM watch_horizon WHERE world_id = p_world_id),
    86400  -- built-in fallback (retune per-world via the table)
  );
$$;

-- Extend seed_world_defaults (existing function, currently seeds movement_type +
-- status_modifier + duration_class_seconds + world_actor_config + world_actor_setting +
-- extent_class_metres + journey_legs_band — copied verbatim below, current as of
-- 20260807100004_journey_legs.sql) with the one watch_horizon row.
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
  INSERT INTO journey_legs_band (world_id, max_span_seconds, legs)
  VALUES (p_world_id, 3600, 5), (p_world_id, 86400, 7), (p_world_id, 31536000, 10) ON CONFLICT DO NOTHING;
  INSERT INTO watch_horizon (world_id, horizon_seconds)
  VALUES (p_world_id, 86400) ON CONFLICT DO NOTHING;
$$;

-- migrate:down

-- Restore the pre-Task-0 seed_world_defaults (everything through journey_legs_band, no
-- watch_horizon row) FIRST — seed_world_defaults (LANGUAGE sql) is parsed against the tables
-- it references, so it must stop referencing watch_horizon before the table is dropped, or
-- the DROP TABLE below would error (the Living World migrations' dependency-order lesson).
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
  INSERT INTO journey_legs_band (world_id, max_span_seconds, legs)
  VALUES (p_world_id, 3600, 5), (p_world_id, 86400, 7), (p_world_id, 31536000, 10) ON CONFLICT DO NOTHING;
$$;

DROP FUNCTION IF EXISTS fn_watch_horizon_seconds(uuid);
DROP TABLE IF EXISTS watch_horizon;
