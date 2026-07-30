package main

// Station F exit gate — Task 10, Step 1.
//
// TestStationF_FakeE2E drives Station F's real deliverables — the UNIFIED move (§2/§3), the §5.3
// portal accessibility gate, the §6 tension budget, and §4 encumbrance — through the REAL HTTP
// beatHandler (NewBeatHandler + NewBridgeWithDrivers, all seats scripted), deterministic and
// zero-network. It is the founder's walk of the Drowned Lantern, proven in CI:
//
//   move A — "approach the bar":   an IN-SCENE ActorMoved (to_target_id = the bar, an artifact in the
//                                  tavern) commits with a REAL few-second duration, consuming the
//                                  scene's `tense` beat budget and REPOSITIONING Kade to the bar's
//                                  coordinate — WITHOUT changing his location_id (same scene, no portal).
//   move B — "go down to the cellar": a CROSS-LOCATION ActorMoved to the cellar is BLOCKED by the
//                                  locked cellar hatch — the move writes NOTHING.
//   move C — "go out the front door": a CROSS-LOCATION ActorMoved to Dock Street COMMITS — the front
//                                  door is an open, unlocked Portal, so location_id crosses to the street.
//   grab   — "grab the ballast crate": an ObjectRelocated (contained_by → Kade) makes a 100 kg load land
//                                  on an 80 kg carrier → Kade goes `encumbered`, carried_weight 100.
//   pin    — the encumbered Kade tries to walk out: speed 0 → the move can't fit ANY budget → turn_budget,
//                                  nothing written. The heavy grab PINS him (§2 -100% floor, §4 eager rule).
//
// Self-contained fixture worlds (fresh random uuids per run) reproduce the play seed's geometry —
// coordinates, portals, walk 1.4 m/s — so the test is hermetic and re-runnable with NO `make reset`
// dependency (the deterministic-CI choice the task brief recommends; the play world 2222… needs seeding
// and carries fixed ticks that a second run would collide with).
//
// NOTE (Task 11, RULINGS-2026-07-30 §1): the bar IS now a bindable candidate for the live decompose seat.
// h.payload's candidate whitelist widened from present actors + the current location to ALSO include the
// artifacts the actor PERCEIVES — co-located artifacts (the bar, the crate) and carried items (the note)
// — so a live founder typing "approach the bar" / "grab the crate" / "give the note" has a real id to bind
// (proven by TestPayload_PerceivedCandidates_* in beathandler_test.go). This E2E still scripts the chain
// (deterministic, zero-network), exercising the move MACHINERY end to end; the live-binding gap it once
// flagged is closed.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── fixture ids ────────────────────────────────────────────────────────────────────────────────────
type stationFMoveIDs struct{ World, Kade, Quarter, Tavern, Cellar, Street, Bar, FrontDoor, Hatch string }
type stationFCarryIDs struct{ World, Kade, Quarter, Room, Street, Door, Crate, Stone string }

