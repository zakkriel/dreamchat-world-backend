-- Shared speech perception: one application path for Communicated, through both commit doors,
-- driven by an already-judged `speech_perception` object rather than a canonical-name scan. Fixed
-- UUIDs, f7000000-prefixed, self-contained, ROLLBACK at end.
--
--   world:                                f7000000-ffff-0000-0000-000000000000
--   A  (ordinary speaker, "Mara-124"):     f7000000-0000-0000-0000-000000000001
--   B  (listener: recognized-actor case):  f7000000-0000-0000-0000-000000000002
--   C  ("Jonas-124", absent throughout):   f7000000-0000-0000-0000-000000000003
--   E  (listener: ambiguous-name case):    f7000000-0000-0000-0000-000000000004
--   F  (listener: description-without-recognition case): f7000000-0000-0000-0000-000000000005
--   G  (listener: blocked case):           f7000000-0000-0000-0000-000000000006
--   H  (ruled speaker):                    f7000000-0000-0000-0000-000000000007
--   I  (ruled listener, co-located w/ H):  f7000000-0000-0000-0000-000000000008
--   J  ("Ren-124", recognized via apply_beat): f7000000-0000-0000-0000-000000000009
--   K  (nonparticipant: related-actor anchor in C, hears-nothing control in I): f7000000-0000-0000-0000-00000000000a
--   room:                                  f7000000-0000-0000-0000-000000000010
BEGIN;
SELECT plan(49);

\set w      'f7000000-ffff-0000-0000-000000000000'
\set a      'f7000000-0000-0000-0000-000000000001'
\set b      'f7000000-0000-0000-0000-000000000002'
\set c      'f7000000-0000-0000-0000-000000000003'
\set e      'f7000000-0000-0000-0000-000000000004'
\set f      'f7000000-0000-0000-0000-000000000005'
\set g      'f7000000-0000-0000-0000-000000000006'
\set h      'f7000000-0000-0000-0000-000000000007'
\set i      'f7000000-0000-0000-0000-000000000008'
\set j      'f7000000-0000-0000-0000-000000000009'
\set k      'f7000000-0000-0000-0000-00000000000a'
\set room   'f7000000-0000-0000-0000-000000000010'

INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
 (:'a','f7000000-ffff-0000-0000-000000000000','actor','Mara-124'),
 (:'b','f7000000-ffff-0000-0000-000000000000','actor','Kade-124'),
 (:'c','f7000000-ffff-0000-0000-000000000000','actor','Jonas-124'),
 (:'e','f7000000-ffff-0000-0000-000000000000','actor','test-E-124'),
 (:'f','f7000000-ffff-0000-0000-000000000000','actor','test-F-124'),
 (:'g','f7000000-ffff-0000-0000-000000000000','actor','test-G-124'),
 (:'h','f7000000-ffff-0000-0000-000000000000','actor','test-H-124'),
 (:'i','f7000000-ffff-0000-0000-000000000000','actor','test-I-124'),
 (:'j','f7000000-ffff-0000-0000-000000000000','actor','Ren-124'),
 (:'k','f7000000-ffff-0000-0000-000000000000','actor','test-K-124'),
 (:'room','f7000000-ffff-0000-0000-000000000000','location','test-room-124');

-- C and J are absent throughout — descriptors only, no location_id, so fn_display_name initially
-- falls through to the descriptor rather than the bare canonical name.
INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
 (:'c','f7000000-ffff-0000-0000-000000000000', jsonb_build_object('descriptor', 'a name mentioned in passing')),
 (:'j','f7000000-ffff-0000-0000-000000000000', jsonb_build_object('descriptor', 'someone Mara mentioned once'));

-- Everyone else co-located at room via canonical setup moves (A, B, E, F, G, H, I).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
SELECT ('f7500000-0000-0000-0000-' || lpad(n::text, 12, '0'))::uuid, 'f7000000-ffff-0000-0000-000000000000',
       'move', 'setup', 2900 + n, 0, 'accepted', now(), 'public', 'fast_path'
