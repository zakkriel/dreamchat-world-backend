package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// The Image Platform client. Written against dreamchat-Image-Platform's
// docs/api/integration-quickstart.md — every request and response shape here is copied from a
// sequence they executed against a real stack, not inferred from an OpenAPI file.
//
// ── PULL IS THE CONTRACT OF RECORD ──────────────────────────────────────────────────────────────
// Generation is asynchronous: POST returns 202 with a job_id, and the outcome is learned by READING
// GET /v1/jobs/{id} then GET /v1/jobs/{id}/assets. Their webhooks exist but are documented as a
// latency hint only — at-least-once, some transitions deliberately never emit, bodies carry ids
// only, and "a consumer that ignores webhooks entirely is still correct". We ignore them entirely.
// That is the same position this repo already holds on the async channel: a channel that can only
// ever be silent is scaffolding pretending to be a feature.
//
// ── IDS OVER THE WIRE ───────────────────────────────────────────────────────────────────────────
// Their first invariant. We persist asset_id and NEVER a *_download_url: those are presigned per
// read and expire (default 15 minutes). Every URL this client obtains is used immediately and
// discarded.
//
// ── DISABLED IS A NORMAL STATE ──────────────────────────────────────────────────────────────────
// No base URL or no token means no client, and every image reference stays null. The world must run
// perfectly well with no image platform attached — it did for the whole product's life until now,
// and a hard dependency on an optional adornment would be the wrong coupling.

const (
	imageBaseURLEnv = "DREAMCHAT_IMAGE_BASE_URL"
	imageTokenEnv   = "DREAMCHAT_IMAGE_API_TOKEN"

	// Optional provider pin for the scene_capable route. EMPTY BY DEFAULT, which means "you route
	// it" — which provider satisfies a capability is the platform's decision, not ours (D-3,
	// mirrored), and hardcoding one here would freeze their router from the outside.
	//
	// It exists because the principle and the deployment disagreed, and the deployment won. Measured
	// from their own startup log: `route_mock_text_to_image_{draft,high,standard}` are valid for
	// scene_capable, `route_bfl_text_to_image_standard` is valid for scene_capable, and the service
	// banner reads `image_provider: mock` — so an unpinned scene request resolves MOCK even with real
	// keys present, and their operator note says so outright: "+fal configured — pin provider_id=fal
	// to use it". Portraits escape this only by accident: identity work fail-closes mock
	// (`synthetic_identity_disabled`), so fal wins by elimination. Scenes have no such elimination.
	//
	// So the choice is a config knob, not a constant, and it mirrors DREAMCHAT_RESOLVE_PROVIDER
	// (main.go) — the pattern this repo already uses to re-point one seat without teaching the code a
	// provider name. Unset it and we are back to letting them route. The real fix is theirs: a
	// platform holding real keys should not default scene work to a mock provider.
	imageSceneProviderEnv = "DREAMCHAT_IMAGE_SCENE_PROVIDER"

	// The governance envelope's authorized_by must appear in the platform's
	// GOVERNANCE_AUTHORIZED_ISSUERS allowlist, or the request is recorded as
	// eligibility_blocked/unknown_issuer — silently under log_only, and 403 under enforce.
	imageAuthorizedBy = "svc_world_backend"

	// Signature verification is StubSignatureVerifier today: it passes any NON-EMPTY string, because
	// the real canonicalization is a cross-system contract with this repo that has not been designed
	// (their TODO(core-signing)), and they correctly refused to invent it. A self-describing sentinel
	// so it can never be mistaken for crypto and is greppable the day signing lands. An empty string
	// is a hard 422 regardless of mode.
	imageSignatureStub = "stub-unsigned-v1"
)

// imageClient talks to the platform. Nil means "no image platform configured", which every caller
// must treat as ordinary.
type imageClient struct {
	baseURL string
	token   string
	// sceneProvider pins provider_id on the scene_capable route. Empty = unpinned; see
	// imageSceneProviderEnv.
	sceneProvider string
	http          *http.Client
}

// newImageClientFromEnv returns nil when the platform is not configured. The token is read once at
// startup and never written to the database, never logged, and never crosses the API boundary.
func newImageClientFromEnv() *imageClient {
	base, token := os.Getenv(imageBaseURLEnv), os.Getenv(imageTokenEnv)
	if base == "" || token == "" {
		return nil
	}
	return &imageClient{
		baseURL:       base,
		token:         token,
		sceneProvider: os.Getenv(imageSceneProviderEnv),
		http:          &http.Client{Timeout: 30 * time.Second},
	}
}

