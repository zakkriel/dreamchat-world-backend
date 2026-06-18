# DreamChat Rules Register

**Status:** v3 — **FULLY VALIDATED 2026-06-10.** All rules in Parts B–F/GA approved (blanket approval for unamended rules; B-7 and B-10 confirmed individually). This register is now the standing law of the docs and the compliance checklist.
**Purpose:** Every normative rule in the documentation corpus, in one place, with an ID. Once you confirm/amend each Part, this register becomes the checklist for the compliance audit of all docs (and later, of code/CI).
**How to validate:** Part A is frozen (challenge only via new engine ADR). For Parts B–F, mark each rule **Confirm / Amend / Drop**.

Status legend: 🔒 Frozen (engine contract) · ✅ Accepted (validated by you, dated) · 🟡 Inherited (stated in docs, not yet explicitly validated)

---

## Part A — Engine rules (🔒 frozen; reference only)

**ADRs:** ADR-001 onward in `canon_engine/02_world_state_adrs.md`. The load-bearing ones for everything downstream:
- **ADR-001** Canon events are the immutable source of truth · **ADR-005** Perception separate from canon; data-layer isolation absolute · **ADR-006** Three time axes; invalidation never deletion · **ADR-007** Causality = reified bundles, not binary edges · **ADR-008** Provenance always, bundles selectively (used from Phase 4) · **ADR-009** LLM proposes, deterministic gate decides · **ADR-016** Corrections present-forward · **ADR-017** Propagation bounded; Traversal Matrix enforced · **ADR-020** Narrator gets no omniscient pass · **ADR-021/030** World Clock owns in-world time; fictional time is logical tick + label · **ADR-026** Replay invariance = domain equivalence

**Invariants (CI-enforced):** I-1 replay invariance · I-2 universal provenance · I-3 no hidden-canon leakage · I-4 causal acyclicity · I-5 traversal safety · I-6 canonization threshold (chatter ≠ canon) · I-7 projections written only by the maintainer · I-8 window closure · I-9 epistemic temporal sanity · I-10 claim non-abandonment

**Hard rules:** G3 — bundle inputs reference durable records only, never free text or live projection reads · forbidden propagation relations: `temporally_before`, `same_scene_as`, `references_*` · `affects_probability_of` never cascades as causality.

---

## Part B — Epistemic & domain rules (cross-cutting product law)

| ID | Rule | Source | Status |
|---|---|---|---|
| B-1 | **Perception-bound surfaces:** every user-facing page renders from the holder's perception, never raw canon; hidden truth is absent from the payload, not hidden by UI. | All Compendium PRDs; engine I-3 | ✅ 2026-06-10 (Actors AC#1) |
| B-2 | **Valid-path knowledge:** unknown facts enter a perspective's knowledge only through valid in-world paths — observation, told, record, broadcast, inference, propagation, **or common knowledge** (presumed ambient/cultural knowledge of the world; see Glossary). | UX loop §AUX; Compendium PRDs; amended 2026-06-10 | ✅ 2026-06-10 |
| B-3 | **No relationship UI (MVP):** relationships are modeled internally but never rendered as UI — no panel, field, synthesis, or label. Relationship-flavored information surfaces only as ordinary sourced knowledge records via valid paths. | Product decision (rev 2) | ✅ 2026-06-10 |
| B-4 | **Player interiority:** the system never authors the player character's inner state; no trust/relationship meters. | Product decision; Glossary §4.4 | ✅ 2026-06-10 |
| B-5 | **Append-only time:** canon and perception are forward-moving; no mutable `updated_at` domain fields; tick + label only; three time axes never conflated. | `00_time_and_mutability_rules.md`; ADR-006/030 | ✅ 2026-06-10 |
| B-6 | **Contradiction lives in perception, never in canon.** Perception-level contradictions are preserved (source, timing, uncertainty stay distinguishable). Canon is never contradictory: conflicting claims resolve at the validation gate, and **the latest accepted event supersedes** (prior state invalidated, never deleted — ADR-006). "NPC dead" vs "NPC alive" can coexist as beliefs, never as canon. | Amended 2026-06-10; ADR-006/009 | ✅ 2026-06-10 |
| B-7 | **Knowledge transfer never copies memory:** a propagated perception is a *new* record with its own epistemic type, teller, and confidence — the receiver knows *that they were told*, not *what the witness saw*. (Forbids the copy-the-record implementation shortcut that makes everyone a witness.) | Parked Concepts §1; rephrased 2026-06-10 | ✅ 2026-06-10 |
| B-9 | **Synthesis honesty:** page syntheses derive deterministically from stored perception versions — no regeneration drift on reload. | Actors PRD AC#3 | ✅ 2026-06-10 |
| B-10 | **`linked_*` fields are provenance/reference, never causality.** Compendium link fields mean "derived from / refers to"; causality exists exclusively in validated causal bundles (ADR-007/008). No shadow causal graph via link fields. | Causality review 2026-06-10 | ✅ 2026-06-10 — clarified: user-facing surfaces never link to canon directly (they link perception-layer records); internal provenance references canon; causality only via validated bundles |
| B-11 | **Event-driven cognition:** an NPC's beliefs and appraisals update only when the actor *perceives* something, never on a free-running idle loop. Every belief-update carries a perceptual trigger and provenance (B-10 `linked_*`). | Chunk-5 play-loop architecture notes (§5); SPEC-012 | ✅ 2026-06-17 — design-intent amendment (precedent D-3: declares a boundary ahead of the build; NOT an engine-behavior claim, so not gated by D-9) |

