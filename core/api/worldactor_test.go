package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Living World / Task 8 (Unit 5) — the World Actor seat. runWorldActor is NOT yet wired into the beat
// (Task 9's world's-turn composer does that), so it is tested directly here against the real seeded
// Drowned Lantern play world (make reset precedes go test in the battery — matches ledger_test.go's and
// orchestrator_worldtime_test.go's wtOrchestrator/dlWorldID/wtTavernID/wtBaseTick pattern).

// waCanonCount counts accepted canon_event rows for worldID (the "did it actually commit" check —
// mirrors ledger_test.go's lgCanonCount).
func waCanonCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND status='accepted'`, worldID).Scan(&n); err != nil {
		t.Fatalf("waCanonCount: %v", err)
	}
	return n
}

// waEventLocation is the B-growth invariant check (task-8-brief Test resolution): the committed canon
// event must carry a location. canon_event has no single "location" column the commit path fills in
// generically — apply_event/apply_ruled_event instead derive the perception boundary from the ACTING
// actor's OWN current position at commit time (apply_event's Communicated co-presence gate reads
// actor_state.attrs.location_id as `here`; apply_ruled_event's own receiver loop does the same). So this
// reads the exact fact the engine's own fan-out reads: the committed event's acting participant's
// current location_id. A non-empty result IS "this event carries a location" — the same fact the commit
// path itself relied on to decide who perceives it (never encoded BY the World Actor — that's the
// invariant: it authors truth-with-a-location, the engine's existing fan-out does the rest).
func waEventLocation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, eventID string) string {
	t.Helper()
	var loc string
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(a.attrs->>'location_id', '')
		FROM event_participant ep
		JOIN actor_state a ON a.world_id = $1 AND a.entity_id = ep.entity_id
		WHERE ep.event_id = $2 AND ep.role_qualifier IN ('instigator', 'speaker')
		LIMIT 1`,
		worldID, eventID).Scan(&loc)
	if err != nil {
		t.Fatalf("waEventLocation: %v", err)
	}
	return loc
}

// TestRunWorldActor_AuthorsWithinSize is the brief's happy path: forcing the fake to author a
// Communicated intrusion (fakeWorldActorDriver.Generate — bridge_fakes.go) for size="medium" must
// commit exactly ONE canon event, that event must carry a location (the B-growth invariant), and — v1
// scope (ambiguity resolution #5) — that location must be the current scene (the tavern): the World
// Actor manifests perceivably at `scene`, never off-scene, in v1.
func TestRunWorldActor_AuthorsWithinSize(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	before := waCanonCount(t, ctx, pool, dlWorldID)
	var out BeatOutcome

	eventID, seqUsed, err := orc.runWorldActor(ctx, dlWorldID, wtTavernID, "medium", baseTick, 0, nil, &out, nil)
	if err != nil {
		t.Fatalf("runWorldActor: %v", err)
	}
	if eventID == "" {
		t.Fatalf("runWorldActor returned an empty event id")
	}
	if seqUsed != 1 {
		t.Fatalf("seqUsed = %d, want 1 (one passthrough commit consumes exactly one (tick,seq) slot)", seqUsed)
	}
	if got := waCanonCount(t, ctx, pool, dlWorldID); got != before+1 {
		t.Fatalf("canon count = %d, want %d (exactly one committed event)", got, before+1)
	}
	if len(out.Committed) != 1 || out.Committed[0] != eventID {
		t.Fatalf("outcome.Committed = %v, want exactly [%s]", out.Committed, eventID)
	}

	loc := waEventLocation(t, ctx, pool, dlWorldID, eventID)
	if loc == "" {
		t.Fatalf("authored event has no location (B-growth invariant violated)")
	}
	if loc != wtTavernID {
		t.Fatalf("authored event location = %s, want the current scene %s (v1: manifests AT the scene)", loc, wtTavernID)
	}
}

// TestRunWorldActor_InvalidAttemptFailsClosed: a fake that authors an out-of-vocabulary/missing-field
// attempt must fail runWorldActor (validateAttemptFields' belt) rather than silently committing garbage
// or falling through to a bypass — the World Actor obeys the same closed-vocabulary field rules as every
// other seat's output (no trusted-seat fast path).
func TestRunWorldActor_InvalidAttemptFailsClosed(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	orc := wtOrchestrator(pool)
	// A Communicated missing its required content — same belt DecodeAndValidateChainV2/
	// DecodeAndValidateNPCDecisions already enforce for every other seat's output.
	orc.WorldActor = &scriptedCognitionDriver{
		name: "bad-world-actor",
		body: `{"actor_id":"` + wtMaraID + `","attempt":{"type":"Communicated","stated":"garble","listener_id":"` + dlKadeID + `"}}`,
	}
	baseTick := wtBaseTick(t, ctx, pool)
	before := waCanonCount(t, ctx, pool, dlWorldID)

	var out BeatOutcome
	if _, _, err := orc.runWorldActor(ctx, dlWorldID, wtTavernID, "small", baseTick, 0, nil, &out, nil); err == nil {
		t.Fatalf("runWorldActor did not fail on an invalid authored attempt")
	}
	if got := waCanonCount(t, ctx, pool, dlWorldID); got != before {
		t.Fatalf("canon count = %d, want unchanged %d — an invalid attempt must not commit anything", got, before)
	}
}

