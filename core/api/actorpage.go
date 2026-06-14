package main

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// route: GET /worlds/{w}/compendium/actors/{id}/page
var routeRE = regexp.MustCompile(
	`^/worlds/([0-9a-fA-F-]{36})/compendium/actors/([0-9a-fA-F-]{36})/page$`)

type actorPageHandler struct {
	pool  *pgxpool.Pool
	debug bool
}

// NewActorPageHandler returns the read-only Actor-page handler. debug enables the creator/debug
// `?viewer=` override (C-4). The handler is a THIN READER: the entire perception/safety filter
// lives in fn_actor_page (SQL), never here (ADR-P017 binding).
func NewActorPageHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &actorPageHandler{pool: pool, debug: debug}
}

func (h *actorPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := routeRE.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID, actorID := m[1], m[2]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.debug)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var payload []byte
	err = h.pool.QueryRow(ctx,
		`SELECT fn_actor_page($1, $2, $3)::text`, worldID, viewerID, actorID).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
