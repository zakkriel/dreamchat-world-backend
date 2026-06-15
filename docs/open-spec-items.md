# Open Spec Items

ADR numbers are assigned at proposal time in `canon_engine/02` (D-5). Do not pre-assign.

## SPEC-001 — mutation → relationship_state addressing
doc 03 does not specify how a single-`entity_id` `state_mutation` addresses a
`(world_id, a_id, b_id)` `relationship_state` row. Phase 0A ships a **no-op stub** branch in
`apply_mutation()` and keeps `relationship_state` empty (pgTAP-asserted zero rows).
- **Owner:** Chunk 5 (Phase-1 fan-out) brainstorm.
- **Expected outcome:** a proposed new ADR in `canon_engine/02`, informed by Phase-1 evidence (D-9).

## SPEC-002 — recorded_at in the canonical replay ordering key
doc 13 §6 and doc 03 §3.4 write the replay order as `(in_world_tick, beat_seq, recorded_at)`.
`recorded_at` is volatile wall-clock (B-5 transaction-time; ADR-026 volatile) and is a latent
determinism foot-gun as an ordering tiebreaker. Phase 0A removes it and guarantees a domain-only
total order via per-world `(in_world_tick, beat_seq)` uniqueness.
- **Owner:** Chunk 1 retro.
- **Expected outcome:** a proposed new ADR in `canon_engine/02` (number assigned at proposal time).
- **Proposal text:** "Canonical event ordering is `(in_world_tick, beat_seq)`, required UNIQUE per
  world; `recorded_at` is transaction-time (B-5) and excluded from domain ordering (ADR-026)."
- **Status:** Resolved → ADR-034 (proposed). The "required UNIQUE" half is now enforced by schema,
  not by seed-data shape: migration `20260610090007` adds partial unique index `uq_ce_accepted_order`
  on `(world_id, in_world_tick, beat_seq) WHERE status='accepted'` (kept out of the verbatim
  doc 03 migrations 0002–0006), with positive/negative pgTAP guards in
  `70_determinism_guards_test.sql`. The owed ADR proposal is now filed as ADR-034 in
  `canon_engine/02_world_state_adrs.md` (supersedes the doc 13 §6 / doc 03 §3.4 ordering text).

## SPEC-003 — projection on the proposed→accepted transition (doc 03 §3.1, second half)
doc 03 §3 rule 1: projection triggers fire "on insert with `status='accepted'` **or transition to
it**". Phase 0A implements only the insert-under-already-accepted half (`sm_project()` on
`state_mutation` insert); there is no trigger that projects an event's mutations when a
`canon_event` flips `proposed→accepted`. Correct and untestable in 0A (doc 13 §3: no proposed
lifecycle), but it is the half Phase 1's validation gate hits first.
`30_apply_mutation_test.sql` case (4) pins the current behaviour (non-accepted parent does not
project).
- **Owner:** the first Phase-1 chunk that introduces the proposed lifecycle (validation gate).
- **Expected outcome:** an `AFTER UPDATE` acceptance-transition trigger on `canon_event` calling
  the same `apply_mutation()`, with pgTAP coverage. No ADR needed — already specified in doc 03;
  this item only tracks the unimplemented half.

## SPEC-004 — idempotency mechanism: absolute-set semantics vs. the §3.2 mutation ledger
doc 03 §3 rule 2 prescribes idempotent projection writes via an applied-mutations ledger or
deterministic upserts keyed by `mutation_id`. Phase 0A instead relies on Rider B absolute-set
semantics (re-applying the same absolute value is a no-op), which is sound **only while deltas
are forbidden**. The moment any delta semantics appear (e.g. `attrs.inventory.gold` arithmetic),
absolute-set idempotency breaks and the §3.2 ledger becomes mandatory. Related:
`apply_mutation()` currently ignores `state_mutation.status`, so a future `reversed` row would
still apply — same revisit.
- **Owner:** the first Phase-1 chunk that introduces non-absolute mutations.
- **Expected outcome:** mutation-id-keyed idempotency per doc 03 §3.2 + a `status` guard in
  `apply_mutation()`. No ADR needed — already specified in doc 03.