// imageAPIError carries the platform's own error taxonomy so callers can act on it rather than on a
// string. Their codes are stable and documented; the ones that change behaviour here are
// rate_limit_exceeded, concurrent_jobs_exceeded and idempotency_conflict.
type imageAPIError struct {
	Status     int
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	RetryAfter time.Duration
}

func (e *imageAPIError) Error() string {
	return fmt.Sprintf("image platform %d %s: %s (request_id=%s)", e.Status, e.Code, e.Message, e.RequestID)
}

// retryable reports whether waiting could plausibly change the answer. A 429 of either kind is the
// clear yes; 5xx is a maybe worth one more try. Everything else — a missing field, an unconfigured
// route, an exhausted budget — will fail identically forever and must surface, not spin.
func (e *imageAPIError) retryable() bool {
	switch e.Code {
	case "rate_limit_exceeded", "concurrent_jobs_exceeded":
		return true
	}
	return e.Status >= 500
}

// do performs one request. It never retries: retry policy belongs to the caller that knows whether
// the request is safe to repeat (a poll always is; a generation POST is only safe because its
// idempotency key and pinned envelope make it byte-identical).
func (c *imageClient) do(ctx context.Context, method, path string, body any, idempotencyKey string, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		// Mandatory and canonical on POST /v1/generations — neither header nor body key is a 422.
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		apiErr := &imageAPIError{Status: resp.StatusCode}
		_ = json.Unmarshal(payload, apiErr) // a non-JSON error body still yields the status
		// Retry-After is authoritative on rate_limit_exceeded and DELIBERATELY absent on
		// concurrent_jobs_exceeded, which clears when a job reaches a terminal state rather than on
		// a clock — so the two 429s are handled differently, never as one "slow down".
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, convErr := strconv.Atoi(ra); convErr == nil && secs >= 0 {
				apiErr.RetryAfter = time.Duration(secs) * time.Second
			}
		}
		return apiErr
	}
	if out != nil && len(payload) > 0 {
		return json.Unmarshal(payload, out)
	}
	return nil
}

// ── governance envelope ─────────────────────────────────────────────────────────────────────────

type govEnvelope struct {
	SchemaVersion    string `json:"schema_version"`
	ClassificationID string `json:"classification_id"`
	Visibility       string `json:"visibility"`
	ContentClass     string `json:"content_class"`
	AuthorizedBy     string `json:"authorized_by"`
	IssuedAt         string `json:"issued_at"`
	Signature        string `json:"signature"`
}

// newGovEnvelope builds all seven required fields. issuedAt is PASSED IN, never taken from the clock
// here, and that is the whole point: their idempotency key hashes the entire body, so an envelope
// rebuilt with a fresh timestamp is a different body under the same key and returns 409
// idempotency_conflict. The caller pins one issued_at per logical request, stores it beside the key,
// and replays it verbatim on every retry.
//
// contentClass is opaque to the platform — stored and logged, never parsed — so it carries our own
// vocabulary rather than one negotiated with them.
func newGovEnvelope(issuedAt time.Time, contentClass string) govEnvelope {
	return govEnvelope{
		SchemaVersion:    "1.0",
		ClassificationID: "cls_poc_default",
		Visibility:       "private",
		ContentClass:     contentClass,
		AuthorizedBy:     imageAuthorizedBy,
		IssuedAt:         issuedAt.UTC().Format(time.RFC3339),
		Signature:        imageSignatureStub,
	}
}

// ── the five calls ──────────────────────────────────────────────────────────────────────────────

type styleProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ensureStyle finds or creates the tenant-wide style profile. Style profiles created through the API
// are tenant-wide (world_id NULL) and per-world ones are not creatable, so there is exactly one to
// find. Listing first keeps this idempotent without relying on a create-conflict behaviour they do
// not document.
func (c *imageClient) ensureStyle(ctx context.Context, name string) (string, error) {
	var list struct {
		Styles []styleProfile `json:"styles"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/styles", nil, "", &list); err == nil {
		for _, s := range list.Styles {
			if s.Name == name {
				return s.ID, nil
			}
		}
	}
	var created styleProfile
	// positive_prompt is the required field (NOT prompt_fragment), and style_mode must be one of
	// open_prompt | preset_style | creator_style | provider_native — both traps hit on the first
	// attempts of their verification run.
	err := c.do(ctx, http.MethodPost, "/v1/styles", map[string]any{
		"name":                 name,
		"style_mode":           "open_prompt",
		"positive_prompt":      "painterly, soft rim light, cinematic",
		"negative_prompt":      "text, watermark",
		"default_quality_tier": "standard",
	}, "", &created)
	if err != nil {
		return "", fmt.Errorf("ensureStyle: %w", err)
	}
	return created.ID, nil
}

type visualIdentity struct {
	ID             string   `json:"id"`
	AnchorAssetIDs []string `json:"anchor_asset_ids"`
}

// upsertIdentity registers an entity with the platform and returns the visual_identity_id that every
// later generation refers to. Upsert is keyed on (tenant_id, world_id, owner_type, owner_id), so
// replaying it is safe; owner_type must match the route and owner_id must equal the path parameter.
//
// world_id here is OUR world uuid as an opaque string — their world_id is a plain scoping column,
// which is exactly why one tenant can hold many worlds (their §7, and the tenancy answer we gave).
func (c *imageClient) upsertIdentity(ctx context.Context, ownerType, ownerID, worldID, displayName, styleID string, traits map[string]string) (string, error) {
	var vi visualIdentity
	path := "/v1/characters/" + ownerID + "/visual-identity"
	if ownerType != "character" {
		path = "/v1/" + ownerType + "s/" + ownerID + "/visual-identity"
	}
	body := map[string]any{
		"owner_type":       ownerType,
		"owner_id":         ownerID,
		"world_id":         worldID,
		"display_name":     displayName,
		"style_profile_id": styleID,
	}
	if len(traits) > 0 {
		body["canonical_visual_traits"] = traits
	}
	if err := c.do(ctx, http.MethodPost, path, body, "", &vi); err != nil {
		return "", fmt.Errorf("upsertIdentity: %w", err)
	}
	return vi.ID, nil
}

// getIdentity reads the current visual identity for one owner, including the anchor assets used to
// condition future generations.
//
// world_id is a REQUIRED query parameter, not decoration: an identity is keyed on
// (tenant, world, owner_type, owner) and the platform answers 400 invalid_request without it. Omitting
// it failed every portrait in this world with a 400 recorded as the slot's error, which reads like a
// generation fault and is not one.
func (c *imageClient) getIdentity(ctx context.Context, ownerID, worldID string) (visualIdentity, error) {
	var vi visualIdentity
	path := "/v1/characters/" + ownerID + "/visual-identity?world_id=" + url.QueryEscape(worldID)
	if err := c.do(ctx, http.MethodGet, path, nil, "", &vi); err != nil {
		return visualIdentity{}, fmt.Errorf("getIdentity: %w", err)
	}
	return vi, nil
}

// bootstrapAnchor asks the platform to mint the first anchor for an identity that has none.
// 200 means the identity is already anchored and no job is started; 202 returns a job to await.
func (c *imageClient) bootstrapAnchor(ctx context.Context, ownerID, worldID, styleID, description, idempotencyKey string, env govEnvelope) (jobID string, alreadyAnchored bool, err error) {
	reqBody := map[string]any{
		"governance":       env,
		"world_id":         worldID,
		"style_profile_id": styleID,
		"description":      description,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", false, fmt.Errorf("bootstrapAnchor: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/characters/"+ownerID+"/visual-identity/bootstrap-anchor", bytes.NewReader(b))
	if err != nil {
		return "", false, fmt.Errorf("bootstrapAnchor: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("bootstrapAnchor: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", false, fmt.Errorf("bootstrapAnchor: %w", err)
	}
	if resp.StatusCode >= 400 {
		apiErr := &imageAPIError{Status: resp.StatusCode}
		_ = json.Unmarshal(payload, apiErr)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, convErr := strconv.Atoi(ra); convErr == nil && secs >= 0 {
				apiErr.RetryAfter = time.Duration(secs) * time.Second
			}
		}
		return "", false, fmt.Errorf("bootstrapAnchor: %w", apiErr)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return "", true, nil
	case http.StatusAccepted:
		var acc generationAccepted
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &acc); err != nil {
				return "", false, fmt.Errorf("bootstrapAnchor: %w", err)
			}
		}
		return acc.JobID, false, nil
	default:
		return "", false, fmt.Errorf("bootstrapAnchor: unexpected status %d", resp.StatusCode)
	}
}

type generationAccepted struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// requestGeneration asks for one image. The body is assembled from the PINNED envelope so that a
// retry under the same idempotency key is byte-identical.
//
// world_id is deliberately absent: it is not accepted here and is derived from the identity. Unknown
// fields are rejected by strict decoding, so nothing extra may be added "for context".
func (c *imageClient) requestGeneration(ctx context.Context, identityID, idempotencyKey string, env govEnvelope) (string, error) {
	var acc generationAccepted
	err := c.do(ctx, http.MethodPost, "/v1/generations", map[string]any{
		"governance": env,
		"subject":    map[string]any{"identity_id": identityID},
		"render":     map[string]any{"intent": "commit"},
	}, idempotencyKey, &acc)
	if err != nil {
		return "", fmt.Errorf("requestGeneration: %w", err)
	}
	return acc.JobID, nil
}

// requestSceneGeneration asks for ONE image rendered from an authored description, over the
// platform's scene_capable route (`POST /v1/artifacts/{id}/generate`). Despite the path segment it
// is not about Artifacts in this repo's sense — it is their description-conditioned single-image
// route, and it is the one their contract marks scene_capable. `id` is OUR key for the thing being
// rendered: a location entity id for a backdrop, a world id for a cover.
//
// The place-pack route (`/v1/places/{id}/generate-pack`) is deliberately NOT used. It asks for
// pack_capable and renders a fixed SET of variant roles from a pack template — time-of-day, state —
// against a place visual identity. We want one backdrop conditioned on the fiction the world already
// authored, so a pack would be several images we did not ask for and an identity we have nothing to
// anchor.
//
// provider_id is sent only when DREAMCHAT_IMAGE_SCENE_PROVIDER is set, and unset is the default.
// Which provider satisfies scene_capable is the platform's routing decision, not ours (D-3,
// mirrored) — but measured against the live platform, an unpinned scene request resolves MOCK even
// with real keys configured, so the knob exists and the deployment sets it. See
// imageSceneProviderEnv for the routing evidence and why this is config rather than a constant.
//
// Same envelope discipline as requestGeneration: the caller pins issued_at and stores it with the
// key, because their idempotency key hashes the whole body.
func (c *imageClient) requestSceneGeneration(ctx context.Context, subjectID, worldID, styleID, description, idempotencyKey string, env govEnvelope) (string, error) {
	body := map[string]any{
		"governance":       env,
		"world_id":         worldID,
		"style_profile_id": styleID,
		"description":      description,
	}
	// Included only when set, so an unpinned deployment sends a body with no provider_id at all
	// rather than an explicit null their strict decoder would have to interpret.
	if c.sceneProvider != "" {
		body["provider_id"] = c.sceneProvider
	}
	var acc generationAccepted
	err := c.do(ctx, http.MethodPost, "/v1/artifacts/"+subjectID+"/generate", body, idempotencyKey, &acc)
	if err != nil {
		return "", fmt.Errorf("requestSceneGeneration: %w", err)
	}
	return acc.JobID, nil
}

type imageJob struct {
	ID            string   `json:"id"`
	Status        string   `json:"status"`
	FinalAssetIDs []string `json:"final_asset_ids"` // OMITTED entirely while empty — never []
	ErrorCode     string   `json:"error_code"`
	ErrorMessage  string   `json:"error_message"`
	Retryable     bool     `json:"retryable"`
}

func (j imageJob) terminal() bool {
	switch j.Status {
	case "completed", "failed", "cancelled":
		return true
	}
	return false
}

func (c *imageClient) getJob(ctx context.Context, jobID string) (imageJob, error) {
	var j imageJob
	if err := c.do(ctx, http.MethodGet, "/v1/jobs/"+jobID, nil, "", &j); err != nil {
		return imageJob{}, fmt.Errorf("getJob: %w", err)
	}
	return j, nil
}

type imageAsset struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	ThumbnailURL     string  `json:"thumbnail_download_url"`
	PreviewURL       string  `json:"preview_download_url"`
	FinalURL         string  `json:"final_download_url"`
	URLExpiresAt     string  `json:"url_expires_at"`
	VisualIdentityID *string `json:"visual_identity_id"` // NULL on this path — see image_slot's migration
}

func (c *imageClient) jobAssets(ctx context.Context, jobID string) ([]imageAsset, error) {
	var out struct {
		Assets []imageAsset `json:"assets"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/jobs/"+jobID+"/assets", nil, "", &out); err != nil {
		return nil, fmt.Errorf("jobAssets: %w", err)
	}
	return out.Assets, nil
}

