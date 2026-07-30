package main

import (
	"encoding/json"
	"fmt"
	"math"
)

// A mint is one typed row the resolve LLM proposes INSIDE an adjudicated ruling — the "vocabulary" half
// of FINAL-action-contracts.md §0 (the engine owns the grammar; the LLM mints the vocabulary the grammar
// runs over). validateMints enforces SHAPE + DERIVABLE BOUNDS only (§8) — NEVER plausibility ("is 400 m/s
// too fast?"). Units are fixed system-wide (meters, seconds, kilograms), so a value out of unit range is
// not checkable numerically; this validates STRUCTURE, not plausibility. Trust that a wild-but-well-formed
// mint is acceptable comes NOT from a numeric limit but from the three nets (§8):
//   1. a mint only happens inside a ruling that already passed the reality check;
//   2. blast radius = one logged row with provenance;
//   3. every mint is audit-trailed to the ruling that produced it.
//
// ── Mint-kind discriminator (by JSON shape/fields) ──────────────────────────────────────────────────
// The two movement-vocabulary shapes are fixed by §8; the artifact/place shape is derived from §3 (a
// newly minted place carries its coordinate-in-parent, validated to lie within the parent's extent) and
// §4 (size 1..10; a mundane container has max_room ≤ 4^(size-1)). The published ruling schema leaves
// `mints` an open object array (no envelope), so nothing pre-constrains the discriminator — it is FIXED
// and DOCUMENTED here:
//
//	has "baseSpeed"                              → MOVEMENT-TYPE mint  {movementTypeId, baseSpeed}
//	has "statusTypeId" | "movementModifiers"     → MODIFIER mint       {statusTypeId, actionType,
//	                                                                     movementModifiers:[{movementTypeId,
//	                                                                     modifierPercent}]}
//	has top-level "movementTypeId" (no baseSpeed)→ MOVEMENT-TYPE mint  (invalid — missing baseSpeed)
//	else has "size"|"maxRoom"|"coordinate"|      → ARTIFACT/PLACE mint (§3/§4)
//	         "parentLocationId"|"locationId"
//	none of the above                            → unknown shape → violation
//
// Envelope keys are camelCase, matching §8's worked examples (movementTypeId, baseSpeed, statusTypeId,
// actionType, movementModifiers, modifierPercent). The coordinate/extent shapes follow the Station-F
// plan: coordinate {x,y} meters in the parent's local frame; parentExtent {w,h} bounds. The parent's
// extent is carried INLINE on the artifact mint so validation stays DB-free (the signature takes no DB
// handle) — the resolve seat already holds the parent's extent in its facts slice.

// mintModRow is one row of a modifier mint's movementModifiers array (§8).
type mintModRow struct {
	MovementTypeID  *string  `json:"movementTypeId"`
	ModifierPercent *float64 `json:"modifierPercent"`
}

