package main

// worldgenesis_test.go — proving a GENERATED world is playable, and that a bad document is refused.
//
// The DB-backed tests all run inside a transaction that is rolled back, and that is not tidiness: canon
// tables carry forbid_delete triggers, so a world committed by a test could never be cleaned up again.
// Rolling back means the assertions run against real Postgres — real triggers, real projections, real
// functions — and leave nothing behind.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testBrief = "A cargo yard where the paperwork is the weapon. Somebody is skimming crates and the person keeping the book knows."

// genesisFixture authors a world with the deterministic fake and commits it inside a rolled-back tx —
// both halves of the split ladder (content, then arrival), so assertions read a whole playable world.
func genesisFixture(t *testing.T) (pgx.Tx, string, *genesisDoc) {
	t.Helper()
	ctx := context.Background()
	pool := testPool(t)
	t.Cleanup(pool.Close)

	doc, _, err := authorWorld(ctx, NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, nil, nil)
	if err != nil {
		t.Fatalf("authorWorld: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	worldID, err := commitWorldContent(ctx, tx, doc, nil, testBrief, "")
	if err != nil {
		t.Fatalf("commitWorldContent: %v", err)
	}
	if err := commitArrival(ctx, tx, worldID, doc, nil); err != nil {
		t.Fatalf("commitArrival: %v", err)
	}
	return tx, worldID, doc
}

// The headline claim of the whole feature: what comes out is playable. Not "playable" in the thin
// player_entity_id sense — every part of the floor the hand-authored template establishes, because a
// world missing any one of them 404s or 500s the moment somebody tries to enter it.
func TestWorldGenesis_AGeneratedWorldIsPlayable(t *testing.T) {
	ctx := context.Background()
	tx, worldID, doc := genesisFixture(t)

	// 1. The one mechanical definition: fn_world_directory reads player_entity_id IS NOT NULL.
	var playerID *string
	var brief *string
	if err := tx.QueryRow(ctx,
		`SELECT player_entity_id::text, brief FROM world WHERE world_id=$1::uuid`, worldID).Scan(&playerID, &brief); err != nil {
		t.Fatalf("read world: %v", err)
	}
	if playerID == nil {
		t.Fatal("world has no player_entity_id — it would be listed as not playable and 404 on every route")
	}
	if brief == nil || *brief != testBrief {
		t.Fatalf("world.brief = %v, want the brief that authored it", brief)
	}

	// 2. The player stands somewhere. buildScene hard-fails with "viewer has no resolvable place" without
	//    this, and beathandler.payload gates the entire candidate assembly on it.
	var placeID string
	if err := tx.QueryRow(ctx,
		`SELECT attrs->>'location_id' FROM actor_state WHERE world_id=$1::uuid AND entity_id=$2::uuid`,
		worldID, *playerID).Scan(&placeID); err != nil {
		t.Fatalf("player location: %v", err)
	}
	if placeID == "" {
		t.Fatal("the player has no location_id")
	}

	// 3. That place is furnished: a description for the narrator, and a tension so the beat budget is
	//    finite. An unstamped room reads as 'none' ⇒ an infinite budget, the exact SPEC-030 blocker.
	var description, tension string
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(attrs->>'description',''), COALESCE(attrs->>'tension','')
		 FROM location_state WHERE world_id=$1::uuid AND entity_id=$2::uuid`,
		worldID, placeID).Scan(&description, &tension); err != nil {
		t.Fatalf("place state: %v", err)
	}
	if description == "" {
		t.Error("the arrival place has no description — the narrator would invent the room")
	}
	switch tension {
	case "frantic", "tense", "normal", "calm", "none":
	default:
		t.Errorf("the arrival place has tension %q, outside the closed set", tension)
	}

	// 4. A way out, and it is a real portal the movement gate will accept.
	var portals int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM artifact_state
		 WHERE world_id=$1::uuid AND attrs ? 'connects' AND attrs->'connects' @> to_jsonb($2::text)`,
		worldID, placeID).Scan(&portals); err != nil {
		t.Fatalf("portals: %v", err)
	}
	if portals == 0 {
		t.Error("nothing connects the arrival place to anywhere — the player cannot leave the room")
	}

	// 5. Enough world to be a world: the region plus every authored place, and somebody to meet.
	var locations, actors int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE entity_kind='location'), count(*) FILTER (WHERE entity_kind='actor')
		 FROM entity_registry WHERE world_id=$1::uuid`, worldID).Scan(&locations, &actors); err != nil {
		t.Fatalf("counts: %v", err)
	}
	if want := len(doc.Places) + 1; locations != want {
		t.Errorf("locations = %d, want %d (every place plus the region)", locations, want)
	}
	if want := len(doc.Cast) + 1; actors != want {
		t.Errorf("actors = %d, want %d (the cast plus the player)", actors, want)
	}

	// 6. Every actor and artifact carries a descriptor. Without one fn_display_name falls through to the
	//    registry canonical name, which is a naming-wall breach by default rather than by accident.
	var bare int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM entity_registry er
		LEFT JOIN actor_state a ON a.entity_id = er.entity_id
		LEFT JOIN artifact_state f ON f.entity_id = er.entity_id
		WHERE er.world_id=$1::uuid AND er.entity_kind IN ('actor','artifact')
		  AND COALESCE(a.attrs->>'descriptor', f.attrs->>'descriptor', '') = ''`, worldID).Scan(&bare); err != nil {
		t.Fatalf("descriptors: %v", err)
	}
	if bare != 0 {
		t.Errorf("%d actors/artifacts have no descriptor, so their canonical names would render", bare)
	}
}

