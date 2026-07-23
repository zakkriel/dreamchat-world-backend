package main

import (
	"encoding/json"
	_ "embed"
	"fmt"
)

//go:embed schema/ruling.v1.schema.json
var rulingV1SchemaJSON string

type Ruling struct {
	Reasoning string         `json:"reasoning"`
	Therefore string         `json:"therefore"`
	Outcome   RulingOutcome  `json:"outcome"`
}

type RulingOutcome struct {
	Kind             string            `json:"kind"`
	Reason           string            `json:"reason,omitempty"`
	Events           []RuledEvent      `json:"events,omitempty"`
	AttributeWrites  []AttrWrite       `json:"attribute_writes,omitempty"`
	Mints            []json.RawMessage `json:"mints,omitempty"`
	Tension          string            `json:"tension,omitempty"`
}

type RuledEvent struct {
	Type           string   `json:"type"`
	Summary        string   `json:"summary"`
	ParticipantIDs []string `json:"participant_ids"`
	Visible        bool     `json:"visible"`
	HiddenSummary  string   `json:"hidden_summary,omitempty"`
}

type AttrWrite struct {
	TargetID  string          `json:"target_id"`
	Attribute string          `json:"attribute"`
	Value     json.RawMessage `json:"value"`
	Tier      int             `json:"tier"`
}

// DecodeAndValidateRuling enforces the explain-first contract mechanically:
// therefore=impossible ⇔ bounce; succeeds|fails ⇔ resolved(≥1 event).
// A FAILURE is an outcome and writes canon (the keeper hardens and lies);
// an impossibility writes nothing ("you can't fly").
func DecodeAndValidateRuling(raw string) (Ruling, error) {
	var r Ruling
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return r, fmt.Errorf("ruling not valid JSON: %w", err)
	}
	if r.Reasoning == "" {
		return r, fmt.Errorf("ruling missing reasoning (explain-first)")
	}
	switch r.Therefore {
	case "impossible":
		if r.Outcome.Kind != "bounce" {
			return r, fmt.Errorf("therefore=impossible but outcome=%q — flip", r.Outcome.Kind)
		}
	case "succeeds", "fails":
		if r.Outcome.Kind != "resolved" || len(r.Outcome.Events) == 0 {
			return r, fmt.Errorf("therefore=%q but outcome=%q/%d events — flip", r.Therefore, r.Outcome.Kind, len(r.Outcome.Events))
		}
	default:
		return r, fmt.Errorf("therefore %q not in enum", r.Therefore)
	}
	return r, nil
}
