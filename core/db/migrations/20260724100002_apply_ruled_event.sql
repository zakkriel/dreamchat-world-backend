-- migrate:up

-- ─── 0. Extend canon_event.origin to include 'ruling' ────────────────────────
-- Ruled events carry p_origin='ruling' to distinguish them from ATTEMPT-path commits.

ALTER TABLE canon_event
  DROP CONSTRAINT IF EXISTS canon_event_origin_check;

ALTER TABLE canon_event
  ADD CONSTRAINT canon_event_origin_check
    CHECK (origin IN ('fast_path','template','freeform','threshold','backstage','compensation','ruling'));

-- ─── 1. apply_ruled_event — sibling of apply_event for RULED OUTCOMES ────────
--
-- Accepts one RuledEventV2 as snake_case jsonb (type, actor_id, truth,
-- appearance?, visible?, receiver_variants?[{receiver_id,text}], + typed slots
-- to_location_id / listener_id / content / object_id / dest_kind / dest_id /
-- target_id / grantee_id).
--
-- STRUCTURAL FLOOR (twin of apply_event floor — keep in sync):
--   Every type:    actor_id exists as kind='actor' in entity_registry.
--   ActorMoved:    to_location_id exists in entity_registry.
--   Communicated:  listener_id co-located with actor via fn_actors_at.
--   ObjectRelocated: object_id + dest_id exist in entity_registry.
--   OwnershipAccessChanged | EntityDestroyed | AttributeChanged: target_id exists.
--   EntityCreated: no target check (actor-only).
-- Floor fail → {"event_id": null, "halt_reason": "gate_reject"}, NOTHING written.
--
-- CANON COMMIT: summary = truth (CANON NEVER LIES).
-- PERCEPTIONS:  visible=false (explicit) → zero perceptions.
--               else receivers = fn_actors_at(actor's location) UNION {actor}
--               each receiver content:
--                 receiver_variants match → variant text
--                 else appearance (if non-empty)
--                 else truth
--               each perception gets perception_subject rows for all participant ids.
--
-- Floor extraction choice: the floor logic is DUPLICATED (not extracted into a
-- shared helper) because apply_event's floor is inlined in its body and
-- extracting it would require a new function signature or refactoring apply_event
-- itself — invasive for a function with existing tests. The duplication is
-- commented "twin of apply_event floor — keep in sync" per the brief's guidance.

CREATE OR REPLACE FUNCTION apply_ruled_event(
  p_world_id  uuid,
  p_ruled     jsonb,
  p_tick      bigint,
  p_seq       int,
  p_origin    text
) RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  ev_type    text;
  actor_id   uuid;
  ev_id      uuid;
  listener   uuid;
  to_loc     uuid;
  object_eid uuid;
  dest_eid   uuid;
  target_eid uuid;
  here       uuid;
  vis_scope  text;
  truth_text text;
  appear_txt text;
  visible    boolean;
  -- perception fan-out
  receiver   uuid;
  recv_text  text;
  var_text   text;
  pid        uuid;
  -- participant ids for about-ness (perception_subject)
  participant_ids uuid[];
BEGIN
  ev_type  := p_ruled->>'type';
  actor_id := (p_ruled->>'actor_id')::uuid;
  truth_text := p_ruled->>'truth';
  appear_txt := NULLIF(TRIM(COALESCE(p_ruled->>'appearance', '')), '');
  visible    := CASE
    WHEN p_ruled ? 'visible' AND (p_ruled->>'visible') = 'false' THEN false
    ELSE true
  END;

  -- ── STRUCTURAL FLOOR (twin of apply_event floor — keep in sync) ─────────────
  -- Every type: actor must exist in entity_registry with kind='actor'.
  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = actor_id
      AND world_id  = p_world_id
      AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- Type-specific floor checks.
  IF ev_type = 'ActorMoved' THEN
    to_loc := (p_ruled->>'to_location_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry
      WHERE entity_id = to_loc AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_ruled->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = actor_id;
    IF NOT EXISTS (
      SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'ObjectRelocated' THEN
    object_eid := (p_ruled->>'object_id')::uuid;
    dest_eid   := (p_ruled->>'dest_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = object_eid AND world_id = p_world_id)
    OR NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = dest_eid  AND world_id = p_world_id)
    THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_ruled->>'target_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- No target check required (actor-only floor above covers it).
    NULL;

  ELSE
    -- Unknown type: reject.
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- ── COMMIT ──────────────────────────────────────────────────────────────────
  ev_id := gen_random_uuid();

  -- visibility_scope: private for Communicated, public otherwise.
  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  -- Canon summary = truth. CANON NEVER LIES.
  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, ev_type, truth_text,
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

  -- event_participant: Communicated → speaker (actor) + listener; all others → instigator.
  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, actor_id, 'actor', 'speaker'),
      (ev_id, listener, 'actor', 'listener');
    participant_ids := ARRAY[actor_id, listener];
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, actor_id, 'actor', 'instigator');
    participant_ids := ARRAY[actor_id];
  END IF;

  -- state_mutation: ActorMoved only — trigger projects it into actor_state.
  IF ev_type = 'ActorMoved' THEN
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id',
            to_jsonb(to_loc::text), p_tick, p_seq);
  END IF;

  -- ── PERCEPTION FAN-OUT ───────────────────────────────────────────────────────
  -- visible=false (explicit) → zero perceptions; no further work.
  IF NOT visible THEN
    RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
  END IF;

  -- Receivers = actors co-located with the ruled actor (fn_actors_at on actor's
  -- current location, NULL-safe) UNION the actor itself (actor already included
  -- when they are present at their own location; UNION deduplicates).
  --
  -- After ActorMoved, the actor's location is already updated in actor_state by
  -- the trigger above, so fn_actors_at returns the destination set. For non-move
  -- types the actor is still at their original location — both are correct behavior.
  SELECT (a.attrs->>'location_id')::uuid INTO here
    FROM actor_state a
    WHERE a.world_id = p_world_id AND a.entity_id = actor_id;

  FOR receiver IN
    SELECT entity_id FROM fn_actors_at(p_world_id, here)
    UNION
    SELECT actor_id
  LOOP
    -- Determine content for this receiver:
    -- 1. receiver_variants match → variant text
    -- 2. appearance (if non-empty) → appearance
    -- 3. truth
    var_text := NULL;
    IF p_ruled ? 'receiver_variants' THEN
      SELECT rv->>'text' INTO var_text
        FROM jsonb_array_elements(p_ruled->'receiver_variants') AS rv
        WHERE (rv->>'receiver_id')::uuid = receiver
        LIMIT 1;
    END IF;

    recv_text := COALESCE(var_text, appear_txt, truth_text);

    INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                   acquired_tick, valid_tick)
    VALUES (p_world_id, receiver, ev_id, recv_text, 'direct', p_tick, p_tick)
    RETURNING perception_id INTO pid;

    -- About-ness: perception_subject rows for ALL participant ids (engine-written,
    -- NOT relying on seed backfill).
    DECLARE
      part_id uuid;
    BEGIN
      FOREACH part_id IN ARRAY participant_ids LOOP
        INSERT INTO perception_subject (perception_id, entity_id, world_id)
          VALUES (pid, part_id, p_world_id);
      END LOOP;
    END;
  END LOOP;

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

