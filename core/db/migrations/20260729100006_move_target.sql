-- migrate:up

-- Station F / Task 8 — UNIFY THE MOVE: the move targets ANY positioned entity (architecture
-- correction, founder-directed 2026-07-30). FINAL-action-contracts.md §2 (move) + §3 (space).
--
-- The clean chain was already built: SPACE answers "how far" (fn_distance, uniform, layer-agnostic),
-- VELOCITY turns distance into time (fn_effective_speed → duration = distance ÷ speed), the GATE checks
-- time vs the tension budget. The ONE gap: the move EVENT was location-keyed — ActorMoved accepted only
-- to_location_id and updated only attrs.location_id, so it could cross rooms but could not reposition an
-- actor WITHIN a scene or move them TOWARD a thing (the bar). This migration removes that restriction.
-- It does NOT add a parallel field: ActorMoved's target is now to_target_id — the id of ANY positioned
-- entity (a location, an object like the bar, an actor). A location is just a target that is a location.
--
-- This is NOT a per-layer special case: "approach the bar" (in-scene) and "walk to the dock street"
-- (cross-location) are the SAME operation. A same-scene move changes only coordinates; a cross-location
-- move changes location_id → the §5.3 portal gate applies exactly then. Both fall out of ONE uniform
-- commit, no branch. This path stays LLM-FREE (D-1): pure arithmetic + stored measurements.

-- ── fn_target_position: resolve ANY positioned entity to its (scene, coordinate) — the destination the
--    mover lands at. Reuses fn_distance's entity→scene resolution (§3): a LOCATION is its own scene; an
--    ACTOR's scene is its attrs.location_id; an ARTIFACT's is COALESCE(attrs.location_id, current_scene).
--    The landing COORDINATE:
--      * target is a LOCATION  → the location's LOCAL origin / authored entry (COALESCE(entry_point,
--        {0,0})). §3 ACCEPTED IMPRECISION (do NOT "fix" this): moving to a location lands you at its
--        entry, NOT an inferred exit point — inferring one is reasoning, which is an LLM, and this path
--        is deliberately LLM-free. NOTE the location's own attrs.coordinates is its coordinate-in-PARENT
--        (its position among its siblings), never a landing spot inside it — that is why a location
--        target uses entry_point/origin, not attrs.coordinates.
--      * target is an ACTOR / ARTIFACT → its OWN attrs.coordinates (its position within its scene), so
--        "approach the bar" lands the mover where the bar stands.
--    STABLE, LLM-free; OUT params so the commit twins and the gate read ONE resolution (no drift).
CREATE FUNCTION public.fn_target_position(p_world_id uuid, p_target uuid,
                                          OUT scene uuid, OUT coord jsonb)
LANGUAGE sql STABLE AS $$
  SELECT
    CASE er.entity_kind
      WHEN 'location' THEN p_target
      WHEN 'actor'    THEN (ast.attrs->>'location_id')::uuid
      WHEN 'artifact' THEN COALESCE((art.attrs->>'location_id')::uuid, er.current_scene_id)
    END AS scene,
    CASE er.entity_kind
      WHEN 'location' THEN COALESCE(ls.attrs->'entry_point', '{"x":0,"y":0}'::jsonb)
      ELSE COALESCE(ast.attrs->'coordinates', art.attrs->'coordinates', '{"x":0,"y":0}'::jsonb)
    END AS coord
  FROM entity_registry er
  LEFT JOIN actor_state    ast ON ast.world_id = p_world_id AND ast.entity_id = p_target
  LEFT JOIN artifact_state art ON art.world_id = p_world_id AND art.entity_id = p_target
  LEFT JOIN location_state  ls ON  ls.world_id = p_world_id AND  ls.entity_id = p_target
  WHERE er.world_id = p_world_id AND er.entity_id = p_target;
$$;

