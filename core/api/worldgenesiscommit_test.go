package main

// worldgenesiscommit_test.go — proving the commit transaction writes exactly what it decided to write,
// beyond what worldgenesis_test.go's playability tests already exercise.

import (
	"context"
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
		var kind, descriptor string
		if err := tx.QueryRow(ctx, `
			SELECT entity_kind, coalesce(descriptor,'')
			  FROM entity_registry
			 WHERE world_id=$1::uuid AND canonical_name=$2`, worldID, c.CanonicalName).
			Scan(&kind, &descriptor); err != nil {
			t.Fatalf("the concept %q was discarded at commit: %v", c.CanonicalName, err)
		}
		if kind != "concept" {
			t.Errorf("%q entity_kind = %q, want concept", c.CanonicalName, kind)
		}
		// The descriptor IS the truth (design §3): one field, one meaning. The short
		// `descriptor` the fill also writes is redundant with it and is not stored.
		if descriptor != c.WhatItIs {
			t.Errorf("%q descriptor must carry what_it_is verbatim; got %q", c.CanonicalName, descriptor)
		}
	}
}
