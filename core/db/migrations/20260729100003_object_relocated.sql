-- migrate:up

-- Station F / FINAL-action-contracts.md §4 (ObjectRelocated): pick up / drop / give / put. TWO
-- dimensions, exactly (founder-locked), with different roles:
--
--   VOLUME — BLOCKS (geometric impossibility). object size 1..10; volume(size) = 4^(size-1); a
--            container has max_room. Room-check, computed at ask-time and NEVER stored:
--                occupied_room + 4^(size-1) <= max_room
--            A size-5 crate cannot enter a size-2 pouch. A true blocker.
--
--   WEIGHT — NEVER blocks; it CONSEQUENCES. Grabbing two tons is not impossible — MOVING with it is.
--            effective_weight(container) = (empty_weight + Σ effective_weight(contents)) × weight_modifier
--            EAGER RULE (founder-locked): on any commit that changes a carry chain (grab, drop, hand
--            over, item added to a held container) recompute carried_weight for every affected carrier
--            — recursively UP the chain — and write/clear the seeded `encumbered` status in the SAME
--            commit. carried_weight > max_load → encumbered (movement -100%, a full stop). Provenance:
--            the relocation event. The world sees the strain the instant it happens, never a stale one.
--
-- `within-load` as a blocker is DEAD (do NOT reimplement it): weight is a consequence (status), not a
-- gate. Relocation has NO time cost of its own — the preceding move carries the time (§4).
--
-- Containment became engine-readable this migration: carry is the single Tier-1 key `contained_by`
-- (a string; added to core/api/tier1.go tier1Registry). It MUST be Tier-1 — the eager rule reads it
-- (Rule A: the engine may only read attributes named in the contracts). The Drowned Lantern seed's
-- former Tier-2 carried_by/held_by are converted to contained_by. Contents of X = artifacts whose
-- attrs->>'contained_by' = X. Actors are the root carriers.

-- ── fn_volume: the DERIVED geometry (§4). volume(size) = 4^(size-1). NOT stored, NOT in the registry
--    (binding decision 3): it is arithmetic over `size`, recomputed at every ask. IMMUTABLE (pure).
CREATE FUNCTION public.fn_volume(p_size int)
RETURNS numeric
LANGUAGE sql IMMUTABLE AS $$
  SELECT power(4, p_size - 1)::numeric;   -- fn_volume(1)=1, fn_volume(5)=256
$$;

-- ── fn_occupied_room: occupied_room is a MEASUREMENT (§0) but DERIVED, not stored (decision: a stored
--    occupied_room is a cached number that rots the moment a content moves — the exact silent-corruption
--    class §0 refuses; recomputing Σ volume(contents) at ask-time is cleaner and measurements-consistent).
--    Contents of the container = artifacts whose contained_by points at it. v1 has no volume modifier
--    (a bag-of-holding is a future contents-volume modifier, §4) — plain sum for now.
CREATE FUNCTION public.fn_occupied_room(p_world_id uuid, p_container uuid)
RETURNS numeric
LANGUAGE sql STABLE AS $$
  SELECT COALESCE(SUM(fn_volume(COALESCE((attrs->>'size')::int, 1))), 0)
  FROM artifact_state
  WHERE world_id = p_world_id
    AND (attrs->>'contained_by')::uuid = p_container;
$$;

