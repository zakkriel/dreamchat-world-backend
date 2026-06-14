BEGIN;
SELECT plan(6);
-- positive: every ORIGINAL event-derived perception got subjects from its source event's participants
SELECT is(
  (SELECT count(*) FROM perception_record pr
   WHERE NOT EXISTS (SELECT 1 FROM perception_subject ps WHERE ps.perception_id = pr.perception_id))::int,
  0, 'every perception has at least one subject (backfill + explicit)');
-- positive: principal-cast name perceptions exist, CK-held, genesis-sourced, subject = the named entity
SELECT is(
  (SELECT count(*) FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.event_type='world_genesis'
   JOIN perception_subject ps ON ps.perception_id = pr.perception_id
   WHERE pr.holder_id='eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee' AND pr.epistemic_type='public')::int,
  5, '5 common-knowledge name perceptions (Player,Mara,Jonas,Tavern,Square)');
-- negative: noise actor O1 has NO common-knowledge name perception (withhold path is real)
SELECT is(
  (SELECT count(*) FROM perception_record pr
   JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.event_type='world_genesis'
   JOIN perception_subject ps ON ps.perception_id = pr.perception_id
   WHERE ps.entity_id='00000000-0000-0000-0000-000000000001')::int,
  0, 'O1 deliberately unnamed (no genesis name perception)');
-- the gate fixture: Player-private-about-Mara perception exists with fixed id, direct, subject = Mara only
SELECT is(
  (SELECT epistemic_type FROM perception_record WHERE perception_id='dca70000-0000-0000-0000-000000000a01'),
  'direct', 'about-Mara fixture is a direct perception');
SELECT is(
  (SELECT count(*) FROM perception_subject WHERE perception_id='dca70000-0000-0000-0000-000000000a01')::int,
  1, 'about-Mara fixture has exactly one subject');
SELECT is(
  (SELECT entity_id FROM perception_subject WHERE perception_id='dca70000-0000-0000-0000-000000000a01'),
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'::uuid, 'about-Mara fixture subject is Mara only (not Player)');
SELECT * FROM finish();
ROLLBACK;
