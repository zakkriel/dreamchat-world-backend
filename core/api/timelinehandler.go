package main

import (
	"context"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// timelineHandler serves GET /worlds/{w}/compendium/timeline?before_tick=… → timeline/1.
// Relevance lens (holder = viewer) lives in fn_timeline (ADR-P017).
type timelineHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	re   *regexp.Regexp
}

func NewTimelineHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &timelineHandler{
		pool: pool, dbg: debug,
		re: regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/compendium/timeline$`),
	}
}

func (h *timelineHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *timelineHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.re.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := context.Background()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var beforeTick *int64 // NULL unless a valid before_tick is supplied
	if s := r.URL.Query().Get("before_tick"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			http.Error(w, "before_tick must be an integer", http.StatusBadRequest)
			return
		}
		beforeTick = &v
	}

	var payload []byte
	err = h.pool.QueryRow(ctx,
		`SELECT fn_timeline($1, $2, $3)::text`, worldID, viewerID, beforeTick).Scan(&payload)
	if err != nil {
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
