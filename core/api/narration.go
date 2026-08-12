package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// narrationV2SchemaJSON is the structured-output leash handed to the narrate seat (founder envelope):
// the beat response stops being one narrator blob and becomes an ORDERED list of typed segments —
// narrator prose plus attributed NPC speech and single-NPC actions. Registered as an input-contract
// schema (ci/schema_contract.py, SPEC-011 house rule) — it is the seat's leash, not a projection
// payload, so it has no SQL payload generator by design.
//
//go:embed schema/narration.v2.schema.json
var narrationV2SchemaJSON string

// NarrationSegment is one element of narration/2: narrator prose (SpeakerID nil, Kind "narration") or an
// attributed NPC line (SpeakerID non-nil, Kind "speech" for the mind's exact words, "action" for one
// clean single-NPC act). SpeakerID is a *string so the null⇔narration correlation is decodable (an
// absent/explicit-null speaker is distinct from the empty string).
type NarrationSegment struct {
	SpeakerID *string `json:"speaker_id"`
	Kind      string  `json:"kind"`
	// Text is prose: the whole segment for narration and action, and for speech the STAGING around the
	// line ("she leans in, her voice dropping"), which may be empty when the line is delivered bare.
	Text string `json:"text"`
	// Quote is the verbatim spoken words, without quotation marks or attribution — non-empty exactly
	// when Kind is "speech". Separating it from Text is what lets the frontend format speech and lets
	// the verbatim belt check the words claimed to be spoken instead of a sentence of narrator prose.
	Quote *string `json:"quote"`
}

// DecodeAndValidateNarration is the DEFENSE-IN-DEPTH belt behind the narrate seat's structured-output
// leash (schema/narration.v2.schema.json). Structured decoding makes the output schema-valid by
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
// NarrationBelts is everything a segment is checked against. It is a struct rather than four
// positional arguments because it grew to four in two days and each one is optional in a different
// way — a direct unit call has no viewer, a first beat has no speech, and a nil belt must mean
// "not checked" rather than "silently passes something it should not".
type NarrationBelts struct {
	PresentIDs  []string            // ghost-speaker check: who can be attributed at all
	SpeechTexts map[string][]string // verbatim check: what each speaker actually said, per speaker id
	Wall        *NamingWall         // naming reach (B-1, I-3): names this viewer has not earned
	Player      *PlayerVoice        // the player's own words and acts, which no NPC may perform
}

// DecodeAndValidateNarration is the gate every segment passes before it can become a frame. A
// rejection is not a failure mode — it is the seat being asked again, which is the cheapest correct
// answer available when a model writes something the world forbids.
func DecodeAndValidateNarration(raw string, b NarrationBelts) ([]NarrationSegment, error) {
	presentIDs, speechTexts, wall := b.PresentIDs, b.SpeechTexts, b.Wall
	var segs []NarrationSegment
	if err := json.Unmarshal([]byte(raw), &segs); err != nil {
		return nil, fmt.Errorf("narration/2 not valid JSON: %w", err)
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
		// A blank segment is dropped (see above) — but a SPEECH segment with empty staging is not
		// blank: its content is the quote, and a bare line with no stage direction is ordinary writing.
		if strings.TrimSpace(s.Text) == "" && strings.TrimSpace(quoteOf(s)) == "" {
			continue
		}
		switch s.Kind {
		case "narration":
			if s.SpeakerID != nil {
				return nil, fmt.Errorf("segment %d: kind=narration requires speaker_id null (got %q)", i, *s.SpeakerID)
			}
			if quoteOf(s) != "" {
				return nil, fmt.Errorf("segment %d: kind=narration carries a quote — spoken words belong to a speech segment with a speaker", i)
			}
		case "speech", "action":
			if s.SpeakerID == nil {
				return nil, fmt.Errorf("segment %d: kind=%s requires a non-null speaker_id", i, s.Kind)
			}
			if !present[*s.SpeakerID] {
				return nil, fmt.Errorf("segment %d: speaker_id %q is not present this beat (ghost speaker)", i, *s.SpeakerID)
			}
			// THE SPLIT (narration/2). A speech segment's words live in `quote`; an action never has
			// words at all. The verbatim belt then checks exactly what is claimed to be SPOKEN, rather
			// than substring-matching a whole line of narrator prose and passing whenever the staging
			// happened to be borrowed from the perception.
			if s.Kind == "action" && quoteOf(s) != "" {
				return nil, fmt.Errorf("segment %d: kind=action carries a quote — an act has no spoken words; use kind=speech", i)
			}
			if s.Kind == "speech" {
				if quoteOf(s) == "" {
					return nil, fmt.Errorf("segment %d: kind=speech requires a non-empty quote — the spoken words go in `quote`, not in `text`", i)
				}
				if !speechIsVerbatim(*s.SpeakerID, quoteOf(s), speechTexts) {
					return nil, fmt.Errorf("segment %d: quote %q is not verbatim — it does not appear in speaker %s's spoken words this beat", i, quoteOf(s), *s.SpeakerID)
				}
			}
			// THE PLAYER'S OWN VOICE. An attributed segment may not perform the player's act or quote
			// his words: he is never in PRESENT, so the only way his line lands under someone's id is
			// the narrator reaching for the nearest available id. Live symptom: the founder's
			// "I raise both hands, empty. 'Easy. I mean no trouble.'" came back as a hooded figure's
			// action. `action` segments carry no verbatim requirement — deliberately, since an act is
			// viewer-relative prose — which left this the one attributed shape with nothing checking it.
			if echo := firstNonEmpty(b.Player.Echoes(s.Text), b.Player.Echoes(quoteOf(s))); echo != "" {
				return nil, fmt.Errorf("segment %d: attributed to %s but repeats the PLAYER's own words/act (%q) — "+
					"the player is never in PRESENT; render his own moment as a narration segment in second person",
					i, *s.SpeakerID, echo)
			}
		default:
			return nil, fmt.Errorf("segment %d: kind %q outside {narration,speech,action}", i, s.Kind)
		}
		// THE NAMING WALL. Runs on every kind: narration prose is the reported case, and speech is no
		// safer — an NPC who says a name the player has not earned teaches it without a knowledge path,
		// which is the same breach wearing quotation marks (see SPEC-033 for learning-by-earshot).
		if v := wall.Violations(s.Text + " " + quoteOf(s)); len(v) > 0 {
			return nil, namingWallError(i, v)
		}
		kept = append(kept, s)
	}
	if len(kept) == 0 {
		return nil, fmt.Errorf("narration/2 carried no segment with any text (%d blank)", len(segs))
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

// quoteOf reads a segment's spoken words, "" when it has none. Nil-safe so every belt can ask without
// repeating the pointer dance.
func quoteOf(s NarrationSegment) string {
	if s.Quote == nil {
		return ""
	}
	return *s.Quote
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
