-- migrate:up
-- Source: canon_engine/03 §1.5 (frozen v4.1)

CREATE TABLE actor_state (
  entity_id     UUID PRIMARY KEY,
  world_id      UUID NOT NULL,
  attrs         JSONB NOT NULL DEFAULT '{}',
  dirty         BOOLEAN NOT NULL DEFAULT false,
  last_event_id UUID,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE location_state (LIKE actor_state INCLUDING ALL);
CREATE TABLE artifact_state (LIKE actor_state INCLUDING ALL);

CREATE TABLE relationship_state (
  world_id      UUID NOT NULL,
  a_id          UUID NOT NULL,
  b_id          UUID NOT NULL,
  attrs         JSONB NOT NULL DEFAULT '{}',
  dirty         BOOLEAN NOT NULL DEFAULT false,
  last_event_id UUID,
  PRIMARY KEY (world_id, a_id, b_id)
);

-- I-7: projections writable only by the maintainer role; app_reader reads only.
REVOKE ALL ON actor_state, location_state, artifact_state, relationship_state FROM PUBLIC;
GRANT  ALL    ON actor_state, location_state, artifact_state, relationship_state TO maintainer;
GRANT  SELECT ON actor_state, location_state, artifact_state, relationship_state TO app_reader;

-- migrate:down
DROP TABLE IF EXISTS relationship_state;
DROP TABLE IF EXISTS artifact_state;
DROP TABLE IF EXISTS location_state;
DROP TABLE IF EXISTS actor_state;
