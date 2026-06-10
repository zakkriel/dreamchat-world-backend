BEGIN;
SELECT plan(2);
SELECT has_role('maintainer', 'maintainer role exists');
SELECT has_role('app_reader', 'app_reader role exists');
SELECT * FROM finish();
ROLLBACK;
