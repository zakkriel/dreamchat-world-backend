# Increment 1 — Your world reaches everything that plays it (landing contract on `world_model/6`)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Scope is `AMENDMENT.md`, and nothing beyond it.** Read
`docs/superpowers/debates/2026-08-26-landing-contract-retarget/AMENDMENT.md` before this plan. It
supersedes `PROPOSAL.md` (`AMENDMENT.md:3-4`) and amends `prd_world_creation_depth.md`, whose mechanism —
`Landing{Declare,Parse,Apply,Refuse}`, readers as a sum type, coverage at registration, the runner owning
ids/ticks/class-resolution/provenance, `shares(key)`, explicit phases — **survived all three seats
unchanged** (`AMENDMENT.md:6-9`; the mechanism itself is `prd_world_creation_depth.md:86-141`). This plan
does not restate the amendment. Every task references it.

---

## Claims — the v6 sections and keys entering the coverage check in this increment

Required by `AMENDMENT.md:64-65`: every increment's plan opens with a `Claims:` list. Coverage runs over
**the union of all `Claims:` lists to date** and refuses to boot on an unclaimed leaf *within that union*
(`:66-67`); everything outside it is **printed as unclaimed remainder at every boot** (`:68-71`). This is
increment 1, so the union is exactly this list. Restated from `AMENDMENT.md:75-77` and expanded to keys,
because a section name is not a leaf and R2 diffs leaves (`prd_world_creation_depth.md:119-123`).

```
Claims: increment-1

world              name · premise · mood                                    SCHEMA-v2.md:47
excluded[]         (present, possibly empty — O11)                          SCHEMA-v2.md:48, SCHEMA-v3.md:64
vocabulary         media[] · movements[] · channels[] · conditions[]
                   · substances[]                                           SCHEMA-v2.md:50
                   channels[]: emitted_by · received_by · latency_class
                   · reach · decay · conceals                               SCHEMA-v2.md:72-73
                   conditions[]: alters                                     SCHEMA-v2.md:70-71
law[]              (with enforced_by and within scoping)                    SCHEMA-v2.md:51, AMENDMENT.md:37
entities[]         general keys: name · facets · within · seen_as
                   · capability{moves_by, carry_class} · senses{}
                   · supports[]                                             SCHEMA-v2.md:120-150
                   within is a general key, ungated                         SCHEMA-v6.md:69-70
  facet extent     extent_class · medium · tension                          SCHEMA-v2.md:28 (less within, v6:69)
  facet matter     bulk_class · integrity · size_class                      SCHEMA-v2.md:29
  facet agency     disposition[] · doing · pursuing[] · hiding              SCHEMA-v2.md:30
  facet passage    connects[2] · admits[] · obstructs[] · hazard_class      SCHEMA-v2.md:33
  facet collective legibility · interest · vulnerability                    SCHEMA-v2.md:36
offices[]          name · held_by · of · confers[] · succeeds_by            SCHEMA-v2.md:53, :152-154
history[]          events · per-holder knowledge · standing: disputed
                   · inferred_from                                          SCHEMA-v2.md:62, SCHEMA-v4.md:118
arrivals[]         keys defined by SCHEMA-v7.md — see Task 7 and D3         SCHEMA-v2.md:63, AMENDMENT.md:125
processes[]        acts_on · direction · rate_class · terminus              SCHEMA-v2.md:56
cycles[]           period_class · phases[]                                  SCHEMA-v2.md:57
                   processes[]/cycles[] are claimed ONLY for the AC-7
                   slice: one authored imminent change whose `when` is a
                   class the runner resolves to fire_at_tick               AMENDMENT.md:38, prd:186-192
```

**Unclaimed remainder — printed at every boot, deliberately unguarded this increment.** Six sections:
`standing[]`, `opposition[]`, `accumulators[]`, `indicators[]`, `traces[]`, `epochs[]`
(`SCHEMA-v2.md:54`, `:55`, `:58`, `:59`, `:60`, `:61`). Six facets: `holding`, `demand`, `borne`,
`motion`, `magnitude`, `record` (`SCHEMA-v2.md:31`, `:32`, `:34`, `:35`, `:37`, `:38`). 10 claimed
sections + 6 unclaimed = the 16 frozen sections (`SCHEMA-v3.md:16`; v3 D2 deleted `layers[]`). 5 claimed
facets + 6 unclaimed = the eleven frozen facets (`SCHEMA-v2.md:26-38`).

**The accepted cost, restated so nobody rediscovers it as a bug:** a field outside this slice can sit
unread and the check will not say so (`AMENDMENT.md:55-57`). It is accepted to break F6's cycle
(`FINDINGS_playloop.md:326`), it is bounded by the staging key, and it is visible on every start.

**A section may not be silently un-claimed** (`AMENDMENT.md:72-73`). Removing one needs a dated line in
the amendment. Task 2 makes that mechanical rather than trusted.

---

**Goal:** a world authored in `world_model/6` loads, validates, commits and plays, and adding the next
concept is one `Declaration` rather than new code across seven functions and one growing validator
(`2026-08-26-world-model-eight-increments.md:92-97`). Concretely: a machine artifact that cannot drift
from its Go type, a document validator that refuses malformed input with a named cause instead of
dropping it, a coverage index that prints what it is not yet guarding, a landing framework whose resolver
cannot become the god object, the eight stranded-capability obligations, and `prd_world_creation_depth.md`
§5's three concepts landing on it as the first customers.

**Architecture:** two halves that never see each other's types, which is the whole point of the round.
The **document validator** takes `*worldDoc` and **no resolver**; **`Refuse` takes `resolver` and no
document** (`AMENDMENT.md:104-106`, adopted verbatim as a type boundary from C20). Nine v6 rules quantify
over the whole document — R1 name resolution, R5, R6, R8, R9/O7, O8, O9, R13 — and every one of them has
exactly one legal home under that boundary, because the type it would need in order to migrate is not
reachable from where it would move. The **machine artifact** is generated, never hand-written, from a
single Go source of truth, so schema and service type cannot disagree; `additionalProperties:false` at
every level plus `DisallowUnknownFields` on the decoder means a key the seat invents is **refused, not
dropped** (`AMENDMENT.md:107-111`). **Coverage** is a static set-difference over schema × declarations,
restricted to the claimed slice, run at registration with no database and no items
(`prd_world_creation_depth.md:119-123`), and it **prints the remainder it is not checking**. The
**landing framework** is `prd_world_creation_depth.md:86-105` unchanged.

**Tech Stack:** Go (`core/api`, package `main` — the package is flat; `prompts/` and `schema/` are data
directories, not packages), PostgreSQL (dbmate migrations + pgTAP `core/db/tests/`), the published JSON
Schema set in `core/api/schema/` gated by `ci/schema_contract.py`. **No migration in this increment** —
see decision 7; every reader the first customers need already exists.

## Global Constraints (the fence — every task implicitly includes these)

- **Scope = `AMENDMENT.md`, exactly.** Not `PROPOSAL.md`, which is kept only as the attack target and is
  wrong in the ways `FINDINGS_*.md` record (`AMENDMENT.md:3-4`). Not the roadmap entry, which says so
  itself (`2026-08-26-world-model-eight-increments.md:99-102`). If a capability is not in the amendment,
  this increment does not build it.
- **The PRD's mechanism is not reopened.** `prd_world_creation_depth.md` §3 survived all three seats
  (`AMENDMENT.md:6-9`). An agent redesigning `Declaration`/`Landing` is reopening a settled decision.
- **The validator is whole-document; only COVERAGE is staged.** Standing gate 1 is *"All 13 refusal rules
  executable, run at emit and in CI over every document on disk"*
  (`2026-08-26-world-model-eight-increments.md:55-56`). The `Claims:` list narrows the **coverage index**
  and nothing else. R9/O7 refuses a `demand` with no supplier even though the `demand` facet is unclaimed.
  Two different gates at two different times — that distinction is the type boundary's whole justification
  (`AMENDMENT.md:104-106`).
