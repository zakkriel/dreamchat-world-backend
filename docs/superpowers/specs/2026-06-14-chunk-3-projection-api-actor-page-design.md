# Chunk 3 — Read-only Projection API + Perception-bound Actor Page (Design)

**Date:** 2026-06-14
**Status:** Design approved (founder, 2026-06-14) — pending spec review before write-plan.
**Validation Ladder:** Q2 — *"Can the user inspect a world and trust it?"*
**Constraints:** one worktree · one plan · one PR · TDD iron law (failing test first) · gate red → stop.
**Governing law:** Rules Register (`docs/00_strategy/06_rules_register.md`). Engine matters: `canon_engine/` wins. Rule IDs cited inline.

---

## 0. Two judgment calls flagged for spec review

These two resolutions go slightly beyond the literal brainstorm instructions; called out so they can be vetoed here rather than discovered in code.

1. **Leak-test holder is Player, not Mara.** The non-vacuity requirement ("the same row PRESENT when Player views, ABSENT when Jonas views") forces viewer = holder = Player. The seeded about-Mara belief is therefore *Player's private observation of Mara* (`epistemic_type='direct'`, source E1, subject genuinely = Mara). A Mara-held belief would be absent for Player too, collapsing the "present" half. (Your option "for Mara to hold" would instead require the present-half viewer to be Mara; the numbered requirement said Player, so Player wins.)

2. **The real name gate needs a `world_genesis` device.** Making "this name is common knowledge" *real data* (Change 3) collides with `perception_record.source_event_id NOT NULL` — names have no natural birth event. Resolution: one deterministic `world_genesis` canon event (tick 0) sources common-knowledge name perceptions for the principal cast only; noise actors are deliberately left unnamed so the gate's *withholding* path is exercised by real seed asymmetry. If you consider this too heavy for Chunk 3, the fallback is a structurally-real gate proven only by a pgTAP-local constructed entity (no seed growth) — lighter, but the "all cast names are common knowledge" posture (going-in 5) is then only asserted in tests, not represented in the seed.

---

## 1. Gate (re-pointed to Mara)

Seren does **not** exist in 0A/0B — she is the Phase-4 golden (theft + bundles; doc 07 §3), and ADR-029 / engine INDEX explicitly bar Seren from 0A/0B so causality thinking never contaminates the spine. The gate's intent is fully served by the existing **Mara** seed: the planted secret is the private disclosure at tick 100 (`P tells M the mayor keeps a hidden ledger`), held only by Mara (`told`) and Player (`shared`).

**Gate (executable):**
- **Coherent view:** `viewer = Player`, `actor = Mara` → a coherent perception-bound page (perceived name via common knowledge, sourced knowledge with tick labels, no relationship UI).
- **Non-vacuous leak test (gate-critical, both halves asserted together on the same row + same page):**
  - `fn_actor_page(world, Player, Mara)` **CONTAINS** the seeded *Player-private-about-Mara* perception → PRESENT.
  - `fn_actor_page(world, Jonas, Mara)` does **NOT** contain it → ABSENT.
  - An absence-only assertion is forbidden: it passes on an empty page. The paired present/absent test fails loudly if the wall ever leaks.
- **Supporting I-3 negative (retained):** Mara's private `told` ledger belief (held only by Mara) is absent from `fn_actor_page(world, Jonas, Mara)`.
- **DevTools by-eye:** founder opens Mara's page as Player; flips the creator/debug `?viewer=` override to Jonas and watches the payload change. No unperceived canon in the network payload (I-3).

**Exit:** all 0A/0B pgTAP green + new tests green; ADR-P017 and ADR-035 move Proposed → Accepted; doc 03 §1.3 gains the `perception_subject` DDL; tag `chunk-3-actor-page-gate` on the verified main merge commit. Playbook lines 28 & 66 reconciled to Mara; "Seren's page" recorded as satisfied later at S4.

**Doc-tension reconciliation:** playbook §"chunk 3" question (line 28, "open Seren's page") and the build row (line 66, "Mara/Seren seed") disagree; both reconcile to **Mara** for 0A.

---

## 2. Schema work (Stage-0 — FIRST phase, TDD failing-test-first) — ADR-035, SPEC-008, D-5

