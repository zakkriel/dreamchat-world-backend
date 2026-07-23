package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// the v1 closed-vocabulary schema (legacy; kept so beatseats_test.go and bridge_test.go compile).
//
//go:embed schema/beat_chain.v1.schema.json
var beatChainSchema []byte

var beatRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beat$`)

// Candidate is a known entity that the player can reference by ID in a beat chain (v2).
type Candidate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// beatHandler serves POST /worlds/{w}/beat. It orchestrates the per-seat bridge around the
// deterministic SQL engine: decompose (perception-bound input §14, STRUCTURED against beat_chain =
// the leash, SPEC-015/D-1) → DecodeAndValidateChainV2 (defense-in-depth belt) → Orchestrator.RunBeat
// (the ONLY canonization point, origin='freeform') → narrate (perception-bound, ADR-020). No canon
// row crosses the boundary (B-1): the response is narration + a committed-event summary.
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

	// 2. decompose: STRUCTURED generation against the v2 closed vocabulary (the leash). The seat's
	//    capability floor guarantees a constrained driver; DecodeAndValidateChainV2 is the belt.
	raw, err := h.bridge.Driver(SeatDecompose.Name).Generate(ctx,
		GenRequest{Payload: pre, Prompt: in.Text, Schema: json.RawMessage(beatChainV2SchemaJSON)})
	if err != nil {
		log.Printf("decompose error: %v", err)
		http.Error(w, "decompose failed", http.StatusBadGateway)
		return
	}
	chain, err := DecodeAndValidateChainV2(raw)
	if err != nil {
		http.Error(w, "outside the closed vocabulary", http.StatusUnprocessableEntity) // 422
		return
	}

	// 3. Orchestrator.RunBeat is the ONLY canonization point (D-1). origin='freeform' = model-proposed, gated.
	orc := &Orchestrator{
		DB:             h.pool,
		Resolve:        h.bridge.Driver(SeatResolve.Name),
		CognitionBatch: h.bridge.Driver(SeatCognitionBatch.Name),
		WorldActor:     h.bridge.Driver(SeatWorldActor.Name),
	}

	var startTick int64
	if err := h.pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1`,
		worldID).Scan(&startTick); err != nil {
		http.Error(w, "start tick", http.StatusInternalServerError)
		return
	}

	outcome, err := orc.RunBeat(ctx, worldID, viewerID, chain, startTick)
	if err != nil {
		log.Printf("RunBeat error: %v", err)
		http.Error(w, "beat failed", http.StatusInternalServerError)
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
		"schema_version": "beat_result/2",
		"narration":      narration,
		"result": map[string]any{
			"committed":             outcome.Committed,
			"halt_reason":           outcome.HaltReason,
			"ticks_advanced":        outcome.TicksAdvanced,
			"unresolved_candidates": outcome.UnresolvedCandidates,
			"telegraphs":            outcome.Telegraphs,
		},
	})
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// payload builds the perception-bound payload from the WALL (fn_visible_perceptions). No raw canon.
// Also populates Candidates: present actors + current location + world artifacts.
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
	if err := rows.Err(); err != nil {
		return PerceptionPayload{}, err
	}

	// Build candidate whitelist: present actors + current location + artifacts at location.
	var loc string
	_ = h.pool.QueryRow(ctx,
		`SELECT (attrs->>'location_id')::text FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		worldID, viewerID).Scan(&loc)

	if loc != "" {
		// Present actors at the same location.
		actorRows, _ := h.pool.Query(ctx,
			`SELECT er.entity_id, er.canonical_name, er.entity_kind
			 FROM fn_actors_at($1, $2::uuid) fa
			 JOIN entity_registry er ON er.entity_id=fa.entity_id AND er.world_id=$1`,
			worldID, loc)
		if actorRows != nil {
			for actorRows.Next() {
				var id, name, kind string
				if err := actorRows.Scan(&id, &name, &kind); err == nil {
					p.Candidates = append(p.Candidates, Candidate{ID: id, Name: name, Kind: kind})
				}
			}
			actorRows.Close()
		}

		// Current location entity.
		var locName string
		_ = h.pool.QueryRow(ctx,
			`SELECT canonical_name FROM entity_registry WHERE entity_id=$1::uuid AND world_id=$2`,
			loc, worldID).Scan(&locName)
		p.Candidates = append(p.Candidates, Candidate{ID: loc, Name: locName, Kind: "location"})

		// Artifacts (no per-location link table — include all artifacts for this world as best effort).
		artRows, _ := h.pool.Query(ctx,
			`SELECT entity_id::text, canonical_name FROM entity_registry WHERE world_id=$1 AND entity_kind='artifact'`,
			worldID)
		if artRows != nil {
			for artRows.Next() {
				var id, name string
				if err := artRows.Scan(&id, &name); err == nil {
					p.Candidates = append(p.Candidates, Candidate{ID: id, Name: name, Kind: "artifact"})
				}
			}
			artRows.Close()
		}
	}

	return p, nil
}
