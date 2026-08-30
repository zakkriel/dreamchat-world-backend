package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Rung 2 (the Journey ladder) / Task 9 Step 3 — THE RUNG GATE. This file is the acceptance test for
// the whole ladder: the founder's own worked example, end to end, driven through RunBeat — the real
// entry point (beathandler's own caller), never journey.go's internals directly.
//
// RULINGS-2026-07-30 §2, verbatim: "The action spans multiple beats. Each beat, the world acts first
// (its slot): it may telegraph, cut in, or redirect the actor — or do nothing. Nobody acts this slot
// -> the action makes progress and carries to the next slot. ... Across the slots the journey needs,
// the world had multiple chances to stop or redirect the actor. If it never did -> the actor arrives.
// 'It tried to get out and managed to, but there were multiple world-action slots in the middle that
// could have stopped or forced a change of plans -- or not, and it just resolves.'"
//
// Both tests below reuse journey_beat_test.go's own jbOrchestrator-adjacent pattern (wtOrchestrator,
// dlWorldID/dlKadeID/wtTavernID/wtMaraID) rather than the tension-geometry world (11111111-...): that
// world has no world_actor_setting row at all (seed_mara_0A.sql never calls seed_world_defaults), so
// wtDisableWorldActor — which this file's own "world is quiet throughout" requirement depends on --
// cannot run against it. The Drowned Lantern play world is seeded with every table these tests touch.
//
// Geometry note for the interrupted test (load-bearing, read before changing coordinates): a first
// attempt targeted a bare LOCATION fixture reachable by a hand-authored direct Portal from the tavern
// (mirroring seedTensionGeometry's own tenA<->tenD "long way" portal) so the opening over-budget move
// could pass runChain's premiseHolds gate. That combination is UNSOUND against the current
// authorPlaceForLeg (placeauthor.go): its own R4 "is fromID already connected to the ultimate goal?"
// check (the barred-or-not decision) is reused, unchanged, to ALSO decide whether the fromID<->newPlace
// edge gets minted — so a pre-existing direct portal from the origin to the goal (exactly what
// premiseHolds needs to let the walk begin at all) makes that edge get skipped, and the interrupted
// leg's own "traveller's own arrival [onto the new place]" commit then gate_rejects for want of a
// portal from the actor's REAL, still-unmoved canonical location. Verified empirically (scratch probe,
// not committed): the continuing RunBeat call returns a non-nil error out of
// runJourneyLeg/authorPlaceForLeg in exactly that shape. This is a genuine gap in Task 8's own
// bookkeeping, not something a test is licensed to route around by touching journey.go/placeauthor.go
// — see the acceptance report for the full writeup. TestJourney_FoundingRuling_InterruptedThenRestated
// instead stages Kade and the goal in the SAME scene: a fresh, per-run "back room" location (so
// premiseHolds's "here == scene, no traversal" branch needs no portal at all), with the goal a fixed
// artifact planted deep in that room's own local frame. Because origin and goal share one scene,
// there is no pre-existing edge for authorPlaceForLeg's R4 check to collide with, and the interruption
// itself mints the only portal this scenario ever needs (the back room <-> the freshly authored
// place) — exactly the pair both the interrupted leg's own arrival commit AND the later restart's own
// premiseHolds check need. The room is FRESH per run (paSpread-offset coordinates, journey_beat_test.go
// precedent) rather than reusing the tavern directly: the interpolated point a same-scene journey walks
// toward is degenerate (always the room's own outer coordinate, never the artifact's inner one — see
// journeyScene/frameCoord in journey.go), so a fixed, shared point would let a PRIOR run's own
// leftover created place (created places persist forever, R2) get recognized as "known" on a later
// run and silently skip creating a new one — turning this from a create-and-move leg into a mere
// recognize leg, which commits nothing and would make the "actor moved off the start" assertion below
// fail exactly the way the first (portal-collision) attempt did, just later and by a different route.
// Both failure modes are reported in full, not just the one this file ended up avoiding.

