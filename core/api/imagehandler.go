package main

// Governed-by: D-1 — nothing mutates canon directly — proposals only, the Core commits. Also B-1, GA-2.
// Promoted from this file's own citations (2026-08-26), not newly decided. Change what this
// file decides and those decisions change with it (D-9).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
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
var imageScenesRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/images/scenes$`)
var imageRegenerateRoute = regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/images/regenerate$`)

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
		return imageAssetRoute.MatchString(r.URL.Path) &&
			!imagePortraitsRoute.MatchString(r.URL.Path) &&
			!imageScenesRoute.MatchString(r.URL.Path)
	case http.MethodPost:
		return imagePortraitsRoute.MatchString(r.URL.Path) || imageScenesRoute.MatchString(r.URL.Path) || imageRegenerateRoute.MatchString(r.URL.Path)
	}
	return false
}

func (h *imageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && imageRegenerateRoute.MatchString(r.URL.Path):
		h.regenerate(w, r)
	case r.Method == http.MethodPost && imageScenesRoute.MatchString(r.URL.Path):
		h.trigger(w, r, imageScenesRoute, fillScenes)
	case r.Method == http.MethodPost:
		h.trigger(w, r, imagePortraitsRoute, fillPortraits)
	default:
		h.serveAsset(w, r)
	}
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
	worldID, assetID := m[1], m[2]
	if h.client == nil {
		// No platform configured is an ordinary state, not a fault: the world runs without images.
		http.Error(w, "image platform not configured", http.StatusNotFound)
		return
	}
	tier := r.URL.Query().Get("tier") // thumbnail (256) | preview (768, default) | final (1024)

	url, err := h.client.assetURL(r.Context(), assetID, tier)
	if err != nil || url == "" {
		// The platform will not serve this id again — deleted, or retired in place by supersession.
		// This is the only place that can ever notice on the read path: the projection read
		// deliberately never calls the platform, so a stale reference is invisible there. Forgetting
		// it lets the world heal itself — the next read reports `image: null`, the frontend shows its
		// placeholder, and the next portrait trigger refills the slot.
		if errors.Is(err, errAssetGone) {
			forgetAsset(r.Context(), h.pool, worldID, assetID, "asset gone at platform: "+err.Error())
		}
		// Anything else — a 5xx, a timeout, a rate limit — is the platform having a bad minute. The
		// asset is presumed fine and the reference is kept; discarding a good portrait over a blip
		// would be a self-inflicted outage.
		log.Printf("images: asset %s: %v", assetID, err)
		http.Error(w, "image unavailable", http.StatusNotFound)
		return
	}
	// no-store: the redirect TARGET is a short-lived credential. Caching this hop would hand a stale
	// signed URL to the next reader; the picture itself is cached by object storage, not by us.
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, url, http.StatusFound)
}

// forgetAsset drops a reference the platform no longer honours, returning the slot to the state it
// had before its first portrait: empty, refillable, and honest about having no picture. The visual
// identity is deliberately kept — an identity is not an asset, and the fill path re-upserts it
// anyway, so keeping it preserves the actor's appearance across a regeneration.
//
// A free function, not a method: BOTH paths that can discover a dead reference need it — the fetch,
// which trips over one, and the trigger, which now goes looking. One writer, one meaning of empty.
func forgetAsset(ctx context.Context, pool *pgxpool.Pool, worldID, assetID, reason string) {
	if _, err := pool.Exec(ctx, `
		UPDATE image_slot
		   SET asset_id = NULL, job_id = NULL, last_error = $3, updated_at = now()
		 WHERE world_id = $1::uuid AND asset_id = $2`,
		worldID, assetID, reason); err != nil {
		log.Printf("images: forget asset %s: %v", assetID, err)
	}
}

type imageRegenerateResult struct {
	SchemaVersion string `json:"schema_version"`
	Cleared       int64  `json:"cleared"`
}

