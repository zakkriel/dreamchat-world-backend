-- The naming wall at the perception seam (B-1, I-3, naming reach RULINGS-2026-07-23 §3).
--
-- Reported live: narration told Kade "Jonas planted between her and the room". Kade has never earned
-- that name. The narrator was faithful — his OWN perception rows carried it, because both fan-out
-- writers copied one canonically-named string into every holder's row.
BEGIN;
SELECT plan(9);

\set w '22222222-2222-2222-2222-222222222222'
\set kade '2ac70000-0000-0000-0000-0000000000a1'
\set mara '2ac70000-0000-0000-0000-0000000000a2'

-- (a) the reported sentence, rendered for the viewer who has not earned the name
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid,
    'Jonas pushes off the bar and steps between Kade and Mara, blocking the way.'),
  'the muscle by the bar pushes off the bar and steps between Kade and Mara, blocking the way.',
  '(a) Kade sees "the muscle by the bar"; Mara he has EARNED and his OWN name he obviously holds'
);

-- (b) viewer-relative both ways: Mara knows Jonas, so nothing is rewritten for her
SELECT is(
  fn_viewer_text(:'w'::uuid, :'mara'::uuid, 'Jonas pushes off the bar.'),
  'Jonas pushes off the bar.',
  '(b) a holder who has earned the name reads the name — the wall is per-viewer, not a global censor'
);

-- (c) case-insensitive: prose capitalises at a sentence start
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'JONAS planted himself there. jonas did not move.'),
  'the muscle by the bar planted himself there. the muscle by the bar did not move.',
  '(c) every casing of an unearned name is rewritten'
);

-- (d) word boundaries: a name must not be rewritten inside a longer word
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'The jonasberry pie and Maras cup sat untouched.'),
  'The jonasberry pie and Maras cup sat untouched.',
  '(d) "jonasberry" is not Jonas'
);

-- (e) THE REGRESSION: the fan-out writes what the holder perceived, not what the referee wrote.
--     An ActorMoved event, which is the shape the founder's leak actually took — the polluted rows
--     on the live world were ActorMoved and AttributeChanged, not speech. Speech is deliberately NOT
--     used here: since SPEC-033 an utterance TEACHES the names in it (see 26_hearing_teaches), so a
--     Communicated fixture would test the learning path while claiming to test the wall.
-- Separate statements on purpose: a data-modifying CTE is not visible to a function called in the
-- same statement, so generate_perceptions would see no participants and write nothing.
WITH ins AS (
  INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin)
  VALUES (:'w'::uuid, 'ActorMoved',
          'Jonas steps into the stranger''s path, blocking the way to Mara.', 900, 0, 'accepted', 'freeform')
  RETURNING event_id
)
SELECT event_id INTO TEMP ev FROM ins;

INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
SELECT ev.event_id, :'kade'::uuid, 'actor', 'instigator' FROM ev;

SELECT generate_perceptions((SELECT event_id FROM ev));

SELECT is(
  (SELECT content FROM perception_record
    WHERE world_id = :'w'::uuid AND holder_id = :'kade'::uuid AND acquired_tick = 900),
  'the muscle by the bar steps into the stranger''s path, blocking the way to Mara.',
  '(e) generate_perceptions renders the mover''s row in the MOVER''S vocabulary'
);

-- (f) canon is untouched: the referee's true account keeps the canonical name (immutability, D-1;
--     I-1 replay reads canon, not this projection)
SELECT is(
  (SELECT summary FROM canon_event WHERE world_id = :'w'::uuid AND in_world_tick = 900),
  'Jonas steps into the stranger''s path, blocking the way to Mara.',
  '(f) canon_event.summary still says Jonas — the wall bounds perception, it does not rewrite truth'
);

-- (g) the same sentence for a holder who HAS earned the name is left alone — per-viewer, not a
--     global censor.
SELECT is(
  fn_viewer_text(:'w'::uuid, :'mara'::uuid, 'Jonas steps into the path, blocking the way.'),
  'Jonas steps into the path, blocking the way.',
  '(g) Mara knows him, so her rendering keeps the name'
);

-- (h) THE BALLAST CRATE. The registry says "Ballast Crate"; Kade's label is "the ballast crate".
--     They differ by an article, so the name carries no knowledge he lacks — and the belt rejected a
--     correct narration line over it in live play before fn_unearned_names got the containment rule.
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'He shoves past the ballast crate.'),
  'He shoves past the ballast crate.',
  '(h) a label that already CONTAINS the canonical name is not a breach'
);

-- (i) ...and the narrow reading is kept: a name the label does NOT contain is still guarded, which is
--     why this is a containment rule and not "artifacts are exempt".
SELECT ok(
  EXISTS (SELECT 1 FROM fn_unearned_names(:'w'::uuid, :'kade'::uuid) WHERE canonical_name = 'Jonas')
  AND NOT EXISTS (SELECT 1 FROM fn_unearned_names(:'w'::uuid, :'kade'::uuid) WHERE canonical_name = 'Ballast Crate'),
  '(i) fn_unearned_names guards Jonas and releases Ballast Crate — one definition, both callers'
);

SELECT * FROM finish();
ROLLBACK;
