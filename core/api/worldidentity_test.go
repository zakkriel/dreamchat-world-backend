package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestScheduleWork_ConstraintsBeforeGenerativeThenSufficiency(t *testing.T) {
	id := &worldIdentity{}
	id.Rules = []identityRule{
		{ID: "g", Kind: "generative", Text: "g", Therefore: "t"},
		{ID: "c", Kind: "constraining", Text: "c", Therefore: "t"},
		{ID: "p", Kind: "prohibiting", Text: "p", Therefore: "t"},
	}
	got := scheduleWork(id)
	want := []string{"c", "p", "g", "sufficiency"}
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
	doc, ident, err := authorWorld(ctx, NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), testBrief, nil)
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

func TestFillFromIdentity_ConstraintsStayEmptyAndPeopleHide(t *testing.T) {
	doc, ident, err := authorWorld(context.Background(), NewFakeWorldUnderstandingDriver(), NewFakeWorldFillDriver(), testBrief, nil)
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

func TestFakeFill_GenerativeIsAFillFragmentNotAGenesisDump(t *testing.T) {
	raw, err := NewFakeWorldFillDriver().Generate(context.Background(), GenRequest{
		Prompt: "\nid: r-ask\nkind: generative\n" + worldGenesisBriefMarker + "\n" + testBrief,
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
		t.Fatalf("generative should author lives, empty=%v cast=%d", frag.Empty, len(frag.Cast))
	}
}

type stubDriver struct{ raw string }

func (s stubDriver) Name() string                 { return "stub-fill" }
func (s stubDriver) Capabilities() CapabilitySet  { return CapabilitySet{CapStructuredOutput: true} }
func (s stubDriver) Generate(context.Context, GenRequest) (string, error) {
	return s.raw, nil
}
