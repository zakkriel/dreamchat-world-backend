package main

import (
	"context"
	"log"
	"net/http"
	"os"

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
	bridge, err := NewBridge(defaultSeatConfig(), DefaultDriverFactory, SeatDecompose, SeatNarrate)
	if err != nil {
		log.Fatalf("bridge: %v", err)
	}

	rt := &router{handlers: []matcher{
		NewPageHandler(pool, debug, "actors", "fn_actor_page").(matcher),
		NewPageHandler(pool, debug, "locations", "fn_location_page").(matcher),
		NewPageHandler(pool, debug, "artifacts", "fn_artifact_page").(matcher),
		NewIndexHandler(pool, debug, "actors", "actor").(matcher),
		NewIndexHandler(pool, debug, "locations", "location").(matcher),
		NewIndexHandler(pool, debug, "artifacts", "artifact").(matcher),
		NewTimelineHandler(pool, debug).(matcher),
		// POST /worlds/{w}/beat — the only write path; everything it commits goes through apply_beat (D-1).
		NewBeatHandler(pool, debug, bridge).(matcher),
	}}

	mux := http.NewServeMux()
	mux.Handle("/worlds/", rt)

	addr := ":8080"
	log.Printf("dreamchat world backend (read-only compendium API) on %s (debug=%v)", addr, debug)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// defaultSeatConfig routes each LLM seat (D-13). Production default = Claude via the Anthropic API
// (the decompose seat's structured tool-use is the leash); the live call needs ANTHROPIC_API_KEY at
// request time (bind succeeds without it — capability is reported, not key-gated). DREAMCHAT_BRIDGE=fake
// selects deterministic drivers for keyless local dev. Re-pointing one seat's entry changes only it.
func defaultSeatConfig() SeatConfig {
	if os.Getenv("DREAMCHAT_BRIDGE") == "fake" {
		return SeatConfig{
			"decompose": {Provider: "fake-structured", Model: "dev"},
			"narrate":   {Provider: "fake-text", Model: "dev"},
		}
	}
	model := os.Getenv("DREAMCHAT_MODEL")
	if model == "" {
		model = "claude-opus-4-8"
	}
	return SeatConfig{
		"decompose": {Provider: "anthropic", Model: model},
		"narrate":   {Provider: "anthropic", Model: model},
	}
}
