package main

import (
	"strings"
	"testing"
)

// ── the structural guards on the LLM-to-canon boundary ──────────────────────────────────────────
//
// WHY THIS FILE EXISTS, AND WHY IT IS SEPARATE.
//
// QA-SPAN-2026-08-11 §2 reported that "every structural guard in the resolve seat's ruling validator
// survived — closed vocabulary, actor_id, truth, zero-event coherence. That is the LLM-to-canon gate."
// Re-measured 2026-08-26 with `ci/mutate.sh` against a clean baseline: still true, all four. Each of
// these guards could be replaced with `if false` and the entire Go suite stayed green.
//
// `ruling_v2_test.go` covers the happy path, one flip case, one per-type field guard and the tier
// guard. What it never did was feed a ruling that violates the four guards named above — so the
// validator's own reason for existing was undefended for fifteen days after being reported.
//
// This matters more than a normal coverage hole. `DecodeAndValidateRulingV2` is the belt on what a
// language model is permitted to write into canon. The closed vocabulary is enforced at token
// generation by the structured-output leash (ADR-009/D-1, SPEC-015), but that is the driver's
// contract, and D-13 says no seat may assume a driver's capabilities beyond its reported floor. This
// function is what stands between a model that ignores its schema and a permanent canon event.
//
// Every test below is written as a MUTATION TARGET: it names the exact guard it defends, so a future
// reader deleting that guard is told which test to expect a failure from.

// baseValidV2 is a minimal ruling that must decode cleanly. Every negative test below is this document
// with exactly ONE thing wrong, so a failure names its own cause rather than "some JSON was rejected".
const baseValidV2 = `{
	"reasoning": "he is unguarded; the shove lands",
	"therefore": "succeeds",
	"outcome": {
		"kind": "resolved",
		"events": [
			{
				"type": "AttributeChanged",
				"actor_id": "33333333-3333-3333-3333-333333333333",
				"target_id": "44444444-4444-4444-4444-444444444444",
				"truth": "Jonas stumbles back a pace.",
				"visible": true
			}
		]
	}
}`

// The control. If this ever fails, every negative test below is meaningless — they would be passing
// because the fixture is broken, not because the guard fired.
func TestRulingV2Guards_BaseFixtureIsValid(t *testing.T) {
	if _, err := DecodeAndValidateRulingV2(baseValidV2); err != nil {
		t.Fatalf("the base fixture must decode cleanly, otherwise every negative test here is vacuous: %v", err)
	}
}

// GUARD: `if !allowedRuledEventTypes[e.Type]` — the closed vocabulary.
// This is the one that matters most. A model that invents an event type must not reach apply_event,
// because the engine's whole contract is that the vocabulary of what can happen is finite and
// founder-locked (FINAL-action-contracts.md).
func TestRulingV2Guards_EventTypeOutsideClosedVocabulary(t *testing.T) {
	bad := strings.Replace(baseValidV2, `"type": "AttributeChanged"`, `"type": "ActorAscended"`, 1)
	_, err := DecodeAndValidateRulingV2(bad)
	if err == nil {
		t.Fatal("an event type outside the closed vocabulary was ACCEPTED — the LLM-to-canon gate is open")
	}
	if !strings.Contains(err.Error(), "closed vocabulary") {
		t.Fatalf("rejected, but not by the vocabulary guard — got %q", err)
	}
}

// GUARD: `if e.ActorID == ""` — every canon event names who did it.
// An event with no actor cannot be attributed, and provenance (I-2) is not reconstructable after the
// fact.
func TestRulingV2Guards_EventMissingActorID(t *testing.T) {
	bad := strings.Replace(baseValidV2,
		`"actor_id": "33333333-3333-3333-3333-333333333333",`, ``, 1)
	_, err := DecodeAndValidateRulingV2(bad)
	if err == nil {
		t.Fatal("an event with no actor_id was ACCEPTED — an unattributable canon event")
	}
	if !strings.Contains(err.Error(), "actor_id") {
		t.Fatalf("rejected, but not by the actor_id guard — got %q", err)
	}
}

// GUARD: `if e.Truth == ""` — every event carries what actually happened.
// `truth` is the authoritative record; `appearance` is what observers may be told instead (B-6, the
// deception split). An event with an appearance and no truth would make the lie the canon.
func TestRulingV2Guards_EventMissingTruth(t *testing.T) {
	bad := strings.Replace(baseValidV2,
		`"truth": "Jonas stumbles back a pace.",`, ``, 1)
	_, err := DecodeAndValidateRulingV2(bad)
	if err == nil {
		t.Fatal("an event with no truth was ACCEPTED — canon with no authoritative record")
	}
	if !strings.Contains(err.Error(), "truth") {
		t.Fatalf("rejected, but not by the truth guard — got %q", err)
	}
}

