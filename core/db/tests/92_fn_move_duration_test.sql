BEGIN;
SELECT plan(3);
-- Duration is INDEPENDENT, hand-recorded fixture data (D-11; §11). Engine-assigned, never model
-- (§10 Q1 guardrail). UUID signatures: any two distinct location uuids cost 5; same uuid = 0.
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111',
          'e5ffffff-0000-0000-0000-000000000010'::uuid,
          'e5ffffff-0000-0000-0000-000000000011'::uuid)::int, 5,
          'tavern-uuid→square-uuid costs 5 ticks');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111',
          'e5ffffff-0000-0000-0000-000000000011'::uuid,
          'e5ffffff-0000-0000-0000-000000000010'::uuid)::int, 5,
          'symmetric: square-uuid→tavern-uuid also 5');
SELECT is(fn_move_duration('11111111-1111-1111-1111-111111111111',
          'e5ffffff-0000-0000-0000-000000000010'::uuid,
          'e5ffffff-0000-0000-0000-000000000010'::uuid)::int, 0,
          'same place = 0 ticks');
SELECT * FROM finish();
ROLLBACK;
