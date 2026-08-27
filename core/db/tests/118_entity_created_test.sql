-- =====================================================================================
-- 118_entity_created_test.sql — Station F Task 9: entity creation on resolve.
-- "When it resolves it updates it all" (founder). Until now the EntityCreated branch in both
-- commit twins was an EMPTY STUB (NULL) — a ruling could decide "a new thing exists" and NOTHING
-- persisted it. This makes the branch REAL: a ruled EntityCreated INSERTs an entity_registry row
-- (provenance-stamped created_by_event, §8 net 3) + its positioned initial state (coordinates +
-- Tier-1 measurements), reusing an existing id when the create matches one (reuse-before-create,
-- §5.4 / doc-05 matcher). A create with no descriptor is not a true introduction → gate_reject.
--
-- CLEAN SEPARATION: EntityCreated creates INSTANCES (registry row + state); apply_mint (Task 6)
-- creates typed VOCABULARY (movement types / status modifiers). Different writers, no overlap.
--
-- Runs against the SEEDED, COMMITTED db (like 116/117) inside a BEGIN/ROLLBACK envelope, reading
-- the seed's fixed uuids. Ticks 7000+ are well above the seed's max (Kade arrives at tick 50) and
-- clear of 117's 6000-block.
--   docs/law/rulings/FINAL-action-contracts.md §8 (minting/three nets) + §5.4
--   docs/law/rulings/FINAL-interaction-loop-PRD.md R3/R4 (doc-05 matcher)
--   core/db/seeds/seed_drowned_lantern.sql
--
-- Fixed seed uuids:
--   world 22222222-…   Kade a1 (in the tavern d1, coordinate {6,1})
--   The Drowned Lantern (tavern) …-d1
-- Minted (LLM-side, not in the seed) ids for the created instances:
--   NEW1 tankard  2a7f0000-…-e1     NEW2 dup       2a7f0000-…-e2     NEW3 no-descriptor 2a7f0000-…-e3
--
-- Each committing create is its OWN statement (captured into a temp table) BEFORE the assertion
-- reads state — a subquery in the SAME statement as apply_ruled_event would read the pre-commit
-- snapshot and never see the write (same discipline as 117).
-- =====================================================================================
BEGIN;
SELECT plan(5);

-- (a) A ruled EntityCreated COMMITS an entity_registry row with created_by_event set. Kade forges a
-- chipped clay tankard: a true introduction (descriptor present), minted at the bar's spot {6,9} in
-- the tavern. Provenance = the create event (§8 net 3): entity_registry.created_by_event = event_id.
CREATE TEMP TABLE r_a AS SELECT apply_ruled_event(
    '22222222-2222-2222-2222-222222222222',
    '{"type":"EntityCreated","actor_id":"2ac70000-0000-0000-0000-0000000000a1",
      "truth":"Kade sets a chipped clay tankard on the bar",
      "target_id":"2a7f0000-0000-0000-0000-0000000000e1",
      "new_entity_kind":"artifact","canonical_name":"clay tankard","descriptor":"a chipped clay tankard",
      "new_attrs":{"location_id":"210c0000-0000-0000-0000-0000000000d1","coordinates":{"x":6,"y":9},"size":1,"weight":1}}'::jsonb,
    7000, 0, 'ruling') AS res;
SELECT ok(
  (SELECT res->>'halt_reason' FROM r_a) = 'committed'
  AND EXISTS (SELECT 1 FROM entity_registry
              WHERE world_id='22222222-2222-2222-2222-222222222222'
                AND entity_id='2a7f0000-0000-0000-0000-0000000000e1'
                AND entity_kind='artifact' AND status='active'
                AND descriptor='a chipped clay tankard'
                AND created_by_event = (SELECT (res->>'event_id')::uuid FROM r_a)),
  '(a) ruled EntityCreated commits an entity_registry row; created_by_event = the ruling event id (provenance §8 net 3)');

