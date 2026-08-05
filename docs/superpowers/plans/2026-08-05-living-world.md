# The Living World (Station G / World Actor) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the world advance on its own clock and act on its own — every beat costs world-time, scheduled events fire when the clock crosses them, and a World Actor erupts on rising per-tier pressure — all through the existing pipeline, exposed as one reusable "world's-turn" unit the Journey will later call per leg.

**Architecture:** Five small units with clean interfaces (clock, scheduled-events ledger, pressure+roll, World Actor seat, world's-turn composer) plus two touch-points on existing code (the decomposer gains a `duration_class`; the trace gains a world's-turn block). The engine owns WHEN (mechanical world-time, deterministic rolls); the LLM owns WHAT (the authored intrusion). Pressure is *derived* (time since a tier last erupted), never a per-beat counter. Builds on Grounded Reasoning (fact sheet + trace) and Station F (contract functions).

**Tech Stack:** Go (`core/api`, package `main`), plpgsql + dbmate migrations, pgTAP tests (`core/db/tests/NN_*_test.sql`), go:embed for schemas/prompts. Test commands: `make reset` (clean+migrate+seed), `make test` (pgTAP via pg_prove), `go test ./core/api/...`, `make schema-check` (schema.sql drift).

## Global Constraints

Every task's requirements implicitly include these (copied from `docs/superpowers/specs/2026-08-05-living-world-design.md`):

- **D-1: SQL is the only canon writer.** Go orchestrates; all canon commits go through existing SQL apply paths.
- **Modular mandate (CORE):** five units + two touch-points; each new file one responsibility; **no cross-layer patches** — if a change doesn't fit cleanly, fix the boundary.
- **World-time, not player-steps.** Pressure and crossings ride `in_world_tick`; a chatty player never accelerates the world.
- **No tension-damping of eruptions.** Disruption at the worst moment is the feature; accrual is pure elapsed time.
- **§5 magnitude cut:** `small` commits and the chain runs on; `medium`/`large` commit, narrate, and **end the beat** (remaining attempts discarded). Eruptions are **not contestable** — no held outcome, no reaction beat.
- **Author truth + engine fans out.** The World Actor authors a truth event carrying a **location**; it **never** encodes who perceives it. Perception is the shared fan-out. (This is the invariant that lets off-scene "B" grow in later without a seat/schema change.)
- **Same pipeline, no bypass.** World output routes through the existing adjudicate/commit + perception fan-out.
- **Config is per-world data, never hardcoded.** Class→seconds, per-tier climb rate/chunk/cap, master intensity/off — all seeded rows, retunable.
- **Deterministic roll (replay-safe).** No nondeterministic RNG; the draw is a pure function of committed state so replay reproduces it.
- **Trace is debug-only, absent-not-null, pure capture** (no new LLM call), same discipline as `trace.go`.
- **No Journey logic in this station** — no progress accumulation, thresholds, or "until" resolution. Only expose the clean world's-turn unit + clock.
- **Cap default ~0.70, configurable; climb keyed to world-time chunks.**
- **Branch:** `feat/living-world`, off the `feat/grounded-reasoning` tip (`ce9568e`).

