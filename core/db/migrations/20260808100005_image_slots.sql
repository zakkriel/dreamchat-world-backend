-- migrate:up

-- The world backend's half of the image storage split, mandated by the Image Platform's own
-- integration quickstart (§5, "Storage requirement"): the asset row's `visual_identity_id` is NULL on
-- the single-image path — the JOB carries the identity, the ASSET does not — and there is no
-- "give me the current asset for identity X" endpoint. So the mapping has to live here, or nobody has
-- it. Their suggested split, adopted verbatim:
--
--   Platform  — assets, provenance, cost, tiers.
--   Us        — entity → visual_identity_id, the last known asset_id per slot (so we can answer the
--               frontend WITHOUT calling them), and job_id while one is in flight.
--   Frontend  — nothing durable; it receives {asset_id | null} and treats every URL as expiring.
--
-- ── A SLOT, not an image ────────────────────────────────────────────────────────────────────────
-- One row per (world, entity): "the picture of this thing". `asset_id` is the LAST KNOWN ready asset,
-- which is exactly what lets a read answer instantly and offline. Reuse is the platform's default —
-- re-issuing the same generation is a zero-cost cache hit returning the SAME asset_id — so this is a
-- cache we never have to invalidate on a perception change. What changes when understanding changes
-- is the TEXT, not the face.
--
-- ── Nothing here is canon ───────────────────────────────────────────────────────────────────────
-- Not a projection, not perception-bound, and deliberately NOT in the canon spine: an image is
-- illustrative and does not create world truth (platform doc 09 §6 — "images support immersion; they
-- do not automatically create canon"). No event is written when a portrait appears. A row here is
-- operational bookkeeping about an external service, which is why it carries wall-clock timestamps
-- freely — those are telemetry, never in-world time (B-5).
--
-- ── issued_at is pinned, and that is load-bearing ───────────────────────────────────────────────
-- Their idempotency key is bound to a hash of the WHOLE request body, and the governance envelope
-- carries `issued_at`, which moves every time you build one. Rebuilding an envelope on retry
-- therefore yields 409 idempotency_conflict — a trap they hit during their own verification run. So
-- the key AND the timestamp it was minted with are stored together: a retry replays the identical
-- body rather than composing a new one.
--
-- Freshness still applies (`GOVERNANCE_MAX_AGE`, default 24h), so a pinned envelope is not eternal:
-- a request abandoned for longer than that must take a NEW key with a NEW issued_at, never a fresh
-- timestamp under the old key.

CREATE TABLE image_slot (
  world_id         uuid NOT NULL,
  owner_kind       text NOT NULL CHECK (owner_kind IN ('actor','location','artifact')),
  owner_id         uuid NOT NULL,

  -- Platform ids are opaque TEXT on their side (vi_…, asset_…, job_…); they are theirs to shape.
  visual_identity_id text,
  asset_id           text,   -- last known READY asset; NULL until the first one completes
  job_id             text,   -- set while a generation is in flight, cleared on a terminal status

  -- the pinned envelope for the in-flight job (see above)
  idempotency_key text,
  issued_at       timestamptz,

  -- last terminal failure, kept so a slot can explain itself instead of just staying blank
  last_error   text,
  updated_at   timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (world_id, owner_kind, owner_id)
);

-- The drain looks for work by state, not by entity: "which slots are in flight" and "which have
-- nothing yet". Partial indexes keep both cheap and stay small, since most slots are settled.
CREATE INDEX idx_image_slot_in_flight ON image_slot (world_id) WHERE job_id IS NOT NULL;
CREATE INDEX idx_image_slot_unfilled  ON image_slot (world_id) WHERE asset_id IS NULL AND job_id IS NULL;

GRANT SELECT ON image_slot TO app_reader;

-- fn_image_ref: the projection the frontend actually reads — the last known asset for one entity, or
-- NULL. NULL is the ordinary state, not an error: it means "no picture yet", and the frontend renders
-- its placeholder silhouette (D-8) and swaps in the image when a later read returns non-null, with no
-- re-request and no polling.
--
-- It returns an ASSET ID and a PATH, never a URL to the platform. Presigned download URLs expire in
-- ~15 minutes and must never be persisted (their §0 invariant: "IDs over the wire, URLs fetched on
-- demand"). The path points back at THIS service, which mints a fresh presigned URL per read — so a
-- payload can be cached, logged or re-rendered later without ever carrying a credential that rots.
CREATE FUNCTION fn_image_ref(p_world_id uuid, p_owner_kind text, p_owner_id uuid) RETURNS json
LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN s.asset_id IS NULL THEN NULL
              ELSE json_build_object(
                     'schema_version', 'image_ref/1',
                     'asset_id',       s.asset_id,
                     'path',           '/worlds/' || p_world_id::text || '/images/' || s.asset_id
                   )
         END
    FROM image_slot s
   WHERE s.world_id = p_world_id AND s.owner_kind = p_owner_kind AND s.owner_id = p_owner_id;
$$;

-- migrate:down

DROP FUNCTION IF EXISTS fn_image_ref(uuid, text, uuid);
DROP TABLE IF EXISTS image_slot;
