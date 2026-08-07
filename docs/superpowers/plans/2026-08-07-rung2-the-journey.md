# Rung 2 — The Journey (Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An action that exceeds the beat's budget stops being a dead end and becomes a **journey** — a span split into legs, the world taking its turn at every one, the actor arriving only if nothing stopped them.

**Architecture:** The journey is its own unit, not a patch inside the beat loop. Loop state lives in a `journey` row (the `held_outcome` precedent). Two lines in `runChain` hand an over-budget attempt to it; everything else — spans, legs, thresholds, stage resolution, arrival — lives in `journey.go` and its neighbours. The world's-turn composer is called once per leg, **unchanged**, exactly as its docstring promised. A new `place_author` seat authors ground that does not exist yet; the engine draws its geometry.

**Tech Stack:** Go (`core/api`, package `main`), plpgsql + dbmate migrations, pgTAP, go:embed for prompts/schemas. Tests: `make reset && make test`; `cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./...`; `make schema-check`.

## Global Constraints

- **Branch:** `rung2/the-journey`, off `rung1/ground`. One PR.
- **Design source:** `docs/superpowers/specs/2026-08-07-journey-design.md` §4.1–4.7 and rulings R2–R13. Read them before starting; the rulings are the founder's words and are not open for reinterpretation.
- **The world's-turn composer is called, never modified.** `runWorldTurn` (`worldturn.go:39`) takes `(worldID, scene, tickBefore, tickAfter, seq, outcome, trace)` and needs zero changes. If you find yourself editing it, stop — the boundary is wrong.
- **The beat loop gains two call sites and nothing else.** No leg, span, threshold, or progress logic in `orchestrator.go`.
- **Journey state is loop state, not canon** (like `held_outcome`, migration `20260724110004`): plain INSERT/UPDATE from Go, rows deletable, no append-only guard. Canon still flows only through the SQL apply paths (D-1).
- **Read journey state fresh from the table on every input** — no server memory, no session object. `pendingHeldOutcomes` (`orchestrator.go:457`) is the pattern.
- **No mutable domain time (B-5).** The journey carries ticks, never a wall-clock column.
- **The engine draws all geometry.** The `place_author` seat authors identity and a size *class*; `fn_area_around` + `fn_extent_class_metres` (rung 1) turn that into an outline. A seat that emits coordinates is a bug.
- **Creation fills gaps only (R4).** Where a connection already exists and is shut or locked, it is obeyed. Never create a way past a barrier the world already has.
- **Pressure is untouched (R8).** No tension term, no change to `fn_pressure_chance` or `pressure.go`.
- No project-wide formatters or linters; `go vet ./...` only. Commit the regenerated `schema.sql` with each migration.

**A stated deviation, not an oversight.** Tasks 1–3 carry literal test code. Tasks 4–9 name each test and
state exactly what it must prove, but leave the bodies to their implementer. This is deliberate: those
tests exercise a `journey` row and a leg loop whose real shape Tasks 1–4 establish, and speculative
bodies written now would mostly be fiction that the implementer has to unpick. The obligations still
bind — each named test must exist, must fail first for the stated reason, and its real failing output
must appear in the implementer's report. An implementer who cannot make a named test fail first should
say so rather than write one that passes on arrival.

**Confirmed interfaces (consume by exact name):**

- `func (o *Orchestrator) runChain(ctx, worldID, actorID string, chain []Attempt, startTick int64, startSeq int, budgetRemaining int64, outcome *BeatOutcome, trace *BeatTrace) error` — `orchestrator.go:118`. The two over-budget gates are `:258` (move) and `:328` (non-move).
- `func overBudget(dur, budgetRemaining, curTick int64) bool` — `orchestrator.go:420`.
- `func (o *Orchestrator) runWorldTurn(ctx, worldID, scene string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, seqUsed int, err error)` — `worldturn.go:39`; `eruptionCutsBeat(mag string) bool` — `worldturn.go:175`.
- `func (o *Orchestrator) fnMoveDurationActor(ctx, worldID, actorID, target string) (int64, error)` — `orchestrator.go:1045`; `fnTargetScene` — `:1060`; `actorLocation` — `:992`.
- `func (o *Orchestrator) applyEvent(ctx, worldID, actorID string, attemptJSON []byte, tick int64, seq int) (map[string]any, error)` — `orchestrator.go:1094`; `commitWorldPayload(..., postCommit postCommitFn, outcome, trace)` — `ledger.go:54`; `type postCommitFn` — `ledger.go:30`.
- `func (o *Orchestrator) RunBeat(ctx, worldID, actorID string, chain []Attempt, startTick int64, trace *BeatTrace) (BeatOutcome, error)` — `orchestrator.go:83`.
- `type Attempt struct` — `beatseats.go:56` (`Type`, `Stated`, `ToTargetID`, `ListenerID`, `Content`, `ObjectID`, …, `DurationClass`); `func validateAttemptFields(i int, a Attempt) error` — `beatseats.go:100`; `var beatChainV2SchemaJSON` — `beatvocab.go:8`; `allowedBeatTypesV2` — `beatvocab.go:61`.
- Seat pattern: `SeatWorldActor = Seat{Name: "world_actor", Requires: []Capability{CapStructuredOutput}}` — `bridge.go:82`; registered in `main.go:49` and `beathandler.go:157`; fake in `bridge_fakes.go:138`.
- Rung 1 SQL: `fn_place_at(world uuid, frame uuid, point jsonb) RETURNS uuid`, `fn_area_polygon(attrs jsonb) RETURNS polygon`, `fn_extent_class_metres(world uuid, class text) RETURNS numeric`, `fn_area_around(centre jsonb, radius numeric) RETURNS jsonb`.
- Beat start tick is computed inline at `beathandler.go:161-163` — Task 1 replaces that query.
- Latest migration: `20260807100002_extent_class.sql`. This rung adds `20260807100003_journey.sql` and `20260807100004_journey_legs.sql`.

