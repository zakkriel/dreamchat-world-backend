-- migrate:up

-- ─── 0. Extend canon_event.origin to include 'telegraph' (append-only) ────────
-- A cognition seat that TELEGRAPHS a disruptive act commits its wind-up as perceivable canon via
-- apply_ruled_event with p_origin='telegraph' — distinguishing the wind-up from ordinary ruled
-- ('ruling') and attempt-path commits. Append-only: every prior origin value is preserved.

ALTER TABLE canon_event
  DROP CONSTRAINT IF EXISTS canon_event_origin_check;

ALTER TABLE canon_event
  ADD CONSTRAINT canon_event_origin_check
    CHECK (origin IN ('fast_path','template','freeform','threshold','backstage','compensation','ruling','telegraph'));

-- ─── 1. held_outcome — the reaction-beat's dedicated record (RULINGS-2026-07-24 §3) ──
--
-- When the wind-up commits, this row holds the NPC's FULL intended act (her typed attempt) plus a
-- link to the committed telegraph event, until the player's next input resolves it (Task 6 flips
-- status → 'resolved' inside the combined ruling's tx).
--
-- This is LOOP STATE, not canon: Go writes it with a plain INSERT in the SAME pgx tx as the wind-up
-- commit (canon still flows only through apply_ruled_event). Because it is loop state, rows may be
-- deleted; there is no append-only delete-guard.

CREATE TABLE held_outcome (
  held_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id uuid NOT NULL,
  actor_id uuid NOT NULL,           -- the NPC whose act is held
  attempt jsonb NOT NULL,           -- her full typed attempt (npc_attempts/1 inner shape)
  telegraph_event_id uuid NOT NULL REFERENCES canon_event(event_id),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','resolved')),
  created_tick bigint NOT NULL
);

-- The reaction beat's next-input lookup scans pending holds by world; a partial index keeps it to
-- exactly the live rows.
CREATE INDEX idx_held_outcome_pending ON held_outcome (world_id) WHERE status = 'pending';

-- A held outcome fires on the PLAYER'S NEXT INPUT, never on the clock —
-- which is why this is NOT a pending_event row (RULINGS-2026-07-24 §3).

-- migrate:down

DROP TABLE IF EXISTS held_outcome;

-- Restore canon_event.origin to the pre-telegraph (Station D) set — 'ruling' stays, 'telegraph' goes.
ALTER TABLE canon_event
  DROP CONSTRAINT IF EXISTS canon_event_origin_check;

ALTER TABLE canon_event
  ADD CONSTRAINT canon_event_origin_check
    CHECK (origin IN ('fast_path','template','freeform','threshold','backstage','compensation','ruling'));
