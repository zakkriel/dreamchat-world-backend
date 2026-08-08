package main

import (
	"context"
	"fmt"
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