---

### Task 1: The journey row and a clock that includes it

**The problem this solves.** World-time is derived from committed events (`max(in_world_tick)`). A quiet leg commits nothing, so hours of travel would not move the clock — and since eruption pressure is driven *entirely* by elapsed world-time (R8), the world could never interrupt a journey. That would gut the feature.

**Files:**
- Create: `core/db/migrations/20260807100003_journey.sql`, `core/db/tests/114_fn_world_now_test.sql`
- Modify: `core/api/beathandler.go:160-166` (use the function instead of the inline query), `core/db/schema.sql` (regenerated)

**Interfaces:**
- Produces: table `journey`; `fn_world_now(p_world_id uuid) RETURNS bigint`.

- [ ] **Step 1: Write the failing pgTAP test**

Create `core/db/tests/114_fn_world_now_test.sql`:

```sql
BEGIN;
SELECT plan(4);

-- (a) a world with nothing at all is at tick 0, not NULL — the caller adds 1 and starts at 1.
SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 0::bigint,
  '(a) an empty world is at 0, never NULL');

-- A journey row alone moves the clock: this is the whole point — quiet legs commit nothing.
INSERT INTO journey (journey_id, world_id, actor_id, kind, threshold, span_seconds,
                     legs_total, legs_done, started_tick, current_tick, status)
VALUES ('fe000000-0000-0000-0000-00000000000a','fe000000-ffff-0000-0000-000000000000',
        'fe000000-0000-0000-0000-00000000000b','wait','{"kind":"tick","at":900}'::jsonb,
        900, 5, 2, 100, 460, 'active');

SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 460::bigint,
  '(b) a journey mid-flight carries the clock even though it has committed nothing');

-- (c) the later of the two wins — a canon event past the journey raises it.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('fe000000-0000-0000-0000-00000000000c','fe000000-ffff-0000-0000-000000000000',
        'AttributeChanged','probe',900,0,'accepted',now(),'public','freeform');
SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 900::bigint,
  '(c) canon past the journey wins — the clock is the later of the two');

-- (d) an ENDED journey still holds its tick: time must never rewind when a journey stops.
UPDATE journey SET status='ended', current_tick=1500
 WHERE journey_id='fe000000-0000-0000-0000-00000000000a';
SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 1500::bigint,
  '(d) an ended journey still holds the clock forward — time never rewinds');

SELECT * FROM finish();
ROLLBACK;
```

Assertion (d) is load-bearing: if `fn_world_now` filtered to `status='active'`, ending a journey would rewind world-time, and B-5 forbids that.

- [ ] **Step 2: Run it and confirm it fails**

Run: `make reset && make test 2>&1 | grep -A5 114_fn_world_now`
Expected: FAIL — relation `journey` does not exist.

- [ ] **Step 3: Write the migration**

Create `core/db/migrations/20260807100003_journey.sql`:

