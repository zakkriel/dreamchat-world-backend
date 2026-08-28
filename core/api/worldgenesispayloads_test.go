package main

// SPEC-011 coverage for the six schemas world creation publishes. `make schema-contract` requires a REAL
// payload for every published schema — a new schema with no payload behind it fails the build, which is the rule that
// keeps a contract from being a document nobody checks.
//
// Two kinds are captured here, for the two kinds of contract:
//
//   - world_genesis/1, world_interview/1 and world_kickstart/1 are SEAT contracts: the structured-output
//     leash a driver's raw answer is validated against. Like world_actor/1 and place_author/1 they carry
//     no schema_version field (that envelope belongs to what the API RETURNS, never to what a seat may
//     EMIT), so schema_contract.py recovers the id from the filename. The bytes written are byte-identical
//     to what Driver.Generate returned — no wrapper, no re-marshal, no hand-written fixture, because a
//     hand-written fixture would only prove the schema matches itself.
//
//   - world_genesis_frame/2, world_interview_turn/1 and world_kickstart_turn/1 are API contracts, and
//     they are captured by driving the REAL handlers through httptest. That is the whole value: what CI
//     validates is what a browser receives, including the SSE framing, the field-omission rules and the
//     turn grammar's oneOf branches, rather than a struct someone believed the handler produced.
//
// Both generators are gated on GENESIS_PAYLOAD_DIR — unset in a normal `go test ./...` run, so they skip
// cleanly and write nothing. ci/gen_payloads.sh sets it.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func genesisPayloadDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("GENESIS_PAYLOAD_DIR")
	if dir == "" {
		t.Skip("GENESIS_PAYLOAD_DIR unset — payload generation is for ci/gen_payloads.sh")
	}
	return dir
}

func writePayload(t *testing.T, dir, name string, raw []byte) {
	t.Helper()
	// Validity is asserted here as well as in CI so a broken generator fails at the point it broke.
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatalf("%s is not valid JSON: %v\n%s", name, err, raw)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Logf("wrote %s (%d bytes)", path, len(raw))
}

// TestGenSeatContractPayloads captures the three seat leashes exactly as their drivers answered them.
func TestGenSeatContractPayloads(t *testing.T) {
	dir := genesisPayloadDir(t)
	ctx := context.Background()

	understanding := &recordingDriver{Driver: NewFakeWorldUnderstandingDriver()}
	fill := &recordingDriver{Driver: NewFakeWorldFillDriver()}
	doc, _, err := authorWorld(ctx, understanding, fill, NewFakeWorldFillReviewDriver(), testBrief, nil, nil, nil)
	if err != nil {
		t.Fatalf("authorWorld: %v", err)
	}
	writePayload(t, dir, "world_identity_1.json", []byte(understanding.last))
	h := NewWorldGenesisHandler(nil, false, mustBridgeForIdentityPayload(t), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, jsonPost("/worlds/identity", `{"brief":"`+testBrief+`"}`))
	if rec.Code != 200 {
		t.Fatalf("identity confirm: %d %s", rec.Code, rec.Body.String())
	}
	writePayload(t, dir, "world_identity_confirm_1.json", rec.Body.Bytes())

	writePayload(t, dir, "world_fill_1.json", []byte(fill.last))
	review := &recordingDriver{Driver: NewFakeWorldFillReviewDriver()}
	if _, err := review.Generate(ctx, GenRequest{Prompt: "review", Schema: json.RawMessage(worldFillReviewSchemaJSON)}); err != nil {
		t.Fatalf("review payload: %v", err)
	}
	writePayload(t, dir, "world_fill_review_1.json", []byte(review.last))
	genesis := &recordingDriver{Driver: NewFakeWorldGenesisDriver()}
	if _, err := genesis.Generate(ctx, GenRequest{Prompt: buildWorldGenesisPrompt(testBrief, nil), Schema: json.RawMessage(worldGenesisSchemaJSON)}); err != nil {
		t.Fatalf("legacy genesis payload: %v", err)
	}
	writePayload(t, dir, "world_genesis_1.json", []byte(genesis.last))

	interview := &recordingDriver{Driver: NewFakeWorldInterviewDriver()}
	if _, err := askNextQuestion(ctx, interview, testBrief, nil); err != nil {
		t.Fatalf("askNextQuestion: %v", err)
	}
	writePayload(t, dir, "world_interview_1.json", []byte(interview.last))

	kickstart := &recordingDriver{Driver: NewFakeWorldKickstartDriver()}
	if _, err := authorKickstart(ctx, kickstart, doc, testBrief, doc.Arrival.CanonicalName, ""); err != nil {
		t.Fatalf("authorKickstart: %v", err)
	}
	writePayload(t, dir, "world_kickstart_1.json", []byte(kickstart.last))
}

