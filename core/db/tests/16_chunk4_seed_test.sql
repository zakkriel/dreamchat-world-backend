BEGIN;
SELECT plan(8);
-- the Sealed Note artifact exists, is an artifact, created mid-timeline by the discovery event
SELECT is( (SELECT entity_kind FROM entity_registry
            WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'artifact', 'Sealed Note is an artifact');
SELECT is( (SELECT created_by_event FROM entity_registry
            WHERE entity_id='a4000000-0000-0000-0000-0000000000a1'),
           'e0000000-0000-0000-0000-0000000000d1'::uuid,
           'note created_by_event = discovery event (introduced mid-timeline)');
-- discovery event accepted at (100,1): unique vs E1 (100,0) under uq_ce_accepted_order
SELECT is( (SELECT in_world_tick::text||','||beat_seq::text FROM canon_event
            WHERE event_id='e0000000-0000-0000-0000-0000000000d1' AND status='accepted'),
           '100,1', 'discovery event accepted at (100,1)');
-- two Player 'direct' perceptions sourced to the discovery event
SELECT is( (SELECT count(*) FROM perception_record
            WHERE source_event_id='e0000000-0000-0000-0000-0000000000d1'
              AND holder_id='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
              AND epistemic_type='direct')::int,
           2, 'two Player direct perceptions from the discovery event');
-- note perception subject = the Note ALONE (subject ≠ participants {Player,Note,Tavern}; ADR-035)
SELECT is( (SELECT count(*) FROM perception_subject
            WHERE perception_id='dca70000-0000-0000-0000-000000000b01')::int,
           1, 'note perception has exactly one subject');
SELECT is( (SELECT entity_id FROM perception_subject
            WHERE perception_id='dca70000-0000-0000-0000-000000000b01'),
           'a4000000-0000-0000-0000-0000000000a1'::uuid,
           'note perception subject is the Note alone');
-- tavern perception subject = the Tavern ALONE (guards the other half of the backfill skip)
SELECT is( (SELECT count(*) FROM perception_subject
            WHERE perception_id='dca70000-0000-0000-0000-000000000c01')::int,
           1, 'tavern perception has exactly one subject');
SELECT is( (SELECT entity_id FROM perception_subject
            WHERE perception_id='dca70000-0000-0000-0000-000000000c01'),
           'dddddddd-dddd-dddd-dddd-dddddddddddd'::uuid,
           'tavern perception subject is the Tavern alone');
SELECT * FROM finish();
ROLLBACK;
