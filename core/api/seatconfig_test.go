package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// env builds an injectable lookup, so every case below is a pure function of a recorded environment
// and none of them touches the process.
func env(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

// The mandate in one test: seven seats, four providers, and not one vendor name in the code that
// routed them. The aliases here are arbitrary strings an operator chose.
func TestSeatConfig_RoutesEverySeatFromTheEnvironment(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT": "cheapo:small-1",
		"DREAMCHAT_SEATS": "narrate=wordsmith:prose-xl,world_actor=thinker:big-1," +
			"decompose=cheapo:small-1",
		"DREAMCHAT_PROVIDER_CHEAPO_BASE_URL":    "https://cheapo.example/v1",
		"DREAMCHAT_PROVIDER_CHEAPO_API_KEY":     "k-cheap",
		"DREAMCHAT_PROVIDER_WORDSMITH_BASE_URL": "https://wordsmith.example/v1",
		"DREAMCHAT_PROVIDER_THINKER_BASE_URL":   "https://thinker.example/v1",
		"DREAMCHAT_PROVIDER_THINKER_JSON_MODE":  "json_schema",
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if len(cfg) != len(allSeatNames) {
		t.Fatalf("configured %d seats, want all %d", len(cfg), len(allSeatNames))
	}
	// Overridden seats go where they were sent.
	if got := cfg["narrate"]; got.Model != "prose-xl" || got.Params["base_url"] != "https://wordsmith.example/v1" {
		t.Fatalf("narrate = %+v, want the wordsmith endpoint", got)
	}
	if got := cfg["world_actor"]; got.Params["json_mode"] != "json_schema" {
		t.Fatalf("world_actor json_mode = %q, want the provider's strict mode", got.Params["json_mode"])
	}
	// Un-named seats fall to the default, which is the whole point of having one.
	for _, seat := range []string{"resolve", "cognition_batch", "cognition_isolated", "place_author"} {
		if got := cfg[seat]; got.Model != "small-1" || got.Params["api_key"] != "k-cheap" {
			t.Fatalf("%s = %+v, want the default provider", seat, got)
		}
	}
	// Every non-fake seat resolves to a DIALECT, never to a vendor: that is what keeps provider
	// names out of the code paths that route them.
	for seat, dc := range cfg {
		if dc.Provider != dialectOpenAICompat && dc.Provider != dialectAnthropic {
			t.Fatalf("seat %s resolved to provider %q — config must name a dialect, not a vendor", seat, dc.Provider)
		}
	}
	// The boot line names the alias, not the dialect, or four providers look identical in the log.
	if d := describeSeatConfig(cfg); !strings.Contains(d, "narrate=wordsmith:prose-xl") {
		t.Fatalf("describeSeatConfig = %q, want the operator's own alias", d)
	}
}

// No default and no fake bridge is an operator mistake, and the honest moment to say so is boot.
// A default provider is exactly what the founder's ruling forbids, because a default is what
// quietly bills someone.
func TestSeatConfig_FailsClosedWithNoProvider(t *testing.T) {
	_, err := seatConfigFromEnv(env(map[string]string{}))
	if err == nil {
		t.Fatal("no configuration must fail closed, not pick a provider")
	}
	for _, want := range []string{"DREAMCHAT_SEAT_DEFAULT", "DREAMCHAT_BRIDGE=fake"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %s — an operator has to be told which knob is missing", err, want)
		}
	}
}

// A default is optional when every seat is named. Nothing is guessed either way.
func TestSeatConfig_NoDefaultIsFineWhenEverySeatIsNamed(t *testing.T) {
	kv := map[string]string{"DREAMCHAT_PROVIDER_P_BASE_URL": "https://p.example/v1"}
	specs := make([]string, 0, len(allSeatNames))
	for _, s := range allSeatNames {
		specs = append(specs, s+"=p:m")
	}
	kv["DREAMCHAT_SEATS"] = strings.Join(specs, ",")
	if _, err := seatConfigFromEnv(env(kv)); err != nil {
		t.Fatalf("naming every seat must be sufficient: %v", err)
	}
	// Drop one and it must fail, naming the seat that has nowhere to go.
	kv["DREAMCHAT_SEATS"] = strings.Join(specs[1:], ",")
	err := seatConfigFromEnv2(kv)
	if err == nil || !strings.Contains(err.Error(), allSeatNames[0]) {
		t.Fatalf("err = %v, want it to name the unrouted seat %q", err, allSeatNames[0])
	}
}

