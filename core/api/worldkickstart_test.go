package main

import (
	"context"
	"strings"
	"testing"
)

func TestAuthorKickstartOffersThree(t *testing.T) {
	gseat := NewFakeWorldGenesisDriver()
	doc, err := authorWorld(context.Background(), gseat, "a harbour town at closing time", nil)
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
	gseat := NewFakeWorldGenesisDriver()
	doc, _ := authorWorld(context.Background(), gseat, "a harbour town at closing time", nil)
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
	gseat := NewFakeWorldGenesisDriver()
	doc, _ := authorWorld(context.Background(), gseat, "a harbour town at closing time", nil)
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
