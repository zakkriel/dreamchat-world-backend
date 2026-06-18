BEGIN;
SELECT plan(2);

-- Generator determinism (the "perceptions replay" guarantee under the durable-canon posture). The
-- SAME move beat is run TWICE from IDENTICAL co-presence state — Player@'tavern', Jonas@'vault' — with
-- a FRESH SETUP between the runs (Player is re-placed at 'tavern' before run 2). It is NOT two calls on
-- accumulated state (the test-96 trap): run 1 leaves Player at 'vault', so run 2 MUST re-place them.
-- 'vault' is a label NO seed/noise actor ever occupies, so co-presence at the destination is exactly
-- {Jonas} regardless of the seed's noise positions (seed-independent, not fragile). A MOVE is used (not
-- a say) precisely so the discovery perception's about-subject (perception_subject) is exercised.
-- The compared set keeps EVERY semantic field — holder_id, epistemic_type, content, the about-subject,
-- and the WITHIN-BEAT relative tick (acquired_tick - start_tick, which is deterministic) — and strips
-- ONLY the non-semantic volatiles: perception_id and the absolute start band (deliberately varied).

-- Jonas → 'vault' (stays put for both runs; 'vault' is off the noise map)
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e5000000-0000-0000-0000-000000000080','11111111-1111-1111-1111-111111111111','move','J→vault',1200,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('e5000000-0000-0000-0000-000000000080','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000080','cccccccc-cccc-cccc-cccc-cccccccccccc','actor','attrs.location_id', to_jsonb('vault'::text),1200,0);

-- RUN 1 — fresh setup Player→'tavern', then the beat [move to vault] at start_tick 1300.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e5000000-0000-0000-0000-000000000081','11111111-1111-1111-1111-111111111111','move','P→tavern (run1 setup)',1210,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('e5000000-0000-0000-0000-000000000081','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000081','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),1210,0);
SELECT apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  '[{"type":"move","to":"vault"}]'::jsonb, 1300, 100, 'fast_path');

-- FRESH SETUP between runs — re-place Player at 'tavern' (run 1 left them at 'vault'); Jonas stays at
-- 'vault'. Run 2's starting co-presence is now identical to run 1's.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('e5000000-0000-0000-0000-000000000082','11111111-1111-1111-1111-111111111111','move','P→tavern (run2 setup)',1320,0,'accepted',now(),'public','fast_path');
INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
VALUES ('e5000000-0000-0000-0000-000000000082','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','instigator');
INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path, new_value, valid_from_tick, valid_from_seq)
VALUES ('11111111-1111-1111-1111-111111111111','e5000000-0000-0000-0000-000000000082','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','actor','attrs.location_id', to_jsonb('tavern'::text),1320,0);
SELECT apply_beat('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  '[{"type":"move","to":"vault"}]'::jsonb, 1350, 100, 'fast_path');

-- COMPARE: keep holder_id, epistemic_type, content, the about-subject, AND the within-beat relative
-- tick; strip ONLY perception_id and the absolute band. LEFT JOIN subject (own-move has none → 'none').
SELECT set_eq(
  $$ SELECT pr.holder_id, pr.epistemic_type, pr.content,
            pr.acquired_tick - 1300 AS rel_tick,
            COALESCE(ps.entity_id::text,'none') AS about
     FROM perception_record pr
     JOIN canon_event ce ON ce.event_id = pr.source_event_id
     LEFT JOIN perception_subject ps ON ps.perception_id = pr.perception_id
     WHERE ce.in_world_tick >= 1300 AND ce.in_world_tick < 1320 $$,
  $$ SELECT pr.holder_id, pr.epistemic_type, pr.content,
            pr.acquired_tick - 1350 AS rel_tick,
            COALESCE(ps.entity_id::text,'none') AS about
     FROM perception_record pr
     JOIN canon_event ce ON ce.event_id = pr.source_event_id
     LEFT JOIN perception_subject ps ON ps.perception_id = pr.perception_id
     WHERE ce.in_world_tick >= 1350 AND ce.in_world_tick < 1400 $$,
  'generator deterministic: identical state + beat → identical (holder, type, content, rel_tick, subject)');
SELECT is((SELECT count(*) FROM perception_record pr JOIN canon_event ce ON ce.event_id = pr.source_event_id
           WHERE ce.in_world_tick >= 1300 AND ce.in_world_tick < 1320)::int,
          2, 'non-vacuous: the move produced 2 perceptions (own-move + discovery-about-Jonas)');

SELECT * FROM finish();
ROLLBACK;
