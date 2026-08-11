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
	"strconv"
	"strings"
	"time"
)

type openAICompatDriver struct {
	name    string
	model   string
	baseURL string
	apiKey  string
	// jsonMode is how a structured request is constrained on the wire. Two values, both real:
	//
	//   json_object  universal. Every provider that speaks this dialect supports it. The schema
	//                travels in the system message and the model is told to answer with one JSON
	//                document; the Go validator decides whether it did.
	//   json_schema  strict, where the provider implements it: the schema goes in the request as a
	//                first-class field and the provider constrains decoding to it. Strictly better
	//                when available, and a 400 from a provider that does not implement it — which is
	//                why it is opt-in per provider rather than assumed.
	//
	// Tool-call forcing is deliberately NOT a third path here. It is the anthropic driver's leash
	// because that dialect has no JSON mode; for this dialect it would reach the same place by a
	// longer road, and an unexercised third branch on the seat path is exactly the kind of surface
	// that hides a defect until a live beat finds it.
	jsonMode string
	// routing is the aggregator's provider-preferences object, already validated as JSON at
	// construction and sent verbatim under "provider" on every request. nil when unset, in which
	// case the field is omitted entirely rather than sent empty — a plain provider does not know
	// what "provider" means and some reject unknown fields.
	routing json.RawMessage
	// reasoning is the aggregator's reasoning-control object, validated as JSON at construction and
	// sent verbatim. Unset leaves the MODEL'S OWN default in force, and that default is not neutral:
	// the mechanical model here advertises default_effort "high", which spends roughly 80% of
	// max_tokens thinking before a token of answer exists. Measured live, that is exactly how
	// decompose died twice in front of the founder — finish_reason "length", content null, the whole
	// 1024-token budget consumed by reasoning. With {"effort":"none"} the same call returns 0
	// reasoning tokens, 54 completion tokens and a valid chain.
	reasoning json.RawMessage
	// maxTokens is the completion budget sent on every request. NOT optional against an aggregator:
	// with no max_tokens it reserves the MODEL'S FULL completion window up front and refuses the
	// call if the account cannot afford that reservation — measured live, a request with no cap was
	// rejected 402 "you requested up to 65536 tokens, but can only afford 11478" on an account with
	// money in it. It is also the only ceiling on a reasoning model's spend: the strong DeepSeek
	// variant emits reasoning tokens billed at the completion rate before it writes a word.
	maxTokens int
	// temperature is sent only when configured; nil leaves the provider's default in place.
	//
	// It is configuration rather than a constant because the right value is a SEAT property, and the
	// seats genuinely differ: decompose and resolve turn a sentence into a classification and want no
	// sampling variety at all, while narrate is a performance and would be duller for it. Measured,
	// twice, on the same input with the default temperature: the founder's "I look around, who is
	// there?" decomposed once to QUERY committing nothing, and once to ActorMoved committing three
	// canon events — a question that moved the player. A mechanical seat left at a sampling default
	// is a coin flip wearing a schema.
	temperature *float64
	client      *http.Client
}

