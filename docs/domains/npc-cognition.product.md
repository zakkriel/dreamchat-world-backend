# npc-cognition · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-8 · NPC cognition and minds ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `npc-cognition.tech.md` holds how it is built; `npc-cognition.seams.md`
holds what crosses its boundary.

---

## What this domain is for

**One job: turn what a mind perceived into what it decides to do next — a proposal, never a
commit.**

Perception (WE-3) records what an actor came to know. This domain is where that knowledge becomes
an appraisal and a decision: an NPC reads the moment as *she* knows it and returns exactly one of
`none`, an act, or a wind-up. The engine commits or refuses (`D-1`); the mind only ever proposes.

The product reason it is its own domain: NPC decisions are the only thing that makes a room move
when the player does nothing — and a mind that reasons over knowledge it never earned is not a
character, it is the database talking. The wall between what everyone saw and what one mind knows
holds **by construction**: the prompt is *assembled from* that mind's perceptions, never *filtered
down from* canon (`tech.md` §The call path).

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Mind** | One present NPC a cognition seat speaks for: roster name plus its personality core, when one exists. |
| **Seat** | One LLM call with one job. Two exist here: the **shared batch** (one call, every mind whose read needs only the public moment) and the **isolated call** (one flagged NPC, her secret riding alone). Which seat is a mechanical id-intersection, never a judgment (`tech.md` §The lookups). |
| **Decision** | Exactly one per mind per action: `none` (most minds, most of the time), `commit` (acts now), or `telegraph` (winds up a disruptive act — the exception, not the rhythm). |
| **Public moment** | The modal face of every event *all* present minds hold a perception of. The only history a shared prompt carries. |
| **Private record** | A perception that fails the public test. Rides only its holder's own isolated call. |
| **Personality core** | Who she is in the room — traits and malleability, authored at genesis with provenance. **No secret ever lives here**: cores ride shared prompts. |
| **The wall** | The construction guarantee above. Distinct from WE-3's epistemic layer (who knows what) and WE-4's naming wall (what they call it). |

## What this domain is not

- **Not whether anyone noticed.** Acquisition of knowledge is Perception & Knowledge (WE-3). A
  perception is an acquisition; this domain is the appraisal and the decision.
- **Not the ruling.** The play loop (WE-7) resolves and commits every NPC attempt through the same
  pipeline the player uses — *"a 'trusted NPC' fast path is a named consistency hole"*
  (`core/api/orchestrator.go`, grep `consistency hole`).
- **Not the World Actor.** The one omniscient seat reasons over the *world* and erupts unscheduled;
  cognition seats reason over a *scene* on behalf of present minds. That seat is WE-12's
  (`core/api/worldactor.go`).
- **Not relationship state.** Modelled stance is Social & Relationships (WE-9); no stance data
  reaches a cognition prompt today (`tech.md` §Open questions).
- **Not what things are called.** Labels come from the naming wall (WE-4), per seat (`seams.md`).

## Product rules — decisions already made

Ids only; the law lives where the id resolves.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `B-2` | Knowledge enters a mind only through valid in-world paths. | A prompt assembled from canon instead of perceptions is a different product. |
| `B-11` | Event-driven cognition: appraisal updates only when the actor *perceives* something — **a law, not a preference**. | A free-running or idle-reflection loop makes minds move on wall-clock time, which the fiction does not have. |
| `B-4` | Player interiority: the player gets no mind. No `personality_core` row exists for the player — *"premise, not a mind"* (`core/db/schema.sql`, grep `Kade gets NO core`). | A seat deciding for the player authors the one reading that is theirs. |
| `B-7` | Told is not witnessed; a propagated perception is a new record. | Copying records upstream would flood the private lookups with false witnesses. |
| `D-1` | The mind proposes; the deterministic gate decides; "no" is an ordinary answer. | An NPC act that skips the gate is canon nobody validated. |

One more rule binds without a register id: **an entity's response is bounded by its competence and
character** — safety expressed as characterisation, not as a content filter. Source:
`04_parked_product_concepts.md` §5, quoted at `digest/S01_the_law_and_the_language.md` §Topic 9;
the source file itself is outside this repo (D-6 Drive layer).

## What is deliberately not built here

- **No idle mind.** No reflection loop, no self-modification: *"respected = the core is in every
  prompt and no seat may rewrite it by whim"* (`digest/S06_specs_and_final_doctrine.md` §Topic 14,
  RULINGS-2026-07-23 §8). `B-11` is why.
- **No free association in retrieval.** Named as an accepted v1 loss: *"'that smell — like the
  night of the fire' is not id-linked and will not surface mechanically"* (RULINGS-2026-07-23 §10,
  via `digest/S06…` §Topic 18). Retrieval is subject-link intersection only — do not bolt on a
  similarity search to "fix" it.
- **No personality evolution yet.** `trait_pool` exists as a table with zero readers and zero
  writers (verified, `tech.md` §Storage); the licensing machinery (magnitude, thresholds, pools) is
  a designed socket awaiting its station. Filling it is a ruling, not a gap-fix.
- **No content filter.** Refusal comes from who the character is (the competence-and-character rule
  above); the seat rulebook (`core/api/prompts/cognition.txt`) forbids sanitising and holds one
  floor. Adding a mood or appropriateness filter re-decides a founder ruling.
