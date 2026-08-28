# canon-spine · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-1 · The canon spine ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `canon-spine.tech.md` holds how it is built; `canon-spine.seams.md` holds
what crosses its boundary.

---

## What this domain is for

**One job: what happened, recorded once, immutably, with its cause attached.**

Canon events are the world's only source of truth; everything else — current state, knowledge,
timelines, pages — is a derived projection, rebuildable by replay (`ADR-001`). The product reason
the spine exists: a persistent world needs auditability, lineage, correction and replay, and *"chat
history cannot safely provide any of these"* (`ADR-001` Rationale). The doctrine's own test of
success: **if the transcript is deleted, the world loses nothing** (`digest/S04` Topic 1, quoting
the frozen `canon_engine/01`).

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Canon event** | One accepted, append-only row in `canon_event`. Never edited; only its lifecycle fields (`status`, `accepted_at`, `superseded_by`) may ever change, and only along legal transitions. |
| **Attempt** | What a caller asks the spine to commit: a typed proposal, not an outcome. The engine blocks impossibilities; it never awards success. |
| **Ruled event** | An attempt that was adjudicated upstream (a referee's `truth`) and arrives pre-decided. Committed by the second door. |
| **Door** | One of the two commit functions — `apply_event`, `apply_ruled_event`. The only runtime write surface for canon (see `tech.md` for the one seed-time exception). |
| **Provenance** | The mandatory link from every derived row back to its licensed cause: `state_mutation.event_id` and `perception_record.source_event_id` are `NOT NULL` (`I-2`, `ADR-008`). |
| **Compensating event** | The only legal repair: a new event that corrects, plus a status transition on the old one. Never an edit (`ADR-001`). No writer exists yet — see §deliberately not built. |
| **Origin** | How an event entered: a closed CHECK set of eight (`fast_path` … `telegraph`). Who may mint which value is a ruling, not a convention. |

**Projection**, **perception** and **replay** are neighbours' vocabulary — WE-2 and WE-3 — and
appear only in `canon-spine.seams.md`.

## What this domain is not

- **Not the read model.** Projections, `apply_mutation`'s effects, and replay invariance belong to
  WE-2 · Projections and the read model. The spine writes the mutation ledger; it never answers
  "what is the current state".
- **Not who noticed.** Perception fan-out is WE-3 · perception-and-knowledge. The doors *trigger*
  it; deciding who perceives is not a spine decision (`SPEC-035` set that precedent).
- **Not adjudication.** Whether an attempt succeeds is the play loop's referee (WE-7). The doors
  enforce a structural floor only — *"contracts arrive pre-adjudicated"* (`tech.md`).
- **Not genesis.** Seed and template writes (`origin='fast_path'`, pre-accepted backstory) are
  WE-10's lane; the spine only defines the shape they must respect.

## Product rules — decisions already made

Ids only; the law lives where the id resolves. Cite it, never restate it.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-001` | Events are immutable; revisions via compensating events and status transitions. | An edit destroys the lineage that makes correction and replay possible at all. |
| `D-1` | Nothing mutates canon directly — proposals in, the Core validates and commits. | A direct write is canon written outside the gate; `beat_authority_test.go` exists to catch exactly this. |
| `ADR-009` | LLM proposes; a deterministic gate decides; events have a lifecycle. | Generated prose becomes source of truth — the failure the whole engine exists to prevent. |
| `B-5` | Append-only time: canonical order is `(in_world_tick, beat_seq)`, never `recorded_at`. | Ordering by transaction time makes replay non-deterministic (`SPEC-002`'s receipt). |
| `SPEC-035` | **Landed.** The event names its witnesses; malformed input is refused, never dropped. | A silent drop means "who saw it" has no answer anywhere in the database. |

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No repair machinery.** `origin='compensation'` is in the CHECK set, and the append-only trigger
  permits `accepted→retconned|superseded` — but no code path produces any of them (verified: no
  writer in `core/db/schema.sql` or `core/api/*.go`, 2026-08-27). The doctrine is settled
  (`ADR-001`); the machinery waits for a need. Corrections, when they come, are **present-forward**
  — deep retroactive rewrite and timeline forks are explicitly parked (`ADR-016`).
- **No proposed lifecycle.** Both doors write `status='accepted'` directly; `proposed` exists in
  the schema and is produced by no live write path. The acceptance-transition projection trigger is
  the deferred half of `SPEC-003` — *"correct and untestable in 0A"* — owned by the first Phase-1
  chunk that introduces the proposed lifecycle.
- **No bundle writers.** The causal tables ship with constraints and acyclicity checks and **no
  automated runtime path writes bundles before Phase 4** — frozen wording, `ADR-029`; the
  provenance-always / bundles-selectively split is `ADR-008`.
- **No mutation idempotency ledger.** Phase 0A relies on absolute-set semantics, sound *"only while
  deltas are forbidden"*; the moment delta semantics appear, the ledger becomes mandatory
  (`SPEC-004`).