// defaultMaxTokens is deliberately generous enough for a narration segment plus a reasoning
// preamble, and far below any model's window. A seat that needs more says so in config.
const defaultMaxTokens = 2048

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
	jsonMode := dc.Params["json_mode"]
	if jsonMode == "" {
		jsonMode = jsonModeObject
	}
	if jsonMode != jsonModeObject && jsonMode != jsonModeSchema && jsonMode != jsonModeOff {
		return nil, fmt.Errorf("openai-compat: unknown json_mode %q (known: %s, %s, %s)",
			jsonMode, jsonModeObject, jsonModeSchema, jsonModeOff)
	}
	// The name is what every log line, error and trace frame says about who answered. With four
	// seats on four providers, "openai-compat:<model>" alone cannot tell you which one failed.
	alias := dc.Params["provider_alias"]
	if alias == "" {
		alias = "openai-compat"
	}
	// Validated HERE so a malformed policy is a boot failure with the seat named, not a 400 on the
	// first beat of a founder playtest. Compliance config that fails late is compliance config that
	// fails in front of the person it was meant to protect.
	maxTokens := defaultMaxTokens
	if v := strings.TrimSpace(dc.Params["max_tokens"]); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("openai-compat: max_tokens %q is not a positive integer", v)
		}
		maxTokens = n
	}
	var temperature *float64
	if v := strings.TrimSpace(dc.Params["temperature"]); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 0 {
			return nil, fmt.Errorf("openai-compat: temperature %q is not a non-negative number", v)
		}
		temperature = &f
	}
	var routing json.RawMessage
	if r := strings.TrimSpace(dc.Params["routing"]); r != "" {
		if !json.Valid([]byte(r)) {
			return nil, fmt.Errorf("openai-compat: routing policy is not valid JSON: %.80s", r)
		}
		routing = json.RawMessage(r)
	}
	var reasoning json.RawMessage
	if r := strings.TrimSpace(dc.Params["reasoning"]); r != "" {
		if !json.Valid([]byte(r)) {
			return nil, fmt.Errorf("openai-compat: reasoning policy is not valid JSON: %.80s", r)
		}
		reasoning = json.RawMessage(r)
	}
	return &openAICompatDriver{
		name:        alias + ":" + model,
		model:       model,
		baseURL:     baseURL,
		apiKey:      apiKey,
		jsonMode:    jsonMode,
		routing:     routing,
		reasoning:   reasoning,
		maxTokens:   maxTokens,
		temperature: temperature,
		client:      &http.Client{Timeout: 60 * time.Second},
	}, nil
}

const (
	jsonModeObject = "json_object"
	jsonModeSchema = "json_schema"
	// jsonModeOff sends NO response_format at all: the schema rides the system message, as it does in
	// every mode, and the Go belt is the only enforcement. It exists because strict decoding is not
	// universally better and narration/1 proved it — measured live, that schema under json_schema
	// came back with every segment structurally perfect and textually EMPTY ("3 blank", "6 blank",
	// 59-character replies that were pure shape), three times a beat, because a constrained decoder
	// that cannot satisfy the item-level oneOf still has to emit the required keys and fills them
	// with "". Constrained decoding guarantees a shape, which is exactly worthless when the shape was
	// never the hard part; the belt has always been what decides acceptance.
	jsonModeOff = "off"
)

func (o *openAICompatDriver) Name() string { return o.name }

// Capabilities reports CapStructuredOutput because this driver ENFORCES it, which is the distinction
// bridge.go insists on: a capability is a report of what a driver does, never a label from config.
// What it does is send the schema as the leash (in the request under json_schema, in the system
// message under json_object), and the Go validator plus the caller's retry decide acceptance. A
// provider that ignores the flag entirely does not slip through — it fails validation and the seat
// rejects it, which is the same trust class as the anthropic driver's tool-use leash (D-13).
func (o *openAICompatDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}

