-- migrate:up

-- Perception content is rendered in the HOLDER's own vocabulary.
--
-- ── THE BREACH ──────────────────────────────────────────────────────────────────────────────────
-- Reported by the founder from live play: Kade's beat returned narration reading "Mara is behind the
-- bar now, Jonas planted between her and the room". Kade has never earned Jonas's name — he has only
-- ever perceived "the muscle by the bar". A canonical name crossed into player-facing prose, which is
-- the product's first law (B-1, I-3: no canon row, and no canon FACT, reaches the player except as a
-- perception-bound projection; naming reach, RULINGS-2026-07-23 §3).
--
-- The narrator was not the culprit and no prompt wording would have saved it. Measured on the live
-- database, Kade's OWN perception rows already contained the name:
--
--   tick 51  "Jonas shifts his weight off the bar, angling between the door and Mara"
--   tick 54  "Jonas pushes off the bar and steps between Kade and Mara, blocking the way."
--   tick 55  "Jonas lunges to snatch the note from the stranger's hands before it can be opened"
--
-- The narrator rendered, faithfully, what the wall handed it.
--
-- ── THE PATH ────────────────────────────────────────────────────────────────────────────────────
-- World-side seats (resolve, world_actor, cognition) read `fn_world_slice` and truth-side fact
-- sheets, both of which carry `entity_registry.canonical_name`. They are LICENSED to: they are the
-- referee and the world's own minds, and they must reason about who someone actually is. Their text
-- then becomes `canon_event.summary` / the ruling's `truth_text` / `appear_txt`.
--
-- The fan-out was where it broke. Both writers copied ONE string into EVERY holder's row:
--
--   fn_apply_event        content := ev.summary                              -- same text, all holders
--   fn_apply_ruled_event  content := COALESCE(var_text, appear_txt, truth_text)
--
-- Canon is viewer-agnostic and true; perception is per-holder and bounded. Copying the first into the
-- second verbatim silently equates them. `receiver_variants` existed to let a ruling differentiate,
-- but it is optional, and everything that omits it falls through to a shared string.
--
-- Because perception_record feeds the compendium and timeline projections as well as the narrate
-- prompt, one hole leaked into every player-facing surface at once. That is also why the fix belongs
-- HERE and not in the narrate prompt: fixing the seam fixes all of them.
--
-- ── THE FIX ─────────────────────────────────────────────────────────────────────────────────────
-- fn_viewer_text() rewrites a canonical name into the label the holder has actually earned, per
-- holder, at the moment the perception is written. What the holder perceived is what gets stored.
--
-- Deliberately NOT exempting speech. A `Communicated` summary naming a third party is the tempting
-- exception ("he heard the name said aloud"), and it is a trap: hearing a name should CREATE
-- name-knowledge — a perception row that teaches it, after which fn_perceived_name returns it and
-- this function stops rewriting anything. Silently passing the raw name through instead grants the
-- knowledge without recording it, which is the same breach with a nicer story. Learning-by-earshot is
-- a knowledge-acquisition feature; it is written up as SPEC-033, not smuggled in as an exemption.
CREATE OR REPLACE FUNCTION public.fn_viewer_text(p_world_id uuid, p_holder uuid, p_text text)
  RETURNS text
  LANGUAGE plpgsql STABLE
  AS $$
DECLARE
  r      record;
  outtxt text := p_text;
