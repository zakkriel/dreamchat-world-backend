package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Task 4 — the journey unit: spans, legs, thresholds. Exercised against the real seeded Drowned
// Lantern play world (`make reset` precedes go test in the battery — matches
// orchestrator_worldtime_test.go's wtOrchestrator/dlWorldID/wtMaraID/wtTavernID/wtBaseTick pattern;
// lgCanonCount is ledger_test.go's own "did it actually commit" check, reused verbatim). This task
// builds and tests activeJourney/startJourney/legSliceSeconds/thresholdMet/endJourney WITHOUT the
// world's turn or stage resolution — those are Tasks 5/6, deliberately kept out so this unit is
// testable alone.
//
// jrDockStreetID is the "Dock Street" location — the seeded target 7 m from the tavern (5 s of real
// walk physics: seed_drowned_lantern.sql's own worked comment, tavern {200,200} → dock street
// {207,200}, CEIL(7/1.4)=5), the far side of the front door portal.
const jrDockStreetID = "210c0000-0000-0000-0000-0000000000d2"

// jrDeleteJourney removes a journey row (test cleanup). The unique partial index on
// (world_id, actor_id) WHERE status='active' means a leaked active row poisons every LATER test for
// that actor — every test below that writes one registers this via t.Cleanup.
func jrDeleteJourney(t *testing.T, ctx context.Context, pool *pgxpool.Pool, journeyID string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DELETE FROM journey WHERE journey_id=$1`, journeyID); err != nil {
		t.Fatalf("jrDeleteJourney: %v", err)
	}
}

// jrActorLocation reads an actor's current attrs.location_id straight off the projection table.
func jrActorLocation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, actorID string) string {
	t.Helper()
	var loc string
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(attrs->>'location_id','') FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		worldID, actorID).Scan(&loc); err != nil {
		t.Fatalf("jrActorLocation: %v", err)
	}
	return loc
}

// jrSetActorLocation directly overwrites an actor's projected location_id — a test fixture move
// (mirrors wtSetTavernTension's direct-projection-table approach), not a committed event.
func jrSetActorLocation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, actorID, locationID string) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`UPDATE actor_state SET attrs = jsonb_set(attrs, '{location_id}', to_jsonb($1::text)) WHERE world_id=$2 AND entity_id=$3`,
		locationID, worldID, actorID); err != nil {
		t.Fatalf("jrSetActorLocation: %v", err)
	}
}

// A slice is what is LEFT divided by the legs remaining, so rounding never strands progress: the last
// leg always closes the span exactly. (Plan's own literal test body, Task 4 Step 1.)
func TestLegSlice_LastLegClosesTheSpanExactly(t *testing.T) {
	j := &Journey{SpanSeconds: 1000, LegsTotal: 3, LegsDone: 0, StartedTick: 0, CurrentTick: 0}
	total := int64(0)
	for i := range 3 {
		s := legSliceSeconds(j)
		if s <= 0 {
			t.Fatalf("leg %d produced a non-positive slice %d", i, s)
		}
		total += s
		j.LegsDone++
		j.CurrentTick += s
	}
	if total != 1000 {
		t.Fatalf("three legs covered %d seconds, want exactly the 1000-second span", total)
	}
}

// Travel's span is the real physics duration, and starting a journey must NOT move the actor. This
// is the "starting a journey is not an event" claim, checked two ways: canon count is unchanged, and
// Kade's projected location is still the tavern he started in — not the goal he has not walked to yet.
func TestStartJourney_TravelSpanIsTheMoveDurationAndNothingCommits(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	wantDur, err := orc.fnMoveDurationActor(ctx, dlWorldID, dlKadeID, jrDockStreetID)
	if err != nil {
		t.Fatalf("fnMoveDurationActor: %v", err)
	}
	if wantDur <= 0 {
		t.Fatalf("fnMoveDurationActor(Kade, Dock Street) = %d, want a real positive physics duration", wantDur)
	}

	beforeCanon := lgCanonCount(t, ctx, pool, dlWorldID)
	beforeLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)

	attempt := Attempt{Type: "ActorMoved", Stated: "I walk out to Dock Street", ToTargetID: jrDockStreetID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	if j.Kind != "travel" {
		t.Fatalf("Kind = %q, want travel", j.Kind)
	}
	if j.SpanSeconds != wantDur {
		t.Fatalf("SpanSeconds = %d, want the real move duration %d", j.SpanSeconds, wantDur)
	}
	if j.GoalTarget != jrDockStreetID {
		t.Fatalf("GoalTarget = %q, want %q", j.GoalTarget, jrDockStreetID)
	}
	if j.StartedTick != baseTick || j.CurrentTick != baseTick {
		t.Fatalf("StartedTick/CurrentTick = %d/%d, want both %d", j.StartedTick, j.CurrentTick, baseTick)
	}
	if j.LegsTotal < 5 || j.LegsTotal > 10 {
		t.Fatalf("LegsTotal = %d, want the 5..10 band (fn_journey_legs, Task 2)", j.LegsTotal)
	}
	if j.Status != "active" {
		t.Fatalf("Status = %q, want active", j.Status)
	}

	// Nothing committed to canon: starting a journey is not an event.
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != beforeCanon {
		t.Fatalf("canon count = %d, want unchanged %d (startJourney must commit nothing)", got, beforeCanon)
	}
	// Kade has NOT teleported to the goal — he is still exactly where he started.
	if got := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); got != beforeLoc {
		t.Fatalf("Kade's location = %q, want unchanged %q (startJourney must not move the actor)", got, beforeLoc)
	}
}

// A wait's span is the stated seconds, passed through untouched (R13) — never reclassified or
// rounded to a duration_class. The threshold is the absolute tick the wait clears: startTick + the
// stated span, converted exactly once.
func TestStartJourney_WaitSpanIsTheStatedSeconds(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	attempt := Attempt{
		Type:     "AttributeChanged",
		Stated:   "I lie hidden for two hours",
		TargetID: dlKadeID,
		Sustain:  &Sustain{Kind: "for", Seconds: 7200},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	if j.Kind != "wait" {
		t.Fatalf("Kind = %q, want wait", j.Kind)
	}
	if j.SpanSeconds != 7200 {
		t.Fatalf("SpanSeconds = %d, want the stated 7200 (R13 — passed through, not classified)", j.SpanSeconds)
	}
	var th journeyTickThreshold
	if err := json.Unmarshal(j.Threshold, &th); err != nil {
		t.Fatalf("threshold not valid JSON: %v (%s)", err, j.Threshold)
	}
	if th.Kind != "tick" || th.At != baseTick+7200 {
		t.Fatalf("threshold = %+v, want {tick, %d}", th, baseTick+7200)
	}
}

// One active journey per actor: the database refuses a second (the unique partial index on
// (world_id, actor_id) WHERE status='active' — journey.sql, Task 1). This is deliberately enforced
// by the schema rather than left to Go to remember.
func TestStartJourney_SecondActiveJourneyIsRefused(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	first := Attempt{Type: "AttributeChanged", Stated: "I wait", TargetID: dlKadeID, Sustain: &Sustain{Kind: "for", Seconds: 60}}
	j1, err := orc.startJourney(ctx, dlWorldID, dlKadeID, first, baseTick)
	if err != nil {
		t.Fatalf("startJourney (first): %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j1.ID) })

	second := Attempt{Type: "AttributeChanged", Stated: "I wait some more", TargetID: dlKadeID, Sustain: &Sustain{Kind: "for", Seconds: 30}}
	if j2, err := orc.startJourney(ctx, dlWorldID, dlKadeID, second, baseTick); err == nil {
		t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j2.ID) })
		t.Fatalf("a second active journey for the same actor was accepted, want the unique index to refuse it")
	}
}

// A watch resolves on a FACT, checked in SQL with no model involved: until_at flips false→true the
// moment the watched entity's containing scene becomes the watched place, and flips back to false
// were it ever to leave — thresholdMet reads live state, it does not remember.
func TestThresholdMet_UntilAtFlipsWhenTheEntityArrives(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, wtMaraID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, wtMaraID, startLoc) })

	attempt := Attempt{
		Type:    "AttributeChanged",
		Stated:  "I wait until Mara steps onto Dock Street",
		Sustain: &Sustain{Kind: "until_at", EntityID: wtMaraID, PlaceID: jrDockStreetID},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	if j.Kind != "watch" {
		t.Fatalf("Kind = %q, want watch", j.Kind)
	}

	met, err := orc.thresholdMet(ctx, j)
	if err != nil {
		t.Fatalf("thresholdMet (before): %v", err)
	}
	if met {
		t.Fatalf("thresholdMet = true before Mara ever reached Dock Street")
	}

	jrSetActorLocation(t, ctx, pool, dlWorldID, wtMaraID, jrDockStreetID)

	met, err = orc.thresholdMet(ctx, j)
	if err != nil {
		t.Fatalf("thresholdMet (after): %v", err)
	}
	if !met {
		t.Fatalf("thresholdMet = false after Mara arrived at Dock Street, want true")
	}
}

// Task 5 — a leg runs, and the world gets its turn. runJourneyLeg composes legSliceSeconds,
// runWorldTurn (unchanged), and thresholdMet/endJourney into ONE leg. wtDisableWorldActor is used
// wherever a stray pressure-roll eruption would corrupt the assertion under test (arrival, the
// horizon expiry); wtForceTierFires("medium") is used to make the interruption case deterministic.

// The world takes a turn on EVERY leg — this is the ruling's "multiple chances to stop you", and it
// is the reason the journey exists rather than a fast-forward: a pending_event scheduled inside the
// leg's (tickBefore, tickAfter] window must fire during a single runJourneyLeg call.
func TestRunJourneyLeg_FiresTheWorldsTurnEveryLeg(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // pressure off — this test is about the ledger, not a roll.

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	attempt := Attempt{
		Type:     "AttributeChanged",
		Stated:   "I wait quietly at the bar",
		TargetID: dlKadeID,
		Sustain:  &Sustain{Kind: "for", Seconds: 1000},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	tickBefore := j.CurrentTick
	slice := legSliceSeconds(j)
	if slice < 1 {
		t.Fatalf("legSliceSeconds = %d, want >=1 (a pending row at tickBefore+1 must land inside the leg's window)", slice)
	}

	// A due pending_event, exactly the shape ledger_test.go's own fixtures use: Mara speaks to Kade.
	pendingAttempt := `{"type":"Communicated","stated":"a gull screeches overhead","listener_id":"` + dlKadeID + `","content":"just a gull"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, tickBefore+1, "small", wtMaraID, pendingAttempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	outcome := &BeatOutcome{}
	if err := orc.runJourneyLeg(ctx, j, outcome, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, outcome.Committed) })

	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired — the world's turn must run on every leg", status)
	}
	if outcome.HaltReason != "journey_leg" {
		t.Fatalf("HaltReason = %q, want journey_leg (a quiet small-magnitude ledger fire never ends the journey)", outcome.HaltReason)
	}
	if j.Status != "active" {
		t.Fatalf("Journey.Status = %q, want active (one quiet leg of many does not end the journey)", j.Status)
	}
}

