package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sceneCurrentSchemaVersion stamps every scene/current response (schema/scene_current.v1.schema.json,
// core/api/schema/ — the frontend repo generates its types from that directory).
const sceneCurrentSchemaVersion = "scene_current/1"

var sceneCurrentRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/scene/current$`)

// scenePlace is the viewer's own naming of where they stand — never the canonical registry name
// they may not know (§3 naming reach). Description/Tone are nil when the place carries none.
type scenePlace struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Description *string `json:"description"`
	Tone        *string `json:"tone"` // the place's authored tension where one exists — a word, not a number
}

// sceneParticipant is one present CHARACTER, labeled with the viewer's own name for them (UX
// doctrine §2.2: "A guard holding a warrant can appear as a participant. The warrant itself should
// not." — never an object, a location, or a faction). The viewer is never listed as their own
// participant.
type sceneParticipant struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"` // always "actor" — the closed vocabulary this endpoint ships (kind exists for the schema's own sake, not because a second value exists yet)
}

// sceneNow is the moment, expressed as tick (ordering) + display_label (rendering) — wall-clock
// never crosses the boundary into UI (B-5). DisplayLabel is nil when the viewer holds no perception
// carrying one yet (e.g. a fresh world).
type sceneNow struct {
	Tick         int64   `json:"tick"`
	DisplayLabel *string `json:"display_label"`
}

// sceneView is the scene_current/1 projection: perception-bound, schema_version-stamped, no canon
// row crosses (B-1, I-3, D-7). Journey is the rung3 Task 2 block (journey.go's journeyBlock), or nil
// when the viewer holds no active journey — never an empty/placeholder value for "not travelling".
type sceneView struct {
	SchemaVersion string             `json:"schema_version"`
	Place         scenePlace         `json:"place"`
	Participants  []sceneParticipant `json:"participants"`
	Now           sceneNow           `json:"now"`
	Journey       *journeyBlock      `json:"journey"`
	Current       []string           `json:"current"` // "what matters now" — payload.Lines verbatim; prose the FE renders as-is (D-7), never structured state to interpret
}

// sceneHandler serves GET /worlds/{w}/scene/current — the first read side of the BE⇄FE contract
// (design §4.8, mvp_slice_and_bridge.md §4.1). It is a THIN wrapper around buildScene, mirroring
// pageHandler's shape (ResolveViewer → build → marshal).
type sceneHandler struct {
	pool *pgxpool.Pool
	dbg  bool
}

// NewSceneHandler injects the pool; debug enables the creator/debug ?viewer= override (run through
// the same ResolveViewer gate every handler uses).
func NewSceneHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &sceneHandler{pool: pool, dbg: debug}
}

func (h *sceneHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && sceneCurrentRoute.MatchString(r.URL.Path)
}

