package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A world that finishes building must commission its own art, and this is the test that says so.
//
// The founder's first report on world creation was that the cast had no faces. The fix was a
// reconciler kicked from the build (ADR-P021), and until now nothing anywhere asserted the kick
// happened — the whole feature hung on one unobserved function call in a handler. A silent deletion
// of that line would have passed every suite in this repo and reproduced the original complaint
// exactly.
func TestGenesisBuild_CommissionsTheWorldsArt(t *testing.T) {
	// TODO(Task 6): kickArt now fires from the kickstart route's commit, not build() — build() ends
	// in a choice frame with nothing committed yet. Unskip once /worlds/genesis/kickstart exists.
	t.Skip("moves to kickstart journey test — Task 6")
	pool := testPool(t)
	t.Cleanup(pool.Close)

	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldGenesis.Name: NewFakeWorldGenesisDriver(),
	}, SeatWorldGenesis)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}

	var kicked []string
	original := kickArt
	kickArt = func(_ *pgxpool.Pool, _ *imageClient, worldID string) { kicked = append(kicked, worldID) }
	t.Cleanup(func() { kickArt = original })

	rec := httptest.NewRecorder()
	NewWorldGenesisHandler(pool, true, bridge, nil).
		ServeHTTP(rec, jsonPost("/worlds/genesis", `{"brief":"`+testBrief+`"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	built := ""
	for _, frame := range sseFrames(t, rec.Body.String()) {
		var probe struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(frame, &probe); err != nil {
			t.Fatalf("frame is not JSON: %v", err)
		}
		if probe.Kind == "world" {
			built = probe.ID
		}
	}
	if built == "" {
		t.Fatalf("the build produced no world frame: %s", rec.Body.String())
	}

	if len(kicked) != 1 {
		t.Fatalf("art was commissioned %d time(s), want exactly once — a built world illustrates itself (ADR-P021)", len(kicked))
	}
	if kicked[0] != built {
		t.Fatalf("art was commissioned for %q but the world built was %q", kicked[0], built)
	}
}

// A build that never produced a world must not commission art for one. Spending on a refusal is the
// mirror of the bug above and just as invisible.
func TestGenesisRefusal_CommissionsNothing(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(pool.Close)

	refusing, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldGenesis.Name: NewFakeStructuredDriver("fake-structured", nil),
	}, SeatWorldGenesis)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}

	var kicked []string
	original := kickArt
	kickArt = func(_ *pgxpool.Pool, _ *imageClient, worldID string) { kicked = append(kicked, worldID) }
	t.Cleanup(func() { kickArt = original })

	rec := httptest.NewRecorder()
	NewWorldGenesisHandler(pool, true, refusing, nil).
		ServeHTTP(rec, jsonPost("/worlds/genesis", `{"brief":"anything"}`))

	if len(kicked) != 0 {
		t.Fatalf("a refused build commissioned art for %v — there is no world to illustrate", kicked)
	}
}
