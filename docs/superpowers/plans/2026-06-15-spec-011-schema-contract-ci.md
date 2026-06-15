# SPEC-011 — Standing payload↔schema contract CI test (small plan)

**Goal:** Permanently close the gap SPEC-010 exposed: the pgTAP/Go suites cannot catch an
*over-permissive* published schema (a non-null payload validates fine against a nullable schema), so a
field like `confidence` (DDL `REAL NOT NULL DEFAULT 1.0`) being typed `["number","null"]` slipped
through. SPEC-011 is a CI test with **two directions**, both required.

**Scope guard (hard):** test/CI only. NO changes to `core/db`, the SQL functions, the Go handlers, or
`core/api/schema`. `git diff -- core/db core/api/*.go core/api/schema` must be empty. New files live in
`ci/` and `.github/workflows/`; `Makefile` and `docs/open-spec-items.md` are edited.

## The two directions
1. **Payload → schema.** Generate REAL payloads by calling the actual JSON functions
   (`fn_actor_page`, `fn_location_page`, `fn_artifact_page`, `fn_timeline`, `fn_compendium_index_json`)
   for the seeded entities, as BOTH the Player and Jonas viewers, and validate each non-null payload
   against its published schema (matched by the payload's own `schema_version`). Catches
   over-tightening and structural drift. *(Alone, this would NOT have caught SPEC-010 — a non-null
   payload passes a nullable schema.)*
2. **DDL → schema nullability.** For every payload field sourced from a NOT NULL DDL column, assert the
   schema types it non-nullable. Concretely: every schema carrying `confidence` types it `"number"`,
   never `["number","null"]`. Also assert the genuinely-nullable fields (`perceived_name`,
   `group_label`, `display_label`) stay nullable (the over-tighten guard). This is the direction that
   catches the SPEC-010 class.

## Negative teeth-test
A `--selftest` mode flips `confidence` to nullable in each confidence-bearing schema (in memory) and
asserts direction (2) **fails** on the mutant — proving the check isn't vacuously green.

## Files
- `ci/gen_payloads.sh` — host bash; uses `docker compose exec -T db psql` to call the SQL functions
  for every seeded entity × {Player, Jonas} and write each non-null payload to `ci/.payloads/`.
- `ci/schema_contract.py` — pure file-based validator (schema dir + payload dir): direction 1
  (jsonschema Draft-07) + direction 2 (DDL nullability) + `--selftest` teeth. Runs in a
  `python:3.12-slim` container (docker is the sanctioned env) so no host Python deps are needed.
- `Makefile` — `schema-contract` target: gen payloads → run validator + selftest in the python
  container. Needs a seeded db (run `make reset` first, like `make test`).
- `.github/workflows/schema-contract.yml` — new workflow mirroring `invariants.yml`
  (build db → `make reset` → `make schema-contract`).
- `docs/open-spec-items.md` — mark SPEC-011 landed.
- `ci/.payloads/` is a build artifact — gitignored.

## Verify
- `make reset && make schema-contract` → PASS (dir1 + dir2 green) and `SELFTEST OK` (teeth bite).
- Demonstrate dir2 bites by hand (mutate a schema's confidence → check FAILS → revert).
- `git diff -- core/db core/api/*.go core/api/schema` is EMPTY.
- One focused PR to `main`.
