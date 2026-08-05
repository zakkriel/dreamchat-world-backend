package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Task 4 — the scheduled-events ledger. fireDuePending fires every pending_event row whose
// fire_at_tick falls in a clock-crossing window (tickBefore, tickAfter], commits each payload
// through the normal pipeline (applyEvent/adjudicate — the SAME routing runChain's Stage 3 uses),
// and flips the row to 'fired'. It is a standalone helper: NOT yet wired into the beat (Task 9's
// composer does that), so it is tested directly here against the real seeded Drowned Lantern play
// world (make reset precedes go test in the battery — matches orchestrator_worldtime_test.go's
// wtOrchestrator/dlWorldID/dlKadeID pattern).

// lgInsertPending inserts a pending_event row for worldID and returns its pending_id. The payload is
// the {"actor_id":..., "attempt":{...}} shape Task 4 establishes (ambiguity resolution #1 — the world
// entity acting, paired with its Attempt JSON) so Task 8 (World Actor) can schedule world truth ahead
// of time in the same shape.
func lgInsertPending(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string, fireAtTick int64, magnitude, actorID, attemptJSON string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO pending_event (pending_id, world_id, fire_at_tick, magnitude, payload, status)
		 VALUES (gen_random_uuid(), $1, $2, $3,
		         jsonb_build_object('actor_id', $4::uuid, 'attempt', $5::jsonb),
		         'pending')
		 RETURNING pending_id::text`,
		worldID, fireAtTick, magnitude, actorID, attemptJSON).Scan(&id); err != nil {
		t.Fatalf("lgInsertPending: %v", err)
	}
	return id
}

// lgDeletePending removes a pending_event row (test cleanup — keeps the table free of leaked rows
// across repeated `go test` runs against the same seeded DB).
func lgDeletePending(t *testing.T, ctx context.Context, pool *pgxpool.Pool, pendingID string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DELETE FROM pending_event WHERE pending_id=$1`, pendingID); err != nil {
		t.Fatalf("lgDeletePending: %v", err)
	}
}

// lgPendingStatus reads back a pending_event row's status.
func lgPendingStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, pendingID string) string {
	t.Helper()
	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM pending_event WHERE pending_id=$1`, pendingID).Scan(&status); err != nil {
		t.Fatalf("lgPendingStatus: %v", err)
	}
	return status
}

// lgCanonCount counts accepted canon_event rows for worldID (the "did it actually commit" check).
func lgCanonCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND status='accepted'`, worldID).Scan(&n); err != nil {
		t.Fatalf("lgCanonCount: %v", err)
	}
	return n
}

