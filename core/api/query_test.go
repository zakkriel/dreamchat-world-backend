package main

// Grounded Reasoning / Unit 2 — the QUERY path (route read-only + the narrator answers).
//
// A QUERY is a beat_chain element that ASKS about the world (Task 4). It is NOT an action: asking is
// not acting (RULINGS-2026-07-23 §3), so it writes NO canon, never advances the clock, and NEVER
// reaches the referee. The orchestrator routes a QUERY to a read-only PERCEIVED fact-sheet build
// (fn_fact_sheet, p_truth_side=false — the narrator is a character-mind seat) and hands the answer to
// the narrator, who phrases it in-world. A mixed [action, QUERY] beat still commits its action AND
// answers the question in one narration.
//
// These tests drive the WHOLE path through the REAL HTTP beatHandler (the station_e/f exit-gate
// pattern): a scripted decompose seat emits the chain, a CAPTURING narrate driver records the prompt
// the narrator is handed (so we can prove the QUESTIONS block + the perceived fact sheet reach it), and
// a CAPTURING resolve driver proves the referee is never consulted. Each test mints a fresh, random
// world so the counts see ONLY its rows (the DB is not reset between `go test` runs).

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// queryIDs holds the freshly-minted ids for one query-world invocation.
type queryIDs struct{ World, K, Tavern, Bar, Note, Crate, Stone string }

