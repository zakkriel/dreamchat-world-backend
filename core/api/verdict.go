package main

import (
	"encoding/json"
	"strconv"
	"strings"
)

// jsonTypeOf returns the JSON type of a raw message as a string:
// "string", "number", "boolean", "array", "object", or "null"
func jsonTypeOf(v json.RawMessage) string {
	if v == nil {
		return "null"
	}

	// Trim leading whitespace to find the first meaningful character
	trimmed := strings.TrimSpace(string(v))
	if len(trimmed) == 0 {
		return "null"
	}

	switch trimmed[0] {
	case '"':
		return "string"
	case 't', 'f':
		// true or false
		return "boolean"
	case '[':
		return "array"
	case '{':
		return "object"
	case 'n':
		// null
		return "null"
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return "number"
	default:
		// Default to unknown/null for safety
		return "null"
	}
}

// verdictRuling checks a RulingV2 for violations:
//  1. Every UUID referenced anywhere in the ruling (event actor_id, all typed slots, attr write target_ids, variant receiver_ids)
//     must be in sliceIDs or attemptIDs (the id-whitelist).
//  2. Each AttrWrite with Tier==1: name must exist in tier1Registry AND jsonTypeOf(Value) must match the registry type.
//  3. Each AttrWrite with Tier==2: name must NOT exist in tier1Registry (the wall violation).
//
// Returns a slice of violation strings (empty slice = pass).
func verdictRuling(r RulingV2, sliceIDs map[string]bool, attemptIDs []string) []string {
	var violations []string

	// Build a map of valid attempt IDs for faster lookup
	attemptIDsMap := make(map[string]bool)
	for _, id := range attemptIDs {
		attemptIDsMap[id] = true
	}

	// created holds the ids admitted by a TRUE-INTRODUCTION EntityCreated EARLIER in this ruling: the ONE
	// place the id-whitelist admits a not-in-slice id (§8). Grown in event order so a SUBSEQUENT event may
	// reference the just-created id (e.g. forge the dagger, then relocate it) without a whitelist violation.
	created := make(map[string]bool)

	// Helper to check if a UUID is in the whitelist
	isWhitelisted := func(uuid string) bool {
		if uuid == "" {
			return true // Non-uuid empty fields are allowed
		}
		return sliceIDs[uuid] || attemptIDsMap[uuid] || created[uuid]
	}

	// Check all UUIDs in events
	for i, event := range r.Outcome.Events {
		// Check actor_id (the creator/mover/speaker — always whitelist-gated, even for a create).
		if !isWhitelisted(event.ActorID) {
			violations = append(violations, formatUUIDViolation("event", i, "actor_id", event.ActorID))
		}

		// ── EntityCreated: the create-write, not a reference (Station F Task 9 / §8) ──────────────
		// target_id is the NEW entity's id. The verdict admits it — the ONE whitelist opening — but ONLY
		// as a TRUE INTRODUCTION: a descriptor is MANDATORY (else it is not an introduction and the id
		// stays un-admitted, keeping the opening from becoming a hole). Reuse-before-create (does this
		// descriptor match an existing entity?) is the SQL commit's job — it holds the authoritative
		// registry (R3 "match against slice + registry"); the verdict, a pure function, admits the SHAPE.
		// The admitted id joins `created` so later events in THIS ruling may reference it.
		if event.Type == "EntityCreated" {
			if strings.TrimSpace(event.Descriptor) == "" {
				violations = append(violations, formatAttrViolation("event", i, "descriptor", "\"\"", "EntityCreated requires a descriptor (true introduction, §8)"))
			} else if event.TargetID != "" {
				created[event.TargetID] = true // admitted as a true introduction
			}
			// A create references nothing else through typed id slots; receiver variants (if any) are
			// still perceivers and must be whitelist-gated.
			for _, variant := range event.ReceiverVariants {
				if variant.ReceiverID != "" && !isWhitelisted(variant.ReceiverID) {
					violations = append(violations, formatUUIDViolation("event", i, "receiver_id", variant.ReceiverID))
				}
			}
			continue
		}

		// Check typed slots
		if event.ToTargetID != "" && !isWhitelisted(event.ToTargetID) {
			violations = append(violations, formatUUIDViolation("event", i, "to_target_id", event.ToTargetID))
		}
		if event.ListenerID != "" && !isWhitelisted(event.ListenerID) {
			violations = append(violations, formatUUIDViolation("event", i, "listener_id", event.ListenerID))
		}
		if event.ObjectID != "" && !isWhitelisted(event.ObjectID) {
			violations = append(violations, formatUUIDViolation("event", i, "object_id", event.ObjectID))
		}
		if event.DestID != "" && !isWhitelisted(event.DestID) {
			violations = append(violations, formatUUIDViolation("event", i, "dest_id", event.DestID))
		}
		if event.TargetID != "" && !isWhitelisted(event.TargetID) {
			violations = append(violations, formatUUIDViolation("event", i, "target_id", event.TargetID))
		}
		if event.GranteeID != "" && !isWhitelisted(event.GranteeID) {
			violations = append(violations, formatUUIDViolation("event", i, "grantee_id", event.GranteeID))
		}

		// Check receiver variants
		for _, variant := range event.ReceiverVariants {
			if variant.ReceiverID != "" && !isWhitelisted(variant.ReceiverID) {
				violations = append(violations, formatUUIDViolation("event", i, "receiver_id", variant.ReceiverID))
			}
		}
	}

	// Check attribute writes
	for i, write := range r.Outcome.AttributeWrites {
		// Check TargetID
		if !isWhitelisted(write.TargetID) {
			violations = append(violations, formatUUIDViolation("attribute_write", i, "target_id", write.TargetID))
		}

		if write.Tier == 1 {
			// Tier-1: name must exist in registry AND type must match
			expectedType, exists := tier1Registry[write.Attribute]
			if !exists {
				violations = append(violations, formatAttrViolation("attribute_write", i, "attribute", write.Attribute, "not in tier1Registry"))
			} else {
				actualType := jsonTypeOf(write.Value)
				if actualType != expectedType {
					violations = append(violations, formatAttrViolation("attribute_write", i, write.Attribute, actualType, "expected "+expectedType))
				}
			}
		} else if write.Tier == 2 {
			// Tier-2: name must NOT exist in tier1Registry (wall violation)
			if _, exists := tier1Registry[write.Attribute]; exists {
				violations = append(violations, formatWallViolation("attribute_write", i, write.Attribute))
			}
		}
	}

	return violations
}

