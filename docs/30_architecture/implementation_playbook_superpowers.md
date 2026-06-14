# DreamChat Implementation Playbook — Superpowers Edition

**Status:** v1 — 2026-06-10 | Companion to `mvp_slice_and_bridge.md`
**Premise:** the doc set is the build contract. Superpowers' brainstorm phase is ~80% already done — your specs are the brainstorm outputs. The loop per chunk is therefore: *short targeted brainstorm (edges only) → write-plan from the spec → execute-plan with TDD → gate check → next chunk.* Nothing advances past a red gate.

---

## 0. One-time setup (before any chunk)

1. **Repos.** `dreamchat-world-backend` and `dreamchat-frontend` (image platform already exists). Commit `dreamchat-docs/` into the backend repo as `/docs` (D-6: Git is truth for 10/20/30). Monorepo is also fine early — the boundary that matters is service, not repo.
2. **CLAUDE.md** (both repos) — short, pointing at truth, stating the iron rules:
   - "Read `/docs/MASTER_INDEX.md` first. `/docs/00_strategy/06_rules_register.md` is law. The engine set (`/docs/30_architecture/canon_engine/`) is FROZEN — never propose changes to it; new decisions require a new ADR superseding by number."
   - "FE never receives canon rows. No mutable domain time. No relationship UI. Modules propose, never write."
   - "Every chunk has a gate (playbook §2). Do not start chunk N+1 with chunk N's gate red."
3. **Custom skill: `dreamchat-rules`.** Superpowers supports project skills — create one SKILL.md that condenses the Rules Register (B/C/D/GA IDs + one-liners) and triggers on any feature work. This puts the law in context on every task without re-pasting docs.
4. **Invariant harness = the test spine.** Engine doc 07 defines I-1…I-10 with pass/fail SQL; doc 13 §pass/fail is executable. Wire these into CI on day one — they are the permanent regression suite every chunk runs against. The TDD iron law (test first) starts here: the invariants are failing tests *by definition* until the spine exists.
5. **Worktree discipline.** One chunk = one branch/worktree = one plan doc = one PR. Superpowers' worktree isolation maps 1:1 onto the chunk ladder.
6. **Sanctioned toolchain.** One sanctioned environment per chunk family: the pinned Docker image (local == CI == deployed family). A `make doctor` preflight verifies the runtime and fails with install instructions. Fallbacks, where unavoidable, are documented as non-sanctioned emergency exceptions whose results are never gate-authoritative.

## 0.5 The Validation Ladder (principle)

> **Every 2–3 chunks must answer a product question, not only an architecture question.**
> A chunk that carries a product question is not done when CI is green — it is done when the question has an honest answer. **A "no" stops the ladder**: you fix the product (or amend the bet via the register/ADR process) before climbing on. This is what makes the ladder an MVP validation plan, not just an execution plan.

| After chunk | Product question | What a "no" looks like (falsification) |
|---|---|---|
| **1** | **Can world state replay deterministically?** ✅ Answered YES 2026-06-11 (operator-verified: identical projection data across fresh DBs; §7 checks all true; replay true incl. negative control; gate tag chunk-1-0A-gate) | Replays diverge, or "determinism" only holds by excluding things that matter. If the spine can't replay, nothing above it is trustworthy — stop. |
| **3** | **Can the user inspect a world and trust it?** | You open Mara's page and catch yourself checking the DB to see if it's right; the payload leaks the secret; sourcing reads as decoration instead of evidence. |
| **5** | **Can the user play without transcript-dependence?** | Delete the transcript and the world gets dumber — the narrator loses facts, an NPC forgets, continuity quietly came from chat history instead of canon. This is the product promise's first live test. |
| **8** | **Can the user fix the world without breaking immersion?** | Correcting feels like filing a ticket: approval UI appears, flow interrupts, or the fix doesn't visibly hold in subsequent play. C-11 exists precisely so the answer is yes. |
| **10** | **Does the world feel alive?** | You jump three in-world weeks, return, and the backstage update reads as generated filler rather than *earned consequence*; decay language feels like a disclaimer instead of a world remembering imperfectly. The soak test passes but the feeling doesn't — that's still a no. |
| **12** | **Can the world carry gameplay without losing itself?** *(added to complete the rhythm — confirm)* | The battle feels bolted on: its consequences don't surface in the Compendium, or winning/losing leaves no mark the world remembers. |
| **14** | **Can a user create their own world and believe in it?** *(added — confirm)* | The magic only works in the hand-seeded Mara world; a fresh world feels generic, faceless, or forgetful — meaning the product is a demo, not a platform. |

Questions at 1/3/5/8/10 are the user's own framing (2026-06-10); 12/14 added to keep the 2–3 chunk rhythm through the end.

