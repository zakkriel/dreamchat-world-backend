package main

// worldkickstart.go — the seat that turns a chosen identity into a chosen opening.
// One call, two modes: offer three scenarios, or ground the user's own words as one.
// Leash then belt, exactly as world_genesis: the schema constrains, validate() refuses.

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed prompts/world_kickstart.txt
var worldKickstartPromptHeader string

//go:embed schema/world_kickstart.v1.schema.json
var worldKickstartSchemaJSON string

const (
	worldKickstartWorldMarker   = "THE WORLD (already authored and immutable — every place and person below exists):"
	worldKickstartBriefMarker   = "BRIEF (the user's own words):"
	worldKickstartWhoMarker     = "WHO THE PLAYER IS (the user's choice — it outranks your judgement completely):"
	worldKickstartOpeningMarker = "THE USER'S OWN OPENING (ground exactly this, as the single scenario):"
)

type kickstartDoc struct {
	Identity  kickstartIdentity   `json:"identity"`
	Scenarios []kickstartScenario `json:"scenarios"`
}

func buildWorldKickstartPrompt(doc *genesisDoc, brief, who, customScenario string) (string, error) {
	world, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal genesis doc for kickstart: %w", err)
	}
	var b strings.Builder
	b.WriteString(worldKickstartPromptHeader)
	b.WriteString("\n\n" + worldKickstartWorldMarker + "\n")
	b.Write(world)
	b.WriteString("\n\n" + worldKickstartBriefMarker + "\n" + brief)
	b.WriteString("\n\n" + worldKickstartWhoMarker + "\n" + who)
	if strings.TrimSpace(customScenario) != "" {
		b.WriteString("\n\n" + worldKickstartOpeningMarker + "\n" + customScenario)
	}
	return b.String(), nil
}

func authorKickstart(ctx context.Context, seat Driver, doc *genesisDoc, brief, who, customScenario string) (*kickstartDoc, error) {
	prompt, err := buildWorldKickstartPrompt(doc, brief, who, customScenario)
	if err != nil {
		return nil, err
	}
	raw, err := seat.Generate(ctx, GenRequest{Prompt: prompt, Schema: json.RawMessage(worldKickstartSchemaJSON)})
	if err != nil {
		return nil, err
	}
	var k kickstartDoc
	if err := json.Unmarshal([]byte(raw), &k); err != nil {
		return nil, refuse("the opening could not be read: %v", err)
	}
	if err := k.validate(doc, strings.TrimSpace(customScenario) == ""); err != nil {
		return nil, err
	}
	return &k, nil
}

func (k *kickstartDoc) validate(doc *genesisDoc, wantOptions bool) error {
	if strings.TrimSpace(k.Identity.Descriptor) == "" || strings.TrimSpace(k.Identity.CanonicalName) == "" {
		return refuse("the player's identity came back incomplete")
	}
	want := 1
	if wantOptions {
		want = 3
	}
	if len(k.Scenarios) != want {
		return refuse("expected %d scenario(s), got %d", want, len(k.Scenarios))
	}
	populated := map[string]bool{}
	for _, a := range doc.Cast {
		populated[strings.TrimSpace(a.StartsIn)] = true
	}
	places := map[string]bool{}
	for _, p := range doc.Places {
		places[strings.TrimSpace(p.CanonicalName)] = true
	}
	rec := 0
	for _, s := range k.Scenarios {
		if strings.TrimSpace(s.Label) == "" || strings.TrimSpace(s.Why) == "" || strings.TrimSpace(s.Stated) == "" {
			return refuse("a scenario is missing its label, why or stated line")
		}
		place := strings.TrimSpace(s.Place)
		if !places[place] {
			return refuse("scenario %q opens in %q, which is not a place this world has", s.Label, s.Place)
		}
		if !populated[place] {
			return refuse("scenario %q opens in %q, where nobody is when the player walks in", s.Label, s.Place)
		}
		if s.Recommended {
			rec++
		}
	}
	if wantOptions && rec != 1 {
		return refuse("exactly one scenario must be recommended, got %d", rec)
	}
	if !wantOptions && rec != 0 {
		return refuse("a grounded opening carries no recommendation")
	}
	return nil
}
