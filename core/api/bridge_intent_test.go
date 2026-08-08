package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The seeded Drowned Lantern's shape, as the decompose seat sees it: present actors, the current
// room, and the objects in it — including the portal artifacts a move binds to.
var devTestCandidates = []Candidate{
	{ID: "2ac70000-0000-0000-0000-0000000000a2", Name: "Mara", Kind: "actor"},
	{ID: "2ac70000-0000-0000-0000-0000000000a3", Name: "the muscle by the bar", Kind: "actor"},
	{ID: "210c0000-0000-0000-0000-0000000000d1", Name: "The Drowned Lantern", Kind: "location"},
	{ID: "a27f0000-0000-0000-0000-0000000000b1", Name: "the bar", Kind: "artifact"},
	{ID: "a27f0000-0000-0000-0000-0000000000b2", Name: "Back Door", Kind: "artifact"},
	{ID: "a27f0000-0000-0000-0000-0000000000b3", Name: "Sealed Note (gray wax)", Kind: "artifact"},
}

// devPrompt builds the REAL decompose prompt rather than a hand-written fixture, so these tests fail
// if buildDecomposePrompt's layout and the dev driver's parser ever drift apart. That drift is the
// whole reason the section markers are shared constants.
func devPrompt(t *testing.T, input string) string {
	t.Helper()
	return buildDecomposePrompt(PerceptionPayload{
		Lines:      []string{"You stepped into the Drowned Lantern."},
		Candidates: devTestCandidates,
	}, input)
}

// devChain drives the driver exactly as the seat does — through Generate with the v2 leash attached
// — and validates the output with the engine's own belt, so a test can never accept a chain the
// orchestrator would reject.
func devChain(t *testing.T, input string) []Attempt {
	t.Helper()
	raw, err := NewFakeIntentDriver().Generate(context.Background(), GenRequest{
		Prompt: devPrompt(t, input),
		Schema: json.RawMessage(beatChainV2SchemaJSON),
	})
	if err != nil {
		t.Fatalf("Generate(%q): %v", input, err)
	}
	chain, err := DecodeAndValidateChainV2(raw)
	if err != nil {
		t.Fatalf("Generate(%q) produced a chain the engine rejects: %v\nraw: %s", input, err, raw)
	}
	return chain
}

// Speech binds the named PRESENT actor and carries what was actually said, so the beat has a real
// Communicated to adjudicate instead of an empty chain.
func TestFakeIntent_SpeechBindsNamedActor(t *testing.T) {
	chain := devChain(t, "tell Mara about the sealed note")

	if len(chain) != 1 {
		t.Fatalf("chain = %+v, want exactly one attempt", chain)
	}
	a := chain[0]
	if a.Type != "Communicated" {
		t.Fatalf("type = %q, want Communicated", a.Type)
	}
	if a.ListenerID != "2ac70000-0000-0000-0000-0000000000a2" {
		t.Fatalf("listener_id = %q, want Mara's id", a.ListenerID)
	}
	if a.Content != "the sealed note" {
		t.Fatalf("content = %q, want the words after \"about\"", a.Content)
	}
}

// Quoted words are taken verbatim — the narrate seat's verbatim-speech belt matches on them, so a
// paraphrase here would be rejected downstream.
func TestFakeIntent_QuotedSpeechIsTakenVerbatim(t *testing.T) {
	chain := devChain(t, `say to Mara "the tide turns at dusk"`)

	if len(chain) != 1 || chain[0].Type != "Communicated" {
		t.Fatalf("chain = %+v, want one Communicated", chain)
	}
	if chain[0].Content != "the tide turns at dusk" {
		t.Fatalf("content = %q, want the quoted words verbatim", chain[0].Content)
	}
}

