-- migrate:up

-- Spoken words become canon, so a quote can be verified.
--
-- ── WHAT WAS BROKEN ─────────────────────────────────────────────────────────────────────────────
-- narration/2 split a speech segment's verbatim `quote` from its staging so the frontend could format
-- dialogue. Measured live the same hour: EVERY speech segment the narrator produced was refused —
--
--   quote "Still here, then." is not verbatim — it does not appear in speaker …a2's spoken words
--   quote "You're not in my way. You're at my bar." … does not appear in speaker …a3's spoken words
--
-- and the refusals were correct. `canon_event.payload` was `{}` for every Communicated event ever
-- committed, and the summary is the referee's ACCOUNT of the utterance ("Mara breaks the silence with
-- a dry remark"), never the words. The world recorded that someone spoke and never what they said, so
-- no quote could be verified and kind:"speech" was unreachable by construction.
--
-- The seats were already carrying the words: `content` is in beat_chain/2, ruling/2 and
-- npc_attempts/1, and resolve.txt's own example puts real dialogue there. Both commit paths simply
-- dropped it — neither wrote `payload` at all.
--
-- Worth naming plainly, because it is the uncomfortable half: under narration/1 the same belt
-- substring-matched the segment TEXT, so a speech segment passed by quoting the paraphrase ITSELF as
-- dialogue. The player was shown `Jonas: "Jonas addresses the stranger, voice low and flat"` as
-- speech. The split did not break dialogue; it stopped shipping invented dialogue, and this migration
-- supplies the real thing.
--
-- ── THE THREE LINKS ─────────────────────────────────────────────────────────────────────────────
--   1. apply_event / apply_ruled_event persist `content` as canon: payload.spoken.
--   2. generate_perceptions carries the words into the perception of everyone who heard them — the
--      narrator can only quote what is in its payload, so without this the model keeps inventing.
--   3. (Go) speechTexts backs the verbatim belt with payload.spoken instead of the perception line,
--      so a paraphrase can no longer pass by being a substring of its own account.
--
-- Nothing here loosens the wall: perception content is still rendered per holder through
-- fn_viewer_text, and a name SAID ALOUD now teaches that name (SPEC-033) because learning reads the
-- utterance including its words — which is precisely the case the founder's ruling was about.

CREATE OR REPLACE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false) RETURNS jsonb
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

CREATE OR REPLACE FUNCTION public.apply_ruled_event(p_world_id uuid, p_ruled jsonb, p_tick bigint, p_seq integer, p_origin text) RETURNS jsonb
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

CREATE OR REPLACE FUNCTION public.generate_perceptions(p_event_id uuid) RETURNS integer
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev   canon_event;
  n    integer := 0;
  spk  uuid;
  lst  uuid;
  pid  uuid;
  -- said = the utterance as a holder PERCEIVES it: the referee's account plus the words themselves.
  -- Without the words in the perception line the narrator has nothing verbatim to quote, so it invents
  -- dialogue and the belt refuses it — which is exactly the deadlock that made kind:"speech"
  -- unreachable. The words are canon (payload.spoken); this is how they reach the person who heard them.
  said text;
BEGIN
  SELECT * INTO ev FROM canon_event WHERE event_id = p_event_id AND status = 'accepted';
  IF NOT FOUND THEN RETURN 0; END IF;

  IF ev.event_type IN ('private_disclosure', 'Communicated') THEN
    said := ev.summary;
    IF NULLIF(TRIM(COALESCE(ev.payload->>'spoken','')),'') IS NOT NULL
       -- ...unless the account ALREADY is the words. The legacy `say` step commits with summary =
       -- content, so appending would render 'I saw the note — "I saw the note"'. Nothing is gained by
       -- quoting a line back to itself, and the duplication would reach the player's own transcript.
       AND position(TRIM(ev.payload->>'spoken') IN ev.summary) = 0 THEN
      -- Quoted so a reader — human or model — can see where the account ends and the words begin.
      said := ev.summary || ' — "' || TRIM(ev.payload->>'spoken') || '"';
    END IF;
    -- speaker → 'shared'; each listener → 'told' (B-7). Recipients = the addressed listeners
    -- (thin slice; co-present overhearers defer with the broader vocabulary, §3).
    SELECT entity_id INTO spk FROM event_participant
      WHERE event_id = p_event_id AND role_qualifier = 'speaker' LIMIT 1;
    IF spk IS NOT NULL THEN
      -- SPEC-033: a name in what was said is earned by whoever heard it said.
      INSERT INTO name_knowledge (world_id, holder_id, entity_id, name, learned_tick, source_event_id)
      SELECT ev.world_id, spk, t.entity_id, t.canonical_name, ev.in_world_tick, p_event_id
        FROM fn_names_in_text(ev.world_id, said) t
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
      -- SPEC-033, the reported case: Mara says "Jonas" where Kade can hear it, and Kade learns it.
      INSERT INTO name_knowledge (world_id, holder_id, entity_id, name, learned_tick, source_event_id)
      SELECT ev.world_id, lst, t.entity_id, t.canonical_name, ev.in_world_tick, p_event_id
        FROM fn_names_in_text(ev.world_id, said) t
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

  RETURN n;
END $$;

-- migrate:down

-- Not reverted: dropping payload.spoken would make every quote unverifiable again and re-break
-- kind:"speech". The functions are replaced forward, never backward.
