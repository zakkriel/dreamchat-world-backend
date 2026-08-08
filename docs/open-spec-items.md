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

## SPEC-011 — standing payload↔schema contract CI test (LANDED)
**Status: LANDED.** Closes the gap SPEC-010 exposed: the pgTAP/Go suites structurally cannot catch an
*over-permissive* published schema, because a non-null payload validates fine against a nullable
schema. A standing CI test (`make schema-contract`, workflow `.github/workflows/schema-contract.yml`,
runner `ci/gen_payloads.sh` + `ci/schema_contract.py`) now checks the published schemas in **two
directions**:
1. **Payload → schema:** real payloads generated by calling the actual JSON functions
   (`fn_actor_page`/`fn_location_page`/`fn_artifact_page`/`fn_timeline`/`fn_compendium_index_json`) for
   every seeded entity, as BOTH the Player and Jonas viewers, validate against their published schema
   (matched by the payload's own `schema_version`). Catches over-tightening / structural drift.
2. **DDL → schema nullability:** every field sourced from a NOT NULL DDL column is typed non-nullable
   (`confidence` is `"number"`, never `["number","null"]`; `epistemic_type`/`content`/
   `occurred_at_tick`/`perception_id` likewise), and the genuinely-nullable fields
   (`perceived_name`/`group_label`/`display_label`) stay nullable. This is the direction that catches
   the SPEC-010 class — direction 1 alone would not have.
A `--selftest` mode flips `confidence` to nullable in each confidence-bearing schema and asserts
direction 2 fails on the mutant, so the check is not vacuously green.
**Scope:** test/CI only — no change to `core/db`, the SQL functions, the Go handlers, or
`core/api/schema` (those are correct post-SPEC-010). To extend the NOT-NULL field map, edit
`NOTNULL_FIELDS` in `ci/schema_contract.py` against the Master DDL (doc 03 §1.3).

## SPEC-012 — NPC cognition engine (deferred subsystem)
The perceive → appraise → believe → decide → act loop, LLM-run and event-driven, is its own
subsystem beside the canon and perception engines. NOT in Chunk 5. Captured in the Chunk-5 play-loop
architecture notes (`docs/superpowers/specs/2026-06-16-chunk-5-play-loop-architecture-notes.md` §5,
§7). **Firing trigger:** when the play loop + write-side perception generation are proven and the
world needs autonomous NPCs (post-Chunk-5).

## SPEC-013 — Outcome-resolution / adjudication engine (deferred)
Resolving *uncertain* actions (persuade, attack, search) requires a ruling (rules+dice or LLM) — the
path that puts the model in the trust/canon-authority position, distinct from the mechanically-
resolvable thin-slice primitives. Captured in the Chunk-5 play-loop architecture notes (§6, §7); it
fills the §8 turn-loop **resolve** stage (identity/passthrough in the thin slice).
**Firing trigger:** when the action set extends beyond the mechanically-resolvable primitives.

## SPEC-014 — Cascading-inference depth bound
Inference → act → others perceive → infer can cascade unbounded; bound its depth/fan-out
(cf. ADR-017 Traversal Matrix). Captured in the Chunk-5 play-loop architecture notes (§5).
**Firing trigger:** when NPC inference is implemented.

## SPEC-015 — Decomposition reliability + canon-authority boundary
The prose → events mapping must be correct, and the LLM must not emit events the player did not take
(inventing canon). Captured in the Chunk-5 play-loop architecture notes (§7, Leg 2). **Firing
trigger:** when the Chunk-5 Leg-2 LLM bridge is built.

## SPEC-016 — Per-attribute perceivability model
AttributeChanged needs an outwardly-visible vs hidden flag deciding discovery-on-inspection (Jimmy's
missing arm shows; Sabin's secret PhD doesn't). Captured in the Chunk-5 play-loop architecture notes
(§3, §4); its visible-vs-hidden split realizes the §15 deception-lives-in-the-world path.
**Firing trigger:** when AttributeChanged is implemented in Chunk 5.

## SPEC-017 — Move-validity / physical-possibility gate
The engine checks an action is physically possible against current world state (locked door,
collapsed bridge, not co-located with the target) before applying it. Captured in the Chunk-5
play-loop architecture notes (§6). **Firing trigger:** when ActorMoved / actions land in Chunk 5.

## SPEC-018 — Spatial engine (deferred subsystem)
A bounded subsystem beside the canon, perception, cognition (SPEC-012), and adjudication (SPEC-013)
engines, owning geometry only. **Coordinates recorded per place** (the only recorded geometric fact);
**distance derived** as the magnitude of the vector between coordinates; **travel time = distance /
speed**, where speed is a recorded property of the mover/mode (walk, mount, …), not of the place.
Two recorded inputs (coordinate-per-place, speed-per-mover); everything else derives — so
coordinate-derived distances are coherent by construction (three of them cannot violate the triangle
inequality), dissolving rather than merely containing the coupled-fact coherence problem. Distinct
from reachability — distance (how far) is SPEC-018; whether one can go at all (a wall, a locked door,
no path) stays with move-validity (SPEC-017). NOT in Chunk 5. Owns the deferred machinery: nested
coordinate frames (containment), dimensionality (2D/3D), non-geometric move overrides
(portals/teleport), record-on-first-use for emergent geography, and rich speed/terrain modifiers.
Captured in the Chunk-5 play-loop architecture notes (§11, §12).
**Standing rule (general):** *record independent facts directly; for coupled facts record the
generating structure and derive the facts.* **Firing trigger:** when travel must span more than a
hand-authored handful of places, or emergent geography appears (post thin-slice); the thin slice uses
hand-set coordinates + a flat default speed and needs none of the deferred machinery.

---

## FE architecture seams (A3, 2026-06-19 — chunk-6 pre-brainstorm)
SPEC-019…024 are the small, concrete seams owed by the FE rendering + theme architecture
(D-14 / D-15 / ADR-P019). Most fold into **Chunk 6** (cheap seams + shell); none touches the frozen
engine canon, an invariant, or the Master DDL. Cross-repo items (FE = `dreamchat-frontend`, plus a
small BE config) are flagged.

## SPEC-019 — World theme-token field
World data carries a small **theme-token field**: accent color, mood/treatment, ornament motif —
plain data read by the FE chrome theme (D-15), never genre labels the system understands (GA-3). It
evolves as module/world JSONB-style data, so it carries `schema_version` + runtime validation (D-4).
- **Owner:** Chunk 6 (BE side: expose the field on world data; FE side: read it into the chrome theme).
- **Expected outcome:** a theme-token shape on world data + the FE reading accent/mood/ornament as
  tokens (the "tokens are the floor" layer of D-15). No engine/DDL change.

## SPEC-020 — Configurable backend API base (FE)
The FE must reach the backend through a **single configurable base** (config/env var) at the existing
request chokepoint — required because FE and BE are separate Railway services (cross-origin). This is
also the only seam that keeps the Electron door open (SPEC-024).
- **Owner:** Chunk 6. **Cross-repo:** FE (`dreamchat-frontend`).
- **Expected outcome:** one config seam through the FE request chokepoint; no hardcoded backend URL.

## SPEC-021 — BE CORS allowing the FE origin
The backend must send CORS headers permitting the FE origin (FE and BE are separate Railway services).
Small backend config; pairs with SPEC-020.
- **Owner:** Chunk 6. **Cross-repo to-do:** small BE config (this repo).
- **Expected outcome:** CORS configured for the FE origin; no change to handlers or the perception
  boundary (B-1, I-3).
- **Status:** **LANDED (2026-08-07).** `core/api/cors.go`, wired around the **mux** in `main.go` so a
  preflight is answered before routing — the router matches only GET/POST and 404'd every `OPTIONS`,
  which is why the FE's text-beat POST (it preflights, `application/json`) could not reach the API at
  all. Allowlist is exact-match from **`DREAMCHAT_CORS_ORIGINS`** (comma-separated full origins);
  unset ⇒ CORS off and logged, `*` and scheme-less entries refuse to boot.
- **Contract, confirmed with the FE (2026-08-07):** local dev is **same-origin through the Vite
  proxy**, so no localhost origin is baked in anywhere — `http://localhost:5173` belongs in dev/staging
  config only, never in prod. Methods `GET, POST, OPTIONS`; allowed request header `Content-Type`
  only; no `Expose-Headers`. **No credentials** — the FE sends no cookies and no `Authorization` (the
  trace key is a query param), so `Access-Control-Allow-Credentials` is absent. The origin is echoed
  rather than `*`, which is what makes credentials a one-line change in one file **when the session
  model lands and a wildcard origin stops being legal**. The Continue POST is a simple request and is
  served by the same echo path. Handlers and the perception boundary are untouched (B-1, I-3).

## SPEC-022 — Dynamic multi-world id
No hardcoded world constant — **world id is runtime state** (multi-world from the start). Pairs with
the viewer-identity seam (`?viewer=` today; real session later — B1/auth).
- **Owner:** Chunk 6. **Cross-repo:** FE (`dreamchat-frontend`).
- **Expected outcome:** world id flows as runtime state through the FE; no hardcoded world constant.

## SPEC-023 — App shell with named slots (+ Aux docked ↔ full-screen)
An **app shell of named slots** (left rail, top bar, scene, right Aux, bottom input) — the neutral
skeleton D-15 names and the slot model modules compose into (D-2). The **Aux slot supports docked ↔
full-screen** via **one responsive component** (bleed-out), not two implementations.
- **Owner:** Chunk 6.  **Cross-repo:** FE (`dreamchat-frontend`).
- **Expected outcome:** named-slot shell + one responsive Aux component covering docked and
  full-screen; chunk-6 Aux gate = **Current + Known** only (Inspect/Intent land chunk 7).

## SPEC-024 — Electron-wrappable delivery target
**Delivery target:** web SPA now, **Electron-wrappable** later. The **configurable API base (SPEC-020)
is the only seam** keeping the desktop door open (near-zero rework). No Electron work now — this entry
records the constraint that nothing may close that door.
- **Owner:** standing constraint; revisit if/when a desktop build is wanted. **Cross-repo:** FE.
- **Firing trigger:** a decision to ship a desktop wrapper. Until then, keep the API-base seam
  (SPEC-020) the single source of the backend origin so the wrap stays near-zero rework.

---

## Gate & state model contracts (chunk-5.5 design, 2026-06-25)
SPEC-025…027 are the three new contracts owed by the **Gate & State Model — v2** design
(`docs/superpowers/specs/2026-06-25-gate-and-state-model.md`). All three are **DESIGN** — needs
running-code evidence before they become canon (D-9); each files as a SPEC when built. None touches
the frozen engine canon, an invariant, or the Master DDL at this stage.

## SPEC-025 — Status-effect catalog
A predefined catalog (the contract) mapping a status (`tied`, `limping`, `gagged`, `blinded`) to
`status → { impacts: [action axes], effect: prevent | modify(param, factor) }`. `prevent` hard-gates
`not-blocked` on that axis; `modify` scales a parameter and feeds `fits-time` (never gates). The
engine applies it generically — new statuses are new catalog rows, never new gate logic. Feeds
**`not-blocked` + `fits-time`**.
- **Source:** Gate & State Model — v2 §5 (`docs/superpowers/specs/2026-06-25-gate-and-state-model.md`).
- **Status:** **⚠ SUPERSEDED IN PART (2026-07-22, A11-final)** — there is NO predefined catalog:
  statuses/modifiers are **LLM-minted typed rows** inside the hardcoded action contracts (percentage
  modifiers on movement types; floor −100%, no cap). Prevention emerges from the arithmetic
  (−100% → speed 0), not from a `prevent` table. See `chunk-5.5-final/FINAL-action-contracts.md` §2/§8.
- **Firing trigger:** when statuses / the status-aware gate land beyond the reachability-only thin
  slice.

## SPEC-026 — Object-physics (size / weight / capacity)
Two independent, pure-arithmetic dimensions on every object: **volume** — a `size` 1–10 where each
tier holds 4× the previous (`size n = 4^(n-1)` base units), with `has-room` ⟺
`used_volume + 4^(size-1) ≤ volume_budget`; and **weight** — a `weight` per object vs a carrier's /
container's `max_load`, with `within-load` ⟺ `used_weight + weight ≤ max_load` (a carrier's
`max_load` is its strength dimension). The two are orthogonal. Feeds **`has-room` + `within-load`**.
- **Source:** Gate & State Model — v2 §7 (`docs/superpowers/specs/2026-06-25-gate-and-state-model.md`).
- **Status:** **⚠ SUPERSEDED IN PART (2026-07-22, founder-ruled)** — the volume half survives
  (measurements `max_room`/`occupied_room`; `has-room` computed at ask-time, never stored). The weight
  half changed: **`within-load` is DEAD as a blocker** ("it does not really support the status system,
  just works around it — get rid of it"). Weight CONSEQUENCES instead: eager recursive `carried_weight`
  recompute on any carry-chain commit + seeded `encumbered` status (movement −100%). Container formula:
  `(empty_weight + Σ contents) × modifier`, recursive. See `chunk-5.5-final/FINAL-action-contracts.md` §4.
- **Firing trigger:** when `ObjectRelocated` / capacity checks land beyond the reachability-only thin
  slice.

## SPEC-027 — Comprehension / language model
A `Communicated` event carries a **language**; an actor holds **known languages**; fan-out compares
the two → **full content** on a match, **content stripped, act-only** on a mismatch (the
non-comprehending listener perceives that someone spoke, at whom, the tone — not the meaning, B-7).
Binary understand/not for v1; partial fluency later. The reception/comprehension axis also gates
meaning-dependent outcomes at resolution (a non-comprehending target can't be talked into anything).
Feeds **fan-out → full vs act-only**.
- **Source:** Gate & State Model — v2 §8 (`docs/superpowers/specs/2026-06-25-gate-and-state-model.md`).
- **Status:** **DESIGN / needs running-code evidence (D-9).** File as a spec with empirical evidence
  when built.
- **Firing trigger:** when `Communicated` fan-out / comprehension-gated reception lands beyond the
  reachability-only thin slice.

---

## FE-discovered contract gap (chunk-6 frontend review, 2026-08-07)

## SPEC-028 — World management API (list / choose / create a world)
SPEC-022 requires world id to be **runtime state, multi-world from the start**, but nothing can answer
*which* worlds exist: the router registers exactly eight handlers — three page, three index, one
timeline, one beat (`core/api/main.go:54-64`) — and viewer identity resolves server-side to the world's
single actor named `'Player'` (`core/api/viewer.go:16-27`, "Auth/session out of scope this chunk").
So the FE can make the world id *flow* (URL-supplied, no compile-time constant) but cannot let anyone
**select** a world, and a world picker would require inventing an endpoint (forbidden —
`implementation_playbook_superpowers.md:90`). Minimum shape: **`GET /worlds`** returning the worlds the
caller may see (`id`, display label, and the SPEC-019 theme tokens), perception-bound like every other
payload — an unreachable world is **absent, not redacted** (B-1, I-3) — plus `schema_version` (D-4);
**world-scoped viewer resolution** to replace the single-`'Player'` default (pairs with the B1/auth
stub, `MASTER_INDEX.md:124`); and a **ruling** on whether `POST /worlds` exists or world creation is
declared seed/tooling-only for now (`MASTER_INDEX.md:125` lists B2 — World creation — as a planned
stub). A world list is a directory, not canon: no world *state* on this surface.
- **Source:** frontend review pass, `dreamchat-frontend` @ `main`; full context in
  `docs/superpowers/handovers/2026-08-07-frontend-needs-from-backend.md` §1.
- **Owner:** Chunk 6 (pairs with SPEC-019/020/021/022). **Cross-repo:** BE (this repo) + FE.
- **Expected outcome:** a `GET /worlds` directory payload + a world-scoped viewer seam + a recorded
  ruling on world creation. No engine/DDL change; no perception-boundary change.
- **Firing trigger:** fired — the FE shell is being built now and ships a URL-supplied world id with a
  dev default as the documented stub until this lands.

## SPEC-029 — Compendium projection lenses are stubs (chunk-4 gate blocker)
Every page endpoint returns its full contract shape while almost every field inside it is a literal
stub in `core/db/migrations/20260615090001_compendium_read_functions.sql`: `perceived_role`,
`current_synthesis`, `last_known_status`, `part_of`, `perceived_type`, `last_known_location` and
`current_holder_owner_access` ship `NULL` (`:100-102`, `:123-125`, `:146-149`); `known_artifacts`,
`inline_links`, `known_areas_inside` and `key_actors` ship `[]` (`:103`, `:105`, `:126-127`, `:129`,
`:151`); `collected_knowledge_groups` is always exactly one group keyed by the target's own id
(`:78-82`); and `decay.stale` is the literal `false` (`:70`, `:181`), so the decay language the PRDs
require can never render. A real Actor page today is an id, a perceived name, and one flat knowledge
list. This is not an FE gap — the frontend renders every field that is populated and renders each lens
only when non-empty, so absence never implies knowledge (B-1).
Unmeetable while the stubs stand: **Actors AC#3, AC#4, AC#10**; **Locations AC#2, AC#3, AC#4, AC#5,
AC#8**; **Artifacts AC#2, AC#3**; **Timeline AC#4** (no per-record version identity in the payload).
- **Source:** frontend review pass while building the Compendium pages onto the design system; full
  table in `docs/superpowers/handovers/2026-08-07-frontend-needs-from-backend.md` §2.5.
- **Owner:** reopens Chunk 4 (its gate is "all four PRDs' read-side ACs on seed",
  `implementation_playbook_superpowers.md:70`). **Cross-repo:** BE (this repo).
- **Expected outcome:** the deferred lenses computed from perception — synthesis, role/type,
  last-known, containment, co-location, carry state — plus a real `decay.stale`, or an explicit
  founder ruling that chunk 4's gate is deferred and why.
- **Firing trigger:** fired — the FE Compendium surfaces are built and waiting on data.

---

## BE-discovered contract gap (integration verification pass, 2026-08-07)

## SPEC-030 — No movement can be expressed through the beat API
`ActorMoved` can only ever target **the room the actor is already standing in**, so no player-stated
movement — a step through a door or a journey across the city — is reachable from any client.

The decompose seat may bind ids **only** from the candidate whitelist, and that whitelist is
assembled from *present* perception alone (`core/api/beathandler.go` `payload`): actors at the
viewer's location, the viewer's current location, and artifacts matched by
`attrs->>'location_id' = <here>` or `attrs->>'contained_by' = <viewer>`. Two independent
consequences, both verified against the seeded Drowned Lantern:

1. **No remote location is ever a candidate.** Exactly one location — the current one — is added.
   The play world has five (Alley, Cellar, Dock Street, Harbor Quarter of Vael, The Drowned Lantern)
   and four of them cannot be named in a beat. The code records the cause in its own comment: the
   "one-hop-known absent entities" set is *"a separate follow-up"* needing
   perception/knowledge-subject-link machinery *"that does not yet exist cleanly"*.
2. **Portals are not co-located artifacts.** Front Door, Back Door and Cellar Hatch carry
   `{"open":…,"locked":…,"connects":[locA,locB]}` and **no `location_id`**, so the co-located-artifact
   query never returns them either. The doors of the room you are standing in are invisible to
   decompose.

So the Journey (shipped in #32) has no reachable path, and neither does walking through a door.
- **Evidence:** the candidate query run by hand as Kade in the tavern returns exactly three artifacts
  — `Sealed Note (gray wax)`, `the bar`, `Ballast Crate` — and no door. The three portal rows carry
  `connects` and no `location_id`. The Journey's own tests build their chains in Go and seed their own
  far geometry (`core/api/journey_beat_test.go` passes `ToTargetID` directly); nothing exercises
  movement through `POST /worlds/{w}/beats`. Hand-driving `go through the Back Door` against the
  running server returns an empty chain, `committed: []`.
- **Source:** backend integration verification pass, 2026-08-07 (three-repo integration).
- **Owner:** unassigned — it reopens a question the Journey design (rung 2) left implicit.
  **Cross-repo:** BE (this repo). The FE renders the journey block already and needs no change.
- **Expected outcome:** a ruling on how a player names somewhere they are not standing, and on how a
  portal becomes nameable from the rooms it connects — the known-but-absent candidate set, a portal
  lens over `connects`, a stated-destination shape the place author can bind, or an explicit deferral
  saying stated movement waits for the spatial engine. **No mechanism is proposed here** (anti-drift,
  `implementation_playbook_superpowers.md:90`): the gap is documented, not invented around.
- **Firing trigger:** fired — movement is unreachable by any client, and the founder's own worked
  example for the Journey gate (walk out, get interrupted, restate, arrive) cannot be driven.
- **Status:** **LANDED (2026-08-08) — movement works; the Journey needs one more thing (below).**
  `core/api/beathandler.go` `payload` gained a second candidate source: the **portals whose
  `connects` contains this room**, and the **rooms on the far side of those portals**. Both come from
  what the actor perceives standing here — a door is part of the room you are in, and a visible exit
  tells you there is somewhere on the other side — so this adds no mechanism, it stops hiding what
  the room already contains. Labels are `fn_display_name` as everywhere else, ids stay real.
  - **Passage is not decided at candidate time.** A candidate is a thing you may NAME, never a thing
    you may DO. The accessibility floor (`fn_actor_move_permitted`, mirrored in `premiseHolds`) still
    requires a portal that is open ∧ ¬locked. Offering the target is precisely what lets the world
    refuse with a reason; hiding it is what made the refusal unreachable. Verified live: `go to the
    Alley` through the shut back door now binds and halts `premise_broken`, and the actor stays put.
  - **A defect this fix introduced, and fixed.** `buildScene` resolved the place as "the last
    candidate of kind `location`", correct only while exactly one location could be a candidate. With
    neighbours now offered, the scene endpoint named a room the player was not in. `PerceptionPayload`
    gained an explicit `Here`, and the scene resolves the place by id. Caught by hand-driving, not by
    any test — then pinned by one.
  - **Deliberately not added:** the absent-but-known set (`fn_entity_visible`). It *would* now yield
    cleanly, but for the seeded viewer it contains only the room he is in, so it would ship as an
    unexercised code path. It belongs with the long-range ruling below.
- **STILL OPEN — the Journey gate cannot be driven, and it is world content, not engine.** A journey
  starts only from an **over-budget** `ActorMoved`. The Drowned Lantern's tension is `tense` →
  **30 s budget** (`tensionBudgetSeconds`), and the farthest room reachable from it is the Alley at
  **29 s** (`fn_move_duration_actor`); Dock Street is 5 s and the Cellar 6 s. **Every destination in
  the world fits inside one beat**, so nothing can start a journey — the four rooms cluster within 40
  units while the Harbor Quarter frame that holds them is 2000×2000 and otherwise empty. Exercising
  the founder's worked example (walk out, get interrupted, restate, arrive) needs **one connected
  destination further than ~41 units from the tavern**. That is a place in his fiction with a name he
  owns, so it is not invented here.
- **Firing trigger:** movement half is delivered; the remaining half fires the moment a distant place
  is authored into the seed.
