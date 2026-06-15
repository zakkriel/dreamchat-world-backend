package main

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pageHandler serves GET /worlds/{w}/compendium/{kind}/{id}/page for any entity kind. It is a THIN
// READER: the entire perception/existence filter lives in the SQL function (ADR-P017). A NULL result
// means the entity is not in the viewer's existence set (§5.1) → 404, indistinguishable from a
// nonexistent id.
type pageHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	fn   string // SQL function name, e.g. "fn_artifact_page" (internal constant, never user input)
	re   *regexp.Regexp
}

// NewPageHandler builds a handler for the given URL kind segment ("actors"/"locations"/"artifacts")
// and SQL function. debug enables the creator/debug ?viewer= override (run through the same gate).
func NewPageHandler(pool *pgxpool.Pool, debug bool, kind, fn string) http.Handler {
	return &pageHandler{
		pool: pool, dbg: debug, fn: fn,
		re: regexp.MustCompile(
			`^/worlds/([0-9a-fA-F-]{36})/compendium/` + kind + `/([0-9a-fA-F-]{36})/page$`),
	}
}

// Match reports whether this handler owns the request (used by the router in main.go).
func (h *pageHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *pageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.re.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID, entityID := m[1], m[2]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var payload []byte
	// h.fn is an internal constant (never request-derived); ids are constrained by the route regex.
	err = h.pool.QueryRow(ctx,
		`SELECT `+h.fn+`($1, $2, $3)::text`, worldID, viewerID, entityID).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	if payload == nil { // SQL NULL → not in viewer's existence set (§5.1) → 404
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
