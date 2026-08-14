-- SPEC-033, the negative half — hearing teaches; being DESCRIBED does not.
--
-- THE FOUNDER'S BREACH (live play, 2026-08-14). A speaker label read "Jonas" to a player who had never
-- been told the name, and no line of dialogue in the transcript ever said it.
--
-- generate_perceptions taught from `said` — the referee's ACCOUNT of an utterance plus the words. An
-- account names its participants CANONICALLY, because canon is where canonical names live, so every
-- Communicated event taught every listener the name of everyone the account mentioned. A nod did it.
-- Two real rows from the seeded world before 20260814170000:
--
--   holder Mara  learned "Kade"         from  "Kade nods to Mara across the bar"
--   holder Kade  learned "Cellar Hatch" from  "a commotion erupts from the cellar hatch"
--
-- The second never contained the name: the match was case-INSENSITIVE, so a common noun read as a
-- proper name. And it compounds — once a name is in name_knowledge, fn_unearned_names drops it from
-- the unearned set, so the wall stops rewriting it in EVERY channel at once, including the
-- speaker_label the founder saw (read straight from fn_display_name, no belt of its own).
--
-- 26_hearing_teaches_test.sql pins that the words DO teach. This pins that nothing else does.
BEGIN;
SELECT plan(6);

\set w '22222222-2222-2222-2222-222222222222'
\set kade '2ac70000-0000-0000-0000-0000000000a1'
\set mara '2ac70000-0000-0000-0000-0000000000a2'
\set jonas '2ac70000-0000-0000-0000-0000000000a3'

SELECT is(fn_display_name(:'w'::uuid, :'kade'::uuid, :'jonas'::uuid), 'the muscle by the bar',
          '(a) before: Kade knows him only as "the muscle by the bar"');

-- The account names Jonas outright. The WORDS never do — he is being talked ABOUT, and what Mara
-- actually says carries no name at all. This is the founder's beat, reduced to its mechanism.
WITH ins AS (
  INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, beat_seq, status, origin, payload)
  VALUES (:'w'::uuid, 'Communicated', 'Jonas plants himself between Kade and Mara.',
          930, 0, 'accepted', 'freeform',
          jsonb_build_object('spoken', 'you sit quiet, you leave quiet'))
  RETURNING event_id
)
SELECT event_id INTO TEMP ev FROM ins;

INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
SELECT ev.event_id, x.id, 'actor', x.role FROM ev,
  (VALUES (:'mara'::uuid, 'speaker'), (:'kade'::uuid, 'listener')) AS x(id, role);

SELECT generate_perceptions((SELECT event_id FROM ev));

-- (b) Nothing was taught. The account is bookkeeping, not an introduction.
SELECT is(
  (SELECT count(*) FROM name_knowledge
    WHERE world_id = :'w'::uuid AND holder_id = :'kade'::uuid AND entity_id = :'jonas'::uuid),
  0::bigint,
  '(b) an account that names him teaches the listener nothing — nobody said it'
);

-- (c) The label the founder actually saw. speaker_label is fn_display_name, so this IS that channel.
SELECT is(fn_display_name(:'w'::uuid, :'kade'::uuid, :'jonas'::uuid), 'the muscle by the bar',
          '(c) the speaker_label still reads the descriptor, not the canonical name');

-- (d) The belt keeps guarding it. If the name had leaked into name_knowledge, fn_unearned_names would
--     have dropped it and every lens would be free to say "Jonas" from here on.
SELECT ok(
  EXISTS (SELECT 1 FROM fn_unearned_names(:'w'::uuid, :'kade'::uuid) WHERE canonical_name = 'Jonas'),
  '(d) the wall still guards the name for the man who was never told it'
);

-- (e) And the perception he actually holds has the name rewritten out of the account.
SELECT is(
  (SELECT content FROM perception_record
    WHERE world_id = :'w'::uuid AND holder_id = :'kade'::uuid AND acquired_tick = 930),
  'the muscle by the bar plants himself between Kade and Mara. — "you sit quiet, you leave quiet"',
  '(e) the account reaches him with the name rewritten; his own name and Mara''s (earned) stand'
);

-- (f) The case-insensitive match that invented three names out of ordinary prose. Teaching is strict:
--     a common noun is not a proper name, however it is spelled. Rewriting stays liberal — the two
--     want opposite strictness and both directions fail safe only when they differ.
SELECT ok(
  NOT EXISTS (
    SELECT 1 FROM fn_names_in_text(:'w'::uuid,
      'a commotion erupts from the cellar hatch and a dog barks in the alley')
    WHERE canonical_name IN ('Cellar Hatch', 'Cellar', 'Alley')
  ),
  '(f) lowercase common nouns teach no proper names'
);

SELECT * FROM finish();
ROLLBACK;
