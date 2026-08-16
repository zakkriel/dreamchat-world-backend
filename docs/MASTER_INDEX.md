# DreamChat Documentation — Master Index

**Date:** 2026-06-19 (rev 9)

**Rev 9 — FE architecture landed (docs-only):** the chunk-6 FE Architecture working-session decisions
(2026-06-19) ratified into the law. Two register rules added — **D-14** (FE rendering model — one
engine, a catalog of kinds; attaches to D-2 / D-7 / GA-3) and **D-15** (theme architecture — neutral
skeleton + swappable skins; reinforces GA-3, cites D-3 / D-4 / D-8). One platform ADR — **ADR-P019**
(FE rendering + theme architecture, A1+A2) with the chunk-6 mockups as evidence. Six FE seams filed in
`open-spec-items.md` — **SPEC-019** (world theme-token field), **SPEC-020** (configurable backend API
base), **SPEC-021** (BE CORS for the FE origin), **SPEC-022** (dynamic multi-world id), **SPEC-023**
(app shell with named slots + Aux docked↔full-screen), **SPEC-024** (Electron-wrappable delivery
target). Three deferred workstreams added as planned stubs below: B1 identity/auth, B2 world-creation,
B3 module-architecture. Docs-only; no engine canon, invariant, or Master DDL touched.

**Rev 8:** Validation Ladder principle added to the implementation playbook (§0.5): every 2–3 chunks must answer a *product* question with a falsification criterion; a product "no" blocks the ladder even with green CI. Q1–Q5 per product owner (replay / trust / transcript-independence / immersive correction / aliveness); Q6–Q7 added pending confirmation (gameplay without losing itself / user-created worlds believable). Chunk 13 clarified: image platform is external and pre-existing — integration only; V2 addendum becomes a parallel sub-chunk in the platform's repo if not yet implemented.

**Rev 7:** `30_architecture/implementation_playbook_superpowers.md` added — 14-chunk build ladder (each chunk: build / personally-testable / CI gate), mapped to engine phases 0A–4 and Bridge slices S0–S4; Superpowers loop per chunk; anti-drift rules; first-session script. Chunks 1–4 are LLM-free by design (ADR-029).

**Rev 6 — MVP Slice & Bridge:**
24. **`30_architecture/mvp_slice_and_bridge.md` added (G-3 closed).** Decisions: all four Compendium surfaces in MVP (+ world entry, return continuity, corrections UX, settings, debug inspector); Stats + JRPG Battle demo modules (landing in FE slice S4); images staged pre-generated → live-async. FE slices S0–S4 mapped to engine phases 0A–4. Per D-9, this is the last load-bearing doc before Phase 0A build; remaining doc work is the PRD→DDL mapping appendix (G-2, after PRD validation) and evidence-driven amendments only.
25. **Portrait regeneration policy decided** (closes Actors PRD open question): canonical appearance changes or creator action only — never perception changes.

**Rev 5 — register fully validated + compliance audit complete:**
21. **Rules Register v3: FULLY VALIDATED** (B-7, B-10 confirmed; blanket approval for the rest). It is now the standing law and CI seed.
22. **Compliance audit round 1: zero open violations** in active docs (results appended to the register). Possessions→Artifacts renamed everywhere; UX loop doc carries an errata banner.
23. **All four Compendium PRDs formalized** with register-mapped acceptance criteria. PRD validation is the gate for the PRD→DDL mapping appendix (G-2), then the MVP Slice / Bridge doc (G-3).

**Decisions recorded (rev 4 — Rules Register validation round 1):**
12. **Rules Register adopted** as `00_strategy/06_rules_register.md` (v2). All rules ✅ except B-7/B-10 (explained, awaiting verdict).
13. **B-3 hardened:** no relationship UI in MVP at all — removed from Actors PRD, Glossary, parked_relationships.
14. **B-6 split:** contradictions preserved in *perception* only; canon resolves by latest accepted event (supersession, never deletion).
15. **B-2 amended:** common knowledge added as a valid knowledge path (Glossary term added).
16. **Decay** reclassified from rule to mechanic; defined in Glossary.
17. **Genre-agnosticism** extracted as its own rule set (GA-1…4), absorbing C-13 and the old banned-terms list (now a principle).
18. **Canonical epistemic enum:** engine `epistemic_type` wins; mapping doc added; all PRD enums patched (G-4 resolved).
19. **World Graph Inspector** minimal debug spec added (G-1 resolved): `30_architecture/world_graph_inspector_debug_view.md`.
20. **Sequencing:** PRD→DDL mapping appendix written only after all Compendium PRDs are validated (G-2).