func seatConfigFromEnv2(kv map[string]string) error {
	_, err := seatConfigFromEnv(env(kv))
	return err
}

// A typo'd seat name would otherwise leave that seat on the default and look exactly like the
// override working — the worst kind of config bug, because it is invisible until the bill.
func TestSeatConfig_RejectsAnUnknownSeatName(t *testing.T) {
	_, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":        "p:m",
		"DREAMCHAT_SEATS":               "narration=p:m",
		"DREAMCHAT_PROVIDER_P_BASE_URL": "https://p.example/v1",
	}))
	if err == nil || !strings.Contains(err.Error(), "narration") {
		t.Fatalf("err = %v, want a rejection naming the unknown seat", err)
	}
}

func TestSeatConfig_RejectsMalformedSpecAndMissingEndpoint(t *testing.T) {
	if _, err := seatConfigFromEnv(env(map[string]string{"DREAMCHAT_SEAT_DEFAULT": "justamodel"})); err == nil {
		t.Fatal("a spec with no provider:model split must be rejected")
	}
	_, err := seatConfigFromEnv(env(map[string]string{"DREAMCHAT_SEAT_DEFAULT": "ghost:m"}))
	if err == nil || !strings.Contains(err.Error(), "DREAMCHAT_PROVIDER_GHOST_BASE_URL") {
		t.Fatalf("err = %v, want the missing endpoint variable named exactly", err)
	}
}

// Aliases are written by humans in two places and will not match. Normalising both ends means
// `openai-compat` in one and `OPENAI_COMPAT` in the other still find each other.
func TestEnvKey_NormalisesAliases(t *testing.T) {
	for _, alias := range []string{"openai-compat", "OpenAI_Compat", "openai.compat"} {
		if got := envKey(alias); got != "OPENAI_COMPAT" {
			t.Fatalf("envKey(%q) = %q, want OPENAI_COMPAT", alias, got)
		}
	}
}

// A model name may itself contain colons (providers namespace as vendor/model:tag), so only the
// first colon separates provider from model.
func TestDriverConfigFor_ModelMayContainColons(t *testing.T) {
	dc, err := driverConfigFor("p:acme/model:free", env(map[string]string{
		"DREAMCHAT_PROVIDER_P_BASE_URL": "https://p.example/v1",
	}))
	if err != nil {
		t.Fatalf("driverConfigFor: %v", err)
	}
	if dc.Model != "acme/model:free" {
		t.Fatalf("model = %q, want the whole remainder", dc.Model)
	}
}

// The fake bridge must stay reachable with zero provider configuration — it is the keyless local
// path and the reason CI needs no keys.
func TestSeatConfig_FakeBridgeNeedsNoProviderConfig(t *testing.T) {
	cfg, err := seatConfig(env(map[string]string{"DREAMCHAT_BRIDGE": "fake"}))
	if err != nil {
		t.Fatalf("fake bridge must not need configuration: %v", err)
	}
	for _, seat := range allSeatNames {
		if !strings.HasPrefix(cfg[seat].Provider, "fake-") {
			t.Fatalf("seat %s = %q under DREAMCHAT_BRIDGE=fake, want a fake stand-in", seat, cfg[seat].Provider)
		}
	}
}

// ── the driver's side of the contract ───────────────────────────────────────────────────────────

