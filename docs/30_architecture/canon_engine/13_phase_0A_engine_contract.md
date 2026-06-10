# 13 — Phase 0A Engine Contract

**Status:** The buildable artifact. This is the one document that ends the documentation phase: it specifies exactly what Phase 0A is, the tables, the manual seed, the expected derived rows, the replay rule, and the pass/fail checks. When the checks here go green on a deployed schema, the architecture has stopped being theory. Nothing in 0A uses an LLM.

**Scope rule:** if a thing is not named in this document, it is **not** in 0A. The whole point of 0A is to prove the deterministic spine before any cleverness touches it.

---

## 1. What 0A proves

> Manually inserted accepted events produce correct materialized projections and correct perception records, and the projections can be dropped and rebuilt from the event log to the same domain state.

That is the entire goal. No narration, no extraction, no entity resolution, no canonization pipeline, no causal bundles, no thresholds, no backstage, no dirty flags, no context assembler. Those arrive in later phases. 0A is the foundation they all silently assume.

## 2. In scope / out of scope

**In 0A:** `canon_event`, `event_participant`, `state_mutation`, `perception_record`, `actor_state`, `location_state`, `artifact_state`, `relationship_state`, `entity_registry`; the append-only trigger; the projection-maintenance triggers (plain SQL, fired on accept); the replay/rebuild procedure; invariants I-1, I-2, I-7.

**Out of 0A (explicitly):** `causal_bundle*` (tables may exist empty; nothing writes them — that is 0B), `narrative_claim`, `review_queue`, `threshold_ledger`, `world_snapshot`, `extraction_log`; visibility-scope fan-out beyond direct/private/public; rumor/distortion; the World Clock *service* (0A assigns ticks by hand); Redis (query Postgres directly); pg_ivm (plain triggers only).

## 3. Tables required (subset of doc 03 §1)

Deploy exactly these, with logical time per ADR-030:

- `canon_event` — but in 0A every row is inserted already `status='accepted'`, `origin='fast_path'`, with `in_world_tick`, `beat_seq`, `in_world_label`, `accepted_at` set. No `proposed` lifecycle in 0A.
- `event_participant` — qualified roles.
- `state_mutation` — deltas with `valid_from_tick`, `valid_from_seq`.
- `perception_record` — direct perceptions only; `epistemic_type IN ('direct','told','public')`; `acquired_tick`, `valid_tick` set; `invalid_tick`/`expired_at` NULL.
- `actor_state`, `location_state`, `artifact_state`, `relationship_state` — maintained by triggers.
- `entity_registry` — pre-seeded with the cast.

Append-only trigger active. `DELETE` revoked. Projection tables writable only by the maintainer role (I-7).

## 4. The Mara seed (deterministic, hand-written)

Cast pre-seeded in `entity_registry`: player `P`, NPCs `Mara (M)`, `Jonas (J)`; location `tavern`; (no artifact needed for the core). World `W`. Ticks are integers.

| seq | tick | event | participants | mutations | perceptions |
|---|---|---|---|---|---|
| E1 | 100 | `private_disclosure` (scope=private) "P tells M the mayor keeps a hidden ledger" | P:speaker, M:listener | — | M:`told` ("P told me the mayor keeps a hidden ledger"), P:`shared` |
| — | 101–200 | 100 noise events (P/M/J/other moves + small disclosures among others, none involving the secret) | various | location/inventory deltas | direct perceptions for participants only |
| E102 | 201 | `publicize` (scope=public) "the hidden ledger becomes common knowledge" | P:instigator | — | public-knowledge record in scope; **M's E1 perception untouched**; J becomes eligible to acquire a `public` perception |

Seed is a fixed SQL script (`seed_mara_0A.sql`), checked into the repo, deterministic, re-runnable into a clean DB.

## 5. Expected derived state (the assertions)

After running the seed and letting triggers fire:

**Projections**
- `actor_state` for P, M, J, and noise NPCs reflect the net of their mutations (locations after moves, etc.). Spot-checked against a hand-computed expected table committed alongside the seed.
- `relationship_state` rows exist only where interaction events created them.

