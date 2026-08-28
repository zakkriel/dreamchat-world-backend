package main

import (
	"context"
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
