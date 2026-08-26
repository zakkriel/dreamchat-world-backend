package main

// Governed-by: D-1 — nothing mutates canon directly — proposals only, the Core commits.
// Promoted from this file's own citations (2026-08-26), not newly decided. Change what this
// file decides and those decisions change with it (D-9).

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Rung 2 (the Journey ladder) / Task 8 — "the world builds the road it needs" (design §4.6, R2/R4).
//
// Nothing is built while a traveller walks. Only when the world's turn (runJourneyLeg's own call to
// runWorldTurn, UNCHANGED) has just decided to ACT — and journeyScene has already found that no known
// place contains the point reached — does this file's machinery run. The order is the founder's own,
// verbatim: does something happen? -> are you somewhere known? -> if not, create it -> then the event
// already authored (by runWorldTurn's own World Actor call, at the anchor journeyScene fell back to)
// gets somewhere real to have happened NEAR, and the traveller a real place to now be standing.
//
// What gets created LASTS (R2) and comes WITH its connections (R4): a place_author seat authors
// IDENTITY ONLY — descriptor, a narrative kind, and a size CLASS from a closed enum
// (place_author.v1.schema.json carries no coordinate, no radius, no number of any kind — the schema
// is the leash). The ENGINE alone draws the footprint (fn_extent_class_metres + fn_area_around, rung
// 1) and commits the place through EntityCreated on the NORMAL ruled path (apply_ruled_event ->
// fn_apply_entity_created, descriptor-mandatory + reuse-before-create already enforced there — no
// bypass, D-1). Connections are Portal artifacts, created ONLY where no connection already exists
// between that exact pair (R4 "fills gaps only") — and an existing SHUT or LOCKED connection between
// the traveller's anchor and the journey's own goal is obeyed, never routed around: the journey ends
// barred instead of minting a bypass.

//go:embed prompts/place_author.txt
var placeAuthorSystemHeader string

//go:embed schema/place_author.v1.schema.json
var placeAuthorSchemaJSON string

// authoredPlace is the place_author seat's whole output — the ONLY three things it may say (the
// schema's own closed property set). No coordinate, no radius, no number of any kind: geometry never
// leaves the engine.
type authoredPlace struct {
	Descriptor  string `json:"descriptor"`
	Kind        string `json:"kind"`
	ExtentClass string `json:"extent_class"`
}

