package main

import (
	"encoding/json"
	_ "embed"
	"fmt"
)

//go:embed schema/ruling.v1.schema.json
var rulingV1SchemaJSON string

//go:embed schema/ruling.v2.schema.json
var rulingV2SchemaJSON string

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

// ── ruling/2 types ──────────────────────────────────────────────────────────

// RulingV2 is the rich ruling contract (Station D). Fields are frozen for Tasks 4-7.
type RulingV2 struct {
	Reasoning string     `json:"reasoning"`
	Therefore string     `json:"therefore"`
	Outcome   OutcomeV2  `json:"outcome"`
}

// OutcomeV2 mirrors RulingOutcome but references the v2 event shape.
type OutcomeV2 struct {
	Kind            string         `json:"kind"`
	Reason          string         `json:"reason,omitempty"`
	Events          []RuledEventV2 `json:"events,omitempty"`
	AttributeWrites []AttrWrite    `json:"attribute_writes,omitempty"`
}

// RuledEventV2 carries TRUTH (canon) + optional default APPEARANCE + optional per-receiver
// variants (§4 of RULINGS-2026-07-24). ActorID and Truth are always required.
// Visible nil ⇒ true. Typed slots mirror Attempt; see validateRuledEventFields.
type RuledEventV2 struct {
	Type             string            `json:"type"`
	ActorID          string            `json:"actor_id"`
	Truth            string            `json:"truth"`
	Appearance       string            `json:"appearance,omitempty"`
	Visible          *bool             `json:"visible,omitempty"`
	ReceiverVariants []ReceiverVariant `json:"receiver_variants,omitempty"`
	// Typed id slots — mirror Attempt fields (sans stated/component_ids/reference/candidate_ids).
	ToLocationID string `json:"to_location_id,omitempty"`
	ListenerID   string `json:"listener_id,omitempty"`
	Content      string `json:"content,omitempty"`
	ObjectID     string `json:"object_id,omitempty"`
	DestKind     string `json:"dest_kind,omitempty"`
	DestID       string `json:"dest_id,omitempty"`
	TargetID     string `json:"target_id,omitempty"`
	GranteeID    string `json:"grantee_id,omitempty"`
}

// ReceiverVariant grants a specific perceiver a sharper or duller read of a ruled event.
type ReceiverVariant struct {
	ReceiverID string `json:"receiver_id"`
	Text       string `json:"text"`
}

// validateRuledEventFields enforces per-type required-slot checks for ruling/2 events,
// mirroring validateAttemptFields in beatseats.go. UNRESOLVED is not a valid ruled-event type.
func validateRuledEventFields(i int, e RuledEventV2) error {
	switch e.Type {
	case "ActorMoved":
		if e.ToLocationID == "" {
			return fmt.Errorf("event %d ActorMoved requires to_location_id", i)
		}
	case "Communicated":
		if e.ListenerID == "" || e.Content == "" {
			return fmt.Errorf("event %d Communicated requires listener_id+content", i)
		}
	case "ObjectRelocated":
		if e.ObjectID == "" || e.DestKind == "" || e.DestID == "" {
			return fmt.Errorf("event %d ObjectRelocated requires object_id+dest_kind+dest_id", i)
		}
		switch e.DestKind {
		case "actor", "location", "container":
		default:
			return fmt.Errorf("event %d ObjectRelocated dest_kind %q not in actor|location|container", i, e.DestKind)
		}
	case "OwnershipAccessChanged", "EntityDestroyed", "AttributeChanged":
		if e.TargetID == "" {
			return fmt.Errorf("event %d %s requires target_id", i, e.Type)
		}
	}
	return nil
}

// allowedRuledEventTypes is the closed set for ruling/2 events. UNRESOLVED is excluded:
// a resolved ruling must commit a definite outcome.
var allowedRuledEventTypes = map[string]bool{
	"ActorMoved":             true,
	"Communicated":           true,
	"ObjectRelocated":        true,
	"OwnershipAccessChanged": true,
	"EntityDestroyed":        true,
	"AttributeChanged":       true,
}

// DecodeAndValidateRulingV2 enforces the ruling/2 contract:
//   - same therefore↔outcome match as v1
//   - per event: Type in closed set (no UNRESOLVED), ActorID + Truth non-empty, per-type slot checks
//   - each AttrWrite: Tier ∈ {1,2}, target_id+attribute+value present
func DecodeAndValidateRulingV2(raw string) (RulingV2, error) {
	var r RulingV2
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return r, fmt.Errorf("ruling/2 not valid JSON: %w", err)
	}
	if r.Reasoning == "" {
		return r, fmt.Errorf("ruling/2 missing reasoning (explain-first)")
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
	for i, e := range r.Outcome.Events {
		if !allowedRuledEventTypes[e.Type] {
			return r, fmt.Errorf("event %d type %q outside closed vocabulary", i, e.Type)
		}
		if e.ActorID == "" {
			return r, fmt.Errorf("event %d missing actor_id", i)
		}
		if e.Truth == "" {
			return r, fmt.Errorf("event %d missing truth", i)
		}
		if err := validateRuledEventFields(i, e); err != nil {
			return r, err
		}
	}
	for i, w := range r.Outcome.AttributeWrites {
		if w.TargetID == "" {
			return r, fmt.Errorf("attribute_write %d missing target_id", i)
		}
		if w.Attribute == "" {
			return r, fmt.Errorf("attribute_write %d missing attribute", i)
		}
		if w.Value == nil {
			return r, fmt.Errorf("attribute_write %d missing value", i)
		}
		if w.Tier != 1 && w.Tier != 2 {
			return r, fmt.Errorf("attribute_write %d tier %d not in {1,2}", i, w.Tier)
		}
	}
	return r, nil
}

// ── ruling/1 ────────────────────────────────────────────────────────────────

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
