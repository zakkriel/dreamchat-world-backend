# perception-and-knowledge · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-3 · The epistemic layer ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the write and read paths, validation, traps.
`perception-and-knowledge.product.md` holds what it means; `perception-and-knowledge.seams.md` holds
what crosses its boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate by
grep before relying on one.

---

## Storage

- **`perception_record`** — one sourced piece of a holder's knowledge. The columns that carry meaning:
  `holder_id` (whose knowledge), `source_event_id` (provenance), `epistemic_type` (closed CHECK set on
  the table), `confidence` / `distortion_level`, and the validity window `acquired_tick` / `valid_tick`
  / **`invalid_tick`** — perceptions are **invalidated, never deleted** (`ADR-006`). Full DDL:
  `grep -n 'CREATE TABLE public.perception_record' core/db/schema.sql`.
- **`perception_subject`** — `perception_id · entity_id · world_id`, the join that makes visibility
  work. **This is the table SPEC-034 was about.**
- `name_knowledge` is **not** this domain's table; it is WE-4's (see `seams.md`).

## The write path

`apply_event` commits, then `PERFORM generate_perceptions(ev_id)` (`core/db/schema.sql:345`;
definition at `:3415`). One arm per event type; each arm inserts `perception_record` rows plus their
`perception_subject` rows, and decides three things: *who perceives*, *what subjects the perception
names*, and *what epistemic type it carries*.

**An event type with no arm perceives nothing, silently.** Do not count the arms from this file — list
them: `grep -n 'ev\.event_type' core/db/schema.sql` (the dispatch lines inside
`generate_perceptions`).

