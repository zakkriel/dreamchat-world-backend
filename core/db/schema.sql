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
        'type',         'ActorMoved',
        'stated',       'move',
        'to_target_id', step->>'to'
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
  ev_type      text;
  ev_id        uuid;
  listener     uuid;
  to_target    uuid;   -- the move's target: ANY positioned entity (location | object | actor)
  v_scene      uuid;   -- resolved: the target's containing scene (a location resolves to itself)
  v_coord      jsonb;  -- resolved: the target's position — the mover's new coordinate
  object_eid   uuid;
  dest_eid     uuid;
  target_eid   uuid;
  here         uuid;
  vis_scope    text;
  final_type   text;
  v_old_holder uuid;   -- object's current carrier, read before the move (eager rule, §4)
  v_witness    uuid;   -- SPEC-035: each named witness, validated co-present before it is recorded
BEGIN
  ev_type := p_attempt->>'type';

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = p_actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_attempt->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, p_actor_id, to_target) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_attempt->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
    IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
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

    -- SPEC-035: a named witness must have been THERE. Co-presence is necessary but not sufficient -
    -- the founder's ruling is "just because they were there doesn't mean they saw it" - so the caller
    -- names who saw it and the gate refuses anyone who could not have. This is byte-for-byte the shape
    -- of the 'Communicated' listener gate above (fn_actors_at against the instigator's location), and
    -- it is deliberate: the engine BLOCKS impossibilities, it never awards perception
    -- (FINAL-action-contracts.md - "deterministic machinery blocks impossibilities").
    -- SPEC-035 amendment: PRESENT-BUT-MALFORMED IS A REFUSAL, NOT A SHRUG.
    -- The first cut of this gate keyed on `= 'array'`, so `witnesses: "<uuid>"` — a bare string —
    -- fell through every branch: committed, zero witness rows, zero perceptions, no halt_reason.
    -- That is the EXACT defect class this SPEC was filed to remove, reintroduced by its own fix.
    -- It is not a hypothetical shape either: the sibling field on 'Communicated' is `listener_id`,
    -- a bare string, so a caller following the nearest precedent writes precisely this and is met
    -- with silence. Absent is fine and means "nobody named"; present and not an array is a caller
    -- bug, and the engine's job is to block it loudly rather than guess what was meant. Coercing a
    -- string into a one-element array was rejected: coercion hides the caller's bug, which is how
    -- the silence got here in the first place.
    IF p_attempt ? 'witnesses'
       AND jsonb_typeof(p_attempt->'witnesses') NOT IN ('array', 'null') THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

    IF jsonb_typeof(p_attempt->'witnesses') = 'array' THEN
      SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      FOR v_witness IN SELECT (value #>> '{}')::uuid FROM jsonb_array_elements(p_attempt->'witnesses') LOOP
        IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = v_witness) THEN
          RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
        END IF;
      END LOOP;
    END IF;
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_attempt->>'target_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- DESCRIPTOR MANDATORY (§8 / §7 provenance-guarded create): no descriptor ⇒ not a true introduction
    -- ⇒ gate_reject. Blocker-only: the reality-check/adjudication already happened; this is the commit
    -- clerk refusing an un-introduced create. (Twin of apply_ruled_event — the shared floor rule.)
    IF NULLIF(btrim(COALESCE(p_attempt->>'descriptor','')),'') IS NULL THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  IF p_legacy_types THEN
    final_type := CASE ev_type
      WHEN 'Communicated' THEN 'private_disclosure'
      WHEN 'ActorMoved'   THEN 'move'
      ELSE ev_type
    END;
  ELSE
    final_type := ev_type;
  END IF;

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin, payload)
  VALUES (ev_id, p_world_id, final_type, p_attempt->>'stated',
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin,
          -- SPOKEN WORDS (SPEC-033 follow-up). `stated` is the referee's account of the utterance
          -- ("tell her about the note"); `content` is what was actually SAID. Only the latter can back
          -- a verbatim quote, and until now it was dropped on the floor here — so canon recorded that
          -- someone spoke and never what they said, and every speech segment the narrator wrote was
          -- correctly refused as unverifiable. Stored only when non-empty, and only for speech.
          CASE WHEN ev_type = 'Communicated'
                AND NULLIF(TRIM(COALESCE(p_attempt->>'content','')),'') IS NOT NULL
               THEN jsonb_build_object('spoken', TRIM(p_attempt->>'content'))
               ELSE '{}'::jsonb END);

  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, p_actor_id, 'actor', 'speaker'),
      (ev_id, listener,   'actor', 'listener');
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, p_actor_id, 'actor', 'instigator');
  END IF;

  -- SPEC-035: THE EVENT NAMES WHO SAW IT. Measured before this was written: a caller could already
  -- pass 'witnesses' and apply_event discarded it without a word - the payload came back {} and
  -- event_participant held nothing but 'instigator', so the question "who watched this handover"
  -- had no answer anywhere in the database. That is not a missing feature, it is a silent drop.
  --
  -- WHY A PARTICIPANT ROW AND NOT A PAYLOAD FIELD. 'Communicated' already records its recipients as
  -- 'listener' participants and generate_perceptions reads them back by role_qualifier; this is the
  -- same question with the same answer, so it takes the same shape (B-2: the event names its
  -- participants). A payload field would have been a second mechanism for one meaning, and the
  -- payload is exactly where SPEC-034 proved data goes to die: a committed ObjectRelocated's payload
  -- is '{}'.
  --
  -- WHY IT CANNOT BE BACKFILLED, which is why this lands now rather than later: replay_0A() asserts
  -- it reproduces domain-equivalent PROJECTION state, and perceptions are not projections - they are
  -- not regenerated on replay (ADR-026). An event committed without its witnesses is unwitnessable
  -- forever. Every handover accepted before this migration is already in that state.
  IF ev_type = 'ObjectRelocated' AND jsonb_typeof(p_attempt->'witnesses') = 'array' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
    SELECT ev_id, w.eid, 'actor', 'witness'
      FROM (SELECT DISTINCT (value #>> '{}')::uuid AS eid
              FROM jsonb_array_elements(p_attempt->'witnesses')) w
     WHERE w.eid IS NOT NULL
       -- A holder is not a witness. Both already perceive as parties to the handover (SPEC-034), and
       -- a duplicate row would mint them a second perception of one event.
       AND w.eid <> p_actor_id
       AND w.eid <> dest_eid
    ON CONFLICT DO NOTHING;
  END IF;

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  -- EntityCreated: persist the new instance (registry row + positioned state) via the shared helper.
  -- Reuse-before-create is inside the helper. Provenance = ev_id (§8 net 3); one logged canon_event row
  -- above (net 2); the ruling already passed the reality check (net 1).
  IF ev_type = 'EntityCreated' THEN
    PERFORM fn_apply_entity_created(ev_id, p_world_id,
      NULLIF(p_attempt->>'target_id','')::uuid,
      p_attempt->>'new_entity_kind',
      p_attempt->>'canonical_name',
      p_attempt->>'descriptor',
      COALESCE(p_attempt->'new_attrs', '{}'::jsonb));
  END IF;

  PERFORM generate_perceptions(ev_id);

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;


--
-- Name: apply_mint(uuid, uuid, jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.apply_mint(p_world_id uuid, p_ruling_event uuid, p_mint jsonb) RETURNS text
    LANGUAGE plpgsql
    AS $$
DECLARE
  v_action text;
  v_row    jsonb;
  v_status text;
BEGIN
  -- MOVEMENT-TYPE mint: { movementTypeId, baseSpeed }.
  IF p_mint ? 'baseSpeed' THEN
    INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps, created_by_event)
    VALUES (p_world_id, p_mint->>'movementTypeId', (p_mint->>'baseSpeed')::numeric, p_ruling_event)
    ON CONFLICT (world_id, movement_type_id) DO NOTHING;
    RETURN p_mint->>'movementTypeId';

  -- MODIFIER mint: { statusTypeId, actionType, movementModifiers:[{ movementTypeId, modifierPercent }] }.
  ELSIF p_mint ? 'statusTypeId' OR p_mint ? 'movementModifiers' THEN
    v_action := COALESCE(p_mint->>'actionType', 'move');   -- 'move' is the only metered action (§6)
    v_status := p_mint->>'statusTypeId';
    FOR v_row IN SELECT jsonb_array_elements(COALESCE(p_mint->'movementModifiers', '[]'::jsonb))
    LOOP
      -- FK (world_id, movement_type_id) → movement_type: satisfied by ordering (validated in Go; the
      -- referenced type was minted by an earlier apply_mint in this tx, or is seeded/committed).
      INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id,
                                   modifier_percent, created_by_event)
      VALUES (p_world_id, v_status, v_action, v_row->>'movementTypeId',
              (v_row->>'modifierPercent')::numeric, p_ruling_event)
      ON CONFLICT (world_id, status_type_id, action_type, movement_type_id) DO NOTHING;
    END LOOP;
    RETURN v_status;

  -- ARTIFACT/PLACE mint: persistence escalated (see header) — refuse to guess the schema.
  ELSIF p_mint ? 'size' OR p_mint ? 'maxRoom' OR p_mint ? 'coordinate'
     OR p_mint ? 'parentLocationId' OR p_mint ? 'locationId' THEN
    RAISE EXCEPTION
      'apply_mint: artifact/place mint persistence is not yet specified (escalated: the entity_registry+state schema is undefined in FINAL §8) — refusing to guess. mint=%',
      p_mint;

  ELSE
    RAISE EXCEPTION 'apply_mint: unrecognized mint shape (no baseSpeed / statusTypeId / artifact fields). mint=%', p_mint;
  END IF;
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
  -- strip leading 'attrs.' (6 chars) -> single-key JSON path under attrs (ABSOLUTE-STATE-SETS, was 0A Rider B)
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
  to_target  uuid;   -- the move's target: ANY positioned entity (location | object | actor)
  v_scene    uuid;   -- resolved: the target's containing scene
  v_coord    jsonb;  -- resolved: the target's position — the mover's new coordinate
  object_eid uuid;
  dest_eid   uuid;
  target_eid uuid;
  here       uuid;
  vis_scope  text;
  truth_text text;
  appear_txt text;
  visible    boolean;
  receiver   uuid;
  recv_text  text;
  var_text   text;
  pid        uuid;
  participant_ids uuid[];
  v_old_holder uuid;   -- object's current carrier, read before the move (eager rule, §4)
BEGIN
  ev_type  := p_ruled->>'type';
  actor_id := (p_ruled->>'actor_id')::uuid;
  truth_text := p_ruled->>'truth';
  appear_txt := NULLIF(TRIM(COALESCE(p_ruled->>'appearance', '')), '');
  visible    := CASE
    WHEN p_ruled ? 'visible' AND (p_ruled->>'visible') = 'false' THEN false
    ELSE true
  END;

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_ruled->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, actor_id, to_target) THEN
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
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_ruled->>'target_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- DESCRIPTOR MANDATORY (§8 / §7 provenance-guarded create): no descriptor ⇒ not a true introduction
    -- ⇒ gate_reject. Blocker-only clerk (the adjudication already happened). Twin of apply_event.
    IF NULLIF(btrim(COALESCE(p_ruled->>'descriptor','')),'') IS NULL THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin, payload)
  VALUES (ev_id, p_world_id, ev_type, truth_text,
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin,
          -- Same rule on the ruled path: `truth` is the referee's account, `content` is the words.
          CASE WHEN ev_type = 'Communicated'
                AND NULLIF(TRIM(COALESCE(p_ruled->>'content','')),'') IS NOT NULL
               THEN jsonb_build_object('spoken', TRIM(p_ruled->>'content'))
               ELSE '{}'::jsonb END);

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

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  -- EntityCreated: persist the new instance via the shared helper (reuse-before-create inside).
  -- Provenance = ev_id (§8 net 3); one logged canon_event row (net 2); ruling passed reality-check (net 1).
  IF ev_type = 'EntityCreated' THEN
    PERFORM fn_apply_entity_created(ev_id, p_world_id,
      NULLIF(p_ruled->>'target_id','')::uuid,
      p_ruled->>'new_entity_kind',
      p_ruled->>'canonical_name',
      p_ruled->>'descriptor',
      COALESCE(p_ruled->'new_attrs', '{}'::jsonb));
  END IF;

  IF NOT visible THEN
    RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
  END IF;

  SELECT (a.attrs->>'location_id')::uuid INTO here
    FROM actor_state a
    WHERE a.world_id = p_world_id AND a.entity_id = actor_id;

  FOR receiver IN
    SELECT entity_id FROM fn_actors_at(p_world_id, here)
    UNION
    SELECT actor_id
  LOOP
    var_text := NULL;
    IF p_ruled ? 'receiver_variants' THEN
      SELECT rv->>'text' INTO var_text
        FROM jsonb_array_elements(p_ruled->'receiver_variants') AS rv
        WHERE (rv->>'receiver_id')::uuid = receiver
        LIMIT 1;
    END IF;

    -- NAMING WALL: rendered for THIS receiver. receiver_variants already differentiate when a
    -- ruling bothered to supply them; this covers the far more common case where it did not and
    -- every holder would otherwise share the referee's canonically-named account.
    recv_text := fn_viewer_text(p_world_id, receiver, COALESCE(var_text, appear_txt, truth_text));

    INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                   acquired_tick, valid_tick)
    VALUES (p_world_id, receiver, ev_id, recv_text, 'direct', p_tick, p_tick)
    RETURNING perception_id INTO pid;

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
-- Name: canon_event_carry_in_world_label(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.canon_event_carry_in_world_label() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  IF NEW.in_world_label IS NULL THEN
    SELECT ce.in_world_label INTO NEW.in_world_label
      FROM canon_event ce
     WHERE ce.world_id = NEW.world_id
       AND ce.in_world_label IS NOT NULL
       AND (ce.in_world_tick, ce.beat_seq) <= (NEW.in_world_tick, NEW.beat_seq)
     ORDER BY ce.in_world_tick DESC, ce.beat_seq DESC
     LIMIT 1;
  END IF;
  RETURN NEW;
END;
$$;


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
-- Name: fn_actor_move_permitted(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_actor_move_permitted(p_world_id uuid, p_actor_id uuid, p_target uuid) RETURNS boolean
    LANGUAGE plpgsql STABLE
    AS $$
DECLARE
  v_here  uuid;
  v_scene uuid;
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM entity_registry WHERE entity_id = p_target AND world_id = p_world_id
  ) THEN
    RETURN false;   -- target not a registered entity: the existence floor (as before Portal)
  END IF;

  SELECT scene INTO v_scene FROM fn_target_position(p_world_id, p_target);

  SELECT (attrs->>'location_id')::uuid INTO v_here
    FROM actor_state WHERE world_id = p_world_id AND entity_id = p_actor_id;

  IF v_here IS NULL OR v_here = v_scene THEN
    RETURN true;    -- same-scene (or no origin): not a traversal → no portal required
  END IF;

  RETURN fn_portal_permits(p_world_id, v_here, v_scene);   -- cross-location: a portal must permit it
END $$;


--
-- Name: fn_actor_page(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_actor_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'actor_page/2',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id',                         p_actor_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      -- Perception rows do not carry a structured role taxonomy for actors.
      'perceived_role',             NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_actor_id),
      'last_known_status',          fn_compendium_latest_fact(p_world_id, p_viewer_id, p_actor_id),
      'known_artifacts',            fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['artifact']),
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['actor','location','artifact']),
      -- NULL until a portrait exists. Never a presigned URL: fn_image_ref emits an asset id and a
      -- path back to this service, which mints a fresh short-lived URL per read, so a payload can be
      -- cached or logged without carrying a credential that expires in fifteen minutes.
      --
      -- NOT perception-scoped, and that is a real decision. A portrait is of the ENTITY, not of the
      -- viewer's opinion of it: two viewers who know an actor by different names still see the same
      -- face, exactly as they would in the room. The wall governs what a viewer KNOWS — names,
      -- facts, synthesis — and this page already renders every one of those through the viewer's own
      -- perception. A picture of a person standing in front of you is not a secret (B-1 unaffected;
      -- the existence gate above still decides whether this page may be seen at all).
      'image',                      fn_image_ref(p_world_id, 'actor', p_actor_id)
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
-- Name: fn_apply_carry_change(uuid, uuid, uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_apply_carry_change(p_event_id uuid, p_world_id uuid, p_object uuid, p_old_holder uuid, p_new_holder uuid) RETURNS void
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  v_tick    bigint;
  v_seq     int;
  v_okind   text;
  v_carrier uuid;
  v_cw      numeric;
  v_maxload numeric;
  v_stat    jsonb;
BEGIN
  -- Provenance tick/seq come from the just-committed relocation event (single source of truth).
  SELECT in_world_tick, beat_seq INTO v_tick, v_seq
  FROM canon_event WHERE event_id = p_event_id;

  SELECT entity_kind INTO v_okind
  FROM entity_registry WHERE world_id = p_world_id AND entity_id = p_object;

  -- (1) new containment edge. Projects into <kind>_state.attrs.contained_by via trg_sm_project.
  INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                              new_value, valid_from_tick, valid_from_seq)
  VALUES (p_world_id, p_event_id, p_object, v_okind, 'attrs.contained_by',
          to_jsonb(p_new_holder::text), v_tick, v_seq);

  -- (2) recompute every affected actor. Walk UP contained_by from {old_holder, new_holder}; the base
  --     rows are the holders themselves (an actor holder is affected directly), the recursion climbs
  --     through nested containers to the root carrier. depth < 64 caps a contained_by cycle (I-4).
  FOR v_carrier IN
    WITH RECURSIVE chain(node, depth) AS (
      SELECT h, 0
      FROM (VALUES (p_old_holder), (p_new_holder)) AS s(h)
      WHERE h IS NOT NULL
      UNION ALL
      SELECT (a.attrs->>'contained_by')::uuid, c.depth + 1
      FROM chain c
      JOIN artifact_state a ON a.world_id = p_world_id AND a.entity_id = c.node
      WHERE a.attrs->>'contained_by' IS NOT NULL AND c.depth < 64
    )
    SELECT DISTINCT c.node
    FROM chain c
    JOIN entity_registry er ON er.world_id = p_world_id AND er.entity_id = c.node
    WHERE er.entity_kind = 'actor'
  LOOP
    v_seq := v_seq + 1;

    SELECT COALESCE(SUM(fn_effective_weight(p_world_id, e.entity_id)), 0)
      INTO v_cw
      FROM artifact_state e
      WHERE e.world_id = p_world_id AND (e.attrs->>'contained_by')::uuid = v_carrier;

    SELECT (attrs->>'max_load')::numeric,
           COALESCE(attrs->'statuses', '[]'::jsonb)
      INTO v_maxload, v_stat
      FROM actor_state
      WHERE world_id = p_world_id AND entity_id = v_carrier;

    -- carried_weight is a MEASUREMENT (§0), written eagerly so cognition/perception read a true state.
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, p_event_id, v_carrier, 'actor', 'attrs.carried_weight',
            to_jsonb(v_cw), v_tick, v_seq);

    -- set/clear encumbered. v_cw > NULL (no max_load) is NULL → the ELSE clears — an unset capacity
    -- can't be exceeded. `?` tests string membership in the statuses array.
    IF v_cw > v_maxload THEN
      IF NOT (v_stat ? 'encumbered') THEN
        v_stat := v_stat || '["encumbered"]'::jsonb;
      END IF;
    ELSE
      SELECT COALESCE(jsonb_agg(x), '[]'::jsonb)
        INTO v_stat
        FROM jsonb_array_elements(v_stat) x
        WHERE x <> '"encumbered"'::jsonb;
    END IF;

    v_seq := v_seq + 1;
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, p_event_id, v_carrier, 'actor', 'attrs.statuses',
            v_stat, v_tick, v_seq);
  END LOOP;
