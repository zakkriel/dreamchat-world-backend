BEGIN;
SELECT plan(6);

-- Grounded Reasoning / Unit 1 (the Fact Sheet): fn_fact_sheet computes the deterministic physics facts
-- AMONG an action's involved entities and hands them over as one action-scoped jsonb sheet. It stacks on
-- the Station F contract functions (fn_distance, fn_move_duration_actor, fn_portal_permits,
-- fn_effective_weight, fn_volume) — pure SQL, NO LLM, no seat reference. One `targets` entry per involved
-- id EXCEPT the viewer; `budget_remaining` is ALWAYS json null here (the orchestrator fills it later —
-- it is beat state, not an entity fact). The ONE flavor difference in v1 is the perception gate on a
-- container's `contents`: truth-side always shows the id array; perceived-side WITHHOLDS it (json null)
-- unless the container is open.
--
-- Self-contained fixture (fixed f6000000-prefixed uuids, no seed dependency), ROLLBACK at end. Mirrors
-- 111/113/114 geometry + physics:
--   world                         f6000000-ffff-0000-0000-000000000000
--   viewer V (actor)  {0,0}        f6000000-0000-0000-0000-000000000001  max_load 80, at the tavern
--   loc tavern (V's scene)         f6000000-0000-0000-0000-000000000010
--   loc cellar                     f6000000-0000-0000-0000-000000000011
--   object "bar"      {8,0}        f6000000-0000-0000-0000-0000000000b1  same scene as V
--   portal "hatch"                 f6000000-0000-0000-0000-0000000000c3  connects[tavern,cellar] open=F locked=T
--   OPEN container "chest"         f6000000-0000-0000-0000-0000000000c4  max_room 4, open=true  → holds coin
--   CLOSED container "crate"       f6000000-0000-0000-0000-0000000000c5  max_room 4, open=false → holds stone
--   coin  (in chest)               f6000000-0000-0000-0000-0000000000e1
--   stone (in crate)               f6000000-0000-0000-0000-0000000000e2
--   heavy "anvil"     {3,0}        f6000000-0000-0000-0000-0000000000a2  size 1, weight 100 (100 > 80)

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('f6000000-0000-0000-0000-000000000001','f6000000-ffff-0000-0000-000000000000','actor',   'V-119'),
  ('f6000000-0000-0000-0000-000000000010','f6000000-ffff-0000-0000-000000000000','location','tavern-119'),
  ('f6000000-0000-0000-0000-000000000011','f6000000-ffff-0000-0000-000000000000','location','cellar-119'),
  ('f6000000-0000-0000-0000-0000000000b1','f6000000-ffff-0000-0000-000000000000','artifact','bar-119'),
  ('f6000000-0000-0000-0000-0000000000c3','f6000000-ffff-0000-0000-000000000000','artifact','hatch-119'),
  ('f6000000-0000-0000-0000-0000000000c4','f6000000-ffff-0000-0000-000000000000','artifact','chest-119'),
  ('f6000000-0000-0000-0000-0000000000c5','f6000000-ffff-0000-0000-000000000000','artifact','crate-119'),
  ('f6000000-0000-0000-0000-0000000000e1','f6000000-ffff-0000-0000-000000000000','artifact','coin-119'),
  ('f6000000-0000-0000-0000-0000000000e2','f6000000-ffff-0000-0000-000000000000','artifact','stone-119'),
  ('f6000000-0000-0000-0000-0000000000a2','f6000000-ffff-0000-0000-000000000000','artifact','anvil-119');

-- tavern is the root scene; cellar is a child (only referenced by the hatch's connects array).
INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('f6000000-0000-0000-0000-000000000010','f6000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0}}'::jsonb),
  ('f6000000-0000-0000-0000-000000000011','f6000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0},"parent_location_id":"f6000000-0000-0000-0000-000000000010"}'::jsonb);

-- viewer V at the tavern, coord {0,0}, max_load 80, no statuses (→ walk speed is the clean base, 1.4).
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('f6000000-0000-0000-0000-000000000001','f6000000-ffff-0000-0000-000000000000',
   '{"location_id":"f6000000-0000-0000-0000-000000000010","coordinates":{"x":0,"y":0},"max_load":80}'::jsonb);

INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
  -- bar: a positioned object 8 m east of V in the same scene (spatial facts only; no size/portal/room).
  ('f6000000-0000-0000-0000-0000000000b1','f6000000-ffff-0000-0000-000000000000',
   '{"location_id":"f6000000-0000-0000-0000-000000000010","coordinates":{"x":8,"y":0}}'::jsonb),
  -- hatch: a Portal (has `connects`) — open=false, locked=true. NO coordinates/location: Portal is
  -- accessibility, NOT geometry (§5.3). Its scene is unresolved → not same-scene, and the locked hatch
  -- permits no passage → reachable=false.
  ('f6000000-0000-0000-0000-0000000000c3','f6000000-ffff-0000-0000-000000000000',
   jsonb_build_object('open', false, 'locked', true,
     'connects', jsonb_build_array('f6000000-0000-0000-0000-000000000010',
                                    'f6000000-0000-0000-0000-000000000011'))),
  -- chest: an OPEN container (max_room 4, open=true) holding the coin.
  ('f6000000-0000-0000-0000-0000000000c4','f6000000-ffff-0000-0000-000000000000',
   '{"location_id":"f6000000-0000-0000-0000-000000000010","coordinates":{"x":1,"y":0},"max_room":4,"open":true}'::jsonb),
  -- crate: a CLOSED container (max_room 4, open=false) holding the stone.
  ('f6000000-0000-0000-0000-0000000000c5','f6000000-ffff-0000-0000-000000000000',
   '{"location_id":"f6000000-0000-0000-0000-000000000010","coordinates":{"x":2,"y":0},"max_room":4,"open":false}'::jsonb),
  -- coin inside the chest; stone inside the crate (contents = artifacts whose contained_by = container).
  ('f6000000-0000-0000-0000-0000000000e1','f6000000-ffff-0000-0000-000000000000',
   '{"contained_by":"f6000000-0000-0000-0000-0000000000c4"}'::jsonb),
  ('f6000000-0000-0000-0000-0000000000e2','f6000000-ffff-0000-0000-000000000000',
   '{"contained_by":"f6000000-0000-0000-0000-0000000000c5"}'::jsonb),
  -- anvil: a heavy object (size 1, weight 100). 100 > V's max_load 80 → would_encumber.
  ('f6000000-0000-0000-0000-0000000000a2','f6000000-ffff-0000-0000-000000000000',
   '{"location_id":"f6000000-0000-0000-0000-000000000010","coordinates":{"x":3,"y":0},"size":1,"weight":100}'::jsonb);

-- Seeded vocabulary for this world: the default walk type (1.4 m/s), so fn_move_duration_actor resolves
-- V's speed (no active statuses → exactly the base).
INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps) VALUES
  ('f6000000-ffff-0000-0000-000000000000', 'walk', 1.4);

-- ──────────────────────────────────────────────────────────────────────────────
-- (a) TRUTH: bar — spatial facts. distance_m = fn_distance(V,bar) = 8; move_duration_s =
--     fn_move_duration_actor(V,bar) = CEIL(8 / 1.4) = 6; reachable = same-scene → true.
-- ──────────────────────────────────────────────────────────────────────────────
WITH fs AS (
  SELECT fn_fact_sheet('f6000000-ffff-0000-0000-000000000000',
                       'f6000000-0000-0000-0000-000000000001',
                       ARRAY['f6000000-0000-0000-0000-0000000000b1',
                             'f6000000-0000-0000-0000-0000000000c3',
                             'f6000000-0000-0000-0000-0000000000c4',
                             'f6000000-0000-0000-0000-0000000000c5',
                             'f6000000-0000-0000-0000-0000000000a2']::uuid[],
                       true) AS j
), bar AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000b1'
)
SELECT ok(
  (SELECT elem->>'distance_m'      FROM bar)::numeric = 8::numeric
  AND (SELECT elem->>'move_duration_s' FROM bar)::bigint = 6::bigint
  AND (SELECT elem->'reachable'    FROM bar) = 'true'::jsonb,
  '(a) truth: bar distance_m=8, move_duration_s=6 (CEIL(8/1.4)), reachable=true');

