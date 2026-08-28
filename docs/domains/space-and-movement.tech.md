# space-and-movement · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-5 · Space, movement and the journey ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the write and read paths, validation, traps.
`space-and-movement.product.md` holds what it means; `space-and-movement.seams.md` holds what
crosses its boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate
by grep before relying on one.

---

## Storage

The defining discipline is `D-11`: coordinates are recorded; distance, duration, reachability and
containment are computed at ask-time, never stored. Most bugs here are a stored number that should
have been a computed one.

- **Positions live in `attrs`**: `attrs.coordinates` (`{x,y}` in the parent frame),
  `attrs.parent_location_id` (the frame edge), `attrs.location_id` (an actor's/artifact's scene),
  `attrs.area` (ordered ≥3-point outline, optional), `attrs.entry_point` (a location's landing
  spot). Seeded examples: grep `attrs.coordinates` in `core/db/schema.sql`.
- **Vocabulary tables** (measurements only, no verdict columns): `movement_type` (base speed,
  CHECK > 0) and `status_modifier` (percent, CHECK ≥ −100; `action_type` is the single value
  `'move'`). `seed_world_defaults` (`schema.sql:3701`) seeds exactly `walk 1.4` and
  `encumbered −100` — every other row is LLM-minted at need.
- **`journey`** (`schema.sql:3951`) — loop state, not canon (the `held_outcome` precedent): rows
  may be deleted, no append-only guard. `idx_journey_one_active` (`:4691`) lets the database refuse
  a second active journey per actor. `journey_legs_band` and `watch_horizon` are per-world dials,
  also seeded there.

## The read path — the function inventory

All in `core/db/schema.sql`; grep `CREATE FUNCTION public.fn_<name>`:

- `fn_distance` (`:1557`) — **the one distance formula.** Euclidean over `x`/`y` at the nearest
  common parent's frame; missing coordinate = frame origin `{0,0}`; parent walks capped at depth 64
  (I-4 mirror, bounded not raised).
- `fn_effective_speed` (`:1663`) — base speed × Π(1 + pct/100) over the actor's *active* statuses;
  floor 0, no cap; `NULL` when the type is not minted. `plpgsql` for exact numeric multiplication —
  the −100% → factor 0 case is the whole point.
- `fn_move_duration_actor` (`:2529`) — distance ÷ effective speed; speed ≤ 0 returns max bigint:
  *"infinite duration: blocked by arithmetic (§2), not a branch."* Prevention emerges, never coded.
  `fn_move_duration` (`:2518`) is the **legacy wrapper** (`CEIL(distance/1.4)`) kept for
  `apply_beat` — two functions, two test files, deliberate (see Validation).
- `fn_portal_permits` (`:2686`) — open ∧ ¬locked ∧ `connects` holds both ids. No stored
  `reachable` anywhere: it would rot the instant a lock flips.
- `fn_actor_move_permitted` (`:792`) — the whole `ActorMoved` accessibility floor: target exists;
  same-scene or no origin → true; cross-location → `fn_portal_permits`. Mirrored by Go's
  `premiseHolds` so the twins cannot drift.
- `fn_target_position` (`:2860`) — resolves any target (location → `entry_point`; actor/artifact →
  its own coordinates) once; gate and commit read the same resolution.
- `fn_actors_at` (`:860`) — **place-level, binary co-presence**: actors whose `attrs.location_id`
  equals the place. Reads `actor_state`, nothing else. We own the predicate; perception consumes it
  (`seams.md`).
- `fn_place_at` (`:2668`) — smallest child area of a frame containing a point; NULL = open road.
  With `fn_area_polygon`, `fn_area_around`, `fn_extent_class_metres` (the engine draws footprints).
- `fn_world_now` (`:3212`) — GREATEST of canon ticks and journey ticks: quiet legs move the clock
  with no filler canon event. Shared with Time (`seams.md`).

## The write path — one unified move

`apply_event`'s `ActorMoved` arm (`schema.sql:318-325`) writes **two** mutations from one
`fn_target_position` call: `attrs.location_id` and `attrs.coordinates`. "Approach the bar" and
"walk to the docks" are the same operation — a same-scene move just writes `location_id` unchanged
in value. `apply_ruled_event` has the identical arm (`:591`). The target is `to_target_id`: any
positioned entity — a location is just a target that is a location.

`ObjectRelocated` routes to `fn_apply_carry_change` (`:874`): the new containment edge, then the
**eager rule** — recompute `carried_weight` recursively up the carry chain and write/clear
`encumbered` **in the same commit**, provenance-traced to the event.

## The journey loop

`core/api/journey.go` — read its header: a journey row is read **fresh from the table on every
input; never held in server memory across calls**. `startJourney` (only callers: the beat chain in
`core/api/orchestrator.go`) turns an over-budget attempt into a row; the one case it does not
swallow is the impossible move (speed 0 → `turn_budget`). Leg order and halt reasons:
`journey.go:326-347` — barred (an existing shut/locked connection is obeyed, never routed around)
→ interrupted → arrived → unresolved → `journey_leg`. Mid-journey place creation is
`core/api/placeauthor.go`, a file `DOMAINS.map` currently assigns to world-genesis (`seams.md`).

