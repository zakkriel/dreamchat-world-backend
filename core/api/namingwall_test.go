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
// fn_unheard_names (built on fn_unearned_names, which reads fn_perceived_name, which reads
// name_knowledge) — so a listener's accepted owner recognition propagates to the belt with
// no second code path and nothing to keep in sync.
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

	// Mara says it where Kade can hear, with an accepted speech-perception judgment recognizing
	// Jonas for Kade — committed through the engine's own writer, not by inserting name_knowledge
	// directly. Shared speech perception replaced the old "scan the spoken words for names" fan-out:
	// a listener now learns a name only from his OWN judged owner actor_id, never
	// merely because the name rode inside the words (naming reach §3) — see
	// TestNamingWall_AnAccountThatNamesHimTeachesNobody for the account-side half of that rule, and
	// TestNamingWall_QuotedHeardWordSurvivesUnidentified below for a heard-but-UNrecognized name.
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
		         jsonb_build_object(
		           'spoken', 'the man at the bar is called Jonas',
		           'speech_perception', jsonb_build_object(
		             'schema_version', 'speech_perception/1',
		             'listeners', jsonb_build_array(jsonb_build_object(
		               'listener_id', $2::text,
		               'attention', jsonb_build_object('kind', 'abstain'),
		               'name_associations', jsonb_build_array(jsonb_build_object(
		                 'name', 'Jonas',
		                 'owner', jsonb_build_object('actor_id', '2ac70000-0000-0000-0000-0000000000a3', 'description', NULL, 'about_actor_ids', '[]'::jsonb))),
		               'heard_words', 'the man at the bar is called Jonas'
		             ))
		           )
		         ))
		 RETURNING event_id::text`, dlWorldID, dlKadeID).Scan(&eventID); err != nil {
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
// An accepted no-association judgment must not trigger teaching by scanning the account.
func TestNamingWall_AnAccountThatNamesHimTeachesNobody(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	// Hand-authored application input, not evidence of model understanding: the listener heard
	// these words but identified nobody. The account still names Jonas and must not teach him.
	var eventID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin, payload)
		 VALUES ($1, 'Communicated', 'Jonas plants himself between Kade and Mara.',
		         921, 0, 'accepted', 'freeform',
		         jsonb_build_object(
		           'spoken', 'you sit quiet, you leave quiet',
		           'speech_perception', jsonb_build_object(
		             'schema_version', 'speech_perception/1',
		             'listeners', jsonb_build_array(jsonb_build_object(
		               'listener_id', $2::text,
		               'attention', jsonb_build_object('kind', 'abstain'),
		               'name_associations', '[]'::jsonb,
		               'heard_words', 'you sit quiet, you leave quiet')))))
		 RETURNING event_id::text`, dlWorldID, dlKadeID).Scan(&eventID); err != nil {
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

// THE IRONMOOR BREACH (live play, 2026-08-20). Genesis stored slug join-keys as canonical names —
// "silas_holton", "emmett_vale" — so the wall guarded strings no model ever writes. The cognition
// seats humanised the slugs in their own prose, that prose became the player's perception content
// verbatim, and narration reading "toward Emmett" and "Silas's voice" reached the player on Railway
// while fn_unearned_names still listed both names, verbatim and useless.
//
// The fix is in fn_unearned_names (migration 20260821120000): for PEOPLE, every distinctive word of
// an unearned name is guarded like the name. This test drives it through the Go belt — the exact
// surface the founder's transcript breached.
func TestNamingWall_GuardsTheHumanTokensOfAName(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Hermetic world: fresh random ids, never seed-dependent. The viewer holds no name-knowledge, so
	// both strangers are unearned; one carries the Ironmoor slug verbatim, one a two-word human name.
	var wID, viewer, slugNPC, namedNPC string
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&wID, &viewer, &slugNPC, &namedNPC); err != nil {
		t.Fatalf("mint ids: %v", err)
	}
	mustExecP := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("exec %s: %v", q, err)
		}
	}
	mustExecP(`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		($1,$4,'actor','Ada Vernon'), ($2,$4,'actor','emmett_vale'), ($3,$4,'actor','Silas Holton')`,
		viewer, slugNPC, namedNPC, wID)
	mustExecP(`INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		($1,$4,'{"descriptor":"a woman in a long coat"}'),
		($2,$4,'{"descriptor":"a younger man by the curtain"}'),
		($3,$4,'{"descriptor":"a man in a grey suit"}')`,
		viewer, slugNPC, namedNPC, wID)

	wall, err := loadNamingWall(ctx, pool, wID, viewer)
	if err != nil {
		t.Fatalf("loadNamingWall: %v", err)
	}

	// The two sentences the founder actually read, near-verbatim.
	for _, leak := range []string{
		"The man in the grey suit turns his head an inch toward Emmett.",
		"Silas's voice still hangs in the beeswax air.",
	} {
		if v := wall.Violations(leak); len(v) == 0 {
			t.Errorf("the wall passed the founder's leaked sentence: %q", leak)
		}
		if got := wall.Scrub(leak); strings.Contains(got, "Silas") || strings.Contains(got, "Emmett") {
			t.Errorf("scrub left a human token standing: %q", got)
		}
	}

	// The viewer's own name is never censored, even though a stranger's registry row also reads "Ada…".
	if got := wall.Scrub("Ada keeps her back to the door."); got != "Ada keeps her back to the door." {
		t.Errorf("the viewer's own name was censored: %q", got)
	}
	// And a token never bites into a longer word.
	if got := wall.Scrub("The silastic tube sat by the emmettite ore."); got != "The silastic tube sat by the emmettite ore." {
		t.Errorf("a token matched inside a longer word: %q", got)
	}
}

