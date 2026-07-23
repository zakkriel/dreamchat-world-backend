-- migrate:up

-- 1) Six canonical event types join the legal set. Legacy labels
-- ('move','private_disclosure','world_genesis') stay legal: append-only
-- history is never rewritten. New writes use canonical labels only.
-- NOTE: canon_event had no prior event_type CHECK; we create one now
-- covering both legacy labels (including 'observation' and 'publicize' found in
-- existing seed data) and the six canonical types.
ALTER TABLE canon_event ADD CONSTRAINT canon_event_event_type_check CHECK (event_type IN (
  'ActorMoved','Communicated','ObjectRelocated','OwnershipAccessChanged',
  'EntityCreated','EntityDestroyed','AttributeChanged',
  'move','private_disclosure','world_genesis','observation','publicize'));

-- 2) Physics vocabulary tables (A11: engine hardcodes grammar; the LLM mints
-- typed rows). MEASUREMENTS only — no verdict columns, ever.
CREATE TABLE movement_type (
  world_id          uuid    NOT NULL,
  movement_type_id  text    NOT NULL,
  base_speed_mps    numeric NOT NULL CHECK (base_speed_mps > 0),
  PRIMARY KEY (world_id, movement_type_id)
);

CREATE TABLE status_modifier (
  world_id          uuid    NOT NULL,
  status_type_id    text    NOT NULL,
  action_type       text    NOT NULL CHECK (action_type IN ('move')),
  movement_type_id  text    NOT NULL,
  modifier_percent  numeric NOT NULL CHECK (modifier_percent >= -100),
  PRIMARY KEY (world_id, status_type_id, action_type, movement_type_id),
  FOREIGN KEY (world_id, movement_type_id) REFERENCES movement_type(world_id, movement_type_id)
);

-- 3) Every world seeds exactly walk 1.4 + encumbered -100 (contracts §2:
-- "these two are the only predefined rows").
CREATE FUNCTION seed_world_defaults(p_world_id uuid) RETURNS void LANGUAGE sql AS $$
  INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps)
  VALUES (p_world_id, 'walk', 1.4) ON CONFLICT DO NOTHING;
  INSERT INTO status_modifier (world_id, status_type_id, action_type, movement_type_id, modifier_percent)
  VALUES (p_world_id, 'encumbered', 'move', 'walk', -100) ON CONFLICT DO NOTHING;
$$;

-- 4) Tension: a scene attribute, enum-guarded at write.
CREATE FUNCTION trg_validate_tension() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.attrs ? 'tension' AND NOT (NEW.attrs->>'tension' IN ('frantic','tense','normal','calm','none')) THEN
    RAISE EXCEPTION 'tension % not in enum', NEW.attrs->>'tension';
  END IF;
  RETURN NEW;
END $$;

CREATE TRIGGER location_state_tension
  BEFORE INSERT OR UPDATE ON location_state
  FOR EACH ROW EXECUTE FUNCTION trg_validate_tension();

-- migrate:down

DROP TRIGGER IF EXISTS location_state_tension ON location_state;
DROP FUNCTION IF EXISTS trg_validate_tension();
DROP FUNCTION IF EXISTS seed_world_defaults(uuid);
DROP TABLE IF EXISTS status_modifier;
DROP TABLE IF EXISTS movement_type;
ALTER TABLE canon_event DROP CONSTRAINT IF EXISTS canon_event_event_type_check;