## Part C — UX rules

| ID | Rule | Source | Status |
|---|---|---|---|
| C-1 | Play-first: entering the product feels like returning to a world, not opening a chatbot or dashboard. Workspace depth is optional. | UX loop §1 | ✅ 2026-06-10 (blanket approval) |
| C-2 | Visuals make the scene readable; prose makes the scene deep. | UX loop §2.1 | ✅ 2026-06-10 (blanket approval) |
| C-3 | Aux follows user attention, not database structure; lenses (Current/Inspect/Intent/Known), no fixed genre taxonomies. | Aux design | ✅ 2026-06-10 (blanket approval) |
| C-4 | Play mode shows the known/perceived world; only creator/debug mode may show authoritative state. | UX loop §3 | ✅ 2026-06-10 (blanket approval) |
| C-5 | A Beat may produce zero, one, or several canonical changes — visible messages never map 1:1 to canon. | UX loop §4.2; ADR-010 | ✅ 2026-06-10 (blanket approval) |
| C-6 | Continue advances the current moment; it does not fast-forward the world. | UX loop §3.4 | ✅ 2026-06-10 (blanket approval) |
| C-7 | World pushback: the longer/more consequential an action chain, the more chances the world has to react, interrupt, resist. Full intent understood ≠ full chain guaranteed. | UX loop §4.2–4.3 | ✅ 2026-06-10 (blanket approval) |
| C-8 | Interpretation confidence is a user-control setting, not hardcoded behavior. | UX loop §4.6 | ✅ 2026-06-10 (blanket approval) |
| C-9 | Scene transitions are internal mechanics; the user experience is seamless. | UX loop §4.7 | ✅ 2026-06-10 (blanket approval) |
| C-10 | Scene Participants shows only beings with presence/agency — never objects, locations, documents as avatars. | UX loop §2.2 | ✅ 2026-06-10 (blanket approval) |
| C-11 | Correction UX is invisible by default — no pending/approval UI; Continue implicitly accepts (explicit lock available). | Engine ADR-011; index governance | 🔒 |
| C-12 | One hierarchy expression per Location page ("Part of" subtitle); no duplicated breadcrumb + tree + panel. | Locations PRD | ✅ 2026-06-10 (blanket approval) |

## Part GA — Genre-agnosticism rules (extracted per validation round 1; absorbs C-13 and old F-2)

| ID | Rule | Source | Status |
|---|---|---|---|
| GA-1 | Interaction structure is stable across genres; only content and language adapt to the world. | UX loop §1 | ✅ 2026-06-10 |
| GA-2 | **System terms must be genre-agnostic.** Genre-specific concepts (quests, mana, factions, relics, …) exist only as world or module content, never as core system vocabulary. Test: the term must make sense in a sci-fi thriller, a workplace drama, and a horror story. | Glossary §5 (rewritten) | ✅ 2026-06-10 |
| GA-3 | Surfaces speak the language of the current world and situation — no fixed genre taxonomies (no hardcoded Rumors/Combat/Quest sections). | Aux design | ✅ 2026-06-10 |
| GA-4 | Genre-specific mechanics (HP, sanity, spells, …) live in modules, never in the Core. | D-2 corollary; module docs | ✅ 2026-06-10 |

