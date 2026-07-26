package main

import (
	_ "embed"
	"strings"
)

// The narrate seat's prompt assembly — the SEAT BOUNDARY where the perception payload becomes the
// model's world. The driver drops req.Payload (D-13 keeps provider shaping IN the driver; the seat
// therefore assembles what the model must see HERE, at the call site). narrate is perception-bound
// (ADR-020): no omniscient pass, only what the holder can currently perceive.
//
// Founder-gate bug this bounds: the live narrator was handed the one-line instruction ONLY — the
// payload was silently dropped by the driver — and, with nothing to render, it invented an entire
// scene (unnamed figures, sounds, smells in no perception) and then broke frame to offer the player
// "a different frame — full third-person, inner monologue, a scripted branch." Two rules close both
// failures: INVENT NOTHING (anti-invention) and the no-meta rule. Each ships with a one-line example
// (house rule: every rule ships with its example) so the boundary is unambiguous.

// narrateAntiInventionMarker is the load-bearing substring of the anti-invention rule. The header
// carries it verbatim; the seat-boundary test greps for it to prove the narrator is bounded.
const narrateAntiInventionMarker = "INVENT NOTHING"

// narrateSparseRuleMarker is the load-bearing substring of the sparse-is-correct rule. A sparse
// moment is a short narration — not a failure to be padded. The zero-lines unit test greps for it.
const narrateSparseRuleMarker = "SHORT AND SPARSE"

// Text lives in prompts/narrate.txt (core/api/prompts/README.md) — every fixed prompt rulebook
// readable in one place, config-style, mirroring the schema/*.json + go:embed pattern.
//
//go:embed prompts/narrate.txt
var narrateSystemHeader string

// buildNarratePrompt assembles the narrate prompt: the bounding header, then PLACE (the location
// candidate — with its scene description when the payload carries one), then the PRESENT roster (actor
// names only, the VIEWER excluded), then the delta-first body: WHAT JUST HAPPENED (the perceptions new
// this beat, oldest first) followed by RECENT BACKGROUND (the rest of the window, labelled as context
// the narrator must not re-narrate). Layout is cache-native like the cognition prompts (stable header
// prefix; the growing perceptions ride the tail).
//
// Two founder-gate bugs this shape closes:
//   - Defect A: the viewer appeared in his OWN present roster ("…and Kade" to Kade). viewerID is
//     threaded in so the candidate whose id IS the viewer is dropped from PRESENT — you never narrate
//     the person you are narrating TO as someone standing in the room.
//   - Defect B: the narrator re-rendered the whole window every beat ("You step into the Drowned
//     Lantern" ×3). preIDs is the set of perception_ids the viewer already held BEFORE the beat; a line
//     whose id is not in it is NEW (WHAT JUST HAPPENED), the rest is RECENT BACKGROUND the header
//     forbids re-narrating. An empty delta is a pure look-around — the header renders the scene from
//     PLACE/PRESENT/description instead.
//
// Scope fence: what the payload CONTAINS (retrieval, fidelity, richness) is Station I's narrator-payload
// map item — this builder adds no retrieval or SQL. With zero lines both bodies are empty and nothing
// is fabricated — a sparse moment is a short narration.
func buildNarratePrompt(payload PerceptionPayload, viewerID string, preIDs map[string]bool) string {
	var sb strings.Builder
	sb.WriteString(narrateSystemHeader)

	// PLACE — where the scene is set, rendered from the location candidate (kind 'location'), with its
	// Tier-2 description when present. PRESENT — actor names only, the VIEWER dropped (Defect A).
	var place, placeDesc string
	names := make([]string, 0, len(payload.Candidates))
	for _, c := range payload.Candidates {
		if c.Kind == "location" {
			place = c.Name
			placeDesc = c.Description
			continue
		}
		if c.ID == viewerID {
			continue // the viewer is narrated TO, never listed as present WITH himself.
		}
		names = append(names, c.Name)
	}
	if place != "" {
		sb.WriteString("\n\nPLACE: ")
		sb.WriteString(place)
		if placeDesc != "" {
			sb.WriteString(" — ")
			sb.WriteString(placeDesc)
		}
		sb.WriteString("\nPRESENT: ")
	} else {
		sb.WriteString("\n\nPRESENT: ")
	}
	sb.WriteString(strings.Join(names, ", "))

	// Delta-first split: a line is NEW (delta) when its perception_id is not among preIDs; the rest is
	// already-known background. With no baseline (preIDs nil) every line is treated as new — the correct
	// behaviour for a first render or a direct unit call. Lines are already acquired_tick-ordered.
	var delta, background []string
	for i, l := range payload.Lines {
		var id string
		if i < len(payload.LineIDs) {
			id = payload.LineIDs[i]
		}
		if preIDs != nil && id != "" && preIDs[id] {
			background = append(background, l)
		} else {
			delta = append(delta, l)
		}
	}

	// WHAT JUST HAPPENED — the delta, the ONLY lines narrated as live events. Empty ⇒ the section is
	// omitted and the header's pure-look-around rule takes over (render the scene as it stands).
	if len(delta) > 0 {
		sb.WriteString("\nWHAT JUST HAPPENED (oldest first):\n")
		for _, l := range delta {
			sb.WriteString("- ")
			sb.WriteString(l)
			sb.WriteString("\n")
		}
	}
	// RECENT BACKGROUND — context the narrator must NOT re-narrate as new. Omitted when empty.
	if len(background) > 0 {
		sb.WriteString("\nRECENT BACKGROUND (context only — do NOT re-narrate as new):\n")
		for _, l := range background {
			sb.WriteString("- ")
			sb.WriteString(l)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
