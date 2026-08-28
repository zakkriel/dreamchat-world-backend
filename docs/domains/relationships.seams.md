# relationships · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-9 · Relationships, modelled and never surfaced ·
**Parent bounded context:** World Engine

A seam belongs to two domains; each row declares an expectation — one side owns a fact, the other
consumes it and must not re-derive or re-decide it. `relationships.product.md` holds what the domain
means; `relationships.tech.md` holds how it is built.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Projections & replay** (WE-2) | the projection substrate | `relationship_state` is one of the four projection tables: grant wall (`I-7`), single write path (`apply_mutation`), truncate-and-rebuild replay (`I-1`). This domain never writes the table by any other path, and never adds a volatile column (`tech.md` §Storage). |
| consumes | **Perception & knowledge** (WE-3) | `[INFER]` perceived interactions | The one seam still inferred, and **WE-3's `seams.md` is its home** — whether relationship state derives from what was *perceived* or what *happened* is stated nowhere. An agent hitting it is deciding something new and must say so. Not restated here (`D-6`). |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **The play loop / adjudication** (WE-7) | the `relationships` array in `gather_slice` output (`schema.sql:3344-3349` → `core/api/orchestrator.go:1439`) | The log an NPC *"plays according to"*. The consumer treats relationship counterparty UUIDs as **fetched context, never grounded entities the LLM may claim** — the orchestrator already enforces this (`orchestrator.go:1449-1452`) and no consumer may relax it. |
| provides | **Compendium surfaces** (UX-1) | **nothing — and the nothing is the contract** (`B-3`) | No page payload carries a relationship field (`45_actor_page_test.sql:23-27`). UX-1 must not synthesize a relationship label, panel, or grouping from Collected Knowledge records — relationship-flavored knowledge renders as ordinary sourced knowledge, undistinguished. |
| provides | **Presentation** (`dream-weaver-visuals`, other repo) | nothing to render | The handoff law list's rule 6 — no panel, meter, slider, heart, trust bar, affinity number, "Relationship to you" card — and rule 13's no-Relationships-nav restate `B-3`/`B-4` for the visual side; their authority is the backend ids (`digest/S11_frontend.md` topic 3). The frontend never invents a relationship-strength field (`D-7`). |

## The seams that do not exist

- **The write seam.** Nothing feeds `relationship_state` — the mutation arm is a `SPEC-001` no-op and
  no other writer exists (`tech.md` §The write path that no-ops). An agent needing a relationship to
  change today has no path, and building one is `SPEC-001`'s ADR, not a local fix.
- **NPC Cognition** (WE-8). The founder's ruling says the NPC plays according to the logged
  relationship *among other inputs* (`product.md`); the only mechanism found is the adjudication
  slice above. Whether cognition reads relationships by any other channel is unstated — `[INFER]`
  nothing was found (`git grep` relationship across `core/api` returns only the slice path).
