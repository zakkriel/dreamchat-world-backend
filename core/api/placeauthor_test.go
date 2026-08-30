package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Rung 2 (the Journey ladder) / Task 8 — "the world builds the road it needs" (design §4.6, R2/R4).
// Exercised against the real seeded Drowned Lantern play world (make reset precedes go test in the
// battery — matches journey_test.go's own wtOrchestrator/dlWorldID/dlKadeID/wtTavernID pattern).
// paHarborQuarterID is the seeded parent region ("Harbor Quarter of Vael") every room in the tavern
// scene sits inside — the frame every travel journey started from the tavern resolves to
// (startJourney reads it off the actor's own current location's attrs.parent_location_id).
const paHarborQuarterID = "210c0000-0000-0000-0000-0000000000d0"

// paCellarID is the seeded "Cellar" location (seed_drowned_lantern.sql) — connected to the tavern by
// the Cellar Hatch, closed AND locked: the seed's own worked example of an existing barred way.
const paCellarID = "210c0000-0000-0000-0000-0000000000d4"

// paCreateLocation inserts a fresh, UNCONNECTED, area-less location fixture as a child of parentID at
// (x,y) IN parentID's own frame — a real registered location a journey can target or discover, without
// any of the seed's existing portal wiring. Registers t.Cleanup to remove it.
func paCreateLocation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, name, parentID string, x, y float64) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO entity_registry (world_id, entity_kind, canonical_name) VALUES ($1,'location',$2) RETURNING entity_id`,
		worldID, name).Scan(&id); err != nil {
		t.Fatalf("paCreateLocation: registry: %v", err)
	}
	attrs, err := json.Marshal(map[string]any{
		"parent_location_id": parentID,
		"coordinates":        map[string]float64{"x": x, "y": y},
	})
	if err != nil {
		t.Fatalf("paCreateLocation: marshal attrs: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO location_state (entity_id, world_id, attrs) VALUES ($1,$2,$3)`, id, worldID, string(attrs)); err != nil {
		t.Fatalf("paCreateLocation: state: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM location_state WHERE entity_id=$1`, id)
		_, _ = pool.Exec(ctx, `DELETE FROM entity_registry WHERE entity_id=$1`, id)
	})
	return id
}

// paSetLocationArea gives an EXISTING location fixture an attrs.area outline (a rectangle from
// (x1,y1) to (x2,y2)) — the "known place" test's own fixture, so fn_place_at can find it.
func paSetLocationArea(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, locID string, x1, y1, x2, y2 float64) {
	t.Helper()
	area, err := json.Marshal(map[string]any{
		"points": []map[string]float64{
			{"x": x1, "y": y1}, {"x": x2, "y": y1}, {"x": x2, "y": y2}, {"x": x1, "y": y2},
		},
	})
	if err != nil {
		t.Fatalf("paSetLocationArea: marshal: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE location_state SET attrs = jsonb_set(attrs, '{area}', $1::jsonb) WHERE entity_id=$2 AND world_id=$3`,
		string(area), locID, worldID); err != nil {
		t.Fatalf("paSetLocationArea: %v", err)
	}
}

// paLocationCount counts location-kind entities registered for worldID — the "did the world build
// scenery" check every named test in this file compares before/after.
func paLocationCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM entity_registry WHERE world_id=$1 AND entity_kind='location'`, worldID).Scan(&n); err != nil {
		t.Fatalf("paLocationCount: %v", err)
	}
	return n
}

// paPortalConnection reports whether a Portal artifact already connects aID<->bID, and if so whether
// it is open+unlocked — the same shape connectionBetween itself reads, so tests can assert on it
// independently of the implementation under test.
func paPortalConnection(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, aID, bID string) (found, open, locked bool) {
	t.Helper()
	err := pool.QueryRow(ctx, `
		SELECT true, COALESCE(attrs->>'open','false')='true', COALESCE(attrs->>'locked','true')='true'
		FROM artifact_state
		WHERE world_id=$1 AND attrs->'connects' ? $2 AND attrs->'connects' ? $3
		LIMIT 1`, worldID, aID, bID).Scan(&found, &open, &locked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, false
		}
		t.Fatalf("paPortalConnection: %v", err)
	}
	return found, open, locked
}

// paPortalCount counts every Portal artifact connecting aID<->bID — used to prove a locked existing
// way was NOT duplicated/routed around (still exactly one).
func paPortalCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, aID, bID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM artifact_state WHERE world_id=$1 AND attrs->'connects' ? $2 AND attrs->'connects' ? $3`,
		worldID, aID, bID).Scan(&n); err != nil {
		t.Fatalf("paPortalCount: %v", err)
	}
	return n
}

