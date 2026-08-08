-- migrate:up

-- FOUNDER-RULED (2026-08-08): when two things a viewer can see wear the SAME label, the label gains
-- PERCEIVED DETAIL so the player can tell them apart — "the hooded figure by the bar" against "the
-- hooded figure by the crate". No chosen-candidate-id on the beat, no positional-language parsing:
-- the player names what they see, in words the world already gave them.
--
-- Why this is a SET function and not a change to fn_display_name. A collision is a property of a
-- GROUP, not of an entity: "a hooded figure" is a perfectly good label right up until a second one
-- walks in. fn_display_name stays pure and per-entity (known name → descriptor → canonical); this
-- sits above it and only speaks when the group forces it to. Nothing is renamed when nothing collides,
-- so the common case is untouched and no existing caller changes meaning.
--
-- PERCEPTION-BOUND (B-1, I-3). The detail is the nearest ARTIFACT IN THE SAME ROOM — the co-located
-- set the viewer already perceives and can already name (the same set the candidate whitelist offers,
-- RULINGS-2026-07-30 §1's "co-location IS perception"). The anchor is itself rendered through
-- fn_display_name, so the detail can never name something in the viewer's own words that the viewer
-- does not hold. No hidden truth, no canon read: an anchor is a thing in the room with you.
--
-- THE HONEST EDGE, ruled explicitly by the founder: if the detail does not actually distinguish —
-- both figures nearest the same crate, or the room holds nothing to stand near — the label is LEFT
-- PLAIN and both entries read identically. That is the fiction, not a bug: two people you genuinely
-- cannot tell apart look the same, and inventing "the first" or "the taller" would be the system
-- asserting something the viewer cannot see. The frontend renders the collision as it is; the player
-- experiences exactly what the character experiences, which is uncertainty.

-- fn_perceived_anchor: the viewer's own name for the nearest artifact sharing this entity's room, or
-- NULL when the room offers nothing to stand near. Distance uses fn_distance so it measures in the
-- pair's own frame, exactly like every other spatial read.
CREATE OR REPLACE FUNCTION fn_perceived_anchor(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS text LANGUAGE sql STABLE AS $$
  WITH here AS (
    SELECT a.attrs->>'location_id' AS loc
      FROM actor_state a
     WHERE a.world_id = p_world_id AND a.entity_id = p_entity_id
  )
  SELECT fn_display_name(p_world_id, p_viewer_id, art.entity_id)
    FROM artifact_state art, here
   WHERE art.world_id = p_world_id
     AND here.loc IS NOT NULL
     AND art.attrs->>'location_id' = here.loc
     AND art.attrs ? 'coordinates'
   ORDER BY fn_distance(p_world_id, p_entity_id, art.entity_id), art.entity_id
   LIMIT 1;
$$;

-- fn_display_names_distinct: label every id the way its viewer would, then break ties with perceived
-- detail — but ONLY where the detail genuinely separates the group. A group whose anchors are all the
-- same keeps its plain label (the honest edge above).
CREATE OR REPLACE FUNCTION fn_display_names_distinct(p_world_id uuid, p_viewer_id uuid, p_ids uuid[])
RETURNS TABLE(entity_id uuid, label text) LANGUAGE sql STABLE AS $$
  WITH base AS (
    SELECT t.id AS entity_id,
           t.ord,
           fn_display_name(p_world_id, p_viewer_id, t.id) AS base_label,
           fn_perceived_anchor(p_world_id, p_viewer_id, t.id) AS anchor
      FROM unnest(p_ids) WITH ORDINALITY AS t(id, ord)
  ),
  -- Per-label aggregate rather than a window: Postgres has no count(DISTINCT …) OVER (…).
  spread AS (
    SELECT b.base_label,
           count(*)                 AS same_label,
           -- distinct anchors in this label's group; 1 (or 0) means detail cannot separate it
           count(DISTINCT b.anchor) AS distinct_anchors
      FROM base b
     GROUP BY b.base_label
  )
  SELECT b.entity_id,
         CASE
           WHEN s.same_label > 1 AND s.distinct_anchors > 1 AND b.anchor IS NOT NULL
             THEN b.base_label || ' by ' || b.anchor
           ELSE b.base_label
         END
    FROM base b
    JOIN spread s ON s.base_label IS NOT DISTINCT FROM b.base_label
   ORDER BY b.ord;  -- callers rely on input order (the beat's own candidate order)
$$;

-- fn_display_name reaches for a descriptor in actor_state ONLY, so an artifact's Tier-2 descriptor
-- was unreachable and every artifact fell through to its canonical registry name. Its own docstring
-- promises "the entity's DESCRIPTOR … else the canonical registry name (engine fallback; a seed lag,
-- never shown once descriptors are seeded)" — for artifacts the fallback was permanent, because the
-- lookup that would end it did not exist.
--
-- That was invisible while artifact labels only reached a model. It stops being invisible the moment
-- an anchor becomes player-facing prose: the ballast crate rendered a disambiguated label as "a
-- hooded figure by Ballast Crate" — a database row wearing a sentence. One more COALESCE branch, in
-- the same order the docstring already states.
CREATE OR REPLACE FUNCTION public.fn_display_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS text
LANGUAGE sql STABLE AS $$
  SELECT COALESCE(
    fn_perceived_name(p_world_id, p_viewer_id, p_entity_id),
    (SELECT ast.attrs->>'descriptor' FROM actor_state ast
      WHERE ast.world_id = p_world_id AND ast.entity_id = p_entity_id),
    (SELECT art.attrs->>'descriptor' FROM artifact_state art
      WHERE art.world_id = p_world_id AND art.entity_id = p_entity_id),
    (SELECT er.canonical_name FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_entity_id)
  );
$$;

-- migrate:down

DROP FUNCTION IF EXISTS fn_display_names_distinct(uuid, uuid, uuid[]);
DROP FUNCTION IF EXISTS fn_perceived_anchor(uuid, uuid, uuid);

-- Restore fn_display_name to its actor-only descriptor lookup.
CREATE OR REPLACE FUNCTION public.fn_display_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS text
LANGUAGE sql STABLE AS $$
  SELECT COALESCE(
    fn_perceived_name(p_world_id, p_viewer_id, p_entity_id),
    (SELECT ast.attrs->>'descriptor' FROM actor_state ast
      WHERE ast.world_id = p_world_id AND ast.entity_id = p_entity_id),
    (SELECT er.canonical_name FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_entity_id)
  );
$$;