### 2.1 `perception_subject` junction
New table added to the frozen Master DDL (doc 03 §1.3), routed via ADR-035 (Proposed → Accepted under this gate; not an ad-hoc migration — D-5):

```sql
CREATE TABLE perception_subject (
  perception_id UUID NOT NULL REFERENCES perception_record(perception_id),
  entity_id     UUID NOT NULL,
  world_id      UUID NOT NULL,              -- carried FROM BIRTH (SPEC-009 tenant-key posture)
  PRIMARY KEY (perception_id, entity_id)
);
CREATE INDEX idx_ps_entity ON perception_subject (entity_id);
CREATE INDEX idx_ps_world  ON perception_subject (world_id);
```

`world_id` is a **delta vs ADR-035's two-column sketch**, recorded in the ADR text on acceptance. Rationale for the asymmetry vs §7 (the three bare tables stay bare): this is a *new* table, so carrying the tenant key costs zero migration and matches SPEC-009 — whereas reopening frozen tables requires a firing trigger.

### 2.2 Backfill
Populate `perception_subject` for every existing seed perception from its source-event participants:
`INSERT ... SELECT pr.perception_id, ep.entity_id, pr.world_id FROM perception_record pr JOIN event_participant ep ON ep.event_id = pr.source_event_id`.
For the seed this makes junction ≡ derivation by construction (the agreement guard then passes for the right reason).

### 2.3 pgTAP (failing first)
- **schema** — table + columns + FK + indexes exist (extends `10_schema_test.sql`).
- **positive** — subjects present for E1 / noise / E102 / genesis perceptions.
- **negative** — an entity that is not a subject of a perception is absent from that perception's subject set.
- **derivation-agreement guard (ADR-035)** — for direct/witnessed perceptions, junction subjects equal `source_event → event_participant`; disagreement is the logged audit flag.

---

## 3. Assembly — two filters, never merged into one predicate

### FILTER 1 — safety wall (I-3 / B-1, gate-critical), SQL function
`fn_visible_perceptions(p_world_id UUID, p_viewer_id UUID) RETURNS SETOF perception_record`:

```
holder_id IN ( p_viewer_id ∪ {world's universal common-knowledge holders} )
AND invalid_tick IS NULL
AND expired_at  IS NULL
```

- This function **is** I-3 made executable. The gate-critical negative pgTAP targets it directly (no HTTP needed).
- **Common-knowledge membership is ambient in 0A** (no membership table). The viewer-set expands only to the world's universal common-knowledge holder(s) (the `Common Knowledge` faction, `eeee…`). A group-holder-with-read-side-membership-expansion is a **deferred storage optimization** for a large standing group whose per-holder rows would be impractical — revisit only under the SPEC-006 scale trigger. It is **not** a new knowledge path and is **never** the default, and is specifically disfavored: a membership-expanded group holder could place a belief on an actor's page with **no originating event** behind it, violating provenance discipline (I-2). Per-holder rows always carry a source event; that is why they are the default (SPEC-006 Path 1 stands unchanged).

### FILTER 2 — relevance lens
Of the permitted rows, which land on `actor`'s page:
- **Primary:** `perception_subject.entity_id = p_actor_id`.
- **Fallback / validation:** derived via `source_event_id → event_participant`.

### Name resolution — `fn_perceived_name(p_world_id, p_viewer_id, p_entity_id) RETURNS TEXT`
A **genuine knowability gate** (Change 3), returning perception-layer content, in priority:
1. A viewer-held *divergent perceived-name* perception's content — **deferred** (no such data in 0A; SPEC ledger, Phase-1+ owner).
2. The entity's **common-knowledge name** perception content (CK-held, sourced to `world_genesis`) — present for the principal cast.
3. Otherwise **NULL (withheld)** — e.g. a noise actor with no CK name and no viewer perception.

The payload's `perceived_name` is **never** a raw `entity_registry.canonical_name` read at projection time (going-in 5). `entity_registry.canonical_name` is only the *seed source* for the genesis name perceptions.