func (h *imageHandler) regenerate(w http.ResponseWriter, r *http.Request) {
	m := imageRegenerateRoute.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	worldID := m[1]
	// Refusing BEFORE the delete: clearing the cast's slots with no platform configured would erase
	// the world's existing art with nothing able to redraw it — a destructive no-op wearing a 200.
	if h.client == nil {
		http.Error(w, "image platform not configured", http.StatusServiceUnavailable)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM image_slot WHERE world_id=$1::uuid AND owner_kind='actor'`, worldID)
	if err != nil {
		log.Printf("images: regenerate %s: %v", worldID, err)
		http.Error(w, "image regeneration failed", http.StatusInternalServerError)
		return
	}
	kickArt(h.pool, h.client, worldID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(imageRegenerateResult{SchemaVersion: "image_regenerate/1", Cleared: tag.RowsAffected()})
}

type portraitsResult struct {
	SchemaVersion string `json:"schema_version"`
	// Slots freed because the asset they named will not be served again. This is the number that
	// explains a `requested: 0` — without it the endpoint reports "nothing to do" whether the world
	// is fully illustrated or entirely stale, which is how the mosaics survived a live art flip.
	Reclaimed int `json:"reclaimed"`
	Requested int `json:"requested"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
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
//
// trigger serves BOTH bounded generation surfaces — /images/portraits and /images/scenes — because
// everything except the fill function is identical, and the two must never drift on the parts that
// bit us live: platform-not-configured is an ordinary answer, a platform failure is a 502 and not a
// 500, and the counters come back as data so `requested: 0` can explain itself.
func (h *imageHandler) trigger(w http.ResponseWriter, r *http.Request, route *regexp.Regexp,
	fill func(context.Context, *pgxpool.Pool, *imageClient, string, int) (portraitsResult, error)) {
	m := route.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	if h.client == nil {
		http.Error(w, "image platform not configured", http.StatusServiceUnavailable)
		return
	}
	res, err := fill(r.Context(), h.pool, h.client, m[1], imageBatchLimit)
	if err != nil {
		log.Printf("images: %s: %v", r.URL.Path, err)
		http.Error(w, "image generation failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// imageBatchLimit matches the platform's default max_concurrent_jobs. Their §4 note is explicit that
// respecting the concurrency cap is also what keeps polling inside the request-rate budget, so this
// is one number serving both limits rather than two guesses.
const imageBatchLimit = 5

var spriteEmotionOrder = []string{"neutral", "happy", "angry", "sad"}

func spriteVariantKey(emotion string) string {
	return "emotion_" + emotion
}

// The sprite vocabulary lives HERE, not on the image platform. The platform renders caller-defined
// pack cells verbatim — it stores, anchors, and reuses variants without knowing what a key means —
// so what a "happy" bust looks like is this repo's decision, changeable without touching their API.
const (
	// spriteFramingPrompt is the visual-novel staging contract: a bust the frontend layers over a
	// backdrop, so never a full body and never a scene.
	spriteFramingPrompt = "bust portrait, head and chest only, subject centered, facing viewer, plain uniform background"
	// spriteConsistencyPrompt holds the outfit constant across the four renders of one anchored
	// identity — the whole reason the variants are one pack against one anchor.
	spriteConsistencyPrompt = "same character, same outfit, same hairstyle as the reference"
	// spriteAspectRatio is portrait orientation for a bust.
	spriteAspectRatio = "3:4"
)

// spriteEmotionPrompts is the per-emotion expression phrase, written in the vocabulary image models
// respond to (face and posture, not abstract mood words alone).
var spriteEmotionPrompts = map[string]string{
	"neutral": "calm, composed expression",
	"happy":   "warm open smile, bright eyes",
	"angry":   "furrowed brow, gritted teeth, hard stare",
	"sad":     "downcast eyes, sorrowful expression",
}

// spritePackVariants composes the four caller-defined cells for one actor: the entity's authored
// appearance, the framing and consistency clauses, then the emotion. The appearance is the same
// prose the identity was registered with — a picture is of the THING, and the thing is described
// once (portraitAppearance).
func spritePackVariants(appearance string) []imagePackVariant {
	out := make([]imagePackVariant, 0, len(spriteEmotionOrder))
	for _, emotion := range spriteEmotionOrder {
		out = append(out, imagePackVariant{
			Key:    spriteVariantKey(emotion),
			Prompt: appearance + ". " + spriteFramingPrompt + ". " + spriteConsistencyPrompt + ". Expression: " + spriteEmotionPrompts[emotion] + ".",
		})
	}
	return out
}

func portraitAppearance(descriptor, name string) string {
	if descriptor != "" {
		return descriptor
	}
	return name
}

// resolveWorldArtStyle reads the look this world was created with.
//
// Read per fill rather than cached: a style is a property of the world, and the day one can be
// changed after creation this path already picks it up. A world with no choice — every world made
// before the picker existed — resolves to the house style and keeps the profile its art is already
// stored under.
func resolveWorldArtStyle(ctx context.Context, pool *pgxpool.Pool, worldID string) (ArtStyle, error) {
	var raw *string
	err := pool.QueryRow(ctx, `SELECT art_style FROM world WHERE world_id = $1::uuid`, worldID).Scan(&raw)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// No directory row for this id. That is not a style question and this is not the place to
		// answer it: the fills' own target queries return nothing for a world that does not exist, so
		// policing existence here would only duplicate that check and give it a misleading error.
		return artStyleFallback, nil
	case err != nil:
		return ArtStyle{}, fmt.Errorf("resolve world art style: %w", err)
	}
	choice := ""
	if raw != nil {
		choice = *raw
	}
	style, err := ResolveArtStyle(choice)
	if err != nil {
		return ArtStyle{}, fmt.Errorf("resolve world art style: %w", err)
	}
	return style, nil
}

// fillPortraits requests a portrait for up to `limit` actors in this world that do not have one and
// do not already have a job in flight, then waits for each to settle.
//
// It first REAPS: a slot naming an asset the platform will not serve again is not a filled slot, it
// is a dangling id, and the fill query below cannot tell the difference. #53 taught the fetch path
// that lesson and the trigger never learned it, so when the platform archived every mock asset this
// endpoint answered `requested: 0` for a world that was serving nothing but mosaics — "already
// illustrated" and "entirely stale" were the same answer. Asking is bounded by the same `limit` as
// the fill, so one call still cannot outrun the platform.
//
// Every step is written to image_slot as it happens, so an interrupted run resumes instead of
// restarting: an identity already upserted is not upserted again, and a job already requested is
// polled rather than re-requested. Re-requesting would in fact be harmless — reuse is the platform's
// default and returns the same asset at zero cost — but "harmless" is not a reason to do it.
func fillPortraits(ctx context.Context, pool *pgxpool.Pool, client *imageClient, worldID string, limit int) (portraitsResult, error) {
	out := portraitsResult{SchemaVersion: "image_portraits/1"}

	reclaimed, err := reapRetiredAssets(ctx, pool, client, worldID, []string{"actor"}, limit)
	if err != nil {
		return out, err
	}
	out.Reclaimed = reclaimed

	style, err := resolveWorldArtStyle(ctx, pool, worldID)
	if err != nil {
		return out, err
	}
	styleID, err := client.ensureStyle(ctx, style)
	if err != nil {
		return out, err
	}

	rows, err := pool.Query(ctx, `
		SELECT er.entity_id::text, er.canonical_name,
		       coalesce(a.attrs->>'descriptor', '')
		  FROM entity_registry er
		  LEFT JOIN actor_state a ON a.entity_id = er.entity_id AND a.world_id = er.world_id
		  LEFT JOIN LATERAL (
			SELECT
				count(*) AS slot_count,
				count(*) FILTER (WHERE s.variant = ANY($3::text[])) AS emotion_rows,
				count(*) FILTER (WHERE s.variant = ANY($3::text[]) AND s.asset_id IS NOT NULL) AS emotion_filled,
				bool_or(s.variant = ANY($3::text[]) AND s.job_id IS NOT NULL) AS emotion_in_flight,
				count(*) FILTER (WHERE s.variant = 'default' AND s.asset_id IS NOT NULL) AS default_filled,
				bool_or(s.variant = 'default' AND s.job_id IS NOT NULL) AS default_in_flight
			  FROM image_slot s
			 WHERE s.world_id = er.world_id AND s.owner_kind = 'actor' AND s.owner_id = er.entity_id
		  ) st ON true
		 WHERE er.world_id = $1 AND er.entity_kind = 'actor'
		   AND (
			 st.slot_count = 0 OR
			 (st.emotion_rows > 0 AND st.emotion_filled < 4 AND NOT coalesce(st.emotion_in_flight, false)) OR
			 -- Legacy rows whose one picture is GONE (reaped or errored): #58's rule holds — a dead
			 -- reference refills, and it refills in the sprite format. A FILLED legacy slot stays
			 -- untouched; converting live art is the regenerate button's job, not the sweep's.
			 (st.emotion_rows = 0 AND st.default_filled = 0 AND NOT coalesce(st.default_in_flight, false))
		   )
		 ORDER BY er.entity_id
		 LIMIT $2`, worldID, limit, spriteEmotionOrder)
	if err != nil {
		return out, err
	}
	type target struct{ id, name, descriptor string }
	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.id, &t.name, &t.descriptor); err != nil {
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
		appearance := portraitAppearance(t.descriptor, t.name)
		// The identity id is re-read below via getIdentity — the platform keys anchors and packs on it,
		// and the freshly-read copy is the one whose anchor list is trusted.
		if _, err := client.upsertIdentity(ctx, "character", t.id, worldID, t.name, styleID,
			map[string]string{"appearance": appearance}); err != nil {
			out.Failed++
			for _, emotion := range spriteEmotionOrder {
				recordSlotError(ctx, pool, worldID, t.id, emotion, err.Error())
			}
			continue
		}

		identity, err := client.getIdentity(ctx, t.id, worldID)
		if err != nil {
			out.Failed++
			for _, emotion := range spriteEmotionOrder {
				recordSlotError(ctx, pool, worldID, t.id, emotion, err.Error())
			}
			continue
		}
		if len(identity.AnchorAssetIDs) == 0 {
			bootstrapIssuedAt := time.Now().UTC()
			bootstrapKey := "portrait-anchor-" + worldID + "-" + t.id + "-" + bootstrapIssuedAt.Format("20060102T150405Z")
			bootstrapEnv := newGovEnvelope(bootstrapIssuedAt, "character_portrait")

			bootstrapJobID, _, err := client.bootstrapAnchor(ctx, t.id, worldID, styleID, appearance, bootstrapKey, bootstrapEnv)
			if err != nil {
				out.Failed++
				for _, emotion := range spriteEmotionOrder {
					recordSlotError(ctx, pool, worldID, t.id, emotion, err.Error())
				}
				continue
			}
			if bootstrapJobID != "" {
				bootstrapJob, err := client.awaitJob(ctx, bootstrapJobID, defaultPollBackoff())
				if err != nil {
					out.Failed++
					for _, emotion := range spriteEmotionOrder {
						recordSlotError(ctx, pool, worldID, t.id, emotion, err.Error())
					}
					continue
				}
				if bootstrapJob.Status != "completed" || len(bootstrapJob.FinalAssetIDs) == 0 {
					out.Failed++
					errText := bootstrapJob.ErrorCode + ": " + bootstrapJob.ErrorMessage
					for _, emotion := range spriteEmotionOrder {
						recordSlotError(ctx, pool, worldID, t.id, emotion, errText)
					}
					continue
				}
			}
			identity, err = client.getIdentity(ctx, t.id, worldID)
			if err != nil {
				out.Failed++
				for _, emotion := range spriteEmotionOrder {
					recordSlotError(ctx, pool, worldID, t.id, emotion, err.Error())
				}
				continue
			}
		}

		issuedAt := time.Now().UTC()
		key := "sprite-pack-" + worldID + "-" + t.id + "-" + issuedAt.Format("20060102T150405Z")
		env := newGovEnvelope(issuedAt, "expression")
		jobID, err := client.generateCharacterSpritePack(ctx, t.id, worldID, styleID, spritePackVariants(appearance), key, env)
		if err != nil {
			out.Failed++
			for _, emotion := range spriteEmotionOrder {
				recordSlotError(ctx, pool, worldID, t.id, emotion, err.Error())
			}
			continue
		}
		out.Requested++

		for _, emotion := range spriteEmotionOrder {
			if _, err := pool.Exec(ctx, `
				INSERT INTO image_slot (world_id, owner_kind, owner_id, variant, visual_identity_id, job_id, idempotency_key, issued_at, updated_at)
				VALUES ($1,'actor',$2,$3,$4,$5,$6,$7, now())
				ON CONFLICT (world_id, owner_kind, owner_id, variant) DO UPDATE
				   SET visual_identity_id = EXCLUDED.visual_identity_id,
				       job_id             = EXCLUDED.job_id,
				       idempotency_key    = EXCLUDED.idempotency_key,
				       issued_at          = EXCLUDED.issued_at,
				       last_error         = NULL,
				       updated_at         = now()`,
				worldID, t.id, emotion, identity.ID, jobID, key, issuedAt); err != nil {
				return out, err
			}
		}

		job, err := client.awaitJob(ctx, jobID, defaultPollBackoff())
		if err != nil {
			log.Printf("images: job %s not settled: %v", jobID, err)
			continue
		}
		if job.Status != "completed" {
			out.Failed++
			errText := job.ErrorCode + ": " + job.ErrorMessage
			for _, emotion := range spriteEmotionOrder {
				recordSlotError(ctx, pool, worldID, t.id, emotion, errText)
			}
			continue
		}

		assets, err := client.resolveEmotionPackAssets(ctx, jobID, identity.ID)
		if err != nil {
			out.Failed++
			for _, emotion := range spriteEmotionOrder {
				recordSlotError(ctx, pool, worldID, t.id, emotion, err.Error())
			}
			continue
		}

		complete := true
		for _, emotion := range spriteEmotionOrder {
			assetID := assets[emotion]
			if assetID == "" {
				complete = false
				recordSlotError(ctx, pool, worldID, t.id, emotion, "missing variant asset")
				continue
			}
			if _, err := pool.Exec(ctx, `
				UPDATE image_slot SET asset_id=$1, job_id=NULL, last_error=NULL, updated_at=now()
				 WHERE world_id=$2 AND owner_kind='actor' AND owner_id=$3 AND variant=$4`,
				assetID, worldID, t.id, emotion); err != nil {
				return out, err
			}
		}
		if complete {
			out.Completed++
		} else {
			out.Failed++
		}
	}
	out.Skipped = len(targets) - out.Requested - out.Failed
	if out.Skipped < 0 {
		out.Skipped = 0
	}
	return out, nil
}

// reapRetiredAssets returns every slot whose asset the platform will not serve again to the empty
// state, so the fill query can see it. Bounded by the same `limit` as the fill.
//
// Why the trigger has to ASK rather than infer: an archived asset answers `200` and still hands back
// working presigned URLs (their contract: "archived assets remain displayable to the owning
// tenant"), so nothing about a stale slot looks wrong from here. The database cannot know, the
// projection deliberately never calls out, and the fetch path only learns when someone happens to
// request that exact picture. One GET per filled slot is the honest price of the truth.
//
// Deliberately conservative in both directions. Only errAssetGone frees a slot: a 5xx, a timeout or
// a rate limit is the platform having a bad minute, and throwing away a good portrait over a blip
// would turn a wobble into a re-generation bill. And a slot with a job in flight is not examined at
// all — its asset_id is already NULL by construction.
func reapRetiredAssets(ctx context.Context, pool *pgxpool.Pool, client *imageClient, worldID string, kinds []string, limit int) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT asset_id
		  FROM image_slot
		 WHERE world_id = $1::uuid AND owner_kind = ANY($2) AND asset_id IS NOT NULL
		 ORDER BY owner_kind, owner_id, variant
		 LIMIT $3`, worldID, kinds, limit)
	if err != nil {
		return 0, err
	}
	var assets []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			rows.Close()
			return 0, err
		}
		assets = append(assets, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	n := 0
	for _, a := range assets {
		err := client.assetAlive(ctx, a)
		if err == nil {
			continue // a real picture; leave it alone
		}
		if !errors.Is(err, errAssetGone) {
			log.Printf("images: could not verify asset %s, keeping the reference: %v", a, err)
			continue
		}
		forgetAsset(ctx, pool, worldID, a, "reaped by portrait trigger: "+err.Error())
		n++
	}
	return n, nil
}

// recordSlotError leaves the reason on the slot so a blank portrait can explain itself. It clears
// job_id: a failed attempt is not in flight, and leaving a dead id there would make the next run
// poll a job that will never move.
func recordSlotError(ctx context.Context, pool *pgxpool.Pool, worldID, ownerID, variant, msg string) {
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_slot (world_id, owner_kind, owner_id, variant, last_error, job_id, updated_at)
		VALUES ($1,'actor',$2,$3,$4,NULL, now())
		ON CONFLICT (world_id, owner_kind, owner_id, variant) DO UPDATE
		   SET last_error = EXCLUDED.last_error, job_id = NULL, updated_at = now()`,
		worldID, ownerID, variant, msg); err != nil {
		log.Printf("images: recording slot error for %s: %v", ownerID, err)
	}
}

// imageRefsFor resolves image references for a whole set of entities in one round trip. A scene read
// asks for every present actor at once, and most of them will have no picture: doing that one query
// per participant would turn a single read into a dozen for a field that is usually null.
//
// Entities with no slot are simply absent from the returned map, and a missing key yields a nil
// json.RawMessage, which marshals as `null` — the ordinary "no picture yet" the frontend expects.
func imageRefsFor(ctx context.Context, pool *pgxpool.Pool, worldID, ownerKind string, ownerIDs []string) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, fn_image_ref($1::uuid, $2, id)::text
		  FROM unnest($3::uuid[]) AS q(id)`,
		worldID, ownerKind, ownerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var ref *string
		if err := rows.Scan(&id, &ref); err != nil {
			return nil, err
		}
		if ref != nil {
			out[id] = json.RawMessage(*ref)
		}
	}
	return out, rows.Err()
}

func spriteSetsFor(ctx context.Context, pool *pgxpool.Pool, worldID string, ownerIDs []string) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, fn_sprite_set($1::uuid, id)::text
		  FROM unnest($2::uuid[]) AS q(id)`,
		worldID, ownerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var payload *string
		if err := rows.Scan(&id, &payload); err != nil {
			return nil, err
		}
		if payload != nil {
			out[id] = json.RawMessage(*payload)
		}
	}
	return out, rows.Err()
}

