BEGIN;
SELECT plan(4);

SET ROLE app_reader;
SELECT throws_ok(
  $$ INSERT INTO actor_state (entity_id, world_id)
     VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111') $$,
  '42501', NULL, 'app_reader INSERT into actor_state is denied (I-7)');
RESET ROLE;

SET ROLE app_reader;
SELECT lives_ok( $$ SELECT count(*) FROM actor_state $$, 'app_reader may SELECT projections');
RESET ROLE;

SET ROLE app_reader;
SELECT throws_ok( $$ SELECT apply_mutation(NULL::state_mutation) $$, '42501', NULL,
  'app_reader cannot EXECUTE apply_mutation (I-7 function hardening)');
SELECT throws_ok( $$ SELECT replay_0A() $$, '42501', NULL,
  'app_reader cannot EXECUTE replay_0A (I-7 function hardening)');
RESET ROLE;

SELECT * FROM finish();
ROLLBACK;
