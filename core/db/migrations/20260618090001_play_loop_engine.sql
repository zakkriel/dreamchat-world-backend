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

  IF ev.event_type = 'move' THEN
    -- mover + destination, from the move's own location mutation.
    DECLARE
      mover uuid;
      dest  text;
      other uuid;
      pid   uuid;
    BEGIN
      SELECT entity_id INTO mover FROM event_participant
        WHERE event_id = p_event_id AND role_qualifier = 'instigator' LIMIT 1;
      SELECT (new_value #>> '{}') INTO dest FROM state_mutation
        WHERE event_id = p_event_id AND attribute_path = 'attrs.location_id' LIMIT 1;
      IF mover IS NOT NULL THEN
        -- witnessing: the mover perceives their own move ('direct').
        INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                       acquired_tick, valid_tick)
        VALUES (ev.world_id, mover, p_event_id, ev.summary, 'direct', ev.in_world_tick, ev.in_world_tick);
        n := n + 1;
        -- discovery-on-arrival (§4 trigger 2): the mover perceives each actor ALREADY at dest
        -- (exclude self). Each carries an explicit subject link → the stop-check reads about-ness.
        IF dest IS NOT NULL THEN
          FOR other IN SELECT entity_id FROM fn_actors_at(ev.world_id, dest)
                        WHERE entity_id <> mover LOOP
            INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                           acquired_tick, valid_tick)
            VALUES (ev.world_id, mover, p_event_id,
                    'On arriving, I noticed someone already here.', 'direct',
                    ev.in_world_tick, ev.in_world_tick)
            RETURNING perception_id INTO pid;
            INSERT INTO perception_subject (perception_id, entity_id, world_id)
            VALUES (pid, other, ev.world_id);
            n := n + 1;
          END LOOP;
        END IF;
      END IF;
    END;
  END IF;

  RETURN n;
END $$;
ALTER FUNCTION generate_perceptions(uuid) OWNER TO maintainer;
REVOKE EXECUTE ON FUNCTION generate_perceptions(uuid) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION generate_perceptions(uuid) TO maintainer;
-- generate_perceptions runs as maintainer (SECURITY DEFINER); grant the canon/epistemic-side
-- reads+writes it performs (mirrors apply_mutation's grants in migration 0006). These are NOT
-- projection tables, so I-7 (projections written only by the maintainer) is unaffected.
GRANT SELECT         ON event_participant  TO maintainer;
GRANT SELECT, INSERT ON perception_record  TO maintainer;  -- SELECT needed for INSERT ... RETURNING
GRANT INSERT         ON perception_subject TO maintainer;  -- discovery-on-arrival subject link (move branch)

-- apply_beat: the deterministic incremental gated loop (§8 four-stage pipeline; thin slice).
-- gate (3a, SPEC-017 co-location) → resolve (3b, identity; SPEC-013 deferred) → apply+generate (3c)
-- → stop-check (3d, discovery breaks next premise). Two first-class halts: pre-apply gate-reject and
-- post-apply stop-check (§8). Clock advances by COMMITTED-prefix durations (ADR-036). origin is a
-- param: 'fast_path' (Leg-1 fixtures) | 'freeform' (Leg-2 model-proposed, gated). D-1: this gate is
-- the ONLY canonization point; the model never writes canon. Returns a jsonb summary.
CREATE FUNCTION apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb,
                           p_start_tick bigint, p_tick_cap bigint, p_origin text DEFAULT 'fast_path')
RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  step       jsonb;
  idx        int := 0;
  cur_tick   bigint := p_start_tick;
  cur_seq    int := 0;
  start_tick bigint := p_start_tick;
  committed  jsonb := '[]'::jsonb;
  halt       text := 'completed';
  ev_id      uuid;
  dur        bigint;
  here       text;
  listener   uuid;
  next_step  jsonb;
  next_ok    boolean;
