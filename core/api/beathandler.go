package main

import (
	"context"
	_ "embed"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// the v1 closed-vocabulary schema (legacy; kept so beatseats_test.go and bridge_test.go compile).
//
//go:embed schema/beat_chain.v1.schema.json
var beatChainSchema []byte

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

// The decompose prompt's section markers. They are constants because the DEV decompose stand-in
// (fakeIntentDriver, bridge_fakes.go) parses this prompt back into candidates and the player's raw
// words: if the marker text lived only as a literal in the writer below, a reword here would
// silently turn the dev seat inert instead of failing loudly — the exact shape of the two defects
// recorded in the 2026-08-07 handover §5.
const (
	decomposeCandidatesMarker  = "\nCANDIDATES (bind ids ONLY from this list):\n"
	decomposePlayerInputMarker = "\nPLAYER INPUT:\n"
)

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
	sb.WriteString(decomposeCandidatesMarker)
	for _, c := range payload.Candidates {
		sb.WriteString(c.ID)
		sb.WriteString("  ")
		sb.WriteString(c.Name)
		sb.WriteString("  (")
		sb.WriteString(c.Kind)
		sb.WriteString(")\n")
	}

	// PLAYER INPUT — the mutable tail: the raw words to decompose, LAST so the whole prefix stays cacheable.
	sb.WriteString(decomposePlayerInputMarker)
	sb.WriteString(playerText)
	return sb.String()
}

