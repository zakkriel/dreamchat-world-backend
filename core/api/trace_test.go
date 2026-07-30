package main

// Grounded Reasoning / Unit 3 — THE REASONING TRACE (BeatTrace → reasoning_log).
//
// The trace is a PURE CAPTURE of what the beat pipeline already computed (no new LLM): the decoded
// chain, per-attempt physics (the fact sheet + the move gate), the world-first decisions, the
// adjudicated ruling's reasoning→therefore→outcome, and the halt. It is TRUTH-REVEALING (it can carry a
// secret's truth-side reasoning), so it is emitted under `reasoning_log` ONLY when the handler is in
// debug mode; a non-debug response carries NO `reasoning_log` key at all (the security-relevant
// invariant — a real player never gets debug, RULINGS-2026-07-23 §9, design Unit 3).
//
// These tests drive the WHOLE path through the REAL HTTP beatHandler (the station_f/query exit-gate
// pattern), toggling the handler's debug flag, and assert on the shipped JSON. Each test mints a fresh,
// random world (setupQueryWorld) so counts see only its rows.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// reasoningLog mirrors the BeatTrace JSON the handler emits under `reasoning_log` — the developer view.
type reasoningLog struct {
	Decompose []struct {
		Type   string   `json:"type"`
		Stated string   `json:"stated"`
		IDs    []string `json:"ids"`
	} `json:"decompose"`
	WorldFirst []struct {
		ActorID  string `json:"actor_id"`
		Decision string `json:"decision"`
		Stated   string `json:"stated"`
	} `json:"world_first"`
	Attempts []struct {
		Type          string          `json:"type"`
		Stated        string          `json:"stated"`
		FactSheet     json.RawMessage `json:"fact_sheet"`
		MoveDurationS int64           `json:"move_duration_s"`
	} `json:"attempts"`
	Rulings []struct {
		ActorIDs  []string `json:"actor_ids"`
		Reasoning string   `json:"reasoning"`
		Therefore string   `json:"therefore"`
		Outcome   string   `json:"outcome"`
	} `json:"rulings"`
	HaltReason string   `json:"halt_reason"`
	Committed  []string `json:"committed"`
}

type traceBeatResp struct {
	Narration string          `json:"narration"`
	Messages  json.RawMessage `json:"messages"`
	Result    struct {
		Committed     []string `json:"committed"`
		HaltReason    string   `json:"halt_reason"`
		TicksAdvanced int64    `json:"ticks_advanced"`
	} `json:"result"`
	ReasoningLog *reasoningLog `json:"reasoning_log"`
}

// traceHandler builds the real HTTP handler with a scripted decompose table + a fixed-ruling resolve
// driver (a sentinel when the beat should never reach the referee) + a deterministic text narrator, so a
// test can toggle debug and read back the shipped reasoning_log. Cognition/world-actor stay quiet fakes.
func traceHandler(t *testing.T, pool *pgxpool.Pool, debug bool, decomposeText, chainJSON, resolveRuling string) (http.Handler, *capturingResolveDriver) {
	t.Helper()
	dec := NewFakeStructuredDriver("fake-structured:trace", map[string]string{decomposeText: chainJSON})
	resolve := &capturingResolveDriver{name: "capture-resolve:trace", ruling: resolveRuling}
	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name:         dec,
		SeatNarrate.Name:           NewFakeTextDriver("fake-text:trace"),
		SeatResolve.Name:           resolve,
		SeatCognitionBatch.Name:    NewFakeCognitionDriver(),
		SeatCognitionIsolated.Name: NewFakeCognitionDriver(),
		SeatWorldActor.Name:        NewFakeWorldActorDriver(),
	}, SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return NewBeatHandler(pool, debug, bridge), resolve
}

