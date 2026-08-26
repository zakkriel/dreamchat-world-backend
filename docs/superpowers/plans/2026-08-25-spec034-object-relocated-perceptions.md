# Plan — SPEC-034: an `ObjectRelocated` branch in `generate_perceptions`

**Status:** BLOCKED on one product ruling (§2). Everything else is specified.
**Round shape:** single-repo (backend only). SPEC-034 says so outright: *"Cross-repo: BE only; the
frontend needs no change and should not design around the 404 as if a lens will remove it."*
**Areas (computed by `../harness/review.sh`, not chosen):** `perception` · `canon-recording` ·
`space-and-journey` · `contracts-and-platform`
**Reviewers:** the four matching experts, plus `harness/roles/process-observer.md` — this is the
harness's first real round and the process is under review as much as the change.

**Rules relied on:** **B-2** (valid-path knowledge — observation is a valid path), **B-1** / **I-3**
(perception-bound surfaces; hidden truth is absent, not un-rendered), **I-2** (universal provenance),
**B-9** (synthesis derives deterministically), **D-4** (schema_version + validation), **D-11**
(record/derive), **B-6** (contradiction lives in perception; never delete), **ADR-006** (three time
axes; invalidation never deletion), SPEC-016 (per-attribute perceivability — the deferred owner of
visible-vs-hidden), SPEC-034.

---

## 1. What is wrong, measured

`generate_perceptions` has branches for `private_disclosure`/`Communicated` and `move`/`ActorMoved`.
It has **no `ObjectRelocated` branch**, so a handover records no perception for anyone — including the
new holder.

```
fn_carrying(DL, Kade)                          → lists the Sealed Note
fn_entity_visible(DL, Kade, note)              → false
fn_artifact_page(DL, Kade, note)               → NULL   ⇒ 404
fn_compendium_index_json(DL, Kade,'artifact')  → entries: []
```

Kade holds **zero** perceptions about a thing in his hands. The 404 is a faithful read of the
perception store and a false statement about the world.

**Latent only because nothing has reached it.** The play world's `canon_event` holds 102 `move`, 4
`Communicated`, 2 `ActorMoved` and **zero `ObjectRelocated`** — the seeded carry edges were authored as
state, never as events. The first in-play handover makes both objects permanently invisible to every
viewer.

## 2. THE BLOCKING RULING — who perceives a handover?

The new holder: certainly, `direct`. The old holder: presumably, `direct`. **Everyone co-present is
the decision**, and it cannot be inferred from the docs because they point both ways.

### What the investigation found (measured, per SPEC-032's lesson)

| Finding | Evidence |
|---|---|
| **No concealment signal exists.** No `concealed`/`hidden` field on `ObjectRelocated`; the event's two founder-locked dimensions are **volume** (blocks) and **weight** (consequences), nothing epistemic. The 18 `secret` hits in `schema.sql` are Mara's *perception* secret, not a mechanism. | `20260729100003_object_relocated.sql` header; `schema.sql:2030` |
| **The "tucked under her cloak" beat is the spec's illustration, not an authored event.** It appears nowhere but inside SPEC-034. | grep of `seeds/` and `docs/` |
| **A witnessed handover is the product's flagship perception example.** *"You saw Seren pass a sealed note to a cloaked figure"* — Actors PRD twice, Locations PRD, `parked_relationships`. | `prd_compendium_actors.md:438,543` |
| **Co-presence is already computable, with exact precedent.** `fn_actors_at` is how `hearing_teaches_a_name` and `spoken_words` decide who overhears. A handover is the same shape of question. | those two migrations |
| **Replay does NOT regenerate perceptions.** `replay_0A()` asserts *"replay reproduces domain-equivalent **projection** state (ADR-026)"*. | `90_replay_test.sql:5` |

### The options, and why the last finding decides it