END $$;


--
-- Name: fn_apply_entity_created(uuid, uuid, uuid, text, text, text, jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_apply_entity_created(p_event_id uuid, p_world_id uuid, p_target_id uuid, p_kind text, p_name text, p_descriptor text, p_attrs jsonb) RETURNS uuid
    LANGUAGE plpgsql
    AS $$
DECLARE
  v_existing uuid;
  v_new      uuid;
  v_tick     bigint;
  v_seq      int;
  v_scene    uuid;
  k          text;
  v          jsonb;
BEGIN
  -- REUSE-BEFORE-CREATE (§5.4 / R3): match an existing entity before minting a new one.
  SELECT entity_id INTO v_existing
    FROM entity_registry
    WHERE world_id = p_world_id
      AND entity_kind = p_kind
      AND status = 'active'
      AND descriptor IS NOT NULL
      AND lower(btrim(descriptor)) = lower(btrim(p_descriptor))
    ORDER BY entity_id
    LIMIT 1;
  IF v_existing IS NOT NULL THEN
    RETURN v_existing;   -- the entity already exists — reuse its id, mint nothing
  END IF;

  -- TRUE INTRODUCTION: create the registry row, provenance-stamped.
  v_new   := COALESCE(p_target_id, gen_random_uuid());
  v_scene := NULLIF(p_attrs->>'location_id', '')::uuid;   -- actor/artifact carry a scene; a location has none
  INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name, descriptor,
                               current_scene_id, created_by_event, status)
  VALUES (v_new, p_world_id, p_kind, COALESCE(NULLIF(btrim(COALESCE(p_name,'')),''), p_descriptor),
          p_descriptor, v_scene, p_event_id, 'active');

  -- INITIAL POSITIONED STATE (replay-safe): one state_mutation per attr key. Tick/seq come from the
  -- committed create event (single source of truth), mirroring fn_apply_carry_change.
  SELECT in_world_tick, beat_seq INTO v_tick, v_seq FROM canon_event WHERE event_id = p_event_id;
  IF p_attrs IS NOT NULL AND jsonb_typeof(p_attrs) = 'object' THEN
    FOR k, v IN SELECT key, value FROM jsonb_each(p_attrs) LOOP
      INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                  new_value, valid_from_tick, valid_from_seq)
      VALUES (p_world_id, p_event_id, v_new, p_kind, 'attrs.' || k, v, v_tick, v_seq);
    END LOOP;
  END IF;

  RETURN v_new;
END $$;


--
-- Name: fn_area_around(jsonb, numeric); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_area_around(p_centre jsonb, p_radius numeric) RETURNS jsonb
    LANGUAGE sql IMMUTABLE
    AS $$
  SELECT jsonb_build_object('points', jsonb_agg(
           jsonb_build_object(
             'x', round(((p_centre->>'x')::numeric + p_radius * cosd(45 * g))::numeric, 3),
             'y', round(((p_centre->>'y')::numeric + p_radius * sind(45 * g))::numeric, 3))
           ORDER BY g))
  FROM generate_series(0, 7) AS g;
$$;


