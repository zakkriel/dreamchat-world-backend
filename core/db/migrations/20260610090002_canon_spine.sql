-- migrate:up
-- Source: canon_engine/03 §1.1 (frozen v4.1)

CREATE TABLE canon_event (
  event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id         UUID NOT NULL,
  scene_id         UUID,
  beat_id          UUID,
  event_type       TEXT NOT NULL,
  summary          TEXT NOT NULL,
  payload          JSONB NOT NULL DEFAULT '{}',
  schema_version   INT  NOT NULL DEFAULT 1,
  in_world_tick    BIGINT NOT NULL,
  in_world_label   TEXT,
  beat_seq         INT NOT NULL DEFAULT 0,
  temporal_uncertainty BOOLEAN NOT NULL DEFAULT false,
  recorded_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  accepted_at      TIMESTAMPTZ,
  status           TEXT NOT NULL DEFAULT 'proposed'
                   CHECK (status IN ('proposed','accepted','rejected','retconned','superseded')),
  visibility_scope TEXT NOT NULL DEFAULT 'private',
  confidence       REAL,
  origin           TEXT NOT NULL DEFAULT 'fast_path'
                   CHECK (origin IN ('fast_path','template','freeform','threshold','backstage','compensation')),
  template_id      TEXT,
  source_refs      JSONB,
  superseded_by    UUID REFERENCES canon_event(event_id)
);
CREATE INDEX idx_ce_world_time   ON canon_event (world_id, in_world_tick, beat_seq);
CREATE INDEX idx_ce_status       ON canon_event (world_id, status) WHERE status = 'accepted';
CREATE INDEX idx_ce_beat         ON canon_event (beat_id);
CREATE INDEX idx_ce_scene        ON canon_event (scene_id);
CREATE INDEX idx_ce_payload_gin  ON canon_event USING GIN (payload);

CREATE TABLE event_participant (
  event_id       UUID NOT NULL REFERENCES canon_event(event_id),
  entity_id      UUID NOT NULL,
  entity_kind    TEXT NOT NULL CHECK (entity_kind IN ('actor','location','artifact','faction','group')),
  role_qualifier TEXT NOT NULL,
  PRIMARY KEY (event_id, entity_id, role_qualifier)
);
CREATE INDEX idx_ep_entity ON event_participant (entity_id);

-- Append-only (doc 03 §1.1): only {status, accepted_at, superseded_by} may change. The ROW() list below
-- enumerates ALL 18 immutable columns verbatim from doc 03 §1.1 (payload + schema_version INCLUDED).
CREATE FUNCTION canon_event_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF ROW(NEW.event_id, NEW.world_id, NEW.scene_id, NEW.beat_id, NEW.event_type, NEW.summary,
         NEW.payload, NEW.schema_version, NEW.in_world_tick, NEW.in_world_label, NEW.beat_seq,
         NEW.temporal_uncertainty, NEW.recorded_at, NEW.visibility_scope, NEW.confidence,
         NEW.origin, NEW.template_id, NEW.source_refs)
     IS DISTINCT FROM
     ROW(OLD.event_id, OLD.world_id, OLD.scene_id, OLD.beat_id, OLD.event_type, OLD.summary,
         OLD.payload, OLD.schema_version, OLD.in_world_tick, OLD.in_world_label, OLD.beat_seq,
         OLD.temporal_uncertainty, OLD.recorded_at, OLD.visibility_scope, OLD.confidence,
         OLD.origin, OLD.template_id, OLD.source_refs)
  THEN
    RAISE EXCEPTION 'canon_event is append-only: only {status, accepted_at, superseded_by} may change (event %)', OLD.event_id;
  END IF;

  IF OLD.status IS DISTINCT FROM NEW.status
     AND NOT ( (OLD.status='proposed' AND NEW.status IN ('accepted','rejected'))
            OR (OLD.status='accepted' AND NEW.status IN ('retconned','superseded')) ) THEN
    RAISE EXCEPTION 'illegal canon_event status transition % -> %', OLD.status, NEW.status;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_canon_event_append_only
  BEFORE UPDATE ON canon_event FOR EACH ROW EXECUTE FUNCTION canon_event_append_only();

-- Generic DELETE guard for canon tables (doc 03 §1.1 "DELETE revoked"; ADR-006 invalidation-never-deletion).
CREATE FUNCTION forbid_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'DELETE forbidden on % (append-only canon, ADR-001/006)', TG_TABLE_NAME;
END $$;
CREATE TRIGGER trg_canon_event_no_delete
  BEFORE DELETE ON canon_event FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_event_participant_no_delete
  BEFORE DELETE ON event_participant FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TABLE IF EXISTS event_participant;   -- cascades its triggers
DROP TABLE IF EXISTS canon_event;         -- cascades its triggers
DROP FUNCTION IF EXISTS canon_event_append_only();
DROP FUNCTION IF EXISTS forbid_delete();