**Perceptions (the knowledge-boundary core)**
- M has exactly one active perception referencing E1, `epistemic_type='told'`, provenance → E1.
- P has exactly one active perception referencing E1, `epistemic_type='shared'`.
- **J has zero perceptions referencing E1** before E102 (the negative assertion — the heart of 0A's knowledge-boundary proof).
- After E102: a `public` knowledge record for the ledger exists; J is *eligible* to hold a `public` perception of it (in 0A, eligibility is enough — actual J-acquisition is a Phase 1+ fan-out behavior); **M's original `told` perception still exists, unchanged, not deleted** (ADR-006).

**Provenance (I-2)**
- Every `state_mutation` and every `perception_record` has an `event_id`/`source_event_id` pointing at an `accepted` event. Zero orphans.

## 6. Replay rule (I-1, domain-equivalent)

```
1. Snapshot live projection tables (actor/location/artifact/relationship_state).
2. TRUNCATE the projection tables.
3. Replay: stream accepted events ORDER BY (in_world_tick, beat_seq, recorded_at);
   re-apply each event's state_mutations idempotently.
4. Diff rebuilt projections vs. snapshot, EXCLUDING volatile columns {updated_at}.
5. PASS iff the diff is empty over the non-volatile column set (ADR-026).
```

Perceptions are append-only and not rebuilt from mutations; they are checked separately for provenance (I-2), not for replay equivalence.

## 7. Pass/fail SQL checks

```sql
-- I-2: no orphan provenance
SELECT count(*) = 0 AS i2_mutations_ok
FROM state_mutation sm
LEFT JOIN canon_event ce ON ce.event_id = sm.event_id AND ce.status='accepted'
WHERE ce.event_id IS NULL;

SELECT count(*) = 0 AS i2_perceptions_ok
FROM perception_record pr
LEFT JOIN canon_event ce ON ce.event_id = pr.source_event_id AND ce.status='accepted'
WHERE ce.event_id IS NULL;

-- Knowledge boundary: Jonas does not know the secret before E102
SELECT count(*) = 0 AS j_ignorant_ok
FROM perception_record
WHERE holder_id = :jonas_id AND source_event_id = :e1_id;

-- Mara remembers, correctly typed
SELECT count(*) = 1 AS mara_knows_ok
FROM perception_record
WHERE holder_id = :mara_id AND source_event_id = :e1_id
  AND epistemic_type='told' AND invalid_tick IS NULL AND expired_at IS NULL;

-- Mara's original perception survives publication (not deleted)
SELECT count(*) = 1 AS mara_perception_survives_ok
FROM perception_record
WHERE holder_id = :mara_id AND source_event_id = :e1_id;  -- still present after E102

-- Public knowledge exists after E102
SELECT count(*) >= 1 AS public_knowledge_ok
FROM perception_record
WHERE source_event_id = :e102_id AND epistemic_type='public';

-- Append-only: attempt an illegal UPDATE must raise
-- (run as a should-fail test)
-- UPDATE canon_event SET summary='tampered' WHERE event_id = :e1_id;  -> expect trigger rejection
```

I-1 (replay) is a procedure (§6) wrapped as a test harness that returns a single boolean. I-7 is a permission test: a write to a projection table by a non-maintainer role must be rejected.

## 8. Definition of done for 0A

All true, on a clean deploy of the seed:

1. Seed runs without error; all triggers fire.
2. Every §7 SQL check returns the asserted boolean.
3. The illegal-UPDATE and non-maintainer-write tests both raise.
4. The replay procedure (§6) returns empty diff (domain-equivalent).
5. Re-running the entire seed into a fresh DB produces identical results (determinism).

When all five hold, 0A is done and Phase 1 (fast-path play loop) may begin. **Do not start Phase 1, and do not touch bundles/0B, until these are green.**

## 9. Deliverables for this phase

- `schema_0A.sql` — the subset DDL with triggers and role grants.
- `seed_mara_0A.sql` — the deterministic seed.
- `expected_projections_0A.csv` — hand-computed expected projection rows.
- `checks_0A.sql` — the §7 assertions as a runnable suite.
- `replay_0A.sql` / harness — the §6 procedure returning pass/fail.

These five files are the first code in the project, and the green suite is the artifact that retires the most risk for the least effort. Everything after 0A builds on a spine that is *proven*, not *assumed*.