**Decisions recorded 2026-06-10 (rev 3):**
7. **Time model fixed everywhere.** All `*_at_in_world_time` / `last_updated_at` mutable fields removed across every Compendium PRD; replaced with tick-based, append-only, derived fields. Normative doc: `10_prds/compendium/00_time_and_mutability_rules.md`.
8. **Relationship gating.** "Relationship to you" is never a default panel or narrator clue — rendered only when qualifying perception records exist (valid in-world information path). Trust slider in the v2 mock: rejected.
9. **Player interiority rule** added to Glossary: the system never authors the player character's inner state.
10. **"Add note" is out of MVP** (parked as future `user_note` perception records).
11. **Actor portraits are an MVP critical-path Image Platform dependency** (from mock review) — flows into the Bridge doc.

**Decisions recorded 2026-06-10:**
1. **MVP, not PoC.** The first release ships a real UI with the Compendium as the value showcase. Compendium PRDs therefore need acceptance criteria now; frontend track runs in parallel with engine Phase 0A.
2. **Engine DDL wins.** ADR-P001's core tables are superseded by the Canon Engine Master DDL; the JSONB-for-module-state principle survives (see banner in ADR-P001).
3. **Source-of-truth rule:** Git `/docs` owns layers 10/20/30. Drive is the drafting space for layer 00 and work-in-progress; promotion = export to markdown + commit + banner the Drive doc with the Git path.
4. **Supersession banners applied** to `platform/04`, `platform/06`, `ADR-P001` (with salvage notes for what survives).
5. **Glossary added** as `00_strategy/05_glossary_ubiquitous_language.md` (Draft — pending sign-off).
6. **Recorded future work — the missing bridge:** there is no doc yet defining the contract between Frontend ⇄ World Backend ⇄ Module UI surfaces ⇄ Image Platform (request/response shapes, streaming, projection payloads). Engine doc 13 covers only the engine spine. This becomes the next architecture doc after the MVP slice is defined.
**Purpose:** Single entry point for all DreamChat product, design, and technical documentation. Every doc below has a status. Nothing was deleted — superseded material lives in `90_archive/`.

**Status legend:** ✅ Active (current truth) · 🔒 Frozen (build contract — change only via ADR) · 🟡 Directional (useful, not yet reconciled with the frozen engine set) · 🅿️ Parked · 🗄️ Archived (superseded, kept as provenance)

---

## Layer 0 — Strategy (`00_strategy/`)

| Doc | Status | Notes |
|---|---|---|
| `01_product_vision_and_promise.md` | ✅ Active | Genre-agnostic wording (Google Doc version). Old fantasy-specific `.md` archived. |
| `02_poc_scope_and_success_criteria.md` | ✅ Active | Defines PoC question, Layer 1/2 validation, world shape, exclusions. |
| `03_market_research_rpg.md` | ✅ Active (reference) | Market landscape as of 2026-06-04. |
| `05_glossary_ubiquitous_language.md` | ✅ Draft — pending sign-off | Ubiquitous language + bounded contexts. Governs terminology once accepted. |
| `04_parked_product_concepts.md` | 🅿️ Parked | Social Memory Propagation, World Relevance Score, Structural Depth Model. Each names the future PRD it belongs to. |

## Layer 1 — PRDs (`10_prds/`)

| Doc | Status | Notes |
|---|---|---|
| `compendium/prd_compendium_actors.md` | ✅ Active — **formal PRD (template for the others)** | This is **v2** (Relationships removed as top-level section). v1 archived. |
| `compendium/prd_compendium_locations.md` | ✅ Active | |
| `compendium/prd_compendium_artifacts_and_carrying.md` | ✅ Active | Artifacts (Compendium) vs Carrying (overlay) split. |
| `compendium/prd_timeline_and_perception.md` | ✅ Active | Timeline links to Perception versions, never to Canon directly. |
| `compendium/parked_relationships.md` | 🅿️ Parked | Modeled internally; surfaced inside Actor pages only in MVP. |
| `prd_private_public_content_governance.md` | ✅ Active | Pairs with ADR-P016. |
| `prd_visual_asset_packs_and_sprite_sheets.md` | ✅ Active | Image Platform V2 addendum. |
| `prd_world_creation.md` | ✅ Active | Writes the `B2 — World creation` stub below: a brief → two lanes (Fast / Custom interview) → an LLM-authored, engine-committed playable world. Cites B-1, B-2, B-4, B-5, GA-2, GA-3, D-1, D-11, E-1, I-2, I-3, I-9. |
| Canon Event Engine PRD | 🔒 Frozen | Lives in `30_architecture/canon_engine/08_prd_canon_event_engine.md` — kept inside the frozen set deliberately. |

## Layer 2 — Design / UX (`20_design_ux/`)

