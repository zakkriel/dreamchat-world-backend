package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A proper name written into a seat prompt's few-shot examples rides in EVERY prompt of that seat, in
// EVERY world, forever. When it collides with a name in the world actually being played, the prompt
// itself becomes a naming-wall leak: the model is handed a canonical name with no knowledge path,
// before a single perception is assembled.
//
// Caught the hard way — a prompt fix for the misattribution bug used "Kade" in its example and
// TestWall_NameStringConfinedToKnower failed. That test guards ONE name in one seat. This one guards
// the play seeds' whole proper-name cast across EVERY seat prompt.
//
// Only PROPER names are listed. The seeds also register entities whose canonical_name is an ordinary
// noun — "the bar", "Dock", "Market", "Square", "Road", "Tavern", "Cellar", "Front Door", "Player" —
// and a static scan cannot tell those from ordinary English, which prompts are made of. Banning them
// here would flag "the muscle steps up to the bar" and teach the next person to disable the test.
// Those names are protected at runtime by the naming wall (fn_unearned_names / NamingWall), which
// knows which world it is in; this test protects the class a static scan CAN judge.
func TestPrompts_CarryNoSeedProperNames(t *testing.T) {
	files, err := filepath.Glob("prompts/*.txt")
	if err != nil || len(files) == 0 {
		t.Fatalf("no seat prompts found to check (err=%v) — this test must not pass vacuously", err)
	}

	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		hay := strings.ToLower(string(body))
		for _, name := range seedProperNames {
			// Word-bounded so a banned name never matches inside a longer word.
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(strings.ToLower(name)) + `\b`)
			if loc := re.FindStringIndex(hay); loc != nil {
				start := max(0, loc[0]-60)
				end := min(len(hay), loc[1]+60)
				t.Errorf("%s hardcodes the seed name %q — every prompt this seat sends will carry it, "+
					"in every world, whether or not the viewer has earned it. Use an invented name.\n  …%s…",
					filepath.Base(path), name, string(body)[start:end])
			}
		}
	}
}

// seedProperNames is the ban list, declared once so the guard and its self-check cannot disagree.
var seedProperNames = []string{
	// seed_drowned_lantern.sql — the play seed
	"Mara", "Jonas", "Kade", "Drowned Lantern", "Dock Street", "Hooded Woman",
	"Hooded Companion", "Ballast Crate", "Ballast Stone", "Sealed Note", "Cellar Key",
	"Cellar Hatch", "Harbormaster", "Harbor Quarter", "Vael", "Salt Quay",
	// seed_mara_0A.sql — the deterministic fixture world
	"Seren", "Reyna", "Dark Foxes", "Fox-ears", "Downfall Market",
}

// The guard is only worth having if it can fail: a name planted in a temporary prompt file must be
// caught. Without this, a broken glob or a silently-empty ban list would look like success forever.
func TestPrompts_SeedNameGuardActuallyFires(t *testing.T) {
	dir := t.TempDir()
	planted := filepath.Join(dir, "planted.txt")
	if err := os.WriteFile(planted, []byte("Example: Mara sets a tankard on the bar.\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	body, err := os.ReadFile(planted)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	re := regexp.MustCompile(`(?i)\bmara\b`)
	if !re.Match(body) {
		t.Fatal("the matching rule used by the guard failed to find a planted seed name")
	}
	// ...and an ordinary noun that is ALSO a seed canonical_name must never be banned ON ITS OWN, or
	// the guard would flag ordinary prose and get switched off. Exact membership, not substring:
	// "Downfall Market" is a proper name that happens to contain "Market", and belongs on the list.
	for _, generic := range []string{"the bar", "Tavern", "Market", "Player", "Dock", "Square", "Road", "Cellar"} {
		for _, banned := range seedProperNames {
			if strings.EqualFold(banned, generic) {
				t.Errorf("%q is an ordinary noun and must not be statically banned — the runtime wall owns it", generic)
			}
		}
	}
}
