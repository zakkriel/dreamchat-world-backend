package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestSchedule_NamespaceThenContent(t *testing.T) {
	// The namespace is sequential and its order is the design: what a world argues over decides what its
	// places, factions and people are, so concepts sit above geography.
	if conceptsWork().ID != "concepts" {
		t.Fatal("concepts is not the first namespace call")
	}
	b := budgetForDepth(0)
	if b.Level != 1 {
		t.Fatalf("omitted depth is %d, want 1 — nothing raises it for the user yet", b.Level)
	}

	// Scaffold 2 is sliced by top location AND by faction. Slicing only by location was my design and
	// the founder's correction: it made factions second-class and grouped people by geography.
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{
		{CanonicalName: "Top A", Relevance: 2},
		{CanonicalName: "Top B", Relevance: 2},
		{CanonicalName: "Inner A", Within: "Top A", Relevance: 1},
	}
	doc.Factions = []genesisFaction{{CanonicalName: "The Tally", Relevance: 2}}
	if got := topLocations(doc); len(got) != 2 {
		t.Fatalf("topLocations found %d, want 2 — `within` is what makes a place a root", len(got))
	}
	if tree := locationTree(doc, "Top A"); len(tree) != 2 {
		t.Fatalf("Top A's tree is %v, want it to contain Inner A", tree)
	}
	items := scaffoldTwoSchedule(doc, b)
	if len(items) != 3 {
		t.Fatalf("scaffold 2 emitted %d items, want 3 (two locations and a faction)", len(items))
	}
	for _, it := range items {
		if it.Subject == "" {
			t.Fatal("a scaffold-2 item has no subject, so it cannot be scoped")
		}
		if it.Scope.Whole {
			t.Fatalf("scaffold-2 %q sees the whole document — the compiled mandate is the saving", it.Subject)
		}
	}

	// THE SAVING: a relevance-1 person is complete and appears in no content call at all.
	doc.Cast = []genesisActor{
		{CanonicalName: "Thin One", Relevance: 1, Tag: "t", StartsIn: "Top A"},
		{CanonicalName: "Owed Two", Relevance: 2, Tag: "t", StartsIn: "Top A"},
		{CanonicalName: "Owed Three", Relevance: 3, Tag: "t", StartsIn: "Top A"},
	}
	packs := peoplePacks(doc, nil)
	named := map[string]bool{}
	for _, it := range packs {
		for _, m := range it.Members {
			named[m] = true
		}
	}
	if named["Thin One"] {
		t.Fatal("a relevance-1 person was scheduled for content — they are already complete, and skipping them is the whole design")
	}
	if !named["Owed Two"] || !named["Owed Three"] {
		t.Fatalf("people owed content were not scheduled: %v", named)
	}

	// Packs cap at ten: the founder's number, and a cost decision rather than a quality one.
	doc.Cast = nil
	for i := 0; i < 23; i++ {
		doc.Cast = append(doc.Cast, genesisActor{
			CanonicalName: "P" + strconv.Itoa(i), Relevance: 2, Tag: "t", StartsIn: "Top A",
		})
	}
	packs = peoplePacks(doc, nil)
	if len(packs) != 3 {
		t.Fatalf("23 owed people made %d packs, want 3", len(packs))
	}
	for _, it := range packs {
		if len(it.Members) > 10 {
			t.Fatalf("a pack carries %d people, over the cap of 10", len(it.Members))
		}
	}
	if mergeTag(workItem{ID: "people", Subject: "Top A"}) != "people:Top A" {
		t.Fatal("per-item work is not distinguishable for retraction")
	}
}

// A thing named thin and deepened later is the ENTIRE layered design, so the merge must deepen rather
// than skip. It skipped for places, factions, concepts and objects — only actors and events deepened —
// which would have discarded every description the content wave wrote and then failed the belt for
// missing it.
func TestMergeFill_DeepensAThingTheScaffoldNamedThin(t *testing.T) {
	doc := &genesisDoc{}
	var tags []taggedName
	mergeFill(doc, &fillFragment{Places: []genesisPlace{{
		CanonicalName: "The Counting Room", Descriptor: "a low room", Kind: "back room",
		ExtentClass: "intimate", Relevance: 2,
	}}}, "scaffold-1", &tags)
	mergeFill(doc, &fillFragment{Places: []genesisPlace{{
		CanonicalName: "The Counting Room", Descriptor: "SHOULD NOT WIN", Kind: "back room",
		ExtentClass: "intimate", Relevance: 1, Description: "One lamp over a table.", Tension: "tense",
	}}}, "geography", &tags)
	if len(doc.Places) != 1 {
		t.Fatalf("the same place merged twice into %d rows", len(doc.Places))
	}
	got := doc.Places[0]
	if got.Description == "" || got.Tension == "" {
		t.Fatal("the content wave's description was discarded — the belt would refuse a world that was authored")
	}
	if got.Descriptor != "a low room" {
		t.Fatalf("the namespace's descriptor was overwritten with %q — every reference resolved against it", got.Descriptor)
	}
	if got.Relevance != 2 {
		t.Fatalf("relevance fell to %d; it is a ratchet (ADR-P027) and content authored at 2 would become unvalidatable", got.Relevance)
	}
}

func TestAuthorWorld_StoresIdentityBesideTheDocument(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	t.Cleanup(pool.Close)
	doc, ident, err := authorWorld(ctx, NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, nil, nil, 0)
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
	doc, ident, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ident == nil {
		t.Fatal("identity missing")
	}
	if len(doc.Cast) < 2 {
		t.Fatalf("cast=%d", len(doc.Cast))
	}
	// Hiding is what being REFERENCED buys (ADR-P027): anyone a scene can turn to holds one thing they
	// will not say, and a person at relevance 1 holds nothing. Both directions matter — a fake that
	// authored everyone richly would pass the first check and hide the entire design.
	thin, deep := 0, 0
	for _, a := range doc.Cast {
		switch {
		case a.Relevance >= 2:
			deep++
			if strings.TrimSpace(a.Hiding) == "" {
				t.Errorf("%s is relevance %d and hides nothing", a.CanonicalName, a.Relevance)
			}
		default:
			thin++
			if strings.TrimSpace(a.Hiding) != "" {
				t.Errorf("%s is relevance 1 but carries a secret — depth was spent on someone nobody has met", a.CanonicalName)
			}
		}
	}
	if thin == 0 {
		t.Error("every person came back at relevance 2 or more, so the build paid for depth it was not asked for")
	}
	if deep == 0 {
		t.Error("nobody reached relevance 2, so no scene in this world can turn to anyone")
	}
	if strings.TrimSpace(doc.World.DisplayName) == "" {
		t.Fatal("sufficiency did not name the world")
	}
}

func TestFakeFill_PeopleLayerIsAFillFragmentNotAGenesisDump(t *testing.T) {
	raw, err := NewFakeWorldFillDriver().Generate(context.Background(), GenRequest{
		Prompt: "\nid: people\nkind: content\nauthor exactly these and nobody else:\n  - \"Del Vas\" at relevance 3\n" +
			worldGenesisBriefMarker + "\n" + testBrief,
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
		t.Fatalf("people layer should author the roster, empty=%v cast=%d", frag.Empty, len(frag.Cast))
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
	doc, _, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), review, testBrief, nil, nil, nil, 0)
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
	doc, ident, err := authorWorld(ctx, NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), testBrief, nil, json.RawMessage(raw), voice, 0)
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

	prompt := buildWorldFillPrompt(&worldIdentity{}, workItem{
		ID: "people", Kind: "content",
		Scope: fillScope{Places: []string{name}, People: []string{"Auscultadora Mayor Del Vas"}},
	}, testBrief, nil, soFar, "")

	// The exact string the model emitted must not be sitting in the prompt for it to copy.
	if joined := name + " — " + desc; strings.Contains(prompt, joined) {
		t.Fatalf("the prompt still offers name and descriptor as one string: %q", joined)
	}
	if !strings.Contains(prompt, `- location "`+name+`"`) {
		t.Fatalf("the location's canonical_name is not quoted on its own; prompt:\n%s", prompt)
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

	prompt := buildWorldFillPrompt(&worldIdentity{}, workItem{ID: "people", Kind: "descent"}, testBrief, nil, soFar, "")
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
	real Driver
	// The ascent runs its per-person calls together, so a stand-in that counts them is shared across
	// goroutines. The race detector caught this on the first run.
	mu    sync.Mutex
	calls int
}

