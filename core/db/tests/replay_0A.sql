-- By-hand I-1 runner (exit gate). Wrapped so the truncate/rebuild does not persist.
BEGIN;
SELECT replay_0A() AS i1_replay_ok;
ROLLBACK;
