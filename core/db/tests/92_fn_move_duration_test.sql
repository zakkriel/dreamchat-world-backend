BEGIN;
SELECT plan(3);
-- Duration is INDEPENDENT, hand-recorded fixture data (D-11; §11). Engine-assigned, never model
-- (§10 Q1 guardrail). Thin slice: a symmetric tavern↔square cost; same-place = 0; say handled = 0.
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111','tavern','square')::int, 5,
          'tavern→square costs 5 ticks');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111','square','tavern')::int, 5,
          'symmetric: square→tavern also 5');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111','tavern','tavern')::int, 0,
          'no move, no time');
SELECT * FROM finish();
ROLLBACK;
