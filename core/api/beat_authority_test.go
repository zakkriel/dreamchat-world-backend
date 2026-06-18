package main

import (
	"context"
	"encoding/json"
	"testing"
)

// E1 = the PRIVATE disclosure (Player→Mara, scope=private) — must NEVER reach an uninvolved viewer.
// E102 = the PUBLIC publicize record (held by Common Knowledge) — correctly visible to everyone.
// The no-leak assertion is PROVENANCE-scoped (by source_event_id), never a content grep: BOTH E1 and
// E102 contain the string "hidden ledger", so a text match false-positives on the legitimate public
// record. Provenance, not text.
const (
	e1ID   = "e0000000-0000-0000-0000-000000000001"
	e102ID = "e0000000-0000-0000-0000-000000000102"
)

// Canon-authority (D-1/SPEC-015): every committed event traces to the gate (apply_beat) committing a
// gated proposal — origin='freeform' for model-proposed beats. The model writes nothing to canon
// directly; apply_beat is the only write path.
func TestBeat_CanonAuthority_EveryCommittedEventIsGatedFreeform(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	var summary string
	err := pool.QueryRow(ctx,
		`SELECT apply_beat($1,$2,'[{"type":"move","to":"vault"}]'::jsonb,
		   COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1, 1000, 'freeform')::text`,
		worldID, playerID).Scan(&summary)
	if err != nil {
		t.Fatalf("apply_beat: %v", err)
	}
	var res struct {
		Committed  []string `json:"committed"`
		HaltReason string   `json:"halt_reason"`
	}
	if err := json.Unmarshal([]byte(summary), &res); err != nil {
		t.Fatalf("summary parse: %v (%s)", err, summary)
	}
	if len(res.Committed) == 0 {
		t.Fatalf("beat committed nothing; cannot prove canon-authority: %s", summary)
	}
	// every committed event came through the gate with the model-proposed origin
	for _, id := range res.Committed {
		var origin string
		if err := pool.QueryRow(ctx, `SELECT origin FROM canon_event WHERE event_id=$1`, id).Scan(&origin); err != nil {
			t.Fatalf("origin lookup for %s: %v", id, err)
		}
		if origin != "freeform" {
			t.Fatalf("committed event %s has origin %q — canon written outside the gate (D-1/SPEC-015)", id, origin)
		}
	}
	// defense: no canon_event anywhere carries an origin outside the recognized gated set
	var illegal int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin NOT IN
		   ('fast_path','template','freeform','threshold','backstage','compensation')`,
		worldID).Scan(&illegal); err != nil {
		t.Fatalf("origin scan: %v", err)
	}
	if illegal != 0 {
		t.Fatalf("%d canon events have an illegal origin (canon-authority breach)", illegal)
	}
}

// No-leak (B-1/I-3), PROVENANCE-scoped: no perception whose source_event_id is the PRIVATE disclosure
// (E1) appears in an uninvolved viewer's (Jonas) payload; the PUBLIC record (E102) IS present
// (correct common-knowledge path, B-2). The payload is built from fn_visible_perceptions, so we
// assert on that source by source_event_id AND that the payload faithfully equals the wall output.
func TestBeat_NoLeak_PrivateE1AbsentFromUninvolvedPayload(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	h := &beatHandler{pool: pool, dbg: true}

	// the payload the seats would receive for the uninvolved viewer
	payload, err := h.payload(ctx, worldID, jonasID)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}

	countBySource := func(src string) int {
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM fn_visible_perceptions($1,$2) WHERE source_event_id=$3`,
			worldID, jonasID, src).Scan(&n); err != nil {
			t.Fatalf("count by source %s: %v", src, err)
		}
		return n
	}
	var wallTotal int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM fn_visible_perceptions($1,$2)`, worldID, jonasID).Scan(&wallTotal); err != nil {
		t.Fatalf("wall total: %v", err)
	}

	// the PRIVATE disclosure is forbidden in Jonas's view (provenance, not text)
	// TEMP BITE CHECK (revert me): inverted no-leak expectation — a CORRECT system hides E1 (count 0),
	// so this deliberately-wrong assertion must turn the job RED to prove the CI check has teeth.
	if e1 := countBySource(e1ID); e1 == 0 {
		t.Fatalf("E1 (private disclosure) leaked into Jonas's payload: %d rows (B-1/I-3 breach)", e1)
	}
	// the PUBLIC record is correctly present (common knowledge, B-2) — NOT a leak
	if e102 := countBySource(e102ID); e102 < 1 {
		t.Fatalf("E102 (public record) missing from Jonas's view: %d (common knowledge should be visible)", e102)
	}
	// the payload faithfully equals the wall output → since the wall has zero E1 rows, so does the
	// payload (no E1-sourced content can be in the lines the seats receive)
	if len(payload.Lines) != wallTotal {
		t.Fatalf("payload (%d lines) != fn_visible_perceptions (%d rows); payload may carry extra source",
			len(payload.Lines), wallTotal)
	}
}
