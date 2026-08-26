-- migrate:up
--
-- SPEC-035 amendment - a malformed `witnesses` field is refused, not silently dropped.
--
-- Governed-by: SPEC-035 (the field this repairs), B-2, D-1 (the Core owns the mechanism).
--
-- -- HOW THIS WAS FOUND -----------------------------------------------------------------------
--
-- By reviewing SPEC-035 one commit after shipping it, and asking the one question its own four
-- mutants never asked: not "what if the code is wrong" but "what if the INPUT is wrong".
--
--     apply_event(..., 'witnesses', '<a bare uuid string>')  ->  halt_reason: committed
--                                                                witness rows:      0
--                                                                perceptions:       0
--
-- Committed. Silent. No halt_reason. That is the same defect SPEC-035 was filed to fix, quoted
-- from its own migration header: "the caller could ALREADY name witnesses and apply_event dropped
-- the field without a word and without a halt_reason."
--
-- The shape is not exotic. `Communicated`'s recipient field is `listener_id` - a BARE STRING in the
-- same attempt payload. A caller reasoning from the nearest sibling writes `witnesses: "<uuid>"`,
-- and every branch of the SPEC-035 gate keys on `jsonb_typeof(...) = 'array'`, so it falls through
-- all of them.
--
-- -- THE RULE ---------------------------------------------------------------------------------
--
--   absent                  -> fine. Nobody was named. No witnesses, no gate.
--   null                    -> fine. Explicitly nobody. Same as absent.
--   an array                -> validated per element, as before.
--   anything else present   -> gate_reject.
--
-- Coercing a lone string into a one-element array was considered and REJECTED. Coercion hides the
-- caller's bug, and hiding the caller's bug is the entire failure being repaired here. The engine
-- blocks impossibilities; it does not guess intent.
--
-- Function body carried forward verbatim per this repo's migration convention.

CREATE OR REPLACE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
AS $function$
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
END $function$;

-- migrate:down
-- Reverting would restore a silent drop on malformed input. Roll forward.