## 1. The loop per chunk

```
/superpowers:brainstorm   →  ONLY for the chunk's open edges (each chunk lists them).
                             Feed it the relevant spec docs; out-of-scope = PRD non-goals + register.
/superpowers:write-plan   →  Plan from the spec. 2–5 min tasks, exact paths, complete failing
                             tests first. The chunk's gate is the plan's final task.
/superpowers:execute-plan →  Subagent execution with review checkpoints.
Gate check                →  Run the gate (SQL / scripted driver / your own hands in the UI).
                             If the chunk carries a Validation Ladder question (§0.5), answer it
                             honestly — CI green + product "no" = chunk NOT done.
                             Green → merge → next chunk. Red → debug skill, never skip.
                             Steps marked operator-run (every chunk's final gate task) are
                             executed by the human, never by an executing agent — including the
                             gate tag. An agent that completes a plan may prepare the gate; it
                             does not run it or tag it.
YOU test                  →  Every chunk ends with something you personally poke at.
```

> **Note (review feedback applied outside the planning session):** verify after each revision that the count of changes matches the count of asks. A revision only contains what was actually pasted into its prompt.

## 2. The chunk ladder

Each chunk: **Build → You test → Gate**. Chunks 1–2 are the engine's own mandated start (doc 13: "build this first — ends the doc phase").

| # | Chunk | Build | **You personally test** | Gate (CI/scripted) |
|---|---|---|---|---|
| **1** | **Deterministic spine (engine 0A)** 🪜Q1 | doc 13 §2 schema subset + bundle/provenance tables (ADR-008 carve-out), append-only + DELETE guards on all canon tables, shared apply_mutation() write path, Mara seed, replay harness. *No LLM anywhere.* | Run the doc 13 pass/fail SQL yourself; drop projections, replay, watch identical domain state rebuild | **Green I-1** (replay invariance) + I-2 on Mara. The engine's own Phase 0A exit |
| **2** | **Bundle regression (0B)** | Manual Seren inserts exercising bundle tables | Insert a cyclic bundle by hand → watch it get rejected | I-4 acyclicity; manual bundle regression |
| **3** | **Projection API + first page** 🪜Q2 | Read-side endpoints (Bridge §4.1) over seeded data; FE shell + **Actor page** rendering Mara seed. Brainstorm must resolve: (a) the backend language decision — deferred from Chunk 1's pure-SQL posture; either a language arrives here, or projections ship as Postgres JSON functions behind a thin reader (PostgREST-style) and the language decision moves to Chunk 5 — an explicit fork, not a drift; (b) SPEC-ledger check. | Open Mara's page in a browser. Check: no relationship field, sourced knowledge, tick labels, **the planted secret is absent from the network payload** (DevTools, not UI) | I-3 audit on page payloads; Actors PRD ACs 1–2, 6–10 against seed data |
(Seren does not exist in 0A/0B — ADR-029; she is the Phase-4 golden. Chunk-3 gate runs on Mara:
viewer=Player sees a coherent Mara page; viewer=Jonas's payload omits Mara's private belief.
"Seren's page" is satisfied later at S4.)
| **4** | **Full read-only Compendium** | Locations, Artifacts, Timeline, Carrying overlay pages + Graph Inspector (debug) | Browse all of Mara's world; click a timeline record → verify it shows a *perception version*; open the Graph Inspector on an event | All four PRDs' read-side ACs on seed; timeline-never-links-canon check |
| **5** | **Fast-path play loop (engine Phase 1)** 🪜Q3 | Deterministic action handlers, perception fan-out, entity registry, context assembler v1, narrator (read-only LLM), minimal beat input + streaming | **Play the Mara slice yourself, end-to-end, in the UI.** Then delete the transcript and verify the world lost nothing | Engine Phase 1 gate (Mara slice, scripted driver) + Bridge S1 gate (human driver) |
| **6** | **Play surface polish** | Scene canvas, participants strip, Aux Current+Known lenses, return-to-world flow | Leave mid-scene, come back tomorrow: do you land oriented? Does Aux answer "what matters now"? | C-1/C-3/C-6/C-10 review checklist; no new invariants |
| **7** | **Canonization pipeline (engine Phase 2)** | Window state machine, template extraction, entity resolution, validation gate + repair, claim ledger | Type free-form prose; watch proposals form and commit. Try to confuse entity resolution ("the guard" when two guards exist) | **Free-form Mara slice**; I-10 sweep clean; resolution gates (doc 05 §8); I-8 |
| **8** | **Corrections + Intent/Inspect (S2)** 🪜Q4 | Invisible correction window UX, explicit lock, Report issue everywhere, Intent + Inspect lenses | Make the narrator say something wrong, correct it, verify present-forward fix; check Continue implicitly accepts (no approval UI ever appears) | Correction round-trip test; C-11; ADR-016 behavior |
| **9** | **Epistemic depth (engine Phase 3)** | Visibility scopes at fidelity, communication events, rumor chains with distortion lineage | Tell NPC A a secret; watch it reach NPC C as a *distorted rumor* with lineage in the Graph Inspector; confirm B-7 (C knows "was told", not "saw") | Live rumor chain; planted-secret-at-scale (I-3); mixed-scene leak test O-1 (early per v4.1) |
| **10** | **Living world (engine Phase 4)** 🪜Q5 | Bundles via templates, deterministic thresholds, backstage review queue + worker, dirty ladder, decay language live | Jump in-world time three weeks; return to an NPC and check the backstage update feels earned; verify stale knowledge says "last known…" instead of vanishing | **Seren golden**; bounded invalidation exact; soak test (1,000 actions) |
| **11** | **Module runtime + Stats** | Runtime skeleton, manifests, capability contracts, Stats module (JSONB + schema_version per D-4), `aux.panel` slot | Enable Stats for the world; check stats render via the module slot and that disabling the module breaks nothing | Module writes exist *only* as gated proposals (D-1 audit); core schema untouched by module concepts (D-2) |
| **12** | **Battle demo** 🪜Q6 | JRPG Battle module: encounter lifecycle, turn resolution, scene overlay + action bar slots, consequences as proposals | **Fight a battle.** Lose it. Verify injuries/relationship consequences became canon through the gate and show up in the Compendium afterwards | Full battle writes canon only via proposals; timeline records of the battle point to perception versions |
| **13** | **Images Stage A** | Asset-pack job at world creation (portraits, location art, typed placeholders); governance classification before dispatch (E-1); manual regenerate. **Assumption: the image platform exists as an external service — this chunk never rebuilds it.** If the V2 addendum (asset packs/sprite sheets) is not yet implemented there, it becomes a sub-chunk executed in the image platform's own repo (spec: `image_platform/implementation_prompt_v2.md` + schema extension SQL); chunks 1–12 have zero image dependency, so this parallelizes freely | Create a fresh world; watch portraits/scenes arrive; regenerate one | No generation request bypasses classification; narration latency unaffected (D-8) |
| **14** | **Images Stage B + world creation flow** 🪜Q7 | Async live triggers (importance threshold, first visit, first inspect) + `image.ready` swap-in; full world creation/entry UX | Create a brand-new world from scratch and play it — a *new* NPC who becomes important gets a face without you asking | E2E: new world → play → images arrive async → MVP definition (Bridge §2) fully met |

