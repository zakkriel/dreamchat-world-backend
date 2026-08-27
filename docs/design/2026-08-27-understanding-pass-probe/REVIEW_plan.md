<!-- Plan-checkpoint review — world-genesis area expert (paired reviewer, step 3 of the round).
Reviewer saw ONLY: harness/roles/area-expert.md, docs/areas/world-genesis.md, and the plan.
Output below is verbatim. Verdict: accept — no block; two gate findings, resolutions recorded at the end. -->

{
  "overall_correctness": "correct",
  "explanation": "Plan checkpoint — world-genesis area expert. Verdict: accept — the plan respects the law of this area; no block; two gate findings (run the dossier's unconditional area-gate commands at step 7; ground or drop the SPEC-036 citation) must be resolved before step 4 and before the PR respectively; six accept-with-reason findings recorded.",
  "confidence": 0.85,
  "findings": [
    {
      "title": "Recovery of deleted design and briefs complies with ADR-W006 as bannered evidence",
      "body": "accept-with-reason (cites workspace:ADR-W006; dossier preamble on D-6; plan §Context 'The recovery decision, made here'). The four inputs are recovered verbatim into the round directory as bannered evidence, cited by the live REPORT.md, not restored to old paths, and marked 'Not law.' That satisfies ADR-W006's restoration rule as quoted ('cited by something live, or it stays out'; the amendment exempts bannered evidence) and avoids the stale-copy failure D-6 exists to prevent. The design being executed is a non-live document; the plan carries that honestly by routing the homeless-design decision to the founder.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 380,
      "line_end": 386
    },
    {
      "title": "The probe authors nothing into the system; canon/transaction checklist does not fire",
      "body": "accept-with-reason (cites dossier 'Owns: … the only path that authors entities'; checklist items 1–5; D-1, I-2). Seat A's identity is a markdown artifact in the round directory; no route, transaction, canon write, or origin='fast_path' use occurs, so the transaction-shape, canon, replay, and provenance questions do not fire. This is why a docs-only probe is legal in the area that owns the sole authoring path — but only as long as nothing in the round wires the identity or the briefs into code.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 420,
      "line_end": 435
    },
    {
      "title": "Recovered testworld briefs must never become templates or seeds",
      "body": "accept-with-reason (cites dossier §2 trap 'World templates are deliberately unbuilt … exactly the archetype poison GA-2/GA-3 forbid'; GA-2, GA-3). Three authored-fiction briefs land in the backend tree. The banner ('Not law. Input to the 2026-08-27 understanding-pass probe') and the round-directory location keep them out of any template or seed path, and the plan proposes no starter-scene use. Accepted on that condition; any future round that promotes a recovered brief into core/db/seeds/ or a template trips this trap and is a fresh finding there.",
      "priority": 2,
      "confidence": 0.85,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 405,
      "line_end": 418
    },
    {
      "title": "GATE: run the dossier's unconditional area-gate commands at step 7",
      "body": "gate (cites dossier '### The gate for this area'; ADR-P020; plan step 1 'If it names gate commands applicable to docs rounds, run them at step 7'). The dossier conditions only make schema-contract ('if a published shape moved') and contract-drift ('if the frontend consumes it'); its first two lines — make reset && make schema-check, and the Genesis/World/Kickstart/Interview/Latitude test run — carry no condition, and schema drift is a boot refusal a user notices. The plan substitutes its own applicability judgement for the dossier's text. Mechanizable and cheap: amend step 7 to run the two unconditional lines and report what ran and what it said, not that it 'passed'. Resolve before step 4.",
      "priority": 1,
      "confidence": 0.85,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 396,
      "line_end": 400
    },
    {
      "title": "GATE: SPEC-036 is cited once and grounded nowhere in the plan",
      "body": "gate (cites role brief 'the mechanical gates are one: rule ids in a PR body must resolve' / ci/check_citations.sh; plan step 7 PR-body citation list). GA-2, GA-3, D-6, and workspace:ADR-W006 all do work elsewhere in the plan; SPEC-036 appears only in the paste list, is named nowhere in the dossier, and nothing states what it refers to. The citations gate will go red if it does not resolve, but resolution is not relevance, and nothing checks relevance except this seat. Resolve before the PR opens: state what SPEC-036 is and why it binds this work, or drop it per the plan's own fallback rule ('an id that does not resolve is cited as a file path instead').",
      "priority": 1,
      "confidence": 0.8,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 516,
      "line_end": 519
    },
    {
      "title": "Contamination policy is internally inconsistent; the keep-and-record branch must govern the report",
      "body": "accept-with-reason (cites plan step 4 'reading a forbidden source invalidates the probe' vs. plan Verification 6 'if it recurs, keep the output and record the contamination prominently in REPORT.md'). The seat prompt claims contamination invalidates; the round's actual policy after one rerun is keep-and-record. The round-level policy is the correct one — it matches 'count what shipped, not what passed' and lets the founder discount degraded evidence — but REPORT.md must then not present a contaminated result as a valid probe, since the plan's own prompt language defined it as invalid. Accepted because prominent recording is already mandated; this paragraph is the written reason.",
      "priority": 2,
      "confidence": 0.85,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 440,
      "line_end": 447
    },
    {
      "title": "Mutation opening move cannot fire on a docs-only probe; the substituted checks are adequate",
      "body": "accept-with-reason (cites role brief 'Your opening move is mechanical'; checklist item 14; plan Verification 2–6). A docs-only probe has no guard to revert and no source for ci/mutate.sh to mutate. The plan substitutes mechanical checks of the same character: verbatim-recovery diffs, exactly-twenty verdict counts, independent recounts of every number in REPORT.md, random citation spot-checks with a one-miss-reverify-all rule, and a contamination scan. These can go red and are cheap. Accepted as the evidence standard for a round whose deliverable is a report, not code.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 528,
      "line_end": 540
    },
    {
      "title": "'No proposed gates' and the seat separation comply with area law",
      "body": "accept-with-reason (cites role brief disposition table 'a gate that grades prose is now a deletion, not an addition'; plan step 6 'No proposed gates. Findings are observations a human reads'; recovered design §7.3 as the plan cites it). Every check this probe could gate on grades prose — world-specificity, refusal derivation, bargain altitude — so routing findings to a human reader is compliance, not omission. Seat separation (generator never reviewed by itself, scorer never sees the generator's prompt, criterion judge sees only the twenty answers) matches the reviewer-isolation law the plan cites.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "local://plan-review-prompt.md",
      "line_start": 490,
      "line_end": 500
    }
  ]
}

---

## Gate resolutions (implementer, before step 4 / before the PR)

1. **Dossier's unconditional gate commands** — resolved by adoption: step 7 runs `make reset && make schema-check` and `cd core/api && go test ./... -count=1 -run 'Genesis|World|Kickstart|Interview|Latitude'`, and the PR reports what ran and what it said. The conditional lines (`make schema-contract`, `contract-drift`) stay conditional; no published shape moves in this round.
2. **SPEC-036 grounding** — resolved before the PR opens: SPEC-036 is checked against `docs/open-spec-items.md`; if it resolves and binds this work its grounding is stated in the PR body, otherwise it is dropped from the citation list per the plan's own fallback rule.
