BEGIN;
SELECT plan(6);

-- Station F / FINAL-action-contracts.md §8 (Minting: persistence + the three nets). apply_mint is the
-- persistence half of minting (the Go validateMints is the shape+bounds half). This proves the physics
-- vocabulary a ruling mints becomes REAL committed rows, audit-trailed to the ruling event (net 3), and
-- that a subsequent fn_effective_speed reads the minted modifier — i.e. move arithmetic now runs against
-- MINTED vocabulary, not just the two seeded rows. Self-contained fixture (fixed uuids, no seed dep).
--
--   world fb11…, ruling event fb11…e1 (the provenance anchor), actor A (statuses: [sure_footed]).
--   mint 1: movement type   { climb, baseSpeed 0.4 }
--   mint 2: status modifier { sure_footed, move, [{ climb, +20% }] }  (references climb minted first)

-- Provenance anchor: a canon_event the mints are audit-trailed to (created_by_event = this event).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick)
VALUES ('fb111111-0000-0000-0000-0000000000e1', 'fb111111-ffff-0000-0000-000000000000',
        'AttributeChanged', 'ruling that mints climb + sure_footed', 0);

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('fb111111-0000-0000-0000-0000000000a1', 'fb111111-ffff-0000-0000-000000000000', 'actor', 'A');
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  ('fb111111-0000-0000-0000-0000000000a1', 'fb111111-ffff-0000-0000-000000000000',
   '{"statuses":["sure_footed"]}'::jsonb);

-- ── Mint 1: the movement type (must land BEFORE the modifier so the FK is satisfied — mint-ordering, §8).
SELECT is(
  apply_mint('fb111111-ffff-0000-0000-000000000000', 'fb111111-0000-0000-0000-0000000000e1',
             '{"movementTypeId":"climb","baseSpeed":0.4}'::jsonb),
  'climb',
  '(a) apply_mint(movement type) returns the minted movement_type_id');

-- (b) the movement_type row committed with the right base speed AND audit-trailed to the ruling event.
SELECT ok(
  EXISTS (SELECT 1 FROM movement_type
          WHERE world_id = 'fb111111-ffff-0000-0000-000000000000'
            AND movement_type_id = 'climb'
            AND base_speed_mps = 0.4
            AND created_by_event = 'fb111111-0000-0000-0000-0000000000e1'),
  '(b) movement_type climb committed, base_speed 0.4, created_by_event = ruling (net 3)');

-- ── Mint 2: the modifier referencing the just-minted climb. FK (world, movement_type) is satisfied
--    because mint 1 already inserted climb in this same (test) transaction — the ordering guarantee.
SELECT lives_ok($$
  SELECT apply_mint('fb111111-ffff-0000-0000-000000000000', 'fb111111-0000-0000-0000-0000000000e1',
    '{"statusTypeId":"sure_footed","actionType":"move","movementModifiers":[{"movementTypeId":"climb","modifierPercent":20}]}'::jsonb)
$$, '(c) apply_mint(modifier referencing climb) commits (FK satisfied by ordering)');

-- (d) the status_modifier row committed with the right percent AND audit-trailed to the ruling event.
SELECT ok(
  EXISTS (SELECT 1 FROM status_modifier
          WHERE world_id = 'fb111111-ffff-0000-0000-000000000000'
            AND status_type_id = 'sure_footed' AND action_type = 'move'
            AND movement_type_id = 'climb' AND modifier_percent = 20
            AND created_by_event = 'fb111111-0000-0000-0000-0000000000e1'),
  '(d) status_modifier sure_footed/climb committed, +20%, created_by_event = ruling (net 3)');

-- (e) THE PAYOFF: fn_effective_speed reads the MINTED modifier — A is sure_footed, so climbing is
--     0.4 * (1 + 20/100) = 0.4 * 1.2 = 0.48 m/s. Minting is now real end-to-end for move arithmetic.
SELECT is(
  fn_effective_speed('fb111111-ffff-0000-0000-000000000000',
                     'fb111111-0000-0000-0000-0000000000a1', 'climb'),
  0.48::numeric,
  '(e) fn_effective_speed(A, climb) = 0.4 * 1.20 = 0.48 (reads the minted modifier)');

-- (f) an ARTIFACT/place mint RAISEs — persistence is escalated (entity_registry+state schema undefined in
--     §8); apply_mint refuses to guess, so a well-formed-but-unpersistable artifact mint rolls back the
--     whole ruling (fail-safe) rather than landing a guessed row.
SELECT throws_ok($$
  SELECT apply_mint('fb111111-ffff-0000-0000-000000000000', 'fb111111-0000-0000-0000-0000000000e1',
    '{"locationId":"fb111111-0000-0000-0000-0000000000c9","size":2,"maxRoom":4,"coordinate":{"x":1,"y":1}}'::jsonb)
$$, NULL, NULL,
  '(f) artifact/place mint RAISEs (persistence escalated — refuses to guess the schema)');

SELECT * FROM finish();
ROLLBACK;
