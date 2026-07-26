package main

// Seat-boundary prompt assembly — the founder-gate fix. The decompose and narrate drivers drop
// req.Payload (they send only req.Prompt), so the perception lines and the candidate whitelist never
// reached the live model: decompose could not bind a real id, and the narrator, handed the bare
// instruction, invented a scene and broke frame. These tests pin the fix at the seat boundary:
//
//   (a) the DECOMPOSE seat's Prompt carries a candidate's uuid AND name AND the player's raw text,
//       with the raw text positioned AFTER the candidates (the mutable tail) — through the REAL HTTP
//       handler with a capturing driver (wall_test.go / station_e_exit_test.go pattern).
//   (b) the NARRATE seat's Prompt carries a real perception line's content AND the anti-invention
//       header marker — again through the real handler.
//   (c) a UNIT test: the narrate builder with ZERO lines still carries the sparse-is-correct rule and
//       fabricates no content section.
//
// (a) and (b) reuse setupWallWorld + runWallBeat: a fresh world with player K co-located with Mara
// (M) and Jonas (J) at L, so the decompose payload has a real candidate whitelist and the narrate
// payload has real perception lines.

import (
	"context"
	"strings"
	"testing"
)

// (a) The decompose seat's Prompt carries a candidate uuid + name + the raw player text, text last.
func TestSeatPrompt_DecomposeCarriesCandidatesAndPlayerText(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true)

	// Benign beat: the raw text names Mara but binds her via the candidate whitelist, not the input.
	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`

	seats := runWallBeat(t, ctx, pool, id, beatText, chain)
	dec := seats[SeatDecompose.Name]
	if len(dec.reqs) == 0 {
		t.Fatalf("(a) decompose seat was never called")
	}
	p := dec.reqs[0].Prompt

	// Mara is a present candidate: BOTH her uuid and her name must reach the seat (so a real id binds).
	mIdx := strings.Index(p, id.M)
	if mIdx < 0 {
		t.Fatalf("(a) decompose Prompt missing candidate uuid %s — the whitelist did not reach the seat\n%s", id.M, p)
	}
	if !strings.Contains(p, "Mara") {
		t.Fatalf("(a) decompose Prompt missing candidate name Mara\n%s", p)
	}

	// The raw player text is present AND sits AFTER the candidate block (the mutable tail): the
	// cache-native layout the header promises, and what lets the model read the scene before the input.
	tIdx := strings.Index(p, beatText)
	if tIdx < 0 {
		t.Fatalf("(a) decompose Prompt missing the raw player text %q\n%s", beatText, p)
	}
	if tIdx < mIdx {
		t.Fatalf("(a) player text (idx %d) must sit AFTER the candidates (candidate uuid idx %d) — the tail\n%s", tIdx, mIdx, p)
	}

	// Keep SQL test 14's every-perception-has-a-subject invariant satisfied for this world's commits.
	perceptionSubjectBackfill(t, ctx, pool, 60000)
}

// (b) The narrate seat's Prompt carries a real perception line AND the anti-invention header marker.
func TestSeatPrompt_NarrateCarriesPerceptionsAndAntiInvention(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true)

	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`

	seats := runWallBeat(t, ctx, pool, id, beatText, chain)
	nar := seats[SeatNarrate.Name]
	if len(nar.reqs) == 0 {
		t.Fatalf("(b) narrate seat was never called")
	}
	req := nar.reqs[0]
	p := req.Prompt

	// The bounding header reached the model (it did NOT before the fix — the driver dropped everything
	// but the bare instruction, so the narrator invented a scene and offered out-of-world frames).
	if !strings.Contains(p, narrateAntiInventionMarker) {
		t.Fatalf("(b) narrate Prompt missing anti-invention marker %q — the narrator is not bounded\n%s", narrateAntiInventionMarker, p)
	}

	// The payload the seat actually received (its perception lines) must now appear IN the prompt — the
	// exact vector the driver used to drop. Assert on a line the seat truly held (known at runtime).
	if len(req.Payload.Lines) == 0 {
		t.Fatalf("(b) narrate payload had zero perception lines — the fixture is too sparse to prove the fix")
	}
	line := req.Payload.Lines[0]
	if !strings.Contains(p, line) {
		t.Fatalf("(b) narrate Prompt missing a real perception line %q — the payload never reached the model\n%s", line, p)
	}

	perceptionSubjectBackfill(t, ctx, pool, 60000)
}

// (c) Unit: the narrate builder with ZERO lines still carries the sparse-is-correct rule and
// fabricates no content — a sparse moment is a short narration, per the header.
func TestBuildNarratePrompt_ZeroLinesSparseNoFabrication(t *testing.T) {
	p := buildNarratePrompt(PerceptionPayload{}) // no lines, no candidates

	if !strings.HasPrefix(p, narrateSystemHeader) {
		t.Fatalf("(c) narrate prompt must open with the bounding header (the stable cache prefix)")
	}
	if !strings.Contains(p, narrateSparseRuleMarker) {
		t.Fatalf("(c) zero-line narrate prompt missing the sparse-is-correct rule %q", narrateSparseRuleMarker)
	}
	// No fabricated content section: with no candidates and no lines, no bullet/roster body is emitted.
	if strings.Contains(p, "\n- ") {
		t.Fatalf("(c) zero-line narrate prompt fabricated a content bullet — nothing must be invented to fill the space:\n%s", p)
	}
}
