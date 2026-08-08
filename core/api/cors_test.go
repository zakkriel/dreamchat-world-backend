package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// okHandler stands in for the mux. served flips only if the wrapped handler was actually reached,
// which is what separates "preflight answered by the middleware" from "request passed through".
type okHandler struct{ served bool }

func (h *okHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.served = true
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

const feOrigin = "http://localhost:5173"

func corsRequest(t *testing.T, allowed []string, method, origin string, preflight bool) (*httptest.ResponseRecorder, *okHandler) {
	t.Helper()
	next := &okHandler{}
	req := httptest.NewRequest(method, "http://localhost:8080/worlds/22222222-2222-2222-2222-222222222222/beats", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if preflight {
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type")
	}
	rec := httptest.NewRecorder()
	withCORS(next, allowed).ServeHTTP(rec, req)
	return rec, next
}

// A preflight from an allowlisted origin is answered by the middleware itself — the router never
// sees it (it only matches GET/POST and would 404). Without this the FE cannot POST a beat at all.
func TestCORS_PreflightFromAllowedOriginIsApproved(t *testing.T) {
	rec, next := corsRequest(t, []string{feOrigin}, http.MethodOptions, feOrigin, true)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if next.served {
		t.Fatal("preflight reached the wrapped handler; it must be answered by the middleware")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != feOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want the echoed origin %q", got, feOrigin)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != corsAllowedMethods {
		t.Fatalf("Access-Control-Allow-Methods = %q, want %q", got, corsAllowedMethods)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != corsAllowedHeaders {
		t.Fatalf("Access-Control-Allow-Headers = %q, want %q", got, corsAllowedHeaders)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin (the response varies per origin)", got)
	}
}

// The wall: an origin that is not on the list gets no CORS headers, so the browser blocks it. Its
// preflight is answered 403 rather than falling through to the router's 404 — a misconfigured
// allowlist must be legible in the network tab, not disguised as a missing route.
func TestCORS_PreflightFromDisallowedOriginIsRefused(t *testing.T) {
	rec, next := corsRequest(t, []string{feOrigin}, http.MethodOptions, "http://evil.example", true)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("disallowed preflight status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if next.served {
		t.Fatal("disallowed preflight reached the wrapped handler")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty for a disallowed origin", got)
	}
}

// A plain cross-origin GET from an allowlisted origin is served normally AND carries the echoed
// origin, which is what makes the FE's compendium and scene reads usable from the browser.
func TestCORS_SimpleRequestFromAllowedOriginIsEchoed(t *testing.T) {
	rec, next := corsRequest(t, []string{feOrigin}, http.MethodGet, feOrigin, false)

	if !next.served {
		t.Fatal("a simple request must reach the wrapped handler")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != feOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, feOrigin)
	}
}

// A disallowed origin still gets its response — the browser, not the server, enforces the block —
// but with no CORS headers, so it never becomes readable JavaScript.
func TestCORS_SimpleRequestFromDisallowedOriginCarriesNoHeaders(t *testing.T) {
	rec, next := corsRequest(t, []string{feOrigin}, http.MethodGet, "http://evil.example", false)

	if !next.served {
		t.Fatal("a non-preflight request must still be served")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

// No Origin header = not a browser cross-origin call. curl, CI's payload generator and any
// server-to-server caller must be untouched, including the absence of a Vary header.
func TestCORS_RequestWithoutOriginIsUntouched(t *testing.T) {
	rec, next := corsRequest(t, []string{feOrigin}, http.MethodGet, "", false)

	if !next.served {
		t.Fatal("an Origin-less request must be served")
	}
	if got := rec.Header().Get("Vary"); got != "" {
		t.Fatalf("Vary = %q, want empty for a request with no Origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

// Unset DREAMCHAT_CORS_ORIGINS means CORS is off entirely — the default must never silently allow
// an origin nobody configured.
func TestCORS_EmptyAllowlistAddsNothing(t *testing.T) {
	rec, next := corsRequest(t, nil, http.MethodGet, feOrigin, false)

	if !next.served {
		t.Fatal("with CORS off the request must still be served")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty when the allowlist is empty", got)
	}
}

// Parsing is the deploy-time contract: whitespace and a trailing slash are forgiven (both are how
// people actually paste an origin), blanks vanish, and anything that cannot work as an exact origin
// is reported so main can refuse to boot instead of serving an unreachable API.
func TestCORSOrigins_ParsesListAndFlagsUnusableEntries(t *testing.T) {
	t.Setenv(corsOriginsEnv, " http://localhost:5173/ , ,https://app.example.com,*,app.example.com")

	allowed, bad := corsOrigins()

	want := []string{"http://localhost:5173", "https://app.example.com"}
	if len(allowed) != len(want) {
		t.Fatalf("allowed = %v, want %v", allowed, want)
	}
	for i := range want {
		if allowed[i] != want[i] {
			t.Fatalf("allowed = %v, want %v", allowed, want)
		}
	}
	wantBad := []string{"*", "app.example.com"}
	if len(bad) != len(wantBad) {
		t.Fatalf("bad = %v, want %v (wildcard and scheme-less entries are not usable origins)", bad, wantBad)
	}
	for i := range wantBad {
		if bad[i] != wantBad[i] {
			t.Fatalf("bad = %v, want %v", bad, wantBad)
		}
	}
}

func TestCORSOrigins_UnsetYieldsNothing(t *testing.T) {
	t.Setenv(corsOriginsEnv, "")

	allowed, bad := corsOrigins()

	if len(allowed) != 0 || len(bad) != 0 {
		t.Fatalf("corsOrigins() = (%v, %v), want both empty when %s is unset", allowed, bad, corsOriginsEnv)
	}
}
