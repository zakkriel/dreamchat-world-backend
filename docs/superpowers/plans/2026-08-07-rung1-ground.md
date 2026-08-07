# Rung 1 — Ground: places get an area (Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give places a real, drawable **area**, and make "which place contains this point?" a deterministic question the engine can answer — the ground the Journey stands on when it decides where a traveller is mid-trip.

**Architecture:** One shape language. The descriptive `attrs.extent` box is retired outright and replaced by `attrs.area`, an ordered outline of points in the place's own frame (founder ruling R12). A new `fn_place_at(world, frame, point)` returns the smallest-area child of a frame containing a point. A per-world `extent_class_metres` config turns an authored size class into a footprint the engine draws — the same author-picks-the-class / engine-owns-the-metres split already used for time. The mint bounds check moves from box comparison to point-in-outline.

**Tech Stack:** Go (`core/api`, package `main`), plpgsql + dbmate migrations, pgTAP. Test commands: `make reset`, `make test` (pgTAP), `cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./...`, `make schema-check`.

## Global Constraints

- **Branch:** `rung1/ground`, off `rung0/living-world-gates` (`28e64ce`) — rung 0 is in review as PR #29 and rung 1 builds on it. One PR.
- **Design source:** `docs/superpowers/specs/2026-08-07-journey-design.md` §4.5 and ruling R12. Read both before starting.
- **Clean cutover (repo law).** `attrs.extent` is **deleted**, not deprecated: no dual read path, no compatibility shim, no "fall back to the box". A second convention beside an existing one is prohibited.
- **No LLM ever draws geometry.** The author picks a size class; the engine computes the outline. Nothing in this rung adds a model call or a prompt that emits coordinates.
- **No Journey logic.** No progress, legs, thresholds, place creation, or stage resolution — those are rung 2. This rung only makes the ground answerable.
- **Nested frames stay deferred** (SPEC-018). `fn_place_at` answers within ONE frame. Do not add coordinate transforms.
- **D-1:** nothing here writes canon outside the existing apply paths. `fn_place_at` is a read-only measurement, recomputed at ask time — never a stored `contains` column (a stored answer rots the moment a place moves, the silent-corruption class this engine refuses).
- Run `make schema-check` after the migration task and commit the regenerated `core/db/schema.sql` with it.
- No project-wide formatters or linters. `go vet ./...` only.

**Confirmed facts about the real code (verified 2026-08-07):**

- **Nothing in SQL reads `attrs.extent`.** `fn_distance`, `fn_move_duration_actor`, `fn_target_position`, `fn_actor_move_permitted` and `fn_portal_permits` all work from `attrs.coordinates` plus `attrs.parent_location_id`. The seed says it outright: *"extent is descriptive"* (`core/db/seeds/seed_drowned_lantern.sql:270`).
- **Its only consumer is Go:** `validateArtifactMint` (`core/api/mint.go:186-226`) checks a minted `coordinate` lies inside `[0,w]×[0,h]` of `parentExtent`, carried **inline on the mint envelope** so validation stays DB-free. Types: `mintCoord{X,Y float64}` (`mint.go:47`), `mintExtent{W,H float64}` (`:52`), `mintEnvelope.Coordinate/*mintCoord`, `.ParentExtent *mintExtent` (`:70-71`).
- **`parentExtent` appears in NO prompt and NO published schema** — only `mint.go` and `mint_test.go`. The resolve prompt teaches no mint shapes at all, so this check is a latent guard today. Rung 2's place-creation is its first real user, which is exactly why it must be right now.
- **A place's `attrs.coordinates` is its position in its PARENT's frame** (`seed_drowned_lantern.sql:264-266`); `attrs.parent_location_id` is the edge. The Harbor Quarter (`210c0000-…-d0`) is the root; the tavern `{200,200}`, Dock Street `{207,200}`, alley `{200,240}`, cellar `{205,205}` are its children.
- **Every `{"w":2000,"h":2000}` fixture** to migrate: `core/db/seeds/seed_drowned_lantern.sql:283`; `core/db/tests/110_fn_distance_test.sql:23`, `111_fn_move_duration_test.sql:24`, `92_fn_move_duration_test.sql:17`, `95_apply_beat_happy_test.sql:16`, `96_apply_beat_partial_beat_test.sql:17`; `core/api/cognition_factsheet_test.go:49`, `orchestrator_test.go:45`, `query_test.go:68`, `resolve_factsheet_test.go:43`, `station_f_exit_test.go:87,147`, `tension_test.go:69`; plus the `parentExtent` envelopes in `mint_test.go:91,101`.
- **Config-table pattern to copy:** `core/db/migrations/20260805100001_duration_class_config.sql` — table + `fn_*` lookup with a built-in fallback, `seed_world_defaults` extended by copying its whole body verbatim and appending, and a down-migration that **restores `seed_world_defaults` FIRST** then drops the function and table (sql-language dependency order).
- **Current `seed_world_defaults` body** (copy verbatim into the new migration, then append the new INSERT): seeds `movement_type` walk 1.4; `status_modifier` encumbered/move/walk −100; five `duration_class_seconds` rows; three `world_actor_config` tiers (0.01 / 60·3600·86400 / 0.70); one `world_actor_setting` row.
- **Latest migration is `20260805100003_world_slice.sql`.** New ones this rung: `20260807100001_place_area.sql`, `20260807100002_extent_class.sql`.
- **Postgres does this natively:** `polygon '((0,0),(10,0),(10,10),(0,10))' @> point '(5,5)'` → true, and `area(polygon)` gives the size for the smallest-match ordering. No PostGIS.

