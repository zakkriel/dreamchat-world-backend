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
	if _, err := DecodeAndValidateNarration(leak, NarrationBelts{Wall: wall}); err == nil {
		t.Fatal("the founder's leaked narration was accepted — the wall is not enforcing")
	} else if !strings.Contains(err.Error(), "Jonas") || !strings.Contains(err.Error(), "has not earned") {
		t.Fatalf("rejection must name the offending word so the repair prompt can use it, got: %v", err)
	}

	// Mara IS earned (Kade holds name-knowledge of her): the same sentence without Jonas must pass,
	// or the wall is just censoring every capital letter.
	clean := `[{"speaker_id":null,"kind":"narration","text":"Mara is behind the bar now, the muscle by the bar planted between her and the room."}]`
	if _, err := DecodeAndValidateNarration(clean, NarrationBelts{Wall: wall}); err != nil {
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
	//
	// `summary` is the referee's ACCOUNT and `payload.spoken` is what was actually SAID — the split
	// apply_event/apply_ruled_event have written for every Communicated event since migration
	// 20260809090009. The name has to be in the WORDS: an account that merely mentions someone
	// canonically teaches nothing (naming reach §3), which is what
	// TestNamingWall_AnAccountThatNamesHimTeachesNobody pins directly below.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	var eventID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin, payload)
		 VALUES ($1, 'Communicated', 'Mara tells the stranger who the man at the bar is.',
		         920, 0, 'accepted', 'freeform',
		         jsonb_build_object('spoken', 'the man at the bar is called Jonas'))
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

// THE FOUNDER'S BREACH (live play, 2026-08-14). A speaker label read "Jonas" to a player who had
// never been told the name, and no line of dialogue in the transcript ever said it.
//
// The leak was in the fan-out, not the belt. generate_perceptions taught from the referee's ACCOUNT
// of an utterance rather than from the utterance, and an account names its participants canonically
// because canon is where canonical names live. So a Communicated event whose account happened to
// mention someone taught every listener that person's name — for a nod, a shove, anything at all.
// Two real rows from the seeded world before the fix: Mara learned "Kade" from "Kade nods to Mara
// across the bar", and Kade learned "Cellar Hatch" from "a commotion erupts from the cellar hatch"
// (the old match was case-insensitive, so a common noun read as a proper name).
//
// It compounds, which is why the founder saw it in a LABEL: once a name is in name_knowledge,
// fn_unearned_names drops it from the unearned set entirely, so the wall stops rewriting it in every
// channel at once — and speaker_label is read straight from fn_display_name with no belt of its own.
//
// Hearing teaches. Being described does not.
func TestNamingWall_AnAccountThatNamesHimTeachesNobody(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	// The account names Jonas outright. The WORDS never do — Jonas is being talked about, and what
	// Mara actually says carries no name at all.
	var eventID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin, payload)
		 VALUES ($1, 'Communicated', 'Jonas plants himself between Kade and Mara.',
		         921, 0, 'accepted', 'freeform',
		         jsonb_build_object('spoken', 'you sit quiet, you leave quiet'))
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

	// Nothing was taught: no row, and the belt still guards the name.
	var taught int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM name_knowledge
		  WHERE world_id = $1 AND holder_id = $2::uuid AND name = 'Jonas'`,
		dlWorldID, dlKadeID).Scan(&taught); err != nil {
		t.Fatalf("count name_knowledge: %v", err)
	}
	if taught != 0 {
		t.Fatalf("Kade was taught %q from an account that merely described the man — nobody said the name", "Jonas")
	}

	var stillGuarded bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM fn_unearned_names($1, $2::uuid) WHERE canonical_name = 'Jonas')`,
		dlWorldID, dlKadeID).Scan(&stillGuarded); err != nil {
		t.Fatalf("fn_unearned_names: %v", err)
	}
	if !stillGuarded {
		t.Fatal("the wall stopped guarding Jonas — the label, the narration and every lens can now leak it")
	}

	// The label the founder actually saw. It must still be the descriptor.
	var label string
	if err := tx.QueryRow(ctx, `SELECT fn_display_name($1, $2::uuid, $3::uuid)`,
		dlWorldID, dlKadeID, "2ac70000-0000-0000-0000-0000000000a3").Scan(&label); err != nil {
		t.Fatalf("fn_display_name: %v", err)
	}
	if label == "Jonas" {
		t.Fatal("speaker_label would render the canonical name — this is the founder's reported breach")
	}
}
