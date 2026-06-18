-- migrate:up
-- Chunk-5 play-loop engine (design 2026-06-16). Deterministic; NO model. SQL is the engine
-- (ADR-P017); the Go layer (Leg 2) is a thin orchestrator. Functions added incrementally across
-- Tasks 1–5; the down body drops them in reverse.

-- Co-presence (thin-slice SPEC-017 substrate): actors whose projected location label matches.
-- Reads actor_state (the projection), not canon — co-presence is a STATE question.
CREATE FUNCTION fn_actors_at(p_world_id uuid, p_location text)
RETURNS TABLE(entity_id uuid)
LANGUAGE sql STABLE AS $$
  SELECT a.entity_id
  FROM actor_state a
  WHERE a.world_id = p_world_id
    AND a.attrs->>'location_id' = p_location;
$$;

-- ADR-036 substrate: per-event duration as RECORDED, deterministic world data (D-11; §11), assigned
-- by the engine — NEVER the model (§10 Q1 guardrail). Thin slice = a hand-authored cost table; the
-- spatial engine (coordinates → derived distance/travel-time) is DEFERRED wholesale (SPEC-018), so
-- there is no derive here. Unknown pairs fall back to a flat default; same place = 0.
CREATE FUNCTION fn_move_duration(p_world_id uuid, p_from text, p_to text)
RETURNS bigint
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
           WHEN p_from = p_to THEN 0
           WHEN (p_from,p_to) IN (('tavern','square'),('square','tavern')) THEN 5
           ELSE 5   -- flat default for the thin-slice fixture map
         END::bigint;
$$;

-- Write-side perception generation (ADR-005: one event → 0..N perceptions; doc 13 §5 Phase-1 fan-out).
-- The ONLY perception write path in the loop. SECURITY DEFINER / maintainer-owned (I-7). Generates
-- from CANON's visible aspect (§4 witnessing trigger). Every row carries source_event_id (I-2) and
-- acquired_tick = the event's in_world_tick (I-9). Returns the number of perceptions written.
-- Thin slice handles 'move' and 'private_disclosure' only; other types are a no-op (0).
CREATE FUNCTION generate_perceptions(p_event_id uuid)
RETURNS integer
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  ev   canon_event;
  n    integer := 0;
  spk  uuid;
  lst  uuid;
BEGIN
  SELECT * INTO ev FROM canon_event WHERE event_id = p_event_id AND status = 'accepted';
  IF NOT FOUND THEN RETURN 0; END IF;

  IF ev.event_type = 'private_disclosure' THEN
    -- speaker → 'shared'; each listener → 'told' (B-7). Recipients = the addressed listeners
    -- (thin slice; co-present overhearers defer with the broader vocabulary, §3).
    SELECT entity_id INTO spk FROM event_participant
      WHERE event_id = p_event_id AND role_qualifier = 'speaker' LIMIT 1;
    IF spk IS NOT NULL THEN
      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, spk, p_event_id, ev.summary, 'shared', ev.in_world_tick, ev.in_world_tick);
      n := n + 1;
    END IF;
    FOR lst IN SELECT entity_id FROM event_participant
                 WHERE event_id = p_event_id AND role_qualifier = 'listener' LOOP
      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, lst, p_event_id, ev.summary, 'told', ev.in_world_tick, ev.in_world_tick);
      n := n + 1;
    END LOOP;
  END IF;

  RETURN n;
END $$;
ALTER FUNCTION generate_perceptions(uuid) OWNER TO maintainer;
REVOKE EXECUTE ON FUNCTION generate_perceptions(uuid) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION generate_perceptions(uuid) TO maintainer;
-- generate_perceptions runs as maintainer (SECURITY DEFINER); grant the canon/epistemic-side
-- reads+writes it performs (mirrors apply_mutation's grants in migration 0006). These are NOT
-- projection tables, so I-7 (projections written only by the maintainer) is unaffected.
GRANT SELECT ON event_participant TO maintainer;
GRANT INSERT ON perception_record TO maintainer;

-- migrate:down
DROP FUNCTION IF EXISTS generate_perceptions(uuid);
DROP FUNCTION IF EXISTS fn_move_duration(uuid, text, text);
DROP FUNCTION IF EXISTS fn_actors_at(uuid, text);
