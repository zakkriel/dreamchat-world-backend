package main

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