```sql
-- migrate:up

-- Rung 2 — THE JOURNEY's loop state, and a clock that can see it.
--
-- A journey is LOOP STATE, not canon — the same standing as held_outcome (20260724110004): Go writes it
-- with plain INSERT/UPDATE, rows may be deleted, and there is no append-only guard. Canon still flows
-- only through the apply twins (D-1). What makes it different from held_outcome is lifespan: a held
-- outcome resolves on the next input, a journey spans many.

CREATE TABLE journey (
  journey_id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id     uuid NOT NULL,
  actor_id     uuid NOT NULL,
  kind         text NOT NULL CHECK (kind IN ('travel','wait','watch')),
  threshold    jsonb NOT NULL,           -- the test run at each leg's end; shape per kind (design §4.4)
  span_seconds bigint NOT NULL CHECK (span_seconds > 0),
  legs_total   int NOT NULL CHECK (legs_total > 0),
  legs_done    int NOT NULL DEFAULT 0 CHECK (legs_done >= 0),
  started_tick bigint NOT NULL,
  current_tick bigint NOT NULL,
  frame_id     uuid,                     -- travel: the frame the trip happens in (design §4.5)
  origin_coord jsonb,                    -- travel: where it started, for interpolation
  goal_coord   jsonb,                    -- travel: where it ends
  goal_target  uuid,                     -- travel: the entity being walked to; the arrival commit's target
  stage_id     uuid,                     -- the place currently containing the traveller; NULL = open road
  status       text NOT NULL DEFAULT 'active' CHECK (status IN ('active','arrived','ended'))
);

-- ONE active journey per actor. A second would make "the journey" ambiguous at every decision point, so
-- the database refuses it rather than leaving Go to remember.
CREATE UNIQUE INDEX idx_journey_one_active ON journey (world_id, actor_id) WHERE status = 'active';

-- The next-input lookup scans a world's active journeys; a partial index keeps it to the live rows.
CREATE INDEX idx_journey_active ON journey (world_id) WHERE status = 'active';

-- ── fn_world_now: the world's clock, INCLUDING journeys in flight.
--
-- World-time was derived from committed events alone. A quiet leg commits nothing, so a journey's hours
-- would not move the clock — and eruption pressure is driven entirely by elapsed world-time, so the
-- world could never interrupt a journey. Rather than write filler canon events for "nothing happened",
-- the clock reads the later of the two sources.
--
-- Journeys are NOT filtered by status: an ended journey still holds its tick, because time must never
-- rewind when one stops (B-5, append-only time).
CREATE FUNCTION public.fn_world_now(p_world_id uuid) RETURNS bigint
LANGUAGE sql STABLE AS $$
  SELECT GREATEST(
    COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id = p_world_id), 0),
    COALESCE((SELECT max(current_tick)  FROM journey     WHERE world_id = p_world_id), 0)
  );
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_world_now(uuid);
DROP TABLE IF EXISTS journey;
```

- [ ] **Step 4: Point the beat at the new clock**

In `core/api/beathandler.go`, replace the inline start-tick query (`:161-163`) with:

```go
	var startTick int64
	if err := h.pool.QueryRow(ctx, `SELECT fn_world_now($1::uuid)+1`, worldID).Scan(&startTick); err != nil {
		http.Error(w, "start tick", http.StatusInternalServerError)
		return
	}
```

Search for any other place computing `max(in_world_tick)+1` as a start tick and point it here too:
`cd core/api && grep -n 'max(in_world_tick)' *.go`. Test helpers that compute their own base tick (e.g. `wtBaseTick`) are fixtures, not the engine — leave them.

- [ ] **Step 5: Verify, check drift, commit**

```bash
make reset && make test
make schema-check
cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../..
git add core/db/migrations/20260807100003_journey.sql core/db/tests/114_fn_world_now_test.sql core/db/schema.sql core/api/beathandler.go
git commit -m "feat(journey): journey loop state and a clock that counts journeys in flight"
```

---

### Task 2: The leg count is data

**Files:**
- Create: `core/db/migrations/20260807100004_journey_legs.sql`, `core/db/tests/115_journey_legs_test.sql`
- Modify: `core/db/schema.sql`

**Interfaces:**
- Produces: table `journey_legs_band(world_id, max_span_seconds, legs)`; `fn_journey_legs(p_world_id uuid, p_span_seconds bigint) RETURNS int`; `seed_world_defaults` extended.

**The rule (R7):** a journey is 5–10 presses, whatever its length — the low end for short trips, the high end for long hauls. The *risk* per press is what scales, and that already happens on its own: pressure climbs with elapsed world-time, so a leg covering eighteen hours sits at the cap while a leg covering ninety seconds barely moves the needle.

- [ ] **Step 1: Write the failing pgTAP test**

Create `core/db/tests/115_journey_legs_test.sql` with 4 assertions:

```sql
BEGIN;
SELECT plan(4);

-- (a) a short hop lands at the low end of the band
SELECT is(fn_journey_legs('ff000000-ffff-0000-0000-000000000000', 900::bigint), 5,
  '(a) a 15-minute span is 5 legs');

-- (b) a multi-day haul lands at the high end
SELECT is(fn_journey_legs('ff000000-ffff-0000-0000-000000000000', 864000::bigint), 10,
  '(b) a ten-day span is 10 legs');

-- (c) EVERY span stays inside the 5..10 band — the promise the player experiences
SELECT ok(
  (SELECT bool_and(n BETWEEN 5 AND 10) FROM (
     SELECT fn_journey_legs('ff000000-ffff-0000-0000-000000000000', s) AS n
     FROM unnest(ARRAY[1,60,900,3600,86400,864000,31536000]::bigint[]) AS s) t),
  '(c) every span from a second to a year yields between 5 and 10 legs');

-- (d) it is data: a per-world row overrides the fallback
INSERT INTO journey_legs_band (world_id, max_span_seconds, legs)
VALUES ('ff000000-ffff-0000-0000-000000000000', 900, 6);
SELECT is(fn_journey_legs('ff000000-ffff-0000-0000-000000000000', 900::bigint), 6,
  '(d) a per-world band row overrides the built-in fallback');

SELECT * FROM finish();
ROLLBACK;
```

- [ ] **Step 2: Run and confirm it fails** — `make reset && make test 2>&1 | grep -A5 115_journey_legs`. Expected: function does not exist.

