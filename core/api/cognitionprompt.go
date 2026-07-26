package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Station E cognition payloads (RULINGS-2026-07-23 §5). One cognition call PER ACTION; each
// present NPC sits in exactly ONE seat. The prompt is laid out cache-native: a STABLE PREFIX
// (header → scene → the minds' cores) then, for the isolated seat only, the secret block, then
// the APPEND-ONLY public moment (oldest first), then the MUTABLE TAIL (the imminent wind-up).
// Providers cache the repeated prefix; anything that changes per call rides the tail. Section
// ORDER is the contract — the unit tests pin it.

// publicMomentRecentK caps the shared public moment to the last K events every present holder
// perceived (fn_public_moment's p_k). A v1 dial; the retrieval-assembly refinement is Station I.
const publicMomentRecentK = 20

// npcMind is a mind the cognition seat speaks for: the roster name, plus the authored personality
// core when one exists. A seed actor may have NO personality_core row yet (the seed lags the
// engine); that is a NAME-ONLY mind — Traits nil, Malleability 0 — and must never fail the beat.
type npcMind struct {
	ID           string
	Name         string
	Traits       json.RawMessage
	Malleability float64
}

// rosterEntry is a present actor as a public fact: name + id. The full roster is public — who is
// in the room is not a secret — so it appears in every seat's SCENE section.
type rosterEntry struct {
	ID   string
	Name string
}

// sceneInfo is the public frame both seats share: where, how tense, and who is present.
type sceneInfo struct {
	LocationName string
	Tension      string
	Present      []rosterEntry
}

// momentLine is one line of the shared public moment (fn_public_moment: modal content + tick).
type momentLine struct {
	Content string
	Tick    int64
}

// privateLine is one of the isolated NPC's own private records (fn_private_records) — carried
// ONLY on her own call, never in a shared prompt.
type privateLine struct {
	Content string
	Tick    int64
}

// cognitionSystemHeader is the stable cache prefix and the seat's standing instruction. It states
// plainly: you speak ONLY for the minds listed under DECIDE FOR; one decision each
// (none | commit | telegraph); telegraph is the exception, not the rhythm (RULINGS-2026-07-24 §1);
// attempts use the six canon types with ids from THIS prompt only; you never invent entities.
//
// Text lives in prompts/cognition.txt (core/api/prompts/README.md) — every fixed prompt rulebook
// readable in one place, config-style, mirroring the schema/*.json + go:embed pattern. Shared by
// both the batch and isolated seats (buildBatchPrompt / buildIsolatedPrompt below).
//
//go:embed prompts/cognition.txt
var cognitionSystemHeader string

// buildBatchPrompt renders the SHARED batch payload — one call for every NPC whose read of the
// moment needs nothing beyond what everyone perceived.
//
// WALL INVARIANT (RULINGS-2026-07-23 §5), enforced HERE by construction: this payload contains
// NOTHING an isolated lookup flagged. It receives ONLY the batch minds' cores and the PUBLIC
// moment — no private lines, and no isolated NPC's core beyond the public roster name line in
// SCENE. Secrets ride only the flagged NPC's own isolated call (buildIsolatedPrompt). A secret
// that never enters a shared prompt cannot bleed into another mind's reasoning — you cannot
// validate a leak away, you can only not create it.
func buildBatchPrompt(scene sceneInfo, minds []npcMind, moment []momentLine, imminentActor string, imminent Attempt) string {
	return buildCognitionPrompt(scene, minds, nil, false, moment, imminentActor, imminent)
}

// buildIsolatedPrompt renders one flagged NPC's ISOLATED payload: the same public frame plus her
// OWN private records (the (3b) block), so her secret rides alone.
func buildIsolatedPrompt(scene sceneInfo, mind npcMind, private []privateLine, moment []momentLine, imminentActor string, imminent Attempt) string {
	return buildCognitionPrompt(scene, []npcMind{mind}, private, true, moment, imminentActor, imminent)
}

