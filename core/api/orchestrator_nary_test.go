package main

import (
	"context"
	"encoding/json"
	"testing"
)

// naryBystanderID is a second bystander actor, co-located with actor C (jonasID) at locB.
// It is NOT co-located with actor A (playerID) at locA, so the pre-fix single-anchor
// gather_slice (co_present keyed on p_ids[1] only) never placed it in the slice — the
// PR #26 multi-actor whitelist gap this test locks down.
const naryBystanderID = "e5000000-0000-0000-0000-000000000005"

// TestAdjudicateTwoActorsTwoLocations is the PR #26 blocker regression: two actors at two
// different locations, each with a co-located bystander, adjudicated in ONE combined ruling.
//
// The ruling's two events reference BOTH bystanders (via receiver_variants). With the
// multi-actor co_present union, both bystanders land in the slice → verdictRuling passes on
// the first try → exactly one Generate call, no bounce, both events commit.
//
// Under the old p_ids[1] anchoring, B2 (at actor C's location) is absent from co_present, so
// verdictRuling flags its receiver_id, the repair retry re-emits the same ruling, and the beat
// bounces — two Generate calls, zero commits. That is the recorded failure mode.
func TestAdjudicateTwoActorsTwoLocations(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	seedOrchestratorEntities(t, ctx)

	// Register the second bystander (co-located with actor C at locB).
	if _, err := pool.Exec(ctx,
		`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		 VALUES ($1, $2, 'actor', 'nary-bystander-B2')
		 ON CONFLICT (entity_id) DO NOTHING`,
		naryBystanderID, worldID); err != nil {
		t.Fatalf("register bystander B2: %v", err)
	}

	bt := nextBaseTick(t, ctx)
	// Actor A (player) + bystander B1 (mara) at locA; actor C (jonas) + bystander B2 at locB.
	seedActorAtLoc(t, ctx, playerID, locA, bt, 0)
	seedActorAtLoc(t, ctx, maraID, locA, bt, 1)
	seedActorAtLoc(t, ctx, jonasID, locB, bt, 2)
	seedActorAtLoc(t, ctx, naryBystanderID, locB, bt, 3)

	// One combined ruling: two AttributeChanged events (one per actor), each carrying a
	// receiver_variant for its own co-located bystander. Both bystander ids must be
	// whitelisted (via the co_present union) for verdictRuling to pass on the first try.
	r := map[string]any{
		"reasoning": "Both actors act at once; each is witnessed by their own co-located bystander.",
		"therefore": "succeeds",
		"outcome": map[string]any{
			"kind": "resolved",
			"events": []map[string]any{
				{
					"type":      "AttributeChanged",
					"actor_id":  playerID,
					"target_id": jonasID,
					"truth":     "The player leans on Jonas.",
					"visible":   true,
					"receiver_variants": []map[string]any{
						{"receiver_id": maraID, "text": "Mara catches the sharper truth of it."},
					},
				},
				{
					"type":      "AttributeChanged",
					"actor_id":  jonasID,
					"target_id": playerID,
					"truth":     "Jonas pushes back at the player.",
					"visible":   true,
					"receiver_variants": []map[string]any{
						{"receiver_id": naryBystanderID, "text": "B2 catches the sharper truth of it."},
					},
				},
			},
		},
	}
	rJSON, _ := json.Marshal(r)

	// Scripted driver: emits the same ruling on every call. Under the fix it is called
	// exactly once; under the old anchoring it is called twice (repair) then bounces.
	driver := &countingRulingDriver{
		name:    "nary-two-actor-driver",
		rulings: []string{string(rJSON), string(rJSON)},
	}

	orc := &Orchestrator{
		DB:             pool,
		Resolve:        driver,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	// A attempts on C, C attempts on A — different actors, different locations, one beat.
	ar, err := orc.adjudicate(ctx, worldID, []ActorAttempt{
		{ActorID: playerID, Attempt: Attempt{Type: "AttributeChanged", Stated: "I lean on Jonas", TargetID: jonasID}},
		{ActorID: jonasID, Attempt: Attempt{Type: "AttributeChanged", Stated: "I push back at the player", TargetID: playerID}},
	}, nil, bt+4, 0)
	if err != nil {
		t.Fatalf("adjudicate: %v", err)
	}

	// Exactly ONE Generate call — the combined n-ary judgment, no wasted repair retry.
	if driver.callCount != 1 {
		t.Fatalf("driver called %d times, want 1 (multi-actor union whitelists both bystanders — no repair/bounce)", driver.callCount)
	}
	if ar.Halt != "" {
		t.Fatalf("expected no halt, got %q (B2 missing from slice → bounce is the pre-fix failure mode)", ar.Halt)
	}
	if len(ar.Committed) != 2 {
		t.Fatalf("expected 2 committed events, got %d: %v", len(ar.Committed), ar.Committed)
	}

	perceptionSubjectBackfill(t, ctx, pool, int(bt))
}
