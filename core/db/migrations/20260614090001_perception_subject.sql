-- migrate:up
-- ADR-035 (Proposed → Accepted under chunk-3 gate): perception about-ness is an explicit junction,
-- not a derivation from the source event's participants (SPEC-008). DELTA vs ADR-035's two-column
-- sketch: world_id carried from birth (SPEC-009 tenant-key posture). New table → carrying the
-- tenant key costs zero migration; reopening frozen tables would require a firing trigger.
-- Additive only: no existing engine ADR/invariant/DDL column is modified.
CREATE TABLE perception_subject (
  perception_id UUID NOT NULL REFERENCES perception_record(perception_id),
  entity_id     UUID NOT NULL,
  world_id      UUID NOT NULL,
  PRIMARY KEY (perception_id, entity_id)
);
CREATE INDEX idx_ps_entity ON perception_subject (entity_id);
CREATE INDEX idx_ps_world  ON perception_subject (world_id);

-- DELETE guard: about-ness is append-only like its parent perception (ADR-006). forbid_delete()
-- was created in migration 0002 and is reused across the schema.
CREATE TRIGGER trg_perception_subject_no_delete
  BEFORE DELETE ON perception_subject FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TRIGGER IF EXISTS trg_perception_subject_no_delete ON perception_subject;
DROP TABLE IF EXISTS perception_subject;
