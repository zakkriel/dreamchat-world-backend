# ADR-P021: Art is reconciled, never commissioned by hand

**Status:** Accepted (2026-08-16)
**Date:** 2026-08-16
**Series:** Platform / Images (ADR-P###, per D-5).
**Owner of decision:** the founder's report that a created world had no pictures.
**Evidence (D-9):** `core/api/artcommission.go`, the genesis kick in `core/api/worldgenesishandler.go`,
the ticker in `core/api/main.go`, `core/api/artcommission_test.go`.

## Context

`POST /worlds/{w}/images/scenes` and `.../portraits` were explicit, bounded triggers, and nothing ever
called them. A world authored by genesis had a cast, rooms and objects and not one picture of any of
them; the only worlds with art had it because somebody ran curl from a runbook.

A creation that does not commission its own art is not finished, and asking a person to remember a
second call is not a design.

Entities also do not all arrive at genesis. The cast grows as the story does — a beat introduces
someone, a place is authored on arrival, an object is produced — and each is a separate asynchronous
act. A commissioning step wired into genesis alone would illustrate the opening cast and nothing after.

## Decision

The unit is **per world and idempotent**: ask what has no picture and fill exactly that, indifferent to
which path created the entity or when.

1. `commissionWorldArt` drains a world: cover, places, objects, cast, paged, with a pass ceiling so a
   permanently failing owner cannot spin.
2. **Genesis kicks it** after the `world` frame, detached, so a new world is illustrated immediately.
3. **A ticker sweeps** every non-archived world, so anything created later — by any path, including
   ones that do not exist yet — is picked up without new wiring.
4. It gates on `pendingArtCount`, pure SQL, BEFORE anything reaches for the platform. The fills open
   with `ensureStyle`, an HTTP round trip; a reconciler that started there would call another service
   on every tick of every world forever just to be told there was nothing to do.
5. **Outside the genesis transaction.** One image is 25–90s of somebody else's service; a dozen would
   put minutes inside the transaction holding the world's canon, and a provider outage would then
   destroy an authored world rather than delay its pictures. The payload already promises this shape:
   `image` is null until it is not and swaps in on a later read (`image_ref/1`, D-8).

## Consequences

- A new creation path inherits art for free. Do NOT add a commissioning call to it.
- Archived worlds are skipped: illustrating them spends real money on something nobody can open.
- Artifacts are drawable and read `attrs->>'descriptor'`; **portals are not** — an artifact carrying
  `connects` is an opening between two places. The descriptor test alone let three doors through and
  billed for pictures of them.
