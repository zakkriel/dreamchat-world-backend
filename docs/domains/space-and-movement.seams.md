# space-and-movement · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-5 · Space, movement and the journey ·
**Parent bounded context:** World Engine

A seam belongs to two domains; each row declares an expectation — one side owns a fact, the other
consumes it and must not re-derive or re-decide it. Neighbouring packages are being written in the
same round; where the two sides' wordings differ, the moderator reconciles.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Time & clock** (WE-6) | the beat's tension-derived budget, in seconds | Time owns tension and the budget ladder; we only supply the duration compared against it (the comparison runs in the play loop's chain). We never read tension directly. |
| consumes | **Play loop** (WE-7) | the attempt (`ActorMoved`/`ObjectRelocated` are passthrough — they never hit resolve) and the vocabulary mints inside rulings | Mints are validated by `core/api/mint.go` (shape + derivable bounds, never plausibility). We never invent vocabulary; the LLM never touches the grammar. |
| consumes | **Canon spine** (WE-1) | the commit | The doors own committing; we define *what a move writes* (`tech.md` §The write path) and compute the gate inputs. `startJourney` commits nothing; only arrival is an event. |
| consumes | **Living world** | the eruption fire decision per leg | Pressure and tiers are the World Actor's; a journey only *offers* the world its chance (one per leg) and ends `journey_interrupted` when told. We never compute eruption chance. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Perception** (WE-3) | `fn_actors_at(world, place)` — place-level, **binary** co-presence | We own the predicate; perception decides what co-presence *means*. Perception must not re-derive presence, nor invent a sub-place answer to "could they see it from there" — that absence is our open question (`tech.md` §Open 1). Matches WE-3's consumes-row wording. |
| provides | **Play loop** (WE-7) | computed spatial facts: `fn_distance`, `fn_move_duration_actor`, `fn_portal_permits`, `fn_actor_move_permitted`, volume/weight | Seats reason FROM computed truth — the fact-sheet gather (WE-7's) calls these functions, never reimplements them. `premiseHolds` mirrors `fn_actor_move_permitted` deliberately; keep the mirror, never fork it. |
| provides | **Time & clock** (WE-6) | `journey.current_tick` as an input to `fn_world_now` | Quiet legs move the clock with no filler canon event. The row STAYS when a journey ends — time never rewinds (`B-5`). Whether `fn_world_now` files under Time or here is flagged for the moderator. |
| provides | **World genesis** (WE-10) | the mid-journey trigger for place creation (R2: "are you somewhere known?") | The journey decides *when* and *where*; the seat authors identity only — **the engine draws the footprint; the seat cannot emit geometry even if it tries** (`place_author/1`, schema title). `DOMAINS.map` files `placeauthor*` under world-genesis — overlap recorded in the proposed map. |
| provides | **Compendium surfaces** (UX-1) | carry state (`contained_by`, `encumbered`) via the projections | The Carrying overlay lists possession; it never recomputes weight, volume or the eager rule. `carrying.v1` is vendored by `dream-weaver-visuals` — a change is a cross-repo round. |
| provides | **Presentation** (frontend) | the journey block — **labels only**: `{active, kind, goal_label, progress, legs_done, legs_total, where_label, interruptible, status}` | No coordinate, radius or distance number reaches a player surface (`D-7`). |

## The seams that do not exist

- **Physics / occlusion.** Sub-place position exists as data (actors carry in-scene coordinates)
  but no function answers "could they see it from there." `ADR-P025` routes concealment to Physics;
  the geometric half would land here. Do not improvise either half — two domains' open questions
  meet at this seam.
- **Frame transforms.** Comparing coordinates across frames (a room's point in its region's frame)
  is the deferred spatial engine (`SPEC-018`). Callers pass the frame they travel in; nothing
  resolves through frames today.
- **NPC journeys.** The row is actor-keyed but only the player's beat chain starts one
  (`product.md` §not built). An NPC that must travel today simply is not modelled as travelling.