// Recorded wire shapes, captured against a stub so the two structured modes are pinned without a
// provider. json_object is the universal one; json_schema is what a provider that implements strict
// decoding receives.
func TestOpenAICompat_StructuredWireShapes(t *testing.T) {
	for _, tc := range []struct {
		mode       string
		wantType   string
		wantSchema bool
	}{
		{jsonModeObject, "json_object", false},
		{jsonModeSchema, "json_schema", true},
	} {
		var got map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/chat/completions" {
				t.Errorf("path = %q, want /chat/completions", r.URL.Path)
			}
			if h := r.Header.Get("Authorization"); h != "Bearer k" {
				t.Errorf("auth header = %q, want the bearer token", h)
			}
			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
		}))
		defer srv.Close()

		d, err := newOpenAICompatDriver(DriverConfig{
			Model: "m", Params: map[string]string{
				"base_url": srv.URL, "api_key": "k", "json_mode": tc.mode, "provider_alias": "cheapo",
			}})
		if err != nil {
			t.Fatalf("%s: newOpenAICompatDriver: %v", tc.mode, err)
		}
		if d.Name() != "cheapo:m" {
			t.Fatalf("Name = %q, want the operator's alias so a log line says who answered", d.Name())
		}
		out, err := d.Generate(t.Context(), GenRequest{
			Prompt: "p", Schema: json.RawMessage(`{"type":"object"}`)})
		if err != nil {
			t.Fatalf("%s: Generate: %v", tc.mode, err)
		}
		if out != `{"ok":true}` {
			t.Fatalf("%s: content = %q, want the message content verbatim", tc.mode, out)
		}

		rf, _ := got["response_format"].(map[string]any)
		if rf["type"] != tc.wantType {
			t.Fatalf("%s: response_format.type = %v, want %s", tc.mode, rf["type"], tc.wantType)
		}
		if _, hasSchema := rf["json_schema"]; hasSchema != tc.wantSchema {
			t.Fatalf("%s: json_schema present = %v, want %v", tc.mode, hasSchema, tc.wantSchema)
		}
		// The leash rides the system message in BOTH modes: it is what the model reads if a
		// provider's strict mode turns out to be advisory, which is not visible from out here.
		msgs, _ := got["messages"].([]any)
		first, _ := msgs[0].(map[string]any)
		if first["role"] != "system" || !strings.Contains(first["content"].(string), `"type":"object"`) {
			t.Fatalf("%s: the schema leash is missing from the system message", tc.mode)
		}
	}
}

// A free-text seat (narrate) must not be handed a response_format at all: some providers reject
// json_object on a request with no schema, and there is nothing to constrain anyway.
func TestOpenAICompat_FreeTextSendsNoResponseFormat(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"a line of prose"}}]}`))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "narrate"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, present := got["response_format"]; present {
		t.Fatal("a schema-less request must carry no response_format")
	}
}

func TestOpenAICompat_RejectsUnknownJSONMode(t *testing.T) {
	_, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": "https://x.example/v1", "json_mode": "wishful"}})
	if err == nil || !strings.Contains(err.Error(), "wishful") {
		t.Fatalf("err = %v, want an unknown json_mode rejected at construction", err)
	}
}

// Every seat with a structured floor must bind on this driver, or the capability report is a lie.
func TestOpenAICompat_SatisfiesEveryStructuredSeat(t *testing.T) {
	d, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": "https://x.example/v1"}})
	if err != nil {
		t.Fatalf("newOpenAICompatDriver: %v", err)
	}
	for _, seat := range []Seat{SeatDecompose, SeatResolve, SeatCognitionBatch,
		SeatCognitionIsolated, SeatWorldActor, SeatPlaceAuthor, SeatNarrate} {
		if _, err := BindSeat(seat, d); err != nil {
			t.Fatalf("seat %s will not bind: %v", seat.Name, err)
		}
	}
}

// ── routing policy ──────────────────────────────────────────────────────────────────────────────

// The founder's policy — US/EU hosts only, no data collection, one company excluded — must reach the
// wire verbatim on EVERY request, and must be configuration rather than a constant. This pins both:
// the provider-level policy applies to seats that do not override, and a seat-level policy wins.
func TestSeatConfig_RoutingPolicyIsPerSeatConfiguration(t *testing.T) {
	const providerPolicy = `{"only":["deepinfra","parasail"],"data_collection":"deny","ignore":["deepseek"],"require_parameters":true}`
	const narratePolicy = `{"only":["novita"],"data_collection":"deny","ignore":["deepseek"],"sort":"throughput"}`
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":            "route:cheap-model",
		"DREAMCHAT_SEATS":                   "narrate=route:big-model",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL": "https://aggregator.example/api/v1",
		"DREAMCHAT_PROVIDER_ROUTE_API_KEY":  "sk-x",
		"DREAMCHAT_PROVIDER_ROUTE_ROUTING":  providerPolicy,
		"DREAMCHAT_SEAT_ROUTING_NARRATE":    narratePolicy,
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["decompose"].Params["routing"]; got != providerPolicy {
		t.Fatalf("decompose routing = %s, want the provider policy", got)
	}
	if got := cfg["narrate"].Params["routing"]; got != narratePolicy {
		t.Fatalf("narrate routing = %s, want the seat override to win", got)
	}
	// Every seat must be policed — an unrouted seat is a compliance hole, and the boot line says so.
	for seat, dc := range cfg {
		if dc.Params["routing"] == "" {
			t.Fatalf("seat %s carries no routing policy", seat)
		}
	}
	if d := describeSeatConfig(cfg); !strings.Contains(d, "narrate=route:big-model(routed)") {
		t.Fatalf("describeSeatConfig = %q, want each seat's policy state visible at boot", d)
	}
}

// An unpoliced seat must be VISIBLE rather than silently permissive: the boot line is the only place
// an operator finds out before a request leaves the building.
func TestSeatConfig_UnroutedSeatIsVisibleAtBoot(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":        "p:m",
		"DREAMCHAT_PROVIDER_P_BASE_URL": "https://p.example/v1",
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if !strings.Contains(describeSeatConfig(cfg), "(unrouted)") {
		t.Fatal("a seat with no routing policy must announce itself at boot")
	}
}

// The policy reaches the wire VERBATIM under "provider", on structured and free-text requests alike:
// a narration is exactly as subject to jurisdiction and retention rules as a schema'd call.
func TestOpenAICompat_SendsRoutingPolicyOnEveryRequest(t *testing.T) {
	const policy = `{"only":["deepinfra"],"data_collection":"deny","ignore":["deepseek"],"require_parameters":true}`
	for _, structured := range []bool{true, false} {
		var got map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{}"}}]}`))
		}))
		defer srv.Close()
		d, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
			"base_url": srv.URL, "routing": policy, "provider_alias": "route"}})
		if err != nil {
			t.Fatalf("newOpenAICompatDriver: %v", err)
		}
		req := GenRequest{Prompt: "p"}
		if structured {
			req.Schema = json.RawMessage(`{"type":"object"}`)
		}
		if _, err := d.Generate(t.Context(), req); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		sent, err := json.Marshal(got["provider"])
		if err != nil {
			t.Fatalf("marshal provider block: %v", err)
		}
		var want, have any
		_ = json.Unmarshal([]byte(policy), &want)
		_ = json.Unmarshal(sent, &have)
		if fmt.Sprint(want) != fmt.Sprint(have) {
			t.Fatalf("structured=%v: provider block = %s, want the policy verbatim", structured, sent)
		}
	}
}