// erroringWorldGenesisDriver always fails Generate — simulating a world_genesis seat outage, so the
// `error` frame is captured from a real infrastructure fault (build()'s h.fail non-refusal branch)
// rather than a hand-typed fixture. Mirrors beatsstream_test.go's erroringNarrateDriver.
type erroringWorldGenesisDriver struct{}

func (erroringWorldGenesisDriver) Name() string { return "erroring-world-genesis" }
func (erroringWorldGenesisDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (erroringWorldGenesisDriver) Generate(context.Context, GenRequest) (string, error) {
	return "", fmt.Errorf("simulated world_genesis seat outage: connection reset by peer at 10.0.4.19:9443")
}

// TestGenGenesisAPIPayloads drives the real journey — genesis stream through both kickstart turns — and
// captures what the wire carried.
//
// The world it builds is COMMITTED — canon tables carry forbid_delete triggers, so there is no way to
// remove it afterward, and pretending otherwise with a rollback would mean capturing frames from a
// transaction that never happened. A payload run therefore leaves one generated world behind in the CI
// database, which is a seeded throwaway. That is stated rather than hidden because it is the one place in
// this file where the honest option costs something.
func TestGenGenesisAPIPayloads(t *testing.T) {
	dir := genesisPayloadDir(t)
	pool := testPool(t)
	defer pool.Close()

	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldUnderstanding.Name: NewFakeWorldUnderstandingDriver(),
		SeatWorldFill.Name:          NewFakeWorldFillDriver(),
		SeatWorldFillReview.Name:    NewFakeWorldFillReviewDriver(),
		SeatWorldInterview.Name:     NewFakeWorldInterviewDriver(),
		SeatWorldKickstart.Name:     NewFakeWorldKickstartDriver(),
	}, SeatWorldUnderstanding, SeatWorldFill, SeatWorldFillReview, SeatWorldInterview, SeatWorldKickstart)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	h := NewWorldGenesisHandler(pool, true, bridge, nil)

	// The style catalogue, through the real handler. Captured here rather than in a fixture because
	// SPEC-011 validates what the WIRE carried: a published schema with no real payload behind it has
	// only ever been checked against someone's idea of the shape.
	styleRec := httptest.NewRecorder()
	NewWorldArtStylesHandler().ServeHTTP(styleRec, httptest.NewRequest(http.MethodGet, "/worlds/art-styles", nil))
	if styleRec.Code != http.StatusOK {
		t.Fatalf("art-styles status = %d, want 200", styleRec.Code)
	}
	writePayload(t, dir, "art_styles_1.json", bytes.TrimSpace(styleRec.Body.Bytes()))

	// The interview turn, through the real handler.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, jsonPost("/worlds/interview", `{"brief":"`+testBrief+`"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("interview status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	writePayload(t, dir, "world_interview_turn_1.json", bytes.TrimSpace(rec.Body.Bytes()))

	// The build stream. Every distinct frame KIND from a successful build is captured, because a schema
	// whose oneOf branches are never all exercised is a schema that has only been half checked. The
	// stream now ends in `choice` (candidates on the brief), never `world` — commit moved to the
	// kickstart route (spec, phase 1).
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, jsonPost("/worlds/genesis", `{"brief":"`+testBrief+`"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("genesis status = %d, want 200", rec.Code)
	}
	byKind := map[string][]byte{}
	var choice map[string]any
	for _, frame := range sseFrames(t, rec.Body.String()) {
		var probe struct{ Kind string }
		if err := json.Unmarshal(frame, &probe); err != nil {
			t.Fatalf("frame is not JSON: %v", err)
		}
		byKind[probe.Kind] = frame
		if probe.Kind == "choice" {
			if err := json.Unmarshal(frame, &choice); err != nil {
				t.Fatalf("choice frame is not JSON: %v", err)
			}
		}
	}
	if choice == nil {
		t.Fatalf("the stream produced no `choice` frame, so the build did not pause for the player: %s", rec.Body.String())
	}
	for kind, frame := range byKind {
		writePayload(t, dir, "world_genesis_frame_3_"+kind+".json", frame)
	}

	// Kickstart turn 1: the character choice. Answers the recommended candidate, exactly as a player
	// clicking the highlighted option would — the response is the `done:false` branch of the turn
	// grammar (the next question, with its own options).
	choiceWorldID, _ := choice["world_id"].(string)
	turn1Body, err := json.Marshal(kickstartRequest{WorldID: choiceWorldID, Answer: recommendedLabel(t, choice["options"])})
	if err != nil {
		t.Fatalf("marshal kickstart turn 1 request: %v", err)
	}
	turn1Rec := httptest.NewRecorder()
	h.ServeHTTP(turn1Rec, jsonPost("/worlds/genesis/kickstart", string(turn1Body)))
	if turn1Rec.Code != http.StatusOK {
		t.Fatalf("kickstart turn 1 status = %d, want 200 (body %s)", turn1Rec.Code, turn1Rec.Body.String())
	}
	writePayload(t, dir, "world_kickstart_turn_2_question.json", bytes.TrimSpace(turn1Rec.Body.Bytes()))

	var turn1 map[string]any
	if err := json.Unmarshal(turn1Rec.Body.Bytes(), &turn1); err != nil {
		t.Fatalf("kickstart turn 1 response is not JSON: %v", err)
	}
	if turn1["done"] != false {
		t.Fatalf("kickstart turn 1 done = %v, want false — both oneOf branches validate against the same "+
			"schema, so a regressed commit path would still pass the schema-contract gate while this "+
			"capture silently stopped being the `question` branch it is named for", turn1["done"])
	}

	// Kickstart turn 2: the scenario choice — the one transaction that commits (AC-2). The response is
	// the `done:true` branch, carrying the playable world.
	turn2Body, err := json.Marshal(kickstartRequest{WorldID: choiceWorldID, Answer: recommendedLabel(t, turn1["options"])})
	if err != nil {
		t.Fatalf("marshal kickstart turn 2 request: %v", err)
	}
	turn2Rec := httptest.NewRecorder()
	h.ServeHTTP(turn2Rec, jsonPost("/worlds/genesis/kickstart", string(turn2Body)))
	if turn2Rec.Code != http.StatusOK {
		t.Fatalf("kickstart turn 2 status = %d, want 200 (body %s)", turn2Rec.Code, turn2Rec.Body.String())
	}
	var turn2 map[string]any
	if err := json.Unmarshal(turn2Rec.Body.Bytes(), &turn2); err != nil {
		t.Fatalf("kickstart turn 2 response is not JSON: %v", err)
	}
	if turn2["done"] != true {
		t.Fatalf("kickstart turn 2 done = %v, want true — same schema, both branches: a stalled commit "+
			"would still validate while world_kickstart_turn_2_world.json silently captured a question "+
			"instead of the world branch it is named for", turn2["done"])
	}
	writePayload(t, dir, "world_kickstart_turn_2_world.json", bytes.TrimSpace(turn2Rec.Body.Bytes()))

	// The refusal branch, from a brief the seat cannot answer (the wrong-shaped fake decodes as a JSON
	// array, which cannot unmarshal into genesisDoc) — captured so the `refused` shape is covered by a
	// real refusal rather than a hand-made one.
	refusing, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldUnderstanding.Name: NewFakeStructuredDriver("fake-structured", nil),
		SeatWorldFill.Name:          NewFakeStructuredDriver("fake-structured", nil),
		SeatWorldFillReview.Name:    NewFakeWorldFillReviewDriver(),
	}, SeatWorldUnderstanding, SeatWorldFill, SeatWorldFillReview)
	if err != nil {
		t.Fatalf("refusing bridge: %v", err)
	}
	rec = httptest.NewRecorder()
	NewWorldGenesisHandler(pool, true, refusing, nil).ServeHTTP(rec, jsonPost("/worlds/genesis", `{"brief":"anything"}`))
	for _, frame := range sseFrames(t, rec.Body.String()) {
		var probe struct{ Kind string }
		_ = json.Unmarshal(frame, &probe)
		if probe.Kind == "refused" {
			writePayload(t, dir, "world_genesis_frame_3_refused.json", frame)
		}
	}

	// The fault branch, from a seat that cannot answer at all — captured so the `error` shape is
	// covered by a real infrastructure fault rather than a hand-made one.
	erroring, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldUnderstanding.Name: erroringWorldGenesisDriver{},
		SeatWorldFill.Name:          NewFakeWorldFillDriver(),
		SeatWorldFillReview.Name:    NewFakeWorldFillReviewDriver(),
	}, SeatWorldUnderstanding, SeatWorldFill, SeatWorldFillReview)
	if err != nil {
		t.Fatalf("erroring bridge: %v", err)
	}
	rec = httptest.NewRecorder()
	NewWorldGenesisHandler(pool, true, erroring, nil).ServeHTTP(rec, jsonPost("/worlds/genesis", `{"brief":"anything"}`))
	for _, frame := range sseFrames(t, rec.Body.String()) {
		var probe struct{ Kind string }
		_ = json.Unmarshal(frame, &probe)
		if probe.Kind == "error" {
			writePayload(t, dir, "world_genesis_frame_3_error.json", frame)
		}
	}
}

func jsonPost(path, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "http://localhost:8080"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// sseFrames splits an SSE body into its `data:` payloads.
func sseFrames(t *testing.T, body string) [][]byte {
	t.Helper()
	var out [][]byte
	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		out = append(out, []byte(strings.TrimPrefix(line, "data: ")))
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan sse: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("no frames in stream body: %q", body)
	}
	return out
}

func mustBridgeForIdentityPayload(t *testing.T) *Bridge {
	t.Helper()
	b, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldUnderstanding.Name: NewFakeWorldUnderstandingDriver(),
	}, SeatWorldUnderstanding)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