FROM generate_series(1, 7) n;
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
 ('f7500000-0000-0000-0000-000000000001',:'a','actor','instigator'),
 ('f7500000-0000-0000-0000-000000000002',:'b','actor','instigator'),
 ('f7500000-0000-0000-0000-000000000003',:'e','actor','instigator'),
 ('f7500000-0000-0000-0000-000000000004',:'f','actor','instigator'),
 ('f7500000-0000-0000-0000-000000000005',:'g','actor','instigator'),
 ('f7500000-0000-0000-0000-000000000006',:'h','actor','instigator'),
 ('f7500000-0000-0000-0000-000000000007',:'i','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq)
SELECT 'f7000000-ffff-0000-0000-000000000000', ('f7500000-0000-0000-0000-' || lpad(n::text, 12, '0'))::uuid,
       actor, 'actor', 'attrs.location_id', to_jsonb(:'room'::text), 2900 + n, 0
FROM (VALUES (1,:'a'::uuid),(2,:'b'::uuid),(3,:'e'::uuid),(4,:'f'::uuid),(5,:'g'::uuid),(6,:'h'::uuid),(7,:'i'::uuid)) AS s(n, actor);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- A. Recognized introduction — B recognizes the ABSENT C. name_knowledge + perception_subject +
--    the actor-specific public read (fn_display_name, fn_entity_visible, fn_actor_page). The SAME
--    association also carries a meaningful description (A7): retained independently of, never
--    suppressed by, the actor_id recognition alongside it — no kind discriminator, no mutually
--    exclusive owner shapes.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT ok(NOT fn_entity_visible(:'w'::uuid, :'b'::uuid, :'c'::uuid),
          '(A0) before: B holds no perception about the absent C');
SELECT ok(fn_actor_page(:'w'::uuid, :'b'::uuid, :'c'::uuid) IS NULL,
          '(A0) before: C''s actor page is NULL for B (unperceived => 404)');

SELECT is(
  (apply_event(:'w'::uuid, :'a'::uuid, jsonb_build_object(
     'type','Communicated','stated','Mara points someone out by name.',
     'listener_id', :'b', 'content', 'That is Jonas.',
     'speech_perception', jsonb_build_object(
       'schema_version','speech_perception/1',
       'listeners', jsonb_build_array(jsonb_build_object(
         'listener_id', :'b',
         'attention', jsonb_build_object('kind','abstain'),
         'name_associations', jsonb_build_array(jsonb_build_object(
           'name','Jonas-124','owner',jsonb_build_object(
             'actor_id',:'c','description','someone Mara has known for years',
             'about_actor_ids',jsonb_build_array(:'k')))),
         'heard_words','That is Jonas.'
       ))
     )
  ), 3000, 0, 'freeform')->>'halt_reason'),
  'committed', '(A1) recognized-actor introduction commits'
);

SELECT is(
  (SELECT jsonb_build_array(name, learned_tick, source_event_id) FROM name_knowledge
   WHERE world_id=:'w'::uuid AND holder_id=:'b'::uuid AND entity_id=:'c'::uuid),
  jsonb_build_array('Jonas-124', 3000,
    (SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3000)),
  '(A2) B learns the recognized name with its original tick and source'
);
SELECT ok(
  EXISTS(SELECT 1 FROM perception_record pr
         JOIN perception_subject ps ON ps.perception_id = pr.perception_id
         WHERE pr.world_id=:'w'::uuid AND pr.holder_id=:'b'::uuid
           AND pr.source_event_id = (SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3000)
           AND ps.entity_id = :'c'::uuid),
  '(A3) the recognized actor becomes a perception_subject of B''s OWN record, though never a participant'
);
SELECT is(fn_display_name(:'w'::uuid, :'b'::uuid, :'c'::uuid), 'Jonas-124',
          '(A4) actor-specific public read: B''s label for C is now the learned name');
SELECT ok(fn_entity_visible(:'w'::uuid, :'b'::uuid, :'c'::uuid),
          '(A5) actor-specific public read: C is now visible to B');
SELECT ok(fn_actor_page(:'w'::uuid, :'b'::uuid, :'c'::uuid) IS NOT NULL,
          '(A6) actor-specific public read: C''s actor page now renders for B');
SELECT ok(
  (SELECT content FROM perception_record WHERE world_id=:'w'::uuid AND holder_id=:'b'::uuid
     AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3000))
    LIKE '%(Jonas-124: someone Mara has known for years)%',
  '(A7) a meaningful description is retained even though this SAME association also recognizes an actor_id'
);
SELECT ok(
  EXISTS(SELECT 1 FROM json_array_elements(
           fn_actor_page(:'w'::uuid, :'b'::uuid, :'k'::uuid)->'actor'->'collected_knowledge_groups') gr,
         json_array_elements(gr->'items') it
         WHERE it->>'perception_id' = (
           SELECT perception_id::text FROM perception_record WHERE world_id=:'w'::uuid AND holder_id=:'b'::uuid
             AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3000)))
  AND NOT EXISTS(SELECT 1 FROM name_knowledge
                 WHERE world_id=:'w'::uuid AND holder_id=:'b'::uuid AND entity_id=:'k'::uuid),
  '(A8) recognition retains the nonparticipant related actor''s dossier link without renaming her'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- B. Ambiguous heard name — E hears "Jonas-124" with no name_associations. Perceived, not identified.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT is(
  (apply_event(:'w'::uuid, :'a'::uuid, jsonb_build_object(
     'type','Communicated','stated','Jonas-124 leaves.',
     'listener_id', :'e', 'content', 'Jonas-124 left already.',
     'speech_perception', jsonb_build_object(
       'schema_version','speech_perception/1',
       'listeners', jsonb_build_array(jsonb_build_object(
         'listener_id', :'e',
         'attention', jsonb_build_object('kind','abstain'),
         'name_associations', '[]'::jsonb,
         'heard_words','Jonas-124 left.'
       ), jsonb_build_object(
         'listener_id', :'b', 'attention', jsonb_build_object('kind','abstain'),
         'name_associations', '[]'::jsonb, 'heard_words','Gone.'
       ))
     )
  ), 3001, 0, 'freeform')->>'halt_reason'),
  'committed', '(B1) ambiguous heard name still commits'
);
SELECT is(
  (SELECT count(*)::int FROM name_knowledge WHERE world_id=:'w'::uuid AND holder_id=:'e'::uuid AND entity_id=:'c'::uuid),
  0, '(B2) no identification => no name_knowledge, even though the word was heard'
);
SELECT is(
  (SELECT jsonb_object_agg(holder_id, spoken) FROM perception_record WHERE world_id=:'w'::uuid
     AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3001)),
  jsonb_build_object(:'a', 'Jonas-124 left already.', :'e', 'Jonas-124 left.', :'b', 'Gone.'),
  '(B3) each listener stores their accepted words; only the speaker retains the source utterance'
);
SELECT is(fn_display_name(:'w'::uuid, :'e'::uuid, :'c'::uuid), 'a name mentioned in passing',
          '(B4) E''s label for C is unchanged — hearing a word is not knowing its owner');

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- C. Description without recognition — F learns "Mara's brother is Jonas-124" without ever
--    recognizing C (owner.actor_id stays null: no false identity). about_actor_ids anchors the
--    description to K, an ALREADY-GROUNDED actor who is NOT an event_participant of this event —
--    deliberately NOT the speaker A, whose own perception_subject link is already inserted
--    unconditionally for every event_participant (see fn_apply_speech_perception's speaker/listener
--    own-record inserts), so A could never prove the about_actor_ids consequence added anything. K's
--    link and knowledge below therefore exercise the related-subject insert.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT ok(NOT fn_entity_visible(:'w'::uuid, :'f'::uuid, :'k'::uuid),
          '(C0) before: F holds no perception about the uninvolved, nonparticipant K');
