# canon-spine · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-1 · The canon spine ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the write path, validation, traps.
`canon-spine.product.md` holds what it means; `canon-spine.seams.md` holds what crosses its
boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate
by grep before relying on one.

---

## Storage

- **`canon_event`** (`schema.sql:3788`; source migration `20260610090002_canon_spine.sql`) — the
  immutable log. Three CHECKs: `status` (five values), `origin` (eight values), `event_type`
  (twelve values — see the count contradiction below). **`visibility_scope` has no CHECK**; both
  doors compute it (`private` for `Communicated`, `public` otherwise) and nothing constrains a
  third value.
- **`event_participant`** (`schema.sql:3888`) — qualified event↔entity spokes,
  PK `(event_id, entity_id, role_qualifier)`. `entity_kind` is CHECKed;
  **`role_qualifier` is plain TEXT with no CHECK** — a typo'd role is silently accepted and
  silently invisible to every reader.
- **`state_mutation`** (`schema.sql:407`) — the WHAT-CHANGED ledger. `event_id` is `NOT NULL`:
  provenance is a column constraint, not a discipline (`I-2`, `ADR-008`).
- **`uq_ce_accepted_order`** (`schema.sql:4768`; migration `20260610090007`) — partial unique on
  `(world_id, in_world_tick, beat_seq) WHERE status='accepted'`, making `B-5`'s ordering a total
  order by constraint. Landed via `SPEC-002`.

**Append-only is three mechanisms, not a convention** (migration `20260610090002`):
`canon_event_append_only()` (`schema.sql:660`) rejects any UPDATE touching the 18 immutable
columns and enforces the status machine (`proposed→accepted|rejected`,
`accepted→retconned|superseded` — a rejected event is terminal); `forbid_delete()`
(`schema.sql:3306`) is attached as a `BEFORE DELETE` trigger to eight tables including
`canon_event`, `event_participant`, `state_mutation`; and DELETE is revoked by grant.

## The write path — two doors, one seeder

