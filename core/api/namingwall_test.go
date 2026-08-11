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
	if _, err := DecodeAndValidateNarration(leak, nil, nil, wall); err == nil {
		t.Fatal("the founder's leaked narration was accepted — the wall is not enforcing")
	} else if !strings.Contains(err.Error(), "Jonas") || !strings.Contains(err.Error(), "has not earned") {
		t.Fatalf("rejection must name the offending word so the repair prompt can use it, got: %v", err)
	}

	// Mara IS earned (Kade holds name-knowledge of her): the same sentence without Jonas must pass,
	// or the wall is just censoring every capital letter.
	clean := `[{"speaker_id":null,"kind":"narration","text":"Mara is behind the bar now, the muscle by the bar planted between her and the room."}]`
	if _, err := DecodeAndValidateNarration(clean, nil, nil, wall); err != nil {
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
