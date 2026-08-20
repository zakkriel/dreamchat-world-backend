package main

// Per-seat provider routing from the environment (D-13, ADR-P018; the provider-neutrality mandate).
//
// ── WHY THIS FILE EXISTS AND WHAT IT REPLACES ───────────────────────────────────────────────────
// Routing used to be a hardcoded all-Anthropic map plus two hand-written override blocks — one for
// `resolve`, one for both cognition seats — each with its own DREAMCHAT_<SEAT>_{PROVIDER,MODEL,
// BASE_URL,API_KEY} family. Four of the seven seats had no override at all, so the honest summary
// was "Anthropic unless you are one of three specific seats". The founder's standing ruling, on the
// record since 2026-08-07, is the opposite: *"never default seats to Anthropic; per-seat overrides
// owed for ALL seats"*. Those two blocks are gone. This is one scheme for every seat.
//
// ── THE SCHEME, DELIBERATELY BORING ─────────────────────────────────────────────────────────────
//
//	DREAMCHAT_SEAT_DEFAULT   provider:model   — the default for every seat
//	DREAMCHAT_SEATS          seat=provider:model,seat=provider:model   — per-seat overrides
//
//	DREAMCHAT_PROVIDER_<NAME>_BASE_URL   where that provider lives
//	DREAMCHAT_PROVIDER_<NAME>_API_KEY    its bearer token
//	DREAMCHAT_PROVIDER_<NAME>_DIALECT    wire dialect; default "openai-compat"
//	DREAMCHAT_PROVIDER_<NAME>_JSON_MODE  "json_object" (default) or "json_schema"
//
// <NAME> is whatever the operator called the provider, uppercased. It is an ALIAS, not a known
// vendor: `deepseek`, `kimi`, `mistral`, `the-cheap-one` are all equally valid and none of them
// appears anywhere in this codebase. That is the entire point of the mandate — the code knows wire
// DIALECTS, which are a technical fact, and the environment knows vendors, which are a commercial
// choice. Adding a provider is two environment variables and no diff.
//
// ── FAILS CLOSED, ON PURPOSE ────────────────────────────────────────────────────────────────────
// With no configuration and no DREAMCHAT_BRIDGE=fake, this returns an error and the server does not
// boot. It would have been easy to keep a default — and a default is exactly what the mandate
// forbids, because a default is what quietly bills someone. A missing config is an operator
// mistake, and the honest time to say so is at boot with the variable named, not at the first beat
// with a 401 from a provider nobody meant to call.

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	seatDefaultEnv = "DREAMCHAT_SEAT_DEFAULT"
	seatsEnv       = "DREAMCHAT_SEATS"
	providerEnvFmt = "DREAMCHAT_PROVIDER_%s_%s"
	// Per-seat routing override. Wins over the provider-level policy when both are set.
	seatRoutingEnvFmt = "DREAMCHAT_SEAT_ROUTING_%s"
	// Per-seat completion budget. Same precedence rule as routing.
	seatMaxTokensEnvFmt = "DREAMCHAT_SEAT_MAX_TOKENS_%s"
	// Per-seat sampling temperature. Same precedence rule again.
	seatTemperatureEnvFmt = "DREAMCHAT_SEAT_TEMPERATURE_%s"
	// Per-seat reasoning policy. Same precedence rule again.
	seatReasoningEnvFmt = "DREAMCHAT_SEAT_REASONING_%s"
	// Per-seat structured-output mode. Same precedence rule again — and the one setting where the
	// seats provably disagree: see the driver's jsonModeOff.
	seatJSONModeEnvFmt = "DREAMCHAT_SEAT_JSON_MODE_%s"

	// dialectOpenAICompat is the lingua franca: chat/completions as popularised by OpenAI and spoken
	// by most hosted providers. It is the default because assuming it is right far more often than
	// not, and being wrong costs one environment variable.
	dialectOpenAICompat = "openai-compat"
	dialectAnthropic    = "anthropic"
)

// allSeatNames is every seat this service binds. Kept here so a new seat cannot be silently left
// unroutable: seatConfigFromEnv builds an entry for each one and nothing else.
var allSeatNames = []string{
	SeatDecompose.Name, SeatNarrate.Name, SeatResolve.Name,
	SeatCognitionBatch.Name, SeatCognitionIsolated.Name,
	SeatWorldActor.Name, SeatPlaceAuthor.Name,
	SeatWorldGenesis.Name, SeatWorldInterview.Name, SeatWorldKickstart.Name,
}

