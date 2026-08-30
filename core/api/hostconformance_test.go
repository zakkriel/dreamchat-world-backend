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
	"os"
	"path/filepath"
	"strings"
	"testing"
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
