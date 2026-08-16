package main

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The guard's whole value is that it refuses to boot when the database is behind. These pin the
// three outcomes that decide whether it is real: a behind database is named and refused, a current
// one boots, and an AHEAD one boots anyway (a rollback must stay possible).

func TestSchemaDrift_MissingMigrationIsNamedAndRefused(t *testing.T) {
	required := requiredMigrations()
	if len(required) < 2 {
		t.Fatalf("manifest should carry the real migration list, got %d", len(required))
	}
	// Every migration applied EXCEPT the newest — the exact shape production was in when the
	// naming-wall fix shipped as code and the database stayed behind.
	behind := append([]string(nil), required[:len(required)-1]...)
	withheld := required[len(required)-1]

	missing, extra := diffMigrations(behind)
	if len(extra) != 0 {
		t.Fatalf("nothing was invented, want no extras, got %v", extra)
	}
	if len(missing) != 1 || missing[0] != withheld {
		t.Fatalf("want exactly the withheld version %q reported missing, got %v", withheld, missing)
	}

	// The refusal must name the version and the fix, or it just relocates the outage.
	msg := schemaDriftError(missing).Error()
	if !strings.Contains(msg, withheld) {
		t.Errorf("boot error must name the missing version %q: %s", withheld, msg)
	}
	if !strings.Contains(msg, "dbmate") {
		t.Errorf("boot error must name the command that clears it: %s", msg)
	}
}

func TestSchemaDrift_CurrentDatabaseBoots(t *testing.T) {
	missing, extra := diffMigrations(requiredMigrations())
	if len(missing) != 0 || len(extra) != 0 {
		t.Fatalf("a database carrying exactly the manifest must boot clean, got missing=%v extra=%v", missing, extra)
	}
}

// A rollback puts the database AHEAD of the binary. Refusing to boot there would take away the
// operator's ability to roll back during an incident, which is when they need it most.
func TestSchemaDrift_DatabaseAheadStillBoots(t *testing.T) {
	ahead := append(requiredMigrations(), "29990101000000")

	missing, extra := diffMigrations(ahead)
	if len(missing) != 0 {
		t.Fatalf("a database ahead of the binary is not missing anything, got %v", missing)
	}
	if len(extra) != 1 || extra[0] != "29990101000000" {
		t.Fatalf("the newer version must be reported as extra, got %v", extra)
	}
}

// An empty database — the from-zero case — must be refused loudly rather than reported as clean.
func TestSchemaDrift_EmptyDatabaseIsRefused(t *testing.T) {
	missing, _ := diffMigrations(nil)
	if len(missing) != len(requiredMigrations()) {
		t.Fatalf("an unmigrated database is missing everything, got %d of %d",
			len(missing), len(requiredMigrations()))
	}
}

// The embedded manifest is what the binary enforces, so a manifest that has drifted from
// core/db/migrations enforces the wrong set. `make schema-check` guards this in CI; this catches it
// in the unit suite, where it costs nothing.
func TestSchemaManifest_MatchesMigrationsDirectory(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join("..", "db", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Skip("migrations directory not reachable from this working directory")
	}

	onDisk := make([]string, 0, len(entries))
	for _, e := range entries {
		name := filepath.Base(e)
		if i := strings.IndexByte(name, '_'); i > 0 {
			onDisk = append(onDisk, name[:i])
		}
	}
	sort.Strings(onDisk)

	got := requiredMigrations()
	if strings.Join(got, ",") != strings.Join(onDisk, ",") {
		t.Fatalf("core/api/migrations.txt has drifted from core/db/migrations.\n"+
			"run `make migrations-manifest` and commit the result.\nembedded=%d on-disk=%d\nfirst diff: %s",
			len(got), len(onDisk), firstManifestDiff(got, onDisk))
	}
}

func firstManifestDiff(a, b []string) string {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return "embedded " + a[i] + " vs on-disk " + b[i]
		}
	}
	return "one list is a prefix of the other"
}
