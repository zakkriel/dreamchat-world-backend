-- migrate:up
-- Source: canon_engine/03 §1.5 (frozen v4.1)
CREATE TABLE entity_registry (
  entity_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  entity_kind      TEXT NOT NULL,
  canonical_name   TEXT NOT NULL,
  aliases          TEXT[] NOT NULL DEFAULT '{}',
  descriptor       TEXT,
  current_scene_id UUID,
  created_by_event UUID REFERENCES canon_event(event_id),
  status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','merged'))
);
CREATE INDEX idx_er_scene ON entity_registry (world_id, current_scene_id);
CREATE INDEX idx_er_name  ON entity_registry (world_id, canonical_name);

-- migrate:down
DROP TABLE IF EXISTS entity_registry;
