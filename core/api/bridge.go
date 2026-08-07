package main

// Chunk-5 LLM bridge — model-agnostic, per-seat routing (D-13; ADR-P018). Each LLM SEAT resolves its
// driver from config (seat → {provider, model, params}); a driver binds to a seat only if the
// driver's REPORTED capabilities satisfy the seat's floor. No seat hardcodes a model; no provider SDK
// lives in the canon engine (core/db) — drivers live only here. Quarantine holds per seat regardless
// of the bound model: decompose PROPOSES only (D-1/SPEC-015); narrate is perception-bound (ADR-020).

import (
	"context"
	"encoding/json"
	"fmt"
)

// --- Capabilities -----------------------------------------------------------
// Capability is validated against a driver's REPORTED set, never a config label (Image-Platform
// latent-risk lesson: a label can claim anything; the report is what the driver can actually do).
type Capability string

const (
	// CapStructuredOutput: the driver constrains generation to a JSON schema (constrained decoding /
	// strict tool-use) — output is schema-valid BY CONSTRUCTION. This is the decompose leash
	// (ADR-009/D-1, SPEC-015): the closed vocabulary is enforced at token generation, NOT post-hoc.
	CapStructuredOutput Capability = "structured_output"
)

type CapabilitySet map[Capability]bool

func (c CapabilitySet) Has(reqs ...Capability) bool {
	for _, r := range reqs {
		if !c[r] {
			return false
		}
	}
	return true
}

func keysOf(c CapabilitySet) []Capability {
	out := []Capability{}
	for k, v := range c {
		if v {
			out = append(out, k)
		}
	}
	return out
}

// --- Driver -----------------------------------------------------------------
// GenRequest carries the perception-bound payload (B-1/I-3) + prompt. A non-nil Schema marks a
// STRUCTURED request: the driver must constrain output to it (requires CapStructuredOutput).
type GenRequest struct {
	Payload PerceptionPayload
	Prompt  string
	Schema  json.RawMessage // non-nil ⇒ constrained/structured generation (the leash)
}

// Driver is one bound model. Drivers live only in this bridge layer (D-13) — never in core/db.
type Driver interface {
	Name() string
	Capabilities() CapabilitySet
	Generate(ctx context.Context, req GenRequest) (string, error)
}

// StreamingDriver is an OPTIONAL capability a Driver may also implement (rung3 Task 4, plan §"real
// line-by-line, where the driver can"): GenerateStream calls onDelta with each raw text chunk as it
// arrives, and still returns the full accumulated text on success — the identical contract Generate
// honors, plus the incremental callback. It is a SEPARATE interface, never a Driver method, because
// granularity is a driver CAPABILITY, not a contract term (the founder chose line-at-a-time delivery
// precisely because the belts need a whole line to judge — narration/1's array-of-objects shape is
// unchanged either way). A driver that cannot stream simply does not implement this; the narrate step
// (beatsstream.go) type-asserts for it and falls back to the ordinary Generate call, producing the
// IDENTICAL frame sequence either way — same protocol, same frontend, better feel where the stack
// supports it.
type StreamingDriver interface {
	GenerateStream(ctx context.Context, req GenRequest, onDelta func(string)) (string, error)
}

// --- Seats ------------------------------------------------------------------
type Seat struct {
	Name     string
	Requires []Capability
}

var (
	// decompose: prose→events; REQUIRES structured output (the leash). Propose-only (D-1/SPEC-015).
	SeatDecompose = Seat{Name: "decompose", Requires: []Capability{CapStructuredOutput}}
	// narrate: perception-bound free text (ADR-020); cheap high-volume; no schema, no requirement.
	SeatNarrate = Seat{Name: "narrate", Requires: nil}
	// resolve: attempt outcome ruling (SPEC-013); REQUIRES structured output for schema-valid ruling.
	SeatResolve = Seat{Name: "resolve", Requires: []Capability{CapStructuredOutput}}
	// cognition_batch: NPC decision batch (SPEC-?); REQUIRES structured output.
	SeatCognitionBatch = Seat{Name: "cognition_batch", Requires: []Capability{CapStructuredOutput}}
	// cognition_isolated: isolated NPC decision (SPEC-?); REQUIRES structured output.
	SeatCognitionIsolated = Seat{Name: "cognition_isolated", Requires: []Capability{CapStructuredOutput}}
	// world_actor: world transformation (SPEC-?); REQUIRES structured output.
	SeatWorldActor = Seat{Name: "world_actor", Requires: []Capability{CapStructuredOutput}}
	// place_author: authors a NEW place's identity (descriptor/kind/extent_class) when the Journey's
	// world's turn needs a stage nothing known contains (design §4.6, R2). REQUIRES structured output —
	// the schema leash is what keeps geometry (a coordinate, a radius, any number) out of the model's
	// hands; the engine alone draws the footprint (fn_extent_class_metres + fn_area_around).
	SeatPlaceAuthor = Seat{Name: "place_author", Requires: []Capability{CapStructuredOutput}}
)