// ── scene backdrops: world covers and place backdrops ───────────────────────────────────────────

// fillScenes renders the world's cover and the backdrops for its places, over the platform's
// scene_capable route. Same shape as fillPortraits — reap first, explicit and bounded, never a read
// side effect — and the same slot table, so #58's rule (archived ⇒ dangling ⇒ re-request) covers
// these pictures the day the platform reroutes them too.
//
// ── ONE RULE DECIDES WHAT GETS AN IMAGE: GENERATE FROM AUTHORED FICTION, OR NOT AT ALL ──────────
// A backdrop needs something to be a picture OF, and the only honest source is fiction the world
// already authored. So a place is rendered from `location_state.attrs.description` and a world from
// its `tagline`, and anything carrying neither is simply not a target. That single rule does three
// jobs at once, none of them as a special case:
//
//   - It excludes the waystations the place author mints mid-journey. Those carry area/kind/
//     coordinates and NO description, because nobody wrote them — they are road, procedurally
//     placed. Rendering one would mean inventing the fiction at the boundary, which is exactly the
//     drift GA-2/D-1 forbid, and they multiply with every journey.
//   - It excludes container areas like the Harbor Quarter, which exist to hold other places and
//     have no described interior to stand in.
//   - It makes the founder's approval of a tagline STRUCTURAL rather than procedural: no tagline,
//     no cover, because there is nothing to render from. The gate is the data, not a promise.
//
// Descriptions are read from *_state rather than from perception on purpose, and it is the same
// decision the portrait path already records: a picture is of the THING, not of anyone's opinion of
// it, and the prompt goes to a private service, never to a player. B-1 governs what reaches the
// FRONTEND — and what reaches the frontend here is an asset id and a path.
func fillScenes(ctx context.Context, pool *pgxpool.Pool, client *imageClient, worldID string, limit int) (portraitsResult, error) {
	out := portraitsResult{SchemaVersion: "image_scenes/1"}

	reclaimed, err := reapRetiredAssets(ctx, pool, client, worldID, []string{"world", "location", "artifact"}, limit)
	if err != nil {
		return out, err
	}
	out.Reclaimed = reclaimed

	style, err := resolveWorldArtStyle(ctx, pool, worldID)
	if err != nil {
		return out, err
	}
	styleID, err := client.ensureStyle(ctx, style)
	if err != nil {
		return out, err
	}

	// The cover and the places in one ordered list: cover first, because it is the card the founder
	// is looking at, then places by id for determinism. A slot with a job in flight is excluded by
	// the same predicate the portrait fill uses.
	rows, err := pool.Query(ctx, `
		SELECT 'world', w.world_id::text, w.tagline
		  FROM world w
		  LEFT JOIN image_slot s
		    ON s.world_id = w.world_id AND s.owner_kind = 'world' AND s.owner_id = w.world_id AND s.variant = 'default'
		 WHERE w.world_id = $1 AND w.tagline IS NOT NULL
		   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
		UNION ALL
		SELECT 'location', er.entity_id::text, l.attrs->>'description'
		  FROM entity_registry er
		  JOIN location_state l ON l.world_id = er.world_id AND l.entity_id = er.entity_id
		  LEFT JOIN image_slot s
		    ON s.world_id = er.world_id AND s.owner_kind = 'location' AND s.owner_id = er.entity_id AND s.variant = 'default'
		 WHERE er.world_id = $1 AND er.entity_kind = 'location'
		   AND coalesce(btrim(l.attrs->>'description'), '') <> ''
		   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
		UNION ALL
		-- Objects read their prose from attrs->>'descriptor', which is where an artifact keeps what
		-- a stranger sees; artifacts have no 'description'.
		--
		-- The connects key is what keeps PORTALS out. A doorway is registered as an artifact and DOES carry
		-- a descriptor ("A heavy sliding door between Carriage Four and the Conductor's Cabin"), so
		-- the descriptor test alone let three of them through and billed for pictures of doors. What
		-- separates them is structural rather than textual: a portal names the two places it joins.
		SELECT 'artifact', er.entity_id::text, a.attrs->>'descriptor'
		  FROM entity_registry er
		  JOIN artifact_state a ON a.world_id = er.world_id AND a.entity_id = er.entity_id
		  LEFT JOIN image_slot s
		    ON s.world_id = er.world_id AND s.owner_kind = 'artifact' AND s.owner_id = er.entity_id AND s.variant = 'default'
		 WHERE er.world_id = $1 AND er.entity_kind = 'artifact'
		   AND coalesce(btrim(a.attrs->>'descriptor'), '') <> ''
		   AND NOT (a.attrs ? 'connects')
		   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
		 ORDER BY 1 DESC, 2
		 LIMIT $2`, worldID, limit)
	if err != nil {
		return out, err
	}
	type target struct{ kind, id, description string }
	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.kind, &t.id, &t.description); err != nil {
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
		// Envelope and key pinned together, for the reason the portrait path documents at length:
		// their idempotency key hashes the whole body, so a fresh issued_at under a stable key is a
		// 409 idempotency_conflict rather than a retry.
		issuedAt := time.Now().UTC()
		key := "scene-" + worldID + "-" + t.id + "-" + issuedAt.Format("20060102T150405Z")
		// Their ContentClass enum is character_portrait|place_scene|artifact|expression|angle_variant
		// and has no world-cover member. `place_scene` is the honest nearest for both: a cover is
		// scenery too. Switch if they ever add one rather than stretching `artifact` to fit.
		env := newGovEnvelope(issuedAt, "place_scene")

		jobID, err := client.requestSceneGeneration(ctx, t.id, worldID, styleID, t.description, key, env)
		if err != nil {
			out.Failed++
			recordSceneSlotError(ctx, pool, worldID, t.kind, t.id, err.Error())
			continue
		}
		out.Requested++
		if _, err := pool.Exec(ctx, `
			INSERT INTO image_slot (world_id, owner_kind, owner_id, variant, job_id, idempotency_key, issued_at, updated_at)
			VALUES ($1,$2,$3,'default',$4,$5,$6, now())
			ON CONFLICT (world_id, owner_kind, owner_id, variant) DO UPDATE
			   SET job_id          = EXCLUDED.job_id,
			       idempotency_key = EXCLUDED.idempotency_key,
			       issued_at       = EXCLUDED.issued_at,
			       last_error      = NULL,
			       updated_at      = now()`,
			worldID, t.kind, t.id, jobID, key, issuedAt); err != nil {
			return out, err
		}

		job, err := client.awaitJob(ctx, jobID, defaultPollBackoff())
		if err != nil {
			// Still in flight, or the budget ran out. The row keeps job_id, so a later call resumes
			// this job rather than paying for a second one.
			log.Printf("images: scene job %s not settled: %v", jobID, err)
			continue
		}
		if job.Status != "completed" || len(job.FinalAssetIDs) == 0 {
			out.Failed++
			recordSceneSlotError(ctx, pool, worldID, t.kind, t.id, job.ErrorCode+": "+job.ErrorMessage)
			continue
		}
		if _, err := pool.Exec(ctx, `
			UPDATE image_slot SET asset_id=$1, job_id=NULL, last_error=NULL, updated_at=now()
			 WHERE world_id=$2 AND owner_kind=$3 AND owner_id=$4 AND variant='default'`,
			job.FinalAssetIDs[0], worldID, t.kind, t.id); err != nil {
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

// recordSceneSlotError is recordSlotError for a non-actor owner: same "a blank picture explains
// itself, and a dead job_id is cleared so the next run does not poll it forever" contract.
func recordSceneSlotError(ctx context.Context, pool *pgxpool.Pool, worldID, ownerKind, ownerID, msg string) {
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_slot (world_id, owner_kind, owner_id, variant, last_error, updated_at)
		VALUES ($1,$2,$3,'default',$4, now())
		ON CONFLICT (world_id, owner_kind, owner_id, variant) DO UPDATE
		   SET last_error = EXCLUDED.last_error, job_id = NULL, updated_at = now()`,
		worldID, ownerKind, ownerID, msg); err != nil {
		log.Printf("images: record scene slot error: %v", err)
	}
}
