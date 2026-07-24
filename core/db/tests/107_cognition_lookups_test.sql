BEGIN;
SELECT plan(9);

-- The mechanical cognition lookups (RULINGS-2026-07-23 §5-§6): public moment (modal face),
-- isolation split, private records. Fresh, self-contained fixture in its own world (d7070000-
-- prefixed UUIDs) so the lookups see ONLY these rows — no seed perception ever leaks in. ROLLBACK
-- at end. Definition under test (plan flagged-decision 3): a perception RECORD is PUBLIC iff its
-- (source_event_id, content) equals the modal content among ALL present holders of that source;
-- everything else is PRIVATE (source not held by all, OR content differs from the modal, OR NULL
-- source). The split is per-record, keyed on (source, content) — which is exactly why E_variant's
-- modal face is public while J's divergent read of the SAME event is his private record (see g).
--   world W:               d7070000-ffff-0000-0000-000000000000
--   actor P (player):      d7070000-0000-0000-0000-000000000001
--   actor M (Mara):        d7070000-0000-0000-0000-000000000002
--   actor J (Jonas):       d7070000-0000-0000-0000-000000000003
--   entity X (fog figure): d7070000-0000-0000-0000-000000000004   (E_variant's subject)
--   E_pub    (all 3, identical):  d7070000-0000-0000-0000-0000000000e1  tick 700
--   E_secret (M only, about P):   d7070000-0000-0000-0000-0000000000e2  tick 701
--   E_variant(all 3, J diverges): d7070000-0000-0000-0000-0000000000e3  tick 702

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('d7070000-0000-0000-0000-000000000001','d7070000-ffff-0000-0000-000000000000','actor','test-P-107'),
 ('d7070000-0000-0000-0000-000000000002','d7070000-ffff-0000-0000-000000000000','actor','test-M-107'),
 ('d7070000-0000-0000-0000-000000000003','d7070000-ffff-0000-0000-000000000000','actor','test-J-107'),
 ('d7070000-0000-0000-0000-000000000004','d7070000-ffff-0000-0000-000000000000','actor','test-X-107');

-- Source events (perception_record.source_event_id is FK -> canon_event).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('d7070000-0000-0000-0000-0000000000e1','d7070000-ffff-0000-0000-000000000000','observation','the public moment',       700,0,'accepted',now(),'public','fast_path'),
 ('d7070000-0000-0000-0000-0000000000e2','d7070000-ffff-0000-0000-000000000000','observation','the secret M alone saw', 701,0,'accepted',now(),'private','fast_path'),
 ('d7070000-0000-0000-0000-0000000000e3','d7070000-ffff-0000-0000-000000000000','observation','the moment J read sharp',702,0,'accepted',now(),'public','fast_path');

-- E_pub: every present id (P, M, J) holds an IDENTICAL perception -> a public moment.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick) VALUES
 ('d7070000-0000-0001-0000-0000000000e1','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-0000000000e1','a torch flares in the doorway','direct',700,700),
 ('d7070000-0000-0002-0000-0000000000e1','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-0000000000e1','a torch flares in the doorway','direct',700,700),
 ('d7070000-0000-0003-0000-0000000000e1','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000003','d7070000-0000-0000-0000-0000000000e1','a torch flares in the doorway','direct',700,700);

-- E_secret: held ONLY by M, subject-linked to P. Private (not perceived by all present).
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick) VALUES
 ('d7070000-0000-0002-0000-0000000000e2','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-0000000000e2','the ledger names the smuggler','direct',701,701);

-- E_variant: all three perceive it, but J's content DIFFERS (his sharper read). The event's subject
-- is X (the fog figure), NOT P. Modal content among present holders = 'a figure passes in the fog'.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick) VALUES
 ('d7070000-0000-0001-0000-0000000000e3','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-0000000000e3','a figure passes in the fog','direct',702,702),
 ('d7070000-0000-0002-0000-0000000000e3','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-0000000000e3','a figure passes in the fog','direct',702,702),
 ('d7070000-0000-0003-0000-0000000000e3','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000003','d7070000-0000-0000-0000-0000000000e3','a hooded smuggler slips past','direct',702,702);

