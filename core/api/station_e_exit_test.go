package main

// Station E exit gate — Task 10, Step 1.
//
// TestStationE_FakeE2E drives the WHOLE Station E loop through the REAL HTTP beatHandler
// (NewBeatHandler + NewBridgeWithDrivers, all six seats bound to scripted drivers), deterministic
// and zero-network. It proves the telegraph beat → reaction beat round trip end to end:
//
//   Beat 1 (a normal beat that the WORLD ends): the player leans on Mara. The mechanical §5 split
//   fires — Mara holds a PRIVATE record subject-linked to the player, so she rides an ISOLATED call
//   (returns "none": guarded, not omniscient); Jonas holds nothing, so he rides the shared BATCH
//   call and TELEGRAPHS his cut-in. The wind-up commits as canon (origin='telegraph'), one pending
//   held_outcome row holds Jonas's full intended act, and the beat ENDS before the player's
//   Communicated resolves — the world seized the moment first, so nothing of the player's is
//   committed (RULINGS-2026-07-24 §1, §3).
//
//   Beat 2 (the reaction): the player's next input meets the pending hold. A SECOND handler — no
//   shared memory with the first — reads the pending hold FRESH from the world (the DB carries the
//   reaction state, not any server session) and routes to the reaction path. Jonas's held act + the
//   reaction's first action collide in ONE combined ruling (§2), actor-attributed to both; the held
//   row flips 'resolved' inside the ruling's tx; the remainder runs afterward as a normal chain
//   (canon order: ruling precedes the freeform remainder). halt_reason='completed'.
//
// Two handlers (one bridge each, same pool + world) model exactly what a fresh server process would
// do on the founder's two POSTs: the ONLY thing bridging the beats is the held_outcome row in the
// world. That is the stronger proof of "the world carries the reaction state" than reusing one
// handler would be.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// stationEIDs holds the freshly-minted ids for one self-contained Station E world.
type stationEIDs struct{ World, K, M, J, L, L2, Secret string }

// setupStationEWorld mints a fresh, random world: player K, NPC Mara (holds ONE private record
// subject-linked to K), NPC Jonas (holds nothing), a bar to cut across to (L2), all co-located at L
// (moves projected into actor_state by the state_mutation trigger). A brand-new world every
// invocation → hermetic and re-runnable with no DB reset (the DB is not reset between `go test`
// runs). Mirrors setupFlowWorld's private-record grounding + telegraph_test's second location.
func setupStationEWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool) stationEIDs {
	t.Helper()
	var id stationEIDs
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.K, &id.M, &id.J, &id.L, &id.L2, &id.Secret); err != nil {
		t.Fatalf("mint ids: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($1,$6,'actor','Player'),
		 ($2,$6,'actor','Mara'),
		 ($3,$6,'actor','Jonas'),
		 ($4,$6,'location','The Drowned Lantern'),
		 ($5,$6,'location','The Bar')`,
		id.K, id.M, id.J, id.L, id.L2, id.World); err != nil {
		t.Fatalf("seed station-e entities: %v", err)
	}

	// Mara's private record about K: a private source event whose truthful summary IS the secret,
	// one perception held ONLY by Mara, subject-linked to K. Not shared by all present → private;
	// subject K is in the action's bound ids → Mara is flagged ISOLATED for the lean-on beat.
	if _, err := pool.Exec(ctx, `
		INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status, accepted_at, visibility_scope, origin)
		VALUES ($1,$2,'observation','the harbormaster secret only Mara saw',90,0,'accepted',now(),'private','fast_path')`,
		id.Secret, id.World); err != nil {
		t.Fatalf("seed secret event: %v", err)
	}
	var mPid string
	if err := pool.QueryRow(ctx, `
		INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
		VALUES ($1,$2,$3,'the ledger names the harbormaster','direct',90,90) RETURNING perception_id`,
		id.World, id.M, id.Secret).Scan(&mPid); err != nil {
		t.Fatalf("seed secret perception: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES ($1,$2,$3)`,
		mPid, id.K, id.World); err != nil {
		t.Fatalf("seed secret subject: %v", err)
	}

	// Co-locate K, M, J at L (each move is the actor's latest state → fn_actors_at returns all three).
	for i, actor := range []string{id.K, id.M, id.J} {
		if _, err := pool.Exec(ctx, `
			WITH ev AS (
			  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
			  VALUES (gen_random_uuid(),$1,'move','station-e-colocate',$2,0,'accepted',now(),'public','fast_path')
			  RETURNING event_id
			),
			ep AS (
			  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
			  SELECT event_id,$3,'actor','instigator' FROM ev
			)
			INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
			SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($4::text),$2,0 FROM ev`,
			id.World, int64(100+i), actor, id.L); err != nil {
			t.Fatalf("colocate %s: %v", actor, err)
		}
	}
	return id
}

