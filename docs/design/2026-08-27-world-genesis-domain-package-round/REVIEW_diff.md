<!-- Diff-checkpoint review — world-genesis area expert (domain-package round, step 5 of the procedure).
Reviewer saw ONLY: harness/roles/area-expert.md, the post-round area dossier, PLAN.md,
harness/review.sh main output, and both diffs (backend + workspace).
Output below is verbatim. Verdict: accept — no block; three gates (findings 5, 8, 10), resolutions
recorded at the end. -->

# Diff checkpoint — world-genesis area expert (domain-package round)

**What I ran:** nothing — this checkpoint reviews six inputs only (role brief, post-round dossier, plan, `harness/review.sh main` output, both diffs); no repository reads, no git. The opening-move revert experiment for a docs round is the routing guard, and it was watched going red on record: EVIDENCE.md Gate 5 captures the BEFORE fall-through (`AREA world-genesis`, no package) and the AFTER package banner with all three files resolving. I relied on that recorded output; there is no other mutant to run in a diff that touches no code path.

**Scope check — the diff is the plan, no more.** Plan Changes 1–6 map one-to-one onto shipped files: the restored design (one provenance line added), `ADR-P026`, the three package files, the DOMAINS.map block, the dossier pointer plus the corrected doc-table rows, and the round directory (PLAN.md, REVIEW_plan.md, plus EVIDENCE.md — the artifact the plan review's own gates mandated, not scope creep). Nothing in either diff falls outside the plan's "Not changed" list: no code, no prompts, no schemas, no testworld briefs.

## Findings

1. **accept-with-reason** — *Gate 3 (verbatim restoration) discharged with output shown.* Citation: EVIDENCE.md "Gate 3" (`5a6` + the single restoration line, "nothing else differs"); the restored file's line 6 in the backend diff carries exactly that line, and the body's own header still reads "settled by grilling session", i.e. the pre-consolidation text. The shown diff output is the D-6 guard the plan review demanded (REVIEW_plan.md finding 3); it is in the round record, not asserted.

2. **accept-with-reason** — *Gate 4 (glob liveness) discharged with per-glob counts.* Citation: EVIDENCE.md "Gate 4" table vs the `world-genesis` block in the `harness2/DOMAINS.map` hunk. The map ships 33 globs; the evidence table carries 33 rows, one-to-one by name, every count ≥ 1, "Zero-match globs: 0" — and the note explains the one glob deliberately *not* added (the deleted depth PRD), with the dossier row corrected instead, which the workspace dossier hunk confirms. This is the per-glob reporting REVIEW_plan.md finding 4 required.

3. **accept-with-reason** — *Gate 5 (guard watched going red) discharged.* Citation: EVIDENCE.md "Gate 5"; role brief "Revert the change. Does a test fail?". The BEFORE capture is the fall-through output recorded at round open, not an after-only observation — precisely what REVIEW_plan.md finding 5 gated on.

4. **accept-with-reason** — *Gate 6 (docs-only premise for skipping the area gate block) holds in substance.* Citation: EVIDENCE.md "Gate 6"; the backend diff's 8 files all sit under `docs/design/`, `docs/domains/`, `docs/law/adr/` (matching `review.sh`'s "No owned backend path in this diff"); the workspace diff touches `harness2/DOMAINS.map`, the friction journal + `.current`, and `docs/areas/world-genesis.md`. One imprecision recorded: REVIEW_plan.md finding 6's literal path list ("DOMAINS.map plus the journal") omitted the dossier edit that the same review's finding 7 accepted, so the dossier hunk does not void the skip. The premise that matters — no `core/` path, so `make reset && make schema-check` and the Go suites would re-prove an unchanged baseline — is true on the diff in front of me.

5. **gate** — *D-8 was promised to the package and is absent from it.* Citation: plan Changes §3, which names the register ids the package draws on — "(D-1, D-8, D-11, B-4, B-5, GA-2, GA-3, E-1, I-2, I-9)" — and the dossier's "Decisions that bind", which lists D-8 for this area. Every id on that list appears in at least one package file except D-8: `world-genesis.product.md`'s rules table has GA-2/GA-3, B-4, B-5/D-11, E-1; `tech.md`'s has D-1, I-2, I-9, SPEC-028 and the ADRs; `seams.md` adds D-7, B-1, SPEC-036. D-8 appears in none of the three. Red-capable today: a grep for `D-8` across the package fails. Discharge before merge: a decisions row for D-8, or a recorded reason it does not bind the domain (in which case the plan's id list was wrong and the PR body says so).

