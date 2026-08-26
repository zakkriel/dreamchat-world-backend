-- migrate:up
--
-- SPEC-035 - an ObjectRelocated event records WHO SAW IT.
--
-- Governed-by: B-2 (the event names its participants), I-2 (provenance), ADR-026 (replay reproduces
-- projection state, not perceptions), SPEC-034 (the holder arm this extends), D-1 (the Core owns the
-- mechanism). Founder ruling, 2026-08-25: "holders and co-present as long as they can see it - just
-- because they were there doesn't mean they saw it."
--
-- -- THE BREACH, measured on the seeded world before this was written --------------------------
--
-- Kade hands the Sealed Note to Mara. Jonas is standing in the same room. Passing
-- `witnesses: [jonas]` to apply_event:
--
--     1. WHO SAW IT, per the event:      instigator          <- Jonas is nowhere
--     2. Jonas's perceptions of it:      0
--     3. witnesses[] kept in payload?    {}                  <- SILENTLY DISCARDED
--     4. Mara's perceptions (SPEC-034):  1                   <- the holder arm works
--
-- Row 3 is the defect. This was not a missing feature request - the caller could ALREADY name
-- witnesses, and apply_event dropped the field without a word or a halt_reason. The question "who
-- watched this handover" had no answer anywhere in the database, and could not be given one later.
--
-- -- WHY IT LANDS NOW AND NOT LATER -----------------------------------------------------------
--
-- A perception rule can be widened whenever we like. An event PAYLOAD cannot be back-filled:
-- replay_0A() asserts it reproduces domain-equivalent *projection* state, and perceptions are not
-- projections - they are never regenerated on replay (ADR-026). So every handover committed before
-- this migration is permanently unwitnessable, and every one committed after is recoverable. That
-- asymmetry is the whole argument for doing it in this round instead of the next.
--
-- -- SHAPE ------------------------------------------------------------------------------------
--
-- The event names its witnesses, exactly as 'Communicated' names its listeners (B-2), and
-- generate_perceptions reads them back by role_qualifier. No new column, no payload field, no new
-- vocabulary: `event_participant.role_qualifier = 'witness'` alongside the existing 'instigator',
-- 'speaker', 'listener', 'subject'.
--
-- apply_event gains two things:
--   * a GATE - a named witness who was not co-present is a gate_reject, byte-for-byte the shape of
--     the existing listener co-presence gate. Co-presence is necessary, not sufficient: the caller
--     supplies sufficiency by naming them, the engine supplies necessity by refusing the impossible.
--   * the RECORD - 'witness' participant rows, holders excluded (they already perceive as parties).
--
-- generate_perceptions changes by one clause: witnesses join the holders in the perceiver set, with
-- the same 'direct' epistemic type and the same subjects, because they are in the same epistemic
-- position. Concealment, when it exists, shortens the list the caller passes and needs nothing here.
--
-- Both function bodies are carried forward verbatim per this repo's migration convention; the new
-- regions are commented inline.

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

CREATE OR REPLACE FUNCTION public.generate_perceptions(p_event_id uuid)
 RETURNS integer
 LANGUAGE plpgsql
 SECURITY DEFINER
AS $function$
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
END $function$;

-- migrate:down
-- Down is a no-op on purpose. Dropping the 'witness' rows would destroy recorded observations - the
-- exact loss this migration exists to prevent - and reverting the functions while those rows remain
-- would leave witnesses recorded and unperceived, which is worse than either end state. Roll forward.
