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
	// The worlds GENESIS is developed against. Same hazard, different door: the play seeds leak into a
	// beat prompt, these leak into a fill prompt — and a fill prompt is the standing instruction for
	// every world anybody ever creates. Added 2026-09-01 after an Andantes example was found baked into
	// the lives prompt.
	//
	// Proper names only, for the reason this file already gives: "pulse", "skip" and "room" are ordinary
	// English, prompts are made of ordinary English, and banning them would flag real prose and teach the
	// next person to switch the guard off.
	"Andantes", "Andante", "Ossa", "Cola Baja", "Del Vas", "Tercera Hembra", "Auscultadora",
	"Auscultador", "Trompetilla", "Colegio de Auscultadores", "Gremio de Peso", "Los Convergentes",
	// bridge_fakes.go — the deterministic fill fake's world, which must never reach a real prompt.
	"Counting Room", "Loading Yard", "Night Loaders",
}

// THE OTHER HALF OF THE STANDING INSTRUCTION. A fill stage's instruction is not only `prompts/world_fill.txt`
// — most of it is the work item's own text, assembled in Go, and that text is equally fixed and equally
// sent to every world. The 2026-09-01 leak was there and not in the prompt file, so a guard that reads only
// `prompts/*.txt` could never have seen it.
//
// The prompts are assembled with NO world content: a neutral brief, an empty identity and a document
// holding only placeholder names. Anything a banned name matches in the result therefore came from the
// fixed instruction, which is the only thing under test here.
//
// It scans every stage the schedule can produce, so a stage invented later is covered without anybody
// remembering to add it.
func TestWorkItemPrompts_CarryNoSeedProperNames(t *testing.T) {
	doc := &genesisDoc{}
	doc.Places = []genesisPlace{
		{CanonicalName: "PlaceOne", Relevance: 2, Descriptor: "d", Kind: "k", ExtentClass: "vast", Tension: "normal"},
		{CanonicalName: "PlaceTwo", Within: "PlaceOne", Relevance: 2, Descriptor: "d", Kind: "k", ExtentClass: "small", Tension: "calm"},
	}
	doc.Factions = []genesisFaction{{CanonicalName: "FactionOne", Relevance: 2, Descriptor: "d", Kind: "faction", Tag: "t"}}
	doc.Concepts = []genesisConcept{{CanonicalName: "ConceptOne", WhatItIs: "w", Descriptor: "d", Relevance: 2}}
	doc.Cast = []genesisActor{{CanonicalName: "PersonOne", StartsIn: "PlaceTwo", Relevance: 2, Tag: "t", Descriptor: "d"}}
	doc.Objects = []genesisObject{{CanonicalName: "ObjectOne", Descriptor: "d", Kind: "k", Relevance: 2}}

	b := budgetForDepth(3)
	items := []workItem{conceptsWork(), scaffoldOneWork(b), arrivalWork()}
	items = append(items, scaffoldTwoSchedule(doc, b)...)
	for _, wave := range contentSchedule(doc, b) {
		items = append(items, wave...)
	}
	items = append(items, afterCanonSchedule(doc, b)...)

	stages := map[string]bool{}
	for _, it := range items {
		if stages[it.ID] {
			continue // the text is fixed per stage; the subject only changes which names are quoted
		}
		stages[it.ID] = true
		body := buildWorldFillPrompt(&worldIdentity{}, it, "a world", nil, doc, "")
		hay := strings.ToLower(body)
		for _, name := range seedProperNames {
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(strings.ToLower(name)) + `\b`)
			if loc := re.FindStringIndex(hay); loc != nil {
				start := max(0, loc[0]-60)
				end := min(len(hay), loc[1]+60)
				t.Errorf("the %q work item hardcodes %q — that text is the standing instruction for every "+
					"world anybody creates.\n  …%s…", it.ID, name, body[start:end])
			}
		}
	}
	if len(stages) < 8 {
		t.Fatalf("only %d stages scanned (%v) — this guard must not pass by covering nothing", len(stages), stages)
	}
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