6. **accept-with-reason** — *Finding 8's obligation performed: all three package files read for pasted law; the deletion contingency does not trigger.* Citation: dossier header "It cites `B-1`; it never pastes `B-1`" (D-6); plan Contingencies. Every restatement found sits either in a decisions-table "what it settles" gist column with the id in the same row (the template shape the perception exemplar set, per plan Changes §3) or is a quotation carrying its citation (`prd_world_creation.md:177`, design §8.1). Nothing restates *instead of* citing. One drift risk recorded: the D-1 `fast_path` exception and its reason ("the actors an event would reference do not exist yet") appear four times — product.md "What this domain is not", seams.md's consume row, tech.md's decisions table and traps table — while tech.md applies a one-home discipline to the two-transactions fact ("the one home for this fact"). Accepted because all four copies anchor to `D-1`, so the register stays authoritative and D-6 rules any drift; if a future round finds the copies disagreeing, the seams row is the natural survivor (the exception is a boundary fact).

7. **accept-with-reason** — *Finding 9's obligation performed: GA-2's three-genre test applied to every term and example the package ships.* Citation: dossier §4 question 12; design §3.10 (the test's own statement). The full ubiquitous-language table — brief, lane, understanding pass, identity, condition, bargain, face, departure, exclusion (exist-kind/happen-kind), register, content demand, voice, the twenty universal functions (phrased as functions, never professions), rule kinds, origin (contingent/axiomatic), stated/inferred, kickstart, arrival, refresh, `playable:false` — survives a sci-fi thriller, a workplace drama, and a horror story without exception; so do the seams and tech vocabularies. Genre names do occur — "haunted-house story" (product.md §What this domain is for), "a serial killer in the cosy village" (product.md, from design §6.1), "like Dune but underwater" (seams.md, design Q3) — every occurrence is a *named failure mode or quoted open question*, i.e. the neighbour the filler must not default to (design §3.2's mechanism), never a shipped term, slot, or archetype. That usage is the design's own and is what GA-2 exists to protect.

8. **gate** — *REVIEW_diff.md must land in the round directory before merge.* Citation: plan Changes §6, which promises `PLAN.md, REVIEW_plan.md, REVIEW_DIFF.md` in `docs/design/2026-08-27-world-genesis-domain-package-round/`; the first two are in the backend diff, and this document cannot precede its own review. Mechanizable: the file exists in the final PR or it does not.

9. **accept-with-reason** — *REVIEW_plan.md carries transcript noise in a permanent audit file.* Citation: REVIEW_plan.md lines 6–16 — a duplicated title with a stray `'.`, "## assistant" headers, and tool-call lines ("→ read(...) ⇒ ok · 306 lines") sit between the banner and the review body, though the banner promises "Output below is verbatim". Accepted because the capture is *faithful* — the noise is evidence of what the reviewer session actually did, and the nine findings and verdict are intact and legible; stripping the scaffolding is a tidy the round may do, not a condition of merge.

10. **gate** — *Three of the four friction entries have no visible disposition.* Citation: role brief closing section ("The friction journal · Ruling on the friction · Is the round CLOSED?" — binding on every reviewer's approval); the journal hunk `docs/00_workspace/friction/2026-08-28-world-genesis-domain-package.md` entries at lines 11–14. Entry 3 (stale dossier citations) is routed by this diff's dossier correction. Entries 1 (packages-live-in-owning-repo surprise), 2 (no closed-questions row for where a founder ruling lands), and 4 (01_TOPIC_MAP S06 off-by-one) are routed nowhere in either diff. Discharge before merge: the PR body or a ledger/doc edit gives each of the three a landing — a fix, a row, or a recorded "no landing exists yet and here is why". Red-capable by inspection of the PR.

**Overall verdict:** No `block` — the diff matches the plan and the law of this area, all four plan-review gates are discharged with output shown, and the round merges once findings 5, 8, and 10 discharge in the PR.


---

## Gate resolutions (implementer, before the PR opens)

1. **Finding 5 — D-8** — resolved: a `D-8` row now sits in `world-genesis.tech.md`'s decisions table
   (the art kick as D-8 applied; a slow call inside the build stream is the failure it names).
2. **Finding 8 — REVIEW_diff.md** — this file.
3. **Finding 10 — friction dispositions** — the journal turned out to be SHARED with a parallel
   domain-package effort (entries from play-loop, UX-1, canon-spine, WE-4, relationships and
   world-model writers landed in it while this round ran; a moderator arbitrates their overlaps).
   The dispositions for the entries this reviewer named: entry 3 (stale dossier citations) — routed,
   the dossier rows are corrected in the workspace diff. Entry 2 (no closed-questions row for where
   a founder ruling lands) — the row already exists: `docs/00_workspace/closed-questions.md:105`,
   landed by the parallel effort while this round ran; no edit needed. Entry 1 (packages expected
   under harness2/) — answered by the shipped artifact: the second package demonstrates packages
   live in the owning repo's `docs/domains/`, and the DOMAINS.map header states it. Entry 4
   (01_TOPIC_MAP S06 off-by-one) — logged by a parallel writer's session, not this round's evidence;
   named in the PR body, left for its owning round: routing another round's evidence would be
   deciding on their behalf.
