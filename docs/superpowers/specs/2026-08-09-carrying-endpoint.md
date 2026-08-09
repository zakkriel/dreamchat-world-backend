# `GET /worlds/{w}/carrying` — the Carrying overlay (2026-08-09)

Closes the last unmet item of the frontend-needs doc (§4) and chunk 4's remaining input.
Spec'd in `mvp_slice_and_bridge.md` §4.1; product in
`docs/10_prds/compendium/prd_compendium_artifacts_and_carrying.md`.

## Inputs read before implementation

- `mvp_slice_and_bridge.md` §4.1 (the one-line contract: "Carry States of the user-controlled Actor
  only") and §4 principles (B-1, I-3, B-5).
- Artifacts & Carrying PRD: AC#1 (Compendium ≠ inventory), AC#2 (perception-bound), AC#3 (Carry
  State derived, `last_confirmed_tick`, decay language), the Carrying Overlay Projection field list,
  and the non-goals — "Carrying for NPCs", no slots/grids/encumbrance UI.
- `fn_apply_carry_change`, `apply_event`'s `ObjectRelocated` branch, `state_mutation` (Master DDL
  doc 03 §1.3), `fn_display_name`, `fn_compendium_latest_fact`, `fn_compendium_decay`.
- `core/api/beathandler.go` `payload` — the existing precedent for reading `contained_by` for the
  viewer's own possessions and labelling with `fn_display_name` (quoted in SPEC-030's status).

## Design call 1 — the carrier is the viewer, structurally

`fn_carrying(p_world_id, p_viewer_id)` takes **no carrier argument** and the route has no carrier
segment. "Show me what that NPC is carrying" is therefore not expressible — the PRD non-goal is a
property of the signature rather than a check a later change can forget. In play mode the viewer is
`world.player_entity_id` (SPEC-028) and `?viewer=` is ignored outside debug, so a client cannot ask
for someone else's pockets even by trying. Pinned by `TestCarrying_PlayModeIgnoresAViewerOverride`.

## Design call 2 — why this reads canon, and why the wall is intact

Possession of your own belongings is not something you hold a perception *about*; it is a fact you
are living. Running this list through `fn_entity_visible` would hide a viewer's own pocket from
them. What the wall governs is what the viewer KNOWS, and every knowledge-bearing field stays
viewer-scoped:

- `label` — `fn_display_name` (the viewer's own naming, descriptor fallback), the same function the
  beat candidate whitelist already uses for exactly these rows.
- `quick_inspect_preview` — `fn_compendium_latest_fact`, which reads `fn_visible_perceptions` and
  nothing else. Pinned by a pgTAP negative: a perception **Mara** holds about the crate does not
  appear in **Kade's** preview even when Kade is the one carrying it.

No canon row crosses the boundary. What crosses is ids, the viewer's labels, and the viewer's own
knowledge (B-1, I-3).

## Design call 3 — the ledger, not `artifact_state`

`trg_sm_project` writes `artifact_state.attrs.contained_by` **from** `state_mutation`, so both carry
the same fact — but only the ledger carries the provenance AC#3 requires: which accepted event last
confirmed this containment, at which in-world tick, under which in-world label. Read from the
projection, `last_confirmed_tick` would have to be invented or shipped null.

Current containment is the newest applied `attrs.contained_by` mutation per entity, ordered by the
domain key `(valid_from_tick, valid_from_seq)` — never `recorded_at`, which is transaction time
(B-5, ADR-034).

Putting a thing down needs no tombstone and no special case: `ObjectRelocated` always names a
destination (`state_mutation.new_value` is `NOT NULL`, so a "contained by nothing" edge is not even
writable), and a destination that is not the viewer simply stops rooting the chain at them.

## Design call 4 — nesting is included, because the engine already means it

Everything whose containment chain **roots at the viewer** appears, with `container` naming the
immediate holder (`null` = directly on you). This is not an answer to the PRD's open
container-semantics question (pouch-inside-bag as a UI affordance); it is consistency with the
engine's own definition of what a carrier carries. `fn_apply_carry_change` climbs `contained_by` to
the root carrier and charges the **whole subtree** to that actor's `carried_weight`. An overlay
showing only the top layer would say you are not carrying weight the same world is penalising you
for. Depth cap 64 — the same `contained_by` cycle fail-safe `fn_apply_carry_change` uses (I-4).

## Design call 5 — `state`, and what is deliberately not in the payload

The PRD's Carry State enum is `carried|worn|held|packed|stored_elsewhere|lost|unknown`. The world
records exactly one distinction — `contained_by`: in your possession, or not — so **`carried` is the
only value it can honestly produce**. `worn`/`held`/`packed` need a signal that does not exist;
`stored_elsewhere`/`lost`/`unknown` describe things that are *not* on you and can never appear on
this surface.

`state` is typed as a plain `string` and is **not enum-pinned**, on purpose: when a real signal
lands the value set widens in place and costs the frontend no re-pin — *provided it treats an
unrecognised value as opaque rather than switching exhaustively*. Pinning one value today would
guarantee a version bump tomorrow for a change that breaks nobody.

Two of the PRD's four projection fields are absent rather than stubbed:

- `contextual_actions` — presentation, and presentation is the frontend's (D-7).
- `open_full_artifact_link` — each entry's `id` **is** the link. A backend emitting
  `/worlds/…/artifacts/…/page` would be hardcoding someone else's URL space.

## Published contract

`core/api/schema/carrying.v1.schema.json`, `$id: carrying/1`, `additionalProperties: false`
throughout, per-field rationale in the descriptions.

```json
{
  "schema_version": "carrying/1",
  "world_id": "<uuid>",
  "viewer_id": "<uuid>",
  "carried": [
    {
      "id": "<artifact uuid>",
      "label": "Sealed Note (gray wax)",
      "state": "carried",
      "container": null,
      "last_confirmed_tick": 40,
      "quick_inspect_preview": null,
      "decay": { "stale": true, "last_confirmed_label": "Scene" }
    }
  ]
}
```

`container` when non-null is `{ "id": "<uuid>", "label": "<string>" }`. Order: directly-carried
entries first, then alphabetically by the viewer's own label, id as the final tiebreak. `carried` is
always an array — empty means "you are carrying nothing", which is an answer, not a missing page,
so this endpoint has **no NULL→404 branch** unlike the page endpoints.

## Found by hand-driving, fixed at the source

A world id nobody ever minted returned `500 viewer resolution failed` — on **every** world-scoped
endpoint, not just this one. `ResolveViewer` let `pgx.ErrNoRows` fall through into the generic error
branch, so an ordinary client typo reported a broken server. That is the same conflation the
existing `errNoPlayerInWorld` comment was written to prevent for the adjacent case. Now
`errNoSuchWorld`, 404, and the eight-line branch copied into six handlers collapsed into one
`writeNoViewer`. Regression-covered at the source and at three unrelated endpoints.

`newRouter` was also extracted out of `main()`: a handler that works perfectly and was never added
to the route list is a 404 in production and a green suite. `TestRouter_ServesCarrying` drives the
composed router, and every future endpoint inherits the check.

## Test coverage

- `core/db/tests/25_carrying_test.sql` (14): envelope and the empty case; label/state/container/
  `last_confirmed_tick`/`decay` on a real carried thing; **Kade's overlay never shows Mara's cellar
  key** and vice versa; nesting names its container; handing a thing over moves it between overlays;
  putting it down into a room removes it; the perception wall on `quick_inspect_preview`.
- `core/api/carryinghandler_test.go` (6): the world's own player resolves; play mode ignores
  `?viewer=`; the debug override swaps the carrier and nothing else; empty is 200 not 404; GET only;
  the composed router serves the route.
- `core/api/viewer_test.go`: unknown world → `errNoSuchWorld`, and 404 at carrying/timeline/index.
- `ci/gen_payloads.sh`: `carrying/1` payloads for both fixture viewers **and** for the play world's
  Kade and Mara. The fixture pair carry nothing, so on their own they would only exercise the empty
  envelope and `carried[*]` would ship unvalidated — a coverage hole that reads green. Contract
  coverage is now **12/18**.