func (h *sceneHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := sceneCurrentRoute.FindStringSubmatch(r.URL.Path)
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

	scene, err := buildScene(ctx, h.pool, worldID, viewerID, h.dbg)
	if err != nil {
		log.Printf("scene/current: %v", err)
		http.Error(w, "scene failed", http.StatusInternalServerError)
		return
	}

	resp, _ := json.Marshal(scene)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// buildScene assembles the scene projection FROM THE PERCEPTION PAYLOAD THE ENGINE ALREADY BUILDS
// AND THROWS AWAY (beathandler.go's `payload` method, `:386`, and `narrateRoster`, `:303`) — no new
// omniscient query. dbg is threaded through for parity with the debug-gated pattern every other
// handler follows (ResolveViewer's override); scene_current carries no debug-only field today.
//
// A minimal *beatHandler{pool: pool} is constructed here to call its `payload` method — the only
// existing entry point to the perception-bound Candidates/Lines assembly (beathandler.go is Task 0's
// file, out of bounds this task; the method needs no other field). Task 3 reuses this exact function
// for the scene frame in the beat stream (design §4.8's SSE `scene` frame).
func buildScene(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string, dbg bool) (sceneView, error) {
	bh := &beatHandler{pool: pool}
	payload, err := bh.payload(ctx, worldID, viewerID)
	if err != nil {
		return sceneView{}, fmt.Errorf("buildScene: payload: %w", err)
	}

	// present = the ghost-speaker roster (narrateRoster excludes the location candidate and the
	// viewer themself); labelFor = each present id's VIEWER-facing name (fn_display_name, already
	// applied). narrateRoster's roster is NOT characters-only by itself — payload.Candidates also
	// widens to perceived scene artifacts (RULINGS-2026-07-30 §1), which narrateRoster does not
	// filter out (only "location" and the viewer are excluded there). UX doctrine §2.2 is a hard
	// requirement for THIS endpoint ("never objects, locations, or factions as participants"), so
	// participants below is filtered to Candidates whose Kind is "actor" — narrateRoster's roster,
	// narrowed to characters, exactly as the design intends even though narrateRoster's own filter
	// does not (yet) enforce it.
	present, labelFor := narrateRoster(payload, viewerID)

	var place *Candidate
	kindByID := make(map[string]string, len(payload.Candidates))
	for i := range payload.Candidates {
		c := &payload.Candidates[i]
		kindByID[c.ID] = c.Kind
		if c.Kind == "location" {
			place = c
		}
	}
	if place == nil {
		return sceneView{}, fmt.Errorf("buildScene: viewer %s has no resolvable place (no location_id set)", viewerID)
	}

	participants := make([]sceneParticipant, 0, len(present))
	for _, id := range present {
		if kindByID[id] != "actor" {
			continue // characters only (UX doctrine §2.2) — drops perceived artifacts narrateRoster's own filter lets through
		}
		participants = append(participants, sceneParticipant{ID: id, Label: labelFor[id], Kind: "actor"})
	}

	var tick int64
	if err := pool.QueryRow(ctx, `SELECT fn_world_now($1::uuid)`, worldID).Scan(&tick); err != nil {
		return sceneView{}, fmt.Errorf("buildScene: fn_world_now: %w", err)
	}

	// display_label rides the viewer's own NEWEST perceived line (payload.LineIDs, oldest-first —
	// the last entry is the newest), never an unscoped canon_event scan: the label is metadata of a
	// perception already established as visible to this viewer (fn_visible_perceptions, inside
	// bh.payload above), so joining canon_event for its in_world_label leaks nothing new.
	var displayLabel string
	if n := len(payload.LineIDs); n > 0 {
		if err := pool.QueryRow(ctx,
			`SELECT COALESCE((SELECT ce.in_world_label FROM perception_record pr
			                    JOIN canon_event ce ON ce.event_id = pr.source_event_id
			                   WHERE pr.perception_id = $1::uuid), '')`,
			payload.LineIDs[n-1]).Scan(&displayLabel); err != nil {
			return sceneView{}, fmt.Errorf("buildScene: display_label: %w", err)
		}
	}

	// tone: the place's authored TENSION attribute, surfaced to the FE under the genre-agnostic name
	// "tone". The stored key is `tension` (the same attribute tensionBudgetSeconds reads and
	// trg_validate_tension guards, seeded per-location e.g. the Drowned Lantern's 'tense') — reading
	// `tone` here returned NULL for every scene, which a hand-driven curl caught and the shape-only
	// unit test did not. Not carried on Candidate (only Description is), so one small scoped lookup
	// against the viewer's OWN current location — already established perceivable above, not a new
	// omniscient reach.
	var tone string
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT attrs->>'tension' FROM location_state WHERE world_id=$1 AND entity_id=$2::uuid), '')`,
		worldID, place.ID).Scan(&tone); err != nil {
		return sceneView{}, fmt.Errorf("buildScene: tone: %w", err)
	}

	current := payload.Lines
	if current == nil {
		current = []string{}
	}

	journey, err := (&Orchestrator{DB: pool}).journeyBlock(ctx, worldID, viewerID)
	if err != nil {
		return sceneView{}, fmt.Errorf("buildScene: journeyBlock: %w", err)
	}

	return sceneView{
		SchemaVersion: sceneCurrentSchemaVersion,
		Place: scenePlace{
			ID:          place.ID,
			Label:       place.Name,
			Description: nullIfEmpty(place.Description),
			Tone:        nullIfEmpty(tone),
		},
		Participants: participants,
		Now: sceneNow{
			Tick:         tick,
			DisplayLabel: nullIfEmpty(displayLabel),
		},
		Journey: journey,
		Current: current,
	}, nil
}

// nullIfEmpty renders the house "" == absent convention (Journey struct, journey.go:15-16) as a real
// JSON null instead of an empty string, for the fields the scene_current schema types string|null.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
