# 00 — INDEX: DreamChat World-State Documentation Set

**Status:** This is the authoritative documentation set for implementing the DreamChat Canon Event Engine. It supersedes all prior research reports, merges, reviews, and playbooks, which are now archived as provenance. Architecture is frozen; this set is the build contract.

---

## The set

| Doc | File | Job | Read when |
|---|---|---|---|
| 00 | `00_INDEX.md` | Map, phase plan, final reconciliations | First |
| 01 | `01_world_state_strategy.md` | The doctrine — why the system is shaped this way | First, everyone |
| 02 | `02_world_state_adrs.md` | Frozen decisions, ADR-001 onward. No architecture debate outside this file. | Before challenging anything |
| 03 | `03_world_state_technical_reference.md` | Master DDL, lifecycle, projection rules, Traversal Matrix, propagation, snapshots, schema evolution | Phase 0 onward |
| 04 | `04_canonization_pipeline_spec.md` | The central service: dual pipeline, correction-window state machine, template library, validation-gate API contract (proposal + error schemas), repair loop | Phase 2 build |
| 05 | `05_entity_resolution_spec.md` | The concentrated-risk subsystem: registry, resolution algorithm, ambiguity rules, new-entity rules | Phase 1–2 build |
| 06 | `06_context_assembly_spec.md` | The read-side control point: knowledge boundaries, dirty ladder, scoring, budget, layout, audit | Phase 1 onward |
| 07 | `07_test_and_invariant_spec.md` | Invariants I-1…I-9, Mara slice, Seren golden, soak test, phase gates | Before Phase 1; lives in CI |
| 08 | `08_prd_canon_event_engine.md` | Product requirement, acceptance criteria, non-goals | Stakeholders; release sign-off |
| 09 | `09_narration_canon_reconciliation_spec.md` | What happens when the narrator depicts a change the gate rejects | Phase 2 build; closes a top gap |
| 10 | `10_world_clock_and_temporal_policy.md` | Who owns `in_world_time`; intra-beat ordering | Phase 0 onward; closes a top gap |
| 11 | `11_open_concerns_and_soft_spots.md` | Honest register of what's unsolved, guessed, or thin — for reviewers | Read for review; before trusting the polish |
| 12 | `12_narrative_claim_ledger_spec.md` | The bridge from prose to canon: every durable assertion gets a tracked lifecycle | Phase 2 build; makes under-canonization measurable |
| 13 | `13_phase_0A_engine_contract.md` | The buildable artifact: tables, Mara seed, expected rows, replay rule, pass/fail SQL | **Build this first** — ends the doc phase |

Reading order for a new engineer: 01 → 02 → 03 → 07, then the spec for the component being built (04/05/06/09/10/12). Doc 08 is the contract with product. **Doc 11 is the contract with honesty — read it before mistaking internal consistency for validation.**

**Note on this version (v4.1):** docs 09, 10, 11 and ADR-019…026 were added after the author's own adversarial self-review. Doc 12 (Narrative Claim Ledger) and ADR-027…029 were added in the first external hardening round. The v4.1 hardening round added ADR-030…033 and patched: fictional time is now **logical** (tick+label, never `TIMESTAMPTZ` — ADR-030, doc 03); high-impact preflight runs on **planned intent before prose streams** (ADR-031, doc 12 §5–6); the claim invariant is **non-abandonment, not immediate closure** (ADR-032, I-10); claim detection is **narrator-hints + an independent pass** with the circularity caveat written down (ADR-033, doc 12 §2.1); the Phase 0 gate is **split 0A/0B** cleanly (doc 07 §6); and the mixed-scene leak test (O-1) is pulled **early** into Phase 3 with a per-speaker-generation escalation trigger. The genuinely hard problems remain explicitly open (O-1 narrator leak, O-9 echo-chamber risk).

## Phase plan (summary — gates in doc 07 §6)

