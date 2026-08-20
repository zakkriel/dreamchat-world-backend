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

## STOP — the four things that are not optional

Read these before your first edit. They are short, and every one of them exists because the same
mistake was made more than once.

1. **`/docs/30_architecture/system_map.md`** — what exists, which repo owns it, and the seams you must
   not duplicate. If you are about to write something that already has an owner there, stop.
2. **`/docs/30_architecture/adr/`** — the decisions. `ADR-P020`…`ADR-P023` are the operational ones
   this repo bleeds from when they are not known.
3. **The pre-flight below**, run before you open a PR.
4. **`/docs/00_strategy/06_rules_register.md`** — the law, as it always was.

### Pre-flight (run it; do not assume)

| Check | Why it is here |
|---|---|
| Is my branch cut from **current `origin/main`**? | An agent branched off a months-old commit and silently reverted several merged features. `git merge-base --is-ancestor origin/main HEAD` must pass. |
| Did I add a migration? Then `make migrate` and commit **BOTH** `core/db/schema.sql` and `core/api/migrations.txt`. | `make schema-check` fails otherwise. And see ADR-P020: merging is not applying. |
| Did I publish a schema under `core/api/schema/`? Then a real payload must be captured in `ci/gen_payloads.sh`. | SPEC-011 fails the build on any published schema with no payload behind it. |
| Do my tests fail if I revert the fix? | A fake must be at least as strict as the service it stands in for. Bugs have shipped green because the stand-in accepted what production rejects. Revert your fix, watch the test fail, restore it. |
| Changing the latitude block? Then EVERY `core/api/prompts/*.txt` and the image latitude in `artstyle.go`, in one PR. | Editing two files and stopping is the exact miss that needed a founder to catch it. `go test -run Latitude` before you open the PR. |
| Am I restating something a module already owns? | A style's look, a seat's latitude, an entity's art status. See the seam table in the system map. |
| Did I change the shape the system map describes? | Then amend it in the SAME PR. |

### Standing answers — do not ask, do not re-derive

- **Art is automatic** (ADR-P021). Genesis kicks a reconciler and a ticker sweeps. Never add a
  commissioning call to a new creation path, and never tell a user to trigger one.
- **The deploy does not run migrations** (ADR-P020). The service refuses to boot on drift, on purpose.
  Releasing is: **apply config → apply migrations → merge → watch it boot → exercise the path.**
- **Seat config is part of the release** (ADR-P024). A seat needing a token ceiling or `json_object`
  must have it set in the environment BEFORE the merge that needs it. Nothing checks this yet.
- **Every seat prompt carries the same byte-identical latitude block** (ADR-P022). Adding a seat means
  adding the block; there is no prompt too small.
- **A style's look lives in `artstyle.go` and nowhere else** (ADR-P023). Clients pick by key.
- **The live frontend is `dream-weaver-visuals`.** `dreamchat-frontend` is the older repo.
  `dc-fix/` is a stale git WORKTREE of this repo, not a separate project, and is not deployed.
- **Canon is written through `apply_event` / `apply_ruled_event`, never directly** (D-1, ADR-009).
  Genesis' `origin='fast_path'` is the one documented exception and it exists because the actors an
  event would reference do not exist yet. Bypassing the gate corrupts replay (I-1) and provenance (I-2).
- **A user-facing read goes through the perception functions** (B-1, I-3) — `fn_visible_perceptions`,
  `fn_display_name`. Querying `*_state` for a payload leaks canon and is how a page learns to lie.
- **In-world time is a logical tick + display label** (B-5, ADR-030). Wall-clock `TIMESTAMPTZ` is
  operational telemetry, never domain time.
- **Every JSONB payload carries `schema_version` and is validated** (D-4).
- **Do not invent constraints.** If you cannot cite a rule ID, an ADR, or a line of code, you do not
  have a constraint — you have a preference, and it does not belong in a plan, a PR, or a refusal.
  The private/public test is ACCESS, not payment (ADR-P016): charging a user for their own world
  keeps it private. Absence of a rule is not a prohibition, and a caution the register did not ask
  for is noise that costs the reader time.
