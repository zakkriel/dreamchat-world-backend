# Chunk 4 — Rest of the Read-only, Perception-bound Compendium (Design)

**Date:** 2026-06-14
**Status:** Design — pending founder approval, then spec review before write-plan.
**Builds on:** Chunk 3 spine (tag `chunk-3-actor-page-gate`) — `fn_visible_perceptions` (FILTER 1), `fn_perceived_name`, `fn_actor_page`, `perception_subject` (ADR-035), `world_genesis` CK names.
**Validation Ladder:** Q2 — *"Can the user inspect a world and trust it?"* (extended from the Actor page to the rest of the Compendium and, newly, to **existence**).
**Constraints:** one worktree · one plan · one PR **per repo leg** · TDD iron law (failing test first) · gate red → stop · LLM-free · live SQL joins only (no materialized projection — SPEC-009 unfired).
**Governing law:** Rules Register (`docs/00_strategy/06_rules_register.md`). Engine matters: `canon_engine/` wins. Rule IDs cited inline.

---

## 0. Judgment calls flagged for spec review

Called out so they can be vetoed here rather than discovered in code (same discipline as Chunk-3 §0).

1. **Shared knowledge-assembly core; `fn_actor_page` is re-pointed to it. — APPROVED (reviewer), with a hard condition.** Going-in position 1 says FILTER 1 *and* the about-ness FILTER 2 are shared, never reimplemented per page — only the page **envelope** differs. FILTER 2 for actors/locations/artifacts is *identical* (`perception_subject.entity_id = target`, genesis excluded). So this design extracts **only** the FILTER-2/collected-knowledge core into one `fn_collected_knowledge(world, viewer, target) → json` (FILTER 1 ∘ subject about-ness ∘ genesis exclusion ∘ grouping) and has the three page functions call it and add their (mostly-NULL-in-0A) type fields. **`fn_actor_page` is refactored to call the shared core.**
   - **HARD CONDITION (reviewer):** `45_actor_page_test.sql` must pass **UNCHANGED** — not one assertion edited to fit the refactor. That green *is* the proof the refactor is behaviour-preserving. **If the refactor requires touching those assertions, STOP — behaviour moved, and the refactor is wrong.**
   - `fn_visible_perceptions` (FILTER 1) and `fn_perceived_name` stay **literally untouched**. Only the collected-knowledge core is extracted.

2. **The artifact has no resolvable name in 0A, by design. — APPROVED (reviewer) as the minimal fixture.** The Sealed Note is deliberately **non-CK** (so its existence is private to Player — the index leak vehicle). Chunk-3's `fn_perceived_name` priority-1 branch (a viewer-held *divergent name* perception) is **deferred to Phase-1+** and stays deferred. Consequence: `fn_perceived_name(viewer, Note) = NULL` for **everyone**. The Artifact page and the index entry therefore show **existence-perceived, name-withheld**.
   - **Required & explicit in §5 (reviewer):** the index renders the known-but-unnamed Note via a **withheld-name label sourced from the perception layer** (`fn_perceived_name` → `NULL`) — it **NEVER** falls back to `entity_registry.canonical_name` (that canon name is the hidden truth — B-1 / I-3 / ADR-005), and it **NEVER** omits the Note (Player knows it exists). We do **not** activate the deferred name-divergence branch (Phase-1 scope creep).

3. **Timeline relevance lens = `holder = viewer`. — APPROVED (reviewer) as a FILTER-2 RELEVANCE choice on an unchanged, CK-inclusive FILTER 1.** FILTER 1 is composed first, **unchanged and CK-inclusive** (the safety wall is never narrowed). FILTER 2 then applies a **relevance** choice: it **excludes the universal-CK rows** (those held by the Common-Knowledge faction — ambient genesis identity and public-ledger rows), keeping the timeline a record of *"what this perspective experienced and learned"* (Timeline & Perception PRD §6, *One Canon Event, Multiple Perceptions* — holder-scoped histories).
   - **Phrase it as relevance, never as narrowing the wall.** Acquired-public perceptions (`epistemic_type='public'` but `holder = viewer`) **stay** on the timeline; only ambient genesis-CK rows (`holder = CK faction`) drop — and they drop because `holder ≠ viewer`, a relevance filter, not a safety filter.

---

## 1. Scope (locked with founder, 2026-06-14)

