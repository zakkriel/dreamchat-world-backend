-- Phase 0B (chunk-2): I-4 causal acyclicity + bundle topology immutability.
-- Seren mini-scenario, TEST-TRANSACTION-ONLY (BEGIN/ROLLBACK) — no standing rows,
-- distinct world 2222… so 0A (world 1111…) is untouched. Events only; no state_mutation,
-- no perception_record => zero projection writes. Spec: docs/design/2026-06-11-phase-0B-causal-bundle-regression-design.md.
BEGIN;
SELECT plan(11);

-- Seren accepted events (durable refs for bundle inputs/effects; G3). Distinct ticks satisfy
-- uq_ce_accepted_order (unique world,tick,beat among accepted). No registry/mutations needed.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin) VALUES
 ('20000000-0000-0000-0000-00000000000a','22222222-2222-2222-2222-222222222222','move','A',10,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000b','22222222-2222-2222-2222-222222222222','move','B',11,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000c','22222222-2222-2222-2222-222222222222','move','C',12,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000d','22222222-2222-2222-2222-222222222222','move','D',13,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000e','22222222-2222-2222-2222-222222222222','move','E',14,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-00000000000f','22222222-2222-2222-2222-222222222222','move','F',15,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-000000000020','22222222-2222-2222-2222-222222222222','move','G',20,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-000000000021','22222222-2222-2222-2222-222222222222','move','H',21,0,'accepted',now(),'public','fast_path'),
 ('20000000-0000-0000-0000-000000000022','22222222-2222-2222-2222-222222222222','move','I',22,0,'accepted',now(),'public','fast_path');

-- (1) Conjunctive: A AND B caused C — one bundle, two necessary inputs (ADR-007).
SELECT lives_ok($$
  DO $do$ BEGIN
    INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
    VALUES ('b0000000-0000-0000-0000-0000000000c0','22222222-2222-2222-2222-222222222222',
            '20000000-0000-0000-0000-00000000000c','event','conjunctive','MANUAL_0B','valid');
    INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role, necessity) VALUES
     ('b0000000-0000-0000-0000-0000000000c0','20000000-0000-0000-0000-00000000000a','event','enabler',true),
     ('b0000000-0000-0000-0000-0000000000c0','20000000-0000-0000-0000-00000000000b','event','trigger',true);
  END $do$;
$$, 'conjunctive A AND B -> C accepted (acyclic)');

-- (2) Disjunctive: D OR E caused F — two bundles, same effect, one input each (ADR-007).
SELECT lives_ok($$
  DO $do$ BEGIN
    INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status) VALUES
     ('b0000000-0000-0000-0000-0000000000d1','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-00000000000f','event','disjunctive_member','MANUAL_0B','valid'),
     ('b0000000-0000-0000-0000-0000000000d2','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-00000000000f','event','disjunctive_member','MANUAL_0B','valid');
    INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role) VALUES
     ('b0000000-0000-0000-0000-0000000000d1','20000000-0000-0000-0000-00000000000d','event','trigger'),
     ('b0000000-0000-0000-0000-0000000000d2','20000000-0000-0000-0000-00000000000e','event','trigger');
  END $do$;
$$, 'disjunctive D OR E -> F accepted (two bundles, one effect)');

-- (3) G3: every event-kind bundle input references a durable canon_event.
SELECT ok( NOT EXISTS (
    SELECT 1 FROM causal_bundle_input cbi
    WHERE cbi.input_kind='event'
      AND NOT EXISTS (SELECT 1 FROM canon_event ce WHERE ce.event_id=cbi.input_ref)
  ), 'G3: all event-kind bundle inputs reference durable canon_event rows');

-- (4) Self-loop: effect == input is rejected (I-4).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-00000000005e','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-00000000000c','event','conjunctive','MANUAL_0B','valid');
SELECT throws_ok($$
  INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
  VALUES ('b0000000-0000-0000-0000-00000000005e','20000000-0000-0000-0000-00000000000c','event','trigger')
$$, 'P0001', NULL, 'I-4: self-loop (input == effect) rejected');

-- (5) 2-cycle: A AND B -> C exists, so C has ancestor A.
--     Adding a bundle "C caused A" closes A -> C -> A. Rejected (I-4). NEGATIVE CONTROL.
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-00000000002c','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-00000000000a','event','conjunctive','MANUAL_0B','valid');
SELECT throws_ok($$
  INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
  VALUES ('b0000000-0000-0000-0000-00000000002c','20000000-0000-0000-0000-00000000000c','event','trigger')
$$, 'P0001', NULL, 'I-4: edge C->A closing a 2-cycle (A->C exists) is rejected');

-- (6) Valid chain G -> H -> I accepted.
SELECT lives_ok($$
  DO $do$ BEGIN
    INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status) VALUES
     ('b0000000-0000-0000-0000-0000000000a1','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-000000000021','event','conjunctive','MANUAL_0B','valid'),
     ('b0000000-0000-0000-0000-0000000000a2','22222222-2222-2222-2222-222222222222','20000000-0000-0000-0000-000000000022','event','conjunctive','MANUAL_0B','valid');
    INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role) VALUES
     ('b0000000-0000-0000-0000-0000000000a1','20000000-0000-0000-0000-000000000020','event','trigger'),
     ('b0000000-0000-0000-0000-0000000000a2','20000000-0000-0000-0000-000000000021','event','trigger');
  END $do$;
$$, 'chain G -> H -> I accepted (acyclic)');

-- (7) Closing the chain: "I caused G" (G ->* I exists) is rejected (I-4).
INSERT INTO causal_bundle (bundle_id, world_id, effect_ref, effect_kind, semantics, template_id, status)
VALUES ('b0000000-0000-0000-0000-0000000000a3','22222222-2222-2222-2222-222222222222',
        '20000000-0000-0000-0000-000000000020','event','conjunctive','MANUAL_0B','valid');
SELECT throws_ok($$
  INSERT INTO causal_bundle_input (bundle_id, input_ref, input_kind, role)
  VALUES ('b0000000-0000-0000-0000-0000000000a3','20000000-0000-0000-0000-000000000022','event','trigger')
$$, 'P0001', NULL, 'I-4: edge I->G closing the G->H->I chain is rejected');

-- (8) IMMUTABLE-BUNDLE-TOPOLOGY (was 0B Rider A): effect_ref is immutable (append-only; only {status} may change).
SELECT throws_ok($$
  UPDATE causal_bundle SET effect_ref='20000000-0000-0000-0000-00000000000b'
  WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'P0001', NULL, 'IMMUTABLE-BUNDLE-TOPOLOGY: UPDATE of effect_ref on causal_bundle rejected (append-only)');

-- (9) IMMUTABLE-BUNDLE-TOPOLOGY: status alone may change.
SELECT lives_ok($$
  UPDATE causal_bundle SET status='invalidated'
  WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'IMMUTABLE-BUNDLE-TOPOLOGY: UPDATE of status alone is permitted');

-- (10) ADR-006: bundle inputs cannot be deleted.
SELECT throws_ok($$
  DELETE FROM causal_bundle_input WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'P0001', NULL, 'ADR-006: DELETE on causal_bundle_input rejected');

-- (11) ADR-006: bundles cannot be deleted.
SELECT throws_ok($$
  DELETE FROM causal_bundle WHERE bundle_id='b0000000-0000-0000-0000-0000000000c0'
$$, 'P0001', NULL, 'ADR-006: DELETE on causal_bundle rejected');

SELECT * FROM finish();
ROLLBACK;
