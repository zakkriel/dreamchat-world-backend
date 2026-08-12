-- The persistent transcript (transcript/1) — the viewer's lived story, kept with the world.
--
-- The property that matters most is the one nothing else in the system has: these rows are a RECORD,
-- not a projection. They are never recomputed and never re-labelled, so an entry keeps the words the
-- player actually read even after the world moves on and he learns a name.
BEGIN;
SELECT plan(9);

\set w '22222222-2222-2222-2222-222222222222'
\set kade '2ac70000-0000-0000-0000-0000000000a1'
\set mara '2ac70000-0000-0000-0000-0000000000a2'
\set jonas '2ac70000-0000-0000-0000-0000000000a3'

-- (a) history starts when the feature does: no rows are invented for beats played before it landed
SELECT is(
  jsonb_array_length(fn_transcript(:'w'::uuid, :'kade'::uuid)->'entries'), 0,
  '(a) a world played before the transcript existed has an EMPTY story, not a fabricated one'
);
SELECT is(
  fn_transcript(:'w'::uuid, :'kade'::uuid)->>'next_before', NULL,
  '(b) an empty story has no cursor — the client stops rather than paging forever'
);

-- Three beats, written the way the API writes them: prose as delivered, labels frozen at write time.
INSERT INTO transcript_entry (world_id, viewer_id, in_world_tick, stated, segments, halt_reason)
VALUES
  (:'w'::uuid, :'kade'::uuid, 50, 'I look around',
   '[{"speaker_id":null,"speaker_label":"","kind":"narration","text":"The muscle by the bar does not move.","quote":null}]'::jsonb,
   'completed'),
  (:'w'::uuid, :'kade'::uuid, 50, 'I ask her his name',
   '[{"speaker_id":"2ac70000-0000-0000-0000-0000000000a2","speaker_label":"Mara","kind":"speech","text":"she tilts her head","quote":"That is Jonas."}]'::jsonb,
   'completed'),
  -- another viewer's story, which must never appear in Kade's
  (:'w'::uuid, :'mara'::uuid, 50, NULL,
   '[{"speaker_id":null,"speaker_label":"","kind":"narration","text":"The boy asks after Jonas.","quote":null}]'::jsonb,
   'completed');

-- (c) newest first
SELECT is(
  fn_transcript(:'w'::uuid, :'kade'::uuid)->'entries'->0->>'stated',
  'I ask her his name',
  '(c) the story is newest-first: the most recent beat leads'
);

-- (d) viewer-scoped: one person's experience, never another's
SELECT is(
  jsonb_array_length(fn_transcript(:'w'::uuid, :'kade'::uuid)->'entries'), 2,
  '(d) Kade sees only his own two beats, not Mara''s'
);

-- (e) the narration/2 split survives storage: a renderer keys on `quote` for speech and `text` for
--     staging, in history exactly as it does live
SELECT is(
  fn_transcript(:'w'::uuid, :'kade'::uuid)->'entries'->0->'segments'->0->>'quote',
  'That is Jonas.',
  '(e) a stored speech segment keeps its verbatim quote separate from its staging'
);
SELECT is(
  fn_transcript(:'w'::uuid, :'kade'::uuid)->'entries'->0->'segments'->0->>'text',
  'she tilts her head',
  '(f) ...and the staging stays in text'
);

-- (g) THE EPISTEMIC RULE. Kade now learns the name — every LIVE surface moves to "Jonas" — and the
--     older entry must still read "the muscle by the bar", because that is what he read.
INSERT INTO name_knowledge (world_id, holder_id, entity_id, name, learned_tick, source_event_id)
SELECT :'w'::uuid, :'kade'::uuid, :'jonas'::uuid, 'Jonas', 60, ce.event_id
FROM canon_event ce WHERE ce.world_id = :'w'::uuid ORDER BY ce.in_world_tick LIMIT 1;

SELECT is(
  fn_display_name(:'w'::uuid, :'kade'::uuid, :'jonas'::uuid), 'Jonas',
  '(g) fixture check: the live world now calls him Jonas for Kade'
);
SELECT is(
  fn_transcript(:'w'::uuid, :'kade'::uuid)->'entries'->1->'segments'->0->>'text',
  'The muscle by the bar does not move.',
  '(h) THE MEMORY IS NOT REWRITTEN: the older entry still says what he read at the time'
);

-- (i) pagination: limit + the exclusive cursor walk the whole story exactly once
SELECT is(
  (SELECT (fn_transcript(:'w'::uuid, :'kade'::uuid,
             (fn_transcript(:'w'::uuid, :'kade'::uuid, NULL, 1)->>'next_before')::bigint, 50)
           ->'entries'->0->>'stated')),
  'I look around',
  '(i) next_before pages back to the older beat, without repeating the newer one'
);

SELECT * FROM finish();
ROLLBACK;
