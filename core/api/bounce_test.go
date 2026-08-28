package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// SPEC-037 — a typed sentence the decompose stage made nothing of must cost the player NOTHING, say
// so, and be counted. Before this, an empty chain was indistinguishable from a continue press, so the
// engine spent the stillness floor, ran the world's turn, and reported `completed`: the sentence was
// discarded in silence and the turn was spent on it.
//
// Every test here reddens against the pre-fix code. Revert the `len(chain) == 0` branch in RunBeat and
// the first three fail; revert recordBeatDerivation and the last two fail.

// TestRunBeat_TypedSentenceThatParsedToNothingCostsNothing is the fix itself. The sibling test
// TestRunBeat_EmptyBeatAdvancesByInstantFloor pins the floor at 2 s on the continue-press path, so
// TicksAdvanced == 0 here is a positive proof the floor branch never ran — and the world's turn is
// inside that same branch, so it did not run either. The canon_event count is belt and braces.
func TestRunBeat_TypedSentenceThatParsedToNothingCostsNothing(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	var before int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND in_world_tick >= $2`, dlWorldID, baseTick).Scan(&before); err != nil {
		t.Fatalf("count canon_event before: %v", err)
	}

	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, []Attempt{}, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}

	if out.HaltReason != "bounce" {
		t.Fatalf("HaltReason = %q, want bounce — a typed sentence that bound nothing must say so, not report success", out.HaltReason)
	}
	if out.TicksAdvanced != 0 {
		t.Fatalf("TicksAdvanced = %d, want 0 — the player was not understood and must not be charged a moment for it", out.TicksAdvanced)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("Committed = %v, want nothing committed", out.Committed)
	}

	var after int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND in_world_tick >= $2`, dlWorldID, baseTick).Scan(&after); err != nil {
		t.Fatalf("count canon_event after: %v", err)
	}
	if after != before {
		t.Fatalf("canon_event count %d -> %d: the world took a turn on a sentence nobody understood", before, after)
	}
}

// TestRunBeat_TypedNothingLeavesAnActiveJourneyUntouched. Typing IS taking back decision priority, so
// a typed beat ends a journey — but only when the sentence actually said something. A sentence that
// bound nothing said nothing, so it must not end the journey either. Nothing happened at all.
func TestRunBeat_TypedNothingLeavesAnActiveJourneyUntouched(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	// A deliberate wait is a REAL attempt carrying a sustain shape — it is not an empty chain, which
	// is the whole reason waiting survives this change untouched.
	waitAttempt := Attempt{
		Type:     "AttributeChanged",
		Stated:   "I wait quietly at the bar",
		TargetID: dlKadeID,
		Sustain:  &Sustain{Kind: "for", Seconds: 1000},
	}
	j, err := orc.startJourney(ctx, dlWorldID, dlKadeID, waitAttempt, baseTick)
	if err != nil {
		t.Fatalf("startJourney: %v", err)
	}
	t.Cleanup(func() { jrDeleteJourney(t, context.Background(), pool, j.ID) })

	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, []Attempt{}, j.CurrentTick+1, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "bounce" {
		t.Fatalf("HaltReason = %q, want bounce", out.HaltReason)
	}

	var status string
	var legsDone int
	if err := pool.QueryRow(ctx,
		`SELECT status, legs_done FROM journey WHERE journey_id=$1::uuid`, j.ID).Scan(&status, &legsDone); err != nil {
		t.Fatalf("read journey: %v", err)
	}
	if status != "active" {
		t.Fatalf("journey status = %q, want active — a sentence that bound nothing must not kill a journey", status)
	}
	if legsDone != 0 {
		t.Fatalf("legs_done = %d, want 0 — nothing was understood, so nothing should have advanced", legsDone)
	}
}

// The "a QUERY-only beat still pays the instant floor" guard — which proves this change is narrow —
// lives in orchestrator_worldtime_test.go as TestRunBeat_QueryOnlyBeatAdvancesByInstantFloor, beside
// the floor it guards. One home (D-6); its doc comment records both jobs it does.

// The continue press had its own guard here — TestRunContinuePress_WithNoJourneyDoesNothing, proving
// that pressing it with no journey spent no world time. Both the test and the operation it guarded
// were deleted on 2026-08-28, hours after they were written: journeys now run their own legs
// (runJourneyToCompletion), so no journey is ever left waiting between beats and there is nothing a
// press could advance. Founder: *"there should be no continue button when you cannot continue... 'you
// are being attacked -> continue' fuck no."* The button is gone rather than disabled.

