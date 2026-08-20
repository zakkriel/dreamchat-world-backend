# Kickstart: Arrival Choice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the world-genesis build into author → user choice (character, then starting scenario) → commit, per the approved spec `docs/superpowers/specs/2026-08-20-kickstart-arrival-choice-design.md`.

**Architecture:** Phase 1 (`POST /worlds/genesis`) authors the doc and streams narration, ending in a `choice` frame carrying a draft handle instead of a committed world. Phase 2 (`POST /worlds/genesis/kickstart`, plain JSON turns) resolves the character then the scenario via a new `world_kickstart` seat; the final answer runs the whole existing commit ladder in one transaction. Drafts live in a new in-memory TTL store; the genesis doc never crosses the wire.

**Tech Stack:** Go (core/api), Postgres via pgx, JSON-Schema seat leashes, SSE over fetch, React/TanStack (dream-weaver-visuals), json2ts contract codegen.

## Global Constraints

- Seat schemas: `additionalProperties: false`, **no numeric field anywhere** (PRD AC-7).
- The genesis doc (secrets, history, knowledge) never appears in any response body (spec AC-7).
- One transaction: the world commits whole at the final kickstart answer or not at all (PRD AC-2). Caller owns Begin/Commit/Rollback; `commitWorldGenesis` signature unchanged.
- Brief wins: identity stated in brief/answers ⇒ no character question (spec decision 2).
- Both lanes get the kickstart; every question carries at most one `recommended` option (spec decision 3).
- Draft TTL: 15 minutes; expiry = stated refusal, no debris (spec AC-6).
- No archetype vocabulary in the kickstart prompt or schema; candidates come from this world's content only (GA-2/GA-3).
- New prompt files must contain **no seed proper names** (`promptnames_test.go` scans `prompts/*.txt`).
- Frontend copy must not match `/\b(view as|switch (?:user|character|viewer)|Billing|Subscription)\b/i` nor Glossary-banned words (Entities, Canon, Epistemic, Inventory, …); no `.toUpperCase()/.toLowerCase()` in JSX interpolations; payload arrays render unsorted/unfiltered; no invented numbers (laws.test.ts).
- Contract discipline: a schema version moves ⇒ vendored contract + generated type + PIN move in the same commit; `world_genesis_frame/1` → `world_genesis_frame/2`.
- Cost: kickstart seat calls bill into the per-request cost sink; the commit request logs one aggregate `world genesis timing:` line for the whole build (spec AC-10). Ceiling env `DREAMCHAT_GENESIS_COST_WARN_USD` unchanged.
- Line numbers cited below were read on 2026-08-20; verify against the file before editing.

---

### Task 1: Draft store

**Files:**
- Create: `dreamchat-world-backend/core/api/genesisdrafts.go`
- Test: `dreamchat-world-backend/core/api/genesisdrafts_test.go`

**Interfaces:**
- Consumes: `genesisDoc` (worldgenesis.go:73-90), `genesisArrival` (worldgenesis.go:148-154).
- Produces (later tasks rely on these exact names):
  - `type genesisDraft struct { doc *genesisDoc; brief, artStyle string; identity *kickstartIdentity; scenarios []kickstartScenario; tally draftTally; deadline time.Time }`
  - The two kickstart value types are declared **in this task** (Task 3 consumes them, not vice versa), so Task 1 compiles alone:
    - `type kickstartIdentity struct { Descriptor string \`json:"descriptor"\`; CanonicalName string \`json:"canonical_name"\` }`
    - `type kickstartScenario struct { Label string \`json:"label"\`; Place string \`json:"place"\`; Why string \`json:"why"\`; Stated string \`json:"stated"\`; Recommended bool \`json:"recommended,omitempty"\` }`
  - `type draftTally struct { usd float64; tokIn, tokOut, cached int64; calls int }` with method `func (t *draftTally) add(usd float64, in, out, cached int64, calls int)`.
  - `func newDraftStore(ttl time.Duration) *draftStore`
  - `func (s *draftStore) mint() string` — returns a random 36-char UUID-shaped handle (use `crypto/rand` + `fmt.Sprintf`, or `github.com/google/uuid` if already in go.mod — check `go.mod` first; pgx pulls it in transitively but only import it if it is a direct dependency already, otherwise hand-roll from crypto/rand).
  - `func (s *draftStore) put(handle string, d *genesisDraft)` — stores with `deadline = time.Now().Add(s.ttl)`.
  - `func (s *draftStore) claim(handle string) (*genesisDraft, bool)` — removes and returns the draft iff present and unexpired; expired entries are deleted on access (lazy sweep). Claim semantics make concurrent duplicate requests on one handle safe: the second claim misses.
- `var errDraftExpired = errors.New("that build has expired — write the brief again and rebuild")` — the stated refusal for AC-6.

- [ ] **Step 1: Write the failing test**

```go
package main

import (
	"testing"
	"time"
)

func TestDraftStoreClaimRoundTrip(t *testing.T) {
	s := newDraftStore(time.Minute)
	h := s.mint()
	if len(h) != 36 {
		t.Fatalf("handle length = %d, want 36", len(h))
	}
	d := &genesisDraft{brief: "a harbour town"}
	s.put(h, d)
	got, ok := s.claim(h)
	if !ok || got.brief != "a harbour town" {
		t.Fatalf("claim = %v, %v", got, ok)
	}
	if _, ok := s.claim(h); ok {
		t.Fatal("second claim succeeded; claim must remove")
	}
}

func TestDraftStoreExpiry(t *testing.T) {
	s := newDraftStore(10 * time.Millisecond)
	h := s.mint()
	s.put(h, &genesisDraft{})
	time.Sleep(20 * time.Millisecond)
	if _, ok := s.claim(h); ok {
		t.Fatal("claimed an expired draft")
	}
}

func TestDraftStoreUnknownHandle(t *testing.T) {
	s := newDraftStore(time.Minute)
	if _, ok := s.claim("00000000-0000-0000-0000-000000000000"); ok {
		t.Fatal("claimed a handle that was never put")
	}
}

func TestDraftTallyAccumulates(t *testing.T) {
	var tl draftTally
	tl.add(0.01, 100, 50, 10, 1)
	tl.add(0.02, 200, 60, 0, 2)
	if tl.usd != 0.03 || tl.tokIn != 300 || tl.tokOut != 110 || tl.cached != 10 || tl.calls != 3 {
		t.Fatalf("tally = %+v", tl)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestDraft' -v`
Expected: FAIL — `undefined: newDraftStore` (compile error).

- [ ] **Step 3: Write the implementation**

```go
package main

// genesisdrafts.go — the between-phases home of an authored-but-uncommitted world.
//
// A build now pauses for the user's kickstart answers, so the genesis document must live
// somewhere between authoring and commit. It lives HERE, in memory, and nowhere else: the
// doc holds every secret and every knowledge path, so it never crosses the wire (spec AC-7),
// and it is not worth a table because an abandoned build is simply lost — the posture the
// PRD already takes for watched builds. Claim semantics (remove-on-read) make concurrent
// requests on one handle race-free without holding a lock across a seat call or a commit.

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"
)

const genesisDraftTTL = 15 * time.Minute

var errDraftExpired = errors.New("that build has expired — write the brief again and rebuild")

type kickstartIdentity struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
}

type kickstartScenario struct {
	Label       string `json:"label"`
	Place       string `json:"place"`
	Why         string `json:"why"`
	Stated      string `json:"stated"`
	Recommended bool   `json:"recommended,omitempty"`
}

type draftTally struct {
	usd            float64
	tokIn, tokOut  int64
	cached         int64
	calls          int
}

func (t *draftTally) add(usd float64, in, out, cached int64, calls int) {
	t.usd += usd
	t.tokIn += in
	t.tokOut += out
	t.cached += cached
	t.calls += calls
}

type genesisDraft struct {
	doc       *genesisDoc
	brief     string
	artStyle  string
	identity  *kickstartIdentity  // nil until the character question is answered
	scenarios []kickstartScenario // authored options awaiting the scenario answer
	tally     draftTally
	deadline  time.Time
}

type draftStore struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]*genesisDraft
}

func newDraftStore(ttl time.Duration) *draftStore {
	return &draftStore{ttl: ttl, m: make(map[string]*genesisDraft)}
}

func (s *draftStore) mint() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // crypto/rand failing means the process has no entropy; nothing sensible continues
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *draftStore) put(handle string, d *genesisDraft) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d.deadline = time.Now().Add(s.ttl)
	s.m[handle] = d
}

func (s *draftStore) claim(handle string) (*genesisDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for h, d := range s.m { // lazy sweep: expired drafts leave on any access
		if now.After(d.deadline) {
			delete(s.m, h)
		}
	}
	d, ok := s.m[handle]
	if !ok {
		return nil, false
	}
	delete(s.m, handle)
	return d, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestDraft' -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git -C dreamchat-world-backend add core/api/genesisdrafts.go core/api/genesisdrafts_test.go
git -C dreamchat-world-backend commit -m "feat: in-memory TTL draft store for paused genesis builds"
```