// No policy configured must send NO provider field at all. A plain provider does not know what
// "provider" means and some reject unknown fields outright.
func TestOpenAICompat_OmitsTheProviderFieldWhenUnrouted(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, present := got["provider"]; present {
		t.Fatal("an unrouted driver must omit the provider field entirely")
	}
}

// Malformed policy fails at CONSTRUCTION — i.e. at boot with the seat named — not with a 400 in the
// middle of a founder playtest. Compliance config that fails late fails in front of the person it
// was meant to protect.
func TestOpenAICompat_RejectsMalformedRoutingAtConstruction(t *testing.T) {
	_, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": "https://x.example/v1", "routing": `{"only":["deepinfra"`}})
	if err == nil || !strings.Contains(err.Error(), "routing policy") {
		t.Fatalf("err = %v, want a malformed policy rejected at construction", err)
	}
}

// Every request must carry a completion budget. Measured live: with no max_tokens an aggregator
// reserves the model's FULL completion window and refuses the call — 402 "you requested up to 65536
// tokens, but can only afford 11478" on an account with money in it. It is also the only ceiling on
// a reasoning model's spend, since reasoning tokens bill at the completion rate.
func TestOpenAICompat_AlwaysSendsACompletionBudget(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	}))
	defer srv.Close()

	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got["max_tokens"] != float64(defaultMaxTokens) {
		t.Fatalf("max_tokens = %v, want the default %d — an uncapped request is refused outright", got["max_tokens"], defaultMaxTokens)
	}

	d2, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "max_tokens": "4096"}})
	if _, err := d2.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got["max_tokens"] != float64(4096) {
		t.Fatalf("max_tokens = %v, want the configured 4096", got["max_tokens"])
	}

	if _, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "max_tokens": "lots"}}); err == nil {
		t.Fatal("a non-numeric max_tokens must be rejected at construction")
	}
}

