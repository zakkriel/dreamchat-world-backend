# MVP Slice & Bridge Architecture

**Status:** Draft v1 — 2026-06-10 | **Owner:** TBD
**Closes:** Rules Register gap G-3. The last load-bearing document before Phase 0A build (per D-9, further docs after this require empirical evidence).
**Governed by:** Rules Register (`00_strategy/06`) — every contract rule below cites its register ID. On engine matters, `canon_engine/` wins.

---

## 1. Decisions this doc is built on (recorded 2026-06-10)

| Decision | Verdict |
|---|---|
| MVP slice 1 Compendium surfaces | **All four** (Actors, Locations, Artifacts+Carrying, Timeline) + the additions in §2 |
| Modules in MVP | **Stats + JRPG Battle demo** (matching the platform doc §10 recommendation; runtime skeleton implied) |
| Image generation liveness | **Staged: pre-generated at world creation → fully live, delivered asynchronously.** Async is the *mechanism* of live, not a separate maturity stage. Images never block the sync play path (D-8). |

## 2. MVP surface inventory

**Play surface (the default — C-1):** Scene Canvas · Scene Participants (C-10) · Conversation/Narration panel with streaming · Aux Context Sidebar with all four lenses (Current, Inspect, Intent, Known — C-3) · beat input with chain pushback (C-5/C-7) · Continue (C-6).

**Compendium (workspace layer):** Timeline · Actors · Locations · Artifacts + Carrying overlay · Known World overview · Report/Corrections. No Relationships section (B-3).

**Added as missing (per slice-1 mandate "anything missing"):**
- **World entry & creation flow** — create/enter a world, seed cast and places, pick experience style. Without it there is no product, only a demo.
- **Return-to-world continuity** — re-entry lands the user in a readable current scene with "what has changed" context (the core emotional promise; C-1).
- **Corrections UX** — invisible-by-default window, Continue implicitly accepts, explicit lock available (C-11 / engine ADR-011).
- **Settings** — interpretation confidence (C-8), content/style preferences.
- **World Graph Inspector** — creator/debug only (`30_architecture/world_graph_inspector_debug_view.md`).

**Explicitly out of MVP:** multiplayer · module marketplace · third-party modules · relationship UI (B-3) · user notes (parked) · deep retroactive rewrite (ADR-016) · public discovery/monetization surfaces (governance rails in §6 still apply to private generation).

## 3. Slice plan — frontend slices mapped to engine phases

Engine phases are fixed (canon_engine doc 07 §6). FE slices attach to them; FE never waits idle.

| FE Slice | Runs against | Delivers | Gate to exit |
|---|---|---|---|
| **S0 — Shell & read models** | Engine 0A/0B (Mara substrate, seeded data) | App shell, design system, world entry flow, read-only Compendium pages rendered from seeded projections, Graph Inspector | All four page types render correct perception-bound payloads from Mara/Seren seed data; I-3 spot audit on payloads |
| **S1 — Play loop** | Engine Phase 1 (fast path, scripted-driver-passing) | Live beat loop: deterministic actions, narration streaming, Current+Known lenses, Timeline & Actor pages live-updating, Carrying overlay | A human can play the Mara slice end-to-end through the UI; transcript deletion loses nothing (engine Phase 1 gate, human driver) |
| **S2 — Free-form & corrections** | Engine Phase 2 (canonization pipeline) | Free-form input, Intent lens, correction window UX (invisible default + lock), Report issue from every page, Inspect lens | Free-form Mara slice through the UI; correction round-trip works; I-8 holds with real users clicking |
| **S3 — Epistemic depth** | Engine Phase 3 | Rumor/told/overheard sourcing fully visible in Collected Knowledge; Locations & Artifacts pages at full fidelity; live epistemic enum rendering | Live rumor chain visible in a page with correct sourcing; planted secret never renders (I-3 at scale) |
| **S4 — Living world & modules** | Engine Phase 4 + Module workstream | Backstage update surfacing, decay language live (Decay mechanic), **Stats + Battle module UI** | Seren golden through the UI; one full battle demo writing canon only via proposals |

**Module workstream** (parallel, lands in S4): runtime skeleton → Stats module (entity attributes via module state, JSONB + `schema_version` per D-4) → Battle demo. Battle requires the Phase 2 proposal pipeline to exist, hence S4. Module mechanics never enter the Core (D-2, GA-4).

## 4. The Bridge contract — Frontend ⇄ World Backend

