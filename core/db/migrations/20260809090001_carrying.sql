-- migrate:up

-- GET /worlds/{w}/carrying — "Carry States of the user-controlled Actor only"
-- (mvp_slice_and_bridge §4.1; PRD: Compendium — Artifacts & Carrying Overlay).
--
-- THE CARRIER IS THE VIEWER. There is no third argument, so "show me what that NPC is carrying" is
-- not expressible through this function at all — the PRD's non-goal ("Carrying for NPCs") is a
-- property of the signature, not a check someone can forget. The overlay is the play-facing answer
-- to "what do I have on me right now?", and it is deliberately NOT the Artifact Compendium (AC#1):
-- the Compendium lists meaningful known objects regardless of ownership, this lists possession.
--
-- WHY THIS READS CANON AND WHY THAT IS NOT A HOLE IN THE WALL (B-1, I-3).
-- Possession of your own belongings is not a thing you hold a perception ABOUT — it is a fact you
-- are living. Filtering this list through fn_entity_visible would hide a viewer's own pocket from
-- them. What the wall governs is what the viewer KNOWS, and every knowledge-bearing field here is
-- still viewer-scoped: `label` is fn_display_name (the viewer's own naming, descriptor fallback —
-- the same function the beat candidate whitelist uses), and `quick_inspect_preview` is
-- fn_compendium_latest_fact, which reads fn_visible_perceptions and nothing else. No canon ROW
-- crosses the boundary; what crosses is ids, the viewer's labels, and the viewer's own knowledge.
-- Precedent, not a new decision: core/api/beathandler.go already builds the candidate whitelist from
-- `attrs->>'contained_by' = <viewer>` with fn_display_name labels (quoted in SPEC-030).
--
-- WHY THE LEDGER AND NOT artifact_state.
-- trg_sm_project writes artifact_state.attrs.contained_by FROM state_mutation, so both carry the
-- same fact — but only the ledger carries the PROVENANCE the overlay is required to render:
-- which accepted event last confirmed this containment, at which in-world tick, under which
-- in-world label. Artifacts AC#3 asks for a `last_confirmed_tick` and for decay language on a stale
-- one; read from the projection, that tick would have to be invented or shipped null. Read from the
-- ledger it is a fact with an event behind it.
CREATE OR REPLACE FUNCTION fn_carrying(p_world_id uuid, p_viewer_id uuid)
RETURNS json
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  -- Current containment edge per entity: the newest applied `attrs.contained_by` mutation, by the
  -- domain ordering key (in_world_tick, beat_seq) — never recorded_at, which is transaction time
  -- (B-5, ADR-034). Putting a thing down is not a special case and needs no tombstone: apply_event's
  -- ObjectRelocated always names a destination (state_mutation.new_value is NOT NULL, so no
  -- "contained by nothing" edge is even writable), and a destination that is not the viewer simply
  -- stops rooting the chain at them. `#>>`/NULLIF are the defensive parse of a jsonb column whose
  -- type permits a JSON null nothing currently writes.
  latest AS (
    SELECT DISTINCT ON (sm.entity_id)
           sm.entity_id,
           NULLIF(sm.new_value #>> '{}', '')::uuid AS holder_id,
           sm.valid_from_tick,
           sm.event_id
    FROM state_mutation sm
    WHERE sm.world_id = p_world_id
      AND sm.attribute_path = 'attrs.contained_by'
      AND sm.status = 'applied'
    ORDER BY sm.entity_id, sm.valid_from_tick DESC, sm.valid_from_seq DESC
  ),
  -- Everything whose containment chain ROOTS AT THE VIEWER. Nesting is included because the engine
  -- already means it: fn_apply_carry_change climbs contained_by to the root carrier and charges the
  -- whole subtree to that actor's carried_weight. An overlay that showed only the top layer would
  -- contradict the encumbrance the same world computes — you would be penalised for weight the
  -- surface says you are not carrying. depth < 64 is the same contained_by cycle fail-safe
  -- fn_apply_carry_change uses (I-4).
  held(entity_id, container_id, depth, confirmed_tick, event_id) AS (
      SELECT l.entity_id, NULL::uuid, 0, l.valid_from_tick, l.event_id
      FROM latest l
      WHERE l.holder_id = p_viewer_id
    UNION ALL
      SELECT l.entity_id, h.entity_id, h.depth + 1, l.valid_from_tick, l.event_id
      FROM held h
      JOIN latest l ON l.holder_id = h.entity_id
      WHERE h.depth < 64
  ),
  items AS (
    SELECT h.entity_id,
           h.container_id,
           fn_display_name(p_world_id, p_viewer_id, h.entity_id) AS label,
           json_build_object(
             -- `id` IS the PRD's `open_full_artifact_link`. The link itself is not shipped: routes
             -- are the frontend's (D-7), and a backend that emitted "/worlds/…/artifacts/…/page"
             -- would be hardcoding someone else's URL space.
             'id',                    h.entity_id,
             'label',                 fn_display_name(p_world_id, p_viewer_id, h.entity_id),
             -- The PRD's Carry State enum is carried|worn|held|packed|stored_elsewhere|lost|unknown.
             -- The world records exactly one distinction — contained_by, i.e. in your possession or
             -- not — so `carried` is the only value it can honestly produce; worn/held/packed need a
             -- signal that does not exist, and stored_elsewhere/lost/unknown describe things that
             -- are NOT on you and so can never appear on this surface. Typed as a plain string and
             -- NOT enum-pinned on purpose: when a real signal lands the value set widens in place
             -- and costs the frontend no re-pin, provided it treats an unknown value as opaque.
             'state',                 'carried',
             -- null = directly on you. Non-null = the thing of yours it is inside. This reports the
             -- chain; it does not answer the PRD's open container-semantics question (pouch inside
             -- bag as a UI affordance), which stays open.
             'container',             CASE WHEN h.container_id IS NULL THEN NULL
                                      ELSE json_build_object(
                                        'id',    h.container_id,
                                        'label', fn_display_name(p_world_id, p_viewer_id, h.container_id))
                                      END,
             'last_confirmed_tick',   h.confirmed_tick,
             -- Viewer-held knowledge only (fn_visible_perceptions), so the preview line can never
             -- say more about the object than its carrier actually knows. Null is ordinary: you can
             -- carry a thing you have learned nothing about.
             'quick_inspect_preview', fn_compendium_latest_fact(p_world_id, p_viewer_id, h.entity_id),
             -- Artifacts AC#3: a stale carry state renders decay language and never disappears.
             -- Same horizon and same shape as every other compendium surface.
             'decay',                 fn_compendium_decay(p_world_id, h.confirmed_tick, ce.in_world_label)
           ) AS item
    FROM held h
    JOIN entity_registry er
      ON er.world_id = p_world_id AND er.entity_id = h.entity_id AND er.entity_kind = 'artifact'
    JOIN canon_event ce
      ON ce.event_id = h.event_id
  )
  SELECT json_build_object(
    'schema_version', 'carrying/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    -- Always an array, never null: "you are carrying nothing" is an answer, not a missing page.
    -- Order is deterministic and stable — what is directly on you first, then alphabetically by the
    -- label you yourself use, id as the final tiebreak.
    'carried', coalesce(
      (SELECT json_agg(i.item ORDER BY (i.container_id IS NOT NULL), i.label, i.entity_id)
       FROM items i),
      '[]'::json)
  );
$$;

-- migrate:down

DROP FUNCTION IF EXISTS fn_carrying(uuid, uuid);