- [ ] **Step 3: Write the migration**

Follow `20260805100001_duration_class_config.sql` exactly in structure. The table holds bands (`max_span_seconds` = the upper bound the row applies to); the function takes the smallest band whose bound the span fits, falling back to a built-in ladder, and **clamps the result to 5..10 whatever the data says** so a bad row can never produce a 400-press journey. Built-in fallback: ≤1 h → 5, ≤1 day → 7, otherwise 10. Extend `seed_world_defaults` by copying its current body verbatim (movement_type, status_modifier, five duration_class rows, three world_actor_config tiers, one world_actor_setting, five extent_class_metres rows) and appending the three band rows. The down-migration restores `seed_world_defaults` **first**, then drops the function and table.

- [ ] **Step 4: Verify, check drift, commit**

```bash
make reset && make test && make schema-check
git add core/db/migrations/20260807100004_journey_legs.sql core/db/tests/115_journey_legs_test.sql core/db/schema.sql
git commit -m "feat(journey): leg count is per-world data, clamped to the 5-10 band"
```

---

### Task 3: "for two hours" and "until the ship is in"

**Files:**
- Modify: `core/api/schema/beat_chain.v2.schema.json`, `core/api/beatseats.go:56` (Attempt) and `:100` (`validateAttemptFields`), `core/api/prompts/decompose.txt`
- Test: `core/api/beatvocab_v2_test.go` (or wherever `DecodeAndValidateChainV2` is exercised — `grep -rn DecodeAndValidateChainV2 *_test.go`)

**Interfaces:**
- Produces: `type Sustain struct` and `Attempt.Sustain *Sustain` (json `sustain,omitempty`).