// TestBuildWorldActorPrompt_ContainsDrawnSize is the Test-resolution's prompt-builder alternative to
// capturing the fake's request: the seat must be handed the drawn SIZE constraint verbatim.
func TestBuildWorldActorPrompt_ContainsDrawnSize(t *testing.T) {
	prompt := buildWorldActorPrompt(`{"scene":{"id":"x","name":"The Drowned Lantern"}}`, "medium")
	if !strings.Contains(prompt, "medium") {
		t.Fatalf("prompt does not contain the drawn size %q:\n%s", "medium", prompt)
	}
}

// TestBuildWorldActorPrompt_CarriesRulesAndSlice pins the section layout (mirrors buildResolvePrompt's
// stable-header-then-data shape): the stable header (world_actor.txt's authoring rules — the location
// invariant + the no-appropriateness-filter rule) precedes the raw world slice, embedded verbatim.
func TestBuildWorldActorPrompt_CarriesRulesAndSlice(t *testing.T) {
	slice := `{"ledger":[],"presence":[],"locations":[],"recent":[],"scene":null}`
	prompt := buildWorldActorPrompt(slice, "small")
	if !strings.Contains(prompt, worldActorLocationRuleMarker) {
		t.Fatalf("prompt missing the location-invariant rule marker %q", worldActorLocationRuleMarker)
	}
	if !strings.Contains(prompt, worldActorNoAppropriatenessMarker) {
		t.Fatalf("prompt missing the no-appropriateness-filter marker %q", worldActorNoAppropriatenessMarker)
	}
	if !strings.Contains(prompt, slice) {
		t.Fatalf("prompt does not carry the raw world slice verbatim")
	}
}

// misScopedWorldActorDriver authors an intrusion that lands somewhere OTHER than the scene it was
// handed: an ActorMoved whose destination is `destID`. Point destID at any location that is not the
// scene under test and the seat's v1 scope check must refuse it.
//
// The tests below need that on purpose. They used to get it for free, because the generic fake had
// two tavern ids hardcoded and simply did not know where it was; once that fake learned to read the
// scene from the slice, the only way to still exercise the ENGINE's runtime check is to force the bad
// input deliberately. A test for a guard should supply the thing the guard guards against, never rely
// on some other component being naive enough to supply it by accident.
//
// The move shape is chosen over a speech shape because it is refused on the DESTINATION, which the
// test controls outright — a speech shape would be refused on the speaker's location, which depends
// on where the seed happens to have put people.
type misScopedWorldActorDriver struct{ actorID, destID string }

func (d *misScopedWorldActorDriver) Name() string { return "mis-scoped-world-actor" }
func (d *misScopedWorldActorDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *misScopedWorldActorDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("mis-scoped-world-actor: used without a schema")
	}
	return `{"actor_id":"` + d.actorID + `","attempt":{"type":"ActorMoved",` +
		`"stated":"something stirs somewhere the player is not","to_target_id":"` + d.destID + `"}}`, nil
}

// Deferral B: v1 scope is that the intrusion manifests AT the scene the composer passed. An intrusion
// authored by someone standing somewhere else must be REFUSED at runtime, not merely discouraged by
// the prompt — and refused means "the world does not erupt", never "the beat fails" (SPEC-030's
// deterministic livelock: see errIntrusionRejected).
func TestRunWorldActor_RefusesToActOutsideTheScene(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	const dockStreetID = "210c0000-0000-0000-0000-0000000000d2" // the scene under test
	const tavernID = "210c0000-0000-0000-0000-0000000000d1"     // where the authored move lands instead
	const maraID = "2ac70000-0000-0000-0000-0000000000a2"

	orc := wtOrchestrator(pool)
	orc.WorldActor = &misScopedWorldActorDriver{actorID: maraID, destID: tavernID}
	tick := wtBaseTick(t, ctx, pool)

	var out BeatOutcome
	eventID, seqUsed, err := orc.runWorldActor(ctx, dlWorldID, dockStreetID, "small", tick, 0, nil, &out, nil)
	if err == nil {
		t.Fatalf("authored intrusion from outside the scene was accepted: eventID=%q seqUsed=%d", eventID, seqUsed)
	}
	if !errors.Is(err, errIntrusionRejected) {
		t.Fatalf("err = %v, want it to wrap errIntrusionRejected — a refused proposal is the gate saying no, not a broken machine, and the caller must be able to tell the difference", err)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("outcome.Committed = %v, want empty — a refused intrusion must commit nothing", out.Committed)
	}
}
