BEGIN;
SELECT plan(6);

-- Station F / §3 Space: nested coordinates + fn_distance. Self-contained fixture (fixed uuids, no seed
-- dependency), mirroring the nested-frame model from FINAL-action-contracts.md §3:
--   district (root, coord {0,0}, extent 2000x2000)
--     ├─ tavern  (child, coord {100,0} in district)
--     └─ alley   (child, coord {900,0} in district)
-- actor A stands inside the tavern at {2,0}; artifact "bar" sits inside the tavern at {14,0}.
-- Coordinates live in attrs JSONB as Tier-1 measurements (actor/artifact position-in-scene;
-- location coordinate-in-parent + parent_location_id edge).

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('fa000000-0000-0000-0000-0000000000d1', 'fa000000-ffff-0000-0000-000000000000', 'location', 'district'),
  ('fa000000-0000-0000-0000-0000000000c1', 'fa000000-ffff-0000-0000-000000000000', 'location', 'tavern'),
  ('fa000000-0000-0000-0000-0000000000c2', 'fa000000-ffff-0000-0000-000000000000', 'location', 'alley'),
  ('fa000000-0000-0000-0000-0000000000a1', 'fa000000-ffff-0000-0000-000000000000', 'actor',    'A'),
  ('fa000000-0000-0000-0000-0000000000b1', 'fa000000-ffff-0000-0000-000000000000', 'artifact', 'bar');

-- district: root (no parent_location_id), sits at its own origin, extent bounds its children.
INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('fa000000-0000-0000-0000-0000000000d1', 'fa000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0},"extent":{"w":2000,"h":2000}}'::jsonb);
-- tavern + alley: children of district, each with a coordinate WITHIN the district frame.
INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('fa000000-0000-0000-0000-0000000000c1', 'fa000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":100,"y":0},"parent_location_id":"fa000000-0000-0000-0000-0000000000d1"}'::jsonb),
  ('fa000000-0000-0000-0000-0000000000c2', 'fa000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":900,"y":0},"parent_location_id":"fa000000-0000-0000-0000-0000000000d1"}'::jsonb);
-- actor A + artifact bar: both inside the tavern (location_id = tavern), each with an in-scene coordinate.
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('fa000000-0000-0000-0000-0000000000a1', 'fa000000-ffff-0000-0000-000000000000',
   '{"location_id":"fa000000-0000-0000-0000-0000000000c1","coordinates":{"x":2,"y":0}}'::jsonb);
INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
  ('fa000000-0000-0000-0000-0000000000b1', 'fa000000-ffff-0000-0000-000000000000',
   '{"location_id":"fa000000-0000-0000-0000-0000000000c1","coordinates":{"x":14,"y":0}}'::jsonb);

-- (a) breadcrumb depth: root = 0, one hop down = 1.
SELECT ok(
  fn_location_depth('fa000000-ffff-0000-0000-000000000000', 'fa000000-0000-0000-0000-0000000000d1') = 0
  AND
  fn_location_depth('fa000000-ffff-0000-0000-000000000000', 'fa000000-0000-0000-0000-0000000000c1') = 1,
  'fn_location_depth: district (root) = 0, tavern (one hop) = 1');

-- (b) A and the bar share a scene → nearest common parent is the tavern itself.
SELECT is(
  fn_nearest_common_parent('fa000000-ffff-0000-0000-000000000000',
    'fa000000-0000-0000-0000-0000000000a1', 'fa000000-0000-0000-0000-0000000000b1'),
  'fa000000-0000-0000-0000-0000000000c1'::uuid,
  'fn_nearest_common_parent(A, bar) = tavern (same scene)');

-- (c) same-scene distance = their own in-scene coordinates: |14 - 2| = 12.
SELECT is(
  fn_distance('fa000000-ffff-0000-0000-000000000000',
    'fa000000-0000-0000-0000-0000000000a1', 'fa000000-0000-0000-0000-0000000000b1'),
  12::numeric,
  'fn_distance(A, bar) = 12 (in-scene coordinates, 14 - 2)');

-- (d) two sibling locations → nearest common parent climbs to the district.
SELECT is(
  fn_nearest_common_parent('fa000000-ffff-0000-0000-000000000000',
    'fa000000-0000-0000-0000-0000000000c1', 'fa000000-0000-0000-0000-0000000000c2'),
  'fa000000-0000-0000-0000-0000000000d1'::uuid,
  'fn_nearest_common_parent(tavern, alley) = district');

-- (e) cross-level distance = the two CHILD LOCATIONS' coordinates in the district frame: |900 - 100| = 800.
SELECT is(
  fn_distance('fa000000-ffff-0000-0000-000000000000',
    'fa000000-0000-0000-0000-0000000000c1', 'fa000000-0000-0000-0000-0000000000c2'),
  800::numeric,
  'fn_distance(tavern, alley) = 800 (child-location coordinates at the district frame)');

-- (f) CYCLE-GUARD REGRESSION (carried-forward Task-1 review item, backstop end): a location whose
-- parent_location_id points at ITSELF must NOT infinite-loop the parent-walk. The defensive depth cap
-- (mint_persistence migration, mirroring the I-4 acyclicity guard) makes the recursion TERMINATE and
-- return bounded — fn_location_depth stops at the cap (64) instead of recursing forever. Before the cap
-- this query never returned (runaway recursion). The primary guard is validateMints rejecting the cycle
-- at mint time; this proves the engine can't hang even if a bad row slips through.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('fa000000-0000-0000-0000-0000000000f9', 'fa000000-ffff-0000-0000-000000000000', 'location', 'loop_room');
INSERT INTO location_state (entity_id, world_id, attrs) VALUES
  ('fa000000-0000-0000-0000-0000000000f9', 'fa000000-ffff-0000-0000-000000000000',
   '{"coordinates":{"x":0,"y":0},"parent_location_id":"fa000000-0000-0000-0000-0000000000f9"}'::jsonb);

SELECT is(
  fn_location_depth('fa000000-ffff-0000-0000-000000000000', 'fa000000-0000-0000-0000-0000000000f9'),
  64,
  '(f) self-parent location: fn_location_depth returns bounded (cap 64), never infinite-loops');

SELECT * FROM finish();
ROLLBACK;