-- ──────────────────────────────────────────────────────────────────────────────
-- (b) TRUTH: hatch — a Portal. open/locked come from Tier-1 attrs; reachable=false (locked + not
--     same-scene).
-- ──────────────────────────────────────────────────────────────────────────────
WITH fs AS (
  SELECT fn_fact_sheet('f6000000-ffff-0000-0000-000000000000',
                       'f6000000-0000-0000-0000-000000000001',
                       ARRAY['f6000000-0000-0000-0000-0000000000b1',
                             'f6000000-0000-0000-0000-0000000000c3',
                             'f6000000-0000-0000-0000-0000000000c4',
                             'f6000000-0000-0000-0000-0000000000c5',
                             'f6000000-0000-0000-0000-0000000000a2']::uuid[],
                       true) AS j
), hatch AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000c3'
)
SELECT ok(
  (SELECT elem->'open'      FROM hatch) = 'false'::jsonb
  AND (SELECT elem->'locked'    FROM hatch) = 'true'::jsonb
  AND (SELECT elem->'reachable' FROM hatch) = 'false'::jsonb,
  '(b) truth: hatch open=false, locked=true, reachable=false');

-- ──────────────────────────────────────────────────────────────────────────────
-- (c) TRUTH: anvil — an object. would_encumber = effective_weight(100) > V.max_load(80) → true.
-- ──────────────────────────────────────────────────────────────────────────────
WITH fs AS (
  SELECT fn_fact_sheet('f6000000-ffff-0000-0000-000000000000',
                       'f6000000-0000-0000-0000-000000000001',
                       ARRAY['f6000000-0000-0000-0000-0000000000b1',
                             'f6000000-0000-0000-0000-0000000000c3',
                             'f6000000-0000-0000-0000-0000000000c4',
                             'f6000000-0000-0000-0000-0000000000c5',
                             'f6000000-0000-0000-0000-0000000000a2']::uuid[],
                       true) AS j
), anvil AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000a2'
)
SELECT ok(
  (SELECT elem->'would_encumber' FROM anvil) = 'true'::jsonb,
  '(c) truth: anvil would_encumber=true (effective_weight 100 > max_load 80)');

