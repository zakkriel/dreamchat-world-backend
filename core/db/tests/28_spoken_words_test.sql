-- Spoken words are canon, so a quote can be verified.
--
-- Before this, canon_event.payload was {} for every Communicated event and the summary was the
-- referee's ACCOUNT of the utterance. The world knew someone spoke and never what they said, so every
-- speech segment the narrator wrote was refused as unverifiable and kind:"speech" was unreachable.
BEGIN;
SELECT plan(7);

\set w '22222222-2222-2222-2222-222222222222'
\set kade '2ac70000-0000-0000-0000-0000000000a1'
\set mara '2ac70000-0000-0000-0000-0000000000a2'
\set jonas '2ac70000-0000-0000-0000-0000000000a3'

-- Mara speaks to Kade. `stated` is the account; `content` is the utterance.
SELECT apply_event(
  :'w'::uuid,
  :'mara'::uuid,
  jsonb_build_object(
    'type', 'Communicated',
    'stated', 'Mara answers the stranger with a dry remark',
    'listener_id', :'kade',
    'content', 'You are at my bar, not in his way.'),
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

-- (d) ...and still through the naming wall, per holder. The hooded woman did not hear this at all.
SELECT is(
  (SELECT count(*) FROM perception_record
    WHERE world_id = :'w'::uuid AND holder_id = '2ac70000-0000-0000-0000-0000000000a4' AND acquired_tick = 800),
  0::bigint,
  '(d) someone who was not addressed perceives nothing — the fan-out still decides who heard it'
);

-- (e) an utterance with no words recorded backs no quote: the honest answer is "nobody knows what was
--     said", never a paraphrase promoted to dialogue
SELECT apply_event(
  :'w'::uuid, :'mara'::uuid,
  jsonb_build_object('type','Communicated','stated','Mara mutters something','listener_id', :'kade'),
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

-- (g) SPEC-033 now reads what was SAID: a name spoken aloud is exactly the case the founder ruled on.
SELECT apply_event(
  :'w'::uuid, :'mara'::uuid,
  jsonb_build_object('type','Communicated','stated','Mara names the man at the bar',
                     'listener_id', :'kade', 'content','That is Jonas, and he is not moving.'),
  802, 0, 'freeform'
) INTO TEMP applied3;

SELECT is(
  fn_display_name(:'w'::uuid, :'kade'::uuid, :'jonas'::uuid), 'Jonas',
  '(g) hearing the name INSIDE the spoken words teaches it — the ruling''s central case'
);

SELECT * FROM finish();
ROLLBACK;
