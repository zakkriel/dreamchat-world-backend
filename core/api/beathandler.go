package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// the closed-vocabulary schema, embedded so the decompose seat can constrain generation to it (the
// leash, generation-time). Same file the frontend codegens from.
//
//go:embed schema/beat_chain.v1.schema.json
var beatChainSchema []byte

var beatRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beat$`)

// beatHandler serves POST /worlds/{w}/beat. It orchestrates the per-seat bridge around the
// deterministic SQL engine: decompose (perception-bound input §14, STRUCTURED against beat_chain =
// the leash, SPEC-015/D-1) → DecodeAndValidateChain (defense-in-depth belt) → apply_beat (the ONLY
// canonization point, origin='freeform') → narrate (perception-bound, ADR-020). No canon row crosses
// the boundary (B-1): the response is narration + a committed-event summary.
type beatHandler struct {
	pool   *pgxpool.Pool
	dbg    bool
	bridge *Bridge
}

// NewBeatHandler injects the bridge so CI uses fakes and the operator gate/production uses the live
// per-seat drivers — both behind the same interface (D-13).
func NewBeatHandler(pool *pgxpool.Pool, debug bool, bridge *Bridge) http.Handler {
	return &beatHandler{pool: pool, dbg: debug, bridge: bridge}
}

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

	// 1. perception payload BEFORE — the decompose seat is perception-bound (§14).
	pre, err := h.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}
	// 2. decompose: STRUCTURED generation against the closed vocabulary (the leash). The seat's
	//    capability floor guarantees a constrained driver; DecodeAndValidateChain is the belt.
	raw, err := h.bridge.Driver(SeatDecompose.Name).Generate(ctx,
		GenRequest{Payload: pre, Prompt: in.Text, Schema: beatChainSchema})
	if err != nil {
		http.Error(w, "decompose failed", http.StatusBadGateway)
		return
	}
	chain, err := DecodeAndValidateChain(raw)
	if err != nil {
		http.Error(w, "outside the closed vocabulary", http.StatusUnprocessableEntity) // 422
		return
	}
	chainJSON, _ := json.Marshal(chain)

	// 3. apply_beat is the ONLY canonization point (D-1). origin='freeform' = model-proposed, gated.
	var summary []byte
	err = h.pool.QueryRow(ctx,
		`SELECT apply_beat($1,$2,$3::jsonb,
		          COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1,
		          $4, 'freeform')::text`,
		worldID, viewerID, string(chainJSON), beatTickCap).Scan(&summary)
	if err != nil {
		http.Error(w, "apply", http.StatusInternalServerError)
		return
	}

	// 4. perception payload AFTER — the narrate seat is perception-bound (ADR-020, no omniscient pass).
	post, err := h.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}
	narration, err := h.bridge.Driver(SeatNarrate.Name).Generate(ctx,
		GenRequest{Payload: post, Prompt: "Narrate the beat from the player's perceptions only."})
	if err != nil {
		http.Error(w, "narrate failed", http.StatusBadGateway)
		return
	}

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
