package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Living World / Task 9 (Unit 6) — the world's-turn composer, wired into runChain's Stage-4 clock
// advance (both the passthrough and adjudicated branches). Exercised against the real seeded Drowned
// Lantern play world (`make reset` precedes go test in the battery — matches ledger_test.go's/
// pressure_test.go's/worldactor_test.go's wtOrchestrator/dlWorldID/wtMaraID pattern).
//
// The roll is deterministic (pressure.go's deterministicUnit — a pure hash, never math/rand), so these
// tests FORCE a fire via world_actor_config (task-9-brief Test resolution): climb_rate=1,
// climb_chunk_ticks=1, cap=1 saturates a tier's chance to exactly 1.0 for any tick past its last
// eruption, and rollTier's roll is always < 1.0 (deterministicUnit's range is [0,1)) — so that tier
// fires with certainty. The other tiers are pinned to climb_rate=0/cap=0 (fn_pressure_chance's outer
// COALESCE guarantees exactly 0 for that shape — TestRollTier_ChanceZeroNeverFires, pressure_test.go),
// so they never fire.

// wtSnapshotTierConfig reads worldID's current world_actor_config (all three tiers) and returns a
// restore func that puts those exact rows back — register it with t.Cleanup (NOT a plain defer; the
// caller must also register its pool.Close via t.Cleanup so LIFO runs this restore BEFORE the pool
// closes, mirroring ledger_test.go's own documented t.Cleanup-ordering pattern).
func wtSnapshotTierConfig(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string) func() {
	t.Helper()
	type row struct {
		tier            string
		climbRate, cap  float64
		climbChunkTicks int64
	}
	rows, err := pool.Query(ctx,
		`SELECT tier, climb_rate, climb_chunk_ticks, cap FROM world_actor_config WHERE world_id=$1`, worldID)
	if err != nil {
		t.Fatalf("wtSnapshotTierConfig: %v", err)
	}
	var saved []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.tier, &r.climbRate, &r.climbChunkTicks, &r.cap); err != nil {
			rows.Close()
			t.Fatalf("wtSnapshotTierConfig scan: %v", err)
		}
		saved = append(saved, r)
	}
	rows.Close()
	return func() {
		for _, r := range saved {
			if _, err := pool.Exec(context.Background(),
				`UPDATE world_actor_config SET climb_rate=$1, climb_chunk_ticks=$2, cap=$3 WHERE world_id=$4 AND tier=$5`,
				r.climbRate, r.climbChunkTicks, r.cap, worldID, r.tier); err != nil {
				t.Errorf("wtSnapshotTierConfig restore(%s): %v", r.tier, err)
			}
		}
	}
}

// wtForceTierFires mutates dlWorldID's SHARED world_actor_config (seeded once by seed_world_defaults,
// PK (world_id,tier)) so `tier` is guaranteed to fire and the other two tiers never do, and registers a
// t.Cleanup to put the original values back — so this shared config never leaks into a LATER test (in
// this file or another) or a re-run of `go test` without an intervening `make reset`
// (pressure_test.go's TestRollTier_FiredMatchesRollLessThanChance depends on dlWorldID's DEFAULT seeded
// small-tier config, 0.01 climb_rate / 0.70 cap).
func wtForceTierFires(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, firingTier string) {
	t.Helper()
	t.Cleanup(wtSnapshotTierConfig(t, ctx, pool, worldID))
	for _, tr := range []string{"small", "medium", "large"} {
		climbRate, cap, chunk := 0.0, 0.0, int64(60)
		if tr == firingTier {
			climbRate, cap, chunk = 1.0, 1.0, 1
		}
		if _, err := pool.Exec(ctx,
			`UPDATE world_actor_config SET climb_rate=$1, climb_chunk_ticks=$2, cap=$3 WHERE world_id=$4 AND tier=$5`,
			climbRate, chunk, cap, worldID, tr); err != nil {
			t.Fatalf("wtForceTierFires(%s): %v", firingTier, err)
		}
	}
}

// wtForceAllTiersFire saturates EVERY tier's chance to 1.0 — used by the ledger-skips-the-roll test (c)
// so a passing assertion ("nothing new in world_eruption") proves the roll was genuinely SKIPPED, not
// just that pressure happened to be off. Restores the original config on cleanup, same as
// wtForceTierFires above.
func wtForceAllTiersFire(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string) {
	t.Helper()
	t.Cleanup(wtSnapshotTierConfig(t, ctx, pool, worldID))
	for _, tr := range []string{"small", "medium", "large"} {
		if _, err := pool.Exec(ctx,
			`UPDATE world_actor_config SET climb_rate=1, climb_chunk_ticks=1, cap=1 WHERE world_id=$1 AND tier=$2`,
			worldID, tr); err != nil {
			t.Fatalf("wtForceAllTiersFire(%s): %v", tr, err)
		}
	}
}