// paSpread derives a spatial offset from baseTick — large enough to move a fixture's coordinates well
// clear of anything an EARLIER run of the SAME test minted nearby (baseTick strictly increases run to
// run within one `go test` invocation and across repeats without an intervening `make reset` — the
// acceptance battery runs the suite twice with none between). Without this, a fixed-coordinate fixture
// re-run without reset lands its interpolated point back inside the FIRST run's own leftover created
// place (never cleaned up — canon persists), and fn_place_at correctly, but misleadingly for the test,
// reuses it instead of minting a fresh one.
func paSpread(baseTick int64) float64 {
	return float64(baseTick % 100000)
}

// Nothing happens on a quiet leg: the world does not build scenery nobody is looking at (R2 — "nothing
// is built while you walk" falls straight out of runJourneyLeg's own firedMag != "" gate).
func TestJourneyLeg_QuietLegCreatesNothing(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // nothing must fire this leg — no stray roll may build.

	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = NewFakePlaceAuthorDriver()
	baseTick := wtBaseTick(t, ctx, pool)

	before := paLocationCount(t, ctx, pool, dlWorldID)

	attempt := Attempt{Type: "ActorMoved", Stated: "I walk out to Dock Street", ToTargetID: jrDockStreetID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	leg := &BeatOutcome{}
	if err := orc.runJourneyLeg(ctx, j, leg, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, leg.Committed) })

	if leg.HaltReason != "journey_leg" {
		t.Fatalf("HaltReason = %q, want journey_leg (leg 1 of a multi-leg trip must not resolve yet)", leg.HaltReason)
	}
	if got := paLocationCount(t, ctx, pool, dlWorldID); got != before {
		t.Fatalf("location-kind entity count = %d, want unchanged %d — a quiet leg must never build scenery", got, before)
	}
}