BEGIN
  IF p_text IS NULL OR p_holder IS NULL THEN
    RETURN p_text;
  END IF;

  -- Longest name first: "Hooded Companion" must be rewritten before "Hooded" can match inside it and
  -- leave a mangled remainder.
  FOR r IN
    SELECT er.canonical_name AS canon,
           fn_display_name(p_world_id, p_holder, er.entity_id) AS label
    FROM entity_registry er
    WHERE er.world_id = p_world_id
      AND er.canonical_name IS NOT NULL
      AND er.canonical_name <> ''
      -- A holder always knows who HE is. Without this, Mara's own memory of an event reads "Jonas
      -- warns the stranger away from the keeper" — her own name replaced by the descriptor strangers
      -- use for her, which is not a perception, it is amnesia. (fn_perceived_name has no self→self
      -- row for seeded actors, so the general rule below would otherwise catch every holder in his
      -- own text.) For the player this also produces better prose: his own name survives into the
      -- narrate payload, where the YOU ARE block binds it to "you".
      AND er.entity_id <> p_holder
      -- Nothing to do when the holder has earned the name, and nothing SAFE to do when the world
      -- offers no other label (fn_display_name falls back to canonical): rewriting it to a
      -- placeholder would invent a perception. The Go-side belt reports that case rather than hiding
      -- it, so a cast member seeded without a descriptor is a loud data defect, not a silent leak.
      AND fn_display_name(p_world_id, p_holder, er.entity_id) IS DISTINCT FROM er.canonical_name
    ORDER BY length(er.canonical_name) DESC
  LOOP
    -- \m..\M are word boundaries: a name must not be rewritten inside a longer word. The name is
    -- regexp-escaped because it is world data — an actor called "St. John" would otherwise be a
    -- pattern. 'gi' because prose capitalises freely at a sentence start.
    outtxt := regexp_replace(outtxt,
                             '\m' || regexp_replace(r.canon, '([.^$*+?()\[\]{}|\\-])', '\\\1', 'g') || '\M',
                             r.label, 'gi');
  END LOOP;

  RETURN outtxt;
END $$;

COMMENT ON FUNCTION public.fn_viewer_text(uuid, uuid, text) IS
  'Rewrites canonical names in perception text into the labels the holder has earned (naming reach, '
  'RULINGS-2026-07-23 §3). Identity for a holder who has earned every name, and for entities the '
  'world can only name canonically.';

-- ── The privilege the fan-out now needs ─────────────────────────────────────────────────────────
-- generate_perceptions is SECURITY DEFINER owned by `maintainer`, which held INSERT on
-- perception_subject but not SELECT — it only ever wrote about-ness rows, never read them. Rendering
-- a holder's vocabulary means asking what that holder has perceived (fn_perceived_name reads
-- perception_subject), so without this the fan-out fails with "permission denied" and every beat
-- that commits an event dies. Found by the pgTAP test below, which is exactly what it is for.
-- SELECT only: the writer still has no business updating or deleting about-ness.
GRANT SELECT ON TABLE public.perception_subject TO maintainer;

-- ── The two fan-out writers now render per holder ───────────────────────────────────────────────
-- Both are reproduced verbatim from schema.sql (CREATE OR REPLACE cannot patch a body); the ONLY
-- edits are the content expressions, each wrapped in fn_viewer_text with that row's own holder.

CREATE OR REPLACE FUNCTION public.generate_perceptions(p_event_id uuid) RETURNS integer
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
      VALUES (ev.world_id, spk, p_event_id, fn_viewer_text(ev.world_id, spk, ev.summary), 'shared',
              ev.in_world_tick, ev.in_world_tick)
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
      VALUES (ev.world_id, lst, p_event_id, fn_viewer_text(ev.world_id, lst, ev.summary), 'told',
              ev.in_world_tick, ev.in_world_tick)
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
        VALUES (ev.world_id, mover, p_event_id, fn_viewer_text(ev.world_id, mover, ev.summary),
                'direct', ev.in_world_tick, ev.in_world_tick)
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
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, ev_type, truth_text,
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

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


-- ── The already-written rows are repaired ───────────────────────────────────────────────────────
-- perception_record is not canon. Canon (`canon_event.summary`) is immutable and stays exactly as
-- written — the referee's true account is untouched, and I-1 replay reads canon, not this table.
-- These rows are the perception PROJECTION, and five of them on the live world currently assert that
-- Kade knows a name he has never been told. Leaving them would mean the founder's next look-around
-- re-leaks out of storage, and every compendium page keeps serving the breach. A projection that is
-- wrong gets rebuilt.
UPDATE perception_record pr
   SET content = fn_viewer_text(pr.world_id, pr.holder_id, pr.content)
 WHERE fn_viewer_text(pr.world_id, pr.holder_id, pr.content) IS DISTINCT FROM pr.content;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_viewer_text(uuid, uuid, text);
-- The two writers are intentionally NOT reverted: restoring them would reintroduce the naming-wall
-- breach, and the repaired perception rows cannot be un-repaired (the canonical names they used to
-- carry survive in canon, which is where they belong).