`generate_perceptions` reads **`state_mutation`** for WHAT CHANGED — never the payload for state,
which is `{}` on commit; the payload carries only `spoken` words (SPEC-033), read by the
Communicated arm of `generate_perceptions` (`schema.sql:3435`) AND by `fn_unearned_names` (the
wall's vocabulary source, `schema.sql:3013`). Getting the state half wrong produces a fix that
applies cleanly and does nothing (SPEC-034's receipt, recorded in the migration's own comments:
`core/db/migrations/20260825120000_object_relocated_perceptions.sql`). This paragraph is the one
home for the payload-on-commit fact; the trap row and the seams row point here.

## The second door, and it disagrees

`apply_ruled_event` writes perceptions itself, without `generate_perceptions`:

- receivers are `fn_actors_at(p_world_id, here) UNION actor` (`schema.sql:619-623`);
- subjects are written from `participant_ids` (`:645-648`), which is `ARRAY[actor_id, listener]` on
  the Communicated branch (`:578`) and `ARRAY[actor_id]` otherwise (`:582`).

Consequence, plainly: **on a ruled handover the object is never a subject**, so `fn_entity_visible`
is false for everyone including the new holder — and perception is granted by presence alone, which
contradicts the named-witness ruling behind SPEC-035. Reproduced 2026-08-27. It has no traffic yet:
every event in the dev database came through `fast_path`.

**This round does not fix it.** The founder ruled `apply_ruled_event` should get the same treatment
as `apply_event`, and the *shape* of that fix depends on SPEC-038 (see `docs/open-spec-items.md`
§SPEC-038 "Related"). Listed in Open questions below.

## The read path

- **`fn_entity_visible(world, viewer, entity)`** (`schema.sql:1762`) — asks whether the viewer holds
  a perception whose **subject** is that entity, via `fn_visible_perceptions(world, viewer)`
  (`:3133`) for the visible set. This is what visibility *is* — which is why a perception naming the
  right holder and the wrong subjects is invisible.
- Every page, index and timeline projection joins through `fn_visible_perceptions` +
  `perception_subject`. List the consumers rather than counting them:
  `grep -n 'fn_visible_perceptions\|fn_entity_visible' core/db/schema.sql`.

## Where the code lives

Verified against `git ls-files`, 2026-08-27:

- `core/db/migrations/*perception*` and `core/db/migrations/*object_relocated*`
- `core/db/schema.sql` (the generated single source for current definitions)
- pgTAP under `core/db/tests/` — the suites named in Validation below

There is **no `core/api/perception*.go`** — `git ls-files 'core/api/perception*'` returns nothing.
The Go files a search will surface (`namingwall.go`, page/compendium/timeline handlers) belong to
WE-4 and UX-1 respectively (see `harness2/DOMAINS.map`).

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-006` | Three time axes; invalidation, never deletion. | A deleted perception erases the history the product sells. |
| `ADR-P025` | Perception is a pipeline of typed blockers: an actor perceives unless something stopped them; links block, never grant; blocks are `physical` (Physics) or `attentional` (resolve seat). | Changing who perceives anything without reading it re-decides the domain's shape. |
| `I-3` | No hidden-canon leakage. CI-enforced. | The build fails. |
| `SPEC-034` | **Landed.** A handover makes the object perceptible to the holders the event names — and the object must be a subject. | Reintroducing a subject gap recreates the 404-on-carried-artifact defect. |
| `SPEC-035` | **Landed, holders-plus-named-witnesses only.** The event names its witnesses; co-presence is necessary, not sufficient; malformed input is refused, never dropped. | Inferring witnesses from co-presence re-litigates a founder ruling. |
| `D-2`, `workspace:ADR-W005` | Perception is permanently core, not a module. Recurring request, settled answer (`docs/00_workspace/closed-questions.md`, "area or module" row). | Calling it a module is a `D-2` violation wearing a modularization costume. |

### What you may not decide alone

1. **Adding an epistemic type.** The set is closed (CHECK on `perception_record`). A new one changes
   what the world can mean.
2. **Widening who perceives an event.** SPEC-035 set the precedent that this needs a ruling — *"just
   because they were there doesn't mean they saw it."*
3. **Anything that lets an unearned name into a payload.** Not a bug to fix later — a breach (WE-4's
   wall, `seams.md`).
4. **Making perception a module.** Settled; see the decisions table.

## Validation for this domain

The named suites — pgTAP in `core/db/tests/`: `40_perception*`, `106_perception_subject*`,
`1[24]_perception_subject*`, `17_entity_visible*`, `42_visible*`, `9[34]_generate_perceptions*`,
`97_perception_determinism*`, `122_object_relocated_perceptions*`,
`123_object_relocated_witnesses*` — plus the Go suite in `core/api`.

Two facts that matter more than the list:

- **`make reset` destroys the dev volume holding twelve worlds and must never be run** — use
  `BEGIN … ROLLBACK` or a copied tree.
- The Go suite in `core/api` is **seed-dependent and not idempotent**, so `-count=1` always — the
  cache will show a stale pass.

**What counts as evidence here:** this domain fails silently — nothing errors, someone just cannot
see something — so **reproduce-first is a rule, not a preference.** SPEC-034's first draft read
`ev.payload`, applied cleanly, and produced zero perceptions; only the reproduction caught it.

**What counts as ceremony here:** asserting on seeded rows. The seeded carry edges were authored as
**state**, so no perception rule can reach them — a suite asserting on them passes with the arm
deleted. That is the definition of a vacuous test, and it is the easiest mistake to make in this
domain.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **`canon_event.payload` carries no state.** | §The write path above — the one home for this fact (SPEC-034's receipt lives there). |
| **Naming the actors is not enough — name the object.** `fn_entity_visible` reads subjects. | SPEC-034; the fix's own comment block in `core/db/schema.sql` (grep `THE FIX. fn_entity_visible`). |
| **`fn_actors_at` reads `actor_state`.** An actor with no state row is nowhere, and every co-presence gate refuses them. | `schema.sql` `fn_actors_at` definition (grep it); a fixture without `actor_state` reads as "the gate is broken" when it is "your fixture has no world." |
| **A witness is not a holder.** The exclusion is load-bearing: without it a party gets two perceptions of one event. | SPEC-035, mutation-tested (`docs/open-spec-items.md` §SPEC-035). |
| **An invariant maintained by the harness is not an invariant.** The Go suite backfilled missing subject rows before pgTAP looked. | `docs/00_workspace/failure-log.md` row 16. If you add a repair helper, you may be deleting a guard. |
| **The Go suite poisons pgTAP.** `pressure_test.go` asserts against an empty `world_eruption` and drains it for tests that follow. | Cost two false regression reports in one day (draft dossier receipt, re-verified: `core/api/pressure_test.go` asserts the empty-table precondition). |
| **A 100%-caught mutation table means nothing about malformed input.** | `docs/00_workspace/failure-log.md` row 45: the silent-drop defect recurred inside its own mutation-tested fix. |
| **The ruled path disagrees with the attempt path on who perceives.** | This file, "The second door"; `docs/open-spec-items.md` §SPEC-038 "Related". |
| **Counts go stale in days.** State no assertion, consumer or arm counts; point at the file or the grep. | `123_object_relocated_witnesses_test.sql` changed its plan while `docs/areas/perception.md` stated the old "(10 assertions)" — the stale count was deleted 2026-08-27 (`digest/03_TIMESERIES.md` row S10a\|9). |

## Open questions

1. **The shape of the `apply_ruled_event` fix** — see "The second door" above (the one home for
   this fact); the shape depends on SPEC-038.
2. **`participant_ids` drift between contract and code.** `core/api/schema/ruling.v2.schema.json:73`
   still carries `participant_ids` (marked "v1 compat — removed when the v1 decode path retires");
   the Go `RuledEventV2` struct (`core/api/ruling.go:83`) does not decode it —
   `apply_ruled_event`'s `participant_ids` is a plpgsql local derived from actor/listener, not the
   payload field. Unexamined beyond this observation.
3. **`SPEC-016`** — per-attribute perceivability. Next, or deliberately parked?
4. Does **"how my understanding changed"** stay a product feature? It is why perceptions are
   versioned rather than updated, and it is currently unrendered.
