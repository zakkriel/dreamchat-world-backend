package main

// auth.go — the single-user login gate (2026-08-12). Exists because the deployed API was a public,
// unauthenticated WRITE surface: anyone with the URL — including a test agent, which is exactly what
// happened — could POST beats, spend real model credits, and write permanent canon into the live
// world. This gate closes that door with the smallest honest mechanism: one operator identity from
// the environment, one deterministic bearer token, no accounts table, no sessions, no new
// dependencies.
//
// ENABLEMENT IS EXPLICIT AND FAIL-OPEN FOR DEV, FAIL-CLOSED FOR PROD: auth turns on only when all
// three of DREAMCHAT_AUTH_EMAIL / DREAMCHAT_AUTH_PASSWORD / DREAMCHAT_AUTH_SECRET are set. Unset ⇒
// the API serves exactly as before (stack.sh's fake-bridge stack, CI, and every existing test keep
// working with zero changes), and the boot log says so loudly — the same posture CORS takes with an
// unset allowlist.
//
// THE TOKEN IS DERIVED, NOT STORED: hex(HMAC-SHA256(secret, email+"\n"+password)). Deterministic, so
// a Railway restart does not log every client out; secret-keyed, so knowing the derivation buys an
// attacker nothing without the secret. Rotating either the password or the secret rotates the token.
//
// TWO CARRIERS, one gate: `Authorization: Bearer <token>` for ordinary fetches, `?token=<token>` for
// requests that cannot set headers — an <img> loading /worlds/{w}/images/... cannot send a header,
// and the CORS file already records the precedent ("the trace key rides a query param").

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// authConfig is the whole gate: the one operator identity and the derived bearer token.
type authConfig struct {
	email string
	token string // hex(HMAC-SHA256(secret, email+"\n"+password))

	// password kept only for the constant-time login comparison; never logged, never echoed.
	password string
}

// authFromEnv reads the three variables and derives the token. Returns nil (auth disabled) unless
// all three are present — a half-set configuration is treated as unset and reported, because a gate
// that only THINKS it is up is worse than no gate.
func authFromEnv(lookup func(string) string) *authConfig {
	email, password, secret := lookup("DREAMCHAT_AUTH_EMAIL"), lookup("DREAMCHAT_AUTH_PASSWORD"), lookup("DREAMCHAT_AUTH_SECRET")
	set := 0
	for _, v := range []string{email, password, secret} {
		if v != "" {
			set++
		}
	}
	switch set {
	case 3:
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(email + "\n" + password))
		return &authConfig{email: email, password: password, token: hex.EncodeToString(mac.Sum(nil))}
	case 0:
		return nil
	default:
		log.Printf("auth: DISABLED — %d of 3 DREAMCHAT_AUTH_{EMAIL,PASSWORD,SECRET} set; set all three to enable the gate", set)
		return nil
	}
}

// loginHandler serves POST /auth/login: JSON {email,password} in, {token} out. Constant-time
// comparisons on both fields so a probe cannot time its way to either. The same 401 body for a
// wrong email and a wrong password — no oracle about which half was right.
func (a *authConfig) loginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "bad body"})
			return
		}
		emailOK := subtle.ConstantTimeCompare([]byte(in.Email), []byte(a.email)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(in.Password), []byte(a.password)) == 1
		if !emailOK || !passOK {
			writeAuthJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		writeAuthJSON(w, http.StatusOK, map[string]string{"token": a.token})
	})
}

// requireAuth wraps the API: every request must carry the bearer token (header or ?token=), except
// POST /auth/login itself. OPTIONS never reaches here on the CORS path (withCORS answers preflights
// first), but is exempted anyway so the gate composes safely if the wrapping order ever changes.
func (a *authConfig) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || r.URL.Path == "/auth/login" {
			next.ServeHTTP(w, r)
			return
		}
		tok := r.URL.Query().Get("token")
		if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
			tok = h[7:]
		}
		if subtle.ConstantTimeCompare([]byte(tok), []byte(a.token)) != 1 {
			writeAuthJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeAuthJSON is the gate's one response writer — always JSON, so a browser client never has to
// sniff whether a 401 came from the gate (JSON) or a proxy (HTML).
func writeAuthJSON(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// withAuth composes the gate around the mux when configured; the identity function otherwise.
// Called from main with the mux BEFORE withCORS wraps it, so preflights stay ungated.
func withAuth(next http.Handler) http.Handler {
	a := authFromEnv(os.Getenv)
	if a == nil {
		log.Printf("auth: disabled (DREAMCHAT_AUTH_EMAIL/PASSWORD/SECRET unset) — API is open; set all three to require login")
		return next
	}
	log.Printf("auth: ENABLED for %s — all routes require the bearer token except POST /auth/login", a.email)
	mux := http.NewServeMux()
	mux.Handle("/auth/login", a.loginHandler())
	mux.Handle("/", a.requireAuth(next))
	return mux
}