// jaOpenPortal creates a fresh, OPEN, unlocked Portal artifact directly connecting aID<->bID -- the
// same shape seedTensionGeometry's own hand-authored portals use. Used only by the quiet-road test
// (TestJourney_FoundingRuling_QuietRoadArrives), which never reaches authorPlaceForLeg (the world
// stays disabled end to end), so the R4/exists interaction documented above never triggers.
func jaOpenPortal(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, aID, bID, name string) {
	t.Helper()
	var portalID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO entity_registry (world_id, entity_kind, canonical_name) VALUES ($1,'artifact',$2) RETURNING entity_id`,
		worldID, name).Scan(&portalID); err != nil {
		t.Fatalf("jaOpenPortal: registry: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO artifact_state (entity_id, world_id, attrs)
		 VALUES ($1,$2, jsonb_build_object('open', true, 'locked', false, 'connects', jsonb_build_array($3::text,$4::text)))`,
		portalID, worldID, aID, bID); err != nil {
		t.Fatalf("jaOpenPortal: state: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM artifact_state WHERE entity_id=$1`, portalID)
		_, _ = pool.Exec(ctx, `DELETE FROM entity_registry WHERE entity_id=$1`, portalID)
	})
}

// jaSetLocationTension stamps locID's CURRENT tension directly on location_state, the same
// direct-projection-table technique orchestrator_worldtime_test.go's wtSetTavernTension already uses
// — needed here because paCreateLocation's own fixture is bare (no tension key at all), which would
// otherwise read as 'none' (infinite budget, tension.go's own documented default) and never halt.
func jaSetLocationTension(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, locID, tension string) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`UPDATE location_state SET attrs = jsonb_set(attrs, '{tension}', to_jsonb($1::text)) WHERE entity_id=$2 AND world_id=$3`,
		tension, locID, worldID); err != nil {
		t.Fatalf("jaSetLocationTension(%s): %v", tension, err)
	}
}

// jaCreateFarCorner registers a fixed artifact fixture inside locID's own local frame, far from
// wherever the traveller starts -- the "far corner of the room" goal TestJourney_FoundingRuling_
// InterruptedThenRestated walks toward. See the file doc comment for why this is inside the actor's
// OWN current scene rather than a distinct, portal-connected location.
func jaCreateFarCorner(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, name, locID string, x, y float64) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO entity_registry (world_id, entity_kind, canonical_name) VALUES ($1,'artifact',$2) RETURNING entity_id`,
		worldID, name).Scan(&id); err != nil {
		t.Fatalf("jaCreateFarCorner: registry: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO artifact_state (entity_id, world_id, attrs)
		 VALUES ($1,$2, jsonb_build_object('location_id',$3::text,'coordinates',jsonb_build_object('x',$4::float8,'y',$5::float8)))`,
		id, worldID, locID, x, y); err != nil {
		t.Fatalf("jaCreateFarCorner: state: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM artifact_state WHERE entity_id=$1`, id)
		_, _ = pool.Exec(ctx, `DELETE FROM entity_registry WHERE entity_id=$1`, id)
	})
	return id
}

// jaDeleteJourneys is the Constraints-mandated cleanup: EVERY journey row this file's tests create,
// by (world_id, actor_id), regardless of status -- an 'arrived' or 'ended' row is just as capable of
// tripping idx_journey_one_active's own successor as a leaked 'active' one would be for THIS actor
// specifically, and a stray row of any status is noise the next reader of this table should not have
// to explain (Task 7's own leaked-row lesson, restated in this rung's Constraints).
func jaDeleteJourneys(t *testing.T, pool *pgxpool.Pool, worldID, actorID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`DELETE FROM journey WHERE world_id=$1 AND actor_id=$2`, worldID, actorID); err != nil {
		t.Errorf("jaDeleteJourneys: %v", err)
	}
}