// setupStationFMoveWorld mints a fresh world reproducing the Drowned Lantern's move geometry (§3):
// a Harbor Quarter parent over the tavern {200,200}, cellar {205,205} and Dock Street {220,200}; Kade
// just inside the tavern door at {6,1} with an 80 kg max_load; the bar at {6,9} (8 m ⇒ CEIL(8/1.4)=6 s);
// an OPEN front door (tavern↔street) and a LOCKED cellar hatch (tavern↔cellar). Tavern tension = `tense`
// (a 30 s beat budget). Dock Street sits 20 m from the tavern in the quarter frame ⇒ CEIL(20/1.4)=15 s,
// which FITS the tense budget (the play seed's 80 m / 58 s door would itself turn_budget-halt in a tense
// scene — a real observation noted in the runbook; here the front door isolates the PORTAL, not the clock).
func setupStationFMoveWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool) stationFMoveIDs {
	t.Helper()
	var id stationFMoveIDs
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.Kade, &id.Quarter, &id.Tavern, &id.Cellar, &id.Street, &id.Bar, &id.FrontDoor, &id.Hatch); err != nil {
		t.Fatalf("mint move ids: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($2,$1,'actor',   'Kade'),
		 ($3,$1,'location','Harbor Quarter of Vael'),
		 ($4,$1,'location','The Drowned Lantern'),
		 ($5,$1,'location','Cellar'),
		 ($6,$1,'location','Dock Street'),
		 ($7,$1,'artifact','the bar'),
		 ($8,$1,'artifact','Front Door'),
		 ($9,$1,'artifact','Cellar Hatch')`,
		id.World, id.Kade, id.Quarter, id.Tavern, id.Cellar, id.Street, id.Bar, id.FrontDoor, id.Hatch); err != nil {
		t.Fatalf("seed move registry: %v", err)
	}

	// walk 1.4 + encumbered -100 (the only two predefined rows, §2).
	if _, err := pool.Exec(ctx, `SELECT seed_world_defaults($1)`, id.World); err != nil {
		t.Fatalf("seed world defaults: %v", err)
	}

	// locations: quarter (root) + three rooms, each a child with a coordinate in the quarter frame.
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('coordinates',jsonb_build_object('x',0,'y',0),'extent',jsonb_build_object('w',2000,'h',2000))),
		 ($3,$1, jsonb_build_object('coordinates',jsonb_build_object('x',200,'y',200),'parent_location_id',$2::text,'tension','tense')),
		 ($4,$1, jsonb_build_object('coordinates',jsonb_build_object('x',205,'y',205),'parent_location_id',$2::text)),
		 ($5,$1, jsonb_build_object('coordinates',jsonb_build_object('x',220,'y',200),'parent_location_id',$2::text))`,
		id.World, id.Quarter, id.Tavern, id.Cellar, id.Street); err != nil {
		t.Fatalf("seed move locations: %v", err)
	}

	// Kade: in the tavern, just inside the door, 80 kg capacity.
	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('location_id',$3::text,'coordinates',jsonb_build_object('x',6,'y',1),'max_load',80))`,
		id.World, id.Kade, id.Tavern); err != nil {
		t.Fatalf("seed Kade: %v", err)
	}

	// artifacts: the bar (a positioned fixture) + two portals (open front door, locked cellar hatch).
	if _, err := pool.Exec(ctx, `
		INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('location_id',$5::text,'coordinates',jsonb_build_object('x',6,'y',9),'size',2)),
		 ($3,$1, jsonb_build_object('open',true, 'locked',false,'connects',jsonb_build_array($5::text,$6::text))),
		 ($4,$1, jsonb_build_object('open',false,'locked',true, 'connects',jsonb_build_array($5::text,$7::text)))`,
		id.World, id.Bar, id.FrontDoor, id.Hatch, id.Tavern, id.Street, id.Cellar); err != nil {
		t.Fatalf("seed move artifacts: %v", err)
	}
	return id
}

// setupStationFCarryWorld mints a fresh world for the §4 encumbrance + pin: Kade (80 kg max_load) in a
// room with a ballast crate (empty 8 kg) holding a ballast stone (92 kg) ⇒ effective_weight 100 kg, so
// grabbing it exceeds his capacity. An adjacent street with an OPEN door lets the pin be shown as a
// move that cannot fit the budget once he is encumbered (speed 0).
func setupStationFCarryWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool) stationFCarryIDs {
	t.Helper()
	var id stationFCarryIDs
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.Kade, &id.Quarter, &id.Room, &id.Street, &id.Door, &id.Crate, &id.Stone); err != nil {
		t.Fatalf("mint carry ids: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($2,$1,'actor',   'Kade'),
		 ($3,$1,'location','Harbor Quarter of Vael'),
		 ($4,$1,'location','The Drowned Lantern'),
		 ($5,$1,'location','Dock Street'),
		 ($6,$1,'artifact','Front Door'),
		 ($7,$1,'artifact','Ballast Crate'),
		 ($8,$1,'artifact','Ballast Stone')`,
		id.World, id.Kade, id.Quarter, id.Room, id.Street, id.Door, id.Crate, id.Stone); err != nil {
		t.Fatalf("seed carry registry: %v", err)
	}

	if _, err := pool.Exec(ctx, `SELECT seed_world_defaults($1)`, id.World); err != nil {
		t.Fatalf("seed carry defaults: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('coordinates',jsonb_build_object('x',0,'y',0),'extent',jsonb_build_object('w',2000,'h',2000))),
		 ($3,$1, jsonb_build_object('coordinates',jsonb_build_object('x',200,'y',200),'parent_location_id',$2::text,'tension','tense')),
		 ($4,$1, jsonb_build_object('coordinates',jsonb_build_object('x',210,'y',200),'parent_location_id',$2::text))`,
		id.World, id.Quarter, id.Room, id.Street); err != nil {
		t.Fatalf("seed carry locations: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('location_id',$3::text,'coordinates',jsonb_build_object('x',5,'y',5),'max_load',80))`,
		id.World, id.Kade, id.Room); err != nil {
		t.Fatalf("seed carry Kade: %v", err)
	}

	// crate (a Container: empty_weight + max_room) with a heavy stone inside it; open door room↔street.
	if _, err := pool.Exec(ctx, `
		INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('location_id',$5::text,'coordinates',jsonb_build_object('x',5,'y',6),'empty_weight',8,'max_room',16,'size',4)),
		 ($3,$1, jsonb_build_object('weight',92,'size',2,'contained_by',$2::text)),
		 ($4,$1, jsonb_build_object('open',true,'locked',false,'connects',jsonb_build_array($5::text,$6::text)))`,
		id.World, id.Crate, id.Stone, id.Door, id.Room, id.Street); err != nil {
		t.Fatalf("seed carry artifacts: %v", err)
	}
	return id
}

