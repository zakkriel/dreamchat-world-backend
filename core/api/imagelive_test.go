package main

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// The LIVE handshake against a running Image Platform. Gated on DREAMCHAT_IMAGE_LIVE so it never
// runs in CI or an ordinary `go test ./...` — it needs their stack up, a real token, and an
// identity-capable provider, none of which exist in this repo's test environment.
//
//	DREAMCHAT_IMAGE_LIVE=1 \
//	DREAMCHAT_IMAGE_BASE_URL=http://localhost:8081 \
//	DREAMCHAT_IMAGE_API_TOKEN=... \
//	DATABASE_URL=... go test -run TestLive_ImageHandshake -v .
//
// It exists as a test rather than a script so the handshake is repeatable by anyone, uses the REAL
// client rather than curl, and leaves its evidence in the same place every other verification lives.
func TestLive_ImageHandshake(t *testing.T) {
	if os.Getenv("DREAMCHAT_IMAGE_LIVE") == "" {
		t.Skip("DREAMCHAT_IMAGE_LIVE unset — the live handshake needs a running Image Platform")
	}
	client := newImageClientFromEnv()
	if client == nil {
		t.Fatal("DREAMCHAT_IMAGE_BASE_URL / DREAMCHAT_IMAGE_API_TOKEN not set")
	}
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	// A private world, so the handshake never disturbs the world the frontend is driving.
	const worldID = "11ace000-0000-0000-0000-000000000c01"
	const actorID = "11ace000-0000-0000-0000-0000000000a1"
	_, _ = pool.Exec(ctx, `DELETE FROM image_slot WHERE world_id=$1`, worldID)
	if _, err := pool.Exec(ctx,
		`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		 VALUES ($1,$2,'actor','Kade') ON CONFLICT DO NOTHING`, actorID, worldID); err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
	})

	res, err := fillPortraits(ctx, pool, client, worldID, 1)
	if err != nil {
		t.Fatalf("fillPortraits: %v", err)
	}
	fmt.Printf("LIVE fill result: %+v\n", res)

	var assetID, identityID string
	var jobID, lastErr *string
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(asset_id,''), coalesce(visual_identity_id,''), job_id, last_error
		   FROM image_slot WHERE world_id=$1 AND owner_kind='actor' AND owner_id=$2`,
		worldID, actorID).Scan(&assetID, &identityID, &jobID, &lastErr); err != nil {
		t.Fatalf("read slot: %v", err)
	}
	if lastErr != nil {
		t.Fatalf("slot recorded an error: %s", *lastErr)
	}
	if identityID == "" || assetID == "" {
		t.Fatalf("identity=%q asset=%q — the mapping the platform cannot answer for us was not stored", identityID, assetID)
	}
	if jobID != nil {
		t.Fatalf("job_id = %v, want NULL once the job settled", *jobID)
	}
	fmt.Printf("LIVE identity_id=%s\nLIVE asset_id=%s\n", identityID, assetID)

	// A fresh presigned URL, minted on demand and never persisted.
	url, err := client.assetURL(ctx, assetID, "final")
	if err != nil || url == "" {
		t.Fatalf("assetURL: %v / %q", err, url)
	}
	fmt.Printf("LIVE presigned URL host+path: %.80s...\n", url)

	// Reuse: asking again must return the SAME asset, at zero cost, without a second render.
	before := assetID
	if _, err := pool.Exec(ctx, `UPDATE image_slot SET asset_id=NULL WHERE world_id=$1`, worldID); err != nil {
		t.Fatalf("reset slot: %v", err)
	}
	if _, err := fillPortraits(ctx, pool, client, worldID, 1); err != nil {
		t.Fatalf("second fillPortraits: %v", err)
	}
	var again string
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(asset_id,'') FROM image_slot WHERE world_id=$1 AND owner_kind='actor' AND owner_id=$2`,
		worldID, actorID).Scan(&again); err != nil {
		t.Fatalf("read slot again: %v", err)
	}
	if again != before {
		t.Fatalf("reuse produced a different asset: %s then %s — a portrait must not be re-rendered", before, again)
	}
	fmt.Printf("LIVE reuse returned the same asset: %s\n", again)
}

