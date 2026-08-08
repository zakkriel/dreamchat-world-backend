package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"time"
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

// fakePlaceAuthorDriver: a deterministic stand-in for the place_author seat (Journey rung 2, Task 8).
// Authors the SAME identity SHAPE every call — kind/extent_class fixed — but the descriptor carries a
// per-process-run suffix (fakePlaceAuthorRunSeed, captured ONCE at package init, combined with a
// package-level atomic call counter): places are minted through the real EntityCreated path, whose
// reuse-before-create floor (fn_apply_entity_created, §5.4) matches on descriptor — a fixed, repeated
// descriptor would make every SECOND call in a test run (and every call in a SECOND `go test`
// invocation, since a package-level counter alone resets to zero each process start — the acceptance
// battery runs the Go suite twice with no reset between, i.e. two separate processes against the SAME
// un-reset database) silently REUSE an EARLIER call's place instead of minting its own, which is not
// what "a stand-in for a live seat authoring a NEW place" should simulate. The run seed is the CI
// stand-in's own analogue of a live model never authoring the identical phrase twice; it is not itself
// asserted on by any test (only fakePlaceAuthorDescriptorPrefix is). Matching place_author.v1.schema.json
// exactly: descriptor + kind + extent_class ONLY, no coordinate, no radius, no number describing
// geometry — the schema is the leash keeping geometry out of the model's hands even in a CI fake.
// Wired into every OTHER package test's Orchestrator too (the PlaceAuthor field is left nil there —
// nothing outside placeauthor_test.go's own place-creation path ever calls PlaceAuthor.Generate).
// Errors when req.Schema == nil (the structured-output floor every other fake enforces).
// FAKE: CI stand-in for an undelivered live model. The DESIGN has no LLM-free path (POST-COMPACTION-RULINGS); this fake is scaffolding, not a design statement.
type fakePlaceAuthorDriver struct{ name string }

// fakePlaceAuthorDescriptorPrefix is the stable, assertable prefix every fakePlaceAuthorDriver
// descriptor starts with — tests match on this rather than a full fixed string, since the run
// seed/counter suffix is deliberately non-constant (see the type's own docstring).
const fakePlaceAuthorDescriptorPrefix = "a huddle of driftwood shacks along the tideline"

var (
	fakePlaceAuthorCounter atomic.Int64
	fakePlaceAuthorRunSeed = time.Now().UnixNano()
)

func NewFakePlaceAuthorDriver() Driver { return &fakePlaceAuthorDriver{name: "fake-place-author"} }

func (f *fakePlaceAuthorDriver) Name() string                { return f.name }
func (f *fakePlaceAuthorDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (f *fakePlaceAuthorDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: place-author driver used without a schema", f.name)
	}
	n := fakePlaceAuthorCounter.Add(1)
	return fmt.Sprintf(
		`{"descriptor":%s,"kind":"waystation","extent_class":"small"}`,
		jsonStr(fmt.Sprintf("%s #%d-%d", fakePlaceAuthorDescriptorPrefix, fakePlaceAuthorRunSeed, n)),
	), nil
}

