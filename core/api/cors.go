package main

import (
	"net/http"
	"os"
	"strings"
)

// SPEC-021 — CORS for the frontend origin. The FE (dreamchat-frontend) and this API are separate
// services on separate origins (Vite dev on :5173 locally, separate Railway services deployed), so
// without these headers a browser cannot call this API at all: a preflighted POST /worlds/{w}/beats
// never reaches the router, and every cross-origin GET is discarded by the browser after the fact.
//
// The allowlist is EXACT-MATCH and comes from DREAMCHAT_CORS_ORIGINS alone — a comma-separated list
// of full origins ("http://localhost:5173,https://app.example.com"). There is deliberately no
// built-in default and no debug-mode default: deployed FE origins are not known here, and a
// hardcoded origin is exactly the drift this repo's anti-invention rule forbids. Unset ⇒ CORS is
// OFF, logged at boot, and the API still serves same-origin, curl, and server-to-server callers
// unchanged. The wildcard "*" is REJECTED at boot rather than honoured: it is never the right answer
// for an API that will carry per-viewer projections, and silently accepting it would hide the
// misconfiguration until something leaked.
//
// No Access-Control-Allow-Credentials: the FE sends no cookies (the trace key rides a query param).
// The origin is echoed rather than "*" anyway, so turning credentials on later is a one-line change
// here and nowhere else.
const corsOriginsEnv = "DREAMCHAT_CORS_ORIGINS"

// corsAllowedMethods / corsAllowedHeaders are the API's whole surface: three page GETs, three index
// GETs, a timeline GET, a scene GET, and two beat POSTs that carry a JSON body. Nothing else is
// advertised, so nothing else is preflight-approved.
const (
	corsAllowedMethods = "GET, POST, OPTIONS"
	corsAllowedHeaders = "Content-Type"
	corsMaxAgeSeconds  = "600"
)

// corsOrigins reads and normalises the allowlist. Returns the exact origins to accept, plus any
// entry that is structurally unusable so main can refuse to boot on a misconfiguration instead of
// quietly serving an API the frontend cannot reach.
func corsOrigins() (allowed []string, bad []string) {
	for _, raw := range strings.Split(os.Getenv(corsOriginsEnv), ",") {
		o := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "/"))
		switch {
		case o == "":
			continue
		case o == "*":
			bad = append(bad, o)
		case !strings.HasPrefix(o, "http://") && !strings.HasPrefix(o, "https://"):
			bad = append(bad, o)
		default:
			allowed = append(allowed, o)
		}
	}
	return allowed, bad
}

// withCORS wraps the whole mux — NOT the router — because a preflight is an OPTIONS request to a
// real path, and the router only answers GET/POST and would 404 it (verified by hand against the
// running server before this existed). Headers are written before next.ServeHTTP, so they land
// ahead of the first SSE flush on the streaming beat routes.
//
// A request with no Origin header is not a browser cross-origin request: it passes through
// untouched. An Origin that is not on the allowlist gets NO CORS headers — the browser then blocks
// it, which is the point — and a preflight from such an origin is answered 403 rather than 404, so
// a misconfigured allowlist is legible in the network tab instead of looking like a missing route.
func withCORS(next http.Handler, allowed []string) http.Handler {
	if len(allowed) == 0 {
		return next
	}
	index := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		index[o] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		// The response body depends on the request's Origin, so it is not cacheable across origins.
		w.Header().Add("Vary", "Origin")

		preflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
		if _, ok := index[origin]; !ok {
			if preflight {
				http.Error(w, "origin not allowed by "+corsOriginsEnv, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		if preflight {
			w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
			w.Header().Set("Access-Control-Max-Age", corsMaxAgeSeconds)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
