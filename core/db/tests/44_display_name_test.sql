BEGIN;
SELECT plan(8);
-- §3 naming reach (RULINGS-2026-07-23): fn_display_name is the per-viewer LOOKUP — known-name (the
-- viewer's own name-knowledge) else descriptor else canonical — and fn_batch_display_name is its
-- shared-by-all intersection for the batch seat (a name only if EVERY batch mind resolves the SAME
-- known name, else descriptor/canonical). Fresh self-contained world (d44 prefix), ROLLBACK at end.
--   world W:  d4400000-ffff-0000-0000-000000000000
--   V1 (knows E by name), V2 (does not), V3 (knows E by name):  ...0001 / ...0002 / ...0003
--   E (named 'Reyna', descriptor 'the stranger'):               ...000e
--   F (no name, no descriptor → canonical fallback 'Fog Figure'):...000f

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('d4400000-0000-0000-0000-000000000001','d4400000-ffff-0000-0000-000000000000','actor','V1-canon'),
 ('d4400000-0000-0000-0000-000000000002','d4400000-ffff-0000-0000-000000000000','actor','V2-canon'),
 ('d4400000-0000-0000-0000-000000000003','d4400000-ffff-0000-0000-000000000000','actor','V3-canon'),
 ('d4400000-0000-0000-0000-00000000000e','d4400000-ffff-0000-0000-000000000000','actor','E-canon'),
 ('d4400000-0000-0000-0000-00000000000f','d4400000-ffff-0000-0000-000000000000','actor','Fog Figure');

-- E carries a Tier-2 descriptor in actor_state (F carries none → canonical fallback).
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
 ('d4400000-0000-0000-0000-00000000000e','d4400000-ffff-0000-0000-000000000000','{"descriptor":"the stranger"}'::jsonb);

-- A world_genesis event sources the name-knowledge (fn_perceived_name keys on it).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('d4400000-0000-0000-0000-0000000000e0','d4400000-ffff-0000-0000-000000000000','world_genesis','names',44,0,'accepted',now(),'public','fast_path');

-- V1 and V3 each privately hold E's name ('Reyna'); V2 holds nothing. Subject-linked to E.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick) VALUES
 ('d4400000-0001-0000-0000-0000000000e1','d4400000-ffff-0000-0000-000000000000','d4400000-0000-0000-0000-000000000001','d4400000-0000-0000-0000-0000000000e0','Reyna','told',44,44),
 ('d4400000-0003-0000-0000-0000000000e1','d4400000-ffff-0000-0000-000000000000','d4400000-0000-0000-0000-000000000003','d4400000-0000-0000-0000-0000000000e0','Reyna','told',44,44);
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('d4400000-0001-0000-0000-0000000000e1','d4400000-0000-0000-0000-00000000000e','d4400000-ffff-0000-0000-000000000000'),
 ('d4400000-0003-0000-0000-0000000000e1','d4400000-0000-0000-0000-00000000000e','d4400000-ffff-0000-0000-000000000000');

-- (a) known name: V1 resolves E as 'Reyna'.
SELECT is( fn_display_name('d4400000-ffff-0000-0000-000000000000','d4400000-0000-0000-0000-000000000001','d4400000-0000-0000-0000-00000000000e'),
           'Reyna', '(a) fn_display_name(V1, E) = the known name');
-- (b) descriptor fallback: V2 has no name-knowledge of E → the descriptor.
SELECT is( fn_display_name('d4400000-ffff-0000-0000-000000000000','d4400000-0000-0000-0000-000000000002','d4400000-0000-0000-0000-00000000000e'),
           'the stranger', '(b) fn_display_name(V2, E) = the descriptor (no name known)');
-- (c) canonical fallback: F has no name and no descriptor → canonical.
SELECT is( fn_display_name('d4400000-ffff-0000-0000-000000000000','d4400000-0000-0000-0000-000000000002','d4400000-0000-0000-0000-00000000000f'),
           'Fog Figure', '(c) fn_display_name(V2, F) = canonical (no name, no descriptor)');

-- (d) BATCH intersection — mixed: {V1 knows, V2 does not} → NOT shared-by-all → the descriptor.
SELECT is( fn_batch_display_name('d4400000-ffff-0000-0000-000000000000',
             ARRAY['d4400000-0000-0000-0000-000000000001','d4400000-0000-0000-0000-000000000002']::uuid[],
             'd4400000-0000-0000-0000-00000000000e'),
           'the stranger', '(d) batch {V1,V2} → descriptor (name not shared by all)');
-- (e) BATCH intersection — agreement: {V1, V3} both know 'Reyna' → the shared name.
SELECT is( fn_batch_display_name('d4400000-ffff-0000-0000-000000000000',
             ARRAY['d4400000-0000-0000-0000-000000000001','d4400000-0000-0000-0000-000000000003']::uuid[],
             'd4400000-0000-0000-0000-00000000000e'),
           'Reyna', '(e) batch {V1,V3} → the shared name (every mind knows the same one)');
-- (f) BATCH single mind who knows → the name.
SELECT is( fn_batch_display_name('d4400000-ffff-0000-0000-000000000000',
             ARRAY['d4400000-0000-0000-0000-000000000001']::uuid[],
             'd4400000-0000-0000-0000-00000000000e'),
           'Reyna', '(f) batch {V1} → the name');
-- (g) BATCH single mind who does NOT know → the descriptor.
SELECT is( fn_batch_display_name('d4400000-ffff-0000-0000-000000000000',
             ARRAY['d4400000-0000-0000-0000-000000000002']::uuid[],
             'd4400000-0000-0000-0000-00000000000e'),
           'the stranger', '(g) batch {V2} → the descriptor');
-- (h) BATCH empty mind set → descriptor (nothing to intersect, never a name).
SELECT is( fn_batch_display_name('d4400000-ffff-0000-0000-000000000000',
             ARRAY[]::uuid[],
             'd4400000-0000-0000-0000-00000000000e'),
           'the stranger', '(h) batch {} → descriptor (no minds to share a name)');

SELECT * FROM finish();
ROLLBACK;