-- About-ness links (RULINGS §6). E_pub -> P; E_secret -> P (M's secret is about the player);
-- E_variant -> X for all three holders (the moment is about the fog figure, not P).
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('d7070000-0000-0001-0000-0000000000e1','d7070000-0000-0000-0000-000000000001','d7070000-ffff-0000-0000-000000000000'),
 ('d7070000-0000-0002-0000-0000000000e1','d7070000-0000-0000-0000-000000000001','d7070000-ffff-0000-0000-000000000000'),
 ('d7070000-0000-0003-0000-0000000000e1','d7070000-0000-0000-0000-000000000001','d7070000-ffff-0000-0000-000000000000'),
 ('d7070000-0000-0002-0000-0000000000e2','d7070000-0000-0000-0000-000000000001','d7070000-ffff-0000-0000-000000000000'),
 ('d7070000-0000-0001-0000-0000000000e3','d7070000-0000-0000-0000-000000000004','d7070000-ffff-0000-0000-000000000000'),
 ('d7070000-0000-0002-0000-0000000000e3','d7070000-0000-0000-0000-000000000004','d7070000-ffff-0000-0000-000000000000'),
 ('d7070000-0000-0003-0000-0000000000e3','d7070000-0000-0000-0000-000000000004','d7070000-ffff-0000-0000-000000000000');

-- (a) public moment contains E_pub (every present holder perceives it identically).
SELECT ok(
  EXISTS (SELECT 1 FROM fn_public_moment(
            'd7070000-ffff-0000-0000-000000000000',
            ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[],
            10)
          WHERE source_event_id = 'd7070000-0000-0000-0000-0000000000e1'),
  '(a) public moment contains E_pub (all three perceive it identically)');

-- (b) E_variant is rendered with the MAJORITY content, not J's sharper read.
SELECT is(
  (SELECT content FROM fn_public_moment(
            'd7070000-ffff-0000-0000-000000000000',
            ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[],
            10)
          WHERE source_event_id = 'd7070000-0000-0000-0000-0000000000e3'),
  'a figure passes in the fog',
  '(b) E_variant rendered with the modal content (majority), not J''s divergent read');

-- (c) public moment does NOT contain E_secret (only M holds it -> not shared).
SELECT ok(
  NOT EXISTS (SELECT 1 FROM fn_public_moment(
            'd7070000-ffff-0000-0000-000000000000',
            ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[],
            10)
          WHERE source_event_id = 'd7070000-0000-0000-0000-0000000000e2'),
  '(c) public moment excludes E_secret (not perceived by every present id)');

-- (d) isolation split: with the action bound to P, only M is pulled out (her secret is about P).
SELECT set_eq(
  $$ SELECT actor_id FROM fn_isolated_npcs(
       'd7070000-ffff-0000-0000-000000000000',
       ARRAY['d7070000-0000-0000-0000-000000000001']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[]) $$,
  $$ VALUES ('d7070000-0000-0000-0000-000000000002'::uuid) $$,
  '(d) only M is isolated for an action about P (her private secret is subject-linked to P)');

-- (e) action bound to J: nobody holds a private record about J -> nobody isolated. M stays in the
-- batch (dull-never-leak): the modal E_variant face is public, and her secret is about P, not J.
SELECT is(
  (SELECT count(*) FROM fn_isolated_npcs(
       'd7070000-ffff-0000-0000-000000000000',
       ARRAY['d7070000-0000-0000-0000-000000000003']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[]))::int,
  0,
  '(e) action about J isolates nobody — M stays in the shared batch');

-- (f) M's private records about P = exactly her secret (E_pub is public; E_variant is public for her).
SELECT set_eq(
  $$ SELECT content FROM fn_private_records(
       'd7070000-ffff-0000-0000-000000000000',
       'd7070000-0000-0000-0000-000000000002',
       ARRAY['d7070000-0000-0000-0000-000000000001']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[]) $$,
  $$ VALUES ('the ledger names the smuggler') $$,
  '(f) fn_private_records(M, about P) returns exactly the secret''s content');

-- (g) J has NO private record about P ({} for p_action_ids={P}), yet his divergent read of E_variant
-- IS his private record and surfaces the moment the action binds E_variant's subject (X). Per-record
-- split, not per-event: the public E_variant face and J's private variant coexist without contradiction.
SELECT ok(
  NOT EXISTS (SELECT 1 FROM fn_private_records(
       'd7070000-ffff-0000-0000-000000000000',
       'd7070000-0000-0000-0000-000000000003',
       ARRAY['d7070000-0000-0000-0000-000000000001']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[]))
  AND EXISTS (SELECT 1 FROM fn_private_records(
       'd7070000-ffff-0000-0000-000000000000',
       'd7070000-0000-0000-0000-000000000003',
       ARRAY['d7070000-0000-0000-0000-000000000004']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[])
     WHERE content = 'a hooded smuggler slips past'),
  '(g) J holds no private record about P, but his sharper E_variant read IS his private record about X');

-- ---------------------------------------------------------------------------------------------
-- (h) Temporal-validity leak scenario (fix pass, review finding "Important"). Present {P, M, J}
-- again, on a NEW source event E_leak. All three originally read the same thing at tick 703
-- ("a hooded smuggler slips through the alley"), but P and M's readings were later corrected: those
-- rows get invalid_tick set (superseded by a current "fog" read at tick 706). J's original read was
-- NEVER invalidated -- it stands as his genuine, still-current, divergent private read. If the
-- lookups counted the stale invalidated copies, P+M's old "smuggler" votes (2) would outnumber their
-- own current "fog" votes... no wait: it's P-smuggler(stale)+M-smuggler(stale)+J-smuggler(current) = 3
-- "smuggler" votes vs P-fog(current)+M-fog(current) = 2 "fog" votes, so the modal face would flip to
-- "smuggler" -- exactly J's private content leaking into the shared public moment. Entity Y is the
-- action id J's smuggler perception is subject-linked to.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('d7070000-0000-0000-0000-000000000005','d7070000-ffff-0000-0000-000000000000','actor','test-Y-107-action');

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('d7070000-0000-0000-0000-0000000000e4','d7070000-ffff-0000-0000-000000000000','observation','stale-vs-current divergence check',703,0,'accepted',now(),'private','fast_path');

-- P: stale invalidated "smuggler" read (must not vote), then a current "fog" read.
-- M: same pattern as P.
-- J: current, NEVER-invalidated "smuggler" read -- his genuine divergent private read.
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick, invalid_tick) VALUES
 ('d7070000-0000-0011-0000-0000000000e4','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-0000000000e4','a hooded smuggler slips through the alley','direct',703,703,705),
 ('d7070000-0000-0001-0000-0000000000e4','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-0000000000e4','a shape drifts through the fog','direct',706,706,NULL),
 ('d7070000-0000-0012-0000-0000000000e4','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-0000000000e4','a hooded smuggler slips through the alley','direct',703,703,705),
 ('d7070000-0000-0002-0000-0000000000e4','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-0000000000e4','a shape drifts through the fog','direct',706,706,NULL),
 ('d7070000-0000-0003-0000-0000000000e4','d7070000-ffff-0000-0000-000000000000','d7070000-0000-0000-0000-000000000003','d7070000-0000-0000-0000-0000000000e4','a hooded smuggler slips through the alley','direct',703,703,NULL);