// seatConfigFromEnv resolves the seat→driver map from the environment. Pure apart from the reads,
// which is why lookup is injected: the whole scheme is unit-testable without setting a process
// environment, and the tests below are the recorded shapes this contract is checked against.
func seatConfigFromEnv(lookup func(string) string) (SeatConfig, error) {
	def := strings.TrimSpace(lookup(seatDefaultEnv))
	overrides, err := parseSeatOverrides(lookup(seatsEnv))
	if err != nil {
		return nil, err
	}
	if def == "" {
		// A default is optional ONLY if every seat is named explicitly; otherwise some seat would
		// have nowhere to go, and guessing on its behalf is the thing this scheme exists to stop.
		for _, name := range allSeatNames {
			if _, ok := overrides[name]; !ok {
				return nil, fmt.Errorf(
					"no provider for seat %q: set %s=provider:model, or name every seat in %s "+
						"(set DREAMCHAT_BRIDGE=fake for keyless local dev)",
					name, seatDefaultEnv, seatsEnv)
			}
		}
	}

	cfg := SeatConfig{}
	for _, name := range allSeatNames {
		spec, ok := overrides[name]
		if !ok {
			spec = def
		}
		dc, err := driverConfigForSeat(name, spec, lookup)
		if err != nil {
			return nil, fmt.Errorf("seat %q: %w", name, err)
		}
		cfg[name] = dc
	}
	return cfg, nil
}

// parseSeatOverrides reads "seat=provider:model,seat=provider:model". An unknown seat name is an
// ERROR rather than a shrug: a typo like "narration=…" would otherwise leave the narrate seat on the
// default and look exactly like the override working.
func parseSeatOverrides(raw string) (map[string]string, error) {
	out := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		seat, spec, ok := strings.Cut(part, "=")
		seat, spec = strings.TrimSpace(seat), strings.TrimSpace(spec)
		if !ok || seat == "" || spec == "" {
			return nil, fmt.Errorf("%s: %q is not seat=provider:model", seatsEnv, part)
		}
		if !knownSeat(seat) {
			return nil, fmt.Errorf("%s: unknown seat %q (known: %s)", seatsEnv, seat,
				strings.Join(allSeatNames, ", "))
		}
		out[seat] = spec
	}
	return out, nil
}

func knownSeat(name string) bool {
	for _, s := range allSeatNames {
		if s == name {
			return true
		}
	}
	return false
}

// driverConfigFor turns "provider:model" plus that provider's environment into a DriverConfig.
// A model may itself contain colons (some providers namespace as "vendor/model:tag"), so only the
// FIRST colon separates.
// routingFor resolves the request-routing policy for a seat: an aggregator such as OpenRouter takes
// a preferences object that decides WHICH underlying host serves the request — data-retention
// policy, jurisdiction, which companies are excluded, whether an endpoint must support the
// parameters we send.
//
// It is CONFIGURATION and never a constant, for the same reason provider names are. The founder's
// current policy is US/EU hosts only, data_collection deny, one company excluded — but that is a
// commercial and legal judgement that will change without this code changing, and a policy compiled
// into a binary is a policy nobody can correct without a deploy. The seat-level knob wins over the
// provider-level one, because a narrative seat may want to trade price for throughput while the
// mechanical seats do not.
//
// The value is passed through verbatim as JSON rather than modelled as a struct: the aggregator owns
// that schema, it gains fields regularly, and a struct here would silently drop any field this repo
// had not heard of — turning "policy not yet supported" into "policy silently not applied", which is
// the worst possible failure for a field whose whole job is compliance.
func routingFor(seat, provider string, lookup func(string) string) string {
	return perSeatOrProvider(seat, provider, "ROUTING", seatRoutingEnvFmt, lookup)
}

// requestTimeoutFor keeps the ordinary short request deadline for every seat except world_genesis,
// whose one whole-world document legitimately takes longer to author. Operators can override either
// level; a positive timeout remains mandatory so one provider request cannot occupy a worker forever.
func requestTimeoutFor(seat, provider string, lookup func(string) string) string {
	timeout := perSeatOrProvider(seat, provider, "REQUEST_TIMEOUT_SECONDS", "DREAMCHAT_SEAT_REQUEST_TIMEOUT_SECONDS_%s", lookup)
	if timeout == "" && seat == SeatWorldGenesis.Name {
		return "300"
	}
	return timeout
}

// perSeatOrProvider reads a setting that has both a per-seat and a per-provider spelling, seat
// first. Two settings use it and both want the same precedence, so the rule lives in one place
// rather than being re-derived and eventually diverging.
func perSeatOrProvider(seat, provider, suffix, seatFmt string, lookup func(string) string) string {
	if seat != "" {
		if v := strings.TrimSpace(lookup(fmt.Sprintf(seatFmt, envKey(seat)))); v != "" {
			return v
		}
	}
	return strings.TrimSpace(lookup(fmt.Sprintf(providerEnvFmt, envKey(provider), suffix)))
}

func driverConfigFor(spec string, lookup func(string) string) (DriverConfig, error) {
	return driverConfigForSeat("", spec, lookup)
}