// The rule that is easiest to get wrong and worst to get wrong: the player has walked in knowing nothing.
// One perception, nobody's name, no inner state — and, from the other side, nobody knows theirs.
func TestWorldGenesis_ThePlayerArrivesKnowingNothing(t *testing.T) {
	ctx := context.Background()
	tx, worldID, doc := genesisFixture(t)

	var playerID string
	if err := tx.QueryRow(ctx,
		`SELECT player_entity_id::text FROM world WHERE world_id=$1::uuid`, worldID).Scan(&playerID); err != nil {
		t.Fatalf("player: %v", err)
	}

	// Exactly one perception: their own arrival. Anything more is fan-out they never received.
	var held int
	var content, epistemic string
	if err := tx.QueryRow(ctx,
		`SELECT count(*), COALESCE(max(content),''), COALESCE(max(epistemic_type),'')
		 FROM perception_record WHERE world_id=$1::uuid AND holder_id=$2::uuid`,
		worldID, playerID).Scan(&held, &content, &epistemic); err != nil {
		t.Fatalf("player perceptions: %v", err)
	}
	if held != 1 {
		t.Fatalf("the player holds %d perceptions, want exactly 1 (their arrival)", held)
	}
	if content != strings.TrimSpace(doc.Arrival.Stated) {
		t.Errorf("the player's one perception is %q, want the authored arrival %q", content, doc.Arrival.Stated)
	}
	if epistemic != "direct" {
		t.Errorf("the arrival perception is %q, want direct — they were there", epistemic)
	}

	// B-4, absolute: a premise, not a mind. No core, not even an empty one.
	var cores int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM personality_core WHERE world_id=$1::uuid AND actor_id=$2::uuid`,
		worldID, playerID).Scan(&cores); err != nil {
		t.Fatalf("player core: %v", err)
	}
	if cores != 0 {
		t.Error("the player has a personality_core — the system never authors the player's inner state (B-4)")
	}

	// Every name in the world is unearned by the player, and every one of them renders as a descriptor.
	for _, a := range doc.Cast {
		var shown string
		if err := tx.QueryRow(ctx, `SELECT fn_display_name($1::uuid,$2::uuid,$3::uuid)`,
			worldID, playerID, castID(t, ctx, tx, worldID, a.CanonicalName)).Scan(&shown); err != nil {
			t.Fatalf("display name for %q: %v", a.CanonicalName, err)
		}
		if shown == a.CanonicalName {
			t.Errorf("the player sees %q by name, which nobody told them", a.CanonicalName)
		}
		if shown != a.Descriptor {
			t.Errorf("the player sees %q as %q, want the authored descriptor %q", a.CanonicalName, shown, a.Descriptor)
		}
	}

	// And the other direction: the player is a stranger to the room too.
	for _, a := range doc.Cast {
		var shown string
		if err := tx.QueryRow(ctx, `SELECT fn_display_name($1::uuid,$2::uuid,$3::uuid)`,
			worldID, castID(t, ctx, tx, worldID, a.CanonicalName), playerID).Scan(&shown); err != nil {
			t.Fatalf("player as seen by %q: %v", a.CanonicalName, err)
		}
		if shown == doc.Arrival.CanonicalName {
			t.Errorf("%q knows the player's name %q — nobody here knows who walked in", a.CanonicalName, shown)
		}
	}
}

// Knowledge is authored WITH its path or not at all: every perception traces to an accepted event (I-2)
// and nobody remembers anything before it happened (I-9). These are CI invariants; a generated world has
// to satisfy them the same way the hand-authored one does.
func TestWorldGenesis_KnowledgeIsGroundedAndNeverPrecedesItsCause(t *testing.T) {
	ctx := context.Background()
	tx, worldID, _ := genesisFixture(t)

	var ungrounded int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM perception_record p
		LEFT JOIN canon_event e ON e.event_id = p.source_event_id AND e.status = 'accepted'
		WHERE p.world_id=$1::uuid AND e.event_id IS NULL`, worldID).Scan(&ungrounded); err != nil {
		t.Fatalf("provenance: %v", err)
	}
	if ungrounded != 0 {
		t.Errorf("%d perceptions cite no accepted event (I-2)", ungrounded)
	}

	var impossible int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM perception_record p
		JOIN canon_event e ON e.event_id = p.source_event_id
		WHERE p.world_id=$1::uuid AND p.acquired_tick < e.in_world_tick`, worldID).Scan(&impossible); err != nil {
		t.Fatalf("temporal sanity: %v", err)
	}
	if impossible != 0 {
		t.Errorf("%d perceptions were acquired before the event that caused them (I-9)", impossible)
	}

	// The arrival is the world's newest moment, so the live handler mints the next beat after it.
	var now int64
	if err := tx.QueryRow(ctx, `SELECT fn_world_now($1::uuid)`, worldID).Scan(&now); err != nil {
		t.Fatalf("fn_world_now: %v", err)
	}
	if now != genesisArrivalTick {
		t.Errorf("world clock = %d, want the arrival tick %d", now, genesisArrivalTick)
	}
}

// A perception sourced from a `world_genesis` event IS a name, as far as this engine is concerned:
// fn_perceived_name reads every such perception that is subject-linked to an entity and returns its
// content as that entity's name. So nothing except name knowledge may cite that event.
//
// This test exists because a live build broke the rule. Secrets were grounded in the naming event — it
// was the one event that existed before the backstory — and the archivist's own compendium entry came
// back with `perceived_name` set to her forgery scheme. Every fake-bridge test passed: the fake authors
// secrets too, but nothing had ever read a name back from an NPC's own point of view.
func TestWorldGenesis_NothingButNamesHangsOffTheNamingEvent(t *testing.T) {
	ctx := context.Background()
	tx, worldID, doc := genesisFixture(t)

	// Every perception the naming event sources must be a name someone holds for someone: its content is
	// a canonical name in this world, and nothing else.
	names := map[string]bool{}
	for _, a := range doc.Cast {
		names[strings.TrimSpace(a.CanonicalName)] = true
	}
	rows, err := tx.Query(ctx, `
		SELECT p.content
		FROM perception_record p
		JOIN canon_event e ON e.event_id = p.source_event_id
		WHERE p.world_id = $1::uuid AND e.event_type = 'world_genesis'`, worldID)
	if err != nil {
		t.Fatalf("query genesis-sourced perceptions: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if !names[content] {
			t.Errorf("a perception sourced from the naming event carries %q, which is not anyone's name — "+
				"fn_perceived_name will render it AS a name", content)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	// And the reading that actually broke: nobody's perceived name is their secret.
	for _, a := range doc.Cast {
		actorID := castID(t, ctx, tx, worldID, a.CanonicalName)
		var shown string
		if err := tx.QueryRow(ctx, `SELECT COALESCE(fn_perceived_name($1::uuid,$2::uuid,$2::uuid),'')`,
			worldID, actorID).Scan(&shown); err != nil {
			t.Fatalf("perceived name of %q to themselves: %v", a.CanonicalName, err)
		}
		if shown == strings.TrimSpace(a.Hiding) {
			t.Errorf("%q's own perceived name is their secret", a.CanonicalName)
		}
	}
}

// Each secret is held by exactly one person and is invisible to everyone else — the planted-secret shape
// the engine's own I-3 test hunts for leaks of.
func TestWorldGenesis_EverySecretIsPrivateToItsHolder(t *testing.T) {
	ctx := context.Background()
	tx, worldID, doc := genesisFixture(t)

	for _, a := range doc.Cast {
		holderID := castID(t, ctx, tx, worldID, a.CanonicalName)
		var holders int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM perception_record WHERE world_id=$1::uuid AND content=$2`,
			worldID, strings.TrimSpace(a.Hiding)).Scan(&holders); err != nil {
			t.Fatalf("secret of %q: %v", a.CanonicalName, err)
		}
		if holders != 1 {
			t.Errorf("%q's secret is held by %d people, want exactly 1", a.CanonicalName, holders)
		}
		var visibleToHolder int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM fn_visible_perceptions($1::uuid,$2::uuid) WHERE content=$3`,
			worldID, holderID, strings.TrimSpace(a.Hiding)).Scan(&visibleToHolder); err != nil {
			t.Fatalf("visibility of %q's secret: %v", a.CanonicalName, err)
		}
		if visibleToHolder != 1 {
			t.Errorf("%q cannot see their own secret", a.CanonicalName)
		}
	}
}

func castID(t *testing.T, ctx context.Context, tx pgx.Tx, worldID, canonicalName string) string {
	t.Helper()
	var id string
	if err := tx.QueryRow(ctx,
		`SELECT entity_id::text FROM entity_registry
		 WHERE world_id=$1::uuid AND entity_kind='actor' AND canonical_name=$2`,
		worldID, canonicalName).Scan(&id); err != nil {
		t.Fatalf("id of %q: %v", canonicalName, err)
	}
	return id
}

// ── The belt, with no database in sight ───────────────────────────────────────────────────────────

// authoredWorld returns the fake's document, decoded — the starting point for corruption tests.
func authoredWorld(t *testing.T) *genesisDoc {
	t.Helper()
	doc, _, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, nil, nil)
	if err != nil {
		t.Fatalf("authorWorld: %v", err)
	}
	return doc
}

// Every one of these is a document the SCHEMA permits and the world cannot survive. They are refusals,
// not crashes: the user is told the brief did not become a world, and nothing was written.
func TestWorldGenesis_RefusesDocumentsTheWorldCannotSurvive(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(*genesisDoc)
		want   string
	}{
		{"a way to nowhere", func(d *genesisDoc) { d.Ways[0].ToPlace = "The Room That Was Never Authored" }, "not a place"},
		{"an arrival into nowhere", func(d *genesisDoc) { d.Arrival.Place = "Elsewhere" }, "not a place"},
		{"a secret held by nobody", func(d *genesisDoc) { d.Cast[0].Hiding = "" }, "hiding nothing"},
		{"an object nowhere at all", func(d *genesisDoc) {
			d.Objects[0].Where.InPlace = ""
			d.Objects[0].Where.CarriedBy = ""
		}, "is nowhere"},
		{"a room with no way out", func(d *genesisDoc) { d.Ways = nil }, "nothing joins the places"},
		{"an empty opening", func(d *genesisDoc) {
			for i := range d.Cast {
				d.Cast[i].StartsIn = d.Places[1].CanonicalName
			}
			d.Arrival.Place = d.Places[0].CanonicalName
		}, "nobody is in"},
		{"one room only", func(d *genesisDoc) { d.Places = d.Places[:1] }, "at least two places"},
		{"a tension outside the set", func(d *genesisDoc) { d.Places[0].Tension = "spooky" }, "outside the closed set"},
		{"a name used twice", func(d *genesisDoc) { d.Cast[0].CanonicalName = d.Places[0].CanonicalName }, "both a person and a place"},
		{"the player in their own cast", func(d *genesisDoc) { d.Arrival.CanonicalName = d.Cast[0].CanonicalName }, "also in the cast"},
		{"a place with no description", func(d *genesisDoc) { d.Places[0].Description = "" }, "nothing to work from"},
		// The Ironmoor breach (2026-08-20): genesis emitted slug join-keys as people's canonical
		// names ("silas_holton"), the registry stored them, and the naming wall guarded strings no
		// model ever writes while the narrator handed the player "Silas". A person's name must be
		// speakable — the wall can only guard what the world will actually say.
		{"a snake_case person", func(d *genesisDoc) { d.Cast[0].CanonicalName = "silas_holton" }, "join key"},
		{"an uncapitalised person", func(d *genesisDoc) { d.Cast[0].CanonicalName = "silas holton" }, "join key"},
		{"a snake_case player", func(d *genesisDoc) { d.Arrival.CanonicalName = "wren_marsh" }, "join key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := authoredWorld(t)
			tc.break_(doc)
			err := doc.validate()
			if err == nil {
				t.Fatalf("%s was accepted; it must be refused", tc.name)
			}
			if !IsGenesisRefusal(err) {
				t.Errorf("error is not a refusal, so it would read as a crash: %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("refusal = %q, want it to mention %q", err.Error(), tc.want)
			}
		})
	}
}

// The single most important refusal, and the one a plausible model will actually attempt: handing the
// arriving player a memory of something that happened before they existed here.
func TestWorldGenesis_RefusesToGiveThePlayerKnowledgeTheyDidNotEarn(t *testing.T) {
	doc := authoredWorld(t)
	doc.History[0].Knowledge = append(doc.History[0].Knowledge, genesisKnowledge{
		Holder:        doc.Arrival.CanonicalName,
		Content:       "I already knew about the crate before I walked in",
		EpistemicType: "told",
	})
	err := doc.validate()
	if err == nil {
		t.Fatal("the player was given backstory knowledge and the document was accepted")
	}
	if !strings.Contains(err.Error(), "they were not there") {
		t.Errorf("refusal = %q, want it to say the player was not there", err.Error())
	}
}

// The brief has to actually reach the seat. Asserted through the fake's one brief-derived field, so the
// prompt assembly and the marker contract are both covered.
func TestWorldGenesis_TheBriefReachesTheSeat(t *testing.T) {
	doc := authoredWorld(t)
	if !strings.Contains(strings.ToLower(doc.World.DisplayName), "cargo") {
		t.Errorf("display_name = %q, want it derived from the brief", doc.World.DisplayName)
	}
	if _, _, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), "   ", nil, nil, nil); err == nil {
		t.Error("an empty brief was accepted; there is nothing to build from")
	}
}

// The seat cannot emit a number, and the schema is what guarantees it. Asserted against the schema text
// itself: this is the leash, and a future edit that adds a numeric field should fail here loudly.
func TestWorldGenesis_SchemaForbidsEveryNumber(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(worldGenesisSchemaJSON), &schema); err != nil {
		t.Fatalf("schema does not parse: %v", err)
	}
	var walk func(node any, path string)
	walk = func(node any, path string) {
		switch n := node.(type) {
		case map[string]any:
			if typ, ok := n["type"].(string); ok && (typ == "number" || typ == "integer") {
				t.Errorf("%s is a %s — the seat must never emit a number (the engine owns all of them)", path, typ)
			}
			for k, v := range n {
				// minLength/minItems/maxItems are OUR constraints on the seat, not fields it emits.
				if k == "minLength" || k == "minItems" || k == "maxItems" {
					continue
				}
				walk(v, path+"."+k)
			}
		case []any:
			for i, v := range n {
				walk(v, path)
				_ = i
			}
		}
	}
	walk(schema["properties"], "properties")
	walk(schema["definitions"], "definitions")
}

// ── The interview ────────────────────────────────────────────────────────────────────────────────

// The Custom lane walks: a question with real options, then another, then done. Driven through the same
// stateless contract the client uses, so what is proved here is what a browser gets.
func TestWorldInterview_AsksThenStops(t *testing.T) {
	ctx := context.Background()
	seat := NewFakeWorldInterviewDriver()

	var answers []InterviewAnswer
	for i := 0; i < fakeInterviewQuestionCount; i++ {
		turn, err := askNextQuestion(ctx, seat, testBrief, answers)
		if err != nil {
			t.Fatalf("question %d: %v", i+1, err)
		}
		if turn.Done {
			t.Fatalf("interview ended after %d questions, want %d", i, fakeInterviewQuestionCount)
		}
		if strings.TrimSpace(turn.Question) == "" {
			t.Fatal("a question with no text is unanswerable")
		}
		if len(turn.Options) < 3 {
			t.Errorf("question %d offers %d options, want at least 3", i+1, len(turn.Options))
		}
		recommended := 0
		for _, o := range turn.Options {
			if o.Recommended {
				recommended++
			}
		}
		if recommended > 1 {
			t.Errorf("question %d marks %d recommendations — two defaults is no default", i+1, recommended)
		}
		answers = append(answers, InterviewAnswer{Question: turn.Question, Answer: turn.Options[0].Label})
	}

	turn, err := askNextQuestion(ctx, seat, testBrief, answers)
	if err != nil {
		t.Fatalf("final turn: %v", err)
	}
	if !turn.Done {
		t.Errorf("the interview asked a %d-th question; it must be willing to stop", fakeInterviewQuestionCount+1)
	}
}

// Answers must reach the genesis seat, or the Custom lane is theatre.
func TestWorldInterview_AnswersReachTheGenesisPrompt(t *testing.T) {
	prompt := buildWorldGenesisPrompt(testBrief, []InterviewAnswer{
		{Question: "Who wants something from you?", Answer: "the one keeping the records"},
	})
	if !strings.Contains(prompt, worldGenesisAnswersMarker) {
		t.Error("the prompt carries no answers block")
	}
	if !strings.Contains(prompt, "the one keeping the records") {
		t.Error("the user's answer is missing from the prompt")
	}
	if !strings.Contains(prompt, testBrief) {
		t.Error("the brief is missing from the prompt")
	}
}

func TestValidateArrivalCandidates(t *testing.T) {
	doc := authoredWorld(t) // reuse/extract the minimal passing doc the existing tests build
	cand := func(name string) genesisCandidate {
		return genesisCandidate{Descriptor: "a stranger in a wet coat", CanonicalName: name, Why: "owed money"}
	}

	// exactly 3
	doc.ArrivalCandidates = []genesisCandidate{cand(doc.Arrival.CanonicalName), cand("Second Name")}
	if err := doc.validate(); err == nil {
		t.Fatal("2 candidates accepted; want refusal (exactly 3)")
	}

	// distinct names
	doc.ArrivalCandidates = []genesisCandidate{cand(doc.Arrival.CanonicalName), cand("Second Name"), cand("Second Name")}
	if err := doc.validate(); err == nil {
		t.Fatal("duplicate candidate names accepted")
	}

	// exactly one must match arrival.canonical_name
	doc.ArrivalCandidates = []genesisCandidate{cand("First Name"), cand("Second Name"), cand("Third Name")}
	if err := doc.validate(); err == nil {
		t.Fatal("no candidate matches the arrival; want refusal")
	}

	// the happy path
	doc.ArrivalCandidates = []genesisCandidate{cand(doc.Arrival.CanonicalName), cand("Second Name"), cand("Third Name")}
	if err := doc.validate(); err != nil {
		t.Fatalf("valid candidates refused: %v", err)
	}

	// A candidate is a person too: the Ironmoor name-shape rule applies before the offer ships.
	doc.ArrivalCandidates = []genesisCandidate{cand(doc.Arrival.CanonicalName), cand("petra_stone"), cand("Third Name")}
	if err := doc.validate(); err == nil || !strings.Contains(err.Error(), "join key") {
		t.Fatalf("a slug candidate name must be refused as a join key, got: %v", err)
	}

	// refusals must be genesisRefusal, not faults
	doc.ArrivalCandidates = doc.ArrivalCandidates[:2]
	if err := doc.validate(); !IsGenesisRefusal(err) {
		t.Fatalf("candidate violation is not a refusal: %v", err)
	}
}

func TestFakeGenesisEmitsCandidatesUnlessIdentityStated(t *testing.T) {
	open, _, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), "a harbour town at closing time", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(open.ArrivalCandidates) != 3 {
		t.Fatalf("identity-open brief: %d candidates, want 3", len(open.ArrivalCandidates))
	}
	stated, _, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), "a harbour town; I am the debt collector", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stated.ArrivalCandidates) != 0 {
		t.Fatalf("identity-stated brief: %d candidates, want 0", len(stated.ArrivalCandidates))
	}
}

// kickstartHarness is the handler-and-server pair one test's kickstart turns run against, keyed by the
// *testing.T that created it so nothing leaks between tests that never share one. A real httptest.Server
// (not a bare recorder) because the kickstart route is exercised as a genuine second-and-third HTTP call
// against the SAME draft store the first call populated — a recorder alone has no URL a later call can
// address independently.
type kickstartHarness struct {
	srv  *httptest.Server
	pool *pgxpool.Pool
}

var kickstartHarnesses = map[*testing.T]*kickstartHarness{}

// kickstartHarnessFor returns this test's harness, creating it on first use. A test that only calls
// postKickstartRaw (e.g. against an unknown handle) gets a fresh, otherwise-empty draft store — which is
// exactly what it needs to prove a handle nobody minted here is expired.
func kickstartHarnessFor(t *testing.T) *kickstartHarness {
	t.Helper()
	if kh, ok := kickstartHarnesses[t]; ok {
		return kh
	}
	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldUnderstanding.Name: NewFakeWorldUnderstandingDriver(),
		SeatWorldFill.Name:          NewFakeWorldFillDriver(),
		SeatWorldFillReview.Name:    NewFakeWorldFillReviewDriver(),
		SeatWorldKickstart.Name:     NewFakeWorldKickstartDriver(),
	}, SeatWorldUnderstanding, SeatWorldFill, SeatWorldFillReview, SeatWorldKickstart)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	pool := testPool(t)
	t.Cleanup(pool.Close)
	h := NewWorldGenesisHandler(pool, true, bridge, nil)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	kh := &kickstartHarness{srv: srv, pool: pool}
	kickstartHarnesses[t] = kh
	return kh
}

// kickstartPool is the DB pool behind this test's harness, for assertions the kickstart route's own
// (internal, unexported) commit tx cannot hand back — the world it commits is real and visible on it.
func kickstartPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return kickstartHarnessFor(t).pool
}

// postGenesisAndCollectFrames drives the real /worlds/genesis route under a bridge that answers both
// world_genesis and world_kickstart, and returns every SSE frame decoded into a map. Shared by every
// test that asserts on the shape of the build stream.
func postGenesisAndCollectFrames(t *testing.T, body string) []map[string]any {
	t.Helper()
	srv := kickstartHarnessFor(t).srv
	resp, err := http.Post(srv.URL+"/worlds/genesis", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post genesis: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read genesis body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("genesis status = %d, want 200 (body %s)", resp.StatusCode, raw)
	}
	frames := make([]map[string]any, 0)
	for _, frame := range sseFrames(t, string(raw)) {
		var f map[string]any
		if err := json.Unmarshal(frame, &f); err != nil {
			t.Fatalf("frame is not JSON: %v", err)
		}
		frames = append(frames, f)
	}
	return frames
}

// postKickstartRaw posts one kickstart turn and returns the raw response — for tests asserting on
// status alone (an unknown world, a finished world).
func postKickstartRaw(t *testing.T, worldID, answer string) *http.Response {
	t.Helper()
	srv := kickstartHarnessFor(t).srv
	body, err := json.Marshal(kickstartRequest{WorldID: worldID, Answer: answer})
	if err != nil {
		t.Fatalf("marshal kickstart request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/worlds/genesis/kickstart", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post kickstart: %v", err)
	}
	return resp
}

// postKickstart posts one kickstart turn and decodes the JSON response, failing the test on anything
// but 200.
func postKickstart(t *testing.T, worldID, answer string) map[string]any {
	t.Helper()
	resp := postKickstartRaw(t, worldID, answer)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read kickstart body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("kickstart status = %d, want 200 (body %s)", resp.StatusCode, raw)
	}
	var turn map[string]any
	if err := json.Unmarshal(raw, &turn); err != nil {
		t.Fatalf("kickstart response is not JSON: %v", err)
	}
	return turn
}

// recommendedLabel returns the label of the option flagged recommended among a choice turn's options,
// or fails the test — every character and scenario turn this fake bridge authors carries exactly one.
func recommendedLabel(t *testing.T, options any) string {
	t.Helper()
	opts, ok := options.([]any)
	if !ok {
		t.Fatalf("options is not a list: %v (%T)", options, options)
	}
	for _, o := range opts {
		m, ok := o.(map[string]any)
		if !ok {
			continue
		}
		if m["recommended"] == true {
			label, _ := m["label"].(string)
			return label
		}
	}
	t.Fatalf("no recommended option among %v", opts)
	return ""
}

// The stream now pauses for the player's own choice rather than committing the player: the terminal
// frame is a `choice` carrying the id of the world the stream already committed (durable-worlds).
func TestBuildEndsInCharacterChoice(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	last := frames[len(frames)-1]
	if last["kind"] != "choice" {
		t.Fatalf("terminal frame kind = %v, want choice", last["kind"])
	}
	if last["schema_version"] != "world_genesis_frame/3" {
		t.Fatalf("schema_version = %v", last["schema_version"])
	}
	if last["question"] != "Who are you here?" {
		t.Fatalf("question = %v", last["question"])
	}
	opts := last["options"].([]any)
	if len(opts) != 3 {
		t.Fatalf("options = %d, want 3", len(opts))
	}
	rec := 0
	for _, o := range opts {
		if o.(map[string]any)["recommended"] == true {
			rec++
		}
	}
	if rec != 1 {
		t.Fatalf("recommended = %d, want 1", rec)
	}
	if id, _ := last["world_id"].(string); len(id) != 36 {
		t.Fatalf("world_id = %q", id)
	}
	// AC-7: the doc never crosses the wire — no frame carries cast, history or hiding.
	for _, f := range frames {
		for _, k := range []string{"cast", "history", "hiding", "knowledge"} {
			if _, present := f[k]; present {
				t.Fatalf("frame leaked %q", k)
			}
		}
	}
}

// When the brief already states who the player is, there are no candidates to choose among — the
// stream authors the scenario options in the same pass and ends there instead, one round-trip saved.
func TestBuildSkipsToScenarioWhenIdentityStated(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town; I am the debt collector"}`)
	last := frames[len(frames)-1]
	if last["kind"] != "choice" || last["question"] != "How does it start?" {
		t.Fatalf("terminal frame = %v", last)
	}
	if len(last["options"].([]any)) != 3 {
		t.Fatal("want 3 scenario options")
	}
}

