package main

import (
	"os"
	"os/exec"
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

var adrEvidenceRef = regexp.MustCompile(`core/api/[a-z0-9_]+\.go`)
var governedBy = regexp.MustCompile(`Governed-by:\s*(ADR-P\d{3})`)

// Evidence-block guard (failure-log #31, gated here since 2026-08-27): inside an ADR's
// `**Evidence` paragraph, a BARE backticked `name.go` is invisible to adrEvidenceRef — the gate
// checked one of four files while appearing to check all four. Bare names in prose elsewhere
// (history, cross-repo mentions) stay legal; only the Evidence paragraph claims coupling.
var evidenceBlock = regexp.MustCompile(`(?m)^\*\*Evidence[^\n]*(?:\n[^\n*][^\n]*)*`)
var bareGoName = regexp.MustCompile("`([a-z0-9_]+\\.go)`")

// adrsUnderTest are the operational ADRs that cite code as evidence.
const adrDir = "../../docs/law/adr"

func TestADRsAndTheCodeTheyGovernPointAtEachOther(t *testing.T) {
	adrs, err := filepath.Glob(filepath.Join(adrDir, "ADR-P*.md"))
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
		// The #31 guard: bare evidence names fail loudly instead of being silently skipped.
		// `_test.go` names are exempt for the same reason the coupling skips them above — a test
		// is evidence of green, not where the decision is implemented, so a bare test name
		// cannot cause a #31-class miss.
		for _, block := range evidenceBlock.FindAllString(string(body), -1) {
			for _, m := range bareGoName.FindAllStringSubmatch(block, -1) {
				if strings.HasSuffix(m[1], "_test.go") {
					continue
				}
				t.Errorf("%s Evidence names bare `%s` — the gate cannot see it; write `core/api/%s`",
					id, m[1], m[1])
			}
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
	adrs, err := filepath.Glob(filepath.Join(adrDir, "ADR-P*.md"))
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

// Every id a Governed-by line cites must resolve — not just ADR-P###. 23 of the live headers name
// register rules, engine ADRs and SPEC items (`D-1`, `ADR-009`, `SPEC-015`, …), and until
// 2026-08-27 none of them was validated by anything: a header claiming the invented `D-99` stayed
// green in the one
// place agents are told to look first (AGENTS.md "Open the file you are about to change").
//
// The resolver is NOT reimplemented here (D-6: a second copy of the law is the one that drifts —
// the harness's duplicated --ask scorer already drifted once). We broad-CAPTURE the id tokens and
// pipe each to ci/check_citations.sh, the one canonical resolver. Two failure modes per id:
//   - nonzero exit        → the id does not resolve (invented or stale);
//   - "no identifiers cited" while we captured one → the token is malformed in a way the gate's
//     exact-shape grammar silently drops (case/padding) — the degrade class must go red here.
//
// Anchored to a line-initial `// Governed-by:` comment — the header idiom — so that regex
// literals and t.Errorf format strings in THIS file do not read as headers.
var governedByLine = regexp.MustCompile(`(?m)^// Governed-by:[^\n]*`)
var citableID = regexp.MustCompile(`\b(ADR-P?[0-9]+|SPEC-[0-9]+|GA-[0-9]+|[A-Z]-[0-9]+)\b`)

// Relative to cmd.Dir (the backend root) — os/exec resolves a relative Path against Dir, not
// against the test's own working directory.
const citationGate = "ci/check_citations.sh"

// gateResolves pipes one id to the citation gate. Returns false on nonzero exit OR when the gate
// could not even extract the token (the silent-degrade class).
func gateResolves(t *testing.T, id string) bool {
	t.Helper()
	cmd := exec.Command(citationGate, "-")
	cmd.Dir = "../.."
	cmd.Stdin = strings.NewReader("Rules: " + id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return !strings.Contains(string(out), "no identifiers cited")
}

func TestEveryGovernedByIdResolves(t *testing.T) {
	if _, err := os.Stat(filepath.Join("../..", "ci/check_citations.sh")); err != nil {
		t.Skipf("citation gate not reachable from this working directory (%v)", err)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	found := 0
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, line := range governedByLine.FindAllString(string(body), -1) {
			ids := citableID.FindAllString(line, -1)
			if len(ids) == 0 {
				t.Errorf("%s carries a Governed-by line with no recognizable id: %q", f, line)
				continue
			}
			for _, id := range ids {
				found++
				if !gateResolves(t, id) {
					t.Errorf("%s claims Governed-by id %s, which does not resolve through ci/check_citations.sh — an invented or malformed constraint in a file header", f, id)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no Governed-by ids found anywhere — this test is asserting nothing")
	}

	// Permanent selftest: the pipe must be able to go red. A resolver you have not watched reject
	// a fake id has not been tested (governance.md §3 step 5).
	t.Run("invented_id_must_fail", func(t *testing.T) {
		if gateResolves(t, "D-99") {
			t.Fatal("the citation gate accepted invented id D-99 — the resolver or this pipe is broken")
		}
	})
}

func slicesContain(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