// Movement binds a named location-or-artifact candidate. NOTE: in the seeded world today this can
// never fire — no remote location is a candidate and portals carry `connects` rather than a
// `location_id`, so nothing movable is ever offered to the seat (SPEC-030). The driver is written
// correctly anyway and this test pins that with a synthetic candidate set, so the day SPEC-030 is
// ruled on, the dev seat already binds. Do not read this test as "movement works end to end".
func TestFakeIntent_MovementBindsNamedDestination(t *testing.T) {
	chain := devChain(t, "go through the Back Door")

	if len(chain) != 1 || chain[0].Type != "ActorMoved" {
		t.Fatalf("chain = %+v, want one ActorMoved", chain)
	}
	if chain[0].ToTargetID != "a27f0000-0000-0000-0000-0000000000b2" {
		t.Fatalf("to_target_id = %q, want the Back Door's id", chain[0].ToTargetID)
	}
	if chain[0].DurationClass != "" {
		t.Fatalf("duration_class = %q, want empty — a move's length is physics, never a stated class", chain[0].DurationClass)
	}
}

// A question is a QUERY, not an action: it binds ids and carries no outcome.
func TestFakeIntent_QuestionBecomesQuery(t *testing.T) {
	chain := devChain(t, "look at the bar")

	if len(chain) != 1 || chain[0].Type != "QUERY" {
		t.Fatalf("chain = %+v, want one QUERY", chain)
	}
	if len(chain[0].QueryTargetIDs) != 1 || chain[0].QueryTargetIDs[0] != "a27f0000-0000-0000-0000-0000000000b1" {
		t.Fatalf("query_target_ids = %v, want the bar's id", chain[0].QueryTargetIDs)
	}
}

// The most specific reference wins: "the muscle by the bar" must not be read as "the bar".
func TestFakeIntent_LongestNameWinsOverASubstring(t *testing.T) {
	chain := devChain(t, "ask the muscle by the bar about the note")

	if len(chain) != 1 || chain[0].Type != "Communicated" {
		t.Fatalf("chain = %+v, want one Communicated", chain)
	}
	if chain[0].ListenerID != "2ac70000-0000-0000-0000-0000000000a3" {
		t.Fatalf("listener_id = %q, want the muscle's id, not the bar's", chain[0].ListenerID)
	}
}

