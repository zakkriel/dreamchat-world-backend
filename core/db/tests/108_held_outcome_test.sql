BEGIN;
SELECT plan(4);

-- Task 5 — held_outcome + the telegraph origin (RULINGS-2026-07-24 §1, §3).
-- A cognition seat that TELEGRAPHS a disruptive act commits its wind-up as perceivable canon
-- (origin='telegraph') and writes a dedicated held_outcome row carrying her full intended act.
-- The hold fires on the PLAYER'S NEXT INPUT, never the clock — so it is NOT a pending_event row.
-- Fresh world (no seed dependency); all rows are rolled back at the end.
--   world: 10800000-ffff-0000-0000-000000000000
--   npc  : 10800000-0000-0000-0000-000000000001  (Jonas — the act is hers/his)
--   ev   : 10800000-0000-0000-0000-000000000002  (the committed wind-up)
--   held : 10800000-0000-0000-0000-000000000003

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('10800000-0000-0000-0000-000000000001','10800000-ffff-0000-0000-000000000000','actor','test-npc-108');

-- (c) origin='telegraph' is accepted on canon_event (the append-only CHECK extension).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('10800000-0000-0000-0000-000000000002','10800000-ffff-0000-0000-000000000000',
        'AttributeChanged','Jonas pushes off the bar, moving to cut in',5000,0,
        'accepted',now(),'public','telegraph');
SELECT is(
  (SELECT origin FROM canon_event WHERE event_id='10800000-0000-0000-0000-000000000002'),
  'telegraph', 'canon_event.origin accepts telegraph (append-only CHECK extension)');

-- (a) a held row referencing the committed wind-up is readable; status DEFAULTS to pending.
INSERT INTO held_outcome (held_id, world_id, actor_id, attempt, telegraph_event_id, created_tick)
VALUES ('10800000-0000-0000-0000-000000000003','10800000-ffff-0000-0000-000000000000',
        '10800000-0000-0000-0000-000000000001',
        '{"type":"ActorMoved","stated":"Jonas pushes off the bar, moving to cut in","to_target_id":"10800000-0000-0000-0000-000000000009"}'::jsonb,
        '10800000-0000-0000-0000-000000000002', 5000);
SELECT is(
  (SELECT status FROM held_outcome WHERE held_id='10800000-0000-0000-0000-000000000003'),
  'pending', 'held_outcome row is readable and defaults to status=pending');

-- (b) the status CHECK rejects anything outside {pending,resolved} — e.g. 'expired'. A held
--     outcome has NO clock-driven expiry: the player is the clock, so there is no timeout (§3).
SELECT throws_ok(
  $$ INSERT INTO held_outcome (world_id, actor_id, attempt, telegraph_event_id, status, created_tick)
     VALUES ('10800000-ffff-0000-0000-000000000000','10800000-0000-0000-0000-000000000001',
             '{}'::jsonb,'10800000-0000-0000-0000-000000000002','expired',5000) $$,
  '23514', NULL,
  'held_outcome.status CHECK rejects expired (no clock-driven expiry — §3)');

-- (d) held_outcome rows are LOOP STATE, not canon — deletion is allowed, so there is no
--     delete-guard to assert. Instead assert the partial pending index exists: the reaction
--     beat's next-input lookup scans WHERE status='pending' on this index.
SELECT has_index('held_outcome', 'idx_held_outcome_pending',
                 'partial pending index on held_outcome exists');

SELECT * FROM finish();
ROLLBACK;
