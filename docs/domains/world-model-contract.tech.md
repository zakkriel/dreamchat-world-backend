# world-model-contract · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-11 · The world-model contract and depth ·
**Parent bounded context:** World Engine

This file holds how the domain is built — where the corpus lives, what is live in code, validation,
traps. `world-model-contract.product.md` holds what it means; `world-model-contract.seams.md` holds
what crosses its boundary.

Line numbers are as of 2026-08-27; re-locate by grep before relying on one.

---

## Where the corpus lives — and it is git history

The contract's documents have **no home in the live tree.** The consolidation (`workspace:ADR-W006`,
commit `88486c1`) deleted `docs/30_architecture/world_model/` (the design record, the engine
capability audit, the aliveness synthesis), the clean-sheet debate directory holding `SCHEMA-v2.md` …
`SCHEMA-v7.md` and all sixteen trials, the depth PRD (`docs/10_prds/prd_world_creation_depth.md`),
and both increment plans — with no successor file. Verified: `git ls-files | grep -i world_model`
returns nothing; `git show 88486c1 --name-status` lists the deletions. Recover any of it as bannered
evidence from the parent commit, e.g.
`git show 5697b7e:docs/30_architecture/world_model/01_engine_capability_audit.md`. `ADR-W006`'s
lookup rule — cited by something live, or it stays out — now has this package as a live citer;
whether that obligates restoration (the `ADR-P026` precedent) is Open question 1.

## The version chain

v1 died on premises (kinds-as-arrays) → v2 facets, died on ambiguity → v3 the contract (obligations,
refusals, reader obligations; facets frozen), died on four fields stating a number twice → v4
generative (genesis *invents*; provenance; sufficiency), its reader half died on `conceals` →
v5 reader-half correction → v6 `within` ungated, containment tree derived by the engine → v7 seven
capabilities `world_genesis/1` already enforced, reclaimed — *"found by reading the engine the
contract is meant to replace, not by design review."* Counts after v7: **14 author obligations, 17
refusals, 27 reader obligations, 11 facets frozen** (`SCHEMA-v7.md`, git-history). v7's counts line
says sixteen top-level sections; the design record's own §4.2 table holds fifteen rows — unreconciled
in the corpus, recorded here, not resolved.

The effective statement is the union **v3∪v4∪v5∪v6∪v7**; reading v3 alone hands you a known-fatal
`conceals` row and a `within` gate every real document violates.

## What is live in the tree today

The shipped contract is **`world_genesis/1`**, not `world_model` — the author's half only:

- `core/api/schema/world_genesis.v1.schema.json` — "No uuid, no coordinate, no tick … no number of
  any kind appears anywhere" (its own `$id` title); `tension` required on every place (`:53`); every
  array bounded (`places` minItems 2 / maxItems 8, `:48-49`).
- `core/api/worldgenesis.go` — 67 `return refuse(` sites (counted); the Ironmoor guard
  `identifierShapedName` (`:541-565`); the two arrival floors ("nothing leads out of …", `:381`;
  "nobody is in … when the player walks in", `:394`).
