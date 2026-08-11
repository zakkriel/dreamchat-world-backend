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
		dc, err := driverConfigFor(spec, lookup)
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
func driverConfigFor(spec string, lookup func(string) string) (DriverConfig, error) {
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
			"json_mode": env("JSON_MODE"),
			// Carried so a driver's Name(), and therefore every log line and trace frame, says which
			// provider answered. Without it four seats on four providers are indistinguishable.
			"provider_alias": provider,
		}
		return DriverConfig{Provider: dialect, Model: model, Params: params}, nil
	case dialectAnthropic:
		// Reachable, and deliberately not special: Anthropic is one provider among several now, and
		// the only thing that makes it different is that it is no longer the default.
		return DriverConfig{Provider: dialect, Model: model, Params: map[string]string{
			"api_key":        env("API_KEY"),
			"provider_alias": provider,
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
		parts = append(parts, fmt.Sprintf("%s=%s:%s", seat, alias, dc.Model))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

// osLookup is the process environment, as an injectable function.
func osLookup(key string) string { return os.Getenv(key) }