// errAssetGone is the ONE condition both image paths act on: the platform will not serve a usable
// picture for this id again, so the slot naming it is a dangling reference. Two shapes reach it and
// they look nothing alike on the wire:
//
//   - **404 not_found** — deleted. Definitive, and the only shape #53 handled.
//   - **200 with a retired status** — retired IN PLACE. `AssetStatus` is
//     `pending|preview_ready|ready|failed|archived`, and supersession leaves `archived` behind.
//     Their contract is explicit that "archived assets remain displayable to the owning tenant",
//     so the platform answers **200 and hands back working presigned URLs** to the picture nobody
//     wants any more. That is exactly why the mock mosaics kept serving cleanly after the fal flip:
//     nothing ever 404'd, so #53's healing never fired and the fill query never saw an empty slot.
//     `failed` is the same class — an asset row with no usable image behind it.
//
// `pending` and `preview_ready` are deliberately NOT gone. They are a live asset mid-flight, and
// reaping one would throw away a portrait that is about to arrive.
var errAssetGone = errors.New("asset will not be served again")

// assetURL mints a FRESH presigned URL for one asset and returns it for immediate use. Never stored.
// tier selects among the three always-available sizes: thumbnail 256, preview 768, final 1024.
func (c *imageClient) assetURL(ctx context.Context, assetID, tier string) (string, error) {
	var a imageAsset
	if err := c.do(ctx, http.MethodGet, "/v1/assets/"+assetID, nil, "", &a); err != nil {
		var apiErr *imageAPIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return "", fmt.Errorf("%w: %s is gone from the platform", errAssetGone, assetID)
		}
		return "", fmt.Errorf("assetURL: %w", err)
	}
	// Checked BEFORE the URLs are read, because a retired asset still carries perfectly valid ones.
	if a.Status == "archived" || a.Status == "failed" {
		return "", fmt.Errorf("%w: %s is %s", errAssetGone, assetID, a.Status)
	}
	switch tier {
	case "thumbnail":
		return a.ThumbnailURL, nil
	case "final":
		return a.FinalURL, nil
	default:
		return a.PreviewURL, nil
	}
}

