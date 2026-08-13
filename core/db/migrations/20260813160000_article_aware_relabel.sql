-- migrate:up

-- The naming wall learns grammar.
--
-- Live symptom, in a telegraph the player actually read:
--
--   "The a hooded figure shifts silently, drawing a step closer to the stranger's flank."
--
-- fn_viewer_text rewrites an unearned canonical name into the label the holder holds. The registry
-- name is "Hooded Companion" and the label is "a hooded figure", so a sentence reading "The Hooded
-- Companion shifts…" becomes "The a hooded figure shifts…". The wall did its job — the name is gone —
-- and produced prose no narrator would write. It reaches the player verbatim, and it also reaches the
-- transcript, where it is frozen forever as the memory of that moment.
--
-- The honest fix is not to strip stray articles afterwards with a cleanup regex — that would paper
-- over whatever else the substitution mangles. It is to make the substitution ARTICLE-AWARE: an
-- article already standing in front of the name is the sentence's article, and a label that carries
-- its own must not add a second one.
--
--   "The Hooded Companion left"  ->  "The hooded figure left"     (sentence's article kept, case kept)
--   "the Hooded Companion left"  ->  "the hooded figure left"
--   "Hooded Companion left"      ->  "a hooded figure left"       (no article present: label intact)
--
-- Two passes, in this order, because the first is the special case of the second: names preceded by an
-- article are rewritten with the label's own article stripped, then whatever remains is rewritten
-- whole. A label with no leading article ("Mara", "the keeper") is unaffected by either rule.
CREATE OR REPLACE FUNCTION public.fn_viewer_text(p_world_id uuid, p_holder uuid, p_text text)
  RETURNS text
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

-- migrate:down

-- Not reverted: the previous body produced "The a hooded figure" in player-facing prose.