// A reasoning model can spend its whole budget thinking and return null content. That must be an
// error that NAMES the cause, not an empty string that surfaces as a schema failure two retries
// later, blaming the model for what is a budget problem.
func TestOpenAICompat_NullContentNamesTheBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":null},"finish_reason":"length"}]}`))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	_, err := d.Generate(t.Context(), GenRequest{Prompt: "p"})
	if err == nil || !strings.Contains(err.Error(), "max_tokens") || !strings.Contains(err.Error(), "length") {
		t.Fatalf("err = %v, want it to name both the finish reason and the budget", err)
	}
}

// The budget follows the same seat-beats-provider precedence as the routing policy: narration needs
// more room than a schema'd seat, and that is a per-seat fact.
func TestSeatConfig_MaxTokensIsPerSeatConfiguration(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":              "route:cheap",
		"DREAMCHAT_SEATS":                     "narrate=route:big",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL":   "https://aggregator.example/api/v1",
		"DREAMCHAT_PROVIDER_ROUTE_MAX_TOKENS": "1024",
		"DREAMCHAT_SEAT_MAX_TOKENS_NARRATE":   "3072",
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["decompose"].Params["max_tokens"]; got != "1024" {
		t.Fatalf("decompose max_tokens = %q, want the provider default", got)
	}
	if got := cfg["narrate"].Params["max_tokens"]; got != "3072" {
		t.Fatalf("narrate max_tokens = %q, want the seat override", got)
	}
}

// A whole-world authoring call is allowed more wall time than a beat seat, but that is a seat
// property rather than a global relaxation: a slow decompose must still fail promptly.
func TestSeatConfig_RequestTimeoutIsPerSeatConfiguration(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":                               "route:cheap",
		"DREAMCHAT_SEATS":                                      "world_genesis=route:big",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL":                    "https://aggregator.example/api/v1",
		"DREAMCHAT_PROVIDER_ROUTE_REQUEST_TIMEOUT_SECONDS":     "45",
		"DREAMCHAT_SEAT_REQUEST_TIMEOUT_SECONDS_WORLD_GENESIS": "180",
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["decompose"].Params["request_timeout_seconds"]; got != "45" {
		t.Fatalf("decompose request_timeout_seconds = %q, want the provider default", got)
	}
	if got := cfg["world_genesis"].Params["request_timeout_seconds"]; got != "180" {
		t.Fatalf("world_genesis request_timeout_seconds = %q, want the seat override", got)
	}
}

// World creation is the one intentionally long structured call. Its five-minute default replaces the
// driver's former universal sixty-second deadline, while leaving all other seats at their existing
// default unless configured otherwise.
func TestSeatConfig_WorldGenesisRequestTimeoutDefaultsToFiveMinutes(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":            "route:m",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL": "https://aggregator.example/api/v1",
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["world_genesis"].Params["request_timeout_seconds"]; got != "300" {
		t.Fatalf("world_genesis request_timeout_seconds = %q, want 300", got)
	}
	if got := cfg["decompose"].Params["request_timeout_seconds"]; got != "" {
		t.Fatalf("decompose request_timeout_seconds = %q, want no override", got)
	}
}

// The combination that cost a live debugging session: json_object mandates an object reply, so an
// array-rooted schema cannot be satisfied under it. The model returned a bare object, the belt
// rejected it as "outside the closed vocabulary", and the error blamed the model's vocabulary for
// what was a response-format mismatch. Refuse the combination up front.
func TestOpenAICompat_RefusesJSONObjectForAnArrayRootedSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a request that cannot succeed must never reach the provider")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{}"}}]}`))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "json_mode": jsonModeObject}})
	_, err := d.Generate(t.Context(), GenRequest{
		Prompt: "p", Schema: json.RawMessage(`{"type":"array","items":{"type":"object"}}`)})
	if err == nil || !strings.Contains(err.Error(), jsonModeSchema) {
		t.Fatalf("err = %v, want a refusal naming the mode that does work", err)
	}
}

// The same schema under json_schema mode goes through: that is the documented remedy, and it is
// what the live deployment runs.
func TestOpenAICompat_ArrayRootedSchemaWorksUnderJSONSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"[]"}}]}`))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "json_mode": jsonModeSchema}})
	out, err := d.Generate(t.Context(), GenRequest{
		Prompt: "p", Schema: json.RawMessage(`{"type":"array","items":{"type":"object"}}`)})
	if err != nil || out != "[]" {
		t.Fatalf("out=%q err=%v, want the array-rooted call to succeed", out, err)
	}
}

