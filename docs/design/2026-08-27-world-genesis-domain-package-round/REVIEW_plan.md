<!-- Plan-checkpoint review — world-genesis area expert (domain-package round, step 3).
Reviewer saw ONLY: harness/roles/area-expert.md, docs/areas/world-genesis.md, and PLAN.md.
Output below is verbatim. Verdict: accept — no block; four gates (findings 3-6), discharged with
shown output in EVIDENCE.md. -->


# Plan checkpoint — world-genesis area expert (domain-package round)'.



# Plan checkpoint — world-genesis area expert (domain-package round)

1. **accept-with-reason** — *The code-facing checklist does not attach.* Citation: dossier §4 questions 1–11 and 13; plan "Not changed, so it is not misread as an omission: no code; no prompts; no schemas". The transaction-shape, canon, spend, contract, prompt/seat, and perception questions all interrogate code paths and published shapes; this plan moves none of them and says so explicitly rather than by silence. Accepted on that stated ground; the diff checkpoint re-confirms the diff actually contains no such path (see finding 6).

2. **accept-with-reason** — *Restoration liveness is circular, and only a founder ruling closes the circle.* Citation: plan "Founder rulings this round encodes" §2, quoting ADR-W006's lookup rule "cited by something live, or it stays out". The live citers the plan names — the package and ADR-P026 — are minted in the same round that restores the document, so liveness is bootstrapped rather than found. That is acceptable solely because founder ruling #2 (2026-08-27) is the authority and ADR-P026 is the artifact that records it, with approval routed through this round's PR (plan Changes §2). If the PR merges without founder approval of the ADR, the restored design has no live citer and the restoration reverts to ADR-W006's default: it stays out.

3. **gate** — *Verbatim restoration must be shown, not claimed.* Citation: plan Changes §1 and Validation "diff of the restored file against `git show 70840b9:<old path>` shows exactly the one added restoration line"; dossier header "the copy is always the one that goes stale" (D-6). The one added provenance line means the restore is deliberately not byte-verbatim, so the diff check is the only guard against the D-6 drift failure the plan itself names. Mechanizable and red-capable: at the diff checkpoint the actual diff output lands in the round record (`docs/design/2026-08-27-world-genesis-domain-package-round/`), not an assertion that it passed.

4. **gate** — *Every DOMAINS.map glob proves a nonzero match.* Citation: plan Changes §4 and Validation "Every DOMAINS.map glob matches ≥1 file in `git ls-files` output"; role brief on routing that "failed by being quiet" (`core/api/governance_test.go:11-25`). A zero-match glob is precisely the silent-routing defect the role brief documents. Mechanizable and red-capable: run at the diff checkpoint and report per-glob match counts, since a glob that matched nothing is an experiment that silently tested nothing.

5. **gate** — *The guard is watched going red, with the BEFORE captured.* Citation: role brief "Revert the change. Does a test fail?"; plan Validation "`harness2/domain.sh core/api/worldgenesis.go` falls through to the area brief BEFORE the DOMAINS.map entry (recorded), and prints the world-genesis package AFTER it". This is the correct docs-round analog of the mutation experiment. Gate: the recorded fall-through output from before the map entry must exist in the round audit trail; an after-only observation is a guard nobody watched go red.

6. **gate** — *Skipping the area gate block is conditional on a docs-only diff.* Citation: dossier "### The gate for this area" (`make reset && make schema-check` and the Go suites are unconditional lines) vs plan Validation "No Go/DB suite runs: no code path changes". For a diff touching no path under `core/`, those suites can only re-prove an unchanged baseline, so the skip is proportionate — but the premise is checkable. Gate: at the diff checkpoint, confirm the diff touches only `docs/` in the backend and `harness2/DOMAINS.map` plus the journal in the workspace; any other path voids the skip and the dossier's gate block runs in full.

7. **accept-with-reason** — *The dossier survives beside the package.* Citation: role brief Load §0 ("where package and dossier disagree, the package wins and the dossier line is deleted in this round (D-6)"); plan Changes §5. The plan deletes exactly the disagreeing lines ("only lines the package supersedes with different content") and keeps the dossier alive because the area — review routing, the art seam, the boundaries table — is wider than the WE-10 domain, per the perception precedent it cites. That is D-6 compliance, not an exception to it: nothing is left standing that the package contradicts.

8. **accept-with-reason** — *No-restatement discipline can only be judged on the prose.* Citation: dossier header "It cites `B-1`; it never pastes `B-1`" (D-6); plan Changes §3 "Decisions cited by id, never restated; only settled rows in decisions tables; open items in §7 only", and Contingencies "the restatement is deleted, never annotated". The plan states the right rule, but a plan cannot prove prose, and the role brief rules that a gate which grades prose is a deletion, not an addition — so this cannot be a gate. Accepted at plan level; the diff checkpoint reads all three package files for pasted law and deletes any restatement found.

9. **accept-with-reason** — *GA-2/GA-3 exposure in the package's own examples is deferred to the diff.* Citation: dossier §4 question 12 and the trap "a starter scene is authored fiction... exactly the archetype poison `GA-2`/`GA-3` forbid"; plan Changes §3. The package and the restored design will carry worked examples and named terms, and the plan names no genre check on its own prose. Semantic, so not a gate: accepted with the obligation that the diff review applies GA-2's test — the term must make sense in a sci-fi thriller, a workplace drama, and a horror story — to every example and term the package ships.

**Overall verdict:** No `block` — the plan respects the law of this area; proceed to implementation with the four gates (findings 3–6) discharged, with output shown, at the diff checkpoint.