### Hard rule — no canon leak via state
The page **never** reads `actor_state` / `location_state` / `artifact_state` (authoritative canon read-models) for user-facing fields. `last_known_status` is perception-bound or NULL. Reading `actor_state.attrs.location_id` directly would be a silent canon leak (B-1 / I-3).

### Identity vs knowledge separation
`world_genesis`-sourced perceptions are identity substrate. `fn_perceived_name` reads them; the collected-knowledge assembly **excludes** genesis-sourced perceptions so a name never masquerades as a knowledge item.

**Tripwire (must be written, not implied):** keying the collected-knowledge exclusion on `source_event = world_genesis` is correct **only while genesis sources names exclusively**. The day genesis ever sources a *non-name* perception, this exclusion would wrongly drop a real knowledge item. At that point the exclusion **must** switch to a genuine name/identity discriminator (an explicit perception kind/marker) rather than keying on the source event. The source-event key is an accepted Chunk-3 expedient; the firing condition is "genesis sources anything other than a name."

### Assembly function
`fn_actor_page(p_world_id, p_viewer_id, p_actor_id) RETURNS JSON` composes FILTER 1 ∘ FILTER 2 + name resolution + perception-bound status + grouped knowledge. Go calls it and never reimplements the filter (ADR-P017 binding). Live SQL joins only — no materialized projection, no snapshot, no sharding (SPEC-009 tripwires all unfired).

---

## 4. About-ness on the real seed (verification with actual rows)

Cast: `Player(aaaa) Mara(bbbb) Jonas(cccc) Tavern(dddd) CommonKnowledge(eeee) O1..O5 Square`.

| viewer | actor | what appears | why |
|---|---|---|---|
| Player | Mara | Player's `direct` private observation of Mara (NEW seed row); Player's `shared` of E1 (about mayor+telling) | holder=Player; subject includes Mara |
| Player | Mara | `last_known_status` = NULL | Player holds no location perception of Mara — correct perception-binding |
| Jonas | Mara | **nothing about Mara**; Mara's `told` belief ABSENT; Player's private-about-Mara ABSENT | Jonas isn't holder; neither row is common knowledge |
| any | (principal cast) | perceived_name via CK | genesis CK name perceptions |
| viewer w/o perception | O1..O5 | perceived_name = NULL (withheld) | no CK name perception — proves the gate is real |

**Every Mara-perception reachable** (junction or derived): the two E1-derived rows + the new about-Mara row all carry Mara as participant/subject → ✓.

**Recorded seed quirk (not solved — going-in 6):** the E102 common-knowledge ledger record's true subject (the mayor) is not a registered entity, so backfill yields the telling's participant `{Player}`. This is the told-about-a-third-party gap, now visible; fixed for new writes by §2, validated-against-seed. The gate's leak test deliberately does **not** rest on this row (it rests on the genuinely-about-Mara row, §0.1).

---

## 5. Seed additions (deterministic; designed to not collide with existing scoped assertions)

All existing 0A assertions are scoped by `holder + source_event + epistemic_type`, never totals (verified: `40_perception_test.sql`, `50_provenance_test.sql`, `70_determinism_guards_test.sql`, `checks_0A.sql`). The additions below are chosen to miss every existing filter.

1. **`world_genesis` event** — tick 0, beat_seq 0, `status='accepted'`, `visibility_scope='public'`, no state_mutation. Unique `(tick,beat_seq)` ⇒ determinism guard holds; tick 0 ∉ [101,200] ⇒ noise-count assertion holds.
2. **CK name perceptions** — for `{Player, Mara, Jonas, Tavern, Square}` only: held by `Common Knowledge (eeee)`, `epistemic_type='public'`, source = `world_genesis`, `content` = canonical name; each subject-linked in `perception_subject`. Source ≠ E102 ⇒ `public_knowledge_ok` (= 1 at E102) holds. **O1–O5 deliberately unnamed.**
3. **Player-private-about-Mara perception** — held by Player, `epistemic_type='direct'` (≠ the asserted `shared`), source = E1, subject = Mara, fixed `perception_id` for precise test reference, deterministic content (e.g. "Mara listened intently and seemed unsettled"). Type `direct` ≠ `shared` ⇒ `Player shared of E1 = 1` holds; holder=Player ⇒ `Mara told of E1 = 1` holds.

