package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"testing"
	"time"
)

// A stand-in for the Image Platform, built from the bodies in their
// docs/api/integration-quickstart.md — every response below is copied from a sequence they executed
// against a real stack (API + worker + Postgres + Redis + MinIO), not invented here.
//
// Their real stack cannot be part of this suite: it needs its own compose, a host port, and an
// identity-capable provider, and it is a separate repository. What CAN be pinned here is that this
// client obeys the contract that document states — the mandatory idempotency header, the pinned
// envelope, the two different 429s, the omitted-not-empty asset list. Those are the things a
// misreading of the doc would get wrong, and they are exactly what an end-to-end run would be too
// slow and too coarse to catch anyway.
type fakePlatform struct {
	mu sync.Mutex

	styles []styleProfile
	// stylesListStatus, when non-zero, refuses GET /v1/styles with that status — the shape the live
	// platform had for weeks while the token was missing styles:read.
	stylesListStatus int
	bodyForKey       map[string]string // Idempotency-Key → the exact body first seen under it
	jobStatus        []string          // consumed one per GET /v1/jobs/{id}
	statusIdx        int

	seenIdempotencyKey string
	generationCalls    int
	jobCalls           int
	requestOrder       []string

	// identity and bootstrap
	identityAnchors             map[string][]string
	bootstrapFailFor            map[string]bool
	upsertAppearanceByOwner     map[string]string
	lastBootstrapDescription    string
	lastBootstrapIdempotencyKey string

	// scripted failures
	rateLimitOnce   bool
	retryAfterSecs  string
	concurrentOnce  bool
	failJob         bool
	omitFinalAssets bool
	// the platform no longer has these assets (its storage has its own lifecycle, and a real reset
	// during integration is exactly how a slot outlives the asset it names)
	forgottenAssets map[string]bool
	// RETIRED IN PLACE, and the shape that fooled #53: the platform answers 200 for these, with a
	// retired status and perfectly working download URLs, because "archived assets remain
	// displayable to the owning tenant". Supersession leaves this behind, so it is what every mock
	// asset became the moment the platform flipped to real art.
	archivedAssets map[string]bool
	// a bad minute rather than a verdict: any non-zero status is returned for every asset read
	assetFailStatus int

	// the scene_capable route
	sceneCalls              int
	lastSceneDescription    string
	lastScenePinnedProvider *string
}

func newFakePlatform() *fakePlatform {
	return &fakePlatform{bodyForKey: map[string]string{}, jobStatus: []string{"completed"}, identityAnchors: map[string][]string{}, bootstrapFailFor: map[string]bool{}, upsertAppearanceByOwner: map[string]string{}}
}

