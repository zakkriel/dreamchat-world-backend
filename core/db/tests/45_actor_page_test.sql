BEGIN;
SELECT plan(7);

-- (1) coherent view: Mara's perceived name resolves for Player
SELECT is(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->>'perceived_name',
  'Mara', 'perceived_name = Mara on Mara page');
-- (2) schema_version present
SELECT is(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->>'schema_version',
  'actor_page/2', 'payload carries schema_version actor_page/2');
-- (3) current_synthesis is a deterministic composition of the viewer's OWN held perceptions
--     (SPEC-029). Asserting the exact text, not merely NOT NULL: "not null" would still pass if the
--     lens leaked canon, invented prose, or rendered a raw tick as a display label (B-5) — all three
--     of which this pins out. Newest-first, newline-joined, no ordinals, no time label.
SELECT is(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->>'current_synthesis',
  E'I told Mara the mayor keeps a hidden ledger\nMara listened intently and seemed unsettled',
  'current_synthesis composes only the held perceptions, newest first, with no invented label');
-- (4) NO relationship fields anywhere in the payload (B-3/B-4, AC#7/#8)
SELECT ok(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')::text NOT ILIKE '%relationship%',
  'no relationship field in payload (B-3)');

-- (5) NON-VACUOUS LEAK TEST — same row (about-Mara, dca7…a01), same page (Mara), both halves:
--     PRESENT for viewer=Player ...
SELECT ok(
  EXISTS (
    SELECT 1
    FROM json_array_elements(
           fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                         'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
         json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000a01'),
  'about-Mara private perception PRESENT for viewer=Player');
--     ... and ABSENT for viewer=Jonas. If (5a) ever went vacuous (empty page), (5b) alone could
--     pass on nothing — so BOTH are asserted on the SAME perception_id. This pair is the gate.
SELECT ok(
  NOT EXISTS (
    SELECT 1
    FROM json_array_elements(
           fn_actor_page('11111111-1111-1111-1111-111111111111','cccccccc-cccc-cccc-cccc-cccccccccccc',
                         'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')->'actor'->'collected_knowledge_groups') g,
         json_array_elements(g->'items') it
    WHERE it->>'perception_id' = 'dca70000-0000-0000-0000-000000000a01'),
  'about-Mara private perception ABSENT for viewer=Jonas (I-3, fails loud on leak)');

-- (6) the page never surfaces a canon row — the source-event summary text never leaks (B-1)
SELECT ok(
  fn_actor_page('11111111-1111-1111-1111-111111111111','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
                'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')::text NOT LIKE '%P tells M%',
  'canon_event.summary never appears in the payload (B-1)');

SELECT * FROM finish();
ROLLBACK;
