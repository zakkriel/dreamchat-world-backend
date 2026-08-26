# Amendment to `prd_world_creation_depth.md` — retarget the landing contract at `world_model/6`

**Status:** decided 2026-08-26, after a three-seat adversarial round. **Supersedes `PROPOSAL.md`**, which
is kept as the attack target and is wrong in the ways `FINDINGS_*.md` record.

**Amends:** `docs/10_prds/prd_world_creation_depth.md` (draft v3). Extends, never re-litigates. That
PRD's mechanism — `Landing{Declare,Parse,Apply,Refuse}`, readers as a sum type, coverage checked at
registration, the runner owning ids/ticks/class-resolution/provenance, grounding as a sum type,
`shares(key)`, explicit phases — **survived all three seats and is unchanged.**

**Round outcome:** 23 blocking findings across three seats. The mechanism survived; the proposal did not.
Everything below is either a decision the product owner made or a block answered.

---

## 1. What the round killed, and what replaced it

| Proposal said | Round said | Decision |
|---|---|---|
| First customers become `integrity`, `latency_class`, `reliability_class`, `excluded[]` | **F1 `block`** — the five-beat window is **≤150 s of world time** (`tension.go:28-45`, `prd_world_creation.md:22`); degradation, week-scale latency and ten-reading reliability cannot show inside it. **F3 `block`** — they were chosen *because* absent from the engine, which is the reader-with-no-consumer class the coverage index exists to catch. **F2 `block`** — retiring §5 orphans §6's four written ACs. Genesis seat: the substitution "does not prove what §5 proved" — §5's three were chosen to isolate the mechanism *from* grounding difficulty | **§5 stands.** Its three concepts are the first customers, re-expressed in v6 |
| Retarget the coverage check at the whole of `world_model/6` | **F6 `block`** — the check is whole-schema all-or-nothing, which is its entire value. At v6 it cannot go green until every leaf has a landing with a real reader, so the platform blocks on the features that declare into it. **A cycle.** And the five wave-1 increments are coupled through a global set difference regardless of which files they touch | **Staged coverage** — §3 |
| Old commit path removed, clean cutover | **C11–C17 `block`** — seven capabilities strand | **Derive what can be derived, obligate the rest** — §4 |
| "17 top-level sections", "25 obligations" | **C8 `block`** — `SCHEMA-v3.md:16` says **16 (frozen)**; the audit covers **24** | Corrected — §6 |
| R1 "survives untouched" | **C7 `block`** — one recursive `entities[]` turns a disjoint union into an overlapping one; the claim is false | Corrected — §3.3 |
| Give the narrator `world.brief` | **F4 `gate`** — the narrator already receives world prose every beat; and handing it the brief while the document is short of it is the exact founder-gate bug `NEVER CONTRADICT OR EXTEND THE STATE` exists to stop (`narrateprompt.go:14-17,28`) | **The document's own content, never the brief** — §5 |

---

## 2. First customers — §5, re-expressed in v6

`prd_world_creation_depth.md` §5 and §6 stand as written. Only the shape changes, because v6 already
carries these ideas:

| §5 concept | v6 expression | §6 acceptance criterion, unchanged |
|---|---|---|
| `collectives[]` | the `collective` facet + `offices[]` | AC-5, AC-8 |
| `norms[]` | `law[]`, with `enforced_by` and `within` scoping | AC-6, AC-9, AC-10 |
| `near_future[]` | one authored imminent change, expressed through `processes[]`/`cycles[]` with `when` as a class the runner resolves | AC-7 — *the player perceives one authored event they did not cause, inside five beats, in every world* |

**Why this is the right proof and the proposal's four were not:** these three have no cross-concept
grounding problem, so they test whether the contract holds without simultaneously testing whether the
resolver can reach across concepts. And AC-7 is the only one of the eleven that puts something in front
of a player inside the window the play-loop seat measured.

**Constraint retained from §5:** all three are optional. A brief implying none authors none, at zero added
token cost, and the pipeline is unchanged.

---

## 3. Staged coverage — the decision, and the rule that keeps it honest

**Decided:** the startup coverage check guards **the slice of the format built so far**, and widens as
each increment lands.

**Accepted cost, stated plainly:** a field outside the current slice can sit unread and the check will
not say so. That is the exact defect the check exists to catch, deferred rather than solved. It is
accepted to break F6's cycle, and it is bounded by §3.1.

