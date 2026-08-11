package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// matcher is any compendium handler that can claim a request (all share the /worlds/ prefix).
type matcher interface {
	http.Handler
	Match(*http.Request) bool
}

// router dispatches to the first handler whose Match returns true; otherwise 404.
type router struct{ handlers []matcher }

func (rt *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, h := range rt.handlers {
		if h.Match(r) {
			h.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

// newRouter is the ONE list of served routes. It is a function rather than a literal inside main()
// so a test can drive the composed router instead of a handler in isolation: a handler that works
// perfectly and was never added to this slice is a 404 in production and a green suite, and that is
// exactly the failure mode "hand-drive everything" exists to catch.
func newRouter(pool *pgxpool.Pool, debug bool, bridge *Bridge, images *imageClient) *router {
	return &router{handlers: []matcher{
		NewPageHandler(pool, debug, "actors", "fn_actor_page").(matcher),
		NewPageHandler(pool, debug, "locations", "fn_location_page").(matcher),
		NewPageHandler(pool, debug, "artifacts", "fn_artifact_page").(matcher),
		NewIndexHandler(pool, debug, "actors", "actor").(matcher),
		NewIndexHandler(pool, debug, "locations", "location").(matcher),
		NewIndexHandler(pool, debug, "artifacts", "artifact").(matcher),
		NewTimelineHandler(pool, debug).(matcher),
		// GET /worlds/{w}/carrying — the Carrying overlay (mvp_slice_and_bridge §4.1). No carrier
		// segment: the carrier is the resolved viewer, which is what makes "Carrying for NPCs" (a
		// PRD non-goal) unreachable by construction rather than by a check.
		NewCarryingHandler(pool, debug).(matcher),
		// GET /worlds/{w}/scene/current — where you are, who is present, what matters now (design §4.8).
		NewSceneHandler(pool, debug).(matcher),
		// POST /worlds/{w}/beats and POST /worlds/{w}/beats/continue — the only write path; everything
		// it commits goes through apply_event (D-1). The singular POST /worlds/{w}/beat endpoint is
		// GONE (rung3 Task 5, founder-approved clean cutover — no alias, no deprecation shim; the only
		// caller was the founder's own throwaway test page). beatsStreamHandler serves both routes,
		// delivering the beat as a stream of validated frames (design §4.8, rung3 Task 3); continue
		// skips decompose entirely — an empty chain against an active journey IS the continue press.
		NewBeatsStreamHandler(pool, debug, bridge).(matcher),
		// SPEC-028: GET /worlds (the directory) and POST /worlds (creation). The one route not under
		// /worlds/{id}, because it is what you call when you do not have an id yet.
		NewWorldsHandler(pool, debug).(matcher),
		// Images (SPEC-033): GET /worlds/{w}/images/{asset_id} redirects to a freshly minted
		// presigned URL; POST /worlds/{w}/images/portraits is the explicit, bounded trigger.
		// A nil client is an ordinary state — the world runs, images simply stay absent.
		NewImageHandler(pool, images, debug).(matcher),
	}}
}

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	debug := os.Getenv("DREAMCHAT_MODE") == "debug"

	// Chunk-5 play loop: build the per-seat LLM bridge (D-13). Default = Claude via the Anthropic API
	// (structured tool-use as the decompose leash), swappable behind the interface; DREAMCHAT_BRIDGE=fake
	// uses deterministic drivers for keyless local dev. Bind fails closed if a seat's driver underpowers
	// its capability floor. CI never starts this server; tests inject their own bridge.
	bridge, err := NewBridge(defaultSeatConfig(), DefaultDriverFactory,
		SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor, SeatPlaceAuthor)
	if err != nil {
		log.Fatalf("bridge: %v", err)
	}

	rt := newRouter(pool, debug, bridge, newImageClientFromEnv())

	mux := http.NewServeMux()
	mux.Handle("/worlds/", rt)
	// "/worlds" without the trailing slash is a DIFFERENT ServeMux pattern: with only "/worlds/"
	// registered, the bare collection path 301s to the trailing-slash form, and a POST does not
	// survive that redirect intact. Both spellings route to the same handler.
	mux.Handle("/worlds", rt)

	// SPEC-021 — CORS. Wraps the mux (not the router) so preflights are answered before routing;
	// refuses to boot on a malformed allowlist rather than serving an API the frontend cannot reach.
	corsAllowed, corsBad := corsOrigins()
	if len(corsBad) > 0 {
		log.Fatalf("%s: %v is not a usable origin — list exact origins like http://localhost:5173 (no wildcard, no path)",
			corsOriginsEnv, corsBad)
	}
	handler := withCORS(mux, corsAllowed)

	// The port is assigned by the platform in a hosted deployment (Railway injects PORT) and fixed
	// at 8080 everywhere else — local dev, compose, stack.sh and every runbook say 8080, so that
	// stays the default rather than becoming a thing you have to remember to set.
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	if len(corsAllowed) == 0 {
		log.Printf("CORS: disabled (%s unset) — browsers on other origins cannot call this API", corsOriginsEnv)
	} else {
		log.Printf("CORS: allowing %d origin(s): %s", len(corsAllowed), strings.Join(corsAllowed, ", "))
	}
	log.Printf("dreamchat world backend (read-only compendium API) on %s (debug=%v)", addr, debug)
	log.Fatal(http.ListenAndServe(addr, handler))
}

// defaultSeatConfig routes each LLM seat (D-13). Production default = Claude via the Anthropic API
// (the decompose seat's structured tool-use is the leash); the live call needs ANTHROPIC_API_KEY at
// request time (bind succeeds without it — capability is reported, not key-gated). DREAMCHAT_BRIDGE=fake
// selects deterministic drivers for keyless local dev. Re-pointing one seat's entry changes only it.
//
// Resolve-seat override: if DREAMCHAT_RESOLVE_PROVIDER is non-empty, the resolve seat is wired to
// that provider instead of the global default. Use with DREAMCHAT_RESOLVE_BASE_URL,
// DREAMCHAT_RESOLVE_MODEL, and DREAMCHAT_RESOLVE_API_KEY (e.g. provider=openai-compat pointing at
// DeepInfra/DeepSeek/OpenRouter). All other seats are unaffected (D-13).
func defaultSeatConfig() SeatConfig {
	if os.Getenv("DREAMCHAT_BRIDGE") == "fake" {
		// Every seat gets the stand-in SHAPED LIKE ITS OWN OUTPUT. Three of these used to be
		// "fake-structured", which the factory builds with a nil scripted table: decompose could only
		// answer "[]", so the server streamed a correct frame sequence and committed nothing —
		// hand-driving three beats left canon_event at its seed rows — and resolve was never reached
		// to notice it had a chain-shaped stand-in bound. Keyless dev now binds ids from the real
		// candidate whitelist and commits, and the first adjudicated attempt reaches a ruling.
		return SeatConfig{
			"decompose":          {Provider: "fake-intent", Model: "dev"},
			"narrate":            {Provider: "fake-text", Model: "dev"},
			"resolve":            {Provider: "fake-resolve", Model: "dev"},
			"cognition_batch":    {Provider: "fake-cognition", Model: "dev"},
			"cognition_isolated": {Provider: "fake-cognition", Model: "dev"},
			"world_actor":        {Provider: "fake-world-actor", Model: "dev"},
			"place_author":       {Provider: "fake-place-author", Model: "dev"},
		}
	}
	model := os.Getenv("DREAMCHAT_MODEL")
	if model == "" {
		model = "claude-opus-4-8"
	}
	cfg := SeatConfig{
		"decompose":          {Provider: "anthropic", Model: model},
		"narrate":            {Provider: "anthropic", Model: model},
		"resolve":            {Provider: "anthropic", Model: model},
		"cognition_batch":    {Provider: "anthropic", Model: model},
		"cognition_isolated": {Provider: "anthropic", Model: model},
		"world_actor":        {Provider: "anthropic", Model: model},
		"place_author":       {Provider: "anthropic", Model: model},
	}

	// Per-seat resolve override: DREAMCHAT_RESOLVE_PROVIDER selects an alternate provider for
	// the resolve seat (e.g. "openai-compat" backed by DeepSeek). base_url and api_key are
	// carried via Params so DriverConfig stays additive and other call sites are untouched.
	if resolveProvider := os.Getenv("DREAMCHAT_RESOLVE_PROVIDER"); resolveProvider != "" {
		resolveModel := os.Getenv("DREAMCHAT_RESOLVE_MODEL")
		if resolveModel == "" {
			resolveModel = model // fall back to global model name as a hint
		}
		cfg["resolve"] = DriverConfig{
			Provider: resolveProvider,
			Model:    resolveModel,
			Params: map[string]string{
				"base_url": os.Getenv("DREAMCHAT_RESOLVE_BASE_URL"),
				"api_key":  os.Getenv("DREAMCHAT_RESOLVE_API_KEY"),
			},
		}
	}

	// Per-seat cognition override (mirror of the resolve block): DREAMCHAT_COGNITION_PROVIDER
	// re-points BOTH cognition seats — batch AND isolated — at one alternate provider (one env
	// family, since the two seats are the same NPC-decision workload split only by the wall). Other
	// seats are unaffected (D-13). base_url/api_key ride Params so DriverConfig stays additive.
	if cogProvider := os.Getenv("DREAMCHAT_COGNITION_PROVIDER"); cogProvider != "" {
		cogModel := os.Getenv("DREAMCHAT_COGNITION_MODEL")
		if cogModel == "" {
			cogModel = model // fall back to the global model name as a hint
		}
		cogCfg := DriverConfig{
			Provider: cogProvider,
			Model:    cogModel,
			Params: map[string]string{
				"base_url": os.Getenv("DREAMCHAT_COGNITION_BASE_URL"),
				"api_key":  os.Getenv("DREAMCHAT_COGNITION_API_KEY"),
			},
		}
		cfg["cognition_batch"] = cogCfg
		cfg["cognition_isolated"] = cogCfg
	}
	return cfg
}
