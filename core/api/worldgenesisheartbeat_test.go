package main

// The build stream must never go quiet while the author writes: a silent minute is indistinguishable
// from a hang, and the founder watched exactly that against a slow provider. These tests pin the
// heartbeat — steady `working` frames carrying only measured fact — around a deliberately slow seat,
// and pin that the heartbeats STOP the moment the author answers.

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// slowGenesisDriver stands in for a provider that takes its time and then refuses. Refusal rather
// than success on purpose: the refusal path never touches the database, so this test needs no pool
// and leaves nothing behind, while still exercising the full authoring wait the heartbeat covers.
type slowGenesisDriver struct{ delay time.Duration }

func (slowGenesisDriver) Name() string { return "slow-genesis" }
func (slowGenesisDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d slowGenesisDriver) Generate(ctx context.Context, _ GenRequest) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(d.delay):
	}
	return "", refuse("the author was interrupted mid-sentence")
}

func TestWorldGenesis_TheBuildStreamNeverGoesQuiet(t *testing.T) {
	oldEvery := genesisHeartbeatEvery
	genesisHeartbeatEvery = 25 * time.Millisecond
	t.Cleanup(func() { genesisHeartbeatEvery = oldEvery })

	bridge, err := NewBridgeWithDrivers(
		map[string]Driver{SeatWorldUnderstanding.Name: slowGenesisDriver{delay: 250 * time.Millisecond}},
		SeatWorldUnderstanding)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}

	rec := httptest.NewRecorder()
	NewWorldGenesisHandler(nil, true, bridge, nil).ServeHTTP(rec,
		jsonPost("/worlds/genesis", `{"brief":"a slow world"}`))

	heartbeats := 0
	last := ""
	for _, raw := range sseFrames(t, rec.Body.String()) {
		var probe struct {
			Kind   string `json:"kind"`
			Stated string `json:"stated"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			t.Fatalf("frame does not parse: %v\n%s", err, raw)
		}
		last = probe.Kind
		if probe.Kind == "working" && strings.Contains(probe.Stated, "Still writing") {
			heartbeats++
			// The line must carry the one measured value it is allowed: elapsed wall time.
			if !strings.Contains(probe.Stated, "seconds in") {
				t.Fatalf("heartbeat %q does not state the measured elapsed time", probe.Stated)
			}
		}
	}
	if heartbeats < 2 {
		t.Fatalf("got %d heartbeat frame(s) across a 250ms authoring call at a 25ms interval — the stream went quiet", heartbeats)
	}
	// The refusal must be the LAST frame: a heartbeat emitted after the stream has answered would mean
	// the ticker outlived the authoring call it reports on.
	if last != "refused" {
		t.Fatalf("stream ended with %q, want the refusal after the slow author answered", last)
	}
}
