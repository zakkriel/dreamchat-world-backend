package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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
	// The decompose seat now assembles a full Prompt (header + SCENE + CANDIDATES + PLAYER INPUT tail);
	// the player's raw words ride the tail, not the whole Prompt. Match a table key that is EXACT (the
	// direct-Generate callers) OR a substring of the assembled prompt (the player's intent embedded at
	// the tail), so a test keyed by the player's text still resolves to its scripted chain.
	if out, ok := f.table[req.Prompt]; ok {
		return out, nil
	}
	for key, out := range f.table {
		if strings.Contains(req.Prompt, key) {
			return out, nil
		}
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

// fakeResolveDriver: returns a ruling/2 for CI.
// Extracts UUID from prompt via regex; echoes it as actor_id + target_id in the AttributeChanged
// event. The JSON includes both v2 fields (actor_id, truth, appearance) AND superset fields
// (summary, participant_ids) — v1-compat superset fields (summary/participant_ids) retained for
// any remaining v1 consumers; the orchestrator is v2-only since Station D Task 5.
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
	actorID := "00000000-0000-0000-0000-000000000001"
	if len(matches) > 0 {
		actorID = matches[0]
	}
	const truthText = "The attempt does not land; the target hardens and deflects."
	// Emit a ruling/2 JSON that ALSO satisfies the v1 validator (v1 uses plain json.Unmarshal so
	// extra fields are silently ignored). v1-required per-event fields: summary + participant_ids.
	// v2-required: actor_id + truth. target_id satisfies AttributeChanged per-type check.
	out := fmt.Sprintf(
		`{"reasoning":"The attempt does not land; the target hardens.","therefore":"fails","outcome":{"kind":"resolved","events":[{"type":"AttributeChanged","actor_id":%s,"target_id":%s,"truth":%s,"appearance":"The target seems unmoved.","visible":true,"summary":%s,"participant_ids":[%s]}]}}`,
		jsonStr(actorID), jsonStr(actorID), jsonStr(truthText), jsonStr(truthText), jsonStr(actorID),
	)
	return out, nil
}

// jsonStr returns a JSON-quoted string literal.
func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
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

// fakeWorldActorDriver: a deterministic stand-in for the World Actor seat (Living World / Task 8). Every
// other seat's fake stays a passive "[]" stub because nothing calls WorldActor.Generate yet ANYWHERE
// except Task 8's own direct tests (the seat is not wired into the beat — Task 9 does that); this one is
// forceable, so a test can drive runWorldActor to a real, size-appropriate eruption. It always authors a
// Communicated attributed to Mara (the seeded play world's tavern keeper, 2ac70000-…-a2), addressed to
// Jonas (…-a3) — both seeded at the Drowned Lantern (seed_drowned_lantern.sql) — matching
// world_actor.v1.schema.json. Hardcoding these seeded ids is safe: the fake is wired into every OTHER
// package test's Orchestrator too (dozens of call sites), but none of them ever exercises
// WorldActor.Generate — only worldactor_test.go calls runWorldActor directly, against the real seeded
// play world these ids belong to. Still errors when req.Schema == nil (the structured-output floor every
// other fake enforces).
// FAKE: CI stand-in for an undelivered live model. The DESIGN has no LLM-free path (POST-COMPACTION-RULINGS); this fake is scaffolding, not a design statement.
type fakeWorldActorDriver struct{ name string }

func NewFakeWorldActorDriver() Driver { return &fakeWorldActorDriver{name: "fake-world-actor"} }

func (f *fakeWorldActorDriver) Name() string                { return f.name }
func (f *fakeWorldActorDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

// fakeWorldActorMaraID/fakeWorldActorJonasID are the seeded play-world actors (seed_drowned_lantern.sql)
// this fake always attributes its authored intrusion to/at — both placed at the Drowned Lantern tavern.
const (
	fakeWorldActorMaraID  = "2ac70000-0000-0000-0000-0000000000a2"
	fakeWorldActorJonasID = "2ac70000-0000-0000-0000-0000000000a3"
)

func (f *fakeWorldActorDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: world-actor driver used without a schema", f.name)
	}
	return fmt.Sprintf(
		`{"actor_id":%s,"attempt":{"type":"Communicated","stated":"a commotion breaks out at the bar",`+
			`"listener_id":%s,"content":"Oi — mind yourself!"}}`,
		jsonStr(fakeWorldActorMaraID), jsonStr(fakeWorldActorJonasID),
	), nil
}
