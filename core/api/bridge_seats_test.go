package main

import (
	"context"
	"testing"
)

func TestNewSeatsBindWithCapabilityFloor(t *testing.T) {
	for _, s := range []Seat{SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor} {
		if _, err := BindSeat(s, NewFakeResolveDriver()); err != nil {
			t.Fatalf("structured fake failed to bind %s: %v", s.Name, err)
		}
		if _, err := BindSeat(s, NewFakeTextDriver("text-driver")); err == nil {
			t.Fatalf("text driver bound to %s — capability floor broken", s.Name)
		}
	}
}

// Extra sanity check: NewFakeResolveDriver().Generate should return a valid ruling/2
// with actor_id echoing the uuid found in the prompt (updated from v1 → v2 in Task 1).
func TestFakeResolveDriverReturnsValidRuling(t *testing.T) {
	driver := NewFakeResolveDriver()
	ctx := context.Background()
	req := GenRequest{
		Prompt: "test: 12345678-1234-1234-1234-123456789abc is the target",
		Schema: []byte("{}"),
	}
	generated, err := driver.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Fake now emits ruling/2; validate with the v2 validator.
	ruling, err := DecodeAndValidateRulingV2(generated)
	if err != nil {
		t.Fatalf("DecodeAndValidateRulingV2 failed: %v", err)
	}
	// Verify the outcome is resolved
	if ruling.Outcome.Kind != "resolved" {
		t.Fatalf("expected outcome kind 'resolved', got %q", ruling.Outcome.Kind)
	}
	// Verify there's at least one event
	if len(ruling.Outcome.Events) == 0 {
		t.Fatalf("expected at least one event, got none")
	}
	// Verify actor_id echoes the prompt uuid (v2 shape: actor_id replaces participant_ids).
	firstEvent := ruling.Outcome.Events[0]
	if firstEvent.ActorID != "12345678-1234-1234-1234-123456789abc" {
		t.Fatalf("expected actor_id '12345678-1234-1234-1234-123456789abc', got %q", firstEvent.ActorID)
	}
}
