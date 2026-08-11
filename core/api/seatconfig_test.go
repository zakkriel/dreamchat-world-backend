package main

import (
	"encoding/json"
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
