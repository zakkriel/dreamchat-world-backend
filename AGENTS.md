# DreamChat World Backend — AGENTS.md

> **Agent-agnostic entry point.** This file is the canonical instruction set for *any*
> coding agent working in this repo (open `AGENTS.md` convention). It depends on no
> specific agent's tooling or conventions. Tool-specific files (e.g. `CLAUDE.md`) are
> one-line pointers here; agent-specific tooling (user-level skills, etc.) is personal
> convenience, never a repo dependency (proposed register rule D-10).

> **Workspace harness:** `../AGENTS.md` + `../docs/00_workspace/` govern anything crossing a repo
> boundary. Read it before cross-repo work; this file governs everything inside this repo.
> Standalone clone (no parent workspace)? The harness lives at `github.com/zakkriel/dreamchat` —
> clone it and run its `bootstrap.sh`. This repo's own law and CI apply in full without it.

**FIRST MOVE, before you write a line — not a courtesy:**

```bash
../harness2/domain.sh core/api/<the file you are about to change>
../harness2/domain.sh --ask "<your question, in your own words>"
```

It prints the owning domain package (or, as its documented fallback for paths with no written
package yet, delegates to `../harness/brief.sh`'s area view), the decisions that govern that path,
what has already gone wrong there, and
the closed questions it touches. Every other gate here fires at PR time, which is too late: by then the
wrong shape is written. `../docs/00_workspace/closed-questions.md` is the same index in file form — if
your question is answered there it is **CLOSED**, and disagreeing with it is a new ADR, not a local
exception.

**Mandate — before any work:** read `docs/law/06_rules_register.md` (**the law**)
and cite the rule IDs you rely on in your plans and PRs. No code, doc, or config change
ships without first checking it against the register.

DreamChat is a persistent AI RPG world platform. **The world is the product; this repo owns world truth.** It is built from a frozen, validated documentation contract — read it before writing code.

## STOP — the four things that are not optional

Read these before your first edit. They are short, and every one of them exists because the same
mistake was made more than once.

1. **`/docs/maps/system_map.md`** — what exists, which repo owns it, and the seams you must
   not duplicate. If you are about to write something that already has an owner there, stop.
2. **`/docs/law/adr/`** — the decisions. Read `ADR-P020`…`ADR-P024` first: they cover
   migrations, art, prompts, art styles and seat config. Not knowing them has caused a production
   outage, a world shipped with no images, and a day of unnecessary work.
3. **The pre-flight below**, run before you open a PR.
4. **`/docs/law/06_rules_register.md`** — the law, as it always was.

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
- **The live frontend is `dream-weaver-visuals`** (port 5273). It vendors this repo's published
  schemas byte-identically; `../harness/check.sh contract-drift` is the only gate that sees both
  sides. **`dreamchat-frontend` was REMOVED** 2026-08-27 — superseded by `dream-weaver-visuals`
  and moved to quarantine pending deletion (`workspace:ADR-W006`, superseding `ADR-W003`'s
  archive-in-place); never read, cite, or restore from it, and port 5173 is retired with it.
  **`dc-fix/` was removed**
  2026-08-25: it was a stale detached worktree of this repo, 78 commits behind and 0 ahead. Need an
  isolated tree? `git worktree add` a fresh one from current `main` — a long-lived worktree is a second
  copy of the law, and the copy is the one that goes stale.
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

## Reference — the file you open tells you which docs it needs

**Open the file you are about to change and read its first lines.** A file bound by a decision says
so: `// Governed-by: ADR-P0xx — <one line on what it decides>`. Read that ADR before you edit. A file
with no such line is not governed by a specific decision; the always-list below still applies.

A test holds this together in both directions: an ADR that names a file as evidence and a file that
names the ADR must agree, and a `Governed-by` pointing at an ADR that does not exist fails the build
(`core/api/governance_test.go`).

The list below is what those pointers lead INTO. It is not a reading list you work through.

**Always, regardless of route:**
- `/docs/law/06_rules_register.md` — **the law.** Every rule has an ID. Cite the IDs you rely on.
- `/docs/maps/system_map.md` — what exists and who owns it.

**Routed — read the one that covers what you are touching:**
- `/docs/law/adr/` — the decisions. P020 migrations · P021 art · P022 prompts ·
  P023 art styles · P024 seat config.
- `/docs/law/02_world_state_adrs.md` — the engine ADRs, a **FROZEN build contract**.
  Required before touching the gate, the Master DDL, or any invariant. Never propose changes
  to this set; a genuine problem produces a proposed superseding ADR appended to that same doc,
  never a code workaround.
- `/docs/design/mvp_slice_and_bridge.md` — API contract + slice plan.
- `/docs/open-spec-items.md` — the deferred seams. Check before building something that was parked.

**Look up, do not read front-to-back:**
- `/docs/MASTER_INDEX.md` — the map of everything, with statuses.
- `/docs/runbooks/full-stack-from-zero.md` — exact commands to bring the stack up. Open it when you
  are bringing the stack up.

## Iron rules (non-negotiable; full versions in the register)
- **Canon events are immutable; nothing mutates canon directly.** LLMs and modules *propose*; the deterministic validation gate decides (ADR-001/009, D-1).
- **No mutable domain time.** In-world time is logical tick + display label, append-only; wall-clock `TIMESTAMPTZ` is operational telemetry only (B-5, ADR-030).
- **API responses never contain canon rows.** Everything crossing to the frontend is a perception-bound projection (B-1, I-3 — enforced at gate, context assembly, AND API boundary).
- **No relationship UI exists** (B-3). The internal relationship model stays internal.
- **Module mechanics never enter the Core** (D-2, GA-4). Module state = JSONB with mandatory `schema_version` + validation (D-4).
- **Invariants I-1…I-10 are the permanent regression suite** (named in `docs/law/06_rules_register.md` Part A). CI enforces a subset (I-1/I-2/I-7 plus guards — `invariants.yml` names them; failure-log #26 is why the claim is scoped); a red invariant blocks merge.

## Process
- **Backend-only repo.** Frontend code lives in **github.com/zakkriel/dream-weaver-visuals** (the live surface; `dreamchat-frontend` is its predecessor) and must **not** be built here — do not recreate a `frontend/` directory (D-7, D-10). The API contract `core/api/schema/actor_page.v2.schema.json` is the source of truth the frontend repo generates its types from; it stays here.
- We build in **chunks**. One chunk = one worktree = one plan = one PR. **Never start chunk N+1 while chunk N's gate is red.** Chunks marked 🪜 also require an honest answer to their Validation Ladder product question — green CI + product "no" = not done.
- Workflow per chunk: targeted brainstorm (open edges only) → write-plan from the spec → execute with TDD (failing test first, no exceptions) → gate check.
- Scope control: PRD non-goals + the register ARE the out-of-scope list. "Wouldn't it be nice to…" already has an answer — look it up.
- Empirical findings go to the tuning logs and, if they change a decision, to a register amendment or new ADR. Docs change because code taught us something — never to make code easier.

## Vocabulary
Use the Glossary (`/docs/law/05_glossary_ubiquitous_language.md`). `entity` is legal **only** as the engine supertype in this repo's internals; user-facing and PRD language is Actor / Location / Artifact / Carrying / Timeline. System terms must be genre-agnostic (GA-2).
