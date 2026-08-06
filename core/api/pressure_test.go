package main

import (
	"context"
	"testing"
)

// Living World / Task 6 — the deterministic pressure roll. TestDeterministicUnit_Stable is pure (no
// DB): the draw must be a pure function of its four inputs (same inputs → byte-identical output
// across repeated calls), land in [0,1), and move when tick changes. TestRollTier_* exercise the DB
// path (fn_pressure_chance, Task 5) against the seeded Drowned Lantern play world (dlWorldID,
// beathandler_test.go) — `make reset` first, matching orchestrator_worldtime_test.go's own note.

// TestDeterministicUnit_Stable is the brief's own literal test (task-6-brief.md Step 1).
func TestDeterministicUnit_Stable(t *testing.T) {
	a := deterministicUnit("w", 100, 0, "small")
	b := deterministicUnit("w", 100, 0, "small")
	if a != b {
		t.Fatalf("not deterministic: %v %v", a, b)
	}
	if a < 0 || a >= 1 {
		t.Fatalf("out of range: %v", a)
	}
	if deterministicUnit("w", 101, 0, "small") == a {
		t.Fatalf("tick should vary the draw")
	}
}

// TestDeterministicUnit_VariesByAllInputs: beyond tick (already covered above), lastEruption and
// tier must each independently move the draw too — a pure hash of only SOME of the four inputs would
// still pass TestDeterministicUnit_Stable but silently ignore the others.
func TestDeterministicUnit_VariesByAllInputs(t *testing.T) {
	base := deterministicUnit("w", 100, 0, "small")
	if deterministicUnit("w", 100, 1, "small") == base {
		t.Fatalf("lastEruption should vary the draw")
	}
	if deterministicUnit("w", 100, 0, "medium") == base {
		t.Fatalf("tier should vary the draw")
	}
	if deterministicUnit("other-world", 100, 0, "small") == base {
		t.Fatalf("worldID should vary the draw")
	}
}

// TestRollTier_ChanceZeroNeverFires: an unconfigured world (no world_actor_config row for any tier —
// mirrors 103_world_pressure_test.sql assertion (f), which proves fn_pressure_chance itself returns
// exactly 0, never NULL, for this exact shape) must NEVER fire, regardless of the roll — the only way
// to see that is chance itself being pinned at 0 so `roll < chance` is false for every possible roll
// in [0,1).
func TestRollTier_ChanceZeroNeverFires(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	orc := &Orchestrator{DB: pool}

	const unconfiguredWorldID = "99999999-9999-9999-9999-999999999999"
	fired, chance, roll, err := orc.rollTier(ctx, unconfiguredWorldID, "small", 1000, 0)
	if err != nil {
		t.Fatalf("rollTier: %v", err)
	}
	if chance != 0 {
		t.Fatalf("chance = %v, want exactly 0 for an unconfigured world", chance)
	}
	if roll < 0 || roll >= 1 {
		t.Fatalf("roll out of range: %v", roll)
	}
	if fired {
		t.Fatalf("fired = true with chance 0 (roll=%v) — a zero chance must never fire", roll)
	}
}

// TestRollTier_FiredMatchesRollLessThanChance is the non-vacuous invariant check: against the real
// seeded play world at tick 6000 (100 climb_chunks of the seeded small-tier config, climb_rate=0.01,
// climb_chunk_ticks=60 — 103_world_pressure_test.sql assertion (b4)), chance is capped at exactly
// 0.70 with no prior eruption (lastEruption=0, matching a fresh `make reset`'s empty world_eruption
// table for this world/tier). rollTier's fired decision must always equal roll < chance — this is
// the contract Task 10's trace and Task 9's caller both depend on.
func TestRollTier_FiredMatchesRollLessThanChance(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	orc := &Orchestrator{DB: pool}

	fired, chance, roll, err := orc.rollTier(ctx, dlWorldID, "small", 6000, 0)
	if err != nil {
		t.Fatalf("rollTier: %v", err)
	}
	if chance != 0.70 {
		t.Fatalf("chance = %v, want exactly 0.70 (100 climb_chunks, capped) — is world_eruption for this world/tier really empty? (run `make reset`)", chance)
	}
	if roll < 0 || roll >= 1 {
		t.Fatalf("roll out of range: %v", roll)
	}
	if want := roll < chance; fired != want {
		t.Fatalf("fired = %v, want roll(%v) < chance(%v) = %v", fired, roll, chance, want)
	}
}
