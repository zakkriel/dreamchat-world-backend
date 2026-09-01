package main

// SPEC-039's standing instrument. One allowlisted host mis-parsed 6 of 24 beats on BYTE-IDENTICAL
// input while two others scored 24 of 24 — same model name, same prompt, same schema, different
// answer, because a host's own implementation of constrained decoding collapsed to the schema's first
// branch. Nothing in this service can see that: the seat reports a valid chain, the gate accepts it
// because it IS structurally valid, and the player watches "I say 'Evening'" become a walk.
//
// The only way to see it is to ask each host the same question separately. That needs the EXACT bytes
// the running driver sends, not a reconstruction — four rounds of this investigation died on
// variables I had not held still (the model, the routing, my own probe beats mutating the scene). So
// this dumps the real assembled prompt and the real schema from the real call path, once, and
// ci/host_conformance.py replays those bytes per host with only the player's sentence substituted.
//
// Gated on HOST_PROBE_DIR, so the ordinary `go test ./...` run skips it and writes nothing — the same
// convention as TestGenSceneCurrentPayloads/SCENE_PAYLOAD_DIR and the SEAT_PAYLOAD_DIR generators
// (schema_payloads_test.go). It costs no API calls: it only writes the request material.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// hostProbePlaceholder is the token ci/host_conformance.py substitutes with each corpus sentence. The
// decompose prompt carries the player's words in its TAIL (buildDecomposePrompt), so substituting a
// placeholder there reproduces a real request exactly, while the header, SCENE and CANDIDATES blocks
// stay byte-identical across every host and every sentence. Holding those still is the whole point.
const hostProbePlaceholder = "__PLAYER_TEXT__"

// TestGenDecomposeHostProbe writes prompt.txt (with the placeholder) and schema.json for
// ci/host_conformance.py. It drives the REAL perception-bound payload assembly against the seeded
// Drowned Lantern world — beatHandler.payload, the same call the live handler makes — so CANDIDATES
// is a genuine viewer-bound list and not a fixture that would prove only that a fake matches itself.
func TestGenDecomposeHostProbe(t *testing.T) {
	dir := os.Getenv("HOST_PROBE_DIR")
	if dir == "" {
		t.Skip("HOST_PROBE_DIR unset — this generator only runs from ci/host_conformance.sh")
	}

	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	bh := &beatHandler{pool: pool}
	pre, err := bh.payload(ctx, dlWorldID, dlKadeID)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}

	prompt := buildDecomposePrompt(pre, hostProbePlaceholder)
	if !strings.Contains(prompt, hostProbePlaceholder) {
		t.Fatalf("the assembled prompt does not carry %s — buildDecomposePrompt no longer places the "+
			"player's words verbatim, so substitution would silently probe the wrong thing", hostProbePlaceholder)
	}

	if err := os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte(prompt), 0o644); err != nil {
		t.Fatalf("write prompt.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "schema.json"), []byte(beatChainV2SchemaJSON), 0o644); err != nil {
		t.Fatalf("write schema.json: %v", err)
	}

	t.Logf("host probe material: prompt %d bytes, %d candidates, schema %d bytes",
		len(prompt), len(pre.Candidates), len(beatChainV2SchemaJSON))
}

// TestGenFillHostProbe dumps the real assembled WORLD_FILL request, for ci/fill_conformance.sh.
//
// Why fill needs its own probe rather than a line in the SPEC-039 corpus. That corpus is a fixed
// ruler for one seat — sentence in, attempt type out — and its own header says a ruler that changes
// between measurements measures nothing. Fill is not a harder sentence; it is a different workload:
//
//   - the request is ~10x larger (identity, plus everything authored so far)
//   - the reply is up to 16384 tokens against a deeply nested schema with additionalProperties:false
//   - production runs it in json_object mode, not the json_schema strict mode the decompose probe uses
//
// A host can be flawless on decompose and still truncate or mis-nest here, and on 2026-08-28 three
// live fill calls failed exactly that way: unknown field "canonical_name", unknown field "hiding",
// and unexpected EOF. Nothing in the decompose corpus would have caught any of them.
//
// The material is a mid-build state: identity minted, descent merged, and the work item is the
// per-person ascent call — the one that failed with unknown field "hiding".
func TestGenFillHostProbe(t *testing.T) {
	dir := os.Getenv("HOST_PROBE_DIR")
	if dir == "" {
		t.Skip("HOST_PROBE_DIR unset — this generator only runs from ci/fill_conformance.sh")
	}
	ctx := context.Background()

	id, err := inferIdentity(ctx, NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}

	// Walk the NAMESPACE with the deterministic fake so the probe's "already authored" block is a real
	// mid-build document rather than a hand-made one.
	doc := &genesisDoc{}
	var tags []taggedName
	seat := NewFakeWorldFillDriver()
	b := budgetForDepth(0)
	namespace := []workItem{conceptsWork(), scaffoldOneWork(b)}
	for _, item := range namespace {
		frag, err := fillOne(ctx, seat, id, item, testBrief, nil, doc, "")
		if err != nil {
			t.Fatalf("namespace %s: %v", item.ID, err)
		}
		mergeFill(doc, frag, mergeTag(item), &tags)
	}
	for _, item := range scaffoldTwoSchedule(doc, b) {
		frag, err := fillOne(ctx, seat, id, item, testBrief, nil, doc, "")
		if err != nil {
			t.Fatalf("scaffold 2 %s: %v", mergeTag(item), err)
		}
		mergeFill(doc, frag, mergeTag(item), &tags)
	}
	if len(doc.Cast) == 0 {
		t.Fatal("the namespace authored nobody — the probe would measure the wrong call")
	}

	// A people pack is the largest fill call there is, so it is the one worth scoring: the most output
	// tokens, the deepest schema, and where truncation showed up in production.
	var person workItem
	for _, wave := range contentSchedule(doc, b) {
		for _, item := range wave {
			if item.ID == "people" {
				person = item
				break
			}
		}
		if person.ID != "" {
			break
		}
	}
	if person.ID == "" {
		t.Fatal("no people work item — the scaffold marked nobody above relevance 1, so nothing is owed")
	}

	prompt := buildWorldFillPrompt(id, person, testBrief, nil, doc, "")
	if err := os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte(prompt), 0o644); err != nil {
		t.Fatalf("write prompt.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "schema.json"), []byte(worldFillSchemaJSON), 0o644); err != nil {
		t.Fatalf("write schema.json: %v", err)
	}
	t.Logf("fill probe material: prompt %d bytes, pack of %d in %q, %d place(s)/%d person(s) already authored, schema %d bytes",
		len(prompt), len(person.Members), person.Subject, len(doc.Places), len(doc.Cast), len(worldFillSchemaJSON))
}