// assetAlive asks the one question the trigger needs — will this id still serve a picture? — over
// the same single GET, rather than adding a second way to interrogate an asset.
func (c *imageClient) assetAlive(ctx context.Context, assetID string) error {
	_, err := c.assetURL(ctx, assetID, "")
	return err
}

// ── polling ─────────────────────────────────────────────────────────────────────────────────────

// pollBackoff is the platform's REQUIRED client behaviour, not a preference: rate limiting is per
// token and counts denied requests too, so naive fixed-interval polling pins your own window and
// extends your own outage.
//
//	start ~1s · multiply ~1.8× · cap ~15s · FULL jitter (sleep = random(0, computed))
//
// Full jitter rather than a fraction: several slots polling at once would otherwise converge into
// synchronised bursts against a 60/minute limit. Attempts are capped so a stuck job surfaces as a
// timeout instead of polling forever — there is no list-jobs endpoint, no cursor and no ETag, so
// every poll is a full request per job and the budget is real (~5 in-flight per token, which is why
// max_concurrent_jobs is also 5).
type pollBackoff struct {
	base, cap time.Duration
	maxTries  int
}

func defaultPollBackoff() pollBackoff {
	return pollBackoff{base: time.Second, cap: 15 * time.Second, maxTries: 40}
}

// wait sleeps before attempt n (0-based). retryAfter, when the platform supplied one, is
// authoritative and REPLACES the computed delay entirely rather than being added to it.
//
// Jitter comes from math/rand/v2's top-level source, which needs no seeding and carries no global
// mutable seed. It is not security-sensitive: the only job of the randomness is to stop concurrent
// pollers converging on the same instant.
func (b pollBackoff) wait(ctx context.Context, n int, retryAfter time.Duration) error {
	d := retryAfter
	if d <= 0 {
		computed := time.Duration(float64(b.base) * pow(1.8, n))
		if computed > b.cap {
			computed = b.cap
		}
		d = time.Duration(rand.Int64N(int64(computed) + 1)) // full jitter: random(0, computed)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func pow(base float64, n int) float64 {
	out := 1.0
	for range n {
		out *= base
	}
	return out
}

var errPollTimeout = errors.New("image job did not reach a terminal status within the attempt budget")

// awaitJob polls until the job is terminal, the attempt budget is spent, or the context ends. It
// stops the instant a terminal status is seen — never polls a completed job again.
func (c *imageClient) awaitJob(ctx context.Context, jobID string, b pollBackoff) (imageJob, error) {
	var retryAfter time.Duration
	for n := range b.maxTries {
		if err := b.wait(ctx, n, retryAfter); err != nil {
			return imageJob{}, err
		}
		retryAfter = 0
		j, err := c.getJob(ctx, jobID)
		if err != nil {
			var apiErr *imageAPIError
			if errors.As(err, &apiErr) && apiErr.retryable() {
				retryAfter = apiErr.RetryAfter // authoritative on rate_limit_exceeded; absent on concurrent_jobs_exceeded
				continue
			}
			return imageJob{}, err
		}
		if j.terminal() {
			return j, nil
		}
	}
	return imageJob{}, errPollTimeout
}
