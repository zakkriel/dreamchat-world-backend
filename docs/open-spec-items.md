# Open Spec Items

ADR numbers are assigned at proposal time in `canon_engine/02` (D-5). Do not pre-assign.

## SPEC-001 — mutation → relationship_state addressing
doc 03 does not specify how a single-`entity_id` `state_mutation` addresses a
`(world_id, a_id, b_id)` `relationship_state` row. Phase 0A ships a **no-op stub** branch in
`apply_mutation()` and keeps `relationship_state` empty (pgTAP-asserted zero rows).
- **Owner:** Chunk 5 (Phase-1 fan-out) brainstorm.
- **Expected outcome:** a proposed new ADR in `canon_engine/02`, informed by Phase-1 evidence (D-9).

## SPEC-002 — recorded_at in the canonical replay ordering key
doc 13 §6 and doc 03 §3.4 write the replay order as `(in_world_tick, beat_seq, recorded_at)`.
`recorded_at` is volatile wall-clock (B-5 transaction-time; ADR-026 volatile) and is a latent
determinism foot-gun as an ordering tiebreaker. Phase 0A removes it and guarantees a domain-only
total order via per-world `(in_world_tick, beat_seq)` uniqueness.
- **Owner:** Chunk 1 retro.
- **Expected outcome:** a proposed new ADR in `canon_engine/02` (number assigned at proposal time).
- **Proposal text:** "Canonical event ordering is `(in_world_tick, beat_seq)`, required UNIQUE per
  world; `recorded_at` is transaction-time (B-5) and excluded from domain ordering (ADR-026)."
- **Status:** non-blocking for 0A.
