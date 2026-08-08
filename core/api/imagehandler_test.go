package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fillPortraits is the whole loop: style, identity, generation with a pinned envelope, poll, and the
// identity→asset mapping persisted on OUR side, because the asset row does not carry it.
func TestFillPortraits_PersistsTheAssetTheAssetRowDoesNotCarry(t *testing.T) {
	pool := testPool(t)
	// t.Cleanup, not defer: Go runs defers BEFORE registered cleanups, so `defer pool.Close()`
	// alongside a t.Cleanup that deletes rows closes the pool first and the delete silently fails
	// against a dead connection. Registering the close FIRST makes LIFO run it LAST, after every
	// row-cleanup. (ledger_test.go documents the same ordering rule; two "Test World" rows leaked
	// into the shared directory before this was fixed.)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID, actorID := "5a5e0000-0000-0000-0000-0000000000f1", "5a5e0000-0000-0000-0000-0000000000a1"
	if _, err := pool.Exec(ctx,
		`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		 VALUES ($1,$2,'actor','Portrait Subject') ON CONFLICT DO NOTHING`, actorID, worldID); err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	// Clear at the START as well as the end. This suite shares one database by design, so a previous
	// run that died mid-test leaves rows behind, and a test that only tidies up on exit inherits them
	// and fails for a reason that has nothing to do with the code.
	clear := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1 AND entity_id<>$2`, worldID, actorID)
	}
	clear()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
	})

	f := newFakePlatform()
	c := testImageClient(t, f)

	res, err := fillPortraits(ctx, pool, c, worldID, 5)
	if err != nil {
		t.Fatalf("fillPortraits: %v", err)
	}
	if res.Requested != 1 || res.Completed != 1 {
		t.Fatalf("result = %+v, want one requested and one completed", res)
	}

	var assetID, identityID string
	var jobID, lastErr *string
	if err := pool.QueryRow(ctx,
		`SELECT asset_id, visual_identity_id, job_id, last_error FROM image_slot
		  WHERE world_id=$1 AND owner_kind='actor' AND owner_id=$2`, worldID, actorID).
		Scan(&assetID, &identityID, &jobID, &lastErr); err != nil {
		t.Fatalf("read slot: %v", err)
	}
	if assetID != "asset_cf63b1d2e6150906" {
		t.Fatalf("asset_id = %q — the mapping the platform cannot answer for us was not stored", assetID)
	}
	if identityID != "vi_c40c1fc21b057d27" {
		t.Fatalf("visual_identity_id = %q", identityID)
	}
	// job_id is cleared on a terminal status: a settled job must never be polled again.
	if jobID != nil {
		t.Fatalf("job_id = %v, want NULL once the job settled", *jobID)
	}
	if lastErr != nil {
		t.Fatalf("last_error = %v, want NULL on success", *lastErr)
	}

	// Re-running skips the filled slot rather than paying for a second job. Reuse would make a repeat
	// harmless and free, but "harmless" is not a reason to ask.
	before := f.generationCalls
	res2, err := fillPortraits(ctx, pool, c, worldID, 5)
	if err != nil {
		t.Fatalf("second fillPortraits: %v", err)
	}
	if res2.Requested != 0 || f.generationCalls != before {
		t.Fatalf("a filled slot was re-requested: %+v (generation calls %d→%d)", res2, before, f.generationCalls)
	}
}

// A failed job leaves the reason on the slot and clears job_id, so a blank portrait can explain
// itself and the next run does not poll a job that will never move.
func TestFillPortraits_FailureIsRecordedAndNotLeftInFlight(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID, actorID := "5a5e0000-0000-0000-0000-0000000000f2", "5a5e0000-0000-0000-0000-0000000000a2"
	if _, err := pool.Exec(ctx,
		`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		 VALUES ($1,$2,'actor','Doomed Subject') ON CONFLICT DO NOTHING`, actorID, worldID); err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	// Clear at the START as well as the end. This suite shares one database by design, so a previous
	// run that died mid-test leaves rows behind, and a test that only tidies up on exit inherits them
	// and fails for a reason that has nothing to do with the code.
	clear := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1 AND entity_id<>$2`, worldID, actorID)
	}
	clear()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
	})

	f := newFakePlatform()
	f.failJob = true
	c := testImageClient(t, f)

	res, err := fillPortraits(ctx, pool, c, worldID, 5)
	if err != nil {
		t.Fatalf("fillPortraits: %v", err)
	}
	if res.Failed != 1 {
		t.Fatalf("result = %+v, want one failure", res)
	}
	var assetID, jobID *string
	var lastErr string
	if err := pool.QueryRow(ctx,
		`SELECT asset_id, job_id, coalesce(last_error,'') FROM image_slot
		  WHERE world_id=$1 AND owner_kind='actor' AND owner_id=$2`, worldID, actorID).
		Scan(&assetID, &jobID, &lastErr); err != nil {
		t.Fatalf("read slot: %v", err)
	}
	if assetID != nil {
		t.Fatalf("asset_id = %v, want NULL after a failed job", *assetID)
	}
	if jobID != nil {
		t.Fatalf("job_id = %v, want NULL — a dead job must not be polled forever", *jobID)
	}
	if lastErr == "" {
		t.Fatal("last_error is empty; a blank portrait should be able to explain itself")
	}
}

