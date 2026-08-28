# relationships · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-9 · Relationships, modelled and never surfaced ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `relationships.tech.md` holds how it is built; `relationships.seams.md` holds
what crosses its boundary.

---

## What this domain is for

**One job: the persistent connection between two actors, as the world logs it — never as anyone sees it.**

There **is** a relationship system; there is **no** relationship UI. Those are one design, not two in
tension. The founder, verbatim (`digest/HOW.md` §2.4, ruling 2026-08-26):

> *"NPC A1 has a certain relationship built with the User's Actor, and that is logged, and the NPC
> plays according to it. But the User's actor has its own human understanding of what the relationship
> is and it might not be what has been logged — AND THAT IS OK. So there is no UI to say what the
> relationship is."*

*"The divergence between the logged relationship and the player's belief about it **is the product**"*
(`digest/HOW.md` §2.4). What is parked is the surfacing, never the modelling.

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Relationship** | *"A meaningful connection between two Actors… trust, suspicion, loyalty, obligation, debt, rivalry, intimacy, fear, dependency, alliance, hierarchy, shared history, family, employment, political alignment, membership, betrayal, protection, hostility, secrecy, or any other persistent social/world connection"* (the parked-relationships PRD, quoted via `digest/S05_the_prds.md` topic 6 — the PRD file itself was deleted in the 2026-08-27 consolidation, see `tech.md` §Traps). |
| **The logged relationship** | What the engine records between two actors (`relationship_state`, `tech.md`). Input to NPC behaviour; never a surface. |
| **The player's belief** | The player's own human reading of where they stand. The system never authors it (`B-4`), and it may diverge from the log — that divergence is product, not drift. |
| **Relationship-flavored knowledge** | The only form in which relationship information legally reaches a user: an ordinary sourced Collected Knowledge record that entered their perspective through a valid path (`B-3`). Owned by perception (WE-3), not by this domain. |

## What this domain is not

- **Not the NPC's appraisal machinery.** What an NPC thinks, feels, or decides is NPC Cognition
  (WE-8). This domain holds the logged connection it plays from — among other inputs.
- **Not knowledge.** Who knows about a relationship, and what they would call it, is perception
  (WE-3). Relationship-flavored information reaches a user only as WE-3's records.
- **Not a surface.** There is no UI in this domain and none may be added (`B-3`; §What is
  deliberately not built).

## Product rules — decisions already made

Ids only; the law lives where the id resolves.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `B-3` | No relationship UI (MVP): modelled internally, never rendered — no panel, field, synthesis, or label. Relationship-flavored information surfaces only as ordinary sourced knowledge records via valid paths. | Any rendered relationship is a reopened founder decision, not a feature. |
| `B-1` | Reason one for the absence, distinct from reason two: a relationship panel hands the player knowledge no valid path delivered. What A1 feels is A1's. | The payload carries unearned knowledge — a `B-1` breach even if the UI looks harmless. |
| `B-4` | Reason two, distinct from reason one: the system never authors the player's inner state; no trust/relationship meters. A meter overwrites the player's own reading with the engine's. | The one reading that is the player's stops being theirs. |
| `C-4` | Only creator/debug mode may ever show authoritative state — the sole legal omniscient view of a relationship. | An omniscient relationship view in play mode is `B-1` + `B-4` at once. |

The two reasons are easy to collapse into one and must not be (`digest/01_TOPIC_MAP.md` §WE-9): `B-1`
is about *unearned knowledge* (the payload), `B-4` is about *player interiority* (the authorship).
A fix that satisfies one can still violate the other.

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No relationship UI, of any kind.** The two distinct reasons above (`B-3`, `B-1`, `B-4`). Enforced
  in code: no page payload carries a relationship field (`tech.md` §Validation). The frontend law
  list restates it as its rule 6 — the founder's own reference mockups (Seren v2 trust slider, the
  companion's Relationships nav item) are **struck** (`digest/S11_frontend.md` topic 3).
- **No top-level Relationships Compendium section.** Rejected as written: *"A top-level Relationship
  section risks becoming a graph dashboard too early… MVP should stay readable, not managerial"*
  (parked-relationships PRD via `digest/S05_the_prds.md` topic 6). A later section is recorded as
  possible — perception-bound, never omniscient outside creator/debug — under conditions (large
  casts, multiple users, faction politics, family trees, institutional hierarchies) none of which
  hold today.
- **Banned surfaces, enumerated:** relationship graph · related-actors panel · relationship meter ·
  trust/fear numeric dashboard · social-network visualization · "also check" section
  (`digest/S05_the_prds.md` topic 6; register compliance rows for `B-3`/`B-4`,
  `docs/law/06_rules_register.md` §Compliance Audit).
- **No relationship write path.** Deliberate 0A stub, with its reason recorded in the code: the
  founding doc did not define how a single-entity mutation addresses a two-actor row (`SPEC-001`;
  `tech.md` §The write path that no-ops).
