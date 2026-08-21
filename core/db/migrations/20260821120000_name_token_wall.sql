-- migrate:up

-- The naming wall guards a person's name TOKENS, not just the whole registry string.
--
-- ── THE BREACH (Railway, live play, 2026-08-20, world "The Ironmoor Reach") ─────────────────────
-- Genesis stored slug join-keys as canonical names — silas_holton, emmett_vale — so this function
-- listed strings no model ever writes. The cognition seats humanised the slugs in their own prose
-- ("Silas smiles, slow and cold"), apply_ruled_event committed that prose verbatim as the player's
-- perception content, and the narrator handed the player "toward Emmett" and "Silas's voice" with
-- the belt watching: Violations() matched "emmett_vale" against a sentence that says "Emmett" and
-- found nothing. The wall was loaded, correct by its own rule, and useless.
--
-- Genesis is fixed at its source in the same change (worldgenesis.go refuses identifier-shaped
-- people's names), but source discipline was the wall's ORIGINAL design and it failed the same way
-- prompt discipline did — so the belt learns the lesson mechanically: prose speaks in words, and a
-- name is only hidden when every distinctive WORD of it is hidden.
--
-- ── THE RULE ────────────────────────────────────────────────────────────────────────────────────
-- For every unearned ACTOR name, each word of the name is guarded like the name itself, when it is
-- worth guarding:
--   · 3+ characters — initials and orphan letters are not names;
--   · not a name particle ("van", "von", "della"…) — those are ordinary prose everywhere else;
--   · not the whole name over again — single-word names are already guarded by their own row;
--   · not already in the label the viewer holds (the Ballast Crate rule, per word);
--   · not a word of the viewer's OWN name or label — a stranger who shares the viewer's given name
--     must not get the viewer's own name censored out of his story (amnesia, again);
--   · not a word the world already uses in LOWERCASE — a token that appears lowercase anywhere in
--     the world's own prose (canon summaries, spoken words, descriptors, place descriptions) is
--     ordinary vocabulary, not a name. This is what keeps "Hooded Woman" from censoring the word
--     "woman" out of every sentence, while "Silas" — which no referee has ever written lowercase —
--     stays guarded. A name lowercased somewhere in canon would slip this net; models capitalise
--     names essentially always, and the whole-string row still stands behind it.
-- ACTORS ONLY. People's names are made of proper nouns; place and object names are made of English
-- ("the_mantel_letter", "The Counting Room"), and guarding their words would eat the narrator's
-- vocabulary one common noun at a time. Those keep whole-string guarding, exactly as before.
--
-- Tokens DERIVE from the unearned set, so earning a name (name_knowledge) removes its tokens in the
-- same statement — no second learning path, nothing to keep in sync. Every consumer inherits the
-- rows: fn_viewer_text (the source seam), NamingWall in core/api (the belt), and test 121 holds the
-- two to the same answer.
CREATE OR REPLACE FUNCTION public.fn_unearned_names(p_world_id uuid, p_viewer uuid)
  RETURNS TABLE(canonical_name text, label text)
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

COMMENT ON FUNCTION public.fn_unearned_names(uuid, uuid) IS
  'The canonical names a viewer has NOT earned — and, for people, every distinctive word of each — '
  'with the label he holds instead. The single definition behind both the perception seam '
  '(fn_viewer_text) and the API-boundary belt (NamingWall in core/api) — naming reach, '
  'RULINGS-2026-07-23 §3; token guarding is the Ironmoor fix, 2026-08-20.';

-- migrate:down

-- The 20260809090006 body: whole-string guarding only.
CREATE OR REPLACE FUNCTION public.fn_unearned_names(p_world_id uuid, p_viewer uuid)
  RETURNS TABLE(canonical_name text, label text)
  LANGUAGE sql STABLE
  AS $$
  SELECT er.canonical_name, fn_display_name(p_world_id, p_viewer, er.entity_id)
  FROM entity_registry er
  WHERE er.world_id = p_world_id
    AND er.canonical_name IS NOT NULL
    AND er.canonical_name <> ''
    AND er.entity_id IS DISTINCT FROM p_viewer
    AND fn_display_name(p_world_id, p_viewer, er.entity_id) IS DISTINCT FROM er.canonical_name
    AND position(lower(er.canonical_name) IN lower(coalesce(fn_display_name(p_world_id, p_viewer, er.entity_id), ''))) = 0
  ORDER BY length(er.canonical_name) DESC
$$;