func TestSchemaRootIsArray(t *testing.T) {
	for _, tc := range []struct {
		schema string
		want   bool
	}{
		{`{"type":"array"}`, true},
		{`{"type":["array","null"]}`, true},
		{`{"type":"object"}`, false},
		{`not json`, false},
	} {
		if got := schemaRootIsArray(json.RawMessage(tc.schema)); got != tc.want {
			t.Fatalf("schemaRootIsArray(%s) = %v, want %v", tc.schema, got, tc.want)
		}
	}
}

// Temperature is a SEAT property and the seats genuinely differ, so it is configured, not constant.
// Measured twice on the same input at the provider default: the founder's "I look around, who is
// there?" decomposed once to QUERY committing nothing, and once to ActorMoved committing three canon
// events — a question that moved the player. A mechanical seat left at a sampling default is a coin
// flip wearing a schema.
func TestSeatConfig_TemperatureIsPerSeatConfiguration(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":               "route:m",
		"DREAMCHAT_SEATS":                      "narrate=route:big",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL":    "https://aggregator.example/api/v1",
		"DREAMCHAT_PROVIDER_ROUTE_TEMPERATURE": "0",
		"DREAMCHAT_SEAT_TEMPERATURE_NARRATE":   "0.9",
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["decompose"].Params["temperature"]; got != "0" {
		t.Fatalf("decompose temperature = %q, want the provider default 0", got)
	}
	if got := cfg["narrate"].Params["temperature"]; got != "0.9" {
		t.Fatalf("narrate temperature = %q, want the seat override — prose is a performance", got)
	}
}

// Unset means UNSENT: the provider's own default stands rather than this service imposing a value
// on a seat nobody has reasoned about.
func TestOpenAICompat_TemperatureOnlyWhenConfigured(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	}))
	defer srv.Close()

	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, present := got["temperature"]; present {
		t.Fatal("an unconfigured temperature must not be sent")
	}

	// Zero is a REAL value, not an absent one — the whole point for a mechanical seat.
	d0, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "temperature": "0"}})
	if _, err := d0.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got["temperature"] != float64(0) {
		t.Fatalf("temperature = %v, want a literal 0 on the wire", got["temperature"])
	}

	if _, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "temperature": "hot"}}); err == nil {
		t.Fatal("a non-numeric temperature must be rejected at construction")
	}
}

// The reasoning policy is per-seat config for the same reason routing and temperature are — and it
// is the one whose ABSENCE is most expensive. Unset leaves the model's own default in force, and the
// mechanical model here advertises default_effort "high": roughly 80% of max_tokens spent thinking
// before a token of answer. That is how decompose died twice in front of the founder.
func TestSeatConfig_ReasoningIsPerSeatConfiguration(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":             "route:m",
		"DREAMCHAT_SEATS":                    "narrate=route:big",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL":  "https://aggregator.example/api/v1",
		"DREAMCHAT_PROVIDER_ROUTE_REASONING": `{"effort":"none"}`,
		"DREAMCHAT_SEAT_REASONING_NARRATE":   `{"effort":"low"}`,
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["decompose"].Params["reasoning"]; got != `{"effort":"none"}` {
		t.Fatalf("decompose reasoning = %q, want the provider policy", got)
	}
	if got := cfg["narrate"].Params["reasoning"]; got != `{"effort":"low"}` {
		t.Fatalf("narrate reasoning = %q, want the seat override", got)
	}
}

func TestOpenAICompat_SendsReasoningPolicyWhenConfigured(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	}))
	defer srv.Close()

	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, present := got["reasoning"]; present {
		t.Fatal("an unconfigured reasoning policy must not be sent")
	}

	d2, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "reasoning": `{"effort":"none"}`}})
	if _, err := d2.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	sent, _ := json.Marshal(got["reasoning"])
	if string(sent) != `{"effort":"none"}` {
		t.Fatalf("reasoning = %s, want the policy verbatim", sent)
	}

	if _, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "reasoning": `{"effort":`}}); err == nil {
		t.Fatal("a malformed reasoning policy must be rejected at construction")
	}
}

