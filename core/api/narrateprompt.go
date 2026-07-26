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

// buildNarratePrompt assembles the narrate prompt: the bounding header, then the PLACE (the location
// candidate) that sets the scene, then the PRESENT roster (actor names only, from the payload's
// candidate whitelist), then the PERCEPTIONS the holder can currently perceive (oldest first —
// payload.Lines is already acquired_tick-ordered). Layout is cache-native like the
// cognition prompts (stable header prefix; the growing perceptions ride the tail).
//
// Scope fence: what the payload CONTAINS (retrieval, fidelity, richness) is Station I's narrator-payload
// map item — this builder does not add retrieval or SQL. It only makes the payload the seat already
// holds actually reach the model, and bounds what the model may do with it. With zero lines the
// PERCEPTIONS body is empty and nothing is fabricated — a sparse moment is a short narration.
func buildNarratePrompt(payload PerceptionPayload) string {
	var sb strings.Builder
	sb.WriteString(narrateSystemHeader)

	// PLACE — where the scene is set, rendered from the location candidate (kind 'location'). Orientation
	// the narrator was missing: without it the location was mixed into PRESENT as if it were a person.
	// Actors go under PRESENT; the location goes here, before PRESENT (§10 orientation, not licence to
	// describe the room — anti-invention still binds).
	var place string
	names := make([]string, 0, len(payload.Candidates))
	for _, c := range payload.Candidates {
		if c.Kind == "location" {
			place = c.Name
			continue
		}
		names = append(names, c.Name)
	}
	if place != "" {
		sb.WriteString("\n\nPLACE: ")
		sb.WriteString(place)
		sb.WriteString("\nPRESENT: ")
	} else {
		sb.WriteString("\n\nPRESENT: ")
	}
	// PRESENT — names only. Who is here is public; the narrator refers to them by exactly these names.
	sb.WriteString(strings.Join(names, ", "))

	// PERCEPTIONS — the ONLY content the narrator may render, oldest first. One bullet per line; with
	// no lines this section is empty (nothing invented to fill it — sparse-is-correct, per the header).
	sb.WriteString("\nPERCEPTIONS (oldest first):\n")
	for _, l := range payload.Lines {
		sb.WriteString("- ")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}
