# 01 — World-State Strategy (The Doctrine)

**Status:** Frozen. This is the document everyone reads first. It explains *why* the system is shaped the way it is. Decisions are frozen in `02_world_state_adrs.md`; implementation detail lives in docs 03–07; product framing lives in 08.

---

## 1. What DreamChat is solving

DreamChat is an AI-driven open RPG world. Current AI roleplay systems treat memory as chat retrieval or summarization, and they fail in predictable ways: NPCs forget important things, know things they could not possibly know, relationships silently reset, rumors quietly become truth, and timelines collapse into lossy summaries. The failure is structural, not a prompt-engineering problem: a conversation transcript is not a world model, and no amount of retrieval makes it one.

The doctrine in one line:

> **The world state is not the transcript. The transcript is evidence. Canon is extracted, validated, stored, and projected.**

## 2. The paradigm

The world is a deterministic, append-only ledger of **Canon Events** — meaningful, validated occurrences — from which everything else is mathematically *derived* and fully *rebuildable*:

```
Canon Events → State Mutations → Materialized Projections
            → Perception Records → Context Assembly → Narration
```

This is the single load-bearing decision. Its consequences:

- If the LLM hallucinates, canon is untouched (nothing reaches canon without the validation gate).
- If a projection corrupts, it is deleted and rebuilt by replaying accepted events.
- If a past event must be revised, consequences are recomputable because lineage was never lost.
- If the transcript is deleted, the world loses nothing.

Three principles govern every other decision:

1. **State is derived, never asserted.** No subsystem writes "current state" directly. Current state is a projection of the event log, maintained incrementally by deltas.
2. **LLM as oracle, system as judge.** The LLM proposes events, mutations, perceptions, and causal links. A deterministic validation layer accepts or rejects each proposal against existing world state before anything is committed. Generated prose is never the source of truth.
3. **Perception is not canon.** What happened (objective) and what any actor — including the player — believes happened (subjective) live in strictly separated layers. No UI surface and no NPC context ever reads the canon table directly.

## 3. The layered topology (why not one graph)

The system is explicitly **not** a unified world graph. A unified graph collapses into "graph soup": supernodes (a popular actor linked to thousands of events) destroy traversal performance; every relation looks equally important, so cascades run away; and canon/belief separation becomes unenforceable. Instead, five layers with one-directional derivation flow and sharply different traversal rules:

| Layer | Contents | Mutability | Traversal for propagation |
|---|---|---|---|
| 1. Canon Event spine | Immutable events + qualified event↔object relations | Append-only | n/a — it is the spine |
| 2. Causal/derivation layer | Causal bundles, provenance edges | Append + invalidate | Yes — bounded depth, DAG only |
| 3. State projections | Current actor/location/artifact/relationship state | Rebuilt incrementally | **No** — read models |
| 4. Epistemic layer | Per-holder perception records, public knowledge | Append + invalidate | Lineage queries only |
| 5. Reference graph | knows / located-in / same-scene lookups | Mutable | **Never** — lookup only |

The causal layer must remain acyclic (cycles are paradoxes and break invalidation). The reference layer is naturally cyclic and that is fine *because* it is never traversed for cascade. The Traversal Matrix in doc 03 makes this an enforced rule, not a convention.

## 4. Time has three axes

| Axis | Field | Meaning | Example |
|---|---|---|---|
| Valid time | `in_world_time` | When it happened in the fiction | The theft occurred at midnight, Day 5 |
| Transaction time | `recorded_at` | When the system committed it | The server logged it Day 7 (real time) |
| Epistemic time | `acquired_at` (per perception) | When a given holder learned it | The guard pieced it together Day 6 |

These never mix. This is what makes both "what was true at time X" and "what did the guard believe at time Y" answerable, and what makes corrections auditable: a revision changes nothing about valid time, gets a new transaction time, and the superseded record is **invalidated, never deleted** — old beliefs survive their own falsification, which is itself a gameplay mechanic (misremembering, outdated rumors, corrections spreading through a town).

## 5. Causality: authored, compound, selective

Temporal order is not causation — the cardinal sin of naive event systems is inferring causal links from sequence. DreamChat never does this. Causal links are *authored* (proposed by templates or the LLM, validated deterministically) and they are **n:n bundles, not binary arrows**: a theft is enabled by the lockpick AND the darkness AND the guard's distraction; a faction's hostility may arise from rumor A OR rumor B; weak influences accumulate by weight toward deterministic thresholds.

