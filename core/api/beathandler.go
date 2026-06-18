package main

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// beatHandler serves POST /worlds/{w}/beat — the thin-slice play loop. It orchestrates the two
// model seats around the deterministic SQL engine: decompose (perception-bound input, §14) →
// DecodeAndValidateChain (the closed-vocabulary leash, SPEC-015/D-1) → apply_beat (the ONLY
// canonization point, origin='freeform') → narrate (perception-bound, ADR-020). No canon row ever
// crosses the boundary (B-1): the response is narration + a committed-event summary, never canon.
type beatHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	dec  Decomposer
	nar  Narrator
}

var beatRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beat$`)

// NewBeatHandler returns the handler. dec/nar are injected so CI uses deterministic fakes; the live
// model is wired separately and kept OUT of CI (operator gate only).
func NewBeatHandler(pool *pgxpool.Pool, debug bool, dec Decomposer, nar Narrator) http.Handler {
	return &beatHandler{pool: pool, dbg: debug, dec: dec, nar: nar}
}

// Match reports whether this handler owns the request (used by the router in main.go).
func (h *beatHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost && beatRoute.MatchString(r.URL.Path)
}

func (h *beatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := beatRoute.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := context.Background()

	// the viewer is the epistemic boundary, resolved SERVER-SIDE (D-7/B-1).
	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if err != nil {
		http.Error(w, "viewer resolution failed", http.StatusInternalServerError)
		return
	}

	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}

	// 1. perception payload BEFORE resolving — the decomposer is perception-bound (§14 extension).
	pre, err := h.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}
	// 2. decompose (PROPOSES ONLY, D-1) → validate against the closed vocabulary (the leash, SPEC-015).
	chain, err := DecodeAndValidateChain(h.dec.Decompose(ctx, pre, in.Text))
	if err != nil {
		http.Error(w, "outside the closed vocabulary", http.StatusUnprocessableEntity) // 422
		return
	}
	raw, _ := json.Marshal(chain)

	// 3. apply_beat is the ONLY canonization point (D-1). origin='freeform' = model-proposed, gated.
	//    start_tick = max accepted tick for the world + 1; cap = the generous backstop (§9).
	var summary []byte
	err = h.pool.QueryRow(ctx,
		`SELECT apply_beat($1,$2,$3::jsonb,
		          COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,
		          $4, 'freeform')::text`,
		worldID, viewerID, string(raw), beatTickCap).Scan(&summary)
	if err != nil {
		http.Error(w, "apply", http.StatusInternalServerError)
		return
	}

	// 4. perception payload AFTER the beat — the narrator is perception-bound (ADR-020, no omniscient).
	post, err := h.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}
	narration := h.nar.Narrate(ctx, post) // presentation only; never written to canon (I-6)

	w.Header().Set("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]any{
		"schema_version": "beat_result/1",
		"narration":      narration,
		"result":         json.RawMessage(summary), // {committed:[...ids], halt_reason, ticks_advanced}
	})
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// payload builds the perception-bound payload from the WALL (fn_visible_perceptions). No raw canon.
func (h *beatHandler) payload(ctx context.Context, worldID, viewerID string) (PerceptionPayload, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT content FROM fn_visible_perceptions($1,$2) ORDER BY acquired_tick`, worldID, viewerID)
	if err != nil {
		return PerceptionPayload{}, err
	}
	defer rows.Close()
	var p PerceptionPayload
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return PerceptionPayload{}, err
		}
		p.Lines = append(p.Lines, c)
	}
	return p, rows.Err()
}

// beatTickCap is the generous hard time-cap backstop (§9; ADR-025 provisional — tune at the gate).
const beatTickCap = 1000