- **Malformed input is refused, never dropped.** `additionalProperties:false` at every level **and**
  `DisallowUnknownFields`. Both, not either. A wrong-type, absent, null and empty fixture per claimed leaf
  must be refused with a **named cause** (`AMENDMENT.md:107-111`). Without both, the SPEC-035 class
  returns: `witnesses: "<uuid>"` as a bare string, committed with zero witnesses and no halt reason.
- **Two examples from two different worlds, or the feature is not understood yet**
  (`2026-08-26-world-model-eight-increments.md:37`). Three real documents exist (`G_grelda_by_simarch.md`,
  `G_marea_by_gamedesign.md`, `G_sueno_by_extraction.md`); a rule justified by one of them is not justified.
- **No identifier whose name could only have come from one world**
  (`2026-08-26-world-model-eight-increments.md:39-41`). `caste`, `granter`, `spectral` are not vocabulary
  the engine may name. Grammar is closed and ours; magnitudes are minted, open, per-world (`:42-44`).
- **Clean cutover** (`:51`, closed decision 1 at `:66`). No shims, no bridges, no deprecated paths, no
  aliases. The old format and its bespoke commit path are removed — **but not before Task 1 has taken its
  before-number**, because after the cutover the evidence no longer exists (`FINDINGS_playloop.md:332`).
- **§4 is a v7 delta, and v7 is the genesis seat's, not ours** (`AMENDMENT.md:130-131`). Author
  obligations only: no new facet, no new section, the facet list stays frozen at eleven. **Depend on
  `SCHEMA-v7.md`; do not duplicate it, and do not pre-empt a key name it owns.**
- **A surviving mutant is a failing build** (`ci/mutate.sh:16-17`) — with the one declared exception in
  Task 1, where SURVIVED is the expected result and is itself the evidence. For every field a change
  reads, name what happens when it is **absent, null, the wrong type, and empty** (`ci/mutate.sh:24-28`,
  `:36-37`). Seven code-path mutants were CAUGHT across SPEC-034/035 and the suite still shipped a silent
  drop, because every mutant asked "what if the code is wrong" and none asked "what if the INPUT is
  wrong" (`ci/mutate.sh:30-34`).
- **Cite, or it is not a constraint.** Every rule id, ADR, SPEC or line this plan or its PRs name must
  resolve — `ci/check_citations.sh`, gated by `citations.yml`. The gate asserts a cited **id** resolves
  and deliberately does not assert a cited **number** does; two of this round's numbers were wrong and
  only counting caught them (`AMENDMENT.md:154-163`). Count before you cite.
- **Battery per task:** the task's own battery, then the area gate —
  `make schema-contract` · `make reset && make schema-check` · `ci/check_citations.sh --selftest` (17
  probes) · `cd core/api && go test ./... -count=1 -run 'Governed|ADRs|Cors|Auth|SchemaVersion|Latitude'`
  · `../harness/check.sh`. Source: `./harness/review.sh` for the `contracts-and-platform` area.
- **TDD failing-test-first; one commit per task minimum; every commit names its exact files in `git add`,
  never `git add .`/`-A`. One increment = one PR**, carrying `Areas:`, `Reviewed-by:`, `Rules:`,
  `Learned:` and `Friction:` (`ci/check_round.sh`, `ci/check_closeout.sh:92-165`).

## Explicit non-goals (in the amendment, NOT in this increment — do not build them here)

- **The narrator receiving the world's own content.** `AMENDMENT.md:147` (C19 `accept`) states outright
  that this work is independent of the retarget and **does not wait on increment 1**. It is a separate
  round with its own three prompt builders (`AMENDMENT.md:143-146`).
- **The brief-to-document coverage check.** Ranked above the narrator work (`AMENDMENT.md:148-150`), and
  outside this plan's six deliverables. Naming it so its absence is a decision, not an oversight.
- **Error legibility under centralised class resolution.** `AMENDMENT.md:169-171` files it as *"still
  open, not decided here"* and orders it into increment 1's **design doc**, not its implementation. Seven
  refusals replaced, not ~40; 67 deleted (`worldgenesis.go:249-495` carries 67 `return refuse(` sites,
  counted). The class→number surface is **four** conversions, two of which **fail open** — `ELSE 50`
  silently 50 m and `ELSE 2` silently 2 s (`01_engine_capability_audit.md:71-82`). A design doc precedes
  Task 8; this plan does not answer the question and must not pretend to.
- **Increment 2's first design question** — whether a container declaring both `extent` and `matter`
  aggregates its contents into its own mass (`SCHEMA-v6.md:84-91`). Placement is settled: `extent` wins.

## Amendment compliance (amendment item → task)

| Amendment item | Task |
|---|---|
| §7:177-178 the four predicted mutation experiments, run | Task 1 (before), Task 7 (after) |
| §3.1:61-73 the staging key — `Claims:` written, not implied | Task 2 |
| §3.4:107-111 machine artifact, `additionalProperties:false` + `DisallowUnknownFields` | Task 3 |
| Standing gate 1 — 13 refusals executable | Task 4 |
| §4:126-127 the Ironmoor guard + both arrival floor refusals | Task 4 |
| §3.4:104-106 the type boundary (C20 verbatim) | Task 5 |
| §3.2:79-88 coverage bidirectional, claimed slice only | Task 6 |
| §3.1:68-71 the unclaimed remainder printed at every boot | Task 6 |
| §3.3:98-100 registration refuses a duplicate `(section, facet, key)` triple | Task 6 |
| §4:119-128 the eight stranded-capability obligations | Task 7 |
| §2:29-46 first customers — §5's three re-expressed in v6 | Task 8 |
| §6 acceptance criteria, unchanged | Task 8 |
| §7:172 this round owes a friction journal (C23) | Task 9 |
| §7:173-174 friction-log row 6, ruled WASTE and still live (C24) | Task 9 |

## Flagged plan-level decisions (grounded in the amendment; founder may veto)

1. **The Go type is the single source; the JSON Schema is generated from it.** "Schema plus Go type from
   one source so they cannot drift" (`AMENDMENT.md:107-109`) does not say which is the source. Chosen:
   the Go struct, with `world_model.v6.schema.json` emitted by reflection. Reason: `additionalProperties:
   false` is then emitted **mechanically at every level** rather than maintained by hand at four levels the
   way `world_genesis.v1.schema.json` does (`:7`, `:12`, `:39`, `:54`) — a hand-maintained invariant at
   every level is exactly the thing C22 says must not be trusted. `DisallowUnknownFields` comes free from
   the same struct. *Vetoable alternative:* JSON Schema as source with a Go generator; costs a
   JSON-Schema→Go codegen, which is strictly more code than reflection over a closed struct.
2. **The drift gate is a golden-file test with a `-update` flag**, not a new binary. `core/api` is a flat
   `package main` with no subpackages; introducing `cmd/` for one generator sets a structural precedent
   this increment does not need. `TestWorldModelSchemaCommitted` compares the generated bytes to the
   committed file and fails on any difference; `make world-model-schema` re-runs it with `-update` to
   rewrite. Same discipline as `schema.sql`: generated, never hand-edited, guarded by an exit code.
   *Vetoable alternative:* `core/api/cmd/worldmodelgen/main.go`.
3. **All 16 sections are declared in the artifact with closed key sets — including the six unclaimed
   ones.** Three endings were available and two are wrong: (a) omit unclaimed sections → a document
   carrying `opposition[]` is refused, which breaks all three real documents and the increment's own proof
   that *"the three existing documents round-trip with no field loss"*
   (`2026-08-26-world-model-eight-increments.md:128-129`); (b) declare them permissively → violates
   `additionalProperties:false` at every level and reintroduces the silent drop for six sections;
   (c) declare them with closed key sets, claimed by no landing, printed in the remainder. **(c).**
   Unclaimed ≠ unspecified. Key sets for the six come from `SCHEMA-v7.md` where it defines them, and
   otherwise from the mechanical union across the three v4 documents, with any divergence named in the
   task rather than silently resolved. **`arrivals[]` is claimed and its key set is v7's to settle** — the
   three documents disagree (`within` in `G_grelda_by_simarch.md:353`, `place` in
   `G_marea_by_gamedesign.md:369` and `G_sueno_by_extraction.md:125`), and this plan does not pick the
   winner.
