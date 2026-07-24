package main

import (
	"encoding/json"
	"strings"
)

const resolveSystemHeader = `You are the impartial referee of this world. Rules:
EXPLAIN FIRST: write your full reasoning from the facts, then write exactly one word: therefore: succeeds|fails|impossible, then state the outcome.
A FAILURE is an outcome that writes canon — the keeper hardens and records it. An impossibility bounces: nothing is written.
EVENT TYPES (the only six): ActorMoved, Communicated, ObjectRelocated, OwnershipAccessChanged, EntityDestroyed, AttributeChanged.
Every event carries: actor_id (who causes it) + truth (what REALLY happens — canon never lies). Optionally: appearance (what a plain observer sees) and receiver_variants (specific perceivers who see something different).
Use ONLY entity ids that appear in the FACTS section.
ATTRIBUTE WRITES: tier 1 = engine-known mechanics (open, locked, connects, size, weight, max_room, occupied_room, empty_weight, max_load, carried_weight, base_speed, location_id, coordinates, tension). A fact that physically stops people goes in tier 1. Tier 2 = free descriptive state. Write both tiers when a mechanic has meaning.`

func buildResolvePrompt(slice string, set []ActorAttempt, repairErrs []string) string {
	var sb strings.Builder
	sb.WriteString(resolveSystemHeader)
	sb.WriteString("\n\nFACTS (the gathered slice):\n")
	sb.WriteString(slice)
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
	if len(repairErrs) > 0 {
		sb.WriteString("\n\nYOUR PREVIOUS ANSWER WAS REJECTED — fix exactly these violations and answer again:\n- ")
		sb.WriteString(strings.Join(repairErrs, "\n- "))
	}
	return sb.String()
}
