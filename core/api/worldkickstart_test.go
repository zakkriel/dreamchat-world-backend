package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthorKickstartOffersThree(t *testing.T) {
	doc, _, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), "a harbour town at closing time", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	kseat := NewFakeWorldKickstartDriver()
	k, err := authorKickstart(context.Background(), kseat, doc, "a harbour town at closing time", doc.Arrival.CanonicalName, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(k.Scenarios) != 3 {
		t.Fatalf("scenarios = %d, want 3", len(k.Scenarios))
	}
	rec := 0
	for _, s := range k.Scenarios {
		if s.Recommended {
			rec++
		}
	}
	if rec != 1 {
		t.Fatalf("recommended scenarios = %d, want exactly 1", rec)
	}
	if strings.TrimSpace(k.Identity.CanonicalName) == "" || strings.TrimSpace(k.Identity.Descriptor) == "" {
		t.Fatalf("identity incomplete: %+v", k.Identity)
	}
}

func TestAuthorKickstartGroundsCustomOpening(t *testing.T) {
	doc, _, _ := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), "a harbour town at closing time", nil, nil, nil)
	kseat := NewFakeWorldKickstartDriver()
	k, err := authorKickstart(context.Background(), kseat, doc, "a harbour town at closing time",
		"the collector nobody expected", "I want to slip in through the kitchen while an argument is going on")
	if err != nil {
		t.Fatal(err)
	}
	if len(k.Scenarios) != 1 {
		t.Fatalf("custom opening: scenarios = %d, want exactly 1", len(k.Scenarios))
	}
}

func TestKickstartValidateRejectsUnpopulatedPlace(t *testing.T) {
	doc, _, _ := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), "a harbour town at closing time", nil, nil, nil)
	k := &kickstartDoc{
		Identity: kickstartIdentity{Descriptor: "a stranger", CanonicalName: "Someone"},
		Scenarios: []kickstartScenario{
			{Label: "x", Place: "a place that does not exist", Why: "y", Stated: "I walked in.", Recommended: true},
			{Label: "a", Place: doc.Arrival.Place, Why: "b", Stated: "I walked in."},
			{Label: "c", Place: doc.Arrival.Place, Why: "d", Stated: "I walked in."},
		},
	}
	if err := k.validate(doc, true); !IsGenesisRefusal(err) {
		t.Fatalf("unknown place accepted or wrong error class: %v", err)
	}
}

// The leash is per-mode (prod, 2026-08-21): the routed seat read a long free-text identity as an
// opening and emitted one scenario in offering mode, three times running, against a schema that
// permitted it. A leash that permits the wrong count is a prompt suggestion — so the cardinality
// the mode requires is IN the schema the seat is handed, not only in the belt that runs after.
func TestKickstartLeashCardinalityFollowsTheMode(t *testing.T) {
	for _, tc := range []struct {
		wantOptions bool
		n           float64
	}{{true, 3}, {false, 1}} {
		leash, err := kickstartLeashFor(tc.wantOptions)
		if err != nil {
			t.Fatal(err)
		}
		var s struct {
			Properties struct {
				Scenarios struct {
					MinItems float64 `json:"minItems"`
					MaxItems float64 `json:"maxItems"`
				} `json:"scenarios"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(leash, &s); err != nil {
			t.Fatal(err)
		}
		if s.Properties.Scenarios.MinItems != tc.n || s.Properties.Scenarios.MaxItems != tc.n {
			t.Fatalf("wantOptions=%v: leash scenarios min/max = %v/%v, want exactly %v",
				tc.wantOptions, s.Properties.Scenarios.MinItems, s.Properties.Scenarios.MaxItems, tc.n)
		}
	}
}
