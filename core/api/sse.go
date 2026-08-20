package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// beatFrameSchemaVersion stamps every SSE frame POST /worlds/{w}/beats emits
// (schema/beat_frame.v4.schema.json — core/api/schema/, the frontend repo generates its types from
// that directory).
//
// v2 (2026-08-08), a CLEAN CUTOVER with no v1 alias: the result frame's unresolved_candidates
// changed from bare ids to {id, label} pairs. The frontend pins this string exactly and fails the
// load on a mismatch (pin-and-fail, the agreed negotiation policy), so a silent reshape under the
// v1 id was never an option — the version moving IS the notification.
const beatFrameSchemaVersion = "beat_frame/4"

// frameWriter emits one SSE `data:` line per frame and flushes IMMEDIATELY after each — the whole reason
// POST /worlds/{w}/beats exists as a stream rather than a buffered response (design §4.8, plan rung3
// Task 3): a client watching with `curl -N` sees each frame the instant the handler accepted it, never
// all of them arriving together the moment ServeHTTP returns. Skipping the flush would make the stream a
// lie told in a nicer shape (plan Task 3).
//
// The schema version is per-writer rather than a constant because there are now two streams: the beat
// loop (beat_frame/4) and world creation (world_genesis_frame/1). Both are long authored acts with
// intermediate results, which is what this transport is for; neither may stamp the other's contract.
type frameWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	version string
	// mu serialises emit: the genesis build emits liveness frames from a ticker goroutine while the
	// handler goroutine owns the rest of the stream, and interleaved partial writes would corrupt the
	// SSE framing for every consumer of that response.
	mu sync.Mutex
}

// newFrameWriter wraps w for SSE framing, stamping every frame with schemaVersion. It fails (ok=false)
// only when w does not implement http.Flusher — every real net/http ResponseWriter and
// httptest.ResponseRecorder do; the check exists so a misconfigured wrapper fails loudly with a normal
// HTTP error instead of silently buffering every frame until the handler returns.
func newFrameWriter(w http.ResponseWriter, schemaVersion string) (*frameWriter, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	return &frameWriter{w: w, flusher: flusher, version: schemaVersion}, true
}

// emit marshals payload, merges its fields into the frame envelope
// ({"schema_version":"beat_frame/4","kind":kind, ...payload fields}), writes the result as one SSE
// event, and flushes before returning — the flush is not an afterthought, it is the point (plan Task
// 3's "flush after every frame, or the whole exercise is theatre"). payload must marshal to a JSON
// object; every per-kind payload type in beatsstream.go does, and deliberately NESTS any reused shape
// that would otherwise collide with the envelope's own "kind" key (a narration segment's own
// narration|speech|action kind; a journey block's own travel|wait|watch kind) under a named field
// instead of flattening it — two properties cannot share one JSON key in the same object.
func (fw *frameWriter) emit(kind string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("beat frame %s: marshal payload: %w", kind, err)
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("beat frame %s: payload must marshal to a JSON object: %w", kind, err)
	}
	fields["schema_version"], _ = json.Marshal(fw.version)
	fields["kind"], _ = json.Marshal(kind)
	out, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("beat frame %s: marshal envelope: %w", kind, err)
	}
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if _, err := fmt.Fprintf(fw.w, "data: %s\n\n", out); err != nil {
		return fmt.Errorf("beat frame %s: write: %w", kind, err)
	}
	fw.flusher.Flush()
	return nil
}
