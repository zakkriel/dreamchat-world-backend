package main

import (
	_ "embed"
	"encoding/json"
	"strings"
)

// Living World / Task 8 (Unit 5) — the World Actor's prompt assembly. Mirrors buildResolvePrompt's
// section layout (resolveprompt.go): a STABLE HEADER (the standing rules, go:embedded from
// prompts/world_actor.txt — mirrors the schema/*.json + go:embed pattern every other seat uses), then
// the gathered payload, then the per-call mutable tail. Unlike every character-mind seat, the World
// Actor is TRUTH-side / world-omniscient (design doc Unit 5): the payload it reads is fn_world_slice's
// raw jsonb, embedded verbatim — never perception-scoped, never run through fn_display_name.
//
//go:embed prompts/world_actor.txt
var worldActorSystemHeader string

// worldActorSchemaJSON is the leash: the World Actor's output must decode to exactly one
// {"actor_id":..., "attempt":{...}} — the SAME shape Task 4's pendingPayload already establishes
// (ledger.go), reused verbatim so a scheduled pending_event row and a freshly authored intrusion are
// indistinguishable in shape.
//
//go:embed schema/world_actor.v1.schema.json
var worldActorSchemaJSON string

// worldActorLocationRuleMarker is the load-bearing substring of the "truth carries a location, never
// who perceives it" rule (the B-growth invariant — design doc Unit 5). world_actor.txt carries it
// verbatim; a content test greps for it to prove the rule reaches the model.
const worldActorLocationRuleMarker = "YOUR INTRUSION HAPPENS AT A LOCATION"

// worldActorNoAppropriatenessMarker is the load-bearing substring of the no-mood-filter rule
// (RULINGS-2026-07-24 §5 — disruption at the worst moment is a feature). world_actor.txt carries it
// verbatim; a content test greps for it.
const worldActorNoAppropriatenessMarker = "NO APPROPRIATENESS OR MOOD FILTER"

// worldSliceScene is the minimal shape buildWorldActorPrompt reads back out of the raw fn_world_slice
// jsonb to render a human-readable CURRENT SCENE line alongside the full slice — the id/name are
// already inside `slice` (the scene key fn_world_slice nests), so this is a read, not a second query.
type worldSliceScene struct {
	Scene struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"scene"`
}

// buildWorldActorPrompt renders the World Actor's prompt: the stable header (world_actor.txt, which
// carries every authoring rule — attribution, the location invariant, the presence-boundary power, no
// appropriateness filter) → the gathered WORLD slice (fn_world_slice's raw jsonb, verbatim — TRUTH-side,
// never perception-scoped) → a readable CURRENT SCENE line (the v1 manifest-at-the-scene target, pulled
// back out of the slice's own `scene` key) → the DRAWN SIZE constraint (the input the pressure roll
// hands this seat — a validated enum, like the tension tiers). size is rendered verbatim so a test can
// assert the seat was handed the drawn constraint (task-8-brief Step 1).
func buildWorldActorPrompt(slice string, size string) string {
	var sb strings.Builder
	sb.WriteString(worldActorSystemHeader)

	sb.WriteString("\n\nWORLD (the gathered world slice — presence, locations, the pending ledger, recent canon, and the current scene):\n")
	sb.WriteString(slice)

	var parsed worldSliceScene
	if err := json.Unmarshal([]byte(slice), &parsed); err == nil && parsed.Scene.ID != "" {
		sb.WriteString("\n\nCURRENT SCENE: ")
		sb.WriteString(parsed.Scene.Name)
		sb.WriteString(" (")
		sb.WriteString(parsed.Scene.ID)
		sb.WriteString(") — v1: your authored intrusion must manifest perceivably HERE.\n")
	}

	sb.WriteString("\nDRAWN SIZE: ")
	sb.WriteString(size)
	sb.WriteString(" — author EXACTLY ONE intrusion sized ")
	sb.WriteString(size)
	sb.WriteString(". No more, no less.\n")

	return sb.String()
}
