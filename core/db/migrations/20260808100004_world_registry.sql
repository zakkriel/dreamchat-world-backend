-- migrate:up

-- SPEC-028 — the world registry. Until now `world_id` was a bare uuid on twenty-seven tables and
-- nothing anywhere said which worlds EXIST, what they are called, or who the player is in each. The
-- frontend could make a world id flow through routing but could never let anyone CHOOSE a world,
-- because no endpoint could answer "which worlds are there".
--
-- A world row is a DIRECTORY ENTRY, not canon: an id, a name, a look, and who you are when you play
-- it. Nothing here is world state and nothing here is perception-bound truth — no scene, no tick, no
-- entities (SPEC-028's own "a world list is a directory, not canon").
--
-- ── display_name ────────────────────────────────────────────────────────────────────────────────
-- The world's name as a chooser sees it, before they are inside it and before any perception exists
-- to filter. This is the ONE label in the system that is not viewer-relative, and it can be, because
-- it names the container rather than anything within it (B-1 governs what is IN a world).
--
-- ── theme (SPEC-019, folded in per the founder's ruling) ─────────────────────────────────────────
-- world_theme/1: ONE accent colour, a mood word, an ornament motif. Carried as JSONB with a
-- schema_version like every other evolving payload (D-4). Deliberately NOT a palette — the frontend
-- derives the rest, because a backend shipping ten colours would own visual design it has no business
-- owning. Deliberately NOT a genre: `mood` is an atmosphere word (daylight/nocturne/mist/ember/bleak)
-- and the system never learns the word "fantasy" (GA-3).
--
-- The CHECK is a floor, not a vocabulary lock: it requires the three keys and a hex accent, and lets
-- any string stand as mood/ornament. Unknown values must DEGRADE, never fail — the frontend's skin
-- falls back safely, so a world authored against a newer vocabulary still renders instead of 500ing.
-- Constraining the enums here would make the backend the thing that breaks on a value it has not
-- heard of, which is exactly backwards.
--
-- ── player_entity_id (the viewer seam) ───────────────────────────────────────────────────────────
-- Who the caller IS in this world. ResolveViewer has been resolving "the actor whose canonical_name
-- is 'Player'" since 0A — a documented stub that cannot survive two worlds, and which already fails
-- in the seeded play world, where the player is named Kade and every non-debug request 500s at the
-- door. The world now states its own player, so resolution is a lookup instead of a naming
-- convention. NULLABLE on purpose: a world may exist before anyone can play it (a world created
-- through the API has no actors yet), and "there is no player here" is a real, answerable state
-- rather than a broken row.
--
-- This is NOT auth. It answers "who does a caller play as in this world", never "who is calling" —
-- that is the B1/session model, still absent. When it lands, it decides WHICH world rows a caller may
-- see and whether they may create one; the shape below does not have to change for that to happen.

CREATE TABLE world (
  world_id     uuid PRIMARY KEY,
  display_name text NOT NULL CHECK (length(btrim(display_name)) > 0),
  theme        jsonb NOT NULL DEFAULT
                 '{"schema_version":"world_theme/1","accent":"#c9a227","mood":"nocturne","ornament":"filigree"}'::jsonb
                 CHECK (
                   theme->>'schema_version' = 'world_theme/1'
                   AND theme->>'accent' ~ '^#[0-9a-fA-F]{6}$'
                   AND length(coalesce(theme->>'mood','')) > 0
                   AND length(coalesce(theme->>'ornament','')) > 0
                 ),
  -- the actor a caller plays as here; NULL until the world has one
  player_entity_id uuid,
  created_at   timestamptz NOT NULL DEFAULT now()   -- operational telemetry only, never in-world time (B-5)
);

GRANT SELECT ON world TO app_reader;

-- fn_world_directory: the world list, as a projection rather than a table scan at the API. It carries
-- ONLY directory fields — no state, no counts, no canon — so there is no surface here for world truth
-- to leak through, and the API layer has nothing to strip.
--
-- Every registered world is returned. SPEC-028 asks for "the worlds the caller may see", and with no
-- session model there is no caller to filter by: inventing an ownership predicate now would be
-- guessing at the auth design rather than waiting for it. This function is the ONE place that filter
-- attaches when B1 lands — a WHERE clause here, and every caller inherits it.
CREATE FUNCTION fn_world_directory() RETURNS json LANGUAGE sql STABLE AS $$
  SELECT json_build_object(
    'schema_version', 'world_directory/1',
    'worlds', COALESCE((
      SELECT json_agg(json_build_object(
               'id',            w.world_id,
               'display_name',  w.display_name,
               'theme',         w.theme,
               'playable',      w.player_entity_id IS NOT NULL
             ) ORDER BY w.display_name, w.world_id)
        FROM world w), '[]'::json)
  );
$$;

-- The two worlds that already exist register themselves in their OWN seeds, not here. Migrations run
-- before seeds, so at this point entity_registry is empty: resolving a player id now would write NULL
-- into every row and quietly mark both worlds unplayable. A world's directory entry belongs with the
-- world it describes anyway.

-- migrate:down

DROP FUNCTION IF EXISTS fn_world_directory();
DROP TABLE IF EXISTS world;
