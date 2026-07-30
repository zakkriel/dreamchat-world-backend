-- migrate:up

-- Station F / Task 9 — ENTITY CREATION ON RESOLVE (founder-directed 2026-07-30: "when it resolves it
-- updates it all"). Until now the EntityCreated branch in BOTH commit twins was an EMPTY STUB (NULL): a
-- ruling could decide "a new thing exists" and NOTHING persisted it. This migration makes the branch
-- REAL. FINAL-action-contracts.md §8 (minting/three nets) + §5.4 (reuse-before-create); §7 (EntityCreated
-- = adjudicated intent, the create-write is provenance-guarded); FINAL-interaction-loop-PRD R3/R4 (doc-05
-- matcher: match → reuse the existing id; no match + true introduction → create, descriptor mandatory).
--
-- CLEAN SEPARATION (resolves the Task-6 deferral): EntityCreated creates entity INSTANCES (an
-- entity_registry row + its positioned state row); apply_mint (Task 6) creates typed VOCABULARY (movement
-- types / status modifiers). Different writers, no overlap — instance creation NEVER routes through
-- apply_mint, vocabulary NEVER routes through EntityCreated.
--
-- Shared helper (the Task-8 discipline — no twin drift): fn_apply_entity_created carries the whole write
-- so apply_event and apply_ruled_event call ONE function. EntityCreated is an ADJUDICATED type, so only
-- apply_ruled_event's branch is LIVE; apply_event's branch is kept in lockstep for twin parity.

-- ── fn_apply_entity_created: persist ONE created instance. REUSE-BEFORE-CREATE first (§5.4 / R3 doc-05
--    matcher against the authoritative registry — a superset of the slice's grounded entities): an
--    existing ACTIVE entity of the SAME (entity_kind, descriptor) → RETURN its id, no new row, no state
--    overwrite. The match is EXACT on (kind, descriptor) after case/whitespace normalization — pinned
--    naming (§5.1) is what makes exact match correct; a fuzzy match would be inventing judgment
--    (measurements-not-verdicts, forbidden). No match → a TRUE INTRODUCTION: INSERT an entity_registry
--    row (created_by_event = the ruling event id, §8 net 3; current_scene_id from new_attrs.location_id;
--    status active) + the initial POSITIONED state via one state_mutation per new_attrs key (replay-safe;
--    sm_project projects into <kind>_state). Genesis measurements (coordinates + Tier-1 size/weight/…) are
--    written here — "minting the new thing inside the contract shape" (§8), not an ad-hoc Tier-1 write.
--    Returns the id that now names the instance (created or reused). Callers pass a NON-EMPTY descriptor
--    (the twins gate-reject an empty one before calling).
CREATE FUNCTION public.fn_apply_entity_created(p_event_id uuid, p_world_id uuid, p_target_id uuid,
                                               p_kind text, p_name text, p_descriptor text, p_attrs jsonb)
RETURNS uuid
LANGUAGE plpgsql AS $$
DECLARE
  v_existing uuid;
  v_new      uuid;
  v_tick     bigint;
  v_seq      int;
  v_scene    uuid;
  k          text;
  v          jsonb;
BEGIN
  -- REUSE-BEFORE-CREATE (§5.4 / R3): match an existing entity before minting a new one.
  SELECT entity_id INTO v_existing
    FROM entity_registry
    WHERE world_id = p_world_id
      AND entity_kind = p_kind
      AND status = 'active'
      AND descriptor IS NOT NULL
      AND lower(btrim(descriptor)) = lower(btrim(p_descriptor))
    ORDER BY entity_id
    LIMIT 1;
  IF v_existing IS NOT NULL THEN
    RETURN v_existing;   -- the entity already exists — reuse its id, mint nothing
  END IF;

  -- TRUE INTRODUCTION: create the registry row, provenance-stamped.
  v_new   := COALESCE(p_target_id, gen_random_uuid());
  v_scene := NULLIF(p_attrs->>'location_id', '')::uuid;   -- actor/artifact carry a scene; a location has none
  INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name, descriptor,
                               current_scene_id, created_by_event, status)
  VALUES (v_new, p_world_id, p_kind, COALESCE(NULLIF(btrim(COALESCE(p_name,'')),''), p_descriptor),
          p_descriptor, v_scene, p_event_id, 'active');

  -- INITIAL POSITIONED STATE (replay-safe): one state_mutation per attr key. Tick/seq come from the
  -- committed create event (single source of truth), mirroring fn_apply_carry_change.
  SELECT in_world_tick, beat_seq INTO v_tick, v_seq FROM canon_event WHERE event_id = p_event_id;
  IF p_attrs IS NOT NULL AND jsonb_typeof(p_attrs) = 'object' THEN
    FOR k, v IN SELECT key, value FROM jsonb_each(p_attrs) LOOP
      INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                  new_value, valid_from_tick, valid_from_seq)
      VALUES (p_world_id, p_event_id, v_new, p_kind, 'attrs.' || k, v, v_tick, v_seq);
    END LOOP;
  END IF;

  RETURN v_new;
