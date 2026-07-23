package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

// fakeStructuredDriver models CONSTRAINED DECODING for CI: it can ONLY return schema-valid chains
// (from its table) or an empty chain — it CANNOT emit out-of-vocab. This is the deterministic CI
// stand-in for the decompose leash (the live equivalent is Anthropic strict tool-use).
type fakeStructuredDriver struct {
	name  string
	table map[string]string // prompt → schema-valid chain JSON
}

func NewFakeStructuredDriver(name string, table map[string]string) Driver {
	if table == nil {
		table = map[string]string{}
	}
	return &fakeStructuredDriver{name: name, table: table}
}

func (f *fakeStructuredDriver) Name() string                { return f.name }
func (f *fakeStructuredDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (f *fakeStructuredDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: structured driver used without a schema", f.name)
	}
	if out, ok := f.table[req.Prompt]; ok {
		return out, nil
	}
	return "[]", nil // unknown prose → empty chain (commits nothing, C-5); never out-of-vocab
}

// fakeTextDriver: free text only; reports NO capabilities. It CANNOT bind to the decompose seat
// (capability floor fails closed) — the structural proof that out-of-vocab can't even be attempted
// through a non-constrained model.
type fakeTextDriver struct{ name string }

func NewFakeTextDriver(name string) Driver { return &fakeTextDriver{name: name} }

func (f *fakeTextDriver) Name() string                { return f.name }
func (f *fakeTextDriver) Capabilities() CapabilitySet { return CapabilitySet{} }

func (f *fakeTextDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema != nil {
		return "", fmt.Errorf("%s: cannot do structured generation (no capability)", f.name)
	}
	out := "Scene:"
	for _, l := range req.Payload.Lines {
		out += " " + l
	}
	return out, nil
}

// fakeResolveDriver: returns a valid ruling/1 for CI (stand-in for the undelivered resolve station).
// Extracts UUID from prompt via regex; uses it as the participant_id in the AttributeChanged event.
// FAKE: CI stand-in for an undelivered station. The DESIGN has no LLM-free path (POST-COMPACTION-RULINGS); this fake is scaffolding, not a design statement.
type fakeResolveDriver struct{ name string }

func NewFakeResolveDriver() Driver { return &fakeResolveDriver{name: "fake-resolve"} }

func (f *fakeResolveDriver) Name() string                { return f.name }
func (f *fakeResolveDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (f *fakeResolveDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: resolve driver used without a schema", f.name)
	}
	// Extract UUID from prompt using regex: [0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}
	uuidRegex := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	matches := uuidRegex.FindStringSubmatch(req.Prompt)
	participantID := "00000000-0000-0000-0000-000000000001"
	if len(matches) > 0 {
		participantID = matches[0]
	}

	ruling := Ruling{
		Reasoning: "The attempt does not land; the target hardens.",
		Therefore: "fails",
		Outcome: RulingOutcome{
			Kind: "resolved",
			Events: []RuledEvent{
				{
					Type:           "AttributeChanged",
					Summary:        "the attempt does not land; the target hardens",
					ParticipantIDs: []string{participantID},
					Visible:        true,
				},
			},
		},
	}
	data, _ := json.Marshal(ruling)
	return string(data), nil
}

// fakeCognitionDriver: returns empty decision list for CI (stand-in for the undelivered cognition station).
// FAKE: CI stand-in for an undelivered station. The DESIGN has no LLM-free path (POST-COMPACTION-RULINGS); this fake is scaffolding, not a design statement.
type fakeCognitionDriver struct{ name string }

func NewFakeCognitionDriver() Driver { return &fakeCognitionDriver{name: "fake-cognition"} }

func (f *fakeCognitionDriver) Name() string                { return f.name }
func (f *fakeCognitionDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (f *fakeCognitionDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: cognition driver used without a schema", f.name)
	}
	return "[]", nil
}

// fakeWorldActorDriver: returns empty action list for CI (stand-in for the undelivered world-actor station).
// FAKE: CI stand-in for an undelivered station. The DESIGN has no LLM-free path (POST-COMPACTION-RULINGS); this fake is scaffolding, not a design statement.
type fakeWorldActorDriver struct{ name string }

func NewFakeWorldActorDriver() Driver { return &fakeWorldActorDriver{name: "fake-world-actor"} }

func (f *fakeWorldActorDriver) Name() string                { return f.name }
func (f *fakeWorldActorDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (f *fakeWorldActorDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: world-actor driver used without a schema", f.name)
	}
	return "[]", nil
}
