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

	// Founder envelope: the narrate seat now emits an ORDERED list of typed segments (narrator prose +
	// attributed NPC speech/actions), not one blob. presentIDs = the PRESENT roster the model sees (the
	// ghost-speaker belt); labelFor = the VIEWER's display name per present id (reused from the payload
	// candidates — fn_display_name results, never a canonical fallback beyond it). speechTexts = the
	// verbatim-speech belt's evidence (this beat's Communicated perception contents, by speaker).
	presentIDs, labelFor := narrateRoster(post, viewerID)
	speechTexts, err := h.speechTexts(ctx, worldID, viewerID, startTick)
	if err != nil {
		log.Printf("narrate: speechTexts lookup failed (belt runs with no evidence): %v", err)
		speechTexts = map[string][]string{} // fail toward repair, never crash the beat
	}

	// Structured narration with ONE repair (schema/belt failure → re-ask with the errors attached, the
	// resolve-seat pattern), then a PLAIN prose fallback (no schema) wrapped as a single narrator segment
	// — the beat is never failed on formatting.
	nd := h.bridge.Driver(SeatNarrate.Name)
	var segments []NarrationSegment
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		prompt := buildNarratePrompt(post, viewerID, preIDs, outcome.HaltReason)
		if attempt > 0 && lastErr != nil {
			prompt = buildNarrateRepairPrompt(post, viewerID, preIDs, outcome.HaltReason, lastErr.Error())
		}
		raw, genErr := nd.Generate(ctx, GenRequest{Payload: post, Prompt: prompt, Schema: json.RawMessage(narrationV1SchemaJSON)})
		if genErr != nil {
			log.Printf("narrate: structured Generate failed (attempt %d/2): %v", attempt+1, genErr)
			lastErr = genErr
			continue
		}
		segs, decErr := DecodeAndValidateNarration(raw, presentIDs, speechTexts)
		if decErr != nil {
			log.Printf("narrate: segment decode/validate failed (attempt %d/2): %v", attempt+1, decErr)
			lastErr = decErr
			continue
		}
		segments = segs
		break
	}
	if segments == nil {
		raw, genErr := nd.Generate(ctx, GenRequest{Payload: post, Prompt: buildNarratePlainPrompt(post, viewerID, preIDs, outcome.HaltReason)})
		if genErr != nil {
			log.Printf("narrate: plain fallback Generate failed: %v", genErr)
			http.Error(w, "narrate failed", http.StatusBadGateway)
			return
		}
		segments = []NarrationSegment{{Kind: "narration", Text: raw}}
	}

	messages, narration := narrateMessages(segments, labelFor)

	w.Header().Set("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]any{
		"schema_version": "beat_result/3",
		"narration":      narration, // legacy narrator-view fallback text — kept for old clients
		"messages":       messages,
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

// beatMessage is one element of the founder-envelope `messages` list: an attributed, typed segment. The
// speaker_label is the VIEWER's display name for the speaker (reused from the payload candidates —
// fn_display_name results — never a canonical fallback beyond what fn_display_name already does); it is
// empty for narrator segments (speaker_id null).
type beatMessage struct {
	SpeakerID    *string `json:"speaker_id"`
	SpeakerLabel string  `json:"speaker_label"`
	Kind         string  `json:"kind"`
	Text         string  `json:"text"`
}

// narrateRoster returns (presentIDs, labelFor): the present-actor ids the narrate prompt lists (the
// ghost-speaker belt's allow-set) and each id's VIEWER-facing display name (the candidate Name, already
// fn_display_name). The location candidate and the viewer himself are excluded — a segment is never
// attributed to the person narrated TO, and the place is not a speaker.
func narrateRoster(payload PerceptionPayload, viewerID string) ([]string, map[string]string) {
	present := make([]string, 0, len(payload.Candidates))
	labelFor := make(map[string]string, len(payload.Candidates))
	for _, c := range payload.Candidates {
		if c.Kind == "location" || c.ID == viewerID {
			continue
		}
		present = append(present, c.ID)
		labelFor[c.ID] = c.Name
	}
	return present, labelFor
}

// narrateMessages projects decoded segments into the response `messages` list (attaching each speaker's
// viewer-facing label) AND the legacy `narration` string — a joined narrator-view: narrator/action text
// as-is, speech quoted under its label. The legacy string stays populated so old clients keep working.
func narrateMessages(segments []NarrationSegment, labelFor map[string]string) ([]beatMessage, string) {
	messages := make([]beatMessage, 0, len(segments))
	view := make([]string, 0, len(segments))
	for _, s := range segments {
		m := beatMessage{SpeakerID: s.SpeakerID, Kind: s.Kind, Text: s.Text}
		if s.SpeakerID != nil {
			m.SpeakerLabel = labelFor[*s.SpeakerID]
		}
		messages = append(messages, m)
		if s.Kind == "speech" && m.SpeakerLabel != "" {
			view = append(view, m.SpeakerLabel+`: "`+s.Text+`"`)
		} else {
			view = append(view, s.Text)
		}
	}
	return messages, strings.Join(view, "\n\n")
}

// speechTexts is the verbatim-speech belt's evidence: every Communicated perception content the viewer
// holds THIS BEAT (acquired_tick >= the beat's start tick — the delta), keyed by the SPEAKER of that
// event. Extraction note (documented honestly): the perception content is the line the engine wrote for
// a Communicated event — apply_event writes the canon summary (the spoken 'stated'); apply_ruled_event
// writes COALESCE(receiver-variant, appearance, truth). Either way the spoken words ride inside the
// perception content the viewer actually sees, so DecodeAndValidateNarration substring-matches a speech
// segment's text against these strings — the exact-words test that rejects narrator paraphrase. Both the
// non-legacy 'Communicated' label (the orchestrator's path, p_legacy_types=false) and the legacy
// 'private_disclosure' label are matched, defensively.
func (h *beatHandler) speechTexts(ctx context.Context, worldID, viewerID string, sinceTick int64) (map[string][]string, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT ep.entity_id::text, pr.content
		   FROM perception_record pr
		   JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.world_id = pr.world_id
		   JOIN event_participant ep ON ep.event_id = ce.event_id AND ep.role_qualifier = 'speaker'
		  WHERE pr.world_id = $1 AND pr.holder_id = $2
		    AND ce.event_type IN ('Communicated', 'private_disclosure')
		    AND pr.acquired_tick >= $3`,
		worldID, viewerID, sinceTick)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var speaker, content string
		if err := rows.Scan(&speaker, &content); err != nil {
			return nil, err
		}
		out[speaker] = append(out[speaker], content)
	}
	return out, rows.Err()
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

	// Build candidate whitelist: present actors + current location. In the SAME row, read the viewer's
	// OWN identity for the narrate YOU ARE block: the descriptor a stranger sees (same Tier-2 attrs.descriptor
	// read for every other candidate) plus the viewer's OWN name-knowledge of himself (fn_perceived_name
	// self→self). NOT the registry canonical name — a viewer may not know his own registered name (§3), and
	// surfacing it would breach the naming wall. Absent both ⇒ no alias list ⇒ the builder omits YOU ARE.
	var loc, viewerDescriptor, viewerKnownName string
	if err := h.pool.QueryRow(ctx,
		`SELECT (attrs->>'location_id')::text,
		        COALESCE(attrs->>'descriptor', ''),
		        COALESCE(fn_perceived_name($1, $2, $2), '')
		 FROM actor_state WHERE world_id=$1 AND entity_id=$2`,
		worldID, viewerID).Scan(&loc, &viewerDescriptor, &viewerKnownName); err != nil {
		return PerceptionPayload{}, err
	}
	// YOU ARE aliases (viewer-relative rendering): descriptor first (the stranger's label), then the
	// viewer's self-known name if he holds one and it differs. Both empty ⇒ ViewerAliases stays nil.
	if viewerDescriptor != "" {
		p.ViewerAliases = append(p.ViewerAliases, viewerDescriptor)
	}
	if viewerKnownName != "" && viewerKnownName != viewerDescriptor {
		p.ViewerAliases = append(p.ViewerAliases, viewerKnownName)
	}

	if loc != "" {
		// Present actors at the same location. The candidate NAME is the VIEWER's display name for each
		// (§3 naming reach): known-name else descriptor else canonical — so decompose CANDIDATES and
		// narrate PRESENT both carry only what the viewer knows. The id stays real (the model binds the
		// id; the label is the viewer's knowledge).
		actorRows, err := h.pool.Query(ctx,
			`SELECT er.entity_id, fn_display_name($1, $3::uuid, er.entity_id), er.entity_kind
			 FROM fn_actors_at($1, $2::uuid) fa
			 JOIN entity_registry er ON er.entity_id=fa.entity_id AND er.world_id=$1`,
			worldID, loc, viewerID)
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

		// Current location entity — the viewer's display name (§3) plus its Tier-2 scene description
		// (empty when unseeded), so the narrate PLACE line renders the room's fixed character as DATA.
		var locName, locDesc string
		if err := h.pool.QueryRow(ctx,
			`SELECT fn_display_name($2, $3::uuid, $1::uuid),
			        COALESCE((SELECT attrs->>'description' FROM location_state WHERE entity_id=$1::uuid AND world_id=$2), '')`,
			loc, worldID, viewerID).Scan(&locName, &locDesc); err != nil {
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