| | Rule | Cost | Reversibility |
|---|---|---|---|
| **A** | holders only (new + old), `direct` | cheapest, no new mechanism | **Bad.** Widening later cannot be done by replay — perceptions are not regenerated. Every handover already committed stays unperceived by observers forever, unless someone writes a bespoke backfill that synthesises perceptions from `canon_event`. That is data repair, not a rule change. |
| **B** | holders + everyone co-present, all `direct` | uses `fn_actors_at`, exact precedent already shipped twice | **Good.** Narrowing later only affects future events; the past keeps what it recorded, which is exactly **B-6** — contradiction lives in perception and nothing is ever deleted. |
| **C** | B, minus a concealment exception | delivers the cloak beat | **Forbidden here.** Requires inventing the concealment signal, and SPEC-034 says explicitly *"No mechanism is proposed here (anti-drift)"*. **SPEC-016** — outwardly-visible vs hidden — is the deferred mechanism that properly owns it. |

**Recommendation: B, with the concealment question filed against SPEC-016.** It matches shipped
precedent, it delivers the PRD's own flagship example, it invents no mechanism, and it is the only
option whose reversal is clean. A is the *unsafe* choice despite looking conservative, because
under-recording is the direction replay cannot repair.

**Falsification criterion (Validation Ladder):** after the fix, a handover in the play world must make
the object appear on the Artifact page for the new holder, the old holder, **and** a third actor who
was in the room — and must **not** appear for an actor who was elsewhere. If the third actor sees
nothing, B was not implemented. If the absent actor sees it, co-presence is not being read.

> **Nothing below is written until this ruling lands.** Choosing for you would be inventing the
> mechanism the spec forbids inventing.

## 3. The change, once ruled

| Step | Files |
|---|---|
| 1. The branch | a new migration adding the `ObjectRelocated` case to `generate_perceptions`: `direct` for new holder, old holder, and (per ruling) co-present actors via `fn_actors_at`. Provenance = the relocation event (**I-2**). |
| 2. Its area declaration | a row in `core/db/migrations/AREAS.tsv` naming **`perception`** — `harness/check.sh areas` refuses without it |
| 3. Map the new test | a glob in `docs/30_architecture/AREAS.map` for the new suite — **the launcher already caught that it is unowned**, before any code was written |
| 4. The test | a new pgTAP suite asserting the falsification criterion above, plus the negative case (an actor elsewhere perceives nothing) |
| 5. Regression | `113_object_relocated_test.sql`, `42_visible_perceptions_test.sql`, `93_/94_generate_perceptions_*` must still pass |
| 6. Committed artifacts | `core/db/schema.sql` + `core/api/migrations.txt`, or `make schema-check` fails |
| 7. Close the SPEC | `docs/open-spec-items.md` — SPEC-034's body still reads OPEN, and a landed SPEC that still describes the old behaviour was graded "actively misleading" once already |

## 4. Non-goals

- **No lens work.** SPEC-034 is explicit: the gate is `fn_entity_visible`, not a lens.
- **No `*_state` reads to manufacture an existence claim.** That is the wall (**B-1**, **I-3**) and it is
  what SPEC-029 forbids in writing.
- **No concealment mechanism.** SPEC-016's, when it fires.
- **No frontend change.** And the frontend must not design around the 404.
- **No backfill of the seeded carry edges.** They were authored as state; whether to convert them to
  events is a separate question and inventing it here is scope.

## 5. The gate

```bash
make reset && make test            # the new suite + 42/93/94/113 + the wall suites
make fingerprint > /tmp/a && make reset && make fingerprint > /tmp/b && diff /tmp/a /tmp/b
make schema-check
make schema-contract
../harness/check.sh areas          # the AREAS.tsv row and the new test's glob
../harness/check.sh               # the whole cross-repo set
```

**Mutation, before claiming any of it works:** delete the new `ObjectRelocated` branch and confirm the
new suite goes red. A guard nobody has watched fail is not a guard — 40 of 70 probes survived a fully
green run in this codebase.

## 6. Close-out

`Learned:` must name where the lesson landed. Candidates known in advance: `perception`'s dossier §3
(SPEC-034 moves from open to closed), and — if the round finds anything about the harness itself —
`areas-stress-test.md`.