SELECT is(
  (apply_event(:'w'::uuid, :'a'::uuid, jsonb_build_object(
     'type','Communicated','stated','Mara describes her brother.',
     'listener_id', :'f', 'content', 'My brother Jonas-124 runs the bakery.',
     'speech_perception', jsonb_build_object(
       'schema_version','speech_perception/1',
       'listeners', jsonb_build_array(jsonb_build_object(
         'listener_id', :'f',
         'attention', jsonb_build_object('kind','abstain'),
         'name_associations', jsonb_build_array(
           jsonb_build_object(
             'name','Jonas-124','owner',jsonb_build_object(
               'actor_id',NULL,'description','Mara''s brother','about_actor_ids',jsonb_build_array(:'k')))
         ),
         'heard_words','My brother Jonas-124 runs the bakery.'
       ))
     )
  ), 3002, 0, 'freeform')->>'halt_reason'),
  'committed', '(C1) a description without recognition commits'
);
SELECT is(
  (SELECT count(*)::int FROM name_knowledge WHERE world_id=:'w'::uuid AND holder_id=:'f'::uuid AND entity_id=:'c'::uuid),
  0, '(C2) a null actor_id never writes name_knowledge — no identity is granted merely from a description'
);
SELECT ok(
  NOT EXISTS(SELECT 1 FROM perception_record pr
             JOIN perception_subject ps ON ps.perception_id = pr.perception_id
             WHERE pr.world_id=:'w'::uuid AND pr.holder_id=:'f'::uuid
               AND pr.source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3002)
               AND ps.entity_id = :'c'::uuid),
  '(C3) the undescribed party (C) is never added as a perception_subject — no guessed actor link'
);
SELECT ok(
  EXISTS(SELECT 1 FROM perception_record pr
         JOIN perception_subject ps ON ps.perception_id = pr.perception_id
         WHERE pr.world_id=:'w'::uuid AND pr.holder_id=:'f'::uuid
           AND pr.source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3002)
           AND ps.entity_id = :'k'::uuid),
  '(C4) the ALREADY-GROUNDED anchor (K, about_actor_ids) becomes a perception_subject of F''s record '
  '— K is NOT an event_participant here, so this link is observable ONLY from the about_actor_ids insert'
);
SELECT is(
  (SELECT count(*)::int FROM name_knowledge WHERE world_id=:'w'::uuid AND holder_id=:'f'::uuid AND entity_id=:'k'::uuid),
  0, '(C4b) K is linked as a related subject but never recognized — about_actor_ids never teaches a name'
);
SELECT is(fn_display_name(:'w'::uuid, :'f'::uuid, :'k'::uuid), 'test-K-124',
          '(C4c) K keeps her own canonical name to F — a related-link mention never renames her');
