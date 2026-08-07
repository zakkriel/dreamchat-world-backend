BEGIN;
SELECT plan(4);

-- (a) a world with nothing at all is at tick 0, not NULL — the caller adds 1 and starts at 1.
SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 0::bigint,
  '(a) an empty world is at 0, never NULL');

-- A journey row alone moves the clock: this is the whole point — quiet legs commit nothing.
INSERT INTO journey (journey_id, world_id, actor_id, kind, threshold, span_seconds,
                     legs_total, legs_done, started_tick, current_tick, status)
VALUES ('fe000000-0000-0000-0000-00000000000a','fe000000-ffff-0000-0000-000000000000',
        'fe000000-0000-0000-0000-00000000000b','wait','{"kind":"tick","at":900}'::jsonb,
        900, 5, 2, 100, 460, 'active');

SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 460::bigint,
  '(b) a journey mid-flight carries the clock even though it has committed nothing');

-- (c) the later of the two wins — a canon event past the journey raises it.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('fe000000-0000-0000-0000-00000000000c','fe000000-ffff-0000-0000-000000000000',
        'AttributeChanged','probe',900,0,'accepted',now(),'public','freeform');
SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 900::bigint,
  '(c) canon past the journey wins — the clock is the later of the two');

-- (d) an ENDED journey still holds its tick: time must never rewind when a journey stops.
UPDATE journey SET status='ended', current_tick=1500
 WHERE journey_id='fe000000-0000-0000-0000-00000000000a';
SELECT is(fn_world_now('fe000000-ffff-0000-0000-000000000000'), 1500::bigint,
  '(d) an ended journey still holds the clock forward — time never rewinds');

SELECT * FROM finish();
ROLLBACK;
