BEGIN;
SELECT plan(4);

-- Station F / FINAL-action-contracts.md §5.3 (Portal — traversal accessibility gate).
-- A move A→B passes the accessibility floor iff a Portal connecting A↔B permits passage
-- (`open` AND NOT `locked` in v1). This is what makes the locked door real for EVERYONE. The check
-- is computed fresh at ask-time (fn_portal_permits) — there is NO stored `reachable` column.
--
-- Portal is ACCESSIBILITY, NOT geometry: it contributes no exit points, no distance, no coordinates,
-- and it must never touch fn_distance. This fixture therefore gives the portals NO coordinates.
--
-- Fresh, self-contained fixture (f5000000-prefixed UUIDs), ROLLBACK at end.
--   world:            f5000000-ffff-0000-0000-000000000000
--   actor P (mover):  f5000000-0000-0000-0000-000000000001  (starts at the tavern)
--   loc tavern:       f5000000-0000-0000-0000-000000000010
--   loc cellar:       f5000000-0000-0000-0000-000000000011
--   loc dock_street:  f5000000-0000-0000-0000-000000000012
--   portal cellar-hatch: f5000000-0000-0000-0000-0000000000c3  (tavern↔cellar, open=false, locked=true)
--   portal front-door:   f5000000-0000-0000-0000-0000000000c1  (tavern↔dock_street, open=true, locked=false)

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('f5000000-0000-0000-0000-000000000001','f5000000-ffff-0000-0000-000000000000','actor',   'test-P-114'),
 ('f5000000-0000-0000-0000-000000000010','f5000000-ffff-0000-0000-000000000000','location','test-tavern-114'),
 ('f5000000-0000-0000-0000-000000000011','f5000000-ffff-0000-0000-000000000000','location','test-cellar-114'),
 ('f5000000-0000-0000-0000-000000000012','f5000000-ffff-0000-0000-000000000000','location','test-dock-street-114'),
 ('f5000000-0000-0000-0000-0000000000c3','f5000000-ffff-0000-0000-000000000000','artifact','test-cellar-hatch-114'),
 ('f5000000-0000-0000-0000-0000000000c1','f5000000-ffff-0000-0000-000000000000','artifact','test-front-door-114');

-- Portals as Tier-1 data (open/locked/connects). NO coordinates: Portal is accessibility, not geometry.
INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
 ('f5000000-0000-0000-0000-0000000000c3','f5000000-ffff-0000-0000-000000000000',
  jsonb_build_object('open', false, 'locked', true,
    'connects', jsonb_build_array('f5000000-0000-0000-0000-000000000010','f5000000-0000-0000-0000-000000000011'))),
 ('f5000000-0000-0000-0000-0000000000c1','f5000000-ffff-0000-0000-000000000000',
  jsonb_build_object('open', true, 'locked', false,
    'connects', jsonb_build_array('f5000000-0000-0000-0000-000000000010','f5000000-0000-0000-0000-000000000012')));

-- Position P at the tavern (setup move → the trigger projects it into actor_state, so the floor's
-- here-read sees the tavern as P's current location).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('f5000000-0000-0000-0000-000000000050','f5000000-ffff-0000-0000-000000000000','move','setup P→tavern',1000,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('f5000000-0000-0000-0000-000000000050','f5000000-0000-0000-0000-000000000001','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq)
VALUES ('f5000000-ffff-0000-0000-000000000000','f5000000-0000-0000-0000-000000000050','f5000000-0000-0000-0000-000000000001','actor','attrs.location_id',to_jsonb('f5000000-0000-0000-0000-000000000010'::text),1000,0);

-- (a) fn_portal_permits(tavern, dock_street) = true: the front door is open and unlocked.
SELECT ok(
  fn_portal_permits('f5000000-ffff-0000-0000-000000000000',
                    'f5000000-0000-0000-0000-000000000010',
                    'f5000000-0000-0000-0000-000000000012'),
  '(a) fn_portal_permits(tavern, dock_street) = true (front door open ∧ ¬locked)'
);

-- (b) fn_portal_permits(tavern, cellar) = false: the hatch connects them but is LOCKED.
SELECT ok(
  NOT fn_portal_permits('f5000000-ffff-0000-0000-000000000000',
                        'f5000000-0000-0000-0000-000000000010',
                        'f5000000-0000-0000-0000-000000000011'),
  '(b) fn_portal_permits(tavern, cellar) = false (cellar hatch locked)'
);

-- (c) ActorMoved tavern→cellar via apply_event → gate_reject AND nothing written to canon (blocker-only).
SELECT ok(
  ((apply_event(
      'f5000000-ffff-0000-0000-000000000000',
      'f5000000-0000-0000-0000-000000000001',
      '{"type":"ActorMoved","stated":"P tries the cellar hatch","to_target_id":"f5000000-0000-0000-0000-000000000011"}'::jsonb,
      2001, 0, 'freeform'
   ))->>'halt_reason' = 'gate_reject')
  AND (SELECT count(*)::int FROM canon_event
       WHERE world_id='f5000000-ffff-0000-0000-000000000000' AND in_world_tick=2001) = 0,
  '(c) ActorMoved tavern→cellar → gate_reject, nothing written (the locked door holds)'
);

-- (d) ActorMoved tavern→dock_street via apply_event → committed (the open front door permits passage).
SELECT is(
  (apply_event(
      'f5000000-ffff-0000-0000-000000000000',
      'f5000000-0000-0000-0000-000000000001',
      '{"type":"ActorMoved","stated":"P steps out the front door","to_target_id":"f5000000-0000-0000-0000-000000000012"}'::jsonb,
      2002, 0, 'freeform'
   ))->>'halt_reason',
  'committed',
  '(d) ActorMoved tavern→dock_street → committed (front door open ∧ ¬locked)'
);

SELECT * FROM finish();
ROLLBACK;