// fn_image_ref is what the frontend will read. NULL is the ORDINARY state — "no picture yet" — and
// the reference never carries a presigned URL, only an id and a path back to this service.
func TestImageRef_NullUntilReadyThenIdAndPathNeverAUrl(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID, actorID := "5a5e0000-0000-0000-0000-0000000000f3", "5a5e0000-0000-0000-0000-0000000000a3"
	_, _ = pool.Exec(ctx, `DELETE FROM image_slot WHERE world_id=$1`, worldID) // residue-proof, see above
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID) })

	var raw *string
	if err := pool.QueryRow(ctx, `SELECT fn_image_ref($1,'actor',$2)::text`, worldID, actorID).Scan(&raw); err != nil {
		t.Fatalf("fn_image_ref: %v", err)
	}
	if raw != nil {
		t.Fatalf("image ref = %v, want NULL before anything is generated", *raw)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO image_slot (world_id, owner_kind, owner_id, asset_id) VALUES ($1,'actor',$2,'asset_cf63b1d2e6150906')`,
		worldID, actorID); err != nil {
		t.Fatalf("insert slot: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT fn_image_ref($1,'actor',$2)::text`, worldID, actorID).Scan(&raw); err != nil {
		t.Fatalf("fn_image_ref: %v", err)
	}
	if raw == nil {
		t.Fatal("image ref is NULL after an asset landed")
	}
	var ref struct {
		SchemaVersion string `json:"schema_version"`
		AssetID       string `json:"asset_id"`
		Path          string `json:"path"`
	}
	if err := json.Unmarshal([]byte(*raw), &ref); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ref.SchemaVersion != "image_ref/1" || ref.AssetID != "asset_cf63b1d2e6150906" {
		t.Fatalf("ref = %+v", ref)
	}
	if ref.Path != "/worlds/"+worldID+"/images/asset_cf63b1d2e6150906" {
		t.Fatalf("path = %q, want a path back at this service", ref.Path)
	}
	// The invariant that makes the reference storable: no presigned URL, ever.
	for _, forbidden := range []string{"X-Amz", "http://", "https://", "download_url", "expires"} {
		if strings.Contains(*raw, forbidden) {
			t.Fatalf("image ref carries %q — presigned URLs expire and must never be persisted:\n%s", forbidden, *raw)
		}
	}
}

// The read surface hands back a FRESH credential per request and refuses to be cached.
func TestImageHandler_RedirectsToAFreshUrlAndForbidsCaching(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })

	f := newFakePlatform()
	h := NewImageHandler(pool, testImageClient(t, f), true)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/worlds/5a5e0000-0000-0000-0000-0000000000f4/images/asset_cf63b1d2e6150906", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302: %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "X-Amz-Signature") {
		t.Fatalf("Location = %q, want a freshly presigned URL", loc)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store — the target is a short-lived credential", cc)
	}
}