-- ─── 2. apply_attribute_writes — batch state_mutation inserts for attribute writes
--
-- p_writes = jsonb array of {target_id, attribute, value, tier}.
-- For each element: INSERT state_mutation with provenance to p_provenance_event.
-- The existing trg_sm_project trigger projects each mutation into the appropriate
-- *_state table automatically.
-- NO tier validation — Go verdict owns tier checking before calling this function.
-- Returns count of rows written.

CREATE OR REPLACE FUNCTION apply_attribute_writes(
  p_world_id        uuid,
  p_writes          jsonb,
  p_provenance_event uuid,
  p_tick            bigint,
  p_seq             int
) RETURNS int
  LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  w        jsonb;
  target   uuid;
  ekind    text;
  n        int := 0;
BEGIN
  FOR w IN SELECT * FROM jsonb_array_elements(p_writes) LOOP
    target := (w->>'target_id')::uuid;

    -- Look up entity_kind from the registry (required by state_mutation schema).
    SELECT entity_kind INTO ekind
      FROM entity_registry
      WHERE entity_id = target AND world_id = p_world_id
      LIMIT 1;

    -- Skip writes for unknown entities (defensive; caller validates).
    IF ekind IS NULL THEN
      CONTINUE;
    END IF;

    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, p_provenance_event, target, ekind,
            'attrs.' || (w->>'attribute'),
            w->'value',
            p_tick, p_seq);

    n := n + 1;
  END LOOP;

  RETURN n;
END $$;

-- migrate:down

DROP FUNCTION IF EXISTS apply_ruled_event(uuid, jsonb, bigint, int, text);
DROP FUNCTION IF EXISTS apply_attribute_writes(uuid, jsonb, uuid, bigint, int);

-- Restore canon_event.origin constraint to the pre-ruling set.
ALTER TABLE canon_event
  DROP CONSTRAINT IF EXISTS canon_event_origin_check;

ALTER TABLE canon_event
  ADD CONSTRAINT canon_event_origin_check
    CHECK (origin IN ('fast_path','template','freeform','threshold','backstage','compensation'));
