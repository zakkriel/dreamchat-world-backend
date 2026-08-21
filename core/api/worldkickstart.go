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
	worldKickstartPlacesMarker  = "WHERE A SCENARIO MAY OPEN (the only legal `place` values — copy one verbatim; every other place is empty when the player walks in):"
	worldKickstartWhoMarker     = "WHO THE PLAYER IS (the user's choice — it outranks your judgement completely):"
	worldKickstartOpeningMarker = "THE USER'S OWN OPENING (ground exactly this, as the single scenario):"
	// worldKickstartOfferMarker closes the offering-mode prompt. The live seat proved it needed
	// saying (prod, 2026-08-21): a long free-text identity reads like an opening, and the model
	// slid into single-scenario grounding mode with no OPENING marker anywhere in the prompt.
	worldKickstartOfferMarker = "OFFER THREE SCENARIOS. The user has NOT written their own opening — the text above is only who they are. Emit exactly three scenarios, exactly one of them recommended."
)

type kickstartDoc struct {
	Identity  kickstartIdentity   `json:"identity"`
	Scenarios []kickstartScenario `json:"scenarios"`
	// NewCast is the people the chosen identity references who do not exist in the cast — "Joe, son
	// of Dalma and Harry" makes Dalma and Harry real. Genesis cast shape exactly, so the merged
	// document passes the same validate() the authored cast did. Empty is the ordinary case.
	NewCast []genesisActor `json:"new_cast,omitempty"`
}

// populatedPlaces lists the canonical names of places somebody already starts in, in authored order,
// including any extra people about to be committed. This is the same set validate() enforces —
// stated verbatim in the prompt because deriving it from the document is a join the routed seat
// failed twice in production (2026-08-20/21), picking real but empty rooms.
func populatedPlaces(doc *genesisDoc, extra []genesisActor) []string {
	populated := map[string]bool{}
	for _, a := range doc.Cast {
		populated[strings.TrimSpace(a.StartsIn)] = true
	}
	for _, a := range extra {
		populated[strings.TrimSpace(a.StartsIn)] = true
	}
	names := make([]string, 0, len(doc.Places))
	for _, p := range doc.Places {
		if name := strings.TrimSpace(p.CanonicalName); populated[name] {
			names = append(names, name)
		}
	}
	return names
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
	b.WriteString("\n\n" + worldKickstartPlacesMarker + "\n" + strings.Join(populatedPlaces(doc, nil), "\n"))
	b.WriteString("\n\n" + worldKickstartWhoMarker + "\n" + who)
	if strings.TrimSpace(customScenario) != "" {
		b.WriteString("\n\n" + worldKickstartOpeningMarker + "\n" + customScenario)
	} else {
		b.WriteString("\n\n" + worldKickstartOfferMarker)
	}
	return b.String(), nil
}

// kickstartLeashFor returns the seat schema with the scenario cardinality the MODE requires —
// exactly three when offering, exactly one when grounding the user's own opening. The published
// file keeps the permissive 1..3 envelope (both real shapes validate against it); the LEASH is
// per-call, because a leash that permits the wrong count is a prompt suggestion, not a leash
// (prod, 2026-08-21: the routed seat emitted one scenario in offering mode three times running).
func kickstartLeashFor(wantOptions bool) (json.RawMessage, error) {
	var s map[string]any
	if err := json.Unmarshal([]byte(worldKickstartSchemaJSON), &s); err != nil {
		return nil, fmt.Errorf("kickstartLeashFor: parse: %w", err)
	}
	props, _ := s["properties"].(map[string]any)
	scen, _ := props["scenarios"].(map[string]any)
	if scen == nil {
		return nil, fmt.Errorf("kickstartLeashFor: schema has no scenarios property")
	}
	n := 1
	if wantOptions {
		n = 3
	}
	scen["minItems"] = n
	scen["maxItems"] = n
	out, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("kickstartLeashFor: marshal: %w", err)
	}
	return out, nil
}

func authorKickstart(ctx context.Context, seat Driver, doc *genesisDoc, brief, who, customScenario string) (*kickstartDoc, error) {
	prompt, err := buildWorldKickstartPrompt(doc, brief, who, customScenario)
	if err != nil {
		return nil, err
	}
	leash, err := kickstartLeashFor(strings.TrimSpace(customScenario) == "")
	if err != nil {
		return nil, err
	}
	raw, err := seat.Generate(ctx, GenRequest{Prompt: prompt, Schema: leash})
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
	cast := map[string]bool{}
	for _, a := range doc.Cast {
		cast[strings.ToLower(strings.TrimSpace(a.CanonicalName))] = true
	}

	// Referenced people first: a seat that re-emits someone who already exists is echoing, not
	// authoring — dropped, not refused. What survives must stand in a real place, and counts toward
	// where a scenario may open (your father's room is a fine place to start).
	kept := k.NewCast[:0]
	for _, a := range k.NewCast {
		name := strings.TrimSpace(a.CanonicalName)
		if name == "" || strings.TrimSpace(a.Descriptor) == "" {
			return refuse("a referenced person came back without a name or a descriptor")
		}
		if cast[strings.ToLower(name)] {
			continue
		}
		where := strings.TrimSpace(a.StartsIn)
		if !places[where] {
			return refuse("%q starts in %q, which is not a place this world has", name, a.StartsIn)
		}
		kept = append(kept, a)
		populated[where] = true
	}
	k.NewCast = kept
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
