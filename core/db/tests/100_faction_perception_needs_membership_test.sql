-- SPEC-051 item 8 / docs/design/2026-09-02-concepts-as-knowledge.md §7: fn_visible_perceptions
-- used to return any perception held by a 'faction'/'group' entity to EVERY viewer, with no
-- membership condition -- broadcast, not shared knowledge. This locks the narrowed WHERE: a
-- non-member never sees faction-held knowledge, and the holder itself still sees its own.
BEGIN;
SELECT plan(2);

INSERT INTO world (world_id, display_name) VALUES
  ('00000000-0000-0000-0000-0000000000f1', 'Faction Leak Fixture');

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-0000000000f1', 'actor',   'Non-member'),
  ('00000000-0000-0000-0000-0000000000fa', '00000000-0000-0000-0000-0000000000f1', 'faction', 'The Guild');

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
  ('00000000-0000-0000-0000-0000000000ce', '00000000-0000-0000-0000-0000000000f1',
   'EntityCreated', 'the Guild learns something', 1, 0, 'accepted', now(), 'private', 'fast_path');

INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
  ('00000000-0000-0000-0000-0000000000cf', '00000000-0000-0000-0000-0000000000f1',
   '00000000-0000-0000-0000-0000000000fa', '00000000-0000-0000-0000-0000000000ce',
   'the Guild''s secret', 'direct', 1, 1);

-- GATE-CRITICAL NEGATIVE: a non-member of the faction sees nothing the faction holds.
SELECT is_empty(
  $$ SELECT perception_id FROM fn_visible_perceptions(
       '00000000-0000-0000-0000-0000000000f1'::uuid,
       '00000000-0000-0000-0000-0000000000a1'::uuid) $$,
  'a perception held by a faction is NOT visible to a non-member'
);

-- the holder itself still sees its own perception (the narrowing must not break plain holder
-- visibility -- a mutation that broke ALL visibility would otherwise pass the assertion above).
SELECT ok(
  (SELECT count(*) FROM fn_visible_perceptions(
     '00000000-0000-0000-0000-0000000000f1'::uuid,
     '00000000-0000-0000-0000-0000000000fa'::uuid)
   WHERE perception_id = '00000000-0000-0000-0000-0000000000cf') = 1,
  'the faction itself still sees its own perception'
);

SELECT * FROM finish();
ROLLBACK;