-- (b) Its STATE row exists with the ruled coordinates/attrs (positioned genesis state, §2/§3): the
-- artifact_state row carries coordinates {6,9}, its scene (location_id) = the tavern, and the Tier-1
-- measurements the ruling set (size 1, weight 1).
SELECT ok(
  (SELECT attrs->'coordinates' FROM artifact_state
     WHERE world_id='22222222-2222-2222-2222-222222222222'
       AND entity_id='2a7f0000-0000-0000-0000-0000000000e1') = '{"x":6,"y":9}'::jsonb
  AND (SELECT attrs->>'location_id' FROM artifact_state
         WHERE world_id='22222222-2222-2222-2222-222222222222'
           AND entity_id='2a7f0000-0000-0000-0000-0000000000e1') = '210c0000-0000-0000-0000-0000000000d1'
  AND (SELECT attrs->>'size' FROM artifact_state
         WHERE world_id='22222222-2222-2222-2222-222222222222'
           AND entity_id='2a7f0000-0000-0000-0000-0000000000e1') = '1',
  '(b) the created entity''s state row carries the ruled coordinates {6,9}, its scene, and size (Tier-1 measurements)');

-- (c) The new entity is IMMEDIATELY REACHABLE — fn_distance to it works because it has a position.
-- Kade {6,1} in the tavern → the tankard {6,9} in the same tavern = 8 m. A created entity with no
-- position would return NULL/0; a real few-meters distance proves the positioned state landed.
SELECT is(
  fn_distance('22222222-2222-2222-2222-222222222222',
              '2ac70000-0000-0000-0000-0000000000a1',   -- Kade
              '2a7f0000-0000-0000-0000-0000000000e1'),   -- the just-created tankard
  8::numeric,
  '(c) the new entity is reachable: fn_distance(Kade {6,1}, tankard {6,9}) = 8 (it has a position)');

-- (d) REUSE-BEFORE-CREATE (§5.4 / doc-05 matcher): a create that matches an existing entity (same
-- entity_kind + descriptor) REUSES that id — NO new row. A second "chipped clay tankard" (a DIFFERENT
-- minted id e2) must not mint a duplicate: e2 is never registered, and exactly one such tankard exists.
CREATE TEMP TABLE r_d AS SELECT apply_ruled_event(
    '22222222-2222-2222-2222-222222222222',
    '{"type":"EntityCreated","actor_id":"2ac70000-0000-0000-0000-0000000000a1",
      "truth":"Kade reaches for a chipped clay tankard",
      "target_id":"2a7f0000-0000-0000-0000-0000000000e2",
      "new_entity_kind":"artifact","canonical_name":"clay tankard","descriptor":"a chipped clay tankard",
      "new_attrs":{"location_id":"210c0000-0000-0000-0000-0000000000d1","coordinates":{"x":6,"y":9},"size":1,"weight":1}}'::jsonb,
    7001, 0, 'ruling') AS res;
SELECT ok(
  (SELECT res->>'halt_reason' FROM r_d) = 'committed'
  AND NOT EXISTS (SELECT 1 FROM entity_registry
                  WHERE world_id='22222222-2222-2222-2222-222222222222'
                    AND entity_id='2a7f0000-0000-0000-0000-0000000000e2')
  AND (SELECT count(*)::int FROM entity_registry
         WHERE world_id='22222222-2222-2222-2222-222222222222'
           AND entity_kind='artifact' AND descriptor='a chipped clay tankard') = 1,
  '(d) reuse-before-create: a create matching an existing (kind, descriptor) reuses the id — no new row');

-- (e) DESCRIPTOR MANDATORY (§8 / §7 provenance-guarded create): a create with NO descriptor is not a
-- true introduction → gate_reject, nothing written. Blocker-only proof: e3 is never registered.
SELECT ok(
  (apply_ruled_event(
    '22222222-2222-2222-2222-222222222222',
    '{"type":"EntityCreated","actor_id":"2ac70000-0000-0000-0000-0000000000a1",
      "truth":"a shape half-forms and never resolves",
      "target_id":"2a7f0000-0000-0000-0000-0000000000e3",
      "new_entity_kind":"artifact",
      "new_attrs":{"location_id":"210c0000-0000-0000-0000-0000000000d1","coordinates":{"x":6,"y":9}}}'::jsonb,
    7002, 0, 'ruling')->>'halt_reason') = 'gate_reject'
  AND NOT EXISTS (SELECT 1 FROM entity_registry
                  WHERE world_id='22222222-2222-2222-2222-222222222222'
                    AND entity_id='2a7f0000-0000-0000-0000-0000000000e3'),
  '(e) a create with no descriptor → gate_reject, nothing written (descriptor mandatory, §8)');

SELECT * FROM finish();
ROLLBACK;
