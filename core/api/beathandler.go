package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

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
	// Description carries a location's Tier-2 scene description (empty for actors/artifacts). The
	// narrate PLACE line renders it so the room's fixed character is DATA, not the narrator's invention.
	Description string `json:"description,omitempty"`
}

// decomposeSystemHeader is the decompose seat's standing instruction and stable cache prefix. It turns
// the player's words into a CHAIN OF ATTEMPTS (what the player TRIES — never outcomes; the referee
// rules outcomes), binds ids ONLY from the CANDIDATES block, emits UNRESOLVED on a genuine reference
// tie rather than guessing, and adds nothing the player did not state (FINAL-decompose). Foundations
// plan Task 9: "Decompose prompt = perception lines + candidates (ids) + the v2 schema" — the driver
// dropping req.Payload made this header + scene + candidates never reach the live model, so a real id
// could not be bound; assembling the prompt HERE, at the seat boundary, is the fix.
//
// Text lives in prompts/decompose.txt (core/api/prompts/README.md) — every fixed prompt rulebook
// readable in one place, config-style, mirroring the schema/*.json + go:embed pattern.
//
//go:embed prompts/decompose.txt
var decomposeSystemHeader string

// buildDecomposePrompt assembles the decompose prompt at the SEAT BOUNDARY — the perception payload's
// lines and candidate whitelist become the model's world HERE, since the driver drops req.Payload
// (D-13 keeps provider shaping in the driver; seat semantics stay at the call site). Layout is
// cache-native (mirrors cognitionprompt.go): the stable header + SCENE + CANDIDATES prefix caches, and
// the player's raw input rides the MUTABLE TAIL (last), so re-decomposing new input reuses the cached
// prefix. The v2 Schema (the closed-vocabulary leash) is passed unchanged on the GenRequest.
func buildDecomposePrompt(payload PerceptionPayload, playerText string) string {
	var sb strings.Builder
	sb.WriteString(decomposeSystemHeader)

	// SCENE — the player's perception lines (what they can currently perceive), oldest first.
	sb.WriteString("\n\nSCENE (what you perceive):\n")
	for _, l := range payload.Lines {
		sb.WriteString("- ")
		sb.WriteString(l)
		sb.WriteString("\n")
	}

	// CANDIDATES — the ONLY ids a bound attempt may reference. One per line: id  name  (kind).
	sb.WriteString("\nCANDIDATES (bind ids ONLY from this list):\n")
	for _, c := range payload.Candidates {
		sb.WriteString(c.ID)
		sb.WriteString("  ")
		sb.WriteString(c.Name)
		sb.WriteString("  (")
		sb.WriteString(c.Kind)
		sb.WriteString(")\n")
	}

	// PLAYER INPUT — the mutable tail: the raw words to decompose, LAST so the whole prefix stays cacheable.
	sb.WriteString("\nPLAYER INPUT:\n")
	sb.WriteString(playerText)
	return sb.String()
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

	// §7 injection bound: the player's raw text can ride into the combined-ruling prompt
	// (RULINGS-2026-07-24 §7), so an unbounded body is an unbounded prompt. Cap it at 64KB before
	// decoding; MaxBytesReader makes an over-cap read fail and the decode error below returns 400.
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
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
		GenRequest{Payload: pre, Prompt: buildDecomposePrompt(pre, in.Text), Schema: json.RawMessage(beatChainV2SchemaJSON)})
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
		DB:                h.pool,
		Resolve:           h.bridge.Driver(SeatResolve.Name),
		CognitionBatch:    h.bridge.Driver(SeatCognitionBatch.Name),
		CognitionIsolated: h.bridge.Driver(SeatCognitionIsolated.Name),
		WorldActor:        h.bridge.Driver(SeatWorldActor.Name),
	}

	var startTick int64
	if err := h.pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+1`,
		worldID).Scan(&startTick); err != nil {
		http.Error(w, "start tick", http.StatusInternalServerError)
		return
	}

	// Held check BEFORE decompose routing (RULINGS-2026-07-24 §3): read the world's pending holds
	// fresh from the table — no server memory, no session machine, the world carries the state. Any
	// pending hold ⇒ this input is a REACTION (§2 — first action + all held acts → one combined
	// ruling; remainder a normal chain). None ⇒ a normal beat.
	held, err := pendingHeldOutcomes(ctx, h.pool, worldID)
	if err != nil {
		log.Printf("pendingHeldOutcomes error: %v", err)
		http.Error(w, "held lookup failed", http.StatusInternalServerError)
		return
	}

	var outcome BeatOutcome
	if len(held) > 0 {
		outcome, err = orc.RunReactionBeat(ctx, worldID, viewerID, chain, held, startTick, in.Text)
	} else {
		outcome, err = orc.RunBeat(ctx, worldID, viewerID, chain, startTick)
	}
	if err != nil {
		log.Printf("beat error: %v", err)
		http.Error(w, "beat failed", http.StatusInternalServerError)
		return
	}

	// 4. perception payload AFTER — the narrate seat is perception-bound (ADR-020, no omniscient pass).
	post, err := h.payload(ctx, worldID, viewerID)
	if err != nil {
		http.Error(w, "payload", http.StatusInternalServerError)
		return
	}
	// Delta-first narration (Defect B): the perceptions the viewer already held BEFORE the beat are the
	// baseline; anything in `post` not among them is WHAT JUST HAPPENED, the rest is RECENT BACKGROUND.
	preIDs := make(map[string]bool, len(pre.LineIDs))
	for _, id := range pre.LineIDs {
		preIDs[id] = true
	}
	narration, err := h.bridge.Driver(SeatNarrate.Name).Generate(ctx,
		GenRequest{Payload: post, Prompt: buildNarratePrompt(post, viewerID, preIDs)})
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

// v1 recency dials (RULINGS-2026-07-23 §10 — "nothing is ever unreachable; not everything is always
// present"): the beat payload shows a BOUNDED RECENT WINDOW of the holder's perceptions, not their
// entire remembered life. Live bug this fixes: the narrator recited a viewer's whole 12-stop travel
// log every beat, because payload fed the full fn_visible_perceptions history. This is the MINIMAL
// stand-in — Station I's retrieval owns the real machinery (relevance, salience, fidelity). Keep
// fn_visible_perceptions untouched (other consumers/tests depend on it); window at the payload here.
const (
	recencyTickWindow = 50 // keep rows within this many ticks of the holder's newest visible row
	recencyMaxRows    = 20 // then cap to at most this many, newest-first, presented oldest-first
)

// payload builds the perception-bound payload from the WALL (fn_visible_perceptions). No raw canon.
// Also populates Candidates: present actors + current location.
func (h *beatHandler) payload(ctx context.Context, worldID, viewerID string) (PerceptionPayload, error) {
	// Recent window, oldest-first: keep rows within recencyTickWindow ticks of the newest visible row,
	// take the newest recencyMaxRows of those (DESC LIMIT), then reverse to oldest-first for the seats.
	rows, err := h.pool.Query(ctx,
		`SELECT content, perception_id::text FROM (
		   SELECT content, perception_id, acquired_tick
		   FROM fn_visible_perceptions($1,$2)
		   WHERE acquired_tick >= (SELECT max(acquired_tick) FROM fn_visible_perceptions($1,$2)) - $3::bigint
		   ORDER BY acquired_tick DESC
		   LIMIT $4
		 ) recent
		 ORDER BY acquired_tick ASC`, worldID, viewerID, recencyTickWindow, recencyMaxRows)
	if err != nil {
		return PerceptionPayload{}, err
	}
	defer rows.Close()
	var p PerceptionPayload
	for rows.Next() {
		var c, pid string
		if err := rows.Scan(&c, &pid); err != nil {
			return PerceptionPayload{}, err
		}
		p.Lines = append(p.Lines, c)
		p.LineIDs = append(p.LineIDs, pid) // parallel to Lines — the delta baseline for narration
	}
	if err := rows.Err(); err != nil {
		return PerceptionPayload{}, err
	}

	// Build candidate whitelist: present actors + current location.
	var loc string
	if err := h.pool.QueryRow(ctx,
		`SELECT (attrs->>'location_id')::text FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		worldID, viewerID).Scan(&loc); err != nil {
		return PerceptionPayload{}, err
	}

	if loc != "" {
		// Present actors at the same location.
		actorRows, err := h.pool.Query(ctx,
			`SELECT er.entity_id, er.canonical_name, er.entity_kind
			 FROM fn_actors_at($1, $2::uuid) fa
			 JOIN entity_registry er ON er.entity_id=fa.entity_id AND er.world_id=$1`,
			worldID, loc)
		if err != nil {
			return PerceptionPayload{}, err
		}
		if actorRows != nil {
			for actorRows.Next() {
				var id, name, kind string
				if err := actorRows.Scan(&id, &name, &kind); err == nil {
					p.Candidates = append(p.Candidates, Candidate{ID: id, Name: name, Kind: kind})
				}
			}
			if err := actorRows.Err(); err != nil {
				actorRows.Close()
				return PerceptionPayload{}, err
			}
			actorRows.Close()
		}

		// Current location entity — plus its Tier-2 scene description (empty when unseeded), so the
		// narrate PLACE line renders the room's fixed character as DATA (Defect B).
		var locName, locDesc string
		if err := h.pool.QueryRow(ctx,
			`SELECT canonical_name,
			        COALESCE((SELECT attrs->>'description' FROM location_state WHERE entity_id=$1::uuid AND world_id=$2), '')
			 FROM entity_registry WHERE entity_id=$1::uuid AND world_id=$2`,
			loc, worldID).Scan(&locName, &locDesc); err != nil {
			return PerceptionPayload{}, err
		}
		p.Candidates = append(p.Candidates, Candidate{ID: loc, Name: locName, Kind: "location", Description: locDesc})

		// Artifacts are deliberately ABSENT from the whitelist: naming reach = the
		// actor's own perceived/known set (RULINGS-2026-07-23 §3), and no
		// perception-subject link machinery exists yet to compute "artifacts this
		// actor knows". Until Station E lands that lookup, artifacts are not
		// nameable-by-id — fail closed beats leaking world contents.
	}

	return p, nil
}