// verdictMints is the verdict-stage entry point for minting (FINAL-action-contracts.md §8): it hands the
// ruling's proposed mints to validateMints (mint.go) and returns violations in the SAME shape verdictRuling
// uses — so mint failures flow into the existing repair-once-then-bounce in adjudicate, no separate path.
// It lives beside verdictRuling because minting shares the verdict stage; it takes existingMovementTypes
// (the world's committed movement vocabulary) because mint-ordering (§8) is a DB-grounded check, which is
// why the caller (adjudicate, which has the pool) fetches the set and passes it in rather than verdictRuling
// (a pure function) growing a DB dependency.
func verdictMints(r RulingV2, existingMovementTypes map[string]bool) []string {
	return validateMints(r.Outcome.Mints, existingMovementTypes)
}

// formatUUIDViolation creates a violation message for an invalid UUID reference
func formatUUIDViolation(kind string, index int, field string, uuid string) string {
	return kind + " " + strconv.Itoa(index) + ": " + field + " " + uuid + " not in the gathered slice"
}

// formatAttrViolation creates a violation message for an attribute error
func formatAttrViolation(kind string, index int, attrName string, actualValue string, expected string) string {
	return kind + " " + strconv.Itoa(index) + ": \"" + attrName + "\" " + actualValue + " " + expected
}

// formatWallViolation creates a violation message for a tier-2 writing a tier-1 attribute
func formatWallViolation(kind string, index int, attrName string) string {
	return kind + " " + strconv.Itoa(index) + ": \"" + attrName + "\" is a Tier-1 engine attribute — writing it as tier 2 is the wall violation (write both: tier 1 for mechanics, tier 2 for meaning)"
}
