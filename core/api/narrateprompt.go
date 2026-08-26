package main

import (
	_ "embed"
	"encoding/json"
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

// narrateQueryRuleMarker is the load-bearing substring of the answer-the-listed-questions rule
// (Grounded Reasoning / Unit 2): the narrator answers each QUESTIONS THE PLAYER ASKED entry in-world
// FROM its computed facts, and a withheld/absent fact means "you can't tell from here" — never invent
// it. It lives BEFORE the segment contract so the plain fallback carries it too; a test greps for it.
const narrateQueryRuleMarker = "ANSWER THE PLAYER'S QUESTIONS"

// narrateQueryBlockMarker is the load-bearing substring of the rendered QUESTIONS block header — the
// per-beat section carrying the read-only queries + their perceived fact sheets (Unit 2). The query
// tests grep for it to prove a QUERY's answer reaches the narrator.
const narrateQueryBlockMarker = "QUESTIONS THE PLAYER ASKED"

// narrateWorldBlockMarker is the load-bearing substring of the rendered THE WORLD block — the world's
// GLOBAL statement (WorldStatement), the one thing no narrate prompt has ever carried. It is rendered
// by narrateSceneBody, NOT appended to narrate.txt, and that placement is the finding it comes from:
// narrateBaseRules slices the header at narrateSegmentContractMarker (:71, :93-98), so a block added to
// the end of narrate.txt is silently dropped by buildNarratePlainPrompt (:143-151) — the fallback taken
// only after two structured attempts have already failed, which is precisely when invention risk is
// highest. The scene body is shared verbatim by all three builders, so rendering it here is what makes
// "present in all three" structural instead of a thing a later edit can undo. A test greps all three.
const narrateWorldBlockMarker = "THE WORLD:"

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

// buildNarratePrompt assembles the narrate prompt: the bounding header, then THE WORLD (this world's
// global statement — what world this is at all, which no narrate prompt carried before), then PLACE
// (the location candidate — with its scene description when the payload carries one), then the PRESENT roster (actor
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
// queryAnswers (Grounded Reasoning / Unit 2) is passed VARIADIC so the many existing direct-call sites
// (narration_test / seatprompt_test) stay source-compatible — a beat with no questions spreads a nil
// slice to zero args and the QUESTIONS block is omitted; beathandler spreads outcome.QueryAnswers.
func buildNarratePrompt(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason string, queryAnswers ...QueryAnswer) string {
	return narrateSystemHeader + narrateSceneBody(payload, viewerID, preIDs, haltReason, queryAnswers...)
}

// buildNarrateRepairPrompt is the ONE repair re-ask after a structured attempt failed the schema or the
// belt (ghost speaker / non-verbatim speech): the same structured prompt with the exact violations
// attached, mirroring the resolve seat's repair pattern. Still a structured (schema-carrying) call.
func buildNarrateRepairPrompt(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason, prevErr string, queryAnswers ...QueryAnswer) string {
	return buildNarratePrompt(payload, viewerID, preIDs, haltReason, queryAnswers...) +
		"\n\nYOUR PREVIOUS ANSWER WAS REJECTED — fix exactly this and answer again with the segment array:\n" + prevErr
}

// buildNarratePlainPrompt is the FALLBACK re-ask after both structured attempts failed: base narrator
// rules only (segment contract sliced off) plus a prose-only instruction, over the identical scene. It
// is issued WITHOUT a schema so the model returns clean prose, which the handler wraps as a single
// narration segment — the beat is never failed on formatting.
func buildNarratePlainPrompt(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason string, queryAnswers ...QueryAnswer) string {
	return narrateBaseRules() +
		"\n\nOutput is prose only: a single flowing narration — no JSON, no segments, no headings, no lists, no notes to the reader." +
		narrateSceneBody(payload, viewerID, preIDs, haltReason, queryAnswers...)
}

// narrateSceneBody assembles everything after the header: THE WORLD (the world's global statement),
// then PLACE, the PRESENT roster (each entry "label [id]" so a segment can attribute speech/actions by
// id), YOU ARE, the NOTHING RESOLVED context, and the delta-first WHAT JUST HAPPENED / RECENT
// BACKGROUND blocks. Shared verbatim by the structured, repair, and plain-fallback prompts so the
// scene the model renders never changes between attempts.
func narrateSceneBody(payload PerceptionPayload, viewerID string, preIDs map[string]bool, haltReason string, queryAnswers ...QueryAnswer) string {
	var sb strings.Builder

	// THE WORLD — the global statement, rendered FIRST and identically every beat of a given world, so
	// it extends the stable cache prefix the header establishes rather than riding the growing tail.
	//
	// It is context, never content: narrate.txt's THE WORLD IS CONTEXT, NEVER CONTENT rule (which sits
	// BEFORE the segment-contract marker, so the plain fallback carries it too) binds it to diction and
	// register and forbids it adding an object, an exit, a person or an event. The facts of the scene
	// still come only from PLACE, PRESENT, YOU ARE and the perception lines.
	//
	// Every field is the committed document's own content, never world.brief — see WorldStatement.
	// An empty statement renders NOTHING, the same discipline YOU ARE follows: an unauthored world must
	// not hand the model a bare header to reason about.
	if !payload.World.Empty() {
		sb.WriteString("\n\n")
		sb.WriteString(narrateWorldBlockMarker)
		sb.WriteString(" ")
		sb.WriteString(strings.TrimSpace(payload.World.Name))
		if premise := strings.TrimSpace(payload.World.Premise); premise != "" {
			sb.WriteString(" — ")
			sb.WriteString(premise)
		}
		if region := strings.TrimSpace(payload.World.Region); region != "" {
			sb.WriteString("\nTHE REGION: ")
			sb.WriteString(region)
		}
		// The register words are this world's OWN minted vocabulary (world_genesis.v1 specifies mood and
		// ornament as free vocabulary the author coins, not an enum), which is why they are handed over
		// verbatim rather than translated into house adjectives.
		register := make([]string, 0, 2)
		if mood := strings.TrimSpace(payload.World.Mood); mood != "" {
			register = append(register, mood)
		}
		if ornament := strings.TrimSpace(payload.World.Ornament); ornament != "" {
			register = append(register, ornament)
		}
		if len(register) > 0 {
			sb.WriteString("\nITS REGISTER: ")
			sb.WriteString(strings.Join(register, ", "))
		}
	}

	// PLACE — where the scene is set, rendered from the location candidate (kind 'location'), with its
	// Tier-2 description when present. PRESENT — "label [id]" per actor, the VIEWER dropped (Defect A);
	// the id lets a speech/action segment attribute to a present NPC, never to the person narrated TO.
	// PLACE is matched BY ID against the room the viewer stands in. It used to be "the last candidate
	// of kind location", which was correct only while exactly one location could ever be a candidate.
	// SPEC-030 widened the whitelist to the portals of this room and the rooms on their far side, and
	// buildScene was fixed for exactly this reason — the narrator was not, so it has been setting
	// scenes in a neighbouring room ever since. Live symptom: the founder's look-around beat in the
	// tavern opened "The dim cellar air presses close around you". The Cellar is through the hatch.
	var place, placeDesc string
	entries := make([]string, 0, len(payload.Candidates))
	for _, c := range payload.Candidates {
		if c.Kind == "location" {
			// No Here (a direct unit call, or an unplaced viewer): fall back to the old behaviour
			// rather than rendering a placeless scene.
			if payload.Here == "" || c.ID == payload.Here {
				place = c.Name
				placeDesc = c.Description
			}
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

	// QUESTIONS THE PLAYER ASKED — the read-only QUERY answers this beat (Grounded Reasoning / Unit 2),
	// rendered AFTER the events so one narration covers a mixed [action, question] beat. Each entry is the
	// question + its PERCEIVED fact sheet (perception-scoped by fn_fact_sheet — a closed container's
	// contents are already withheld), embedded VERBATIM; the narrate.txt rule tells the narrator to answer
	// each in-world from those facts and to say "you can't tell from here" for a withheld/absent fact
	// rather than invent it. Omitted entirely when the beat carried no questions.
	if len(queryAnswers) > 0 {
		sb.WriteString("\n\nQUESTIONS THE PLAYER ASKED (answer each in-world from its facts; tell them ONLY what they'd perceive):\n")
		for _, qa := range queryAnswers {
			sb.WriteString("- ")
			sb.WriteString(qa.Stated)
			sb.WriteString("\n  facts: ")
			sb.WriteString(narratorFactSheet(qa.FactSheet))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// narratorFactSheet rewrites a fact sheet for the NARRATOR: measured geometry becomes a staging
// band, and the raw numbers never reach the prompt.
//
// The engine measures the world in metres and seconds because the cognition and resolve seats need
// to reason with them — and the cognition path still gets them, untouched. The narrator does not.
// Handed distance_m and move_duration_s it recites them, because that is the only honest thing to
// do with a number you have been given: the founder's first live beat came back "maybe nine strides
// off — close to a seven-count", "barely five meters distant, a mere four-second walk". Accurate,
// and a range table rather than a room.
//
// So the numbers are translated here, at the narrator's boundary and nowhere else. Proximity still
// informs staging — who is close, who is across the room — which is what the geometry was FOR; what
// is removed is the ability to read it out. Every other field (open, locked, contents, reachable,
// weight) passes through untouched: those are perceptible facts a person could state, not
// instrument readings.
//
// Unparseable input is returned verbatim. A fact sheet the narrator cannot read is a worse failure
// than one it reads too literally, and this function must never be the reason a beat has no answer.
func narratorFactSheet(raw json.RawMessage) string {
	var sheet map[string]any
	if err := json.Unmarshal(raw, &sheet); err != nil {
		return string(raw)
	}
	targets, ok := sheet["targets"].([]any)
	if !ok {
		return string(raw)
	}
	for _, t := range targets {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if d, ok := tm["distance_m"].(float64); ok {
			tm["proximity"] = proximityBand(d)
		}
		// Deleted rather than banded: a duration is the one number with no staging value at all —
		// "how many seconds to walk there" is not something a person senses, it is a stopwatch.
		delete(tm, "distance_m")
		delete(tm, "move_duration_s")
	}
	out, err := json.Marshal(sheet)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// proximityBand renders a measured distance as the narrator's own vocabulary. The thresholds are
// scene-scaled, not universal: the seeded tavern is roughly ten metres across, so "across the room"
// has to reach about that far to mean what it says.
func proximityBand(metres float64) string {
	switch {
	case metres <= 1.5:
		return "within arm's reach"
	case metres <= 4:
		return "a few steps away"
	case metres <= 12:
		return "across the room"
	case metres <= 40:
		return "some way off"
	default:
		return "far off"
	}
}