- `core/api/worldstatement.go` — the declared landing seam: its comment names the fields
  `world_model/6` supersedes (`premise`, `vocabulary`, `law[]` + `excluded[]` "statable here for the
  first time", `:30-36`).
- The class→number surface, and it **fails open**: `fn_extent_class_metres` ends `ELSE 50`
  (`core/db/schema.sql:1784`), `fn_duration_class_seconds` ends `ELSE 2` (`:1654`) — a hallucinated
  class word becomes 50 m or 2 s and play continues on a wrong number. `tension` alone is
  enum-enforced (`tension.go:28-43`).
- **`excluded[]` has no live representation at all**: zero matches in the genesis schema, zero in
  `core/api/*.go`, and `schema.sql`'s one match is an unrelated comment (`:3657`). Negative canon is
  law without an enforcer.

**Increment 1 has not landed.** No landing framework, no `Claims:` manifest, no document validator
exist in code; `worldgenesis.go`/`worldgenesiscommit.go` — the files the increment deletes at
cutover — are still present. *"Today no document in this project has ever been validated"* (the
increments plan) still holds.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `workspace:ADR-W006` | The corpus is out of the tree until something live cites it; recovery is bannered evidence from git history. | Citing a deleted path as if it resolved. |
| `ADR-P026` | The understanding pass (WE-10) was restored by founder ruling; restoration is a ruling, not a judgement call. | Restoring this domain's files yourself. |
| `SPEC-030` | The lesson behind v7's O12: an authorable world where nothing can become a journey is a regression class, not a tuning choice. Missing `tension` COALESCEs to `none` → `math.MaxInt64` budget (`tension.go:38-43`) — silent at every layer. | Making `tension` optional again, or adding a silent default. |
| `D-6` | One home per fact. This file is the one home for "the corpus is history-only"; product.md and seams.md point here. | A second copy that goes stale. |

### What you may not decide alone

1. **Facet or section changes.** Frozen; a twelfth facet only by deleting one (product.md §8).
2. **Reopening the nine closed decisions** of the increments plan (rebuild-not-bridged; grammar/
   vocabulary split non-negotiable per-scale; ladders never averaged; drift-prevention-is-cause;
   exclusions permanent; moving locations real; secrets toggle server-side; withheld interiors
   never on a confirmation surface; the epistemic wall not disturbed). Founder-ruled; recover the
   plan from `5697b7e` before touching.
3. **Restoring any deleted corpus file** — founder ruling (`ADR-P026` precedent).
4. **The landing target** (`world_model/6` vs v7) — Open question 2.

## Validation for this domain

- Live-code half: `cd core/api && go test -run 'WorldGenesis|WorldStatement' -count=1 .` — always
  `-count=1` (the suite is seed-dependent). Never `make reset`.
- The contract half has **no executable validation** — no validator, no fixture ever validated. So
  **evidence here means a version-attached citation checked against git history**, nothing weaker.

**What counts as ceremony here:** asserting that a table or column exists. The audit's rubric is the
guard: WORKING means *"code computes it and the value changes what happens in play"* — this codebase
shipped a table instead of a mechanism five times, and a sixth is not progress.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The class resolvers fail open, under a comment reading "never fails closed."** | §What is live above (the one home). The genesis seat is forbidden any number, so the ladder is the *only* supply route — and most rungs do not exist. |
| **Counts rot, then propagate.** "21 reader obligations" (wrong, 23), "fourteen `within` violations" (wrong, 9), and two scoring figures were each corrected only in a *later* file. | S07b Topic 21's table (`digest/`); cite counts with a version attached. |
| **A smaller, later contract is not a superset of what it replaces.** Seven enforced behaviours were silently dropped across the rewrite, caught only by reading the shipped engine. | `SCHEMA-v7.md` (git-history), its closing paragraph. |
| **An empty `excluded[]` flips world rules.** | product.md §language (the one home). |
| **Trials are reasoning, never state.** The schema versions are candidates; the testworlds are fixtures written to break the schema. | S07c's own banner: *"a trial's conclusion is evidence inside an argument, not a decision."* |
| **The filename says eight increments; the document's H1 says nine and lists nine.** | The plan file (git-history); recorded as a corpus contradiction, founder's to rule. |

## Open questions

1. **Where does the contract live now?** History-only, cited by this package. Restore (the
   `ADR-P026` pattern) or stays out — a founder ruling either way.
2. **`world_model/6` or v7 as the landing target?** `worldstatement.go:31` names `/6`; the chain
   ends at v7 (the stranded-capabilities delta). Both recorded; not resolved here.
3. **Composition.** Six keys decide whether a receiver receives anything and carry no reader
   obligation — the reader round's largest hole; a `receipt` obligation over (emitter, receiver) was
   proposed and never ruled on.
4. **Is the player an entity?** Four trials chose three different answers (placeholder entity /
   deleted references / bare `"you"`); no version says which. An agent hitting this is deciding
   something new.
5. **Sixteen or fifteen top-level sections** (§The version chain above).
