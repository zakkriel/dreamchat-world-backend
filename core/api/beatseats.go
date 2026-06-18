package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// PerceptionPayload is the ONLY world input either model seat receives (B-1/I-3; §14). It is built
// upstream from fn_visible_perceptions — there is deliberately no field that can carry raw canon.
type PerceptionPayload struct {
	Lines []string `json:"lines"` // perception-bound, epistemically framed lines for the holder
}

// BeatStep is one element of the closed-vocabulary chain (beat_chain/1).
type BeatStep struct {
	Type     string `json:"type"`               // "move" | "say" — the closed set
	To       string `json:"to,omitempty"`       // move
	Listener string `json:"listener,omitempty"` // say (uuid)
	Content  string `json:"content,omitempty"`  // say
}

// Decomposer: prose → proposed chain. PROPOSES ONLY (D-1). Perception-bound input (§14).
type Decomposer interface {
	Decompose(ctx context.Context, payload PerceptionPayload, text string) string // returns raw JSON
}

// Narrator: post-beat perception payload → prose. Perception-bound (ADR-020, no omniscient pass).
// Output is presentation, NOT canon — never written to canon_event (I-6).
type Narrator interface {
	Narrate(ctx context.Context, payload PerceptionPayload) string
}

// DecodeAndValidateChain enforces the leash: valid JSON AND every step's type ∈ the closed set
// (SPEC-015/D-1). Anything else is rejected — the model cannot widen the vocabulary.
func DecodeAndValidateChain(raw string) ([]BeatStep, error) {
	var chain []BeatStep
	if err := json.Unmarshal([]byte(raw), &chain); err != nil {
		return nil, fmt.Errorf("decompose output is not valid chain JSON: %w", err)
	}
	for i, s := range chain {
		if !allowedBeatTypes[s.Type] {
			return nil, fmt.Errorf("step %d: type %q is outside the closed vocabulary {move,say}", i, s.Type)
		}
		if s.Type == "move" && s.To == "" {
			return nil, fmt.Errorf("step %d: move requires 'to'", i)
		}
		if s.Type == "say" && (s.Listener == "" || s.Content == "") {
			return nil, fmt.Errorf("step %d: say requires 'listener' and 'content'", i)
		}
	}
	return chain, nil
}

// --- deterministic fakes for CI (the live model is wired only at the operator gate) ---

type fakeDecomposer struct{ table map[string]string }

func NewFakeDecomposer(table map[string]string) Decomposer { return &fakeDecomposer{table} }

func (f *fakeDecomposer) Decompose(_ context.Context, _ PerceptionPayload, text string) string {
	if out, ok := f.table[text]; ok {
		return out
	}
	return "[]" // unknown prose → empty chain (a beat that commits nothing; C-5)
}

type fakeNarrator struct{ prefix string }

func NewFakeNarrator(prefix string) Narrator { return &fakeNarrator{prefix} }

func (f *fakeNarrator) Narrate(_ context.Context, p PerceptionPayload) string {
	out := f.prefix
	for _, l := range p.Lines {
		out += " " + l
	}
	return out
}