-- ── fn_move_duration_actor: GENERALIZED to the target form (3-arg). duration = CEIL(fn_distance(actor,
--    target) ÷ effective_speed). The actor's current position is IMPLICIT (its own coordinate/scene) —
--    fn_distance is entity-agnostic, so the target can be a location, an object, or an actor. Speed 0
--    (a -100% modifier, e.g. encumbered/tied) → max bigint so "never fits any budget" EMERGES from the
--    arithmetic (§2), never a branch. 1 tick = 1 s. The old 4-arg (from,to) LOCATION-PAIR form is
--    DROPPED — the actor's origin is no longer a caller-supplied location, it is where the actor stands.
DROP FUNCTION IF EXISTS public.fn_move_duration_actor(uuid, uuid, uuid, uuid);
CREATE FUNCTION public.fn_move_duration_actor(p_world_id uuid, p_actor uuid, p_target uuid)
RETURNS bigint
LANGUAGE sql STABLE AS $$
  SELECT CASE
    WHEN COALESCE(fn_effective_speed(p_world_id, p_actor, 'walk'), 0) <= 0
      THEN 9223372036854775807::bigint   -- infinite duration: blocked by arithmetic (§2), not a branch
    ELSE CEIL(
      fn_distance(p_world_id, p_actor, p_target) / fn_effective_speed(p_world_id, p_actor, 'walk')
    )::bigint
  END;
$$;

-- ── fn_actor_move_permitted: the WHOLE ActorMoved accessibility floor, now keyed on the TARGET. It
--    resolves the target's SCENE via the shared fn_target_position (same resolution the commit writes),
--    then gates the portal ONLY for a cross-location move:
--      1. the TARGET exists in entity_registry (the pre-Portal existence floor); else FALSE.
--      2. SAME-SCENE move (the target's scene == the actor's current location, e.g. crossing to the bar)
--         is not a traversal → TRUE, no portal. An actor with no origin yet (here IS NULL) is likewise
--         not traversing → TRUE.
--      3. cross-location move → a Portal connecting here↔(target's scene) must permit passage (§5.3 v1).
--    Portal is accessibility, NOT geometry: this resolves the scene but never reads fn_distance/coords.
--    Referenced only inside plpgsql twins (late-bound) → DROP+CREATE is safe (no pg_depend edge).
DROP FUNCTION IF EXISTS public.fn_actor_move_permitted(uuid, uuid, uuid);
CREATE FUNCTION public.fn_actor_move_permitted(p_world_id uuid, p_actor_id uuid, p_target uuid)
RETURNS boolean
LANGUAGE plpgsql STABLE AS $$
DECLARE
  v_here  uuid;
  v_scene uuid;
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM entity_registry WHERE entity_id = p_target AND world_id = p_world_id
  ) THEN
    RETURN false;   -- target not a registered entity: the existence floor (as before Portal)
  END IF;

  SELECT scene INTO v_scene FROM fn_target_position(p_world_id, p_target);

  SELECT (attrs->>'location_id')::uuid INTO v_here
    FROM actor_state WHERE world_id = p_world_id AND entity_id = p_actor_id;

  IF v_here IS NULL OR v_here = v_scene THEN
    RETURN true;    -- same-scene (or no origin): not a traversal → no portal required
  END IF;

  RETURN fn_portal_permits(p_world_id, v_here, v_scene);   -- cross-location: a portal must permit it
END $$;

