package main

// POST /worlds/{w}/beats (design §4.8, plan rung3 Task 3) — the SAME beat as the singular /beat
// endpoint, delivered as a stream of validated frames. Three named tests pin the three things that
// decide whether this is real or theatre (plan Task 3): frames are flushed individually (not
// buffered), the belts still run before a narration frame reaches the wire, and a mid-stream failure
// is a defined `error` frame rather than a dropped connection or a leaked stack trace.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// ── a minimal Draft-7-ish JSON Schema validator, just enough for beat_frame.v1.schema.json ─────────
// (const/enum/type/required/properties/additionalProperties/items/oneOf/$ref/minLength/minimum/
// maximum) — the repo carries no jsonschema dependency (go.mod), and `make schema-contract`'s python
// validator (ci/schema_contract.py) already does the AUTHORITATIVE two-sided check against real
// generated payloads; this is the unit-test-time net that pins "every frame validates against the
// published schema" per plan Task 3, step 1.

type miniSchema = map[string]any

func schemaDefs(root miniSchema) miniSchema {
	defs, _ := root["$defs"].(miniSchema)
	return defs
}

// validateAgainstSchema returns every violation of schema found in value (empty = valid).
func validateAgainstSchema(defs miniSchema, schema miniSchema, value any, path string) []string {
	if ref, ok := schema["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/$defs/")
		sub, ok := defs[name].(miniSchema)
		if !ok {
			return []string{fmt.Sprintf("%s: unresolved $ref %q", path, ref)}
		}
		return validateAgainstSchema(defs, sub, value, path)
	}
	if branches, ok := schema["oneOf"].([]any); ok {
		var branchErrs []string
		for i, b := range branches {
			bm, _ := b.(miniSchema)
			e := validateAgainstSchema(defs, bm, value, path)
			if len(e) == 0 {
				return nil // exactly what oneOf needs: at least one clean match
			}
			branchErrs = append(branchErrs, fmt.Sprintf("  branch %d: %s", i, strings.Join(e, "; ")))
		}
		return []string{fmt.Sprintf("%s: matched none of %d oneOf branches:\n%s", path, len(branches), strings.Join(branchErrs, "\n"))}
	}

	var errs []string
	if constVal, ok := schema["const"]; ok {
		if fmt.Sprint(constVal) != fmt.Sprint(value) {
			errs = append(errs, fmt.Sprintf("%s: want const %v, got %v", path, constVal, value))
		}
	}
	if enumVals, ok := schema["enum"].([]any); ok {
		found := false
		for _, e := range enumVals {
			if fmt.Sprint(e) == fmt.Sprint(value) {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("%s: %v not in enum %v", path, value, enumVals))
		}
	}
	if typ, ok := schema["type"]; ok {
		if !schemaTypeMatches(typ, value) {
			errs = append(errs, fmt.Sprintf("%s: type mismatch, schema wants %v, got %T (%v)", path, typ, value, value))
			return errs // descending further into a wrongly-typed value is not informative
		}
	}
	if minLen, ok := schema["minLength"].(float64); ok {
		if s, ok := value.(string); ok && float64(len(s)) < minLen {
			errs = append(errs, fmt.Sprintf("%s: length %d < minLength %v", path, len(s), minLen))
		}
	}
	if min, ok := schema["minimum"].(float64); ok {
		if n, ok := value.(float64); ok && n < min {
			errs = append(errs, fmt.Sprintf("%s: %v < minimum %v", path, n, min))
		}
	}
	if max, ok := schema["maximum"].(float64); ok {
		if n, ok := value.(float64); ok && n > max {
			errs = append(errs, fmt.Sprintf("%s: %v > maximum %v", path, n, max))
		}
	}

	switch v := value.(type) {
	case map[string]any:
		if required, ok := schema["required"].([]any); ok {
			for _, r := range required {
				if _, ok := v[r.(string)]; !ok {
					errs = append(errs, fmt.Sprintf("%s: missing required field %q", path, r))
				}
			}
		}
		props, _ := schema["properties"].(miniSchema)
		additionalOK := true
		if ap, ok := schema["additionalProperties"]; ok {
			if b, ok := ap.(bool); ok {
				additionalOK = b
			}
		}
		for k, val := range v {
			sub, known := props[k].(miniSchema)
			if !known {
				if !additionalOK {
					errs = append(errs, fmt.Sprintf("%s.%s: additional property not allowed", path, k))
				}
				continue
			}
			errs = append(errs, validateAgainstSchema(defs, sub, val, path+"."+k)...)
		}
	case []any:
		if items, ok := schema["items"].(miniSchema); ok {
			for i, el := range v {
				errs = append(errs, validateAgainstSchema(defs, items, el, fmt.Sprintf("%s[%d]", path, i))...)
			}
		}
	}
	return errs
}

