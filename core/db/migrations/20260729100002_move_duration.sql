-- migrate:up

-- Station F / FINAL-action-contracts.md §2 (move contract): the REAL move duration replaces the
-- flat-5 stub. Grammar (fixed, engine-owned):
--
--     duration        = distance / effective_speed
--     effective_speed = base_speed(movement_type) * Pi(1 + modifier_percent/100)
--
-- Floor -100% (speed 0 => infinite duration => never fits any budget => the action is prevented). This
-- is how "tied/encumbered → can't move" works: prevention is NOT a special rule, it EMERGES from the
-- arithmetic. NO upper cap (founder ruling): a +900% haste is legal data. This path is deliberately
-- LLM-FREE (D-1): pure arithmetic over stored measurements + minted vocabulary, so the gate can compute
-- a blocker without ever asking a model. Measurements-not-verdicts (§0): speed/duration are COMPUTED at
-- ask-time, never stored.

-- fn_effective_speed: base_speed(movement_type) * Pi(1 + modifier_percent/100) over the actor's ACTIVE
-- statuses. Active statuses = actor_state.attrs->'statuses' (a JSON array of status_type_id strings --
-- the read model this migration introduces) INTERSECTED with status_modifier rows for that movement
-- type. Written in plpgsql so the product is folded with EXACT numeric multiplication -- exp(sum(ln()))
-- would (a) lose exactness and (b) blow up on a factor of 0 (the -100% case, which is the whole point).
CREATE FUNCTION public.fn_effective_speed(p_world_id uuid, p_actor uuid, p_movement_type text DEFAULT 'walk')
RETURNS numeric
LANGUAGE plpgsql STABLE AS $$
DECLARE
  v_base   numeric;
  v_factor numeric := 1;
  v_pct    numeric;
BEGIN
  SELECT base_speed_mps INTO v_base
  FROM movement_type
  WHERE world_id = p_world_id AND movement_type_id = p_movement_type;

  IF v_base IS NULL THEN
    RETURN NULL;   -- movement type not minted for this world; mint ordering is upstream (§8), not here.
  END IF;

  -- Multiply in every ACTIVE-status modifier for this movement type. status_modifier's PK is
  -- (world, status, action, movement_type) so at most one factor per active status. -30% => x0.70 (§2);
  -- modifiers stack multiplicatively.
  FOR v_pct IN
    SELECT sm.modifier_percent
    FROM status_modifier sm
    WHERE sm.world_id         = p_world_id
      AND sm.movement_type_id = p_movement_type
      AND sm.action_type      = 'move'
      AND sm.status_type_id IN (
        SELECT jsonb_array_elements_text(
          COALESCE((SELECT attrs->'statuses' FROM actor_state
                    WHERE world_id = p_world_id AND entity_id = p_actor), '[]'::jsonb))
      )
  LOOP
    v_factor := v_factor * (1 + v_pct / 100.0);
  END LOOP;

  -- Floor at 0 (a -100% modifier => factor 0 => speed 0). NO upper cap.
  -- Worked example: walk 1.4 x baby(-90%) x trained(+20%) x limping(-30%)
  --               = 1.4 x 0.10 x 1.20 x 0.70 = 0.1176 m/s.
  RETURN GREATEST(v_base * v_factor, 0);
END $$;

-- fn_move_duration_actor: the REAL, status-aware duration. duration = CEIL(distance / effective_speed).
-- Distance is between the two LOCATIONS (fn_distance, Task 1); the ACTOR only supplies the speed (its
-- movement type + active statuses). Speed 0 (a -100% modifier, e.g. encumbered/tied) => return max
-- bigint so "never fits any budget" EMERGES from the arithmetic (§2), never a special case. 1 tick = 1 s.
CREATE FUNCTION public.fn_move_duration_actor(p_world_id uuid, p_actor uuid, p_from uuid, p_to uuid)
RETURNS bigint
LANGUAGE sql STABLE AS $$
  SELECT CASE
    WHEN COALESCE(fn_effective_speed(p_world_id, p_actor, 'walk'), 0) <= 0
      THEN 9223372036854775807::bigint   -- infinite duration: blocked by arithmetic (§2), not a branch
    ELSE CEIL(
      fn_distance(p_world_id, p_from, p_to) / fn_effective_speed(p_world_id, p_actor, 'walk')
    )::bigint
  END;
$$;

-- fn_move_duration: signature UNCHANGED (legacy apply_beat compat, decision 6). No actor in scope, so
-- it assumes the seeded default walk (1.4 m/s) and NO status modifiers. Was the flat-5 stub; now a thin
-- wrapper over the REAL distance so apply_beat ticks the clock by distance/speed. Real, status-aware
-- duration is fn_move_duration_actor. STABLE (not IMMUTABLE): it now reads location_state via fn_distance.
CREATE OR REPLACE FUNCTION public.fn_move_duration(p_world_id uuid, p_from uuid, p_to uuid)
RETURNS bigint
LANGUAGE sql STABLE AS $$
  SELECT CEIL(fn_distance(p_world_id, p_from, p_to) / 1.4)::bigint;
$$;

-- The SECURITY DEFINER apply_beat (owned by maintainer) calls fn_move_duration, which now reaches
-- fn_distance -- and fn_distance reads entity_registry to resolve each entity's kind. The old flat-5
-- stub touched no tables, so maintainer never needed a grant here (it already has the _state tables via
-- 20260610090005). Grant SELECT so the definer path can compute the real distance.
GRANT SELECT ON public.entity_registry TO maintainer;

-- migrate:down

REVOKE SELECT ON public.entity_registry FROM maintainer;

DROP FUNCTION IF EXISTS public.fn_move_duration_actor(uuid, uuid, uuid, uuid);
DROP FUNCTION IF EXISTS public.fn_effective_speed(uuid, uuid, text);

-- Restore the flat-5 IMMUTABLE stub (copied from 20260723100001_location_ids.sql).
CREATE OR REPLACE FUNCTION public.fn_move_duration(p_world_id uuid, p_from uuid, p_to uuid)
RETURNS bigint
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
           WHEN p_from = p_to THEN 0
           ELSE 5   -- flat default for the thin-slice fixture map (SPEC-018 spatial engine deferred)
         END::bigint;
$$;
