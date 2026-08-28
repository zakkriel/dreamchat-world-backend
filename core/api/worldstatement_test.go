package main

// worldstatement_test.go — the two guarantees the world's global statement ships with.
//
//  1. It reaches ALL THREE narrate builders, including the plain fallback. That fallback is the path
//     taken only after two structured attempts have already failed — the moment invention risk is
//     highest — and narrateBaseRules slices the header at narrateSegmentContractMarker, so a block
//     added to the end of narrate.txt reaches the structured path and silently dies on that one.
//  2. It is NEVER world.brief. The brief is the prose a user typed; the document is what the world was
//     actually built from. Handing the narrator the brief while the document is short of it hands it
//     material the state cannot back — the founder-gate bug NEVER CONTRADICT OR EXTEND THE STATE was
//     written for (narrateprompt.go:14-17,28).
//
// The DB-backed test runs inside a rolled-back transaction, for the reason worldgenesis_test.go states:
// canon tables carry forbid_delete triggers, so a world a test commits could never be cleaned up.

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// aWorldStatement is the fixture. Every value is deliberately absent from prompts/narrate.txt, so a
// hit proves the RENDERED block reached the prompt rather than the fixed rulebook mentioning it.
func aWorldStatement() WorldStatement {
	return WorldStatement{
		Name:     "Kirrowmede",
		Premise:  "every debt here is owed to the water, and the water keeps its books",
		Mood:     "ashen",
		Ornament: "filigreed",
		Region:   "a terrace of drowned counting-houses",
	}
}

// (o) The F5 guard. All three builders, or the fallback narrates a world it was never told about.
func TestNarrateWorldBlock_ReachesAllThreeBuilders(t *testing.T) {
	ws := aWorldStatement()
	payload := PerceptionPayload{
		World:      ws,
		Here:       "loc-1",
		Candidates: []Candidate{{ID: "loc-1", Kind: "location", Name: "The Drowned Lantern"}},
		Lines:      []string{"Mara sets a tankard on the bar."},
	}

	builders := map[string]string{
		"buildNarratePrompt":       buildNarratePrompt(payload, "", nil, ""),
		"buildNarrateRepairPrompt": buildNarrateRepairPrompt(payload, "", nil, "", "a ghost speaker"),
		"buildNarratePlainPrompt":  buildNarratePlainPrompt(payload, "", nil, ""),
	}
	for name, prompt := range builders {
		for _, want := range []string{
			narrateWorldBlockMarker, ws.Name, ws.Premise, ws.Region, ws.Mood, ws.Ornament,
		} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("(o) %s does not carry the world's global statement (missing %q):\n%s", name, want, prompt)
			}
		}
	}

	// The trap this test exists for, asserted directly: the plain fallback drops everything from the
	// segment-contract marker onward, and the world block must not be in the dropped part.
	plain := builders["buildNarratePlainPrompt"]
	if strings.Contains(plain, narrateSegmentContractMarker) {
		t.Fatal("(o) fixture is not meaningful: the plain fallback still carries the segment contract, so the slice under test never ran")
	}

	// The instruction that bounds the block travels with it — including on the fallback, which is why
	// the rule lives BEFORE the marker in narrate.txt rather than after it.
	for name, prompt := range builders {
		if !strings.Contains(prompt, "THE WORLD IS CONTEXT, NEVER CONTENT") {
			t.Fatalf("(o) %s carries the world block without the rule that bounds it to register", name)
		}
	}
}

// (p) An unauthored world renders no block at all — the same discipline YOU ARE follows. A bare
// "THE WORLD:" header with nothing behind it is material for the model to invent against.
func TestNarrateWorldBlock_OmittedWhenTheWorldHasNoStatement(t *testing.T) {
	body := strings.TrimPrefix(buildNarratePrompt(PerceptionPayload{
		Lines: []string{"Mara sets a tankard on the bar."},
	}, "", nil, ""), narrateSystemHeader)

	if strings.Contains(body, narrateWorldBlockMarker) {
		t.Fatalf("(p) an empty statement rendered a bare world header:\n%s", body)
	}
	if strings.Contains(body, "ITS REGISTER:") || strings.Contains(body, "THE REGION:") {
		t.Fatalf("(p) an empty statement rendered a dangling sub-line:\n%s", body)
	}
}