// A hard cut-in ends the journey outright (R5) — nothing is suspended, nothing auto-resumes.
func TestRunJourneyLeg_MediumEruptionEndsTheJourney(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtForceTierFires(t, ctx, pool, dlWorldID, "medium")

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	attempt := Attempt{
		Type:     "AttributeChanged",
		Stated:   "I wait quietly at the bar",
		TargetID: dlKadeID,
		Sustain:  &Sustain{Kind: "for", Seconds: 1000},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	outcome := &BeatOutcome{}
	if err := orc.runJourneyLeg(ctx, j, outcome, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, outcome.Committed) })

	if outcome.HaltReason != "journey_interrupted" {
		t.Fatalf("HaltReason = %q, want journey_interrupted (R5 — a hard cut-in ends the journey outright)", outcome.HaltReason)
	}
	if j.Status != "ended" {
		t.Fatalf("Journey.Status (in-memory) = %q, want ended", j.Status)
	}
	var dbStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM journey WHERE journey_id=$1::uuid`, j.ID).Scan(&dbStatus); err != nil {
		t.Fatalf("read back journey.status: %v", err)
	}
	if dbStatus != "ended" {
		t.Fatalf("journey.status in the DB = %q, want ended — nothing is suspended, nothing auto-resumes", dbStatus)
	}
}

// Walking the whole span arrives, and arrival is a real committed move — the actor is actually
// there, not merely flagged 'arrived'.
func TestRunJourneyLeg_LastLegArrivesAndCommitsTheMove(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // no stray eruption may cut the trip short.

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	attempt := Attempt{Type: "ActorMoved", Stated: "I walk out to Dock Street", ToTargetID: jrDockStreetID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })

	var last *BeatOutcome
	for i := range j.LegsTotal {
		leg := &BeatOutcome{}
		if err := orc.runJourneyLeg(ctx, j, leg, nil); err != nil {
			t.Fatalf("runJourneyLeg (leg %d): %v", i, err)
		}
		committed := leg.Committed
		t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, committed) })
		last = leg
		if j.Status != "active" {
			break
		}
	}

	if last == nil || last.HaltReason != "journey_arrived" {
		got := ""
		if last != nil {
			got = last.HaltReason
		}
		t.Fatalf("HaltReason = %q, want journey_arrived", got)
	}
	if j.Status != "arrived" {
		t.Fatalf("Journey.Status = %q, want arrived", j.Status)
	}
	if got := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); got != jrDockStreetID {
		t.Fatalf("actor_state.location_id = %q, want the goal %q — arrival is a committed move, not a status flag", got, jrDockStreetID)
	}
}

// A watch whose horizon expires ends unresolved — nothing waits forever.
func TestRunJourneyLeg_WatchHorizonExpiresUnresolved(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // this proves the horizon, not a stray interruption.

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	// Watch for Mara reaching Dock Street — she never does, so the predicate never flips.
	attempt := Attempt{
		Type:    "AttributeChanged",
		Stated:  "I wait until Mara reaches Dock Street",
		Sustain: &Sustain{Kind: "until_at", EntityID: wtMaraID, PlaceID: jrDockStreetID},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })
	if j.Kind != "watch" {
		t.Fatalf("Kind = %q, want watch", j.Kind)
	}

	legsTotal := j.LegsTotal
	var last *BeatOutcome
	for i := range legsTotal {
		leg := &BeatOutcome{}
		if err := orc.runJourneyLeg(ctx, j, leg, nil); err != nil {
			t.Fatalf("runJourneyLeg (leg %d): %v", i, err)
		}
		committed := leg.Committed
		t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, committed) })
		last = leg
		if j.Status != "active" {
			break
		}
	}

	if last == nil || last.HaltReason != "journey_unresolved" {
		got := ""
		if last != nil {
			got = last.HaltReason
		}
		t.Fatalf("HaltReason = %q, want journey_unresolved (the watch's horizon ran out without the fact)", got)
	}
	if j.Status != "ended" {
		t.Fatalf("Journey.Status = %q, want ended", j.Status)
	}
	if j.LegsDone != legsTotal {
		t.Fatalf("LegsDone = %d, want all %d legs consumed (nothing waits forever)", j.LegsDone, legsTotal)
	}
}

// Task 7 — changing your mind (R6): "the actions are all typed… there is no waiting or loading, so
// the user cannot ever interrupt its own actions while they are being computed unless the world
// interrupts him first. And after an interruption the user has full autonomy." Any real input ends an
// active journey and then runs as an ordinary beat (the test below this one).
//
// This rung once routed an EMPTY chain to "advance one leg", and rung 3 put a POST /beats/continue
// button on it. Both are gone (2026-08-28): a journey runs its own legs, so nothing is ever left
// mid-trip for a press to advance, and an empty chain now means only "the parse produced nothing".

// One leg costs exactly one leg. The slice is the whole of it — no instant floor stacked on top, no
// second world's turn — which is what keeps a ten-leg trip costing ten legs of world time and not
// twenty. legs_done and status pin that the leg neither skipped ahead nor ended the journey early.
func TestRunJourneyLeg_AdvancesByExactlyOneSlice(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // isolate the routing decision from a stray eruption.

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	waitAttempt := Attempt{
		Type:     "AttributeChanged",
		Stated:   "I wait quietly at the bar",
		TargetID: dlKadeID,
		Sustain:  &Sustain{Kind: "for", Seconds: 1000},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, waitAttempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	wantSlice := legSliceSeconds(j) // pure — computed BEFORE the leg mutates j, off the fresh row's own fields.
	if wantSlice <= 0 {
		t.Fatalf("legSliceSeconds = %d, want a positive first-leg slice", wantSlice)
	}

	outcome := &BeatOutcome{}
	if err := orc.runJourneyLeg(ctx, j, outcome, nil); err != nil {
		t.Fatalf("runJourneyLeg: %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, outcome.Committed) })

	if outcome.HaltReason != "journey_leg" {
		t.Fatalf("HaltReason = %q, want journey_leg (a quiet leg never ends the journey)", outcome.HaltReason)
	}

	var legsDone int
	var status string
	var currentTick int64
	if err := pool.QueryRow(ctx, `SELECT legs_done, status, current_tick FROM journey WHERE journey_id=$1::uuid`, j.ID).
		Scan(&legsDone, &status, &currentTick); err != nil {
		t.Fatalf("read back journey: %v", err)
	}
	if legsDone != 1 {
		t.Fatalf("legs_done = %d, want exactly 1 — one leg advances ONE leg, never two", legsDone)
	}
	if status != "active" {
		t.Fatalf("status = %q, want active (one leg of many does not end the journey)", status)
	}
	if currentTick != baseTick+wantSlice {
		t.Fatalf("current_tick = %d, want %d (baseTick + exactly the leg's slice — NOT also the 2s instant floor)",
			currentTick, baseTick+wantSlice)
	}
}

// Any real input ends the journey and then runs normally, where the actor actually stands (R6): the
// player changed their mind mid-wait, so the wait journey is discarded — not suspended, not
// auto-resumed — and the new action resolves as an ordinary beat against wherever Kade actually is.
func TestRunBeat_NewActionEndsTheJourneyAndRunsWhereYouStand(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // isolate the routing decision from a stray eruption.
	wtSetTavernTension(t, ctx, pool, "none")     // unbounded budget: the new action must COMPLETE, not start a second journey.

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	waitAttempt := Attempt{
		Type:     "AttributeChanged",
		Stated:   "I wait quietly at the bar",
		TargetID: dlKadeID,
		Sustain:  &Sustain{Kind: "for", Seconds: 1000},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, waitAttempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	beforeCanon := lgCanonCount(t, ctx, pool, dlWorldID)

	realAttempt := Attempt{Type: "Communicated", Stated: "I greet Mara instead", ListenerID: wtMaraID, Content: "Morning, Mara."}
	outcome, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, []Attempt{realAttempt}, j.CurrentTick+1, nil)
	if err != nil {
		t.Fatalf("RunBeat (new action): %v", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, outcome.Committed) })

	if strings.HasPrefix(outcome.HaltReason, "journey_") {
		t.Fatalf("HaltReason = %q, want an ordinary beat outcome — the discarded journey must not still be steering this beat", outcome.HaltReason)
	}
	if outcome.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed — the new action resolved", outcome.HaltReason)
	}
	if len(outcome.Committed) == 0 {
		t.Fatalf("Committed is empty, want the new action to have actually resolved")
	}
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != beforeCanon+len(outcome.Committed) {
		t.Fatalf("canon count = %d, want beforeCanon(%d)+committed(%d) — the new action must actually commit", got, beforeCanon, len(outcome.Committed))
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM journey WHERE journey_id=$1::uuid`, j.ID).Scan(&status); err != nil {
		t.Fatalf("read back journey: %v", err)
	}
	if status != "ended" {
		t.Fatalf("journey status = %q, want ended (R6 — any other input ends the journey)", status)
	}

	if got := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); got != startLoc {
		t.Fatalf("actor location = %q, want unchanged %q — the new action ran WHERE the player stands, not at some journey waypoint", got, startLoc)
	}
}

