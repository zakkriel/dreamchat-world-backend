package main

// openAICompatDriver — a generic OpenAI-compatible chat/completions driver. Any provider that speaks
// the OpenAI chat/completions wire format (DeepInfra, DeepSeek, OpenRouter, self-hosted vLLM, etc.)
// can serve a seat via env config — model-agnostic per-seat routing (D-13/ADR-P018).
//
// STRUCTURED requests (Schema != nil) send a system message with the JSON Schema leash and
// response_format.type=json_object, constraining the model's output. The json_object flag, the
// Go belt (DecodeAndValidateChain), and the repair path form the enforcement chain — same trust class
// as everywhere else in this bridge (the schema is the leash; generation-time for providers that
// support constrained decoding, post-hoc validation for those that don't).
//
// NOT exercised in CI without a live provider; use DREAMCHAT_BRIDGE=fake for keyless local dev.
// This file is compiled + vetted; the live model is wired at the operator gate.
//
// Per-seat config is carried via DriverConfig.Params:
//   "base_url" — e.g. "https://api.deepinfra.com/v1/openai"
//   "api_key"  — bearer token

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type openAICompatDriver struct {
	name    string
	model   string
	baseURL string
	apiKey  string
	client  *http.Client
}

// newOpenAICompatDriver constructs an openai-compat driver from DriverConfig.
// base_url and api_key are read from dc.Params; model from dc.Model.
func newOpenAICompatDriver(dc DriverConfig) (Driver, error) {
	baseURL := ""
	apiKey := ""
	if dc.Params != nil {
		baseURL = dc.Params["base_url"]
		apiKey = dc.Params["api_key"]
	}
	if baseURL == "" {
		return nil, fmt.Errorf("openai-compat: Params[\"base_url\"] is required")
	}
	model := dc.Model
	if model == "" {
		return nil, fmt.Errorf("openai-compat: Model is required")
	}
	return &openAICompatDriver{
		name:    "openai-compat:" + model,
		model:   model,
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (o *openAICompatDriver) Name() string { return o.name }

// Capabilities reports CapStructuredOutput: the json_object flag + the Go validation belt + the
// repair path form the enforcement chain — same trust class as the anthropic driver (D-13).
func (o *openAICompatDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}

func (o *openAICompatDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	body := map[string]any{
		"model": o.model,
	}

	if req.Schema != nil {
		// Structured: system message with schema leash + user message; constrain to json_object.
		systemContent := "You are a quarantined seat in a play loop. Propose only; never assert canon. Answer ONLY with a single JSON document valid against this JSON Schema:\n" + string(req.Schema)
		body["messages"] = []any{
			map[string]any{"role": "system", "content": systemContent},
			map[string]any{"role": "user", "content": req.Prompt},
		}
		body["response_format"] = map[string]any{"type": "json_object"}
	} else {
		// Free-text: single user message, no response_format.
		body["messages"] = []any{
			map[string]any{"role": "user", "content": req.Prompt},
		}
	}

	raw, err := o.post(ctx, body)
	if err != nil {
		return "", err
	}

	// Parse choices[0].message.content and return verbatim.
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("openai-compat: parse response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai-compat: empty choices in response")
	}
	return resp.Choices[0].Message.Content, nil
}

func (o *openAICompatDriver) post(ctx context.Context, body map[string]any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai-compat: marshal: %w", err)
	}
	url := o.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("openai-compat: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	res, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai-compat: request: %w", err)
	}
	defer res.Body.Close()

	out, err := io.ReadAll(io.LimitReader(res.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, fmt.Errorf("openai-compat: read body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		snippet := string(out)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return nil, fmt.Errorf("openai-compat: status %d: %s", res.StatusCode, snippet)
	}
	return out, nil
}
