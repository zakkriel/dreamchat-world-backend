package main

import (
	"encoding/json"
	"testing"
)

// TestVerdictRulingActorIDNotInSlice tests that an event's actor_id must be in sliceIDs or attemptIDs
func TestVerdictRulingActorIDNotInSlice(t *testing.T) {
	r := RulingV2{
		Reasoning: "The actor moved.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:       "ActorMoved",
					ActorID:    "unknown-uuid-12345",
					Truth:      "moved to location",
					ToLocationID: "loc-456",
				},
			},
		},
	}

	sliceIDs := map[string]bool{"actor-123": true}
	attemptIDs := []string{"attempt-789"}

	violations := verdictRuling(r, sliceIDs, attemptIDs)

	if len(violations) == 0 {
		t.Fatalf("expected violation for actor_id not in slice, got none")
	}

	// Should mention the unknown actor_id
	found := false
	for _, v := range violations {
		if contains(v, "unknown-uuid-12345") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("violation does not mention the unknown actor_id: %v", violations)
	}
}

// TestVerdictRulingTier1ValidWrite tests Tier-1 attribute write with correct type
func TestVerdictRulingTier1ValidWrite(t *testing.T) {
	r := RulingV2{
		Reasoning: "The door was locked.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:     "AttributeChanged",
					ActorID:  "actor-123",
					Truth:    "locked door",
					TargetID: "door-456",
				},
			},
			AttributeWrites: []AttrWrite{
				{
					TargetID:  "door-456",
					Attribute: "locked",
					Value:     json.RawMessage(`true`),
					Tier:      1,
				},
			},
		},
	}

	sliceIDs := map[string]bool{"actor-123": true, "door-456": true}
	attemptIDs := []string{}

	violations := verdictRuling(r, sliceIDs, attemptIDs)

	if len(violations) != 0 {
		t.Errorf("expected no violations for valid Tier-1 write, got: %v", violations)
	}
}

// TestVerdictRulingTier1InvalidType tests Tier-1 attribute write with wrong type
func TestVerdictRulingTier1InvalidType(t *testing.T) {
	r := RulingV2{
		Reasoning: "The door state changed.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:     "AttributeChanged",
					ActorID:  "actor-123",
					Truth:    "changed state",
					TargetID: "door-456",
				},
			},
			AttributeWrites: []AttrWrite{
				{
					TargetID:  "door-456",
					Attribute: "locked",
					Value:     json.RawMessage(`"yes"`), // string, not boolean
					Tier:      1,
				},
			},
		},
	}

	sliceIDs := map[string]bool{"actor-123": true, "door-456": true}
	attemptIDs := []string{}

	violations := verdictRuling(r, sliceIDs, attemptIDs)

	if len(violations) == 0 {
		t.Fatalf("expected violation for wrong type on locked, got none")
	}

	found := false
	for _, v := range violations {
		if contains(v, "locked") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("violation does not mention the attribute 'locked': %v", violations)
	}
}

// TestVerdictRulingTier1UnknownAttribute tests Tier-1 write with attribute not in registry
func TestVerdictRulingTier1UnknownAttribute(t *testing.T) {
	r := RulingV2{
		Reasoning: "Applied a curse.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:     "AttributeChanged",
					ActorID:  "actor-123",
					Truth:    "applied curse",
					TargetID: "target-789",
				},
			},
			AttributeWrites: []AttrWrite{
				{
					TargetID:  "target-789",
					Attribute: "cursed", // not in tier1Registry
					Value:     json.RawMessage(`true`),
					Tier:      1,
				},
			},
		},
	}

	sliceIDs := map[string]bool{"actor-123": true, "target-789": true}
	attemptIDs := []string{}

	violations := verdictRuling(r, sliceIDs, attemptIDs)

	if len(violations) == 0 {
		t.Fatalf("expected violation for unknown Tier-1 attribute, got none")
	}

	found := false
	for _, v := range violations {
		if contains(v, "cursed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("violation does not mention the attribute 'cursed': %v", violations)
	}
}

// TestVerdictRulingTier2WallViolation tests that Tier-2 cannot write a Tier-1 attribute
func TestVerdictRulingTier2WallViolation(t *testing.T) {
	r := RulingV2{
		Reasoning: "Attempted to write locked as Tier-2.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:     "AttributeChanged",
					ActorID:  "actor-123",
					Truth:    "state change",
					TargetID: "door-456",
				},
			},
			AttributeWrites: []AttrWrite{
				{
					TargetID:  "door-456",
					Attribute: "locked", // Tier-1 name but written as Tier-2
					Value:     json.RawMessage(`true`),
					Tier:      2,
				},
			},
		},
	}

	sliceIDs := map[string]bool{"actor-123": true, "door-456": true}
	attemptIDs := []string{}

	violations := verdictRuling(r, sliceIDs, attemptIDs)

	if len(violations) == 0 {
		t.Fatalf("expected wall violation for Tier-2 write to Tier-1 attribute, got none")
	}

	found := false
	for _, v := range violations {
		if contains(v, "Tier-1 engine attribute") || contains(v, "wall violation") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("violation does not mention the tier wall: %v", violations)
	}
}

// TestVerdictRulingTier2ValidWrite tests Tier-2 attribute write with unknown name (should pass)
func TestVerdictRulingTier2ValidWrite(t *testing.T) {
	r := RulingV2{
		Reasoning: "Applied a property.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:     "AttributeChanged",
					ActorID:  "actor-123",
					Truth:    "applied property",
					TargetID: "item-789",
				},
			},
			AttributeWrites: []AttrWrite{
				{
					TargetID:  "item-789",
					Attribute: "barred_from_inside",
					Value:     json.RawMessage(`"oak beam"`),
					Tier:      2,
				},
			},
		},
	}

	sliceIDs := map[string]bool{"actor-123": true, "item-789": true}
	attemptIDs := []string{}

	violations := verdictRuling(r, sliceIDs, attemptIDs)

	if len(violations) != 0 {
		t.Errorf("expected no violations for valid Tier-2 write, got: %v", violations)
	}
}

// jsonTypeOf tests
func TestJsonTypeOfString(t *testing.T) {
	tests := []struct {
		name     string
		input    json.RawMessage
		expected string
	}{
		{"string", json.RawMessage(`"hello"`), "string"},
		{"number", json.RawMessage(`42`), "number"},
		{"negative number", json.RawMessage(`-3.14`), "number"},
		{"boolean true", json.RawMessage(`true`), "boolean"},
		{"boolean false", json.RawMessage(`false`), "boolean"},
		{"array", json.RawMessage(`[1,2,3]`), "array"},
		{"object", json.RawMessage(`{"key":"value"}`), "object"},
		{"null", json.RawMessage(`null`), "null"},
		{"whitespace string", json.RawMessage(`  "hello"  `), "string"},
		{"whitespace number", json.RawMessage(`  42  `), "number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonTypeOf(tt.input)
			if result != tt.expected {
				t.Errorf("jsonTypeOf(%q) = %q, want %q", string(tt.input), result, tt.expected)
			}
		})
	}
}

// helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0)
}