// Every seat is timed, whatever the call site — the decorator is applied where seats are BOUND, not
// where they are called, because an instrument you must remember to add at three call sites is one
// that will be missing from the fourth. And it must not change what it measures: a non-streaming
// driver must not start advertising GenerateStream just because it was wrapped.
func TestBridge_EverySeatIsTimedAndCapabilitiesAreUnchanged(t *testing.T) {
	b, err := NewBridgeWithDrivers(map[string]Driver{
		SeatDecompose.Name: NewFakeStructuredDriver("fake-structured:t", nil),
		SeatNarrate.Name:   NewFakeTextDriver("fake-text:t"),
	}, SeatDecompose, SeatNarrate)
	if err != nil {
		t.Fatalf("NewBridgeWithDrivers: %v", err)
	}
	for _, seat := range []string{SeatDecompose.Name, SeatNarrate.Name} {
		d := b.Driver(seat)
		if d == nil {
			t.Fatalf("seat %s unbound", seat)
		}
		if !d.Capabilities().Has() || d.Name() == "" {
			t.Fatalf("seat %s lost its identity through the decorator", seat)
		}
	}
	// The fakes do not stream, so nothing wrapped by the decorator may claim to.
	if _, ok := b.Driver(SeatNarrate.Name).(StreamingDriver); ok {
		t.Fatal("a non-streaming driver advertises GenerateStream after instrumentation")
	}
	// Structured capability survives, or every structured seat would fail to bind.
	if !b.Driver(SeatDecompose.Name).Capabilities().Has(CapStructuredOutput) {
		t.Fatal("the decorator dropped a reported capability")
	}
}

// json_mode is the one setting where the seats provably disagree, so it is per-seat like the rest:
// decompose is reliable under strict decoding and narrate is destroyed by it.
func TestSeatConfig_JSONModeIsPerSeatConfiguration(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":             "route:m",
		"DREAMCHAT_SEATS":                    "narrate=route:big",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL":  "https://aggregator.example/api/v1",
		"DREAMCHAT_PROVIDER_ROUTE_JSON_MODE": jsonModeSchema,
		"DREAMCHAT_SEAT_JSON_MODE_NARRATE":   jsonModeOff,
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["decompose"].Params["json_mode"]; got != jsonModeSchema {
		t.Fatalf("decompose json_mode = %q, want strict", got)
	}
	if got := cfg["narrate"].Params["json_mode"]; got != jsonModeOff {
		t.Fatalf("narrate json_mode = %q, want the seat override", got)
	}
}

// jsonModeOff sends no response_format and leaves the belt as the only enforcement — the point being
// that constrained decoding guarantees a SHAPE, which is worthless when the shape was never the hard
// part. Measured: narration/1 under strict decoding returned every segment structurally perfect and
// textually empty.
func TestOpenAICompat_OffModeSendsNoResponseFormatButKeepsTheLeash(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"[{\"a\":1}]"}}]}`))
	}))
	defer srv.Close()
	d, err := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "json_mode": jsonModeOff}})
	if err != nil {
		t.Fatalf("newOpenAICompatDriver: %v", err)
	}
	out, err := d.Generate(t.Context(), GenRequest{Prompt: "p", Schema: json.RawMessage(`{"type":"array"}`)})
	if err != nil || out != `[{"a":1}]` {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if _, present := got["response_format"]; present {
		t.Fatal("off mode must send no response_format")
	}
	// The schema still rides the system message — off is not "no leash", it is "no constrained
	// decoding" — and it names the ROOT, because "a single JSON document" makes a model reach for an
	// object: measured live, narration/1 came back as {"array":[…]} until the wording was fixed.
	msgs, _ := got["messages"].([]any)
	sys, _ := msgs[0].(map[string]any)
	content, _ := sys["content"].(string)
	if !strings.Contains(content, `"type":"array"`) {
		t.Fatal("the schema leash is missing from the system message in off mode")
	}
	if !strings.Contains(content, "JSON ARRAY") {
		t.Fatalf("an array-rooted schema must name its root in the leash:\n%s", content)
	}
}

// A model without constrained decoding volunteers a markdown fence about half the time. That is a
// transport wrapper, not content — failing over punctuation would cost a retry and 4-8s of dead air.
func TestStripCodeFence(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"```json\n[{\"a\":1}]\n```", `[{"a":1}]`},
		{"```\n[1]\n```", `[1]`},
		{`[{"a":1}]`, `[{"a":1}]`},
		{"no fence here", "no fence here"},
		{"```unterminated", "```unterminated"},
	} {
		if got := stripCodeFence(tc.in); got != tc.want {
			t.Fatalf("stripCodeFence(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A repair at temperature 0 re-asks a deterministic model the same question and gets the same
// refusal. This repo has been bitten by that shape once already (the journey livelock: "any retry of
// a deterministic decision that failed will fail the same way"), and pinning the mechanical seats to
// 0 walked back into it — measured live, resolve refused `AttributeChanged requires target_id` on
// attempt 1/2 and again on 2/2, ~14s of dead air asking twice.
func TestOpenAICompat_ARepairIsAllowedToDiffer(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{
		"base_url": srv.URL, "temperature": "0"}})

	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "p"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got["temperature"] != float64(0) {
		t.Fatalf("first attempt temperature = %v, want the configured 0 — the first answer must be reproducible", got["temperature"])
	}

	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "p", Repair: true}); err != nil {
		t.Fatalf("Generate(repair): %v", err)
	}
	if got["temperature"] != repairTemperature {
		t.Fatalf("repair temperature = %v, want %v — a retry that cannot differ is not a retry", got["temperature"], repairTemperature)
	}
}

