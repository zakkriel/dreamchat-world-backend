.PHONY: doctor db-up db-down migrate seed test replay reset schema-check pgtap

doctor:          ## verify the sanctioned Docker runtime is available
	@command -v docker >/dev/null 2>&1 || { \
	  echo "ERROR: Docker is required (the only sanctioned environment)."; \
	  echo "Install a runtime manually, then re-run:"; \
	  echo "  brew install colima docker docker-compose && colima start   (or Docker Desktop)"; \
	  exit 1; }
	@docker compose version >/dev/null 2>&1 || { echo "ERROR: 'docker compose' v2 not available"; exit 1; }
	@echo "docker OK: $$(docker --version)"

db-up: doctor    ## start Postgres, wait for healthy
	docker compose up -d --build db
	docker compose exec -T db bash -c 'until pg_isready -U postgres -d dreamchat; do sleep 1; done'

db-down:
	docker compose down -v

pgtap:           ## install pgTAP extension into the db
	docker compose exec -T db psql -U postgres -d dreamchat -c 'CREATE EXTENSION IF NOT EXISTS pgtap;'

migrate:         ## apply dbmate migrations + dump schema.sql
	docker compose run --rm dbmate up

seed:            ## load the deterministic Mara seed
	docker compose exec -T db psql -U postgres -d dreamchat -v ON_ERROR_STOP=1 -f /work/seeds/seed_mara_0A.sql

test: pgtap      ## run the pgTAP suite (run `make reset` first for seed-dependent tests)
	docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/*_test.sql'

replay:          ## run I-1 replay by hand (boolean); ROLLBACK so the truncate/rebuild never commits
	docker compose exec -T db psql -U postgres -d dreamchat -c 'BEGIN; SELECT replay_0A(); ROLLBACK;'

fingerprint:     ## dump domain-only projection state (volatile updated_at excluded) for determinism diffs
	@docker compose exec -T db psql -U postgres -d dreamchat -Atc " \
	  SELECT 'actor',    entity_id, world_id, attrs, last_event_id, dirty FROM actor_state    \
	  UNION ALL SELECT 'location', entity_id, world_id, attrs, last_event_id, dirty FROM location_state \
	  UNION ALL SELECT 'artifact', entity_id, world_id, attrs, last_event_id, dirty FROM artifact_state \
	  UNION ALL SELECT 'rel', a_id, world_id, attrs, last_event_id, dirty FROM relationship_state \
	  ORDER BY 1, 2"

reset: db-down db-up migrate seed ## clean DB from scratch (determinism check helper)

schema-check: db-down db-up migrate ## fail if dbmate schema.sql has uncommitted drift (always against a clean migrated DB)
	git diff --exit-code core/db/schema.sql
