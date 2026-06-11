-- doc 13 §7 pass/fail checks, verbatim. Run by hand:
--   docker compose exec -T db psql -U postgres -d dreamchat -f /work/tests/helpers.sql -f /work/tests/checks_0A.sql
-- Every column must return TRUE.
SELECT count(*) = 0 AS i2_mutations_ok
FROM state_mutation sm
LEFT JOIN canon_event ce ON ce.event_id = sm.event_id AND ce.status='accepted'
WHERE ce.event_id IS NULL;

SELECT count(*) = 0 AS i2_perceptions_ok
FROM perception_record pr
LEFT JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.status='accepted'
WHERE ce.event_id IS NULL;

SELECT count(*) = 0 AS j_ignorant_ok
FROM perception_record WHERE holder_id = :'jonas_id' AND source_event_id = :'e1_id';

SELECT count(*) = 1 AS mara_knows_ok
FROM perception_record
WHERE holder_id = :'mara_id' AND source_event_id = :'e1_id'
  AND epistemic_type='told' AND invalid_tick IS NULL AND expired_at IS NULL;

SELECT count(*) = 1 AS mara_perception_survives_ok
FROM perception_record WHERE holder_id = :'mara_id' AND source_event_id = :'e1_id';

SELECT count(*)  >= 1 AS public_knowledge_ok
FROM perception_record WHERE source_event_id = :'e102_id' AND epistemic_type='public';

-- I-1 replay (single boolean). Wrapped so the truncate/rebuild does not persist.
BEGIN;
SELECT replay_0A() AS i1_replay_ok;
ROLLBACK;