---

### Task 1: `attrs.area` and `fn_place_at`

**Files:**
- Create: `core/db/migrations/20260807100001_place_area.sql`
- Create: `core/db/tests/112_fn_place_at_test.sql`
- Modify: `core/db/schema.sql` (regenerated by `make migrate`, committed with the migration)

**Interfaces:**
- Produces:
  - `fn_area_polygon(p_attrs jsonb) RETURNS polygon` — converts `attrs.area` (`{"points":[{"x":…,"y":…},…]}`) to a Postgres polygon; NULL when absent or fewer than 3 points.
  - `fn_place_at(p_world_id uuid, p_frame uuid, p_point jsonb) RETURNS uuid` — the smallest-area child of `p_frame` whose area contains the point; NULL when none does.

- [ ] **Step 1: Write the failing pgTAP test**

Create `core/db/tests/112_fn_place_at_test.sql`. Build an isolated world (do not lean on the seed — this must pass standalone, matching `110_fn_distance_test.sql`'s pattern): a root frame, and inside it a big region with a 4-point area, a small square wholly inside that region, and a room with NO area.

```sql
BEGIN;
SELECT plan(6);

-- Isolated fixture world: root frame R; inside it BIG (0,0)-(1000,1000) and SMALL (100,100)-(200,200),
-- SMALL entirely inside BIG. DOT has no area at all. Mirrors 110_fn_distance_test.sql's standalone style.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('fc000000-0000-0000-0000-0000000000r0','fc000000-ffff-0000-0000-000000000000','location','Root'),
  ('fc000000-0000-0000-0000-0000000000b1','fc000000-ffff-0000-0000-000000000000','location','Big'),
  ('fc000000-0000-0000-0000-0000000000s1','fc000000-ffff-0000-0000-000000000000','location','Small'),
  ('fc000000-0000-0000-0000-0000000000t1','fc000000-ffff-0000-0000-000000000000','location','Dot');

INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('fc000000-0000-0000-0000-0000000000r0','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0}}'::jsonb),
  ('fc000000-0000-0000-0000-0000000000b1','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0},"parent_location_id":"fc000000-0000-0000-0000-0000000000r0",
     "area":{"points":[{"x":0,"y":0},{"x":1000,"y":0},{"x":1000,"y":1000},{"x":0,"y":1000}]}}'::jsonb),
  ('fc000000-0000-0000-0000-0000000000s1','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":150,"y":150},"parent_location_id":"fc000000-0000-0000-0000-0000000000r0",
     "area":{"points":[{"x":100,"y":100},{"x":200,"y":100},{"x":200,"y":200},{"x":100,"y":200}]}}'::jsonb),
  ('fc000000-0000-0000-0000-0000000000t1','fc000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":500,"y":500},"parent_location_id":"fc000000-0000-0000-0000-0000000000r0"}'::jsonb);

-- (a) a point inside BIG only resolves to BIG
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000r0','{"x":800,"y":800}'::jsonb),
  'fc000000-0000-0000-0000-0000000000b1'::uuid,
  '(a) point inside only BIG resolves to BIG');

-- (b) a point inside BOTH resolves to the SMALLER area — the whole point of "smallest wins"
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000r0','{"x":150,"y":150}'::jsonb),
  'fc000000-0000-0000-0000-0000000000s1'::uuid,
  '(b) point inside BIG and SMALL resolves to SMALL (smallest area wins)');

-- (c) a point inside nothing is the open road
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000r0','{"x":5000,"y":5000}'::jsonb),
  NULL,
  '(c) point outside every area returns NULL — the open road');

-- (d) an arealess place never contains anybody, even standing on its exact coordinate
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000r0','{"x":500,"y":500}'::jsonb),
  'fc000000-0000-0000-0000-0000000000b1'::uuid,
  '(d) DOT has no area, so its own coordinate still resolves to BIG, never to DOT');

-- (e) a frame with no children yields NULL rather than erroring
SELECT is(
  fn_place_at('fc000000-ffff-0000-0000-000000000000','fc000000-0000-0000-0000-0000000000t1','{"x":0,"y":0}'::jsonb),
  NULL,
  '(e) a frame with no children returns NULL');

-- (f) fewer than 3 points is not a polygon and must not be treated as one
SELECT is(
  fn_area_polygon('{"area":{"points":[{"x":0,"y":0},{"x":1,"y":1}]}}'::jsonb),
  NULL,
  '(f) a 2-point area is NULL, not a degenerate polygon');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `make reset && make test 2>&1 | grep -A5 112_fn_place_at`
Expected: FAIL — `function fn_place_at(...) does not exist`.

- [ ] **Step 3: Write the migration**

Create `core/db/migrations/20260807100001_place_area.sql`:

```sql
-- migrate:up

-- Rung 1 (the Journey ladder) — PLACES GET AN AREA.
--
-- Founder ruling R12 (2026-08-07): the descriptive attrs.extent {"w","h"} box is RETIRED, not extended
-- ("that was a cheap solution… an area is the more real and even 'drawable if needed'"). Nothing in SQL
-- ever read it (fn_distance and friends work off coordinates + the parent edge; the seed itself called it
-- "descriptive"), so there is no dual form and no compatibility shim: attrs.area replaces it outright.
--
-- attrs.area = {"points":[{"x":…,"y":…}, …]} (≥3), an ordered outline in the place's OWN frame — the same
-- frame its attrs.coordinates live in, i.e. the parent's. Optional: a place with no area is a point and
-- contains nobody, which is every room that ships today.
--
-- Containment is a MEASUREMENT recomputed at ask time, never a stored `contains` column — a stored answer
-- rots the moment a place moves or grows (the silent-corruption class §0 refuses).

-- ── fn_area_polygon: attrs → polygon. NULL when there is no area or fewer than 3 points (2 points is a
--    line, not a footprint, and must not silently become a degenerate polygon). STABLE, pure.
CREATE FUNCTION public.fn_area_polygon(p_attrs jsonb) RETURNS polygon
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
    WHEN jsonb_array_length(COALESCE(p_attrs->'area'->'points', '[]'::jsonb)) >= 3
    THEN (
      SELECT ('(' || string_agg('(' || (pt->>'x') || ',' || (pt->>'y') || ')', ',' ORDER BY ord) || ')')::polygon
      FROM jsonb_array_elements(p_attrs->'area'->'points') WITH ORDINALITY AS t(pt, ord)
    )
    ELSE NULL
  END;
$$;

-- ── fn_place_at: which child of p_frame contains this point? The SMALLEST such area wins, so a square
--    inside a region resolves to the square. NULL = the point is inside nothing — the open road.
--
--    FRAME-SCOPED BY CONSTRUCTION (design §4.5): a place's coordinates are expressed in its PARENT's
--    frame, so a region and a room inside it are measured in different frames and cannot be compared in
--    one call. Callers pass the frame they are travelling in. Resolving THROUGH frames needs coordinate
--    transforms and belongs to the deferred spatial engine (SPEC-018) — do not add it here.
CREATE FUNCTION public.fn_place_at(p_world_id uuid, p_frame uuid, p_point jsonb) RETURNS uuid
LANGUAGE sql STABLE AS $$
  SELECT ls.entity_id
  FROM location_state ls
  WHERE ls.world_id = p_world_id
    AND (ls.attrs->>'parent_location_id')::uuid = p_frame
    AND fn_area_polygon(ls.attrs) IS NOT NULL
    AND fn_area_polygon(ls.attrs) @> point((p_point->>'x')::float8, (p_point->>'y')::float8)
  ORDER BY area(fn_area_polygon(ls.attrs)) ASC, ls.entity_id ASC
  LIMIT 1;
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_place_at(uuid, uuid, jsonb);
DROP FUNCTION IF EXISTS public.fn_area_polygon(jsonb);
```

- [ ] **Step 4: Apply and run the test**

Run: `make reset && make test 2>&1 | tail -5`
Expected: all pgTAP files pass, test count risen by 6.

If `area()` returns a negative number for a clockwise-wound polygon, do **not** flip the ordering — wrap it in `abs()` and say so in your report; winding order is authored data and both directions are legal outlines.

- [ ] **Step 5: Check schema drift and commit**

```bash
make schema-check
git add core/db/migrations/20260807100001_place_area.sql core/db/tests/112_fn_place_at_test.sql core/db/schema.sql
git commit -m "feat(ground): places get attrs.area; fn_place_at answers which place contains a point"
```

---

### Task 2: Retire the box

**Files:**
- Modify: `core/api/mint.go:47-74` (types), `:186-226` (`validateArtifactMint`), `:20-39` (the doc comment describing the shapes)
- Modify: `core/api/mint_test.go:87-105` (the two extent cases)
- Modify: `core/db/seeds/seed_drowned_lantern.sql:265-283` (the quarter's box → an area, and the comments describing it)
- Modify fixtures: `core/db/tests/110_fn_distance_test.sql:23`, `111_fn_move_duration_test.sql:24`, `92_fn_move_duration_test.sql:17`, `95_apply_beat_happy_test.sql:16`, `96_apply_beat_partial_beat_test.sql:17`; `core/api/cognition_factsheet_test.go:49`, `orchestrator_test.go:45`, `query_test.go:68`, `resolve_factsheet_test.go:43`, `station_f_exit_test.go:87,147`, `tension_test.go:69`

**Interfaces:**
- Consumes: `fn_area_polygon` from Task 1 (SQL side only; the Go check stays DB-free).
- Produces:
  - `type mintArea struct { Points []mintCoord `json:"points"` }` replacing `mintExtent`
  - `mintEnvelope.ParentArea *mintArea `json:"parentArea"`` replacing `.ParentExtent`
  - `func pointInOutline(p mintCoord, poly []mintCoord) bool` — ray-casting containment, pure, no dependency

- [ ] **Step 1: Write the failing test**

Rewrite the two extent cases in `core/api/mint_test.go` and add a containment unit test. The existing (g)/(g2) cases keep their meaning exactly — a coordinate outside its parent violates, inside passes — only the shape changes:

```go
// (g) a coordinate outside the parent AREA → violation (§3: a minted coordinate must lie within its
// parent). The envelope carries the parent's outline inline so validation stays DB-free.
func TestValidateMints_G_CoordinateOutsideArea(t *testing.T) {
	m := `{"locationId":"11111111-1111-1111-1111-111111111111","parentLocationId":"22222222-2222-2222-2222-222222222222","coordinate":{"x":5000,"y":0},"parentArea":{"points":[{"x":0,"y":0},{"x":2000,"y":0},{"x":2000,"y":2000},{"x":0,"y":2000}]}}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(g) coordinate x:5000 outside a 2000x2000 outline must violate; got pass")
	}
}

// (g2) a coordinate INSIDE the parent area → pass.
func TestValidateMints_G2_CoordinateInsideArea(t *testing.T) {
	m := `{"locationId":"11111111-1111-1111-1111-111111111111","parentLocationId":"22222222-2222-2222-2222-222222222222","coordinate":{"x":100,"y":50},"parentArea":{"points":[{"x":0,"y":0},{"x":2000,"y":0},{"x":2000,"y":2000},{"x":0,"y":2000}]}}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) != 0 {
		t.Fatalf("(g2) coordinate {100,50} inside the outline must PASS; got %v", v)
	}
}