```go
// Sustain is the "until/for <condition>" parse-shape (design §4.4) — a SHAPE the decomposer recognises,
// like QUERY, never a judgment about how long something ought to take. Exactly one kind:
//
//	{"kind":"for",        "seconds":7200}                              — a span the player STATED (R13)
//	{"kind":"until_at",   "entity_id":…, "place_id":…}                 — that thing, at that place
//	{"kind":"until_attr", "entity_id":…, "attr":"open", "value":"true"} — that thing, in that state
//
// seconds is passed through, NOT classified: reading "two hours" back as 7200 is parsing, the same act as
// binding a name to an id. duration_class stays what it is — the ladder for acts with an inherent length,
// whose cap exists so an UNSTATED length is never invented.
type Sustain struct {
	Kind     string `json:"kind"`
	Seconds  int64  `json:"seconds,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
	PlaceID  string `json:"place_id,omitempty"`
	Attr     string `json:"attr,omitempty"`
	Value    string `json:"value,omitempty"`
}
```

- [ ] **Step 1: Write the failing tests**

```go
func TestDecodeChainV2_Sustain(t *testing.T) {
	forOK := `[{"type":"AttributeChanged","stated":"I lie hidden for two hours","target_id":"11111111-1111-1111-1111-111111111111","sustain":{"kind":"for","seconds":7200}}]`
	chain, err := DecodeAndValidateChainV2(forOK)
	if err != nil {
		t.Fatalf("valid sustain rejected: %v", err)
	}
	if chain[0].Sustain == nil || chain[0].Sustain.Seconds != 7200 {
		t.Fatalf("sustain not decoded: %+v", chain[0].Sustain)
	}

	// A stated span far past the duration_class ceiling is exactly what this shape exists for (R13).
	century := `[{"type":"AttributeChanged","stated":"I wait a hundred years","target_id":"11111111-1111-1111-1111-111111111111","sustain":{"kind":"for","seconds":3153600000}}]`
	if _, err := DecodeAndValidateChainV2(century); err != nil {
		t.Fatalf("a stated century must decode — the class cap does not apply to sustain: %v", err)
	}

	// A move never sustains: its length is physics, and the schema forbids extra fields on ActorMoved.
	move := `[{"type":"ActorMoved","stated":"I walk home","to_target_id":"11111111-1111-1111-1111-111111111111","sustain":{"kind":"for","seconds":60}}]`
	if _, err := DecodeAndValidateChainV2(move); err == nil {
		t.Fatalf("ActorMoved with sustain must be rejected")
	}

	// Kind-specific required fields are enforced, not merely declared.
	bad := `[{"type":"AttributeChanged","stated":"I wait","target_id":"11111111-1111-1111-1111-111111111111","sustain":{"kind":"until_at","entity_id":"11111111-1111-1111-1111-111111111111"}}]`
	if _, err := DecodeAndValidateChainV2(bad); err == nil {
		t.Fatalf("until_at without place_id must be rejected")
	}

	// A non-positive span is not a wait.
	zero := `[{"type":"AttributeChanged","stated":"I wait","target_id":"11111111-1111-1111-1111-111111111111","sustain":{"kind":"for","seconds":0}}]`
	if _, err := DecodeAndValidateChainV2(zero); err == nil {
		t.Fatalf("sustain for 0 seconds must be rejected")
	}
}
```

- [ ] **Step 2: Run and confirm failure** — `undefined: field Sustain`.

- [ ] **Step 3: Extend the schema**

Add `"sustain"` to the properties of **every non-move branch** in `beat_chain.v2.schema.json` (Communicated, ObjectRelocated, OwnershipAccessChanged, EntityCreated, EntityDestroyed, AttributeChanged) — never to `ActorMoved`, whose `additionalProperties:false` then rejects it for free:

```json
"sustain": {"type":"object","oneOf":[
  {"properties":{"kind":{"const":"for"},"seconds":{"type":"integer","minimum":1}},"required":["kind","seconds"],"additionalProperties":false},
  {"properties":{"kind":{"const":"until_at"},"entity_id":{"type":"string","format":"uuid"},"place_id":{"type":"string","format":"uuid"}},"required":["kind","entity_id","place_id"],"additionalProperties":false},
  {"properties":{"kind":{"const":"until_attr"},"entity_id":{"type":"string","format":"uuid"},"attr":{"type":"string","minLength":1},"value":{"type":"string"}},"required":["kind","entity_id","attr","value"],"additionalProperties":false}
]}
```

- [ ] **Step 4: Add the Go type, field, and belt**

Add `Sustain` (above) to `beatseats.go`, the `Sustain *Sustain` field on `Attempt`, and to `validateAttemptFields` a cross-type check mirroring the schema — the defense-in-depth belt the file already applies to every other field: reject `Sustain` on `ActorMoved`; reject an unknown `kind`; require `Seconds > 0` for `for`; require both ids for `until_at`; require `entity_id`, `attr`, and a non-empty `value` for `until_attr`.

- [ ] **Step 5: Teach the decomposer**

Add to `core/api/prompts/decompose.txt`, alongside the existing `duration_class` rule, a rule and two worked examples: an act the player says they sustain **for a stated span** carries `sustain {"kind":"for","seconds":N}` with N converted from the player's own words; an act sustained **until something is true** carries `until_at` or `until_attr`, binding ids only from the candidates block. State plainly that `sustain` is never guessed — no sustain unless the player said one — and that a move never carries it.

- [ ] **Step 6: Verify and commit**

```bash
make reset && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../..
git add core/api/schema/beat_chain.v2.schema.json core/api/beatseats.go core/api/prompts/decompose.txt core/api/*_test.go
git commit -m "feat(journey): the until/for parse-shape (sustain) on non-move attempts"
```

---

### Task 4: The journey unit — spans, legs, thresholds

**Files:**
- Create: `core/api/journey.go`, `core/api/journey_test.go`

**Interfaces:**
- Produces:

```go
type Journey struct {
	ID, WorldID, ActorID, Kind string
	Threshold                  json.RawMessage
	SpanSeconds                int64
	LegsTotal, LegsDone        int
	StartedTick, CurrentTick   int64
	FrameID, StageID           string // "" when absent
	OriginCoord, GoalCoord     json.RawMessage
	GoalTarget                 string
	Status                     string
}

func (o *Orchestrator) activeJourney(ctx context.Context, worldID, actorID string) (*Journey, error)
func (o *Orchestrator) startJourney(ctx context.Context, worldID, actorID string, a Attempt, now int64) (*Journey, error)
func legSliceSeconds(j *Journey) int64
func (o *Orchestrator) thresholdMet(ctx context.Context, j *Journey) (bool, error)
func (o *Orchestrator) endJourney(ctx context.Context, j *Journey, status string) error
```

This task builds and tests them **without** the world's turn or stage resolution — those are Tasks 5 and 6. Keep it that way; a unit that can be tested alone is the point.

- [ ] **Step 1: Write the failing tests**

```go
// A slice is what is LEFT divided by the legs remaining, so rounding never strands progress: the last
// leg always closes the span exactly.
func TestLegSlice_LastLegClosesTheSpanExactly(t *testing.T) {
	j := &Journey{SpanSeconds: 1000, LegsTotal: 3, LegsDone: 0, StartedTick: 0, CurrentTick: 0}
	total := int64(0)
	for i := 0; i < 3; i++ {
		s := legSliceSeconds(j)
		if s <= 0 {
			t.Fatalf("leg %d produced a non-positive slice %d", i, s)
		}
		total += s
		j.LegsDone++
		j.CurrentTick += s
	}
	if total != 1000 {
		t.Fatalf("three legs covered %d seconds, want exactly the 1000-second span", total)
	}
}

// Travel's span is the real physics duration, and starting a journey must NOT move the actor.
func TestStartJourney_TravelSpanIsTheMoveDurationAndNothingCommits(t *testing.T) { /* … */ }

// A wait's span is the stated seconds, passed through untouched (R13).
func TestStartJourney_WaitSpanIsTheStatedSeconds(t *testing.T) { /* … */ }

// One active journey per actor: the database refuses a second.
func TestStartJourney_SecondActiveJourneyIsRefused(t *testing.T) { /* … */ }