- **A listed gap is ordered work, not a question.** Anything in `system_map.md` §7 under "not
  enforced" may be built without asking. Same for a test that locks a Standing answer down.
- **Clean cutover is the default.** Migrate every caller and delete the old path. Where a
  compatibility field genuinely survives it is documented AT the code and is not precedent — e.g. the
  legacy joined `narration` string in `beathandler.go` is kept deliberately for older clients. Adding
  a new shim needs a reason written next to it; "for safety" is not one.

## Ground truth (read in this order)
1. `/docs/MASTER_INDEX.md` — map of all docs, statuses, decisions.
2. `/docs/00_strategy/06_rules_register.md` — **the law.** Every rule has an ID (B/C/D/E/F/GA + engine ADRs/invariants). Cite IDs in plans and PRs.
3. `/docs/30_architecture/canon_engine/` — **FROZEN build contract (v4.1).** Start at its `00_INDEX.md`. The Master DDL (doc 03) is the only core schema. Never propose changes to this set; if implementation reveals a genuine problem, the output is a *proposed new ADR superseding by number in doc 02* — never a code workaround.
4. `/docs/30_architecture/mvp_slice_and_bridge.md` — API contract + slice plan.
5. `/docs/30_architecture/implementation_playbook_superpowers.md` — the chunk ladder we are executing.
6. `/docs/superpowers/handovers/2026-08-08-three-repo-integration-handover.md` — **history, not current state.** Useful for the cross-repo contracts and the mistakes that cost time. **Superseded on world creation and images** by `system_map.md` + ADR-P021/P023: it still describes explicit image triggers and a `POST /worlds` that authors no entities. Where they disagree, the system map wins.
7. `/docs/runbooks/full-stack-from-zero.md` — bringing up all four processes (DB, backend, image platform, frontend) with exact verified commands, plus the battery order that actually works and why.

## Iron rules (non-negotiable; full versions in the register)
- **Canon events are immutable; nothing mutates canon directly.** LLMs and modules *propose*; the deterministic validation gate decides (ADR-001/009, D-1).
- **No mutable domain time.** In-world time is logical tick + display label, append-only; wall-clock `TIMESTAMPTZ` is operational telemetry only (B-5, ADR-030, `/docs/10_prds/compendium/00_time_and_mutability_rules.md`).
- **API responses never contain canon rows.** Everything crossing to the frontend is a perception-bound projection (B-1, I-3 — enforced at gate, context assembly, AND API boundary).
- **No relationship UI exists** (B-3). The internal relationship model stays internal.
- **Module mechanics never enter the Core** (D-2, GA-4). Module state = JSONB with mandatory `schema_version` + validation (D-4).
- **Invariants I-1…I-10 are the permanent regression suite** (engine doc 07). They run in CI; a red invariant blocks merge, always.

## Process
- **Backend-only repo.** Frontend code lives in **github.com/zakkriel/dream-weaver-visuals** (the live surface; `dreamchat-frontend` is its predecessor) and must **not** be built here — do not recreate a `frontend/` directory (D-7, D-10). The API contract `core/api/schema/actor_page.v2.schema.json` is the source of truth the frontend repo generates its types from; it stays here.
- We build in **chunks** (playbook §2). One chunk = one worktree = one plan = one PR. **Never start chunk N+1 while chunk N's gate is red.** Chunks marked 🪜 also require an honest answer to their Validation Ladder product question (§0.5) — green CI + product "no" = not done.
- Workflow per chunk: targeted brainstorm (open edges only) → write-plan from the spec → execute with TDD (failing test first, no exceptions) → gate check.
- Scope control: PRD non-goals + the register ARE the out-of-scope list. "Wouldn't it be nice to…" already has an answer — look it up.
- Empirical findings go to the tuning logs and, if they change a decision, to a register amendment or new ADR. Docs change because code taught us something — never to make code easier.

## Vocabulary
Use the Glossary (`/docs/00_strategy/05_glossary_ubiquitous_language.md`). `entity` is legal **only** as the engine supertype in this repo's internals; user-facing and PRD language is Actor / Location / Artifact / Carrying / Timeline. System terms must be genre-agnostic (GA-2).