// (g3) a NON-RECTANGULAR outline is the reason this replaced the box: an L-shape must reject a point in
// the notch that any bounding box would have accepted.
func TestPointInOutline_LShapeRejectsTheNotch(t *testing.T) {
	l := []mintCoord{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 4}, {X: 4, Y: 4}, {X: 4, Y: 10}, {X: 0, Y: 10}}
	if pointInOutline(mintCoord{X: 8, Y: 8}, l) {
		t.Fatalf("point {8,8} sits in the L's notch and must be OUTSIDE; a bounding box would wrongly accept it")
	}
	if !pointInOutline(mintCoord{X: 2, Y: 2}, l) {
		t.Fatalf("point {2,2} is inside the L and must be accepted")
	}
}
```

- [ ] **Step 2: Run and confirm failure**

Run: `cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... -run 'TestValidateMints_G|TestPointInOutline' -v`
Expected: FAIL to compile — `pointInOutline` undefined, and `parentArea` is not a field.

- [ ] **Step 3: Replace the type and the check**

In `core/api/mint.go`, delete `mintExtent` and add:

```go
// mintArea is a place's outline — an ordered ring of points in its own frame. It replaces the retired
// {w,h} box (founder ruling R12): the box could not describe a road, a shoreline, or anything else that
// is not a rectangle, and a bounding box around such a place claims ground the place does not occupy.
type mintArea struct {
	Points []mintCoord `json:"points"`
}