// stationFBeatResp narrows the shipped beat_result/3 JSON to the fields the exit gate asserts on
// (ticks_advanced is what proves a move cost real beat-time — it is absent from stationEBeatResp).
type stationFBeatResp struct {
	Narration string `json:"narration"`
	Result    struct {
		Committed     []string `json:"committed"`
		HaltReason    string   `json:"halt_reason"`
		TicksAdvanced int64    `json:"ticks_advanced"`
	} `json:"result"`
}

// stationFHandler builds the real HTTP handler with a single-entry scripted decompose table (so the
// substring match is unambiguous per beat) and deterministic fakes in every other seat. The cognition
// seats never fire (no NPC is present in these fixtures), so a beat is pure player-move machinery.
func stationFHandler(t *testing.T, pool *pgxpool.Pool, decomposeText, chainJSON string) http.Handler {
	t.Helper()
	dec := NewFakeStructuredDriver("fake-structured:station-f", map[string]string{decomposeText: chainJSON})
	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:         dec,
		SeatNarrate.Name:           NewFakeTextDriver("fake-text:station-f"),
		SeatResolve.Name:           NewFakeResolveDriver(),
		SeatCognitionBatch.Name:    NewFakeCognitionDriver(),
		SeatCognitionIsolated.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:        NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return NewBeatHandler(pool, true, bridge) // debug=true so ?viewer= is honored
}

// postFBeat POSTs one player text through the real handler as viewer Kade and parses the response.
func postFBeat(t *testing.T, pool *pgxpool.Pool, worldID, kadeID, text, chainJSON string) stationFBeatResp {
	t.Helper()
	h := stationFHandler(t, pool, text, chainJSON)
	code, body := postBeat(h, worldID, kadeID, text) // postBeat is defined in station_e_exit_test.go
	if code != http.StatusOK {
		t.Fatalf("beat %q status = %d, want 200\nbody: %s", text, code, body)
	}
	var r stationFBeatResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("beat %q parse: %v\nbody: %s", text, err, body)
	}
	return r
}

// kadePosition reads Kade's current scene + in-scene coordinate from actor_state.
func kadePosition(t *testing.T, ctx context.Context, pool *pgxpool.Pool, world, kade string) (loc string, x, y float64) {
	t.Helper()
	if err := pool.QueryRow(ctx,
		`SELECT attrs->>'location_id',
		        COALESCE((attrs->'coordinates'->>'x')::float8, 0),
		        COALESCE((attrs->'coordinates'->>'y')::float8, 0)
		 FROM actor_state WHERE world_id=$1 AND entity_id=$2`, world, kade).Scan(&loc, &x, &y); err != nil {
		t.Fatalf("read Kade position: %v", err)
	}
	return loc, x, y
}