-- ── apply_event: identical to 20260729100004 EXCEPT ActorMoved now reads to_target_id, gates on the
--    target's scene (fn_actor_move_permitted), and COMMITS the mover's new coordinate AND scene via
--    fn_target_position. Twin of apply_ruled_event below — the shared helpers keep them in lockstep, and
--    core/api/orchestrator.go premiseHolds mirrors the same rule.
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

  -- ── STRUCTURAL FLOOR ───────────────────────────────────────────────────────
  -- Every type: actor must exist in entity_registry with kind='actor'.
  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = p_actor_id
      AND world_id  = p_world_id
      AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- Type-specific floor checks.
  IF ev_type = 'ActorMoved' THEN
    to_target := (p_attempt->>'to_target_id')::uuid;
    -- ACCESSIBILITY FLOOR (§5.3): the target exists AND a Portal permits here→(target's scene). A
    -- SAME-SCENE move (the target's scene == here) is not a traversal and needs no portal. The whole
    -- decision lives in the shared fn_actor_move_permitted so this twin, apply_ruled_event, and the Go
    -- premiseHolds mirror never drift. Portal is accessibility, NOT geometry — never touches fn_distance.
    IF NOT fn_actor_move_permitted(p_world_id, p_actor_id, to_target) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_attempt->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
    IF NOT EXISTS (
      SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener
    ) THEN
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
    -- VOLUME FLOOR (§4): a true blocker, ONLY when dest is a Container (carries max_room). A
    -- non-container dest (a location, an actor's hands) has no room check in v1. occupied_room and
    -- volume are BOTH computed here, never stored. Missing object size → treated as 1 (smallest).
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
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    -- No target check required (adjudicated intent; actor floor above covers the writer).
    NULL;

  ELSE
    -- Unknown type: reject.
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  -- ── COMMIT ─────────────────────────────────────────────────────────────────
  ev_id := gen_random_uuid();

  -- Determine the final event_type string to write.
  -- p_legacy_types=true: write old labels so apply_beat's delegation is behavior-identical.
  IF p_legacy_types THEN
    final_type := CASE ev_type
      WHEN 'Communicated' THEN 'private_disclosure'
      WHEN 'ActorMoved'   THEN 'move'
      ELSE ev_type
    END;
  ELSE
    final_type := ev_type;
  END IF;

  -- visibility_scope: private for Communicated (or legacy private_disclosure), public otherwise.
  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin)
  VALUES (ev_id, p_world_id, final_type, p_attempt->>'stated',
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin);

  -- event_participant: Communicated → speaker + listener; all others → instigator.
  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, p_actor_id, 'actor', 'speaker'),
      (ev_id, listener,   'actor', 'listener');
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, p_actor_id, 'actor', 'instigator');
  END IF;

  -- state_mutation: ActorMoved only — the trigger projects it into actor_state.
  -- §2/§3 UNIFIED MOVE: resolve the target's position ONCE (the same fn_target_position the gate used)
  -- and write BOTH the mover's new scene AND coordinate. location_id = the target's containing scene (or
  -- the target itself when it IS a location); coordinates = the target's position. A SAME-SCENE move
  -- writes location_id unchanged-in-value — only coordinates move — so the "in-scene reposition" case
  -- falls out of this uniform write with no branch. Moving to a LOCATION lands the actor at that
  -- location's local origin/authored entry (§3 ACCEPTED IMPRECISION — no exit-point inference).
  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  -- ObjectRelocated: EAGER carry change (§4) — write the new contained_by edge, then recompute
  -- carried_weight + set/clear encumbered for every affected carrier, all provenance = this event.
  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  PERFORM generate_perceptions(ev_id);

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