// TestRunJourneyLeg_RefusedArrivalIsBarredNotAnError guards the error path that journeys-running-
// their-own-legs created.
//
// A traveller can END a leg somewhere the goal is not lawfully reachable from — the world fires, R2
// mints a waystation, and the way onward from THAT place is shut or was never built. The distance is
// still covered, so the threshold is still met, and the arrival commit is then REFUSED by the
// accessibility floor. (A doorless goal chosen from the start never gets this far: the beat's own
// premise check rejects it up front with "premise_broken", which is why this test displaces the actor
// mid-journey instead — that is the shape the real case has.)
//
// A shut road must read as "journey_barred" with the journey ended and the actor standing where they
// are, never as a server error. While a journey advanced one leg per press you had to click all the
// way to the threshold to provoke this; runJourneyToCompletion reaches every arrival, so it went live
// the same day (2026-08-28).
func TestRunJourneyLeg_RefusedArrivalIsBarredNotAnError(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID) // nothing may fire: this test is about the arrival, not an interruption.

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	// The journey starts lawfully, from the tavern to Dock Street — exactly as the arriving test does.
	attempt := Attempt{Type: "ActorMoved", Stated: "I walk out to Dock Street", ToTargetID: jrDockStreetID}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, attempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	startLoc := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID)
	t.Cleanup(func() { jrSetActorLocation(t, context.Background(), pool, dlWorldID, dlKadeID, startLoc) })

	// Then the road takes them somewhere with no way on: a spur with no portal to the goal. This
	// stands in for the minted-waystation case without needing to force an eruption to mint one.
	spurID := paCreateLocation(t, ctx, pool, dlWorldID, "The Sealed Spur (barred-arrival test)", paHarborQuarterID, 400, 200)
	jrSetActorLocation(t, ctx, pool, dlWorldID, dlKadeID, spurID)

	outcome := &BeatOutcome{}
	if err := orc.runJourneyToCompletion(ctx, j, outcome, nil); err != nil {
		t.Fatalf("runJourneyToCompletion: %v — a shut road must not surface as a server error", err)
	}
	t.Cleanup(func() { wtDeleteEruptionRows(t, context.Background(), pool, dlWorldID, outcome.Committed) })

	if outcome.HaltReason != "journey_barred" {
		t.Fatalf("halt_reason = %q, want journey_barred — a gate-refused arrival is the way being shut", outcome.HaltReason)
	}
	if j.Status != "ended" {
		t.Fatalf("Journey.Status = %q, want ended — a barred journey does not stay open", j.Status)
	}
	// The traveller never got in: they are still on the spur, not at the goal.
	if got := jrActorLocation(t, ctx, pool, dlWorldID, dlKadeID); got != spurID {
		t.Fatalf("actor location = %q, want the spur %q — a barred arrival commits nothing", got, spurID)
	}
}
