# Proposal under attack — retarget the landing contract at `world_model/6`

> **SUPERSEDED 2026-08-26 by `AMENDMENT.md`. Kept because it is the attack target.**
>
> Three seats returned **23 blocking findings** against this document (`FINDINGS_contracts.md`,
> `FINDINGS_genesis.md`, `FINDINGS_playloop.md`). Its mechanism claim survived; its three substantive
> proposals did not. Specifically wrong here, and corrected in the amendment: retiring §5 (all three
> seats blocked it — the replacement first customers have no player observable inside the five-beat
> window, and were chosen *because* absent from the engine, which is the very defect class §3.1's index
> exists to catch); retargeting the coverage check at the whole contract (it is whole-schema
> all-or-nothing, so this is a dependency cycle, and it couples the five increments this round called
> independent); "17 top-level sections" (16, frozen) and "25 obligations" against a 24-row audit; the
> claim that R1 survives untouched (one recursive `entities[]` makes the union non-disjoint); and giving
> the narrator the raw brief (it already receives world prose, and the brief is the wrong payload).
>
> Read the amendment for what was decided. Read this only for what was attacked.

**Status:** draft for adversarial round, 2026-08-26. Superseded. Written to be attacked.

**What this is:** an **amendment** to `docs/10_prds/prd_world_creation_depth.md` (draft v3), not a new
PRD. That PRD is the landing contract — `Landing{Declare,Parse,Apply,Refuse}`, R1 readers as a sum type,
R2 leaf coverage at registration, R3 the runner owning ids/ticks/class-resolution/provenance, R4 grounding
as a sum type, R5 `shares(key)`, R6 explicit phases. **Its mechanism is not under attack. Its target is.**

**The one change proposed:** the landing contract's target document becomes **`world_model/6`**
(`SCHEMA-v3.md` + v4 + v5 + v6) instead of `world_genesis/N`.

---

## 1. Why the target must change

`prd_world_creation_depth.md` targets `world_genesis/N` — the schema the live pipeline speaks. Meanwhile a
separate line of work took the world model to v6, and **v6 is a different and much larger contract**: 17
top-level sections, 11 frozen facets, 25 reader obligations, provenance on every element, and `excluded[]`.

The engine has been audited against it: `docs/30_architecture/world_model/01_engine_capability_audit.md`
— **3 of 25 obligations working, 12 partial, 9 absent**, every row cited to file:line. The live pipeline
cannot emit a v6 document and the engine cannot consume one. There is no machine representation of v6 at
all: no schema file, no Go type.

So the landing contract as specified would be built against a schema that the product is abandoning.
**Building it at `world_genesis/N` means building it twice.**

## 2. What survives untouched

Claimed, and the round should test each claim:

- **The `Landing` interface.** Declare/Parse/Apply/Refuse is target-agnostic.
- **R1 (reader sum type).** `state | perception | referenced` is about destinations, not sources.
- **R3 (the runner owns ids, ticks, class→number, provenance).** v6 needs this *more*: 13 of its 25
  obligations dead-end in a class word the engine cannot resolve, and the whole engine has three
  hand-built conversions. R3 is where the generic resolver belongs.
- **R4 (grounding sum type), R5 (`shares`), R6 (phases).** Unaffected by target.
- **§8 Non-Goals.** All still hold. Nothing here re-proposes a social-structure identifier, Tier-1 growth,
  `relationship_state` writes, group-held perceptions, engine-side norm enforcement, or new DDL.
- **§11 Final Product Rule.** Unchanged and still the bar.

## 3. What the amendment changes

### 3.1 R2 becomes bidirectional

R2 today: registration computes `⋃ Consumes` across declarations and diffs it against every non-numeric
leaf of the target schema; an unclaimed leaf is a named registration failure.

**Add the second direction:** every engine input must have an author. `r3_extraction_by_gamedesign.md:19-22`
named this as the direction **nobody ever computed**, and it is how the second defect class grew silently.
The audit found it live — `perception_record.confidence` permanently 1.0, `distortion_level` written by
nothing and read by nothing, `invalid_tick`/`expired_at` read on every knowledge path and written only by
pgTAP fixtures, three `epistemic_type` values produced by no code, and `world_pressure(accrued, threshold)`
touched by nothing at all.

One index, two directions, three defect classes: authored-with-no-reader, engine-input-with-no-author,
reader-with-no-consumer.

### 3.2 §5's three concepts are subsumed, and the first customer changes

`prd_world_creation_depth.md` §5 lands `collectives[]`, `norms[]` and `near_future[]` as the contract's
first customers. **v6 already has all three, differently shaped:**