func driverConfigForSeat(seat, spec string, lookup func(string) string) (DriverConfig, error) {
	provider, model, ok := strings.Cut(strings.TrimSpace(spec), ":")
	provider, model = strings.TrimSpace(provider), strings.TrimSpace(model)
	if !ok || provider == "" || model == "" {
		return DriverConfig{}, fmt.Errorf("%q is not provider:model", spec)
	}

	// A fake provider is a complete driver on its own — no endpoint, no key, nothing to look up.
	if strings.HasPrefix(provider, "fake-") {
		return DriverConfig{Provider: provider, Model: model}, nil
	}

	env := func(suffix string) string {
		return strings.TrimSpace(lookup(fmt.Sprintf(providerEnvFmt, envKey(provider), suffix)))
	}
	routing := routingFor(seat, provider, lookup)
	maxTokens := perSeatOrProvider(seat, provider, "MAX_TOKENS", seatMaxTokensEnvFmt, lookup)
	temperature := perSeatOrProvider(seat, provider, "TEMPERATURE", seatTemperatureEnvFmt, lookup)
	reasoning := perSeatOrProvider(seat, provider, "REASONING", seatReasoningEnvFmt, lookup)
	requestTimeout := requestTimeoutFor(seat, provider, lookup)
	dialect := env("DIALECT")
	if dialect == "" {
		dialect = dialectOpenAICompat
	}
	switch dialect {
	case dialectOpenAICompat:
		baseURL := env("BASE_URL")
		if baseURL == "" {
			return DriverConfig{}, fmt.Errorf(
				"provider %q speaks %s but has no endpoint: set %s",
				provider, dialect, fmt.Sprintf(providerEnvFmt, envKey(provider), "BASE_URL"))
		}
		params := map[string]string{
			"base_url": baseURL,
			"api_key":  env("API_KEY"),
			// Empty is fine — the driver defaults it, and every provider supports the default.
			"json_mode": perSeatOrProvider(seat, provider, "JSON_MODE", seatJSONModeEnvFmt, lookup),
			// Carried so a driver's Name(), and therefore every log line and trace frame, says which
			// provider answered. Without it four seats on four providers are indistinguishable.
			"provider_alias": provider,
			// Routing policy, verbatim JSON, merged into the request body as an extra field. See
			// routingFor for why it is configuration and not a constant.
			"routing": routing,
			// Completion budget. Empty means the driver's default; see the driver for why sending
			// one at all is not optional against an aggregator.
			"max_tokens": maxTokens,
			// Sampling temperature. Empty means "do not send one" — the provider's default stands.
			// Set it to 0 on the seats whose output is a classification rather than a performance.
			"temperature": temperature,
			// Reasoning policy, verbatim JSON. Empty means "do not send one", which leaves the
			// MODEL'S OWN default in force — and that default is not neutral: see the driver.
			"reasoning":               reasoning,
			"request_timeout_seconds": requestTimeout,
		}
		return DriverConfig{Provider: dialect, Model: model, Params: params}, nil
	case dialectAnthropic:
		// Reachable, and deliberately not special: Anthropic is one provider among several now, and
		// the only thing that makes it different is that it is no longer the default.
		return DriverConfig{Provider: dialect, Model: model, Params: map[string]string{
			"api_key":                 env("API_KEY"),
			"provider_alias":          provider,
			"request_timeout_seconds": requestTimeout,
		}}, nil
	default:
		return DriverConfig{}, fmt.Errorf("provider %q: unknown dialect %q (known: %s, %s)",
			provider, dialect, dialectOpenAICompat, dialectAnthropic)
	}
}

// envKey normalises a provider alias into the environment-variable fragment: upper case, and any
// character that cannot appear in a shell variable name becomes an underscore. So `openai-compat`,
// `openai_compat` and `OpenAI-Compat` all read DREAMCHAT_PROVIDER_OPENAI_COMPAT_*, and an operator
// who writes the alias one way in DREAMCHAT_SEATS and another in the variable name still lands on
// their feet.
func envKey(provider string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(provider) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// describeSeatConfig is the boot log line: which seat is answered by which provider and model, with
// no keys. Sorted so two boots of the same config produce the same line and a diff means something.
func describeSeatConfig(cfg SeatConfig) string {
	parts := make([]string, 0, len(cfg))
	for seat, dc := range cfg {
		alias := dc.Provider
		if a := dc.Params["provider_alias"]; a != "" {
			alias = a
		}
		policy := "unrouted"
		if dc.Params["routing"] != "" {
			policy = "routed"
		}
		parts = append(parts, fmt.Sprintf("%s=%s:%s(%s)", seat, alias, dc.Model, policy))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

// osLookup is the process environment, as an injectable function.
func osLookup(key string) string { return os.Getenv(key) }
