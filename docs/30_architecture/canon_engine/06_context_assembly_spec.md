# 06 — Context Assembly Specification

**Status:** Defines the read-side control point (ADR-014): the service that decides everything the narrator and NPCs are allowed to know, turn by turn. This is where knowledge boundaries are enforced, where the dirty-NPC race is resolved, and where the product becomes real — the player never sees the schema; the player sees what this component permits.

---

## 1. Contract

> **Input:** `(world_id, scene_id, acting_holder_id, current beat transcript tail, token_budget)`
> **Output:** a packed prompt context + an audit record of exactly what was included and why.

Hard guarantees the assembler must uphold on every call:

1. **No hidden canon.** Nothing from `canon_event` enters any prompt. All world knowledge arrives via projections and perception/knowledge records scoped to the holder. (Defense in depth: the gate checks scopes at write time; the assembler re-checks at read time; I-3 in doc 07 tests the output.)

**Scope of the guarantee (honest framing).** Epistemic isolation is **guaranteed at the data layer** (what enters the prompt is provably holder-scoped) and **measured at the generation layer** (what the model does with multi-holder context in one call is not guaranteed — see O-1, ADR-020). When a single narrator call generates for multiple holders with different knowledge, the data wall is necessary but not sufficient; the generation-side wall is an instruction plus output audit, validated empirically in Phase 3, not a proof. Do not describe the wall as "absolute" globally — it is absolute at the data layer and probabilistic at the generation layer.
2. **Holder-scoped epistemics.** Only perceptions where `holder_id` ∈ {acting holder, its faction(s)} and `invalid_at IS NULL AND expired_at IS NULL` and scope rules pass.
3. **Dirty handling before packing.** No entity enters the context while flagged dirty without passing the ladder (§4).
4. **Deterministic packing.** Same inputs → same context (modulo the live clock). Scoring and truncation are rule-based; no LLM decides what the LLM gets to know.

## 2. Assembly pipeline

```
resolve scene & present entities (registry slice)
        │
load projections: scene/location state, present actors' states,
held/visible artifacts, pairwise relationship rows for present actors
        │
dirty check on every loaded entity ──► §4 ladder if flagged
        │
select perceptions for the acting holder (§3)
        │
pack to budget (§5) ──► render fixed layout (§6)
        │
write assembly audit record (§7)
```

Data reads come from the Redis hot cache (doc 03 §7); cache miss falls through to Postgres. Target: data acquisition < 10 ms cached; full assembly excluding any JIT work < 50 ms.

## 3. Perception selection

Candidate set: the holder's active perceptions, plus public-knowledge records in scope for the current location/faction.

Score per record (generative-agents pattern, all weights configurable per world):

```
score = w_r · recency + w_v · relevance + w_i · (importance / 10)
recency    = 0.995 ^ (in-world hours since acquired_at or last retrieval)
relevance  = lexical overlap with beat tail (PoC) | embedding cosine (post-PoC, retrieval-only per ADR-018)
importance = stored 1–10, set at perception creation (template default, or extractor-suggested and gate-clamped)
defaults: w_r = w_v = w_i = 1
```

Mandatory inclusions regardless of score: perceptions about entities present in the scene (capped per entity); perceptions created in the last N in-world hours (default 24); any perception directly referenced by the beat. Then top-K by score to fill the remaining epistemic budget. Distorted/rumor records are included *with their epistemic framing* ("you heard a rumor that…", "you're not certain, but…") — the framing is part of the rendered line, derived from `epistemic_type`, `confidence`, `distortion_level`. The assembler never upgrades a rumor to a fact in rendering.

## 4. The dirty ladder (ADR-014)

When a loaded entity (projection or its perceptions) is flagged dirty:

**Tier 1 — JIT scoped re-evaluation (primary).** Budget: 1.5 s wall-clock (config). Scope: only this entity's flagged records and their queue items. Process: pull the relevant `review_queue` items, apply deterministic resolutions where possible (threshold-ledger recomputes, mutation reversals), invoke a single localized LLM resolution call only if the queue item requires narrative reconciliation; write resolving records with provenance; clear flags. Within budget → proceed with clean state.

**Tier 2 — optimistic context injection (budget exceeded).** Do not block further. Inject the *pending item itself* as a system note in the context ("System note: word is spreading that the museum was robbed — you have just heard this"), flagged as unprocessed. The NPC reacts narratively in real time. **Guardrail:** injected pending items are context, never canon — the NPC's reaction flows through normal canonization, and the injected note is recorded in the assembly audit. The queue item's priority is promoted.

**Tier 3 — stale read with uncertainty posture (last resort: queue unreachable or item malformed).** Use last-known-good state; add a posture hint ("you feel as though you've missed something recently"); promote priority; alert ops if Tier 3 fires repeatedly for the same entity.

The ladder degrades in believability, never coherence: at no tier can the NPC assert state the ledger contradicts, because Tier 2/3 only *add* hedged context — they never fabricate resolved state.

## 5. Token budget

Default allocation of the context budget (configurable; percentages of the world-state share of the prompt, i.e., excluding system frame and conversation tail owned by the chat runtime):

| Block | Share | Truncation rule |
|---|---|---|
| Scene & location state | 15% | drop cosmetic attrs first |
| Present entities (actor/artifact projections) | 25% | per-entity cap; drop non-interacting bystanders first |
| Relationship rows (acting holder ↔ present) | 10% | keep extreme values, drop neutral |
| Holder perceptions (selected, §3) | 40% | drop lowest score first; mandatory inclusions are last to go |
| Active threads / unresolved hooks | 10% | FIFO oldest-dropped |

If mandatory inclusions alone exceed budget, the assembler truncates *content length per record* (summarizing the rendered line, never the stored record) before dropping records, and logs a budget-pressure warning — recurring pressure is a tuning signal, not a normal state.

## 6. Rendered layout (fixed order)

```
[WORLD/SCENE]   location state, time of day, in-world date
[PRESENT]       entities in scene: name — one-line state
[YOU KNOW]      the holder's selected perceptions, epistemically framed, newest-last
[RELATIONSHIPS] holder ↔ present entities, qualitative rendering of attrs
[OPEN THREADS]  unresolved hooks relevant to scene
[SYSTEM NOTES]  Tier-2 injected pending items, if any (clearly marked)
```

Qualitative rendering rule: numeric state is rendered as bands ("trusts you deeply" not "trust=0.87") — numbers invite the narrator to do arithmetic; bands invite it to roleplay.

## 7. Assembly audit record

Every assembly writes: holder, scene, included record ids per block, scores at cut line, dirty-ladder tier taken per entity (with timings), budget pressure events, and the final rendered byte size. This record is what makes I-3 (no-canon-leakage) *testable*, makes "why did the NPC say that?" answerable, and feeds tuning (weights, budgets, importance defaults).

## 8. Failure posture

Cache down → Postgres direct (slower, correct). Queue down → Tier 3 with ops alert. Registry inconsistency (present entity missing projection) → exclude the entity from context, log integrity error — never improvise an entity. The assembler fails *closed* with respect to knowledge: when in doubt, the holder knows less, never more.
