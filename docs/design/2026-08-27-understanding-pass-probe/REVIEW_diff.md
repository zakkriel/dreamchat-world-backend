<!-- Diff-checkpoint review — world-genesis area expert (paired reviewer, step 7 of the round).
Reviewer saw ONLY: harness/roles/area-expert.md, docs/areas/world-genesis.md, the plan,
harness/review.sh main output, and the diff origin/main...HEAD at f7faf77.
Output below is verbatim. Verdict: accept — no block; three gate findings, resolutions recorded at the end. -->

{
  "overall_correctness": "correct",
  "explanation": "Diff checkpoint — world-genesis area expert. Verdict: accept — no block; the shipped round matches the plan with deviations disclosed and no area law violated. Three gates before the PR opens: fix REPORT.md's \"inferred seven exclusions\" sentence (the identity's own provenance marks only four inferred), produce the promised close evidence (gate commands, verbatim-recovery diffs, check.sh, friction drain, SPEC-036 grounding), and answer review.sh's workspace-harness cross-repo instruction.",
  "confidence": 0.85,
  "findings": [
    {
      "title": "Recount substitutes for mutation and passes on every number",
      "body": "accept-with-reason (cites role brief opening move; REVIEW_plan.md finding 7; plan Verification 3–5). Re-derived from the artifacts: 20 verdicts with 3 could-exist-elsewhere (answers 9, 14, 19); 13 refusal rows with 3, 9, 13 missed → 10 of 13, and rows 3+9 collapsing to one rule makes \"two distinct rules missed\" correct; 11 pressure derivations; 2 contradictions; 11 rules summing 5C+1P+3G+2V with the 6-axiomatic/5-contingent mix consistent. Citation spot-check on countable lines (tier2:190-195, tier3:290-296, tier2:21/23/41) — all exact.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/U_andantes_score.md",
      "line_start": 16,
      "line_end": 25
    },
    {
      "title": "Recovered inputs shipped as bannered evidence per plan and ADR-W006",
      "body": "accept-with-reason (cites plan step 2 banner block; workspace:ADR-W006 as the plan quotes it). All four INPUT files open with the specified banner verbatim, none restored to old paths, all cited by the live REPORT.md — satisfying \"cited by something live, or it stays out.\" The verbatim-recovery diff itself is unevidenced in the five inputs; folded into the close-evidence gate.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/INPUT_andantes_tier1.md",
      "line_start": 1,
      "line_end": 4
    },
    {
      "title": "Nothing authored into the system; the area checklist does not fire",
      "body": "accept-with-reason (cites dossier §4 items 1–13; dossier trap \"World templates are deliberately unbuilt … archetype poison GA-2/GA-3 forbid\"). No route, schema, seat prompt, seed, migration, or canon write moves. The three authored-fiction briefs land only as bannered round-directory evidence, not in seeds or template paths. The identity's twenty functions are human functions, not professions — the diff practices GA-2's discipline.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/INPUT_andantes_tier2.md",
      "line_start": 1,
      "line_end": 4
    },
    {
      "title": "Seat mechanism substitution (stateless one-shots) disclosed and isolation-preserving",
      "body": "accept-with-reason (cites plan steps 4–5 \"Spawn one subagent\"; REPORT.md \"How the probe actually ran\"). After two capacity-killed subagent attempts, the generator and criterion judge ran as stateless completions with allowed inputs embedded verbatim — no tools, no file access, so strictly stronger isolation than the plan's mechanism. The deviation is recorded in the report for the founder rather than silently absorbed.",
      "priority": 3,
      "confidence": 0.85,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/REPORT.md",
      "line_start": 193,
      "line_end": 202
    },
    {
      "title": "Seat C input excluded the generator's trailing self-assessment line",
      "body": "accept-with-reason (cites plan step 5 \"ONLY the verbatim text of section 12\"; the criterion file's banner; identity §12's closing italic advocacy line). The excluded line is Seat A advocating its own pass verdict; including it would contaminate exactly the judgement the isolation protects. Honors the plan's stated isolation reason over its letter; disclosed in both the criterion banner and REPORT.md.",
      "priority": 3,
      "confidence": 0.85,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/U_andantes_criterion.md",
      "line_start": 1,
      "line_end": 6
    },
    {
      "title": "Identity rules table carries an unnumbered eleventh rule",
      "body": "accept-with-reason (cites plan step 4 item 13 \"every rule numbered\"; identity §13 row \"—\"). The inferred single-signature rule is rowed as \"—\" instead of \"11\". Numbering exists so the recount is mechanical, and the recount survived (REPORT counts 11; kind and origin sums check), so the purpose was met. Letter deviation, not a defect.",
      "priority": 3,
      "confidence": 0.85,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/U_andantes_identity.md",
      "line_start": 100,
      "line_end": 109
    },
    {
      "title": "GATE: REPORT overstates \"inferred seven exclusions\" against the identity's own provenance",
      "body": "gate (cites REPORT.md Judgement 1 closing paragraph; identity §14: \"Exclusions 1, 5, 6, 7 `inferred`; exclusions 2, 3, 4 `stated`\"). Three of the seven exclusion rows restate brief rules the generator was handed; calling all seven \"inferred\" overstates the pass's protective inference in the headlined judgement, against the artifact's own labeling — and the plan requires every REPORT number re-derived from the artifacts. Cheap and mechanical: amend to \"emitted seven exclusions, four of them inferred\" before the PR opens.",
      "priority": 1,
      "confidence": 0.85,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/REPORT.md",
      "line_start": 48,
      "line_end": 56
    },
    {
      "title": "No visible contamination in any seat's output",
      "body": "accept-with-reason (cites plan Verification 6; identity §14). Checked the identity against tier-2/3-only material — Illa, Belna, Renn, Uma Ret, Vieja Marda, Quinta's 94%, Primera's 12,000 capacity, Campana de Marcha, the meat prohibition — none appears; everything traces to tier-1 sections the provenance table names. The criterion file carries only section-12 text; the score file cites only its five allowed files.",
      "priority": 3,
      "confidence": 0.85,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/U_andantes_identity.md",
      "line_start": 114,
      "line_end": 123
    },
    {
      "title": "GATE: round-close evidence not produced anywhere in the five inputs",
      "body": "gate (cites role brief \"Run it. Report what you ran and what it said\"; dossier \"### The gate for this area\"; plan Verification 1–8; REVIEW_plan.md gate resolutions 1–2). The plan checkpoint resolved its first gate by adoption — step 7 runs `make reset && make schema-check` and the Genesis/World/Kickstart/Interview/Latitude test line with results reported — yet nothing in the diff, review.sh output, or REPORT.md evidences those runs, the verbatim-recovery diffs, check.sh green, the friction drain, or the SPEC-036 grounding promised by resolution 2 (SPEC-036 appears nowhere in the shipped files). Resolve before the PR opens: paste what ran and what it said into the PR body, and ground SPEC-036 or drop it per the plan's fallback.",
      "priority": 1,
      "confidence": 0.85,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/REVIEW_plan.md",
      "line_start": 85,
      "line_end": 90
    },
    {
      "title": "GATE: review.sh flags workspace-harness dirty and invokes the cross-repo protocol, unanswered",
      "body": "gate (cites review.sh output: \"other surfaces dirty in this round: workspace-harness … Cross-repo round — round-protocol.md §0. Open the PRs together and name each in the other\"; REPORT.md homeless-design section). The workspace dirt is plausibly the friction journal this round opened, whose entries REPORT.md routes as homeless pending the founder's ruling — but the instruction is unconditional and the shipped work does not answer it. Resolve before the PR opens: open the paired workspace PR with mutual naming per §0, or state in the PR body exactly what the workspace-harness dirt is and why no paired PR exists.",
      "priority": 1,
      "confidence": 0.8,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/REPORT.md",
      "line_start": 170,
      "line_end": 179
    },
    {
      "title": "REVIEW_diff.md absent from the diff is correct sequencing",
      "body": "accept-with-reason (cites plan step 7 \"Commit the round directory. Run the diff-checkpoint review … output verbatim to REVIEW_diff.md\"; plan Verification 1). This review is that file; the plan's own ordering commits the round before the diff review runs, so its absence from the reviewed diff is correct. Verification 1's ten-file check must be re-run after this document lands.",
      "priority": 3,
      "confidence": 0.9,
      "file_path": "docs/design/2026-08-27-understanding-pass-probe/REPORT.md",
      "line_start": 1,
      "line_end": 6
    }
  ]
}

---

## Gate resolutions (implementer, before the PR opens)

1. **"Inferred seven exclusions" overstatement** — resolved: REPORT.md Judgement 1 now reads "emitted seven exclusions, four of them inferred (the identity's own provenance marks exclusions 2, 3, 4 as stated by the brief's rules)". The identity's §14 is the source.
2. **Round-close evidence** — resolved in the PR body: the area-gate results (make reset && make schema-check green; the Genesis/World/Kickstart/Interview/Latitude test line fails on TestRunWorldTurn_Standalone_CallableWithoutBeatLoop, verified identical on origin/main — pre-existing, this round touches no code), the four verbatim-recovery diffs (all empty), check.sh result, the friction-journal drain, and SPEC-036's grounding (it resolves at docs/open-spec-items.md:918 — "A world's own rules have no enforcement path" — and binds this work because the probe measures exactly the per-world rules genesis emits).
3. **workspace-harness dirty / cross-repo instruction** — resolved in the PR body: the workspace dirt is this round's friction journal (docs/00_workspace/friction/2026-08-27-understanding-pass-probe.md), committed in the workspace repo and named in this PR; its entries are routed in REPORT.md's homeless-design section, which states the landing spot does not exist pending the founder's ruling.