`core/api/mint.go` validates movement/spatial vocabulary mints: shape and derivable bounds only,
never plausibility — the three nets, not a numeric limit, guard a wild mint.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `D-11` | Record the generating structure, derive the quantity. | A stored distance/reachable/contains is two facts that can disagree. |
| `D-12` | Geometry only; reachability is move-validity's (`SPEC-017`), distance is ours (`SPEC-018`). | Wall-passing is a predicate change, not a speed modifier — it cannot emerge from the duration formula. |
| `D-1` | This path is LLM-free; the engine may compute only what `FINAL-action-contracts.md` grants. | A Go predicate added quietly is the failure mode; new physics is an amendment to that finite table. |
| `B-5`, `ADR-030` | Durations are logical ticks; wall-clock is telemetry. 1 tick = 1 second here. | `fn_world_now` is a read, not a clock you may advance — and time never rewinds when a journey ends. |
| `SPEC-026` | Superseded in part: the volume half survives; **`within-load` is dead as a blocker.** | Re-deriving a load constraint revives a superseded rule. |
| `SPEC-032` | Landed (PR #47) — and its recorded diagnosis was wrong (`failure-log` #35). | Building on a SPEC's *stated* cause without measuring repeats the lesson that cost real time. |

### What you may not decide alone

1. **A new physical computation.** The contracts document is finite and founder-locked; what is not
   in it routes to the resolution LLM. The deliverable is an amendment there, not a branch.
2. **Traversal without a portal.** Every crossing today presupposes a `connects` artifact —
   load-bearing and (per the area dossier) previously undocumented. Immaterial actors, teleporting:
   structural change, needs a ruling.
3. **The dials:** leg band, watch horizon, interruption chunk ticks. `SPEC-031`'s precedent — a
   felt-experience judgement, the founder's, answered by playing at a candidate value.
4. **Anything that lets geometry cross the seat boundary** (`place_author/1` — `seams.md`).

## Validation for this domain

pgTAP in `core/db/tests/`: `91_fn_actors_at*`, `92_` + `111_fn_move_duration*` (**both** — `92_`
pins the legacy wrapper, `111_` the status-aware formula; deleting either drops a contract),
`99_location_ids*`, `110_fn_distance*`, `112_fn_place_at*`, `113_extent_class*`,
`113_object_relocated*`, `114_portal_traversal*`, `114_fn_world_now*`, `115_journey_legs*`,
`115_mint_persistence*`, `116_drowned_lantern_space*`, `116_watch_horizon*`, `117_move_target*`.
Go: `cd core/api && go test -run 'Journey|Move|Space' -count=1 .` — always `-count=1` (the suite is
seed-dependent; a cached pass is stale). Never `make reset` (it destroys the dev volume —
`perception-and-knowledge.tech.md` Validation is the one home for that warning).

- **What counts as evidence here:** this domain fails *plausibly* — a wrong distance still renders
  as a walk, just the wrong length. Assert exact numbers against the seeded geometry
  (`116_drowned_lantern_space_test.sql` does: 8 m to the bar, 6 s at walk).
- **What counts as ceremony here:** a journey fixture that forgets to connect origin and goal —
  `TestJourney_LegMintsAWaystationTheTravellerCanStandOn` (`core/api/journey_beat_test.go`) exists
  because the SPEC-032 defect *only* appears when they are already connected; and any co-presence
  assertion against an actor with no `actor_state` row (nowhere ≠ broken gate).

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **A location's `attrs.coordinates` is its position in its PARENT, never a landing spot inside it.** Landing spots are `entry_point`. | `fn_target_position`'s location arm (`schema.sql:2870`); note originated in `20260729100006_move_target.sql`. |
| **A missing coordinate is silently `{0,0}`.** Unpositioned fixtures measure distance to the frame origin and pass wrong. | `fn_distance` COALESCEs (`schema.sql:1605-1608`). |
| **`fn_actors_at` reads `actor_state`.** An actor with no state row is nowhere. | `schema.sql:860`; same trap in `perception-and-knowledge.tech.md`. |
| **An unstamped room is an infinite budget.** No `tension` reads as `none` → `math.MaxInt64` → everything fits and the Journey is unreachable. | SPEC-030's quieter half (`docs/open-spec-items.md` §SPEC-030); `core/api/orchestrator.go:193`. |
| **Portals must never touch distance** (or vice versa) — a founder-locked boundary. | `20260729100004_portal_traversal.sql` header; `FINAL-action-contracts.md` §5.3. |
| **The two duration functions are not duplicates.** | `92_` vs `111_` — see Validation. |
| **`journey_acceptance_test.go` deletes live rows in cleanup**; most canon-writing tests do no cleanup at all. | `failure-log` #25; grep `DELETE FROM journey`. |
| **The design docs are gone from this repo.** The `docs/superpowers/` journey/living-world/rung files were consolidated away; R-numbered rulings survive in code comments (`journey.go`, `placeauthor.go`) and `digest/S06` §Topic 19. | `git ls-files '*journey-design*'` returns nothing, 2026-08-27; the area dossier's doc table predates this. |

## Open questions

1. **Can an actor be positioned *within* a place, or only *at* one?** Open and unsourced
   (`docs/00_workspace/closed-questions.md` § Unsourced): actors *carry* in-scene coordinates
   (Storage above) but co-presence is place-level binary, and `D-12` does not settle intra-place
   occlusion. Hitting this is deciding something new.
2. **Is two-dimensional distance a decision or an accident?** `fn_distance` reads `x`/`y` only;
   nothing anywhere rules on `z`. `[INFER]` — likely deliberate simplicity; no receipt exists.
3. **The area dossier contradicts the ledger on SPEC-034** (dossier: open; ledger and WE-3's
   package: landed 2026-08-25). Both recorded; a dossier amendment is a ruling-holder's call.
4. **Who owns `102_duration_class_test.sql` and `114_fn_world_now_test.sql`?** Built by this
   cluster's rungs, semantically Time's. Flagged in the proposed map for the moderator.