// setupQueryWorld mints a fresh world reproducing the fact-sheet geometry (mirrors 119/station-F):
// player K at the tavern origin {0,0} with an 80 kg max_load; the bar 8 m east ({8,0} ⇒ CEIL(8/1.4)=6 s)
// — an in-scene object that is BOTH a move target and the "how long to the bar?" query target; a sealed
// note carried on K's person (contained_by K); and a CLOSED ballast crate holding a stone (its contents
// are WITHHELD on the perceived side — the wall). A brand-new random world every invocation → hermetic
// and re-runnable with no DB reset.
func setupQueryWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool) queryIDs {
	t.Helper()
	var id queryIDs
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.K, &id.Tavern, &id.Bar, &id.Note, &id.Crate, &id.Stone); err != nil {
		t.Fatalf("mint query ids: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($2,$1,'actor',   'Player'),
		 ($3,$1,'location','The Drowned Lantern'),
		 ($4,$1,'artifact','the bar'),
		 ($5,$1,'artifact','sealed note'),
		 ($6,$1,'artifact','ballast crate'),
		 ($7,$1,'artifact','ballast stone')`,
		id.World, id.K, id.Tavern, id.Bar, id.Note, id.Crate, id.Stone); err != nil {
		t.Fatalf("seed query registry: %v", err)
	}

	// walk 1.4 m/s (+ encumbered) so fn_move_duration_actor resolves K's speed.
	if _, err := pool.Exec(ctx, `SELECT seed_world_defaults($1)`, id.World); err != nil {
		t.Fatalf("seed query defaults: %v", err)
	}

	// tavern: a root scene with a coordinate frame.
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('coordinates',jsonb_build_object('x',0,'y',0),'extent',jsonb_build_object('w',2000,'h',2000)))`,
		id.World, id.Tavern); err != nil {
		t.Fatalf("seed query location: %v", err)
	}

	// player K: in the tavern, at the origin, 80 kg capacity.
	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('location_id',$3::text,'coordinates',jsonb_build_object('x',0,'y',0),'max_load',80))`,
		id.World, id.K, id.Tavern); err != nil {
		t.Fatalf("seed query player: %v", err)
	}

	// bar: a positioned in-scene object 8 m east (size 2). note: carried by K (contained_by). crate: a
	// CLOSED container (max_room 4, open=false) holding the stone (perceived-side contents WITHHELD).
	if _, err := pool.Exec(ctx, `
		INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
		 ($3,$1, jsonb_build_object('location_id',$2::text,'coordinates',jsonb_build_object('x',8,'y',0),'size',2)),
		 ($4,$1, jsonb_build_object('contained_by',$5::text)),
		 ($6,$1, jsonb_build_object('location_id',$2::text,'coordinates',jsonb_build_object('x',2,'y',0),'max_room',4,'open',false)),
		 ($7,$1, jsonb_build_object('contained_by',$6::text))`,
		id.World, id.Tavern, id.Bar, id.Note, id.K, id.Crate, id.Stone); err != nil {
		t.Fatalf("seed query artifacts: %v", err)
	}
	return id
}

// capturingNarrateDriver records every prompt the narrate seat is handed and returns ONE valid
// narration segment, so DecodeAndValidateNarration succeeds on the first STRUCTURED attempt — the
// captured prompt is buildNarratePrompt's structured output, the exact scene (incl. the QUESTIONS
// block) the narrator reasons from. It reports CapStructuredOutput so it binds like the live seat.
type capturingNarrateDriver struct {
	name    string
	calls   int
	prompts []string
}

func (d *capturingNarrateDriver) Name() string { return d.name }
func (d *capturingNarrateDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *capturingNarrateDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	d.calls++
	d.prompts = append(d.prompts, req.Prompt)
	return `[{"speaker_id":null,"kind":"narration","text":"You take in the room."}]`, nil
}

// queryHandler builds the real HTTP handler with a scripted decompose table (keyed by the player text)
// plus a CAPTURING narrate driver and a CAPTURING resolve driver, so a test can read back the narrate
// prompt and prove the referee was never consulted. Cognition/world-actor stay quiet fakes.
func queryHandler(t *testing.T, pool *pgxpool.Pool, decomposeText, chainJSON string) (http.Handler, *capturingNarrateDriver, *capturingResolveDriver) {
	t.Helper()
	dec := NewFakeStructuredDriver("fake-structured:query", map[string]string{decomposeText: chainJSON})
	narr := &capturingNarrateDriver{name: "capture-narrate"}
	resolve := &capturingResolveDriver{name: "capture-resolve", ruling: `SHOULD NOT BE CALLED`}
	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:         dec,
		SeatNarrate.Name:           narr,
		SeatResolve.Name:           resolve,
		SeatCognitionBatch.Name:    NewFakeCognitionDriver(),
		SeatCognitionIsolated.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:        NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return NewBeatHandler(pool, true, bridge), narr, resolve // debug=true so ?viewer= is honored
}

// TestQueryBeat_PureQuery_NoCanon_NoReferee_AnswersToNarrate is the brief's (a)+(b): a pure QUERY beat
// commits nothing (canon UNCHANGED), TicksAdvanced==0, the referee gets ZERO Generate calls, and the
// narrate prompt carries the QUESTIONS block with the bar's PERCEIVED fact sheet (its computed distance).
func TestQueryBeat_PureQuery_NoCanon_NoReferee_AnswersToNarrate(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const q = "how long to the bar?"
	chain := `[{"type":"QUERY","stated":"` + q + `","query_target_ids":["` + id.Bar + `"]}]`
	h, narr, resolve := queryHandler(t, pool, q, chain)

	before := countCanon(t, ctx, pool, id.World)
	code, body := postBeat(h, id.World, id.K, q)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var r stationFBeatResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}

	// (a1) a query writes NO canon and does not advance the clock.
	if len(r.Result.Committed) != 0 {
		t.Fatalf("committed = %v, want none (asking is not acting — a query writes no canon)", r.Result.Committed)
	}
	if r.Result.HaltReason != "completed" {
		t.Fatalf("halt = %q, want completed", r.Result.HaltReason)
	}
	if r.Result.TicksAdvanced != 0 {
		t.Fatalf("ticks_advanced = %d, want 0 (a query never advances the clock)", r.Result.TicksAdvanced)
	}
	if after := countCanon(t, ctx, pool, id.World); after != before {
		t.Fatalf("canon grew by %d, want 0 (a query writes NO canon)", after-before)
	}

	// (b) the QUERY never reached the resolve seat — the referee got zero Generate calls.
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (a QUERY never reaches the referee)", resolve.calls)
	}

	// (a2) the narrate prompt carries the QUESTIONS block with the question + the perceived fact sheet.
	if len(narr.prompts) == 0 {
		t.Fatalf("the narrator was never consulted")
	}
	p := narr.prompts[0]
	if !strings.Contains(p, narrateQueryBlockMarker) {
		t.Fatalf("narrate prompt missing the QUESTIONS block:\n%s", p)
	}
	if !strings.Contains(p, q) {
		t.Fatalf("narrate prompt missing the question text %q:\n%s", q, p)
	}

	// The PERCEIVED fact sheet the narrator must answer FROM, embedded VERBATIM (mirrors factSheetJSON:
	// jsonb scanned to bytes). Its distance (8 m) is the answer to "how long to the bar?".
	var wantSheetBytes []byte
	if err := pool.QueryRow(ctx,
		`SELECT fn_fact_sheet($1::uuid,$2::uuid,ARRAY[$3]::uuid[],false)`,
		id.World, id.K, id.Bar).Scan(&wantSheetBytes); err != nil {
		t.Fatalf("compute expected perceived fact sheet: %v", err)
	}
	wantSheet := string(wantSheetBytes)
	if !strings.Contains(p, wantSheet) {
		t.Fatalf("narrate prompt missing the query's perceived fact sheet\nwant:\n%s\n\nprompt:\n%s", wantSheet, p)
	}
	// Fixture sanity: the geometry produced a REAL distance/duration (8 m / 6 s), so the assertion is
	// not vacuous against a degenerate all-null sheet.
	var distStr, durStr string
	if err := pool.QueryRow(ctx, `
		SELECT (fs->'targets'->0->>'distance_m'), (fs->'targets'->0->>'move_duration_s')
		FROM (SELECT fn_fact_sheet($1::uuid,$2::uuid,ARRAY[$3]::uuid[],false) AS fs) q`,
		id.World, id.K, id.Bar).Scan(&distStr, &durStr); err != nil {
		t.Fatalf("read bar distance/duration: %v", err)
	}
	// distance_m is jsonb numeric (renders "8.0000000000000000"); duration is bigint ("6").
	if !strings.HasPrefix(distStr, "8") || durStr != "6" {
		t.Fatalf("fixture sanity: bar distance/duration = %q/%q, want ~8/6", distStr, durStr)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestQueryBeat_Mixed_MoveCommits_AndQueryAnswers is the brief's (c): a chain [ActorMoved(bar),
// QUERY(note)] commits the move (canon +1, the clock advances the move's real duration) AND rides the
// query answer to the narrator — one beat, one narration covers both. The referee is still never
// consulted (the move is a passthrough; the query is read-only).
func TestQueryBeat_Mixed_MoveCommits_AndQueryAnswers(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	const q = "how long to slip her the note?"
	text := move + " — " + q
	chain := `[` +
		`{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"},` +
		`{"type":"QUERY","stated":"` + q + `","query_target_ids":["` + id.Note + `"]}` +
		`]`
	h, narr, resolve := queryHandler(t, pool, text, chain)

	before := countCanon(t, ctx, pool, id.World)
	code, body := postBeat(h, id.World, id.K, text)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var r stationFBeatResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}

	// The move committed exactly once and advanced the clock its real duration; the query added neither.
	if r.Result.HaltReason != "completed" {
		t.Fatalf("halt = %q, want completed", r.Result.HaltReason)
	}
	if len(r.Result.Committed) != 1 {
		t.Fatalf("committed = %d, want exactly 1 (the move; the query commits nothing)", len(r.Result.Committed))
	}
	if after := countCanon(t, ctx, pool, id.World); after != before+1 {
		t.Fatalf("canon grew by %d, want 1 (only the move; the query writes no canon)", after-before)
	}
	if r.Result.TicksAdvanced != 6 {
		t.Fatalf("ticks_advanced = %d, want 6 (CEIL(8/1.4); the query added no time)", r.Result.TicksAdvanced)
	}
	// The move is a passthrough and the query read-only, so the referee never fired.
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (passthrough move + read-only query never reach the referee)", resolve.calls)
	}

	// The narrate prompt carries the query answer ALONGSIDE the committed move (one narration, both).
	if len(narr.prompts) == 0 {
		t.Fatalf("the narrator was never consulted")
	}
	p := narr.prompts[0]
	if !strings.Contains(p, narrateQueryBlockMarker) {
		t.Fatalf("narrate prompt missing the QUESTIONS block in a mixed beat:\n%s", p)
	}
	if !strings.Contains(p, q) {
		t.Fatalf("narrate prompt missing the note question %q — the query answer did not ride to narrate alongside the move:\n%s", q, p)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestQueryBeat_ClosedContainer_ContentsWithheld is the brief's (d): querying a CLOSED container yields
// a PERCEIVED fact sheet whose contents are WITHHELD (fn_fact_sheet's perception gate) — the hidden
// item's id is ABSENT from what the narrator is handed. Proven non-vacuous: the truth-side sheet WOULD
// name the item, so it is the wall that removed it, not a degenerate fixture.
func TestQueryBeat_ClosedContainer_ContentsWithheld(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const q = "what's in the crate?"
	chain := `[{"type":"QUERY","stated":"` + q + `","query_target_ids":["` + id.Crate + `"]}]`
	h, narr, resolve := queryHandler(t, pool, q, chain)

	code, body := postBeat(h, id.World, id.K, q)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (a container query never reaches the referee)", resolve.calls)
	}
	if len(narr.prompts) == 0 {
		t.Fatalf("the narrator was never consulted")
	}
	p := narr.prompts[0]
	if !strings.Contains(p, narrateQueryBlockMarker) {
		t.Fatalf("narrate prompt missing the QUESTIONS block:\n%s", p)
	}

	// Non-vacuous: on the TRUTH side, the crate's contents WOULD name the stone.
	var truthSheet string
	if err := pool.QueryRow(ctx,
		`SELECT fn_fact_sheet($1::uuid,$2::uuid,ARRAY[$3]::uuid[],true)::text`,
		id.World, id.K, id.Crate).Scan(&truthSheet); err != nil {
		t.Fatalf("compute truth-side crate sheet: %v", err)
	}
	if !strings.Contains(truthSheet, id.Stone) {
		t.Fatalf("fixture sanity: truth-side crate should list the stone %s\nsheet:\n%s", id.Stone, truthSheet)
	}

	// THE WALL: the narrator (perceived side) got a sheet with the closed crate's contents WITHHELD —
	// the stone id never reaches the prompt, so the narrator cannot tell what's inside from here.
	if strings.Contains(p, id.Stone) {
		t.Fatalf("closed crate's hidden contents (stone %s) leaked into the narrator's fact sheet:\n%s", id.Stone, p)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestNarrateRule_AnswerQuestionsShips proves the answer-the-listed-questions rule reaches the narrator:
// narrate.txt carries the marker, and it sits BEFORE the structured-segment contract so the plain-prose
// fallback (narrateBaseRules, which slices the contract off) still ships it. No DB needed.
func TestNarrateRule_AnswerQuestionsShips(t *testing.T) {
	if !strings.Contains(narrateSystemHeader, narrateQueryRuleMarker) {
		t.Fatalf("narrate.txt missing the answer-questions rule marker %q — the rule never reaches the narrator", narrateQueryRuleMarker)
	}
	if !strings.Contains(narrateBaseRules(), narrateQueryRuleMarker) {
		t.Fatalf("the answer-questions rule is sliced off the plain-prose fallback (it must sit before the segment contract)")
	}
}

// TestQueryBeat_ReactionLeadingQuery_NeverReachesReferee guards the invariant on the REACTION path: a
// QUERY as a reaction's leading element must be routed read-only, NEVER folded into the combined ruling.
// A held act is pending; the reaction is a pure question. The held act still resolves (one combined
// ruling), the query is answered for the narrator, but the question NEVER enters the referee's prompt.
func TestQueryBeat_ReactionLeadingQuery_NeverReachesReferee(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupReactionWorld(t, ctx, pool)

	const windUp = "Jonas moves to cut in"
	seedHeld(t, ctx, pool, id.World, id.J, Attempt{Type: "ActorMoved", Stated: windUp, ToTargetID: id.L2}, reactionBaseTick(t, ctx, pool, id.World))
	held, err := pendingHeldOutcomes(ctx, pool, id.World)
	if err != nil {
		t.Fatalf("pendingHeldOutcomes: %v", err)
	}

	reactTick := reactionBaseTick(t, ctx, pool, id.World)
	ruling := validRulingJSON(id.J, id.J, "Jonas's cut-in completes uncontested.", "Jonas cuts in.")
	resolve := &capturingResolveDriver{name: "capture-resolve", ruling: ruling}
	orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: NewFakeCognitionDriver(), CognitionIsolated: NewFakeCognitionDriver(), WorldActor: NewFakeWorldActorDriver()}

	const q = "how far to the bar?"
	reactionChain := []Attempt{{Type: "QUERY", Stated: q, QueryTargetIDs: []string{id.Note}}}
	out, err := orc.RunReactionBeat(ctx, id.World, id.P, reactionChain, held, reactTick, q, nil)
	if err != nil {
		t.Fatalf("RunReactionBeat: %v", err)
	}

	// The reaction proceeds — the held act resolves in ONE combined ruling (the query did not block it).
	if out.HaltReason != "completed" {
		t.Fatalf("halt = %q, want completed", out.HaltReason)
	}
	if resolve.calls != 1 {
		t.Fatalf("resolve calls = %d, want 1 (the HELD act resolves; the query is not part of the ruling)", resolve.calls)
	}
	// THE INVARIANT: the QUERY was NEVER adjudicated. No typed attempt was synthesized for the player,
	// so the referee's only ACTOR ATTEMPTS line is Jonas's held act. (The raw words may still ride in as
	// the player's stated, not-an-act answer per RULINGS-2026-07-24 §7 — that sanctioned context channel
	// is orthogonal to routing; what matters is the QUERY never became an adjudicated attempt.)
	if _, ok := actorAttemptSegment(resolve.prompts[0], id.P); ok {
		t.Fatalf("a QUERY was adjudicated — an ACTOR %s ATTEMPTS line was synthesized for the player:\n%s", id.P, resolve.prompts[0])
	}
	jSeg, ok := actorAttemptSegment(resolve.prompts[0], id.J)
	if !ok || !strings.Contains(jSeg, windUp) {
		t.Fatalf("the held act is missing from the combined ruling (it must still resolve):\n%s", resolve.prompts[0])
	}
	// The query WAS answered read-only for the narrator.
	if len(out.QueryAnswers) != 1 || out.QueryAnswers[0].Stated != q {
		t.Fatalf("QueryAnswers = %+v, want exactly the note question answered read-only", out.QueryAnswers)
	}
}