// TestFireDuePending_CrossingFires: a pending_event at fire_at_tick=baseTick (magnitude medium) sits
// inside the crossing window (baseTick-1, baseTick+2] — strict lower bound, inclusive upper (ambiguity
// resolution #3), the same (tickBefore, fire_at_tick, tickAfter) shape as the brief's (9,10,12) example
// — so it fires: the Communicated payload commits through applyEvent (co-located actor=Mara →
// listener=Kade, the passthrough path runChain's Stage 3 uses for Communicated), canon count goes up
// by one, the row flips to 'fired', and the returned magnitude is "medium".
//
// baseTick comes from wtBaseTick (orchestrator_worldtime_test.go), not a hardcoded literal: fireDuePending
// commits at tickAfter (this helper's own design — the row's fire_at_tick is only ever the WHERE-clause
// cutoff, never the commit tick), and canon_event's uq_ce_accepted_order unique index means a fixed
// literal tick would collide with itself on a second run within the same seeded DB (e.g. the full
// `go test ./core/api/...` battery re-invoking this test without an intervening `make reset`) — the
// dynamic baseTick is what makes this test re-runnable, matching wtBaseTick's own stated purpose.
func TestFireDuePending_CrossingFires(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with lgDeletePending's own t.Cleanup below, so the pending-row
	// delete runs BEFORE the pool closes (a plain `defer pool.Close()` would run first, since defers
	// fire on function return, before the testing framework's own Cleanup queue unwinds).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)

	baseTick := wtBaseTick(t, ctx, pool)
	attempt := `{"type":"Communicated","stated":"the bell tolls","listener_id":"` + dlKadeID + `","content":"a bell tolls over the docks"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick, "medium", wtMaraID, attempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := lgCanonCount(t, ctx, pool, dlWorldID)
	var out BeatOutcome

	mag, err := orc.fireDuePending(ctx, dlWorldID, baseTick-1, baseTick+2, 0, &out, nil)
	if err != nil {
		t.Fatalf("fireDuePending: %v", err)
	}
	if mag != "medium" {
		t.Fatalf("fired mag = %q, want medium", mag)
	}
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != before+1 {
		t.Fatalf("canon count = %d, want %d (payload not committed)", got, before+1)
	}
	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired", status)
	}
	if len(out.Committed) != 1 {
		t.Fatalf("outcome.Committed = %v, want exactly one committed id", out.Committed)
	}
}

// TestFireDuePending_BeforeWindowFiresNothing: the SAME shape of pending_event (fire_at_tick=baseTick)
// is NOT due for a crossing window ending before it (baseTick-100, baseTick-1] — fire_at_tick is above
// the inclusive upper bound — so nothing fires: empty magnitude, the row stays 'pending', and canon
// count is untouched. A fresh baseTick (re-read from wtBaseTick, so strictly above the first test's
// committed tick) keeps this independent of TestFireDuePending_CrossingFires's own row.
func TestFireDuePending_BeforeWindowFiresNothing(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)

	baseTick := wtBaseTick(t, ctx, pool)
	attempt := `{"type":"Communicated","stated":"the bell tolls","listener_id":"` + dlKadeID + `","content":"a bell tolls over the docks"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick, "medium", wtMaraID, attempt)
	// This row is deliberately left 'pending' by the assertions below (nothing fires it here) — clean
	// it up so it can never be picked up by a LATER test's freshly-computed wtBaseTick window on a
	// re-run without `make reset` (wtBaseTick only looks at canon_event, not pending_event, so a
	// leaked pending row can land inside a future crossing window and fire unexpectedly).
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := lgCanonCount(t, ctx, pool, dlWorldID)
	var out BeatOutcome

	mag, err := orc.fireDuePending(ctx, dlWorldID, baseTick-100, baseTick-1, 0, &out, nil)
	if err != nil {
		t.Fatalf("fireDuePending: %v", err)
	}
	if mag != "" {
		t.Fatalf("fired mag = %q, want \"\" (nothing due before fire_at_tick)", mag)
	}
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != before {
		t.Fatalf("canon count = %d, want unchanged %d", got, before)
	}
	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "pending" {
		t.Fatalf("pending_event status = %q, want still pending", status)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("outcome.Committed = %v, want empty", out.Committed)
	}
}

