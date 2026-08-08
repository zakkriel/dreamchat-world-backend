-- migrate:up

-- An image reference reaches the frontend as a PROJECTION FIELD on payloads it already reads —
-- not through an async channel. `image` is plain data: NULL until a picture exists, then an
-- {asset_id, path} pair. The frontend renders its placeholder silhouette on NULL and swaps the
-- picture in on a later read, with no subscription, no polling and no re-request.
--
-- That is deliberate rather than convenient. The async channel (image.ready and its siblings) stays
-- unbuilt: standing one up so it can announce this single field would be a subsystem whose only
-- message is one a read already carries. When a channel exists for reasons that need it, an image
-- can ride it as a latency hint — exactly the status the image platform gives its own webhooks.
--
-- ── WHY THE VERSION MOVES ───────────────────────────────────────────────────────────────────────
-- actor_page's payload is `additionalProperties: false`, and the frontend pins the schema_version
-- exactly and fails the load on a mismatch. Adding a field is therefore a BREAKING change however
-- additive it looks, and the honest signal is the version: actor_page/1 → actor_page/2. Clean
-- cutover, no alias — the same protocol beat_frame/2 used, and the third time the frontend has run
-- it, so it is a known quantity rather than an adventure.
--
-- Only fn_actor_page moves here. location_page and artifact_page keep /1 and gain no field: the PoC
-- scope is one portrait per entity and one background per scene, and bumping payloads that have
-- nothing to put in the field would cost the frontend two more re-pins for nothing. image_slot
-- already keys on owner_kind ('actor' | 'location' | 'artifact'), so those surfaces are a field and
-- a version bump away whenever there is something to show.

CREATE OR REPLACE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_actor_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'actor_page/2',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id',                         p_actor_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      -- Perception rows do not carry a structured role taxonomy for actors.
      'perceived_role',             NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_actor_id),
      'last_known_status',          fn_compendium_latest_fact(p_world_id, p_viewer_id, p_actor_id),
      'known_artifacts',            fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['artifact']),
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['actor','location','artifact']),
      -- NULL until a portrait exists. Never a presigned URL: fn_image_ref emits an asset id and a
      -- path back to this service, which mints a fresh short-lived URL per read, so a payload can be
      -- cached or logged without carrying a credential that expires in fifteen minutes.
      --
      -- NOT perception-scoped, and that is a real decision. A portrait is of the ENTITY, not of the
      -- viewer's opinion of it: two viewers who know an actor by different names still see the same
      -- face, exactly as they would in the room. The wall governs what a viewer KNOWS — names,
      -- facts, synthesis — and this page already renders every one of those through the viewer's own
      -- perception. A picture of a person standing in front of you is not a secret (B-1 unaffected;
      -- the existence gate above still decides whether this page may be seen at all).
      'image',                      fn_image_ref(p_world_id, 'actor', p_actor_id)
    )
  ) END;
$$;

-- migrate:down

CREATE OR REPLACE FUNCTION fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid)
RETURNS json LANGUAGE sql STABLE AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_actor_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'actor_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id',                         p_actor_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      'perceived_role',             NULL,
      'current_synthesis',          fn_compendium_current_synthesis(p_world_id, p_viewer_id, p_actor_id),
      'last_known_status',          fn_compendium_latest_fact(p_world_id, p_viewer_id, p_actor_id),
      'known_artifacts',            fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['artifact']),
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               fn_compendium_related_entities(p_world_id, p_viewer_id, p_actor_id, ARRAY['actor','location','artifact'])
    )
  ) END;
$$;
