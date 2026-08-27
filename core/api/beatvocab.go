package main

// Governed-by: ADR-009 — the structured-output leash. Also D-1, SPEC-015.
// Promoted from this file's own citations (2026-08-26), not newly decided. Change what this
// file decides and those decisions change with it (D-9).

import (
	_ "embed"
	"encoding/json"
)

//go:embed schema/beat_chain.v2.schema.json
var beatChainV2SchemaJSON string

// vocabularyTypes extracts the closed set of allowed event "type" const values from the
// beat_chain schema's items.oneOf. The closed set IS the leash (ADR-009/D-1, SPEC-015).
func vocabularyTypes(schema map[string]any) map[string]bool {
	out := map[string]bool{}
	items, _ := schema["items"].(map[string]any)
	if items == nil {
		return out
	}
	oneOf, _ := items["oneOf"].([]any)
	for _, alt := range oneOf {
		m, _ := alt.(map[string]any)
		props, _ := m["properties"].(map[string]any)
		typ, _ := props["type"].(map[string]any)
		if c, ok := typ["const"].(string); ok {
			out[c] = true
		}
	}
	return out
}

// allowedBeatTypes is the canonical closed set used by the runtime validator (kept in sync with
// beat_chain.v1.schema.json; TestBeatChainSchema_ClosedVocabulary asserts they match).
var allowedBeatTypes = map[string]bool{"move": true, "say": true}

// vocabularyTypesV2 extracts the closed set of allowed event "type" const values from the
// beat_chain/2 schema's items.oneOf (six types + UNRESOLVED + QUERY).
func vocabularyTypesV2() map[string]bool {
	out := map[string]bool{}
	var schema map[string]any
	if err := json.Unmarshal([]byte(beatChainV2SchemaJSON), &schema); err != nil {
		return out
	}
	items, _ := schema["items"].(map[string]any)
	if items == nil {
		return out
	}
	oneOf, _ := items["oneOf"].([]any)
	for _, alt := range oneOf {
		m, _ := alt.(map[string]any)
		props, _ := m["properties"].(map[string]any)
		typ, _ := props["type"].(map[string]any)
		if c, ok := typ["const"].(string); ok {
			out[c] = true
		}
	}
	return out
}

// allowedBeatTypesV2 is the canonical closed set for beat_chain/2 (kept in sync with
// beat_chain.v2.schema.json; TestVocabularyV2IsTheSixTypesPlusUnresolvedAndQuery asserts they match).
var allowedBeatTypesV2 = map[string]bool{
	"ActorMoved":             true,
	"Communicated":           true,
	"ObjectRelocated":        true,
	"OwnershipAccessChanged": true,
	"EntityCreated":          true,
	"EntityDestroyed":        true,
	"AttributeChanged":       true,
	"UNRESOLVED":             true,
	"QUERY":                  true,
}
