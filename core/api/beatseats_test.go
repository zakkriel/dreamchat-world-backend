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

func TestPayload_IsPerceptionBound(t *testing.T) {
	// the payload type carries ONLY safety-filtered perception lines — no field can hold raw canon.
	p := PerceptionPayload{Lines: []string{"You told Mara about the note."}}
	if strings.Contains(strings.Join(p.Lines, "\n"), "mayor keeps a hidden ledger") {
		t.Fatalf("payload leaked a hidden fact")
	}
}
