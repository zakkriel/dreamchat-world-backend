# ADR-P020: A schema-drifted database is a boot refusal, not a runtime surprise

**Status:** Accepted (2026-08-16)
**Date:** 2026-08-16
**Series:** Platform / Operations (ADR-P###, per D-5) — does NOT touch frozen engine canon.
**Governing rules:** D-9 (evidence once code runs) and D-6 (git `/docs` is the source of truth) —
this ADR records an operational contract the register did not previously cover.
**Owner of decision:** the naming-wall incident (below).
**Evidence (D-9):** `core/api/schemaversion.go`, `core/api/migrations.txt`, the manifest guard in
`Makefile` (`migrate` regenerates, `schema-check` diffs), `core/api/schemaversion_test.go`. Filed
*with* the code, not ahead of it.

## Context

Railway builds the Dockerfile and runs the binary. Nothing in the deploy path applies migrations —
not the Dockerfile, not a release command, not a start hook. Applying them is a separate human act
against `DATABASE_URL`.

So a merged, tested, green schema change reaches production as CODE while the database stays where it
was, and the mismatch is silent: the service boots, serves every route that does not touch the new
object, and fails only when a user walks into the one that does.

That is not hypothetical. `20260814170000_hearing_teaches_only_spoken_names.sql` merged 2026-08-15 and
deployed the same hour; its migration was never applied. The naming-wall leak it fixes stayed live for
a day behind a green pipeline and a SUCCESS deployment, and was found only by reading production:
five `name_knowledge` rows taught by events that never spoke the name, including a holder who learned
"Jonas" from an utterance whose spoken words were "Drink."

## Decision

The API **refuses to serve** a database missing any migration the binary was built against.

1. `core/api/migrations.txt` is the required version list, `//go:embed`-ed into the binary. It is
   generated from `core/db/migrations/` by `make migrate` and diffed by `make schema-check`, the same
   discipline that already keeps `schema.sql` honest. It lives under `core/api/` because `go:embed`
   cannot reach outside the module, and a runtime directory read would depend on image layout the
   binary cannot verify.
2. At boot, `main` compares it against `schema_migrations`. Missing versions → log every one of them,
   name the command that clears it, `exit 1`.
3. A database **AHEAD** of the binary boots and logs the difference. That is a rollback, and refusing
   there would remove the operator's ability to roll back during an incident — the wrong thing to take
   away at the worst moment.

**Refusal, not migrate-on-start.** Applying schema changes automatically under a rolling deploy means
two binary versions racing one database, and an ALTER that is safe for the new code is not
automatically safe for the old replica still serving beside it. Failing closed keeps the schema change
a human act with a human's timing.

## Consequences

- Shipping a migration is now TWO acts and the second is not optional: merge, then apply. The service
  tells you, loudly, if you forget.
- A new migration means `make migrate` and committing BOTH `core/db/schema.sql` and
  `core/api/migrations.txt`. `make schema-check` is the gate.
- This has already fired in production once as designed (`20260815150001`, the art-style column):
  world-api refused to start and named the version instead of 500-ing on the new path.