| depth PRD §5 | v6 |
|---|---|
| `collectives[]` | the `collective` facet, plus `offices[]` |
| `norms[]` | `law[]`, with `enforced_by` and `within` scoping |
| `near_future[]` | `processes[]`, `cycles[]`, `accumulators[]` — a richer and more demanding shape |

**Proposed:** §5 is retired as written, and the contract's first customer becomes the v6 sections that are
**already used by every test world and absent from the engine.** From the audit's world-usage scan, all
three v4 documents author these and the engine has nothing:

- `integrity` — degradation toward a terminus
- `latency_class` — information that arrives late
- `reliability_class` — signs that misreport
- `excluded[]` — the per-world never-allow list, written by all three worlds and read by no code

Those four are one declaration each if the contract holds, which is exactly the test §11 demands.

### 3.3 Migration §7 is replaced by a clean cutover

§7 stages migration of eight `world_genesis/N` concepts. The product decision is a **clean rebuild** — the
old format and its bespoke commit path are removed rather than bridged. §7's step 4 ("the hard cases get a
decision, not indefinite coexistence") survives in spirit and its list changes: the hard cases are now
whichever v6 sections resist the `Landing` shape.

### 3.4 Two additions the audit forces

- **The document validator.** v6 has 13 refusal rules and **no document in this project has ever been
  validated.** R2 is a registration-time check over schema × declarations; this is a per-document check.
  Different gate, different time, both needed.
- **The world reaches the narrator.** `world.brief` carries a `COMMENT` at `schema.sql:4234` stating it is
  never rendered and no projection selects it; `world.theme` is frontend chrome. The narrator has never
  been told what world it is in. Cheap, and the largest single playability lever in the increment.

## 4. What this proposal deliberately does not decide

- **Whether a container declaring both `extent` and `matter`** — a living house, a train — aggregates its
  contents into its own mass. `SCHEMA-v6.md` settles placement (`extent` wins) and files this.
- **Error legibility under centralised class resolution.** `prd_world_creation_depth.md` §9 Q2 is
  unanswered and this amendment makes it more acute, not less.
- **Whether `Refuse`'s resolver becomes the new god object** (§9 Q1). Unanswered, and v6's 17 sections
  raise the pressure on that seam.

## 5. Questions for the round

1. **Does retargeting break any static guarantee of the landing contract?** R2's leaf coverage is a set
   difference over the target schema. v6 has 17 sections, nested facets, and keys gated by facet
   declaration (D4). Is leaf coverage still statically computable, or does facet gating make "every leaf"
   a runtime question?
2. **Is §5's retirement right, or does it lose the cheap proof?** §5's three concepts were chosen because
   none has a cross-concept grounding problem. Do the four proposed replacements have one?
3. **Does the clean cutover strand anything?** The old commit path is what writes a world into the engine
   today. Name what stops working on the day it is removed and has no v6 equivalent.
4. **Does any of this reach the table?** §6's ACs are the only non-structural victory conditions. State
   the 5-beat observable for each of the four proposed first customers, or say it has none.
5. **Is the parallel plan sound?** `docs/superpowers/plans/2026-08-26-world-model-eight-increments.md`
   claims five increments can run concurrently after this one. Attack the dependency graph.

---

## Reading list

| File | Why |
|---|---|
| `docs/10_prds/prd_world_creation_depth.md` | the PRD being amended; §3 mechanism, §4-6 ACs, §7 migration, §8 non-goals, §9 open questions |
| `docs/30_architecture/world_model/01_engine_capability_audit.md` | what the engine has vs the 25 obligations, cited to file:line; the world-usage scan |
| `docs/superpowers/plans/2026-08-26-world-model-eight-increments.md` | the roadmap, the fence, nine closed decisions, the parallel waves |
| `docs/30_architecture/world_model/00_world_model_and_genesis_pipeline.md` | §3 version history — why v1-v5 died; §4 the schema |
| `.../2026-08-25-world-model-clean-sheet/SCHEMA-v3.md` §1,§4 · `SCHEMA-v4.md` · `SCHEMA-v5.md` · `SCHEMA-v6.md` | the contract itself |
| `.../2026-08-25-world-model-clean-sheet/R_score_grelda.md` | the reader-half evidence: rulings transmit, quantities do not |
| `docs/10_prds/prd_world_creation.md` | the baseline PRD this one amends transitively |

**Standing reviewer rules** (`harness/review.sh`): revert the change, does a test fail — report which
mutants survived. Every finding is `block`, `gate` or `accept-with-reason`, **never "noted"**. A finding
with no citation is not a finding. No scope proposals.