// The whole point: two answers turn a committed-but-unentered world into a playable one, the arrival
// landing in one transaction on the final answer — and a third answer against the now-finished world
// answers 409, because there is nothing left to resume.
func TestKickstartFullJourney(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	choice := frames[len(frames)-1]
	worldIDFromChoice, _ := choice["world_id"].(string)

	// Turn 1: pick the recommended character.
	recLabel := recommendedLabel(t, choice["options"])
	turn := postKickstart(t, worldIDFromChoice, recLabel)
	if turn["schema_version"] != "world_kickstart_turn/2" || turn["done"] != false {
		t.Fatalf("turn 1 = %v", turn)
	}
	if turn["question"] != "How does it start?" {
		t.Fatalf("turn 1 question = %v", turn["question"])
	}

	// Turn 2: pick the recommended scenario — this commits.
	turn2 := postKickstart(t, worldIDFromChoice, recommendedLabel(t, turn["options"]))
	if turn2["done"] != true {
		t.Fatalf("turn 2 = %v", turn2)
	}
	world, ok := turn2["world"].(map[string]any)
	if !ok {
		t.Fatalf("turn 2 carries no world: %v", turn2)
	}
	worldID, _ := world["id"].(string)
	if world["playable"] != true || worldID == "" {
		t.Fatalf("world = %v", world)
	}

	// The world is finished: a third answer finds nothing left to resume.
	res := postKickstartRaw(t, worldIDFromChoice, "anything")
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("finished world status = %d, want 409", res.StatusCode)
	}

	// DB floor (AC-4/AC-9): the player is stamped on the committed world and holds exactly one
	// perception — their own arrival. Nothing rolls this back (the kickstart route commits for real,
	// same as the single-shot build did before the split), so this queries the pool directly.
	ctx := context.Background()
	pool := kickstartPool(t)
	var playerID string
	if err := pool.QueryRow(ctx,
		`SELECT player_entity_id::text FROM world WHERE world_id=$1::uuid`, worldID).Scan(&playerID); err != nil {
		t.Fatalf("player: %v", err)
	}
	if playerID == "" {
		t.Fatal("world has no player_entity_id — the arrival never committed")
	}
	var held int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM perception_record WHERE world_id=$1::uuid AND holder_id=$2::uuid`,
		worldID, playerID).Scan(&held); err != nil {
		t.Fatalf("player perceptions: %v", err)
	}
	if held != 1 {
		t.Fatalf("the player holds %d perceptions, want exactly 1 (their arrival)", held)
	}
}

// Free text is a first-class answer at both turns, not a fallback: the character question grounds it as
// who the player is, the scenario question grounds it as how the opening goes — and either way the
// build still commits.
func TestKickstartCustomAnswersFlowIn(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	worldID, _ := frames[len(frames)-1]["world_id"].(string)

	turn := postKickstart(t, worldID, "the harbour master's estranged child, back unannounced")
	if turn["done"] != false {
		t.Fatalf("custom character rejected: %v", turn)
	}

	turn2 := postKickstart(t, worldID, "I slip in through the kitchen while an argument is going on")
	if turn2["done"] != true {
		t.Fatalf("custom scenario did not commit: %v", turn2)
	}
}

// An unknown world id — never committed by any build — answers 404: there is nothing there at all,
// which is a different truth from a finished world's 409.
func TestKickstartUnknownWorld(t *testing.T) {
	res := postKickstartRaw(t, "00000000-0000-0000-0000-000000000000", "anything")
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

// The durable-worlds headline: the world exists, listed and not yet enterable, the moment the build
// stream ends — BEFORE any kickstart answer. Losing every server memory after that costs at most one
// re-asked question: an empty answer re-serves the pending question against the world id alone.
func TestBuildCommitsWorldBeforeChoice(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	choice := frames[len(frames)-1]
	worldID, _ := choice["world_id"].(string)
	if worldID == "" {
		t.Fatalf("choice frame carries no world_id: %v", choice)
	}

	ctx := context.Background()
	pool := kickstartPool(t)
	var playerID *string
	var hasDoc, hasState bool
	if err := pool.QueryRow(ctx,
		`SELECT player_entity_id::text, genesis_doc IS NOT NULL, kickstart_state IS NOT NULL
		 FROM world WHERE world_id=$1::uuid`, worldID).Scan(&playerID, &hasDoc, &hasState); err != nil {
		t.Fatalf("read world: %v", err)
	}
	if playerID != nil {
		t.Fatal("player_entity_id set before any kickstart answer — the choice was never offered")
	}
	if !hasDoc || !hasState {
		t.Fatalf("genesis_doc present=%v kickstart_state present=%v — the creation is not resumable", hasDoc, hasState)
	}

	// Resume with an empty answer: the pending question comes back. Nothing about this request relies
	// on process memory — the harness could have restarted between these two calls.
	turn := postKickstart(t, worldID, "")
	if turn["done"] != false || turn["question"] != "Who are you here?" {
		t.Fatalf("resume turn = %v", turn)
	}

	// And the journey still finishes from there.
	turn1 := postKickstart(t, worldID, recommendedLabel(t, turn["options"]))
	turn2 := postKickstart(t, worldID, recommendedLabel(t, turn1["options"]))
	if turn2["done"] != true {
		t.Fatalf("resumed journey did not commit: %v", turn2)
	}
}

// Referenced people become real (durable-worlds AC-5): a free-text identity naming kin produces
// cast entries for exactly those people, with minds, placement, and mutual name knowledge with the
// player at the arrival tick.
func TestKickstartKinMaterialized(t *testing.T) {
	frames := postGenesisAndCollectFrames(t, `{"brief":"a harbour town at closing time"}`)
	choice := frames[len(frames)-1]
	worldID, _ := choice["world_id"].(string)

	turn := postKickstart(t, worldID, "Joe, son of Dalma and Harry")
	if turn["done"] != false {
		t.Fatalf("character turn = %v", turn)
	}
	turn2 := postKickstart(t, worldID, recommendedLabel(t, turn["options"]))
	if turn2["done"] != true {
		t.Fatalf("scenario turn did not commit: %v", turn2)
	}

	ctx := context.Background()
	pool := kickstartPool(t)
	var playerID string
	if err := pool.QueryRow(ctx,
		`SELECT player_entity_id::text FROM world WHERE world_id=$1::uuid`, worldID).Scan(&playerID); err != nil {
		t.Fatalf("player: %v", err)
	}
	for _, kin := range []string{"Dalma", "Harry"} {
		var kinID string
		if err := pool.QueryRow(ctx,
			`SELECT entity_id::text FROM entity_registry
			 WHERE world_id=$1::uuid AND canonical_name=$2 AND entity_kind='actor'`,
			worldID, kin).Scan(&kinID); err != nil {
			t.Fatalf("%s was referenced but never registered: %v", kin, err)
		}
		var cores int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM personality_core WHERE world_id=$1::uuid AND actor_id=$2::uuid`,
			worldID, kinID).Scan(&cores); err != nil {
			t.Fatalf("%s core: %v", kin, err)
		}
		if cores != 1 {
			t.Fatalf("%s has %d personality cores, want 1 — a referenced person is a whole person", kin, cores)
		}
		var placed bool
		if err := pool.QueryRow(ctx,
			`SELECT attrs->>'location_id' IS NOT NULL FROM actor_state
			 WHERE world_id=$1::uuid AND entity_id=$2::uuid`, worldID, kinID).Scan(&placed); err != nil {
			t.Fatalf("%s state: %v", kin, err)
		}
		if !placed {
			t.Fatalf("%s stands nowhere", kin)
		}
		// Mutual name knowledge at the arrival tick (I-9: earned exactly when the premise landed).
		var playerKnows, kinKnows int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM perception_record pr
			 JOIN perception_subject ps ON ps.perception_id = pr.perception_id
			 WHERE pr.world_id=$1::uuid AND pr.holder_id=$2::uuid AND ps.entity_id=$3::uuid
			   AND pr.content=$4 AND pr.acquired_tick=$5`,
			worldID, playerID, kinID, kin, genesisArrivalTick).Scan(&playerKnows); err != nil {
			t.Fatalf("player knows %s: %v", kin, err)
		}
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM perception_record pr
			 JOIN perception_subject ps ON ps.perception_id = pr.perception_id
			 WHERE pr.world_id=$1::uuid AND pr.holder_id=$2::uuid AND ps.entity_id=$3::uuid
			   AND pr.acquired_tick=$4`,
			worldID, kinID, playerID, genesisArrivalTick).Scan(&kinKnows); err != nil {
			t.Fatalf("%s knows player: %v", kin, err)
		}
		if playerKnows != 1 || kinKnows != 1 {
			t.Fatalf("name knowledge player→%s=%d, %s→player=%d, want 1 and 1", kin, playerKnows, kin, kinKnows)
		}
	}
}
