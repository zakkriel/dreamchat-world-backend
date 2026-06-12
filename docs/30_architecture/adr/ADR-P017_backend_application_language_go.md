# ADR-P017: Backend application language — Go

**Status:** Proposed
**Date:** 2026-06-11
**Series:** Platform (ADR-P###, per D-5) — does NOT touch frozen engine canon
**Owner of decision:** Chunk 3 (first backend application surface)

## Context

No backend application-language decision existed in the repo. The Chunk 1
design doc explicitly *deferred* it ("chosen later when Chunk 5 needs it").
Go appears in the repo only for the **external image platform**
(`docs/30_architecture/image_platform/`: "Go service", "Go API + workers"),
which is a separate pre-existing service, not the world backend. The
🟡-directional module/platform docs sketch a TypeScript-flavored backend
(`*.ts` folder trees), but those are explicitly suggestive-not-governing and
predate the engine freeze.

Chunk 3 introduces the first real backend application surface: a read-only,
perception-bound projection API serving the Actor-page payload. A language
must be chosen before that code is written.

Candidates weighed: **Go** vs **TypeScript** (Python rejected — this backend
is deterministic-core-with-LLM-at-the-edges per ADR-009, not a framework-driven
agent; the agent-framework ecosystem gravity does not apply). Performance does
not force the choice (LLM generation dominates wall-clock; both stream via SSE).
The decision rests on: stack consistency (the one existing backend service is
Go) and scale headroom (true multicore concurrency, steadier tail latency under
high concurrent session load) vs single-language-across-the-stack velocity (TS
matches the React frontend).

## Decision

The world backend application/transport tier is written in **Go**.

Rationale: (1) one backend language across services — the image platform is
already Go, so one founder maintains one backend language; (2) the deterministic
core / bounded-oracle architecture is a natural Go fit, and an official Anthropic
Go SDK exists for the Chunk-5 beat-loop LLM calls; (3) Go's small surface,
strict compiler, explicit error handling, and canonical formatting reduce
idiom-drift and silent-error-swallowing across a long AI-implemented,
TDD-gated build — aligning with the silent-workaround ban.

## Binding constraint (load-bearing)

**Perception filtering and the epistemic wall (B-1, I-3) live in SQL functions,
pgTAP-tested. The Go application layer calls them and never reimplements the
filter.** "Hidden truth is absent from the payload" is enforced at the database
boundary — which is exactly what the Chunk-3 DevTools gate verifies. This keeps
the language choice (Fork 1) from contaminating the perception-bound assembly
rule (Fork 3), and keeps the invariant guarantee language-agnostic.

## Consequences

- FE⇄BE types (Bridge §4, versioned by `schema_version`) are generated from a
  single schema source of truth (Go structs / OpenAPI / DB schema → TS), not
  shared directly. Cost: a codegen step + a two-language daily context tax for
  the solo founder. Accepted.
- A Python sidecar for evals/embeddings remains available later as a clean
  service boundary (ADR-018 keeps vector search retrieval-only and off the hot
  path); it is not part of this decision.
- No engine ADR, invariant (I-1…I-10), or the engine Master DDL is modified.
  D-1/I-7 unaffected: the gate/projection logic that protects canon stays in
  SQL/the maintainer, not in app code.

## Reversibility

The frozen design already isolates the LLM as an HTTP edge and runs extraction
async (D-8), so at scale the hot connection-gateway can be carved into its own
service later without rewriting the world engine. The choice is not a one-way
door.