| Doc | Status | Notes |
|---|---|---|
| `core_ux_loop_and_aux_sidebar.md` | ✅ Active | The `aux_updated` version — verified to contain the full Aux Sidebar design merged in. Three older versions archived. |
| `mockups/` | ✅ Reference | Renamed by content. `mock_compendium_actor_seren_v1_superseded.png` shows the old nav (Relationships top-level) — keep only for history. `unsorted_concept_art/` needs manual naming. |

## Layer 3 — Architecture (`30_architecture/`)

### `canon_engine/` — 🔒 **FROZEN BUILD CONTRACT (v4.1)**
The 14-doc world-state set (00–13). Its own `00_INDEX.md` is authoritative for the engine: doctrine, ADR-001 onward, master DDL, pipeline/resolution/assembly specs, invariants, phase plan 0A→4. **Per its own governance: no architecture debate outside doc 02; next artifact is a green I-1 on a deployed Phase 0A schema.** Nothing in this folder was touched.

### `platform/` — 🟡 Directional (Jun 5 generation, pre-dates the frozen engine set)
The 8-doc architecture set (platform shape, world core, modules, memory/canon, AI orchestration, frontend, image integration, marketplace) + its confidence README.
**Reconciliation needed:** docs 04 (world core) and 06 (memory/canon/timeline) are now substantially superseded by `canon_engine/`. Docs 03, 07, 08, 09, 10 remain the best available description of their areas. Treat as directional until each is either re-validated against the engine set or rewritten.

### `modules/` — 🟡 Directional, overlapping
`modular_architecture_world_engine.md` and `plug_and_play_module_architecture.md` cover the same ground at different depth (and overlap with `platform/05`). **Action:** merge into one module-architecture doc + extract the decision ("modular monolith + registry + capability contracts + proposal/validation/commit") as a platform ADR. Note the engine set already enforces the proposal→validation→commit rule (ADR-009), so the module docs must defer to it.

### `adr/` — Platform ADR series (prefix **ADR-P**)
| ADR | Status | Notes |
|---|---|---|
| `ADR-P001_database_strategy_postgres_jsonb.md` | 🗄️ Superseded in part (bannered) | Postgres-first is reaffirmed by engine ADR-003; but its proposed core tables (`entity`, `relationship_edge`, …) conflict with the engine Master DDL (doc 03). **The engine DDL wins.** JSONB-for-module-state guidance remains valid. Needs a rewrite or formal supersession note. |
| `ADR-P016_private_vs_public_world_governance.md` | ✅ Active | Platform-level; no engine conflict. JSON schemas included alongside. |
| `ADR-P017_backend_application_language_go.md` | 🟡 Proposed | World backend application/transport tier = Go (Chunk 3 owns the decision). Does not touch frozen engine canon; perception filter stays in SQL (B-1, I-3). |
| `ADR-P018_llm_bridge_per_seat_model_routing.md` | ✅ Accepted (chunk-5 Leg-2 gate) | Model-agnostic per-seat LLM routing in the bridge layer (`core/api`); governing rule D-13. No provider SDK in the canon engine. |
| `ADR-P019_fe_rendering_and_theme_architecture.md` | ✅ Accepted (FE working session 2026-06-19) | FE rendering model (one engine, a catalog of kinds) + theme architecture (neutral skeleton + swappable skins). Governing rules D-14 / D-15; chunk-6 mockups as evidence. Does not touch frozen engine canon (D-7, GA-3). |

> ⚠️ **Numbering rule (new):** the engine owns plain `ADR-001` onward inside `canon_engine/02_world_state_adrs.md`. All platform/product ADRs use the `ADR-P###` prefix. The old loose `ADR_001` and governance `ADR-016` collided with engine numbers — resolved by the P-prefix.

### `image_platform/` — ✅ Active
Sprite-sheet pipeline architecture, API addendum, DB schema extension (SQL), JSON schema, runbook, implementation prompt, change summary. The Image Platform remains a separate service that never owns world truth (engine + platform docs agree).

## Planned docs (stubs — not yet written)

Deferred workstreams surfaced by the 2026-06-19 FE Architecture working session (chunk-6
pre-brainstorm, §B). Each is its **own chunk** and is **out of chunk 6**; they are filed here so the
work is tracked, not lost. Status = **planned / stub** until the doc lands, at which point the row
flips to ✅ and names the file — B2's PRD landed 2026-08-15 and its build work is still open.

