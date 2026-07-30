package main

import (
	"encoding/json"
	"strings"
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
					ToTargetID: "loc-456",
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
		if strings.Contains(v, "unknown-uuid-12345") {
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
		if strings.Contains(v, "locked") {
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
		if strings.Contains(v, "cursed") {
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
		if strings.Contains(v, "Tier-1 engine attribute") || strings.Contains(v, "wall violation") {
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

// TestVerdictRulingEventIndex10 tests proper formatting of event index 10+
func TestVerdictRulingEventIndex10(t *testing.T) {
	events := make([]RuledEventV2, 11)
	// Populate first 10 events with valid actor IDs
	for i := 0; i < 10; i++ {
		events[i] = RuledEventV2{
			Type:    "ActorMoved",
			ActorID: "actor-123",
		}
	}
	// Event at index 10 has out-of-slice ActorID
	events[10] = RuledEventV2{
		Type:    "ActorMoved",
		ActorID: "invalid-actor-999",
	}

	r := RulingV2{
		Reasoning: "Test index 10 formatting.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind:   "resolved",
			Events: events,
		},
	}

	sliceIDs := map[string]bool{"actor-123": true}
	attemptIDs := []string{}

	violations := verdictRuling(r, sliceIDs, attemptIDs)

	// Should have violation for event 10
	found := false
	for _, v := range violations {
		if strings.Contains(v, "event 10") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("violation does not contain 'event 10': %v", violations)
	}
}

// TestVerdictRulingEntityCreatedTrueIntroductionAdmitted — a ruled EntityCreated's target_id is the ONE
// case the id-whitelist admits a not-in-slice id: a TRUE INTRODUCTION (descriptor present, §8). Its new
// id is not a violation, and it becomes referenceable by a SUBSEQUENT event in the SAME ruling (the
// created dagger is immediately relocated). Reuse-before-create (the descriptor→existing-id match) is the
// SQL commit's job (it holds the registry); the verdict only admits the true-introduction SHAPE.
func TestVerdictRulingEntityCreatedTrueIntroductionAdmitted(t *testing.T) {
	const newID = "f2000000-0000-0000-0000-000000000abc" // LLM-minted, NOT in the slice
	r := RulingV2{
		Reasoning: "Kade forges a plain dagger from the shard, then sets it on the bar.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:          "EntityCreated",
					ActorID:       "actor-123",
					Truth:         "a plain dagger takes shape under his hammer",
					TargetID:      newID,
					NewEntityKind: "artifact",
					Descriptor:    "a plain iron dagger",
				},
				{
					// SUBSEQUENT event references the just-created id — must be admitted (in-ruling scope).
					Type:     "ObjectRelocated",
					ActorID:  "actor-123",
					Truth:    "he sets the dagger on the bar",
					ObjectID: newID,
					DestKind: "location",
					DestID:   "loc-789",
				},
			},
		},
	}
	sliceIDs := map[string]bool{"actor-123": true, "loc-789": true}
	violations := verdictRuling(r, sliceIDs, []string{})
	if len(violations) != 0 {
		t.Fatalf("expected a true-introduction EntityCreated + its downstream reference to be admitted, got: %v", violations)
	}
}

// TestVerdictRulingEntityCreatedNoDescriptorViolation — a create with NO descriptor is not a true
// introduction: the whitelist does not admit its not-in-slice id, and the missing descriptor is a
// violation (descriptor mandatory, §8). This keeps the one whitelist opening from becoming a hole.
func TestVerdictRulingEntityCreatedNoDescriptorViolation(t *testing.T) {
	const newID = "f2000000-0000-0000-0000-000000000def"
	r := RulingV2{
		Reasoning: "Something half-forms.",
		Therefore: "succeeds",
		Outcome: OutcomeV2{
			Kind: "resolved",
			Events: []RuledEventV2{
				{
					Type:          "EntityCreated",
					ActorID:       "actor-123",
					Truth:         "a shape that never resolves",
					TargetID:      newID,
					NewEntityKind: "artifact",
					Descriptor:    "", // MISSING — not a true introduction
				},
			},
		},
	}
	sliceIDs := map[string]bool{"actor-123": true}
	violations := verdictRuling(r, sliceIDs, []string{})
	if len(violations) == 0 {
		t.Fatalf("expected a descriptor violation for a create with no descriptor, got none")
	}
	found := false
	for _, v := range violations {
		if strings.Contains(v, "descriptor") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("violation does not mention the missing descriptor: %v", violations)
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