**Net effect on existing 0A tests: none.** New tests are added (§2.3, §6). Build-time check: confirm `80_golden_projection_test.sql` / `90_replay_test.sql` assert no global event/perception totals and no first-event identity (expected: they don't — they rebuild from `state_mutation`, and no mutations are added).

---

## 6. API contract — Bridge §4.1, D-7, B-1, B-5

**Endpoint (exact, from Bridge §4.1):**
```
GET /worlds/{w}/compendium/actors/{id}/page   → Actor Page Projection
```
The actors-**list** endpoint is **deferred** to the next S0 increment (scope = "one Actor page"). FE deep-links Mara.

**Viewer resolution (D-7 / B-1):** server-derived from session / world ownership — the player-controlled actor. **Never** a play-mode client param (presentation must not choose the epistemic boundary). A `?viewer=` override is permitted **only in creator/debug mode**: it swaps the resolved identity and then runs the **identical** safety filter — it never bypasses or weakens the wall. Inspecting "as Jonas" returns exactly Jonas's filtered view, never unfiltered canon. Any path where the override returns rows the resolved viewer wouldn't see is a gate failure.

**Go layer:** thin reader — call `fn_actor_page`, stamp `schema_version`, serialize. Never reimplements the filter (ADR-P017 binding; ADR-P017 Proposed → Accepted). FE⇄BE types **generated** from the schema source of truth (Go structs / OpenAPI / DB → TS), versioned by `schema_version`.

**Payload (`schema_version: "actor_page/1"`):**
```json
{
  "schema_version": "actor_page/1",
  "world_id": "…",
  "viewer_id": "…",
  "actor": {
    "id": "…",
    "perceived_name": "Mara",
    "perceived_role": null,
    "current_synthesis": null,
    "last_known_status": null,
    "known_artifacts": [],
    "collected_knowledge_groups": [
      { "group_key": "<subject-entity-id>", "group_label": "…",
        "items": [
          { "perception_id": "…", "content": "…", "epistemic_type": "direct",
            "occurred_at_tick": 100, "display_label": "Day 1",
            "confidence": 1.0, "decay": {"stale": false, "last_confirmed_label": "Day 1"},
            "source": { "…": "perception-layer only — never a canon row" } } ] }
    ],
    "inline_links": []
  }
}
```

- **Dropped vs PRD §6 field list:** `relationship_to_you` and `known_relationships` — removed **entirely** (AC#7/#8; B-3/B-4). Relationship-flavored info reaches the user only as ordinary collected-knowledge records via valid paths.
- **`current_synthesis` = `null` in 0A** (Change 2). No LLM in chunks 1–4 (ADR-029); a templated stand-in is sourcing-as-decoration and would let the page read as "done" when synthesis isn't built. FE renders honest emptiness.
- **`collected_knowledge_groups`:** grouped **by subject-entity** in 0A (the junction gives this for free); semantic topic-clustering deferred. Each item ships `epistemic_type`, `occurred_at_tick`, `display_label` (from `source_event.in_world_label`), `confidence`, decay flags (AC#2/#4, Bridge §4.1). **Never** surfaces `canon_event.summary` or any canon row.
- **`last_known_status`** perception-bound or NULL (§3 hard rule).
- Time crosses as `tick` + `display_label` only; wall-clock never crosses (B-5).

---

## 7. The 4b decision — world_id on the three bare junction/edge tables

**Defer.** Do **not** add `world_id` to `event_participant`, `provenance_edge`, `causal_bundle_input` in Chunk 3.
- No Chunk-3 path requires it: `event_participant` is only ever reached through its world-scoped parent (`canon_event.world_id`), so scoping is enforced at the parent and never lost; `provenance_edge` and `causal_bundle_input` are **not read in Chunk 3 at all**.
- The change that would require it — row-based sharding by world (SPEC-009) — is an **unfired** tripwire. Modifying frozen DDL with no firing trigger is the speculative pre-building the register's evidence-triggered posture rejects.
- The new `perception_subject` carries `world_id` from birth (new table, zero cost), covering the only place it matters now.

**Two mandatory docs-only actions in this PR (not deferred):**
1. **Correct the SPEC-009 wording.** It currently claims "every core table already carries `world_id`"; that is **inaccurate** for these three. Fix the sentence so the ledger stops asserting something false.
2. **Record the deferral with its firing trigger named:** when SPEC-009 sharding is implemented, these three tables must either gain `world_id` as the distribution key **or** be co-located by their world-scoped parent — decided at that time. Until the trigger fires, they stay as-is.

---

## 8. Frontend minimal shell — D-7, D-2, C-4, F-1/F-2, AC#6/#10

- New FE repo wakes with: app shell + **one Actor page route** + typed client (codegen'd from schema source of truth) + render. Presentation only (D-7) — never canon, never decides outcomes.
- Vocabulary: Glossary terms only (F-1/F-2) — "Collected Knowledge", "Actor"; no "entity / perception / confirmed / false" labels; contradictions co-exist unresolved (PRD §7). Decay/source language per AC#2/#4.
- Named slots (D-2); no module UI in Chunk 3. No relationship UI (B-3).
- Mockup parity to `mock_compendium_actor_seren_v2.png` **minus** the removed bits (AC#10): no trust slider (AC#8), no "Relationship to you" panel (AC#7), "Artifacts" label per Glossary, no "Add note" (parked).
- Play mode shows the perceived world (C-4); founder views as Player; creator/debug override flips to Jonas to watch the payload change in DevTools — the by-eye gate atop the SQL-layer proof.

---

## 9. Build order (TDD iron law; one worktree, one plan, one PR)

1. **Schema (Stage-0):** failing pgTAP → `perception_subject` migration (+world_id) → backfill → positive / negative / agreement green.
2. **Seed additions:** `world_genesis` event + CK name perceptions (principal cast) + Player-private-about-Mara row → confirm all existing 0A/0B pgTAP still green (no scoped-assertion collisions, §5).
3. **SQL wall:** failing I-3 paired present/absent test first → `fn_visible_perceptions` → `fn_perceived_name` (with withholding test on an unnamed noise actor) → `fn_actor_page` → coherent-view positive (Player/Mara) green.
4. **Go reader:** failing endpoint test → handler + server-resolved viewer + creator/debug `?viewer=` override (filter never bypassed) → `schema_version` stamp → green; FE type codegen.
5. **FE shell:** render Mara's page from the real payload; vocabulary + decay/source language; debug viewer flip.
6. **Docs:** ADR-P017 & ADR-035 Proposed → Accepted; doc 03 §1.3 gains `perception_subject` DDL; new SPEC entry (§7) + SPEC-009 wording correction; playbook lines 28/66 reconciled to Mara.
7. **Gate:** all 0A/0B + new tests green → founder browser check (coherent Mara page as Player; flip to Jonas → secret absent in payload) → tag `chunk-3-actor-page-gate` on verified main merge.

---

## 10. Out of scope (frozen — PRD non-goals + register ARE the out-of-scope list)

beat loop / any LLM / event creation / any write path · images & portraits · relationship UI (B-3) · Timeline / Location / Artifact pages · actors-list endpoint (next S0 increment) · materialized projections / snapshots / sharding (SPEC-009 unfired — live SQL joins only) · true `perceived_name` divergence (SPEC ledger, Phase-1+) · per-actor group-membership table (SPEC-006 scale trigger) · world_id on the three bare junction/edge tables (§7 firing trigger).

---

## 11. Rule / ADR / SPEC ledger touchpoints

- **Accepted under this gate:** ADR-P017 (Go backend), ADR-035 (`perception_subject` about-ness).
- **DDL updated on acceptance:** doc 03 §1.3 gains `perception_subject`.
- **SPEC ledger:** SPEC-006 (Path 1 — unchanged, cited); SPEC-008 (resolved by ADR-035 impl); SPEC-009 (wording corrected + sharding firing-trigger for the three bare tables); new deferrals recorded (perceived_name divergence; group-membership storage optimization).
- **Rules cited:** B-1, B-2, B-3, B-4, B-5, B-7, B-9, B-10 · C-4, C-12 · D-1, D-2, D-7, D-9 · F-1, F-2 · GA-2, GA-4 · I-2, I-3 · ADR-006, ADR-016, ADR-029.