-- ── fn_effective_weight (§4 container formula, recursive). A plain object = its own `weight`. A
--    container (has empty_weight and/or max_room — the container Tier-1 props, §5.2) =
--        (empty_weight + Σ effective_weight(contents)) × COALESCE(weight_modifier, 1)
--    The modifier wraps BOTH terms deliberately (a soaked pack's own fabric is heavier too; a
--    lightening enchantment lightens the whole thing). Mundane container: modifier = 1.
--
--    Worked example (the waterlogged pack): pack empty 2 kg, modifier 1.0, holding 4 crates each
--    (25 + 0) × 1.6 = 40 → pack = (2 + 4×40) × 1.0 = 162.
--
--    CYCLE GUARD (mirrors causal_bundle_assert_acyclic's depth cap, I-4): a carried item that contains
--    its own carrier would recurse forever; the private _r helper caps depth at 64 and RAISES. The
--    public 2-arg signature is the contract's; the depth lives only in the private recursion.
CREATE FUNCTION public.fn_effective_weight_r(p_world_id uuid, p_entity uuid, p_depth int)
RETURNS numeric
LANGUAGE plpgsql STABLE AS $$
DECLARE
  v_attrs jsonb;
  v_empty numeric;
  v_mod   numeric;
  v_sum   numeric := 0;
  v_child uuid;
BEGIN
  IF p_depth > 64 THEN
    RAISE EXCEPTION
      'containment depth cap (64) exceeded weighing % — likely a contained_by cycle (mirror I-4 guard)',
      p_entity;
  END IF;

  SELECT attrs INTO v_attrs
  FROM artifact_state
  WHERE world_id = p_world_id AND entity_id = p_entity;

  -- plain object (no container props): its own weight, defaulting to 0.
  IF v_attrs IS NULL OR NOT (v_attrs ? 'empty_weight' OR v_attrs ? 'max_room') THEN
    RETURN COALESCE((v_attrs->>'weight')::numeric, 0);
  END IF;

  -- container: (empty + Σ contents) × modifier.
  v_empty := COALESCE((v_attrs->>'empty_weight')::numeric, 0);
  v_mod   := COALESCE((v_attrs->>'weight_modifier')::numeric, 1);
  FOR v_child IN
    SELECT entity_id FROM artifact_state
    WHERE world_id = p_world_id AND (attrs->>'contained_by')::uuid = p_entity
  LOOP
    v_sum := v_sum + fn_effective_weight_r(p_world_id, v_child, p_depth + 1);
  END LOOP;

  RETURN (v_empty + v_sum) * v_mod;
END $$;

CREATE FUNCTION public.fn_effective_weight(p_world_id uuid, p_entity uuid)
RETURNS numeric
LANGUAGE sql STABLE AS $$
  SELECT fn_effective_weight_r(p_world_id, p_entity, 0);
$$;

-- ── fn_apply_carry_change: the EAGER rule, factored into ONE plpgsql helper called by BOTH apply_event
--    and apply_ruled_event so the twins can never drift (recon flagged the "keep in sync" comment on
--    the ObjectRelocated floor; a shared helper is the DRY fix). Called AFTER the relocation event is
--    committed; provenance for every write it makes is that event.
--
--    1. Write the object's new containment edge (Tier-1 contained_by). This IS the carry-chain change;
--       trg_sm_project projects it into artifact_state immediately (same txn), so the recompute below
--       reads the post-move contents.
--    2. Recompute carried_weight for every affected ACTOR — walking UP contained_by from BOTH the old
--       and the new holder (an item can leave one carrier and join another). carried_weight(actor) =
--       Σ effective_weight(direct contents). Write it (a measurement) and set/clear `encumbered` where
--       carried_weight > max_load (a person's static capacity; absent max_load → never encumbered).
--    Containers never store a weight — theirs is recomputed fresh by fn_effective_weight; only actors
--    carry the stored carried_weight + the encumbered status (movement is an actor concern).
CREATE FUNCTION public.fn_apply_carry_change(
  p_event_id   uuid,
  p_world_id   uuid,
  p_object     uuid,
  p_old_holder uuid,
  p_new_holder uuid
)
RETURNS void
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  v_tick    bigint;
  v_seq     int;
  v_okind   text;
  v_carrier uuid;
  v_cw      numeric;
  v_maxload numeric;
  v_stat    jsonb;
BEGIN
  -- Provenance tick/seq come from the just-committed relocation event (single source of truth).
  SELECT in_world_tick, beat_seq INTO v_tick, v_seq
  FROM canon_event WHERE event_id = p_event_id;

  SELECT entity_kind INTO v_okind
  FROM entity_registry WHERE world_id = p_world_id AND entity_id = p_object;

  -- (1) new containment edge. Projects into <kind>_state.attrs.contained_by via trg_sm_project.
  INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                              new_value, valid_from_tick, valid_from_seq)
  VALUES (p_world_id, p_event_id, p_object, v_okind, 'attrs.contained_by',
          to_jsonb(p_new_holder::text), v_tick, v_seq);

  -- (2) recompute every affected actor. Walk UP contained_by from {old_holder, new_holder}; the base
  --     rows are the holders themselves (an actor holder is affected directly), the recursion climbs
  --     through nested containers to the root carrier. depth < 64 caps a contained_by cycle (I-4).
  FOR v_carrier IN
    WITH RECURSIVE chain(node, depth) AS (
      SELECT h, 0
      FROM (VALUES (p_old_holder), (p_new_holder)) AS s(h)
      WHERE h IS NOT NULL
      UNION ALL
      SELECT (a.attrs->>'contained_by')::uuid, c.depth + 1
      FROM chain c
      JOIN artifact_state a ON a.world_id = p_world_id AND a.entity_id = c.node
      WHERE a.attrs->>'contained_by' IS NOT NULL AND c.depth < 64
    )
    SELECT DISTINCT c.node
    FROM chain c
    JOIN entity_registry er ON er.world_id = p_world_id AND er.entity_id = c.node
    WHERE er.entity_kind = 'actor'
  LOOP
    v_seq := v_seq + 1;

    SELECT COALESCE(SUM(fn_effective_weight(p_world_id, e.entity_id)), 0)
      INTO v_cw
      FROM artifact_state e
      WHERE e.world_id = p_world_id AND (e.attrs->>'contained_by')::uuid = v_carrier;

    SELECT (attrs->>'max_load')::numeric,
           COALESCE(attrs->'statuses', '[]'::jsonb)
      INTO v_maxload, v_stat
      FROM actor_state
      WHERE world_id = p_world_id AND entity_id = v_carrier;

    -- carried_weight is a MEASUREMENT (§0), written eagerly so cognition/perception read a true state.
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, p_event_id, v_carrier, 'actor', 'attrs.carried_weight',
            to_jsonb(v_cw), v_tick, v_seq);

    -- set/clear encumbered. v_cw > NULL (no max_load) is NULL → the ELSE clears — an unset capacity
    -- can't be exceeded. `?` tests string membership in the statuses array.
    IF v_cw > v_maxload THEN
      IF NOT (v_stat ? 'encumbered') THEN
        v_stat := v_stat || '["encumbered"]'::jsonb;
      END IF;
    ELSE
      SELECT COALESCE(jsonb_agg(x), '[]'::jsonb)
        INTO v_stat
        FROM jsonb_array_elements(v_stat) x
        WHERE x <> '"encumbered"'::jsonb;
    END IF;

    v_seq := v_seq + 1;
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, p_event_id, v_carrier, 'actor', 'attrs.statuses',
            v_stat, v_tick, v_seq);
  END LOOP;
