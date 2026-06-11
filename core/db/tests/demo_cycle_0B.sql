-- Phase 0B operator demo (by hand): insert a causality loop and watch the database refuse.
-- Run:  docker compose exec -T db psql -U postgres -d dreamchat -f /work/tests/demo_cycle_0B.sql
-- ON_ERROR_STOP is OFF so psql prints the rejection ERROR and proceeds to ROLLBACK.
-- Nothing persists. Excluded from the CI glob (filename is not *_test.sql).
\set ON_ERROR_STOP off
BEGIN;

INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('20000000-0000-0000-0000-0000000000f1','22222222-2222-2222-2222-222222222222','move','demo A',1,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-0000000000f2','22222222-2222-2222-2222-222222222222','move','demo B',2,0,'accepted',now(),'public','fast_path');

-- Bundle 1: A causes B  (valid).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-0000000000f1','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-0000000000f2','event','conjunctive','MANUAL_0B','valid');
INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
VALUES ('b0000000-0000-0000-0000-0000000000f1','20000000-0000-0000-0000-0000000000f1','event','trigger');

-- Bundle 2: B causes A  (CLOSES THE LOOP — the next input insert must be REFUSED).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-0000000000f2','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-0000000000f1','event','conjunctive','MANUAL_0B','valid');
\echo '>>> Attempting to close the loop (B causes A). Expect: ERROR ... causal cycle rejected (I-4)'
INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
VALUES ('b0000000-0000-0000-0000-0000000000f2','20000000-0000-0000-0000-0000000000f2','event','trigger');

ROLLBACK;
