-- migrate:up

-- SPEC-031 — make interruption a real presence. FOUNDER-RULED (2026-08-08): medium's
-- climb_chunk_ticks 3600 → 300.
--
-- Why this number and not another. Only medium and large eruptions cut a beat (eruptionCutsBeat,
-- worldturn.go); a small fire commits an event but never interrupts. The chance a tier fires is
--
--   LEAST(cap, climb_rate * ((now - last_fired) / climb_chunk_ticks) * intensity)
--
-- so with medium at 3600 ticks per climb, and a journey leg advancing 60 ticks from a seed that
-- starts near tick 50, medium sat at roughly a quarter of one percent per leg. The frontend's gate
-- run saw zero interruptions across fourteen legs, which was not a defect and not bad luck — it is
-- what the arithmetic says: on the order of four hundred legs for an even chance. A world that
-- interrupts a traveller about once a career.
--
-- At 300 ticks per climb, medium reaches ~3% per leg at the ticks a real session occupies — about
-- one interruption every two journeys, the founder's stated target. large is deliberately LEFT ALONE
-- at 86400: it is the rare, heavy event, and the point of tuning medium is that the COMMON
-- interruption should be the medium one.
--
-- This is a felt-experience setting, not a correctness one. The founder plays at this value and
-- re-tunes from here, so both halves below exist to make re-tuning a one-line change:
--   1. seed_world_defaults, so every NEW world is born with it;
--   2. an UPDATE, because the defaults insert ON CONFLICT DO NOTHING and would therefore never
--      touch the already-seeded play world — the only world anyone is actually playing.
-- Forgetting the second half is how a tuning change lands green and changes nothing.
--
-- NOTE ON THE FUNCTION BODY: seed_world_defaults is EXTENDED by successive migrations, each copying
-- the previous body verbatim and appending one INSERT. The body below is current as of
-- 20260807100005_watch_horizon.sql (movement_type, status_modifier, duration_class_seconds,
-- world_actor_config, world_actor_setting, extent_class_metres, journey_legs_band, watch_horizon)
-- with ONLY the medium tier's chunk changed. Copying an older body silently reverts every later
-- extension — the first draft of this migration did exactly that and dropped the watch_horizon row,
-- which its own pgTAP test caught.

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
         (p_world_id, 'medium', 0.01, 300, 0.70),
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

UPDATE world_actor_config SET climb_chunk_ticks = 300
 WHERE tier = 'medium' AND climb_chunk_ticks = 3600;

-- migrate:down

UPDATE world_actor_config SET climb_chunk_ticks = 3600
 WHERE tier = 'medium' AND climb_chunk_ticks = 300;

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