// fakeIntentDriver — the DEV stand-in for the DECOMPOSE seat, i.e. the one DREAMCHAT_BRIDGE=fake
// binds when a human is driving the server by hand or a frontend is integrating against it.
//
// Why it exists. fakeStructuredDriver answers from a scripted table, and the factory built it with a
// NIL table (bridge.go), so every player input decomposed to "[]" — the empty chain. The server came
// up, streamed a correct frame sequence, and committed NOTHING, ever: no canon event, no ruling, no
// world turn with anything to turn on. Hand-driving three beats against the seeded Drowned Lantern
// produced `committed: []` every time while canon_event stayed at its seed rows. That is a testbed
// that can only prove the pipe is connected, never that the world moves — and the scripted-table
// fake could not see it, because every test supplies its own table.
//
// What it is. A deterministic PARSER, not a model: it reads the candidate whitelist and the player's
// raw words back out of the assembled decompose prompt and binds real ids from that list. It is
// still a leash — it can only ever emit four of the closed vocabulary's shapes (ActorMoved,
// Communicated, QUERY, UNRESOLVED), never out-of-vocab, and unrecognised prose still yields "[]"
// (commits nothing, C-5) exactly as before.
//
// The rule is FIRST WORD = verb, deliberately: a dev stand-in that is guessable beats one that is
// clever, because the person hand-driving needs to predict what their sentence will do.
//
//	movement (go/walk/head/…)   -> ActorMoved, bound to the named location or portal artifact
//	speech   (say/tell/ask/…)   -> Communicated, bound to the named present actor
//	query    (look/read/who/…)  -> QUERY, bound to any named candidate
//
// A reference that ties between two DISTINCT candidates emits UNRESOLVED rather than picking one —
// the same refusal-to-guess the live seat owes, and the only way the frontend's unresolved-candidate
// surface can be exercised without keys.
//
// NOT built, with the reason: ObjectRelocated (give/take/drop). Its destination is an actor — for
// "take the crate" that actor is the VIEWER, whose id is not in the candidate block and cannot be
// recovered from the prompt. Binding dest_id to a guess would make the carry path silently wrong,
// which is worse than absent. It needs the viewer id at the seat boundary; that is its own change.
//
// FAKE: a dev/CI stand-in for a live model. The DESIGN has no LLM-free path
// (POST-COMPACTION-RULINGS); this is scaffolding for driving the loop without keys, not a statement
// that decompose can be done by string matching.
type fakeIntentDriver struct{ name string }

func NewFakeIntentDriver() Driver { return &fakeIntentDriver{name: "fake-intent"} }

func (f *fakeIntentDriver) Name() string                { return f.name }
func (f *fakeIntentDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }

func (f *fakeIntentDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: intent driver used without a schema", f.name)
	}
	chain := devIntentChain(parseDecomposePlayerInput(req.Prompt), parseDecomposeCandidates(req.Prompt))
	out, err := json.Marshal(chain)
	if err != nil {
		return "", fmt.Errorf("%s: marshal chain: %w", f.name, err)
	}
	return string(out), nil
}

// devIntentVerbs maps a first word to the shape it produces. Kept as data so the set is readable in
// one place and so a reader can answer "what will my sentence do" without tracing code.
var devIntentVerbs = map[string]string{
	"go": "move", "walk": "move", "head": "move", "move": "move", "travel": "move",
	"enter": "move", "step": "move", "climb": "move", "leave": "move", "return": "move", "run": "move",

	"say": "speak", "tell": "speak", "ask": "speak", "talk": "speak", "speak": "speak",
	"greet": "speak", "warn": "speak", "shout": "speak", "whisper": "speak", "reply": "speak",

	"look": "query", "examine": "query", "inspect": "query", "study": "query", "read": "query",
	"watch": "query", "who": "query", "what": "query", "where": "query",
}

// devIntentChain is the whole decision. It returns an EMPTY (never nil) chain whenever it cannot
// bind honestly — no verb it knows, or no candidate the words name — so the caller marshals "[]".
func devIntentChain(input string, cands []Candidate) []Attempt {
	chain := []Attempt{}
	if input == "" || len(cands) == 0 {
		return chain
	}
	lower := strings.ToLower(input)
	first, _, _ := strings.Cut(strings.TrimLeft(lower, `"' `), " ")
	shape := devIntentVerbs[strings.Trim(first, ",.!?")]
	if shape == "" {
		return chain
	}

	var want []string
	switch shape {
	case "move":
		want = []string{"location", "artifact"} // a room, or the portal object that reaches one
	case "speak":
		want = []string{"actor"}
	}
	matched := matchDevCandidates(lower, cands, want...)
	switch {
	case len(matched) == 0:
		return chain
	case len(matched) > 1:
		ids := make([]string, 0, len(matched))
		for _, c := range matched {
			ids = append(ids, c.ID)
		}
		return append(chain, Attempt{
			Type: "UNRESOLVED", Stated: input, Reference: matched[0].Name, CandidateIDs: ids,
		})
	}

	target := matched[0]
	switch shape {
	case "move":
		return append(chain, Attempt{Type: "ActorMoved", Stated: input, ToTargetID: target.ID})
	case "speak":
		return append(chain, Attempt{
			Type: "Communicated", Stated: input, ListenerID: target.ID,
			Content: devSpokenContent(input), DurationClass: "short",
		})
	default:
		return append(chain, Attempt{Type: "QUERY", Stated: input, QueryTargetIDs: []string{target.ID}})
	}
}