**IN — read-only, perception-bound, all over the Chunk-3 two-filter spine:**
- **Location page** — `GET /worlds/{w}/compendium/locations/{id}/page` → `location_page/1`
- **Artifact page** — `GET /worlds/{w}/compendium/artifacts/{id}/page` → `artifact_page/1`
- **Timeline** — `GET /worlds/{w}/compendium/timeline?before_tick=…` → `timeline/1`
- **Compendium index (per type)** — `GET /worlds/{w}/compendium/{actors|locations|artifacts}` → `compendium_index/1`, all backed by **one** `fn_compendium_index(world, viewer, kind)`.
- **Inline cross-links** — the existence-respecting **rule is defined**; payloads ship `inline_links: []` this chunk (no structured refs in seed content).

**OUT — frozen, each recorded in the ledger (§12) with a firing trigger:**
- **Carrying overlay** — play-facing (the user-controlled actor's current items); belongs to the play loop. *Trigger: Chunk 5 (play loop).*
- **Graph Inspector (debug)** — reads provenance/bundles; bundles are frozen SCOPE OUT and `provenance_edge` is read nowhere yet. *Trigger: when provenance/bundle reads are genuinely needed and bundles are unfrozen.*
- **Timeline perception-version evolution** (Timeline AC#4, v1→v2→v3) — the seed has zero versioned/superseded perceptions. *Trigger: the chunk that introduces perception versioning (Phase-1+).*
- **Location known-areas-inside + Key Actors** (Locations AC#3/#4) — need containment + co-location data the seed has none of. *Trigger: a seed/chunk with location hierarchy and co-location perceptions.*
- **Artifact holder/owner/access as derived Carry State** (Artifacts AC#3) — perception-bound or NULL only this chunk. *Trigger: Carry-state derivation (with Carrying, Chunk 5).*
- **Live inline cross-links** — *Trigger: the chunk that introduces structured entity references in perception content.*
- **Cross-type index counts / unified Known-World overview** — *Trigger: a future overview surface needing one round-trip → a thin aggregating façade over the same `fn_compendium_index`, never a parallel filter path.*
- Standing Chunk-1–4 freezes: beat loop / any LLM / writes / event creation (ADR-029) · bundles / causality / Seren / Phase-4 golden · relationship UI (B-3) · materialized projections / snapshots / sharding (SPEC-009) · images.

---

## 2. The gate (operator-run, non-delegable)

The founder browses Mara's world across **all four** surfaces, as **two viewers (Player and Jonas)**, and confirms by eye in the DevTools payloads:

**(a) Every page type is perception-bound — no unperceived canon in any payload, AND no existence leak via the status code (§5.1).**
- **Location** — Player opens Tavern → `200`, sees the Player-private Tavern observation; flips `?viewer=` to Jonas → `200` (Tavern is common knowledge) with name `"Tavern"` but **empty** collected knowledge. No `canon_event.summary` text anywhere.
- **Artifact** — Player opens the Sealed Note → `200`, the discovery observation, `perceived_name: null` (name withheld, §0.2); Jonas opens the same id → **`404`** (the note is not in Jonas's existence set — §5.1), indistinguishable from a fabricated id. A `200`-with-withheld-name for Jonas would itself leak existence.
- **Timeline** — Player's timeline lists his held perceptions in tick order, each pointing to a perception record; Jonas's timeline is **empty** (he learned nothing in 0A). The planted secret never appears in Jonas's timeline.

**New gate assertion (reviewer):** `GET` the Sealed Note's page as Jonas and `GET` a **fabricated id** as Jonas — both return **`404`**, byte-indistinguishable. Player's `GET` on the same note returns `200`. Asserted as a pair (present-200 for Player / absent-404 for Jonas on the same id) — teeth-proven: removing the `fn_entity_visible` gate turns the Jonas-404 assertion red while Player-200 stays green.

**(b) THE NEW SHARP CONDITION — the index does not reveal the EXISTENCE of entities the viewer has never perceived.**
- The Sealed Note is **PRESENT** in Player's artifact index and **ABSENT** from Jonas's — not redacted, not placeholdered: the `entity_id` simply is not in Jonas's payload. Existence is itself canon.
- Asserted as a **paired present/absent test on the same `entity_id`** (an absence-only assertion is forbidden — it passes on an empty index). **Teeth-proven:** breach `fn_compendium_index` (drop the FILTER-1 composition) → the note appears in Jonas's index → the absence half goes red.
- Second, symmetric negative on the actor index: O1–O5 (unnamed, unperceived by anyone) are absent for **both** viewers.

**(c) Expanded-seed invariants (reviewer requirement) — the spine must not silently regress under the new rows.** The gate re-runs **I-1 (replay), I-2 (universal provenance), I-7 (projection-writer)** on the **EXPANDED** seed, not merely confirms existing assertions stay green. The new artifact, the discovery event, and the two new perceptions must themselves be **replay-clean** and **provenance-complete**:
- **I-2:** both new perceptions reference an `accepted` event (the discovery event); no orphan rows.
- **I-1:** replay rebuilds identically. **The discovery event is an `observation` — it changes *perception*, not canon (ADR-005 data-layer isolation), so it carries NO `state_mutation` and produces NO `artifact_state` row** (the Sealed Note exists in `entity_registry` with no state projection, exactly as Tavern/Square locations already do). Replay rebuilds from `state_mutation`; adding zero mutations ⇒ golden/replay identical (`80`/`90` untouched). *(RESOLVED §6: reviewer took the no-mutation default; carry-state/artifact_state deferred.)*
- **I-7:** no new projection writes; `actor_state`/`location_state`/`artifact_state` remain maintainer-only.

**Exit:** all prior pgTAP green + new per-page-type tests + the index existence-filter negative test (teeth-proven) + **I-1/I-2/I-7 re-run on the expanded seed** → ADRs/contract updated → founder browser check → tag `chunk-4-compendium-gate` on the verified **backend** main merge. Frontend leg PRs to `dreamchat-frontend` main consuming the published schemas.

---

## 3. The reused spine — FILTER 1 unchanged, only FILTER 2 varies

| Layer | Function | Status in Chunk 4 |
|---|---|---|
| **FILTER 1 — safety wall (I-3/B-1)** | `fn_visible_perceptions(world, viewer)` | **REUSED UNCHANGED.** `holder ∈ {viewer} ∪ {world faction/group holders}` AND `invalid_tick IS NULL` AND `expired_at IS NULL`. Never reimplemented per page. |
| **Name gate (B-1, going-in 5)** | `fn_perceived_name(world, viewer, entity)` | **REUSED UNCHANGED.** CK-gated; withholds (NULL) when no CK name and no viewer name. |
| **FILTER 2 — about-ness lens** | varies per surface (below) | The *only* thing that changes per page type. |
| **Shared assembly core (§0.1)** | `fn_collected_knowledge(world, viewer, target)` | NEW. FILTER 1 ∘ `perception_subject.entity_id = target` ∘ genesis exclusion ∘ grouped JSON. |

**FILTER 2 per surface:**
- **Actor / Location / Artifact page:** `perception_subject.entity_id = target` (identical lens; shared core).
- **Timeline:** `holder = viewer` (the perspective's own history; §0.3).
- **Index:** `EXISTS (perception_subject row reaching the entity, admitted by FILTER 1)` — see §6.

**Identity/name substrate exclusion (unchanged tripwire from Chunk-3 §3):** the collected-knowledge core excludes `world_genesis`-sourced rows so a name never masquerades as a knowledge item. The exclusion keys on `event_type <> 'world_genesis'` and **must** switch to a real name/identity discriminator the day genesis sources any non-name perception.

---

## 4. Per-page about-ness, walked on real seed rows

Cast: `Player(aaaa) Mara(bbbb) Jonas(cccc) Tavern(dddd) CommonKnowledge(eeee) O1..O5 Square(…a1)` + **NEW** `SealedNote(artifact)`.

### 4.1 Timeline (`timeline/1`) — FILTER 1 ∘ `holder = viewer`, ordered by `valid_tick`

**FILTER 2 = relevance, not a narrower wall (§0.3):** FILTER 1 stays unchanged and CK-inclusive; the relevance lens then keeps `holder = viewer` and so excludes the universal-CK rows (held by the Common-Knowledge faction). Acquired-public rows (`holder = viewer`) stay; only ambient genesis-CK rows drop.

| viewer | what appears | why |
|---|---|---|
| Player | shared-of-E1 (`Day 1`, tick 100); the about-Mara observation (tick 100); the new Note + Tavern observations (tick 100); the ~12 noise "I moved to …" rows where Player was the mover (ticks where `actors[(i%8)+1]=Player`) | all held by Player; ambient genesis-CK rows excluded by relevance (Player holds none anyway) |
| Jonas | **empty** — Jonas holds no perceptions in 0A | perception-binding: no history ⇒ honest emptiness, not omniscient canon |

- Each item points to a **perception record** (`perception_id`), never a canon row (Timeline & Perception PRD AC#1 / §4 — *"Timeline points to perception versions, never canon"*).
- **Ordered by `valid_tick`** (world-history order; **I-9 guarantees `acquired_tick ≥ valid_tick`**, so world-history order is well-defined; Timeline & Perception PRD §7 registry mechanics, ADR-030 tick semantics). **[citation (c)]**
- `before_tick` is an optional cursor: `valid_tick < before_tick` when supplied.
- **Leak test (paired):** Player's shared-of-E1 row is PRESENT in Player's timeline and ABSENT from Jonas's — same `perception_id`, both halves.

### 4.2 Location page (`location_page/1`) — FILTER 2 = subject = location

| viewer | loc | status | what appears | why |
|---|---|---|---|---|
| Player | Tavern | `200` | the Player-private Tavern observation (NEW seed row, subject = Tavern); `perceived_name = "Tavern"` (CK) | holder=Player; subject = Tavern |
| Jonas | Tavern | `200` | `perceived_name = "Tavern"` (CK) but **empty** collected knowledge | Tavern is common knowledge (exists for all — §5.1), but Jonas holds nothing about it |

- Envelope fields pinned to seed: `part_of = null`, `current_synthesis = null`, `last_known_status` perception-bound or `null`, `known_areas_inside = []`, `key_actors = []` (the last two **deferred** — §1).
- Never reads `location_state` (B-1/I-3 hard rule).

### 4.3 Artifact page (`artifact_page/1`) — FILTER 2 = subject = artifact

| viewer | artifact | status | what appears | why |
|---|---|---|---|---|
| Player | Sealed Note | `200` | the discovery observation (NEW, subject = Note alone); `perceived_name = null` (withheld, §0.2) | holder=Player; subject = Note |
| Jonas | Sealed Note | **`404`** | nothing — indistinguishable from a fabricated id (§5.1) | Jonas holds nothing and the note is non-CK ⇒ not in his existence set; a 200 would leak existence |

- Envelope: `perceived_type = null`, `current_synthesis = null`, `last_known_location` perception-bound or `null`, `current_holder_owner_access = null` (Carry-state **deferred** — §1).
- Never reads `artifact_state` (B-1/I-3).
- **[citation (b)]** The discovery perception's `perception_subject = {Note}` **alone** — *not* the discovery event's participants `{Player, Note, Tavern}`. Subject ≠ participants: the belief is *about the note*. `{Note} ⊆ participants` keeps it future-proof, exactly the about-Mara precedent (`{Mara} ⊆ {Player,Mara}`). The seed's generic backfill skips it (explicit subject already present), so it is never over-attributed (ADR-035 derivation cross-check).

---

## 5. The index existence predicate (the new wall)

**[citation (a)]** The index existence rule is a **read-side application of B-1 / I-3 / B-2** to the index payload. It introduces **no new engine ADR** and **no new mechanism** — it is the Chunk-3 safety wall composed with about-ness.

```sql
-- FILTER 1 (unchanged) ∘ perception_subject about-ness, bucketed by kind AFTER the existence join.
CREATE FUNCTION fn_compendium_index(p_world_id uuid, p_viewer_id uuid, p_kind text)
RETURNS TABLE (entity_id uuid, perceived_name text)
LANGUAGE sql STABLE AS $$
  SELECT DISTINCT er.entity_id,
         fn_perceived_name(p_world_id, p_viewer_id, er.entity_id)
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp      -- FILTER 1, unchanged
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id
  JOIN entity_registry er ON er.entity_id = ps.entity_id AND er.world_id = p_world_id
  WHERE er.entity_kind = p_kind;                                -- kind = post-filter, never a parallel path
$$;
```

- **CK needs no separate set or membership table:** CK entities carry universal-holder `perception_subject` rows (the genesis name perceptions, subject = the named entity) that FILTER 1 admits for **every** viewer — they fall out of the *same* predicate (consistent with the Chunk-3 no-CK-membership-table deferral, SPEC-006).
- **`entity_kind` read is not a leak:** the entity is already in the viewer's existence set before its kind is read; bucketing an in-set entity is safe.
- **An entity not in the predicate's output is ABSENT** — never redacted, never placeholdered.
- **Known-but-unnamed rendering (reviewer requirement, §0.2):** `perceived_name` on every index entry is `fn_perceived_name(world, viewer, entity)` — **perception-layer-sourced**. For the Sealed Note it is `NULL` (withheld). The entry is **still present** (Player knows the note exists), rendered as a withheld-name label. The backend **NEVER** substitutes `entity_registry.canonical_name` (that canon name is hidden truth — B-1 / I-3 / ADR-005) and **NEVER** omits the note. Existence is perceived; the name is withheld; the canon name never leaks.
- This SETOF function is the safety-critical core, targeted **directly** by the gate-critical negative pgTAP (no HTTP needed — exactly as `42_visible_perceptions_test` targets `fn_visible_perceptions`). The endpoint wraps it into the `compendium_index/1` JSON envelope.

**Walk on real rows:**

| viewer | kind | index contains | absent | proves |
|---|---|---|---|---|
| Player | artifact | `{Sealed Note}` | — | Player perceives the note |
| Jonas | artifact | `{}` | **Sealed Note** | existence not leaked — the new sharp condition |
| Player / Jonas | actor | `{Player, Mara, Jonas}` (CK names) | **O1..O5** | unperceived, un-CK entities absent for both |
| Player / Jonas | location | `{Tavern, Square}` (CK names) | — | CK locations exist symmetrically (no leak) |

### 5.1 Two existence channels, one predicate — the direct-id page must 404 (reviewer MUST-RESOLVE)

The index closes the **browse** path. The **direct-id page** path (`GET /compendium/{kind}/{id}/page`) is a *second existence channel*: a `200` is itself a revelation that the entity EXISTS, even when the payload carries no canon name. The `?viewer=` debug flip makes an unperceived-id request directly reachable, so this is not theoretical.

**Fix:** the page endpoints apply the **same existence predicate** and return **`404` for an entity not in the viewer's existence set — indistinguishable from a request for a nonexistent id.** Defense-in-depth at the API boundary (`mvp_slice_and_bridge` §4: I-3 enforced at gate, assembly, *and* API boundary).

```sql
-- the boolean form of the SAME predicate fn_compendium_index rests on (FILTER 1 ∘ perception_subject).
CREATE FUNCTION fn_entity_visible(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT EXISTS (
    SELECT 1
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp     -- FILTER 1, unchanged
    JOIN perception_subject ps ON ps.perception_id = vp.perception_id
    WHERE ps.entity_id = p_entity_id);
$$;
```

- Each page function (`fn_actor_page` / `fn_location_page` / `fn_artifact_page`) **returns `NULL` when `fn_entity_visible` is false**; the Go reader maps `NULL → 404`. The gate stays in SQL (ADR-P017), never in Go.
- **Applies to the actor page too** — closing a latent Chunk-3 direct-id leak (`fn_actor_page(Player, O1)` previously 200'd a withheld page for an unperceived actor). `45_actor_page_test.sql` stays green **unchanged**: it only ever exercises Mara, who is CK ⇒ always visible ⇒ still `200` (HARD CONDITION §0.1 honored).
- **CK entities (Tavern/Square) still `200` for every viewer** — their existence is common knowledge.
- **Same predicate, both channels:** membership in `fn_compendium_index(world, viewer, kind)` ⟺ `fn_entity_visible(world, viewer, id)` (set-form ⟺ boolean-form of FILTER 1 ∘ `perception_subject`). The browse path and the direct-id path can never disagree.

| viewer | GET page on… | status | indistinguishable from |
|---|---|---|---|
| Player | Sealed Note | `200` (withheld name + collected knowledge) | — |
| Jonas | Sealed Note | **`404`** | a GET on a **fabricated id** — existence not leaked |
| any | Tavern / Square (CK) | `200` | — (existence is CK) |

**Inline cross-links rule (defined; payload empty this chunk):** a link is rendered **only if** the target is in the viewer's `fn_compendium_index` existence set for its kind; a link to a non-perceivable entity is **OMITTED** (never a placeholder that reveals existence) — the *same* predicate, not a second check. Seed content has no structured refs ⇒ `inline_links: []`. Live extraction deferred (§1, §12).

---

## 6. Seed additions (deterministic; one cast-manifest update)

Minimal, non-colliding additions (verified against every 0A/0B assertion, §11):

1. **`SealedNote` artifact** — one `entity_registry` row, `entity_kind = 'artifact'`, fixed UUID, **NOT** CK-named (no genesis name perception). **`created_by_event` = the discovery event** (reviewer confirm #1 — introduced mid-timeline, unlike the genesis cast whose `created_by_event` is null).
2. **One discovery event** — `event_type = 'observation'`, tick 100 **beat_seq 1**, `status='accepted'`, `visibility_scope='private'`, **no state_mutation**. **`(in_world_tick, beat_seq) = (100,1)` does not collide with E1 `(100,0)` under `uq_ce_accepted_order` (ADR-034/SPEC-002)** — verified: E1 is `(100,0)`, so `(100,1)` is free (reviewer confirm #2). ∉ noise range [101,200]. Participants `{Player(observer), SealedNote(artifact,'discovered'), Tavern(location,'setting')}`.
3. **Two Player perceptions** sourced to that event, `epistemic_type='direct'`, fixed `perception_id`s:
   - subject = **SealedNote** alone — content e.g. *"A small folded note, sealed with dark wax. No markings, no sender."*
   - subject = **Tavern** alone — content e.g. *"The tavern was tense and quieter than usual."*
   Each gets an **explicit** `perception_subject` row; the generic backfill (`NOT EXISTS`) skips them → precise, never over-attributed.

**The one existing assertion that must change:** `40_perception_test.sql:4` asserts the cast manifest `entity_registry = 11` → update to **12** with a comment (`+ SealedNote artifact, chunk-4`). This is a cast manifest *meant* to track the cast — an honest update, not a workaround. **Everything else is purely additive** (§11).

**Why these miss every other filter** (same discipline as Chunk-3 §5): new event ≠ E1/E102/genesis and tick 100 ∉ [101,200] ⇒ noise-count and determinism guards hold; perceptions are Player-held `direct` sourced to the new event ⇒ `Player shared-of-E1 = 1`, `Mara told-of-E1 = 1`, `Jonas E1 = 0`, `5 CK names`, and "every perception has ≥1 subject" all hold; no state_mutation ⇒ replay/golden untouched.

**The `observation` event carries NO `state_mutation` and produces NO `artifact_state` row — RESOLVED: reviewer took the default.** An observation changes *perception*, not canon (ADR-005 data-layer isolation: perception is never a canon mutation). The Sealed Note exists in `entity_registry` with **no state projection**, exactly as the Tavern and Square location entities already do today. This keeps the expanded seed replay-clean (I-1: zero added mutations ⇒ golden/replay identical) and provenance-complete (I-2: perceptions → accepted event) without reopening the `80`/`90` golden assertions. **The Artifact page this chunk = withheld-name + collected knowledge; carry-state / artifact_state deferred** — the note gains canonical state in a later chunk if/when carry-state lands (§1, §12 ledger).

**Expanded-seed invariant re-run (reviewer requirement):** after the additions, the gate explicitly re-runs **I-1 / I-2 / I-7** on the expanded seed (not just the scoped assertions) — see §2(c), §10 step 1.

---

## 7. Schema work & functions (TDD failing-test-first)

One additive migration (`2026061409000X_compendium_read_functions.sql`), no frozen DDL reopened:

- `fn_entity_visible(world, viewer, entity) → boolean` — boolean form of the existence predicate (§5.1); the page-endpoint 404 gate.
- `fn_collected_knowledge(world, viewer, target) → json` — shared core (§0.1, §3).
- `fn_location_page(world, viewer, location) → json` — `location_page/1`; **returns `NULL` when `fn_entity_visible` is false → Go maps to 404** (§5.1).
- `fn_artifact_page(world, viewer, artifact) → json` — `artifact_page/1`; same `NULL → 404` existence gate.
- `fn_timeline(world, viewer, before_tick bigint DEFAULT NULL) → json` — `timeline/1` (no per-id gate — timeline is the viewer's own holdings, no foreign id).
- `fn_compendium_index(world, viewer, kind) → SETOF (entity_id, perceived_name)` — existence predicate (§5); plus a thin JSON wrapper for the endpoint.
- `fn_actor_page` **refactored** to call `fn_collected_knowledge` **and** gated by `fn_entity_visible` (`NULL → 404`, closing the latent Chunk-3 direct-id leak, §5.1); `45_actor_page_test.sql` unchanged = regression guard (only exercises CK Mara ⇒ always 200).

`fn_visible_perceptions` and `fn_perceived_name` are **untouched**.

---

## 8. API contract — Bridge §4.1, D-7, B-1, B-5

Endpoints (exact, from Bridge §4.1):
```
GET /worlds/{w}/compendium/locations/{id}/page    → location_page/1
GET /worlds/{w}/compendium/artifacts/{id}/page     → artifact_page/1
GET /worlds/{w}/compendium/timeline?before_tick=…  → timeline/1
GET /worlds/{w}/compendium/actors                   → compendium_index/1  (extends the existing actor list)
GET /worlds/{w}/compendium/locations                → compendium_index/1
GET /worlds/{w}/compendium/artifacts                → compendium_index/1
```

- **Viewer resolution (D-7/B-1):** reuse Chunk-3 `ResolveViewer` verbatim — server-derived; `?viewer=` override **only** in creator/debug mode, which swaps identity then runs the **identical** filter (never bypasses the wall).
- **Go stays a thin reader (ADR-P017):** each handler calls its SQL function, stamps `schema_version`, serializes. **Never** reimplements the filter. Same handler shape as `actorpage.go`.
- **Existence 404 (§5.1, defense-in-depth):** the per-id page handlers map a `NULL` SQL result → **`404`** (the existence gate lives in `fn_entity_visible`, in SQL, not Go). An unperceived id is byte-indistinguishable from a nonexistent id. The `?viewer=` debug override runs the **identical** gate on the resolved viewer — it never bypasses it.
- **Published schemas (source of truth for FE codegen):** `core/api/schema/location_page.v1.schema.json`, `artifact_page.v1.schema.json`, `timeline.v1.schema.json`, `compendium_index.v1.schema.json`.

**`compendium_index/1` payload:**
```json
{
  "schema_version": "compendium_index/1",
  "world_id": "…", "viewer_id": "…", "kind": "artifact",
  "entries": [ { "id": "…", "perceived_name": null } ]
}
```
- `entries` is a **flat per-kind list** (cross-type counts deferred — §1). `perceived_name` may be `null` (existence perceived, name withheld).

**`timeline/1` payload:**
```json
{
  "schema_version": "timeline/1",
  "world_id": "…", "viewer_id": "…",
  "records": [
    { "perception_id": "…", "content": "…", "epistemic_type": "direct",
      "occurred_at_tick": 100, "display_label": "Day 1",
      "confidence": 1.0, "decay": {"stale": false, "last_confirmed_label": "Day 1"} }
  ]
}
```
- Ordered by `occurred_at_tick` (= `valid_tick`). Each record references a `perception_id`, never a canon row. Wall-clock never crosses (B-5).

`location_page/1` and `artifact_page/1` mirror `actor_page/1` (Chunk-3 §6) with `perceived_role`→`perceived_type` (artifact) and the location envelope (`part_of`, `known_areas_inside: []`, `key_actors: []`); both carry `collected_knowledge_groups` and `inline_links: []`.

---

## 9. Frontend leg — separate repo `dreamchat-frontend` (D-7, D-10, C-4, F-1/F-2)

**Never built in this repo** (AGENTS.md governance, D-10). The frontend PR consumes the published schemas:
- New routes: Location page, Artifact page, Timeline, and a per-type **Compendium index** (browse → navigate into a page).
- Existence-respecting inline links: the renderer honors the §5 rule; with `inline_links: []` this chunk it renders none (no placeholder for absent entities).
- Vocabulary: Glossary terms only (F-1/F-2) — "Locations", "Artifacts", "Timeline", "Collected Knowledge"; never entity/perception/canon. Unnamed artifacts render gracefully (no name leak).
- Play mode shows the perceived world (C-4); creator/debug `?viewer=` flip lets the founder watch the payload change in DevTools — the by-eye gate atop the SQL proof.
- One Location hierarchy expression (C-12) — moot in 0A (`part_of = null`); honored when hierarchy lands.

---

## 10. Build order (TDD iron law; one plan, one PR per repo leg)

**Backend leg → backend main:**
1. **Seed additions** (§6) → update `40_perception_test.sql` 11→12 → confirm all 0A/0B pgTAP still green → **re-run I-1 / I-2 / I-7 on the EXPANDED seed** (replay-clean, provenance-complete, projection-writer intact — §2(c)), not merely "existing assertions unchanged".
2. **Existence gate + shared core + page functions:** failing per-page tests first → `fn_entity_visible` → `fn_collected_knowledge` → refactor `fn_actor_page` onto both (45 stays green) → `fn_location_page` / `fn_artifact_page` (each `NULL → 404` gated) → coherent-view positives (Player `200`) green.
3. **Timeline:** failing paired present/absent test → `fn_timeline` → order-by-`valid_tick` + `before_tick` green.
4. **Index existence wall:** failing paired present-for-Player / absent-for-Jonas test on the same note `entity_id` → `fn_compendium_index` → teeth-prove by breaching the FILTER-1 composition → green; O1..O5 symmetric-absence negative.
5. **Go readers:** failing endpoint tests → thin handlers (reuse `ResolveViewer`, `?viewer=` override) → **`NULL → 404` mapping (§5.1) + the page-existence assertion: Jonas's `GET` on the note is byte-indistinguishable (both `404`) from a fabricated id, Player's is `200`; teeth-proven by removing the `fn_entity_visible` gate** → `schema_version` stamps → publish 4 JSON schemas → green.
6. **Docs/ledger:** §12 touchpoints.
7. **Gate:** all pgTAP green → founder browser check (both viewers, all four surfaces) → tag `chunk-4-compendium-gate` on the verified backend main merge.

**Frontend leg → dreamchat-frontend main:** index + 4 page types + existence-respecting links (empty), codegen'd from the published schemas. Opened as its own PR; never strands on a session branch.

---

## 11. Non-collision verification (must hold before merge)

| Assertion | Effect of additions | Verdict |
|---|---|---|
| `40_perception_test.sql:4` registry = 11 | +1 artifact → 12 | **UPDATE 11→12** (cast manifest) |
| `40` Mara told-of-E1 = 1 / Player shared-of-E1 = 1 / Jonas E1 = 0 | new rows are `direct`, new-event-sourced, Player-held | hold |
| `14` every perception has ≥1 subject | new rows carry explicit subjects | hold |
| `14` exactly 5 CK name perceptions | note is non-CK, no genesis name | hold (=5) |
| `42` Player visible > 0 / Jonas no Mara-held / closed = 0 | +2 Player-held open rows | hold |
| `43` perceived_name (Mara/O1/Player) | note untouched; add new `Note=NULL` assertion | hold |
| `70` determinism `(tick,beat_seq)` unique | new event (100,1) unique | hold |
| `uq_ce_accepted_order` (ADR-034/SPEC-002) on `(world,tick,beat_seq) WHERE accepted` | discovery (100,1) ≠ E1 (100,0) | hold (verified) |
| `80`/`90` golden/replay | no state_mutation added | hold |
| `checks_0A` i2_perceptions_ok | new event `status='accepted'` | hold |

---

## 12. Rule / ADR / SPEC ledger touchpoints

- **No new engine ADR.** The index existence rule is read-side B-1/I-3/B-2 **[citation (a)]**; `perception_subject` (ADR-035) and `fn_visible_perceptions` (FILTER 1) are reused unchanged.
- **ADR-035 derivation-agreement audit — subset, not equality (reviewer ledger line) [citation (b)]:** for a `direct` perception, `perception_subject ⊆ source_event participants` is the *agreement* condition — the participant-derivation over-counts (it includes the observer and the setting), and the explicit subject is the *precise* about-ness. **Any future derivation-agreement audit MUST check `subset (⊆)`, never `equality (=)`** — an equality check would false-flag every `direct` perception (the about-Mara fixture `{Mara} ⊆ {Player,Mara}` and the note fixture `{Note} ⊆ {Player,Note,Tavern}` both already rely on this).
- **Schemas published:** `location_page/1`, `artifact_page/1`, `timeline/1`, `compendium_index/1` (source of truth for FE codegen across the repo boundary).
- **Deferrals recorded with firing triggers** (§1): Carrying (Chunk 5) · Graph Inspector (provenance/bundle reads needed + bundles unfrozen) · Timeline evolution (perception versioning, Phase-1+) · Location known-areas/Key-Actors (hierarchy + co-location data) · Artifact Carry-state (Carrying) · live inline links (structured entity refs) · cross-type counts / unified index (Known-World overview façade over the same function).
- **Rules cited:** B-1, B-2, B-3, B-5, B-9, B-10 · C-4, C-12 · D-1, D-2, D-7, D-9, D-10 · F-1, F-2 · GA-2, GA-4 · I-3, I-9 · ADR-006, ADR-016, ADR-029, ADR-035, ADR-P017.

---

## 13. Out of scope (frozen)

beat loop / any LLM / writes / event creation (ADR-029) · bundles / causality / Seren / Phase-4 golden · relationship UI (B-3) · play loop (Chunk 5) · Carrying overlay · Graph Inspector · materialized projections / snapshots / sharding (SPEC-009 — live SQL joins only) · images · perception-version evolution · live inline links · location hierarchy lenses · artifact carry-state · unified cross-type index.
