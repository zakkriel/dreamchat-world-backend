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
	_, err := fillOne(context.Background(), seat, &worldIdentity{}, workItem{ID: "r", Kind: "generative", Text: "t", Therefore: "t"}, testBrief, nil, &genesisDoc{})
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