SELECT ok(
  (SELECT content FROM perception_record WHERE world_id=:'w'::uuid AND holder_id=:'f'::uuid
     AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3002))
    LIKE '%(Jonas-124: Mara''s brother)%',
  '(C7) F''s stored knowledge content itself carries the name+description, not just the payload — a '
  'future facts read does not have to re-derive it from the utterance alone'
);
SELECT is(fn_display_name(:'w'::uuid, :'f'::uuid, :'c'::uuid), 'a name mentioned in passing',
          '(C8) C''s FUTURE label to F is unchanged — a description does not pre-name a later meeting'
);
SELECT ok(
  EXISTS(SELECT 1 FROM json_array_elements(
           fn_actor_page(:'w'::uuid, :'f'::uuid, :'k'::uuid)->'actor'->'collected_knowledge_groups') gr,
         json_array_elements(gr->'items') it
         WHERE it->>'perception_id' = (
           SELECT perception_id::text FROM perception_record WHERE world_id=:'w'::uuid AND holder_id=:'f'::uuid
             AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3002))),
  '(C9) discoverable: F''s dossier on K, a NONPARTICIPANT related actor, surfaces this perception '
  'through the ordinary subject-keyed read path — proving the about_actor_ids insert ran, not merely '
  'that K was already linked some other way'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- D. Missed speech — G is addressed (co-located, passes the gate) but the model judges her blocked.