---

### Task 2: `arrival_candidates` in the genesis document

**Files:**
- Modify: `dreamchat-world-backend/core/api/schema/world_genesis.v1.schema.json` (arrival object at lines 271-303; top-level `properties`)
- Modify: `dreamchat-world-backend/core/api/worldgenesis.go` (genesisDoc 73-90, validate() 255-464)
- Modify: `dreamchat-world-backend/core/api/bridge_fakes.go` (fakeWorldGenesisDriver ~648)
- Modify: `dreamchat-world-backend/core/api/prompts/world_genesis.txt`
- Test: `dreamchat-world-backend/core/api/worldgenesis_test.go` (extend)

**Interfaces:**
- Produces: `genesisDoc.ArrivalCandidates []genesisCandidate` with
  `type genesisCandidate struct { Descriptor string \`json:"descriptor"\`; CanonicalName string \`json:"canonical_name"\`; Why string \`json:"why"\` }`.
- Contract: candidates are optional (identity-stated briefs emit none). When present: exactly 3, all fields non-blank, canonical names pairwise distinct, and **exactly one** candidate's `canonical_name` equals `arrival.canonical_name` — that match is the recommended default; `arrival` stays required and is that candidate's full premise.
- Fake rule (deterministic, tests depend on it): the fake genesis driver emits 3 candidates **unless** the brief contains the substring `"i am "` (case-insensitive), in which case it emits none.

- [ ] **Step 1: Write the failing validator tests**

Add to `worldgenesis_test.go` (follow the file's existing table/fixture style — read its first working `validate()` test first and reuse its minimal-valid-doc helper if one exists; if not, build the doc by unmarshalling the fake driver's output):

```go
func TestValidateArrivalCandidates(t *testing.T) {
	doc := validGenesisDocForTest(t) // reuse/extract the minimal passing doc the existing tests build
	cand := func(name string) genesisCandidate {
		return genesisCandidate{Descriptor: "a stranger in a wet coat", CanonicalName: name, Why: "owed money"}
	}

	// exactly 3
	doc.ArrivalCandidates = []genesisCandidate{cand(doc.Arrival.CanonicalName), cand("Second Name")}
	if err := doc.validate(); err == nil {
		t.Fatal("2 candidates accepted; want refusal (exactly 3)")
	}

	// distinct names
	doc.ArrivalCandidates = []genesisCandidate{cand(doc.Arrival.CanonicalName), cand("Second Name"), cand("Second Name")}
	if err := doc.validate(); err == nil {
		t.Fatal("duplicate candidate names accepted")
	}

	// exactly one must match arrival.canonical_name
	doc.ArrivalCandidates = []genesisCandidate{cand("First Name"), cand("Second Name"), cand("Third Name")}
	if err := doc.validate(); err == nil {
		t.Fatal("no candidate matches the arrival; want refusal")
	}

	// the happy path
	doc.ArrivalCandidates = []genesisCandidate{cand(doc.Arrival.CanonicalName), cand("Second Name"), cand("Third Name")}
	if err := doc.validate(); err != nil {
		t.Fatalf("valid candidates refused: %v", err)
	}

	// refusals must be genesisRefusal, not faults
	doc.ArrivalCandidates = doc.ArrivalCandidates[:2]
	if err := doc.validate(); !IsGenesisRefusal(err) {
		t.Fatalf("candidate violation is not a refusal: %v", err)
	}
}

func TestFakeGenesisEmitsCandidatesUnlessIdentityStated(t *testing.T) {
	seat := NewFakeWorldGenesisDriver() // match the fake's real constructor name in bridge_fakes.go
	open, err := authorWorld(context.Background(), seat, "a harbour town at closing time", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(open.ArrivalCandidates) != 3 {
		t.Fatalf("identity-open brief: %d candidates, want 3", len(open.ArrivalCandidates))
	}
	stated, err := authorWorld(context.Background(), seat, "a harbour town; I am the debt collector", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stated.ArrivalCandidates) != 0 {
		t.Fatalf("identity-stated brief: %d candidates, want 0", len(stated.ArrivalCandidates))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestValidateArrivalCandidates|TestFakeGenesisEmitsCandidates' -v`
Expected: FAIL — `undefined: genesisCandidate`.

- [ ] **Step 3: Schema change**

In `world_genesis.v1.schema.json`, add beside `arrival` (NOT inside it):

```json
"arrival_candidates": {
  "type": "array",
  "minItems": 3,
  "maxItems": 3,
  "description": "Present ONLY when the brief and answers leave the player's identity open — if the user said who they are, their words outrank you and this array is omitted entirely. Three ways the player could enter THIS world, each drawn from the authored cast, places and history — never from a stock of archetypes. Exactly one candidate's canonical_name must equal arrival.canonical_name: that one is the recommended default, and arrival is its full premise.",
  "items": {
    "type": "object",
    "required": ["descriptor", "canonical_name", "why"],
    "additionalProperties": false,
    "properties": {
      "descriptor": { "type": "string", "minLength": 1, "description": "How the room sees this stranger at first sight." },
      "canonical_name": { "type": "string", "minLength": 1, "description": "This candidate's name. Nobody in the world holds a perception of it." },
      "why": { "type": "string", "minLength": 1, "description": "Why they are here, as premise — a situation, never an interior state. It must connect to this world's authored content." }
    }
  }
}
```

Do NOT add it to the top-level `required` list.

- [ ] **Step 4: Go struct + validator**

In `worldgenesis.go`, add after `genesisArrival` (line ~154):

```go
type genesisCandidate struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
	Why           string `json:"why"`
}
```