// nearbyKnownPlace is one entry of the deterministic context handed to the seat — a known location
// already charted near the point, so the seat can let what already exists imply what plausibly exists
// here too (a stretch between two great cities reads differently than one between two hamlets).
type nearbyKnownPlace struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// authorPlaceForLeg is runJourneyLeg's own hand-off, called EXACTLY when the world's turn just fired
// (firedMag != "") and journeyScene found the traveller standing nowhere a known place contains — the
// R2 gate. fromID is the anchor journeyScene fell back to (the last known stage, or the actor's own
// departure location on the very first such leg) — where the traveller is coming FROM. point is the
// interpolated coordinate (already in j.FrameID's frame) the new place must contain.
//
// R4 first: before minting anything, this checks whether fromID and the journey's ultimate goal
// (j.GoalTarget) are ALREADY directly connected by a Portal. If one exists and does NOT permit passage
// (shut or locked), the road is deliberately barred — barred=true tells the caller to end the journey
// outright, surfaced honestly rather than bypassed with a fresh route around it. Otherwise (no
// connection yet, the ordinary open-road case — or one that already happens to permit passage) this
// proceeds to mint:
//
//  1. the place — descriptor/kind/extent_class from the seat, footprint from the engine, committed
//     through EntityCreated on the normal ruled path.
//  2. its ways — an open Portal fromID<->newPlace and an open Portal newPlace<->j.GoalTarget, each
//     created ONLY where no connection already exists for that exact pair (ensurePortal's own R4
//     gap-fill, applied per-edge too).
//  3. the traveller's own arrival there — an ordinary ActorMoved onto the new place, legal now that
//     step 2 opened the way in, so j.StageID and actor_state agree on where the traveller stands.
//
// seq starts at seqStart (the leg's own runWorldTurn already consumed slots up to it there) and every
// commit here advances a local copy, so no two commits in this leg ever land on the same (tick,seq).
func (o *Orchestrator) authorPlaceForLeg(ctx context.Context, j *Journey, fromID string, point json.RawMessage, tick int64, seqStart int, outcome *BeatOutcome, trace *BeatTrace) (barred bool, err error) {
	seq := seqStart

	exists, permits, err := o.connectionBetween(ctx, j.WorldID, fromID, j.GoalTarget)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: connection check: %w", err)
	}
	if exists && !permits {
		// R4: an existing shut or locked connection is obeyed, never routed around.
		return true, nil
	}

	descriptor, kind, extentClass, err := o.authorPlaceIdentity(ctx, j.WorldID, j.FrameID, point)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: seat: %w", err)
	}

	radius, err := o.fnExtentClassMetres(ctx, j.WorldID, extentClass)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: extent class: %w", err)
	}
	area, err := o.fnAreaAround(ctx, point, radius)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: area: %w", err)
	}

	newAttrs, err := json.Marshal(map[string]any{
		"parent_location_id": j.FrameID,
		"coordinates":        json.RawMessage(point),
		"area":               json.RawMessage(area),
		"kind":               kind,
	})
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: attrs: %w", err)
	}

	placeID, err := o.newEntityID(ctx)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: new id: %w", err)
	}

	createEvt := RuledEventV2{
		Type:          "EntityCreated",
		ActorID:       j.ActorID,
		Truth:         "the road opens onto " + descriptor,
		TargetID:      placeID,
		NewEntityKind: "location",
		Descriptor:    descriptor,
		CanonicalName: descriptor,
		NewAttrs:      newAttrs,
	}
	result, err := o.applyRuledEvent(ctx, j.WorldID, createEvt, tick, seq)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: commit place: %w", err)
	}
	seq++
	if halt, _ := result["halt_reason"].(string); halt == "gate_reject" || result["event_id"] == nil {
		return false, fmt.Errorf("authorPlaceForLeg: place creation gate-rejected")
	}
	if outcome != nil {
		if id, ok := result["event_id"].(string); ok {
			outcome.Committed = append(outcome.Committed, id)
		}
	}

	// fn_apply_entity_created's own reuse-before-create can silently hand back an EXISTING id whose
	// descriptor already matched (case/whitespace-insensitive) instead of minting placeID — that
	// return value is PERFORMed away inside apply_ruled_event, so it never reaches this Go layer.
	// Re-resolve the SAME match here rather than trust placeID blindly: two different stretches of
	// unmapped road can legitimately earn the identical descriptor from the seat (a fixed/templated
	// driver, or simply the same honest phrase twice), and reuse-before-create is a floor this program
	// must obey too, not bypass by minting a second row under a self-generated id nobody else agrees is
	// canonical.
	resolvedPlaceID, err := o.resolveEntityByDescriptor(ctx, j.WorldID, "location", descriptor)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: resolve created place: %w", err)
	}
	placeID = resolvedPlaceID

	// The way BEHIND you is not optional. This used to be gated on `!exists` — "only bridge from→place
	// if there was no direct from→goal connection" — which conflated two different questions:
	//
	//   * "is there already a way from here to the goal?"  → R4's bar check, answered above, which
	//     returns `barred` for a shut or locked one and never routes around it.
	//   * "can the traveller stand on the stretch of road they just walked onto?" → ALWAYS yes. They
	//     are already there; the place is minted BECAUSE they are there.
	//
	// With the guard, a journey whose destination happened to be directly connected to its origin
	// minted the waystation, wired it only to the GOAL, and then failed to move the traveller onto it
	// — gate_reject, a hard error that killed the beat. Latent until the Harbormaster's Office was
	// seeded behind an open door from Dock Street, which made `exists` true for the first time and
	// turned every eruption on that road into a dead beat. It is also why no waystation ever survived
	// a leg, and therefore why journey.where_label was permanently null.
	seq, err = o.ensurePortal(ctx, j.WorldID, j.ActorID, fromID, placeID, tick, seq, outcome)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: portal (from): %w", err)
	}
	seq, err = o.ensurePortal(ctx, j.WorldID, j.ActorID, placeID, j.GoalTarget, tick, seq, outcome)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: portal (to): %w", err)
	}

	mover := Attempt{Type: "ActorMoved", Stated: "the road brings me to " + descriptor, ToTargetID: placeID}
	_, _, halt, err := o.commitWorldPayload(ctx, j.WorldID, j.ActorID, mover, tick, seq, nil, outcome, trace)
	if err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: move onto place: %w", err)
	}
	if halt != "" {
		return false, fmt.Errorf("authorPlaceForLeg: move onto new place halted: %s", halt)
	}

	j.StageID = placeID
	if _, err := o.DB.Exec(ctx,
		`UPDATE journey SET stage_id=$1::uuid WHERE journey_id=$2::uuid`, placeID, j.ID); err != nil {
		return false, fmt.Errorf("authorPlaceForLeg: persist stage: %w", err)
	}
	return false, nil
}

