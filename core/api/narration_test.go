package main

import (
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
	raw := `[
	  {"speaker_id":null,"kind":"narration","text":"The common room is low and dim."},
	  {"speaker_id":"m1","kind":"speech","text":"The tide turns at dusk."},
	  {"speaker_id":"m1","kind":"action","text":"Mara sets a tankard on the bar."}
	]`
	segs, err := DecodeAndValidateNarration(raw, unitPresentIDs, unitSpeechTexts)
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
		t.Fatalf("segment 1 must be Mara's speech: %+v", segs[1])
	}
}

// A speech segment attributed to someone NOT in the present roster is a ghost speaker — rejected.
func TestDecodeAndValidateNarration_GhostSpeakerRejected(t *testing.T) {
	raw := `[{"speaker_id":"99999999-9999-9999-9999-999999999999","kind":"speech","text":"The tide turns at dusk."}]`
	_, err := DecodeAndValidateNarration(raw, unitPresentIDs, unitSpeechTexts)
	if err == nil || !strings.Contains(err.Error(), "ghost speaker") {
		t.Fatalf("ghost speaker must be rejected, got err=%v", err)
	}
}

// The belt rejects narrator paraphrase: a speech segment whose words are NOT a substring of the speaker's
// actual perception line — NPC speech must be the mind's exact words.
func TestDecodeAndValidateNarration_ParaphrasedSpeechRejected(t *testing.T) {
	raw := `[{"speaker_id":"m1","kind":"speech","text":"The sea shifts when night comes."}]`
	_, err := DecodeAndValidateNarration(raw, unitPresentIDs, unitSpeechTexts)
	if err == nil || !strings.Contains(err.Error(), "not verbatim") {
		t.Fatalf("paraphrased speech must be rejected by the belt, got err=%v", err)
	}
}

// The speaker_id↔kind correlation: a null speaker with kind=speech is invalid (speech must be attributed).
func TestDecodeAndValidateNarration_NullSpeakerSpeechRejected(t *testing.T) {
	raw := `[{"speaker_id":null,"kind":"speech","text":"The tide turns at dusk."}]`
	_, err := DecodeAndValidateNarration(raw, unitPresentIDs, unitSpeechTexts)
	if err == nil || !strings.Contains(err.Error(), "non-null speaker_id") {
		t.Fatalf("null-speaker speech must be rejected, got err=%v", err)
	}
}

// The other half of the correlation: a narrator segment (kind=narration) may not carry a speaker_id.
func TestDecodeAndValidateNarration_NarrationWithSpeakerRejected(t *testing.T) {
	raw := `[{"speaker_id":"m1","kind":"narration","text":"The room is dim."}]`
	_, err := DecodeAndValidateNarration(raw, unitPresentIDs, unitSpeechTexts)
	if err == nil || !strings.Contains(err.Error(), "speaker_id null") {
		t.Fatalf("narration-with-speaker must be rejected, got err=%v", err)
	}
}

// Empty text is rejected (schema minLength 1, re-enforced by the belt).
func TestDecodeAndValidateNarration_EmptyTextRejected(t *testing.T) {
	raw := `[{"speaker_id":null,"kind":"narration","text":""}]`
	_, err := DecodeAndValidateNarration(raw, unitPresentIDs, unitSpeechTexts)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty text must be rejected, got err=%v", err)
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
