-- migrate:up

-- The persistent transcript — the viewer's lived story, kept with the world.
--
-- Today a reload loses everything ever said and narrated: the played history exists only in the
-- browser tab that watched it stream. The founder wants the full story back on load, so it has to be
-- server-side — the rendered narration IS the viewer's experience, and experience belongs with the
-- world, not with a client's memory.
--
-- ── WHAT IS STORED, AND WHY IT IS NOT A PROJECTION ──────────────────────────────────────────────
-- Every other read surface in this system is DERIVED: ask again and the lens recomputes from canon
-- and perception. The transcript cannot be, and this is the whole design:
--
--   The prose the player read is not recoverable from the world state. It was written once, by a
--   model, from a payload assembled at one instant, and it passed the belts as they stood then. Ask
--   the narrator the same question tomorrow and you get different words. So the transcript is not a
--   view of history — it IS the record. Nothing else holds it.
--
-- ── THE EPISTEMIC RULE: NEVER RETRO-LABEL ───────────────────────────────────────────────────────
-- Stored prose keeps the labels it had AT THE TIME. An entry written before Kade learned the name
-- says "the muscle by the bar" forever, even though `fn_display_name` now answers "Jonas" and every
-- live surface has moved on. That is not staleness to be fixed — it is the point. A memory of an
-- experience is itself a perception: the player genuinely did not know the name when he read that
-- line, and rewriting his past to match his present knowledge would forge the one record that proves
-- he learned anything.
--
-- Mechanically this is guaranteed by storing rendered TEXT rather than ids: there is nothing here for
-- a later render to re-resolve. No fn_display_name call, no fn_viewer_text pass, no join to
-- entity_registry — deliberately. The one thing this table must never grow is a "current label"
-- column, because the moment it exists someone will use it to "fix" the old entries.
--
-- Rows are written post-belt: exactly the segments that reached the wire, after the naming wall and
-- the player-voice belt, including a scrubbed fallback line. What the player legitimately saw, and
-- nothing he did not.
--
-- ── ORDERING ────────────────────────────────────────────────────────────────────────────────────
-- `in_world_tick` is the domain clock and is stored for display, but it is NOT an ordering key: a
-- QUERY beat advances no tick, so several entries legitimately share one. `entry_no` is a per-row
-- monotonic sequence — an operational ordering handle over an append-only log, not domain time, so
-- B-5 is untouched (nothing here is a mutable in-world clock). It doubles as the pagination cursor,
-- which a tick could not do without ties.
CREATE TABLE IF NOT EXISTS public.transcript_entry (
  entry_no      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  world_id      uuid   NOT NULL,
  -- Viewer-scoped: the transcript is one person's experience. Another viewer of the same world has a
  -- different story, and must never be served this one.
  viewer_id     uuid   NOT NULL,
  in_world_tick bigint NOT NULL,
  -- The player's own words for this beat. NULL for a beat he did not type (a continue press).
  stated        text,
  -- The delivered narration: [{speaker_id, speaker_label, kind, text}], the SAME shape the live
  -- `narration` frame carries (beatMessage), so the frontend renders history and live prose with one
  -- renderer instead of two that drift.
  segments      jsonb  NOT NULL,
  halt_reason   text,
  -- The journey block as delivered, or NULL when the beat had none — the marker the FE needs to
  -- render travel in history the way it rendered it live.
  journey       jsonb,
  -- Operational telemetry only (B-5): never an ordering key, never rendered as in-world time.
  created_at    timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE public.transcript_entry IS
  'The viewer''s lived story as DELIVERED: rendered prose, post-belt, never retro-labelled. Not a '
  'projection — the prose is unrecoverable from world state. Viewer-scoped; entry_no orders and paginates.';

-- The only access pattern: one viewer's story, newest first, paginated.
CREATE INDEX IF NOT EXISTS idx_transcript_viewer_recent
  ON public.transcript_entry (world_id, viewer_id, entry_no DESC);

GRANT SELECT, INSERT ON TABLE public.transcript_entry TO maintainer;

-- ── The read lens (ADR-P017: the lens lives in the SQL; the Go handler stays a thin reader) ──────
-- Newest-first with a `before` cursor. `next_before` is the cursor for the NEXT (older) page, and is
-- null when the caller has reached the beginning of the story — the frontend pages until it is null
-- rather than guessing from a short page.
CREATE OR REPLACE FUNCTION public.fn_transcript(p_world_id uuid, p_viewer_id uuid,
                                                p_before bigint DEFAULT NULL,
                                                p_limit  integer DEFAULT 50)
  RETURNS jsonb
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

COMMENT ON FUNCTION public.fn_transcript(uuid, uuid, bigint, integer) IS
  'transcript/1 — one viewer''s delivered story, newest-first, cursor-paginated on entry_no. Returns '
  'stored prose verbatim: no re-labelling, no re-derivation.';

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_transcript(uuid, uuid, bigint, integer);
DROP TABLE IF EXISTS public.transcript_entry;