// TestTrace_DebugBeat_ReasoningLogHasMovePhysicsAndHalt is the brief's (a): a DEBUG move beat ships a
// `reasoning_log` object carrying the committed move's distance/duration (from the fact sheet) and the
// halt reason. K→bar is 8 m / 6 s (setupQueryWorld's geometry).
func TestTrace_DebugBeat_ReasoningLogHasMovePhysicsAndHalt(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	chain := `[{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"}]`
	h, resolve := traceHandler(t, pool, true, move, chain, `SHOULD NOT BE CALLED`)

	code, body := postBeat(h, id.World, id.K, move)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var r traceBeatResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}

	// A passthrough move never reaches the referee.
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (a passthrough move never reaches the referee)", resolve.calls)
	}
	// The move committed and completed.
	if r.Result.HaltReason != "completed" || len(r.Result.Committed) != 1 {
		t.Fatalf("result halt=%q committed=%v, want completed/1", r.Result.HaltReason, r.Result.Committed)
	}

	// (a) the reasoning_log is present under debug.
	if r.ReasoningLog == nil {
		t.Fatalf("debug beat shipped NO reasoning_log\nbody: %s", body)
	}
	rl := r.ReasoningLog

	// The halt reason rides the trace.
	if rl.HaltReason != "completed" {
		t.Fatalf("reasoning_log.halt_reason = %q, want completed", rl.HaltReason)
	}
	// The decoded chain is captured (the move, bound to the bar id).
	if len(rl.Decompose) != 1 || rl.Decompose[0].Type != "ActorMoved" {
		t.Fatalf("reasoning_log.decompose = %+v, want one ActorMoved", rl.Decompose)
	}
	// The per-attempt physics: the move's computed duration (6 s) + the fact sheet's distance/duration.
	if len(rl.Attempts) != 1 || rl.Attempts[0].Type != "ActorMoved" {
		t.Fatalf("reasoning_log.attempts = %+v, want one ActorMoved attempt", rl.Attempts)
	}
	if rl.Attempts[0].MoveDurationS != 6 {
		t.Fatalf("reasoning_log.attempts[0].move_duration_s = %d, want 6 (CEIL(8/1.4))", rl.Attempts[0].MoveDurationS)
	}
	var fs struct {
		Targets []struct {
			DistanceM     json.Number `json:"distance_m"`
			MoveDurationS json.Number `json:"move_duration_s"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(rl.Attempts[0].FactSheet, &fs); err != nil {
		t.Fatalf("reasoning_log.attempts[0].fact_sheet not the fn_fact_sheet shape: %v\n%s", err, rl.Attempts[0].FactSheet)
	}
	if len(fs.Targets) != 1 {
		t.Fatalf("fact sheet targets = %+v, want the bar only", fs.Targets)
	}
	if !strings.HasPrefix(fs.Targets[0].DistanceM.String(), "8") {
		t.Fatalf("fact sheet distance_m = %q, want ~8 m to the bar", fs.Targets[0].DistanceM.String())
	}
	if fs.Targets[0].MoveDurationS.String() != "6" {
		t.Fatalf("fact sheet move_duration_s = %q, want 6 s", fs.Targets[0].MoveDurationS.String())
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestTrace_NonDebugBeat_NoReasoningLogKey is the brief's (b) — the SECURITY-relevant assertion: a
// NON-debug beat's response has NO `reasoning_log` key AT ALL (not null — absent). The trace is
// truth-revealing, so a real player (never in debug) must never receive it.
func TestTrace_NonDebugBeat_NoReasoningLogKey(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	chain := `[{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"}]`
	// debug=false. ResolveViewer ignores ?viewer= without debug and resolves the actor named 'Player'
	// (= K in setupQueryWorld), so the same move runs.
	h, _ := traceHandler(t, pool, false, move, chain, `SHOULD NOT BE CALLED`)

	code, body := postBeat(h, id.World, id.K, move)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}

	// THE WALL: the raw response text must not carry the key at all.
	if strings.Contains(body, "reasoning_log") {
		t.Fatalf("non-debug response leaked a reasoning_log key (truth-revealing → debug-only):\n%s", body)
	}
	// Structural proof of absence (not merely null): the key is not present in the top-level object.
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}
	if _, present := top["reasoning_log"]; present {
		t.Fatalf("non-debug response has a reasoning_log key (must be ABSENT):\n%s", body)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestTrace_NonDebugBeat_ResponseShapeUnchanged is the brief's (d): the nil trace changes NOTHING. A
// non-debug beat's response carries exactly the pre-existing key set {schema_version, narration,
// messages, result} — the trace threading added no field and altered no other.
func TestTrace_NonDebugBeat_ResponseShapeUnchanged(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const move = "I walk over to the bar"
	chain := `[{"type":"ActorMoved","stated":"` + move + `","to_target_id":"` + id.Bar + `"}]`
	h, _ := traceHandler(t, pool, false, move, chain, `SHOULD NOT BE CALLED`)

	code, body := postBeat(h, id.World, id.K, move)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}
	want := map[string]bool{"schema_version": true, "narration": true, "messages": true, "result": true}
	for k := range top {
		if !want[k] {
			t.Fatalf("non-debug response gained an unexpected key %q (nil trace must change nothing):\n%s", k, body)
		}
	}
	for k := range want {
		if _, ok := top[k]; !ok {
			t.Fatalf("non-debug response dropped the pre-existing key %q:\n%s", k, body)
		}
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}

// TestTrace_AdjudicatedRuling_ThereforeCaptured is the brief's (c): an adjudicated attempt drives the
// referee (a scripted ruling), and the trace records that ruling's reasoning→therefore→outcome. The
// distinctive reasoning string proves it is THIS ruling captured, not a coincidence.
func TestTrace_AdjudicatedRuling_ThereforeCaptured(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupQueryWorld(t, ctx, pool)

	const marker = "Kade's grip tests the bar; it holds firm."
	ruling := `{"reasoning":"` + marker + `","therefore":"succeeds","outcome":{"kind":"resolved","events":[` +
		`{"type":"AttributeChanged","actor_id":"` + id.K + `","target_id":"` + id.Bar + `","truth":"The bar holds.","appearance":"The bar holds.","visible":true}` +
		`]}}`

	const grip = "I grip the bar to steady myself"
	chain := `[{"type":"AttributeChanged","stated":"` + grip + `","target_id":"` + id.Bar + `"}]`
	h, resolve := traceHandler(t, pool, true, grip, chain, ruling)

	code, body := postBeat(h, id.World, id.K, grip)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", code, body)
	}
	var r traceBeatResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("parse: %v\nbody: %s", err, body)
	}

	// The attempt actually reached the referee (proving the ruling in the trace is a real capture).
	if resolve.calls != 1 {
		t.Fatalf("resolve calls = %d, want 1 (an AttributeChanged is adjudicated)", resolve.calls)
	}
	if r.ReasoningLog == nil {
		t.Fatalf("debug beat shipped NO reasoning_log\nbody: %s", body)
	}
	if len(r.ReasoningLog.Rulings) != 1 {
		t.Fatalf("reasoning_log.rulings = %+v, want exactly one adjudicated ruling", r.ReasoningLog.Rulings)
	}
	got := r.ReasoningLog.Rulings[0]
	if got.Therefore != "succeeds" {
		t.Fatalf("reasoning_log.rulings[0].therefore = %q, want succeeds", got.Therefore)
	}
	if !strings.Contains(got.Reasoning, "grip tests the bar") {
		t.Fatalf("reasoning_log.rulings[0].reasoning = %q, want the scripted ruling's reasoning", got.Reasoning)
	}
	if got.Outcome != "resolved" {
		t.Fatalf("reasoning_log.rulings[0].outcome = %q, want resolved", got.Outcome)
	}

	perceptionSubjectBackfill(t, ctx, pool, 0)
}
