-- migrate:up

-- Beat derivation — the parse record. What did the decompose stage make of each player sentence?
-- Written for EVERY beat, including the ones that produced NOTHING: the empty parse is the row this
-- table exists for. Counting those, with the sentence attached, is what turns "which action types are
-- missing?" from an argument into a query (SPEC-037; founder ruling 2026-08-28).
--
-- WHY NOT transcript_entry. Two reasons, both fatal to reusing it. It is a PUBLISHED contract
-- (transcript/2), so widening it makes every change a cross-repo round. And it deliberately writes
-- nothing when a beat produced no prose — "a beat that produced no prose leaves no memory"
-- (transcript.go) — which silently drops the exact case being measured. Today an unparsed sentence
-- leaves no record anywhere at all.
--
-- NOT a player surface. No projection reads this and no payload carries it. It holds only the
-- player's own typed words and entity ids the decompose stage bound from the CANDIDATES list, which
-- is already perception-bound to that viewer (B-1) — so it reveals nothing to anyone that they could
-- not already see. It is not canon: nothing here is an event, and replay never reads it.
--
-- recorded_at is WALL-CLOCK on purpose. In-world time is a logical tick (B-5) and is recorded
-- separately as in_world_tick for correlating with canon; retention is an operational concern and is
-- measured against operational time.

CREATE TABLE beat_derivation (
  derivation_id  bigserial   PRIMARY KEY,
  world_id       uuid        NOT NULL,
  viewer_id      uuid        NOT NULL,
  in_world_tick  bigint      NOT NULL,
  recorded_at    timestamptz NOT NULL DEFAULT now(),
  -- The player's own typed sentence. NULL only if a client sent no text field at all: every beat
  -- carries a sentence now that the continue press is gone (deleted 2026-08-28 with
  -- runJourneyToCompletion — a journey runs its own legs, so there was nothing left to press).
  stated         text,
  -- One object per decoded chain element: {"type","stated","ids"} — the same shape the debug
  -- reasoning trace already builds (BeatTrace.Decompose), minus every truth-side field. An EMPTY
  -- array is the meaningful value: the parse produced no actions.
  elements       jsonb       NOT NULL DEFAULT '[]'::jsonb
);

-- element count is NOT stored: it is jsonb_array_length(elements). One home per fact (D-6).
CREATE INDEX beat_derivation_recorded_at_idx ON beat_derivation (recorded_at);
CREATE INDEX beat_derivation_world_idx       ON beat_derivation (world_id, recorded_at DESC);

COMMENT ON TABLE beat_derivation IS
  'Operational telemetry: what the decompose stage made of each player sentence, including the empty parse. Retention 15 days (founder ruling 2026-08-28). Never a player surface; no projection reads it; not canon.';

-- migrate:down

DROP TABLE IF EXISTS beat_derivation;
