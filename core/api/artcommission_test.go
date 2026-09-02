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

// A doorway is registered as an artifact and DOES carry a descriptor, so the descriptor test alone
// is not enough — it let three doors through in the first world to use this path and billed for
// pictures of them. What separates a portal from an object is structural: it names the two places it
// joins. Both shapes are asserted here, because only one of them was caught by reading the data.
func TestPendingArtCount_IgnoresAPortal(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID := "a17c0000-0000-0000-0000-000000000002"
	portalID := "a17c0000-0000-0000-0000-000000000201"
	const placeA = "a17c0000-0000-0000-0000-0000000002a1"
	const placeB = "a17c0000-0000-0000-0000-0000000002a2"

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
	describedPortal := "a17c0000-0000-0000-0000-000000000202"
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		  ($1,$3,'artifact','a heavy door at the bottom of the stairwell'),
		  ($2,$3,'artifact','a sliding door between the carriage and the cabin')`,
		portalID, describedPortal, worldID); err != nil {
		t.Fatalf("seed portals: %v", err)
	}
	// The shape that actually shipped: a portal WITH a descriptor, told apart by `connects`.
	if _, err := pool.Exec(ctx, `
		INSERT INTO artifact_state (entity_id, world_id, attrs)
		VALUES ($1,$2,'{"descriptor":"a sliding door between the carriage and the cabin","connects":["`+placeA+`","`+placeB+`"]}'::jsonb)`,
		describedPortal, worldID); err != nil {
		t.Fatalf("seed described portal: %v", err)
	}

	// No tagline, so no cover. One portal has no state row; the other has a descriptor and would be
	// drawn on the descriptor test alone. Neither is a picture.
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

// A TERMINAL provider refusal must leave the pending set. The old code's comment
// claimed a failed owner "drops out of the pending set on its own" because it is
// "no longer nothing in flight" — but a failed slot has asset_id NULL and job_id
// NULL, which is exactly the pending condition, so it never dropped out at all.
//
// Measured in production 2026-09-01: 875 artifact jobs failed in 24h, all
// `submit returned status 402` (the BFL account had no credit), because the
// 2-minute sweep re-commissioned the same doomed owners forever. Those submits
// consumed the image platform's entire 1000-requests/hour token budget, and the
// asset READ path shares that budget — so every already-rendered picture in the
// product became unfetchable. The art blackout was caused by retrying, not by
// the billing failure itself.
//
// A transient failure must keep being retried: that is the self-healing the
// sweep exists for. Only a refusal that cannot change is excluded.
func TestPendingArtCount_TerminalRefusalStopsCostingSweeps(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID := "a17c0000-0000-0000-0000-000000000201"
	placeID := "a17c0000-0000-0000-0000-000000000202"

	clear := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_state WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, worldID)
	}
	clear()
	t.Cleanup(clear)

	// A world with no tagline: the cover is not drawable, so the place is the only
	// owner in play and the count reads as one thing, not two.
	if _, err := pool.Exec(ctx, `INSERT INTO world (world_id, display_name) VALUES ($1,'Unpaid World')`, worldID); err != nil {
		t.Fatalf("seed world: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		VALUES ($1,$2,'location','somewhere')`, placeID, worldID); err != nil {
		t.Fatalf("seed entity: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_state (entity_id, world_id, attrs)
		VALUES ($1,$2,'{"description":"a room worth drawing"}'::jsonb)`, placeID, worldID); err != nil {
		t.Fatalf("seed place: %v", err)
	}
	if n := count(t, ctx, pool, worldID); n != 1 {
		t.Fatalf("a described place with no picture is 1 unillustrated owner, got %d", n)
	}

	failWith := func(msg string) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO image_slot (world_id, owner_kind, owner_id, variant, last_error, updated_at)
			VALUES ($1,'location',$2,'default',$3, now())
			ON CONFLICT (world_id, owner_kind, owner_id, variant) DO UPDATE
			   SET last_error = EXCLUDED.last_error, asset_id = NULL, job_id = NULL, updated_at = now()`,
			worldID, placeID, msg); err != nil {
			t.Fatalf("record failure: %v", err)
		}
	}

	// Transient: the account is fine and the next sweep can genuinely succeed.
	failWith("provider_failure: bfl: submit returned status 500")
	if n := count(t, ctx, pool, worldID); n != 1 {
		t.Fatalf("a transient failure must stay pending so the sweep heals it, got %d", n)
	}

	// Terminal: an unpaid invoice is not settled by asking again.
	failWith("provider_unpaid: bfl: submit returned status 402")
	if n := count(t, ctx, pool, worldID); n != 0 {
		t.Fatalf("an unpaid refusal must stop costing sweeps, want 0 pending, got %d", n)
	}

	// Terminal: re-sending identical content re-bills a deterministic refusal.
	failWith(`provider_content_rejected: bfl: provider returned terminal status "Content Moderated"`)
	if n := count(t, ctx, pool, worldID); n != 0 {
		t.Fatalf("a content rejection must stop costing sweeps, want 0 pending, got %d", n)
	}

	// And a cleared error is drawable again: paying the invoice and clearing the
	// slot must bring it back, or the world could never heal.
	if _, err := pool.Exec(ctx, `UPDATE image_slot SET last_error=NULL WHERE world_id=$1`, worldID); err != nil {
		t.Fatalf("clear error: %v", err)
	}
	if n := count(t, ctx, pool, worldID); n != 1 {
		t.Fatalf("clearing the error must make the owner drawable again, got %d", n)
	}
}