// pointInOutline reports whether p lies inside the closed ring poly, by ray casting: count the edges a
// ray cast in +x crosses; odd means inside. Points ON an edge are not guaranteed either way — a boundary
// coordinate is a degenerate mint regardless, and the caller treats a false as a violation. Pure: no
// database, no dependency, so mint validation stays DB-free as designed.
func pointInOutline(p mintCoord, poly []mintCoord) bool {
	if len(poly) < 3 {
		return false
	}
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		pi, pj := poly[i], poly[j]
		if (pi.Y > p.Y) != (pj.Y > p.Y) &&
			p.X < (pj.X-pi.X)*(p.Y-pi.Y)/(pj.Y-pi.Y)+pi.X {
			inside = !inside
		}
	}
	return inside
}
```

Change `mintEnvelope.ParentExtent *mintExtent` to `ParentArea *mintArea `json:"parentArea"``, and replace the bounds check in `validateArtifactMint` (currently `mint.go:218-226`) with:

```go
	// coordinate within the parent's AREA (§3). Validated only when BOTH are present (the outline is
	// carried inline so this stays DB-free).
	if e.Coordinate != nil && e.ParentArea != nil {
		if !pointInOutline(*e.Coordinate, e.ParentArea.Points) {
			out = append(out, fmt.Sprintf(
				"mint %d: coordinate {%g,%g} outside the parent's area", i, e.Coordinate.X, e.Coordinate.Y))
		}
	}
