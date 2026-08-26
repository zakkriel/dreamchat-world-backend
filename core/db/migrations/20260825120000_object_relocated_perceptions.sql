-- migrate:up
-- SPEC-034 — an ObjectRelocated arm in generate_perceptions.
--
-- The two existing arms (private_disclosure/Communicated and move/ActorMoved) are carried forward
-- VERBATIM, per the convention every migration touching this function follows — see
-- 20260814170000_hearing_teaches_only_spoken_names.sql: "the move/ActorMoved arm is carried forward
-- verbatim". The new arm is appended immediately before the final RETURN. Full rationale is in the
-- arm's own comment block, so it lives next to the code it explains.

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
        FOR or_who IN
          SELECT DISTINCT h FROM (
            SELECT entity_id AS h FROM event_participant
              WHERE event_id = p_event_id AND role_qualifier = 'instigator'
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
-- Restoring the previous definition means dropping the arm; the two older arms are unaffected.
-- Down is intentionally a no-op marker: this function is replaced in place by every migration that
-- touches it, so a mechanical revert would resurrect whichever body preceded THIS one, which the
-- next migration up would then clobber. Reverting SPEC-034 means writing the removal explicitly.
SELECT 1;
