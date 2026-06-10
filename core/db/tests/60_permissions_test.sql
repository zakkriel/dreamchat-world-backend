BEGIN;
SELECT plan(2);

SET ROLE app_reader;
SELECT throws_ok(
  $$ INSERT INTO actor_state (entity_id, world_id)
     VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa','11111111-1111-1111-1111-111111111111') $$,
  '42501', NULL, 'app_reader INSERT into actor_state is denied (I-7)');
RESET ROLE;

SET ROLE app_reader;
SELECT lives_ok( $$ SELECT count(*) FROM actor_state $$, 'app_reader may SELECT projections');
RESET ROLE;

SELECT * FROM finish();
ROLLBACK;
