# Round: write the world-genesis domain package

**Domain:** world-genesis — cluster **WE-10 · World genesis and world creation**, parent bounded
context **World Engine** (`digest/01_TOPIC_MAP.md` §WE-10). No package exists; `harness2/domain.sh`
falls through to the area brief for every genesis path. This round writes the package — the second
of the twenty, after `perception-and-knowledge` (the exemplar whose shape is copied, per
`harness2/DOMAIN_PACKAGE_TEMPLATE.md`: "nineteen copiers do not re-derive it").

**Founder rulings this round encodes** (2026-08-27, this session):
1. **World genesis is a domain on its own.** The package is written now, not when convenient.
2. **The understanding-pass design is stage-1 genesis law again.** The ADR-W006 consolidation
   deleted it settled-and-unhomed (probe PR dreamchat-world-backend#126 carries the evidence);
   ADR-W006's lookup rule is "cited by something live, or it stays out" — the package and the
   accepting ADR are the live citers, so it comes back as a live document, not as bannered evidence.

## Changes

1. **Restore the design** verbatim from `git show
   70840b9:docs/30_architecture/world_model/03_world_identity_and_the_understanding_pass.md` to
   `docs/design/2026-08-26-world-identity-and-the-understanding-pass.md` (the dated-name convention
   of `docs/design/`, where the two other live genesis designs sit). One added line under its header:
   restored 2026-08-27 from `70840b9` by ADR-P026, after the consolidation deleted it with no
   successor. Nothing else in the body changes — a restored copy that drifts is the D-6 failure.
2. **Write `docs/law/adr/ADR-P026_world_genesis_is_a_domain_and_its_design_is_restored.md`** —
   records both rulings, cites probe PR #126 as evidence, resolves from the package's decisions
   table. This is the artifact `harness2/MAINTENANCE.md` says requires founder approval; it gets it
   in this round's PR.
3. **Write `docs/domains/world-genesis.{product,tech,seams}.md`** per the template's section
   mapping. Sources, all read in full: the area dossier `docs/areas/world-genesis.md`, topic map
   §WE-10, the restored design, ADR-P020…P024, the two amending specs
   (`docs/design/2026-08-20-kickstart-arrival-choice-design.md`,
   `2026-08-21-durable-worlds-design.md`), `prd_world_creation.md`, SPEC-028/SPEC-036, the probe
   round's findings (PR #126) for traps and open questions, and the rules register for cited ids
   (D-1, D-8, D-11, B-4, B-5, GA-2, GA-3, E-1, I-2, I-9). Decisions cited by id, never restated;
   only settled rows in decisions tables; open items in §7 only (the §E8 failure class).
4. **Register the domain in `harness2/DOMAINS.map`** (workspace repo — this makes the round
   cross-repo): `@repo dreamchat-world-backend`, `@context World Engine`,
   `@package docs/domains/world-genesis`, globs taken from AREAS.map's world-genesis block plus the
   genesis design docs, every glob verified against `git ls-files` before it lands (the map's own
   header rule).
5. **Area dossier**: add a pointer line naming the package as the governing document
   (START_HERE §4: package wins). Delete only lines the package supersedes with different content —
   following the perception precedent, where the dossier lives on beside the package because the
   area (review routing, art seams) is wider than the domain.
6. **Round audit trail**: plan and both reviewer outputs land in
   `docs/design/2026-08-27-world-genesis-domain-package-round/` (PLAN.md, REVIEW_plan.md,
   REVIEW_diff.md), as the probe round did.

**Not changed, so it is not misread as an omission:** no code; no prompts; no schemas; the twelve
testworld briefs stay in history (recoverable per-round as bannered evidence — the probe's method —
until something live cites them); the fill mechanism is untouched (founder ruled 2026-08-27 the fill
probe runs on the flawed identity as-is; that is the NEXT round and its ruling is recorded in the
package's open questions); `apply_ruled_event` and everything outside WE-10 untouched.

## Cited ids (each verified to resolve)

GA-2, GA-3, D-1, D-6, D-7, D-8, D-11, B-4, B-5, E-1, I-2, I-9, ADR-P020, ADR-P021, ADR-P022,
ADR-P023, ADR-P024, SPEC-028, SPEC-036, workspace:ADR-W005, workspace:ADR-W006. New id minted:
ADR-P026 (next free in `docs/law/adr/`; P025 exists).

## Validation

- **The guard watched going red:** `harness2/domain.sh core/api/worldgenesis.go` falls through to
  the area brief BEFORE the DOMAINS.map entry (recorded), and prints the world-genesis package
  AFTER it.
- `harness2/domain.sh --ask` returns package rows for a genesis question after the round.
- Every DOMAINS.map glob matches ≥1 file in `git ls-files` output.
- Restoration verbatim: diff of the restored file against `git show 70840b9:<old path>` shows
  exactly the one added restoration line.
- Every id cited in the package, the ADR, and the PR bodies resolves per the `harness2/START_HERE.md`
  table.
- `harness/check.sh` green (docs-only in the backend; DOMAINS.map + journal in the workspace).
- No Go/DB suite runs: no code path changes; the package's own validation section names the suites
  future code rounds run.

## Review

Paired world-genesis expert at two checkpoints (plan, then diff), same setup as the probe round:
role brief + area dossier + this plan (diff round adds `harness/review.sh main` output and the
diff). Verdicts block/gate/accept-with-reason only. A block goes to the founder.

## Contingencies

- A CLOSED answer from `domain.sh --ask` contradicting a location/shape choice here: follow it,
  note the substitution. (Asked twice at open; nothing returned contradicts this plan.)
- If the diff reviewer finds the package restates law instead of citing it, the restatement is
  deleted, never annotated.
