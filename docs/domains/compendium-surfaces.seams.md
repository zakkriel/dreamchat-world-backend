# compendium-surfaces · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** UX-1 · The compendium surfaces ·
**Parent bounded context:** Compendium & Play UX

A seam belongs to two domains, so it gets its own file. Each row declares an expectation — one side
owns a fact, the other consumes it and must not re-derive or re-decide it. Neighbour claims below
were cross-confirmed with the sibling writers 2026-08-27; the moderator reconciles residual drift.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Perception & Knowledge** (WE-3) | the visible set and the existence gate — `fn_visible_perceptions`, `fn_entity_visible` | Matches WE-3's provides-row ("every page, index and timeline read"): no surface reads canon for knowledge (`B-1`); hidden truth is absent from the payload, not hidden by the UI. This domain **never decides who perceives** and never re-derives visibility — every lens filters through WE-3's two functions and nothing else. |
| consumes | **The naming wall** (WE-4) | every name a surface prints — `fn_perceived_name` (index entries, dossier names) and `fn_display_name` (group headings, carrying labels) | The wall owns substitution; these surfaces never invent, re-resolve or retro-apply a name. A null `perceived_name` is normal and permanent. Bypassing `fn_display_name` to `entity_registry.canonical_name` is a `B-1` breach (trap shared with WE-4; receipt in `tech.md`). |
| consumes | **Canon spine** | committed rows, read-only: `canon_event.in_world_label` (joined via `source_event_id` for display labels) and the `state_mutation` containment ledger (`fn_carrying`'s provenance) | Consumers read committed rows, never insert into `canon_event` or `state_mutation` (`D-1`). `fn_carrying`'s canon read is deliberate and documented (migration `20260809090001`): possession is a lived fact, not a perception you hold about yourself; every knowledge-bearing field stays viewer-scoped. Ordering is `(valid_from_tick, valid_from_seq)`, never `recorded_at` (`B-5`). Page functions never read the `*_state` projection tables (WE-2's) — populating a page from `*_state` breaches the wall (SPEC-029); cross-confirmed with WE-2's writer. |
| consumes | **Time & clock** | `fn_world_now` (the decay horizon's clock) and authored display labels | Ticks order and compute; labels display; when no label is authored the honest answer is silence, never a manufactured "[Tick 51]" (`B-5`; `fn_compendium_current_synthesis`'s own comment). |
| consumes | **Play loop** (WE-7) | one transcript row per delivered beat — post-belt `[]beatMessage`, via `persistTranscript` | The loop decides WHEN (after the belts, before the result frame, `context.WithoutCancel`, never fails the beat — `beatsstream.go:633`); this domain owns the record's shape and the never-recomputed / never-retro-labelled rule. Cross-confirmed with WE-7's writer. |
| consumes | **Art & image seam** | `image_ref/1` via `fn_image_ref` — an asset id and a PATH, never a presigned URL; `null` is the ordinary state | Pulled, never pushed. The portrait is deliberately **not** perception-scoped (`schema.sql:844-849`); the page's existence gate still runs first, so "not perception-scoped" never means "not perception-gated". These surfaces never mint, persist or re-derive image URLs. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Frontend / presentation** (UX-3, `dream-weaver-visuals`) | the seven pinned payloads: `compendium_index/1`, `actor_page/2`, `location_page/1`, `artifact_page/1`, `timeline/1`, `transcript/2`, `carrying/1` | Consumed verbatim; the frontend never re-derives — no client-side regrouping (deciding two subjects are one topic is world truth), no day-structure parsing of `display_label`, no invented names, counts or verbs (`D-7`, `D-14`). Unpinned enums (`carrying.state`) are treated as opaque. **Only the transcript and carrying contracts are vendored+pinned in the frontend today** (versions: the `const PIN` block, `dream-weaver-visuals/src/api/index.ts`); the five compendium schemas are deliberately unvendored until a surface reads them (`dream-weaver-visuals/scripts/verify-contract.sh:15-27`) — cross-confirmed with UX-3's writer.. |
| provides | **Play surface** (UX-2) | the Carrying overlay payload and the transcript history the play route renders | Carrying sits in the play column but its shape is this domain's: no carrier argument exists, so the surface cannot be pointed at an NPC. History and live prose share one segment shape (`segments` = the live `narration` frame's), so one renderer serves both — a divergence is a defect on this side. |

## The seams that do not exist

Name them, because this is where an agent will otherwise improvise.

- **Inspect and Known lens endpoints.** Not built; Known "would duplicate the dossier — a second
  path for one job" (`digest/S11` Topic 19, `D-14`). Do not feed the lenses from compendium
  payloads as a workaround; the decision of what those lenses carry is open, not implied.
- **A world-home / world-summary endpoint.** The frontend is told not to design one in silently
  (`digest/S11` Topic 13). A summary of "the world as this character knows it" would be a new
  perception-bound projection — a WE-3 + UX-1 decision, not a frontend fill-in.
- **Index enrichment.** The index is an id and a nullable name. Thumbnails, subtitles, counts are a
  raise-with-the-backend case (`digest/S11` Topic 15); inventing them client-side violates the
  frontend's own law, and adding them here is a `compendium_index/2` version event (see
  `tech.md` §Open questions).
- **Relationship data crossing any seam.** There is no relationship payload field anywhere in these
  schemas, and that absence is `B-3` enforced by shape, not a gap.
