# 11 — Open Concerns & Soft Spots (For Review)

**Status:** Deliberately not a confident document. This is an honest register of where the set is thinner than it looks, what is unsolved, and what is guessed. It exists so a reviewer can attack the real weak points instead of rediscovering them, and so no reader mistakes internal consistency for validation.

**Posture (ADR-025):** the architecture is converged across many rounds, but it has never been validated by running code. Its confidence ceiling is "well-reasoned and internally consistent." Several load-bearing numbers are placeholders. Phase 0–2 are the experiment that earns the confidence the rest of the set projects.

---

## O-1 — The multi-holder narrator leak (most likely to be partially intractable)

The data wall scopes perceptions per holder (doc 06). But one narrator model generating a scene with both Mara (knows the secret) and Jonas (doesn't) sees both holders' context in a single call, and can leak Mara's knowledge through Jonas's dialogue. I-3 won't catch it — it audits prompt *input*, not cross-character *output*. ADR-020 names this but does not solve it: per-holder generation calls are costly and break single-narrator coherence; output auditing is brittle against paraphrase; an accepted leak rate is unprincipled. **This is the concern I am least able to resolve on paper.** Honest status: open, validate in Phase 3, do not claim the epistemic wall holds until measured at the generation layer.

## O-2 — Reconciliation timing (a genuine architectural fork, not a detail)

Doc 09's clean repair routes assume the system can intercept a rejection before the player reads the prose; the asynchronous design means it usually can't. So reconciliation is generally retrospective and reads as in-fiction reversal, not correction. Option A (synchronous pre-validation for state-changing beats) reintroduces the latency the correction-window design removed; Option B (always retrospective) risks reversals reading as NPC caprice. The set picks B for the PoC and defers the choice to evidence. **This may be the deepest unresolved design question in the project.**

## O-3 — Intra-beat ordering via `beat_seq` (novel, unvalidated)

Doc 10 §4 introduces a monotonic intra-beat sequence number so the causal layer can order events within a single coarse timestamp. This has **no prior art in the research inputs** — it is my own synthesis to fix a real gap (bundles reference "darkness at the moment of the theft," which a single scene timestamp can't express). It may be the right primitive or it may be a leaky abstraction that pushes ordering complexity into every provenance reference. Worth a skeptical look.

## O-4 — The canonization threshold is an empirical unknown dressed as a rule

"Canonize meaningful changes, not chatter" (doc 04 §1) is a crisp principle with no operational definition the extractor can apply consistently. Whether template-first extraction reliably distinguishes "Seren promises to return the vase" (canon) from "Seren grumbles about the weather" (not) is unknown until Phase 2. ADR-023 makes under-canonization the primary Phase 2 risk and audits it asymmetrically, but the honest position is: this is the single largest source of "will it actually work" risk, and no amount of specification resolves it — only logged play does.

## O-5 — Placeholder numbers (ADR-025)

The 1.5 s JIT budget, rumor weight 35 / threshold 100, 100-event snapshot cadence, token-budget percentages, K=10 scene window, the 120 s window-timeout — all invented. None is a measured decision. The dirty-ladder honesty edit (ADR-025): Tier 1 will in practice be deterministic-only and Tier 2 (optimistic injection) will be the common narrative path, not the rare fallback doc 06 implies.

## O-6 — Cost posture is understated

Steady-state LLM cost is roughly 2–3× narration alone once extraction, occasional repair retries, and JIT narrative reconciliations are counted. The set treats narration as the cost center; the world-state apparatus is a comparable second one. Product should price this in.

## O-7 — Template coverage is a representational bias on the world itself

The world becomes only as expressive as the template vocabulary; narrative that doesn't classify gets squeezed into the free-form escape hatch (→ pending review) or risks being missed (O-4). The "designer-authorable templates" framing has a shadow: template authoring is **ongoing content production with headcount implications**, not one-time setup, and coverage metrics (hit rate, unmatched-beat rate, pending-review queue age) are health signals that need watching from Phase 2.

## O-8 — The chained-action race (specified but not yet mechanized)

ADR-022 fixes the requirement: a fast player's follow-on action may depend on a still-*proposed* event from the prior beat. The gate must read same-scene pending proposals as optimistic state, or serialize acceptance per scene. The requirement is fixed; the mechanism is a Phase 2 choice and changes the gate's logic.

## O-9 — The meta-risk: convergence may be partly echo

Multiple models agreeing across many rounds is weaker evidence than it appears: they share training corpora (event sourcing, CQRS, Zep/Graphiti, the generative-agents paper), so unanimity partly reflects shared priors, not independent derivation. No one with production game-simulation or LLM-extraction-at-scale experience has reviewed this. Across the whole process, **not one hard "don't do this" disagreement has surfaced** — and that absence is itself worth distrusting. The architecture is probably right; the confidence attached to it should be lower than the polish implies.

---

## What I am NOT worried about (to focus the review)

The data architecture is sound: append-only canon, derived projections, replay rebuild, perception/canon separation, bounded propagation, Postgres-first, LLM-proposes/system-validates. These earned their confidence across the research and are not where the risk lives. The risk lives at the **seams** — narration↔canon (O-1, O-2), extraction↔meaning (O-4, O-7), and the gap between *specified* and *measured* (O-3, O-5, O-6). A review's highest value is pressure on those seams and on O-9, not on the settled spine.
