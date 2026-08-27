-- migrate:up
-- Source: canon_engine/03 §3 (projection rules), §6 (replay); design §4.3/§6.5.
-- SOLE-PROJECTION-WRITER (was 0A Rider A): apply_mutation() is the ONLY projection write path; live trigger AND replay both call it.

CREATE FUNCTION apply_mutation(m state_mutation) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  -- strip leading 'attrs.' (6 chars) -> single-key JSON path under attrs (ABSOLUTE-STATE-SETS, was 0A Rider B)
  jpath text[] := string_to_array(substring(m.attribute_path from 7), '.');
BEGIN
  IF m.entity_kind = 'actor' THEN
    INSERT INTO actor_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(actor_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'location' THEN
    INSERT INTO location_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(location_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'artifact' THEN
    INSERT INTO artifact_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(artifact_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'relationship' THEN
    -- SPEC-001: doc 03 does not define mutation->(a_id,b_id) addressing. NO-OP stub in 0A.
    NULL;
  END IF;
END $$;
ALTER FUNCTION apply_mutation(state_mutation) OWNER TO maintainer;

-- Live projection trigger: fire on accepted parent only (doc 03 §3.1).
CREATE FUNCTION sm_project() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
  IF (SELECT status FROM canon_event WHERE event_id = NEW.event_id) = 'accepted' THEN
    PERFORM apply_mutation(NEW);
  END IF;
  RETURN NEW;
END $$;
ALTER FUNCTION sm_project() OWNER TO maintainer;
CREATE TRIGGER trg_sm_project
  AFTER INSERT ON state_mutation FOR EACH ROW EXECUTE FUNCTION sm_project();

-- I-1 replay (design §6.5): snapshot -> truncate -> rebuild via the SAME apply_mutation -> domain diff.
-- DROP TABLE IF EXISTS makes it re-entrant within one transaction (the negative-control test calls it 3x).
CREATE FUNCTION replay_0A() RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE ev RECORD; m state_mutation; diff_count int;
BEGIN
  DROP TABLE IF EXISTS snap_actor, snap_location, snap_artifact, snap_rel;
  CREATE TEMP TABLE snap_actor    ON COMMIT DROP AS SELECT * FROM actor_state;
  CREATE TEMP TABLE snap_location ON COMMIT DROP AS SELECT * FROM location_state;
  CREATE TEMP TABLE snap_artifact ON COMMIT DROP AS SELECT * FROM artifact_state;
  CREATE TEMP TABLE snap_rel      ON COMMIT DROP AS SELECT * FROM relationship_state;

  TRUNCATE actor_state, location_state, artifact_state, relationship_state;

  -- DETERMINISTIC-DOMAIN-ORDER (was 0A Rider C): domain-only deterministic order. recorded_at (volatile) excluded.
  FOR ev IN SELECT event_id FROM canon_event WHERE status='accepted'
            ORDER BY world_id, in_world_tick, beat_seq LOOP
    FOR m IN SELECT * FROM state_mutation WHERE event_id = ev.event_id
             ORDER BY valid_from_tick, valid_from_seq LOOP
      PERFORM apply_mutation(m);
    END LOOP;
  END LOOP;

  -- §6.5.1 per-table domain diff (exclude volatile updated_at; identity = PK).
  SELECT
      (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state)) d)
    + (SELECT count(*) FROM (
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel)
        UNION ALL
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state)) d)
  INTO diff_count;
  RETURN diff_count = 0;
END $$;
ALTER FUNCTION replay_0A() OWNER TO maintainer;

-- The SECURITY DEFINER functions run as maintainer; grant the reads they perform.
GRANT SELECT ON canon_event   TO maintainer;
GRANT SELECT ON state_mutation TO maintainer;

-- I-7 function hardening: SECURITY DEFINER functions are doors through the grant wall.
REVOKE EXECUTE ON FUNCTION apply_mutation(state_mutation) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION sm_project()                   FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION replay_0A()                    FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION apply_mutation(state_mutation) TO maintainer;
GRANT  EXECUTE ON FUNCTION replay_0A()                    TO maintainer;

-- migrate:down
DROP FUNCTION IF EXISTS replay_0A();
DROP TRIGGER IF EXISTS trg_sm_project ON state_mutation;
DROP FUNCTION IF EXISTS sm_project();
DROP FUNCTION IF EXISTS apply_mutation(state_mutation);
