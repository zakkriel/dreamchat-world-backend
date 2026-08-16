package main

import (
	"context"
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// WHY THIS EXISTS. Railway builds and runs the binary; nothing in the deploy path applies
// migrations. So a merged, tested, green schema change reaches production as CODE while the
// database stays where it was, and the mismatch is silent: the service boots, serves every route
// that does not touch the new object, and fails only when a player walks into the one that does.
//
// That is not hypothetical. `20260814170000_hearing_teaches_only_spoken_names.sql` merged on
// 2026-08-15 and deployed the same hour; its migration was never applied, so the naming-wall leak
// it fixes stayed live for a day behind a green pipeline and a SUCCESS deployment. The founder saw
// a character's real name in production while the fix sat in main.
//
// The guard is deliberately a REFUSAL TO BOOT rather than a migrate-on-start. Applying schema
// changes automatically under a rolling deploy means two binary versions racing one database, and
// an ALTER that is safe for the new code is not automatically safe for the old replica still
// serving traffic beside it. Failing closed keeps the schema change a human act with a human's
// timing, and turns the invisible failure into a loud one at the only moment where nothing has
// been served yet.
//
// migrations.txt is the required set, generated from core/db/migrations by `make migrate` and
// diffed by `make schema-check` — the same discipline that keeps schema.sql honest. It lives here
// rather than being read from core/db because go:embed cannot reach outside the module, and a
// runtime directory read would make the check depend on image layout it cannot verify.

//go:embed migrations.txt
var migrationsManifest string

// rowQuerier is the subset of pgxpool.Pool used to read schema_migrations, mirroring dbQuerier
// (orchestrator.go).
type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// requiredMigrations is the version list this binary was built against, ascending.
func requiredMigrations() []string {
	out := make([]string, 0, 64)
	for _, line := range strings.Split(migrationsManifest, "\n") {
		if v := strings.TrimSpace(line); v != "" {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// appliedMigrations reads what the database actually carries.
func appliedMigrations(ctx context.Context, q rowQuerier) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0, 64)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		out = append(out, strings.TrimSpace(v))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	return out, nil
}

// diffMigrations compares what the binary requires against what the database carries. Pure, so the
// decision this guard makes is testable without a database or a faked driver.
//
// Extra versions in the database are NOT an error: that is a rollback, where the database is ahead
// of the binary, and refusing to boot would remove the operator's ability to roll back at all —
// the wrong thing to take away during an incident. It is reported so the log says why they differ.
func diffMigrations(applied []string) (missing []string, extra []string) {
	have := make(map[string]struct{}, len(applied))
	for _, v := range applied {
		have[strings.TrimSpace(v)] = struct{}{}
	}

	want := make(map[string]struct{}, len(have))
	for _, v := range requiredMigrations() {
		want[v] = struct{}{}
		if _, ok := have[v]; !ok {
			missing = append(missing, v)
		}
	}
	for v := range have {
		if _, ok := want[v]; !ok {
			extra = append(extra, v)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}

// schemaDriftError is the boot-time message. It names every missing version and the exact command
// that fixes it, because a refusal that does not say how to clear itself just moves the outage.
func schemaDriftError(missing []string) error {
	return fmt.Errorf(
		"database is missing %d migration(s) this binary requires: %s\n"+
			"    The deploy does not apply migrations. Apply them against this DATABASE_URL, then restart:\n"+
			"        dbmate --migrations-dir core/db/migrations up\n"+
			"    Refusing to serve: the code expects schema objects the database does not have",
		len(missing), strings.Join(missing, ", "))
}
