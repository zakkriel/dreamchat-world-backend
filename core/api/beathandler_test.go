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

func (rogueStructuredDriver) Name() string { return "rogue-structured" }
func (rogueStructuredDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
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
	h := NewBeatsStreamHandler(pool, true, bridge) // debug → honor ?viewer=

	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beats?viewer="+playerID, strings.NewReader(`{"text":"tell mara about the note"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	rawBody := rec.Body.String()
	if strings.Contains(rawBody, `"status":"accepted"`) {
		t.Fatalf("response leaked a raw canon row (B-1): %s", rawBody)
	}
	collapsed, err := collapseBeatFrames(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("collapse beat frames: %v\n%s", err, rawBody)
	}
	body := string(collapsed)
	if !strings.Contains(body, "narration") {
		t.Fatalf("response missing narration: %s", body)
	}
	if !strings.Contains(body, `"halt_reason":"completed"`) {
		t.Fatalf("beat did not commit end-to-end (halt_reason != completed): %s", body)
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
	h := NewBeatsStreamHandler(pool, true, mustBridge(t, NewFakeStructuredDriver("fake-structured:test", map[string]string{}), NewFakeTextDriver("fake-text:test")))

	// >64KB body: a text field padded well past the cap. MaxBytesReader trips during decode → 400.
	// (Decode fails before decompose is ever called, so the driver bodies are irrelevant here.)
	huge := `{"text":"` + strings.Repeat("a", 70<<10) + `"}`
	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beats?viewer="+playerID, strings.NewReader(huge))
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

func (d *scriptedNarrateDriver) Name() string { return d.name }
func (d *scriptedNarrateDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
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
	h := NewBeatsStreamHandler(pool, true, bridge)
	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+worldID+"/beats?viewer="+playerID, strings.NewReader(`{"text":"I look around"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	collapsed, err := collapseBeatFrames(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("collapse beat frames: %v\n%s", err, rec.Body.String())
	}
	var resp beatMsgResp
	if err := json.Unmarshal(collapsed, &resp); err != nil {
		t.Fatalf("decode response: %v\n%s", err, collapsed)
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

// ── Task 11 — the candidate whitelist is everything the actor PERCEIVES (RULINGS-2026-07-30 §1) ──────
// Fixed play-seed uuids (seed_drowned_lantern.sql). The `dl` prefix keeps them clear of the world-1111
// fixture constants (worldID/playerID/maraID) and of station_f_exit_test.go's `kadeID` PARAMETER name.
const (
	dlWorldID   = "22222222-2222-2222-2222-222222222222"
	dlKadeID    = "2ac70000-0000-0000-0000-0000000000a1"
	dlBarID     = "2a7f0000-0000-0000-0000-0000000000f1" // the bar — co-located in the tavern
	dlCrateID   = "2a7f0000-0000-0000-0000-0000000000f2" // ballast crate — co-located in the tavern
	dlNoteID    = "2a7f0000-0000-0000-0000-0000000000b1" // sealed note — carried by Kade (contained_by=Kade)
	dlMaraKeyID = "2a7f0000-0000-0000-0000-0000000000d1" // cellar key — held by MARA, not Kade (the wall)
	dlStoneID   = "2a7f0000-0000-0000-0000-0000000000f3" // ballast stone — nested INSIDE the crate (the wall)
)

// TestPayload_PerceivedCandidates_DrownedLantern asserts the widened candidate whitelist against the
// SEEDED play world: Kade's candidates now include every artifact he PERCEIVES — the bar he can approach
// and the ballast crate he can grab (both co-located in the tavern), plus the sealed note he carries
// (contained_by = Kade) — each by its real id with a viewer-relative label (fn_display_name), kind
// "artifact". The naming-reach wall (RULINGS-2026-07-23 §3) still holds: an item on ANOTHER actor's
// person (Mara's cellar key) and a thing nested INSIDE a container (the ballast stone) are NOT bindable
// by Kade. Requires the play seed (make reset precedes go test in the battery); payload only reads.
func TestPayload_PerceivedCandidates_DrownedLantern(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	h := &beatHandler{pool: pool, dbg: true}

	p, err := h.payload(ctx, dlWorldID, dlKadeID)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	byID := map[string]Candidate{}
	for _, c := range p.Candidates {
		byID[c.ID] = c
	}

	// The perceived artifacts are bindable, each with a non-empty viewer-relative label and kind "artifact".
	for _, want := range []struct{ id, what string }{
		{dlBarID, "the bar (co-located artifact)"},
		{dlCrateID, "the ballast crate (co-located artifact)"},
		{dlNoteID, "the sealed note (carried item)"},
	} {
		c, ok := byID[want.id]
		if !ok {
			t.Fatalf("candidate whitelist is missing %s (%s) — a perceived artifact must be bindable (RULINGS-2026-07-30 §1)", want.id, want.what)
		}
		if c.Kind != "artifact" {
			t.Errorf("%s kind = %q, want \"artifact\"", want.what, c.Kind)
		}
		if c.Name == "" {
			t.Errorf("%s has an empty label — candidates carry the viewer's fn_display_name (viewer-relative)", want.what)
		}
	}

	// The wall: an item on another actor's person, and a thing nested in a container, are NOT perceived
	// as bindable by Kade (perception-bounded, not a global id dump).
	if _, leaked := byID[dlMaraKeyID]; leaked {
		t.Errorf("Mara's cellar key leaked into Kade's candidates — carried items are the VIEWER's own only")
	}
	if _, leaked := byID[dlStoneID]; leaked {
		t.Errorf("the ballast stone (nested in the crate) leaked in — co-location is by location_id, not nesting")
	}
}

// TestPayload_PerceivedCandidates_OtherLocationExcluded builds a self-contained world to nail the
// naming-reach wall directly: a viewer in a tavern perceives the bar + crate co-located there and the
// note he carries, but an artifact placed in ANOTHER location (a cellar) is NOT a candidate — the
// candidate set is perception-bounded, never a global id dump (RULINGS-2026-07-30 §1 + 2026-07-23 §3).
// Hermetic: fresh random uuids, no make-reset dependency; payload only reads.
func TestPayload_PerceivedCandidates_OtherLocationExcluded(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	var world, tavern, cellar, viewer, bar, crate, note, farLantern string
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&world, &tavern, &cellar, &viewer, &bar, &crate, &note, &farLantern); err != nil {
		t.Fatalf("mint ids: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($3,$1,'location','Tavern'),
		 ($4,$1,'location','Cellar'),
		 ($2,$1,'actor',   'Viewer'),
		 ($5,$1,'artifact','the bar'),
		 ($6,$1,'artifact','a crate'),
		 ($7,$1,'artifact','a note'),
		 ($8,$1,'artifact','a far lantern')`,
		world, viewer, tavern, cellar, bar, crate, note, farLantern); err != nil {
		t.Fatalf("registry: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		 ($2,$1, jsonb_build_object('location_id',$3::text))`,
		world, viewer, tavern); err != nil {
		t.Fatalf("actor_state: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
		 ($5,$1, jsonb_build_object('location_id',$3::text)),    -- bar co-located in the tavern
		 ($6,$1, jsonb_build_object('location_id',$3::text)),    -- crate co-located in the tavern
		 ($7,$1, jsonb_build_object('contained_by',$2::text)),   -- note carried by the viewer
		 ($8,$1, jsonb_build_object('location_id',$4::text))     -- lantern in ANOTHER location (the cellar)
		`,
		world, viewer, tavern, cellar, bar, crate, note, farLantern); err != nil {
		t.Fatalf("artifact_state: %v", err)
	}

	h := &beatHandler{pool: pool, dbg: true}
	p, err := h.payload(ctx, world, viewer)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	in := map[string]bool{}
	for _, c := range p.Candidates {
		in[c.ID] = true
	}
	for _, want := range []string{bar, crate, note} {
		if !in[want] {
			t.Errorf("perceived artifact %s missing from candidates (co-located/carried must bind)", want)
		}
	}
	if in[farLantern] {
		t.Errorf("an artifact in ANOTHER location leaked into candidates — the naming-reach wall must hold (perception-bounded, not global)")
	}
}

// Defense-in-depth: a misbehaving structured driver that emits out-of-vocab is rejected by the
// handler's belt (the primary leash is the capability floor + constrained decoding; this is the
// backstop, SPEC-015/D-1).
//
// The REJECTION is unchanged; the transport is. It used to be a 422 with a text/plain body, sent to
// a client that asked for text/event-stream — which a browser mid-play does not experience as an
// error message but as the connection dying, with an edge proxy free to substitute its own page.
// Decompose now runs inside the stream and refuses with an honest `error` frame, like every other
// failure in this handler. What must not change, and is asserted here, is that nothing reaches canon
// and the player is told something.
func TestBeat_OutOfVocabularyRejectedByBelt(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	var before int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM canon_event WHERE world_id=$1`, worldID).Scan(&before); err != nil {
		t.Fatalf("count canon before: %v", err)
	}

	h := NewBeatsStreamHandler(pool, true, mustBridge(t, rogueStructuredDriver{}, NewFakeTextDriver("fake-text:test")))
	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beats?viewer="+playerID,
		strings.NewReader(`{"text":"anything"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200 — the refusal rides the stream as a frame", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	var kinds []string
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var f struct {
			Kind    string `json:"kind"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(line[5:])), &f); err != nil {
			continue
		}
		kinds = append(kinds, f.Kind)
		if f.Kind == "error" && strings.TrimSpace(f.Message) == "" {
			t.Fatal("the error frame carries no message — a silent refusal is the bug this replaces")
		}
	}
	if len(kinds) != 1 || kinds[0] != "error" {
		t.Fatalf("frames = %v, want exactly one error frame (no interpretation frame for a chain the belt refused)", kinds)
	}

	var after int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM canon_event WHERE world_id=$1`, worldID).Scan(&after); err != nil {
		t.Fatalf("count canon after: %v", err)
	}
	if after != before {
		t.Fatalf("canon grew %d→%d — an out-of-vocabulary chain reached the world", before, after)
	}
}