4. **`world_model/6` joins `INPUT_CONTRACT_SCHEMAS`; it does not get a payload generator.** A published
   schema no payload exercised fails `ci/schema_contract.py:147-149`, and there are exactly two correct
   endings. This one is a seat contract — the structured-output leash — not an output projection
   (`ci/schema_contract.py:42-49`), which is precisely what that set is for. One line, in the same commit
   as the schema file, or `make schema-contract` goes red on merge.
5. **The `Claims:` list is code, and this document is asserted against it.** `AMENDMENT.md:66` makes
   registration compute over the union of claims, so the union must be machine-readable; `:72-73` forbids
   silent un-claiming, so the document and the code must not be able to disagree. A test parses this
   file's fenced `Claims:` block and requires equality with the Go manifest. *Consequence, stated:* editing
   the claim in code without editing this plan fails the build, and vice versa.
6. **The unclaimed remainder is a log line at every boot, not a failure.** `AMENDMENT.md:68-71`. It prints
   through the existing boot-refusal surface in `main.go` (`:103-106`, `:157`), before
   `http.ListenAndServe` (`:175`). An **unclaimed leaf inside the union** is `log.Fatalf`; the remainder
   **outside** it is `log.Printf`. Two different severities on purpose — conflating them recreates F6's
   cycle.
7. **No migration.** Every reader the first customers need already exists: `collectives` mints into
   `entity_registry`, *"legal today, no DDL, unused by genesis"* (`prd_world_creation_depth.md:171-172`);
   `norms` writes a top-level key in the existing traits jsonb beside `speech_manner`
   (`:176-178`); `near_future` writes `pending_event`, read on every clock crossing and *"today written by
   nothing but three test inserts"* (`:186-189`). Tagline, ornament and per-place tension all land in
   columns and `attrs` keys that exist. If an agent finds itself writing DDL in this increment, it has
   left the amendment — stop and report.
8. **The "eleven acceptance criteria" is a miscount, and this plan carries the criteria unchanged while
   naming it.** `2026-08-26-world-model-eight-increments.md:123-124` says *"§6's eleven acceptance criteria
   stand unchanged"* and `AMENDMENT.md:42` says *"the only one of the eleven"*. Counted:
   `prd_world_creation_depth.md` carries **twelve** numbered ACs — AC-1…4 in §4 (`:145`, `:148`, `:152`,
   `:159`), AC-5…8 in §5 (`:168`, `:174`, `:186`, `:193`), AC-9…12 in §6 (`:206`, `:209`, `:212`, `:217`).
   §6 has **four**, which `AMENDMENT.md:20` itself says (*"§6's four written ACs"*). **Ruling:** the
   criteria are unchanged; the count is wrong; the coherent eleven is **AC-1…AC-11**, with AC-12 — the
   pre-spend review surface — belonging to increment 8, where `AMENDMENT.md:121` already sends invented
   content. This is the third instance of the class C8 and C21 caught and `ci/check_citations.sh` cannot:
   a citation whose **id** resolves and whose **number** does not. *Vetoable:* if the founder meant a
   different eleven, say which and Task 8's battery changes; nothing else in this plan moves.

## File structure

- **Go, new:** `core/api/worldmodel.go` (the source-of-truth type + facet types + the claims manifest),
  `core/api/worldmodelschema.go` (reflection → JSON Schema), `core/api/worldmodelvalidate.go` (the 13
  refusals + the three carried guards; takes the document, no resolver), `core/api/landing.go`
  (`Declaration`, `Landing`, `Reader`, `EventMode`, `Phase`, `resolver`), `core/api/landingregistry.go`
  (registration, coverage, remainder, triple-collision), `core/api/landingcollectives.go`,
  `core/api/landinglaw.go`, `core/api/landingnearfuture.go`.
- **Go, tests:** `worldmodelschema_test.go`, `worldmodelvalidate_test.go`, `landingregistry_test.go`,
  `landingboundary_test.go`, `landingcustomers_test.go`, `claimsmanifest_test.go`.
- **Schema, generated:** `core/api/schema/world_model.v6.schema.json` (+ the
  `INPUT_CONTRACT_SCHEMAS` row in `ci/schema_contract.py:49`).
- **Removed at cutover (Task 8, after Task 1 has its before-number):** `core/api/worldgenesis.go`,
  `core/api/worldgenesiscommit.go` and the bespoke commit path they carry (closed decision 1,
  `2026-08-26-world-model-eight-increments.md:66`).
