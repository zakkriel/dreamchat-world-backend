SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: apply_attribute_writes(uuid, jsonb, uuid, bigint, integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.apply_attribute_writes(p_world_id uuid, p_writes jsonb, p_provenance_event uuid, p_tick bigint, p_seq integer) RETURNS integer
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  w        jsonb;
  target   uuid;
  ekind    text;
  n        int := 0;
BEGIN
  FOR w IN SELECT * FROM jsonb_array_elements(p_writes) LOOP
    target := (w->>'target_id')::uuid;

    -- Look up entity_kind from the registry (required by state_mutation schema).
    SELECT entity_kind INTO ekind
      FROM entity_registry
      WHERE entity_id = target AND world_id = p_world_id
      LIMIT 1;

    -- Skip writes for unknown entities (defensive; caller validates).
    IF ekind IS NULL THEN
      CONTINUE;
    END IF;

    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, p_provenance_event, target, ekind,
            'attrs.' || (w->>'attribute'),
            w->'value',
            p_tick, p_seq);

    n := n + 1;
  END LOOP;

  RETURN n;
END $$;


--
-- Name: apply_beat(uuid, uuid, jsonb, bigint, bigint, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb, p_start_tick bigint, p_tick_cap bigint, p_origin text DEFAULT 'fast_path'::text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
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


--
-- Name: apply_event(uuid, uuid, jsonb, bigint, integer, text, boolean); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
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


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: state_mutation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.state_mutation (
    mutation_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    event_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    entity_kind text NOT NULL,
    attribute_path text NOT NULL,
    old_value jsonb,
    new_value jsonb NOT NULL,
    valid_from_tick bigint NOT NULL,
    valid_from_seq integer DEFAULT 0 NOT NULL,
    status text DEFAULT 'applied'::text NOT NULL,
    CONSTRAINT state_mutation_status_check CHECK ((status = ANY (ARRAY['applied'::text, 'reversed'::text, 'dirty'::text])))
);


--
-- Name: apply_mutation(public.state_mutation); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.apply_mutation(m public.state_mutation) RETURNS void
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  -- strip leading 'attrs.' (6 chars) -> single-key JSON path under attrs (0A convention, Rider B)
  jpath text[] := string_to_array(substring(m.attribute_path from 7), '.');
BEGIN
  IF m.entity_kind = 'actor' THEN
    INSERT INTO actor_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(actor_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'location' THEN
    INSERT INTO location_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(location_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'artifact' THEN
    INSERT INTO artifact_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(artifact_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'relationship' THEN
    -- SPEC-001: doc 03 does not define mutation->(a_id,b_id) addressing. NO-OP stub in 0A.
    NULL;
  END IF;
END $$;


--
-- Name: apply_ruled_event(uuid, jsonb, bigint, integer, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.apply_ruled_event(p_world_id uuid, p_ruled jsonb, p_tick bigint, p_seq integer, p_origin text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type    text;
  actor_id   uuid;
  ev_id      uuid;
  listener   uuid;
  to_loc     uuid;
  object_eid uuid;
  dest_eid   uuid;
  target_eid uuid;
  here       uuid;
  vis_scope  text;
  truth_text text;
  appear_txt text;
  visible    boolean;
  -- perception fan-out
  receiver   uuid;
  recv_text  text;
  var_text   text;
  pid        uuid;
  -- participant ids for about-ness (perception_subject)
  participant_ids uuid[];
BEGIN
  ev_type  := p_ruled->>'type';
  actor_id := (p_ruled->>'actor_id')::uuid;
  truth_text := p_ruled->>'truth';
  appear_txt := NULLIF(TRIM(COALESCE(p_ruled->>'appearance', '')), '');
  visible    := CASE
    WHEN p_ruled ? 'visible' AND (p_ruled->>'visible') = 'false' THEN false
    ELSE true
  END;

  -- ── STRUCTURAL FLOOR (twin of apply_event floor — keep in sync) ─────────────
  -- Every type: actor must exist in entity_registry with kind='actor'.
  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = actor_id
      AND world_id  = p_world_id
      AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- Type-specific floor checks.
  IF ev_type = 'ActorMoved' THEN
    to_loc := (p_ruled->>'to_location_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry
      WHERE entity_id = to_loc AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_ruled->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = actor_id;
    IF NOT EXISTS (
      SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'ObjectRelocated' THEN
    object_eid := (p_ruled->>'object_id')::uuid;
    dest_eid   := (p_ruled->>'dest_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = object_eid AND world_id = p_world_id)
    OR NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = dest_eid  AND world_id = p_world_id)
    THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_ruled->>'target_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- No target check required (actor-only floor above covers it).
    NULL;

  ELSE
    -- Unknown type: reject.
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- ── COMMIT ──────────────────────────────────────────────────────────────────
  ev_id := gen_random_uuid();

  -- visibility_scope: private for Communicated, public otherwise.
  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  -- Canon summary = truth. CANON NEVER LIES.
  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, ev_type, truth_text,
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

  -- event_participant: Communicated → speaker (actor) + listener; all others → instigator.
  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, actor_id, 'actor', 'speaker'),
      (ev_id, listener, 'actor', 'listener');
    participant_ids := ARRAY[actor_id, listener];
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, actor_id, 'actor', 'instigator');
    participant_ids := ARRAY[actor_id];
  END IF;

  -- state_mutation: ActorMoved only — trigger projects it into actor_state.
  IF ev_type = 'ActorMoved' THEN
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id',
            to_jsonb(to_loc::text), p_tick, p_seq);
  END IF;

  -- ── PERCEPTION FAN-OUT ───────────────────────────────────────────────────────
  -- visible=false (explicit) → zero perceptions; no further work.
  IF NOT visible THEN
    RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
  END IF;

  -- Receivers = actors co-located with the ruled actor (fn_actors_at on actor's
  -- current location, NULL-safe) UNION the actor itself (actor already included
  -- when they are present at their own location; UNION deduplicates).
  --
  -- After ActorMoved, the actor's location is already updated in actor_state by
  -- the trigger above, so fn_actors_at returns the destination set. For non-move
  -- types the actor is still at their original location — both are correct behavior.
  SELECT (a.attrs->>'location_id')::uuid INTO here
    FROM actor_state a
    WHERE a.world_id = p_world_id AND a.entity_id = actor_id;

  FOR receiver IN
    SELECT entity_id FROM fn_actors_at(p_world_id, here)
    UNION
    SELECT actor_id
  LOOP
    -- Determine content for this receiver:
    -- 1. receiver_variants match → variant text
    -- 2. appearance (if non-empty) → appearance
    -- 3. truth
    var_text := NULL;
    IF p_ruled ? 'receiver_variants' THEN
      SELECT rv->>'text' INTO var_text
        FROM jsonb_array_elements(p_ruled->'receiver_variants') AS rv
        WHERE (rv->>'receiver_id')::uuid = receiver
        LIMIT 1;
    END IF;

    recv_text := COALESCE(var_text, appear_txt, truth_text);

    INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                   acquired_tick, valid_tick)
    VALUES (p_world_id, receiver, ev_id, recv_text, 'direct', p_tick, p_tick)
    RETURNING perception_id INTO pid;

    -- About-ness: perception_subject rows for ALL participant ids (engine-written,
    -- NOT relying on seed backfill).
    DECLARE
      part_id uuid;
    BEGIN
      FOREACH part_id IN ARRAY participant_ids LOOP
        INSERT INTO perception_subject (perception_id, entity_id, world_id)
          VALUES (pid, part_id, p_world_id);
      END LOOP;
    END;
  END LOOP;

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;


--
-- Name: canon_event_append_only(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.canon_event_append_only() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  IF ROW(NEW.event_id, NEW.world_id, NEW.scene_id, NEW.beat_id, NEW.event_type, NEW.summary,
         NEW.payload, NEW.schema_version, NEW.in_world_tick, NEW.in_world_label, NEW.beat_seq,
         NEW.temporal_uncertainty, NEW.recorded_at, NEW.visibility_scope, NEW.confidence,
         NEW.origin, NEW.template_id, NEW.source_refs)
     IS DISTINCT FROM
     ROW(OLD.event_id, OLD.world_id, OLD.scene_id, OLD.beat_id, OLD.event_type, OLD.summary,
         OLD.payload, OLD.schema_version, OLD.in_world_tick, OLD.in_world_label, OLD.beat_seq,
         OLD.temporal_uncertainty, OLD.recorded_at, OLD.visibility_scope, OLD.confidence,
         OLD.origin, OLD.template_id, OLD.source_refs)
  THEN
    RAISE EXCEPTION 'canon_event is append-only: only {status, accepted_at, superseded_by} may change (event %)', OLD.event_id;
  END IF;

  IF OLD.status IS DISTINCT FROM NEW.status
     AND NOT ( (OLD.status='proposed' AND NEW.status IN ('accepted','rejected'))
            OR (OLD.status='accepted' AND NEW.status IN ('retconned','superseded')) ) THEN
    RAISE EXCEPTION 'illegal canon_event status transition % -> %', OLD.status, NEW.status;
  END IF;
  RETURN NEW;
END $$;


--
-- Name: causal_bundle_append_only(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.causal_bundle_append_only() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  IF ROW(NEW.bundle_id, NEW.world_id, NEW.effect_ref, NEW.effect_kind,
         NEW.semantics, NEW.template_id, NEW.created_at)
     IS DISTINCT FROM
     ROW(OLD.bundle_id, OLD.world_id, OLD.effect_ref, OLD.effect_kind,
         OLD.semantics, OLD.template_id, OLD.created_at)
  THEN
    RAISE EXCEPTION 'causal_bundle is append-only: only {status} may change (bundle %)', OLD.bundle_id;
  END IF;
  RETURN NEW;
END $$;


--
-- Name: causal_bundle_assert_acyclic(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.causal_bundle_assert_acyclic() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
  v_effect_ref  uuid;
  v_effect_kind text;
  v_cycle       boolean;
  v_maxdepth    int;
BEGIN
  SELECT effect_ref, effect_kind INTO v_effect_ref, v_effect_kind
  FROM causal_bundle WHERE bundle_id = NEW.bundle_id;

  WITH RECURSIVE anc(ref, kind, depth) AS (
    SELECT NEW.input_ref, NEW.input_kind, 0
    UNION ALL
    SELECT cbi.input_ref, cbi.input_kind, anc.depth + 1
    FROM anc
    JOIN causal_bundle cb
      ON cb.effect_ref = anc.ref AND cb.effect_kind = anc.kind
    JOIN causal_bundle_input cbi
      ON cbi.bundle_id = cb.bundle_id
    WHERE anc.depth < 64
  )
  SELECT bool_or(ref = v_effect_ref AND kind = v_effect_kind), max(depth)
  INTO v_cycle, v_maxdepth
  FROM anc;

  IF v_cycle THEN
    RAISE EXCEPTION
      'causal cycle rejected (I-4): effect %/% is already a causal ancestor of input %/% (bundle %)',
      v_effect_kind, v_effect_ref, NEW.input_kind, NEW.input_ref, NEW.bundle_id;
  END IF;

  IF v_maxdepth >= 64 THEN
    RAISE EXCEPTION
      'causal acyclicity depth cap (64) exceeded walking ancestors of input %/% — investigate (I-4)',
      NEW.input_kind, NEW.input_ref;
  END IF;

  RETURN NEW;
END $$;


--
-- Name: causal_bundle_input_immutable(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.causal_bundle_input_immutable() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  RAISE EXCEPTION 'causal_bundle_input is immutable: UPDATE forbidden (bundle %, input %)',
    OLD.bundle_id, OLD.input_ref;
END $$;


--
-- Name: fn_actor_page(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_actor_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'actor_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id',                         p_actor_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      'perceived_role',             NULL,
      'current_synthesis',          NULL,
      'last_known_status',          NULL,
      'known_artifacts',            '[]'::json,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;


--
-- Name: fn_actors_at(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_actors_at(p_world_id uuid, p_location_id uuid) RETURNS TABLE(entity_id uuid)
    LANGUAGE sql STABLE
    AS $$
  SELECT a.entity_id
  FROM actor_state a
  WHERE a.world_id = p_world_id
    AND a.attrs->>'location_id' = p_location_id::text;
$$;


--
-- Name: fn_artifact_page(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_artifact_page(p_world_id uuid, p_viewer_id uuid, p_artifact_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_artifact_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'artifact_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'artifact', json_build_object(
      'id',                         p_artifact_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_artifact_id),
      'perceived_type',             NULL,
      'current_synthesis',          NULL,
      'last_known_location',        NULL,
      'current_holder_owner_access',NULL,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_artifact_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;


--
-- Name: fn_batch_display_name(uuid, uuid[], uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_batch_display_name(p_world_id uuid, p_minds uuid[], p_entity_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  WITH names AS (
    SELECT fn_perceived_name(p_world_id, m, p_entity_id) AS nm
    FROM unnest(p_minds) AS m
  ), agreed AS (
    SELECT max(nm) AS nm
    FROM names
    HAVING cardinality(p_minds) > 0
       AND count(*) = count(nm)          -- every mind resolved a name (no NULLs among the minds)
       AND count(DISTINCT nm) = 1        -- and it is the SAME name for all of them
  )
  SELECT COALESCE(
    (SELECT nm FROM agreed),
    (SELECT ast.attrs->>'descriptor' FROM actor_state ast
      WHERE ast.world_id = p_world_id AND ast.entity_id = p_entity_id),
    (SELECT er.canonical_name FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_entity_id)
  );
$$;


--
-- Name: fn_collected_knowledge(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_collected_knowledge(p_world_id uuid, p_viewer_id uuid, p_target_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  WITH about AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_target_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',  json_build_object('stale', false, 'last_confirmed_label', a.in_world_label),
             'source', json_build_object('epistemic_type', a.epistemic_type,
                                         'source_event_label', a.in_world_label)
           ) AS item,
           a.valid_tick AS sort_tick
    FROM about a
  )
  SELECT CASE WHEN count(*) = 0 THEN '[]'::json
              ELSE json_build_array(json_build_object(
                     'group_key',   p_target_id::text,
                     'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_target_id),
                     'items',       coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                   ))
         END
  FROM items i;
$$;


--
-- Name: fn_compendium_index(uuid, uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_index(p_world_id uuid, p_viewer_id uuid, p_kind text) RETURNS TABLE(entity_id uuid, perceived_name text)
    LANGUAGE sql STABLE
    AS $$
  SELECT DISTINCT er.entity_id,
         fn_perceived_name(p_world_id, p_viewer_id, er.entity_id)
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp       -- FILTER 1, unchanged
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id
  JOIN entity_registry er ON er.entity_id = ps.entity_id AND er.world_id = p_world_id
  WHERE er.entity_kind = p_kind;
$$;


--
-- Name: fn_compendium_index_json(uuid, uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_index_json(p_world_id uuid, p_viewer_id uuid, p_kind text) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
    'schema_version', 'compendium_index/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'kind',      p_kind,
    'entries', coalesce(
      (SELECT json_agg(json_build_object('id', entity_id, 'perceived_name', perceived_name)
                       ORDER BY entity_id)
       FROM fn_compendium_index(p_world_id, p_viewer_id, p_kind)), '[]'::json)
  );
$$;


--
-- Name: fn_display_name(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_display_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  -- The label an entity wears in ONE viewer's mind (§3). Known name (the viewer's own name-knowledge,
  -- via fn_perceived_name — a world_genesis-sourced perception subject-linked to the entity that the
  -- viewer holds) if any; else the entity's DESCRIPTOR (Tier-2 'descriptor' attr — what a stranger
  -- sees, e.g. "the muscle by the bar"); else the canonical registry name (engine fallback; a seed lag,
  -- never shown once descriptors are seeded). ids stay real at every call site — the model binds the
  -- id, this is only the label the viewer's knowledge puts on it.
  SELECT COALESCE(
    fn_perceived_name(p_world_id, p_viewer_id, p_entity_id),
    (SELECT ast.attrs->>'descriptor' FROM actor_state ast
      WHERE ast.world_id = p_world_id AND ast.entity_id = p_entity_id),
    (SELECT er.canonical_name FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_entity_id)
  );
$$;


--
-- Name: fn_distance(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_distance(p_world_id uuid, p_a uuid, p_b uuid) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
  WITH RECURSIVE
  ncp AS (
    SELECT fn_nearest_common_parent(p_world_id, p_a, p_b) AS loc
  ),
  ent AS (
    -- each entity's kind, its scene, and its OWN coordinates (from whichever state table it lives in)
    SELECT x.which, x.eid,
      CASE er.entity_kind
        WHEN 'location' THEN x.eid
        WHEN 'actor'    THEN (ast.attrs->>'location_id')::uuid
        WHEN 'artifact' THEN COALESCE((art.attrs->>'location_id')::uuid, er.current_scene_id)
      END AS scene,
      COALESCE(ast.attrs->'coordinates', art.attrs->'coordinates', ls.attrs->'coordinates') AS own_coord
    FROM (VALUES ('a', p_a), ('b', p_b)) AS x(which, eid)
    JOIN entity_registry er ON er.world_id = p_world_id AND er.entity_id = x.eid
    LEFT JOIN actor_state    ast ON ast.world_id = p_world_id AND ast.entity_id = x.eid
    LEFT JOIN artifact_state art ON art.world_id = p_world_id AND art.entity_id = x.eid
    LEFT JOIN location_state  ls ON  ls.world_id = p_world_id AND  ls.entity_id = x.eid
  ),
  climb AS (
    -- ancestor-or-self chain of each entity's scene, stopping once we reach the common parent
    SELECT e.which, e.scene AS loc
    FROM ent e
    UNION ALL
    SELECT c.which, (ls.attrs->>'parent_location_id')::uuid
    FROM climb c
    JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
      AND c.loc IS DISTINCT FROM (SELECT loc FROM ncp)   -- never climb above the common parent
  ),
  frame_coord AS (
    SELECT e.which,
      CASE
        WHEN e.scene = (SELECT loc FROM ncp)
          THEN e.own_coord   -- same frame: the entity's own coordinate is already parent-local
        ELSE (
          -- cross-level: the ancestor location that is the direct child of the common parent
          SELECT ls.attrs->'coordinates'
          FROM climb c
          JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = c.loc
          WHERE c.which = e.which
            AND (ls.attrs->>'parent_location_id')::uuid = (SELECT loc FROM ncp)
          LIMIT 1
        )
      END AS coord
    FROM ent e
  )
  SELECT sqrt(
      power(COALESCE((ca.coord->>'x')::numeric, 0) - COALESCE((cb.coord->>'x')::numeric, 0), 2)
    + power(COALESCE((ca.coord->>'y')::numeric, 0) - COALESCE((cb.coord->>'y')::numeric, 0), 2)
  )
  FROM (SELECT coord FROM frame_coord WHERE which = 'a') ca,
       (SELECT coord FROM frame_coord WHERE which = 'b') cb;
$$;


--
-- Name: pending_event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pending_event (
    pending_id uuid NOT NULL,
    world_id uuid NOT NULL,
    fire_at_tick bigint NOT NULL,
    magnitude text NOT NULL,
    payload jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    CONSTRAINT pending_event_magnitude_check CHECK ((magnitude = ANY (ARRAY['small'::text, 'medium'::text, 'large'::text]))),
    CONSTRAINT pending_event_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'fired'::text, 'cancelled'::text])))
);


--
-- Name: fn_due_pending(uuid, bigint); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_due_pending(p_world_id uuid, p_tick bigint) RETURNS SETOF public.pending_event
    LANGUAGE sql STABLE
    AS $$
  SELECT * FROM pending_event
  WHERE world_id = p_world_id AND status = 'pending' AND fire_at_tick <= p_tick
  ORDER BY fire_at_tick;
$$;


--
-- Name: fn_effective_speed(uuid, uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_effective_speed(p_world_id uuid, p_actor uuid, p_movement_type text DEFAULT 'walk'::text) RETURNS numeric
    LANGUAGE plpgsql STABLE
    AS $$
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


--
-- Name: fn_entity_visible(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_entity_visible(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS boolean
    LANGUAGE sql STABLE
    AS $$
  SELECT EXISTS (
    SELECT 1
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp     -- FILTER 1, unchanged
    JOIN perception_subject ps ON ps.perception_id = vp.perception_id
    WHERE ps.entity_id = p_entity_id);
$$;


--
-- Name: fn_isolated_npcs(uuid, uuid[], uuid[], uuid[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_isolated_npcs(p_world_id uuid, p_action_ids uuid[], p_present uuid[], p_npcs uuid[]) RETURNS TABLE(actor_id uuid)
    LANGUAGE sql STABLE
    AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick
    FROM perception_record pr
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
  )
  -- an NPC is isolated iff she holds >=1 PRIVATE record whose about-ness intersects the action ids
  SELECT DISTINCT pr.holder_id AS actor_id
  FROM perception_record pr
  LEFT JOIN shared s
    ON s.source_event_id = pr.source_event_id
   AND s.modal_content   = pr.content        -- matches only the public (modal) face
  WHERE pr.world_id = p_world_id
    AND pr.holder_id = ANY(p_npcs)
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    AND s.source_event_id IS NULL             -- private: no matching public face
    -- name-knowledge is an identity substrate, not a secret: a world_genesis-sourced record must
    -- never pull its holder isolated (§3; mirrors the fn_actor_page tripwire).
    AND NOT EXISTS (
      SELECT 1 FROM canon_event ce
      WHERE ce.event_id = pr.source_event_id AND ce.event_type = 'world_genesis'
    )
    AND EXISTS (
      SELECT 1 FROM perception_subject ps
      WHERE ps.perception_id = pr.perception_id
        AND ps.entity_id = ANY(p_action_ids)  -- one-hop id intersection (ADR-035)
    )
$$;


--
-- Name: fn_location_depth(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_location_depth(p_world_id uuid, p_location uuid) RETURNS integer
    LANGUAGE sql STABLE
    AS $$
  WITH RECURSIVE chain AS (
    SELECT p_location AS loc, 0 AS depth
    UNION ALL
    SELECT (ls.attrs->>'parent_location_id')::uuid, c.depth + 1
    FROM chain c
    JOIN location_state ls
      ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL   -- stop at the root (no parent edge)
  )
  SELECT max(depth) FROM chain;   -- deepest step reached climbing to root = the location's depth
$$;


--
-- Name: fn_location_page(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_location_page(p_world_id uuid, p_viewer_id uuid, p_location_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_location_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'location_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'location', json_build_object(
      'id',                         p_location_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_location_id),
      'part_of',                    NULL,
      'current_synthesis',          NULL,
      'last_known_status',          NULL,
      'known_areas_inside',         '[]'::json,
      'key_actors',                 '[]'::json,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_location_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;


--
-- Name: fn_move_duration(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_move_duration(p_world_id uuid, p_from uuid, p_to uuid) RETURNS bigint
    LANGUAGE sql STABLE
    AS $$
  SELECT CEIL(fn_distance(p_world_id, p_from, p_to) / 1.4)::bigint;
$$;


--
-- Name: fn_move_duration_actor(uuid, uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_move_duration_actor(p_world_id uuid, p_actor uuid, p_from uuid, p_to uuid) RETURNS bigint
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE
    WHEN COALESCE(fn_effective_speed(p_world_id, p_actor, 'walk'), 0) <= 0
      THEN 9223372036854775807::bigint   -- infinite duration: blocked by arithmetic (§2), not a branch
    ELSE CEIL(
      fn_distance(p_world_id, p_from, p_to) / fn_effective_speed(p_world_id, p_actor, 'walk')
    )::bigint
  END;
$$;


--
-- Name: fn_nearest_common_parent(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_nearest_common_parent(p_world_id uuid, p_a uuid, p_b uuid) RETURNS uuid
    LANGUAGE sql STABLE
    AS $$
  WITH RECURSIVE
  scenes AS (
    SELECT x.which,
      CASE er.entity_kind
        WHEN 'location' THEN x.eid
        WHEN 'actor'    THEN (ast.attrs->>'location_id')::uuid
        WHEN 'artifact' THEN COALESCE((art.attrs->>'location_id')::uuid, er.current_scene_id)
      END AS scene
    FROM (VALUES ('a', p_a), ('b', p_b)) AS x(which, eid)
    JOIN entity_registry er ON er.world_id = p_world_id AND er.entity_id = x.eid
    LEFT JOIN actor_state    ast ON ast.world_id = p_world_id AND ast.entity_id = x.eid
    LEFT JOIN artifact_state art ON art.world_id = p_world_id AND art.entity_id = x.eid
  ),
  anc AS (
    SELECT s.which, s.scene AS loc, 0 AS up
    FROM scenes s
    UNION ALL
    SELECT a.which, (ls.attrs->>'parent_location_id')::uuid, a.up + 1
    FROM anc a
    JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = a.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
  )
  SELECT aa.loc
  FROM anc aa
  JOIN anc bb ON bb.which = 'b' AND bb.loc = aa.loc   -- common ancestor of both scenes
  WHERE aa.which = 'a'
  ORDER BY aa.up ASC   -- fewest steps up from A = closest to A = the DEEPEST common ancestor
  LIMIT 1;
$$;


--
-- Name: fn_perceived_name(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_perceived_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  SELECT vp.content
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id AND ps.entity_id = p_entity_id
  JOIN canon_event ce ON ce.event_id = vp.source_event_id
  WHERE ce.event_type = 'world_genesis'
  ORDER BY vp.acquired_tick
  LIMIT 1;
$$;


--
-- Name: fn_private_records(uuid, uuid, uuid[], uuid[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_private_records(p_world_id uuid, p_npc uuid, p_action_ids uuid[], p_present uuid[]) RETURNS TABLE(content text, acquired_tick bigint)
    LANGUAGE sql STABLE
    AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick
    FROM perception_record pr
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
  ), freshest AS (
    -- cap is a v1 dial; §10's retrieval assembly refines it in Station I. Keep the FRESHEST 20, not
    -- the oldest: a private cap must drop old records first, never the ones most likely to matter
    -- now. Full tie-break (content, perception_id) keeps the 20-row cut deterministic.
    SELECT pr.content, pr.acquired_tick, pr.perception_id
    FROM perception_record pr
    LEFT JOIN shared s
      ON s.source_event_id = pr.source_event_id
     AND s.modal_content   = pr.content
    WHERE pr.world_id = p_world_id
      AND pr.holder_id = p_npc
      -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
      AND s.source_event_id IS NULL             -- private records only
      -- name-knowledge is an identity substrate, not a secret: a world_genesis-sourced record must
      -- never ride the private block (§3; mirrors the fn_actor_page tripwire).
      AND NOT EXISTS (
        SELECT 1 FROM canon_event ce
        WHERE ce.event_id = pr.source_event_id AND ce.event_type = 'world_genesis'
      )
      AND EXISTS (
        SELECT 1 FROM perception_subject ps
        WHERE ps.perception_id = pr.perception_id
          AND ps.entity_id = ANY(p_action_ids)  -- subjects intersect the action's bound ids
      )
    ORDER BY pr.acquired_tick DESC, pr.content DESC, pr.perception_id DESC
    LIMIT 20
  )
  SELECT f.content, f.acquired_tick
  FROM freshest f
  ORDER BY f.acquired_tick ASC, f.content ASC, f.perception_id ASC   -- present oldest-of-the-freshest first
$$;


--
-- Name: fn_public_moment(uuid, uuid[], integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_public_moment(p_world_id uuid, p_present uuid[], p_k integer) RETURNS TABLE(source_event_id uuid, acquired_tick bigint, content text)
    LANGUAGE sql STABLE
    AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick
    FROM perception_record pr
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content,
           min(h.tick) AS tick
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
  ), recent AS (
    -- the LAST p_k shared source events by tick (most recent), deterministic tie-break by id
    SELECT s.source_event_id, s.tick, s.modal_content
    FROM shared s
    ORDER BY s.tick DESC, s.source_event_id DESC
    LIMIT p_k
  )
  SELECT r.source_event_id, r.tick AS acquired_tick, r.modal_content AS content
  FROM recent r
  ORDER BY r.tick ASC, r.source_event_id ASC   -- append-only, cache-native
$$;


--
-- Name: fn_timeline(uuid, uuid, bigint); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_timeline(p_world_id uuid, p_viewer_id uuid, p_before_tick bigint DEFAULT NULL::bigint) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  WITH mine AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE v.holder_id = p_viewer_id                              -- FILTER 2: relevance = own holdings
      AND (p_before_tick IS NULL OR v.valid_tick < p_before_tick)
  )
  SELECT json_build_object(
    'schema_version', 'timeline/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'records', coalesce(
      (SELECT json_agg(json_build_object(
                'perception_id',    m.perception_id,
                'content',          m.content,
                'epistemic_type',   m.epistemic_type,
                'occurred_at_tick', m.valid_tick,
                'display_label',    m.in_world_label,
                'confidence',       m.confidence,
                'decay', json_build_object('stale', false, 'last_confirmed_label', m.in_world_label))
              ORDER BY m.valid_tick, m.perception_id)
       FROM mine m), '[]'::json)
  );
$$;


--
-- Name: perception_record; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perception_record (
    perception_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    holder_id uuid NOT NULL,
    source_event_id uuid NOT NULL,
    content text NOT NULL,
    epistemic_type text NOT NULL,
    sensory_mode text,
    confidence real DEFAULT 1.0 NOT NULL,
    distortion_level real DEFAULT 0 NOT NULL,
    acquired_tick bigint NOT NULL,
    valid_tick bigint NOT NULL,
    invalid_tick bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expired_at timestamp with time zone,
    visibility_scope text DEFAULT 'private'::text NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    importance real DEFAULT 5.0 NOT NULL,
    CONSTRAINT perception_record_epistemic_type_check CHECK ((epistemic_type = ANY (ARRAY['direct'::text, 'shared'::text, 'told'::text, 'overheard'::text, 'public'::text, 'rumor'::text, 'inference'::text, 'mistaken'::text, 'confirmed'::text, 'disputed'::text])))
);


--
-- Name: fn_visible_perceptions(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_visible_perceptions(p_world_id uuid, p_viewer_id uuid) RETURNS SETOF public.perception_record
    LANGUAGE sql STABLE
    AS $$
  SELECT pr.*
  FROM perception_record pr
  WHERE pr.world_id = p_world_id
    AND pr.invalid_tick IS NULL
    AND pr.expired_at  IS NULL
    AND ( pr.holder_id = p_viewer_id
          OR pr.holder_id IN (
            SELECT er.entity_id FROM entity_registry er
            WHERE er.world_id = p_world_id
              AND er.entity_kind IN ('faction','group')
          )
        );
$$;


--
-- Name: forbid_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.forbid_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  RAISE EXCEPTION 'DELETE forbidden on % (append-only canon, ADR-001/006)', TG_TABLE_NAME;
END $$;


--
-- Name: gather_slice(uuid, uuid[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.gather_slice(p_world_id uuid, p_ids uuid[]) RETURNS jsonb
    LANGUAGE sql STABLE
    AS $$
  SELECT jsonb_build_object(
    'entities',      COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'id',   er.entity_id,
          'kind', er.entity_kind,
          'name', er.canonical_name,
          'attrs', COALESCE(
            CASE er.entity_kind
              WHEN 'actor'    THEN (SELECT ast.attrs FROM actor_state    ast WHERE ast.entity_id = er.entity_id)
              WHEN 'artifact' THEN (SELECT aft.attrs FROM artifact_state aft WHERE aft.entity_id = er.entity_id)
              WHEN 'location' THEN (SELECT lst.attrs FROM location_state lst WHERE lst.entity_id = er.entity_id)
              ELSE NULL
            END,
            '{}'::jsonb
          )
        )
      )
      FROM entity_registry er
      WHERE er.world_id  = p_world_id
        AND er.entity_id = ANY(p_ids)
    ), '[]'::jsonb),

    'relationships', COALESCE((
      SELECT jsonb_agg(to_jsonb(rs))
      FROM relationship_state rs
      WHERE rs.world_id = p_world_id
        AND (rs.a_id = ANY(p_ids) OR rs.b_id = ANY(p_ids))
    ), '[]'::jsonb),

    'recent_events', COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'event_id', sub.event_id,
          'type',     sub.event_type,
          'tick',     sub.in_world_tick,
          'seq',      sub.beat_seq,
          'summary',  sub.summary
        )
        ORDER BY sub.in_world_tick DESC, sub.beat_seq DESC
      )
      FROM (
        SELECT DISTINCT ON (ranked.event_id)
               ranked.event_id,
               ranked.event_type,
               ranked.in_world_tick,
               ranked.beat_seq,
               ranked.summary
        FROM (
          SELECT
            ce.event_id,
            ce.event_type,
            ce.in_world_tick,
            ce.beat_seq,
            ce.summary,
            ep.entity_id,
            ROW_NUMBER() OVER (
              PARTITION BY ep.entity_id
              ORDER BY ce.in_world_tick DESC, ce.beat_seq DESC
            ) AS rn
          FROM event_participant ep
          JOIN canon_event ce ON ce.event_id = ep.event_id
          WHERE ep.entity_id = ANY(p_ids)
            AND ce.world_id  = p_world_id
        ) ranked
        WHERE ranked.rn <= 10
        ORDER BY ranked.event_id, ranked.in_world_tick DESC, ranked.beat_seq DESC
      ) sub
    ), '[]'::jsonb),

    -- co_present: DISTINCT union of actors at EVERY location any actor in p_ids occupies,
    -- minus p_ids themselves (they already appear in 'entities').
    'co_present', COALESCE((
      SELECT jsonb_agg(DISTINCT jsonb_build_object('id', er.entity_id, 'name', er.canonical_name))
      FROM entity_registry er
      WHERE er.world_id = p_world_id
        AND er.entity_id IN (
          SELECT fa.entity_id
          FROM ( SELECT DISTINCT (ast.attrs->>'location_id')::uuid AS loc
                 FROM actor_state ast
                 WHERE ast.world_id = p_world_id
                   AND ast.entity_id = ANY(p_ids)
                   AND ast.attrs ? 'location_id' ) locs,
          LATERAL fn_actors_at(p_world_id, locs.loc) fa )
        AND NOT (er.entity_id = ANY(p_ids))
    ), '[]'::jsonb)
  )
$$;


--
-- Name: generate_perceptions(uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.generate_perceptions(p_event_id uuid) RETURNS integer
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev   canon_event;
  n    integer := 0;
  spk  uuid;
  lst  uuid;
  pid  uuid;
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
      VALUES (ev.world_id, spk, p_event_id, ev.summary, 'shared', ev.in_world_tick, ev.in_world_tick)
      RETURNING perception_id INTO pid;
      -- about-ness: subjects = the source event's participants (RULINGS-2026-07-23 §6).
      INSERT INTO perception_subject (perception_id, entity_id, world_id)
      SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
      WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
      n := n + 1;
    END IF;
    FOR lst IN SELECT entity_id FROM event_participant
                 WHERE event_id = p_event_id AND role_qualifier = 'listener' LOOP
      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, lst, p_event_id, ev.summary, 'told', ev.in_world_tick, ev.in_world_tick)
      RETURNING perception_id INTO pid;
      INSERT INTO perception_subject (perception_id, entity_id, world_id)
      SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
      WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
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
        VALUES (ev.world_id, mover, p_event_id, ev.summary, 'direct', ev.in_world_tick, ev.in_world_tick)
        RETURNING perception_id INTO pid;
        -- about-ness: subjects = the source event's participants (RULINGS-2026-07-23 §6).
        INSERT INTO perception_subject (perception_id, entity_id, world_id)
        SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
        WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
        n := n + 1;
        -- discovery-on-arrival (§4 trigger 2): the mover perceives each actor ALREADY at dest
        -- (exclude self). Each carries an explicit subject link → the stop-check reads about-ness.
        -- UNTOUCHED: this loop already wrote subjects before this migration.
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


--
-- Name: replay_0a(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.replay_0a() RETURNS boolean
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE ev RECORD; m state_mutation; diff_count int;
BEGIN
  DROP TABLE IF EXISTS snap_actor, snap_location, snap_artifact, snap_rel;
  CREATE TEMP TABLE snap_actor    ON COMMIT DROP AS SELECT * FROM actor_state;
  CREATE TEMP TABLE snap_location ON COMMIT DROP AS SELECT * FROM location_state;
  CREATE TEMP TABLE snap_artifact ON COMMIT DROP AS SELECT * FROM artifact_state;
  CREATE TEMP TABLE snap_rel      ON COMMIT DROP AS SELECT * FROM relationship_state;

  TRUNCATE actor_state, location_state, artifact_state, relationship_state;

  -- Rider C: domain-only deterministic order. recorded_at (volatile) excluded.
  FOR ev IN SELECT event_id FROM canon_event WHERE status='accepted'
            ORDER BY world_id, in_world_tick, beat_seq LOOP
    FOR m IN SELECT * FROM state_mutation WHERE event_id = ev.event_id
             ORDER BY valid_from_tick, valid_from_seq LOOP
      PERFORM apply_mutation(m);
    END LOOP;
  END LOOP;

  -- §6.5.1 per-table domain diff (exclude volatile updated_at; identity = PK).
  SELECT
      (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state)) d)
    + (SELECT count(*) FROM (
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel)
        UNION ALL
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state)) d)
  INTO diff_count;
  RETURN diff_count = 0;
END $$;


--
-- Name: seed_world_defaults(uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.seed_world_defaults(p_world_id uuid) RETURNS void
    LANGUAGE sql
    AS $$
  INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps)
  VALUES (p_world_id, 'walk', 1.4) ON CONFLICT DO NOTHING;
  INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
  VALUES (p_world_id, 'encumbered', 'move', 'walk', -100) ON CONFLICT DO NOTHING;
$$;


--
-- Name: sm_project(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.sm_project() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
BEGIN
  IF (SELECT status FROM canon_event WHERE event_id = NEW.event_id) = 'accepted' THEN
    PERFORM apply_mutation(NEW);
  END IF;
  RETURN NEW;
END $$;


--
-- Name: trg_validate_tension(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.trg_validate_tension() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  IF NEW.attrs ? 'tension' AND NOT (NEW.attrs->>'tension' IN ('frantic','tense','normal','calm','none')) THEN
    RAISE EXCEPTION 'tension % not in enum', NEW.attrs->>'tension';
  END IF;
  RETURN NEW;
END $$;


--
-- Name: actor_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.actor_state (
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: artifact_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artifact_state (
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: canon_event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canon_event (
    event_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    scene_id uuid,
    beat_id uuid,
    event_type text NOT NULL,
    summary text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    schema_version integer DEFAULT 1 NOT NULL,
    in_world_tick bigint NOT NULL,
    in_world_label text,
    beat_seq integer DEFAULT 0 NOT NULL,
    temporal_uncertainty boolean DEFAULT false NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    accepted_at timestamp with time zone,
    status text DEFAULT 'proposed'::text NOT NULL,
    visibility_scope text DEFAULT 'private'::text NOT NULL,
    confidence real,
    origin text DEFAULT 'fast_path'::text NOT NULL,
    template_id text,
    source_refs jsonb,
    superseded_by uuid,
    CONSTRAINT canon_event_event_type_check CHECK ((event_type = ANY (ARRAY['ActorMoved'::text, 'Communicated'::text, 'ObjectRelocated'::text, 'OwnershipAccessChanged'::text, 'EntityCreated'::text, 'EntityDestroyed'::text, 'AttributeChanged'::text, 'move'::text, 'private_disclosure'::text, 'world_genesis'::text, 'observation'::text, 'publicize'::text]))),
    CONSTRAINT canon_event_origin_check CHECK ((origin = ANY (ARRAY['fast_path'::text, 'template'::text, 'freeform'::text, 'threshold'::text, 'backstage'::text, 'compensation'::text, 'ruling'::text, 'telegraph'::text]))),
    CONSTRAINT canon_event_status_check CHECK ((status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'retconned'::text, 'superseded'::text])))
);


--
-- Name: causal_bundle; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.causal_bundle (
    bundle_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    effect_ref uuid NOT NULL,
    effect_kind text NOT NULL,
    semantics text NOT NULL,
    template_id text,
    status text DEFAULT 'valid'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT causal_bundle_effect_kind_check CHECK ((effect_kind = ANY (ARRAY['event'::text, 'mutation'::text]))),
    CONSTRAINT causal_bundle_semantics_check CHECK ((semantics = ANY (ARRAY['conjunctive'::text, 'disjunctive_member'::text, 'probabilistic'::text]))),
    CONSTRAINT causal_bundle_status_check CHECK ((status = ANY (ARRAY['valid'::text, 'invalidated'::text, 'pending_review'::text])))
);


--
-- Name: causal_bundle_input; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.causal_bundle_input (
    bundle_id uuid NOT NULL,
    input_ref uuid NOT NULL,
    input_kind text NOT NULL,
    role text NOT NULL,
    polarity smallint DEFAULT 1 NOT NULL,
    weight real DEFAULT 1.0 NOT NULL,
    necessity boolean DEFAULT true NOT NULL,
    CONSTRAINT causal_bundle_input_input_kind_check CHECK ((input_kind = ANY (ARRAY['event'::text, 'mutation'::text, 'perception'::text]))),
    CONSTRAINT causal_bundle_input_polarity_check CHECK ((polarity = ANY (ARRAY[1, '-1'::integer]))),
    CONSTRAINT causal_bundle_input_role_check CHECK ((role = ANY (ARRAY['trigger'::text, 'enabler'::text, 'blocker'::text, 'influence'::text])))
);


--
-- Name: entity_registry; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_registry (
    entity_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    entity_kind text NOT NULL,
    canonical_name text NOT NULL,
    aliases text[] DEFAULT '{}'::text[] NOT NULL,
    descriptor text,
    current_scene_id uuid,
    created_by_event uuid,
    status text DEFAULT 'active'::text NOT NULL,
    CONSTRAINT entity_registry_status_check CHECK ((status = ANY (ARRAY['active'::text, 'inactive'::text, 'merged'::text])))
);


--
-- Name: event_participant; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_participant (
    event_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    entity_kind text NOT NULL,
    role_qualifier text NOT NULL,
    CONSTRAINT event_participant_entity_kind_check CHECK ((entity_kind = ANY (ARRAY['actor'::text, 'location'::text, 'artifact'::text, 'faction'::text, 'group'::text])))
);


--
-- Name: held_outcome; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.held_outcome (
    held_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    attempt jsonb NOT NULL,
    telegraph_event_id uuid NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    created_tick bigint NOT NULL,
    CONSTRAINT held_outcome_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'resolved'::text])))
);


--
-- Name: location_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.location_state (
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: movement_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.movement_type (
    world_id uuid NOT NULL,
    movement_type_id text NOT NULL,
    base_speed_mps numeric NOT NULL,
    CONSTRAINT movement_type_base_speed_mps_check CHECK ((base_speed_mps > (0)::numeric))
);


--
-- Name: perception_subject; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perception_subject (
    perception_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL
);


--
-- Name: personality_core; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.personality_core (
    world_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    traits jsonb NOT NULL,
    malleability numeric NOT NULL,
    CONSTRAINT personality_core_malleability_check CHECK (((malleability > (0)::numeric) AND (malleability <= (1)::numeric)))
);


--
-- Name: provenance_edge; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.provenance_edge (
    derived_id uuid NOT NULL,
    derived_kind text NOT NULL,
    source_id uuid NOT NULL,
    source_kind text NOT NULL,
    how_type text NOT NULL,
    CONSTRAINT provenance_edge_derived_kind_check CHECK ((derived_kind = ANY (ARRAY['perception'::text, 'mutation'::text, 'event'::text, 'bundle'::text]))),
    CONSTRAINT provenance_edge_how_type_check CHECK ((how_type = ANY (ARRAY['derived_from'::text, 'inferred_from'::text, 'reported_by'::text, 'witnessed_by'::text, 'compensates'::text, 'supersedes'::text]))),
    CONSTRAINT provenance_edge_source_kind_check CHECK ((source_kind = ANY (ARRAY['perception'::text, 'mutation'::text, 'event'::text])))
);


--
-- Name: relationship_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.relationship_state (
    world_id uuid NOT NULL,
    a_id uuid NOT NULL,
    b_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying(128) NOT NULL
);


--
-- Name: status_modifier; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.status_modifier (
    world_id uuid NOT NULL,
    status_type_id text NOT NULL,
    action_type text NOT NULL,
    movement_type_id text NOT NULL,
    modifier_percent numeric NOT NULL,
    CONSTRAINT status_modifier_action_type_check CHECK ((action_type = 'move'::text)),
    CONSTRAINT status_modifier_modifier_percent_check CHECK ((modifier_percent >= ('-100'::integer)::numeric))
);


--
-- Name: trait_pool; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trait_pool (
    world_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    trait_key text NOT NULL,
    accrued numeric DEFAULT 0 NOT NULL,
    threshold numeric NOT NULL
);


--
-- Name: trait_provenance; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trait_provenance (
    world_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    trait_key text NOT NULL,
    event_id uuid NOT NULL
);


--
-- Name: world_pressure; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_pressure (
    world_id uuid NOT NULL,
    tier text NOT NULL,
    accrued numeric DEFAULT 0 NOT NULL,
    last_fired_tick bigint DEFAULT 0 NOT NULL,
    CONSTRAINT world_pressure_tier_check CHECK ((tier = ANY (ARRAY['small'::text, 'medium'::text, 'large'::text])))
);


--
-- Name: actor_state actor_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.actor_state
    ADD CONSTRAINT actor_state_pkey PRIMARY KEY (entity_id);


--
-- Name: artifact_state artifact_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_state
    ADD CONSTRAINT artifact_state_pkey PRIMARY KEY (entity_id);


--
-- Name: canon_event canon_event_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canon_event
    ADD CONSTRAINT canon_event_pkey PRIMARY KEY (event_id);


--
-- Name: causal_bundle_input causal_bundle_input_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.causal_bundle_input
    ADD CONSTRAINT causal_bundle_input_pkey PRIMARY KEY (bundle_id, input_ref, role);


--
-- Name: causal_bundle causal_bundle_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.causal_bundle
    ADD CONSTRAINT causal_bundle_pkey PRIMARY KEY (bundle_id);


--
-- Name: entity_registry entity_registry_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_registry
    ADD CONSTRAINT entity_registry_pkey PRIMARY KEY (entity_id);


--
-- Name: event_participant event_participant_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_participant
    ADD CONSTRAINT event_participant_pkey PRIMARY KEY (event_id, entity_id, role_qualifier);


--
-- Name: held_outcome held_outcome_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.held_outcome
    ADD CONSTRAINT held_outcome_pkey PRIMARY KEY (held_id);


--
-- Name: location_state location_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.location_state
    ADD CONSTRAINT location_state_pkey PRIMARY KEY (entity_id);


--
-- Name: movement_type movement_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.movement_type
    ADD CONSTRAINT movement_type_pkey PRIMARY KEY (world_id, movement_type_id);


--
-- Name: pending_event pending_event_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pending_event
    ADD CONSTRAINT pending_event_pkey PRIMARY KEY (pending_id);


--
-- Name: perception_record perception_record_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_record
    ADD CONSTRAINT perception_record_pkey PRIMARY KEY (perception_id);


--
-- Name: perception_subject perception_subject_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_subject
    ADD CONSTRAINT perception_subject_pkey PRIMARY KEY (perception_id, entity_id);


--
-- Name: personality_core personality_core_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.personality_core
    ADD CONSTRAINT personality_core_pkey PRIMARY KEY (actor_id);


--
-- Name: provenance_edge provenance_edge_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provenance_edge
    ADD CONSTRAINT provenance_edge_pkey PRIMARY KEY (derived_id, source_id, how_type);


--
-- Name: relationship_state relationship_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.relationship_state
    ADD CONSTRAINT relationship_state_pkey PRIMARY KEY (world_id, a_id, b_id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: state_mutation state_mutation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.state_mutation
    ADD CONSTRAINT state_mutation_pkey PRIMARY KEY (mutation_id);


--
-- Name: status_modifier status_modifier_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.status_modifier
    ADD CONSTRAINT status_modifier_pkey PRIMARY KEY (world_id, status_type_id, action_type, movement_type_id);


--
-- Name: trait_pool trait_pool_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trait_pool
    ADD CONSTRAINT trait_pool_pkey PRIMARY KEY (actor_id, trait_key);


--
-- Name: trait_provenance trait_provenance_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trait_provenance
    ADD CONSTRAINT trait_provenance_pkey PRIMARY KEY (actor_id, trait_key, event_id);


--
-- Name: world_pressure world_pressure_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_pressure
    ADD CONSTRAINT world_pressure_pkey PRIMARY KEY (world_id, tier);


--
-- Name: idx_cb_effect; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cb_effect ON public.causal_bundle USING btree (effect_ref);


--
-- Name: idx_ce_beat; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_beat ON public.canon_event USING btree (beat_id);


--
-- Name: idx_ce_payload_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_payload_gin ON public.canon_event USING gin (payload);


--
-- Name: idx_ce_scene; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_scene ON public.canon_event USING btree (scene_id);


--
-- Name: idx_ce_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_status ON public.canon_event USING btree (world_id, status) WHERE (status = 'accepted'::text);


--
-- Name: idx_ce_world_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_world_time ON public.canon_event USING btree (world_id, in_world_tick, beat_seq);


--
-- Name: idx_ep_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ep_entity ON public.event_participant USING btree (entity_id);


--
-- Name: idx_er_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_er_name ON public.entity_registry USING btree (world_id, canonical_name);


--
-- Name: idx_er_scene; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_er_scene ON public.entity_registry USING btree (world_id, current_scene_id);


--
-- Name: idx_held_outcome_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_held_outcome_pending ON public.held_outcome USING btree (world_id) WHERE (status = 'pending'::text);


--
-- Name: idx_pe_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pe_source ON public.provenance_edge USING btree (source_id);


--
-- Name: idx_pr_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_active ON public.perception_record USING btree (holder_id) WHERE ((invalid_tick IS NULL) AND (expired_at IS NULL));


--
-- Name: idx_pr_holder; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_holder ON public.perception_record USING btree (holder_id, acquired_tick);


--
-- Name: idx_pr_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_source ON public.perception_record USING btree (source_event_id);


--
-- Name: idx_ps_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ps_entity ON public.perception_subject USING btree (entity_id);


--
-- Name: idx_ps_world; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ps_world ON public.perception_subject USING btree (world_id);


--
-- Name: idx_sm_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sm_entity ON public.state_mutation USING btree (entity_id, valid_from_tick, valid_from_seq);


--
-- Name: idx_sm_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sm_event ON public.state_mutation USING btree (event_id);


--
-- Name: uq_ce_accepted_order; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_ce_accepted_order ON public.canon_event USING btree (world_id, in_world_tick, beat_seq) WHERE (status = 'accepted'::text);


--
-- Name: location_state location_state_tension; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER location_state_tension BEFORE INSERT OR UPDATE ON public.location_state FOR EACH ROW EXECUTE FUNCTION public.trg_validate_tension();


--
-- Name: canon_event trg_canon_event_append_only; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_canon_event_append_only BEFORE UPDATE ON public.canon_event FOR EACH ROW EXECUTE FUNCTION public.canon_event_append_only();


--
-- Name: canon_event trg_canon_event_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_canon_event_no_delete BEFORE DELETE ON public.canon_event FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: causal_bundle trg_causal_bundle_append_only; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_append_only BEFORE UPDATE ON public.causal_bundle FOR EACH ROW EXECUTE FUNCTION public.causal_bundle_append_only();


--
-- Name: causal_bundle_input trg_causal_bundle_input_immutable; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_input_immutable BEFORE UPDATE ON public.causal_bundle_input FOR EACH ROW EXECUTE FUNCTION public.causal_bundle_input_immutable();


--
-- Name: causal_bundle_input trg_causal_bundle_input_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_input_no_delete BEFORE DELETE ON public.causal_bundle_input FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: causal_bundle trg_causal_bundle_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_no_delete BEFORE DELETE ON public.causal_bundle FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: causal_bundle_input trg_cbi_assert_acyclic; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_cbi_assert_acyclic BEFORE INSERT ON public.causal_bundle_input FOR EACH ROW EXECUTE FUNCTION public.causal_bundle_assert_acyclic();


--
-- Name: event_participant trg_event_participant_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_event_participant_no_delete BEFORE DELETE ON public.event_participant FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: perception_record trg_perception_record_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_perception_record_no_delete BEFORE DELETE ON public.perception_record FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: perception_subject trg_perception_subject_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_perception_subject_no_delete BEFORE DELETE ON public.perception_subject FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: provenance_edge trg_provenance_edge_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_provenance_edge_no_delete BEFORE DELETE ON public.provenance_edge FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: state_mutation trg_sm_project; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_sm_project AFTER INSERT ON public.state_mutation FOR EACH ROW EXECUTE FUNCTION public.sm_project();


--
-- Name: state_mutation trg_state_mutation_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_state_mutation_no_delete BEFORE DELETE ON public.state_mutation FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: canon_event canon_event_superseded_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canon_event
    ADD CONSTRAINT canon_event_superseded_by_fkey FOREIGN KEY (superseded_by) REFERENCES public.canon_event(event_id);


--
-- Name: causal_bundle_input causal_bundle_input_bundle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.causal_bundle_input
    ADD CONSTRAINT causal_bundle_input_bundle_id_fkey FOREIGN KEY (bundle_id) REFERENCES public.causal_bundle(bundle_id);


--
-- Name: entity_registry entity_registry_created_by_event_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_registry
    ADD CONSTRAINT entity_registry_created_by_event_fkey FOREIGN KEY (created_by_event) REFERENCES public.canon_event(event_id);


--
-- Name: event_participant event_participant_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_participant
    ADD CONSTRAINT event_participant_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.canon_event(event_id);


--
-- Name: held_outcome held_outcome_telegraph_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.held_outcome
    ADD CONSTRAINT held_outcome_telegraph_event_id_fkey FOREIGN KEY (telegraph_event_id) REFERENCES public.canon_event(event_id);


--
-- Name: perception_record perception_record_source_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_record
    ADD CONSTRAINT perception_record_source_event_id_fkey FOREIGN KEY (source_event_id) REFERENCES public.canon_event(event_id);


--
-- Name: perception_subject perception_subject_perception_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_subject
    ADD CONSTRAINT perception_subject_perception_id_fkey FOREIGN KEY (perception_id) REFERENCES public.perception_record(perception_id);


--
-- Name: state_mutation state_mutation_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.state_mutation
    ADD CONSTRAINT state_mutation_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.canon_event(event_id);


--
-- Name: status_modifier status_modifier_world_id_movement_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.status_modifier
    ADD CONSTRAINT status_modifier_world_id_movement_type_id_fkey FOREIGN KEY (world_id, movement_type_id) REFERENCES public.movement_type(world_id, movement_type_id);


--
-- Name: trait_provenance trait_provenance_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trait_provenance
    ADD CONSTRAINT trait_provenance_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.canon_event(event_id);


--
-- PostgreSQL database dump complete
--


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20260610090001'),
    ('20260610090002'),
    ('20260610090003'),
    ('20260610090004'),
    ('20260610090005'),
    ('20260610090006'),
    ('20260610090007'),
    ('20260611090001'),
    ('20260614090001'),
    ('20260614090002'),
    ('20260615090001'),
    ('20260618090001'),
    ('20260723100001'),
    ('20260723100002'),
    ('20260723100003'),
    ('20260723100004'),
    ('20260724100001'),
    ('20260724100002'),
    ('20260724110001'),
    ('20260724110002'),
    ('20260724110003'),
    ('20260724110004'),
    ('20260726100001'),
    ('20260729100001'),
    ('20260729100002');