// TestJourney_FoundingRuling_QuietRoadArrives is RULINGS-2026-07-30 §2's first half: "nobody acts
// this slot -> the action makes progress and carries to the next slot... if it never did -> the actor
// arrives." A tense scene (30 s budget) and a walk that cannot fit it (200 m, ~143 s of real physics)
// must still BEGIN rather than bounce, and — with the world quiet end to end — must carry the actor
// all the way to the goal over a bounded run of presses (R7: 5-10, whatever the trip's length).
func TestJourney_FoundingRuling_QuietRoadArrives(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // the world's whole turn is off: nothing may ever fire.
	wtSetTavernTension(t, ctx, pool, "tense")    // 30 s budget — an earlier test in this run may have left a different value (wtSetTavernTension itself never restores).

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	// A fresh location 200 m from the tavern (~143 s of walk, comfortably over the 30 s budget), with
	// its own direct, open, unlocked Portal — a real door out, exactly the founder's "you tried to
	// leave" premise. The world stays quiet for the whole test, so authorPlaceForLeg is never reached
	// and the R4/exists interaction the file doc comment describes never applies here.
	goalID := paCreateLocation(t, ctx, pool, dlWorldID, "The Far Landing (quiet-road acceptance test)", paHarborQuarterID, 400, 200)
	jaOpenPortal(t, ctx, pool, dlWorldID, wtTavernID, goalID, "the coast door (quiet-road acceptance test)")

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })
	t.Cleanup(func() { jaDeleteJourneys(t, pool, dlWorldID, dlKadeID) })
	var allCommitted []string
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, allCommitted) })

	walk := Attempt{Type: "ActorMoved", Stated: "I walk the coast road to the far landing", ToTargetID: goalID}

	// ONE beat. The walk does not fit, so it becomes a journey — and the journey now runs its legs
	// back to back rather than waiting to be clicked through (2026-08-28). On a quiet road that means
	// it arrives inside this beat: no presses, no active row left behind.
	outcome, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, []Attempt{walk}, baseTick+1, nil)
	if err != nil {
		t.Fatalf("RunBeat (the walk): %v", err)
	}
	allCommitted = append(allCommitted, outcome.Committed...)
	if outcome.HaltReason == "turn_budget" {
		t.Fatalf("halt_reason = turn_budget — the founding ruling: an over-budget walk BEGINS, it never bounces")
	}
	if outcome.HaltReason != "journey_arrived" {
		t.Fatalf("halt_reason = %q, want journey_arrived — a quiet road with no obstruction must let the actor arrive in the one beat (RULINGS-2026-07-30 §2)", outcome.HaltReason)
	}

	// Nothing is left waiting for a press.
	if still, err := orc.activeJourney(ctx, dlWorldID, dlKadeID); err != nil {
		t.Fatalf("activeJourney: %v", err)
	} else if still != nil {
		t.Fatalf("a journey is still active after the beat (%s) — the player would be shown a continue button with nothing to continue", still.ID)
	}

	// THE WORLD KEPT EVERY CHANCE IT HAD. R7's 5-10 band is now legs the world rolled on, not clicks
	// the player made: the danger is unchanged, only the clicking is gone.
	var legsTotal, legsDone int
	var status string
	if err := pool.QueryRow(ctx,
		`SELECT legs_total, legs_done, status FROM journey
		   WHERE world_id=$1 AND actor_id=$2 ORDER BY started_tick DESC LIMIT 1`,
		dlWorldID, dlKadeID).Scan(&legsTotal, &legsDone, &status); err != nil {
		t.Fatalf("read back the journey: %v", err)
	}
	if status != "arrived" {
		t.Fatalf("journey.status = %q, want arrived", status)
	}
	if legsTotal < 5 || legsTotal > 10 {
		t.Fatalf("legs_total = %d, want within the founder's 5-10 band (R7)", legsTotal)
	}
	if legsDone != legsTotal {
		t.Fatalf("legs_done = %d, want all %d — every leg must run, because every leg is an interruption roll", legsDone, legsTotal)
	}
	t.Logf("quiet road: arrived in ONE beat over %d legs (span_seconds=%d)", legsTotal, legsTotal)

	if loc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); loc != goalID {
		t.Fatalf("actor_state.attrs.location_id = %q, want the goal %q — 'if it never did -> the actor arrives' (RULINGS-2026-07-30 §2)", loc, goalID)
	}
}

