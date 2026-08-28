package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScheduleWork_PlacesThenHistoryThenLivesThenObjectsThenRevise(t *testing.T) {
	got := scheduleWork(&worldIdentity{})
	want := []string{"places", "history", "lives", "objects", "revise", "sufficiency"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Errorf("item %d = %q, want %q", i, got[i].ID, want[i])
		}
	}
}

func TestAuthorWorld_StoresIdentityBesideTheDocument(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	t.Cleanup(pool.Close)
	doc, ident, err := authorWorld(ctx, NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ident == nil || ident.Bargain.Text == "" {
		t.Fatal("identity missing")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	worldID, err := commitWorldContent(ctx, tx, doc, ident, testBrief, "")
	if err != nil {
		t.Fatal(err)
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT world_identity FROM world WHERE world_id=$1::uuid`, worldID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("world_identity was not stored")
	}
}

func TestFillOne_RefusesUnknownFields(t *testing.T) {
	seat := stubDriver{raw: `{"empty":false,"places":[],"not_a_field":true}`}
	_, err := fillOne(context.Background(), seat, &worldIdentity{}, workItem{ID: "r", Kind: "generative", Text: "t", Therefore: "t"}, testBrief, nil, &genesisDoc{}, "")
	if err == nil || !IsGenesisRefusal(err) {
		t.Fatalf("want refusal for extra keys, got %v", err)
	}
}

func TestFillFromIdentity_PeopleHideAndBatchesCloseTheBelt(t *testing.T) {
	doc, ident, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ident == nil {
		t.Fatal("identity missing")
	}
	if len(doc.Cast) < 2 {
		t.Fatalf("cast=%d", len(doc.Cast))
	}
	for _, a := range doc.Cast {
		if strings.TrimSpace(a.Hiding) == "" {
			t.Errorf("%s has no hiding", a.CanonicalName)
		}
	}
	if strings.TrimSpace(doc.World.DisplayName) == "" {
		t.Fatal("sufficiency did not name the world")
	}
}

func TestFakeFill_LivesBatchIsAFillFragmentNotAGenesisDump(t *testing.T) {
	raw, err := NewFakeWorldFillDriver().Generate(context.Background(), GenRequest{
		Prompt: "\nid: lives\nkind: batch\n" + worldGenesisBriefMarker + "\n" + testBrief,
		Schema: json.RawMessage(worldFillSchemaJSON),
	})
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var frag fillFragment
	if err := dec.Decode(&frag); err != nil {
		t.Fatalf("fragment is not world_fill/1: %v\n%s", err, raw)
	}
	if frag.Empty || len(frag.Cast) == 0 {
		t.Fatalf("lives batch should author people, empty=%v cast=%d", frag.Empty, len(frag.Cast))
	}
}

func TestRetractBreaches_DropsNamedObject(t *testing.T) {
	doc := &genesisDoc{}
	doc.Objects = []genesisObject{{CanonicalName: "The Yard Key"}, {CanonicalName: "The Ledger"}}
	retractBreaches(doc, []fillBreach{{Kind: "object", Name: "The Yard Key", Why: "the neighbour's prop"}})
	if len(doc.Objects) != 1 || doc.Objects[0].CanonicalName != "The Ledger" {
		t.Fatalf("objects=%v", doc.Objects)
	}
}

// The review seat must actually be CONSULTED, and its verdict must actually land. Without this the
// whole `if review != nil` block in fillFromIdentity is deletable: the live fake finds no breaches,
// so every other test passes with the review wired to nothing. Revert the block and watch this fail.
func TestFillFromIdentity_ReviewRetractsTheBreachItNames(t *testing.T) {
	review := stubDriver{raw: `{"breaches":[{"kind":"object","name":"The Yard Key","why":"a prop the departure ruled out"}]}`}
	doc, _, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), review, testBrief, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range doc.Objects {
		if o.CanonicalName == "The Yard Key" {
			t.Fatal("the review named The Yard Key a breach and it survived into the committed world")
		}
	}
	// Retraction is scoped, never a purge: what the review did NOT name has to still be there, and the
	// belt has to still pass — a review that empties the world is not a review, it is a refusal.
	if len(doc.Objects) == 0 {
		t.Fatal("retraction took every object with it")
	}
	if err := doc.validate(); err != nil {
		t.Fatalf("the world stopped being playable after retraction: %v", err)
	}
}

type stubDriver struct{ raw string }

func (s stubDriver) Name() string                { return "stub-fill" }
func (s stubDriver) Capabilities() CapabilitySet { return CapabilitySet{CapStructuredOutput: true} }
func (s stubDriver) Generate(context.Context, GenRequest) (string, error) {
	return s.raw, nil
}

func TestIdentityConfirm_RoundTripsIntoFill(t *testing.T) {
	ctx := context.Background()
	raw, err := NewFakeWorldUnderstandingDriver().Generate(ctx, GenRequest{
		Prompt: worldIdentityBriefMarker + "\n" + testBrief,
		Schema: json.RawMessage(worldIdentitySchemaJSON),
	})
	if err != nil {
		t.Fatal(err)
	}
	voice := []string{"The page is wet.", "The lamp is out.", "Someone still writes."}
	doc, ident, err := authorWorld(ctx, NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, json.RawMessage(raw), voice)
	if err != nil {
		t.Fatal(err)
	}
	if ident.Voice[0] != voice[0] || ident.Voice[2] != voice[2] {
		t.Fatalf("voice rewrite lost: %#v", ident.Voice)
	}
	if strings.TrimSpace(doc.World.DisplayName) == "" {
		t.Fatal("fill did not run after confirmed identity")
	}
}

func TestIdentityHandler_OmitsTheTwenty(t *testing.T) {
	bridge, err := NewBridgeWithDrivers(map[string]Driver{
		SeatWorldUnderstanding.Name: NewFakeWorldUnderstandingDriver(),
	}, SeatWorldUnderstanding)
	if err != nil {
		t.Fatal(err)
	}
	h := NewWorldGenesisHandler(nil, false, bridge, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, jsonPost("/worlds/identity", `{"brief":"`+testBrief+`"}`))
	if rec.Code != 200 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["schema_version"] != "world_identity_confirm/1" {
		t.Fatalf("schema %v", body["schema_version"])
	}
	if _, ok := body["functions"]; ok {
		t.Fatal("confirmation leaked the twenty functions")
	}
	if _, ok := body["identity"]; !ok {
		t.Fatal("identity round-trip missing")
	}
	if body["register"] == "" {
		t.Fatal("register missing")
	}
}

// The Andantes refusal, 2026-08-28, as a test. A live build spent 234 seconds and refused because
// fill set starts_in to "Colegio de Auscultadores — Sede del Colegio de Auscultadores, en la cima
// del distrito" — the whole ALREADY AUTHORED line, name and descriptor joined by the em-dash the
// prompt itself put there. Place names in a real world contain dashes and commas, so the separator
// could never have been a boundary. The name is quoted now and nothing shares its line.
func TestFillPrompt_CrossReferenceNamesCannotSwallowTheDescriptor(t *testing.T) {
	name := "Colegio de Auscultadores"
	desc := "Sede del Colegio de Auscultadores, en la cima del distrito"
	soFar := &genesisDoc{}
	soFar.Places = []genesisPlace{{CanonicalName: name, Descriptor: desc}}
	soFar.Cast = []genesisActor{{CanonicalName: "Auscultadora Mayor Del Vas", Hiding: "she already knows", StartsIn: name}}

	prompt := buildWorldFillPrompt(&worldIdentity{}, workItem{ID: "lives", Kind: "batch"}, testBrief, nil, soFar, "")

	// The exact string the model emitted must not be sitting in the prompt for it to copy.
	if joined := name + " — " + desc; strings.Contains(prompt, joined) {
		t.Fatalf("the prompt still offers name and descriptor as one string: %q", joined)
	}
	if !strings.Contains(prompt, `- place "`+name+`"`) {
		t.Fatalf("the place's canonical_name is not quoted on its own; prompt:\n%s", prompt)
	}
	if !strings.Contains(prompt, "    looks like: "+desc) {
		t.Fatal("the descriptor is not on its own labelled line")
	}
	if !strings.Contains(prompt, `- person "Auscultadora Mayor Del Vas"`) {
		t.Fatal("the person's canonical_name is not quoted on its own")
	}
}

// The revise refusal, 2026-08-28. A live build died at 168s because the revise batch answered
// {"empty":false} with no entities, and the fragment validator called that a refusal. `empty` is a
// self-report; the content is the fact. A second pass with nothing to add is a legitimate answer.
func TestFillFragment_SelfContradictoryEmptyFlagIsNotARefusal(t *testing.T) {
	// Says non-empty, carries nothing — the exact live shape.
	nothing := &fillFragment{}
	if err := nothing.validate(); err != nil {
		t.Fatalf("a fragment with nothing in it must not refuse the build: %v", err)
	}

	// Says empty, carries places — the content must still land, not be dropped on the flag.
	frag := &fillFragment{Empty: true, WhyEmpty: "nothing to add"}
	frag.Places = []genesisPlace{{CanonicalName: "The Counting Room", Descriptor: "one lamp"}}
	doc := &genesisDoc{}
	var tags []taggedName
	mergeFill(doc, frag, "revise", &tags)
	if len(doc.Places) != 1 {
		t.Fatalf("content was dropped because the flag said empty; places=%d", len(doc.Places))
	}

	// Content that IS present is still held to the real standard.
	bad := &fillFragment{}
	bad.Cast = []genesisActor{{CanonicalName: "Adaeze", Hiding: ""}}
	if err := bad.validate(); err == nil {
		t.Fatal("a person with no hiding must still be refused")
	}
}

// The forward-reference refusal, 2026-08-28. history named "Auscultadora Mayor Del Vas", the lives
// batch never authored her, and the belt refused the world 227 seconds later — the one repair call
// answered in 59 tokens and could not fix it. The order (places -> history -> lives) is the founder's
// and is right; what was missing is that the debt was only a sentence in the prompt. Now it is data.
//
// Note WHY no fake caught this: the fill fake is self-consistent by construction — its history names
// exactly the two people its lives batch authors — so it can never produce a forward reference. A
// stand-in that cannot express the failure cannot defend against it.
func TestFillPrompt_CanonsUnpaidNamesAreCarriedForward(t *testing.T) {
	soFar := &genesisDoc{}
	soFar.Places = []genesisPlace{{CanonicalName: "El Lomo", Descriptor: "the back of the beast"}}
	soFar.History = []genesisEvent{{
		WhatHappened: "she heard the beast's heart stumble and wrote nothing down",
		Where:        "El Lomo",
		Who:          []string{"Auscultadora Mayor Del Vas"},
		Knowledge: []genesisKnowledge{
			{Holder: "Auscultadora Mayor Del Vas", Content: "the rhythm changed", EpistemicType: "direct"},
			{Holder: "El Cartografo", Content: "she has stopped reporting", EpistemicType: "inference"},
		},
	}}

	people, places := fillDebts(soFar)
	if len(places) != 0 {
		t.Fatalf("El Lomo is authored; nothing should be owed: %v", places)
	}
	if len(people) != 2 {
		t.Fatalf("both named holders are owed, got %v", people)
	}

	prompt := buildWorldFillPrompt(&worldIdentity{}, workItem{ID: "lives", Kind: "batch"}, testBrief, nil, soFar, "")
	if !strings.Contains(prompt, worldFillOwedMarker) {
		t.Fatal("the lives batch is not told what canon already owes")
	}
	for _, want := range []string{`person "Auscultadora Mayor Del Vas"`, `person "El Cartografo"`} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("owed name missing from prompt: %s", want)
		}
	}

	// Once authored, the debt disappears — the block must not nag about paid debts.
	soFar.Cast = []genesisActor{
		{CanonicalName: "Auscultadora Mayor Del Vas", Hiding: "she never filed it", StartsIn: "El Lomo"},
		{CanonicalName: "El Cartografo", Hiding: "he has already redrawn the route", StartsIn: "El Lomo"},
	}
	if people, places := fillDebts(soFar); len(people) != 0 || len(places) != 0 {
		t.Fatalf("debt survived being paid: people=%v places=%v", people, places)
	}
}

// flakyFillDriver rejects once, then behaves. Stands in for the live failure of 2026-08-28, where the
// revise batch spelled `places` as `place` and DisallowUnknownFields ended a 147-second build.
type flakyFillDriver struct {
	real  Driver
	calls int
}

func (f *flakyFillDriver) Name() string { return "flaky-fill" }
func (f *flakyFillDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (f *flakyFillDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	f.calls++
	if f.calls == 1 {
		// A key the schema does not name — exactly what DisallowUnknownFields refuses.
		return `{"empty":false,"place":[{"canonical_name":"The Counting Room"}]}`, nil
	}
	return f.real.Generate(ctx, req)
}

func TestFillFromIdentity_OneMalformedBatchIsRetriedNotFatal(t *testing.T) {
	seat := &flakyFillDriver{real: NewFakeWorldFillDriver()}
	id, err := inferIdentity(context.Background(), NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(context.Background(), seat, NewFakeWorldFillReviewDriver(), id, testBrief, nil)
	if err != nil {
		t.Fatalf("one malformed batch killed the build instead of being retried: %v", err)
	}
	if err := doc.validate(); err != nil {
		t.Fatalf("the retried build is not playable: %v", err)
	}
	// 6 batches + 1 wasted first attempt. If this is 6, no retry happened and the test proves nothing.
	if seat.calls < 7 {
		t.Fatalf("expected a retry call, got %d generate calls", seat.calls)
	}

	// And the rejection must actually be quoted back to the seat, not silently retried.
	prompt := buildWorldFillPrompt(id, workItem{ID: "revise", Kind: "batch"}, testBrief, nil, &genesisDoc{}, `json: unknown field "place"`)
	if !strings.Contains(prompt, worldFillRejectedMarker) || !strings.Contains(prompt, `unknown field "place"`) {
		t.Fatal("the retry does not tell the seat what was wrong")
	}
}

// The dangling-reference class, caught at the batch instead of at the belt. Live 2026-08-28: the lives
// batch authored "Sento" with starts_in "Cola Baja", a place no batch ever wrote, and the belt refused
// the build 177 seconds later. Seven of eight live refusals that day were this same class.
func TestFillFragment_DanglingReferencesAreCaughtAtTheBatch(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{{CanonicalName: "El Lomo", Descriptor: "the living back"}}

	// The exact live shape: a person standing in a place nobody authored.
	frag := &fillFragment{}
	frag.Cast = []genesisActor{{CanonicalName: "Sento", Hiding: "he has not reported the tremor", StartsIn: "Cola Baja"}}
	bad := frag.danglingRefs(doc)
	if len(bad) != 1 || !strings.Contains(bad[0], `"Cola Baja"`) {
		t.Fatalf("the dangling place was not reported: %v", bad)
	}

	// A fragment that authors the place it stands in is fine — resolution includes the fragment itself.
	ok := &fillFragment{}
	ok.Places = []genesisPlace{{CanonicalName: "Cola Baja", Descriptor: "the low tail"}}
	ok.Cast = []genesisActor{{CanonicalName: "Sento", Hiding: "he has not reported it", StartsIn: "Cola Baja"}}
	if bad := ok.danglingRefs(doc); len(bad) != 0 {
		t.Fatalf("a self-contained fragment was rejected: %v", bad)
	}

	// History naming a person lives has not written yet is the DELIBERATE forward debt (fillDebts),
	// never a dangling reference. If this starts failing, the batch order has been broken.
	fwd := &fillFragment{}
	fwd.History = []genesisEvent{{
		WhatHappened: "the rhythm changed and nobody wrote it down",
		Where:        "El Lomo",
		Who:          []string{"Nobody Yet Authored"},
		Knowledge:    []genesisKnowledge{{Holder: "Nobody Yet Authored", Content: "she heard it", EpistemicType: "direct"}},
	}}
	if bad := fwd.danglingRefs(doc); len(bad) != 0 {
		t.Fatalf("history's forward reference to a person was wrongly treated as dangling: %v", bad)
	}

	// And it is a REFUSAL out of fillOne, which is what makes the one-shot retry fire.
	seat := stubDriver{raw: `{"empty":false,"cast":[{"descriptor":"a man","canonical_name":"Sento","standing":"s","speech_manner":"m","hiding":"h","malleability":"faint","starts_in":"Cola Baja"}]}`}
	_, err := fillOne(context.Background(), seat, &worldIdentity{}, workItem{ID: "lives", Kind: "batch"}, testBrief, nil, doc, "")
	if err == nil || !IsGenesisRefusal(err) {
		t.Fatalf("want a refusal the retry can act on, got %v", err)
	}
	if !strings.Contains(err.Error(), "Cola Baja") {
		t.Fatalf("the refusal does not name the offending place: %v", err)
	}
}