func schemaTypeMatches(typ any, value any) bool {
	switch t := typ.(type) {
	case string:
		return scalarSchemaTypeMatches(t, value)
	case []any:
		for _, tt := range t {
			if s, ok := tt.(string); ok && scalarSchemaTypeMatches(s, value) {
				return true
			}
		}
	}
	return false
}

func scalarSchemaTypeMatches(t string, value any) bool {
	switch t {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	case "integer":
		f, ok := value.(float64)
		return ok && f == float64(int64(f))
	case "number":
		_, ok := value.(float64)
		return ok
	}
	return false
}

// beatFrameSchemaRoot loads the published schema fresh (cheap; the file is a few KB) — reloading
// per-call means a schema edit is picked up without process restart, and avoids a shared cache that
// could mask a load failure in one test while another already primed it.
func beatFrameSchemaRoot(t *testing.T) miniSchema {
	t.Helper()
	b, err := os.ReadFile("schema/beat_frame.v1.schema.json")
	if err != nil {
		t.Fatalf("read schema/beat_frame.v1.schema.json: %v", err)
	}
	var root miniSchema
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("schema/beat_frame.v1.schema.json is not valid JSON: %v", err)
	}
	return root
}

// assertValidBeatFrame decodes raw (one SSE frame's data payload) and fails t unless it validates
// against the published beat_frame/1 schema — the plan's "assert every frame validates against the
// published schema" (Task 3, step 1). Returns the decoded frame so callers can inspect fields.
func assertValidBeatFrame(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var frame map[string]any
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("frame is not valid JSON: %v (raw=%s)", err, raw)
	}
	root := beatFrameSchemaRoot(t)
	if errs := validateAgainstSchema(schemaDefs(root), root, frame, "$"); len(errs) > 0 {
		t.Fatalf("frame does not validate against beat_frame/1:\n%s\nraw=%s", strings.Join(errs, "\n"), raw)
	}
	return frame
}

// ── SSE parsing ──────────────────────────────────────────────────────────────────────────────────

// sseReader reads one `data: ...\n\n` event at a time off r, blocking on the underlying read exactly
// as a real client does — which is what lets TestBeats_EmitsFramesInOrder prove a frame arrived
// before the handler finished, rather than merely asserting it in prose.
type sseReader struct{ r *bufio.Reader }

func newSSEReader(r io.Reader) *sseReader { return &sseReader{r: bufio.NewReader(r)} }

// nextRaw returns the next frame's raw JSON bytes (the exact bytes frameWriter.emit wrote after the
// "data: " prefix), or io.EOF once the stream is exhausted with no partial event pending.
func (s *sseReader) nextRaw() ([]byte, error) {
	var buf bytes.Buffer
	for {
		line, err := s.r.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" && err == nil {
			if buf.Len() == 0 {
				continue // stray blank line before any data
			}
			break // blank line = SSE event boundary
		}
		if after, ok := strings.CutPrefix(trimmed, "data: "); ok {
			buf.WriteString(after)
		}
		if err != nil {
			if buf.Len() == 0 {
				return nil, err // clean end of stream (or a genuine read error) with nothing pending
			}
			break // last event with no trailing blank line — take what we have
		}
	}
	if buf.Len() == 0 {
		return nil, io.EOF
	}
	return buf.Bytes(), nil
}

// ── a driver that blocks its first Generate call until the test releases it ────────────────────────

