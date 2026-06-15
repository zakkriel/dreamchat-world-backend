package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewActorPageHandler returns the read-only Actor-page handler, delegating to the generic page
// handler (route GET /worlds/{w}/compendium/actors/{id}/page → fn_actor_page). The filter and the
// existence 404 live in SQL (ADR-P017 / §5.1).
func NewActorPageHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return NewPageHandler(pool, debug, "actors", "fn_actor_page")
}
