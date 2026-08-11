package main

import (
	"context"
	"strings"
	"testing"
)

// The founder's live leak, as a test. Kade has never earned "Jonas" — he has only ever perceived
// "the muscle by the bar" — and narration reading "Jonas planted between her and the room" reached
// him on Railway. Both halves are pinned here: the wall knows the name is unearned, and a narration
// segment carrying it is refused.
func TestNamingWall_RefusesTheFoundersLeak(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	wall, err := loadNamingWall(ctx, pool, dlWorldID, dlKadeID)
	if err != nil {
		t.Fatalf("loadNamingWall: %v", err)
	}

	leak := `[{"speaker_id":null,"kind":"narration","text":"Mara is behind the bar now, Jonas planted between her and the room."}]`
	if _, err := DecodeAndValidateNarration(leak, nil, nil, wall); err == nil {
		t.Fatal("the founder's leaked narration was accepted — the wall is not enforcing")
	} else if !strings.Contains(err.Error(), "Jonas") || !strings.Contains(err.Error(), "has not earned") {
		t.Fatalf("rejection must name the offending word so the repair prompt can use it, got: %v", err)
	}

	// Mara IS earned (Kade holds name-knowledge of her): the same sentence without Jonas must pass,
	// or the wall is just censoring every capital letter.
	clean := `[{"speaker_id":null,"kind":"narration","text":"Mara is behind the bar now, the muscle by the bar planted between her and the room."}]`
	if _, err := DecodeAndValidateNarration(clean, nil, nil, wall); err != nil {
		t.Fatalf("a segment naming only what the viewer HAS earned must pass, got: %v", err)
	}
}

// Scrub is the last-resort belt on the paths with no retry (plain fallback, telegraphs). It must be
// total — after Scrub the text cannot breach — and it must leave earned names alone.
func TestNamingWall_ScrubIsTotalAndTargeted(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	wall, err := loadNamingWall(ctx, pool, dlWorldID, dlKadeID)
	if err != nil {
		t.Fatalf("loadNamingWall: %v", err)
	}

	got := wall.Scrub("JONAS blocks the way while Mara watches; jonas does not move.")
	if strings.Contains(strings.ToLower(got), "jonas") {
		t.Fatalf("scrub left an unearned name behind: %q", got)
	}
	if !strings.Contains(got, "Mara") {
		t.Fatalf("scrub removed an EARNED name: %q", got)
	}
	if v := wall.Violations(got); len(v) > 0 {
		t.Fatalf("scrubbed text still violates the wall with %v: %q", v, got)
	}

	// Word boundaries: a name must not be rewritten inside a longer word.
	if got := wall.Scrub("The jonasberry pie sat untouched."); got != "The jonasberry pie sat untouched." {
		t.Fatalf("scrub bit into a longer word: %q", got)
	}
}

// A viewer who has earned every name gets an inert wall rather than a broken one: nil regexp, and
// Violations/Scrub must stay safe to call.
func TestNamingWall_NilSafeAndInertWhenNothingIsUnearned(t *testing.T) {
	var none *NamingWall
	if v := none.Violations("Jonas"); v != nil {
		t.Fatalf("a nil wall must report nothing, got %v", v)
	}
	if got := none.Scrub("Jonas"); got != "Jonas" {
		t.Fatalf("a nil wall must be identity, got %q", got)
	}
	empty := &NamingWall{}
	if v := empty.Violations("Jonas"); v != nil {
		t.Fatalf("an empty wall must report nothing, got %v", v)
	}
}

// SPEC-033 at the belt. The wall is loaded once per beat, so the question that matters for play is
// whether the NEXT beat admits a name the player was just told. It must: the wall reads
// fn_unearned_names, which reads fn_perceived_name, which now reads name_knowledge — so learning
// propagates to the belt with no second code path and nothing to keep in sync.
func TestNamingWall_AdmitsANameTheViewerJustLearned(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	before, err := loadNamingWall(ctx, pool, dlWorldID, dlKadeID)
	if err != nil {
		t.Fatalf("loadNamingWall: %v", err)
	}
	if v := before.Violations("Jonas blocks the way."); len(v) == 0 {
		t.Fatal("fixture is not meaningful: Kade already knows the name before hearing it")
	}

	// Mara says it where Kade can hear. Committed through the engine's own writer, not by inserting
	// name_knowledge directly — the point of the test is that the FAN-OUT teaches.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	var eventID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin)
		 VALUES ($1, 'Communicated', 'Mara tells the stranger the man at the bar is called Jonas.',
		         920, 0, 'accepted', 'freeform')
		 RETURNING event_id::text`, dlWorldID).Scan(&eventID); err != nil {
		t.Fatalf("insert utterance: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
		 VALUES ($1::uuid, $2::uuid, 'actor', 'speaker'), ($1::uuid, $3::uuid, 'actor', 'listener')`,
		eventID, "2ac70000-0000-0000-0000-0000000000a2", dlKadeID); err != nil {
		t.Fatalf("insert participants: %v", err)
	}
	if _, err := tx.Exec(ctx, `SELECT generate_perceptions($1::uuid)`, eventID); err != nil {
		t.Fatalf("generate_perceptions: %v", err)
	}

	// The next beat's wall, loaded inside the same transaction that heard it.
	rows, err := tx.Query(ctx, `SELECT canonical_name FROM fn_unearned_names($1, $2::uuid)`, dlWorldID, dlKadeID)
	if err != nil {
		t.Fatalf("fn_unearned_names: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if name == "Jonas" {
			t.Fatal("the belt still guards a name the viewer was told to his face — narration could never say it")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
}