// GUARD: `if r.Reasoning == ""` — explain-first.
// The seat must state its reasoning before its verdict. A ruling with no reasoning is a verdict with
// no audit trail, and the resolve seat's entire design is that the reasoning comes first so the
// outcome cannot be back-fitted to it.
func TestRulingV2Guards_MissingReasoning(t *testing.T) {
	bad := strings.Replace(baseValidV2,
		`"reasoning": "he is unguarded; the shove lands",`, ``, 1)
	_, err := DecodeAndValidateRulingV2(bad)
	if err == nil {
		t.Fatal("a ruling with no reasoning was ACCEPTED — explain-first is not enforced")
	}
	if !strings.Contains(err.Error(), "reasoning") {
		t.Fatalf("rejected, but not by the reasoning guard — got %q", err)
	}
}

// GUARD: the `therefore` / `outcome` coherence switch, in all four of its directions.
// QA named "zero-event coherence" specifically: `therefore=succeeds` with an empty event list is a
// seat claiming something happened and recording nothing, which is the shape of a silent no-op
// reaching the player as a success.
func TestRulingV2Guards_ThereforeOutcomeCoherence(t *testing.T) {
	cases := []struct {
		name, body, wantIn string
	}{
		{
			name:   "succeeds with zero events",
			body:   `{"reasoning":"it lands","therefore":"succeeds","outcome":{"kind":"resolved","events":[]}}`,
			wantIn: "events",
		},
		{
			name:   "fails with zero events",
			body:   `{"reasoning":"it does not land","therefore":"fails","outcome":{"kind":"resolved","events":[]}}`,
			wantIn: "events",
		},
		{
			name:   "impossible but resolved",
			body:   `{"reasoning":"physics forbids it","therefore":"impossible","outcome":{"kind":"resolved","events":[]}}`,
			wantIn: "flip",
		},
		{
			name:   "therefore outside the enum",
			body:   `{"reasoning":"unclear","therefore":"maybe","outcome":{"kind":"bounce","reason":"x"}}`,
			wantIn: "enum",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := DecodeAndValidateRulingV2(c.body)
			if err == nil {
				t.Fatalf("%s was ACCEPTED — the therefore/outcome switch does not hold", c.name)
			}
			if !strings.Contains(err.Error(), c.wantIn) {
				t.Fatalf("rejected, but not by the expected branch (want %q) — got %q", c.wantIn, err)
			}
		})
	}
}

// GUARD: the remaining attribute_write field guards. Tier is already covered by
// TestRulingV2_AttrWriteTierThreeRejected; these are its three untested siblings.
func TestRulingV2Guards_AttributeWriteFields(t *testing.T) {
	base := `{
		"reasoning": "the door gives",
		"therefore": "succeeds",
		"outcome": {
			"kind": "resolved",
			"events": [{"type":"AttributeChanged","actor_id":"33333333-3333-3333-3333-333333333333",
			            "target_id":"44444444-4444-4444-4444-444444444444","truth":"it opens","visible":true}],
			"attribute_writes": [
				{"target_id":"44444444-4444-4444-4444-444444444444","attribute":"open","value":true,"tier":1}
			]
		}
	}`
	if _, err := DecodeAndValidateRulingV2(base); err != nil {
		t.Fatalf("the attribute_write fixture must be valid first: %v", err)
	}

	cases := []struct{ name, from, to, wantIn string }{
		{"missing target_id", `"target_id":"44444444-4444-4444-4444-444444444444","attribute":"open"`, `"attribute":"open"`, "target_id"},
		{"missing attribute", `"attribute":"open",`, ``, "attribute"},
		{"missing value", `"value":true,`, ``, "value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bad := strings.Replace(base, c.from, c.to, 1)
			if bad == base {
				t.Fatalf("the mutation for %q did not change the fixture — this test would pass vacuously", c.name)
			}
			_, err := DecodeAndValidateRulingV2(bad)
			if err == nil {
				t.Fatalf("attribute_write %s was ACCEPTED", c.name)
			}
			if !strings.Contains(err.Error(), c.wantIn) {
				t.Fatalf("rejected, but not by the %s guard — got %q", c.name, err)
			}
		})
	}
}