// With no platform configured the surfaces answer plainly instead of erroring obscurely: the world
// runs without images, and that has to stay true.
func TestImageHandler_UnconfiguredPlatformIsAnOrdinaryAnswer(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	h := NewImageHandler(pool, nil, true)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/worlds/5a5e0000-0000-0000-0000-0000000000f5/images/asset_x", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("asset read status = %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/worlds/5a5e0000-0000-0000-0000-0000000000f5/images/portraits", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("portraits status = %d, want 503", rec.Code)
	}
}

// scene_current/2: participants carry `image`, null until a portrait exists. The version moved
// because the payload is additionalProperties:false and the frontend pins it exactly — an added
// field is a breaking change however additive it looks.
func TestSceneCurrentV2_ParticipantsCarryImageNullUntilReady(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()
	id := setupSceneWorld(t, ctx, pool)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, id.World) })

	decode := func() (string, map[string]json.RawMessage) {
		t.Helper()
		rec := sceneGet(t, NewSceneHandler(pool, true), id.World, id.Viewer)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		var v struct {
			SchemaVersion string `json:"schema_version"`
			Participants  []struct {
				ID    string          `json:"id"`
				Image json.RawMessage `json:"image"`
			} `json:"participants"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatalf("decode: %v", err)
		}
		byID := map[string]json.RawMessage{}
		for _, p := range v.Participants {
			byID[p.ID] = p.Image
		}
		return v.SchemaVersion, byID
	}

	sv, byID := decode()
	if sv != "scene_current/2" {
		t.Fatalf("schema_version = %q, want scene_current/2", sv)
	}
	img, ok := byID[id.Companion]
	if !ok {
		t.Fatal("the companion is not a participant; this proves nothing")
	}
	// The field must be PRESENT and null, not absent: the schema requires it.
	if string(img) != "null" {
		t.Fatalf("image = %s, want null before any portrait exists", img)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO image_slot (world_id, owner_kind, owner_id, asset_id) VALUES ($1,'actor',$2,'asset_cf63b1d2e6150906')`,
		id.World, id.Companion); err != nil {
		t.Fatalf("insert slot: %v", err)
	}
	_, byID = decode()
	var ref struct {
		SchemaVersion string `json:"schema_version"`
		AssetID       string `json:"asset_id"`
		Path          string `json:"path"`
	}
	if err := json.Unmarshal(byID[id.Companion], &ref); err != nil {
		t.Fatalf("decode image ref: %v (raw %s)", err, byID[id.Companion])
	}
	if ref.SchemaVersion != "image_ref/1" || ref.AssetID != "asset_cf63b1d2e6150906" {
		t.Fatalf("ref = %+v", ref)
	}
	if !strings.HasPrefix(ref.Path, "/worlds/"+id.World+"/images/") {
		t.Fatalf("path = %q, want a path back at this service", ref.Path)
	}
	// The invariant that lets the payload be cached at all.
	if strings.Contains(string(byID[id.Companion]), "X-Amz") {
		t.Fatalf("participant image carried a presigned URL: %s", byID[id.Companion])
	}
}

// A busy room must not cost one query per participant for a field that is usually null.
func TestImageRefsFor_ResolvesTheWholeRosterInOneQuery(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID := "5a5e0000-0000-0000-0000-0000000000f6"
	a1, a2 := "5a5e0000-0000-0000-0000-0000000000b1", "5a5e0000-0000-0000-0000-0000000000b2"
	_, _ = pool.Exec(ctx, `DELETE FROM image_slot WHERE world_id=$1`, worldID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID) })
	if _, err := pool.Exec(ctx,
		`INSERT INTO image_slot (world_id, owner_kind, owner_id, asset_id) VALUES ($1,'actor',$2,'asset_one')`,
		worldID, a1); err != nil {
		t.Fatalf("insert: %v", err)
	}

	refs, err := imageRefsFor(ctx, pool, worldID, "actor", []string{a1, a2})
	if err != nil {
		t.Fatalf("imageRefsFor: %v", err)
	}
	if _, ok := refs[a1]; !ok {
		t.Fatal("the actor with an asset is missing from the batch")
	}
	// An entity with no slot is simply absent; a missing key marshals as null.
	if _, ok := refs[a2]; ok {
		t.Fatal("an entity with no picture must be absent, not an empty reference")
	}
	if empty, err := imageRefsFor(ctx, pool, worldID, "actor", nil); err != nil || len(empty) != 0 {
		t.Fatalf("empty roster: %v / %v", empty, err)
	}
}

