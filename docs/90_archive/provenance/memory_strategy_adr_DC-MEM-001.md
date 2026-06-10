# ADR/DC-MEM-001 — Canon-First Memory Strategy and Continuity Bottleneck

**Status:** Proposed  
**Product:** DreamChat  
**Date:** 2026-06-09  
**Scope:** Memory, RAG, canon, retrieval, knowledge boundaries, backstage continuity  
**Related concepts:** MIRA memory/RAG review, Information Bottleneck Theory, DreamChat World Core, Memory / Canon / Timeline layer

---

## 1. Decision Summary

DreamChat should not be designed as a chatbot with RAG.

DreamChat should be designed as a **canon-first persistent world system** where retrieval supports continuity but does not decide truth.

The core decision is:

> **RAG retrieves supporting memory. Canon decides truth.**

DreamChat can benefit from MIRA’s memory architecture, especially its governed retrieval substrate: embedded records, source metadata, entity links, access control, reranking, and auditability. But DreamChat needs stricter world-continuity rules than MIRA because the product is not only remembering user context. It is maintaining a fictional world with entities, relationships, timeline, secrets, public/private knowledge, corrections, and in-world time.

The recommended DreamChat memory model is:

> **Lossless canon. Compressed memory. Contextual retrieval.**

This means:

- Canonical world events and state transitions should be stored in structured, durable form.
- Raw play history should be compressed into useful episodic and semantic memories.
- RAG should retrieve relevant context for narration and reasoning.
- The narrator and LLM agents should receive memory as evidence, not as authority.
- The World Core must validate, commit, reject, or correct proposed changes.

---

## 2. Background

DreamChat’s product promise is a persistent AI RPG world where dynamic NPCs remember, relationships evolve, and time matters inside the story.

This creates a different memory problem from a normal assistant or companion chatbot.

A normal RAG assistant asks:

> “What relevant context should I retrieve to answer this user?”

DreamChat asks:

> “What does this world remember, what is currently true, who knows it, how did they learn it, and what must remain coherent after hundreds or thousands of future interactions?”

That is a much harder continuity problem.

The DreamChat PoC already makes this explicit. It must prove that one small world can feel persistent, remembered, and changed across in-world time. It must support long-gap entity recovery, long-running entity recall, memory evolution recall, public/private knowledge boundaries, correction UX, and backstage updates.

So the architecture cannot rely on chat history, summaries, or vector retrieval alone.

---

## 3. MIRA Lessons

The MIRA review is useful because it shows a practical memory/retrieval substrate.

Useful MIRA-inspired ideas:

1. **Postgres + pgvector**  
   A strong default for early DreamChat because it keeps structured world state and vector retrieval in the same operational database.

2. **Source records and metadata**  
   Every stored memory should know where it came from.

3. **Entity links**  
   Retrieved memories should be connected to entities, scenes, locations, artifacts, groups, and threads.

4. **Reranking**  
   Embedding similarity alone is too weak. DreamChat should rerank by current scene, entity involvement, relationship relevance, recency in in-world time, source reliability, and knowledge visibility.

5. **Auditability**  
   The system should be able to explain why a memory was retrieved, why a fact became canon, and why an entity knows or does not know something.

6. **Memory write guards**  
   The system should prevent duplicate, noisy, contradictory, or low-value memory writes.

7. **Asynchronous memory processing**  
   Not every memory update belongs in the synchronous play path. Summarization, reflection, semantic distillation, and backstage review should usually happen asynchronously.

But MIRA is not enough by itself.

MIRA is closer to assistant memory: useful context about users, files, tasks, and previous interactions.

DreamChat needs world memory:

- what is canon
- what changed
- who knows what
- whether the knowledge is direct, indirect, public, rumor, lie, inference, or record
- what is hidden from the player
- what changed during in-world time
- what was corrected
- what should remain visible in play mode versus creator/debug mode

Therefore DreamChat should reuse MIRA-like infrastructure, not MIRA’s product model.

---

