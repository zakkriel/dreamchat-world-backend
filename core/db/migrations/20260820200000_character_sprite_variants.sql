-- migrate:up

-- Character sprites: an actor's picture becomes FOUR emotion variants of one identity
-- (neutral/happy/angry/sad — the closed set narration/3 tags and the frontend's sprite layer
-- renders). image_slot therefore gains a variant dimension. Every pre-existing row becomes
-- variant 'default', which is exactly what it was: the one picture an owner had.
ALTER TABLE image_slot ADD COLUMN variant text NOT NULL DEFAULT 'default';

-- The closed set, enforced where the rows live. 'default' is the single-image variant every
-- non-actor owner (world cover, place backdrop, artifact) keeps using; the four emotions are
-- actor sprites. An open column would let a typo'd variant vanish from every reader silently.
ALTER TABLE image_slot ADD CONSTRAINT image_slot_variant_check
  CHECK (variant IN ('default', 'neutral', 'happy', 'angry', 'sad'));

ALTER TABLE image_slot DROP CONSTRAINT image_slot_pkey;
ALTER TABLE image_slot ADD CONSTRAINT image_slot_pkey
  PRIMARY KEY (world_id, owner_kind, owner_id, variant);

-- fn_image_ref keeps its shape (image_ref/1) and its callers (scene participants, actor pages,
-- world covers, carrying is untouched) — only its row selection learns about variants: the
-- legacy 'default' picture wins, else the neutral sprite stands in as the owner's face. So a
-- sprite-format world still has avatars everywhere image_ref/1 is consumed, with no consumer
-- change and no version bump.
CREATE OR REPLACE FUNCTION fn_image_ref(p_world_id uuid, p_owner_kind text, p_owner_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
           'schema_version', 'image_ref/1',
           'asset_id',       s.asset_id,
           'path',           '/worlds/' || p_world_id::text || '/images/' || s.asset_id
         )
    FROM image_slot s
   WHERE s.world_id = p_world_id AND s.owner_kind = p_owner_kind AND s.owner_id = p_owner_id
     AND s.variant IN ('default', 'neutral') AND s.asset_id IS NOT NULL
   ORDER BY CASE s.variant WHEN 'default' THEN 0 ELSE 1 END
   LIMIT 1;
$$;

