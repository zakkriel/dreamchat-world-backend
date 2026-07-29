package main

import (
	"context"
	"fmt"
	"math"
)

// tensionBudgetSeconds maps a scene's tension enum to the beat's time budget in seconds
// (FINAL-action-contracts.md §6 — "Tension (the beat budget)"). 1 tick = 1 second (binding), so
// the returned seconds ARE the tick budget the chain must fit cumulatively.
//
//	§6 enum table (values are data, retunable; the SHAPE is the rule):
//	  frantic = 5 s
//	  tense   = 30 s
//	  normal  = 60 s
//	  calm    = 600 s
//	  none    = ∞   (math.MaxInt64 sentinel — the deliberate atomic time-skip, §5/§6)
//
// The budget is a BLOCKER, never an award (§6): fitting means one specific impossibility — not
// enough time — didn't fire; it never turns "fits" into "happens." Consumption is cumulative across
// the chain (§6): each resolved move eats its duration; the next runs only if budget remains.
//
// Unknown → treated as `none` (∞) defensively. The scene's tension is trigger-validated against the
// five-value enum at write time (trg_validate_tension), so a non-enum value here is a data anomaly,
// not a routine input; defaulting to the unbounded budget keeps a corrupt read from spuriously
// halting a beat (the budget is measured at ask-time, never stored — measurements, not verdicts).
func tensionBudgetSeconds(tension string) int64 {
	switch tension {
	case "frantic":
		return 5
	case "tense":
		return 30
	case "normal":
		return 60
	case "calm":
		return 600
	case "none":
		return math.MaxInt64
	default:
		// Defensive: an out-of-enum value (should be impossible past trg_validate_tension) is the
		// unbounded budget, matching loadScene's "missing location_state → tension none" default.
		return math.MaxInt64
	}
}

// beatBudgetSeconds reads the scene's CURRENT tension at the acting actor's location and maps it to
// the beat's cumulative time budget (§6). This is the read the orchestrator does once at beat start.
// It mirrors loadScene's tension read (cognitionprompt.go) — a missing location_state row COALESCEs
// to 'none' (∞) — so a scene that was never stamped with a tension never spuriously budget-halts.
func (o *Orchestrator) beatBudgetSeconds(ctx context.Context, worldID, actorID string) (int64, error) {
	loc, err := o.actorLocation(ctx, worldID, actorID)
	if err != nil {
		return 0, fmt.Errorf("actor location (beat budget): %w", err)
	}
	var tension string
	if err := o.DB.QueryRow(ctx,
		`SELECT COALESCE((SELECT attrs->>'tension' FROM location_state WHERE entity_id=$1::uuid AND world_id=$2), 'none')`,
		loc, worldID).Scan(&tension); err != nil {
		return 0, fmt.Errorf("read scene tension: %w", err)
	}
	return tensionBudgetSeconds(tension), nil
}
