package main

import (
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
	var kicked []string
	original := kickArt
	kickArt = func(_ *pgxpool.Pool, _ *imageClient, worldID string) { kicked = append(kicked, worldID) }
	t.Cleanup(func() { kickArt = original })

	// The full journey: genesis ends in a choice, two kickstart answers commit. Art is commissioned
	// from the kickstart route's own commit now (build() no longer commits anything), so the kick has
	// to be observed there or it proves nothing (ADR-P021 — the founder's first report on world
	// creation was a created world with no pictures, from exactly this kind of unobserved call).
	frames := postGenesisAndCollectFrames(t, `{"brief":"`+testBrief+`"}`)
	choice := frames[len(frames)-1]
	handle, _ := choice["handle"].(string)

	turn := postKickstart(t, handle, recommendedLabel(t, choice["options"]))
	if turn["done"] != false {
		t.Fatalf("turn 1 = %v", turn)
	}
	turn2 := postKickstart(t, handle, recommendedLabel(t, turn["options"]))
	if turn2["done"] != true {
		t.Fatalf("turn 2 = %v", turn2)
	}
	world, ok := turn2["world"].(map[string]any)
	if !ok {
		t.Fatalf("turn 2 carries no world: %v", turn2)
	}
	built, _ := world["id"].(string)
	if built == "" {
		t.Fatalf("the build produced no world id: %v", world)
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
