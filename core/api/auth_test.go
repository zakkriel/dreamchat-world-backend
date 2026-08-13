package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// authEnv builds the three-variable lookup the gate reads.
func authEnv(email, pass, secret string) func(string) string {
	m := map[string]string{
		"DREAMCHAT_AUTH_EMAIL":    email,
		"DREAMCHAT_AUTH_PASSWORD": pass,
		"DREAMCHAT_AUTH_SECRET":   secret,
	}
	return func(k string) string { return m[k] }
}

// TestAuth_DisabledUnlessFullyConfigured pins the enablement rule: all three variables or nothing.
// A half-set configuration must read as DISABLED — a gate that only thinks it is up is worse than
// no gate, and CI/stack.sh (zero variables) must keep working untouched.
func TestAuth_DisabledUnlessFullyConfigured(t *testing.T) {
	if a := authFromEnv(authEnv("", "", "")); a != nil {
		t.Fatal("no variables set: want disabled (nil), got a live gate")
	}
	if a := authFromEnv(authEnv("a@b.c", "pw", "")); a != nil {
		t.Fatal("2 of 3 set: want disabled (nil), got a live gate")
	}
	if a := authFromEnv(authEnv("a@b.c", "pw", "s3cret")); a == nil {
		t.Fatal("all three set: want a live gate, got nil")
	}
}

// TestAuth_TokenIsDeterministicAndSecretKeyed pins the token derivation contract: same inputs, same
// token (a Railway restart must not log clients out); different secret, different token (knowing
// the scheme buys nothing without the secret).
func TestAuth_TokenIsDeterministicAndSecretKeyed(t *testing.T) {
	a1 := authFromEnv(authEnv("a@b.c", "pw", "s1"))
	a2 := authFromEnv(authEnv("a@b.c", "pw", "s1"))
	a3 := authFromEnv(authEnv("a@b.c", "pw", "s2"))
	if a1.token != a2.token {
		t.Fatalf("same inputs produced different tokens: %s vs %s", a1.token, a2.token)
	}
	if a1.token == a3.token {
		t.Fatal("different secrets produced the SAME token — derivation is not secret-keyed")
	}
}

// gateAround wraps a sentinel handler in the configured gate, mirroring main.go's composition.
func gateAround(a *authConfig) http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("inner"))
	})
	mux := http.NewServeMux()
	mux.Handle("/auth/login", a.loginHandler())
	mux.Handle("/", a.requireAuth(inner))
	return mux
}

// TestAuth_LoginIssuesTokenAndGateHonorsBothCarriers walks the whole contract end to end: login
// with the right credentials yields the token; the gate rejects a bare request with a JSON 401 and
// admits the token via BOTH carriers — the Authorization header (fetches) and ?token= (an <img>
// cannot send a header).
func TestAuth_LoginIssuesTokenAndGateHonorsBothCarriers(t *testing.T) {
	a := authFromEnv(authEnv("op@example.com", "hunter2", "s3cret"))
	h := gateAround(a)

	// Wrong password → 401, and the body must not distinguish which half was wrong.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"op@example.com","password":"wrong"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad password: status = %d, want 401", rec.Code)
	}

	// Right credentials → the derived token.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"op@example.com","password":"hunter2"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), a.token) {
		t.Fatalf("login body %q does not carry the derived token", rec.Body.String())
	}

	// No token → JSON 401, request never reaches the inner handler.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bare request: status = %d, want 401", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("gate 401 Content-Type = %q, want application/json", ct)
	}

	// Header carrier.
	req := httptest.NewRequest(http.MethodGet, "/worlds", nil)
	req.Header.Set("Authorization", "Bearer "+a.token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "inner" {
		t.Fatalf("header carrier: status = %d body = %q, want 200 %q", rec.Code, rec.Body.String(), "inner")
	}

	// Query carrier (the <img> path).
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds/x/images/y?token="+a.token, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("query carrier: status = %d, want 200", rec.Code)
	}

	// A wrong token on either carrier stays out.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds?token=nope", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong query token: status = %d, want 401", rec.Code)
	}
}

// TestAuth_OptionsExempt pins the composition safety valve: even if the wrapping order ever puts
// the gate outside CORS, a preflight (which by definition carries no credentials) must pass.
func TestAuth_OptionsExempt(t *testing.T) {
	a := authFromEnv(authEnv("op@example.com", "hunter2", "s3cret"))
	rec := httptest.NewRecorder()
	gateAround(a).ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/worlds", nil))
	if rec.Code == http.StatusUnauthorized {
		t.Fatal("OPTIONS was gated — preflights must never require the token")
	}
}
