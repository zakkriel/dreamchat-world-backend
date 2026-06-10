-- migrate:up
-- Source: canon_engine/03 §1.2, §1.3, §1.4 (frozen v4.1)

CREATE TABLE state_mutation (
  mutation_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id       UUID NOT NULL,
  event_id       UUID NOT NULL REFERENCES canon_event(event_id),
  entity_id      UUID NOT NULL,
  entity_kind    TEXT NOT NULL,
  attribute_path TEXT NOT NULL,
  old_value      JSONB,
  new_value      JSONB NOT NULL,
  valid_from_tick BIGINT NOT NULL,
  valid_from_seq  INT NOT NULL DEFAULT 0,
  status         TEXT NOT NULL DEFAULT 'applied'
                 CHECK (status IN ('applied','reversed','dirty'))
);
CREATE INDEX idx_sm_entity ON state_mutation (entity_id, valid_from_tick, valid_from_seq);
CREATE INDEX idx_sm_event  ON state_mutation (event_id);

CREATE TABLE provenance_edge (
  derived_id   UUID NOT NULL,
  derived_kind TEXT NOT NULL CHECK (derived_kind IN ('perception','mutation','event','bundle')),
  source_id    UUID NOT NULL,
  source_kind  TEXT NOT NULL CHECK (source_kind  IN ('perception','mutation','event')),
  how_type     TEXT NOT NULL CHECK (how_type IN
               ('derived_from','inferred_from','reported_by','witnessed_by','compensates','supersedes')),
  PRIMARY KEY (derived_id, source_id, how_type)
);
CREATE INDEX idx_pe_source ON provenance_edge (source_id);

CREATE TABLE perception_record (
  perception_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  holder_id        UUID NOT NULL,
  source_event_id  UUID NOT NULL REFERENCES canon_event(event_id),
  content          TEXT NOT NULL,
  epistemic_type   TEXT NOT NULL CHECK (epistemic_type IN
                   ('direct','shared','told','overheard','public','rumor',
                    'inference','mistaken','confirmed','disputed')),
  sensory_mode     TEXT,
  confidence       REAL NOT NULL DEFAULT 1.0,
  distortion_level REAL NOT NULL DEFAULT 0,
  acquired_tick    BIGINT NOT NULL,
  valid_tick       BIGINT NOT NULL,
  invalid_tick     BIGINT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  expired_at       TIMESTAMPTZ,
  visibility_scope TEXT NOT NULL DEFAULT 'private',
  dirty            BOOLEAN NOT NULL DEFAULT false,
  importance       REAL NOT NULL DEFAULT 5.0
);
CREATE INDEX idx_pr_holder  ON perception_record (holder_id, acquired_tick);
CREATE INDEX idx_pr_source  ON perception_record (source_event_id);
CREATE INDEX idx_pr_active  ON perception_record (holder_id) WHERE invalid_tick IS NULL AND expired_at IS NULL;

CREATE TABLE causal_bundle (
  bundle_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id    UUID NOT NULL,
  effect_ref  UUID NOT NULL,
  effect_kind TEXT NOT NULL CHECK (effect_kind IN ('event','mutation')),
  semantics   TEXT NOT NULL CHECK (semantics IN ('conjunctive','disjunctive_member','probabilistic')),
  template_id TEXT,
  status      TEXT NOT NULL DEFAULT 'valid'
              CHECK (status IN ('valid','invalidated','pending_review')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cb_effect ON causal_bundle (effect_ref);

CREATE TABLE causal_bundle_input (
  bundle_id  UUID NOT NULL REFERENCES causal_bundle(bundle_id),
  input_ref  UUID NOT NULL,
  input_kind TEXT NOT NULL CHECK (input_kind IN ('event','mutation','perception')),
  role       TEXT NOT NULL CHECK (role IN ('trigger','enabler','blocker','influence')),
  polarity   SMALLINT NOT NULL DEFAULT 1 CHECK (polarity IN (1,-1)),
  weight     REAL NOT NULL DEFAULT 1.0,
  necessity  BOOLEAN NOT NULL DEFAULT true,
  PRIMARY KEY (bundle_id, input_ref, role)
);

-- DELETE guards on the canon/lineage tables (forbid_delete() defined in migration 0002).
CREATE TRIGGER trg_state_mutation_no_delete
  BEFORE DELETE ON state_mutation FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_perception_record_no_delete
  BEFORE DELETE ON perception_record FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_provenance_edge_no_delete
  BEFORE DELETE ON provenance_edge FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TABLE IF EXISTS causal_bundle_input;
DROP TABLE IF EXISTS causal_bundle;
DROP TABLE IF EXISTS perception_record;   -- cascades its DELETE-guard trigger
DROP TABLE IF EXISTS provenance_edge;     -- cascades its DELETE-guard trigger
DROP TABLE IF EXISTS state_mutation;      -- cascades its DELETE-guard trigger