**Pre-existing sockets (do NOT recreate):** `pending_event(pending_id, world_id, fire_at_tick, magnitude, payload jsonb, status)` table; `fn_due_pending(p_world_id, p_tick) RETURNS SETOF pending_event`; `world_pressure(world_id, tier, accrued, last_fired_tick)` table (its `accrued` column becomes vestigial — leave it, don't depend on it); `SeatWorldActor = Seat{Name:"world_actor", Requires:[CapStructuredOutput]}` (bridge.go:82); `fakeWorldActorDriver` (bridge_fakes.go:128); `Orchestrator.WorldActor Driver` (orchestrator.go:33), wired in main.go/beathandler.go.

**Confirmed interfaces from the real code (consume these by exact name):**
- `func (o *Orchestrator) RunBeat(ctx, worldID, actorID string, chain []Attempt, startTick int64, trace *BeatTrace) (BeatOutcome, error)` — orchestrator.go:83
- `func (o *Orchestrator) runChain(ctx, worldID, actorID string, chain []Attempt, startTick int64, startSeq int, budgetRemaining int64, outcome *BeatOutcome, trace *BeatTrace) error` — orchestrator.go:116 (the per-attempt loop; clock advances at the `// Stage 4` block ~orchestrator.go:244-260)
- `func (o *Orchestrator) applyEvent(ctx, worldID, actorID string, attemptJSON []byte, tick int64, seq int) (map[string]any, error)` — orchestrator.go:936 (passthrough commit; `result["halt_reason"]`, `result["event_id"]`)
- `func (o *Orchestrator) adjudicate(ctx, worldID string, set []ActorAttempt, resolveHeldIDs []string, tick int64, curSeq int, playerAnswer string, trace *BeatTrace) (adjResult, error)` — orchestrator.go:1012 (`adjResult{Committed []string, Halt string, SeqAdvance int}`)
- `func (o *Orchestrator) factSheetJSON(ctx, worldID, viewer string, involved []string, truthSide bool) (string, error)` — orchestrator.go:958 (pattern for a Go→SQL jsonb helper)
- `func (o *Orchestrator) beatBudgetSeconds(ctx, worldID, actorID string) (int64, error)` — tension.go:51
- `type Attempt struct {...}` — beatseats.go:56; `func validateAttemptFields(i int, a Attempt) error` — beatseats.go:100; `var beatChainV2SchemaJSON string` (go:embed `schema/beat_chain.v2.schema.json`) — beatvocab.go:8; `var allowedBeatTypesV2` — beatvocab.go:61
- `type BeatTrace struct {...}` with nil-safe append methods — trace.go
- Config seed point: `seed_world_defaults(p_world_id uuid)` — schema.sql (currently seeds movement_type + status_modifier)
- Migrations: `core/db/migrations/YYYYMMDDHHMMSS_name.sql` (dbmate up/down); latest is `20260730100001_fact_sheet.sql`. pgTAP tests: `core/db/tests/NN_*_test.sql`.

---

### Task 1: `duration_class` on the beat chain (decode + schema + prompt)

**Files:**
- Modify: `core/api/beatseats.go:56` (Attempt struct), `core/api/beatseats.go:100` (validateAttemptFields)
- Modify: `core/api/schema/beat_chain.v2.schema.json` (add `duration_class` to the non-move attempt shapes)
- Modify: `core/api/prompts/decompose.txt` (the classification rule + examples)
- Test: `core/api/beatseats_test.go` (or the existing decode test file — grep `DecodeAndValidateChainV2` in `*_test.go`)

**Interfaces:**
- Produces: `Attempt.DurationClass string` (json `duration_class,omitempty`); valid values `instant|short|medium|long|extremely_long`; empty allowed on move attempts and legacy input.

- [ ] **Step 1: Write the failing test** — a decoded non-move attempt carries its class, and an out-of-enum class is rejected:

```go
func TestDecodeChainV2_DurationClass(t *testing.T) {
	ok := `[{"type":"Communicated","stated":"I tell Mara my whole life story","listener_id":"11111111-1111-1111-1111-111111111111","content":"...","duration_class":"long"}]`
	chain, err := DecodeAndValidateChainV2(ok)
	if err != nil { t.Fatalf("valid class rejected: %v", err) }
	if chain[0].DurationClass != "long" { t.Fatalf("class not decoded: %q", chain[0].DurationClass) }

	bad := `[{"type":"Communicated","stated":"x","listener_id":"11111111-1111-1111-1111-111111111111","content":"x","duration_class":"aeon"}]`
	if _, err := DecodeAndValidateChainV2(bad); err == nil {
		t.Fatalf("out-of-enum duration_class accepted")
	}
}
```

- [ ] **Step 2: Run it, verify it fails** — `go test ./core/api/ -run TestDecodeChainV2_DurationClass -v` → FAIL (field missing / not validated).

- [ ] **Step 3: Add the field** to `Attempt` (beatseats.go, after `QueryTargetIDs`):

```go
	// DurationClass is the decomposer's parse-shape estimate of how long a NON-MOVE act takes in the
	// fiction — one of instant|short|medium|long|extremely_long (a validated enum, like a QUERY shape,
	// NOT a raw number and NOT the banned outcome/tension/intent judgment; RULINGS-2026-07-23 §4). The
	// engine maps class→seconds per world (fn_duration_class_seconds). Empty on ActorMoved (physics owns
	// move duration) and on legacy input.
	DurationClass string `json:"duration_class,omitempty"`
```

- [ ] **Step 4: Validate the enum** — in `validateAttemptFields`, add after the type switch (a cross-type check, since any non-move type may carry it):

```go
	if a.DurationClass != "" {
		switch a.DurationClass {
		case "instant", "short", "medium", "long", "extremely_long":
		default:
			return fmt.Errorf("step %d duration_class %q outside enum", i, a.DurationClass)
		}
	}
```

- [ ] **Step 5: Add `duration_class` to the schema JSON** — in `schema/beat_chain.v2.schema.json`, add to each NON-move `oneOf` alternative's `properties` an optional enum (do NOT add to ActorMoved):

```json
"duration_class": { "type": "string", "enum": ["instant","short","medium","long","extremely_long"] }
```

- [ ] **Step 6: Teach the decomposer** — append to `core/api/prompts/decompose.txt` a rule: every NON-move attempt gets a `duration_class` by the *shape* of the act, never a number — `instant` (a nod, a glance), `short` (a sentence, hand over an object), `medium` (explain something), `long` (tell a life story, search a room), `extremely_long` (stand watch for hours). Moves omit it. Add one worked example: `"I tell Mara my whole life story"` → `duration_class: "long"`.

- [ ] **Step 7: Run tests + vocab guard** — `go test ./core/api/ -run 'TestDecodeChainV2|Vocabulary' -v` → PASS.

- [ ] **Step 8: Commit** — `git commit -am "feat(livingworld): decomposer tags non-move acts with a duration_class (parse-shape enum) [U1]"`

---

### Task 2: class→seconds config + `fn_duration_class_seconds`

**Files:**
- Create: `core/db/migrations/20260805100001_duration_class_config.sql`
- Modify: `core/db/schema.sql` (regenerated by `make migrate` — do not hand-edit)
- Test: `core/db/tests/102_duration_class_test.sql`

**Interfaces:**
- Produces: table `duration_class_seconds(world_id, class, seconds)`; `fn_duration_class_seconds(p_world_id uuid, p_class text) RETURNS bigint` (falls back to a built-in default when unseeded); `seed_world_defaults` seeds the five classes + a beat floor.

- [ ] **Step 1: Write the failing pgTAP test** `core/db/tests/102_duration_class_test.sql` — three `DO $$` blocks in the house style: (a) `seed_world_defaults(w)` populates all five classes; (b) `fn_duration_class_seconds(w,'long')` returns the seeded seconds and is strictly greater than `'short'`; (c) `fn_duration_class_seconds(w,'instant')` is > 0 (the floor is non-zero).

```sql
-- (b)
PREPARE want AS SELECT fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','long')
                     > fn_duration_class_seconds('22222222-2222-2222-2222-222222222222','short');
```

- [ ] **Step 2: Run it, verify it fails** — `make reset && make test 2>&1 | grep 102_duration` → FAIL (function/table absent).

- [ ] **Step 3: Write the migration** `20260805100001_duration_class_config.sql`:

```sql
-- migrate:up
CREATE TABLE duration_class_seconds (
  world_id uuid NOT NULL,
  class    text NOT NULL CHECK (class IN ('instant','short','medium','long','extremely_long')),
  seconds  bigint NOT NULL CHECK (seconds > 0),
  PRIMARY KEY (world_id, class)
);

CREATE FUNCTION fn_duration_class_seconds(p_world_id uuid, p_class text) RETURNS bigint
  LANGUAGE sql STABLE AS $$
  SELECT COALESCE(
    (SELECT seconds FROM duration_class_seconds WHERE world_id = p_world_id AND class = p_class),
    CASE p_class  -- built-in fallback (retune per-world via the table)
      WHEN 'instant' THEN 2 WHEN 'short' THEN 5 WHEN 'medium' THEN 60
      WHEN 'long' THEN 300 WHEN 'extremely_long' THEN 7200 ELSE 2 END
  );
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_duration_class_seconds(uuid, text);
DROP TABLE IF EXISTS duration_class_seconds;
```

- [ ] **Step 4: Extend `seed_world_defaults`** — this function lives in an existing migration; add its new INSERT via a *new* statement in THIS migration's `up` (idempotent), matching the seed pattern:

```sql
INSERT INTO duration_class_seconds (world_id, class, seconds)
SELECT w.world_id, v.class, v.seconds
FROM (SELECT DISTINCT world_id FROM world) w,
     (VALUES ('instant',2),('short',5),('medium',60),('long',300),('extremely_long',7200)) AS v(class,seconds)
ON CONFLICT DO NOTHING;
```
Also add the same five-row INSERT (parameterized on `p_world_id`) into `seed_world_defaults` by `CREATE OR REPLACE FUNCTION seed_world_defaults` in this migration (copy its current body from schema.sql, append the INSERT).

- [ ] **Step 5: Migrate + run test** — `make reset && make test 2>&1 | grep 102_duration` → PASS. Then `make schema-check` → clean.

- [ ] **Step 6: Commit** — `git add -A && git commit -m "feat(livingworld): per-world duration_class→seconds config + fn_duration_class_seconds [U1]"`

---

### Task 3: Every beat costs world-time (non-move clock consumption + floor)

**Files:**
- Modify: `core/api/orchestrator.go` (the `runChain` clock-advance block, ~orchestrator.go:244-260, the `else` non-move branch; and the loop tail where `outcome.TicksAdvanced` is set)
- Test: `core/api/orchestrator_worldtime_test.go` (new)

**Interfaces:**
- Consumes: `fn_duration_class_seconds` (via a small helper `func (o *Orchestrator) durationClassSeconds(ctx, worldID, class string) (int64, error)` mirroring `factSheetJSON`'s Go→SQL pattern).
- Produces: non-move attempts advance `curTick` by their class duration (floored); a beat with no clock-advancing attempt still advances by the floor.

- [ ] **Step 1: Write the failing test** — through `RunBeat`, a single non-move `Communicated{duration_class:"long"}` beat advances `TicksAdvanced` by the seeded `long` seconds (300), not 0:

```go
func TestRunBeat_NonMoveCostsWorldTime(t *testing.T) {
	// seed world 22222222; Kade 2ac70000-...-a1; fakes for all seats.
	chain := []Attempt{{Type: "Communicated", Stated: "life story", ListenerID: maraID, Content: "...", DurationClass: "long"}}
	out, err := orc.RunBeat(ctx, playWorld, kadeID, chain, 0, nil)
	if err != nil { t.Fatal(err) }
	if out.TicksAdvanced != 300 { t.Fatalf("non-move beat advanced %d, want 300", out.TicksAdvanced) }
}
```

- [ ] **Step 2: Run it, verify it fails** — `go test ./core/api/ -run TestRunBeat_NonMoveCostsWorldTime -v` → FAIL (advances 0).

- [ ] **Step 3: Add the helper** (new small method near `factSheetJSON`):

```go
func (o *Orchestrator) durationClassSeconds(ctx context.Context, worldID, class string) (int64, error) {
	var s int64
	err := o.DB.QueryRow(ctx, `SELECT fn_duration_class_seconds($1,$2)`, worldID, class).Scan(&s)
	return s, err
}
```

- [ ] **Step 4: Consume it in the non-move branch** — in `runChain`'s Stage-4 clock advance, replace the non-move `else` (`curSeq++` only) so a non-move with a class advances the clock and consumes budget cumulatively (same shape as the move branch, honoring §6): compute `dur` from `durationClassSeconds` (0 → treat as the class fallback; empty class → the `instant` floor value so stillness still ticks), apply the same `overBudget` check + `turn_budget` halt as moves (interim over-budget behavior until the Journey; do NOT build journey logic), then `budgetRemaining -= dur; curTick += dur` (and `curSeq = 0` when `dur > 0`, else `curSeq++`). Keep moves on `fn_move_duration_actor` unchanged.

- [ ] **Step 5: Floor for the empty/zero-advance beat** — after the loop, if `curTick == startTick` (no attempt advanced world-time — e.g. an empty "I watch" beat, or only QUERY), advance by the `instant` floor once: `floor, _ := o.durationClassSeconds(ctx, worldID, "instant"); curTick += floor` before setting `outcome.TicksAdvanced = curTick - startTick`. (A QUERY-only beat still ticks the floor; a pure-halt beat keeps its halt semantics — only apply the floor on `completed`.)

- [ ] **Step 6: Run tests** — `go test ./core/api/ -run 'TestRunBeat|runChain' -v` → PASS; run the full package `go test ./core/api/...` to catch budget-interaction regressions.

- [ ] **Step 7: Commit** — `git commit -am "feat(livingworld): every beat costs world-time — non-move acts consume their class duration, floor for stillness [U1/U2]"`

---

### Task 4: The scheduled-events ledger — fire due events on a clock-crossing

**Files:**
- Create: `core/api/ledger.go`
- Test: `core/api/ledger_test.go`

**Interfaces:**
- Consumes: `fn_due_pending`, `applyEvent`, `adjudicate`.
- Produces: `func (o *Orchestrator) fireDuePending(ctx, worldID string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, err error)` — fires every pending row with `fire_at_tick` in `(tickBefore, tickAfter]`, commits each payload through the pipeline, flips `status='fired'`, and returns the **largest magnitude** fired (`""|small|medium|large`).

- [ ] **Step 1: Write the failing test** — a `pending_event` at tick 10, magnitude `medium`, with a valid Communicated payload: a slot crossing `(9,12]` fires it (canon count +1, row `status='fired'`) and returns `"medium"`; a slot crossing `(0,9]` fires nothing.

```go
func TestFireDuePending_CrossingFires(t *testing.T) {
	insertPending(t, playWorld, 10, "medium", `{"type":"Communicated","stated":"the bell tolls","listener_id":"`+kadeID+`","content":"a bell tolls over the docks"}`)
	mag, err := orc.fireDuePending(ctx, playWorld, 9, 12, 0, &out, nil)
	if err != nil { t.Fatal(err) }
	if mag != "medium" { t.Fatalf("fired mag %q want medium", mag) }
	if canonCount(t, playWorld) != before+1 { t.Fatalf("payload not committed") }
	if pendingStatus(t, 10) != "fired" { t.Fatalf("row not flipped") }
}
```

- [ ] **Step 2: Run it, verify it fails** — `go test ./core/api/ -run TestFireDuePending -v` → FAIL.

- [ ] **Step 3: Implement `fireDuePending`** in `ledger.go`: `SELECT pending_id, magnitude, payload FROM pending_event WHERE world_id=$1 AND status='pending' AND fire_at_tick > $2 AND fire_at_tick <= $3 ORDER BY fire_at_tick`; for each row, unmarshal payload → `Attempt`, route it exactly as `runChain` routes an attempt (passthrough types → `applyEvent`; adjudicated types → `adjudicate` with the payload's actor as a single-element `set`), append committed ids to `outcome.Committed`, `UPDATE pending_event SET status='fired' WHERE pending_id=$1`, and track the max magnitude by rank `small<medium<large`. The payload carries its own actor id (the world entity acting); ledger events are **world truth with a location** — do not synthesize perceptions here, the commit path fans out. Return the max magnitude.

- [ ] **Step 4: Run test** — `go test ./core/api/ -run TestFireDuePending -v` → PASS.

- [ ] **Step 5: Commit** — `git commit -am "feat(livingworld): scheduled-events ledger fires due events on a clock-crossing [U3]"`

---

### Task 5: Pressure config + fire-log + `fn_pressure_chance`

**Files:**
- Create: `core/db/migrations/20260805100002_world_actor_pressure.sql`
- Test: `core/db/tests/103_world_pressure_test.sql`

**Interfaces:**
- Produces: `world_actor_config(world_id, tier, climb_rate, climb_chunk_ticks, cap)`; `world_actor_setting(world_id PK, enabled bool, intensity numeric)`; append-only `world_eruption(world_id, tier, fired_tick, event_id)` (the last-eruption source + audit log, written only on fire); `fn_pressure_chance(p_world_id uuid, p_tier text, p_now bigint) RETURNS numeric` = `LEAST(cap, climb_rate * ((p_now - last_eruption_tick) / climb_chunk_ticks)) * intensity`, `0` when disabled; `seed_world_defaults` seeds the three tiers + setting.

- [ ] **Step 1: Write the failing pgTAP test** `103_world_pressure_test.sql` (house style, world 22222222): (a) `seed_world_defaults` populates three tiers + one setting; (b) with no prior eruption, `fn_pressure_chance(w,'small',now)` **rises** as `now` grows and **never exceeds** the seeded cap; (c) a `world_eruption` row at tick T **drops** the chance at T back toward 0 (drain); (d) `world_actor_setting.enabled=false` → chance is exactly 0; (e) pools are independent — a `small` eruption does not change `fn_pressure_chance(...,'large',...)`.

- [ ] **Step 2: Run it, verify it fails** — `make reset && make test 2>&1 | grep 103_world_pressure` → FAIL.

- [ ] **Step 3: Write the migration** `20260805100002_world_actor_pressure.sql`:

```sql
-- migrate:up
CREATE TABLE world_actor_config (
  world_id uuid NOT NULL,
  tier text NOT NULL CHECK (tier IN ('small','medium','large')),
  climb_rate numeric NOT NULL CHECK (climb_rate >= 0),        -- chance added per climb_chunk of world-time
  climb_chunk_ticks bigint NOT NULL CHECK (climb_chunk_ticks > 0),  -- one "climb" = this many ticks
  cap numeric NOT NULL CHECK (cap >= 0 AND cap <= 1),
  PRIMARY KEY (world_id, tier)
);
CREATE TABLE world_actor_setting (
  world_id uuid PRIMARY KEY,
  enabled boolean NOT NULL DEFAULT true,
  intensity numeric NOT NULL DEFAULT 1.0 CHECK (intensity >= 0)
);
CREATE TABLE world_eruption (           -- append-only: the last-eruption source + fire audit log
  world_id uuid NOT NULL,
  tier text NOT NULL CHECK (tier IN ('small','medium','large')),
  fired_tick bigint NOT NULL,
  event_id uuid NOT NULL
);
CREATE INDEX idx_world_eruption_lookup ON world_eruption (world_id, tier, fired_tick DESC);

CREATE FUNCTION fn_pressure_chance(p_world_id uuid, p_tier text, p_now bigint) RETURNS numeric
  LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN COALESCE((SELECT enabled FROM world_actor_setting WHERE world_id=p_world_id), true) IS FALSE
              THEN 0
         ELSE LEAST(c.cap,
                    c.climb_rate * ((p_now - COALESCE(
                      (SELECT max(fired_tick) FROM world_eruption WHERE world_id=p_world_id AND tier=p_tier), 0
                    ))::numeric / c.climb_chunk_ticks))
              * COALESCE((SELECT intensity FROM world_actor_setting WHERE world_id=p_world_id), 1.0)
         END
  FROM world_actor_config c WHERE c.world_id=p_world_id AND c.tier=p_tier;
$$;

-- migrate:down
DROP FUNCTION IF EXISTS fn_pressure_chance(uuid, text, bigint);
DROP TABLE IF EXISTS world_eruption;
DROP TABLE IF EXISTS world_actor_setting;
DROP TABLE IF EXISTS world_actor_config;
```

- [ ] **Step 4: Seed defaults** — in this migration, `CREATE OR REPLACE FUNCTION seed_world_defaults` (copy current body from schema.sql + append), and also backfill existing worlds. Seed small (fast climb, small chunk), medium (slower), large (slowest), each cap `0.70`; one `world_actor_setting` row per world (enabled, intensity 1.0):

```sql
INSERT INTO world_actor_config (world_id, tier, climb_rate, climb_chunk_ticks, cap)
SELECT w.world_id, v.tier, v.rate, v.chunk, 0.70
FROM (SELECT DISTINCT world_id FROM world) w,
     (VALUES ('small',0.01,60),('medium',0.01,3600),('large',0.01,86400)) AS v(tier,rate,chunk)
ON CONFLICT DO NOTHING;
INSERT INTO world_actor_setting (world_id) SELECT DISTINCT world_id FROM world ON CONFLICT DO NOTHING;
```

- [ ] **Step 5: Migrate + run test** — `make reset && make test 2>&1 | grep 103_world_pressure` → PASS; `make schema-check` clean.

- [ ] **Step 6: Commit** — `git add -A && git commit -m "feat(livingworld): pressure config + append-only fire-log + fn_pressure_chance (derived, capped, per-tier) [U4]"`

---

### Task 6: The deterministic roll

**Files:**
- Create: `core/api/pressure.go`
- Test: `core/api/pressure_test.go`

**Interfaces:**
- Consumes: `fn_pressure_chance` (via a Go helper `pressureChance(ctx, worldID, tier string, now int64) (float64, error)`).
- Produces: `func deterministicUnit(worldID string, tick, lastEruption int64, tier string) float64` (in `[0,1)`, pure); `func (o *Orchestrator) rollTier(ctx, worldID, tier string, now, lastEruption int64) (fired bool, chance, roll float64, err error)`.

- [ ] **Step 1: Write the failing test** — the draw is a pure function (same inputs → identical output across calls) and lands in `[0,1)`; and `rollTier` fires iff `roll < chance`:

```go
func TestDeterministicUnit_Stable(t *testing.T) {
	a := deterministicUnit("w", 100, 0, "small")
	b := deterministicUnit("w", 100, 0, "small")
	if a != b { t.Fatalf("not deterministic: %v %v", a, b) }
	if a < 0 || a >= 1 { t.Fatalf("out of range: %v", a) }
	if deterministicUnit("w", 101, 0, "small") == a { t.Fatalf("tick should vary the draw") }
}
```

- [ ] **Step 2: Run it, verify it fails** — `go test ./core/api/ -run TestDeterministicUnit -v` → FAIL.

- [ ] **Step 3: Implement** `pressure.go` — `deterministicUnit` hashes `worldID|tick|lastEruption|tier` (fnv64a) → the top 53 bits mapped to `[0,1)`; `pressureChance` calls the SQL fn; `rollTier` reads the chance, computes the draw with the tier's last-eruption tick, returns `fired = roll < chance` plus both numbers (for the trace).

- [ ] **Step 4: Run test** — `go test ./core/api/ -run 'TestDeterministicUnit|TestRollTier' -v` → PASS.

- [ ] **Step 5: Commit** — `git commit -am "feat(livingworld): deterministic per-tier pressure roll (replay-safe, no storage) [U4]"`

---

### Task 7: The world-scope payload — `fn_world_slice`

**Files:**
- Create: `core/db/migrations/20260805100003_world_slice.sql`
- Test: `core/db/tests/104_world_slice_test.sql`

**Interfaces:**
- Produces: `fn_world_slice(p_world_id uuid, p_scene uuid) RETURNS jsonb` — a bounded WORLD-scope payload: `{ledger:[pending rows], presence:[{actor,location}], locations:[...], recent:[world-level canon tail], scene:{...}}`. Truth-side (the World Actor is world-omniscient by role); it never encodes perception.

- [ ] **Step 1: Write the failing pgTAP test** `104_world_slice_test.sql` — on the seeded play world: the result is `jsonb` with non-null `ledger`, `presence`, and `scene` keys; `presence` includes an NPC that is **not** in the current scene (proving world-scope, not scene-scope).

- [ ] **Step 2: Run it, verify it fails** — `make reset && make test 2>&1 | grep 104_world_slice` → FAIL.

- [ ] **Step 3: Write the migration** — `fn_world_slice` assembling the keys via `jsonb_build_object` + `jsonb_agg` over `pending_event` (status='pending'), actor presence (actor_state location_id path), `location`/region rows, and a bounded recent-canon tail (ORDER BY in_world_tick DESC LIMIT N). Mirror `fn_fact_sheet`'s construction style (schema.sql, migration `20260730100001_fact_sheet.sql`). Include the scene as a nested object so the seat *can* aim at it.

- [ ] **Step 4: Migrate + run test** — `make reset && make test 2>&1 | grep 104_world_slice` → PASS; `make schema-check` clean.

- [ ] **Step 5: Commit** — `git add -A && git commit -m "feat(livingworld): fn_world_slice — bounded world-scope payload for the World Actor [U5]"`

---

### Task 8: The World Actor seat — author an intrusion within the drawn size

**Files:**
- Create: `core/api/schema/world_actor.v1.schema.json`, `core/api/prompts/world_actor.txt`, `core/api/worldactorprompt.go`, `core/api/worldactor.go`
- Modify: `core/api/bridge_fakes.go` (make `fakeWorldActorDriver` return a size-appropriate deterministic stub instead of `[]`, so tests can force an eruption); `core/api/main.go` (real driver config already present)
- Test: `core/api/worldactor_test.go`

**Interfaces:**
- Consumes: `fn_world_slice`, `Orchestrator.WorldActor` driver, `applyEvent`/`adjudicate`, the perception fan-out (via the commit path).
- Produces: `func (o *Orchestrator) runWorldActor(ctx, worldID, scene, size string, now int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (eventID string, err error)` — builds the world slice, calls the seat with `size` as an input constraint + `world_actor.v1` schema, decodes the authored `Attempt` (validated against the closed vocabulary), commits it through the pipeline (truth event **with a location**, never a perceived edge), and returns the committed event id.

- [ ] **Step 1: Write the failing test** — force the fake to author a `Communicated` intrusion for `size="medium"`; assert `runWorldActor` commits exactly one canon event, the event carries a `location_id`, and the seat was handed the `size` constraint (assert the prompt contains `medium`).

```go
func TestRunWorldActor_AuthorsWithinSize(t *testing.T) {
	orc := &Orchestrator{DB: pool, WorldActor: NewFakeWorldActorDriver(), /*+ resolve/narrate fakes*/}
	id, err := orc.runWorldActor(ctx, playWorld, tavernID, "medium", 100, 0, &out, nil)
	if err != nil { t.Fatal(err) }
	if id == "" || canonCount(t, playWorld) != before+1 { t.Fatalf("no single commit") }
	if eventLocation(t, id) == "" { t.Fatalf("authored event has no location (B-growth invariant)") }
}
```

- [ ] **Step 2: Run it, verify it fails** — `go test ./core/api/ -run TestRunWorldActor -v` → FAIL.

- [ ] **Step 3: Author the schema + prompt** — `world_actor.v1.schema.json` = a single `Attempt` object (the six-type vocabulary, requires `type` + a `location`), go:embedded via a `var worldActorSchemaJSON string`. `world_actor.txt` states: you are the world beyond the scene; author ONE intrusion of the drawn SIZE; attribute it to a world entity; it carries a location; you author TRUTH, never who perceives it; you may bring a non-present NPC into the scene (`ActorMoved` of an off-scene actor toward/into `scene`); no appropriateness filter.

- [ ] **Step 4: Implement `worldactorprompt.go`** — `buildWorldActorPrompt(slice string, size string, scene sceneInfo) string`, mirroring `buildResolvePrompt`'s section layout (stable header → world slice → the SIZE constraint line → the authoring rules).

- [ ] **Step 5: Implement `worldactor.go`** — assemble slice via `fn_world_slice`, call `o.WorldActor.Generate` with the prompt + schema, decode one `Attempt`, run it through `validateAttemptFields`, route to `applyEvent`/`adjudicate` (same as the ledger), return the event id.

- [ ] **Step 6: Make the fake forceable** — `fakeWorldActorDriver.Generate` returns a deterministic size-tagged `Communicated` at the scene (so tests can force + assert), still erroring when `req.Schema == nil`.

- [ ] **Step 7: Run tests** — `go test ./core/api/ -run 'TestRunWorldActor|Wall' -v` → PASS (include the wall test: the authored event must not encode a perceived edge).

- [ ] **Step 8: Commit** — `git commit -am "feat(livingworld): the World Actor seat authors a truth intrusion within the drawn size; engine fans out (B-ready) [U5]"`

---

### Task 9: The world's-turn composer + wire it into the beat

**Files:**
- Create: `core/api/worldturn.go`
- Modify: `core/api/orchestrator.go` (call `runWorldTurn` in `runChain` after each attempt's clock advance; apply the §5 cut)
- Test: `core/api/worldturn_test.go`

**Interfaces:**
- Consumes: `fireDuePending` (Task 4), `rollTier` (Task 6), `runWorldActor` (Task 8), `world_eruption` insert.
- Produces: `func (o *Orchestrator) runWorldTurn(ctx, worldID, scene string, tickBefore, tickAfter int64, seq int, outcome *BeatOutcome, trace *BeatTrace) (firedMag string, err error)` — (1) fires due pending events; if a medium/large pending fired, return it (skip the roll); (2) else roll each tier `small→medium→large`; on the first/most-significant fire, call `runWorldActor(size)`, insert a `world_eruption` row `(world,tier,tickAfter,eventID)`, and return that magnitude; (3) return `""` if nothing fired.

- [ ] **Step 1: Write the failing test** — two assertions: (a) with the fake pressure forced (seed a config that guarantees `small` fires at the test tick, `medium/large` disabled), a normal beat commits an extra `small` event and the chain **continues**; (b) with `medium` forced, `runChain` halts after the current attempt (`outcome.HaltReason == "world_eruption"`) and a following attempt in the chain does **not** commit.

- [ ] **Step 2: Run it, verify it fails** — `go test ./core/api/ -run TestWorldTurn -v` → FAIL.

- [ ] **Step 3: Implement `runWorldTurn`** in `worldturn.go` per the interface above (rank magnitudes `small<medium<large`; ledger fires first, then the roll; a fire inserts the `world_eruption` row in the same logical step as the commit).

- [ ] **Step 4: Wire into `runChain`** — after the Stage-4 clock advance for a committed attempt, call `runWorldTurn(ctx, worldID, scene, tickBefore, curTick, curSeq, outcome, trace)` where `tickBefore` is the tick before this attempt advanced it; if it returns `medium`/`large`, set `outcome.HaltReason = "world_eruption"`, set `TicksAdvanced`, and `return nil` (discard the rest — the §5 cut, same shape as the telegraph halt); `small`/`""` → continue the loop. `scene` = the actor's current location id (read once per beat). Guard: do NOT run the world's turn after a QUERY/UNRESOLVED (they don't advance the clock).

- [ ] **Step 5: Run tests** — `go test ./core/api/...` → PASS (whole package, to catch beat-loop regressions).

- [ ] **Step 6: Commit** — `git commit -am "feat(livingworld): the world's-turn composer — the reusable per-slot unit; §5 cut wired into the beat [U6/U2]"`

---

### Task 10: The trace — a debug-only world's-turn block

**Files:**
- Modify: `core/api/trace.go` (add `WorldTurn []TraceWorldTurn` + a nil-safe append), `core/api/worldturn.go` (append during the turn), `core/api/beathandler.go` (already serializes `reasoning_log` when debug — no gating change)
- Test: `core/api/trace_test.go`

**Interfaces:**
- Produces: `TraceWorldTurn{ClockDeltaS int64; Fired []string; Rolls []TraceRoll; Eruption *TraceElement}` where `TraceRoll{Tier string; Chance float64; Roll float64; Fired bool}`; `func (t *BeatTrace) appendWorldTurn(w TraceWorldTurn)` (nil-safe).

- [ ] **Step 1: Write the failing test** — a debug beat's `reasoning_log` contains a `world_turn` entry whose `rolls` includes **all three tiers** (including non-firing), and the clock delta matches; a non-debug beat has **no** `reasoning_log` key (reuse the existing non-debug assertion from `trace_test.go`).

- [ ] **Step 2: Run it, verify it fails** — `go test ./core/api/ -run 'TestTrace.*World' -v` → FAIL.

- [ ] **Step 3: Implement** — add the struct + nil-safe `appendWorldTurn` to `trace.go` (mirror the existing append methods); in `runWorldTurn`, when `trace != nil`, record the clock delta, every tier's `(chance, roll, fired)` from `rollTier`, fired scheduled events, and the eruption element. Serialize under the existing `reasoning_log` (no new gating — `beathandler` already emits it only in debug).

- [ ] **Step 4: Run tests** — `go test ./core/api/ -run 'TestTrace' -v` → PASS; confirm non-debug byte-shape unchanged.

- [ ] **Step 5: Commit** — `git commit -am "feat(livingworld): world's-turn trace block (debug-only, shows every pool roll for tuning) [U7]"`

---

## Self-Review

**1. Spec coverage:**
- U1 world-time → Tasks 1–3 (duration_class decode/schema/prompt; class→seconds config; non-move clock consumption + floor). ✓
- U2 ordering → Task 9 Step 4 (world's-turn after each attempt; §5 cut). ✓
- U3 ledger crossing → Task 4. ✓
- U4 pressure + roll → Tasks 5 (config/fire-log/chance) + 6 (deterministic roll). ✓ Derived-pressure (fire-log as last-eruption source), capped, per-tier, no tension-damping, master intensity/off — all in Task 5's config + `fn_pressure_chance`. ✓
- U5 World Actor → Tasks 7 (world slice) + 8 (seat, size constraint, truth+location invariant, presence-mover, same pipeline). ✓
- U6 composer → Task 9 (reusable `runWorldTurn`; no Journey logic). ✓
- U7 trace → Task 10 (debug-only, all three rolls). ✓

**2. Placeholder scan:** No TBD/TODO. Tasks 4, 7, 8 Step 3 describe SQL/routing by the exact functions to consume + a concrete failing test that pins correctness (TDD); the implementer discovers the payload-commit line via the named `applyEvent`/`adjudicate`/`fn_fact_sheet` patterns. Acceptable under TDD — the test defines success.

**3. Type consistency:** `runWorldTurn` (Task 9) consumes `fireDuePending` (Task 4), `rollTier` (Task 6), `runWorldActor` (Task 8) — signatures match. `firedMag`/magnitude strings are the same `small|medium|large` set throughout. `world_eruption` written in Task 9, read by `fn_pressure_chance` in Task 5 — same columns `(world_id, tier, fired_tick, event_id)`. `duration_class` enum identical in Task 1 (Go), Task 1 (schema JSON), Task 2 (SQL CHECK). `TicksAdvanced` semantics preserved across Task 3.

**Refinement noted for the founder (flagged, not silent):** the design said "last-eruption tick is derived from the eruption event itself." The plan makes that concrete as a tiny **append-only `world_eruption` fire-log** (written only on a fire, never per beat) — the modular home for the last-eruption lookup + the §7b "logged" audit, symmetric to `pending_event`. Still no per-beat pressure-state churn; pressure remains derived. If you'd rather tag `canon_event` directly instead of a side table, that's a one-task swap.

## Notes for execution
- Branch `feat/living-world` off `feat/grounded-reasoning` (`ce9568e`) before Task 1.
- pgTAP counts: each new `NN_*_test.sql` adds to the suite total — update no hardcoded count (the suite globs `*_test.sql`).
- Run `make schema-check` after every migration task; commit `schema.sql` regen with the migration.
