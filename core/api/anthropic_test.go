package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// anthropicCapturingRT records the outgoing request and returns a canned response, so the test
// exercises the real Generate path (schema wrap + response unwrap) with no network call.
type anthropicCapturingRT struct {
	capturedBody []byte
	response     string
}

func (f *anthropicCapturingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		f.capturedBody, _ = io.ReadAll(req.Body)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(f.response)),
		Header:     make(http.Header),
	}, nil
}

// TestAnthropicSchemaWrappedAsObject guards the live decompose path found broken at the chunk-5
// operator gate: the array-root beat_chain schema must be nested under an object envelope before it
// reaches Anthropic's tool input_schema (a non-object root is a 400), and the model's {"chain":[...]}
// must come back unwrapped to the bare array DecodeAndValidateChain expects.
func TestAnthropicSchemaWrappedAsObject(t *testing.T) {
	arraySchema := []byte(`{"type":"array","items":{"type":"object"}}`) // the real root shape that broke it

	fake := &anthropicCapturingRT{
		response: `{"content":[{"type":"tool_use","name":"emit_beat_chain","input":{"chain":[{"type":"move","to":"market"}]}}]}`,
	}
	drv := &anthropicDriver{
		model:  "claude-opus-4-8",
		apiKey: "test-key", // non-empty so Generate doesn't short-circuit on the key check
		client: &http.Client{Transport: fake},
	}

	out, err := drv.Generate(context.Background(), GenRequest{
		Prompt: "I head over to the market.",
		Schema: arraySchema,
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// 1) THE GUARD: the input_schema sent to Anthropic must have an OBJECT root.
	var sent struct {
		Tools []struct {
			InputSchema struct {
				Type       string         `json:"type"`
				Properties map[string]any `json:"properties"`
			} `json:"input_schema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(fake.capturedBody, &sent); err != nil {
		t.Fatalf("could not parse captured request body: %v", err)
	}
	if len(sent.Tools) != 1 {
		t.Fatalf("expected 1 tool in request, got %d", len(sent.Tools))
	}
	if got := sent.Tools[0].InputSchema.Type; got != "object" {
		t.Fatalf("input_schema.type = %q, want \"object\" (an array root is exactly the bug this guards)", got)
	}
	if _, ok := sent.Tools[0].InputSchema.Properties["chain"]; !ok {
		t.Fatalf("input_schema must nest the vocabulary under a \"chain\" property; got %v", sent.Tools[0].InputSchema.Properties)
	}

	// 2) ROUND-TRIP: the model's {"chain":[...]} must come back as a bare array, not the envelope.
	var chain []map[string]any
	if err := json.Unmarshal([]byte(out), &chain); err != nil {
		t.Fatalf("Generate output is not a bare JSON array — unwrap failed: %v (got %q)", err, out)
	}
	if len(chain) != 1 || chain[0]["type"] != "move" {
		t.Fatalf("unexpected unwrapped chain: %q", out)
	}
}