// wtSpeakerEventCount counts how many of eventIDs were spoken/instigated by speakerID (role_qualifier
// 'speaker' for Communicated, 'instigator' otherwise — mirrors worldactor_test.go's waEventLocation
// query shape). Used to distinguish the acting PLAYER's own commits from the World Actor's (the fake
// always speaks as Mara — bridge_fakes.go).
func wtSpeakerEventCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, eventIDs []string, speakerID string) int {
	t.Helper()
	if len(eventIDs) == 0 {
		return 0
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM event_participant WHERE event_id = ANY($1) AND entity_id=$2 AND role_qualifier IN ('instigator','speaker')`,
		eventIDs, speakerID).Scan(&n); err != nil {
		t.Fatalf("wtSpeakerEventCount: %v", err)
	}
	return n
}

// wtEruptionRowForEvent reads back the world_eruption row (task-9 fire-log) matching one of eventIDs —
// proving a fire wrote (world_id, tier, fired_tick, event_id), test resolution (d). ok is false if no
// row in world_eruption references any of eventIDs.
func wtEruptionRowForEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string, eventIDs []string) (tier string, firedTick int64, eventID string, ok bool) {
	t.Helper()
	if len(eventIDs) == 0 {
		return "", 0, "", false
	}
	err := pool.QueryRow(ctx,
		`SELECT tier, fired_tick, event_id::text FROM world_eruption WHERE world_id=$1 AND event_id = ANY($2) LIMIT 1`,
		worldID, eventIDs).Scan(&tier, &firedTick, &eventID)
	if err != nil {
		return "", 0, "", false
	}
	return tier, firedTick, eventID, true
}

// wtEruptionCount counts every world_eruption row for (worldID,tier) — used as a before/after delta to
// prove the roll never ran (test c: a medium ledger fire must SKIP the roll entirely).
func wtEruptionCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, tier string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM world_eruption WHERE world_id=$1 AND tier=$2`, worldID, tier).Scan(&n); err != nil {
		t.Fatalf("wtEruptionCount: %v", err)
	}
	return n
}

