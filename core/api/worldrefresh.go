package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// worldRefreshRoute is POST /worlds/{w}/refresh — refresh means "supersede this world with a new
// canonical instance", never "erase and mutate in place".
var worldRefreshRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/refresh$`)

type worldRefreshHandler struct {
	pool *pgxpool.Pool
	dbg  bool
}

func NewWorldRefreshHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &worldRefreshHandler{pool: pool, dbg: debug}
}

func (h *worldRefreshHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost && worldRefreshRoute.MatchString(r.URL.Path)
}

func (h *worldRefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := worldRefreshRoute.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	sourceWorldID := m[1]
	ctx := r.Context()

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var displayName string
	var themeJSON []byte
	var templateKey *string
	if err := tx.QueryRow(ctx,
		`SELECT display_name, theme, template_key FROM world WHERE world_id=$1`, sourceWorldID,
	).Scan(&displayName, &themeJSON, &templateKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "world not found")
			return
		}
		log.Printf("world refresh: read source: %v", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	if templateKey == nil {
		writeJSONError(w, http.StatusNotFound, "world has no template")
		return
	}
	if *templateKey != "drowned_lantern" {
		writeJSONError(w, http.StatusNotImplemented, "unknown template")
		return
	}

	newWorldID, err := createWorldTx(ctx, tx, displayName, themeJSON)
	if err != nil {
		log.Printf("world refresh: create successor: %v", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec(ctx, `SELECT fn_instantiate_drowned_lantern($1::uuid, NULL)`, newWorldID); err != nil {
		log.Printf("world refresh: instantiate template: %v", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE world SET archived_at = now() WHERE world_id = $1`, sourceWorldID,
	); err != nil {
		log.Printf("world refresh: archive source: %v", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("world refresh: commit: %v", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}

	var theme worldTheme
	if err := json.Unmarshal(themeJSON, &theme); err != nil {
		log.Printf("world refresh: decode theme: %v", err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schema_version":  "world_refreshed/1",
		"source_world_id": sourceWorldID,
		"id":              newWorldID,
		"display_name":    displayName,
		"theme":           theme,
		"playable":        true,
	})
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