## Part D — Architecture & platform rules

| ID | Rule | Source | Status |
|---|---|---|---|
| D-1 | Nothing mutates canon directly — modules and AI produce proposals; the Core validates, commits, rejects, converts. | ADR-009; module docs | 🔒 |
| D-2 | The Core owns the substrate (world, entities, scenes, canon, knowledge, corrections, module registry); modules own their mechanics (HP, mana, sanity…). Core never learns module semantics. | Module architecture docs | ✅ 2026-06-10 (blanket approval) |
| D-3 | The Image Platform is a separate service and never owns world truth; it receives only classified/authorized generation requests. | Platform/09; ADR-P016 | ✅ 2026-06-10 (blanket approval) |
| D-4 | Engine DDL is the only core schema; JSONB is for module-owned/evolving state, always with `schema_version` + runtime validation. | Decision 2026-06-10; ADR-P001 banner | ✅ 2026-06-10 |
| D-5 | ADR numbering: engine owns plain ADR-001 onward (doc 02 only); platform/product ADRs use `ADR-P###`. Decisions change only via superseding ADR. | MASTER_INDEX | ✅ 2026-06-10 |
| D-6 | Git `/docs` is the source of truth for layers 10/20/30; Drive is drafting space for layer 00 and WIP; promotion is one-way (export → commit → banner). | Decision 2026-06-10 | ✅ 2026-06-10 |
| D-7 | Frontend owns presentation only — never world truth, canon decisions, knowledge checks, or correction validity. | Modular architecture §2.1 | ✅ 2026-06-10 (blanket approval) |
| D-8 | Synchronous path stays small (parse → read → route → validate → stream); summarization, reflection, backstage, images, evals run async. | Platform/03 §7; engine phase plan | ✅ 2026-06-10 (blanket approval) |
| D-9 | Document changes need empirical evidence once code runs — no further doc convergence rounds without Phase 0–2 data. | Engine index governance | 🔒 |
| D-10 | **Agent-agnostic repo governance.** No repo may depend on any specific coding agent's conventions or tooling. The canonical agent entry point is `AGENTS.md` (open convention) at each repo root; tool-specific files (`CLAUDE.md`, etc.) may exist **only** as one-line pointers to `AGENTS.md`. Agent-specific tooling (e.g. user-level skills) is personal convenience, never a repo dependency. The law itself stays in the backend `/docs` (D-6) — `AGENTS.md` references it, never copies it. | Founder decision 2026-06-12 | ✅ 2026-06-12 — founder-ratified |
| D-11 | **Record/derive discipline.** Coupled quantities are *derived* from a recorded generating structure, never stored as independent facts; only genuinely independent facts may be recorded directly. Distances derive from coordinates; independent process/craft durations may be recorded as-is. | Chunk-5 play-loop architecture notes (§11); SPEC-018 | ✅ 2026-06-17 — design-intent amendment (precedent D-3: declares a boundary ahead of the build; NOT an engine-behavior claim, so not gated by D-9) |
| D-12 | **Spatial is a bounded subsystem — its own engine.** It owns geometry only: positions, and derived distance / route-time. It never owns world-truth beyond geometry. Distance (how far) is distinct from reachability (whether one can go); reachability stays with move-validity (SPEC-017). | Chunk-5 play-loop architecture notes (§12); SPEC-018 | ✅ 2026-06-17 — design-intent amendment (precedent D-3: declares a boundary ahead of the build; NOT an engine-behavior claim, so not gated by D-9) |
| D-13 | **Model-agnostic per-seat LLM routing**  Every play-loop LLM call runs through a per-seat driver, bound by config and freely swappable, with no provider SDK in the Core; a seat binds a model only if the driver satisfies that seat's declared capabilities (canon-emitting seats — decompose, resolve — require constrained-decoding of the closed event vocabulary). The quarantine (perception-bound, propose-only, never canon authority) holds per seat regardless of bound model — model choice affects output quality, never canon integrity or perception isolation.| Bridge ADR (Leg-2, TBD#); doc 07; cf. ADR-009, ADR-020, D-3, D-10 | Proposed — ✅ at Leg-2 gate (D-9) |