`apply_event(world, actor, attempt, tick, seq, origin, legacy_types)` (`schema.sql:141`, migration
`20260723100004`): a structural floor per type (blocker-only — *"contracts arrive
pre-adjudicated"*); every rejection returns exactly `{"event_id": null, "halt_reason":
"gate_reject"}`, **nothing written**. On pass: one `canon_event` row inserted directly as
`'accepted'` (`:266-278`), participant rows (`speaker`/`listener` for `Communicated`, else
`instigator`), the `SPEC-035` witness rows for `ObjectRelocated` (`:305-316` — a holder is not a
witness; malformed `witnesses` is a refusal, migration `20260825140000`), per-type effects as
`state_mutation` rows or helpers, then `PERFORM generate_perceptions(ev_id)` (`:345`) — the seam
into WE-3.

`apply_ruled_event(world, ruled, tick, seq, origin)` (`schema.sql:463`, migration
`20260724100002`): same floor, **duplicated, not shared** — the migration says outright it is
*"DUPLICATED (not extracted into a shared helper)"* with a *"keep in sync"* comment, so
twin-identity is a thing a reader maintains, never a thing the code guarantees. Canon summary is
the ruling's `truth` (*"CANON NEVER LIES"*, `:566-568`); per-receiver appearances never touch
canon. It writes its perceptions itself, without `generate_perceptions` — and it disagrees with
the first door on who perceives. **The one home for that defect is
`perception-and-knowledge.tech.md` §"The second door"**; this file only points there.

`apply_beat` delegates every step to `apply_event` with `p_legacy_types => true` (`schema.sql:108`)
— so **the live fast path still writes legacy labels** (`ActorMoved→'move'`,
`Communicated→'private_disclosure'`, `:254-262`). A query filtering on canonical labels only
misses live data.

The one non-door writer in the schema: `fn_instantiate_drowned_lantern` (`schema.sql:1889`)
inserts pre-accepted backstory/genesis rows directly (`origin='fast_path'`) — WE-10's lane.

## The event-type count is stated inconsistently — recorded, not resolved

- `core/api/prompts/resolve.txt:4` says *"EVENT TYPES (the only six)"* and
  `core/api/prompts/cognition.txt:6` says *"the six canon types"* — both list six, **omitting
  `EntityCreated`**.
- `core/api/prompts/world_actor.txt:4` says *"the same closed six as everyone else"* and lists
  **seven**, including `EntityCreated`; `core/api/ruling.go:164` (`allowedRuledEventTypes`) accepts
  the same seven.
- The DB CHECK (`schema.sql:3810`, migration `20260723100002` — filename literally
  `six_type_spine`) holds **twelve**: the seven canonical plus five legacy labels that *"stay
  legal: append-only history is never rewritten."*
- `digest/01_TOPIC_MAP.md` §WE-1 itself says "six" and lists seven.

Consequence, plainly: the resolve and cognition seats are never offered a type the schema and the
validator both accept. Whether that is a starved prompt or a deliberate withholding is a ruling —
see Open questions.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-002` | Canon events are the only dependency spine; entities carry no causal edges to each other. | A unified graph makes canon/belief separation unenforceable. |
| `ADR-008` + `ADR-029` | Provenance always; bundles selectively; bundle tables present, **no runtime writer before Phase 4** (frozen wording). | An early bundle writer reopens wording that flip-flopped across rounds and was frozen on purpose. |
| `ADR-026` | Replay invariance is domain-equivalence, not byte-identity — and **perceptions are not regenerated on replay**. | This is why an event committed without its witnesses is unwitnessable forever (migration `20260825130000`'s own argument). |
| `SPEC-002` | Canonical order is `(world_id, in_world_tick, beat_seq)` among accepted rows, unique by constraint; `recorded_at` excluded. | Seed-data shape becomes the only thing keeping replay deterministic. |
| `SPEC-033` | A committed event's payload carries only `spoken` words; `stated` is the referee's account, `content` the words said. | Only `content` can back a verbatim quote; conflating them made every speech segment unverifiable. |
| `SPEC-034` | **Landed.** State lives in `state_mutation`; the payload is `{}` on commit. | The one home for this fact is `perception-and-knowledge.tech.md` §The write path. |
| `D-5` | Additions to the frozen DDL are ADR-gated, separately-numbered migrations — the frozen files are never edited. | `20260610090007` records the procedure; editing doc-03 DDL in place breaks it. |

### What you may not decide alone

1. **Adding an event type.** The doctrine requires all five artifacts — payload schema, upcaster,
   projection handler, Traversal Matrix row, template entry (`digest/S04` Topic 5, §G6) — *plus*,
   in the code as built: the DB CHECK, both doors' floors, `ruling.go`'s set, and the seat prompts.
   `EntityCreated`'s half-adoption above shows what a partial add looks like.
2. **Adding an `origin` value.** The CHECK is closed; at least one Go test hard-codes the old set
   (see Traps).
3. **Widening the mutable column set** on `canon_event` — the 18-column enumeration is the
   product's memory guarantee (`ADR-001`).
4. **Adding a `role_qualifier` value.** Closed by convention only; every reader greps for literal
   strings.

## Validation for this domain

Single-file pgTAP (from repo root; the db container must be up):

```
docker compose exec -T db sh -c 'pg_prove -U postgres -d dreamchat --ext .sql /work/tests/102_apply_event_test.sql'
```

The named suites: `20_append_only_test.sql`, `25_delete_guard_test.sql`, `50_provenance_test.sql`
(I-2), `102_apply_event_test.sql` (18 assertions across floors and effects),
`104_apply_ruled_event_test.sql`. CI (`.github/workflows/invariants.yml`) runs the full suite plus
the two-deploy determinism fingerprint. **`make reset` destroys the dev volume and must never be
run locally** — CI-only; use `BEGIN … ROLLBACK`.

**What counts as evidence here:** the spine's characteristic failure is the **silent accept** — a
typo'd role, a malformed `witnesses` field, a legacy label — where nothing errors and a row is
simply wrong or missing. `SPEC-035`'s own comment block records the shape: *"committed, zero
witness rows, zero perceptions, no halt_reason."* Reproduce-first.

**What counts as ceremony here:** `50_provenance_test.sql`'s third assertion counts 100 seeded
noise events — it passes with both doors deleted. The I-2 join checks in the same file are the
real guards.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The live path writes legacy labels.** Filtering `event_type='ActorMoved'` misses live data. | `schema.sql:108` (`p_legacy_types => true`); mapping at `:254-262`. |
| **A Go test's origin set is stale.** `beat_authority_test.go:136` recognizes seven origins, omitting `telegraph` (in the CHECK since `20260724110004`); a telegraph-origin event in a scanned world fails its defense query. | Compare `core/api/beat_authority_test.go:136` against `schema.sql:3811`. Recorded, not resolved. |
| **The floors are twins by promise only.** | `20260724100002`: *"DUPLICATED … keep in sync"*. The `ObjectRelocated` witnesses gate already differs (apply_event only). |
| **Three types commit with no effect.** `OwnershipAccessChanged`, `EntityDestroyed`, `AttributeChanged` pass the floor, write the event and participant, and no `state_mutation`; `AttributeChanged`'s effect happens via Go-called `apply_attribute_writes`. | `digest/S13b` Topic 11; grep the per-type branches in both door bodies. |
| **`ADR-034` is Proposed while its behavior is enforced.** The ordering rule is gated by `uq_ce_accepted_order` and cited as settled law by `20260809090001_carrying.sql:36`; the ADR's status line (`docs/law/02_world_state_adrs.md:242`) still reads Proposed. | Both sides named; a status change is a ruling. |
| **The second door disagrees with the first on who perceives.** | `perception-and-knowledge.tech.md` §"The second door" — the one home. |

## Open questions

1. **The event-type count.** Six or seven? Are `resolve.txt`/`cognition.txt` starving their seats
   of `EntityCreated` deliberately or by drift? (§the count contradiction above.)
2. **`ADR-034`'s status** — Proposed in the log, enforced by constraint. Accept it or amend it?
3. **When do legacy labels retire?** `p_legacy_types` exists so `apply_beat` preserves history's
   labels, but every new consumer must now know two names per type. Who owns the migration?
4. **The shape of the `apply_ruled_event` perception fix** depends on `SPEC-038` — WE-3's open
   question; the spine inherits the floor-and-fanout consequences of whatever is ruled.