func (f *fakePlatform) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/styles", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") == "" {
			writeErr(w, 403, "forbidden", "missing token")
			return
		}
		if r.Method == http.MethodGet {
			if f.stylesListStatus != 0 {
				writeErr(w, f.stylesListStatus, "forbidden", "token missing required scope")
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"styles": f.styles})
			return
		}
		s := styleProfile{ID: "sty_17c36e08344742e1", Name: "dreamchat-default"}
		f.styles = append(f.styles, s)
		_ = json.NewEncoder(w).Encode(s)
	})

	mux.HandleFunc("/v1/characters/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/characters/"), "/")
		if len(parts) < 2 || parts[1] != "visual-identity" {
			writeErr(w, 404, "not_found", "route not found")
			return
		}
		ownerID := parts[0]

		f.mu.Lock()
		anchors, ok := f.identityAnchors[ownerID]
		if !ok {
			anchors = []string{"asset_anchor_seeded"}
		}
		bootstrapFails := f.bootstrapFailFor[ownerID]
		f.mu.Unlock()

		if len(parts) == 3 && parts[2] == "bootstrap-anchor" {
			if r.Method != http.MethodPost {
				writeErr(w, 405, "method_not_allowed", "use POST")
				return
			}
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				writeErr(w, 422, "invalid_request", "Idempotency-Key is required")
				return
			}
			var body struct {
				WorldID     string         `json:"world_id"`
				StyleID     string         `json:"style_profile_id"`
				Description string         `json:"description"`
				Governance  map[string]any `json:"governance"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			switch {
			case body.WorldID == "":
				writeErr(w, 400, "invalid_request", "world_id is required")
				return
			case body.StyleID == "":
				writeErr(w, 400, "invalid_request", "style_profile_id is required")
				return
			case body.Description == "":
				writeErr(w, 400, "invalid_request", "description is required")
				return
			case body.Governance == nil:
				writeErr(w, 400, "invalid_request", "governance is required")
				return
			}
			f.mu.Lock()
			f.requestOrder = append(f.requestOrder, "bootstrap")
			f.lastBootstrapDescription = body.Description
			f.lastBootstrapIdempotencyKey = key
			f.mu.Unlock()
			if bootstrapFails {
				writeErr(w, 500, "internal_error", "bootstrap failed")
				return
			}
			if len(anchors) > 0 {
				_ = json.NewEncoder(w).Encode(map[string]any{"id": "vi_c40c1fc21b057d27", "anchor_asset_ids": anchors})
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "job_anchor_041c843f24940ad4", "status": "queued"})
			return
		}

		if r.Method == http.MethodGet {
			// The real platform keys an identity on (tenant, world, owner_type, owner) and answers
			// 400 invalid_request without world_id. The fake enforces it for one reason: it did not,
			// and a getIdentity that omitted the parameter passed every test here while failing
			// every portrait in production with "world_id query parameter is required" recorded as
			// the slot's error. A stand-in laxer than the service it stands in for proves nothing.
			if r.URL.Query().Get("world_id") == "" {
				writeErr(w, 400, "invalid_request", "world_id query parameter is required")
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":               "vi_c40c1fc21b057d27",
				"anchor_asset_ids": anchors,
			})
			return
		}

		if r.Method == http.MethodPost {
			var body struct {
				Traits map[string]string `json:"canonical_visual_traits"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Traits != nil {
				f.mu.Lock()
				f.upsertAppearanceByOwner[ownerID] = body.Traits["appearance"]
				f.mu.Unlock()
			}
		}

		// visual-identity upsert; keyed on (tenant, world, owner_type, owner_id) so replay is safe
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "vi_c40c1fc21b057d27", "current_version": 1})
	})

	mux.HandleFunc("/v1/generations", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.generationCalls++
		f.requestOrder = append(f.requestOrder, "generation")
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			writeErr(w, 422, "invalid_request", "Idempotency-Key is required")
			return
		}
		f.seenIdempotencyKey = key
		b, _ := io.ReadAll(r.Body)
		if prev, ok := f.bodyForKey[key]; ok && prev != string(b) {
			// THE TRAP: the key is bound to a hash of the whole body, and issued_at moves.
			writeErr(w, 409, "idempotency_conflict", "idempotency key reused with a different body or endpoint")
			return
		}
		f.bodyForKey[key] = string(b)
		w.WriteHeader(202)
		_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "job_041c843f24940ad4", "status": "queued"})
	})

	mux.HandleFunc("/v1/jobs/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/assets") {
			asset := map[string]any{
				"id": "asset_cf63b1d2e6150906", "status": "ready", "variant_key": "default",
				"visual_identity_id":     nil, // NULL on this path — the storage requirement
				"thumbnail_download_url": "http://minio/thumb.png?X-Amz-Signature=a",
				"preview_download_url":   "http://minio/low.png?X-Amz-Signature=b",
				"final_download_url":     "http://minio/high.png?X-Amz-Signature=c",
				"url_expires_at":         time.Now().Add(15 * time.Minute).Format(time.RFC3339),
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"assets": []any{asset}})
			return
		}
		f.jobCalls++
		if f.rateLimitOnce {
			f.rateLimitOnce = false
			if f.retryAfterSecs != "" {
				w.Header().Set("Retry-After", f.retryAfterSecs)
			}
			writeErr(w, 429, "rate_limit_exceeded", "slow down")
			return
		}
		if f.concurrentOnce {
			f.concurrentOnce = false
			// deliberately NO Retry-After on this one — it clears on a terminal state, not a clock
			writeErr(w, 429, "concurrent_jobs_exceeded", "too many live jobs")
			return
		}
		status := f.jobStatus[min(f.statusIdx, len(f.jobStatus)-1)]
		f.statusIdx++
		body := map[string]any{"id": "job_041c843f24940ad4", "status": status}
		if f.failJob {
			body["status"] = "failed"
			body["error_code"] = "missing_reference_assets"
			body["error_message"] = "identity has no anchor assets"
			body["retryable"] = false
		} else if status == "completed" && !f.omitFinalAssets {
			body["final_asset_ids"] = []string{"asset_cf63b1d2e6150906"}
		}
		_ = json.NewEncoder(w).Encode(body)
	})

	// The scene_capable route: POST /v1/artifacts/{id}/generate. Their contract requires world_id,
	// style_profile_id and description in the body, and a description is the whole point here — a
	// backdrop with nothing authored behind it would be invented fiction, so an empty one is an
	// error rather than a silently blank picture.
	mux.HandleFunc("/v1/artifacts/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.sceneCalls++
		f.mu.Unlock()
		var body struct {
			WorldID     string         `json:"world_id"`
			StyleID     string         `json:"style_profile_id"`
			Description string         `json:"description"`
			Governance  map[string]any `json:"governance"`
			Provider    *string        `json:"provider_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case body.WorldID == "":
			writeErr(w, 400, "invalid_request", "world_id is required")
			return
		case body.StyleID == "":
			writeErr(w, 400, "invalid_request", "style_profile_id is required")
			return
		case body.Description == "":
			writeErr(w, 400, "invalid_request", "description is required")
			return
		case body.Governance == nil:
			writeErr(w, 400, "invalid_request", "governance is required")
			return
		}
		f.mu.Lock()
		f.lastSceneDescription = body.Description
		f.lastScenePinnedProvider = body.Provider
		f.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "job_041c843f24940ad4", "status": "queued"})
	})

	mux.HandleFunc("/v1/assets/", func(w http.ResponseWriter, r *http.Request) {
		id := path.Base(r.URL.Path)
		if f.assetFailStatus != 0 {
			writeErr(w, f.assetFailStatus, "internal_error", "the platform is having a bad minute")
			return
		}
		if f.forgottenAssets[id] {
			writeErr(w, http.StatusNotFound, "not_found", "asset not found")
			return
		}
		if f.archivedAssets[id] {
			// 200, a retired status, and URLs that still work. Nothing here looks like a failure.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": id, "status": "archived",
				"thumbnail_download_url": "http://minio/old-mosaic-thumb.png?X-Amz-Signature=a",
				"preview_download_url":   "http://minio/old-mosaic-low.png?X-Amz-Signature=b",
				"final_download_url":     "http://minio/old-mosaic-high.png?X-Amz-Signature=c",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": id, "status": "ready",
			"thumbnail_download_url": "http://minio/thumb.png?X-Amz-Signature=a",
			"preview_download_url":   "http://minio/low.png?X-Amz-Signature=b",
			"final_download_url":     "http://minio/high.png?X-Amz-Signature=c",
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": msg, "request_id": "req_test"})
}

func testImageClient(t *testing.T, f *fakePlatform) *imageClient {
	t.Helper()
	srv := f.server(t)
	return &imageClient{baseURL: srv.URL, token: "dci_dev_test_secret", http: srv.Client()}
}

// fastBackoff keeps the timing tests instant while still exercising the real wait path.
func fastBackoff() pollBackoff {
	return pollBackoff{base: time.Millisecond, cap: 2 * time.Millisecond, maxTries: 20}
}

// The whole documented sequence: style → identity → generation → poll → assets.
func TestImageClient_RunsTheVerifiedSequence(t *testing.T) {
	f := newFakePlatform()
	c := testImageClient(t, f)
	ctx := context.Background()

	styleID, err := c.ensureStyle(ctx, "dreamchat-default")
	if err != nil || styleID == "" {
		t.Fatalf("ensureStyle: %v / %q", err, styleID)
	}
	identityID, err := c.upsertIdentity(ctx, "character", "2ac70000-0000-0000-0000-0000000000a1", "w1", "Kade", styleID, nil)
	if err != nil || identityID == "" {
		t.Fatalf("upsertIdentity: %v / %q", err, identityID)
	}
	jobID, err := c.requestGeneration(ctx, identityID, "", "portrait-w1-kade", newGovEnvelope(time.Now(), "character_portrait"))
	if err != nil || jobID == "" {
		t.Fatalf("requestGeneration: %v / %q", err, jobID)
	}
	job, err := c.awaitJob(ctx, jobID, fastBackoff())
	if err != nil {
		t.Fatalf("awaitJob: %v", err)
	}
	if job.Status != "completed" || len(job.FinalAssetIDs) != 1 {
		t.Fatalf("job = %+v, want completed with one final asset", job)
	}
	assets, err := c.jobAssets(ctx, jobID)
	if err != nil || len(assets) != 1 {
		t.Fatalf("jobAssets: %v / %d", err, len(assets))
	}
	// Their storage requirement, reproduced: the ASSET does not carry the identity, the JOB does.
	if assets[0].VisualIdentityID != nil {
		t.Fatalf("asset carried visual_identity_id = %v; the mapping must be stored on our side", *assets[0].VisualIdentityID)
	}
}

// The idempotency header is mandatory and canonical — neither header nor body key is a 422.
func TestImageClient_SendsTheIdempotencyKey(t *testing.T) {
	f := newFakePlatform()
	c := testImageClient(t, f)

	if _, err := c.requestGeneration(context.Background(), "vi_x", "", "portrait-w1-kade",
		newGovEnvelope(time.Now(), "character_portrait")); err != nil {
		t.Fatalf("requestGeneration: %v", err)
	}
	if f.seenIdempotencyKey != "portrait-w1-kade" {
		t.Fatalf("Idempotency-Key = %q, want the pinned key", f.seenIdempotencyKey)
	}
}

// THE TRAP, pinned. Their key hashes the whole body and the envelope carries issued_at, so a retry
// that rebuilds the envelope is a different body under the same key: 409. Replaying the PINNED
// envelope must not be.
func TestImageClient_PinnedEnvelopeSurvivesRetryButAFreshOneDoesNot(t *testing.T) {
	f := newFakePlatform()
	c := testImageClient(t, f)
	ctx := context.Background()
	const key = "portrait-w1-kade"

	pinned := newGovEnvelope(time.Date(2026, 8, 8, 16, 33, 23, 0, time.UTC), "character_portrait")
	if _, err := c.requestGeneration(ctx, "vi_x", "", key, pinned); err != nil {
		t.Fatalf("first request: %v", err)
	}
	// Replaying the SAME pinned envelope under the same key is the supported retry.
	if _, err := c.requestGeneration(ctx, "vi_x", "", key, pinned); err != nil {
		t.Fatalf("replaying the pinned envelope must be accepted, got: %v", err)
	}
	// Rebuilding the envelope moves issued_at, which is the mistake their verification run made.
	fresh := newGovEnvelope(time.Date(2026, 8, 8, 16, 40, 0, 0, time.UTC), "character_portrait")
	_, err := c.requestGeneration(ctx, "vi_x", "", key, fresh)
	var apiErr *imageAPIError
	if !errors.As(err, &apiErr) || apiErr.Code != "idempotency_conflict" {
		t.Fatalf("a rebuilt envelope under the same key must conflict; got %v", err)
	}
}

// The seven-field envelope, with the signature sentinel. An empty signature is a hard 422 whatever
// the enforcement mode, so the sentinel is load-bearing rather than decorative.
func TestGovEnvelope_CarriesAllSevenFields(t *testing.T) {
	env := newGovEnvelope(time.Date(2026, 8, 8, 16, 33, 23, 0, time.UTC), "character_portrait")
	b, _ := json.Marshal(env)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, f := range []string{"schema_version", "classification_id", "visibility", "content_class", "authorized_by", "issued_at", "signature"} {
		if v, ok := m[f]; !ok || v == "" {
			t.Fatalf("envelope field %q missing or empty: %s", f, b)
		}
	}
	if m["signature"] != "stub-unsigned-v1" {
		t.Fatalf("signature = %v, want the greppable sentinel", m["signature"])
	}
	if m["issued_at"] != "2026-08-08T16:33:23Z" {
		t.Fatalf("issued_at = %v, want the RFC3339 timestamp it was pinned with", m["issued_at"])
	}
}

// rate_limit_exceeded is retried and Retry-After is honoured as authoritative.
func TestImageClient_HonoursRetryAfterOnRateLimit(t *testing.T) {
	f := newFakePlatform()
	f.rateLimitOnce, f.retryAfterSecs = true, "0"
	c := testImageClient(t, f)

	job, err := c.awaitJob(context.Background(), "job_1", fastBackoff())
	if err != nil {
		t.Fatalf("awaitJob should ride out a 429: %v", err)
	}
	if job.Status != "completed" {
		t.Fatalf("status = %q", job.Status)
	}
	if f.jobCalls < 2 {
		t.Fatalf("jobCalls = %d, want the denied call plus a retry", f.jobCalls)
	}
}

// concurrent_jobs_exceeded is the OTHER 429: no Retry-After, because it clears when a job reaches a
// terminal state rather than on a clock. It must still be retried, not treated as fatal.
func TestImageClient_RetriesConcurrencyLimitWithoutRetryAfter(t *testing.T) {
	f := newFakePlatform()
	f.concurrentOnce = true
	c := testImageClient(t, f)

	if _, err := c.awaitJob(context.Background(), "job_1", fastBackoff()); err != nil {
		t.Fatalf("awaitJob should ride out concurrent_jobs_exceeded: %v", err)
	}
}

// A non-retryable error must surface immediately instead of burning the attempt budget.
func TestImageAPIError_NonRetryableCodesDoNotSpin(t *testing.T) {
	for _, tc := range []struct {
		code string
		want bool
	}{
		{"rate_limit_exceeded", true},
		{"concurrent_jobs_exceeded", true},
		{"route_capability_mismatch", false},
		{"invalid_request", false},
		{"budget_exceeded", false},
		{"idempotency_conflict", false},
	} {
		e := &imageAPIError{Status: 422, Code: tc.code}
		if got := e.retryable(); got != tc.want {
			t.Fatalf("%s retryable = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// A failed job is terminal: read the reason, stop polling, do not treat it as pending.
func TestImageClient_FailedJobIsTerminal(t *testing.T) {
	f := newFakePlatform()
	f.failJob = true
	c := testImageClient(t, f)

	job, err := c.awaitJob(context.Background(), "job_1", fastBackoff())
	if err != nil {
		t.Fatalf("awaitJob: %v", err)
	}
	if job.Status != "failed" || job.ErrorCode != "missing_reference_assets" {
		t.Fatalf("job = %+v, want a terminal failure carrying its reason", job)
	}
	if len(job.FinalAssetIDs) != 0 {
		t.Fatal("a failed job must not carry assets")
	}
}

// Backoff is bounded and jittered, and full jitter means a delay may legitimately be near zero —
// what must hold is that it never exceeds the cap, or concurrent pollers would drift unbounded.
func TestPollBackoff_IsBoundedAndJittered(t *testing.T) {
	b := pollBackoff{base: 10 * time.Millisecond, cap: 40 * time.Millisecond, maxTries: 5}
	ctx := context.Background()
	for n := range 8 {
		start := time.Now()
		if err := b.wait(ctx, n, 0); err != nil {
			t.Fatalf("wait: %v", err)
		}
		if elapsed := time.Since(start); elapsed > b.cap+50*time.Millisecond {
			t.Fatalf("attempt %d slept %v, past the %v cap", n, elapsed, b.cap)
		}
	}
	// An explicit Retry-After replaces the computed delay rather than adding to it.
	start := time.Now()
	if err := b.wait(ctx, 7, time.Millisecond); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Retry-After was not authoritative: slept %v", elapsed)
	}
}

// No base URL or no token means no client at all, and that is an ordinary state: the world has run
// without an image platform for its entire life and must keep doing so.
func TestNewImageClientFromEnv_AbsentConfigYieldsNoClient(t *testing.T) {
	t.Setenv(imageBaseURLEnv, "")
	t.Setenv(imageTokenEnv, "")
	if c := newImageClientFromEnv(); c != nil {
		t.Fatal("a client was built with no configuration")
	}
	t.Setenv(imageBaseURLEnv, "http://localhost:8088")
	if c := newImageClientFromEnv(); c != nil {
		t.Fatal("a client was built with a base URL but no token")
	}
	t.Setenv(imageTokenEnv, "dci_dev_x_y")
	if c := newImageClientFromEnv(); c == nil {
		t.Fatal("a fully configured client was not built")
	}
}

// A refused style list must not read as "this style does not exist yet".
//
// It did. ensureStyle swallowed the list error, so a token missing styles:read meant every call
// 403'd and then created another profile — twenty-five identical "dreamchat-default" rows in
// production. The clutter was the harmless part: the artifact reuse key folds style_profile_id, so a
// fresh id per call meant the cache could never hit and every regeneration was billed in full.
func TestEnsureStyle_ARefusedListIsAnErrorNotAnEmptyList(t *testing.T) {
	f := newFakePlatform()
	f.stylesListStatus = 403
	c := testImageClient(t, f)

	if _, err := c.ensureStyle(context.Background(), "dreamchat-default"); err == nil {
		t.Fatal("a 403 on the style list must fail, not fall through to creating another profile")
	}

	f.mu.Lock()
	created := len(f.styles)
	f.mu.Unlock()
	if created != 0 {
		t.Fatalf("nothing may be created when the list could not be read, got %d style(s)", created)
	}
}
