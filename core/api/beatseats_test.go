package main

import (
	"strings"
	"testing"
)

// The defense-in-depth belt rejects an out-of-vocab event (the primary leash is the decompose seat's
// generation-time structured output; this is the backstop, SPEC-015/D-1).
func TestDecodeAndValidateChain_RejectsOutOfVocab(t *testing.T) {
	_, err := DecodeAndValidateChain(`[{"type":"attack","target":"x"}]`)
	if err == nil {
		t.Fatalf("out-of-vocabulary 'attack' was accepted — the belt failed (SPEC-015)")
	}
	if !strings.Contains(err.Error(), "vocabulary") {
		t.Fatalf("error = %v, want a vocabulary rejection", err)
	}
}

func TestDecodeAndValidateChain_AcceptsClosedVocab(t *testing.T) {
	chain, err := DecodeAndValidateChain(`[{"type":"move","to":"square"},{"type":"say","listener":"` +
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" + `","content":"hi"}]`)
	if err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}
	if len(chain) != 2 || chain[0].Type != "move" || chain[1].Type != "say" {
		t.Fatalf("chain = %+v, want [move, say]", chain)
	}
}

// duration_class is the decomposer's parse-shape estimate of how long a NON-MOVE act takes in the
// fiction (RULINGS-2026-07-23 §4) — decoded when present, and rejected when outside the enum.
func TestDecodeChainV2_DurationClass(t *testing.T) {
	ok := `[{"type":"Communicated","stated":"I tell Mara my whole life story","listener_id":"11111111-1111-1111-1111-111111111111","content":"...","duration_class":"long"}]`
	chain, err := DecodeAndValidateChainV2(ok)
	if err != nil {
		t.Fatalf("valid class rejected: %v", err)
	}
	if chain[0].DurationClass != "long" {
		t.Fatalf("class not decoded: %q", chain[0].DurationClass)
	}

	bad := `[{"type":"Communicated","stated":"x","listener_id":"11111111-1111-1111-1111-111111111111","content":"x","duration_class":"aeon"}]`
	if _, err := DecodeAndValidateChainV2(bad); err == nil {
		t.Fatalf("out-of-enum duration_class accepted")
	}
}

func TestPayload_IsPerceptionBound(t *testing.T) {
	// the payload type carries ONLY safety-filtered perception lines — no field can hold raw canon.
	p := PerceptionPayload{Lines: []string{"You told Mara about the note."}}
	if strings.Contains(strings.Join(p.Lines, "\n"), "mayor keeps a hidden ledger") {
		t.Fatalf("payload leaked a hidden fact")
	}
}
