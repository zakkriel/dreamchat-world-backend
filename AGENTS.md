# DreamChat World Backend — AGENTS.md

> **Agent-agnostic entry point.** This file is the canonical instruction set for *any*
> coding agent working in this repo (open `AGENTS.md` convention). It depends on no
> specific agent's tooling or conventions. Tool-specific files (e.g. `CLAUDE.md`) are
> one-line pointers here; agent-specific tooling (user-level skills, etc.) is personal
> convenience, never a repo dependency (proposed register rule D-10).

**Mandate — before any work:** read `docs/00_strategy/06_rules_register.md` (**the law**)
and cite the rule IDs you rely on in your plans and PRs. No code, doc, or config change
ships without first checking it against the register.

DreamChat is a persistent AI RPG world platform. **The world is the product; this repo owns world truth.** It is built from a frozen, validated documentation contract — read it before writing code.

## Ground truth (read in this order)
1. `/docs/MASTER_INDEX.md` — map of all docs, statuses, decisions.
2. `/docs/00_strategy/06_rules_register.md` — **the law.** Every rule has an ID (B/C/D/E/F/GA + engine ADRs/invariants). Cite IDs in plans and PRs.
3. `/docs/30_architecture/canon_engine/` — **FROZEN build contract (v4.1).** Start at its `00_INDEX.md`. The Master DDL (doc 03) is the only core schema. Never propose changes to this set; if implementation reveals a genuine problem, the output is a *proposed new ADR superseding by number in doc 02* — never a code workaround.
4. `/docs/30_architecture/mvp_slice_and_bridge.md` — API contract + slice plan.
5. `/docs/30_architecture/implementation_playbook_superpowers.md` — the chunk ladder we are executing.

## Iron rules (non-negotiable; full versions in the register)
- **Canon events are immutable; nothing mutates canon directly.** LLMs and modules *propose*; the deterministic validation gate decides (ADR-001/009, D-1).
- **No mutable domain time.** In-world time is logical tick + display label, append-only; wall-clock `TIMESTAMPTZ` is operational telemetry only (B-5, ADR-030, `/docs/10_prds/compendium/00_time_and_mutability_rules.md`).
- **API responses never contain canon rows.** Everything crossing to the frontend is a perception-bound projection (B-1, I-3 — enforced at gate, context assembly, AND API boundary).
- **No relationship UI exists** (B-3). The internal relationship model stays internal.
- **Module mechanics never enter the Core** (D-2, GA-4). Module state = JSONB with mandatory `schema_version` + validation (D-4).
- **Invariants I-1…I-10 are the permanent regression suite** (engine doc 07). They run in CI; a red invariant blocks merge, always.

## Process
- We build in **chunks** (playbook §2). One chunk = one worktree = one plan = one PR. **Never start chunk N+1 while chunk N's gate is red.** Chunks marked 🪜 also require an honest answer to their Validation Ladder product question (§0.5) — green CI + product "no" = not done.
- Workflow per chunk: targeted brainstorm (open edges only) → write-plan from the spec → execute with TDD (failing test first, no exceptions) → gate check.
- Scope control: PRD non-goals + the register ARE the out-of-scope list. "Wouldn't it be nice to…" already has an answer — look it up.
- Empirical findings go to the tuning logs and, if they change a decision, to a register amendment or new ADR. Docs change because code taught us something — never to make code easier.

## Vocabulary
Use the Glossary (`/docs/00_strategy/05_glossary_ubiquitous_language.md`). `entity` is legal **only** as the engine supertype in this repo's internals; user-facing and PRD language is Actor / Location / Artifact / Carrying / Timeline. System terms must be genre-agnostic (GA-2).