Add `ArrivalCandidates []genesisCandidate \`json:"arrival_candidates,omitempty"\`` to `genesisDoc`. In `validate()`, after the existing arrival checks (i.e. after line ~383's someone-there check), add:

```go
if len(d.ArrivalCandidates) > 0 {
	if len(d.ArrivalCandidates) != 3 {
		return refuse("arrival candidates must number exactly three, got %d", len(d.ArrivalCandidates))
	}
	seen := map[string]bool{}
	matches := 0
	for _, c := range d.ArrivalCandidates {
		name := strings.TrimSpace(c.CanonicalName)
		if name == "" || strings.TrimSpace(c.Descriptor) == "" || strings.TrimSpace(c.Why) == "" {
			return refuse("an arrival candidate is missing its name, descriptor or why")
		}
		if seen[name] {
			return refuse("two arrival candidates share the name %q", name)
		}
		seen[name] = true
		if name == strings.TrimSpace(d.Arrival.CanonicalName) {
			matches++
		}
	}
	if matches != 1 {
		return refuse("exactly one arrival candidate must be the arrival itself (the recommended default); %d match", matches)
	}
}
```

(Note: the validator cannot know whether the brief "stated" an identity — that judgment is the seat's, leashed by the prompt. The mechanical checks above are the belt.)

- [ ] **Step 5: Prompt + fake**

Append to `prompts/world_genesis.txt` (keep the file's second-person voice; no proper names):

```
WHO THE VISITOR IS MAY BE AN OPEN QUESTION. If the brief or the answers say who the visitor is — in any words — that identity is theirs, you author `arrival` from it, and you omit `arrival_candidates` entirely. If they left it open, you offer them the choice: emit exactly three `arrival_candidates`, three different strangers this particular world could receive, each one implied by the cast, the places or the history you just authored — a stranger this world was already about to meet. Never a stock figure that would fit any world. Exactly one of the three is the arrival you authored (same canonical_name): that one is your recommendation.
```

In `bridge_fakes.go`, extend the fake genesis driver's emitted JSON: when `strings.Contains(strings.ToLower(brief), "i am ")` emit no `arrival_candidates`; otherwise emit 3 candidates where candidate 1 reuses the doc's arrival `descriptor`/`canonical_name` with a why, and candidates 2-3 use invented names derived the same way the fake derives its other names (mirror its existing naming scheme — read the fake before editing). The fake's doc must still pass `validate()`.

- [ ] **Step 6: Run the tests, then the package**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestValidateArrivalCandidates|TestFakeGenesisEmitsCandidates' -v && go test ./core/api/ -run 'TestWorldGenesis' -v`
Expected: new tests PASS; existing genesis tests still PASS (the field is optional and the fake stays valid). Do not run the full suite here.

- [ ] **Step 7: Commit**

```bash
git -C dreamchat-world-backend add core/api/schema/world_genesis.v1.schema.json core/api/worldgenesis.go core/api/bridge_fakes.go core/api/prompts/world_genesis.txt core/api/worldgenesis_test.go
git -C dreamchat-world-backend commit -m "feat: genesis doc offers three grounded arrival candidates when identity is open"
```

---

### Task 3: The `world_kickstart` seat

**Files:**
- Create: `dreamchat-world-backend/core/api/worldkickstart.go`
- Create: `dreamchat-world-backend/core/api/prompts/world_kickstart.txt`
- Create: `dreamchat-world-backend/core/api/schema/world_kickstart.v1.schema.json`
- Modify: `dreamchat-world-backend/core/api/bridge.go` (Seat var block ~93-125; DefaultDriverFactory switch tail ~305-325)
- Modify: `dreamchat-world-backend/core/api/seatconfig.go` (allSeatNames 70-75)
- Modify: `dreamchat-world-backend/core/api/main.go` (fake seat map ~200-212; NewBridge call ~125-127)
- Modify: `dreamchat-world-backend/core/api/bridge_fakes.go` (new fake driver)
- Test: `dreamchat-world-backend/core/api/worldkickstart_test.go`

**Interfaces:**
- Consumes: `kickstartIdentity`, `kickstartScenario` (Task 1), `genesisDoc` (Task 2), `Driver`/`GenRequest` (bridge.go), `refuse`/`IsGenesisRefusal` (worldgenesis.go:56-69).
- Produces:
  - `var SeatWorldKickstart = Seat{Name: "world_kickstart", Requires: []Capability{CapStructuredOutput}}`
  - `type kickstartDoc struct { Identity kickstartIdentity \`json:"identity"\`; Scenarios []kickstartScenario \`json:"scenarios"\` }`
  - `func authorKickstart(ctx context.Context, seat Driver, doc *genesisDoc, brief, who, customScenario string) (*kickstartDoc, error)` — `who` is the chosen candidate's canonical name or the user's own free text; `customScenario` non-empty means "ground exactly this user-written opening as the single scenario" (then `Scenarios` has exactly 1 item), empty means "offer three" (exactly 3, exactly one `recommended:true`).
  - `func (k *kickstartDoc) validate(doc *genesisDoc, wantOptions bool) error` — belt: scenario count (3 vs 1); every field non-blank; every `Place` is a place in `doc.Places` **with at least one cast member whose `starts_in` equals it** (spec AC-5, same predicate as worldgenesis.go:374-383); when `wantOptions`, exactly one `Recommended`; identity fields non-blank.
  - Prompt marker constants (the fake parses these; shared constants prevent drift, same trick as worldgenesis.go:48-51):

```go
const (
	worldKickstartWorldMarker    = "THE WORLD (already authored and immutable — every place and person below exists):"
	worldKickstartBriefMarker    = "BRIEF (the user's own words):"
	worldKickstartWhoMarker      = "WHO THE PLAYER IS (the user's choice — it outranks your judgement completely):"
	worldKickstartOpeningMarker  = "THE USER'S OWN OPENING (ground exactly this, as the single scenario):"
)
```

- [ ] **Step 1: Write the failing tests**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestAuthorKickstart|TestKickstartValidate' -v`
Expected: FAIL — `undefined: authorKickstart`.

- [ ] **Step 3: Schema file** — `schema/world_kickstart.v1.schema.json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "world_kickstart/1",
  "title": "world_kickstart/1 — three ways this player's story could start, or the user's own opening grounded. The seat emits fiction only: no id, no coordinate, no tick, no number of any kind, anywhere.",
  "type": "object",
  "required": ["identity", "scenarios"],
  "additionalProperties": false,
  "properties": {
    "identity": {
      "type": "object",
      "required": ["descriptor", "canonical_name"],
      "additionalProperties": false,
      "description": "The player as chosen or written by the user, echoed as premise. When the user wrote their own words, author descriptor and name FROM those words — their words outrank yours. No traits, no disposition, no interior (B-4).",
      "properties": {
        "descriptor": { "type": "string", "minLength": 1, "description": "How the room sees this stranger at first sight." },
        "canonical_name": { "type": "string", "minLength": 1 }
      }
    },
    "scenarios": {
      "type": "array",
      "minItems": 1,
      "maxItems": 3,
      "description": "Exactly three when offering; exactly one when grounding the user's own opening. Each names a real, already-populated place from the authored world.",
      "items": {
        "type": "object",
        "required": ["label", "place", "why", "stated"],
        "additionalProperties": false,
        "properties": {
          "label": { "type": "string", "minLength": 1, "description": "The opening in a short line, the way a chapter is named — this renders as the option the user picks." },
          "place": { "type": "string", "minLength": 1, "description": "canonical_name of the place they walk into. It must be a place the world authored, and at least one member of the cast must already start there — a visitor never enters an empty room." },
          "why": { "type": "string", "minLength": 1, "description": "Why they are here, as premise — a situation, never a feeling." },
          "stated": { "type": "string", "minLength": 1, "description": "The arrival in the player's own first person. This becomes their single perception and the only thing they know." },
          "recommended": { "type": "boolean", "description": "At most one scenario carries this; the surface highlights it as the default. Omit entirely when grounding the user's own opening." }
        }
      }
    }
  }
}
```

- [ ] **Step 4: Prompt file** — `prompts/world_kickstart.txt` (no proper names anywhere; genre-agnostic; the GA-2/GA-3 sentence is load-bearing):

```
You are a world that has already been authored, deciding how its first visitor's story starts. The world below is immutable — every place, person, object and piece of history in it already exists and none of it can change. Your whole job is to offer the way in.
You are told who the player is. When it is a name from the offered candidates, that person's premise is settled — echo it as `identity`. When it is the user's own words, those words outrank your judgement completely: author `identity` from them and nothing else. Either way the player is a PREMISE, NOT A MIND: a descriptor, a name, and a situation. Never a trait, never a feeling, never an intention.
Offer three scenarios — three different doors into the same world, each one implied by what was already authored: a place that exists, people who already start there, a situation the history already set in motion. Mark exactly one as recommended. A scenario that would fit any world is wrong; a scenario this world was already about to have happen is right. Never offer a stock opening — no figure or entrance that belongs to a kind of story rather than to this one.
When THE USER'S OWN OPENING is supplied, do not offer alternatives: ground exactly that, as one single scenario, in this world's real places and people — their words outrank yours, your job is only to make them true here. Emit exactly one scenario and no recommended flag.
Every scenario's place must be a place the world authored, and someone from the cast must already start there. The visitor knows nothing and nobody knows the visitor: `stated` is one first-person sentence of arrival, and it is the only thing the player will know.
You emit no id, no coordinate, no tick, no count — no number of any kind, anywhere, in any field.
```

- [ ] **Step 5: Go implementation** — `worldkickstart.go`:

```go
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
```

NOTE: `genesisPlace`'s canonical-name field — confirm the field name in worldgenesis.go (~92-112) and use it; the snippet assumes `CanonicalName`. Same for `genesisActor.StartsIn` (confirmed at 114-123).

- [ ] **Step 6: Seat registration (five touch points) + fake**

1. `bridge.go` var block: `SeatWorldKickstart = Seat{Name: "world_kickstart", Requires: []Capability{CapStructuredOutput}}` beside `SeatWorldInterview`.
2. `bridge.go` `DefaultDriverFactory` tail: `case "fake-world-kickstart": return NewFakeWorldKickstartDriver(), nil`.
3. `seatconfig.go` `allSeatNames`: append `SeatWorldKickstart.Name`.
4. `main.go` fake seat map: `"world_kickstart": {Provider: "fake-world-kickstart", Model: "dev"},`.
5. `main.go` `NewBridge(...)`: append `SeatWorldKickstart`.

Fake driver in `bridge_fakes.go` (pattern: fakeWorldGenesisDriver ~648 — errors without `req.Schema`, parses its inputs back out of the prompt via the shared markers):

```go
type fakeWorldKickstartDriver struct{}

func NewFakeWorldKickstartDriver() Driver { return &fakeWorldKickstartDriver{} }

func (f *fakeWorldKickstartDriver) Name() string { return "fake-world-kickstart" }

func (f *fakeWorldKickstartDriver) Capabilities() map[Capability]bool {
	return map[Capability]bool{CapStructuredOutput: true}
}

func (f *fakeWorldKickstartDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("fake world kickstart requires a schema, got none")
	}
	// Re-extract the authored world from the prompt: the fake grounds itself in the real doc,
	// so prompt and fake cannot drift and the belt checks pass against whatever the genesis fake made.
	world := sectionAfter(req.Prompt, worldKickstartWorldMarker, worldKickstartBriefMarker)
	var doc genesisDoc
	if err := json.Unmarshal([]byte(strings.TrimSpace(world)), &doc); err != nil {
		return "", fmt.Errorf("fake world kickstart could not parse the world block: %w", err)
	}
	who := strings.TrimSpace(sectionAfter(req.Prompt, worldKickstartWhoMarker, worldKickstartOpeningMarker))
	opening := strings.TrimSpace(sectionAfter(req.Prompt, worldKickstartOpeningMarker, ""))
	place := strings.TrimSpace(doc.Arrival.Place) // populated by construction (worldgenesis.go:374-383)
	name := who
	if len(name) > 40 || strings.ContainsAny(name, ".,;") { // free text, not a candidate name: derive a short name
		name = "The One Who Wrote In"
	}
	ident := fmt.Sprintf(`{"descriptor":"a stranger the room was not expecting","canonical_name":%q}`, name)
	if opening != "" {
		return fmt.Sprintf(`{"identity":%s,"scenarios":[{"label":"as you wrote it","place":%q,"why":"exactly what the user asked for","stated":%q}]}`,
			ident, place, "I arrived the way I said I would."), nil
	}
	return fmt.Sprintf(`{"identity":%s,"scenarios":[`+
		`{"label":"through the front door","place":%q,"why":"expected by nobody","stated":"I stepped in off the street.","recommended":true},`+
		`{"label":"in the middle of it","place":%q,"why":"summoned by a note","stated":"I walked into an argument already going."},`+
		`{"label":"the wrong address","place":%q,"why":"looking for somewhere else","stated":"I came in out of the rain."}]}`,
		ident, place, place, place), nil
}
```

`sectionAfter(s, from, until string) string` — small helper (add near the fake): returns the substring after `from` up to `until` (or end when `until` is `""` or absent). Write it if no equivalent exists in bridge_fakes.go; check first.

- [ ] **Step 7: Run the tests**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestAuthorKickstart|TestKickstartValidate' -v && go test ./core/api/ -run 'TestSeat|TestBridge|TestPromptNames' -v`
Expected: new tests PASS; seat/bridge/prompt-name suites still PASS (the prompt has no proper names; the seat is fully registered).

- [ ] **Step 8: Commit**

```bash
git -C dreamchat-world-backend add core/api/worldkickstart.go core/api/worldkickstart_test.go core/api/prompts/world_kickstart.txt core/api/schema/world_kickstart.v1.schema.json core/api/bridge.go core/api/bridge_fakes.go core/api/seatconfig.go core/api/main.go
git -C dreamchat-world-backend commit -m "feat: world_kickstart seat — scenarios offered or the user's own opening grounded"
```

---

### Task 4: Frame contract v2 + kickstart turn contract

**Files:**
- Create: `dreamchat-world-backend/core/api/schema/world_genesis_frame.v2.schema.json`
- Create: `dreamchat-world-backend/core/api/schema/world_kickstart_turn.v1.schema.json`
- Delete: `dreamchat-world-backend/core/api/schema/world_genesis_frame.v1.schema.json` (clean cutover; the frontend vendors v2 in Task 8)
- Modify: `dreamchat-world-backend/core/api/worldgenesishandler.go:41` (version const)

**Interfaces:**
- Produces: `worldGenesisFrameSchemaVersion = "world_genesis_frame/2"`, `worldKickstartTurnSchemaVersion = "world_kickstart_turn/1"`.
- Frame v2 kinds: `working | choice | refused | error`. **The `world` kind is gone** — the world id now arrives on the kickstart turn's `done:true`, because commit no longer happens inside the stream.
- The choice frame carries: `handle` (36-char), `question`, `options[]` (label/implication/recommended — the interview option shape).

- [ ] **Step 1: Write frame v2** — copy `world_genesis_frame.v1.schema.json`, then:
  - `$id`: `"world_genesis_frame/2"`; `schema_version` const likewise.
  - `kind` enum: `["working", "choice", "refused", "error"]`.
  - Remove `id`, `display_name`, `tagline`, `playable` properties and the `world` oneOf branch.
  - Add properties:

```json
"handle": { "type": "string", "minLength": 36, "maxLength": 36, "description": "The draft's opaque handle. POST it to /worlds/genesis/kickstart with each answer. It expires; an expired handle is a stated refusal, and the build is simply made again." },
"question": { "type": "string", "minLength": 1 },
"options": {
  "type": "array",
  "minItems": 1,
  "items": {
    "type": "object",
    "required": ["label"],
    "additionalProperties": false,
    "properties": {
      "label": { "type": "string", "minLength": 1 },
      "implication": { "type": "string", "minLength": 1 },
      "recommended": { "type": "boolean", "description": "At most one option per question carries this." }
    }
  }
}
```

  - `oneOf`: `working|refused|error` ⇒ required `["stated"]`; `choice` ⇒ required `["handle", "question", "options"]`.
  - Title/description: state plainly that the stream now ends in a choice, and the world arrives on the kickstart turn.

- [ ] **Step 2: Write the kickstart turn schema** — `world_kickstart_turn.v1.schema.json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "world_kickstart_turn/1",
  "title": "world_kickstart_turn/1 — the response to POST /worlds/genesis/kickstart. Same grammar as the interview turn: done:false carries the next question; done:true carries the world, built and playable. The free-text answer is a property of the surface and is deliberately not enumerated here.",
  "type": "object",
  "required": ["schema_version", "done"],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "const": "world_kickstart_turn/1" },
    "done": { "type": "boolean" },
    "question": { "type": "string", "minLength": 1 },
    "why": { "type": "string", "minLength": 1 },
    "options": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["label"],
        "additionalProperties": false,
        "properties": {
          "label": { "type": "string", "minLength": 1 },
          "implication": { "type": "string", "minLength": 1 },
          "recommended": { "type": "boolean", "description": "At most one option per turn carries this." }
        }
      }
    },
    "world": {
      "type": "object",
      "required": ["id", "display_name", "tagline", "playable"],
      "additionalProperties": false,
      "properties": {
        "id": { "type": "string", "pattern": "^[0-9a-fA-F-]{36}$" },
        "display_name": { "type": "string", "minLength": 1 },
        "tagline": { "type": "string", "minLength": 1 },
        "playable": { "const": true }
      }
    }
  },
  "oneOf": [
    { "properties": { "done": { "const": false } }, "required": ["question", "options"], "not": { "required": ["world"] } },
    { "properties": { "done": { "const": true } }, "required": ["world"], "not": { "required": ["question"] } }
  ]
}
```

- [ ] **Step 3: Move the consts** — in `worldgenesishandler.go`: change line 41 to `const worldGenesisFrameSchemaVersion = "world_genesis_frame/2"`; add below it `const worldKickstartTurnSchemaVersion = "world_kickstart_turn/1"`. Delete `schema/world_genesis_frame.v1.schema.json`.

- [ ] **Step 4: Verify compile only** — `cd dreamchat-world-backend && go build ./core/api/`. Expected: builds. (Frame-shape tests go red until Task 5 rewires the handler — that is why Tasks 4+5 land as consecutive commits and the full suite is not run between them.)

- [ ] **Step 5: Commit**

```bash
git -C dreamchat-world-backend add -A core/api/schema/ core/api/worldgenesishandler.go
git -C dreamchat-world-backend commit -m "feat: world_genesis_frame/2 (choice frame, world kind removed) + world_kickstart_turn/1"
```

---

### Task 5: Handler split — phase 1 ends in a choice

**Files:**
- Modify: `dreamchat-world-backend/core/api/worldgenesishandler.go` (struct 62-73, ctor 71, `build` 162-258, `genesisNarration` 275-296)
- Modify: `dreamchat-world-backend/core/api/main.go` (ctor callsite in `newRouter`)
- Test: `dreamchat-world-backend/core/api/worldgenesis_test.go` (rewire existing terminal-frame assertions), new cases

**Interfaces:**
- Consumes: `draftStore` (Task 1), `ArrivalCandidates` (Task 2), `authorKickstart` + `SeatWorldKickstart` (Task 3), frame v2 consts (Task 4).
- Produces: `worldGenesisHandler.drafts *draftStore` (field), shared with Task 6's kickstart route (same handler). Constructor becomes:

```go
func NewWorldGenesisHandler(pool *pgxpool.Pool, debug bool, bridge *Bridge, images *imageClient) http.Handler {
	return &worldGenesisHandler{pool: pool, dbg: debug, bridge: bridge, images: images, drafts: newDraftStore(genesisDraftTTL)}
}
```

  (Signature unchanged ⇒ main.go needs no edit; listed only for verification.)
- Produces two helpers Task 6 reuses:

```go
// characterTurnOptions renders candidates as choice options; recommended = the one matching the arrival.
func characterTurnOptions(doc *genesisDoc) []map[string]any

// scenarioTurnOptions renders authored scenarios as choice options.
func scenarioTurnOptions(scenarios []kickstartScenario) []map[string]any
```

- Question copy (fixed strings, checked against law regexes — neither contains "view as"/"switch character"): character question = `"Who are you here?"`; scenario question = `"How does it start?"`.

- [ ] **Step 1: Write the failing tests** — rewrite the existing build-stream assertions in `worldgenesis_test.go` (find every test asserting a terminal `"kind":"world"` frame from `/worlds/genesis` — they now assert a terminal `choice` frame) and add:

```go
func TestBuildEndsInCharacterChoice(t *testing.T) {
	// harness: reuse the file's existing httptest setup for POST /worlds/genesis under the fake bridge
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	last := frames[len(frames)-1]
	if last["kind"] != "choice" {
		t.Fatalf("terminal frame kind = %v, want choice", last["kind"])
	}
	if last["schema_version"] != "world_genesis_frame/2" {
		t.Fatalf("schema_version = %v", last["schema_version"])
	}
	if last["question"] != "Who are you here?" {
		t.Fatalf("question = %v", last["question"])
	}
	opts := last["options"].([]any)
	if len(opts) != 3 {
		t.Fatalf("options = %d, want 3", len(opts))
	}
	rec := 0
	for _, o := range opts {
		if o.(map[string]any)["recommended"] == true {
			rec++
		}
	}
	if rec != 1 {
		t.Fatalf("recommended = %d, want 1", rec)
	}
	if h, _ := last["handle"].(string); len(h) != 36 {
		t.Fatalf("handle = %q", h)
	}
	// AC-7: the doc never crosses the wire — no frame carries cast, history or hiding.
	for _, f := range frames {
		for _, k := range []string{"cast", "history", "hiding", "knowledge"} {
			if _, present := f[k]; present {
				t.Fatalf("frame leaked %q", k)
			}
		}
	}
}

func TestBuildSkipsToScenarioWhenIdentityStated(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town; I am the debt collector"}`)
	last := frames[len(frames)-1]
	if last["kind"] != "choice" || last["question"] != "How does it start?" {
		t.Fatalf("terminal frame = %v", last)
	}
	if len(last["options"].([]any)) != 3 {
		t.Fatal("want 3 scenario options")
	}
}
```

`postGenesisAndCollectFrames` — extract from the existing tests' SSE-splitting code if it is inline; otherwise write once here (httptest server from the file's harness, POST, split body on `\n\n`, strip `data:` prefixes, unmarshal each into `map[string]any`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestBuildEndsInCharacterChoice|TestBuildSkipsToScenario' -v`
Expected: FAIL — terminal frame is still `world`.

- [ ] **Step 3: Rewire `build`** — in `worldgenesishandler.go`, replace the post-`authorWorld` tail of `build` (currently: Begin → commitWorldGenesis → narration frames → tx.Commit → world frame → kickArt) with:

```go
	// Narration first — every line names authored content (law 2), commit or not.
	for _, line := range genesisNarration(doc) {
		if err := frames.emit("working", map[string]any{"stated": line}); err != nil {
			return
		}
	}

	draft := &genesisDraft{doc: doc, brief: req.Brief, artStyle: req.ArtStyle}
	usd, in, out, cached, calls := costs.snapshot()
	draft.tally.add(usd, in, out, cached, calls)

	var question string
	var options []map[string]any
	if len(doc.ArrivalCandidates) > 0 {
		question = "Who are you here?"
		options = characterTurnOptions(doc)
	} else {
		// Identity stated in the brief: author the scenario options now, so the stream
		// ends in the scenario question with no extra round-trip (spec, phase 1).
		k, err := authorKickstart(ctx, h.bridge.Driver(SeatWorldKickstart.Name), doc, req.Brief, doc.Arrival.CanonicalName, "")
		if err != nil {
			h.fail(frames, err)
			return
		}
		draft.identity = &k.Identity
		draft.scenarios = k.Scenarios
		usd, in, out, cached, calls = costs.snapshot()
		draft.tally = draftTally{}
		draft.tally.add(usd, in, out, cached, calls) // snapshot is cumulative per sink; reset-then-add keeps the tally honest
		question = "How does it start?"
		options = scenarioTurnOptions(k.Scenarios)
	}

	handle := h.drafts.mint()
	h.drafts.put(handle, draft)
	_ = frames.emit("choice", map[string]any{"handle": handle, "question": question, "options": options})
```

Also: remove the now-unused `tx`/commit/kickArt lines from `build` (they move to Task 6); update the deferred timing line's comment to say the aggregate line now lands at commit (keep the phase-1 line, `world=` stays `"(none)"` — it is now the *authoring* timing). Check whether `genesisNarration`'s last line renders `doc.Arrival.Stated` (worldgenesishandler.go:275-296) — it does; **drop that last line when `len(doc.ArrivalCandidates) > 0`**, because the arrival is not yet chosen and a frame must not state what has not been decided (law 2).

Add the two option helpers:

```go
func characterTurnOptions(doc *genesisDoc) []map[string]any {
	opts := make([]map[string]any, 0, len(doc.ArrivalCandidates))
	rec := strings.TrimSpace(doc.Arrival.CanonicalName)
	for _, c := range doc.ArrivalCandidates {
		o := map[string]any{"label": c.CanonicalName, "implication": c.Descriptor + " — " + c.Why}
		if strings.TrimSpace(c.CanonicalName) == rec {
			o["recommended"] = true
		}
		opts = append(opts, o)
	}
	return opts
}

func scenarioTurnOptions(scenarios []kickstartScenario) []map[string]any {
	opts := make([]map[string]any, 0, len(scenarios))
	for _, s := range scenarios {
		o := map[string]any{"label": s.Label, "implication": s.Why}
		if s.Recommended {
			o["recommended"] = true
		}
		opts = append(opts, o)
	}
	return opts
}
```

- [ ] **Step 4: Run the genesis test file**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestBuild|TestWorldGenesis' -v`
Expected: PASS, including the rewired legacy assertions. (Tests that previously walked a committed world through phase 1 alone now need Task 6's kickstart round-trip; if any such test exists, mark it with `t.Skip("moves to kickstart journey test — Task 6")` and a TODO naming Task 6, and Task 6 MUST unskip it.)

- [ ] **Step 5: Commit**

```bash
git -C dreamchat-world-backend add core/api/worldgenesishandler.go core/api/worldgenesis_test.go
git -C dreamchat-world-backend commit -m "feat: genesis stream ends in a choice frame; commit deferred to the kickstart"
```

---

### Task 6: `POST /worlds/genesis/kickstart` — turns and commit

**Files:**
- Modify: `dreamchat-world-backend/core/api/worldgenesishandler.go` (Match 75-78, ServeHTTP 91-100, new methods)
- Test: `dreamchat-world-backend/core/api/worldgenesis_test.go` (full journey), `dreamchat-world-backend/core/api/worldkickstart_test.go`

**Interfaces:**
- Consumes: everything above; `commitWorldGenesis(ctx, tx, doc, brief, artStyleChoice) (string, error)` (worldgenesiscommit.go:70) — **unchanged**; `writeJSONError` (existing helper — confirm its signature where `readBrief` uses it, worldgenesishandler.go:103-116); `kickArt(h.pool, h.images, newID)` (build's old tail).
- Produces route: `POST /worlds/genesis/kickstart`, request body:

```go
type kickstartRequest struct {
	Handle string `json:"handle"`
	Answer string `json:"answer"` // a chosen option's label, or the user's own words
}
```

- Turn logic (the whole state machine — `draft.identity == nil` is the discriminator):
  - **Character turn** (`identity == nil`): match `Answer` against `doc.ArrivalCandidates` by label (= canonical name, trimmed, case-sensitive first, case-insensitive fallback); no match ⇒ the answer is the user's own words. Call `authorKickstart(ctx, seat, doc, brief, who, "")` where `who` = matched candidate's canonical name or the free text. Store identity + scenarios in the draft, re-`put` under the **same** handle (fresh TTL), respond `done:false` with the scenario question.
  - **Scenario turn** (`identity != nil`): match `Answer` against `draft.scenarios` by label; no match ⇒ `authorKickstart(ctx, seat, doc, brief, identity.CanonicalName, Answer)` grounds the user's own opening (1 scenario). Overwrite the doc's arrival:

```go
doc.Arrival = genesisArrival{
	Descriptor:    draft.identity.Descriptor,
	CanonicalName: draft.identity.CanonicalName,
	Place:         chosen.Place,
	Stated:        chosen.Stated,
	Why:           chosen.Why,
}
doc.ArrivalCandidates = nil
```

    then `doc.validate()` (belt: the new arrival must still hit a populated place with an exit), Begin tx → `commitWorldGenesis` → Commit → drop nothing (claim already removed it) → `kickArt` → respond `done:true` with the world object → log the aggregate timing line.
  - **Expired/unknown handle** ⇒ `writeJSONError(w, http.StatusGone, errDraftExpired.Error())` (adapt to the helper's real signature). No debris: nothing was ever committed.
  - **Seat error or refusal mid-turn** ⇒ the draft was claimed; `put` it back under the same handle before responding, so the user may retry: refusal ⇒ 422 with the refusal's stated reason; other errors ⇒ 502 `"the opening could not be authored"`. A commit failure also re-puts (tx rolled back, world absent — AC-2 holds).
- Cost/timing: `withCostSink` per request; after each seat call `draft.tally.add(costs.snapshot())`-style accumulation (snapshot is cumulative within one sink — take it once per request, after all calls). On commit:

```go
log.Printf("world genesis timing: total_ms=%d world=%s calls=%d tok_in=%d cached=%d tok_out=%d cost_usd=%.6f session_usd=%.4f",
	/* total_ms: this request's elapsed */, newID, draft.tally.calls, draft.tally.tokIn, draft.tally.cached, draft.tally.tokOut, draft.tally.usd, sessionTotalUSD())
```

  and the `COST WARNING` check against `genesisCostCeilingUSD()` uses `draft.tally.usd` (the whole build's spend — spec AC-10). Mirror the exact format of the existing deferred logger (build, step 3 of its closure).

- [ ] **Step 1: Write the failing journey test**

```go
func TestKickstartFullJourney(t *testing.T) {
	// Same httptest harness as TestBuildEndsInCharacterChoice; needs the test pg pool
	// (follow the DB-backed genesis tests' setup in this file).
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	choice := frames[len(frames)-1]
	handle := choice["handle"].(string)

	// Turn 1: pick the recommended character.
	recLabel := recommendedLabel(t, choice["options"])
	turn := postKickstart(t, handle, recLabel) // POST /worlds/genesis/kickstart, decode JSON
	if turn["schema_version"] != "world_kickstart_turn/1" || turn["done"] != false {
		t.Fatalf("turn 1 = %v", turn)
	}
	if turn["question"] != "How does it start?" {
		t.Fatalf("turn 1 question = %v", turn["question"])
	}

	// Turn 2: pick the recommended scenario — this commits.
	turn2 := postKickstart(t, handle, recommendedLabel(t, turn["options"]))
	if turn2["done"] != true {
		t.Fatalf("turn 2 = %v", turn2)
	}
	world := turn2["world"].(map[string]any)
	if world["playable"] != true || world["id"] == "" {
		t.Fatalf("world = %v", world)
	}

	// The handle is spent: a third answer is a stated refusal, no debris.
	res := postKickstartRaw(t, handle, "anything")
	if res.StatusCode != http.StatusGone {
		t.Fatalf("spent handle status = %d, want 410", res.StatusCode)
	}

	// DB floor: player stamped, exactly one perception (AC-4/AC-9) — reuse the assertions
	// the pre-split committed-world test made (the one Task 5 skipped); unskip and fold it in here.
}

func TestKickstartCustomAnswersFlowIn(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	handle := frames[len(frames)-1]["handle"].(string)
	turn := postKickstart(t, handle, "the harbour master's estranged child, back unannounced")
	if turn["done"] != false {
		t.Fatalf("custom character rejected: %v", turn)
	}
	turn2 := postKickstart(t, handle, "I slip in through the kitchen while an argument is going on")
	if turn2["done"] != true {
		t.Fatalf("custom scenario did not commit: %v", turn2)
	}
}

func TestKickstartExpiredHandle(t *testing.T) {
	res := postKickstartRaw(t, "00000000-0000-0000-0000-000000000000", "anything")
	if res.StatusCode != http.StatusGone {
		t.Fatalf("status = %d, want 410", res.StatusCode)
	}
}
```

Helpers `postKickstart`/`postKickstartRaw`/`recommendedLabel`: thin wrappers over the harness's server URL; write them beside `postGenesisAndCollectFrames`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestKickstart' -v`
Expected: FAIL — 404/405 from the router (route not matched).

- [ ] **Step 3: Implement the route** — in `worldgenesishandler.go`:
  - Add `genesisKickstartRoute = regexp.MustCompile(`^/worlds/genesis/kickstart$`)` beside the existing two (53-56); extend `Match` and `ServeHTTP` to dispatch it to a new `func (h *worldGenesisHandler) kickstart(w http.ResponseWriter, r *http.Request)`.
  - **Route order caution:** the existing genesis route regex is `^/worlds/genesis$` (exact), so no collision — verify with the router test if one exists.
  - Implement `kickstart` exactly per the Interfaces block above. Response writing mirrors `interview`'s conditional-body pattern (121-157):

```go
	body := map[string]any{"schema_version": worldKickstartTurnSchemaVersion, "done": done}
	if !done {
		body["question"] = question
		body["options"] = options
	} else {
		body["world"] = map[string]any{"id": newID, "display_name": doc.World.DisplayName, "tagline": doc.World.Tagline, "playable": true}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
```

  - Unskip the test(s) Task 5 skipped and fold their DB assertions into `TestKickstartFullJourney`.

- [ ] **Step 4: Run the whole genesis + kickstart surface**

Run: `cd dreamchat-world-backend && go test ./core/api/ -run 'TestKickstart|TestBuild|TestWorldGenesis|TestAuthorKickstart' -v`
Expected: PASS, zero skips remaining.

- [ ] **Step 5: Commit**

```bash
git -C dreamchat-world-backend add core/api/worldgenesishandler.go core/api/worldgenesis_test.go core/api/worldkickstart_test.go
git -C dreamchat-world-backend commit -m "feat: kickstart turns — character, scenario, then the one-transaction commit"
```

---

### Task 7: Payload captures + contract CI

**Files:**
- Modify: `dreamchat-world-backend/core/api/worldgenesispayloads_test.go`
- Modify: `dreamchat-world-backend/ci/gen_payloads.sh` (only if the `-run` regex or output list needs the new names — read it first)
- Regenerate: the payload fixtures directory the script writes (run it; commit outputs wherever the repo tracks them)

**Interfaces:**
- Consumes: `recordingDriver` (schema_payloads_test.go:42-56), `sseFrames`/`jsonPost` helpers, the fake drivers.
- Produces payload files (filename-keyed by `ci/schema_contract.py`):
  - `world_kickstart_1.json` — raw seat output via `recordingDriver` around the fake, driven through the REAL `authorKickstart` path.
  - `world_genesis_frame_2_working.json`, `world_genesis_frame_2_choice.json`, `world_genesis_frame_2_refused.json`, `world_genesis_frame_2_error.json` — one per v2 frame kind, captured from the real handler via httptest (the refusal capture keeps its wrong-shaped-fake trick).
  - `world_kickstart_turn_1_question.json`, `world_kickstart_turn_1_world.json` — both oneOf branches of the turn, captured from the real kickstart route.
  - Regenerated `world_genesis_1.json` (the fake now emits `arrival_candidates`).
  - **Delete** the stale `world_genesis_frame_1_*.json` captures.

- [ ] **Step 1: Extend `TestGenSeatContractPayloads`** — add a `world_kickstart` block mirroring the existing `world_genesis` one: build a doc with the genesis fake, wrap `NewFakeWorldKickstartDriver()` in `recordingDriver`, call `authorKickstart(ctx, rec, doc, brief, doc.Arrival.CanonicalName, "")`, then `writePayload(t, dir, "world_kickstart_1.json", []byte(rec.last))`.

- [ ] **Step 2: Extend `TestGenGenesisAPIPayloads`** — drive the real journey (Task 6's shape): capture each distinct frame `kind` from the stream into `world_genesis_frame_2_<kind>.json`; then POST both kickstart turns and write `world_kickstart_turn_1_question.json` / `world_kickstart_turn_1_world.json` from the raw response bodies. Update the refusal/error captures' filenames to `_2_`.

- [ ] **Step 3: Run the generator and the contract check**

Run: `cd dreamchat-world-backend && bash ci/gen_payloads.sh && make schema-contract`
Expected: generator writes all files named above; `make schema-contract` PASSES — every published schema (`world_kickstart.v1`, `world_kickstart_turn.v1`, `world_genesis_frame.v2`) has a validating payload, and no orphan v1 frame schema remains.

- [ ] **Step 4: Commit**

```bash
git -C dreamchat-world-backend add -A
git -C dreamchat-world-backend commit -m "test: contract payloads for world_kickstart, kickstart turn and frame v2"
```

---

### Task 8: Frontend contract plumbing

**Files:**
- Create: `dream-weaver-visuals/contracts/world_genesis_frame.v2.schema.json`, `dream-weaver-visuals/contracts/world_kickstart_turn.v1.schema.json` (vendored byte-for-byte from `dreamchat-world-backend/core/api/schema/`)
- Delete: `dream-weaver-visuals/contracts/world_genesis_frame.v1.schema.json`
- Modify: `dream-weaver-visuals/scripts/verify-contract.sh` (FILES array), `dream-weaver-visuals/scripts/verify-types.sh` (PAIRS array)
- Regenerate: `dream-weaver-visuals/src/api/types/world_genesis_frame.ts`; Create: `dream-weaver-visuals/src/api/types/world_kickstart_turn.ts`

**Interfaces:**
- Produces the generated types Task 9 aliases. The generated names are sentence-long (derived from schema titles) — read the regenerated files for the exact names before writing Task 9's imports.

- [ ] **Step 1: Vendor the schemas**

```bash
cp dreamchat-world-backend/core/api/schema/world_genesis_frame.v2.schema.json dream-weaver-visuals/contracts/
cp dreamchat-world-backend/core/api/schema/world_kickstart_turn.v1.schema.json dream-weaver-visuals/contracts/
rm dream-weaver-visuals/contracts/world_genesis_frame.v1.schema.json
```

- [ ] **Step 2: Update the two script arrays** — in `verify-contract.sh` FILES: replace `world_genesis_frame.v1.schema.json` with `world_genesis_frame.v2.schema.json`, add `world_kickstart_turn.v1.schema.json`. In `verify-types.sh` PAIRS: update the frame pair's schema filename, add `"contracts/world_kickstart_turn.v1.schema.json:src/api/types/world_kickstart_turn.ts"`.

- [ ] **Step 3: Regenerate types**

```bash
cd dream-weaver-visuals
./node_modules/.bin/json2ts -i contracts/world_genesis_frame.v2.schema.json > src/api/types/world_genesis_frame.ts
./node_modules/.bin/json2ts -i contracts/world_kickstart_turn.v1.schema.json > src/api/types/world_kickstart_turn.ts
```

- [ ] **Step 4: Verify**

Run: `cd dream-weaver-visuals && bash scripts/verify-types.sh && DREAMCHAT_BACKEND_SCHEMA=../dreamchat-world-backend/core/api/schema bash scripts/verify-contract.sh`
Expected: both PASS (contract check against the local sibling, since backend main has not merged yet — note this in the commit message).

- [ ] **Step 5: Commit** (types + contracts + scripts, one commit — the PIN moves in Task 9's commit alongside its consumer, which is acceptable because nothing imports the new types yet; TypeScript still compiles)

```bash
git -C dream-weaver-visuals add contracts scripts src/api/types
git -C dream-weaver-visuals commit -m "contract: vendor world_genesis_frame/2 and world_kickstart_turn/1 (backend sibling, pre-merge)"
```

---

### Task 9: Frontend API — choice frame + kickstart turns

**Files:**
- Modify: `dream-weaver-visuals/src/api/genesis.ts` (type aliases 19-23, PIN 42-46, new function)
- Test: colocated with the module's existing test conventions — check for a `genesis.test.ts`; if none exists, the laws suite plus Task 10's flow exercise it and no new unit file is created.

**Interfaces:**
- Consumes: generated types (Task 8 — use their real names), `apiBase`/`apiFetch`/`SchemaMismatchError` (src/api/index.ts).
- Produces (Task 10 relies on these exact names):

```ts
export type GenesisFrame = /* alias of the regenerated frame v2 type */;
export type KickstartTurn = /* alias of the generated turn type */;
export type ChoiceOption = NonNullable<KickstartTurn["options"]>[number];

export async function answerKickstart(handle: string, answer: string): Promise<KickstartTurn>
```

- PIN moves: `genesisFrame: "world_genesis_frame/2"`, add `kickstart: "world_kickstart_turn/1"`.

- [ ] **Step 1: Implement** — in `genesis.ts`:
  - Update the frame type alias import to the regenerated name; add the turn alias.
  - Bump/extend `PIN`:

```ts
const PIN = {
  artStyles: "art_styles/1",
  interview: "world_interview_turn/1",
  kickstart: "world_kickstart_turn/1",
  genesisFrame: "world_genesis_frame/2",
} as const;
```

  - Add, mirroring `askInterview` (81-96) exactly — same fetch wrapper, same non-ok handling, same pin check:

```ts
/** One kickstart answer in, the next question or the built world out. `answer` is a chosen
 *  option's label or the user's own words — the server cannot tell and must not care. */
export async function answerKickstart(handle: string, answer: string): Promise<KickstartTurn> {
  const res = await apiFetch(`${apiBase()}/worlds/genesis/kickstart`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ handle, answer }),
  });
  if (!res.ok) {
    if (res.status === 410) throw new Error("that build has expired — write the brief again and rebuild");
    throw new Error(`request failed: ${res.status}`);
  }
  const turn = (await res.json()) as KickstartTurn;
  if (turn.schema_version !== PIN.kickstart) throw new SchemaMismatchError(PIN.kickstart, turn.schema_version);
  return turn;
}
```

  (Match `askInterview`'s real error copy/pattern when it differs from the above — mirror, don't invent.)

- [ ] **Step 2: Typecheck**

Run: `cd dream-weaver-visuals && ./node_modules/.bin/tsc --noEmit`
Expected: clean. `buildWorld`'s frame switch consumers may now fail on the removed `world` kind — that is Task 10's edit; if tsc fails ONLY inside `create.tsx` on the frame type, proceed to Task 10 before committing and land Tasks 9+10 as one commit instead.

- [ ] **Step 3: Commit** (or fold into Task 10 per Step 2)

```bash
git -C dream-weaver-visuals add src/api/genesis.ts
git -C dream-weaver-visuals commit -m "feat: kickstart turn client + frame v2 pin"
```

---

### Task 10: Create flow — the two choice screens

**Files:**
- Modify: `dream-weaver-visuals/src/routes/create.tsx` (Lane union 509-517, build() frame switch 125-145, Question component 400-497, render 300-400)
- Test: `cd dream-weaver-visuals && npx vitest run src/laws` (the laws suite is the binding test); manual smoke via the running stack.

**Interfaces:**
- Consumes: `answerKickstart`, `KickstartTurn`, `GenesisFrame` (Task 9); existing `Question` component grammar.
- Produces Lane additions (existing states unchanged):

```ts
type Lane =
  | { state: "writing" } | { state: "asking" }
  | { state: "question"; turn: InterviewTurn }
  | { state: "ready" }
  | { state: "building"; lines: string[] }
  | { state: "choosing"; handle: string; question: string; options: ChoiceOption[] }
  | { state: "committing" }
  | { state: "built"; id: string; displayName: string; tagline: string }
  | { state: "refused"; stated: string } | { state: "failed"; stated: string };
```

- Copy constraints (laws): the choice screens' strings must not contain `view as`, `switch character/user/viewer`, Glossary words, or invented numbers. Question copy arrives verbatim from the server (`frame.question`, law 1). The accept-defaults escape button reads **"Start here"**.

- [ ] **Step 1: Handle the choice frame** — in `build()`'s frame switch (125-145): remove the `world` case (frame v2 has none); add:

```ts
case "choice":
  setLane({
    state: "choosing",
    handle: frame.handle ?? "",
    question: frame.question ?? "",
    options: frame.options ?? [],
  });
  return;
```

(`working`/`refused`/`error` cases unchanged.)

- [ ] **Step 2: Answer flow** — add beside `answer()`:

```ts
async function choose(handle: string, said: string) {
  if (inFlight.current) return;
  inFlight.current = true;
  setLane({ state: "committing" });
  try {
    const turn = await answerKickstart(handle, said);
    if (turn.done && turn.world) {
      setLane({ state: "built", id: turn.world.id, displayName: turn.world.display_name, tagline: turn.world.tagline });
    } else {
      setLane({ state: "choosing", handle, question: turn.question ?? "", options: turn.options ?? [] });
    }
  } catch (err) {
    setLane({ state: "failed", stated: err instanceof Error ? err.message : "the opening could not be reached" });
  } finally {
    inFlight.current = false;
  }
}
```

(Match the file's real double-submit/`typed`-reset conventions — read `answer()` (99-115) and mirror it, including clearing `typed` after a send.)

- [ ] **Step 3: Render the choosing lane** — reuse the `Question` component. It takes an `InterviewTurn`; construct one (`{ done: false, question: lane.question, options: lane.options }`) and parameterize the escape button: add an optional prop `escape?: { label: string; onPress: () => void }` to `Question`, defaulting to the current "Build it now" behavior so interview callsites are untouched. For the choosing lane:

```tsx
{lane.state === "choosing" && (
  <Question
    turn={{ done: false, question: lane.question, options: lane.options }}
    typed={typed}
    onTyped={setTyped}
    answered={0}
    onChoose={(label) => void choose(lane.handle, label)}
    onBuild={() => {}}
    escape={{
      label: "Start here",
      onPress: () => {
        const rec = lane.options.find((o) => o.recommended) ?? lane.options[0];
        if (rec) void choose(lane.handle, rec.label);
      },
    }}
  />
)}
{lane.state === "committing" && <p aria-live="polite">Opening the way in&hellip;</p>}
```

Note on `answered={0}`: check what `Question` renders for it ("N answered so far") — if 0 renders an awkward line, make the counter line conditional on `answered > 0` inside `Question` (a targeted improvement, not a restyle). The `lane.options.find(...)` on a payload array is a lookup, not a reorder — the laws regex targets `.sort/.filter/.reverse` on payload field names, and `options` is not in that list; render the list itself unfiltered and in server order.

- [ ] **Step 4: Laws + typecheck**

Run: `cd dream-weaver-visuals && ./node_modules/.bin/tsc --noEmit && npx vitest run src/laws`
Expected: both PASS. If a law trips, fix the copy, not the law.

- [ ] **Step 5: Commit**

```bash
git -C dream-weaver-visuals add src/routes/create.tsx src/api/genesis.ts
git -C dream-weaver-visuals commit -m "feat: create flow pauses for who-are-you and how-it-starts before the world commits"
```

---

### Task 11: PRD amendment + full verification

**Files:**
- Modify: `dreamchat-world-backend/docs/10_prds/prd_world_creation.md` (AC-6, line 67)

- [ ] **Step 1: Amend AC-6** — append to the AC-6 paragraph (do not delete its existing text):

```
**Amended 2026-08-20** (spec: `docs/superpowers/specs/2026-08-20-kickstart-arrival-choice-design.md`): a pre-play choice among authored *premises* — the kickstart's character candidates and scenarios — is not the roster this criterion forbids, by the same reasoning that amended law 12 for the login gate: what the rule protects is the D-7 mid-play perception boundary, and a premise choice at tick 0 touches no perception. What stays forbidden is unchanged: no mid-play switching, no "view as", and no traits, core or inner state for the player under any candidate.
```

- [ ] **Step 2: Full backend suite**

Run: `cd dreamchat-world-backend && go build ./... && go test ./core/api/ && make schema-contract`
Expected: all PASS.

- [ ] **Step 3: Full frontend gates**

Run: `cd dream-weaver-visuals && ./node_modules/.bin/tsc --noEmit && npx vitest run && bash scripts/verify-types.sh && DREAMCHAT_BACKEND_SCHEMA=../dreamchat-world-backend/core/api/schema bash scripts/verify-contract.sh`
Expected: all PASS.

- [ ] **Step 4: Smoke the journey end-to-end** — start the stack the way `.stack/` scripts do (fake bridge: `DREAMCHAT_BRIDGE=fake`), open `/create`, write an identity-open brief, build, answer both questions (once via options, once via free text on a second build; once via "Start here" on a third), click "Walk in", and confirm the arrival scene renders with exactly the chosen premise. Confirm an identity-stated brief ("… I am the …") skips straight to "How does it start?".

- [ ] **Step 5: Commit**

```bash
git -C dreamchat-world-backend add docs/10_prds/prd_world_creation.md
git -C dreamchat-world-backend commit -m "docs: AC-6 amended — a pre-play premise choice is not the forbidden roster"
```
