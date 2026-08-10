-- migrate:up

-- The founder's approved taglines, applied to the worlds that ALREADY EXIST.
--
-- ── WHY A MIGRATION CARRIES CONTENT, WHICH IT NORMALLY MUST NOT ─────────────────────────────────
-- The taglines belong in the seeds, and that is where they live. But the seeds refuse to run against
-- a populated database on purpose — `seed_mara_0A targets a CLEAN DB only` — so a seed-only change
-- reaches a fresh world and never reaches the shared one everybody is driving. That is precisely the
-- failure SPEC-031 recorded: the interruption tuning needed BOTH `seed_world_defaults` (so new
-- worlds are born with it) and an `UPDATE` (because the seeds insert ON CONFLICT DO NOTHING and
-- would otherwise have left the only world anyone plays untouched). *A change that lands green and
-- changes nothing is the failure mode here.*
--
-- So: the seeds are the source of truth for authored fiction, and this migration is the one-time
-- reach into the two worlds that predate it. Deliberately NOT a general mechanism — it names two
-- known ids and nothing else, and no future world is touched by it.
--
-- Guarded by `WHERE tagline IS NULL`, so it can never overwrite a line somebody has since edited or
-- re-authored. Re-running it is a no-op, which is what makes it safe to keep in the ledger forever.
UPDATE world
   SET tagline = 'A harbor town where everyone is owed something, and the tide keeps the ledger.'
 WHERE world_id = '22222222-2222-2222-2222-222222222222' AND tagline IS NULL;

UPDATE world
   SET tagline = 'A test world. Two people, one room, and every rule watching.'
 WHERE world_id = '11111111-1111-1111-1111-111111111111' AND tagline IS NULL;

-- migrate:down

UPDATE world SET tagline = NULL
 WHERE world_id IN ('22222222-2222-2222-2222-222222222222',
                    '11111111-1111-1111-1111-111111111111');