// wtCanonSummaryExists reports whether a canon_event with this exact summary (apply_event stores
// p_attempt->>'stated' verbatim as canon_event.summary) exists for worldID — a marker-text check that a
// specific chain attempt did (or did not) land, independent of out.Committed's exact shape (which the
// world's turn can also append to).
func wtCanonSummaryExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID, summary string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM canon_event WHERE world_id=$1 AND summary=$2)`,
		worldID, summary).Scan(&exists); err != nil {
		t.Fatalf("wtCanonSummaryExists: %v", err)
	}
	return exists
}

// TestRunWorldTurn_ForcedSmallFireContinuesChain is the brief's Test resolution (a): with the small
// tier forced to fire (medium/large disabled), a normal two-attempt chain commits an EXTRA world event
// (the forced eruption's World Actor intrusion) and the chain CONTINUES — the following attempt still
// commits. Also covers resolution (d): the fire is recorded in world_eruption.
func TestRunWorldTurn_ForcedSmallFireContinuesChain(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with wtForceTierFires' own config-restore t.Cleanup, so the restore
	// runs BEFORE the pool closes (ledger_test.go's documented t.Cleanup-ordering pattern).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	wtForceTierFires(t, ctx, pool, dlWorldID, "small")

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	chain := []Attempt{
		{Type: "Communicated", Stated: "Kade greets Mara over the bar", ListenerID: wtMaraID, Content: "morning, Mara"},
		{Type: "Communicated", Stated: "Kade asks Mara for a second round", ListenerID: wtMaraID, Content: "another ale, please"},
	}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "completed" {
		t.Fatalf("HaltReason = %q, want completed (a forced SMALL fire must not halt the beat — §5 lets small run on)", out.HaltReason)
	}

	// Both of Kade's own attempts landed — the chain continued straight past the world's turn.
	if got := wtSpeakerEventCount(t, ctx, pool, out.Committed, dlKadeID); got != 2 {
		t.Fatalf("Kade-spoken committed events = %d, want 2 (BOTH player attempts must land — the chain continued)", got)
	}
	// An EXTRA event — the forced eruption's World Actor intrusion (the fake always speaks as Mara,
	// bridge_fakes.go) — also landed.
	if got := wtSpeakerEventCount(t, ctx, pool, out.Committed, wtMaraID); got < 1 {
		t.Fatalf("Mara-spoken committed events = %d, want >=1 (the forced small eruption's world-actor intrusion)", got)
	}

	// (d) the fire is recorded in the append-only fire-log.
	tier, firedTick, eventID, ok := wtEruptionRowForEvent(t, ctx, pool, dlWorldID, out.Committed)
	if !ok {
		t.Fatalf("no world_eruption row references any committed event id %v — the fire-log write is missing", out.Committed)
	}
	if tier != "small" {
		t.Fatalf("world_eruption.tier = %q, want small", tier)
	}
	if firedTick < baseTick {
		t.Fatalf("world_eruption.fired_tick = %d, want >= baseTick (%d)", firedTick, baseTick)
	}
	if eventID == "" {
		t.Fatalf("world_eruption.event_id is empty")
	}
}

// TestRunWorldTurn_ForcedMediumFireHaltsChain is the brief's Test resolution (b): with the medium tier
// forced to fire (small/large disabled), runChain/RunBeat halts with HaltReason == "world_eruption"
// after the CURRENT attempt, and a FOLLOWING attempt in the chain does NOT commit — the §5 cut. Also
// covers resolution (d) again, on the medium path.
func TestRunWorldTurn_ForcedMediumFireHaltsChain(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — see TestRunWorldTurn_ForcedSmallFireContinuesChain's comment above.
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	wtForceTierFires(t, ctx, pool, dlWorldID, "medium")

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	const neverLands = "Kade should never get to say this — the medium eruption already cut the beat"
	chain := []Attempt{
		{Type: "Communicated", Stated: "Kade raises a toast before the eruption", ListenerID: wtMaraID, Content: "to the house"},
		{Type: "Communicated", Stated: neverLands, ListenerID: wtMaraID, Content: "never lands"},
	}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "world_eruption" {
		t.Fatalf("HaltReason = %q, want world_eruption (a forced MEDIUM fire must apply the §5 cut)", out.HaltReason)
	}

	// Only the FIRST attempt landed.
	if got := wtSpeakerEventCount(t, ctx, pool, out.Committed, dlKadeID); got != 1 {
		t.Fatalf("Kade-spoken committed events = %d, want exactly 1 (only the first attempt commits)", got)
	}
	// Directly prove the second attempt never reached canon, independent of out.Committed's shape.
	if wtCanonSummaryExists(t, ctx, pool, dlWorldID, neverLands) {
		t.Fatalf("the SECOND chain attempt committed despite the medium eruption halt — §5 cut violated")
	}

	// (d) the fire is recorded in the append-only fire-log.
	tier, _, eventID, ok := wtEruptionRowForEvent(t, ctx, pool, dlWorldID, out.Committed)
	if !ok {
		t.Fatalf("no world_eruption row references any committed event id %v — the fire-log write is missing", out.Committed)
	}
	if tier != "medium" {
		t.Fatalf("world_eruption.tier = %q, want medium", tier)
	}
	if eventID == "" {
		t.Fatalf("world_eruption.event_id is empty")
	}
}

// TestRunWorldTurn_DueMediumPendingSkipsRoll is the brief's Test resolution (c): a pending_event at
// medium magnitude due inside a normal attempt's clock-crossing applies the SAME §5 cut (the ledger
// fires FIRST) and — critically — the pressure roll is SKIPPED entirely: even with every tier forced to
// certain-fire (wtForceAllTiersFire), world_eruption gains NO new row, proving the roll body never ran
// (ambiguity resolution #2a).
func TestRunWorldTurn_DueMediumPendingSkipsRoll(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — LIFO with lgDeletePending's own t.Cleanup below, so the pending-row delete
	// runs BEFORE the pool closes (ledger_test.go's own documented gotcha: a plain `defer pool.Close()`
	// would run first, since defers fire on function return, before the testing framework's Cleanup queue
	// unwinds).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	wtForceAllTiersFire(t, ctx, pool, dlWorldID)

	orc := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)

	// The first attempt (instant, 2s per the seeded duration_class_seconds) crosses to baseTick+2;
	// schedule the pending row exactly AT that tick — inside the crossing (tickBefore, tickAfter], the
	// SAME shape ledger_test.go's TestFireDuePending_AtTickAfterFires already established.
	pendingAttempt := `{"type":"Communicated","stated":"a commotion erupts from the cellar hatch","listener_id":"` + dlKadeID + `","content":"something below just broke loose"}`
	pendingID := lgInsertPending(t, ctx, pool, dlWorldID, baseTick+2, "medium", wtMaraID, pendingAttempt)
	t.Cleanup(func() { lgDeletePending(t, context.Background(), pool, pendingID) })

	beforeSmall := wtEruptionCount(t, ctx, pool, dlWorldID, "small")
	beforeMedium := wtEruptionCount(t, ctx, pool, dlWorldID, "medium")
	beforeLarge := wtEruptionCount(t, ctx, pool, dlWorldID, "large")

	const neverLands = "Kade should never get to say this — the scheduled medium event already cut the beat"
	chain := []Attempt{
		{Type: "Communicated", Stated: "Kade orders a round before the crossing", ListenerID: wtMaraID, Content: "one more round"},
		{Type: "Communicated", Stated: neverLands, ListenerID: wtMaraID, Content: "never lands"},
	}
	out, err := orc.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if out.HaltReason != "world_eruption" {
		t.Fatalf("HaltReason = %q, want world_eruption (the due medium pending_event applies the same §5 cut)", out.HaltReason)
	}
	if got := wtSpeakerEventCount(t, ctx, pool, out.Committed, dlKadeID); got != 1 {
		t.Fatalf("Kade-spoken committed events = %d, want exactly 1 (only the first attempt commits)", got)
	}
	if wtCanonSummaryExists(t, ctx, pool, dlWorldID, neverLands) {
		t.Fatalf("the SECOND chain attempt committed despite the ledger-fired halt — §5 cut violated")
	}
	if status := lgPendingStatus(t, ctx, pool, pendingID); status != "fired" {
		t.Fatalf("pending_event status = %q, want fired", status)
	}

	// The roll must never have run: world_eruption is untouched for EVERY tier, even though all three
	// were forced to certain-fire.
	if got := wtEruptionCount(t, ctx, pool, dlWorldID, "small"); got != beforeSmall {
		t.Fatalf("world_eruption 'small' count = %d, want unchanged %d — the roll must be skipped when the ledger already fired medium/large", got, beforeSmall)
	}
	if got := wtEruptionCount(t, ctx, pool, dlWorldID, "medium"); got != beforeMedium {
		t.Fatalf("world_eruption 'medium' count = %d, want unchanged %d — the roll must be skipped when the ledger already fired medium/large", got, beforeMedium)
	}
	if got := wtEruptionCount(t, ctx, pool, dlWorldID, "large"); got != beforeLarge {
		t.Fatalf("world_eruption 'large' count = %d, want unchanged %d — the roll must be skipped when the ledger already fired medium/large", got, beforeLarge)
	}
}

// TestRunWorldTurn_LastEruptionTick pins lastEruptionTick's own contract directly (task-9-brief's named
// helper): 0 for a tier with no prior fire, and the max fired_tick once one exists.
func TestRunWorldTurn_LastEruptionTick(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup (not defer) — see TestRunWorldTurn_ForcedSmallFireContinuesChain's comment above
	// (wtForceTierFires below registers its own config-restore t.Cleanup).
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	orc := &Orchestrator{DB: pool}

	const freshWorld = "88888888-8888-8888-8888-888888888888"
	tick, err := orc.lastEruptionTick(ctx, freshWorld, "small")
	if err != nil {
		t.Fatalf("lastEruptionTick: %v", err)
	}
	if tick != 0 {
		t.Fatalf("lastEruptionTick = %d, want 0 for a tier that has never fired", tick)
	}

	// Force a real fire against dlWorldID, then confirm lastEruptionTick reads back a matching tick.
	wtForceTierFires(t, ctx, pool, dlWorldID, "small")
	orcReal := wtOrchestrator(pool)
	baseTick := wtBaseTick(t, ctx, pool)
	chain := []Attempt{{Type: "Communicated", Stated: "Kade mutters into his cup", ListenerID: wtMaraID, Content: "just talking to myself"}}
	out, err := orcReal.RunBeat(ctx, dlWorldID, dlKadeID, chain, baseTick, nil)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	_, firedTick, _, ok := wtEruptionRowForEvent(t, ctx, pool, dlWorldID, out.Committed)
	if !ok {
		t.Fatalf("expected a forced small eruption to have fired and been logged")
	}
	after, err := orc.lastEruptionTick(ctx, dlWorldID, "small")
	if err != nil {
		t.Fatalf("lastEruptionTick after fire: %v", err)
	}
	if after < firedTick {
		t.Fatalf("lastEruptionTick = %d, want >= the just-logged fired_tick %d", after, firedTick)
	}
}