// buildCognitionPrompt is the shared layout. isolated=true inserts the (3b) private block between
// the minds and the public moment; the batch seat passes isolated=false and never carries it.
func buildCognitionPrompt(scene sceneInfo, minds []npcMind, private []privateLine, isolated bool, moment []momentLine, imminentActor string, imminent Attempt) string {
	var sb strings.Builder
	sb.WriteString(cognitionSystemHeader)

	// (2) SCENE — public facts: where, tension, and the full present roster (name + id).
	sb.WriteString("\n\nSCENE\n")
	sb.WriteString("Location: ")
	sb.WriteString(scene.LocationName)
	sb.WriteString("\nTension: ")
	sb.WriteString(scene.Tension)
	sb.WriteString("\nPresent:\n")
	for _, r := range scene.Present {
		sb.WriteString("- ")
		sb.WriteString(r.Name)
		sb.WriteString(" (")
		sb.WriteString(r.ID)
		sb.WriteString(")\n")
	}

	// (3) THE MINDS YOU SPEAK FOR — cores only for the decided-for minds. A name-only mind (no
	// personality_core row) renders its name and nothing more.
	sb.WriteString("\nTHE MINDS YOU SPEAK FOR\n")
	for _, m := range minds {
		sb.WriteString("- ")
		sb.WriteString(m.Name)
		sb.WriteString(" (")
		sb.WriteString(m.ID)
		sb.WriteString(")")
		if len(m.Traits) > 0 {
			sb.WriteString(" — traits: ")
			sb.Write(m.Traits)
			fmt.Fprintf(&sb, " | malleability: %.2f", m.Malleability)
		} else {
			sb.WriteString(" (no personality core yet)")
		}
		sb.WriteString("\n")
	}

	// (3b) WHAT ONLY YOU KNOW — isolated seat ONLY. This block is what the wall keeps out of every
	// shared prompt (see buildBatchPrompt's invariant note).
	if isolated {
		sb.WriteString("\nWHAT ONLY YOU KNOW (private, yours alone):\n")
		for _, pl := range private {
			fmt.Fprintf(&sb, "- [tick %d] %s\n", pl.Tick, pl.Content)
		}
	}

	// (4) PUBLIC MOMENT — the modal face of every event shared by all present holders, oldest
	// first. Append-only canon makes this cache-native (it only grows at the end).
	sb.WriteString("\nPUBLIC MOMENT (oldest first)\n")
	for _, ml := range moment {
		fmt.Fprintf(&sb, "- [tick %d] %s\n", ml.Tick, ml.Content)
	}

	// (5) MUTABLE TAIL — the imminent wind-up. This is the only per-call-varying section; it sits
	// at the tail so the whole prefix above stays cacheable.
	sb.WriteString("\nIMMINENT: ")
	sb.WriteString(imminentActor)
	sb.WriteString(" is about to: ")
	sb.WriteString(imminent.Stated)
	attJSON, _ := json.Marshal(imminent)
	sb.WriteString("\nATTEMPT: ")
	sb.Write(attJSON)
	sb.WriteString("\nDECIDE FOR: [")
	for i, m := range minds {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(m.ID)
	}
	sb.WriteString("]")
	return sb.String()
}

// loadMinds reads each id's roster name (entity_registry) LEFT JOINed to its personality_core.
// An NPC with no core row becomes a NAME-ONLY mind (Traits nil, Malleability 0) — the seed may
// lag the engine, and a missing core must never fail the beat. Ordered by id for a stable,
// cache-native DECIDE FOR list.
func (o *Orchestrator) loadMinds(ctx context.Context, worldID string, ids []string) ([]npcMind, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := o.DB.Query(ctx, `
		SELECT er.entity_id::text, er.canonical_name, pc.traits, pc.malleability::float8
		FROM entity_registry er
		LEFT JOIN personality_core pc ON pc.actor_id = er.entity_id AND pc.world_id = er.world_id
		WHERE er.world_id = $1 AND er.entity_id = ANY($2::uuid[])
		ORDER BY er.entity_id`, worldID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var minds []npcMind
	for rows.Next() {
		var m npcMind
		var traits []byte
		var mall *float64
		if err := rows.Scan(&m.ID, &m.Name, &traits, &mall); err != nil {
			return nil, err
		}
		if len(traits) > 0 {
			m.Traits = json.RawMessage(traits)
		}
		if mall != nil {
			m.Malleability = *mall
		}
		minds = append(minds, m)
	}
	return minds, rows.Err()
}

// displayLabels returns each entity's DISPLAY NAME as ONE viewer knows it (§3 naming reach): known
// name else descriptor else canonical (fn_display_name). Used to relabel an isolated NPC's scene +
// imminent line so she reads the room as SHE knows it — ids unchanged, only the labels.
func (o *Orchestrator) displayLabels(ctx context.Context, worldID, viewerID string, ids []string) (map[string]string, error) {
	labels := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return labels, nil
	}
	rows, err := o.DB.Query(ctx,
		`SELECT e::text, fn_display_name($1::uuid, $2::uuid, e) FROM unnest($3::uuid[]) e`,
		worldID, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, err
		}
		labels[id] = label
	}
	return labels, rows.Err()
}