// A watch resolves on a FACT, checked in SQL with no model involved.
func TestThresholdMet_UntilAtFlipsWhenTheEntityArrives(t *testing.T) { /* … */ }
```

Fill each body following `journey_test.go`'s sibling patterns (`testPool`, `wtOrchestrator`, `wtBaseTick`, `dlWorldID`, `wtMaraID`, `wtTavernID`, and Dock Street `210c0000-0000-0000-0000-0000000000d2`). Every test that writes a `journey` row must register a `t.Cleanup` deleting it — the unique partial index means a leaked active row breaks every later test for that actor.

- [ ] **Step 2: Run and confirm failure** (no such file/functions).

- [ ] **Step 3: Implement `journey.go`**

Key semantics, each of which a test above pins:

- `activeJourney` reads fresh, returns `nil, nil` when there is none — never an error for absence.
- `startJourney` derives the kind and span from the attempt: an over-budget `ActorMoved` → `travel`, span = `fnMoveDurationActor`, `goal_target` = `ToTargetID`, `frame_id` = the actor's current location's `parent_location_id`, `origin_coord`/`goal_coord` from `fn_target_position`; a `sustain.for` → `wait`, span = `Seconds`, threshold `{"kind":"tick","at":<now+seconds>}`; a `sustain.until_*` → `watch`, span = the per-world horizon default, threshold = the predicate. `legs_total` comes from `fn_journey_legs`. It writes the row and **commits nothing to canon** — starting a journey is not an event.
- `legSliceSeconds` = `ceil(remaining / legsRemaining)` where `remaining = SpanSeconds - (CurrentTick - StartedTick)`; when `legsRemaining <= 1` it returns the whole remainder, so the final leg lands exactly on the span.
- `thresholdMet` switches on kind: `travel` → `CurrentTick - StartedTick >= SpanSeconds`; `wait` → `CurrentTick >= at`; `watch` → the predicate, evaluated in ONE SQL query (`until_at`: the entity's `attrs.location_id` equals the place; `until_attr`: `attrs->>attr = value`). No model, ever.
- `endJourney` sets the status and nothing else — the row stays so `fn_world_now` keeps its tick (Task 1's assertion (d)).

- [ ] **Step 4: Verify and commit**

```bash
make reset && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... -run TestLegSlice -run TestStartJourney -run TestThresholdMet -v
git add core/api/journey.go core/api/journey_test.go
git commit -m "feat(journey): the journey unit — spans, legs, thresholds"
```

---

### Task 5: A leg runs, and the world gets its turn

**Files:**
- Create: nothing
- Modify: `core/api/journey.go` (add `runJourneyLeg`), `core/api/journey_test.go`

**Interfaces:**
- Produces: `func (o *Orchestrator) runJourneyLeg(ctx context.Context, j *Journey, outcome *BeatOutcome, trace *BeatTrace) error`

A leg, in order: compute the slice; advance the journey's clock; run the world's turn for `(tickBefore, tickAfter]` in the actor's current scene; persist `current_tick` and `legs_done`; then decide. Decisions, in priority order:

1. **A medium or large eruption fired** → `endJourney(j, "ended")`, `outcome.HaltReason = "journey_interrupted"` (R5: it ends; the player restates).
2. **The threshold is met** → for travel, commit the arrival `ActorMoved` to `goal_target` through the normal path; `endJourney(j, "arrived")`; `outcome.HaltReason = "journey_arrived"`.
3. **The last leg is spent and the threshold is not met** — only reachable for a watch, whose horizon ran out → `endJourney(j, "ended")`, `outcome.HaltReason = "journey_unresolved"`.
4. **Otherwise** → `outcome.HaltReason = "journey_leg"`; the player may continue.

- [ ] **Step 1: Write the failing tests**

```go
// The world takes a turn on EVERY leg — this is the ruling's "multiple chances to stop you", and it is
// the reason the journey exists rather than a fast-forward.
func TestRunJourneyLeg_FiresTheWorldsTurnEveryLeg(t *testing.T) { /* force a due pending event inside the leg's window; assert it fired */ }

// A hard cut-in ends the journey outright (R5) — nothing is suspended, nothing auto-resumes.
func TestRunJourneyLeg_MediumEruptionEndsTheJourney(t *testing.T) { /* wtForceTierFires medium; assert status='ended' and halt journey_interrupted */ }

// Walking the whole span arrives, and arrival is a real committed move — the actor is actually there.
func TestRunJourneyLeg_LastLegArrivesAndCommitsTheMove(t *testing.T) { /* run legs_total legs with the world quiet; assert halt journey_arrived, status='arrived', and actor_state.location_id is the goal */ }

