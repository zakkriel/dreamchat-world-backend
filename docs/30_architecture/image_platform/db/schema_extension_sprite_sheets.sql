-- DreamChat Image Platform V2 Addendum
-- Schema extension proposal for sprite sheets and expression slicing.
-- This is a migration proposal, not final production SQL.

ALTER TABLE visual_asset
  ADD COLUMN IF NOT EXISTS parent_asset_id TEXT NULL,
  ADD COLUMN IF NOT EXISTS crop_index INT NULL,
  ADD COLUMN IF NOT EXISTS crop_box JSONB NULL,
  ADD COLUMN IF NOT EXISTS expression_key TEXT NULL,
  ADD COLUMN IF NOT EXISTS scene_state JSONB NULL,
  ADD COLUMN IF NOT EXISTS match_tags JSONB NULL;

CREATE TABLE IF NOT EXISTS sprite_sheet_contract (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  rows INT NOT NULL,
  columns INT NOT NULL,
  expression_order JSONB NOT NULL,
  safe_margin_px INT NOT NULL DEFAULT 0,
  allow_text_labels BOOLEAN NOT NULL DEFAULT FALSE,
  allow_internal_frames BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sprite_sheet_slice (
  id TEXT PRIMARY KEY,
  sheet_asset_id TEXT NOT NULL REFERENCES visual_asset(id),
  slice_asset_id TEXT NOT NULL REFERENCES visual_asset(id),
  visual_identity_id TEXT NOT NULL REFERENCES visual_identity(id),
  expression_key TEXT NOT NULL,
  crop_index INT NOT NULL,
  crop_box JSONB NOT NULL,
  quality_status TEXT NOT NULL DEFAULT 'pending',
  quality_notes TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_visual_asset_parent_asset_id ON visual_asset(parent_asset_id);
CREATE INDEX IF NOT EXISTS idx_visual_asset_expression_key ON visual_asset(visual_identity_id, expression_key);
CREATE INDEX IF NOT EXISTS idx_visual_asset_scene_state ON visual_asset USING GIN(scene_state);
CREATE INDEX IF NOT EXISTS idx_sprite_sheet_slice_sheet ON sprite_sheet_slice(sheet_asset_id);
CREATE INDEX IF NOT EXISTS idx_sprite_sheet_slice_expression ON sprite_sheet_slice(visual_identity_id, expression_key);

INSERT INTO sprite_sheet_contract (
  id,
  name,
  rows,
  columns,
  expression_order,
  safe_margin_px,
  allow_text_labels,
  allow_internal_frames
) VALUES (
  'contract_character_expression_2x5_v1',
  'Character Expression Sheet 2x5 V1',
  2,
  5,
  '["neutral","warm","amused","suspicious","angry","afraid","sad","surprised","focused","exhausted"]'::jsonb,
  24,
  FALSE,
  FALSE
) ON CONFLICT (id) DO NOTHING;