## SPEC-005 — nightly full acyclicity check (deferred half of I-4)
Phase 0B implements the **insert-time** half of I-4 (doc 03 §1.4: bounded ancestor walk on
`causal_bundle_input` insert; migration `20260611090001`). doc 07 I-4 also specifies a **nightly
full check** (recursive CTE with depth cap; cap hit = investigation). Not built in 0B — no
scheduled/operational jobs exist yet, and 0B's gate is satisfied by insert-time rejection.
- **Owner:** the first chunk that introduces scheduled/operational jobs (per-world nightly sweeps).
- **Expected outcome:** a nightly per-world recursive-CTE acyclicity sweep + alert on a positive
  hit or a depth-cap hit. **No ADR needed** — already specified in doc 07 I-4.
- **Note (depth cap):** the insert-time walk's depth cap of 64 is a Phase-0B fail-safe ceiling
  (it raises a distinct "investigate" error on cap-hit), **not** a domain limit on causal-chain
  length. The future full-graph check must raise or remove this cap deliberately rather than
  inherit 64 as if it were a modeled bound.

## SPEC-006 — perception holder cardinality and invalidation tracking
`perception_record` currently has a scalar `holder_id`. When multiple actors acquire identical
knowledge identically (same `epistemic_type`, same `source_event`, same `acquired_tick`), should
they share one perception row via a junction table `perception_holder(perception_id, actor_id,
acquired_at, invalid_at)`, or write N identical rows?
- **Junction table design:**
  - One `perception_record` per unique knowledge (fact + how-learned + source-event).
  - `perception_holder` tracks the (perception, actor) relationship with invalidation (ADR-006:
    `invalid_at` stamp, never DELETE, so "what did X used to know" remains queryable).
  - Satisfies B-1 (perception-bound), I-2 (perception traces source), B-6 (contradiction resolved
    by latest accepted event — applies to invalidation too).
  - Read-side assembles: `SELECT perception WHERE holder IN (...) AND invalid_at IS NULL` (current
    knowledge) or `invalid_at IS NOT NULL` (historical).
  - Satisfies the "world feels alive" constraint: knowledge relationships have lifecycle (acquired,
    possibly invalidated), visible in audit trails and character memory.
- **Constraints to verify:**
  - Does the junction table play well with B-2 (valid perception paths)?
  - Does invalidation order matter if multiple holders lose a perception at different ticks?
  - Does the read-assembly logic (current + historical) break any epistemic rules (B-3 through B-10)?
- **Owner:** Chunk 3 brainstorm (projection API assembly, when the read-side semantics become
  concrete).
- **Expected outcome:** a proposed ADR formalizing the junction table shape, invalidation semantics,
  and read-assembly rules — informed by evidence from the first perception-bound page (Seren's Actor
  page). Contract amendment waiting for evidence; no implementation yet.

### Resolution (2026-06-12, pre-Chunk-3 fork) — **Path 1: scalar `holder_id` stays; one perception record per holder.**
**Status:** Resolved. No schema change; no new ADR. The frozen DDL (doc 03 §1.3) is already correct.

The junction-table framing above is **withdrawn.** Holder cardinality remains the scalar
`perception_record.holder_id`, and identical knowledge acquired by N actors writes **N rows**, one
per holder.

