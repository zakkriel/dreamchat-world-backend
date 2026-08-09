package main

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// indexHandler serves GET /worlds/{w}/compendium/{urlKind} → compendium_index/1 (flat per-kind list).
// The existence filter is fn_compendium_index in SQL (ADR-P017); an unperceived entity is simply
// absent from entries (never redacted/placeholdered).
type indexHandler struct {
	pool       *pgxpool.Pool
	dbg        bool
	entityKind string // entity_registry.entity_kind, e.g. "artifact"
	re         *regexp.Regexp
}

// NewIndexHandler builds an index handler for a URL segment ("artifacts") mapped to an entity_kind
// ("artifact").
func NewIndexHandler(pool *pgxpool.Pool, debug bool, urlKind, entityKind string) http.Handler {
	return &indexHandler{
		pool: pool, dbg: debug, entityKind: entityKind,
		re: regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/compendium/` + urlKind + `$`),
	}
}

func (h *indexHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.re.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if writeNoViewer(w, err) {
		return
	}

	var payload []byte
	err = h.pool.QueryRow(ctx,
		`SELECT fn_compendium_index_json($1, $2, $3)::text`, worldID, viewerID, h.entityKind).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