// batchDisplayLabels returns each entity's DISPLAY NAME for the SHARED batch (§3/§5): a name only if
// EVERY batch mind resolves the SAME known name (fn_batch_display_name's shared-by-all intersection),
// else the descriptor, else canonical. Relabels the batch scene + imminent line so no name reaches the
// shared prompt past a mind that does not know it — the same philosophy as the wall (mechanical, no
// judgment): the safe failure shows a descriptor, never a name a mind lacks.
func (o *Orchestrator) batchDisplayLabels(ctx context.Context, worldID string, minds, ids []string) (map[string]string, error) {
	labels := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return labels, nil
	}
	rows, err := o.DB.Query(ctx,
		`SELECT e::text, fn_batch_display_name($1::uuid, $2::uuid[], e) FROM unnest($3::uuid[]) e`,
		worldID, minds, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, err
		}
		labels[id] = label
	}
	return labels, rows.Err()
}

// relabelScene returns a copy of scene with each present roster name replaced by the caller's display
// label for that id (unchanged when a label is missing/empty). ids stay real; only the labels change.
func relabelScene(scene sceneInfo, labels map[string]string) sceneInfo {
	out := scene
	out.Present = make([]rosterEntry, len(scene.Present))
	for i, r := range scene.Present {
		if lbl, ok := labels[r.ID]; ok && lbl != "" {
			r.Name = lbl
		}
		out.Present[i] = r
	}
	return out
}

// imminentLabel is the acting actor's label for one seat: the seat's display name for the player, or
// the player id when unresolved (the prior fallback).
func imminentLabel(labels map[string]string, playerID string) string {
	if lbl, ok := labels[playerID]; ok && lbl != "" {
		return lbl
	}
	return playerID
}

// loadScene builds the public frame: the location's name + tension, and the present roster
// (name + id), ordered by id for determinism. Missing location_state → tension "none".
func (o *Orchestrator) loadScene(ctx context.Context, worldID, locationID string, present []string) (sceneInfo, error) {
	var scene sceneInfo
	if err := o.DB.QueryRow(ctx, `
		SELECT
		  COALESCE((SELECT canonical_name FROM entity_registry WHERE entity_id=$1::uuid AND world_id=$2), 'somewhere'),
		  COALESCE((SELECT attrs->>'tension' FROM location_state WHERE entity_id=$1::uuid AND world_id=$2), 'none')`,
		locationID, worldID).Scan(&scene.LocationName, &scene.Tension); err != nil {
		return scene, err
	}
	if len(present) == 0 {
		return scene, nil
	}
	rows, err := o.DB.Query(ctx, `
		SELECT entity_id::text, canonical_name FROM entity_registry
		WHERE world_id=$1 AND entity_id = ANY($2::uuid[]) ORDER BY entity_id`, worldID, present)
	if err != nil {
		return scene, err
	}
	defer rows.Close()
	for rows.Next() {
		var r rosterEntry
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return scene, err
		}
		scene.Present = append(scene.Present, r)
	}
	return scene, rows.Err()
}

// publicMoment reads fn_public_moment — the modal face of every event shared by ALL present
// holders, oldest first (already ordered by the function). The batch and isolated seats share it.
func (o *Orchestrator) publicMoment(ctx context.Context, worldID string, present []string) ([]momentLine, error) {
	rows, err := o.DB.Query(ctx,
		`SELECT content, acquired_tick FROM fn_public_moment($1, $2::uuid[], $3)`,
		worldID, present, publicMomentRecentK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []momentLine
	for rows.Next() {
		var ml momentLine
		if err := rows.Scan(&ml.Content, &ml.Tick); err != nil {
			return nil, err
		}
		lines = append(lines, ml)
	}
	return lines, rows.Err()
}

// isolatedNPCs reads fn_isolated_npcs: the NPCs whose PRIVATE about-ness intersects the action's
// bound ids (one hop). These are pulled out of the batch and get their own call — a mechanical
// lookup, not a judgment (§5).
func (o *Orchestrator) isolatedNPCs(ctx context.Context, worldID string, actionIDs, present, npcs []string) ([]string, error) {
	rows, err := o.DB.Query(ctx,
		`SELECT actor_id::text FROM fn_isolated_npcs($1, $2::uuid[], $3::uuid[], $4::uuid[])`,
		worldID, actionIDs, present, npcs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// privateRecords reads fn_private_records — one flagged NPC's own private lines whose subjects
// intersect the action ids, freshest 20, presented oldest-first. Carried ONLY on her own call.
func (o *Orchestrator) privateRecords(ctx context.Context, worldID, npcID string, actionIDs, present []string) ([]privateLine, error) {
	rows, err := o.DB.Query(ctx,
		`SELECT content, acquired_tick FROM fn_private_records($1, $2::uuid, $3::uuid[], $4::uuid[])`,
		worldID, npcID, actionIDs, present)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []privateLine
	for rows.Next() {
		var pl privateLine
		if err := rows.Scan(&pl.Content, &pl.Tick); err != nil {
			return nil, err
		}
		lines = append(lines, pl)
	}
	return lines, rows.Err()
}