// The world needs a stage and none contains the point → it creates one, WITH its connections, and the
// traveller actually arrives there (design §4.6 steps 1-5, R2/R4).
func TestJourneyLeg_EruptionOnOpenRoadCreatesThePlaceAndItsWays(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // determinism: only the forced pending row below may fire — an unforced roll at high accumulated ticks must never interfere.

	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = NewFakePlaceAuthorDriver()
	baseTick := wtBaseTick(t, ctx, pool)

	// A fresh, distant, UNCONNECTED goal — far enough that leg 1 alone cannot arrive, and with no
	// existing portal to anything (so creating the bridging place is the only lawful way through).
	// The Y offset (paSpread) keeps repeated same-session runs (no reset between) from landing on a
	// PRIOR run's own leftover created place.
	goalID := paCreateLocation(t, ctx, pool, dlWorldID, "Fish Market (eruption test)", paHarborQuarterID, 900, 200+paSpread(baseTick))

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })

	attempt := Attempt{Type: "ActorMoved", Stated: "I walk the coast road toward the fish market", ToTargetID: goalID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })
	if j.LegsTotal < 2 {
		t.Fatalf("LegsTotal = %d, want >=2 (the fixture must be far enough that leg 1 alone cannot arrive)", j.LegsTotal)
	}

	// A due pending row inside leg 1's window forces the world's turn to fire — small magnitude, so it
	// never itself ends the journey (eruptionCutsBeat("small") is false); the CREATION is what's under
	// test, not the interruption path.
	pendingAttempt := `{"type":"Communicated","stated":"gulls scatter off the coast road","listener_id":"` + dlKadeID + `","content":"just gulls"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick+1, "small", wtMaraID, pendingAttempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := paLocationCount(t, ctx, pool, dlWorldID)

	leg := &BeatOutcome{}
	if err := orc.runJourneyLeg(ctx, j, leg, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, leg.Committed) })

	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired (the world's turn must have fired for creation to trigger)", status)
	}
	if leg.HaltReason != "journey_leg" {
		t.Fatalf("HaltReason = %q, want journey_leg", leg.HaltReason)
	}
	if got := paLocationCount(t, ctx, pool, dlWorldID); got != before+1 {
		t.Fatalf("location-kind entity count = %d, want %d — exactly one new place minted", got, before+1)
	}
	if j.StageID == "" || j.StageID == wtTavernID || j.StageID == goalID {
		t.Fatalf("StageID = %q, want a freshly minted place distinct from the tavern and the goal", j.StageID)
	}

	var descriptor string
	if err := pool.QueryRow(ctx, `SELECT canonical_name FROM entity_registry WHERE entity_id=$1`, j.StageID).Scan(&descriptor); err != nil {
		t.Fatalf("read created place: %v", err)
	}
	if !strings.HasPrefix(descriptor, fakePlaceAuthorDescriptorPrefix) {
		t.Fatalf("created place descriptor = %q, want the seat's authored descriptor", descriptor)
	}

	if found, open, locked := paPortalConnection(t, ctx, pool, dlWorldID, wtTavernID, j.StageID); !found || !open || locked {
		t.Fatalf("tavern<->new place portal: found=%v open=%v locked=%v, want an OPEN way in", found, open, locked)
	}
	if found, open, locked := paPortalConnection(t, ctx, pool, dlWorldID, j.StageID, goalID); !found || !open || locked {
		t.Fatalf("new place<->goal portal: found=%v open=%v locked=%v, want an OPEN way onward", found, open, locked)
	}
	if got := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); got != j.StageID {
		t.Fatalf("Kade's location = %q, want the new place %q — the traveller actually stands there now", got, j.StageID)
	}
}

// A known place containing the point is USED, never duplicated — this is what stops the map filling
// with hamlets every time anyone walks the road (rung 1's whole reason places gained areas).
func TestJourneyLeg_KnownPlaceContainingThePointIsReused(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // a stray roll at a DIFFERENT scene would wrongly fail the fixed world-actor fake — only the forced pending row below should fire.

	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = NewFakePlaceAuthorDriver()
	baseTick := wtBaseTick(t, ctx, pool)

	// A distinct VERTICAL corridor (tavern {200,200} -> goal {200,900+offset}) — deliberately NOT along
	// the horizontal road the other tests in this file walk (y~200, x 200->900), so this fixture's own
	// waystation area can never spatially collide with a place ANOTHER test already minted nearby
	// (entities created by other tests in this same run are never cleaned up — canon persists). The Y
	// offset (paSpread) additionally keeps repeated same-session runs (no reset between — the
	// acceptance battery runs the suite twice) from re-finding a PRIOR run's own waystation fixture.
	goalY := 900 + paSpread(baseTick)
	goalID := paCreateLocation(t, ctx, pool, dlWorldID, "Northgate (reuse test)", paHarborQuarterID, 200, goalY)
	// The waystation is created bare (no area yet) — its area is set below, AFTER startJourney reports
	// the real span/legs, from the EXACT leg-1 point (legSliceSeconds's own formula, not an estimate) —
	// precise regardless of how far the offset above pushed the goal.
	waystationID := paCreateLocation(t, ctx, pool, dlWorldID, "Waystation (reuse test)", paHarborQuarterID, 200, 340)

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })

	attempt := Attempt{Type: "ActorMoved", Stated: "I walk the coast road toward the salt quay", ToTargetID: goalID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	// The EXACT leg-1 point: legSliceSeconds(j) is the identical formula runJourneyLeg itself will use
	// (j.LegsDone/j.CurrentTick are both still at their startJourney values here), interpolated linearly
	// along the SAME straight corridor (origin {200,200} -> goal {200,goalY}) journeyScene itself walks.
	// A generous ±100 pad absorbs the small path-vs-frameCoord difference (fn_move_duration_actor's own
	// distance uses Kade's exact seeded offset within the tavern, not the tavern's bare coordinate).
	leg1Progress := float64(legSliceSeconds(j)) / float64(j.SpanSeconds)
	leg1Y := 200 + leg1Progress*(goalY-200)
	paSetLocationArea(t, ctx, pool, dlWorldID, waystationID, 100, leg1Y-100, 300, leg1Y+100)

	pendingAttempt := `{"type":"Communicated","stated":"gulls scatter off the coast road","listener_id":"` + dlKadeID + `","content":"just gulls"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick+1, "small", wtMaraID, pendingAttempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := paLocationCount(t, ctx, pool, dlWorldID)

	leg := &BeatOutcome{}
	if err := orc.runJourneyLeg(ctx, j, leg, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, leg.Committed) })

	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired (the world's turn must have run for reuse to be a real test)", status)
	}
	if j.StageID != waystationID {
		t.Fatalf("StageID = %q, want the already-known waystation %q — containment resolves to it directly", j.StageID, waystationID)
	}
	if got := paLocationCount(t, ctx, pool, dlWorldID); got != before {
		t.Fatalf("location-kind entity count = %d, want unchanged %d — a known place is reused, never duplicated", got, before)
	}
	if found, _, _ := paPortalConnection(t, ctx, pool, dlWorldID, wtTavernID, waystationID); found {
		t.Fatalf("a portal to the waystation was minted, want none — reusing a known place needs no new way")
	}
}