## 4. Information Bottleneck → Continuity Bottleneck

Information Bottleneck Theory says a good representation compresses the input while preserving information relevant to the prediction target.

For DreamChat, the analogy is useful:

```text
X = raw play stream
    messages, narration, reactions, temporary details, user actions

T = stored memory representation
    canon events, knowledge claims, relationship deltas, scene summaries, timeline entries

Y = future continuity tasks
    recover an NPC, preserve relationships, maintain knowledge boundaries, support backstage updates
```

The DreamChat adaptation should be called:

> **Continuity Bottleneck**

The Continuity Bottleneck asks:

> What happened in this interaction that must survive compression so the world can behave correctly later?

It is not about preserving every detail. It is about preserving the right details.

The important difference from generic Information Bottleneck theory is that DreamChat has many future continuity targets:

- recovering an entity after 1,000 unrelated actions
- recalling an early topic after 1,000 related actions
- preserving memory evolution over time
- keeping private knowledge private
- knowing what became public later
- supporting backstage updates after in-world time passes
- maintaining relationship and emotional continuity
- showing “what matters now” in the UI
- allowing correction without reverse-causality collapse

So DreamChat should not compress everything into one global summary. It should write different memory artifacts for different future uses.

---

## 5. Core Principle

The central rule:

> **Do not compress authoritative canon away.**

Some things may be compressed:

- raw transcript
- narration prose
- minor reactions
- scene descriptions
- temporary conversational flow
- low-stakes details
- retrieved context packs
- summarizable scene history

Some things should remain structured and durable:

- accepted canon events
- timeline entries
- current entity state
- relationship state
- knowledge ownership
- source of knowledge
- public/private/secret status
- corrections
- promises and commitments
- unresolved threads
- meaningful artifact ownership
- location changes
- deaths, injuries, betrayals, alliances, reputation changes
- backstage updates

The resulting rule is:

```text
Lossless canon.
Compressed memory.
Contextual retrieval.
```

---

## 6. Proposed Memory Layers

DreamChat should use layered memory instead of one RAG index.

### 6.1 Canonical State

This is the current authoritative state of the world.

Examples:

- entity identity
- current location
- current status
- relationship state
- group affiliation
- artifact ownership
- open commitments
- known scene participants
- current in-world time
- current active threads

This layer should be queried directly through SQL/graph queries, not retrieved through fuzzy semantic search.

Canon is not “top-k similar text.”

Canon is truth.

### 6.2 Canon Event Log

This is the append-only or mostly append-only history of accepted changes.

Examples:

- `entity.created`
- `entity.learned`
- `relationship.changed`
- `artifact.transferred`
- `promise.made`
- `rumor.started`
- `location.changed`
- `scene.closed`
- `backstage.reviewed`
- `correction.applied`

The event log should answer:

- what happened
- when it happened in-world
- who was involved
- who witnessed it
- what changed
- what source produced the change
- whether it is accepted, corrected, rejected, or present-forward corrected

### 6.3 Knowledge Ledger

This is one of the most important DreamChat-specific layers.

It tracks who knows what and how they know it.

Example shape:

```text
knowledge_claim
  id
  world_id
  claim_text
  claim_type
  canon_event_id
  truth_status: true / false / disputed / unknown / rumor / lie / subjective
  visibility: private / shared / public / hidden
  created_in_world_time
  created_at

entity_knowledge
  entity_id
  knowledge_claim_id
  acquisition_type: direct / told / rumor / public_record / inferred / surveillance / propaganda
  source_entity_id
  source_event_id
  reliability
  trust_level
  bias
  distortion
  omissions
  uncertainty
  believes_it
  can_act_on_it
```

This prevents magical omniscience.

An NPC should not know something because the vector search retrieved it. An NPC should know it only if the Knowledge Ledger says there is a valid path.

### 6.4 Episodic Memory

This stores scene and scene-segment memories.

Examples:

- “At the harbor office, Mara accused the player of hiding the ledger.”
- “The guard presented a warrant but did not understand its political implications.”
- “The negotiation ended unresolved; Mara left angry.”