// devSpokenContent extracts what was actually said: the quoted words if the player quoted them, else
// whatever follows "about", else the whole line. Never empty — Communicated.content has minLength 1.
func devSpokenContent(input string) string {
	if open := strings.Index(input, `"`); open >= 0 {
		if close := strings.Index(input[open+1:], `"`); close > 0 {
			if said := strings.TrimSpace(input[open+1 : open+1+close]); said != "" {
				return said
			}
		}
	}
	if _, after, ok := strings.Cut(strings.ToLower(input), " about "); ok {
		if said := strings.TrimSpace(after); said != "" {
			return said
		}
	}
	return input
}

// matchDevCandidates finds the candidates the player's words NAME: of every candidate whose name
// appears in the input, the ones sharing the LONGEST name win, because the longest match is the most
// specific reference ("the back door" over "the door"). More than one survivor means the words
// genuinely do not distinguish two entities — the caller turns that into UNRESOLVED instead of
// guessing. kinds filters the candidate block; empty kinds means any kind.
func matchDevCandidates(lowerInput string, cands []Candidate, kinds ...string) []Candidate {
	var best []Candidate
	bestLen := 0
	for _, c := range cands {
		if len(kinds) > 0 && !slices.Contains(kinds, c.Kind) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(c.Name))
		// Two characters is the floor: a one-letter label would match almost any sentence.
		if len(name) < 2 || !strings.Contains(lowerInput, name) {
			continue
		}
		switch {
		case len(name) > bestLen:
			best, bestLen = []Candidate{c}, len(name)
		case len(name) == bestLen && c.ID != best[0].ID:
			best = append(best, c)
		}
	}
	return best
}

// parseDecomposeCandidates reads the candidate whitelist back out of an assembled decompose prompt.
// Line shape is fixed by buildDecomposePrompt: "{id}  {name}  ({kind})". The kind is taken from the
// LAST "  (" so a name containing brackets cannot shift the split. Anything unparseable is skipped
// rather than guessed at — a malformed line must not become a bound id.
func parseDecomposeCandidates(prompt string) []Candidate {
	_, rest, ok := strings.Cut(prompt, decomposeCandidatesMarker)
	if !ok {
		return nil
	}
	block, _, _ := strings.Cut(rest, decomposePlayerInputMarker)
	var out []Candidate
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		id, tail, ok := strings.Cut(line, "  ")
		if !ok || id == "" {
			continue
		}
		open := strings.LastIndex(tail, "  (")
		if open <= 0 || !strings.HasSuffix(tail, ")") {
			continue
		}
		name, kind := tail[:open], tail[open+3:len(tail)-1]
		if name == "" || kind == "" {
			continue
		}
		out = append(out, Candidate{ID: id, Name: name, Kind: kind})
	}
	return out
}

// parseDecomposePlayerInput returns the raw words the player typed — the prompt's mutable tail.
func parseDecomposePlayerInput(prompt string) string {
	_, tail, ok := strings.Cut(prompt, decomposePlayerInputMarker)
	if !ok {
		return ""
	}
	return strings.TrimSpace(tail)
}
