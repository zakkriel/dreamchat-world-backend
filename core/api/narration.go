package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// narrationV1SchemaJSON is the structured-output leash handed to the narrate seat (founder envelope):
// the beat response stops being one narrator blob and becomes an ORDERED list of typed segments —
// narrator prose plus attributed NPC speech and single-NPC actions. Registered as an input-contract
// schema (ci/schema_contract.py, SPEC-011 house rule) — it is the seat's leash, not a projection
// payload, so it has no SQL payload generator by design.
//
//go:embed schema/narration.v1.schema.json
var narrationV1SchemaJSON string

// NarrationSegment is one element of narration/1: narrator prose (SpeakerID nil, Kind "narration") or an
// attributed NPC line (SpeakerID non-nil, Kind "speech" for the mind's exact words, "action" for one
// clean single-NPC act). SpeakerID is a *string so the null⇔narration correlation is decodable (an
// absent/explicit-null speaker is distinct from the empty string).
type NarrationSegment struct {
	SpeakerID *string `json:"speaker_id"`
	Kind      string  `json:"kind"`
	Text      string  `json:"text"`
}

// DecodeAndValidateNarration is the DEFENSE-IN-DEPTH belt behind the narrate seat's structured-output
// leash (schema/narration.v1.schema.json). Structured decoding makes the output schema-valid by
// construction; this belt re-enforces the same contract AND adds two engine-grounded checks the schema
// alone cannot express:
//
//   - GHOST SPEAKER: every non-null speaker_id must be a present entity this beat (presentIDs — the
//     narrate PRESENT roster the model saw). A segment attributed to someone not in the room is rejected.
//   - VERBATIM SPEECH: a kind=speech segment's text must appear (exact or substring) within one of that
//     speaker's speech perception contents this beat (speechTexts: speaker id → the perception contents
//     the viewer holds from that speaker's Communicated events this beat). NPC speech must be the mind's
//     exact words — never narrator paraphrase. See the extraction note in beatHandler.speechTexts: the
//     perception content is the resolve/attempt-authored line for a Communicated event (COALESCE of
//     receiver-variant, appearance, then truth), so the words may ride inside scaffolding; a SUBSTRING
//     match on the quoted words is therefore the honest belt (a paraphrase is not a substring, and is
//     rejected). action segments carry no verbatim requirement (a viewer-relative act, not spoken words).
//
// Plus the schema's structural rules, restated so a rogue/misbound driver still trips them: kind in the
// closed set, text non-empty, and the speaker_id↔kind correlation (null ⇔ narration; non-null ⇔
// speech|action).
func DecodeAndValidateNarration(raw string, presentIDs []string, speechTexts map[string][]string) ([]NarrationSegment, error) {
	var segs []NarrationSegment
	if err := json.Unmarshal([]byte(raw), &segs); err != nil {
		return nil, fmt.Errorf("narration/1 not valid JSON: %w", err)
	}
	present := make(map[string]bool, len(presentIDs))
	for _, id := range presentIDs {
		present[id] = true
	}
	// A BLANK SEGMENT IS DROPPED, not fatal. Strict json_schema decoding requires every declared
	// property on every element, so a model with nothing to say in a slot cannot omit the slot — it
	// emits `"text": ""` and satisfies the structure. Refusing the whole array for that discards the
	// prose that came with it and buys two repair attempts that fail the same way: measured live on
	// Railway, three narrate calls and ~12s of founder-visible dead air to render one paragraph that
	// arrived correctly on the first call.
	//
	// A blank carries no content, so dropping it loses nothing and hides nothing — and if EVERY
	// segment is blank the model genuinely said nothing, which is still an error below. Every other
	// belt (ghost speaker, verbatim speech, kind/speaker correlation) runs unchanged on what survives:
	// this widens what counts as an acceptable ARRAY, never what counts as an acceptable segment.
	kept := segs[:0]
	for i, s := range segs {
		if strings.TrimSpace(s.Text) == "" {
			continue
		}
		switch s.Kind {
		case "narration":
			if s.SpeakerID != nil {
				return nil, fmt.Errorf("segment %d: kind=narration requires speaker_id null (got %q)", i, *s.SpeakerID)
			}
		case "speech", "action":
			if s.SpeakerID == nil {
				return nil, fmt.Errorf("segment %d: kind=%s requires a non-null speaker_id", i, s.Kind)
			}
			if !present[*s.SpeakerID] {
				return nil, fmt.Errorf("segment %d: speaker_id %q is not present this beat (ghost speaker)", i, *s.SpeakerID)
			}
			if s.Kind == "speech" && !speechIsVerbatim(*s.SpeakerID, s.Text, speechTexts) {
				return nil, fmt.Errorf("segment %d: speech %q is not verbatim — it does not appear in speaker %s's spoken words this beat", i, s.Text, *s.SpeakerID)
			}
		default:
			return nil, fmt.Errorf("segment %d: kind %q outside {narration,speech,action}", i, s.Kind)
		}
		kept = append(kept, s)
	}
	if len(kept) == 0 {
		return nil, fmt.Errorf("narration/1 carried no segment with any text (%d blank)", len(segs))
	}
	return kept, nil
}

// speechIsVerbatim passes when text is an exact or substring match of one of the speaker's speech
// perception contents this beat. Substring (not equality) is deliberate: the perception content can wrap
// the spoken words in scaffolding (see the extraction note above), so the model quoting exactly the
// words is a substring of the fuller line. A paraphrase is not a substring and fails.
func speechIsVerbatim(speaker, text string, speechTexts map[string][]string) bool {
	for _, content := range speechTexts[speaker] {
		if strings.Contains(content, text) {
			return true
		}
	}
	return false
}