Episodic memory is useful for retrieval and narration, but it should not override canonical state.

Episodic memory should be source-linked to:

- scene
- scene segment
- canon events
- participants
- location
- in-world time
- visibility scope
- relevant entities
- unresolved threads

### 6.5 Semantic Memory

This stores distilled stable facts.

Examples:

- “Mara distrusts the user-controlled entity because of the harbor betrayal.”
- “The Red Office uses legal pressure before open violence.”
- “The player’s reputation in District Seven is dangerous but useful.”

Semantic memory should be versioned. It should not simply overwrite older meaning.

A semantic memory should know:

- what it summarizes
- what it replaced or updated
- what events support it
- what confidence it has
- whether it is public, private, or perspective-specific
- whether it belongs to the world, the narrator, a specific entity, or the user-controlled entity

### 6.6 Relationship Memory

Relationship state deserves its own layer because it is central to the product promise.

It should not be hidden inside summaries.

Relationship memory should track:

- relationship type
- trust
- fear
- affection
- resentment
- obligation
- debt
- loyalty
- betrayal
- unresolved emotional topics
- important shared history
- last meaningful interaction
- current stance
- confidence / ambiguity

It should support evolution, not only latest state.

### 6.7 Backstage Review Memory

Backstage is not just retrieval.

Backstage is a controlled world-state review process.

It should track:

- last reviewed in-world time
- decay/review pressure
- active unresolved consequences
- connected nodes
- background pressure
- world relevance
- user relevance
- volatility
- review result
- whether no change was explicitly chosen

A backstage review can conclude:

- no meaningful change
- small change
- relationship change
- knowledge update
- status change
- new dependent node
- meaningful world change, only if justified

Backstage should be lazy, bounded, and priority-driven.

### 6.8 Raw Transcript Archive

Raw messages should still be retained, at least for debugging, export, replay, and correction.

But raw transcript should not be the main memory source for live narration.

It is too noisy and too expensive.

---

## 7. Write Path

DreamChat should not write memory directly from every message.

Visible interaction and canon processing should be separated.

The hierarchy should remain:

```text
Message → Beat → Scene Segment → Scene
```

The write path should be:

```text
User input / Continue
  → intent parsing
  → current canon + relevant memory retrieval
  → world response generation
  → proposed canon changes
  → validation
  → correction window
  → accepted canon commit
  → memory candidate generation
  → async memory distillation
  → retrieval indexes updated
```

### 7.1 Message-Level Writes

Avoid writing major memory per visible message.

A message may create no canon event.

Examples:

- flavor narration
- minor facial expression
- repeated dialogue
- atmospheric description
- temporary banter

### 7.2 Beat-Level Writes

A Beat may produce small canon updates.

Examples:

- an entity learned something
- an entity refused a question
- a suspicion increased
- a minor object changed hands
- the user made a promise

### 7.3 Scene-Segment Writes

A Scene Segment should produce richer memory updates.

Examples:

- negotiation outcome
- emotional shift
- unresolved conflict
- changed relationship stance
- updated knowledge claims
- new public/private distinction

### 7.4 Scene-Level Writes

A Scene should produce:

- scene summary
- timeline entry
- final accepted canon events
- participant state updates
- unresolved threads
- relationship deltas
- embedding records
- potential backstage triggers

---

## 8. Read Path

The read path should not be “retrieve similar chunks and answer.”

It should be layered.

Recommended runtime context pack:

```text
1. Current scene state
2. Current participants
3. Direct canonical state for involved entities
4. Knowledge ledger filtered by perspective
5. Relevant relationship state
6. Open threads / promises / commitments
7. Relevant episodic memories
8. Relevant semantic memories
9. Relevant backstage review notes
10. Retrieved supporting context with citations/provenance
```

### 8.1 Retrieval Filters

Retrieval should be constrained by:

- world_id
- current scene_id
- entity_ids involved
- location_id
- in-world time
- participant visibility
- knowledge scope
- public/private/secret status
- canon status
- correction status
- relationship relevance
- unresolved thread relevance
- world_relevance_score
- user-controlled entity perspective

### 8.2 Reranking

Reranking should combine:

- semantic similarity
- entity involvement
- current scene relevance
- relationship weight
- unresolved thread weight
- recency in in-world time
- importance score
- knowledge visibility
- source reliability
- correction status
- backstage decay pressure

Embedding similarity alone is not enough.

---

## 9. Prompting Guidance

The narrator/agent should receive context in a structured pack.

Do not dump raw memories as untyped text.

Recommended format:

```text
AUTHORITATIVE CANON
- Current facts that are true.

PERSPECTIVE-SAFE KNOWLEDGE
- What the current user-controlled entity knows.
- What each active NPC knows.
- What is public.
- What is rumor or uncertain.

RELATIONSHIP STATE
- Current relationship stance and relevant emotional history.

EPISODIC SUPPORT
- Relevant past scenes and why they matter.

OPEN THREADS
- Promises, conflicts, secrets, obligations, unresolved consequences.

BACKSTAGE NOTES
- Reviewed changes or pending review pressure.

CONSTRAINTS
- What must not be contradicted.
- What this NPC must not know.
- What is hidden from play mode.
```

The prompt should explicitly say:

> Retrieved memories are supporting evidence. Canon and knowledge boundaries are authoritative.

---

## 10. MIRA-Inspired Technical Choices

### 10.1 Storage

Use Postgres as the primary world database.

Use pgvector for embedded memories.

Suggested tables:

```text
worlds
entities
scenes
scene_segments
messages
canon_events
entity_state
relationship_state
knowledge_claims
entity_knowledge
timeline_entries
memory_episodes
memory_semantic_facts
memory_embeddings
backstage_reviews
corrections
retrieval_audit
```

### 10.2 Embeddings

Embed:

- scene segment summaries
- scene summaries
- semantic memories
- knowledge claims
- timeline entries
- entity descriptions
- relationship summaries
- unresolved threads

Do not rely on embeddings for:

- current entity status
- current location
- current relationship values
- artifact ownership
- public/private truth
- correction status
- whether an entity knows something

Those should be direct structured lookups.

### 10.3 Metadata

Every embedded memory should include:

```text
world_id
source_type
source_id
canon_event_ids
entity_ids
location_ids
scene_id
scene_segment_id
in_world_time_start
in_world_time_end
visibility_scope
knowledge_scope
truth_status
importance_score
recency_score
created_at
updated_at
```

### 10.4 Retrieval Audit

Every response should be able to store:

```text
query_text
retrieved_memory_ids
retrieval_scores
rerank_scores
included_in_prompt
excluded_due_to_visibility
excluded_due_to_canon_status
final_context_pack_hash
```

This matters for debugging memory failures.

---

## 11. Correction Strategy

DreamChat should support two correction modes.

### 11.1 Correction Window

Before the next user action accepts the generated moment into history, the user can correct the current moment.

Examples:

- “Mara was not in the room.”
- “He should not know that.”
- “This was a rumor, not confirmed.”
- “The letter was never opened.”
- “The relationship is colder than this.”

During this window, the system can rewrite the pending canon proposal before committing.

### 11.2 Present-Forward Correction

After the correction window has passed, corrections should usually affect current canon and future behavior only.

Do not automatically rewrite every downstream consequence.

This prevents reverse butterfly-effect collapse.

Example:

If the world wrongly believed an entity died and the user later corrects that the entity is alive, the PoC should treat the entity as alive from now forward. It should not automatically recalculate every past consequence.

---

## 12. Backstage Strategy

Backstage updates should be based on in-world time, not real-world absence.

Backstage review should happen when:

- the user returns to a location
- the user contacts an old entity
- a time jump occurs
- enough in-world time has accumulated
- a volatile node has not been reviewed
- a connected event creates pressure

Backstage should create a review queue, not update the whole world.

Review radius:

```text
Radius 0 = target node only
Radius 1 = directly connected nodes
Radius 2+ = usually convert to background pressure, hook, or future review
```

Backstage should avoid random drama.

A review should be justified by:

- elapsed in-world time
- existing motivations
- relationships
- background pressure
- unresolved consequences
- public/private information paths
- world_relevance_score
- user relevance
- volatility

---

## 13. Success Criteria

The memory system succeeds when:

- an entity can return after 1,000 unrelated actions with identity, relationship state, known context, and knowledge boundaries intact
- active entities can remember early topics after long related sequences
- the system remembers how a topic changed over time
- entities do not know private events without a valid information path
- relationship state evolves but does not randomly reset
- backstage updates feel justified, not random
- the user can correct memory/canon issues
- the world feels persistent, not like disconnected chat scenes

The core user-facing success statement:

> This world remembers what matters.  
> The people/entities inside it have their own context.  
> Time changed things.  
> The world did not become confused.  
> I can return to it and still believe in it.

---

## 14. Failure Modes to Avoid

### 14.1 Chat Transcript as Truth

If truth only lives in chat history or generated prose, the world will eventually collapse.

Avoid.

### 14.2 Naive RAG as Memory

If the narrator simply retrieves top-k similar chunks, NPCs will become omniscient and contradictions will leak into play.

Avoid.

### 14.3 Global Summaries

If everything becomes one global summary, memory evolution is lost.

Avoid.

### 14.4 Latest-State Collapse

If the system only remembers the latest version of a topic, it loses how relationships and beliefs evolved.

Avoid.

### 14.5 Every Message Becomes Canon

If every visible message becomes a canon event, the world model becomes noisy, expensive, and fragmented.

Avoid.

### 14.6 Backstage Explosion

If every time jump updates every node, the system becomes expensive and chaotic.

Avoid.

### 14.7 Hidden State Leakage

If creator/debug truth leaks into play mode, the world becomes omniscient and less believable.

Avoid.

---

## 15. Implementation Recommendation for PoC

Build a small but correct version.

### PoC Memory Scope

Include:

- one world
- one user-controlled entity
- 5–10 meaningful entities
- 3–5 locations
- relationship state
- timeline
- canon event log
- knowledge ledger
- scene segment summaries
- pgvector retrieval
- correction window
- present-forward correction
- backstage review queue

Do not build yet:

- full social propagation system
- full global simulation
- public marketplace
- full module ecosystem
- advanced timeline rewrite engine
- huge lorebook system

### MVP Technical Slice

1. Store every accepted canon event.
2. Store current entity/relationship/location state.
3. Store who knows what and how they know it.
4. Summarize each Scene Segment.
5. Embed Scene Segment summaries and semantic facts.
6. Retrieve context by entity, scene, relationship, and semantic similarity.
7. Filter retrieved context by knowledge boundaries.
8. Rerank by continuity relevance.
9. Let the narrator generate a response using structured context.
10. Validate proposed canon changes before commit.
11. Allow correction before acceptance.
12. Audit what memory was used.

---

## 16. Final ADR Decision

DreamChat will use a **Canon-First Memory Engine** inspired by MIRA-style retrieval infrastructure.

The system will not treat RAG as the source of truth.

The World Core owns canon, knowledge boundaries, correction, and timeline.

RAG retrieves supporting context.

The Continuity Bottleneck decides what raw interaction details should become durable memory.

The guiding rule is:

> **Lossless canon. Compressed memory. Contextual retrieval.**

This is the memory strategy most aligned with DreamChat’s product promise: a persistent AI RPG world where NPCs remember, relationships evolve, and time matters inside the story.

---

## 17. Reference Inputs

This document is based on:

- DreamChat Product Vision and Promise
- DreamChat PoC Scope and Success Criteria
- DreamChat Platform Architecture
- DreamChat Core User Experience Loop
- DreamChat Modular / Plug-and-Play Module Architecture
- Prior MIRA repository memory/RAG review
- User-provided Information Bottleneck framing
