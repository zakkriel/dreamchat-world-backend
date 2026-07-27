package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// rogueStructuredDriver REPORTS structured output but emits out-of-vocab. It binds to decompose
// (capability floor passes) yet proves the handler's DEFENSE-IN-DEPTH belt (DecodeAndValidateChainV2)
// still rejects out-of-vocab → 422, even if a bound driver misbehaves.
type rogueStructuredDriver struct{}

func (rogueStructuredDriver) Name() string                { return "rogue-structured" }
func (rogueStructuredDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }
func (rogueStructuredDriver) Generate(context.Context, GenRequest) (string, error) {
	return `[{"type":"attack","to":"x"}]`, nil
}

func mustBridge(t *testing.T, decompose, narrate Driver) *Bridge {
	t.Helper()
	b, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:      decompose,
		SeatNarrate.Name:        narrate,
		SeatResolve.Name:        NewFakeResolveDriver(),
		SeatCognitionBatch.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:     NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return b
}

// Happy path: position Player + Mara together at seedTavernID (existing seed entity, no registry row
// added) so the Communicated-gate passes and the beat commits end-to-end through the bridge. Event IDs
// are random per run and ticks start at ≥50000 so re-runs never hit canon_event_pkey or
// uq_ce_accepted_order collisions, and SQL-suite tick-range assertions (all ≤1400) are unaffected.
// After the beat, perception_subject rows are backfilled (mirrors seed_mara_0A) so SQL test 14's
// every-perception-has-a-subject invariant holds even when this test runs before the SQL suite.
// Writes persist (additive, legal origin) — make reset is run before go test per the Makefile.
func TestBeat_HappyPath_CommitsAndNarrates(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Compute base tick: ≥50000 and above any existing max to avoid uq_ce_accepted_order collision.
	var baseTick int
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}

	// Position Player and Mara at seedTavernID using the uuid location model (migration 20260723100001).
	// Using the existing seed entity avoids entity_registry row additions (SQL test 40 asserts count=12).
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→tavern',$2,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep1 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$3,'actor','instigator' FROM ev1
		),
		sm1 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($5::text),$2,0 FROM ev1
		),
		ev2 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','M→tavern',$4,0,'accepted',now(),'public','fast_path')
		  RETURNING event_id
		),
		ep2 AS (
		  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		  SELECT event_id,$6,'actor','instigator' FROM ev2
		),
		sm2 AS (
		  INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		  SELECT $1,event_id,$6,'actor','attrs.location_id',to_jsonb($5::text),$4,0 FROM ev2
		)
		SELECT 1`,
		worldID, baseTick, playerID, baseTick+1, seedTavernID, maraID)
	if err != nil {
		t.Fatalf("setup positions: %v", err)
	}

	// v2 format: Communicated (listener_id + content), not legacy "say".
	bridge := mustBridge(t,
		NewFakeStructuredDriver("fake-structured:test", map[string]string{
			"tell mara about the note": `[{"type":"Communicated","stated":"tell mara about the note","listener_id":"` + maraID + `","content":"the note"}]`,
		}),
		NewFakeTextDriver("fake-text:test"))
	h := NewBeatHandler(pool, true, bridge) // debug → honor ?viewer=

	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+playerID, strings.NewReader(`{"text":"tell mara about the note"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "narration") {
		t.Fatalf("response missing narration: %s", body)
	}
	if !strings.Contains(body, `"halt_reason":"completed"`) {
		t.Fatalf("beat did not commit end-to-end (halt_reason != completed): %s", body)
	}
	if strings.Contains(body, `"status":"accepted"`) {
		t.Fatalf("response leaked a raw canon row (B-1): %s", body)
	}

	// Backfill perception_subject for runtime-generated perceptions (mirrors seed_mara_0A backfill)
	// so SQL test 14's every-perception-has-a-subject invariant holds after this test runs.
	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// §7 injection bound at the HTTP edge: a request body over the 64KB cap (http.MaxBytesReader in
