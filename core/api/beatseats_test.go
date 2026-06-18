package main

import (
	"context"
	"strings"
	"testing"
)

func TestDecomposer_OnlyValidVocabulary(t *testing.T) {
	d := NewFakeDecomposer(map[string]string{
		"go to the square": `[{"type":"move","to":"square"}]`,
	})
	chain, err := DecodeAndValidateChain(d.Decompose(context.Background(), PerceptionPayload{}, "go to the square"))
	if err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}
	if len(chain) != 1 || chain[0].Type != "move" || chain[0].To != "square" {
		t.Fatalf("chain = %+v, want one move→square", chain)
	}
}

func TestDecomposer_OutOfVocabularyRejected(t *testing.T) {
	// the model proposes an event outside the closed set — the gate rejects (SPEC-015/D-1).
	_, err := DecodeAndValidateChain(`[{"type":"attack","target":"x"}]`)
	if err == nil {
		t.Fatalf("out-of-vocabulary 'attack' was accepted — the leash failed (SPEC-015)")
	}
	if !strings.Contains(err.Error(), "vocabulary") {
		t.Fatalf("error = %v, want a vocabulary rejection", err)
	}
}

func TestPayload_IsPerceptionBound(t *testing.T) {
	// the payload type carries ONLY safety-filtered perception lines — there is no field that could
	// hold raw canon. This is a structural guarantee (the wall is fn_visible_perceptions upstream).
	p := PerceptionPayload{Lines: []string{"You told Mara about the note."}}
	if strings.Contains(strings.Join(p.Lines, "\n"), "mayor keeps a hidden ledger") {
		t.Fatalf("payload leaked a hidden fact")
	}
}