-- ── apply_ruled_event: the SAME unified move (twin of apply_event; keep in sync). ActorMoved reads
--    to_target_id, gates via fn_actor_move_permitted, and commits coordinate + scene via fn_target_position.
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
  -- perception fan-out
  receiver   uuid;
  recv_text  text;
  var_text   text;
  pid        uuid;
  -- participant ids for about-ness (perception_subject)
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
    to_target := (p_ruled->>'to_target_id')::uuid;
    -- ACCESSIBILITY FLOOR (§5.3): the target exists AND a Portal permits here→(target's scene), unless
    -- it is a SAME-SCENE move (the target's scene == here, not a traversal). Shared fn_actor_move_permitted
    -- = same decision as apply_event and the Go premiseHolds mirror (no twin drift). Accessibility, NOT
    -- geometry.
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
    -- VOLUME FLOOR (§4): true blocker, ONLY when dest is a Container (carries max_room). Non-container
    -- dest → no room check in v1. occupied_room + volume computed here, never stored.
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

  -- state_mutation: ActorMoved only — §2/§3 unified move (twin of apply_event; keep in sync). Resolve the
  -- target's position ONCE and write the mover's new scene + coordinate. Same-scene ⇒ location_id
  -- unchanged-in-value (only coordinates move); location target ⇒ its local origin/authored entry (§3).
  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  -- ObjectRelocated: EAGER carry change (§4), identical to apply_event via the shared helper.
  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
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

-- ── apply_beat: the SECURITY DEFINER fast-path delegator. Its move step builds the ActorMoved attempt
--    that apply_event now reads — so the field it emits is to_target_id (the beat_chain/1 `to` is a
--    location, which is just a target that is a location). Duration for the clock stays the legacy
--    fn_move_duration(from,to) walk wrapper (decision 6). Otherwise identical to 20260723100004.
CREATE OR REPLACE FUNCTION public.apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb, p_start_tick bigint, p_tick_cap bigint, p_origin text DEFAULT 'fast_path'::text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  step       jsonb;
  idx        int := 0;
  cur_tick   bigint := p_start_tick;
  cur_seq    int := 0;
  start_tick bigint := p_start_tick;
  committed  jsonb := '[]'::jsonb;
  halt       text := 'completed';
  dur        bigint;
  here       uuid;
  listener   uuid;
  next_step  jsonb;
  next_ok    boolean;
  attempt    jsonb;
  result     jsonb;
BEGIN
  FOR step IN SELECT * FROM jsonb_array_elements(p_chain) LOOP
    idx := idx + 1;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;

    -- Build the attempt shape and compute duration for budget check.
    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      attempt := jsonb_build_object(
        'type',        'Communicated',
        'stated',      COALESCE(step->>'content', 'say'),
        'listener_id', listener,
        'content',     step->>'content'
      );
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      attempt := jsonb_build_object(
        'type',         'ActorMoved',
        'stated',       'move',
        'to_target_id', step->>'to'
      );
      dur := fn_move_duration(p_world_id, here, (step->>'to')::uuid);
    ELSE
      halt := 'gate_reject'; EXIT;     -- out-of-vocabulary (closed set; ADR-009/D-1, SPEC-015)
    END IF;

    -- turn-budget backstop (§9 third pushback face): would committing exceed the cap?
    IF (cur_tick + dur) - start_tick > p_tick_cap THEN
      halt := 'turn_budget'; EXIT;
    END IF;

    -- Delegate to apply_event with p_legacy_types=true to preserve legacy event_type labels.
    result := apply_event(p_world_id, p_actor_id, attempt, cur_tick, cur_seq, p_origin, true);

    IF result->>'halt_reason' = 'gate_reject' THEN
      halt := 'gate_reject'; EXIT;
    END IF;

    committed := committed || to_jsonb(result->>'event_id');

    -- advance the clock by THIS committed event's duration (ADR-036).
    IF dur > 0 THEN cur_tick := cur_tick + dur; cur_seq := 0; ELSE cur_seq := cur_seq + 1; END IF;

    -- (3d) STOP-CHECK — runs ONLY after a committed MOVE: a move is the only thing that changes the
    -- actor's co-presence AND the only thing that generates a discovery perception. Gating on 'move'
    -- keeps the two halts DISTINCT (§8): a committed SAY must never pre-empt a later step's own gate.
    next_step := p_chain -> idx;   -- 0-based: element after the current 1-based idx
    IF step->>'type' = 'move' AND next_step IS NOT NULL AND next_step->>'type' = 'say' THEN
      SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      next_ok := EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here)
                         WHERE entity_id = (next_step->>'listener')::uuid);
      IF NOT next_ok THEN halt := 'stop_check'; EXIT; END IF;  -- prefix stands; remainder never runs
    END IF;
  END LOOP;

  RETURN jsonb_build_object('committed', committed, 'halt_reason', halt,
                            'ticks_advanced', cur_tick - start_tick);
END $$;

-- migrate:down

-- Restore the 20260723100004 apply_beat (to_location_id attempt builder).
CREATE OR REPLACE FUNCTION public.apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb, p_start_tick bigint, p_tick_cap bigint, p_origin text DEFAULT 'fast_path'::text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  step       jsonb;
  idx        int := 0;
  cur_tick   bigint := p_start_tick;
  cur_seq    int := 0;
  start_tick bigint := p_start_tick;
  committed  jsonb := '[]'::jsonb;
  halt       text := 'completed';
  dur        bigint;
  here       uuid;
  listener   uuid;
  next_step  jsonb;
  next_ok    boolean;
  attempt    jsonb;
  result     jsonb;
