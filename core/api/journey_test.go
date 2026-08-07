package main

import (
	"context"
	"encoding/json"
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
