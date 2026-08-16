-- migrate:up

-- The description a user typed to make this world (PRD: World Creation, AC-1).
--
-- A created world is authored FROM a sentence or three paragraphs of prose. That prose is the one
-- input the whole build hangs on, and until now the `world` row had nowhere to keep it: the eight
-- existing columns are all structure (display_name, theme, player_entity_id, tagline, template_key,
-- archived_at) and none of them is the user's own words.
--
-- Why it is stored at all: when a built world reads wrong — a cast that ignores the premise, a room
-- that contradicts the brief — the first question is always "what was it actually asked for?". Without
-- the brief on the row, that question is unanswerable after the fact, and the failure gets diagnosed
-- by guesswork. It is operational truth about how the row came to exist, in the same family as
-- template_key, which records the same fact for a templated world.
--
-- Why it is NEVER rendered: the brief is not world content. The fiction the player may see is the
-- world's own tagline, its places and its people — authored artifacts that went through the gate. The
-- prompt that produced them did not, and putting it on screen would be showing the player the
-- scaffolding instead of the building (D-7, and the frontend's law 1: world-authored strings render
-- verbatim, which the brief is not). No projection function selects it; `fn_world_directory` is
-- deliberately left alone.
--
-- Nullable, because every world that already exists was authored by hand rather than from a brief,
-- and inventing a brief for them retroactively would be a lie. NULL means "not authored from prose".
-- Non-blank when present, matching the discipline display_name and tagline already keep: a column
-- that permits '' permits a row that claims to have a brief and does not.

ALTER TABLE world ADD COLUMN brief text;

ALTER TABLE world ADD CONSTRAINT world_brief_check
  CHECK (brief IS NULL OR length(btrim(brief)) > 0);

COMMENT ON COLUMN world.brief IS
  'The prose a user typed to author this world. Operational provenance, never rendered: no projection selects it. NULL for hand-authored worlds.';

-- migrate:down

ALTER TABLE world DROP CONSTRAINT IF EXISTS world_brief_check;
ALTER TABLE world DROP COLUMN IF EXISTS brief;