func (f *flakyFillDriver) Name() string { return "flaky-fill" }
func (f *flakyFillDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (f *flakyFillDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	f.mu.Lock()
	f.calls++
	n := f.calls
	f.mu.Unlock()
	if n == 1 {
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
	doc, err := fillFromIdentity(context.Background(), seat, NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatalf("one malformed batch killed the build instead of being retried: %v", err)
	}
	if err := doc.validate(); err != nil {
		t.Fatalf("the retried build is not playable: %v", err)
	}
	// 6 batches + 1 wasted first attempt. If this is 6, no retry happened and the test proves nothing.
	seat.mu.Lock()
	calls := seat.calls
	seat.mu.Unlock()
	if calls < 7 {
		t.Fatalf("expected a retry call, got %d generate calls", calls)
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
	frag.Cast = []genesisActor{{CanonicalName: "Sento", Relevance: 1, Tag: "answers late", Hiding: "he has not reported the tremor", StartsIn: "Cola Baja"}}
	bad := frag.danglingRefs(doc)
	if len(bad) != 1 || !strings.Contains(bad[0], `"Cola Baja"`) {
		t.Fatalf("the dangling place was not reported: %v", bad)
	}

	// A fragment that authors the place it stands in is fine — resolution includes the fragment itself.
	ok := &fillFragment{}
	ok.Places = []genesisPlace{{CanonicalName: "Cola Baja", Descriptor: "the low tail"}}
	ok.Cast = []genesisActor{{CanonicalName: "Sento", Relevance: 1, Tag: "answers late", Hiding: "he has not reported it", StartsIn: "Cola Baja"}}
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

	// And it is NOT fatal. Founder 2026-08-28: a good invention is never discarded for arriving before
	// its room. fillOne hands the fragment back; the debt is carried and the closing pass authors it.
	seat := stubDriver{raw: `{"empty":false,"cast":[{"descriptor":"a man","canonical_name":"Sento","standing":"s","speech_manner":"m","hiding":"h","malleability":"faint","starts_in":"Cola Baja","relevance":1,"tag":"answers late"}]}`}
	got, err := fillOne(context.Background(), seat, &worldIdentity{}, workItem{ID: "people", Kind: "descent"}, testBrief, nil, doc, "")
	if err != nil {
		t.Fatalf("an unpaid reference must not lose the fragment: %v", err)
	}
	if len(got.Cast) != 1 || got.Cast[0].CanonicalName != "Sento" {
		t.Fatal("Sento was discarded rather than kept for the closing pass")
	}
	// Merged in, the debt is visible to fillDebts and therefore to the closing pass.
	var tags []taggedName
	mergeFill(doc, got, "lives", &tags)
	if _, places := fillDebts(doc); len(places) != 1 || places[0] != "Cola Baja" {
		t.Fatalf("the unpaid place is not queued for the closing pass: %v", places)
	}
}

// The closing pass authors what canon owes, rather than the build refusing it. This is the mechanism
// the founder asked for on 2026-08-28: creation, not constriction.
func TestFillFromIdentity_ClosingPassAuthorsWhatCanonOwes(t *testing.T) {
	// A fill seat whose lives batch stands someone in a room it never authors, and whose closing batch
	// pays that debt — the live Andantes shape.
	seat := &owingFillDriver{real: NewFakeWorldFillDriver()}
	id, err := inferIdentity(context.Background(), NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(context.Background(), seat, NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatalf("the build refused instead of authoring the owed place: %v", err)
	}
	seat.mu.Lock()
	closed := seat.closed
	seat.mu.Unlock()
	if !closed {
		t.Fatal("the closing pass never ran")
	}
	if !hasPlace(doc, "Cola Baja") {
		t.Fatal("the owed place was never authored")
	}
	if people, places := fillDebts(doc); len(people) > 0 || len(places) > 0 {
		t.Fatalf("debts survived the closing pass: people=%v places=%v", people, places)
	}
	if err := doc.validate(); err != nil {
		t.Fatalf("closed world is not playable: %v", err)
	}
}

// owingFillDriver adds an unpaid place reference on the lives batch, and authors it when the closing
// batch asks. Everything else delegates to the ordinary fake.
type owingFillDriver struct {
	real   Driver
	mu     sync.Mutex
	closed bool
}

func (o *owingFillDriver) Name() string { return "owing-fill" }
func (o *owingFillDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (o *owingFillDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	switch {
	case strings.Contains(req.Prompt, "\nid: people\n"):
		// Keep everything the ordinary fake authors and ADD one life standing in a room nobody wrote.
		// Replacing the batch would empty the arrival neighbourhood and test the wrong failure.
		raw, err := o.real.Generate(ctx, req)
		if err != nil {
			return "", err
		}
		var frag map[string]any
		if err := json.Unmarshal([]byte(raw), &frag); err != nil {
			return "", err
		}
		cast, _ := frag["cast"].([]any)
		frag["cast"] = append(cast, map[string]any{
			"descriptor": "a man on the low tail", "canonical_name": "Sento", "relevance": 1, "tag": "answers late",
			"standing": "walks the tail", "speech_manner": "few words",
			"traits": []any{map[string]any{
				"key": "watchful", "strength": "strong",
				"manner": "listens to the ground before he answers",
			}},
			"hiding":       "he has not reported the tremor",
			"malleability": "faint", "starts_in": "Cola Baja",
		})
		out, err := json.Marshal(frag)
		if err != nil {
			return "", err
		}
		return string(out), nil
	case strings.Contains(req.Prompt, "\nid: closing\n"):
		o.mu.Lock()
		o.closed = true
		o.mu.Unlock()
		return `{"empty":false,"places":[{"descriptor":"the low tail, awash","canonical_name":"Cola Baja","kind":"district","description":"The tail drags and the water comes over it twice a day; nobody builds high here.","tension":"normal","extent_class":"small","relevance":1,"tag":"spoken of before it was seen"}],"ways":[{"descriptor":"the climb up the spine","from_place":"Cola Baja","to_place":"The Counting Room","state":"open"}]}`, nil
	}
	return o.real.Generate(ctx, req)
}

// The last-check refusal, 2026-08-28. A 550-second build authored a good arrival and three good
// candidates, none of which was the arrival, and the belt threw the whole world away on the final
// rule before commit. The arrival IS the recommendation; it takes a seat among its own alternatives.
func TestReconcileArrival_SeatsTheArrivalAmongItsCandidates(t *testing.T) {
	doc := &genesisDoc{}
	doc.Arrival = genesisArrival{
		Descriptor: "a stranger off the tail", CanonicalName: "Wren",
		Place: "El Lomo", Stated: "I came up the spine.", Why: "sent to read a heart nobody will discuss",
	}
	doc.ArrivalCandidates = []genesisCandidate{
		{Descriptor: "a courier", CanonicalName: "Petra", Why: "carrying a delivery the ledger has no line for"},
		{Descriptor: "a man in a borrowed coat", CanonicalName: "Osei", Why: "came to the wrong address"},
		{Descriptor: "a girl with a listening horn", CanonicalName: "Ise", Why: "apprenticed to a trade nobody wants"},
	}
	reconcileArrival(doc)

	if len(doc.ArrivalCandidates) != 3 {
		t.Fatalf("want exactly three candidates, got %d", len(doc.ArrivalCandidates))
	}
	if doc.ArrivalCandidates[0].CanonicalName != "Wren" {
		t.Fatalf("the arrival is not the recommended default: %q", doc.ArrivalCandidates[0].CanonicalName)
	}
	// Two of the seat's own alternatives must survive — we reconcile, we do not replace its work.
	kept := map[string]bool{}
	for _, c := range doc.ArrivalCandidates {
		kept[c.CanonicalName] = true
	}
	if !kept["Osei"] || !kept["Ise"] {
		t.Fatalf("the seat's alternatives were discarded: %v", kept)
	}

	// A coherent offer is left exactly as authored.
	doc2 := &genesisDoc{}
	doc2.Arrival = genesisArrival{Descriptor: "d", CanonicalName: "Wren", Place: "p", Stated: "s", Why: "w"}
	doc2.ArrivalCandidates = []genesisCandidate{
		{Descriptor: "d", CanonicalName: "Wren", Why: "w"},
		{Descriptor: "d", CanonicalName: "Petra", Why: "w"},
		{Descriptor: "d", CanonicalName: "Osei", Why: "w"},
	}
	before := append([]genesisCandidate(nil), doc2.ArrivalCandidates...)
	reconcileArrival(doc2)
	if len(doc2.ArrivalCandidates) != 3 || doc2.ArrivalCandidates[0].CanonicalName != before[0].CanonicalName {
		t.Fatal("a coherent offer was disturbed")
	}

	// An offer that cannot be made coherent costs the LIST, never the world.
	doc3 := &genesisDoc{}
	doc3.Arrival = genesisArrival{Descriptor: "d", CanonicalName: "Wren", Place: "p", Stated: "s"}
	doc3.ArrivalCandidates = []genesisCandidate{{Descriptor: "d", CanonicalName: "Petra", Why: "w"}}
	reconcileArrival(doc3)
	if doc3.ArrivalCandidates != nil {
		t.Fatalf("an incomplete offer should be dropped, got %v", doc3.ArrivalCandidates)
	}
	if doc3.Arrival.CanonicalName != "Wren" {
		t.Fatal("the arrival itself must survive")
	}
}

// A 472-second build with 27,383 tokens of authored world was refused for "object 7 has no
// canonical_name". Objects and history events are leaves — nothing points at them — so an unstorable
// one costs itself, never the world.
func TestDropUnstorable_CostsTheLeafNotTheWorld(t *testing.T) {
	doc := &genesisDoc{}
	doc.Objects = []genesisObject{
		{CanonicalName: "The Listening Horn", Descriptor: "a brass cone", Kind: "instrument"},
		{CanonicalName: "", Descriptor: "something", Kind: "thing"},
		{CanonicalName: "The Ledger", Descriptor: "an open book", Kind: "record"},
		{CanonicalName: "No Kind", Descriptor: "a thing", Kind: ""},
	}
	doc.History = []genesisEvent{
		{WhatHappened: "the heart stumbled and nobody wrote it down"},
		{WhatHappened: "   "},
	}
	dropUnstorable(doc)

	if len(doc.Objects) != 2 {
		t.Fatalf("want the two storable objects, got %d", len(doc.Objects))
	}
	for _, o := range doc.Objects {
		if o.CanonicalName != "The Listening Horn" && o.CanonicalName != "The Ledger" {
			t.Fatalf("wrong object survived: %q", o.CanonicalName)
		}
	}
	if len(doc.History) != 1 {
		t.Fatalf("want the one real event, got %d", len(doc.History))
	}

	// Places and people are NOT leaves and are deliberately left alone — things stand in them and canon
	// names them, so the belt must keep refusing a nameless one plainly.
	doc2 := &genesisDoc{}
	doc2.Places = []genesisPlace{{CanonicalName: "", Descriptor: "nowhere"}}
	doc2.Cast = []genesisActor{{CanonicalName: "", Hiding: "h"}}
	dropUnstorable(doc2)
	if len(doc2.Places) != 1 || len(doc2.Cast) != 1 {
		t.Fatal("dropUnstorable must not silently remove places or people")
	}
}

// A 407-second Andantes build was refused over "once familias" — Spanish for "eleven families", a
// speakable collective in a Spanish-language world. identifierShapedName assumes English
// capitalisation. Capitalisation is typography, not content, so it is normalised — and every reference
// is rewritten with it, or the rename would orphan canon and refuse one check later.
func TestNormalisePersonNames_CapitalisesRatherThanRefusing(t *testing.T) {
	doc := &genesisDoc{}
	doc.Cast = []genesisActor{
		{CanonicalName: "once familias", Hiding: "they have not said which heart they heard"},
		{CanonicalName: "Adaeze", Hiding: "the last two pages are another hand"},
	}
	doc.History = []genesisEvent{{
		WhatHappened: "the rhythm changed and nobody filed it",
		Where:        "El Lomo",
		Who:          []string{"once familias", "Adaeze"},
		Knowledge: []genesisKnowledge{
			{Holder: "once familias", Content: "we felt it through the floor", EpistemicType: "direct"},
			{Holder: "Adaeze", Content: "they have stopped reporting", EpistemicType: "inference"},
		},
	}}
	horn := genesisObject{CanonicalName: "The Listening Horn", Descriptor: "a brass cone", Kind: "instrument"}
	horn.Where.CarriedBy = "once familias"
	doc.Objects = []genesisObject{horn}

	normalisePersonNames(doc)

	if doc.Cast[0].CanonicalName != "Once Familias" {
		t.Fatalf("name not normalised: %q", doc.Cast[0].CanonicalName)
	}
	if doc.Cast[1].CanonicalName != "Adaeze" {
		t.Fatalf("an already-capitalised name was disturbed: %q", doc.Cast[1].CanonicalName)
	}
	// Every reference must move with it, or canon points at somebody who no longer exists.
	if doc.History[0].Who[0] != "Once Familias" {
		t.Fatalf("history.who was orphaned: %v", doc.History[0].Who)
	}
	if doc.History[0].Knowledge[0].Holder != "Once Familias" {
		t.Fatalf("knowledge holder was orphaned: %q", doc.History[0].Knowledge[0].Holder)
	}
	if doc.Objects[0].Where.CarriedBy != "Once Familias" {
		t.Fatalf("carried_by was orphaned: %q", doc.Objects[0].Where.CarriedBy)
	}
	// And the normalised name now passes the guard that refused it.
	if identifierShapedName("Once Familias") {
		t.Fatal("the normalised name still reads as a join key")
	}

	// Underscores are NOT normalised — a slug is a signal, not a typo (the Ironmoor breach).
	slug := &genesisDoc{}
	slug.Cast = []genesisActor{{CanonicalName: "silas_holton", Hiding: "h"}}
	normalisePersonNames(slug)
	if slug.Cast[0].CanonicalName != "silas_holton" {
		t.Fatalf("a join key was quietly repaired instead of refused: %q", slug.Cast[0].CanonicalName)
	}
}

// The tick ladder must have room for the depth the fill schema can produce. A 499-second Andantes
// build was refused with "too much history to place before the world opens" because scene genesis sat
// ten ticks above the backstory base while the fill ceilings had been raised for depth. Ticks are
// logical (B-5, ADR-030), so the gap is spacing — but schema and ladder must not drift apart again.
func TestTickLadder_HasRoomForTheCanonFillCanAuthor(t *testing.T) {
	slots := genesisSceneTick - genesisBackstoryBaseTick
	if slots < 60 {
		t.Fatalf("only %d backstory slots between tick %d and %d — depth will hit the wall again",
			slots, genesisBackstoryBaseTick, genesisSceneTick)
	}
	if genesisArrivalTick <= genesisSceneTick {
		t.Fatalf("arrival (%d) must be the world's highest tick, above scene (%d)",
			genesisArrivalTick, genesisSceneTick)
	}
	if genesisNamingTick >= genesisBackstoryBaseTick {
		t.Fatalf("naming (%d) must precede the backstory (%d)", genesisNamingTick, genesisBackstoryBaseTick)
	}

	// And the belt must actually accept a deep canon now.
	doc := &genesisDoc{}
	for i := 0; i < 40; i++ {
		doc.History = append(doc.History, genesisEvent{WhatHappened: "something happened"})
	}
	if genesisBackstoryBaseTick+int64(len(doc.History)) > genesisSceneTick {
		t.Fatalf("40 authored events still do not fit before the world opens")
	}
}

// An 803-second build was refused at the BATCH for the cast name "un aprendiz de 27 años" ("a
// 27-year-old apprentice"), a rule the document-level normaliser would have satisfied a moment later.
// A fragment-level copy of a document-level rule just means the repair never gets to run.
func TestFillFragment_LeavesTheJoinKeyRuleToTheBeltAndTheNormaliser(t *testing.T) {
	frag := &fillFragment{}
	frag.Cast = []genesisActor{{CanonicalName: "un aprendiz de 27 años", Relevance: 1, Tag: "flinches at a raised ledger", Hiding: "he has not told anyone", StartsIn: "El Lomo"}}
	if err := frag.validate(); err != nil {
		t.Fatalf("the fragment refused a name the normaliser fixes: %v", err)
	}
	// And the normaliser does fix it, into something the belt accepts.
	doc := &genesisDoc{}
	doc.Cast = frag.Cast
	normalisePersonNames(doc)
	if identifierShapedName(doc.Cast[0].CanonicalName) {
		t.Fatalf("still join-key shaped after normalising: %q", doc.Cast[0].CanonicalName)
	}
	// Depth is still enforced at the fragment: a person with no private cost is still refused.
	bad := &fillFragment{}
	bad.Cast = []genesisActor{{CanonicalName: "Adaeze", Hiding: ""}}
	if err := bad.validate(); err == nil {
		t.Fatal("a person with no hiding must still be refused at the batch")
	}
}

// The arrival sharing a name with one of the world's own people, resolved from the seat's own authored
// alternatives rather than by refusing the world. Live 2026-08-28: caused first by the closing pass and
// then, 789 seconds in, by the repair pass — which is why it is resolved here rather than at each caller.
func TestResolveArrivalCollision_TakesAnAuthoredAlternative(t *testing.T) {
	doc := &genesisDoc{}
	doc.Cast = []genesisActor{
		{CanonicalName: "El Nuevo Aprendiz", Hiding: "he has not filed it", StartsIn: "El Lomo"},
		{CanonicalName: "Petra", Hiding: "she signed for it", StartsIn: "El Lomo"},
	}
	doc.Arrival = genesisArrival{
		Descriptor: "the new apprentice", CanonicalName: "El Nuevo Aprendiz",
		Place: "El Lomo", Stated: "I came up the spine.", Why: "sent to listen",
	}
	doc.ArrivalCandidates = []genesisCandidate{
		{Descriptor: "the new apprentice", CanonicalName: "El Nuevo Aprendiz", Why: "sent to listen"},
		{Descriptor: "Petra", CanonicalName: "Petra", Why: "already here"},
		{Descriptor: "a courier with wet boots", CanonicalName: "Ise", Why: "carrying an unlogged delivery"},
	}
	reconcileArrival(doc)

	if doc.Arrival.CanonicalName != "Ise" {
		t.Fatalf("the collision was not resolved onto a free authored alternative: %q", doc.Arrival.CanonicalName)
	}
	// The room and the opening line the arrival was authored with must survive the swap.
	if doc.Arrival.Place != "El Lomo" || doc.Arrival.Stated != "I came up the spine." {
		t.Fatalf("the arrival lost its room or its opening line: %+v", doc.Arrival)
	}
	if doc.Arrival.Descriptor != "a courier with wet boots" {
		t.Fatalf("the new arrival kept the old descriptor: %q", doc.Arrival.Descriptor)
	}
	// The world's own person is untouched — they are the side embedded in canon.
	if doc.Cast[0].CanonicalName != "El Nuevo Aprendiz" {
		t.Fatal("the cast member was changed instead of the visitor")
	}

	// No free alternative: the world is left for the belt to judge, not silently mangled.
	doc2 := &genesisDoc{}
	doc2.Cast = []genesisActor{{CanonicalName: "Solo", Hiding: "h", StartsIn: "p"}}
	doc2.Arrival = genesisArrival{Descriptor: "d", CanonicalName: "Solo", Place: "p", Stated: "s"}
	doc2.ArrivalCandidates = []genesisCandidate{{Descriptor: "d", CanonicalName: "Solo", Why: "w"}}
	resolveArrivalCollision(doc2)
	if doc2.Arrival.CanonicalName != "Solo" {
		t.Fatal("a name was invented where no authored alternative existed")
	}
}

// Canon decides the name; the newcomer yields. Canon is the record of what happened — append-only and
// immutable (D-1, I-1, I-2) — authored before the arrival and never edited to make a later choice fit.
// An earlier version of this pipeline stripped the visitor out of history instead. That was wrong: if
// something happened it stays recorded, and it is the arrival that changes.
func TestArrivalYieldsToCanon_HistoryIsNeverEdited(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{{CanonicalName: "El Lomo", Descriptor: "the living back"}}
	doc.Cast = []genesisActor{{CanonicalName: "Adaeze", Hiding: "she never filed it", StartsIn: "El Lomo"}}
	doc.History = []genesisEvent{{
		WhatHappened: "the heart stumbled and two people heard it",
		Where:        "El Lomo",
		Who:          []string{"Adaeze", "Elián"},
		Knowledge: []genesisKnowledge{
			{Holder: "Adaeze", Content: "the rhythm changed", EpistemicType: "direct"},
			{Holder: "Elián", Content: "I heard it too", EpistemicType: "direct"},
		},
	}}
	doc.Arrival = genesisArrival{
		Descriptor: "a newcomer", CanonicalName: "Elián",
		Place: "El Lomo", Stated: "I came up the spine.", Why: "sent to listen",
	}
	doc.ArrivalCandidates = []genesisCandidate{
		{Descriptor: "a newcomer", CanonicalName: "Elián", Why: "sent to listen"},
		{Descriptor: "Adaeze", CanonicalName: "Adaeze", Why: "already here"},
		{Descriptor: "a courier with wet boots", CanonicalName: "Ise", Why: "carrying an unlogged delivery"},
	}

	before := len(doc.History[0].Who)
	beforeKnowledge := len(doc.History[0].Knowledge)
	reconcileArrival(doc)

	// The arrival yielded to the record.
	if doc.Arrival.CanonicalName != "Ise" {
		t.Fatalf("the arrival did not yield to canon: %q", doc.Arrival.CanonicalName)
	}
	// And canon is exactly as authored — nothing detached, nothing unrecorded.
	if len(doc.History) != 1 {
		t.Fatalf("history was edited: %d events", len(doc.History))
	}
	if len(doc.History[0].Who) != before || len(doc.History[0].Knowledge) != beforeKnowledge {
		t.Fatalf("witnesses or knowledge were stripped: who=%v knowledge=%d",
			doc.History[0].Who, len(doc.History[0].Knowledge))
	}
	found := false
	for _, w := range doc.History[0].Who {
		if w == "Elián" {
			found = true
		}
	}
	if !found {
		t.Fatal("Elián was removed from the record of an event they were part of")
	}
}

// Canon exists once. History used to merge by blind append, which was already wrong — the ascent
// revisits layers and repair re-answers — and became a hazard when the per-person calls started running
// together: independent writers describing the same night produced one event each. A perception may
// disagree freely because it belongs to one holder; the event underneath it does not multiply.
func TestMergeFill_CanonExistsOnceAndGainsItsWitnesses(t *testing.T) {
	doc := &genesisDoc{}
	var tags []taggedName

	first := &fillFragment{History: []genesisEvent{{
		WhatHappened: "The heart stumbled and nobody wrote it down.",
		Where:        "El Lomo",
		Who:          []string{"Adaeze"},
		Knowledge:    []genesisKnowledge{{Holder: "Adaeze", Content: "the rhythm changed", EpistemicType: "direct"}},
	}}}
	mergeFill(doc, first, "people", &tags)

	// The same night, told again by another person's call — with a witness and a belief the first lacked.
	second := &fillFragment{History: []genesisEvent{{
		WhatHappened: "the heart stumbled and nobody wrote it down.",
		Where:        "El Lomo",
		Who:          []string{"Ferro"},
		Knowledge: []genesisKnowledge{
			{Holder: "Ferro", Content: "she heard it and said nothing", EpistemicType: "inference"},
			{Holder: "Adaeze", Content: "the rhythm changed", EpistemicType: "direct"},
		},
	}}}
	mergeFill(doc, second, "person:Ferro", &tags)

	if len(doc.History) != 1 {
		t.Fatalf("canon multiplied: %d events for one night", len(doc.History))
	}
	h := doc.History[0]
	if len(h.Who) != 2 {
		t.Fatalf("the second witness was lost: %v", h.Who)
	}
	if len(h.Knowledge) != 2 {
		t.Fatalf("want both holders' beliefs once each, got %d", len(h.Knowledge))
	}
}

// The belt's new contract (ADR-P027 §5): it validates against the LEVEL, so a thin entity is complete
// rather than defective, and an entity claiming a level it did not earn is refused. Without this, "the
// belt is level-aware" is a comment rather than a rule.
func TestBelt_ValidatesAgainstTheLevelNotOneFullness(t *testing.T) {
	// A world whose entities are all complete at relevance 1, plus the one person a scene can turn to.
	base := func() *genesisDoc {
		d := &genesisDoc{}
		d.World.DisplayName, d.World.Tagline = "The Short Line", "somebody is owed"
		d.World.Mood, d.World.Ornament = "nocturne", "filigree"
		d.Region.Descriptor, d.Region.ExtentClass = "a wet quarter", "small"
		d.Places = []genesisPlace{
			{CanonicalName: "The Counting Room", Descriptor: "one lamp", Kind: "back room", ExtentClass: "intimate", Tension: "tense", Relevance: 1, Tag: "the lamp never moves"},
			{CanonicalName: "The Loading Yard", Descriptor: "crates two high", Kind: "yard", ExtentClass: "small", Tension: "normal", Relevance: 1, Tag: "nothing leaves unlined"},
		}
		d.Ways = []genesisWay{{Descriptor: "the door", FromPlace: "The Counting Room", ToPlace: "The Loading Yard", State: "open"}}
		d.Cast = []genesisActor{
			{CanonicalName: "Runner", Descriptor: "hands that never stop", StartsIn: "The Counting Room", Relevance: 1, Tag: "never finishes a sentence"},
		}
		ledger := genesisObject{CanonicalName: "The Ledger", Descriptor: "a book", Kind: "ledger", Relevance: 1, Tag: "heavier than it looks"}
		ledger.Where.InPlace = "The Counting Room"
		d.Objects = []genesisObject{ledger}
		d.History = []genesisEvent{{WhatHappened: "a line was disputed", Where: "The Counting Room", Who: []string{"Runner"},
			Knowledge: []genesisKnowledge{{Holder: "Runner", EpistemicType: "direct", Content: "they were there"}}}}
		d.Arrival = genesisArrival{CanonicalName: "The Auditor", Descriptor: "no crate, no reason", Place: "The Counting Room", Stated: "You are owed a line."}
		return d
	}

	// A world of nothing but relevance 1 is LEGAL. This is the whole point: thin is complete.
	if err := base().validate(); err != nil {
		t.Fatalf("a world complete at relevance 1 was refused: %v", err)
	}

	for _, c := range []struct {
		name string
		bend func(*genesisDoc)
		want string
	}{
		{"a person claiming 2 without standing", func(d *genesisDoc) {
			d.Cast[0].Relevance = 2
			d.Cast[0].SpeechManner, d.Cast[0].Hiding = "flat", "which line they fixed"
			d.Cast[0].Traits = []genesisTrait{{Key: "exact", Strength: "strong", Manner: "repeats a number"}}
		}, "no standing"},
		{"a person claiming 3 with no inner life", func(d *genesisDoc) {
			d.Cast[0].Relevance = 3
			d.Cast[0].Standing, d.Cast[0].SpeechManner, d.Cast[0].Hiding = "answers for the book", "flat", "which line they fixed"
			d.Cast[0].Traits = []genesisTrait{{Key: "exact", Strength: "strong", Manner: "repeats a number"}}
		}, "wants nothing"},
		{"a location claiming 2 with no description", func(d *genesisDoc) {
			d.Places[0].Relevance = 2
		}, "no description"},
		{"a person with no tag", func(d *genesisDoc) { d.Cast[0].Tag = "" }, "no tag"},
		{"an entity outside the ladder", func(d *genesisDoc) { d.Cast[0].Relevance = 7 }, "outside 1-4"},
		{"an entity with no level at all", func(d *genesisDoc) { d.Cast[0].Relevance = 0 }, "outside 1-4"},
	} {
		d := base()
		c.bend(d)
		err := d.validate()
		if err == nil {
			t.Errorf("%s: the belt accepted it", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: refused with %q, want it to name %q", c.name, err.Error(), c.want)
		}
	}

	// Factions and concepts were never checked at all before this round; a duplicate name reached the
	// database. One namespace, everywhere.
	d := base()
	d.Factions = []genesisFaction{
		{CanonicalName: "The Tally", Descriptor: "keeps the book", Kind: "faction", Relevance: 1, Tag: "what is written arrived"},
		{CanonicalName: "The Tally", Descriptor: "also keeps the book", Kind: "group", Relevance: 1, Tag: "again"},
	}
	if err := d.validate(); err == nil || !strings.Contains(err.Error(), "two factions") {
		t.Errorf("a duplicate faction name was accepted: %v", err)
	}
	d = base()
	d.Concepts = []genesisConcept{{CanonicalName: "The Counting Room", WhatItIs: "a doctrine", Relevance: 1}}
	if err := d.validate(); err == nil || !strings.Contains(err.Error(), "also a place") {
		t.Errorf("a concept collided with a place name and was accepted: %v", err)
	}
}

// A world where the scaffold left EVERYTHING at relevance 1 is coherent and unplayable: nowhere has a
// description and nobody has a standing. Measured live 2026-08-31 on the Andantes brief — five
// locations, ten people, every one at relevance 1. The prompt asks for a floor; this guarantees it.
func TestEnsurePlayableFloor_PromotesTheFewestThingsThatMakeAScene(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{
		{CanonicalName: "Empty Ledge", Relevance: 1},
		{CanonicalName: "The Crowded Hall", Relevance: 1},
	}
	doc.Cast = []genesisActor{
		{CanonicalName: "One", StartsIn: "The Crowded Hall", Relevance: 1},
		{CanonicalName: "Two", StartsIn: "The Crowded Hall", Relevance: 1},
	}
	ensurePlayableFloor(doc)

	if doc.Places[1].Relevance < 2 {
		t.Error("the location with people in it was not promoted, so nowhere in this world is described")
	}
	if doc.Places[0].Relevance != 1 {
		t.Error("an empty ledge was promoted too — the floor must promote the FEWEST things, not raise the world")
	}
	speakable := 0
	for _, a := range doc.Cast {
		if a.Relevance >= 3 {
			speakable++
		}
	}
	if speakable != 1 {
		t.Errorf("%d people reached relevance 3, want exactly 1 — a floor is not a general promotion", speakable)
	}

	// Already playable: the floor must do NOTHING. It is a floor, not a policy.
	rich := &genesisDoc{}
	rich.Places = []genesisPlace{{CanonicalName: "A", Relevance: 2}, {CanonicalName: "B", Relevance: 1}}
	rich.Cast = []genesisActor{{CanonicalName: "Keeper", StartsIn: "A", Relevance: 3}, {CanonicalName: "Thin", StartsIn: "A", Relevance: 1}}
	ensurePlayableFloor(rich)
	if rich.Places[1].Relevance != 1 || rich.Cast[1].Relevance != 1 {
		t.Error("the floor promoted things in a world that already had a scene")
	}
}

// thinScaffoldDriver answers the scaffold exactly as the live model did on 2026-08-31: everything at
// relevance 1. Content calls it delegates, so whatever the floor promotes really does get authored.
type thinScaffoldDriver struct{ real Driver }

func (d *thinScaffoldDriver) Name() string { return "thin-scaffold" }
func (d *thinScaffoldDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *thinScaffoldDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	raw, err := d.real.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	if id := fillBatchID(req.Prompt); id != "scaffold-1" && id != "scaffold-2" {
		return raw, nil
	}
	var frag map[string]any
	if err := json.Unmarshal([]byte(raw), &frag); err != nil {
		return "", err
	}
	for _, key := range []string{"places", "cast", "factions"} {
		rows, _ := frag[key].([]any)
		for _, r := range rows {
			if m, ok := r.(map[string]any); ok {
				m["relevance"] = 1
			}
		}
	}
	out, err := json.Marshal(frag)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// The floor must be WIRED IN, not merely present. Deleting the one call site left the direct unit test
// green, which is the deletable-route failure AGENTS.md names — so this drives the real pipeline.
func TestFillFromIdentity_AnAllThinScaffoldStillYieldsAPlayableWorld(t *testing.T) {
	ctx := context.Background()
	seat := &thinScaffoldDriver{real: NewFakeWorldFillDriver()}
	id, err := inferIdentity(ctx, NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(ctx, seat, NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatalf("a world whose scaffold named everything at relevance 1 was refused: %v", err)
	}

	described := 0
	for _, p := range doc.Places {
		if p.Relevance >= 2 && strings.TrimSpace(p.Description) != "" {
			described++
		}
	}
	if described == 0 {
		t.Error("nowhere in this world is described, so the narrator has nothing to work from")
	}
	speakable := 0
	for _, a := range doc.Cast {
		if a.Relevance >= 3 && strings.TrimSpace(a.Goal) != "" {
			speakable++
		}
	}
	if speakable == 0 {
		t.Error("nobody in this world can be dealt with, so it has no scene in it")
	}
	// Connectivity and canon are STRUCTURAL and must not depend on anyone's level. This is the exact
	// refusal the live build hit: "nothing joins the places, so no one can leave the room they start in".
	if len(doc.Ways) == 0 {
		t.Error("nothing joins the places — connectivity was left to a level-gated wave")
	}
	if len(doc.History) == 0 {
		t.Error("nothing happened before the player arrived — canon was left to a level-gated wave")
	}
	// NO ORPHANS. The belt only demands one way and an exit from the arrival, so a world can pass it
	// while most of it is unreachable — quieter than a refusal and worse. Every location a wave named
	// must be joined to something, which is only true if connectivity runs for EVERY tree rather than
	// only the trees that happened to owe a description.
	touched := map[string]bool{}
	for _, w := range doc.Ways {
		touched[strings.TrimSpace(w.FromPlace)] = true
		touched[strings.TrimSpace(w.ToPlace)] = true
	}
	for _, p := range doc.Places {
		if n := strings.TrimSpace(p.CanonicalName); !touched[n] {
			t.Errorf("nothing joins %q — it exists and nobody can reach it", n)
		}
	}
}

// lateNamerDriver reproduces the 2026-08-31 refusal exactly: the closing pass pays a canon debt by
// authoring a person at relevance 2, AFTER the wave that would have given them a standing.
type lateNamerDriver struct{ real Driver }

func (d *lateNamerDriver) Name() string { return "late-namer" }
func (d *lateNamerDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *lateNamerDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	raw, err := d.real.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	if fillBatchID(req.Prompt) != "geography" {
		return raw, nil
	}
	// Canon names somebody nobody authored — the live shape, where history referenced 13 such people.
	var frag map[string]any
	if err := json.Unmarshal([]byte(raw), &frag); err != nil {
		return "", err
	}
	cast, _ := frag["cast"].([]any)
	frag["cast"] = append(cast, map[string]any{
		"canonical_name": "Kar", "descriptor": "a name in somebody else's account",
		"starts_in": fillSubject(req.Prompt), "relevance": 2, "tag": "spoken of, never present",
	})
	out, err := json.Marshal(frag)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// A build must not be thrown away because a LATE pass named someone at a level no later pass can author.
// Measured live: refused at 806 s and $0.09 over exactly this, on a person who could simply be thin.
func TestFillFromIdentity_SomeoneNamedTooLateSettlesThinRatherThanRefusing(t *testing.T) {
	ctx := context.Background()
	id, err := inferIdentity(ctx, NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(ctx, &lateNamerDriver{real: NewFakeWorldFillDriver()},
		NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatalf("the world was refused over a late arrival instead of settling them thin: %v", err)
	}
	var kar *genesisActor
	for i := range doc.Cast {
		if doc.Cast[i].CanonicalName == "Kar" {
			kar = &doc.Cast[i]
		}
	}
	if kar == nil {
		t.Fatal("Kar was dropped — a name canon references must exist, or the reference dangles")
	}
	if kar.Relevance != 1 {
		t.Errorf("Kar sits at relevance %d with nothing authored; settling records the level actually reached", kar.Relevance)
	}
	// Settling must not touch anyone who WAS authored.
	for _, a := range doc.Cast {
		if a.Relevance >= 2 && strings.TrimSpace(a.Standing) == "" {
			t.Errorf("%q is relevance %d with no standing — the belt would refuse this world", a.CanonicalName, a.Relevance)
		}
	}
	authored := 0
	for _, a := range doc.Cast {
		if a.Relevance >= 3 {
			authored++
		}
	}
	if authored == 0 {
		t.Error("settling demoted the whole cast — it is a reconciliation, not a policy")
	}
}

// A reference that names a real thing and then keeps talking must be snapped, not refused. This is the
// em-dash failure in its third costume: `starts_in` cost a 234 s build on 2026-08-28, and a faction's
// `seat` cost a 750 s build on 2026-08-31 as "Alto Omóplato, en el edificio de contraventanas de hueso."
func TestReconcileReferences_ARealNameFollowedByProseResolves(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{
		{CanonicalName: "Alto"},
		{CanonicalName: "Alto Omóplato"},
	}
	doc.Factions = []genesisFaction{
		{CanonicalName: "Colegio", Seat: "Alto Omóplato, en el edificio de contraventanas de hueso."},
		{CanonicalName: "Gremio", Seat: "Alto Omóplato"},
		{CanonicalName: "Orden", Seat: "somewhere nobody authored"},
		{CanonicalName: "Sin Sede", Seat: ""},
	}
	doc.Concepts = []genesisConcept{
		{CanonicalName: "Auscultación", TaughtBy: "Colegio — la sede del oficio"},
		{CanonicalName: "Peso", TaughtBy: "a guild that does not exist"},
	}
	reconcileReferences(doc)

	// The guard that actually does this work is the BOUNDARY check, not the longest-match: "Alto" is
	// rejected because what follows it is "Omóplato, …" rather than a comma or a dash. Verified by
	// mutation — deleting the longest-match leaves this green, deleting the boundary check turns it red.
	if got := doc.Factions[0].Seat; got != "Alto Omóplato" {
		t.Errorf("seat snapped to %q, want %q", got, "Alto Omóplato")
	}
	if got := doc.Factions[1].Seat; got != "Alto Omóplato" {
		t.Errorf("an exact seat was altered to %q", got)
	}
	if got := doc.Factions[2].Seat; got != "" {
		t.Errorf("an unresolvable seat survived as %q — it would dangle and cost the whole world", got)
	}
	if got := doc.Factions[3].Seat; got != "" {
		t.Errorf("an empty seat became %q", got)
	}
	if got := doc.Concepts[0].TaughtBy; got != "Colegio" {
		t.Errorf("taught_by snapped to %q, want %q", got, "Colegio")
	}
	if got := doc.Concepts[1].TaughtBy; got != "" {
		t.Errorf("an unresolvable taught_by survived as %q", got)
	}

	// It must NOT snap a name that merely shares a prefix with a real one: "Altozano" is not "Alto".
	other := &genesisDoc{}
	other.Places = []genesisPlace{{CanonicalName: "Alto"}}
	other.Factions = []genesisFaction{{CanonicalName: "F", Seat: "Altozano"}}
	reconcileReferences(other)
	if got := other.Factions[0].Seat; got != "" {
		t.Errorf("%q was snapped to a different place — a prefix is not a match unless the name ends there", got)
	}
}

// proseSeatDriver returns a faction seated in a REAL location with a description appended — the exact
// answer that refused a 750-second live build on 2026-08-31.
type proseSeatDriver struct{ real Driver }

func (d *proseSeatDriver) Name() string { return "prose-seat" }
func (d *proseSeatDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *proseSeatDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	raw, err := d.real.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	if fillBatchID(req.Prompt) != "faction" {
		return raw, nil
	}
	var frag map[string]any
	if err := json.Unmarshal([]byte(raw), &frag); err != nil {
		return "", err
	}
	rows, _ := frag["factions"].([]any)
	for _, r := range rows {
		if m, ok := r.(map[string]any); ok {
			m["seat"] = "The Counting Room, en el edificio de contraventanas de hueso."
		}
	}
	out, err := json.Marshal(frag)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// The snap must be WIRED IN. Removing its call sites left the direct unit test green — the same
// deletable-route failure as the playable floor, in the same round. This drives the real pipeline.
func TestFillFromIdentity_AProseShapedSeatDoesNotCostTheWorld(t *testing.T) {
	ctx := context.Background()
	id, err := inferIdentity(ctx, NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(ctx, &proseSeatDriver{real: NewFakeWorldFillDriver()},
		NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatalf("a faction seated in a real location with prose after it cost the whole world: %v", err)
	}
	seated := 0
	for _, f := range doc.Factions {
		if s := strings.TrimSpace(f.Seat); s != "" {
			seated++
			if s != "The Counting Room" {
				t.Errorf("the faction %q kept the prose-shaped seat %q", f.CanonicalName, s)
			}
		}
	}
	if seated == 0 {
		t.Error("every seat was cleared — the location named WAS real and must survive as a reference")
	}
}

// Every reference degrades in the cheapest honest way rather than costing the world. The choice differs
// per kind, and getting one of them wrong is how a repaired reference becomes a dangling one.
func TestReconcileReferences_EachKindDegradesInItsOwnWay(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{{CanonicalName: "Ossa"}, {CanonicalName: "Cola Baja"}}
	doc.Factions = []genesisFaction{{CanonicalName: "El Gremio"}}
	doc.Cast = []genesisActor{
		{CanonicalName: "Sento", StartsIn: "Cola Baja", BelongsTo: []string{"El Gremio", "a guild nobody wrote"}},
		{CanonicalName: "Nadie", StartsIn: "Segundo"}, // a location no pass ever authored
	}
	doc.Ways = []genesisWay{
		{Descriptor: "the climb", FromPlace: "Ossa", ToPlace: "Cola Baja", State: "open"},
		{Descriptor: "the crossing", FromPlace: "Ossa", ToPlace: "Segundo", State: "open"},
	}
	doc.Objects = []genesisObject{
		{CanonicalName: "El Libro", Descriptor: "a book", Kind: "ledger"},
		{CanonicalName: "El Perdido", Descriptor: "lost", Kind: "thing"},
	}
	doc.Objects[0].Where.InPlace = "Ossa, en la sala del fondo"
	doc.Objects[1].Where.InPlace = "Segundo"
	doc.History = []genesisEvent{
		{WhatHappened: "the tally was disputed", Where: "Ossa", Who: []string{"Sento", "Nadie"},
			Knowledge: []genesisKnowledge{
				{Holder: "Sento", EpistemicType: "direct", Content: "he was there"},
				{Holder: "Nadie", EpistemicType: "told", Content: "she heard"},
			}},
		{WhatHappened: "something nobody surviving knows", Where: "Ossa", Who: []string{"Nadie"},
			Knowledge: []genesisKnowledge{{Holder: "Nadie", EpistemicType: "direct", Content: "only she knew"}}},
		{WhatHappened: "something nowhere", Where: "Segundo",
			Knowledge: []genesisKnowledge{{Holder: "Sento", EpistemicType: "direct", Content: "x"}}},
	}
	doc.Concepts = []genesisConcept{
		{CanonicalName: "La Lectura", TaughtBy: "Sento"},              // a PERSON teaches a craft
		{CanonicalName: "El Cálculo", TaughtBy: "El Gremio, central"}, // faction plus prose
		{CanonicalName: "El Hueco", TaughtBy: "nobody at all"},
	}
	doc.Arrival = genesisArrival{CanonicalName: "Wren", Place: "Ossa, bajando por la rampa"}

	reconcileReferences(doc)

	// A way to nowhere is one edge, not the world.
	if len(doc.Ways) != 1 || doc.Ways[0].Descriptor != "the climb" {
		t.Errorf("ways = %+v, want only the resolvable one", doc.Ways)
	}
	// An unplaceable person cannot be stored, and canon must follow who survived.
	if len(doc.Cast) != 1 || doc.Cast[0].CanonicalName != "Sento" {
		t.Fatalf("cast = %+v, want only Sento", doc.Cast)
	}
	if got := doc.Cast[0].BelongsTo; len(got) != 1 || got[0] != "El Gremio" {
		t.Errorf("belongs_to = %v, want the one faction that exists", got)
	}
	// One event survives, with the dropped person cleaned out of it. One dies for having no holder left,
	// one for happening nowhere.
	if len(doc.History) != 1 {
		t.Fatalf("history = %d events, want 1", len(doc.History))
	}
	if got := doc.History[0].Who; len(got) != 1 || got[0] != "Sento" {
		t.Errorf("who = %v — a dropped person must not survive as a participant", got)
	}
	if got := doc.History[0].Knowledge; len(got) != 1 || got[0].Holder != "Sento" {
		t.Errorf("knowledge = %+v — a dropped person must not survive as a holder", got)
	}
	// An object snapped to a real location; one that is nowhere is dropped.
	if len(doc.Objects) != 1 || doc.Objects[0].Where.InPlace != "Ossa" {
		t.Errorf("objects = %+v, want El Libro in Ossa", doc.Objects)
	}
	// taught_by takes a person OR a faction. Clearing the person threw away true content.
	if doc.Concepts[0].TaughtBy != "Sento" {
		t.Errorf("a person teaching a craft was cleared: %q", doc.Concepts[0].TaughtBy)
	}
	if doc.Concepts[1].TaughtBy != "El Gremio" {
		t.Errorf("faction-plus-prose snapped to %q", doc.Concepts[1].TaughtBy)
	}
	if doc.Concepts[2].TaughtBy != "" {
		t.Errorf("an unresolvable teacher survived as %q", doc.Concepts[2].TaughtBy)
	}
	if doc.Arrival.Place != "Ossa" {
		t.Errorf("the arrival place snapped to %q", doc.Arrival.Place)
	}
}

// wayToNowhereDriver reproduces the fourth live refusal: geography joins a location to one of the nine
// Andantes that no pass ever authored.
type wayToNowhereDriver struct{ real Driver }

func (d *wayToNowhereDriver) Name() string { return "way-to-nowhere" }
func (d *wayToNowhereDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *wayToNowhereDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	raw, err := d.real.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	if fillBatchID(req.Prompt) != "geography" {
		return raw, nil
	}
	var frag map[string]any
	if err := json.Unmarshal([]byte(raw), &frag); err != nil {
		return "", err
	}
	ways, _ := frag["ways"].([]any)
	frag["ways"] = append(ways, map[string]any{
		"descriptor": "the crossing to Segundo", "from_place": fillSubject(req.Prompt),
		"to_place": "Segundo", "state": "open",
	})
	out, err := json.Marshal(frag)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// The reconciliation must be WIRED IN. Live run four: refused at 1,201 s and $0.063 on
// `a way leads to "Segundo", which is not a place in this world`.
func TestFillFromIdentity_AWayToNowhereCostsTheEdgeNotTheWorld(t *testing.T) {
	ctx := context.Background()
	id, err := inferIdentity(ctx, NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(ctx, &wayToNowhereDriver{real: NewFakeWorldFillDriver()},
		NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatalf("one bad edge cost the whole world: %v", err)
	}
	for _, w := range doc.Ways {
		if strings.TrimSpace(w.ToPlace) == "Segundo" {
			t.Error("the dangling edge survived into the document")
		}
	}
	if len(doc.Ways) == 0 {
		t.Error("every way was dropped — the good edges must survive with the bad one")
	}
}

// ONE NAME BELONGS TO ONE THING, and precedence is by how much depends on the name. Measured live
// 2026-08-31: refused at 1,460 s and $0.037 because "Colegio de Auscultadores de Ossa" arrived as both a
// location and a person.
func TestResolveNameCollisions_TheKindEverythingDependsOnKeepsTheName(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{
		{CanonicalName: "Colegio", Descriptor: "a hall", Kind: "hall", Relevance: 2, Description: "kept"},
		{CanonicalName: "Colegio", Descriptor: "SHOULD LOSE", Kind: "hall", Relevance: 1, Tension: "calm"},
		{CanonicalName: "Ossa", Descriptor: "a back", Kind: "back", Relevance: 1},
	}
	doc.Cast = []genesisActor{
		{CanonicalName: "Colegio", Descriptor: "the institution, misfiled as a person", StartsIn: "Ossa", Relevance: 1, Tag: "t"},
		{CanonicalName: "Del Vas", Descriptor: "a person", StartsIn: "Ossa", Relevance: 1, Tag: "t"},
		{CanonicalName: "Del Vas", Descriptor: "the same person again", StartsIn: "Ossa", Relevance: 2, Tag: "t", Standing: "kept"},
	}
	doc.Factions = []genesisFaction{{CanonicalName: "Ossa", Descriptor: "clashes with a location", Kind: "faction", Relevance: 1, Tag: "t"}}
	doc.Concepts = []genesisConcept{{CanonicalName: "Del Vas", WhatItIs: "clashes with a person", Relevance: 1}}
	doc.Objects = []genesisObject{{CanonicalName: "Colegio", Descriptor: "clashes with a location", Kind: "thing", Relevance: 1}}

	resolveNameCollisions(doc)

	if len(doc.Places) != 2 {
		t.Fatalf("locations = %d, want 2 — the duplicate must MERGE, not be dropped", len(doc.Places))
	}
	if doc.Places[0].Descriptor != "a hall" {
		t.Errorf("the merge overwrote the first answer with %q", doc.Places[0].Descriptor)
	}
	if doc.Places[0].Tension != "calm" {
		t.Error("the merge discarded what the second answer added — two half-answers must become one whole one")
	}
	if len(doc.Cast) != 1 || doc.Cast[0].CanonicalName != "Del Vas" {
		t.Fatalf("cast = %+v, want only Del Vas: a location outranks a person for the same name", doc.Cast)
	}
	if doc.Cast[0].Standing != "kept" {
		t.Error("the two Del Vas rows did not merge")
	}
	if len(doc.Factions) != 0 {
		t.Error("a faction kept a name a location already owns")
	}
	if len(doc.Concepts) != 0 {
		t.Error("a concept kept a name a person already owns")
	}
	if len(doc.Objects) != 0 {
		t.Error("an object kept a name a location already owns")
	}
}

// A synonym for a closed-set value is a fine English answer to a question with a fixed list of legal
// ones. Snapping costs a shade of meaning; refusing costs the world.
func TestNormaliseClosedSets_ASynonymIsNotABrokenWorld(t *testing.T) {
	doc := &genesisDoc{}
	doc.Region.ExtentClass = "enormous"
	doc.Places = []genesisPlace{{CanonicalName: "P", Tension: "uneasy", ExtentClass: "tiny"}}
	doc.Ways = []genesisWay{{Descriptor: "W", State: "ajar"}}
	doc.Cast = []genesisActor{{CanonicalName: "A", Malleability: "unyielding",
		Traits: []genesisTrait{{Key: "k", Strength: "overwhelming", Manner: "m"}}}}
	doc.History = []genesisEvent{{WhatHappened: "x", Where: "P",
		Knowledge: []genesisKnowledge{{Holder: "A", EpistemicType: "guessed", Content: "c"}}}}

	normaliseClosedSets(doc)

	for _, c := range []struct{ got, want, what string }{
		{doc.Region.ExtentClass, "medium", "region extent_class"},
		{doc.Places[0].Tension, "normal", "tension"},
		{doc.Places[0].ExtentClass, "small", "place extent_class"},
		{doc.Ways[0].State, "open", "way state"},
		{doc.Cast[0].Malleability, "moderate", "malleability"},
		{doc.Cast[0].Traits[0].Strength, "moderate", "trait strength"},
		{doc.History[0].Knowledge[0].EpistemicType, "told", "epistemic_type"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.what, c.got, c.want)
		}
	}
	// `direct` is a claim about PRESENCE. Defaulting to it would put someone at an event they missed.
	if doc.History[0].Knowledge[0].EpistemicType == "direct" {
		t.Error("an unknown epistemic type defaulted to `direct`, inventing a witness")
	}
}

// collidingNameDriver reproduces the fifth live refusal: an institution filed as both a location and a
// person, in the same build.
type collidingNameDriver struct{ real Driver }

func (d *collidingNameDriver) Name() string { return "colliding-name" }
func (d *collidingNameDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *collidingNameDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	raw, err := d.real.Generate(ctx, req)
	if err != nil {
		return "", err
	}
	if fillBatchID(req.Prompt) != "scaffold-2" || !strings.Contains(req.Prompt, "Name what sits inside") {
		return raw, nil
	}
	var frag map[string]any
	if err := json.Unmarshal([]byte(raw), &frag); err != nil {
		return "", err
	}
	subject := fillSubject(req.Prompt)
	cast, _ := frag["cast"].([]any)
	frag["cast"] = append(cast, map[string]any{
		"canonical_name": subject, "descriptor": "the institution, misfiled as a person",
		"starts_in": subject, "relevance": 1, "tag": "answers as an office",
	})
	out, err := json.Marshal(frag)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// The collision pass must be WIRED IN, and the world must survive the collision.
func TestFillFromIdentity_AnInstitutionFiledTwiceDoesNotCostTheWorld(t *testing.T) {
	ctx := context.Background()
	id, err := inferIdentity(ctx, NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(ctx, &collidingNameDriver{real: NewFakeWorldFillDriver()},
		NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatalf("a name filed as both a location and a person cost the whole world: %v", err)
	}
	places := map[string]bool{}
	for _, p := range doc.Places {
		places[strings.TrimSpace(p.CanonicalName)] = true
	}
	for _, a := range doc.Cast {
		if places[strings.TrimSpace(a.CanonicalName)] {
			t.Errorf("%q survived as both a person and a location", a.CanonicalName)
		}
	}
	if len(doc.Places) == 0 || len(doc.Cast) == 0 {
		t.Error("the collision emptied the world instead of costing one row")
	}
}

// EVERY scheduled call must be able to see something. Setting no Scope at all was three separate bugs in
// one round — closing, repair and arrival were each blind to the document they exist to work on — and the
// geography wave could see locations but no PEOPLE, which made its own instruction ("every event needs a
// holder named below") unsatisfiable and produced a world where nothing had ever happened. Live
// 2026-08-31: refused at 950 s with no history at all, and nothing dropped to explain it.
func TestEveryScheduledCallCanSeeSomething(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{
		{CanonicalName: "Top", Relevance: 2, Descriptor: "d", Kind: "k", ExtentClass: "small"},
		{CanonicalName: "Inner", Within: "Top", Relevance: 2, Descriptor: "d", Kind: "k", ExtentClass: "intimate"},
	}
	doc.Factions = []genesisFaction{{CanonicalName: "Gremio", Relevance: 2, Descriptor: "d", Kind: "faction", Tag: "t"}}
	doc.Concepts = []genesisConcept{{CanonicalName: "Craft", WhatItIs: "w", Relevance: 1}}
	doc.Cast = []genesisActor{
		{CanonicalName: "Keeper", StartsIn: "Inner", Relevance: 3, Tag: "t", Descriptor: "d"},
		{CanonicalName: "Thin", StartsIn: "Top", Relevance: 1, Tag: "t", Descriptor: "d"},
	}
	b := budgetForDepth(0)

	items := []workItem{conceptsWork(), scaffoldOneWork(b), arrivalWork()}
	items = append(items, scaffoldTwoSchedule(doc, b)...)
	for _, wave := range contentSchedule(doc, b) {
		items = append(items, wave...)
	}
	for _, it := range items {
		s := it.Scope
		if !s.Whole && len(s.Places) == 0 && len(s.Factions) == 0 && len(s.Concepts) == 0 && len(s.People) == 0 {
			t.Errorf("%q can see nothing at all — a blind call cannot reference what exists", mergeTag(it))
		}
	}

	// Geography owns canon, so it must be shown somebody who can hold an event.
	var geo *workItem
	for i := range items {
		if items[i].ID == "geography" {
			geo = &items[i]
			break
		}
	}
	if geo == nil {
		t.Fatal("no geography item — connectivity and canon have no owner")
	}
	if len(geo.Scope.People) == 0 {
		t.Error("geography sees no people, so every event it writes would have no holder and the wave returns no history")
	}
	found := false
	for _, n := range geo.Scope.People {
		if n == "Keeper" {
			found = true
		}
	}
	if !found {
		t.Errorf("geography for %q does not see Keeper, who stands inside its tree: %v", geo.Subject, geo.Scope.People)
	}
	if got := peopleIn(doc, []string{"Top"}); len(got) != 1 || got[0] != "Thin" {
		t.Errorf("peopleIn(Top) = %v, want only Thin — it must not sweep in the whole cast", got)
	}
}

// The blind-scope fallback itself: a work item that names nothing sees the whole document rather than an
// empty one, because blind is never what anybody meant.
func TestFillPrompt_AnEmptyMandateFallsBackToTheWholeDocument(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{{CanonicalName: "The Counting Room", Descriptor: "one lamp"}}
	doc.Cast = []genesisActor{{CanonicalName: "Del Vas", Descriptor: "a person", Tag: "t"}}

	blind := buildWorldFillPrompt(&worldIdentity{}, workItem{ID: "repair", Kind: "batch"}, testBrief, nil, doc, "")
	if !strings.Contains(blind, "The Counting Room") || !strings.Contains(blind, "Del Vas") {
		t.Fatal("a scopeless work item was shown an empty world — it cannot avoid re-authoring names it cannot see")
	}
	// A scope that names something real is still honoured exactly.
	scoped := buildWorldFillPrompt(&worldIdentity{}, workItem{
		ID: "people", Kind: "content", Scope: fillScope{People: []string{"Del Vas"}},
	}, testBrief, nil, doc, "")
	if strings.Contains(scoped, "The Counting Room") {
		t.Error("a scoped call was shown a location outside its mandate — the compiled mandate is the saving")
	}
}

// Canon is its own layer, between the places and the lives, which is where the founder's 2026-08-28
// ordering put it. It was folded into the geography wave as a second job, and the measured result was ONE
// event in a world of 71 entities (live run seven). A layer with two jobs does the first one.
func TestCanonIsItsOwnLayerBetweenPlacesAndLives(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{
		{CanonicalName: "Top", Relevance: 2, Descriptor: "d", Kind: "k", ExtentClass: "small"},
		{CanonicalName: "Inner", Within: "Top", Relevance: 1, Descriptor: "d", Kind: "k", ExtentClass: "intimate"},
	}
	doc.Cast = []genesisActor{{CanonicalName: "Keeper", StartsIn: "Inner", Relevance: 2, Tag: "t", Descriptor: "d"}}
	b := budgetForDepth(0)

	var order []string
	for _, wave := range contentSchedule(doc, b) {
		order = append(order, wave[0].ID)
	}
	gi, ci, pi := indexOf(order, "geography"), indexOf(order, "canon"), indexOf(order, "people")
	if ci < 0 {
		t.Fatalf("no canon layer at all; waves are %v", order)
	}
	if gi < 0 || gi > ci {
		t.Errorf("canon runs before geography (%v) — holders need somewhere for it to have happened", order)
	}
	if pi >= 0 && ci > pi {
		t.Errorf("canon runs after the lives (%v) — the founder's order is places, history, lives", order)
	}

	// The canon call must see the people who can hold what it writes, or it authors nothing.
	for _, wave := range contentSchedule(doc, b) {
		for _, it := range wave {
			if it.ID != "canon" {
				continue
			}
			if len(it.Scope.People) == 0 {
				t.Error("the canon layer sees no people, so every event it writes would have no holder")
			}
			if len(it.Scope.Places) == 0 {
				t.Error("the canon layer sees no locations, so nothing it writes can have happened anywhere")
			}
		}
	}

	// A tree with nobody in it gets no canon call: an event nobody holds cannot be perceived, so asking
	// for one would only produce something the belt has to throw away.
	empty := &genesisDoc{}
	empty.Places = []genesisPlace{{CanonicalName: "Nowhere", Relevance: 1, Descriptor: "d", Kind: "k", ExtentClass: "small"}}
	if items := canonSchedule(empty, b); len(items) != 0 {
		t.Errorf("canon was scheduled for a tree with nobody in it: %d item(s)", len(items))
	}
}

func indexOf(in []string, want string) int {
	for i, s := range in {
		if s == want {
			return i
		}
	}
	return -1
}

// The layer must actually produce canon that TRAVELS: a holder who was not present is what makes lore
// rather than a private memory.
func TestFillFromIdentity_CanonTravelsBeyondTheRoom(t *testing.T) {
	ctx := context.Background()
	id, err := inferIdentity(ctx, NewFakeWorldUnderstandingDriver(), testBrief, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := fillFromIdentity(ctx, NewFakeWorldFillDriver(), NewFakeWorldFillReviewDriver(), id, testBrief, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.History) < 2 {
		t.Errorf("history = %d event(s); a dedicated layer should author more than the one a second job managed", len(doc.History))
	}
	travelled := false
	for _, h := range doc.History {
		present := map[string]bool{}
		for _, w := range h.Who {
			present[strings.TrimSpace(w)] = true
		}
		for _, k := range h.Knowledge {
			if !present[strings.TrimSpace(k.Holder)] {
				travelled = true
			}
		}
	}
	if !travelled {
		t.Error("every event is known only to the people who were there — nothing in this world travels, and that is a world without lore")
	}
}
