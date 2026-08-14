package main

import (
	"context"
	"testing"
)

// The sink is the engine's own accounting, so it must be exactly right about three things: what one
// beat cost, what the process has spent, and that a call outside a beat cannot panic the server.
func TestCostSink_TotalsPerBeatAndPerSession(t *testing.T) {
	sessionBefore := sessionTotalUSD()

	ctx, beat := withCostSink(context.Background())
	// Two seats, as a real beat: a cheap classification and an expensive narration.
	costSinkFrom(ctx).add(0.00071, 4910, 75, 0)
	costSinkFrom(ctx).add(0.00565, 4340, 250, 3000)

	usd, in, out, cached, calls := beat.snapshot()
	if calls != 2 || in != 9250 || out != 325 || cached != 3000 {
		t.Fatalf("snapshot = %d calls, %d in, %d out, %d cached; want 2/9250/325/3000", calls, in, out, cached)
	}
	if usd < 0.006359 || usd > 0.006361 {
		t.Fatalf("beat cost = %.6f, want 0.00636", usd)
	}
	// The process total moves by the same amount — that is what makes a long session watchable.
	if got := sessionTotalUSD() - sessionBefore; got < 0.006359 || got > 0.006361 {
		t.Fatalf("session delta = %.6f, want the beat's 0.00636", got)
	}

	// A second beat is independent, and the session keeps accumulating.
	_, beat2 := withCostSink(context.Background())
	if usd2, _, _, _, calls2 := beat2.snapshot(); usd2 != 0 || calls2 != 0 {
		t.Fatalf("a fresh beat must start at zero, got %.6f over %d calls", usd2, calls2)
	}
}

// A driver used outside a beat (every direct unit call, and the seat-config tests) has no sink in its
// context. That must be inert, not a nil dereference on the server's hot path.
func TestCostSink_NilContextIsInert(t *testing.T) {
	costSinkFrom(context.Background()).add(1.23, 10, 10, 0) // must not panic
	var none *costSink
	if usd, in, out, cached, calls := none.snapshot(); usd != 0 || in != 0 || out != 0 || cached != 0 || calls != 0 {
		t.Fatalf("nil sink must read as zero, got %.4f/%d/%d/%d/%d", usd, in, out, cached, calls)
	}
}
