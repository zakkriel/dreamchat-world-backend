package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WHY THIS EXISTS. Art was commissioned by hand. `POST /worlds/{w}/images/scenes` and
// `.../portraits` are explicit, bounded triggers, and nothing ever called them — so a world authored
// by genesis had a cast, rooms and objects, and not one picture of any of them. The two worlds that
// did have art had it because somebody ran curl from a runbook.
//
// A creation that does not commission its own art is not finished, and asking the founder to
// remember a second call is not a design.
//
// WHY A RECONCILER RATHER THAN A STEP INSIDE CREATION. Entities do not all arrive at genesis. The
// cast grows as the story does — a beat introduces someone, a place is authored on arrival, an
// object is produced — and each of those is a separate, asynchronous act. A commissioning step
// wired into genesis alone would illustrate the opening cast and nothing that came after.
//
// So the unit here is per WORLD and idempotent: it asks what has no picture and fills exactly that.
// It does not care which path created the entity, or when. Genesis kicks it so a new world is
// illustrated immediately, and a ticker sweeps for everything created later — including retries of
// whatever failed transiently the first time. A new creation path added tomorrow inherits art
// without touching this file, which is the property the hand-rolled triggers did not have.
//
// It is NOT inside the genesis transaction, deliberately. One image is 25-90 seconds of somebody
// else's service; a dozen would put minutes inside a transaction that holds the world's canon, and
// a provider outage would then destroy the world rather than delay its pictures. The contract the
// payload already states is that `image` is null until it is not, and swaps in on a later read
// (image_ref/1, D-8) — so art arriving after the world does is the shape the API already promises.

// artCommissionInterval is how often the reconciler looks for unillustrated entities. Art is not
// urgent — nothing is blocked on it, and the payload renders a placeholder meanwhile — so this is
// deliberately slow enough to be invisible in cost and load.
const artCommissionInterval = 2 * time.Minute

// artCommissionTimeout bounds one world's sweep. Generous, because a full world is a dozen images at
// up to 90 seconds each, but finite: a wedged provider must not hold the slot forever.
const artCommissionTimeout = 20 * time.Minute

// inFlightWorlds keeps one sweep per world. Genesis kicks a sweep and the ticker may reach the same
// world moments later; without this they would race for the same slots, and both would pay.
var inFlightWorlds sync.Map

