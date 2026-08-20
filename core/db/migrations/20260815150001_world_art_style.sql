-- migrate:up

-- The style choice the world should render pictures with.
--
-- This is stored on `world` because both image fills (scenes and portraits) are world-scoped jobs:
-- one world chooses one baseline look, and every generated image in that world should resolve from
-- the same stored choice. Keeping it on image_slot would duplicate the same value across rows and
-- allow one world to drift into several styles by accident.
--
-- Nullable for the same reason `brief` is nullable: every world that already exists predates this
-- column. NULL means "no explicit choice was stored", which resolves to the house fallback style in
-- the API layer and keeps the existing dreamchat-default profile key.
--
-- Non-blank when present. Empty text would be a third state ("set, but nothing"), which is not a
-- meaningful style choice and would force every caller to special-case '' beside NULL.
ALTER TABLE world ADD COLUMN art_style text;

ALTER TABLE world ADD CONSTRAINT world_art_style_check
  CHECK (art_style IS NULL OR length(btrim(art_style)) > 0);

COMMENT ON COLUMN world.art_style IS
  'Requested art style choice for this world (preset key or custom prose). NULL means no explicit choice and resolves to the house fallback profile.';

-- migrate:down

ALTER TABLE world DROP CONSTRAINT IF EXISTS world_art_style_check;
ALTER TABLE world DROP COLUMN IF EXISTS art_style;