// A watch whose horizon expires ends unresolved — nothing waits forever.
func TestRunJourneyLeg_WatchHorizonExpiresUnresolved(t *testing.T) { /* … */ }
```

Use `wtDisableWorldActor` in the arrival test so no stray eruption cuts the trip short, and `wtForceTierFires` in the interruption test. Both helpers are established (`worldturn_test.go:66,115`).

- [ ] **Step 2: Run and confirm failure.**

- [ ] **Step 3: Implement `runJourneyLeg`.** Call `runWorldTurn` **unchanged**, passing the scene from `actorLocation`. Thread `seq` from 0 for the leg. The arrival commit goes through `commitWorldPayload` with a `nil` hook — it is an ordinary commit, and reusing that path means the perception fan-out happens for free.

- [ ] **Step 4: Verify and commit**

```bash
make reset && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../..
git add core/api/journey.go core/api/journey_test.go
git commit -m "feat(journey): a leg runs, the world takes its turn, arrival commits"
```

---

### Task 6: Over-budget becomes a journey (the dead end dies)

**Files:**
- Modify: `core/api/orchestrator.go:244-261` (the move gate) and `:324-332` (the non-move gate); `core/api/orchestrator_worldtime_test.go` or a new `journey_beat_test.go`

**This is the two-line change the whole rung exists for.** At each gate, an over-budget attempt no longer halts — it starts a journey and runs its first leg. `turn_budget` survives **only** as the impossible-move guard: when `dur` is the speed-0 sentinel (`math.MaxInt64`) or would overflow, that is not "too long for this beat", it is "cannot be done at all", and it must keep halting.

- [ ] **Step 1: Write the failing test**

```go
// The founding ruling: "You can't even try to leave" is dramatically dead. An over-budget move must
// BEGIN rather than bounce.
func TestRunBeat_OverBudgetMoveBecomesAJourney(t *testing.T) {
	// A tense scene (30 s budget) and a target far enough that the walk cannot fit.
	// Assert: HaltReason is "journey_leg", NOT "turn_budget"; an active journey row exists for the actor;
	// the actor has NOT teleported to the goal.
}

// The impossible move is not a journey: an encumbered actor with speed 0 still halts turn_budget.
func TestRunBeat_ImpossibleMoveStillHaltsTurnBudget(t *testing.T) { /* … */ }
```

- [ ] **Step 2: Run and confirm it fails** (`HaltReason = "turn_budget"`).

- [ ] **Step 3: Change the two gates**

```go
			if over {
				// RULINGS-2026-07-30 §2: over-budget is NOT a reject — it is the Journey. The attempt
				// does not fit this beat, so it becomes a span the world gets to interrupt. The impossible
				// move (speed 0 → MaxInt64, or an overflow) is NOT over-budget in that sense: it cannot be
				// done at all, and still halts.
				if dur == math.MaxInt64 || dur > math.MaxInt64-curTick {
					outcome.HaltReason = "turn_budget"
					outcome.TicksAdvanced = curTick - startTick
					return nil
				}
				j, jErr := o.startJourney(ctx, worldID, actorID, attempt, curTick)
				if jErr != nil {
					return fmt.Errorf("start journey: %w", jErr)
				}
				if legErr := o.runJourneyLeg(ctx, j, outcome, trace); legErr != nil {
					return fmt.Errorf("journey leg: %w", legErr)
				}
				outcome.TicksAdvanced = j.CurrentTick - startTick
				return nil
			}
```

Apply the same shape at the non-move gate, where the trigger is a `sustain` attempt or an over-budget duration. Nothing else in `runChain` changes.

- [ ] **Step 4: Verify, then hunt the fallout**

Run the full suite. Existing tests that assert `turn_budget` on an over-budget *move* now legitimately get `journey_leg` — that is the feature, and those assertions SHOULD change. This is the one place in the program where editing an existing assertion is correct; for each one, state in your report which test, what it asserted, and why the new value is right. Tests asserting `turn_budget` for the speed-0 case must be untouched.

- [ ] **Step 5: Commit**

```bash
git add core/api/orchestrator.go core/api/*_test.go
git commit -m "feat(journey): an over-budget attempt becomes a journey, not a dead end"
```

---

### Task 7: Continue, and changing your mind

**Files:**
- Modify: `core/api/orchestrator.go:83` (`RunBeat`), `core/api/journey_test.go`

**The rule (R6):** continue advances one leg; any other input **ends** the journey and runs as a normal turn where you stand. In this rung, "continue" is an **empty chain** — the player said nothing new. Rung 3 maps `POST /beats/continue` onto exactly that, so nothing here needs a contract change.

- [ ] **Step 1: Write the failing tests**

```go
// Continue = an empty chain while a journey is active: it advances one leg and commits no new action.
func TestRunBeat_EmptyChainAdvancesTheActiveJourney(t *testing.T) { /* assert legs_done incremented, halt journey_leg */ }

