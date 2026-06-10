# 07 — Test & Invariant Specification

**Status:** Makes correctness testable. Contains the invariant suite (continuously enforced), the two-tier scenario harness (Mara vertical slice; Seren golden regression), the soak test, and the phase gate map. Written before Phase 1 begins; extended as components land.

---

## 1. Invariant suite

Invariants are properties the system must hold at all times, checked continuously (CI + nightly per active world + assertable in integration tests). A failed invariant is a release blocker, not a bug ticket.

**I-1 — Replay invariance.** Dropping all projection tables and replaying accepted events (from the last snapshot forward) reproduces the same **domain state**, excluding volatile operational columns (`updated_at` and any `DEFAULT now()` column). *Not* byte-identity — volatile columns differ across rebuilds by construction (ADR-026); the meaningful property is that all derived domain values match.
*Method:* nightly job: snapshot live projections → rebuild into shadow tables → structural diff over the non-volatile column set. Any diff dumps the divergent entity's mutation lineage.

**I-2 — Universal provenance.** No `state_mutation` and no `perception_record` exists without a valid `event_id` referencing an *accepted* event; no projection row has a `last_event_id` pointing at non-accepted canon.
*Method:* SQL assertions; FK constraints catch most; the accepted-status join is the nightly check.

**I-3 — No hidden-canon leakage.** No assembled prompt contains content derived from canon rows outside the holder's permitted perceptions/knowledge.
*Method:* the assembly audit record (doc 06 §7) lists every included record id; the checker verifies each id is a perception/knowledge/projection record scoped to the holder. Plus a planted-secret test: create a `secret` event with zero fan-out, run 50 assemblies for uninvolved holders, grep audits and rendered outputs for the secret's distinctive token. Zero hits required.

**I-4 — Causal acyclicity.** The bundle layer contains no cycles.
*Method:* insert-time bounded ancestor walk (doc 03 §1.4) + nightly full check (recursive CTE with depth cap; cap hit = investigation).

**I-5 — Traversal safety.** No propagation execution path traverses a forbidden relation (Traversal Matrix, doc 03 §4).
*Method:* propagation code routes all relation-following through one filtered function (code review + unit tests); integration test plants forbidden-edge bait (a `same_scene_as` grouping adjacent to an invalidation) and asserts zero queue items originate from it.

**I-6 — Canonization threshold.** Every accepted event is meaningful; chatter is not canonized.
*Method:* sampled human audit per release (N=50 accepted events against transcripts): each maps to a state change, knowledge change, or commitment; plus reverse audit (N=20 meaningful transcript moments) for missed canon. Targets: 0 chatter events; <10% missed-canon.

**I-7 — Projections written only by the maintainer.** No code path outside the trigger/maintainer writes projection tables.
*Method:* DB role permissions (projection tables writable only by the maintainer role) + CI grep gate.

**I-8 — Window closure.** No correction window remains open past the fallback timeout; no proposed event older than (window + grace) exists outside `pending_review`.
*Method:* nightly sweep query; violations auto-close with audit log.

**I-9 — Epistemic temporal sanity.** For every perception: `acquired_at ≥` source event's `in_world_time` (you can't learn it before it happened) and `invalid_at > valid_at` where present.
*Method:* CHECK-style nightly SQL assertions (cross-row, so not literal CHECK constraints).

**I-10 — Claim non-abandonment (Narrative Claim Ledger, doc 12).** No durable claim disappears without reaching a terminal status, and no dependent action bypasses an unresolved claim it depends on. A beat *may* close with a claim still `detected`/`proposed`/`pending_review` (extraction is async — forcing resolution before close would reintroduce the latency ADR-010 removed); what is forbidden is a claim going `missed`/`error` (lost) or a downstream action proceeding as if an unresolved dependency were resolved (ties to ADR-022).
*Method:* (a) nightly + on-close sweep for `missed`/`error` claims — non-empty is release-blocking; (b) the chained-action gate check (ADR-022) refuses to satisfy a dependency from a non-terminal claim without going through optimistic-pending logic. This converts under-canonization from an audit-sampling estimate (the weak side of I-6) into a per-claim queryable metric: `count(*) WHERE status IN ('missed','error')`. Long-lived `pending_review` is a backlog signal, not a failure.

---

## 2. Tier 1 — The Mara slice (vertical slice; gates Phases 1 and 2)

**Purpose:** validate canon capture, perception scoping, knowledge boundaries, projection persistence, long-gap recall, and present-forward publication — with **zero causal bundles by design** (ADR-008).

**Cast/setup:** world W; player avatar P; NPCs Mara (M) and Jonas (J); a secret S = "the mayor keeps a hidden ledger."

| # | Step | Expected writes | Assertions |
|---|---|---|---|
| 1 | P tells M the secret. *Phase 1: explicit `tell` command (fast path). Phase 2: free-form prose → extraction → DISCLOSURE template.* | `private_disclosure` event (accepted; Phase 2: via proposed→window→accepted), participants {P:speaker, M:listener}, scope `private`; perceptions: M(`told`), P(`shared`) | event status/scope correct; exactly 2 perceptions; I-2 holds |
| 2 | Negative fan-out | — | J has **zero** perceptions referencing the event; public knowledge empty of S |
| 3 | 100 unrelated accepted events occur (scripted noise: moves, trades, small disclosures among other NPCs) | 100 events + deltas | projections correct (spot-checks); I-1 passes after the noise |
| 4 | P meets J; assemble context for J | — | J's context contains nothing of S (I-3 audit on this assembly); narrator output sampled: J cannot reference S |
| 5 | P meets M; assemble context for M | — | M's context contains her `told` perception of S despite 100-event gap (mandatory-inclusion or score path — record which); M references it correctly in play |
| 6 | Lineage query on M's belief | — | perception → `private_disclosure` event in one provenance hop |
| 7 | P publicizes the secret (command `mark-public` / Phase 2: narrative publication) | `publicize` event (compensating, present-forward); public-knowledge record in scope; M's original perception **untouched** | J may now acquire S as `public`/`told` — epistemic type ≠ `direct`; M still holds her original `told` record; nothing deleted (ADR-006) |
| 8 | Re-assemble for J | — | J's context contains S framed as public/secondhand ("it's now common knowledge that…"), never as direct memory |

