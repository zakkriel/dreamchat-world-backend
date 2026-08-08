package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The two image surfaces this service owns.
//
//	GET  /worlds/{w}/images/{asset_id}   — hand the browser a picture
//	POST /worlds/{w}/images/portraits    — ask the platform for the portraits this world is missing
//
// The read is what makes an `image_ref/1` usable without ever persisting a credential; the write is
// where generation is TRIGGERED, deliberately kept off every read path.
var imageAssetRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/images/([A-Za-z0-9_-]{1,128})$`)
var imagePortraitsRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/images/portraits$`)

type imageHandler struct {
	pool   *pgxpool.Pool
	client *imageClient
	dbg    bool
}

func NewImageHandler(pool *pgxpool.Pool, client *imageClient, debug bool) http.Handler {
	return &imageHandler{pool: pool, client: client, dbg: debug}
}

func (h *imageHandler) Match(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet:
		return imageAssetRoute.MatchString(r.URL.Path) && !imagePortraitsRoute.MatchString(r.URL.Path)
	case http.MethodPost:
		return imagePortraitsRoute.MatchString(r.URL.Path)
	}
	return false
}

func (h *imageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.portraits(w, r)
		return
	}
	h.serveAsset(w, r)
}

// serveAsset redirects to a FRESHLY minted presigned URL.
//
// This is the whole reason an image reference can be plain data. Their presigned URLs expire in ~15
// minutes and must never be persisted ("IDs over the wire, URLs fetched on demand"), so a payload
// that embedded one would be a payload that rots — unusable from a cache, a log, or a page left open
// over lunch. A payload carrying THIS path instead stays valid forever, and the credential is minted
// per read and never written down.
//
// A 302 rather than a proxy: the browser fetches the bytes straight from object storage, so images
// never travel through this service and a large asset costs us one small request instead of a
// stream. `<img src="{apiBase}{path}">` simply works, with no expiry handling in the client at all.
func (h *imageHandler) serveAsset(w http.ResponseWriter, r *http.Request) {
	m := imageAssetRoute.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	assetID := m[2]
	if h.client == nil {
		// No platform configured is an ordinary state, not a fault: the world runs without images.
		http.Error(w, "image platform not configured", http.StatusNotFound)
		return
	}
	tier := r.URL.Query().Get("tier") // thumbnail (256) | preview (768, default) | final (1024)

	url, err := h.client.assetURL(r.Context(), assetID, tier)
	if err != nil || url == "" {
		log.Printf("images: asset %s: %v", assetID, err)
		http.Error(w, "image unavailable", http.StatusNotFound)
		return
	}
	// no-store: the redirect TARGET is a short-lived credential. Caching this hop would hand a stale
	// signed URL to the next reader; the picture itself is cached by object storage, not by us.
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, url, http.StatusFound)
}

type portraitsResult struct {
	SchemaVersion string `json:"schema_version"`
	Requested     int    `json:"requested"`
	Completed     int    `json:"completed"`
	Failed        int    `json:"failed"`
	Skipped       int    `json:"skipped"`
}