--    Zero perception for her; the speaker still perceives her own words unconditionally.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT is(
  (apply_event(:'w'::uuid, :'a'::uuid, jsonb_build_object(
     'type','Communicated','stated','Mara addresses someone distracted.',
     'listener_id', :'g', 'content', 'Pay attention.',
     'speech_perception', jsonb_build_object(
       'schema_version','speech_perception/1',
       'listeners', jsonb_build_array(jsonb_build_object(
         'listener_id', :'g',
         'attention', jsonb_build_object('kind','blocked','reason','absorbed in another conversation'),
         'name_associations', '[]'::jsonb, 'heard_words',''
       ))
     )
  ), 3003, 0, 'freeform')->>'halt_reason'),
  'committed', '(D1) a blocked listener does not stop the event from committing'
);
SELECT is(
  (SELECT count(*)::int FROM perception_record WHERE world_id=:'w'::uuid AND holder_id=:'g'::uuid
     AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3003)),
  0, '(D2) the blocked listener holds no perception row at all from this event'
);
SELECT is(
  (SELECT count(*)::int FROM perception_record WHERE world_id=:'w'::uuid AND holder_id=:'a'::uuid
     AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3003)),
  1, '(D3) the speaker still perceives her own words unconditionally, independent of the listener''s block'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- E. Hidden ruled speech (visible:false) — zero listeners, zero learning, INCLUDING no speaker
--    perception. Go skips the model entirely; no speech_perception is ever attached.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT is(
  (apply_ruled_event(:'w'::uuid, jsonb_build_object(
     'type','Communicated','actor_id',:'h','truth','H whispers something no one should hear.',
     'listener_id', :'i', 'visible', false
  ), 3010, 0, 'ruling')->>'halt_reason'),
  'committed', '(E1) hidden ruled speech still commits the event itself'
);
SELECT is(
  (SELECT count(*)::int FROM perception_record
   WHERE source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3010)),
  0, '(E2) hidden speech: zero perception rows for anyone, including the speaker'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- G. Missing/malformed speech_perception cannot half-commit — structural gate on both doors.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

-- (ruled, visible defaults true, no speech_perception at all)
SELECT is(
  (apply_ruled_event(:'w'::uuid, jsonb_build_object(
     'type','Communicated','actor_id',:'h','truth','H says something.', 'listener_id', :'i'
  ), 3011, 0, 'ruling')->>'halt_reason'),
  'gate_reject', '(G1) visible ruled Communicated with no speech_perception is refused'
);
SELECT is(
  (SELECT count(*)::int FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3011),
  0, '(G1b) nothing committed — not even the event row'
);

-- (ordinary, no speech_perception key at all)
SELECT is(
  (apply_event(:'w'::uuid, :'a'::uuid, jsonb_build_object(
     'type','Communicated','stated','Mara says something.', 'listener_id', :'b', 'content','x'
  ), 3012, 0, 'freeform')->>'halt_reason'),
  'gate_reject', '(G2) ordinary Communicated with no speech_perception at all is refused'
);
SELECT is(
  (SELECT count(*)::int FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3012),
  0, '(G2b) nothing committed'
);

