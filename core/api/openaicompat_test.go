package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cannedChatCompletionsBody returns a minimal OpenAI-compatible chat/completions response.
func cannedChatCompletionsBody(content string) string {
	body, _ := json.Marshal(map[string]any{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": content}}},
	})
	return string(body)
}

// TestOpenAICompat_WithSchema asserts:
//   - Authorization: Bearer <key> header is set
//   - "model" field matches configured model
//   - "response_format.type" == "json_object" is present
//   - system message contains the schema text
//   - user message content equals the prompt
//   - driver returns choices[0].message.content verbatim
func TestOpenAICompat_WithSchema(t *testing.T) {
	const (
		wantModel  = "deepseek-chat"
		wantAPIKey = "sk-test-key"
		wantPrompt = "What just happened?"
	)
	schema := json.RawMessage(`{"type":"object","properties":{"ruling":{"type":"string"}}}`)
	wantContent := `{"ruling":"success"}`

	var capturedBody []byte
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cannedChatCompletionsBody(wantContent)))
	}))
	defer srv.Close()

	dc := DriverConfig{
		Provider: "openai-compat",
		Model:    wantModel,
		Params: map[string]string{
			"base_url": srv.URL,
			"api_key":  wantAPIKey,
		},
	}
	drv, err := newOpenAICompatDriver(dc)
	if err != nil {
		t.Fatalf("newOpenAICompatDriver: %v", err)
	}

	out, err := drv.Generate(context.Background(), GenRequest{
		Prompt: wantPrompt,
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// 1. Auth header
	wantBearer := "Bearer " + wantAPIKey
	if capturedAuth != wantBearer {
		t.Errorf("Authorization header = %q, want %q", capturedAuth, wantBearer)
	}

	// 2. Parse captured body
	var body struct {
		Model          string `json:"model"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("parse captured body: %v", err)
	}

	// 3. Model field
	if body.Model != wantModel {
		t.Errorf("model = %q, want %q", body.Model, wantModel)
	}

	// 4. response_format.type == "json_object"
	if body.ResponseFormat.Type != "json_object" {
		t.Errorf("response_format.type = %q, want \"json_object\"", body.ResponseFormat.Type)
	}

	// 5. Two messages: system + user
	if len(body.Messages) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(body.Messages))
	}
	if body.Messages[0].Role != "system" {
		t.Errorf("messages[0].role = %q, want \"system\"", body.Messages[0].Role)
	}
	if !strings.Contains(body.Messages[0].Content, string(schema)) {
		t.Errorf("system message should contain the schema text, got: %q", body.Messages[0].Content)
	}
	if body.Messages[1].Role != "user" {
		t.Errorf("messages[1].role = %q, want \"user\"", body.Messages[1].Role)
	}
	if body.Messages[1].Content != wantPrompt {
		t.Errorf("messages[1].content = %q, want %q", body.Messages[1].Content, wantPrompt)
	}

	// 6. Output is verbatim content
	if out != wantContent {
		t.Errorf("Generate output = %q, want %q", out, wantContent)
	}
}

// TestOpenAICompat_NoSchema asserts that when Schema is nil:
//   - no response_format field is sent
//   - only a single user message is sent (no system message)
//   - output is verbatim
func TestOpenAICompat_NoSchema(t *testing.T) {
	const wantPrompt = "Describe the scene."
	wantContent := "The tavern is dimly lit."

	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cannedChatCompletionsBody(wantContent)))
	}))
	defer srv.Close()

	dc := DriverConfig{
		Provider: "openai-compat",
		Model:    "some-model",
		Params: map[string]string{
			"base_url": srv.URL,
			"api_key":  "sk-test",
		},
	}
	drv, err := newOpenAICompatDriver(dc)
	if err != nil {
		t.Fatalf("newOpenAICompatDriver: %v", err)
	}

	out, err := drv.Generate(context.Background(), GenRequest{
		Prompt: wantPrompt,
		Schema: nil,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Parse to check no response_format and single user message
	var body map[string]json.RawMessage
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("parse captured body: %v", err)
	}
	if _, ok := body["response_format"]; ok {
		t.Error("response_format should not be present when Schema is nil")
	}

	var messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body["messages"], &messages); err != nil {
		t.Fatalf("parse messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message (user only), got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Errorf("messages[0].role = %q, want \"user\"", messages[0].Role)
	}
	if messages[0].Content != wantPrompt {
		t.Errorf("messages[0].content = %q, want %q", messages[0].Content, wantPrompt)
	}
	if out != wantContent {
		t.Errorf("Generate output = %q, want %q", out, wantContent)
	}
}

// TestOpenAICompat_Non2xx asserts that a non-200 response becomes an error containing the status.
func TestOpenAICompat_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	dc := DriverConfig{
		Provider: "openai-compat",
		Model:    "m",
		Params: map[string]string{
			"base_url": srv.URL,
			"api_key":  "sk-test",
		},
	}
	drv, err := newOpenAICompatDriver(dc)
	if err != nil {
		t.Fatalf("newOpenAICompatDriver: %v", err)
	}

	_, err = drv.Generate(context.Background(), GenRequest{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error for non-2xx response, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should contain status 400, got: %v", err)
	}
}

// TestOpenAICompat_BindSeatResolvePassesCapabilityFloor asserts that an openai-compat driver
// satisfies the SeatResolve capability floor (CapStructuredOutput).
func TestOpenAICompat_BindSeatResolvePassesCapabilityFloor(t *testing.T) {
	// Use a minimal server; we only need to construct the driver (no Generate called).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	dc := DriverConfig{
		Provider: "openai-compat",
		Model:    "deepseek-chat",
		Params: map[string]string{
			"base_url": srv.URL,
			"api_key":  "sk-test",
		},
	}
	drv, err := newOpenAICompatDriver(dc)
	if err != nil {
		t.Fatalf("newOpenAICompatDriver: %v", err)
	}

	if _, err := BindSeat(SeatResolve, drv); err != nil {
		t.Fatalf("BindSeat(SeatResolve, openai-compat driver) failed — capability floor not met: %v", err)
	}
}

// TestOpenAICompat_DefaultDriverFactory asserts "openai-compat" is registered in DefaultDriverFactory.
func TestOpenAICompat_DefaultDriverFactory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	dc := DriverConfig{
		Provider: "openai-compat",
		Model:    "test-model",
		Params: map[string]string{
			"base_url": srv.URL,
			"api_key":  "sk-test",
		},
	}
	drv, err := DefaultDriverFactory(dc)
	if err != nil {
		t.Fatalf("DefaultDriverFactory(openai-compat): %v", err)
	}
	if drv == nil {
		t.Fatal("DefaultDriverFactory(openai-compat) returned nil driver")
	}
}