-- ──────────────────────────────────────────────────────────────────────────────
-- (d) TRUTH: crate — a container. contents is ALWAYS the id array on the truth side, closed or not.
-- ──────────────────────────────────────────────────────────────────────────────
WITH fs AS (
  SELECT fn_fact_sheet('f6000000-ffff-0000-0000-000000000000',
                       'f6000000-0000-0000-0000-000000000001',
                       ARRAY['f6000000-0000-0000-0000-0000000000b1',
                             'f6000000-0000-0000-0000-0000000000c3',
                             'f6000000-0000-0000-0000-0000000000c4',
                             'f6000000-0000-0000-0000-0000000000c5',
                             'f6000000-0000-0000-0000-0000000000a2']::uuid[],
                       true) AS j
), crate AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000c5'
)
SELECT is(
  (SELECT elem->'contents' FROM crate),
  jsonb_build_array('f6000000-0000-0000-0000-0000000000e2'::text),
  '(d) truth: crate contents = ["<stone id>"] (truth-side always shows contents)');

-- ──────────────────────────────────────────────────────────────────────────────
-- (e) PERCEIVED (p_truth_side=false): crate is CLOSED → contents WITHHELD as json null (the wall).
-- ──────────────────────────────────────────────────────────────────────────────
WITH fs AS (
  SELECT fn_fact_sheet('f6000000-ffff-0000-0000-000000000000',
                       'f6000000-0000-0000-0000-000000000001',
                       ARRAY['f6000000-0000-0000-0000-0000000000b1',
                             'f6000000-0000-0000-0000-0000000000c3',
                             'f6000000-0000-0000-0000-0000000000c4',
                             'f6000000-0000-0000-0000-0000000000c5',
                             'f6000000-0000-0000-0000-0000000000a2']::uuid[],
                       false) AS j
), crate AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000c5'
)
SELECT ok(
  (SELECT jsonb_typeof(elem->'contents') FROM crate) = 'null',
  '(e) perceived: crate contents IS json null (closed container → withheld)');

-- ──────────────────────────────────────────────────────────────────────────────
-- (f) PERCEIVED: chest is OPEN → contents SHOWN. And spatial facts (bar/hatch) are unchanged from the
--     truth flavor — only container contents differs by flavor in v1.
-- ──────────────────────────────────────────────────────────────────────────────
WITH fs AS (
  SELECT fn_fact_sheet('f6000000-ffff-0000-0000-000000000000',
                       'f6000000-0000-0000-0000-000000000001',
                       ARRAY['f6000000-0000-0000-0000-0000000000b1',
                             'f6000000-0000-0000-0000-0000000000c3',
                             'f6000000-0000-0000-0000-0000000000c4',
                             'f6000000-0000-0000-0000-0000000000c5',
                             'f6000000-0000-0000-0000-0000000000a2']::uuid[],
                       false) AS j
), chest AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000c4'
), bar AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000b1'
), hatch AS (
  SELECT elem FROM fs CROSS JOIN LATERAL jsonb_array_elements(fs.j->'targets') AS t(elem)
  WHERE elem->>'id' = 'f6000000-0000-0000-0000-0000000000c3'
)
SELECT ok(
  (SELECT elem->'contents' FROM chest) = jsonb_build_array('f6000000-0000-0000-0000-0000000000e1'::text)
  AND (SELECT elem->>'distance_m'      FROM bar)::numeric = 8::numeric
  AND (SELECT elem->>'move_duration_s' FROM bar)::bigint  = 6::bigint
  AND (SELECT elem->'reachable'        FROM bar)   = 'true'::jsonb
  AND (SELECT elem->'reachable'        FROM hatch) = 'false'::jsonb,
  '(f) perceived: chest contents = ["<coin id>"] (open → shown); bar/hatch spatial facts unchanged');

SELECT * FROM finish();
ROLLBACK;
