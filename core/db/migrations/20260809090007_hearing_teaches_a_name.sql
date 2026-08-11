-- migrate:up

-- SPEC-033 — hearing teaches, if present.
--
-- Founder ruling: a name spoken in the viewer's perceived scene becomes earned. Direct address and
-- introduction included; overhearing across the room counts only when the world says the viewer could
-- hear it. Until now there was NO in-play way to learn a name at all — `fn_perceived_name` read only
-- perception rows sourced from `world_genesis`, so every name a world did not seed was unlearnable,
-- and migration 20260809090005 (the naming wall) therefore rewrote a spoken name out of the listener's
-- own perception without ever giving him the chance to acquire it.
--
-- ── WHERE "COULD HEAR" IS DECIDED ───────────────────────────────────────────────────────────────
-- Not re-decided here. `generate_perceptions` already answers it: for a Communicated /
-- private_disclosure event it mints a perception row for the speaker ('shared') and for each ADDRESSED
-- listener ('told'), and co-present overhearers are a documented deferral (§3, the broader vocabulary).
-- So "the viewer could hear it" is exactly "the viewer got a perception row from this utterance", and
-- learning is bolted to that same decision. When the overhearer rule lands, hearing-teaches follows it
-- for free — one seam, and the wall and the learning path cannot drift apart, because a name is taught
-- to precisely the holders the fan-out is already writing to.
--
-- ── WHY A TABLE AND NOT A PERCEPTION ROW ────────────────────────────────────────────────────────
-- The tempting shape is "write a perception row whose content is the name", mirroring genesis. It is
-- wrong twice over:
--
--   1. It needs a marker anyway. Without one, `fn_perceived_name` cannot tell "this row IS his name"
--      from "this row is prose that happens to be about him" — and would start returning whole
--      sentences as people's names.
--   2. Six lenses (schema.sql:1146, 1243, 1331, 1361, 1383, and the fn_actor_page tripwire at 1842)
--      exclude genesis-sourced rows from prose precisely because name substrate is not content. A new
--      marker means editing all six or watching the bare string "Jonas" surface as a timeline entry
--      and a dossier line.
--
-- A dedicated table is the marker, needs no lens edits, and keeps perception_record's semantics
-- exactly as they are. Name-knowledge is knowledge, so it lives in the perception layer, not canon.
CREATE TABLE IF NOT EXISTS public.name_knowledge (
  world_id        uuid    NOT NULL,
  holder_id       uuid    NOT NULL,
  entity_id       uuid    NOT NULL,
  name            text    NOT NULL,
  learned_tick    bigint  NOT NULL,
  source_event_id uuid    NOT NULL REFERENCES public.canon_event(event_id),
  created_at      timestamp with time zone DEFAULT now() NOT NULL,
  -- One name per (holder, subject): the FIRST hearing wins. A later utterance using the same name is
  -- a no-op, and a later utterance using a DIFFERENT name for the same person is the alias question,
  -- which nothing in the thin slice can answer honestly — it stays out rather than being guessed at.
  PRIMARY KEY (world_id, holder_id, entity_id)
);

COMMENT ON TABLE public.name_knowledge IS
  'Names a holder has learned IN PLAY (SPEC-033). Genesis-seeded name-knowledge stays in '
  'perception_record; fn_perceived_name reads both. Perception layer, never canon.';

-- generate_perceptions is SECURITY DEFINER owned by `maintainer`, and every naming read
-- (fn_perceived_name → fn_display_name → fn_unearned_names → fn_viewer_text) now touches this table.
GRANT SELECT, INSERT ON TABLE public.name_knowledge TO maintainer;

-- ── Escaping, once ──────────────────────────────────────────────────────────────────────────────
-- Canonical names are world data: an actor called "St. John" is a regex if pasted into a pattern
-- unescaped. 20260809090005 inlined this; two callers now need it, so it gets a name.
CREATE OR REPLACE FUNCTION public.fn_regexp_quote(p_text text) RETURNS text
  LANGUAGE sql IMMUTABLE STRICT
  AS $$ SELECT regexp_replace(p_text, '([.^$*+?()\[\]{}|\\-])', '\\\1', 'g') $$;

-- The names actually present in a piece of text, word-bounded and case-insensitive — the same match
-- rule fn_viewer_text rewrites with, so the wall and the learning path agree about what "the text says
-- this name" means.
CREATE OR REPLACE FUNCTION public.fn_names_in_text(p_world_id uuid, p_text text)
  RETURNS TABLE(entity_id uuid, canonical_name text)
  LANGUAGE sql STABLE
  AS $$
  SELECT er.entity_id, er.canonical_name
  FROM entity_registry er
  WHERE er.world_id = p_world_id
    AND er.canonical_name IS NOT NULL
    AND er.canonical_name <> ''
    AND p_text ~* ('\m' || fn_regexp_quote(er.canonical_name) || '\M')
$$;

-- ── fn_perceived_name learns to read both sources ───────────────────────────────────────────────
-- Genesis-seeded knowledge and in-play learning are the same kind of fact, so they are unioned and the
-- EARLIEST wins — which keeps the seeded name authoritative for a viewer who was born knowing it.
CREATE OR REPLACE FUNCTION public.fn_perceived_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
  RETURNS text
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

-- ── The fan-out teaches, then renders ───────────────────────────────────────────────────────────
-- ORDER IS THE WHOLE DESIGN. Name-knowledge is written BEFORE the holder's content is rendered, so
-- fn_viewer_text — reading fn_perceived_name, which now reads name_knowledge — sees a name the holder
-- has just earned and leaves it standing. No exemption for speech, no second code path, no flag: the
-- wall keeps its single rule ("rewrite what he has not earned") and hearing simply changes what he has
-- earned, one statement earlier.
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
      -- SPEC-033: a name in what was said is earned by whoever heard it said.
      INSERT INTO name_knowledge (world_id, holder_id, entity_id, name, learned_tick, source_event_id)
      SELECT ev.world_id, spk, t.entity_id, t.canonical_name, ev.in_world_tick, p_event_id
        FROM fn_names_in_text(ev.world_id, ev.summary) t
       WHERE t.entity_id <> spk
      ON CONFLICT DO NOTHING;

      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (ev.world_id, spk, p_event_id, fn_viewer_text(ev.world_id, spk, ev.summary), 'shared',
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
        FROM fn_names_in_text(ev.world_id, ev.summary) t
       WHERE t.entity_id <> lst
      ON CONFLICT DO NOTHING;

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

-- fn_viewer_text reuses the shared escaping rather than keeping its own copy.
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

  FOR r IN SELECT * FROM fn_unearned_names(p_world_id, p_holder) LOOP
    outtxt := regexp_replace(outtxt,
                             '\m' || fn_regexp_quote(r.canonical_name) || '\M',
                             r.label, 'gi');
  END LOOP;

  RETURN outtxt;
END $$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_names_in_text(uuid, text);
DROP TABLE IF EXISTS public.name_knowledge;
DROP FUNCTION IF EXISTS public.fn_regexp_quote(text);
