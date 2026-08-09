-- migrate:up

-- Collected Knowledge is grouped by SUBJECT, not by source event.
--
-- ── WHAT WAS WRONG ──────────────────────────────────────────────────────────────────────────────
-- SPEC-029 (#40) grouped by `source_event_id` and labelled each group with that event's
-- `in_world_label`. `in_world_label` is the MOMENT label — B-5's authored display companion to the
-- logical tick — and `trg_canon_event_carry_in_world_label` deliberately carries it forward to every
-- later event in the same moment. So it is neither unique nor a topic. Measured on a played world:
-- Mara's dossier rendered TWENTY-FIVE groups, every one of them headed "Arrival".
--
-- The label was only the visible half. One group per event is one group per log line, and both
-- Compendium PRDs forbid exactly that in the same words: "Collected Knowledge should be grouped by
-- topic, not by raw timeline/log order… The user should not read an Actor page as a log… The raw
-- event order belongs more naturally in Timeline" (Actors PRD §10; Artifacts PRD, Collected
-- Knowledge). Event-keyed grouping is log order wearing a heading.
--
-- ── WHY SUBJECT, AND NOT THE OTHER THREE CANDIDATES ─────────────────────────────────────────────
-- A topic must come from something the world actually records. Only four signals exist per record:
--
--   epistemic_type   HOW you learned it. The PRD lists source as a WITHIN-group attribute ("where
--                    each piece came from"), and every item already ships it, so grouping by it
--                    duplicates data the frontend already renders per line.
--   source_event_id  WHICH event. That is the log order the PRDs are ruling out.
--   in_world_label   WHEN. Time, and coarse: the defect above.
--   perception_subject   WHAT IT IS ABOUT. Written at write time by whatever creates the
--                    perception (SPEC-008 / ADR-035), for precisely this reason.
--
-- About-ness is the only axis that is genuinely a topic, and it is the shape the mockups already
-- use: "The informant" and "Dark Foxes connection" are entities. Nothing is invented and no model is
-- consulted — SQL cannot call one, and a synthesised topic would assert more than the viewer holds.
--
-- ── A TOPIC MUST BE SOMETHING THE VIEWER CAN NAME, AND IS NEVER THE READER ──────────────────────
-- Headings use `fn_display_name` — this repo's single answer to "what does THIS viewer call that
-- thing", and the same labels the beat candidate whitelist and the scene surface already put in
-- front of the player every beat. `fn_perceived_name` alone was measured and rejected: it reads only
-- the world_genesis naming substrate, so on the fixture it returns NULL for the Sealed Note — a
-- thing the Player has observed and can obviously see — and the best topic on the page vanished into
-- the remainder. A fifth naming rule invented for one field would be the drift; if
-- fn_display_name's tail ever leaks, that is a fix for fn_display_name and all its callers.
-- A co-subject with no label at all still falls to the remainder rather than heading a null group.
--
-- THE VIEWER IS NEVER A TOPIC. They are a co-subject of very nearly every record they hold — they
-- were there — so without this they would become a spurious mega-topic on every page. Measured: an
-- unfiltered pass grouped two of Mara's records under "Player". The reader is not a subject of the
-- dossier they are reading.
--
-- ── ONE GROUP PER RECORD, AND A TOPIC IS A THING THAT RECURS ────────────────────────────────────
-- A record may be about several entities. It is filed under exactly ONE — the co-subject it shares
-- the most records with on this page, for this viewer — rather than repeated under each, because
-- the same paragraph printed twice under two headings is a new redundancy defect on a surface whose
-- whole complaint was redundancy. "What topics matter" is answered by recurrence, which is a fact
-- about the viewer's own knowledge and is fully deterministic (ties break by label, then id).
--
-- ── THE SHAPE DOES NOT MOVE ─────────────────────────────────────────────────────────────────────
-- `group_key` / `group_label` / `items` are unchanged, and so are the item fields. What changes is
-- the VALUE of group_key ("event:<uuid>" → "subject:<uuid>") and what group_label means. No field is
-- added, removed or retyped, so actor_page/2, location_page/1 and artifact_page/1 all stay put —
-- adding a field would have been a breaking change, this is not one.
--
-- Contract notes for the frontend:
--   • The FIRST group may have a null `group_label`. It is the remainder — records about the page's
--     own subject and nothing else nameable — and it is deliberately unheaded so it
--     cannot be misread as content belonging to the heading above it. Its key is
--     'subject:' || <the page's own entity id>. It is emitted first for that reason; every other
--     group follows by recurrence (most records first), then recency, then id.
--   • Items inside a group stay in in-world chronological order, so a topic reads as it evolved.
CREATE OR REPLACE FUNCTION fn_collected_knowledge(
  p_world_id uuid,
  p_viewer_id uuid,
  p_target_id uuid
)
RETURNS json
LANGUAGE sql STABLE AS $$
  WITH about AS (
    SELECT v.perception_id,
           v.source_event_id,
           v.content,
           v.epistemic_type,
           v.valid_tick,
           v.confidence,
           ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps
      ON ps.perception_id = v.perception_id
     AND ps.entity_id = p_target_id
    JOIN canon_event ce
      ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  -- Candidate topics: the other things these records are about — never the page's own subject, and
  -- never the reader.
  named_cosubject AS (
    SELECT a.perception_id,
           ps.entity_id,
           fn_display_name(p_world_id, p_viewer_id, ps.entity_id) AS label
    FROM about a
    JOIN perception_subject ps
      ON ps.perception_id = a.perception_id
     AND ps.entity_id <> p_target_id
     AND ps.entity_id <> p_viewer_id
    WHERE fn_display_name(p_world_id, p_viewer_id, ps.entity_id) IS NOT NULL
  ),
  -- A topic is a thing that keeps coming up: how many of this page's records each candidate shares.
  recurrence AS (
    SELECT entity_id, label, count(DISTINCT perception_id) AS n
    FROM named_cosubject
    GROUP BY entity_id, label
  ),
  -- Exactly one topic per record — its strongest, deterministically.
  filed AS (
    SELECT DISTINCT ON (nc.perception_id)
           nc.perception_id, nc.entity_id, nc.label
    FROM named_cosubject nc
    JOIN recurrence r ON r.entity_id = nc.entity_id
    ORDER BY nc.perception_id, r.n DESC, nc.label, nc.entity_id
  ),
  items AS (
    SELECT coalesce(f.entity_id, p_target_id) AS group_entity,
           f.label AS group_label,
           a.valid_tick AS sort_tick,
           a.perception_id,
           json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',            fn_compendium_decay(p_world_id, a.valid_tick, a.in_world_label),
             'source',           json_build_object(
                                   'epistemic_type', a.epistemic_type,
                                   'source_event_label', a.in_world_label
                                 )
           ) AS item
    FROM about a
    LEFT JOIN filed f ON f.perception_id = a.perception_id
  ),
  grouped AS (
    SELECT i.group_entity,
           max(i.group_label) AS group_label,   -- one label per group; NULL for the remainder
           count(*) AS n,
           max(i.sort_tick) AS latest_tick,
           coalesce(
             json_agg(i.item ORDER BY i.sort_tick, i.perception_id),
             '[]'::json
           ) AS group_items
    FROM items i
    GROUP BY i.group_entity
  )
  SELECT coalesce(
           json_agg(
             json_build_object(
               'group_key',   'subject:' || g.group_entity::text,
               'group_label', g.group_label,
               'items',       g.group_items
             )
             -- Unheaded remainder first (a heading-less block between two headed groups reads as
             -- theirs), then what recurs most, then what is most recent, then id.
             ORDER BY (g.group_label IS NOT NULL), g.n DESC, g.latest_tick DESC, g.group_entity
           ),
           '[]'::json
         )
  FROM grouped g;
$$;

-- migrate:down

-- Restores the SPEC-029 (#40) event-keyed grouping verbatim.
CREATE OR REPLACE FUNCTION fn_collected_knowledge(
  p_world_id uuid,
  p_viewer_id uuid,
  p_target_id uuid
)
RETURNS json
LANGUAGE sql STABLE AS $$
  WITH about AS (
    SELECT v.perception_id, v.source_event_id, v.content, v.epistemic_type,
           v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps
      ON ps.perception_id = v.perception_id AND ps.entity_id = p_target_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT a.source_event_id, a.in_world_label, a.valid_tick AS sort_tick, a.perception_id,
           json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',            fn_compendium_decay(p_world_id, a.valid_tick, a.in_world_label),
             'source',           json_build_object('epistemic_type', a.epistemic_type,
                                                   'source_event_label', a.in_world_label)
           ) AS item
    FROM about a
  ),
  grouped AS (
    SELECT i.source_event_id,
           max(i.in_world_label) AS group_label,
           max(i.sort_tick) AS group_sort_tick,
           coalesce(json_agg(i.item ORDER BY i.sort_tick, i.perception_id), '[]'::json) AS group_items
    FROM items i
    GROUP BY i.source_event_id
  )
  SELECT coalesce(
           json_agg(
             json_build_object('group_key', 'event:' || g.source_event_id::text,
                               'group_label', g.group_label,
                               'items', g.group_items)
             ORDER BY g.group_sort_tick DESC, g.source_event_id
           ),
           '[]'::json)
  FROM grouped g;
$$;