// connectionBetween reports whether a Portal artifact already connects aID<->bID (regardless of
// open/locked), and if so whether it currently permits passage (open AND NOT locked — the same rule
// fn_portal_permits enforces in SQL; mirrored here in Go because R4's gap-fill needs to know
// EXISTENCE separately from PERMISSION — fn_portal_permits alone only ever answers the second).
func (o *Orchestrator) connectionBetween(ctx context.Context, worldID, aID, bID string) (exists, permits bool, err error) {
	if aID == "" || bID == "" {
		return false, false, nil
	}
	var open, locked bool
	err = o.DB.QueryRow(ctx, `
		SELECT COALESCE(attrs->>'open','false')='true', COALESCE(attrs->>'locked','true')='true'
		FROM artifact_state
		WHERE world_id=$1::uuid AND attrs->'connects' ? $2 AND attrs->'connects' ? $3
		LIMIT 1`, worldID, aID, bID).Scan(&open, &locked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, open && !locked, nil
}

// ensurePortal creates an OPEN Portal connecting aID<->bID — but ONLY where no connection already
// exists for that exact pair (R4 "fills gaps only", applied per-edge: a leg that already opened this
// exact way a moment ago must never mint a second one beside it). Returns the next free seq slot
// (unchanged when nothing was minted).
func (o *Orchestrator) ensurePortal(ctx context.Context, worldID, actorID, aID, bID string, tick int64, seq int, outcome *BeatOutcome) (int, error) {
	exists, _, err := o.connectionBetween(ctx, worldID, aID, bID)
	if err != nil {
		return seq, err
	}
	if exists {
		return seq, nil
	}

	nameA, err := o.entityLabel(ctx, worldID, aID)
	if err != nil {
		return seq, fmt.Errorf("ensurePortal: label a: %w", err)
	}
	nameB, err := o.entityLabel(ctx, worldID, bID)
	if err != nil {
		return seq, fmt.Errorf("ensurePortal: label b: %w", err)
	}

	attrs, err := json.Marshal(map[string]any{
		"open":     true,
		"locked":   false,
		"connects": []string{aID, bID},
	})
	if err != nil {
		return seq, err
	}
	portalID, err := o.newEntityID(ctx)
	if err != nil {
		return seq, err
	}
	descriptor := "the way connecting " + nameA + " and " + nameB

	evt := RuledEventV2{
		Type:          "EntityCreated",
		ActorID:       actorID,
		Truth:         "a way opens between " + nameA + " and " + nameB,
		TargetID:      portalID,
		NewEntityKind: "artifact",
		Descriptor:    descriptor,
		CanonicalName: descriptor,
		NewAttrs:      attrs,
	}
	result, err := o.applyRuledEvent(ctx, worldID, evt, tick, seq)
	if err != nil {
		return seq, err
	}
	if halt, _ := result["halt_reason"].(string); halt == "gate_reject" || result["event_id"] == nil {
		return seq, fmt.Errorf("ensurePortal: portal creation gate-rejected")
	}
	if outcome != nil {
		if id, ok := result["event_id"].(string); ok {
			outcome.Committed = append(outcome.Committed, id)
		}
	}
	return seq + 1, nil
}

// entityLabel reads a display-worthy label for id — canonical_name, falling back to descriptor, then
// the raw id — used only to write a readable Portal descriptor; never a perception-facing name (that
// wall is fn_display_name's, untouched here).
func (o *Orchestrator) entityLabel(ctx context.Context, worldID, id string) (string, error) {
	var name string
	if err := o.DB.QueryRow(ctx,
		`SELECT COALESCE(NULLIF(canonical_name,''), NULLIF(descriptor,''), entity_id::text)
		 FROM entity_registry WHERE world_id=$1::uuid AND entity_id=$2::uuid`,
		worldID, id).Scan(&name); err != nil {
		return "", err
	}
	return name, nil
}

// resolveEntityByDescriptor mirrors fn_apply_entity_created's OWN reuse-before-create match exactly
// (same normalization, same tie-break) — the ONLY way to learn which id an EntityCreated commit
// actually landed on, since apply_ruled_event's SQL discards fn_apply_entity_created's return value.
func (o *Orchestrator) resolveEntityByDescriptor(ctx context.Context, worldID, kind, descriptor string) (string, error) {
	var id string
	if err := o.DB.QueryRow(ctx, `
		SELECT entity_id FROM entity_registry
		WHERE world_id=$1::uuid AND entity_kind=$2 AND status='active'
		  AND descriptor IS NOT NULL AND lower(btrim(descriptor))=lower(btrim($3))
		ORDER BY entity_id LIMIT 1`, worldID, kind, descriptor).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// newEntityID mints a fresh id up front — authorPlaceForLeg/ensurePortal need the new entity's id
// BEFORE committing (to wire it into a Portal's connects array and the journey's stage_id), and
// apply_ruled_event's own EntityCreated branch only ever returns {event_id, halt_reason} — the new
// entity's id never round-trips back through it (fn_apply_entity_created's return value is PERFORMed
// away). Reusing the DB's own gen_random_uuid() keeps the id space identical to every id the schema
// itself would have minted.
func (o *Orchestrator) newEntityID(ctx context.Context) (string, error) {
	var id string
	if err := o.DB.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// fnExtentClassMetres calls fn_extent_class_metres(world, class) — the engine's own size-class ->
// radius lookup (rung 1). The seat never sees the number this returns.
func (o *Orchestrator) fnExtentClassMetres(ctx context.Context, worldID, class string) (float64, error) {
	var r float64
	err := o.DB.QueryRow(ctx, `SELECT fn_extent_class_metres($1::uuid,$2)`, worldID, class).Scan(&r)
	return r, err
}

// fnAreaAround calls fn_area_around(centre, radius) — the engine's own footprint-drawing function
// (rung 1). This is the ONLY place a created place's attrs.area is computed; the seat supplies neither
// input.
func (o *Orchestrator) fnAreaAround(ctx context.Context, centre json.RawMessage, radius float64) (json.RawMessage, error) {
	var area []byte
	err := o.DB.QueryRow(ctx, `SELECT fn_area_around($1::jsonb,$2)`, string(centre), radius).Scan(&area)
	return json.RawMessage(area), err
}

// authorPlaceIdentity gathers the deterministic context R2 names (the point, the parent region, the
// nearest known places) and hands it to the place_author seat, which authors ONLY identity — a
// descriptor, a narrative kind, and a size CLASS from the closed enum. No coordinate, no radius, no
// number of any kind ever leaves the seat (place_author.v1.schema.json is the leash) — the ENGINE
// draws the footprint afterward (fnExtentClassMetres + fnAreaAround), never the model.
func (o *Orchestrator) authorPlaceIdentity(ctx context.Context, worldID, frameID string, point json.RawMessage) (descriptor, kind, extentClass string, err error) {
	prompt, err := o.buildPlaceAuthorPrompt(ctx, worldID, frameID, point)
	if err != nil {
		return "", "", "", err
	}
	raw, genErr := o.PlaceAuthor.Generate(ctx, GenRequest{Prompt: prompt, Schema: json.RawMessage(placeAuthorSchemaJSON)})
	if genErr != nil {
		return "", "", "", fmt.Errorf("authorPlaceIdentity: Generate: %w", genErr)
	}
	var authored authoredPlace
	if err := json.Unmarshal([]byte(raw), &authored); err != nil {
		return "", "", "", fmt.Errorf("authorPlaceIdentity: decode: %w", err)
	}
	if strings.TrimSpace(authored.Descriptor) == "" {
		return "", "", "", fmt.Errorf("authorPlaceIdentity: seat returned an empty descriptor")
	}
	if strings.TrimSpace(authored.Kind) == "" {
		return "", "", "", fmt.Errorf("authorPlaceIdentity: seat returned an empty kind")
	}
	switch authored.ExtentClass {
	case "intimate", "small", "medium", "large", "vast":
	default:
		return "", "", "", fmt.Errorf("authorPlaceIdentity: extent_class %q outside the closed enum", authored.ExtentClass)
	}
	return authored.Descriptor, authored.Kind, authored.ExtentClass, nil
}

// buildPlaceAuthorPrompt renders the place_author prompt: the stable header (place_author.txt) then a
// deterministic CONTEXT block — the point, the parent region's own name, and up to 5 nearest known
// children of that region (by straight-line distance to point) — assembled entirely in Go from plain
// reads, no model involved (design §4.6 step 1: "Engine assembles (deterministic, no model)").
func (o *Orchestrator) buildPlaceAuthorPrompt(ctx context.Context, worldID, frameID string, point json.RawMessage) (string, error) {
	var pt struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := json.Unmarshal(point, &pt); err != nil {
		return "", fmt.Errorf("buildPlaceAuthorPrompt: point: %w", err)
	}

	var regionName string
	if err := o.DB.QueryRow(ctx,
		`SELECT COALESCE(canonical_name,'') FROM entity_registry WHERE world_id=$1::uuid AND entity_id=$2::uuid`,
		worldID, frameID).Scan(&regionName); err != nil {
		return "", fmt.Errorf("buildPlaceAuthorPrompt: region: %w", err)
	}

	rows, err := o.DB.Query(ctx, `
		SELECT er.entity_id, COALESCE(er.canonical_name,''),
		       COALESCE((ls.attrs->'coordinates'->>'x')::float8,0), COALESCE((ls.attrs->'coordinates'->>'y')::float8,0)
		FROM location_state ls
		JOIN entity_registry er ON er.entity_id = ls.entity_id AND er.world_id = ls.world_id
		WHERE ls.world_id=$1::uuid AND (ls.attrs->>'parent_location_id')::uuid=$2::uuid
		ORDER BY ((COALESCE((ls.attrs->'coordinates'->>'x')::float8,0)-$3)^2
		        + (COALESCE((ls.attrs->'coordinates'->>'y')::float8,0)-$4)^2) ASC
		LIMIT 5`, worldID, frameID, pt.X, pt.Y)
	if err != nil {
		return "", fmt.Errorf("buildPlaceAuthorPrompt: nearby: %w", err)
	}
	defer rows.Close()
	var nearby []nearbyKnownPlace
	for rows.Next() {
		var np nearbyKnownPlace
		if err := rows.Scan(&np.ID, &np.Name, &np.X, &np.Y); err != nil {
			return "", fmt.Errorf("buildPlaceAuthorPrompt: scan nearby: %w", err)
		}
		nearby = append(nearby, np)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("buildPlaceAuthorPrompt: nearby rows: %w", err)
	}

	ctxJSON, err := json.Marshal(map[string]any{
		"point":               map[string]float64{"x": pt.X, "y": pt.Y},
		"region":              map[string]string{"id": frameID, "name": regionName},
		"nearby_known_places": nearby,
	})
	if err != nil {
		return "", fmt.Errorf("buildPlaceAuthorPrompt: marshal: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(placeAuthorSystemHeader)
	sb.WriteString("\n\nCONTEXT (the point on the road, its parent region, and the nearest known places):\n")
	sb.Write(ctxJSON)
	return sb.String(), nil
}