// stationEBeatResp is the beatHandler's shipped JSON shape (beat_result/2), narrowed to the fields
// the exit gate asserts on.
type stationEBeatResp struct {
	Narration string `json:"narration"`
	Result    struct {
		Committed  []string `json:"committed"`
		HaltReason string   `json:"halt_reason"`
		Telegraphs []string `json:"telegraphs"`
	} `json:"result"`
}

// postBeat POSTs one player text through the real handler as viewer K and returns status + body.
func postBeat(h http.Handler, worldID, viewerID, text string) (int, string) {
	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+viewerID,
		strings.NewReader(`{"text":`+jsonStr(text)+`}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// stationECombinedRuling is the scripted resolve seat's combined ruling/2 for beat 2: TWO events
// covering BOTH actors — Jonas's telegraphed cut-in resolving (actor J) and the player's press
// (actor K on target J). Both actor ids are folded into the gathered slice by adjudicate, so
// verdictRuling passes on the first try (one Generate, no repair/bounce).
func stationECombinedRuling(heldActor, playerActor, target string) string {
	r := map[string]any{
		"reasoning": "Jonas's telegraphed cut-in meets the player's shove in one combined moment.",
		"therefore": "succeeds",
		"outcome": map[string]any{
			"kind": "resolved",
			"events": []map[string]any{
				{"type": "AttributeChanged", "actor_id": heldActor, "target_id": heldActor,
					"truth": "Jonas's cut-in is checked as the player rounds on him.", "visible": true},
				{"type": "AttributeChanged", "actor_id": playerActor, "target_id": target,
					"truth": "The player shoves Jonas back, breaking his advance.", "appearance": "The player lunges at Jonas.", "visible": true},
			},
		},
	}
	b, _ := json.Marshal(r)
	return string(b)
}

// actorAttemptSegment returns the slice of a buildResolvePrompt prompt running from
// "ACTOR <actorID> ATTEMPTS: " up to (but not including) the NEXT "ACTOR " marker, or the end of the
// prompt if none follows. This is buildResolvePrompt's own per-attempt line format
// (resolveprompt.go:24-31: "ACTOR " + actorID + " ATTEMPTS: " + attempt JSON, one line per attempt).
// Scoping an assertion to this segment — rather than the whole prompt — is what actually pins content
// to its actor: the id and other attempt fields are also echoed independently in the FACTS/
// gather_slice section, so a whole-prompt substring check cannot distinguish "attributed to this
// actor" from "appears somewhere in the prompt". ok is false if the actor has no ATTEMPTS line.
func actorAttemptSegment(prompt, actorID string) (segment string, ok bool) {
	marker := "ACTOR " + actorID + " ATTEMPTS: "
	start := strings.Index(prompt, marker)
	if start == -1 {
		return "", false
	}
	rest := prompt[start+len(marker):]
	if next := strings.Index(rest, "ACTOR "); next != -1 {
		return rest[:next], true
	}
	return rest, true
}

func TestStationE_FakeE2E(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupStationEWorld(t, ctx, pool)

	// The runtime tick floor for this world (co-location moves at 100..102) — the perception-subject
	// backfill at the end covers every runtime commit above it.
	var runtimeBase int64
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)`,
		id.World).Scan(&runtimeBase); err != nil {
		t.Fatalf("runtime base tick: %v", err)
	}

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// Beat 1 — the telegraph beat. Player leans on Mara; Mara (isolated) stays quiet; Jonas
	// (batch) telegraphs his cut-in; the wind-up commits and the beat ends.
	// ─────────────────────────────────────────────────────────────────────────────────────────
	const beat1Text = "I lean on Mara about the harbormaster"
	const windUp = "Jonas pushes off the bar, moving to cut in"

	// decompose maps the player text → a single Communicated K→M attempt (bound ids {M,K}).
	beat1Decompose := NewFakeStructuredDriver("fake-structured:station-e-b1", map[string]string{
		beat1Text: `[{"type":"Communicated","stated":"` + beat1Text + `","listener_id":"` + id.M + `","content":"tell me about the harbormaster"}]`,
	})
	// Jonas (batch): a disruptive TELEGRAPH of his full intended ActorMoved (his real act, held).
	batch1 := &scriptedCognitionDriver{name: "scripted-batch-b1", body: `[{"actor_id":"` + id.J +
		`","decision":{"commit_kind":"telegraph","attempt":{"type":"ActorMoved","stated":"` + windUp +
		`","to_target_id":"` + id.L2 + `"}}}]`}
	// Mara (isolated): her secret rides her own call; she does nothing this moment.
	iso1 := &scriptedCognitionDriver{name: "scripted-isolated-b1", body: `[{"actor_id":"` + id.M + `","decision":"none"}]`}
	// Resolve must be BOUND but is never called on the telegraph path (it commits via apply_ruled_event).
	resolve1 := &countingResolveDriver{inner: NewFakeResolveDriver()}

	bridge1, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:         beat1Decompose,
		SeatNarrate.Name:           NewFakeTextDriver("fake-text:station-e-b1"),
		SeatResolve.Name:           resolve1,
		SeatCognitionBatch.Name:    batch1,
		SeatCognitionIsolated.Name: iso1,
		SeatWorldActor.Name:        NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge 1: %v", err)
	}
	h1 := NewBeatHandler(pool, true, bridge1) // debug=true so ?viewer= is honored

	code, body := postBeat(h1, id.World, id.K, beat1Text)
	if code != http.StatusOK {
		t.Fatalf("beat 1 status = %d, want 200\nbody: %s", code, body)
	}
	var r1 stationEBeatResp
	if err := json.Unmarshal([]byte(body), &r1); err != nil {
		t.Fatalf("beat 1 parse: %v\nbody: %s", err, body)
	}

	// (1a) The world ended the beat on the telegraph, and the wind-up string surfaced.
	if r1.Result.HaltReason != "telegraph" {
		t.Fatalf("beat 1 halt_reason = %q, want telegraph\nbody: %s", r1.Result.HaltReason, body)
	}
	if len(r1.Result.Telegraphs) != 1 || r1.Result.Telegraphs[0] != windUp {
		t.Fatalf("beat 1 telegraphs = %v, want [%q]", r1.Result.Telegraphs, windUp)
	}
	// (1b) The split fired: batch spoke for Jonas, the isolated seat spoke for Mara — exactly one
	// call each. Mara's secret rode her ISOLATED call alone (the wall by construction).
	if batch1.calls != 1 {
		t.Fatalf("beat 1 batch calls = %d, want 1 (Jonas)", batch1.calls)
	}
	if iso1.calls != 1 {
		t.Fatalf("beat 1 isolated calls = %d, want 1 (Mara, secret-flagged)", iso1.calls)
	}
	if resolve1.calls != 0 {
		t.Fatalf("beat 1 resolve calls = %d, want 0 (telegraph commits via apply_ruled_event, not resolve)", resolve1.calls)
	}

	// (1c) Exactly one canon wind-up event, origin='telegraph', summary = the stated wind-up.
	var windUpEvID, windUpSummary, windUpType string
	if err := pool.QueryRow(ctx,
		`SELECT event_id::text, summary, event_type FROM canon_event WHERE world_id=$1 AND origin='telegraph'`,
		id.World).Scan(&windUpEvID, &windUpSummary, &windUpType); err != nil {
		t.Fatalf("beat 1 read telegraph canon event: %v", err)
	}
	if windUpType != "AttributeChanged" {
		t.Fatalf("beat 1 wind-up event_type = %q, want AttributeChanged", windUpType)
	}
	if windUpSummary != windUp {
		t.Fatalf("beat 1 wind-up summary = %q, want %q (canon summary = truth)", windUpSummary, windUp)
	}
	var telegraphCanon int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin='telegraph'`, id.World).Scan(&telegraphCanon); err != nil {
		t.Fatalf("beat 1 count telegraph canon: %v", err)
	}
	if telegraphCanon != 1 {
		t.Fatalf("beat 1 telegraph canon events = %d, want exactly 1", telegraphCanon)
	}

	// (1d) Exactly ONE pending held_outcome row, holding Jonas's FULL intended ActorMoved.
	var heldCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM held_outcome WHERE world_id=$1 AND status='pending'`, id.World).Scan(&heldCount); err != nil {
		t.Fatalf("beat 1 count held_outcome: %v", err)
	}
	if heldCount != 1 {
		t.Fatalf("beat 1 pending held_outcome rows = %d, want exactly 1", heldCount)
	}
	var heldActor, heldAttemptJSON string
	if err := pool.QueryRow(ctx,
		`SELECT actor_id::text, attempt::text FROM held_outcome WHERE world_id=$1 AND status='pending'`,
		id.World).Scan(&heldActor, &heldAttemptJSON); err != nil {
		t.Fatalf("beat 1 read held_outcome: %v", err)
	}
	if heldActor != id.J {
		t.Fatalf("beat 1 held actor = %q, want Jonas %q", heldActor, id.J)
	}
	var heldAttempt Attempt
	if err := json.Unmarshal([]byte(heldAttemptJSON), &heldAttempt); err != nil {
		t.Fatalf("beat 1 held attempt not valid Attempt JSON: %v", err)
	}
	if heldAttempt.Type != "ActorMoved" || heldAttempt.Stated != windUp || heldAttempt.ToTargetID != id.L2 {
		t.Fatalf("beat 1 held attempt = %+v, want Jonas's full ActorMoved to the bar", heldAttempt)
	}

	// (1e) The player's Communicated was DISCARDED — the world acted first, the beat ended. Player
	// attempts commit via apply_event with origin='freeform', so zero 'freeform' canon proves it.
	var freeform1 int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin='freeform'`, id.World).Scan(&freeform1); err != nil {
		t.Fatalf("beat 1 count freeform canon: %v", err)
	}
	if freeform1 != 0 {
		t.Fatalf("beat 1 freeform canon events = %d, want 0 (player's Communicated discarded on telegraph)", freeform1)
	}

	// (1f) Narration (from the player's post-beat perceptions) is non-empty.
	if r1.Narration == "" || strings.Contains(body, `"narration":""`) {
		t.Fatalf("beat 1 narration missing or empty\nbody: %s", body)
	}

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// Beat 2 — the reaction beat. A SECOND handler (no shared memory) reads the pending hold fresh
	// from the world, collides it with the player's press in one combined ruling, then runs the
	// remainder as a normal chain.
	// ─────────────────────────────────────────────────────────────────────────────────────────
	const beat2Text = "I press on and shove Jonas back, then whisper to Mara"
	const pressText = "I shove Jonas back"
	const whisperText = "I whisper to Mara"

	// decompose maps the reaction text → a two-attempt chain: the press (AttributeChanged on Jonas,
	// the collision's first action) then a Communicated (the remainder, a normal post-collision step).
	beat2Decompose := NewFakeStructuredDriver("fake-structured:station-e-b2", map[string]string{
		beat2Text: `[` +
			`{"type":"AttributeChanged","stated":"` + pressText + `","target_id":"` + id.J + `"},` +
			`{"type":"Communicated","stated":"` + whisperText + `","listener_id":"` + id.M + `","content":"later"}` +
			`]`,
	})
	// Combined ruling covering BOTH actors; the capturing driver lets us assert the ONE prompt.
	resolve2 := &capturingResolveDriver{name: "capture-resolve-b2", ruling: stationECombinedRuling(id.J, id.K, id.J)}

	bridge2, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:         beat2Decompose,
		SeatNarrate.Name:           NewFakeTextDriver("fake-text:station-e-b2"),
		SeatResolve.Name:           resolve2,
		SeatCognitionBatch.Name:    NewFakeCognitionDriver(), // quiet: the collision IS the world's move (depth-1)
		SeatCognitionIsolated.Name: NewFakeCognitionDriver(), // quiet
		SeatWorldActor.Name:        NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge 2: %v", err)
	}
	h2 := NewBeatHandler(pool, true, bridge2)

	code, body = postBeat(h2, id.World, id.K, beat2Text)
	if code != http.StatusOK {
		t.Fatalf("beat 2 status = %d, want 200\nbody: %s", code, body)
	}
	var r2 stationEBeatResp
	if err := json.Unmarshal([]byte(body), &r2); err != nil {
		t.Fatalf("beat 2 parse: %v\nbody: %s", err, body)
	}

	// (2a) The reaction beat completed.
	if r2.Result.HaltReason != "completed" {
		t.Fatalf("beat 2 halt_reason = %q, want completed\nbody: %s", r2.Result.HaltReason, body)
	}

	// (2b) EXACTLY ONE combined ruling, whose prompt carries BOTH Jonas's held attempt content and
	// the reaction's first action, each pinned to ITS OWN "ACTOR <id> ATTEMPTS: …" line
	// (buildResolvePrompt, resolveprompt.go:24-31). A bare substring check on the whole prompt is not
	// enough: windUp, pressText, id.J and id.K are all ALSO independently recoverable from the FACTS/
	// gather_slice section above the ATTEMPT(S) block, so mis-attribution inside the ACTOR lines
	// themselves (e.g. Jonas's wind-up wrongly landing on the player's line) would slip past it.
	if resolve2.calls != 1 {
		t.Fatalf("beat 2 resolve calls = %d, want 1 (held act + first action → ONE combined ruling)", resolve2.calls)
	}
	p := resolve2.prompts[0]
	jSeg, ok := actorAttemptSegment(p, id.J)
	if !ok {
		t.Fatalf("beat 2 combined ruling prompt missing an \"ACTOR %s ATTEMPTS: \" line\nprompt:\n%s", id.J, p)
	}
	kSeg, ok := actorAttemptSegment(p, id.K)
	if !ok {
		t.Fatalf("beat 2 combined ruling prompt missing an \"ACTOR %s ATTEMPTS: \" line\nprompt:\n%s", id.K, p)
	}
	if !strings.Contains(jSeg, windUp) {
		t.Fatalf("beat 2 Jonas's ACTOR %s ATTEMPTS line does not carry his held wind-up %q\nsegment:\n%s", id.J, windUp, jSeg)
	}
	if !strings.Contains(kSeg, pressText) {
		t.Fatalf("beat 2 the player's ACTOR %s ATTEMPTS line does not carry the press %q\nsegment:\n%s", id.K, pressText, kSeg)
	}
	// Pin the swap case: Jonas's wind-up must NOT leak into the player's ACTOR line — that is exactly
	// the mis-attribution a bare prompt-wide substring check would miss.
	if strings.Contains(kSeg, windUp) {
		t.Fatalf("beat 2 the player's ACTOR %s ATTEMPTS line wrongly carries Jonas's wind-up %q (mis-attribution)\nsegment:\n%s", id.K, windUp, kSeg)
	}
	// The remainder whisper must NOT be inside the collision (only the FIRST action joins it).
	if strings.Contains(p, whisperText) {
		t.Fatalf("beat 2 combined ruling prompt wrongly contains the remainder whisper (only the first action joins the collision)")
	}

	// (2c) The held row flipped 'resolved' inside the ruling's tx — zero pending remain.
	var stillPending int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM held_outcome WHERE world_id=$1 AND status='pending'`, id.World).Scan(&stillPending); err != nil {
		t.Fatalf("beat 2 count pending held: %v", err)
	}
	if stillPending != 0 {
		t.Fatalf("beat 2 pending held rows = %d, want 0 (Jonas's hold resolved in the ruling's tx)", stillPending)
	}
	if s := heldStatus(t, ctx, pool, id.World, id.J); s != "resolved" {
		t.Fatalf("beat 2 held status = %q, want resolved", s)
	}

	// (2d) The ruled canon committed: both combined-ruling events landed under origin='ruling'.
	var rulingCanon int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin='ruling'`, id.World).Scan(&rulingCanon); err != nil {
		t.Fatalf("beat 2 count ruling canon: %v", err)
	}
	if rulingCanon != 2 {
		t.Fatalf("beat 2 ruling canon events = %d, want 2 (both combined-ruling events)", rulingCanon)
	}

	// (2e) The remainder ran AFTER the collision: exactly one freeform (the whisper), and its
	// (tick,seq) is strictly after the latest ruling event's (canon ordering).
	var freeform2 int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin='freeform'`, id.World).Scan(&freeform2); err != nil {
		t.Fatalf("beat 2 count freeform canon: %v", err)
	}
	if freeform2 != 1 {
		t.Fatalf("beat 2 freeform canon events = %d, want 1 (the post-collision whisper remainder)", freeform2)
	}
	var rt, ft int64
	var rs, fs int
	if err := pool.QueryRow(ctx,
		`SELECT in_world_tick, beat_seq FROM canon_event WHERE world_id=$1 AND origin='ruling' ORDER BY in_world_tick DESC, beat_seq DESC LIMIT 1`,
		id.World).Scan(&rt, &rs); err != nil {
		t.Fatalf("beat 2 read latest ruling canon: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT in_world_tick, beat_seq FROM canon_event WHERE world_id=$1 AND origin='freeform'`,
		id.World).Scan(&ft, &fs); err != nil {
		t.Fatalf("beat 2 read freeform canon: %v", err)
	}
	if ft < rt || (ft == rt && fs <= rs) {
		t.Fatalf("beat 2 canon order wrong: freeform remainder (%d,%d) must follow the latest ruling event (%d,%d)", ft, fs, rt, rs)
	}

	// (2f) Narration is non-empty.
	if r2.Narration == "" || strings.Contains(body, `"narration":""`) {
		t.Fatalf("beat 2 narration missing or empty\nbody: %s", body)
	}

	// Backfill perception_subject for this world's runtime commits (mirrors the reaction-beat E2E
	// pattern) so the every-perception-has-a-subject invariant stays satisfied for a passthrough
	// remainder. Non-scoped + ON CONFLICT DO NOTHING: it only fills genuine gaps, never overwrites.
	perceptionSubjectBackfill(t, ctx, pool, int(runtimeBase))
}
