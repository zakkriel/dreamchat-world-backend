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
		// GET /worlds/{w}/transcript — the persistent transcript (transcript/1). Viewer-scoped, newest
		// first, cursor-paginated. The one read surface that is a RECORD rather than a projection: the
		// prose a model wrote once cannot be recomputed from world state.
		NewTranscriptHandler(pool, debug).(matcher),
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
		// World creation (PRD: prd_world_creation.md). POST /worlds/interview asks the next question about
		// a brief; POST /worlds/genesis authors a whole world and streams what lands. Neither sits under
		// /worlds/{id} because there is no id yet — that is the point of them.
		NewWorldGenesisHandler(pool, debug, bridge).(matcher),
		// POST /worlds/{w}/refresh mints a successor world and archives the source. The old world is
		// superseded, never reset in place: canon stays append-only, old ids stay citable, and refresh
		// cannot silently erase history someone already referenced.
		NewWorldRefreshHandler(pool, debug).(matcher),
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

	// The database must carry every migration this binary was built against, or the service refuses
	// to serve (schemaversion.go). Nothing in the deploy path applies migrations, so without this a
	// schema change ships as code and waits silently for the first request that touches it.
	applied, err := appliedMigrations(context.Background(), pool)
	if err != nil {
		log.Fatalf("schema check: %v", err)
	}
	if missing, extra := diffMigrations(applied); len(missing) > 0 {
		log.Fatalf("%v", schemaDriftError(missing))
	} else if len(extra) > 0 {
		log.Printf("schema: database is AHEAD of this binary by %d migration(s): %s (rollback?)",
			len(extra), strings.Join(extra, ", "))
	}

	debug := os.Getenv("DREAMCHAT_MODE") == "debug"

	// Raindrop/Workshop observability (raindrop.go): keyless no-op unless a local Workshop daemon or
	// RAINDROP_WRITE_KEY is present, so this line is inert in CI and in an unconfigured deployment.
	defer initRaindrop()()

	// Chunk-5 play loop: build the per-seat LLM bridge (D-13). Every seat's provider and model come
	// from the environment (seatconfig.go) — there is no default provider, by founder ruling, so a
	// misconfigured deployment fails at boot with the missing variable named rather than at the first
	// beat with somebody's bill. DREAMCHAT_BRIDGE=fake is the keyless local path. Bind fails closed if
	// a seat's driver underpowers its capability floor. CI never starts this server; tests inject
	// their own bridge.
	seatCfg, err := seatConfig(osLookup)
	if err != nil {
		log.Fatalf("seat config: %v", err)
	}
	bridge, err := NewBridge(seatCfg, DefaultDriverFactory,
		SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor, SeatPlaceAuthor,
		SeatWorldGenesis, SeatWorldInterview)
	if err != nil {
		log.Fatalf("bridge: %v", err)
	}
	log.Printf("seats: %s", describeSeatConfig(seatCfg))

	rt := newRouter(pool, debug, bridge, newImageClientFromEnv())

	mux := http.NewServeMux()
	mux.Handle("/worlds/", rt)
	// "/worlds" without the trailing slash is a DIFFERENT ServeMux pattern: with only "/worlds/"
	// registered, the bare collection path 301s to the trailing-slash form, and a POST does not
	// survive that redirect intact. Both spellings route to the same handler.
	mux.Handle("/worlds", rt)

	// SPEC-021 — CORS. Wraps the mux (not the router) so preflights are answered before routing;
	// refuses to boot on a malformed allowlist rather than serving an API the frontend cannot reach.
	// The auth gate (auth.go) sits INSIDE the CORS wrapper: preflights carry no credentials by
	// definition, so they must be answered ungated, while every real request is checked.
	corsAllowed, corsBad := corsOrigins()
	if len(corsBad) > 0 {
		log.Fatalf("%s: %v is not a usable origin — list exact origins like http://localhost:5173 (no wildcard, no path)",
			corsOriginsEnv, corsBad)
	}
	handler := withCORS(withAuth(mux), corsAllowed)

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

// seatConfig routes each LLM seat (D-13). DREAMCHAT_BRIDGE=fake keeps the deterministic keyless
// stand-ins; everything else is resolved from the environment by seatconfig.go, with no default
// provider anywhere in the code.
//
// The all-Anthropic map that used to live here, and the two hand-written per-seat override blocks
// beside it, are GONE. The founder's ruling on the record since 2026-08-07 is *"never default seats
// to Anthropic; per-seat overrides owed for all seats"*, and a map naming one vendor seven times is
// not a neutral contract however swappable it claims to be. Anthropic is now reachable exactly the
// way every other provider is: name it in DREAMCHAT_SEATS and give it an endpoint.
func seatConfig(lookup func(string) string) (SeatConfig, error) {
	if lookup("DREAMCHAT_BRIDGE") == "fake" {
		// Every seat gets the stand-in SHAPED LIKE ITS OWN OUTPUT. Three of these used to be
		// "fake-structured", which the factory builds with a nil scripted table: decompose could only
		// answer "[]", so the server streamed a correct frame sequence and committed nothing —
		// hand-driving three beats left canon_event at its seed rows — and resolve was never reached
		// to notice it had a chain-shaped stand-in bound. Keyless dev now binds ids from the real
		// candidate whitelist and commits, and the first adjudicated attempt reaches a ruling.
		//
		// narrate was the LAST seat left pointing at a stand-in not shaped like its own output:
		// fake-text reports no capabilities, so it errored on the narration/2 schema and every
		// hand-driven beat fell through to the belt-less plain prose fallback.
		return SeatConfig{
			"decompose":          {Provider: "fake-intent", Model: "dev"},
			"narrate":            {Provider: "fake-narrate", Model: "dev"},
			"resolve":            {Provider: "fake-resolve", Model: "dev"},
			"cognition_batch":    {Provider: "fake-cognition", Model: "dev"},
			"cognition_isolated": {Provider: "fake-cognition", Model: "dev"},
			"world_actor":        {Provider: "fake-world-actor", Model: "dev"},
			"place_author":       {Provider: "fake-place-author", Model: "dev"},
			"world_genesis":      {Provider: "fake-world-genesis", Model: "dev"},
			"world_interview":    {Provider: "fake-world-interview", Model: "dev"},
		}, nil
	}
	return seatConfigFromEnv(lookup)
}
