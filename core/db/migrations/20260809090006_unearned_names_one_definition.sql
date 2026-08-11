-- migrate:up

-- One definition of "unearned", shared by the seam and the belt.
--
-- ── TWO DEFECTS, ONE CAUSE ──────────────────────────────────────────────────────────────────────
-- Migration 20260809090005 wrote the predicate twice — once in fn_viewer_text's loop, once in the Go
-- belt's query. Live on Railway within a minute of deploying, both problems showed up:
--
-- 1. A FALSE POSITIVE. The narrator wrote "the ballast crate" and the belt rejected the line:
--    "segment 0 names \"ballast crate\", which this viewer has not earned". The registry calls it
--    `Ballast Crate`; Kade's label for it is `the ballast crate`. They differ by an article and a
--    capital. Nothing is hidden by saying it, and the narrator lost a correct sentence and a retry to
--    a rule that was measuring string inequality where it meant knowledge.
--
-- 2. THE LATENT ONE. Two copies of a predicate drift. If the belt ever considers a name unearned
--    that the seam does not rewrite, every beat mentioning it burns two retries and lands on the
--    scrub; if the seam rewrites what the belt permits, the fix is silently doing nothing. The belt
--    exists to check the seam, so they MUST share the definition or the check is theatre.
--
-- ── THE RULE ────────────────────────────────────────────────────────────────────────────────────
-- A canonical name is unearned when the viewer has no knowledge path to it AND his own label does not
-- already contain it. Containment is the honest test: "the ballast crate" already says "ballast
-- crate", so the name carries no knowledge he lacks — while "the muscle by the bar" says nothing of
-- "Jonas", which stays guarded. It keeps the wall for an artifact whose name IS the secret (a label
-- of "a leather book" against a registry name of "Mara's Ledger of Debts" is still a breach), so this
-- is narrower than exempting artifacts wholesale, which was the tempting shortcut.
CREATE OR REPLACE FUNCTION public.fn_unearned_names(p_world_id uuid, p_viewer uuid)
  RETURNS TABLE(canonical_name text, label text)
  LANGUAGE sql STABLE
  AS $$
  SELECT er.canonical_name, fn_display_name(p_world_id, p_viewer, er.entity_id)
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
  ORDER BY length(er.canonical_name) DESC  -- longest first: "Hooded Companion" before "Hooded"
$$;

COMMENT ON FUNCTION public.fn_unearned_names(uuid, uuid) IS
  'The canonical names a viewer has NOT earned, with the label he holds instead. The single '
  'definition behind both the perception seam (fn_viewer_text) and the API-boundary belt '
  '(NamingWall in core/api) — naming reach, RULINGS-2026-07-23 §3.';

-- fn_viewer_text now reads that definition instead of restating it.
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
    -- \m..\M are word boundaries so a name is never rewritten inside a longer word; the name is
    -- regexp-escaped because it is world data (an actor called "St. John" would otherwise be a
    -- pattern); 'gi' because prose capitalises freely at a sentence start.
    outtxt := regexp_replace(outtxt,
                             '\m' || regexp_replace(r.canonical_name, '([.^$*+?()\[\]{}|\\-])', '\\\1', 'g') || '\M',
                             r.label, 'gi');
  END LOOP;

  RETURN outtxt;
END $$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_unearned_names(uuid, uuid);
