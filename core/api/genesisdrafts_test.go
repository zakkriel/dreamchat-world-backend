package main

import (
	"testing"
	"time"
)

func TestDraftStoreClaimRoundTrip(t *testing.T) {
	s := newDraftStore(time.Minute)
	h := s.mint()
	if len(h) != 36 {
		t.Fatalf("handle length = %d, want 36", len(h))
	}
	d := &genesisDraft{brief: "a harbour town"}
	s.put(h, d)
	got, ok := s.claim(h)
	if !ok || got.brief != "a harbour town" {
		t.Fatalf("claim = %v, %v", got, ok)
	}
	if _, ok := s.claim(h); ok {
		t.Fatal("second claim succeeded; claim must remove")
	}
}

func TestDraftStoreExpiry(t *testing.T) {
	s := newDraftStore(10 * time.Millisecond)
	h := s.mint()
	s.put(h, &genesisDraft{})
	time.Sleep(20 * time.Millisecond)
	if _, ok := s.claim(h); ok {
		t.Fatal("claimed an expired draft")
	}
}

func TestDraftStoreUnknownHandle(t *testing.T) {
	s := newDraftStore(time.Minute)
	if _, ok := s.claim("00000000-0000-0000-0000-000000000000"); ok {
		t.Fatal("claimed a handle that was never put")
	}
}

func TestDraftTallyAccumulates(t *testing.T) {
	var tl draftTally
	tl.add(0.01, 100, 50, 10, 1)
	tl.add(0.02, 200, 60, 0, 2)
	if tl.usd != 0.03 || tl.tokIn != 300 || tl.tokOut != 110 || tl.cached != 10 || tl.calls != 3 {
		t.Fatalf("tally = %+v", tl)
	}
}