// The founder's ruling on shared speech perception: hearing a word is not learning whose it is. A
// canonical name this viewer's own recorded perceived speech already contains, literally, is not a
// leak merely for being quoted back — fn_unheard_names exempts exactly that word, and only that
// word; a name nobody ever said in this viewer's hearing stays guarded. The viewer's LABEL for the
// still-unidentified owner is untouched either way: fn_display_name has no knowledge path here, so
// the owner renders as his descriptor, never the canonical name, if he ever appears.
//
// Hermetic world (fresh random ids): the DL fixture's Kade/Jonas pair never has a recorded spoken
// perception for "Jonas" (the seed's Jonas backstory is authored fact, never a Communicated event),
// so it cannot exercise the exemption — this test needs a viewer who genuinely HEARD the name.
//
// Perception rows are inserted directly, not through generate_perceptions/apply_event — this pins
// the READ-side contract (fn_unheard_names/loadNamingWall) only; the write side (attention/
// heard_words judgment application) belongs to the runtime/database owners.
func TestNamingWall_QuotedHeardWordSurvivesUnidentified(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	var wID, viewer, speaker, heardNPC, neverHeardNPC string
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), gen_random_uuid()`,
	).Scan(&wID, &viewer, &speaker, &heardNPC, &neverHeardNPC); err != nil {
		t.Fatalf("mint ids: %v", err)
	}
	mustExecP := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("exec %s: %v", q, err)
		}
	}
	mustExecP(`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		($1,$5,'actor','Rin Ashworth'), ($2,$5,'actor','Corvine'),
		($3,$5,'actor','Dorian'), ($4,$5,'actor','Petra')`,
		viewer, speaker, heardNPC, neverHeardNPC, wID)
	mustExecP(`INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		($1,$5,'{"descriptor":"a dockhand"}'), ($2,$5,'{"descriptor":"the barkeep"}'),
		($3,$5,'{"descriptor":"a stranger nobody names"}'), ($4,$5,'{"descriptor":"a woman by the door"}')`,
		viewer, speaker, heardNPC, neverHeardNPC, wID)

	var eventID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin, payload)
		 VALUES ($1, 'Communicated', 'The barkeep mentions someone by name.', 10, 0, 'accepted', 'freeform',
		         jsonb_build_object('spoken', 'Dorian left.'))
		 RETURNING event_id::text`, wID).Scan(&eventID); err != nil {
		t.Fatalf("insert utterance: %v", err)
	}
	mustExecP(`INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
		 VALUES ($1::uuid, $2::uuid, 'actor', 'speaker'), ($1::uuid, $3::uuid, 'actor', 'listener')`,
		eventID, speaker, viewer)

	// The viewer's own recorded perceived speech.
	mustExecP(`INSERT INTO perception_record
		 (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick, spoken)
		 VALUES ($1, $2, $3::uuid, 'the barkeep says something about someone leaving', 'told', 10, 10, 'Dorian left.')`,
		wID, viewer, eventID)

	wall, err := loadNamingWall(ctx, pool, wID, viewer)
	if err != nil {
		t.Fatalf("loadNamingWall: %v", err)
	}

	// Heard, unidentified: the wall must not fire, in narration prose or in a quote.
	if v := wall.Violations("Someone mentions Dorian in passing."); len(v) != 0 {
		t.Fatalf("a genuinely heard name tripped the wall: %v", v)
	}
	if _, err := DecodeAndValidateNarration(
		`[{"speaker_id":"`+speaker+`","kind":"speech","text":"she glances at the door","quote":"Dorian left."}]`,
		NarrationBelts{
			PresentIDs:  []string{speaker},
			SpeechTexts: map[string][]string{speaker: {"Dorian left."}},
			Wall:        wall,
		},
	); err != nil {
		t.Fatalf("a quote carrying a genuinely heard name must pass the wall, got: %v", err)
	}

	// Never heard: the control name must still trip the wall — the exemption is literal-word-shaped,
	// not "any canonical name eventually becomes sayable".
	if v := wall.Violations("Petra was seen by the door."); len(v) == 0 {
		t.Fatal("a name this viewer never heard must still trip the wall — the exemption leaked past the word it names")
	}

	// The label is untouched: hearing "Dorian" spoken teaches no identity, so fn_display_name for
	// that entity must still be the descriptor, never the canonical name.
	var label string
	if err := pool.QueryRow(ctx, `SELECT fn_display_name($1, $2::uuid, $3::uuid)`,
		wID, viewer, heardNPC).Scan(&label); err != nil {
		t.Fatalf("fn_display_name: %v", err)
	}
	if label == "Dorian" {
		t.Fatal("hearing the word taught identity — fn_display_name must still render the descriptor")
	}
}