// A genuine tie is refused, not guessed — the same obligation the live seat has, and the only way
// the frontend's unresolved-candidate surface can be exercised without API keys.
func TestFakeIntent_TiedReferenceIsUnresolvedNotAGuess(t *testing.T) {
	raw, err := NewFakeIntentDriver().Generate(context.Background(), GenRequest{
		Prompt: buildDecomposePrompt(PerceptionPayload{Candidates: []Candidate{
			{ID: "2ac70000-0000-0000-0000-0000000000a4", Name: "a hooded figure", Kind: "actor"},
			{ID: "2ac70000-0000-0000-0000-0000000000a5", Name: "a hooded figure", Kind: "actor"},
		}}, "ask a hooded figure about the tide"),
		Schema: json.RawMessage(beatChainV2SchemaJSON),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	chain, err := DecodeAndValidateChainV2(raw)
	if err != nil {
		t.Fatalf("chain rejected by the engine: %v\nraw: %s", err, raw)
	}
	if len(chain) != 1 || chain[0].Type != "UNRESOLVED" {
		t.Fatalf("chain = %+v, want one UNRESOLVED", chain)
	}
	if len(chain[0].CandidateIDs) != 2 {
		t.Fatalf("candidate_ids = %v, want both tied ids (the schema requires at least two)", chain[0].CandidateIDs)
	}
}

// Out-of-vocab stays impossible: prose the driver does not understand commits nothing (C-5) rather
// than inventing an attempt.
func TestFakeIntent_UnrecognisedProseCommitsNothing(t *testing.T) {
	for _, input := range []string{
		"ponder the meaning of the tides", // no verb it knows
		"go to the harbour",               // known verb, but nothing in the room is named that
		"",                                // empty input
	} {
		if chain := devChain(t, input); len(chain) != 0 {
			t.Fatalf("input %q produced %+v, want an empty chain", input, chain)
		}
	}
}

// The structured-output floor every fake enforces: no schema, no generation.
func TestFakeIntent_RefusesWithoutASchema(t *testing.T) {
	if _, err := NewFakeIntentDriver().Generate(context.Background(), GenRequest{Prompt: devPrompt(t, "look at the bar")}); err == nil {
		t.Fatal("Generate without a schema must fail — the structured-output floor")
	}
}

// THE REGRESSION. DREAMCHAT_BRIDGE=fake bound decompose, resolve and both cognition seats to the
// generic chain fake, which the factory builds with a NIL table: decompose could only answer "[]",
// so the dev server streamed a correct frame sequence and committed nothing, and resolve was never
// reached to notice it had a chain-shaped stand-in where a ruling/2 belongs. Every existing test
// bound its own drivers, so none of them could see it. This one goes through the real factory.
func TestFakeBridge_FactoryBuiltSeatsProduceTheirOwnShape(t *testing.T) {
	t.Setenv("DREAMCHAT_BRIDGE", "fake")

	bridge, err := NewBridge(defaultSeatConfig(), DefaultDriverFactory,
		SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor, SeatPlaceAuthor)
	if err != nil {
		t.Fatalf("NewBridge with the fake seat config: %v", err)
	}

	// Decompose must bind a real id from the candidate whitelist, not answer with the empty chain.
	rawChain, err := bridge.Driver(SeatDecompose.Name).Generate(context.Background(), GenRequest{
		Prompt: devPrompt(t, "tell Mara about the sealed note"),
		Schema: json.RawMessage(beatChainV2SchemaJSON),
	})
	if err != nil {
		t.Fatalf("factory-built decompose seat: %v", err)
	}
	chain, err := DecodeAndValidateChainV2(rawChain)
	if err != nil {
		t.Fatalf("factory-built decompose produced an invalid chain: %v\nraw: %s", err, rawChain)
	}
	if len(chain) == 0 {
		t.Fatal("factory-built decompose seat returned an empty chain — the fake bridge commits nothing and the testbed is inert")
	}

	// Resolve must return a ruling, not a chain. "[]" here is the shape mismatch that was hidden
	// behind decompose never producing anything to adjudicate.
	rawRuling, err := bridge.Driver(SeatResolve.Name).Generate(context.Background(), GenRequest{
		Prompt: "resolve for actor " + devTestCandidates[0].ID,
		Schema: json.RawMessage(rulingV2SchemaJSON),
	})
	if err != nil {
		t.Fatalf("factory-built resolve seat: %v", err)
	}
	if strings.TrimSpace(rawRuling) == "[]" {
		t.Fatal("factory-built resolve seat returned a chain-shaped \"[]\" where a ruling/2 belongs")
	}
	var ruling Ruling
	if err := json.Unmarshal([]byte(rawRuling), &ruling); err != nil {
		t.Fatalf("factory-built resolve produced no ruling: %v\nraw: %s", err, rawRuling)
	}
	if ruling.Reasoning == "" || ruling.Therefore == "" {
		t.Fatalf("ruling missing required fields: %+v", ruling)
	}
}

// The parser reads back exactly what the writer emitted, including a name containing brackets — the
// kind is split from the LAST "  (", so "Sealed Note (gray wax)" keeps its brackets and its kind.
func TestParseDecomposeCandidates_RoundTripsTheWriter(t *testing.T) {
	got := parseDecomposeCandidates(devPrompt(t, "look at the bar"))

	if len(got) != len(devTestCandidates) {
		t.Fatalf("parsed %d candidates, want %d: %+v", len(got), len(devTestCandidates), got)
	}
	for i, want := range devTestCandidates {
		if got[i] != want {
			t.Fatalf("candidate %d = %+v, want %+v", i, got[i], want)
		}
	}
	if input := parseDecomposePlayerInput(devPrompt(t, "look at the bar")); input != "look at the bar" {
		t.Fatalf("player input = %q, want the raw words", input)
	}
}

// Disambiguated labels have to survive the round trip the player actually makes: ask vaguely, be
// offered the detail, then answer with it. Both halves must work or the ask is unanswerable in one
// direction or the other.
var devTwinCandidates = []Candidate{
	{ID: "2ac70000-0000-0000-0000-0000000000a4", Name: "a hooded figure by the ballast crate", Kind: "actor"},
	{ID: "2ac70000-0000-0000-0000-0000000000aa", Name: "a hooded figure by the bar", Kind: "actor"},
}

func devChainWith(t *testing.T, input string, cands []Candidate) []Attempt {
	t.Helper()
	raw, err := NewFakeIntentDriver().Generate(context.Background(), GenRequest{
		Prompt: buildDecomposePrompt(PerceptionPayload{Candidates: cands}, input),
		Schema: json.RawMessage(beatChainV2SchemaJSON),
	})
	if err != nil {
		t.Fatalf("Generate(%q): %v", input, err)
	}
	chain, err := DecodeAndValidateChainV2(raw)
	if err != nil {
		t.Fatalf("Generate(%q) produced a chain the engine rejects: %v\nraw: %s", input, err, raw)
	}
	return chain
}

// The VAGUE ask still raises the choice. Detail exists so the player CAN be specific, not so they
// must be: "the hooded figure" must still name both and produce UNRESOLVED.
func TestFakeIntent_VagueAskStillTiesAcrossDetailedLabels(t *testing.T) {
	chain := devChainWith(t, "ask the hooded figure about the note", devTwinCandidates)

	if len(chain) != 1 || chain[0].Type != "UNRESOLVED" {
		t.Fatalf("chain = %+v, want one UNRESOLVED — the vague ask must still raise the choice", chain)
	}
	if len(chain[0].CandidateIDs) != 2 {
		t.Fatalf("candidate_ids = %v, want both figures", chain[0].CandidateIDs)
	}
}

// And the SPECIFIC answer binds the one the player picked — the detail they were shown is the detail
// they can say back. Without this the whole ruling is decorative.
func TestFakeIntent_DetailedAnswerBindsTheRightOne(t *testing.T) {
	chain := devChainWith(t, "ask the hooded figure by the bar about the note", devTwinCandidates)

	if len(chain) != 1 || chain[0].Type != "Communicated" {
		t.Fatalf("chain = %+v, want one Communicated — the specific answer must bind", chain)
	}
	if chain[0].ListenerID != "2ac70000-0000-0000-0000-0000000000aa" {
		t.Fatalf("listener_id = %q, want the figure AT THE BAR", chain[0].ListenerID)
	}
}

// Labels are authored as descriptors ("a hooded figure") but players type "the hooded figure".
// Requiring the article to match made the most natural sentence in the game bind nothing.
func TestFakeIntent_LeadingArticleIsNotPartOfTheReference(t *testing.T) {
	chain := devChainWith(t, "ask the keeper about the tide", []Candidate{
		{ID: "2ac70000-0000-0000-0000-0000000000a2", Name: "the keeper", Kind: "actor"},
	})
	if len(chain) != 1 || chain[0].ListenerID != "2ac70000-0000-0000-0000-0000000000a2" {
		t.Fatalf("chain = %+v, want the keeper bound", chain)
	}
	chain = devChainWith(t, "ask the hooded figure about the tide", []Candidate{
		{ID: "2ac70000-0000-0000-0000-0000000000a4", Name: "a hooded figure", Kind: "actor"},
	})
	if len(chain) != 1 || chain[0].ListenerID != "2ac70000-0000-0000-0000-0000000000a4" {
		t.Fatalf(`chain = %+v, want "a hooded figure" bound from "the hooded figure"`, chain)
	}
}