// gatedDriver wraps inner and, on its FIRST Generate call only, closes `started` (so the test can
// observe it was reached) and then blocks until the test closes `resume`. This is what lets
// TestBeats_EmitsFramesInOrder read a frame off the wire while the handler is still working on a
// LATER stage — the only honest way to prove per-frame flushing rather than assert it in prose.
type gatedDriver struct {
	inner   Driver
	started chan struct{}
	resume  chan struct{}
	once    sync.Once
}

func newGatedDriver(inner Driver) *gatedDriver {
	return &gatedDriver{inner: inner, started: make(chan struct{}), resume: make(chan struct{})}
}

func (g *gatedDriver) Name() string                { return g.inner.Name() }
func (g *gatedDriver) Capabilities() CapabilitySet { return g.inner.Capabilities() }
func (g *gatedDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	g.once.Do(func() {
		close(g.started)
		<-g.resume
	})
	return g.inner.Generate(ctx, req)
}

// ── TestBeats_EmitsFramesInOrder ─────────────────────────────────────────────────────────────────

// TestBeats_EmitsFramesInOrder drives the real committed-beat happy path (Player tells Mara about
// the note — the same fixture TestBeat_HappyPath_CommitsAndNarrates uses) through a REAL
// httptest.Server (not httptest.ResponseRecorder — a recorder only exposes its buffer AFTER
// ServeHTTP returns, which cannot prove per-frame flushing). The narrate driver is gated: by the
// time we can read the interpretation frame off the wire, the handler is proven to be blocked
// several stages later (about to narrate), which is only possible if the interpretation frame was
// flushed the instant it was emitted rather than buffered until the response finished.
func TestBeats_EmitsFramesInOrder(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	baseTick := seatPlayerAndMara(t, ctx, pool)

	nd := newGatedDriver(NewFakeTextDriver("fake-text:test"))
	bridge := mustBridge(t,
		NewFakeStructuredDriver("fake-structured:test", map[string]string{
			"tell mara about the note": `[{"type":"Communicated","stated":"tell mara about the note","listener_id":"` + maraID + `","content":"the note"}]`,
		}),
		nd)
	h := NewBeatsStreamHandler(pool, true, bridge)
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/worlds/"+worldID+"/beats?viewer="+playerID,
		strings.NewReader(`{"text":"tell mara about the note"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	sr := newSSEReader(resp.Body)

	// Frame 1: interpretation. This read must complete BEFORE the handler proceeds through RunBeat
	// and reaches narrate — proving it, not asserting it, is the point of the next block.
	raw, err := sr.nextRaw()
	if err != nil {
		t.Fatalf("reading first frame: %v", err)
	}
	first := assertValidBeatFrame(t, raw)
	if first["kind"] != "interpretation" {
		t.Fatalf("first frame kind = %v, want interpretation", first["kind"])
	}

	// FLUSH PROOF: we already have the interpretation frame on our side of the wire. Confirm the
	// handler is now blocked several stages later, at the narrate driver's first call (past
	// RunBeat's commit, past building the post-beat payload) — only reachable if frame 1 was
	// flushed on emission rather than held in a buffer until the handler finished.
	select {
	case <-nd.started:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for the narrate driver to be reached — the handler never progressed past the interpretation frame")
	}
	close(nd.resume)

	kinds := []string{first["kind"].(string)}
	for {
		raw, err := sr.nextRaw()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading frame: %v", err)
		}
		frame := assertValidBeatFrame(t, raw)
		kinds = append(kinds, frame["kind"].(string))
	}

	if len(kinds) < 4 || kinds[0] != "interpretation" {
		t.Fatalf("frame kinds = %v, want interpretation first, then narration*, then scene/journey/result", kinds)
	}
	tail := kinds[len(kinds)-3:]
	if tail[0] != "scene" || tail[1] != "journey" || tail[2] != "result" {
		t.Fatalf("frame kinds tail = %v, want [scene journey result]; full sequence = %v", tail, kinds)
	}
	for _, k := range kinds[1 : len(kinds)-3] {
		if k != "narration" {
			t.Fatalf("frame kind %q between interpretation and scene, want only narration; full sequence = %v", k, kinds)
		}
	}
	if len(kinds) == 4 {
		t.Fatalf("no narration frame at all in a beat that committed and narrated: %v", kinds)
	}

	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// ── TestBeats_GhostSpeakerNeverReachesTheWire ────────────────────────────────────────────────────

// TestBeats_GhostSpeakerNeverReachesTheWire drives a narrate driver that ALWAYS authors a speaker
// who is not present (never a single valid reply): the structured belt (DecodeAndValidateNarration)
// rejects it on both the first attempt and the one repair, exactly as beathandler.go's own
// TestBeat_Narrate_GhostSpeakerRepairedOnSecondCall pins for the singular endpoint — the difference
// here is there is no valid repair, so the beat falls all the way to the unstructured plain-prose
// fallback (which carries no speaker at all). The belt still bit mid-stream: not one narration frame
// is ever attributed to the ghost.
func TestBeats_GhostSpeakerNeverReachesTheWire(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	baseTick := seatPlayerAndMara(t, ctx, pool)

	const ghostID = "99999999-9999-9999-9999-999999999999"
	ghost := `[{"speaker_id":"` + ghostID + `","kind":"action","text":"A stranger who is not here moves."}]`
	nd := &scriptedNarrateDriver{name: "scripted-narrate-ghost", replies: []string{ghost, ghost}}
	bridge := mustBridge(t, NewFakeStructuredDriver("fake-structured:test", nil), nd) // empty decompose table -> [] chain, beat completes
	h := NewBeatsStreamHandler(pool, true, bridge)

	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beats?viewer="+playerID, strings.NewReader(`{"text":"I look around"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sr := newSSEReader(bytes.NewReader(rec.Body.Bytes()))
	var narrationFrames []map[string]any
	for {
		raw, err := sr.nextRaw()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading frame: %v", err)
		}
		frame := assertValidBeatFrame(t, raw)
		if frame["kind"] == "narration" {
			narrationFrames = append(narrationFrames, frame)
		}
	}

	// 2 structured attempts (both ghost, both rejected) + 1 plain fallback call = 3.
	if nd.calls != 3 {
		t.Fatalf("narrate Generate calls = %d, want 3 (ghost -> one repair, still ghost -> plain fallback)", nd.calls)
	}

	for _, f := range narrationFrames {
		msg, _ := f["message"].(map[string]any)
		if sid, ok := msg["speaker_id"].(string); ok && sid == ghostID {
			t.Fatalf("the ghost speaker reached the wire in a narration frame: %+v", f)
		}
	}
	if len(narrationFrames) != 1 {
		t.Fatalf("want exactly 1 narration frame (the unattributed plain-prose fallback), got %d: %+v", len(narrationFrames), narrationFrames)
	}
	msg, _ := narrationFrames[0]["message"].(map[string]any)
	if msg["speaker_id"] != nil {
		t.Fatalf("fallback narration frame speaker_id = %v, want null (narrator, never the rejected ghost)", msg["speaker_id"])
	}

	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// ── TestBeats_NarrateFailureEmitsAnErrorFrame ────────────────────────────────────────────────────

// erroringNarrateDriver always fails Generate — simulating a narrate seat outage that survives both
// structured attempts AND the plain-prose fallback (beathandler.go's own last-resort path), so the
// streaming handler must choose an `error` frame rather than a 5xx (status 200 is already on the
// wire by the time narrate is even reached — the interpretation frame proves it).
type erroringNarrateDriver struct{ name string }

func (d *erroringNarrateDriver) Name() string                { return d.name }
func (d *erroringNarrateDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }
func (d *erroringNarrateDriver) Generate(context.Context, GenRequest) (string, error) {
	return "", fmt.Errorf("simulated narrate seat outage: connection reset by peer at 10.0.4.19:9443, goroutine 42")
}

func TestBeats_NarrateFailureEmitsAnErrorFrame(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	baseTick := seatPlayerAndMara(t, ctx, pool)

	bridge := mustBridge(t, NewFakeStructuredDriver("fake-structured:test", nil), &erroringNarrateDriver{name: "erroring-narrate"})
	h := NewBeatsStreamHandler(pool, true, bridge)

	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beats?viewer="+playerID, strings.NewReader(`{"text":"I look around"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// The belt fires per plan Task 3: once the interpretation frame is on the wire, status is
	// already 200 — a downstream failure can never become a 500 again.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (frames were already on the wire when narrate failed): %s", rec.Code, rec.Body.String())
	}

	sr := newSSEReader(bytes.NewReader(rec.Body.Bytes()))
	var frames []map[string]any
	for {
		raw, err := sr.nextRaw()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading frame: %v", err)
		}
		frames = append(frames, assertValidBeatFrame(t, raw))
	}

	if len(frames) != 2 {
		t.Fatalf("want exactly 2 frames (interpretation, error) — narrate failure must stop the stream cleanly, got %d: %+v", len(frames), frames)
	}
	if frames[0]["kind"] != "interpretation" {
		t.Fatalf("frames[0].kind = %v, want interpretation", frames[0]["kind"])
	}
	if frames[1]["kind"] != "error" {
		t.Fatalf("frames[1].kind = %v, want error (\"it just stops\" is the bug this design exists to prevent)", frames[1]["kind"])
	}
	msg, _ := frames[1]["message"].(string)
	if strings.TrimSpace(msg) == "" {
		t.Fatalf("error frame carries no message")
	}
	if strings.Contains(msg, "simulated narrate seat outage") || strings.Contains(msg, "goroutine") || strings.Contains(msg, "10.0.4.19") {
		t.Fatalf("error frame leaked engine internals (never a stack trace, never engine internals): %q", msg)
	}

	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}

