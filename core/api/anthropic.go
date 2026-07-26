package main

// anthropicDriver — Claude via the Anthropic Messages API, the production default driver. STRUCTURED
// requests use a single tool whose input_schema is the caller's schema, FORCED via tool_choice — the
// model's tool_use input is schema-constrained BY CONSTRUCTION (the decompose leash, generation-time;
// ADR-009/D-1, SPEC-015). Free-text requests are an ordinary message (the narrate seat). This lives
// only in the bridge layer (D-13) — the canon engine (core/db) never imports a provider.
//
// NOT exercised in CI: it requires ANTHROPIC_API_KEY and makes a network call. The live model is
// wired separately (operator gate / production); CI uses the deterministic fake drivers. This file is
// compiled + vetted, never invoked by a test.

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
const anthropicVersion = "2023-06-01"

// anthropicSystemHeader is the driver-owned system field sent on every anthropic call: this is a
// quarantined seat, propose-only, never-assert-canon (D-13 keeps provider shaping in the driver).
// Text lives in prompts/system-anthropic.txt (core/api/prompts/README.md) — still driver-owned, but
// now readable alongside the rest of the fixed prompt rulebooks.
//
//go:embed prompts/system-anthropic.txt
var anthropicSystemHeader string

type anthropicDriver struct {
	model  string
	apiKey string
	client *http.Client
}

func newAnthropicDriver(dc DriverConfig) (Driver, error) {
	model := dc.Model
	if model == "" {
		model = "claude-opus-4-8"
	}
	return &anthropicDriver{
		model:  model,
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		client: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (a *anthropicDriver) Name() string { return "anthropic:" + a.model }

// Reports structured output: the binding contract is satisfied via forced tool-use below. A seat with
// CapStructuredOutput required (decompose) therefore binds; the actual constraint is applied per call.
func (a *anthropicDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (a *anthropicDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	if a.apiKey == "" {
		return "", fmt.Errorf("anthropic: ANTHROPIC_API_KEY not set (live model is wired separately, out of CI)")
	}
	body := map[string]any{
		"model":      a.model,
		"max_tokens": 1024,
		"system":     anthropicSystemHeader,
		"messages":   []any{map[string]any{"role": "user", "content": req.Prompt}},
	}
	if req.Schema != nil {
		var schema any
		if err := json.Unmarshal(req.Schema, &schema); err != nil {
			return "", fmt.Errorf("anthropic: bad schema: %w", err)
		}
		// Anthropic requires an OBJECT-root tool input_schema; the closed-vocabulary chain is an array,
		// so nest it under "chain" for the wire and unwrap back to the bare array below. The leash is
		// unchanged (forced tool_use → schema-valid by construction); the shared schema and
		// DecodeAndValidateChain stay untouched. Provider shaping lives in the driver (D-13).
		wrapped := map[string]any{
			"type":                 "object",
			"properties":           map[string]any{"chain": schema},
			"required":             []any{"chain"},
			"additionalProperties": false,
		}
		body["tools"] = []any{map[string]any{
			"name":         "emit_beat_chain",
			"description":  "Emit the player's beat as an ordered chain of closed-vocabulary events.",
			"input_schema": wrapped,
		}}
		body["tool_choice"] = map[string]any{"type": "tool", "name": "emit_beat_chain"}
	}

	resp, err := a.post(ctx, body)
	if err != nil {
		return "", err
	}
if req.Schema != nil {
		input, err := extractToolInput(resp, "emit_beat_chain")
		if err != nil {
			return "", err
		}
		// unwrap {"chain":[...]} → the bare array DecodeAndValidateChain expects
		var env struct {
			Chain json.RawMessage `json:"chain"`
		}
		if err := json.Unmarshal([]byte(input), &env); err != nil {
			return "", fmt.Errorf("anthropic: unwrap chain: %w", err)
		}
		if len(env.Chain) == 0 {
			return "", fmt.Errorf("anthropic: tool input missing chain")
		}
		return string(env.Chain), nil
	}
	return extractText(resp), nil
}

func (a *anthropicDriver) post(ctx context.Context, body map[string]any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicMessagesURL, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	res, err := a.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic: status %d: %s", res.StatusCode, string(out))
	}
	return out, nil
}

// anthropicResponse is the subset of the Messages API response we read.
type anthropicResponse struct {
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
}

// extractToolInput returns the JSON input of the forced tool_use block (the schema-constrained chain).
func extractToolInput(raw []byte, toolName string) (string, error) {
	var r anthropicResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("anthropic: parse: %w", err)
	}
	for _, c := range r.Content {
		if c.Type == "tool_use" && c.Name == toolName {
			return string(c.Input), nil
		}
	}
	return "", fmt.Errorf("anthropic: no tool_use %q in response", toolName)
}

func extractText(raw []byte) string {
	var r anthropicResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return ""
	}
	out := ""
	for _, c := range r.Content {
		if c.Type == "text" {
			out += c.Text
		}
	}
	return out
}
