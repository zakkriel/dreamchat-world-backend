-- migrate:up
-- Source: canon_engine/03 §1 (frozen v4.1) — roles support I-7. PG16: gen_random_uuid() is core.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'maintainer') THEN CREATE ROLE maintainer NOLOGIN; END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_reader') THEN CREATE ROLE app_reader NOLOGIN; END IF;
END $$;

-- migrate:down
DROP ROLE IF EXISTS app_reader;
DROP ROLE IF EXISTS maintainer;