```

Update the file's doc comment (`mint.go:20-39`) so its description of the shapes matches: the artifact/place mint carries `parentArea` (an outline), not `parentExtent {w,h}`.

- [ ] **Step 4: Run and confirm the tests pass**

Run: `cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... -run 'TestValidateMints|TestPointInOutline' -v`
Expected: PASS, including the L-shape notch case.

- [ ] **Step 5: Migrate the seed**

In `core/db/seeds/seed_drowned_lantern.sql`, replace the Harbor Quarter's extent write (`:283`) with the equivalent outline, and update the two comments that describe it (`:265`, `:270`, `:278`):

```sql
 ('22222222-2222-2222-2222-222222222222','2e000000-0000-0000-0000-0000000000f9','210c0000-0000-0000-0000-0000000000d0','location','attrs.area',
  '{"points":[{"x":0,"y":0},{"x":2000,"y":0},{"x":2000,"y":2000},{"x":0,"y":2000}]}'::jsonb, 40,27),
```

The comment at `:270` currently reads *"extent is descriptive"* — that is no longer true, since `fn_place_at` reads the area. Say so: the area is engine-read (Tier-1), the outline bounds the quarter's children, and the four rooms sit inside it.

- [ ] **Step 6: Migrate the remaining fixtures**

Replace every `'extent':{'w':2000,'h':2000}` with the four-corner area, in each file listed under **Files** above. These are mechanical, one-line-each edits. Do NOT change any coordinate, assertion, or expected value — the outline is the same 2000×2000 square the box described, so every distance and duration assertion must hold unchanged. If one moves, stop and report.

- [ ] **Step 7: Prove the box is gone**

```bash
grep -rn '"extent"\|attrs.extent\|parentExtent\|mintExtent' core/ && echo "STILL PRESENT — not a clean cutover" || echo "box retired"
```
Expected: `box retired`.

- [ ] **Step 8: Full suite and commit**

```bash
make reset && make test && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../..
git add core/api/mint.go core/api/mint_test.go core/db/seeds/seed_drowned_lantern.sql core/db/tests core/api/*_test.go
git commit -m "refactor(ground): retire the {w,h} box for attrs.area outlines (clean cutover, R12)"
```

---

### Task 3: Size classes — the engine draws the footprint

**Files:**
- Create: `core/db/migrations/20260807100002_extent_class.sql`
- Create: `core/db/tests/113_extent_class_test.sql`
- Modify: `core/db/schema.sql` (regenerated)

**Interfaces:**
- Produces:
  - table `extent_class_metres(world_id uuid, class text, radius_m numeric, PRIMARY KEY (world_id, class))`, class ∈ `intimate|small|medium|large|vast`
  - `fn_extent_class_metres(p_world_id uuid, p_class text) RETURNS numeric` — table lookup with a built-in fallback
  - `fn_area_around(p_centre jsonb, p_radius numeric) RETURNS jsonb` — an 8-point regular outline centred on the point, ready to write straight into `attrs.area`
  - `seed_world_defaults` extended with the five class rows

**Why classes and not numbers:** the author of a new place says *what* it is; the engine owns what that means in metres. Identical split to `duration_class_seconds`, and it is what keeps geometry out of the model's hands (design §4.5, R3).

Class names are genre-agnostic per rule GA-2 — `intimate|small|medium|large|vast` must read sensibly in a sci-fi thriller, a workplace drama, and a horror story. Do not use `hamlet`/`town`/`city`.

- [ ] **Step 1: Write the failing pgTAP test**

Create `core/db/tests/113_extent_class_test.sql`:

```sql
BEGIN;
SELECT plan(5);

-- (a) an UNSEEDED world must fall back, never return NULL — the lookup never fails closed
-- (the fn_duration_class_seconds lesson: an unconfigured world returned NULL and broke the caller).
SELECT ok(
  fn_extent_class_metres('fd000000-ffff-0000-0000-000000000000','small') > 0,
  '(a) an unseeded world falls back to a positive radius, not NULL');

-- (b) a table row WINS over the fallback — proven with a radius no fallback would ever produce.
INSERT INTO extent_class_metres (world_id, class, radius_m)
VALUES ('fd000000-ffff-0000-0000-000000000000','small',12345);
SELECT is(
  fn_extent_class_metres('fd000000-ffff-0000-0000-000000000000','small'),
  12345::numeric,
  '(b) the per-world row overrides the built-in fallback exactly');

-- (c) the classes are strictly increasing — otherwise "bigger place" is meaningless.
SELECT ok(
  fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','intimate')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','small')
  AND fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','small')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','medium')
  AND fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','medium')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','large')
  AND fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','large')
    < fn_extent_class_metres('fd000000-ffff-0000-0000-000000000001','vast'),
  '(c) intimate < small < medium < large < vast');

-- (d) the engine draws an 8-point outline around the centre.
SELECT is(
  jsonb_array_length(fn_area_around('{"x":100,"y":100}'::jsonb, 50)->'points'),
  8,
  '(d) fn_area_around draws an 8-point outline');

-- (e) THE ROUND TRIP: the drawn outline is a polygon fn_area_polygon accepts, and it contains the
-- centre it was drawn around. This is what rung 2 depends on — a created place must contain the
-- traveller standing at the point it was created for.
SELECT ok(
  fn_area_polygon(jsonb_build_object('area', fn_area_around('{"x":100,"y":100}'::jsonb, 50)))
    @> point(100,100),
  '(e) the engine-drawn footprint contains its own centre');

SELECT * FROM finish();
ROLLBACK;
```

Assertion (e) is the one that matters: it proves the engine-drawn footprint is a real polygon the
containment function accepts, which is precisely what rung 2 will depend on.

- [ ] **Step 2: Run it and confirm it fails**

Run: `make reset && make test 2>&1 | grep -A5 113_extent_class`
Expected: FAIL — the function does not exist.

- [ ] **Step 3: Write the migration**

Follow `20260805100001_duration_class_config.sql` exactly in structure: create the table; create `fn_extent_class_metres` with a `COALESCE(table, CASE fallback)`; create `fn_area_around`; then `CREATE OR REPLACE FUNCTION seed_world_defaults` **copying the current body verbatim** (movement_type, status_modifier, five duration_class rows, three world_actor_config tiers, one world_actor_setting) and appending the five `extent_class_metres` rows.

Suggested defaults, retunable data rather than truth: `intimate` 5 m, `small` 50 m, `medium` 200 m, `large` 1000 m, `vast` 5000 m.

`fn_area_around` draws 8 points at `radius` from the centre — regular, deterministic, no randomness:

```sql
CREATE FUNCTION public.fn_area_around(p_centre jsonb, p_radius numeric) RETURNS jsonb
LANGUAGE sql IMMUTABLE AS $$
  SELECT jsonb_build_object('points', jsonb_agg(
           jsonb_build_object(
             'x', round(((p_centre->>'x')::numeric + p_radius * cosd(45 * g))::numeric, 3),
             'y', round(((p_centre->>'y')::numeric + p_radius * sind(45 * g))::numeric, 3))
           ORDER BY g))
  FROM generate_series(0, 7) AS g;
$$;
```

The down-migration **restores the previous `seed_world_defaults` first** (the body as it stands before this migration), then drops the two functions and the table — the dependency order the Living World migrations already learned the hard way.

- [ ] **Step 4: Apply, test, check drift, commit**

```bash
make reset && make test
make schema-check
git add core/db/migrations/20260807100002_extent_class.sql core/db/tests/113_extent_class_test.sql core/db/schema.sql
git commit -m "feat(ground): extent size classes and the engine-drawn footprint"
```

---

### Task 4: Rung gate

**Files:** none — verification only.

- [ ] **Step 1: Full battery from a clean database**

```bash
make reset && make test && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../.. && make schema-check
```
Expected: pgTAP up from 362 by the new assertions, Go suite green, vet clean, no schema drift.

- [ ] **Step 2: Re-run the Go suite with no reset in between**

```bash
cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./...
```
Expected: PASS twice — proves nothing leaks state.

- [ ] **Step 3: Prove the cutover is clean**

```bash
grep -rn '"extent"\|attrs.extent\|parentExtent\|mintExtent' core/ && echo "FAIL: box survives" || echo "one shape language"
```

- [ ] **Step 4: Record the rung in the ledger**

Append a `# RUNG 1 COMPLETE` entry to `.git/sdd/progress.md` in the style of the existing entries: the ruling it implements (R12), the two migrations, the retired field, and the gate output.

- [ ] **Step 5: Open the PR**

Base it on `rung0/living-world-gates` (or on `feat/living-world` if rung 0 has merged by then — check before opening). Body: what an area is and why the box went, quoting R12; the frame-scoped containment rule and why deeper nesting is still deferred; the class→metres split; and the gate output. Cite D-1 (containment is a measurement, never stored) and design §4.5.
