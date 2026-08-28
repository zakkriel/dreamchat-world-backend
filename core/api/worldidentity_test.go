package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSchedule_DescendsThenAscends(t *testing.T) {
	down := []string{"places", "factions", "concepts", "people", "artifacts"}
	got := descentSchedule()
	if len(got) != len(down) {
		t.Fatalf("descent has %d layers, want %d", len(got), len(down))
	}
	for i := range down {
		if got[i].ID != down[i] {
			t.Fatalf("descent layer %d is %q, want %q — nothing may be named before it exists", i, got[i].ID, down[i])
		}
	}

	// The ascent is per-person, and it can only be built after the descent says who exists.
	doc := &genesisDoc{}
	doc.Cast = []genesisActor{{CanonicalName: "Adaeze"}, {CanonicalName: "Ferro"}}
	up := ascentSchedule(doc)
	people := 0
	for _, it := range up {
		if it.ID == "person" {
			people++
			if it.Subject == "" {
				t.Fatal("a per-person ascent item carries no subject")
			}
		}
	}
	if people != 2 {
		t.Fatalf("ascent authored %d person items for 2 people", people)
	}
	// Coming up, the layers are revisited in reverse: artifacts before people before places.
	order := []string{}
	for _, it := range up {
		if it.ID != "person" {
			order = append(order, it.ID)
		}
	}
	want := []string{"artifacts-connect", "concepts-connect", "factions-connect", "places-connect"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("ascent %d is %q, want %q — the way up is the way down reversed", i, order[i], want[i])
		}
	}
	if mergeTag(workItem{ID: "person", Subject: "Adaeze"}) != "person:Adaeze" {
		t.Fatal("per-item work is not distinguishable for retraction")
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

func TestFakeFill_PeopleLayerIsAFillFragmentNotAGenesisDump(t *testing.T) {
	raw, err := NewFakeWorldFillDriver().Generate(context.Background(), GenRequest{
		Prompt: "\nid: people\nkind: descent\n" + worldGenesisBriefMarker + "\n" + testBrief,
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

	prompt := buildWorldFillPrompt(&worldIdentity{}, workItem{ID: "people", Kind: "descent"}, testBrief, nil, soFar, "")

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

	// And it is NOT fatal. Founder 2026-08-28: a good invention is never discarded for arriving before
	// its room. fillOne hands the fragment back; the debt is carried and the closing pass authors it.
	seat := stubDriver{raw: `{"empty":false,"cast":[{"descriptor":"a man","canonical_name":"Sento","standing":"s","speech_manner":"m","hiding":"h","malleability":"faint","starts_in":"Cola Baja"}]}`}
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
	doc, err := fillFromIdentity(context.Background(), seat, NewFakeWorldFillReviewDriver(), id, testBrief, nil)
	if err != nil {
		t.Fatalf("the build refused instead of authoring the owed place: %v", err)
	}
	if !seat.closed {
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
			"descriptor": "a man on the low tail", "canonical_name": "Sento",
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
		o.closed = true
		return `{"empty":false,"places":[{"descriptor":"the low tail, awash","canonical_name":"Cola Baja","kind":"district","description":"The tail drags and the water comes over it twice a day; nobody builds high here.","tension":"normal","extent_class":"small"}],"ways":[{"descriptor":"the climb up the spine","from_place":"Cola Baja","to_place":"The Counting Room","state":"open"}]}`, nil
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
	frag.Cast = []genesisActor{{CanonicalName: "un aprendiz de 27 años", Hiding: "he has not told anyone", StartsIn: "El Lomo"}}
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