// ── TestGenBeatFramePayloads — SPEC-011 real-payload generator ──────────────────────────────────

// TestGenBeatFramePayloads is the Go-side payload generator ci/gen_payloads.sh drives (mirroring
// TestGenSceneCurrentPayloads/SCENE_PAYLOAD_DIR and TestGenWorldActorPayload/SEAT_PAYLOAD_DIR):
// gated on BEAT_STREAM_PAYLOAD_DIR (skipped in the normal `go test ./...` suite, so it writes
// nothing by default), it drives the REAL committed-beat happy path through the REAL
// beatsStreamHandler.ServeHTTP and writes every frame's RAW bytes (exactly what frameWriter.emit
// wrote to the wire — no re-marshal, no hand-written fixture) so `make schema-contract` has real
// interpretation/narration/scene/journey/result payloads to validate against beat_frame.v1.schema.json.
func TestGenBeatFramePayloads(t *testing.T) {
	dir := os.Getenv("BEAT_STREAM_PAYLOAD_DIR")
	if dir == "" {
		t.Skip("BEAT_STREAM_PAYLOAD_DIR unset — this generator only runs from ci/gen_payloads.sh")
	}
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	baseTick := seatPlayerAndMara(t, ctx, pool)

	bridge := mustBridge(t,
		NewFakeStructuredDriver("fake-structured:test", map[string]string{
			"tell mara about the note": `[{"type":"Communicated","stated":"tell mara about the note","listener_id":"` + maraID + `","content":"the note"}]`,
		}),
		NewFakeTextDriver("fake-text:test"))
	h := NewBeatsStreamHandler(pool, true, bridge)

	req := httptest.NewRequest(http.MethodPost, "/worlds/"+worldID+"/beats?viewer="+playerID, strings.NewReader(`{"text":"tell mara about the note"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	sr := newSSEReader(bytes.NewReader(rec.Body.Bytes()))
	n := 0
	for {
		raw, err := sr.nextRaw()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading frame: %v", err)
		}
		var kindProbe struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(raw, &kindProbe); err != nil {
			t.Fatalf("frame %d: %v", n, err)
		}
		name := fmt.Sprintf("beat_frame_%s_%d.json", kindProbe.Kind, n)
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatalf("%s: write: %v", name, err)
		}
		n++
	}
	if n == 0 {
		t.Fatalf("no frames captured — the beat produced nothing to validate")
	}

	perceptionSubjectBackfill(t, ctx, pool, baseTick)
}
