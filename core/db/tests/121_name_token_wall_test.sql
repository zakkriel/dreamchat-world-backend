-- The naming wall guards a person's name TOKENS, not just the whole registry string.
--
-- Live breach (Railway, 2026-08-20, world "The Ironmoor Reach"): genesis stored slug join-keys as
-- canonical names — silas_holton, emmett_vale — so fn_unearned_names guarded strings no model ever
-- writes. The cognition seats humanised the slugs in their own prose, that prose became the player's
-- perception content verbatim, and the narrator handed the player "Silas" and "Emmett" with the belt
-- watching. Prose speaks in words: for PEOPLE, every distinctive word of an unearned name is guarded
-- like the name itself. Objects and places keep whole-string guarding only — their names are made of
-- ordinary nouns ("the mantel letter"), and censoring those words would eat the narrator's English.
BEGIN;
SELECT plan(12);
\set w 'ffab0000-ffff-0000-0000-000000000000'
\set ada 'ffab0000-0000-0000-0000-0000000000a0'

-- The viewer, three strangers she has earned no name for, and one artifact. One stranger shares her
-- given name; one carries a nobiliary particle; one is the Ironmoor slug, verbatim.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  (:'ada'::uuid,                              :'w'::uuid, 'actor',    'Ada Vernon'),
  ('ffab0000-0000-0000-0000-0000000000a1',    :'w'::uuid, 'actor',    'Silas Holton'),
  ('ffab0000-0000-0000-0000-0000000000a2',    :'w'::uuid, 'actor',    'emmett_vale'),
  ('ffab0000-0000-0000-0000-0000000000a3',    :'w'::uuid, 'actor',    'Ada Marsh'),
  ('ffab0000-0000-0000-0000-0000000000a5',    :'w'::uuid, 'actor',    'Hooded Woman'),
  ('ffab0000-0000-0000-0000-0000000000a4',    :'w'::uuid, 'actor',    'Theo van Housen'),
  ('ffab0000-0000-0000-0000-0000000000b1',    :'w'::uuid, 'artifact', 'the_mantel_letter');
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
  (:'ada'::uuid,                           :'w'::uuid, '{"descriptor":"a woman in a long coat"}'::jsonb),
  ('ffab0000-0000-0000-0000-0000000000a1', :'w'::uuid, '{"descriptor":"a man in a grey suit"}'::jsonb),
  ('ffab0000-0000-0000-0000-0000000000a2', :'w'::uuid, '{"descriptor":"a younger man by the curtain"}'::jsonb),
  ('ffab0000-0000-0000-0000-0000000000a3', :'w'::uuid, '{"descriptor":"a clerk with ink-stained hands"}'::jsonb),
  ('ffab0000-0000-0000-0000-0000000000a5', :'w'::uuid, '{"descriptor":"a hooded figure"}'::jsonb),
  ('ffab0000-0000-0000-0000-0000000000a4', :'w'::uuid, '{"descriptor":"a traveller with road dust on his boots"}'::jsonb);
INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
  ('ffab0000-0000-0000-0000-0000000000b1', :'w'::uuid, '{"descriptor":"a sealed letter on the mantelpiece"}'::jsonb);

SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'Silas smiles, slow and cold.'),
  'a man in a grey suit smiles, slow and cold.',
  '(a) a given name alone is rewritten — the whole-string wall alone never fires on it'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'He turns his head an inch toward Emmett.'),
  'He turns his head an inch toward a younger man by the curtain.',
  '(b) a slug canonical name (the Ironmoor shape) guards its human tokens'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'Holton and Vale trade a look.'),
  'a man in a grey suit and a younger man by the curtain trade a look.',
  '(c) family-name tokens are guarded too'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'SILAS waits.'),
  'a man in a grey suit waits.',
  '(d) every casing of a token is rewritten'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'Silas Holton waits.'),
  'a man in a grey suit waits.',
  '(e) the whole name still becomes ONE label — longest match first, tokens never double-substitute'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'Ada waits by the door.'),
  'Ada waits by the door.',
  '(f) the viewer''s OWN name is never censored, even when a stranger shares a word of it'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'Marsh waits by the door.'),
  'a clerk with ink-stained hands waits by the door.',
  '(g) …but that stranger''s other token still guards'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'The van idles in the alley.'),
  'The van idles in the alley.',
  '(h) a name particle is not a name — "van" stays ordinary English'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'The silastic tube sat by the emmettite ore.'),
  'The silastic tube sat by the emmettite ore.',
  '(i) tokens never bite into longer words'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'Dust gathers on the mantel and the letter stays sealed.'),
  'Dust gathers on the mantel and the letter stays sealed.',
  '(j) object-name tokens are NOT guarded — only people''s names are made of proper nouns'
);
SELECT is(
  fn_viewer_text(:'w'::uuid, :'ada'::uuid, 'A woman waits by the door.'),
  'A woman waits by the door.',
  '(l) an epithet token the world already uses in lowercase ("a woman in a long coat") is ordinary vocabulary, not a name — "Hooded Woman" must not censor "woman"'
);

-- Belt parity, same shape as 29(f): whatever fn_unearned_names lists — names and tokens alike —
-- fn_viewer_text removes, so the Go belt and the seam can never disagree about a row.
SELECT ok(
  (SELECT count(*) FROM fn_unearned_names(:'w'::uuid, :'ada'::uuid) u
    WHERE fn_viewer_text(:'w'::uuid, :'ada'::uuid, u.canonical_name || ' waits.')
          ~* ('\m' || fn_regexp_quote(u.canonical_name) || '\M')) = 0,
  '(k) not one listed name or token survives its own rewrite'
);

SELECT * FROM finish();
ROLLBACK;
