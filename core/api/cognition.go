package main

// Governed-by: D-1 — the LLM proposes, a deterministic gate decides. This file IS that gate for the
// cognition seats: present-actor-only, the closed `none | commit | telegraph` set, and the shared
// validateAttemptFields — the same shapes and the same pipeline as the player's chain, no bypass.
// Also ADR-009 — the closed attempt vocabulary (allowedBeatTypesV2, minus UNRESOLVED and QUERY,
// because NPCs act and never ask) is the post-hoc belt behind the structured-output leash, exactly
// as ruling.go is that belt for the player's chain.
// Derived 2026-08-28 from this file's own checks — promoted out of docs/domains/npc-cognition.tech.md,
// where it was prose no test could hold (D-9).

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed schema/npc_attempts.v1.schema.json
var npcAttemptsSchemaJSON string

// Package-internal decode for the cognition seats' output. A decision may
// only speak for a PRESENT actor, and its attempt obeys the same six-type
// field rules as the player's chain (no bypass: same shapes, same pipeline).
type NPCDecision struct {
	ActorID  string
	Reaction *NPCReaction
}

type NPCReaction struct {
	Attempt    Attempt
	CommitKind string // "commit" | "telegraph" (the held wind-up)
}

func DecodeAndValidateNPCDecisions(raw string, presentIDs []string) ([]NPCDecision, error) {
	present := make(map[string]bool, len(presentIDs))
	for _, id := range presentIDs {
		present[id] = true
	}
	var rows []struct {
		ActorID  string          `json:"actor_id"`
		Decision json.RawMessage `json:"decision"`
	}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, fmt.Errorf("npc decisions not valid JSON: %w", err)
	}
	out := make([]NPCDecision, 0, len(rows))
	for i, row := range rows {
		if !present[row.ActorID] {
			return nil, fmt.Errorf("decision %d for non-present actor %s", i, row.ActorID)
		}
		var none string
		if err := json.Unmarshal(row.Decision, &none); err == nil {
			if none != "none" {
				return nil, fmt.Errorf("decision %d string %q, want \"none\"", i, none)
			}
			out = append(out, NPCDecision{ActorID: row.ActorID})
			continue
		}
		var react struct {
			CommitKind string  `json:"commit_kind"`
			Attempt    Attempt `json:"attempt"`
		}
		if err := json.Unmarshal(row.Decision, &react); err != nil {
			return nil, fmt.Errorf("decision %d: %w", i, err)
		}
		if react.CommitKind != "commit" && react.CommitKind != "telegraph" {
			return nil, fmt.Errorf("decision %d commit_kind %q", i, react.CommitKind)
		}
		// NPCs act, never ask — QUERY (and UNRESOLVED) are player-decompose-only elements; an NPC
		// decision typed either is rejected here rather than falling through to applyNPCDecisions'
		// default case, which would otherwise route a question into o.adjudicate as if it were an action.
		if react.Attempt.Stated == "" || !allowedBeatTypesV2[react.Attempt.Type] || react.Attempt.Type == "UNRESOLVED" || react.Attempt.Type == "QUERY" {
			return nil, fmt.Errorf("decision %d attempt type %q invalid", i, react.Attempt.Type)
		}
		if err := validateAttemptFields(i, react.Attempt); err != nil {
			return nil, err
		}
		out = append(out, NPCDecision{ActorID: row.ActorID, Reaction: &NPCReaction{Attempt: react.Attempt, CommitKind: react.CommitKind}})
	}
	return out, nil
}
