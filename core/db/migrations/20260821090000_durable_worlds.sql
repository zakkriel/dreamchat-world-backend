-- migrate:up

-- Durable worlds (spec: docs/superpowers/specs/2026-08-21-durable-worlds-design.md).
--
-- A generated world now commits when AUTHORING ends, not when the player finally answers the last
-- kickstart question. Production taught the ordering of value the hard way (2026-08-20/21): the
-- world_genesis call is the long, expensive, reliable part; the kickstart call is the short, cheap,
-- flaky part — and the old posture let the flaky part destroy the expensive part through a 15-minute
-- in-memory draft TTL. From this migration on, the world row and its whole content exist the moment
-- authoring succeeds; only the player and the arrival wait for the user's answers.
--
-- ── genesis_doc ──────────────────────────────────────────────────────────────────────────────────
-- The authored world_genesis/1 document, verbatim. This is the between-phases home the in-memory
-- draft store used to be: the kickstart seat authors against it, and the arrival commit recomputes
-- deterministic geometry (place coordinates) from it. It holds every secret, every hiding, every
-- knowledge path — so the AC-7 boundary it lived behind in memory holds here identically: NO
-- projection function selects it, no route serves it, and it never crosses the wire. Same family as
-- `brief`: operational truth about how the row came to exist. NULL means "not authored from a
-- document" — every hand-seeded and templated world.
--
-- ── kickstart_state ──────────────────────────────────────────────────────────────────────────────
-- The between-turns state of an unfinished creation: the chosen identity, the authored scenario
-- options, any referenced-but-new cast the character turn authored, and the running spend tally.
-- NULL for a finished world (the arrival commit clears it) and for worlds never built this way.
-- Losing it costs the user at most one re-asked question — never the world — which is the whole
-- point of this design. Secrets discipline identical to genesis_doc: server-side only.
--
-- ── world_character ──────────────────────────────────────────────────────────────────────────────
-- Every character ever committed into a world, one row per arrival. Today exactly one row per world
-- and `world.player_entity_id` (SPEC-028) remains the single active pointer every consumer reads —
-- this table exists so that allowing a SECOND character later is an insert and a pointer move, not a
-- migration of the viewer seam. entity_id references the registered actor; descriptor/canonical_name
-- are denormalised from the premise so the creation surface can list characters without joining
-- through a viewer-scoped projection.

ALTER TABLE world ADD COLUMN genesis_doc jsonb;
ALTER TABLE world ADD COLUMN kickstart_state jsonb;

CREATE TABLE world_character (
  character_id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id       uuid NOT NULL REFERENCES world(world_id) ON DELETE CASCADE,
  entity_id      uuid NOT NULL,
  descriptor     text NOT NULL CHECK (length(btrim(descriptor)) > 0),
  canonical_name text NOT NULL CHECK (length(btrim(canonical_name)) > 0),
  created_at     timestamptz NOT NULL DEFAULT now()  -- operational telemetry only, never in-world time (B-5)
);

CREATE INDEX world_character_world_idx ON world_character (world_id);

GRANT SELECT ON world_character TO app_reader;

-- migrate:down

DROP TABLE world_character;
ALTER TABLE world DROP COLUMN kickstart_state;
ALTER TABLE world DROP COLUMN genesis_doc;
