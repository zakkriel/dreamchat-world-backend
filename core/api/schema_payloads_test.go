package main

// SPEC-011 gap closure (rung 3, "the connection" follow-up) — world_actor/1 and place_author/1 had NO
// real-payload coverage, so `make schema-contract` exited 1 on two published, unexercised schemas. Both
// are SEAT-CONTRACT schemas (bridge.go's SeatWorldActor/SeatPlaceAuthor: the structured-output LEASH a
// driver's raw response is validated against), not API response envelopes — unlike every projection
// schema (scene_current/2, actor_page/2, ...), their `additionalProperties: false` property set has no
// room for a "schema_version" field (the envelope belongs to what the API RETURNS, never to what a seat
// may EMIT — see bridge_fakes.go's fakeWorldActorDriver/fakePlaceAuthorDriver doc comments for the exact
// shape each fake authors). ci/schema_contract.py's Direction-1 check therefore cannot key these two
// payloads off a "schema_version" field the payload will never carry; it instead recovers the schema id
// from the payload's own FILENAME (sid_from_filename, ci/schema_contract.py) — world_actor_1.json /
// place_author_1.json name themselves after the schema's own $id ("world_actor/1" -> "world_actor_1"),
// so the payload bytes stay byte-identical to what Driver.Generate actually returned. No wrapper, no
// hand-written fixture: a hand-written fixture would only prove the schema matches itself.
//
// Both generators are gated on SEAT_PAYLOAD_DIR (unset in the normal `go test ./...` run, so they skip
// cleanly and write nothing — no stray files, mirrors scenehandler_test.go's
// TestGenSceneCurrentPayloads/SCENE_PAYLOAD_DIR). ci/gen_payloads.sh sets it and runs them explicitly
// against the seeded Drowned Lantern play world.
//
// FAKE: CI stand-in for an undelivered live model (the design has no LLM-free path,
// POST-COMPACTION-RULINGS). Both generators drive the REAL runWorldActor/authorPlaceForLeg call paths
// (fn_world_slice, validateAttemptFields, the commit pipeline; connectionBetween,
// fn_extent_class_metres/fn_area_around) through the fake driver — the identical calls
// TestRunWorldActor_AuthorsWithinSize and TestPlaceAuthor_SeatSuppliesClassEngineDrawsOutline make — and
// capture the EXACT raw string the driver returned via recordingDriver below. The fake's output shape IS
// the seat's contract in CI; a live model would return the identical shape, just not deterministically.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// recordingDriver wraps a real Driver and captures the raw string its LAST Generate() call returned,
// untouched — the exact bytes the seat itself produced, for the two payload generators below to write
// straight to disk. No re-marshal, no wrapper: what schema_contract.py validates is byte-identical to
// what the seat's own commit path decoded.
type recordingDriver struct {
	Driver
	last string
}

func (r *recordingDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	raw, err := r.Driver.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	r.last = raw
	return raw, nil
}

// TestGenWorldActorPayload drives runWorldActor (worldactor.go) end-to-end against the real seeded
// Drowned Lantern play world and writes the fake World Actor driver's raw
// {"actor_id":...,"attempt":{...}} response to world_actor_1.json — the real payload
// world_actor.v1.schema.json validates against.
func TestGenWorldActorPayload(t *testing.T) {
	dir := os.Getenv("SEAT_PAYLOAD_DIR")
	if dir == "" {
		t.Skip("SEAT_PAYLOAD_DIR unset — this generator only runs from ci/gen_payloads.sh")
	}
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	rec := &recordingDriver{Driver: NewFakeWorldActorDriver()}
	orc := wtOrchestrator(pool)
	orc.WorldActor = rec
	baseTick := wtBaseTick(t, ctx, pool)

	if _, _, err := orc.runWorldActor(ctx, dlWorldID, wtTavernID, "medium", baseTick, 0, nil, &BeatOutcome{}, nil); err != nil {
		t.Fatalf("runWorldActor: %v", err)
	}
	if rec.last == "" {
		t.Fatalf("recordingDriver captured no output — runWorldActor never called Generate")
	}
	if err := os.WriteFile(filepath.Join(dir, "world_actor_1.json"), []byte(rec.last), 0o644); err != nil {
		t.Fatalf("write world_actor_1.json: %v", err)
	}
}

// TestGenPlaceAuthorPayload drives authorPlaceForLeg (placeauthor.go) end-to-end against a fresh,
// unconnected goal in the real seeded Drowned Lantern play world and writes the fake Place Author
// driver's raw {"descriptor":...,"kind":...,"extent_class":...} response to place_author_1.json — the
// real payload place_author.v1.schema.json validates against.
func TestGenPlaceAuthorPayload(t *testing.T) {
	dir := os.Getenv("SEAT_PAYLOAD_DIR")
	if dir == "" {
		t.Skip("SEAT_PAYLOAD_DIR unset — this generator only runs from ci/gen_payloads.sh")
	}
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	rec := &recordingDriver{Driver: NewFakePlaceAuthorDriver()}
	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = rec
	baseTick := wtBaseTick(t, ctx, pool)

	// A fresh, distant, UNCONNECTED goal (mirrors TestPlaceAuthor_SeatSuppliesClassEngineDrawsOutline) —
	// paSpread keeps repeated runs (no `make reset` between — the same convention every other test in
	// placeauthor_test.go relies on) from landing back inside an earlier run's own leftover place.
	goalID := paCreateLocation(t, ctx, pool, dlWorldID, "Salt Quay (schema payload gen)", paHarborQuarterID, 900, 300+paSpread(baseTick))

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })

	attempt := Attempt{Type: "ActorMoved", Stated: "I walk toward the salt quay", ToTargetID: goalID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	point, err := json.Marshal(map[string]float64{"x": 500 + paSpread(baseTick), "y": 250})
	if err != nil {
		t.Fatalf("marshal point: %v", err)
	}
	outcome := &BeatOutcome{}
	barred, err := orc.authorPlaceForLeg(ctx, j, wtTavernID, point, baseTick, 0, outcome, nil)
	if err != nil {
		t.Fatalf("authorPlaceForLeg: %v", err)
	}
	if barred {
		t.Fatalf("authorPlaceForLeg reported barred, want a clean create — a fresh unconnected goal must never be barred")
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, outcome.Committed) })

	if rec.last == "" {
		t.Fatalf("recordingDriver captured no output — authorPlaceForLeg never called Generate")
	}
	if err := os.WriteFile(filepath.Join(dir, "place_author_1.json"), []byte(rec.last), 0o644); err != nil {
		t.Fatalf("write place_author_1.json: %v", err)
	}
}
