package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// A STREAMED speech line must reach the wire carrying its quote.
//
// Regression test with a live scar. narration/2 added `quote`; narrateMessages learned to carry it;
// the streaming emit site — which built its beatMessage BY HAND from four fields — silently kept
// emitting the old four. The frontend received
//
//	{"kind":"speech","speaker_label":"Jonas","text":"","quote":null}
//
// an empty bubble where dialogue should be, while the transcript row (derived through
// narrateMessages) held the words correctly. Live frames and stored history disagreeing is precisely
// the failure the shared message shape exists to prevent, so the fix was to delete the second
// derivation rather than to patch it — and this test fails if anyone reintroduces one.
//
// Aimed at narrateStream directly: the defect was in the emit path, and a full beat would drag in
// cognition and adjudication to make an NPC speak, testing five other things and this one by accident.
func TestNarrateStream_SpeechFrameCarriesItsQuote(t *testing.T) {
	const speaker = "2ac70000-0000-0000-0000-0000000000a2"
	const spoken = "the note is not yours"

	sd := newFakeStreamingNarrateDriver("fake-streaming:quote", []string{
		`{"speaker_id":null,"kind":"narration","text":"The common room holds still.","quote":null}`,
		`{"speaker_id":"` + speaker + `","kind":"speech","text":"she looks up","quote":"` + spoken + `"}`,
	})
	go func() { close(sd.resume) }()

	rec := httptest.NewRecorder()
	frames, ok := newFrameWriter(rec, beatFrameSchemaVersion)
	if !ok {
		t.Fatal("recorder is not a flusher")
	}

	belts := NarrationBelts{
		PresentIDs: []string{speaker},
		// The verbatim belt is backed by canon spoken words (payload.spoken); here it stands in for the
		// utterance the speaker was perceived to make.
		SpeechTexts: map[string][]string{speaker: {spoken}},
	}
	segs, err := narrateStream(context.Background(), sd, GenRequest{}, belts,
		map[string]string{speaker: "Mara"}, frames)
	if err != nil {
		t.Fatalf("narrateStream: %v", err)
	}
	if len(segs) != 2 {
		t.Fatalf("want both lines validated, got %d", len(segs))
	}

	var sawSpeech bool
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var f struct {
			SchemaVersion string `json:"schema_version"`
			Kind          string `json:"kind"`
			Message       struct {
				Kind         string  `json:"kind"`
				SpeakerLabel string  `json:"speaker_label"`
				Text         string  `json:"text"`
				Quote        *string `json:"quote"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &f); err != nil {
			t.Fatalf("frame is not JSON: %v", err)
		}
		if f.SchemaVersion != "beat_frame/4" {
			t.Fatalf("frame envelope = %q, want beat_frame/4", f.SchemaVersion)
		}
		if f.Message.Kind != "speech" {
			continue
		}
		sawSpeech = true
		if f.Message.Quote == nil || *f.Message.Quote != spoken {
			t.Fatalf("streamed speech frame lost its quote: text=%q quote=%v", f.Message.Text, f.Message.Quote)
		}
		if f.Message.Text != "she looks up" {
			t.Fatalf("staging text = %q, want it preserved BESIDE the quote, not merged into it", f.Message.Text)
		}
		if f.Message.SpeakerLabel != "Mara" {
			t.Fatalf("speaker_label = %q, want the viewer's label for the speaker", f.Message.SpeakerLabel)
		}
	}
	if !sawSpeech {
		t.Fatal("no speech frame reached the wire")
	}
}
