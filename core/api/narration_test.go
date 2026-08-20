package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The belt's evidence for these unit tests: one present speaker (Mara) whose one Communicated line this
// beat wraps her exact words in scaffolding — so the verbatim check must SUBSTRING-match the quoted words.
var (
	unitPresentIDs  = []string{"m1"}
	unitSpeechTexts = map[string][]string{"m1": {`Mara says, "The tide turns at dusk."`}}
)

// Happy path: a narrator segment (null speaker), a VERBATIM speech segment (its text a substring of the
// speaker's perception line), and an action segment (no verbatim requirement) all decode cleanly.
func TestDecodeAndValidateNarration_HappyPath(t *testing.T) {
	// narration/3: the spoken words live in `quote`, and `text` carries only the staging around them.
	raw := `[
	  {"speaker_id":null,"kind":"narration","text":"The common room is low and dim.","quote":null},
	  {"speaker_id":"m1","kind":"speech","text":"she looks up from the tap","quote":"The tide turns at dusk."},
	  {"speaker_id":"m1","kind":"action","text":"Mara sets a tankard on the bar.","quote":null}
	]`
	segs, err := DecodeAndValidateNarration(raw, NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts})
	if err != nil {
		t.Fatalf("happy path rejected: %v", err)
	}
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d", len(segs))
	}
	if segs[0].SpeakerID != nil || segs[0].Kind != "narration" {
		t.Fatalf("segment 0 must be a narrator segment: %+v", segs[0])
	}
	if segs[1].SpeakerID == nil || *segs[1].SpeakerID != "m1" || segs[1].Kind != "speech" {
		t.Fatalf("segment 1 must be the speaker's speech: %+v", segs[1])
	}
	// The split is the point: the words are in `quote`, the staging stays in `text`, and neither
	// carries the other. A renderer keying on `quote` must never receive stage directions.
	if quoteOf(segs[1]) != "The tide turns at dusk." {
		t.Fatalf("speech quote must hold the verbatim words, got %q", quoteOf(segs[1]))
	}
	if segs[1].Text != "she looks up from the tap" {
		t.Fatalf("speech text must hold only the staging, got %q", segs[1].Text)
	}
	if quoteOf(segs[0]) != "" || quoteOf(segs[2]) != "" {
		t.Fatal("narration and action segments must carry no quote")
	}
}

// A speech segment attributed to someone NOT in the present roster is a ghost speaker — rejected.
func TestDecodeAndValidateNarration_GhostSpeakerRejected(t *testing.T) {
	raw := `[{"speaker_id":"99999999-9999-9999-9999-999999999999","kind":"speech","text":"The tide turns at dusk."}]`
	_, err := DecodeAndValidateNarration(raw, NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts})
	if err == nil || !strings.Contains(err.Error(), "ghost speaker") {
		t.Fatalf("ghost speaker must be rejected, got err=%v", err)
	}
}

// The belt rejects narrator paraphrase: a speech segment whose words are NOT a substring of the speaker's
// actual perception line — NPC speech must be the mind's exact words.
func TestDecodeAndValidateNarration_ParaphrasedSpeechRejected(t *testing.T) {
	// The paraphrase is now checked where the words actually are — the quote — instead of against a
	// whole line of narrator prose that could pass on borrowed staging alone.
	raw := `[{"speaker_id":"m1","kind":"speech","text":"she looks up","quote":"The sea shifts when night comes."}]`
	_, err := DecodeAndValidateNarration(raw, NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts})
	if err == nil || !strings.Contains(err.Error(), "not verbatim") {
		t.Fatalf("paraphrased speech must be rejected by the belt, got err=%v", err)
	}
}

// The speaker_id↔kind correlation: a null speaker with kind=speech is invalid (speech must be attributed).
func TestDecodeAndValidateNarration_NullSpeakerSpeechRejected(t *testing.T) {
	raw := `[{"speaker_id":null,"kind":"speech","text":"The tide turns at dusk."}]`
	_, err := DecodeAndValidateNarration(raw, NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts})
	if err == nil || !strings.Contains(err.Error(), "non-null speaker_id") {
		t.Fatalf("null-speaker speech must be rejected, got err=%v", err)
	}
}

// The other half of the correlation: a narrator segment (kind=narration) may not carry a speaker_id.
func TestDecodeAndValidateNarration_NarrationWithSpeakerRejected(t *testing.T) {
	raw := `[{"speaker_id":"m1","kind":"narration","text":"The room is dim."}]`
	_, err := DecodeAndValidateNarration(raw, NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts})
	if err == nil || !strings.Contains(err.Error(), "speaker_id null") {
		t.Fatalf("narration-with-speaker must be rejected, got err=%v", err)
	}
}

