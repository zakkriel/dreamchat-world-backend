-- migrate:up

-- The world-surface pass: the founder's Lovable design loop flagged three fields the world picker
-- needs and one the play surface needs, all of them riding the same image/world surface.
--
-- ── tagline: AUTHORED FICTION, AND THEREFORE NULLABLE FOREVER ───────────────────────────────────
-- One world-authored line per card. It is fiction (GA-2), so this service must never compose it: no
-- default, no template, no "A world of {mood} {ornament}". A world with no authored tagline ships
-- NULL and the card renders without one — the same honest absence `image` uses. The CHECK forbids a
-- blank string specifically so "" can never become a second spelling of absent.
ALTER TABLE world ADD COLUMN tagline text;
ALTER TABLE world ADD CONSTRAINT world_tagline_check
  CHECK (tagline IS NULL OR length(btrim(tagline)) > 0);

-- ── image_slot gains a fourth owner kind ────────────────────────────────────────────────────────
-- 'location' was already allowed and is what a place backdrop uses. A world cover has no entity to
-- hang on, so the world IS its own owner: owner_kind='world', owner_id=world_id. That keeps one
-- slot table, one reap rule (#58: archived ⇒ dangling ⇒ re-request), and one fetch endpoint for
-- every picture this service knows about.
ALTER TABLE image_slot DROP CONSTRAINT image_slot_owner_kind_check;
ALTER TABLE image_slot ADD CONSTRAINT image_slot_owner_kind_check
  CHECK (owner_kind = ANY (ARRAY['actor'::text, 'location'::text, 'artifact'::text, 'world'::text]));

-- ── world_directory/1 → /2 ──────────────────────────────────────────────────────────────────────
-- Three added fields, and the payload is additionalProperties:false, so the version moves. Clean
-- cutover, no alias — the version moving IS the notification.
--
-- last_place_label is the world's own label for where the player stands or left off. Deliberately a
-- LABEL AND NOTHING ELSE: no tick, no timestamp, no "2 hours ago". B-5 keeps wall-clock off this
-- boundary entirely, and the directory is a shelf of doors, not a save-game list — "Dock Street" is
-- the whole answer to "where was I". It is rendered through fn_display_name as the WORLD'S OWN
-- PLAYER, because that is whose whereabouts it is and whose naming the card is quoting; a caller who
-- opens the door becomes exactly that viewer (ResolveViewer). NULL when the world has no player yet,
-- or has one who has never been placed anywhere — never entered, nothing to say.
CREATE OR REPLACE FUNCTION fn_world_directory() RETURNS json
LANGUAGE sql STABLE AS $$
  SELECT json_build_object(
    'schema_version', 'world_directory/2',
    'worlds', COALESCE((
      SELECT json_agg(json_build_object(
               'id',            w.world_id,
               'display_name',  w.display_name,
               'tagline',       w.tagline,
               'theme',         w.theme,
               'playable',      w.player_entity_id IS NOT NULL,
               'cover_image',   fn_image_ref(w.world_id, 'world', w.world_id),
               'last_place_label', (
                  SELECT fn_display_name(w.world_id, w.player_entity_id,
                                         (a.attrs->>'location_id')::uuid)
                    FROM actor_state a
                   WHERE a.world_id = w.world_id
                     AND a.entity_id = w.player_entity_id
                     AND a.attrs->>'location_id' IS NOT NULL
               )
             ) ORDER BY w.display_name, w.world_id)
        FROM world w), '[]'::json)
  );
$$;

-- migrate:down

CREATE OR REPLACE FUNCTION fn_world_directory() RETURNS json
LANGUAGE sql STABLE AS $$
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

ALTER TABLE image_slot DROP CONSTRAINT image_slot_owner_kind_check;
ALTER TABLE image_slot ADD CONSTRAINT image_slot_owner_kind_check
  CHECK (owner_kind = ANY (ARRAY['actor'::text, 'location'::text, 'artifact'::text]));

ALTER TABLE world DROP CONSTRAINT world_tagline_check;
ALTER TABLE world DROP COLUMN tagline;