BEGIN
  FOR step IN SELECT * FROM jsonb_array_elements(p_chain) LOOP
    idx := idx + 1;
    SELECT a.attrs->>'location_id' INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;

    -- (3a) GATE — thin-slice move-validity / co-location precondition (SPEC-017).
    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
        halt := 'gate_reject'; EXIT;   -- nothing committed for this step (3a)
      END IF;
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      dur := fn_move_duration(p_world_id, COALESCE(here,'?'), step->>'to');
    ELSE
      halt := 'gate_reject'; EXIT;     -- out-of-vocabulary (closed set; ADR-009/D-1, SPEC-015)
    END IF;

    -- turn-budget backstop (§9 third pushback face): would committing exceed the cap?
    IF (cur_tick + dur) - start_tick > p_tick_cap THEN
      halt := 'turn_budget'; EXIT;
    END IF;

    -- (3b) RESOLVE = identity (SPEC-013 deferred). (3c) APPLY + GENERATE + advance clock.
    ev_id := gen_random_uuid();
    INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                             status, accepted_at, visibility_scope, origin)
    VALUES (ev_id, p_world_id,
            CASE step->>'type' WHEN 'say' THEN 'private_disclosure' ELSE 'move' END,
            COALESCE(step->>'content', step->>'type'), cur_tick, cur_seq,
            'accepted', now(),
            CASE step->>'type' WHEN 'say' THEN 'private' ELSE 'public' END, p_origin);
    IF step->>'type' = 'say' THEN
      INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
        (ev_id, p_actor_id, 'actor', 'speaker'),
        (ev_id, listener,   'actor', 'listener');
    ELSE
      INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
        VALUES (ev_id, p_actor_id, 'actor', 'instigator');
      INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                  new_value, valid_from_tick, valid_from_seq)
        VALUES (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id',
                to_jsonb(step->>'to'), cur_tick, cur_seq);  -- trigger applies the projection
    END IF;
    PERFORM generate_perceptions(ev_id);
    committed := committed || to_jsonb(ev_id);

    -- advance the clock by THIS committed event's duration (ADR-036).
    IF dur > 0 THEN cur_tick := cur_tick + dur; cur_seq := 0; ELSE cur_seq := cur_seq + 1; END IF;

    -- (3d) STOP-CHECK — runs ONLY after a committed MOVE: a move is the only thing that changes the
    -- actor's co-presence AND the only thing that generates a discovery perception. Gating on 'move'
    -- keeps the two halts DISTINCT (§8): a committed SAY must never pre-empt a later step's own gate.
    next_step := p_chain -> idx;   -- 0-based: element after the current 1-based idx
    IF step->>'type' = 'move' AND next_step IS NOT NULL AND next_step->>'type' = 'say' THEN
      SELECT a.attrs->>'location_id' INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      next_ok := EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here)
                         WHERE entity_id = (next_step->>'listener')::uuid);
      IF NOT next_ok THEN halt := 'stop_check'; EXIT; END IF;  -- prefix stands; remainder never runs
    END IF;
  END LOOP;

  RETURN jsonb_build_object('committed', committed, 'halt_reason', halt,
                            'ticks_advanced', cur_tick - start_tick);
END $$;
ALTER FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) OWNER TO maintainer;
REVOKE EXECUTE ON FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) TO maintainer;
-- apply_beat runs as maintainer (SECURITY DEFINER); grant the canon-side INSERTs it performs.
-- canon_event keeps its append-only UPDATE trigger + DELETE revoke; only INSERT is granted (I-7
-- concerns projection tables, which are untouched here).
GRANT INSERT ON canon_event       TO maintainer;
GRANT INSERT ON event_participant TO maintainer;
GRANT INSERT ON state_mutation    TO maintainer;

-- migrate:down
DROP FUNCTION IF EXISTS apply_beat(uuid, uuid, jsonb, bigint, bigint, text);
DROP FUNCTION IF EXISTS generate_perceptions(uuid);
DROP FUNCTION IF EXISTS fn_move_duration(uuid, text, text);
DROP FUNCTION IF EXISTS fn_actors_at(uuid, text);
