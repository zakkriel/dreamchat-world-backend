-- fn_collected_knowledge — Collected Knowledge is grouped by SUBJECT (2026-08-09), replacing the
-- event-keyed grouping SPEC-029 (#40) shipped.
--
-- The fixture: 25 accepted events inside ONE authored moment labelled "Arrival" (exactly what
-- trg_canon_event_carry_in_world_label produces on a played world), each yielding one perception the
-- Player holds about Mara. Five of them also mention the Sealed Note, four also mention the Tavern,
-- and one of the note records mentions BOTH — the multi-topic case. Under the old grouping this
-- rendered 25 groups every one of which was headed "Arrival"; that is the defect these tests stand
-- against, so the fixture reproduces it rather than approximating it.
--
--   world  11111111-…   Player aaaaaaaa-…   Mara bbbbbbbb-…   Jonas cccccccc-…
--          Sealed Note a4000000-…a1         Tavern dddddddd-…
BEGIN;
SELECT plan(10);

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
SELECT ('e0000000-0000-0000-0000-0000009' || lpad(g::text, 5, '0'))::uuid,
       '11111111-1111-1111-1111-111111111111', 'observation',
       'beat ' || g, 900 + g, 0,
       'Arrival', 'accepted', now(), 'private', 'fast_path'
FROM generate_series(1, 25) g;

INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick)
SELECT ('dca70000-0000-0000-0000-0000009' || lpad(g::text, 5, '0'))::uuid,
       '11111111-1111-1111-1111-111111111111',
       'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
       ('e0000000-0000-0000-0000-0000009' || lpad(g::text, 5, '0'))::uuid,
       'observation ' || g, 'direct', 900 + g, 900 + g
FROM generate_series(1, 25) g;

INSERT INTO perception_subject (perception_id, entity_id, world_id)
SELECT ('dca70000-0000-0000-0000-0000009' || lpad(g::text, 5, '0'))::uuid,
       'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111'
FROM generate_series(1, 25) g;

INSERT INTO perception_subject (perception_id, entity_id, world_id)
SELECT ('dca70000-0000-0000-0000-0000009' || lpad(g::text, 5, '0'))::uuid,
       'a4000000-0000-0000-0000-0000000000a1', '11111111-1111-1111-1111-111111111111'
FROM generate_series(1, 5) g;

INSERT INTO perception_subject (perception_id, entity_id, world_id)
SELECT ('dca70000-0000-0000-0000-0000009' || lpad(g::text, 5, '0'))::uuid,
       'dddddddd-dddd-dddd-dddd-dddddddddddd', '11111111-1111-1111-1111-111111111111'
FROM generate_series(6, 9) g;

-- the multi-topic record: note AND tavern
INSERT INTO perception_subject (perception_id, entity_id, world_id)
VALUES ('dca70000-0000-0000-0000-000000900003', 'dddddddd-dddd-dddd-dddd-dddddddddddd',
        '11111111-1111-1111-1111-111111111111');

-- the reader is a co-subject of one record; they must never become a topic
INSERT INTO perception_subject (perception_id, entity_id, world_id)
VALUES ('dca70000-0000-0000-0000-000000900011', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        '11111111-1111-1111-1111-111111111111');

CREATE TEMP VIEW mara_groups AS
SELECT g, ord
FROM json_array_elements(
       fn_actor_page('11111111-1111-1111-1111-111111111111',
                     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                     'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups')
     WITH ORDINALITY AS t(g, ord);

-- ── the reported defect ─────────────────────────────────────────────────────────────────────────
-- 25 records inside one moment produced 25 groups, all headed "Arrival". The number that matters is
-- "far fewer groups than records", not an exact count — pinning the count would make this test a
-- fixture checksum instead of a defence.
SELECT cmp_ok((SELECT count(*) FROM mara_groups)::int, '<', 10,
  '25 records in one authored moment no longer render as 25 groups');

SELECT is((SELECT count(*) FROM mara_groups WHERE g->>'group_label' = 'Arrival')::int, 0,
  'no group is headed with a moment label — in_world_label is time, not a topic');

SELECT is((SELECT count(*) FROM mara_groups WHERE g->>'group_key' LIKE 'event:%')::int, 0,
  'no group is keyed by a source event — event keying is log order wearing a heading');

-- ── grouping by about-ness ──────────────────────────────────────────────────────────────────────
SELECT ok( EXISTS (SELECT 1 FROM mara_groups
                    WHERE g->>'group_key' = 'subject:a4000000-0000-0000-0000-0000000000a1'
                      AND g->>'group_label' = 'Sealed Note'),
  'a recurring co-subject becomes a topic, labelled with the viewer''s own name for it');

SELECT ok( EXISTS (SELECT 1 FROM mara_groups
                    WHERE g->>'group_key' = 'subject:dddddddd-dddd-dddd-dddd-dddddddddddd'
                      AND g->>'group_label' = 'Tavern'),
  'a second recurring co-subject is a second topic');

-- ── the reader is never a topic ─────────────────────────────────────────────────────────────────
-- The viewer co-subjects nearly every record they hold (they were there), so without excluding them
-- "Player" would be a spurious mega-topic on every page in the world.
SELECT is((SELECT count(*) FROM mara_groups
            WHERE g->>'group_key' = 'subject:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')::int, 0,
  'the viewer is never a topic of the dossier they are reading');

-- ── the remainder ───────────────────────────────────────────────────────────────────────────────
-- Records about the page's own subject and nothing else nameable. Unheaded, and FIRST — a
-- heading-less block between two headed groups reads as belonging to the heading above it.
SELECT ok( (SELECT g->>'group_key' = 'subject:bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
                   AND g->'group_label' IS NOT NULL
                   AND json_typeof(g->'group_label') = 'null'
            FROM mara_groups WHERE ord = 1),
  'the unheaded remainder is keyed by the page''s own subject and comes first');

-- ── no record is duplicated, and none is lost ───────────────────────────────────────────────────
-- A record about several things is filed under exactly one of them. Printing it under each would
-- trade 25 redundant headings for redundant bodies.
SELECT is(
  (SELECT count(*) FROM mara_groups mg, json_array_elements(mg.g->'items') it)::int,
  (SELECT count(DISTINCT it->>'perception_id')::int
     FROM mara_groups mg, json_array_elements(mg.g->'items') it),
  'every record appears exactly once across all groups');

SELECT is(
  (SELECT count(DISTINCT it->>'perception_id')::int
     FROM mara_groups mg, json_array_elements(mg.g->'items') it),
  (SELECT count(DISTINCT v.perception_id)::int
     FROM fn_visible_perceptions('11111111-1111-1111-1111-111111111111',
                                 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa') v
     JOIN perception_subject ps ON ps.perception_id = v.perception_id
      AND ps.entity_id = 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
     JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'),
  'regrouping loses nothing: every held non-genesis record about the target is still on the page');

-- ── the wall ────────────────────────────────────────────────────────────────────────────────────
-- Topics come from the VIEWER's own records. Jonas holds none of the above, so none of the Player's
-- topics can appear on his copy of the same page.
SELECT is(
  (SELECT count(*)::int
     FROM json_array_elements(
            fn_actor_page('11111111-1111-1111-1111-111111111111',
                          'cccccccc-cccc-cccc-cccc-cccccccccccc',
                          'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g
    WHERE g->>'group_key' = 'subject:a4000000-0000-0000-0000-0000000000a1'),
  0,
  'perception wall: a topic built from the Player''s records never appears on Jonas''s page');

SELECT * FROM finish();
ROLLBACK;