// BindSeat validates the driver's REPORTED capabilities satisfy the seat floor; fail CLOSED.
func BindSeat(s Seat, d Driver) (Driver, error) {
	if d == nil {
		return nil, fmt.Errorf("seat %q: nil driver", s.Name)
	}
	if !d.Capabilities().Has(s.Requires...) {
		return nil, fmt.Errorf("seat %q requires %v but driver %q reports %v (%s missing; fail closed)",
			s.Name, s.Requires, d.Name(), keysOf(d.Capabilities()), CapStructuredOutput)
	}
	return d, nil
}

// --- Routing ----------------------------------------------------------------
// seat name → {provider, model, params}. Re-pointing one entry changes ONLY that seat.
type DriverConfig struct {
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Params   map[string]string `json:"params,omitempty"`
}
type SeatConfig map[string]DriverConfig

// DriverFactory builds a Driver from config (the provider registry). Keeps provider SDKs out of the
// canon engine and lets a seat re-point to a different model with NO code change.
type DriverFactory func(DriverConfig) (Driver, error)

type Bridge struct {
	seats map[string]Driver
}

// NewBridge resolves each seat's driver from config via the factory and binds it (capability floor).
func NewBridge(cfg SeatConfig, factory DriverFactory, seats ...Seat) (*Bridge, error) {
	b := &Bridge{seats: map[string]Driver{}}
	for _, s := range seats {
		dc, ok := cfg[s.Name]
		if !ok {
			return nil, fmt.Errorf("no driver configured for seat %q", s.Name)
		}
		d, err := factory(dc)
		if err != nil {
			return nil, fmt.Errorf("seat %q: %w", s.Name, err)
		}
		bound, err := BindSeat(s, d)
		if err != nil {
			return nil, err
		}
		b.seats[s.Name] = bound
	}
	return b, nil
}

// NewBridgeWithDrivers binds already-built drivers (tests + the operator gate's live wiring).
func NewBridgeWithDrivers(bound map[string]Driver, seats ...Seat) (*Bridge, error) {
	b := &Bridge{seats: map[string]Driver{}}
	for _, s := range seats {
		d, ok := bound[s.Name]
		if !ok {
			return nil, fmt.Errorf("no driver provided for seat %q", s.Name)
		}
		bd, err := BindSeat(s, d)
		if err != nil {
			return nil, err
		}
		b.seats[s.Name] = bd
	}
	return b, nil
}

func (b *Bridge) Driver(seat string) Driver { return b.seats[seat] }

// DefaultDriverFactory is the provider registry. "anthropic" is the production default (Claude via the
// Anthropic API; structured output / strict tool-use as the leash). "fake-*" are CI deterministic
// stand-ins. Adding a provider is one case here — no seat changes (D-13).
func DefaultDriverFactory(dc DriverConfig) (Driver, error) {
	switch dc.Provider {
	case "anthropic":
		return newAnthropicDriver(dc)
	case "openai-compat":
		return newOpenAICompatDriver(dc)
	case "fake-structured":
		return NewFakeStructuredDriver("fake-structured:"+dc.Model, nil), nil
	case "fake-text":
		return NewFakeTextDriver("fake-text:" + dc.Model), nil
	// Two seats have a NON-CHAIN output shape, so the generic structured fake (which returns a chain
	// array) cannot stand in for them: the World Actor authors {actor_id, attempt} and the place author
	// authors {descriptor, kind, extent_class}. Their dedicated fakes existed but were reachable only by
	// tests binding drivers directly (NewBridgeWithDrivers) — never through this factory, so
	// DREAMCHAT_BRIDGE=fake produced a server that died the moment a pressure tier fired
	// ("cannot unmarshal array into pendingPayload"). Caught by hand-driving the endpoint, which is
	// exactly the class of defect a test binding its own fakes cannot see.
	case "fake-world-actor":
		return NewFakeWorldActorDriver(), nil
	case "fake-place-author":
		return NewFakePlaceAuthorDriver(), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", dc.Provider)
	}
}
