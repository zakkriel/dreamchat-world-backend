# relationships · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-9 · Relationships, modelled and never surfaced ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the write and read paths, validation, traps.
`relationships.product.md` holds what it means; `relationships.seams.md` holds what crosses its
boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate by
grep before relying on one.

---

## Storage

One table. **`relationship_state`** (`core/db/migrations/20260610090005_projections.sql:15`; DDL at
`schema.sql:4082`): `world_id · a_id · b_id` (composite PK) `· attrs jsonb · dirty · last_event_id`.

- It is a **projection**, created alongside `actor_state`/`location_state`/`artifact_state`, under
  the same grant wall: `REVOKE ALL … FROM PUBLIC`, maintainer-only writes (`I-7`,
  `20260610090005_projections.sql:26-28`).
- **No `updated_at` column, on purpose** — no volatile column to exclude from the `I-1` replay
  comparison (pgTAP-pinned: `core/db/tests/10_schema_test.sql:28-30`).
- The PRD specified a three-structure internal model (Relationship Canon / Relationship Perception /
  Relationship Perception Record — `digest/S05_the_prds.md` topic 6). **None of it exists in DDL**
  (`git grep -i relationship_perception` returns nothing). The one table above is the whole model.
  Both sides recorded; whether the PRD model is still the target is Open question 2.

## The write path that no-ops

`apply_mutation`'s `entity_kind = 'relationship'` arm is a **documented no-op stub**
(`schema.sql:452-454`): *"SPEC-001: doc 03 does not define mutation->(a_id,b_id) addressing. NO-OP
stub in 0A."* pgTAP asserts the no-op leaves zero rows (`core/db/tests/30_apply_mutation_test.sql:27-34`).

Consequence, plainly: **nothing in production writes `relationship_state`** — no migration, seed,
template, or Go file inserts a row (`git grep -ln relationship_state` — the only INSERT is a test
fixture, `core/db/tests/103_gather_slice_test.sql:38`). The founder's ruling says the relationship
*"is logged"* (`product.md`); today the table it would be logged into is empty. Open question 3.

A ruled `EntityCreated` cannot mint one either: `entityKindSet` in `core/api/ruling.go:111-113`
closes the set to actor/artifact/location — *"Not 'relationship' (no coordinate addressing)"*.

## The read path

- **`gather_slice(world, ids[])`** (`schema.sql:3344-3349`) builds a `relationships` array — every
  `relationship_state` row touching an actor in the slice — into the JSON the referee (adjudication
  seat) receives. Caller: `core/api/orchestrator.go:1439`. This is the mechanism by which an NPC
  would *"play according to"* the log.
- Relationship counterparty UUIDs in the slice are **fetched context, never grounded entities** — the
  orchestrator deliberately does not whitelist them as ids the LLM may claim
  (`core/api/orchestrator.go:1449-1452`).
- No compendium lens reads the table; the actor page is tested to carry no relationship field at all
  (`core/db/tests/45_actor_page_test.sql:23-27`, `B-3`).
- Replay includes it: `replay_0A()` snapshots, truncates, rebuilds and EXCEPT-compares
  `relationship_state` with the other three projections (`schema.sql:3653-3691`).

## Where the code lives

Verified against `git ls-files` / `git grep`, 2026-08-27 — **this domain owns no file exclusively**:

- storage + grant wall: `core/db/migrations/20260610090005_projections.sql` (shared with WE-2)
- write stub: `core/db/migrations/20260610090006_apply_mutation_and_triggers.sql` (shared with WE-2)
- reader: `core/db/migrations/20260724100001_gather_slice.sql`, `…110001_gather_slice_multiactor.sql`
- pgTAP receipts: `10_schema_test.sql`, `30_apply_mutation_test.sql`, `70_determinism_guards_test.sql`,
  `45_actor_page_test.sql`, `103_gather_slice_test.sql`

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `I-7` | Projections are written only by the maintainer, through the single write path. | A direct INSERT into `relationship_state` bypasses the grant wall — and dies at the next replay (below). |
| `I-1` | Replay invariance covers `relationship_state`; hence no `updated_at` column. | A row no event can regenerate reads as replay drift. |
| `B-5` | Append-only time; no mutable domain timestamps. | Adding `updated_at` "for convenience" breaks `I-1`'s comparison and the schema test. |

