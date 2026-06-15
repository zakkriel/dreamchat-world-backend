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
	rt := &router{handlers: []matcher{
		NewPageHandler(pool, debug, "actors", "fn_actor_page").(matcher),
		NewPageHandler(pool, debug, "locations", "fn_location_page").(matcher),
		NewPageHandler(pool, debug, "artifacts", "fn_artifact_page").(matcher),
		NewIndexHandler(pool, debug, "actors", "actor").(matcher),
		NewIndexHandler(pool, debug, "locations", "location").(matcher),
		NewIndexHandler(pool, debug, "artifacts", "artifact").(matcher),
		NewTimelineHandler(pool, debug).(matcher),
	}}

	mux := http.NewServeMux()
	mux.Handle("/worlds/", rt)

	addr := ":8080"
	log.Printf("dreamchat world backend (read-only compendium API) on %s (debug=%v)", addr, debug)
	log.Fatal(http.ListenAndServe(addr, mux))
}
