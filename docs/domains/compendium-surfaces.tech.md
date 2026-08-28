# compendium-surfaces · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** UX-1 · The compendium surfaces ·
**Parent bounded context:** Compendium & Play UX

This file holds how the domain is built — storage, the read and write paths, validation, traps.
`compendium-surfaces.product.md` holds what it means; `compendium-surfaces.seams.md` holds what
crosses its boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate
by grep before relying on one.

---

## Architecture: thin readers over SQL lenses

Every surface is a Go handler that resolves the viewer, calls one SQL function, and writes the
JSON it returns (`ADR-P017`). The handlers decide nothing epistemic.

| Route (GET) | Handler | SQL lens | Pinned schema |
|---|---|---|---|
| `/worlds/{w}/compendium/{actors,locations,artifacts}` | `core/api/indexhandler.go` | `fn_compendium_index_json` | `compendium_index/1` |
| `…/compendium/{kind}/{id}/page` | `core/api/pagehandler.go` | `fn_actor_page` / `fn_location_page` / `fn_artifact_page` | `actor_page/2`, `location_page/1`, `artifact_page/1` |
| `…/compendium/timeline?before_tick=` | `core/api/timelinehandler.go` | `fn_timeline` | `timeline/1` |
| `/worlds/{w}/transcript?before=&limit=` | `core/api/transcript.go` | `fn_transcript` | `transcript/2` |
| `/worlds/{w}/carrying` | `core/api/carryinghandler.go` | `fn_carrying` | `carrying/1` |

JSON Schemas: `core/api/schema/` (same basenames). All are `additionalProperties: false` with a
pinned version — an added field is a breaking change however additive it looks; the version moving
IS the notification (`digest/S13a` Topic 16).

- **Viewer resolution** (`core/api/viewer.go`, shared with every world-scoped surface, not owned
  here): the viewer is the world's own player; `?viewer=` is honored only in debug — `C-4`'s
  mechanical form. No-such-world and no-player-yet are both 404s.
- **The existence 404**: a page function returns SQL `NULL` when the entity is not in the viewer's
  existence set; the handler maps it to 404, indistinguishable from a nonexistent id
  (`core/api/pagehandler.go:63`; the gate is `fn_actor_page`'s first line, `schema.sql:825`).
- `actorpage.go`'s `NewActorPageHandler` is a wrapper used **only by tests**; `main.go:45`
  registers `NewPageHandler` for actors directly.

## Storage

This domain owns exactly one table: **`transcript_entry`** (`entry_no` identity PK and pagination
cursor, `viewer_id`, `in_world_tick`, `stated`, `segments jsonb`, `halt_reason`, `journey`;
migration `20260809090008`). Everything else is projection — recomputable, no storage of its own.

## Grouping — the one algorithm this domain owns

`fn_collected_knowledge` (`schema.sql:1204`) groups by **subject** — `group_key =
'subject:<uuid>'`, headings via `fn_display_name`. Third generation; generation 2 (per source
event, SPEC-029) failed measurably — *"Mara's dossier rendered TWENTY-FIVE groups, every one of
them headed 'Arrival'"* — because one group per event is one group per log line (migration
`20260809090002`, which carries the full four-axis rejection). Load-bearing rules inside it:

- **The viewer is never a topic** (`ps.entity_id <> p_viewer_id`, `schema.sql:1233`) — without it
  the reader becomes a spurious mega-topic on every page.
- **The unheaded remainder is emitted first** (`ORDER BY (g.group_label IS NOT NULL)`,
  `schema.sql:1289`) — a heading-less block between two headed groups reads as belonging to the one
  above it.
- **One group per record**, filed under the co-subject it recurs with most; items inside a group
  stay in in-world chronological order.

`fn_compendium_current_synthesis` (`schema.sql:1304`) is `B-9`'s mechanical form: newest 3 held
perceptions, newline-joined, no LLM, no ordinals, **no manufactured time label** — an unlabelled
event renders as silence, not "[Tick 51]" (`B-5`).

`fn_compendium_decay` (`schema.sql:1332`): `stale = elapsed in-world ticks > 72`
(`fn_compendium_decay_horizon_ticks`, one named home for the threshold). Decay wording, never
visibility.

## Transcript — the second write path

`persistTranscript` (`core/api/transcript.go:31`) is the only write this domain performs. Called by
the play loop post-belt (`seams.md`); stores rendered **text**, not ids — nothing exists for a
later render to re-resolve, deliberately. Migration `20260809090008`: *"The one thing this table
must never grow is a 'current label' column, because the moment it exists someone will use it to
'fix' the old entries."* Failure never touches the beat. `entry_no` orders and paginates; the tick
cannot (a QUERY beat advances no tick). A malformed `before` cursor is a 400, never silently page 1.