BEGIN
  FOR step IN SELECT * FROM jsonb_array_elements(p_chain) LOOP
    idx := idx + 1;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;

    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      attempt := jsonb_build_object(
        'type',        'Communicated',
        'stated',      COALESCE(step->>'content', 'say'),
        'listener_id', listener,
        'content',     step->>'content'
      );
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      attempt := jsonb_build_object(
        'type',           'ActorMoved',
        'stated',         'move',
        'to_location_id', step->>'to'
      );
      dur := fn_move_duration(p_world_id, here, (step->>'to')::uuid);
    ELSE
      halt := 'gate_reject'; EXIT;
    END IF;

    IF (cur_tick + dur) - start_tick > p_tick_cap THEN
      halt := 'turn_budget'; EXIT;
    END IF;

    result := apply_event(p_world_id, p_actor_id, attempt, cur_tick, cur_seq, p_origin, true);

    IF result->>'halt_reason' = 'gate_reject' THEN
      halt := 'gate_reject'; EXIT;
    END IF;

    committed := committed || to_jsonb(result->>'event_id');

    IF dur > 0 THEN cur_tick := cur_tick + dur; cur_seq := 0; ELSE cur_seq := cur_seq + 1; END IF;

    next_step := p_chain -> idx;
    IF step->>'type' = 'move' AND next_step IS NOT NULL AND next_step->>'type' = 'say' THEN
      SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      next_ok := EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here)
                         WHERE entity_id = (next_step->>'listener')::uuid);
      IF NOT next_ok THEN halt := 'stop_check'; EXIT; END IF;
    END IF;
  END LOOP;

  RETURN jsonb_build_object('committed', committed, 'halt_reason', halt,
                            'ticks_advanced', cur_tick - start_tick);
END $$;

-- Restore the 20260729100004 apply_ruled_event (to_location_id; single location_id mutation).
CREATE OR REPLACE FUNCTION public.apply_ruled_event(p_world_id uuid, p_ruled jsonb, p_tick bigint, p_seq integer, p_origin text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
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
    to_loc := (p_ruled->>'to_location_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, actor_id, to_loc) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_ruled->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = actor_id;
    IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
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
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id) THEN
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
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id',
            to_jsonb(to_loc::text), p_tick, p_seq);
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

-- Restore the 20260729100004 apply_event (to_location_id; single location_id mutation).
CREATE OR REPLACE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type      text;
  ev_id        uuid;
  listener     uuid;
  to_loc       uuid;
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
    to_loc := (p_attempt->>'to_location_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, p_actor_id, to_loc) THEN
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
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id',
            to_jsonb(to_loc::text), p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  PERFORM generate_perceptions(ev_id);

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

-- Restore the 20260729100004 fn_actor_move_permitted (world, actor, to_loc — LOCATION-keyed).
DROP FUNCTION IF EXISTS public.fn_actor_move_permitted(uuid, uuid, uuid);
CREATE FUNCTION public.fn_actor_move_permitted(p_world_id uuid, p_actor_id uuid, p_to_loc uuid)
RETURNS boolean
LANGUAGE plpgsql STABLE AS $$
DECLARE
  v_here uuid;
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM entity_registry WHERE entity_id = p_to_loc AND world_id = p_world_id
  ) THEN
    RETURN false;
  END IF;

  SELECT (attrs->>'location_id')::uuid INTO v_here
    FROM actor_state WHERE world_id = p_world_id AND entity_id = p_actor_id;

  IF v_here IS NULL OR v_here = p_to_loc THEN
    RETURN true;
  END IF;

  RETURN fn_portal_permits(p_world_id, v_here, p_to_loc);
END $$;

-- Restore the 20260729100002 fn_move_duration_actor (4-arg location-pair form).
DROP FUNCTION IF EXISTS public.fn_move_duration_actor(uuid, uuid, uuid);
CREATE FUNCTION public.fn_move_duration_actor(p_world_id uuid, p_actor uuid, p_from uuid, p_to uuid)
RETURNS bigint
LANGUAGE sql STABLE AS $$
  SELECT CASE
    WHEN COALESCE(fn_effective_speed(p_world_id, p_actor, 'walk'), 0) <= 0
      THEN 9223372036854775807::bigint
    ELSE CEIL(
      fn_distance(p_world_id, p_from, p_to) / fn_effective_speed(p_world_id, p_actor, 'walk')
    )::bigint
  END;
$$;

DROP FUNCTION IF EXISTS public.fn_target_position(uuid, uuid);