// ServeHTTP) fails to decode and returns 400. The player's raw text can ride into the combined-ruling
// prompt (RULINGS-2026-07-24 §7), so an unbounded body must never become an unbounded prompt.
func TestBeat_OversizeBodyRejected(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewBeatHandler(pool, true, mustBridge(t, NewFakeStructuredDriver("fake-structured:test", map[string]string{}), NewFakeTextDriver("fake-text:test")))

	// >64KB body: a text field padded well past the cap. MaxBytesReader trips during decode → 400.
	// (Decode fails before decompose is ever called, so the driver bodies are irrelevant here.)
	huge := `{"text":"` + strings.Repeat("a", 70<<10) + `"}`
	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+playerID, strings.NewReader(huge))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body over the 64KB cap)", rec.Code)
	}
}

// scriptedNarrateDriver returns a canned reply per Generate call (last reply reused past the end) and
// counts calls — so a flow test can drive the structured attempt → repair → plain-fallback sequence and
// assert the exact call count. Reports structured output (the narrate seat requires no capability, but
// the founder-envelope narrate call is structured).
type scriptedNarrateDriver struct {
	name    string
	replies []string
	calls   int
}

func (d *scriptedNarrateDriver) Name() string                { return d.name }
func (d *scriptedNarrateDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }
func (d *scriptedNarrateDriver) Generate(_ context.Context, _ GenRequest) (string, error) {
	i := d.calls
	d.calls++
	if i < len(d.replies) {
		return d.replies[i], nil
	}
	return d.replies[len(d.replies)-1], nil
}

// seatPlayerAndMara positions Player and Mara together at seedTavernID (base tick ≥50000 to dodge
// canon_event uniqueness collisions across re-runs) so a beat viewed as the player finds Mara present in
// the post payload. Mirrors TestBeat_HappyPath's setup. Returns the base tick for perception backfill.
func seatPlayerAndMara(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	var baseTick int
	if err := pool.QueryRow(ctx,
		`SELECT GREATEST(COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,50000)`,
		worldID).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	_, err := pool.Exec(ctx, `
		WITH ev1 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','P→tavern',$2,0,'accepted',now(),'public','fast_path') RETURNING event_id
		),
		ep1 AS (INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier) SELECT event_id,$3,'actor','instigator' FROM ev1),
		sm1 AS (INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		        SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($5::text),$2,0 FROM ev1),
		ev2 AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES (gen_random_uuid(),$1,'move','M→tavern',$4,0,'accepted',now(),'public','fast_path') RETURNING event_id
		),
		ep2 AS (INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier) SELECT event_id,$6,'actor','instigator' FROM ev2),
		sm2 AS (INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
		        SELECT $1,event_id,$6,'actor','attrs.location_id',to_jsonb($5::text),$4,0 FROM ev2)
		SELECT 1`,
		worldID, baseTick, playerID, baseTick+1, seedTavernID, maraID)
	if err != nil {
		t.Fatalf("seat player and mara: %v", err)
	}
	return baseTick
}

type beatMsgResp struct {
	SchemaVersion string        `json:"schema_version"`
	Narration     string        `json:"narration"`
	Messages      []beatMessage `json:"messages"`
}

func runNarrateFlowBeat(t *testing.T, pool *pgxpool.Pool, nd *scriptedNarrateDriver) beatMsgResp {
	t.Helper()
	bridge := mustBridge(t, NewFakeStructuredDriver("fake-structured:test", nil), nd) // empty decompose table → [] chain, beat completes
	h := NewBeatHandler(pool, true, bridge)
	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beat?viewer="+playerID, strings.NewReader(`{"text":"I look around"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp beatMsgResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v\n%s", err, rec.Body.String())
	}
	if resp.SchemaVersion != "beat_result/3" {
		t.Fatalf("schema_version = %q, want beat_result/3", resp.SchemaVersion)
	}
	return resp
}

