package main

import (
	_ "embed"
	"encoding/json"
	"strings"
)

// Text lives in prompts/resolve.txt (core/api/prompts/README.md) — every fixed prompt rulebook
// readable in one place, config-style, mirroring the schema/*.json + go:embed pattern.
//
//go:embed prompts/resolve.txt
var resolveSystemHeader string

// resolveSpeechIsOwnEventMarker is the load-bearing substring of the dialogue-is-its-own-event
// rule (founder gate: a referee ruling embedded an NPC's spoken reply inside another event's truth
// text instead of emitting it as its own Communicated event — the narration belt then correctly
// refused an unverifiable speech card, and her words rendered as unattributed world-prose). The
// header carries it verbatim; the content test greps for it to prove the rule reaches the model.
const resolveSpeechIsOwnEventMarker = "OWN Communicated event"

// resolveFactsRuleMarker is the load-bearing substring of the reason-from-computed-facts rule
// (Grounded Reasoning / Unit 1 → adjudication): the FACTS section is engine-computed truth the referee
// must reason FROM, never contradict or invent. resolve.txt carries it verbatim; the content test greps
// for it to prove the rule reaches the model.
const resolveFactsRuleMarker = "reason from these"

// buildResolvePrompt renders the referee prompt. playerAnswer is empty on every call except a
// reaction whose decompose emitted ZERO attempts — see the RULINGS-2026-07-24 §7 comment below.
//
// factSheet is the action-scoped, TRUTH-SIDE physics fact sheet computed by fn_fact_sheet (Grounded
// Reasoning / Unit 1): the deterministic distances, move durations, reachability, weight, and lock/open
// state among the involved entities. It is rendered as its OWN section AFTER the gathered slice and
// BEFORE the mutable attempt tail, preserving the stable→mutable cache layout (the header + slice +
// facts are stable across a repair retry; only the attempts/repair block change). Empty ⇒ omitted (a
// degenerate adjudicate with no involved entities has nothing to compute).
func buildResolvePrompt(slice string, factSheet string, set []ActorAttempt, repairErrs []string, playerAnswer string) string {
	var sb strings.Builder
	sb.WriteString(resolveSystemHeader)
	sb.WriteString("\n\nFACTS (the gathered slice):\n")
	sb.WriteString(slice)
	// The engine-computed physics facts (Unit 1). Rendered right after the slice, before the attempts, so
	// the referee reasons FROM the measured truth (distance/duration/reachability) instead of guessing it.
	if factSheet != "" {
		sb.WriteString("\n\nFACTS (computed by the engine — reason from these; do not invent distances, durations, or reachability):\n")
		sb.WriteString(factSheet)
	}
	sb.WriteString("\n\nATTEMPT(S) to resolve (one combined judgment covering all of them):\n")
	// One line per attempt, attributed to its actor — the referee must know who does what
	// (RULINGS-2026-07-23 §9's wall note licenses the referee to see all involved parties).
	for _, aa := range set {
		attJSON, _ := json.Marshal(aa.Attempt)
		sb.WriteString("ACTOR ")
		sb.WriteString(aa.ActorID)
		sb.WriteString(" ATTEMPTS: ")
		sb.Write(attJSON)
		sb.WriteString("\n")
	}
	// RULINGS-2026-07-24 §7 (the empty reaction): decompose can legitimately emit ZERO attempts
	// from a reaction ("I do nothing", "I just watch") — stillness is a real answer. The player's
	// raw text still enters the combined ruling as their stated answer, marked words-not-an-act: no
	// typed attempt is invented for it, and it commits no canon event of its own. Example: Jonas's
	// cut-in is held, the player types "I just watch" (decompose emits []) — the referee reads it as
	// not-contesting and Jonas's act likely completes. Rendered AFTER the ACTOR lines, before any
	// repair block. playerAnswer is empty (omitted) whenever a first attempt exists — its own
	// `stated` field already carries the player's words per §2, so forwarding it here too would
	// double-inject — and on every non-reaction adjudicate call.
	if playerAnswer != "" {
		// Render the answer via json.Marshal, not raw quotes: it escapes newlines and quotes so a
		// crafted answer cannot forge an `ACTOR <uuid> ATTEMPTS:` line or a repair block by embedding
		// raw newlines. The marshaled string keeps its own surrounding double-quotes.
		answerJSON, _ := json.Marshal(playerAnswer)
		sb.WriteString("\nTHE PLAYER'S ANSWER (stated, not an act): ")
		sb.Write(answerJSON)
		sb.WriteString("\n")
	}
	if len(repairErrs) > 0 {
		sb.WriteString("\n\nYOUR PREVIOUS ANSWER WAS REJECTED — fix exactly these violations and answer again:\n- ")
		sb.WriteString(strings.Join(repairErrs, "\n- "))
	}
	return sb.String()
}
