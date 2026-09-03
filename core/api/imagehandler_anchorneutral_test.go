package main

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedSpriteActor gives each test its own actor and a clean set of slots. The
// suite shares one database by design, so rows are cleared at the START as well
// as on exit: a previous run that died mid-test would otherwise be inherited.
func seedSpriteActor(t *testing.T, pool *pgxpool.Pool) (worldID, actorID string) {
	t.Helper()
	ctx := context.Background()
	worldID, actorID = "5a5e0000-0000-0000-0000-0000000000f2", "5a5e0000-0000-0000-0000-0000000000a2"
	clear := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM image_slot WHERE world_id=$1`, worldID)
		_, _ = pool.Exec(ctx, `DELETE FROM entity_registry WHERE world_id=$1`, worldID)
	}
	clear()
	if _, err := pool.Exec(ctx,
		`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		 VALUES ($1,$2,'actor',$3) ON CONFLICT DO NOTHING`, actorID, worldID, spriteActorName); err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	t.Cleanup(clear)
	return worldID, actorID
}

const spriteActorName = "Anchor Neutral Subject"

// A character used to cost FIVE renders: an anchor, then four emotion cells —
// and the anchor and the neutral cell were the same picture of the same face
// under two prompts. The anchor is now minted with the neutral cell's exact
// prompt and serves as that cell, so a character costs FOUR.
//
// This asserts the saving where it is spendable: the pack must ask for three
// variants, and none of them may be the neutral one.
func TestSpritePackSkipsNeutralWhenTheAnchorIsMintedHere(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	worldID, actorID := seedSpriteActor(t, pool)

	f := newFakePlatform()
	// No anchors: this run mints one, so it doubles as the neutral sprite.
	f.identityAnchors[actorID] = []string{}
	c := testImageClient(t, f)

	if _, err := fillPortraits(ctx, pool, c, worldID, 5); err != nil {
		t.Fatalf("fillPortraits: %v", err)
	}

	if len(f.lastPackVariantPrompts) != len(spriteEmotionsWithoutNeutral) {
		t.Fatalf("pack asked for %d cells (%v), want %d — the anchor already covers neutral",
			len(f.lastPackVariantPrompts), packCellKeys(f), len(spriteEmotionsWithoutNeutral))
	}
	if _, present := f.lastPackVariantPrompts[spriteVariantKey(spriteNeutralEmotion)]; present {
		t.Fatalf("the neutral cell was generated AND anchored — that is the duplicate render this removes: %v",
			packCellKeys(f))
	}

	// The anchor must be minted with the neutral cell's prompt, or it is a
	// differently-framed picture standing in for a sprite.
	want := spriteCellPrompt(portraitAppearance("", spriteActorName), spriteNeutralEmotion)
	if f.lastBootstrapDescription != want {
		t.Fatalf("anchor prompt drifted from the neutral cell prompt:\n got: %q\nwant: %q",
			f.lastBootstrapDescription, want)
	}
	if !strings.Contains(f.lastBootstrapDescription, spriteFramingPrompt) {
		t.Fatal("the anchor must carry the sprite framing clause; without it the neutral face is composed differently from the other three")
	}
}

// fn_sprite_set returns all four variants or NULL — "three right faces and one
// wrong one is worse than waiting". Generating three cells is only safe if the
// neutral slot is genuinely filled from the anchor, with no job left in flight.
func TestNeutralSlotIsFilledFromTheAnchorAsset(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	worldID, actorID := seedSpriteActor(t, pool)

	f := newFakePlatform()
	f.identityAnchors[actorID] = []string{}
	c := testImageClient(t, f)

	if _, err := fillPortraits(ctx, pool, c, worldID, 5); err != nil {
		t.Fatalf("fillPortraits: %v", err)
	}

	for _, emotion := range spriteEmotionOrder {
		var assetID *string
		var jobID *string
		if err := pool.QueryRow(ctx,
			`SELECT asset_id, job_id FROM image_slot
			  WHERE world_id=$1 AND owner_kind='actor' AND owner_id=$2 AND variant=$3`,
			worldID, actorID, emotion).Scan(&assetID, &jobID); err != nil {
			t.Fatalf("read %s slot: %v", emotion, err)
		}
		if assetID == nil || *assetID == "" {
			t.Fatalf("%s slot is unfilled — fn_sprite_set returns NULL for the whole set unless all four carry an asset", emotion)
		}
		if jobID != nil {
			t.Fatalf("%s slot still has job_id=%q; a settled slot must never be polled again", emotion, *jobID)
		}
	}

	var neutralAsset string
	if err := pool.QueryRow(ctx,
		`SELECT asset_id FROM image_slot
		  WHERE world_id=$1 AND owner_kind='actor' AND owner_id=$2 AND variant=$3`,
		worldID, actorID, spriteNeutralEmotion).Scan(&neutralAsset); err != nil {
		t.Fatalf("read neutral slot: %v", err)
	}
	// The pack's assets are suffixed per emotion by the fake; the anchor's is not.
	if strings.HasSuffix(neutralAsset, "_"+spriteNeutralEmotion) {
		t.Fatalf("neutral resolved to a generated pack cell (%q) rather than the anchor asset", neutralAsset)
	}
}

// An identity anchored BEFORE this change carries an unframed portrait. Showing
// it as one of four sprites would put a differently-composed neutral beside
// three framed figures, so those characters keep the four-cell pack.
func TestExistingAnchorStillGeneratesAllFourCells(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	worldID, actorID := seedSpriteActor(t, pool)

	f := newFakePlatform()
	f.identityAnchors[actorID] = []string{"asset_anchor_from_before"}
	c := testImageClient(t, f)

	if _, err := fillPortraits(ctx, pool, c, worldID, 5); err != nil {
		t.Fatalf("fillPortraits: %v", err)
	}
	if len(f.lastPackVariantPrompts) != len(spriteEmotionOrder) {
		t.Fatalf("an already-anchored character must still get all %d cells, got %v",
			len(spriteEmotionOrder), packCellKeys(f))
	}
	if _, present := f.lastPackVariantPrompts[spriteVariantKey(spriteNeutralEmotion)]; !present {
		t.Fatalf("legacy anchors are unframed; the neutral cell must still be generated: %v",
			packCellKeys(f))
	}
}

// packCellKeys names the cells the pack actually asked for, so a failure says
// which ones rather than only how many.
func packCellKeys(f *fakePlatform) []string {
	out := make([]string, 0, len(f.lastPackVariantPrompts))
	for k := range f.lastPackVariantPrompts {
		out = append(out, k)
	}
	return out
}