**Principles (non-negotiable):** the FE owns presentation only (D-7). The FE **never receives canon rows** — every payload is a perception-bound projection assembled server-side (B-1; I-3 is enforced in the gate *and* in context assembly *and* at the API boundary — defense in depth). All payloads carry `schema_version`. Time travels as `tick` (ordering) + `display_label` (rendering); wall-clock never crosses the boundary into UI (B-5).

### 4.1 Read side — projection endpoints

```
GET /worlds/{w}/compendium/actors                 → list (perceived names, syntheses, importance order)
GET /worlds/{w}/compendium/actors/{id}/page       → Actor Page Projection (PRD §Actors; no relationship field — B-3)
GET /worlds/{w}/compendium/locations/{id}/page    → Location Page Projection (one "Part of" — C-12; known areas only)
GET /worlds/{w}/compendium/artifacts/{id}/page    → Artifact Page Projection (dossier order; links = provenance only — B-10)
GET /worlds/{w}/compendium/timeline?before_tick=… → Timeline records (point to perception versions, never canon)
GET /worlds/{w}/carrying                          → Carry States of the user-controlled Actor only
GET /worlds/{w}/scene/current                     → scene canvas state, participants, aux Current payload
```
Exact JSON shapes = the Page Projection schemas in the Compendium PRDs; field-level binding lands in the **PRD→DDL mapping appendix (G-2 — next doc after PRD validation)**. Every knowledge item ships `epistemic_type` (canonical enum), `occurred_at_tick`, `display_label`, source metadata, and decay flags ("unconfirmed since tick").

### 4.2 Write side — the beat loop

```
POST /worlds/{w}/beats        body: free text or structured action
  → streamed response (SSE):
     interpretation frame      (Intent lens payload; confidence per C-8)
     narration tokens          (stream)
     scene delta               (participants, canvas state)
     aux delta                 (Current lens refresh)
     pushback frame            (if chain interrupted — C-7)
     correction-window frame   (window id + state; UI stays invisible per C-11)
POST /worlds/{w}/beats/continue                    (C-6: advances the moment, never fast-forwards)
POST /worlds/{w}/corrections/{window}/lock|report
```
A beat may yield zero/one/many canonical changes (C-5); the FE never assumes 1:1. The FE submits intent and renders results — it never decides outcomes (D-1/D-7).

### 4.3 Async channel (SSE/WebSocket, one per world session)

Events: `image.ready` (asset swap-in) · `projection.updated` (page invalidation → refetch) · `backstage.applied` (S4) · `correction.window_closed`. The sync path stays small (D-8); everything here is eventually-rendered.

### 4.4 Module UI slots

Module manifests declare UI contributions to **named slots only**: `aux.panel` (e.g., battle state), `scene.overlay` (e.g., turn order), `action.bar` (module actions). Module UI calls module action endpoints → proposals → validation gate (D-1). Module UI receives module-scoped state, never raw world internals. The FE renders unknown modules generically from the manifest — it never hardcodes battle semantics (D-2).

## 5. Image Platform integration (staged per §1)

**Stage A — launch baseline (pre-generated):** world creation triggers an **asset pack job** (existing sprite/asset-pack pipeline, `image_platform/` docs): portraits for seeded Actors, scene art for seeded Locations, a typed placeholder set (actor/location/artifact silhouettes with atmosphere). Manual regenerate from creator tools.

**Stage B — live, async:** the world backend emits image jobs on qualifying triggers; assets swap in via `image.ready`. Triggers (initial set, tune with data):
- a dynamically created Actor crosses an importance threshold (background → meaningful)
- a Location is first visited / an Artifact first inspected
- a scene change with no suitable cached art

**Rules:** every generation request passes governance classification *before* dispatch (E-1; the Image Platform never decides eligibility — D-3). Jobs are queued, budgeted per world, and never block narration (D-8). **Portrait regeneration policy (decides the Actors PRD open question):** portraits regenerate only on creator action or a *canonical* appearance change (scarred, aged, transformed) — never on perception change. What changes with the user's understanding is the *text*, not the face; identity stability is part of continuity.

## 6. Open items → engineering (`technical_questions`)

1. Streaming transport: SSE vs WebSocket for §4.2/4.3 (one choice for both).
2. Projection caching/invalidation strategy at the API layer (engine doc 03 §7 governs the DB side).
3. Image job budget defaults per world; placeholder asset pack scope.
4. Auth/session model for world ownership (single-user MVP, but governance classification needs identity).
5. Payload `schema_version` negotiation between FE and backend releases.

## 7. What this doc does *not* reopen

Engine architecture (frozen) · module runtime internals (post-0A editorial per D-9) · marketplace · multiplayer. Changes to this doc after Phase 0A starts require evidence, same as everything else.