-- (ordinary, speech_perception present but malformed: not an object)
SELECT is(
  (apply_event(:'w'::uuid, :'a'::uuid, jsonb_build_object(
     'type','Communicated','stated','Mara says something else.', 'listener_id', :'b', 'content','y',
     'speech_perception', 'oops'
  ), 3013, 0, 'freeform')->>'halt_reason'),
  'gate_reject', '(G3) a malformed (non-object) speech_perception is refused, never guessed at'
);
SELECT is(
  (SELECT count(*)::int FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3013),
  0, '(G3b) nothing committed'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- F. Legacy forwarding through apply_beat — a say step with no speech_perception is refused (no
--    scanner fallback resurrected); one recognizing an actor_id applies exactly like the direct
--    apply_event door.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT is(
  (apply_beat(:'w'::uuid, :'a'::uuid,
     jsonb_build_array(jsonb_build_object('type','say','listener',:'b','content','hi')),
     3020, 100, 'fast_path')->>'halt_reason'),
  'gate_reject', '(F1) apply_beat forwards a missing speech_perception straight into the same refusal'
);
SELECT is(
  (SELECT count(*)::int FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick>=3020 AND in_world_tick<3021),
  0, '(F1b) nothing committed through the beat door either'
);

SELECT is(
  (apply_beat(:'w'::uuid, :'a'::uuid,
     jsonb_build_array(jsonb_build_object(
       'type','say','listener',:'b','content','That one is Ren-124, and Jonas-124 goes by Another-124.',
       'speech_perception', jsonb_build_object(
         'schema_version','speech_perception/1',
         'listeners', jsonb_build_array(jsonb_build_object(
           'listener_id', :'b',
           'attention', jsonb_build_object('kind','abstain'),
           'name_associations', jsonb_build_array(jsonb_build_object(
             'name','Ren-124','owner',jsonb_build_object(
               'actor_id',:'j','description',NULL,'about_actor_ids','[]'::jsonb)),
             jsonb_build_object('name','Another-124','owner',jsonb_build_object(
               'actor_id',:'c','description',NULL,'about_actor_ids','[]'::jsonb))),
           'heard_words','That one is Ren-124, and Jonas-124 goes by Another-124.'
         ))
       )
     )),
     3021, 100, 'fast_path')->>'halt_reason'),
  'completed', '(F2) apply_beat forwards an accepted speech_perception through to a normal commit'
);
SELECT is(
  (SELECT jsonb_agg(jsonb_build_array(n.entity_id, n.name, n.learned_tick, ce.in_world_tick)
                   ORDER BY n.entity_id)
   FROM name_knowledge n JOIN canon_event ce ON ce.event_id=n.source_event_id
   WHERE n.world_id=:'w'::uuid AND n.holder_id=:'b'::uuid AND n.entity_id IN (:'c'::uuid, :'j'::uuid)),
  jsonb_build_array(jsonb_build_array(:'c', 'Jonas-124', 3000, 3000),
                    jsonb_build_array(:'j', 'Ren-124', 3021, 3021)),
  '(F3) a repeated introduction commits without replacing the first name, tick or source; a new actor is learned'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- H. Historical backfill — exact content, exact quote, and the unrecoverable receiver-variant case.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

-- H1: the legacy say-step shape (summary = content exactly; no quote was ever appended).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status,
                         accepted_at, visibility_scope, origin, payload)
VALUES ('f7900000-0000-0000-0000-000000000001', :'w'::uuid, 'Communicated', 'Exact quote', 3100, 0,
        'accepted', now(), 'private', 'fast_path', jsonb_build_object('spoken','Exact quote'));
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 ('f7910000-0000-0000-0000-000000000001', :'w'::uuid, :'b'::uuid, 'f7900000-0000-0000-0000-000000000001',
  'Exact quote', 'told', 3100, 3100);

-- H2: the account+quote shape (`said := summary || ' — "' || spoken || '"'`).
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status,
                         accepted_at, visibility_scope, origin, payload)
VALUES ('f7900000-0000-0000-0000-000000000002', :'w'::uuid, 'Communicated', 'Mara answers dryly.', 3101, 0,
        'accepted', now(), 'private', 'fast_path', jsonb_build_object('spoken','the words'));
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 ('f7910000-0000-0000-0000-000000000002', :'w'::uuid, :'b'::uuid, 'f7900000-0000-0000-0000-000000000002',
  'Mara answers dryly. — "the words"', 'told', 3101, 3101);

-- H3: a ruled receiver-variant that renders flavor text instead of quoting the words at all —
--     demonstrably NOT present as the complete utterance, so it must stay unset.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status,
                         accepted_at, visibility_scope, origin, payload)
VALUES ('f7900000-0000-0000-0000-000000000003', :'w'::uuid, 'Communicated', 'H says something curt.', 3102, 0,
        'accepted', now(), 'private', 'ruling', jsonb_build_object('spoken','the real words'));
INSERT INTO perception_record (perception_id, world_id, holder_id, source_event_id, content,
                               epistemic_type, acquired_tick, valid_tick) VALUES
 ('f7910000-0000-0000-0000-000000000003', :'w'::uuid, :'i'::uuid, 'f7900000-0000-0000-0000-000000000003',
  'She seems upset, though her exact words do not carry.', 'direct', 3102, 3102);

SELECT fn_backfill_perception_spoken();

SELECT is((SELECT spoken FROM perception_record WHERE perception_id='f7910000-0000-0000-0000-000000000001'),
          'Exact quote', '(H1) exact-content historical row is backfilled');
SELECT is((SELECT spoken FROM perception_record WHERE perception_id='f7910000-0000-0000-0000-000000000002'),
          'the words', '(H2) exact-quoted-utterance historical row is backfilled');