The frozen-label pin: `core/db/tests/27_transcript_test.sql:65` asserts an older entry still reads
*"the muscle by the bar"* after the viewer learns the name;
`TestTranscript_StoresWhatWasDeliveredAndServesItBack` (`core/api/transcript_test.go`) pins the
round trip.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-P017` | Handlers are thin readers; lenses live in SQL. | Epistemic logic in Go splits the wall into two enforcement points that drift. |
| `D-7` | The backend ships data, never presentation — no routes, no ordinals, no action verbs in payloads. | `carrying/1` omitting `contextual_actions` is this rule, not an oversight. |
| `D-14` | Data is tagged by kind; the frontend renders by catalog. Unpinned enums (`carrying.state`, theme words) widen in place; unknown values degrade, never fail. | Enum-pinning `state` costs a re-pin the day `worn` lands. |
| `SPEC-008` / `ADR-035` | About-ness (`perception_subject`) is engine-written and is the only axis that is genuinely a topic. | Grouping by anything else re-runs the generation-2 failure. |
| `B-9` | Synthesis is deterministic (see §Grouping). | Regeneration drift. |

### What you may not decide alone

1. **Adding a field to a pinned payload** — a breaking change by construction; the frontend pins
   the version string exactly.
2. **Changing the grouping axis or group order** — a founder-adjacent ruling with a measured
   failure behind it (migration `20260809090002`).
3. **Making the transcript re-derivable in any way** — the never-retro-label rule is epistemic law,
   not a storage choice.
4. **Widening `?viewer=` beyond debug** — that is `C-4`'s enforcement point.

## Validation for this domain

pgTAP in `core/db/tests/`: `18_compendium_index*`, `19_actor_page_gate*`, `20_location_page*`,
`21_artifact_page*`, `22_timeline*`, `24_compendium_lenses*`, `25_carrying*`,
`26_knowledge_grouping*`, `27_transcript*`, `45_actor_page*`. Go:
`cd core/api && go test -run 'Compendium|ActorPage|ArtifactPage|Timeline|Transcript|Carrying' -count=1 .`
— always `-count=1` (the suite is seed-dependent; a cached pass is a stale pass).

**What counts as evidence here:** a `B-1` test must assert **absence** — that the other viewer's
payload does *not* contain the secret (`compendium_index_test.go` asserts Jonas gets no note;
`compendium_test.go` asserts the 404 is indistinguishable from a fabricated id). A test that only
asserts presence for the entitled viewer passes with the wall deleted.

**What counts as ceremony here:** existence-shaped assertions like `19_actor_page_gate_test.sql`'s
`IS NOT NULL` — honest as a gate check, vacuous about content. Do not count such a pass as coverage
of grouping, decay or synthesis; `26_knowledge_grouping*` and `24_compendium_lenses*` are where
those live.

`make reset` destroys the dev volume and must never be run (WE-3's package, same rule, one home:
`perception-and-knowledge.tech.md` §Validation).

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **A digest claim can be stale against the tree.** S11 BLOCK-2 says dossier decay is hardcoded `stale=false`; that was `20260615090001:70,181,207` and is superseded — current lenses call `fn_compendium_decay` (`schema.sql:1262,2914`). | Verified 2026-08-27 by grep; friction-logged. |
| **Bypassing `fn_display_name` to `entity_registry.canonical_name` is a `B-1` breach.** Label fields have no belt of their own; protection is viewer-relativity at source. | WE-4's package; `core/api/namingwall_test.go:166`. |
| **The Carrying→dossier 404 is honest, not broken.** You can carry a thing you know nothing about; the Compendium lists known, Carrying lists held. | `digest/S11` Topic 15 (the Sealed Note case); `carryinghandler.go:22-24`. |
| **`carrying/1.state` is always `'carried'` today** — the world records one distinction (`contained_by`); worn/held/packed need a signal that does not exist. | Migration `20260809090001`; `digest/S13b` Topic 40. |
| **`perceived_role` is hardcoded `NULL`** — the slot exists in the schema and no code populates it. Designing against it renders empty. | `schema.sql:834` and its own comment. |
| **The actor portrait is deliberately not perception-scoped.** Two viewers who know an actor by different names see the same face; the existence gate above it still decides whether the page renders at all. "Fixing" it breaks a documented decision. | Comment block `schema.sql:844-849`; same ruling as `imagehandler.go:669-672`. |
| **`confidence` is never written (always 1.0) and `disputed`/`mistaken`/`confirmed` are never produced** — a treatment demanding them demands what the engine cannot emit. | `git show 5697b7e:docs/30_architecture/world_model/01_engine_capability_audit.md` (history-only; WE-11's tech.md §Where the corpus lives is the one home for that fact). |

## Open questions

1. **`perceived_role`** — populate from a real signal, or delete the slot from `actor_page/2`?
   Currently a permanent `NULL` the frontend is told to design around.
2. **Location page stubs** — `part_of` always `null`; `known_areas_inside` hardcoded `[]`
   (`digest/S11` Topic 15). Next, or parked?
3. **Artifact dossier's `type`/`location`/`holder`** — `null` in every payload the backend can
   currently produce. Same question.
4. **Index enrichment** — the index is an id and a nullable name; the frontend must raise, not
   invent, thumbnails/subtitles. Is a richer `compendium_index/2` wanted?
5. **`45_actor_page_test.sql` vs `19_actor_page_gate_test.sql`** — two suites over one function,
   overlapping assertions; merge candidates, not this package's call.