But causality is also **selective**. The operational rule:

> **Every meaningful world change needs provenance. Only complex downstream consequences need causal bundles.**

Provenance — which event produced this mutation or perception — is mandatory, universal, and free (it is a foreign key). Full multi-factor bundles are created only when a **template fires** or a **threshold trips**: faction shifts, major consequences, retcon-sensitive outcomes. "Mara learned the secret" needs provenance. "The city watch turned suspicious because three rumors, a public notice, and a witness testimony accumulated" needs a bundle. The bundle tables exist in the schema from day 0 so the architecture has room to grow, but routine events never carry them.

## 6. The epistemic wall

A single canon event fans out to zero-to-N perception records — one per holder who could plausibly perceive it, shaped by sensory mode, visibility scope, and distance from the source. Communication between characters is itself a canon event whose effect is a *new, possibly distorted* perception for the listener, linked by lineage to the speaker's perception. That chain **is** the rumor mechanic, and it never touches canon: the whole town can believe a ghost robbed the museum while the ledger quietly records who actually did it.

Conflicting perceptions coexist; the system never force-collapses them — the narrator resolves trust narratively using provenance (who said it, how reliable, how distorted). The player's timeline is a projection of the *player-avatar's* perception records exclusively. Knowledge has to travel; secrets are real; omniscience is a bug.

## 7. The read/write contract (why it stays fast forever)

**Write path (narrow):** the play loop appends events plus their direct radius-1 consequences (mutations, perceptions) synchronously. Mechanical actions write deterministically with no LLM. Ambiguous narrative extraction runs *asynchronously inside the correction window* — the user reads the narration immediately; extraction has the entire window to complete; proposed events are invisible to projections until accepted. Latency became a lifecycle, not a bottleneck.

**Read path (narrower):** the live loop reads only materialized projections and the active holder's perception records, through the context assembler, served from a hot cache. No event replay, no graph traversal, no deep lineage on the hot path. Everything expensive — downstream consequences, rumor spread, threshold evaluation at scale, lineage analytics — is queued and bounded. This is what guarantees context assembly stays fast regardless of world age.

**Propagation is always bounded:** radius 0 (event only), radius 1 (direct mutations/perceptions, synchronous), radius 2+ (queued, budgeted, lazily resolved via dirty flags). The dirty-NPC race is resolved at the context assembler with a three-tier ladder: just-in-time scoped re-evaluation → optimistic context injection of the pending item → stale read with an uncertainty posture. The ladder degrades in believability, never in coherence.

## 8. Technology posture

PostgreSQL with JSONB is the entire MVP: append-only tables, join tables for all n:n relations, plain triggers for projection maintenance (pg_ivm only when triggers measurably hurt), recursive CTEs with hard depth caps for bounded lineage, Redis as the hot cache, periodic snapshots to object storage. A native graph database is **explicitly excluded** from the MVP and arrives later only as an asynchronously-fed *secondary projection* for deep lineage analytics — never as the transactional source of truth. Vector search is retrieval-only: it may select candidate perceptions for context, it never determines truth, state, or causality.

## 9. Scope: PoC vs. later

**In scope for the PoC:** the deterministic spine with replay invariance; fast-path mechanical actions; the canonization pipeline with the correction window; entity resolution; context assembly with knowledge boundaries; epistemic depth (direct/told/overheard/public/rumor, distortion, lineage); selective causality (templates + thresholds); backstage worker v1; present-forward correction; the two-tier test scenarios.

**Explicitly parked:** deep retroactive timeline rewrite and timeline forks; graph-DB projection; fine-tuned SLM extractor; full social-memory propagation at scale; full civilization simulation; multiplayer and marketplace implications.

## 10. Anti-patterns (the doctrine's negative space)

Inferring causality from sequence. Letting LLM summaries or narration become source of truth. Canonizing chatter. Unbounded cascades. Traversing reference/temporal/scene edges for propagation. Exposing canon to any UI or NPC context. Mixing the three time axes. Deleting instead of invalidating. Losing provenance after summarization. Adopting a graph DB before traversal depth demands it. Storing state only as prose. One graph where every edge looks equally important. Each of these is enforced somewhere in docs 03–07, not merely discouraged.