SELECT is((SELECT spoken FROM perception_record WHERE perception_id='f7910000-0000-0000-0000-000000000003'),
          NULL, '(H3) a ruled receiver-variant that never quoted the words stays unset — not merely attached-to-speech'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- I. fn_unheard_names vs fn_unearned_names — a literally-heard word is exempted from the more
--    permissive belt without ever being learned; fn_perceived_speech backs it with the same row.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

SELECT ok(
  EXISTS(SELECT 1 FROM fn_unearned_names(:'w'::uuid, :'e'::uuid) WHERE canonical_name = 'Jonas-124'),
  '(I1) fn_unearned_names still guards the name — E never learned WHO Jonas-124 is'
);
SELECT ok(
  NOT EXISTS(SELECT 1 FROM fn_unheard_names(:'w'::uuid, :'e'::uuid) WHERE canonical_name = 'Jonas-124'),
  '(I2) fn_unheard_names exempts it — E literally heard the word spoken to her'
);
SELECT ok(
  EXISTS(SELECT speaker_id, spoken FROM fn_perceived_speech(:'w'::uuid, :'e'::uuid, 0)
         WHERE speaker_id = :'a'::uuid AND spoken = 'Jonas-124 left.')
  AND (SELECT content FROM perception_record WHERE world_id=:'w'::uuid AND holder_id=:'e'::uuid
         AND source_event_id=(SELECT event_id FROM canon_event WHERE world_id=:'w'::uuid AND in_world_tick=3001))
      = 'a name mentioned in passing leaves. — "Jonas-124 left."',
  '(I3) account prose hides an unknown identity while the holder''s heard name remains available to quote'
);
SELECT ok(
  EXISTS(SELECT 1 FROM fn_unheard_names(:'w'::uuid, :'k'::uuid) WHERE canonical_name = 'Jonas-124'),
  '(I4) negative control: K never heard anything, so the more permissive guard still applies to her'
);

-- ══════════════════════════════════════════════════════════════════════════════════════════════
-- J. Direct shared-writer rejection and hidden handling, independent of the commit-door gates.
-- ══════════════════════════════════════════════════════════════════════════════════════════════

-- J1: hidden (payload.visible=false), no speech_perception at all — the function's OWN explicit
--     check, not merely a benefit of apply_ruled_event never calling it in this case.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status,
                         accepted_at, visibility_scope, origin, payload) VALUES
 ('f7900000-0000-0000-0000-000000000010', :'w'::uuid, 'Communicated', 'H whispers off the record.', 3200, 0,
  'accepted', now(), 'private', 'ruling', jsonb_build_object('visible', false));
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('f7900000-0000-0000-0000-000000000010', :'h'::uuid, 'actor', 'speaker');

SELECT is(
  (SELECT fn_apply_speech_perception('f7900000-0000-0000-0000-000000000010'::uuid, 'H whispers off the record.', '{}'::jsonb)),
  0, '(J1) called directly, a hidden event returns 0 with no exception — an explicit, self-contained check'
);
SELECT is(
  (SELECT count(*)::int FROM perception_record WHERE source_event_id='f7900000-0000-0000-0000-000000000010'),
  0, '(J1b) ...and writes nothing, including no speaker perception'
);

-- J2: an empty judgment object is not an accepted judgment. SQL NULL must not turn a missing
-- listeners field into successful speaker-only application through the shared writer.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status,
                         accepted_at, visibility_scope, origin, payload) VALUES
 ('f7900000-0000-0000-0000-000000000011', :'w'::uuid, 'Communicated', 'A speaks with no judgment attached.', 3201, 0,
  'accepted', now(), 'private', 'freeform', '{"speech_perception":{}}'::jsonb);
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('f7900000-0000-0000-0000-000000000011', :'a'::uuid, 'actor', 'speaker');

SELECT throws_ok(
  $$ SELECT fn_apply_speech_perception('f7900000-0000-0000-0000-000000000011'::uuid, 'A speaks with no judgment attached.', '{}'::jsonb) $$,
  NULL, NULL,
  '(J2) a visible event with an empty judgment RAISEs — no speaker-only success'
);

SELECT * FROM finish();
ROLLBACK;