- **Docs:** `docs/superpowers/debates/2026-08-26-landing-contract-retarget/MUTATION-2026-08-27.md`
  (Task 1's recorded table), `docs/00_workspace/friction/<round>.md` (Task 9), and the amendment gains one
  dated line if any claim moves.
- **Makefile:** `world-model-schema` target.

**Branch:** `feat/increment-1-landing-contract`, off current `origin/main` — check it, the workspace has
been bitten by a stale base (`AGENTS.md` pre-flight; gated by `.github/workflows/branch-currency.yml`).

**Depends on, and does not duplicate:** `SCHEMA-v7.md`, written in parallel by the genesis seat. Task 7 is
BLOCKED on it. Tasks 1–6 and 8 are not: v7 is author obligations only, no new facet and no new section
(`AMENDMENT.md:130-131`), so the artifact's *shape* comes from v6 and only the obligations wait.

---

### Task 1: Run the four predicted mutation experiments — the before-number, taken first

**Why this task is first, and cannot be later.** `FINDINGS_contracts.md` §7 predicted four verdicts and
ran none; `AMENDMENT.md:177-178` orders the implementing round to run them, because *"a prediction is not
a result"*, and if any is CAUGHT the matching finding weakens. All four probe behaviour in
`worldgenesiscommit.go` and `worldgenesis.go` — files this increment **deletes** at cutover (closed
decision 1). Run after the rebuild and there is nothing to mutate. This is F12's defect
(*"no before-number, and after increment 1 there cannot be one"*, `FINDINGS_playloop.md:332`) caught one
task before it happens.

**Files:**
- Create: `docs/superpowers/debates/2026-08-26-landing-contract-retarget/MUTATION-2026-08-27.md`
- Modify: nothing. `ci/mutate.sh` backs the file up, applies each mutant with `sed -i`, runs the test, and
  restores — including on interrupt (`ci/mutate.sh:45-46`).

**Declared exception to the fence.** `ci/mutate.sh` exits 0 only if every mutant was CAUGHT
(`:46`), and a SURVIVED mutant is normally a failing build (`:16-17`). Here **SURVIVED is the predicted
result and is the evidence** for C11, C12, C13 and C14. This task therefore expects a **non-zero exit**
and records the table; it is the only task in this plan permitted to. Task 7 re-runs the equivalent
mutants against the new path and requires exit 0.

- [ ] **Step 1: Trust the harness before trusting a verdict from it.** `ci/mutate.sh --selftest` (7
      probes) must pass. A NO-OP verdict is a typo in a sed script, not a result, and it fails the build
      precisely because *"a mutation experiment that silently tested nothing is worse than none"*
      (`ci/mutate.sh:48-53`).
- [ ] **Step 2: Run all four**, each with a genesis-exercising test command. Note the corrected line for
      mutant 3: the ring constant is `ring := regionRadius * 0.6` at `worldgenesiscommit.go:898`;
      `:888-891` is the rationale comment, not the code.

| # | File:line | Mutant | Predicted | Why the prediction |
|---|---|---|---|---|
| 1 | `worldgenesiscommit.go:627` | delete the region `attrs.tension` write | **SURVIVED** | `trg_validate_tension` fires only `IF NEW.attrs ? 'tension'` (`core/db/schema.sql:3749`), so an unstamped location passes the trigger; `beatBudgetSeconds` COALESCEs a missing row to `'none'` (`core/api/tension.go:58`) and `none` maps to `math.MaxInt64` (`:38-39`). Infinite budget, nothing becomes a journey, no test asserts a finite one |
| 2 | `worldgenesiscommit.go:666` | delete the cast `attrs.max_load` write | **SURVIVED** | `fn_apply_carry_change` clears rather than sets on a NULL `max_load`, by design and by its own comment — *"v_cw > NULL (no max_load) is NULL → the ELSE clears — an unset capacity can't be exceeded"* (`core/db/schema.sql:937-938`). `encumbered` never sets |
| 3 | `worldgenesiscommit.go:898` | `regionRadius * 0.6` → `regionRadius * 0.05` | **SURVIVED** | AC-3 exists *because* this world-feel constant has no owner and no gate (`prd_world_creation_depth.md:152-158`) |
| 4 | `worldgenesis.go:253-255` | delete the tagline refusal | **SURVIVED** | the consequence is a missing cover in `fillScenes`, which selects world covers `WHERE w.tagline IS NOT NULL` (`core/api/imagehandler.go:691-695`); no genesis test exercises it |

- [ ] **Step 3: Record the table `ci/mutate.sh` prints, verbatim**, in `MUTATION-2026-08-27.md`, with the
      command line for each mutant so it is reproducible. Verbatim: a paraphrased mutation report is an
      opinion.
- [ ] **Step 4: Re-disposition honestly.** For each mutant that came back **CAUGHT**, the matching finding
      (1→C11, 2→C14, 3→C12, 4→C13) is weaker than stated: record which test caught it, and say so in the
      PR body. `AMENDMENT.md:177-178` requires this either way. Do **not** quietly drop a weakened finding
      — §4's decision for it still stands unless the founder changes it.
- [ ] **Step 5: Commit** — exact files — `test(mutation): the four predicted verdicts, run before the cutover deletes the evidence [inc1-T1]`

**Battery:** `ci/mutate.sh --selftest` green · the four runs recorded · `cd core/api && go test -count=1 ./...`
unchanged from `main` (this task modifies no source) · area gate.

---

### Task 2: The claimed slice, written once and machine-readable

**Why second.** Every later task keys off it: the artifact marks which leaves are in the slice, the
coverage index diffs against it, and the remainder printer enumerates its complement. Written before any
of them, it cannot be retrofitted to match whatever got built.

**Files:**
- Create: `core/api/worldmodel.go` (the manifest only, at this task), `core/api/claimsmanifest_test.go`

**Interfaces:**
- Produces (consumed by Tasks 3 and 6):
  - `type claimedLeaf struct { Section, Facet, Key string }` — `Facet` empty for a non-facet key. The
    triple is the unit `AMENDMENT.md:99-100` names for collision detection, so it is the unit here too.
  - `var increment1Claims []claimedLeaf` — exactly this plan's `Claims:` block, no more.
  - `func claimedSlice() map[claimedLeaf]bool` — the union of all increments' claims to date. Increment 1
    is the whole union; the function exists so increment 2 appends rather than edits.
  - `func unclaimedSections() []string`, `func unclaimedFacets() []string` — the complement, derived from
    the artifact's declared sections/facets (Task 3) rather than hand-listed, so the remainder cannot
    silently shrink when a section is added.

- [ ] **Step 1: Failing test** `claimsmanifest_test.go`: (a) parse the fenced `Claims:` block out of
      `docs/superpowers/plans/2026-08-27-increment-1-landing-contract.md` and assert set-equality with
      `increment1Claims` — decision 5, and the mechanical form of *"a section may not be silently
      un-claimed"* (`AMENDMENT.md:72-73`); (b) assert the claimed sections number 10 and, with the
      unclaimed six, total the 16 frozen sections (`SCHEMA-v3.md:16`); (c) assert claimed facets number 5
      and total eleven with the unclaimed six (`SCHEMA-v2.md:26-38`); (d) assert no claimed leaf names a
      facet outside the eleven, and none names `layers[]` (deleted by v3 D2).
- [ ] **Step 2: Run** → FAIL (`worldmodel.go` missing).
- [ ] **Step 3: Implement** the manifest, transcribed from the `Claims:` block above. Transcribe; do not
      re-derive from the schema documents, or the two sources stop being one.
- [ ] **Step 4: Full battery** green.
- [ ] **Step 5: Commit** — `feat(claims): the increment-1 claimed slice, asserted against its own plan document [inc1-T2]`

**Battery:** `cd core/api && go test -count=1 ./...` · area gate. No DB touched — the manifest is static
data, and the test that reads a doc file must not need Postgres.

---

### Task 3: The machine artifact — schema and Go type from one source, refuse-not-drop

**Why third.** The validator (Task 4) needs a type to take, and the coverage index (Task 6) needs a leaf
enumeration to diff against. Both are downstream of this. It is not first only because Task 1 must precede
every source change and Task 2 defines what "in the slice" means.

**Files:**
- Create: `core/api/worldmodelschema.go`, `core/api/worldmodelschema_test.go`
- Modify: `core/api/worldmodel.go` (the document type), `ci/schema_contract.py:49`
  (`INPUT_CONTRACT_SCHEMAS`), `Makefile` (`world-model-schema`)
- Generate: `core/api/schema/world_model.v6.schema.json`

**Interfaces:**
- Produces (consumed by Tasks 4, 5, 6, 8):
  - `type worldDoc struct{…}` — the source of truth. All 16 sections (decision 3), the eleven facets as
    optional facet structs on `entities[]`, `entities[]` recursive (`SCHEMA-v2.md:52`), `within` a general
    key on any entity and **not** an `extent` key (`SCHEMA-v6.md:69-70`). Every struct closed.
  - `func worldModelSchemaJSON() ([]byte, error)` — reflection over `worldDoc` emitting draft-07 with
    `"additionalProperties": false` on **every** object node, and `$id: "world_model/6"`. Array ceilings
    (`maxItems`) come from struct tags and are populated in Task 7, not invented here.
  - `func decodeWorldDoc(r io.Reader) (*worldDoc, error)` — `json.Decoder` with `DisallowUnknownFields()`,
    returning an error that **names the offending key and its path**. A decoder that refuses without
    saying what it refused is a silent drop with extra steps.
- Consumes: `claimedSlice()` (Task 2), only to mark leaves; the artifact declares the full 16 sections
  regardless of claim status.

- [ ] **Step 1: Failing tests** `worldmodelschema_test.go`: (a) **every** object node in the generated
      schema — walked recursively, the way `ci/schema_contract.py:62-73` walks published schemas — carries
      `additionalProperties:false`; assert on the count of object nodes so a node added later without the
      flag fails; (b) `decodeWorldDoc` on a document with one invented key returns an error naming that
      key, and the returned `*worldDoc` is nil — this is the SPEC-035 class, and the assertion is that it
      is **refused**, not that it round-trips; (c) all 16 sections present, `layers[]` absent; (d) all
      eleven facets present on the entity type; (e) `within` accepted on an entity with no `extent` facet
      and rejected as an `extent`-gated key (`SCHEMA-v6.md:69-70`); (f) `TestWorldModelSchemaCommitted` —
      the generated bytes equal `core/api/schema/world_model.v6.schema.json` byte-for-byte.
- [ ] **Step 2: The four input-shape questions, as tests, per claimed leaf** (`ci/mutate.sh:24-28`,
      `:36-37`; `AMENDMENT.md:109-111`). For every leaf in `claimedSlice()`, a table-driven case for
      **absent · null · wrong type · empty**, each asserting a refusal with a named cause. Generate the
      table from `increment1Claims` so a leaf claimed later cannot skip its four questions. Absent is the
      one that needs a per-leaf answer rather than a blanket refusal: an optional leaf's absence is legal,
      and the test asserts which — so "optional" is written down per leaf instead of assumed.
- [ ] **Step 3: Run** → FAIL.
- [ ] **Step 4: Implement** the type, the reflection generator and the decoder. Add `world_model/6` to
      `INPUT_CONTRACT_SCHEMAS` (`ci/schema_contract.py:49`) **in this commit** — decision 4; without it
      `make schema-contract` fails on a published schema no payload exercised
      (`ci/schema_contract.py:147-149`). Add the `world-model-schema` Makefile target.
- [ ] **Step 5: Round-trip the three real documents.** `G_grelda_by_simarch.md`,
      `G_marea_by_gamedesign.md`, `G_sueno_by_extraction.md` decode with no field loss
      (`2026-08-26-world-model-eight-increments.md:128-129`). Where a document disagrees with another on a
      key name — `arrivals[].within` vs `arrivals[].place` (`G_grelda_by_simarch.md:353` vs
      `G_marea_by_gamedesign.md:369`, `G_sueno_by_extraction.md:125`) — **do not pick a winner**; mark the
      case `t.Skip` with a comment naming SCHEMA-v7.md and Task 7, and list it in the PR body. Decision 3.
- [ ] **Step 6: Full battery** green, including `make schema-contract`.
- [ ] **Step 7: Commit** — exact files — `feat(contract): world_model/6 as a machine artifact — one source, closed at every level, unknown keys refused [inc1-T3]`

**Battery:** `cd core/api && go test -count=1 ./...` · `make schema-contract` · `make reset && make schema-check`
· `make world-model-schema && git diff --exit-code core/api/schema/world_model.v6.schema.json` · area gate.

---

### Task 4: The document validator — the document, and no resolver

**Why fourth.** It consumes `*worldDoc` and nothing else, so it can be written the moment the type exists,
and it must exist before any landing does — standing gate 1 is *green at the end of every increment*
(`2026-08-26-world-model-eight-increments.md:53-56`), and today **no document in this project has ever
been validated** (`:56`).

**Files:**
- Create: `core/api/worldmodelvalidate.go`, `core/api/worldmodelvalidate_test.go`

**Interfaces:**
- Produces (consumed by Tasks 5, 6, 8):
  - `func validateDoc(d *worldDoc) []violation` — **takes the document and no resolver**
    (`AMENDMENT.md:104-106`). `violation{Rule, Path, Cause}`. A document violating any refusal is
    **rejected whole, with the reason named; there is no partial build** (`SCHEMA-v3.md:85`).
  - The 13 refusals, each its own function, each with a worked example from **two different worlds** in its
    comment (`2026-08-26-world-model-eight-increments.md:37`): R1 unresolved name reference; R2
    `passage.connects` ≠ exactly 2 extents; R3 a number in an engine-computed field; R4 a class outside its
    ladder; R5 `magnitude` referenced individually; R6 `excluded[]` contradicted by an authored entity; R7
    a facet key without its facet; R8 `within` cycle; R9 `demand` with no supplier; R10 `agency` with no
    `pursuing`; R11 threshold ladder out of order or with a repeated `at`; R12 `history` entry
    `standing:"disputed"` whose knowledge holders all agree (`SCHEMA-v3.md:70-84`); R13 an `inferred_from`
    chain that does not terminate in stated content (`SCHEMA-v4.md:118`).
  - **The two arrival floor refusals, executable, not sufficiency prose** (`AMENDMENT.md:127`): nothing
    leads out of the arrival extent, and nobody is in it when the player walks in. Carried from
    `worldgenesis.go:380-382` and `:393-395`, whose messages name the place — keep that shape.
  - **The Ironmoor guard** (`AMENDMENT.md:126`): a person's name that is machine-shaped — snake_case, or
    cased script with no capital anywhere — is refused. Carried from `worldgenesis.go:541-545`, whose
    comment records the live incident of 2026-08-20: genesis emitted slug join-keys as canonical names, the
    registry stored them, the naming wall guarded strings no model writes, and the player read two
    humanised slugs. A guard with a logged incident behind it is not dropped in a rewrite.
- **Boundary (the point of the whole task):** `worldmodelvalidate.go` must not import, reference or accept
  a `resolver`. Nine rules — R1, R5, R6, R8, R9/O7, O8, O9, R13, and O-cross-checks — quantify over the
  whole document, and under a resolver-shaped signature every one of them would pull the resolver into
  document scope. That is §9 Q1's failure mode exactly. Task 5 makes it a compile-time fact.

- [ ] **Step 1: Failing tests** `worldmodelvalidate_test.go`: one positive and one negative fixture per
      refusal — 13 + 2 floors + 1 name guard = **16 rules, 32 cases**, each asserting the rule id **and**
      the path named in the violation. Fixtures come from two different worlds wherever the rule permits.
      Plus: a valid document produces zero violations, and `validateDoc` compiles with no `resolver` in
      scope.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement.** Each rule is a named function; the error message names the rule, the path and
      the cause. Preserve the two message shapes worth keeping: the closed-set form
      `"%q's trait %q has strength %q, outside the closed set"` (`worldgenesis.go:328`), which is what the
      seven class refusals a resolver would replace look like today, and the arrival floors' place-naming
      form. 67 named refusals are being deleted (`worldgenesis.go:249-495`); the replacement set must not
      lose the information those messages carried.
- [ ] **Step 4: Run it over every document on disk** — standing gate 1's second half
      (`2026-08-26-world-model-eight-increments.md:55-56`). A CI-visible target that validates all three
      v4 documents plus any test world. Documents that fail: record the failures, do **not** weaken a rule
      to make one pass. A rule bent to fit a document is the ontology problem AC-11 warns about
      (`prd_world_creation_depth.md:216`).
- [ ] **Step 5: Full battery** green. **Step 6: Commit** — exact files — `feat(validate): all 13 refusals executable, plus both arrival floors and the Ironmoor name guard; document in, no resolver [inc1-T4]`

**Battery:** `cd core/api && go test -count=1 ./...` · the all-documents validation run, output recorded ·
area gate.

---

### Task 5: The landing framework — `Refuse` takes the resolver and no document

**Why fifth.** It defines `Declaration`, which is the coverage index's left operand; Task 6 cannot compile
before it. It comes after the validator so that the type boundary is created with **both** sides already
written — a boundary asserted before either side exists is a comment.

**Files:**
- Create: `core/api/landing.go`, `core/api/landingboundary_test.go`

**Interfaces:**
- Produces (consumed by Tasks 6 and 8): `Declaration{Concept, Consumes []LeafPath, Mints []EntityKind,
  Event EventMode, Readers []Reader, Phase, DependsOn}` and `Landing{Declare, Parse, Apply, Refuse}`,
  **unchanged from `prd_world_creation_depth.md:86-105`**. `Reader` is the sum type
  `state(path) | perception(holderRule) | referenced(concept)` (`:111-114`); a declaration with zero
  readers fails registration (`:117`). `EventMode` includes `shares(key)` (`:136-137`). `Phase` is
  `content | arrival`, separate transactions, arrival retryable (`:139-141`).
- **The type boundary, and it is the deliverable** (`AMENDMENT.md:104-106`):
  > The document validator takes `*worldDoc` and **no resolver**. `Refuse` takes `resolver` and **no
  > document**. Neither type is in the other's scope.
  `Refuse(item, resolver) error` — `item` is one parsed element, never the document. `resolver` exposes
  other concepts' minted ids (`prd_world_creation_depth.md:102`) and exposes **no** document accessor.
- **R3 — the runner owns the invariants, once** (`prd_world_creation_depth.md:125-129`): it mints every
  uuid, assigns every tick and `beat_seq`, resolves every class to its number, stamps `source_event_id`
  and asserts `acquired_tick ≥` its event's tick. A landing containing a uuid, tick, coordinate or
  class→number call is a defect, and AC-2 makes that assertion testable
  (`prd_world_creation_depth.md:148-151`).
- **R4** — the runner refuses `references(world_genesis)` for any perception that is not a name
  (`prd_world_creation_depth.md:131-134`), centrally, not patched per concept.

- [ ] **Step 1: Failing tests** `landingboundary_test.go`: (a) a test landing whose `Refuse` tries to reach
      the document does not compile — assert with a compile-check fixture under `go vet`-visible build
      tags, or an explicit `var _ = func(r resolver) { _ = r.document }` negative that must fail to build,
      recorded as a golden compiler error; (b) a declaration with zero readers fails registration with a
      named error (AC-1, `prd_world_creation_depth.md:145-147`); (c) a landing attempting
      `references(world_genesis)` for a non-name perception is refused **by the runner**, not by a
      concept-specific check (AC-2, `:148-151`); (d) a grep-style assertion that no file matching
      `landing*.go` contains a uuid literal, a tick constant or a class→number map — AC-2's mechanical
      half.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** the types and the runner-side invariants.
- [ ] **Step 4: State the bound in the package comment**, in the amendment's own words, with the citation.
      Nine rules have exactly one legal home because the type they would need in order to migrate is not
      reachable — write that down where the next agent will read it, not only here.
- [ ] **Step 5: Full battery** green. **Step 6: Commit** — exact files — `feat(landing): the contract types, with the resolver bounded by type — Refuse takes the resolver and no document [inc1-T5]`

**Battery:** `cd core/api && go test -count=1 ./...` · the compile-negative recorded · area gate.

---

### Task 6: The bidirectional coverage index — claimed slice only, remainder printed

**Why sixth.** Its left operand is the artifact's leaf enumeration (Task 3), its right operand is
`⋃ Consumes` over `Declaration` (Task 5), and its scope is the claimed slice (Task 2). It is the last of
the three that can be written, not a choice.

**Files:**
- Create: `core/api/landingregistry.go`, `core/api/landingregistry_test.go`
- Modify: `core/api/main.go` (register + check before `http.ListenAndServe`, `:175`)

**Interfaces:**
- Produces:
  - `func registerLandings(ls []Landing) error` — R1 (≥1 reader) and R2 (leaf coverage) **with no database
    and no items** (`prd_world_creation_depth.md:145-147`, `:121-123`).
  - **Direction 1** — every claimed leaf is claimed by at least one landing. An unclaimed leaf **inside the
    claimed slice** is a registration failure naming the leaf → `log.Fatalf`, the boot-refusal pattern
    already in `main.go:103-106` and `:157`.
  - **Direction 2** — every engine input has an author. The direction *"nobody ever computed"*
    (`AMENDMENT.md:81-86`), which found `perception_record.confidence` permanently 1.0, `distortion_level`
    written and read by nothing, `invalid_tick`/`expired_at` read on every knowledge path and written only
    by test fixtures, three `epistemic_type` values produced by no code, and
    `world_pressure(accrued, threshold)` touched by nothing. Both directions run over the claimed slice
    only (`AMENDMENT.md:88`).
  - `func unclaimedRemainder() []string` — printed at **every** boot as `log.Printf`, never fatal
    (decision 6, `AMENDMENT.md:68-71`). The six sections and six facets by name, with a one-line count. An
    exclusion nobody can see is the schema-validator defect again: `beat_frame.v3` carried
    `"format":"uuid"` 35 times and none was ever checked (`AMENDMENT.md:69-71`, failure-log #20).
  - **Triple-collision refusal** — registration refuses two landings claiming the same
    `(section, facet, key)` (`AMENDMENT.md:98-100`). A landing declares the **facet** it lands, not the
    section.
- **The weakened guarantee, stated in the code comment, not denied.** C7 stands and the amendment concedes
  it in full (`AMENDMENT.md:90-97`): `world_genesis/1` partitioned nouns into disjoint typed arrays so
  coverage was a disjoint union; v6 has one recursive `entities[]` whose shape is its `facets`, so the
  check weakens from *"every leaf is parsed by exactly one landing"* to *"every leaf is claimed by at
  least one landing."* The triple-collision refusal turns overlap into a named registration failure
  instead of a silent double-write; it does not restore the stronger property. Write that in
  `landingregistry.go`, with the citation, so the next reader does not assume the stronger one.

- [ ] **Step 1: Failing tests** `landingregistry_test.go`: (a) a deliberately-unclaimed test leaf **inside**
      the slice produces a named registration failure (AC-1's second half,
      `prd_world_creation_depth.md:145-147`); (b) a leaf **outside** the slice produces **no** failure and
      **does** appear in the remainder — the staged-coverage decision, tested in both directions; (c) a
      deliberately-inert test declaration (zero readers) fails registration; (d) two landings claiming the
      same `(section, facet, key)` fail registration naming the triple; (e) direction 2: an engine input
      inside the slice with no author is named; (f) the remainder lists exactly the six sections and six
      facets, and the test asserts the **names**, so adding a section without claiming it changes this test.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement**, and wire registration into `main.go` before
      `http.ListenAndServe` (`:175`), after the existing schema-drift refusal (`:103-106`).
- [ ] **Step 4: Boot the service and read the log.** The remainder must appear on a real start, not only in
      a test. This is the one deliverable whose whole purpose is being seen; verify it by looking at it.
      Record the exact line in the PR body.
- [ ] **Step 5: Full battery** green. **Step 6: Commit** — exact files — `feat(coverage): bidirectional index over the claimed slice, unclaimed remainder printed at every boot [inc1-T6]`

**Battery:** `cd core/api && go test -count=1 ./...` · a real boot with the remainder line captured ·
`make reset && make schema-check` · area gate.

---

### Task 7: The eight stranded-capability obligations — on `SCHEMA-v7.md`

**BLOCKED until `SCHEMA-v7.md` lands.** The genesis seat is writing it in parallel. **Depend on it; do not
duplicate it, and do not invent a key name it owns** (`AMENDMENT.md:130-131` — v7 is author obligations
only: no new facet, no new section, the facet list stays frozen at eleven). If v7 has not landed when this
task is reached, report BLOCKED naming the file; do not proceed by guessing key names, and in particular do
not settle `arrivals[]`'s `within`-vs-`place` divergence unilaterally (decision 3).

**Why seventh.** Six of the eight are validator rules or artifact constraints, so they need Tasks 3 and 4
in place; two are derivations that need the commit path the first customers exercise. And every one of them
is an author obligation whose text is v7's.

**Files:**
- Modify: `core/api/worldmodel.go` (`maxItems` struct tags, required-ness),
  `core/api/worldmodelvalidate.go` (the obligation rules), `core/api/schema/world_model.v6.schema.json`
  (regenerated), `core/api/worldmodelvalidate_test.go`

**The eight** (`AMENDMENT.md:119-128`) — decision and citation per row, not restated here:

| # | Capability | Decision | Where it lands |
|---|---|---|---|
| 1 | `world.tagline` → world cover art | **Derive** from `world.premise` | commit path; flagged — a derived tagline is a line the founder never approved, reviewed at increment 8's surface (`AMENDMENT.md:121`) |
| 2 | `world.ornament` | **Derive** from `world.mood` | commit path (`:122`) |
| 3 | per-place `tension` | **Obligate** on every `extent` entity; **no silent default — absence is a refusal, never a fallback** | validator (`:123`) |
| 4 | single root / coordinate origin | **Obligate exactly one root `extent`**; the 0.6 ring stays a named world-feel constant under AC-3 | validator (`:124`) |
| 5 | `arrival_candidates`, `arrival.why`, `newCast` | **Obligate** three candidates with exactly one chosen, and add `why` | validator + artifact (`:125`) |
| 6 | the Ironmoor guard | **Obligate** — already carried in Task 4; this row is the v7 text catching up | validator (`:126`) |
| 7 | the two arrival floor refusals | **Obligate** — already carried in Task 4; same | validator (`:127`) |
| 8 | array ceilings / tick ladder bound | **Obligate** — every array carries `maxItems` | artifact (`:128`) |

- [ ] **Step 1: Failing tests**, one positive and one negative per row. Row 3's negative is the important
      one: an `extent` entity with no `tension` is **refused**, and the test asserts the refusal names the
      entity — because the failure it replaces was silent at every layer (`schema.sql:3749`,
      `tension.go:58`, `:38-39`), which is why 6 of 8 extent entities in the Grelda document carry
      `tension` and the root and the granary do not (`AMENDMENT.md:123`).
- [ ] **Step 2: `maxItems` on every array** (row 8). Ceiling values come from v7, not from this plan.
      Rationale to carry across: *"Array ceilings bound COST, not shape"*
      (`core/api/schema/world_genesis.v1.schema.json:4`; e.g. `places` `minItems:2, maxItems:8` at
      `:48-49`). A schema-level test asserts **no array anywhere lacks `maxItems`**, so a section added
      later cannot skip it.
- [ ] **Step 3: Rows 1 and 2 are derivations, and their honesty is the deliverable.** Cover art is
      structurally gated on the tagline — *"no tagline, no cover, because there is nothing to render from.
      The gate is the data, not a promise"* (`imagehandler.go:662-663`; the predicate is
      `imagehandler.go:691-695` and `artcommission.go:66`). **Citation corrected:** `AMENDMENT.md:121`
      attributes that comment to `worldgenesiscommit.go:662-663`, which is an actor-descriptor write. The
      id resolved and the content did not — the fourth instance of the class C8 and C21 caught, and the
      second in this plan after decision 8. Fix it in the amendment when the round closes. Deriving it keeps the art path alive and
      converts a founder-approved line into an invented one. Mark the derived value's provenance as
      inferred so increment 8's surface can correct it, per `AMENDMENT.md:121`.
- [ ] **Step 4: Regenerate** the schema (`make world-model-schema`) and commit the regenerated file in the
      same commit as the tags — the artifact is generated, never hand-edited.
- [ ] **Step 5: Re-run the four mutation experiments against the NEW path** and require **CAUGHT** on all
      four. The old lines are gone by now, so these are the equivalent mutants: delete the `tension`
      obligation (row 3); delete the `max_load` write in the new commit path; change the ring factor at its
      new home; delete the tagline derivation (row 1). `ci/mutate.sh` must exit **0** here — Task 1's
      declared exception does not apply. If a mutant survives, the obligation is prose and the row is not
      done.
- [ ] **Step 6: Full battery** green. **Step 7: Commit** — exact files — `feat(obligations): the eight stranded capabilities — tension required, one root, three candidates, ceilings, tagline/ornament derived [inc1-T7]`

**Battery:** `cd core/api && go test -count=1 ./...` · `ci/mutate.sh` on all four equivalents, **exit 0** ·
`make world-model-schema && git diff --exit-code` · `make schema-contract` · area gate.

---

### Task 8: The first customers — `prd_world_creation_depth.md` §5's three, re-expressed in v6

**Why last of the build tasks.** They are the proof that the contract holds, and a proof that runs before
the thing it proves is a rehearsal. §5 and §6 **stand as written**; only the shape changes
(`AMENDMENT.md:31-33`). These three were kept over the proposal's four because they have **no cross-concept
grounding problem** — they test whether the contract holds without simultaneously testing whether the
resolver can reach across concepts (`AMENDMENT.md:40-43`).

**Files:**
- Create: `core/api/landingcollectives.go`, `core/api/landinglaw.go`, `core/api/landingnearfuture.go`,
  `core/api/landingcustomers_test.go`
- Remove (the cutover, closed decision 1 — Task 1 already has its before-number):
  `core/api/worldgenesis.go`, `core/api/worldgenesiscommit.go` and their bespoke commit path
- Modify: `core/api/worldgenesishandler.go` (route through the framework)

**The three** (`AMENDMENT.md:34-38`):

| §5 concept | v6 expression | §6 AC, unchanged |
|---|---|---|
| `collectives[]` | the `collective` facet + `offices[]` | AC-5, AC-8 |
| `norms[]` | `law[]`, with `enforced_by` and `within` scoping | AC-6, AC-9, AC-10 |
| `near_future[]` | one authored imminent change through `processes[]`/`cycles[]`, `when` as a class the runner resolves | AC-7 |

- **Constraint retained from §5:** all three are **optional**. A brief implying none authors none, at zero
  added token cost, and the pipeline is unchanged (`AMENDMENT.md:45-46`,
  `prd_world_creation_depth.md:165-166`).
- **Readers, unchanged from the PRD:** collectives mint `group` into `entity_registry`
  (`prd_world_creation_depth.md:171-172`); law reads `state(personality_core.traits.norms)`, a top-level
  key in the traits jsonb beside `speech_manner` (`:176-178`), with a naming-wall pass over norm text
  before render because trait JSON is emitted raw (`:182-183`); the imminent change reads
  `state(pending_event)`, read on every clock crossing (`:188`).

- [ ] **Step 1: Failing tests** `landingcustomers_test.go`, AC by AC:
      **AC-5** a brief implying a collective yields one queryable group row and resolvable membership; one
      implying none yields zero rows (`prd_world_creation_depth.md:172-173`).
      **AC-6** the law appears in the prompt of every mind it binds and **no other**, carries no magnitude,
      is unchanged by any personality-module update across a 20-beat run, and leaks no unearned name
      (`:184-185`).
      **AC-7** within the baseline's 5-beat window (`prd_world_creation.md:22`) the player perceives one
      authored event they did not cause, **in every world**, law or no law (`:191-192`). This is the only one of the criteria that puts
      something in front of a player inside the window the play-loop seat measured — ≤150 s of world time
      (`AMENDMENT.md:20`, `:42-43`) — so it is the one whose failure means the increment did not reach the
      game.
      **AC-8** two briefs differing only in `marked`/`concealed` produce different **knowledge
      distributions**, not different formatting; vacuous when no collective is authored (`:197-199`).
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** the three landings — each is **one `Declaration`**. If a
      landing needs a helper the framework does not provide, that is a finding about the framework, not a
      licence to write bespoke code; report it.
- [ ] **Step 4: The cutover.** Delete the old format and its bespoke commit path
      (`2026-08-26-world-model-eight-increments.md:66`). No shims, no aliases, no deprecated path. Name
      every deleted file in the commit and every caller migrated. **Do not start this step until Task 1's
      mutation table is recorded** — after this, it cannot be taken.
- [ ] **Step 5: §6's remaining criteria**, honestly scoped. **AC-9** — one norm-implying brief built with
      and without the `law` landing produces differing NPC decisions within five beats, **diffed, not
      asserted** (`prd_world_creation_depth.md:206-208`). **AC-10** — across the eval runs, at least one
      authored world produces an NPC act that contravenes a bound norm and the world responds (`:209-211`).
      **AC-11** — N≥20 planted briefs whose implied norms share no shape, plus a negative control; 0%
      invented structure, <10% missed structure; **sampled human audit per I-6, never a CI equality**
      (`:212-216`). AC-9 is a CI-runnable diff; AC-10 and AC-11 are eval-run criteria with a human in the
      loop and they gate the increment's close, not its build. Say which is which in the PR body — a
      criterion silently reclassified as CI is a criterion deleted. **AC-12 is increment 8's** (decision 8).
- [ ] **Step 6: The increment's proof** (`2026-08-26-world-model-eight-increments.md:128-130`): the three
      existing documents round-trip with no field loss; a pipeline-created world passes the validator and
      the coverage check **first try**; everything the person said survives byte-identical across two runs
      and only inferred parts differ; a world plays at least as well as today.
- [ ] **Step 7: Full battery** green. **Step 8: Commit** — exact files — `feat(customers): collectives, law and one imminent change land on the contract; old genesis path removed [inc1-T8]`

**Battery:** `make reset && make test` · `cd core/api && go test -count=1 ./...` (twice, for stability) ·
`make schema-check` · `make schema-contract` · the AC-7 five-beat run observed, not inferred · area gate.

---

### Task 9: The friction journal this round owes, and the ledger ruling

**Why this is a task and not a PR footer.** `AMENDMENT.md:172` records C23 as **owed before the
implementing round closes**. The first version of this harness asked for friction at close-out and got a
retrospective written from memory by someone who already knew how the story ended — it captured the six
big frictions and lost every small one, and the small ones are the signal (`harness/friction.sh:13-17`).
Friction is only recordable **while you are still confused** (`:19-22`).

**Files:**
- Create: `docs/00_workspace/friction/<round>.md` (via `./harness/friction.sh --open`)
- Modify: the PR body's `Friction:` line; `docs/00_workspace/friction-log.md` if row 6's fix lands

- [ ] **Step 1: Open the journal at the START of the round, not here.** `./harness/friction.sh --open
      "increment 1 landing contract"`. This checkbox sits in Task 9 for accounting; the command runs
      before Task 1. Log with `gap` / `conflict` / `surprise` / `decision` as they happen
      (`harness/friction.sh:4-9`, `:26-29`). The bar is deliberately low: **anything not expected**.
- [ ] **Step 2: The entry C23 already identified**, if the round meets it — file it as `gap`: the section,
      facet, obligation and refusal counts for `world_model/6` are not stated in one place, so an author
      reconstructs them from `SCHEMA-v3.md`'s summary table, which is **already known to be wrong in at
      least one row** (`SCHEMA-v5.md:65-66`). A table wrong in one row and authoritative in the others is
      how the "17 sections" and "25 obligations" errors happened in one document
      (`AMENDMENT.md:156-159`), and how decision 8's "eleven" happened in another. **Doc to fix:**
      `SCHEMA-v6.md` already carries *"Reader-obligation count: 24 → 25"* (`:78`) — it should carry
      sections, facets and refusals the same way.
- [ ] **Step 3: `Friction:` needs a verdict.** Every entry ends **EARNED**, **WASTE** or **UNCLEAR** — a
      description with no verdict is a complaint, and no rule can die from a complaint
      (`ci/check_closeout.sh:161-164`). `Friction: none` is legitimate but must be reasoned after a dash
      (`:140-142`, `:157-159`).
- [ ] **Step 4: Row 6 of the ledger — WASTE, upheld, and still live** (`AMENDMENT.md:173-174`).
      `AREAS.map` membership is by indentation and an unindented line silently becomes an area name:
      `harness/check.sh:406` takes every unindented line as an area name with no validation, and the
      close-out reader at `:586` skips a phantom area with zero globs, so the second reader cannot catch
      what the first invented. **The named fix:** `check_areas` refuses an area name for which neither
      `docs/areas/<name>.md` nor `harness/roles/<name>-expert.md` exists — one condition in the loop at
      `harness/check.sh:416-426`. It is one check and has already cost two agents. **This is a workspace
      change, not a backend one**: it belongs in the workspace-harness PR of this round, opened together
      with the backend PR (round-protocol §0). If it does not land here, say why in the ledger; do not
      leave it silently live for a third round.
- [ ] **Step 5: `Learned:` must name a file that is in the diff** (`ci/check_closeout.sh:110-125`). Point at
      the dossier, the failure log, the map or the checklist actually updated. A round that teaches nobody
      anything leaves the next reviewer exactly as ignorant as this one was (`:92-93`).
- [ ] **Step 6: Commit** — exact files — `docs(friction): this round's journal, and the row-6 ruling carried into a fix [inc1-T9]`

**Battery:** `./harness/friction.sh --show` non-empty · `ci/check_closeout.sh --selftest` · `../harness/check.sh`
· `./harness/review.sh --pr-body` output pasted **unedited** into the PR body (that round trip is tested).

---

## Verification (the increment gate)

1. **Full battery green:** `make reset && make test` · `cd core/api && go test -count=1 ./...` ×2 stable ·
   `make schema-check` · `make schema-contract` · `ci/check_citations.sh --selftest` (17 probes) ·
   `../harness/check.sh`.
2. **The artifact cannot drift:** `make world-model-schema && git diff --exit-code
   core/api/schema/world_model.v6.schema.json` is clean, and every object node in it carries
   `additionalProperties:false` (asserted by count, not by spot check).
3. **Malformed input is refused, not dropped:** for every leaf in `claimedSlice()`, the absent / null /
   wrong-type / empty cases each produce a refusal with a named cause. This is the SPEC-035 class and it is
   the one thing a green suite has hidden before (`ci/mutate.sh:30-34`).
4. **The type boundary holds at compile time:** `validateDoc` has no `resolver` in scope; `Refuse` has no
   document in scope; the compile-negative is recorded.
5. **Coverage is staged, and says so:** an unclaimed leaf inside the union refuses the boot naming the
   leaf; a leaf outside it does not, and appears in the remainder line printed on a real start — captured
   in the PR body.
6. **Standing gate 1:** all 13 refusals plus both arrival floors plus the name guard are executable and run
   over every document on disk (`2026-08-26-world-model-eight-increments.md:55-56`).
7. **Mutation, both ends:** Task 1's four verdicts recorded verbatim; Task 7's four equivalents all
   **CAUGHT**, `ci/mutate.sh` exit 0.
8. **AC-7 observed, not inferred:** a player perceives one authored event they did not cause inside the
   five-beat window, in every world.
9. **The round closes with a journal:** `Friction:` carries a verdict, `Learned:` names a file in the diff,
   `Areas:` and `Reviewed-by:` are `./harness/review.sh --pr-body`'s output unedited.

## Self-review (done during writing)

- **Amendment coverage:** §2 → T8; §3.1 → T2 + T6; §3.2 → T6; §3.3 → T6; §3.4 first bullet → T5, second →
  T3; §4 → T7; §6 → the fence's citation rule; §7's four experiments → T1 + T7, its friction obligation →
  T9, its ledger ruling → T9. §5 and §7's error-legibility question are **explicit non-goals** with
  citations, not omissions.
- **The one reordering, and why.** The brief lists the coverage index before the landing framework. Built
  in that order the index has no left operand: its input is `⋃ Consumes` over `Declaration`, which the
  framework defines (`prd_world_creation_depth.md:88-96`), so the index cannot compile first. Framework
  (T5) → index (T6). Similarly the mutation experiments move to **first**, not last: all four probe files
  the cutover deletes, so run late they cannot be run at all (`FINDINGS_playloop.md:332`).
- **Blast-radius honesty:** T8 is the widest — it deletes `worldgenesis.go` and `worldgenesiscommit.go`
  (67 named refusals among them) and rewires the handler. T3 is second: adding a published schema touches
  `ci/schema_contract.py` and every schema-set assertion. Both name their fallout in the task.
- **Placeholder scan:** no TBDs. Three genuine unknowns are named as blocks or skips rather than guessed —
  `SCHEMA-v7.md`'s key names (T7 BLOCKED), the `arrivals[]` `within`/`place` divergence (T3 Step 5,
  `t.Skip` with the owner named), and v7's `maxItems` ceiling values (T7 Step 2).
- **Counts, checked rather than copied**, because this round's two corrected citations were both numbers
  whose ids resolved: 16 frozen sections = 10 claimed + 6 unclaimed; 11 facets = 5 claimed + 6 unclaimed;
  13 refusals = R1–R12 (`SCHEMA-v3.md:70-84`) + R13 (`SCHEMA-v4.md:118`); 16 validator rules = 13 + 2
  arrival floors + 1 name guard; 12 acceptance criteria, of which §6 holds four — the "eleven" is flagged
  as decision 8 rather than propagated; four class→number conversions, two failing open
  (`01_engine_capability_audit.md:71-82`); 67 `return refuse(` sites in `validate()`
  (`worldgenesis.go:249-495`).