// The LIVE verification for the fal flip: run the real trigger against the real play world, then
// fetch a portrait back through THIS service's own endpoint and prove the bytes are real art rather
// than the mock colour grid. Gated the same way as the handshake above, and it SPENDS MONEY — every
// role whose mock asset was archived is regenerated on fal.
//
//	DREAMCHAT_IMAGE_LIVE=1 DREAMCHAT_IMAGE_PORTRAIT_REFRESH=1 \
//	DREAMCHAT_IMAGE_BASE_URL=http://localhost:8081 DREAMCHAT_IMAGE_API_TOKEN=... \
//	DATABASE_URL=... go test -run TestLive_PortraitsAreRealArt -timeout 20m -v .
//
// It is a test and not a curl script for the reason the handshake is: it drives the REAL
// fillPortraits and the REAL imageHandler.ServeHTTP, so what it proves is what a browser gets.
func TestLive_PortraitsAreRealArt(t *testing.T) {
	if os.Getenv("DREAMCHAT_IMAGE_LIVE") == "" || os.Getenv("DREAMCHAT_IMAGE_PORTRAIT_REFRESH") == "" {
		t.Skip("needs DREAMCHAT_IMAGE_LIVE and DREAMCHAT_IMAGE_PORTRAIT_REFRESH — this one spends money")
	}
	client := newImageClientFromEnv()
	if client == nil {
		t.Fatal("DREAMCHAT_IMAGE_BASE_URL / DREAMCHAT_IMAGE_API_TOKEN not set")
	}
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	const worldID = "22222222-2222-2222-2222-222222222222" // The Drowned Lantern, the world anyone plays

	res, err := fillPortraits(ctx, pool, client, worldID, imageBatchLimit)
	if err != nil {
		t.Fatalf("fillPortraits: %v", err)
	}
	fmt.Printf("LIVE portraits: %+v\n", res)
	if res.Reclaimed == 0 && res.Requested == 0 {
		t.Fatalf("nothing reclaimed and nothing requested — the trigger still cannot see stale slots")
	}

	rows, err := pool.Query(ctx, `
		SELECT owner_id::text, coalesce(asset_id,''), coalesce(last_error,'')
		  FROM image_slot WHERE world_id=$1 AND owner_kind='actor' ORDER BY owner_id`, worldID)
	if err != nil {
		t.Fatalf("read slots: %v", err)
	}
	type slot struct{ owner, asset, lastErr string }
	var slots []slot
	for rows.Next() {
		var s slot
		if err := rows.Scan(&s.owner, &s.asset, &s.lastErr); err != nil {
			rows.Close()
			t.Fatalf("scan: %v", err)
		}
		slots = append(slots, s)
	}
	rows.Close()

	h := NewImageHandler(pool, client, true)
	checked := 0
	for _, s := range slots {
		if s.asset == "" {
			t.Errorf("slot %s has no asset (last_error=%q)", s.owner, s.lastErr)
			continue
		}
		// Provenance, straight from the platform: the mock assets were provider_id=mock,
		// model_id=pm_mock_v1. Anything still carrying those is a mosaic by definition.
		var a struct {
			Status     string `json:"status"`
			ProviderID string `json:"provider_id"`
			ModelID    string `json:"model_id"`
		}
		if err := client.do(ctx, http.MethodGet, "/v1/assets/"+s.asset, nil, "", &a); err != nil {
			t.Errorf("asset %s: %v", s.asset, err)
			continue
		}
		if a.Status == "archived" || a.Status == "failed" {
			t.Errorf("slot %s still names a %s asset %s", s.owner, a.Status, s.asset)
		}
		if a.ProviderID == "mock" || a.ModelID == "pm_mock_v1" {
			t.Errorf("slot %s is still mock art: asset=%s provider=%s model=%s",
				s.owner, s.asset, a.ProviderID, a.ModelID)
		}

		// Now the bytes, through OUR endpoint, following the redirect exactly as a browser does.
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/images/"+s.asset+"?tier=preview", nil))
		if rec.Code != http.StatusFound {
			t.Errorf("fetch %s: status %d, want 302", s.asset, rec.Code)
			continue
		}
		resp, err := http.Get(rec.Header().Get("Location"))
		if err != nil {
			t.Errorf("follow redirect for %s: %v", s.asset, err)
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("decode %s: %v (%d bytes)", s.asset, err, len(raw))
			continue
		}
		// A generated portrait has thousands of distinct colours; the mock grid has a handful of
		// flat blocks. This is the difference the founder is looking at, measured rather than
		// asserted from provenance alone.
		colours := map[uint32]struct{}{}
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y += 2 {
			for x := b.Min.X; x < b.Max.X; x += 2 {
				r, g, bl, _ := img.At(x, y).RGBA()
				colours[r>>8<<16|g>>8<<8|bl>>8] = struct{}{}
			}
		}
		fmt.Printf("LIVE %s asset=%s provider=%s model=%s status=%s %dx%d %d bytes %d distinct colours\n",
			s.owner, s.asset, a.ProviderID, a.ModelID, a.Status,
			b.Dx(), b.Dy(), len(raw), len(colours))
		if len(colours) < 256 {
			t.Errorf("slot %s renders %d distinct colours — that is a colour grid, not a portrait",
				s.owner, len(colours))
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no portrait was verified end to end")
	}
}

// The LIVE scene batch: world cover + place backdrops for the play world, over the platform's
// scene_capable route, then every image fetched back through THIS service's own endpoint and its
// bytes decoded. Same gating and same proof standard as TestLive_PortraitsAreRealArt, and it SPENDS
// MONEY — one generation per authored place plus one for the cover, paid once each thereafter.
//
//	DREAMCHAT_IMAGE_LIVE=1 DREAMCHAT_IMAGE_SCENE_BATCH=1 \
//	DREAMCHAT_IMAGE_BASE_URL=http://localhost:8081 DREAMCHAT_IMAGE_API_TOKEN=... \
//	DATABASE_URL=... go test -run TestLive_SceneBackdropsAreRealArt -timeout 30m -v .
func TestLive_SceneBackdropsAreRealArt(t *testing.T) {
	if os.Getenv("DREAMCHAT_IMAGE_LIVE") == "" || os.Getenv("DREAMCHAT_IMAGE_SCENE_BATCH") == "" {
		t.Skip("needs DREAMCHAT_IMAGE_LIVE and DREAMCHAT_IMAGE_SCENE_BATCH — this one spends money")
	}
	client := newImageClientFromEnv()
	if client == nil {
		t.Fatal("DREAMCHAT_IMAGE_BASE_URL / DREAMCHAT_IMAGE_API_TOKEN not set")
	}
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	const worldID = "22222222-2222-2222-2222-222222222222"

	// The batch is capped at the platform's concurrency default, so drain it in rounds rather than
	// raising the cap: the cap is the reason one call cannot outrun them.
	for round := 1; round <= 4; round++ {
		res, err := fillScenes(ctx, pool, client, worldID, imageBatchLimit)
		if err != nil {
			t.Fatalf("fillScenes round %d: %v", round, err)
		}
		fmt.Printf("LIVE scenes round %d: %+v\n", round, res)
		if res.Requested == 0 && res.Reclaimed == 0 {
			break
		}
	}

	rows, err := pool.Query(ctx, `
		SELECT s.owner_kind, s.owner_id::text, coalesce(s.asset_id,''), coalesce(s.last_error,''),
		       coalesce(er.canonical_name, w.display_name, '?')
		  FROM image_slot s
		  LEFT JOIN entity_registry er ON er.world_id = s.world_id AND er.entity_id = s.owner_id
		  LEFT JOIN world w ON w.world_id = s.owner_id
		 WHERE s.world_id = $1 AND s.owner_kind IN ('world','location')
		 ORDER BY s.owner_kind DESC, 5`, worldID)
	if err != nil {
		t.Fatalf("read slots: %v", err)
	}
	type slot struct{ kind, id, asset, lastErr, name string }
	var slots []slot
	for rows.Next() {
		var s slot
		if err := rows.Scan(&s.kind, &s.id, &s.asset, &s.lastErr, &s.name); err != nil {
			rows.Close()
			t.Fatalf("scan: %v", err)
		}
		slots = append(slots, s)
	}
	rows.Close()

	h := NewImageHandler(pool, client, true)
	checked := 0
	for _, s := range slots {
		if s.asset == "" {
			t.Errorf("%s %q has no asset (last_error=%q)", s.kind, s.name, s.lastErr)
			continue
		}
		var a struct {
			Status     string `json:"status"`
			ProviderID string `json:"provider_id"`
			ModelID    string `json:"model_id"`
		}
		if err := client.do(ctx, http.MethodGet, "/v1/assets/"+s.asset, nil, "", &a); err != nil {
			t.Errorf("asset %s: %v", s.asset, err)
			continue
		}
		if a.Status == "archived" || a.Status == "failed" {
			t.Errorf("%s %q still names a %s asset", s.kind, s.name, a.Status)
		}
		if a.ProviderID == "mock" || a.ModelID == "pm_mock_v1" {
			t.Errorf("%s %q is mock art: provider=%s model=%s — the router did not resolve a real "+
				"scene_capable route", s.kind, s.name, a.ProviderID, a.ModelID)
		}

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
			"/worlds/"+worldID+"/images/"+s.asset+"?tier=preview", nil))
		if rec.Code != http.StatusFound {
			t.Errorf("fetch %s: status %d, want 302", s.asset, rec.Code)
			continue
		}
		resp, err := http.Get(rec.Header().Get("Location"))
		if err != nil {
			t.Errorf("follow redirect for %s: %v", s.asset, err)
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("decode %s: %v (%d bytes)", s.asset, err, len(raw))
			continue
		}
		colours := map[uint32]struct{}{}
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y += 2 {
			for x := b.Min.X; x < b.Max.X; x += 2 {
				r, g, bl, _ := img.At(x, y).RGBA()
				colours[r>>8<<16|g>>8<<8|bl>>8] = struct{}{}
			}
		}
		fmt.Printf("LIVE %-8s %-24s asset=%s provider=%s model=%s status=%s %dx%d %d bytes %d colours\n",
			s.kind, s.name, s.asset, a.ProviderID, a.ModelID, a.Status, b.Dx(), b.Dy(), len(raw), len(colours))
		if len(colours) < 256 {
			t.Errorf("%s %q renders %d distinct colours — that is a colour grid, not a backdrop",
				s.kind, s.name, len(colours))
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no backdrop was verified end to end")
	}
	fmt.Printf("LIVE scene batch verified: %d image(s)\n", checked)
}