// The cast has its own pending shape - four emotion slots per actor rather than one
// picture - so it needs its own proof. Without this the actor exclusion is deletable
// code: removing it leaves the location test green, which is how a guard nobody has
// watched go red gets shipped.
func TestPendingArtCount_TerminalRefusalStopsCostingSweepsForTheCast(t *testing.T) {
	pool := testPool(t)
	t.Cleanup(func() { pool.Close() })
	ctx := context.Background()

	worldID := "a17c0000-0000-0000-0000-000000000301"
	actorID := "a17c0000-0000-0000-0000-000000000302"

	clear := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM world WHERE world_id=$1`, worldID)
	}
	clear()
	t.Cleanup(clear)

	// No tagline, so the cover is not drawable and the actor is the only owner in play.
	if _, err := pool.Exec(ctx, `INSERT INTO world (world_id, display_name) VALUES ($1,'Unpaid Cast')`, worldID); err != nil {
		t.Fatalf("seed world: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		VALUES ($1,$2,'actor','Someone')`, actorID, worldID); err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	if n := count(t, ctx, pool, worldID); n != 1 {
		t.Fatalf("an actor with no sprites is 1 unillustrated owner, got %d", n)
	}

	failSprite := func(msg string) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO image_slot (world_id, owner_kind, owner_id, variant, last_error, updated_at)
			VALUES ($1,'actor',$2,'neutral',$3, now())
			ON CONFLICT (world_id, owner_kind, owner_id, variant) DO UPDATE
			   SET last_error = EXCLUDED.last_error, asset_id = NULL, job_id = NULL, updated_at = now()`,
			worldID, actorID, msg); err != nil {
			t.Fatalf("record failure: %v", err)
		}
	}

	// Transient: an incomplete sprite set must keep being retried.
	failSprite("provider_failure: fal: submit returned status 503")
	if n := count(t, ctx, pool, worldID); n != 1 {
		t.Fatalf("a transient sprite failure must stay pending, got %d", n)
	}

	// Terminal: the pack cannot complete until someone pays, so stop asking.
	failSprite("provider_unpaid: fal: submit returned status 402")
	if n := count(t, ctx, pool, worldID); n != 0 {
		t.Fatalf("an unpaid sprite refusal must stop costing sweeps, want 0, got %d", n)
	}

	if _, err := pool.Exec(ctx, `UPDATE image_slot SET last_error=NULL WHERE world_id=$1`, worldID); err != nil {
		t.Fatalf("clear error: %v", err)
	}
	if n := count(t, ctx, pool, worldID); n != 1 {
		t.Fatalf("clearing the error must make the cast drawable again, got %d", n)
	}
}