func (o *openAICompatDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	body := map[string]any{
		"model":      o.model,
		"max_tokens": o.maxTokens,
	}
	if o.temperature != nil {
		body["temperature"] = *o.temperature
	}
	if o.reasoning != nil {
		body["reasoning"] = o.reasoning
	}
	// The routing policy rides EVERY request, structured or not: a free-text narration is exactly as
	// subject to "which jurisdiction, which retention policy" as a schema'd one.
	if o.routing != nil {
		body["provider"] = o.routing
	}

	if req.Schema != nil {
		// Structured: system message with schema leash + user message; constrain to json_object.
		// "a single JSON document" is ambiguous for an array-rooted schema, and models resolve the
		// ambiguity by reaching for an object: measured live, this exact prompt with narration/1 came
		// back as {"array":[…]} — correct prose inside a wrapper nothing asked for. Naming the root
		// costs one word and removes the guess.
		root := "JSON document"
		if schemaRootIsArray(req.Schema) {
			root = "JSON ARRAY (a bare [ … ], not an object wrapping one)"
		}
		systemContent := "You are a quarantined seat in a play loop. Propose only; never assert canon. Answer ONLY with a single " +
			root + " valid against this JSON Schema:\n" + string(req.Schema)
		body["messages"] = []any{
			map[string]any{"role": "system", "content": systemContent},
			map[string]any{"role": "user", "content": req.Prompt},
		}
		// jsonModeOff: the leash is the system message plus the belt, and nothing constrains decoding.
		if o.jsonMode == jsonModeOff {
			raw, err := o.post(ctx, body)
			if err != nil {
				return "", err
			}
			return o.content(raw)
		}
		// json_object CANNOT express an array-rooted schema — the mode's whole contract is "the reply
		// is a JSON object". Measured live: beat_chain/2 is array-rooted, and under json_object the
		// model returned a bare {"type":"QUERY",…} instead of [{…}], which the belt then rejected as
		// "outside the closed vocabulary" — an error that blames the model's vocabulary for what is
		// a response-format mismatch, and cost a live debugging session to see through. Refuse the
		// combination instead of sending a request that cannot succeed.
		if o.jsonMode == jsonModeObject && schemaRootIsArray(req.Schema) {
			return "", fmt.Errorf("openai-compat: this seat's schema is array-rooted and json_mode is "+
				"%s, which mandates an object — set JSON_MODE=%s for this provider", jsonModeObject, jsonModeSchema)
		}
		if o.jsonMode == jsonModeSchema {
			// The provider constrains decoding itself. The system-message leash above stays anyway:
			// it costs a few tokens and it is what the model reads if the provider's strict mode is
			// advisory rather than enforced, which is not something we can tell from out here.
			body["response_format"] = map[string]any{
				"type": jsonModeSchema,
				"json_schema": map[string]any{
					"name":   "seat_output",
					"schema": json.RawMessage(req.Schema),
					"strict": true,
				},
			}
		} else {
			body["response_format"] = map[string]any{"type": jsonModeObject}
		}
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
	return o.content(raw)
}

// content extracts choices[0].message.content verbatim. One place, because every json_mode returns
// through it and a second copy is a second chance to disagree about what an empty reply means.
func (o *openAICompatDriver) content(raw []byte) (string, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("openai-compat: parse response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai-compat: empty choices in response")
	}
	// A reasoning model can spend the whole completion budget thinking and return NULL content with
	// finish_reason "length". Returning "" there would surface as a schema validation failure two
	// retries later, blaming the model for a budget problem. Name it instead.
	if resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("openai-compat: model returned no content (finish_reason=%q) — "+
			"raise max_tokens (currently %d) if this is a reasoning model",
			resp.Choices[0].FinishReason, o.maxTokens)
	}
	return stripCodeFence(resp.Choices[0].Message.Content), nil
}

// stripCodeFence removes a ```…``` wrapper when a model volunteers one. Without constrained decoding
// some replies arrive fenced and some do not — measured live, two identical narrate calls, one bare
// array and one wrapped in ```json — and a fence is a MARKDOWN transport wrapper, not content: the
// document inside is exactly what was asked for. Stripping it is not leniency about what the model
// said, which is still the belt's business; it is refusing to fail over punctuation.
func stripCodeFence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return s
	}
	// Drop the opening fence and its optional language tag, then the closing fence.
	if i := strings.IndexByte(t, '\n'); i >= 0 {
		t = t[i+1:]
	} else {
		return s
	}
	if j := strings.LastIndex(t, "```"); j >= 0 {
		t = t[:j]
	}
	return strings.TrimSpace(t)
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

// schemaRootIsArray reports whether a JSON Schema describes an array at its root. Only the root
// "type" is inspected: that is the one fact json_object mode conflicts with, and a deeper walk would
// be guessing at intent.
func schemaRootIsArray(schema json.RawMessage) bool {
	var head struct {
		Type any `json:"type"`
	}
	if err := json.Unmarshal(schema, &head); err != nil {
		return false
	}
	switch t := head.Type.(type) {
	case string:
		return t == "array"
	case []any: // a union such as ["array","null"]
		for _, v := range t {
			if s, ok := v.(string); ok && s == "array" {
				return true
			}
		}
	}
	return false
}
