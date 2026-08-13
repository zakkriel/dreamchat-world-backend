-- The naming wall's substitution is article-aware.
--
-- Live symptom, in a telegraph the player read: "The a hooded figure shifts silently, drawing a step
-- closer to the stranger's flank." The wall did its job — the canonical name is gone — and produced
-- prose no narrator would write, which then froze into the transcript as the memory of that moment.
BEGIN;
SELECT plan(6);

\set w '22222222-2222-2222-2222-222222222222'
\set kade '2ac70000-0000-0000-0000-0000000000a1'

-- The registry name is "Hooded Companion"; Kade's label for her is "a hooded figure".
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'The Hooded Companion shifts silently.'),
  'The hooded figure shifts silently.',
  '(a) an article already in the sentence is kept, and the label does not add a second'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'the Hooded Companion shifts silently.'),
  'the hooded figure shifts silently.',
  '(b) the sentence''s own capitalisation survives — "the" stays lowercase mid-sentence'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'Hooded Companion shifts silently.'),
  'a hooded figure shifts silently.',
  '(c) with NO article present the label arrives whole, article included'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'A Hooded Woman waits by the door.'),
  'A hooded figure waits by the door.',
  '(d) "A" is an article too, and is kept as written'
);

-- The rule must not touch a label that has no article of its own, earned or not.
SELECT is(
  fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'Jonas blocks the way and Mara watches.'),
  'the muscle by the bar blocks the way and Mara watches.',
  '(e) an articleless label is substituted whole; an EARNED name is left alone'
);

-- And the wall itself still holds: no unearned canonical name survives any of this.
SELECT ok(
  (SELECT count(*) FROM fn_unearned_names(:'w'::uuid, :'kade'::uuid) u
    WHERE fn_viewer_text(:'w'::uuid, :'kade'::uuid, 'The ' || u.canonical_name || ' waits.')
          ~* ('\m' || u.canonical_name || '\M')) = 0,
  '(f) after an article-aware rewrite, not one unearned name is left standing'
);

SELECT * FROM finish();
ROLLBACK;
