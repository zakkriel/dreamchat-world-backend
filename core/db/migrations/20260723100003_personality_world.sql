-- migrate:up

-- Personality Module shapes (Station G fills behavior; RULINGS §8.1, §8.5)
-- World Actor ledger shapes (Station H fills behavior; RULINGS §7a, §7b)

-- 1) personality_core: one row per actor, malleability is a measurement (0 < m ≤ 1)
CREATE TABLE personality_core (
  world_id    uuid    NOT NULL,
  actor_id    uuid    PRIMARY KEY,
  traits      jsonb   NOT NULL,
  malleability numeric NOT NULL CHECK (malleability > 0 AND malleability <= 1)
);

-- 2) trait_provenance: backstory canon events explaining each trait (RULINGS §8.1)
CREATE TABLE trait_provenance (
  world_id  uuid NOT NULL,
  actor_id  uuid NOT NULL,
  trait_key text NOT NULL,
  event_id  uuid REFERENCES canon_event(event_id),
  PRIMARY KEY (actor_id, trait_key, event_id)
);

-- 3) trait_pool: sub-threshold accumulation (RULINGS §8.5)
CREATE TABLE trait_pool (
  world_id  uuid    NOT NULL,
  actor_id  uuid    NOT NULL,
  trait_key text    NOT NULL,
  accrued   numeric NOT NULL DEFAULT 0,
  threshold numeric NOT NULL,
  PRIMARY KEY (actor_id, trait_key)
);

-- 4) pending_event: the world-actor ledger (RULINGS §7a)
CREATE TABLE pending_event (
  pending_id   uuid    PRIMARY KEY,
  world_id     uuid    NOT NULL,
  fire_at_tick bigint  NOT NULL,
  magnitude    text    NOT NULL CHECK (magnitude IN ('small','medium','large')),
  payload      jsonb   NOT NULL,
  status       text    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','fired','cancelled'))
);

-- 5) world_pressure: independent pools, one row per magnitude tier (RULINGS §7b)
CREATE TABLE world_pressure (
  world_id        uuid   NOT NULL,
  tier            text   CHECK (tier IN ('small','medium','large')),
  accrued         numeric NOT NULL DEFAULT 0,
  last_fired_tick bigint  NOT NULL DEFAULT 0,
  PRIMARY KEY (world_id, tier)
);

-- 6) fn_due_pending: return pending events that are due at or before p_tick
CREATE FUNCTION fn_due_pending(p_world_id uuid, p_tick bigint)
RETURNS SETOF pending_event LANGUAGE sql STABLE AS $$
  SELECT * FROM pending_event
  WHERE world_id = p_world_id AND status = 'pending' AND fire_at_tick <= p_tick
  ORDER BY fire_at_tick;
$$;

-- migrate:down

DROP FUNCTION IF EXISTS fn_due_pending(uuid, bigint);
DROP TABLE IF EXISTS world_pressure;
DROP TABLE IF EXISTS pending_event;
DROP TABLE IF EXISTS trait_pool;
DROP TABLE IF EXISTS trait_provenance;
DROP TABLE IF EXISTS personality_core;