-- fn_sprite_set renders a participant's whole emotion set for scene_current/4, or NULL. All four
-- or nothing, on purpose: a stage that swaps three emotions and silently shows the wrong face for
-- the fourth is worse than a stage that waits — and NULL is already the ordinary "no art yet"
-- state every consumer handles (D-8).
CREATE FUNCTION fn_sprite_set(p_world_id uuid, p_owner_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN count(*) = 4 THEN
           json_object_agg(
             s.variant,
             json_build_object(
               'schema_version', 'image_ref/1',
               'asset_id',       s.asset_id,
               'path',           '/worlds/' || p_world_id::text || '/images/' || s.asset_id
             )
           )
         END
    FROM image_slot s
   WHERE s.world_id = p_world_id AND s.owner_kind = 'actor' AND s.owner_id = p_owner_id
     AND s.variant IN ('neutral', 'happy', 'angry', 'sad') AND s.asset_id IS NOT NULL;
$$;

-- transcript/2: the envelope version moves because the stored segments may now carry the optional
-- per-line `emotion` the narrate seat tags (narration/3) — the same closed set the sprite layer
-- renders. The function body is otherwise identical: segments are stored prose, returned verbatim.
CREATE OR REPLACE FUNCTION fn_transcript(p_world_id uuid, p_viewer_id uuid, p_before bigint DEFAULT NULL::bigint, p_limit integer DEFAULT 50) RETURNS jsonb
    LANGUAGE sql STABLE
    AS $$
  WITH lim AS (
    -- Bounded server-side: a client asking for a million entries gets 200. 50 when unspecified.
    SELECT LEAST(GREATEST(COALESCE(p_limit, 50), 1), 200) AS n
  ), page AS (
    SELECT te.entry_no, te.in_world_tick, te.stated, te.segments, te.halt_reason, te.journey
    FROM transcript_entry te, lim
    WHERE te.world_id = p_world_id
      AND te.viewer_id = p_viewer_id
      AND (p_before IS NULL OR te.entry_no < p_before)
    ORDER BY te.entry_no DESC
    LIMIT (SELECT n FROM lim)
  )
  SELECT jsonb_build_object(
    'schema_version', 'transcript/2',
    'world_id',       p_world_id,
    'viewer_id',      p_viewer_id,
    'entries',        COALESCE((
      SELECT jsonb_agg(jsonb_build_object(
               'entry_no',    p.entry_no,
               'tick',        p.in_world_tick,
               'stated',      p.stated,
               'halt_reason', p.halt_reason,
               'journey',     p.journey,
               'segments',    p.segments
             ) ORDER BY p.entry_no DESC)
      FROM page p), '[]'::jsonb),
    -- The oldest entry on this page is the exclusive cursor for the next one. Null when this page
    -- reached the beginning: no more story behind it.
    'next_before',    (
      SELECT CASE WHEN EXISTS (
               SELECT 1 FROM transcript_entry older
               WHERE older.world_id = p_world_id AND older.viewer_id = p_viewer_id
                 AND older.entry_no < (SELECT min(entry_no) FROM page))
             THEN (SELECT min(entry_no) FROM page) END
      FROM page LIMIT 1)
  );
$$;

COMMENT ON FUNCTION fn_transcript(p_world_id uuid, p_viewer_id uuid, p_before bigint, p_limit integer) IS 'transcript/2 — one viewer''s delivered story, newest-first, cursor-paginated on entry_no. Returns stored prose verbatim: no re-labelling, no re-derivation; segments may carry the optional narration/3 emotion tag.';

-- migrate:down

DROP FUNCTION fn_sprite_set(uuid, uuid);

-- Restore the transcript/1 envelope.
CREATE OR REPLACE FUNCTION fn_transcript(p_world_id uuid, p_viewer_id uuid, p_before bigint DEFAULT NULL::bigint, p_limit integer DEFAULT 50) RETURNS jsonb
    LANGUAGE sql STABLE
    AS $$
  WITH lim AS (
    -- Bounded server-side: a client asking for a million entries gets 200. 50 when unspecified.
    SELECT LEAST(GREATEST(COALESCE(p_limit, 50), 1), 200) AS n
  ), page AS (
    SELECT te.entry_no, te.in_world_tick, te.stated, te.segments, te.halt_reason, te.journey
    FROM transcript_entry te, lim
    WHERE te.world_id = p_world_id
      AND te.viewer_id = p_viewer_id
      AND (p_before IS NULL OR te.entry_no < p_before)
    ORDER BY te.entry_no DESC
    LIMIT (SELECT n FROM lim)
  )
  SELECT jsonb_build_object(
    'schema_version', 'transcript/1',
    'world_id',       p_world_id,
    'viewer_id',      p_viewer_id,
    'entries',        COALESCE((
      SELECT jsonb_agg(jsonb_build_object(
               'entry_no',    p.entry_no,
               'tick',        p.in_world_tick,
               'stated',      p.stated,
               'halt_reason', p.halt_reason,
               'journey',     p.journey,
               'segments',    p.segments
             ) ORDER BY p.entry_no DESC)
      FROM page p), '[]'::jsonb),
    -- The oldest entry on this page is the exclusive cursor for the next one. Null when this page
    -- reached the beginning: no more story behind it.
    'next_before',    (
      SELECT CASE WHEN EXISTS (
               SELECT 1 FROM transcript_entry older
               WHERE older.world_id = p_world_id AND older.viewer_id = p_viewer_id
                 AND older.entry_no < (SELECT min(entry_no) FROM page))
             THEN (SELECT min(entry_no) FROM page) END
      FROM page LIMIT 1)
  );
$$;

COMMENT ON FUNCTION fn_transcript(p_world_id uuid, p_viewer_id uuid, p_before bigint, p_limit integer) IS 'transcript/1 — one viewer''s delivered story, newest-first, cursor-paginated on entry_no. Returns stored prose verbatim: no re-labelling, no re-derivation.';

CREATE OR REPLACE FUNCTION fn_image_ref(p_world_id uuid, p_owner_kind text, p_owner_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN s.asset_id IS NULL THEN NULL
              ELSE json_build_object(
                     'schema_version', 'image_ref/1',
                     'asset_id',       s.asset_id,
                     'path',           '/worlds/' || p_world_id::text || '/images/' || s.asset_id
                   )
         END
    FROM image_slot s
   WHERE s.world_id = p_world_id AND s.owner_kind = p_owner_kind AND s.owner_id = p_owner_id;
$$;

DELETE FROM image_slot WHERE variant <> 'default';
ALTER TABLE image_slot DROP CONSTRAINT image_slot_pkey;
ALTER TABLE image_slot ADD CONSTRAINT image_slot_pkey PRIMARY KEY (world_id, owner_kind, owner_id);
ALTER TABLE image_slot DROP CONSTRAINT image_slot_variant_check;
ALTER TABLE image_slot DROP COLUMN variant;
