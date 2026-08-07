BEGIN;
SELECT plan(4);

-- (a) a short hop lands at the low end of the band
SELECT is(fn_journey_legs('ff000000-ffff-0000-0000-000000000000', 900::bigint), 5,
  '(a) a 15-minute span is 5 legs');

-- (b) a multi-day haul lands at the high end
SELECT is(fn_journey_legs('ff000000-ffff-0000-0000-000000000000', 864000::bigint), 10,
  '(b) a ten-day span is 10 legs');

-- (c) EVERY span stays inside the 5..10 band — the promise the player experiences
SELECT ok(
  (SELECT bool_and(n BETWEEN 5 AND 10) FROM (
     SELECT fn_journey_legs('ff000000-ffff-0000-0000-000000000000', s) AS n
     FROM unnest(ARRAY[1,60,900,3600,86400,864000,31536000]::bigint[]) AS s) t),
  '(c) every span from a second to a year yields between 5 and 10 legs');

-- (d) it is data: a per-world row overrides the fallback
INSERT INTO journey_legs_band (world_id, max_span_seconds, legs)
VALUES ('ff000000-ffff-0000-0000-000000000000', 900, 6);
SELECT is(fn_journey_legs('ff000000-ffff-0000-0000-000000000000', 900::bigint), 6,
  '(d) a per-world band row overrides the built-in fallback');

SELECT * FROM finish();
ROLLBACK;
