package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SPEC-028 — world management. `GET /worlds` answers "which worlds are there", which nothing could
// answer before: world_id was a bare uuid on twenty-seven tables and the frontend could make it flow
// through routing but never let anyone CHOOSE. `POST /worlds` creates one, per the founder's ruling
// that creation is a real endpoint rather than a seed-only affordance.
//
// This is the one route in the service that is NOT under /worlds/{id}, because it is the route you
// call when you do not have an id yet.
var worldsRoute = regexp.MustCompile(`^/worlds/?$`)

// worldTheme mirrors world_theme/1 (SPEC-019): one accent colour, a mood word, an ornament motif.
//
// One colour, not a palette — the frontend derives the rest, because a backend shipping ten colours
// would own visual design it has no business owning. `mood` is an ATMOSPHERE word, never a genre: the
// system never learns the word "fantasy" (GA-3).
//
// The draft vocabulary is mood ∈ {daylight, nocturne, mist, ember, bleak} and ornament ∈ {none,
// filigree, rivet, vine, circuit}, and it is deliberately NOT enforced here or in the CHECK
// constraint. Unknown values must DEGRADE, never fail: the frontend's skin already falls back
// safely, so a world authored against a newer vocabulary should still render. Validating the enum
// backend-side would make this service the thing that breaks on a word it has not heard of, which is
// exactly backwards for a field whose whole purpose is to evolve.
type worldTheme struct {
	SchemaVersion string `json:"schema_version"`
	Accent        string `json:"accent"`
	Mood          string `json:"mood"`
	Ornament      string `json:"ornament"`
}

// createWorldRequest is the POST body. Theme is optional — a world with no stated look gets the
// registry's default rather than an error, because "I have not chosen colours yet" should not block
// creating a world.
type createWorldRequest struct {
	DisplayName string      `json:"display_name"`
	Theme       *worldTheme `json:"theme,omitempty"`
}

type worldsHandler struct {
	pool *pgxpool.Pool
	dbg  bool
}

// NewWorldsHandler builds the /worlds collection handler.
func NewWorldsHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &worldsHandler{pool: pool, dbg: debug}
}

func (h *worldsHandler) Match(r *http.Request) bool {
	return worldsRoute.MatchString(r.URL.Path) &&
		(r.Method == http.MethodGet || r.Method == http.MethodPost)
}

func (h *worldsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && worldsRoute.MatchString(r.URL.Path):
		h.list(w, r)
	case r.Method == http.MethodPost && worldsRoute.MatchString(r.URL.Path):
		h.create(w, r)
	default:
		http.NotFound(w, r)
	}
}

// list serves GET /worlds from fn_world_directory — directory fields only, so there is no world
// state on this surface for the API layer to have to strip (SPEC-028: "a world list is a directory,
// not canon").
func (h *worldsHandler) list(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	if err := h.pool.QueryRow(r.Context(), `SELECT fn_world_directory()::text`).Scan(&payload); err != nil {
		log.Printf("worlds: directory: %v", err)
		http.Error(w, "world directory unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

// create serves POST /worlds.
//
// What it does and does not do. It writes a DIRECTORY ENTRY and gives the new world the per-world
// defaults every world needs to function (seed_world_defaults — movement speeds, duration classes,
// pressure config, extent classes, journey bands, watch horizon). It creates NO entities: no player,
// no rooms, nothing to perceive. So a freshly created world is real, listed, and not yet playable —
// `playable:false` in the directory, and a beat against it answers 404 rather than pretending.
//
// That is the honest shape for v1, and the alternative was worse. "Create a world with a starter
// scene" means shipping a world TEMPLATE, which is authored fiction — somebody's tavern, somebody's
// player character — and this service has no business inventing either (GA-2/GA-3: it must not learn
// what a world is "usually" like). Templates are a real feature and belong to whoever authors the
// fiction; the seam for them is here, unbuilt on purpose.
//
// NOT AUTHENTICATED, and this is the deployment risk to close first. No session model exists (B1),
// so there is no caller identity to check and anyone who can reach the service can create a world.
// That is stated rather than papered over with an invented token scheme, which would be drift. It is
// safe behind a private deployment and is NOT safe on a public origin: this endpoint should be the
// first thing auth is put in front of when B1 lands.
func (h *worldsHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var in createWorldRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(in.DisplayName)
	if name == "" {
		http.Error(w, "display_name is required", http.StatusBadRequest)
		return
	}

	theme := in.Theme
	if theme == nil {
		theme = &worldTheme{SchemaVersion: "world_theme/1", Accent: "#c9a227", Mood: "nocturne", Ornament: "filigree"}
	}
	if theme.SchemaVersion == "" {
		theme.SchemaVersion = "world_theme/1"
	}
	// The only theme rules enforced here are the ones the storage CHECK also enforces, so a caller
	// gets a 400 with a reason instead of a 500 from a constraint. Mood and ornament are NOT checked
	// against the draft vocabulary — see worldTheme.
	if theme.SchemaVersion != "world_theme/1" {
		http.Error(w, "theme.schema_version must be world_theme/1", http.StatusBadRequest)
		return
	}
	if !hexColour.MatchString(theme.Accent) {
		http.Error(w, "theme.accent must be a hex colour like #c9a227", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(theme.Mood) == "" || strings.TrimSpace(theme.Ornament) == "" {
		http.Error(w, "theme.mood and theme.ornament are required", http.StatusBadRequest)
		return
	}

	themeJSON, err := json.Marshal(theme)
	if err != nil {
		http.Error(w, "bad theme", http.StatusBadRequest)
		return
	}

	worldID, err := createWorld(r.Context(), h.pool, name, themeJSON)
	if err != nil {
		log.Printf("worlds: create: %v", err)
		http.Error(w, "world could not be created", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schema_version": "world_created/1",
		"id":             worldID,
		"display_name":   name,
		"theme":          theme,
		"playable":       false, // nothing has been authored into it yet
	})
}

var hexColour = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// createWorld writes the directory row and the world's operating defaults in ONE transaction: a
// world that exists but has no movement speeds or duration classes is a world every later call
// fails against, so the pair must land together or not at all.
func createWorld(ctx context.Context, pool *pgxpool.Pool, displayName string, themeJSON []byte) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	id, err := createWorldTx(ctx, tx, displayName, themeJSON)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
}

// createWorldTx is the write half of createWorld: insert the directory row plus seed_world_defaults.
// Callers supply the surrounding transaction when this work must be part of a larger atomic step.
func createWorldTx(ctx context.Context, tx pgx.Tx, displayName string, themeJSON []byte) (string, error) {
	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO world (world_id, display_name, theme)
		 VALUES (gen_random_uuid(), $1, $2::jsonb) RETURNING world_id::text`,
		displayName, string(themeJSON)).Scan(&id); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `SELECT seed_world_defaults($1::uuid)`, id); err != nil {
		return "", err
	}
	return id, nil
}
