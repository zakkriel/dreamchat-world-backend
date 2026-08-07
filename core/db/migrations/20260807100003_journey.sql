-- migrate:up

-- Rung 2 — THE JOURNEY's loop state, and a clock that can see it.
--
-- A journey is LOOP STATE, not canon — the same standing as held_outcome (20260724110004): Go writes it
-- with plain INSERT/UPDATE, rows may be deleted, and there is no append-only guard. Canon still flows
-- only through the apply twins (D-1). What makes it different from held_outcome is lifespan: a held
-- outcome resolves on the next input, a journey spans many.

CREATE TABLE journey (
  journey_id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id     uuid NOT NULL,
  actor_id     uuid NOT NULL,
  kind         text NOT NULL CHECK (kind IN ('travel','wait','watch')),
  threshold    jsonb NOT NULL,           -- the test run at each leg's end; shape per kind (design §4.4)
  span_seconds bigint NOT NULL CHECK (span_seconds > 0),
  legs_total   int NOT NULL CHECK (legs_total > 0),
  legs_done    int NOT NULL DEFAULT 0 CHECK (legs_done >= 0),
  started_tick bigint NOT NULL,
  current_tick bigint NOT NULL,
  frame_id     uuid,                     -- travel: the frame the trip happens in (design §4.5)
  origin_coord jsonb,                    -- travel: where it started, for interpolation
  goal_coord   jsonb,                    -- travel: where it ends
  goal_target  uuid,                     -- travel: the entity being walked to; the arrival commit's target
  stage_id     uuid,                     -- the place currently containing the traveller; NULL = open road
  status       text NOT NULL DEFAULT 'active' CHECK (status IN ('active','arrived','ended'))
);

-- ONE active journey per actor. A second would make "the journey" ambiguous at every decision point, so
-- the database refuses it rather than leaving Go to remember.
CREATE UNIQUE INDEX idx_journey_one_active ON journey (world_id, actor_id) WHERE status = 'active';

-- The next-input lookup scans a world's active journeys; a partial index keeps it to the live rows.
CREATE INDEX idx_journey_active ON journey (world_id) WHERE status = 'active';

-- ── fn_world_now: the world's clock, INCLUDING journeys in flight.
--
-- World-time was derived from committed events alone. A quiet leg commits nothing, so a journey's hours
-- would not move the clock — and eruption pressure is driven entirely by elapsed world-time, so the
-- world could never interrupt a journey. Rather than write filler canon events for "nothing happened",
-- the clock reads the later of the two sources.
--
-- Journeys are NOT filtered by status: an ended journey still holds its tick, because time must never
-- rewind when one stops (B-5, append-only time).
CREATE FUNCTION public.fn_world_now(p_world_id uuid) RETURNS bigint
LANGUAGE sql STABLE AS $$
  SELECT GREATEST(
    COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id = p_world_id), 0),
    COALESCE((SELECT max(current_tick)  FROM journey     WHERE world_id = p_world_id), 0)
  );
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_world_now(uuid);
DROP TABLE IF EXISTS journey;