// Any real input ends the journey and then runs normally, where the actor actually stands (R6).
func TestRunBeat_NewActionEndsTheJourneyAndRunsWhereYouStand(t *testing.T) { /* assert status='ended', the action resolved, no journey halt reason */ }
```

- [ ] **Step 2: Run and confirm failure** — the empty chain currently falls to the instant floor.

- [ ] **Step 3: Implement in `RunBeat`**, before the chain runs: read `activeJourney`; if one exists and the chain is empty, run a leg and return; if one exists and the chain is not empty, `endJourney(j, "ended")` and fall through to the normal path. Mirror the shape of the existing held-outcome check in `beathandler.go:172` — read the world's state fresh, then route.

- [ ] **Step 4: Verify and commit**

```bash
make reset && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../..
git add core/api/orchestrator.go core/api/journey_test.go
git commit -m "feat(journey): continue advances a leg; any other input ends the journey"
```

---

### Task 8: The world builds the road it needs

**Files:**
- Create: `core/api/placeauthor.go`, `core/api/prompts/place_author.txt`, `core/api/schema/place_author.v1.schema.json`, `core/api/placeauthor_test.go`
- Modify: `core/api/bridge.go` (the seat), `core/api/bridge_fakes.go` (a forceable fake), `core/api/main.go:49` and `core/api/beathandler.go:152-158` (registration), `core/api/journey.go` (call it from the leg)

**The ruling (R2/R4), verbatim in the design:** nothing is built while you walk. When the world's turn is about to act mid-journey and no known place contains the point you have reached, the world **creates** one — derived from the parent region and the known places nearby — together with its connections either side. What gets created lasts.

**Order inside a leg** (the founder's own sequence): does something happen? → are you somewhere known? → if not, create it → then author the event *for that place*.

- [ ] **Step 1: Write the failing tests**

```go
// Nothing happens on a quiet leg: the world does not build scenery nobody is looking at.
func TestJourneyLeg_QuietLegCreatesNothing(t *testing.T) { /* assert no new location entities */ }

// The world needs a stage and none contains the point → it creates one, WITH its connections, and acts there.
func TestJourneyLeg_EruptionOnOpenRoadCreatesThePlaceAndItsWays(t *testing.T) { /* … */ }

// A known place containing the point is USED, never duplicated — this is what stops the map filling with hamlets.
func TestJourneyLeg_KnownPlaceContainingThePointIsReused(t *testing.T) { /* give a place an area covering the point; assert no creation and stage_id = that place */ }

// Creation fills gaps only (R4): an existing shut or locked way is obeyed, never replaced.
func TestJourneyLeg_LockedWayIsNotRoutedAround(t *testing.T) { /* … */ }

// The seat never emits geometry: the engine draws the outline from the authored size class.
func TestPlaceAuthor_SeatSuppliesClassEngineDrawsOutline(t *testing.T) { /* … */ }
```

- [ ] **Step 2: Run and confirm failure.**

- [ ] **Step 3: Build the seat**

- `SeatPlaceAuthor = Seat{Name: "place_author", Requires: []Capability{CapStructuredOutput}}` in `bridge.go`, registered everywhere `SeatWorldActor` is.
- `place_author.v1.schema.json`: `{descriptor (required, non-empty), kind, extent_class (enum intimate|small|medium|large|vast)}`. **No coordinates, no radius, no numbers** — the schema is the leash that keeps geometry out of the model's hands.
- `place_author.txt`: the seat is told where it is (the parent region, the nearby known places, how far along the road) and asked what is *there*. Genre-agnostic (GA-2). It authors identity, not geography.
- `placeauthor.go`: assemble the payload deterministically; call the seat; validate; then the **engine** computes the outline via `fn_extent_class_metres` + `fn_area_around` and commits the place through `EntityCreated` on the normal ruled path (`fn_apply_entity_created` already enforces descriptor-mandatory and reuse-before-create), plus the connecting Portal artifacts — **only where no connection already exists**.

- [ ] **Step 4: Wire it into the leg**, at exactly the point the ruling describes: after the world's turn has decided it will act, before the World Actor authors anything. A quiet leg must never reach this code.

- [ ] **Step 5: Verify and commit**

```bash
make reset && make test && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../..
git add core/api/placeauthor.go core/api/prompts/place_author.txt core/api/schema/place_author.v1.schema.json core/api/placeauthor_test.go core/api/bridge.go core/api/bridge_fakes.go core/api/main.go core/api/beathandler.go core/api/journey.go
git commit -m "feat(journey): the world creates the ground it needs, with its ways"
```

---

### Task 9: Rung gate

**Files:** none — verification only.

- [ ] **Step 1: Full battery**

```bash
make reset && make test && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... && go vet ./... && cd ../.. && make schema-check
```

- [ ] **Step 2: Twice with no reset** — proves no leaked journey rows (the unique active index makes a leak loud).

- [ ] **Step 3: The founding ruling, end to end.** Write it as a test if one does not already exist: a tense scene, a walk that cannot fit the beat, the world quiet — the actor leaves, advances across legs, and arrives. Then the same trip with a medium eruption forced on leg two — the journey ends, the actor is standing where it happened, and saying "carry on" starts a fresh journey that arrives. That second half IS the founder's worked example, and it is the real gate.

- [ ] **Step 4: Confirm the dead end is gone**

```bash
grep -rn '"turn_budget"' core/api/ | grep -v _test.go
```
Expected: only the impossible-move guard in `runChain`.

- [ ] **Step 5: Ledger and PR.** Append `# RUNG 2 COMPLETE` to `.git/sdd/progress.md`. Open the PR based on `rung1/ground` (or `feat/living-world` if the earlier rungs have merged), quoting RULINGS-2026-07-30 §2 and citing D-1, R2–R7, and design §4.1–4.7.