// pendingArtCount reports how many owners in this world have no picture and nothing in flight.
//
// It is pure SQL and runs BEFORE anything reaches for the image platform. The fill functions open
// with ensureStyle, an HTTP round trip, so a sweep that started there would call another service on
// every tick of every world forever just to be told there was nothing to do.
func pendingArtCount(ctx context.Context, pool *pgxpool.Pool, worldID string) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT w.world_id
			  FROM world w
			  LEFT JOIN image_slot s
			    ON s.world_id = w.world_id AND s.owner_kind = 'world' AND s.owner_id = w.world_id
			 WHERE w.world_id = $1 AND w.tagline IS NOT NULL
			   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
			UNION ALL
			SELECT er.entity_id
			  FROM entity_registry er
			  JOIN location_state l ON l.world_id = er.world_id AND l.entity_id = er.entity_id
			  LEFT JOIN image_slot s
			    ON s.world_id = er.world_id AND s.owner_kind = 'location' AND s.owner_id = er.entity_id
			 WHERE er.world_id = $1 AND er.entity_kind = 'location'
			   AND coalesce(btrim(l.attrs->>'description'), '') <> ''
			   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
			UNION ALL
			SELECT er.entity_id
			  FROM entity_registry er
			  JOIN artifact_state a ON a.world_id = er.world_id AND a.entity_id = er.entity_id
			  LEFT JOIN image_slot s
			    ON s.world_id = er.world_id AND s.owner_kind = 'artifact' AND s.owner_id = er.entity_id
			 WHERE er.world_id = $1 AND er.entity_kind = 'artifact'
			   AND coalesce(btrim(a.attrs->>'descriptor'), '') <> ''
			   AND NOT (a.attrs ? 'connects')
			   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
			UNION ALL
			SELECT er.entity_id
			  FROM entity_registry er
			  LEFT JOIN image_slot s
			    ON s.world_id = er.world_id AND s.owner_kind = 'actor' AND s.owner_id = er.entity_id
			 WHERE er.world_id = $1 AND er.entity_kind = 'actor'
			   AND (s.owner_id IS NULL OR (s.asset_id IS NULL AND s.job_id IS NULL))
		) pending`, worldID).Scan(&n)
	return n, err
}

// commissionWorldArt fills every missing picture in one world: cover, places, objects, cast.
//
// The fills are paged (imageBatchLimit at a time), so this drains them rather than illustrating the
// first five things and stopping. The pass ceiling is what keeps a permanently failing owner from
// spinning: a slot that fails records its error and is no longer "nothing in flight", so it drops
// out of the pending set on its own — but a fill that keeps returning the same work without
// progressing must still terminate.
func commissionWorldArt(ctx context.Context, pool *pgxpool.Pool, client *imageClient, worldID string) {
	if client == nil {
		return
	}

	const maxPasses = 12
	for range maxPasses {
		pending, err := pendingArtCount(ctx, pool, worldID)
		if err != nil {
			log.Printf("art: world %s: counting unillustrated owners: %v", worldID, err)
			return
		}
		if pending == 0 {
			return
		}

		before := pending

		if _, err := fillScenes(ctx, pool, client, worldID, imageBatchLimit); err != nil {
			log.Printf("art: world %s: scenes: %v", worldID, err)
			return
		}
		if _, err := fillPortraits(ctx, pool, client, worldID, imageBatchLimit); err != nil {
			log.Printf("art: world %s: portraits: %v", worldID, err)
			return
		}

		after, err := pendingArtCount(ctx, pool, worldID)
		if err != nil {
			log.Printf("art: world %s: counting unillustrated owners: %v", worldID, err)
			return
		}
		// No progress means every remaining owner failed rather than filled. They carry their error
		// now and the next sweep will try them again; continuing here would just re-fail them in a
		// tighter loop and bill for it.
		if after >= before {
			log.Printf("art: world %s: %d owner(s) still unillustrated after a full pass — leaving them for the next sweep", worldID, after)
			return
		}
	}
}

// commissionArtInBackground detaches a sweep from whatever triggered it.
//
// Genesis calls this from a request that has already told the user their world is ready; the sweep
// must outlive that request, so it takes a fresh context rather than the caller's. One sweep per
// world at a time — genesis kicks one and the ticker may arrive moments later, and both would
// otherwise pay for the same slots.
// kickArt is the seam creation paths call, and exists as a var for one reason: a direct call is
// unobservable, so nothing could prove that genesis actually commissions anything. The founder's
// first report on this feature was a created world with no pictures — a silent missing call is
// exactly the failure that started it, and it must be a test, not a comment.
var kickArt = commissionArtInBackground

func commissionArtInBackground(pool *pgxpool.Pool, client *imageClient, worldID string) {
	if client == nil {
		return
	}
	if _, busy := inFlightWorlds.LoadOrStore(worldID, struct{}{}); busy {
		return
	}

	go func() {
		defer inFlightWorlds.Delete(worldID)
		defer func() {
			// A panic in art must never take the server down. Nothing is blocked on a picture.
			if r := recover(); r != nil {
				log.Printf("art: world %s: recovered from panic: %v", worldID, r)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), artCommissionTimeout)
		defer cancel()
		commissionWorldArt(ctx, pool, client, worldID)
	}()
}

// runArtReconciler sweeps every live world for unillustrated entities, forever.
//
// This is what makes art automatic for creation paths that do not know it exists. A beat that
// introduces a character, a place authored on arrival, an object produced mid-scene — none of them
// call anything here, and all of them get pictures.
//
// Archived worlds are skipped: they are out of the directory, and illustrating them spends real
// money on something nobody can open.
func runArtReconciler(pool *pgxpool.Pool, client *imageClient) {
	if client == nil {
		log.Printf("art: reconciler disabled (no image platform configured) — worlds run, pictures stay absent")
		return
	}

	go func() {
		for {
			time.Sleep(artCommissionInterval)

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			rows, err := pool.Query(ctx, `SELECT world_id::text FROM world WHERE archived_at IS NULL`)
			if err != nil {
				cancel()
				log.Printf("art: reconciler: listing worlds: %v", err)
				continue
			}
			var worlds []string
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil {
					worlds = append(worlds, id)
				}
			}
			rows.Close()
			cancel()

			for _, id := range worlds {
				kickArt(pool, client, id)
			}
		}
	}()
}
