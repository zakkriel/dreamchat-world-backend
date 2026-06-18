package main

import (
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

// DecodeAndValidateChain is the DEFENSE-IN-DEPTH belt behind the generation-time leash: the primary
// enforcement is structured output at the decompose seat (schema-valid by construction), but the
// handler still re-validates the decoded chain against the closed vocabulary (SPEC-015/D-1) before it
// reaches apply_beat. A correctly-bound structured driver never trips this; a rogue/misbound one does.
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
