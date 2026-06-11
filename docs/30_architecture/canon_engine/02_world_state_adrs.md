# 02 — World-State ADRs (Frozen Decision Log)

**Status:** This log supersedes all ADR numbering in prior documents (merged architecture doc, Playbook v1/v2). It is the single canonical decision record. Rule: **no architectural discussion happens outside this file** — a new disagreement gets a new numbered ADR or it didn't happen.

Each ADR: Status / Decision / Rationale / Consequences. Provenance notes record which research round produced or contested the decision.

---

## ADR-001 — Canon Events are the immutable source of truth

**Status:** Accepted (unanimous across all rounds).
**Decision:** Every meaningful world change is recorded as an append-only, immutable Canon Event. Current state, knowledge, timelines, and compendium content are derived projections, rebuildable at any time by replaying accepted events. Events are never updated in place; revisions occur via compensating events and status transitions.
**Rationale:** Persistent worlds need auditability, lineage, correction, and replay. Chat history cannot safely provide any of these. Immutability is what makes every other guarantee in the system possible.
**Consequences:** Append-only enforcement at the database level (trigger forbidding row updates except lifecycle fields). Storage grows monotonically; bounded by snapshots and cold archiving. Not every message becomes canon (ADR-009's gate enforces the threshold).

## ADR-002 — Layered event-centric architecture, not a unified world graph

**Status:** Accepted (unanimous).
**Decision:** Five layers — canon spine, causal/derivation layer, state projections, epistemic layer, reference graph — with one-directional derivation and per-layer traversal rules. Canon Events are the only dependency spine. Entities carry no causal edges to each other; their relationships are projection rows whose lineage points at events.
**Rationale:** Unified graphs collapse into graph soup: supernodes kill traversal, every relation looks equally important, cascades run away, and canon/belief separation becomes unenforceable. Layering bounds propagation, localizes corruption, keeps the hot path narrow, and makes causality legible.
**Consequences:** Projection-maintenance machinery is day-one infrastructure. The Traversal Matrix (doc 03) is enforced in code.

## ADR-003 — PostgreSQL-first; graph database deferred to a secondary projection

**Status:** Accepted (unanimous).
**Decision:** The MVP runs entirely on PostgreSQL with JSONB: append-only event/relation tables, projection tables, recursive CTEs with hard depth caps, Redis hot cache, periodic snapshots to object storage. A native graph engine arrives post-PoC only as an asynchronously-fed secondary projection for deep lineage analytics — never the transactional source of truth.
**Rationale:** Index-free adjacency pays off only at traversal depth the architecture deliberately avoids on the hot path. Dual-store transactional sync is a heavy complexity tax with no MVP payoff. Relational join tables are the most inspectable representation of the required n:n relations.
**Consequences:** Deep lineage is second-class at MVP (depth-capped, off the hot path). Design the outbound change stream now even if the graph projection ships later.

## ADR-004 — Materialized projections are the read model; plain triggers first

**Status:** Accepted (unanimous; trigger pragmatism from the Gemini review round).
**Decision:** Actor, location, artifact, relationship, and knowledge state are read from materialized projection tables updated by deltas (state mutations) from accepted events. Maintenance starts with plain PostgreSQL triggers; pg_ivm or an IVM engine is adopted only when triggers measurably hurt. Projections are ephemeral: corrupt or buggy projections are dropped and rebuilt by replay.
**Rationale:** The play loop needs O(1) state reads; replay and traversal at read time are unacceptable. Boring technology first — the extension is an optimization, not a dependency.
**Consequences:** Projection builders must be idempotent (at-least-once delivery must not double-apply). Replay invariance (doc 07, I-1) is the permanent correctness check.

## ADR-005 — Perception is separate from canon; data-layer isolation is absolute, generation-layer isolation is measured

**Status:** Accepted (unanimous); generation-layer caveat added in v4.1.1.
**Decision:** Canon Events record what happened. Perception Records record what each holder believes, with epistemic type, source, confidence, distortion, and validity window. One canon event fans out to zero-to-N perceptions. All NPC context and all user-facing timelines are built exclusively from perceptions and scoped knowledge projections. Nothing outside the engine — no prompt, no UI surface — reads the canon table. (A debug/creator mode may expose canon behind an explicit flag.) **Scope of the guarantee:** epistemic isolation is *absolute at the data/context layer* — what enters a prompt is provably holder-scoped. It is *enforced and measured, but not assumed perfect, at the generation layer* — a single narrator call generating for multiple holders can still leak knowledge across characters (the unsolved O-1 problem, ADR-020). Do not state the isolation as globally absolute; that overpromises.
**Rationale:** Knowledge boundaries are the product. Rumor, secrecy, deception, mistaken belief, and information travel only exist if belief is first-class and truth is shielded. Omniscient timelines destroy the game. But honesty requires distinguishing the wall we can prove (data layer) from the wall we can only measure (generation layer).
**Consequences:** Context assembly (doc 06) enforces hidden-canon exclusion mechanically and testably (doc 07, I-3) at the data layer; the mixed-scene leak test (doc 07 §6, Phase 3) measures the generation layer, with per-speaker generation as the escalation if leakage is frequent.

## ADR-006 — Three time axes; invalidation, never deletion

**Status:** Accepted (extended-time model won over plain bitemporality in the merge round).
**Decision:** Canon events carry valid time (`in_world_time`) and transaction time (`recorded_at`). Perceptions additionally carry the holder's acquisition time and a validity window (`valid_at` / `invalid_at`, plus system-level `expired_at`). Superseded or contradicted records are closed with invalidation timestamps and remain queryable forever.
**Rationale:** Plain bitemporality cannot answer the question the product is built on: "what did *this character* believe at that moment." Misremembering and rumor correction are mechanics; they require old beliefs to survive falsification. Permanent lineage depends on never deleting.
**Consequences:** Every epistemic query carries a temporal filter. Storage grows; mitigated by snapshots/archiving.

## ADR-007 — When causality is recorded, it is recorded as causal bundles, not binary edges

**Status:** Accepted (contested in research round — bundle reification won over flat edge tables).
**Decision:** The causal primitive is the reified bundle: one bundle = one sufficient condition, with many typed inputs (events, mutations, perceptions; each with role, polarity, weight, necessity) and one effect. Conjunction lives inside a bundle; disjunction is multiple bundles for one effect. Binary edges may exist only as denormalized traversal indexes derived from bundles.
**Rationale:** A flat edge table is structurally incapable of distinguishing "A AND B caused C" from "A OR B caused C" — and their invalidation semantics differ completely. Bundles map one-to-one onto INUS conditions and PROV derivations; the model is grounded, not ad hoc.
**Consequences:** Invalidation operates on bundles: an effect is dirty when no surviving bundle suffices.

## ADR-008 — Provenance is mandatory and universal; causal bundles are selective

**Status:** Accepted (ChatGPT review round; selectivity mechanism from Gemini review; schema timing finalized in the last round).
**Decision:** Two distinct obligations. (a) **Provenance, always:** every mutation and every perception carries its source-event foreign key. (b) **Bundles, selectively:** full bundles are created only when a template fires (ADR-012) or a threshold trips (ADR-015), or via the gated free-form escape hatch routing to review. Routine events never get bundles. **The bundle tables are present in the Phase 0 schema** (schema is cheap; migration churn is not) but remain unused until Phase 4.
**Rationale:** Provenance alone answers "which event made this true" for the whole world at zero cost. Universal bundle modeling would collapse the PoC under process complexity while adding nothing for mundane events. Selectivity needs a deterministic mechanism, and templates/thresholds are exactly that.
**Consequences:** Pre-Phase-4 invalidation logic operates on provenance edges alone. "Schema-ready, selectively used" is the final converged position, superseding Playbook v2's table deferral.

## ADR-009 — LLM proposes; a deterministic validation gate decides; events have a lifecycle

**Status:** Accepted (unanimous).
**Decision:** Canon events carry the lifecycle `proposed → accepted | rejected → (retconned | superseded)`. Only accepted events drive projections and perceptions. The validation gate is deterministic backend logic checking proposals against current world state (entity existence and presence, possession, knowledge paths, temporal sanity, scope rules) and returning structured verdicts with machine-readable errors (contract in doc 04). Rejections feed a single repair retry to the extractor; persistent failure discards or parks the proposal.
**Rationale:** This is the hallucination firewall and the canonization threshold in one mechanism. The repair loop is only possible with structured errors.
**Consequences:** Small write-path latency on acceptance (hidden by ADR-010's window); a rejection/repair protocol is a required component.

## ADR-010 — Dual-pipeline canonization; the correction window IS the canonization window

**Status:** Accepted (fast/slow from Gemini review; window fusion synthesized in playbook rounds; confirmed by converged view).
**Decision:** Mechanical actions (move, take, give, trade, enter, leave, explicit commands) write fully-formed accepted events deterministically with no LLM — the fast path. Ambiguous narrative beats (promise, betrayal, rumor, threat, disclosure, relationship shift) route to asynchronous LLM extraction — the slow path — producing `proposed` events. Narration streams to the user immediately; extraction runs in the background; the user may correct the moment during the window; when the window closes, surviving proposals are validated and accepted.
**Rationale:** Proposed events are invisible to projections, so extraction latency is invisible: it has the entire window to run, retry, or fall back. A latency problem became a lifecycle design. Correction feels like directing the world, not debugging it.
**Consequences:** Canonization runs at beat granularity, never per message. The window trigger is ADR-011.

## ADR-011 — Window closure: explicit user lock primary, automatic fallback always

**Status:** Accepted (new in the final round, from the Gemini playbook).
**Decision:** The correction window closes primarily on an **explicit user lock** — the user reads the narration and acts to continue (the "Continue" interaction or simply submitting the next action), which locks the block and triggers acceptance of surviving proposals. An **automatic fallback** (idle timeout and scene-change heuristic) guarantees windows always close even if the user walks away.
**Rationale:** The explicit lock is deterministic, free (the user was going to click anyway), perfectly aligned with the correction UX, and removes the need for a clever beat classifier at MVP. The fallback prevents unbounded open windows. A learned beat-boundary classifier remains a possible later refinement, not an MVP dependency.
**Consequences:** The UI owns a small but load-bearing contract: lock semantics, window state, and the correction affordance (doc 04 §6).

## ADR-012 — Template-first extraction; schema-constrained free-form as gated escape hatch; SLM deferred

**Status:** Accepted (templates from Gemini review; layering synthesized in playbook rounds; confirmed unanimously since).
**Decision:** Extraction modes in priority order: (1) **pre-baked templates** — common narrative patterns authored once as parameterized shapes (event type + roles + optional bundle skeleton); the LLM classifies and slot-fills entity IDs only; (2) **schema-constrained free-form** for genuinely novel situations, with one repair retry, always routing to `pending_review`, never auto-accepting; (3) **fine-tuned SLM** strictly post-PoC, after the extraction log corpus exists and schemas have stabilized.
**Rationale:** The methods are layers, not rivals. Templates convert the hardest LLM task (inventing causal structure) into the easiest (classification + slot-filling), concentrate residual risk into entity resolution where it can be engineered, and double as designer-authorable content. You cannot fine-tune before training data exists.
**Consequences:** Template library format and registry are specified in doc 04 §4. Every extraction is logged from day one.

## ADR-013 — Entity resolution is a first-class subsystem

**Status:** Accepted (gap identified in playbook round; elevated by converged view and Gemini playbook).
**Decision:** All extracted natural-language references ("the guard", "her", "the old vase") resolve to entity UUIDs through a dedicated subsystem: a scene-scoped registry whitelist injected into extraction prompts; constrained selection against known IDs; deterministic post-hoc verification; conservative ambiguity handling (ambiguous → structured rejection, never a guess); new-entity creation only on clear introduction; unresolved mentions logged as validation failures.
**Rationale:** Incorrect resolution produces a database that is *wrong while structurally valid* — worse than a failed extraction, and silent. Under template-first extraction, slot-filling accuracy *is* extraction accuracy, so this subsystem carries the concentrated risk and must be deterministic where possible.
**Consequences:** Full spec in doc 05. The registry is also a Phase 1 dependency (fast-path commands need it too).

## ADR-014 — The context assembler is the control point

**Status:** Accepted (converged view promotion of the playbook's G4 + ADR-008 ladder).
**Decision:** A first-class service owns the entire read side: which entities are present; which projections load; which perceptions are valid *for this holder at this time*; dirty-flag detection and the three-tier ladder (JIT scoped re-evaluation under a hard latency budget → optimistic injection of the pending item as a system note → stale read with uncertainty posture and queue promotion); token-budget packing; and mechanical exclusion of hidden canon from every prompt.
**Rationale:** The player never sees the schema — they see what the narrator and NPCs know. If this component is weak, the world leaks secrets, forgets relevant facts, or speaks from stale state. Guardrail: optimistically injected items are *context, never canon* — the NPC's reaction still flows through normal canonization.
**Consequences:** Full spec in doc 06. The ladder degrades in believability, never coherence.

## ADR-015 — Partial-cause thresholds are deterministic

**Status:** Accepted (triple-confirmed across review rounds).
**Decision:** Weighted probabilistic influences accumulate in a threshold ledger per (target, attribute). Crossing a hardcoded, configurable threshold emits a *proposed* state-change event through the normal gate — a database-level evaluation, no LLM in the trigger path. A backstage "gossip" job may later feed accumulated sub-threshold evidence to an LLM strictly as a *proposal generator*, never as trigger authority. Every threshold evaluation is logged.
**Rationale:** LLM-judged tipping points are inconsistent, expensive, and untestable. Deterministic thresholds are tunable config with an audit trail.
**Consequences:** Weights/thresholds live in per-world config; expect tuning iterations against the logs.

## ADR-016 — Corrections are present-forward; deep retroactive rewrite is parked

**Status:** Accepted (ChatGPT review round; confirmed since).
**Decision:** Inside the window: free correction of proposed material. After acceptance: corrections apply present-forward as compensating events — the world *learns* the correction. Deep retroactive rewrite (cascading retcons through history) and timeline forks are explicitly out of PoC scope, parked behind a future advanced feature.
**Rationale:** This deletes the hardest unsolved problem from the PoC while still exercising the full lineage/invalidation machinery through present-forward paths and thresholds. It protects the world from reverse butterfly-effect collapse.
**Consequences:** The review queue and dirty-flag machinery are built anyway (Phase 4); the nightmare consumer of them is deferred.

## ADR-017 — Propagation is bounded by radii; the Traversal Matrix is enforced in code

**Status:** Accepted (unanimous).
**Decision:** Radius 0: event only (cosmetic compensations). Radius 1: direct mutations and perceptions, synchronous, in the play loop. Radius 2+: downstream consequences enter the review/conflict queue with a propagation budget and a hard depth cap, resolved by the backstage worker or lazily via dirty flags. Propagation traversals may follow only cascade-safe relations; reference, temporal, and scene edges are forbidden, enforced by a code-level filter, asserted by tests.
**Rationale:** Some incremental graph computations are provably unboundable by change size — bounds must be imposed by architecture, not hoped for. Lazy dirty-flag resolution makes corrections O(touched entities), not O(world).
**Consequences:** Transient inconsistency in cold regions is accepted by design; the ADR-014 ladder makes it invisible at the moment of contact.

## ADR-018 — Vector search is retrieval-only

**Status:** Accepted (unanimous).
**Decision:** Embeddings may score and retrieve candidate perception records for context assembly. They never determine canonical state, never create causal edges, never substitute for projections.
**Rationale:** Semantic similarity is correlation; the entire architecture exists because correlation must not be confused with state or causation.
**Consequences:** Embedding infrastructure is optional for the PoC and can be added to doc 06's selection stage without touching truth paths.

## ADR-019 — Rejected state-changing narration must be reconciled, not silently discarded

**Status:** Accepted; central timing question flagged open (doc 09 §3, doc 00).
**Decision:** When the gate rejects a proposal whose source prose asserted a durable state change, the pipeline must resolve it visibly — convert-to-attempt, diegetic contradiction repair, clarification prompt, or pending-review pause (doc 09) — never a silent discard. Flavor prose is exempt.
**Rationale:** The player's experienced fiction is the prose they read. Silent discard reintroduces, at the narration seam, exactly the drift the system exists to eliminate. This protects the player's fiction without making narration authoritative over canon.
**Consequences:** New spec (doc 09). The synchronous-vs-retrospective reconciliation choice (Option A/B) is deferred to Phase 2 evidence; PoC starts retrospective. A high clarification-prompt rate signals weak upstream extraction.

## ADR-020 — The narrator gets no omniscient pass; multi-holder generation needs a knowledge wall

**Status:** Accepted in principle; **mechanism explicitly unsolved** (doc 00 open concern O-1).
**Decision:** In a scene where one narrator generates dialogue for multiple holders with different knowledge, the system must prevent a knowledgeable holder's perceptions from leaking through an ignorant holder's mouth. The data-layer wall (per-holder perception scoping, doc 06) is necessary but **not sufficient**, because a single generation call sees all injected holders' context.
**Rationale:** I-3 audits what *enters* the prompt, not what the model *does with it across characters*. The epistemic wall is the product; a generation-side leak silently breaks it.
**Consequences:** Candidate mechanisms — per-holder generation calls (cost/latency, breaks single-narrator coherence), explicit negative constraints with output auditing (brittle against paraphrase), or an accepted leak rate — are all unsatisfying. This is the concern most likely to be partially intractable; it is **not** marked resolved. Validate empirically in Phase 3; do not claim the wall holds until measured.

## ADR-021 — The World Clock owns in-world time; extraction proposes, validation assigns

**Status:** Accepted.
**Decision:** A World Clock service assigns authoritative `in_world_time`; the LLM only proposes temporal interpretation (doc 10). Intra-beat ordering uses a monotonic `beat_seq` alongside the coarse timestamp.
**Rationale:** Fictional time is load-bearing for the epistemic and causal layers and must be deterministic and replayable. Coarse human-meaningful timestamps plus `beat_seq` give the causal layer the ordering it needs without false precision.
**Consequences:** New spec (doc 10). `beat_seq` is the one piece of temporal policy with no prior art in the research inputs and is flagged for scrutiny. Provenance and bundle inputs reference `(in_world_time, beat_seq)`.

## ADR-022 — Pending proposals participate in chained-action validation

**Status:** Accepted.
**Decision:** When a later action depends on a state change that is still a *proposed* (not yet accepted) event from an earlier beat in the same scene, the gate must resolve the earlier proposal first — either by treating same-scene pending proposals as optimistic state for the dependency check, or by serializing acceptance per scene. The chosen mechanism is a Phase 2 decision; the requirement is fixed.
**Rationale:** A fast player chains actions faster than the correction window closes; without this rule the gate rejects legal follow-on actions ("unlock the door" before the proposed "guard gives key" has accepted) and triggers spurious reconciliation.
**Consequences:** The gate's state-check reads pending same-scene proposals, not just accepted canon. Adds ordering complexity to doc 04 §5.4; specify before Phase 2.

## ADR-023 — Under-canonization is the primary Phase 2 risk and is audited asymmetrically

**Status:** Accepted (posture).
**Decision:** Missed canon (a meaningful change the player witnessed that never became an event) is treated as a more serious failure than over-canonization (harmless noise events). Phase 2 weights its missed-canon audit far more heavily than I-6's baseline sampling, with a much larger early sample, and considers a backstop rule: any narrator-asserted durable change must canonize or be flagged (ties to ADR-019).
**Rationale:** An extra event is noise; a missed event is the world forgetting — the exact drift the system exists to kill, reintroduced through the extraction layer. The asymmetry must be reflected in how aggressively each side is checked.
**Consequences:** Heavier Phase 2 audit cost early; the missed-canon metric becomes a release gate, not a trend line.

## ADR-024 — Memory consolidation / reflection is real and is parked with a placeholder

**Status:** Parked (post-PoC), explicitly acknowledged.
**Decision:** An NPC accumulating hundreds of perceptions about the player will eventually need a reflection mechanism — derived `inference`-type perceptions with provenance (the schema already supports this; no *process* exists). This is out of PoC scope but named so it is not mistaken for an oversight. Adjacent: when invalidation closes a perception, who authors the replacement's content text (template vs. LLM call) is also deferred.
**Rationale:** Banning early summarization (doc 01) was correct, but it leaves a long-horizon memory-volume problem unaddressed. Naming it prevents the schema's `inference` type from looking like a settled feature when it is only a settled *slot*.
**Consequences:** Post-PoC design item; flagged in doc 00.

## ADR-025 — Operational numbers are provisional placeholders

**Status:** Accepted (posture).
**Decision:** Every quantitative value in the set — the 1.5 s JIT budget, rumor weight 35 / threshold 100, the 100-event snapshot cadence, the token-budget percentages, the K=10 scene window — is an invented placeholder, not a measured decision. They are marked as such wherever they appear and are expected to change once Phase 0–2 produce real latency, cost, and quality data.
**Rationale:** The architecture's confidence ceiling is "well-reasoned and internally consistent," not "validated." Several load-bearing numbers wear the costume of decisions; treating them as provisional is intellectual honesty, not weakness.
**Consequences:** The three tuning logs (doc 03 §extraction_log, threshold log, assembly audit) exist precisely to replace these guesses with measurements. The honest claim about the dirty ladder is that Tier 1 will in practice be deterministic-resolutions-only and Tier 2 (optimistic injection) will be the *common* narrative path — doc 06's framing of Tier 2 as rare is corrected here.

## ADR-026 — Replay invariance is domain-equivalence, not byte-identity

**Status:** Accepted (correction to I-1).
**Decision:** The replay invariant (doc 07 I-1) compares projection state for *domain equivalence* excluding volatile columns (`updated_at DEFAULT now()` and similar), not literal byte-identity. The set's earlier "byte-identical" phrasing is corrected here and should be read as domain-equivalent everywhere.
**Rationale:** Volatile timestamp columns differ across rebuilds by construction; literal byte comparison would always fail. The meaningful property is that all derived domain state matches.
**Consequences:** The I-1 checker excludes a defined volatile-column set; doc 07's wording inherits this correction.

## ADR-027 — The Narrative Claim Ledger is the bridge between prose and canon

**Status:** Accepted (adversarial review; adopted as the common structural fix for O-2/O-4/O-7/O-8).
**Decision:** Between narration and proposed canon, the pipeline records a **Narrative Claim** for every durable assertion in the prose (doc 12). Each claim must reach exactly one terminal status (canonized / non_canon_flavor / converted_to_attempt / repaired / pending_review / missed-error) before its beat closes (invariant I-10).
**Rationale:** Going straight from prose to proposed-canon left missed assertions with no trace, so under-canonization could only be sampled, not measured. The ledger gives every durable assertion a lifecycle: under-canonization becomes a query, reconciliation becomes enforced (not merely asserted), chained actions can inspect pending claims, and template coverage gaps become visible. One lightweight table fixes four previously-separate soft spots.
**Consequences:** New spec (doc 12), new invariant I-10 (doc 07), new table `narrative_claim`. Arrives in Phase 2 with the slow path. Does not exist in Phase 0A.

## ADR-028 — High-impact narrated changes get a feasibility preflight; reversible ones use retrospective reconciliation

**Status:** Accepted (adversarial review; resolves the O-2 timing fork left open in ADR-019/doc 09).
**Decision:** A bounded hybrid. Irreversible claim types (death, severe injury, irreversible transfer, major time jump, public reveal, faction stance shift, permanent relationship break, location destruction) get a **feasibility preflight** against optimistic state *before* their committing prose streams — fail means the narrator never streams the impossible version. All other (reversible) durable claims rely on retrospective reconciliation (doc 09 Option B). Preflight is feasibility only, not full canonization, and operates on the claim ledger (doc 12 §5).
**Rationale:** All-synchronous validation reintroduces the latency the correction-window design removed; all-retrospective repair feels cheap for high-stakes irreversible events. Scoping synchronous preflight to the rare irreversible set is the correct asymmetry — latency is paid only where retrospective repair would be unacceptable.
**Consequences:** Supersedes ADR-019's "PoC starts retrospective for everything" with a committed split. Requires a defined irreversible-category list and optimistic-state read (accepted canon + same-scene pending claims). Phase 3.

## ADR-029 — Phase 0 splits into 0A (deterministic spine) and 0B (optional manual bundle regression)

**Status:** Accepted (adversarial review).
**Decision:** Phase 0A = canon, perception, projection, replay (Mara substrate only); no bundle tables in active use, no Seren. Phase 0B = optional manual Seren inserts exercising bundle tables and the acyclicity invariant. Frozen rule: *bundle tables may exist in the Phase 0 schema and manual bundle test data is allowed; no automated runtime path writes bundles before Phase 4.*
**Rationale:** Keeping bundles and Seren inside an undivided Phase 0 invites engineers to think about causality before the spine is proven. The split is a discipline guardrail, not a technical constraint — it costs nothing and protects Phase 0's only job: prove replay/projection/perception correctness.
**Consequences:** Doc 07's Phase 0 gate splits accordingly; the index build order reflects 0A first. This finalizes the bundle-timing wording that flip-flopped across rounds — it is now frozen and should not reopen without empirical evidence.

## ADR-030 — Fictional time is logical (tick + label), not a calendar timestamp

**Status:** Accepted (v4.1 hardening review). Amends ADR-021 and the doc 03 DDL.
**Decision:** In-world time is stored as `in_world_tick BIGINT` (monotonic, the only thing compared for sequence) + `in_world_label TEXT` (fiction-facing) + per-world `calendar_system` + `temporal_uncertainty BOOLEAN`, with `beat_seq` for intra-tick ordering. **Never `TIMESTAMPTZ`.** System/transaction times (`recorded_at`, `accepted_at`, `created_at`, `expired_at`) remain `TIMESTAMPTZ`. Worlds on real-world time may map a tick→timestamp in config, but the core never depends on it.
**Rationale:** DreamChat is genre-agnostic — voyages, eras, dream-time, time loops, non-Earth calendars. A `TIMESTAMPTZ` fictional clock silently imposes a Gregorian linear model on every world; the choice was buried in the DDL where it looked innocent. Logical time is the correct primitive and is far cheaper to fix now than to migrate later. (Caught in v4.1 review.)
**Consequences:** All fictional-time columns across the DDL converted (event `in_world_tick`; mutation `valid_from_tick`; perception `acquired_tick`/`valid_tick`/`invalid_tick`). Replay orders by `(in_world_tick, beat_seq, recorded_at)`. Doc 10's World Clock owns tick assignment. The transaction/epistemic/fiction axes stay distinct by column type.

## ADR-031 — High-impact preflight runs on planned intent before prose streams

**Status:** Accepted (v4.1 hardening review). Sharpens ADR-028.
**Decision:** For irreversible claim types, the narrator emits *intended* durable claims as a plan pre-pass, the feasibility preflight runs on that plan, and only on pass does the committing prose stream. The sequence is `plan intent → preflight → stream`, never `stream → detect → preflight`. The latter could only repair after the fact, defeating preflight's purpose.
**Rationale:** A claim normally exists only after prose is generated; preflighting a post-hoc claim prevents nothing. Prevention requires the intent to exist *before* streaming. (Caught in v4.1 review — a real sequencing error in v4.)
**Consequences:** The generation call for irreversible-intent beats is two-step (intent → prose). Reversible beats are unchanged (stream freely, detect after). Doc 12 §5–6 rewritten.

## ADR-032 — Claim ledger invariant is non-abandonment, not immediate closure

**Status:** Accepted (v4.1 hardening review). Corrects ADR-027/I-10.
**Decision:** I-10 is reformulated: *no unresolved claim may be ignored, and no dependent action may bypass an unresolved claim it depends on.* A beat may close with a claim in `detected`/`proposed`/`pending_review`; forbidden states are `missed`/`error` (lost) and any downstream action treating an unresolved dependency as resolved. `pending_review` is a legal resting state.
**Rationale:** The original "all claims resolved before beat close" fought ADR-010's async design — with extraction in flight at window close, hard closure becomes a blocking join and reintroduces the latency the whole pipeline avoids. Non-abandonment preserves flow while still making under-canonization a hard, queryable metric. (Caught in v4.1 review.)
**Consequences:** Beat close is non-blocking; the chained-action guard (ADR-022) enforces the dependency half. The metric `count(*) WHERE status IN ('missed','error')` is the release gate; long-lived `pending_review` is backlog, not failure.

## ADR-033 — Claim detection is narrator-hints plus an independent pass

**Status:** Accepted (v4.1 hardening review). Adds to ADR-027.
**Decision:** Durable claims are detected by *both* narrator-emitted hints (cheap first signal, returned with prose) and an independent detection pass that does not trust the hints. Disagreement between them is a high-value review flag.
**Rationale:** The reviewer correctly noted the ledger only works if detection works, and detecting with the same unreliable narrator is circular. But narrator self-reported hints are *correlated with the prose's own errors* (a hallucinated transfer gets a hallucinated hint; an omitted one gets no hint), so hints alone don't break the circularity — they'd relocate under-canonization one layer earlier and hide it behind a name. The independent pass supplies the uncorrelated signal. Hints make detection cheaper/better, not trustworthy on their own. (Adopted-with-caveat in v4.1 review — the one place the review's own proposal looked more solved than it was.)
**Consequences:** Two detection signals per beat; their disagreement is logged and routed to review. Do not collapse to hints-only.

## ADR-034 — Canonical event ordering excludes recorded_at

**Status:** Proposed (Chunk 1 retro; SPEC-002).
**Decision:** The canonical replay/derivation order over accepted events is `(in_world_tick, beat_seq)` — a domain-only key, required UNIQUE per world. `recorded_at` is removed from the ordering key. It is transaction-time (B-5) — volatile wall-clock telemetry (ADR-026 classes it volatile) — and is excluded from domain ordering. Per-world uniqueness of `(in_world_tick, beat_seq)` over accepted events is what guarantees a deterministic total order without a transaction-time tiebreaker. This uniqueness is already enforced by schema: migration `20260610090007` (0007) adds the partial unique index `uq_ce_accepted_order` on `(world_id, in_world_tick, beat_seq) WHERE status='accepted'`, with positive/negative pgTAP guards in `70_determinism_guards_test.sql`.
**Rationale:** Writing the order as `(in_world_tick, beat_seq, recorded_at)` makes a volatile transaction timestamp a domain ordering tiebreaker — a latent determinism foot-gun. Two rebuilds, or a restore at a different wall-clock, could reorder events that share `(in_world_tick, beat_seq)`, silently breaking I-1 replay invariance (which is itself domain-equivalence, not byte-identity, per ADR-026). Removing `recorded_at` from the key and instead *forbidding* ties via per-world uniqueness gives a domain-only total order that is replay-stable by construction. This keeps the three time axes (fiction / transaction / epistemic, ADR-030) cleanly separated: transaction time never participates in domain sequencing.
**Consequences:** Supersedes the replay-order phrasing `(in_world_tick, beat_seq, recorded_at)` in **doc 13 §6** (replay rule, step 3) and **doc 03 §3.4** (rebuild procedure). Those texts are to read `(in_world_tick, beat_seq)`. The "required UNIQUE per world" half is enforced by `uq_ce_accepted_order` (migration 0007), not by seed-data shape. Evidence: the Chunk 1 gate (tag `chunk-1-0A-gate`), whose determinism guards exercise the uniqueness constraint and the domain-only replay order. Closes the SPEC-002 ADR obligation (D-9).
**Provenance:** Raised as SPEC-002 in `docs/open-spec-items.md` during Phase 0A; filed as a proposed ADR in the Chunk 1 retro.
