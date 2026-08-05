-- migrate:up

-- Living World / Task 5 — per-world pressure config + append-only fire-log +
-- fn_pressure_chance. The World Actor erupts on rising per-tier "pressure."
-- Pressure is DERIVED — how much world-time has passed since that tier last
-- erupted — never a stored counter. world_actor_config holds the per-tier
-- climb curve, world_actor_setting is the per-world on/off + intensity dial,
-- and world_eruption is the append-only fire-log (the last-eruption source +
-- audit trail). Later tasks (6/8/9) roll against fn_pressure_chance and write
-- world_eruption rows on fire; this task only creates and reads them.

CREATE TABLE world_actor_config (
  world_id uuid NOT NULL,
  tier text NOT NULL CHECK (tier IN ('small','medium','large')),
  climb_rate numeric NOT NULL CHECK (climb_rate >= 0),        -- chance added per climb_chunk of world-time
  climb_chunk_ticks bigint NOT NULL CHECK (climb_chunk_ticks > 0),  -- one "climb" = this many ticks
  cap numeric NOT NULL CHECK (cap >= 0 AND cap <= 1),
  PRIMARY KEY (world_id, tier)
);
CREATE TABLE world_actor_setting (
  world_id uuid PRIMARY KEY,
  enabled boolean NOT NULL DEFAULT true,
  intensity numeric NOT NULL DEFAULT 1.0 CHECK (intensity >= 0)
);
CREATE TABLE world_eruption (           -- append-only: the last-eruption source + fire audit log
  world_id uuid NOT NULL,
  tier text NOT NULL CHECK (tier IN ('small','medium','large')),
  fired_tick bigint NOT NULL,
  event_id uuid NOT NULL
);
CREATE INDEX idx_world_eruption_lookup ON world_eruption (world_id, tier, fired_tick DESC);

CREATE FUNCTION fn_pressure_chance(p_world_id uuid, p_tier text, p_now bigint) RETURNS numeric
  LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN COALESCE((SELECT enabled FROM world_actor_setting WHERE world_id=p_world_id), true) IS FALSE
              THEN 0
         ELSE LEAST(c.cap,
                    c.climb_rate * ((p_now - COALESCE(
                      (SELECT max(fired_tick) FROM world_eruption WHERE world_id=p_world_id AND tier=p_tier), 0
                    ))::numeric / c.climb_chunk_ticks))
              * COALESCE((SELECT intensity FROM world_actor_setting WHERE world_id=p_world_id), 1.0)
         END
  FROM world_actor_config c WHERE c.world_id=p_world_id AND c.tier=p_tier;
$$;

-- Extend seed_world_defaults (existing function, currently seeds movement_type +
-- status_modifier + duration_class_seconds — copied verbatim below) with the
-- three pressure tiers + one world_actor_setting row, so every newly-seeded
-- world gets a default pressure config. small climbs fast in small chunks,
-- medium slower, large slowest; all share cap 0.70.
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

-- migrate:down

-- Restore the pre-Task-5 seed_world_defaults (movement_type + status_modifier
-- + duration_class_seconds only) FIRST — seed_world_defaults (LANGUAGE sql) is
-- parsed against the tables it references, so it must stop referencing the
-- pressure tables before they're dropped, or the DROP TABLE below would error.
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

DROP FUNCTION IF EXISTS fn_pressure_chance(uuid, text, bigint);
DROP TABLE IF EXISTS world_eruption;
DROP TABLE IF EXISTS world_actor_setting;
DROP TABLE IF EXISTS world_actor_config;
