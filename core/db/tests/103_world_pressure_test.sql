BEGIN;
SELECT plan(18);

-- Living World Task 5: per-world pressure config + append-only fire-log
-- (world_eruption) + fn_pressure_chance. Pressure is DERIVED — "how much
-- world-time has passed since that tier last erupted" — never a stored
-- counter. Self-contained fixture on the shared seeded world.
-- world:   22222222-2222-2222-2222-222222222222
-- unconfigured world (assertion f only): 44444444-4444-4444-4444-444444444444
--   (NEVER passed to seed_world_defaults — no world_actor_config row at all,
--   exercises the outer COALESCE NULL-safety net rather than the formula)
--
-- Seeded pressure config for this world (via seed_world_defaults):
--   small:  climb_rate=0.01, climb_chunk_ticks=60,    cap=0.70
--   medium: climb_rate=0.01, climb_chunk_ticks=3600,  cap=0.70
--   large:  climb_rate=0.01, climb_chunk_ticks=86400, cap=0.70
--   setting: enabled=true, intensity=1.0 (defaults)
--
-- Tick points below are chosen as exact multiples of climb_chunk_ticks so the
-- expected numeric values are exact (no recurring-decimal rounding to worry
-- about in the is() comparisons).
--
-- Execution order deliberately differs from the brief's (a)-(e) letter order:
-- (e) independence must run BEFORE (d) disables the setting, because
-- world_actor_setting is a per-world (not per-tier) row — disabling it after
-- (e) would zero every tier and make the independence check meaningless.
-- (d) therefore runs last. (g) — the whole-branch-review intensity-cap
-- invariant, added after Task 5 — runs right before (d) for the same reason:
-- it needs enabled=true (the default) to isolate intensity's own effect, so
-- it must run before (d) flips enabled to false.

-- (a) seed_world_defaults(w) populates three tiers + one setting.
SELECT seed_world_defaults('22222222-2222-2222-2222-222222222222');

SELECT is(
  (SELECT count(*)::int FROM world_actor_config
   WHERE world_id = '22222222-2222-2222-2222-222222222222'
     AND tier IN ('small','medium','large')),
  3,
  '(a1) seed_world_defaults(w) seeds all three pressure tiers');

SELECT is(
  (SELECT count(*)::int FROM world_actor_setting
   WHERE world_id = '22222222-2222-2222-2222-222222222222'),
  1,
  '(a2) seed_world_defaults(w) seeds exactly one world_actor_setting row');

-- (b) with no prior eruption, fn_pressure_chance('small') rises as now grows
-- and never exceeds the seeded cap (0.70).
SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',60),
  0.01,
  '(b1) small@1 climb_chunk (tick 60) = climb_rate * 1 = 0.01');

SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',1800),
  0.30,
  '(b2) small@30 climb_chunks (tick 1800) = climb_rate * 30 = 0.30 (still under cap)');

SELECT ok(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',1800)
    > fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',60),
  '(b3) chance strictly rises as now grows');

SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',6000),
  0.70,
  '(b4) small@100 climb_chunks (tick 6000) is capped at the seeded cap 0.70 (raw climb would be 1.0)');

SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',100000000),
  0.70,
  '(b5) far beyond the cap-crossing point, chance is still exactly 0.70 — never exceeds the cap');

-- (c) a world_eruption row at tick T drops the chance at T back toward 0
-- (drain), then it resumes climbing from that new baseline — proving the
-- accrual is derived from the fire-log, not a persisted counter.
SELECT lives_ok(
  $$INSERT INTO world_eruption (world_id, tier, fired_tick, event_id)
    VALUES ('22222222-2222-2222-2222-222222222222','small',6000,gen_random_uuid())$$,
  '(c0) inserting a world_eruption row (the fire-log) succeeds');

SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',6000),
  0::numeric,
  '(c1) at the fired_tick itself, elapsed time since last eruption is 0 — chance drains to 0');

SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',6060),
  0.01,
  '(c2) one climb_chunk after the eruption, chance resumes climbing from the new baseline (0.01)');

-- (e) pools are independent — the 'small' eruption inserted above does not
-- change fn_pressure_chance for 'large'. tick 864000 = exactly 10 climb_chunks
-- of large's 86400-tick chunk, with zero eruptions ever logged for 'large'.
SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','large',864000),
  0.10,
  '(e) large tier is unaffected by the small-tier eruption above — pools are independent');

-- (g) whole-branch review Fix 2: intensity must stay INSIDE the cap — the cap
-- is the ceiling, so nothing is ever a guaranteed eruption. Bumping intensity
-- to 5.0 (world_actor_setting is per-world; 'small' alone is enough to prove
-- the invariant) and evaluating at a saturating elapsed time (p_now far past
-- the cap-crossing point, same shape as (b5) above) — pre-fix this multiplied
-- an ALREADY-capped chance by intensity (LEAST(cap, raw) * intensity), which
-- for intensity=5.0 would return 3.5, ten times past a "chance" even being a
-- valid probability. Post-fix, intensity multiplies BEFORE the LEAST
-- (LEAST(cap, raw * intensity)), so the result can never exceed cap.
SELECT lives_ok(
  $$UPDATE world_actor_setting SET intensity = 5.0
    WHERE world_id = '22222222-2222-2222-2222-222222222222'$$,
  '(g0) setting intensity to 5.0 succeeds');

SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',100000000),
  0.70,
  '(g1) intensity=5.0 at a saturating elapsed time still returns exactly the seeded cap 0.70 — intensity never pushes the chance past the ceiling');

SELECT ok(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',100000000) <= 0.70,
  '(g2) belt-and-suspenders: the chance is never greater than the seeded cap, even with intensity > 1 (the cap is the ceiling — nothing is ever a guaranteed eruption)');

-- (d) world_actor_setting.enabled=false forces chance to exactly 0, world-wide
-- (the setting is per-world, not per-tier) — run last, after (e), since it
-- would otherwise zero every tier's chance and make (e) meaningless.
SELECT lives_ok(
  $$UPDATE world_actor_setting SET enabled = false
    WHERE world_id = '22222222-2222-2222-2222-222222222222'$$,
  '(d0) disabling the world_actor_setting row succeeds');

SELECT is(
  fn_pressure_chance('22222222-2222-2222-2222-222222222222','small',6060),
  0::numeric,
  '(d1) disabled setting forces small-tier chance to exactly 0 despite climbed pressure');

-- (f) a completely unconfigured world (no world_actor_config row at all, for
-- any tier) must return exactly 0, NOT NULL — a missing config means "no
-- eruptions ever", the safe default. Task 6 will do `roll < fn_pressure_chance
-- (...)`; a NULL there either fails the Go float64 scan or silently never
-- fires. Sanity-check the world really is unconfigured first, same pattern as
-- 102_duration_class_test.sql's unseeded-world fallback check.
SELECT is(
  (SELECT count(*)::int FROM world_actor_config
   WHERE world_id = '44444444-4444-4444-4444-444444444444'),
  0,
  '(f0) unconfigured world has no world_actor_config rows (sanity: NULL-safety path is live)');

SELECT is(
  fn_pressure_chance('44444444-4444-4444-4444-444444444444','small',1000),
  0::numeric,
  '(f1) fn_pressure_chance on a totally unconfigured world returns exactly 0, never NULL');

SELECT * FROM finish();
ROLLBACK;