// A slot can outlive the asset it names — the platform's storage is a separate system, and a reset
// on its side is exactly how that happens. When the platform says the asset is gone, the reference
// must go with it, because nothing else in the system can notice: the projection read never calls
// the platform, and the fill query skips any slot that already names an asset. Left in place, one
// vanished asset is a permanently broken picture in the frontend that no re-trigger can repair.
func TestImageHandler_AVanishedAssetIsForgottenSoTheWorldCanHeal(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID, actorID := "5a5e0000-0000-0000-0000-0000000000f7", "5a5e0000-0000-0000-0000-0000000000a7"
	const gone = "asset_vanished_from_the_platform"
	_, _ = pool.Exec(ctx, `DELETE FROM image_slot WHERE world_id=$1`, worldID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_slot (world_id, owner_kind, owner_id, visual_identity_id, asset_id)
		VALUES ($1::uuid, 'actor', $2::uuid, 'vi_kept', $3)`, worldID, actorID, gone); err != nil {
		t.Fatalf("seed slot: %v", err)
	}

	f := newFakePlatform()
	f.forgottenAssets = map[string]bool{gone: true}
	h := NewImageHandler(pool, testImageClient(t, f), true)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds/"+worldID+"/images/"+gone, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 — the picture really is gone", rec.Code)
	}

	var assetID *string
	var identity string
	if err := pool.QueryRow(ctx,
		`SELECT asset_id, visual_identity_id FROM image_slot WHERE world_id=$1::uuid AND owner_id=$2::uuid`,
		worldID, actorID).Scan(&assetID, &identity); err != nil {
		t.Fatalf("read slot: %v", err)
	}
	if assetID != nil {
		t.Fatalf("asset_id = %q, want NULL — the slot still points at an asset the platform lost, so "+
			"the projection keeps reporting a picture the browser cannot fetch and no trigger can fix it", *assetID)
	}
	// The identity is not the asset. Keeping it is what makes the regenerated portrait the same
	// character rather than a new stranger.
	if identity != "vi_kept" {
		t.Fatalf("visual_identity_id = %q, want it kept across the loss", identity)
	}
}

// The mirror image, and the reason this cannot simply clear on any error: a platform having a bad
// minute must never cost the world a good portrait.
func TestImageHandler_ATransientFailureKeepsTheReference(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID, actorID := "5a5e0000-0000-0000-0000-0000000000f8", "5a5e0000-0000-0000-0000-0000000000a8"
	const good = "asset_perfectly_fine"
	_, _ = pool.Exec(ctx, `DELETE FROM image_slot WHERE world_id=$1`, worldID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_slot (world_id, owner_kind, owner_id, asset_id)
		VALUES ($1::uuid, 'actor', $2::uuid, $3)`, worldID, actorID, good); err != nil {
		t.Fatalf("seed slot: %v", err)
	}

	f := newFakePlatform()
	f.assetFailStatus = http.StatusInternalServerError
	h := NewImageHandler(pool, testImageClient(t, f), true)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/worlds/"+worldID+"/images/"+good, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for this read", rec.Code)
	}

	var assetID *string
	if err := pool.QueryRow(ctx,
		`SELECT asset_id FROM image_slot WHERE world_id=$1::uuid AND owner_id=$2::uuid`,
		worldID, actorID).Scan(&assetID); err != nil {
		t.Fatalf("read slot: %v", err)
	}
	if assetID == nil || *assetID != good {
		t.Fatalf("asset_id was discarded over a 500 — a blip must not cost the world a good portrait")
	}
}
