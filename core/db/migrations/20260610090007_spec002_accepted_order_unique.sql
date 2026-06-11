-- migrate:up
-- SPEC-002 (docs/open-spec-items.md): replay_0A() orders accepted events by
-- (world_id, in_world_tick, beat_seq) with volatile recorded_at excluded. That is only a total
-- order if the key is unique per world among ACCEPTED events — enforce it by constraint, not by
-- seed-data shape. Partial: proposed/rejected rows may share a slot (only canonical order matters).
-- NOT part of frozen doc 03 §1 (kept out of migrations 0002–0006); lands in the register via the
-- SPEC-002 proposed ADR.
CREATE UNIQUE INDEX uq_ce_accepted_order
  ON canon_event (world_id, in_world_tick, beat_seq)
  WHERE status = 'accepted';

-- migrate:down
DROP INDEX IF EXISTS uq_ce_accepted_order;