// (q) The static half of "never the brief": the read cannot fetch it, and the struct cannot hold it.
// A prompt-level assertion alone would pass forever on a payload nobody put a brief into; these two
// fail the moment an edit makes the brief reachable, before any prompt is assembled.
func TestWorldStatement_CannotReachTheBrief(t *testing.T) {
	if strings.Contains(strings.ToLower(worldStatementQuery), "brief") {
		t.Fatalf("(q) worldStatementQuery selects the brief — it is operational provenance, never rendered "+
			"(core/db/schema.sql:4234):\n%s", worldStatementQuery)
	}

	// Every field is document-derived and named here. A new field is a deliberate act that has to be
	// added to this list, which is where someone reads why the brief is not among them.
	allowed := map[string]bool{"Name": true, "Premise": true, "Mood": true, "Ornament": true, "Region": true}
	ty := reflect.TypeOf(WorldStatement{})
	for i := range ty.NumField() {
		if !allowed[ty.Field(i).Name] {
			t.Fatalf("(q) WorldStatement grew an undeclared field %q — every field must be the committed "+
				"document's own content, and the brief is not", ty.Field(i).Name)
		}
	}
}

// (r) The live half, against real Postgres: author and commit a world from a brief whose tail is a
// sentinel the authored document provably does not contain, then assemble all three narrate prompts
// from the statement the production loader actually returns. The document's own tagline must be there
// (or the test is vacuous); the brief must not be anywhere.
func TestWorldStatement_LoadedFromTheDocumentAndNeverTheBrief(t *testing.T) {
	ctx := context.Background()

	// The sentinel sits AFTER the first four words on purpose: the fake genesis driver composes its
	// display_name from a four-word slug of the brief, so a check on the brief's opening would report a
	// leak the production path does not have. What is under test is the brief's own prose reaching the
	// narrator, which nothing in the document ever restates.
	const sentinel = "the ledger keeper skims crates by candlelight and bills the tide for it"
	const brief = "A cargo yard. " + sentinel + "."

	doc, _, err := authorWorld(ctx, NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), brief, nil, nil, nil)
	if err != nil {
		t.Fatalf("authorWorld: %v", err)
	}
	pool := testPool(t)
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	worldID, err := commitWorldContent(ctx, tx, doc, nil, brief, "")
	if err != nil {
		t.Fatalf("commitWorldContent: %v", err)
	}

	// The fixture is only meaningful if the brief really is on the row to leak.
	var stored string
	if err := tx.QueryRow(ctx, `SELECT brief FROM world WHERE world_id=$1::uuid`, worldID).Scan(&stored); err != nil {
		t.Fatalf("read brief: %v", err)
	}
	if stored != brief {
		t.Fatalf("(r) fixture is not meaningful: world.brief = %q, want the brief that authored it", stored)
	}

	ws, err := loadWorldStatement(ctx, tx, worldID)
	if err != nil {
		t.Fatalf("loadWorldStatement: %v", err)
	}
	if ws.Empty() {
		t.Fatal("(r) a committed world produced an empty statement — the narrator would be told nothing")
	}
	// Every field is what the DOCUMENT authored, read back through the production query.
	for _, c := range []struct{ got, want, field string }{
		{ws.Name, doc.World.DisplayName, "display_name"},
		{ws.Premise, doc.World.Tagline, "tagline"},
		{ws.Mood, doc.World.Mood, "mood"},
		{ws.Ornament, doc.World.Ornament, "ornament"},
		{ws.Region, doc.Region.Descriptor, "region descriptor"},
	} {
		if c.got != strings.TrimSpace(c.want) {
			t.Fatalf("(r) %s = %q, want the committed document's %q", c.field, c.got, c.want)
		}
	}

	payload := PerceptionPayload{World: ws, Lines: []string{"Mara sets a tankard on the bar."}}
	for name, prompt := range map[string]string{
		"buildNarratePrompt":       buildNarratePrompt(payload, "", nil, ""),
		"buildNarrateRepairPrompt": buildNarrateRepairPrompt(payload, "", nil, "", "a ghost speaker"),
		"buildNarratePlainPrompt":  buildNarratePlainPrompt(payload, "", nil, ""),
	} {
		if !strings.Contains(prompt, doc.World.Tagline) {
			t.Fatalf("(r) %s does not carry the document's own premise, so the brief check below proves nothing:\n%s", name, prompt)
		}
		if strings.Contains(prompt, brief) {
			t.Fatalf("(r) %s carries world.brief verbatim", name)
		}
		if strings.Contains(prompt, sentinel) {
			t.Fatalf("(r) %s leaked the brief's own prose (%q) into the narrator's prompt", name, sentinel)
		}
	}
}