// A narration with no words in it is rejected (schema minLength 1, re-enforced by the belt). A blank
// segment ALONGSIDE real prose is now dropped rather than fatal — see
// TestNarration_BlankSegmentsAreDroppedNotFatal — but a narration that is nothing but blanks said
// nothing, and the caller must fall back rather than emit silence.
func TestDecodeAndValidateNarration_EmptyTextRejected(t *testing.T) {
	raw := `[{"speaker_id":null,"kind":"narration","text":""}]`
	_, err := DecodeAndValidateNarration(raw, NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts})
	if err == nil || !strings.Contains(err.Error(), "no segment with any text") {
		t.Fatalf("a wholly empty narration must be rejected, got err=%v", err)
	}
}

// The structured narrate prompt carries the segment-output contract AND each PRESENT entry's speaker id
// ("label [id]"), so a model can attribute speech/actions; the PLAIN fallback prompt strips the segment
// contract and demands prose only.
func TestBuildNarratePrompt_CarriesSegmentContractAndSpeakerIDs(t *testing.T) {
	payload := PerceptionPayload{Candidates: []Candidate{
		{ID: "m1", Name: "Mara", Kind: "actor"},
		{ID: "l1", Name: "The Drowned Lantern", Kind: "location"},
	}}
	structured := buildNarratePrompt(payload, "", nil, "")
	if !strings.Contains(structured, narrateSegmentContractMarker) {
		t.Fatalf("structured narrate prompt missing the segment-output contract %q", narrateSegmentContractMarker)
	}
	if !strings.Contains(structured, "Mara [m1]") {
		t.Fatalf("PRESENT roster must carry the speaker id as \"Mara [m1]\":\n%s", structured)
	}

	plain := buildNarratePlainPrompt(payload, "", nil, "")
	if strings.Contains(plain, narrateSegmentContractMarker) {
		t.Fatalf("plain fallback must NOT carry the segment contract (it re-asks for prose):\n%s", plain)
	}
	if !strings.Contains(plain, "prose only") {
		t.Fatalf("plain fallback must demand prose only:\n%s", plain)
	}
}

// ── the founder's first live beat, as two regressions ───────────────────────────────────────────

// PLACE was "the last candidate of kind location", correct only while exactly one location could be
// a candidate. SPEC-030 widened the whitelist to this room's portals AND the rooms beyond them;
// buildScene was fixed for exactly this reason and the narrator was not. Live symptom: a
// look-around in the tavern opened "The dim cellar air presses close around you".
func TestNarratePrompt_SetsTheSceneInTheRoomTheViewerIsIn(t *testing.T) {
	payload := PerceptionPayload{
		Here: "loc-tavern",
		Candidates: []Candidate{
			{ID: "loc-tavern", Kind: "location", Name: "The Drowned Lantern", Description: "Low beams, salt-rot, one hearth."},
			{ID: "loc-cellar", Kind: "location", Name: "Cellar", Description: "A cold stone undercroft."},
			{ID: "npc-1", Kind: "actor", Name: "Mara"},
		},
	}
	got := buildNarratePlainPrompt(payload, "viewer-1", nil, "completed")
	if !strings.Contains(got, "PLACE: The Drowned Lantern") {
		t.Fatalf("prompt does not set the scene in the viewer's own room:\n%s", got)
	}
	if strings.Contains(got, "PLACE: Cellar") || strings.Contains(got, "cold stone undercroft") {
		t.Fatalf("a neighbouring room reached the PLACE block:\n%s", got)
	}
}

// With no Here — a direct unit call or an unplaced viewer — the old behaviour stands rather than
// rendering a placeless scene.
func TestNarratePrompt_FallsBackWhenTheViewerHasNoRoom(t *testing.T) {
	payload := PerceptionPayload{Candidates: []Candidate{
		{ID: "loc-tavern", Kind: "location", Name: "The Drowned Lantern"},
	}}
	if got := buildNarratePlainPrompt(payload, "viewer-1", nil, "completed"); !strings.Contains(got, "PLACE: The Drowned Lantern") {
		t.Fatalf("a placeless payload lost its only location:\n%s", got)
	}
}

