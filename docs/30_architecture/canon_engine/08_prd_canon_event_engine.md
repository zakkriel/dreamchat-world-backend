# 08 — PRD: Canon Event Engine & World-State Lineage

**Status:** Product requirement. Deliberately product-level: user-visible outcomes, mechanics as experienced, acceptance criteria, non-goals. Implementation lives in docs 03–07; decisions in 02.

---

## 1. Why this feature exists

DreamChat needs a world-state engine that records meaningful fictional events as durable canon, derives the current world from those events, and preserves *who knows what* through scoped perception. The feature exists to make the world feel **persistent, remembered, and changed** across in-world time — a world the player can leave and return to, lie in, keep secrets in, and trust.

## 2. The product problem

Current AI roleplay treats memory as chat retrieval or summarization, which drifts: NPCs forget important things; NPCs know things they should not know; relationships reset; rumors become truth; timelines collapse into generic summaries. The system must distinguish, durably and per character: what happened · who participated · what changed · who perceived it · who later learned it · what became public · what remained secret · what was rumor · what caused later consequences.

## 3. Product goal

Allow one small RPG world to maintain coherent continuity over long play. The user should be able to: create meaningful events; see important history preserved; return to old entities after long gaps; trust that entities remember correctly; trust that entities do not know impossible information; watch relationships and world state evolve for reasons; and correct the world without breaking it.

## 4. Core mechanics (as experienced)

**1 — The world remembers what matters.** Meaningful moments — a promise, a theft, a secret shared, a relationship turning — become durable canon. Idle chatter does not. The player never manages this; it simply holds.

**2 — Living state.** Characters, places, items, and relationships always have a current state, and the world can explain it: "why does the merchant distrust me?" has a real answer traceable to real events.

**3 — Knowledge boundaries.** Every character knows only what they have a valid path to: experienced it, were told, overheard, read a notice, heard a rumor. Secrets are real. Telling Mara does not inform Jonas. Indirect knowledge keeps its source, reliability, and distortion — a rumor *feels* like a rumor when an NPC repeats it.

**4 — Rumors and distortion.** Information travels person to person and can mutate in transit. The town can believe a ghost robbed the museum while the truth sits untouched underneath. The player's timeline shows what *their character* knows — never the omniscient truth.

**5 — Consequences with reasons.** Significant outcomes can have multiple joint causes, and accumulating pressure (several rumors, a notice, a witness) can tip factions and relationships — predictably, with the reasons inspectable.

**6 — Directing, not debugging.** Right after a moment is narrated, the player can freely reshape or reject it (the correction window). Once a moment is accepted into history, corrections apply forward — the world *learns* the change — rather than rewriting the past. Time can pass off-screen, and the world reviews only what that passage should plausibly touch.

## 5. Experience requirements

- The play loop feels instantaneous: scene context assembles in well under a tenth of a second of data work regardless of world age; narration is never blocked by world-state bookkeeping.
- Timelines, compendium pages, and sidebars reflect the *character's* knowledge in real time.
- Corrections never stall a session; off-screen consequences resolve quietly.
- When an NPC is momentarily "catching up" on events, it reads as believable in-fiction latency, never as a glitch or contradiction.

## 6. Acceptance criteria

The feature succeeds if all hold:

1. An entity revisited after **1,000+ unrelated actions** retains identity, relationship state, shared history, and knowledge boundaries intact.
2. An active entity recalls early important context after long related interaction.
3. The system can explain why any current relationship, item state, location state, or belief exists (lineage to events).
4. Entities never know private events without a valid information path (planted-secret test: zero leaks).
5. Public / private / rumor / confirmed knowledge remain distinct in NPC behavior and in the timeline.
6. In-world time progression triggers bounded backstage review — relevant nodes only, never world-wide recomputation.
7. Correction never causes uncontrolled retroactive cascade; in-window correction is free; post-acceptance correction is present-forward.
8. The visible timeline is useful without exposing hidden omniscient truth.
9. Deleting the chat transcript loses zero world state.

Validation matrix: criteria 1 → soak test; 2, 3, 4, 5, 9 → Mara slice + planted-secret + lineage queries; 5 (distortion), 3 (multi-cause) → Seren golden; 6, 7 → Phase 3/4 gates. (Full mapping in doc 07.)

## 7. Non-goals for the PoC

Full civilization simulation; a graph database as source of truth; automatic deep retroactive timeline rewrites or timeline forks; causal discovery from observation alone; canonizing every message; an omniscient user timeline; full social-memory propagation at scale; fine-tuned extraction models; multiplayer and marketplace implications.

## 8. Open product questions (tracked, non-blocking)

- Retcon presentation: how are post-window corrections framed in-fiction without breaking the fourth wall, and how much background compute do their consequences deserve?
- Misremembering as flavor: how long should falsified beliefs persist before quiet correction — and should some never correct?
- Threshold feel: what accumulation pace makes faction shifts feel earned rather than scripted? (Empirical; tuned from the threshold logs.)
- Creator surface: how much of the inspectability (lineage, "why did this happen") is exposed to players vs. reserved for a creator/debug mode?

## 9. Success statement

The mechanic succeeds when the user feels:

> "This world remembers what matters. The people inside it know what they should know, forget what they should not know, and change because of what actually happened."