**Chunks 1–4 need no LLM at all.** That's deliberate (engine ADR-029: "Phase 0 is protected from cleverness") — you validate the entire data model and every Compendium page against seeded data before a single token is generated. If something is wrong in the model, you find it at chunk 3 for the price of a SQL fix, not at chunk 9.

## 3. Anti-drift rules during execution

- **The engine set is read-only input to plans.** If execution discovers a real engine problem, the output is a *proposed ADR-034*, not a code workaround. (Doc 11 — open concerns — is the honest list of where this is most likely.)
- **PRD non-goals are the out-of-scope list** fed to every brainstorm/plan. "Wouldn't it be nice to add a relationship meter" already has its answer (B-3/B-4).
- **Empirical findings flow back** to the three tuning logs (extraction, threshold, assembly audit — engine governance) and, where they change decisions, to register amendments. Docs change because code taught you something — never the other way around now (D-9).
- **One chunk at a time.** Parallel worktrees are fine *within* a chunk (Superpowers' parallel subagents); chunks themselves are sequential because each gate is the next chunk's foundation.
- **Open-Spec Ledger (`docs/open-spec-items.md`):** when a chunk discovers the frozen contract is silent or inconsistent on a mechanism, the gap gets a SPEC-### entry (description, owner chunk, expected ADR outcome) and the code ships an explicit, documented stub — never an invented mechanism. Every chunk brainstorm opens by checking the ledger for items it owns.

## 4. First session, concretely

```
1. Set up repos + CLAUDE.md + /docs + dreamchat-rules skill (this doc, §0).
2. /superpowers:brainstorm  "Chunk 1: deploy the Phase 0A engine contract
   (docs/30_architecture/canon_engine/13_phase_0A_engine_contract.md).
   Open edges only: Postgres hosting, migration tool, test runner choice."
3. /superpowers:write-plan  — doc 13 already specifies tables, Mara seed,
   expected rows, replay rule, pass/fail SQL. The plan is mostly transcription.
4. /superpowers:execute-plan
5. Run the pass/fail SQL with your own hands. Green I-1 = the doc phase is
   officially over, exactly as the engine index demanded.
```