END $$;

-- ── apply_event: identical to 20260729100006 EXCEPT the EntityCreated branch is now REAL (twin parity;
--    the fast path never emits EntityCreated, so this branch is dormant but kept in lockstep).
CREATE OR REPLACE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type      text;
  ev_id        uuid;
  listener     uuid;
  to_target    uuid;   -- the move's target: ANY positioned entity (location | object | actor)
  v_scene      uuid;   -- resolved: the target's containing scene (a location resolves to itself)
  v_coord      jsonb;  -- resolved: the target's position — the mover's new coordinate
  object_eid   uuid;
  dest_eid     uuid;
  target_eid   uuid;
  here         uuid;
  vis_scope    text;
  final_type   text;
  v_old_holder uuid;   -- object's current carrier, read before the move (eager rule, §4)
BEGIN
  ev_type := p_attempt->>'type';

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = p_actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_attempt->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, p_actor_id, to_target) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_attempt->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
    IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'ObjectRelocated' THEN
    object_eid := (p_attempt->>'object_id')::uuid;
    dest_eid   := (p_attempt->>'dest_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = object_eid AND world_id = p_world_id)
    OR NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = dest_eid  AND world_id = p_world_id)
    THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_attempt->>'target_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- DESCRIPTOR MANDATORY (§8 / §7 provenance-guarded create): no descriptor ⇒ not a true introduction
    -- ⇒ gate_reject. Blocker-only: the reality-check/adjudication already happened; this is the commit
    -- clerk refusing an un-introduced create. (Twin of apply_ruled_event — the shared floor rule.)
    IF NULLIF(btrim(COALESCE(p_attempt->>'descriptor','')),'') IS NULL THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  IF p_legacy_types THEN
    final_type := CASE ev_type
      WHEN 'Communicated' THEN 'private_disclosure'
      WHEN 'ActorMoved'   THEN 'move'
      ELSE ev_type
    END;
  ELSE
    final_type := ev_type;
  END IF;

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, final_type, p_attempt->>'stated',
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, p_actor_id, 'actor', 'speaker'),
      (ev_id, listener,   'actor', 'listener');
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, p_actor_id, 'actor', 'instigator');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  -- EntityCreated: persist the new instance (registry row + positioned state) via the shared helper.
  -- Reuse-before-create is inside the helper. Provenance = ev_id (§8 net 3); one logged canon_event row
  -- above (net 2); the ruling already passed the reality check (net 1).
  IF ev_type = 'EntityCreated' THEN
    PERFORM fn_apply_entity_created(ev_id, p_world_id,
      NULLIF(p_attempt->>'target_id','')::uuid,
      p_attempt->>'new_entity_kind',
      p_attempt->>'canonical_name',
      p_attempt->>'descriptor',
      COALESCE(p_attempt->'new_attrs', '{}'::jsonb));
  END IF;

  PERFORM generate_perceptions(ev_id);

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

-- ── apply_ruled_event: the LIVE twin. Identical to 20260729100006 EXCEPT the EntityCreated branch is now
--    REAL — descriptor-mandatory floor + the shared-helper commit.
CREATE OR REPLACE FUNCTION public.apply_ruled_event(p_world_id uuid, p_ruled jsonb, p_tick bigint, p_seq integer, p_origin text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type    text;
  actor_id   uuid;
  ev_id      uuid;
  listener   uuid;
  to_target  uuid;   -- the move's target: ANY positioned entity (location | object | actor)
  v_scene    uuid;   -- resolved: the target's containing scene
  v_coord    jsonb;  -- resolved: the target's position — the mover's new coordinate
  object_eid uuid;
  dest_eid   uuid;
  target_eid uuid;
  here       uuid;
  vis_scope  text;
  truth_text text;
  appear_txt text;
  visible    boolean;
  receiver   uuid;
  recv_text  text;
  var_text   text;
  pid        uuid;
  participant_ids uuid[];
  v_old_holder uuid;   -- object's current carrier, read before the move (eager rule, §4)
BEGIN
  ev_type  := p_ruled->>'type';
  actor_id := (p_ruled->>'actor_id')::uuid;
  truth_text := p_ruled->>'truth';
  appear_txt := NULLIF(TRIM(COALESCE(p_ruled->>'appearance', '')), '');
  visible    := CASE
    WHEN p_ruled ? 'visible' AND (p_ruled->>'visible') = 'false' THEN false
    ELSE true
  END;

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_ruled->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, actor_id, to_target) THEN
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
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_ruled->>'target_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- DESCRIPTOR MANDATORY (§8 / §7 provenance-guarded create): no descriptor ⇒ not a true introduction
    -- ⇒ gate_reject. Blocker-only clerk (the adjudication already happened). Twin of apply_event.
    IF NULLIF(btrim(COALESCE(p_ruled->>'descriptor','')),'') IS NULL THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, ev_type, truth_text,
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

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

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  -- EntityCreated: persist the new instance via the shared helper (reuse-before-create inside).
  -- Provenance = ev_id (§8 net 3); one logged canon_event row (net 2); ruling passed reality-check (net 1).
  IF ev_type = 'EntityCreated' THEN
    PERFORM fn_apply_entity_created(ev_id, p_world_id,
      NULLIF(p_ruled->>'target_id','')::uuid,
      p_ruled->>'new_entity_kind',
      p_ruled->>'canonical_name',
      p_ruled->>'descriptor',
      COALESCE(p_ruled->'new_attrs', '{}'::jsonb));
  END IF;

  IF NOT visible THEN
    RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
  END IF;

  SELECT (a.attrs->>'location_id')::uuid INTO here
    FROM actor_state a
    WHERE a.world_id = p_world_id AND a.entity_id = actor_id;

  FOR receiver IN
    SELECT entity_id FROM fn_actors_at(p_world_id, here)
    UNION
    SELECT actor_id
  LOOP
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