--
-- Name: fn_area_polygon(jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_area_polygon(p_attrs jsonb) RETURNS polygon
    LANGUAGE sql IMMUTABLE
    AS $$
  SELECT CASE
    WHEN jsonb_array_length(COALESCE(p_attrs->'area'->'points', '[]'::jsonb)) >= 3
    THEN (
      SELECT ('(' || string_agg('(' || (pt->>'x') || ',' || (pt->>'y') || ')', ',' ORDER BY ord) || ')')::polygon
      FROM jsonb_array_elements(p_attrs->'area'->'points') WITH ORDINALITY AS t(pt, ord)
    )
    ELSE NULL
  END;
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
      -- Perception rows do not encode a typed artifact classification.
      'perceived_type',             NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_artifact_id),
      'last_known_location',        fn_compendium_last_known_location(p_world_id, p_viewer_id, p_artifact_id),
      -- Holder/owner/access requires carry-state lenses that are not modeled in perception rows.
      'current_holder_owner_access',NULL,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_artifact_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_artifact_id, ARRAY['actor','location','artifact'])
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
-- Name: fn_carrying(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_carrying(p_world_id uuid, p_viewer_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  WITH RECURSIVE
  -- Current containment edge per entity: the newest applied `attrs.contained_by` mutation, by the
  -- domain ordering key (in_world_tick, beat_seq) — never recorded_at, which is transaction time
  -- (B-5, ADR-034). Putting a thing down is not a special case and needs no tombstone: apply_event's
  -- ObjectRelocated always names a destination (state_mutation.new_value is NOT NULL, so no
  -- "contained by nothing" edge is even writable), and a destination that is not the viewer simply
  -- stops rooting the chain at them. `#>>`/NULLIF are the defensive parse of a jsonb column whose
  -- type permits a JSON null nothing currently writes.
  latest AS (
    SELECT DISTINCT ON (sm.entity_id)
           sm.entity_id,
           NULLIF(sm.new_value #>> '{}', '')::uuid AS holder_id,
           sm.valid_from_tick,
           sm.event_id
    FROM state_mutation sm
    WHERE sm.world_id = p_world_id
      AND sm.attribute_path = 'attrs.contained_by'
      AND sm.status = 'applied'
    ORDER BY sm.entity_id, sm.valid_from_tick DESC, sm.valid_from_seq DESC
  ),
  -- Everything whose containment chain ROOTS AT THE VIEWER. Nesting is included because the engine
  -- already means it: fn_apply_carry_change climbs contained_by to the root carrier and charges the
  -- whole subtree to that actor's carried_weight. An overlay that showed only the top layer would
  -- contradict the encumbrance the same world computes — you would be penalised for weight the
  -- surface says you are not carrying. depth < 64 is the same contained_by cycle fail-safe
  -- fn_apply_carry_change uses (I-4).
  held(entity_id, container_id, depth, confirmed_tick, event_id) AS (
      SELECT l.entity_id, NULL::uuid, 0, l.valid_from_tick, l.event_id
      FROM latest l
      WHERE l.holder_id = p_viewer_id
    UNION ALL
      SELECT l.entity_id, h.entity_id, h.depth + 1, l.valid_from_tick, l.event_id
      FROM held h
      JOIN latest l ON l.holder_id = h.entity_id
      WHERE h.depth < 64
  ),
  items AS (
    SELECT h.entity_id,
           h.container_id,
           fn_display_name(p_world_id, p_viewer_id, h.entity_id) AS label,
           json_build_object(
             -- `id` IS the PRD's `open_full_artifact_link`. The link itself is not shipped: routes
             -- are the frontend's (D-7), and a backend that emitted "/worlds/…/artifacts/…/page"
             -- would be hardcoding someone else's URL space.
             'id',                    h.entity_id,
             'label',                 fn_display_name(p_world_id, p_viewer_id, h.entity_id),
             -- The PRD's Carry State enum is carried|worn|held|packed|stored_elsewhere|lost|unknown.
             -- The world records exactly one distinction — contained_by, i.e. in your possession or
             -- not — so `carried` is the only value it can honestly produce; worn/held/packed need a
             -- signal that does not exist, and stored_elsewhere/lost/unknown describe things that
             -- are NOT on you and so can never appear on this surface. Typed as a plain string and
             -- NOT enum-pinned on purpose: when a real signal lands the value set widens in place
             -- and costs the frontend no re-pin, provided it treats an unknown value as opaque.
             'state',                 'carried',
             -- null = directly on you. Non-null = the thing of yours it is inside. This reports the
             -- chain; it does not answer the PRD's open container-semantics question (pouch inside
             -- bag as a UI affordance), which stays open.
             'container',             CASE WHEN h.container_id IS NULL THEN NULL
                                      ELSE json_build_object(
                                        'id',    h.container_id,
                                        'label', fn_display_name(p_world_id, p_viewer_id, h.container_id))
                                      END,
             'last_confirmed_tick',   h.confirmed_tick,
             -- Viewer-held knowledge only (fn_visible_perceptions), so the preview line can never
             -- say more about the object than its carrier actually knows. Null is ordinary: you can
             -- carry a thing you have learned nothing about.
             'quick_inspect_preview', fn_compendium_latest_fact(p_world_id, p_viewer_id, h.entity_id),
             -- Artifacts AC#3: a stale carry state renders decay language and never disappears.
             -- Same horizon and same shape as every other compendium surface.
             'decay',                 fn_compendium_decay(p_world_id, h.confirmed_tick, ce.in_world_label)
           ) AS item
    FROM held h
    JOIN entity_registry er
      ON er.world_id = p_world_id AND er.entity_id = h.entity_id AND er.entity_kind = 'artifact'
    JOIN canon_event ce
      ON ce.event_id = h.event_id
  )
  SELECT json_build_object(
    'schema_version', 'carrying/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    -- Always an array, never null: "you are carrying nothing" is an answer, not a missing page.
    -- Order is deterministic and stable — what is directly on you first, then alphabetically by the
    -- label you yourself use, id as the final tiebreak.
    'carried', coalesce(
      (SELECT json_agg(i.item ORDER BY (i.container_id IS NOT NULL), i.label, i.entity_id)
       FROM items i),
      '[]'::json)
  );
$$;


--
-- Name: fn_collected_knowledge(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_collected_knowledge(p_world_id uuid, p_viewer_id uuid, p_target_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  WITH about AS (
    SELECT v.perception_id,
           v.source_event_id,
           v.content,
           v.epistemic_type,
           v.valid_tick,
           v.confidence,
           ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps
      ON ps.perception_id = v.perception_id
     AND ps.entity_id = p_target_id
    JOIN canon_event ce
      ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  -- Candidate topics: the other things these records are about — never the page's own subject, and
  -- never the reader.
  named_cosubject AS (
    SELECT a.perception_id,
           ps.entity_id,
           fn_display_name(p_world_id, p_viewer_id, ps.entity_id) AS label
    FROM about a
    JOIN perception_subject ps
      ON ps.perception_id = a.perception_id
     AND ps.entity_id <> p_target_id
     AND ps.entity_id <> p_viewer_id
    WHERE fn_display_name(p_world_id, p_viewer_id, ps.entity_id) IS NOT NULL
  ),
  -- A topic is a thing that keeps coming up: how many of this page's records each candidate shares.
  recurrence AS (
    SELECT entity_id, label, count(DISTINCT perception_id) AS n
    FROM named_cosubject
    GROUP BY entity_id, label
  ),
  -- Exactly one topic per record — its strongest, deterministically.
  filed AS (
    SELECT DISTINCT ON (nc.perception_id)
           nc.perception_id, nc.entity_id, nc.label
    FROM named_cosubject nc
    JOIN recurrence r ON r.entity_id = nc.entity_id
    ORDER BY nc.perception_id, r.n DESC, nc.label, nc.entity_id
  ),
  items AS (
    SELECT coalesce(f.entity_id, p_target_id) AS group_entity,
           f.label AS group_label,
           a.valid_tick AS sort_tick,
           a.perception_id,
           json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',            fn_compendium_decay(p_world_id, a.valid_tick, a.in_world_label),
             'source',           json_build_object(
                                   'epistemic_type', a.epistemic_type,
                                   'source_event_label', a.in_world_label
                                 )
           ) AS item
    FROM about a
    LEFT JOIN filed f ON f.perception_id = a.perception_id
  ),
  grouped AS (
    SELECT i.group_entity,
           max(i.group_label) AS group_label,   -- one label per group; NULL for the remainder
           count(*) AS n,
           max(i.sort_tick) AS latest_tick,
           coalesce(
             json_agg(i.item ORDER BY i.sort_tick, i.perception_id),
             '[]'::json
           ) AS group_items
    FROM items i
    GROUP BY i.group_entity
  )
  SELECT coalesce(
           json_agg(
             json_build_object(
               'group_key',   'subject:' || g.group_entity::text,
               'group_label', g.group_label,
               'items',       g.group_items
             )
             -- Unheaded remainder first (a heading-less block between two headed groups reads as
             -- theirs), then what recurs most, then what is most recent, then id.
             ORDER BY (g.group_label IS NOT NULL), g.n DESC, g.latest_tick DESC, g.group_entity
           ),
           '[]'::json
         )
  FROM grouped g;
$$;


--
-- Name: fn_compendium_current_synthesis(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_current_synthesis(p_world_id uuid, p_viewer_id uuid, p_target_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  WITH ranked AS (
    SELECT vp.content,
           row_number() OVER (
             ORDER BY vp.valid_tick DESC, vp.acquired_tick DESC, vp.perception_id DESC
           ) AS rn
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
    JOIN perception_subject ps
      ON ps.perception_id = vp.perception_id
     AND ps.entity_id = p_target_id
    JOIN canon_event ce
      ON ce.event_id = vp.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  )
  SELECT CASE WHEN count(*) = 0 THEN NULL
              ELSE string_agg(r.content, E'\n' ORDER BY r.rn)
         END
  FROM ranked r
  WHERE r.rn <= 3;
$$;


--
-- Name: fn_compendium_decay(uuid, bigint, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_decay(p_world_id uuid, p_valid_tick bigint, p_last_confirmed_label text) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
    'stale', (fn_world_now(p_world_id) - p_valid_tick) > fn_compendium_decay_horizon_ticks(),
    'last_confirmed_label', p_last_confirmed_label
  );
$$;


--
-- Name: fn_compendium_decay_horizon_ticks(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_decay_horizon_ticks() RETURNS bigint
    LANGUAGE sql IMMUTABLE
    AS $$
  SELECT 72::bigint;
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
-- Name: fn_compendium_last_known_location(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_last_known_location(p_world_id uuid, p_viewer_id uuid, p_artifact_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  WITH about_artifact AS (
    SELECT vp.perception_id,
           vp.valid_tick,
           vp.acquired_tick
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
    JOIN perception_subject psa
      ON psa.perception_id = vp.perception_id
     AND psa.entity_id = p_artifact_id
    JOIN canon_event ce
      ON ce.event_id = vp.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  )
  SELECT fn_perceived_name(p_world_id, p_viewer_id, er.entity_id)
  FROM about_artifact aa
  JOIN perception_subject psl
    ON psl.perception_id = aa.perception_id
   AND psl.entity_id <> p_artifact_id
  JOIN entity_registry er
    ON er.world_id = p_world_id
   AND er.entity_id = psl.entity_id
   AND er.entity_kind = 'location'
  ORDER BY aa.valid_tick DESC, aa.acquired_tick DESC, aa.perception_id DESC, er.entity_id
  LIMIT 1;
$$;


--
-- Name: fn_compendium_latest_fact(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_latest_fact(p_world_id uuid, p_viewer_id uuid, p_target_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  SELECT vp.content
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
  JOIN perception_subject ps
    ON ps.perception_id = vp.perception_id
   AND ps.entity_id = p_target_id
  JOIN canon_event ce
    ON ce.event_id = vp.source_event_id
  WHERE ce.event_type <> 'world_genesis'
  ORDER BY vp.valid_tick DESC, vp.acquired_tick DESC, vp.perception_id DESC
  LIMIT 1;
$$;


--
-- Name: fn_compendium_related_entities(uuid, uuid, uuid, text[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_related_entities(p_world_id uuid, p_viewer_id uuid, p_target_id uuid, p_related_kinds text[]) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  WITH about_target AS (
    SELECT vp.perception_id,
           vp.valid_tick
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
    JOIN perception_subject pst
      ON pst.perception_id = vp.perception_id
     AND pst.entity_id = p_target_id
    JOIN canon_event ce
      ON ce.event_id = vp.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  related AS (
    SELECT er.entity_id,
           er.entity_kind,
           at.valid_tick,
           at.perception_id
    FROM about_target at
    JOIN perception_subject ps2
      ON ps2.perception_id = at.perception_id
     AND ps2.entity_id <> p_target_id
    JOIN entity_registry er
      ON er.world_id = p_world_id
     AND er.entity_id = ps2.entity_id
    WHERE er.entity_kind = ANY (p_related_kinds)
  ),
  collapsed AS (
    SELECT r.entity_id,
           r.entity_kind,
           max(r.valid_tick) AS last_seen_tick,
           count(DISTINCT r.perception_id) AS evidence_count
    FROM related r
    GROUP BY r.entity_id, r.entity_kind
  )
  SELECT coalesce(
           json_agg(
             json_build_object(
               'id', c.entity_id,
               'kind', c.entity_kind,
               'perceived_name', fn_perceived_name(p_world_id, p_viewer_id, c.entity_id),
               'last_seen_tick', c.last_seen_tick,
               'evidence_count', c.evidence_count
             )
             ORDER BY c.last_seen_tick DESC, c.entity_id
           ),
           '[]'::json
         )
  FROM collapsed c;
$$;


--
-- Name: fn_display_name(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_display_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  SELECT COALESCE(
    fn_perceived_name(p_world_id, p_viewer_id, p_entity_id),
    (SELECT ast.attrs->>'descriptor' FROM actor_state ast
      WHERE ast.world_id = p_world_id AND ast.entity_id = p_entity_id),
    (SELECT art.attrs->>'descriptor' FROM artifact_state art
      WHERE art.world_id = p_world_id AND art.entity_id = p_entity_id),
    (SELECT er.canonical_name FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_entity_id)
  );
$$;


--
-- Name: fn_display_names_distinct(uuid, uuid, uuid[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_display_names_distinct(p_world_id uuid, p_viewer_id uuid, p_ids uuid[]) RETURNS TABLE(entity_id uuid, label text)
    LANGUAGE sql STABLE
    AS $$
  WITH base AS (
    SELECT t.id AS entity_id,
           t.ord,
           fn_display_name(p_world_id, p_viewer_id, t.id) AS base_label,
           fn_perceived_anchor(p_world_id, p_viewer_id, t.id) AS anchor
      FROM unnest(p_ids) WITH ORDINALITY AS t(id, ord)
  ),
  -- Per-label aggregate rather than a window: Postgres has no count(DISTINCT …) OVER (…).
  spread AS (
    SELECT b.base_label,
           count(*)                 AS same_label,
           -- distinct anchors in this label's group; 1 (or 0) means detail cannot separate it
           count(DISTINCT b.anchor) AS distinct_anchors
      FROM base b
     GROUP BY b.base_label
  )
  SELECT b.entity_id,
         CASE
           WHEN s.same_label > 1 AND s.distinct_anchors > 1 AND b.anchor IS NOT NULL
             THEN b.base_label || ' by ' || b.anchor
           ELSE b.base_label
         END
    FROM base b
    JOIN spread s ON s.base_label IS NOT DISTINCT FROM b.base_label
   ORDER BY b.ord;  -- callers rely on input order (the beat's own candidate order)
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
    SELECT e.which, e.scene AS loc, 0 AS up
    FROM ent e
    UNION ALL
    SELECT c.which, (ls.attrs->>'parent_location_id')::uuid, c.up + 1
    FROM climb c
    JOIN location_state ls ON ls.world_id = p_world_id AND ls.entity_id = c.loc
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
      AND c.loc IS DISTINCT FROM (SELECT loc FROM ncp)
      AND c.up < 64   -- defensive depth cap (I-4 mirror)
  ),
  frame_coord AS (
    SELECT e.which,
      CASE
        WHEN e.scene = (SELECT loc FROM ncp)
          THEN e.own_coord
        ELSE (
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
-- Name: fn_duration_class_seconds(uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_duration_class_seconds(p_world_id uuid, p_class text) RETURNS bigint
    LANGUAGE sql STABLE
    AS $$
  SELECT COALESCE(
    (SELECT seconds FROM duration_class_seconds WHERE world_id = p_world_id AND class = p_class),
    CASE p_class  -- built-in fallback (retune per-world via the table)
      WHEN 'instant' THEN 2 WHEN 'short' THEN 5 WHEN 'medium' THEN 60
      WHEN 'long' THEN 300 WHEN 'extremely_long' THEN 7200 ELSE 2 END
  );
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
-- Name: fn_effective_weight(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_effective_weight(p_world_id uuid, p_entity uuid) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
  SELECT fn_effective_weight_r(p_world_id, p_entity, 0);
$$;


--
-- Name: fn_effective_weight_r(uuid, uuid, integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_effective_weight_r(p_world_id uuid, p_entity uuid, p_depth integer) RETURNS numeric
    LANGUAGE plpgsql STABLE
    AS $$
DECLARE
  v_attrs jsonb;
  v_empty numeric;
  v_mod   numeric;
  v_sum   numeric := 0;
  v_child uuid;
BEGIN
  IF p_depth > 64 THEN
    RAISE EXCEPTION
      'containment depth cap (64) exceeded weighing % — likely a contained_by cycle (mirror I-4 guard)',
      p_entity;
  END IF;

  SELECT attrs INTO v_attrs
  FROM artifact_state
  WHERE world_id = p_world_id AND entity_id = p_entity;

  -- plain object (no container props): its own weight, defaulting to 0.
  IF v_attrs IS NULL OR NOT (v_attrs ? 'empty_weight' OR v_attrs ? 'max_room') THEN
    RETURN COALESCE((v_attrs->>'weight')::numeric, 0);
  END IF;

  -- container: (empty + Σ contents) × modifier.
  v_empty := COALESCE((v_attrs->>'empty_weight')::numeric, 0);
  v_mod   := COALESCE((v_attrs->>'weight_modifier')::numeric, 1);
  FOR v_child IN
    SELECT entity_id FROM artifact_state
    WHERE world_id = p_world_id AND (attrs->>'contained_by')::uuid = p_entity
  LOOP
    v_sum := v_sum + fn_effective_weight_r(p_world_id, v_child, p_depth + 1);
  END LOOP;

  RETURN (v_empty + v_sum) * v_mod;
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
-- Name: fn_extent_class_metres(uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_extent_class_metres(p_world_id uuid, p_class text) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
  SELECT COALESCE(
    (SELECT radius_m FROM extent_class_metres WHERE world_id = p_world_id AND class = p_class),
    CASE p_class  -- built-in fallback (retune per-world via the table) — never fails closed
      WHEN 'intimate' THEN 5 WHEN 'small' THEN 50 WHEN 'medium' THEN 200
      WHEN 'large' THEN 1000 WHEN 'vast' THEN 5000 ELSE 50 END
  );
$$;


--
-- Name: fn_fact_sheet(uuid, uuid, uuid[], boolean); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_fact_sheet(p_world_id uuid, p_viewer uuid, p_involved uuid[], p_truth_side boolean) RETURNS jsonb
    LANGUAGE sql STABLE
    AS $$
  SELECT jsonb_build_object(
    -- ALWAYS json null: beat state the orchestrator fills, never an entity fact (see header).
    'budget_remaining', NULL::jsonb,
    'targets', COALESCE(
      (SELECT jsonb_agg(s.obj ORDER BY s.ord)
       FROM (
         SELECT
           t.ord,
           jsonb_build_object(
             'id',              t.target_id,
             'kind',            er.entity_kind,
             -- THE WALL (naming-reach, §3): truth-side always sees the canonical name (the referee is
             -- licensed); perceived-side sees the viewer-relative label (known name → descriptor →
             -- canonical fallback) — never a name the viewer hasn't earned a knowledge path to.
             'name',            CASE WHEN p_truth_side THEN er.canonical_name
                                      ELSE fn_display_name(p_world_id, p_viewer, t.target_id) END,
             'distance_m',      fn_distance(p_world_id, p_viewer, t.target_id),
             'move_duration_s', fn_move_duration_actor(p_world_id, p_viewer, t.target_id),
             -- same-scene is trivially reachable; else a Portal must permit viewer_scene → target_scene.
             -- COALESCE folds a NULL scene comparison to false so an unresolved scene never yields a
             -- null `reachable` (json false, not json null).
             'reachable',       COALESCE(vs.scene = ts.scene, false)
                                OR fn_portal_permits(p_world_id, vs.scene, ts.scene),
             -- PORTAL facts (target has `connects`): the Tier-1 open/locked booleans; else json null.
             'open',            CASE WHEN art.attrs ? 'connects' THEN art.attrs->'open'   ELSE NULL::jsonb END,
             'locked',          CASE WHEN art.attrs ? 'connects' THEN art.attrs->'locked' ELSE NULL::jsonb END,
             -- OBJECT facts (target has `size`): recursive effective weight, derived volume, and whether
             -- grabbing it would encumber THIS viewer (weight > the viewer's max_load); else json null.
             'weight_kg',       CASE WHEN art.attrs ? 'size'
                                     THEN fn_effective_weight(p_world_id, t.target_id)
                                     ELSE NULL::numeric END,
             'volume',          CASE WHEN art.attrs ? 'size'
                                     THEN fn_volume((art.attrs->>'size')::int)
                                     ELSE NULL::numeric END,
             'would_encumber',  CASE WHEN art.attrs ? 'size'
                                     THEN fn_effective_weight(p_world_id, t.target_id)
                                          > (SELECT (a.attrs->>'max_load')::numeric FROM actor_state a
                                             WHERE a.world_id = p_world_id AND a.entity_id = p_viewer)
                                     ELSE NULL::boolean END,
             -- CONTAINER facts (target has `max_room`): the ids it holds (contents = artifacts whose
             -- contained_by = the container). THE WALL: truth-side always shows them; perceived-side
             -- withholds a CLOSED container's contents (json null). Non-container → json null.
             'contents',        CASE
                                  WHEN NOT (art.attrs ? 'max_room') THEN NULL::jsonb
                                  WHEN p_truth_side OR art.attrs->>'open' = 'true' THEN
                                    COALESCE(
                                      (SELECT jsonb_agg(c.entity_id::text ORDER BY c.entity_id)
                                       FROM artifact_state c
                                       WHERE c.world_id = p_world_id
                                         AND (c.attrs->>'contained_by')::uuid = t.target_id),
                                      '[]'::jsonb)
                                  ELSE NULL::jsonb
                                END
           ) AS obj
         FROM unnest(p_involved) WITH ORDINALITY AS t(target_id, ord)
         JOIN entity_registry er
           ON er.world_id = p_world_id AND er.entity_id = t.target_id
         LEFT JOIN artifact_state art
           ON art.world_id = p_world_id AND art.entity_id = t.target_id
         -- scene resolution (§3), reusing the move gate's resolver so facts never disagree with commits.
         LEFT JOIN LATERAL fn_target_position(p_world_id, t.target_id) AS ts ON true
         LEFT JOIN LATERAL fn_target_position(p_world_id, p_viewer)    AS vs ON true
         WHERE t.target_id <> p_viewer            -- one entry per involved id EXCEPT the viewer itself
       ) s),
      '[]'::jsonb)                                -- no targets (only the viewer, or empty) → empty array
  );
$$;


--
-- Name: fn_image_ref(uuid, text, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_image_ref(p_world_id uuid, p_owner_kind text, p_owner_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
           'schema_version', 'image_ref/1',
           'asset_id',       s.asset_id,
           'path',           '/worlds/' || p_world_id::text || '/images/' || s.asset_id
         )
    FROM image_slot s
   WHERE s.world_id = p_world_id AND s.owner_kind = p_owner_kind AND s.owner_id = p_owner_id
     AND s.variant IN ('default', 'neutral') AND s.asset_id IS NOT NULL
   ORDER BY CASE s.variant WHEN 'default' THEN 0 ELSE 1 END
   LIMIT 1;
$$;


--
-- Name: fn_instantiate_drowned_lantern(uuid, jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_instantiate_drowned_lantern(p_world_id uuid, p_pin jsonb DEFAULT NULL::jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
  v_kade uuid := COALESCE((p_pin->>'kade')::uuid, gen_random_uuid());
  v_mara uuid := COALESCE((p_pin->>'mara')::uuid, gen_random_uuid());
  v_jonas uuid := COALESCE((p_pin->>'jonas')::uuid, gen_random_uuid());
  v_hooded_woman uuid := COALESCE((p_pin->>'hooded_woman')::uuid, gen_random_uuid());
  v_hooded_companion uuid := COALESCE((p_pin->>'hooded_companion')::uuid, gen_random_uuid());
  v_harbor_quarter uuid := COALESCE((p_pin->>'harbor_quarter')::uuid, gen_random_uuid());
  v_drowned_lantern uuid := COALESCE((p_pin->>'drowned_lantern')::uuid, gen_random_uuid());
  v_dock_street uuid := COALESCE((p_pin->>'dock_street')::uuid, gen_random_uuid());
  v_alley uuid := COALESCE((p_pin->>'alley')::uuid, gen_random_uuid());
  v_cellar uuid := COALESCE((p_pin->>'cellar')::uuid, gen_random_uuid());
  v_harbormaster_office uuid := COALESCE((p_pin->>'harbormaster_office')::uuid, gen_random_uuid());
  v_sealed_note uuid := COALESCE((p_pin->>'sealed_note')::uuid, gen_random_uuid());
  v_front_door uuid := COALESCE((p_pin->>'front_door')::uuid, gen_random_uuid());
  v_back_door uuid := COALESCE((p_pin->>'back_door')::uuid, gen_random_uuid());
  v_cellar_hatch uuid := COALESCE((p_pin->>'cellar_hatch')::uuid, gen_random_uuid());
  v_office_door uuid := COALESCE((p_pin->>'office_door')::uuid, gen_random_uuid());
  v_cellar_key uuid := COALESCE((p_pin->>'cellar_key')::uuid, gen_random_uuid());
  v_bar_fixture uuid := COALESCE((p_pin->>'bar_fixture')::uuid, gen_random_uuid());
  v_ballast_crate uuid := COALESCE((p_pin->>'ballast_crate')::uuid, gen_random_uuid());
  v_ballast_stone uuid := COALESCE((p_pin->>'ballast_stone')::uuid, gen_random_uuid());
  v_event_m_e1 uuid := COALESCE((p_pin->>'event_m_e1')::uuid, gen_random_uuid());
  v_event_m_e2 uuid := COALESCE((p_pin->>'event_m_e2')::uuid, gen_random_uuid());
  v_event_m_e3 uuid := COALESCE((p_pin->>'event_m_e3')::uuid, gen_random_uuid());
  v_event_m_e4_private uuid := COALESCE((p_pin->>'event_m_e4_private')::uuid, gen_random_uuid());
  v_event_j_e1 uuid := COALESCE((p_pin->>'event_j_e1')::uuid, gen_random_uuid());
  v_event_j_e2 uuid := COALESCE((p_pin->>'event_j_e2')::uuid, gen_random_uuid());
  v_event_j_e3_private uuid := COALESCE((p_pin->>'event_j_e3_private')::uuid, gen_random_uuid());
  v_event_h_e1_private uuid := COALESCE((p_pin->>'event_h_e1_private')::uuid, gen_random_uuid());
  v_event_scene_genesis uuid := COALESCE((p_pin->>'event_scene_genesis')::uuid, gen_random_uuid());
  v_event_kade_arrival uuid := COALESCE((p_pin->>'event_kade_arrival')::uuid, gen_random_uuid());
  v_event_world_genesis uuid := COALESCE((p_pin->>'event_world_genesis')::uuid, gen_random_uuid());
  v_perception_mara_secret uuid := COALESCE((p_pin->>'perception_mara_secret')::uuid, gen_random_uuid());
  v_perception_jonas_secret uuid := COALESCE((p_pin->>'perception_jonas_secret')::uuid, gen_random_uuid());
  v_perception_hooded_contract uuid := COALESCE((p_pin->>'perception_hooded_contract')::uuid, gen_random_uuid());
  v_perception_kade_arrival uuid := COALESCE((p_pin->>'perception_kade_arrival')::uuid, gen_random_uuid());
  v_name_perception_kade_knows_mara uuid := COALESCE((p_pin->>'name_perception_kade_knows_mara')::uuid, gen_random_uuid());
  v_name_perception_mara_knows_jonas uuid := COALESCE((p_pin->>'name_perception_mara_knows_jonas')::uuid, gen_random_uuid());
  v_name_perception_jonas_knows_mara uuid := COALESCE((p_pin->>'name_perception_jonas_knows_mara')::uuid, gen_random_uuid());
  v_name_perception_mara_knows_kade uuid := COALESCE((p_pin->>'name_perception_mara_knows_kade')::uuid, gen_random_uuid());
BEGIN
-- Own idempotence guard: refuse a double-load. `make reset` is the clean re-run path.
-- Guard on the target world itself — any existing registry row means this template already ran there.
IF EXISTS (SELECT 1 FROM entity_registry WHERE world_id = p_world_id) THEN
  RAISE EXCEPTION 'fn_instantiate_drowned_lantern already applied for world % — run `make reset` for a clean load', p_world_id;
END IF;

-- Physics defaults for the play world (contracts §2: exactly walk 1.4 + encumbered -100 on walk).
PERFORM seed_world_defaults(p_world_id);

-- ── Registry: 4 actors + 5 locations (REAL names) + 8 artifacts ───────────────────────
-- All-new fixed uuids (entity_registry PK is global). Kade is 'Kade' now — a real name, not the
-- fixture world's 'Player'. The tavern is 'The Drowned Lantern', not 'Tavern'.
--
-- Task 7 (Station F) adds the SPATIAL layer (§3 nested coordinates): a parent location 'Harbor Quarter
-- of Vael' (…-d0) over the four rooms, a fixed room feature 'the bar' (…-f1, the anchor Kade walks to),
-- and a Container instance 'ballast crate' (…-f2) holding a 'ballast stone' (…-f3) so the §4 ObjectRelocated
-- physics has a heavy thing to bite on in play. Coordinates are a SANCTIONED hand-authored test artifact
-- (spec §3 — the hand-placed seed world is a test artifact; production mints coordinates via Task 6).
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 (v_kade,p_world_id,'actor',   'Kade'),
 (v_mara,p_world_id,'actor',   'Mara'),
 (v_jonas,p_world_id,'actor',   'Jonas'),
 (v_hooded_woman,p_world_id,'actor',   'Hooded Woman'),
 -- A SECOND hooded figure at the same table. Kade has no name for either, so fn_display_name renders
 -- both as the identical descriptor 'a hooded figure' — which is the point: "ask the hooded figure
 -- about the note" now names two people equally well, and decompose must refuse to guess (UNRESOLVED)
 -- instead of silently picking one. Until this, every candidate in the room resolved uniquely and the
 -- UNRESOLVED path could not be reached in play at all. Id is …aa, not the next free …a5: pgTAP's
 -- 104_world_slice_test mints …a5 itself, and entity_registry's PK is global, so the seed and the
 -- tests share one id space. Anything added here needs a suffix no test already claims.
 (v_hooded_companion,p_world_id,'actor',   'Hooded Companion'),
 (v_harbor_quarter,p_world_id,'location','Harbor Quarter of Vael'),
 (v_drowned_lantern,p_world_id,'location','The Drowned Lantern'),
 (v_dock_street,p_world_id,'location','Dock Street'),
 (v_alley,p_world_id,'location','Alley'),
 (v_cellar,p_world_id,'location','Cellar'),
 -- SPEC-030 (founder-named, 2026-08-08): the Harbormaster's Office, off Dock Street and far enough up
 -- the quarter that walking there cannot fit in one beat — the first destination in this world that
 -- starts a JOURNEY rather than an instant arrival. See the geometry note in the spatial block.
 (v_harbormaster_office,p_world_id,'location','Harbormaster''s Office'),
 (v_sealed_note,p_world_id,'artifact','Sealed Note (gray wax)'),
 (v_front_door,p_world_id,'artifact','Front Door'),
 (v_back_door,p_world_id,'artifact','Back Door'),
 (v_cellar_hatch,p_world_id,'artifact','Cellar Hatch'),
 (v_office_door,p_world_id,'artifact','Office Door'),
 (v_cellar_key,p_world_id,'artifact','Cellar Key'),
 (v_bar_fixture,p_world_id,'artifact','the bar'),
 (v_ballast_crate,p_world_id,'artifact','Ballast Crate'),
 (v_ballast_stone,p_world_id,'artifact','Ballast Stone');

-- ── Backstory canon events (ticks 30–37) + one scene-genesis event (tick 40) ──────────
-- event_type='AttributeChanged' (backstory grounds who they are); origin='fast_path'. M-E4 / J-E3 /
-- H-E1 are PRIVATE — each grounds exactly one NPC's private perception below. The scene-genesis event
-- (f9) is public and carries the room state AND places the three residents (Mara behind the bar,
-- Jonas by the bar, the hooded woman at the corner table) via absolute location writes.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 (v_event_m_e1,p_world_id,'AttributeChanged',
  'M-E1: grew up behind this bar; her father taught her a keeper who reacts has already lost',30,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_m_e2,p_world_id,'AttributeChanged',
  'M-E2: the harbormaster''s predecessor shook the tavern for protection money; the watch shrugged; her father died that winter',31,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_m_e3,p_world_id,'AttributeChanged',
  'M-E3: a dock brawl left Jonas half-dead outside her door; she stitched him up and gave him work',32,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_m_e4_private,p_world_id,'AttributeChanged',
  'M-E4 (private): she hid Reyna''s family in the cellar nine days; Reyna''s teenage brother ran the messages that got them out',33,0,
  'Backstory','accepted',now(),'private','fast_path'),
 (v_event_j_e1,p_world_id,'AttributeChanged',
  'J-E1: beaten near to death over a fixed fight and left in the alley; Mara took him in',34,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_j_e2,p_world_id,'AttributeChanged',
  'J-E2: a prizefighter until he killed a man in the ring with one unlucky blow; never fought clean for money again',35,0,
  'Backstory','accepted',now(),'public','fast_path'),
 (v_event_j_e3_private,p_world_id,'AttributeChanged',
  'J-E3 (private): twice he watched Mara go pale at a harbor face and learned to stand closer instead of asking',36,0,
  'Backstory','accepted',now(),'private','fast_path'),
 (v_event_h_e1_private,p_world_id,'AttributeChanged',
  'H-E1 (private): took the paymaster''s contract in a counting-house above the silk quay, three days ago',37,0,
  'Backstory','accepted',now(),'private','fast_path'),
 (v_event_scene_genesis,p_world_id,'AttributeChanged',
  'the Drowned Lantern is set: Mara behind the bar, Jonas by it, a hooded woman at the corner table; tension, the doors, the hatch, the note, the key',40,0,
  'Scene','accepted',now(),'public','fast_path');

-- Participants (brief: the NPC + any named co-subject). subject ≠ about-ness (perception_subject
-- carries the precise about-ness, ADR-035) — these are the event''s people. The scene-genesis event
-- names the room (setting) and the three residents it places.
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 (v_event_m_e1,v_mara,'actor','subject'),
 (v_event_m_e2,v_mara,'actor','subject'),
 (v_event_m_e3,v_mara,'actor','subject'),
 (v_event_m_e4_private,v_mara,'actor','subject'),
 (v_event_m_e4_private,v_kade,'actor','co_subject'),
 (v_event_j_e1,v_jonas,'actor','subject'),
 (v_event_j_e1,v_mara,'actor','co_subject'),
 (v_event_j_e2,v_jonas,'actor','subject'),
 (v_event_j_e3_private,v_jonas,'actor','subject'),
 (v_event_j_e3_private,v_mara,'actor','co_subject'),
 (v_event_h_e1_private,v_hooded_woman,'actor','subject'),
 (v_event_scene_genesis,v_drowned_lantern,'location','setting'),
 (v_event_scene_genesis,v_mara,'actor','subject'),
 (v_event_scene_genesis,v_jonas,'actor','subject'),
 (v_event_scene_genesis,v_hooded_woman,'actor','subject');

-- ── personality_core — WHO THEY ARE IN THE ROOM. No secret ever lives here. ───────────
-- traits jsonb: real traits are objects {value, manner}; schema_version + speech_manner are
-- strings. Kade gets NO core (premise, not a mind). Malleability per FINAL (Mara 0.25 / Jonas 0.45 /
-- hooded 0.6).
INSERT INTO personality_core (world_id, actor_id, traits, malleability) VALUES
 (p_world_id,v_mara,
  '{"schema_version":"traits/1",
    "guarded":{"value":0.8,"manner":"answers questions with questions; volunteers nothing"},
    "dry_witted":{"value":0.7,"manner":"deflects with humor before she deflects with silence"},
    "loyal_to_jonas":{"value":0.9,"manner":"treats Jonas as family; will not see him harmed"},
    "distrusts_authority":{"value":0.85,"manner":"the harbormaster''s men drink free and learn nothing"},
    "steady_under_pressure":{"value":0.8,"manner":"the last in the room to raise her voice"},
    "speech_manner":"short sentences; harbor slang; calls strangers sailor regardless of trade; never says a name she was not given"}'::jsonb,
  0.25),
 (p_world_id,v_jonas,
  '{"schema_version":"traits/1",
    "protective_of_mara":{"value":0.9,"manner":"reads every stranger as a threat to her first, himself second"},
    "slow_to_speak":{"value":0.7,"manner":"acts before he explains; three words where others use ten"},
    "brawler_not_killer":{"value":0.8,"manner":"ends fights; does not start them; hates blades"},
    "debt_of_gratitude":{"value":0.85,"manner":"the tavern is the only place that ever took him back"},
    "speech_manner":"monosyllables; states facts not opinions; uses names only for Mara"}'::jsonb,
  0.45),
 (p_world_id,v_hooded_woman,
  '{"schema_version":"traits/1",
    "watchful":{"value":0.8,"manner":"tracks the door and every hand near a purse"},
    "unhurried":{"value":0.7,"manner":"never the first to move, never the last to leave"},
    "clean_coin":{"value":0.6,"manner":"pays in coin too clean for this district"},
    "speech_manner":"says little; asks less; watches much"}'::jsonb,
  0.6);

-- ── trait_provenance — every trait traces to a backstory event (D-11 for character) ───
INSERT INTO trait_provenance (world_id, actor_id, trait_key, event_id) VALUES
 -- Mara
 (p_world_id,v_mara,'dry_witted',           v_event_m_e1),
 (p_world_id,v_mara,'steady_under_pressure',v_event_m_e1),
 (p_world_id,v_mara,'guarded',              v_event_m_e2),
 (p_world_id,v_mara,'distrusts_authority',  v_event_m_e2),
 (p_world_id,v_mara,'loyal_to_jonas',       v_event_m_e3),
 -- Jonas
 (p_world_id,v_jonas,'protective_of_mara',   v_event_j_e1),
 (p_world_id,v_jonas,'debt_of_gratitude',    v_event_j_e1),
 (p_world_id,v_jonas,'slow_to_speak',        v_event_j_e2),
 (p_world_id,v_jonas,'brawler_not_killer',   v_event_j_e2),
 -- Hooded woman (one event grounds her thin core)
 (p_world_id,v_hooded_woman,'watchful',   v_event_h_e1_private),
 (p_world_id,v_hooded_woman,'unhurried',  v_event_h_e1_private),
 (p_world_id,v_hooded_woman,'clean_coin', v_event_h_e1_private);

-- ── Private knowledge — perception_record WITH subject links (the whole point) ────────
-- Only the holder holds a perception of each private event → private to the lookups. Fixed
-- perception_ids so the subject links are unambiguous. source_event_id is NOT NULL (grounded).
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 -- Mara''s secret: recognition + the life-debt + how she knows him ("Reyna''s brother").
 (v_perception_mara_secret,p_world_id,
  v_mara,v_event_m_e4_private,
  'Kade is Reyna''s brother — the boy who ran messages while I hid Reyna''s family in the cellar nine days, five winters back. I owe that family a life-debt I have never said aloud. To him, and to this room, I am a stranger; if the wrong people learn I know him, the debt gets us both killed.',
  'direct',33,33),
 -- Jonas knows OF a secret without knowing IT: a debt she never explains. (No "Reyna", no "ledger".)
 (v_perception_jonas_secret,p_world_id,
  v_jonas,v_event_j_e3_private,
  'Mara keeps a knife under the till and a debt she never explains. Twice in four years I have watched her go pale at a face off the harbor. I have learned to stand closer and not to ask. I do not know what it is.',
  'inference',36,36),
 -- The hooded woman''s contract: a description and a purse, and one word of doubt — "Yet." (founder-trimmed;
 -- NO characterization of the note''s contents.)
 (v_perception_hooded_contract,p_world_id,
  v_hooded_woman,v_event_h_e1_private,
  'The paymaster''s coin bought a description: a courier, young, dark-haired, moves like a dock rat — and a purse for whoever confirms him. The one by the door could be him. I am not sure. Yet.',
  'told',37,37);

-- about-ness (RULINGS §6): Mara''s secret → Kade AND Mara; Jonas → Mara; hooded → Kade.
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 (v_perception_mara_secret,v_kade,p_world_id),
 (v_perception_mara_secret,v_mara,p_world_id),
 (v_perception_jonas_secret,v_mara,p_world_id),
 (v_perception_hooded_contract,v_kade,p_world_id);

-- ── Scene state via state_mutation (event f9) — projects through sm_project; replay-safe ──
-- Single-key absolute sets under attrs (ABSOLUTE-STATE-SETS, was 0A Rider B). Tier-1 keys: open, locked, connects, size,
-- weight, tension (see core/api/tier1.go). carry is the single Tier-1 key `contained_by` (§4 eager
-- encumbrance requires carry to be engine-readable state; the former Tier-2 carried_by/held_by are
-- unified into it — contents of X = entities whose contained_by = X, actors are root carriers). connects is the
-- Portal''s [room, room] pair. The cellar hatch is closed and LOCKED — the first Tier-1 lock in play.
-- The three residents are PLACED here (absolute attrs.location_id → the Drowned Lantern); Kade arrives
-- separately below. Each (entity, attribute_path) is written exactly once → replay-order-independent.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- residents in the room (Mara behind the bar, Jonas by it, the hooded woman at the corner table)
 (p_world_id,v_event_scene_genesis,v_mara,'actor',   'attrs.location_id', to_jsonb(v_drowned_lantern::text), 40,0),
 (p_world_id,v_event_scene_genesis,v_jonas,'actor',   'attrs.location_id', to_jsonb(v_drowned_lantern::text), 40,1),
 (p_world_id,v_event_scene_genesis,v_hooded_woman,'actor',   'attrs.location_id', to_jsonb(v_drowned_lantern::text), 40,2),
 -- the room reads calm; two of the four people in it are pretending → tension 'tense'
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.tension',     to_jsonb('tense'::text),                                                                            40,3),
 -- art_note: sealed, near-weightless, carried by Kade. Contents deliberately UNAUTHORED (Tier-2 flavor only).
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.size',                 to_jsonb(1),                                                                                    40,4),
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.weight',               to_jsonb(0),                                                                                    40,5),
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.sealed_with_gray_wax', to_jsonb(true),                                                                                 40,6),
 (p_world_id,v_event_scene_genesis,v_sealed_note,'artifact','attrs.contained_by',         to_jsonb(v_kade::text),                                          40,7),
 -- front door: OPEN, unlocked, tavern↔dock street
 (p_world_id,v_event_scene_genesis,v_front_door,'artifact','attrs.open',                 to_jsonb(true),                                                                                 40,8),
 (p_world_id,v_event_scene_genesis,v_front_door,'artifact','attrs.locked',               to_jsonb(false),                                                                                40,9),
 (p_world_id,v_event_scene_genesis,v_front_door,'artifact','attrs.connects',             jsonb_build_array(v_drowned_lantern,v_dock_street), 40,10),
 -- back door: closed, unlocked, tavern↔alley
 (p_world_id,v_event_scene_genesis,v_back_door,'artifact','attrs.open',                 to_jsonb(false),                                                                                40,11),
 (p_world_id,v_event_scene_genesis,v_back_door,'artifact','attrs.locked',               to_jsonb(false),                                                                                40,12),
 (p_world_id,v_event_scene_genesis,v_back_door,'artifact','attrs.connects',             jsonb_build_array(v_drowned_lantern,v_alley), 40,13),
 -- cellar hatch: closed and LOCKED (Tier-1), tavern↔cellar. Mara holds the key; the cellar is where M-E4 happened.
 (p_world_id,v_event_scene_genesis,v_cellar_hatch,'artifact','attrs.open',                 to_jsonb(false),                                                                                40,14),
 (p_world_id,v_event_scene_genesis,v_cellar_hatch,'artifact','attrs.locked',               to_jsonb(true),                                                                                 40,15),
 (p_world_id,v_event_scene_genesis,v_cellar_hatch,'artifact','attrs.connects',             jsonb_build_array(v_drowned_lantern,v_cellar), 40,16),
 -- the cellar key, held by Mara
 (p_world_id,v_event_scene_genesis,v_cellar_key,'artifact','attrs.size',                 to_jsonb(1),                                                                                    40,17),
 (p_world_id,v_event_scene_genesis,v_cellar_key,'artifact','attrs.contained_by',         to_jsonb(v_mara::text),                                          40,18),
 -- Tier-2 scene DESCRIPTION per location (Defect B): the narrate PLACE line renders it, so the room's
 -- fixed character is DATA the narrator draws on, never something it invents. The Drowned Lantern text
 -- is verbatim from FINAL-drowned-lantern-souls.md's scene section; the three stubs are brief, honest
 -- one-liners so movement has somewhere with a described face to go.
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.description', to_jsonb('Low beams, salt-rot, one hearth, a bar with a hatch, a back door to the alley.'::text), 40,19),
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.description', to_jsonb('A rain-slick harbor road; gulls, tar, and black water past the pilings.'::text),        40,20),
 (p_world_id,v_event_scene_genesis,v_alley,'location','attrs.description', to_jsonb('A narrow dead-end behind the tavern; stacked crates and standing water.'::text),         40,21),
 (p_world_id,v_event_scene_genesis,v_cellar,'location','attrs.description', to_jsonb('A cold stone undercroft beneath the tavern; barrels, damp, one shuttered lantern.'::text),   40,22);

-- ── §3 SPATIAL LAYER (Station F Task 7) — the scene gets space, under the same scene-genesis event f9 ──
-- Nested coordinates (FINAL-action-contracts.md §3): every location has a coordinate WITHIN its parent
-- (attrs.coordinates) + a parent edge (attrs.parent_location_id, Tier-1 string); a parent carries an
-- attrs.area outlining its children (an ordered ring of ≥3 points, founder ruling R12 — no {w,h} box).
-- Things inside a scene (actors + fixed features) carry a coordinate in that scene's LOCAL frame.
-- fn_distance measures any pair at their nearest common parent's frame; fn_place_at measures which
-- child's area contains a point.
-- Coordinates are a SANCTIONED hand-authored test artifact (§3); production mints them (Task 6). Each
-- (entity, attribute_path) is written EXACTLY ONCE → replay-order-independent (ABSOLUTE-STATE-SETS, was 0A Rider B; D-1). Tier-1 keys
-- only for engine-read attrs (coordinates, parent_location_id, max_room, empty_weight, weight, size,
-- contained_by, area — fn_place_at reads it, so it is engine-read, not descriptive). seq 26+ continues
-- f9's single monotonic seq space.
--
-- Harbor Quarter frame (meters): tavern {200,200}; dock street {207,200} → 7 m ⇒ CEIL(7/1.4)=5 s, a SHORT
-- STEP out the front door onto the harbor road (Task 11 seed tune, RULINGS-2026-07-30 §1: the playable
-- moves must fit the beat budget — 5 s fits the tense 30 s budget so "step out the front" plays; the
-- earlier {280,200} put it 80 m ⇒ 58 s away, an over-budget dead end. Dock Street stays a DISTINCT
-- location behind the front-door portal — just a short step, not merged into the tavern); alley {200,240}
-- → 40 m (out the back); cellar {205,205} → beneath the tavern (portal-locked anyway). The quarter's
-- 2000×2000 area (a four-corner outline) bounds them.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- Harbor Quarter of Vael: the root parent (no parent edge), its own origin + the area outline that bounds the rooms.
 (p_world_id,v_event_scene_genesis,v_harbor_quarter,'location','attrs.coordinates',        '{"x":0,"y":0}'::jsonb,       40,26),
 (p_world_id,v_event_scene_genesis,v_harbor_quarter,'location','attrs.area',               '{"points":[{"x":0,"y":0},{"x":2000,"y":0},{"x":2000,"y":2000},{"x":0,"y":2000}]}'::jsonb, 40,27),
 -- the four rooms: each a child of Harbor Quarter with a coordinate in the quarter frame.
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,28),
 (p_world_id,v_event_scene_genesis,v_drowned_lantern,'location','attrs.coordinates',        '{"x":200,"y":200}'::jsonb,   40,29),
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,30),
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.coordinates',        '{"x":207,"y":200}'::jsonb,   40,31),
 (p_world_id,v_event_scene_genesis,v_alley,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,32),
 (p_world_id,v_event_scene_genesis,v_alley,'location','attrs.coordinates',        '{"x":200,"y":240}'::jsonb,   40,33),
 (p_world_id,v_event_scene_genesis,v_cellar,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,34),
 (p_world_id,v_event_scene_genesis,v_cellar,'location','attrs.coordinates',        '{"x":205,"y":205}'::jsonb,   40,35),
 -- Tavern local frame (meters): the three residents where the scene-genesis places them, and the bar
 -- feature along the back wall. Kade's own coordinate rides his arrival event (fa) below.
 --   Mara behind the bar {6,10}; Jonas by it {5,8}; the hooded woman at the corner table {1,1}; the bar {6,9}.
 (p_world_id,v_event_scene_genesis,v_mara,'actor',   'attrs.coordinates',        '{"x":6,"y":10}'::jsonb,      40,36),
 (p_world_id,v_event_scene_genesis,v_jonas,'actor',   'attrs.coordinates',        '{"x":5,"y":8}'::jsonb,       40,37),
 (p_world_id,v_event_scene_genesis,v_hooded_woman,'actor',   'attrs.coordinates',        '{"x":1,"y":1}'::jsonb,       40,38),
 -- the bar: a fixed room feature (FINAL "contains: the bar…"). location_id = the tavern (its scene) so
 -- fn_distance resolves it to the tavern frame; coordinates {6,9} along the back wall — the anchor Kade
 -- walks to (Task 8). size-2, weightless fixture (never relocated); Tier-2 descriptor for the narrator.
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.location_id',        to_jsonb(v_drowned_lantern::text), 40,39),
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.coordinates',        '{"x":6,"y":9}'::jsonb,       40,40),
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.size',              to_jsonb(2),                  40,41),
 (p_world_id,v_event_scene_genesis,v_bar_fixture,'artifact','attrs.descriptor',        to_jsonb('the bar'::text),    40,42),
 -- §4 ObjectRelocated physics has something to grab: a Container instance (ballast crate) + a heavy
 -- ballast stone inside it. crate = (empty_weight 8 + effective_weight(stone 92)) × 1 = 100 kg; Kade's
 -- max_load is 80, so "grab the crate → encumbered" is REACHABLE (the eager rule flips it on that commit).
 -- The crate RESTS on the tavern floor: like the bar (f1) above, that is attrs.location_id = the tavern
 -- (a location is not a carrier -- `contained_by` is the carry-chain key for "inside a container / held
 -- by an actor", which this crate is not, yet). fn_distance's artifact-scene COALESCE has no other
 -- source of scene (current_scene_id is never written), so an omitted location_id here would silently
 -- resolve to NULL and fn_distance(Kade, crate) would silently read 0 instead of the true ~8.94 m.
 -- The crate is a mundane container (weight_modifier absent → 1), by the hatch {2,9}; the stone lives
 -- inside the crate (attrs.contained_by = the crate). size-2 stone (vol 4) fits max_room 16.
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.max_room',          to_jsonb(16),                 40,43),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.empty_weight',      to_jsonb(8),                  40,44),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.size',              to_jsonb(4),                  40,45),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.location_id',       to_jsonb(v_drowned_lantern::text), 40,46),
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.coordinates',        '{"x":2,"y":9}'::jsonb,       40,47),
 (p_world_id,v_event_scene_genesis,v_ballast_stone,'artifact','attrs.weight',            to_jsonb(92),                 40,48),
 (p_world_id,v_event_scene_genesis,v_ballast_stone,'artifact','attrs.size',              to_jsonb(2),                  40,49),
 (p_world_id,v_event_scene_genesis,v_ballast_stone,'artifact','attrs.contained_by',      to_jsonb(v_ballast_crate::text), 40,50);

-- ── Kade's arrival (tick 50) — he steps into the room the scene is set in ─────────────
-- Replay-safe & append-only: one accepted ActorMoved with an ABSOLUTE attrs.location_id set (the
-- sm_project trigger projects it; replay_0a rebuilds it). Tick 50 is this world's max; the live
-- handler mints the next beat after it.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 (v_event_kade_arrival,p_world_id,'ActorMoved',
  'Kade steps into the Drowned Lantern.',50,0,
  'Arrival','accepted',now(),'public','fast_path');
-- Participant: the mover (instigator).
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 (v_event_kade_arrival,v_kade,'actor','instigator');
-- Absolute location set → the Drowned Lantern. The projection trigger places Kade in the room.
-- §3/§4 (Task 7): Kade also arrives WITH a position and a carrying capacity. coordinates {6,1} put him
-- just inside the front door — 8 m from the bar {6,9} ⇒ fn_distance(Kade,bar)=8, CEIL(8/1.4)=6 s (fits
-- tense's 30 s beat: the Task-8 "walk to the bar"). max_load 80 is his static capacity: the ballast crate
-- weighs 100 kg, so grabbing it exceeds max_load → the eager encumbrance rule (§4) can fire in play.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.location_id',
  to_jsonb(v_drowned_lantern::text),50,0),
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.max_load',    to_jsonb(80),           50,2),
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.coordinates', '{"x":6,"y":1}'::jsonb, 50,3);
-- Kade's own honest, minimal perception of stepping in. NOT an authored roster of who is present (that
-- would fake fan-out he never received) — just the move itself, subject-linked to the mover + the room.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 (v_perception_kade_arrival,p_world_id,
  v_kade,v_event_kade_arrival,
  'I stepped into the Drowned Lantern.','direct',50,50);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 (v_perception_kade_arrival,v_kade,p_world_id),
 (v_perception_kade_arrival,v_drowned_lantern,p_world_id);

-- ── §3 naming reach: per-viewer NAME KNOWLEDGE + DESCRIPTOR fallbacks (Defect C) ──────────
-- The live founder-gate leak: canonical names reached the character-mind seats past knowledge paths
-- (the narration named "Jonas" to Kade, who knows him only as "the muscle"; Jonas's wind-up named
-- "Kade"). fn_display_name closes it — known-name (a viewer's own name-knowledge) else descriptor else
-- canonical. Name knowledge is stored as chunk-4's identity substrate: a world_genesis-sourced
-- perception, subject-linked to the named entity, HELD BY the viewer who knows the name (per-viewer, so
-- Kade knowing "Mara" never grants Jonas or the hooded woman that name). Held by ONE viewer ⇒ private to
-- that viewer's calls (fn_visible_perceptions is holder-keyed) — the wall holds by construction.
--
-- Who knows whose name (FINAL-drowned-lantern-souls.md): Kade knows Mara (five winters back). Mara and
-- Jonas know each other as regulars would. Mara knows Kade ONLY privately — as "Reyna's brother", the
-- name he had then, NOT the name he carries now (her secret cluster); NO public name record for Kade
-- exists, so nobody in the room publicly knows his name. The hooded woman knows no one's name (she has
-- a description of a courier, not a name), and no one knows hers — she stays "a hooded figure".
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin) VALUES
 (v_event_world_genesis,p_world_id,'world_genesis',
  'the harbor-quarter figures known to each other by name (per-viewer identity substrate)',25,0,
  'Genesis','accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 (v_event_world_genesis,v_kade,'actor','named'),
 (v_event_world_genesis,v_mara,'actor','named'),
 (v_event_world_genesis,v_jonas,'actor','named'),
 (v_event_world_genesis,v_hooded_woman,'actor','named');

-- Name perceptions (content = the name the viewer knows; source = genesis; subject = the named entity).
-- Fixed perception_ids (prefix 2a4e = "name"). Held per-viewer ⇒ each is that viewer's knowledge alone.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 -- Kade knows Mara by name (five years ago).
 (v_name_perception_kade_knows_mara,p_world_id,
  v_kade,v_event_world_genesis,'Mara','told',25,25),
 -- Mara knows Jonas by name (regulars).
 (v_name_perception_mara_knows_jonas,p_world_id,
  v_mara,v_event_world_genesis,'Jonas','told',25,25),
 -- Jonas knows Mara by name (regulars).
 (v_name_perception_jonas_knows_mara,p_world_id,
  v_jonas,v_event_world_genesis,'Mara','told',25,25),
 -- Mara PRIVATELY knows Kade — as "Reyna's brother", the name he had then, not the one he carries now.
 -- Held by Mara ALONE ⇒ only her own calls resolve it; the wall holds (part of her secret cluster).
 (v_name_perception_mara_knows_kade,p_world_id,
  v_mara,v_event_world_genesis,'Reyna''s brother','told',25,25);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 (v_name_perception_kade_knows_mara,v_mara,p_world_id),
 (v_name_perception_mara_knows_jonas,v_jonas,p_world_id),
 (v_name_perception_jonas_knows_mara,v_mara,p_world_id),
 (v_name_perception_mara_knows_kade,v_kade,p_world_id);

-- DESCRIPTOR fallbacks (Tier-2 attrs.descriptor) — what a viewer with no name-knowledge sees. Via
-- state_mutation (sm_project → actor_state), replay-safe (each (entity,path) written once). The three
-- residents under the scene-genesis event f9 (tick 40); Kade under his arrival fa (tick 50).
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 (p_world_id,v_event_scene_genesis,v_mara,'actor','attrs.descriptor', to_jsonb('the keeper'::text),                    40,23),
 (p_world_id,v_event_scene_genesis,v_jonas,'actor','attrs.descriptor', to_jsonb('the muscle by the bar'::text),         40,24),
 (p_world_id,v_event_scene_genesis,v_hooded_woman,'actor','attrs.descriptor', to_jsonb('a hooded figure'::text),               40,25),
 (p_world_id,v_event_kade_arrival,v_kade,'actor','attrs.descriptor', to_jsonb('a young stranger, dark-haired'::text), 50,1);

-- ── THE WAY OUT OF TOWN (SPEC-030, founder-named 2026-08-08) — the first JOURNEY in this world ──
-- Everything in the quarter used to sit inside one beat: the tavern's tension is 'tense' → a 30 s
-- budget, and its farthest neighbour (the alley) is 40 m ⇒ 29 s. Every destination fit, so no move
-- could ever go over budget and the Journey shipped in #32 was unreachable by any client.
--
-- Two things were needed, and neither is engine work:
--
--  1. A DESTINATION FAR ENOUGH. The Harbormaster's Office sits off Dock Street at {627,200} — 420 m
--     from the road at {207,200} ⇒ CEIL(420/1.4) = 300 s of walking (the same 1.4 m/s the rest of
--     this seed is tuned against). That is five times the origin's budget, so the walk cannot be
--     swallowed by one beat: it becomes a journey with legs the world can interrupt.
--  2. A FINITE BUDGET TO EXCEED. Dock Street carried no tension at all, and a missing tension reads
--     as 'none' ⇒ an INFINITE budget (tensionBudgetSeconds), which means no move from the road could
--     ever be over budget however far it went. It is now 'normal' ⇒ 60 s: an open harbour road is
--     not tense, but it is not timeless either.
--
-- So the founder's worked example plays: step out the front door onto Dock Street (5 s, instant),
-- then walk for the office (300 s vs a 60 s budget) → journey → interruption → restate → arrival.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 -- Dock Street gains a finite beat budget (see note 2 above).
 (p_world_id,v_event_scene_genesis,v_dock_street,'location','attrs.tension',            to_jsonb('normal'::text),      40,51),
 -- The Harbormaster's Office: a child of the quarter, up the road from the docks.
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.parent_location_id', to_jsonb(v_harbor_quarter::text), 40,52),
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.coordinates',        '{"x":627,"y":200}'::jsonb,    40,53),
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.tension',            to_jsonb('normal'::text),      40,54),
 (p_world_id,v_event_scene_genesis,v_harbormaster_office,'location','attrs.description',        to_jsonb('A ledger-room above the wharf; tide charts, a brass scale, and the harbourmaster''s long window over the water.'::text), 40,55),
 -- Office Door: OPEN and unlocked, dock street↔office. The way is clear; the DISTANCE is the obstacle.
 (p_world_id,v_event_scene_genesis,v_office_door,'artifact','attrs.open',               to_jsonb(true),                40,56),
 (p_world_id,v_event_scene_genesis,v_office_door,'artifact','attrs.locked',             to_jsonb(false),               40,57),
 (p_world_id,v_event_scene_genesis,v_office_door,'artifact','attrs.connects',           jsonb_build_array(v_dock_street,v_harbormaster_office), 40,58),
 -- The second hooded figure: same table, same descriptor as the first, so Kade cannot tell them apart
 -- and "the hooded figure" names both. This is what makes UNRESOLVED reachable in play.
 (p_world_id,v_event_scene_genesis,v_hooded_companion,'actor',   'attrs.location_id',        to_jsonb(v_drowned_lantern::text), 40,59),
 -- At the BAR, not the corner table: the two hooded figures wear the same descriptor, so the only way
 -- a player can tell them apart is where each one is standing. Standing them beside DIFFERENT things
 -- is what gives fn_display_names_distinct something true to say ("by the bar" vs "by the ballast
 -- crate"); put them side by side and the honest answer becomes "you cannot tell", which is a real
 -- outcome but a poor one to ship as the only one the seeded world can produce.
 (p_world_id,v_event_scene_genesis,v_hooded_companion,'actor',   'attrs.coordinates',        '{"x":5,"y":9}'::jsonb,        40,60),
 (p_world_id,v_event_scene_genesis,v_hooded_companion,'actor',   'attrs.descriptor',         to_jsonb('a hooded figure'::text), 40,61);

-- A descriptor for the ballast crate. It had none, so fn_display_name fell through to the canonical
-- registry name and a disambiguated label read "a hooded figure by the Ballast Crate" — a database
-- row wearing a sentence. Every OTHER thing a viewer can be told about carries a descriptor (§ the
-- DESCRIPTOR fallbacks block above); this closes the gap now that anchors are player-visible text.
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value,
                            valid_from_tick, valid_from_seq) VALUES
 (p_world_id,v_event_scene_genesis,v_ballast_crate,'artifact','attrs.descriptor', to_jsonb('the ballast crate'::text), 40,62);

-- SPEC-028 directory entry. THIS is the row that retires the 'Player' naming convention: the player
-- here is KADE, and because ResolveViewer could only look for an actor literally named 'Player',
-- every non-debug request against the one world anyone actually plays used to 500 at the door.
-- Theme is the tavern's own: lamplight gold, nocturne, filigree.
-- The tagline is AUTHORED FICTION and the seed is where it lives (GA-2): the service never composes
-- one, so the only way a world has a line is that somebody wrote it here. Founder-approved verbatim
-- 2026-08-09; do not reword it in passing.
--
-- DO UPDATE on the tagline specifically, and the rest still DO NOTHING. The seeds are re-run against
-- a live shared database, and `ON CONFLICT DO NOTHING` is exactly how the SPEC-031 tuning nearly
-- landed green while changing nothing in the only world anyone plays. The other columns keep DO
-- NOTHING because they are identity, not content: re-seeding must never reset a world's player or
-- rename it, but it MUST converge the fiction it owns.
INSERT INTO world (world_id, display_name, tagline, theme, player_entity_id) VALUES
 (p_world_id, 'The Drowned Lantern',
  'A harbor town where everyone is owed something, and the tide keeps the ledger.',
  '{"schema_version":"world_theme/1","accent":"#c9a227","mood":"nocturne","ornament":"filigree"}'::jsonb,
  v_kade)
ON CONFLICT (world_id) DO UPDATE SET tagline = EXCLUDED.tagline;

UPDATE world
   SET player_entity_id = v_kade,
       template_key = 'drowned_lantern'
 WHERE world_id = p_world_id;

END;
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
-- Name: fn_journey_legs(uuid, bigint); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_journey_legs(p_world_id uuid, p_span_seconds bigint) RETURNS integer
    LANGUAGE sql STABLE
    AS $$
  SELECT GREATEST(5, LEAST(10, COALESCE(
    (SELECT legs FROM journey_legs_band
       WHERE world_id = p_world_id AND max_span_seconds >= p_span_seconds
       ORDER BY max_span_seconds ASC LIMIT 1),
    CASE  -- built-in fallback (retune per-world via the table)
      WHEN p_span_seconds <= 3600 THEN 5     -- <= 1 hour
      WHEN p_span_seconds <= 86400 THEN 7    -- <= 1 day
      ELSE 10
    END
  )));
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
    WHERE ls.attrs->>'parent_location_id' IS NOT NULL
      AND c.depth < 64   -- defensive depth cap (I-4 mirror): a parent_location_id cycle can't infinite-loop
  )
  SELECT max(depth) FROM chain;
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
      -- No perception-level containment relation exists yet; "part_of" stays a stub.
      'part_of',                    NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_location_id),
      'last_known_status',          fn_compendium_latest_fact(p_world_id, p_viewer_id, p_location_id),
      -- Co-mentioned locations exist, but "inside" requires a containment edge not present in perception rows.
      'known_areas_inside',         '[]'::json,
      'key_actors',                 fn_compendium_related_entities(p_world_id, p_viewer_id, p_location_id, ARRAY['actor']),
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_location_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_location_id, ARRAY['actor','location','artifact'])
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
-- Name: fn_move_duration_actor(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_move_duration_actor(p_world_id uuid, p_actor uuid, p_target uuid) RETURNS bigint
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE
    WHEN COALESCE(fn_effective_speed(p_world_id, p_actor, 'walk'), 0) <= 0
      THEN 9223372036854775807::bigint   -- infinite duration: blocked by arithmetic (§2), not a branch
    ELSE CEIL(
      fn_distance(p_world_id, p_actor, p_target) / fn_effective_speed(p_world_id, p_actor, 'walk')
    )::bigint
  END;
$$;


--
-- Name: fn_names_in_text(uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_names_in_text(p_world_id uuid, p_text text) RETURNS TABLE(entity_id uuid, canonical_name text)
    LANGUAGE sql STABLE
    AS $$
  SELECT er.entity_id, er.canonical_name
  FROM entity_registry er
  WHERE er.world_id = p_world_id
    AND er.canonical_name IS NOT NULL
    AND er.canonical_name <> ''
    AND p_text ~ ('\m' || fn_regexp_quote(er.canonical_name) || '\M')
$$;


--
-- Name: FUNCTION fn_names_in_text(p_world_id uuid, p_text text); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.fn_names_in_text(p_world_id uuid, p_text text) IS 'The canonical names a piece of text SPEAKS, word-bounded and case-SENSITIVE. Read only by the hearing-teaches path (generate_perceptions), which must not grant name-knowledge because a sentence contained a common noun spelled like a proper name (SPEC-033).';


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
      AND a.up < 64   -- defensive depth cap (I-4 mirror)
  )
  SELECT aa.loc
  FROM anc aa
  JOIN anc bb ON bb.which = 'b' AND bb.loc = aa.loc
  WHERE aa.which = 'a'
  ORDER BY aa.up ASC
  LIMIT 1;
$$;


--
-- Name: fn_occupied_room(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_occupied_room(p_world_id uuid, p_container uuid) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
  SELECT COALESCE(SUM(fn_volume(COALESCE((attrs->>'size')::int, 1))), 0)
  FROM artifact_state
  WHERE world_id = p_world_id
    AND (attrs->>'contained_by')::uuid = p_container;
$$;


--
-- Name: fn_perceived_anchor(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_perceived_anchor(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  WITH here AS (
    SELECT a.attrs->>'location_id' AS loc
      FROM actor_state a
     WHERE a.world_id = p_world_id AND a.entity_id = p_entity_id
  )
  SELECT fn_display_name(p_world_id, p_viewer_id, art.entity_id)
    FROM artifact_state art, here
   WHERE art.world_id = p_world_id
     AND here.loc IS NOT NULL
     AND art.attrs->>'location_id' = here.loc
     AND art.attrs ? 'coordinates'
   ORDER BY fn_distance(p_world_id, p_entity_id, art.entity_id), art.entity_id
   LIMIT 1;
$$;


--
-- Name: fn_perceived_name(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_perceived_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  SELECT nm FROM (
    SELECT vp.content AS nm, vp.acquired_tick AS t
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
    JOIN perception_subject ps ON ps.perception_id = vp.perception_id AND ps.entity_id = p_entity_id
    JOIN canon_event ce ON ce.event_id = vp.source_event_id
    WHERE ce.event_type = 'world_genesis'
    UNION ALL
    SELECT nk.name, nk.learned_tick
    FROM name_knowledge nk
    WHERE nk.world_id = p_world_id AND nk.holder_id = p_viewer_id AND nk.entity_id = p_entity_id
  ) sources
  ORDER BY t
  LIMIT 1;
$$;


--
-- Name: fn_place_at(uuid, uuid, jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_place_at(p_world_id uuid, p_frame uuid, p_point jsonb) RETURNS uuid
    LANGUAGE sql STABLE
    AS $$
  SELECT ls.entity_id
  FROM location_state ls
  WHERE ls.world_id = p_world_id
    AND (ls.attrs->>'parent_location_id')::uuid = p_frame
    AND fn_area_polygon(ls.attrs) IS NOT NULL
    AND fn_area_polygon(ls.attrs) @> point((p_point->>'x')::float8, (p_point->>'y')::float8)
  ORDER BY abs(area(fn_area_polygon(ls.attrs)::path)) ASC, ls.entity_id ASC
  LIMIT 1;
$$;


--
-- Name: fn_portal_permits(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_portal_permits(p_world_id uuid, p_from uuid, p_to uuid) RETURNS boolean
    LANGUAGE sql STABLE
    AS $$
  SELECT EXISTS (
    SELECT 1 FROM artifact_state
    WHERE world_id = p_world_id
      AND attrs->>'open'   = 'true'
      AND attrs->>'locked' = 'false'
      AND attrs->'connects' ? p_from::text
      AND attrs->'connects' ? p_to::text
  );
$$;


--
-- Name: fn_pressure_chance(uuid, text, bigint); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_pressure_chance(p_world_id uuid, p_tier text, p_now bigint) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
  -- Outer COALESCE guarantees a defined number in [0,1] even for a world with
  -- no world_actor_config row: the inner FROM yields zero rows for an
  -- unconfigured (world_id, tier), which would otherwise make this scalar
  -- function return NULL rather than 0 ("no config" == "no eruptions").
  --
  -- intensity multiplies INSIDE the LEAST, not outside (whole-branch review,
  -- Fix 2): the cap is the ceiling on the returned chance — nothing is ever a
  -- guaranteed eruption. Multiplying an ALREADY-capped value by intensity
  -- (LEAST(cap, raw) * intensity) would let intensity > 1 push the result
  -- past cap, up to a guaranteed fire; multiplying BEFORE the LEAST
  -- (LEAST(cap, raw * intensity)) keeps intensity's effect — it still climbs
  -- the chance faster — while the cap remains the hard ceiling.
  SELECT COALESCE(
    (SELECT CASE WHEN COALESCE((SELECT enabled FROM world_actor_setting WHERE world_id=p_world_id), true) IS FALSE
                 THEN 0
            ELSE LEAST(c.cap,
                       c.climb_rate * ((p_now - COALESCE(
                         (SELECT max(fired_tick) FROM world_eruption WHERE world_id=p_world_id AND tier=p_tier), 0
                       ))::numeric / c.climb_chunk_ticks)
                       * COALESCE((SELECT intensity FROM world_actor_setting WHERE world_id=p_world_id), 1.0))
            END
     FROM world_actor_config c WHERE c.world_id=p_world_id AND c.tier=p_tier),
    0
  );
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
-- Name: fn_regexp_quote(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_regexp_quote(p_text text) RETURNS text
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$ SELECT regexp_replace(p_text, '([.^$*+?()\[\]{}|\\-])', '\\\1', 'g') $_$;


--
-- Name: fn_sprite_set(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_sprite_set(p_world_id uuid, p_owner_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN count(*) = 4 THEN
           json_object_agg(
             s.variant,
             json_build_object(
               'schema_version', 'image_ref/1',
               'asset_id',       s.asset_id,
               'path',           '/worlds/' || p_world_id::text || '/images/' || s.asset_id
             )
           )
         END
    FROM image_slot s
   WHERE s.world_id = p_world_id AND s.owner_kind = 'actor' AND s.owner_id = p_owner_id
     AND s.variant IN ('neutral', 'happy', 'angry', 'sad') AND s.asset_id IS NOT NULL;
$$;


--
-- Name: fn_target_position(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_target_position(p_world_id uuid, p_target uuid, OUT scene uuid, OUT coord jsonb) RETURNS record
    LANGUAGE sql STABLE
    AS $$
  SELECT
    CASE er.entity_kind
      WHEN 'location' THEN p_target
      WHEN 'actor'    THEN (ast.attrs->>'location_id')::uuid
      WHEN 'artifact' THEN COALESCE((art.attrs->>'location_id')::uuid, er.current_scene_id)
    END AS scene,
    CASE er.entity_kind
      WHEN 'location' THEN COALESCE(ls.attrs->'entry_point', '{"x":0,"y":0}'::jsonb)
      ELSE COALESCE(ast.attrs->'coordinates', art.attrs->'coordinates', '{"x":0,"y":0}'::jsonb)
    END AS coord
  FROM entity_registry er
  LEFT JOIN actor_state    ast ON ast.world_id = p_world_id AND ast.entity_id = p_target
  LEFT JOIN artifact_state art ON art.world_id = p_world_id AND art.entity_id = p_target
  LEFT JOIN location_state  ls ON  ls.world_id = p_world_id AND  ls.entity_id = p_target
  WHERE er.world_id = p_world_id AND er.entity_id = p_target;
$$;


--
-- Name: fn_timeline(uuid, uuid, bigint); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_timeline(p_world_id uuid, p_viewer_id uuid, p_before_tick bigint DEFAULT NULL::bigint) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  WITH mine AS (
    SELECT v.perception_id,
           v.content,
           v.epistemic_type,
           v.valid_tick,
           v.confidence,
           ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN canon_event ce
      ON ce.event_id = v.source_event_id
    WHERE v.holder_id = p_viewer_id
      AND (p_before_tick IS NULL OR v.valid_tick < p_before_tick)
  )
  SELECT json_build_object(
    'schema_version', 'timeline/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'records', coalesce(
      (SELECT json_agg(
                json_build_object(
                  'perception_id',    m.perception_id,
                  'content',          m.content,
                  'epistemic_type',   m.epistemic_type,
                  'occurred_at_tick', m.valid_tick,
                  'display_label',    m.in_world_label,
                  'confidence',       m.confidence,
                  'decay',            fn_compendium_decay(p_world_id, m.valid_tick, m.in_world_label)
                )
                ORDER BY m.valid_tick, m.perception_id
              )
       FROM mine m),
      '[]'::json
    )
  );
$$;


--
-- Name: fn_transcript(uuid, uuid, bigint, integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_transcript(p_world_id uuid, p_viewer_id uuid, p_before bigint DEFAULT NULL::bigint, p_limit integer DEFAULT 50) RETURNS jsonb
    LANGUAGE sql STABLE
    AS $$
  WITH lim AS (
    -- Bounded server-side: a client asking for a million entries gets 200. 50 when unspecified.
    SELECT LEAST(GREATEST(COALESCE(p_limit, 50), 1), 200) AS n
  ), page AS (
    SELECT te.entry_no, te.in_world_tick, te.stated, te.segments, te.halt_reason, te.journey
    FROM transcript_entry te, lim
    WHERE te.world_id = p_world_id
      AND te.viewer_id = p_viewer_id
      AND (p_before IS NULL OR te.entry_no < p_before)
    ORDER BY te.entry_no DESC
    LIMIT (SELECT n FROM lim)
  )
  SELECT jsonb_build_object(
    'schema_version', 'transcript/2',
    'world_id',       p_world_id,
    'viewer_id',      p_viewer_id,
    'entries',        COALESCE((
      SELECT jsonb_agg(jsonb_build_object(
               'entry_no',    p.entry_no,
               'tick',        p.in_world_tick,
               'stated',      p.stated,
               'halt_reason', p.halt_reason,
               'journey',     p.journey,
               'segments',    p.segments
             ) ORDER BY p.entry_no DESC)
      FROM page p), '[]'::jsonb),
    -- The oldest entry on this page is the exclusive cursor for the next one. Null when this page
    -- reached the beginning: no more story behind it.
    'next_before',    (
      SELECT CASE WHEN EXISTS (
               SELECT 1 FROM transcript_entry older
               WHERE older.world_id = p_world_id AND older.viewer_id = p_viewer_id
                 AND older.entry_no < (SELECT min(entry_no) FROM page))
             THEN (SELECT min(entry_no) FROM page) END
      FROM page LIMIT 1)
  );
$$;


--
-- Name: FUNCTION fn_transcript(p_world_id uuid, p_viewer_id uuid, p_before bigint, p_limit integer); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.fn_transcript(p_world_id uuid, p_viewer_id uuid, p_before bigint, p_limit integer) IS 'transcript/2 — one viewer''s delivered story, newest-first, cursor-paginated on entry_no. Returns stored prose verbatim: no re-labelling, no re-derivation; segments may carry the optional narration/3 emotion tag.';


--
-- Name: fn_unearned_names(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_unearned_names(p_world_id uuid, p_viewer uuid) RETURNS TABLE(canonical_name text, label text)
    LANGUAGE sql STABLE
    AS $$
  WITH unearned AS (
    SELECT er.entity_id, er.entity_kind, er.canonical_name,
           fn_display_name(p_world_id, p_viewer, er.entity_id) AS label
    FROM entity_registry er
    WHERE er.world_id = p_world_id
      AND er.canonical_name IS NOT NULL
      AND er.canonical_name <> ''
      -- A holder always knows who HE is: rewriting a man's own name to the descriptor strangers use
      -- for him is not perception, it is amnesia.
      AND er.entity_id IS DISTINCT FROM p_viewer
      -- No knowledge path: fn_display_name fell through to something other than the registry name.
      -- When they AGREE the name is either earned or the only label the world has, and inventing a
      -- placeholder for the latter would fabricate a perception.
      AND fn_display_name(p_world_id, p_viewer, er.entity_id) IS DISTINCT FROM er.canonical_name
      -- ...and the label does not already contain the name (the Ballast Crate case).
      AND position(lower(er.canonical_name) IN lower(coalesce(fn_display_name(p_world_id, p_viewer, er.entity_id), ''))) = 0
  ),
  -- What the viewer calls HIMSELF: his registry name and his own display label. No token of these
  -- is ever guarded, whoever else's name it appears in.
  self_names AS (
    SELECT coalesce(er.canonical_name, '') AS nm,
           coalesce(fn_display_name(p_world_id, p_viewer, p_viewer), '') AS lbl
    FROM entity_registry er
    WHERE er.world_id = p_world_id AND er.entity_id = p_viewer
  ),
  -- Every piece of prose this world has ever written down, for the lowercase test above.
  corpus AS (
    SELECT ce.summary AS t FROM canon_event ce WHERE ce.world_id = p_world_id AND ce.summary IS NOT NULL
    UNION ALL SELECT ce.payload->>'spoken' FROM canon_event ce WHERE ce.world_id = p_world_id AND ce.payload ? 'spoken'
    UNION ALL SELECT er.descriptor FROM entity_registry er WHERE er.world_id = p_world_id AND er.descriptor IS NOT NULL
    UNION ALL SELECT a.attrs->>'descriptor' FROM actor_state a WHERE a.world_id = p_world_id AND a.attrs ? 'descriptor'
    UNION ALL SELECT f.attrs->>'descriptor' FROM artifact_state f WHERE f.world_id = p_world_id AND f.attrs ? 'descriptor'
    UNION ALL SELECT l.attrs->>'descriptor' FROM location_state l WHERE l.world_id = p_world_id AND l.attrs ? 'descriptor'
    UNION ALL SELECT l.attrs->>'description' FROM location_state l WHERE l.world_id = p_world_id AND l.attrs ? 'description'
  ),
  tokens AS (
    SELECT DISTINCT ON (lower(tok)) tok, u.label
    FROM unearned u
    CROSS JOIN LATERAL regexp_split_to_table(u.canonical_name, '[^[:alnum:]]+') AS tok
    WHERE u.entity_kind = 'actor'
      AND length(tok) >= 3
      AND lower(tok) NOT IN ('the','and','von','van','der','den','del','della','delle','dos','das',
                             'bin','ibn','abu','mac','mck','saint','sant','santa')
      AND lower(tok) <> lower(u.canonical_name)
      AND coalesce(u.label, '') !~* ('\m' || fn_regexp_quote(tok) || '\M')
      AND NOT EXISTS (
        SELECT 1 FROM self_names s
        WHERE s.nm  ~* ('\m' || fn_regexp_quote(tok) || '\M')
           OR s.lbl ~* ('\m' || fn_regexp_quote(tok) || '\M'))
      -- The lowercase test: case-SENSITIVE match of the lowercased token against world prose.
      AND NOT EXISTS (
        SELECT 1 FROM corpus c
        WHERE c.t ~ ('\m' || fn_regexp_quote(lower(tok)) || '\M'))
    ORDER BY lower(tok), u.label
  )
  SELECT canonical_name, label FROM (
    SELECT u.canonical_name, u.label FROM unearned u
    UNION ALL
    SELECT t.tok, t.label FROM tokens t
  ) guarded
  -- Longest first: "Silas Holton" is rewritten as ONE label before "Silas" or "Holton" can bite
  -- into it — the ORDER BY is part of the shared definition, not an incidental detail.
  ORDER BY length(canonical_name) DESC
$$;


--
-- Name: FUNCTION fn_unearned_names(p_world_id uuid, p_viewer uuid); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.fn_unearned_names(p_world_id uuid, p_viewer uuid) IS 'The canonical names a viewer has NOT earned — and, for people, every distinctive word of each — with the label he holds instead. The single definition behind both the perception seam (fn_viewer_text) and the API-boundary belt (NamingWall in core/api) — naming reach, RULINGS-2026-07-23 §3; token guarding is the Ironmoor fix, 2026-08-20.';


--
-- Name: fn_viewer_text(uuid, uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_viewer_text(p_world_id uuid, p_holder uuid, p_text text) RETURNS text
    LANGUAGE plpgsql STABLE
    AS $$
DECLARE
  r        record;
  outtxt   text := p_text;
  bare     text;
  namepat  text;
BEGIN
  IF p_text IS NULL OR p_holder IS NULL THEN
    RETURN p_text;
  END IF;

  FOR r IN SELECT * FROM fn_unearned_names(p_world_id, p_holder) LOOP
    namepat := '\m' || fn_regexp_quote(r.canonical_name) || '\M';

    -- The label without its own leading article, when it has one. "a hooded figure" -> "hooded
    -- figure"; "the keeper" -> "keeper"; "Mara" -> "Mara" (unchanged, no article to strip).
    bare := regexp_replace(r.label, '^(a|an|the)\s+', '', 'i');

    IF bare <> r.label THEN
      -- Pass 1: the name already has an article in front of it. Keep the sentence's article exactly as
      -- written (\1 preserves "The" vs "the") and drop the label's.
      outtxt := regexp_replace(outtxt, '(\m(?:the|an|a)\s+)' || namepat, '\1' || bare, 'gi');
    END IF;

    -- Pass 2: every remaining occurrence takes the label whole.
    outtxt := regexp_replace(outtxt, namepat, r.label, 'gi');
  END LOOP;

  RETURN outtxt;
END $$;


--
-- Name: FUNCTION fn_viewer_text(p_world_id uuid, p_holder uuid, p_text text); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.fn_viewer_text(p_world_id uuid, p_holder uuid, p_text text) IS 'Rewrites canonical names in perception text into the labels the holder has earned (naming reach, RULINGS-2026-07-23 §3). Identity for a holder who has earned every name, and for entities the world can only name canonically.';


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
-- Name: fn_volume(integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_volume(p_size integer) RETURNS numeric
    LANGUAGE sql IMMUTABLE
    AS $$
  SELECT power(4, p_size - 1)::numeric;   -- fn_volume(1)=1, fn_volume(5)=256
$$;


--
-- Name: fn_watch_horizon_seconds(uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_watch_horizon_seconds(p_world_id uuid) RETURNS bigint
    LANGUAGE sql STABLE
    AS $$
  SELECT COALESCE(
    (SELECT horizon_seconds FROM watch_horizon WHERE world_id = p_world_id),
    86400  -- built-in fallback (retune per-world via the table)
  );
$$;


--
-- Name: fn_world_directory(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_world_directory() RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
    'schema_version', 'world_directory/2',
    'worlds', COALESCE((
      SELECT json_agg(json_build_object(
               'id',            w.world_id,
               'display_name',  w.display_name,
               'tagline',       w.tagline,
               'theme',         w.theme,
               'playable',      w.player_entity_id IS NOT NULL,
               'cover_image',   fn_image_ref(w.world_id, 'world', w.world_id),
               'last_place_label', (
                  SELECT fn_display_name(w.world_id, w.player_entity_id,
                                         (a.attrs->>'location_id')::uuid)
                    FROM actor_state a
                   WHERE a.world_id = w.world_id
                     AND a.entity_id = w.player_entity_id
                     AND a.attrs->>'location_id' IS NOT NULL
               )
             ) ORDER BY w.display_name, w.world_id)
        FROM world w
       WHERE w.archived_at IS NULL), '[]'::json)
  );
$$;


--
-- Name: fn_world_now(uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_world_now(p_world_id uuid) RETURNS bigint
    LANGUAGE sql STABLE
    AS $$
  SELECT GREATEST(
    COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id = p_world_id), 0),
    COALESCE((SELECT max(current_tick)  FROM journey     WHERE world_id = p_world_id), 0)
  );
$$;


--
-- Name: fn_world_slice(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_world_slice(p_world_id uuid, p_scene uuid) RETURNS jsonb
    LANGUAGE sql STABLE
    AS $$
  WITH loc AS (
    -- The world's location rows ({id,name,attrs}) computed ONCE; both `locations`
    -- (all of them) and `scene` (the p_scene one) select from this, instead of
    -- running the same entity_registry⋈location_state join twice.
    SELECT er.entity_id AS id, er.canonical_name AS name, COALESCE(ls.attrs, '{}'::jsonb) AS attrs
    FROM entity_registry er
    LEFT JOIN location_state ls
      ON ls.world_id = er.world_id AND ls.entity_id = er.entity_id
    WHERE er.world_id = p_world_id AND er.entity_kind = 'location'
  )
  SELECT jsonb_build_object(
    'ledger',
    COALESCE(
      (SELECT jsonb_agg(jsonb_build_object(
                 'pending_id',   pe.pending_id,
                 'fire_at_tick', pe.fire_at_tick,
                 'magnitude',    pe.magnitude,
                 'payload',      pe.payload
               ) ORDER BY pe.fire_at_tick)
       FROM pending_event pe
       WHERE pe.world_id = p_world_id AND pe.status = 'pending'),
      '[]'::jsonb),

    'presence',
    COALESCE(
      (SELECT jsonb_agg(jsonb_build_object(
                 'actor',    a.entity_id,
                 'location', (a.attrs->>'location_id')::uuid
               ) ORDER BY a.entity_id)
       FROM actor_state a
       WHERE a.world_id = p_world_id),
      '[]'::jsonb),

    'locations',
    COALESCE(
      (SELECT jsonb_agg(jsonb_build_object(
                 'id',    loc.id,
                 'name',  loc.name,
                 'attrs', loc.attrs
               ) ORDER BY loc.id)
       FROM loc),
      '[]'::jsonb),

    'recent',
    COALESCE(
      (SELECT jsonb_agg(r.obj ORDER BY r.in_world_tick DESC, r.beat_seq DESC)
       FROM (
         SELECT ce.in_world_tick, ce.beat_seq,
                jsonb_build_object(
                  'event_id',      ce.event_id,
                  'event_type',    ce.event_type,
                  'summary',       ce.summary,
                  'in_world_tick', ce.in_world_tick,
                  'beat_seq',      ce.beat_seq
                ) AS obj
         FROM canon_event ce
         WHERE ce.world_id = p_world_id AND ce.status = 'accepted'
         ORDER BY ce.in_world_tick DESC, ce.beat_seq DESC
         LIMIT 20
       ) r),
      '[]'::jsonb),

    'scene',
    (SELECT jsonb_build_object(
               'id',    loc.id,
               'name',  loc.name,
               'attrs', loc.attrs
             )
     FROM loc WHERE loc.id = p_scene)
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
  -- said = the utterance as a holder PERCEIVES it: the referee's account plus the words themselves.
  said text;
  -- spoken = the words THEMSELVES, and the only thing that can teach a name. NULL when the event
  -- carries no utterance (a nod, a shove, an ambient noise), and a NULL text teaches nothing because
  -- fn_names_in_text's regex match over NULL yields no rows.
  spoken text;
BEGIN
  SELECT * INTO ev FROM canon_event WHERE event_id = p_event_id AND status = 'accepted';
  IF NOT FOUND THEN RETURN 0; END IF;

  IF ev.event_type IN ('private_disclosure', 'Communicated') THEN
    spoken := NULLIF(TRIM(COALESCE(ev.payload->>'spoken','')),'');
    said := ev.summary;
    IF spoken IS NOT NULL
       -- ...unless the account ALREADY is the words. The legacy `say` step commits with summary =
       -- content, so appending would render 'I saw the note — "I saw the note"'.
       AND position(spoken IN ev.summary) = 0 THEN
      said := ev.summary || ' — "' || spoken || '"';
    END IF;
    -- speaker → 'shared'; each listener → 'told' (B-7).
    SELECT entity_id INTO spk FROM event_participant
      WHERE event_id = p_event_id AND role_qualifier = 'speaker' LIMIT 1;
    IF spk IS NOT NULL THEN
      -- SPEC-033: a name in what was SAID OUT LOUD is earned by whoever heard it said.
      INSERT INTO name_knowledge (world_id, holder_id, entity_id, name, learned_tick, source_event_id)
      SELECT ev.world_id, spk, t.entity_id, t.canonical_name, ev.in_world_tick, p_event_id
        FROM fn_names_in_text(ev.world_id, spoken) t
       WHERE t.entity_id <> spk
      ON CONFLICT DO NOTHING;

      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, spk, p_event_id, fn_viewer_text(ev.world_id, spk, said), 'shared',
              ev.in_world_tick, ev.in_world_tick)
      RETURNING perception_id INTO pid;
      INSERT INTO perception_subject (perception_id, entity_id, world_id)
      SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
      WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
      n := n + 1;
    END IF;
    FOR lst IN SELECT entity_id FROM event_participant
                 WHERE event_id = p_event_id AND role_qualifier = 'listener' LOOP
      -- SPEC-033, the reported case: Mara SAYS "Jonas" where Kade can hear it, and Kade learns it.
      -- The case the founder caught: Mara nods at Jonas, the account says "Jonas", and Kade learns
      -- nothing — because nobody said it.
      INSERT INTO name_knowledge (world_id, holder_id, entity_id, name, learned_tick, source_event_id)
      SELECT ev.world_id, lst, t.entity_id, t.canonical_name, ev.in_world_tick, p_event_id
        FROM fn_names_in_text(ev.world_id, spoken) t
       WHERE t.entity_id <> lst
      ON CONFLICT DO NOTHING;

      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, lst, p_event_id, fn_viewer_text(ev.world_id, lst, said), 'told',
              ev.in_world_tick, ev.in_world_tick)
      RETURNING perception_id INTO pid;
      INSERT INTO perception_subject (perception_id, entity_id, world_id)
      SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
      WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
      n := n + 1;
    END LOOP;
  END IF;

  IF ev.event_type IN ('move', 'ActorMoved') THEN
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
        INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                       acquired_tick, valid_tick)
        VALUES (ev.world_id, mover, p_event_id, fn_viewer_text(ev.world_id, mover, ev.summary),
                'direct', ev.in_world_tick, ev.in_world_tick)
        RETURNING perception_id INTO pid;
        INSERT INTO perception_subject (perception_id, entity_id, world_id)
        SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
        WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
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

  -- ── SPEC-035 amends this arm: witnesses perceive too. See the perceiver set below and the
  -- gate + participant recording in the same migration. The arm's shape is unchanged.
  --
  -- ── SPEC-034: ObjectRelocated ─────────────────────────────────────────────────────────────────
  -- Reproduced on the seeded world before this was written: Kade carries the Sealed Note
  -- (fn_carrying lists it) and fn_entity_visible(DL,Kade,note) is FALSE, fn_artifact_page returns
  -- NULL => 404, and Kade holds ZERO perceptions about a thing in his hands. There was no
  -- ObjectRelocated arm at all, so a handover recorded nothing for anybody — not even the new holder.
  -- Possession without knowledge is not a lawful epistemic state (B-2; I-2 provenance).
  --
  -- THE SUBJECT IS THE POINT. fn_entity_visible asks "does this holder hold a perception whose
  -- SUBJECT is this entity". The two arms above take subjects from event_participant, and
  -- ObjectRelocated records only an `instigator` (apply_event: "Communicated -> speaker + listener;
  -- all others -> instigator") — the object and destination live in the PAYLOAD. So a perception
  -- alone would not lift the 404; the object must be named as a subject explicitly. That is the
  -- defect, and it is why this arm reads the payload instead of copying the participant pattern.
  --
  -- WHO PERCEIVES (founder ruling 2026-08-25): "holders and co-present as long as they can see it".
  -- The second half cannot be honoured today and the reason is recorded rather than worked around:
  -- no concealment signal exists (ObjectRelocated's two founder-locked dimensions are volume, which
  -- blocks, and weight, which consequences — neither epistemic); actors are AT a place, not
  -- positioned within it, so there is no sub-place geometry to answer "could they see it"; and this
  -- world's pattern for who perceived something is that the EVENT SAYS SO — Communicated carries a
  -- required singular listener_id, ObjectRelocated carries no witness field. SPEC-033's correction is
  -- the precedent: stop inferring from the referee's account, read what the payload actually states.
  -- So this arm records only what the event names. Witnesses are SPEC-035, and that is the urgent
  -- half: no rule can recover who was watching if the event never recorded it.
  --
  -- SCOPE, measured: the play world holds ZERO ObjectRelocated events — the seeded carry edges were
  -- authored as state, never as events. This arm cannot repair the seeded row; it prevents every
  -- FUTURE handover from creating the same hole. Converting seeded state to events is a stated
  -- non-goal of the plan.
  IF ev.event_type = 'ObjectRelocated' THEN
    DECLARE
      or_obj  uuid;
      or_dest uuid;
      or_kind text;
      or_who  uuid;
      or_pid  uuid;
    BEGIN
      -- WHERE THE RELOCATION ACTUALLY LIVES. Measured: apply_event commits an ObjectRelocated with an
      -- EMPTY canon_event.payload — `{}`. The object and destination are not in the event at all; they
      -- are in the state_mutation the commit writes:
      --
      --     entity_id = <the object>   attribute_path = 'attrs.contained_by'   new_value = <destination>
      --
      -- So this arm reads state_mutation, exactly as the move/ActorMoved arm above reads
      -- `attrs.location_id` for its destination. The first draft of this arm read ev.payload and
      -- silently produced zero perceptions, because a NULL object_id makes the whole arm a no-op — a
      -- fix that looked applied, passed `make migrate`, and changed nothing. Found by running it.
      SELECT sm.entity_id, (sm.new_value #>> '{}')::uuid
        INTO or_obj, or_dest
        FROM state_mutation sm
       WHERE sm.event_id = p_event_id
         AND sm.attribute_path = 'attrs.contained_by'
       LIMIT 1;

      IF or_obj IS NOT NULL THEN
        -- The destination's kind comes from entity_registry, never from a payload field: the payload
        -- is empty here, and even where beat_chain.v2 marks `dest_kind` required, apply_event's gate
        -- only verifies the ids EXIST (20260723100004_apply_event.sql:149-155).
        SELECT entity_kind INTO or_kind FROM entity_registry
          WHERE entity_id = or_dest AND world_id = ev.world_id;

        -- Whoever the event names, deduplicated: the actor who did it, and the destination when the
        -- destination is a person. A location or container destination names no perceiver — "the
        -- packet is now in the back room" is not knowledge anybody acquired.
        --
        -- SPEC-035: and whoever the event names as a WITNESS. They take the same branch as the
        -- holders on purpose, because the founder's ruling is that they are in the same epistemic
        -- position: they saw it happen, so 'direct', and the same subjects. What separates a witness
        -- from a co-present bystander is not the perception rule, it is that apply_event only records
        -- the ones the caller NAMED and could prove were there — "just because they were there
        -- doesn't mean they saw it" is enforced at the gate, not re-litigated here. This is the same
        -- division 'Communicated' already makes: addressed listeners get 'told', and the comment on
        -- that arm says co-present overhearers defer. Concealment, when it lands, shortens the list
        -- the caller passes; it needs no change on this side.
        FOR or_who IN
          SELECT DISTINCT h FROM (
            SELECT entity_id AS h FROM event_participant
              WHERE event_id = p_event_id AND role_qualifier IN ('instigator', 'witness')
            UNION
            SELECT or_dest WHERE or_dest IS NOT NULL AND or_kind = 'actor'
          ) s WHERE h IS NOT NULL
        LOOP
          INSERT INTO perception_record (world_id, holder_id, source_event_id, content,
                                         epistemic_type, acquired_tick, valid_tick)
          VALUES (ev.world_id, or_who, p_event_id,
                  fn_viewer_text(ev.world_id, or_who, ev.summary), 'direct',
                  ev.in_world_tick, ev.in_world_tick)
          RETURNING perception_id INTO or_pid;

          -- THE FIX. fn_entity_visible asks "does this holder hold a perception whose SUBJECT is this
          -- entity". The two arms above take subjects from event_participant, and ObjectRelocated
          -- records only an `instigator` — so without naming the OBJECT here, a perception exists and
          -- the 404 survives it. That is the defect.
          INSERT INTO perception_subject (perception_id, entity_id, world_id)
          SELECT or_pid, e, ev.world_id FROM (
            SELECT or_obj AS e
            UNION SELECT or_dest WHERE or_dest IS NOT NULL
            UNION SELECT entity_id FROM event_participant WHERE event_id = p_event_id
          ) s WHERE e IS NOT NULL
          ON CONFLICT DO NOTHING;

          n := n + 1;
        END LOOP;
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

  -- DETERMINISTIC-DOMAIN-ORDER (was 0A Rider C): domain-only deterministic order. recorded_at (volatile) excluded.
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
-- Name: duration_class_seconds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.duration_class_seconds (
    world_id uuid NOT NULL,
    class text NOT NULL,
    seconds bigint NOT NULL,
    CONSTRAINT duration_class_seconds_class_check CHECK ((class = ANY (ARRAY['instant'::text, 'short'::text, 'medium'::text, 'long'::text, 'extremely_long'::text]))),
    CONSTRAINT duration_class_seconds_seconds_check CHECK ((seconds > 0))
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
-- Name: extent_class_metres; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.extent_class_metres (
    world_id uuid NOT NULL,
    class text NOT NULL,
    radius_m numeric NOT NULL,
    CONSTRAINT extent_class_metres_class_check CHECK ((class = ANY (ARRAY['intimate'::text, 'small'::text, 'medium'::text, 'large'::text, 'vast'::text]))),
    CONSTRAINT extent_class_metres_radius_m_check CHECK ((radius_m > (0)::numeric))
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
-- Name: image_slot; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.image_slot (
    world_id uuid NOT NULL,
    owner_kind text NOT NULL,
    owner_id uuid NOT NULL,
    visual_identity_id text,
    asset_id text,
    job_id text,
    idempotency_key text,
    issued_at timestamp with time zone,
    last_error text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    variant text DEFAULT 'default'::text NOT NULL,
    CONSTRAINT image_slot_owner_kind_check CHECK ((owner_kind = ANY (ARRAY['actor'::text, 'location'::text, 'artifact'::text, 'world'::text]))),
    CONSTRAINT image_slot_variant_check CHECK ((variant = ANY (ARRAY['default'::text, 'neutral'::text, 'happy'::text, 'angry'::text, 'sad'::text])))
);


--
-- Name: journey; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.journey (
    journey_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    kind text NOT NULL,
    threshold jsonb NOT NULL,
    span_seconds bigint NOT NULL,
    legs_total integer NOT NULL,
    legs_done integer DEFAULT 0 NOT NULL,
    started_tick bigint NOT NULL,
    current_tick bigint NOT NULL,
    frame_id uuid,
    origin_coord jsonb,
    goal_coord jsonb,
    goal_target uuid,
    stage_id uuid,
    status text DEFAULT 'active'::text NOT NULL,
    CONSTRAINT journey_kind_check CHECK ((kind = ANY (ARRAY['travel'::text, 'wait'::text, 'watch'::text]))),
    CONSTRAINT journey_legs_done_check CHECK ((legs_done >= 0)),
    CONSTRAINT journey_legs_total_check CHECK ((legs_total > 0)),
    CONSTRAINT journey_span_seconds_check CHECK ((span_seconds > 0)),
    CONSTRAINT journey_status_check CHECK ((status = ANY (ARRAY['active'::text, 'arrived'::text, 'ended'::text])))
);


--
-- Name: journey_legs_band; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.journey_legs_band (
    world_id uuid NOT NULL,
    max_span_seconds bigint NOT NULL,
    legs integer NOT NULL,
    CONSTRAINT journey_legs_band_legs_check CHECK ((legs > 0)),
    CONSTRAINT journey_legs_band_max_span_seconds_check CHECK ((max_span_seconds > 0))
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
    created_by_event uuid,
    CONSTRAINT movement_type_base_speed_mps_check CHECK ((base_speed_mps > (0)::numeric))
);


--
-- Name: name_knowledge; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.name_knowledge (
    world_id uuid NOT NULL,
    holder_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    name text NOT NULL,
    learned_tick bigint NOT NULL,
    source_event_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE name_knowledge; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.name_knowledge IS 'Names a holder has learned IN PLAY (SPEC-033). Genesis-seeded name-knowledge stays in perception_record; fn_perceived_name reads both. Perception layer, never canon.';


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
    created_by_event uuid,
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
-- Name: transcript_entry; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transcript_entry (
    entry_no bigint NOT NULL,
    world_id uuid NOT NULL,
    viewer_id uuid NOT NULL,
    in_world_tick bigint NOT NULL,
    stated text,
    segments jsonb NOT NULL,
    halt_reason text,
    journey jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE transcript_entry; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.transcript_entry IS 'The viewer''s lived story as DELIVERED: rendered prose, post-belt, never retro-labelled. Not a projection — the prose is unrecoverable from world state. Viewer-scoped; entry_no orders and paginates.';


--
-- Name: transcript_entry_entry_no_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.transcript_entry ALTER COLUMN entry_no ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.transcript_entry_entry_no_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: watch_horizon; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.watch_horizon (
    world_id uuid NOT NULL,
    horizon_seconds bigint NOT NULL,
    CONSTRAINT watch_horizon_horizon_seconds_check CHECK ((horizon_seconds > 0))
);


--
-- Name: world; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world (
    world_id uuid NOT NULL,
    display_name text NOT NULL,
    theme jsonb DEFAULT '{"mood": "nocturne", "accent": "#c9a227", "ornament": "filigree", "schema_version": "world_theme/1"}'::jsonb NOT NULL,
    player_entity_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tagline text,
    template_key text,
    archived_at timestamp with time zone,
    brief text,
    art_style text,
    genesis_doc jsonb,
    kickstart_state jsonb,
    world_identity jsonb,
    CONSTRAINT world_art_style_check CHECK (((art_style IS NULL) OR (length(btrim(art_style)) > 0))),
    CONSTRAINT world_brief_check CHECK (((brief IS NULL) OR (length(btrim(brief)) > 0))),
    CONSTRAINT world_display_name_check CHECK ((length(btrim(display_name)) > 0)),
    CONSTRAINT world_tagline_check CHECK (((tagline IS NULL) OR (length(btrim(tagline)) > 0))),
    CONSTRAINT world_theme_check CHECK ((((theme ->> 'schema_version'::text) = 'world_theme/1'::text) AND ((theme ->> 'accent'::text) ~ '^#[0-9a-fA-F]{6}$'::text) AND (length(COALESCE((theme ->> 'mood'::text), ''::text)) > 0) AND (length(COALESCE((theme ->> 'ornament'::text), ''::text)) > 0)))
);


--
-- Name: COLUMN world.template_key; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.world.template_key IS 'Template lineage key so worlds can be re-instantiated without deleting canon (append-only, ADR-001/006).';


--
-- Name: COLUMN world.archived_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.world.archived_at IS 'Superseded marker so retired worlds leave canon intact but drop out of active directory listings (append-only, ADR-001/006).';


--
-- Name: COLUMN world.brief; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.world.brief IS 'The prose a user typed to author this world. Operational provenance, never rendered: no projection selects it. NULL for hand-authored worlds.';


--
-- Name: COLUMN world.art_style; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.world.art_style IS 'Requested art style choice for this world (preset key or custom prose). NULL means no explicit choice and resolves to the house fallback profile.';


--
-- Name: COLUMN world.world_identity; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.world.world_identity IS 'world_identity/1 from the understanding pass. Server-side only. NULL if the world was not built this way.';


--
-- Name: world_actor_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_actor_config (
    world_id uuid NOT NULL,
    tier text NOT NULL,
    climb_rate numeric NOT NULL,
    climb_chunk_ticks bigint NOT NULL,
    cap numeric NOT NULL,
    CONSTRAINT world_actor_config_cap_check CHECK (((cap >= (0)::numeric) AND (cap <= (1)::numeric))),
    CONSTRAINT world_actor_config_climb_chunk_ticks_check CHECK ((climb_chunk_ticks > 0)),
    CONSTRAINT world_actor_config_climb_rate_check CHECK ((climb_rate >= (0)::numeric)),
    CONSTRAINT world_actor_config_tier_check CHECK ((tier = ANY (ARRAY['small'::text, 'medium'::text, 'large'::text])))
);


--
-- Name: world_actor_setting; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_actor_setting (
    world_id uuid NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    intensity numeric DEFAULT 1.0 NOT NULL,
    CONSTRAINT world_actor_setting_intensity_check CHECK ((intensity >= (0)::numeric))
);


--
-- Name: world_character; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_character (
    character_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    descriptor text NOT NULL,
    canonical_name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT world_character_canonical_name_check CHECK ((length(btrim(canonical_name)) > 0)),
    CONSTRAINT world_character_descriptor_check CHECK ((length(btrim(descriptor)) > 0))
);


--
-- Name: world_eruption; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_eruption (
    eruption_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    tier text NOT NULL,
    fired_tick bigint NOT NULL,
    event_id uuid NOT NULL,
    CONSTRAINT world_eruption_tier_check CHECK ((tier = ANY (ARRAY['small'::text, 'medium'::text, 'large'::text])))
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
-- Name: duration_class_seconds duration_class_seconds_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.duration_class_seconds
    ADD CONSTRAINT duration_class_seconds_pkey PRIMARY KEY (world_id, class);


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
-- Name: extent_class_metres extent_class_metres_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extent_class_metres
    ADD CONSTRAINT extent_class_metres_pkey PRIMARY KEY (world_id, class);


--
-- Name: held_outcome held_outcome_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.held_outcome
    ADD CONSTRAINT held_outcome_pkey PRIMARY KEY (held_id);


--
-- Name: image_slot image_slot_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_slot
    ADD CONSTRAINT image_slot_pkey PRIMARY KEY (world_id, owner_kind, owner_id, variant);


--
-- Name: journey_legs_band journey_legs_band_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.journey_legs_band
    ADD CONSTRAINT journey_legs_band_pkey PRIMARY KEY (world_id, max_span_seconds);


--
-- Name: journey journey_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.journey
    ADD CONSTRAINT journey_pkey PRIMARY KEY (journey_id);


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
-- Name: name_knowledge name_knowledge_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.name_knowledge
    ADD CONSTRAINT name_knowledge_pkey PRIMARY KEY (world_id, holder_id, entity_id);


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
-- Name: transcript_entry transcript_entry_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transcript_entry
    ADD CONSTRAINT transcript_entry_pkey PRIMARY KEY (entry_no);


--
-- Name: watch_horizon watch_horizon_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.watch_horizon
    ADD CONSTRAINT watch_horizon_pkey PRIMARY KEY (world_id);


--
-- Name: world_actor_config world_actor_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_actor_config
    ADD CONSTRAINT world_actor_config_pkey PRIMARY KEY (world_id, tier);


--
-- Name: world_actor_setting world_actor_setting_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_actor_setting
    ADD CONSTRAINT world_actor_setting_pkey PRIMARY KEY (world_id);


--
-- Name: world_character world_character_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_character
    ADD CONSTRAINT world_character_pkey PRIMARY KEY (character_id);


--
-- Name: world_eruption world_eruption_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_eruption
    ADD CONSTRAINT world_eruption_pkey PRIMARY KEY (eruption_id);


--
-- Name: world world_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world
    ADD CONSTRAINT world_pkey PRIMARY KEY (world_id);


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
-- Name: idx_image_slot_in_flight; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_slot_in_flight ON public.image_slot USING btree (world_id) WHERE (job_id IS NOT NULL);


--
-- Name: idx_image_slot_unfilled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_slot_unfilled ON public.image_slot USING btree (world_id) WHERE ((asset_id IS NULL) AND (job_id IS NULL));


--
-- Name: idx_journey_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_journey_active ON public.journey USING btree (world_id) WHERE (status = 'active'::text);


--
-- Name: idx_journey_one_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_journey_one_active ON public.journey USING btree (world_id, actor_id) WHERE (status = 'active'::text);


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
-- Name: idx_transcript_viewer_recent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transcript_viewer_recent ON public.transcript_entry USING btree (world_id, viewer_id, entry_no DESC);


--
-- Name: idx_world_eruption_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_world_eruption_lookup ON public.world_eruption USING btree (world_id, tier, fired_tick DESC);


--
-- Name: uq_ce_accepted_order; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_ce_accepted_order ON public.canon_event USING btree (world_id, in_world_tick, beat_seq) WHERE (status = 'accepted'::text);


--
-- Name: world_character_world_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX world_character_world_idx ON public.world_character USING btree (world_id);


--
-- Name: location_state location_state_tension; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER location_state_tension BEFORE INSERT OR UPDATE ON public.location_state FOR EACH ROW EXECUTE FUNCTION public.trg_validate_tension();


--
-- Name: canon_event trg_canon_event_append_only; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_canon_event_append_only BEFORE UPDATE ON public.canon_event FOR EACH ROW EXECUTE FUNCTION public.canon_event_append_only();


--
-- Name: canon_event trg_canon_event_carry_in_world_label; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_canon_event_carry_in_world_label BEFORE INSERT ON public.canon_event FOR EACH ROW EXECUTE FUNCTION public.canon_event_carry_in_world_label();


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
-- Name: movement_type movement_type_created_by_event_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.movement_type
    ADD CONSTRAINT movement_type_created_by_event_fkey FOREIGN KEY (created_by_event) REFERENCES public.canon_event(event_id);


--
-- Name: name_knowledge name_knowledge_source_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.name_knowledge
    ADD CONSTRAINT name_knowledge_source_event_id_fkey FOREIGN KEY (source_event_id) REFERENCES public.canon_event(event_id);


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
-- Name: status_modifier status_modifier_created_by_event_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.status_modifier
    ADD CONSTRAINT status_modifier_created_by_event_fkey FOREIGN KEY (created_by_event) REFERENCES public.canon_event(event_id);


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
-- Name: world_character world_character_world_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_character
    ADD CONSTRAINT world_character_world_id_fkey FOREIGN KEY (world_id) REFERENCES public.world(world_id) ON DELETE CASCADE;


--
-- Name: world_eruption world_eruption_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_eruption
    ADD CONSTRAINT world_eruption_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.canon_event(event_id);


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
    ('20260729100002'),
    ('20260729100003'),
    ('20260729100004'),
    ('20260729100005'),
    ('20260729100006'),
    ('20260729100007'),
    ('20260730100001'),
    ('20260805100001'),
    ('20260805100002'),
    ('20260805100003'),
    ('20260807100001'),
    ('20260807100002'),
    ('20260807100003'),
    ('20260807100004'),
    ('20260807100005'),
    ('20260808090001'),
    ('20260808100001'),
    ('20260808100002'),
    ('20260808100003'),
    ('20260808100004'),
    ('20260808100005'),
    ('20260808100006'),
    ('20260809090001'),
    ('20260809090002'),
    ('20260809090003'),
    ('20260809090004'),
    ('20260809090005'),
    ('20260809090006'),
    ('20260809090007'),
    ('20260809090008'),
    ('20260809090009'),
    ('20260813142100'),
    ('20260813160000'),
    ('20260814140000'),
    ('20260814170000'),
    ('20260815150000'),
    ('20260815150001'),
    ('20260820200000'),
    ('20260821090000'),
    ('20260821120000'),
    ('20260825120000'),
    ('20260825130000'),
    ('20260825140000'),
    ('20260828090000');
