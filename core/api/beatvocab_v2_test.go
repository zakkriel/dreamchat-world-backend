package main

import "testing"

func TestVocabularyV2IsTheSixTypesPlusUnresolved(t *testing.T) {
	want := []string{"ActorMoved", "Communicated", "ObjectRelocated",
		"OwnershipAccessChanged", "EntityCreated", "EntityDestroyed",
		"AttributeChanged", "UNRESOLVED"}
	got := vocabularyTypesV2()
	if len(got) != len(want) {
		t.Fatalf("vocabulary = %v, want %v", got, want)
	}
	for _, w := range want {
		if !allowedBeatTypesV2[w] {
			t.Fatalf("missing type %q", w)
		}
	}
}

func TestDecodeV2RejectsOutcomeShapedAndAcceptsAttempts(t *testing.T) {
	// A valid three-attempt chain (the Drowned Lantern opening).
	ok := `[{"type":"ActorMoved","stated":"I cross to the bar","to_location_id":"11111111-1111-1111-1111-111111111111"},
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
}