// Creation fills gaps only (R4): an existing shut or locked way is obeyed, never replaced by a
// bridging place that would route around it.
func TestJourneyLeg_LockedWayIsNotRoutedAround(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // determinism: only the forced pending row below may fire.

	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = NewFakePlaceAuthorDriver()
	baseTick := wtBaseTick(t, ctx, pool)

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })

	// The Cellar (seed_drowned_lantern.sql) is already connected to the tavern by the Cellar Hatch —
	// closed AND locked, the first Tier-1 lock in play. No area exists anywhere along the short
	// tavern->cellar stretch, so the point is genuinely "standing nowhere" every leg.
	attempt := Attempt{Type: "ActorMoved", Stated: "I try the cellar hatch", ToTargetID: paCellarID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	pendingAttempt := `{"type":"Communicated","stated":"a floorboard creaks","listener_id":"` + dlKadeID + `","content":"just the old boards"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick+1, "small", wtMaraID, pendingAttempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := paLocationCount(t, ctx, pool, dlWorldID)
	beforeWays := paPortalCount(t, ctx, pool, dlWorldID, wtTavernID, paCellarID)

	leg := &BeatOutcome{}
	if err := orc.runJourneyLeg(ctx, j, leg, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, leg.Committed) })

	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired (the barred case only exists once the world tries to act)", status)
	}
	if leg.HaltReason != "journey_barred" {
		t.Fatalf("HaltReason = %q, want journey_barred — an existing locked way must end the journey, not be routed around", leg.HaltReason)
	}
	if j.Status != "ended" {
		t.Fatalf("Journey.Status = %q, want ended", j.Status)
	}
	if got := paLocationCount(t, ctx, pool, dlWorldID); got != before {
		t.Fatalf("location-kind entity count = %d, want unchanged %d — a barred road mints no bridging place", got, before)
	}
	if got := paPortalCount(t, ctx, pool, dlWorldID, wtTavernID, paCellarID); got != beforeWays {
		t.Fatalf("portal count tavern<->cellar = %d, want unchanged %d — the lock was obeyed, not duplicated around", got, beforeWays)
	}
	if found, open, locked := paPortalConnection(t, ctx, pool, dlWorldID, wtTavernID, paCellarID); !found || open || !locked {
		t.Fatalf("the ORIGINAL Cellar Hatch: found=%v open=%v locked=%v, want unchanged closed+locked", found, open, locked)
	}
	if got := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); got != startLoc {
		t.Fatalf("Kade's location = %q, want unchanged %q — a barred journey never moves the traveller", got, startLoc)
	}
}

// The seat never emits geometry: the engine draws the outline from the authored size class alone.
func TestPlaceAuthor_SeatSuppliesClassEngineDrawsOutline(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = NewFakePlaceAuthorDriver()
	baseTick := wtBaseTick(t, ctx, pool)

	goalID := paCreateLocation(t, ctx, pool, dlWorldID, "Salt Quay (seat test)", paHarborQuarterID, 900, 300)

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })

	attempt := Attempt{Type: "ActorMoved", Stated: "I walk toward the salt quay", ToTargetID: goalID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	// The offset (paSpread) keeps repeated same-session runs (no reset between) from landing this
	// manually-chosen point back inside a PRIOR run's own leftover created place.
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
		t.Fatalf("authorPlaceForLeg reported barred, want a clean create — no existing connection stands in the way")
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, outcome.Committed) })
	if j.StageID == "" {
		t.Fatalf("StageID not set after creation")
	}

	var descriptor string
	var areaRaw []byte
	if err := pool.QueryRow(ctx,
		`SELECT er.canonical_name, ls.attrs->'area'
		 FROM entity_registry er JOIN location_state ls ON ls.entity_id=er.entity_id AND ls.world_id=er.world_id
		 WHERE er.entity_id=$1`, j.StageID).Scan(&descriptor, &areaRaw); err != nil {
		t.Fatalf("read created place: %v", err)
	}
	if !strings.HasPrefix(descriptor, fakePlaceAuthorDescriptorPrefix) {
		t.Fatalf("descriptor = %q, want the seat's own authored descriptor", descriptor)
	}

	// The engine's own independent computation — fn_extent_class_metres(world,"small") (the fake's
	// fixed extent_class) + fn_area_around(point,radius) — must match the persisted outline EXACTLY.
	// The seat supplied only the class; every number in the outline came from the engine.
	var wantRadius float64
	if err := pool.QueryRow(ctx, `SELECT fn_extent_class_metres($1,'small')`, dlWorldID).Scan(&wantRadius); err != nil {
		t.Fatalf("fn_extent_class_metres: %v", err)
	}
	var wantAreaRaw []byte
	if err := pool.QueryRow(ctx, `SELECT fn_area_around($1::jsonb,$2)`, string(point), wantRadius).Scan(&wantAreaRaw); err != nil {
		t.Fatalf("fn_area_around: %v", err)
	}
	var got, want any
	if err := json.Unmarshal(areaRaw, &got); err != nil {
		t.Fatalf("persisted area not valid json: %v", err)
	}
	if err := json.Unmarshal(wantAreaRaw, &want); err != nil {
		t.Fatalf("expected area not valid json: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("created area = %v, want the engine's own fn_area_around/fn_extent_class_metres output %v", got, want)
	}

	// Belt-and-suspenders on the RAW seat output itself: re-decode it and confirm no geometry field
	// rode along even loosely (the schema is the leash; this proves the fake obeys it structurally).
	raw, err := orc.PlaceAuthor.Generate(ctx, GenRequest{Schema: json.RawMessage(placeAuthorSchemaJSON)})
	if err != nil {
		t.Fatalf("PlaceAuthor.Generate: %v", err)
	}
	var rawFields map[string]any
	if err := json.Unmarshal([]byte(raw), &rawFields); err != nil {
		t.Fatalf("decode raw seat output: %v", err)
	}
	for _, forbidden := range []string{"x", "y", "coordinate", "coordinates", "radius"} {
		if _, ok := rawFields[forbidden]; ok {
			t.Fatalf("seat output carries forbidden geometry field %q", forbidden)
		}
	}
}