// TestRecordBeatDerivation_RecordsTheEmptyParseWithTheSentence is the measurement, and the empty row
// is the entire point: an unparsed sentence leaves no trace anywhere else in the system. transcript
// writes nothing when a beat produced no prose, and the debug trace builds its element list FROM the
// chain, so an empty chain leaves an empty trace even with debug on.
func TestRecordBeatDerivation_RecordsTheEmptyParseWithTheSentence(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	tick := wtBaseTick(t, ctx, pool)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM beat_derivation WHERE world_id=$1::uuid AND in_world_tick=$2`, dlWorldID, tick)
	})

	const sentence = "I hand the drowned lantern to whoever is listening"
	recordBeatDerivation(ctx, pool, dlWorldID, dlKadeID, tick, sentence, []Attempt{})

	var stated string
	var elements []byte
	if err := pool.QueryRow(ctx,
		`SELECT stated, elements FROM beat_derivation
		   WHERE world_id=$1::uuid AND in_world_tick=$2 ORDER BY derivation_id DESC LIMIT 1`,
		dlWorldID, tick).Scan(&stated, &elements); err != nil {
		t.Fatalf("read derivation: %v — the empty parse must still be recorded", err)
	}
	if stated != sentence {
		t.Fatalf("stated = %q, want the player's own sentence back verbatim", stated)
	}
	var got []TraceElement
	if err := json.Unmarshal(elements, &got); err != nil {
		t.Fatalf("elements is not a JSON array: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("elements = %v, want an empty array (the parse produced nothing)", got)
	}
}

// TestRecordBeatDerivation_RecordsWhatTheParseProduced is the other half: when the parse DID produce
// something, the row carries the type and the bound ids, so the distribution across action types —
// and the UNRESOLVED rate — is a query.
func TestRecordBeatDerivation_RecordsWhatTheParseProduced(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	tick := wtBaseTick(t, ctx, pool)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM beat_derivation WHERE world_id=$1::uuid AND in_world_tick=$2`, dlWorldID, tick)
	})

	chain := []Attempt{{
		Type:       "Communicated",
		Stated:     "I greet Mara",
		ListenerID: wtMaraID,
		Content:    "Evening.",
	}}
	recordBeatDerivation(ctx, pool, dlWorldID, dlKadeID, tick, "I greet Mara", chain)

	var elements []byte
	if err := pool.QueryRow(ctx,
		`SELECT elements FROM beat_derivation
		   WHERE world_id=$1::uuid AND in_world_tick=$2 ORDER BY derivation_id DESC LIMIT 1`,
		dlWorldID, tick).Scan(&elements); err != nil {
		t.Fatalf("read derivation: %v", err)
	}
	var got []TraceElement
	if err := json.Unmarshal(elements, &got); err != nil {
		t.Fatalf("elements is not a JSON array: %v", err)
	}
	if len(got) != 1 || got[0].Type != "Communicated" {
		t.Fatalf("elements = %+v, want one Communicated element", got)
	}
	if len(got[0].IDs) != 1 || got[0].IDs[0] != wtMaraID {
		t.Fatalf("bound ids = %v, want [%s]", got[0].IDs, wtMaraID)
	}
}

// TestBeats_TypedSentenceThatBoundNothingReturnsBounceOnTheWire is the same fix proved END TO END,
// through the real HTTP route and the real SSE frames — the layer the player actually meets.
//
// The fake decompose driver "can ONLY return schema-valid chains (from its table) or an empty chain"
// (bridge_fakes.go). With an empty table it returns the empty chain for any input, which is exactly
// the case under test: the player typed, and the parse bound nothing.
func TestBeats_TypedSentenceThatBoundNothingReturnsBounceOnTheWire(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	wtDisableWorldActor(t, ctx, pool, dlWorldID)

	// Anchor on the table's OWN id, never on a tick. The handler picks its start tick from
	// fn_world_now, and wtBaseTick's 95000 floor is a local convention that a fresh CI database does
	// not share — anchoring on it made this test pass on a developer's seeded box and fail in CI on
	// the identical code (2026-08-28).
	var beforeID int64
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(max(derivation_id), 0) FROM beat_derivation`).Scan(&beforeID); err != nil {
		t.Fatalf("read the derivation high-water mark: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM beat_derivation WHERE derivation_id > $1`, beforeID)
	})

	h := NewBeatsStreamHandler(pool, true, fakeSeatBridge(t, nil))

	const sentence = "I hand the drowned lantern to whoever is listening"
	req := httptest.NewRequest(http.MethodPost, "/worlds/"+dlWorldID+"/beats?viewer="+dlKadeID,
		strings.NewReader(`{"text":"`+sentence+`"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var got resultBlock
	sr := newSSEReader(bytes.NewReader(rec.Body.Bytes()))
	for {
		raw, err := sr.nextRaw()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading frame: %v", err)
		}
		frame := assertValidBeatFrame(t, raw)
		if frame["kind"] == "result" {
			var f resultFrame
			if err := json.Unmarshal(raw, &f); err != nil {
				t.Fatalf("decode result frame: %v", err)
			}
			got = f.Result
		}
	}

	if got.HaltReason != "bounce" {
		t.Fatalf("halt_reason = %q, want bounce — the surface renders this as \"Nothing came of that. No time passed.\"", got.HaltReason)
	}
	if got.TicksAdvanced != 0 {
		t.Fatalf("ticks_advanced = %d, want 0 — QA measured 2 here on 2026-08-11, which is the whole defect", got.TicksAdvanced)
	}

	// The sentence must have been counted, empty parse and all: that record is the reason this round
	// exists, and it is the only place the sentence survives at all.
	var stated string
	var elements []byte
	if err := pool.QueryRow(ctx,
		`SELECT stated, elements FROM beat_derivation
		   WHERE derivation_id > $1 AND world_id=$2::uuid ORDER BY derivation_id DESC LIMIT 1`,
		beforeID, dlWorldID).Scan(&stated, &elements); err != nil {
		t.Fatalf("no derivation row for the beat: %v", err)
	}
	if stated != sentence {
		t.Fatalf("recorded sentence = %q, want %q", stated, sentence)
	}
	if string(elements) != "[]" {
		t.Fatalf("recorded elements = %s, want [] (the parse produced nothing)", elements)
	}
}
