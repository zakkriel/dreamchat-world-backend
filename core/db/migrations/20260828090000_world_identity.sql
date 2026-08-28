-- migrate:up

-- World identity inferred by the understanding pass, stored beside genesis_doc (design Q5 for this
-- slice: beside, not inside world_genesis/1, so the published fiction schema does not gain
-- instructions-about-content). Operational, never rendered: no projection selects it.
-- NULL for hand-seeded and templated worlds.

ALTER TABLE world ADD COLUMN world_identity jsonb;

COMMENT ON COLUMN world.world_identity IS
  'world_identity/1 from the understanding pass. Server-side only. NULL if the world was not built this way.';

-- migrate:down

ALTER TABLE world DROP COLUMN IF EXISTS world_identity;