// TestJourney_FoundingRuling_InterruptedThenRestated is RULINGS-2026-07-30 §2's second half, made
// executable: "it tried to get out and managed to, but there were multiple world-action slots in the
// middle that could have stopped or forced a change of plans." The same kind of walk, but a medium
// cut-in is forced (not left to chance) inside a later leg's own window: the journey must END outright
// (R5 — nothing suspends, nothing auto-resumes), leaving the actor standing wherever that leg's world's
// turn found them — genuinely past the start, short of the goal — and R6's "full autonomy" afterward
// must let a restated attempt of the SAME walk actually arrive.
//
// See the file doc comment for the geometry rationale: Kade and the goal both stand in a FRESH,
// per-run "back room" (never the tavern itself, never a distinct portal-connected location) so that
// (a) the opening press needs no portal for premiseHolds — same-scene, no traversal — and (b) the
// interpolated point a same-scene journey walks toward (degenerate: always the room's own outer
// coordinate) cannot collide with any earlier run's own leftover created scenery.
func TestJourney_FoundingRuling_InterruptedThenRestated(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // pressure off — ONLY the pending row scheduled below may ever fire.

	orc := wtOrchestrator(pool)
	orc.PlaceAuthor = NewFakePlaceAuthorDriver() // the interruption lands on open road; the world must be able to author the ground it stands on (R2/R4).
	baseTick := wtBaseTick(t, ctx, pool)

	// A fresh "back room" — a real registered location, child of the same Harbor Quarter every room
	// in this scene sits inside — placed at a per-run-unique coordinate (paSpread, journey_beat_test.go
	// precedent) so this run's own interpolated point never lands on an earlier run's leftover
	// creation. Stamped 'tense' directly (paCreateLocation's own fixture carries no tension key at
	// all, which would otherwise read as the unbounded 'none' default and never halt).
	backRoomID := paCreateLocation(t, ctx, pool, dlWorldID, "a back room off the coast road (interrupted-road acceptance test)",
		paHarborQuarterID, 900, 200+paSpread(baseTick))
	jaSetLocationTension(t, ctx, pool, dlWorldID, backRoomID, "tense")

	// The goal: a fixed point ~980 m across that SAME room's own local frame — a walk that cannot
	// possibly fit a 30 s budget, yet whose resolved SCENE is the back room itself, so the opening
	// press needs no portal at all (premiseHolds's same-scene branch).
	goalID := jaCreateFarCorner(t, ctx, pool, dlWorldID, "the far corner of the back room (interrupted-road acceptance test)", backRoomID, 700, 700)

	trueStartLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, trueStartLoc) })
	jrSetActorLocation(t, ctx, pool, dlWorldID, dlKadeID, backRoomID) // stage Kade in the back room for this test's own "start".
	startLoc := backRoomID

	t.Cleanup(func() { jaDeleteJourneys(t, pool, dlWorldID, dlKadeID) })
	var allCommitted []string
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, allCommitted) })

	walk := Attempt{Type: "ActorMoved", Stated: "I cross the back room toward its far corner", ToTargetID: goalID}

	// The cut-in is scheduled BEFORE the beat now, because the whole journey runs inside one beat.
	// It is placed in LEG 2's own window — the SECOND of the "multiple chances" the ruling describes,
	// so the road is still shown standing on its own for one quiet leg before the world acts on it.
	// Leg 1 covers (start, start+slice]; slice is ceil(span/legs), legSliceSeconds' own first-leg value.
	span, err := orc.fnMoveDurationActor(ctx, dlWorldID, dlKadeID, goalID)
	if err != nil {
		t.Fatalf("move duration: %v", err)
	}
	var legs int64
	if err := pool.QueryRow(ctx, `SELECT fn_journey_legs($1::uuid, $2::bigint)`, dlWorldID, span).Scan(&legs); err != nil {
		t.Fatalf("leg count: %v", err)
	}
	start := baseTick + 1
	slice := (span + legs - 1) / legs
	fireAt := start + slice + 1 // inside leg 2

	pendingAttempt := `{"type":"AttributeChanged","stated":"a stack of crates goes over in the dark","target_id":"` + dlBarID + `"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, fireAt, "medium", wtMaraID, pendingAttempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	interrupted, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, []Attempt{walk}, start, nil)
	if err != nil {
		t.Fatalf("RunBeat (the walk, cut short in leg 2): %v", err)
	}
	allCommitted = append(allCommitted, interrupted.Committed...)

	var journeyID string
	if err := pool.QueryRow(ctx,
		`SELECT journey_id::text FROM journey WHERE world_id=$1 AND actor_id=$2 ORDER BY started_tick DESC LIMIT 1`,
		dlWorldID, dlKadeID).Scan(&journeyID); err != nil {
		t.Fatalf("find the journey the beat started: %v", err)
	}

	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired — the forced medium cut-in must actually have run this leg", status)
	}
	if interrupted.HaltReason != "journey_interrupted" {
		t.Fatalf("halt_reason = %q, want journey_interrupted — R5: a medium (or larger) cut-in ends the journey outright, nothing suspended, nothing auto-resumed", interrupted.HaltReason)
	}

	var journeyStatus, stageID string
	if err := pool.QueryRow(ctx, `SELECT status, COALESCE(stage_id::text,'') FROM journey WHERE journey_id=$1::uuid`, journeyID).
		Scan(&journeyStatus, &stageID); err != nil {
		t.Fatalf("read back the interrupted journey: %v", err)
	}
	if journeyStatus != "ended" {
		t.Fatalf("journey.status = %q, want ended — R5: the journey ends outright, it does not pause", journeyStatus)
	}

	standingAt := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	if standingAt == startLoc {
		t.Fatalf("actor location = %q, unchanged from the start %q — 'it tried to get out and managed to': a real leg ran and the world acted on real ground before the cut-in landed", standingAt, startLoc)
	}
	if standingAt == goalID {
		t.Fatalf("actor location = %q, want short of the goal %q — the cut-in ends the trip before arrival", standingAt, goalID)
	}
	if stageID == "" || standingAt != stageID {
		t.Fatalf("actor location %q != journey.stage_id %q — the ruling's own words: 'the player is standing where it happened', not merely ticking down in place", standingAt, stageID)
	}

	// Full autonomy after the cut-in (R5/R6): the player restates the SAME walk. The ground the
	// interruption authored carries no tension attribute of its own (place_author.v1.schema.json
	// never authors one — only descriptor/kind/extent_class), so beatBudgetSeconds reads it as 'none'
	// (∞, tension.go's own documented default for an unstamped scene) and this restated attempt fits
	// inside a single beat — R2's own "or not, and it just resolves" holding one level below the leg
	// loop, not only across it. The loop below stays generic (bounded, not hardcoded to one shape) so
	// a different geometry that DID need further legs would still be exercised honestly.
	restartBase := wtBaseTick(t, ctx, pool)
	last, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, []Attempt{walk}, restartBase+1, nil)
	if err != nil {
		t.Fatalf("RunBeat (restated walk): %v", err)
	}
	allCommitted = append(allCommitted, last.Committed...)
	if last.HaltReason == "premise_broken" || last.HaltReason == "gate_reject" || last.HaltReason == "turn_budget" {
		t.Fatalf("halt_reason = %q restating the SAME walk from where the interruption left Kade — R6's full autonomy means the player CAN try again and it must not be structurally refused", last.HaltReason)
	}

	t.Logf("interrupted inside leg 2 of one beat; restating resolved with halt_reason=%q", last.HaltReason)
	if last.HaltReason != "completed" && last.HaltReason != "journey_arrived" {
		t.Fatalf("final halt_reason after restating = %q, want completed or journey_arrived — restating the walk must eventually arrive, not dead-end a second time", last.HaltReason)
	}
	// The goal is a fixture inside the back room's own frame (see the file doc comment): arriving AT
	// it resolves, via fn_target_position, to its containing scene — the back room — exactly as
	// arriving at any other fixed feature within a room already does elsewhere in this codebase
	// (e.g. "the bar").
	if loc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); loc != backRoomID {
		t.Fatalf("actor_state.attrs.location_id after restating = %q, want the goal's own scene %q — the founder's own words: 'it just resolves'", loc, backRoomID)
	}
}
