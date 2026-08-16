-- migrate:up

-- SPEC-033, corrected at the source: hearing teaches only what was SPOKEN.
--
-- ── THE BREACH ──────────────────────────────────────────────────────────────────────────────────
-- Founder-reported in live play: a speaker label read "Jonas" to a player who had never been told the
-- name, and no line of dialogue in the transcript ever said it. Same character, same name, and the
-- same class of leak the naming wall itself was built for (20260809090005) — arriving through a door
-- the wall does not watch.
--
-- generate_perceptions (20260809090007, amended 20260809090009) taught from `said`:
--
--     said := ev.summary [|| ' — "' || payload.spoken || '"'];
--     INSERT INTO name_knowledge ... FROM fn_names_in_text(ev.world_id, said)
--
-- `said` is the referee's ACCOUNT of the event plus the words. The account is bookkeeping prose and it
-- names its participants CANONICALLY, because canon is where canonical names live. So every
-- Communicated event taught every listener the canonical name of everyone the account mentioned —
-- whether or not a single syllable of it was uttered. Two rows from the seeded world, both real:
--
--   holder Mara  learned "Kade"        from  "Kade nods to Mara across the bar"
--   holder Kade  learned "Cellar Hatch" from  "a commotion erupts from the cellar hatch"
--
-- The first is a NOD — a silent gesture that taught a name. The second never contained the name at
-- all: fn_names_in_text matched case-INSENSITIVELY, so the common noun "the cellar hatch" was read as
-- the proper name "Cellar Hatch". On the seeded Drowned Lantern one ordinary sentence taught four:
-- Alley, Cellar, Cellar Hatch, the bar.
--
-- It then compounds. Once a name is in name_knowledge, fn_perceived_name returns it, fn_display_name
-- returns it, and fn_unearned_names DROPS it from the unearned set (20260809090006: a name whose
-- display already equals it is not protectable). So the wall stops rewriting that name in every
-- channel at once — narration prose, and the speaker_label the founder actually saw, which is read
-- straight from fn_display_name and has no belt of its own.
--
-- ── THE FIX ─────────────────────────────────────────────────────────────────────────────────────
-- Teach from payload.spoken — the utterance itself, canon since 20260809090009 and already the
-- verbatim belt's source of truth (speechTexts). The referee's account keeps its job (it is what the
-- holder PERCEIVES, so `said` still renders the perception line); it simply stops conferring
-- knowledge. "A name spoken in the viewer's perceived scene becomes earned" is the ruling, and
-- payload.spoken is the only column that holds what was spoken.
--
-- An event with no spoken words teaches nothing. A nod is not an introduction.

-- ── Matching, made strict ───────────────────────────────────────────────────────────────────────
-- fn_names_in_text is read by exactly one caller: the teach below. Its old 'gi' match claimed parity
-- with fn_viewer_text, but the two want OPPOSITE strictness, and both directions fail safe only when
-- they differ:
--
--   REWRITING (fn_viewer_text) stays case-INSENSITIVE — prose capitalises freely at a sentence start,
--   and catching one casing too many only ever hides a name that was already unearned.
--   TEACHING (here) becomes case-SENSITIVE — a name is earned when the words carry the NAME, not when
--   they happen to contain the same letters as a common noun. Over-teaching is a wall breach;
--   under-teaching leaves a player calling someone "the muscle by the bar" one beat longer.
CREATE OR REPLACE FUNCTION public.fn_names_in_text(p_world_id uuid, p_text text)
  RETURNS TABLE(entity_id uuid, canonical_name text)
  LANGUAGE sql STABLE
  AS $$
  SELECT er.entity_id, er.canonical_name
  FROM entity_registry er
  WHERE er.world_id = p_world_id
    AND er.canonical_name IS NOT NULL
    AND er.canonical_name <> ''
    AND p_text ~ ('\m' || fn_regexp_quote(er.canonical_name) || '\M')
$$;

COMMENT ON FUNCTION public.fn_names_in_text(uuid, text) IS
  'The canonical names a piece of text SPEAKS, word-bounded and case-SENSITIVE. Read only by the '
  'hearing-teaches path (generate_perceptions), which must not grant name-knowledge because a '
  'sentence contained a common noun spelled like a proper name (SPEC-033).';

-- ── The fan-out: teach from the words, render the account ────────────────────────────────────────
-- Body is 20260809090009's, with ONE change: the two name_knowledge inserts read `spoken`, not `said`.
-- Everything else — the `said` assembly, both perception_record writes, perception_subject fan-out,
-- and the move/ActorMoved arm — is carried forward verbatim.
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

  RETURN n;
END $$;

-- ── Repairing what was already taught ───────────────────────────────────────────────────────────
-- The forward fix cannot un-teach a name: name_knowledge rows persist, and the founder's "Jonas" is
-- one of them. This deletes exactly the rows the corrected rule could never have written — a row
-- whose name does not appear, case-sensitively and word-bounded, in its OWN source event's spoken
-- words. It is not a guess about which knowledge is legitimate: a row that fails this test records a
-- name the world never said out loud, so the holder never earned it.
--
-- Genesis-seeded name-knowledge is untouched: that lives in perception_record, not this table.
DELETE FROM public.name_knowledge nk
USING public.canon_event ce
WHERE ce.event_id = nk.source_event_id
  AND NOT EXISTS (
    SELECT 1
    FROM fn_names_in_text(nk.world_id, NULLIF(TRIM(COALESCE(ce.payload->>'spoken','')),'')) t
    WHERE t.entity_id = nk.entity_id
      AND t.canonical_name = nk.name
  );

-- migrate:down

-- Not reverted. Down would restore a path that teaches canonical names from bookkeeping prose, which
-- is a naming-reach breach (RULINGS-2026-07-23 §3, B-1/I-3) — the functions are replaced forward.
