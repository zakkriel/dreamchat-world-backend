package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// An ADR and the code it governs must point at each other, and a test proves they still do.
//
// WHY THIS REPLACED A ROUTING FILE. The first attempt at "load only the docs relevant to what you are
// editing" was a hand-written map of path globs to documents. Measured against the tree it covered
// 231 of 322 files: 91 matched nothing, 64 matched several routes with no rule for which won, and
// `prompts/world_genesis.txt` routed to "prompts" but not to world creation — reproducing, inverted,
// the exact scoping failure the map existed to prevent. A hand-maintained map of a moving tree is a
// second job nobody does, and its holes are invisible.
//
// So the pointer lives ON the code instead. A governed file says which ADR governs it in its first
// lines; the ADR names the file as its evidence (D-9). Neither can drift without this test failing,
// and an agent opening the file sees its governing decision before it reads a line of logic.
//
// It does NOT claim to cover everything — that claim is what made the routing file dishonest. It
// covers exactly the files an ADR asserts as evidence, which is the set where a silent divergence
// between decision and code actually costs something.

var adrEvidenceRef = regexp.MustCompile(`core/api/[a-z_]+\.go`)
var governedBy = regexp.MustCompile(`Governed-by:\s*(ADR-P\d{3})`)

// adrsUnderTest are the operational ADRs that cite code as evidence.
const adrDir = "../../docs/30_architecture/adr"

func TestADRsAndTheCodeTheyGovernPointAtEachOther(t *testing.T) {
	adrs, err := filepath.Glob(filepath.Join(adrDir, "ADR-P0*.md"))
	if err != nil || len(adrs) == 0 {
		t.Skipf("ADR directory not reachable from this working directory (%v)", err)
	}

	// ADR -> the core/api files it names as evidence.
	cited := map[string][]string{}
	for _, path := range adrs {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		id := strings.SplitN(filepath.Base(path), "_", 2)[0]
		for _, ref := range adrEvidenceRef.FindAllString(string(body), -1) {
			if strings.HasSuffix(ref, "_test.go") {
				continue // a test is evidence, but it is not where the decision is implemented
			}
			cited[id] = append(cited[id], ref)
		}
	}
	if len(cited) == 0 {
		t.Fatal("no ADR cites any core/api file — this test is asserting nothing")
	}

	for adr, files := range cited {
		for _, rel := range files {
			name := filepath.Base(rel)
			body, err := os.ReadFile(name)
			if os.IsNotExist(err) {
				t.Errorf("%s cites %s as evidence, and that file does not exist", adr, rel)
				continue
			}
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}

			// Only the first 60 lines: the pointer has to be where a reader lands, not buried.
			head := strings.Join(strings.SplitN(string(body), "\n", 60), "\n")
			found := governedBy.FindAllStringSubmatch(head, -1)
			if len(found) == 0 {
				t.Errorf("%s is named as evidence by %s but carries no `Governed-by: %s` line in its first 60 lines — "+
					"an agent editing it would never learn which decision it is bound by", name, adr, adr)
				continue
			}
			var names []string
			for _, m := range found {
				names = append(names, m[1])
			}
			if !slicesContain(names, adr) {
				t.Errorf("%s says Governed-by: %s but %s claims it as evidence — the decision and the code disagree",
					name, strings.Join(names, ", "), adr)
			}
		}
	}
}

// A Governed-by line must name an ADR that exists. A pointer to a deleted or renamed decision is
// worse than none: it reads as authority and resolves to nothing.
func TestEveryGovernedByNamesARealADR(t *testing.T) {
	adrs, err := filepath.Glob(filepath.Join(adrDir, "ADR-P0*.md"))
	if err != nil || len(adrs) == 0 {
		t.Skipf("ADR directory not reachable (%v)", err)
	}
	known := map[string]bool{}
	for _, a := range adrs {
		known[strings.SplitN(filepath.Base(a), "_", 2)[0]] = true
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, m := range governedBy.FindAllStringSubmatch(string(body), -1) {
			if !known[m[1]] {
				t.Errorf("%s claims Governed-by: %s, which is not an ADR in %s", f, m[1], adrDir)
			}
		}
	}
}

func slicesContain(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