// TestFireDuePending_AdjudicatedTypeCommits: review fix (Important #3a). A pending row whose Attempt
// is an ADJUDICATED type (AttributeChanged — none of the three passthrough types) must route through
// o.adjudicate (the Stage-3 default branch), and on a successful ruling behaves exactly like the
// passthrough case: canon count +1, row 'fired', magnitude folded. The driver is a fixed
// inlineRulingDriver (defined in orchestrator_ruled_test.go, same package) rather than
// wtOrchestrator's fakeResolveDriver, so the ruling's actor_id/target_id are deterministic and known
// to satisfy verdictRuling's whitelist (they equal the single ActorAttempt's own actor/target ids,
// which adjudicate always seeds into sliceIDs regardless of what gather_slice returns).
func TestFireDuePending_AdjudicatedTypeCommits(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	baseTick := wtBaseTick(t, ctx, pool)

	rulingJSON := `{"reasoning":"The bell's echo rattles the bar counter.","therefore":"succeeds","outcome":{"kind":"resolved","events":[` +
		`{"type":"AttributeChanged","actor_id":"` + wtMaraID + `","target_id":"` + dlBarID + `","truth":"The bar rattles faintly as the bell tolls."}` +
		`]}}`
	orc := &Orchestrator{
		DB:                pool,
		Resolve:           &inlineRulingDriver{name: "ledger-adjudicated-commit", ruling: rulingJSON},
		CognitionBatch:    NewFakeCognitionDriver(),
		CognitionIsolated: NewFakeCognitionDriver(),
		WorldActor:        NewFakeWorldActorDriver(),
	}

	attempt := `{"type":"AttributeChanged","stated":"the bell rattles the bar","target_id":"` + dlBarID + `"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick, "large", wtMaraID, attempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := lgCanonCount(t, ctx, pool, dlWorldID)
	var out BeatOutcome

	mag, err := orc.fireDuePending(ctx, dlWorldID, baseTick-1, baseTick+2, 0, &out, nil)
	if err != nil {
		t.Fatalf("fireDuePending: %v", err)
	}
	if mag != "large" {
		t.Fatalf("fired mag = %q, want large", mag)
	}
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != before+1 {
		t.Fatalf("canon count = %d, want %d (adjudicated payload not committed)", got, before+1)
	}
	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired", status)
	}
	if len(out.Committed) != 1 {
		t.Fatalf("outcome.Committed = %v, want exactly one committed id", out.Committed)
	}
}

// TestFireDuePending_GateRejectCancelsRow: review fix (Critical #1 + Important #3b). A pending row
// whose Communicated payload targets a listener who is NOT co-located with the actor fails
// apply_event's structural floor (gate_reject) — the SAME check runChain's own passthrough branch
// honors. Before the review fix, fireDuePending folded ANY row into 'fired' + magnitude regardless of
// whether anything actually committed; this test directly guards that fix: the row must land in the
// terminal 'cancelled' state (not 'fired', not left 'pending' for an endless retry), the returned
// magnitude must exclude it ("" — nothing else is due), canon count must be untouched, and
// outcome.Committed must stay empty.
func TestFireDuePending_GateRejectCancelsRow(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)

	baseTick := wtBaseTick(t, ctx, pool)
	// A listener uuid that is guaranteed NOT co-located with wtMaraID (it doesn't exist in
	// entity_registry at all) — fn_actors_at(Mara's location) will never contain it.
	const nowhereListener = "99999999-9999-9999-9999-999999999999"
	attempt := `{"type":"Communicated","stated":"a message meant for someone not here","listener_id":"` + nowhereListener + `","content":"can anyone hear me?"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick, "large", wtMaraID, attempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := lgCanonCount(t, ctx, pool, dlWorldID)
	var out BeatOutcome

	mag, err := orc.fireDuePending(ctx, dlWorldID, baseTick-1, baseTick+2, 0, &out, nil)
	if err != nil {
		t.Fatalf("fireDuePending: %v", err)
	}
	if mag != "" {
		t.Fatalf("fired mag = %q, want \"\" (the only due row gate-rejected, so nothing actually fired)", mag)
	}
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != before {
		t.Fatalf("canon count = %d, want unchanged %d (a gate-rejected payload must not commit canon)", got, before)
	}
	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "cancelled" {
		t.Fatalf("pending_event status = %q, want cancelled", status)
	}
	if len(out.Committed) != 0 {
		t.Fatalf("outcome.Committed = %v, want empty", out.Committed)
	}
}

// TestFireDuePending_AtTickBeforeDoesNotFire: review fix (Minor #5). A row at fire_at_tick EXACTLY
// equal to tickBefore is outside the window — the lower bound is strict (>) — so it must not fire.
func TestFireDuePending_AtTickBeforeDoesNotFire(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)

	baseTick := wtBaseTick(t, ctx, pool)
	tickBefore := baseTick
	tickAfter := baseTick + 3

	attempt := `{"type":"Communicated","stated":"the bell tolls","listener_id":"` + dlKadeID + `","content":"a bell tolls over the docks"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, tickBefore, "small", wtMaraID, attempt) // fire_at_tick == tickBefore
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := lgCanonCount(t, ctx, pool, dlWorldID)
	var out BeatOutcome

	mag, err := orc.fireDuePending(ctx, dlWorldID, tickBefore, tickAfter, 0, &out, nil)
	if err != nil {
		t.Fatalf("fireDuePending: %v", err)
	}
	if mag != "" {
		t.Fatalf("fired mag = %q, want \"\" (fire_at_tick == tickBefore must NOT fire — strict lower bound)", mag)
	}
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != before {
		t.Fatalf("canon count = %d, want unchanged %d", got, before)
	}
	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "pending" {
		t.Fatalf("pending_event status = %q, want still pending", status)
	}
}

// TestFireDuePending_AtTickAfterFires: review fix (Minor #5). A row at fire_at_tick EXACTLY equal to
// tickAfter IS inside the window — the upper bound is inclusive (<=) — so it must fire.
func TestFireDuePending_AtTickAfterFires(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := wtOrchestrator(pool)

	baseTick := wtBaseTick(t, ctx, pool)
	tickBefore := baseTick
	tickAfter := baseTick + 3

	attempt := `{"type":"Communicated","stated":"the bell tolls","listener_id":"` + dlKadeID + `","content":"a bell tolls over the docks"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, tickAfter, "small", wtMaraID, attempt) // fire_at_tick == tickAfter
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	before := lgCanonCount(t, ctx, pool, dlWorldID)
	var out BeatOutcome

	mag, err := orc.fireDuePending(ctx, dlWorldID, tickBefore, tickAfter, 0, &out, nil)
	if err != nil {
		t.Fatalf("fireDuePending: %v", err)
	}
	if mag != "small" {
		t.Fatalf("fired mag = %q, want small (fire_at_tick == tickAfter must fire — inclusive upper bound)", mag)
	}
	if got := lgCanonCount(t, ctx, pool, dlWorldID); got != before+1 {
		t.Fatalf("canon count = %d, want %d", got, before+1)
	}
	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired", status)
	}
}
