BEGIN;
SELECT plan(1);
SELECT ok( true, 'pgTAP harness is wired and pg_prove can run a test' );
SELECT * FROM finish();
ROLLBACK;