// The narrator was handed distance_m and move_duration_s verbatim and did the only honest thing with
// a number: read it out. "maybe nine strides off — close to a seven-count", "barely five meters
// distant, a mere four-second walk". Accurate, and a range table rather than a room.
func TestNarratePrompt_GeometryBecomesStagingNotAReadout(t *testing.T) {
	fs := json.RawMessage(`{"targets":[
		{"id":"a","name":"Mara","distance_m":11.66,"move_duration_s":9,"reachable":true,"locked":null},
		{"id":"b","name":"the bar","distance_m":0.9,"move_duration_s":1,"open":true}],"budget_remaining":null}`)
	got := buildNarratePlainPrompt(PerceptionPayload{Here: "loc-1", Candidates: []Candidate{
		{ID: "loc-1", Kind: "location", Name: "The Drowned Lantern"}}},
		"viewer-1", nil, "completed", QueryAnswer{Stated: "who is there?", FactSheet: fs})

	for _, banned := range []string{"distance_m", "move_duration_s", "11.66", "0.9"} {
		if strings.Contains(got, banned) {
			t.Fatalf("reciteable geometry reached the narrator (%q):\n%s", banned, got)
		}
	}
	// Proximity survives, because staging is what the measurement was FOR.
	if !strings.Contains(got, "across the room") || !strings.Contains(got, "within arm's reach") {
		t.Fatalf("staging bands missing — the narrator can no longer tell near from far:\n%s", got)
	}
	// Perceptible facts are untouched: a person can say a door is open; they cannot say it is 11.66m away.
	for _, kept := range []string{`"reachable":true`, `"open":true`, "Mara"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("a perceptible fact was stripped along with the numbers (%q):\n%s", kept, got)
		}
	}
}

func TestNarratorFactSheet_BandsAndSurvivesGarbage(t *testing.T) {
	for _, tc := range []struct {
		m    float64
		want string
	}{
		{0.5, "within arm's reach"}, {3, "a few steps away"},
		{11.66, "across the room"}, {25, "some way off"}, {500, "far off"},
	} {
		if got := proximityBand(tc.m); got != tc.want {
			t.Fatalf("proximityBand(%v) = %q, want %q", tc.m, got, tc.want)
		}
	}
	// A fact sheet the narrator cannot read is a worse failure than one it reads too literally: junk
	// passes through rather than costing the beat its answer.
	if got := narratorFactSheet(json.RawMessage(`not json`)); got != "not json" {
		t.Fatalf("unparseable fact sheet = %q, want it passed through verbatim", got)
	}
}

// A blank segment is DROPPED, not fatal. Strict json_schema decoding requires every declared
// property on every element, so a model with nothing to say in a slot cannot omit it — it emits
// `"text": ""`. Measured live: refusing the array for that cost three narrate calls and ~12s of dead
// air to render a paragraph that had arrived correctly on the first one.
func TestNarration_BlankSegmentsAreDroppedNotFatal(t *testing.T) {
	raw := `[{"speaker_id":null,"kind":"narration","text":""},
	         {"speaker_id":null,"kind":"narration","text":"The tide mutters against the quay."}]`
	segs, err := DecodeAndValidateNarration(raw, NarrationBelts{})
	if err != nil {
		t.Fatalf("a blank leading segment must not cost the whole narration: %v", err)
	}
	if len(segs) != 1 || segs[0].Text != "The tide mutters against the quay." {
		t.Fatalf("segments = %+v, want just the one with words in it", segs)
	}
}

// All-blank is a model that genuinely said nothing, and that IS an error — the widening is about
// what counts as an acceptable ARRAY, never about accepting an empty narration.
func TestNarration_AllBlankIsStillAnError(t *testing.T) {
	_, err := DecodeAndValidateNarration(
		`[{"speaker_id":null,"kind":"narration","text":""},{"speaker_id":null,"kind":"narration","text":"   "}]`, NarrationBelts{})
	if err == nil || !strings.Contains(err.Error(), "no segment with any text") {
		t.Fatalf("err = %v, want an all-blank narration rejected", err)
	}
}

// The belts still run on what survives: a blank segment must not become a way to smuggle a ghost
// speaker past the wall by hiding behind an earlier empty element.
func TestNarration_BeltsStillRunAfterDropping(t *testing.T) {
	raw := `[{"speaker_id":null,"kind":"narration","text":""},
	         {"speaker_id":"11111111-1111-1111-1111-111111111111","kind":"speech","text":"I never said this."}]`
	if _, err := DecodeAndValidateNarration(raw, NarrationBelts{}); err == nil {
		t.Fatal("a ghost speaker after a dropped blank must still be refused")
	}
}

// ── narration/3: the speech/staging split ────────────────────────────────────────────────────────

// The split is only real if the shape enforces it. Words in the wrong field must not decode, or the
// frontend's "format speech differently" reduces to guessing where a quotation mark starts.
func TestNarration_SpeechRequiresItsWordsInQuote(t *testing.T) {
	belts := NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts}

	// The narration/1 habit: words in `text`, no quote. Rejected — and the message must say where the
	// words belong, because this is the mistake a model will make for as long as narration/1 examples
	// exist anywhere in its training.
	_, err := DecodeAndValidateNarration(
		`[{"speaker_id":"m1","kind":"speech","text":"The tide turns at dusk.","quote":null}]`, belts)
	if err == nil || !strings.Contains(err.Error(), "requires a non-empty quote") {
		t.Fatalf("speech with no quote must be rejected with guidance, got: %v", err)
	}

	// A bare line — no staging at all — is ordinary writing and must pass: `text` may be empty when
	// `quote` carries the segment.
	segs, err := DecodeAndValidateNarration(
		`[{"speaker_id":"m1","kind":"speech","text":"","quote":"The tide turns at dusk."}]`, belts)
	if err != nil {
		t.Fatalf("a bare quote with no staging must pass, got: %v", err)
	}
	if len(segs) != 1 {
		t.Fatalf("the bare-quote segment must survive the blank-segment drop, got %d segments", len(segs))
	}
}