// Flow: a scripted narrate driver returning valid segments (narrator prose + an attributed action for a
// present NPC) yields messages[] with the VIEWER's display label for that NPC — in exactly one call.
func TestBeat_Narrate_StructuredMessagesWithLabels(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	baseTick := seatPlayerAndMara(t, ctx, pool)

	var wantLabel string
	if err := pool.QueryRow(ctx, `SELECT fn_display_name($1,$2::uuid,$3::uuid)`, worldID, playerID, maraID).Scan(&wantLabel); err != nil {
		t.Fatalf("expected label: %v", err)
	}

	segments := `[{"speaker_id":null,"kind":"narration","text":"The common room holds still."},` +
		`{"speaker_id":"` + maraID + `","kind":"action","text":"Mara sets a tankard on the bar."}]`
	nd := &scriptedNarrateDriver{name: "scripted-narrate", replies: []string{segments}}
	resp := runNarrateFlowBeat(t, pool, nd)

	if nd.calls != 1 {
		t.Fatalf("narrate Generate calls = %d, want 1 (valid on first structured attempt)", nd.calls)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("want 2 messages, got %d: %+v", len(resp.Messages), resp.Messages)
	}
	var sawNarration, sawAction bool
	for _, m := range resp.Messages {
		if m.Kind == "narration" && m.SpeakerID == nil {
			sawNarration = true
		}
		if m.Kind == "action" && m.SpeakerID != nil && *m.SpeakerID == maraID {
			sawAction = true
			if m.SpeakerLabel == "" || m.SpeakerLabel != wantLabel {
				t.Fatalf("action speaker_label = %q, want the viewer's display name %q", m.SpeakerLabel, wantLabel)
			}
		}
	}
	if !sawNarration || !sawAction {
		t.Fatalf("messages missing narrator and/or attributed action: %+v", resp.Messages)
	}
	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// Flow: a ghost-speaker segment is rejected by the belt; the ONE repair re-ask returns valid segments →
// messages after exactly 2 Generate calls.
func TestBeat_Narrate_GhostSpeakerRepairedOnSecondCall(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	baseTick := seatPlayerAndMara(t, ctx, pool)

	ghost := `[{"speaker_id":"99999999-9999-9999-9999-999999999999","kind":"action","text":"A stranger who is not here moves."}]`
	valid := `[{"speaker_id":"` + maraID + `","kind":"action","text":"Mara sets a tankard on the bar."}]`
	nd := &scriptedNarrateDriver{name: "scripted-narrate", replies: []string{ghost, valid}}
	resp := runNarrateFlowBeat(t, pool, nd)

	if nd.calls != 2 {
		t.Fatalf("narrate Generate calls = %d, want 2 (ghost → one repair → valid)", nd.calls)
	}
	if len(resp.Messages) != 1 || resp.Messages[0].Kind != "action" || resp.Messages[0].SpeakerID == nil || *resp.Messages[0].SpeakerID != maraID {
		t.Fatalf("want a single valid action for Mara after repair, got: %+v", resp.Messages)
	}
	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// Flow: two structured failures fall back to a single narrator-blob segment from a plain re-ask (no
// schema) — 3 Generate calls total, and the beat still 200s (never fail on formatting).
func TestBeat_Narrate_FallsBackToProseAfterTwoFailures(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	baseTick := seatPlayerAndMara(t, ctx, pool)

	const blob = "The tavern is quiet, and the damp beams hold their hush."
	nd := &scriptedNarrateDriver{name: "scripted-narrate", replies: []string{"not valid json", "still not json", blob}}
	resp := runNarrateFlowBeat(t, pool, nd)

	if nd.calls != 3 {
		t.Fatalf("narrate Generate calls = %d, want 3 (two structured fails + one plain fallback)", nd.calls)
	}
	if len(resp.Messages) != 1 || resp.Messages[0].Kind != "narration" || resp.Messages[0].SpeakerID != nil {
		t.Fatalf("fallback must be a single narrator segment, got: %+v", resp.Messages)
	}
	if resp.Messages[0].Text != blob || resp.Narration != blob {
		t.Fatalf("fallback text/narration mismatch: msg=%q narration=%q want %q", resp.Messages[0].Text, resp.Narration, blob)
	}
	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// Defense-in-depth: a misbehaving structured driver that emits out-of-vocab is rejected by the
// handler's belt → 422 (the primary leash is the capability floor + constrained decoding; this is
// the backstop, SPEC-015/D-1).
func TestBeat_OutOfVocabularyRejectedByBelt(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	h := NewBeatHandler(pool, true, mustBridge(t, rogueStructuredDriver{}, NewFakeTextDriver("fake-text:test")))
	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beat?viewer="+playerID,
		strings.NewReader(`{"text":"anything"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422 (belt rejects out-of-vocab; SPEC-015)", rec.Code)
	}
}
