# projections-and-replay · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-2 · Projections and the read model ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the write and read paths, validation, traps.
`projections-and-replay.product.md` holds what it means; `projections-and-replay.seams.md` holds
what crosses its boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate
by grep before relying on one.

---

## Storage

Four tables, one migration (`core/db/migrations/20260610090005_projections.sql`):

- **`actor_state`** — `entity_id` **PRIMARY KEY alone** (not `(world_id, entity_id)`), `world_id`,
  `attrs JSONB DEFAULT '{}'`, `dirty`, `last_event_id` (provenance of the last applied delta),
  `updated_at` (volatile — excluded from every diff).
- **`location_state`**, **`artifact_state`** — created as `(LIKE actor_state INCLUDING ALL)`:
  identical by construction *once*; a column added to one is not added to the others.
- **`relationship_state`** — `PRIMARY KEY (world_id, a_id, b_id)`, **no `updated_at`**
  (pgTAP-pinned: nothing volatile to exclude), and **empty by design** (`SPEC-001`,
  `product.md` §deliberately not built).

**The grant wall (`I-7`) lives only in the migrations, not in `schema.sql`** — the dump carries no
GRANT/REVOKE lines (verified: grep for either returns nothing). The wall is
`20260610090005:26-28` (`REVOKE ALL … FROM PUBLIC; GRANT ALL … TO maintainer; GRANT SELECT … TO
app_reader`) plus function hardening at `20260610090006:107-111` — *"SECURITY DEFINER functions are
doors through the grant wall."* Both roles are `NOLOGIN` (`20260610090001`).

## The write path

`apply_mutation(m state_mutation)` is **the only projection write path** — the migration header
says so: *"apply_mutation() is the ONLY projection write path; live trigger AND replay both call
it"* (`20260610090006:3`; definition in `schema.sql:427`). It strips `attrs.` from
`attribute_path`, splits the rest into a JSON path, and `jsonb_set`s the **absolute** new value
(ABSOLUTE-STATE-SETS convention). Idempotency rests entirely on values being absolute — re-applying
an absolute set is a no-op. **The moment delta semantics appear, this breaks** (`SPEC-004`).

Live maintenance is one trigger: `trg_sm_project AFTER INSERT ON state_mutation`
(`schema.sql:4873`) → `sm_project()` (`:3730`), which applies the mutation **only if the parent
event is already `accepted`**. There is no trigger on `canon_event` status change, so a mutation
inserted under a `proposed` event that is later accepted never projects live — unreachable today
because every commit path inserts events as `accepted` first, and owed as `SPEC-003`.

