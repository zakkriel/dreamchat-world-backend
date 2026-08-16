package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The reconciler's whole claim is that it finds what has no picture, whoever created it and
// whenever. pendingArtCount is the part that decides, so these pin what counts as unillustrated.

func TestPendingArtCount_SeesEveryKindThatCanBeDrawn(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID := "a17c0000-0000-0000-0000-000000000001"
	actorID := "a17c0000-0000-0000-0000-000000000101"
	placeID := "a17c0000-0000-0000-0000-000000000102"
	thingID := "a17c0000-0000-0000-0000-000000000103"

	clear := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM artifact_state WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_state WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, worldID)
	}
	clear()
	t.Cleanup(clear)

	if _, err := pool.Exec(ctx, `
		INSERT INTO world (world_id, display_name, tagline, theme)
		VALUES ($1,'Reconciler World','a tagline is what makes a cover drawable',
		        '{"schema_version":"world_theme/1","accent":"#4f6d8c","mood":"mist","ornament":"none"}'::jsonb)`,
		worldID); err != nil {
		t.Fatalf("seed world: %v", err)
	}

	// The cover alone: a world with a tagline and no cover slot.
	if n := count(t, ctx, pool, worldID); n != 1 {
		t.Fatalf("a world with a tagline and no cover is 1 unillustrated owner, got %d", n)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		  ($1,$4,'actor','Someone'), ($2,$4,'location','somewhere'), ($3,$4,'artifact','something')`,
		actorID, placeID, thingID, worldID); err != nil {
		t.Fatalf("seed entities: %v", err)
	}
	// An actor needs no state row to be drawable — its descriptor is optional and its name stands in.
	if n := count(t, ctx, pool, worldID); n != 2 {
		t.Fatalf("the cast is drawable on registration alone, want 2, got %d", n)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_state (entity_id, world_id, attrs)
		VALUES ($1,$2,'{"description":"a room worth drawing"}'::jsonb)`, placeID, worldID); err != nil {
		t.Fatalf("seed place: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO artifact_state (entity_id, world_id, attrs)
		VALUES ($1,$2,'{"descriptor":"an object worth drawing"}'::jsonb)`, thingID, worldID); err != nil {
		t.Fatalf("seed artifact: %v", err)
	}

	// Cover + place + object + actor. The object is the one the hand-rolled triggers never covered.
	if n := count(t, ctx, pool, worldID); n != 4 {
		t.Fatalf("cover, place, object and actor are all drawable, want 4, got %d", n)
	}
}

// A doorway is registered as an artifact but carries no artifact_state row, so it has no descriptor
// and is not a thing to draw. Without this the reconciler would spend real money on every exit.
func TestPendingArtCount_IgnoresAPortalWithNothingToDraw(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID := "a17c0000-0000-0000-0000-000000000002"
	portalID := "a17c0000-0000-0000-0000-000000000201"

	clear := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM artifact_state WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, worldID)
	}
	clear()
	t.Cleanup(clear)

	if _, err := pool.Exec(ctx, `
		INSERT INTO world (world_id, display_name, theme)
		VALUES ($1,'Portal World','{"schema_version":"world_theme/1","accent":"#4f6d8c","mood":"mist","ornament":"none"}'::jsonb)`,
		worldID); err != nil {
		t.Fatalf("seed world: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		VALUES ($1,$2,'artifact','a heavy door at the bottom of the stairwell')`, portalID, worldID); err != nil {
		t.Fatalf("seed portal: %v", err)
	}

	// No tagline, so no cover; a portal with no artifact_state, so nothing to draw.
	if n := count(t, ctx, pool, worldID); n != 0 {
		t.Fatalf("a portal is not a picture, want 0 unillustrated owners, got %d", n)
	}
}

// A filled slot and a slot with a job in flight are both "not pending". Without the in-flight half
// the reconciler would re-request everything it had just asked for, every tick.
func TestPendingArtCount_SkipsFilledAndInFlight(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID := "a17c0000-0000-0000-0000-000000000003"
	drawn := "a17c0000-0000-0000-0000-000000000301"
	inFlight := "a17c0000-0000-0000-0000-000000000302"

	clear := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, worldID)
	}
	clear()
	t.Cleanup(clear)

	if _, err := pool.Exec(ctx, `
		INSERT INTO world (world_id, display_name, theme)
		VALUES ($1,'Busy World','{"schema_version":"world_theme/1","accent":"#4f6d8c","mood":"mist","ornament":"none"}'::jsonb)`,
		worldID); err != nil {
		t.Fatalf("seed world: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		VALUES ($1,$3,'actor','Drawn'), ($2,$3,'actor','Waiting')`, drawn, inFlight, worldID); err != nil {
		t.Fatalf("seed actors: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_slot (world_id, owner_kind, owner_id, asset_id) VALUES ($1,'actor',$2,'asset_done')`,
		worldID, drawn); err != nil {
		t.Fatalf("seed filled slot: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_slot (world_id, owner_kind, owner_id, job_id) VALUES ($1,'actor',$2,'job_running')`,
		worldID, inFlight); err != nil {
		t.Fatalf("seed in-flight slot: %v", err)
	}

	if n := count(t, ctx, pool, worldID); n != 0 {
		t.Fatalf("a drawn actor and one already asked for are both settled, want 0, got %d", n)
	}
}

// count is pendingArtCount with the error folded into a fatal, so each assertion above reads as the
// number it is about.
func count(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worldID string) int {
	t.Helper()
	n, err := pendingArtCount(ctx, pool, worldID)
	if err != nil {
		t.Fatalf("pendingArtCount: %v", err)
	}
	return n
}