func TestStationF_FakeE2E(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// ═══════════════════════════════════════════════════════════════════════════════════════════════
	// PART A — the three moves against the Drowned Lantern's geometry.
	// ═══════════════════════════════════════════════════════════════════════════════════════════════
	m := setupStationFMoveWorld(t, ctx, pool)

	// ── move A: "approach the bar" — an IN-SCENE move to an artifact. Commits; Kade's coordinate
	//    becomes the bar's {6,9}; his location_id stays the tavern; duration = CEIL(8 / 1.4) = 6 s,
	//    consuming 6 of the `tense` scene's 30 s budget. NO portal needed (same scene). ───────────────
	rA := postFBeat(t, pool, m.World, m.Kade, "approach the bar",
		`[{"type":"ActorMoved","stated":"approach the bar","to_target_id":"`+m.Bar+`"}]`)

	if rA.Result.HaltReason != "completed" {
		t.Fatalf("move A halt = %q, want completed", rA.Result.HaltReason)
	}
	if len(rA.Result.Committed) != 1 {
		t.Fatalf("move A committed = %d, want exactly 1 (the in-scene move)", len(rA.Result.Committed))
	}
	if rA.Result.TicksAdvanced != 6 {
		t.Fatalf("move A ticks_advanced = %d, want 6 (CEIL(8 m / 1.4 m/s)); the move cost real beat-time",
			rA.Result.TicksAdvanced)
	}
	// The committed canon event IS an ActorMoved with the stated summary.
	var evType, evSummary string
	if err := pool.QueryRow(ctx, `SELECT event_type, summary FROM canon_event WHERE event_id=$1::uuid`,
		rA.Result.Committed[0]).Scan(&evType, &evSummary); err != nil {
		t.Fatalf("move A canon lookup: %v", err)
	}
	if evType != "ActorMoved" || evSummary != "approach the bar" {
		t.Fatalf("move A canon = (%q,%q), want (ActorMoved, \"approach the bar\")", evType, evSummary)
	}
	// Kade repositioned to the bar's coordinate; location UNCHANGED (same scene, no portal).
	if loc, x, y := kadePosition(t, ctx, pool, m.World, m.Kade); loc != m.Tavern || x != 6 || y != 9 {
		t.Fatalf("move A left Kade at (loc=%s, %g,%g), want (tavern=%s, 6,9): the in-scene move must update the coordinate, not the location",
			loc, x, y, m.Tavern)
	}
	// The scene really was `tense` (a finite 30 s budget), so ticks_advanced=6 is 6-of-30 consumed —
	// not a free teleport. (budgetRemaining is internal; the finite tension + real duration prove it.)
	var tension string
	if err := pool.QueryRow(ctx, `SELECT attrs->>'tension' FROM location_state WHERE world_id=$1 AND entity_id=$2`,
		m.World, m.Tavern).Scan(&tension); err != nil {
		t.Fatalf("read tavern tension: %v", err)
	}
	if tension != "tense" {
		t.Fatalf("tavern tension = %q, want tense (the 30 s budget the 6 s walk drew down)", tension)
	}

	// ── move B: "go down to the cellar" — a CROSS-LOCATION move BLOCKED by the locked cellar hatch.
	//    The move writes NOTHING. NOTE the halt is `premise_broken`, not `gate_reject`: the Go premise
	//    re-check (premiseHolds → fn_portal_permits) mirrors the SAME §5.3 portal gate that apply_event's
	//    SQL twin enforces, and it fires one stage earlier — both block the move and commit nothing. ───
	before := countCanon(t, ctx, pool, m.World)
	rB := postFBeat(t, pool, m.World, m.Kade, "go down to the cellar",
		`[{"type":"ActorMoved","stated":"go down to the cellar","to_target_id":"`+m.Cellar+`"}]`)

	if rB.Result.HaltReason != "premise_broken" {
		t.Fatalf("move B halt = %q, want premise_broken (locked hatch blocks the traversal)", rB.Result.HaltReason)
	}
	if len(rB.Result.Committed) != 0 {
		t.Fatalf("move B committed = %d, want 0 (the locked hatch writes nothing)", len(rB.Result.Committed))
	}
	if after := countCanon(t, ctx, pool, m.World); after != before {
		t.Fatalf("move B added %d canon events, want 0 (nothing written)", after-before)
	}
	// Kade did not budge from the bar.
	if loc, x, y := kadePosition(t, ctx, pool, m.World, m.Kade); loc != m.Tavern || x != 6 || y != 9 {
		t.Fatalf("move B moved Kade to (loc=%s, %g,%g); the blocked cellar move must leave him at the bar", loc, x, y)
	}

	// ── move C: "go out the front door" — a CROSS-LOCATION move PERMITTED by the open, unlocked front
	//    door. Commits; location_id crosses to Dock Street. duration = CEIL(20/1.4) = 15 s (fits the
	//    fresh tense budget). ────────────────────────────────────────────────────────────────────────
	rC := postFBeat(t, pool, m.World, m.Kade, "go out the front door",
		`[{"type":"ActorMoved","stated":"go out the front door","to_target_id":"`+m.Street+`"}]`)

	if rC.Result.HaltReason != "completed" {
		t.Fatalf("move C halt = %q, want completed (open front door permits the traversal)", rC.Result.HaltReason)
	}
	if len(rC.Result.Committed) != 1 {
		t.Fatalf("move C committed = %d, want exactly 1", len(rC.Result.Committed))
	}
	if rC.Result.TicksAdvanced != 15 {
		t.Fatalf("move C ticks_advanced = %d, want 15 (CEIL(20 m / 1.4 m/s))", rC.Result.TicksAdvanced)
	}
	if loc, _, _ := kadePosition(t, ctx, pool, m.World, m.Kade); loc != m.Street {
		t.Fatalf("move C left Kade at %s, want Dock Street %s (the open portal crosses him out)", loc, m.Street)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)

	// ═══════════════════════════════════════════════════════════════════════════════════════════════
	// PART B — §4 encumbrance: the heavy grab, and the pin it causes.
	// ═══════════════════════════════════════════════════════════════════════════════════════════════
	c := setupStationFCarryWorld(t, ctx, pool)

	// ── grab: "grab the ballast crate" — ObjectRelocated, contained_by → Kade. The crate weighs 100 kg
	//    (empty 8 + stone 92); Kade's max_load is 80, so the eager rule sets `encumbered` and writes
	//    carried_weight 100 in the SAME commit. ────────────────────────────────────────────────────────
	rG := postFBeat(t, pool, c.World, c.Kade, "grab the ballast crate",
		`[{"type":"ObjectRelocated","stated":"grab the ballast crate","object_id":"`+c.Crate+`","dest_kind":"actor","dest_id":"`+c.Kade+`"}]`)

	if rG.Result.HaltReason != "completed" {
		t.Fatalf("grab halt = %q, want completed (weight consequences, never blocks §4)", rG.Result.HaltReason)
	}
	if len(rG.Result.Committed) != 1 {
		t.Fatalf("grab committed = %d, want exactly 1", len(rG.Result.Committed))
	}
	// The crate's carry edge now points at Kade.
	var crateHolder string
	if err := pool.QueryRow(ctx, `SELECT attrs->>'contained_by' FROM artifact_state WHERE world_id=$1 AND entity_id=$2`,
		c.World, c.Crate).Scan(&crateHolder); err != nil {
		t.Fatalf("read crate holder: %v", err)
	}
	if crateHolder != c.Kade {
		t.Fatalf("crate contained_by = %q, want Kade %q", crateHolder, c.Kade)
	}
	// Kade is encumbered, carried_weight 100 > max_load 80.
	var encumbered bool
	var carried, maxLoad float64
	if err := pool.QueryRow(ctx,
		`SELECT (attrs->'statuses' ? 'encumbered'),
		        COALESCE((attrs->>'carried_weight')::float8, 0),
		        COALESCE((attrs->>'max_load')::float8, 0)
		 FROM actor_state WHERE world_id=$1 AND entity_id=$2`, c.World, c.Kade).Scan(&encumbered, &carried, &maxLoad); err != nil {
		t.Fatalf("read Kade encumbrance: %v", err)
	}
	if !encumbered {
		t.Fatalf("Kade not encumbered after the heavy grab (statuses lacks 'encumbered')")
	}
	if carried != 100 {
		t.Fatalf("Kade carried_weight = %g, want 100 (empty 8 + stone 92)", carried)
	}
	if !(carried > maxLoad) {
		t.Fatalf("carried_weight %g is not > max_load %g — the eager rule should have found the overload", carried, maxLoad)
	}

	// ── pin: the encumbered Kade tries to haul it to the street. -100% ⇒ speed 0 ⇒ the move can't fit
	//    ANY budget ⇒ turn_budget, nothing written. The heavy grab PINS him (§2 floor × §4 eager rule). ──
	beforePin := countCanon(t, ctx, pool, c.World)
	rP := postFBeat(t, pool, c.World, c.Kade, "haul it to the street",
		`[{"type":"ActorMoved","stated":"haul it to the street","to_target_id":"`+c.Street+`"}]`)

	if rP.Result.HaltReason != "turn_budget" {
		t.Fatalf("pin halt = %q, want turn_budget (encumbered → speed 0 → no move fits any budget)", rP.Result.HaltReason)
	}
	if len(rP.Result.Committed) != 0 {
		t.Fatalf("pin committed = %d, want 0 (pinned in place)", len(rP.Result.Committed))
	}
	if after := countCanon(t, ctx, pool, c.World); after != beforePin {
		t.Fatalf("pin added %d canon events, want 0", after-beforePin)
	}
	if loc, _, _ := kadePosition(t, ctx, pool, c.World, c.Kade); loc != c.Room {
		t.Fatalf("pin moved Kade to %s; the encumbered actor must stay put", loc)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// countCanon returns the number of canon events in a world (the "nothing written" proof for blocked moves).
func countCanon(t *testing.T, ctx context.Context, pool *pgxpool.Pool, world string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM canon_event WHERE world_id=$1`, world).Scan(&n); err != nil {
		t.Fatalf("count canon: %v", err)
	}
	return n
}
