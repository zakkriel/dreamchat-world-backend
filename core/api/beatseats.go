package main

// Governed-by: D-1 — nothing mutates canon directly — proposals only, the Core commits. Also B-1, I-3, SPEC-015.
// Promoted from this file's own citations (2026-08-26), not newly decided. Change what this
// file decides and those decisions change with it (D-9).

import (
	"encoding/json"
	"fmt"
)

// PerceptionPayload is the ONLY world input either model seat receives (B-1/I-3; §14). It is built
// upstream from fn_visible_perceptions — there is deliberately no field that can carry raw canon.
type PerceptionPayload struct {
	Lines      []string    `json:"lines"`                // perception-bound, epistemically framed lines for the holder
	LineIDs    []string    `json:"line_ids,omitempty"`   // perception_id parallel to Lines (delta-first narration; no external API change — these are ids the holder already perceives, never raw canon)
	Candidates []Candidate `json:"candidates,omitempty"` // entity whitelist for v2 decompose (beat_chain/2)
	// Here is the id of the room the viewer is STANDING IN — stated outright rather than inferred.
	// Callers used to recover it by scanning Candidates for the single entry of kind "location",
	// which was only ever correct while exactly one location could be a candidate. SPEC-030 added the
	// rooms this one connects to, so that scan silently began returning a neighbouring room; the scene
	// endpoint reported the wrong place the moment the player walked through a door. The room you are
	// in is a fact the assembler already knows, so it says so. Empty only when the viewer has no
	// location_id at all. Not serialised to the seats: it is already the "location" candidate they see.
	Here string `json:"-"`
	// ViewerAliases is how OTHERS may refer to the VIEWER inside perception text — the viewer's own
	// descriptor (what a stranger sees, e.g. "a young stranger, dark-haired") plus the viewer's OWN
	// name-knowledge of himself, when he holds one. It is NEVER the registry canonical name he may not
	// know (§3 naming reach — leaking it would breach the naming wall). The narrate YOU ARE block
	// renders it so a third-person reference to the viewer resolves to "you". Empty ⇒ no YOU ARE block.
	ViewerAliases []string `json:"viewer_aliases,omitempty"`
	// World is this world's GLOBAL statement — its name, its authored premise, its region and its
	// minted register words. Every field is the committed genesis document's own content (see
	// WorldStatement); world.brief is structurally excluded, and that exclusion is tested.
	//
	// Not serialised to the seats (json:"-"), for the same reason Here is not: it is prompt material
	// the narrate builder renders as THE WORLD, not a fact the seats decompose over. Keeping it out of
	// the JSON also keeps the seat payload contract and its generated fixtures untouched.
	//
	// This carries no raw canon and breaches no wall: it is world-level authorship visible to anyone
	// standing in the world, holder-independent by construction, with nothing per-entity in it.
	World WorldStatement `json:"-"`
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
// Sustain is the "until/for <condition>" parse-shape (design §4.4) — a SHAPE the decomposer recognises,
// like QUERY, never a judgment about how long something ought to take. Exactly one kind:
//
//	{"kind":"for",        "seconds":7200}                              — a span the player STATED (R13)
//	{"kind":"until_at",   "entity_id":…, "place_id":…}                 — that thing, at that place
//	{"kind":"until_attr", "entity_id":…, "attr":"open", "value":"true"} — that thing, in that state
//
// seconds is passed through, NOT classified: reading "two hours" back as 7200 is parsing, the same act as
// binding a name to an id. duration_class stays what it is — the ladder for acts with an inherent length,
// whose cap exists so an UNSTATED length is never invented.
type Sustain struct {
	Kind     string `json:"kind"`
	Seconds  int64  `json:"seconds,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
	PlaceID  string `json:"place_id,omitempty"`
	Attr     string `json:"attr,omitempty"`
	Value    string `json:"value,omitempty"`
}

type Attempt struct {
	Type         string   `json:"type"`
	Stated       string   `json:"stated"`
	ToTargetID   string   `json:"to_target_id,omitempty"` // ActorMoved: id of ANY positioned entity (location|object|actor)
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
	// QueryTargetIDs binds a QUERY's asked-about entities (Grounded Reasoning Unit 2). QUERY is
	// NOT an action — it carries no outcome and binds no other slot; it is a parse shape, a
	// sibling of UNRESOLVED, recognizing interrogative form and binding ids from the candidate
	// whitelist (RULINGS-2026-07-23 §4; RULINGS-2026-07-30 §1).
	QueryTargetIDs []string `json:"query_target_ids,omitempty"`
	// DurationClass is the decomposer's parse-shape estimate of how long a NON-MOVE act takes in the
	// fiction — one of instant|short|medium|long|extremely_long (a validated enum, like a QUERY shape,
	// NOT a raw number and NOT the banned outcome/tension/intent judgment; RULINGS-2026-07-23 §4). The
	// engine maps class→seconds per world (fn_duration_class_seconds). Empty on ActorMoved (physics owns
	// move duration) and on legacy input.
	DurationClass string `json:"duration_class,omitempty"`
	// Sustain is the "until/for <condition>" parse-shape (design §4.4, R13) carried by non-move
	// attempts only — a move's length is physics, never a stated span. See type Sustain.
	Sustain *Sustain `json:"sustain,omitempty"`
}

// DecodeAndValidateChainV2 is the belt behind the leash: valid JSON, every
// type in the closed six-type set (+UNRESOLVED, +QUERY), per-type required fields.
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
		if a.ToTargetID == "" {
			return fmt.Errorf("step %d ActorMoved requires to_target_id", i)
		}
	case "Communicated":
		if a.ListenerID == "" || a.Content == "" {
			return fmt.Errorf("step %d Communicated requires listener_id+content", i)
		}
	case "ObjectRelocated":
		if a.ObjectID == "" || a.DestKind == "" || a.DestID == "" {
			return fmt.Errorf("step %d ObjectRelocated requires object_id+dest_kind+dest_id", i)
		}
		switch a.DestKind {
		case "actor", "location", "container":
		default:
			return fmt.Errorf("step %d ObjectRelocated dest_kind %q not in actor|location|container", i, a.DestKind)
		}
	case "OwnershipAccessChanged", "EntityDestroyed", "AttributeChanged":
		if a.TargetID == "" {
			return fmt.Errorf("step %d %s requires target_id", i, a.Type)
		}
	case "UNRESOLVED":
		if a.Reference == "" || len(a.CandidateIDs) < 2 {
			return fmt.Errorf("step %d UNRESOLVED requires reference + >=2 candidates", i)
		}
	case "QUERY":
		if len(a.QueryTargetIDs) < 1 {
			return fmt.Errorf("step %d QUERY requires >=1 query_target_ids", i)
		}
	}
	if a.DurationClass != "" {
		switch a.DurationClass {
		case "instant", "short", "medium", "long", "extremely_long":
		default:
			return fmt.Errorf("step %d duration_class %q outside enum", i, a.DurationClass)
		}
	}
	if a.Sustain != nil {
		if a.Type == "ActorMoved" {
			return fmt.Errorf("step %d ActorMoved cannot carry sustain — its length is physics", i)
		}
		switch a.Sustain.Kind {
		case "for":
			if a.Sustain.Seconds <= 0 {
				return fmt.Errorf("step %d sustain kind \"for\" requires seconds > 0", i)
			}
		case "until_at":
			if a.Sustain.EntityID == "" || a.Sustain.PlaceID == "" {
				return fmt.Errorf("step %d sustain kind \"until_at\" requires entity_id+place_id", i)
			}
		case "until_attr":
			if a.Sustain.EntityID == "" || a.Sustain.Attr == "" || a.Sustain.Value == "" {
				return fmt.Errorf("step %d sustain kind \"until_attr\" requires entity_id+attr+value", i)
			}
		default:
			return fmt.Errorf("step %d sustain kind %q outside for|until_at|until_attr", i, a.Sustain.Kind)
		}
	}
	return nil
}