Who inserts `state_mutation` rows: the two doors (`apply_event` `schema.sql:320`,
`apply_ruled_event` `:587` — canon spine's functions), their helpers `fn_apply_carry_change`
(`:874`) and `fn_apply_entity_created` (`:963`), `apply_attribute_writes` (`:39`), and the world
template `fn_instantiate_drowned_lantern` (`:1889`) — seeds also go through the mutation log,
never into `*_state` directly, which is what makes seeded worlds replay-safe.

## Replay — what `I-1` actually asserts

`replay_0A()` (`schema.sql:3644`; migration `20260610090006:50`): snapshot the four tables into
`ON COMMIT DROP` temp tables → `TRUNCATE` all four → re-apply every accepted event through the
**same** `apply_mutation` → count domain differences both ways → `RETURN diff_count = 0`.
Re-entrant on purpose (`DROP TABLE IF EXISTS` at the top; the negative-control test calls it 3×).

Two details that are the design:

- **Domain-only order.** Events: `ORDER BY world_id, in_world_tick, beat_seq`; mutations within an
  event: `ORDER BY valid_from_tick, valid_from_seq`. `recorded_at` is transaction time and
  excluded (`ADR-034`). Ties among accepted events are forbidden by the partial unique index
  `uq_ce_accepted_order` (`schema.sql:4768`, `SPEC-002` resolved).
- **Domain-only diff.** Per table, `EXCEPT` both ways over `(pk…, world_id, attrs, last_event_id,
  dirty)` — `updated_at` excluded by name (`ADR-026`: domain equivalence, not byte identity).

**What replay does NOT reproduce: perceptions.** They are not projections and are never
regenerated on replay (`ADR-026` as applied by migration `20260825130000`, whose header states the
consequence: an event committed without its witnesses is unwitnessable forever). Anything that
treats replay as "rebuild the world" instead of "rebuild the four `*_state` tables" is wrong.

## The read path

Projection readers are **engine-internal**: the doors' premise checks (containment, capacity,
`fn_actors_at` via `actor_state.attrs.location_id`), the play loop (`gather_slice`, `apply_beat`),
journey/space arithmetic, cognition's tension read, and art generation (`imagehandler.go` — reads
authoritative state by documented decision; see WE-3's seams file). Compendium pages read **none
of this** (`seams.md`). List consumers rather than counting them:
`grep -n 'FROM actor_state\|FROM location_state\|FROM artifact_state\|FROM relationship_state' core/db/schema.sql`.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-002` | Layered architecture; state projections are layer 3, never traversed for propagation. Projection-maintenance machinery is day-one infrastructure. | Cascading through a read model is the graph-soup failure the layering exists to prevent. |
| `ADR-004` | Plain triggers first; builders idempotent; projections ephemeral. | See product.md table — one home. |
| `ADR-026` | Replay compares domain equivalence; volatile columns excluded by a defined set. | Byte comparison always fails; adding a volatile column without extending the exclusion set breaks `I-1` falsely. |
| `ADR-034` | Domain order is `(in_world_tick, beat_seq)`, unique per world for accepted events; transaction time never orders the domain. | A wall-clock tiebreaker makes two rebuilds disagree — silent `I-1` breakage. |
| `SPEC-002` | **Landed.** The uniqueness half is schema-enforced (`uq_ce_accepted_order`), not seed-shaped. | Relying on data shape instead of the index re-opens the foot-gun. |
| `ADR-P017` | Correctness walls live in SQL, pgTAP-tested; Go is a thin reader. | A Go-side re-implementation of projection logic is a second copy of the law. |

### What you may not decide alone

1. **Adding a column to any `*_state` table.** The `LIKE` construction means the three entity
   tables only *look* coupled; and every new column must be classified volatile-or-domain for the
   replay diff and the fingerprint (`Makefile:39`). Getting that wrong breaks `I-1` silently.
2. **Delta semantics on mutations** (`attrs.gold + 5`). Breaks absolute-set idempotency;
   `SPEC-004` says what replaces it, and it is a ruling, not a refactor.
3. **Relationship projection addressing.** `SPEC-001`'s ruling, informed by Phase-1 evidence.
4. **Any new `SECURITY DEFINER` function that writes projections.** Each is a door through the
   grant wall; the set of doors is the invariant.

## Validation for this domain

- `make test` — pgTAP; the suites that are this domain: `30_apply_mutation_test.sql` (live
  trigger, idempotency, SPEC-001 no-op, non-accepted parent), `60_permissions_test.sql` (`I-7`:
  app_reader denied INSERT and EXECUTE), `70_determinism_guards_test.sql` (order-key uniqueness by
  constraint), `80_golden_projection_test.sql` (hand-derived final locations),
  `90_replay_test.sql` (`I-1` happy path + corruption detected + rebuild repairs).
- `make replay` — `I-1` by hand, wrapped in `BEGIN…ROLLBACK`.
- `make fingerprint` — domain-only projection dump; CI (`.github/workflows/invariants.yml`) diffs
  it across two fresh deploys (determinism), then `make schema-check` for dump drift.
- **`make reset` destroys the dev volume holding twelve worlds and must never be run locally** —
  CI runs it in a container; you use `BEGIN…ROLLBACK` or a copied tree.

**What counts as evidence here:** replay's diff, or a pinned pgTAP assertion — this domain fails
loudly *only when replay is actually run*, and a corrupt projection serves reads happily until
then. Reproduce with `make replay` before and after.

**What counts as ceremony here:** a test that seeds `*_state` directly and then asserts on it —
it passes with the entire write path deleted, and it plants exactly the corruption replay exists
to catch (see Traps). The golden test earns its keep only because its expectations are
hand-derived from the seed loop, not read back from the table.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The Go suite writes projections directly.** `tension_test.go:100-106` INSERTs into `artifact_state`, bypassing `apply_mutation` — the exact operation `I-7`/`I-2` forbid, and it is still in the live suite (re-verified 2026-08-27). The test pool runs as superuser, so the grant wall does not bite in tests. | `docs/00_workspace/failure-log.md` row 24; `core/api/tension_test.go:100` (grep `INSERT INTO artifact_state`). |
| **An invariant maintained by the harness is not an invariant.** The Go suite once backfilled rows before pgTAP looked; a repair helper can be a deleted guard. | `docs/00_workspace/failure-log.md` row 16. |
| **The trigger covers inserts, not acceptance transitions.** Propose-then-accept projects nothing live. | `SPEC-003`; pinned by `30_apply_mutation_test.sql` case (4). |
| **Replay does not rebuild perceptions.** "Drop and replay" repairs `*_state` and only `*_state`. | §Replay above — the one home for this fact. |
| **The dump is not the law on grants.** `schema.sql` shows no wall at all; auditing `I-7` from the dump concludes it does not exist. | §Storage above (grep receipt there). |
| **`relationship_state` emptiness is intentional.** Filling it "helpfully" without SPEC-001 breaks the pinned zero-row guards. | `70_determinism_guards_test.sql`; `30_apply_mutation_test.sql` case (3). |

## Open questions

1. **The Traversal Matrix has no home and no teeth.** `ADR-002`/`ADR-017` say it is *"enforced in
   code"*, and the corpus says a relation type without a matrix row *"fails CI"* — but the doc
   that held the matrix table (`canon_engine/03`) was **deleted** in commit `88486c1` (docs
   consolidation, 2026-08-27), no CI workflow or `ci/` script references it, and the R2+
   propagation machinery it governs has no writers. Both sides recorded; whether the matrix is
   restated somewhere citable or the ADR wording is superseded is a ruling.
2. **`SPEC-003`** — the acceptance-transition trigger: owed to the first chunk that introduces the
   proposed lifecycle. Until then the gap is unreachable but real.
3. **`SPEC-004`** — mutation-id-keyed idempotency and a `status` guard (today `apply_mutation`
   ignores `state_mutation.status`, so a future `reversed` row would still apply).
4. **`SPEC-001`** — relationship addressing (see product.md §deliberately not built).
