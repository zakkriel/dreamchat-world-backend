BEGIN;
SELECT plan(4);

-- Pre-position Jonas at cellar-uuid (a seed-clean location uuid). Then a MOVE of Player → cellar-uuid
-- (with its location mutation applied). The cellar uuid has no seed occupant, so discovery at the
-- destination is exactly {Jonas} regardless of the seed's noise positions.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
VALUES ('e5ffffff-0000-0000-0000-000000000002','11111111-1111-1111-1111-111111111111','location','test-cellar-94');

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('e5000000-0000-0000-0000-000000000030','11111111-1111-1111-1111-111111111111','move','setup J→cellar',310,0,'accepted',now(),'public','fast_path'),
 ('e5000000-0000-0000-0000-000000000031','11111111-1111-1111-1111-111111111111','move','P moves to cellar',311,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('e5000000-0000-0000-0000-000000000030','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','instigator'),
 ('e5000000-0000-0000-0000-000000000031','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq) VALUES
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000030','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000002'::text),310,0),
 ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000031','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('e5ffffff-0000-0000-0000-000000000002'::text),311,0);

-- mover 'direct' (witnessed own move) + discovery 'direct' about Jonas already present = 2.
SELECT is(generate_perceptions('e5000000-0000-0000-0000-000000000031')::int, 2,
          'MOVE generates the mover own-move + one discovery (Jonas present)');
SELECT is((SELECT count(*) FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000031'
             AND holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
             AND epistemic_type='direct')::int,
          2, 'both perceptions are the mover (Player), direct');
-- the discovery perception is ABOUT Jonas (perception_subject) → feeds the stop-check
SELECT ok(EXISTS(
  SELECT 1 FROM perception_record pr
  JOIN perception_subject ps ON ps.perception_id = pr.perception_id
  WHERE pr.source_event_id='e5000000-0000-0000-0000-000000000031'
    AND ps.entity_id='cccccccc-cccc-cccc-cccc-cccccccccccc'),
  'discovery perception is ABOUT Jonas (subject link)');
-- Jonas, already present, is NOT handed a perception of Player arriving in the thin slice
-- (witnessing-others defers with the broader vocabulary; the mover-discovery axis is what the
-- stop-check needs). Keep the slice minimal and honest.
SELECT is((SELECT count(*) FROM perception_record
           WHERE source_event_id='e5000000-0000-0000-0000-000000000031'
             AND holder_id='cccccccc-cccc-cccc-cccc-cccccccccccc')::int,
          0, 'thin slice: only the mover perceives (others-witness-mover deferred)');

SELECT * FROM finish();
ROLLBACK;
