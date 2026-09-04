package main

// worldgenesiscommit_test.go — proving the commit transaction writes exactly what it decided to write,
// beyond what worldgenesis_test.go's playability tests already exercise.

import (
	"context"
	"strings"
	"testing"
)

// A world's ideas must survive genesis. Measured 2026-09-02 on the committed Los
// Andantes world: concepts=7 in genesis_doc against 0 rows in entity_registry, so
// every idea the fill authored was discarded at the seam.
func TestCommit_ConceptsAreRegisteredWithTheirTruth(t *testing.T) {
	ctx := context.Background()
	tx, worldID, doc := genesisFixture(t)

	if len(doc.Concepts) == 0 {
		t.Fatal("the fake fill authored no concepts — nothing for this test to prove")
	}

	for _, c := range doc.Concepts {
		// The regression this test guards is distinguishing `descriptor` from `what_it_is`; if the
		// fixture ever stops giving them different strings, a bug that stores the wrong field would
		// pass silently.
		if c.Descriptor == c.WhatItIs {
			t.Fatalf("fixture no longer discriminates: %q has descriptor == what_it_is", c.CanonicalName)
		}

		var entityID, kind, descriptor string
		if err := tx.QueryRow(ctx, `
			SELECT entity_id::text, entity_kind, coalesce(descriptor,'')
			  FROM entity_registry
			 WHERE world_id=$1::uuid AND canonical_name=$2`, worldID, c.CanonicalName).
			Scan(&entityID, &kind, &descriptor); err != nil {
			t.Fatalf("the concept %q was discarded at commit: %v", c.CanonicalName, err)
		}
		if kind != "concept" {
			t.Errorf("%q entity_kind = %q, want concept", c.CanonicalName, kind)
		}
		// The descriptor IS the truth (design §3): one field, one meaning, stored trimmed like every
		// sibling insert. The short `descriptor` the fill also writes is redundant with it and is not
		// stored.
		if want := strings.TrimSpace(c.WhatItIs); descriptor != want {
			t.Errorf("%q descriptor must carry what_it_is (trimmed); got %q, want %q", c.CanonicalName, descriptor, want)
		}

		// No state row: a concept has no position and cannot act (design §3). Structural, not incidental
		// — see loadGenesisIDs, which must never file a concept under ids.things either.
		var stateRows int
		if err := tx.QueryRow(ctx, `
			SELECT (SELECT count(*) FROM actor_state    WHERE entity_id=$1::uuid)
			     + (SELECT count(*) FROM artifact_state WHERE entity_id=$1::uuid)
			     + (SELECT count(*) FROM location_state WHERE entity_id=$1::uuid)`, entityID).Scan(&stateRows); err != nil {
			t.Fatalf("state count for %q: %v", c.CanonicalName, err)
		}
		if stateRows != 0 {
			t.Errorf("%q: a concept has no position and cannot act; got %d state row(s)", c.CanonicalName, stateRows)
		}
	}
}