type mintCoord struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type mintExtent struct {
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// mintEnvelope is the union of every mint kind's fields; a nil pointer means "field absent", which is
// exactly what the discriminator and the presence-sensitive bound checks need.
type mintEnvelope struct {
	// movement-type
	MovementTypeID *string  `json:"movementTypeId"`
	BaseSpeed      *float64 `json:"baseSpeed"`
	// modifier
	StatusTypeID      *string      `json:"statusTypeId"`
	ActionType        *string      `json:"actionType"`
	MovementModifiers []mintModRow `json:"movementModifiers"`
	// artifact / place
	Size             *float64    `json:"size"`
	MaxRoom          *float64    `json:"maxRoom"`
	Coordinate       *mintCoord  `json:"coordinate"`
	ParentExtent     *mintExtent `json:"parentExtent"`
	LocationID       *string     `json:"locationId"`
	ParentLocationID *string     `json:"parentLocationId"`
}

// validateMints returns the SHAPE + BOUNDS violations across a ruling's mints slice (empty = pass).
// existingMovementTypes is the set of movement_type_ids already committed for the world (seeded + any
// minted by earlier rulings); mint-ordering (§8) additionally lets a modifier reference a movement type
// minted EARLIER in THIS same slice, so the running set grows as we walk the slice in order.
func validateMints(mints []json.RawMessage, existingMovementTypes map[string]bool) []string {
	var violations []string

	// Running set of usable movement types = seeded/committed ∪ minted-earlier-in-this-slice (§8 ordering).
	known := make(map[string]bool, len(existingMovementTypes))
	for k := range existingMovementTypes {
		known[k] = true
	}

	// Pre-pass: collect the artifact/place parent edges (locationId → parentLocationId) for cycle
	// detection. Cycles can span the slice in any order, so the map is built before the per-mint walk.
	parentOf := map[string]string{}
	for _, raw := range mints {
		var e mintEnvelope
		if json.Unmarshal(raw, &e) != nil {
			continue // decode errors surface in the main pass below
		}
		if mintKindOf(e) == "artifact" && e.LocationID != nil && *e.LocationID != "" &&
			e.ParentLocationID != nil && *e.ParentLocationID != "" {
			parentOf[*e.LocationID] = *e.ParentLocationID
		}
	}

	for i, raw := range mints {
		var e mintEnvelope
		if err := json.Unmarshal(raw, &e); err != nil {
			violations = append(violations, fmt.Sprintf("mint %d: not a valid mint object: %v", i, err))
			continue
		}

		switch mintKindOf(e) {
		case "movement":
			// base_speed a positive number; movementTypeId a non-empty string (§2 vocabulary shape).
			if e.MovementTypeID == nil || *e.MovementTypeID == "" {
				violations = append(violations, fmt.Sprintf("mint %d: movement type mint missing movementTypeId", i))
			}
			if e.BaseSpeed == nil {
				violations = append(violations, fmt.Sprintf("mint %d: movement type mint missing baseSpeed", i))
			} else if *e.BaseSpeed <= 0 {
				violations = append(violations, fmt.Sprintf("mint %d: baseSpeed must be > 0 (m/s), got %g", i, *e.BaseSpeed))
			}
			// Register the minted type so a LATER modifier in this slice may reference it (§8 ordering).
			if e.MovementTypeID != nil && *e.MovementTypeID != "" {
				known[*e.MovementTypeID] = true
			}

		case "modifier":
			if e.StatusTypeID == nil || *e.StatusTypeID == "" {
				violations = append(violations, fmt.Sprintf("mint %d: modifier mint missing statusTypeId", i))
			}
			// action_type is engine-constrained to 'move' (the only action the contract meters, §6).
			if e.ActionType != nil && *e.ActionType != "move" {
				violations = append(violations, fmt.Sprintf("mint %d: modifier actionType %q not 'move'", i, *e.ActionType))
			}
			if len(e.MovementModifiers) == 0 {
				violations = append(violations, fmt.Sprintf("mint %d: modifier mint has no movementModifiers", i))
			}
			for j, mm := range e.MovementModifiers {
				if mm.MovementTypeID == nil || *mm.MovementTypeID == "" {
					violations = append(violations, fmt.Sprintf("mint %d.mod %d: missing movementTypeId", i, j))
					continue
				}
				// Mint-ordering (§8): the referenced movement type must already exist (seeded/committed
				// or minted earlier in this slice). An unknown ref is a SHAPE failure → repair-then-bounce.
				if !known[*mm.MovementTypeID] {
					violations = append(violations, fmt.Sprintf(
						"mint %d.mod %d: references unknown movement type %q (mint-ordering: seed or mint it first)",
						i, j, *mm.MovementTypeID))
				}
				// Floor -100%, NO upper cap (§2). Below -100% is negative speed — meaningless.
				if mm.ModifierPercent == nil {
					violations = append(violations, fmt.Sprintf("mint %d.mod %d: missing modifierPercent", i, j))
				} else if *mm.ModifierPercent < -100 {
					violations = append(violations, fmt.Sprintf(
						"mint %d.mod %d: modifierPercent %g below the -100%% floor", i, j, *mm.ModifierPercent))
				}
			}

		case "artifact":
			violations = append(violations, validateArtifactMint(i, e, parentOf)...)

		default:
			violations = append(violations, fmt.Sprintf("mint %d: unrecognized mint shape (no movementTypeId/baseSpeed, statusTypeId, or size/coordinate/parentLocationId)", i))
		}
	}

	return violations
}

// mintKindOf discriminates a mint by its JSON shape/fields (see the file header). Priority order makes
// it deterministic even when a malformed mint carries fields of two kinds.
func mintKindOf(e mintEnvelope) string {
	switch {
	case e.BaseSpeed != nil:
		return "movement"
	case e.StatusTypeID != nil || len(e.MovementModifiers) > 0:
		return "modifier"
	case e.MovementTypeID != nil:
		return "movement" // top-level movementTypeId without baseSpeed → an (invalid) movement-type mint
	case e.Size != nil || e.MaxRoom != nil || e.Coordinate != nil || e.ParentLocationID != nil || e.LocationID != nil:
		return "artifact"
	default:
		return "unknown"
	}
}

// validateArtifactMint enforces the §3/§4 artifact/place bounds: size 1..10; a mundane container's
// max_room ≤ 4^(size-1); a coordinate within the parent's extent (when both are carried); and no
// parent_location_id cycle (self-parent or a cycle among co-minted locations — carried-forward guard I
// from the Task-1 review, the mint-write end; the SQL CTEs carry the defensive depth/cycle cap).
func validateArtifactMint(i int, e mintEnvelope, parentOf map[string]string) []string {
	var out []string

	// size: an integer in 1..10 (§4). Absent size defaults to 1 (the smallest), matching the volume
	// floor's convention (fn_volume on a missing size treats it as 1).
	size := 1.0
	if e.Size != nil {
		size = *e.Size
		if size != math.Trunc(size) || size < 1 || size > 10 {
			out = append(out, fmt.Sprintf("mint %d: size %g not an integer in 1..10", i, size))
		}
	}

	// container: max_room ≤ 4^(size-1) (§4 — a thing cannot hold more than its own volume). Uses the
	// clamped size (1..10) so the bound is well-defined even when size itself is out of range.
	if e.MaxRoom != nil {
		clamped := size
		if clamped < 1 {
			clamped = 1
		} else if clamped > 10 {
			clamped = 10
		}
		cap := math.Pow(4, clamped-1)
		if *e.MaxRoom > cap {
			out = append(out, fmt.Sprintf("mint %d: max_room %g exceeds 4^(size-1)=%g (mundane container bound)", i, *e.MaxRoom, cap))
		}
	}

	// coordinate within the parent's extent (§3). Validated only when BOTH are present (the extent is
	// carried inline so this stays DB-free). Origin is the parent frame's (0,0); the box is [0,w]×[0,h].
	if e.Coordinate != nil && e.ParentExtent != nil {
		c, x := e.Coordinate, e.ParentExtent
		if c.X < 0 || c.Y < 0 || c.X > x.W || c.Y > x.H {
			out = append(out, fmt.Sprintf(
				"mint %d: coordinate {%g,%g} outside parent extent {w:%g,h:%g}", i, c.X, c.Y, x.W, x.H))
		}
	}

	// cycle guard (I-a): walk the proposed parent chain THROUGH the co-minted slice; if it revisits the
	// starting location (self-parent, or A→B→A), that is a cycle → violation. A parent that points OUT of
	// the slice terminates the walk safely — the committed hierarchy is kept acyclic by the SQL guard.
	if e.LocationID != nil && *e.LocationID != "" {
		start := *e.LocationID
		seen := map[string]bool{start: true}
		cur := start
		for steps := 0; steps <= len(parentOf); steps++ {
			next, ok := parentOf[cur]
			if !ok {
				break // reached a location whose parent is not in this slice → no cycle on this path
			}
			if next == start || seen[next] {
				out = append(out, fmt.Sprintf(
					"mint %d: parent_location_id chain from %s forms a cycle (reaches %s)", i, start, next))
				break
			}
			seen[next] = true
			cur = next
		}
	}

	return out
}