### What you may not decide alone

1. **The `SPEC-001` addressing shape** — how a mutation names an `(a_id, b_id)` row. Its expected
   outcome is a proposed new ADR, informed by Phase-1 evidence (`docs/open-spec-items.md` §SPEC-001).
2. **Writing `relationship_state` by any path other than `apply_mutation`.** That re-decides `I-7`.
3. **Surfacing anything from this table.** That reopens `B-3` (see `product.md`).

## Validation for this domain

pgTAP (in `core/db/tests/`): `10_schema_test` (shape, no `updated_at`) · `30_apply_mutation_test`
(SPEC-001 no-op) · `70_determinism_guards_test` (table empty in 0A) · `103_gather_slice_test`
(relationships array populated from a fixture) · `45_actor_page_test` (no relationship field in the
page payload). `make reset` destroys the dev volume — never run it; see
`perception-and-knowledge.tech.md` §Validation, which owns that warning.

**What counts as ceremony here — the characteristic vacuous test, named by its own file:**
`70_determinism_guards_test.sql:11-13` asserts zero rows and calls itself *"intentional vacuous
satisfaction"*. Every guard on this domain currently passes because the table is empty. The moment a
writer lands (SPEC-001), all of them assert nothing about correctness — only about emptiness — and
must be rewritten, not extended.

**What counts as evidence here:** the domain has no behaviour yet, so evidence is *absence* twice
over — zero rows (writer stub) and zero payload fields (`B-3`). Both are pinned by the tests above.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The table looks writable; the write path no-ops.** A "fix" that inserts directly violates `I-7` and surfaces later as `I-1` replay drift, not at commit time. | `schema.sql:452-454`; `30_apply_mutation_test.sql:27-34`; replay compare `schema.sql:3687-3691`. |
| **The `B-3` guard bans the substring, not the field.** `fn_actor_page(...)::text NOT ILIKE '%relationship%'` fails on *fiction content* containing the word — an NPC saying "our relationship" in a perception's content breaks the actor-page suite. | `core/db/tests/45_actor_page_test.sql:24-27`, read 2026-08-27. |
| **The domain's product spec is a deleted file.** `docs/10_prds/compendium/parked_relationships.md` (amendment banner, banned-surfaces list, internal model) was deleted in the 2026-08-27 consolidation. The register row `B-3` and the digests are what remain. | backend commit `88486c1` (`git show 88486c1 --name-status`); `workspace:ADR-W006`. |
| **Two banned-surface lists, not one.** The PRD's "Do not add" block carries six items; the Actors PRD carried three, differently worded. Treating either as exhaustive is a digest-flagged inference. | `digest/S05_the_prds.md` topic 6, its own `[INFER]` note. |

## Open questions

1. **`SPEC-001`** — mutation → `(a_id, b_id)` addressing. Open; owner "Chunk 5 (Phase-1 fan-out)
   brainstorm"; expected outcome a new ADR (`docs/open-spec-items.md` §SPEC-001).
2. **Is the PRD's three-structure model still the target,** or does the single `relationship_state`
   projection supersede it? The PRD is deleted; nothing rules either way. (§Storage records both sides.)
3. **What does "it is logged" mean today?** The founder's ruling (2026-08-26, `product.md`) is
   present tense; `relationship_state` has no writer. Does logging arrive with SPEC-001, or does the
   ruling count relationship-flavored perception records (WE-3's) as the log? Not mine to pick.
4. **Which domain owns `gather_slice`?** Its topic (`digest/S13b` topic 34) is assigned to no cluster
   in `digest/01_TOPIC_MAP.md`. This package claims only its `relationships` section (see `seams.md`).
