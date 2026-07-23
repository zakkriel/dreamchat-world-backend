-- migrate:up
-- Task 4 (Chunk 5.5): convert location references from text labels → entity_registry UUIDs.
-- fn_actors_at and fn_move_duration gain uuid signatures; generate_perceptions and apply_beat are
-- updated in-place (CREATE OR REPLACE) so callers need no change. Data: actor_state.attrs.location_id
-- values are converted from the text label → the matching entity_registry entity_id.
-- NOTE: the label→uuid conversion is one-way; the down section restores the functions but cannot
-- reverse the data (existing rows that were never text labels are not affected).

-- 1. DATA: convert text location labels in actor_state to uuid strings via entity_registry lookup.
-- Rows whose attrs->>'location_id' already looks like a uuid (no match in canonical_name) are left
-- untouched; rows matching a canonical_name get the corresponding entity_id.
UPDATE actor_state AS a
SET attrs = jsonb_set(
              a.attrs,
              '{location_id}',
              to_jsonb(er.entity_id::text)
            )
FROM entity_registry er
WHERE er.world_id        = a.world_id
  AND er.entity_kind     = 'location'
  AND a.attrs->>'location_id' = er.canonical_name;

-- 2. fn_actors_at: p_location text → p_location_id uuid.
DROP FUNCTION IF EXISTS fn_actors_at(uuid, text);
CREATE FUNCTION fn_actors_at(p_world_id uuid, p_location_id uuid)
RETURNS TABLE(entity_id uuid)
LANGUAGE sql STABLE AS $$
  SELECT a.entity_id
  FROM actor_state a
  WHERE a.world_id = p_world_id
    AND a.attrs->>'location_id' = p_location_id::text;
$$;

-- 3. fn_move_duration: p_from text, p_to text → uuid, uuid.
-- Flat CASE WHEN same uuid → 0 ELSE 5; the hand-authored cost table is superseded by uuid identity.
DROP FUNCTION IF EXISTS fn_move_duration(uuid, text, text);
CREATE FUNCTION fn_move_duration(p_world_id uuid, p_from uuid, p_to uuid)
RETURNS bigint
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
           WHEN p_from = p_to THEN 0
           ELSE 5   -- flat default for the thin-slice fixture map (SPEC-018 spatial engine deferred)
         END::bigint;
$$;

-- 4. generate_perceptions: dest text → dest uuid; fn_actors_at call now passes uuid.
CREATE OR REPLACE FUNCTION generate_perceptions(p_event_id uuid)
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
      dest  uuid;
      other uuid;
      pid   uuid;
    BEGIN
      SELECT entity_id INTO mover FROM event_participant
        WHERE event_id = p_event_id AND role_qualifier = 'instigator' LIMIT 1;
      SELECT (new_value #>> '{}')::uuid INTO dest FROM state_mutation
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
GRANT SELECT         ON event_participant  TO maintainer;
GRANT SELECT, INSERT ON perception_record  TO maintainer;
GRANT INSERT         ON perception_subject TO maintainer;

-- 5. apply_beat: here text → here uuid; fn_actors_at/fn_move_duration args updated accordingly.
CREATE OR REPLACE FUNCTION apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb,
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
  here       uuid;
  listener   uuid;
  next_step  jsonb;
  next_ok    boolean;
BEGIN
  FOR step IN SELECT * FROM jsonb_array_elements(p_chain) LOOP
    idx := idx + 1;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;

    -- (3a) GATE — thin-slice move-validity / co-location precondition (SPEC-017).
    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
        halt := 'gate_reject'; EXIT;   -- nothing committed for this step (3a)
      END IF;
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      dur := fn_move_duration(p_world_id, here, (step->>'to')::uuid);
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
      SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
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
GRANT INSERT ON canon_event       TO maintainer;
GRANT INSERT ON event_participant TO maintainer;
GRANT INSERT ON state_mutation    TO maintainer;

-- migrate:down
-- NOTE: data conversion (label → uuid) is one-way; the down section restores the text-signature
-- functions but cannot recover text labels from uuid strings already written to actor_state.
DROP FUNCTION IF EXISTS fn_actors_at(uuid, uuid);
DROP FUNCTION IF EXISTS fn_move_duration(uuid, uuid, uuid);

-- Restore original text-signature versions (exact copies from 20260618090001_play_loop_engine.sql).
CREATE FUNCTION fn_actors_at(p_world_id uuid, p_location text)
RETURNS TABLE(entity_id uuid)
LANGUAGE sql STABLE AS $$
  SELECT a.entity_id
  FROM actor_state a
  WHERE a.world_id = p_world_id
    AND a.attrs->>'location_id' = p_location;
$$;

CREATE FUNCTION fn_move_duration(p_world_id uuid, p_from text, p_to text)
RETURNS bigint
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
           WHEN p_from = p_to THEN 0
           WHEN (p_from,p_to) IN (('tavern','square'),('square','tavern')) THEN 5
           ELSE 5   -- flat default for the thin-slice fixture map
         END::bigint;
$$;

CREATE OR REPLACE FUNCTION generate_perceptions(p_event_id uuid)
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
        INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                       acquired_tick, valid_tick)
        VALUES (ev.world_id, mover, p_event_id, ev.summary, 'direct', ev.in_world_tick, ev.in_world_tick);
        n := n + 1;
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
GRANT SELECT         ON event_participant  TO maintainer;
GRANT SELECT, INSERT ON perception_record  TO maintainer;
GRANT INSERT         ON perception_subject TO maintainer;

CREATE OR REPLACE FUNCTION apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb,
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

    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
        halt := 'gate_reject'; EXIT;
      END IF;
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      dur := fn_move_duration(p_world_id, COALESCE(here,'?'), step->>'to');
    ELSE
      halt := 'gate_reject'; EXIT;
    END IF;

    IF (cur_tick + dur) - start_tick > p_tick_cap THEN
      halt := 'turn_budget'; EXIT;
    END IF;

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
                to_jsonb(step->>'to'), cur_tick, cur_seq);
    END IF;
    PERFORM generate_perceptions(ev_id);
    committed := committed || to_jsonb(ev_id);

    IF dur > 0 THEN cur_tick := cur_tick + dur; cur_seq := 0; ELSE cur_seq := cur_seq + 1; END IF;

    next_step := p_chain -> idx;
    IF step->>'type' = 'move' AND next_step IS NOT NULL AND next_step->>'type' = 'say' THEN
      SELECT a.attrs->>'location_id' INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      next_ok := EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here)
                         WHERE entity_id = (next_step->>'listener')::uuid);
      IF NOT next_ok THEN halt := 'stop_check'; EXIT; END IF;
    END IF;
  END LOOP;

  RETURN jsonb_build_object('committed', committed, 'halt_reason', halt,
                            'ticks_advanced', cur_tick - start_tick);
END $$;
ALTER FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) OWNER TO maintainer;
REVOKE EXECUTE ON FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION apply_beat(uuid, uuid, jsonb, bigint, bigint, text) TO maintainer;
GRANT INSERT ON canon_event       TO maintainer;
GRANT INSERT ON event_participant TO maintainer;
GRANT INSERT ON state_mutation    TO maintainer;
