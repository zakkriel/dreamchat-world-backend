-- SPEC-033 — hearing teaches, if present (founder ruling 2026-08-09).
--
-- A name spoken where the viewer can hear it becomes earned, for THAT viewer only. The learning write
-- lives in the same fan-out as the naming wall (generate_perceptions), so the two cannot drift: a name
-- is taught to exactly the holders the fan-out is already writing perception rows for.
BEGIN;
SELECT plan(8);

\set w '22222222-2222-2222-2222-222222222222'
\set kade '2ac70000-0000-0000-0000-0000000000a1'
\set mara '2ac70000-0000-0000-0000-0000000000a2'
\set jonas '2ac70000-0000-0000-0000-0000000000a3'
\set hooded '2ac70000-0000-0000-0000-0000000000a4'

-- Baseline: Kade has not earned the name, and neither has the hooded woman who will not be addressed.
SELECT is(fn_display_name(:'w'::uuid, :'kade'::uuid, :'jonas'::uuid), 'the muscle by the bar',
          '(a) before: Kade knows him only as "the muscle by the bar"');
SELECT is(fn_display_name(:'w'::uuid, :'hooded'::uuid, :'jonas'::uuid), 'the muscle by the bar',
          '(b) before: so does the hooded woman across the room');

-- Mara says the name TO Kade. The hooded woman is present in the world but not an addressed
-- listener — the thin slice's "could hear" rule (§3 overhearers deferred), which this test pins so
-- that widening it later is a deliberate change and not an accident.
WITH ins AS (
  INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin)
  VALUES (:'w'::uuid, 'Communicated',
          'Mara tells the stranger the man at the bar is called Jonas.', 910, 0, 'accepted', 'freeform')
  RETURNING event_id
)
SELECT event_id INTO TEMP ev FROM ins;

INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
SELECT ev.event_id, x.id, 'actor', x.role FROM ev,
  (VALUES (:'mara'::uuid, 'speaker'), (:'kade'::uuid, 'listener')) AS x(id, role);

SELECT generate_perceptions((SELECT event_id FROM ev));

-- (c) THE RULING: named in your presence → earned by you.
SELECT is(fn_display_name(:'w'::uuid, :'kade'::uuid, :'jonas'::uuid), 'Jonas',
          '(c) after: Kade heard the name, so Kade has earned it');

-- (d) ...for you ONLY. Out of earshot is still unearned — the wall does not leak sideways.
SELECT is(fn_display_name(:'w'::uuid, :'hooded'::uuid, :'jonas'::uuid), 'the muscle by the bar',
          '(d) after: the unaddressed hooded woman learned nothing');

-- (e) The listener's own perception row KEEPS the name — teach-then-render means the wall sees a name
--     he has just earned and leaves it standing. This is the whole ordering argument, pinned.
SELECT is(
  (SELECT content FROM perception_record
    WHERE world_id = :'w'::uuid AND holder_id = :'kade'::uuid AND acquired_tick = 910),
  'Mara tells the stranger the man at the bar is called Jonas.',
  '(e) the utterance reaches the listener with the name intact — he is being told it'
);

-- (f) A holder who did NOT hear it still gets it rewritten, on the very same text.
SELECT is(
  fn_viewer_text(:'w'::uuid, :'hooded'::uuid, 'Mara tells the stranger the man at the bar is called Jonas.'),
  -- She has earned NEITHER name: "Mara" is rewritten too, which is the wall working per-viewer
  -- rather than per-word. The interesting half is that "Jonas" is still hidden from her after Kade
  -- learned it.
  'the keeper tells the stranger the man at the bar is called the muscle by the bar.',
  '(f) the same sentence rendered for someone out of earshot still hides the name'
);

-- (g) THE BELT admits it from the next beat: the Go-side wall reads fn_unearned_names, so a name that
--     has been earned must leave that set immediately — otherwise narration saying "Jonas" would be
--     rejected forever after the player was told it.
SELECT ok(
  NOT EXISTS (SELECT 1 FROM fn_unearned_names(:'w'::uuid, :'kade'::uuid) WHERE canonical_name = 'Jonas')
  AND EXISTS (SELECT 1 FROM fn_unearned_names(:'w'::uuid, :'hooded'::uuid) WHERE canonical_name = 'Jonas'),
  '(g) the belt releases the name for the listener and keeps guarding it for everyone else'
);

-- (h) First hearing wins, and a second utterance is a harmless no-op rather than a duplicate-key
--     failure that would kill the beat.
WITH ins2 AS (
  INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin)
  VALUES (:'w'::uuid, 'Communicated', 'Mara says Jonas again.', 911, 0, 'accepted', 'freeform')
  RETURNING event_id
)
SELECT event_id INTO TEMP ev2 FROM ins2;
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
SELECT ev2.event_id, x.id, 'actor', x.role FROM ev2,
  (VALUES (:'mara'::uuid, 'speaker'), (:'kade'::uuid, 'listener')) AS x(id, role);
SELECT generate_perceptions((SELECT event_id FROM ev2));

SELECT is(
  (SELECT count(*) FROM name_knowledge
    WHERE world_id = :'w'::uuid AND holder_id = :'kade'::uuid AND entity_id = :'jonas'::uuid),
  1::bigint,
  '(h) hearing the same name twice records it once, at the tick he first heard it'
);

SELECT * FROM finish();
ROLLBACK;
