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
// Founder-gate bugs this bounds: (1) the live narrator was handed the one-line instruction ONLY — the
// payload was silently dropped by the driver — and, with nothing to render, it invented an entire scene
// and broke frame to offer the player "a different frame — full third-person, a scripted branch";
// (2) an NPC-authored perception described the VIEWER in the third person ("the stranger coming toward
// her") and by the NPC's canonical name the viewer doesn't know ("Jonas") — the narrator rendered both
// verbatim instead of "you" and the viewer's own label. The founder's contract (prompts/narrate.txt)
// closes them with two walls the narration lives between — never play the player, never contradict or
// extend the state — plus VIEWER-RELATIVE RENDERING (the addressee is always "you"; everyone else wears
// the VIEWER's label) and licence to create sensory texture INSIDE those walls. Each rule ships with an
// example (house rule) so the boundary is unambiguous.

// narrateAntiInventionMarker is the load-bearing substring of the never-contradict-or-extend rule (the
// contract's second wall — the old "INVENT NOTHING" line, rewritten). The header carries it verbatim;
// the seat-boundary test greps for it to prove the narrator is bounded to the state.
const narrateAntiInventionMarker = "NEVER CONTRADICT OR EXTEND THE STATE"

// narrateViewerRelativeMarker is the load-bearing substring of the viewer-relative rule: the person the
// narration addresses is always "you", and everyone else wears the VIEWER's label. The unit + flow
// tests grep for it to prove the relabel instruction reaches the model alongside YOU ARE and PRESENT.
const narrateViewerRelativeMarker = "VIEWER-RELATIVE RENDERING"

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

	// YOU ARE — the viewer's own identity, rendered BETWEEN the PRESENT roster and the perception body.
	// It lists how OTHERS may name or describe the viewer inside the perception text (his descriptor, and
	// his self-known name when he holds one); the VIEWER-RELATIVE rule then binds every such reference to
	// "you". Omitted when the payload carries no aliases (an unseeded viewer) — no empty header line.
	if len(payload.ViewerAliases) > 0 {
		sb.WriteString("\nYOU ARE: ")
		sb.WriteString(strings.Join(payload.ViewerAliases, "; "))
	}

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