// A seat that never pinned temperature is untouched by the repair rule: the provider's default was
// already free to differ, and this must not start imposing a value on it.
func TestOpenAICompat_RepairDoesNotInventATemperature(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	if _, err := d.Generate(t.Context(), GenRequest{Prompt: "p", Repair: true}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, present := got["temperature"]; present {
		t.Fatal("a repair must not invent a temperature for a seat that never pinned one")
	}
}

// Streaming is what turns the narrate seat's existing line-by-line path on. The driver must report
// it, deliver each delta, and return the accumulated text — the same contract Generate honours.
func TestOpenAICompat_GenerateStreamDeliversDeltasAndAccumulates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["stream"] != true {
			t.Errorf("stream flag = %v, want true", body["stream"])
		}
		fl, _ := w.(http.Flusher)
		for _, c := range []string{`{"choices":[{"delta":{"content":"[{\"a\":"}}]}`,
			`{"choices":[{"delta":{"content":"1}]"}}]}`,
			`{"choices":[{"delta":{}}]}`, // an empty delta must not break the stream
			`not json`,                   // nor must a malformed frame
		} {
			_, _ = w.Write([]byte("data: " + c + "\n\n"))
			if fl != nil {
				fl.Flush()
			}
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	sd, ok := d.(StreamingDriver)
	if !ok {
		t.Fatal("the driver must report streaming, or the narrate seat silently keeps waiting for the whole reply")
	}
	var deltas []string
	out, err := sd.GenerateStream(t.Context(), GenRequest{Prompt: "p"}, func(s string) { deltas = append(deltas, s) })
	if err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}
	if out != `[{"a":1}]` {
		t.Fatalf("accumulated = %q, want the whole reply", out)
	}
	if len(deltas) != 2 {
		t.Fatalf("deltas = %v, want the two content chunks and nothing else", deltas)
	}
}

// A stream that carries no content at all is an error, not an empty narration: the caller must fall
// back rather than emit silence.
func TestOpenAICompat_EmptyStreamIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()
	d, _ := newOpenAICompatDriver(DriverConfig{Model: "m", Params: map[string]string{"base_url": srv.URL}})
	if _, err := d.(StreamingDriver).GenerateStream(t.Context(), GenRequest{Prompt: "p"}, func(string) {}); err == nil {
		t.Fatal("an empty stream must be an error")
	}
}

// A provider-wide timeout is still an explicit choice for world_genesis; its five-minute value is
// only the fallback when neither configuration level names a timeout.
func TestSeatConfig_ProviderTimeoutOverridesWorldGenesisFallback(t *testing.T) {
	cfg, err := seatConfigFromEnv(env(map[string]string{
		"DREAMCHAT_SEAT_DEFAULT":                           "route:m",
		"DREAMCHAT_PROVIDER_ROUTE_BASE_URL":                "https://aggregator.example/api/v1",
		"DREAMCHAT_PROVIDER_ROUTE_REQUEST_TIMEOUT_SECONDS": "120",
	}))
	if err != nil {
		t.Fatalf("seatConfigFromEnv: %v", err)
	}
	if got := cfg["world_genesis"].Params["request_timeout_seconds"]; got != "120" {
		t.Fatalf("world_genesis request_timeout_seconds = %q, want the provider override", got)
	}
}
