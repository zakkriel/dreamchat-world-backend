package main

// SPEC-011 coverage for the four schemas world creation publishes. `make schema-contract` requires a REAL
// payload for every published schema — a new schema with no payload fails the build, which is the rule that
// keeps a contract from being a document nobody checks.
//
// Two kinds are captured here, for the two kinds of contract:
//
//   - world_genesis/1 and world_interview/1 are SEAT contracts: the structured-output leash a driver's raw
//     answer is validated against. Like world_actor/1 and place_author/1 they carry no schema_version field
//     (that envelope belongs to what the API RETURNS, never to what a seat may EMIT), so schema_contract.py
//     recovers the id from the filename. The bytes written are byte-identical to what Driver.Generate
//     returned — no wrapper, no re-marshal, no hand-written fixture, because a hand-written fixture would
//     only prove the schema matches itself.
//
//   - world_genesis_frame/1 and world_interview_turn/1 are API contracts, and they are captured by driving
//     the REAL handlers through httptest. That is the whole value: what CI validates is what a browser
//     receives, including the SSE framing and the field-omission rules, rather than a struct someone
//     believed the handler produced.
//
// Both generators are gated on GENESIS_PAYLOAD_DIR — unset in a normal `go test ./...` run, so they skip
// cleanly and write nothing. ci/gen_payloads.sh sets it.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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

// TestGenSeatContractPayloads captures the two seat leashes exactly as their drivers answered them.
func TestGenSeatContractPayloads(t *testing.T) {
	dir := genesisPayloadDir(t)
	ctx := context.Background()

	genesis := &recordingDriver{Driver: NewFakeWorldGenesisDriver()}
	if _, err := authorWorld(ctx, genesis, testBrief, nil); err != nil {
		t.Fatalf("authorWorld: %v", err)
	}
	writePayload(t, dir, "world_genesis_1.json", []byte(genesis.last))

	interview := &recordingDriver{Driver: NewFakeWorldInterviewDriver()}
	if _, err := askNextQuestion(ctx, interview, testBrief, nil); err != nil {
		t.Fatalf("askNextQuestion: %v", err)
	}
	writePayload(t, dir, "world_interview_1.json", []byte(interview.last))
}

// TestGenGenesisAPIPayloads drives both real routes and captures what the wire carried.
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
		SeatWorldGenesis.Name:   NewFakeWorldGenesisDriver(),
		SeatWorldInterview.Name: NewFakeWorldInterviewDriver(),
	}, SeatWorldGenesis, SeatWorldInterview)
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

	// The build stream. Every distinct frame KIND is captured, because a schema whose oneOf branches are
	// never all exercised is a schema that has only been half checked.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, jsonPost("/worlds/genesis", `{"brief":"`+testBrief+`"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("genesis status = %d, want 200", rec.Code)
	}
	byKind := map[string][]byte{}
	for _, frame := range sseFrames(t, rec.Body.String()) {
		var probe struct{ Kind string }
		if err := json.Unmarshal(frame, &probe); err != nil {
			t.Fatalf("frame is not JSON: %v", err)
		}
		byKind[probe.Kind] = frame
	}
	if _, ok := byKind["world"]; !ok {
		t.Fatalf("the stream produced no `world` frame, so the build did not finish: %s", rec.Body.String())
	}
	for kind, frame := range byKind {
		writePayload(t, dir, "world_genesis_frame_1_"+kind+".json", frame)
	}

	// The refusal branch, from a brief the seat cannot answer — captured so the `refused` shape is covered
	// by a real refusal rather than a hand-made one.
	refusing, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldGenesis.Name: NewFakeStructuredDriver("fake-structured", nil),
	}, SeatWorldGenesis)
	if err != nil {
		t.Fatalf("refusing bridge: %v", err)
	}
	rec = httptest.NewRecorder()
	NewWorldGenesisHandler(pool, true, refusing, nil).ServeHTTP(rec, jsonPost("/worlds/genesis", `{"brief":"anything"}`))
	for _, frame := range sseFrames(t, rec.Body.String()) {
		var probe struct{ Kind string }
		_ = json.Unmarshal(frame, &probe)
		if probe.Kind == "refused" {
			writePayload(t, dir, "world_genesis_frame_1_refused.json", frame)
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