// An act has no spoken words, and prose is not speech: a quote on either is a category error the
// renderer would have to disambiguate at display time.
func TestNarration_OnlySpeechMayCarryAQuote(t *testing.T) {
	belts := NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: unitSpeechTexts}

	if _, err := DecodeAndValidateNarration(
		`[{"speaker_id":"m1","kind":"action","text":"she sets down the tankard","quote":"The tide turns at dusk."}]`,
		belts); err == nil || !strings.Contains(err.Error(), "kind=action carries a quote") {
		t.Fatalf("an action with a quote must be rejected, got: %v", err)
	}
	if _, err := DecodeAndValidateNarration(
		`[{"speaker_id":null,"kind":"narration","text":"The room holds still.","quote":"The tide turns at dusk."}]`,
		belts); err == nil || !strings.Contains(err.Error(), "kind=narration carries a quote") {
		t.Fatalf("narrator prose with a quote must be rejected, got: %v", err)
	}
}

// The belts must see BOTH fields. A wall breach or a stolen player line hidden inside `quote` is
// exactly as delivered to the player as one in `text` — the split must not open a bypass.
func TestNarration_BeltsInspectTheQuoteToo(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	wall, err := loadNamingWall(context.Background(), pool, dlWorldID, dlKadeID)
	if err != nil {
		t.Fatalf("loadNamingWall: %v", err)
	}
	speech := map[string][]string{"m1": {`she says, "Jonas will not move." and "Easy. I mean no trouble."`}}

	if _, err := DecodeAndValidateNarration(
		`[{"speaker_id":"m1","kind":"speech","text":"she leans in","quote":"Jonas will not move."}]`,
		NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: speech, Wall: wall},
	); err == nil || !strings.Contains(err.Error(), "has not earned") {
		t.Fatalf("an unearned name inside a QUOTE must trip the naming wall, got: %v", err)
	}

	if _, err := DecodeAndValidateNarration(
		`[{"speaker_id":"m1","kind":"speech","text":"she leans in","quote":"Easy. I mean no trouble."}]`,
		NarrationBelts{PresentIDs: unitPresentIDs, SpeechTexts: speech,
			Player: newPlayerVoice("I raise both hands. 'Easy. I mean no trouble.'")},
	); err == nil || !strings.Contains(err.Error(), "PLAYER's own") {
		t.Fatalf("the player's words inside an NPC's QUOTE must be rejected, got: %v", err)
	}
}

// narration/3's emotion tag: optional, closed-set, and speech/action-only. The belt is the wall a
// misbehaving driver hits — the schema constrains a cooperative provider, this constrains reality.
func TestNarration_EmotionIsClosedSetAndNeverOnNarratorProse(t *testing.T) {
	speaker := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	belts := NarrationBelts{PresentIDs: []string{speaker},
		SpeechTexts: map[string][]string{speaker: {"The tide turns at dusk.", "Hm."}}}

	segs, err := DecodeAndValidateNarration(
		`[{"speaker_id":"`+speaker+`","kind":"speech","text":"","quote":"The tide turns at dusk.","emotion":"happy"}]`, belts)
	if err != nil {
		t.Fatalf("a tagged speech segment was rejected: %v", err)
	}
	if segs[0].Emotion == nil || *segs[0].Emotion != "happy" {
		t.Fatalf("emotion = %v, want happy carried through the belt", segs[0].Emotion)
	}

	if _, err := DecodeAndValidateNarration(
		`[{"speaker_id":"`+speaker+`","kind":"speech","text":"","quote":"Hm.","emotion":"smug"}]`, belts); err == nil {
		t.Fatal("an out-of-vocabulary emotion was accepted")
	}

	if _, err := DecodeAndValidateNarration(
		`[{"speaker_id":null,"kind":"narration","text":"The room is dim.","quote":null,"emotion":"sad"}]`, belts); err == nil {
		t.Fatal("narrator prose carried an emotion — the narrator is not a character and has no sprite")
	}

	// Absent is the ordinary state: an untagged line is simply neutral downstream.
	segs, err = DecodeAndValidateNarration(
		`[{"speaker_id":"`+speaker+`","kind":"action","text":"sets a tankard down.","quote":null}]`, belts)
	if err != nil || segs[0].Emotion != nil {
		t.Fatalf("untagged action: err=%v emotion=%v, want accepted with no emotion", err, segs[0].Emotion)
	}
}