### 3.1 The staging key — so the boundary is never ad hoc

A section enters the checked slice **when an increment claims it, and never before.** The claim is
written, not implied:

- Every increment's plan document opens with a **`Claims:` list** naming the v6 sections and keys it
  brings into the slice.
- Registration computes coverage **over the union of all `Claims:` lists to date** and refuses to boot on
  an unclaimed leaf *within that union*.
- Registration **prints the unclaimed remainder** at every boot. The deferred risk is visible on every
  start rather than discovered later. This answers **C6** — an exclusion nobody can see is the
  schema-validator defect again (`beat_frame.v3` carried `"format":"uuid"` 35 times and none was ever
  checked, failure-log #20).
- A section may not be silently un-claimed. Removing a section from the slice needs a dated line in the
  amendment, same discipline as retiring a schema version.

**Increment 1 claims exactly:** `world`, `excluded[]`, `vocabulary`, `law[]`, `entities[]` with the
`extent`/`matter`/`agency`/`passage`/`collective` facets, `offices[]`, `history[]`, `arrivals[]`, and
the `processes[]`/`cycles[]` subset AC-7 needs. Everything else is printed as unclaimed remainder.

### 3.2 The coverage check becomes bidirectional

Unchanged from the proposal and unattacked: the second direction — every engine input must have an
author — was named in `r3_extraction_by_gamedesign.md:19-22` as the direction **nobody ever computed**.
The audit found it live: `perception_record.confidence` permanently 1.0, `distortion_level` written and
read by nothing, `invalid_tick`/`expired_at` read on every knowledge path and written only by test
fixtures, three `epistemic_type` values produced by no code, `world_pressure(accrued, threshold)` touched
by nothing.

Both directions run over the **claimed slice** only.

### 3.3 What the recursive `entities[]` costs, stated rather than denied

**C7 stands.** `world_genesis/1` partitions nouns into disjoint typed arrays, so concept and section
coincide and coverage is a disjoint union. v6 has **one recursive `entities[]`** whose shape is decided
by its `facets`, so two landings can claim keys on the same section and the union stops being disjoint.

The check therefore weakens from *"every leaf is parsed by exactly one landing"* to *"every leaf is
claimed by at least one landing."* **That is a real loss and the amendment does not pretend otherwise.**
Mitigation, and it is narrow: a landing declares the **facet** it lands, not the section, and registration
refuses two landings claiming the same `(section, facet, key)` triple. Overlap becomes a named
registration failure instead of a silent double-write.

### 3.4 Two additions the round forced

- **The document validator takes the document and no resolver; `Refuse` takes the resolver and no
  document** (**C20 `gate`**, adopted verbatim as a type boundary). This is what stops the resolver
  becoming the god object §9 Q1 warned about — a bound in the type system, not a sentence in a PRD.
- **Malformed input is refused, not dropped** (**C22 `block`**, **F0 `gate`**). The machine artifact
  carries `additionalProperties:false` at every level and the Go decoder uses
  `DisallowUnknownFields`. A wrong-type, absent, null and empty fixture per claimed leaf must be
  *refused with a named cause*. Without this the SPEC-035 silent-drop class returns — `witnesses:
  "<uuid>"` as a bare string committed with zero witnesses and no halt reason, found one commit late.

---

## 4. The stranded capabilities — derive what can be derived, obligate the rest

**Decided.** Per finding, with the verification that refined two of them:

| Capability | Round finding | Decision |
|---|---|---|
| `world.tagline` → world cover art | **C13 `block`** — no v6 source. Cover art is structurally gated on it: *"no tagline, no cover, because there is nothing to render from"* (`worldgenesiscommit.go:662-663`, `artcommission.go:66`) | **Derive** from `world.premise`. Flagged honestly: a derived tagline is a line the founder never approved, and the gate was deliberately structural. Reviewed at increment 8's surface, where invented content is already correctable |
| `world.ornament` | **C13 `block`** | **Derive** from `world.mood` |
| per-place `tension` | **C11 `block`** — required today (`world_genesis.v1.schema.json:53`), and its absence gives a `none` budget = **infinite**, so nothing ever becomes a journey. Verified independently: **6 of 8** extent entities in the Grelda document carry it; the root and the granary do not. Silent SPEC-030 regression, invisible to the suite | **Obligate** — `tension` required on every `extent` entity. New author obligation, v7 delta. **No silent default:** absence is a refusal, never a fallback |
| single root / coordinate origin / the 0.6 ring | **C12 `block`** | **Weaker than reported, and verified:** v6 *does* have a single root (`Grelda`) and R8 makes `within` a tree. But nothing **obligates** exactly one root. → **Obligate** exactly one root `extent`. The ring factor stays a named world-feel constant under AC-3 |
| `arrival_candidates`, `arrival.why`, `newCast` | **C15 `block`** | **Weaker than reported, and verified:** `arrivals[]` is an array, so three candidates are *expressible*; nothing obligates three, nor marks one as chosen. → **Obligate** three candidates with exactly one chosen, and add `why`. Author obligation, not a new section |
| `identifierShapedName` — the Ironmoor guard | **C16 `block`** — refuses a machine-shaped person name at three sites; its comment records the live incident | **Obligate** — carried into v6's refusals unchanged. A guard with a logged incident behind it is not dropped in a rewrite |
| the two arrival floor refusals | **C17 `block`** — *"nothing leads out of the arrival place"* becomes sufficiency prose with no checker, so a world can be authored the player cannot leave | **Obligate** — both become executable refusals in the validator, not S1–S6 prose |
| array ceilings / tick ladder bound | **C18 `gate`** | **Obligate** — every array in the machine artifact carries `maxItems`. Restores the tick-ladder assertion's bound and caps per-build token cost |

**All of §4 is a v7 delta**, not an edit to v6. Author obligations only: no new facet, no new section, and
the facet list stays frozen at eleven.

---

## 5. The narrator gets the document, never the brief

**F4 `gate`** and **F5 `gate`**, adopted.

- The claim that this is "the largest single playability lever" is **withdrawn** — unproven, and the
  narrator already receives world prose every beat. What reaches it must be **the committed document's
  own content and minted vocabulary**, never the raw brief, and a test asserts the brief string does not
  appear in the assembled narration prompt.
- The world block must be present in **all three** prompt builders — structured, repair, and plain
  fallback. The fallback slices at the segment-contract marker (`narrateprompt.go:71,93-98,143-151`) and
  would drop a block appended at the end, on the one path taken *after* two structured attempts already
  failed, which is when invention risk is highest.
- **C19 `accept`**: this work is independent of the retarget and does not wait on increment 1.
- **F4's ranking stands:** the brief-to-document coverage check ranks **above** this, because every
  per-beat thing the narrator reads is document-derived, and a document missing two thirds of its brief
  starves every beat of every world.

---

## 6. Citation corrections

- **Top-level sections: 16, frozen** (`SCHEMA-v3.md:16`), not 17. v3 D2 deleted `layers[]`.
- **Reader obligations: the audit covers 24** — `SCHEMA-v3.md` §4's 23 rows with the `conceals` row split
  by v5. `SCHEMA-v6.md` adds the containment-tree derivation, making **25**, and **the audit has not been
  re-run against that added row.** Stated rather than papered over.
- **Class→number conversions in the engine: five, not three**, and two carry a silent numeric default
  (**C21 `block`**). The audit's §"most load-bearing absence" undercounts and must be corrected.
- Both original errors are the class `ci/check_citations.sh` exists for, with the limit the dossier
  records: the gate asserts a cited **id** resolves, never that a cited **number** does.

---

## 7. Still open, and not decided here

- **Error legibility under centralised class resolution** (`prd_world_creation_depth.md` §9 Q2,
  sharpened by **C21**). Seven refusals replaced, not ~40; 67 deleted. Needs its own answer in increment
  1's design doc.
- **This round has no friction journal** (**C23 `block`**). Owed before the implementing round closes.
- **`friction-log.md` row 6 ruled `WASTE`, upheld and still live** (**C24**) at `harness/check.sh:406`
  and `:586`. The fix is one check and has already cost two agents.
- **Whether a container declaring both `extent` and `matter`** aggregates its contents into its own mass.
  `SCHEMA-v6.md` settles placement; this is increment 2's first design question.
- **The four mutation experiments the contracts seat predicted rather than ran.** A prediction is not a
  result. The implementing round runs them; if any is CAUGHT, the matching finding weakens.