// beatHandler is no longer an HTTP entry point (rung3 Task 5 deleted POST /worlds/{w}/beat, the
// singular route, with no alias — founder-approved clean cutover). It survives as the shared
// pipeline-helper struct beatsStreamHandler (beatsstream.go) and buildScene (scenehandler.go)
// construct as `&beatHandler{pool: pool}` to reach its two surviving methods: `payload` (the
// perception-bound PerceptionPayload/Candidates assembly) and `speechTexts` (the verbatim-speech
// belt's evidence). dbg/bridge stay on the struct only because a handful of existing tests still
// build a `beatHandler{..., dbg: true}` literal to call `payload` directly; neither field is read by
// anything anymore.
type beatHandler struct {
	pool   *pgxpool.Pool
	dbg    bool
	bridge *Bridge
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
// Also populates Candidates: present actors + current location + the artifacts the viewer PERCEIVES
// (co-located in his room + carried on his person) — the perceived-candidate whitelist (RULINGS-2026-07-30
// §1), bounded by perception (the naming-reach wall, RULINGS-2026-07-23 §3), never a global id dump.
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
		p.Here = loc
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

		// Perceived SCENE ARTIFACTS — the widening (RULINGS-2026-07-30 §1: candidates = everything the
		// actor PERCEIVES, not actors+location only). Two perceived sets, so "approach the bar" / "grab
		// the crate" / "give the note" finally have an id to bind:
		//   • CO-LOCATED artifacts — attrs.location_id = the viewer's current location: the bar he can
		//     approach, the crate he can grab. Same room = perceived, exactly the way present actors are
		//     already treated (co-location IS perception in v1; the founder's "Before You Begin" ruling).
		//   • CARRIED/HELD items — attrs.contained_by = the viewer (the Tier-1 carry edge, §4): the note
		//     he carries, to give.
		// BOUNDED BY PERCEPTION, not global — the naming-reach wall stands (RULINGS-2026-07-23 §3): this
		// yields ONLY artifacts in the viewer's own room or on his own person. An artifact in ANOTHER
		// location (location_id ≠ here) and an item on ANOTHER actor (contained_by ≠ viewer) are excluded
		// by construction. Each labeled by the viewer's fn_display_name (viewer-relative, §3 — the model
		// binds the real id, the label is the viewer's knowledge), same Candidate shape as the actors
		// above. viewerID rides twice ($3 uuid for fn_display_name, $4 text for the contained_by compare)
		// so each param has ONE unambiguous type; DISTINCT folds the (model-disjoint) sets defensively.
		//
		// FURTHER REFINEMENT (noted, deliberately NOT built here — RULINGS-2026-07-30 §1's "one-hop-known
		// absent entities may be included IF the perception/knowledge query already yields them cleanly"):
		// a thing the actor knows-of but cannot currently see needs perception/knowledge-subject-link
		// machinery that does not yet exist cleanly. The co-located + carried set is the immediate correct
		// fix that makes the founder's walkthrough bind; the absent-but-known set is a separate follow-up.
		artRows, err := h.pool.Query(ctx,
			`SELECT DISTINCT er.entity_id, fn_display_name($1, $3::uuid, er.entity_id), er.entity_kind
			 FROM artifact_state a
			 JOIN entity_registry er ON er.entity_id = a.entity_id AND er.world_id = $1
			 WHERE a.world_id = $1
			   AND ( a.attrs->>'location_id' = $2      -- co-located in the viewer's room (perceived)
			      OR a.attrs->>'contained_by' = $4 )`, // carried/held by the viewer (§4 carry edge)
			worldID, loc, viewerID, viewerID)
		if err != nil {
			return PerceptionPayload{}, err
		}
		for artRows.Next() {
			var id, name, kind string
			if err := artRows.Scan(&id, &name, &kind); err != nil {
				artRows.Close()
				return PerceptionPayload{}, err
			}
			p.Candidates = append(p.Candidates, Candidate{ID: id, Name: name, Kind: kind})
		}
		if err := artRows.Err(); err != nil {
			artRows.Close()
			return PerceptionPayload{}, err
		}
		artRows.Close()

		// SPEC-030 — THE WAY OUT. Until this, `ActorMoved` could only ever target the room the actor was
		// already standing in, so no movement of any kind was expressible: no remote location was ever a
		// candidate, and portals carry {"open","locked","connects":[a,b]} with NO location_id, so the
		// co-located-artifact query above never returned the doors either. The Journey shipped in #32
		// and was unreachable by any client; so was walking through a door.
		//
		// Two candidate sources, both from what the actor genuinely PERCEIVES standing here — this adds
		// no mechanism, it stops hiding what the room already contains:
		//
		//   • the PORTALS of this room. A door is part of the room you are in; you can see it, look at
		//     it, and talk about it. Same "co-location IS perception" ruling the artifacts above rest on
		//     (RULINGS-2026-07-30 §1, the founder's "Before You Begin" ruling).
		//   • the LOCATIONS those portals connect to. A visible exit tells you there is somewhere on the
		//     other side; naming it is the viewer's own naming (fn_display_name), exactly as with every
		//     other candidate — the model binds the real id, the label is the viewer's knowledge.
		//
		// STILL BOUNDED BY PERCEPTION: only portals whose `connects` contains THIS room, and only the
		// far side of those portals. A location two rooms away is not here and is not offered.
		//
		// Passage is NOT decided here. A candidate is a thing you may NAME, never a thing you may DO:
		// the accessibility floor (fn_actor_move_permitted, mirrored in premiseHolds' ActorMoved branch)
		// still requires a portal that is open ∧ ¬locked, so "go to the Alley" through a shut back door
		// binds cleanly and is then refused with an honest reason. Offering the target is what lets the
		// world say no; hiding it is what made the refusal impossible to reach.
		//
		// Not built here, and why: the absent-but-known set (RULINGS-2026-07-30 §1's "one-hop-known
		// absent entities … IF the perception/knowledge query already yields them cleanly"). It now
		// WOULD yield cleanly, via fn_entity_visible — but for the seeded viewer that set contains only
		// the room he is in, so it would ship as an unexercised code path. It is the natural next source
		// and belongs with the ruling on long-range travel (see the spec note).
		wayRows, err := h.pool.Query(ctx,
			`SELECT DISTINCT er.entity_id, fn_display_name($1, $3::uuid, er.entity_id), er.entity_kind
			   FROM artifact_state a
			   JOIN entity_registry er ON er.entity_id = a.entity_id AND er.world_id = $1
			  WHERE a.world_id = $1
			    AND a.attrs->'connects' @> to_jsonb($2::text)   -- a portal of THIS room
			 UNION
			 SELECT DISTINCT er.entity_id, fn_display_name($1, $3::uuid, er.entity_id), er.entity_kind
			   FROM artifact_state a
			   CROSS JOIN LATERAL jsonb_array_elements_text(a.attrs->'connects') AS c(loc)
			   JOIN entity_registry er ON er.entity_id = c.loc::uuid AND er.world_id = $1
			  WHERE a.world_id = $1
			    AND a.attrs->'connects' @> to_jsonb($2::text)
			    AND c.loc <> $2`, // the far side only — this room is already a candidate
			worldID, loc, viewerID)
		if err != nil {
			return PerceptionPayload{}, err
		}
		for wayRows.Next() {
			var id, name, kind string
			if err := wayRows.Scan(&id, &name, &kind); err != nil {
				wayRows.Close()
				return PerceptionPayload{}, err
			}
			p.Candidates = append(p.Candidates, Candidate{ID: id, Name: name, Kind: kind})
		}
		if err := wayRows.Err(); err != nil {
			wayRows.Close()
			return PerceptionPayload{}, err
		}
		wayRows.Close()
	}

	// One last pass over the assembled whitelist: where two candidates wear the SAME label, give each
	// the perceived detail that tells them apart (founder ruling, 2026-08-08 — fn_display_names_distinct).
	// It happens HERE, once, over the whole set, because a collision is a property of the group and no
	// per-entity query can see it. Labels that do not collide come back byte-identical, so nothing but
	// the ambiguous case changes.
	//
	// The candidate whitelist has to carry the detail, not just the frontend's "which did you mean?"
	// list: this is the vocabulary the player's next sentence is bound against. Disambiguate only the
	// display list and the ask becomes visible but still unanswerable — the exact complaint that
	// started this — because "by the bar" would name nothing the decomposer could bind.
	if err := relabelDistinct(ctx, h.pool, worldID, viewerID, p.Candidates); err != nil {
		return PerceptionPayload{}, err
	}

	return p, nil
}
