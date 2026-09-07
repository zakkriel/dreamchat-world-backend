-- Spoken words are canon, so a quote can be verified.
--
-- Before this, canon_event.payload was {} for every Communicated event and the summary was the
-- referee's ACCOUNT of the utterance. The world knew someone spoke and never what they said, so every
-- speech segment the narrator wrote was refused as unverifiable and kind:"speech" was unreachable.
--
-- Since the shared speech perception migration, every visible Communicated apply_event call also
-- requires an already-judged `speech_perception` object (missing/malformed -> gate_reject, never a
-- runtime fallback) — see 124_speech_perception_test.sql for that contract's own coverage. This file
-- stays focused on its original concern: payload.spoken persistence, the listener actually receiving
-- the verbatim words, and the naming wall's account-level rendering being unaffected by any of it.
BEGIN;
SELECT plan(6);

\set w '22222222-2222-2222-2222-222222222222'
\set kade '2ac70000-0000-0000-0000-0000000000a1'
\set mara '2ac70000-0000-0000-0000-0000000000a2'
\set jonas '2ac70000-0000-0000-0000-0000000000a3'

-- Mara speaks to Kade. `stated` is the account; `content` is the utterance. Kade's judgment: attended,
-- no name association (nobody is named in the words), heard the words verbatim.
SELECT apply_event(
  :'w'::uuid,
  :'mara'::uuid,
  jsonb_build_object(
    'type', 'Communicated',
    'stated', 'Mara answers the stranger with a dry remark',
    'listener_id', :'kade',
    'content', 'You are at my bar, not in his way.',
    'speech_perception', jsonb_build_object(
      'schema_version', 'speech_perception/1',
      'listeners', jsonb_build_array(jsonb_build_object(
        'listener_id', :'kade',
        'attention', jsonb_build_object('kind', 'abstain'),
        'name_associations', '[]'::jsonb,
        'heard_words', 'You are at my bar, not in his way.'
      ))
    )),
  800, 0, 'freeform'
) INTO TEMP applied;

-- (a) the words are canon now, not lost
SELECT is(
  (SELECT payload->>'spoken' FROM canon_event WHERE world_id = :'w'::uuid AND in_world_tick = 800),
  'You are at my bar, not in his way.',
  '(a) apply_event persists `content` as canon payload.spoken — the utterance survives the commit'
);

-- (b) the summary is untouched: the account and the words are different facts, both kept
SELECT is(
  (SELECT summary FROM canon_event WHERE world_id = :'w'::uuid AND in_world_tick = 800),
  'Mara answers the stranger with a dry remark',
  '(b) `stated` still records the ACT; the words did not overwrite it'
);

-- (c) THE LISTENER HEARS THE WORDS. Without this the narrator has nothing verbatim in its payload and
--     keeps inventing dialogue the belt then refuses — the deadlock that made speech unreachable.
SELECT ok(
  (SELECT content FROM perception_record
    WHERE world_id = :'w'::uuid AND holder_id = :'kade'::uuid AND acquired_tick = 800)
    LIKE '%You are at my bar, not in his way.%',
  '(c) the listener''s perception carries the spoken words, not just the account'
);

-- (d) ...and still through the naming wall, per holder. The hooded woman was never a candidate in
--     this judgment at all (absent from speech_perception.listeners) — she perceives nothing.
SELECT is(
  (SELECT count(*) FROM perception_record
    WHERE world_id = :'w'::uuid AND holder_id = '2ac70000-0000-0000-0000-0000000000a4' AND acquired_tick = 800),
  0::bigint,
  '(d) someone who was not a judged candidate perceives nothing — the fan-out still decides who heard it'
);

-- (e) an utterance with no words recorded backs no quote: the honest answer is "nobody knows what was
--     said", never a paraphrase promoted to dialogue. Kade is still attended (he heard SOMETHING, a
--     mutter) but heard_words is empty — there is nothing to quote.
SELECT apply_event(
  :'w'::uuid, :'mara'::uuid,
  jsonb_build_object('type','Communicated','stated','Mara mutters something','listener_id', :'kade',
    'speech_perception', jsonb_build_object(
      'schema_version', 'speech_perception/1',
      'listeners', jsonb_build_array(jsonb_build_object(
        'listener_id', :'kade',
        'attention', jsonb_build_object('kind', 'abstain'),
        'name_associations', '[]'::jsonb,
        'heard_words', ''
      ))
    )),
  801, 0, 'freeform'
) INTO TEMP applied2;

SELECT is(
  (SELECT payload->>'spoken' FROM canon_event WHERE world_id = :'w'::uuid AND in_world_tick = 801),
  NULL,
  '(e) a Communicated event with no content stores no spoken words — nothing to quote, and no pretence'
);
SELECT is(
  (SELECT content FROM perception_record
    WHERE world_id = :'w'::uuid AND holder_id = :'kade'::uuid AND acquired_tick = 801),
  'Mara mutters something',
  '(f) ...and its perception is the bare account, with no invented quotation'
);

-- Recognized-actor-teaches-a-name coverage (the model-driven replacement for the old regex scanner)
-- lives in 124_speech_perception_test.sql, which exercises it directly against name_knowledge and
-- fn_display_name rather than duplicating it here.

SELECT * FROM finish();
ROLLBACK;
