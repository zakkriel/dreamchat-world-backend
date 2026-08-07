-- migrate:up

-- Rung 2 (the Journey ladder) / Task 2 — THE LEG COUNT IS DATA.
--
-- Founder ruling R7: a journey is 5-10 presses, whatever its length — the low end for short trips, the
-- high end for long hauls. The *risk* per press is what scales with span, and that already happens on
-- its own: pressure climbs with elapsed world-time (fn_pressure_chance, migration 20260805100002), so a
-- leg covering eighteen hours sits at the cap while a leg covering ninety seconds barely moves the
-- needle. This table only owns how many presses a span is cut into. Same split as time
-- (fn_duration_class_seconds, 20260805100001) and size (fn_extent_class_metres, 20260807100002): a
-- per-world table with a built-in fallback so an unseeded world never fails closed.

CREATE TABLE journey_legs_band (
  world_id          uuid NOT NULL,
  max_span_seconds  bigint NOT NULL CHECK (max_span_seconds > 0),  -- the upper bound this row applies to
  legs              int NOT NULL CHECK (legs > 0),
  PRIMARY KEY (world_id, max_span_seconds)
);

-- fn_journey_legs: the smallest band whose bound the span fits, falling back to a built-in ladder, ALWAYS
-- clamped to 5..10 — a bad config row (or a bad fallback edit) can never hand a journey 400 presses or 1.
CREATE FUNCTION fn_journey_legs(p_world_id uuid, p_span_seconds bigint) RETURNS int
  LANGUAGE sql STABLE AS $$
  SELECT GREATEST(5, LEAST(10, COALESCE(
    (SELECT legs FROM journey_legs_band
       WHERE world_id = p_world_id AND max_span_seconds >= p_span_seconds
       ORDER BY max_span_seconds ASC LIMIT 1),
    CASE  -- built-in fallback (retune per-world via the table)
      WHEN p_span_seconds <= 3600 THEN 5     -- <= 1 hour
      WHEN p_span_seconds <= 86400 THEN 7    -- <= 1 day
      ELSE 10
    END
  )));
$$;

-- Extend seed_world_defaults (existing function, currently seeds movement_type + status_modifier +
-- duration_class_seconds + world_actor_config + world_actor_setting + extent_class_metres — copied
-- verbatim below) with the three band rows, mirroring the built-in fallback ladder so a seeded world's
-- default behaviour matches an unseeded one until someone retunes it.
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

-- migrate:down

-- Restore the pre-Task-2 seed_world_defaults (everything through extent_class_metres, no bands) FIRST —
-- seed_world_defaults (LANGUAGE sql) is parsed against the tables it references, so it must stop
-- referencing journey_legs_band before the table is dropped, or the DROP TABLE below would error (the
-- Living World migrations' dependency-order lesson).
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

DROP FUNCTION IF EXISTS fn_journey_legs(uuid, bigint);
DROP TABLE IF EXISTS journey_legs_band;