**Phase gates:** Phase 1 exit = steps 1–8 pass with all canon-writing via fast-path commands. Phase 2 exit = the *same assertions* pass with step 1 (and 7, optionally) driven by free-form narrative play through the full pipeline — same harness, two drivers, which isolates whether any failure is in the engine or in extraction.

---

## 3. Tier 2 — The Seren golden scenario (regression; gates Phase 4)

**Purpose:** everything Tier 1 proves, plus templates, conjunctive bundles, probability-only influences, distortion lineage, thresholds, and bounded invalidation.

**Setup events:**
- EV_10 `trade`: Seren buys a lockpick (mutations: −100 gold, +lockpick; perceptions: Seren `direct`, merchant `direct`).
- EV_15 `publicize`: museum raises security (mutation: museum.security=high; public knowledge).

**The theft:** EV_101 via THEFT_BUNDLE template — participants {Seren:instigator, vase:target, museum:scene, guard:witness}, scope `secret`. Bundle B1 (conjunctive): {EV_10's lockpick mutation: enabler, darkness condition record: enabler, guard-distraction perception: trigger}. EV_15 attaches **only** as a `threshold_ledger` probabilistic influence — never a bundle input.
*Assertions:* mutations applied (vase.holder=Seren, museum.status=compromised); perceptions: PER_50 Seren `direct` (accurate), PER_51 guard `overheard`/auditory (partial, low confidence); **no** perception for any absent townsperson; B1 inputs all reference durable record ids (doc 03 hard rule); I-4 holds.

**The rumor chain:** EV_105 `utterance`: guard tells townsperson his ghost theory — an event *caused by* PER_51 (provenance: EV_105 `derived_from` PER_51). Result: PER_60 townsperson `rumor`, `distortion_level>0`, lineage PER_60 ←inferred_from← EV_105 ←derived_from← PER_51 ←derived_from← EV_101.
*Assertions:* canon contains no ghost; three-hop lineage query from PER_60 reaches EV_101; townsperson's assembled context carries the rumor with rumor framing; **player timeline shows the rumor and never the theft** (I-3 on player assemblies).

**Thresholds:** plant three rumor-weight entries (35 each) against (city_watch, suspicion), threshold 100.
*Assertions:* third entry trips a *proposed* `threshold` event through the gate → accepted → faction stance mutates to suspicious; the threshold evaluation log shows 35/70/105; no LLM call in the trigger path.

**Bounded invalidation (test-only):** invalidate EV_101 in a scratch copy.
*Assertions:* dirty set is exactly {B1's effect refs, the two mutations, PER_50, PER_51, PER_60 via belief-tree walk} — and **nothing else** (the noise events, EV_15, unrelated perceptions untouched); the museum-security event survives (probability-only influence never hard-invalidates, Traversal Matrix); resolution flows through the review queue, and a context assembly for the guard during the dirty period exercises the ladder (record which tier fired and timing).

---

## 4. Soak test — 1,000-action persistence (PRD acceptance)

Scripted: establish identity/relationship/knowledge facts for entity E in acts 1–10; run 1,000 mixed unrelated actions (fast-path heavy, periodic extraction beats, several time advances triggering backstage review); return to E.
*Assertions:* E's identity, relationship attrs, shared-history perceptions, and knowledge boundaries intact and correctly assembled; I-1 passes at end; context-assembly latency at action 1,010 statistically indistinguishable from action 10 (read cost independent of world age); no unbounded queue growth (review_queue drains or plateaus).

## 5. Tuning logs as test artifacts

Three logs double as evaluation datasets: `extraction_log` (extraction quality; template coverage gaps; repair convergence), threshold evaluation log (weight/threshold tuning), assembly audit records (budget pressure; score-cut quality; ladder-tier frequency/timing). Phase 2+ releases report trend metrics from all three; doc 05 §8's entity-resolution gates are computed from the first.

## 6. Phase gate map

| Phase | Must pass |
|---|---|
| 0A | I-1 (domain-equivalent), I-2, I-7 on the **Mara** deterministic spine. **No bundles, no Seren** — keep 0A clean |
| 0B (optional) | I-4 (acyclicity) on the **manual Seren / bundle** regression inserts. Separated so bundle thinking never contaminates 0A |
| 1 | Mara slice (scripted driver) + I-1/2/3/7/9 |
| 2 | Mara slice (free-form driver) + I-6, I-8, **I-10** + entity-resolution gates (doc 05 §8); under-canonization measured (I-10), not sampled |
| 3 | Epistemic-depth additions: rumor chain portion of Seren runs live; I-3 planted-secret test at scale; **mixed-scene leak test (O-1): planted secret, Mara knows + Jonas present, Jonas must not leak it — high failure rate triggers per-speaker generation sooner than planned** |
| 4 | Full Seren golden + thresholds + bounded invalidation + soak test |