// portraits fills in the portraits a world is missing.
//
// ── WHY GENERATION IS TRIGGERED HERE, AND NOT ANYWHERE ELSE ─────────────────────────────────────
// Two placements suggest themselves and both are wrong:
//
//   - ON ENTITY CREATION — a seeded world would fire a job per entity the moment it loads. The
//     platform's default concurrency is 5 and its rate limit is 60/minute per token counted on reads
//     too, so a dozen entities is an instant self-inflicted 429 before anyone has looked at anything.
//   - ON FIRST VIEW — a GET would acquire a side effect, an external call, and a latency spike, and
//     compendium reads would start failing for reasons that have nothing to do with the compendium.
//     A read that mutates is a read you cannot retry.
//
// So generation is an EXPLICIT, BOUNDED, operator-driven action, and reads stay pure. A slot with no
// asset simply reports null, the frontend shows its placeholder, and a later read swaps the picture
// in with no re-request. Nothing about the world's behaviour depends on whether this endpoint has
// ever been called — which is the property that lets the image platform be genuinely optional.
//
// `limit` caps the batch at the platform's own concurrency default so one call cannot outrun it.
func (h *imageHandler) portraits(w http.ResponseWriter, r *http.Request) {
	m := imagePortraitsRoute.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	if h.client == nil {
		http.Error(w, "image platform not configured", http.StatusServiceUnavailable)
		return
	}
	worldID := m[1]

	res, err := fillPortraits(r.Context(), h.pool, h.client, worldID, imageBatchLimit)
	if err != nil {
		log.Printf("images: portraits: %v", err)
		http.Error(w, "portrait generation failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// imageBatchLimit matches the platform's default max_concurrent_jobs. Their §4 note is explicit that
// respecting the concurrency cap is also what keeps polling inside the request-rate budget, so this
// is one number serving both limits rather than two guesses.
const imageBatchLimit = 5

// fillPortraits requests a portrait for up to `limit` actors in this world that do not have one and
// do not already have a job in flight, then waits for each to settle.
//
// Every step is written to image_slot as it happens, so an interrupted run resumes instead of
// restarting: an identity already upserted is not upserted again, and a job already requested is
// polled rather than re-requested. Re-requesting would in fact be harmless — reuse is the platform's
// default and returns the same asset at zero cost — but "harmless" is not a reason to do it.
func fillPortraits(ctx context.Context, pool *pgxpool.Pool, client *imageClient, worldID string, limit int) (portraitsResult, error) {
	out := portraitsResult{SchemaVersion: "image_portraits/1"}

	styleID, err := client.ensureStyle(ctx, "dreamchat-default")
	if err != nil {
		return out, err
	}

	// Actors with no picture and nothing in flight. fn_display_name is NOT used: a portrait is of the
	// entity itself, not of anyone's opinion of it, so the platform is told the canonical name. This
	// is the one place a canonical name legitimately leaves the engine, and it leaves it to a private
	// service rather than to a player — no perception boundary is crossed (B-1 governs what reaches
	// the FRONTEND, and a portrait's prompt never does).
	rows, err := pool.Query(ctx, `
		SELECT er.entity_id::text, er.canonical_name
		  FROM entity_registry er
		  LEFT JOIN image_slot s
		    ON s.world_id = er.world_id AND s.owner_kind = 'actor' AND s.owner_id = er.entity_id
		 WHERE er.world_id = $1 AND er.entity_kind = 'actor'
		   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
		 ORDER BY er.entity_id
		 LIMIT $2`, worldID, limit)
	if err != nil {
		return out, err
	}
	type target struct{ id, name string }
	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.id, &t.name); err != nil {
			rows.Close()
			return out, err
		}
		targets = append(targets, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	for _, t := range targets {
		identityID, err := client.upsertIdentity(ctx, "character", t.id, worldID, t.name, styleID, nil)
		if err != nil {
			out.Failed++
			recordSlotError(ctx, pool, worldID, t.id, err.Error())
			continue
		}

		// The envelope is pinned HERE, once, and stored with the key. Their idempotency key hashes the
		// whole body, and issued_at moves every time an envelope is built — so composing a fresh one
		// on retry is a different body under the same key and returns 409 idempotency_conflict. This
		// is the trap their own verification run hit, and the reason both values live in the row.
		issuedAt := time.Now().UTC()
		key := "portrait-" + worldID + "-" + t.id
		env := newGovEnvelope(issuedAt, "character_portrait")

		jobID, err := client.requestGeneration(ctx, identityID, key, env)
		if err != nil {
			out.Failed++
			recordSlotError(ctx, pool, worldID, t.id, err.Error())
			continue
		}
		out.Requested++
		if _, err := pool.Exec(ctx, `
			INSERT INTO image_slot (world_id, owner_kind, owner_id, visual_identity_id, job_id, idempotency_key, issued_at, updated_at)
			VALUES ($1,'actor',$2,$3,$4,$5,$6, now())
			ON CONFLICT (world_id, owner_kind, owner_id) DO UPDATE
			   SET visual_identity_id = EXCLUDED.visual_identity_id,
			       job_id             = EXCLUDED.job_id,
			       idempotency_key    = EXCLUDED.idempotency_key,
			       issued_at          = EXCLUDED.issued_at,
			       last_error         = NULL,
			       updated_at         = now()`,
			worldID, t.id, identityID, jobID, key, issuedAt); err != nil {
			return out, err
		}

		job, err := client.awaitJob(ctx, jobID, defaultPollBackoff())
		if err != nil {
			// Still in flight, or the budget ran out. The row keeps job_id, so a later call resumes
			// this job rather than paying for a second one.
			log.Printf("images: job %s not settled: %v", jobID, err)
			continue
		}
		if job.Status != "completed" || len(job.FinalAssetIDs) == 0 {
			out.Failed++
			recordSlotError(ctx, pool, worldID, t.id, job.ErrorCode+": "+job.ErrorMessage)
			continue
		}

		// final_asset_ids is OMITTED entirely while empty rather than sent as [], so its presence is
		// itself the signal — checked above before indexing.
		if _, err := pool.Exec(ctx, `
			UPDATE image_slot SET asset_id=$1, job_id=NULL, last_error=NULL, updated_at=now()
			 WHERE world_id=$2 AND owner_kind='actor' AND owner_id=$3`,
			job.FinalAssetIDs[0], worldID, t.id); err != nil {
			return out, err
		}
		out.Completed++
	}
	out.Skipped = len(targets) - out.Requested - out.Failed
	if out.Skipped < 0 {
		out.Skipped = 0
	}
	return out, nil
}

// recordSlotError leaves the reason on the slot so a blank portrait can explain itself. It clears
// job_id: a failed attempt is not in flight, and leaving a dead id there would make the next run
// poll a job that will never move.
func recordSlotError(ctx context.Context, pool *pgxpool.Pool, worldID, ownerID, msg string) {
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_slot (world_id, owner_kind, owner_id, last_error, job_id, updated_at)
		VALUES ($1,'actor',$2,$3,NULL, now())
		ON CONFLICT (world_id, owner_kind, owner_id) DO UPDATE
		   SET last_error = EXCLUDED.last_error, job_id = NULL, updated_at = now()`,
		worldID, ownerID, msg); err != nil {
		log.Printf("images: recording slot error for %s: %v", ownerID, err)
	}
}
