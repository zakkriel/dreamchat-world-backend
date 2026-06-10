# 05 — Entity Resolution Specification

**Status:** Defines the subsystem that turns natural-language mentions into entity UUIDs. This is the concentrated-risk component of the whole pipeline (ADR-013): under template-first extraction, slot-filling accuracy *is* extraction accuracy, and a wrong resolution produces a database that is **wrong while structurally valid** — worse than a failed extraction, because it corrupts silently.

Design stance for the PoC: **scene-scoped and conservative.** Prefer a rejected proposal over a guessed canon.

---

## 1. The problem, concretely

The extractor reads: *"the guard warned her about the old vase."* The system must resolve: which guard (there are two active)? who is "her" (last female referent? the player? Mara)? which vase (the Ming vase or the kitchen vase)? Is "the mayor's assistant" an existing entity, a new one, or ambiguous? Every one of these is a slot fill, and every slot fill is a potential silent corruption.

## 2. The substrate: entity registry

`entity_registry` (DDL in doc 03 §1.5): UUID, kind, canonical name, aliases, short descriptor ("night-shift museum guard"), current scene, creating event, status. Maintained by: fast-path events (moves update `current_scene_id`), accepted creation events, and merges (§7).

**The scene slice.** For each beat, the resolver builds the candidate slice:

1. entities with `current_scene_id` = active scene;
2. the player avatar and party (always);
3. entities referenced in the last K accepted events of this scene (default K=10) — covers "she just left the room";
4. entities explicitly named in the beat transcript that exist anywhere in the world (exact/alias match) — covers "send a letter to the Duke" where the Duke is off-scene.

The slice — id, kind, canonical name, aliases, descriptor — is **injected into the extraction prompt as a whitelist**, and the extractor is instructed (and schema-constrained where the runtime allows enums) to fill entity slots only from it or with the literal sentinel `"NEW:"`-prefixed descriptor or `"UNRESOLVED"`.

## 3. Resolution algorithm (per mention/slot)

```
mention ─► (a) exact id?            extractor already chose a whitelist id ─► verify ─► done
        ─► (b) name/alias match     deterministic, case/diacritic-folded, against slice
                                    1 hit  ─► resolve
                                    >1 hit ─► ENTITY_AMBIGUOUS (+ candidates)
                                    0 hits ─► (c)
        ─► (c) pronoun/deixis       resolve against beat-local referent stack
                                    (last matching-kind/gender participant in this beat's
                                    accepted+proposed participants); confidence < 0.8 ─► AMBIGUOUS
        ─► (d) descriptor match     unique descriptor containment within slice
                                    ("the night guard" ⊂ "night-shift museum guard")
                                    1 hit ─► resolve; else ─► (e)
        ─► (e) new-entity check     §5 criteria met ─► create; else ─► ENTITY_UNRESOLVED
```

Steps (a)–(b)–(d) are deterministic. Step (c) is heuristic but beat-local and conservative. No embedding similarity in the PoC resolution path (ADR-018 keeps fuzziness out of truth paths); a fuzzy *suggestion* may populate the `candidates` array of an error, but never auto-resolves.

**Post-hoc verification (always, even for path (a)):** the chosen UUID exists, is `active`, kind matches the slot, and is scene-eligible for the role (a `witness` must be present; a `recipient` of a letter need not be). Failures → `ENTITY_OUT_OF_SCENE` or `ENTITY_UNRESOLVED`.

## 4. Ambiguity handling

Ambiguity is **never** resolved by guessing. `ENTITY_AMBIGUOUS` returns the candidate list with descriptors in the structured error (doc 04 §5.2); the single repair retry re-prompts the extractor with the candidates inline, which resolves the common case ("the guard" → the transcript said *museum*, pick guard-042). If ambiguity survives repair, the proposal parks as `pending_review` with the candidates attached — a human-resolvable item, not lost work. Optionally (post-PoC), persistent ambiguity can surface as an in-fiction clarification from the narrator ("Which guard do you mean?") — the cheapest resolver of all is the player.

## 5. New-entity creation rules

Create a new registry entry only when **all** hold:

1. the mention is introduced as new by the narration or the user ("a stranger in a grey cloak enters") — introduction, not casual reference;
2. no candidate in the slice matches at steps (b)/(d);
3. the slot's template allows entity creation (e.g., RUMOR_START's `subject` may be a new off-scene actor; THEFT's `item` may not invent artifacts);
4. the creating event itself passes the gate.

The new entity is created **by the accepted event** (`created_by_event` set), with kind, canonical name, descriptor, and scene from the event — provenance for entities, same as for everything else. Casual definite references with no candidate ("the king" in a world with no registered king) are `ENTITY_UNRESOLVED`, not auto-creation: definite articles presuppose existence, and presupposition is not introduction.

## 6. Unresolved references

`ENTITY_UNRESOLVED` after repair → the proposal (or the affected sub-proposal) is rejected and logged. Data integrity beats event-log completeness: a missing event is recoverable (the user can restate; backstage can re-derive); a corrupt event poisons projections, perceptions, and every downstream belief. The extraction log keeps the full context so recurring unresolved patterns drive registry hygiene (missing aliases, missing descriptors) — the practical fix is usually better registry data, not smarter matching.

## 7. Registry hygiene

- **Aliases** accrete from accepted usage: when repair resolves "the night guard" → guard-042, add the surface form to aliases (deduped, capped).
- **Merges:** if two registry entries are discovered to be the same entity, merge via a compensating event; the loser's status → `merged` with a pointer; resolution treats merged ids as redirects. Never delete.
- **Descriptors** are mandatory at creation for kinds prone to ambiguity (actors, artifacts) — enforced by the gate.

## 8. Metrics & acceptance (feeds doc 07)

Tracked from Phase 2 on, per world: resolution rate (resolved / total slots), ambiguity rate, repair-success rate on entity errors, false-resolution rate (audited by sampling against transcripts — the one metric that matters most and can't be computed automatically), new-entity precision (sampled). PoC acceptance gates: false-resolution < 1% on the audit sample; unresolved-after-repair < 5% of slots; zero guessed resolutions by construction.
