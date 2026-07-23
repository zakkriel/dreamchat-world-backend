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

// Attempt is one element of beat_chain/2 — an ATTEMPT with ids, never an
// outcome. Example: {"type":"AttributeChanged","stated":"I open the door",
// "target_id":"<door uuid>"} — what happens to the door is resolve's job.
type Attempt struct {
	Type         string   `json:"type"`
	Stated       string   `json:"stated"`
	ToLocationID string   `json:"to_location_id,omitempty"`
	ListenerID   string   `json:"listener_id,omitempty"`
	Content      string   `json:"content,omitempty"`
	ObjectID     string   `json:"object_id,omitempty"`
	DestKind     string   `json:"dest_kind,omitempty"`
	DestID       string   `json:"dest_id,omitempty"`
	TargetID     string   `json:"target_id,omitempty"`
	GranteeID    string   `json:"grantee_id,omitempty"`
	ComponentIDs []string `json:"component_ids,omitempty"`
	Reference    string   `json:"reference,omitempty"`
	CandidateIDs []string `json:"candidate_ids,omitempty"`
}

// DecodeAndValidateChainV2 is the belt behind the leash: valid JSON, every
// type in the closed six-type set (+UNRESOLVED), per-type required fields.
func DecodeAndValidateChainV2(raw string) ([]Attempt, error) {
	var chain []Attempt
	if err := json.Unmarshal([]byte(raw), &chain); err != nil {
		return nil, fmt.Errorf("chain not valid JSON: %w", err)
	}
	for i, a := range chain {
		if !allowedBeatTypesV2[a.Type] {
			return nil, fmt.Errorf("step %d type %q outside closed vocabulary", i, a.Type)
		}
		if a.Stated == "" {
			return nil, fmt.Errorf("step %d missing stated", i)
		}
		if err := validateAttemptFields(i, a); err != nil {
			return nil, err
		}
	}
	return chain, nil
}

// validateAttemptFields holds the per-type required-field checks; shared with
// the cognition validator (Task 3) so NPC attempts obey the same shapes.
func validateAttemptFields(i int, a Attempt) error {
	switch a.Type {
	case "ActorMoved":
		if a.ToLocationID == "" {
			return fmt.Errorf("step %d ActorMoved requires to_location_id", i)
		}
	case "Communicated":
		if a.ListenerID == "" || a.Content == "" {
			return fmt.Errorf("step %d Communicated requires listener_id+content", i)
		}
	case "ObjectRelocated":
		if a.ObjectID == "" || a.DestKind == "" || a.DestID == "" {
			return fmt.Errorf("step %d ObjectRelocated requires object_id+dest_kind+dest_id", i)
		}
	case "OwnershipAccessChanged", "EntityDestroyed", "AttributeChanged":
		if a.TargetID == "" {
			return fmt.Errorf("step %d %s requires target_id", i, a.Type)
		}
	case "UNRESOLVED":
		if a.Reference == "" || len(a.CandidateIDs) < 2 {
			return fmt.Errorf("step %d UNRESOLVED requires reference + >=2 candidates", i)
		}
	}
	return nil
}
