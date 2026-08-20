package main

import (
	"context"
	"strings"
	"testing"
)

// The catalogue is a contract: a picker renders these keys, a world row stores one, and every
// picture that world ever draws is keyed to the profile they name. Renaming one silently re-keys
// every world that chose it and orphans its art.
func TestArtStyleCatalogue_IsTheFivePromisedLooksInOrder(t *testing.T) {
	got := ArtStyleCatalogue()

	want := []string{"anime", "realistic", "manhwa", "comic", "3d"}
	if len(got) != len(want) {
		t.Fatalf("catalogue has %d styles, want %d", len(got), len(want))
	}
	for i, key := range want {
		if got[i].Key != key {
			t.Errorf("style %d is %q, want %q — order is display order and the picker follows it", i, got[i].Key, key)
		}
		if strings.TrimSpace(got[i].Label) == "" || strings.TrimSpace(got[i].Blurb) == "" {
			t.Errorf("%s has no label or blurb; a picker has nothing to render", key)
		}
	}
}

// Every style carries the latitude, or picking "realistic" quietly opts a world out of the content
// the seats were told to write.
func TestEveryStyleCarriesTheLatitudeAndRefusesCensorship(t *testing.T) {
	styles := append(ArtStyleCatalogue(), mustResolve(t, ""), mustResolve(t, "custom:oil painting, heavy impasto"))

	for _, s := range styles {
		if !strings.Contains(s.Positive(), artStyleLatitude) {
			t.Errorf("%s does not carry the latitude in its positive prompt", s.Key)
		}
		for _, refused := range []string{"censorship bar", "coy crop", "tasteful cutaway"} {
			if !strings.Contains(s.Negative(), refused) {
				t.Errorf("%s does not refuse %q", s.Key, refused)
			}
		}
	}
}

// A style's look must actually reach the prompt — a catalogue where every entry produces the same
// image is five buttons that do nothing.
func TestEachPresetSendsItsOwnLook(t *testing.T) {
	seen := map[string]string{}
	for _, s := range ArtStyleCatalogue() {
		p := s.Positive()
		if prev, dup := seen[p]; dup {
			t.Fatalf("%s and %s send an identical prompt — one of them is not a style", prev, s.Key)
		}
		seen[p] = s.Key
	}

	anime := mustResolve(t, "anime")
	if !strings.Contains(anime.Positive(), "cel shaded") {
		t.Errorf("anime does not describe itself to the model: %q", anime.Positive())
	}
}

// A world that predates styles keeps the profile its art is already stored under. Re-keying it would
// orphan every picture those worlds have.
func TestUnchosenStyleKeepsTheExistingHouseProfile(t *testing.T) {
	s := mustResolve(t, "")
	if got := s.ProfileName(); got != "dreamchat-default" {
		t.Fatalf("profile = %q, want dreamchat-default — legacy art is stored under that name", got)
	}
	if !strings.Contains(s.Positive(), "painterly") {
		t.Errorf("the house look changed; existing worlds would start drawing differently: %q", s.Positive())
	}
}

// Presets are shared profiles; a written style is named by its CONTENT. Two worlds described the
// same way must share one profile, or the platform's reuse cache never hits and every world pays to
// redraw the same look.
func TestCustomStylesAreNamedByTheirContent(t *testing.T) {
	a := mustResolve(t, "custom:oil painting, heavy impasto")
	b := mustResolve(t, "custom:  OIL PAINTING,   heavy impasto  ")
	c := mustResolve(t, "custom:charcoal sketch")

	if a.ProfileName() != b.ProfileName() {
		t.Errorf("the same description in different whitespace and case made two profiles: %s vs %s",
			a.ProfileName(), b.ProfileName())
	}
	if a.ProfileName() == c.ProfileName() {
		t.Error("two different descriptions collided onto one profile")
	}
	if !strings.HasPrefix(a.ProfileName(), "dreamchat-custom-") {
		t.Errorf("custom profile is named %q", a.ProfileName())
	}
	if !strings.Contains(a.Positive(), "oil painting, heavy impasto") {
		t.Errorf("the user's own words did not reach the prompt: %q", a.Positive())
	}
}

// Presets must never collide with each other or with the house profile.
func TestPresetProfileNamesAreDistinct(t *testing.T) {
	seen := map[string]bool{"dreamchat-default": true}
	for _, s := range ArtStyleCatalogue() {
		n := s.ProfileName()
		if seen[n] {
			t.Fatalf("profile name %q is used twice", n)
		}
		seen[n] = true
	}
}

// A bad choice must be refused where it costs nothing, and the message must say what IS allowed —
// this error is rendered to the person who just typed it.
func TestUnknownAndEmptyStylesAreRefusedUsefully(t *testing.T) {
	if _, err := ResolveArtStyle("watercolour"); err == nil {
		t.Fatal("an unknown key must be refused")
	} else {
		for _, key := range []string{"anime", "realistic", "manhwa", "comic", "3d"} {
			if !strings.Contains(err.Error(), key) {
				t.Errorf("the refusal does not name %q as an option: %s", key, err)
			}
		}
	}

	if _, err := ResolveArtStyle("custom:   "); err == nil {
		t.Error("a written style with no words must be refused, not sent as an empty prompt")
	}

	long := "custom:" + strings.Repeat("a", artStyleMaxCustom+1)
	if _, err := ResolveArtStyle(long); err == nil {
		t.Error("an unbounded description would ride on every image this world ever draws")
	}
}

func mustResolve(t *testing.T, choice string) ArtStyle {
	t.Helper()
	s, err := ResolveArtStyle(choice)
	if err != nil {
		t.Fatalf("ResolveArtStyle(%q): %v", choice, err)
	}
	return s
}

// The choice has to survive creation, because every picture the world ever draws is resolved from
// this column — not from the request that made it. A style that is validated and then dropped gives
// the user a picker that does nothing.
func TestGenesisPersistsTheChosenStyle(t *testing.T) {
	for _, tc := range []struct {
		name   string
		choice string
		want   any
	}{
		{"a preset is stored by key", "anime", "anime"},
		{"a written style is stored whole", "custom:oil painting, heavy impasto", "custom:oil painting, heavy impasto"},
		{"no choice stays NULL rather than empty text", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			pool := testPool(t)
			t.Cleanup(pool.Close)

			doc, err := authorWorld(ctx, NewFakeWorldGenesisDriver(), testBrief, nil)
			if err != nil {
				t.Fatalf("authorWorld: %v", err)
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin: %v", err)
			}
			t.Cleanup(func() { _ = tx.Rollback(ctx) })

			worldID, err := commitWorldGenesis(ctx, tx, doc, testBrief, tc.choice)
			if err != nil {
				t.Fatalf("commitWorldGenesis: %v", err)
			}

			var got *string
			if err := tx.QueryRow(ctx, `SELECT art_style FROM world WHERE world_id = $1::uuid`, worldID).Scan(&got); err != nil {
				t.Fatalf("read back art_style: %v", err)
			}
			switch want := tc.want.(type) {
			case nil:
				if got != nil {
					t.Fatalf("art_style = %q, want NULL — blank is not a choice anyone made", *got)
				}
			case string:
				if got == nil || *got != want {
					t.Fatalf("art_style = %v, want %q", got, want)
				}
			}

			// And the stored value must resolve back to a style, or the column is unreadable by the
			// only code that consumes it.
			choice := ""
			if got != nil {
				choice = *got
			}
			if _, err := ResolveArtStyle(choice); err != nil {
				t.Fatalf("the stored choice does not resolve: %v", err)
			}
		})
	}
}
