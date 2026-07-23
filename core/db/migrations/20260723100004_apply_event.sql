-- migrate:up

-- ─── 1. generate_perceptions — extend branches to accept canonical labels ─────
-- Legacy rows ('move', 'private_disclosure') still replay identically.
-- New canonical labels ('ActorMoved', 'Communicated') route to the same branch.

CREATE OR REPLACE FUNCTION generate_perceptions(p_event_id uuid) RETURNS integer
  LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  ev   canon_event;
  n    integer := 0;
  spk  uuid;
  lst  uuid;
BEGIN
  SELECT * INTO ev FROM canon_event WHERE event_id = p_event_id AND status = 'accepted';
  IF NOT FOUND THEN RETURN 0; END IF;

  IF ev.event_type IN ('private_disclosure', 'Communicated') THEN
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

  IF ev.event_type IN ('move', 'ActorMoved') THEN
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

-- ─── 2. apply_event — single-step, six-type canon commit ─────────────────────
-- STRUCTURAL FLOOR (blocker-only — contracts arrive pre-adjudicated):
--   every type: p_actor_id exists in entity_registry (kind='actor') for this world.
--   ActorMoved:  to_location_id exists in entity_registry.
--   Communicated: listener co-located with actor via fn_actors_at.
--   ObjectRelocated: object_id + dest_id exist in entity_registry.
--   OwnershipAccessChanged | EntityDestroyed | AttributeChanged: target_id exists.
--   EntityCreated: no target check (adjudicated intent — actor existence covers it).
-- Floor fails → {"event_id": null, "halt_reason": "gate_reject"}, NOTHING written.
--
-- p_legacy_types boolean DEFAULT false:
--   false (default) → writes canonical type strings ('ActorMoved', 'Communicated').
--   true            → writes legacy labels ('move', 'private_disclosure') so apply_beat
--                     delegation is behavior-identical for existing tests.

CREATE OR REPLACE FUNCTION apply_event(
  p_world_id     uuid,
  p_actor_id     uuid,
  p_attempt      jsonb,
  p_tick         bigint,
  p_seq          int,
  p_origin       text,
  p_legacy_types boolean DEFAULT false
) RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  ev_type     text;
  ev_id       uuid;
  listener    uuid;
  to_loc      uuid;
  object_eid  uuid;
  dest_eid    uuid;
  target_eid  uuid;
  here        uuid;
  vis_scope   text;
  final_type  text;
BEGIN
  ev_type := p_attempt->>'type';

  -- ── STRUCTURAL FLOOR ───────────────────────────────────────────────────────
  -- Every type: actor must exist in entity_registry with kind='actor'.
  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = p_actor_id
      AND world_id  = p_world_id
      AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- Type-specific floor checks.
  IF ev_type = 'ActorMoved' THEN
    to_loc := (p_attempt->>'to_location_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry
      WHERE entity_id = to_loc AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_attempt->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
    IF NOT EXISTS (
      SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'ObjectRelocated' THEN
    object_eid := (p_attempt->>'object_id')::uuid;
    dest_eid   := (p_attempt->>'dest_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = object_eid AND world_id = p_world_id)
    OR NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = dest_eid  AND world_id = p_world_id)
    THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_attempt->>'target_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- No target check required (adjudicated intent; actor floor above covers the writer).
    NULL;

  ELSE
    -- Unknown type: reject.
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- ── COMMIT ─────────────────────────────────────────────────────────────────
  ev_id := gen_random_uuid();

  -- Determine the final event_type string to write.
  -- p_legacy_types=true: write old labels so apply_beat's delegation is behavior-identical.
  IF p_legacy_types THEN
    final_type := CASE ev_type
      WHEN 'Communicated' THEN 'private_disclosure'
      WHEN 'ActorMoved'   THEN 'move'
      ELSE ev_type
    END;
  ELSE
    final_type := ev_type;
  END IF;

  -- visibility_scope: private for Communicated (or legacy private_disclosure), public otherwise.
  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, final_type, p_attempt->>'stated',
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

  -- event_participant: Communicated → speaker + listener; all others → instigator.
  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, p_actor_id, 'actor', 'speaker'),
      (ev_id, listener,   'actor', 'listener');
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, p_actor_id, 'actor', 'instigator');
  END IF;

  -- state_mutation: ActorMoved only — trigger projects it into actor_state.
  IF ev_type = 'ActorMoved' THEN
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id',
            to_jsonb(to_loc::text), p_tick, p_seq);
  END IF;

  PERFORM generate_perceptions(ev_id);

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

-- ─── 3. apply_beat — refactored to delegate per-step commit to apply_event ───
-- apply_beat's own return contract ({committed, halt_reason, ticks_advanced}) is unchanged.
-- Legacy event_type labels ('move'/'private_disclosure') are preserved via p_legacy_types=true.

CREATE OR REPLACE FUNCTION apply_beat(
  p_world_id   uuid,
  p_actor_id   uuid,
  p_chain      jsonb,
  p_start_tick bigint,
  p_tick_cap   bigint,
  p_origin     text DEFAULT 'fast_path'
) RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  step       jsonb;
  idx        int := 0;
  cur_tick   bigint := p_start_tick;
  cur_seq    int := 0;
  start_tick bigint := p_start_tick;
  committed  jsonb := '[]'::jsonb;
  halt       text := 'completed';
  dur        bigint;
  here       uuid;
  listener   uuid;
  next_step  jsonb;
  next_ok    boolean;
  attempt    jsonb;
  result     jsonb;
BEGIN
  FOR step IN SELECT * FROM jsonb_array_elements(p_chain) LOOP
    idx := idx + 1;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;

    -- Build the attempt shape and compute duration for budget check.
    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      attempt := jsonb_build_object(
        'type',        'Communicated',
        'stated',      COALESCE(step->>'content', 'say'),
        'listener_id', listener,
        'content',     step->>'content'
      );
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      attempt := jsonb_build_object(
        'type',           'ActorMoved',
        'stated',         'move',
        'to_location_id', step->>'to'
      );
      dur := fn_move_duration(p_world_id, here, (step->>'to')::uuid);
    ELSE
      halt := 'gate_reject'; EXIT;     -- out-of-vocabulary (closed set; ADR-009/D-1, SPEC-015)
    END IF;

    -- turn-budget backstop (§9 third pushback face): would committing exceed the cap?
    IF (cur_tick + dur) - start_tick > p_tick_cap THEN
      halt := 'turn_budget'; EXIT;
    END IF;

    -- Delegate to apply_event with p_legacy_types=true to preserve legacy event_type labels.
    result := apply_event(p_world_id, p_actor_id, attempt, cur_tick, cur_seq, p_origin, true);

    IF result->>'halt_reason' = 'gate_reject' THEN
      halt := 'gate_reject'; EXIT;
    END IF;

    committed := committed || to_jsonb(result->>'event_id');

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

-- migrate:down

DROP FUNCTION IF EXISTS apply_event(uuid, uuid, jsonb, bigint, int, text, boolean);

-- Restore apply_beat to the pre-delegation body (verbatim from schema.sql as of migration ~100003).
CREATE OR REPLACE FUNCTION apply_beat(
  p_world_id   uuid,
  p_actor_id   uuid,
  p_chain      jsonb,
  p_start_tick bigint,
  p_tick_cap   bigint,
  p_origin     text DEFAULT 'fast_path'
) RETURNS jsonb
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

-- Restore generate_perceptions to pre-canonical-labels body.
CREATE OR REPLACE FUNCTION generate_perceptions(p_event_id uuid) RETURNS integer
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