## Part E — Governance rules

| ID | Rule | Source | Status |
|---|---|---|---|
| E-1 | Private single-user worlds and public-governed content are distinct regimes; classification (private/shareable/public/monetizable/media-eligible) happens in the core platform **before** any media generation request. | ADR-P016 + governance PRD | ✅ 2026-06-10 (blanket approval) |
| E-2 | Mature fictional content can exist in private worlds within legal/safety bounds; public distribution, discovery, and monetization apply stricter eligibility. | ADR-P016 | ✅ 2026-06-10 (blanket approval) |

## Part F — Terminology rules

| ID | Rule | Source | Status |
|---|---|---|---|
| F-1 | Ubiquitous language per Glossary §2; `entity` legal only inside the engine context as supertype. | Glossary | ✅ 2026-06-10 — Glossary signed off |
| F-2 | UI copy expresses uncertainty in story language; users never see entity/canon/projection/perception-record vocabulary. | Glossary §4.3 | ✅ 2026-06-10 (blanket approval) |

---

## Mechanics register (not rules — definitions live in the Glossary)

| Mechanic | One-liner |
|---|---|
| **Decay** | A known state uncorroborated long enough that current validity is uncertain → lowers confidence, drives "last known…" language. Never a visibility filter. (Moved from old rule B-8 per validation round 1.) |

## Gaps surfaced by this review (not rules yet — need decisions)

| ID | Gap |
|---|---|
| G-1 | ✅ **Resolved:** minimal read-only **World Graph Inspector** spec added (`30_architecture/world_graph_inspector_debug_view.md`) — provenance up/downstream + bundles from Phase 4, creator/debug only, no engine changes. |
| G-2 | **PRD→DDL mapping appendix** — sequenced: written after all Compendium PRDs are validated. |
| G-3 | ✅ **Resolved:** `30_architecture/mvp_slice_and_bridge.md` — slices S0–S4 vs engine phases, projection/beat API contract, module UI slots, staged image pipeline. |
| G-4 | ✅ **Resolved:** engine `epistemic_type` is canonical. Mapping doc `10_prds/compendium/01_epistemic_type_canonical_enum.md` created; all PRD `source_type` enums patched to reference it. |

## Process once validated

1. You return this register with Confirm/Amend/Drop per rule (Parts B–F).
2. Compliance audit: every active doc checked against the validated register; violations fixed or bannered; audit results appended here.
3. The register becomes `00_strategy/06_rules_register.md` — the standing law of the docs, and later the seed for CI checks (Part A already is).


---

## Compliance Audit — round 1 (2026-06-10)

Checked: all active docs in 10_prds, 20_design_ux, plus glossary cross-references. Engine set excluded (frozen; it *defines* Part A).

| Violation found | Rule | Resolution |
|---|---|---|
| "Possessions" as Compendium category + `linked_possessions` fields across all 5 Compendium PRDs | GA-2 / F-1 | ✅ Renamed to Artifacts / `linked_artifacts` throughout |
| Core UX loop doc lists "Relationships" in workspace nav; uses possessions/objects naming; pre-dates time + enum rules | B-3, GA-2, B-5, G-4 | ✅ Errata banner prepended (structural corrections enumerated; 850-line doc not silently rewritten) |
| `source_type` drifted enums in PRDs | G-4 | ✅ Patched to canonical epistemic enum (round 1 of validation) |
| Mutable `*_at_in_world_time` fields | B-5 | ✅ Patched all PRDs (earlier today) |
| Relationship panel in Actors PRD + v2 mock (slider, panel, Add note) | B-3, B-4 | ✅ Removed; mock exceptions enumerated in Actors AC#10 |
| Timeline mock nav ("Entities", Possessions, Relationships) | GA-2, B-3, F-1 | ✅ Listed as mock exceptions in Timeline PRD AC#7 |

**Result: zero open violations in active docs.** Remaining known debt is in 🟡-directional architecture docs (`platform/`, `modules/`), already bannered and scheduled for post-Phase-0A editorial (D-9: no doc convergence without empirical evidence).

**Formalization status:** all four Compendium PRDs (Actors, Locations, Artifacts+Carrying, Timeline/Perception) now carry full PRD wrappers with register-mapped acceptance criteria.