-- Only J's (current) smuggler read is subject-linked to the action id (Y) under test.
INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES
 ('d7070000-0000-0003-0000-0000000000e4','d7070000-0000-0000-0000-000000000005','d7070000-ffff-0000-0000-000000000000');

SELECT ok(
  (SELECT content FROM fn_public_moment(
            'd7070000-ffff-0000-0000-000000000000',
            ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[],
            10)
          WHERE source_event_id = 'd7070000-0000-0000-0000-0000000000e4') = 'a shape drifts through the fog'
  AND EXISTS (SELECT 1 FROM fn_isolated_npcs(
       'd7070000-ffff-0000-0000-000000000000',
       ARRAY['d7070000-0000-0000-0000-000000000005']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000002','d7070000-0000-0000-0000-000000000003']::uuid[])
       WHERE actor_id = 'd7070000-0000-0000-0000-000000000003'),
  '(h) stale invalidated smuggler reads do NOT vote (fog wins the modal face) AND J''s still-current divergent smuggler read stays isolated');

-- ---------------------------------------------------------------------------------------------
-- (i) Freshest-20 cap (fix pass, review finding "Minor"). NPC K holds 21 private records that all
-- intersect action id Z, at 21 distinct ticks (800..820). Present = {P, K}: P never holds any of
-- these source events, so none of them ever clears the "held by all present" bar -- all 21 stay
-- private. The cap must keep the FRESHEST 20 (ticks 801..820), drop the OLDEST (800), and present
-- them ascending -- so row 1's tick is 801, the second-oldest tick overall.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 ('d7070000-0000-0000-0000-000000000006','d7070000-ffff-0000-0000-000000000000','actor','test-K-107-npc'),
 ('d7070000-0000-0000-0000-000000000007','d7070000-ffff-0000-0000-000000000000','actor','test-Z-107-action');

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
SELECT ('d7070000-0000-0000-0000-' || lpad((1000+n)::text,12,'0'))::uuid,
       'd7070000-ffff-0000-0000-000000000000',
       'observation',
       'freshest-20 cap fixture, tick ' || (800+n)::text,
       800+n,
       0,
       'accepted',
       now(),
       'private',
       'fast_path'
FROM generate_series(0,20) AS n;

INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
SELECT ('d7070000-0000-0006-0000-' || lpad((1000+n)::text,12,'0'))::uuid,
       'd7070000-ffff-0000-0000-000000000000',
       'd7070000-0000-0000-0000-000000000006',
       ('d7070000-0000-0000-0000-' || lpad((1000+n)::text,12,'0'))::uuid,
       'K senses the shape at tick ' || (800+n)::text,
       'direct',
       800+n,
       800+n
FROM generate_series(0,20) AS n;

INSERT INTO perception_subject (perception_id, entity_id, world_id)
SELECT ('d7070000-0000-0006-0000-' || lpad((1000+n)::text,12,'0'))::uuid,
       'd7070000-0000-0000-0000-000000000007',
       'd7070000-ffff-0000-0000-000000000000'
FROM generate_series(0,20) AS n;

SELECT results_eq(
  $$ SELECT content, acquired_tick FROM fn_private_records(
       'd7070000-ffff-0000-0000-000000000000',
       'd7070000-0000-0000-0000-000000000006',
       ARRAY['d7070000-0000-0000-0000-000000000007']::uuid[],
       ARRAY['d7070000-0000-0000-0000-000000000001','d7070000-0000-0000-0000-000000000006']::uuid[]) $$,
  $$ SELECT 'K senses the shape at tick ' || t::text, t::bigint
     FROM generate_series(801,820) AS t
     ORDER BY t ASC $$,
  '(i) freshest-20 cap: oldest tick (800) excluded, remaining 20 presented ascending starting at the second-oldest tick (801)');

SELECT * FROM finish();
ROLLBACK;