// TestFillProbe runs THE FILL AND NOTHING ELSE against the real seats: understanding pass, every wave,
// the reconciliation, the belt. No database, no commit, no kickstart, no play.
//
// WHY IT IS NOT THE GENESIS ENDPOINT. The filling stage is the thing under development, and measuring it
// through `POST /worlds/genesis` measures the whole pipeline: it needs a database, it writes a half-world
// into production on the way to answering a question about prose, it is cut off by a 900-second proxy
// edge, and its logs are gone within the hour. This runs the fill in-process, prints the document to a
// file you can read, and prints the ledger of where every token went.
//
// NOT A CI GATE — it spends real money. Run it when you want to know what the fill costs or whether it
// got better:
//
//	FILL_PROBE_DIR=/tmp/fill ci/fill_probe.sh path/to/brief.md
func TestFillProbe(t *testing.T) {
	dir := os.Getenv("FILL_PROBE_DIR")
	if dir == "" {
		t.Skip("FILL_PROBE_DIR unset — this probe spends real money and is never part of a suite")
	}
	briefPath := os.Getenv("FILL_PROBE_BRIEF")
	if briefPath == "" {
		t.Fatal("FILL_PROBE_BRIEF unset — the probe needs a brief to fill from")
	}
	raw, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatalf("read brief: %v", err)
	}
	brief := strings.TrimSpace(string(raw))
	if brief == "" {
		t.Fatal("the brief is empty")
	}
	depth := 1
	if v := os.Getenv("FILL_PROBE_DEPTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			depth = n
		}
	}

	cfg, err := seatConfigFromEnv(os.Getenv)
	if err != nil {
		t.Fatalf("seat config: %v", err)
	}
	bridge, err := NewBridge(cfg, DefaultDriverFactory, SeatWorldUnderstanding, SeatWorldFill, SeatWorldFillReview)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	for _, s := range []Seat{SeatWorldUnderstanding, SeatWorldFill, SeatWorldFillReview} {
		d := bridge.Driver(s.Name)
		if d == nil {
			t.Fatalf("seat %s is not bound — set DREAMCHAT_SEAT_DEFAULT or DREAMCHAT_SEATS", s.Name)
		}
		t.Logf("seat %-20s -> %s", s.Name, d.Name())
	}

	// The same cost sink the handler installs, so the probe reports the same dollars production would.
	ctx, costs := withCostSink(context.Background())
	start := time.Now()
	doc, ident, ferr := authorWorld(ctx,
		bridge.Driver(SeatWorldUnderstanding.Name),
		bridge.Driver(SeatWorldFill.Name),
		bridge.Driver(SeatWorldFillReview.Name),
		brief, nil, nil, nil, depth)
	wall := time.Since(start)
	usd, tokIn, tokOut, cached, calls := costs.snapshot()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// The document is the product of this stage. Write it whether or not the belt accepted it: a refused
	// document is the one you most want to read.
	if doc != nil {
		body, _ := json.MarshalIndent(doc, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "document.json"), body, 0o644); err != nil {
			t.Fatalf("write document.json: %v", err)
		}
		t.Logf("document.json: %d bytes", len(body))
	}
	if ident != nil {
		body, _ := json.MarshalIndent(ident, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, "identity.json"), body, 0o644)
	}

	t.Logf("FILL: wall=%.0fs calls=%d tok_in=%d cached=%d tok_out=%d usd=%.4f depth=%d",
		wall.Seconds(), calls, tokIn, cached, tokOut, usd, depth)
	if ferr != nil {
		// A refusal is a RESULT here, not a test failure: the probe exists to report what the fill did.
		t.Logf("FILL REFUSED: %v", ferr)
		return
	}
	t.Logf("WORLD: %d location(s), %d person/people, %d faction(s), %d concept(s), %d object(s), %d way(s), %d event(s)",
		len(doc.Places), len(doc.Cast), len(doc.Factions), len(doc.Concepts), len(doc.Objects), len(doc.Ways), len(doc.History))
	rel := map[int]int{}
	for _, a := range doc.Cast {
		rel[a.Relevance]++
	}
	t.Logf("PEOPLE BY RELEVANCE: 1=%d 2=%d 3=%d 4=%d", rel[1], rel[2], rel[3], rel[4])
}