-- migrate:down

-- Restore the 20260729100006 twins (EntityCreated = empty stub) and drop the helper.
DROP FUNCTION IF EXISTS public.fn_apply_entity_created(uuid, uuid, uuid, text, text, text, jsonb);

CREATE OR REPLACE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type      text;
  ev_id        uuid;
  listener     uuid;
  to_target    uuid;
  v_scene      uuid;
  v_coord      jsonb;
  object_eid   uuid;
  dest_eid     uuid;
  target_eid   uuid;
  here         uuid;
  vis_scope    text;
  final_type   text;
  v_old_holder uuid;
BEGIN
  ev_type := p_attempt->>'type';

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = p_actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_attempt->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, p_actor_id, to_target) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_attempt->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
    IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'ObjectRelocated' THEN
    object_eid := (p_attempt->>'object_id')::uuid;
    dest_eid   := (p_attempt->>'dest_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = object_eid AND world_id = p_world_id)
    OR NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = dest_eid  AND world_id = p_world_id)
    THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_attempt->>'target_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    NULL;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  IF p_legacy_types THEN
    final_type := CASE ev_type
      WHEN 'Communicated' THEN 'private_disclosure'
      WHEN 'ActorMoved'   THEN 'move'
      ELSE ev_type
    END;
  ELSE
    final_type := ev_type;
  END IF;

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, final_type, p_attempt->>'stated',
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, p_actor_id, 'actor', 'speaker'),
      (ev_id, listener,   'actor', 'listener');
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, p_actor_id, 'actor', 'instigator');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  PERFORM generate_perceptions(ev_id);

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

CREATE OR REPLACE FUNCTION public.apply_ruled_event(p_world_id uuid, p_ruled jsonb, p_tick bigint, p_seq integer, p_origin text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type    text;
  actor_id   uuid;
  ev_id      uuid;
  listener   uuid;
  to_target  uuid;
  v_scene    uuid;
  v_coord    jsonb;
  object_eid uuid;
  dest_eid   uuid;
  target_eid uuid;
  here       uuid;
  vis_scope  text;
  truth_text text;
  appear_txt text;
  visible    boolean;
  receiver   uuid;
  recv_text  text;
  var_text   text;
  pid        uuid;
  participant_ids uuid[];
  v_old_holder uuid;
BEGIN
  ev_type  := p_ruled->>'type';
  actor_id := (p_ruled->>'actor_id')::uuid;
  truth_text := p_ruled->>'truth';
  appear_txt := NULLIF(TRIM(COALESCE(p_ruled->>'appearance', '')), '');
  visible    := CASE
    WHEN p_ruled ? 'visible' AND (p_ruled->>'visible') = 'false' THEN false
    ELSE true
  END;

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_ruled->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, actor_id, to_target) THEN
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
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_ruled->>'target_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    NULL;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, ev_type, truth_text,
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

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

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  IF NOT visible THEN
    RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
  END IF;

  SELECT (a.attrs->>'location_id')::uuid INTO here
    FROM actor_state a
    WHERE a.world_id = p_world_id AND a.entity_id = actor_id;

  FOR receiver IN
    SELECT entity_id FROM fn_actors_at(p_world_id, here)
    UNION
    SELECT actor_id
  LOOP
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
