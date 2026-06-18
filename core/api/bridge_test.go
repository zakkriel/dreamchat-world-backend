package main

import (
	"context"
	"strings"
	"testing"
)

// Capability floor (fail closed): the decompose seat REQUIRES structured output. A driver that does
// not REPORT that capability cannot be bound — so an out-of-vocab event can't even be ATTEMPTED
// through a non-constrained model (structurally impossible, not merely unlikely).
func TestBindSeat_DecomposeRefusesNonStructuredDriver(t *testing.T) {
	if _, err := BindSeat(SeatDecompose, NewFakeTextDriver("free-text:m")); err == nil {
		t.Fatalf("decompose bound a non-structured driver — the capability floor failed (no leash)")
	} else if !strings.Contains(err.Error(), string(CapStructuredOutput)) {
		t.Fatalf("error %v should name the missing capability %q", err, CapStructuredOutput)
	}
	if _, err := BindSeat(SeatDecompose, NewFakeStructuredDriver("structured:m", nil)); err != nil {
		t.Fatalf("decompose rejected a structured driver: %v", err)
	}
	if _, err := BindSeat(SeatNarrate, NewFakeTextDriver("free-text:m")); err != nil {
		t.Fatalf("narrate (requires nothing) rejected a free-text driver: %v", err)
	}
}

// Validated against the driver's REPORTED capabilities, never a config label (Image-Platform lesson).
func TestBindSeat_ValidatesReportedCapabilityNotLabel(t *testing.T) {
	d := NewFakeTextDriver("totally-a-structured-model-wink") // label lies; reports {}
	if _, err := BindSeat(SeatDecompose, d); err == nil {
		t.Fatalf("bound a driver by its label, not its reported capabilities")
	}
}

// Per-seat routing is real: re-pointing one seat's configured driver changes ONLY that seat.
func TestBridge_PerSeatRouting_RepointIsolatesOneSeat(t *testing.T) {
	cfg := SeatConfig{
		"decompose": {Provider: "fake-structured", Model: "m-decompose-1"},
		"narrate":   {Provider: "fake-text", Model: "m-narrate-1"},
	}
	b, err := NewBridge(cfg, DefaultDriverFactory, SeatDecompose, SeatNarrate)
	if err != nil {
		t.Fatalf("bridge build: %v", err)
	}
	narBefore, decBefore := b.Driver("narrate").Name(), b.Driver("decompose").Name()

	cfg["decompose"] = DriverConfig{Provider: "fake-structured", Model: "m-decompose-2"} // re-point ONLY decompose
	b2, err := NewBridge(cfg, DefaultDriverFactory, SeatDecompose, SeatNarrate)
	if err != nil {
		t.Fatalf("bridge rebuild: %v", err)
	}
	if b2.Driver("narrate").Name() != narBefore {
		t.Fatalf("narrate changed when only decompose was re-pointed: %q -> %q", narBefore, b2.Driver("narrate").Name())
	}
	if b2.Driver("decompose").Name() == decBefore {
		t.Fatalf("decompose did NOT change when re-pointed: still %q", decBefore)
	}
	if !strings.Contains(b2.Driver("decompose").Name(), "m-decompose-2") {
		t.Fatalf("decompose did not bind the new model: %q", b2.Driver("decompose").Name())
	}
}

// Bridge build fails CLOSED if a seat's configured driver underpowers its capability floor.
func TestBridge_FailsClosedOnUnderpoweredDecomposeDriver(t *testing.T) {
	cfg := SeatConfig{
		"decompose": {Provider: "fake-text", Model: "m"}, // lacks structured output
		"narrate":   {Provider: "fake-text", Model: "m"},
	}
	if _, err := NewBridge(cfg, DefaultDriverFactory, SeatDecompose, SeatNarrate); err == nil {
		t.Fatalf("bridge built with a non-structured decompose driver — should fail closed")
	}
}

// Structured generation is schema-valid BY CONSTRUCTION: the structured fake models constrained
// decoding — it returns only table chains (valid) or an empty chain, never out-of-vocab.
func TestStructuredDriver_NeverEmitsOutOfVocab(t *testing.T) {
	d := NewFakeStructuredDriver("structured:m", map[string]string{"go": `[{"type":"move","to":"square"}]`})
	out, err := d.Generate(context.Background(), GenRequest{Prompt: "go", Schema: []byte(`{}`)})
	if err != nil {
		t.Fatalf("structured generate: %v", err)
	}
	if _, err := DecodeAndValidateChain(out); err != nil {
		t.Fatalf("structured driver emitted non-schema-valid output: %v (%s)", err, out)
	}
	out2, _ := d.Generate(context.Background(), GenRequest{Prompt: "unknown", Schema: []byte(`{}`)})
	if _, err := DecodeAndValidateChain(out2); err != nil {
		t.Fatalf("structured driver fallback not schema-valid: %v (%s)", err, out2)
	}
}
