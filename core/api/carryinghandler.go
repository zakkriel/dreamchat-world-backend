package main

// Governed-by: ADR-P017 — Go is the backend language.
// Promoted from this file's own citations (2026-08-26), not newly decided. Change what this
// file decides and those decisions change with it (D-9).

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// carryingHandler serves GET /worlds/{w}/carrying → carrying/1 (mvp_slice_and_bridge §4.1,
// "Carry States of the user-controlled Actor only").
//
// There is no carrier segment in the route and no carrier argument in fn_carrying: the carrier IS
// the resolved viewer. That is what makes the PRD's non-goal ("Carrying for NPCs") unreachable by
// construction rather than by a check someone can forget — no request can name a carrier at all.
//
// Like every other projection endpoint this is a THIN READER; the whole lens lives in the SQL
// (ADR-P017). Unlike the page endpoints it has no NULL→404 branch, because fn_carrying always
// returns an envelope: "you are carrying nothing" is an answer, not a missing page.
type carryingHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	re   *regexp.Regexp
}

func NewCarryingHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &carryingHandler{
		pool: pool, dbg: debug,
		re: regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/carrying$`),
	}
}

func (h *carryingHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *carryingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		`SELECT fn_carrying($1, $2)::text`, worldID, viewerID).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
