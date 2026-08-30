package main

// Governed-by: ADR-P017 — Go is the backend language.
// Promoted from this file's own citations (2026-08-26), not newly decided. Change what this
// file decides and those decisions change with it (D-9).

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// persistTranscript records one beat of the viewer's lived story: what he typed, and the narration
// exactly as it reached him.
//
// Written POST-BELT, from the same `[]beatMessage` the frames carry — including a naming-wall-scrubbed
// fallback line — so the row is what the player legitimately saw and nothing he did not.
//
// It stores rendered TEXT, never ids to re-resolve later. An entry written before Kade learned the
// name says "the muscle by the bar" forever, and must: rewriting a memory to match present knowledge
// would forge the only record that proves he learned anything (see migration 20260809090008).
//
// Failure NEVER touches the beat. The canon is committed and the prose is already on the wire by the
// time this runs; losing a transcript row is a lost memory, not a lost world, and killing the
// response over it would trade the founder's beat for his scrollback.
func persistTranscript(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string,
	tick int64, stated string, messages []beatMessage, haltReason string, journey *journeyBlock) {

	if len(messages) == 0 {
		return // nothing was narrated: a beat that produced no prose leaves no memory
	}
	segments, err := json.Marshal(messages)
	if err != nil {
		log.Printf("transcript: marshal segments: %v", err)
		return
	}
	var journeyJSON []byte
	if journey != nil {
		if journeyJSON, err = json.Marshal(journey); err != nil {
			log.Printf("transcript: marshal journey: %v", err)
			journeyJSON = nil
		}
	}
	// A NULL `stated` means "no text came in with this beat", which is a different fact from an empty
	// string and is worth keeping distinguishable in the record. It was the ordinary case while the
	// continue press existed; since that was deleted (2026-08-28) every beat carries a sentence, so a
	// NULL here now marks either a historical continue row or a malformed client — and the published
	// transcript/2 contract still declares the field nullable for exactly the historical rows.
	var statedArg any
	if stated != "" {
		statedArg = stated
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO transcript_entry (world_id, viewer_id, in_world_tick, stated, segments, halt_reason, journey)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7::jsonb)`,
		worldID, viewerID, tick, statedArg, string(segments), nullableText(haltReason), nullableJSON(journeyJSON)); err != nil {
		log.Printf("transcript: write failed for world %s viewer %s: %v", worldID, viewerID, err)
	}
}

// nullableText keeps "" out of the record as SQL NULL: "the beat had no halt reason" is a different
// fact from "its halt reason was the empty string". (scenehandler.go's nullIfEmpty returns *string for
// JSON rendering; this one feeds a query argument.)
func nullableText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

// transcriptHandler serves GET /worlds/{w}/transcript → transcript/2: the viewer's story, newest
// first, paginated with an entry_no cursor.
//
// Thin reader (ADR-P017): the lens is fn_transcript. Viewer-scoped by the same ResolveViewer every
// other read surface uses — a transcript is one person's experience, and no request can name someone
// else's, because the route has no viewer segment to name it with.
type transcriptHandler struct {
	pool *pgxpool.Pool
	dbg  bool
	re   *regexp.Regexp
}

func NewTranscriptHandler(pool *pgxpool.Pool, debug bool) http.Handler {
	return &transcriptHandler{
		pool: pool, dbg: debug,
		re: regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/transcript$`),
	}
}

func (h *transcriptHandler) Match(r *http.Request) bool {
	return r.Method == http.MethodGet && h.re.MatchString(r.URL.Path)
}

func (h *transcriptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := h.re.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	ctx := r.Context()

	viewerID, err := ResolveViewer(ctx, h.pool, worldID, r.URL.Query().Get("viewer"), h.dbg)
	if writeNoViewer(w, err) {
		return
	}

	// `before` is the exclusive cursor from the previous page's `next_before`. A malformed value is
	// refused rather than silently ignored: quietly serving page 1 to a client that asked for page 4
	// looks like the end of the story.
	var before any
	if raw := r.URL.Query().Get("before"); raw != "" {
		n, convErr := strconv.ParseInt(raw, 10, 64)
		if convErr != nil {
			http.Error(w, "before must be an integer entry_no", http.StatusBadRequest)
			return
		}
		before = n
	}
	var limit any
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, convErr := strconv.Atoi(raw)
		if convErr != nil {
			http.Error(w, "limit must be an integer", http.StatusBadRequest)
			return
		}
		limit = n // fn_transcript clamps to 1..200; the caller does not get to ask for the whole world
	}

	var payload []byte
	if err := h.pool.QueryRow(ctx,
		`SELECT fn_transcript($1, $2, $3::bigint, $4::int)::text`,
		worldID, viewerID, before, limit).Scan(&payload); err != nil {
		log.Printf("transcript: read failed: %v", err)
		http.Error(w, "projection failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
