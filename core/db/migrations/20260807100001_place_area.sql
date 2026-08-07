-- migrate:up

-- Rung 1 (the Journey ladder) — PLACES GET AN AREA.
--
-- Founder ruling R12 (2026-08-07): the descriptive attrs.extent {"w","h"} box is RETIRED, not extended
-- ("that was a cheap solution… an area is the more real and even 'drawable if needed'"). Nothing in SQL
-- ever read it (fn_distance and friends work off coordinates + the parent edge; the seed itself called it
-- "descriptive"), so there is no dual form and no compatibility shim: attrs.area replaces it outright.
--
-- attrs.area = {"points":[{"x":…,"y":…}, …]} (≥3), an ordered outline in the place's OWN frame — the same
-- frame its attrs.coordinates live in, i.e. the parent's. Optional: a place with no area is a point and
-- contains nobody, which is every room that ships today.
--
-- Containment is a MEASUREMENT recomputed at ask time, never a stored `contains` column — a stored answer
-- rots the moment a place moves or grows (the silent-corruption class §0 refuses).

-- ── fn_area_polygon: attrs → polygon. NULL when there is no area or fewer than 3 points (2 points is a
--    line, not a footprint, and must not silently become a degenerate polygon). STABLE, pure.
CREATE FUNCTION public.fn_area_polygon(p_attrs jsonb) RETURNS polygon
LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
    WHEN jsonb_array_length(COALESCE(p_attrs->'area'->'points', '[]'::jsonb)) >= 3
    THEN (
      SELECT ('(' || string_agg('(' || (pt->>'x') || ',' || (pt->>'y') || ')', ',' ORDER BY ord) || ')')::polygon
      FROM jsonb_array_elements(p_attrs->'area'->'points') WITH ORDINALITY AS t(pt, ord)
    )
    ELSE NULL
  END;
$$;

-- ── fn_place_at: which child of p_frame contains this point? The SMALLEST such area wins, so a square
--    inside a region resolves to the square. NULL = the point is inside nothing — the open road.
--
--    FRAME-SCOPED BY CONSTRUCTION (design §4.5): a place's coordinates are expressed in its PARENT's
--    frame, so a region and a room inside it are measured in different frames and cannot be compared in
--    one call. Callers pass the frame they are travelling in. Resolving THROUGH frames needs coordinate
--    transforms and belongs to the deferred spatial engine (SPEC-018) — do not add it here.
--
--    Postgres has NO area(polygon) overload (only box/circle/path) — cast to path first. area(path) is
--    already winding-independent in practice, but abs() is kept as a defensive belt: winding order is
--    authored data and both directions are legal outlines, so this must never depend on which way an
--    outline was walked.
CREATE FUNCTION public.fn_place_at(p_world_id uuid, p_frame uuid, p_point jsonb) RETURNS uuid
LANGUAGE sql STABLE AS $$
  SELECT ls.entity_id
  FROM location_state ls
  WHERE ls.world_id = p_world_id
    AND (ls.attrs->>'parent_location_id')::uuid = p_frame
    AND fn_area_polygon(ls.attrs) IS NOT NULL
    AND fn_area_polygon(ls.attrs) @> point((p_point->>'x')::float8, (p_point->>'y')::float8)
  ORDER BY abs(area(fn_area_polygon(ls.attrs)::path)) ASC, ls.entity_id ASC
  LIMIT 1;
$$;

-- migrate:down

DROP FUNCTION IF EXISTS public.fn_place_at(uuid, uuid, jsonb);
DROP FUNCTION IF EXISTS public.fn_area_polygon(jsonb);
