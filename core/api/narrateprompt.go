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

// narrateNothingResolvedMarker is the load-bearing substring of the failed/empty-beat context (a
// smoke-beat finding: a reaction beat BOUNCED — committed=[], a held outcome still pending — and,
// handed an empty delta, the narrator AUTHORED A RESOLUTION that never happened, contradicting the
// state (rule 2) in a case the header's pure-look-around rule (halt "completed") doesn't cover. The
// unit tests grep for it.
const narrateNothingResolvedMarker = "NOTHING RESOLVED"

// nothingResolvedHalts is the set of halt reasons that, PAIRED WITH an empty delta, mean nothing
// committed this beat — the attempted action did not happen. "completed" (a pure look-around) and
// "telegraph" (the wind-up itself commits, so its delta is never empty) are deliberately excluded:
// both keep the existing look-around rendering, not NOTHING RESOLVED.
var nothingResolvedHalts = map[string]bool{
	"bounce":               true,
	"gate_reject":          true,
	"premise_broken":       true,
	"ruled_event_rejected": true,
	"unresolved":           true,
	"turn_budget":          true,
}

// narrateEmbeddedQuoteMarker is the load-bearing substring of the embedded-quotes-attribute rule
// (founder gate: a perception embedded an NPC's quoted words inside another action's text with no
// separate speech line — the narrator had folded the quote into anonymous narrator prose instead of
// that NPC's own attributed ACTION segment). The header carries it verbatim; the content test greps
// for it.
const narrateEmbeddedQuoteMarker = "attributed ACTION segment"

// narrateSegmentContractMarker delimits the base narrator rules from the structured-segment OUTPUT
// contract inside narrate.txt. The structured path ships the whole header (rules + segment contract);
// the PLAIN FALLBACK re-ask (buildNarratePlainPrompt, no schema) ships only the base rules plus a
// prose-only instruction, so a driver that could not produce valid segments still returns clean prose
// (never a JSON blob) — the beat is never failed on formatting.
const narrateSegmentContractMarker = "OUTPUT — STRUCTURED NARRATION SEGMENTS"

// Text lives in prompts/narrate.txt (core/api/prompts/README.md) — every fixed prompt rulebook
// readable in one place, config-style, mirroring the schema/*.json + go:embed pattern. The founder
// envelope added the segmenting contract ON TOP of the existing rules (all of them apply per segment).
//
//go:embed prompts/narrate.txt
var narrateSystemHeader string

// narrateBaseRules returns the base narrator rules — the header with the structured-segment OUTPUT
// contract sliced off at narrateSegmentContractMarker. Used only by the plain-prose fallback.
func narrateBaseRules() string {
	if i := strings.Index(narrateSystemHeader, narrateSegmentContractMarker); i >= 0 {
		return strings.TrimRight(narrateSystemHeader[:i], "\n ")
	}
	return narrateSystemHeader
}

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
//
// haltReason is the beat's outcome.HaltReason (beathandler already holds the BeatOutcome before
// narrating). A smoke-beat finding: a reaction beat BOUNCED (committed=[], a held outcome still
// pending) and the narrator, handed the resulting empty delta, AUTHORED A RESOLUTION that never
// happened — contradicting the state in a case the pure-look-around rule doesn't cover (that rule is
// for halt "completed", a genuine quiet moment, not a failed/rejected/halted one). When the delta is
// empty AND haltReason is one of nothingResolvedHalts, a NOTHING RESOLVED section renders before the
// perception blocks so the model is told nothing committed instead of inferring a scene from silence.
func buildNarratePrompt(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason string) string {
	return narrateSystemHeader + narrateSceneBody(payload, viewerID, preIDs, haltReason)
}

// buildNarrateRepairPrompt is the ONE repair re-ask after a structured attempt failed the schema or the
// belt (ghost speaker / non-verbatim speech): the same structured prompt with the exact violations
// attached, mirroring the resolve seat's repair pattern. Still a structured (schema-carrying) call.
func buildNarrateRepairPrompt(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason, prevErr string) string {
	return buildNarratePrompt(payload, viewerID, preIDs, haltReason) +
		"\n\nYOUR PREVIOUS ANSWER WAS REJECTED — fix exactly this and answer again with the segment array:\n" + prevErr
}

// buildNarratePlainPrompt is the FALLBACK re-ask after both structured attempts failed: base narrator
// rules only (segment contract sliced off) plus a prose-only instruction, over the identical scene. It
// is issued WITHOUT a schema so the model returns clean prose, which the handler wraps as a single
// narration segment — the beat is never failed on formatting.
func buildNarratePlainPrompt(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason string) string {
	return narrateBaseRules() +
		"\n\nOutput is prose only: a single flowing narration — no JSON, no segments, no headings, no lists, no notes to the reader." +
		narrateSceneBody(payload, viewerID, preIDs, haltReason)
}

// narrateSceneBody assembles everything after the header: PLACE, the PRESENT roster (each entry
// "label [id]" so a segment can attribute speech/actions by id), YOU ARE, the NOTHING RESOLVED context,
// and the delta-first WHAT JUST HAPPENED / RECENT BACKGROUND blocks. Shared verbatim by the structured,
// repair, and plain-fallback prompts so the scene the model renders never changes between attempts.
func narrateSceneBody(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason string) string {
	var sb strings.Builder

	// PLACE — where the scene is set, rendered from the location candidate (kind 'location'), with its
	// Tier-2 description when present. PRESENT — "label [id]" per actor, the VIEWER dropped (Defect A);
	// the id lets a speech/action segment attribute to a present NPC, never to the person narrated TO.
	var place, placeDesc string
	entries := make([]string, 0, len(payload.Candidates))
	for _, c := range payload.Candidates {
		if c.Kind == "location" {
			place = c.Name
			placeDesc = c.Description
			continue
		}
		if c.ID == viewerID {
			continue // the viewer is narrated TO, never listed as present WITH himself.
		}
		entries = append(entries, c.Name+" ["+c.ID+"]")
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
	sb.WriteString(strings.Join(entries, ", "))

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

	// NOTHING RESOLVED — the failed/empty-beat context (see the doc comment above). Rendered BEFORE the
	// perception blocks, only when the delta is empty AND the halt reason says nothing committed; a
	// "completed" halt (pure look-around) and "telegraph" (never empty here) are excluded.
	if len(delta) == 0 && nothingResolvedHalts[haltReason] {
		sb.WriteString("\n\nNOTHING RESOLVED: the attempted action did not happen; the situation is exactly as it was. Any held tension still hangs unanswered.")
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