- **Why the junction collapses back to per-holder rows.** The per-holder epistemic attributes —
  `epistemic_type`, teller/source, `confidence`, `distortion_level`, and the **Decay** mechanic
  (glossary: a *mechanic*, confidence lowered by uncorroborated age, "last known…" language) — are
  **independent per holder.** Two actors who "acquire identical knowledge identically" almost never
  stay identical: they decay on different clocks, get corrected by different later events, and carry
  different confidence. A junction (`perception_holder`) or array of holders would force those
  independent attributes either onto the shared record (wrong — they aren't shared) or back onto a
  per-(perception, holder) row (which **is** a per-holder record under a different name). The shapes
  collapse to Path 1; the junction buys nothing but write-time complexity and a B-7 foot-gun
  (copy-the-record-as-shared is exactly the "everyone becomes a witness" shortcut B-7 forbids).
- **Common knowledge needs no fan-out.** World-ambient facts (glossary: *Common Knowledge*, a valid
  knowledge path) are modeled with a **group-entity holder** (a `faction`/`group` entity, e.g. a
  public/`PUB` audience holder) rather than fanning one row out to every member. One row, group
  holder, read-side expands membership at assembly time. So the "N identical rows" cost does not
  apply to the public/ambient case that would have been the only real motivation for sharing.
- **External corroboration.** Per-figure knowledge stores in the wild keep knowledge **per knower**,
  not shared-with-links (Dwarf Fortress tracks what each historical figure knows individually); the
  belief-base / doxastic-logic literature likewise models each agent's belief set separately. The
  independent-attributes argument is the standard one.
- **Invalidation tracking** (the second half of the original item) is already handled by ADR-006 on
  the per-holder row: `invalid_tick` (in-world falsification) and `expired_at` (system supersession)
  close a holder's perception without deletion, so "what did X used to know" stays queryable
  per holder. No junction needed for lifecycle either.
- **Re-open trigger (narrow, recorded):** revisit **only** if a *shared-object-with-no-per-holder-
  attributes* case actually appears — a body of knowledge genuinely shared by many holders that
  carries **zero** per-holder epistemic attributes (no per-holder type/teller/confidence/decay). If
  that case is ever evidenced, the answer is a **junction table over an array column** (normalized
  membership, not an inline `holder_id[]`). Absent that evidence, Path 1 stands.

## SPEC-007 — CI invariant workflow never executed (two stacked causes)
**Status:** Resolved — CI had **two independent reasons** it never ran, stacked so the first
masked the second; both fixed from PR #4 onward.
1. **$0 Actions stop-budget.** GitHub Free with a $0 spending limit and stop-usage-on ($0 owed,
   168/2000 free minutes — **not** a real billing problem) made every `invariants.yml` run 0s-fail
   at startup with **zero jobs**, *before* the workflow file was ever parsed. Fixed by adding a
   payment method / non-zero Actions budget.
2. **Invalid workflow YAML (line 14).** Once Actions could start, GitHub surfaced
   `Invalid workflow file …#L14 — YAML syntax error`: the step name
   `Run invariant suite (pgTAP: I-1/I-2/I-7 + guards + golden)` was an unquoted plain scalar
   containing a colon-space (`pgTAP: I-1`), which YAML reads as a nested mapping value ("mapping
   values are not allowed here", col 41). Pre-existing since chunk-1, masked by cause 1. Fixed by
   quoting the value (validated with pyyaml + ruby/psych).
- **Consequence for the gate map:** `chunk-1-0A-gate` was cut on **local evidence only** (it
  predated any CI execution). **chunk-2 gates on CI green + local** — PR #4 is the first genuinely-
  green CI run in this repo, after both causes were cleared.
- **Owner:** chunk-2 (this chunk). No engine/code change — an operational/account fix plus a
  one-line CI workflow-file fix.

## SPEC-008 — perception subject / about-ness (the told-about-a-third-party gap)
`perception_record` (doc 03 §1.3) has **no subject column**. The thing a perception is *about* is
currently derivable **only** indirectly: `perception_record.source_event_id → event_participant`
(the participants of the source event). That derivation is correct for direct/witnessed knowledge —
the perception is about the entities that were *in* the event — but it has a structural failure mode.

- **The failure mode (told-about-a-third-party).** When A tells B about C, the *telling* event's
  participants are A (speaker/source) and B (recipient/listener). **C is not a participant of the
  telling event.** So B's resulting `told`-type perception is *about* C, but the only available
  derivation (`source_event → event_participant`) yields {A, B}, never C. About-ness is lost exactly
  in the rumor/disclosure/gossip path that is the product's whole point (B-1, B-7). doc 06 already
  *relies* on about-ness ("perceptions about entities present in the scene, capped per entity") with
  no column to read it from — it works today only because Phase 0A has no third-party tellings.
- **External corroboration.** This is a known modeling hazard, not a novel one: W3C PROV deliberately
  separates the *entity an activity is about* from the activity's agents/used-entities
  (PROV ISSUE-49, "aboutness"); RDF needs **reification** (or RDF-star) to state a statement *about*
  a statement/third party; ActivityStreams carries an explicit `object` distinct from `actor` and
  `target` for precisely this reason. The lesson across all three: the subject of an assertion is a
  first-class edge, not something you back out of the assertion's participants.
- **Resolution direction (decision recorded; migration NOT in this entry).** Add an explicit
  junction `perception_subject(perception_id, entity_id)`, **populated at write time** by whatever
  creates the perception (the telling/extraction path knows C). The derived
  `source_event → event_participant` path is **retained as a fallback/validation** signal, not
  discarded — for direct/witnessed perceptions the junction and the derivation should agree, and a
  disagreement is a useful audit flag (cf. ADR-033's two-signal philosophy).
  - **One perception, many subjects.** A single perception may be *about* several entities; these are
    **links, not record-splits**, because the epistemic attributes (`epistemic_type`, `confidence`,
    teller, decay) are **shared across the subjects of that one belief**.
  - **But differing per-subject confidence ⇒ separate records.** If B is sure about C but unsure
    about D within the "same" telling, that is **two legitimately separate `perception_record`
    rows** (different `confidence`), each with its own subject link — not one record with
    per-subject confidence. This keeps the per-holder, per-belief attribute model (and B-7) intact
    and mirrors the SPEC-006 reasoning: independent epistemic attributes never share a row.
- **Frozen-DDL check (D-5 / engine governance).** `perception_record` **is** defined in the engine
  Master DDL (doc 03 §1.3, the frozen build contract). Adding `perception_subject` is therefore a
  **schema addition to the frozen set**, which per the engine's own governance is *not* a code
  workaround — it requires a **proposed engine ADR** (next free number in `canon_engine/02`). That
  ADR is drafted as **ADR-035 (Proposed)** in `canon_engine/02_world_state_adrs.md`. (Had the table
  lived outside the frozen DDL, a platform `ADR-P###` would have sufficed; it does not.)
- **Owner:** **Chunk 3 plan owns the implementation** (the migration, the write-time population, the
  pgTAP coverage, and the derived-vs-junction validation check). This entry records the *decision and
  the owed ADR*, not the migration. No table is added by this ledger entry.
- **Expected outcome:** ADR-035 moves Proposed → Accepted under the Chunk 3 gate; the
  `perception_subject` migration ships in Chunk 3 with positive/negative pgTAP and a
  derivation-agreement guard.
  DONE (chunk-3, 2026-06-14): perception_subject shipped (migration 20260614090001), write-time
  population in the seed, pgTAP positive/negative + derivation usage; ADR-035 Accepted.

## SPEC-009 — recorded scale triggers (capacity tripwires from the architecture pressure-test)
Not an open question — a **standing tripwire ledger** captured from the external pressure-test so
scale work is evidence-triggered (ADR-025: operational numbers are provisional), never speculative or
premature. Each item names the **measured condition** that should reopen it; until a condition fires,
building the mitigation is out of scope. None of these touch the frozen engine canon, an invariant,
or the Master DDL.

- **Event-replay snapshots** (`world_snapshot` exists, doc 03 §1.6; the cadence does not): build the
  snapshot/restore path when a world's replay/projection rebuild approaches **~10k events per
  stream** or **tens of seconds** to rebuild. (Supersedes ADR-025's placeholder "100-event
  snapshot cadence" — that number is a guess; this is the tripwire.)
- **Materialized Actor-page projections:** introduce a materialized read model for the Actor page
  **only when the live perception-join p95 misses the latency budget.** Hard constraint regardless
  of materialization: **the perception/safety filter (B-1, I-3) must run on authoritative rows, not
  on the materialized projection** — a stale materialized page could otherwise leak now-hidden truth
  (I-3). Materialize the *display* join, never the *epistemic* wall.
- **Citus / row-based sharding:** adopt when **single-node capacity binds.** Shard **by row on
  `world_id`** (core tables carry world_id; EXCEPTION: the junction/edge tables event_participant,
  provenance_edge, causal_bundle_input do NOT — see the deferral below).
  **Explicitly NOT** schema-per-world and **NOT** partition-per-world — those fragment the tenant key
  and break cross-world operational queries for no capacity win at this stage.
- **Table partitioning:** consider per-table partitioning at **~10M rows per table** (the append-only
  `canon_event` / `state_mutation` / `perception_record` tables hit this first).
- **Apache AGE (in-PG graph):** add the graph projection (consistent with ADR-003's "graph DB is a
  secondary, async-fed projection, never the source of truth") **only if intra-world traversals
  routinely exceed ~5 hops.** The architecture deliberately keeps the hot path shallow (ADR-017
  radii), so this should stay dormant.
- **JSONB hygiene (standing constraint, not a trigger):** keep hot-path JSONB values **<~2KB**;
  promote **hot filter keys to generated columns** (don't filter inside the blob on the hot path);
  use **LZ4 TOAST compression** for the larger payloads. This applies continuously, not at a
  threshold.
- **Rejected alternatives (do not re-litigate without new evidence):**
  - **Access-list holder model** for perceptions — rejected; per-holder records win (SPEC-006).
  - **Dedicated external graph DB as source of truth** — rejected; Postgres-first, graph is an
    async secondary projection only (ADR-003).
  - **External event-store product** — rejected; the append-only `canon_event` spine in Postgres is
    the event store (ADR-001).
  - **Day-one materialized read models** — rejected; plain trigger-maintained projections first,
    materialize only on measured need (ADR-004, and the Actor-page tripwire above).
- **Owner:** the first chunk that hits any named condition. **Expected outcome:** per fired trigger,
  a scoped implementation chunk; conditions that change a decision get a register amendment or new
  ADR. No action while every tripwire is unfired.

### SPEC-009b — world_id absent on three junction/edge tables (deferred, chunk-3 audit)
event_participant, provenance_edge, causal_bundle_input carry no world_id. Chunk-3 (read-only
Actor page) does not need it: event_participant is reached only through its world-scoped parent
canon_event; the other two are not read in chunk-3. The new perception_subject carries world_id
from birth. **Firing trigger:** when SPEC-009 row-based sharding is implemented, these three must
either gain world_id as the distribution key OR be co-located by their world-scoped parent —
decided then. Until then, unchanged. No frozen-DDL change in chunk-3.

## SPEC-010 — published-schema nullability matches the engine DDL (chunk-4 audit)
**Verified contract:** `perception_record.confidence` is `REAL NOT NULL DEFAULT 1.0` (Master DDL,
doc 03 §1.3; migration `20260610090004`), so `confidence` on every projected page/timeline item is
**non-nullable**. The chunk-4 schemas `location_page/1`, `artifact_page/1`, `timeline/1` initially
typed it `["number","null"]` (over-permissive — the pgTAP/Go suite cannot catch over-permissiveness
because real non-null payloads validate against a nullable schema; only a field-by-field DDL check
catches it). Corrected to `"number"`. `actor_page/1` was already correct and is unchanged.
**Standing rule:** a published projection field's nullability must match its source —
NOT-NULL column ⇒ non-nullable type; nullable column or `fn_perceived_name` (withheld ⇒ NULL) ⇒
nullable type (`perceived_name`/`group_label`/`display_label` stay nullable). Deferred placeholder
fields hardcoded to `NULL` (`part_of`, `current_synthesis`, `last_known_status`, `perceived_type`,
`last_known_location`, `current_holder_owner_access`) stay `["string","null"]` — forward-compatible
for when their lens lands. If `confidence`'s engine DDL ever becomes nullable, bump `schema_version`.
