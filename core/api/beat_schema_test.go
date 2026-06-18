package main

import (
	"encoding/json"
	"os"
	"testing"
)

// The published schema is the leash (ADR-009/D-1, SPEC-015): the decomposer may emit ONLY the
// closed thin-slice vocabulary {move, say}. This test pins that the schema and the runtime
// validator's vocabulary agree.
func TestBeatChainSchema_ClosedVocabulary(t *testing.T) {
	b, err := os.ReadFile("schema/beat_chain.v1.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var s map[string]any
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	if s["schema_version"] != "beat_chain/1" {
		t.Fatalf("schema_version = %v, want beat_chain/1", s["schema_version"])
	}
	types := vocabularyTypes(s)
	if len(types) != 2 || !types["move"] || !types["say"] {
		t.Fatalf("vocabulary = %v, want exactly {move, say}", types)
	}
	// the runtime validator's closed set must match the published schema (no drift)
	for tname := range types {
		if !allowedBeatTypes[tname] {
			t.Fatalf("schema type %q not in allowedBeatTypes (runtime/schema drift)", tname)
		}
	}
	if len(allowedBeatTypes) != len(types) {
		t.Fatalf("allowedBeatTypes %v != schema vocabulary %v", allowedBeatTypes, types)
	}
}