| Planned doc | Kind | Status | Scope (what it must cover) |
|---|---|---|---|
| B1 — Identity / Auth | ADR or PRD | 🅿️ Planned / stub — **the seam landed, the model has not** | Still no account model and nothing knows who is calling. What changed (2026-08-08, SPEC-028): viewer identity is a fact the world records — `world.player_entity_id` — instead of "the actor named `Player`", so a world resolves its own player and `?viewer=` is a debug override rather than the only way in. That makes the swap to a real session a one-spot change in `ResolveViewer`, and `fn_world_directory()` is the single place a "worlds the caller may see" filter attaches. Still to cover: the account model; the **account→player binding**; **per-user isolation** (multi-world + Railway → user A must never read user B's worlds). **Now security-load-bearing: `POST /worlds` is unauthenticated**, so anyone who can reach the service can create a world — safe privately, not safe on a public origin. Coordinates with B2. |
| B2 — World creation | PRD / spec | ✅ **PRD written 2026-08-15 → `10_prds/prd_world_creation.md`** (pipeline still to build) | The two jobs have split cleanly. The **creation flow exists** (2026-08-08, SPEC-028): `POST /worlds` takes a name and theme tokens, writes the directory row and the world's operating defaults in one transaction, and authors **no entities** — a created world is real, listed and `playable:false`. The **seeding pipeline** (generate a coherent populated world — cast, places, objects, relationships, initial knowledge — from a prompt; wired to the engine for canon validity and the Image Platform for art) remains the iceberg and is likely **multiple chunks**. World *templates* are deliberately unbuilt: a starter scene is authored fiction, and the service must not learn what a world is "usually" like (GA-2/GA-3) — the PRD keeps that as a hard non-goal and authors from the brief instead. Sequencing holds — prove a hand-made world plays well before building the world-factory; that proof now exists. Art at genesis is a **non-goal** until core-side classification exists (E-1/D-3). The account→player binding **coordinates with B1**. |
| B3 — Module-architecture | render contract / manifest / trust | 🅿️ Planned / stub | Pins the three deferred module pieces: the catalog/fragment **render contract**; the **manifest** (how a module declares its slot + kinds + actions); the **trust model** (declarative-only now; WASM/server sandbox if/when 3rd-party code arrives). Engine + manifest + trust deferred to **S4**. Chunk 6 carries only the already-locked seams (named slots + interactive-component pattern — D-2, D-14, ADR-P019). |

## Archive (`90_archive/`)

| File | Superseded by |
|---|---|
| `actors_compendium_model_v1_…` | `10_prds/compendium/prd_compendium_actors.md` (v2) |
| `03_core_user_experience_loop_draft.md` / `…updated.md` / `core_ux_loop_gdoc_jun5_…` | `20_design_ux/core_ux_loop_and_aux_sidebar.md` |
| `aux_context_sidebar_standalone_…` | merged into the same |
| `product_vision_v1_OLD_fantasy_wording.md` | `00_strategy/01_…` (genericized) |
| `platform_architecture_loose_duplicate.md` | identical copy in `30_architecture/platform/` |
| `provenance/memory_strategy_adr_DC-MEM-001.md` | `canon_engine/` set (its index explicitly supersedes prior memory/RAG reports; MIRA lessons absorbed into ADR-003/014/018 and doc 06) |

---

## Known conflicts still to resolve (editorial work, not filing)

1. **Vocabulary drift.** Older docs say *Entity / Possession / Faction / Quest*; Compendium PRDs say *Actor / Artifact + Carrying / Group / Thread*; the engine set uses *entity* internally. Decide and write a one-page **Glossary** declaring: user-facing terms (Actor, Artifact, Carrying, Timeline, Known World) vs internal model terms (entity, canon event, perception, projection). Until then, every doc reader hits apparent contradictions that are really naming drift.
2. **Compendium PRDs ↔ engine schema mapping.** The PRD schemas (Actor Canon / Actor Perception / Perception Record) and the engine Master DDL (canon events, perceptions, projections) describe the same concepts from product and engine sides, but no doc maps PRD fields → DDL tables/projections. A short mapping appendix (probably in doc 03 or in each PRD) prevents the frontend/team from inventing a third model.
3. **`platform/04` and `platform/06`** must be marked superseded-in-part or rewritten against the engine set (see above).
4. **Module docs merge** (see `modules/` above).
5. **ADR-P001 rewrite** to formally defer its table designs to the engine DDL.
6. **PRD→DDL mapping appendix** must use the tick-based fields from `00_time_and_mutability_rules.md` (the schemas are patched; the mapping to engine Master DDL columns is still to write).
7. **Perception source_type enums** differ slightly across the Compendium PRDs (some include `confidence_metadata`, some don't; Artifacts adds fields Actors lacks). Normalize once in the Glossary/shared schema section.

## Suggested next actions

1. Adopt this tree as `/docs` in the repo (or mirror it as Drive folders).
2. Per the engine set's own governance: **stop writing docs, build Phase 0A** (`canon_engine/13_phase_0A_engine_contract.md`) and get I-1 green.
3. In parallel (product track): write the Glossary, then wrap the four Compendium models into formal PRD shape (goals, non-goals, acceptance criteria — content is ~80% there).
4. Editorial pass on `platform/` and `modules/` after Phase 0A learnings, not before (consistent with the engine index: "further document rounds without running code now have negative value").