| Phase | Goal | LLM writes canon? | Exit |
|---|---|---|---|
| **0A** | Deterministic spine: canon, relations, mutations, perceptions, projections via plain triggers, replay rebuild (Mara substrate). Bundle tables present but unused; no Seren | No | Replay invariance (I-1, domain-equivalent) on Mara scenario |
| **0B** | *Optional* manual Seren inserts exercising bundle tables + acyclicity (I-4). No automated bundle writes | No | Manual bundle regression passes; acyclicity holds |
| **1** | Fast-path play loop + minimal epistemics: deterministic actions, perception fan-out, entity registry, context assembler v1 | No (narrator reads only) | **Mara slice passes, scripted driver**; transcript deletion loses nothing |
| **2** | Canonization pipeline: window state machine, template extraction, entity resolution, validation gate + repair, extraction logging, **Narrative Claim Ledger (doc 12)** | Proposes only | **Mara slice passes, free-form driver**; under-canonization measured via I-10 (not sampled); resolution gates (doc 05 §8) |
| **3** | Epistemic depth: scopes at full fidelity, communication events, rumor chains with distortion lineage, holder timelines | Proposes only | Live rumor chain; planted-secret at scale (I-3) |
| **4** | Selective causality + backstage: bundles via templates, deterministic thresholds, review queue + worker, dirty ladder live, present-forward correction | Proposes only | **Seren golden passes**; bounded invalidation exact; soak test (1,000 actions) |
| Parked | Graph-DB projection · SLM extractor · deep retroactive rewrite / forks · social memory at scale · multiplayer | — | Post-PoC ADRs required |

## Final-round reconciliations (recorded for provenance)

Four deltas existed between the last three inputs (Playbook v2, the converged view, the Gemini playbook). Resolutions, now reflected throughout the set:

1. **Bundle tables in Phase 0 schema, used from Phase 4.** The converged view's reversal of Playbook v2's table deferral is accepted: schema is cheap, migration churn is not, and selectivity is an authoring rule (templates/thresholds), not a schema rule. → ADR-008.
2. **Correction-window closure: explicit user lock primary, automatic fallback always.** New from the Gemini playbook; deterministic, free, aligned with the correction UX; the idle/scene fallback guarantees closure. A learned beat classifier is a later refinement, not a dependency. → ADR-011, doc 04 §3.
3. **Eight-document set.** Canonization, entity resolution, and context assembly promoted from sections to standalone specs (converged view) — they are where silent corruption lives, and each has one owner. → docs 04/05/06.
4. **Final phase sequence** per the converged view: spine → fast path → canonization → epistemic depth → causality+backstage. The Mara slice gates 1–2 with two drivers (scripted, then free-form), isolating engine failures from extraction failures.

## Build order (first two weeks)

1. Deploy the Master DDL (doc 03 §1) + append-only trigger + projection triggers. *(days 1–2)*
2. Script the Phase 0 scenario inserts (Mara events manually; Seren events manually) and stand up I-1/I-2/I-4 checks. *(days 2–4)*
3. Fast-path action handlers + entity registry + context assembler v1 (no ladder yet — nothing is dirty before Phase 3/4). *(days 4–9)*
4. Mara slice harness, scripted driver — Phase 1 gate. *(days 9–10)*
5. In parallel: window state machine skeleton + proposal/verdict schemas as code-level contracts (doc 04 §5), so Phase 2 starts against frozen interfaces.

## Governance

- The ADR log (doc 02) is the only place decisions change; changes require a new ADR superseding an old one by number.
- The Traversal Matrix (doc 03 §4) and the error-code registry (doc 04 §5.3) are enforced artifacts: code that bypasses them fails CI.
- The three tuning logs (extraction, threshold, assembly audit) are part of the product, not optional telemetry — every empirical open question in doc 08 §8 is answered from them.
- **Hard requirements (not posture), from the hardening review:** (1) the correction window is invisible by default — no pending/approval UI, Continue implicitly accepts (ADR-011, doc 04 §6); (2) template authoring is ongoing product/content work with headcount implications, not one-time setup (doc 11 O-7); (3) Phase 0 is protected from cleverness — 0A proves the spine before anything else (ADR-029); (4) the Narrative Claim Ledger (doc 12) is the under-canonization instrument — I-10 is a release gate, not a trend metric; (5) further document convergence requires *empirical* evidence (Phase 0–2 findings, failed invariants, extraction/claim data, cost/latency), not more reasoning.
- Further document rounds without running code now have negative value. The next artifact is a green I-1 on a deployed Phase 0A schema.
