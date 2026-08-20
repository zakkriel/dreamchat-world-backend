package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The transcript's contract is "what the player actually saw". This drives the REAL handler over the
// REAL write path: persist a beat the way the stream does, then read it back through ServeHTTP.
func TestTranscript_StoresWhatWasDeliveredAndServesItBack(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// A beat as delivered: narrator prose, then a speech line with its quote split from its staging.
	speaker := "2ac70000-0000-0000-0000-0000000000a2"
	label := "Mara"
	quote := "That is Reyna's brother."
	delivered := []beatMessage{
		{SpeakerID: nil, SpeakerLabel: "", Kind: "narration", Text: "The hearth pops once."},
		{SpeakerID: &speaker, SpeakerLabel: label, Kind: "speech", Text: "she tilts her head", Quote: &quote},
	}
	persistTranscript(ctx, pool, dlWorldID, dlKadeID, 77, "I ask about the muscle", delivered, "completed", nil)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM transcript_entry WHERE world_id=$1 AND viewer_id=$2 AND in_world_tick=77`,
			dlWorldID, dlKadeID)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/worlds/"+dlWorldID+"/transcript?viewer="+dlKadeID+"&limit=1", nil)
	NewTranscriptHandler(pool, true).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var got struct {
		SchemaVersion string `json:"schema_version"`
		Entries       []struct {
			Tick     int64   `json:"tick"`
			Stated   *string `json:"stated"`
			Segments []struct {
				Kind         string  `json:"kind"`
				SpeakerLabel string  `json:"speaker_label"`
				Text         string  `json:"text"`
				Quote        *string `json:"quote"`
			} `json:"segments"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not transcript/2 JSON: %v", err)
	}
	if got.SchemaVersion != "transcript/2" {
		t.Fatalf("schema_version = %q, want transcript/2", got.SchemaVersion)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("want the one entry just written, got %d", len(got.Entries))
	}
	e := got.Entries[0]
	if e.Tick != 77 || e.Stated == nil || *e.Stated != "I ask about the muscle" {
		t.Fatalf("the player's own words must be stored raw: tick=%d stated=%v", e.Tick, e.Stated)
	}
	if len(e.Segments) != 2 {
		t.Fatalf("want both delivered segments, got %d", len(e.Segments))
	}
	// The split must survive the round trip, or the frontend cannot render history the way it renders
	// live frames — which is the entire reason the shapes were made identical.
	s := e.Segments[1]
	if s.Kind != "speech" || s.Quote == nil || *s.Quote != quote || s.Text != "she tilts her head" {
		t.Fatalf("speech segment lost its split: %+v", s)
	}
	if s.SpeakerLabel != label {
		t.Fatalf("speaker label = %q, want the label as delivered (%q)", s.SpeakerLabel, label)
	}
}

// A beat that narrated nothing leaves no memory — an empty entry would render as a blank bubble in
// the founder's chat, which is worse than an honest gap.
func TestTranscript_SilentBeatWritesNothing(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	var before int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM transcript_entry WHERE world_id=$1`, dlWorldID).Scan(&before); err != nil {
		t.Fatalf("count: %v", err)
	}
	persistTranscript(ctx, pool, dlWorldID, dlKadeID, 78, "I wait", nil, "completed", nil)
	var after int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM transcript_entry WHERE world_id=$1`, dlWorldID).Scan(&after); err != nil {
		t.Fatalf("count: %v", err)
	}
	if after != before {
		t.Fatalf("a beat with no narration wrote %d row(s)", after-before)
	}
}

// A malformed cursor is refused, not ignored: silently serving page 1 to a client asking for page 4
// looks exactly like reaching the end of the story.
func TestTranscript_RefusesAMalformedCursor(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/worlds/"+dlWorldID+"/transcript?viewer="+dlKadeID+"&before=banana", nil)
	NewTranscriptHandler(pool, true).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400 for a non-numeric cursor", rec.Code)
	}
}
