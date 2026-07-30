package main

import (
	"context"
	"testing"
)

// (a) flip case: therefore=succeeds but outcome=bounce — must be rejected.
func TestRulingV2_FlipRejected(t *testing.T) {
	flip := `{"reasoning":"the press lands cleanly","therefore":"succeeds","outcome":{"kind":"bounce","reason":"cannot"}}`
	if _, err := DecodeAndValidateRulingV2(flip); err == nil {
		t.Fatal("therefore=succeeds with outcome=bounce accepted — flip bug in v2")
	}
}

// (b) valid failure ruling with appearance + one receiver variant must decode correctly.
func TestRulingV2_ValidFailureWithAppearanceAndVariant(t *testing.T) {
	valid := `{
		"reasoning": "she is guarded; the press does not land",
		"therefore": "fails",
		"outcome": {
			"kind": "resolved",
			"events": [
				{
					"type": "AttributeChanged",
					"actor_id": "33333333-3333-3333-3333-333333333333",
					"target_id": "44444444-4444-4444-4444-444444444444",
					"truth": "Mara, secretly shaken, masks it and deflects.",
					"appearance": "Mara seems unmoved and shrugs it off.",
					"receiver_variants": [
						{"receiver_id": "55555555-5555-5555-5555-555555555555", "text": "you catch the tremor in her hand"}
					],
					"visible": true
				}
			]
		}
	}`
	r, err := DecodeAndValidateRulingV2(valid)
	if err != nil {
		t.Fatalf("valid failure ruling rejected: %v", err)
	}
	ev := r.Outcome.Events[0]
	if ev.Truth != "Mara, secretly shaken, masks it and deflects." {
		t.Fatalf("wrong truth: %q", ev.Truth)
	}
	if ev.Appearance != "Mara seems unmoved and shrugs it off." {
		t.Fatalf("wrong appearance: %q", ev.Appearance)
	}
	if len(ev.ReceiverVariants) != 1 || ev.ReceiverVariants[0].Text != "you catch the tremor in her hand" {
		t.Fatalf("wrong receiver variants: %+v", ev.ReceiverVariants)
	}
}

// (c) ruled ActorMoved without to_target_id must be rejected.
func TestRulingV2_ActorMovedMissingToTargetID(t *testing.T) {
	bad := `{
		"reasoning": "she slips away",
		"therefore": "fails",
		"outcome": {
			"kind": "resolved",
			"events": [
				{
					"type": "ActorMoved",
					"actor_id": "33333333-3333-3333-3333-333333333333",
					"truth": "Mara moves to the door."
				}
			]
		}
	}`
	if _, err := DecodeAndValidateRulingV2(bad); err == nil {
		t.Fatal("ActorMoved without to_target_id accepted — per-type check missing")
	}
}

// (d) AttrWrite with tier=3 must be rejected.
func TestRulingV2_AttrWriteTierThreeRejected(t *testing.T) {
	bad := `{
		"reasoning": "the door opens",
		"therefore": "succeeds",
		"outcome": {
			"kind": "resolved",
			"events": [
				{
					"type": "AttributeChanged",
					"actor_id": "33333333-3333-3333-3333-333333333333",
					"target_id": "44444444-4444-4444-4444-444444444444",
					"truth": "The door opens.",
					"visible": true
				}
			],
			"attribute_writes": [
				{
					"target_id": "44444444-4444-4444-4444-444444444444",
					"attribute": "open",
					"value": true,
					"tier": 3
				}
			]
		}
	}`
	if _, err := DecodeAndValidateRulingV2(bad); err == nil {
		t.Fatal("attr_write with tier=3 accepted — tier check missing")
	}
}

// (e) FakeResolveDriver output must decode via DecodeAndValidateRulingV2 and echo a prompt uuid
// into ActorID.
func TestRulingV2_FakeResolveDriverOutput(t *testing.T) {
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
	r, err := DecodeAndValidateRulingV2(generated)
	if err != nil {
		t.Fatalf("DecodeAndValidateRulingV2 failed on fake output: %v", err)
	}
	if r.Outcome.Kind != "resolved" || len(r.Outcome.Events) == 0 {
		t.Fatalf("expected resolved with events, got kind=%q events=%d", r.Outcome.Kind, len(r.Outcome.Events))
	}
	if r.Outcome.Events[0].ActorID != "12345678-1234-1234-1234-123456789abc" {
		t.Fatalf("expected actor_id to echo prompt uuid, got %q", r.Outcome.Events[0].ActorID)
	}
}
