package main

import "testing"

func TestVocabularyV2IsTheSixTypesPlusUnresolvedAndQuery(t *testing.T) {
	// QUERY joined the set (Task 4, Grounded Reasoning Unit 2): a bound, read-only
	// question is a parse shape, not an attempt — a sibling of UNRESOLVED.
	want := []string{"ActorMoved", "Communicated", "ObjectRelocated",
		"OwnershipAccessChanged", "EntityCreated", "EntityDestroyed",
		"AttributeChanged", "UNRESOLVED", "QUERY"}
	got := vocabularyTypesV2()
	if len(got) != len(want) {
		t.Fatalf("vocabulary = %v, want %v", got, want)
	}
	for _, w := range want {
		if !allowedBeatTypesV2[w] {
			t.Fatalf("missing type %q in allowedBeatTypesV2", w)
		}
		if !got[w] {
			t.Fatalf("missing type %q in parsed schema", w)
		}
	}
}

func TestDecodeV2RejectsOutcomeShapedAndAcceptsAttempts(t *testing.T) {
	// A valid three-attempt chain (the Drowned Lantern opening).
	ok := `[{"type":"ActorMoved","stated":"I cross to the bar","to_target_id":"11111111-1111-1111-1111-111111111111"},
	 {"type":"ObjectRelocated","stated":"slip her the note","object_id":"22222222-2222-2222-2222-222222222222","dest_kind":"actor","dest_id":"33333333-3333-3333-3333-333333333333"},
	 {"type":"Communicated","stated":"ask about the harbormaster","listener_id":"33333333-3333-3333-3333-333333333333","content":"what do you know of the harbormaster?"}]`
	if _, err := DecodeAndValidateChainV2(ok); err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}
	// Old v1 shape must fail (no silent legacy acceptance).
	old := `[{"type":"move","to":"bar"}]`
	if _, err := DecodeAndValidateChainV2(old); err == nil {
		t.Fatal("v1 'move' accepted by v2 validator")
	}
	// UNRESOLVED is a first-class emission (a rejected proposal beats a guessed canon).
	unres := `[{"type":"UNRESOLVED","stated":"give the note to her","reference":"her","candidate_ids":["33333333-3333-3333-3333-333333333333","44444444-4444-4444-4444-444444444444"]}]`
	if _, err := DecodeAndValidateChainV2(unres); err != nil {
		t.Fatalf("UNRESOLVED rejected: %v", err)
	}
	// ObjectRelocated must reject invalid dest_kind values.
	invalidDestKind := `[{"type":"ObjectRelocated","stated":"slip her the note","object_id":"22222222-2222-2222-2222-222222222222","dest_kind":"basket","dest_id":"33333333-3333-3333-3333-333333333333"}]`
	if _, err := DecodeAndValidateChainV2(invalidDestKind); err == nil {
		t.Fatal("ObjectRelocated with invalid dest_kind 'basket' accepted")
	}
}

// A question is not an action (RULINGS-2026-07-23 §3: the player gets a reaction, never a record).
// QUERY binds the asked entity's id from the CANDIDATES and carries no outcome (Task 4, Unit 2).
func TestDecodeV2AcceptsQuery(t *testing.T) {
	query := `[{"type":"QUERY","stated":"how long to the bar?","query_target_ids":["11111111-1111-1111-1111-111111111111"]}]`
	chain, err := DecodeAndValidateChainV2(query)
	if err != nil {
		t.Fatalf("valid QUERY rejected: %v", err)
	}
	if len(chain) != 1 || chain[0].Type != "QUERY" ||
		len(chain[0].QueryTargetIDs) != 1 ||
		chain[0].QueryTargetIDs[0] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("chain = %+v, want one QUERY bound to the bar", chain)
	}
}

func TestDecodeV2RejectsQueryWithoutTargetIDs(t *testing.T) {
	missing := `[{"type":"QUERY","stated":"is the door locked?"}]`
	if _, err := DecodeAndValidateChainV2(missing); err == nil {
		t.Fatal("QUERY with missing query_target_ids accepted")
	}
	empty := `[{"type":"QUERY","stated":"is the door locked?","query_target_ids":[]}]`
	if _, err := DecodeAndValidateChainV2(empty); err == nil {
		t.Fatal("QUERY with empty query_target_ids accepted")
	}
}

func TestDecodeV2MixedChainResolvesActionAndQuery(t *testing.T) {
	// "I walk over — how long to slip her the note?" (design doc Unit 2, mixed input).
	mixed := `[{"type":"ActorMoved","stated":"I walk over","to_target_id":"11111111-1111-1111-1111-111111111111"},
	 {"type":"QUERY","stated":"how long to slip her the note?","query_target_ids":["22222222-2222-2222-2222-222222222222","33333333-3333-3333-3333-333333333333"]}]`
	chain, err := DecodeAndValidateChainV2(mixed)
	if err != nil {
		t.Fatalf("mixed ActorMoved+QUERY chain rejected: %v", err)
	}
	if len(chain) != 2 || chain[0].Type != "ActorMoved" || chain[1].Type != "QUERY" {
		t.Fatalf("chain = %+v, want [ActorMoved, QUERY]", chain)
	}
	if len(chain[1].QueryTargetIDs) != 2 {
		t.Fatalf("QUERY target ids = %v, want 2 bound ids", chain[1].QueryTargetIDs)
	}
}
