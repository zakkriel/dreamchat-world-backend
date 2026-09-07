package main

import (
	"context"
	"strings"
	"testing"
)

// A CANON EVENT NOBODY SAW IS LEGITIMATE, AND THE BELT WAS REFUSING IT.
//
// Founder, 2026-09-04: "a car getting on fire and blowing up alone.. it can happen
// and nobody is there to watch it. it destroys an item so a canon event needs to be
// there." State changed, so canon must record it or replay breaks (I-1). Nobody
// witnessed it and nobody is hiding anything.
//
// This is not a reversal of the law — ADR-005 already decided it: "One canon event
// fans out to ZERO-to-N perceptions." The belt contradicted its own founding ADR,
// and its doc comment listed "a secret held by nobody" among things "the world
// cannot survive". SPEC-040 recorded that a superseding ADR was needed; that was
// wrong too. ADR-005 permits this already, so removing the refusal is a bug fix.
//
// What does NOT change: the player still earns nothing at genesis. "Nobody knows
// this" and "the player knows this" are different claims, and the second stays
// refused — see the sibling assertion below.
func TestValidate_AnUnwitnessedEventIsAllowed(t *testing.T) {
	doc := authoredWorld(t)
	if len(doc.History) == 0 {
		t.Fatal("fixture authored no history, so this proves nothing")
	}
	doc.History[0].Knowledge = nil

	if err := doc.validate(); err != nil {
		t.Fatalf("an event nobody witnessed must be allowed (ADR-005: zero-to-N): %v", err)
	}
}

// The player floor is a different rule and it stays. Without this, deleting the
// knowledge-list refusal could be mistaken for permission to hand the player
// unearned knowledge.
func TestValidate_ThePlayerStillEarnsNothingAtGenesis(t *testing.T) {
	doc := authoredWorld(t)
	if len(doc.History) == 0 {
		t.Fatal("fixture authored no history")
	}
	doc.History[0].Knowledge[0].Holder = doc.Arrival.CanonicalName

	err := doc.validate()
	if err == nil {
		t.Fatal("the player was handed knowledge of an event they were not at; that must be refused")
	}
	if !strings.Contains(err.Error(), "know nothing yet") {
		t.Errorf("refusal = %q, want it to say the player knows nothing yet", err.Error())
	}
}

// `indirect` is knowledge perceived THROUGH A MEDIUM — a recording, a spell, a
// dream. Founder, 2026-09-04: "like direct but perceived through another medium".
//
// It is what makes an unwitnessed event reachable later without inventing a second
// door into canon: the camera in the garage holds its own perception of the fire,
// and whoever watches the tape acquires theirs `indirect`. Reliability is NOT this
// column's job — perception_record already carries `confidence` and
// `distortion_level`, so a dream is `indirect` with low confidence and a tape is
// `indirect` with high.
//
// Distinct from `told` (a person's mind stands between you and the event) and from
// `inference` (your own reasoning does). A tape is neither: you perceived the thing
// itself, displaced in time.
func TestEpistemicType_IndirectIsAcceptedByTheEngine(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(pool.Close)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	// Behaviour, not constraint text: write one and see whether the engine takes it.
	// A perception must point at a real canon event (source_event_id is NOT NULL with an
	// FK) -- which is the rule that makes this whole design work: nothing but a perception
	// ever links to canon. Borrow any accepted event from the seeded world.
	var worldID, holderID, eventID string
	if err := tx.QueryRow(ctx, `
		SELECT ce.world_id::text, ce.world_id::text, ce.event_id::text
		  FROM canon_event ce WHERE ce.status = 'accepted' LIMIT 1`).
		Scan(&worldID, &holderID, &eventID); err != nil {
		t.Fatalf("no accepted canon event in the seeded world: %v", err)
	}

	// acquired_tick > valid_tick on purpose: this is the LATE perception the design needs --
	// learned now, about something that happened long ago. Nothing ties the two columns, which
	// is why there are two of them.
	if _, err := tx.Exec(ctx, `
		INSERT INTO perception_record
		  (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
		SELECT $1::uuid, $2::uuid, $3::uuid, 'the tape shows the car catching fire on its own',
		       'indirect', ce.in_world_tick + 500, ce.in_world_tick
		  FROM canon_event ce WHERE ce.event_id = $3::uuid`,
		worldID, holderID, eventID); err != nil {
		t.Fatalf("the engine refused a perception acquired through a medium: %v", err)
	}

	// And the closed set is still closed -- the addition must not have opened the column.
	if _, err := tx.Exec(ctx, `
		INSERT INTO perception_record
		  (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'x', 'clairvoyant', 1, 1)`,
		worldID, holderID, eventID); err == nil {
		t.Fatal("an invented epistemic type was accepted; the set must stay closed")
	}
}
