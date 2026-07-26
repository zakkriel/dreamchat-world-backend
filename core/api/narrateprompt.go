package main

import "strings"

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

const narrateSystemHeader = `You are the narrator of this world. Render ONLY the perceptions listed below into second-person, present-tense prose. Refer to entities by the names given under PRESENT and in the perceptions; use no other names.
INVENT NOTHING that is not in the perceptions — no objects, sounds, smells, figures, or motives the lines do not contain. Example: given only "Mara watches you from the bar", you write that she watches you; you do NOT add a guttering lantern, a smell of brine, or a second figure at the door.
If the perceptions are sparse, the narration is SHORT AND SPARSE — that is correct, not a shortcoming. Two lines of perception become at most a sentence or two; never pad to fill the space.
NEVER address the player out-of-world, and NEVER offer options, alternative frames, scripted branches, or meta-commentary of any kind. Example: you do not write "I could tell this in third person instead, or give you a menu of choices" — you write the scene and nothing else.
Output is prose only: no headings, no lists, no notes to the reader.`

// buildNarratePrompt assembles the narrate prompt: the bounding header, then the PRESENT roster (names
// only, from the payload's candidate whitelist), then the PERCEPTIONS the holder can currently perceive
// (oldest first — payload.Lines is already acquired_tick-ordered). Layout is cache-native like the
// cognition prompts (stable header prefix; the growing perceptions ride the tail).
//
// Scope fence: what the payload CONTAINS (retrieval, fidelity, richness) is Station I's narrator-payload
// map item — this builder does not add retrieval or SQL. It only makes the payload the seat already
// holds actually reach the model, and bounds what the model may do with it. With zero lines the
// PERCEPTIONS body is empty and nothing is fabricated — a sparse moment is a short narration.
func buildNarratePrompt(payload PerceptionPayload) string {
	var sb strings.Builder
	sb.WriteString(narrateSystemHeader)

	// PRESENT — names only. Who is here is public; the narrator refers to them by exactly these names.
	names := make([]string, 0, len(payload.Candidates))
	for _, c := range payload.Candidates {
		names = append(names, c.Name)
	}
	sb.WriteString("\n\nPRESENT: ")
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
