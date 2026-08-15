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
	"log"
	"time"
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
	// Repair marks a RETRY of a call whose previous answer was refused. It exists because of a trap
	// this repo has already been bitten by once and I walked straight back into: "determinism plus a
	// fatal path equals a permanent trap — any retry of a deterministic decision that failed will
	// fail the same way" (2026-08-08 handover §6, the journey livelock).
	//
	// Setting temperature 0 on the mechanical seats fixed decompose, which has no repair loop and is
	// a pure classification. Resolve DOES have a repair loop, so temperature 0 turned a flaky seat
	// into one that failed identically twice: measured live, `AttributeChanged requires target_id`
	// rejected on attempt 1/2 and again on 2/2, ~14s spent asking the same question of a model that
	// had no reason to answer differently. A driver that pins temperature honours this flag by
	// letting the repair explore.
	Repair bool
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
		b.seats[s.Name] = instrument(s.Name, bound)
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
		b.seats[s.Name] = instrument(s.Name, bd)
	}
	return b, nil
}

func (b *Bridge) Driver(seat string) Driver { return b.seats[seat] }

// --- Instrumentation ---------------------------------------------------------
// timedDriver logs how long a seat took, every time it is called.
//
// It exists because the founder asked "how long did one reply take" and the logs could not answer.
// A beat fans out across up to seven seats and two of them retry, so the only honest answer is
// per-call — a total alone cannot tell you whether a slow beat was one slow seat or four ordinary
// ones, and the retries are invisible in a total by construction.
//
// A DECORATOR, applied once in the binding path, rather than a timer at each call site: the seats
// are called from beatsstream, orchestrator and worldturn, and an instrument you have to remember to
// add at three call sites is an instrument that will be missing from the fourth.
type timedDriver struct {
	Driver
	seat string
}

func (t timedDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	start := time.Now()
	sink := costSinkFrom(ctx)
	usdBefore, inBefore, outBefore, cachedBefore, _ := sink.snapshot()
	out, err := t.Driver.Generate(ctx, req)
	ms := time.Since(start).Milliseconds()
	// The driver sees the bill but not the seat; this wrapper knows the seat but not the bill. The
	// delta across the call joins them (costsink.go documents why a delta is sound here).
	usdAfter, inAfter, outAfter, cachedAfter, _ := sink.snapshot()
	usd, tokIn, tokOut, cached := usdAfter-usdBefore, inAfter-inBefore, outAfter-outBefore, cachedAfter-cachedBefore
	// Outcome, not just duration: a 6-second failure and a 6-second success are the same number and
	// very different problems, and the retry that follows a failure is the founder's dead air.
	status := "ok"
	if err != nil {
		status = "ERR"
	}
	log.Printf("seat timing: seat=%s model=%s ms=%d status=%s chars=%d tok_in=%d cached=%d tok_out=%d cost_usd=%.6f",
		t.seat, t.Driver.Name(), ms, status, len(out), tokIn, cached, tokOut, usd)
	// Raindrop/Workshop: the same per-call truth the log line carries, as a span on the beat's
	// interaction (raindrop.go; no-op when the context carries none).
	trackSeatCall(ctx, t.seat, t.Driver.Name(), req.Prompt, out, start, err)
	return out, err
}

// GenerateStream keeps the streaming capability visible through the decorator. Without this the type
// assertion in the narrate path fails and a streaming-capable driver silently loses streaming — the
// exact class of bug where an instrument changes the thing it measures.
func (t timedDriver) GenerateStream(ctx context.Context, req GenRequest, onDelta func(string)) (string, error) {
	sd, ok := t.Driver.(StreamingDriver)
	if !ok {
		return "", fmt.Errorf("seat %s: driver %s does not stream", t.seat, t.Driver.Name())
	}
	start := time.Now()
	out, err := sd.GenerateStream(ctx, req, onDelta)
	status := "ok"
	if err != nil {
		status = "ERR"
	}
	log.Printf("seat timing: seat=%s model=%s ms=%d status=%s chars=%d streamed=true",
		t.seat, t.Driver.Name(), time.Since(start).Milliseconds(), status, len(out))
	trackSeatCall(ctx, t.seat, t.Driver.Name(), req.Prompt, out, start, err)
	return out, err
}

// instrument wraps a bound driver, preserving streaming only when the underlying driver has it —
// wrapping a non-streaming driver in a type that advertises GenerateStream would make every seat
// claim a capability it does not have.
func instrument(seat string, d Driver) Driver {
	if _, ok := d.(StreamingDriver); ok {
		return timedDriver{Driver: d, seat: seat}
	}
	return struct{ Driver }{timedDriver{Driver: d, seat: seat}}
}

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
	// fake-intent is the DEV decompose seat: fake-structured built HERE gets a nil table, so it can
	// only ever answer "[]" — a server that streams a correct frame sequence and commits nothing.
	// Tests keep using fake-structured with their own scripted tables; a human or a frontend driving
	// the server by hand gets a chain bound to the real candidate whitelist instead.
	case "fake-intent":
		return NewFakeIntentDriver(), nil
	// fake-resolve and fake-cognition close the SAME hole the two cases below closed for the world
	// actor and place author: their dedicated fakes existed but were reachable only by tests binding
	// drivers directly, so DREAMCHAT_BRIDGE=fake pointed resolve and both cognition seats at the
	// chain-shaped generic fake. Resolve is reached by ADJUDICATED attempts — everything outside the
	// passthrough three (applyNPCDecisions, orchestrator.go) — so an NPC decision or a World Actor
	// intrusion of any other type would hand it "[]" where a ruling/2 belongs. Cognition's own fake
	// also answers "[]", so that seat was accidentally right rather than actually bound; it is wired
	// here so the next person reading the config sees seven seats and seven matching stand-ins.
	case "fake-resolve":
		return NewFakeResolveDriver(), nil
	case "fake-cognition":
		return NewFakeCognitionDriver(), nil
	case "fake-text":
		return NewFakeTextDriver("fake-text:" + dc.Model), nil
	// narrate's own stand-in. It used to point at fake-text, which reports no capabilities and errors
	// on a schema, so every hand-driven beat burned both structured attempts and fell to the plain
	// prose fallback — the one narration path with no belts on it (see fakeNarrateDriver's header).
	// fake-text stays available: it is still the honest stand-in for a genuinely text-only model.
	case "fake-narrate":
		return NewFakeNarrateDriver("fake-narrate:" + dc.Model), nil
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
