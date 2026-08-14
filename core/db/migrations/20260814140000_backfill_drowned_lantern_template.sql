-- migrate:up

-- Backfill the canonical Drowned Lantern's template lineage.
--
-- 20260813142100 introduced world.template_key and set it inside
-- fn_instantiate_drowned_lantern, so every world seeded or refreshed AFTER that migration carries
-- its lineage. The world that matters most does not: 22222222-… was seeded long before the template
-- mechanism existed, so in any long-lived database (production, specifically) its template_key is
-- NULL — and POST /worlds/{id}/refresh answers 404 "world has no template" on exactly the world
-- people actually play. A feature that works only on a freshly reset database is not shipped.
--
-- The claim this makes is true rather than convenient: the canonical world IS an instance of that
-- template. The seed authored it, and the template function now holds that same content verbatim.
--
-- Guarded by `template_key IS NULL`, so it is idempotent and a no-op on a freshly seeded database
-- where fn_instantiate_drowned_lantern already stamped the row.
--
-- The Mara 0A fixture (11111111-…) is deliberately NOT backfilled: it comes from seed_mara_0A.sql,
-- there is no template function for it, and refresh correctly reports that it cannot be refreshed.

UPDATE world
   SET template_key = 'drowned_lantern'
 WHERE world_id = '22222222-2222-2222-2222-222222222222'
   AND template_key IS NULL;

-- migrate:down

UPDATE world
   SET template_key = NULL
 WHERE world_id = '22222222-2222-2222-2222-222222222222'
   AND template_key = 'drowned_lantern';