END $$;

-- ── apply_event: add the ObjectRelocated VOLUME floor + the eager-encumbrance commit. Twin of
--    apply_ruled_event below (shared helper fn_apply_carry_change keeps them in lockstep).
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
    to_loc := (p_attempt->>'to_location_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry
      WHERE entity_id = to_loc AND world_id = p_world_id
    ) THEN
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

  -- state_mutation: ActorMoved only — trigger projects it into actor_state.
  IF ev_type = 'ActorMoved' THEN
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id',
            to_jsonb(to_loc::text), p_tick, p_seq);
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

-- ── apply_ruled_event: the SAME ObjectRelocated volume floor + eager-encumbrance commit (twin of
--    apply_event; both delegate the weight work to fn_apply_carry_change — keep in sync).
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

  -- state_mutation: ActorMoved only — trigger projects it into actor_state.
  IF ev_type = 'ActorMoved' THEN
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id',
            to_jsonb(to_loc::text), p_tick, p_seq);
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

-- migrate:down

-- Restore the existence-only ObjectRelocated floor (no volume, no eager encumbrance) in both twins,
-- then drop the Station-F §4 helpers. NOTE: the Drowned Lantern seed's contained_by conversion is a
-- seed-file edit (seeds load AFTER migrate) — not reverted here; re-seed from the prior seed to undo.

CREATE OR REPLACE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type     text;
  ev_id       uuid;
  listener    uuid;
  to_loc      uuid;
  object_eid  uuid;
  dest_eid    uuid;
  target_eid  uuid;
  here        uuid;
  vis_scope   text;
  final_type  text;
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
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = to_loc AND world_id = p_world_id) THEN
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
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = to_loc AND world_id = p_world_id) THEN
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

DROP FUNCTION IF EXISTS public.fn_apply_carry_change(uuid, uuid, uuid, uuid, uuid);
DROP FUNCTION IF EXISTS public.fn_effective_weight(uuid, uuid);
DROP FUNCTION IF EXISTS public.fn_effective_weight_r(uuid, uuid, int);
DROP FUNCTION IF EXISTS public.fn_occupied_room(uuid, uuid);
DROP FUNCTION IF EXISTS public.fn_volume(int);
