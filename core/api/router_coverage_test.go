package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── every served route is served, and every registered handler is reachable ──────────────────────
//
// WHY THIS FILE EXISTS.
//
// `main.go:35-38` has said it since the router was written: "a handler that works perfectly and was
// never added to this slice is a 404 in production and a green suite, and that is exactly the failure
// mode 'hand-drive everything' exists to catch." The comment was right and the test was never written.
//
// QA-SPAN-2026-08-11 §2 measured the consequence: 7 of 8 route registrations could be deleted one at a
// time with all 285 tests passing, "including POST /worlds/{w}/beats" — the only write path in the
// service. Re-measured 2026-08-26 with `ci/mutate.sh` against a clean baseline: **4 of 12 still
// deletable** — the Actor/Location/Artifact pages, the compendium indexes, the timeline, and
// scene/current. Every one of those is a surface a player looks at.
//
// WHY IT IS BIDIRECTIONAL, which is the part that makes it a class-killer rather than four tests:
//
//   forward   every route in the table below must be claimed by SOME handler.
//             Delete a registration -> its route goes unclaimed -> red.
//
//   backward  every handler in the router must be claimed BY some route in the table.
//             Add a registration without a table row -> unreached handler -> red.
//
// Without the backward half, this test decays the moment someone adds a route: it would keep passing
// while covering less and less, which is the same silent-narrowing failure QA found everywhere else.
// The migration-declaration check in `harness/check.sh areas` is bidirectional for the same reason.
//
// This test drives the COMPOSED router, never a handler in isolation. A handler tested directly is
// evidence about the handler; only the composed router is evidence about what the service serves.

// A router built with nil dependencies. Match() never touches the pool, the bridge or the image
// client — it reads the method and the path — so nil is the honest fixture here: it makes the test
// structurally incapable of asserting anything about handler behaviour, which is not its job.
func coverageRouter() *router { return newRouter(nil, false, nil, nil) }

const (
	w1 = "22222222-2222-2222-2222-222222222222" // any well-formed world id
	e1 = "2ac70000-0000-0000-0000-0000000000a1" // any well-formed entity id
)

// routeTable is the contract: what this service serves. One row per (method, path) shape.
// Several rows may legitimately share a handler — /beats and /beats/continue do — but every handler
// must appear at least once, and the backward assertion below enforces it.
var routeTable = []struct {
	method, path, why string
}{
	{http.MethodGet, "/worlds/art-styles", "the art style catalogue world creation offers"},

	{http.MethodGet, "/worlds/" + w1 + "/compendium/actors/" + e1 + "/page", "Actor page"},
	{http.MethodGet, "/worlds/" + w1 + "/compendium/locations/" + e1 + "/page", "Location page"},
	{http.MethodGet, "/worlds/" + w1 + "/compendium/artifacts/" + e1 + "/page", "Artifact page"},

	{http.MethodGet, "/worlds/" + w1 + "/compendium/actors", "Actors index"},
	{http.MethodGet, "/worlds/" + w1 + "/compendium/locations", "Locations index"},
	{http.MethodGet, "/worlds/" + w1 + "/compendium/artifacts", "Artifacts index"},
	{http.MethodGet, "/worlds/" + w1 + "/compendium/timeline", "Timeline"},

	{http.MethodGet, "/worlds/" + w1 + "/carrying", "the Carrying overlay"},
	{http.MethodGet, "/worlds/" + w1 + "/transcript", "the persistent transcript"},
	{http.MethodGet, "/worlds/" + w1 + "/scene/current", "where you are and who is present"},

	{http.MethodPost, "/worlds/" + w1 + "/beats", "THE ONLY WRITE PATH — QA found this one deletable"},
	{http.MethodPost, "/worlds/" + w1 + "/beats/continue", "the continue press, same handler"},

	{http.MethodGet, "/worlds", "the world directory (SPEC-028)"},
	{http.MethodPost, "/worlds", "world creation"},

	{http.MethodPost, "/worlds/interview", "world creation asks the next question"},
	{http.MethodPost, "/worlds/genesis", "world creation authors a world"},

	{http.MethodPost, "/worlds/" + w1 + "/refresh", "mint a successor world, archive the source"},

	{http.MethodGet, "/worlds/" + w1 + "/images/" + e1, "presigned image redirect"},
	{http.MethodPost, "/worlds/" + w1 + "/images/portraits", "the explicit portrait trigger"},
}

// claimant returns the index of the first handler that claims the request, or -1.
// It mirrors router.ServeHTTP's first-match-wins dispatch exactly; if that loop ever changes, this
// test is measuring the wrong thing and should change with it.
func claimant(rt *router, method, path string) int {
	req := httptest.NewRequest(method, path, nil)
	for i, h := range rt.handlers {
		if h.Match(req) {
			return i
		}
	}
	return -1
}

// FORWARD: every route the service promises is claimed by something.
func TestRouterCoverage_EveryRouteIsServed(t *testing.T) {
	rt := coverageRouter()
	for _, r := range routeTable {
		if got := claimant(rt, r.method, r.path); got < 0 {
			t.Errorf("%s %s is UNSERVED — %s\n    a registration is missing from newRouter; in production this is a 404",
				r.method, r.path, r.why)
		}
	}
}

// BACKWARD: every handler registered is reached by the table. This is what stops the test decaying.
func TestRouterCoverage_EveryHandlerIsReached(t *testing.T) {
	rt := coverageRouter()
	reached := make([]bool, len(rt.handlers))
	for _, r := range routeTable {
		if i := claimant(rt, r.method, r.path); i >= 0 {
			reached[i] = true
		}
	}
	for i, ok := range reached {
		if !ok {
			t.Errorf("handler at newRouter index %d (%T) is reached by NO row in routeTable\n"+
				"    add the route it serves to the table, or it is untested and may be deleted silently",
				i, rt.handlers[i])
		}
	}
}

// TestRouterCoverage_ArtStylesPrecedesIDMatchers was deleted 2026-08-27: a proven tautology. Moving
// the art-styles registration to the LAST position still passed, because `worldArtStylesRoute` is
// `^/worlds/art-styles$` and every id matcher is `([0-9a-fA-F-]{36})`, so a 10-character literal can
// never be swallowed. The ordering constraint `main.go:41-43` describes is not live today. The only
// mutation that reddened it — breaking the art-styles regex — also reddens the two tests above, and
// the permissive-matcher case it claimed to guard is caught by UnknownPathsAreClaimedByNobody below.
// Strictly subsumed, and never mutation-verified despite shipping under a "7 of 7 caught" claim.
// Receipts: docs/00_workspace/review-test-suite-2026-08-26.md §Q1.

// A path nothing serves must reach nobody. Without this, a matcher whose regex is accidentally
// permissive — `.*` instead of a uuid class — would satisfy both tests above while swallowing
// everything, and first-match-wins means it would shadow every handler after it.
func TestRouterCoverage_UnknownPathsAreClaimedByNobody(t *testing.T) {
	rt := coverageRouter()
	for _, p := range []struct{ method, path string }{
		{http.MethodGet, "/worlds/" + w1 + "/does-not-exist"},
		{http.MethodGet, "/health"},
		{http.MethodDelete, "/worlds/" + w1 + "/beats"},
		{http.MethodGet, "/worlds/not-a-uuid/carrying"},
	} {
		if i := claimant(rt, p.method, p.path); i >= 0 {
			t.Errorf("%s %s was claimed by %T — an over-permissive route shadows everything after it",
				p.method, p.path, rt.handlers[i])
		}
	}
}
